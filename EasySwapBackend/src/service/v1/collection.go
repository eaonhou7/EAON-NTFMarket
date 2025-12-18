package service

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/ProjectsTask/EasySwapBase/errcode"
	"github.com/ProjectsTask/EasySwapBase/evm/eip"
	"github.com/ProjectsTask/EasySwapBase/logger/xzap"
	"github.com/ProjectsTask/EasySwapBase/ordermanager"
	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/multi"
	"github.com/pkg/errors"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"

	"github.com/ProjectsTask/EasySwapBackend/src/dao"
	"github.com/ProjectsTask/EasySwapBackend/src/service/mq"
	"github.com/ProjectsTask/EasySwapBackend/src/service/svc"
	"github.com/ProjectsTask/EasySwapBackend/src/types/v1"
)

// GetBids 获取集合的 Bids 信息
// 功能: 分页查询针对指定 Collection 的所有有效出价
func GetBids(ctx context.Context, svcCtx *svc.ServerCtx, chain string, collectionAddr string, page, pageSize int) (*types.CollectionBidsResp, error) {
	bids, count, err := svcCtx.Dao.QueryCollectionBids(ctx, chain, collectionAddr, page, pageSize)
	if err != nil {
		return nil, errors.Wrap(err, "failed on get item info")
	}

	return &types.CollectionBidsResp{
		Result: bids,
		Count:  count,
	}, nil
}

