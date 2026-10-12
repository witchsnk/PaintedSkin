// Package paintedskin 是根包：构造器糖 + 容器（ADR-004「保留模式对象树 + 构造器糖」）。
//
// 典型用法：
//
//	err := app.Run(app.Config{
//	    Setup: func(a *app.App) { paintedskin.Mount(a, paintedskin.Column(
//	        paintedskin.Label("Hello"),
//	        paintedskin.Button("确定", func() { ... }),
//	    )) },
//	})
package paintedskin

import (
	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/style"
	"PaintedSkin/widget"
	"PaintedSkin/widget/layout"
)

// ---- 叶子构造器 ----

// Label 返回文本控件。
func Label(text string) *widget.Label { return widget.NewLabel(text) }

// Button 返回按钮控件。
func Button(text string, onClick func()) *widget.Button {
	b := widget.NewButton(text)
	b.SetOnClick(onClick)
	return b
}

// DisabledButton 返回禁用态按钮。
func DisabledButton(text string) *widget.Button {
	b := widget.NewButton(text)
	b.SetDisabled(true)
	return b
}

// Checkbox 返回复选框。
func Checkbox(text string) *widget.Checkbox { return widget.NewCheckbox(text) }

// RadioGroup 返回单选组。
func RadioGroup(name string, items []string) *widget.RadioGroup {
	return widget.NewRadioGroup(name, items)
}

// Slider 返回滑块（默认垂直，见 HSlider）。
func Slider() *widget.Slider { return widget.NewSlider() }

// HSlider 返回水平滑块（行高 1，最常用形态）。
func HSlider(value int) *widget.Slider {
	s := widget.NewSlider()
	s.SetHorizontal(true)
	s.SetRange(0, 100)
	if value != 0 {
		s.SetValue(value)
	}
	return s
}

// ProgressBar 返回进度条。
func ProgressBar() *widget.ProgressBar { return widget.NewProgressBar() }

// TextInput 返回单行输入框。
func TextInput() *widget.TextInput { return widget.NewTextInput() }

// List 返回列表。
func List(items ...string) *widget.List {
	l := widget.NewList()
	l.SetItems(items)
	return l
}

// Table 返回表格（先设列，随后用 TableRows 填数据）。
func Table(cols ...widget.TableColumn) *widget.Table {
	tb := widget.NewTable()
	tb.SetColumns(cols)
	return tb
}

// TableRows 设置表格数据并返回自身（便于链式）。
func TableRows(tb *widget.Table, rows [][]string) *widget.Table {
	tb.SetRows(rows)
	return tb
}

// Tabs 返回标签页容器。
func Tabs(labels ...string) *widget.Tabs {
	tb := widget.NewTabs()
	tb.SetTabs(labels)
	return tb
}

// Dropdown 返回下拉选择器。
func Dropdown(items ...string) *widget.Dropdown {
	d := widget.NewDropdown()
	d.SetItems(items)
	return d
}

// StatusBar 返回状态栏（左/中/右三段）。
func StatusBar(left, center, right string) *widget.StatusBar {
	s := widget.NewStatusBar()
	s.SetLeft(left)
	s.SetCenter(center)
	s.SetRight(right)
	return s
}

// Separator 返回分隔线。
func Separator() *widget.Separator { return widget.NewSeparator() }

// Panel 返回带边框容器。
func Panel(title string, child widget.Widget) *widget.Panel {
	p := widget.NewPanel(title)
	p.SetChild(child)
	return p
}

// Dialog 返回模态对话框。
func Dialog(title, message string, buttons []string, onButton func(int)) *widget.Dialog {
	return widget.NewDialog(title, message, buttons, onButton)
}

// mounter 是能接收控件树的宿主（app.App 满足）。
type mounter interface {
	Add(widget.Widget)
}

// Mount 把构造好的控件树挂到宿主（通常是 app.App）。
func Mount(m mounter, w widget.Widget) {
	m.Add(w)
}

