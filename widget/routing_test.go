package widget

import (
	"testing"

	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/widget/layout"
)

// group 是冒泡测试用的多子容器桩。
type group struct {
	Base
	kids []Widget
	got  []event.Event
}

func (g *group) add(w Widget) {
	if g.self == nil {
		g.Base = NewBase()
		g.Bind(g)
	}
	g.kids = append(g.kids, w)
	if sp, ok := w.(interface{ SetParent(Widget) }); ok {
		sp.SetParent(g)
	}
}

func (g *group) Measure(c layout.Constraints) geom.Size { return geom.Size{} }

func (g *group) Children() []Widget { return g.kids }

func (g *group) HitTest(p geom.Point) Widget {
	if !g.Visible || !g.Bounds.Contains(p) {
		return nil
	}
	for i := len(g.kids) - 1; i >= 0; i-- {
		if hit := g.kids[i].HitTest(p); hit != nil {
			return hit
		}
	}
	return g
}

func (g *group) Handle(e event.Event) bool {
	g.got = append(g.got, e)
	return false
}

func (g *group) Paint(c screen.Canvas) {}

// z 序：后加入（后绘）的先命中。
func TestHitTopmost(t *testing.T) {
	r := mkRoot()
	a := newStub("a", false, geom.Size{W: 10, H: 10})
	b := newStub("b", false, geom.Size{W: 10, H: 10})
	r.Add(a)
	r.Add(b)
	a.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 10})
	b.Layout(geom.Rect{X: 5, Y: 5, W: 10, H: 10})

	if got := r.HitTest(geom.Point{X: 6, Y: 6}); got != Widget(b) {
		t.Errorf("overlap hit = %v want b", got)
	}
	if got := r.HitTest(geom.Point{X: 1, Y: 1}); got != Widget(a) {
		t.Errorf("hit = %v want a", got)
	}
	if got := r.HitTest(geom.Point{X: 30, Y: 15}); got != nil {
		t.Errorf("empty hit = %v want nil", got)
	}
	// 隐藏控件不可命中
	b.Visible = false
	if got := r.HitTest(geom.Point{X: 6, Y: 6}); got != Widget(a) {
		t.Errorf("hidden hit = %v want a", got)
	}
	b.Visible = true
}

// 捕获：Press 命中的控件在移出边界后仍收到 Move/Release 直到 Release。
func TestCapture(t *testing.T) {
	r := mkRoot()
	a := newStub("a", false, geom.Size{W: 5, H: 5})
	r.Add(a)
	a.Layout(geom.Rect{X: 0, Y: 0, W: 5, H: 5})

	r.Dispatch(pressAt(1, 1))
	r.Dispatch(moveAt(30, 15)) // 移出边界（拖拽）
	found := false
	for _, e := range a.got {
		if m, ok := e.(event.Mouse); ok && m.Kind == event.MouseMove {
			found = true
		}
	}
	if !found {
		t.Errorf("captured widget must get outside Move: %+v", a.got)
	}
	if a.left != 0 {
		t.Errorf("no Leave during capture: %d", a.left)
	}
	r.Dispatch(releaseAt(30, 15))
	found = false
	for _, e := range a.got {
		if m, ok := e.(event.Mouse); ok && m.Kind == event.MouseRelease {
			found = true
		}
	}
	if !found {
		t.Errorf("captured widget must get outside Release: %+v", a.got)
	}
	n := len(a.got)
	r.Dispatch(moveAt(30, 15)) // 捕获已结束：不再路由
	if len(a.got) != n {
		t.Errorf("after release no more events: %+v", a.got[n:])
	}
}

// 冒泡：子不消费 → 父收到；子消费 → 父收不到。
func TestBubble(t *testing.T) {
	r := mkRoot()
	g := &group{}
	child := newStub("child", false, geom.Size{W: 5, H: 1})
	g.add(child)
	r.Add(g)
	g.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 10})
	child.Layout(geom.Rect{X: 2, Y: 2, W: 5, H: 1})

	child.consume = false
	r.Dispatch(pressAt(2, 2))
	if len(g.got) == 0 {
		t.Errorf("parent must receive bubbled press")
	}

	child.consume = true
	g.got = nil
	r.Dispatch(pressAt(2, 2))
	if len(g.got) != 0 {
		t.Errorf("consumed event must not bubble")
	}
}

// hover：Enter/Leave 序列；按下期间不发 Leave。
func TestHoverEnterLeave(t *testing.T) {
	r := mkRoot()
	a := newStub("a", false, geom.Size{W: 5, H: 5})
	b := newStub("b", false, geom.Size{W: 5, H: 5})
	r.Add(a)
	r.Add(b)
	a.Layout(geom.Rect{X: 0, Y: 0, W: 5, H: 5})
	b.Layout(geom.Rect{X: 10, Y: 0, W: 5, H: 5})

	r.Dispatch(moveAt(1, 1))
	if a.entered != 1 {
		t.Errorf("a.entered = %d", a.entered)
	}
	r.Dispatch(moveAt(11, 1))
	if a.left != 1 || b.entered != 1 {
		t.Errorf("a.left=%d b.entered=%d", a.left, b.entered)
	}
	r.Dispatch(moveAt(30, 15))
	if b.left != 1 {
		t.Errorf("b.left = %d", b.left)
	}
}

// 模态：Dialog 打开后栈下控件收不到鼠标键盘。
func TestModalShield(t *testing.T) {
	r := mkRoot()
	a := newStub("a", true, geom.Size{W: 5, H: 1})
	r.Add(a)
	a.Layout(geom.Rect{X: 0, Y: 0, W: 5, H: 1})
	d := NewDialog("T", "msg", []string{"OK"}, nil)
	r.PushDialog(d)

	n := len(a.got)
	r.Dispatch(pressAt(1, 0))
	r.Dispatch(releaseAt(1, 0))
	if len(a.got) != n {
		t.Errorf("modal must shield below: %+v", a.got[n:])
	}
	kbd := len(a.got)
	r.Dispatch(event.Key{Code: event.KeyRune, Rune: 'x', Press: true})
	if len(a.got) != kbd {
		t.Errorf("modal must shield kbd: %+v", a.got[kbd:])
	}
}

// 覆盖层优先命中（M6 Dropdown 的机制，M5 只建机制）。
func TestOverlayPriority(t *testing.T) {
	r := mkRoot()
	a := newStub("a", false, geom.Size{W: 20, H: 20})
	o := newStub("o", false, geom.Size{W: 20, H: 20})
	r.Add(a)
	a.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 20})
	o.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 20})
	r.SetOverlay(o)
	if got := r.HitTest(geom.Point{X: 1, Y: 1}); got != Widget(o) {
		t.Errorf("overlay hit = %v want o", got)
	}
}
