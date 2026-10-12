package widget

import (
	"strings"
	"testing"

	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/style"
	"PaintedSkin/text"
	"PaintedSkin/widget/layout"
)

// P3：字形/宽度口径归一、命名常量、死代码、Base.Bind、边框补档与 hatch。

// ---- P3-1：控件统一走 text.Truncate / text.Fit ----

// fitText 必须丢弃"没有基字符的组合附标"（它会渲染成豆腐块）。
// 这是被删掉的 widget.truncate（text.Truncate 的重写版）漏掉的零宽字符处理。
//
// 注意：渲染层 `Buffer.Text` 目前**丢弃**零宽字符（ADR-017 的已知遗留，
// 与 ZWJ emoji 同族），所以这一层只能直接断言 fitText 的输出。
func TestFitTextDropsLeadingCombiningMark(t *testing.T) {
	const acute = "\u0301"
	if got := fitText(acute+"ab", 1, layout.AlignStart); got != "a" {
		t.Errorf("fitText = %q want %q (leading combining mark must be dropped)", got, "a")
	}
}

// fitText 保留跟随基字符的组合附标（不丢）。
func TestFitTextKeepsCombiningMarkOnBase(t *testing.T) {
	const acute = "\u0301"
	labeled := "e" + acute + "e" + acute + "e"
	got := fitText(labeled, 2, layout.AlignStart)
	want := "e" + acute + "e" + acute
	if got != want {
		t.Errorf("fitText = %q want %q", got, want)
	}
	if w := text.Width(got); w != 2 {
		t.Errorf("fitText width = %d want 2", w)
	}
}

// Tabs 标签按列宽截断（此前走私有 truncate）。
func TestTabsLabelTruncatesByColumn(t *testing.T) {
	tb := NewTabs()
	tb.SetTabs([]string{"中文标签"})
	tb.SetContent(NewLabel("x"))
	buf := paintBounds(tb, geom.Rect{X: 0, Y: 0, W: 6, H: 2})
	row := rowOf(buf, 0)
	if w := text.Width(row); w > 6 {
		t.Errorf("tab row width = %d > 6 (%q)", w, row)
	}
	// 截断不劈宽字符：不得出现孤立续格（后续 cell 校验覆盖，这里只看宽度）
}

// Table 单元格按对齐补齐到**列宽**（Center / End / Start）。
func TestTableCellAlignment(t *testing.T) {
	cases := []struct {
		align layout.Align
		want  string
	}{
		{layout.AlignStart, "ab  "},
		{layout.AlignCenter, " ab "},
		{layout.AlignEnd, "  ab"},
	}
	for _, tc := range cases {
		tb := NewTable()
		tb.SetColumns([]TableColumn{{Title: "H", Width: 4, Align: tc.align}})
		tb.SetRows([][]string{{"ab"}})
		buf := paintBounds(tb, geom.Rect{X: 0, Y: 0, W: 10, H: 2})
		if got := colsOf(buf, 1, 4); got != tc.want {
			t.Errorf("align %v = %q want %q", tc.align, got, tc.want)
		}
	}
}

// 表格单元格严格占满列宽（这是"按列补齐"的可执行验收）。
func TestTableCellFillsColumnWidth(t *testing.T) {
	tb := NewTable()
	tb.SetColumns([]TableColumn{{Title: "H", Width: 4}})
	tb.SetRows([][]string{{"中文"}}) // 占 2 列 → 补 2 格
	buf := paintBounds(tb, geom.Rect{X: 0, Y: 0, W: 10, H: 2})
	if got := colsOf(buf, 1, 4); got != "中文  " {
		t.Errorf("cell = %q want %q (CJK width 2 + 2 pad)", got, "中文  ")
	}
}

// ---- P3-3：命名常量与被替换的魔数等价 ----

