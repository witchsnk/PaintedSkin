package widget

import (
	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/style"
	"PaintedSkin/widget/layout"
)

// TableColumn 描述一列：标题、固定宽（0=自动均分）、对齐方式。
type TableColumn struct {
	Title string
	Width int
	Align layout.Align
}

// Table 是表格（docs/08）：表头固定、主体滚动；垂直滚轮 + 行选择（同 List）、
// Enter 激活、点击选行；列总宽超视口时支持水平滚动。
type Table struct {
	Base
	columns    []TableColumn
	rows       [][]string
	selected   int
	offset     int
	hoff       int
	widths     []int
	onChange   func(row int)
	onActivate func(row int)
}

// NewTable 创建表格（选中初始 -1）。
func NewTable() *Table {
	t := &Table{selected: -1}
	t.Base = NewBase()
	t.Bind(t)
	t.Focusable = true
	return t
}

// SetColumns 设置列定义。
func (t *Table) SetColumns(cols []TableColumn) {
	t.columns = append([]TableColumn(nil), cols...)
	t.widths = nil
	t.Invalidate()
}

// Columns 返回列定义。
func (t *Table) Columns() []TableColumn { return t.columns }

// SetRows 设置数据（选中重置到首行）。
func (t *Table) SetRows(rows [][]string) {
	t.rows = append([][]string(nil), rows...)
	t.offset, t.hoff = 0, 0
	t.selected = -1
	if len(t.rows) > 0 {
		t.selected = 0
	}
	t.Invalidate()
}

// Rows 返回数据。
func (t *Table) Rows() [][]string { return t.rows }

// SetOnChange 设置选中行变更回调。
func (t *Table) SetOnChange(fn func(row int)) { t.onChange = fn }

// SetOnActivate 设置回车激活回调。
func (t *Table) SetOnActivate(fn func(row int)) { t.onActivate = fn }

// SetDisabled 开关禁用态；禁用时吞掉全部事件。
func (t *Table) SetDisabled(v bool) {
	t.Enabled = !v
	t.Invalidate()
}

// Disabled 报告禁用态。
func (t *Table) Disabled() bool { return !t.Enabled }

// SelectedRow 返回选中行（-1 表示无选中）。
func (t *Table) SelectedRow() int { return t.selected }

// Offset 返回主体垂直滚动偏移。
func (t *Table) Offset() int { return t.offset }

// HOffset 返回水平滚动偏移。
func (t *Table) HOffset() int { return t.hoff }

// SetSelectedRow 设置选中行（越界钳制；-1 允许）。
func (t *Table) SetSelectedRow(i int) {
	if len(t.rows) == 0 {
		return
	}
	n := i
	if n >= len(t.rows) {
		n = len(t.rows) - 1
	}
	if n < -1 {
		n = -1
	}
	if n == t.selected {
		return
	}
	t.selected = n
	t.scrollIntoView()
	t.Invalidate()
	if t.onChange != nil {
		t.onChange(n)
	}
}

// viewRows 返回主体视口行数（总高减表头）。
func (t *Table) viewRows() int {
	h := t.Bounds.H - 1
	if h < 0 {
		return 0
	}
	return h
}

// scrollIntoView 保证选中行在主体视口内。
func (t *Table) scrollIntoView() {
	rows := t.viewRows()
	if rows <= 0 || t.selected < 0 {
		return
	}
	if t.selected < t.offset {
		t.offset = t.selected
	}
	if t.selected >= t.offset+rows {
		t.offset = t.selected - rows + 1
	}
	t.offset = layout.ClampScroll(t.offset, len(t.rows), rows)
}

// contentWidth 返回列宽总和。
func (t *Table) contentWidth() int {
	w := 0
	for _, cw := range t.computeWidths(t.Bounds.W) {
		w += cw
	}
	return w
}

// maxHOffset 返回水平滚动上界。
func (t *Table) maxHOffset() int {
	m := t.contentWidth() - t.Bounds.W
	if m < 0 {
		return 0
	}
	return m
}

// computeWidths 计算各列实际宽：固定宽优先，0 宽列均分剩余。
func (t *Table) computeWidths(area int) []int {
	if len(t.columns) == 0 {
		return nil
	}
	out := make([]int, len(t.columns))
	used := 0
	autoCount := 0
	for i, col := range t.columns {
		if col.Width > 0 {
			out[i] = col.Width
			used += col.Width
		} else {
			autoCount++
		}
	}
	if autoCount > 0 {
		rest := area - used
		if rest < autoCount {
			rest = autoCount // 每个至少 1 格
		}
		base := rest / autoCount
		extra := rest % autoCount
		j := 0
		for i := range t.columns {
			if t.columns[i].Width > 0 {
				continue
			}
			out[i] = base
			if j < extra {
				out[i]++
				j++
			}
		}
	}
	return out
}

// ColumnWidths 返回当前生效的列宽。
func (t *Table) ColumnWidths() []int { return t.computeWidths(t.Bounds.W) }

// Measure 宽=列宽和（钳制），高=min(1+行数, MaxH)；空表高 0。
func (t *Table) Measure(c layout.Constraints) geom.Size {
	if len(t.columns) == 0 {
		return c.Clamp(geom.Size{W: 0, H: 0})
	}
	sum := 0
	for _, col := range t.columns {
		sum += col.Width
	}
	h := 1 + len(t.rows)
	if c.MaxH > 0 && h > c.MaxH {
		h = c.MaxH
	}
	return c.Clamp(geom.Size{W: sum, H: h})
}

