package widget

import (
	"testing"

	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/widget/layout"
)

// Measure：关闭态 = 当前项文本 + " ▼"。
// overlayBounds 取出覆盖层的边界。
func overlayBounds(t *testing.T, ov Widget) geom.Rect {
	t.Helper()
	type bounded interface{ BoundsOf() geom.Rect }
	b, ok := ov.(bounded)
	if !ok {
		t.Fatalf("overlay %T has no bounds", ov)
	}
	return b.BoundsOf()
}

func TestDropdownMeasure(t *testing.T) {
	d := NewDropdown()
	d.SetItems([]string{"one", "three", "two"})
	d.SetSelected(1)
	got := d.Measure(layout.Constraints{MaxW: 80, MaxH: 24})
	// "three" 5 + 2 = 7
	if got.W != 7 || got.H != 1 {
		t.Errorf("measure = %+v", got)
	}
}

// 点击展开 → overlay 挂到宿主；再点收起。
func TestDropdownOpenClose(t *testing.T) {
	r := mkRoot()
	d := NewDropdown()
	d.SetItems([]string{"a", "b"})
	d.Layout(geom.Rect{X: 2, Y: 1, W: 10, H: 1})
	r.Add(d)

	if r.Overlay() != nil {
		t.Fatal("must start closed")
	}
	d.Handle(pressAt(3, 1))
	d.Handle(releaseAt(3, 1))
	if r.Overlay() == nil {
		t.Fatal("click must open overlay")
	}
	if !d.IsOpen() {
		t.Error("IsOpen must be true")
	}
	// 关闭
	d.Handle(pressAt(3, 1))
	d.Handle(releaseAt(3, 1))
	if r.Overlay() != nil {
		t.Error("second click must close")
	}
	if d.IsOpen() {
		t.Error("IsOpen must be false")
	}
}

// 弹层定位在下方。
func TestDropdownOverlayBelow(t *testing.T) {
	r := mkRoot()
	d := NewDropdown()
	d.SetItems([]string{"a", "b", "c"})
	d.Layout(geom.Rect{X: 2, Y: 1, W: 10, H: 1})
	r.Add(d)
	d.Handle(pressAt(3, 1))
	ov := r.Overlay()
	if ov == nil {
		t.Fatal("no overlay")
	}
	br := overlayBounds(t, ov)
	if br.Y != 2 {
		t.Errorf("overlay Y = %d want 2 (below)", br.Y)
	}
	if br.X != 2 {
		t.Errorf("overlay X = %d want 2 (align)", br.X)
	}
	// 3 个选项 + 上下边框 2 行。
	if br.H != 5 {
		t.Errorf("overlay H = %d want 5 (3 items + frame)", br.H)
	}
}

// 空间不足时翻转到上方。
func TestDropdownFlipUp(t *testing.T) {
	r := mkRoot() // 40x20
	d := NewDropdown()
	d.SetItems([]string{"a", "b", "c"})
	// 贴底放置：下方只剩 0 行
	d.Layout(geom.Rect{X: 2, Y: 19, W: 10, H: 1})
	r.Add(d)
	d.Handle(pressAt(3, 19))
	ov := r.Overlay()
	if ov == nil {
		t.Fatal("no overlay")
	}
	br := overlayBounds(t, ov)
	if br.Y+br.H > 19 {
		t.Errorf("must flip up: %+v", br)
	}
	if br.Y != 19-br.H {
		t.Errorf("flip up Y = %d want %d", br.Y, 19-br.H)
	}
}

// 点选项 → 选择并收起 + OnChange。
func TestDropdownSelectItem(t *testing.T) {
	r := mkRoot()
	d := NewDropdown()
	d.SetItems([]string{"a", "bb", "ccc"})
	d.Layout(geom.Rect{X: 2, Y: 1, W: 10, H: 1})
	r.Add(d)
	got := -1
	d.SetOnChange(func(i int) { got = i })
	d.Handle(pressAt(3, 1))
	ov := r.Overlay()
	if ov == nil {
		t.Fatal("not open")
	}
	br := overlayBounds(t, ov)
	// 点第 2 个选项（经宿主命中测试路由到弹层）；弹层顶边框占第 1 行。
	r.Dispatch(pressAt(br.X+1, br.Y+1+1))
	r.Dispatch(releaseAt(br.X+1, br.Y+1+1))
	if d.Selected() != 1 {
		t.Errorf("selected = %d want 1", d.Selected())
	}
	if got != 1 {
		t.Errorf("onChange = %d", got)
	}
	if r.Overlay() != nil {
		t.Error("select must close")
	}
}

// Esc 收起。
func TestDropdownEsc(t *testing.T) {
	r := mkRoot()
	d := NewDropdown()
	d.SetItems([]string{"a", "b"})
	d.Layout(geom.Rect{X: 2, Y: 1, W: 10, H: 1})
	r.Add(d)
	d.Handle(pressAt(3, 1))
	if r.Overlay() == nil {
		t.Fatal("not open")
	}
	d.Handle(event.Key{Code: event.KeyEsc, Press: true})
	if r.Overlay() != nil {
		t.Error("esc must close")
	}
}

