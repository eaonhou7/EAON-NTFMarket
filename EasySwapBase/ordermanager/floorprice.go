package ordermanager

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"go.uber.org/zap"

	"github.com/ProjectsTask/EasySwapBase/stores/gdb"

	"github.com/ProjectsTask/EasySwapBase/logger/xzap"
	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/multi"
	"github.com/ProjectsTask/EasySwapBase/stores/xkv"
)

type EventType int

const (
	Buy EventType = iota + 1
	Mint
	Listing
	Cancel
	Transfer         EventType = 8
	Expired          EventType = 9
	ImportCollection EventType = 10
	UpdateCollection EventType = 11
)

const (
	MaxBatchReqNum = 100
	maxQueueLength = 100
)

const CacheTradeEventsQueuePre = "cache:es:trade:events:%s"

type collectionTradeInfo struct {
	floorPrice decimal.Decimal
	orders     *PriorityQueueMap // 优先级队列
}

type TradeEvent struct {
	EventType      EventType       `json:"event_type"`
	CollectionAddr string          `json:"collection_addr"`
	TokenID        string          `json:"token_id"`
	OrderId        string          `json:"order_id"`
	OrderHash      string          `json:"order_hash"`
	Price          decimal.Decimal `json:"price"`
	From           string          `json:"from"`
	To             string          `json:"to"`
	TxHash         string          `json:"txHash"`
}

