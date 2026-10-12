package widget

import (
	"testing"

	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/style"
	"PaintedSkin/theme"
)

// P1：控件语义样式入主题。
//
// 此前 17 个 widget.Theme 字段里只有 7 个被生产代码读取，10 个是死的；
// FocusColor / DisabledColor 完全没接线（焦点/禁用硬编码 AttrReverse /
// WithDim），theme.Theme 的 HoverBG / SelectionBG / ScrollbarThumbColor
// 被 ThemeFrom 直接丢掉。本文件把"每个字段都真的影响绘制"钉成用例。

// onlyTheme 返回一个"只设置了给定字段"的主题：字段之间互不干扰，
// 于是"该字段是否被读"可以靠"标记色是否出现在帧里"判定。
//
// 注意不能用"每个字段都给一个颜色"的探测主题：applyAll 的语义是
// "保留基础样式已设的颜色，只补未设的"，多个字段同时设色会互相屏蔽。
func onlyTheme(set func(*theme.Theme)) theme.Theme {
	th := theme.Theme{} // 其余字段全零 → 走内置兜底
	set(&th)
	return th
}

// mark 是本文件统一用的标记色。
var mark = style.ANSIColor(199)

// mkFg 返回一个"前景色 = mark"的样式字段。
func mkFg() style.Style { return style.Style{}.WithForeground(mark) }

// mkBg 返回一个"背景色 = mark"的样式字段。
func mkBg() style.Style { return style.Style{}.WithBackground(mark) }

// bufHasFG 报告帧内是否出现指定前景色。
func bufHasFG(buf *screen.Buffer, c style.Color) bool {
	for y := 0; y < buf.Height(); y++ {
		for x := 0; x < buf.Width(); x++ {
			cell, _ := buf.GetCell(x, y)
			if cell.Style.FG == c {
				return true
			}
		}
	}
	return false
}

// bufHasBG 报告帧内是否出现指定背景色。
func bufHasBG(buf *screen.Buffer, c style.Color) bool {
	for y := 0; y < buf.Height(); y++ {
		for x := 0; x < buf.Width(); x++ {
			cell, _ := buf.GetCell(x, y)
			if cell.Style.BG == c {
				return true
			}
		}
	}
	return false
}

