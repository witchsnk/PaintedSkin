package layout

import (
	"testing"

	"PaintedSkin/geom"
)

// 约束钳制：落在 [Min,Max] 内；Min>Max 时以裁剪优先（Max 胜）；永不返回负尺寸。
func TestConstraintsClamp(t *testing.T) {
	c := Constraints{MinW: 1, MinH: 2, MaxW: 5, MaxH: 6}
	if got := c.Clamp(geom.Size{W: 0, H: 0}); got != (geom.Size{W: 1, H: 2}) {
		t.Errorf("Clamp small = %+v", got)
	}
	if got := c.Clamp(geom.Size{W: 9, H: 9}); got != (geom.Size{W: 5, H: 6}) {
		t.Errorf("Clamp big = %+v", got)
	}
	if got := c.Clamp(geom.Size{W: 3, H: 4}); got != (geom.Size{W: 3, H: 4}) {
		t.Errorf("Clamp mid = %+v", got)
	}
	// Min > Max 冲突：裁剪优先，Max 胜出
	bad := Constraints{MinW: 8, MinH: 8, MaxW: 4, MaxH: 4}
	if got := bad.Clamp(geom.Size{W: 6, H: 6}); got != (geom.Size{W: 4, H: 4}) {
		t.Errorf("Clamp conflict = %+v", got)
	}
	// 永不为负
	neg := Constraints{}.Clamp(geom.Size{W: -3, H: -1})
	if neg.W < 0 || neg.H < 0 {
		t.Errorf("Clamp negative = %+v", neg)
	}
}

// Flex 行：理想尺寸原样分配。
func TestFlexRowIdeal(t *testing.T) {
	area := geom.Rect{X: 0, Y: 0, W: 20, H: 5}
	items := []FlexItem{
		{Pref: geom.Size{W: 4, H: 2}},
		{Pref: geom.Size{W: 6, H: 2}},
	}
	rects := FlexRow(area, items, AlignStart, 0)
	if len(rects) != 2 {
		t.Fatalf("len = %d", len(rects))
	}
	if rects[0] != (geom.Rect{X: 0, Y: 0, W: 4, H: 2}) {
		t.Errorf("r0 = %+v", rects[0])
	}
	if rects[1] != (geom.Rect{X: 4, Y: 0, W: 6, H: 2}) {
		t.Errorf("r1 = %+v", rects[1])
	}
}

// Flex 行：主轴有剩余 → 按 Grow 权重分配。
func TestFlexRowGrow(t *testing.T) {
	area := geom.Rect{X: 0, Y: 0, W: 20, H: 5}
	items := []FlexItem{
		{Pref: geom.Size{W: 4, H: 1}, Grow: 1},
		{Pref: geom.Size{W: 4, H: 1}, Grow: 3},
	}
	rects := FlexRow(area, items, AlignStart, 0)
	if rects[0].W != 7 || rects[1].W != 13 {
		t.Errorf("grow = %+v", rects)
	}
	if rects[0].X != 0 || rects[1].X != 7 {
		t.Errorf("grow pos = %+v", rects)
	}
}

// Flex 行：主轴不足 → 先按 Shrink 压缩到 Min。
func TestFlexRowShrink(t *testing.T) {
	area := geom.Rect{X: 0, Y: 0, W: 10, H: 5}
	items := []FlexItem{
		{Min: geom.Size{W: 2}, Pref: geom.Size{W: 6, H: 1}, Shrink: 1},
		{Min: geom.Size{W: 2}, Pref: geom.Size{W: 6, H: 1}, Shrink: 1},
	}
	rects := FlexRow(area, items, AlignStart, 0)
	if rects[0].W != 5 || rects[1].W != 5 {
		t.Errorf("shrink = %+v", rects)
	}
}

// Flex 行：压到 Min 仍不够 → 裁剪保底（不越界、不崩溃）。
func TestFlexRowClip(t *testing.T) {
	area := geom.Rect{X: 0, Y: 0, W: 4, H: 5}
	items := []FlexItem{
		{Min: geom.Size{W: 4}, Pref: geom.Size{W: 6, H: 1}, Shrink: 1},
		{Min: geom.Size{W: 4}, Pref: geom.Size{W: 6, H: 1}, Shrink: 1},
	}
	rects := FlexRow(area, items, AlignStart, 0)
	for i, r := range rects {
		if r.X < area.X || r.Right() > area.Right() {
			t.Errorf("r%d out of area: %+v", i, r)
		}
		if r.W < 0 {
			t.Errorf("r%d negative: %+v", i, r)
		}
	}
	if rects[0].W+rects[1].W != 4 {
		t.Errorf("clip widths = %+v", rects)
	}
}

