package screen

import (
	"strings"
	"testing"

	"PaintedSkin/geom"
	"PaintedSkin/style"
	"PaintedSkin/text"
)

// boxString 把缓冲的一行还原成字符串（宽字符右格跳过），
// 用于校验"整行显示宽 == 缓冲宽度"这条不变量。
func rowText(b *Buffer, y int) string {
	var sb strings.Builder
	for x := 0; x < b.Width(); x++ {
		c, _ := b.GetCell(x, y)
		if c.IsContinuation() {
			continue
		}
		if c.Rune == 0 {
			sb.WriteRune(' ')
			continue
		}
		sb.WriteRune(c.Rune)
	}
	return sb.String()
}

// rowWidth 返回该行在真实终端里占据的列数。
// 规则：宽字符右格占位 = 0 列（被左格吃掉）；空白格 = 1 列；
// 其余按 RuneWidth。
func rowWidth(b *Buffer, y int) int {
	w := 0
	for x := 0; x < b.Width(); x++ {
		c, _ := b.GetCell(x, y)
		if c.IsContinuation() {
			continue
		}
		if c.Rune == 0 {
			w++ // 空白占一列
			continue
		}
		w += text.RuneWidth(c.Rune)
	}
	return w
}

// 不变量：任何内容写入后，整行显示宽都必须等于缓冲宽度（边框不被破坏）。
func TestRowWidthInvariant(t *testing.T) {
	cases := []string{
		"中文",
		"中文abc",
		"abc中文",
		"a中b文c",
		"中a文b",
		"中",
		"a",
		"混合Mixed文本Text",
		"emoji😀测试",
		"ｱｲｳ全角",
	}
	for _, s := range cases {
		b := NewBuffer(20, 1)
		b.Text(0, 0, s, style.Default)
		if got := rowWidth(b, 0); got > 20 {
			t.Errorf("%q: row width %d > 20 (overflow)", s, got)
		}
		if got := rowWidth(b, 0); got != b.Width() {
			t.Errorf("%q: row occupies %d columns, buffer is %d", s, got, b.Width())
		}
	}
}

// 带边框的框：内容中英混排后，右边框列必须完好。
func TestBoxRightBorderSurvivesMixedText(t *testing.T) {
	const w = 16
	inner := w - 2
	for _, s := range []string{
		"中文", "中文abc", "abc中文", "中a文b", "混合Mixed", "😀测试", "ｱｲｳ",
	} {
		b := NewBuffer(w, 3)
		b.Box(geom.Rect{X: 0, Y: 0, W: w, H: 3}, BorderSingle, style.Default)
		text := truncateTo(s, inner)
		b.Text(1, 1, text, style.Default)

		right, _ := b.GetCell(w-1, 1)
		if right.Rune != '│' {
			t.Errorf("%q: right border broken, cell=%q row=%q", s, right.Rune, rowText(b, 1))
		}
		if got := rowWidth(b, 1); got != w {
			t.Errorf("%q: row width %d != %d", s, got, w)
		}
	}
}

// 宽字符正好卡在右边界时不能越界写入。
func TestWideCharAtRightEdge(t *testing.T) {
	b := NewBuffer(4, 1)
	// 第 3 列放宽字符 → 只能占 3..3，越界部分丢弃
	b.Text(3, 0, "中", style.Default)
	if got := rowWidth(b, 0); got > 4 {
		t.Errorf("overflow: %d", got)
	}
	// 右边放不下 → 退化为空格，不丢内容也不越界
	if got, _ := b.GetCell(3, 0); got.Rune != ' ' {
		t.Errorf("cell3 = %q want space fallback", got.Rune)
	}
	// 不能留下悬空占位（否则终端会把宽字符拆到下一行）
	if c, _ := b.GetCell(3, 0); c.IsContinuation() {
		t.Error("must not create dangling continuation")
	}
}

// Text 截断：截断点必须落在字符边界，且不超宽。
func TestTruncateKeepsColumnBudget(t *testing.T) {
	for _, s := range []string{"中文abc", "abc中文", "中a文b", "😀测试test"} {
		for _, budget := range []int{0, 1, 2, 3, 5, 8} {
			got := truncateTo(s, budget)
			if w := text.Width(got); w > budget {
				t.Errorf("%q @%d -> %q width %d exceeds", s, budget, got, w)
			}
		}
	}
}

// truncateTo 走权威实现。
func truncateTo(s string, w int) string { return text.Truncate(s, w) }

// 边框 + 中英混排内容：每一行的最右可见字符都必须是右边框。
// 这是"整体边框规整"的可执行定义。
func TestFrameIntegrityWithMixedContent(t *testing.T) {
	const w, h = 20, 5
	body := []string{
		"中文标题 mixed",
		"ascii only line",
		"中a文b混排",
		"",
		"emoji 😀 行",
	}
	for _, bi := range body {
		b := NewBuffer(w, h)
		b.Box(geom.Rect{X: 0, Y: 0, W: w, H: h}, BorderSingle, style.Default)
		b.Text(1, 1, truncateTo(bi, w-2), style.Default)

		for y := 0; y < h; y++ {
			right, _ := b.GetCell(w-1, y)
			switch y {
			case 0, h - 1:
				if right.Rune != '┐' && right.Rune != '┘' {
					t.Errorf("%q row %d: right = %q want corner", bi, y, right.Rune)
				}
			default:
				if right.Rune != '│' {
					t.Errorf("%q row %d: right border broken = %q", bi, y, right.Rune)
				}
			}
			if got := rowWidth(b, y); got != w {
				t.Errorf("%q row %d: width %d != %d", bi, y, got, w)
			}
		}
	}
}

// 标题也不能破坏顶边框：标题写在顶行内部，两侧仍是横线。
func TestFrameTopBorderWithTitle(t *testing.T) {
	const w = 20
	b := NewBuffer(w, 3)
	b.Box(geom.Rect{X: 0, Y: 0, W: w, H: 3}, BorderSingle, style.Default)
	title := " 中文 mixed "
	b.Text(2, 0, truncateTo(title, w-4), style.Default)

	// 顶行左右角必须是角
	lt, _ := b.GetCell(0, 0)
	rt, _ := b.GetCell(w-1, 0)
	if lt.Rune != '┌' || rt.Rune != '┐' {
		t.Errorf("corners broken: %q %q", lt.Rune, rt.Rune)
	}
	if got := rowWidth(b, 0); got != w {
		t.Errorf("top row width %d != %d", got, w)
	}
}
