package dao

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ProjectsTask/EasySwapBase/ordermanager"
	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/multi"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"

	"github.com/ProjectsTask/EasySwapBackend/src/types/v1"
)

const MaxBatchReadCollections = 500
const MaxRetries = 3
const QueryTimeout = time.Second * 30

var collectionFields = []string{"id", "chain_id", "token_standard", "name", "address", "image_uri", "floor_price", "sale_price", "item_amount", "owner_amount"}

// QueryHistorySalesPriceInfo 查询指定时间段内的NFT销售历史价格信息
// 功能: 用于绘制价格走势图或计算近期均价
// SQL逻辑:
// SELECT price, token_id, event_time FROM {activity_table}
// WHERE activity_type = Order.Sale
// AND collection_address = ?
// AND event_time BETWEEN (now - duration) AND now
func (d *Dao) QueryHistorySalesPriceInfo(ctx context.Context, chain string, collectionAddr string, durationTimeStamp int64) ([]multi.Activity, error) {
	var historySalesInfo []multi.Activity
	now := time.Now().Unix()

	if err := d.DB.WithContext(ctx).
		Table(multi.ActivityTableName(chain)).
		Select("price", "token_id", "event_time").
		Where("activity_type = ? and collection_address = ? and event_time >= ? and event_time <= ?",
			multi.Sale,            // 筛选已成交(Sale)的记录
			collectionAddr,        // 指定集合
			now-durationTimeStamp, // 起始时间
			now).                  // 结束时间
		Find(&historySalesInfo).Error; err != nil {
		return nil, errors.Wrap(err, "failed on get history sales info")
	}

	return historySalesInfo, nil
}

// QueryAllCollectionInfo 查询指定链上的所有NFT集合信息 (全量扫描)
// 功能:
// 1. 使用游标分页 (Cursor Pagination) 方式遍历全表
// 2. 带有超时控制和重试机制 (MaxRetries = 3)
// 3. 使用事务保证读取一致性 (尽管对于长扫描一致性有限, 但避免了幻读导致的游标问题)
func (d *Dao) QueryAllCollectionInfo(ctx context.Context, chain string) ([]multi.Collection, error) {
	// 设置总超时时间
	ctx, cancel := context.WithTimeout(ctx, QueryTimeout)
	defer cancel()

	// 开启只读事务 (或普通事务)
	tx := d.DB.WithContext(ctx).Begin()
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			panic(r)
		}
	}()

	cursor := int64(0) // 游标: 上一次读取的最后一条 ID
	var allCollections []multi.Collection

	// 循环分页查询所有集合信息
	for {
		var collections []multi.Collection
		var err error

		// 重试机制: 最多重试 MaxRetries 次
		for i := 0; i < MaxRetries; i++ {
			// SQL: SELECT ... FROM collections WHERE id > cursor ORDER BY id ASC LIMIT 500
			err = tx.Table(multi.CollectionTableName(chain)).
				Select(collectionFields).
				Where("id > ?", cursor).
				Limit(MaxBatchReadCollections).
				Order("id asc").
				Scan(&collections).Error

			if err == nil {
				break // 成功则退出重试
			}

			// 如果是最后一次尝试仍失败, 则回滚并返回
			if i == MaxRetries-1 {
				tx.Rollback()
				return nil, errors.Wrap(err, "failed on get collections info")
			}
			// 指数退避或线性等待
			time.Sleep(time.Duration(i+1) * time.Second)
		}

		// 追加结果
		allCollections = append(allCollections, collections...)

		// 终止条件: 查询到的数量小于批大小, 说明已无更多数据
		if len(collections) < MaxBatchReadCollections {
			break
		}

		// 更新游标: 设置为当前批次最后一条记录的 ID
		cursor = collections[len(collections)-1].Id
	}

	// 提交事务 (这里主要是释放连接资源)
	if err := tx.Commit().Error; err != nil {
		return nil, errors.Wrap(err, "failed to commit transaction")
	}
	return allCollections, nil
}

// QueryCollectionInfo 查询指定链上的NFT集合信息
func (d *Dao) QueryCollectionInfo(ctx context.Context, chain string, collectionAddr string) (*multi.Collection, error) {
	var collection multi.Collection
	if err := d.DB.WithContext(ctx).Table(multi.CollectionTableName(chain)).
		Select(collectionDetailFields).Where("address = ?", collectionAddr).
		First(&collection).Error; err != nil {
		return nil, errors.Wrap(err, "failed on get collection info")
	}

	return &collection, nil
}