// 展开时上下键预选、Enter 确认。
func TestDropdownKeyboard(t *testing.T) {
	r := mkRoot()
	d := NewDropdown()
	d.SetItems([]string{"a", "b", "c"})
	d.Layout(geom.Rect{X: 2, Y: 1, W: 10, H: 1})
	r.Add(d)
	d.OnFocus()
	got := -1
	d.SetOnChange(func(i int) { got = i })
	r.Dispatch(pressAt(3, 1))
	r.Dispatch(releaseAt(3, 1))
	if !d.IsOpen() {
		t.Fatal("not open")
	}
	r.Dispatch(event.Key{Code: event.KeyDown, Press: true})
	if d.Selected() != 1 {
		t.Errorf("down = %d", d.Selected())
	}
	r.Dispatch(event.Key{Code: event.KeyDown, Press: true})
	if d.Selected() != 2 {
		t.Errorf("down = %d", d.Selected())
	}
	r.Dispatch(event.Key{Code: event.KeyEnter, Press: true})
	if got != 2 {
		t.Errorf("enter onChange = %d", got)
	}
	if r.Overlay() != nil {
		t.Error("enter must close")
	}
}

// 点外部收起（Root 机制）。
func TestDropdownOutsideClickCloses(t *testing.T) {
	r := mkRoot()
	d := NewDropdown()
	d.SetItems([]string{"a", "b"})
	d.Layout(geom.Rect{X: 2, Y: 1, W: 10, H: 1})
	r.Add(d)
	r.Dispatch(pressAt(3, 1))
	r.Dispatch(releaseAt(3, 1))
	if r.Overlay() == nil {
		t.Fatal("not open")
	}
	// 点远处空白 → overlay 收起
	r.Dispatch(pressAt(30, 15))
	r.Dispatch(releaseAt(30, 15))
	if r.Overlay() != nil {
		t.Error("outside click must close overlay")
	}
	if d.IsOpen() {
		t.Error("outside click must set IsOpen to false")
	}
	// 再次点击必须能立即重新展开
	r.Dispatch(pressAt(3, 1))
	r.Dispatch(releaseAt(3, 1))
	if r.Overlay() == nil || !d.IsOpen() {
		t.Error("click after outside click must reopen immediately")
	}
}

// 展开后方向键移动，弹层视觉高亮必须同步更新。
func TestDropdownKeyboardVisualHover(t *testing.T) {
	r := mkRoot()
	d := NewDropdown()
	d.SetItems([]string{"a", "b", "c"})
	d.Layout(geom.Rect{X: 2, Y: 1, W: 10, H: 1})
	r.Add(d)
	d.OnFocus()
	r.Dispatch(pressAt(3, 1))
	r.Dispatch(releaseAt(3, 1))
	if !d.IsOpen() {
		t.Fatal("not open")
	}
	r.Dispatch(event.Key{Code: event.KeyDown, Press: true})
	l, ok := r.Overlay().(*dropdownList)
	if !ok {
		t.Fatalf("overlay not dropdownList: %T", r.Overlay())
	}
	if l.Hover() != 1 {
		t.Errorf("down key: visual hover = %d want 1", l.Hover())
	}
}

// 禁用吞事件。
func TestDropdownDisabled(t *testing.T) {
	r := mkRoot()
	d := NewDropdown()
	d.SetItems([]string{"a", "b"})
	d.Layout(geom.Rect{X: 2, Y: 1, W: 10, H: 1})
	r.Add(d)
	d.SetDisabled(true)
	if !d.Handle(pressAt(3, 1)) {
		t.Error("disabled press must swallow")
	}
	if r.Overlay() != nil {
		t.Error("disabled must not open")
	}
}

// Paint golden：关闭态显示当前项 + ▼，聚焦反色。
func TestDropdownPaint(t *testing.T) {
	d := NewDropdown()
	d.SetItems([]string{"alpha", "beta"})
	d.SetSelected(1)
	d.Layout(geom.Rect{X: 1, Y: 1, W: 12, H: 1})
	mc := newMemCanvas(12, 1)
	d.Paint(mc)
	if len(mc.texts) != 1 {
		t.Fatalf("texts = %+v", mc.texts)
	}
	if mc.texts[0].s != "beta ▼      " {
		t.Errorf("closed label = %q want padded to width", mc.texts[0].s)
	}
	d.OnFocus()
	mc2 := newMemCanvas(12, 1)
	d.Paint(mc2)
	if !mc2.texts[0].st.Reverse {
		t.Errorf("focused must be reverse: %+v", mc2.texts[0].st)
	}
}

// 空下拉不崩溃。
func TestDropdownEmpty(t *testing.T) {
	r := mkRoot()
	d := NewDropdown()
	d.Layout(geom.Rect{X: 1, Y: 1, W: 10, H: 1})
	r.Add(d)
	d.Handle(pressAt(2, 1))
	if r.Overlay() != nil {
		t.Error("empty list must not open")
	}
	mc := newMemCanvas(10, 1)
	d.Paint(mc)
	_ = layout.AlignStart
}
