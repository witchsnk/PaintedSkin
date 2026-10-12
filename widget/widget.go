package widget

import (
	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/style"
	"PaintedSkin/widget/layout"
)

// Widget 是所有 UI 控件的根接口（docs/06）。
//
// 四阶段生命周期：Measure（自底向上报期望尺寸）→ Layout（自顶向下分地盘）
// → Paint（画到 Canvas，越界由 Canvas/上层裁剪）。事件经 Handle 消费，
// 未消费的由容器/Root 沿父链冒泡。
//
// 注：docs/06 原文 Paint 签名为 *screen.Canvas；Canvas 本身是接口，
// 取指针无意义，此处用接口值（见 ADR-011）。
type Widget interface {
	// Measure 按约束报告期望尺寸，返回值必须落在约束区间内。
	Measure(c layout.Constraints) geom.Size
	// Layout 接受分配到的区域（含定位子控件）。
	Layout(r geom.Rect)
	// Paint 绘制到画布。
	Paint(c screen.Canvas)
	// Handle 处理事件，返回 true 表示已消费。
	Handle(e event.Event) bool
	// HitTest 返回落点处 z 序最顶层的控件（含自身），无命中返回 nil。
	HitTest(p geom.Point) Widget
	// Children 返回子控件（叶控件返回 nil）。
	Children() []Widget
	// SetHost 递归挂接宿主（聚焦/重绘/弹层经 Host 访问，不直连 app）。
	SetHost(h Host)
}

// Host 是控件访问框架能力的接口（docs/06）。控件禁止 import app 包。
type Host interface {
	// Invalidate 打脏，请求重绘。
	Invalidate()
	// Focus 主动聚焦。
	Focus(w Widget)
	// PushDialog 模态入栈（屏蔽栈下输入 + 焦点陷阱）。
	PushDialog(w Widget)
	// PopDialog 模态出栈并恢复焦点。
	PopDialog()
	// SetOverlay 设置单实例覆盖层（Dropdown 弹层，M6；M5 只建机制）。
	SetOverlay(w Widget)
	// Capture 显式请求鼠标捕获。
	Capture(w Widget)
}

// Base 提供公共字段与默认实现，控件只写差异逻辑（docs/06）。
type Base struct {
	Visible   bool
	Enabled   bool
	Focusable bool
	Style     style.Style
	Bounds    geom.Rect
	Dirty     bool
	Parent    Widget

	host      Host
	focused   bool
	self      Widget // 外层控件身份（HitTest 返回用；构造时 Bind）
	th        *Theme // 可选主题（nil = 用控件 Style 兜底）
	focusRing bool   // 是否在聚焦时套一圈 outline（默认关闭）
}

// Theme 是控件绘制所需的样式集合。由 app 层从 theme 包转换注入，
// 避免 widget→theme 的依赖环。控件优先用自身 Style，未设时回落主题。
type Theme struct {
	ButtonNormal, ButtonHover, ButtonPressed, ButtonFocused, ButtonDisabled style.Style
	ListSelected, TableHeader, DialogMask, ScrollbarThumb, ScrollbarTrack   style.Style
	Separator, StatusBar, TextInputNormal, TextInputFocus                   style.Style

	// BorderColor 是 Panel/Dialog 的框线前景色。
	BorderColor style.Color
	// FocusColor 是焦点态前景色（FocusColor 为零值时回落到内置"反色"）。
	FocusColor style.Color
	// DisabledColor 是禁用态前景色（为零值时回落到内置"暗淡"）。
	DisabledColor style.Color
	// HoverBG 是悬停态背景色（为零值时回落到内置"加粗"）。
	HoverBG style.Color
	// SelectionBG 是选中态背景色（为零值时回落到内置"反色"）。
	SelectionBG style.Color
	// ScrollbarThumbColor 是滚动条滑块前景色：ScrollbarThumb 未设时的回落值。
	ScrollbarThumbColor style.Color
}

