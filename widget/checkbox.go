package widget

import (
	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/text"
	"PaintedSkin/widget/layout"
)

// Checkbox 是复选框控件（M6 范围；本文件仅做新 Widget 接口的编译迁移，
// 行为与旧实现一致，不扩展）。
type Checkbox struct {
	Base
	text     string
	onChange func(checked bool)
	checked  bool
	hover    bool
	pressed  bool
}

// NewCheckbox 创建复选框。
func NewCheckbox(text string) *Checkbox {
	c := &Checkbox{text: text}
	c.Base = NewBase()
	c.Base.Bind(c)
	c.Focusable = true
	return c
}

// SetText 设置文本。
func (c *Checkbox) SetText(text string) {
	c.text = text
	c.Invalidate()
}

// Text 返回文本。
func (c *Checkbox) Text() string { return c.text }

// SetOnChange 设置状态变更回调。
func (c *Checkbox) SetOnChange(fn func(checked bool)) { c.onChange = fn }

// SetChecked 直接设置选中状态。
func (c *Checkbox) SetChecked(v bool) {
	c.checked = v
	c.Invalidate()
}

// Checked 报告选中态。
func (c *Checkbox) Checked() bool { return c.checked }

// SetDisabled 开关禁用态；禁用时吞掉全部事件。
func (c *Checkbox) SetDisabled(v bool) {
	c.Enabled = !v
	c.Invalidate()
}

// Disabled 报告禁用态。
func (c *Checkbox) Disabled() bool { return !c.Enabled }

// OnBlur 覆盖：失焦复位按下态。
func (c *Checkbox) OnBlur() {
	c.Base.OnBlur()
	c.pressed = false
}

func (c *Checkbox) toggle() {
	c.checked = !c.checked
	c.Invalidate()
	if c.onChange != nil {
		c.onChange(c.checked)
	}
}

// Measure 复选框 + 文本 + padding。
func (c *Checkbox) Measure(lc layout.Constraints) geom.Size {
	return lc.Clamp(geom.Size{W: text.Width(c.text) + 2*markCapW + 2*markPadW, H: 1})
}

// Layout 记录边界。
func (c *Checkbox) Layout(r geom.Rect) { c.Bounds = r }

// Paint 以 `[X] 文字` 形式绘制；铺底 + 补齐到控件宽（焦点反色带覆盖整宽）。
func (c *Checkbox) Paint(cnv screen.Canvas) {
	if c.Bounds.Empty() {
		return
	}
	mark := ' '
	if c.checked {
		mark = 'X'
	}
	// 四种语义状态走主题（focus/hover/selected/disabled），无主题时回落到
	// 内置的"反色/加粗/暗淡"。
	st := c.Style
	switch {
	case !c.EffectivelyEnabled():
		st = c.disabledStyle(st)
	case c.pressed || c.Focused():
		st = c.focusStyle(st)
	case c.hover:
		st = c.hoverStyle(st)
	}
	if c.checked && st == c.Style {
		st = c.selectedStyle(st)
	}
	label := "[" + string(mark) + "] " + c.text
	paintBackground(cnv, c.Bounds, st)
	cnv.Text(c.Bounds.X, c.Bounds.Y, text.Fit(label, c.Bounds.W, text.Left), st)
}

// Handle 空格/点击切换；禁用全吞。
func (c *Checkbox) Handle(ev event.Event) bool {
	if !c.EffectivelyEnabled() {
		return true
	}
	switch e := ev.(type) {
	case event.Key:
		if e.Press && c.Focused() && (e.Code == event.KeyEnter || e.Code == event.KeySpace) {
			c.toggle()
			return true
		}
	case event.Mouse:
		inside := c.Bounds.Contains(geom.Point{X: e.X, Y: e.Y})
		switch e.Kind {
		case event.MouseEnter:
			c.hover = true
			c.Invalidate()
			return true
		case event.MouseLeave:
			c.hover = false
			c.Invalidate()
			return true
		case event.MousePress:
			if e.Button == event.ButtonLeft && inside {
				c.pressed = true
				c.Invalidate()
				if h := c.HostOf(); h != nil {
					h.Capture(c)
				}
				return true
			}
		case event.MouseRelease:
			if c.pressed {
				c.pressed = false
				if inside {
					c.toggle()
				} else {
					c.Invalidate()
				}
				return true
			}
		case event.MouseMove:
			c.hover = inside
			return true
		}
	}
	return false
}
