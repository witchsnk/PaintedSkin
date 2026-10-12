package screen

import (
	"strings"
	"testing"

	"PaintedSkin/geom"
	"PaintedSkin/style"
	"PaintedSkin/text"
)

// P2：续格语义。
//
// 续格（Width=-1）是宽字符右半的占位标记，**不能独立存在**。一旦某格是续格而
// 其左格已被覆盖，终端上就会残留半个字（或整行错位一列）。

// 覆盖宽字符左格时，右半占位必须同步清掉。
func TestSetCellOverWideLeftClearsContinuation(t *testing.T) {
	b := NewBuffer(6, 1)
	b.Text(0, 0, "中文", style.Default) // 0='中'(W2) 1=续 2='文'(W2) 3=续
	if !b.At(1, 0).IsContinuation() {
		t.Fatalf("precondition: col1 must be a continuation")
	}
	// 用空格覆盖左格 → 右半必须变成空格，不能留孤儿续格。
	b.SetCell(0, 0, ' ', style.Default)
	if c := b.At(1, 0); c.IsContinuation() {
		t.Errorf("orphan continuation left at col1: %+v", c)
	}
	if c := b.At(1, 0); c.Rune != ' ' || c.Width != 1 {
		t.Errorf("col1 = %+v want blank width-1", c)
	}
	assertNoDanglingContinuation(t, b)
}

// 覆盖宽字符右半所在的列：本格必须是合法格，且**左半降级**。
//
// 留下"没有配对的宽字符左格"会让 diff 为它写两列 → 整行右移一列。
func TestSetCellOverContinuationColumn(t *testing.T) {
	b := NewBuffer(6, 1)
	b.Text(0, 0, "中", style.Default)
	b.SetCell(1, 0, 'x', style.Default)
	if c := b.At(1, 0); c.IsContinuation() {
		t.Errorf("col1 = %+v want a real cell", c)
	}
	if c := b.At(1, 0); c.Rune != 'x' {
		t.Errorf("col1 = %+v want 'x'", c)
	}
	if c := b.At(0, 0); c.Width == 2 {
		t.Errorf("unpaired wide left cell: %+v", c)
	}
	assertNoUnpairedWide(t, b)
}

// 写入新宽字符时必须清掉它右侧的旧续格（否则新字与旧占位错位一列）。
func TestSetCellWideOverOldWide(t *testing.T) {
	b := NewBuffer(6, 1)
	b.Text(0, 0, "中文", style.Default)
	b.SetCell(0, 0, 'x', style.Default)
	// col1 原本是 '中' 的续格，已被 SetCell 清掉；col2 仍是 '文' 左格。
	if c := b.At(1, 0); c.IsContinuation() || c.Rune != ' ' {
		t.Errorf("col1 = %+v want blank", c)
	}
	assertNoDanglingContinuation(t, b)
}

// 覆盖续格后再写回宽字符，成对关系必须重新建立。
func TestWidePairRestored(t *testing.T) {
	b := NewBuffer(6, 1)
	b.Text(0, 0, "中", style.Default)
	b.SetCell(0, 0, ' ', style.Default)
	if c := b.At(1, 0); c.IsContinuation() {
		t.Fatalf("col1 must be cleared first")
	}
	b.SetCell(0, 0, '文', style.Default)
	if c := b.At(1, 0); !c.IsContinuation() {
		t.Errorf("col1 = %+v want continuation", c)
	}
	if c := b.At(0, 0); c.Rune != '文' || c.Width != 2 {
		t.Errorf("col0 = %+v want wide left cell", c)
	}
	assertNoDanglingContinuation(t, b)
}

// lineDiff：新旧都是续格时不得多发空格（否则整行右移一列）。
//
// 复现路径：旧帧 col0='中'(W2) col1=续；新帧 col0='中'(W2) col1=续，
// 但两者样式不同 → col1 单独进入 diff 段。此时绝不能再补一个空格，
// 因为左格已经把光标推过了这一列。
func TestLineDiffNoExtraSpaceForContinuation(t *testing.T) {
	old := NewBuffer(4, 1)
	old.SetCell(0, 0, '中', style.Style{}.WithForeground(style.Red))

	new := NewBuffer(4, 1)
	new.SetCell(0, 0, '中', style.Style{}.WithForeground(style.Blue))

	out := string(new.Present(old))
	if n := strings.Count(out, "中"); n != 1 {
		t.Errorf("wide rune must be emitted exactly once, got %d in %q", n, out)
	}
	// 断言：输出里不含"续格补空格"产生的那一列。
	if strings.Contains(out, "中 ") {
		t.Errorf("diff emitted an extra space after the wide rune: %q", out)
	}
	// 行内容仍是 4 列（中占 2 列 + 两个空格）。
	if got := rowText(new, 0); text.Width(got) != 4 {
		t.Errorf("row width = %d want 4 (row=%q)", text.Width(got), got)
	}
}

