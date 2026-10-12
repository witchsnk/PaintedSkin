package widget

import (
	"strings"

	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/text"
	"PaintedSkin/widget/layout"
)

// Dialog 是模态对话框（docs/08）：标题 + 内容 + 按钮组。
// 经 Host.PushDialog 入栈：模态屏蔽 + 焦点陷阱 + 焦点默认落第一个按钮；
// Esc/关闭 → PopDialog 并恢复原焦点。绘制时居中于所给区域，遮罩可选。
type Dialog struct {
	Base
	title    string
	message  string
	buttons  []*Button
	onButton func(i int)
	onClose  func()
	fixedW   int
	fixedH   int
	dim      bool
	border   BorderStyle
	spec     screen.Border // 覆盖 border 枚举（SetBorderSpec）；零值表示用枚举
}

// NewDialog 创建对话框；labels 为按钮组文本，onButton 收按钮索引（可为 nil）。
func NewDialog(title, message string, labels []string, onButton func(i int)) *Dialog {
	d := &Dialog{title: title, message: message, onButton: onButton, border: BorderSingle}
	d.Base = NewBase()
	d.Base.Bind(d)
	for i, lb := range labels {
		i := i
		btn := NewButton(lb)
		btn.SetOnClick(func() {
			if d.onButton != nil {
				d.onButton(i)
			}
		})
		btn.Base.SetParent(d)
		d.buttons = append(d.buttons, btn)
	}
	return d
}

// SetTitle 设置标题。
func (d *Dialog) SetTitle(s string) {
	d.title = s
	d.Invalidate()
}

// Title 返回标题。
func (d *Dialog) Title() string { return d.title }

// SetMessage 设置内容文本。
func (d *Dialog) SetMessage(s string) {
	d.message = s
	d.Invalidate()
}

// Message 返回内容文本。
func (d *Dialog) Message() string { return d.message }

// SetOnButton 设置按钮组回调（参数为按钮索引）。
func (d *Dialog) SetOnButton(fn func(i int)) { d.onButton = fn }

// SetOnClose 设置 Esc/关闭回调（PopDialog 之前触发）。
func (d *Dialog) SetOnClose(fn func()) { d.onClose = fn }

// SetSize 设置尺寸（相对/绝对；0 = 自适应内容）。
func (d *Dialog) SetSize(w, h int) {
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	d.fixedW, d.fixedH = w, h
	d.Invalidate()
}

// SetDim 开关遮罩。
func (d *Dialog) SetDim(v bool) {
	d.dim = v
	d.Invalidate()
}

// SetBorder 设置边框样式（枚举档）。
func (d *Dialog) SetBorder(s BorderStyle) {
	d.border = s
	d.spec = screen.Border{}
	d.Invalidate()
}

// Border 返回边框样式枚举档。
func (d *Dialog) Border() BorderStyle { return d.border }

// SetBorderSpec 直接指定字形表里的任意边框（传入零值回到枚举档）。
func (d *Dialog) SetBorderSpec(b screen.Border) {
	d.spec = b
	d.Invalidate()
}

// BorderSpec 返回生效的字形表边框。
func (d *Dialog) BorderSpec() screen.Border {
	if d.spec.Visible() {
		return d.spec
	}
	return borderSpec(d.border)
}

// SetBorderEdges 逐边开关（不改变布局）。
func (d *Dialog) SetBorderEdges(e screen.BorderEdges) {
	d.spec = d.BorderSpec().WithEdges(e)
	d.Invalidate()
}

// BorderEdges 返回当前逐边开关。
func (d *Dialog) BorderEdges() screen.BorderEdges { return d.BorderSpec().Edges }

// Buttons 返回按钮组（测试用）。
func (d *Dialog) Buttons() []*Button { return d.buttons }

// Close 主动关闭（经宿主 PopDialog，恢复原焦点）。
func (d *Dialog) Close() {
	if d.onClose != nil {
		d.onClose()
	}
	if h := d.HostOf(); h != nil {
		h.PopDialog()
	}
}

// Children 返回按钮组。
func (d *Dialog) Children() []Widget {
	out := make([]Widget, len(d.buttons))
	for i, b := range d.buttons {
		out[i] = b
	}
	return out
}

// SetHost 递归挂接宿主。
func (d *Dialog) SetHost(h Host) {
	d.Base.SetHost(h)
	for _, b := range d.buttons {
		b.SetHost(h)
	}
}

