package widget

import (
	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/widget/layout"
)

// Separator 是分隔线控件（docs/08）：方向（H/V）+ 线字符，固定 1 格；交互无。
type Separator struct {
	Base
	horizontal bool
	line       rune
}

// NewSeparator 创建水平分隔线（线字符留空，按方向取默认）。
func NewSeparator() *Separator {
	s := &Separator{horizontal: true}
	s.Base = NewBase()
	s.Base.Bind(s)
	return s
}

// SetHorizontal 设置方向（true=水平，false=垂直）。
func (s *Separator) SetHorizontal(v bool) {
	s.horizontal = v
	s.Invalidate()
}

// Horizontal 报告是否水平。
func (s *Separator) Horizontal() bool { return s.horizontal }

// SetLine 设置线字符（0 时按方向回落默认）。
func (s *Separator) SetLine(r rune) {
	s.line = r
	s.Invalidate()
}

// Line 返回当前线字符（0 表示按方向取默认）。
func (s *Separator) Line() rune { return s.line }

func (s *Separator) glyph() rune {
	if s.line != 0 {
		return s.line
	}
	return s.defaultGlyph()
}

// defaultGlyph 取边框表里的线字形（字形的唯一真相源在 screen/border.go）。
func (s *Separator) defaultGlyph() rune {
	if s.horizontal {
		return screen.BorderSingle.HLine()
	}
	return screen.BorderSingle.VLine()
}

// SetDisabled 开关禁用态；禁用时吞掉全部事件。
func (s *Separator) SetDisabled(v bool) {
	s.Enabled = !v
	s.Invalidate()
}

// Disabled 报告禁用态。
func (s *Separator) Disabled() bool { return !s.Enabled }

// Measure 水平高 1 格（宽取 MaxW，MaxW<0 时取 0）；垂直宽 1 格（高取 MaxH，
// MaxH<0 时取 0）。"固定 1 格"的那一维恒为 1，不随约束退化。
func (s *Separator) Measure(c layout.Constraints) geom.Size {
	if s.horizontal {
		return geom.Size{W: clampDim(c.MaxW, c.MinW), H: 1}
	}
	return geom.Size{W: 1, H: clampDim(c.MaxH, c.MinH)}
}

// clampDim 把可选上界钳进 [min,∞)，负上界视为 0。
func clampDim(max, min int) int {
	w := max
	if w < 0 {
		w = 0
	}
	if w < min {
		w = min
	}
	return w
}

// Layout 记录边界。
func (s *Separator) Layout(r geom.Rect) { s.Bounds = r }

// Paint 沿线长重复线字符（禁用时暗淡）。
func (s *Separator) Paint(c screen.Canvas) {
	b := s.Bounds
	if b.Empty() {
		return
	}
	st := applyAll(s.Style, s.themeOrZero().Separator, nil)
	st = s.disabledStyle(st)
	paintBackground(c, b, st)
	g := string(s.glyph())
	if s.horizontal {
		c.Text(b.X, b.Y, screen.Bar.Line(s.glyph(), b.W), st)
		return
	}
	for i := 0; i < b.H; i++ {
		c.Text(b.X, b.Y+i, g, st)
	}
}

// Handle 无交互：禁用吞事件，否则不消费（冒泡）。
func (s *Separator) Handle(event.Event) bool { return !s.EffectivelyEnabled() }
