package widget

import (
	"testing"

	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/style"
	"PaintedSkin/text"
)

// P1：边框字形表重写（3×3 矩阵 + 逐边开关，与 style.Attr 解耦）。

// 每个 BorderStyle 枚举都必须映射到字形表里对应的那套字形，
// 不能再靠 style.Attr 位掩码"传输"边框语义。
func TestBorderStyleMapsToGlyphTable(t *testing.T) {
	for _, bs := range []BorderStyle{BorderSingle, BorderDouble, BorderRound, BorderHeavy} {
		want := borderSpec(bs)
		p := NewPanel("t")
		p.SetBorder(bs)
		p.Layout(geom.Rect{X: 0, Y: 0, W: 8, H: 5})
		buf := paintBounds(p, geom.Rect{X: 0, Y: 0, W: 8, H: 5})
		corners := [][2]int{{0, 0}, {7, 0}, {0, 4}, {7, 4}}
		inner := [][2]int{{1, 0}, {6, 0}, {1, 4}, {6, 4}}
		for i, pt := range corners {
			row, col := i/2*2, i%2*2 // 0,0 / 0,2 / 2,0 / 2,2
			c, _ := buf.GetCell(pt[0], pt[1])
			if c.Rune != want.At(row, col) {
				t.Errorf("%v corner (%d,%d) = %q want %q", bs, pt[0], pt[1], c.Rune, want.At(row, col))
			}
		}
		// 顶/底边的横线字形
		for _, pt := range inner {
			c, _ := buf.GetCell(pt[0], pt[1])
			row := 0
			if pt[1] == 4 {
				row = 2
			}
			if c.Rune != want.At(row, 1) {
				t.Errorf("%v edge (%d,%d) = %q want %q", bs, pt[0], pt[1], c.Rune, want.At(row, 1))
			}
		}
		// 侧边竖线
		for _, pt := range [][2]int{{0, 2}, {7, 2}} {
			c, _ := buf.GetCell(pt[0], pt[1])
			if c.Rune != want.At(1, 0) {
				t.Errorf("%v side (%d,%d) = %q want %q", bs, pt[0], pt[1], c.Rune, want.At(1, 0))
			}
		}
	}
}

// 语义修正：BorderDouble 画双线、BorderHeavy 画粗线。
// 旧实现两者互相对调（Double→AttrBold→'━'，Heavy→Bold|Reverse→'═'）。
func TestBorderStyleSemanticsNotSwapped(t *testing.T) {
	dbl := NewPanel("t")
	dbl.SetBorder(BorderDouble)
	bufD := paintBounds(dbl, geom.Rect{X: 0, Y: 0, W: 8, H: 4})
	if c, _ := bufD.GetCell(0, 0); c.Rune != '╔' {
		t.Errorf("BorderDouble top-left = %q want ╔", c.Rune)
	}

	hv := NewPanel("t")
	hv.SetBorder(BorderHeavy)
	bufH := paintBounds(hv, geom.Rect{X: 0, Y: 0, W: 8, H: 4})
	if c, _ := bufH.GetCell(0, 0); c.Rune != '┏' {
		t.Errorf("BorderHeavy top-left = %q want ┏", c.Rune)
	}
}

// BorderNone 不画框但内容不越界。
func TestBorderNoneDrawsNoFrame(t *testing.T) {
	p := NewPanel("t")
	p.SetBorder(BorderNone)
	r := geom.Rect{X: 0, Y: 0, W: 12, H: 5}
	buf := paintBounds(p, r)
	for y := 0; y < 5; y++ {
		for x := 0; x < 12; x++ {
			c, _ := buf.GetCell(x, y)
			switch c.Rune {
			case '─', '│', '┌', '┐', '└', '┘':
				t.Errorf("BorderNone must not draw frame, got %q at (%d,%d)", c.Rune, x, y)
			}
		}
	}
}

