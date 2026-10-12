package screen

import (
	"PaintedSkin/geom"
	"PaintedSkin/style"
	"PaintedSkin/text"
)

// 本文件让 *Buffer 直接满足 Canvas 接口：控件 Paint 的绘制目标。
// 绘制坐标一律为绝对单元格坐标；Clip 用栈保存，越界写入静默丢弃。

// clipRect 是裁剪栈的一层。
type clipRect struct {
	r    geom.Rect
	full bool // 无裁剪（全缓冲）
}

// Clip 压入裁剪区（与已有裁剪取交集）。
func (b *Buffer) Clip(r geom.Rect) {
	full := geom.Rect{X: 0, Y: 0, W: b.w, H: b.h}
	n := len(b.clips) - 1
	if n < 0 {
		b.clips = append(b.clips, clipRect{r: r.Intersect(full)})
		return
	}
	if b.clips[n].full {
		b.clips = append(b.clips, clipRect{r: r.Intersect(full)})
		return
	}
	b.clips = append(b.clips, clipRect{r: r.Intersect(b.clips[n].r)})
}

// PopClip 弹出最后一层裁剪（栈空时无操作）。
func (b *Buffer) PopClip() {
	if len(b.clips) == 0 {
		return
	}
	b.clips = b.clips[:len(b.clips)-1]
}

// ResetClip 清空裁剪栈。
func (b *Buffer) ResetClip() { b.clips = nil }

// clip 返回当前生效的裁剪区。
func (b *Buffer) clip() geom.Rect {
	n := len(b.clips) - 1
	if n < 0 || b.clips[n].full {
		return geom.Rect{X: 0, Y: 0, W: b.w, H: b.h}
	}
	return b.clips[n].r
}

// inClip 报告 (x,y) 是否在当前裁剪区内。
func (b *Buffer) inClip(x, y int) bool {
	if x < 0 || y < 0 || x >= b.w || y >= b.h {
		return false
	}
	return b.clip().Contains(geom.Point{X: x, Y: y})
}

// Size 返回缓冲尺寸。
func (b *Buffer) Size() geom.Size { return geom.Size{W: b.w, H: b.h} }

// SetCell 写入单个单元格（越界或裁剪外丢弃）。
//
// 宽字符成对不变量：**续格（Width=-1）不能独立存在**，其左邻必须是一个
// `Width==2` 的宽字符左格。任何一次写入都要维护这条不变量，否则终端上会残留
// 半个字，且 diff 阶段会把续格当"半格残留"多补一个空格、整行右移一列。
func (b *Buffer) SetCell(x, y int, r rune, s style.Style) {
	if !b.inClip(x, y) {
		return
	}
	_, w := Normalize(r)
	row := y * b.w

	// 零宽字符（组合附标 / ZWJ）并入**前一格**的 Tail：终端只有收到
	// "基字符 + 零宽序列"整串才会合成一个字素。没有可依附的基字符
	// （行首、上一格是续格的左半之外）时丢弃。
	if w == 0 {
		b.appendTail(x, row, r)
		return
	}

	// 宽字符需要两格；右边放不下时退化成空格而不是丢弃内容——
	// 丢弃会让调用方（如按列排布的布局）出现空洞，而写入宽字符会让
	// 终端把光标推到下一行、撑破边框（docs/05）。
	if w == 2 && (x+1 >= b.w || !b.inClip(x+1, y)) {
		b.clearPair(x, row, s)
		b.cells[row+x] = Cell{Rune: ' ', Width: 1, Style: s}
		return
	}

	b.clearPair(x, row, s)
	cell := Cell{Rune: r, Width: w, Style: s}
	if w == 2 && b.inClip(x+1, y) {
		cell.Tail = ""
		b.cells[row+x+1] = Cell{Rune: 0, Width: -1, Style: s}
	}
	// 零样式 = **透明叠加**：只写字符，保留格子上已有的样式。
	// 这样未设样式的控件叠在有底色的容器上时，文字格不会把容器底色戳穿
	// （容器背景由 `Column`/`Row`/`Root` 的 SetStyle 提供）。
	if s == (style.Style{}) {
		cell.Style = b.cells[row+x].Style
	}
	b.cells[row+x] = cell
}

