package dao

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/multi"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ProjectsTask/EasySwapBackend/src/types/v1"
)

const (
	BuyNow   = 1
	HasOffer = 2
	All      = 3
)

const (
	listTime      = 0
	listPriceAsc  = 1
	listPriceDesc = 2
	salePriceDesc = 3
	salePriceAsc  = 4
)

type CollectionItem struct {
	multi.Item
	MarketID       int    `json:"market_id"`
	Listing        bool   `json:"listing"`
	OrderID        string `json:"order_id"`
	OrderStatus    int    `json:"order_status"`
	ListMaker      string `json:"list_maker"`
	ListTime       int64  `json:"list_time"`
	ListExpireTime int64  `json:"list_expire_time"`
	ListSalt       int64  `json:"list_salt"`
}

// QueryCollectionBids 查询NFT集合的出价信息 (Collection Offers)
// 功能: 获取某个集合的"有效"出价单分布情况
// 统计维度: 按价格(Price)分组统计
// 统计指标:
//   - size: 该价格下的挂单总份数 (quantity_remaining)
//   - total: 该价格下的总资金规模 (size * price)
//   - bidders: 该价格下的独立出价人数 (count distinct maker)
func (d *Dao) QueryCollectionBids(ctx context.Context, chain string, collectionAddr string, page, pageSize int) ([]types.CollectionBids, int64, error) {
	var count int64

	// 1. 统计不同价格档位的数量 (用于分页)
	// SQL逻辑:
	// SELECT count(*) FROM (
	//   SELECT price FROM orders
	//   WHERE collection_address = ? AND order_type = CollectionBid AND status = Active AND expire_time > Now
	//   GROUP BY price
	// )
	if err := d.DB.WithContext(ctx).
		Table(multi.OrderTableName(chain)).
		Where("collection_address = ? and order_type = ? and order_status = ? and expire_time > ?",
			collectionAddr, multi.CollectionBidOrder, multi.OrderStatusActive, time.Now().Unix()).
		Group("price").
		Count(&count).Error; err != nil {
		return nil, 0, errors.Wrap(err, "failed on count user items")
	}

	var bids []types.CollectionBids
	db := d.DB.WithContext(ctx).Table(multi.OrderTableName(chain))

	// 2. 查询详细的出价分布
	// 注意: quantity_remaining > 0 确保只统计还有效的余额
	if err := db.Select(`
			sum(quantity_remaining) AS size, 
			price,
			sum(quantity_remaining)*price as total,
			COUNT(DISTINCT maker) AS bidders`).
		Where(`collection_address = ? and order_type = ? and order_status = ? 
			   and expire_time > ? and quantity_remaining > 0`,
			collectionAddr, multi.CollectionBidOrder, multi.OrderStatusActive, time.Now().Unix()).
		Group("price").
		Order("price desc"). // 按价格从高到低排序 (买单通常看最高价)
		Limit(int(pageSize)).
		Offset(int(pageSize * (page - 1))).
		Scan(&bids).Error; err != nil {
		return nil, 0, errors.Wrap(err, "failed on query collection bids")
	}

	return bids, count, nil
}