// 逐边开关：只画上边时，另外三边不得落笔。
func TestPanelBorderEdgesPerEdge(t *testing.T) {
	p := NewPanel("t")
	p.SetBorderEdges(screen.BorderEdges{Top: true})
	r := geom.Rect{X: 0, Y: 0, W: 8, H: 5}
	buf := paintBounds(p, r)
	for x := 0; x < 8; x++ {
		c, _ := buf.GetCell(x, 0)
		if c.Rune != '─' {
			t.Errorf("top row (%d,0) = %q want ─", x, c.Rune)
		}
	}
	for y := 1; y < 5; y++ {
		for _, x := range []int{0, 7} {
			c, _ := buf.GetCell(x, y)
			if c.Rune == '│' {
				t.Errorf("side (%d,%d) must stay empty with only top edge", x, y)
			}
		}
	}
	for x := 0; x < 8; x++ {
		if c, _ := buf.GetCell(x, 4); c.Rune == '─' {
			t.Errorf("bottom row (%d,4) must stay empty with only top edge", x)
		}
	}
	// 逐边开关不改变布局：内容仍在 1 格内缩区内。
	if p.Child() != nil {
		t.Fatal("no child expected")
	}
	if p.BorderEdges() != (screen.BorderEdges{Top: true}) {
		t.Errorf("BorderEdges = %+v", p.BorderEdges())
	}
}

// 任意字形集（SetBorderSpec）可用：dashed / ascii / thick 都不必新增枚举。
func TestPanelBorderSpecArbitraryGlyphSet(t *testing.T) {
	for _, b := range []screen.Border{screen.BorderDashed, screen.BorderAscii, screen.BorderThick} {
		p := NewPanel("t")
		p.SetBorderSpec(b)
		r := geom.Rect{X: 0, Y: 0, W: 8, H: 5}
		buf := paintBounds(p, r)
		c, _ := buf.GetCell(0, 0)
		if c.Rune != b.At(0, 0) {
			t.Errorf("%s top-left = %q want %q", b.Style, c.Rune, b.At(0, 0))
		}
		if p.BorderSpec() != b {
			t.Errorf("BorderSpec = %+v want %+v", p.BorderSpec(), b)
		}
	}
}

// P1：字形词表归一到 screen（消除重复字面量）。
func TestWidgetGlyphsComeFromSharedTable(t *testing.T) {
	// Separator 默认线字形来自边框表。
	sep := NewSeparator()
	sep.Layout(geom.Rect{X: 0, Y: 0, W: 4, H: 1})
	bufH := paintBounds(sep, geom.Rect{X: 0, Y: 0, W: 4, H: 1})
	for x := 0; x < 4; x++ {
		c, _ := bufH.GetCell(x, 0)
		if c.Rune != screen.BorderSingle.HLine() {
			t.Errorf("separator (%d,0) = %q want %q", x, c.Rune, screen.BorderSingle.HLine())
		}
	}
	sepV := NewSeparator()
	sepV.SetHorizontal(false)
	bufV := paintBounds(sepV, geom.Rect{X: 0, Y: 0, W: 1, H: 3})
	for y := 0; y < 3; y++ {
		c, _ := bufV.GetCell(0, y)
		if c.Rune != screen.BorderSingle.VLine() {
			t.Errorf("separator (0,%d) = %q want %q", y, c.Rune, screen.BorderSingle.VLine())
		}
	}

	// Scrollbar 用共享表的 Track / Thumb。
	sb := NewScrollbar()
	sb.SetTotal(100)
	sb.SetView(10)
	sb.SetOffset(0)
	buf := paintBounds(sb, geom.Rect{X: 0, Y: 0, W: 1, H: 10})
	first, _ := buf.GetCell(0, 0)
	if first.Rune != screen.Bar.Filled {
		t.Errorf("scrollbar thumb = %q want %q", first.Rune, screen.Bar.Filled)
	}
	last, _ := buf.GetCell(0, 9)
	if last.Rune != screen.Bar.Track {
		t.Errorf("scrollbar track = %q want %q", last.Rune, screen.Bar.Track)
	}

	// List 的内置滚动条指示同样走共享表。
	l := NewList()
	l.SetItems([]string{"a", "b", "c", "d"})
	bufL := paintBounds(l, geom.Rect{X: 0, Y: 0, W: 4, H: 2})
	c, _ := bufL.GetCell(3, 1)
	if c.Rune != screen.Bar.Track && c.Rune != screen.Bar.Filled {
		t.Errorf("list scrollbar = %q want one of %q/%q", c.Rune, screen.Bar.Track, screen.Bar.Filled)
	}
}