// appendTail 把零宽字符 r 追加到 x 左侧那一格的 Tail。
//
// 定位规则：目标格是 x-1；若该格是宽字符的续格（说明 x-1 本身是宽字符的
// 右半），再往左退到那个宽字符的左格 —— 附标必须紧跟基字符，不能挂到宽字符
// **前面**那一格去。没有可依附的非空格时丢弃。
func (b *Buffer) appendTail(x, row int, r rune) {
	i := x - 1
	for i >= 0 && b.cells[row+i].IsContinuation() {
		i--
	}
	if i < 0 {
		return // 行首：没有基字符
	}
	c := &b.cells[row+i]
	if c.Rune == 0 && c.Tail == "" {
		return // 目标格是空的 → 丢弃孤立的零宽字符
	}
	c.Tail += string(r)
}

// clearPair 清理"因覆盖 x 而被拆散"的宽字符对。
//
// 宽字符占两格（`Width==2` 的左格 + `Width==-1` 的续格），拆散任何一半
// 都会让 diff 写错列数：宽字符被写出时会吃掉两列，而缓冲只打算写一格。
// 所以覆盖任一半时都要把配对的另一半降级成普通空格：
//
//   - `x` 原本是左格 → 其右半续格清成空格；
//   - `x` 原本是续格 → 其左半降级成空格（否则它会多写一列）。
func (b *Buffer) clearPair(x, row int, s style.Style) {
	sp := Cell{Rune: ' ', Width: 1, Style: s}
	switch {
	case b.cells[row+x].Width == 2 && x+1 < b.w:
		b.cells[row+x+1] = sp
	case x+1 < b.w && b.cells[row+x+1].IsContinuation():
		b.cells[row+x+1] = sp
	}
	if x > 0 && b.cells[row+x-1].Width == 2 {
		b.cells[row+x-1] = sp
	}
}

// GetCell 返回单元格副本。
func (b *Buffer) GetCell(x, y int) (Cell, bool) {
	if x < 0 || y < 0 || x >= b.w || y >= b.h {
		return Cell{}, false
	}
	return b.cells[y*b.w+x], true
}

// Fill 用字符填充矩形（逐格裁剪）。
func (b *Buffer) Fill(rect geom.Rect, r rune, s style.Style) { b.fillCells(rect, r, s) }

// FillStyle 用空格铺满矩形：控件"先铺自身背景、再写内容"的原语，
// 保证区域内每格要么是内容、要么是控件自己的背景色（docs/05 尾部空档纪律）。
func (b *Buffer) FillStyle(rect geom.Rect, s style.Style) { b.fillCells(rect, ' ', s) }

// fillCells 用同一个格内容铺满矩形。
//
// 成对处理宽字符：续格（Width=-1）不能独立存在，**宽字符左格也不能失去配对**
// —— 两者拆散任一半都会让 diff 写错列数。因此铺底时双向清理：
// 覆盖左格清右半，覆盖续格把左半降级成空格。
func (b *Buffer) fillCells(rect geom.Rect, r rune, s style.Style) {
	area := rect.Intersect(b.clip())
	if area.Empty() {
		return
	}
	cell := Cell{Rune: r, Width: 1, Style: s} // Tail 空：覆盖整格连带清掉附标
	for y := area.Y; y < area.Bottom(); y++ {
		row := y * b.w
		if area.X > 0 && b.cells[row+area.X-1].Width == 2 {
			// 区域左边界切开了别人的宽字符对：把左半降级，避免它多写一列。
			b.cells[row+area.X-1] = cell
		}
		for x := area.X; x < area.Right(); x++ {
			b.cells[row+x] = cell
			if x+1 < area.Right() && b.cells[row+x+1].IsContinuation() {
				b.cells[row+x+1] = cell
			}
		}
	}
}

