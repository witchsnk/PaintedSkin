package widget

import (
	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/style"
	"PaintedSkin/text"
	"PaintedSkin/widget/layout"
)

// TextInput 是单行文本输入（docs/08）：光标移动/词跳、编辑、选区、
// 横向滚动（光标始终可见）、占位符与长度上限；硬件光标定位由宿主负责。
type TextInput struct {
	Base
	runes     []rune
	cursor    int
	selStart  int // -1 表示无选区
	selEnd    int
	viewOff   int
	placehold string
	maxLength int
	pressed   bool
	border    BorderStyle
	onChange  func(text string)
	onSubmit  func(text string)
	onConfirm func(text string)
}

// NewTextInput 创建文本输入。
func NewTextInput() *TextInput {
	t := &TextInput{selStart: -1, selEnd: -1, border: BorderHeavy}
	t.Base = NewBase()
	t.Bind(t)
	t.Focusable = true
	return t
}

// Value 返回当前文本。
func (t *TextInput) Value() string { return string(t.runes) }

// Text 返回当前文本（Value 的别名）。
func (t *TextInput) Text() string { return t.Value() }

// SetText 设置文本（受 MaxLength 约束，光标移到末尾）。
func (t *TextInput) SetText(s string) {
	t.setText(s, true)
}

func (t *TextInput) setText(s string, resetCursor bool) {
	rs := []rune(s)
	if t.maxLength > 0 && len(rs) > t.maxLength {
		rs = rs[:t.maxLength]
	}
	changed := string(rs) != t.Value()
	t.runes = rs
	if resetCursor {
		t.cursor = len(rs)
		t.selStart, t.selEnd = -1, -1
		t.viewOff = 0
	}
	t.Invalidate()
	if changed && t.onChange != nil {
		t.onChange(t.Value())
	}
}

// SetPlaceholder 设置占位提示（值为空时显示）。
func (t *TextInput) SetPlaceholder(s string) {
	t.placehold = s
	t.Invalidate()
}

// Placeholder 返回占位提示。
func (t *TextInput) Placeholder() string { return t.placehold }

// SetBorder 设置边框样式（枚举档；默认 BorderHeavy，与改动前视觉一致）。
func (t *TextInput) SetBorder(s BorderStyle) {
	t.border = s
	t.Invalidate()
}

// Border 返回边框样式。
func (t *TextInput) Border() BorderStyle { return t.border }

// SetMaxLength 设置最大长度（<=0 表示不限）。
func (t *TextInput) SetMaxLength(n int) {
	if n < 0 {
		n = 0
	}
	t.maxLength = n
	if n > 0 && len(t.runes) > n {
		t.runes = t.runes[:n]
		if t.cursor > n {
			t.cursor = n
		}
	}
	t.Invalidate()
}

// MaxLength 返回最大长度。
func (t *TextInput) MaxLength() int { return t.maxLength }

// Cursor 返回光标位置（rune 索引）。
func (t *TextInput) Cursor() int { return t.cursor }

// Selection 返回选区（起, 止）；无选区返回 (-1,-1)。
func (t *TextInput) Selection() (int, int) { return t.selStart, t.selEnd }

// ViewOffset 返回横向滚动偏移（首个可见 rune 索引）。
func (t *TextInput) ViewOffset() int { return t.viewOff }

// CursorScreenPos 返回插入位的屏幕坐标（宿主据此定位硬件光标）。
// 未布局、无内容区域或未聚焦时 ok=false（宿主转为隐藏光标）。
func (t *TextInput) CursorScreenPos() (x, y int, ok bool) {
	b := t.Bounds
	if b.Empty() || b.W < 3 {
		return 0, 0, false
	}
	if !t.Focused() {
		return 0, 0, false
	}
	off := widthOf(t.runes, t.viewOff, t.cursor)
	return b.X + 1 + off, b.Y, true
}

// SetOnChange 设置编辑回调。
func (t *TextInput) SetOnChange(fn func(text string)) { t.onChange = fn }

