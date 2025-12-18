package service

import (
	"context"

	"github.com/pkg/errors"

	"github.com/ProjectsTask/EasySwapBackend/src/service/svc"
	"github.com/ProjectsTask/EasySwapBackend/src/types/v1"
)

// GetMultiChainActivities 获取多链活动信息
// 功能:
// 1. 查询多链上的交易活动 (Transfer, Sale, List, etc.)
// 2. 支持按链名称、合约地址、TokenID、用户地址、事件类型进行过滤
// 3. 关联查询外部信息 (如 NFT 图片、名称等)
func GetMultiChainActivities(ctx context.Context, svcCtx *svc.ServerCtx, chainID []int, chainName []string, collectionAddrs []string, tokenID string, userAddrs []string, eventTypes []string, page, pageSize int) (*types.ActivityResp, error) {
	// 1. 查询基础活动列表 (DB 查询)
	// 根据传入的过滤条件(链、集合、Token、用户、事件类型)分查询数据库
	activities, total, err := svcCtx.Dao.QueryMultiChainActivities(ctx, chainName, collectionAddrs, tokenID, userAddrs, eventTypes, page, pageSize)
	if err != nil {
		return nil, errors.Wrap(err, "failed on query multi-chain activity")
	}

	// 2. 空结果处理
	// 如果没有查询到活动记录,直接返回空结果,避免后续无意义的查询
	if total == 0 || len(activities) == 0 {
		return &types.ActivityResp{
			Result: nil,
			Count:  0,
		}, nil
	}

	// 3. 关联查询外部信息 (External Info)
	// 根据活动记录中的 Token 信息,去查询对应的图片、名称等元数据
	// 这一步通常涉及跨表查询或调用其他服务
	results, err := svcCtx.Dao.QueryMultiChainActivityExternalInfo(ctx, chainID, chainName, activities)
	if err != nil {
		return nil, errors.Wrap(err, "failed on query activity external info")
	}

	// 4. 返回最终结果
	return &types.ActivityResp{
		Result: results,
		Count:  total,
	}, nil
}