// 20 个主题字段全部被接线：逐个只设一个字段，断言标记色出现在帧里。
func TestEveryThemeFieldIsWired(t *testing.T) {
	t.Run("TableHeader", func(t *testing.T) {
		tb := NewTable()
		tb.SetColumns([]TableColumn{{Title: "H", Width: 4}})
		tb.SetRows([][]string{{"a"}})
		tb.SetTheme(ThemeFrom(onlyTheme(func(th *theme.Theme) { th.TableHeader = mkFg() })))
		buf := paintBounds(tb, geom.Rect{X: 0, Y: 0, W: 8, H: 3})
		if !bufHasFG(buf, mark) {
			t.Error("TableHeader not wired")
		}
	})

	t.Run("DialogMask", func(t *testing.T) {
		d := NewDialog("t", "m", nil, nil)
		d.SetDim(true)
		d.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 8})
		d.SetTheme(ThemeFrom(onlyTheme(func(th *theme.Theme) { th.DialogMask = mkFg() })))
		buf := screen.NewBuffer(20, 8)
		d.Paint(buf)
		if !bufHasFG(buf, mark) {
			t.Error("DialogMask not wired")
		}
	})

	t.Run("ScrollbarThumbAndTrack", func(t *testing.T) {
		sb := NewScrollbar()
		sb.SetTotal(100)
		sb.SetView(10)
		sb.SetTheme(ThemeFrom(onlyTheme(func(th *theme.Theme) {
			th.ScrollbarThumb = mkFg()
			th.ScrollbarTrack = style.Style{}.WithForeground(style.ANSIColor(198))
		})))
		buf := paintBounds(sb, geom.Rect{X: 0, Y: 0, W: 1, H: 10})
		if !bufHasFG(buf, mark) {
			t.Error("ScrollbarThumb not wired")
		}
		if !bufHasFG(buf, style.ANSIColor(198)) {
			t.Error("ScrollbarTrack not wired")
		}
	})

	t.Run("Separator", func(t *testing.T) {
		sep := NewSeparator()
		sep.SetTheme(ThemeFrom(onlyTheme(func(th *theme.Theme) { th.Separator = mkFg() })))
		buf := paintBounds(sep, geom.Rect{X: 0, Y: 0, W: 4, H: 1})
		if !bufHasFG(buf, mark) {
			t.Error("Separator not wired")
		}
	})

	t.Run("StatusBar", func(t *testing.T) {
		s := NewStatusBar()
		s.SetLeft("l")
		s.SetTheme(ThemeFrom(onlyTheme(func(th *theme.Theme) { th.StatusBar = mkFg() })))
		buf := paintBounds(s, geom.Rect{X: 0, Y: 0, W: 10, H: 1})
		if !bufHasFG(buf, mark) {
			t.Error("StatusBar not wired")
		}
	})

	t.Run("TextInputNormal", func(t *testing.T) {
		ti := NewTextInput()
		ti.SetTheme(ThemeFrom(onlyTheme(func(th *theme.Theme) { th.TextInputNormal = mkFg() })))
		buf := paintBounds(ti, geom.Rect{X: 0, Y: 0, W: 10, H: 1})
		if !bufHasFG(buf, mark) {
			t.Error("TextInputNormal not wired")
		}
	})

	t.Run("TextInputFocus", func(t *testing.T) {
		ti := NewTextInput()
		ti.SetTheme(ThemeFrom(onlyTheme(func(th *theme.Theme) { th.TextInputFocus = mkFg() })))
		ti.OnFocus()
		buf := paintBounds(ti, geom.Rect{X: 0, Y: 0, W: 10, H: 1})
		if !bufHasFG(buf, mark) {
			t.Error("TextInputFocus not wired")
		}
	})

	t.Run("BorderColor", func(t *testing.T) {
		p := NewPanel("t")
		p.SetTheme(ThemeFrom(onlyTheme(func(th *theme.Theme) { th.BorderColor = mark })))
		buf := paintBounds(p, geom.Rect{X: 0, Y: 0, W: 8, H: 4})
		if !bufHasFG(buf, mark) {
			t.Error("BorderColor not wired")
		}
	})

	t.Run("FocusColor", func(t *testing.T) {
		cb := NewCheckbox("x")
		cb.SetTheme(ThemeFrom(onlyTheme(func(th *theme.Theme) { th.FocusColor = mark })))
		cb.OnFocus()
		buf := paintBounds(cb, geom.Rect{X: 0, Y: 0, W: 8, H: 1})
		if !bufHasFG(buf, mark) {
			t.Error("FocusColor not wired")
		}
	})

	t.Run("DisabledColor", func(t *testing.T) {
		r := NewRadio("x")
		r.SetTheme(ThemeFrom(onlyTheme(func(th *theme.Theme) { th.DisabledColor = mark })))
		r.SetDisabled(true)
		buf := paintBounds(r, geom.Rect{X: 0, Y: 0, W: 8, H: 1})
		if !bufHasFG(buf, mark) {
			t.Error("DisabledColor not wired")
		}
	})

	t.Run("HoverBG", func(t *testing.T) {
		r := NewRadio("x")
		r.SetTheme(ThemeFrom(onlyTheme(func(th *theme.Theme) { th.HoverBG = mark })))
		r.Handle(eventMouseEnter())
		buf := paintBounds(r, geom.Rect{X: 0, Y: 0, W: 8, H: 1})
		if !bufHasBG(buf, mark) {
			t.Error("HoverBG not wired")
		}
	})

	t.Run("SelectionBG", func(t *testing.T) {
		cb := NewCheckbox("x")
		cb.SetChecked(true)
		cb.SetTheme(ThemeFrom(onlyTheme(func(th *theme.Theme) { th.SelectionBG = mark })))
		buf := paintBounds(cb, geom.Rect{X: 0, Y: 0, W: 8, H: 1})
		if !bufHasBG(buf, mark) {
			t.Error("SelectionBG not wired")
		}
	})

	t.Run("ScrollbarThumbColor", func(t *testing.T) {
		// ScrollbarThumb 未设时回落到 ScrollbarThumbColor。
		sb := NewScrollbar()
		sb.SetTotal(100)
		sb.SetView(10)
		sb.SetTheme(ThemeFrom(onlyTheme(func(th *theme.Theme) { th.ScrollbarThumbColor = mark })))
		buf := paintBounds(sb, geom.Rect{X: 0, Y: 0, W: 1, H: 10})
		if !bufHasFG(buf, mark) {
			t.Error("ScrollbarThumbColor not wired as ScrollbarThumb fallback")
		}
	})

	t.Run("ListSelected", func(t *testing.T) {
		l := NewList()
		l.SetItems([]string{"a"})
		l.SetTheme(ThemeFrom(onlyTheme(func(th *theme.Theme) { th.ListSelected = mkFg() })))
		l.Layout(geom.Rect{X: 0, Y: 0, W: 8, H: 1})
		l.SetSelected(0)
		l.OnFocus()
		buf := paintBounds(l, geom.Rect{X: 0, Y: 0, W: 8, H: 1})
		if !bufHasFG(buf, mark) {
			t.Error("ListSelected not wired")
		}
	})

	t.Run("ButtonStates", func(t *testing.T) {
		cases := []struct {
			name  string
			field string
			set   func(*Button)
		}{
			{"ButtonNormal", "ButtonNormal", func(b *Button) {}},
			{"ButtonHover", "ButtonHover", func(b *Button) { b.Handle(eventMouseEnter()) }},
			{"ButtonPressed", "ButtonPressed", func(b *Button) { b.Handle(pressAt(1, 0)) }},
			{"ButtonFocused", "ButtonFocused", func(b *Button) { b.OnFocus() }},
			{"ButtonDisabled", "ButtonDisabled", func(b *Button) { b.SetDisabled(true) }},
		}
		for _, tc := range cases {
			b := NewButton("x")
			b.SetTheme(ThemeFrom(onlyTheme(func(th *theme.Theme) {
				switch tc.field {
				case "ButtonNormal":
					th.ButtonNormal = mkFg()
				case "ButtonHover":
					th.ButtonHover = mkFg()
				case "ButtonPressed":
					th.ButtonPressed = mkFg()
				case "ButtonFocused":
					th.ButtonFocused = mkFg()
				case "ButtonDisabled":
					th.ButtonDisabled = mkFg()
				}
			})))
			b.Layout(geom.Rect{X: 0, Y: 0, W: 8, H: 1})
			tc.set(b)
			buf := screen.NewBuffer(8, 1)
			b.Paint(buf)
			if !bufHasFG(buf, mark) {
				t.Errorf("%s not wired", tc.name)
			}
		}
	})
}

