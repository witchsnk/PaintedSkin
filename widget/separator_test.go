package widget

import (
	"testing"

	"PaintedSkin/geom"
	"PaintedSkin/widget/layout"
)

// Measure：H 方向固定高 1 格，V 方向固定宽 1 格。
func TestSeparatorMeasure(t *testing.T) {
	loose := layout.Constraints{MaxW: 80, MaxH: 24}
	h := NewSeparator()
	if got := h.Measure(loose); got.H != 1 {
		t.Errorf("H height = %+v want H=1", got)
	}
	v := NewSeparator()
	v.SetHorizontal(false)
	if got := v.Measure(loose); got.W != 1 {
		t.Errorf("V width = %+v want W=1", got)
	}
	// 边界：MaxW/MaxH 为 0 时"固定 1 格"那一维不退化
	if got := NewSeparator().Measure(layout.Constraints{}); got.H != 1 {
		t.Errorf("zero constraints = %+v", got)
	}
}

// Paint：沿线长重复线字符。
func TestSeparatorPaint(t *testing.T) {
	h := NewSeparator()
	h.Layout(geom.Rect{X: 2, Y: 1, W: 5, H: 1})
	mc := newMemCanvas(10, 3)
	h.Paint(mc)
	if len(mc.texts) != 1 {
		t.Fatalf("texts = %+v", mc.texts)
	}
	if mc.texts[0].x != 2 || mc.texts[0].y != 1 || mc.texts[0].s != "─────" {
		t.Errorf("H paint = %+v", mc.texts[0])
	}

	v := NewSeparator()
	v.SetHorizontal(false)
	v.Layout(geom.Rect{X: 3, Y: 1, W: 1, H: 3})
	mc2 := newMemCanvas(10, 5)
	v.Paint(mc2)
	if len(mc2.texts) != 3 {
		t.Fatalf("V texts = %+v", mc2.texts)
	}
	for i := 0; i < 3; i++ {
		if mc2.texts[i].x != 3 || mc2.texts[i].y != 1+i || mc2.texts[i].s != "│" {
			t.Errorf("V[%d] = %+v", i, mc2.texts[i])
		}
	}
}

// 自定义线字符生效。
func TestSeparatorLineRune(t *testing.T) {
	s := NewSeparator()
	s.SetLine('=')
	s.Layout(geom.Rect{X: 0, Y: 0, W: 3, H: 1})
	mc := newMemCanvas(3, 1)
	s.Paint(mc)
	if len(mc.texts) != 1 || mc.texts[0].s != "===" {
		t.Errorf("custom line = %+v", mc.texts)
	}
}

// 空区域不绘制；不可聚焦；禁用吞事件。
func TestSeparatorNoInteract(t *testing.T) {
	s := NewSeparator()
	if s.IsFocusable() {
		t.Error("separator must not be focusable")
	}
	if s.Handle(pressAt(0, 0)) {
		t.Error("normal separator must not consume")
	}
	s.SetDisabled(true)
	if !s.Handle(pressAt(0, 0)) {
		t.Error("disabled separator must swallow")
	}
	s.Layout(geom.Rect{})
	mc := newMemCanvas(3, 1)
	s.Paint(mc)
	if len(mc.texts) != 0 {
		t.Errorf("empty bounds must not paint: %+v", mc.texts)
	}
}