// QueryCollectionItemOrder 查询集合内 NFT Item 的市场状态信息
// 功能: 支持复杂的筛选条件 (Status, Markets, TokenID, UserAddress)
// 核心逻辑: 动态构建 SQL, 联结 Items 和 Orders 表, 计算 Floor Price / Best Offer
// Status 枚举:
//
//	1: BuyNow (只看有挂卖单的)
//	2: HasOffer (只看有收到 Offer 的)
//	3: All (全部 Items, 附带挂单/Offer状态)
func (d *Dao) QueryCollectionItemOrder(ctx context.Context, chain string, filter types.CollectionItemFilterParams, collectionAddr string) ([]*CollectionItem, int64, error) {
	// 默认只查询 OrderBookDex 市场
	if len(filter.Markets) == 0 {
		filter.Markets = []int{int(multi.OrderBookDex)}
	}

	// 基础表: Items 表 (ci)
	db := d.DB.WithContext(ctx).Table(fmt.Sprintf("%s as ci", multi.ItemTableName(chain)))
	coTableName := multi.OrderTableName(chain)

	// 根据 Status 构建不同的查询逻辑
	// Status 为单选数组: [1] 或 [2] 或 [1,2](特殊逻辑?) 或 [3]
	if len(filter.Status) == 1 {
		// 基础 SELECT 字段
		// market_id 生成逻辑:
		//   GROUP_CONCAT + ORDER BY + SUBSTRING_INDEX
		//   找出价格最低的那个订单对应的 marketplace_id
		db.Select(
			"ci.id as id, ci.chain_id as chain_id, " +
				"ci.collection_address as collection_address,ci.token_id as token_id, " +
				"ci.name as name, ci.owner as owner, " +
				"min(co.price) as list_price, " + // 最低挂单价
				"SUBSTRING_INDEX(GROUP_CONCAT(co.marketplace_id ORDER BY co.price,co.marketplace_id),',', 1) AS market_id, " +
				"min(co.price) != 0 as listing") // 如果有价格则标记 listing=true

		// Case 1: BuyNow (查询正在出售的 Items)
		if filter.Status[0] == BuyNow {
			// INNER JOIN Orders: 必须有对应的出售单
			db.Joins(fmt.Sprintf(
				"join %s co on co.collection_address=ci.collection_address and co.token_id=ci.token_id",
				coTableName)).
				Where(
					"co.collection_address = ? and co.order_type = ? and co.order_status=? "+
						"and co.maker = ci.owner", // 挂单者必须是当前持有者
					collectionAddr, multi.ListingOrder, multi.OrderStatusActive)

			// 市场过滤
			if len(filter.Markets) == 1 {
				db.Where("co.marketplace_id = ?", filter.Markets[0])
			} else if len(filter.Markets) != 5 {
				db.Where("co.marketplace_id in (?)", filter.Markets)
			}

			// TokenID 精确查找
			if filter.TokenID != "" {
				db.Where("co.token_id =?", filter.TokenID)
			}
			// User (Owner/Maker) 查找
			if filter.UserAddress != "" {
				db.Where("ci.owner =?", filter.UserAddress)
			}

			db.Group("co.token_id")
		}

		// Case 2: HasOffer (查询收到 Offer 的 Items)
		if filter.Status[0] == HasOffer {
			// INNER JOIN Orders: 必须有对应的 Offer 单 (OfferOrder)
			db.Joins(fmt.Sprintf(
				"join %s co on co.collection_address=ci.collection_address and co.token_id=ci.token_id",
				coTableName)).
				Where(
					"co.collection_address = ? and co.order_type = ? and co.order_status = ?",
					collectionAddr, multi.OfferOrder, multi.OrderStatusActive)

			if len(filter.Markets) == 1 {
				db.Where("co.marketplace_id = ?", filter.Markets[0])
			} else if len(filter.Markets) != 5 {
				db.Where("co.marketplace_id in (?)", filter.Markets)
			}

			if filter.TokenID != "" {
				db.Where("co.token_id =?", filter.TokenID)
			}
			if filter.UserAddress != "" {
				db.Where("ci.owner =?", filter.UserAddress)
			}

			db.Group("co.token_id")
		}
	} else if len(filter.Status) == 2 {
		// Case 3: BuyNow AND HasOffer (同时满足正在出售且有 Offer?)
		// 原逻辑: len(Status) == 2, 假设是 [1, 2] ???
		// SQL 逻辑: HAVING min(type) = Listing AND max(type) = Offer ???
		// 该逻辑表示同一个 TokenID 分组下，既有 ListingOrder 又有 OfferOrder。
		db.Select(
			"ci.id as id, ci.chain_id as chain_id," +
				"ci.collection_address as collection_address,ci.token_id as token_id, " +
				"ci.name as name, ci.owner as owner, " +
				"min(co.price) as list_price, " +
				"SUBSTRING_INDEX(GROUP_CONCAT(co.marketplace_id ORDER BY co.price,co.marketplace_id),',', 1) AS market_id")

		db.Joins(fmt.Sprintf(
			"join %s co on co.collection_address=ci.collection_address and co.token_id=ci.token_id",
			coTableName)).
			Where(
				"co.collection_address = ? and co.order_status=? and co.maker = ci.owner",
				collectionAddr, multi.OrderStatusActive)

		// ... (Filters same as above) ...

		// ... (Filters part 1 above)

		// 市场过滤
		if len(filter.Markets) == 1 {
			db.Where("co.marketplace_id = ?", filter.Markets[0])
		} else if len(filter.Markets) != 5 {
			db.Where("co.marketplace_id in (?)", filter.Markets)
		}

		// Token / User 过滤
		if filter.TokenID != "" {
			db.Where("co.token_id =?", filter.TokenID)
		}
		if filter.UserAddress != "" {
			db.Where("ci.owner =?", filter.UserAddress)
		}

		// Having 条件: 同时存在 Listing 和 Offer
		// 注意: 这里 type 比较逻辑可能有误 (ListingOrder vs OfferOrder 数字大小问题),
		// 但暂且保留原意: 该 Group 下 min(type) 和 max(type) 不同, 说明混合了两种订单
		db.Group("co.token_id").Having(
			"min(co.type)=? and max(co.type)=?",
			multi.ListingOrder, multi.OfferOrder)

	} else {
		// Case 4: Status 3 or Empty (All Items)
		// 需求: 列出所有 Items, 如果有挂单则显示最低挂单价

		// 1. 子查询: 仅查询有挂单的 Token 统计信息
		//    目的: 提前聚合 Orders 表, 避免主查询 Left Join 全表导致性能问题
		subQuery := d.DB.WithContext(ctx).Table(
			fmt.Sprintf("%s as cis", multi.ItemTableName(chain))).
			Select(
				"cis.id as item_id,cis.collection_address as collection_address,"+
					"cis.token_id as token_id, cis.owner as owner, cos.order_id as order_id, "+
					"min(cos.price) as list_price, "+
					"SUBSTRING_INDEX(GROUP_CONCAT(cos.marketplace_id ORDER BY cos.price,cos.marketplace_id),',', 1) AS market_id, "+
					"min(cos.price) != 0 as listing").
			Joins(fmt.Sprintf(
				"join %s cos on cos.collection_address=cis.collection_address and cos.token_id=cis.token_id",
				coTableName)).
			Where(
				"cos.collection_address = ? and cos.order_type = ? and cos.order_status=? "+
					"and cos.maker = cis.owner", // 挂单者 = 持有者
				collectionAddr, multi.ListingOrder, multi.OrderStatusActive)

		if len(filter.Markets) == 1 {
			subQuery.Where("cos.marketplace_id = ?", filter.Markets[0])
		} else if len(filter.Markets) != 5 {
			subQuery.Where("cos.marketplace_id in (?)", filter.Markets)
		}
		subQuery.Group("cos.token_id")

		// 2. 主查询: Items 表 LEFT JOIN 子查询结果
		db.Joins("left join (?) co on co.collection_address=ci.collection_address and co.token_id=ci.token_id",
			subQuery).
			Select(
				"ci.id as id, ci.chain_id as chain_id," +
					"ci.collection_address as collection_address, ci.token_id as token_id, " +
					"ci.name as name, ci.owner as owner, " +
					"co.list_price as list_price, co.market_id as market_id, co.listing as listing").
			Where(fmt.Sprintf("ci.collection_address = '%s'", collectionAddr))

		if filter.TokenID != "" {
			db.Where(fmt.Sprintf("ci.token_id = '%s'", filter.TokenID))
		}
		if filter.UserAddress != "" {
			db.Where(fmt.Sprintf("ci.owner = '%s'", filter.UserAddress))
		}
	}

	// -------------------------------------------------------------
	// 统计总数 (Count)
	// -------------------------------------------------------------
	var count int64
	countTx := db.Session(&gorm.Session{})
	if err := countTx.Count(&count).Error; err != nil {
		return nil, 0, errors.Wrap(db.Error, "failed on count items")
	}

	// -------------------------------------------------------------
	// 排序 (Order By)
	// -------------------------------------------------------------

	// 若无状态筛选, 默认优先显示 listing=true 即 "在售" 的 Item
	if len(filter.Status) == 0 {
		db.Order("listing desc")
	}

	if filter.Sort == 0 {
		filter.Sort = listPriceAsc
	}

	// 根据不同排序条件设置 ORDER BY
	switch filter.Sort {
	case listTime:
		// 最新上架
		db.Order("list_time desc,ci.id asc")
	case listPriceAsc:
		// 价格升序 (价格相同按 ID 顺序)
		db.Order("list_price asc, ci.id asc")
	case listPriceDesc:
		// 价格降序
		db.Order("list_price desc,ci.id asc")
	// 注意: 下面两个 case 涉及 sale_price, 但上文 Select 中并未查询 sale_price ???
	// 这可能是一个 Bug 或者依赖隐式 Join, 需确认 sale_price 来源。
	// 在 QueryCollectionsSellPrice 或其他逻辑中有 sale_price, 但此处 Select 中只有 list_price.
	// 假设 list_price 是意图, 或者 sale_price 是笔误. 暂保留原样.
	case salePriceDesc:
		db.Order("sale_price desc,ci.id asc")
	case salePriceAsc:
		db.Order("sale_price = 0,sale_price asc,ci.id asc")
	}

	// -------------------------------------------------------------
	// 执行分页查询 (Pagination)
	// -------------------------------------------------------------
	var items []*CollectionItem
	db.Offset(int((filter.Page - 1) * filter.PageSize)).
		Limit(int(filter.PageSize)).
		Scan(&items)

	if db.Error != nil {
		return nil, 0, errors.Wrap(db.Error, "failed on get query items info")
	}

	return items, count, nil
}

