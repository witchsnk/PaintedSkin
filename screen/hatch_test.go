package screen

import (
	"testing"

	"PaintedSkin/geom"
	"PaintedSkin/style"
)

// P3：inner / outer 边框（半块"贴边"粗线）与 hatch 底纹。

// inner / outer 两套半块边框字形（上下边字形不同，左右边也不同）。
func TestBorderInnerOuterGlyphs(t *testing.T) {
	cases := []struct {
		name   string
		border Border
		want   BorderGlyphs
	}{
		{"inner", BorderInner, BorderGlyphs{
			{'▗', '▄', '▖'},
			{'▐', ' ', '▌'},
			{'▝', '▀', '▘'},
		}},
		{"outer", BorderOuter, BorderGlyphs{
			{'▛', '▀', '▜'},
			{'▌', ' ', '▐'},
			{'▙', '▄', '▟'},
		}},
	}
	for _, tc := range cases {
		if tc.border.Style != tc.name {
			t.Errorf("style name = %q want %q", tc.border.Style, tc.name)
		}
		if tc.border.Glyphs != tc.want {
			t.Errorf("%s glyphs = %q want %q", tc.name, tc.border.Glyphs, tc.want)
		}
		if !tc.border.Visible() {
			t.Errorf("%s must be visible", tc.name)
		}
		// 上下边字形不同（这正是"内贴 / 外扩"的观感来源）。
		if tc.border.At(0, 1) == tc.border.At(2, 1) {
			t.Errorf("%s: top and bottom edge must differ", tc.name)
		}
		// 左右侧边字形也不同。
		if tc.border.At(1, 0) == tc.border.At(1, 2) {
			t.Errorf("%s: left and right edges must differ", tc.name)
		}
	}
}

// inner / outer 与 thick 一起纳入 AllBorders（可按名查表、可被 Box 使用）。
func TestInnerOuterAreUsable(t *testing.T) {
	for _, b := range []Border{BorderInner, BorderOuter} {
		got, ok := BorderByName(b.Style)
		if !ok || got.Style != b.Style {
			t.Errorf("BorderByName(%q) not found", b.Style)
		}
		buf := NewBuffer(4, 3)
		buf.Box(geom.Rect{X: 0, Y: 0, W: 4, H: 3}, b, style.Default)
		if c, _ := buf.GetCell(0, 0); c.Rune != b.At(0, 0) {
			t.Errorf("%s top-left = %q want %q", b.Style, c.Rune, b.At(0, 0))
		}
	}
	// AllBorders 覆盖全部内置套（含 none 之外的全部）。
	if len(AllBorders) < 9 {
		t.Errorf("AllBorders has %d entries, want >= 9", len(AllBorders))
	}
}

// ---- hatch 底纹 ----

// hatch 必须按固定密度点缀（~20%），不能铺满。
func TestHatchDensity(t *testing.T) {
	const w, h = 20, 10
	b := NewBuffer(w, h)
	b.FillStyle(geom.Rect{X: 0, Y: 0, W: w, H: h}, style.Default)
	b.Hatch(geom.Rect{X: 0, Y: 0, W: w, H: h}, HatchLeft, style.Default)
	n := 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c, _ := b.GetCell(x, y)
			if c.Rune != ' ' {
				n++
			}
		}
	}
	total := w * h
	if n == 0 {
		t.Fatal("hatch drew nothing")
	}
	if n*5 > total {
		t.Errorf("hatch density too high: %d/%d cells (>20%%)", n, total)
	}
	if n*20 < total/2 {
		t.Errorf("hatch density too low: %d/%d cells", n, total)
	}
}

