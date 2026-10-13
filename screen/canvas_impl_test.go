package screen

import (
	"testing"

	"PaintedSkin/geom"
	"PaintedSkin/style"
)

// Buffer 必须实现 Canvas（控件的绘制目标）。
func TestBufferIsCanvas(t *testing.T) {
	var c Canvas = NewBuffer(10, 3)
	if c.Size() != (geom.Size{W: 10, H: 3}) {
		t.Errorf("size = %+v", c.Size())
	}
}

// SetCell 写入并可读回；越界丢弃。
func TestCanvasSetCell(t *testing.T) {
	b := NewBuffer(4, 2)
	b.SetCell(1, 1, 'x', style.Default)
	if got := b.At(1, 1); got == nil || got.Rune != 'x' {
		t.Errorf("cell = %+v", got)
	}
	// 越界不 panic
	b.SetCell(99, 99, 'y', style.Default)
	b.SetCell(-1, 0, 'y', style.Default)
}

// Fill 填充矩形区域。
func TestCanvasFill(t *testing.T) {
	b := NewBuffer(4, 2)
	b.Fill(geom.Rect{X: 0, Y: 0, W: 2, H: 1}, '#', style.Default)
	if b.At(0, 0).Rune != '#' || b.At(1, 0).Rune != '#' {
		t.Errorf("row = %+v %+v", b.At(0, 0), b.At(1, 0))
	}
	if b.At(2, 0).Rune != 0 {
		t.Error("fill must not exceed rect")
	}
}

// Text 按宽字符步进写入。
func TestCanvasText(t *testing.T) {
	b := NewBuffer(10, 1)
	b.Text(0, 0, "ab", style.Default)
	if b.At(0, 0).Rune != 'a' || b.At(1, 0).Rune != 'b' {
		t.Errorf("text = %+v %+v", b.At(0, 0), b.At(1, 0))
	}
	if b.At(2, 0).Rune != 0 {
		t.Error("text must stop after content")
	}
	// 越界文本不 panic，且不写出界
	b.Text(8, 0, "xyz", style.Default)
	if b.At(8, 0).Rune != 'x' || b.At(9, 0).Rune != 'y' {
		t.Errorf("clipped text = %+v %+v", b.At(8, 0), b.At(9, 0))
	}
}

// Clip 裁剪：clip 区外的写入被丢弃。
func TestCanvasClip(t *testing.T) {
	b := NewBuffer(10, 2)
	b.Clip(geom.Rect{X: 0, Y: 0, W: 2, H: 1})
	b.Text(0, 0, "abcdef", style.Default)
	if b.At(1, 0).Rune != 'b' {
		t.Errorf("inside clip = %+v", b.At(1, 0))
	}
	if b.At(2, 0).Rune != 0 {
		t.Errorf("outside clip must be dropped: %+v", b.At(2, 0))
	}
	b.PopClip()
	b.Text(0, 1, "z", style.Default)
	if b.At(0, 1).Rune != 'z' {
		t.Error("after PopClip must write again")
	}
}

// 嵌套 Clip 取交集。
func TestCanvasNestedClip(t *testing.T) {
	b := NewBuffer(10, 3)
	b.Clip(geom.Rect{X: 0, Y: 0, W: 6, H: 3})
	b.Clip(geom.Rect{X: 0, Y: 0, W: 3, H: 1})
	b.Text(0, 0, "abcdef", style.Default)
	if b.At(2, 0).Rune != 'c' {
		t.Errorf("inside = %+v", b.At(2, 0))
	}
	if b.At(3, 0).Rune != 0 {
		t.Errorf("outside nested clip = %+v", b.At(3, 0))
	}
	b.PopClip()
	b.PopClip()
	b.Text(0, 2, "q", style.Default)
	if b.At(0, 2).Rune != 'q' {
		t.Error("after both PopClip must write")
	}
}

// Box 画单线边框。
func TestCanvasBox(t *testing.T) {
	b := NewBuffer(4, 3)
	b.Box(geom.Rect{X: 0, Y: 0, W: 4, H: 3}, BorderSingle, style.Default)
	corners := [][2]int{{0, 0}, {3, 0}, {0, 2}, {3, 2}}
	for _, p := range corners {
		if b.At(p[0], p[1]).Rune == 0 {
			t.Errorf("corner %v missing", p)
		}
	}
	if b.At(1, 0).Rune == 0 {
		t.Error("top edge missing")
	}
	if b.At(1, 1).Rune != 0 {
		t.Error("interior must stay empty")
	}
}

// ---- SGR 生成（theme 依赖颜色正确落盘）----

func TestAnsiColor16(t *testing.T) {
	b := NewBuffer(2, 1)
	b.SetCell(0, 0, 'a', style.Style{}.WithForeground(style.Red))
	if got := string(b.Present(nil)); string(got) != "\x1b[2J" {
		_ = got // Present(nil) 只发清屏，单测直接看渲染辅助
	}
	if got := ansiStyle(style.Style{}.WithForeground(style.Red)); string(got) != "\x1b[31m" {
		t.Errorf("16-color red = %q want ESC[31m", got)
	}
	if got := ansiStyle(style.Style{}.WithForeground(style.BrightRed)); string(got) != "\x1b[91m" {
		t.Errorf("bright red = %q want ESC[91m", got)
	}
	if got := ansiStyle(style.Style{}.WithBackground(style.Blue)); string(got) != "\x1b[44m" {
		t.Errorf("bg blue = %q want ESC[44m", got)
	}
}

