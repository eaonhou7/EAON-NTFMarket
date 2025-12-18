package dao

import (
	"fmt"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/shopspring/decimal"

	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/multi"
)

// CollectionTrade 集合交易统计信息
// 用于返回排行榜或统计页面的数据
type CollectionTrade struct {
	ContractAddress string          `json:"contract_address"` // 集合合约地址
	ItemCount       int64           `json:"item_count"`       // 交易发生次数(成交单数)
	Volume          decimal.Decimal `json:"volume"`           // 总交易额 (Volume)
	VolumeChange    int             `json:"volume_change"`    // 交易额变化率 (相对于上一周期, 百分比整数)
	PreFloorPrice   decimal.Decimal `json:"pre_floor_price"`  // 上一周期地板价 (用于计算变化)
	FloorChange     int             `json:"floor_change"`     // 地板价变化率 (相对于上一周期, 百分比整数)
}

// GenRankingKey 生成排行榜缓存 Key
// Key 格式: cache:project:chain:ranking:volume:period
func GenRankingKey(project, chain string, period int) string {
	return fmt.Sprintf("cache:%s:%s:ranking:volume:%d", strings.ToLower(project), strings.ToLower(chain), period)
}

// EpochUnit 定义时间周期的基本单位 (5分钟)
// 这里的 periodToEpoch 值是基于 5 分钟为一个 epoch 计算的
const EpochUnit = 5 * time.Minute

type periodEpochMap map[string]int

// periodToEpoch 时间段映射表
// Key: 时间段字符串 (如 "1d", "24h")
// Value: Epoch 数量 (每个 Epoch 为 5 分钟)
var periodToEpoch = periodEpochMap{
	"15m": 3,    // 3 * 5min = 15m
	"1h":  12,   // 12 * 5min = 60m
	"6h":  72,   // 72 * 5min = 6h
	"24h": 288,  // 288 * 5min = 1440m = 24h
	"1d":  288,  // 24h
	"7d":  2016, // 2016 * 5min = 7d
	"30d": 8640, // 8640 * 5min = 30d
}

// GetTradeInfoByCollection 获取指定集合在特定时间段内的交易统计信息
// 功能: 统计 Volume, Floor Price 及其涨跌幅
func (d *Dao) GetTradeInfoByCollection(chain, collectionAddr, period string) (*CollectionTrade, error) {
	// 查询当前时间段的交易信息
	var tradeCount int64
	var totalVolume decimal.Decimal
	var floorPrice decimal.Decimal

	// 1. 获取时间段对应的 Epoch 数量
	epoch, ok := periodToEpoch[period]
	if !ok {
		return nil, errors.Errorf("invalid period: %s", period)
	}
	// 2. 计算查询的时间范围 [Now - Period, Now]
	// 修正: 乘以 EpochUnit (5分钟) 以获取正确的总时长
	duration := time.Duration(epoch) * EpochUnit
	endTime := time.Now()
	startTime := endTime.Add(-duration)

	// 3. 统计当前时间段内的 交易数量 和 总交易额
	// ActivityType = Sale (仅统计成交)
	err := d.DB.WithContext(d.ctx).Table(multi.ActivityTableName(chain)).
		Where("collection_address = ? AND activity_type = ? AND event_time >= ? AND event_time <= ?",
			collectionAddr, multi.Sale, startTime, endTime).
		Select("COUNT(*) as trade_count, COALESCE(SUM(price), 0) as total_volume").
		Row().Scan(&tradeCount, &totalVolume)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get trade count and volume")
	}

	// 4. 获取当前时间段内对应的 地板价 (最低成交价)
	// 注意: 这里的 Floor Price 是取该时间段内的 Min(Price), 而不是当前瞬时 Floor
	err = d.DB.WithContext(d.ctx).Table(multi.ActivityTableName(chain)).
		Where("collection_address = ? AND activity_type = ? AND event_time >= ? AND event_time <= ?",
			collectionAddr, multi.Sale, startTime, endTime).
		Select("COALESCE(MIN(price), 0)").
		Row().Scan(&floorPrice)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get floor price")
	}

	// 5. 计算上一周期的时间范围 [CurrentStart - Period, CurrentStart]
	// 用于计算环比变化 (Change %)
	prevStartTime := startTime.Add(-duration)
	prevEndTime := startTime

	var prevVolume decimal.Decimal
	var prevFloorPrice decimal.Decimal

	// 6. 获取上一周期的 总交易额
	err = d.DB.WithContext(d.ctx).Table(multi.ActivityTableName(chain)).
		Where("collection_address = ? AND activity_type = ? AND event_time >= ? AND event_time <= ?",
			collectionAddr, multi.Sale, prevStartTime, prevEndTime).
		Select("COALESCE(SUM(price), 0)").
		Row().Scan(&prevVolume)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get previous volume")
	}

	// 7. 获取上一周期的 地板价
	err = d.DB.WithContext(d.ctx).Table(multi.ActivityTableName(chain)).
		Where("collection_address = ? AND activity_type = ? AND event_time >= ? AND event_time <= ?",
			collectionAddr, multi.Sale, prevStartTime, prevEndTime).
		Select("COALESCE(MIN(price), 0)").
		Row().Scan(&prevFloorPrice)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get previous floor price")
	}

	// 8. 计算交易额和地板价的环比变化百分比
	volumeChange := 0
	floorChange := 0

	// (Current - Prev) / Prev * 100
	if !prevVolume.IsZero() {
		volumeChangeDecimal := totalVolume.Sub(prevVolume).Div(prevVolume).Mul(decimal.NewFromInt(100))
		volumeChange = int(volumeChangeDecimal.IntPart())
	}

	if !prevFloorPrice.IsZero() {
		floorChangeDecimal := floorPrice.Sub(prevFloorPrice).Div(prevFloorPrice).Mul(decimal.NewFromInt(100))
		floorChange = int(floorChangeDecimal.IntPart())
	}

	return &CollectionTrade{
		ContractAddress: collectionAddr,
		ItemCount:       tradeCount,
		Volume:          totalVolume,
		VolumeChange:    volumeChange,
		PreFloorPrice:   prevFloorPrice,
		FloorChange:     floorChange,
	}, nil
}

