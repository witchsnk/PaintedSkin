package theme

import (
	"testing"

	"PaintedSkin/style"
)

// Default 必须给出全部规格项，且都能被 style.Style 承载。
func TestDefaultComplete(t *testing.T) {
	th := Default()
	if th.FocusColor.Kind == style.ColorDefault {
		t.Error("FocusColor must be set")
	}
	if th.BorderColor.Kind == style.ColorDefault {
		t.Error("BorderColor must be set")
	}
	if th.DisabledColor.Kind == style.ColorDefault {
		t.Error("DisabledColor must be set")
	}
	if th.HoverBG.Kind == style.ColorDefault {
		t.Error("HoverBG must be set")
	}
	if th.ScrollbarThumbColor.Kind == style.ColorDefault {
		t.Error("ScrollbarThumbColor must be set")
	}
}

// 按钮四态样式齐全。
func TestButtonStates(t *testing.T) {
	th := Default()
	// normal 允许是零值（终端默认色）；其余三态必须有明确视觉
	for name, st := range map[string]style.Style{
		"hover":   th.ButtonHover,
		"pressed": th.ButtonPressed,
		"focused": th.ButtonFocused,
	} {
		if st == (style.Style{}) {
			t.Errorf("button %s style is zero", name)
		}
	}
	// hover 与 focused 必须视觉可区分
	if th.ButtonHover.Equal(th.ButtonFocused) {
		t.Error("hover and focused must differ")
	}
	if th.ButtonPressed.Equal(th.ButtonHover) {
		t.Error("pressed and hover must differ")
	}
}

// 列表选中行样式反色。
func TestListSelected(t *testing.T) {
	th := Default()
	if !th.ListSelected.Reverse {
		t.Error("selected row must be reverse")
	}
}

// Dialog 遮罩是暗淡的。
func TestDialogMask(t *testing.T) {
	th := Default()
	if !th.DialogMask.Dim {
		t.Error("dialog mask must be dim")
	}
}

// Apply 回落：控件未设 Style 时用主题，否则保留控件自身样式。
func TestApplyFallback(t *testing.T) {
	th := Default()
	// 控件 Style 为零值 → 回落主题
	got := th.Apply(th.ButtonNormal, style.Style{})
	if got != th.ButtonNormal {
		t.Errorf("zero style must fall back: %+v", got)
	}
	// 控件已设 Style → 保留
	custom := style.Style{}.WithBold(true)
	if got := th.Apply(th.ButtonNormal, custom); got != custom {
		t.Errorf("custom style must be kept: %+v", got)
	}
}

// Dark/Light 主题都可用且主色不同。
func TestThemes(t *testing.T) {
	d, l := Dark(), Light()
	if d.FocusColor == l.FocusColor {
		t.Error("dark and light focus colors must differ")
	}
	if d.ButtonFocused.Equal(l.ButtonFocused) {
		t.Error("dark and light focused styles must differ")
	}
	if d.ListSelected.Equal(l.ListSelected) {
		t.Error("dark and light selected-row styles must differ")
	}
}

// SetDefault / Current 可替换全局主题。
func TestSetDefault(t *testing.T) {
	orig := Current()
	defer SetDefault(orig)
	custom := Dark()
	custom.FocusColor = style.Red
	SetDefault(custom)
	if Current().FocusColor != style.Red {
		t.Errorf("SetDefault must take effect: %+v", Current().FocusColor)
	}
}
