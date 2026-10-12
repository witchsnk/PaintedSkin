package style

// Attr 是文本属性位掩码，供 SGR 生成与批量比较使用。
type Attr uint8

const (
	AttrBold Attr = 1 << iota
	AttrDim
	AttrItalic
	AttrUnderline
	AttrBlink
	AttrReverse
	AttrStrike
	AttrNone Attr = 0
)

// Style 描述一个单元格的完整视觉样式。
//
// 布尔属性字段（Bold/Dim/Italic/Underline/Blink/Reverse/Strike）是事实来源，
// 渲染时据此生成 SGR；Attrs 是由所有 With* 方法同步维护的位掩码镜像，
// 供需要掩码运算的场景（如比较、批量开关）使用。请勿直接写入字段，
// 统一通过 With* 方法构造样式。
//
// 所有 With* 方法返回新值，不修改接收者，便于样式组合与比较。
type Style struct {
	FG, BG Color

	Bold, Dim, Italic, Underline, Blink, Reverse, Strike bool

	// Attrs 是上述布尔属性的位掩码镜像（由 With* 同步）。
	Attrs Attr
}

// Default 是零值样式：终端默认色、无属性。
var Default = Style{}

// WithForeground 返回前景色为 fg 的新样式。
func (s Style) WithForeground(fg Color) Style { s.FG = fg; return s }

// WithBackground 返回背景色为 bg 的新样式。
func (s Style) WithBackground(bg Color) Style { s.BG = bg; return s }

// WithAttrs 用掩码整体替换布尔属性，并同步镜像。
func (s Style) WithAttrs(a Attr) Style {
	s.Bold = a&AttrBold != 0
	s.Dim = a&AttrDim != 0
	s.Italic = a&AttrItalic != 0
	s.Underline = a&AttrUnderline != 0
	s.Blink = a&AttrBlink != 0
	s.Reverse = a&AttrReverse != 0
	s.Strike = a&AttrStrike != 0
	s.Attrs = a
	return s
}

// WithBold 开关粗体。
func (s Style) WithBold(v bool) Style { s.Bold = v; return s.sync(AttrBold, v) }

// WithDim 开关暗淡。
func (s Style) WithDim(v bool) Style { s.Dim = v; return s.sync(AttrDim, v) }

// WithItalic 开关斜体。
func (s Style) WithItalic(v bool) Style { s.Italic = v; return s.sync(AttrItalic, v) }

// WithUnderline 开关下划线。
func (s Style) WithUnderline(v bool) Style { s.Underline = v; return s.sync(AttrUnderline, v) }

// WithBlink 开关闪烁。
func (s Style) WithBlink(v bool) Style { s.Blink = v; return s.sync(AttrBlink, v) }

// WithReverse 开关反色。
func (s Style) WithReverse(v bool) Style { s.Reverse = v; return s.sync(AttrReverse, v) }

// WithStrike 开关删除线。
func (s Style) WithStrike(v bool) Style { s.Strike = v; return s.sync(AttrStrike, v) }

// sync 更新镜像掩码并返回新样式。
func (s Style) sync(a Attr, v bool) Style {
	if v {
		s.Attrs |= a
	} else {
		s.Attrs &^= a
	}
	return s
}

// AttrMask 从事实来源的布尔字段计算当前属性掩码。
// 渲染层应使用本方法而非镜像字段，以容忍直接构造的结构体字面量。
func (s Style) AttrMask() Attr {
	var a Attr
	if s.Bold {
		a |= AttrBold
	}
	if s.Dim {
		a |= AttrDim
	}
	if s.Italic {
		a |= AttrItalic
	}
	if s.Underline {
		a |= AttrUnderline
	}
	if s.Blink {
		a |= AttrBlink
	}
	if s.Reverse {
		a |= AttrReverse
	}
	if s.Strike {
		a |= AttrStrike
	}
	return a
}

// Equal 比较两样式是否产生相同的视觉效果。
// 比较事实来源（颜色 + 布尔属性），忽略 Attrs 镜像，避免镜像不同步导致误判。
func (s Style) Equal(o Style) bool {
	return s.FG == o.FG && s.BG == o.BG &&
		s.Bold == o.Bold && s.Dim == o.Dim && s.Italic == o.Italic &&
		s.Underline == o.Underline && s.Blink == o.Blink &&
		s.Reverse == o.Reverse && s.Strike == o.Strike
}

// Swap 返回交换前景/背景的新样式（用于反色高亮）。
func (s Style) Swap() Style {
	s.FG, s.BG = s.BG, s.FG
	return s
}
