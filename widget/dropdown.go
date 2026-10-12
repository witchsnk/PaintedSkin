package widget

import (
	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/text"
	"PaintedSkin/widget/layout"
)

// dropdownMaxVisible 是弹层最多同时显示的选项数。
const dropdownMaxVisible = 6

// Dropdown 是下拉选择器（docs/08）：关闭态显示当前项；点击经 Host.SetOverlay
// 展开弹层（下方优先、空间不足翻转到上方）；点选项选择并收起；Esc/点外部收起；
// 展开时上下键选择、Enter 确认。
type Dropdown struct {
	Base
	items    []string
	selected int
	open     bool
	layer    *dropdownList
	onChange func(index int)
}

// NewDropdown 创建下拉选择器。
func NewDropdown() *Dropdown {
	d := &Dropdown{selected: -1}
	d.Base = NewBase()
	d.Bind(d)
	d.Focusable = true
	return d
}

// SetItems 设置选项（选中重置到首项）。
func (d *Dropdown) SetItems(items []string) {
	d.items = append([]string(nil), items...)
	d.selected = -1
	if len(d.items) > 0 {
		d.selected = 0
	}
	d.Invalidate()
}

// Items 返回选项。
func (d *Dropdown) Items() []string { return d.items }

// SetOnChange 设置选择变更回调。
func (d *Dropdown) SetOnChange(fn func(index int)) { d.onChange = fn }

// SetDisabled 开关禁用态；禁用时吞掉全部事件。
func (d *Dropdown) SetDisabled(v bool) {
	d.Enabled = !v
	d.Invalidate()
}

// Disabled 报告禁用态。
func (d *Dropdown) Disabled() bool { return !d.Enabled }

// IsOpen 报告是否展开。
func (d *Dropdown) IsOpen() bool { return d.open }

// Selected 返回选中索引（-1 表示无）。
func (d *Dropdown) Selected() int { return d.selected }

// SetSelected 设置选中索引（越界钳制）并回调。
func (d *Dropdown) SetSelected(i int) {
	n := i
	if n >= len(d.items) {
		n = len(d.items) - 1
	}
	if n < -1 {
		n = -1
	}
	if n == d.selected {
		return
	}
	d.selected = n
	d.Invalidate()
	if d.onChange != nil {
		d.onChange(n)
	}
}

// current 返回当前项文本。
func (d *Dropdown) current() string {
	if d.selected < 0 || d.selected >= len(d.items) {
		return ""
	}
	return d.items[d.selected]
}

// Measure 关闭态 = 当前项宽 + " ▼"。
func (d *Dropdown) Measure(c layout.Constraints) geom.Size {
	w := text.Width(d.current()) + 2
	if w < 4 {
		w = 4
	}
	return c.Clamp(geom.Size{W: w, H: 1})
}

// Layout 记录边界。
func (d *Dropdown) Layout(r geom.Rect) { d.Bounds = r }

// HitTest 关闭态仅自身；展开态交给弹层（弹层由宿主命中）。
func (d *Dropdown) HitTest(p geom.Point) Widget {
	if !d.EffectivelyVisible() || !d.Bounds.Contains(p) {
		return nil
	}
	return d
}

// Handle 切换展开、选择、键盘导航；禁用全吞。
func (d *Dropdown) Handle(e event.Event) bool {
	if !d.EffectivelyEnabled() {
		return true
	}
	switch ev := e.(type) {
	case event.Mouse:
		if ev.Kind != event.MousePress || ev.Button != event.ButtonLeft {
			return false
		}
		inside := d.Bounds.Contains(geom.Point{X: ev.X, Y: ev.Y})
		if d.open {
			// 点击自身区域视为收起；弹层内的点击由弹层自行处理。
			if inside {
				d.Close()
				return true
			}
			return true
		}
		if inside && len(d.items) > 0 {
			d.Open()
			return true
		}
		return false
	case event.Key:
		if !ev.Press {
			return false
		}
		switch ev.Code {
		case event.KeyEnter, event.KeySpace:
			if d.open {
				d.confirm()
				return true
			}
			if d.Focused() && len(d.items) > 0 {
				d.Open()
				return true
			}
			return false
		case event.KeyEsc:
			if d.open {
				d.Close()
				return true
			}
			return false
		case event.KeyUp, event.KeyDown:
			if d.open {
				d.move(ev.Code == event.KeyDown)
				return true
			}
			if d.Focused() && len(d.items) > 0 {
				d.Open()
				return true
			}
			return false
		}
	}
	return false
}

