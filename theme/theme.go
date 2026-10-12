// Package theme 提供控件的默认配色与样式（docs/08 主题）。
//
// 控件优先使用自身 Style，未设（零值）时回落主题，保证换肤只需替换
// 一次 SetDefault 即可全树生效。
package theme

import (
	"sync"

	"PaintedSkin/style"
)

// Theme 汇总框架用到的颜色与控件样式。
type Theme struct {
	// 基础色
	FocusColor          style.Color
	BorderColor         style.Color
	DisabledColor       style.Color
	HoverBG             style.Color
	SelectionBG         style.Color
	ScrollbarThumbColor style.Color

	// 控件样式
	ButtonNormal   style.Style
	ButtonHover    style.Style
	ButtonPressed  style.Style
	ButtonFocused  style.Style
	ButtonDisabled style.Style

	ListSelected    style.Style
	TableHeader     style.Style
	DialogMask      style.Style
	ScrollbarThumb  style.Style
	ScrollbarTrack  style.Style
	Separator       style.Style
	StatusBar       style.Style
	TextInputNormal style.Style
	TextInputFocus  style.Style
}

// Dark 返回深色终端主题（默认）。
func Dark() Theme {
	focus := style.BrightCyan
	return Theme{
		FocusColor:          focus,
		BorderColor:         style.BrightBlack,
		DisabledColor:       style.BrightBlack,
		HoverBG:             style.ANSIColor(238),
		SelectionBG:         style.ANSIColor(24),
		ScrollbarThumbColor: style.BrightBlack,

		ButtonNormal:   style.Style{},
		ButtonHover:    style.Style{}.WithBold(true),
		ButtonPressed:  style.Style{}.WithReverse(true),
		ButtonFocused:  style.Style{}.WithAttrs(style.AttrUnderline),
		ButtonDisabled: style.Style{}.WithDim(true),

		ListSelected:    style.Style{}.WithAttrs(style.AttrReverse),
		TableHeader:     style.Style{}.WithBold(true),
		DialogMask:      style.Style{}.WithDim(true),
		ScrollbarThumb:  style.Style{}.WithForeground(style.BrightBlack),
		ScrollbarTrack:  style.Style{}.WithDim(true),
		Separator:       style.Style{}.WithForeground(style.BrightBlack),
		StatusBar:       style.Style{}.WithForeground(style.BrightBlack),
		TextInputNormal: style.Style{},
		TextInputFocus:  style.Style{}.WithAttrs(style.AttrUnderline),
	}
}

// Light 返回浅色终端主题。
func Light() Theme {
	focus := style.Blue
	return Theme{
		FocusColor:          focus,
		BorderColor:         style.BrightBlack,
		DisabledColor:       style.BrightBlack,
		HoverBG:             style.ANSIColor(254),
		SelectionBG:         style.ANSIColor(153),
		ScrollbarThumbColor: style.BrightBlack,

		ButtonNormal:   style.Style{},
		ButtonHover:    style.Style{}.WithBold(true),
		ButtonPressed:  style.Style{}.WithAttrs(style.AttrReverse),
		ButtonFocused:  style.Style{}.WithForeground(style.Blue),
		ButtonDisabled: style.Style{}.WithDim(true),

		ListSelected:    style.Style{}.WithBackground(style.ANSIColor(153)),
		TableHeader:     style.Style{}.WithBold(true),
		DialogMask:      style.Style{}.WithDim(true),
		ScrollbarThumb:  style.Style{}.WithForeground(style.BrightBlack),
		ScrollbarTrack:  style.Style{}.WithDim(true),
		Separator:       style.Style{}.WithForeground(style.BrightBlack),
		StatusBar:       style.Style{}.WithForeground(style.BrightBlack),
		TextInputNormal: style.Style{},
		TextInputFocus:  style.Style{}.WithForeground(style.Blue),
	}
}

// Default 返回默认（深色）主题。
func Default() Theme { return Dark() }

var (
	mu      sync.RWMutex
	current = Dark()
)

// Current 返回当前全局主题。
func Current() Theme {
	mu.RLock()
	defer mu.RUnlock()
	return current
}

// SetDefault 替换全局主题（控件下一帧生效）。
func SetDefault(th Theme) {
	mu.Lock()
	defer mu.Unlock()
	current = th
}

// Apply 在控件未设自身 Style（零值）时回落 def，否则保留控件样式。
func (t Theme) Apply(fallback, own style.Style) style.Style {
	if own == (style.Style{}) {
		return fallback
	}
	return own
}