// GetItems 获取 NFT Item 列表信息
// 功能:
// 1. 获取集合下的 items 列表的基本信息
// 2. 并发查询关联信息:
//   - 订单详情 (Listing info)
//   - 图片/视频资源 (Image/Video info)
//   - 用户持有数量 (User Balance)
//   - 最近成交价 (Last Sale Price)
//   - 最佳出价 (Best Bid)
//
// 3. 聚合 Collection 级别和 Item 级别的最佳 Bid，计算对卖家最优的出价
func GetItems(ctx context.Context, svcCtx *svc.ServerCtx, chain string, filter types.CollectionItemFilterParams, collectionAddr string) (*types.NFTListingInfoResp, error) {
	// 1. 查询基础Item信息和订单信息
	items, count, err := svcCtx.Dao.QueryCollectionItemOrder(ctx, chain, filter, collectionAddr)
	if err != nil {
		return nil, errors.Wrap(err, "failed on get item info")
	}

	// 2. 提取需要查询的ItemID和所有者地址
	var ItemIds []string
	var ItemOwners []string
	var itemPrice []types.ItemPriceInfo
	for _, item := range items {
		if item.TokenId != "" {
			ItemIds = append(ItemIds, item.TokenId)
		}
		if item.Owner != "" {
			ItemOwners = append(ItemOwners, item.Owner)
		}
		// 记录已上架Item的价格信息
		if item.Listing {
			itemPrice = append(itemPrice, types.ItemPriceInfo{
				CollectionAddress: item.CollectionAddress,
				TokenID:           item.TokenId,
				Maker:             item.Owner,
				Price:             item.ListPrice,
				OrderStatus:       multi.OrderStatusActive,
			})
		}
	}

	// 3. 并发查询各类扩展信息
	// 使用 WaitGroup 同步多个 goroutine 的执行结果
	var queryErr error
	var wg sync.WaitGroup

	// 3.1 [并发任务 1] 查询部分订单详情 (Listing Info)
	// 根据 Item 的价格信息查询对应的 Listing 订单详情(如过期时间、Salt等)
	ordersInfo := make(map[string]multi.Order)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if len(itemPrice) > 0 {
			// 调用 DAO 层批量查询 Listing 信息
			orders, err := svcCtx.Dao.QueryListingInfo(ctx, chain, itemPrice)
			if err != nil {
				queryErr = errors.Wrap(err, "failed on get orders time info")
				return
			}
			// 将查询结果 list 转为 map, 方便后续 O(1) 查找
			// Key: CollectionAddress + TokenId (转小写)
			for _, order := range orders {
				ordersInfo[strings.ToLower(order.CollectionAddress+order.TokenId)] = order
			}
		}
	}()

	// 3.2 [并发任务 2] 查询 Item 图片和视频资源 (External Info)
	ItemsExternal := make(map[string]multi.ItemExternal)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if len(ItemIds) != 0 {
			// 查询 Items 的外部资源链接
			items, err := svcCtx.Dao.QueryCollectionItemsImage(ctx, chain, collectionAddr, ItemIds)
			if err != nil {
				queryErr = errors.Wrap(err, "failed on get items image info")
				return
			}
			// 构建 map 索引
			for _, item := range items {
				ItemsExternal[strings.ToLower(item.TokenId)] = item
			}
		}
	}()

	// 3.3 [并发任务 3] 查询用户持有数量 (Owner Balance)
	userItemCount := make(map[string]int64)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if len(ItemIds) != 0 {
			// 查询每个 Owner 在该 Collection 下持有的 NFT 数量
			itemCount, err := svcCtx.Dao.QueryUsersItemCount(ctx, chain, collectionAddr, ItemOwners)
			if err != nil {
				queryErr = errors.Wrap(err, "failed on get items image info")
				return
			}
			for _, v := range itemCount {
				userItemCount[strings.ToLower(v.Owner)] = v.Counts
			}
		}
	}()

	// 3.4 [并发任务 4] 查询 Item 最近一次成交价格 (Last Sale Price)
	lastSales := make(map[string]decimal.Decimal)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if len(ItemIds) != 0 {
			lastSale, err := svcCtx.Dao.QueryLastSalePrice(ctx, chain, collectionAddr, ItemIds)
			if err != nil {
				queryErr = errors.Wrap(err, "failed on get items last sale info")
				return
			}
			for _, v := range lastSale {
				lastSales[strings.ToLower(v.TokenId)] = v.Price
			}
		}
	}()

	// 3.5 [并发任务 5] 查询 Item 级别的最佳出价 (Item Best Bid)
	bestBids := make(map[string]multi.Order)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if len(ItemIds) != 0 {
			// 查询针对特定 Item 的 Offer
			bids, err := svcCtx.Dao.QueryBestBids(ctx, chain, filter.UserAddress, collectionAddr, ItemIds)
			if err != nil {
				queryErr = errors.Wrap(err, "failed on get items last sale info")
				return
			}
			// 筛选每个 Item 的最高出价
			for _, bid := range bids {
				order, ok := bestBids[strings.ToLower(bid.TokenId)]
				if !ok {
					bestBids[strings.ToLower(bid.TokenId)] = bid
					continue
				}
				// 如果当前 Bid 价格更高，则更新
				if bid.Price.GreaterThan(order.Price) {
					bestBids[strings.ToLower(bid.TokenId)] = bid
				}
			}
		}
	}()

	// 3.6 [并发任务 6] 查询 Collection 级别的最佳出价 (Collection Best Bid)
	// 这是一个通用的 Offer, 适用于该 Collection 下的任意 NFT
	var collectionBestBid multi.Order
	wg.Add(1)
	go func() {
		defer wg.Done()
		collectionBestBid, err = svcCtx.Dao.QueryCollectionBestBid(ctx, chain, filter.UserAddress, collectionAddr)
		if err != nil {
			queryErr = errors.Wrap(err, "failed on get items last sale info")
			return
		}
	}()

	// 4. 等待所有查询完成
	wg.Wait()
	if queryErr != nil {
		return nil, errors.Wrap(queryErr, "failed on get items info")
	}

	// 5. 整合所有信息
	var respItems []*types.NFTListingInfo
	for _, item := range items {
		// 设置Item名称, 如果数据库中名称为空,则使用 "#TokenID" 格式
		nameStr := item.Name
		if nameStr == "" {
			nameStr = fmt.Sprintf("#%s", item.TokenId)
		}

		// 构建基础返回结构, 默认填充 Collection 级别的最佳出价信息
		respItem := &types.NFTListingInfo{
			Name:              nameStr,
			CollectionAddress: item.CollectionAddress,
			TokenID:           item.TokenId,
			OwnerAddress:      item.Owner,
			ListPrice:         item.ListPrice,
			MarketID:          item.MarketID,
			// 默认使用 Collection Best Bid
			BidOrderID:    collectionBestBid.OrderID,
			BidExpireTime: collectionBestBid.ExpireTime,
			BidPrice:      collectionBestBid.Price,
			BidTime:       collectionBestBid.EventTime,
			BidSalt:       collectionBestBid.Salt,
			BidMaker:      collectionBestBid.Maker,
			BidType:       getBidType(collectionBestBid.OrderType),
			BidSize:       collectionBestBid.Size,
			BidUnfilled:   collectionBestBid.QuantityRemaining,
		}

		// 填充挂单(Listing)信息
		listOrder, ok := ordersInfo[strings.ToLower(item.CollectionAddress+item.TokenId)]
		if ok {
			respItem.ListTime = listOrder.EventTime
			respItem.ListOrderID = listOrder.OrderID
			respItem.ListExpireTime = listOrder.ExpireTime
			respItem.ListSalt = listOrder.Salt
		}

		// 比较并填充最佳出价(Best Bid) 信息
		// 逻辑: 如果存在针对该 Item 的单独出价, 且价格高于 Collection 级别的出价, 则展示 Item 出价
		// 否则, 继续展示上面已经填充的 Collection 出价
		bidOrder, ok := bestBids[strings.ToLower(item.TokenId)]
		if ok {
			if bidOrder.Price.GreaterThan(collectionBestBid.Price) {
				respItem.BidOrderID = bidOrder.OrderID
				respItem.BidExpireTime = bidOrder.ExpireTime
				respItem.BidPrice = bidOrder.Price
				respItem.BidTime = bidOrder.EventTime
				respItem.BidSalt = bidOrder.Salt
				respItem.BidMaker = bidOrder.Maker
				respItem.BidType = getBidType(bidOrder.OrderType)
				respItem.BidSize = bidOrder.Size
				respItem.BidUnfilled = bidOrder.QuantityRemaining
			}
		}

		// 填充图片和视频信息
		itemExternal, ok := ItemsExternal[strings.ToLower(item.TokenId)]
		if ok {
			// 优先使用 OSS 链接, 其次使用原始链接
			if itemExternal.IsUploadedOss {
				respItem.ImageURI = itemExternal.OssUri
			} else {
				respItem.ImageURI = itemExternal.ImageUri
			}
			// 处理视频信息
			if len(itemExternal.VideoUri) > 0 {
				respItem.VideoType = itemExternal.VideoType
				if itemExternal.IsVideoUploaded {
					respItem.VideoURI = itemExternal.VideoOssUri
				} else {
					respItem.VideoURI = itemExternal.VideoUri
				}
			}
		}

		// 填充用户持有数量
		count, ok := userItemCount[strings.ToLower(item.Owner)]
		if ok {
			respItem.OwnerOwnedAmount = count
		}

		// 填充最近成交价格
		price, ok := lastSales[strings.ToLower(item.TokenId)]
		if ok {
			respItem.LastSellPrice = price
		}

		respItems = append(respItems, respItem)
	}

	return &types.NFTListingInfoResp{
		Result: respItems,
		Count:  count,
	}, nil
}