// Open 展开弹层（经宿主 SetOverlay）。
func (d *Dropdown) Open() {
	if d.open || len(d.items) == 0 {
		return
	}
	d.open = true
	h := d.HostOf()
	if h == nil {
		d.Invalidate()
		return // 无宿主：仅标记展开（测试/嵌入场景）
	}
	w := d.Bounds.W
	if w <= 0 {
		w = 4
	}
	hgt := len(d.items)
	if hgt > dropdownMaxVisible {
		hgt = dropdownMaxVisible
	}
	rect := d.layerRect(w, hgt+2) // +2 = 上下边框
	l := newDropdownList(d.items, d.selected, d)
	l.Layout(rect)
	h.SetOverlay(l)
	d.layer = l
	d.Invalidate()
}

// Close 收起弹层。
func (d *Dropdown) Close() {
	if !d.open {
		return
	}
	d.open = false
	d.layer = nil
	if h := d.HostOf(); h != nil {
		h.SetOverlay(nil)
	}
	d.Invalidate()
}

// layerRect 计算弹层位置：下方优先，空间不足翻转到上方。
func (d *Dropdown) layerRect(w, h int) geom.Rect {
	b := d.Bounds
	// 取宿主（Root）区域作为屏幕参考；拿不到则假设下方空间充足。
	top := b.Y + 1
	bottomSpace := d.screenBottom() - top
	if bottomSpace >= h {
		return geom.Rect{X: b.X, Y: top, W: w, H: h}
	}
	y := b.Y - h
	if y < 0 {
		y = 0
	}
	return geom.Rect{X: b.X, Y: y, W: w, H: h}
}

// screenBottom 返回屏幕底行（Root 边界；无宿主时按自身估算）。
func (d *Dropdown) screenBottom() int {
	if r, ok := d.HostOf().(*Root); ok && !r.Bounds.Empty() {
		return r.Bounds.Bottom()
	}
	return d.Bounds.Bottom()
}

// move 在展开态上下预选。
func (d *Dropdown) move(down bool) {
	if len(d.items) == 0 {
		return
	}
	n := d.selected
	if n < 0 {
		n = 0
	} else if down {
		if n+1 < len(d.items) {
			n++
		}
	} else if n > 0 {
		n--
	}
	d.selected = n
	if d.layer != nil {
		d.layer.hover = n
		d.layer.Invalidate()
	}
	d.Invalidate()
}

// confirm 确认当前预选并收起。
func (d *Dropdown) confirm() {
	d.Close()
	if d.onChange != nil && d.selected >= 0 {
		d.onChange(d.selected)
	}
}

// Pick 由弹层回调：选择并收起。
func (d *Dropdown) Pick(i int) {
	if i < 0 || i >= len(d.items) {
		return
	}
	d.selected = i
	d.Close()
	if d.onChange != nil {
		d.onChange(i)
	}
	d.Invalidate()
}

// Paint 铺底，再把"当前项 + ▼"补齐到 Measure 报出的宽度
// （空选时标签只有 2 列，此前与 Measure 差 2 列形成确定性缺口）。
func (d *Dropdown) Paint(c screen.Canvas) {
	b := d.Bounds
	if b.Empty() {
		return
	}
	label := d.current() + " ▼"
	st := d.Style
	switch {
	case !d.EffectivelyEnabled():
		st = d.disabledStyle(st)
	case d.Focused():
		st = d.focusStyle(st)
	}
	paintBackground(c, b, st)
	c.Text(b.X, b.Y, text.Fit(label, b.W, text.Left), st)
}

// ---- 弹层 ----

// dropdownList 是下拉弹层：作为单实例覆盖层绘制在宿主之上。
type dropdownList struct {
	Base
	items  []string
	hover  int
	owner  *Dropdown
	border BorderStyle
}

