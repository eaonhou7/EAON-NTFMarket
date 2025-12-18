package v1

import (
	"encoding/json"
	"strconv"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/ProjectsTask/EasySwapBase/errcode"
	"github.com/ProjectsTask/EasySwapBase/logger/xzap"
	"github.com/ProjectsTask/EasySwapBase/xhttp"

	"github.com/ProjectsTask/EasySwapBackend/src/service/svc"
	"github.com/ProjectsTask/EasySwapBackend/src/service/v1"
	"github.com/ProjectsTask/EasySwapBackend/src/types/v1"
)

// CollectionItemsHandler 查询集合内 Item 列表
// 功能: 分页查询指定 Collection 下的 NFT 列表，支持按状态(BuyNow, HasOffer)过滤
func CollectionItemsHandler(svcCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 获取并校验过滤参数
		filterParam := c.Query("filters")
		if filterParam == "" {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}

		// 2. 解析 JSON 过滤参数
		var filter types.CollectionItemFilterParams
		err := json.Unmarshal([]byte(filterParam), &filter)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}

		// 3. 获取路径参数中的集合地址
		collectionAddr := c.Params.ByName("address")
		if collectionAddr == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		// 4. 验证链 ID 是否合法, 并获取对应的链名称
		chain, ok := chainIDToChain[filter.ChainID]
		if !ok {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		// 5. 调用 Service 层获取 Item 列表
		res, err := service.GetItems(c.Request.Context(), svcCtx, chain, filter, collectionAddr)
		if err != nil {
			// 记录错误并返回通用错误信息
			xhttp.Error(c, errcode.ErrUnexpected)
			return
		}
		// 6. 返回成功结果
		xhttp.OkJson(c, res)
	}
}

// CollectionBidsHandler 查询集合 Bids 信息
// 功能: 分页查询针对该 Collection 的所有出价信息
func CollectionBidsHandler(svcCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 获取过滤参数
		filterParam := c.Query("filters")
		if filterParam == "" {
			// 参数为空则返回错误
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}

		// 2. 解析 JSON 过滤参数
		var filter types.CollectionBidFilterParams
		err := json.Unmarshal([]byte(filterParam), &filter)
		if err != nil {
			// 解析失败返回错误
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}

		// 3. 获取路径参数中的集合地址
		collectionAddr := c.Params.ByName("address")
		if collectionAddr == "" {
			// 地址为空返回参数无效错误
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		// 4. 验证链ID并获取链名称
		chain, ok := chainIDToChain[int(filter.ChainID)]
		if !ok {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		// 5. 调用 Service 层查询 Bids 信息
		res, err := service.GetBids(c.Request.Context(), svcCtx, chain, collectionAddr, filter.Page, filter.PageSize)
		if err != nil {
			// 查询失败返回通用错误
			xhttp.Error(c, errcode.ErrUnexpected)
			return
		}
		// 6. 返回成功结果
		xhttp.OkJson(c, res)
	}
}

// CollectionItemBidsHandler 查询单个 Item 的 Bids
// 功能: 查询指定 Token ID 的出价单 (Item Offer)
func CollectionItemBidsHandler(svcCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		filterParam := c.Query("filters")
		if filterParam == "" {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}

		var filter types.CollectionBidFilterParams
		err := json.Unmarshal([]byte(filterParam), &filter)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}

		collectionAddr := c.Params.ByName("address")
		if collectionAddr == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		tokenID := c.Params.ByName("token_id")
		if tokenID == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		chain, ok := chainIDToChain[int(filter.ChainID)]
		if !ok {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		res, err := service.GetItemBidsInfo(c.Request.Context(), svcCtx, chain, collectionAddr, tokenID, filter.Page, filter.PageSize)
		if err != nil {
			xhttp.Error(c, errcode.ErrUnexpected)
			return
		}
		xhttp.OkJson(c, res)
	}
}

// ItemDetailHandler 查询 NFT 详情
// 功能: 获取单个 NFT 的详细信息，包括属性、所有者等
func ItemDetailHandler(svcCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		collectionAddr := c.Params.ByName("address")
		if collectionAddr == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		tokenID := c.Params.ByName("token_id")
		if tokenID == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		chainID, err := strconv.ParseInt(c.Query("chain_id"), 10, 64)
		if err != nil {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		chain, ok := chainIDToChain[int(chainID)]
		if !ok {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		res, err := service.GetItem(c.Request.Context(), svcCtx, chain, int(chainID), collectionAddr, tokenID)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("get item error"))
			return

		}
		xhttp.OkJson(c, res)
	}
}

// ItemTopTraitPriceHandler 查询 Top Trait 价格
// 功能: 获取集合中特定稀有属性 (Trait) 对应的最高 floor price
func ItemTopTraitPriceHandler(svcCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		collectionAddr := c.Params.ByName("address")
		if collectionAddr == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		filterParam := c.Query("filters")
		if filterParam == "" {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}

		var filter types.TopTraitFilterParams
		err := json.Unmarshal([]byte(filterParam), &filter)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}

		chain, ok := chainIDToChain[filter.ChainID]
		if !ok {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		res, err := service.GetItemTopTraitPrice(c.Request.Context(), svcCtx, chain, collectionAddr, filter.TokenIds)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("get item error"))
			return
		}
		xhttp.OkJson(c, res)
	}
}

