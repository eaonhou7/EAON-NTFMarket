package dao

import (
	"context"
	"fmt"
	"strings"

	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/multi"
	"github.com/pkg/errors"
)

// QueryCollectionItemsImage 查询集合内 Item 的多媒体资源信息 (图片/视频)
// 功能:
// 1. 根据集合地址和 TokenID 列表, 批量查询 Items 的外部资源链接
// 2. 返回字段包括: 图片URI, OSS链接, 视频URI, 视频类型等
// 3. 主要用于前端展示 NFT 的多媒体内容
func (d *Dao) QueryCollectionItemsImage(ctx context.Context, chain string,
	collectionAddr string, tokenIds []string) ([]multi.ItemExternal, error) {
	var itemsExternal []multi.ItemExternal

	// SQL 逻辑:
	// SELECT ... FROM item_external_table
	// WHERE collection_address = ? AND token_id IN (?)
	if err := d.DB.WithContext(ctx).
		Table(multi.ItemExternalTableName(chain)).
		Select("collection_address, token_id, is_uploaded_oss, "+
			"image_uri, oss_uri, video_type, is_video_uploaded, "+
			"video_uri, video_oss_uri").
		Where("collection_address = ? and token_id in (?)",
			collectionAddr, tokenIds).
		Scan(&itemsExternal).Error; err != nil {
		return nil, errors.Wrap(err, "failed on query items external info")
	}

	return itemsExternal, nil
}

// QueryMultiChainCollectionsItemsImage 跨链批量查询 NFT Item 的图片信息
// 功能:
// 1. 针对多个 Item (可能分布在不同链上), 获取其图片/多媒体信息
// 2. 自动根据 ChainName 分组并构建 UNION ALL 查询
// 3. 这里的 ItemInfo 只需要包含 ChainName, CollectionAddress, TokenID
func (d *Dao) QueryMultiChainCollectionsItemsImage(ctx context.Context, itemInfos []MultiChainItemInfo) ([]multi.ItemExternal, error) {
	var itemsExternal []multi.ItemExternal

	// SQL 拼接所需的首尾部分
	sqlHead := "SELECT * FROM ("
	sqlTail := ") as combined"
	var sqlMids []string

	// 1. 按链 (ChainName) 对传入的 itemInfos 进行分组
	//    Input: [Item(Eth, A, 1), Item(Poly, B, 2), Item(Eth, A, 3)]
	//    Output: { "eth": [Item(A, 1), Item(A, 3)], "poly": [Item(B, 2)] }
	chainItems := make(map[string][]MultiChainItemInfo)
	for _, itemInfo := range itemInfos {
		items, ok := chainItems[strings.ToLower(itemInfo.ChainName)]
		if ok {
			items = append(items, itemInfo)
			chainItems[strings.ToLower(itemInfo.ChainName)] = items
		} else {
			chainItems[strings.ToLower(itemInfo.ChainName)] = []MultiChainItemInfo{itemInfo}
		}
	}

	// 2. 遍历每个链, 构建对应的子查询 SQL
	for chainName, items := range chainItems {
		// 构建 IN 子句的值列表: (('addr1', 'id1'), ('addr2', 'id2'), ...)
		tmpStat := fmt.Sprintf("(('%s','%s')", items[0].CollectionAddress, items[0].TokenID)
		for i := 1; i < len(items); i++ {
			tmpStat += fmt.Sprintf(",('%s','%s')", items[i].CollectionAddress, items[i].TokenID)
		}
		tmpStat += ") "

		// 构建单个链的 SELECT 语句
		// SELECT collection_address, token_id... FROM item_external_{chain} WHERE (col, id) IN (...)
		sqlMid := "("
		sqlMid += "select collection_address, token_id, is_uploaded_oss, image_uri, oss_uri "
		sqlMid += fmt.Sprintf("from %s ", multi.ItemExternalTableName(chainName))
		sqlMid += "where (collection_address,token_id) in "
		sqlMid += tmpStat
		sqlMid += ")"

		sqlMids = append(sqlMids, sqlMid)
	}

	// 3. 使用 UNION ALL 拼接所有子查询
	sql := sqlHead
	for i := 0; i < len(sqlMids); i++ {
		if i != 0 {
			sql += " UNION ALL "
		}
		sql += sqlMids[i]
	}
	sql += sqlTail

	// 4. 执行最终的 SQL
	if err := d.DB.WithContext(ctx).Raw(sql).Scan(&itemsExternal).Error; err != nil {
		return nil, errors.Wrap(err, "failed on query multi chain items external info")
	}

	return itemsExternal, nil
}
