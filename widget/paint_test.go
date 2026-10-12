package widget

import (
	"strings"
	"testing"

	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/style"
	"PaintedSkin/text"
	"PaintedSkin/widget/layout"
)

// 本文件是"尾部空档纪律"的可执行验收（docs/05 / docs/11 P0）：
//
// 每格要么被写入真实内容，要么被写入"控件自己的背景色空格"，不存在第三种状态。
// 未被写入的格是零值 Cell{Rune:0, Width:0}，帧尾会被 lineDiff 写成终端
// 默认背景的空格 —— 这在视觉上就是"控件没画满"。

// paintBounds 把控件画进一块真实内存缓冲（不用 mock，直接看格）。
func paintBounds(w Widget, r geom.Rect) *screen.Buffer {
	buf := screen.NewBuffer(r.Right(), r.Bottom())
	w.Layout(r)
	w.Paint(buf)
	return buf
}

// boundsOf 取控件实际占用的区域（Dialog 会在给定区域内居中，Bounds 更小）。
func boundsOf(w Widget) geom.Rect {
	if b, ok := w.(interface{ BoundsOf() geom.Rect }); ok {
		return b.BoundsOf()
	}
	return w.(*Label).Bounds
}

// assertNoHoles 断言 rect 内不存在未写入的格。
func assertNoHoles(t *testing.T, buf *screen.Buffer, rect geom.Rect, name string) {
	t.Helper()
	for y := rect.Y; y < rect.Bottom(); y++ {
		for x := rect.X; x < rect.Right(); x++ {
			c, ok := buf.GetCell(x, y)
			if !ok {
				t.Fatalf("%s: cell (%d,%d) outside buffer", name, x, y)
			}
			if c.Rune == 0 && c.Width == 0 {
				t.Errorf("%s: hole at (%d,%d) — never painted", name, x, y)
			}
		}
	}
}

