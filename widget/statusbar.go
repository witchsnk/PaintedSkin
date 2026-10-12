package widget

import (
	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/style"
	"PaintedSkin/text"
	"PaintedSkin/widget/layout"
)

// StatusBar 是常驻底行状态栏（docs/08）：左/中/右三段文本；交互无。
type StatusBar struct {
	Base
	left, center, right       string
	leftSt, centerSt, rightSt style.Style
}

// NewStatusBar 创建状态栏（三段为空，H=1）。
func NewStatusBar() *StatusBar {
	s := &StatusBar{}
	s.Base = NewBase()
	s.Base.Bind(s)
	return s
}

// SetLeft 设置左段文本。
func (s *StatusBar) SetLeft(v string) { s.left = v; s.Invalidate() }

// Left 返回左段文本。
func (s *StatusBar) Left() string { return s.left }

// SetCenter 设置中段文本。
func (s *StatusBar) SetCenter(v string) { s.center = v; s.Invalidate() }

// Center 返回中段文本。
func (s *StatusBar) Center() string { return s.center }

// SetRight 设置右段文本。
func (s *StatusBar) SetRight(v string) { s.right = v; s.Invalidate() }

// Right 返回右段文本。
func (s *StatusBar) Right() string { return s.right }

// SetLeftStyle 设置左段样式（零值表示回落控件 Style）。
func (s *StatusBar) SetLeftStyle(st style.Style) { s.leftSt = st; s.Invalidate() }

// SetCenterStyle 设置中段样式。
func (s *StatusBar) SetCenterStyle(st style.Style) { s.centerSt = st; s.Invalidate() }

// SetRightStyle 设置右段样式。
func (s *StatusBar) SetRightStyle(st style.Style) { s.rightSt = st; s.Invalidate() }

// SetDisabled 开关禁用态；禁用时吞掉全部事件。
func (s *StatusBar) SetDisabled(v bool) {
	s.Enabled = !v
	s.Invalidate()
}

// Disabled 报告禁用态。
func (s *StatusBar) Disabled() bool { return !s.Enabled }

func (s *StatusBar) styleFor(seg style.Style) style.Style {
	if seg != (style.Style{}) {
		return seg
	}
	return s.Style
}

// Measure H 恒 1（"常驻底行"，不随 MaxH 退化）；宽取 MaxW，
// MaxW<0（无上界）时按三段显示宽之和。
func (s *StatusBar) Measure(c layout.Constraints) geom.Size {
	w := 0
	if c.MaxW >= 0 {
		w = c.MaxW
	} else {
		for _, seg := range []string{s.left, s.center, s.right} {
			w += text.Width(seg)
		}
	}
	if w < c.MinW {
		w = c.MinW
	}
	if w < 0 {
		w = 0
	}
	return geom.Size{W: w, H: 1}
}

// Layout 记录边界。
func (s *StatusBar) Layout(r geom.Rect) { s.Bounds = r }

// Paint 先铺满自身背景，再按左/中/右三段绘制。
//
// 三段各自截断到不重叠的区间（右 > 中 > 左 优先保位）：中段在"左段之后、
// 右段之前"的空隙里居中，因此段间既不留空洞、也不会互相压过。
func (s *StatusBar) Paint(c screen.Canvas) {
	b := s.Bounds
	if b.Empty() {
		return
	}
	// 段级样式优先，其次控件 Style，最后主题 StatusBar。
	base := applyAll(s.Style, s.themeOrZero().StatusBar, nil)
	base = s.disabledStyle(base)
	dim := base.Dim
	paintBackground(c, b, base)
	if s.left == "" && s.center == "" && s.right == "" {
		return
	}
	// 分配互不重叠的区间宽度：先保右段，再保中段，剩余给左段。
	rw := minInt(text.Width(s.right), b.W)
	cw := minInt(text.Width(s.center), b.W-rw)
	lw := minInt(text.Width(s.left), b.W-rw-cw)
	free := b.W - rw - cw - lw // 左段之后、右段之前的空隙
	if s.left != "" {
		c.Text(b.X, b.Y, text.Fit(s.left, lw, text.Left), withDim(s.styleFor(s.leftSt), dim))
	}
	if s.center != "" {
		c.Text(b.X+lw+free/2, b.Y, text.Fit(s.center, cw, text.Left),
			withDim(s.styleFor(s.centerSt), dim))
	}
	if s.right != "" {
		c.Text(b.Right()-rw, b.Y, text.Fit(s.right, rw, text.Left),
			withDim(s.styleFor(s.rightSt), dim))
	}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Handle 无交互：禁用吞事件，否则不消费（冒泡）。
func (s *StatusBar) Handle(event.Event) bool { return !s.EffectivelyEnabled() }

// withDim 按需叠加暗淡。
func withDim(st style.Style, dim bool) style.Style {
	if dim {
		return st.WithDim(true)
	}
	return st
}