// QueryCollectionsInfo 批量查询指定链上的NFT集合信息
func (d *Dao) QueryCollectionsInfo(ctx context.Context, chain string, collectionAddrs []string) ([]multi.Collection, error) {
	addrs := removeRepeatedElement(collectionAddrs)
	var collections []multi.Collection
	if err := d.DB.WithContext(ctx).Table(multi.CollectionTableName(chain)).
		Select(collectionDetailFields).Where("address in (?)", addrs).
		Scan(&collections).Error; err != nil {
		return nil, errors.Wrap(err, "failed on get collection info")
	}

	return collections, nil
}

// QueryMultiChainCollectionsInfo 批量查询多条链上的NFT集合信息
// 参数collectionAddrs是一个二维数组,每个元素包含[合约地址,链名称]
// 返回多条链上的NFT集合信息列表
func (d *Dao) QueryMultiChainCollectionsInfo(ctx context.Context, collectionAddrs [][]string) ([]multi.Collection, error) {
	addrs := removeRepeatedElementArr(collectionAddrs)
	var collections []multi.Collection
	var collection multi.Collection
	for _, collectionAddr := range addrs {
		if err := d.DB.WithContext(ctx).Table(multi.CollectionTableName(collectionAddr[1])).
			Select(collectionDetailFields).Where("address = ?", collectionAddr[0]).
			Scan(&collection).Error; err != nil {
			return nil, errors.Wrap(err, "failed on get collection info")
		}
		collections = append(collections, collection)
	}

	return collections, nil
}

// QueryMultiChainUserCollectionInfos 查询用户在多条链上的Collection信息
// 功能:
// 1. 跨链查询用户持有的 Collection 列表
// 2. 聚合统计用户在每个 Collection 中持有的 Item 数量
// 3. 计算: floor_price * item_count 用于排序 (预估持仓价值)
func (d *Dao) QueryMultiChainUserCollectionInfos(ctx context.Context, chainID []int,
	chainNames []string, userAddrs []string) ([]types.UserCollections, error) {
	var userCollections []types.UserCollections

	// 1. 构建用户地址参数字符串, 格式: 'addr1','addr2',...
	var userAddrsParam string
	for i, addr := range userAddrs {
		userAddrsParam += fmt.Sprintf(`'%s'`, addr)
		if i < len(userAddrs)-1 {
			userAddrsParam += ","
		}
	}

	// 2. SQL 头部
	sqlHead := "SELECT * FROM ("

	// 3. SQL 尾部: 排序逻辑
	// 按照 [地板价 * 持有数量] 降序排序，即优先展示高价值持仓集合
	sqlTail := ") as combined ORDER BY combined.floor_price * " +
		"CAST(combined.item_count AS DECIMAL) DESC"
	var sqlMids []string

	// 4. 遍历每条链, 构建 UNION 子查询
	for _, chainName := range chainNames {
		sqlMid := "("
		// 4.1 联表查询: Collections (gc) JOIN Items (gi)
		// 目的: 筛选出用户(Owner)持有的 Item 对应的 Collection
		sqlMid += "select " +
			"gc.address as address, " +
			"gc.name as name, " +
			"gc.floor_price as floor_price, " +
			"gc.chain_id as chain_id, " +
			"gc.item_amount as item_amount, " +
			"gc.symbol as symbol, " +
			"gc.image_uri as image_uri, " +
			"count(*) as item_count " // 统计该用户在此 Collection 下持有的 Token 数量

		sqlMid += fmt.Sprintf("from %s as gc ", multi.CollectionTableName(chainName))
		sqlMid += fmt.Sprintf("join %s as gi ", multi.ItemTableName(chainName))
		sqlMid += "on gc.address = gi.collection_address "

		// 4.2 过滤条件: Item Owner 属于目标用户列表
		sqlMid += fmt.Sprintf("where gi.owner in (%s) ", userAddrsParam)

		// 4.3 分组: 按 Collection Address 分组并统计数量
		sqlMid += "group by gc.address"
		sqlMid += ")"

		sqlMids = append(sqlMids, sqlMid)
	}

	// 5. 组装完整 SQL: 使用 UNION ALL 合并多链结果
	sql := sqlHead
	for i := 0; i < len(sqlMids); i++ {
		if i != 0 {
			sql += " UNION ALL "
		}
		sql += sqlMids[i]
	}
	sql += sqlTail

	// 6. 执行查询
	if err := d.DB.WithContext(ctx).Raw(sql).Scan(&userCollections).Error; err != nil {
		return nil, errors.Wrap(err, "failed on get user multi chain collection infos")
	}

	return userCollections, nil
}