type UserItemCount struct {
	Owner  string `json:"owner"`
	Counts int64  `json:"counts"`
}

// QueryUsersItemCount 查询指定用户在某 Collection 中的持仓数量
// 参数:
//   - owners: 批量查询的用户地址列表
//
// 返回:
//   - UserItemCount 列表 {Owner, Counts}
func (d *Dao) QueryUsersItemCount(ctx context.Context, chain string,
	collectionAddr string, owners []string) ([]UserItemCount, error) {

	var itemCount []UserItemCount

	// SQL: SELECT owner, COUNT(*) FROM items
	// WHERE collection_address = ? AND owner IN (?) GROUP BY owner
	if err := d.DB.WithContext(ctx).
		Table(fmt.Sprintf("%s as ci", multi.ItemTableName(chain))).
		Select("owner,COUNT(*) AS counts").
		Where("collection_address = ? and owner in (?)",
			collectionAddr, owners).
		Group("owner").
		Scan(&itemCount).Error; err != nil {
		return nil, errors.Wrap(err, "failed on get user item count")
	}

	return itemCount, nil
}

// QueryLastSalePrice 批量查询 Item 的最近一次成交价
// 功能: 用于在 Item 列表页显示 "Last Sale: x ETH"
// 逻辑:
// 1. 找到每个 Item 最近一次 (Max EventTime) 的 Sale 类型 Activity
// 2. Join 原表获取对应的 Price
func (d *Dao) QueryLastSalePrice(ctx context.Context, chain string,
	collectionAddr string, tokenIds []string) ([]multi.Activity, error) {
	var lastSales []multi.Activity

	// SQL 逻辑:
	// INNER JOIN 自关联查询
	// 子查询 groupedA: 找出 Collection+TokenID 分组下的 MAX(event_time)
	// 主查询 a: 根据 (Address, TokenID, EventTime) 匹配获取 Price
	sql := fmt.Sprintf(`
		SELECT a.collection_address, a.token_id, a.price
		FROM %s a
		INNER JOIN (
			SELECT collection_address,token_id, 
				MAX(event_time) as max_event_time
			FROM %s
			WHERE collection_address = ?
				AND token_id IN (?)
				AND activity_type = ?
			GROUP BY collection_address,token_id
		) groupedA 
		ON a.collection_address = groupedA.collection_address
		AND a.token_id = groupedA.token_id
		AND a.event_time = groupedA.max_event_time
		AND a.activity_type = ?`,
		multi.ActivityTableName(chain),
		multi.ActivityTableName(chain))

	if err := d.DB.Raw(sql, collectionAddr, tokenIds,
		multi.Sale, multi.Sale).Scan(&lastSales).Error; err != nil {
		return nil, errors.Wrap(err, "failed on get item last sale price")
	}

	return lastSales, nil
}

// QueryBestBids 查询 Item 级别的最佳出价 (Item Offer)
// 注意: 这是针对 *Specific Token* (Item Bid) 的查询, 不包含 Collection Offer
// 功能: 获取指定 items 的最高有效出价
func (d *Dao) QueryBestBids(ctx context.Context, chain string, userAddr string,
	collectionAddr string, tokenIds []string) ([]multi.Order, error) {
	var bestBids []multi.Order
	var sql string

	// 查询条件:
	// - OrderType = ItemBidOrder (2)
	// - Status = Active
	// - Unexpired & QuantityRemaining > 0
	// - (Optional) Maker != userAddr (排除自己的出价)
	baseSql := fmt.Sprintf(`
			SELECT order_id, token_id, event_time, price, salt, 
				expire_time, maker, order_type, quantity_remaining, size   
			FROM %s
			WHERE collection_address = ?
				AND token_id IN (?)
				AND order_type = ?
				AND order_status = ?
				AND expire_time > ?
				AND quantity_remaining > 0
		`, multi.OrderTableName(chain))

	if userAddr == "" {
		sql = baseSql
	} else {
		sql = baseSql + fmt.Sprintf(" AND maker != '%s'", userAddr)
	}

	// 注意: 这里并没有 GROUP BY token_id 取 MAX(Price),
	// 而是返回了所有符合条件的 Bids?
	// 函数名 QueryBestBids 暗示返回"最佳", 但 SQL 似乎返回列表.
	// 调用方可能需要自己处理, 或者这里只是获取所有有效出价.
	if err := d.DB.Raw(sql, collectionAddr, tokenIds,
		multi.ItemBidOrder, multi.OrderStatusActive,
		time.Now().Unix()).Scan(&bestBids).Error; err != nil {
		return nil, errors.Wrap(err, "failed on get item best bids")
	}

	return bestBids, nil
}

