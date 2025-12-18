package types

import "github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/multi"

// TraitCount 集合属性统计信息
type TraitCount struct {
	multi.ItemTrait
	Count int64 `json:"count"` // 拥有该属性的 NFT 数量
}

type ItemTraitsResp struct {
	Result interface{} `json:"result"`
}

// TraitInfo 属性百分比信息
type TraitInfo struct {
	Trait        string  `json:"trait"`         // 属性名
	TraitValue   string  `json:"trait_value"`   // 属性值
	TraitAmount  int64   `json:"trait_amount"`  // 拥有数量
	TraitPercent float64 `json:"trait_percent"` // 稀有度百分比 (拥有量/总量)
}

// TraitValue 属性值详情
type TraitValue struct {
	TraitValue  string `json:"trait_value"`  // 属性值
	TraitAmount int64  `json:"trait_amount"` // 数量
}

// CollectionTraitInfo 集合属性聚合信息
type CollectionTraitInfo struct {
	Trait  string       `json:"trait"`  // 属性名 (e.g. Background)
	Values []TraitValue `json:"values"` // 该属性下的所有可选值
}
