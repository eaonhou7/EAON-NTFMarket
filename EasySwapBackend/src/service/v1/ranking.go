package service

import (
	"context"
	"strconv"
	"strings"
	"sync"

	"github.com/ProjectsTask/EasySwapBase/errcode"
	"github.com/ProjectsTask/EasySwapBase/logger/xzap"
	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/multi"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/ProjectsTask/EasySwapBackend/src/dao"
	"github.com/ProjectsTask/EasySwapBackend/src/service/svc"
	"github.com/ProjectsTask/EasySwapBackend/src/types/v1"
)

const MinuteSeconds = 60
const HourSeconds = 60 * 60
const DaySeconds = 3600 * 24

// GetTopRanking 获取指定链上的 NFT 集合排名信息
// 功能:
// 1. 统计指定时间窗口 (period) 内的交易量、销量
// 2. 计算地板价涨跌幅 (Floor Change)
// 3. 关联查询集合基本信息 (FloorPrice, OwnerNum, ItemNum, etc.)
// 4. 支持按交易量降序返回 Top N 集合
//
// 参数:
// - chain: 链名称 (e.g. "eth", "polygon")
// - period: 时间范围 (15m, 1h, 6h, 1d, 7d, 30d)
// - limit: 返回数量限制
func GetTopRanking(ctx context.Context, svcCtx *svc.ServerCtx, chain string, period string, limit int64) ([]*types.CollectionRankingInfo, error) {
	// 获取集合交易信息
	tradeInfos, err := svcCtx.Dao.GetCollectionRankingByActivity(chain, period)
	if err != nil {
		xzap.WithContext(ctx).Error("failed on get collection trade info", zap.Error(err))
		//return nil, errcode.NewCustomErr("cache error")
	}

	// 构建交易信息map
	collectionTradeMap := make(map[string]dao.CollectionTrade)
	for _, tradeInfo := range tradeInfos {
		collectionTradeMap[strings.ToLower(tradeInfo.ContractAddress)] = *tradeInfo
	}

	// 时间范围映射表
	periodTime := map[string]int64{
		"15m": MinuteSeconds * 15,
		"1h":  HourSeconds,
		"6h":  HourSeconds * 6,
		"1d":  DaySeconds,
		"7d":  DaySeconds * 7,
		"30d": DaySeconds * 30,
	}
	// 获取地板价变化信息
	collectionFloorChange, err := svcCtx.Dao.QueryCollectionFloorChange(chain, periodTime[period])
	if err != nil {
		xzap.WithContext(ctx).Error("failed on get collection floor change", zap.Error(err))
	}

	var wg sync.WaitGroup
	var queryErr error

	// 并发获取集合销售价格信息
	collectionSells := make(map[string]multi.Collection)
	wg.Add(1)
	go func() {
		defer wg.Done()
		sellInfos, err := svcCtx.Dao.QueryCollectionsSellPrice(ctx, chain)
		if err != nil {
			xzap.WithContext(ctx).Error("failed on get all collections info", zap.Error(err))
			queryErr = errcode.NewCustomErr("failed on get all collections info")
			return
		}
		for _, sell := range sellInfos {
			collectionSells[strings.ToLower(sell.Address)] = sell
		}
	}()

	// 并发获取所有集合基本信息
	var allCollections []multi.Collection
	wg.Add(1)
	go func() {
		defer wg.Done()
		allCollections, err = svcCtx.Dao.QueryAllCollectionInfo(ctx, chain)
		if err != nil {
			xzap.WithContext(ctx).Error("failed on get all collections info", zap.Error(err))
			queryErr = errcode.NewCustomErr("failed on get all collections info")
			return
		}
	}()

	wg.Wait()

	if queryErr != nil {
		return nil, queryErr
	}

	// 构建返回结果
	var respInfos []*types.CollectionRankingInfo
	for _, collection := range allCollections {
		var priceChange float64
		var volume decimal.Decimal
		var sellPrice decimal.Decimal
		var sales int64

		// 获取交易相关信息
		tradeInfo, ok := collectionTradeMap[strings.ToLower(collection.Address)] // 统一小写
		if ok {
			priceChange = collectionFloorChange[strings.ToLower(collection.Address)]
			volume = tradeInfo.Volume
			sales = tradeInfo.ItemCount
		}
		// 获取销售价格信息
		sellInfo, ok := collectionSells[strings.ToLower(collection.Address)]
		if ok {
			sellPrice = sellInfo.SalePrice
		}

		// 获取上架数量
		var listAmount int
		listed, err := svcCtx.Dao.QueryCollectionsListed(ctx, chain, []string{collection.Address})
		if err != nil {
			xzap.WithContext(ctx).Error("failed on query collection listed", zap.Error(err))
		} else {
			listAmount = listed[0].Count
		}

		// 构建单个集合的排名信息
		respInfos = append(respInfos, &types.CollectionRankingInfo{
			Name:        collection.Name,
			Address:     collection.Address,
			ImageUri:    collection.ImageUri,
			FloorPrice:  collection.FloorPrice.String(),
			FloorChange: strconv.FormatFloat(priceChange, 'f', 4, 32),
			SellPrice:   sellPrice.String(),
			Volume:      volume,
			ItemSold:    sales,
			ItemNum:     collection.ItemAmount,
			ItemOwner:   collection.OwnerAmount,
			ListAmount:  listAmount,
			ChainID:     collection.ChainId,
		})
	}

	// 限制返回数量
	if limit < int64(len(respInfos)) {
		respInfos = respInfos[:limit]
	}

	return respInfos, nil
}
