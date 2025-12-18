package main

import (
	"flag"
	_ "net/http/pprof"

	"github.com/ProjectsTask/EasySwapBackend/src/api/router"
	"github.com/ProjectsTask/EasySwapBackend/src/app"
	"github.com/ProjectsTask/EasySwapBackend/src/config"
	"github.com/ProjectsTask/EasySwapBackend/src/service/svc"
)

const (
	// port       = ":9000"
	repoRoot          = ""
	defaultConfigPath = "./config/config.toml"
)

func main() {
	// 解析命令行参数，默认为 ./config/config.toml
	conf := flag.String("conf", defaultConfigPath, "conf file path")
	flag.Parse()
	// 加载并解析配置文件
	c, err := config.UnmarshalConfig(*conf)
	if err != nil {
		panic(err)
	}

	// 验证配置中的链信息是否有效
	for _, chain := range c.ChainSupported {
		if chain.ChainID == 0 || chain.Name == "" {
			panic("invalid chain_suffix config")
		}
	}

	// 初始化服务上下文 (Context)，包含DB, Redis等连接
	serverCtx, err := svc.NewServiceContext(c)
	if err != nil {
		panic(err)
	}
	// Initialize router
	// 初始化 Gin 路由实例
	r := router.NewRouter(serverCtx)
	// 创建应用程序实例，并将路由和服务上下文注入
	app, err := app.NewPlatform(c, r, serverCtx)
	if err != nil {
		panic(err)
	}
	// 启动 HTTP 服务
	app.Start()
}
