package svc

import (
	"github.com/ProjectsTask/EasySwapBase/evm/erc"
	//"github.com/ProjectsTask/EasySwapBase/image"
	"github.com/ProjectsTask/EasySwapBase/stores/xkv"
	"gorm.io/gorm"

	"github.com/ProjectsTask/EasySwapBackend/src/dao"
)

// CtxConfig 服务上下文配置构建器
// 用于使用 Option 模式构建 ServerCtx
type CtxConfig struct {
	db *gorm.DB
	//imageMgr image.ImageManager
	dao     *dao.Dao
	KvStore *xkv.Store
	Evm     erc.Erc
}

type CtxOption func(conf *CtxConfig)

// NewServerCtx 创建新的服务上下文
// 使用 Option 模式初始化 DB, KVStore, Dao 等组件
func NewServerCtx(options ...CtxOption) *ServerCtx {
	c := &CtxConfig{}
	for _, opt := range options {
		opt(c)
	}
	return &ServerCtx{
		DB: c.db,
		//ImageMgr: c.imageMgr,
		KvStore: c.KvStore,
		Dao:     c.dao,
	}
}

func WithKv(kv *xkv.Store) CtxOption {
	return func(conf *CtxConfig) {
		conf.KvStore = kv
	}
}

func WithDB(db *gorm.DB) CtxOption {
	return func(conf *CtxConfig) {
		conf.db = db
	}
}

func WithDao(dao *dao.Dao) CtxOption {
	return func(conf *CtxConfig) {
		conf.dao = dao
	}
}
