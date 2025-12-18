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

// ActivityMultiChainHandler 处理多链活动查询请求
// 主要功能:
// 1. 解析前端传递的过滤参数 (filters)
// 2. 支持按链 ID、合约地址、Token ID、用户地址、事件类型等多维度过滤
// 3. 调用 service 层接口查询跨链活动数据
func ActivityMultiChainHandler(svcCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 从请求查询参数中获取 'filters' 字段
		filterParam := c.Query("filters")
		// 校验参数是否为空
		if filterParam == "" {
			// 如果过滤器参数为空,则返回自定义错误提示
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}

		// 定义过滤器结构体用于接收解析后的 JSON 数据
		var filter types.ActivityMultiChainFilterParams
		// 将 JSON 字符串解析为 Go 结构体
		err := json.Unmarshal([]byte(filterParam), &filter)
		if err != nil {
			// 解析失败(如 JSON 格式不正确),返回错误
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}

		// 根据 ChainID 列表映射对应的链名称 (ChainName)
		// Service 层需要使用链名称来定位对应的数据库表
		var chainName []string
		for _, id := range filter.ChainID {
			chainName = append(chainName, chainIDToChain[id])
		}

		// 调用 Service 层方法查询多链活动数据
		// 传入参数包括: 上下文, 服务上下文, 链ID列表, 链名称列表, 集合地址, TokenID, 用户地址, 事件类型, 分页参数
		res, err := service.GetMultiChainActivities(
			c.Request.Context(),
			svcCtx,
			filter.ChainID,
			chainName,
			filter.CollectionAddresses,
			filter.TokenID,
			filter.UserAddresses,
			filter.EventTypes,
			filter.Page,
			filter.PageSize,
		)
		if err != nil {
			// 查询过程发生错误,返回标准错误响应
			xhttp.Error(c, errcode.NewCustomErr("Get multi-chain activities failed."))
			return
		}
		// 查询成功,返回 JSON 格式结果
		xhttp.OkJson(c, res)
	}

}
