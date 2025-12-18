package types

// OrderInfosParam 订单信息聚合查询参数
type OrderInfosParam struct {
	ChainID           int      `json:"chain_id"`           // 链 ID
	UserAddress       string   `json:"user_address"`       // 用户地址 (查询该用户相关订单)
	CollectionAddress string   `json:"collection_address"` // 集合地址
	TokenIds          []string `json:"token_ids"`          // Token ID 列表
}
