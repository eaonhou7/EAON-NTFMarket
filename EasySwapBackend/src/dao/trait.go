package dao

import (
	"context"

	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/multi"
	"github.com/pkg/errors"

	"github.com/ProjectsTask/EasySwapBackend/src/types/v1"
)

// QueryItemTraits 查询单个 NFT Item 的 Trait (属性) 信息
// 功能: 根据集合地址和 TokenID 从数据库查询其属性列表
func (d *Dao) QueryItemTraits(ctx context.Context, chain string, collectionAddr string, tokenID string) ([]multi.ItemTrait, error) {
	var itemTraits []multi.ItemTrait
	// SQL 逻辑:
	// SELECT collection_address, token_id, trait, trait_value
	// FROM item_trait_table
	// WHERE collection_address = ? AND token_id = ?
	if err := d.DB.WithContext(ctx).Table(multi.ItemTraitTableName(chain)).
		Select("collection_address, token_id, trait, trait_value").
		Where("collection_address = ? and token_id = ?", collectionAddr, tokenID).
		Scan(&itemTraits).Error; err != nil {
		return nil, errors.Wrap(err, "failed on query items trait info")
	}

	return itemTraits, nil
}

// QueryItemsTraits 批量查询多个 NFT Item 的 Trait 信息
// 功能: 用于列表页或购物车展示多个 Item 的属性详情
func (d *Dao) QueryItemsTraits(ctx context.Context, chain string, collectionAddr string, tokenIds []string) ([]multi.ItemTrait, error) {
	var itemsTraits []multi.ItemTrait
	// SQL 逻辑:
	// SELECT ... FROM item_trait_table
	// WHERE collection_address = ? AND token_id IN (?)
	if err := d.DB.WithContext(ctx).Table(multi.ItemTraitTableName(chain)).
		Select("collection_address, token_id, trait, trait_value").
		Where("collection_address = ? and token_id in (?)", collectionAddr, tokenIds).
		Scan(&itemsTraits).Error; err != nil {
		return nil, errors.Wrap(err, "failed on query items trait info")
	}

	return itemsTraits, nil
}

// QueryCollectionTraits 查询集合内所有 Trait 的统计信息
// 功能: 统计每个属性 (Trait Type + Value) 出现的次数
// 用途: 侧边栏筛选器 (Filter Sidebar), 展示如 "Background: Red (15)"
func (d *Dao) QueryCollectionTraits(ctx context.Context, chain string, collectionAddr string) ([]types.TraitCount, error) {
	var traitCounts []types.TraitCount

	// SQL 逻辑:
	// SELECT trait, trait_value, COUNT(*) as count
	// FROM item_trait_table
	// WHERE collection_address = ?
	// GROUP BY trait, trait_value
	// 注意: 使用反引号 `trait` 防止关键字冲突(尽管trait本身不是sql关键字，但保持习惯)
	if err := d.DB.WithContext(ctx).Table(multi.ItemTraitTableName(chain)).
		Select("`trait`,`trait_value`,count(*) as count").Where("collection_address=?", collectionAddr).
		Group("`trait`,`trait_value`").
		Scan(&traitCounts).Error; err != nil {
		return nil, errors.Wrap(err, "failed on query collection trait amount")
	}

	return traitCounts, nil
}