func (om *OrderManager) floorPriceProcess() {
	// 清空缓存中的剩余事件
	key := genTradeEventsCacheKey(om.chain)
	if err := om.Xkv.Redis.Ltrim(key, 1, 0); err != nil { // clear all value
		xzap.WithContext(om.Ctx).Error("failed on flush remaining trade events", zap.Error(err))
	}

	// 从数据库加载订单并更新地板价
	if err := om.loadCollectionTradeInfo(); err != nil {
		xzap.WithContext(om.Ctx).Error("[Order Manage] load orders to queue", zap.Error(err))
		return
	}

	// 持续监听并处理交易事件
	for {
		// 从缓存中获取交易事件
		result, err := om.Xkv.Lpop(key)
		if err != nil || result == "" {
			if err != nil && err != redis.Nil {
				xzap.WithContext(om.Ctx).Warn("failed on get trade events from cache", zap.Error(err), zap.String("result", result))
			}
			time.Sleep(1 * time.Second)
			continue
		}

		xzap.WithContext(om.Ctx).Info("get trade events from cache", zap.String("event content", result))
		// 解析事件内容
		var event TradeEvent
		if err := json.Unmarshal([]byte(result), &event); err != nil {
			xzap.WithContext(om.Ctx).Warn("failed on unmarshal trade event info", zap.Error(err))
			continue
		}

		// 检查collection是否被跟踪
		tradeInfo, ok := om.collectionOrders[strings.ToLower(event.CollectionAddr)]
		if !ok && event.EventType != ImportCollection { //
			xzap.WithContext(om.Ctx).Warn("untracked collection", zap.String("collection_addr", event.CollectionAddr))
			continue
		}

		// 通知collection状态更新
		if event.CollectionAddr != "" {
			om.collectionListedCh <- event.CollectionAddr
		}

		// 根据不同事件类型处理
		switch event.EventType {
		case Listing: // 上架事件
			// 只有当价格低于队列最高价或队列为空时才添加订单
			_, price := tradeInfo.orders.GetMax()
			if price.GreaterThan(event.Price) || tradeInfo.orders.Len() == 0 {
				tradeInfo.orders.Add(event.OrderId, event.Price, event.From, event.TokenID)
			}
			// 更新地板价
			if err := om.checkAndUpdateFloorPrice(event.CollectionAddr); err != nil {
				xzap.WithContext(om.Ctx).Error("failed on update collection floor price",
					zap.Int("event_type", int(event.EventType)), zap.String("order_id", event.OrderId),
					zap.String("collection_addr", event.CollectionAddr), zap.String("floor_price", event.Price.String()),
					zap.Error(err))
			}

		case Cancel, Expired: // 取消或过期事件
			// 从队列中删除订单
			tradeInfo.orders.Remove(event.OrderId)
			if tradeInfo.orders.Len() == 0 {
				// 队列为空时重新加载订单
				if err := om.reloadCollectionOrders(event.CollectionAddr); err != nil {
					xzap.WithContext(om.Ctx).Error("failed on reload orders at the lowest price",
						zap.Int("event_type", int(event.EventType)), zap.String("order_id", event.OrderId),
						zap.String("collection_addr", event.CollectionAddr), zap.Error(err))
					continue
				}
			}
			// 更新地板价
			if err := om.checkAndUpdateFloorPrice(event.CollectionAddr); err != nil {
				xzap.WithContext(om.Ctx).Error("failed on update collection floor price",
					zap.Int("event_type", int(event.EventType)), zap.String("order_id", event.OrderId),
					zap.String("collection_addr", event.CollectionAddr),
					zap.Error(err))
			}

		case Buy, Transfer: // 购买或转移事件
			// 如果是购买事件,从队列中删除订单
			if event.EventType == Buy {
				tradeInfo.orders.Remove(event.OrderId)
			}

			// 检查队列是否为空,为空则重新加载订单
			if tradeInfo.orders.Len() == 0 {
				if err := om.reloadCollectionOrders(event.CollectionAddr); err != nil {
					xzap.WithContext(om.Ctx).Error("failed on reload orders at the lowest price",
						zap.Int("event_type", int(event.EventType)), zap.String("order_id", event.OrderId),
						zap.String("collection_addr", event.CollectionAddr), zap.Error(err))
					continue
				}
			} else {
				// 移除卖家的所有订单
				tradeInfo.orders.RemoveMakerOrders(event.From, event.TokenID)
				// 获取买家的有效订单
				orders, err := om.getUserValidOrders(event.CollectionAddr, event.TokenID, event.To)
				if err != nil {
					xzap.WithContext(om.Ctx).Error("failed on get users valid orders",
						zap.Int("event_type", int(event.EventType)), zap.String("order_id", event.OrderId),
						zap.String("collection_addr", event.CollectionAddr),
						zap.String("from", event.From), zap.String("to", event.To),
						zap.Error(err))
					continue
				}

				// 添加买家的有效订单到队列
				for _, order := range orders {
					if !order.IsOpenseaBanned {
						_, maxPrice := tradeInfo.orders.GetMax()
						if maxPrice.GreaterThan(order.Price) {
							tradeInfo.orders.Add(order.OrderID, order.Price, order.Maker, order.TokenId)
						}
					}
				}

				// 如果队列为空,重新加载订单
				if tradeInfo.orders.Len() == 0 {
					if err := om.reloadCollectionOrders(event.CollectionAddr); err != nil {
						xzap.WithContext(om.Ctx).Error("failed on reload orders at the lowest price",
							zap.Int("event_type", int(event.EventType)), zap.String("order_id", event.OrderId),
							zap.String("collection_addr", event.CollectionAddr), zap.Error(err))
						continue
					}
				}
			}

			// 更新地板价
			if err := om.checkAndUpdateFloorPrice(event.CollectionAddr); err != nil {
				xzap.WithContext(om.Ctx).Error("failed on update collection floor price",
					zap.Int("event_type", int(event.EventType)), zap.String("order_id", event.OrderId),
					zap.String("collection_addr", event.CollectionAddr), zap.String("floor_price", event.Price.String()),
					zap.Error(err))
			}

		case ImportCollection: // 导入新的Collection事件
			// 检查Collection是否已存在
			if _, ok := om.collectionOrders[strings.ToLower(event.CollectionAddr)]; ok {
				xzap.WithContext(om.Ctx).Warn("import collection repeated",
					zap.String("collection_addr", event.CollectionAddr))
				continue
			}

			// 初始化新的Collection信息
			om.collectionOrders[strings.ToLower(event.CollectionAddr)] = &collectionTradeInfo{
				floorPrice: decimal.Zero,
				orders:     NewPriorityQueueMap(maxQueueLength),
			}

		case UpdateCollection: // 更新Collection事件
			// 检查地板价是否变化
			_, floorPrice := tradeInfo.orders.GetMin()
			if floorPrice.Equal(event.Price) {
				continue
			}

			// 重新加载订单并更新地板价
			if err := om.reloadCollectionOrders(event.CollectionAddr); err != nil {
				xzap.WithContext(om.Ctx).Error("failed on reload orders at the lowest price",
					zap.Int("event_type", int(event.EventType)), zap.String("order_id", event.OrderId),
					zap.String("collection_addr", event.CollectionAddr), zap.Error(err))
				continue
			}
			if err := om.checkAndUpdateFloorPrice(event.CollectionAddr); err != nil {
				xzap.WithContext(om.Ctx).Error("failed on update collection floor price",
					zap.Int("event_type", int(event.EventType)), zap.String("order_id", event.OrderId),
					zap.String("collection_addr", event.CollectionAddr), zap.String("floor_price", event.Price.String()),
					zap.Error(err))
			}

		default:
			xzap.WithContext(om.Ctx).Error("unsupported event type", zap.String("event content", result))
		}
	}
}

