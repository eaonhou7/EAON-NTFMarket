package router

import (
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/ProjectsTask/EasySwapBackend/src/api/middleware"

	"github.com/ProjectsTask/EasySwapBackend/src/service/svc"
)

func NewRouter(svcCtx *svc.ServerCtx) *gin.Engine {
	// 强制控制台颜色输出，使日志更易读
	gin.ForceConsoleColor()
	// 设置 Gin 为发布模式 (ReleaseMode)
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()                        // 新建一个gin引擎实例
	r.Use(middleware.RecoverMiddleware()) // 使用自定义的恢复中间件，处理 Panic
	r.Use(middleware.RLog())              // 使用请求日志中间件，记录API访问日志

	r.Use(cors.New(cors.Config{ // 使用cors中间件，配置跨域访问策略
		AllowAllOrigins:  true,                                                         // 允许所有源
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"}, // 允许的方法
		AllowHeaders:     []string{"Origin", "Content-Length", "Content-Type", "X-CSRF-Token", "Authorization", "AccessToken", "Token"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type", "Access-Control-Allow-Origin", "Access-Control-Allow-Headers", "X-GW-Error-Code", "X-GW-Error-Message"},
		AllowCredentials: true,
		MaxAge:           1 * time.Hour,
	}))
	loadV1(r, svcCtx) // 加载 v1 版本的路由分组
	// loadV2(r, svcCtx) // 预留 v2 路由入口

	return r
}
