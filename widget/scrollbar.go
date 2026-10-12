package widget

import (
	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/style"
	"PaintedSkin/widget/layout"
)

// Scrollbar 是滚动条（docs/08）：Offset/Total/View + OnChange；
// 轨道点击跳页、拖拽滑块、滚轮转发给宿主；贴容器右/下缘，宽度 1 格。
type Scrollbar struct {
	Base
	offset, total, view int
	horizontal          bool
	onChange            func(offset int)
	dragging            bool
}

// NewScrollbar 创建垂直滚动条（贴右缘、宽 1 格）。
func NewScrollbar() *Scrollbar {
	s := &Scrollbar{}
	s.Base = NewBase()
	s.Base.Bind(s)
	return s
}

// SetTotal 设置内容总长度（<0 视为 0）。
func (s *Scrollbar) SetTotal(v int) {
	if v < 0 {
		v = 0
	}
	s.total = v
	s.SetOffset(s.offset)
}

// Total 返回内容总长度。
func (s *Scrollbar) Total() int { return s.total }

// SetView 设置视口长度（<0 视为 0）。
func (s *Scrollbar) SetView(v int) {
	if v < 0 {
		v = 0
	}
	s.view = v
	s.SetOffset(s.offset)
}

// View 返回视口长度。
func (s *Scrollbar) View() int { return s.view }

// SetOffset 设置偏移（钳制在 [0, Total-View]）；不触发 OnChange。
func (s *Scrollbar) SetOffset(v int) {
	s.offset = layout.ClampScroll(v, s.total, s.view)
	s.Invalidate()
}

// Offset 返回当前偏移。
func (s *Scrollbar) Offset() int { return s.offset }

// SetHorizontal 设置方向（true=水平）。
func (s *Scrollbar) SetHorizontal(v bool) {
	s.horizontal = v
	s.Invalidate()
}

// Horizontal 报告是否水平。
func (s *Scrollbar) Horizontal() bool { return s.horizontal }

// SetOnChange 设置交互触发偏移变化的回调（setter 不触发）。
func (s *Scrollbar) SetOnChange(fn func(offset int)) { s.onChange = fn }

// SetDisabled 开关禁用态；禁用时吞掉全部事件。
func (s *Scrollbar) SetDisabled(v bool) {
	s.Enabled = !v
	s.dragging = false
	s.Invalidate()
}

// Disabled 报告禁用态。
func (s *Scrollbar) Disabled() bool { return !s.Enabled }

func (s *Scrollbar) trackLen() int {
	if s.horizontal {
		return s.Bounds.W
	}
	return s.Bounds.H
}

func (s *Scrollbar) posAt(coord int) int {
	if s.horizontal {
		return coord - s.Bounds.X
	}
	return coord - s.Bounds.Y
}

// thumbLen 返回滑块长度（至少 1 格）。
func (s *Scrollbar) thumbLen() int {
	track := s.trackLen()
	if track <= 0 || s.total <= 0 || s.total <= s.view {
		return track
	}
	n := track * s.view / s.total
	if n < 1 {
		n = 1
	}
	return n
}

// thumbPos 返回滑块起点（轨道内偏移）。
func (s *Scrollbar) thumbPos() int {
	track := s.trackLen()
	span := s.total - s.view
	if span <= 0 || track <= s.thumbLen() {
		return 0
	}
	p := s.offset * (track - s.thumbLen()) / span
	if p < 0 {
		p = 0
	}
	return p
}

func (s *Scrollbar) onThumb(pos int) bool {
	tp := s.thumbPos()
	return pos >= tp && pos < tp+s.thumbLen()
}

// applyOffset 钳制并触发回调（仅变化时）。
func (s *Scrollbar) applyOffset(v int) {
	n := layout.ClampScroll(v, s.total, s.view)
	if n == s.offset {
		return
	}
	s.offset = n
	s.Invalidate()
	if s.onChange != nil {
		s.onChange(n)
	}
}

