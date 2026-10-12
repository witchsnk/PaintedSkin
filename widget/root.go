package widget

import (
	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/style"
	"PaintedSkin/widget/layout"
)

// Root 是控件树的宿主容器：多子、z 序（后加入后绘、先命中）、
// 鼠标捕获、hover 合成、键盘路由、模态 Dialog 栈、单实例覆盖层。
//
// Root 实现 Host 接口，子控件经 SetHost 拿到它。Root 自身不做自动排布
// （Layout 只定自身边界，子控件由测试/上层显式 Layout）；M7 app 主循环
// 接入时由布局引擎统一排布。
type Root struct {
	Base
	kids      []Widget
	focus     *FocusManager
	capture   Widget
	hover     Widget
	dialogs   []Widget
	saved     []Widget
	overlay   Widget
	needPaint bool
}

// NewRoot 创建根容器（自举为 FocusManager 的域与子控件的 Host）。
func NewRoot() *Root {
	r := &Root{focus: NewFocusManager()}
	r.Visible = true
	r.Enabled = true
	r.focus.SetRoot(r)
	r.Base.SetHost(r)
	return r
}

// Add 追加子控件（后加入者 z 序居顶），并挂接父链、宿主与主题。
func (r *Root) Add(w Widget) {
	if w == nil {
		return
	}
	r.kids = append(r.kids, w)
	setParent(w, r)
	setHost(w, r)
	if r.th != nil {
		setTheme(w, r.th)
	}
}

// ApplyTheme 给整棵树换肤（含已挂载的 Dialog 与覆盖层）。
func (r *Root) ApplyTheme(t *Theme) {
	r.Base.SetTheme(t)
	for _, w := range r.kids {
		if w != nil {
			setTheme(w, t)
		}
	}
	if r.overlay != nil {
		setTheme(r.overlay, t)
	}
	for _, d := range r.dialogs {
		setTheme(d, t)
	}
}

// Children 返回子控件。
func (r *Root) Children() []Widget { return r.kids }

// SetHost 覆盖：Root 即宿主，忽略外部宿主但递归保证子树一致。
func (r *Root) SetHost(h Host) {
	r.Base.SetHost(r)
	for _, c := range r.kids {
		if c != nil {
			setHost(c, r)
		}
	}
	if r.overlay != nil {
		setHost(r.overlay, r)
	}
	for _, d := range r.dialogs {
		setHost(d, r)
	}
}

// Measure 返回当前边界尺寸钳制后结果。
func (r *Root) Measure(c layout.Constraints) geom.Size {
	return c.Clamp(r.Bounds.Size())
}

// Layout 设定根区域（不自动排布子控件）。
func (r *Root) Layout(b geom.Rect) { r.Bounds = b }

// Paint 按 z 序绘制：子 → 覆盖层 → Dialog 栈顶最后。
//
// 每个控件都被 Clip 到自己的 Bounds 再绘制：控件不必自己判断"有没有超出界"
// （Table/Tabs 的横向滚动曾经能画到最后一个可见列之外最多 cw 列）。
// Clip 是**栈式**的，嵌套控件的区域自然与父区域取交集。
func (r *Root) Paint(c screen.Canvas) {
	// Style 非零时铺满自身区域（零值 = 透明，与容器语义一致）。
	if r.Style != (style.Style{}) && !r.Bounds.Empty() {
		c.FillStyle(r.Bounds, r.Style)
	}
	for _, w := range r.kids {
		if w != nil {
			paintClipped(c, w)
		}
	}
	if r.overlay != nil {
		paintClipped(c, r.overlay)
	}
	for _, d := range r.dialogs {
		paintClipped(c, d)
	}
}

// paintClipped 把 w 裁剪到自己的 Bounds 后绘制（越界写入被丢弃）。
//
// 未暴露 BoundsOf 的自定义控件跳过裁剪（向后兼容），但框架内所有控件
// 都通过内嵌 widget.Base 拿到了它。
func paintClipped(c screen.Canvas, w Widget) {
	b, ok := w.(interface{ BoundsOf() geom.Rect })
	if !ok {
		w.Paint(c)
		return
	}
	c.Clip(b.BoundsOf())
	defer c.PopClip()
	w.Paint(c)
}

// HitTest z 序最顶层优先：覆盖层 > Dialog 栈顶 > 子控件。
func (r *Root) HitTest(p geom.Point) Widget {
	if !r.EffectivelyVisible() {
		return nil
	}
	if r.overlay != nil {
		if hit := r.overlay.HitTest(p); hit != nil {
			return hit
		}
	}
	for i := len(r.dialogs) - 1; i >= 0; i-- {
		if hit := r.dialogs[i].HitTest(p); hit != nil {
			return hit
		}
	}
	for i := len(r.kids) - 1; i >= 0; i-- {
		if hit := r.kids[i].HitTest(p); hit != nil {
			return hit
		}
	}
	return nil
}

