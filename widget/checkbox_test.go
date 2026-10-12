package widget

import (
	"testing"

	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/widget/layout"
)

// 点击方框与点击标签均切换（界内任意位置）。
func TestCheckboxClickToggles(t *testing.T) {
	c := NewCheckbox("agree")
	c.Layout(geom.Rect{X: 0, Y: 0, W: 12, H: 1})
	got := -1
	c.SetOnChange(func(b bool) {
		if b {
			got = 1
		} else {
			got = 0
		}
	})
	// 点方框（第 0 列）
	c.Handle(pressAt(0, 0))
	c.Handle(releaseAt(0, 0))
	if !c.Checked() || got != 1 {
		t.Errorf("box click: checked=%v cb=%d", c.Checked(), got)
	}
	// 点标签
	c.Handle(pressAt(6, 0))
	c.Handle(releaseAt(6, 0))
	if c.Checked() || got != 0 {
		t.Errorf("label click: checked=%v cb=%d", c.Checked(), got)
	}
}

// Space 切换。
func TestCheckboxSpace(t *testing.T) {
	c := NewCheckbox("x")
	c.Layout(geom.Rect{X: 0, Y: 0, W: 8, H: 1})
	c.OnFocus()
	c.Handle(event.Key{Code: event.KeySpace, Press: true})
	if !c.Checked() {
		t.Error("space must toggle on")
	}
	c.Handle(event.Key{Code: event.KeySpace, Press: true})
	if c.Checked() {
		t.Error("space must toggle off")
	}
}

// 界外释放不切换。
func TestCheckboxReleaseOutside(t *testing.T) {
	c := NewCheckbox("x")
	c.Layout(geom.Rect{X: 0, Y: 0, W: 6, H: 1})
	c.Handle(pressAt(1, 0))
	c.Handle(releaseAt(30, 15))
	if c.Checked() {
		t.Error("release outside must not toggle")
	}
}

// 禁用吞事件。
func TestCheckboxDisabled(t *testing.T) {
	c := NewCheckbox("x")
	c.Layout(geom.Rect{X: 0, Y: 0, W: 8, H: 1})
	c.SetDisabled(true)
	if !c.Handle(pressAt(1, 0)) {
		t.Error("disabled press must swallow")
	}
	c.OnFocus()
	if !c.Handle(event.Key{Code: event.KeySpace, Press: true}) {
		t.Error("disabled key must swallow")
	}
	if c.Checked() {
		t.Error("disabled must not toggle")
	}
}

// Measure：方框(3) + 空格 + 文本。
func TestCheckboxMeasure(t *testing.T) {
	got := NewCheckbox("abc").Measure(layout.Constraints{MaxW: 80, MaxH: 24})
	if got.H != 1 || got.W != 7 {
		t.Errorf("measure = %+v", got)
	}
}

// Paint golden：未选 "[ ] "，选中 "[X] "；两者都补齐到控件宽（尾部空档纪律）。
func TestCheckboxPaint(t *testing.T) {
	c := NewCheckbox("hi")
	c.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 1})
	mc := newMemCanvas(10, 1)
	c.Paint(mc)
	if len(mc.texts) != 1 || mc.texts[0].s != "[ ] hi    " {
		t.Errorf("unchecked = %+v", mc.texts)
	}
	c.SetChecked(true)
	mc2 := newMemCanvas(10, 1)
	c.Paint(mc2)
	if len(mc2.texts) != 1 || mc2.texts[0].s != "[X] hi    " {
		t.Errorf("checked = %+v", mc2.texts)
	}
}
