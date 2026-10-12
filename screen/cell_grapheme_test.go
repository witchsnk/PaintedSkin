package screen

import (
	"strings"
	"testing"

	"PaintedSkin/geom"
	"PaintedSkin/style"
	"PaintedSkin/text"
)

// 零宽字符（组合附标 / ZWJ 连接符）必须并入前一格，而不是被丢弃。
//
// Cell.Rune 存基字符，Cell.Tail 存紧随其后的零宽序列（按顺序）。
// 渲染时按"基字符 + Tail"整串写出，终端才会把它们合成为一个字素。

const (
	acute      = "\u0301" // 组合尖音符
	macronRune = '\u0304' // 组合长音符
	acuteRune  = '\u0301'
	zwj        = "\u200d" // 零宽连接符
)

// Text 必须把组合附标并入基字符所在格。
func TestTextMergesCombiningMark(t *testing.T) {
	b := NewBuffer(6, 1)
	b.Text(0, 0, "e"+acute, style.Default)
	c, _ := b.GetCell(0, 0)
	if c.Rune != 'e' {
		t.Errorf("Rune = %q want 'e'", c.Rune)
	}
	if c.Tail != acute {
		t.Errorf("Tail = %q want %q (combining mark was dropped)", c.Tail, acute)
	}
	if c.Width != 1 {
		t.Errorf("Width = %d want 1 (marks are zero-width)", c.Width)
	}
	assertNoUnpairedWide(t, b)
	assertNoDanglingContinuation(t, b)
}

// 多个附标按出现顺序串在 Tail 里。
func TestTextKeepsMarkOrder(t *testing.T) {
	b := NewBuffer(6, 1)
	b.Text(0, 0, "a"+acute+acute, style.Default)
	c, _ := b.GetCell(0, 0)
	if c.Tail != acute+acute {
		t.Errorf("Tail = %q want %q", c.Tail, acute+acute)
	}
}

// 宽字符后的附标必须并入**宽字符左格**（不是它前面的格）。
func TestTextMergesMarkAfterWideChar(t *testing.T) {
	b := NewBuffer(6, 1)
	b.Text(0, 0, "中"+acute, style.Default)
	left, _ := b.GetCell(0, 0)
	if left.Rune != '中' || left.Width != 2 {
		t.Errorf("col0 = %+v want intact wide left cell", left)
	}
	if left.Tail != acute {
		t.Errorf("col0 Tail = %q want %q", left.Tail, acute)
	}
	cont, _ := b.GetCell(1, 0)
	if !cont.IsContinuation() {
		t.Errorf("col1 = %+v want continuation", cont)
	}
	if cont.Tail != "" {
		t.Errorf("continuation must not carry Tail: %q", cont.Tail)
	}
}

// 行首的孤立附标没有基字符可依附 → 丢弃（否则渲染成豆腐块）。
func TestZeroWidthAtRowStartDropped(t *testing.T) {
	b := NewBuffer(6, 1)
	b.Text(0, 0, acute+"a", style.Default)
	c, _ := b.GetCell(0, 0)
	if c.Rune != 'a' || c.Tail != "" {
		t.Errorf("col0 = %+v want plain 'a' (orphan mark must be dropped)", c)
	}
}

// ZWJ 序列：连接符并入前一格，终端才能合成为一个字形。
func TestTextMergesZWJ(t *testing.T) {
	const family = "\U0001F468" + zwj + "\U0001F469" + zwj + "\U0001F467"
	b := NewBuffer(12, 1)
	b.Text(0, 0, family, style.Default)
	// 三个宽字符各带一个 ZWJ
	for i, col := range []int{0, 2, 4} {
		c, _ := b.GetCell(col, 0)
		if c.Width != 2 {
			t.Errorf("col%d = %+v want wide left cell", col, c)
		}
		if i < 2 && c.Tail != zwj {
			t.Errorf("col%d Tail = %q want %q", col, c.Tail, zwj)
		}
	}
	assertNoUnpairedWide(t, b)
	assertNoDanglingContinuation(t, b)
	// 列宽口径不变：ZWJ 是零宽，只由 3 个宽字符贡献 6 列。
	if got := contentWidth(b, 0); got != 6 {
		t.Errorf("content width = %d want 6 (ZWJ is zero-width)", got)
	}
	if got := rowWidth(b, 0); got != b.Width() {
		t.Errorf("row width = %d want buffer width %d", got, b.Width())
	}
}

