package paintedskin

import (
	"testing"

	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/widget"
)

// click 生成一次按下/释放。
func click(x, y int, down bool) event.Mouse {
	k := event.MouseRelease
	if down {
		k = event.MousePress
	}
	return event.Mouse{X: x, Y: y, Kind: k, Button: event.ButtonLeft}
}

func boundsOf(w widget.Widget) geom.Rect {
	if b, ok := w.(interface{ BoundsOf() geom.Rect }); ok {
		return b.BoundsOf()
	}
	return geom.Rect{}
}

// Label 返回可直接使用的 Label。
func TestSugarLabel(t *testing.T) {
	if Label("hi").Text() != "hi" {
		t.Errorf("text = %q", Label("hi").Text())
	}
}

// Button 带回调。
func TestSugarButton(t *testing.T) {
	n := 0
	b := Button("ok", func() { n++ })
	b.Layout(geom.Rect{X: 0, Y: 0, W: 8, H: 1})
	b.Handle(click(1, 0, true))
	b.Handle(click(1, 0, false))
	if n != 1 {
		t.Errorf("clicks = %d", n)
	}
}

// Column 纵向排布：子项依次下移。
func TestSugarColumn(t *testing.T) {
	c := Column(Label("a"), Label("b"), Label("c"))
	c.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 10})
	kids := c.Children()
	if len(kids) != 3 {
		t.Fatalf("children = %d", len(kids))
	}
	for i, wantY := range []int{0, 1, 2} {
		if got := boundsOf(kids[i]).Y; got != wantY {
			t.Errorf("child %d Y = %d want %d", i, got, wantY)
		}
	}
}

// Row 横向排布：子项依次右移。
func TestSugarRow(t *testing.T) {
	r := Row(Label("aa"), Label("bbb"))
	r.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 5})
	kids := r.Children()
	if len(kids) != 2 {
		t.Fatalf("children = %d", len(kids))
	}
	if got := boundsOf(kids[0]).W; got != 2 {
		t.Errorf("col0 w = %d want 2", got)
	}
	if got := boundsOf(kids[1]).X; got != 2 {
		t.Errorf("col1 X = %d want 2", got)
	}
}

// Column 与 Row 必须为其子控件正确挂接 Parent 链。
func TestColumnAndRowSetParent(t *testing.T) {
	b1 := Button("1", nil)
	b2 := Button("2", nil)
	col := Column(b1)
	col.Add(b2)
	for i, w := range col.Children() {
		p, ok := w.(interface{ ParentWidget() widget.Widget })
		if !ok || p.ParentWidget() != col {
			t.Errorf("col child %d ParentWidget = %v want col", i, p.ParentWidget())
		}
	}

	b3 := Button("3", nil)
	b4 := Button("4", nil)
	row := Row(b3)
	row.Add(b4)
	for i, w := range row.Children() {
		p, ok := w.(interface{ ParentWidget() widget.Widget })
		if !ok || p.ParentWidget() != row {
			t.Errorf("row child %d ParentWidget = %v want row", i, p.ParentWidget())
		}
	}
}

// Grow 让某项占据剩余空间。
func TestSugarColumnGrow(t *testing.T) {
	c := Column(Label("a"), Grow(1, Label("fill")))
	c.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 10})
	kids := c.Children()
	if len(kids) != 2 {
		t.Fatalf("children = %d", len(kids))
	}
	if got := boundsOf(kids[1]).H; got != 9 {
		t.Errorf("grown child H = %d want 9", got)
	}
}

// Gap 设置兄弟间距。
func TestSugarGap(t *testing.T) {
	c := Column(Gap(2, Label("a"), Label("b")))
	c.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 10})
	kids := c.Children()
	if got := boundsOf(kids[1]).Y; got != 3 {
		t.Errorf("row1 Y = %d want 3 (gap 2)", got)
	}
}

// Centered 让子项在交叉轴（垂直）居中。
func TestSugarCentered(t *testing.T) {
	c := Row(Centered(Label("ab"))) // 高 1
	c.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 5})
	k := boundsOf(c.Children()[0])
	if k.Y != 2 {
		t.Errorf("centered Y = %d want 2", k.Y)
	}
	// Column 的 Centered 则在水平方向居中
	cc := Column(Centered(Label("ab"))) // 宽 2
	cc.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 5})
	if got := boundsOf(cc.Children()[0]).X; got != 4 {
		t.Errorf("column centered X = %d want 4", got)
	}
}

// Panel 带边框与内容。
func TestSugarPanel(t *testing.T) {
	p := Panel("title", Label("in"))
	p.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 6})
	if p.Title() != "title" {
		t.Errorf("title = %q", p.Title())
	}
	kids := p.Children()
	if len(kids) != 1 {
		t.Fatalf("panel children = %d", len(kids))
	}
	// 边框 1 + padding 1 + 标题行 1 → 内容从 (2,3) 起
	if got := boundsOf(kids[0]); got.X != 2 || got.Y != 3 {
		t.Errorf("content = %+v want x=2 y=3 (border 1 + padding 1 + title 1)", got)
	}
}

// StatusBar / Separator 等包装同样可用。
func TestSugarMisc(t *testing.T) {
	StatusBar("L", "C", "R")
	Separator()
	Slider()
	TextInput()
	Checkbox("c")
	Dropdown("a", "b")
	List("x", "y")
	if List("x", "y").Len() != 2 {
		t.Error("list len")
	}
	if len(Dropdown("a", "b").Items()) != 2 {
		t.Error("dropdown items")
	}
}

// 容器满足 widget.Widget。
func TestSugarSatisfiesWidget(t *testing.T) {
	var _ widget.Widget = Column()
	var _ widget.Widget = Row()
}
