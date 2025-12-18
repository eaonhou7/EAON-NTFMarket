package ordermanager

import (
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/ProjectsTask/EasySwapBase/stores/gdb"

	"github.com/ProjectsTask/EasySwapBase/logger/xzap"
)

type CollectionListed struct {
	CollectionAddress string `json:"collection_address"`
	ListCount         int    `json:"list_count"`
}

func GenCollectionListedKey(chain, address string) string {
	return fmt.Sprintf("cache:es:%s:collection:listed:%s", strings.ToLower(chain), strings.ToLower(address))
}

// listCountProcess 函数负责处理和维护NFT集合的上架数量统计
// 主要功能包括:
// 1. 启动时统计所有集合的上架数量
// 2. 定时(每分钟)更新有变动的集合的上架数量
// 3. 实时接收集合状态变更通知并记录需要更新的集合
func (om *OrderManager) listCountProcess() {
	// 启动时重新统计所有的collection list数量
	collectionsListed, err := om.countCollectionListed([]string{})
	if err != nil {
		xzap.WithContext(om.Ctx).Error("failed on count collection listed",
			zap.Error(err))
	}

	// 将统计结果缓存到Redis
	if err := om.cacheCollectionListCount(collectionsListed); err != nil {
		xzap.WithContext(om.Ctx).Error("failed on cache collection listed count",
			zap.Error(err))
	}

	// 用于记录需要更新的集合地址
	collections := make(map[string]bool)

	// 创建定时器,每分钟执行一次更新
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	// 持续监听和处理
	for {
		select {
		case <-ticker.C: // 定时器触发
			// 将map中记录的集合地址转换为切片
			var cs []string
			for c := range collections {
				cs = append(cs, c)
			}
			// 如果有需要更新的集合
			if len(cs) > 0 {
				// 重新统计这些集合的上架数量
				collectionsListed, err := om.countCollectionListed(cs)
				if err != nil {
					xzap.WithContext(om.Ctx).Error("failed on count collection listed",
						zap.Error(err))
				}

				// 更新缓存
				if err := om.cacheCollectionListCount(collectionsListed); err != nil {
					xzap.WithContext(om.Ctx).Error("failed on cache collection listed count",
						zap.Error(err))
				}
				// 清空记录
				collections = make(map[string]bool)
			}
		case addr := <-om.collectionListedCh: // 接收到集合状态变更通知
			collections[strings.ToLower(addr)] = true // 记录需要更新的集合地址
		case <-om.Ctx.Done(): // 上下文取消时退出
			xzap.WithContext(om.Ctx).Info("collection list count process exit",
				zap.Error(err))
			return
		}
	}
}

// countCollectionListed 函数用于统计NFT集合的上架数量
// 参数 cs []string: 需要统计的集合地址列表,如果为空则统计所有集合
// 返回 []CollectionListed: 包含集合地址和对应上架数量的结构体切片
func (om *OrderManager) countCollectionListed(cs []string) ([]CollectionListed, error) {
	var collectionsListed []CollectionListed

	// 构建基础SQL查询语句
	// 统计每个集合中上架的NFT数量(去重后的token_id数量)
	// 查询条件:
	// - order_type=1 表示listing类型订单
	// - order_status=0 表示订单状态为active
	// - maker必须是token的owner
	// - 非OpenSea禁止的item
	query := fmt.Sprintf(`SELECT co.collection_address,count(distinct (co.token_id)) as list_count
FROM %s as ci
         join %s co on co.collection_address = ci.collection_address and co.token_id = ci.token_id
WHERE  co.order_type = 1
  and co.order_status = 0
  and co.maker = ci.owner
  and (ci.is_opensea_banned, co.marketplace_id) != (true, 1)
group by co.collection_address`, gdb.GetMultiProjectItemTableName(om.project, om.chain), gdb.GetMultiProjectOrderTableName(om.project, om.chain))

	// 如果指定了集合地址,则修改查询语句添加collection_address筛选条件
	if len(cs) > 0 {
		query = fmt.Sprintf(`SELECT co.collection_address,count(distinct (co.token_id)) as list_count
FROM %s as ci
         join %s co on co.collection_address = ci.collection_address and co.token_id = ci.token_id
WHERE  co.collection_address in (?) and co.order_type = 1
  and co.order_status = 0
  and co.maker = ci.owner
  and (ci.is_opensea_banned, co.marketplace_id) != (true, 1)
group by co.collection_address`, gdb.GetMultiProjectItemTableName(om.project, om.chain), gdb.GetMultiProjectOrderTableName(om.project, om.chain))
	}

	// 执行SQL查询
	// 如果指定了集合地址,使用cs作为参数执行查询
	// 否则执行不带参数的查询
	if len(cs) > 0 {
		if err := om.DB.Raw(query, cs).Scan(&collectionsListed).Error; err != nil {
			return nil, errors.Wrap(err, "failed on statistical market earn")
		}
	} else {
		if err := om.DB.Raw(query).Scan(&collectionsListed).Error; err != nil {
			return nil, errors.Wrap(err, "failed on statistical market earn")
		}
	}

	return collectionsListed, nil
}

func (om *OrderManager) cacheCollectionListCount(collectionsListed []CollectionListed) error {
	for _, c := range collectionsListed {
		if err := om.Xkv.SetInt(GenCollectionListedKey(om.chain, c.CollectionAddress), c.ListCount); err != nil {
			return errors.Wrap(err, "failed on set collection listed count")
		}
	}

	return nil
}
