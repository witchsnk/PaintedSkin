package term

import (
	"io"

	"PaintedSkin/event"
)

// Terminal is the interface that wraps the basic terminal operations.
// It encapsulates opening/closing the terminal, reading events, getting size,
// and writing output. All platform-specific details are hidden behind this interface.
type Terminal interface {
	// Events returns a channel that receives terminal events such as resize,
	// focus changes, and mouse events. Events are produced by the backend
	// and consumed by the main loop.
	Events() <-chan event.Event

	// Input returns the raw input stream (tty / console) for the input
	// decoder to read from. 后端只产生 resize 等"非输入"事件；
	// 键盘与鼠标字节一律经此流交给 input.Decoder 统一解码。
	Input() io.Reader

	// Writer returns the raw output writer. screen.Renderer uses this to
	// write ANSI sequences directly to the terminal.
	Writer() io.Writer

	// Size returns the current terminal size in cells (rows, cols).
	Size() Size

	// Capabilities reports terminal capabilities such as color depth and
	// mouse support.
	Capabilities() Caps

	// Close terminates the terminal, restoring the original terminal state.
	// This operation is idempotent (safe to call multiple times).
	Close() error
}

// Size represents the terminal size in cells.
type Size struct {
	W int // width in cells (columns)
	H int // height in cells (rows)
}

// Caps reports terminal capabilities.
type Caps struct {
	// ColorDepth indicates the terminal's color capabilities.
	// Can be "16", "256", "truecolor", or "default".
	ColorDepth string

	// HasMouse reports whether the terminal supports mouse events.
	HasMouse bool

	// HasFocusReports reports whether focus gain/loss events are supported.
	HasFocusReports bool
}
