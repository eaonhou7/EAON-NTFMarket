package service

import (
	"context"
	"fmt"
	"sync"

	"github.com/ProjectsTask/EasySwapBase/chain"
	"github.com/ProjectsTask/EasySwapBase/chain/chainclient"
	"github.com/ProjectsTask/EasySwapBase/ordermanager"
	"github.com/ProjectsTask/EasySwapBase/stores/xkv"
	"github.com/pkg/errors"
	"github.com/zeromicro/go-zero/core/stores/cache"
	"github.com/zeromicro/go-zero/core/stores/kv"
	"github.com/zeromicro/go-zero/core/stores/redis"
	"gorm.io/gorm"

	"github.com/ProjectsTask/EasySwapSync/service/orderbookindexer"

	"github.com/ProjectsTask/EasySwapSync/model"
	"github.com/ProjectsTask/EasySwapSync/service/collectionfilter"
	"github.com/ProjectsTask/EasySwapSync/service/config"
)

// Service 结构体定义了后台服务的核心组件
type Service struct {
	ctx              context.Context
	config           *config.Config
	kvStore          *xkv.Store // KV存储 (Redis)
	db               *gorm.DB   // 数据库 (MySQL)
	wg               *sync.WaitGroup
	collectionFilter *collectionfilter.Filter   // 集合过滤器，用于管理允许的 NFT 集合
	orderbookIndexer *orderbookindexer.Service  // 订单簿索引器，核心业务逻辑，负责同步链上事件
	orderManager     *ordermanager.OrderManager // 订单管理器，负责订单的验证和管理
}

// New 初始化一个新的 Service 实例
func New(ctx context.Context, cfg *config.Config) (*Service, error) {
	// 1. 初始化 Redis 配置
	var kvConf kv.KvConf
	for _, con := range cfg.Kv.Redis {
		kvConf = append(kvConf, cache.NodeConf{
			RedisConf: redis.RedisConf{
				Host: con.Host,
				Type: con.Type,
				Pass: con.Pass,
			},
			Weight: 2,
		})
	}

	// 创建 KVStore 实例
	kvStore := xkv.NewStore(kvConf)

	var err error
	// 2. 初始化数据库 (GORM)
	db := model.NewDB(cfg.DB)

	// 3. 初始化集合过滤器
	collectionFilter := collectionfilter.New(ctx, db, cfg.ChainCfg.Name, cfg.ProjectCfg.Name)

	// 4. 初始化订单管理器
	orderManager := ordermanager.New(ctx, db, kvStore, cfg.ChainCfg.Name, cfg.ProjectCfg.Name)

	var orderbookSyncer *orderbookindexer.Service
	var chainClient chainclient.ChainClient
	fmt.Println("chainClient url:" + cfg.AnkrCfg.HttpsUrl + cfg.AnkrCfg.ApiKey)

	// 5. 初始化链客户端 (EVM client)
	chainClient, err = chainclient.New(int(cfg.ChainCfg.ID), cfg.AnkrCfg.HttpsUrl+cfg.AnkrCfg.ApiKey)
	if err != nil {
		return nil, errors.Wrap(err, "failed on create evm client")
	}

	// 6. 根据链 ID 初始化对应的 OrderBookIndexer
	switch cfg.ChainCfg.ID {
	case chain.EthChainID, chain.OptimismChainID, chain.SepoliaChainID:
		// 初始化订单簿索引服务，传入数据库、KV存储、链客户端等依赖
		orderbookSyncer = orderbookindexer.New(ctx, cfg, db, kvStore, chainClient, cfg.ChainCfg.ID, cfg.ChainCfg.Name, orderManager)
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed on create trade info server")
	}

	// 构造 Service 对象
	manager := Service{
		ctx:              ctx,
		config:           cfg,
		db:               db,
		kvStore:          kvStore,
		collectionFilter: collectionFilter,
		orderbookIndexer: orderbookSyncer,
		orderManager:     orderManager,
		wg:               &sync.WaitGroup{},
	}
	return &manager, nil
}

// Start 启动服务
func (s *Service) Start() error {
	// 不要移动位置
	// 1. 预加载 NFT 集合信息到过滤器中
	if err := s.collectionFilter.PreloadCollections(); err != nil {
		return errors.Wrap(err, "failed on preload collection to filter")
	}

	// 2. 启动订单簿索引服务 (异步运行)
	// 这里会启动 goroutine 监听和处理链上事件
	s.orderbookIndexer.Start()

	// 3. 启动订单管理器
	s.orderManager.Start()
	return nil
}