// GetItem 获取单个 NFT 的详细信息
// 功能:
// 1. 聚合查询 NFT 的全方位信息: 基本属性、所有者、图片、挂单、出价、历史成交、FloorPrice 等
// 2. 也是采用并发查询 (ErrGroup/WaitGroup) 模式提高响应速度
func GetItem(ctx context.Context, svcCtx *svc.ServerCtx, chain string, chainID int, collectionAddr, tokenID string) (*types.ItemDetailInfoResp, error) {
	var queryErr error
	var wg sync.WaitGroup

	// 并发查询以下信息:
	// 1. [并发任务 1] 查询 Collection 基本信息 (Name, FloorPrice 等)
	var collection *multi.Collection
	wg.Add(1)
	go func() {
		defer wg.Done()
		collection, queryErr = svcCtx.Dao.QueryCollectionInfo(ctx, chain, collectionAddr)
		if queryErr != nil {
			return
		}
	}()

	// 2. [并发任务 2] 查询 Item 基本信息 (Owner, Name, TokenID)
	var item *multi.Item
	wg.Add(1)
	go func() {
		defer wg.Done()
		item, queryErr = svcCtx.Dao.QueryItemInfo(ctx, chain, collectionAddr, tokenID)
		if queryErr != nil {
			return
		}
	}()

	// 3. [并发任务 3] 查询 Item 挂单信息 (Listing Info)
	// 查询该 Item 当前是否处于上架状态，以及挂单价格、过期时间等
	var itemListInfo *dao.CollectionItem
	wg.Add(1)
	go func() {
		defer wg.Done()
		itemListInfo, queryErr = svcCtx.Dao.QueryItemListInfo(ctx, chain, collectionAddr, tokenID)
		if queryErr != nil {
			return
		}
	}()

	// 4. [并发任务 4] 查询 Item 图片和视频资源 (External Info)
	ItemExternals := make(map[string]multi.ItemExternal)
	wg.Add(1)
	go func() {
		defer wg.Done()
		items, err := svcCtx.Dao.QueryCollectionItemsImage(ctx, chain, collectionAddr, []string{tokenID})
		if err != nil {
			queryErr = errors.Wrap(err, "failed on get items image info")
			return
		}

		for _, item := range items {
			ItemExternals[strings.ToLower(item.TokenId)] = item
		}
	}()

	// 5. [并发任务 5] 查询 Item 最近成交价格 (Last Sale Price)
	lastSales := make(map[string]decimal.Decimal)
	wg.Add(1)
	go func() {
		defer wg.Done()
		lastSale, err := svcCtx.Dao.QueryLastSalePrice(ctx, chain, collectionAddr, []string{tokenID})
		if err != nil {
			queryErr = errors.Wrap(err, "failed on get items last sale info")
			return
		}

		for _, v := range lastSale {
			lastSales[strings.ToLower(v.TokenId)] = v.Price
		}
	}()

	// 6. [并发任务 6] 查询 Item 级别的最高出价 (Item Best Bid)
	bestBids := make(map[string]multi.Order)
	wg.Add(1)
	go func() {
		defer wg.Done()
		bids, err := svcCtx.Dao.QueryBestBids(ctx, chain, "", collectionAddr, []string{tokenID})
		if err != nil {
			queryErr = errors.Wrap(err, "failed on get items last sale info")
			return
		}

		// 筛选出价格最高的 Bid
		for _, bid := range bids {
			order, ok := bestBids[strings.ToLower(bid.TokenId)]
			if !ok {
				bestBids[strings.ToLower(bid.TokenId)] = bid
				continue
			}
			if bid.Price.GreaterThan(order.Price) {
				bestBids[strings.ToLower(bid.TokenId)] = bid
			}
		}
	}()

	// 7. [并发任务 7] 查询 Collection 级别的最高出价 (Collection Best Bid)
	var collectionBestBid multi.Order
	wg.Add(1)
	go func() {
		defer wg.Done()
		bid, err := svcCtx.Dao.QueryCollectionBestBid(ctx, chain, "", collectionAddr)
		if err != nil {
			queryErr = errors.Wrap(err, "failed on get items last sale info")
			return
		}
		collectionBestBid = bid
	}()

	// 等待所有查询完成
	wg.Wait()
	if queryErr != nil {
		return nil, errors.Wrap(queryErr, "failed on get items info")
	}

	// 组装返回数据
	var itemDetail types.ItemDetailInfo
	itemDetail.ChainID = chainID

	// 1. 设置 Item 基本信息和默认的最佳出价 (Collection Bid)
	if item != nil {
		itemDetail.Name = item.Name
		itemDetail.CollectionAddress = item.CollectionAddress
		itemDetail.TokenID = item.TokenId
		itemDetail.OwnerAddress = item.Owner
		// 默认填充 Collection 级别的最高出价信息
		itemDetail.BidOrderID = collectionBestBid.OrderID
		itemDetail.BidExpireTime = collectionBestBid.ExpireTime
		itemDetail.BidPrice = collectionBestBid.Price
		itemDetail.BidTime = collectionBestBid.EventTime
		itemDetail.BidSalt = collectionBestBid.Salt
		itemDetail.BidMaker = collectionBestBid.Maker
		itemDetail.BidType = getBidType(collectionBestBid.OrderType)
		itemDetail.BidSize = collectionBestBid.Size
		itemDetail.BidUnfilled = collectionBestBid.QuantityRemaining
	}

	// 2. 比较并设置最佳出价 (Item Bid vs Collection Bid)
	// 如果 Item 级别的最高出价大于 Collection 级别的最高出价,则使用 Item 级别的出价信息
	bidOrder, ok := bestBids[strings.ToLower(item.TokenId)]
	if ok {
		if bidOrder.Price.GreaterThan(collectionBestBid.Price) {
			itemDetail.BidOrderID = bidOrder.OrderID
			itemDetail.BidExpireTime = bidOrder.ExpireTime
			itemDetail.BidPrice = bidOrder.Price
			itemDetail.BidTime = bidOrder.EventTime
			itemDetail.BidSalt = bidOrder.Salt
			itemDetail.BidMaker = bidOrder.Maker
			itemDetail.BidType = getBidType(bidOrder.OrderType)
			itemDetail.BidSize = bidOrder.Size
			itemDetail.BidUnfilled = bidOrder.QuantityRemaining
		}
	}

	// 3. 设置挂单(Listing)信息
	// 如果 Item 处于上架状态, 填充 Listing 详情
	if itemListInfo != nil {
		itemDetail.ListPrice = itemListInfo.ListPrice
		itemDetail.MarketplaceID = itemListInfo.MarketID
		itemDetail.ListOrderID = itemListInfo.OrderID
		itemDetail.ListTime = itemListInfo.ListTime
		itemDetail.ListExpireTime = itemListInfo.ListExpireTime
		itemDetail.ListSalt = itemListInfo.ListSalt
		itemDetail.ListMaker = itemListInfo.ListMaker
	}

	// 4. 设置 Collection 信息
	if collection != nil {
		itemDetail.CollectionName = collection.Name
		itemDetail.FloorPrice = collection.FloorPrice
		itemDetail.CollectionImageURI = collection.ImageUri
		// 如果 Item 没有独立名字, 使用 Collection Name + #TokenID 组合
		if itemDetail.Name == "" {
			itemDetail.Name = fmt.Sprintf("%s #%s", collection.Name, tokenID)
		}
	}

	// 5. 设置最近成交价格
	price, ok := lastSales[strings.ToLower(tokenID)]
	if ok {
		itemDetail.LastSellPrice = price
	}

	// 6. 设置图片和视频信息
	itemExternal, ok := ItemExternals[strings.ToLower(tokenID)]
	if ok {
		// 优先使用 OSS 链接
		itemDetail.ImageURI = itemExternal.ImageUri
		if itemExternal.IsUploadedOss {
			itemDetail.ImageURI = itemExternal.OssUri
		}
		// 处理视频链接
		if len(itemExternal.VideoUri) > 0 {
			itemDetail.VideoType = itemExternal.VideoType
			if itemExternal.IsVideoUploaded {
				itemDetail.VideoURI = itemExternal.VideoOssUri
			} else {
				itemDetail.VideoURI = itemExternal.VideoUri
			}
		}
	}

	return &types.ItemDetailInfoResp{
		Result: itemDetail,
	}, nil
}

