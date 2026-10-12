package widget

import (
	"testing"

	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/widget/layout"
)

// 拖拽：Press 捕获 → Move 连续换算 → Release 结束（docs/08 合成事件序列）。
func TestSliderDrag(t *testing.T) {
	s := NewSlider()
	s.SetHorizontal(true)
	s.SetRange(0, 100)
	s.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 1})
	s.Handle(pressAt(0, 0))
	s.Handle(moveAt(10, 0))
	mid := s.Value()
	if mid <= 0 {
		t.Fatalf("drag must change value: %d", mid)
	}
	s.Handle(moveAt(18, 0))
	high := s.Value()
	if high <= mid {
		t.Errorf("continued drag must increase: %d -> %d", mid, high)
	}
	s.Handle(releaseAt(18, 0))
	after := s.Value()
	// 释放后 Move 不再影响
	s.Handle(moveAt(0, 0))
	if s.Value() != after {
		t.Errorf("after release value changed: %d -> %d", after, s.Value())
	}
}

// 点击轨道跳转。
func TestSliderTrackClick(t *testing.T) {
	s := NewSlider()
	s.SetHorizontal(true)
	s.SetRange(0, 100)
	s.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 1})
	s.Handle(pressAt(15, 0))
	v := s.Value()
	if v < 50 {
		t.Errorf("click at 15/20 must jump well past half: %d", v)
	}
}

// 方向键 ±Step；Home/End 到端点。
func TestSliderKeys(t *testing.T) {
	s := NewSlider()
	s.SetHorizontal(true)
	s.SetRange(0, 100)
	s.SetStep(5)
	s.SetValue(50)
	s.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 1})
	s.OnFocus()
	s.Handle(event.Key{Code: event.KeyRight, Press: true})
	if s.Value() != 55 {
		t.Errorf("right = %d want 55", s.Value())
	}
	s.Handle(event.Key{Code: event.KeyLeft, Press: true})
	if s.Value() != 50 {
		t.Errorf("left = %d want 50", s.Value())
	}
	s.Handle(event.Key{Code: event.KeyUp, Press: true})
	if s.Value() != 55 {
		t.Errorf("up = %d want 55", s.Value())
	}
	s.Handle(event.Key{Code: event.KeyDown, Press: true})
	if s.Value() != 50 {
		t.Errorf("down = %d want 50", s.Value())
	}
	s.Handle(event.Key{Code: event.KeyHome, Press: true})
	if s.Value() != 0 {
		t.Errorf("home = %d want 0", s.Value())
	}
	s.Handle(event.Key{Code: event.KeyEnd, Press: true})
	if s.Value() != 100 {
		t.Errorf("end = %d want 100", s.Value())
	}
	// 端点钳制
	s.Handle(event.Key{Code: event.KeyEnd, Press: true})
	if s.Value() != 100 {
		t.Errorf("clamp at max = %d", s.Value())
	}
}

// 未聚焦时方向键不生效。
func TestSliderKeysNeedFocus(t *testing.T) {
	s := NewSlider()
	s.SetHorizontal(true)
	s.SetRange(0, 100)
	s.SetValue(50)
	s.Handle(event.Key{Code: event.KeyRight, Press: true})
	if s.Value() != 50 {
		t.Errorf("unfocused must not change: %d", s.Value())
	}
}

// 禁用吞事件。
func TestSliderDisabled(t *testing.T) {
	s := NewSlider()
	s.SetHorizontal(true)
	s.SetRange(0, 100)
	s.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 1})
	s.SetDisabled(true)
	s.OnFocus()
	if !s.Handle(event.Key{Code: event.KeyRight, Press: true}) {
		t.Error("disabled key must swallow")
	}
	if !s.Handle(pressAt(5, 0)) {
		t.Error("disabled press must swallow")
	}
	if s.Value() != 50 {
		t.Errorf("disabled must not change value: %d", s.Value())
	}
}

// Measure 水平 30×3、垂直 15×7。
func TestSliderMeasure(t *testing.T) {
	h := NewSlider()
	h.SetHorizontal(true)
	if got := h.Measure(layout.Constraints{MaxW: 80, MaxH: 24}); got.W != 30 || got.H != 3 {
		t.Errorf("H = %+v", got)
	}
	v := NewSlider()
	v.SetHorizontal(false)
	if got := v.Measure(layout.Constraints{MaxW: 80, MaxH: 24}); got.W != 15 || got.H != 7 {
		t.Errorf("V = %+v", got)
	}
}

// Paint golden：填充 '=' + 滑块 '>' + 轨道空格。
func TestSliderPaint(t *testing.T) {
	s := NewSlider()
	s.SetHorizontal(true)
	s.SetRange(0, 100)
	s.SetValue(50)
	s.Layout(geom.Rect{X: 0, Y: 0, W: 12, H: 1})
	mc := newMemCanvas(12, 1)
	s.Paint(mc)
	if len(mc.texts) != 1 {
		t.Fatalf("texts = %+v", mc.texts)
	}
	// 内区 10 格，50% → 5 个 '='，第 6 格为 '>'
	if mc.texts[0].s != "[=====>    ]" {
		t.Errorf("bar = %q", mc.texts[0].s)
	}
}

// 点击最右端/最下端应能到达最大值。
func TestSliderClickReachMax(t *testing.T) {
	s := NewSlider()
	s.SetHorizontal(true)
	s.SetRange(0, 100)
	s.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 1})
	// 点击最右侧有效坐标 (19, 0)
	s.Handle(pressAt(19, 0))
	if s.Value() != 100 {
		t.Errorf("click rightmost cell want 100, got: %d", s.Value())
	}

	// 垂直方向
	sv := NewSlider()
	sv.SetHorizontal(false)
	sv.SetRange(0, 100)
	sv.Layout(geom.Rect{X: 0, Y: 0, W: 5, H: 20})
	sv.Handle(pressAt(0, 19))
	if sv.Value() != 100 {
		t.Errorf("click bottommost cell want 100, got: %d", sv.Value())
	}
}