// QueryMultiChainUserItemInfos 查询用户拥有的 NFT Item 详细信息
// 功能:
// 1. 跨链查询所有用户持有的 Item
// 2. 关联查询每个 Item 的 "Last Sale Event" (最后一次成交时间)
// 3. 支持分页 (Global Pagination across chains)
// 4. 返回 Item 基本信息 + 最近成交时间 (Used for sorting)
func (d *Dao) QueryMultiChainUserItemInfos(ctx context.Context, chain []string, userAddrs []string,
	contractAddrs []string, page, pageSize int) ([]types.PortfolioItemInfo, int64, error) {
	var count int64
	var items []types.PortfolioItemInfo

	// 1. 构建用户地址参数
	var userAddrsParam string
	for i, addr := range userAddrs {
		userAddrsParam += fmt.Sprintf(`'%s'`, addr)
		if i < len(userAddrs)-1 {
			userAddrsParam += ","
		}
	}

	// 2. SQL 结构准备
	// Count查询用于分页总数计算
	sqlCntHead := "SELECT COUNT(*) FROM ("
	// 主查询用于获取数据
	sqlHead := "SELECT * FROM ("
	// 尾部: 按「持有时间/最后成交时间」倒序排列并分页
	sqlTail := fmt.Sprintf(") as combined ORDER BY combined.owned_time DESC LIMIT %d OFFSET %d",
		pageSize, page-1)
	var sqlMids []string

	// 3. 遍历每条链构建子查询
	for _, chainName := range chain {
		sqlMid := "("
		// 3.1 主查询字段: ChainID, Collection, TokenID, Name, Owner, OwnedTime(LastEventTime)
		sqlMid += "select gi.chain_id as chain_id, " +
			"gi.collection_address as collection_address, " +
			"gi.token_id as token_id, " +
			"gi.name as name, " +
			"gi.owner as owner, " +
			"sub.last_event_time as owned_time " // 将最后一次Sale时间作为持有时间参考(近似)
		sqlMid += fmt.Sprintf("from %s gi ", multi.ItemTableName(chainName))

		// 3.2 左连接子查询 (SubQuery): 获取每个 Item 的最后一次 Sale 时间
		sqlMid += "left join "
		sqlMid += "(select sgi.collection_address, sgi.token_id, " +
			"max(sga.event_time) as last_event_time " // 取最大时间
		sqlMid += fmt.Sprintf("from %s sgi join %s sga ",
			multi.ItemTableName(chainName), multi.ActivityTableName(chainName))
		sqlMid += "on sgi.collection_address = sga.collection_address " +
			"and sgi.token_id = sga.token_id "

		// 3.3 子查询过滤: 仅查询目标用户的 Item 且 EventType=Sale
		sqlMid += fmt.Sprintf("where sgi.owner in (%s) and sga.activity_type = %d ",
			userAddrsParam, multi.Sale)

		// 可选过滤: 合约地址
		if len(contractAddrs) > 0 {
			sqlMid += fmt.Sprintf("and sgi.collection_address in ('%s'", contractAddrs[0])
			for i := 1; i < len(contractAddrs); i++ {
				sqlMid += fmt.Sprintf(",'%s'", contractAddrs[i])
			}
			sqlMid += ") "
		}
		// 子查询分组
		sqlMid += "group by sgi.collection_address, sgi.token_id) sub "

		// 3.4 联结条件
		sqlMid += "on gi.collection_address = sub.collection_address " +
			"and gi.token_id = sub.token_id "

		// 3.5 主表过滤条件 (Items 表)
		sqlMid += fmt.Sprintf("where gi.owner in (%s) ", userAddrsParam)
		if len(contractAddrs) > 0 {
			sqlMid += fmt.Sprintf("and gi.collection_address in ('%s'", contractAddrs[0])
			for i := 1; i < len(contractAddrs); i++ {
				sqlMid += fmt.Sprintf(",'%s'", contractAddrs[i])
			}
			sqlMid += ")"
		}
		sqlMid += ")"

		sqlMids = append(sqlMids, sqlMid)
	}

	// 4. 合并 SQL (UNION ALL)
	sqlCnt := sqlCntHead
	sql := sqlHead
	for i := 0; i < len(sqlMids); i++ {
		if i != 0 {
			sql += " UNION ALL "
			sqlCnt += " UNION ALL "
		}
		sql += sqlMids[i]
		sqlCnt += sqlMids[i]
	}
	sql += sqlTail
	sqlCnt += ") as combined"

	// 5. 执行查询
	// 5.1 总数查询
	if err := d.DB.WithContext(ctx).Raw(sqlCnt).Scan(&count).Error; err != nil {
		return nil, 0, errors.Wrap(err, "failed on count user multi chain items")
	}
	// 5.2 数据列表查询
	if err := d.DB.WithContext(ctx).Raw(sql).Scan(&items).Error; err != nil {
		return nil, 0, errors.Wrap(err, "failed on get user multi chain items")
	}

	return items, count, nil
}