// 每个控件画完后，其 Bounds 内不得留空洞。
func TestWidgetsLeaveNoHoles(t *testing.T) {
	focusedBtn := NewButton("OK")
	focusedBtn.Layout(geom.Rect{X: 0, Y: 0, W: 12, H: 1})
	focusedBtn.OnFocus()

	focusedChk := NewCheckbox("选项")
	focusedChk.Layout(geom.Rect{X: 0, Y: 0, W: 14, H: 1})
	focusedChk.OnFocus()

	hoverRad := NewRadio("选项")
	hoverRad.Layout(geom.Rect{X: 0, Y: 0, W: 14, H: 1})
	hoverRad.Handle(eventMouseEnter())

	shortList := NewList()
	shortList.SetItems([]string{"a", "b"})

	longList := NewList()
	longList.SetItems([]string{"甲", "乙", "丙", "丁", "戊", "己", "庚", "辛"})

	emptyList := NewList()
	emptyList.SetItems([]string{"only"})

	emptyDrop := NewDropdown()

	drop := NewDropdown()
	drop.SetItems([]string{"中文"})

	dropLayer := newDropdownList([]string{"中文", "b"}, 0, drop)

	bar := NewStatusBar()
	bar.SetLeft("L")
	bar.SetCenter("C")
	bar.SetRight("R")

	barCJK := NewStatusBar()
	barCJK.SetLeft("状态")
	barCJK.SetCenter("中")
	barCJK.SetRight("右侧")

	barOver := NewStatusBar()
	barOver.SetLeft("very long left segment")
	barOver.SetCenter("very long center segment")
	barOver.SetRight("very long right segment")

	rightLabel := NewLabel("ab")
	rightLabel.SetAlign(layout.AlignEnd)

	centerLabel := NewLabel("中文")
	centerLabel.SetAlign(layout.AlignCenter)

	overLabel := NewLabel("这段文字远宽于控件自身区域")

	panel := NewPanel("标题")
	panel.SetChild(NewLabel("内容"))

	panelTight := NewPanel("很长的面板标题 mixed")
	panelTight.Layout(geom.Rect{X: 0, Y: 0, W: 14, H: 5})

	dialog := NewDialog("标题", "内容", []string{"OK"}, nil)
	dialog.Layout(geom.Rect{X: 0, Y: 0, W: 30, H: 10})

	dialogDim := NewDialog("标题", "内容", nil, nil)
	dialogDim.SetDim(true)
	dialogDim.Layout(geom.Rect{X: 0, Y: 0, W: 30, H: 10})

	sliderH := NewSlider()
	sliderH.SetHorizontal(true)
	sliderH.SetRange(0, 100)
	sliderH.SetValue(40)

	sliderV := NewSlider()
	sliderV.SetHorizontal(false)
	sliderV.SetRange(0, 100)
	sliderV.SetValue(40)

	scrollbar := NewScrollbar()
	scrollbar.SetTotal(100)
	scrollbar.SetView(10)
	scrollbar.SetOffset(30)

	scrollbarH := NewScrollbar()
	scrollbarH.SetHorizontal(true)
	scrollbarH.SetTotal(100)
	scrollbarH.SetView(10)

	table := NewTable()
	table.SetColumns([]TableColumn{{Title: "A", Width: 4}, {Title: "B", Width: 4}})
	table.SetRows([][]string{{"a", "b"}, {"c", "d"}})

	tableWide := NewTable()
	tableWide.SetColumns([]TableColumn{{Title: "A", Width: 4}})
	tableWide.SetRows([][]string{{"a"}, {"b"}})

	tabs := NewTabs()
	tabs.SetTabs([]string{"一", "二"})
	tabs.SetContent(NewLabel("body"))

	group := NewRadioGroup("g", []string{"甲", "乙"})

	cases := []struct {
		name string
		w    Widget
		r    geom.Rect
	}{
		{"Label", NewLabel("中文ab"), geom.Rect{X: 0, Y: 0, W: 10, H: 1}},
		{"LabelRight", rightLabel, geom.Rect{X: 0, Y: 0, W: 10, H: 1}},
		{"LabelCenter", centerLabel, geom.Rect{X: 0, Y: 0, W: 10, H: 1}},
		{"LabelOverlong", overLabel, geom.Rect{X: 0, Y: 0, W: 6, H: 1}},
		{"ButtonFocused", focusedBtn, geom.Rect{X: 0, Y: 0, W: 12, H: 1}},
		{"CheckboxFocused", focusedChk, geom.Rect{X: 0, Y: 0, W: 14, H: 1}},
		{"RadioHover", hoverRad, geom.Rect{X: 0, Y: 0, W: 14, H: 1}},
		{"ListShort", shortList, geom.Rect{X: 0, Y: 0, W: 10, H: 3}},
		{"ListScrolled", longList, geom.Rect{X: 0, Y: 0, W: 10, H: 2}},
		{"ListShortRows", emptyList, geom.Rect{X: 0, Y: 0, W: 10, H: 4}},
		{"DropdownEmpty", emptyDrop, geom.Rect{X: 0, Y: 0, W: 4, H: 1}},
		{"DropdownCJK", drop, geom.Rect{X: 0, Y: 0, W: 7, H: 1}},
		{"DropdownLayer", dropLayer, geom.Rect{X: 0, Y: 0, W: 6, H: 3}},
		{"StatusBar", bar, geom.Rect{X: 0, Y: 0, W: 20, H: 1}},
		{"StatusBarCJK", barCJK, geom.Rect{X: 0, Y: 0, W: 20, H: 1}},
		{"StatusBarOverflow", barOver, geom.Rect{X: 0, Y: 0, W: 20, H: 1}},
		{"Panel", panel, geom.Rect{X: 0, Y: 0, W: 20, H: 8}},
		{"PanelTightTitle", panelTight, geom.Rect{X: 0, Y: 0, W: 14, H: 5}},

		// 其余控件同样必须铺满自己的 Bounds（docs/08 通用绘制纪律）。
		{"Separator", NewSeparator(), geom.Rect{X: 0, Y: 0, W: 8, H: 1}},
		{"SeparatorV", func() Widget { s := NewSeparator(); s.SetHorizontal(false); return s }(),
			geom.Rect{X: 0, Y: 0, W: 1, H: 4}},
		{"SliderH", sliderH, geom.Rect{X: 0, Y: 0, W: 16, H: 3}},
		{"SliderV", sliderV, geom.Rect{X: 0, Y: 0, W: 15, H: 7}},
		{"Progress1", NewProgressBar(), geom.Rect{X: 0, Y: 0, W: 12, H: 1}},
		{"Progress3", NewProgressBar(), geom.Rect{X: 0, Y: 0, W: 12, H: 3}},
		{"Scrollbar", scrollbar, geom.Rect{X: 0, Y: 0, W: 1, H: 6}},
		{"ScrollbarH", scrollbarH, geom.Rect{X: 0, Y: 0, W: 6, H: 1}},
		{"TextInput", NewTextInput(), geom.Rect{X: 0, Y: 0, W: 12, H: 1}},
		{"TextInputTall", NewTextInput(), geom.Rect{X: 0, Y: 0, W: 12, H: 3}},
		{"Table", table, geom.Rect{X: 0, Y: 0, W: 16, H: 5}},
		{"TableWide", tableWide, geom.Rect{X: 0, Y: 0, W: 24, H: 4}},
		{"Tabs", tabs, geom.Rect{X: 0, Y: 0, W: 16, H: 3}},
		{"RadioGroup", group, geom.Rect{X: 0, Y: 0, W: 14, H: 2}},
		{"Dialog", dialog, geom.Rect{X: 0, Y: 0, W: 30, H: 10}},
		{"DialogDim", dialogDim, geom.Rect{X: 0, Y: 0, W: 30, H: 10}},
	}
	// 铺满纪律的适用场景是"控件有背景可铺"：零样式 = 透明（见 paintBackground），
	// 那种情况下控件不写背景，底层（容器底色或终端默认）自然透上来。
	bg := style.Style{}.WithBackground(style.ANSIColor(238))
	for _, tc := range cases {
		if st, ok := tc.w.(interface{ SetStyle(style.Style) }); ok {
			st.SetStyle(bg)
		}
		buf := paintBounds(tc.w, tc.r)
		// 断言控件**实际**占用的区域（Dialog 会居中，Bounds 小于给定区域）。
		assertNoHoles(t, buf, boundsOf(tc.w), tc.name)
	}
}