// SetTheme 注入主题（容器需递归）。
func (b *Base) SetTheme(t *Theme) { b.th = t }

// ThemePtr 返回注入的主题（可能为 nil）。
func (b *Base) ThemePtr() *Theme { return b.th }

// pick 在控件自身 Style 与主题回落之间二选一（docs/08：控件优先）。
func (b *Base) pick(own, themed style.Style) style.Style {
	if own != (style.Style{}) {
		return own
	}
	if b.th != nil {
		return themed
	}
	return own
}

// ---- 语义样式（focus / hover / selected / disabled）----
//
// 这四种"控件语义状态"此前是把 AttrReverse / WithDim / WithBold 写死在各控件
// 的 Paint 里，导致完全不可换肤。现在统一走主题的语义色：
//
//	focus     FocusColor（前景）→ 无主题回落到反色
//	hover     HoverBG（背景）    → 无主题回落到加粗
//	selected  SelectionBG（背景）→ 无主题回落到反色
//	disabled  DisabledColor（前景）→ 无主题回落到暗淡
//
// 规则与 pick/applyAll 一致：**控件自身 Style 优先**，其次主题语义色，
// 最后内置兜底。语义辅助只返回新值，不修改控件字段。

// focusStyle 返回焦点态样式。
func (b *Base) focusStyle(st style.Style) style.Style {
	if b.Style != (style.Style{}) {
		return b.Style
	}
	if c := b.themeOrZero().FocusColor; c.Kind != style.ColorDefault {
		return applyAll(st, style.Style{}.WithForeground(c), nil)
	}
	return st.WithAttrs(style.AttrReverse)
}

// hoverStyle 返回悬停态样式。
func (b *Base) hoverStyle(st style.Style) style.Style {
	if b.Style != (style.Style{}) {
		return b.Style
	}
	if c := b.themeOrZero().HoverBG; c.Kind != style.ColorDefault {
		return applyAll(st, style.Style{}.WithBackground(c), nil)
	}
	return st.WithBold(true)
}

// selectedStyle 返回选中态样式。
//
// 刻意**不**回落 FocusColor：选中与焦点是两个语义位置，串味会让"未聚焦的
// 选中项"看起来像聚焦。都没有时回落到内置"反色"。
func (b *Base) selectedStyle(st style.Style) style.Style {
	if b.Style != (style.Style{}) {
		return b.Style
	}
	if c := b.themeOrZero().SelectionBG; c.Kind != style.ColorDefault {
		return applyAll(st, style.Style{}.WithBackground(c), nil)
	}
	return st.WithAttrs(style.AttrReverse)
}

// disabledStyle 返回禁用态样式。
func (b *Base) disabledStyle(st style.Style) style.Style {
	if c := b.themeOrZero().DisabledColor; c.Kind != style.ColorDefault {
		return applyAll(st, style.Style{}.WithForeground(c), nil)
	}
	return st.WithDim(true)
}

// SetFocusRing 开关"聚焦时套一圈 outline"（Textual §1.5 的焦点环）。
//
// 默认关闭：叶子控件的 Bounds 就是全部内容，套环会盖住首末字符；
// 有外框的控件（`TextInput` 等）打开后观感最好。outline 只换样式、
// 不动字符，因此**不吃内容也不占布局**。
func (b *Base) SetFocusRing(v bool) {
	b.focusRing = v
	b.Invalidate()
}

// FocusRingEnabled 报告是否启用焦点环。
func (b *Base) FocusRingEnabled() bool { return b.focusRing }

// drawFocusRing 在聚焦且启用时套一圈 outline（用焦点语义样式）。
// rect 通常传控件自身 Bounds —— 不需要为环预留任何布局空间。
func (b *Base) drawFocusRing(c screen.Canvas, rect geom.Rect) {
	if !b.focusRing || !b.focused || rect.Empty() {
		return
	}
	c.Outline(rect, screen.AllEdges, b.focusStyle(b.Style).WithUnderline(true))
}