// newDropdownList 创建弹层。
func newDropdownList(items []string, selected int, owner *Dropdown) *dropdownList {
	l := &dropdownList{items: items, hover: selected, owner: owner, border: BorderSingle}
	l.Base = NewBase()
	l.Bind(l)
	return l
}

// Close 响应宿主 overlay 移除，收起自身并通知所有者。
func (l *dropdownList) Close() {
	if l.owner != nil && l.owner.open {
		l.owner.Close()
	}
}

// BoundsOf 返回弹层边界（测试与调试用）。
func (l *dropdownList) BoundsOf() geom.Rect { return l.Bounds }

// Items 返回选项。
func (l *dropdownList) Items() []string { return l.items }

// Hover 返回预选索引。
func (l *dropdownList) Hover() int { return l.hover }

// Measure 高 = min(项数, dropdownMaxVisible)。
func (l *dropdownList) Measure(c layout.Constraints) geom.Size {
	h := len(l.items)
	if h > dropdownMaxVisible {
		h = dropdownMaxVisible
	}
	w := 0
	for _, it := range l.items {
		if sw := text.Width(it); sw > w {
			w = sw
		}
	}
	// 宽度 = 最长项 + 左右边框各 1 格；高度 = 选项数 + 上下边框各 1 格。
	return c.Clamp(geom.Size{W: w + 2, H: h + 2})
}

// Layout 记录边界。
func (l *dropdownList) Layout(r geom.Rect) { l.Bounds = r }

// Paint 画弹层：铺底 → 边框 → 选项（选项只落在框内）。
func (l *dropdownList) Paint(c screen.Canvas) {
	b := l.Bounds
	if b.Empty() {
		return
	}
	st := l.Style
	paintBackground(c, b, st)
	spec := l.borderSpec()
	if spec.Visible() {
		c.Box(b, spec, l.borderColor())
	}
	inner := b.Inflate(geom.SymInsets(1)).Intersect(b)
	if inner.Empty() {
		return
	}
	for i := 0; i < inner.H && i < len(l.items); i++ {
		ist := st
		if i == l.hover {
			ist = l.selectedStyle(st)
		}
		c.Text(inner.X, inner.Y+i, text.Fit(l.items[i], inner.W, text.Left), ist)
	}
}

// borderSpec 返回弹层边框（可用 SetBorder 换档位）。
func (l *dropdownList) borderSpec() screen.Border { return borderSpec(l.border) }

// SetBorder 设置弹层边框样式。
func (l *dropdownList) SetBorder(s BorderStyle) {
	l.border = s
	l.Invalidate()
}

// Border 返回弹层边框样式。
func (l *dropdownList) Border() BorderStyle { return l.border }

// HitTest 落在弹层内即命中自身。
func (l *dropdownList) HitTest(p geom.Point) Widget {
	if !l.Bounds.Contains(p) {
		return nil
	}
	return l
}

// Handle 点击选项选择；键盘上下预选、Enter 确认、Esc 取消。
func (l *dropdownList) Handle(e event.Event) bool {
	if l.owner == nil {
		return false
	}
	switch ev := e.(type) {
	case event.Mouse:
		if ev.Kind != event.MousePress {
			return false
		}
		if !l.Bounds.Contains(geom.Point{X: ev.X, Y: ev.Y}) {
			return false
		}
		idx := ev.Y - l.Bounds.Y - 1 // 顶边框占 1 行
		if idx < 0 || idx >= len(l.items) {
			return true
		}
		l.owner.Pick(idx)
		return true
	case event.Key:
		if !ev.Press {
			return false
		}
		switch ev.Code {
		case event.KeyDown:
			if l.hover+1 < len(l.items) {
				l.hover++
				l.owner.selected = l.hover
				l.Invalidate()
			}
			return true
		case event.KeyUp:
			if l.hover > 0 {
				l.hover--
				l.owner.selected = l.hover
				l.Invalidate()
			}
			return true
		case event.KeyEnter:
			l.owner.confirm()
			return true
		case event.KeyEsc:
			l.owner.Close()
			return true
		}
	}
	return false
}