// SetCell 写零宽字符时并入前一格（x-1），与 Text 的列推进一致。
func TestSetCellMergesIntoPrevious(t *testing.T) {
	b := NewBuffer(6, 1)
	b.SetCell(0, 0, 'e', style.Default)
	b.SetCell(1, 0, acuteRune, style.Default) // 并入 col0
	c, _ := b.GetCell(0, 0)
	if c.Rune != 'e' || c.Tail != acute {
		t.Errorf("col0 = %+v want 'e' with Tail %q", c, acute)
	}
	if n, _ := b.GetCell(1, 0); n.Rune != 0 || n.Width != 0 {
		t.Errorf("col1 = %+v want untouched", n)
	}
}

// Cell.Equal 必须比较 Tail（否则改了附标不会重画）。
func TestCellEqualComparesTail(t *testing.T) {
	a := Cell{Rune: 'e', Width: 1, Tail: acute}
	b := Cell{Rune: 'e', Width: 1}
	if a.Equal(b) || b.Equal(a) {
		t.Error("Tail must take part in Cell.Equal")
	}
	if !a.Equal(Cell{Rune: 'e', Width: 1, Tail: acute}) {
		t.Error("identical cells must compare equal")
	}
}

// 渲染输出必须包含完整的"基字符 + 附标"序列。
func TestRenderEmitsCombiningSequence(t *testing.T) {
	b := NewBuffer(6, 1)
	b.Text(0, 0, "e"+acute, style.Default)
	out := string(b.Present(nil))
	if !strings.Contains(out, "e"+acute) {
		t.Errorf("Present = %q must contain the full grapheme %q", out, "e"+acute)
	}
}

// 帧间只改附标也必须重画（Tail 参与 diff）。
func TestLineDiffRedrawsWhenTailChanges(t *testing.T) {
	old := NewBuffer(6, 1)
	old.SetCell(0, 0, 'e', style.Default)
	old.SetCell(1, 0, acuteRune, style.Default)

	newB := NewBuffer(6, 1)
	newB.SetCell(0, 0, 'e', style.Default)
	newB.SetCell(1, 0, macronRune, style.Default) // 换成另一个附标（长音符）

	out := string(newB.Present(old))
	if !strings.Contains(out, "e") || !strings.Contains(out, "̄") {
		t.Errorf("Present = %q must re-emit the cell with the new mark", out)
	}
}

// 行宽不变量不受附标影响。
func TestRowWidthInvariantWithMarks(t *testing.T) {
	for _, s := range []string{
		"e" + acute,
		"e" + acute + acute,
		"中" + acute,
		"中文" + acute,
		"éèê", // e+´ / e+` / e+^
	} {
		b := NewBuffer(20, 1)
		b.Text(0, 0, s, style.Default)
		if got, want := rowWidth(b, 0), 20; got != want {
			t.Errorf("%q: row width = %d want %d", s, got, want)
		}
		if got := text.Width(s); got != text.Width(strings.TrimSuffix(s, acute)) {
			t.Errorf("%q: text.Width disagrees with cell accounting", s)
		}
	}
}

// SetCell 覆盖宽字符左格时，附标一并清掉（不能留下悬空的附标）。
func TestSetCellClearsTailOnOverwrite(t *testing.T) {
	b := NewBuffer(6, 1)
	b.Text(0, 0, "e"+acute, style.Default)
	b.SetCell(0, 0, 'x', style.Default)
	c, _ := b.GetCell(0, 0)
	if c.Tail != "" {
		t.Errorf("Tail = %q want empty after overwrite", c.Tail)
	}
}

// Fill/FillStyle 铺底时把 Tail 一起清掉。
func TestFillClearsTail(t *testing.T) {
	b := NewBuffer(6, 1)
	b.Text(0, 0, "e"+acute, style.Default)
	b.FillStyle(geom.Rect{X: 0, Y: 0, W: 6, H: 1}, style.Default)
	for x := 0; x < 6; x++ {
		c, _ := b.GetCell(x, 0)
		if c.Tail != "" {
			t.Errorf("col%d Tail = %q want empty", x, c.Tail)
		}
	}
}

// 退化写入不得 panic。
func TestZeroWidthDegenerate(t *testing.T) {
	b := NewBuffer(2, 1)
	b.SetCell(0, 0, acuteRune, style.Default) // 行首，无基字符 → 丢弃
	b.SetCell(0, 0, 'a', style.Default)
	b.SetCell(1, 0, acuteRune, style.Default)
	b.Text(0, 0, acute, style.Default) // 行首孤立附标
	assertNoUnpairedWide(t, b)
	assertNoDanglingContinuation(t, b)
}

// contentWidth 返回该行非续格、非空格格子的列宽之和（不含尾部空白）。
func contentWidth(b *Buffer, y int) int {
	w := 0
	for x := 0; x < b.Width(); x++ {
		c, _ := b.GetCell(x, y)
		if c.IsContinuation() || c.Rune == 0 {
			continue
		}
		w += text.RuneWidth(c.Rune)
	}
	return w
}
