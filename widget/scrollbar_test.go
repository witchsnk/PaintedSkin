package widget

import (
	"testing"

	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/widget/layout"
)

// Measure：垂直 W=1、H=MaxH；水平 W=MaxW、H=1。
func TestScrollbarMeasure(t *testing.T) {
	v := NewScrollbar()
	if got := v.Measure(layout.Constraints{MaxW: 80, MaxH: 20}); got.W != 1 || got.H != 20 {
		t.Errorf("V = %+v", got)
	}
	h := NewScrollbar()
	h.SetHorizontal(true)
	if got := h.Measure(layout.Constraints{MaxW: 80, MaxH: 20}); got.W != 80 || got.H != 1 {
		t.Errorf("H = %+v", got)
	}
}

// Offset 钳制在 [0, Total-View]；内容不足视口时为 0。
func TestScrollbarClamp(t *testing.T) {
	s := NewScrollbar()
	s.SetTotal(20)
	s.SetView(10)
	s.SetOffset(99)
	if s.Offset() != 10 {
		t.Errorf("over = %d want 10", s.Offset())
	}
	s.SetOffset(-5)
	if s.Offset() != 0 {
		t.Errorf("neg = %d want 0", s.Offset())
	}
	s.SetTotal(5)
	s.SetView(10)
	if s.Offset() != 0 {
		t.Errorf("small content = %d want 0", s.Offset())
	}
}

// 轨道点击跳页；界外点击不跳。
func TestScrollbarTrackClick(t *testing.T) {
	s := NewScrollbar()
	s.SetTotal(100)
	s.SetView(10)
	s.Layout(geom.Rect{X: 0, Y: 0, W: 1, H: 20})
	// 底部轨道点击 → 向下一页（Press 即跳页并消费）
	if !s.Handle(pressAt(0, 18)) {
		t.Error("track click must be consumed")
	}
	if s.Offset() != 10 {
		t.Errorf("page down = %d want 10", s.Offset())
	}
	// 顶部轨道点击 → 上一页
	s.Handle(pressAt(0, 0))
	s.Handle(releaseAt(0, 0))
	if s.Offset() != 0 {
		t.Errorf("page up = %d want 0", s.Offset())
	}
}

// 拖拽滑块：Press 命中滑块后 Move 换算 Offset，Release 结束。
func TestScrollbarDrag(t *testing.T) {
	s := NewScrollbar()
	s.SetTotal(100)
	s.SetView(10)
	s.Layout(geom.Rect{X: 0, Y: 0, W: 1, H: 20})
	// Total=100/View=10 → 滑块长 = 20*10/100 = 2，位于 offset 0 处
	if !s.Handle(pressAt(0, 0)) {
		t.Fatal("thumb press must consume")
	}
	// 拖到 y=18 → offset ≈ 18/18*90 = 90
	s.Handle(moveAt(0, 18))
	if s.Offset() <= 0 {
		t.Errorf("drag must change offset: %d", s.Offset())
	}
	before := s.Offset()
	s.Handle(releaseAt(0, 18))
	if s.Offset() != before {
		t.Errorf("release keeps offset: %d vs %d", s.Offset(), before)
	}
	// 释放后再 Move 不应改变
	after := s.Offset()
	s.Handle(moveAt(0, 0))
	if s.Offset() != after {
		t.Errorf("after release no change: %d vs %d", s.Offset(), after)
	}
}

// OnChange 在交互触发时被调用。
func TestScrollbarOnChange(t *testing.T) {
	s := NewScrollbar()
	s.SetTotal(100)
	s.SetView(10)
	s.Layout(geom.Rect{X: 0, Y: 0, W: 1, H: 20})
	n := 0
	s.SetOnChange(func(off int) { n++ })
	s.Handle(pressAt(0, 0))
	s.Handle(moveAt(0, 10))
	if n == 0 {
		t.Error("drag must fire onChange")
	}
	// 直接 setter 不触发回调
	prev := n
	s.SetOffset(3)
	if n != prev {
		t.Error("SetOffset must not fire onChange")
	}
}

// 滚轮转发给宿主（不消费）。
func TestScrollbarWheelForwards(t *testing.T) {
	s := NewScrollbar()
	w := event.Mouse{Kind: event.MouseWheelDown}
	if s.Handle(w) {
		t.Error("wheel must not be consumed")
	}
	u := event.Mouse{Kind: event.MouseWheelUp}
	if s.Handle(u) {
		t.Error("wheel up must not be consumed")
	}
}

// 禁用吞事件。
func TestScrollbarDisabled(t *testing.T) {
	s := NewScrollbar()
	s.SetTotal(100)
	s.SetView(10)
	s.Layout(geom.Rect{X: 0, Y: 0, W: 1, H: 20})
	s.SetDisabled(true)
	if !s.Handle(pressAt(0, 18)) {
		t.Error("disabled press must swallow")
	}
	if s.Offset() != 0 {
		t.Errorf("disabled must not scroll: %d", s.Offset())
	}
}

// Paint：画轨道 + 滑块。
func TestScrollbarPaint(t *testing.T) {
	s := NewScrollbar()
	s.SetTotal(100)
	s.SetView(10)
	s.SetOffset(0)
	s.Layout(geom.Rect{X: 3, Y: 0, W: 1, H: 10})
	mc := newMemCanvas(5, 10)
	s.Paint(mc)
	if len(mc.texts) != 10 {
		t.Fatalf("texts = %+v", mc.texts)
	}
	if mc.texts[0].x != 3 || mc.texts[0].s != "█" {
		t.Errorf("thumb at top = %+v", mc.texts[0])
	}
	if mc.texts[9].s != "░" {
		t.Errorf("track = %+v", mc.texts[9])
	}
}

// 内容不足视口：滑块铺满轨道。
func TestScrollbarFullTrack(t *testing.T) {
	s := NewScrollbar()
	s.SetTotal(5)
	s.SetView(10)
	s.Layout(geom.Rect{X: 0, Y: 0, W: 1, H: 6})
	mc := newMemCanvas(1, 6)
	s.Paint(mc)
	for i := 0; i < 6; i++ {
		if mc.texts[i].s != "█" {
			t.Errorf("[%d] = %+v", i, mc.texts[i])
		}
	}
}
