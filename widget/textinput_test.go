package widget

import (
	"testing"

	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/widget/layout"
)

// Measure：固定高 1 行、宽按约束。
func TestTextInputMeasure(t *testing.T) {
	got := NewTextInput().Measure(layout.Constraints{MaxW: 30, MaxH: 5})
	if got.H != 1 || got.W != 30 {
		t.Errorf("constrained = %+v", got)
	}
	// 无上界时回落固定宽
	got2 := NewTextInput().Measure(layout.Constraints{MaxW: -1, MaxH: -1})
	if got2.H != 1 || got2.W != 17 {
		t.Errorf("unbounded = %+v want 17x1", got2)
	}
}

// 输入插入 / 退格 / Delete。
func TestTextInputEdit(t *testing.T) {
	ti := NewTextInput()
	ti.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 1})
	ti.OnFocus()
	typeRune := func(r rune) { ti.Handle(event.Key{Code: event.KeyRune, Rune: r, Press: true}) }

	typeRune('a')
	typeRune('b')
	typeRune('c')
	if ti.Value() != "abc" {
		t.Fatalf("insert = %q", ti.Value())
	}
	ti.Handle(event.Key{Code: event.KeyLeft, Press: true})
	ti.Handle(event.Key{Code: event.KeyBackspace, Press: true})
	if ti.Value() != "ac" {
		t.Errorf("backspace = %q want ac", ti.Value())
	}
	ti.Handle(event.Key{Code: event.KeyHome, Press: true})
	ti.Handle(event.Key{Code: event.KeyDelete, Press: true})
	if ti.Value() != "c" {
		t.Errorf("delete = %q want c", ti.Value())
	}
	ti.Handle(event.Key{Code: event.KeyEnd, Press: true})
	typeRune('z')
	if ti.Value() != "cz" {
		t.Errorf("append = %q want cz", ti.Value())
	}
}

// Home/End 光标定位。
func TestTextInputHomeEnd(t *testing.T) {
	ti := NewTextInput()
	ti.SetText("hello")
	ti.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 1})
	ti.OnFocus()
	if ti.Cursor() != 5 {
		t.Errorf("focus puts cursor at end: %d", ti.Cursor())
	}
	ti.Handle(event.Key{Code: event.KeyHome, Press: true})
	if ti.Cursor() != 0 {
		t.Errorf("home = %d", ti.Cursor())
	}
	ti.Handle(event.Key{Code: event.KeyEnd, Press: true})
	if ti.Cursor() != 5 {
		t.Errorf("end = %d", ti.Cursor())
	}
}

// Ctrl+A 全选 + 删除选区。
func TestTextInputSelectAll(t *testing.T) {
	ti := NewTextInput()
	ti.SetText("abcdef")
	ti.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 1})
	ti.OnFocus()
	ti.Handle(event.Key{Code: event.KeyRune, Rune: 0x01, Mod: event.ModCtrl, Press: true})
	s, e := ti.Selection()
	if s != 0 || e != 6 {
		t.Fatalf("select all = %d..%d want 0..6", s, e)
	}
	ti.Handle(event.Key{Code: event.KeyBackspace, Press: true})
	if ti.Value() != "" {
		t.Errorf("delete selection = %q want empty", ti.Value())
	}
}

// 真实 input.Decoder 解码的 Ctrl+A 为 Rune='a' + ModCtrl。
func TestTextInputSelectAllDecoded(t *testing.T) {
	ti := NewTextInput()
	ti.SetText("abcdef")
	ti.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 1})
	ti.OnFocus()
	ti.Handle(event.Key{Code: event.KeyRune, Rune: 'a', Mod: event.ModCtrl, Press: true})
	s, e := ti.Selection()
	if s != 0 || e != 6 {
		t.Fatalf("select all with decoded 'a' = %d..%d want 0..6", s, e)
	}
}

// MaxLength 截断插入与 SetText。
func TestTextInputMaxLength(t *testing.T) {
	ti := NewTextInput()
	ti.SetMaxLength(3)
	ti.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 1})
	ti.OnFocus()
	for _, r := range "abcdef" {
		ti.Handle(event.Key{Code: event.KeyRune, Rune: r, Press: true})
	}
	if ti.Value() != "abc" {
		t.Errorf("maxlen insert = %q want abc", ti.Value())
	}
	ti2 := NewTextInput()
	ti2.SetMaxLength(2)
	ti2.SetText("xyz")
	if ti2.Value() != "xy" {
		t.Errorf("maxlen settext = %q want xy", ti2.Value())
	}
}

// Placeholder：空值时显示。
func TestTextInputPlaceholder(t *testing.T) {
	ti := NewTextInput()
	ti.SetPlaceholder("type...")
	ti.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 1})
	mc := newMemCanvas(20, 1)
	ti.Paint(mc)
	found := false
	for _, tc := range mc.texts {
		if tc.s == "type..." && tc.st.Dim {
			found = true
		}
	}
	if !found {
		t.Errorf("placeholder missing/dim: %+v", mc.texts)
	}
	// 有值时不显示 placeholder
	ti.SetText("v")
	mc2 := newMemCanvas(20, 1)
	ti.Paint(mc2)
	for _, tc := range mc2.texts {
		if tc.s == "type..." {
			t.Error("placeholder must hide when value present")
		}
	}
}

