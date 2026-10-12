package widget

import (
	"strconv"
	"strings"

	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/text"
	"PaintedSkin/widget/layout"
)

// ProgressBar 是进度条（docs/08）：Min/Max/Value + 是否显示百分比文本；交互无。
type ProgressBar struct {
	Base
	min, max    int
	value       int
	format      string
	showPercent bool
	border      BorderStyle
	onChange    func(value int)
}

// NewProgressBar 创建进度条（0..100，默认值 0）。
func NewProgressBar() *ProgressBar {
	p := &ProgressBar{max: 100, border: BorderHeavy}
	p.Base = NewBase()
	p.Base.Bind(p)
	p.Focusable = false // 交互：无
	return p
}

// Progress 是 ProgressBar 的兼容别名（docs/08 名称为 ProgressBar）。
type Progress = ProgressBar

// NewProgress 是 NewProgressBar 的兼容别名。
func NewProgress() *ProgressBar { return NewProgressBar() }

// SetRange 设置范围（钳制当前值）。
func (p *ProgressBar) SetRange(min, max int) {
	if max < min {
		max = min
	}
	p.min, p.max = min, max
	p.SetValue(p.value)
}

// Min 返回下界。
func (p *ProgressBar) Min() int { return p.min }

// Max 返回上界。
func (p *ProgressBar) Max() int { return p.max }

// SetValue 设置进度（钳制进范围）并打脏。
func (p *ProgressBar) SetValue(v int) {
	if v < p.min {
		v = p.min
	}
	if v > p.max {
		v = p.max
	}
	p.value = v
	p.Invalidate()
	if p.onChange != nil {
		p.onChange(v)
	}
}

// Value 返回当前值。
func (p *ProgressBar) Value() int { return p.value }

// SetFormat 设置百分比文本格式（默认 "NN%"）。
func (p *ProgressBar) SetFormat(f string) {
	p.format = f
	p.Invalidate()
}

// SetShowPercent 设置是否显示百分比文本。
func (p *ProgressBar) SetShowPercent(v bool) {
	p.showPercent = v
	p.Invalidate()
}

// ShowPercent 报告是否显示百分比。
func (p *ProgressBar) ShowPercent() bool { return p.showPercent }

// SetBorder 设置边框样式（枚举档；默认 BorderHeavy，与改动前视觉一致）。
func (p *ProgressBar) SetBorder(s BorderStyle) {
	p.border = s
	p.Invalidate()
}

// Border 返回边框样式。
func (p *ProgressBar) Border() BorderStyle { return p.border }

// SetOnChange 设置值变更回调。
func (p *ProgressBar) SetOnChange(fn func(value int)) { p.onChange = fn }

// SetDisabled 开关禁用态；禁用时吞掉全部事件。
func (p *ProgressBar) SetDisabled(v bool) {
	p.Enabled = !v
	p.Invalidate()
}

// Disabled 报告禁用态。
func (p *ProgressBar) Disabled() bool { return !p.Enabled }

// ratio 返回 0..1 进度比。
func (p *ProgressBar) ratio() float64 {
	span := p.max - p.min
	if span <= 0 {
		return 0
	}
	r := float64(p.value-p.min) / float64(span)
	if r < 0 {
		return 0
	}
	if r > 1 {
		return 1
	}
	return r
}

// Measure 首选 20×3（钳制进约束）。
func (p *ProgressBar) Measure(c layout.Constraints) geom.Size {
	return c.Clamp(geom.Size{W: progressPrefW, H: progressPrefH})
}

// Layout 记录边界。
func (p *ProgressBar) Layout(r geom.Rect) { p.Bounds = r }

// Paint 三行高画框 + 填充块 + 可选百分比；单行高退化为 `[===>  ]`。
func (p *ProgressBar) Paint(c screen.Canvas) {
	b := p.Bounds
	if b.W < minFrameW || b.H < minFrameH {
		return
	}
	st := p.Style
	if !p.EffectivelyEnabled() {
		st = p.disabledStyle(st)
	}
	paintBackground(c, b, st)
	filled := int(float64(b.W-2) * p.ratio())
	if filled < 0 {
		filled = 0
	}
	if filled > b.W-2 {
		filled = b.W - 2
	}

	if b.H >= 3 {
		c.Box(b, borderSpec(p.border), st)
		inner := geom.Rect{X: b.X + 1, Y: b.Y + 1, W: b.W - 2, H: b.H - 2}
		if filled > 0 {
			c.Fill(geom.Rect{X: inner.X, Y: inner.Y, W: filled, H: inner.H}, screen.Bar.Filled, st)
		}
		if rest := inner.W - filled; rest > 0 {
			c.Fill(geom.Rect{X: inner.X + filled, Y: inner.Y, W: rest, H: inner.H}, ' ', st)
		}
	} else {
		// 单行高退化为 ASCII 条（字形取自共享表）。
		track := b.W - 2
		c.Text(b.X, b.Y, string(screen.Bar.CapL)+
			screen.Bar.Line(screen.Bar.Pad, filled)+
			strings.Repeat(" ", track-filled)+
			string(screen.Bar.CapR), st)
	}

	if p.showPercent {
		label := p.format
		if label == "" {
			label = strconv.Itoa(int(p.ratio()*100+0.5)) + "%"
		}
		// 居中偏移按**列宽**算（不是字节数），否则中文百分号标签居中偏左。
		x := b.X + (b.W-text.Width(label))/2
		if x < b.X {
			x = b.X
		}
		c.Text(x, b.Y+b.H/2, text.Truncate(label, borderTitleBudget(b.W)), p.Style)
	}
}

// Handle 交互无：禁用吞事件，否则一律不消费（冒泡）。
func (p *ProgressBar) Handle(event.Event) bool { return !p.EffectivelyEnabled() }
