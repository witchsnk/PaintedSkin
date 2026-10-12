//go:build darwin || linux

package term

import (
	"fmt"
	"io"
	"os"
	"os/signal"
	"sync"
	"syscall"

	"PaintedSkin/event"
	"golang.org/x/sys/unix"
)

// unixTerminal 是 Unix 平台的终端实现：termios raw mode + ANSI 模式 +
// SIGWINCH 产生 resize 事件（docs/03）。
type unixTerminal struct {
	base
	in       *os.File
	events   chan event.Event
	stop     chan struct{}
	stopOnce sync.Once
	sigs     chan os.Signal
}

// New 打开当前终端，进入 raw mode 并开启备用屏等模式。
func New(opts Options) (Terminal, error) {
	in := os.Stdin
	out := os.Stdout
	fd := int(in.Fd())

	orig, err := unix.IoctlGetTermios(fd, ioctlRead)
	if err != nil {
		return nil, fmt.Errorf("term: get termios: %w", err)
	}

	t := &unixTerminal{
		in:     in,
		events: make(chan event.Event, 256),
		stop:   make(chan struct{}),
	}
	t.base = newBase(out, opts, querySize(fd), capsFromEnv(envSnapshot(), opts))
	t.base.restore = func() error { return unix.IoctlSetTermios(fd, ioctlWrite, orig) }

	if err := setRaw(fd, orig); err != nil {
		return nil, fmt.Errorf("term: set raw mode: %w", err)
	}
	Enter(out, opts)
	t.watchResize(fd)
	return t, nil
}

// Events 返回事件通道（resize 由后端产生）。
func (t *unixTerminal) Events() <-chan event.Event { return t.events }

// Input 返回原始输入流（tty），交由 input.Decoder 解码。
func (t *unixTerminal) Input() io.Reader { return t.in }

// watchResize 监听 SIGWINCH，尺寸变化时投递 resize 事件。
func (t *unixTerminal) watchResize(fd int) {
	t.sigs = make(chan os.Signal, 1)
	signal.Notify(t.sigs, syscall.SIGWINCH)
	go func() {
		for {
			select {
			case <-t.stop:
				signal.Stop(t.sigs)
				return
			case <-t.sigs:
				sz := querySize(fd)
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
				default: // 满则丢弃，UI 线程下一帧会重新取尺寸
				}
			}
		}
	}()
}

// Close 停止信号监听并走共享恢复路径（幂等）。
func (t *unixTerminal) Close() error {
	t.stopOnce.Do(func() {
		close(t.stop)
	})
	return t.base.Close()
}

// setRaw 清掉 ECHO|ICANON|ISIG|IEXTEN|IXON 等标志，置 VMIN=1 VTIME=0。
func setRaw(fd int, orig *unix.Termios) error {
	raw := *orig
	raw.Lflag &^= unix.ECHO | unix.ICANON | unix.ISIG | unix.IEXTEN
	raw.Iflag &^= unix.IXON | unix.ICRNL | unix.BRKINT | unix.INPCK | unix.ISTRIP
	raw.Cc[unix.VMIN] = 1
	raw.Cc[unix.VTIME] = 0
	return unix.IoctlSetTermios(fd, ioctlWrite, &raw)
}

// querySize 用 TIOCGWINSZ 取当前行列数。
func querySize(fd int) Size {
	ws, err := unix.IoctlGetWinsize(fd, unix.TIOCGWINSZ)
	if err != nil || ws.Col == 0 || ws.Row == 0 {
		return Size{W: 80, H: 24}
	}
	return Size{W: int(ws.Col), H: int(ws.Row)}
}
