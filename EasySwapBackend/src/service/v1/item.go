package service

import (
	"context"

	"github.com/pkg/errors"

	"github.com/ProjectsTask/EasySwapBackend/src/service/svc"
	"github.com/ProjectsTask/EasySwapBackend/src/types/v1"
)

// GetItemBidsInfo 获取 Item 的历史出价信息
// 功能: 分页查询针对该 Item 的所有 Bid 记录
func GetItemBidsInfo(ctx context.Context, svcCtx *svc.ServerCtx, chain string, collectionAddr, tokenID string, page, pageSize int) (*types.CollectionBidsResp, error) {
	bids, count, err := svcCtx.Dao.QueryItemBids(ctx, chain, collectionAddr, tokenID, page, pageSize)
	if err != nil {
		return nil, errors.Wrap(err, "failed on get item info")
	}

	for i := 0; i < len(bids); i++ {
		bids[i].OrderType = getBidType(bids[i].OrderType)
	}
	return &types.CollectionBidsResp{
		Result: bids,
		Count:  count,
	}, nil
}