// lineDiff：新帧该列是续格、旧帧不是（旧宽字符被新宽字符顶掉）时，
// 必须重画左格且不额外补空格。
func TestLineDiffContinuationReplacedByWide(t *testing.T) {
	old := NewBuffer(4, 1)
	old.SetCell(0, 0, '中', style.Default) // 0='中' 1=续

	new := NewBuffer(4, 1)
	new.SetCell(0, 0, 'x', style.Default)
	new.SetCell(2, 0, '文', style.Default) // 2='文' 3=续

	out := string(new.Present(old))
	// '中' 被覆盖 → 必须出现；'x' 必须出现；不得有多余空格顶位。
	if !strings.Contains(out, "x") || !strings.Contains(out, "文") {
		t.Errorf("missing content in %q", out)
	}
	assertNoDanglingContinuation(t, new)
}

// 旧帧有宽字符、新帧同位置换成空格（半格残留）时必须补空格清掉半字。
func TestLineDiffClearsOldWideHalf(t *testing.T) {
	old := NewBuffer(4, 1)
	old.SetCell(0, 0, '中', style.Default) // 0/1

	new := NewBuffer(4, 1)
	new.SetCell(0, 0, ' ', style.Default) // 只剩 0 被覆盖，1 已被 SetCell 清成空格

	out := string(new.Present(old))
	if !strings.Contains(out, " ") {
		t.Errorf("old wide char must be cleared with a space: %q", out)
	}
	assertNoDanglingContinuation(t, new)
}

// 反复在同一行上改写宽字符，任何一帧都不得出现孤儿续格。
func TestNoDanglingContinuationAcrossFrames(t *testing.T) {
	prev := NewBuffer(8, 2)
	cur := NewBuffer(8, 2)
	contents := []string{"中文", "abc", "中a文", "", "😀x", "aa", "文文文"}
	for i, s := range contents {
		cur.Reset()
		cur.Text(0, 0, s, style.Default)
		cur.Text(0, 1, s, style.Style{}.WithReverse(true))
		assertNoDanglingContinuation(t, cur)
		_ = cur.Present(prev)
		old := prev
		prev = NewBuffer(8, 2)
		for y := 0; y < 2; y++ {
			for x := 0; x < 8; x++ {
				c, _ := cur.GetCell(x, y)
				prev.Set(x, y, c)
			}
		}
		_ = old
		_ = i
	}
}

// ---- P2：outline（不占布局的焦点环）----
//
// 本项目的 outline 是**只换样式、不动字符**的环：Textual 的实现会吃掉边框那一圈
// 的内容（它画在 border box 外的独立 gutter 里），而本项目布局没有预留 gutter，
// 且叶控件的 Bounds 就是全部内容。详见 screen/canvas_impl.go 的说明。

// outline 覆盖内容但不改变内容本身（不清空、不移动、不改宽度）。
func TestOutlineCoversWithoutMovingContent(t *testing.T) {
	b := NewBuffer(6, 3)
	b.Text(0, 0, "ab", style.Default)
	b.Text(0, 1, "cd", style.Default)
	b.Text(0, 2, "ef", style.Default)
	before := snapshot(b)
	b.Outline(geom.Rect{X: 0, Y: 0, W: 6, H: 3}, AllEdges, style.Style{}.WithReverse(true))
	for y := 0; y < 3; y++ {
		for x := 0; x < 6; x++ {
			bc, _ := before.GetCell(x, y)
			ac, _ := b.GetCell(x, y)
			// 内容（字符与宽度）一个都不能变。
			if ac.Rune != bc.Rune || ac.Width != bc.Width {
				t.Errorf("(%d,%d): outline changed the cell: %+v -> %+v", x, y, bc, ac)
			}
			// 只有环上（外圈）被覆盖，内部必须原样。
			ring := y == 0 || y == 2 || x == 0 || x == 5
			if got := ac.Style.Reverse; got != ring {
				t.Errorf("(%d,%d) reverse = %v want %v (ring only)", x, y, got, ring)
			}
		}
	}
}

// outline 绝不写字形（否则会吃掉内容）。
func TestOutlineDoesNotDrawGlyphs(t *testing.T) {
	for _, border := range AllBorders {
		b := NewBuffer(5, 3)
		b.Text(0, 0, "hello", style.Default)
		b.Text(0, 1, "world", style.Default)
		b.Text(0, 2, "!", style.Default)
		before := snapshot(b)
		b.Outline(geom.Rect{X: 0, Y: 0, W: 5, H: 3}, AllEdges,
			style.Style{}.WithForeground(style.Red))
		for y := 0; y < 3; y++ {
			for x := 0; x < 5; x++ {
				bc, _ := before.GetCell(x, y)
				ac, _ := b.GetCell(x, y)
				if ac.Rune != bc.Rune {
					t.Errorf("%s: outline drew %q over content %q at (%d,%d)",
						border.Style, ac.Rune, bc.Rune, x, y)
				}
			}
		}
		_ = border
	}
}

