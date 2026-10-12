package widget

import (
	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/text"
	"PaintedSkin/widget/layout"
)

// BorderStyle 是 Panel/Dialog 的边框样式（docs/08）。
type BorderStyle int

const (
	// BorderNone 无边框。
	BorderNone BorderStyle = iota
	// BorderSingle 单线边框。
	BorderSingle
	// BorderDouble 双线边框。
	BorderDouble
	// BorderRound 圆角边框。
	BorderRound
	// BorderHeavy 粗边框。
	BorderHeavy
)

// borderSpec 把边框样式枚举映射为 screen.Border（字形表）。
//
// 边框语义层与文本属性层解耦：这里不再用 style.Attr 位掩码"传输"边框线型
// （旧实现 BorderDouble→AttrBold→'━'、BorderHeavy→AttrBold|Reverse→'═'，
// 两者语义互相对调）。
func borderSpec(s BorderStyle) screen.Border {
	switch s {
	case BorderSingle:
		return screen.BorderSingle
	case BorderDouble:
		return screen.BorderDouble
	case BorderRound:
		return screen.BorderRound
	case BorderHeavy:
		return screen.BorderHeavy
	default:
		return screen.BorderNone
	}
}

// borderTitleInset 是标题相对边框左上角的固定偏移：左右各留 1 格空格。
const borderTitleInset = 2

// borderTitleBudget 返回给定框宽下标题可占的列数（框宽 - 左右各 2 格）。
// 标题列预算恒 ≤ 框宽 - 4，因此永不覆盖右边框。
func borderTitleBudget(frameW int) int {
	n := frameW - 2*borderTitleInset
	if n < 0 {
		return 0
	}
	return n
}

// borderTitle 返回可安全绘制的标题文本：按框宽列预算截断（不劈宽字符），
// 退化框宽下返回空串。Panel 与 Dialog 共用同一策略。
func borderTitle(title string, frameW int) string {
	if title == "" {
		return ""
	}
	return text.Truncate(title, borderTitleBudget(frameW))
}

// Panel 是容器控件（docs/08）：边框 + 标题 + 内边距 + 内容子控件。
// Measure = 内容 + padding + border；自身不消费事件（冒泡给父）。
type Panel struct {
	Base
	title   string
	border  BorderStyle
	spec    screen.Border // 覆盖 border 枚举（SetBorderSpec）；零值表示用枚举
	hatch   screen.Hatch  // 内区底纹装饰
	padding geom.Insets
	child   Widget
}

// NewPanel 创建单线边框 Panel（默认内边距 1，内容不贴边框）。
func NewPanel(title string) *Panel {
	p := &Panel{title: title, border: BorderSingle, padding: geom.SymInsets(1)}
	p.Base = NewBase()
	p.Base.Bind(p)
	return p
}

// SetTitle 设置标题。
func (p *Panel) SetTitle(s string) {
	p.title = s
	p.Invalidate()
}

// Title 返回标题。
func (p *Panel) Title() string { return p.title }

// SetBorder 设置边框样式（枚举档）。
func (p *Panel) SetBorder(s BorderStyle) {
	p.border = s
	p.spec = screen.Border{}
	p.Invalidate()
}

// Border 返回边框样式枚举档。
func (p *Panel) Border() BorderStyle { return p.border }

// SetBorderSpec 直接指定字形表里（或自建）的任意边框样式。
// 传入 screen.Border{} 表示回到枚举档。
func (p *Panel) SetBorderSpec(b screen.Border) {
	p.spec = b
	p.Invalidate()
}

// BorderSpec 返回生效的字形表边框（枚举档会被解析成对应字形）。
func (p *Panel) BorderSpec() screen.Border {
	if p.spec.Visible() {
		return p.spec
	}
	return borderSpec(p.border)
}

// SetBorderEdges 逐边开关（对应 Textual 的 border-top/right/bottom/left）。
// 只改开关、不改字形；不影响布局（只要任一边开着就仍占 1 格边框）。
func (p *Panel) SetBorderEdges(e screen.BorderEdges) {
	p.spec = p.BorderSpec().WithEdges(e)
	p.Invalidate()
}

// BorderEdges 返回当前逐边开关。
func (p *Panel) BorderEdges() screen.BorderEdges { return p.BorderSpec().Edges }

// SetHatch 设置内区底纹（screen.HatchNone / Left / Right / Cross）。
// 底纹只落在内区的空格上，不覆盖边框也不覆盖内容。
func (p *Panel) SetHatch(h screen.Hatch) {
	p.hatch = h
	p.Invalidate()
}

// Hatch 返回内区底纹。
func (p *Panel) Hatch() screen.Hatch { return p.hatch }

