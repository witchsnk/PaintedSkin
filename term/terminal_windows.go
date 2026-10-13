//go:build windows

package term

import (
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"PaintedSkin/event"
	"golang.org/x/sys/windows"
)

// Windows 控制台模式位。
const (
	enableVirtualTerminalInput  = 0x0200
	enableVirtualTerminalOutput = 0x0004
	enableProcessedInput        = 0x0001
	enableLineInput             = 0x0002
	enableEchoInput             = 0x0004
	enableWindowInput           = 0x0008
)

// windowsTerminal 是 Windows 实现：开启 VT console mode 后输入即为标准
// VT 序列（复用与 Unix 相同的 input 解码器）；无 SIGWINCH，用轮询
// GetConsoleScreenBufferInfo 产生 resize（docs/03）。
type windowsTerminal struct {
	base
	in       *os.File
	events   chan event.Event
	stop     chan struct{}
	stopOnce sync.Once
	closer   func()
}

// New 打开控制台并开启 VT 模式。
func New(opts Options) (Terminal, error) {
	in := os.Stdin
	out := os.Stdout

	var closer func()
	origIn, origOut, err := consoleModes(windows.Handle(in.Fd()), windows.Handle(out.Fd()))
	if err != nil {
		// 管道重定向时尝试显式打开 CONIN$ / CONOUT$
		conin, errIn := os.OpenFile("CONIN$", os.O_RDWR, 0)
		conout, errOut := os.OpenFile("CONOUT$", os.O_RDWR, 0)
		if errIn == nil && errOut == nil {
			origIn, origOut, err = consoleModes(windows.Handle(conin.Fd()), windows.Handle(conout.Fd()))
			if err == nil {
				in = conin
				out = conout
				closer = func() {
					_ = conin.Close()
					_ = conout.Close()
				}
			} else {
				_ = conin.Close()
				_ = conout.Close()
			}
		}
		if err != nil {
			return nil, fmt.Errorf("term: get console mode: %w", err)
		}
	}

	t := &windowsTerminal{
		in:     in,
		events: make(chan event.Event, 256),
		stop:   make(chan struct{}),
		closer: closer,
	}
	t.base = newBase(out, opts, querySize(windows.Handle(out.Fd())), capsFromEnv(envSnapshot(), opts))
	t.base.restore = func() error {
		if err := windows.SetConsoleMode(windows.Handle(in.Fd()), origIn); err != nil {
			return err
		}
		return windows.SetConsoleMode(windows.Handle(out.Fd()), origOut)
	}

	rawIn := (origIn &^ (enableLineInput | enableEchoInput | enableProcessedInput | enableWindowInput)) | enableVirtualTerminalInput
	rawOut := origOut | enableVirtualTerminalOutput
	if err := windows.SetConsoleMode(windows.Handle(in.Fd()), rawIn); err != nil {
		if closer != nil {
			closer()
		}
		return nil, fmt.Errorf("term: enable VT input: %w", err)
	}
	if err := windows.SetConsoleMode(windows.Handle(out.Fd()), rawOut); err != nil {
		if closer != nil {
			closer()
		}
		return nil, fmt.Errorf("term: enable VT output: %w", err)
	}
	Enter(out, opts)
	t.watchResize(windows.Handle(out.Fd()))
	return t, nil
}

// Events 返回事件通道。
func (t *windowsTerminal) Events() <-chan event.Event { return t.events }

// Input 返回原始输入流（console），交由 input.Decoder 解码。
func (t *windowsTerminal) Input() io.Reader { return t.in }

// watchResize 轮询控制台尺寸，变化时投递 resize 事件。
func (t *windowsTerminal) watchResize(h windows.Handle) {
	go func() {
		ticker := time.NewTicker(300 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-t.stop:
				return
			case <-ticker.C:
				sz := querySize(h)
				if sz.W <= 0 || sz.H <= 0 {
					continue
				}
				t.mu.Lock()
				changed := sz != t.size
				t.size = sz
				t.mu.Unlock()
				if !changed {
					continue
				}
				select {
				case t.events <- event.Resize{W: sz.W, H: sz.H}:
				default:
				}
			}
		}
	}()
}

// Close 停止轮询并走共享恢复路径（幂等）。
func (t *windowsTerminal) Close() error {
	t.stopOnce.Do(func() {
		close(t.stop)
	})
	err := t.base.Close()
	if t.closer != nil {
		t.closer()
	}
	return err
}

// consoleModes 读取 stdin/stdout 当前 console mode。
func consoleModes(in, out windows.Handle) (uint32, uint32, error) {
	var mi, mo uint32
	if err := windows.GetConsoleMode(in, &mi); err != nil {
		return 0, 0, err
	}
	if err := windows.GetConsoleMode(out, &mo); err != nil {
		return 0, 0, err
	}
	return mi, mo, nil
}

// setConsoleMode 设置 console mode；out 为 true 时追加 ENABLE_PROCESSED_OUTPUT。
func setConsoleMode(h windows.Handle, mode uint32, out bool) error {
	if out {
		mode |= enableProcessedInput
	}
	return windows.SetConsoleMode(h, mode)
}

// querySize 用 GetConsoleScreenBufferInfo 取可见窗口尺寸。
func querySize(h windows.Handle) Size {
	var info windows.ConsoleScreenBufferInfo
	if err := windows.GetConsoleScreenBufferInfo(h, &info); err != nil {
		return Size{W: 80, H: 24}
	}
	w := int(info.Window.Right - info.Window.Left + 1)
	hgt := int(info.Window.Bottom - info.Window.Top + 1)
	if w <= 0 || hgt <= 0 {
		return Size{W: 80, H: 24}
	}
	return Size{W: w, H: hgt}
}