// loadCollectionTradeInfo 函数主要负责初始化和加载集合(Collection)的交易信息,主要包含以下步骤:
func (om *OrderManager) loadCollectionTradeInfo() error {
	// 1. 从数据库加载所有集合信息
	var collections []*multi.Collection
	if err := om.DB.WithContext(om.Ctx).Table(gdb.GetMultiProjectCollectionTableName(om.project, om.chain)).
		Select("id, address, floor_price").Where("1=1").Scan(&collections).Error; err != nil {
		return errors.Wrap(err, "failed on get collection floor price")
	}
	var collectionAddrs []string
	for _, collection := range collections {
		collectionAddrs = append(collectionAddrs, collection.Address)
		// 为每个集合初始化交易信息,包含地板价和订单优先队列
		om.collectionOrders[strings.ToLower(collection.Address)] = &collectionTradeInfo{
			floorPrice: collection.FloorPrice,
			orders:     NewPriorityQueueMap(maxQueueLength), // 优先级队列,限制最大长度
		}
	}

	// 2. 分批加载每个集合的前100个有效订单
	var totalOrders []*multi.Order
	var id int64
	for {
		var orders []*multi.Order
		// 查询条件:
		// - 订单类型为Listing
		// - 订单状态为Active
		// - maker必须是token的owner
		// - 非OpenSea禁止的item
		if err := om.DB.WithContext(om.Ctx).Table(fmt.Sprintf("%s as co", gdb.GetMultiProjectOrderTableName(om.project, om.chain))).
			Select("co.id as id ,co.order_id as order_id, co.collection_address as collection_address, co.price as price,co.maker as maker,co.token_id as token_id").
			Joins(fmt.Sprintf("join %s ci on co.collection_address = ci.collection_address and co.token_id = ci.token_id", gdb.GetMultiProjectItemTableName(om.project, om.chain))).
			Where("co.order_type=? and co.order_status = ? and co.maker = ci.owner  and (ci.is_opensea_banned,co.marketplace_id)!=(true,1) and co.id > ?", multi.ListingType, multi.OrderStatusActive, id).
			Order("co.id asc").Limit(1000).
			Scan(&orders).Error; err != nil {
			return errors.Wrap(err, "failed on get collection orders")
		}
		totalOrders = append(totalOrders, orders...)
		if len(orders) < 1000 {
			break
		}

		id = orders[len(orders)-1].ID
	}

	// 3. 将订单添加到对应集合的优先队列中
	for _, order := range totalOrders {
		ordersQueue, ok := om.collectionOrders[strings.ToLower(order.CollectionAddress)]
		if !ok {
			xzap.WithContext(om.Ctx).Warn("untracked collection", zap.String("collection_addr", order.CollectionAddress))
			continue
		}
		ordersQueue.orders.Add(order.OrderID, order.Price, order.Maker, order.TokenId)
	}

	// 4. 检查并更新每个集合的地板价
	for addr, tradeInfo := range om.collectionOrders {
		_, floorPrice := tradeInfo.orders.GetMin()
		if !floorPrice.Equal(tradeInfo.floorPrice) {
			om.collectionOrders[strings.ToLower(addr)].floorPrice = floorPrice
			if err := om.updateFloorPrice(addr, floorPrice); err != nil {
				xzap.WithContext(om.Ctx).Warn("failed on update collection floor price",
					zap.String("collection_addr", addr), zap.Error(err))
			}
		}
	}

	return nil
}

func (om *OrderManager) updateFloorPrice(collectionAddr string, price decimal.Decimal) error {
	if err := om.DB.WithContext(om.Ctx).Table(gdb.GetMultiProjectCollectionTableName(om.project, om.chain)).
		Where("address=?", collectionAddr).Update("floor_price", price).Error; err != nil {
		return errors.Wrap(err, "failed on update collection floor price")
	}
	return nil
}