// Text 写入文本，按显示宽步进（宽字符占两格，右格写占位）。
//
// 零宽字符（组合附标 / ZWJ）**不推进列**，交给 SetCell 并入前一格的 Tail
// —— 逐格丢弃会让终端收不到完整的字素序列（ADR-022/023 的遗留修复）。
func (b *Buffer) Text(x, y int, s string, st style.Style) {
	cx := x
	for _, r := range s {
		rw := text.RuneWidth(r)
		if rw == 0 {
			if cx > b.w {
				return
			}
			b.SetCell(cx, y, r, st)
			continue
		}
		if cx >= b.w {
			return
		}
		// SetCell 内部已处理宽字符的右格占位。
		b.SetCell(cx, y, r, st)
		cx += rw
	}
}

// Box 绘制矩形边框。
//
// 字形来自 border 的 3×3 表，逐边开关由 border.Edges 决定：
// 关掉的边不落笔（对应"占位但不画"，布局仍保留该边的间距）。
// 退化尺寸（1×1 / H=1 / W=1）下只画仍开启的边，绝不越界。
func (b *Buffer) Box(rect geom.Rect, border Border, s style.Style) {
	area := rect.Intersect(b.clip())
	if area.Empty() || !border.Visible() {
		return
	}
	e := border.Edges
	x0, y0 := area.X, area.Y
	x1, y1 := area.Right()-1, area.Bottom()-1

	// 退化尺寸：H=1 只画横线（否则上下边框互相覆盖），
	// W=1 只画竖线，1x1 画一个横线点。
	if area.H == 1 {
		if e.Top {
			b.hline(x0, x1, y0, 0, border, s)
		}
		if e.Bottom {
			b.hline(x0, x1, y0, 2, border, s)
		}
		return
	}
	if area.W == 1 {
		if e.Left || e.Right {
			b.vline(y0, y1, x0, border, s)
		}
		return
	}

	// 顶边 / 底边（含角）
	if e.Top {
		b.hline(x0, x1, y0, 0, border, s)
		if e.Left {
			b.SetCell(x0, y0, border.At(0, 0), s)
		}
		if e.Right {
			b.SetCell(x1, y0, border.At(0, 2), s)
		}
	}
	if e.Bottom {
		b.hline(x0, x1, y1, 2, border, s)
		if e.Left {
			b.SetCell(x0, y1, border.At(2, 0), s)
		}
		if e.Right {
			b.SetCell(x1, y1, border.At(2, 2), s)
		}
	}
	// 侧边（不含角，避免与顶/底边的角标互相覆盖）
	for y := y0 + 1; y < y1; y++ {
		if e.Left {
			b.SetCell(x0, y, border.At(1, 0), s)
		}
		if e.Right {
			b.SetCell(x1, y, border.At(1, 2), s)
		}
	}
}

