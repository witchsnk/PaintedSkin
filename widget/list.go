package widget

import (
	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/style"
	"PaintedSkin/text"
	"PaintedSkin/widget/layout"
)

// List 是列表控件（docs/08）：虚拟滚动（只画可见行）、滚轮滚动、
// 上下键移动选择并自动滚入视、Enter 激活、点击选中、右侧细滚动条指示。
type List struct {
	Base
	items      []string
	selected   int
	offset     int
	onChange   func(index int)
	onActivate func(index int)
}

// NewList 创建列表。
func NewList() *List {
	l := &List{selected: -1}
	l.Base = NewBase()
	l.Bind(l)
	l.Focusable = true
	return l
}

// SetItems 设置数据（选中重置到首项）。
func (l *List) SetItems(items []string) {
	l.items = append([]string(nil), items...)
	l.offset = 0
	l.selected = -1
	if len(l.items) > 0 {
		l.selected = 0
	}
	l.Invalidate()
}

// Items 返回数据。
func (l *List) Items() []string { return l.items }

// Len 返回项数。
func (l *List) Len() int { return len(l.items) }

// SetOnChange 设置选中变更回调。
func (l *List) SetOnChange(fn func(index int)) { l.onChange = fn }

// SetOnActivate 设置回车激活回调。
func (l *List) SetOnActivate(fn func(index int)) { l.onActivate = fn }

// SetDisabled 开关禁用态；禁用时吞掉全部事件。
func (l *List) SetDisabled(v bool) {
	l.Enabled = !v
	l.Invalidate()
}

// Disabled 报告禁用态。
func (l *List) Disabled() bool { return !l.Enabled }

// viewRows 返回视口行数。
func (l *List) viewRows() int {
	if l.Bounds.H < 0 {
		return 0
	}
	return l.Bounds.H
}

// SetSelected 设置选中索引（越界钳制；-1 表示无选中）。
func (l *List) SetSelected(i int) {
	if l.items == nil {
		return
	}
	n := i
	if n >= len(l.items) {
		n = len(l.items) - 1
	}
	if n < -1 {
		n = -1
	}
	if n == l.selected {
		return
	}
	l.selected = n
	l.scrollIntoView()
	l.Invalidate()
	if l.onChange != nil {
		l.onChange(n)
	}
}

// Selected 返回选中索引（-1 表示无选中）。
func (l *List) Selected() int { return l.selected }

// SetOffset 设置滚动偏移（钳制）。
func (l *List) SetOffset(v int) {
	l.offset = layout.ClampScroll(v, len(l.items), l.viewRows())
	l.Invalidate()
}

// Offset 返回滚动偏移。
func (l *List) Offset() int { return l.offset }

// scrollIntoView 保证选中行在视口内。
func (l *List) scrollIntoView() {
	rows := l.viewRows()
	if rows <= 0 || l.selected < 0 {
		return
	}
	if l.selected < l.offset {
		l.offset = l.selected
	}
	if l.selected >= l.offset+rows {
		l.offset = l.selected - rows + 1
	}
	l.offset = layout.ClampScroll(l.offset, len(l.items), rows)
}

// Measure 宽 = 最长项宽 + 滚动条 + 1；高 = min(项数, MaxH)；空列表高 0。
func (l *List) Measure(c layout.Constraints) geom.Size {
	if len(l.items) == 0 {
		return c.Clamp(geom.Size{W: 0, H: 0})
	}
	maxW := 0
	for _, it := range l.items {
		if w := text.Width(it); w > maxW {
			maxW = w
		}
	}
	h := len(l.items)
	if c.MaxH > 0 && h > c.MaxH {
		h = c.MaxH
	}
	return c.Clamp(geom.Size{W: maxW + listScrollbarW + listGapW, H: h})
}

// Layout 记录边界并同步偏移。
func (l *List) Layout(r geom.Rect) {
	l.Bounds = r
	l.scrollIntoView()
}

