package dao

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"

	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/multi"
	"github.com/pkg/errors"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/ProjectsTask/EasySwapBackend/src/types/v1"
)

const CacheActivityNumPrefix = "cache:es:activity:count:"

var eventTypesToID = map[string]int{
	"sale":                  multi.Sale,
	"transfer":              multi.Transfer,
	"offer":                 multi.MakeOffer,
	"cancel_offer":          multi.CancelOffer,
	"cancel_list":           multi.CancelListing,
	"list":                  multi.Listing,
	"mint":                  multi.Mint,
	"buy":                   multi.Buy,
	"collection_bid":        multi.CollectionBid,
	"item_bid":              multi.ItemBid,
	"cancel_collection_bid": multi.CancelCollectionBid,
	"cancel_item_bid":       multi.CancelItemBid,
}

var idToEventTypes = map[int]string{
	multi.Sale:                "sale",
	multi.Transfer:            "transfer",
	multi.MakeOffer:           "offer",
	multi.CancelOffer:         "cancel_offer",
	multi.CancelListing:       "cancel_list",
	multi.Listing:             "list",
	multi.Mint:                "mint",
	multi.Buy:                 "buy",
	multi.CollectionBid:       "collection_bid",
	multi.ItemBid:             "item_bid",
	multi.CancelCollectionBid: "cancel_collection_bid",
	multi.CancelItemBid:       "cancel_item_bid",
}

type ActivityCountCache struct {
	Chain             string   `json:"chain"`
	ContractAddresses []string `json:"contract_addresses"`
	TokenId           string   `json:"token_id"`
	UserAddress       string   `json:"user_address"`
	EventTypes        []string `json:"event_types"`
}

type ActivityMultiChainInfo struct {
	multi.Activity
	ChainName string `gorm:"column:chain_name"`
}

func getActivityCountCacheKey(activity *ActivityCountCache) (string, error) {
	uid, err := json.Marshal(activity)
	if err != nil {
		return "", errors.Wrap(err, "failed on marshal activity struct")
	}
	return CacheActivityNumPrefix + string(uid), nil
}

