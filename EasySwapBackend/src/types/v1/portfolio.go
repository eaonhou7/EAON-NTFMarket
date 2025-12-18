package types

import (
	"github.com/shopspring/decimal"
)

// UserCollectionsParams 用户集合列表查询参数
type UserCollectionsParams struct {
	UserAddresses []string `json:"user_addresses"` // 用户地址列表
}

// UserCollections 用户集合聚合信息
type UserCollections struct {
	ChainID    int             `json:"chain_id"`    // 链 ID
	Address    string          `json:"address"`     // 集合地址
	Name       string          `json:"name"`        // 集合名称
	Symbol     string          `json:"symbol"`      // 符号
	ImageURI   string          `json:"image_uri"`   // 图片
	ItemCount  int64           `json:"item_count"`  // 用户持有数量
	FloorPrice decimal.Decimal `json:"floor_price"` // 地板价
	ItemAmount int64           `json:"item_amount"` // 集合总供应量
}

// CollectionInfo 集合基础信息
type CollectionInfo struct {
	ChainID    int             `json:"chain_id"`
	Name       string          `json:"name"`
	Address    string          `json:"address"`
	Symbol     string          `json:"symbol"`
	ImageURI   string          `json:"image_uri"`
	ListAmount int             `json:"list_amount"` // 挂单总数
	ItemAmount int64           `json:"item_amount"` // 发行总量
	FloorPrice decimal.Decimal `json:"floor_price"` // 地板价
}

// ChainInfo 链维度资产统计
type ChainInfo struct {
	ChainID   int             `json:"chain_id"`
	ItemOwned int64           `json:"item_owned"` // 该链下持有的 NFT 总数
	ItemValue decimal.Decimal `json:"item_value"` // 该链下持有的 NFT 估值 (基于地板价?)
}

// UserCollectionsData 用户概览数据
type UserCollectionsData struct {
	CollectionInfos []CollectionInfo `json:"collection_info"` // 各集合详情
	ChainInfos      []ChainInfo      `json:"chain_info"`      // 各链统计
}

type UserCollectionsResp struct {
	Result interface{} `json:"result"`
}

// PortfolioMultiChainItemFilterParams 多链 Item 列表查询参数
type PortfolioMultiChainItemFilterParams struct {
	ChainID             []int    `json:"chain_id"`             // 链 ID 列表
	CollectionAddresses []string `json:"collection_addresses"` // 集合地址过滤
	UserAddresses       []string `json:"user_addresses"`       // 用户地址 (查询谁的 NFT)

	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// PortfolioMultiChainListingFilterParams 多链挂单查询参数
type PortfolioMultiChainListingFilterParams struct {
	ChainID             []int    `json:"chain_id"`
	CollectionAddresses []string `json:"collection_addresses"`
	UserAddresses       []string `json:"user_addresses"`

	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// PortfolioMultiChainBidFilterParams 多链 Bid 查询参数
type PortfolioMultiChainBidFilterParams struct {
	ChainID             []int    `json:"chain_id"`
	CollectionAddresses []string `json:"collection_addresses"`
	UserAddresses       []string `json:"user_addresses"`

	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}

// PortfolioItemInfo 个人中心 Item 详情
type PortfolioItemInfo struct {
	ChainID            int    `json:"chain_id"`
	CollectionAddress  string `json:"collection_address"`
	CollectionName     string `json:"collection_name"`
	CollectionImageURI string `json:"collection_image_uri"`
	TokenID            string `json:"token_id"`
	ImageURI           string `json:"image_uri"`

	LastCostPrice float64         `json:"last_cost_price"` // 上次购买价格
	OwnedTime     int64           `json:"owned_time"`      // 持有时长/获取时间
	Owner         string          `json:"owner"`           // 所有者
	Listing       bool            `json:"listing"`         // 是否挂单中
	MarketplaceID int             `json:"marketplace_id"`  // 市场 ID
	Name          string          `json:"name"`            // 名称
	FloorPrice    decimal.Decimal `json:"floor_price"`     // 当前地板价

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

type UserItemsResp struct {
	Result interface{} `json:"result"`
	Count  int64       `json:"count"`
}

type UserListingsResp struct {
	Count  int64     `json:"count"`
	Result []Listing `json:"result"`
}

// Listing 个人中心挂单信息
type Listing struct {
	CollectionAddress string          `json:"collection_address"`
	CollectionName    string          `json:"collection_name"`
	ImageURI          string          `json:"image_uri"`
	Name              string          `json:"name"`
	TokenID           string          `json:"token_id"`
	LastCostPrice     decimal.Decimal `json:"last_cost_price"`
	MarketplaceID     int             `json:"marketplace_id"`
	ChainID           int             `json:"chain_id"`

	ListOrderID    string          `json:"list_order_id"`
	ListTime       int64           `json:"list_time"`
	ListPrice      decimal.Decimal `json:"list_price"`
	ListExpireTime int64           `json:"list_expire_time"`
	ListSalt       int64           `json:"list_salt"`
	ListMaker      string          `json:"list_maker"`

	BidOrderID    string          `json:"bid_order_id"`
	BidTime       int64           `json:"bid_time"`
	BidExpireTime int64           `json:"bid_expire_time"`
	BidPrice      decimal.Decimal `json:"bid_price"`
	BidSalt       int64           `json:"bid_salt"`
	BidMaker      string          `json:"bid_maker"`
	BidType       int64           `json:"bid_type"`
	BidSize       int64           `json:"bid_size"`
	BidUnfilled   int64           `json:"bid_unfilled"`
	FloorPrice    decimal.Decimal `json:"floor_price"`
}

// BidInfo 出价详情
type BidInfo struct {
	BidOrderID    string          `json:"bid_order_id"`
	BidTime       int64           `json:"bid_time"`
	BidExpireTime int64           `json:"bid_expire_time"`
	BidPrice      decimal.Decimal `json:"bid_price"`
	BidSalt       int64           `json:"bid_salt"`
	BidSize       int64           `json:"bid_size"`
	BidUnfilled   int64           `json:"bid_unfilled"`
}

type UserBidsResp struct {
	Count  int       `json:"count"`
	Result []UserBid `json:"result"`
}

// UserBid 用户出价信息
type UserBid struct {
	ChainID           int             `json:"chain_id"`
	CollectionAddress string          `json:"collection_address"`
	TokenID           string          `json:"token_id"`
	BidPrice          decimal.Decimal `json:"bid_price"`
	MarketplaceID     int             `json:"marketplace_id"`
	ExpireTime        int64           `json:"expire_time"`
	BidType           int64           `json:"bid_type"`
	CollectionName    string          `json:"collection_name"`
	ImageURI          string          `json:"image_uri"`
	OrderSize         int64           `json:"order_size"`
	BidInfos          []BidInfo       `json:"bid_infos"`
}

type MultichainCollection struct {
	CollectionAddress string `json:"collection_address"`
	Chain             string `json:"chain"`
}
