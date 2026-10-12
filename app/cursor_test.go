package app

import (
	"bytes"
	"testing"

	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/widget"
)

// 硬件光标定位（docs/06 / docs/08）：TextInput 聚焦时把插入位换算成
// 屏幕坐标交给 Renderer；失焦隐藏。

// TextInput 聚焦 → 输出里出现光标定位序列。
func TestCursorShownWhenFocused(t *testing.T) {
	var out bytes.Buffer
	a := New(Config{Width: 30, Height: 5, Writer: &out})
	ti := widget.NewTextInput()
	ti.SetText("hello")
	a.Root().Add(ti)
	ti.Layout(geom.Rect{X: 2, Y: 1, W: 20, H: 1})
	a.Layout()
	a.Root().Focus(ti)
	a.Draw()

	if !bytes.Contains(out.Bytes(), []byte("\x1b[?25h")) {
		t.Errorf("focused TextInput must show hardware cursor: %q", out.String())
	}
	// 插入位在 "hello" 之后：边框 x=2，内容起 x=3，文本宽 5 → x=8 → 列 9；b.Y=1 → 行 2
	if !bytes.Contains(out.Bytes(), []byte("\x1b[2;9H")) {
		t.Errorf("cursor must be positioned at insertion point: %q", out.String())
	}
}

// 失焦 → 隐藏硬件光标。
func TestCursorHiddenWhenBlurred(t *testing.T) {
	var out bytes.Buffer
	a := New(Config{Width: 30, Height: 5, Writer: &out})
	ti := widget.NewTextInput()
	btn := widget.NewButton("B")
	a.Root().Add(ti)
	a.Root().Add(btn)
	ti.Layout(geom.Rect{X: 2, Y: 1, W: 20, H: 1})
	btn.Layout(geom.Rect{X: 2, Y: 3, W: 6, H: 1})
	a.Layout()
	a.Root().Focus(ti)
	out.Reset()
	a.Root().Focus(btn) // 焦点移走
	a.Draw()
	if !bytes.Contains(out.Bytes(), []byte("\x1b[?25l")) {
		t.Errorf("blur must hide hardware cursor: %q", out.String())
	}
}

// 焦点在非光标类控件 → 不请求光标。
func TestNoCursorWithoutFocusableCursor(t *testing.T) {
	var out bytes.Buffer
	a := New(Config{Width: 30, Height: 5, Writer: &out})
	btn := widget.NewButton("B")
	a.Root().Add(btn)
	btn.Layout(geom.Rect{X: 2, Y: 1, W: 6, H: 1})
	a.Layout()
	a.Root().Focus(btn)
	a.Draw()
	if bytes.Contains(out.Bytes(), []byte("\x1b[?25h")) {
		t.Errorf("non-cursor widget must not show hardware cursor: %q", out.String())
	}
}

// 光标位置随编辑移动。
func TestCursorMovesWithEditing(t *testing.T) {
	var out bytes.Buffer
	a := New(Config{Width: 30, Height: 5, Writer: &out})
	ti := widget.NewTextInput()
	a.Root().Add(ti)
	ti.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 1})
	a.Layout()
	a.Root().Focus(ti)
	ti.Handle(event.Key{Code: event.KeyRune, Rune: 'a', Press: true})
	out.Reset()
	a.Draw()
	// 内容起 x=1，"a" 宽 1，光标在其后 → x=2 → 列 3
	if !bytes.Contains(out.Bytes(), []byte("\x1b[1;3H")) {
		t.Errorf("cursor must follow editing: %q", out.String())
	}
}

// TextInput 满足 CursorRequester。
func TestTextInputIsCursorRequester(t *testing.T) {
	ti := widget.NewTextInput()
	var _ CursorRequester = ti
	_, _, ok := ti.CursorScreenPos()
	if ok {
		t.Error("unlaid-out input must report ok=false")
	}
}