// Measure 垂直 W=1、H=MaxH；水平 W=MaxW、H=1（"宽度 1 格"那一维恒 1）。
func (s *Scrollbar) Measure(c layout.Constraints) geom.Size {
	if s.horizontal {
		return geom.Size{W: clampDim(c.MaxW, c.MinW), H: 1}
	}
	return geom.Size{W: 1, H: clampDim(c.MaxH, c.MinH)}
}

// Layout 记录边界。
func (s *Scrollbar) Layout(r geom.Rect) { s.Bounds = r }

// Paint 画轨道 '░' 与滑块 '█'（禁用时暗淡）。
func (s *Scrollbar) Paint(c screen.Canvas) {
	b := s.Bounds
	if b.Empty() {
		return
	}
	th := s.themeOrZero()
	thumbStyle := th.ScrollbarThumb
	if thumbStyle == (style.Style{}) && th.ScrollbarThumbColor.Kind != style.ColorDefault {
		// 更细的 ScrollbarThumb 未设时，回落到基础滑块色。
		thumbStyle = style.Style{}.WithForeground(th.ScrollbarThumbColor)
	}
	thumbSt := applyAll(s.Style, thumbStyle, nil)
	trackSt := applyAll(s.Style, th.ScrollbarTrack, nil)
	thumbSt = s.disabledStyle(thumbSt)
	trackSt = s.disabledStyle(trackSt)
	paintBackground(c, b, trackSt)
	tp := s.thumbPos()
	tl := s.thumbLen()
	n := b.W
	if !s.horizontal {
		n = b.H
	}
	for i := 0; i < n; i++ {
		g, st := screen.Bar.Track, trackSt
		if i >= tp && i < tp+tl {
			g, st = screen.Bar.Filled, thumbSt
		}
		if s.horizontal {
			c.Text(b.X+i, b.Y, string(g), st)
		} else {
			c.Text(b.X, b.Y+i, string(g), st)
		}
	}
}

// Handle 轨道点击跳页、拖拽滑块、滚轮转发（不消费）。
func (s *Scrollbar) Handle(e event.Event) bool {
	if !s.EffectivelyEnabled() {
		return true
	}
	switch ev := e.(type) {
	case event.Mouse:
		switch ev.Kind {
		case event.MouseWheelUp, event.MouseWheelDown:
			return false // 转发给宿主
		case event.MousePress:
			if ev.Button != event.ButtonLeft {
				return false
			}
			pos := s.posAt(ev.Y)
			if s.horizontal {
				pos = s.posAt(ev.X)
			}
			if pos < 0 || pos >= s.trackLen() {
				return false
			}
			if s.onThumb(pos) {
				s.dragging = true
				s.Invalidate()
				if h := s.HostOf(); h != nil {
					h.Capture(s)
				}
				return true
			}
			// 轨道点击：滑块在轨道上方则下翻，否则上翻。
			if pos < s.thumbPos() {
				s.applyOffset(s.offset - s.view)
			} else {
				s.applyOffset(s.offset + s.view)
			}
			return true
		case event.MouseMove:
			if !s.dragging {
				return false
			}
			pos := s.posAt(ev.Y)
			if s.horizontal {
				pos = s.posAt(ev.X)
			}
			s.dragTo(pos)
			return true
		case event.MouseRelease:
			if s.dragging {
				s.dragging = false
				s.Invalidate()
				return true
			}
		}
	}
	return false
}

// dragTo 按滑块中心对齐换算偏移。
func (s *Scrollbar) dragTo(pos int) {
	track := s.trackLen()
	span := s.total - s.view
	if span <= 0 {
		return
	}
	free := track - s.thumbLen()
	if free <= 0 {
		return
	}
	center := pos - s.thumbLen()/2
	if center < 0 {
		center = 0
	}
	if center > free {
		center = free
	}
	s.applyOffset(center * span / free)
}