// Outline 在区域外围套一圈"只换样式"的环：字符与宽度都不动，
// 因此**既不吃内容、也不占布局**（Textual §1.5 的 outline 语义）。
//
// 与 Textual 的差别是刻意的：Textual 的实现是"裁掉首尾格 + 把 outline 拼回
// 新端点"，会吃掉边框那一圈的内容 —— 它能这么做是因为 outline 画在 border box
// 外侧的独立 gutter，而本项目的布局没有预留 gutter，且叶控件（`[ OK ]`、
// 单行输入、列表项）的 `Bounds` 就是全部内容。
//
// 逐边开关：只勾选部分边时只套那些边（可做"只高亮上边"这类变体）。
func (b *Buffer) Outline(rect geom.Rect, edges BorderEdges, s style.Style) {
	area := rect.Intersect(b.clip())
	if area.Empty() || edges == (BorderEdges{}) {
		return
	}
	x0, y0 := area.X, area.Y
	x1, y1 := area.Right()-1, area.Bottom()-1

	// 退化尺寸：H=1 视作上下边重叠（任一开启即套整行），W=1 同理。
	if area.H == 1 {
		if edges.Top || edges.Bottom {
			for x := x0; x <= x1; x++ {
				b.restyle(x, y0, s)
			}
		}
		return
	}
	if area.W == 1 {
		if edges.Left || edges.Right {
			for y := y0; y <= y1; y++ {
				b.restyle(x0, y, s)
			}
		}
		return
	}
	if edges.Top {
		b.outlineRow(x0, x1, y0, s)
	}
	if edges.Bottom {
		b.outlineRow(x0, x1, y1, s)
	}
	// 左右两列含角（角属于相邻的两条边）。
	for y := y0; y <= y1; y++ {
		if edges.Left {
			b.restyle(x0, y, s)
		}
		if edges.Right {
			b.restyle(x1, y, s)
		}
	}
}

// outlineRow 沿一行套上 outline 样式（字符不变）。
func (b *Buffer) outlineRow(x0, x1, y int, s style.Style) {
	for x := x0; x <= x1; x++ {
		b.restyle(x, y, s)
	}
}

// restyle 只替换单元格样式，不改字符与宽度（outline 的核心）。
// 续格也一并换色，这样整条环的观感一致。
func (b *Buffer) restyle(x, y int, s style.Style) {
	if !b.inClip(x, y) {
		return
	}
	b.cells[y*b.w+x].Style = s
}

// hline 画一整行横线（不含角）。row 取 0/2 —— 多数边框两行的横线字形
// 相同，但半块粗框用 ▀/▄ 区分上下，必须分别取。
func (b *Buffer) hline(x0, x1, y, row int, border Border, s style.Style) {
	g := border.At(row, 1)
	for x := x0; x <= x1; x++ {
		b.SetCell(x, y, g, s)
	}
}

// vline 画一整列竖线。
func (b *Buffer) vline(y0, y1, x int, border Border, s style.Style) {
	g := border.At(1, 0)
	for y := y0; y <= y1; y++ {
		b.SetCell(x, y, g, s)
	}
}

// ensure Buffer 满足 Canvas。
var _ Canvas = (*Buffer)(nil)

// Hatch 在区域内的空格上按固定密度点缀底纹（Textual 的 `hatch`）。
//
// 三条纪律：
//  1. **只落在空格上** —— 非空格（含续格）一律跳过，绝不覆盖真实内容；
//  2. **不改字符宽度** —— 底纹字形是单列宽，不影响行宽口径（docs/05）；
//  3. 密度固定（每 5 格 1 个 ≈ 20%），相位由 (x+y) 决定，斜纹因此连续。
func (b *Buffer) Hatch(rect geom.Rect, hatch Hatch, s style.Style) {
	glyphs, ok := hatchGlyphs[hatch]
	if !ok {
		return
	}
	area := rect.Intersect(b.clip())
	if area.Empty() {
		return
	}
	for y := area.Y; y < area.Bottom(); y++ {
		row := y * b.w
		for x := area.X; x < area.Right(); x++ {
			c := b.cells[row+x]
			// 纪律 1：只装饰空格（续格是宽字符的右半，也跳过）。
			if c.IsContinuation() || (c.Rune != 0 && c.Rune != ' ') {
				continue
			}
			phase := (x + y) % hatchDensity
			if phase < 0 {
				phase += hatchDensity
			}
			// 只有相位 0 落主字形、相位 1 落副字形；其余相位留空 ——
			// 这就是 20% 密度。
			var g rune
			switch phase {
			case 0:
				g = glyphs[0]
			case 1:
				g = glyphs[1]
			}
			if g == 0 {
				continue
			}
			b.cells[row+x] = Cell{Rune: g, Width: 1, Style: s}
		}
	}
}
