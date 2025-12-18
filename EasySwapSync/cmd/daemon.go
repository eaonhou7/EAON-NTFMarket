package cmd

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof" // 引入 pprof 用于性能分析
	"os"
	"os/signal"
	"sync"
	"syscall"

	"github.com/ProjectsTask/EasySwapBase/logger/xzap"
	"github.com/spf13/cobra"
	"go.uber.org/zap"

	"github.com/ProjectsTask/EasySwapSync/service"
	"github.com/ProjectsTask/EasySwapSync/service/config"
)

// DaemonCmd 定义了 "daemon" 子命令
var DaemonCmd = &cobra.Command{
	Use:   "daemon",
	Short: "sync easy swap order info.", // 命令简短描述：同步 EasySwap 订单信息
	Long:  "sync easy swap order info.", // 命令详细描述
	Run: func(cmd *cobra.Command, args []string) {
		// 使用 WaitGroup 等待所有 goroutine 完成
		wg := &sync.WaitGroup{}
		wg.Add(1)

		// 创建一个根 Context
		ctx := context.Background()
		// 创建一个带有取消功能的 Context，用于优雅退出
		ctx, cancel := context.WithCancel(ctx)

		// rpc退出信号通知chan，用于接收服务启动或运行过程中的错误
		onSyncExit := make(chan error, 1)

		// 启动一个 goroutine 来运行主服务逻辑
		go func() {
			defer wg.Done() // goroutine 结束时减少 WaitGroup 计数

			// 1. 读取和解析配置文件 (config.toml)
			cfg, err := config.UnmarshalCmdConfig()
			if err != nil {
				xzap.WithContext(ctx).Error("Failed to unmarshal config", zap.Error(err))
				onSyncExit <- err // 发送错误信号
				return
			}

			// 2. 初始化日志模块
			_, err = xzap.SetUp(*cfg.Log)
			if err != nil {
				xzap.WithContext(ctx).Error("Failed to set up logger", zap.Error(err))
				onSyncExit <- err
				return
			}

			// 打印服务启动日志
			xzap.WithContext(ctx).Info("sync server start", zap.Any("config", cfg))

			// 3. 初始化服务 (Service)
			// 这里会创建数据库连接、Redis 连接、链客户端等
			s, err := service.New(ctx, cfg)
			if err != nil {
				xzap.WithContext(ctx).Error("Failed to create sync server", zap.Error(err))
				onSyncExit <- err
				return
			}

			// 4. 启动服务
			// 开始同步区块事件
			if err := s.Start(); err != nil {
				xzap.WithContext(ctx).Error("Failed to start sync server", zap.Error(err))
				onSyncExit <- err
				return
			}

			// 5. 如果配置开启了 Pprof，启动 HTTP 服务进行性能监控
			if cfg.Monitor.PprofEnable {
				http.ListenAndServe(fmt.Sprintf("0.0.0.0:%d", cfg.Monitor.PprofPort), nil)
			}
		}()

		// 信号通知chan，用于接收系统信号
		onSignal := make(chan os.Signal)
		// 监听 SIGINT (Ctrl+C) 和 SIGTERM (kill) 信号，实现优雅退出
		signal.Notify(onSignal, syscall.SIGINT, syscall.SIGTERM)

		select {
		case sig := <-onSignal: // 收到系统信号
			switch sig {
			case syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM:
				cancel() // 取消 Context，通知所有子 goroutine 退出
				xzap.WithContext(ctx).Info("Exit by signal", zap.String("signal", sig.String()))
			}
		case err := <-onSyncExit: // 收到服务内部错误
			cancel() // 取消 Context
			xzap.WithContext(ctx).Error("Exit by error", zap.Error(err))
		}

		// 等待所有 goroutine 退出
		wg.Wait()
	},
}

func init() {
	// 将 daemon 命令添加到 root 命令中，使其可以被执行
	// 这样可以通过 go run main.go daemon 调用
	rootCmd.AddCommand(DaemonCmd)
}