// ---- 修饰器 ----

// item 是一个待排布的子项：控件 + Grow 权重 + 修饰标记。
type item struct {
	w        widget.Widget
	grow     float64
	fixedH   int // >0 时固定高度
	centered bool
	gap      int // 该项之前的间距
	hasGap   bool
}

// optCentered 让 Row 的子项在交叉轴居中。
type optCentered struct{}

func (optCentered) isOption() {}

// Centered 让子项在交叉轴居中（用于 Row）。
func Centered(w widget.Widget) any { return &centeredItem{w: w} }

type centeredItem struct{ w widget.Widget }

func (*centeredItem) isOption() {}

// GrowItem 标记某项按权重占据剩余空间。
type GrowItem struct {
	Weight float64
	W      widget.Widget
}

func (*GrowItem) isOption() {}

// Grow 让某项按权重占据剩余空间。
func Grow(weight float64, w widget.Widget) any {
	return &GrowItem{Weight: weight, W: w}
}

// Gap 设置兄弟间距，其后可跟控件或其他修饰器。
func Gap(n int, rest ...any) any {
	return &gapOpt{n: n, rest: rest}
}

// fixedItem 标记某项的固定高度。
type fixedItem struct {
	h int
	w widget.Widget
}

func (*fixedItem) isOption() {}

// Fixed 让控件占据固定高度（用于 List/Table 这类需要成屏显示的控件）。
func Fixed(h int, w widget.Widget) any { return &fixedItem{h: h, w: w} }

type gapOpt struct {
	n    int
	rest []any
}

func (*gapOpt) isOption() {}

// parse 把可变参数解析为 (子项列表, 对齐, 间距)。
// 支持三种修饰：Centered(w)、Grow(weight, w)、Gap(n, rest...)。
func parse(args []any) ([]item, layout.Align, int) {
	var items []item
	align := layout.AlignStart
	gap := 0
	for _, a := range args {
		switch v := a.(type) {
		case widget.Widget:
			items = append(items, item{w: v})
		case *centeredItem:
			align = layout.AlignCenter
			items = append(items, item{w: v.w, centered: true})
		case *GrowItem:
			items = append(items, item{w: v.W, grow: v.Weight})
		case *fixedItem:
			items = append(items, item{w: v.w, fixedH: v.h})
		case *gapOpt:
			gap = v.n
			sub, _, _ := parse(v.rest)
			items = append(items, sub...)
		case int:
			gap = v
		}
	}
	return items, align, gap
}

// ---- 容器 ----

// colT 是纵向容器：子项自上而下依次排布。
type colT struct {
	base  widget.Base
	items []item
	align layout.Align
	gap   int
}

// Column 构造纵向容器（accepts 任意组合的控件与修饰器）。
func Column(args ...any) *colT {
	items, align, gap := parse(args)
	c := &colT{items: items, align: align, gap: gap}
	c.base = widget.NewBase()
	c.base.Bind(c)
	for _, it := range c.items {
		if it.w != nil {
			setParent(it.w, c)
		}
	}
	return c
}

// BoundsOf 返回容器边界（便于测试与调试）。
func (c *colT) BoundsOf() geom.Rect { return c.base.Bounds }

// Add 追加子项。
func (c *colT) Add(w widget.Widget) {
	if w != nil {
		c.items = append(c.items, item{w: w})
		setParent(w, c)
	}
}

// Children 返回子控件。
func (c *colT) Children() []widget.Widget {
	out := make([]widget.Widget, 0, len(c.items))
	for _, it := range c.items {
		if it.w != nil {
			out = append(out, it.w)
		}
	}
	return out
}

// SetHost 递归挂接宿主与主题。
func (c *colT) SetHost(h widget.Host) {
	c.base.SetHost(h)
	for _, w := range c.Children() {
		w.SetHost(h)
	}
}