// checkAndUpdateFloorPrice 函数用于检查和更新NFT集合的地板价
// 主要功能包括:
// 1. 检查集合是否被跟踪
// 2. 获取集合当前最低价格
// 3. 如果最低价格发生变化,则更新缓存和数据库中的地板价
// 参数说明:
// - address: NFT集合地址
func (om *OrderManager) checkAndUpdateFloorPrice(address string) error {
	// 1. 检查集合是否被跟踪
	tradeInfo, ok := om.collectionOrders[strings.ToLower(address)]
	if !ok {
		xzap.WithContext(om.Ctx).Warn("untracked collection", zap.String("collection_addr", address))
		return errors.New("untracked collection")
	}

	// 2. 获取集合当前最低价格
	_, newFloorPrice := tradeInfo.orders.GetMin()

	// 3. 如果最低价格发生变化,则更新地板价
	if !newFloorPrice.Equal(tradeInfo.floorPrice) {
		// 更新内存缓存中的地板价
		tradeInfo.floorPrice = newFloorPrice
		om.collectionOrders[strings.ToLower(address)] = tradeInfo

		// 更新数据库中的地板价
		if err := om.updateFloorPrice(address, newFloorPrice); err != nil {
			return errors.Wrap(err, "failed on update collection floor price")
		}

		// 记录地板价更新日志
		xzap.WithContext(om.Ctx).Info("update collection floor price",
			zap.String("collection_addr", address), zap.String("floor_price", newFloorPrice.String()))
	}
	return nil
}

// getLowestPrice100Orders 函数用于获取指定NFT集合中价格最低的100个订单
// 主要功能包括:
// 1. 从数据库中查询指定集合的订单信息
// 2. 过滤条件:
//   - 订单类型为listing
//   - 订单状态为active
//   - maker必须是token的owner
//   - 非OpenSea禁止的item
//
// 3. 按价格升序排序并限制返回100条记录
// 参数说明:
// - address: NFT集合地址
// 返回值:
// - []*multi.Order: 订单列表
// - error: 错误信息
func (om *OrderManager) getLowestPrice100Orders(address string) ([]*multi.Order, error) {
	var orders []*multi.Order
	if err := om.DB.WithContext(om.Ctx).Table(fmt.Sprintf("%s as co", gdb.GetMultiProjectOrderTableName(om.project, om.chain))).
		Select("co.id,co.order_id as order_id, co.collection_address, co.price, co.maker,co.token_id").
		Joins(fmt.Sprintf("join %s ci on co.collection_address = ci.collection_address and co.token_id = ci.token_id", gdb.GetMultiProjectItemTableName(om.project, om.chain))).
		Where("co.order_type=? and co.order_status = ? and co.maker = ci.owner  and (ci.is_opensea_banned,co.marketplace_id)!=(true,1)", multi.ListingType, multi.OrderStatusActive).
		Where("co.collection_address = ?", address).Order("co.price asc").Limit(100).
		Scan(&orders).Error; err != nil {
		return nil, errors.Wrap(err, "failed on get collection lowest price orders")
	}
	return orders, nil
}

// reloadCollectionOrders 函数用于重新加载指定NFT集合的订单信息
// 主要功能包括:
// 1. 检查集合是否被跟踪
// 2. 获取该集合价格最低的100个订单
// 3. 重新构建优先级队列并更新内存中的订单信息
// 参数说明:
// - address: NFT集合地址
// 返回值:
// - error: 错误信息
func (om *OrderManager) reloadCollectionOrders(address string) error {
	// 1. 检查集合是否被跟踪
	tradeInfo, ok := om.collectionOrders[strings.ToLower(address)]
	if !ok {
		xzap.WithContext(om.Ctx).Warn("untracked collection", zap.String("collection_addr", address))
		return errors.New("untracked collection")
	}

	// 2. 获取该集合价格最低的100个订单
	orders, err := om.getLowestPrice100Orders(address)
	if err != nil {
		return errors.Wrap(err, "failed on get 100 orders at the lowest price")
	}

	// 3. 重新构建优先级队列
	tradeInfo.orders = NewPriorityQueueMap(maxQueueLength)
	// 将订单添加到优先级队列中
	for _, order := range orders {
		tradeInfo.orders.Add(order.OrderID, order.Price, order.Maker, order.TokenId)
	}
	// 更新内存中的订单信息
	om.collectionOrders[strings.ToLower(address)] = tradeInfo
	return nil
}

type ValidOrder struct {
	multi.Order
	IsOpenseaBanned bool `json:"is_opensea_banned"`
}