// Handle 滚轮/方向键/Enter/点击选中；禁用全吞。
func (l *List) Handle(e event.Event) bool {
	if !l.EffectivelyEnabled() {
		return true
	}
	switch ev := e.(type) {
	case event.Mouse:
		switch ev.Kind {
		case event.MouseWheelUp:
			l.SetOffset(l.offset - 1)
			return true
		case event.MouseWheelDown:
			l.SetOffset(l.offset + 1)
			return true
		case event.MousePress:
			if ev.Button != event.ButtonLeft {
				return false
			}
			if !l.Bounds.Contains(geom.Point{X: ev.X, Y: ev.Y}) {
				return false
			}
			row := ev.Y - l.Bounds.Y + l.offset
			if row >= 0 && row < len(l.items) {
				l.SetSelected(row)
				if h := l.HostOf(); h != nil {
					h.Capture(l)
				}
			}
			return true
		case event.MouseRelease:
			return false
		}
	case event.Key:
		if !ev.Press || !l.Focused() || len(l.items) == 0 {
			return false
		}
		switch ev.Code {
		case event.KeyUp:
			l.move(-1)
			return true
		case event.KeyDown:
			l.move(1)
			return true
		case event.KeyHome:
			l.SetSelected(0)
			return true
		case event.KeyEnd:
			l.SetSelected(len(l.items) - 1)
			return true
		case event.KeyPgUp:
			l.move(-l.viewRows())
			return true
		case event.KeyPgDn:
			l.move(l.viewRows())
			return true
		case event.KeyEnter:
			if l.onActivate != nil && l.selected >= 0 {
				l.onActivate(l.selected)
			}
			return true
		}
	}
	return false
}

// move 相对移动选中。
func (l *List) move(d int) {
	if len(l.items) == 0 {
		return
	}
	cur := l.selected
	if cur < 0 {
		if d < 0 {
			cur = len(l.items) - 1
		} else {
			cur = 0
		}
	} else {
		cur += d
	}
	l.SetSelected(cur)
}

// Paint 铺底 → 逐行画可见项 → 画右侧滚动条指示。
// 选中/焦点行补齐到滚动条列之前，反色带覆盖整行；项数少于视口时空行也铺底。
func (l *List) Paint(c screen.Canvas) {
	b := l.Bounds
	if b.Empty() {
		return
	}
	base := l.Style
	if !l.EffectivelyEnabled() {
		base = l.disabledStyle(base)
	}
	paintBackground(c, b, base)
	textW := b.W - listScrollbarW // 右侧留 1 格给滚动条
	if textW <= 0 {
		textW = b.W
	}
	for i := 0; i < b.H; i++ {
		idx := l.offset + i
		if idx >= len(l.items) {
			break
		}
		y := b.Y + i
		sel := idx == l.selected
		st := base
		// 选中行以主题 ListSelected 为权威（含焦点态）；未设则回落到
		// 通用选中语义 SelectionBG / 内置反色。
		if sel {
			if themed := l.themeOrZero().ListSelected; themed != (style.Style{}) {
				st = applyAll(st, themed, nil)
			} else {
				st = l.selectedStyle(st)
			}
		}
		// text.Fit = 截断到 textW + 补齐到 textW：选中态的背景/反色到行尾。
		c.Text(b.X, y, text.Fit(l.items[idx], textW, text.Left), st)
	}
	l.paintScrollbar(c, b, base)
}

// paintScrollbar 画右侧滚动条指示（非交互）。
func (l *List) paintScrollbar(c screen.Canvas, b geom.Rect, st style.Style) {
	if b.W < 2 {
		return
	}
	total, view := len(l.items), b.H
	if total <= view {
		return
	}
	span := total - view
	pos := 0
	if span > 0 {
		pos = l.offset * (b.H - 1) / span
	}
	thumb := b.H * view / total
	if thumb < 1 {
		thumb = 1
	}
	x := b.Right() - 1
	for i := 0; i < b.H; i++ {
		g := screen.Bar.Track
		if i >= pos && i < pos+thumb {
			g = screen.Bar.Filled
		}
		c.SetCell(x, b.Y+i, g, st)
	}
}