// 控件自有背景色必须到达尾部空档（这是"没空洞"的实际收益：底色连成一片）。
func TestWidgetBackgroundReachesTrailingGap(t *testing.T) {
	bg := style.Style{}.WithBackground(style.ANSIColor(238))
	l := NewLabel("ab")
	l.Style = bg
	buf := paintBounds(l, geom.Rect{X: 0, Y: 0, W: 10, H: 1})
	for x := 0; x < 10; x++ {
		c, _ := buf.GetCell(x, 0)
		if c.Style.BG != bg.BG {
			t.Errorf("column %d bg = %+v want own background", x, c.Style.BG)
		}
	}
}

// 焦点态的反色带必须覆盖整个控件宽度，而不是停在文字末尾。
func TestButtonFocusBandCoversWidth(t *testing.T) {
	b := NewButton("OK")
	b.Layout(geom.Rect{X: 0, Y: 0, W: 12, H: 1})
	b.OnFocus()
	buf := paintBounds(b, geom.Rect{X: 0, Y: 0, W: 12, H: 1})
	for x := 0; x < 12; x++ {
		c, _ := buf.GetCell(x, 0)
		if !c.Style.Reverse {
			t.Errorf("column %d = %+v want focus band to cover full width", x, c)
		}
	}
}

// Checkbox / Radio 的焦点/悬停态同样覆盖整宽。
func TestToggleFocusBandCoversWidth(t *testing.T) {
	c := NewCheckbox("ok")
	c.Layout(geom.Rect{X: 0, Y: 0, W: 14, H: 1})
	c.OnFocus()
	buf := paintBounds(c, geom.Rect{X: 0, Y: 0, W: 14, H: 1})
	for x := 0; x < 14; x++ {
		if cell, _ := buf.GetCell(x, 0); !cell.Style.Reverse {
			t.Errorf("checkbox column %d = %+v want reverse band", x, cell)
		}
	}

	r := NewRadio("ok")
	r.Layout(geom.Rect{X: 0, Y: 0, W: 14, H: 1})
	r.Handle(eventMouseEnter())
	buf2 := paintBounds(r, geom.Rect{X: 0, Y: 0, W: 14, H: 1})
	for x := 0; x < 14; x++ {
		if cell, _ := buf2.GetCell(x, 0); !cell.Style.Bold {
			t.Errorf("radio column %d = %+v want hover band", x, cell)
		}
	}
}