// 横向滚动：内容宽于视口时视口跟随光标。
func TestTextInputHorizontalScroll(t *testing.T) {
	ti := NewTextInput()
	ti.Layout(geom.Rect{X: 0, Y: 0, W: 8, H: 1})
	ti.OnFocus()
	for _, r := range "abcdefghij" {
		ti.Handle(event.Key{Code: event.KeyRune, Rune: r, Press: true})
	}
	if ti.Value() != "abcdefghij" {
		t.Fatalf("value = %q", ti.Value())
	}
	off := ti.ViewOffset()
	if off == 0 {
		t.Error("cursor at end must scroll viewport")
	}
	// 回到 Home 附近视口应左移
	for i := 0; i < 8; i++ {
		ti.Handle(event.Key{Code: event.KeyLeft, Press: true})
	}
	if ti.ViewOffset() >= off {
		t.Errorf("cursor to head must scroll back: %d", ti.ViewOffset())
	}
}

// 光标点击定位：点击落在字符上 = 插入到该字符之前。
func TestTextInputClickCursor(t *testing.T) {
	ti := NewTextInput()
	ti.SetText("hello")
	ti.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 1})
	ti.OnFocus()
	// 边框在 x=0，内首字符 'h' 在 x=1 → 插入位 0
	ti.Handle(pressAt(1, 0))
	if ti.Cursor() != 0 {
		t.Errorf("click at first char = %d want 0", ti.Cursor())
	}
	// x=3 是第 3 个字符 'l'（h=1,e=2,l=3）→ 插入位 2
	ti.Handle(pressAt(3, 0))
	if ti.Cursor() != 2 {
		t.Errorf("click cursor = %d want 2", ti.Cursor())
	}
	// 界内空白处 → 末尾
	ti.Handle(pressAt(15, 0))
	if ti.Cursor() != 5 {
		t.Errorf("click past end = %d want 5", ti.Cursor())
	}
}

// 拖动选区。
func TestTextInputDragSelect(t *testing.T) {
	ti := NewTextInput()
	ti.SetText("hello")
	ti.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 1})
	ti.OnFocus()
	ti.Handle(pressAt(1, 0)) // anchor = 0
	ti.Handle(moveAt(4, 0))  // 拖到插入位 3 → 选中 "hel"
	s, e := ti.Selection()
	if s != 0 || e != 3 {
		t.Errorf("forward selection = %d..%d want 0..3", s, e)
	}
	// 反向拖动
	ti.Handle(pressAt(4, 0)) // anchor = 3
	ti.Handle(moveAt(2, 0))  // pos = 1
	s, e = ti.Selection()
	if s != 1 || e != 3 {
		t.Errorf("backward selection = %d..%d want 1..3", s, e)
	}
}

// Enter 触发 OnSubmit。
func TestTextInputSubmit(t *testing.T) {
	ti := NewTextInput()
	ti.SetText("v")
	ti.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 1})
	ti.OnFocus()
	var got string
	n := 0
	ti.SetOnSubmit(func(s string) { got = s; n++ })
	ti.Handle(event.Key{Code: event.KeyEnter, Press: true})
	if got != "v" || n != 1 {
		t.Errorf("submit = %q n=%d", got, n)
	}
}

// OnChange 随编辑触发。
func TestTextInputOnChange(t *testing.T) {
	ti := NewTextInput()
	ti.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 1})
	ti.OnFocus()
	n := 0
	ti.SetOnChange(func(string) { n++ })
	ti.Handle(event.Key{Code: event.KeyRune, Rune: 'a', Press: true})
	ti.Handle(event.Key{Code: event.KeyBackspace, Press: true})
	if n != 2 {
		t.Errorf("onChange calls = %d want 2", n)
	}
}

// 禁用吞事件。
func TestTextInputDisabled(t *testing.T) {
	ti := NewTextInput()
	ti.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 1})
	ti.OnFocus()
	ti.SetDisabled(true)
	if !ti.Handle(event.Key{Code: event.KeyRune, Rune: 'a', Press: true}) {
		t.Error("disabled key must swallow")
	}
	if ti.Value() != "" {
		t.Errorf("disabled must not edit: %q", ti.Value())
	}
}

// 粘贴事件批量插入。
func TestTextInputPaste(t *testing.T) {
	ti := NewTextInput()
	ti.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 1})
	ti.OnFocus()
	ti.Handle(event.Paste{Text: "xyz"})
	if ti.Value() != "xyz" {
		t.Errorf("paste = %q", ti.Value())
	}
}

// Paint golden：文本按格写入（宽字符需步进）。
func TestTextInputPaint(t *testing.T) {
	ti := NewTextInput()
	ti.SetText("ab")
	ti.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 1})
	mc := newMemCanvas(10, 1)
	ti.Paint(mc)
	var got []rune
	for _, cc := range mc.cellCalls {
		if cc.x == 1 || cc.x == 2 {
			got = append(got, cc.r)
		}
	}
	if string(got) != "ab" {
		t.Errorf("cells = %q want ab (%+v)", string(got), mc.cellCalls)
	}
	if len(mc.boxes) != 1 {
		t.Errorf("must draw box: %+v", mc.boxes)
	}
}