// GetItemTopTraitPrice 获取指定 token ids的Trait的最高价格信息
func GetItemTopTraitPrice(ctx context.Context, svcCtx *svc.ServerCtx, chain, collectionAddr string, tokenIDs []string) (*types.ItemTopTraitResp, error) {
	// 1. 查询 Trait 对应的最低挂单价格 (Floor Price per Trait)
	traitsPrice, err := svcCtx.Dao.QueryTraitsPrice(ctx, chain, collectionAddr, tokenIDs)
	if err != nil {
		return nil, errors.Wrap(err, "failed on calc top trait")
	}

	// 2. 空结果处理
	if len(traitsPrice) == 0 {
		return &types.ItemTopTraitResp{
			Result: []types.TraitPrice{},
		}, nil
	}

	// 3. 构建 Trait -> 最低挂单价格映射 (Lookup Map)
	// Key 格式: "TraitType:TraitValue" (e.g. "Color:Red")
	traitsPrices := make(map[string]decimal.Decimal)
	for _, traitPrice := range traitsPrice {
		traitsPrices[strings.ToLower(fmt.Sprintf("%s:%s", traitPrice.Trait, traitPrice.TraitValue))] = traitPrice.Price
	}

	// 4. 查询指定 token ids 的所有 Trait 属性
	traits, err := svcCtx.Dao.QueryItemsTraits(ctx, chain, collectionAddr, tokenIDs)
	if err != nil {
		return nil, errors.Wrap(err, "failed on query items trait")
	}

	// 5. 计算每个 Token 的"最高价值 Trait"
	// 逻辑: 一个 Token 可能有多个 Trait (e.g. Color:Red, Eyes:Blue)
	// 我们希望找到该 Token 所拥有的所有 Trait 中, 地板价最高的那个 Trait
	// 这通常代表了该 NFT 的"最稀有/最值钱"属性
	topTraits := make(map[string]types.TraitPrice)
	for _, trait := range traits {
		key := strings.ToLower(fmt.Sprintf("%s:%s", trait.Trait, trait.TraitValue))
		price, ok := traitsPrices[key]
		if ok {
			topPrice, ok := topTraits[trait.TokenId]
			// 如果已有记录, 比较当前 Trait 价格与已记录的 Trait 价格
			if ok {
				// 如果当前 Trait 价格不高于已记录的最高 Trait 价格, 则跳过
				if price.LessThanOrEqual(topPrice.Price) {
					continue
				}
			}

			// 更新该 Token 的最高价值 Trait 信息
			topTraits[trait.TokenId] = types.TraitPrice{
				CollectionAddress: collectionAddr,
				TokenID:           trait.TokenId,
				Trait:             trait.Trait,
				TraitValue:        trait.TraitValue,
				Price:             price,
			}
		}
	}

	// 6. 整理返回结果
	var results []types.TraitPrice
	for _, topTrait := range topTraits {
		results = append(results, topTrait)
	}

	return &types.ItemTopTraitResp{
		Result: results,
	}, nil
}

