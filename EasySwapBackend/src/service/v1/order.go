package service

import (
	"context"
	"sort"

	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/multi"
	"github.com/pkg/errors"

	"github.com/ProjectsTask/EasySwapBackend/src/service/svc"
	"github.com/ProjectsTask/EasySwapBackend/src/types/v1"
)

// GetOrderInfos 获取订单信息
// 该函数主要用于获取指定NFT的出价信息,包括单个NFT的最高出价和整个Collection的最高出价
func GetOrderInfos(ctx context.Context, svcCtx *svc.ServerCtx, chainID int, chain string, userAddr string, collectionAddr string, tokenIds []string) ([]types.ItemBid, error) {
	// 1. 构建NFT信息列表
	var items []types.ItemInfo
	for _, tokenID := range tokenIds {
		items = append(items, types.ItemInfo{
			CollectionAddress: collectionAddr,
			TokenID:           tokenID,
		})
	}

	// 2. 查询每个NFT的最高出价信息
	bids, err := svcCtx.Dao.QueryItemsBestBids(ctx, chain, userAddr, items)
	if err != nil {
		return nil, errors.Wrap(err, "failed on query items best bids")
	}

	// 3. 处理每个NFT的最高出价,如果有多个出价选择最高的
	itemsBestBids := make(map[string]multi.Order)
	for _, bid := range bids {
		order, ok := itemsBestBids[bid.TokenId]
		if !ok {
			itemsBestBids[bid.TokenId] = bid
			continue
		}
		if bid.Price.GreaterThan(order.Price) {
			itemsBestBids[bid.TokenId] = bid
		}
	}

	// 4. 查询整个Collection的最高出价信息
	collectionBids, err := svcCtx.Dao.QueryCollectionTopNBid(ctx, chain, userAddr, collectionAddr, len(tokenIds))
	if err != nil {
		return nil, errors.Wrap(err, "failed on query collection best bids")
	}

	// 5. 处理并返回最终的出价信息
	return processBids(tokenIds, itemsBestBids, collectionBids, collectionAddr), nil
}