// QueryMultiChainUserListingItemInfos 查询多链上用户挂单Item信息
func (d *Dao) QueryMultiChainUserListingItemInfos(ctx context.Context, chain []string, userAddrs []string,
	contractAddrs []string, page, pageSize int) ([]types.PortfolioItemInfo, int64, error) {
	var count int64
	var items []types.PortfolioItemInfo

	// 构建用户地址参数字符串
	var userAddrsParam string
	for i, addr := range userAddrs {
		userAddrsParam += fmt.Sprintf(`'%s'`, addr)
		if i < len(userAddrs)-1 {
			userAddrsParam += ","
		}
	}

	// SQL语句头部
	sqlCntHead := "SELECT COUNT(*) FROM ("
	sqlHead := "SELECT * FROM ("
	// 分页SQL
	sqlTail := fmt.Sprintf(") as combined ORDER BY combined.owned_time DESC LIMIT %d OFFSET %d",
		pageSize, page-1)
	var sqlMids []string

	// 遍历每条链构建SQL
	for _, chainName := range chain {
		sqlMid := "("
		// 查询Item基本信息和最后交易时间
		sqlMid += "select gi.chain_id as chain_id, gi.collection_address as collection_address, " +
			"gi.token_id as token_id, gi.name as name, gi.owner as owner, " +
			"sub.last_event_time as owned_time "
		sqlMid += fmt.Sprintf("from %s gi ", multi.ItemTableName(chainName))
		sqlMid += "left join "
		// 子查询获取每个Item最后的交易时间
		sqlMid += "(select sgi.collection_address, sgi.token_id, " +
			"max(sga.event_time) as last_event_time "
		sqlMid += fmt.Sprintf("from %s sgi join %s sga ",
			multi.ItemTableName(chainName), multi.ActivityTableName(chainName))
		sqlMid += "on sgi.collection_address = sga.collection_address " +
			"and sgi.token_id = sga.token_id "
		// 过滤条件:指定用户和Sale类型活动
		sqlMid += fmt.Sprintf("where sgi.owner in (%s) and sga.activity_type = %d ",
			userAddrsParam, multi.Sale)

		// 添加合约地址过滤
		if len(contractAddrs) > 0 {
			sqlMid += fmt.Sprintf("and sgi.collection_address in ('%s'", contractAddrs[0])
			for i := 1; i < len(contractAddrs); i++ {
				sqlMid += fmt.Sprintf(",'%s'", contractAddrs[i])
			}
			sqlMid += ") "
		}
		sqlMid += "group by sgi.collection_address, sgi.token_id) sub "
		sqlMid += "on gi.collection_address = sub.collection_address " +
			"and gi.token_id = sub.token_id "

		// 主查询过滤条件
		sqlMid += fmt.Sprintf("where gi.owner in (%s) ", userAddrsParam)
		if len(contractAddrs) > 0 {
			sqlMid += fmt.Sprintf("and gi.collection_address in ('%s'", contractAddrs[0])
			for i := 1; i < len(contractAddrs); i++ {
				sqlMid += fmt.Sprintf(",'%s'", contractAddrs[i])
			}
			sqlMid += ")"
		}
		sqlMid += ")"

		sqlMids = append(sqlMids, sqlMid)
	}

	// 使用UNION ALL合并多链结果
	sqlCnt := sqlCntHead
	sql := sqlHead
	for i := 0; i < len(sqlMids); i++ {
		if i != 0 {
			sql += " UNION ALL "
			sqlCnt += " UNION ALL "
		}
		sql += sqlMids[i]
		sqlCnt += sqlMids[i]
	}
	sql += sqlTail
	sqlCnt += ") as combined"

	// 执行SQL查询
	if err := d.DB.WithContext(ctx).Raw(sqlCnt).Scan(&count).Error; err != nil {
		return nil, 0, errors.Wrap(err, "failed on count user multi chain items")
	}
	if err := d.DB.WithContext(ctx).Raw(sql).Scan(&items).Error; err != nil {
		return nil, 0, errors.Wrap(err, "failed on get user multi chain items")
	}

	return items, count, nil
}