// QueryItemsBestBids 批量查询多个 Items 的最佳出价
// 与 QueryBestBids 类似, 但入参是 []ItemInfo (包含不同的 CollectionAddr)
// 适用于列表页混合展示不同 Collection 的场景
func (d *Dao) QueryItemsBestBids(ctx context.Context, chain string, userAddr string, itemInfos []types.ItemInfo) ([]multi.Order, error) {
	// 1. 构建 (Collection, TokenID) 复合条件
	var conditions []clause.Expr
	for _, info := range itemInfos {
		conditions = append(conditions, gorm.Expr("(?, ?)", info.CollectionAddress, info.TokenID))
	}

	var bestBids []multi.Order
	var sql string

	baseSql := fmt.Sprintf(`
SELECT order_id, token_id, event_time, price, salt, expire_time, maker, order_type, quantity_remaining, size
    FROM %s
    WHERE (collection_address,token_id) IN (?)
      AND order_type = ?
      AND order_status = ?
	  AND quantity_remaining > 0
      AND expire_time > ?
`, multi.OrderTableName(chain))

	if userAddr == "" {
		sql = baseSql
	} else {
		sql = baseSql + fmt.Sprintf(" AND maker != '%s'", userAddr)
	}

	if err := d.DB.Raw(sql, conditions, multi.ItemBidOrder, multi.OrderStatusActive, time.Now().Unix()).Scan(&bestBids).Error; err != nil {
		return nil, errors.Wrap(err, "failed on get item best bids")
	}

	return bestBids, nil
}

// QueryCollectionsBestBid 批量查询多个集合的"最高" Collection Offer (集合级出价)
// 功能: 用于在集合列表页显示每个集合的最佳 Offer
func (d *Dao) QueryCollectionsBestBid(ctx context.Context, chain string, userAddr string, collectionAddrs []string) ([]*multi.Order, error) {
	var bestBid []*multi.Order

	// SQL 逻辑分析:
	// 需求: 对每个 collection_address, 找到 price 最高的那个 Active Collection Offer
	// 1. 基础表: orders 表
	// 2. 过滤条件:
	//    - OrderType = CollectionBidOrder
	//    - Status = Active & Unexpired & Remaining > 0
	// 3. 分组求 Max Price: (SELECT collection_address, max(price) FROM orders WHERE ... GROUP BY address)
	// 4. 主查询: 匹配 (Address, Price) IN 子查询结果
	//    注意: 这种写法可能会返回多条记录如果最高价有多个订单(需业务层处理或依赖 Limit/Order)

	sql := fmt.Sprintf(`
		SELECT collection_address, order_id, price,event_time, expire_time, salt, maker, order_type, quantity_remaining, size  
		FROM %s `, multi.OrderTableName(chain))

	// SubQuery: Find MAX Price per Collection
	sql += fmt.Sprintf(`where (collection_address,price) in (SELECT collection_address, max(price) as price FROM %s `, multi.OrderTableName(chain))

	// SubQuery Filters
	sql += `where collection_address in (?) and order_type = ? and order_status = ? and quantity_remaining > 0 and expire_time > ? `
	if userAddr != "" {
		sql += fmt.Sprintf(" and maker != '%s'", userAddr)
	}
	sql += `group by collection_address ) `

	// MainQuery Filters (必须再次重复条件以确保匹配到正确的 Order)
	sql += `and order_type = ? and order_status = ? and quantity_remaining > 0 and expire_time > ? `
	if userAddr != "" {
		sql += fmt.Sprintf(" and maker != '%s'", userAddr)
	}

	now := time.Now().Unix()
	if err := d.DB.Raw(sql,
		collectionAddrs, multi.CollectionBidOrder, multi.OrderStatusActive, now, // subquery params
		multi.CollectionBidOrder, multi.OrderStatusActive, now, // mainquery params
	).Scan(&bestBid).Error; err != nil {
		return bestBid, errors.Wrap(err, "failed on get item best bids")
	}

	return bestBid, nil
}

// QueryCollectionBestBid 查询单个集合的最高 Collection Offer (Limit 1)
// 功能: 直接返回价格最高的一个订单
func (d *Dao) QueryCollectionBestBid(ctx context.Context, chain string,
	userAddr string, collectionAddr string) (multi.Order, error) {
	var bestBid multi.Order

	baseSql := fmt.Sprintf(`
			SELECT order_id, price, event_time, expire_time, salt, maker, 
				order_type, quantity_remaining, size  
			FROM %s
			WHERE collection_address = ?
			AND order_type = ?
			AND order_status = ?
			AND quantity_remaining > 0
			AND expire_time > ? 
		`, multi.OrderTableName(chain))

	var sql string
	if userAddr == "" {
		sql = baseSql
	} else {
		sql = baseSql + fmt.Sprintf(" AND maker != '%s'", userAddr)
	}

	// 按价格降序取 Limit 1
	sql += " ORDER BY price DESC LIMIT 1"

	if err := d.DB.Raw(sql, collectionAddr, multi.CollectionBidOrder,
		multi.OrderStatusActive, time.Now().Unix()).Scan(&bestBid).Error; err != nil {
		return bestBid, errors.Wrap(err, "failed on get item best bids")
	}

	return bestBid, nil
}

// QueryCollectionTopNBid 查询集合的前 N 个最高出价 (用于深度图或 Top Offer 列表)
// 功能: 支持按 quantity_remaining 展开 (Expand)
// 逻辑:
// 1. 查询 Top N 的订单列表 (Order By Price DESC Limit N)
// 2. 内存中展开: 如果一个订单 Quantity=2, 则在结果列表中占 2 个位置 (模拟订单簿深度)
// 3. 截取前 N 个返回
func (d *Dao) QueryCollectionTopNBid(ctx context.Context, chain string,
	userAddr string, collectionAddr string, num int) ([]multi.Order, error) {
	var bestBids []multi.Order

	baseSql := fmt.Sprintf(`
			SELECT order_id, price, event_time, expire_time, salt, maker, 
				order_type, quantity_remaining, size 
			FROM %s
			WHERE collection_address = ?
				AND order_type = ?
				AND order_status = ?
				AND quantity_remaining > 0
				AND expire_time > ? 
		`, multi.OrderTableName(chain))

	var sql string
	if userAddr == "" {
		sql = baseSql
	} else {
		sql = baseSql + fmt.Sprintf(" AND maker != '%s'", userAddr)
	}

	// 这里的 Limit num 实际上可能不够，因为后续还有"展开"逻辑
	// 比如 Limit 10, 但第一个订单 Quantity=10, 展开后就够 10 个了.
	// 但如果 Quantity=1, 则只取10个订单.
	// 考虑到 depth 图通常不需要太深, Limit N 作为 DB 限制是可以接受的优化.
	sql += fmt.Sprintf(" ORDER BY price DESC LIMIT %d", num)

	if err := d.DB.Raw(sql, collectionAddr, multi.CollectionBidOrder,
		multi.OrderStatusActive, time.Now().Unix()).Scan(&bestBids).Error; err != nil {
		return nil, errors.Wrap(err, "failed on get item best bids")
	}

	// 订单深度展开逻辑
	var results []multi.Order
	for i := 0; i < len(bestBids); i++ {
		qty := int(bestBids[i].QuantityRemaining)
		for j := 0; j < qty; j++ {
			results = append(results, bestBids[i])
			if len(results) >= num {
				break
			}
		}
		if len(results) >= num {
			break
		}
	}

	return results, nil
}