// SetOnSubmit 设置回车提交回调（OnConfirm 为兼容别名，同时设置时都调用）。
func (t *TextInput) SetOnSubmit(fn func(text string)) { t.onSubmit = fn }

// SetOnConfirm 设置回车确认回调（兼容别名）。
func (t *TextInput) SetOnConfirm(fn func(text string)) { t.onConfirm = fn }

// SetDisabled 开关禁用态；禁用时吞掉全部事件。
func (t *TextInput) SetDisabled(v bool) {
	t.Enabled = !v
	t.Invalidate()
}

// Disabled 报告禁用态。
func (t *TextInput) Disabled() bool { return !t.Enabled }

// OnFocus 覆盖：聚焦时光标移到末尾并清选区。
func (t *TextInput) OnFocus() {
	t.Base.OnFocus()
	t.cursor = len(t.runes)
	t.selStart, t.selEnd = -1, -1
	t.viewOff = 0
}

// OnBlur 覆盖：失焦清选区与按下态。
func (t *TextInput) OnBlur() {
	t.Base.OnBlur()
	t.pressed = false
	t.selStart, t.selEnd = -1, -1
}

// Measure 固定高 1 行；宽按 MaxW（无上界时回落 17）。
func (t *TextInput) Measure(c layout.Constraints) geom.Size {
	w := c.MaxW
	if w < 0 {
		w = textInputPrefW
	}
	return geom.Size{W: clampDim(w, c.MinW), H: 1}
}

// Layout 记录边界。
func (t *TextInput) Layout(r geom.Rect) {
	t.Bounds = r
	t.syncView()
}

func (t *TextInput) viewWidth() int {
	w := t.Bounds.W - 2
	if w < 0 {
		return 0
	}
	return w
}

// widthOf 返回 runes[i:j] 的显示宽。
func widthOf(rs []rune, i, j int) int {
	w := 0
	for k := i; k < j && k < len(rs); k++ {
		w += text.RuneWidth(rs[k])
	}
	return w
}

// syncView 保证光标可见（横向滚动，docs/08）。
func (t *TextInput) syncView() {
	vw := t.viewWidth()
	if vw <= 0 {
		t.viewOff = 0
		return
	}
	if t.viewOff > t.cursor {
		t.viewOff = t.cursor
	}
	for widthOf(t.runes, t.viewOff, t.cursor) >= vw {
		t.viewOff++
		if t.viewOff >= t.cursor {
			break
		}
	}
	if t.viewOff < 0 || t.viewOff > len(t.runes) {
		t.viewOff = 0
	}
}

// Paint 绘制边框、占位符/文本、选区与聚焦光标。
func (t *TextInput) Paint(c screen.Canvas) {
	b := t.Bounds
	if b.W < minFrameW || b.H < minFrameH {
		return
	}
	// 常态/焦点走主题 TextInputNormal/TextInputFocus；禁用走 DisabledColor。
	st := applyAll(t.Style, t.themeOrZero().TextInputNormal, nil)
	if themed := t.themeOrZero().TextInputFocus; themed != (style.Style{}) {
		if t.Focused() || t.pressed {
			st = applyAll(t.Style, themed, nil)
		}
	}
	switch {
	case !t.EffectivelyEnabled():
		st = t.disabledStyle(st)
	case t.Focused() || t.pressed:
		st = t.focusStyle(st)
	}
	paintBackground(c, b, st)
	c.Box(b, borderSpec(t.border), st)
	textX := b.X + 1
	textY := b.Y

	if len(t.runes) == 0 {
		if t.placehold != "" {
			c.Text(textX, textY, t.placehold, t.Style.WithDim(true))
		}
		t.drawFocusRing(c, b)
		return
	}
	vw := t.viewWidth()
	vis := t.viewOff
	// 选区行覆盖（先铺底色）
	if t.selStart >= 0 && t.selEnd > t.selStart {
		s, e := t.selStart, t.selEnd
		if e > len(t.runes) {
			e = len(t.runes)
		}
		x := textX + widthOf(t.runes, vis, s)
		w := widthOf(t.runes, s, e)
		if w > vw {
			w = vw
		}
		if w > 0 {
			c.Fill(geom.Rect{X: x, Y: textY, W: w, H: 1}, ' ', t.selectedStyle(st))
		}
	}
	// 可见文本
	x := textX
	for i := vis; i < len(t.runes); i++ {
		rw := text.RuneWidth(t.runes[i])
		if x-textX+rw > vw {
			break
		}
		c.SetCell(x, textY, t.runes[i], st)
		x += rw
	}
	// 聚焦光标
	if t.Focused() {
		cx := textX + widthOf(t.runes, vis, t.cursor)
		if cx <= b.Right()-2 {
			c.SetCell(cx, textY, '|', st)
		}
	}
	// 焦点环画在**内容之后**（Textual §1.5）：只换样式、不动字符、不占布局。
	t.drawFocusRing(c, b)
}

