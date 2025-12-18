package types

import "github.com/shopspring/decimal"

// ItemBid 单个 Item 的出价详情
type ItemBid struct {
	MarketplaceId     int             `json:"marketplace_id"`     // 市场 ID
	CollectionAddress string          `json:"collection_address"` // 集合地址
	TokenId           string          `json:"token_id"`           // Token ID
	OrderID           string          `json:"order_id"`           // 订单唯一 ID (Hash)
	EventTime         int64           `json:"event_time"`         // 出价时间
	ExpireTime        int64           `json:"expire_time"`        // 过期时间 (秒)
	Price             decimal.Decimal `json:"price"`              // 价格
	Salt              int64           `json:"salt"`               // 盐值
	BidSize           int64           `json:"bid_size"`           // 想要购买的数量
	BidUnfilled       int64           `json:"bid_unfilled"`       // 剩余未成交数量
	Bidder            string          `json:"bidder"`             // 出价人地址
	OrderType         int64           `json:"order_type"`         // 订单类型
}