// GetHistorySalesPrice 获取历史成交价格数据
// 功能:
// 1. 查询指定 Collection 在过去一段时间 (24h, 7d, 30d) 内的成交记录
// 2. 用于前端展示价格走势图
func GetHistorySalesPrice(ctx context.Context, svcCtx *svc.ServerCtx, chain, collectionAddr, duration string) ([]types.HistorySalesPriceInfo, error) {
	var durationTimeStamp int64
	if duration == "24h" {
		durationTimeStamp = 24 * 60 * 60
	} else if duration == "7d" {
		durationTimeStamp = 7 * 24 * 60 * 60
	} else if duration == "30d" {
		durationTimeStamp = 30 * 24 * 60 * 60
	} else {
		return nil, errors.New("only support 24h/7d/30d")
	}

	historySalesPriceInfo, err := svcCtx.Dao.QueryHistorySalesPriceInfo(ctx, chain, collectionAddr, durationTimeStamp)
	if err != nil {
		return nil, errors.Wrap(err, "failed on get history sales price info")
	}

	res := make([]types.HistorySalesPriceInfo, len(historySalesPriceInfo))

	for i, ele := range historySalesPriceInfo {
		res[i] = types.HistorySalesPriceInfo{
			Price:     ele.Price,
			TokenID:   ele.TokenId,
			TimeStamp: ele.EventTime,
		}
	}

	return res, nil
}