// outline 只是换样式：宽字符的成对结构必须保持。
func TestOutlinePreservesWidePair(t *testing.T) {
	b := NewBuffer(6, 1)
	b.Text(0, 0, "中文", style.Default)
	b.Outline(geom.Rect{X: 0, Y: 0, W: 6, H: 1}, AllEdges, style.Style{}.WithReverse(true))
	if c := b.At(1, 0); !c.IsContinuation() {
		t.Errorf("col1 = %+v want continuation preserved", c)
	}
	if c := b.At(3, 0); !c.IsContinuation() {
		t.Errorf("col3 = %+v want continuation preserved", c)
	}
	assertNoDanglingContinuation(t, b)
}

// 逐边开关：只套勾选的边。
func TestOutlineEdgeControl(t *testing.T) {
	cases := []struct {
		name  string
		edges BorderEdges
		want  func(x, y int) bool
	}{
		{"top", BorderEdges{Top: true}, func(x, y int) bool { return y == 0 }},
		{"bottom", BorderEdges{Bottom: true}, func(x, y int) bool { return y == 2 }},
		{"left", BorderEdges{Left: true}, func(x, y int) bool { return x == 0 }},
		{"right", BorderEdges{Right: true}, func(x, y int) bool { return x == 4 }},
		{"lr", BorderEdges{Left: true, Right: true},
			func(x, y int) bool { return x == 0 || x == 4 }},
		{"tb", BorderEdges{Top: true, Bottom: true},
			func(x, y int) bool { return y == 0 || y == 2 }},
	}
	for _, tc := range cases {
		b := NewBuffer(5, 3)
		b.Outline(geom.Rect{X: 0, Y: 0, W: 5, H: 3}, tc.edges,
			style.Style{}.WithReverse(true))
		for y := 0; y < 3; y++ {
			for x := 0; x < 5; x++ {
				c, _ := b.GetCell(x, y)
				if got := c.Style.Reverse; got != tc.want(x, y) {
					t.Errorf("%s: (%d,%d) reverse = %v want %v", tc.name, x, y, got, tc.want(x, y))
				}
			}
		}
	}
}

// 空的边开关不做任何事。
func TestOutlineNoEdges(t *testing.T) {
	b := NewBuffer(4, 3)
	b.FillStyle(geom.Rect{X: 0, Y: 0, W: 4, H: 3}, style.Default)
	b.Outline(geom.Rect{X: 0, Y: 0, W: 4, H: 3}, BorderEdges{}, style.Style{}.WithReverse(true))
	if c, _ := b.GetCell(0, 0); c.Style.Reverse {
		t.Error("empty edges must not draw")
	}
}

// outline 尊重裁剪区。
func TestOutlineRespectsClip(t *testing.T) {
	b := NewBuffer(8, 3)
	b.FillStyle(geom.Rect{X: 0, Y: 0, W: 8, H: 3}, style.Default)
	b.Clip(geom.Rect{X: 0, Y: 0, W: 3, H: 3})
	b.Outline(geom.Rect{X: 0, Y: 0, W: 6, H: 3}, AllEdges, style.Style{}.WithReverse(true))
	if c, _ := b.GetCell(3, 0); c.Style.Reverse {
		t.Error("outside clip must stay untouched")
	}
	if c, _ := b.GetCell(0, 0); !c.Style.Reverse {
		t.Error("inside clip must be covered")
	}
}

// outline 只在区域内落笔，绝不越界。
func TestOutlineStaysInsideRect(t *testing.T) {
	b := NewBuffer(8, 5)
	b.FillStyle(geom.Rect{X: 0, Y: 0, W: 8, H: 5}, style.Default)
	b.Outline(geom.Rect{X: 2, Y: 1, W: 3, H: 2}, AllEdges, style.Style{}.WithReverse(true))
	for y := 0; y < 5; y++ {
		for x := 0; x < 8; x++ {
			c, _ := b.GetCell(x, y)
			inside := (y == 1 || y == 2) && x >= 2 && x < 5
			if c.Style.Reverse != inside {
				t.Errorf("(%d,%d) reverse = %v want %v", x, y, c.Style.Reverse, inside)
			}
		}
	}
}

// 退化尺寸不 panic。
func TestOutlineDegenerate(t *testing.T) {
	for _, size := range []geom.Size{{W: 1, H: 1}, {W: 6, H: 1}, {W: 1, H: 4}, {W: 0, H: 3}, {W: 3, H: 0}} {
		b := NewBuffer(8, 6)
		b.Outline(geom.Rect{X: 0, Y: 0, W: size.W, H: size.H}, AllEdges, style.Default)
		assertNoDanglingContinuation(t, b)
	}
}

