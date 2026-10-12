package text

import "testing"

// 中英混排的列宽是唯一权威口径。
func TestWidth(t *testing.T) {
	cases := map[string]int{
		"":      0,
		"abc":   3,
		"中文":    4,
		"中文ab":  6,
		"ab中文":  6,
		"中a文b":  6,
		"ｱｲｳ":   3, // 半角片假名
		"ＡＢＣ":   6, // 全角
		"a中b文c": 7,
	}
	for s, want := range cases {
		if got := Width(s); got != want {
			t.Errorf("Width(%q) = %d want %d", s, got, want)
		}
	}
}

// 组合字符与零宽字符不占列。
func TestZeroWidth(t *testing.T) {
	if got := Width("é"); got != 1 {
		t.Errorf("combining = %d want 1", got)
	}
	if got := RuneWidth(0x0301); got != 0 {
		t.Errorf("combining mark = %d want 0", got)
	}
}

// 截断不劈开宽字符。
func TestTruncate(t *testing.T) {
	cases := []struct {
		in     string
		budget int
		want   string
	}{
		{"中文", 1, ""},  // 放不下一个宽字符
		{"中文", 2, "中"}, // 恰好一个
		{"中文", 3, "中"}, // 留 1 列放不下第二个
		{"中文", 4, "中文"},
		{"abc", 2, "ab"},
		{"abc", 10, "abc"},
		{"a中b", 2, "a"},
		{"a中b", 3, "a中"},
	}
	for _, c := range cases {
		if got := Truncate(c.in, c.budget); got != c.want {
			t.Errorf("Truncate(%q,%d) = %q want %q", c.in, c.budget, got, c.want)
		}
	}
	if got := Truncate("x", 0); got != "" {
		t.Errorf("budget 0 = %q", got)
	}
}

// Fit 裁剪 + 填充，结果列宽恒等于 width。
func TestFit(t *testing.T) {
	for _, s := range []string{"", "a", "中文", "中文abc", "abc"} {
		for _, w := range []int{0, 1, 2, 3, 6, 10} {
			got := Fit(s, w, Left)
			if Width(got) != w && w > 0 {
				t.Errorf("Fit(%q,%d,Left) = %q width %d", s, w, got, Width(got))
			}
		}
	}
	if got := Fit("ab", 6, Left); got != "ab    " {
		t.Errorf("left = %q", got)
	}
	if got := Fit("ab", 6, Right); got != "    ab" {
		t.Errorf("right = %q", got)
	}
	if got := Fit("ab", 6, Center); got != "  ab  " {
		t.Errorf("center = %q", got)
	}
}

// 索引 ↔ 列号互换（仅在字符边界列成立；宽字符内部不可逆）。
func TestIndexColumnRoundTrip(t *testing.T) {
	s := "a中b文c"
	for idx := 0; idx <= len([]rune(s)); idx++ {
		col := OffsetAt(s, idx)
		if got := IndexAtColumn(s, col); got != idx {
			t.Errorf("idx %d -> col %d -> idx %d", idx, col, got)
		}
	}
}

// 列号落在宽字符右半格时，光标按"已越过该字符"处理。
func TestIndexAtColumnInsideWide(t *testing.T) {
	s := "中文" // 列 0-1 = 中，列 2-3 = 文
	if got := IndexAtColumn(s, 0); got != 0 {
		t.Errorf("col 0 -> %d want 0", got)
	}
	if got := IndexAtColumn(s, 1); got != 1 {
		t.Errorf("col 1 (中右半格) -> %d want 1", got)
	}
	if got := IndexAtColumn(s, 2); got != 1 {
		t.Errorf("col 2 -> %d want 1", got)
	}
	if got := IndexAtColumn(s, 4); got != 2 {
		t.Errorf("col 4 (末尾) -> %d want 2", got)
	}
}

// Pad 系列不裁剪。
func TestPad(t *testing.T) {
	if got := PadEnd("中文", 6); Width(got) != 6 {
		t.Errorf("PadEnd width = %d", Width(got))
	}
	if got := PadStart("中文", 6); got != "  中文" {
		t.Errorf("PadStart = %q", got)
	}
	if got := PadEnd("中文abc", 2); got != "中文abc" {
		t.Errorf("PadEnd must not truncate, got %q", got)
	}
}
