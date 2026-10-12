package widget

import (
	"testing"

	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/widget/layout"
)

// 规格：点击=选中（不再 toggle）；已选中点击保持选中。
func TestRadioClickSelects(t *testing.T) {
	r := NewRadio("A")
	r.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 1})
	changes := 0
	r.SetOnChange(func(bool) { changes++ })

	r.Handle(pressAt(1, 0))
	r.Handle(releaseAt(1, 0))
	if !r.Checked() {
		t.Error("click must select")
	}
	if changes != 1 {
		t.Errorf("changes = %d want 1", changes)
	}
	// 再次点击仍选中，不反转、不重复回调
	r.Handle(pressAt(1, 0))
	r.Handle(releaseAt(1, 0))
	if !r.Checked() {
		t.Error("second click must stay selected")
	}
	if changes != 1 {
		t.Errorf("no further change: %d", changes)
	}
}

// Space/Enter 同样为选中语义。
func TestRadioKeySelects(t *testing.T) {
	r := NewRadio("A")
	r.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 1})
	r.Handle(pressAt(1, 0))
	r.Handle(releaseAt(1, 0))
	r.OnBlur()
	r.OnFocus()
	r.Handle(event.Key{Code: event.KeySpace, Press: true})
	if !r.Checked() {
		t.Error("space must keep selected")
	}
}

// 界外释放不选中。
func TestRadioReleaseOutside(t *testing.T) {
	r := NewRadio("A")
	r.Layout(geom.Rect{X: 0, Y: 0, W: 5, H: 1})
	r.Handle(pressAt(1, 0))
	r.Handle(releaseAt(30, 15))
	if r.Checked() {
		t.Error("release outside must not select")
	}
}

// Radio 自身不占 Tab 位（组内成员）。
func TestRadioNotFocusable(t *testing.T) {
	if NewRadio("A").IsFocusable() {
		t.Error("radio must not occupy a Tab slot")
	}
}

// ---- RadioGroup ----

// 组内互斥：选中一个，其余取消。
func TestRadioGroupMutual(t *testing.T) {
	g := NewRadioGroup("g1", []string{"A", "B", "C"})
	g.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 3})
	var idx = -1
	g.SetOnChange(func(i int) { idx = i })
	g.SetSelected(1)
	if g.Selected() != 1 {
		t.Fatalf("selected = %d", g.Selected())
	}
	// 点击第 0 项
	r0 := g.Radio(0)
	r0.Handle(pressAt(1, 0))
	r0.Handle(releaseAt(1, 0))
	if g.Selected() != 0 {
		t.Errorf("selected = %d want 0", g.Selected())
	}
	if idx != 0 {
		t.Errorf("onChange idx = %d want 0", idx)
	}
	if g.Radio(1).Checked() {
		t.Error("previous must be deselected")
	}
	if !g.Radio(0).Checked() {
		t.Error("clicked must be checked")
	}
}

// 方向键在组内移动选择。
func TestRadioGroupArrows(t *testing.T) {
	g := NewRadioGroup("g", []string{"A", "B", "C"})
	g.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 3})
	g.SetSelected(0)
	var seen []int
	g.SetOnChange(func(i int) { seen = append(seen, i) })
	g.OnFocus()

	g.Handle(event.Key{Code: event.KeyDown, Press: true})
	if g.Selected() != 1 {
		t.Errorf("down = %d want 1", g.Selected())
	}
	g.Handle(event.Key{Code: event.KeyDown, Press: true})
	if g.Selected() != 2 {
		t.Errorf("down = %d want 2", g.Selected())
	}
	g.Handle(event.Key{Code: event.KeyDown, Press: true})
	if g.Selected() != 2 {
		t.Errorf("clamp at end = %d want 2", g.Selected())
	}
	g.Handle(event.Key{Code: event.KeyUp, Press: true})
	if g.Selected() != 1 {
		t.Errorf("up = %d want 1", g.Selected())
	}
	g.Handle(event.Key{Code: event.KeyLeft, Press: true})
	if g.Selected() != 0 {
		t.Errorf("left = %d want 0", g.Selected())
	}
	g.Handle(event.Key{Code: event.KeyRight, Press: true})
	if g.Selected() != 1 {
		t.Errorf("right = %d want 1", g.Selected())
	}
	if len(seen) != 5 {
		t.Errorf("onChange calls = %d want 5 (末尾 clamp 不重复触发) %v", len(seen), seen)
	}
}

// 未聚焦时方向键不生效（避免抢全局键）。
func TestRadioGroupArrowsNeedFocus(t *testing.T) {
	g := NewRadioGroup("g", []string{"A", "B"})
	g.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 2})
	g.SetSelected(0)
	g.Handle(event.Key{Code: event.KeyDown, Press: true})
	if g.Selected() != 0 {
		t.Errorf("unfocused must not move: %d", g.Selected())
	}
}

// 组占一个 Tab 位；成员不成环。
func TestRadioGroupFocusSlot(t *testing.T) {
	r := mkRoot()
	g := NewRadioGroup("g", []string{"A", "B"})
	after := NewButton("After")
	r.Add(g)
	r.Add(after)
	g.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 2})
	after.Layout(geom.Rect{X: 0, Y: 5, W: 6, H: 1})

	r.Dispatch(tabKey(false))
	if r.Focused() != Widget(g) {
		t.Fatalf("tab1 = %v want group", r.Focused())
	}
	r.Dispatch(tabKey(false))
	if r.Focused() != Widget(after) {
		t.Errorf("tab2 = %v want button", r.Focused())
	}
	r.Dispatch(tabKey(false))
	if r.Focused() != Widget(g) {
		t.Errorf("ring must skip radios: %v", r.Focused())
	}
}

// Measure / Paint。
func TestRadioGroupMeasurePaint(t *testing.T) {
	g := NewRadioGroup("g", []string{"中", "B"})
	got := g.Measure(layout.Constraints{MaxW: 80, MaxH: 24})
	if got.H != 2 || got.W < 4 {
		t.Errorf("measure = %+v", got)
	}
	g.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 2})
	g.SetSelected(0)
	mc := newMemCanvas(10, 2)
	g.Paint(mc)
	if len(mc.texts) != 2 {
		t.Fatalf("texts = %+v", mc.texts)
	}
	if mc.texts[0].x != 0 || mc.texts[0].y != 0 {
		t.Errorf("row0 = %+v", mc.texts[0])
	}
	if mc.texts[1].x != 0 || mc.texts[1].y != 1 {
		t.Errorf("row1 = %+v", mc.texts[1])
	}
}

// 禁用吞事件。
func TestRadioGroupDisabled(t *testing.T) {
	g := NewRadioGroup("g", []string{"A", "B"})
	g.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 2})
	g.SetDisabled(true)
	if !g.Handle(pressAt(1, 0)) {
		t.Error("disabled group must swallow")
	}
	if !g.Handle(event.Key{Code: event.KeyDown, Press: true}) {
		t.Error("disabled group must swallow key")
	}
	if g.Radio(0).Checked() {
		t.Error("disabled must not select")
	}
}

// 空组不崩溃。
func TestRadioGroupEmpty(t *testing.T) {
	g := NewRadioGroup("g", nil)
	g.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 2})
	if got := len(g.Children()); got != 0 {
		t.Errorf("children = %d", got)
	}
	mc := newMemCanvas(10, 2)
	g.Paint(mc)
	if len(mc.texts) != 0 {
		t.Errorf("empty must not paint: %+v", mc.texts)
	}
	g.SetSelected(0)
	_ = g.Selected()
}