// ---- helpers ----

// assertNoUnpairedWide 断言宽字符成对不变量（**双向**）：
// 每个 `Width==2` 的左格必须紧跟一个续格。
//
// 反方向（每个续格的左邻必须是 `Width==2`）由
// assertNoDanglingContinuation 覆盖。
func assertNoUnpairedWide(t *testing.T, b *Buffer) {
	t.Helper()
	for y := 0; y < b.Height(); y++ {
		for x := 0; x < b.Width(); x++ {
			c, _ := b.GetCell(x, y)
			if c.Width != 2 {
				continue
			}
			if x+1 >= b.Width() {
				t.Errorf("(%d,%d): wide left cell with no room for its right half", x, y)
				continue
			}
			if r, _ := b.GetCell(x+1, y); !r.IsContinuation() {
				t.Errorf("(%d,%d): unpaired wide left cell, right half = %+v", x, y, r)
			}
		}
	}
}

// assertNoDanglingContinuation 断言没有"孤立续格"：每个续格的左邻
// 必须是一个宽度为 2 的宽字符左格。
func assertNoDanglingContinuation(t *testing.T, b *Buffer) {
	t.Helper()
	for y := 0; y < b.Height(); y++ {
		for x := 0; x < b.Width(); x++ {
			c, _ := b.GetCell(x, y)
			if !c.IsContinuation() {
				continue
			}
			if x == 0 {
				t.Errorf("(%d,%d): continuation with no left half", x, y)
				continue
			}
			l, _ := b.GetCell(x-1, y)
			if l.Width != 2 {
				t.Errorf("(%d,%d): orphan continuation, left half = %+v", x, y, l)
			}
		}
	}
}

// snapshot 复制整块缓冲。
func snapshot(b *Buffer) *Buffer {
	out := NewBuffer(b.Width(), b.Height())
	for y := 0; y < b.Height(); y++ {
		for x := 0; x < b.Width(); x++ {
			c, _ := b.GetCell(x, y)
			out.Set(x, y, c)
		}
	}
	return out
}

// text2Width 按列宽算字符串宽度。
func text2Width(s string) int { return text.Width(s) }

// ---- 回归：铺底切开宽字符对会让 diff 写错列数 ----

// FillStyle 从续格那一列开始铺时，左半必须降级；否则该宽字符在 diff 里
// 会被写成两列，而缓冲只打算写一格 → 终端右移一列。
func TestFillStyleMustNotLeaveUnpairedWide(t *testing.T) {
	b := NewBuffer(6, 1)
	b.SetCell(1, 0, '中', style.Default) // 1=左格，2=续格
	b.FillStyle(geom.Rect{X: 2, Y: 0, W: 4, H: 1}, style.Default)
	assertNoUnpairedWide(t, b)
	assertNoDanglingContinuation(t, b)
	// col2 是空格（不是续格）
	if c := b.At(2, 0); c.Rune != ' ' || c.Width != 1 {
		t.Errorf("col2 = %+v want blank", c)
	}
}

// 端到端：老帧有宽字符、新帧把它切成空格，diff 不得再吐出那个宽字符。
func TestLineDiffDoesNotResurrectSplitWide(t *testing.T) {
	old := NewBuffer(6, 1)
	old.SetCell(1, 0, '中', style.Default)

	new := NewBuffer(6, 1)
	new.SetCell(1, 0, '中', style.Default)
	new.FillStyle(geom.Rect{X: 2, Y: 0, W: 4, H: 1}, style.Default)

	out := string(new.Present(old))
	if strings.Contains(out, "中") {
		t.Errorf("diff must not re-emit the split wide rune: %q", out)
	}
	if got := text.Width(rowText(new, 0)); got != 6 {
		t.Errorf("row width = %d want 6", got)
	}
}

// BorderNone 被 WithEdges 打开边后仍不可见（否则 Box 会写出一圈零值格）。
func TestBorderNoneWithEdgesStaysInvisible(t *testing.T) {
	b := BorderNone.WithEdges(BorderEdges{Top: true})
	if b.Visible() {
		t.Errorf("BorderNone(%+v) has no glyphs and must stay invisible", b.Edges)
	}
	buf := NewBuffer(4, 2)
	buf.FillStyle(geom.Rect{X: 0, Y: 0, W: 4, H: 2}, style.Default)
	buf.Box(geom.Rect{X: 0, Y: 0, W: 4, H: 2}, b, style.Default)
	for y := 0; y < 2; y++ {
		for x := 0; x < 4; x++ {
			c, _ := buf.GetCell(x, y)
			if c.Rune == 0 && c.Width == 0 {
				t.Errorf("Box must not write zero cells: (%d,%d)", x, y)
			}
		}
	}
}
