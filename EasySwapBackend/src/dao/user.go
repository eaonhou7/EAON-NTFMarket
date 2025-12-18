package dao

import (
	"context"

	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/base"
	"github.com/ProjectsTask/EasySwapBase/stores/gdb/orderbookmodel/multi"
	"github.com/pkg/errors"
)

// GetUserSigStatus 获取用户的签名状态
// 功能: 查询用户表中是否存在该地址, 并返回其 IsSigned 状态字段
// 用途: 前端调用此接口判断用户是否需要进行 "Login with Wallet" 的签名操作 (首次登录或Session过期)
func (d *Dao) GetUserSigStatus(ctx context.Context, userAddr string) (bool, error) {
	var userInfo base.User
	// SQL 逻辑:
	// SELECT * FROM user_table WHERE address = ?
	db := d.DB.WithContext(ctx).Table(base.UserTableName()).
		Where("address = ?", userAddr).
		Find(&userInfo)
	if db.Error != nil {
		return false, errors.Wrap(db.Error, "failed on get user info")
	}

	// 如果找不到用户 (userInfo.Id == 0), 这里的 userInfo.IsSigned 默认为 false
	// 这符合业务逻辑: 新用户未签名
	return userInfo.IsSigned, nil
}

// QueryUserBids 查询用户的出价订单信息 (Listing & Offers)
// 功能: "My Offers" 页面, 展示用户发出的所有有效出价
func (d *Dao) QueryUserBids(ctx context.Context, chain string, userAddrs []string, contractAddrs []string) ([]multi.Order, error) {
	var userBids []multi.Order

	// SQL 逻辑:
	// 1. SELECT collection, token_id, price, ... FROM order_table
	// 2. WHERE maker IN (?)          -- 用户是相关出价人
	// 3. AND order_type IN (2, 3)    -- 类型为 ItemBid(2) 或 CollectionBid(3)
	// 4. AND order_status = 1        -- 状态为 Active
	// 5. AND quantity_remaining > 0  -- 还有剩余成交量
	// 6. [Optional] AND collection_address IN (?) -- 按集合筛选
	db := d.DB.WithContext(ctx).
		Table(multi.OrderTableName(chain)).
		Select("collection_address, token_id, order_id, token_id,order_type,"+
			"quantity_remaining, size, event_time, price, salt, expire_time").
		Where("maker in (?) and order_type in (?,?) and order_status = ? and quantity_remaining > 0",
			userAddrs, multi.ItemBidOrder, multi.CollectionBidOrder, multi.OrderStatusActive)

	// 如果指定了 Contract Address 列表 (通常是侧边栏筛选), 则增加过滤条件
	if len(contractAddrs) != 0 {
		db.Where("collection_address in (?)", contractAddrs)
	}

	if err := db.Scan(&userBids).Error; err != nil {
		return nil, errors.Wrap(err, "failed on get user bids")
	}

	return userBids, nil
}
