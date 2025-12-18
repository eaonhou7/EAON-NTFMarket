package svc

import (
	"context"

	"github.com/ProjectsTask/EasySwapBase/chain/nftchainservice"
	"github.com/ProjectsTask/EasySwapBase/logger/xzap"
	"github.com/ProjectsTask/EasySwapBase/stores/gdb"
	"github.com/ProjectsTask/EasySwapBase/stores/xkv"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/kv"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/gorm"

	"github.com/ProjectsTask/EasySwapBackend/src/config"
	"github.com/ProjectsTask/EasySwapBackend/src/dao"
)

type ServerCtx struct {
	C  *config.Config
	DB *gorm.DB
	//ImageMgr image.ImageManager
	Dao      *dao.Dao
	KvStore  *xkv.Store
	RankKey  string
	NodeSrvs map[int64]*nftchainservice.Service
}

// NewServiceContext 初始化服务上下文
// 该函数负责初始化后端服务所需的所有基础设施组件
func NewServiceContext(c *config.Config) (*ServerCtx, error) {
	var err error
	//imageMgr, err = image.NewManager(c.ImageCfg)
	//if err != nil {
	//	return nil, errors.Wrap(err, "failed on create image manager")
	//}

	// Log
	// 1. 初始化日志系统 (Zap Logger)
	_, err = xzap.SetUp(c.Log)
	if err != nil {
		return nil, err
	}

	var kvConf kv.KvConf
	// 2. 构造 Redis 配置
	for _, con := range c.Kv.Redis {
		kvConf = append(kvConf, cache.NodeConf{
			RedisConf: redis.RedisConf{
				Host: con.Host,
				Type: con.Type,
				Pass: con.Pass,
			},
			Weight: 1,
		})
	}

	// redis
	// 3. 初始化 Redis 客户端 (xkv Store)
	store := xkv.NewStore(kvConf)
	// db
	// 4. 初始化数据库连接 (GORM)
	db, err := gdb.NewDB(&c.DB)
	if err != nil {
		return nil, err
	}

	// 5. 初始化多链节点服务 (Chain Services)
	// 遍历配置支持的每一条链，创建对应的链服务实例
	nodeSrvs := make(map[int64]*nftchainservice.Service)
	for _, supported := range c.ChainSupported {
		nodeSrvs[int64(supported.ChainID)], err = nftchainservice.New(context.Background(), supported.Endpoint, supported.Name, supported.ChainID,
			c.MetadataParse.NameTags, c.MetadataParse.ImageTags, c.MetadataParse.AttributesTags,
			c.MetadataParse.TraitNameTags, c.MetadataParse.TraitValueTags)

		if err != nil {
			return nil, errors.Wrap(err, "failed on start onchain sync service")
		}
	}

	// 6. 初始化数据访问层 (DAO)
	dao := dao.New(context.Background(), db, store)

	// 7. 组装 ServerCtx 对象
	serverCtx := NewServerCtx(
		WithDB(db),
		WithKv(store),
		//WithImageMgr(imageMgr),
		WithDao(dao),
	)
	serverCtx.C = c

	serverCtx.NodeSrvs = nodeSrvs

	return serverCtx, nil
}
