package widget

import (
	"reflect"
	"testing"

	"PaintedSkin/style"
	"PaintedSkin/theme"
)

// ThemeFrom 是 23 个字段的手工搬运。加字段容易漏 —— `HoverBG` / `SelectionBG` /
// `ScrollbarThumbColor` 就这样死过一轮（ADR-020）。本文件用反射把"不许漏"钉住。

// theme.Theme 的每个字段都必须在 widget.Theme 里有同名字段，且被 ThemeFrom 搬运。
func TestThemeFromIsComplete(t *testing.T) {
	src := reflect.TypeOf(theme.Theme{})
	dst := reflect.TypeOf(Theme{})

	missing := 0
	for i := 0; i < src.NumField(); i++ {
		name := src.Field(i).Name
		f, ok := dst.FieldByName(name)
		if !ok {
			t.Errorf("widget.Theme 缺少字段 %q（theme.Theme 有）", name)
			missing++
			continue
		}
		if f.Type != src.Field(i).Type {
			t.Errorf("%q 类型不一致：theme=%v widget=%v", name, src.Field(i).Type, f.Type)
			missing++
		}
	}
	if missing == 0 {
		t.Logf("theme.Theme 的 %d 个字段全部有对应的 widget.Theme 字段", src.NumField())
	}
}

// 搬运的**正确性**：给 theme.Theme 每个字段填一个可区分的非零值，
// ThemeFrom 之后 widget.Theme 的同名字段必须逐个相等。
func TestThemeFromCarriesEveryValue(t *testing.T) {
	full := fillTheme(t)

	got := ThemeFrom(full)
	gotV := reflect.ValueOf(*got)
	wantV := reflect.ValueOf(full)

	for i := 0; i < wantV.NumField(); i++ {
		name := wantV.Type().Field(i).Name
		f := gotV.FieldByName(name)
		if !f.IsValid() {
			t.Errorf("widget.Theme 缺少字段 %q", name)
			continue
		}
		if !reflect.DeepEqual(f.Interface(), wantV.Field(i).Interface()) {
			t.Errorf("ThemeFrom 未搬运 %q：得到 %v，期望 %v",
				name, f.Interface(), wantV.Field(i).Interface())
		}
	}
}

// 反向：widget.Theme 多出来的字段（theme.Theme 没有）必须在测试里被显式登记，
// 否则无法判断它是"有意新增"还是"忘了搬运"。
func TestWidgetThemeExtraFieldsAreKnown(t *testing.T) {
	// 允许 widget.Theme 有 theme.Theme 没有的字段（纯 widget 侧语义），
	// 但必须在此登记，避免悄悄膨胀。
	known := map[string]bool{}
	src := reflect.TypeOf(theme.Theme{})
	dst := reflect.TypeOf(Theme{})
	for i := 0; i < dst.NumField(); i++ {
		name := dst.Field(i).Name
		if _, ok := src.FieldByName(name); !ok && !known[name] {
			t.Errorf("widget.Theme.%q 在 theme.Theme 中没有对应字段；"+
				"若为有意新增，请登记到 TestWidgetThemeExtraFieldsAreKnown", name)
		}
	}
}

// styleType / colorType 用于按类型分别填充。
var (
	styleType = reflect.TypeOf(style.Style{})
	colorType = reflect.TypeOf(style.Color{})
)

// fillTheme 给 theme.Theme 的每个字段填一个可区分的非零值：
// 每个字段用序号派生的颜色，因此任何一个字段漏搬运都会被 DeepEqual 抓到。
func fillTheme(t *testing.T) theme.Theme {
	t.Helper()
	th = theme.Theme{}
	v := reflect.ValueOf(&th).Elem()
	tt := v.Type()
	for i := 0; i < tt.NumField(); i++ {
		f := v.Field(i)
		name := tt.Field(i).Name
		col := style.Color{Kind: style.ColorANSI, N: uint8(i%8 + 1), R: uint8(i + 1), B: uint8(200 - i)}
		switch f.Type() {
		case styleType:
			f.Set(reflect.ValueOf(style.Style{}.
				WithForeground(col).
				WithBold(true)))
		case colorType:
			f.Set(reflect.ValueOf(col))
		default:
			t.Fatalf("theme.Theme.%s 的类型 %v 未被 fillTheme 覆盖", name, f.Type())
		}
	}
	return th
}

// th 是被 fillTheme 填充的目标。
var th theme.Theme
