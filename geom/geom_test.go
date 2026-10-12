package geom

import "testing"

func TestRectBasics(t *testing.T) {
	r := Rect{X: 3, Y: 4, W: 10, H: 5}
	if got := r.Right(); got != 13 {
		t.Errorf("Right()=%d want 13", got)
	}
	if got := r.Bottom(); got != 9 {
		t.Errorf("Bottom()=%d want 9", got)
	}
	if !r.Contains(Point{X: 3, Y: 4}) {
		t.Error("top-left must be contained")
	}
	if !r.Contains(Point{X: 12, Y: 8}) {
		t.Error("bottom-right inner cell must be contained")
	}
	if r.Contains(Point{X: 13, Y: 4}) {
		t.Error("x==Right must be excluded")
	}
	if r.Contains(Point{X: 3, Y: 9}) {
		t.Error("y==Bottom must be excluded")
	}
	if r.Contains(Point{X: 2, Y: 4}) {
		t.Error("x<Left must be excluded")
	}
}

func TestRectIntersect(t *testing.T) {
	a := Rect{X: 0, Y: 0, W: 10, H: 10}
	b := Rect{X: 5, Y: 5, W: 10, H: 10}
	got := a.Intersect(b)
	want := Rect{X: 5, Y: 5, W: 5, H: 5}
	if got != want {
		t.Errorf("Intersect=%v want %v", got, want)
	}

	c := Rect{X: 20, Y: 20, W: 5, H: 5}
	if got := a.Intersect(c); !got.Empty() {
		t.Errorf("disjoint intersect must be empty, got %v", got)
	}
}

func TestRectInflate(t *testing.T) {
	r := Rect{X: 4, Y: 4, W: 10, H: 10}
	got := r.Inflate(Insets{Top: 1, Right: 2, Bottom: 3, Left: 4})
	want := Rect{X: 8, Y: 5, W: 4, H: 6}
	if got != want {
		t.Errorf("Inflate=%v want %v", got, want)
	}
	if !got.Valid() {
		t.Errorf("positive inset on 10x10 must remain valid, got %v", got)
	}

	// 过大内边距必须钳制为 empty 而不是负尺寸。
	heavy := r.Inflate(Insets{Top: 6, Right: 6, Bottom: 6, Left: 6})
	if !heavy.Empty() {
		t.Errorf("over-inflated rect must be empty, got %v", heavy)
	}
	if heavy.W < 0 || heavy.H < 0 {
		t.Errorf("empty rect must not report negative size, got %v", heavy)
	}
}

func TestRectUnion(t *testing.T) {
	a := Rect{X: 0, Y: 0, W: 5, H: 5}
	b := Rect{X: 10, Y: 10, W: 5, H: 5}
	got := a.Union(b)
	want := Rect{X: 0, Y: 0, W: 15, H: 15}
	if got != want {
		t.Errorf("Union=%v want %v", got, want)
	}
}

func TestSize(t *testing.T) {
	if (Size{W: 3, H: 4}).Area() != 12 {
		t.Error("Area wrong")
	}
	if !(Size{}).Empty() {
		t.Error("zero size must be empty")
	}
	if (Size{W: 1, H: 1}).Empty() {
		t.Error("1x1 must not be empty")
	}
}