// SetPadding 设置内边距。
func (p *Panel) SetPadding(in geom.Insets) {
	p.padding = in
	p.Invalidate()
}

// Padding 返回内边距。
func (p *Panel) Padding() geom.Insets { return p.padding }

// SetChild 设置内容子控件（挂接父链与宿主）。
func (p *Panel) SetChild(w Widget) {
	p.child = w
	if w != nil {
		setParent(w, p)
		if h := p.HostOf(); h != nil {
			setHost(w, h)
		}
	}
	p.Invalidate()
}

// Child 返回内容子控件。
func (p *Panel) Child() Widget { return p.child }

// Children 返回子控件集。
func (p *Panel) Children() []Widget {
	if p.child == nil {
		return nil
	}
	return []Widget{p.child}
}

// SetHost 递归挂接宿主。
func (p *Panel) SetHost(h Host) {
	p.Base.SetHost(h)
	if p.child != nil {
		setHost(p.child, h)
	}
}

func (p *Panel) chrome() int {
	if p.border == BorderNone {
		return 0
	}
	return 1
}

// contentConstraints 把外层约束换算为内容区约束。
func (p *Panel) contentConstraints(c layout.Constraints) layout.Constraints {
	d := p.chrome()*2 + p.padding.Horizontal()
	dv := p.chrome()*2 + p.padding.Vertical()
	out := c
	out.MinW -= d
	out.MinH -= dv
	out.MaxW -= d
	out.MaxH -= dv
	return out
}

// titleRows 返回标题占用的高度（无标题或无边框时为 0）。
func (p *Panel) titleRows() int {
	if p.title == "" {
		return 0
	}
	return 1
}

// Measure = 内容 + 标题行 + padding + border。
func (p *Panel) Measure(c layout.Constraints) geom.Size {
	content := geom.Size{}
	if p.child != nil {
		content = p.child.Measure(p.contentConstraints(c))
		content.H += p.titleRows()
	}
	d := p.chrome()*2 + p.padding.Horizontal()
	dv := p.chrome()*2 + p.padding.Vertical()
	w := content.W + d
	if tw := text.Width(p.title) + 4; tw > w {
		w = tw
	}
	return c.Clamp(geom.Size{W: w, H: content.H + dv})
}

// Layout 记录边界并把内容区（扣掉标题行）Layout 给子控件。
func (p *Panel) Layout(r geom.Rect) {
	p.Bounds = r
	if p.child != nil {
		p.child.Layout(p.contentRect())
	}
}

// contentRect 返回子控件可用区：外框 - padding - border - 标题行。
func (p *Panel) contentRect() geom.Rect {
	r := layout.ContentRect(p.Bounds, p.padding, p.chrome())
	r.Y += p.titleRows()
	r.H -= p.titleRows()
	if r.H < 0 {
		r.H = 0
	}
	return r.Intersect(p.Bounds)
}

// Paint 铺底 → 画边框 → 画标题 → 画子控件。
//
// 顺序即纪律：先铺自身背景（内区 + padding 环），再叠加边框与内容，
// 保证框内每格要么是边框字形、要么是控件自己的背景（无空洞）。
func (p *Panel) Paint(c screen.Canvas) {
	if p.Bounds.Empty() {
		return
	}
	paintBackground(c, p.Bounds, p.Style)
	if p.hatch != screen.HatchNone {
		// 底纹只装饰内区（含 padding 环），不碰边框。
		c.Hatch(p.Bounds.Inflate(geom.SymInsets(p.chrome())), p.hatch, p.Style)
	}
	if spec := p.BorderSpec(); spec.Visible() {
		c.Box(p.Bounds, spec, p.borderColor())
	}
	// 标题画在边框内侧的首行，并按框宽截断，绝不覆盖或越过边框。
	// 高度不足以容下框内首行时不画标题（退化尺寸下边框优先）。
	if p.title != "" && p.Bounds.H > p.chrome()*2 {
		c.Text(p.Bounds.X+borderTitleInset, p.Bounds.Y+p.chrome(),
			borderTitle(p.title, p.Bounds.W), p.Style.WithBold(true))
	}
	if p.child != nil {
		p.child.Paint(c)
	}
}

// HitTest 先探子控件（z 序顶），否则自身。
func (p *Panel) HitTest(pt geom.Point) Widget {
	if !p.EffectivelyVisible() || !p.Bounds.Contains(pt) {
		return nil
	}
	if p.child != nil {
		if hit := p.child.HitTest(pt); hit != nil {
			return hit
		}
	}
	return p
}

// Handle 容器不消费（子未消费的上浮由 Root 沿父链冒泡）。
func (p *Panel) Handle(event.Event) bool { return false }