// GetItemOwner 获取NFT Item的所有者信息
func GetItemOwner(ctx context.Context, svcCtx *svc.ServerCtx, chainID int64, chain, collectionAddr, tokenID string) (*types.ItemOwner, error) {
	// 从链上获取NFT所有者地址
	address, err := svcCtx.NodeSrvs[chainID].FetchNftOwner(collectionAddr, tokenID)
	if err != nil {
		xzap.WithContext(ctx).Error("failed on fetch nft owner onchain", zap.Error(err))
		return nil, errcode.ErrUnexpected
	}

	// 将地址转换为校验和格式
	owner, err := eip.ToCheckSumAddress(address.String())
	if err != nil {
		xzap.WithContext(ctx).Error("invalid address", zap.Error(err), zap.String("address", address.String()))
		return nil, errcode.ErrUnexpected
	}

	// 更新数据库中的所有者信息
	if err := svcCtx.Dao.UpdateItemOwner(ctx, chain, collectionAddr, tokenID, owner); err != nil {
		xzap.WithContext(ctx).Error("failed on update item owner", zap.Error(err), zap.String("address", address.String()))
	}

	// 返回NFT所有者信息
	return &types.ItemOwner{
		CollectionAddress: collectionAddr,
		TokenID:           tokenID,
		Owner:             owner,
	}, nil
}

