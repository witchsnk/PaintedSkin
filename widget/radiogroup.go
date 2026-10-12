package widget

import (
	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/text"
	"PaintedSkin/widget/layout"
)

// RadioGroup 是单选组（docs/08）：组内互斥、方向键移动选择、Tab 在组间移动
// （组占一个 Tab 位，成员不逐个占位）。
type RadioGroup struct {
	Base
	name     string
	items    []string
	radios   []*Radio
	selected int
	onChange func(index int)
}

// NewRadioGroup 创建单选组（selected 初始 -1 表示未选）。
func NewRadioGroup(name string, items []string) *RadioGroup {
	g := &RadioGroup{name: name, selected: -1}
	g.Base = NewBase()
	g.Focusable = true
	g.Bind(g)
	g.SetItems(items)
	return g
}

// bindMember 让成员在自行选中时回调宿主组，保证互斥（docs/08）。
func (g *RadioGroup) bindMember(r *Radio) {
	r.SetOnChange(func(checked bool) {
		if !checked {
			return
		}
		for i, rad := range g.radios {
			if rad == r {
				g.selected = i
				break
			}
		}
		for i, rad := range g.radios {
			rad.SetChecked(i == g.selected)
		}
		g.Invalidate()
		if g.onChange != nil {
			g.onChange(g.selected)
		}
	})
}

// SetItems 重建成员（保持已选项索引有效）。
func (g *RadioGroup) SetItems(items []string) {
	g.items = append([]string(nil), items...)
	g.radios = make([]*Radio, len(g.items))
	for i, label := range g.items {
		r := NewRadio(label)
		r.SetGroup(g.name)
		r.SetParent(g)
		g.bindMember(r)
		g.radios[i] = r
	}
	g.selected = clampSelected(g.selected, len(g.items))
	if g.selected >= 0 {
		g.applySelection(false)
	}
}

// Name 返回组名。
func (g *RadioGroup) Name() string { return g.name }

// Items 返回选项文本。
func (g *RadioGroup) Items() []string { return g.items }

// Radio 返回第 i 个成员（越界返回 nil）。
func (g *RadioGroup) Radio(i int) *Radio {
	if i < 0 || i >= len(g.radios) {
		return nil
	}
	return g.radios[i]
}

// Radios 返回全部成员。
func (g *RadioGroup) Radios() []*Radio { return g.radios }

// Selected 返回选中索引（-1 表示未选）。
func (g *RadioGroup) Selected() int { return g.selected }

// SetSelected 设置选中索引（越界钳制到合法范围，-1 允许）并回调。
// 同值仍触发回调（用于组外代码显式设置选中项）。
func (g *RadioGroup) SetSelected(i int) {
	n := clampSelected(i, len(g.items))
	g.selected = n
	g.applySelection(true)
}

// SetOnChange 设置选中变更回调（参数为选中索引）。
func (g *RadioGroup) SetOnChange(fn func(index int)) { g.onChange = fn }

// SetDisabled 开关禁用态（整组）。
func (g *RadioGroup) SetDisabled(v bool) {
	g.Enabled = !v
	g.Invalidate()
}

// Disabled 报告禁用态。
func (g *RadioGroup) Disabled() bool { return !g.Enabled }

// clampIndex 把索引钳到 [0,n-1]；n==0 时为 -1。
func clampIndex(i, n int) int {
	if n == 0 {
		return -1
	}
	if i < 0 {
		return 0
	}
	if i >= n {
		return n - 1
	}
	return i
}

// clampSelected 保留"未选"（-1）语义：越界向上钳，负值保持 -1。
func clampSelected(i, n int) int {
	if n == 0 || i < 0 {
		return -1
	}
	if i >= n {
		return n - 1
	}
	return i
}

// applySelection 同步成员勾选态；fire 为 true 时触发 OnChange。
func (g *RadioGroup) applySelection(fire bool) {
	for i, r := range g.radios {
		r.SetChecked(i == g.selected)
	}
	g.Invalidate()
	if fire && g.onChange != nil {
		g.onChange(g.selected)
	}
}

// Children 返回成员（供焦点遍历；成员 Focusable=false，故组本身占唯一 Tab 位）。
func (g *RadioGroup) Children() []Widget {
	out := make([]Widget, len(g.radios))
	for i, r := range g.radios {
		out[i] = r
	}
	return out
}

// SetHost 递归挂接宿主。
func (g *RadioGroup) SetHost(h Host) {
	g.Base.SetHost(h)
	for _, r := range g.radios {
		r.SetHost(h)
	}
}

// Measure 高 = 成员数；宽 = 最长成员首选宽。
func (g *RadioGroup) Measure(c layout.Constraints) geom.Size {
	if len(g.items) == 0 {
		return c.Clamp(geom.Size{W: 0, H: 0})
	}
	w := 0
	for _, label := range g.items {
		if sw := text.Width(label) + 4; sw > w {
			w = sw
		}
	}
	return c.Clamp(geom.Size{W: w, H: len(g.items)})
}

// Layout 每行一个成员（H=1）。
func (g *RadioGroup) Layout(r geom.Rect) {
	g.Bounds = r
	for i, rad := range g.radios {
		row := geom.Rect{X: r.X, Y: r.Y + i, W: r.W, H: 1}
		rad.Layout(row.Intersect(r))
	}
}

// Paint 绘制各成员。
func (g *RadioGroup) Paint(c screen.Canvas) {
	for _, rad := range g.radios {
		rad.Paint(c)
	}
}

// HitTest 命中成员（z 序顶优先）；成员未命中回落组自身。
func (g *RadioGroup) HitTest(p geom.Point) Widget {
	if !g.EffectivelyVisible() || !g.Bounds.Contains(p) {
		return nil
	}
	for i := len(g.radios) - 1; i >= 0; i-- {
		if hit := g.radios[i].HitTest(p); hit != nil {
			return hit
		}
	}
	return g
}

// Handle 组内方向键移动选择；其余转发成员（点击选中）。
func (g *RadioGroup) Handle(e event.Event) bool {
	if !g.EffectivelyEnabled() {
		return true
	}
	if k, ok := e.(event.Key); ok && k.Press && g.Focused() {
		switch k.Code {
		case event.KeyDown, event.KeyRight:
			g.move(1)
			return true
		case event.KeyUp, event.KeyLeft:
			g.move(-1)
			return true
		}
	}
	// 未命中成员的事件不消费，允许冒泡。
	return false
}

// move 以步长移动选择（在 [0,n-1] 内钳制）；无实际变化时不触发回调。
func (g *RadioGroup) move(d int) {
	if len(g.items) == 0 {
		return
	}
	cur := g.selected
	if cur < 0 {
		// 未选时：向下从首项起，向上从末项起。
		if d < 0 {
			cur = len(g.items) - 1
		} else {
			cur = 0
		}
	} else {
		cur = clampIndex(cur+d, len(g.items))
	}
	if cur == g.selected {
		return
	}
	g.selected = cur
	g.applySelection(true)
}
