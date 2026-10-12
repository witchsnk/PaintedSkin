package widget

import (
	"testing"

	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/widget/layout"
)

// Measure：标签行高 1 + 内容高；宽取标签行与内容较大者。
func TestTabsMeasure(t *testing.T) {
	tb := NewTabs()
	tb.SetTabs([]string{"A", "BB"})
	content := newStub("c", true, geom.Size{W: 20, H: 3})
	tb.SetContent(content)
	got := tb.Measure(layout.Constraints{MaxW: 80, MaxH: 24})
	if got.H != 4 { // 标签行 1 + 内容 3
		t.Errorf("height = %+v", got)
	}
	if got.W < 20 {
		t.Errorf("width must fit content = %+v", got)
	}
}

// Layout：内容区为 bounds 去掉首行。
func TestTabsLayout(t *testing.T) {
	tb := NewTabs()
	tb.SetTabs([]string{"A"})
	c := newStub("c", true, geom.Size{W: 5, H: 2})
	tb.SetContent(c)
	tb.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 6})
	if c.Bounds != (geom.Rect{X: 0, Y: 1, W: 20, H: 5}) {
		t.Errorf("content bounds = %+v", c.Bounds)
	}
}

// 点击标签切换。
func TestTabsClickSwitch(t *testing.T) {
	tb := NewTabs()
	tb.SetTabs([]string{"A", "B", "C"})
	tb.Layout(geom.Rect{X: 0, Y: 0, W: 30, H: 5})
	got := -1
	tb.SetOnChange(func(i int) { got = i })
	if tb.Current() != 0 {
		t.Fatalf("initial = %d", tb.Current())
	}
	// 标签区在第 0 行；每个标签占 titleWidth
	w2 := tb.tabWidth(1)
	tb.Handle(pressAt(w2+1, 0))
	if tb.Current() != 1 {
		t.Errorf("click = %d want 1", tb.Current())
	}
	if got != 1 {
		t.Errorf("onChange = %d", got)
	}
}

// 左右键切页。
func TestTabsArrows(t *testing.T) {
	tb := NewTabs()
	tb.SetTabs([]string{"A", "B", "C"})
	tb.Layout(geom.Rect{X: 0, Y: 0, W: 30, H: 5})
	tb.OnFocus()
	tb.Handle(event.Key{Code: event.KeyRight, Press: true})
	if tb.Current() != 1 {
		t.Errorf("right = %d", tb.Current())
	}
	tb.Handle(event.Key{Code: event.KeyLeft, Press: true})
	if tb.Current() != 0 {
		t.Errorf("left = %d", tb.Current())
	}
	// 边界钳制
	tb.Handle(event.Key{Code: event.KeyLeft, Press: true})
	if tb.Current() != 0 {
		t.Errorf("clamp head = %d", tb.Current())
	}
}

// 标签条视为一组：Tab 键进入内容区。
func TestTabsFocusOrder(t *testing.T) {
	r := mkRoot()
	tb := NewTabs()
	tb.SetTabs([]string{"A", "B"})
	content := newStub("content", true, geom.Size{W: 10, H: 2})
	tb.SetContent(content)
	after := NewButton("After")
	r.Add(tb)
	r.Add(after)
	tb.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 6})
	after.Layout(geom.Rect{X: 0, Y: 8, W: 6, H: 1})

	r.Dispatch(tabKey(false))
	if r.Focused() != Widget(tb) {
		t.Fatalf("tab1 = %v want tabs", r.Focused())
	}
	r.Dispatch(tabKey(false))
	if r.Focused() != Widget(content) {
		t.Errorf("tab2 = %v want content", r.Focused())
	}
	r.Dispatch(tabKey(false))
	if r.Focused() != Widget(after) {
		t.Errorf("tab3 = %v want after", r.Focused())
	}
}

// HitTest：标签行命中自身，内容区转给内容。
func TestTabsHitTest(t *testing.T) {
	tb := NewTabs()
	tb.SetTabs([]string{"A"})
	c := newStub("c", false, geom.Size{W: 10, H: 2})
	tb.SetContent(c)
	tb.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 5})
	if hit := tb.HitTest(geom.Point{X: 1, Y: 0}); hit != Widget(tb) {
		t.Errorf("tab row hit = %v want tabs", hit)
	}
	if hit := tb.HitTest(geom.Point{X: 1, Y: 3}); hit != Widget(c) {
		t.Errorf("content hit = %v want content", hit)
	}
	if hit := tb.HitTest(geom.Point{X: 1, Y: 9}); hit != nil {
		t.Errorf("outside = %v want nil", hit)
	}
}

// 禁用吞事件。
func TestTabsDisabled(t *testing.T) {
	tb := NewTabs()
	tb.SetTabs([]string{"A", "B"})
	tb.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 5})
	tb.OnFocus()
	tb.SetDisabled(true)
	if !tb.Handle(event.Key{Code: event.KeyRight, Press: true}) {
		t.Error("disabled key must swallow")
	}
	if !tb.Handle(pressAt(1, 0)) {
		t.Error("disabled press must swallow")
	}
	if tb.Current() != 0 {
		t.Errorf("disabled must not switch: %d", tb.Current())
	}
}

// Paint golden：标签行，当前页反色；内容绘制。
func TestTabsPaint(t *testing.T) {
	tb := NewTabs()
	tb.SetTabs([]string{"A", "B"})
	c := newStub("c", false, geom.Size{W: 10, H: 2})
	tb.SetContent(c)
	tb.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 5})
	tb.SetCurrent(1)
	mc := newMemCanvas(20, 5)
	tb.Paint(mc)
	// 标签逐段绘制：第 0 段 " A "（x=0），第 1 段 " B "（x=3，当前页反色）
	if len(mc.texts) < 2 {
		t.Fatalf("tab segments missing: %+v", mc.texts)
	}
	if mc.texts[0].x != 0 || mc.texts[0].y != 0 || mc.texts[0].s != " A " {
		t.Errorf("tab0 = %+v", mc.texts[0])
	}
	if mc.texts[1].x != 3 || mc.texts[1].s != " B " {
		t.Errorf("tab1 = %+v", mc.texts[1])
	}
	if !mc.texts[1].st.Reverse {
		t.Errorf("current tab must be reverse: %+v", mc.texts[1].st)
	}
	if mc.texts[0].st.Reverse {
		t.Errorf("inactive tab must not be reverse")
	}
	// 内容被绘制（stub 用 Fill('.')）
	foundContent := false
	for _, f := range mc.fills {
		if f.ch == '.' {
			foundContent = true
		}
	}
	if !foundContent {
		t.Errorf("content must paint: %+v", mc)
	}
}