// Flex 列：交叉轴对齐 Center。
func TestFlexColumnGrowAlign(t *testing.T) {
	area := geom.Rect{X: 0, Y: 0, W: 10, H: 10}
	items := []FlexItem{
		{Pref: geom.Size{W: 4, H: 2}, Grow: 1},
		{Pref: geom.Size{W: 4, H: 3}, Grow: 1},
	}
	rects := FlexColumn(area, items, AlignCenter, 0)
	if len(rects) != 2 {
		t.Fatalf("len = %d", len(rects))
	}
	// 主轴 2+3=5，剩余 5 按 Grow 均分 → 5,5（含最大余数法）
	if rects[0].H != 5 || rects[1].H != 5 {
		t.Errorf("col heights = %+v", rects)
	}
	// 交叉轴居中：(10-4)/2=3
	if rects[0].X != 3 || rects[1].X != 3 {
		t.Errorf("col cross = %+v", rects)
	}
	if rects[0].Y != 0 || rects[1].Y != 5 {
		t.Errorf("col pos = %+v", rects)
	}
}

// Flex 行交叉轴：Start/Center/End/Stretch。
func TestFlexRowCrossAlign(t *testing.T) {
	area := geom.Rect{X: 0, Y: 0, W: 10, H: 10}
	mk := func() []FlexItem { return []FlexItem{{Pref: geom.Size{W: 2, H: 4}}} }
	if r := FlexRow(area, mk(), AlignStart, 0)[0]; r.Y != 0 || r.H != 4 {
		t.Errorf("start = %+v", r)
	}
	if r := FlexRow(area, mk(), AlignCenter, 0)[0]; r.Y != 3 || r.H != 4 {
		t.Errorf("center = %+v", r)
	}
	if r := FlexRow(area, mk(), AlignEnd, 0)[0]; r.Y != 6 || r.H != 4 {
		t.Errorf("end = %+v", r)
	}
	if r := FlexRow(area, mk(), AlignStretch, 0)[0]; r.Y != 0 || r.H != 10 {
		t.Errorf("stretch = %+v", r)
	}
}

// Stack 四式：左上 / 居中 / 右下 / 填充。
func TestStackAlign(t *testing.T) {
	area := geom.Rect{X: 0, Y: 0, W: 20, H: 10}
	want := geom.Size{W: 6, H: 4}
	if r := Stack(area, want, StackTopLeft); r != (geom.Rect{X: 0, Y: 0, W: 6, H: 4}) {
		t.Errorf("topleft = %+v", r)
	}
	if r := Stack(area, want, StackCenter); r != (geom.Rect{X: 7, Y: 3, W: 6, H: 4}) {
		t.Errorf("center = %+v", r)
	}
	if r := Stack(area, want, StackBottomRight); r != (geom.Rect{X: 14, Y: 6, W: 6, H: 4}) {
		t.Errorf("bottomright = %+v", r)
	}
	if r := Stack(area, want, StackFill); r != area {
		t.Errorf("fill = %+v", r)
	}
	// 超大子项裁剪保底
	if r := Stack(area, geom.Size{W: 99, H: 99}, StackCenter); r != area {
		t.Errorf("clip = %+v", r)
	}
}

// Padding + Border 计入父约束。
func TestContentRect(t *testing.T) {
	outer := geom.Rect{X: 0, Y: 0, W: 20, H: 10}
	got := ContentRect(outer, geom.Insets{Top: 1, Right: 2, Bottom: 1, Left: 2}, 1)
	want := geom.Rect{X: 3, Y: 2, W: 14, H: 6}
	if got != want {
		t.Errorf("content = %+v want %+v", got, want)
	}
}

// 滚动偏移钳制在 [0, content-view]。
func TestClampScroll(t *testing.T) {
	if v := ClampScroll(5, 20, 10); v != 5 {
		t.Errorf("mid = %d", v)
	}
	if v := ClampScroll(-3, 20, 10); v != 0 {
		t.Errorf("neg = %d", v)
	}
	if v := ClampScroll(99, 20, 10); v != 10 {
		t.Errorf("over = %d", v)
	}
	if v := ClampScroll(3, 5, 10); v != 0 {
		t.Errorf("small content = %d", v)
	}
}
