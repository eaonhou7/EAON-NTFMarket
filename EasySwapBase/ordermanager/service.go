package ordermanager

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"github.com/zeromicro/go-zero/core/threading"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"github.com/ProjectsTask/EasySwapBase/logger/xzap"
	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/multi"
	"github.com/ProjectsTask/EasySwapBase/stores/xkv"
)

const (
	// WheelSize : length of time wheel, 1s for a unit, 1h for a cycle
	WheelSize           = 3600
	List                = 3
	CacheOrdersQueuePre = "cache:es:orders:%s"
)

func GenOrdersCacheKey(chain string) string {
	return fmt.Sprintf(CacheOrdersQueuePre, chain)
}

// Activity ：collection order-activity
type Order struct {
	// order Id
	orderID        string
	CollectionAddr string
	// chain suffix name: ethw/bsc
	ChainSuffix string
	// expireIn - createIn (unit: s)
	CycleCount int64
	// position of the task on the time wheel
	WheelPosition int64

	Next *Order
}

type wheel struct {
	// linked list
	NotifyActivities *Order
}

type OrderManager struct {
	chain string

	// cycle time wheel
	TimeWheel [WheelSize]wheel
	// current time wheel index
	CurrentIndex int64

	collectionOrders map[string]*collectionTradeInfo

	collectionListedCh chan string
	project            string

	Xkv *xkv.Store
	DB  *gorm.DB
	Ctx context.Context
	Mux *sync.RWMutex
}

// NewDelayQueue : create func instance entrance
func New(ctx context.Context, db *gorm.DB, xkv *xkv.Store, chain string, project string) *OrderManager {
	return &OrderManager{
		chain:              chain,
		Xkv:                xkv,
		DB:                 db,
		Ctx:                ctx,
		Mux:                new(sync.RWMutex),
		collectionOrders:   make(map[string]*collectionTradeInfo),
		collectionListedCh: make(chan string, 1000),
		project:            project,
	}
}

func (om *OrderManager) Start() {
	// listen redis cache
	threading.GoSafe(om.ListenNewListingLoop) // 处理新订单
	threading.GoSafe(om.orderExpiryProcess)   // 处理订单过期状态
	threading.GoSafe(om.floorPriceProcess)    // 处理floorprice更新
	threading.GoSafe(om.listCountProcess)     // 处理listCount更新
}

func (om *OrderManager) Stop() {
}

type ListingInfo struct {
	ExpireIn       int64           `json:"expire_in"`
	OrderId        string          `json:"order_id"`
	CollectionAddr string          `json:"collection_addr"`
	TokenID        string          `json:"token_id"`
	Price          decimal.Decimal `json:"price"`
	Maker          string          `json:"maker"`
}

func (om *OrderManager) ListenNewListingLoop() {
	key := GenOrdersCacheKey(om.chain)
	for {
		result, err := om.Xkv.Lpop(key)
		if err != nil || result == "" {
			if err != nil && err != redis.Nil {
				xzap.WithContext(context.Background()).Warn("failed on get order from cache", zap.Error(err), zap.String("result", result))
			}
			time.Sleep(1 * time.Second)
			continue
		}

		xzap.WithContext(om.Ctx).Info("get listing from cache", zap.String("result", result))
		var listing ListingInfo
		if err := json.Unmarshal([]byte(result), &listing); err != nil {
			xzap.WithContext(om.Ctx).Warn("failed on Unmarshal order info", zap.Error(err))
			continue
		}
		if listing.OrderId == "" {
			xzap.WithContext(om.Ctx).Error("invalid null order id")
			continue
		}

		if listing.ExpireIn < time.Now().Unix() { // 订单已经过期
			xzap.WithContext(om.Ctx).Info("expired activity order", zap.String("order_id", listing.OrderId))

			// 更新订单状态
			if err := om.updateOrdersStatus(listing.OrderId, multi.OrderStatusExpired); err != nil {
				xzap.WithContext(om.Ctx).Error("failed on update activity status", zap.String("order_id", listing.OrderId), zap.Error(err))
			}

			// 添加更新floorprice事件
			if err := om.addUpdateFloorPriceEvent(&TradeEvent{
				EventType:      Expired,
				CollectionAddr: listing.CollectionAddr,
				TokenID:        listing.TokenID,
				OrderId:        listing.OrderId,
				From:           listing.Maker,
			}); err != nil {
				xzap.WithContext(om.Ctx).Error("failed on add update floor price event", zap.String("order_id", listing.OrderId), zap.Error(err))
			}
			continue
		} else { // 订单未过期
			if err := om.addUpdateFloorPriceEvent(&TradeEvent{ // 添加更新floorprice事件
				EventType:      Listing,
				CollectionAddr: listing.CollectionAddr,
				TokenID:        listing.TokenID,
				OrderId:        listing.OrderId,
				Price:          listing.Price,
				From:           listing.Maker,
			}); err != nil {
				xzap.WithContext(om.Ctx).Error("failed on push order to update price queue", zap.Error(err), zap.String("order_id", listing.OrderId),
					zap.String("order_id", listing.OrderId),
					zap.String("price", listing.Price.String()),
					zap.String("chain", om.chain))
			}

			// 添加到订单过期检查队列
			delaySeconds := listing.ExpireIn - time.Now().Unix()
			if err := om.addToOrderExpiryCheckQueue(delaySeconds, om.chain, listing.OrderId, listing.CollectionAddr); err != nil {
				xzap.WithContext(om.Ctx).Error("failed on push order to expired check queue", zap.Error(err), zap.String("order_id", listing.OrderId),
					zap.String("chain", om.chain))
			}
		}
	}
}

func (om *OrderManager) AddToOrderManagerQueue(order *multi.Order) error {
	if order.TokenId == "" {
		return errors.New("order manger need token id")
	}
	rawInfo, err := json.Marshal(ListingInfo{
		ExpireIn:       order.ExpireTime,
		OrderId:        order.OrderID,
		CollectionAddr: order.CollectionAddress,
		TokenID:        order.TokenId,
		Price:          order.Price,
		Maker:          order.Maker,
	})
	if err != nil {
		return errors.Wrap(err, "failed on marshal listing info")
	}

	if _, err := om.Xkv.Lpush(GenOrdersCacheKey(om.chain), string(rawInfo)); err != nil {
		return errors.Wrap(err, "failed on add to queue")
	}

	return nil
}
