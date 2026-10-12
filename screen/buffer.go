package screen

import (
	"PaintedSkin/style"
	"PaintedSkin/text"
)

// Cell 是终端单元格。
//
// 宽字符（text.RuneWidth == 2）占两格：左格存字且 Width=2，右格写占位
// （Rune=0、Width=-1）。diff 时成对处理；当新内容只覆盖了旧宽字符的
// 一半时，必须在残留位置补空格清掉半字（docs/05）。
//
// 零宽字符（组合附标 U+0301、ZWJ 连接符 U+200D…）**并入前一格的 Tail**：
// 终端只有收到"基字符 + 紧随的零宽序列"这一整串才会把它们合成一个字素，
// 逐格写出必然拆散（`é` 变成 `e` 加一个孤零零的尖音符）。
// Tail 不占列（Width 不变），因此行宽口径与 text.Width 保持一致。
type Cell struct {
	Rune  rune
	Width int    // 1=普通；2=宽字符左格；-1=宽字符右格占位；0=空
	Tail  string // 紧随 Rune 的零宽字符序列（组合附标 / ZWJ），按出现顺序
	Style style.Style
}

// Grapheme 返回该格要写给终端的完整字符序列（基字符 + 零宽序列）。
// 空格的 Rune 为 0。
func (c Cell) Grapheme() string {
	if c.Rune == 0 {
		return c.Tail
	}
	return string(c.Rune) + c.Tail
}

// 空单元格。
func EmptyCell() Cell { return Cell{} }

// IsContinuation 报告是否为宽字符右格占位。
func (c Cell) IsContinuation() bool { return c.Width == -1 }

// Normalize 返回归一后的字符与宽度（供写入使用）。
func Normalize(r rune) (rune, int) {
	w := text.RuneWidth(r)
	if w < 0 {
		return r, 1
	}
	return r, w
}

// Equal 比较两个 Cell 是否相等（用于 diff）。Tail 参与比较 —— 换了附标
// 必须重画，否则终端上还是旧的字素。
func (c Cell) Equal(o Cell) bool {
	return c.Rune == o.Rune && c.Width == o.Width &&
		c.Tail == o.Tail && c.Style.Equal(o.Style)
}

// Buffer 是双缓冲区结构，包含前缓冲（已上屏）和后缓冲（本帧绘制目标）。
type Buffer struct {
	w, h  int
	cells []Cell
	clips []clipRect
}

// NewBuffer 创建一个新的 Buffer，尺寸为 w×h。
func NewBuffer(w, h int) *Buffer {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return &Buffer{
		w:     w,
		h:     h,
		cells: make([]Cell, w*h),
	}
}

// Reset 重置缓冲区，清空所有单元格为默认状态。
func (b *Buffer) Reset() {
	empty := Cell{}
	for i := range b.cells {
		b.cells[i] = empty
	}
	b.ResetClip()
}

// At 返回 (x, y) 位置的单元格引用。
func (b *Buffer) At(x, y int) *Cell {
	if x < 0 || x >= b.w || y < 0 || y >= b.h {
		return nil
	}
	return &b.cells[y*b.w+x]
}

// Set 设置 (x, y) 位置的单元格。
func (b *Buffer) Set(x, y int, c Cell) {
	cell := b.At(x, y)
	if cell != nil {
		*cell = c
	}
}

// Width 返回缓冲区的宽度（列数）。
func (b *Buffer) Width() int { return b.w }

// Height 返回缓冲区的高度（行数）。
func (b *Buffer) Height() int { return b.h }

// Cells 返回所有单元格的只读快照。
func (b *Buffer) Cells() []Cell { return b.cells }

// Present 将后缓冲与前缓冲 diff，返回需要写入终端的 ANSI 序列字节。
// old 为 nil 或尺寸不同 → 全量重绘。
func (b *Buffer) Present(old *Buffer) []byte {
	if old == nil || old.w != b.w || old.h != b.h {
		return b.fullRedraw()
	}
	var out []byte
	for y := 0; y < b.h; y++ {
		out = append(out, b.lineDiff(old, y)...)
	}
	return out
}

