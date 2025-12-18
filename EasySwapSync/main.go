package main

import (
	"github.com/ProjectsTask/EasySwapSync/cmd"
)

// main 是程序的入口函数
// 当执行 go run main.go daemon 时，会从这里开始执行
func main() {
	// 调用 cmd 包的 Execute 方法，解析命令行参数并执行相应的命令
	// 在这里它会识别 "daemon" 参数并执行 cmd/daemon.go 中定义的 DaemonCmd
	cmd.Execute()
}