// 三种标记形式（`[ x ]` / `[X] ` / `(x) `）的宽度必须由常量算出，
// 且与 Button / Checkbox / Radio 的 Measure 一致。
func TestMarkerWidthConstantsMatchRendering(t *testing.T) {
	for _, s := range []string{"", "a", "中文", "abcdef"} {
		want := text.Width(s) + 2*markCapW + 2*markPadW
		if got := NewButton(s).Measure(looseC()).W; got != want {
			t.Errorf("Button(%q) width = %d want %d", s, got, want)
		}
		if got := NewCheckbox(s).Measure(looseC()).W; got != want {
			t.Errorf("Checkbox(%q) width = %d want %d", s, got, want)
		}
		if got := NewRadio(s).Measure(looseC()).W; got != want {
			t.Errorf("Radio(%q) width = %d want %d", s, got, want)
		}
	}
	if markCapW != 1 || markPadW != 1 {
		t.Errorf("markCapW/markPadW = %d/%d want 1/1", markCapW, markPadW)
	}
}

// 列表的滚动条预留宽与 Measure 一致。
func TestListScrollbarReserveConstant(t *testing.T) {
	l := NewList()
	l.SetItems([]string{"abcde"})
	got := l.Measure(looseC()).W
	want := text.Width("abcde") + listScrollbarW + listGapW
	if got != want {
		t.Errorf("List width = %d want %d", got, want)
	}
}

// Dialog 的最小尺寸常量。
func TestDialogMinSizeConstants(t *testing.T) {
	d := NewDialog("t", "", nil, nil)
	got := d.Measure(looseC())
	if got.W != dialogMinW {
		t.Errorf("Dialog min width = %d want %d", got.W, dialogMinW)
	}
	if got.H != dialogMinH {
		t.Errorf("Dialog min height = %d want %d", got.H, dialogMinH)
	}
}

// Tabs 的标签 padding 常量。
func TestTabsPadConstant(t *testing.T) {
	tb := NewTabs()
	tb.SetTabs([]string{"ab"})
	if got := tb.tabWidth(0); got != text.Width("ab")+2*tabPadW {
		t.Errorf("tabWidth = %d want %d", got, text.Width("ab")+2*tabPadW)
	}
	if tabPadW != 1 {
		t.Errorf("tabPadW = %d want 1", tabPadW)
	}
}

// ProgressBar / Slider / Scrollbar / TextInput / Tabs 的首选尺寸常量。
func TestPreferredSizeConstants(t *testing.T) {
	if got := NewProgressBar().Measure(looseC()); got != (geom.Size{W: progressPrefW, H: progressPrefH}) {
		t.Errorf("ProgressBar pref = %+v", got)
	}
	if progressPrefW != 20 || progressPrefH != 3 {
		t.Errorf("progress pref = %dx%d want 20x3", progressPrefW, progressPrefH)
	}
	sh := NewSlider()
	sh.SetHorizontal(true)
	if got := sh.Measure(looseC()); got != (geom.Size{W: sliderPrefW, H: sliderPrefH}) {
		t.Errorf("HSlider pref = %+v", got)
	}
	if sliderPrefW != 30 || sliderPrefH != 3 {
		t.Errorf("slider h pref = %dx%d want 30x3", sliderPrefW, sliderPrefH)
	}
	sv := NewSlider()
	if got := sv.Measure(looseC()); got != (geom.Size{W: sliderPrefVW, H: sliderPrefVH}) {
		t.Errorf("VSlider pref = %+v", got)
	}
	if sliderPrefVW != 15 || sliderPrefVH != 7 {
		t.Errorf("slider v pref = %dx%d want 15x7", sliderPrefVW, sliderPrefVH)
	}
	if got := NewTextInput().Measure(layout.Constraints{MaxW: -1, MaxH: -1}); got.W != textInputPrefW {
		t.Errorf("TextInput pref width = %d want %d", got.W, textInputPrefW)
	}
	if textInputPrefW != 17 {
		t.Errorf("textInputPrefW = %d want 17", textInputPrefW)
	}
	sep := NewSeparator()
	if got := sep.Measure(layout.Constraints{MaxW: 20, MaxH: 5}); got != (geom.Size{W: 20, H: 1}) {
		t.Errorf("Separator h = %+v", got)
	}
}