var collectionDetailFields = []string{"id", "chain_id", "token_standard", "name", "address", "image_uri", "floor_price", "sale_price", "item_amount", "owner_amount"}

const OrderType = 1
const OrderStatus = 0

// QueryListedAmount 查询集合中已上架 NFT 的数量 (Distinct TokenID)
// 功能: 统计某个集合当前有多少个独立的 Token 正在 Listing 状态
func (d *Dao) QueryListedAmount(ctx context.Context, chain string, collectionAddr string) (int64, error) {
	// SQL解释:
	// 1. INNER JOIN 连接 Items(ci) 和 Orders(co)
	// 2. 过滤条件:
	//    - 指定 CollectionAddress
	//    - OrderType = Listing
	//    - OrderStatus = Active
	//    - Maker = Item Owner (防止虚假挂单)
	//    - Exclude Marketplace 1
	// 3. COUNT(DISTINCT co.token_id): 即使同一 Token 有多个挂单(不同平台/价格), 也只算 1 个
	sql := fmt.Sprintf(`SELECT count(distinct (co.token_id)) as counts
			FROM %s as ci
					join %s co on co.collection_address = ci.collection_address and co.token_id = ci.token_id
			WHERE (co.collection_address=? and co.order_type = ? and
				co.order_status = ? and co.maker = ci.owner and co.marketplace_id != ?)
		`, multi.ItemTableName(chain), multi.OrderTableName(chain))

	var counts int64
	if err := d.DB.WithContext(ctx).Raw(
		sql,
		collectionAddr,
		OrderType,
		OrderStatus,
		1,
	).Scan(&counts).Error; err != nil {
		return 0, errors.Wrap(err, "failed on get listed item amount")
	}

	return counts, nil
}

// QueryListedAmountEachCollection 批量查询多个集合的上架数量
// 功能: 列表页使用, 批量获取每个 Collection 的 Listed Count
func (d *Dao) QueryListedAmountEachCollection(ctx context.Context, chain string, collectionAddrs []string, userAddrs []string) ([]types.CollectionInfo, error) {
	var counts []types.CollectionInfo

	// SQL 逻辑:
	// 1. 同时过滤 CollectionAddress IN (?) AND Owner IN (?)
	//    (这个 Owner 过滤似乎是为了查询"特定用户在这些集合中的挂单数"? 而不是集合总挂单数)
	//    (根据函数名 EachCollection 这里的 lists 可能指 用户视角的 listing?)
	//    (查看代码: ci.owner in (?). 如果 userAddrs 传入空可能导致问题, 但通常业务层会处理)
	// 2. Group By CollectionAddress
	sql := fmt.Sprintf(`SELECT  ci.collection_address as address, count(distinct (co.token_id)) as list_amount
			FROM %s as ci
					join %s co on co.collection_address = ci.collection_address and co.token_id = ci.token_id
			WHERE (co.collection_address in (?) and ci.owner in (?) and co.order_type = ? and
				co.order_status = ? and co.maker = ci.owner and co.marketplace_id != ?) group by ci.collection_address`,
		multi.ItemTableName(chain), multi.OrderTableName(chain))

	if err := d.DB.WithContext(ctx).Raw(
		sql,
		collectionAddrs,
		userAddrs,
		OrderType,
		OrderStatus,
		1,
	).Scan(&counts).Error; err != nil {
		return nil, errors.Wrap(err, "failed on get listed item amount")
	}

	return counts, nil
}

type MultiChainItemInfo struct {
	types.ItemInfo
	ChainName string
}

// QueryMultiChainUserItemsListInfo 查询用户在多链上持有 Items 的挂单状态
// 功能: Portfolio 页面显示用户 Items 时, 需要知道哪些是 "Listing" 状态, 以及最低挂单价
// 逻辑:
// 1. 入参: 用户持有的 Items 列表 (包含 Chain, Address, TokenID)
// 2. 按 Chain 分组构建 SQL
// 3. UNION ALL 查询所有链
func (d *Dao) QueryMultiChainUserItemsListInfo(ctx context.Context, userAddrs []string,
	itemInfos []MultiChainItemInfo) ([]*CollectionItem, error) {
	var collectionItems []*CollectionItem

	// 1. 构建 User Filters
	var userAddrsParam string
	for i, addr := range userAddrs {
		userAddrsParam += fmt.Sprintf(`'%s'`, addr)
		if i < len(userAddrs)-1 {
			userAddrsParam += ","
		}
	}

	// 2. 按链分组 ItemInfo
	chainItems := make(map[string][]MultiChainItemInfo)
	for _, itemInfo := range itemInfos {
		items, ok := chainItems[strings.ToLower(itemInfo.ChainName)]
		if ok {
			items = append(items, itemInfo)
			chainItems[strings.ToLower(itemInfo.ChainName)] = items
		} else {
			chainItems[strings.ToLower(itemInfo.ChainName)] = []MultiChainItemInfo{itemInfo}
		}
	}

	sqlHead := "SELECT * FROM ("
	sqlTail := ") as combined"
	var sqlMids []string

	// 3. 遍历链构建子查询
	for chainName, items := range chainItems {
		// 构建 IN ((addr, id), (addr, id)...) 列表
		tmpStat := fmt.Sprintf("(('%s','%s')", items[0].CollectionAddress, items[0].TokenID)
		for i := 1; i < len(items); i++ {
			tmpStat += fmt.Sprintf(",('%s','%s')", items[i].CollectionAddress, items[i].TokenID)
		}
		tmpStat += ") "

		sqlMid := "("
		// Select: Min Price & Best Market
		sqlMid += "select ci.id as id, ci.chain_id as chain_id,"
		sqlMid += "ci.collection_address as collection_address,ci.token_id as token_id, ci.name as name, ci.owner as owner,"
		sqlMid += "min(co.price) as list_price, " +
			"SUBSTRING_INDEX(GROUP_CONCAT(co.marketplace_id ORDER BY co.price,co.marketplace_id),',', 1) " +
			"AS market_id, min(co.price) != 0 as listing "

		sqlMid += fmt.Sprintf("from %s as ci ", multi.ItemTableName(chainName))
		sqlMid += fmt.Sprintf("join %s co ", multi.OrderTableName(chainName))
		sqlMid += "on co.collection_address=ci.collection_address and co.token_id=ci.token_id "

		// 过滤: (Address, TokenID) 匹配 AND Order.Maker = Owner (有效挂单)
		sqlMid += "where (co.collection_address,co.token_id) in "
		sqlMid += tmpStat
		sqlMid += fmt.Sprintf("and co.order_type = %d and co.order_status=%d "+
			"and co.maker = ci.owner and co.maker in (%s) ",
			multi.ListingOrder, multi.OrderStatusActive, userAddrsParam)

		sqlMid += "group by co.collection_address,co.token_id"
		sqlMid += ")"

		sqlMids = append(sqlMids, sqlMid)
	}

	// 4. 执行 UNION 查询
	sql := sqlHead
	for i := 0; i < len(sqlMids); i++ {
		if i != 0 {
			sql += " UNION ALL "
		}
		sql += sqlMids[i]
	}
	sql += sqlTail

	if err := d.DB.WithContext(ctx).Raw(sql).Scan(&collectionItems).Error; err != nil {
		return nil, errors.Wrap(err, "failed on query user multi chain items list info")
	}

	return collectionItems, nil
}

