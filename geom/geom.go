// Package geom 提供终端单元格坐标系下的几何类型。
// 所有坐标以"格"为单位，X 向右增长，Y 向下增长，(0,0) 为左上角。
package geom

// Point 是一个单元格坐标。
type Point struct {
	X, Y int
}

// Size 是以单元格为单位的尺寸。
type Size struct {
	W, H int
}

// Area 返回面积（格数）。
func (s Size) Area() int { return s.W * s.H }

// Empty 报告尺寸是否为零或负。
func (s Size) Empty() bool { return s.W <= 0 || s.H <= 0 }

// Rect 是一个轴对齐矩形：(X,Y) 为左上角，W/H 为宽高。
// 约定右边界 Right() 与下边界 Bottom() 为开区间，即 [X, X+W) × [Y, Y+H)。
type Rect struct {
	X, Y, W, H int
}

// Left 返回左边界（含）。
func (r Rect) Left() int { return r.X }

// Top 返回上边界（含）。
func (r Rect) Top() int { return r.Y }

// Right 返回右边界（不含）。
func (r Rect) Right() int { return r.X + r.W }

// Bottom 返回下边界（不含）。
func (r Rect) Bottom() int { return r.Y + r.H }

// Size 返回矩形尺寸。
func (r Rect) Size() Size { return Size{W: r.W, H: r.H} }

// Empty 报告矩形是否不含任何单元格。
func (r Rect) Empty() bool { return r.W <= 0 || r.H <= 0 }

// Valid 报告矩形尺寸是否非负。
func (r Rect) Valid() bool { return r.W >= 0 && r.H >= 0 }

// Contains 报告 p 是否落在矩形内（右/下开区间）。
func (r Rect) Contains(p Point) bool {
	return p.X >= r.X && p.X < r.Right() && p.Y >= r.Y && p.Y < r.Bottom()
}

// Intersect 返回两矩形交集；无交集时返回零尺寸矩形（Empty）。
func (r Rect) Intersect(o Rect) Rect {
	x1 := max(r.X, o.X)
	y1 := max(r.Y, o.Y)
	x2 := min(r.Right(), o.Right())
	y2 := min(r.Bottom(), o.Bottom())
	if x2 <= x1 || y2 <= y1 {
		return Rect{X: x1, Y: y1}
	}
	return Rect{X: x1, Y: y1, W: x2 - x1, H: y2 - y1}
}

// Union 返回能同时包含两者的最小矩形；任一方为空时返回另一方。
func (r Rect) Union(o Rect) Rect {
	if r.Empty() {
		return o
	}
	if o.Empty() {
		return r
	}
	x1 := min(r.X, o.X)
	y1 := min(r.Y, o.Y)
	x2 := max(r.Right(), o.Right())
	y2 := max(r.Bottom(), o.Bottom())
	return Rect{X: x1, Y: y1, W: x2 - x1, H: y2 - y1}
}

// Inflate 按内边距收缩矩形（左内边距把左边界右移）。
// 收缩过量时尺寸钳制为 0，绝不返回负宽高。
func (r Rect) Inflate(in Insets) Rect {
	w := r.W - in.Left - in.Right
	h := r.H - in.Top - in.Bottom
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return Rect{X: r.X + in.Left, Y: r.Y + in.Top, W: w, H: h}
}

// Insets 是四边内边距。
type Insets struct {
	Top, Right, Bottom, Left int
}

// SymInsets 返回四边相等的内边距。
func SymInsets(n int) Insets {
	return Insets{Top: n, Right: n, Bottom: n, Left: n}
}

// Horizontal 返回左右之和。
func (in Insets) Horizontal() int { return in.Left + in.Right }

// Vertical 返回上下之和。
func (in Insets) Vertical() int { return in.Top + in.Bottom }