// 带框控件的最小尺寸常量。
func TestMinFrameConstants(t *testing.T) {
	if minFrameW != 3 || minFrameH != 1 {
		t.Errorf("minFrame = %dx%d want 3x1", minFrameW, minFrameH)
	}
}

// ---- P3-4：死代码清理后行为不变 ----

// Tabs.contentRect 被 Layout 真正使用（内容区 = Bounds 去掉标签行）。
func TestTabsContentRectUsedByLayout(t *testing.T) {
	tb := NewTabs()
	c := NewLabel("x")
	tb.SetTabs([]string{"a"})
	tb.SetContent(c)
	r := geom.Rect{X: 2, Y: 3, W: 10, H: 5}
	tb.Layout(r)
	if got, want := c.Bounds, tb.contentRect(); got != want {
		t.Errorf("content laid out at %+v, contentRect says %+v", got, want)
	}
	if c.Bounds != (geom.Rect{X: 2, Y: 4, W: 10, H: 4}) {
		t.Errorf("content bounds = %+v", c.Bounds)
	}
}

// Table.maxHOffset 被 setHOffset 真正使用（横向滚动钳制在上界内）。
func TestTableMaxHOffsetUsedByScroll(t *testing.T) {
	tb := NewTable()
	tb.SetColumns([]TableColumn{{Title: "A", Width: 6}, {Title: "B", Width: 6}})
	tb.SetRows([][]string{{"a", "b"}})
	tb.Layout(geom.Rect{X: 0, Y: 0, W: 8, H: 3})
	tb.OnFocus()
	for i := 0; i < 10; i++ {
		tb.Handle(event.Key{Code: event.KeyRight, Press: true})
	}
	if got := tb.HOffset(); got != tb.maxHOffset() {
		t.Errorf("HOffset = %d want maxHOffset %d", got, tb.maxHOffset())
	}
	if tb.maxHOffset() != 12-8 {
		t.Errorf("maxHOffset = %d want 4", tb.maxHOffset())
	}
}

// ---- P3-5：Base.Bind ----

// 五个此前漏调 Base.Bind 的控件：HitTest 必须返回控件自身而不是 *Base。
func TestBaseBindIdentity(t *testing.T) {
	sep := NewSeparator()
	sep.SetHorizontal(false)
	cases := []struct {
		name string
		w    Widget
		pt   geom.Point
	}{
		{"Radio", NewRadio("x"), geom.Point{X: 0, Y: 0}},
		{"ProgressBar", NewProgressBar(), geom.Point{X: 0, Y: 0}},
		{"Scrollbar", NewScrollbar(), geom.Point{X: 0, Y: 0}},
		{"Separator", sep, geom.Point{X: 0, Y: 0}},
		{"StatusBar", NewStatusBar(), geom.Point{X: 0, Y: 0}},
	}
	for _, tc := range cases {
		r := geom.Rect{X: 0, Y: 0, W: 6, H: 2}
		tc.w.Layout(r)
		hit := tc.w.HitTest(tc.pt)
		if hit == nil {
			t.Errorf("%s: HitTest returned nil inside its own Bounds", tc.name)
			continue
		}
		if hit != tc.w {
			t.Errorf("%s: HitTest = %T, want the widget itself (missing Base.Bind?)", tc.name, hit)
		}
	}
}

// ---- P3-2：Panel 的 hatch 与边框补档 ----