// getUserValidOrders 函数用于获取指定用户在特定NFT上的有效订单
// 主要功能:
// 1. 查询指定NFT集合地址、tokenID和用户地址的订单信息
// 2. 过滤条件包括:
//   - 订单类型为listing
//   - 订单状态为active
//   - maker必须是token的owner
//   - 非OpenSea禁止的item
//
// 3. 按价格升序排序并限制返回100条记录
// 参数说明:
// - address: NFT集合地址
// - tokenID: NFT的tokenID
// - maker: 订单创建者地址
// 返回值:
// - []*ValidOrder: 有效订单列表
// - error: 错误信息
func (om *OrderManager) getUserValidOrders(address, tokenID, maker string) ([]*ValidOrder, error) {
	var orders []*ValidOrder
	if err := om.DB.WithContext(om.Ctx).Table(fmt.Sprintf("%s as co", gdb.GetMultiProjectOrderTableName(om.project, om.chain))).
		Select("co.id as id,co.order_id as order_id, co.maker as maker,ci.is_opensea_banned as is_opensea_banned, co.collection_address as collection_address, co.price as price,co.token_id as token_id").
		Joins(fmt.Sprintf("join %s ci on co.collection_address = ci.collection_address and co.token_id = ci.token_id", gdb.GetMultiProjectItemTableName(om.project, om.chain))).
		Where("co.order_type=? and co.order_status = ? and co.maker = ci.owner  and (ci.is_opensea_banned,co.marketplace_id)!=(true,1)", multi.ListingType, multi.OrderStatusActive).
		Where("co.collection_address = ? and co.token_id=?"+
			" and co.maker = ?", address, tokenID, maker).Order("co.price asc").Limit(100).
		Scan(&orders).Error; err != nil {
		return nil, errors.Wrap(err, "failed on get user orders")
	}

	return orders, nil
}

// addUpdateFloorPriceEvent 函数用于添加更新地板价的事件
// 主要功能:
// 1. 验证事件的合法性:
//   - 对于Transfer/ImportCollection/UpdateCollection事件,必须有OrderId
//   - 对于Listing事件,价格不能为0
//
// 2. 将事件序列化并添加到Redis队列中
// 参数说明:
// - event: 交易事件,包含事件类型、订单ID、价格等信息
// 返回值:
// - error: 错误信息
func (om *OrderManager) addUpdateFloorPriceEvent(event *TradeEvent) error {
	// 验证事件合法性
	// 如果是Transfer/ImportCollection/UpdateCollection事件,必须有OrderId
	if event.EventType != Transfer && event.EventType != ImportCollection && event.EventType != UpdateCollection && event.OrderId == "" {
		return errors.New("invalid update collection floor price. event order id is null")
	}
	// 如果是Listing事件,价格不能为0
	if event.EventType == Listing && event.Price.IsZero() {
		return errors.New("invalid update collection floor price. price is 0")
	}

	// 序列化事件
	rawEvent, err := json.Marshal(event)
	if err != nil {
		return errors.Wrap(err, "failed on marshal event")
	}

	// 获取Redis队列key
	key := genTradeEventsCacheKey(om.chain)
	// 将事件添加到Redis队列
	if _, err := om.Xkv.Rpush(key, string(rawEvent)); err != nil {
		return errors.Wrap(err, "failed on push trade event to queue")
	}
	return nil
}

func genTradeEventsCacheKey(chain string) string {
	return fmt.Sprintf(CacheTradeEventsQueuePre, chain)
}

// AddUpdatePriceEvent 函数用于添加更新价格的事件
// 主要功能:
// 1. 验证事件的合法性:
//   - 对于非Transfer/ImportCollection/UpdateCollection事件,必须有OrderId
//   - 对于Listing事件,价格不能为0且必须有TokenID
//
// 2. 将事件序列化并添加到Redis队列中
// 参数说明:
// - kv: Redis存储实例
// - event: 交易事件,包含事件类型、订单ID、价格等信息
// - chain: 链标识
// 返回值:
// - error: 错误信息
func AddUpdatePriceEvent(kv *xkv.Store, event *TradeEvent, chain string) error {
	// 验证事件合法性
	// 如果是非Transfer/ImportCollection/UpdateCollection事件,必须有OrderId
	if event.EventType != Transfer && event.EventType != ImportCollection && event.EventType != UpdateCollection && event.OrderId == "" {
		return errors.New("invalid update collection floor price. event order id is null")
	}
	// 如果是Listing事件,价格不能为0
	if event.EventType == Listing && event.Price.IsZero() {
		return errors.New("invalid update collection floor price. price is 0")
	}
	// 如果是Listing事件,必须有TokenID
	if event.EventType == Listing && event.TokenID == "" {
		return errors.New("invalid update collection floor price. token_id is null")
	}

	// 序列化事件
	rawEvent, err := json.Marshal(event)
	if err != nil {
		return errors.Wrap(err, "failed on marshal event")
	}

	// 获取Redis队列key
	key := genTradeEventsCacheKey(chain)
	// 将事件添加到Redis队列
	if _, err := kv.Rpush(key, string(rawEvent)); err != nil {
		return errors.Wrap(err, "failed on push trade event to queue")
	}
	return nil
}
