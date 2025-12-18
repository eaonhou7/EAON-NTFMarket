package types

import (
	"github.com/shopspring/decimal"
)

// CollectionItemFilterParams 集合 Item 列表查询过滤参数
type CollectionItemFilterParams struct {
	Sort        int    `json:"sort"`         // 排序方式: 1-价格升序 2-挂单时间降序 3-成交价降序
	Status      []int  `json:"status"`       // 状态过滤: 1-一口价(BuyNow) 2-有出价(HasOffer) 3-全选
	Markets     []int  `json:"markets"`      // 市场过滤: 0-NS 1-OpenSea 2-LooksRare 3-X2Y2
	TokenID     string `json:"token_id"`     // 按 TokenID 搜索
	UserAddress string `json:"user_address"` // 当前用户地址(用于查询是否持有)
	ChainID     int    `json:"chain_id"`     // 链 ID
	Page        int    `json:"page"`         // 页码
	PageSize    int    `json:"page_size"`    // 每页数量
}

// CollectionBidFilterParams 集合 Bids 查询过滤参数
type CollectionBidFilterParams struct {
	ChainID  int `json:"chain_id"`
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// CollectionBids 集合出价统计信息
type CollectionBids struct {
	Price   decimal.Decimal `json:"price"`   // 出价金额
	Size    int             `json:"size"`    // 出价数量
	Total   decimal.Decimal `json:"total"`   // 总金额
	Bidders int             `json:"bidders"` // 出价人数
}

// CollectionBidsResp 集合 Bids 响应
type CollectionBidsResp struct {
	Result interface{} `json:"result"`
	Count  int64       `json:"count"`
}

// HistorySalesPriceInfo 历史成交价格信息
type HistorySalesPriceInfo struct {
	Price     decimal.Decimal `json:"price"`      // 成交价格
	TokenID   string          `json:"token_id"`   // Token ID
	TimeStamp int64           `json:"time_stamp"` // 成交时间戳
}

// TopTraitFilterParams 热门属性查询参数
type TopTraitFilterParams struct {
	TokenIds []string `json:"token_ids"`
	ChainID  int      `json:"chain_id"`
}

type NFTListingInfoResp struct {
	Result interface{} `json:"result"`
	Count  int64       `json:"count"`
}

// NFTListingInfo NFT 列表展示信息 (聚合了 Item、Listing、Bid)
type NFTListingInfo struct {
	Name              string      `json:"name"`               // NFT 名称
	ImageURI          string      `json:"image_uri"`          // 图片链接
	VideoType         string      `json:"video_type"`         // 视频类型
	VideoURI          string      `json:"video_uri"`          // 视频链接
	CollectionAddress string      `json:"collection_address"` // 集合地址
	TokenID           string      `json:"token_id"`           // Token ID
	OwnerAddress      string      `json:"owner_address"`      // 所有者地址
	Traits            []ItemTrait `json:"traits"`             // 属性列表

	// 挂单信息 (Listing)
	ListOrderID    string          `json:"list_order_id"`    // 挂单 Order ID
	ListTime       int64           `json:"list_time"`        // 挂单时间
	ListPrice      decimal.Decimal `json:"list_price"`       // 挂单价格
	ListExpireTime int64           `json:"list_expire_time"` // 过期时间
	ListSalt       int64           `json:"list_salt"`        // 盐值
	ListMaker      string          `json:"list_maker"`       // 挂单者

	// 最佳出价信息 (Best Bid)
	BidOrderID    string          `json:"bid_order_id"`    // 出价 Order ID
	BidTime       int64           `json:"bid_time"`        // 出价时间
	BidExpireTime int64           `json:"bid_expire_time"` // 过期时间
	BidPrice      decimal.Decimal `json:"bid_price"`       // 出价金额
	BidSalt       int64           `json:"bid_salt"`        // 盐值
	BidMaker      string          `json:"bid_maker"`       // 出价者
	BidType       int64           `json:"bid_type"`        // 出价类型 (Item/Collection)
	BidSize       int64           `json:"bid_size"`        // 出价数量
	BidUnfilled   int64           `json:"bid_unfilled"`    // 未成交数量

	MarketID int `json:"market_id"` // 市场 ID

	LastSellPrice    decimal.Decimal `json:"last_sell_price"`    // 最近成交价
	OwnerOwnedAmount int64           `json:"owner_owned_amount"` // 当前用户持有数量
}

type ItemTrait struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

// CollectionRankingInfo 集合排名信息
type CollectionRankingInfo struct {
	ImageUri    string          `json:"image_uri"`          // 集合图片
	Name        string          `json:"name"`               // 集合名称
	Address     string          `json:"address"`            // 集合地址
	FloorPrice  string          `json:"floor_price"`        // 地板价
	FloorChange string          `json:"floor_price_change"` // 地板价涨跌幅
	SellPrice   string          `json:"sell_price"`         // 挂单价格
	Volume      decimal.Decimal `json:"volume"`             // 交易量
	ItemNum     int64           `json:"item_num"`           // NFT 总数
	ItemOwner   int64           `json:"item_owner"`         // 持有人数
	ItemSold    int64           `json:"item_sold"`          // 已售数量
	ListAmount  int             `json:"list_amount"`        // 挂单数量
	ChainID     int             `json:"chain_id"`           // 链 ID
}

type CollectionRankingResp struct {
	Result interface{} `json:"result"`
}

// CollectionDetail 集合详情
type CollectionDetail struct {
	ImageUri       string          `json:"image_uri"`        // 集合图片
	Name           string          `json:"name"`             // 名称
	Address        string          `json:"address"`          // 地址
	ChainId        int             `json:"chain_id"`         // 链 ID
	FloorPrice     decimal.Decimal `json:"floor_price"`      // 地板价
	SellPrice      string          `json:"sell_price"`       // 挂单价
	VolumeTotal    decimal.Decimal `json:"volume_total"`     // 总交易量
	Volume24h      decimal.Decimal `json:"volume_24h"`       // 24小时交易量
	Sold24h        int64           `json:"sold_24h"`         // 24小时销量
	ListAmount     int64           `json:"list_amount"`      // 上架数量
	TotalSupply    int64           `json:"total_supply"`     // 总供应量
	OwnerAmount    int64           `json:"owner_amount"`     // 持有人数
	RoyaltyFeeRate string          `json:"royalty_fee_rate"` // 版权税率
}

type CollectionDetailResp struct {
	Result interface{} `json:"result"`
}

type CommonResp struct {
	Result interface{} `json:"result"`
}

type RefreshItem struct {
	ChainID        int64  `json:"chain_id"`
	CollectionAddr string `json:"collection_addr"`
	TokenID        string `json:"token_id"`
}

type CollectionListed struct {
	CollectionAddr string `json:"collection_address"`
	Count          int    `json:"count"`
}
