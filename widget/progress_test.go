package widget

import (
	"testing"

	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/widget/layout"
)

// 规格：交互无 → 不可聚焦、不消费方向键。
func TestProgressNoInteract(t *testing.T) {
	p := NewProgress()
	if p.IsFocusable() {
		t.Error("progress bar must not be focusable (docs/08 交互：无)")
	}
	p.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 3})
	p.SetValue(50)
	before := p.Value()
	if p.Handle(event.Key{Code: event.KeyUp, Press: true}) {
		t.Error("progress must not consume keys")
	}
	if p.Value() != before {
		t.Errorf("value changed = %d want %d", p.Value(), before)
	}
	if p.Handle(pressAt(1, 1)) {
		t.Error("progress must not consume mouse press")
	}
}

// 禁用吞事件。
func TestProgressDisabled(t *testing.T) {
	p := NewProgress()
	p.SetDisabled(true)
	if !p.Handle(pressAt(1, 1)) {
		t.Error("disabled progress must swallow")
	}
	if !p.Handle(event.Key{Code: event.KeyUp, Press: true}) {
		t.Error("disabled progress must swallow key")
	}
}

// SetValue / SetRange 钳制。
func TestProgressClamp(t *testing.T) {
	p := NewProgress()
	p.SetRange(0, 100)
	p.SetValue(150)
	if p.Value() != 100 {
		t.Errorf("over = %d want 100", p.Value())
	}
	p.SetValue(-10)
	if p.Value() != 0 {
		t.Errorf("under = %d want 0", p.Value())
	}
	p.SetRange(10, 20)
	if p.Value() < 10 || p.Value() > 20 {
		t.Errorf("after range change = %d", p.Value())
	}
}

// Measure 首选 20×3 钳制。
func TestProgressMeasure(t *testing.T) {
	if got := NewProgress().Measure(layout.Constraints{MaxW: 80, MaxH: 24}); got.W != 20 || got.H != 3 {
		t.Errorf("measure = %+v", got)
	}
	if got := NewProgress().Measure(layout.Constraints{MaxW: 10, MaxH: 2}); got.W != 10 || got.H != 2 {
		t.Errorf("clamped = %+v", got)
	}
}

// Paint golden：填充宽度随 Value，空闲格与边框。
func TestProgressPaint(t *testing.T) {
	p := NewProgress()
	p.SetRange(0, 100)
	p.SetValue(50)
	p.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 3})
	mc := newMemCanvas(10, 3)
	p.Paint(mc)
	if len(mc.boxes) != 1 {
		t.Errorf("must draw box: %+v", mc.boxes)
	}
	// 内区宽 8，50% → 4 格填充（按字形找，不依赖 fills 的下标：
	// Paint 第一步是铺自身背景，fills[0] 是那次铺底）。
	var filled, rest *fillCall
	for i := range mc.fills {
		f := &mc.fills[i]
		switch f.ch {
		case screen.Bar.Filled:
			filled = f
		case ' ':
			if f.r.W != 10 || f.r.H != 3 {
				rest = f // 未铺满整区的空格填充 = 内区未填充部分
			}
		}
	}
	if filled == nil {
		t.Fatalf("no filled bar: %+v", mc.fills)
	}
	if filled.r.W != 4 {
		t.Errorf("filled width = %+v want W=4", filled.r)
	}
	if rest == nil {
		t.Fatalf("no unfilled bar: %+v", mc.fills)
	}
	if rest.r.W != 4 {
		t.Errorf("unfilled = %+v want W=4", rest.r)
	}
}

// 百分比文本 golden。
func TestProgressPercentText(t *testing.T) {
	p := NewProgress()
	p.SetShowPercent(true)
	p.SetFormat("50%")
	p.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 3})
	mc := newMemCanvas(10, 3)
	p.Paint(mc)
	found := false
	for _, tc := range mc.texts {
		if tc.s == "50%" {
			found = true
		}
	}
	if !found {
		t.Errorf("percent text missing: %+v", mc.texts)
	}
}

// 未设 Format 时，默认百分比按 Value/Range 计算，而非固定为 0%。
func TestProgressDefaultPercentText(t *testing.T) {
	p := NewProgress()
	p.SetRange(0, 100)
	p.SetValue(60)
	p.SetShowPercent(true)
	p.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 3})
	mc := newMemCanvas(10, 3)
	p.Paint(mc)
	found := false
	for _, tc := range mc.texts {
		if tc.s == "60%" {
			found = true
		}
	}
	if !found {
		t.Errorf("default percent text for value=60 want 60%%: %+v", mc.texts)
	}
}

// H=1 时退化为单行条，不画框。
func TestProgressSingleRow(t *testing.T) {
	p := NewProgress()
	p.Layout(geom.Rect{X: 0, Y: 0, W: 8, H: 1})
	mc := newMemCanvas(8, 1)
	p.Paint(mc)
	if len(mc.boxes) != 0 {
		t.Errorf("H=1 must not draw box: %+v", mc.boxes)
	}
	if len(mc.texts) != 1 {
		t.Fatalf("texts = %+v", mc.texts)
	}
	// 8 宽 → 内区 6，0% → 无 '=' 无 '>'
	if mc.texts[0].s != "[      ]" {
		t.Errorf("bar = %q", mc.texts[0].s)
	}
}