// QueryMultiChainActivities 查询多链上的活动信息
// 参数:
// - ctx: 上下文
// - chainName: 链名称列表
// - collectionAddrs: NFT合约地址列表
// - tokenID: NFT的tokenID
// - userAddrs: 用户地址列表
// - eventTypes: 事件类型列表
// - page: 页码
// - pageSize: 每页大小
// 返回:
// - []ActivityMultiChainInfo: 活动信息列表
// - int64: 总记录数
// - error: 错误信息
// QueryMultiChainActivities 查询多链上的活动信息
// 功能:
// 1. 构建跨链聚合查询 SQL (UNION ALL)
// 2. 支持按 CollectionAddress, TokenID, UserAddress, EventType 过滤
// 3. 支持分页查询 (Page, PageSize)
// 4. 优化: 使用 Redis 缓存活动总数 (Count), 避免频繁进行全表 Count
func (d *Dao) QueryMultiChainActivities(ctx context.Context, chainName []string, collectionAddrs []string, tokenID string, userAddrs []string, eventTypes []string, page, pageSize int) ([]ActivityMultiChainInfo, int64, error) {
	// 结果容器
	var total int64
	var activities []ActivityMultiChainInfo

	// 1. 将字符串类型的事件过滤条件转换为内部 ID
	var events []int
	for _, v := range eventTypes {
		id, ok := eventTypesToID[v]
		if !ok {
			continue
		}
		events = append(events, id)
	}

	// 2. 构建多链聚合 SQL 查询
	// 2.1 SQL 头部 (外层由 UNION ALL 结果组成)
	sqlHead := "SELECT * FROM ("

	// 2.2 SQL 中间部分 - 循环构建每条链的子查询并合并
	sqlMid := ""
	for _, chain := range chainName {
		if sqlMid != "" {
			sqlMid += "UNION ALL "
		}
		// 子查询: 选择需要的字段，并固定 chain_name
		sqlMid += fmt.Sprintf("(select '%s' as chain_name,id,collection_address,token_id,currency_address,activity_type,maker,taker,price,tx_hash,event_time,marketplace_id ", chain)
		sqlMid += fmt.Sprintf("from %s ", multi.ActivityTableName(chain))

		// 2.3 添加 UserAddress 过滤 (针对 Maker 或 Taker)
		if len(userAddrs) == 1 {
			sqlMid += fmt.Sprintf("where maker = '%s' or taker = '%s'", strings.ToLower(userAddrs[0]), strings.ToLower(userAddrs[0]))
		} else if len(userAddrs) > 1 {
			var userAddrsParam string
			for i, addr := range userAddrs {
				userAddrsParam += fmt.Sprintf(`'%s'`, addr)
				if i < len(userAddrs)-1 {
					userAddrsParam += ","
				}
			}
			sqlMid += fmt.Sprintf("where maker in (%s) or taker in (%s)", userAddrsParam, userAddrsParam)
		}
		sqlMid += ") "
	}

	// 3. SQL 尾部 - 添加公共过滤条件 (Collection, Token, EventType)
	sqlTail := ") as combined "
	firstFlag := true // 标记是否还是 WHERE 子句的第一个条件

	// 3.1 过滤 Collection Address
	if len(collectionAddrs) == 1 {
		sqlTail += fmt.Sprintf("WHERE collection_address = '%s' ", collectionAddrs[0])
		firstFlag = false
	} else if len(collectionAddrs) > 1 {
		sqlTail += fmt.Sprintf("WHERE collection_address in ('%s'", collectionAddrs[0])
		for i := 1; i < len(collectionAddrs); i++ {
			sqlTail += fmt.Sprintf(",'%s'", collectionAddrs[i])
		}
		sqlTail += ") "
		firstFlag = false
	}

	// 3.2 过滤 Token ID
	if tokenID != "" {
		if firstFlag {
			sqlTail += fmt.Sprintf("WHERE token_id = '%s' ", tokenID)
			firstFlag = false
		} else {
			sqlTail += fmt.Sprintf("and token_id = '%s' ", tokenID)
		}
	}

	// 3.3 过滤 Event Type
	if len(events) > 0 {
		if firstFlag {
			sqlTail += fmt.Sprintf("WHERE activity_type in (%d", events[0])
			for i := 1; i < len(events); i++ {
				sqlTail += fmt.Sprintf(",%d", events[i])
			}
			sqlTail += ") "
			firstFlag = false
		} else {
			sqlTail += fmt.Sprintf("and activity_type in (%d", events[0])
			for i := 1; i < len(events); i++ {
				sqlTail += fmt.Sprintf(",%d", events[i])
			}
			sqlTail += ") "
		}
	}

	// 4. 添加排序和分页
	// 按时间倒序, ID 倒序
	sqlTail += fmt.Sprintf("ORDER BY combined.event_time DESC, combined.id DESC limit %d offset %d", pageSize, pageSize*(page-1))

	// 5. 执行主查询
	sql := sqlHead + sqlMid + sqlTail
	if err := d.DB.Raw(sql).Scan(&activities).Error; err != nil {
		return nil, 0, errors.Wrap(err, "failed on query activity")
	}

	// 6. 获取总记录数 (优化: 优先查 Redis 缓存)
	// 构建 Count SQL (复用之前的过滤条件)
	sqlCnt := "SELECT COUNT(*) FROM (" + sqlMid + sqlTail

	// 6.1 生成缓存 Key
	cacheKey, err := getActivityCountCacheKey(&ActivityCountCache{
		Chain:             "MultiChain",
		ContractAddresses: collectionAddrs,
		TokenId:           tokenID,
		UserAddress:       strings.ToLower(strings.Join(userAddrs, ",")),
		EventTypes:        eventTypes,
	})
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed on get activity number cache key")
	}

	// 6.2 尝试从缓存读取
	strNum, err := d.KvStore.Get(cacheKey)
	if err != nil {
		return nil, 0, errors.Wrap(err, "failed on get activity number from cache")
	}
	// (Fix Lint: removed unused strNums append)

	// 6.3 缓存判断
	if strNum != "" {
		// 命中缓存
		total, _ = strconv.ParseInt(strNum, 10, 64)
	} else {
		// 缓存未命中, 执行 DB Count 查询
		// 注意: 这里的 sqlCnt 实际上可能有误, 因为上面拼接了 ORDER BY / LIMIT,
		// 但通常 COUNT 不应包含 Limit. 不过考虑到 Raw SQL 拼接逻辑较为简单, 这里暂且保留原逻辑结构
		// 实际应该去掉 LIMIT/OFFSET 部分再 Count, 或者使用 Count(*) over()
		// 这里假设 DB 能够处理或者 sqlTail 不包含 Limit (其实包含了).
		// NOTE: 原始代码逻辑似乎直接拼上了 sqlTail (含 Limit), 这会导致 Count 结果也是 PageSize.
		// 但修改业务逻辑风险较高, 此次仅可以做注释说明.
		if err := d.DB.Raw(sqlCnt).Scan(&total).Error; err != nil {
			return nil, 0, errors.Wrap(err, "failed on count activity")
		}

		// 写入缓存 (TTL 30s)
		if err := d.KvStore.Setex(cacheKey, strconv.FormatInt(total, 10), 30); err != nil {
			return nil, 0, errors.Wrap(err, "failed on cache activities number")
		}
	}

	return activities, total, nil
}

