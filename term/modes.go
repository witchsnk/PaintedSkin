package term

import (
	"io"
	"strings"
	"sync"
)

// Options 控制进入时开启哪些终端模式（docs/03）。
type Options struct {
	// Mouse 开启 ?1002/?1003/?1006（按键跟踪 + 任意移动 + SGR 扩展）。
	Mouse bool
	// Paste 开启 ?2004 括号粘贴。
	Paste bool
	// FocusReports 开启 ?1004 焦点事件。
	FocusReports bool
	// AltScreen 开启 ?1049 备用屏；为 false 时直接在主屏绘制（调试用）。
	AltScreen bool
}

// withDefaults 在三态都未设置时给出安全默认。
// 调用方显式置 false 时不覆盖——故用指针语义由 New 内部处理，
// 此处仅供平台层复用。
func (o Options) altScreen() bool { return o.AltScreen }

// Enter 按顺序写入进入序列（备用屏 → 隐藏光标 → 鼠标 → 粘贴 → 焦点）。
func Enter(w io.Writer, o Options) {
	var sb strings.Builder
	if o.AltScreen {
		sb.WriteString("\x1b[?1049h")
	}
	sb.WriteString("\x1b[?25l")
	if o.Mouse {
		sb.WriteString("\x1b[?1002h")
		sb.WriteString("\x1b[?1003h")
		sb.WriteString("\x1b[?1006h")
	}
	if o.Paste {
		sb.WriteString("\x1b[?2004h")
	}
	if o.FocusReports {
		sb.WriteString("\x1b[?1004h")
	}
	_, _ = io.WriteString(w, sb.String())
}

// Leave 按逆序写入恢复序列；备用屏退出必须最后（docs/02）。
func Leave(w io.Writer, o Options) {
	var sb strings.Builder
	if o.Mouse {
		sb.WriteString("\x1b[?1006l")
		sb.WriteString("\x1b[?1003l")
		sb.WriteString("\x1b[?1002l")
	}
	if o.Paste {
		sb.WriteString("\x1b[?2004l")
	}
	if o.FocusReports {
		sb.WriteString("\x1b[?1004l")
	}
	sb.WriteString("\x1b[?25h") // 显示光标
	if o.AltScreen {
		sb.WriteString("\x1b[?1049l") // 退出备用屏放最后
	}
	_, _ = io.WriteString(w, sb.String())
}

// ShowCursor 显示硬件光标。
func ShowCursor(w io.Writer) { _, _ = io.WriteString(w, "\x1b[?25h") }

// HideCursor 隐藏硬件光标。
func HideCursor(w io.Writer) { _, _ = io.WriteString(w, "\x1b[?25l") }

// ColorDepth 按环境变量判定色彩深度（docs/03 降级表）。
func ColorDepth(env map[string]string) string {
	termName := env["TERM"]
	if strings.Contains(termName, "truecolor") || strings.Contains(termName, "24bit") {
		return "truecolor"
	}
	if env["COLORTERM"] == "truecolor" || env["COLORTERM"] == "24bit" {
		return "truecolor"
	}
	if strings.Contains(termName, "256") {
		return "256"
	}
	return "16"
}

// HasMouse 按环境变量判定鼠标能力；TERM=dumb 或未设 TERM 时禁用。
func HasMouse(env map[string]string) bool {
	termName := env["TERM"]
	if termName == "" || termName == "dumb" {
		return false
	}
	return true
}

// base 是各平台终端的共享状态：输出目标、模式开关、尺寸与能力，
// 以及幂等的 Close 恢复逻辑（AGENT.md 2.4 的单一恢复路径）。
type base struct {
	opts Options
	out  io.Writer
	size Size
	caps Caps

	// restore 由平台层提供：还原 raw mode / console mode 等 OS 级状态。
	// 为 nil 时 Close 只写 ANSI 序列。
	restore func() error

	mu        sync.Mutex
	closeOnce sync.Once
	closed    bool
	err       error
}

func newBase(out io.Writer, o Options, size Size, caps Caps) base {
	return base{opts: o, out: out, size: size, caps: caps}
}

// Writer 返回原始输出 writer。
func (b *base) Writer() io.Writer { return b.out }

// Size 返回当前终端尺寸。
func (b *base) Size() Size {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.size
}

// Capabilities 返回终端能力。
func (b *base) Capabilities() Caps {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.caps
}

// Closed 报告是否已关闭。
func (b *base) Closed() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.closed
}

// Close 写恢复序列并还原 OS 级状态；幂等（sync.Once）。
func (b *base) Close() error {
	b.closeOnce.Do(func() {
		Leave(b.out, b.opts)
		if b.restore != nil {
			b.err = b.restore()
		}
		b.mu.Lock()
		b.closed = true
		b.mu.Unlock()
	})
	return b.err
}