// Panel 的 hatch 只落在内区（不覆盖边框）。
func TestPanelHatchInsideFrame(t *testing.T) {
	p := NewPanel("t")
	p.SetHatch(screen.HatchLeft)
	r := geom.Rect{X: 0, Y: 0, W: 10, H: 5}
	buf := paintBounds(p, r)
	if p.Hatch() != screen.HatchLeft {
		t.Errorf("Hatch = %v want HatchLeft", p.Hatch())
	}
	// 边框格保持框线字形
	for _, pt := range [][2]int{{0, 0}, {9, 0}, {0, 4}, {9, 4}} {
		c, _ := buf.GetCell(pt[0], pt[1])
		if !isFrameGlyph(c.Rune) {
			t.Errorf("frame cell (%d,%d) = %q want a border glyph", pt[0], pt[1], c.Rune)
		}
	}
	// 内区出现底纹
	hatched := 0
	for y := 1; y < 4; y++ {
		for x := 1; x < 9; x++ {
			c, _ := buf.GetCell(x, y)
			if !isFrameGlyph(c.Rune) && c.Rune != ' ' {
				hatched++
			}
		}
	}
	if hatched == 0 {
		t.Error("hatch drew nothing inside the panel")
	}
}

// Panel 的边框可用 inner / outer 半块套。
func TestPanelBorderInnerOuter(t *testing.T) {
	for _, b := range []screen.Border{screen.BorderInner, screen.BorderOuter, screen.BorderThick, screen.BorderDashed} {
		p := NewPanel("t")
		p.SetBorderSpec(b)
		r := geom.Rect{X: 0, Y: 0, W: 8, H: 4}
		buf := paintBounds(p, r)
		if c, _ := buf.GetCell(0, 0); c.Rune != b.At(0, 0) {
			t.Errorf("%s top-left = %q want %q", b.Style, c.Rune, b.At(0, 0))
		}
		if c, _ := buf.GetCell(7, 3); c.Rune != b.At(2, 2) {
			t.Errorf("%s bottom-right = %q want %q", b.Style, c.Rune, b.At(2, 2))
		}
	}
}

// ---- helpers ----

// colsOf 取一行的前 n **列**（续格跳过，零值格算空格）。
func colsOf(buf *screen.Buffer, y, n int) string {
	var sb strings.Builder
	col := 0
	for x := 0; x < buf.Width() && col < n; x++ {
		c, _ := buf.GetCell(x, y)
		if c.IsContinuation() {
			continue
		}
		if c.Rune == 0 {
			sb.WriteRune(' ')
		} else {
			sb.WriteRune(c.Rune)
		}
		col++
	}
	return sb.String()
}

func isFrameGlyph(r rune) bool {
	switch r {
	case '─', '│', '┌', '┐', '└', '┘',
		'═', '║', '╔', '╗', '╚', '╝',
		'━', '┃', '┏', '┓', '┗', '┛',
		'╭', '╮', '╰', '╯',
		'-', '|', '+',
		'╌', '╎',
		'▀', '▄', '█', '▐', '▌',
		'▗', '▖', '▝', '▘', '▛', '▜', '▙', '▟':
		return true
	}
	return false
}

// Panel(BorderNone) + SetBorderEdges 不得产生空洞（无字形的边框不可见）。
func TestPanelBorderlessWithEdgesHasNoHoles(t *testing.T) {
	for _, bs := range []BorderStyle{BorderNone, BorderSingle} {
		p := NewPanel("t")
		p.SetBorder(bs)
		// 有背景可铺时必须铺满（零样式 = 透明，见 paintBackground）。
		p.SetStyle(style.Style{}.WithBackground(style.ANSIColor(238)))
		p.SetBorderEdges(screen.BorderEdges{Top: true})
		r := geom.Rect{X: 0, Y: 0, W: 8, H: 4}
		buf := paintBounds(p, r)
		for y := 0; y < 4; y++ {
			for x := 0; x < 8; x++ {
				c, _ := buf.GetCell(x, y)
				if c.Rune == 0 && c.Width == 0 {
					t.Errorf("border %v: hole at (%d,%d)", bs, x, y)
				}
			}
		}
	}
}
