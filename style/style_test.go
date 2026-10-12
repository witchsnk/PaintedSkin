package style

import "testing"

func TestColorKinds(t *testing.T) {
	if DefaultColor.Kind != ColorDefault {
		t.Errorf("DefaultColor.Kind=%v want ColorDefault", DefaultColor.Kind)
	}
	c := RGB(0x11, 0x22, 0x33)
	if c.Kind != ColorRGB || c.R != 0x11 || c.G != 0x22 || c.B != 0x33 {
		t.Errorf("RGB() wrong: %+v", c)
	}
	if ANSIColor(3).Kind != ColorANSI || ANSIColor(3).N != 3 {
		t.Errorf("ANSIColor wrong: %+v", ANSIColor(3))
	}
}

func TestQuantizeRGBTo256(t *testing.T) {
	// 纯黑与纯白必须落在 16 色立方的确定端点上。
	if got := RGB(0, 0, 0).To256(); got != 16 {
		t.Errorf("black To256=%d want 16", got)
	}
	if got := RGB(255, 255, 255).To256(); got != 231 {
		t.Errorf("white To256=%d want 231", got)
	}
	// 中灰应落在灰阶带 232..255 内。
	got := RGB(128, 128, 128).To256()
	if got < 232 || got > 255 {
		t.Errorf("mid gray To256=%d want in [232,255]", got)
	}
	// 典型主题色：橙红应落在 16 色立方（16..231）。
	got = RGB(0xFF, 0x55, 0x55).To256()
	if got < 16 || got > 231 {
		t.Errorf("accent To256=%d want in [16,231]", got)
	}
}

func TestQuantizeRGBTo16(t *testing.T) {
	if got := RGB(0, 0, 0).To16(); got != 0 {
		t.Errorf("black To16=%d want 0", got)
	}
	if got := RGB(255, 255, 255).To16(); got != 15 {
		t.Errorf("white To16=%d want 15", got)
	}
	// 亮红应映射到亮红 9。
	got := RGB(255, 0, 0).To16()
	if got != 9 && got != 1 {
		t.Errorf("red To16=%d want 9 (or 1)", got)
	}
}

func TestStyleChain(t *testing.T) {
	s := Default.
		WithForeground(RGB(1, 2, 3)).
		WithBackground(ANSIColor(4)).
		WithBold(true).
		WithUnderline(true)

	if s.FG.Kind != ColorRGB || s.FG.R != 1 {
		t.Errorf("FG wrong: %+v", s.FG)
	}
	if s.BG.Kind != ColorANSI || s.BG.N != 4 {
		t.Errorf("BG wrong: %+v", s.BG)
	}
	if !s.Bold || !s.Underline || s.Reverse {
		t.Errorf("attrs wrong: %+v", s)
	}

	// 关闭属性必须得到新值而不影响原样式（不可变链式）。
	s2 := s.WithBold(false)
	if !s.Bold {
		t.Error("WithBold must not mutate receiver")
	}
	if s2.Bold {
		t.Error("WithBold(false) must clear bold")
	}
}

func TestAttrMask(t *testing.T) {
	s := Default.WithReverse(true).WithItalic(true).WithDim(true)
	want := AttrReverse | AttrItalic | AttrDim
	if s.Attrs != want {
		t.Errorf("Attrs=%b want %b", s.Attrs, want)
	}
	s2 := s.WithReverse(false)
	if s2.Attrs != AttrItalic|AttrDim {
		t.Errorf("after clear: %b", s2.Attrs)
	}
}

func TestEqual(t *testing.T) {
	a := Default.WithForeground(RGB(9, 9, 9)).WithBold(true)
	b := Default.WithForeground(RGB(9, 9, 9)).WithBold(true)
	if !a.Equal(b) {
		t.Error("identical styles must be Equal")
	}
	c := a.WithBold(false)
	if a.Equal(c) {
		t.Error("styles differing in attrs must not be Equal")
	}
}
