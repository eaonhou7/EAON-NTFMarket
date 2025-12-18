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

func UserMultiChainCollectionsHandler(svcCtx *svc.ServerCtx) gin.HandlerFunc {
	return func(c *gin.Context) {
		filterParam := c.Query("filters")
		if filterParam == "" {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}

		var filter types.UserCollectionsParams
		err := json.Unmarshal([]byte(filterParam), &filter)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("Filter param is nil."))
			return
		}

		var chainNames []string
		var chainIDs []int
		for _, chain := range svcCtx.C.ChainSupported {
			chainIDs = append(chainIDs, chain.ChainID)
			chainNames = append(chainNames, chain.Name)
		}

		res, err := service.GetMultiChainUserCollections(c.Request.Context(), svcCtx, chainIDs, chainNames, filter.UserAddresses)
		if err != nil {
			xhttp.Error(c, errcode.NewCustomErr("query user multi chain collections err."))
			return
		}

		xhttp.OkJson(c, res)
	}
}

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
