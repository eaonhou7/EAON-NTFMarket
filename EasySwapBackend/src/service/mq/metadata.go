package mq

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/ProjectsTask/EasySwapBase/logger/xzap"
	"github.com/ProjectsTask/EasySwapBase/stores/xkv"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/ProjectsTask/EasySwapBackend/src/types/v1"
)

const CacheRefreshSingleItemMetadataKey = "cache:%s:%s:item:refresh:metadata"

func GetRefreshSingleItemMetadataKey(project, chain string) string {
	return fmt.Sprintf(CacheRefreshSingleItemMetadataKey, strings.ToLower(project), strings.ToLower(chain))
}

const CacheRefreshPreventReentrancyKeyPrefix = "cache:es:item:refresh:prevent:reentrancy:%d:%s:%s"
const PreventReentrancyPeriod = 10 //second

// AddSingleItemToRefreshMetadataQueue 添加单个 Item 到元数据刷新队列
// 功能:
// 1. 检查防重入锁(Reentrancy Lock),防止短时间内重复刷新同一 Item
// 2. 将刷新请求(RefreshItem 结构体)序列化为 JSON
// 3. 将请求推送到 Redis Set 队列中 (SAdd)
// 4. 设置防重入锁过期时间 (10秒)
func AddSingleItemToRefreshMetadataQueue(kvStore *xkv.Store, project, chainName string, chainID int64, collectionAddr, tokenID string) error {
	isRefreshed, err := kvStore.Get(fmt.Sprintf(CacheRefreshPreventReentrancyKeyPrefix, chainID, collectionAddr, tokenID))
	if err != nil {
		return errors.Wrap(err, "failed on check reentrancy status")
	}

	if isRefreshed != "" {
		xzap.WithContext(context.Background()).Info("refresh within 10s", zap.String("collection_addr", collectionAddr), zap.String("token_id", tokenID))
		return nil
	}

	item := types.RefreshItem{
		ChainID:        chainID,
		CollectionAddr: collectionAddr,
		TokenID:        tokenID,
	}

	rawInfo, err := json.Marshal(&item)
	if err != nil {
		return errors.Wrap(err, "failed on marshal item info")
	}

	_, err = kvStore.Sadd(GetRefreshSingleItemMetadataKey(project, chainName), string(rawInfo))
	if err != nil {
		return errors.Wrap(err, "failed on push item to refresh metadata queue")
	}

	_ = kvStore.Setex(fmt.Sprintf(CacheRefreshPreventReentrancyKeyPrefix, chainID, collectionAddr, tokenID), "true", PreventReentrancyPeriod)

	return nil
}
