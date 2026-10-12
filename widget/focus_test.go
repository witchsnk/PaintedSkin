package widget

import (
	"testing"

	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/widget/layout"
)

// stub 是路由/焦点测试用的最小控件：行为全由字段开关控制。
type stub struct {
	Base
	name     string
	consume  bool
	wantSize geom.Size
	got      []event.Event
	entered  int
	left     int
}

func newStub(name string, focusable bool, size geom.Size) *stub {
	s := &stub{name: name, wantSize: size}
	s.Visible = true
	s.Enabled = true
	s.Focusable = focusable
	s.Bind(s)
	return s
}

func (s *stub) Measure(c layout.Constraints) geom.Size { return c.Clamp(s.wantSize) }

func (s *stub) Handle(e event.Event) bool {
	if m, ok := e.(event.Mouse); ok {
		switch m.Kind {
		case event.MouseEnter:
			s.entered++
			return true
		case event.MouseLeave:
			s.left++
			return true
		}
	}
	s.got = append(s.got, e)
	return s.consume
}

func (s *stub) Paint(c screen.Canvas) {
	b := s.Bounds
	if b.Empty() {
		return
	}
	c.Fill(b, '.', s.Style)
}

func mkRoot() *Root {
	r := NewRoot()
	r.Layout(geom.Rect{X: 0, Y: 0, W: 40, H: 20})
	return r
}

func pressAt(x, y int) event.Mouse {
	return event.Mouse{X: x, Y: y, Kind: event.MousePress, Button: event.ButtonLeft}
}

func releaseAt(x, y int) event.Mouse {
	return event.Mouse{X: x, Y: y, Kind: event.MouseRelease, Button: event.ButtonLeft}
}

func moveAt(x, y int) event.Mouse {
	return event.Mouse{X: x, Y: y, Kind: event.MouseMove, Button: event.ButtonNone}
}

func tabKey(shift bool) event.Key {
	m := event.ModNone
	if shift {
		m = event.ModShift
	}
	return event.Key{Code: event.KeyTab, Mod: m, Press: true}
}

// Tab 按树序成环遍历。
func TestFocusTabOrder(t *testing.T) {
	r := mkRoot()
	a, b, c := newStub("a", true, geom.Size{W: 5, H: 1}), newStub("b", true, geom.Size{W: 5, H: 1}), newStub("c", true, geom.Size{W: 5, H: 1})
	r.Add(a)
	r.Add(b)
	r.Add(c)
	a.Layout(geom.Rect{X: 0, Y: 0, W: 5, H: 1})
	b.Layout(geom.Rect{X: 0, Y: 1, W: 5, H: 1})
	c.Layout(geom.Rect{X: 0, Y: 2, W: 5, H: 1})

	if r.Focused() != nil {
		t.Fatalf("initial focus = %v", r.Focused())
	}
	r.Dispatch(tabKey(false))
	if r.Focused() != Widget(a) {
		t.Errorf("tab1 = %v want a", r.Focused())
	}
	r.Dispatch(tabKey(false))
	if r.Focused() != Widget(b) {
		t.Errorf("tab2 = %v want b", r.Focused())
	}
	r.Dispatch(tabKey(false))
	if r.Focused() != Widget(c) {
		t.Errorf("tab3 = %v want c", r.Focused())
	}
	r.Dispatch(tabKey(false))
	if r.Focused() != Widget(a) {
		t.Errorf("wrap = %v want a", r.Focused())
	}
	r.Dispatch(tabKey(true))
	if r.Focused() != Widget(c) {
		t.Errorf("shift-tab = %v want c", r.Focused())
	}
}

// 禁用/隐藏控件跳过遍历；聚焦变化打脏旧新双方。
func TestFocusSkipsDisabledInvisible(t *testing.T) {
	r := mkRoot()
	a := newStub("a", true, geom.Size{W: 5, H: 1})
	b := newStub("b", true, geom.Size{W: 5, H: 1})
	c := newStub("c", true, geom.Size{W: 5, H: 1})
	d := newStub("d", true, geom.Size{W: 5, H: 1})
	r.Add(a)
	r.Add(b)
	r.Add(c)
	r.Add(d)
	b.Enabled = false
	c.Visible = false

	r.Dispatch(tabKey(false))
	if r.Focused() != Widget(a) {
		t.Fatalf("tab1 = %v want a", r.Focused())
	}
	a.Dirty, d.Dirty = false, false
	r.Dispatch(tabKey(false))
	if r.Focused() != Widget(d) {
		t.Errorf("tab2 = %v want d", r.Focused())
	}
	if !a.Dirty || !d.Dirty {
		t.Errorf("focus change must dirty old+new: a=%v d=%v", a.Dirty, d.Dirty)
	}
}

// 点击聚焦：可聚焦夺取焦点；点不可聚焦/空白焦点不变。
func TestClickFocus(t *testing.T) {
	r := mkRoot()
	a := newStub("a", true, geom.Size{W: 5, H: 1})
	lbl := newStub("lbl", false, geom.Size{W: 5, H: 1})
	r.Add(a)
	r.Add(lbl)
	a.Layout(geom.Rect{X: 0, Y: 0, W: 5, H: 1})
	lbl.Layout(geom.Rect{X: 0, Y: 2, W: 5, H: 1})

	r.Dispatch(pressAt(1, 0))
	if r.Focused() != Widget(a) {
		t.Errorf("press a: focus = %v", r.Focused())
	}
	r.Dispatch(releaseAt(1, 0))
	r.Dispatch(pressAt(1, 2))
	if r.Focused() != Widget(a) {
		t.Errorf("press non-focusable must not steal: %v", r.Focused())
	}
	r.Dispatch(releaseAt(1, 2))
	r.Dispatch(pressAt(30, 15))
	if r.Focused() != Widget(a) {
		t.Errorf("press empty must not drop: %v", r.Focused())
	}
	r.Dispatch(releaseAt(30, 15))
}

// Dialog 焦点陷阱：打开后 Tab 只在 Dialog 内循环，关闭后恢复原焦点。
func TestDialogTrapRestore(t *testing.T) {
	r := mkRoot()
	a := newStub("a", true, geom.Size{W: 5, H: 1})
	r.Add(a)
	a.Layout(geom.Rect{X: 0, Y: 0, W: 5, H: 1})
	r.Dispatch(tabKey(false))
	if r.Focused() != Widget(a) {
		t.Fatalf("pre focus = %v", r.Focused())
	}

	d := NewDialog("T", "msg", []string{"OK", "Cancel"}, nil)
	r.PushDialog(d)
	if r.Focused() == Widget(a) {
		t.Fatalf("dialog must move focus inside")
	}
	first := r.Focused()
	r.Dispatch(tabKey(false))
	second := r.Focused()
	if second == Widget(a) || second == nil {
		t.Fatalf("trap escaped: %v", second)
	}
	r.Dispatch(tabKey(false))
	if r.Focused() != first {
		t.Errorf("trap ring broken: %v want %v", r.Focused(), first)
	}
	_ = second
	r.PopDialog()
	if r.Focused() != Widget(a) {
		t.Errorf("pop must restore: %v want a", r.Focused())
	}
}