// GetItemTraits 获取NFT的 Trait信息
// 主要功能:
// 1. 并发查询三个信息:
//   - NFT的 Trait信息
//   - 集合中每个 Trait的数量统计
//   - 集合基本信息
//
// 2. 计算每个 Trait的百分比
// 3. 组装返回数据
func GetItemTraits(ctx context.Context, svcCtx *svc.ServerCtx, chain, collectionAddr, tokenID string) ([]types.TraitInfo, error) {
	var traitInfos []types.TraitInfo
	var itemTraits []multi.ItemTrait
	var collection *multi.Collection
	var traitCounts []types.TraitCount
	var queryErr error
	var wg sync.WaitGroup

	// 并发查询以下信息:
	// 1. [并发任务 1] 查询 NFT 具体 Trait (属性)
	wg.Add(1)
	go func() {
		defer wg.Done()
		// 查询该 TokenID 拥有的所有 Trait
		itemTraits, queryErr = svcCtx.Dao.QueryItemTraits(
			ctx,
			chain,
			collectionAddr,
			tokenID,
		)
		if queryErr != nil {
			return
		}
	}()

	// 2. [并发任务 2] 查询 Collection Trait 统计 (Trait Counts)
	// 统计该 Collection 下每种 Trait 的数量 (用于计算稀有度)
	wg.Add(1)
	go func() {
		defer wg.Done()
		traitCounts, queryErr = svcCtx.Dao.QueryCollectionTraits(
			ctx,
			chain,
			collectionAddr,
		)
		if queryErr != nil {
			return
		}
	}()

	// 3. [并发任务 3] 查询 Collection 基本信息 (ItemAmount)
	// 需要获取集合的总 Item 数量来计算稀有度百分比
	wg.Add(1)
	go func() {
		defer wg.Done()
		collection, queryErr = svcCtx.Dao.QueryCollectionInfo(
			ctx,
			chain,
			collectionAddr,
		)
		if queryErr != nil {
			return
		}
	}()

	// 4. 等待所有查询完成
	wg.Wait()
	if queryErr != nil {
		return nil, queryErr
	}

	// 如果 NFT 没有 Trait 信息,返回空数组
	if len(itemTraits) == 0 {
		return traitInfos, nil
	}

	// 5. 构建 Trait 数量映射 (Key: TraitType-TraitValue, Value: Count)
	traitCountMap := make(map[string]int64)
	for _, trait := range traitCounts {
		traitCountMap[fmt.Sprintf("%s-%s", trait.Trait, trait.TraitValue)] = trait.Count
	}

	// 6. 计算每个 Trait 的稀有度百分比并组装返回数据
	for _, trait := range itemTraits {
		key := fmt.Sprintf("%s-%s", trait.Trait, trait.TraitValue)
		// 查找该 Trait 在集合中的总出现次数
		if count, ok := traitCountMap[key]; ok {
			traitPercent := 0.0
			if collection.ItemAmount != 0 {
				// 计算公式: (TraitCount / TotalItemAmount) * 100
				traitPercent = decimal.NewFromInt(count).
					DivRound(decimal.NewFromInt(collection.ItemAmount), 4). // 保留4位小数
					Mul(decimal.NewFromInt(100)).
					InexactFloat64()
			}
			traitInfos = append(traitInfos, types.TraitInfo{
				Trait:        trait.Trait,
				TraitValue:   trait.TraitValue,
				TraitAmount:  count,
				TraitPercent: traitPercent,
			})
		}
	}

	return traitInfos, nil
}