// List 选中行的反色必须盖住整行文本区（到滚动条列为止）。
func TestListSelectedRowCoversTextWidth(t *testing.T) {
	l := NewList()
	l.SetItems([]string{"a", "b"})
	l.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 2})
	l.SetSelected(0)
	l.OnFocus()
	buf := paintBounds(l, geom.Rect{X: 0, Y: 0, W: 10, H: 2})
	for x := 0; x < 9; x++ { // 右侧 1 列预留给滚动条
		c, _ := buf.GetCell(x, 0)
		if !c.Style.Reverse {
			t.Errorf("selected row column %d = %+v want reverse to row end", x, c)
		}
	}
}

// List 的空行（项数少于视口）也必须铺底（有背景可铺时）。
func TestListEmptyRowsPainted(t *testing.T) {
	bg := style.ANSIColor(238)
	l := NewList()
	l.SetItems([]string{"only"})
	l.SetStyle(style.Style{}.WithBackground(bg))
	l.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 4})
	buf := paintBounds(l, geom.Rect{X: 0, Y: 0, W: 10, H: 4})
	for y := 1; y < 4; y++ {
		for x := 0; x < 10; x++ {
			c, _ := buf.GetCell(x, y)
			if c.Rune != ' ' || c.Width != 1 {
				t.Errorf("empty row (%d,%d) = %+v want blank", x, y, c)
			}
			if c.Style.BG != bg {
				t.Errorf("empty row (%d,%d) bg = %+v want %v", x, y, c.Style.BG, bg)
			}
		}
	}
}

// 零样式控件是**透明**的：不铺底，容器底色 / 终端默认自然透上来。
// 这是尾部空档纪律的边界：纪律约束的是"有背景可铺时必须铺满"。
func TestZeroStyleWidgetsAreTransparent(t *testing.T) {
	bg := style.Style{}.WithBackground(style.ANSIColor(238))
	sl := NewSlider()
	sl.SetHorizontal(true)
	r := geom.Rect{X: 0, Y: 0, W: 12, H: 3}
	buf := paintBounds(sl, r)
	// 未设样式 → 什么都不铺
	if c, _ := buf.GetCell(0, 1); c.Rune != 0 {
		t.Errorf("zero-style widget painted %q, want transparent", c.Rune)
	}
	// 同一个控件设了背景 → 铺满
	sl.SetStyle(bg)
	buf2 := paintBounds(sl, r)
	for y := 0; y < 3; y++ {
		for x := 0; x < 12; x++ {
			c, _ := buf2.GetCell(x, y)
			if c.Style.BG != bg.BG {
				t.Errorf("(%d,%d) bg = %+v want %v", x, y, c.Style.BG, bg.BG)
			}
		}
	}
}

