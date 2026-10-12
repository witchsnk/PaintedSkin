package widget

import (
	"testing"

	"PaintedSkin/geom"
	"PaintedSkin/style"
	"PaintedSkin/widget/layout"
)

// Measure：三段文本；H 固定 1 行，宽优先取 MaxW。
func TestStatusBarMeasure(t *testing.T) {
	s := NewStatusBar()
	s.SetLeft("L")
	s.SetCenter("C")
	s.SetRight("R")
	if got := s.Measure(layout.Constraints{MaxW: 80, MaxH: 24}); got != (geom.Size{W: 80, H: 1}) {
		t.Errorf("full = %+v", got)
	}
	if got := s.Measure(layout.Constraints{MaxW: 10, MaxH: 24}); got != (geom.Size{W: 10, H: 1}) {
		t.Errorf("clamped = %+v", got)
	}
	// MaxW<0（无上界）时按三段显示宽之和
	sum := NewStatusBar()
	sum.SetLeft("LL")
	sum.SetCenter("CC")
	sum.SetRight("R")
	if got := sum.Measure(layout.Constraints{MaxW: -1, MaxH: -1}); got != (geom.Size{W: 5, H: 1}) {
		t.Errorf("natural = %+v want W=5", got)
	}
}

// Paint：左段左对齐、中段居中、右段右对齐。
func TestStatusBarPaint(t *testing.T) {
	s := NewStatusBar()
	s.SetLeft("L")
	s.SetCenter("C")
	s.SetRight("R")
	s.Layout(geom.Rect{X: 0, Y: 0, W: 11, H: 1})
	mc := newMemCanvas(11, 1)
	s.Paint(mc)
	if len(mc.texts) != 3 {
		t.Fatalf("texts = %+v", mc.texts)
	}
	byText := map[string]textCall{}
	for _, tc := range mc.texts {
		byText[tc.s] = tc
	}
	if byText["L"].x != 0 {
		t.Errorf("left x = %d", byText["L"].x)
	}
	if byText["C"].x != 5 {
		t.Errorf("center x = %d want 5", byText["C"].x)
	}
	if byText["R"].x != 10 {
		t.Errorf("right x = %d want 10", byText["R"].x)
	}
}

// 空段不绘制；CJK 按显示宽对齐。
func TestStatusBarEmptyAndCJK(t *testing.T) {
	s := NewStatusBar()
	s.SetLeft("中文")
	s.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 1})
	mc := newMemCanvas(10, 1)
	s.Paint(mc)
	if len(mc.texts) != 1 {
		t.Errorf("empty segments must not paint: %+v", mc.texts)
	}

	s2 := NewStatusBar()
	s2.SetRight("中文")
	s2.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 1})
	mc2 := newMemCanvas(10, 1)
	s2.Paint(mc2)
	if len(mc2.texts) != 1 || mc2.texts[0].x != 6 {
		t.Errorf("CJK right align = %+v", mc2.texts)
	}
}

// 独立段样式生效。
func TestStatusBarSegmentStyle(t *testing.T) {
	s := NewStatusBar()
	s.SetLeft("L")
	s.SetRightStyle(style.Style{}.WithBold(true))
	s.Layout(geom.Rect{X: 0, Y: 0, W: 6, H: 1})
	s.SetRight("R")
	mc := newMemCanvas(6, 1)
	s.Paint(mc)
	var right *textCall
	for i := range mc.texts {
		if mc.texts[i].s == "R" {
			right = &mc.texts[i]
		}
	}
	if right == nil || !right.st.Bold {
		t.Errorf("right style = %+v", mc.texts)
	}
}

// 不可聚焦；正常不消费；禁用吞。
func TestStatusBarNoInteract(t *testing.T) {
	s := NewStatusBar()
	if s.IsFocusable() {
		t.Error("status bar must not be focusable")
	}
	if s.Handle(pressAt(0, 0)) {
		t.Error("must not consume")
	}
	s.SetDisabled(true)
	if !s.Handle(pressAt(0, 0)) {
		t.Error("disabled must swallow")
	}
}
