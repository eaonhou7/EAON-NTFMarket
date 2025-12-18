package dao

import (
	"context"

	"github.com/ProjectsTask/EasySwapBase/stores/xkv"
	"gorm.io/gorm"
)

// Dao 数据访问对象
// 封装了数据库 (GORM) 和 Redis (KvStore) 的操作
// 所有的数据库交互逻辑应在此层实现, 避免在 Service 层直接操作 DB
type Dao struct {
	ctx context.Context

	DB      *gorm.DB   // 关系型数据库连接实例 (MySQL/PostgreSQL)
	KvStore *xkv.Store // 键值存储实例 (Redis), 用于缓存
}

// New 创建一个新的 Dao 实例
// 参数:
//
//	ctx: 上下文
//	db: GORM DB 实例
//	kvStore: KV Store 实例
//
// 返回:
//
//	*Dao: 初始化的 Dao 指针
func New(ctx context.Context, db *gorm.DB, kvStore *xkv.Store) *Dao {
	return &Dao{
		ctx:     ctx,
		DB:      db,
		KvStore: kvStore,
	}
}
