//go:build !windows

package app

import (
	"os"
	"os/signal"
	"syscall"
)

// signalStop 监听 SIGINT/SIGTERM 并调用 stop（AGENT.md 2.4 第二路径）。
// 返回 cleanup 函数，在应用退出时注销信号监听并回收 goroutine。
func signalStop(stop func()) func() {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		select {
		case <-ch:
			stop()
		case <-done:
		}
	}()
	return func() {
		signal.Stop(ch)
		close(done)
	}
}