// GetCollectionRankingByActivity 获取基于交易活动的集合排行榜信息
// 功能: 批量计算所有集合在指定时间段内的 Volume, Floor Price 及其排名数据
func (d *Dao) GetCollectionRankingByActivity(chain, period string) ([]*CollectionTrade, error) {
	// 1. 获取时间段对应的 Epoch
	epoch, ok := periodToEpoch[period]
	if !ok {
		return nil, errors.Errorf("invalid period: %s", period)
	}

	// 2. 计算当前和上一周期的时间范围
	// 修正: 考虑 EpochUnit (5分钟)
	duration := time.Duration(epoch) * EpochUnit
	endTime := time.Now()
	startTime := endTime.Add(-duration)

	prevEndTime := startTime
	prevStartTime := startTime.Add(-duration)

	// 定义中间结果结构体
	type TradeStats struct {
		CollectionAddress string
		ItemCount         int64
		Volume            decimal.Decimal
		FloorPrice        decimal.Decimal
	}

	// 3. 聚合查询 当前周期 统计数据
	// Group By CollectionAddress
	var currentStats []TradeStats
	err := d.DB.WithContext(d.ctx).Table(multi.ActivityTableName(chain)).
		Select("collection_address, COUNT(*) as item_count, COALESCE(SUM(price), 0) as volume, COALESCE(MIN(price), 0) as floor_price").
		Where("activity_type = ? AND event_time >= ? AND event_time <= ?", multi.Sale, startTime, endTime).
		Group("collection_address").
		Find(&currentStats).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed to get current stats")
	}

	// 4. 聚合查询 上一周期 统计数据
	var prevStats []TradeStats
	err = d.DB.WithContext(d.ctx).Table(multi.ActivityTableName(chain)).
		Select("collection_address, COUNT(*) as item_count, COALESCE(SUM(price), 0) as volume, COALESCE(MIN(price), 0) as floor_price").
		Where("activity_type = ? AND event_time >= ? AND event_time <= ?", multi.Sale, prevStartTime, prevEndTime).
		Group("collection_address").
		Find(&prevStats).Error
	if err != nil {
		return nil, errors.Wrap(err, "failed to get previous stats")
	}

	// 5. 构建上一周期的 Map 索引, 方便快速查找
	prevStatsMap := make(map[string]TradeStats)
	for _, stat := range prevStats {
		prevStatsMap[stat.CollectionAddress] = stat
	}

	// 6. 组装最终结果, 计算变化率
	var result []*CollectionTrade
	for _, curr := range currentStats {
		trade := &CollectionTrade{
			ContractAddress: curr.CollectionAddress,
			ItemCount:       curr.ItemCount,
			Volume:          curr.Volume,
			VolumeChange:    0,
			PreFloorPrice:   decimal.Zero,
			FloorChange:     0,
		}

		if prev, ok := prevStatsMap[curr.CollectionAddress]; ok {
			trade.PreFloorPrice = prev.FloorPrice

			// Volume Change %
			if !prev.Volume.IsZero() {
				volumeChangeDecimal := curr.Volume.Sub(prev.Volume).Div(prev.Volume).Mul(decimal.NewFromInt(100))
				trade.VolumeChange = int(volumeChangeDecimal.IntPart())
			}

			// Floor Price Change %
			if !prev.FloorPrice.IsZero() {
				floorChangeDecimal := curr.FloorPrice.Sub(prev.FloorPrice).Div(prev.FloorPrice).Mul(decimal.NewFromInt(100))
				trade.FloorChange = int(floorChangeDecimal.IntPart())
			}
		}

		result = append(result, trade)
	}

	return result, nil
}

// GetCollectionVolume 获取指定 Collection 的历史总交易额
func (d *Dao) GetCollectionVolume(chain, collectionAddr string) (decimal.Decimal, error) {
	var volume decimal.Decimal
	err := d.DB.WithContext(d.ctx).Table(multi.ActivityTableName(chain)).
		Where("collection_address = ? AND activity_type = ?", collectionAddr, multi.Sale).
		Select("COALESCE(SUM(price), 0)").
		Row().Scan(&volume)
	if err != nil {
		return decimal.Zero, errors.Wrap(err, "failed to get collection volume")
	}

	return volume, nil
}