// QueryMultiChainUserItemsExpireListInfo 查询用户 Items 的"过期"或"活跃"挂单
// 逻辑类似 QueryMultiChainUserItemsListInfo, 但增加了 OrderStatusExpired 状态
// 可能用于显示历史挂单记录
func (d *Dao) QueryMultiChainUserItemsExpireListInfo(ctx context.Context, userAddrs []string,
	itemInfos []MultiChainItemInfo) ([]*CollectionItem, error) {
	var collectionItems []*CollectionItem

	// (Similar logic for User params)
	var userAddrsParam string
	for i, addr := range userAddrs {
		userAddrsParam += fmt.Sprintf(`'%s'`, addr)
		if i < len(userAddrs)-1 {
			userAddrsParam += ","
		}
	}

	sqlHead := "SELECT * FROM ("
	sqlTail := ") as combined"
	var sqlMids []string

	// (Optimization Hint: Could reuse item grouping logic function)
	// Build IN clause one by one is inefficient if list is huge, but acceptable for page size.
	// Note: Here loop iterates itemInfos directly?
	// Wait, code logic below:
	//   Iterate `itemInfos` OUTSIDE, but inside calls `multi.ItemTableName(info.ChainName)`.
	//   This loop seems to assume `itemInfos` are already grouped OR it will generate many single-item queries if mixed chains?
	//   Actually the loop: `for _, info := range itemInfos` generates ONE subquery PER ITEM.
	//   This matches logic: UNION ALL of many single-item SELECTs (or grouped by chain if optimized, but here is per item).
	//   Wait, original code loop:
	//     tmpStat := ... (builds ALL pairs)
	//     loop itemInfos:
	//       build sqlMid for EACH item??
	//       No, Look at line 864 in original: `for _, info := range itemInfos`
	//       Inside it formats table name `multi.ItemTableName(info.ChainName)`.
	//       If itemInfos has 20 items from same chain, it generates 20 subqueries?
	//       Yes, looks like it. This is inefficient compared to previous function `QueryMultiChainUserItemsListInfo`.
	//       Annotating functionality as is.

	// 修正逻辑说明:
	// 下面的 tmpStat 构建了所有 items 的 ID 列表.
	// 但循环又是针对 itemInfos 的. 逻辑似乎试图生成 N 个 SQL 块 union.
	// 这是一个潜在的性能点 (N次 Select Union).
	tmpStat := fmt.Sprintf("(('%s','%s')", itemInfos[0].CollectionAddress, itemInfos[0].TokenID)
	for i := 1; i < len(itemInfos); i++ {
		tmpStat += fmt.Sprintf(",('%s','%s')", itemInfos[i].CollectionAddress, itemInfos[i].TokenID)
	}
	tmpStat += ") "

	for _, info := range itemInfos {
		sqlMid := "("
		sqlMid += "select ci.id as id, ci.chain_id as chain_id,"
		sqlMid += "ci.collection_address as collection_address,ci.token_id as token_id, " +
			"ci.name as name, ci.owner as owner,"
		sqlMid += "min(co.price) as list_price, " +
			"SUBSTRING_INDEX(GROUP_CONCAT(co.marketplace_id ORDER BY co.price,co.marketplace_id),',', 1) " +
			"AS market_id, min(co.price) != 0 as listing "

		sqlMid += fmt.Sprintf("from %s as ci ", multi.ItemTableName(info.ChainName))
		sqlMid += fmt.Sprintf("join %s co ", multi.OrderTableName(info.ChainName))
		sqlMid += "on co.collection_address=ci.collection_address and co.token_id=ci.token_id "

		// Where: IN set (ALL items from the list, even if chain mismatch?
		// If info.ChainName is Eth, but tmpStat contains Polygon items, it won't match anyway. Use carefully.)
		sqlMid += "where (co.collection_address,co.token_id) in "
		sqlMid += tmpStat

		// Status: Active OR Expired
		sqlMid += fmt.Sprintf("and co.order_type = %d and (co.order_status=%d or co.order_status=%d) "+
			"and co.maker = ci.owner and co.maker in (%s) ",
			multi.ListingOrder, multi.OrderStatusActive, multi.OrderStatusExpired, userAddrsParam)
		sqlMid += "group by co.collection_address,co.token_id"
		sqlMid += ")"

		sqlMids = append(sqlMids, sqlMid)
	}

	// EXECUTE Queries
	sql := sqlHead
	for i := 0; i < len(sqlMids); i++ {
		if i != 0 {
			sql += " UNION ALL "
		}
		sql += sqlMids[i]
	}
	sql += sqlTail

	if err := d.DB.WithContext(ctx).Raw(sql).Scan(&collectionItems).Error; err != nil {
		return nil, errors.Wrap(err, "failed on query user multi chain items list info")
	}

	return collectionItems, nil
}