// posFromX 把横坐标换算为插入位 rune 索引。
// 边框内首列对应"字符 0 之前"，即列 x 的插入位为 x - (Bounds.X+1)。
func (t *TextInput) posFromX(x int) int {
	rel := x - t.Bounds.X - 1
	if rel <= 0 {
		return t.viewOff
	}
	w := 0
	for i := t.viewOff; i < len(t.runes); i++ {
		rw := text.RuneWidth(t.runes[i])
		if w >= rel {
			return i
		}
		w += rw
	}
	return len(t.runes)
}

// hasSelection 报告是否有有效选区。
func (t *TextInput) hasSelection() bool {
	return t.selStart >= 0 && t.selEnd > t.selStart
}

// deleteSelection 删除选区内容（若有）。
func (t *TextInput) deleteSelection() {
	if !t.hasSelection() {
		return
	}
	s, e := t.selStart, t.selEnd
	if e > len(t.runes) {
		e = len(t.runes)
	}
	t.runes = append(t.runes[:s:s], t.runes[e:]...)
	t.cursor = s
	t.selStart, t.selEnd = -1, -1
}

func (t *TextInput) changed() {
	t.Invalidate()
	t.syncView()
	if t.onChange != nil {
		t.onChange(t.Value())
	}
}

// insert 插入字符串（受 MaxLength 约束），先删选区。
func (t *TextInput) insert(s string) {
	t.deleteSelection()
	rs := []rune(s)
	if t.maxLength > 0 {
		room := t.maxLength - len(t.runes)
		if room <= 0 {
			return
		}
		if len(rs) > room {
			rs = rs[:room]
		}
	}
	head := append([]rune{}, t.runes[:t.cursor]...)
	tail := append([]rune{}, t.runes[t.cursor:]...)
	t.runes = append(append(head, rs...), tail...)
	t.cursor += len(rs)
	t.changed()
}

func (t *TextInput) deleteBefore() {
	if t.hasSelection() {
		t.deleteSelection()
		t.changed()
		return
	}
	if t.cursor <= 0 {
		return
	}
	t.runes = append(t.runes[:t.cursor-1:t.cursor-1], t.runes[t.cursor:]...)
	t.cursor--
	t.changed()
}

func (t *TextInput) deleteAfter() {
	if t.hasSelection() {
		t.deleteSelection()
		t.changed()
		return
	}
	if t.cursor >= len(t.runes) {
		return
	}
	t.runes = append(t.runes[:t.cursor:t.cursor], t.runes[t.cursor+1:]...)
	t.changed()
}

func (t *TextInput) moveCursor(to int) {
	if to < 0 {
		to = 0
	}
	if to > len(t.runes) {
		to = len(t.runes)
	}
	t.cursor = to
	t.selStart, t.selEnd = -1, -1
	t.syncView()
}

