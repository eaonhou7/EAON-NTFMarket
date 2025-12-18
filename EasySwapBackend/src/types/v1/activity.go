package types

import (
	"github.com/shopspring/decimal"
)

// ActivityMultiChainFilterParams 多链活动查询过滤参数
type ActivityMultiChainFilterParams struct {
	ChainID             []int    `json:"filter_ids"`
	CollectionAddresses []string `json:"collection_addresses"` // 集合地址列表
	TokenID             string   `json:"token_id"`             // Token ID
	UserAddresses       []string `json:"user_addresses"`       // 用户地址列表 (作为 Maker 或 Taker)
	EventTypes          []string `json:"event_types"`          // 事件类型: Sale, List, Offer, Transfer, Mint, Cancel
	Page                int      `json:"page"`
	PageSize            int      `json:"page_size"`
}

// ActivityInfo 活动详情信息
type ActivityInfo struct {
	EventType          string          `json:"event_type"`           // 事件类型
	EventTime          int64           `json:"event_time"`           // 发生时间
	ImageURI           string          `json:"image_uri"`            // NFT 图片
	CollectionAddress  string          `json:"collection_address"`   // 集合地址
	CollectionName     string          `json:"collection_name"`      // 集合名称
	CollectionImageURI string          `json:"collection_image_uri"` // 集合图片
	TokenID            string          `json:"token_id"`             // Token ID
	ItemName           string          `json:"item_name"`            // Item 名称
	Currency           string          `json:"currency"`             // 支付代币 (ETH/WETH)
	Price              decimal.Decimal `json:"price"`                // 价格
	Maker              string          `json:"maker"`                // 发起方
	Taker              string          `json:"taker"`                // 接收方 (成交时)
	TxHash             string          `json:"tx_hash"`              // 交易哈希
	MarketplaceID      int             `json:"marketplace_id"`       // 市场平台 ID
	ChainID            int             `json:"chain_id"`             // 链 ID
}

type ActivityResp struct {
	Result interface{} `json:"result"`
	Count  int64       `json:"count"`
}