// QueryItemListInfo 查询单个 NFT Item 的挂单详情 (Listing Detail)
// 功能: Item 详情页使用, 获取当前最低挂单价和对应的订单详情
// 逻辑:
// 1. 查询基础信息和 Min(Price)
// 2. 如果存在 Listing, 再查询详细的 OrderID, ExpireTime 等
func (d *Dao) QueryItemListInfo(ctx context.Context, chain, collectionAddr, tokenID string) (*CollectionItem, error) {
	var collectionItem CollectionItem
	db := d.DB.WithContext(ctx).Table(fmt.Sprintf("%s as ci", multi.ItemTableName(chain)))
	coTableName := multi.OrderTableName(chain)

	// 1. Base Query with Min Price
	err := db.Select(
		"ci.id as id, ci.chain_id as chain_id, "+
			"ci.collection_address as collection_address,ci.token_id as token_id, "+
			"ci.name as name, ci.owner as owner, "+
			"min(co.price) as list_price, "+
			"SUBSTRING_INDEX(GROUP_CONCAT(co.marketplace_id ORDER BY co.price,co.marketplace_id),',', 1) AS market_id, "+
			"min(co.price) != 0 as listing").
		Joins(fmt.Sprintf("join %s co on co.collection_address=ci.collection_address and co.token_id=ci.token_id",
			coTableName)).
		Where("ci.collection_address =? and ci.token_id = ? and co.order_type = ? and co.order_status=? "+
			"and co.maker = ci.owner",
			collectionAddr, tokenID, multi.ListingOrder, multi.OrderStatusActive).
		Group("ci.collection_address,ci.token_id").
		Scan(&collectionItem).Error

	if err != nil {
		return nil, errors.Wrap(err, "failed on query user items list info")
	}

	// 如果没有 Active Listing, 提前返回
	if !collectionItem.Listing {
		return &collectionItem, nil
	}

	// 2. Detail Query: 获取具体那个 MinPrice 订单的详情 (OrderID, Salt, etc)
	var listOrder multi.Order
	if err := d.DB.WithContext(ctx).Table(fmt.Sprintf("%s as ci", multi.OrderTableName(chain))).
		Select("order_id, expire_time, maker, salt, event_time").
		Where("collection_address=? and token_id=? and maker=? and order_status=? and price = ?",
			collectionItem.CollectionAddress, collectionItem.TokenId,
			collectionItem.Owner, multi.OrderStatusActive, collectionItem.ListPrice). // Match exact price
		Scan(&listOrder).Error; err != nil {
		return nil, errors.Wrap(err, "failed on query item order id")
	}

	collectionItem.OrderID = listOrder.OrderID
	collectionItem.ListExpireTime = listOrder.ExpireTime
	collectionItem.ListMaker = listOrder.Maker
	collectionItem.ListSalt = listOrder.Salt
	collectionItem.ListTime = listOrder.EventTime

	return &collectionItem, nil
}

// QueryListingInfo 批量查询指定价格的订单详情
// 功能: 已知 (Collection, Token, Maker, Price), 这里的 priceInfos 应该是前端传来的 key,
// 用于反查具体的 OrderID.
func (d *Dao) QueryListingInfo(ctx context.Context, chain string,
	priceInfos []types.ItemPriceInfo) ([]multi.Order, error) {
	// 1. 构建 IN Tuple 条件
	var conditions []clause.Expr
	for _, price := range priceInfos {
		conditions = append(conditions,
			gorm.Expr("(?, ?, ?, ?, ?)",
				price.CollectionAddress,
				price.TokenID,
				price.Maker,
				price.OrderStatus,
				price.Price))
	}

	var orders []multi.Order
	// 2. Select matching orders
	if err := d.DB.WithContext(ctx).
		Table(multi.OrderTableName(chain)).
		Select("collection_address,token_id,order_id,event_time,"+
			"expire_time,salt,maker ").
		Where("(collection_address,token_id,maker,order_status,price) in (?)",
			conditions).
		Scan(&orders).Error; err != nil {
		return nil, errors.Wrap(err, "failed on query items order id")
	}

	return orders, nil
}

type MultiChainItemPriceInfo struct {
	types.ItemPriceInfo
	ChainName string
}

// QueryMultiChainListingInfo 跨链批量查询订单详情
// 功能: 同 QueryListingInfo, 但支持多链
func (d *Dao) QueryMultiChainListingInfo(ctx context.Context, priceInfos []MultiChainItemPriceInfo) ([]multi.Order, error) {
	var orders []multi.Order

	// 1. Group by Chain
	chainItemPrices := make(map[string][]MultiChainItemPriceInfo)
	for _, priceInfo := range priceInfos {
		items, ok := chainItemPrices[strings.ToLower(priceInfo.ChainName)]
		if ok {
			items = append(items, priceInfo)
			chainItemPrices[strings.ToLower(priceInfo.ChainName)] = items
		} else {
			chainItemPrices[strings.ToLower(priceInfo.ChainName)] = []MultiChainItemPriceInfo{priceInfo}
		}
	}

	sqlHead := "SELECT * FROM ("
	sqlTail := ") as combined"
	var sqlMids []string

	// 2. Build SubQueries
	for chainName, priceInfos := range chainItemPrices {
		// IN clause conditions
		tmpStat := fmt.Sprintf("(('%s','%s','%s',%d, %s)", priceInfos[0].CollectionAddress, priceInfos[0].TokenID, priceInfos[0].Maker, priceInfos[0].OrderStatus, priceInfos[0].Price.String())
		for i := 1; i < len(priceInfos); i++ {
			tmpStat += fmt.Sprintf(",('%s','%s','%s',%d, %s)", priceInfos[i].CollectionAddress, priceInfos[i].TokenID, priceInfos[i].Maker, priceInfos[i].OrderStatus, priceInfos[i].Price.String())
		}
		tmpStat += ") "

		sqlMid := "("
		sqlMid += "select collection_address,token_id,order_id,salt,event_time,expire_time,maker "
		sqlMid += fmt.Sprintf("from %s ", multi.OrderTableName(chainName))
		sqlMid += "where (collection_address,token_id,maker,order_status,price) in "
		sqlMid += tmpStat
		sqlMid += ")"

		sqlMids = append(sqlMids, sqlMid)
	}

	// 3. Union All
	sql := sqlHead
	for i := 0; i < len(sqlMids); i++ {
		if i != 0 {
			sql += " UNION ALL "
		}
		sql += sqlMids[i]
	}
	sql += sqlTail

	if err := d.DB.WithContext(ctx).Raw(sql).Scan(&orders).Error; err != nil {
		return nil, errors.Wrap(err, "failed on query user multi chain order list info")
	}

	return orders, nil
}

