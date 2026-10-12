package screen

import (
	"testing"

	"PaintedSkin/geom"
	"PaintedSkin/style"
	"PaintedSkin/text"
)

// 边框字形表：3×3 矩阵 + 样式名 + 逐边开关（docs/11 P1）。
// 边框语义层与文本属性层（style.Attr）解耦，字形只由表表达。

// 每套内置边框的 9 个位置字形都被钉住。
func TestBorderGlyphTable(t *testing.T) {
	cases := []struct {
		name   string
		border Border
		want   [3][3]rune
	}{
		{"single", BorderSingle, [3][3]rune{
			{'┌', '─', '┐'},
			{'│', ' ', '│'},
			{'└', '─', '┘'},
		}},
		{"double", BorderDouble, [3][3]rune{
			{'╔', '═', '╗'},
			{'║', ' ', '║'},
			{'╚', '═', '╝'},
		}},
		{"round", BorderRound, [3][3]rune{
			{'╭', '─', '╮'},
			{'│', ' ', '│'},
			{'╰', '─', '╯'},
		}},
		{"heavy", BorderHeavy, [3][3]rune{
			{'┏', '━', '┓'},
			{'┃', ' ', '┃'},
			{'┗', '━', '┛'},
		}},
		{"ascii", BorderAscii, [3][3]rune{
			{'+', '-', '+'},
			{'|', ' ', '|'},
			{'+', '-', '+'},
		}},
		{"dashed", BorderDashed, [3][3]rune{
			{'┌', '╌', '┐'},
			{'╎', ' ', '╎'},
			{'└', '╌', '┘'},
		}},
		{"thick", BorderThick, [3][3]rune{
			{'█', '▀', '█'},
			{'█', ' ', '█'},
			{'█', '▄', '█'},
		}},
	}
	for _, tc := range cases {
		if tc.border.Style != tc.name {
			t.Errorf("style name = %q want %q", tc.border.Style, tc.name)
		}
		if tc.border.Glyphs != tc.want {
			t.Errorf("%s glyphs = %q want %q", tc.name, tc.border.Glyphs, tc.want)
		}
	}
}

// 上下共用同一个横线字形（Textual 的核心技巧：一张表推导 9 个位置）。
// 例外是 BorderThick：半块粗框用 ▀/▄ 表达上下边，天然不同。
func TestBorderTopBottomShareHLine(t *testing.T) {
	for _, b := range []Border{BorderSingle, BorderDouble, BorderRound, BorderHeavy, BorderAscii, BorderDashed} {
		if got := b.At(0, 1); got != b.At(2, 1) {
			t.Errorf("%s: top edge %q != bottom edge %q", b.Style, got, b.At(2, 1))
		}
		if b.HLine() != b.At(0, 1) {
			t.Errorf("%s: HLine = %q want %q", b.Style, b.HLine(), b.At(0, 1))
		}
		if b.VLine() != b.At(1, 0) {
			t.Errorf("%s: VLine = %q want %q", b.Style, b.VLine(), b.At(1, 0))
		}
	}
}

// 无边框（零值）不可见。
func TestBorderNoneInvisible(t *testing.T) {
	if BorderNone.Visible() {
		t.Error("zero-value border must be invisible")
	}
	if !BorderSingle.Visible() {
		t.Error("single border must be visible")
	}
	if (BorderSingle.WithEdges(BorderEdges{})).Visible() {
		t.Error("all edges off must be invisible")
	}
}

// At 越界返回 0。
func TestBorderAtOutOfRange(t *testing.T) {
	for _, p := range [][2]int{{-1, 0}, {3, 0}, {0, -1}, {0, 3}} {
		if got := BorderSingle.At(p[0], p[1]); got != 0 {
			t.Errorf("At%v = %q want 0", p, got)
		}
	}
}

// Box 按字形表绘制 9 个位置。
func TestBoxFollowsGlyphTable(t *testing.T) {
	for _, border := range []Border{BorderSingle, BorderDouble, BorderRound, BorderHeavy, BorderAscii, BorderDashed, BorderThick} {
		b := NewBuffer(5, 3)
		b.Box(geom.Rect{X: 0, Y: 0, W: 5, H: 3}, border, style.Default)
		// 8 个落笔位置（矩阵中心 [1][1] 是内部，Box 不画）。
		pos := [][2]int{{0, 0}, {0, 1}, {0, 2}, {1, 0}, {1, 2}, {2, 0}, {2, 1}, {2, 2}}
		for _, p := range pos {
			row, col := p[0], p[1]
			c, _ := b.GetCell(col*2, row)
			if c.Rune != border.At(row, col) {
				t.Errorf("%s at (row %d,col %d) = %q want %q",
					border.Style, row, col, c.Rune, border.At(row, col))
			}
		}
		// 内部格不被写
		if c, _ := b.GetCell(2, 1); c.Rune != 0 {
			t.Errorf("%s: interior must stay empty, got %q", border.Style, c.Rune)
		}
	}
}