// Dropdown 关闭态必须铺满 Measure 报出的宽度（空选时曾经差 2 列）。
func TestDropdownClosedFillsMeasureWidth(t *testing.T) {
	d := NewDropdown()
	want := d.Measure(layout.Constraints{MaxW: 40, MaxH: 24}).W
	buf := paintBounds(d, geom.Rect{X: 0, Y: 0, W: want, H: 1})
	assertNoHoles(t, buf, geom.Rect{X: 0, Y: 0, W: want, H: 1}, "dropdown")
}

// StatusBar 三段必须互不重叠，且各自不越界。
func TestStatusBarSegmentsDoNotOverlap(t *testing.T) {
	cases := []struct {
		left, center, right string
		w                   int
	}{
		{"L", "C", "R", 20},
		{"状态", "中", "右侧", 20},
		{"", "C", "", 10},
		{"a very long left segment", "center", "right", 20},
		{"very long left segment here", "a very long center segment", "and a long right one", 20},
		{"中文中文中文中文中文", "中中中中中中中", "右右右", 10},
	}
	for _, tc := range cases {
		s := NewStatusBar()
		s.SetLeft(tc.left)
		s.SetCenter(tc.center)
		s.SetRight(tc.right)
		mc := newMemCanvas(tc.w, 1)
		s.Layout(geom.Rect{X: 0, Y: 0, W: tc.w, H: 1})
		s.Paint(mc)
		type span struct{ lo, hi int }
		var spans []span
		for _, call := range mc.texts {
			w := text.Width(call.s)
			if w == 0 {
				continue
			}
			if call.x < 0 || call.x+w > tc.w {
				t.Errorf("%q/%q/%q@%d: %q overruns bar width %d", tc.left, tc.center, tc.right, tc.w, call.s, tc.w)
			}
			for _, sp := range spans {
				if call.x < sp.hi && sp.lo < call.x+w {
					t.Errorf("%q/%q/%q@%d: %q overlaps another segment", tc.left, tc.center, tc.right, tc.w, call.s)
				}
			}
			spans = append(spans, span{call.x, call.x + w})
		}
	}
}

// ---- P0-2：Panel / Dialog 必须填内区与 padding 环 ----

// Panel 的内区与 padding 环必须铺控件自身背景（不再是"只有边框"）。
func TestPanelFillsInteriorAndPaddingRing(t *testing.T) {
	p := NewPanel("标题")
	p.SetStyle(style.Style{}.WithBackground(style.ANSIColor(238)))
	p.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 8})
	buf := paintBounds(p, geom.Rect{X: 0, Y: 0, W: 20, H: 8})

	// padding 环（左列 + 底行）必须是带底色的空格。
	for y := 1; y < 7; y++ {
		c, _ := buf.GetCell(1, y)
		if c.Rune != ' ' || c.Width != 1 {
			t.Errorf("padding column (1,%d) = %+v want blank", y, c)
		}
	}
	for x := 1; x < 19; x++ {
		c, _ := buf.GetCell(x, 6)
		if c.Rune != ' ' || c.Width != 1 {
			t.Errorf("padding row (%d,6) = %+v want blank", x, c)
		}
	}
}

// Panel 的内区不得残留未绘制的格（无边框时同样要铺满）。
func TestPanelBorderlessStillFills(t *testing.T) {
	p := NewPanel("")
	p.SetBorder(BorderNone)
	p.SetStyle(style.Style{}.WithBackground(style.ANSIColor(238)))
	p.SetChild(NewLabel("内容"))
	p.Layout(geom.Rect{X: 0, Y: 0, W: 12, H: 4})
	buf := paintBounds(p, geom.Rect{X: 0, Y: 0, W: 12, H: 4})
	assertNoHoles(t, buf, geom.Rect{X: 0, Y: 0, W: 12, H: 4}, "borderless panel")
}

