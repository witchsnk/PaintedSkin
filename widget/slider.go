package widget

import (
	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/style"
	"PaintedSkin/widget/layout"
)

// Slider 是滑块控件（M6 范围；本文件仅做新 Widget 接口的编译迁移，
// 行为与旧实现一致，不扩展）。
type Slider struct {
	Base
	min          int
	max          int
	value        int
	step         int
	horizontal   bool
	hoverTrack   bool
	pressedTrack bool
	onChange     func(value int)
}

// NewSlider 创建滑块。
func NewSlider() *Slider {
	s := &Slider{min: 0, max: 100, value: 50, step: 1}
	s.Base = NewBase()
	s.Base.Bind(s)
	s.Focusable = true
	return s
}

// SetRange 设置最小/最大值（钳制当前值）。
func (s *Slider) SetRange(min, max int) {
	s.min, s.max = min, max
	s.clamp()
}

// SetValue 设置滑块值并触发回调。
func (s *Slider) SetValue(v int) {
	s.value = v
	s.clamp()
	s.Invalidate()
	if s.onChange != nil {
		s.onChange(s.value)
	}
}

// Value 返回当前值。
func (s *Slider) Value() int { return s.value }

// SetStep 设置步进。
func (s *Slider) SetStep(step int) { s.step = step }

// Step 返回步进。
func (s *Slider) Step() int { return s.step }

// SetHorizontal 设置是否水平。
func (s *Slider) SetHorizontal(v bool) {
	s.horizontal = v
	s.Invalidate()
}

// Horizontal 报告是否水平。
func (s *Slider) Horizontal() bool { return s.horizontal }

// SetOnChange 设置变更回调。
func (s *Slider) SetOnChange(fn func(value int)) { s.onChange = fn }

// SetDisabled 开关禁用态；禁用时吞掉全部事件。
func (s *Slider) SetDisabled(v bool) {
	s.Enabled = !v
	s.Invalidate()
}

// Disabled 报告禁用态。
func (s *Slider) Disabled() bool { return !s.Enabled }

// OnBlur 覆盖：失焦结束拖拽。
func (s *Slider) OnBlur() {
	s.Base.OnBlur()
	s.pressedTrack = false
}

func (s *Slider) clamp() {
	if s.value < s.min {
		s.value = s.min
	}
	if s.value > s.max {
		s.value = s.max
	}
}

func (s *Slider) ratio() float64 {
	span := s.max - s.min
	if span <= 0 {
		return 0
	}
	r := float64(s.value-s.min) / float64(span)
	if r < 0 {
		return 0
	}
	if r > 1 {
		return 1
	}
	return r
}

// valueFromX 根据横坐标换算值（轨道点击/拖拽）。
func (s *Slider) valueFromX(x int) {
	w := s.Bounds.W
	if w <= 0 {
		return
	}
	rel := x - s.Bounds.X
	if rel < 0 {
		rel = 0
	}
	denom := w - 1
	if denom <= 0 {
		s.value = s.max
		s.clamp()
		return
	}
	if rel > denom {
		rel = denom
	}
	s.value = s.min + (s.max-s.min)*rel/denom
	s.clamp()
}

// valueFromY 根据纵坐标换算值。
func (s *Slider) valueFromY(y int) {
	h := s.Bounds.H
	if h <= 0 {
		return
	}
	rel := y - s.Bounds.Y
	if rel < 0 {
		rel = 0
	}
	denom := h - 1
	if denom <= 0 {
		s.value = s.max
		s.clamp()
		return
	}
	if rel > denom {
		rel = denom
	}
	s.value = s.min + (s.max-s.min)*rel/denom
	s.clamp()
}

func (s *Slider) dragTo(x, y int) {
	if s.horizontal {
		s.valueFromX(x)
	} else {
		s.valueFromY(y)
	}
	s.Invalidate()
	if s.onChange != nil {
		s.onChange(s.value)
	}
}

// Measure 水平 30×3，垂直 15×7。
func (s *Slider) Measure(lc layout.Constraints) geom.Size {
	if s.horizontal {
		return lc.Clamp(geom.Size{W: sliderPrefW, H: sliderPrefH})
	}
	return lc.Clamp(geom.Size{W: sliderPrefVW, H: sliderPrefVH})
}

// Layout 记录边界。
func (s *Slider) Layout(r geom.Rect) { s.Bounds = r }

