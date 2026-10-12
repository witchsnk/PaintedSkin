package widget

import (
	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/text"
	"PaintedSkin/widget/layout"
)

// Radio 是单选按钮（docs/08）：点击选中（不再 toggle）；Space/Enter 同样选中。
// 成员自身不占 Tab 位（组内不逐个占位），互斥与方向键由 RadioGroup 负责。
type Radio struct {
	Base
	group    string
	text     string
	checked  bool
	onChange func(checked bool)
	hover    bool
	pressed  bool
}

// NewRadio 创建单选按钮。
func NewRadio(text string) *Radio {
	r := &Radio{text: text}
	r.Base = NewBase()
	r.Base.Bind(r)
	r.Focusable = false // 组内不占 Tab 位
	return r
}

// SetText 设置文本。
func (r *Radio) SetText(text string) {
	r.text = text
	r.Invalidate()
}

// Text 返回文本。
func (r *Radio) Text() string { return r.text }

// SetGroup 设置组名。
func (r *Radio) SetGroup(group string) { r.group = group }

// Group 返回组名。
func (r *Radio) Group() string { return r.group }

// SetOnChange 设置选中回调。
func (r *Radio) SetOnChange(fn func(checked bool)) { r.onChange = fn }

// SetChecked 直接设置选中态。
func (r *Radio) SetChecked(v bool) {
	r.checked = v
	r.Invalidate()
}

// Checked 报告选中态。
func (r *Radio) Checked() bool { return r.checked }

// SetDisabled 开关禁用态；禁用时吞掉全部事件。
func (r *Radio) SetDisabled(v bool) {
	r.Enabled = !v
	r.Invalidate()
}

// Disabled 报告禁用态。
func (r *Radio) Disabled() bool { return !r.Enabled }

// OnBlur 覆盖：失焦复位按下态。
func (r *Radio) OnBlur() {
	r.Base.OnBlur()
	r.pressed = false
}

// markSelected 置选中并回调（幂等：已选中不重复回调）。
func (r *Radio) markSelected() {
	if r.checked {
		return
	}
	r.checked = true
	r.Invalidate()
	if r.onChange != nil {
		r.onChange(true)
	}
}

// Measure 单选标记 + 文本 + padding。
func (r *Radio) Measure(lc layout.Constraints) geom.Size {
	return lc.Clamp(geom.Size{W: text.Width(r.text) + 2*markCapW + 2*markPadW, H: 1})
}

// Layout 记录边界。
func (r *Radio) Layout(b geom.Rect) { r.Bounds = b }

// Paint 以 `(x) 文字` 形式绘制；铺底 + 补齐到控件宽（状态色带覆盖整宽）。
func (r *Radio) Paint(cnv screen.Canvas) {
	if r.Bounds.Empty() {
		return
	}
	mark := " "
	if r.checked {
		mark = "*"
	}
	// 四种语义状态走主题（focus/hover/selected/disabled）。
	st := r.Style
	switch {
	case !r.EffectivelyEnabled():
		st = r.disabledStyle(st)
	case r.pressed:
		st = r.focusStyle(st)
	case r.hover:
		st = r.hoverStyle(st)
	}
	if r.checked && st == r.Style {
		st = r.selectedStyle(st)
	}
	label := "(" + mark + ") " + r.text
	paintBackground(cnv, r.Bounds, st)
	cnv.Text(r.Bounds.X, r.Bounds.Y, text.Fit(label, r.Bounds.W, text.Left), st)
}

// Handle 点击/Space/Enter 均为"选中"语义；禁用全吞。
func (r *Radio) Handle(ev event.Event) bool {
	if !r.EffectivelyEnabled() {
		return true
	}
	switch e := ev.(type) {
	case event.Key:
		if e.Press && (e.Code == event.KeySpace || e.Code == event.KeyEnter) {
			r.markSelected()
			return true
		}
	case event.Mouse:
		inside := r.Bounds.Contains(geom.Point{X: e.X, Y: e.Y})
		switch e.Kind {
		case event.MouseEnter:
			r.hover = true
			r.Invalidate()
			return true
		case event.MouseLeave:
			r.hover = false
			r.Invalidate()
			return true
		case event.MousePress:
			if e.Button == event.ButtonLeft && inside {
				r.pressed = true
				r.Invalidate()
				if h := r.HostOf(); h != nil {
					h.Capture(r)
				}
				return true
			}
		case event.MouseRelease:
			if r.pressed {
				r.pressed = false
				if inside {
					r.markSelected()
				} else {
					r.Invalidate()
				}
				return true
			}
		case event.MouseMove:
			r.hover = inside
			return true
		}
	}
	return false
}
