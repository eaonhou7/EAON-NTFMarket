package v1

import (
	"encoding/json"

	"github.com/ProjectsTask/EasySwapBase/errcode"
	"github.com/ProjectsTask/EasySwapBase/xhttp"
	"github.com/gin-gonic/gin"

	"github.com/ProjectsTask/EasySwapBackend/src/service/svc"
	"github.com/ProjectsTask/EasySwapBackend/src/service/v1"
	"github.com/ProjectsTask/EasySwapBackend/src/types/v1"
)

// OrderInfosHandler 处理订单信息聚合查询请求
// 主要功能:
// 1. 根据过滤条件 (filters) 查询 NFT 的最佳出价信息
// 2. 支持按链 ID、合约地址、Token ID、用户地址过滤
// 3. 聚合查询结果，混合 Item Bid 和 Collection Bid
// OrderInfosHandler 处理订单信息聚合查询请求
// 主要功能:
// 1. 根据过滤条件 (filters) 查询 NFT 的最佳出价信息
// 2. 支持按链 ID、合约地址、Token ID、用户地址过滤
// 3. 聚合查询结果，混合 Item Bid 和 Collection Bid
func OrderInfosHandler(svcCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 获取过滤参数
		filterParam := c.Query("filters")
		if filterParam == "" {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}

		// 2. 解析 JSON 过滤参数
		var filter types.OrderInfosParam
		err := json.Unmarshal([]byte(filterParam), &filter)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}

		// 3. 根据 ChainID 获取链名称
		chain, ok := chainIDToChain[filter.ChainID]
		if !ok {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		// 4. 调用 Service 层获取聚合订单信息
		res, err := service.GetOrderInfos(c.Request.Context(), svcCtx, filter.ChainID, chain, filter.UserAddress, filter.CollectionAddress, filter.TokenIds)
		if err != nil {
			// 返回具体的错误信息
			xhttp.Error(c, errcode.NewCustomErr(err.Error()))
			return
		}
		// 5. 返回结果 (包装在 result 字段中)
		xhttp.OkJson(c, struct {
			Result interface{} `json:"result"`
		}{Result: res})
	}
}