// 无主题时的内置兜底仍然是"反色 / 暗淡"（保证任何配置下状态可区分）。
func TestSemanticFallbackWithoutTheme(t *testing.T) {
	cb := NewCheckbox("x")
	cb.OnFocus()
	buf := paintBounds(cb, geom.Rect{X: 0, Y: 0, W: 8, H: 1})
	if c, _ := buf.GetCell(0, 0); !c.Style.Reverse {
		t.Errorf("no theme: focus must fall back to reverse, got %+v", c.Style)
	}
	r := NewRadio("x")
	r.SetDisabled(true)
	buf2 := paintBounds(r, geom.Rect{X: 0, Y: 0, W: 8, H: 1})
	if c, _ := buf2.GetCell(0, 0); !c.Style.Dim {
		t.Errorf("no theme: disabled must fall back to dim, got %+v", c.Style)
	}
}

// 主题只提供语义色时不得引入文本属性（换肤只改颜色，不改字形表现）。
func TestSemanticStylesDoNotForceAttrs(t *testing.T) {
	th := theme.Dark()
	cb := NewCheckbox("x")
	cb.SetTheme(ThemeFrom(th))
	cb.OnFocus()
	buf := paintBounds(cb, geom.Rect{X: 0, Y: 0, W: 8, H: 1})
	c, _ := buf.GetCell(0, 0)
	if c.Style.Reverse {
		t.Error("themed focus must not fall back to reverse")
	}
}