// QueryCollectionsListed 查询多个集合的上架数量
func (d *Dao) QueryCollectionsListed(ctx context.Context, chain string, collectionAddrs []string) ([]types.CollectionListed, error) {
	var collectionsListed []types.CollectionListed
	if len(collectionAddrs) == 0 {
		return collectionsListed, nil
	}

	for _, address := range collectionAddrs {
		count, err := d.KvStore.GetInt(ordermanager.GenCollectionListedKey(chain, address))
		if err != nil {
			return nil, errors.Wrap(err, "failed on set collection listed count")
		}
		collectionsListed = append(collectionsListed, types.CollectionListed{
			CollectionAddr: address,
			Count:          count,
		})
	}

	return collectionsListed, nil
}

// CacheCollectionsListed 缓存集合的上架数量
func (d *Dao) CacheCollectionsListed(ctx context.Context, chain string, collectionAddr string, listedCount int) error {
	err := d.KvStore.SetInt(ordermanager.GenCollectionListedKey(chain, collectionAddr), listedCount)
	if err != nil {
		return errors.Wrap(err, "failed on set collection listed count")
	}

	return nil
}

// QueryFloorPrice 查询指定 Collection 的实时地板价
// 功能:
// 1. 获取当前市场上该集合最低的挂单价格 (Active Sales)
// 2. 过滤条件:
//   - OrderType = Listing (卖单)
//   - OrderStatus = Active (有效)
//   - Maker = Item Owner (防止过期或虚假挂单)
//   - Marketplace != 1 (排除特定市场, 可选逻辑)
//
// 3. 按价格升序取 Limit 1
func (d *Dao) QueryFloorPrice(ctx context.Context, chain string, collectionAddr string) (decimal.Decimal, error) {
	var order multi.Order

	// SQL 逻辑详解:
	// INNER JOIN: 联结 Items 和 Orders 表
	// 确保 Order 关联的 Item 属于当前的 Owner (co.maker = ci.owner)
	sql := fmt.Sprintf(`SELECT co.price as price
		FROM %s as ci
				left join %s co on co.collection_address = ci.collection_address and co.token_id = ci.token_id
		WHERE (co.collection_address= ? and co.order_type = ? and
			co.order_status = ? and co.maker = ci.owner and co.marketplace_id != ?)
		order by co.price asc limit 1`, multi.ItemTableName(chain), multi.OrderTableName(chain))

	// 执行查询
	if err := d.DB.WithContext(ctx).Raw(
		sql,
		collectionAddr,
		OrderType,   // 1 (Listing)
		OrderStatus, // 0 (Active)
		1,           // Exclude Marketplace ID 1
	).Scan(&order).Error; err != nil {
		return decimal.Zero, errors.Wrap(err, "failed on get collection floor price")
	}

	return order.Price, nil
}

func GetCollectionTradeInfoKey(project, chain string, collectionAddr string) string {
	return fmt.Sprintf("cache:%s:%s:collection:%s:trade", strings.ToLower(project), strings.ToLower(chain), strings.ToLower(collectionAddr))
}

type CollectionVolume struct {
	Volume decimal.Decimal `json:"volume"`
}

func GetHoldersCountKey(chain string) string {
	return fmt.Sprintf("cache:es:%s:holders:count", chain)
}

