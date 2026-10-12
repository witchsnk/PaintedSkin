//go:build !windows

package app

import (
	"os"
	"os/signal"
	"syscall"
)

// signalStop 监听 SIGINT/SIGTERM 并调用 stop（AGENT.md 2.4 第二路径）。
func signalStop(stop func()) {
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-ch
		stop()
	}()
}