// 控件自身 Style 仍然优先于主题语义（docs/08：控件优先）。
func TestOwnStyleBeatsSemanticTheme(t *testing.T) {
	cb := NewCheckbox("x")
	cb.SetTheme(ThemeFrom(onlyTheme(func(th *theme.Theme) { th.FocusColor = mark })))
	cb.SetStyle(style.Style{}.WithForeground(style.Red))
	cb.OnFocus()
	buf := paintBounds(cb, geom.Rect{X: 0, Y: 0, W: 8, H: 1})
	if c, _ := buf.GetCell(0, 0); c.Style.FG != style.Red {
		t.Errorf("own style must win: got %v want red", c.Style.FG)
	}
}

// 语义色只落在语义位置上：未聚焦的选中行不得被涂上焦点色。
func TestSemanticColorsDoNotLeak(t *testing.T) {
	l := NewList()
	l.SetItems([]string{"a", "b"})
	l.SetTheme(ThemeFrom(onlyTheme(func(th *theme.Theme) { th.FocusColor = mark })))
	l.Layout(geom.Rect{X: 0, Y: 0, W: 8, H: 2})
	l.SetSelected(0) // 未聚焦：只有"选中"，不是"焦点"
	buf := paintBounds(l, geom.Rect{X: 0, Y: 0, W: 8, H: 2})
	if bufHasFG(buf, mark) {
		t.Error("focus color must not leak to an unfocused list")
	}
}

// theme.Theme 的三个基础背景色必须穿过 ThemeFrom 传到 widget.Theme
// （此前被直接丢弃，属于端到端死字段）。
func TestThemeFromCarriesBaseColors(t *testing.T) {
	th := theme.Dark()
	th.HoverBG = style.ANSIColor(31)
	th.SelectionBG = style.ANSIColor(32)
	th.ScrollbarThumbColor = style.ANSIColor(33)
	th.FocusColor = style.ANSIColor(34)
	th.DisabledColor = style.ANSIColor(35)
	w := ThemeFrom(th)
	if w.HoverBG != th.HoverBG {
		t.Errorf("HoverBG = %v want %v", w.HoverBG, th.HoverBG)
	}
	if w.SelectionBG != th.SelectionBG {
		t.Errorf("SelectionBG = %v want %v", w.SelectionBG, th.SelectionBG)
	}
	if w.ScrollbarThumbColor != th.ScrollbarThumbColor {
		t.Errorf("ScrollbarThumbColor = %v want %v", w.ScrollbarThumbColor, th.ScrollbarThumbColor)
	}
	if w.FocusColor != th.FocusColor {
		t.Errorf("FocusColor = %v want %v", w.FocusColor, th.FocusColor)
	}
	if w.DisabledColor != th.DisabledColor {
		t.Errorf("DisabledColor = %v want %v", w.DisabledColor, th.DisabledColor)
	}
}

// 语义辅助函数是纯函数式的：不得改动控件自身的 Style 字段。
func TestSemanticHelpersDoNotMutate(t *testing.T) {
	cb := NewCheckbox("x")
	before := cb.Style
	cb.focusStyle(cb.Style)
	cb.disabledStyle(cb.Style)
	cb.hoverStyle(cb.Style)
	cb.selectedStyle(cb.Style)
	if cb.Style != before {
		t.Errorf("semantic helpers mutated Style: %+v -> %+v", before, cb.Style)
	}
}

// List 选中行用 SelectionBG 兜底（当 ListSelected 未设时）。
func TestListSelectedFallsBackToSelectionBG(t *testing.T) {
	th := theme.Dark()
	th.ListSelected = style.Style{} // 未设
	l := NewList()
	l.SetItems([]string{"a"})
	l.SetTheme(ThemeFrom(th))
	l.Layout(geom.Rect{X: 0, Y: 0, W: 8, H: 1})
	l.SetSelected(0)
	l.OnFocus()
	buf := paintBounds(l, geom.Rect{X: 0, Y: 0, W: 8, H: 1})
	if !bufHasBG(buf, th.SelectionBG) && !bufHasFG(buf, th.FocusColor) {
		t.Error("selected row must fall back to a themed highlight")
	}
}