// fullRedraw 全量重绘：清屏 + 光标归位 + 逐格输出。
func (b *Buffer) fullRedraw() []byte {
	var out []byte
	out = append(out, "\x1b[2J"...) // 清整屏
	out = append(out, "\x1b[H"...)  // 光标归左上
	// 逐行输出；空行跳过（清屏已覆盖）。
	for y := 0; y < b.h; y++ {
		out = append(out, b.lineCells(y)...)
	}
	return out
}

// lineCells 输出整行所有非空内容（用于全量重绘）：连续段只定位一次光标。
func (b *Buffer) lineCells(y int) []byte {
	row := y * b.w
	var out []byte
	var cur style.Style
	styleSet := false
	lastX := -1
	for x := 0; x < b.w; x++ {
		c := b.cells[row+x]
		if c.IsContinuation() {
			continue // 由左格一并输出
		}
		if c.Rune == 0 && c.Width == 0 {
			continue // 空格子：清屏已覆盖
		}
		// 与上一段不连续才重新定位。
		if lastX < 0 || x != lastX+1 {
			out = append(out, ansiPos(y, x)...)
		}
		if !styleSet || !c.Style.Equal(cur) {
			out = append(out, ansiStyle(c.Style)...)
			cur, styleSet = c.Style, true
		}
		out = appendGrapheme(out, c)
		lastX = x + maxInt(c.Width, 1) - 1
	}
	return out
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// lineDiff 计算第 y 行相对于 old 的差异，返回 ANSI 序列字节。
//
// 宽字符处理的三条纪律（uv 的 Line.Render 同思路）：
//
//  1. **新帧的续格永不由本格写出** —— 写左格的宽字符时终端已吃掉右格；
//     再补一个空格会让整行右移一列。
//  2. **宽字符按整对比较**：左格样式没变但右格变了，同样要重画这一对
//     （续格的样式无法单独表达）。
//  3. **旧帧宽字符被只覆盖一半时补空格**：新内容在旧宽字符左格的位置上只有
//     1 列宽时，右半仍是旧字的残片，必须显式清掉。
func (b *Buffer) lineDiff(old *Buffer, y int) []byte {
	row := b.w * y
	orow := old.w * y
	var out []byte
	var cur style.Style
	styleSet := false

	x := 0
	for x < b.w {
		// 新帧续格：永不改写（纪律 1）。
		if b.cells[row+x].IsContinuation() {
			x++
			continue
		}
		if b.unchanged(old, row, orow, x) {
			x++
			continue
		}
		out = append(out, ansiPos(y, x)...)

		// 写连续变化段：连续变化只定位一次光标。
		for x < b.w {
			cc := b.cells[row+x]
			if cc.IsContinuation() {
				x++
				continue
			}
			oo := old.cells[orow+x]
			// 旧帧此处是宽字符左格、而新内容不是 2 列宽 → 右半会残留半字（纪律 3）。
			staleWide := oo.Width == 2 && cc.Width != 2
			if !staleWide && b.unchanged(old, row, orow, x) {
				break
			}
			if !styleSet || !cc.Style.Equal(cur) {
				out = append(out, ansiStyle(cc.Style)...)
				cur, styleSet = cc.Style, true
			}
			if cc.Rune == 0 && cc.Width == 0 {
				out = append(out, ' ') // 清空：写空格
			} else {
				out = appendGrapheme(out, cc)
			}
			x += maxInt(cc.Width, 1)
			if staleWide && x < b.w && b.cells[row+x].Rune == 0 && b.cells[row+x].Width == 0 {
				out = append(out, ' ')
				x++
			}
		}
	}
	return out
}

// unchanged 报告新帧第 x 格（含其宽字符右半）相对旧帧无需输出。
func (b *Buffer) unchanged(old *Buffer, row, orow, x int) bool {
	cc := b.cells[row+x]
	if !cc.Equal(old.cells[orow+x]) {
		return false
	}
	// 宽字符按整对比较（纪律 2）：右半样式变了也要重画这一对。
	if cc.Width == 2 && x+1 < b.w {
		return b.cells[row+x+1].Equal(old.cells[orow+x+1])
	}
	return true
}

// appendGrapheme 输出一个单元格的完整字素（基字符 + 零宽序列）。
// 宽字符占两格，终端自动吃掉后一格；空格写成 ' '。
func appendGrapheme(dst []byte, c Cell) []byte {
	if c.Rune == 0 {
		if c.Tail == "" {
			return append(dst, ' ')
		}
		return append(dst, c.Tail...)
	}
	return append(dst, c.Grapheme()...)
}

// ansiPos 生成光标定位序列 CSI row;colH（参数均 1-based）。
func ansiPos(row, col int) []byte {
	buf := make([]byte, 0, 12)
	buf = append(buf, "\x1b["...)
	buf = append(buf, itoaA(row+1)...)
	buf = append(buf, ';')
	buf = append(buf, itoaA(col+1)...)
	return append(buf, 'H')
}

// ansiStyle 生成 SGR 序列（末位 ';' 收尾为 'm'）。
func ansiStyle(s style.Style) []byte {
	var seq []byte
	seq = append(seq, "\x1b["...)
	seq = appendColor(seq, s.FG, true)
	seq = append(seq, appendColor(nil, s.BG, false)...)
	attrs := s.AttrMask()
	if attrs&style.AttrBold != 0 {
		seq = append(seq, '1', ';')
	}
	if attrs&style.AttrDim != 0 {
		seq = append(seq, '2', ';')
	}
	if attrs&style.AttrItalic != 0 {
		seq = append(seq, '3', ';')
	}
	if attrs&style.AttrUnderline != 0 {
		seq = append(seq, '4', ';')
	}
	if attrs&style.AttrBlink != 0 {
		seq = append(seq, '5', ';')
	}
	if attrs&style.AttrReverse != 0 {
		seq = append(seq, '7', ';')
	}
	if attrs&style.AttrStrike != 0 {
		seq = append(seq, '9', ';')
	}
	if len(seq) > 0 && seq[len(seq)-1] == ';' {
		seq[len(seq)-1] = 'm'
	} else {
		seq = append(seq, 'm')
	}
	return seq
}

// appendColor 按颜色深度发射 SGR 参数（docs/05 色彩降级）：
//
//	ColorANSI(0..7)   → 30–37 / 40–47
//	ColorANSI(8..15)  → 90–97 / 100–107
//	Color256          → 38;5;n / 48;5;n
//	ColorRGB          → 38;2;r;g;b / 48;2;r;g;b
//
// ColorDefault 不产出任何参数（终端默认色）。
func appendColor(dst []byte, c style.Color, fg bool) []byte {
	switch c.Kind {
	case style.ColorANSI:
		n := c.N
		if n < 8 {
			base := 30
			if !fg {
				base = 40
			}
			return append(dst, itoaA(base+int(n))...)
		}
		base := 90
		if !fg {
			base = 100
		}
		return append(dst, itoaA(base+int(n)-8)...)
	case style.Color256:
		if fg {
			dst = append(dst, "38;5;"...)
		} else {
			dst = append(dst, "48;5;"...)
		}
		return append(dst, itoaA(int(c.N))...)
	case style.ColorRGB:
		if fg {
			dst = append(dst, "38;2;"...)
		} else {
			dst = append(dst, "48;2;"...)
		}
		dst = append(dst, itoaA(int(c.R))...)
		dst = append(dst, ';')
		dst = append(dst, itoaA(int(c.G))...)
		dst = append(dst, ';')
		return append(dst, itoaA(int(c.B))...)
	}
	return dst
}

// itoaA 将整数转换为 ASCII 字节序列。
func itoaA(n int) []byte {
	if n == 0 {
		return []byte("0")
	}
	var buf []byte
	for n > 0 {
		buf = append(buf, byte('0'+n%10))
		n /= 10
	}
	for i, j := 0, len(buf)-1; i < j; i, j = i+1, j-1 {
		buf[i], buf[j] = buf[j], buf[i]
	}
	return buf
}