// Paint 铺满自身区域后画轨道。
//
// 水平：单行 ASCII 条 `[===>   ]`（CapL + Pad* + KnobH + 空格 + CapR）。
// 垂直：逐行铺满 —— 手柄行之前是已填充块（Bar.Filled），之后是轨道
// （Bar.Track），手柄本身落在行中央（Bar.KnobV）。
// 两种方向都填满 Bounds（尾部空档纪律），手柄恒可见（100% 时也在轨道内）。
func (s *Slider) Paint(cnv screen.Canvas) {
	b := s.Bounds
	if b.Empty() {
		return
	}
	st := s.Style
	switch {
	case !s.EffectivelyEnabled():
		st = s.disabledStyle(st)
	case s.pressedTrack || s.Focused():
		st = s.focusStyle(st)
	case s.hoverTrack:
		st = s.hoverStyle(st)
	}
	paintBackground(cnv, b, st)
	if s.horizontal {
		s.paintHorizontal(cnv, b, st)
		return
	}
	s.paintVertical(cnv, b, st)
}

// paintHorizontal 画单行 ASCII 条。
func (s *Slider) paintHorizontal(cnv screen.Canvas, b geom.Rect, st style.Style) {
	track := b.W - 2
	if track < 1 {
		return
	}
	// 手柄恒在轨道内：满值时贴着右端而不是被挤出。
	knob := int(float64(track) * s.ratio())
	if knob > track-1 {
		knob = track - 1
	}
	if knob < 0 {
		knob = 0
	}
	var line []rune
	line = append(line, screen.Bar.CapL)
	for i := 0; i < track; i++ {
		switch {
		case i < knob:
			line = append(line, screen.Bar.Pad)
		case i == knob:
			line = append(line, screen.Bar.KnobH)
		default:
			line = append(line, ' ')
		}
	}
	line = append(line, screen.Bar.CapR)
	cnv.Text(b.X, b.Y, string(line), st)
}

// paintVertical 画整列块条：已填充块 → 手柄行 → 轨道。
// 与水平方向同构：手柄之前的格子是已填充，手柄格本身是手柄，之后是轨道。
func (s *Slider) paintVertical(cnv screen.Canvas, b geom.Rect, st style.Style) {
	knob := int(float64(b.H-1) * s.ratio())
	if knob > b.H-1 {
		knob = b.H - 1
	}
	if knob < 0 {
		knob = 0
	}
	center := b.X + (b.W-1)/2
	for y := 0; y < b.H; y++ {
		g := screen.Bar.Track
		if y < knob {
			g = screen.Bar.Filled
		}
		cnv.Text(b.X, b.Y+y, screen.Bar.Line(g, b.W), st)
		if y == knob {
			cnv.SetCell(center, b.Y+y, screen.Bar.KnobV, st)
		}
	}
}

// Handle 方向键步进、轨道点击跳转、拖拽；禁用全吞。
func (s *Slider) Handle(ev event.Event) bool {
	if !s.EffectivelyEnabled() {
		return true
	}
	switch e := ev.(type) {
	case event.Key:
		if e.Press && s.Focused() {
			switch e.Code {
			case event.KeyUp, event.KeyRight:
				s.SetValue(s.value + s.step)
			case event.KeyDown, event.KeyLeft:
				s.SetValue(s.value - s.step)
			case event.KeyHome:
				s.SetValue(s.min)
			case event.KeyEnd:
				s.SetValue(s.max)
			default:
				return false
			}
			return true
		}
	case event.Mouse:
		inside := s.Bounds.Contains(geom.Point{X: e.X, Y: e.Y})
		switch e.Kind {
		case event.MouseEnter:
			s.hoverTrack = true
			s.Invalidate()
			return true
		case event.MouseLeave:
			s.hoverTrack = false
			s.Invalidate()
			return true
		case event.MousePress:
			if e.Button == event.ButtonLeft && inside {
				s.pressedTrack = true
				s.dragTo(e.X, e.Y)
				if h := s.HostOf(); h != nil {
					h.Capture(s)
				}
				return true
			}
		case event.MouseRelease:
			if s.pressedTrack {
				s.pressedTrack = false
				s.dragTo(e.X, e.Y)
				return true
			}
		case event.MouseMove:
			if s.pressedTrack {
				s.dragTo(e.X, e.Y)
			}
			s.hoverTrack = inside
			return true
		}
	}
	return false
}