// Dialog 非 dim 时也必须铺内区（从"仅 dim 才 Fill"改为"总是铺底"）。
func TestDialogFillsInteriorWithoutDim(t *testing.T) {
	d := NewDialog("标题", "内容", []string{"OK"}, nil)
	d.SetStyle(style.Style{}.WithBackground(style.ANSIColor(238)))
	d.Layout(geom.Rect{X: 0, Y: 0, W: 30, H: 10})
	buf := paintBounds(d, geom.Rect{X: 0, Y: 0, W: 30, H: 10})
	assertNoHoles(t, buf, d.Bounds, "dialog")
	c, _ := buf.GetCell(d.Bounds.X+5, d.Bounds.Y+4)
	if c.Rune != ' ' {
		t.Errorf("interior = %+v want blank (dim is off)", c)
	}
}

// Dialog 开启遮罩时：内区铺底且带暗淡属性。
func TestDialogDimFillsWithDim(t *testing.T) {
	d := NewDialog("t", "m", nil, nil)
	d.SetStyle(style.Style{}.WithBackground(style.ANSIColor(238)))
	d.SetDim(true)
	d.Layout(geom.Rect{X: 0, Y: 0, W: 30, H: 10})
	buf := paintBounds(d, geom.Rect{X: 0, Y: 0, W: 30, H: 10})
	assertNoHoles(t, buf, d.Bounds, "dialog-dim")
	c, _ := buf.GetCell(d.Bounds.X+5, d.Bounds.Y+5)
	if !c.Style.Dim {
		t.Errorf("interior = %+v want dim", c.Style)
	}
}

// ---- P0-3：框体标题必须进框内、按框宽截断、绝不越框 ----

// 标题列预算：左右各留 1 格空格 + 2 格边框。
func titleBudget(w int) int { return w - 4 }

// Panel 标题超长时必须截断，且右边框完好。
func TestPanelTitleTruncatedInsideFrame(t *testing.T) {
	const w, h = 16, 6
	p := NewPanel("这是一个非常非常长的标题 mixed text")
	p.Layout(geom.Rect{X: 0, Y: 0, W: w, H: h})
	buf := paintBounds(p, geom.Rect{X: 0, Y: 0, W: w, H: h})
	assertFrameIntact(t, buf, p.Bounds, "panel long title")
}

// Dialog 标题必须落在框内首行（不压顶边框），且按框宽截断。
func TestDialogTitleInsideFrame(t *testing.T) {
	area := geom.Rect{X: 0, Y: 0, W: 40, H: 20}
	d := NewDialog("这是一个非常非常长的标题 mixed", "内容", nil, nil)
	d.Layout(area)

	mc := newMemCanvas(area.Right(), area.Bottom())
	d.Paint(mc)
	for _, call := range mc.texts {
		if call.y == d.Bounds.Y {
			t.Errorf("nothing may paint on the top border row: %+v", call)
		}
	}
	buf := screen.NewBuffer(area.Right(), area.Bottom())
	d.Paint(buf)
	assertFrameIntact(t, buf, d.Bounds, "dialog long title")
}

