package widget

import (
	"testing"

	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/widget/layout"
)

// Measure：宽=最长项+滚动条，高=min(项数, MaxH)。
func TestListMeasure(t *testing.T) {
	l := NewList()
	l.SetItems([]string{"a", "bbb", "cc"})
	got := l.Measure(layout.Constraints{MaxW: 80, MaxH: 10})
	if got.W != 5 || got.H != 3 { // 最长 3 + 滚动条 1 + 余量 1
		t.Errorf("measure = %+v", got)
	}
	// 高度按 MaxH 钳制
	got2 := l.Measure(layout.Constraints{MaxW: 80, MaxH: 2})
	if got2.H != 2 {
		t.Errorf("clamped height = %+v", got2.H)
	}
	// 空列表
	if got3 := NewList().Measure(layout.Constraints{MaxW: 80, MaxH: 5}); got3.H != 0 {
		t.Errorf("empty = %+v", got3)
	}
}

// Up/Down 移动选择并自动滚入视。
func TestListSelectionScroll(t *testing.T) {
	l := NewList()
	l.SetItems([]string{"0", "1", "2", "3", "4"})
	l.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 2})
	l.OnFocus()
	if l.Offset() != 0 {
		t.Errorf("initial offset = %d", l.Offset())
	}
	// 初始选中首项
	if l.Selected() != 0 {
		t.Errorf("initial selected = %d want 0", l.Selected())
	}
	// 下移到视口底部
	l.Handle(event.Key{Code: event.KeyDown, Press: true})
	if l.Selected() != 1 {
		t.Errorf("selected = %d want 1", l.Selected())
	}
	for i := 0; i < 2; i++ {
		l.Handle(event.Key{Code: event.KeyDown, Press: true})
	}
	if l.Selected() != 3 {
		t.Fatalf("selected = %d want 3", l.Selected())
	}
	if l.Offset() == 0 {
		t.Error("selection must scroll into view")
	}
	// 末项再往下不越界
	l.Handle(event.Key{Code: event.KeyDown, Press: true})
	if l.Selected() != 4 {
		t.Errorf("clamp at end = %d want 4", l.Selected())
	}
	// 回到顶部
	for i := 0; i < 10; i++ {
		l.Handle(event.Key{Code: event.KeyUp, Press: true})
	}
	if l.Selected() != 0 {
		t.Errorf("clamp at head = %d want 0", l.Selected())
	}
	if l.Offset() != 0 {
		t.Errorf("offset must return to 0 = %d", l.Offset())
	}
}

// Home/End/PgDn/PgUp。
func TestListPageKeys(t *testing.T) {
	l := NewList()
	items := []string{"0", "1", "2", "3", "4", "5"}
	l.SetItems(items)
	l.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 3})
	l.OnFocus()
	l.Handle(event.Key{Code: event.KeyEnd, Press: true})
	if l.Selected() != 5 {
		t.Errorf("end = %d want 5", l.Selected())
	}
	l.Handle(event.Key{Code: event.KeyHome, Press: true})
	if l.Selected() != 0 {
		t.Errorf("home = %d", l.Selected())
	}
	l.Handle(event.Key{Code: event.KeyPgDn, Press: true})
	if l.Selected() != 3 {
		t.Errorf("pgdn = %d want 3", l.Selected())
	}
	l.Handle(event.Key{Code: event.KeyPgUp, Press: true})
	if l.Selected() != 0 {
		t.Errorf("pgup = %d want 0", l.Selected())
	}
}

// 滚轮滚动（不移动选择）。
func TestListWheel(t *testing.T) {
	l := NewList()
	l.SetItems([]string{"0", "1", "2", "3", "4"})
	l.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 2})
	l.OnFocus()
	before := l.Selected()
	if !l.Handle(event.Mouse{Kind: event.MouseWheelDown}) {
		t.Error("wheel must be consumed")
	}
	if l.Offset() == 0 {
		t.Error("wheel must scroll")
	}
	if l.Selected() != before {
		t.Errorf("wheel must not move selection: %d -> %d", before, l.Selected())
	}
	l.Handle(event.Mouse{Kind: event.MouseWheelUp})
	if l.Offset() != 0 {
		t.Errorf("wheel up = %d", l.Offset())
	}
}