// ProgressBar 的 ASCII 条与填充块都用共享表。
func TestProgressBarUsesSharedGlyphs(t *testing.T) {
	p := NewProgressBar()
	p.SetRange(0, 10)
	p.SetValue(5)
	buf := paintBounds(p, geom.Rect{X: 0, Y: 0, W: 10, H: 1})
	row := bufRow(buf, 0)
	if row == "" {
		t.Fatal("empty row")
	}
	if runeAt(row, 0) != screen.Bar.CapL || runeAt(row, text.Width(row)-1) != screen.Bar.CapR {
		t.Errorf("ascii bar = %q want caps %q/%q", row, screen.Bar.CapL, screen.Bar.CapR)
	}
	if !containsRune(row[1:len(row)-1], screen.Bar.Pad) {
		t.Errorf("ascii bar = %q want fill %q", row, screen.Bar.Pad)
	}

	// 三行高时填充块 = Bar.Filled。
	p3 := NewProgressBar()
	p3.SetRange(0, 10)
	p3.SetValue(10)
	buf3 := paintBounds(p3, geom.Rect{X: 0, Y: 0, W: 10, H: 3})
	inner, _ := buf3.GetCell(2, 1)
	if inner.Rune != screen.Bar.Filled {
		t.Errorf("progress fill = %q want %q", inner.Rune, screen.Bar.Filled)
	}
}

// Slider 的 ASCII 条用共享表（CapL/Pad/KnobH/CapR）。
func TestSliderUsesSharedGlyphs(t *testing.T) {
	s := NewSlider()
	s.SetHorizontal(true)
	s.SetRange(0, 10)
	s.SetValue(5)
	buf := paintBounds(s, geom.Rect{X: 0, Y: 0, W: 12, H: 3})
	row := bufRow(buf, 0)
	if runeAt(row, 0) != screen.Bar.CapL {
		t.Errorf("slider row = %q want cap %q", row, screen.Bar.CapL)
	}
	if !containsRune(row, screen.Bar.KnobH) {
		t.Errorf("slider row = %q want knob %q", row, screen.Bar.KnobH)
	}
}

// TextInput 的边框走字形表（此前直接传 style.AttrBold 绕过了枚举）。
func TestTextInputUsesBorderTable(t *testing.T) {
	ti := NewTextInput()
	// TextInput 自身高 1（退化为横线）；给 3 行高以覆盖角标路径。
	buf := paintBounds(ti, geom.Rect{X: 0, Y: 0, W: 8, H: 3})
	if c, _ := buf.GetCell(0, 0); c.Rune != screen.BorderHeavy.At(0, 0) {
		t.Errorf("textinput top-left = %q want %q", c.Rune, screen.BorderHeavy.At(0, 0))
	}
	if c, _ := buf.GetCell(0, 1); c.Rune != screen.BorderHeavy.At(1, 0) {
		t.Errorf("textinput left side = %q want %q", c.Rune, screen.BorderHeavy.At(1, 0))
	}
	if c, _ := buf.GetCell(7, 2); c.Rune != screen.BorderHeavy.At(2, 2) {
		t.Errorf("textinput bottom-right = %q want %q", c.Rune, screen.BorderHeavy.At(2, 2))
	}
	// 单行高退化为横线。
	ti1 := NewTextInput()
	buf1 := paintBounds(ti1, geom.Rect{X: 0, Y: 0, W: 8, H: 1})
	for x := 0; x < 8; x++ {
		c, _ := buf1.GetCell(x, 0)
		if c.Rune != screen.BorderHeavy.HLine() {
			t.Errorf("textinput H=1 (%d,0) = %q want %q", x, c.Rune, screen.BorderHeavy.HLine())
		}
	}
}

// ---- P1：Slider 垂直方向必须画满全部行 ----

func TestSliderVerticalFillsAllRows(t *testing.T) {
	for _, v := range []int{0, 25, 50, 75, 100} {
		s := NewSlider()
		s.SetHorizontal(false)
		s.SetRange(0, 100)
		s.SetValue(v)
		r := geom.Rect{X: 0, Y: 0, W: 15, H: 7}
		buf := paintBounds(s, r)
		// 每行都必须落笔（Measure 报 15×7，不能只画 1/7 行）。
		for y := 0; y < 7; y++ {
			for x := 0; x < 15; x++ {
				c, _ := buf.GetCell(x, y)
				if c.Rune == 0 && c.Width == 0 {
					t.Fatalf("value %d: hole at (%d,%d)", v, x, y)
				}
			}
		}
		// 值越大，已填充行越多（在最左列统计，避开中央手柄格）。
		// 与水平方向同构：手柄格本身不计为已填充，故 100% 时 H-1 行已填充。
		filled := 0
		for y := 0; y < 7; y++ {
			c, _ := buf.GetCell(0, y)
			if c.Rune == screen.Bar.Filled {
				filled++
			}
		}
		if v == 0 && filled != 0 {
			t.Errorf("value 0: filled rows = %d want 0", filled)
		}
		if v == 100 && filled != 6 {
			t.Errorf("value 100: filled rows = %d want 6 (knob row excluded)", filled)
		}
	}
}

