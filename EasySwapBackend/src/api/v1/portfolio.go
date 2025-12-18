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

// UserMultiChainCollectionsHandler 查询用户在多链上拥有的 NFT 集合
// 功能:
// 1. 获取用户在所有支持链上持有的 Collection 列表
// 2. 统计每个 Collection 的持有数量、地板价等信息
// UserMultiChainCollectionsHandler 查询用户在多链上拥有的 NFT 集合
// 功能:
// 1. 获取用户在所有支持链上持有的 Collection 列表
// 2. 统计每个 Collection 的持有数量、地板价等信息
func UserMultiChainCollectionsHandler(svcCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 1. 获取请求中的过滤参数
		filterParam := c.Query("filters")
		if filterParam == "" {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}

		// 2. 解析 JSON 参数到结构体
		var filter types.UserCollectionsParams
		err := json.Unmarshal([]byte(filterParam), &filter)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}

		// 3. 构建多链查询参数
		// 遍历所有支持的链,收集其 ID 和 Name
		var chainNames []string
		var chainIDs []int
		for _, chain := range svcCtx.C.ChainSupported {
			chainIDs = append(chainIDs, chain.ChainID)
			chainNames = append(chainNames, chain.Name)
		}

		// 4. 调用 Service 层并发查询多链数据
		res, err := service.GetMultiChainUserCollections(c.Request.Context(), svcCtx, chainIDs, chainNames, filter.UserAddresses)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("query user multi chain collections err."))
			return
		}

		// 5. 返回结果
		xhttp.OkJson(c, res)
	}
}

// UserMultiChainItemsHandler 查询用户在多链上持有的 NFT Item 列表
// 功能:
// 1. 支持按链、合约地址分页查询用户持有的具体 NFT
// 2. 返回 Item 的名称、图片、Token ID 等详细信息
func UserMultiChainItemsHandler(svcCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		filterParam := c.Query("filters")
		if filterParam == "" {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}

		var filter types.PortfolioMultiChainItemFilterParams
		err := json.Unmarshal([]byte(filterParam), &filter)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}

		// if filter.ChainID is empty, show all chain info
		if len(filter.ChainID) == 0 {
			for _, chain := range svcCtx.C.ChainSupported {
				filter.ChainID = append(filter.ChainID, chain.ChainID)
			}
		}

		var chainNames []string
		for _, chainID := range filter.ChainID {
			chain, ok := chainIDToChain[chainID]
			if !ok {
				xhttp.Error(c, errcode.ErrInvalidParams)
				return
			}
			chainNames = append(chainNames, chain)
		}

		res, err := service.GetMultiChainUserItems(c.Request.Context(), svcCtx, filter.ChainID, chainNames, filter.UserAddresses, filter.CollectionAddresses, filter.Page, filter.PageSize)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("query user multi chain items err."))
			return
		}

		xhttp.OkJson(c, res)
	}
}

// UserMultiChainListingsHandler 查询用户的 Listing (我的卖单)
// 功能:
// 1. 获取用户当前正在上架出售的 NFT 列表
// 2. 包含挂单价格、过期时间等信息
func UserMultiChainListingsHandler(svcCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		filterParam := c.Query("filters")
		if filterParam == "" {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}

		var filter types.PortfolioMultiChainListingFilterParams
		err := json.Unmarshal([]byte(filterParam), &filter)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}

		// if filter.ChainID is empty, show all chain info
		if len(filter.ChainID) == 0 {
			for _, chain := range svcCtx.C.ChainSupported {
				filter.ChainID = append(filter.ChainID, chain.ChainID)
			}
		}

		var chainNames []string
		for _, chainID := range filter.ChainID {
			chain, ok := chainIDToChain[chainID]
			if !ok {
				xhttp.Error(c, errcode.ErrInvalidParams)
				return
			}
			chainNames = append(chainNames, chain)
		}

		res, err := service.GetMultiChainUserListings(c.Request.Context(), svcCtx, filter.ChainID, chainNames, filter.UserAddresses, filter.CollectionAddresses, filter.Page, filter.PageSize)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("query user multi chain items err."))
			return
		}

		xhttp.OkJson(c, res)
	}
}

// UserMultiChainBidsHandler 查询用户的 Bids (我的买单)
// 功能:
// 1. 获取用户做出的所有有效出价 (Item Offer & Collection Offer)
func UserMultiChainBidsHandler(svcCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		filterParam := c.Query("filters")
		if filterParam == "" {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}

		var filter types.PortfolioMultiChainBidFilterParams
		err := json.Unmarshal([]byte(filterParam), &filter)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}

		// if filter.ChainID is empty, show all chain info
		if len(filter.ChainID) == 0 {
			for _, chain := range svcCtx.C.ChainSupported {
				filter.ChainID = append(filter.ChainID, chain.ChainID)
			}
		}

		var chainNames []string
		for _, chainID := range filter.ChainID {
			chain, ok := chainIDToChain[chainID]
			if !ok {
				xhttp.Error(c, errcode.ErrInvalidParams)
				return
			}
			chainNames = append(chainNames, chain)
		}

		res, err := service.GetMultiChainUserBids(c.Request.Context(), svcCtx, filter.ChainID, chainNames, filter.UserAddresses, filter.CollectionAddresses, filter.Page, filter.PageSize)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("query user multi chain items err."))
			return
		}

		xhttp.OkJson(c, res)
	}
}