// GetCollectionDetail 获取NFT集合的详细信息：基本信息、24小时交易信息、上架数量、地板价、卖单价格、总交易量
// GetCollectionDetail 获取NFT集合的详细信息：基本信息、24小时交易信息、上架数量、地板价、卖单价格、总交易量
func GetCollectionDetail(ctx context.Context, svcCtx *svc.ServerCtx, chain string, collectionAddr string) (*types.CollectionDetailResp, error) {
	// 1. 查询集合基本信息 (Name, Image, OwnerCount 等)
	collection, err := svcCtx.Dao.QueryCollectionInfo(ctx, chain, collectionAddr)
	if err != nil {
		return nil, errors.Wrap(err, "failed on get collection info")
	}

	// 2. 获取集合 24小时 交易统计信息 (Volume, Sales)
	tradeInfos, err := svcCtx.Dao.GetTradeInfoByCollection(chain, collectionAddr, "1d")
	if err != nil {
		xzap.WithContext(ctx).Error("failed on get collection trade info", zap.Error(err))
		// Log error but continue
	}

	// 3. 查询当前上架数量 (Listed Count)
	listed, err := svcCtx.Dao.QueryListedAmount(ctx, chain, collectionAddr)
	if err != nil {
		xzap.WithContext(ctx).Error("failed on get listed count", zap.Error(err))
	} else {
		// 缓存上架数量到 Redis, 供其他无需实时查询的场景使用
		if err := svcCtx.Dao.CacheCollectionsListed(ctx, chain, collectionAddr, int(listed)); err != nil {
			xzap.WithContext(ctx).Error("failed on cache collection listed", zap.Error(err))
		}
	}

	// 4. 查询实时地板价 (Floor Price)
	floorPrice, err := svcCtx.Dao.QueryFloorPrice(ctx, chain, collectionAddr)
	if err != nil {
		xzap.WithContext(ctx).Error("failed on get floor price", zap.Error(err))
	}

	// 5. 查询最近卖单价格 (用于计算 Sell Price)
	collectionSell, err := svcCtx.Dao.QueryCollectionSellPrice(ctx, chain, collectionAddr)
	if err != nil {
		xzap.WithContext(ctx).Error("failed on get floor price", zap.Error(err))
	}

	// 6. 地板价变更检测与事件推送
	// 如果计算出的实时地板价与数据库中存储/上次的地板价不一致, 则触发更新事件
	if !floorPrice.Equal(collection.FloorPrice) {
		// 推送 UpdateCollection 事件到消息队列或处理引擎, 异步更新系统状态
		if err := ordermanager.AddUpdatePriceEvent(svcCtx.KvStore, &ordermanager.TradeEvent{
			EventType:      ordermanager.UpdateCollection,
			CollectionAddr: collectionAddr,
			Price:          floorPrice,
		}, chain); err != nil {
			xzap.WithContext(ctx).Error("failed on update floor price", zap.Error(err))
		}
	}

	// 7. 处理 24小时 交易量数据
	var volume24h decimal.Decimal
	var sold int64
	if tradeInfos != nil {
		volume24h = tradeInfos.Volume
		sold = tradeInfos.ItemCount
	}

	// 8. 查询集合历史总交易量 (Total Volume)
	var allVol decimal.Decimal
	collectionVol, err := svcCtx.Dao.GetCollectionVolume(chain, collectionAddr)
	if err != nil {
		xzap.WithContext(ctx).Error("failed on query collection all volume", zap.Error(err))
	} else {
		allVol = collectionVol
	}

	// 9. 构建完整返回结果
	detail := types.CollectionDetail{
		ImageUri:    collection.ImageUri, // svcCtx.ImageMgr.GetFileUrl(collection.ImageUri),
		Name:        collection.Name,
		Address:     collection.Address,
		ChainId:     collection.ChainId,
		FloorPrice:  floorPrice,
		SellPrice:   collectionSell.SalePrice.String(),
		VolumeTotal: allVol,
		Volume24h:   volume24h,
		Sold24h:     sold,
		ListAmount:  listed,
		TotalSupply: collection.ItemAmount,  // 总发行量
		OwnerAmount: collection.OwnerAmount, // 持有人数
	}

	return &types.CollectionDetailResp{
		Result: detail,
	}, nil
}

// RefreshItemMetadata以此刷新 Item 元数据
// 功能:
// 1. 将刷新任务推送到 Redis 队列
// 2. 后台 Indexer (EasySwapSync) 会消费队列并重新抓取链上/IPFS 元数据
func RefreshItemMetadata(ctx context.Context, svcCtx *svc.ServerCtx, chainName string, chainId int64, collectionAddress, tokenId string) error {
	if err := mq.AddSingleItemToRefreshMetadataQueue(svcCtx.KvStore, svcCtx.C.ProjectCfg.Name, chainName, chainId, collectionAddress, tokenId); err != nil {
		xzap.WithContext(ctx).Error("failed on add item to refresh queue", zap.Error(err), zap.String("collection address: ", collectionAddress), zap.String("item_id", tokenId))
		return errcode.ErrUnexpected
	}

	return nil

}

// GetItemImage 获取 Item 图片链接
// 功能: 优先返回 CDN/OSS 链接，如果没有则返回原始链接
func GetItemImage(ctx context.Context, svcCtx *svc.ServerCtx, chain string, collectionAddress, tokenId string) (*types.ItemImage, error) {
	items, err := svcCtx.Dao.QueryCollectionItemsImage(ctx, chain, collectionAddress, []string{tokenId})
	if err != nil || len(items) == 0 {
		return nil, errors.Wrap(err, "failed on get item image")
	}
	var imageUri string
	if items[0].IsUploadedOss {
		imageUri = items[0].OssUri // svcCtx.ImageMgr.GetSmallSizeImageUrl(items[0].OssUri)
	} else {
		imageUri = items[0].ImageUri // svcCtx.ImageMgr.GetSmallSizeImageUrl(items[0].ImageUri)
	}

	return &types.ItemImage{
		CollectionAddress: collectionAddress,
		TokenID:           tokenId,
		ImageUri:          imageUri,
	}, nil
}