// Focused 返回当前焦点控件。
func (r *Root) Focused() Widget { return r.focus.Focused() }

// NeedPaint 报告是否有重绘请求（测试用）。
func (r *Root) NeedPaint() bool { return r.needPaint }

// Dispatch 是事件总入口：命中测试 / 焦点路由 / 捕获 / 冒泡。
func (r *Root) Dispatch(e event.Event) bool {
	switch ev := e.(type) {
	case event.Mouse:
		return r.dispatchMouse(ev)
	case event.Key:
		return r.dispatchKey(ev)
	default:
		return false
	}
}

func (r *Root) dispatchMouse(m event.Mouse) bool {
	// 模态下：只允许落到栈顶 Dialog（含覆盖层）内的事件。
	if len(r.dialogs) > 0 {
		top := r.dialogs[len(r.dialogs)-1]
		var hit Widget
		if r.overlay != nil {
			hit = r.overlay.HitTest(geom.Point{X: m.X, Y: m.Y})
		}
		if hit == nil {
			hit = top.HitTest(geom.Point{X: m.X, Y: m.Y})
		}
		if hit == nil {
			return true // 栈外输入吞掉
		}
		return r.deliverMouse(hit, m)
	}
	// 捕获中：Move/Release 直达捕获者（按下期间不发 Leave）。
	if r.capture != nil && (m.Kind == event.MouseMove || m.Kind == event.MouseRelease) {
		hit := r.capture
		if m.Kind == event.MouseRelease {
			r.capture = nil
		}
		return r.bubble(hit, m)
	}
	hit := r.HitTest(geom.Point{X: m.X, Y: m.Y})
	switch m.Kind {
	case event.MousePress:
		// 覆盖层外点击 → 收起覆盖层并吞掉（M6 Dropdown 语义，机制在此）。
		if r.overlay != nil && hit != r.overlay && !within(hit, r.overlay) {
			r.SetOverlay(nil)
			return true
		}
		if hit == nil {
			return false // 点空白：焦点不变
		}
		r.clickFocus(hit)
		r.capture = hit
		return r.bubble(hit, m)
	case event.MouseMove:
		if r.capture != nil {
			return r.bubble(r.capture, m)
		}
		r.updateHover(hit)
		if hit == nil {
			return false
		}
		return r.bubble(hit, m)
	default: // Release（无捕获）/ Wheel / 其他
		if hit == nil {
			return false
		}
		return r.bubble(hit, m)
	}
}

// deliverMouse 模态域内的投递（含捕获与 hover 更新）。
func (r *Root) deliverMouse(hit Widget, m event.Mouse) bool {
	if r.capture != nil && (m.Kind == event.MouseMove || m.Kind == event.MouseRelease) {
		hit = r.capture
		if m.Kind == event.MouseRelease {
			r.capture = nil
		}
		return r.bubble(hit, m)
	}
	if m.Kind == event.MousePress {
		r.clickFocus(hit)
		r.capture = hit
		return r.bubble(hit, m)
	}
	if m.Kind == event.MouseMove && r.capture == nil {
		r.updateHover(hit)
	}
	return r.bubble(hit, m)
}

// updateHover 合成 Enter/Leave 并投递。
func (r *Root) updateHover(hit Widget) {
	if hit == r.hover {
		return
	}
	old := r.hover
	r.hover = hit
	if old != nil {
		old.Handle(event.Mouse{Kind: event.MouseLeave, Button: event.ButtonNone})
	}
	if hit != nil {
		hit.Handle(event.Mouse{Kind: event.MouseEnter, Button: event.ButtonNone})
	}
}

// clickFocus 可聚焦则夺取焦点（不可聚焦/禁用则焦点不变）。
func (r *Root) clickFocus(hit Widget) {
	if f, ok := hit.(interface{ IsFocusable() bool }); ok && f.IsFocusable() {
		r.focus.Focus(hit)
	}
}

func (r *Root) dispatchKey(k event.Key) bool {
	// 全局键：Tab / Shift+Tab 由 FocusManager 先拦截。
	if k.Code == event.KeyTab {
		if k.Mod&event.ModShift != 0 {
			r.focus.FocusPrev()
		} else {
			r.focus.FocusNext()
		}
		return true
	}
	if r.overlay != nil && r.overlay.Handle(k) {
		return true
	}
	f := r.focus.Focused()
	if len(r.dialogs) > 0 {
		top := r.dialogs[len(r.dialogs)-1]
		if f == nil || !within(f, top) {
			// 焦点不在栈顶内：首选陷阱内第一可聚焦者。
			if list := r.focus.Focusables(); len(list) > 0 {
				r.focus.Focus(list[0])
				f = list[0]
			} else {
				return true
			}
		}
		return r.bubbleUntil(f, top, k)
	}
	if f == nil {
		return false
	}
	return r.bubble(f, k)
}