// QueryCollectionFloorChange 查询集合地板价变化趋势 (24h/7d etc.)
// 功能: 计算涨跌幅 = (最新价格 - 历史价格) / 历史价格
// 参数:
//   - chain: 链名称
//   - timeDiff: 比较的时间间隔 (秒), e.g. 86400 (1天)
//
// 返回:
//   - map[string]float64: CollectionAddr -> ChangeRate (小数格式, e.g. 0.05 = 5%)
func (d *Dao) QueryCollectionFloorChange(chain string, timeDiff int64) (map[string]float64, error) {
	collectionFloorChange := make(map[string]float64)
	var collectionPrices []multi.CollectionFloorPrice

	// SQL 逻辑分析:
	// 目标: 获取每个 Collection 的「最新价格」和「指定的历史价格」
	// 实现方式: UNION 两个子查询结果，并按时间倒序排列
	// 1. 子查询 A: 获取全量集合的最新一条 FloorPrice 记录 (GROUP BY address MAX(event_time))
	// 2. 子查询 B: 获取全量集合在 T-timeDiff 之前的最新一条记录 (作为历史锚点)
	// 结果: 每个 Collection Address 会有 1~2 条记录. 排序后第一条是最新, 第二条是历史.
	rawSql := fmt.Sprintf(`SELECT collection_address, price, event_time 
		FROM %s 
		WHERE (collection_address, event_time) IN (
			SELECT collection_address, MAX(event_time)
			FROM %s
			GROUP BY collection_address
		) OR (collection_address, event_time) IN (
			SELECT collection_address, MAX(event_time)
			FROM %s 
			WHERE event_time <= UNIX_TIMESTAMP() - ? 
			GROUP BY collection_address
		) 
		ORDER BY collection_address,event_time DESC`,
		multi.CollectionFloorPriceTableName(chain),
		multi.CollectionFloorPriceTableName(chain),
		multi.CollectionFloorPriceTableName(chain))

	if err := d.DB.Raw(rawSql, timeDiff).Scan(&collectionPrices).Error; err != nil {
		return nil, errors.Wrap(err, "failed on get collection floor change")
	}

	// 遍历计算涨跌幅
	// 由于 SQL 已经按 Address, Time DESC 排序
	// 相同的 Address 的记录会相邻, 且最新在前, 历史在后
	for i := 0; i < len(collectionPrices); i++ {
		// 检查是否有配对的历史记录 (当前是最新, 下一条是同一集合的历史记录)
		if i < len(collectionPrices)-1 &&
			collectionPrices[i].CollectionAddress == collectionPrices[i+1].CollectionAddress &&
			collectionPrices[i+1].Price.GreaterThan(decimal.Zero) {

			// 计算公式: (Current - History) / History
			change := collectionPrices[i].Price.Sub(collectionPrices[i+1].Price).
				Div(collectionPrices[i+1].Price).
				InexactFloat64()

			collectionFloorChange[collectionPrices[i].CollectionAddress] = change

			// 跳过下一条(历史记录), 继续处理下一个集合
			i++
		} else {
			// 如果没有历史记录, 涨跌幅视为 0
			collectionFloorChange[collectionPrices[i].CollectionAddress] = 0.0
		}
	}

	return collectionFloorChange, nil
}

// QueryCollectionsSellPrice 查询批量集合的 Collection Offer 最高价 (Buy Order)
// 功能: 获取每个集合当前的最高“求购”价格 (Best Collection Bid)
func (d *Dao) QueryCollectionsSellPrice(ctx context.Context, chain string) ([]multi.Collection, error) {
	var collections []multi.Collection

	// SQL:
	// MAX(co.price): 获取最高出价
	// OrderType = CollectionBidOrder
	// OrderStatus = Active
	// ExpireTime > Now (未过期)
	sql := fmt.Sprintf(`SELECT collection_address as address, max(co.price) as sale_price
FROM %s as co where order_status = ? and order_type = ? and expire_time > ? group by collection_address`, multi.OrderTableName(chain))

	if err := d.DB.WithContext(ctx).Raw(
		sql,
		multi.OrderStatusActive,
		multi.CollectionBidOrder,
		time.Now().Unix()).Scan(&collections).Error; err != nil {
		return nil, errors.Wrap(err, "failed on get collection sell price")
	}

	return collections, nil
}

// QueryCollectionSellPrice 查询单个集合的 Collection Offer 最高价
func (d *Dao) QueryCollectionSellPrice(ctx context.Context, chain, collectionAddr string) (*multi.Collection, error) {
	var collection multi.Collection

	// 同样逻辑, 但限制了 Collection Address 并 Limit 1
	sql := fmt.Sprintf(`SELECT collection_address as address, co.price as sale_price
FROM %s as co where collection_address = ? and order_status = ? and order_type = ? and quantity_remaining > 0 and expire_time > ? order by price desc limit 1`, multi.OrderTableName(chain))

	if err := d.DB.WithContext(ctx).Raw(
		sql,
		collectionAddr,
		multi.OrderStatusActive,
		multi.CollectionBidOrder,
		time.Now().Unix()).Scan(&collection).Error; err != nil {
		return nil, errors.Wrap(err, "failed on get collection sell price")
	}

	return &collection, nil
}
