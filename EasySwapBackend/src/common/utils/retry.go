package utils

import (
	"fmt"
	"time"
)

// Retry 通用重试函数
// @param name: 操作名称(用于日志或错误提示)
// @param attempts: 最大重试次数
// @param sleep: 每次重试间隔时间
// @param fn: 需要执行的函数,返回 error 表示失败需要的重试
// @return error: 如果所有尝试都失败,返回"retry time over"错误
func Retry(name string, attempts int, sleep time.Duration, fn func() error) error {
	// 循环执行指定次数
	for i := 0; i < attempts; i++ {
		// 执行函数,如果无错误则直接返回成功
		if err := fn(); err == nil {
			return nil
		}
		// 如果有错误,等待指定时间后继续下一次尝试
		time.Sleep(sleep)
		continue
	}
	// 所有尝试都失败
	return fmt.Errorf("retry time over")
}