// wordLeft 返回左侧词首索引。
func (t *TextInput) wordLeft() int {
	i := t.cursor
	for i > 0 && isWordRune(t.runes[i-1]) {
		i--
	}
	for i > 0 && !isWordRune(t.runes[i-1]) {
		i--
	}
	return i
}

// wordRight 返回右侧词尾索引。
func (t *TextInput) wordRight() int {
	i := t.cursor
	for i < len(t.runes) && !isWordRune(t.runes[i]) {
		i++
	}
	for i < len(t.runes) && isWordRune(t.runes[i]) {
		i++
	}
	return i
}

func isWordRune(r rune) bool {
	return r == '_' || (r >= '0' && r <= '9') ||
		(r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || r > 0x7F
}

// Handle 实现编辑、光标、选区与粘贴；禁用全吞。
func (t *TextInput) Handle(e event.Event) bool {
	if !t.EffectivelyEnabled() {
		return true
	}
	switch ev := e.(type) {
	case event.Paste:
		if t.Focused() {
			t.insert(ev.Text)
			return true
		}
	case event.Key:
		if !ev.Press || !t.Focused() {
			return false
		}
		return t.handleKey(ev)
	case event.Mouse:
		return t.handleMouse(ev)
	}
	return false
}

func (t *TextInput) handleKey(k event.Key) bool {
	// Ctrl+A 全选（解码为 KeyRune 'a'/0x01 + ModCtrl）
	if k.Code == event.KeyRune && k.Mod&event.ModCtrl != 0 {
		if k.Rune == 0x01 || k.Rune == 'a' || k.Rune == 'A' {
			if len(t.runes) > 0 {
				t.selStart, t.selEnd = 0, len(t.runes)
				t.cursor = len(t.runes)
				t.syncView()
			}
			return true
		}
	}
	switch k.Code {
	case event.KeyEnter:
		if t.onSubmit != nil {
			t.onSubmit(t.Value())
		}
		if t.onConfirm != nil {
			t.onConfirm(t.Value())
		}
		return true
	case event.KeyBackspace:
		t.deleteBefore()
		return true
	case event.KeyDelete:
		t.deleteAfter()
		return true
	case event.KeyLeft:
		if k.Mod&event.ModCtrl != 0 {
			t.moveCursor(t.wordLeft())
		} else {
			t.moveCursor(t.cursor - 1)
		}
		return true
	case event.KeyRight:
		if k.Mod&event.ModCtrl != 0 {
			t.moveCursor(t.wordRight())
		} else {
			t.moveCursor(t.cursor + 1)
		}
		return true
	case event.KeyHome:
		t.moveCursor(0)
		return true
	case event.KeyEnd:
		t.moveCursor(len(t.runes))
		return true
	case event.KeyRune:
		if k.Mod&(event.ModCtrl|event.ModAlt) != 0 {
			return false
		}
		if k.Rune >= 0x20 {
			t.insert(string(k.Rune))
			return true
		}
	}
	return false
}

func (t *TextInput) handleMouse(m event.Mouse) bool {
	inside := t.Bounds.Contains(geom.Point{X: m.X, Y: m.Y})
	switch m.Kind {
	case event.MouseEnter:
		return false
	case event.MousePress:
		if m.Button == event.ButtonLeft && inside {
			t.pressed = true
			t.cursor = t.posFromX(m.X)
			t.selStart, t.selEnd = -1, -1
			t.Invalidate()
			if h := t.HostOf(); h != nil {
				h.Capture(t)
			}
			return true
		}
	case event.MouseMove:
		if t.pressed {
			pos := t.posFromX(m.X)
			anchor := t.cursor
			if pos < anchor {
				t.selStart, t.selEnd = pos, anchor
			} else {
				t.selStart, t.selEnd = anchor, pos
			}
			t.Invalidate()
			return true
		}
	case event.MouseRelease:
		if t.pressed {
			t.pressed = false
			t.Invalidate()
			return true
		}
	}
	return false
}
