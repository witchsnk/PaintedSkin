package widget

import (
	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/style"
	"PaintedSkin/text"
	"PaintedSkin/widget/layout"
)

// Button 是可点击按钮（docs/08）：normal / hover / pressed / focused /
// disabled 五态（hover 与 focus 可叠加）；禁用时全吞事件。
type Button struct {
	Base
	text    string
	onClick func()
	hover   bool
	pressed bool
	fixedW  int // >0 时为固定宽，否则文本 + 左右 padding
}

// NewButton 创建聚焦可达的按钮。
func NewButton(text string) *Button {
	b := &Button{text: text}
	b.Base = NewBase()
	b.Base.Bind(b)
	b.Focusable = true
	return b
}

// SetText 设置文本并打脏。
func (b *Button) SetText(s string) {
	b.text = s
	b.Invalidate()
}

// Text 返回文本。
func (b *Button) Text() string { return b.text }

// SetOnClick 设置点击回调（界内 Release / 聚焦 Enter/Space 时触发）。
func (b *Button) SetOnClick(fn func()) { b.onClick = fn }

// SetWidth 设置固定宽（0 = 自适应文本 + padding）。
func (b *Button) SetWidth(w int) {
	if w < 0 {
		w = 0
	}
	b.fixedW = w
	b.Invalidate()
}

// Width 返回固定宽（0 表示自适应）。
func (b *Button) Width() int { return b.fixedW }

// SetDisabled 开关禁用态；禁用时吞掉全部事件。
func (b *Button) SetDisabled(v bool) {
	b.Enabled = !v
	b.Invalidate()
}

// Disabled 报告是否为禁用态。
func (b *Button) Disabled() bool { return !b.Enabled }

// Hover 报告悬停态（测试用）。
func (b *Button) Hover() bool { return b.hover }

// Pressed 报告按下态（测试用）。
func (b *Button) Pressed() bool { return b.pressed }

// OnBlur 覆盖：失焦复位按下态。
func (b *Button) OnBlur() {
	b.Base.OnBlur()
	b.pressed = false
}

// Measure 文本 + 左右 padding（各一格 + 方括号形式共 4 格）；可设固定宽。
func (b *Button) Measure(c layout.Constraints) geom.Size {
	w := text.Width(b.text) + 2*markCapW + 2*markPadW
	if b.fixedW > 0 {
		w = b.fixedW
	}
	return c.Clamp(geom.Size{W: w, H: 1})
}

// Layout 记录边界。
func (b *Button) Layout(r geom.Rect) { b.Bounds = r }

// Paint 以 `[ 文本 ]` 形式绘制，五态叠样式（disabled > pressed/focused > hover）。
// 先铺控件自身背景，再把标签补齐到控件宽（尾部空档不带样式就是空洞）。
func (b *Button) Paint(c screen.Canvas) {
	if b.Bounds.Empty() {
		return
	}
	label := "[" + b.text + "]"
	// 五态样式：主题字段优先，未设则回落到内置默认态（保证任何配置下可区分）。
	th := b.themeOrZero()
	normal := th.ButtonNormal
	hover := b.themed(th.ButtonHover, style.Style{}.WithBold(true))
	pressedSt := b.themed(th.ButtonPressed, style.Style{}.WithAttrs(style.AttrReverse))
	focusedSt := b.themed(th.ButtonFocused, style.Style{}.WithAttrs(style.AttrReverse))
	// 控件自身 Style 优先，未设则回落主题（docs/08）。
	st := b.pick(b.Style, normal)
	switch {
	case !b.EffectivelyEnabled():
		st = applyAll(b.disabledStyle(st), th.ButtonDisabled, nil)
	case b.pressed:
		st = applyAll(st, pressedSt, nil)
	case b.Focused():
		st = applyAll(st, focusedSt, nil)
	case b.hover:
		st = applyAll(st, hover, nil)
	}
	paintBackground(c, b.Bounds, st)
	// 标签补齐到控件宽：焦点/按下的反色带覆盖整个按钮，而不是停在文字末尾。
	c.Text(b.Bounds.X, b.Bounds.Y, text.Fit(label, b.Bounds.W, text.Left), st)
}

// Handle 实现五态交互（见 docs/08）。
func (b *Button) Handle(e event.Event) bool {
	if !b.EffectivelyEnabled() {
		return true // 禁用时全吞事件
	}
	switch ev := e.(type) {
	case event.Mouse:
		inside := b.Bounds.Contains(geom.Point{X: ev.X, Y: ev.Y})
		switch ev.Kind {
		case event.MouseEnter:
			b.hover = true
			b.Invalidate()
			return true
		case event.MouseLeave:
			b.hover = false
			b.Invalidate()
			return true
		case event.MousePress:
			if ev.Button == event.ButtonLeft && inside {
				b.pressed = true
				b.Invalidate()
				if h := b.HostOf(); h != nil {
					h.Capture(b)
				}
				return true
			}
			return false
		case event.MouseRelease:
			if b.pressed {
				b.pressed = false
				b.Invalidate()
				if inside && b.onClick != nil {
					b.onClick()
				}
				return true
			}
			return false
		case event.MouseMove:
			b.hover = inside
			return true
		}
	case event.Key:
		if ev.Press && b.Focused() && (ev.Code == event.KeyEnter || ev.Code == event.KeySpace) {
			if b.onClick != nil {
				b.onClick()
			}
			return true
		}
	}
	return false
}
