package widget

import (
	"strings"

	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/text"
	"PaintedSkin/widget/layout"
)

// Tabs 是标签页容器（docs/08）：标签条 + 内容区；点击标签或左右键切页；
// 标签条视为一组（占一个 Tab 位），Tab 键进入内容区。
type Tabs struct {
	Base
	labels   []string
	current  int
	content  Widget
	onChange func(index int)
}

// NewTabs 创建标签页容器。
func NewTabs() *Tabs {
	t := &Tabs{}
	t.Base = NewBase()
	t.Bind(t)
	t.Focusable = true
	return t
}

// SetTabs 设置标签（选中重置到首项）。
func (t *Tabs) SetTabs(labels []string) {
	t.labels = append([]string(nil), labels...)
	t.current = 0
	t.Invalidate()
}

// Labels 返回标签。
func (t *Tabs) Labels() []string { return t.labels }

// SetContent 设置内容区控件。
func (t *Tabs) SetContent(w Widget) {
	t.content = w
	if w != nil {
		setParent(w, t)
		if h := t.HostOf(); h != nil {
			setHost(w, h)
		}
	}
	t.Invalidate()
}

// Content 返回内容区控件。
func (t *Tabs) Content() Widget { return t.content }

// SetOnChange 设置切页回调。
func (t *Tabs) SetOnChange(fn func(index int)) { t.onChange = fn }

// SetDisabled 开关禁用态；禁用时吞掉全部事件。
func (t *Tabs) SetDisabled(v bool) {
	t.Enabled = !v
	t.Invalidate()
}

// Disabled 报告禁用态。
func (t *Tabs) Disabled() bool { return !t.Enabled }

// Current 返回当前页索引。
func (t *Tabs) Current() int { return t.current }

// SetCurrent 设置当前页（越界钳制）并回调。
func (t *Tabs) SetCurrent(i int) {
	if len(t.labels) == 0 {
		return
	}
	if i < 0 {
		i = 0
	}
	if i >= len(t.labels) {
		i = len(t.labels) - 1
	}
	if i == t.current {
		return
	}
	t.current = i
	t.Invalidate()
	if t.onChange != nil {
		t.onChange(i)
	}
}

// Children 返回内容区（供焦点遍历）。
func (t *Tabs) Children() []Widget {
	if t.content == nil {
		return nil
	}
	return []Widget{t.content}
}

// SetHost 递归挂接宿主。
func (t *Tabs) SetHost(h Host) {
	t.Base.SetHost(h)
	if t.content != nil {
		setHost(t.content, h)
	}
}

// tabWidth 返回第 i 个标签的占宽（两侧各 1 格空格）。
func (t *Tabs) tabWidth(i int) int {
	if i < 0 || i >= len(t.labels) {
		return 0
	}
	return text.Width(t.labels[i]) + 2*tabPadW
}

// Measure 高 = 1（标签行）+ 内容高；宽取标签行与内容较大者。
func (t *Tabs) Measure(c layout.Constraints) geom.Size {
	if len(t.labels) == 0 && t.content == nil {
		return c.Clamp(geom.Size{W: 0, H: 0})
	}
	tabW := 0
	for i := range t.labels {
		tabW += t.tabWidth(i)
	}
	w, h := tabW, 1
	if t.content != nil {
		cs := t.content.Measure(c)
		if cs.W > w {
			w = cs.W
		}
		h += cs.H
	}
	return c.Clamp(geom.Size{W: w, H: h})
}

// Layout 标签行高 1，内容区为其下全部区域。
func (t *Tabs) Layout(r geom.Rect) {
	t.Bounds = r
	if t.content != nil {
		t.content.Layout(t.contentRect())
	}
}

// contentRect 返回内容区。
func (t *Tabs) contentRect() geom.Rect {
	return geom.Rect{X: t.Bounds.X, Y: t.Bounds.Y + 1, W: t.Bounds.W, H: t.Bounds.H - 1}
}

// HitTest 标签行命中自身；内容区转给内容控件。
func (t *Tabs) HitTest(p geom.Point) Widget {
	if !t.EffectivelyVisible() || !t.Bounds.Contains(p) {
		return nil
	}
	if p.Y > t.Bounds.Y {
		if t.content != nil {
			if hit := t.content.HitTest(p); hit != nil {
				return hit
			}
		}
	}
	return t
}

// Handle 点击标签切页、左右键切页；禁用全吞。
func (t *Tabs) Handle(e event.Event) bool {
	if !t.EffectivelyEnabled() {
		return true
	}
	switch ev := e.(type) {
	case event.Mouse:
		if ev.Kind == event.MousePress {
			if ev.Button != event.ButtonLeft {
				return false
			}
			if !t.Bounds.Contains(geom.Point{X: ev.X, Y: ev.Y}) {
				return false
			}
			if ev.Y > t.Bounds.Y {
				return false // 内容区不处理
			}
			if idx := t.indexAt(ev.X); idx >= 0 {
				t.SetCurrent(idx)
				return true
			}
			return false
		}
	case event.Key:
		if ev.Press && t.Focused() {
			switch ev.Code {
			case event.KeyRight:
				t.SetCurrent(t.current + 1)
				return true
			case event.KeyLeft:
				t.SetCurrent(t.current - 1)
				return true
			}
		}
	}
	return false
}

// indexAt 返回 x 命中的标签索引（-1 表示未命中）。
func (t *Tabs) indexAt(x int) int {
	rel := x - t.Bounds.X
	if rel < 0 {
		return -1
	}
	pos := 0
	for i := range t.labels {
		w := t.tabWidth(i)
		if rel >= pos && rel < pos+w {
			return i
		}
		pos += w
	}
	return -1
}

// Paint 画标签行（当前页反色）+ 内容区。
func (t *Tabs) Paint(c screen.Canvas) {
	b := t.Bounds
	if b.Empty() {
		return
	}
	st := t.Style
	if !t.EffectivelyEnabled() {
		st = t.disabledStyle(st)
	}
	paintBackground(c, b, st)
	x := b.X
	for i, label := range t.labels {
		w := t.tabWidth(i)
		if x >= b.Right() {
			break
		}
		seg := " " + text.Truncate(label, w-2*tabPadW) + " "
		segSt := st
		if i == t.current {
			segSt = t.selectedStyle(st)
		} else if t.Focused() {
			segSt = t.focusStyle(st)
		}
		c.Text(x, b.Y, seg, segSt)
		x += w
	}
	if t.content != nil {
		t.content.Paint(c)
	}
}

// tabRowString 便于测试断言标签行文本。
func (t *Tabs) tabRowString() string {
	var sb strings.Builder
	for i := range t.labels {
		sb.WriteString(" ")
		sb.WriteString(t.labels[i])
		sb.WriteString(" ")
	}
	return sb.String()
}