// 逐边开关：只画上边。
func TestBoxEdgeControl(t *testing.T) {
	cases := []struct {
		name  string
		edges BorderEdges
	}{
		{"top", BorderEdges{Top: true}},
		{"bottom", BorderEdges{Bottom: true}},
		{"left", BorderEdges{Left: true}},
		{"right", BorderEdges{Right: true}},
		{"top+bottom", BorderEdges{Top: true, Bottom: true}},
		{"left+right", BorderEdges{Left: true, Right: true}},
		{"all-but-top", BorderEdges{Right: true, Bottom: true, Left: true}},
	}
	for _, tc := range cases {
		b := NewBuffer(4, 3)
		b.Box(geom.Rect{X: 0, Y: 0, W: 4, H: 3}, BorderSingle.WithEdges(tc.edges), style.Default)
		for y := 0; y < 3; y++ {
			for x := 0; x < 4; x++ {
				c, _ := b.GetCell(x, y)
				on := false
				switch {
				case y == 0 && tc.edges.Top:
					on = true
				case y == 2 && tc.edges.Bottom:
					on = true
				case x == 0 && y == 1 && tc.edges.Left:
					on = true
				case x == 3 && y == 1 && tc.edges.Right:
					on = true
				}
				if on && c.Rune == 0 {
					t.Errorf("%s: edge cell (%d,%d) missing", tc.name, x, y)
				}
				if !on && c.Rune != 0 {
					t.Errorf("%s: cell (%d,%d) = %q must stay empty", tc.name, x, y, c.Rune)
				}
			}
		}
	}
}

// 四边全关时不画任何东西。
func TestBoxNoEdgesDrawsNothing(t *testing.T) {
	b := NewBuffer(4, 3)
	b.Box(geom.Rect{X: 0, Y: 0, W: 4, H: 3}, BorderSingle.WithEdges(BorderEdges{}), style.Default)
	for y := 0; y < 3; y++ {
		for x := 0; x < 4; x++ {
			if c, _ := b.GetCell(x, y); c.Rune != 0 {
				t.Errorf("(%d,%d) = %q want untouched", x, y, c.Rune)
			}
		}
	}
}

// 无边框（BorderNone）不画任何东西，且不 panic。
func TestBoxBorderNone(t *testing.T) {
	b := NewBuffer(4, 3)
	b.Box(geom.Rect{X: 0, Y: 0, W: 4, H: 3}, BorderNone, style.Default)
	if c, _ := b.GetCell(0, 0); c.Rune != 0 {
		t.Errorf("BorderNone must draw nothing, got %q", c.Rune)
	}
}

// 逐边开关下的退化尺寸不得 panic 且不越界。
func TestBoxDegenerateWithEdges(t *testing.T) {
	edges := []BorderEdges{
		AllEdges,
		{Top: true},
		{Left: true},
		{Right: true, Bottom: true},
		{},
	}
	for _, e := range edges {
		for _, size := range []geom.Size{
			{W: 1, H: 1}, {W: 6, H: 1}, {W: 1, H: 4},
			{W: 2, H: 2}, {W: 0, H: 3}, {W: 3, H: 0},
		} {
			b := NewBuffer(8, 6)
			b.Box(geom.Rect{X: 0, Y: 0, W: size.W, H: size.H},
				BorderSingle.WithEdges(e), style.Default)
			for y := 0; y < 6; y++ {
				for x := 0; x < 8; x++ {
					c, _ := b.GetCell(x, y)
					if c.IsContinuation() {
						t.Errorf("edges %+v size %+v: dangling continuation at (%d,%d)", e, size, x, y)
					}
				}
			}
		}
	}
}

// 边框与文本属性解耦：同一份 style 下换边框只换字形，不改属性；
// 不同边框可以共用同一套文本属性（这是旧实现做不到的）。
func TestBorderDecoupledFromAttr(t *testing.T) {
	st := style.Style{}.WithBold(true)
	for _, border := range []Border{BorderSingle, BorderDouble, BorderRound, BorderHeavy} {
		b := NewBuffer(4, 3)
		b.Box(geom.Rect{X: 0, Y: 0, W: 4, H: 3}, border, st)
		c, _ := b.GetCell(0, 0)
		if !c.Style.Equal(st) {
			t.Errorf("%s: style = %+v want %+v", border.Style, c.Style, st)
		}
	}
}

// WithEdges 不改动原值（值语义）。
func TestBorderWithEdgesIsValue(t *testing.T) {
	b := BorderSingle.WithEdges(BorderEdges{Top: true})
	if BorderSingle.Edges != AllEdges {
		t.Error("WithEdges must not mutate the receiver")
	}
	if b.Glyphs != BorderSingle.Glyphs {
		t.Error("WithEdges must keep glyphs")
	}
}

// 条形字形表：全框架唯一的块元素词汇（消除字形重复）。
func TestBarGlyphTable(t *testing.T) {
	want := BarGlyphs{
		Track:  '░',
		Filled: '█',
		KnobH:  '>',
		KnobV:  '▐',
		Pad:    '=',
		CapL:   '[',
		CapR:   ']',
	}
	if Bar != want {
		t.Errorf("Bar = %+v want %+v", Bar, want)
	}
	// 所有字形都必须是单列宽（否则会撑破框体）。
	for name, r := range map[string]rune{
		"Track": Bar.Track, "Filled": Bar.Filled, "KnobH": Bar.KnobH,
		"KnobV": Bar.KnobV, "Pad": Bar.Pad, "CapL": Bar.CapL, "CapR": Bar.CapR,
	} {
		if w := text.RuneWidth(r); w != 1 {
			t.Errorf("Bar.%s = %q width %d, want 1", name, r, w)
		}
	}
}