// Panel / Dialog 标题统一策略：同一列预算、同一行号偏移、都截断、都加粗。
func TestPanelAndDialogShareTitleStrategy(t *testing.T) {
	const w, h = 18, 8
	title := "统一标题策略 mixed"

	p := NewPanel(title)
	p.Layout(geom.Rect{X: 0, Y: 0, W: w, H: h})
	mcP := newMemCanvas(w, h)
	p.Paint(mcP)

	d := NewDialog(title, "msg", nil, nil)
	d.Layout(geom.Rect{X: 0, Y: 0, W: w, H: h})
	mcD := newMemCanvas(w, h)
	d.Paint(mcD)

	// 标题落在边框内侧首行（Panel 与 Dialog 相同）。
	tp, ok := textOnRow(mcP, p.Bounds.Y+1)
	if !ok {
		t.Fatalf("panel title row missing: %+v", mcP.texts)
	}
	td, ok := textOnRow(mcD, d.Bounds.Y+1)
	if !ok {
		t.Fatalf("dialog title row missing: %+v", mcD.texts)
	}
	if tp.x != td.x {
		t.Errorf("title x differs: panel %d dialog %d", tp.x, td.x)
	}
	if !tp.st.Bold || !td.st.Bold {
		t.Errorf("titles must be bold: panel %+v dialog %+v", tp.st, td.st)
	}
	if text.Width(tp.s) > titleBudget(w) || text.Width(td.s) > titleBudget(w) {
		t.Errorf("title exceeds budget %d: panel %d dialog %d",
			titleBudget(w), text.Width(tp.s), text.Width(td.s))
	}
	// 标题内容必须是原文的前缀（按字符边界截断，不劈宽字符）。
	for _, got := range []string{tp.s, td.s} {
		if !strings.HasPrefix(title, got) {
			t.Errorf("title %q is not a prefix of %q", got, title)
		}
	}
}

// 退化框宽（<= 4）时标题必须为空串，不得画出任何内容。
func TestTitleDegenerateFrame(t *testing.T) {
	for _, w := range []int{0, 1, 2, 3, 4} {
		p := NewPanel("标题")
		p.Layout(geom.Rect{X: 0, Y: 0, W: w, H: 3})
		mc := newMemCanvas(w+1, 3)
		p.Paint(mc)
		if call, ok := textOnRow(mc, 1); ok && call.s != "" {
			t.Errorf("width %d: title = %q want empty", w, call.s)
		}
	}
}

// 退化框高（容不下框内首行）时标题必须让位给边框。
func TestTitleDegenerateHeight(t *testing.T) {
	for _, h := range []int{1, 2, 3} {
		p := NewPanel("标题")
		r := geom.Rect{X: 0, Y: 0, W: 16, H: h}
		p.Layout(r)
		buf := paintBounds(p, r)
		mc := newMemCanvas(r.Right(), r.Bottom())
		p.Paint(mc)
		// 高度容不下框内首行时不得画标题。
		if h <= 2 {
			if call, ok := textOnRow(mc, 1); ok && call.s != "" {
				t.Errorf("height %d: title = %q must yield to the border", h, call.s)
			}
		}
		// 每一行的最右可见字符仍必须是边框（H=1 时退化为横线）。
		for y := 0; y < h; y++ {
			c, _ := buf.GetCell(r.Right()-1, y)
			switch c.Rune {
			case '┐', '┘', '│', '─':
			default:
				t.Errorf("height %d row %d right = %q want border", h, y, c.Rune)
			}
		}
	}
}

// textOnRow 返回该行上的文本调用（可能有多段，取第一段）。
func textOnRow(mc *memCanvas, y int) (textCall, bool) {
	for _, call := range mc.texts {
		if call.y == y {
			return call, true
		}
	}
	return textCall{}, false
}

// assertFrameIntact 断言框体每一行的最右可见字符仍是右边框（角在首末行）。
func assertFrameIntact(t *testing.T, buf *screen.Buffer, frame geom.Rect, name string) {
	t.Helper()
	for y := frame.Y; y < frame.Bottom(); y++ {
		c, _ := buf.GetCell(frame.Right()-1, y)
		switch y {
		case frame.Y:
			if c.Rune != '┐' {
				t.Errorf("%s: row %d right = %q want ┐", name, y, c.Rune)
			}
		case frame.Bottom() - 1:
			if c.Rune != '┘' {
				t.Errorf("%s: row %d right = %q want ┘", name, y, c.Rune)
			}
		default:
			if c.Rune != '│' {
				t.Errorf("%s: row %d right border broken = %q", name, y, c.Rune)
			}
		}
	}
}