// contentSize 计算内容自适应尺寸。
func (d *Dialog) contentSize() geom.Size {
	w := text.Width(d.title) + 4
	lines := strings.Split(d.message, "\n")
	for _, ln := range lines {
		if sw := text.Width(ln); sw+4 > w {
			w = sw + 4
		}
	}
	btnW := 0
	for _, b := range d.buttons {
		btnW += text.Width(b.Text()) + 6
	}
	if btnW+2 > w {
		w = btnW + 2
	}
	if w < dialogMinW {
		w = dialogMinW
	}
	h := len(lines) + 5 // 标题行 + 内容 + 按钮行 + 边框
	if h < dialogMinH {
		h = dialogMinH
	}
	return geom.Size{W: w, H: h}
}

// Measure 自适应内容（或固定尺寸），钳制进约束。
func (d *Dialog) Measure(c layout.Constraints) geom.Size {
	want := d.contentSize()
	if d.fixedW > 0 {
		want.W = d.fixedW
	}
	if d.fixedH > 0 {
		want.H = d.fixedH
	}
	return c.Clamp(want)
}

// Layout 居中于所给区域，并排布按钮行。
func (d *Dialog) Layout(r geom.Rect) {
	size := d.Measure(layout.Constraints{MaxW: r.W, MaxH: r.H})
	d.Bounds = layout.Stack(r, size, layout.StackCenter)
	d.layoutButtons()
}

func (d *Dialog) layoutButtons() {
	n := len(d.buttons)
	if n == 0 {
		return
	}
	// 按钮行：底部边框上一行，均分。
	y := d.Bounds.Y + d.Bounds.H + dialogBtnRowDy
	slot := d.Bounds.W / (n + 1)
	for i, b := range d.buttons {
		bw := text.Width(b.Text()) + 4
		x := d.Bounds.X + slot*(i+1) - bw/2
		b.Layout(geom.Rect{X: x, Y: y, W: bw, H: 1})
	}
}

// Paint 铺底 → 画边框 → 画标题 → 画内容 → 画按钮。
//
// 遮罩（dim）不再是"才铺底"而是"铺什么底"：对话框总是先铺满自身背景，
// dim 时该背景连同边框、标题、正文一并变暗。
func (d *Dialog) Paint(c screen.Canvas) {
	if d.Bounds.Empty() {
		return
	}
	// 遮罩底色走主题 DialogMask；未设时回落到内置"暗淡"。
	base := d.Style
	if d.dim {
		base = d.disabledStyle(applyAll(d.Style, d.themeOrZero().DialogMask, nil))
	}
	paintBackground(c, d.Bounds, base)

	borderSt := d.borderColor()
	if d.dim {
		borderSt = d.disabledStyle(borderSt)
	}
	if spec := d.BorderSpec(); spec.Visible() {
		c.Box(d.Bounds, spec, borderSt)
	}

	// 标题与 Panel 同一策略：落在框内首行 + 按框宽截断，绝不压边框。
	// 高度不足以容下框内首行时（H <= 2）不画标题。
	if d.title != "" && d.Bounds.H > 2 {
		c.Text(d.Bounds.X+borderTitleInset, d.Bounds.Y+1,
			borderTitle(d.title, d.Bounds.W), base.WithBold(true))
	}
	lines := strings.Split(d.message, "\n")
	for i, ln := range lines {
		y := d.Bounds.Y + 2 + i
		if y >= d.Bounds.Bottom()-1 {
			break
		}
		// 正文按框宽截断，长行不会吃掉右边框。
		c.Text(d.Bounds.X+borderTitleInset, y,
			text.Truncate(ln, borderTitleBudget(d.Bounds.W)), base)
	}
	for _, b := range d.buttons {
		b.Paint(c)
	}
}

// HitTest 按钮优先，否则自身。
func (d *Dialog) HitTest(p geom.Point) Widget {
	if !d.EffectivelyVisible() || !d.Bounds.Contains(p) {
		return nil
	}
	for i := len(d.buttons) - 1; i >= 0; i-- {
		if hit := d.buttons[i].HitTest(p); hit != nil {
			return hit
		}
	}
	return d
}

// Handle Esc 关闭并冒泡其余（按钮事件由路由直达按钮）。
func (d *Dialog) Handle(e event.Event) bool {
	if k, ok := e.(event.Key); ok && k.Press && k.Code == event.KeyEsc {
		d.Close()
		return true
	}
	return false
}
