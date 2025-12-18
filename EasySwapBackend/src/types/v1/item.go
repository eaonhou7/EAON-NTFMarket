package types

import "github.com/shopspring/decimal"

// ItemInfo Item 基本标识信息
type ItemInfo struct {
	CollectionAddress string `json:"collection_address"`
	TokenID           string `json:"token_id"`
}

// ItemPriceInfo Item 价格和挂单状态
type ItemPriceInfo struct {
	CollectionAddress string          `json:"collection_address"`
	TokenID           string          `json:"token_id"`
	Maker             string          `json:"maker"`        // 挂单者
	Price             decimal.Decimal `json:"price"`        // 价格
	OrderStatus       int             `json:"order_status"` // 订单状态
}

// ItemOwner Item 所有权信息
type ItemOwner struct {
	CollectionAddress string `json:"collection_address"`
	TokenID           string `json:"token_id"`
	Owner             string `json:"owner"`
}

// ItemImage Item 图片资源信息
type ItemImage struct {
	CollectionAddress string `json:"collection_address"`
	TokenID           string `json:"token_id"`
	ImageUri          string `json:"image_uri"`
}

// ItemDetailInfo Item 完整详情 (聚合视图)
type ItemDetailInfo struct {
	ChainID            int             `json:"chain_id"`
	Name               string          `json:"name"`                 // 名称
	CollectionAddress  string          `json:"collection_address"`   // 集合地址
	CollectionName     string          `json:"collection_name"`      // 集合名称
	CollectionImageURI string          `json:"collection_image_uri"` // 集合图片
	TokenID            string          `json:"token_id"`             // Token ID
	ImageURI           string          `json:"image_uri"`            // 图片
	VideoType          string          `json:"video_type"`           // 视频类型
	VideoURI           string          `json:"video_uri"`            // 视频链接
	LastSellPrice      decimal.Decimal `json:"last_sell_price"`      // 最近成交价
	FloorPrice         decimal.Decimal `json:"floor_price"`          // 当前集合地板价
	OwnerAddress       string          `json:"owner_address"`        // 持有人
	MarketplaceID      int             `json:"marketplace_id"`       // 挂单所在市场

	// 挂单详情
	ListOrderID    string          `json:"list_order_id"`
	ListTime       int64           `json:"list_time"`
	ListPrice      decimal.Decimal `json:"list_price"`
	ListExpireTime int64           `json:"list_expire_time"`
	ListSalt       int64           `json:"list_salt"`
	ListMaker      string          `json:"list_maker"`

	// 最佳出价详情
	BidOrderID    string          `json:"bid_order_id"`
	BidTime       int64           `json:"bid_time"`
	BidExpireTime int64           `json:"bid_expire_time"`
	BidPrice      decimal.Decimal `json:"bid_price"`
	BidSalt       int64           `json:"bid_salt"`
	BidMaker      string          `json:"bid_maker"`
	BidType       int64           `json:"bid_type"`
	BidSize       int64           `json:"bid_size"`
	BidUnfilled   int64           `json:"bid_unfilled"`
}

type ItemDetailInfoResp struct {
	Result interface{} `json:"result"`
}

type ListingInfo struct {
	MarketplaceId int32           `json:"marketplace_id"`
	Price         decimal.Decimal `json:"price"`
}

// TraitPrice 属性价格信息
type TraitPrice struct {
	CollectionAddress string          `json:"collection_address"`
	TokenID           string          `json:"token_id"`
	Trait             string          `json:"trait"`       // 属性名
	TraitValue        string          `json:"trait_value"` // 属性值
	Price             decimal.Decimal `json:"price"`       // 对应的最低挂单价
}

type ItemTopTraitResp struct {
	Result interface{} `json:"result"`
}
