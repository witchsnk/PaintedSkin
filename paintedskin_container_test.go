package paintedskin

import (
	"testing"

	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/style"
	"PaintedSkin/widget"
)

// 容器（Column / Row / Root）默认**透明**：只有设了非零 Style 才铺自己的底色。
// 叶控件各自铺底（ADR-019 尾部空档纪律），容器不代劳。

// 未设 Style 的容器不得铺底（否则会盖掉终端背景与兄弟控件的观感）。
func TestContainerDefaultTransparent(t *testing.T) {
	c := Column(Label("hi"))
	area := geom.Rect{X: 0, Y: 0, W: 6, H: 2}
	c.Layout(area)
	buf := screen.NewBuffer(6, 2)
	c.Paint(buf)
	// 第 1 行没有任何子控件 → 必须保持未绘制（零值格）
	for x := 0; x < 6; x++ {
		cell, _ := buf.GetCell(x, 1)
		if cell.Rune != 0 {
			t.Errorf("transparent container painted %q at (x=%d,y=1)", cell.Rune, x)
		}
	}
}

// 设了 Style 的容器必须铺满自己的区域。
func TestContainerFillsWhenStyled(t *testing.T) {
	bg := style.Style{}.WithBackground(style.ANSIColor(238))
	c := Column(Label("hi"))
	c.SetStyle(bg)
	area := geom.Rect{X: 0, Y: 0, W: 6, H: 2}
	c.Layout(area)
	buf := screen.NewBuffer(6, 2)
	c.Paint(buf)
	for y := 0; y < 2; y++ {
		for x := 0; x < 6; x++ {
			cell, _ := buf.GetCell(x, y)
			if cell.Style.BG != bg.BG {
				t.Errorf("(%d,%d) bg = %+v want container background", x, y, cell.Style.BG)
			}
		}
	}
}

// Row 同样遵守。
func TestRowFillsWhenStyled(t *testing.T) {
	bg := style.Style{}.WithBackground(style.ANSIColor(240))
	r := Row(widget.NewButton("a"), widget.NewButton("b"))
	r.SetStyle(bg)
	area := geom.Rect{X: 0, Y: 0, W: 20, H: 1}
	r.Layout(area)
	buf := screen.NewBuffer(20, 1)
	r.Paint(buf)
	for x := 0; x < 20; x++ {
		cell, _ := buf.GetCell(x, 0)
		if cell.Style.BG != bg.BG {
			t.Errorf("x=%d bg = %+v want container background", x, cell.Style.BG)
		}
	}
}

// 嵌套容器：内层透明时外层的底色透出来。
func TestNestedContainerKeepsOuterBackground(t *testing.T) {
	bg := style.Style{}.WithBackground(style.ANSIColor(238))
	inner := Column(Label("x"))
	outer := Column(inner)
	outer.SetStyle(bg)
	area := geom.Rect{X: 0, Y: 0, W: 6, H: 3}
	outer.Layout(area)
	buf := screen.NewBuffer(6, 3)
	outer.Paint(buf)
	for y := 0; y < 3; y++ {
		for x := 0; x < 6; x++ {
			cell, _ := buf.GetCell(x, y)
			if cell.Style.BG != bg.BG {
				t.Errorf("(%d,%d) bg = %+v want outer background to show through", x, y, cell.Style.BG)
			}
		}
	}
}

// Root 设了 Style 时铺满整屏。
func TestRootFillsWhenStyled(t *testing.T) {
	bg := style.Style{}.WithBackground(style.ANSIColor(17))
	r := widget.NewRoot()
	r.SetStyle(bg)
	area := geom.Rect{X: 0, Y: 0, W: 8, H: 4}
	r.Layout(area)
	buf := screen.NewBuffer(8, 4)
	r.Paint(buf)
	for y := 0; y < 4; y++ {
		for x := 0; x < 8; x++ {
			cell, _ := buf.GetCell(x, y)
			if cell.Style.BG != bg.BG {
				t.Errorf("(%d,%d) bg = %+v want root background", x, y, cell.Style.BG)
			}
		}
	}
}

// Root 未设 Style 时保持透明。
func TestRootDefaultTransparent(t *testing.T) {
	r := widget.NewRoot()
	r.Layout(geom.Rect{X: 0, Y: 0, W: 4, H: 2})
	buf := screen.NewBuffer(4, 2)
	r.Paint(buf)
	cell, _ := buf.GetCell(3, 1)
	if cell.Rune != 0 {
		t.Errorf("transparent root painted %q", cell.Rune)
	}
}