// Enter 触发 Activate；点击选中。
func TestListActivateAndClick(t *testing.T) {
	l := NewList()
	l.SetItems([]string{"a", "b", "c"})
	l.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 3})
	l.OnFocus()
	activated := -1
	l.SetOnActivate(func(i int) { activated = i })
	l.Handle(event.Key{Code: event.KeyEnter, Press: true})
	if activated != 0 {
		t.Errorf("activate = %d want 0", activated)
	}
	// 点击第 2 行
	l.Handle(pressAt(2, 2))
	if l.Selected() != 2 {
		t.Errorf("click select = %d want 2", l.Selected())
	}
}

// SetSelected 钳制并回调 OnChange。
func TestListSetSelected(t *testing.T) {
	l := NewList()
	l.SetItems([]string{"a", "b"})
	l.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 3})
	got := -1
	l.SetOnChange(func(i int) { got = i })
	l.SetSelected(5)
	if l.Selected() != 1 {
		t.Errorf("clamp = %d want 1", l.Selected())
	}
	if got != 1 {
		t.Errorf("onChange = %d", got)
	}
}

// 禁用吞事件。
func TestListDisabled(t *testing.T) {
	l := NewList()
	l.SetItems([]string{"a", "b"})
	l.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 3})
	l.OnFocus()
	l.SetDisabled(true)
	if !l.Handle(event.Key{Code: event.KeyDown, Press: true}) {
		t.Error("disabled key must swallow")
	}
	if !l.Handle(event.Mouse{Kind: event.MouseWheelDown}) {
		t.Error("disabled wheel must swallow")
	}
	if l.Selected() != 0 {
		t.Errorf("disabled must not change: %d", l.Selected())
	}
}

// Paint golden：只画可见行，选中行反色，右侧滚动条指示。
func TestListPaint(t *testing.T) {
	l := NewList()
	l.SetItems([]string{"aa", "bb", "cc", "dd"})
	l.Layout(geom.Rect{X: 0, Y: 0, W: 6, H: 2})
	l.SetSelected(1)
	l.OnFocus() // 聚焦时选中行反色
	mc := newMemCanvas(6, 2)
	l.Paint(mc)
	if len(mc.texts) != 2 {
		t.Fatalf("must draw 2 visible rows: %+v", mc.texts)
	}
	if mc.texts[0].s != "aa   " || mc.texts[1].s != "bb   " {
		t.Errorf("rows = %q %q want padded to scrollbar column", mc.texts[0].s, mc.texts[1].s)
	}
	if !mc.texts[1].st.Reverse {
		t.Errorf("selected row must be reverse: %+v", mc.texts[1].st)
	}
	if mc.texts[0].st.Reverse {
		t.Errorf("unselected must not be reverse")
	}
	// 右侧滚动条指示（内容超视口）
	foundBar := false
	for _, cc := range mc.cellCalls {
		if cc.x == 5 && (cc.r == '█' || cc.r == '░') {
			foundBar = true
		}
	}
	if !foundBar {
		t.Errorf("scrollbar indicator missing: %+v", mc.cellCalls)
	}
}

// 偏移裁剪：滚动后只画对应行。
func TestListPaintAfterScroll(t *testing.T) {
	l := NewList()
	l.SetItems([]string{"a0", "b1", "c2", "d3"})
	l.Layout(geom.Rect{X: 0, Y: 0, W: 6, H: 2})
	l.Handle(event.Mouse{Kind: event.MouseWheelDown})
	mc := newMemCanvas(6, 2)
	l.Paint(mc)
	if len(mc.texts) != 2 {
		t.Fatalf("rows = %+v", mc.texts)
	}
	if mc.texts[0].s != "b1   " {
		t.Errorf("after scroll row0 = %q want b1 padded", mc.texts[0].s)
	}
}
