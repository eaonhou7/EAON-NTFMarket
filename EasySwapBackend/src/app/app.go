package app

import (
	"context"

	"github.com/ProjectsTask/EasySwapBase/logger/xzap"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/ProjectsTask/EasySwapBackend/src/config"
	"github.com/ProjectsTask/EasySwapBackend/src/service/svc"
)

// Platform 平台结构体，作为整个应用程序的容器
type Platform struct {
	config    *config.Config
	router    *gin.Engine
	serverCtx *svc.ServerCtx
}

// NewPlatform 创建一个新的 Platform 实例
func NewPlatform(config *config.Config, router *gin.Engine, serverCtx *svc.ServerCtx) (*Platform, error) {
	return &Platform{
		config:    config,
		router:    router,
		serverCtx: serverCtx,
	}, nil
}

// Start 启动平台服务
// 这是一个阻塞调用，会启动 HTTP 服务器监听指定端口
func (p *Platform) Start() {
	xzap.WithContext(context.Background()).Info("EasySwap-End run", zap.String("port", p.config.Api.Port))
	if err := p.router.Run(p.config.Api.Port); err != nil {
		panic(err)
	}
}
