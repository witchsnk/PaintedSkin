package widget

import (
	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/text"
	"PaintedSkin/widget/layout"
)

// Label 是基础展示控件（docs/08）：文本 + 对齐 + 自动换行 + 样式。
// 交互：无（不可聚焦，鼠标事件冒泡给父）。
type Label struct {
	Base
	text  string
	align layout.Align
	wrap  bool
}

// NewLabel 创建左对齐、单行 Label。
func NewLabel(text string) *Label {
	l := &Label{text: text, align: layout.AlignStart}
	l.Base = NewBase()
	l.Base.Bind(l)
	return l
}

// SetText 设置文本并打脏。
func (l *Label) SetText(s string) {
	l.text = s
	l.Invalidate()
}

// Text 返回文本。
func (l *Label) Text() string { return l.text }

// SetAlign 设置对齐（AlignStart/AlignCenter/AlignEnd）。
func (l *Label) SetAlign(a layout.Align) {
	l.align = a
	l.Invalidate()
}

// Align 返回对齐方式。
func (l *Label) Align() layout.Align { return l.align }

// SetWrap 开关自动换行。
func (l *Label) SetWrap(v bool) {
	l.wrap = v
	l.Invalidate()
}

// Wrap 报告是否自动换行。
func (l *Label) Wrap() bool { return l.wrap }

// lines 按当前换行设置把文本折成行（按 text.Width 的列宽折行）。
func (l *Label) lines(width int) []string {
	if !l.wrap || width <= 0 {
		return splitLines(l.text)
	}
	var out []string
	for _, para := range splitLines(l.text) {
		out = append(out, foldLine(para, width)...)
	}
	if len(out) == 0 {
		return []string{""}
	}
	return out
}

// Measure 单行 = 文本宽×1；换行 = 约束宽内多行最大宽×行数。
func (l *Label) Measure(c layout.Constraints) geom.Size {
	maxW := c.MaxW
	if maxW < 0 {
		maxW = 0
	}
	lines := l.lines(maxW)
	w := 0
	for _, ln := range lines {
		if sw := text.Width(ln); sw > w {
			w = sw
		}
	}
	return c.Clamp(geom.Size{W: w, H: len(lines)})
}

// Layout 记录边界。
func (l *Label) Layout(r geom.Rect) { l.Bounds = r }

// Paint 先铺自身背景，再按对齐绘制各行；每行按区域宽截断，长行不越界。
func (l *Label) Paint(c screen.Canvas) {
	if l.Bounds.Empty() {
		return
	}
	paintBackground(c, l.Bounds, l.Style)
	for i, ln := range l.lines(l.Bounds.W) {
		y := l.Bounds.Y + i
		if y >= l.Bounds.Bottom() {
			break
		}
		off := alignOffset(l.align, l.Bounds.W, text.Width(ln))
		// 对齐偏移已把行首定位到区内，剩余列宽即本行可用预算；
		// text.Truncate 保证超长行不会越出区域（尾部空档由上面的铺底负责）。
		c.Text(l.Bounds.X+off, y, text.Truncate(ln, l.Bounds.W-off), l.Style)
	}
}

// Handle 恒不消费（冒泡给父）。
func (l *Label) Handle(event.Event) bool { return false }

func alignOffset(a layout.Align, total, used int) int {
	if used >= total {
		return 0
	}
	switch a {
	case layout.AlignCenter:
		return (total - used) / 2
	case layout.AlignEnd:
		return total - used
	default:
		return 0
	}
}

func splitLines(s string) []string {
	if s == "" {
		return []string{""}
	}
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	return append(out, s[start:])
}

// foldLine 把单段按显示宽度折行（贪心，不切分宽字符）。
func foldLine(s string, width int) []string {
	if width <= 0 {
		return []string{s}
	}
	var out []string
	cur := ""
	curW := 0
	for _, r := range s {
		rw := text.RuneWidth(r)
		if curW+rw > width && cur != "" {
			out = append(out, cur)
			cur, curW = "", 0
		}
		cur += string(r)
		curW += rw
	}
	out = append(out, cur)
	return out
}