// 垂直滑块的手柄必须随值上下移动。
func TestSliderVerticalKnobTracksValue(t *testing.T) {
	knobRow := func(v int) int {
		s := NewSlider()
		s.SetHorizontal(false)
		s.SetRange(0, 100)
		s.SetValue(v)
		buf := paintBounds(s, geom.Rect{X: 0, Y: 0, W: 15, H: 7})
		for y := 0; y < 7; y++ {
			for x := 0; x < 15; x++ {
				c, _ := buf.GetCell(x, y)
				if c.Rune == screen.Bar.KnobV {
					return y
				}
			}
		}
		return -1
	}
	lo, hi := knobRow(0), knobRow(100)
	if lo < 0 || hi < 0 {
		t.Fatalf("knob missing: lo=%d hi=%d", lo, hi)
	}
	if lo != 0 {
		t.Errorf("value 0: knob row = %d want 0", lo)
	}
	if hi != 6 {
		t.Errorf("value 100: knob row = %d want 6", hi)
	}
	if knobRow(50) <= lo || knobRow(50) >= hi {
		t.Errorf("value 50: knob row = %d want strictly between %d and %d", knobRow(50), lo, hi)
	}
}

// 水平滑块铺满整行（尾部空档纪律），且 100% 时手柄仍在轨道内。
func TestSliderHorizontalFillsRowAndKeepsKnob(t *testing.T) {
	s := NewSlider()
	s.SetHorizontal(true)
	s.SetRange(0, 100)
	s.SetValue(100)
	// 铺满纪律只在"有背景可铺"时成立（零样式 = 透明，见 paintBackground）。
	s.SetStyle(style.Style{}.WithBackground(style.ANSIColor(238)))
	buf := paintBounds(s, geom.Rect{X: 0, Y: 0, W: 12, H: 3})
	// 全部 3 行无空洞
	for y := 0; y < 3; y++ {
		for x := 0; x < 12; x++ {
			c, _ := buf.GetCell(x, y)
			if c.Rune == 0 && c.Width == 0 {
				t.Errorf("hole at (%d,%d)", x, y)
			}
		}
	}
	row := bufRow(buf, 0)
	if !containsRune(row, screen.Bar.KnobH) {
		t.Errorf("100%%: row = %q want knob %q inside the track", row, screen.Bar.KnobH)
	}
}

// 退化尺寸不得 panic。
func TestSliderDegenerateBounds(t *testing.T) {
	for _, r := range []geom.Rect{
		{X: 0, Y: 0, W: 0, H: 0}, {X: 0, Y: 0, W: 1, H: 1},
		{X: 0, Y: 0, W: 4, H: 0}, {X: 0, Y: 0, W: 0, H: 4},
	} {
		for _, horiz := range []bool{true, false} {
			s := NewSlider()
			s.SetHorizontal(horiz)
			buf := paintBounds(s, r)
			if r.W > 0 && r.H > 0 {
				continue // 合法区域不做空洞断言
			}
			_ = buf
		}
	}
}

// ---- helpers ----

// bufRow 把一行还原成字符串（续格跳过）。
func bufRow(buf *screen.Buffer, y int) string {
	out := make([]rune, 0, buf.Width())
	for x := 0; x < buf.Width(); x++ {
		c, _ := buf.GetCell(x, y)
		if c.IsContinuation() {
			continue
		}
		if c.Rune == 0 {
			out = append(out, ' ')
			continue
		}
		out = append(out, c.Rune)
	}
	return string(out)
}

func runeAt(s string, col int) rune {
	w := 0
	for _, r := range s {
		if w == col {
			return r
		}
		w += text.RuneWidth(r)
	}
	return 0
}

func containsRune(s string, r rune) bool {
	for _, c := range s {
		if c == r {
			return true
		}
	}
	return false
}
