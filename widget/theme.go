package widget

import (
	"PaintedSkin/style"
	"PaintedSkin/theme"
)

// ThemeFrom 把 theme 包的 Theme 转成控件可用的形态。
// 转换函数放在 widget 而非 theme，是为了保持依赖方向 widget→theme 单向
// （theme 不 import widget；app 负责两者的桥接与全局换肤推送）。
func ThemeFrom(th theme.Theme) *Theme {
	return &Theme{
		ButtonNormal:    th.ButtonNormal,
		ButtonHover:     th.ButtonHover,
		ButtonPressed:   th.ButtonPressed,
		ButtonFocused:   th.ButtonFocused,
		ButtonDisabled:  th.ButtonDisabled,
		ListSelected:    th.ListSelected,
		TableHeader:     th.TableHeader,
		DialogMask:      th.DialogMask,
		ScrollbarThumb:  th.ScrollbarThumb,
		ScrollbarTrack:  th.ScrollbarTrack,
		Separator:       th.Separator,
		StatusBar:       th.StatusBar,
		TextInputNormal: th.TextInputNormal,
		TextInputFocus:  th.TextInputFocus,
		BorderColor:     th.BorderColor,
		// 以下五个此前被 ThemeFrom 直接丢弃，属于端到端死字段。
		FocusColor:          th.FocusColor,
		DisabledColor:       th.DisabledColor,
		HoverBG:             th.HoverBG,
		SelectionBG:         th.SelectionBG,
		ScrollbarThumbColor: th.ScrollbarThumbColor,
	}
}

// CurrentTheme 返回当前全局主题转换后的控件形态。
func CurrentTheme() *Theme { return ThemeFrom(theme.Current()) }

// ensure style 被引用（Theme 字段类型）。
var _ style.Style = style.Default

// applyAll 合并基础样式与主题样式：
// themed 为零值时返回 st；否则以 themed 为基底并保留 st 的颜色设置。
// mutate 非 nil 时在结果上追加变换（如强制 Dim）。
func applyAll(st, themed style.Style, mutate func(style.Style) style.Style) style.Style {
	out := st
	if themed != (style.Style{}) {
		if out.FG.Kind == style.ColorDefault {
			out.FG = themed.FG
		}
		if out.BG.Kind == style.ColorDefault {
			out.BG = themed.BG
		}
		out.Attrs = themed.Attrs
		out.Bold, out.Dim, out.Italic = themed.Bold, themed.Dim, themed.Italic
		out.Underline, out.Blink, out.Reverse, out.Strike =
			themed.Underline, themed.Blink, themed.Reverse, themed.Strike
	}
	if mutate != nil {
		out = mutate(out)
	}
	return out
}
