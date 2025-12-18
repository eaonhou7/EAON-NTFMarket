package v1

import (
	"sort"
	"strconv"
	"sync"

	"github.com/ProjectsTask/EasySwapBase/errcode"
	"github.com/ProjectsTask/EasySwapBase/logger/xzap"
	"github.com/ProjectsTask/EasySwapBase/xhttp"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/ProjectsTask/EasySwapBackend/src/service/svc"
	"github.com/ProjectsTask/EasySwapBackend/src/service/v1"
	"github.com/ProjectsTask/EasySwapBackend/src/types/v1"
)

// TopRankingHandler 处理获取 NFT 集合排行榜请求
// 主要功能:
// 1. 根据时间范围 (range: 15m, 1h, 1d 等) 统计集合交易量
// 2. 返回按交易量排序的前 N 个 (limit) 热门集合
// 3. 支持跨链数据聚合
// TopRankingHandler 处理获取 NFT 集合排行榜请求
// 主要功能:
// 1. 根据时间范围 (range: 15m, 1h, 1d 等) 统计集合交易量
// 2. 返回按交易量排序的前 N 个 (limit) 热门集合
// 3. 支持跨链数据聚合
func TopRankingHandler(svcCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 解析 limit 参数,获取需要返回的记录数量
		limit, err := strconv.ParseInt(c.Query("limit"), 10, 64)
		if err != nil {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		// 2. 获取并校验时间范围参数 (range)
		period := c.Query("range")
		if period != "" {
			// 定位支持的时间范围
			validParams := map[string]bool{
				"15m": true, // 15分钟
				"1h":  true, // 1小时
				"6h":  true, // 6小时
				"1d":  true, // 1天
				"7d":  true, // 7天
				"30d": true, // 30天
			}
			// 如果参数不在支持列表中,记录日志并返回错误
			if ok := validParams[period]; !ok {
				xzap.WithContext(c).Error("range parse error: ", zap.String("range", period))
				xhttp.Error(c, errcode.ErrInvalidParams)
				return
			}
		} else {
			// 默认使用 1 天的时间范围
			period = "1d"
		}

		// 3. 跨链并发查询
		// allResult 用于存储所有链的排名聚合结果
		var allResult []*types.CollectionRankingInfo

		// 使用 WaitGroup 等待所有 Goroutine 完成
		// 使用 Mutex 保护并发写入 allResult
		var wg sync.WaitGroup
		var mu sync.Mutex

		// 遍历所有支持的链
		for _, chain := range svcCtx.C.ChainSupported {
			wg.Add(1)
			go func(chain string) {
				defer wg.Done()

				// 获取该链的排名数据
				result, err := service.GetTopRanking(c.Copy(), svcCtx, chain, period, limit)
				if err != nil {
					// 仅返回错误即可,这里不应该 return, 而是记录错误
					// 实际上如果这里 return, 则只会中断当前 goroutine
					xhttp.Error(c, err)
					return
				}

				// 将结果安全地追加到总结果切片中
				mu.Lock()
				allResult = append(allResult, result...)
				mu.Unlock()
			}(chain.Name)
		}

		// 等待所有查询任务完成
		wg.Wait()

		// 4. 对聚合后的结果进行全量排序
		// 根据交易量 (Volume) 降序排列
		sort.SliceStable(allResult, func(i, j int) bool {
			return allResult[i].Volume.GreaterThan(allResult[j].Volume)
		})

		// 5. 返回 JSON 结果
		xhttp.OkJson(c, types.CollectionRankingResp{Result: allResult})
	}
}