// QueryItemListingAcrossPlatforms 查询单个 Item 在各平台的最低挂单
// 功能: 聚合展示 (OpenSea, LooksRare, X2Y2 等) 的不同报价
// Group By Marketplace
func (d *Dao) QueryItemListingAcrossPlatforms(ctx context.Context, chain, collectionAddr, tokenID string, user []string) ([]types.ListingInfo, error) {
	var listings []types.ListingInfo
	if err := d.DB.WithContext(ctx).Table(multi.OrderTableName(chain)).
		Select("marketplace_id, min(price) as price").
		Where("collection_address=? and token_id=? and maker in (?) and order_type=? and order_status = ?",
						collectionAddr,
						tokenID,
						user,
						multi.ListingOrder,       // Listing
						multi.OrderStatusActive). // Active
		Group("marketplace_id"). // 分平台统计最低价
		Scan(&listings).Error; err != nil {
		return nil, errors.Wrap(err, "failed on query listing from db")
	}

	return listings, nil
}

// QueryItemInfo 查询单个 NFT Item 的基础元数据
func (d *Dao) QueryItemInfo(ctx context.Context, chain, collectionAddr, tokenID string) (*multi.Item, error) {
	var item multi.Item

	err := d.DB.WithContext(ctx).
		Table(fmt.Sprintf("%s as ci", multi.ItemTableName(chain))).
		Select("ci.id as id, "+
			"ci.chain_id as chain_id, "+
			"ci.collection_address as collection_address, "+
			"ci.token_id as token_id, "+
			"ci.name as name, "+
			"ci.owner as owner").
		Where("ci.collection_address =? and ci.token_id = ? ",
			collectionAddr, tokenID).
		Scan(&item).Error

	if err != nil {
		return nil, errors.Wrap(err, "failed on query user items list info")
	}

	return &item, nil
}

// QueryTraitsPrice 查询 NFT Trait 的价格信息 (Floor Price per Trait)
// 功能: 分析 Traits 的稀有度溢价
// 逻辑:
// 1. 找到该 Token 拥有的 Traits
// 2. 查询拥有相同 Trait 的所有 Active Orders
// 3. 取 MIN(Price) 作为该 Trait 的地板价
func (d *Dao) QueryTraitsPrice(ctx context.Context, chain, collectionAddr string, tokenIds []string) ([]types.TraitPrice, error) {
	var traitsPrice []types.TraitPrice

	// 1. SubQuery: Filter Orders with valid Traits
	listSubQuery := d.DB.WithContext(ctx).
		Table(fmt.Sprintf("%s as gf_order", multi.OrderTableName(chain))).
		Select("gf_attribute.trait,gf_attribute.trait_value,min(gf_order.price) as price"). // Min Price
		Where("gf_order.collection_address=? and gf_order.order_type=? and gf_order.order_status = ?",
			collectionAddr,
			multi.ListingOrder,
			multi.OrderStatusActive).
		// Filter 2: Only traits that appear in requested tokenIds
		Where("(gf_attribute.trait,gf_attribute.trait_value) in (?)",
			d.DB.WithContext(ctx).
				Table(fmt.Sprintf("%s as gf_attr", multi.ItemTraitTableName(chain))).
				Select("gf_attr.trait, gf_attr.trait_value").
				Where("gf_attr.collection_address=? and gf_attr.token_id in (?)",
					collectionAddr, tokenIds))

	// 2. JOIN Orders -> ItemTraits (gf_attribute)
	if err := listSubQuery.
		Joins(fmt.Sprintf("join %s as gf_attribute on gf_order.collection_address = gf_attribute.collection_address "+
			"and gf_order.token_id=gf_attribute.token_id", multi.ItemTraitTableName(chain))).
		Group("gf_attribute.trait, gf_attribute.trait_value").
		Scan(&traitsPrice).Error; err != nil {
		return nil, errors.Wrap(err, "failed on query trait price")
	}

	return traitsPrice, nil
}

func (d *Dao) UpdateItemOwner(ctx context.Context, chain string, collectionAddr, tokenID string, owner string) error {
	if err := d.DB.WithContext(ctx).Table(fmt.Sprintf("%s as ci", multi.ItemTableName(chain))).
		Where("collection_address = ? and token_id = ?", collectionAddr, tokenID).Update("owner", owner).
		Error; err != nil {
		return errors.Wrap(err, "failed on get user item count")
	}
	return nil
}

// QueryItemBids 查询单 Item 的出价列表 (Item Bids + Collection Bids)
// 功能: Items 详情页下的 "Offers" 表格
// 逻辑: UNION 两种类型的 Bids
//  1. Collection Bids: 针对整个集合的出价 (OrderType = CollectionBid)
//  2. Item Bids: 针对特定 Token 的出价 (OrderType = ItemBid)
func (d *Dao) QueryItemBids(ctx context.Context, chain string, collectionAddr, tokenID string,
	page, pageSize int) ([]types.ItemBid, int64, error) {

	db := d.DB.WithContext(ctx).Table(multi.OrderTableName(chain)).
		Select("marketplace_id, collection_address, token_id, order_id, salt, "+
			"event_time, expire_time, price, maker as bidder, order_type, "+
			"quantity_remaining as bid_unfilled, size as bid_size").

		// Condition 1: Collection Level Bids
		Where("collection_address = ? and order_type = ? and order_status = ? "+
			"and expire_time > ? and quantity_remaining > 0",
			collectionAddr, multi.CollectionBidOrder, multi.OrderStatusActive, time.Now().Unix()).

		// Condition 2: Item Level Bids (OR)
		Or("collection_address = ? and token_id=? and order_type = ? and order_status = ? "+
			"and expire_time > ? and quantity_remaining > 0",
			collectionAddr, tokenID, multi.ItemBidOrder, multi.OrderStatusActive, time.Now().Unix())

	// Count Total
	var count int64
	countTx := db.Session(&gorm.Session{})
	if err := countTx.Count(&count).Error; err != nil {
		return nil, 0, errors.Wrap(db.Error, "failed on count user items")
	}

	var itemBids []types.ItemBid
	if count == 0 {
		return itemBids, count, nil
	}

	// Pagination
	if err := db.Order("price desc"). // Best Offer First
						Offset(int((page - 1) * pageSize)).
						Limit(int(pageSize)).
						Scan(&itemBids).Error; err != nil {
		return nil, 0, errors.Wrap(err, "failed on get user items")
	}

	return itemBids, count, nil
}