// themeOrZero 返回注入的主题；**无主题时返回零值主题**，因此调用方
// 可以无条件读字段（`b.themeOrZero().Separator`）而不会 nil 解引用。
// 零值字段的语义统一是"未设 → 走内置兜底"。
func (b *Base) themeOrZero() Theme {
	if b.th == nil {
		return Theme{}
	}
	return *b.th
}

// themed 在主题字段未设（零值）时回落到 fallback。
func (b *Base) themed(field, fallback style.Style) style.Style {
	if field != (style.Style{}) {
		return field
	}
	return fallback
}

// Border 返回边框色（优先控件 Style 的前景色，否则主题 BorderColor）。
func (b *Base) borderColor() style.Style {
	if b.Style.FG.Kind != style.ColorDefault {
		return b.Style
	}
	if c := b.themeOrZero().BorderColor; c.Kind != style.ColorDefault {
		return style.Style{}.WithForeground(c)
	}
	return b.Style
}

// Bind 绑定外层控件身份（嵌入 Base 的控件构造后必须调用一次）。
func (b *Base) Bind(w Widget) { b.self = w }

// NewBase 返回字段就绪的 Base（可见、可交互；焦点默认不可，需控件显式声明）。
func NewBase() Base {
	return Base{Visible: true, Enabled: true}
}

// SetHost 挂接宿主（叶控件版本；容器需覆盖以递归）。
func (b *Base) SetHost(h Host) { b.host = h }

// HostOf 返回已挂接的宿主（可能为 nil）。
func (b *Base) HostOf() Host { return b.host }

// SetParent 设置父控件（容器在 Add/SetChild/Layout 时调用）。
func (b *Base) SetParent(p Widget) { b.Parent = p }

// ParentWidget 返回父控件。
func (b *Base) ParentWidget() Widget { return b.Parent }

// Focused 报告是否持有焦点。
func (b *Base) Focused() bool { return b.focused }

// OnFocus 获得焦点：置位并打脏（FocusManager 调用）。
func (b *Base) OnFocus() {
	b.focused = true
	b.Dirty = true
}

// OnBlur 失去焦点：复位并打脏（FocusManager 调用）。
func (b *Base) OnBlur() {
	b.focused = false
	b.Dirty = true
}

// Invalidate 标记自身脏并请求宿主重绘（无宿主时仅置 Dirty）。
func (b *Base) Invalidate() {
	b.Dirty = true
	if b.host != nil {
		b.host.Invalidate()
	}
}

// EffectivelyVisible 含祖先链的可见性。
func (b *Base) EffectivelyVisible() bool {
	if !b.Visible {
		return false
	}
	for p := b.Parent; p != nil; {
		pb, ok := p.(interface{ ParentWidget() Widget })
		bv, ok2 := p.(interface{ IsVisible() bool })
		if ok2 && !bv.IsVisible() {
			return false
		}
		if !ok {
			return true
		}
		p = pb.ParentWidget()
	}
	return true
}

// EffectivelyEnabled 含祖先链的可用性。
func (b *Base) EffectivelyEnabled() bool {
	if !b.Enabled {
		return false
	}
	for p := b.Parent; p != nil; {
		pb, ok := p.(interface{ ParentWidget() Widget })
		be, ok2 := p.(interface{ IsEnabled() bool })
		if ok2 && !be.IsEnabled() {
			return false
		}
		if !ok {
			return true
		}
		p = pb.ParentWidget()
	}
	return true
}

// IsVisible 返回自身可见标志。
func (b *Base) IsVisible() bool { return b.Visible }

// IsEnabled 返回自身可用标志。
func (b *Base) IsEnabled() bool { return b.Enabled }

// IsFocusable 报告是否可参与 Tab 遍历（Focusable && 有效可见 && 有效可用）。
func (b *Base) IsFocusable() bool {
	return b.Focusable && b.EffectivelyVisible() && b.EffectivelyEnabled()
}

// Measure 默认实现：零尺寸。
func (b *Base) Measure(c layout.Constraints) geom.Size {
	return c.Clamp(geom.Size{})
}