// HistorySalesHandler 查询历史销售记录
// 功能: 获取集合在指定时间段 (Duration) 内的销售价格历史，用于绘制 K 线或散点图
func HistorySalesHandler(svcCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		collectionAddr := c.Params.ByName("address")
		if collectionAddr == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		chainID, err := strconv.ParseInt(c.Query("chain_id"), 10, 64)
		if err != nil {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		chain, ok := chainIDToChain[int(chainID)]
		if !ok {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		duration := c.Query("duration")
		if duration != "" {
			validParams := map[string]bool{
				"24h": true,
				"7d":  true,
				"30d": true,
			}
			if ok := validParams[duration]; !ok {
				xzap.WithContext(c).Error("duration parse error: ", zap.String("duration", duration))
				xhttp.Error(c, errcode.ErrInvalidParams)
				return
			}
		} else {
			duration = "7d"
		}

		res, err := service.GetHistorySalesPrice(c.Request.Context(), svcCtx, chain, collectionAddr, duration)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("get history sales price error"))
			return
		}

		xhttp.OkJson(c, struct {
			Result interface{} `json:"result"`
		}{
			Result: res,
		})
	}
}

// ItemTraitsHandler 查询 Item 属性
// 功能: 获取单个 NFT 的所有 Trait 属性信息
func ItemTraitsHandler(svcCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		collectionAddr := c.Params.ByName("address")
		if collectionAddr == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		tokenID := c.Params.ByName("token_id")
		if tokenID == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		chainID, err := strconv.ParseInt(c.Query("chain_id"), 10, 64)
		if err != nil {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		chain, ok := chainIDToChain[int(chainID)]
		if !ok {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		itemTraits, err := service.GetItemTraits(c.Request.Context(), svcCtx, chain, collectionAddr, tokenID)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("get item traits error"))
			return
		}

		xhttp.OkJson(c, types.ItemTraitsResp{Result: itemTraits})
	}
}

// ItemOwnerHandler 查询 Item 所有者
// 功能: 获取指定 NFT 的当前 Owner
func ItemOwnerHandler(svcCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		collectionAddr := c.Params.ByName("address")
		if collectionAddr == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		tokenID := c.Params.ByName("token_id")
		if tokenID == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		chainID, err := strconv.ParseInt(c.Query("chain_id"), 10, 64)
		if err != nil {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		chain, ok := chainIDToChain[int(chainID)]
		if !ok {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		owner, err := service.GetItemOwner(c.Request.Context(), svcCtx, chainID, chain, collectionAddr, tokenID)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("get item owner error"))
			return
		}

		xhttp.OkJson(c, struct {
			Result interface{} `json:"result"`
		}{
			Result: owner,
		})
	}
}

// GetItemImageHandler 获取 Item 图片
// 功能: 返回 NFT 的图片 URL (支持 OSS 或原始 URI)
func GetItemImageHandler(svcCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		collectionAddr := c.Params.ByName("address")
		if collectionAddr == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		tokenID := c.Params.ByName("token_id")
		if tokenID == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		chainID, err := strconv.ParseInt(c.Query("chain_id"), 10, 64)
		if err != nil {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		chain, ok := chainIDToChain[int(chainID)]
		if !ok {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		result, err := service.GetItemImage(c.Request.Context(), svcCtx, chain, collectionAddr, tokenID)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("failed on get item image"))
			return
		}

		xhttp.OkJson(c, struct {
			Result interface{} `json:"result"`
		}{Result: result})
	}
}

// ItemMetadataRefreshHandler 刷新元数据
// 功能: 手动触发 NFT 元数据更新任务
func ItemMetadataRefreshHandler(svcCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		chainId, err := strconv.ParseInt(c.Query("chain_id"), 10, 32)
		if err != nil {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		chain, ok := chainIDToChain[int(chainId)]
		if !ok {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		collectionAddr := c.Params.ByName("address")
		if collectionAddr == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		tokenId := c.Params.ByName("token_id")
		if collectionAddr == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		err = service.RefreshItemMetadata(c.Request.Context(), svcCtx, chain, chainId, collectionAddr, tokenId)
		if err != nil {
			xhttp.Error(c, err)
			return
		}

		successStr := "Success to joined the refresh queue and waiting for refresh."
		xhttp.OkJson(c, types.CommonResp{Result: successStr})
	}
}

// CollectionDetailHandler 查询集合详情
// 功能: 获取集合的基本信息、FloorPrice、总销量等
func CollectionDetailHandler(svcCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		chainID, err := strconv.ParseInt(c.Query("chain_id"), 10, 32)
		if err != nil {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		chain, ok := chainIDToChain[int(chainID)]
		if !ok {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}

		collectionAddr := c.Params.ByName("address")
		if collectionAddr == "" {
			xhttp.Error(c, errcode.ErrInvalidParams)
			return
		}
		res, err := service.GetCollectionDetail(c.Request.Context(), svcCtx, chain, collectionAddr)
		if err != nil {
			xhttp.Error(c, errcode.ErrUnexpected)
			return
		}

		xhttp.OkJson(c, res)
	}
}