// SetTheme 递归注入主题。
func (c *colT) SetTheme(t *widget.Theme) {
	c.base.SetTheme(t)
	for _, w := range c.Children() {
		if st, ok := w.(interface{ SetTheme(*widget.Theme) }); ok {
			st.SetTheme(t)
		}
	}
}

// Measure 返回各子项高度之和。
func (c *colT) Measure(lc layout.Constraints) geom.Size {
	h := 0
	w := 0
	for _, it := range c.items {
		sz := it.w.Measure(lc)
		h += sz.H
		if sz.W > w {
			w = sz.W
		}
	}
	h += c.gap * maxInt(0, len(c.items)-1)
	return lc.Clamp(geom.Size{W: w, H: h})
}

// Layout 自上而下逐项排布；有 grow 权重时按剩余空间分配。
func (c *colT) Layout(r geom.Rect) {
	c.base.Bounds = r
	rects := c.flex(r)
	for i, it := range c.items {
		if i >= len(rects) {
			break
		}
		it.w.Layout(rects[i])
	}
}

// flex 计算各子项 Rect：主轴用布局引擎（有 grow 则分配剩余），交叉轴按 align。
func (c *colT) flex(r geom.Rect) []geom.Rect {
	fis := make([]layout.FlexItem, len(c.items))
	for i, it := range c.items {
		sz := it.w.Measure(layout.Constraints{MaxW: r.W, MaxH: r.H})
		if it.fixedH > 0 {
			// 固定高度：Min=Pref=fixedH，使其不被压缩。
			sz.H = it.fixedH
		}
		fis[i] = layout.FlexItem{Min: geom.Size{H: sz.H}, Pref: sz, Grow: it.grow}
	}
	return layout.FlexColumn(r, fis, c.align, c.gap)
}

// SetStyle 设置容器自身的背景样式。
//
// 零值 = **透明**（不铺底）：叶控件各自铺自己的底，容器默认不代劳，
// 这样兄弟控件与终端背景不会被容器整片盖掉。给了非零 Style 才铺满 Bounds。
func (c *colT) SetStyle(s style.Style) { c.base.SetStyle(s) }

// Paint 先按需铺自身背景（Style 非零时），再依次绘制子项。
func (c *colT) Paint(cv screen.Canvas) {
	if c.base.Style != (style.Style{}) && !c.base.Bounds.Empty() {
		cv.FillStyle(c.base.Bounds, c.base.Style)
	}
	for _, it := range c.items {
		it.w.Paint(cv)
	}
}

// Handle 不消费（冒泡）。
func (c *colT) Handle(event.Event) bool { return false }

// HitTest 命中子项，否则自身。
func (c *colT) HitTest(p geom.Point) widget.Widget {
	if !c.base.Bounds.Contains(p) {
		return nil
	}
	for i := len(c.items) - 1; i >= 0; i-- {
		if hit := c.items[i].w.HitTest(p); hit != nil {
			return hit
		}
	}
	return c
}

// rowT 是横向容器：子项自左向右依次排布。
type rowT struct {
	base  widget.Base
	items []item
	align layout.Align
	gap   int
}

// Row 构造横向容器。
func Row(args ...any) *rowT {
	items, align, gap := parse(args)
	r := &rowT{items: items, align: align, gap: gap}
	r.base = widget.NewBase()
	r.base.Bind(r)
	for _, it := range r.items {
		if it.w != nil {
			setParent(it.w, r)
		}
	}
	return r
}

// BoundsOf 返回容器边界。
func (r *rowT) BoundsOf() geom.Rect { return r.base.Bounds }

// Add 追加子项。
func (r *rowT) Add(w widget.Widget) {
	if w != nil {
		r.items = append(r.items, item{w: w})
		setParent(w, r)
	}
}

// Children 返回子控件。
func (r *rowT) Children() []widget.Widget {
	out := make([]widget.Widget, 0, len(r.items))
	for _, it := range r.items {
		if it.w != nil {
			out = append(out, it.w)
		}
	}
	return out
}