// Layout 默认实现：记录边界。
func (b *Base) Layout(r geom.Rect) { b.Bounds = r }

// Paint 默认实现：无绘制。
func (b *Base) Paint(screen.Canvas) {}

// paintBackground 铺控件自身背景：控件 Paint 的第一步。
//
// 纪律（docs/05 / docs/11 P0）：控件 Bounds 内每格要么被写入真实内容，
// 要么被写入"控件自己的背景色空格"，不存在第三种状态。
//
// **零样式 = 透明**（不铺）：控件没有背景色时若也铺，等于用"终端默认背景"
// 盖掉父容器的主题底色。零值格与零样式空格在终端上本来就同色
// （`lineDiff` 两者都发一个带默认样式的空格），所以不铺既不产生空洞，
// 又能让 `Column`/`Row`/`Root` 的容器底色透上来 —— 这正是分层背景的前提。
func paintBackground(c screen.Canvas, r geom.Rect, st style.Style) {
	if st == (style.Style{}) || r.Empty() {
		return
	}
	c.FillStyle(r, st)
}

// Handle 默认实现：不消费（调用方沿父链冒泡）。
func (b *Base) Handle(event.Event) bool { return false }

// HitTest 默认实现：有效可见且落点在边界内则命中外层控件（Bind 绑定者）。
func (b *Base) HitTest(p geom.Point) Widget {
	if !b.EffectivelyVisible() {
		return nil
	}
	if b.Bounds.Contains(p) {
		if b.self != nil {
			return b.self
		}
		return b
	}
	return nil
}

// Children 默认实现：无子控件。
func (b *Base) Children() []Widget { return nil }

// BoundsOf 返回边界（与字段 Bounds 同值的访问方法，便于跨包取值）。
func (b *Base) BoundsOf() geom.Rect { return b.Bounds }

// SetVisible 切换可见性。
func (b *Base) SetVisible(v bool) { b.Visible = v; b.Invalidate() }

// SetEnabled 切换可用性。
func (b *Base) SetEnabled(v bool) { b.Enabled = v; b.Invalidate() }

// SetStyle 设置控件自有样式（覆盖主题回落）。
func (b *Base) SetStyle(s style.Style) { b.Style = s; b.Invalidate() }

// SetFocusable 设置是否参与 Tab 遍历。
func (b *Base) SetFocusable(v bool) { b.Focusable = v; b.Invalidate() }

// BoundsCenter 返回边界中心（合成 hover 事件坐标用）。
func (b *Base) BoundsCenter() geom.Point {
	return geom.Point{X: b.Bounds.X + b.Bounds.W/2, Y: b.Bounds.Y + b.Bounds.H/2}
}

// setParent 把 w 的父设为 p（w 未嵌入 Base 时静默跳过）。
func setParent(w, p Widget) {
	if sp, ok := w.(interface{ SetParent(Widget) }); ok {
		sp.SetParent(p)
	}
}

// setHost 把 w 的宿主设为 h（顺带递归子控件）。
func setHost(w Widget, h Host) {
	w.SetHost(h)
	for _, c := range w.Children() {
		if c != nil {
			setHost(c, h)
		}
	}
}

// setTheme 把主题注入 w（顺带递归子控件）。
func setTheme(w Widget, t *Theme) {
	if st, ok := w.(interface{ SetTheme(*Theme) }); ok {
		st.SetTheme(t)
	}
	for _, c := range w.Children() {
		if c != nil {
			setTheme(c, t)
		}
	}
}

// notifyFocus 触发获焦通知（支持 OnFocus 的控件）。
func notifyFocus(w Widget) {
	if f, ok := w.(interface{ OnFocus() }); ok {
		f.OnFocus()
	}
}

// notifyBlur 触发失焦通知。
func notifyBlur(w Widget) {
	if f, ok := w.(interface{ OnBlur() }); ok {
		f.OnBlur()
	}
}