// processBids 处理NFT的出价信息,返回每个NFT的最高出价
// 参数说明:
// - tokenIds: NFT的token ID列表
// - itemsBestBids: 每个NFT的"单品"最高出价信息,key为tokenId (Specific Item Offer)
// - collectionBids: 整个"Collection"的最高出价列表 (Collection Offer)
// - collectionAddr: Collection地址
//
// 处理逻辑 (撮合算法):
// 1. 将 单品出价 (itemsBestBids) 按价格升序排序
// 2. 遍历目标 tokenIds 列表:
//   - 情况A (无单品出价): 如果该 Token 没有指定出价，尝试匹配当前可用的最高 Collection Offer。
//   - 情况B (有单品出价):
//   - 比较 单品出价(ItemPrice) vs 集合出价(CollectionPrice)
//   - 优先展示价格更高者 (Price Discovery)
//   - 若使用集合出价，则消耗该集合出价额度 (cBidIndex++)
//
// 3. 返回每个 Token 最终使用的有效出价(ItemBid)列表
func processBids(tokenIds []string, itemsBestBids map[string]multi.Order, collectionBids []multi.Order, collectionAddr string) []types.ItemBid {
	// 将 itemsBestBids map 转换为切片并按价格升序排序
	var itemsSortedBids []multi.Order
	for _, bid := range itemsBestBids {
		itemsSortedBids = append(itemsSortedBids, bid)
	}
	sort.SliceStable(itemsSortedBids, func(i, j int) bool {
		return itemsSortedBids[i].Price.LessThan(itemsSortedBids[j].Price)
	})

	var resultBids []types.ItemBid
	var cBidIndex int // Collection级别出价的遍历索引

	// 第一阶段：处理没有单独出价的NFT (优先消耗 Collection Offers)
	for _, tokenId := range tokenIds {
		if _, ok := itemsBestBids[tokenId]; !ok {
			// 如果有剩余的 Collection 级别出价, 使用它
			if cBidIndex < len(collectionBids) {
				resultBids = append(resultBids, types.ItemBid{
					MarketplaceId:     collectionBids[cBidIndex].MarketplaceId,
					CollectionAddress: collectionAddr,
					TokenId:           tokenId,
					OrderID:           collectionBids[cBidIndex].OrderID,
					EventTime:         collectionBids[cBidIndex].EventTime,
					ExpireTime:        collectionBids[cBidIndex].ExpireTime,
					Price:             collectionBids[cBidIndex].Price,
					Salt:              collectionBids[cBidIndex].Salt,
					BidSize:           collectionBids[cBidIndex].Size,
					BidUnfilled:       collectionBids[cBidIndex].QuantityRemaining,
					Bidder:            collectionBids[cBidIndex].Maker,
					OrderType:         getBidType(collectionBids[cBidIndex].OrderType),
				})
				cBidIndex++ // 消耗一个集合出价
			}
		}
	}

	// 第二阶段：处理有单独出价的NFT (比较 Item Offer vs Collection Offer)
	for _, itemBid := range itemsSortedBids {
		if cBidIndex >= len(collectionBids) {
			// case 1: 如果没有更多 Collection 级别的出价 (Collection Offers 耗尽)
			// 直接使用 NFT 自己的单品出价
			resultBids = append(resultBids, types.ItemBid{
				MarketplaceId:     itemBid.MarketplaceId,
				CollectionAddress: collectionAddr,
				TokenId:           itemBid.TokenId,
				OrderID:           itemBid.OrderID,
				EventTime:         itemBid.EventTime,
				ExpireTime:        itemBid.ExpireTime,
				Price:             itemBid.Price,
				Salt:              itemBid.Salt,
				BidSize:           itemBid.Size,
				BidUnfilled:       itemBid.QuantityRemaining,
				Bidder:            itemBid.Maker,
				OrderType:         getBidType(itemBid.OrderType),
			})
		} else {
			// case 2: 还有 Collection Offer，需要比较 Item Offer 和 Collection Offer 的价格
			cBid := collectionBids[cBidIndex]

			if cBid.Price.GreaterThan(itemBid.Price) {
				// [Price Logic] 如果 Collection 的出价更高
				// 优先展示 Collection 出价 (对卖家有利)
				// 并消耗一个 Collection Offer 配额
				resultBids = append(resultBids, types.ItemBid{
					MarketplaceId:     cBid.MarketplaceId,
					CollectionAddress: collectionAddr,
					TokenId:           itemBid.TokenId,
					OrderID:           cBid.OrderID,
					EventTime:         cBid.EventTime,
					ExpireTime:        cBid.ExpireTime,
					Price:             cBid.Price,
					Salt:              cBid.Salt,
					BidSize:           cBid.Size,
					BidUnfilled:       cBid.QuantityRemaining,
					Bidder:            cBid.Maker,
					OrderType:         getBidType(cBid.OrderType),
				})
				cBidIndex++ // 消耗该集合出价
			} else {
				// [Price Logic] 如果 NFT 自己的出价更高
				// 使用 NFT 的单品出价, 不消耗 Collection Offer (留给其他没有更高单品出价的 Item)
				resultBids = append(resultBids, types.ItemBid{
					MarketplaceId:     itemBid.MarketplaceId,
					CollectionAddress: collectionAddr,
					TokenId:           itemBid.TokenId,
					OrderID:           itemBid.OrderID,
					EventTime:         itemBid.EventTime,
					ExpireTime:        itemBid.ExpireTime,
					Price:             itemBid.Price,
					Salt:              itemBid.Salt,
					BidSize:           itemBid.Size,
					BidUnfilled:       itemBid.QuantityRemaining,
					Bidder:            itemBid.Maker,
					OrderType:         getBidType(itemBid.OrderType),
				})
			}
		}
	}

	return resultBids
}