// Layout 记录边界并计算列宽、同步偏移。
func (t *Table) Layout(r geom.Rect) {
	t.Bounds = r
	t.widths = t.computeWidths(r.W)
	t.hoff = layout.ClampScroll(t.hoff, t.contentWidth(), r.W)
	t.scrollIntoView()
}

// rowAt 返回 y 坐标对应的数据行索引（表头占 0 行）。
func (t *Table) rowAt(y int) int {
	if y < t.Bounds.Y+1 {
		return -1
	}
	idx := y - t.Bounds.Y - 1 + t.offset
	if idx < 0 || idx >= len(t.rows) {
		return -1
	}
	return idx
}

// Handle 垂直滚轮、水平滚动、选行、Enter 激活、点击；禁用全吞。
func (t *Table) Handle(e event.Event) bool {
	if !t.EffectivelyEnabled() {
		return true
	}
	switch ev := e.(type) {
	case event.Mouse:
		switch ev.Kind {
		case event.MouseWheelUp:
			t.setOffset(t.offset - 1)
			return true
		case event.MouseWheelDown:
			t.setOffset(t.offset + 1)
			return true
		case event.MousePress:
			if ev.Button != event.ButtonLeft {
				return false
			}
			if !t.Bounds.Contains(geom.Point{X: ev.X, Y: ev.Y}) {
				return false
			}
			if row := t.rowAt(ev.Y); row >= 0 {
				t.SetSelectedRow(row)
				if h := t.HostOf(); h != nil {
					h.Capture(t)
				}
			}
			return true
		}
	case event.Key:
		if !ev.Press || !t.Focused() {
			return false
		}
		switch ev.Code {
		case event.KeyDown:
			t.moveRow(1)
			return true
		case event.KeyUp:
			t.moveRow(-1)
			return true
		case event.KeyHome:
			t.SetSelectedRow(0)
			return true
		case event.KeyEnd:
			t.SetSelectedRow(len(t.rows) - 1)
			return true
		case event.KeyPgDn:
			t.moveRow(t.viewRows())
			return true
		case event.KeyPgUp:
			t.moveRow(-t.viewRows())
			return true
		case event.KeyLeft:
			t.setHOffset(t.hoff - 1)
			return true
		case event.KeyRight:
			t.setHOffset(t.hoff + 1)
			return true
		case event.KeyEnter:
			if t.onActivate != nil && t.selected >= 0 {
				t.onActivate(t.selected)
			}
			return true
		}
	}
	return false
}

func (t *Table) setOffset(v int) {
	n := layout.ClampScroll(v, len(t.rows), t.viewRows())
	if n == t.offset {
		return
	}
	t.offset = n
	t.Invalidate()
}

func (t *Table) setHOffset(v int) {
	n := layout.ClampScroll(v, t.contentWidth(), t.Bounds.W)
	if n > t.maxHOffset() {
		n = t.maxHOffset()
	}
	if n == t.hoff {
		return
	}
	t.hoff = n
	t.Invalidate()
}

func (t *Table) moveRow(d int) {
	if len(t.rows) == 0 {
		return
	}
	cur := t.selected
	if cur < 0 {
		cur = 0
	} else {
		cur += d
	}
	// 上下移动不产生"无选中"态：越界时钳到有效行。
	if cur < 0 {
		cur = 0
	}
	if cur >= len(t.rows) {
		cur = len(t.rows) - 1
	}
	t.SetSelectedRow(cur)
}

// Paint 画固定表头 + 可见数据行（按列宽与对齐绘制，水平裁剪）。
func (t *Table) Paint(c screen.Canvas) {
	b := t.Bounds
	if b.Empty() {
		return
	}
	base := t.Style
	if !t.EffectivelyEnabled() {
		base = t.disabledStyle(base)
	}
	paintBackground(c, b, base)
	widths := t.computeWidths(b.W)
	t.widths = widths

	// 表头（固定，不随 offset 滚动）
	x := b.X - t.hoff
	for i, col := range t.columns {
		if x >= b.Right() {
			break
		}
		cw := widths[i]
		cell := fitText(col.Title, cw, col.Align)
		// 表头走主题 TableHeader，未设时回落到加粗。
		hdrSt := base.WithBold(true)
		if themed := t.themeOrZero().TableHeader; themed != (style.Style{}) {
			hdrSt = applyAll(base, themed, nil)
		}
		c.Text(x, b.Y, cell, hdrSt)
		x += cw
	}

	// 数据行
	for i := 0; i < b.H-1; i++ {
		idx := t.offset + i
		if idx >= len(t.rows) {
			break
		}
		y := b.Y + 1 + i
		st := base
		switch {
		case idx == t.selected && t.Focused():
			st = t.focusStyle(st)
		case idx == t.selected:
			st = t.selectedStyle(st)
		}
		x = b.X - t.hoff
		for ci := range t.columns {
			if x >= b.Right() {
				break
			}
			cw := widths[ci]
			val := ""
			if ci < len(t.rows[idx]) {
				val = t.rows[idx][ci]
			}
			c.Text(x, y, fitText(val, cw, t.columns[ci].Align), st)
			x += cw
		}
	}
}