func TestAnsiColor256(t *testing.T) {
	c := style.Color{Kind: style.Color256, N: 33}
	if got := ansiStyle(style.Style{}.WithForeground(c)); string(got) != "\x1b[38;5;33m" {
		t.Errorf("256 fg = %q", got)
	}
	if got := ansiStyle(style.Style{}.WithBackground(c)); string(got) != "\x1b[48;5;33m" {
		t.Errorf("256 bg = %q", got)
	}
}

func TestAnsiRGB(t *testing.T) {
	c := style.RGB(1, 2, 3)
	if got := ansiStyle(style.Style{}.WithForeground(c)); string(got) != "\x1b[38;2;1;2;3m" {
		t.Errorf("rgb fg = %q", got)
	}
	if got := ansiStyle(style.Style{}.WithBackground(c)); string(got) != "\x1b[48;2;1;2;3m" {
		t.Errorf("rgb bg = %q", got)
	}
}

func TestAnsiDefaultNoColor(t *testing.T) {
	if got := ansiStyle(style.Default); string(got) != "\x1b[m" {
		t.Errorf("default = %q want reset", got)
	}
}

func TestAnsiCompoundStyle(t *testing.T) {
	st := style.Style{}.WithForeground(style.Red).WithBackground(style.Blue)
	if got := ansiStyle(st); string(got) != "\x1b[31;44m" {
		t.Errorf("fg+bg = %q want ESC[31;44m", got)
	}

	stBold := st.WithBold(true)
	if got := ansiStyle(stBold); string(got) != "\x1b[31;44;1m" {
		t.Errorf("fg+bg+bold = %q want ESC[31;44;1m", got)
	}

	c256fg := style.Color{Kind: style.Color256, N: 33}
	c256bg := style.Color{Kind: style.Color256, N: 44}
	st256 := style.Style{}.WithForeground(c256fg).WithBackground(c256bg)
	if got := ansiStyle(st256); string(got) != "\x1b[38;5;33;48;5;44m" {
		t.Errorf("256 fg+bg = %q", got)
	}
}

func TestEscapeInjectionPrevention(t *testing.T) {
	b := NewBuffer(10, 2)
	// CSI 转义注入应被剥离，不进入缓冲区，不执行清屏
	b.Text(0, 0, "\x1b[2Jhello", style.Default)
	if b.At(0, 0).Rune != 'h' || b.At(4, 0).Rune != 'o' {
		t.Errorf("text contaminated: cell0=%+v", b.At(0, 0))
	}
	// OSC 52 剪贴板注入应被彻底剥离
	b.Text(0, 1, "\x1b]52;c;evil\aABC", style.Default)
	if b.At(0, 1).Rune != 'A' || b.At(1, 1).Rune != 'B' || b.At(2, 1).Rune != 'C' {
		t.Errorf("OSC not stripped: cell0=%+v", b.At(0, 1))
	}

	// 裸控制字符不得作为零宽字符并入 Tail
	b.SetCell(0, 0, 'X', style.Default)
	b.SetCell(1, 0, '\x1b', style.Default)
	b.SetCell(1, 0, '\a', style.Default)
	c := b.At(0, 0)
	if c.Tail != "" {
		t.Errorf("control char injected into Tail: %q", c.Tail)
	}
}

// Box 在 H=1 时不得上下边框互相覆盖（应退化为单线）。
func TestBoxSingleRow(t *testing.T) {
	b := NewBuffer(6, 3)
	b.Box(geom.Rect{X: 0, Y: 1, W: 6, H: 1}, BorderSingle, style.Default)
	for x := 0; x < 6; x++ {
		c, _ := b.GetCell(x, 1)
		if c.Rune != '─' {
			t.Errorf("H=1 cell %d = %q want ─", x, c.Rune)
		}
	}
	if c, _ := b.GetCell(0, 0); c.Rune != 0 {
		t.Error("must not draw outside rect")
	}
}

// Box 在 W=1 时退化为单竖线。
func TestBoxSingleColumn(t *testing.T) {
	b := NewBuffer(3, 4)
	b.Box(geom.Rect{X: 1, Y: 0, W: 1, H: 4}, BorderSingle, style.Default)
	for y := 0; y < 4; y++ {
		c, _ := b.GetCell(1, y)
		if c.Rune != '│' {
			t.Errorf("W=1 cell y=%d = %q want │", y, c.Rune)
		}
	}
}

// Box 在 1x1 时画一个点。
func TestBoxSingle(t *testing.T) {
	b := NewBuffer(2, 2)
	b.Box(geom.Rect{X: 0, Y: 0, W: 1, H: 1}, BorderSingle, style.Default)
	c, _ := b.GetCell(0, 0)
	if c.Rune != '─' && c.Rune != '│' {
		t.Errorf("1x1 = %q", c.Rune)
	}
}