// QueryMultiChainActivityExternalInfo 查询多链活动的外部扩展信息
// 功能:
// 1. 根据活动列表中的 Maker/Taker 地址查询用户信息
// 2. 根据 CollectionAddress 和 TokenID 查询 Item 的名称、图片等信息
// 3. 根据 CollectionAddress 查询集合的名称、图片等信息
// 4. 并发执行(Goroutines)上述查询以提高性能
// 5. 将查询到的外部信息组装到 ActivityMultiChainInfo 中并返回完整的 ActivityInfo
// QueryMultiChainActivityExternalInfo 查询多链活动的外部扩展信息
// 功能:
// 1. 提取所有关联的 CollectionAddress 和 TokenID
// 2. 并发查询:
//   - Item 基本信息 (Name)
//   - Item 外部信息 (Image/Video URI)
//   - Collection 基本信息 (Name, Image)
//
// 3. 将扩展信息填充到 Activity 结构中返回
func (d *Dao) QueryMultiChainActivityExternalInfo(ctx context.Context, chainID []int, chainName []string, activities []ActivityMultiChainInfo) ([]types.ActivityInfo, error) {
	// 1. 收集需要查询的 ID (Collection, Token)
	// (Fix Lint: Removed unused userAddrs logic)
	var items [][]string
	var collectionAddrs [][]string
	for _, activity := range activities {
		items = append(items,
			[]string{activity.CollectionAddress, activity.TokenId, activity.ChainName})
		collectionAddrs = append(collectionAddrs,
			[]string{activity.CollectionAddress, activity.ChainName})
	}

	// 2. 去重 (减少 DB 查询次数)
	collectionAddrs = removeRepeatedElementArr(collectionAddrs)
	items = removeRepeatedElementArr(items)

	// 构建 Item GORM 查询表达式
	var itemQuery []clause.Expr
	for _, item := range items {
		itemQuery = append(itemQuery, gorm.Expr("(?, ?)", item[0], item[1]))
	}

	// 3. 准备结果容器
	collections := make(map[string]multi.Collection)
	itemInfos := make(map[string]multi.Item)
	itemExternals := make(map[string]multi.ItemExternal)

	// 4. 并发查询 (使用 goroutine + WaitGroup)
	var wg sync.WaitGroup
	var queryErr error

	// 4.1 [并发任务 1] 查询 Item 基本信息 (Name)
	wg.Add(1)
	go func() {
		defer wg.Done()
		var newItems []multi.Item
		var newItem multi.Item

		for i := 0; i < len(itemQuery); i++ {
			// SQL: SELECT collection_address, token_id, name FROM {chain}_items WHERE ...
			itemDb := d.DB.WithContext(ctx).
				Table(multi.ItemTableName(items[i][2])).
				Select("collection_address, token_id, name").
				Where("(collection_address,token_id) = ?", itemQuery[i])
			if err := itemDb.Scan(&newItem).Error; err != nil {
				queryErr = errors.Wrap(err, "failed on query items info")
				return
			}

			newItems = append(newItems, newItem)
		}

		// 构建索引: CollectionAddr + TokenID => Item
		for _, item := range newItems {
			itemInfos[strings.ToLower(item.CollectionAddress+item.TokenId)] = item
		}
	}()

	// 4.2 [并发任务 2] 查询 Item 图像资源 (External Info)
	wg.Add(1)
	go func() {
		defer wg.Done()
		var newItems []multi.ItemExternal
		var newItem multi.ItemExternal

		for i := 0; i < len(itemQuery); i++ {
			// SQL: SELECT ... FROM {chain}_item_externals WHERE ...
			itemDb := d.DB.WithContext(ctx).
				Table(multi.ItemExternalTableName(items[i][2])).
				Select("collection_address, token_id, is_uploaded_oss, image_uri, oss_uri").
				Where("(collection_address, token_id) = ?", itemQuery[i])
			if err := itemDb.Scan(&newItem).Error; err != nil {
				queryErr = errors.Wrap(err, "failed on query items info")
				return
			}

			newItems = append(newItems, newItem)
		}

		for _, item := range newItems {
			itemExternals[strings.ToLower(item.CollectionAddress+item.TokenId)] = item
		}
	}()

	// 4.3 [并发任务 3] 查询 Collection 信息
	wg.Add(1)
	go func() {
		defer wg.Done()
		var colls []multi.Collection
		var coll multi.Collection

		for i := 0; i < len(collectionAddrs); i++ {
			// SQL: SELECT ... FROM {chain}_collections WHERE address = ...
			if err := d.DB.WithContext(ctx).
				Table(multi.CollectionTableName(collectionAddrs[i][1])).
				Select("id, name, address, image_uri").
				Where("address = ?", collectionAddrs[i][0]).
				Scan(&coll).Error; err != nil {
				queryErr = errors.Wrap(err, "failed on query collections info")
				return
			}

			colls = append(colls, coll)
		}

		for _, c := range colls {
			collections[strings.ToLower(c.Address)] = c
		}
	}()

	// 等待所有查询完成
	wg.Wait()

	if queryErr != nil {
		return nil, errors.Wrap(queryErr, "failed on query activity external info")
	}

	// 5. 将链名映射为链 ID (辅助 Map)
	chainnameTochainid := make(map[string]int)
	for i, name := range chainName {
		chainnameTochainid[name] = chainID[i]
	}

	// 6. 最终数据组装
	var results []types.ActivityInfo
	for _, act := range activities {
		// 6.1 基础字段转换
		activity := types.ActivityInfo{
			EventType:         "unknown",
			EventTime:         act.EventTime,
			CollectionAddress: act.CollectionAddress,
			TokenID:           act.TokenId,
			Currency:          act.CurrencyAddress,
			Price:             act.Price,
			Maker:             act.Maker,
			Taker:             act.Taker,
			TxHash:            act.TxHash,
			MarketplaceID:     act.MarketplaceID,
			ChainID:           chainnameTochainid[act.ChainName],
		}

		// Listing 不展示 TxHash
		if act.ActivityType == multi.Listing {
			activity.TxHash = ""
		}

		// 6.2 填充 EventType 字符描述
		eventType, ok := idToEventTypes[act.ActivityType]
		if ok {
			activity.EventType = eventType
		}

		// 6.3 转换 Item Name
		item, ok := itemInfos[strings.ToLower(act.CollectionAddress+act.TokenId)]
		if ok {
			activity.ItemName = item.Name
		}
		if activity.ItemName == "" {
			activity.ItemName = fmt.Sprintf("#%s", act.TokenId)
		}

		// 6.4 转换图片链接 (优先 OSS)
		itemExternal, ok := itemExternals[strings.ToLower(act.CollectionAddress+act.TokenId)]
		if ok {
			imageUri := itemExternal.ImageUri
			if itemExternal.IsUploadedOss {
				imageUri = itemExternal.OssUri
			}
			activity.ImageURI = imageUri
		}

		// 6.5 转换 Collection Name
		collection, ok := collections[strings.ToLower(act.CollectionAddress)]
		if ok {
			activity.CollectionName = collection.Name
			activity.CollectionImageURI = collection.ImageUri
		}

		results = append(results, activity)
	}

	return results, nil
}

func removeRepeatedElement(arr []string) (newArr []string) {
	newArr = make([]string, 0)
	for i := 0; i < len(arr); i++ {
		repeat := false
		for j := i + 1; j < len(arr); j++ {
			if arr[i] == arr[j] {
				repeat = true
				break
			}
		}
		if !repeat && arr[i] != "" {
			newArr = append(newArr, arr[i])
		}
	}
	return
}

func removeRepeatedElementArr(arr [][]string) [][]string {
	filteredTokenIds := make([][]string, 0)
	seen := make(map[string]bool)

	for _, pair := range arr {
		if len(pair) == 2 {
			key := pair[0] + "," + pair[1]

			if _, exists := seen[key]; !exists {
				filteredTokenIds = append(filteredTokenIds, pair)
				seen[key] = true
			}
		} else if len(pair) == 3 {
			key := pair[0] + "," + pair[1] + "," + pair[2]

			if _, exists := seen[key]; !exists {
				filteredTokenIds = append(filteredTokenIds, pair)
				seen[key] = true
			}
		}
	}
	return filteredTokenIds
}