// bubble 投递事件，未消费则沿父链冒泡到根。
func (r *Root) bubble(w Widget, e event.Event) bool {
	for cur := w; cur != nil; {
		if cur.Handle(e) {
			return true
		}
		cur = parentOf(cur)
	}
	return false
}

// bubbleUntil 冒泡上界止于 stop（含 stop，不外泄到模态外）。
func (r *Root) bubbleUntil(w, stop Widget, e event.Event) bool {
	for cur := w; cur != nil; {
		if cur.Handle(e) {
			return true
		}
		if cur == stop {
			return false
		}
		next := parentOf(cur)
		if next == nil || (!within(next, stop) && next != stop) {
			return false
		}
		cur = next
	}
	return false
}

func parentOf(w Widget) Widget {
	if p, ok := w.(interface{ ParentWidget() Widget }); ok {
		return p.ParentWidget()
	}
	return nil
}

// ---- Host 实现 ----

// Invalidate 打脏，请求重绘。
func (r *Root) Invalidate() { r.needPaint = true }

// Focus 主动聚焦（陷阱外目标在模态下被忽略）。
func (r *Root) Focus(w Widget) { r.focus.Focus(w) }

// PushDialog 模态入栈：屏蔽下层 + 焦点陷阱 + 焦点落入 Dialog 内。
func (r *Root) PushDialog(w Widget) {
	if w == nil {
		return
	}
	setParent(w, r)
	setHost(w, r)
	if r.th != nil {
		setTheme(w, r.th)
	}
	r.saved = append(r.saved, r.focus.Focused())
	r.dialogs = append(r.dialogs, w)
	r.focus.SetTrap(w)
	// 焦点默认落 Dialog 内第一个可聚焦者。
	if list := r.focus.Focusables(); len(list) > 0 {
		r.focus.Focus(list[0])
	} else {
		r.focus.Focus(nil)
	}
	r.Invalidate()
}

// PopDialog 模态出栈并恢复入栈前焦点。
func (r *Root) PopDialog() {
	if len(r.dialogs) == 0 {
		return
	}
	popped := r.dialogs[len(r.dialogs)-1]
	r.dialogs = r.dialogs[:len(r.dialogs)-1]
	if r.capture != nil && within(r.capture, popped) {
		r.capture = nil
	}
	if r.hover != nil && within(r.hover, popped) {
		r.hover = nil
	}
	var save Widget
	if len(r.saved) > 0 {
		save = r.saved[len(r.saved)-1]
		r.saved = r.saved[:len(r.saved)-1]
	}
	if len(r.dialogs) > 0 {
		top := r.dialogs[len(r.dialogs)-1]
		r.focus.SetTrap(top)
		if save != nil && within(save, top) {
			if f, ok := save.(interface{ IsFocusable() bool }); ok && f.IsFocusable() {
				r.focus.Focus(save)
				r.Invalidate()
				return
			}
		}
		if list := r.focus.Focusables(); len(list) > 0 {
			r.focus.Focus(list[0])
		} else {
			r.focus.Focus(nil)
		}
	} else {
		r.focus.ClearTrap()
		// 恢复入栈前焦点；若已不可聚焦或原无焦点则清空。
		if save != nil {
			if f, ok := save.(interface{ IsFocusable() bool }); ok && f.IsFocusable() {
				r.focus.Focus(save)
			} else {
				r.focus.Focus(nil)
			}
		} else {
			r.focus.Focus(nil)
		}
	}
	r.Invalidate()
}

// SetOverlay 设置单实例覆盖层（nil 收起）。
func (r *Root) SetOverlay(w Widget) {
	old := r.overlay
	if old != nil && r.hover != nil && within(r.hover, old) {
		r.hover = nil
	}
	if old != nil && r.capture != nil && within(r.capture, old) {
		r.capture = nil
	}
	r.overlay = w
	if old != nil && old != w {
		if c, ok := old.(interface{ Close() }); ok {
			c.Close()
		}
	}
	if w != nil {
		setParent(w, r)
		setHost(w, r)
		if r.th != nil {
			setTheme(w, r.th)
		}
	}
	r.Invalidate()
}

// Overlay 返回当前覆盖层（测试用）。
func (r *Root) Overlay() Widget { return r.overlay }

// Dialogs 返回当前模态对话框栈（测试用）。
func (r *Root) Dialogs() []Widget { return r.dialogs }

// Capture 显式请求鼠标捕获。
func (r *Root) Capture(w Widget) { r.capture = w }