// HatchNone 与未知值不画任何东西。
func TestHatchNoneDrawsNothing(t *testing.T) {
	for _, hh := range []Hatch{HatchNone, Hatch(99)} {
		b := NewBuffer(8, 4)
		b.FillStyle(geom.Rect{X: 0, Y: 0, W: 8, H: 4}, style.Default)
		b.Hatch(geom.Rect{X: 0, Y: 0, W: 8, H: 4}, hh, style.Default)
		for y := 0; y < 4; y++ {
			for x := 0; x < 8; x++ {
				if c, _ := b.GetCell(x, y); c.Rune != ' ' {
					t.Errorf("Hatch(%d) drew %q at (%d,%d)", hh, c.Rune, x, y)
				}
			}
		}
	}
}

// 三种模式用不同字形。
func TestHatchModes(t *testing.T) {
	got := map[Hatch]rune{}
	for _, hh := range []Hatch{HatchLeft, HatchRight, HatchCross} {
		b := NewBuffer(10, 3)
		b.FillStyle(geom.Rect{X: 0, Y: 0, W: 10, H: 3}, style.Default)
		b.Hatch(geom.Rect{X: 0, Y: 0, W: 10, H: 3}, hh, style.Default)
		for y := 0; y < 3; y++ {
			for x := 0; x < 10; x++ {
				c, _ := b.GetCell(x, y)
				if c.Rune != ' ' {
					got[hh] = c.Rune
				}
			}
		}
	}
	if len(got) < 3 {
		t.Errorf("three hatch modes must use distinct glyph families: %v", got)
	}
	for hh, r := range got {
		if r == 0 {
			t.Errorf("Hatch(%d) produced an empty glyph", hh)
		}
	}
}

// hatch 绝不覆盖真实内容。
func TestHatchKeepsContent(t *testing.T) {
	b := NewBuffer(10, 2)
	b.FillStyle(geom.Rect{X: 0, Y: 0, W: 10, H: 2}, style.Default)
	b.Text(0, 0, "abcdefghij", style.Default)
	b.Text(0, 1, "中文中文中文中文", style.Default)
	before := snapshot(b)
	b.Hatch(geom.Rect{X: 0, Y: 0, W: 10, H: 2}, HatchCross, style.Default)
	for y := 0; y < 2; y++ {
		for x := 0; x < 10; x++ {
			bc, _ := before.GetCell(x, y)
			ac, _ := b.GetCell(x, y)
			if bc.Rune != ' ' && ac.Rune != bc.Rune {
				t.Errorf("hatch overwrote content at (%d,%d): %q -> %q", x, y, bc.Rune, ac.Rune)
			}
		}
	}
	assertNoDanglingContinuation(t, b)
}

// hatch 尊重裁剪区与区域边界（界外一格都不能点）。
func TestHatchRespectsClipAndRect(t *testing.T) {
	b := NewBuffer(12, 6)
	b.FillStyle(geom.Rect{X: 0, Y: 0, W: 12, H: 6}, style.Default)
	b.Hatch(geom.Rect{X: 2, Y: 1, W: 4, H: 2}, HatchLeft, style.Default)
	insideHits := 0
	for y := 0; y < 6; y++ {
		for x := 0; x < 12; x++ {
			c, _ := b.GetCell(x, y)
			hatched := c.Rune != ' '
			inside := y >= 1 && y < 3 && x >= 2 && x < 6
			if hatched && !inside {
				t.Errorf("(%d,%d) hatched outside rect: %q", x, y, c.Rune)
			}
			if hatched {
				insideHits++
			}
		}
	}
	if insideHits == 0 {
		t.Error("hatch drew nothing inside the rect")
	}
}

// 退化矩形不 panic。
func TestHatchDegenerate(t *testing.T) {
	b := NewBuffer(6, 4)
	for _, r := range []geom.Rect{
		{X: 0, Y: 0, W: 0, H: 0}, {X: 0, Y: 0, W: 5, H: 0},
		{X: 0, Y: 0, W: 0, H: 3}, {X: 9, Y: 9, W: 3, H: 3},
	} {
		b.Hatch(r, HatchCross, style.Default)
	}
	assertNoDanglingContinuation(t, b)
}