// SetHost 递归挂接宿主。
func (r *rowT) SetHost(h widget.Host) {
	r.base.SetHost(h)
	for _, w := range r.Children() {
		w.SetHost(h)
	}
}

// SetTheme 递归注入主题。
func (r *rowT) SetTheme(t *widget.Theme) {
	r.base.SetTheme(t)
	for _, w := range r.Children() {
		if st, ok := w.(interface{ SetTheme(*widget.Theme) }); ok {
			st.SetTheme(t)
		}
	}
}

// Measure 返回各子项宽度之和的最大值与高度最大值。
func (r *rowT) Measure(lc layout.Constraints) geom.Size {
	w, h := 0, 0
	for _, it := range r.items {
		sz := it.w.Measure(lc)
		w += sz.W
		if sz.H > h {
			h = sz.H
		}
	}
	w += r.gap * maxInt(0, len(r.items)-1)
	return lc.Clamp(geom.Size{W: w, H: h})
}

// Layout 自左向右逐项排布；有 grow 权重时按剩余空间分配。
func (r *rowT) Layout(rect geom.Rect) {
	r.base.Bounds = rect
	rects := r.flex(rect)
	for i, it := range r.items {
		if i >= len(rects) {
			break
		}
		it.w.Layout(rects[i])
	}
}

// flex 计算各子项 Rect：主轴用布局引擎（有 grow 则分配剩余），交叉轴按 align。
func (r *rowT) flex(rect geom.Rect) []geom.Rect {
	fis := make([]layout.FlexItem, len(r.items))
	for i, it := range r.items {
		sz := it.w.Measure(layout.Constraints{MaxW: rect.W, MaxH: rect.H})
		fis[i] = layout.FlexItem{Min: geom.Size{W: 1}, Pref: sz, Grow: it.grow}
	}
	return layout.FlexRow(rect, fis, r.align, r.gap)
}

// SetStyle 设置容器自身的背景样式（零值 = 透明，见 colT.SetStyle）。
func (r *rowT) SetStyle(s style.Style) { r.base.SetStyle(s) }

// Paint 先按需铺自身背景（Style 非零时），再依次绘制子项。
func (r *rowT) Paint(cv screen.Canvas) {
	if r.base.Style != (style.Style{}) && !r.base.Bounds.Empty() {
		cv.FillStyle(r.base.Bounds, r.base.Style)
	}
	for _, it := range r.items {
		it.w.Paint(cv)
	}
}

// Handle 不消费。
func (r *rowT) Handle(event.Event) bool { return false }

// HitTest 命中子项，否则自身。
func (r *rowT) HitTest(p geom.Point) widget.Widget {
	if !r.base.Bounds.Contains(p) {
		return nil
	}
	for i := len(r.items) - 1; i >= 0; i-- {
		if hit := r.items[i].w.HitTest(p); hit != nil {
			return hit
		}
	}
	return r
}

// alignCross 返回交叉轴按 align 的 (尺寸, 起点)。
func alignCross(origin, total, pref int, a layout.Align) (int, int) {
	if pref >= total {
		return total, origin
	}
	switch a {
	case layout.AlignCenter:
		return pref, origin + (total-pref)/2
	case layout.AlignEnd:
		return pref, origin + total - pref
	default:
		return pref, origin
	}
}

// alignCenter 返回交叉轴居中的 (位置, 尺寸)。
func alignCenter(origin, total, pref int) (int, int) {
	if pref >= total {
		return origin, total
	}
	return origin + (total-pref)/2, pref
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ensure style 被引用（容器默认样式）。
var _ style.Style = style.Default

func setParent(w, parent widget.Widget) {
	if p, ok := w.(interface{ SetParent(widget.Widget) }); ok {
		p.SetParent(parent)
	}
}
