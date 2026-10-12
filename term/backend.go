// Package term 封装"把字节送进真实终端、把终端能力打开/还原"的全部平台差异，
// 对上层只暴露 Terminal 接口（docs/03）。
package term

import (
	"os"

	"PaintedSkin/event"
)

// envSnapshot 读取当前进程环境（便于测试注入）。
func envSnapshot() map[string]string {
	return map[string]string{
		"TERM":      os.Getenv("TERM"),
		"COLORTERM": os.Getenv("COLORTERM"),
	}
}

// capsFromEnv 按环境变量探测能力（docs/03 能力探测与降级）。
func capsFromEnv(env map[string]string, opts Options) Caps {
	mouse := HasMouse(env)
	return Caps{
		ColorDepth:      ColorDepth(env),
		HasMouse:        mouse && opts.Mouse,
		HasFocusReports: opts.FocusReports,
	}
}

// eventsChan 是后端共享的事件通道构造：带缓冲，非阻塞投递。
func eventsChan() (chan event.Event, func(event.Event)) {
	ch := make(chan event.Event, 256)
	send := func(e event.Event) {
		defer func() {
			// 通道满时丢弃而非阻塞 UI 线程（慢终端下宁可丢事件）。
			_ = recover()
		}()
		select {
		case ch <- e:
		default:
		}
	}
	return ch, send
}
