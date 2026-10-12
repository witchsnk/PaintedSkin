package screen

import (
	"strings"
	"testing"

	"PaintedSkin/geom"
	"PaintedSkin/style"
)

// ---- Cell.Width：宽字符右格占位（docs/05）----

// Text 写宽字符时，左格存字、右格写 Width=-1 占位。
func TestWideCharContinuation(t *testing.T) {
	b := NewBuffer(6, 1)
	b.Text(0, 0, "中", style.Default)
	left, _ := b.GetCell(0, 0)
	right, _ := b.GetCell(1, 0)
	if left.Rune != '中' || left.Width != 2 {
		t.Errorf("left = %+v want rune=中 width=2", left)
	}
	if right.Rune != 0 || right.Width != -1 {
		t.Errorf("right = %+v want rune=0 width=-1", right)
	}
	// 宽字符占两格，后续字符从第 2 格开始
	b.Text(0, 0, "中a", style.Default)
	at2, _ := b.GetCell(2, 0)
	if at2.Rune != 'a' {
		t.Errorf("after wide char = %+v want 'a'", at2)
	}
}

// 窄字符 Width=1；空格 Width=1。
func TestNarrowCharWidth(t *testing.T) {
	b := NewBuffer(4, 1)
	b.SetCell(0, 0, 'a', style.Default)
	c, _ := b.GetCell(0, 0)
	if c.Width != 1 {
		t.Errorf("narrow width = %d want 1", c.Width)
	}
	b.Fill(geom.Rect{X: 1, Y: 0, W: 2, H: 1}, ' ', style.Default)
	f, _ := b.GetCell(1, 0)
	if f.Width != 1 || f.Rune != ' ' {
		t.Errorf("fill = %+v", f)
	}
}

// Reset 把宽度恢复为 1（不留 -1 残留）。
func TestResetRestoresWidth(t *testing.T) {
	b := NewBuffer(4, 1)
	b.Text(0, 0, "中", style.Default)
	b.Reset()
	c, _ := b.GetCell(0, 0)
	if c.Rune != 0 || c.Width != 0 {
		t.Errorf("reset = %+v", c)
	}
}

// ---- 半格覆盖必须补空格清残留（docs/05）----

// 宽字符被窄字符覆盖其左半格时，右格占位必须补空格。
func TestHalfOverwriteClearsResidue(t *testing.T) {
	old := NewBuffer(4, 1)
	old.Text(0, 0, "中", style.Default) // 左格=中(宽2) 右格=占位(-1)
	nb := NewBuffer(4, 1)
	nb.SetCell(0, 0, 'x', style.Default) // 只覆盖左格 → 右格成半格残留
	out := string(nb.Present(old))
	if !strings.Contains(out, "x") {
		t.Errorf("must emit new char: %q", out)
	}
	// 右格残留必须被空格清掉
	if !strings.Contains(out, " ") {
		t.Errorf("wide residue must be cleared with a space: %q", out)
	}
}

// 宽字符被两个窄字符连续覆盖时，右格的窄字符必须被正常输出，不能被误判为残留而替换为空格。
func TestWideCharOverwriteByMultipleNarrowChars(t *testing.T) {
	old := NewBuffer(4, 1)
	old.Text(0, 0, "中", style.Default)
	nb := NewBuffer(4, 1)
	nb.Text(0, 0, "xy", style.Default)
	out := string(nb.Present(old))
	if !strings.Contains(out, "xy") {
		t.Errorf("must emit both chars 'xy', got: %q", out)
	}
}

// ---- fullRedraw 逐格输出（docs/05）----

// 首帧全量重绘必须逐格输出内容，不能只有清屏。
func TestFullRedrawEmitsCells(t *testing.T) {
	b := NewBuffer(3, 1)
	b.Text(0, 0, "abc", style.Default)
	out := string(b.Present(nil))
	if !strings.Contains(out, "abc") {
		t.Errorf("full redraw must emit cell content: %q", out)
	}
	if !strings.Contains(out, "\x1b[2J") {
		t.Errorf("full redraw must clear screen: %q", out)
	}
}

// 尺寸变化视为全量重绘。
func TestResizeTriggersFullRedraw(t *testing.T) {
	old := NewBuffer(4, 1)
	old.Text(0, 0, "aaaa", style.Default)
	nb := NewBuffer(6, 1)
	nb.Text(0, 0, "bbbbbb", style.Default)
	out := string(nb.Present(old))
	if !strings.Contains(out, "bbbbbb") {
		t.Errorf("resize must full redraw: %q", out)
	}
}

// ---- 样式批处理：同样式连续段只发一条 SGR（docs/05）----

func TestStyleBatching(t *testing.T) {
	old := NewBuffer(6, 1)
	old.Fill(geom.Rect{X: 0, Y: 0, W: 6, H: 1}, ' ', style.Default)
	nb := NewBuffer(6, 1)
	red := style.Style{}.WithForeground(style.Red)
	nb.SetCell(0, 0, 'a', red)
	nb.SetCell(1, 0, 'b', red)
	nb.SetCell(2, 0, 'c', red)
	out := string(nb.Present(old))
	if n := strings.Count(out, "31m"); n != 1 {
		t.Errorf("same-style run must emit one SGR, got %d: %q", n, out)
	}
	if !strings.Contains(out, "abc") {
		t.Errorf("must emit content: %q", out)
	}
}

// 样式变化才发第二条 SGR。
func TestStyleChangeEmitsNewSGR(t *testing.T) {
	old := NewBuffer(4, 1)
	old.Fill(geom.Rect{X: 0, Y: 0, W: 4, H: 1}, ' ', style.Default)
	nb := NewBuffer(4, 1)
	nb.SetCell(0, 0, 'a', style.Style{}.WithForeground(style.Red))
	nb.SetCell(1, 0, 'b', style.Style{}.WithForeground(style.Blue))
	out := string(nb.Present(old))
	if !strings.Contains(out, "31m") || !strings.Contains(out, "34m") {
		t.Errorf("style change must emit both SGRs: %q", out)
	}
}

// 未变化行整体跳过。
func TestUnchangedRowSkipped(t *testing.T) {
	old := NewBuffer(4, 2)
	old.Text(0, 0, "aaaa", style.Default)
	old.Text(0, 1, "bbbb", style.Default)
	nb := NewBuffer(4, 2)
	nb.Text(0, 0, "aaaa", style.Default)
	nb.Text(0, 1, "cccc", style.Default)
	out := string(nb.Present(old))
	if strings.Contains(out, "\x1b[1;1H") {
		t.Errorf("unchanged row 0 must not be re-emitted: %q", out)
	}
	if !strings.Contains(out, "cccc") {
		t.Errorf("changed row must be emitted: %q", out)
	}
}

// 连续变化段只定位一次光标。
func TestSingleCursorPositionPerRun(t *testing.T) {
	old := NewBuffer(8, 1)
	old.Fill(geom.Rect{X: 0, Y: 0, W: 8, H: 1}, ' ', style.Default)
	nb := NewBuffer(8, 1)
	for x := 0; x < 8; x++ {
		nb.SetCell(x, 0, 'x', style.Default)
	}
	out := string(nb.Present(old))
	if n := strings.Count(out, "H"); n != 1 {
		t.Errorf("one contiguous run needs one cursor move, got %d: %q", n, out)
	}
}
