package widget

import (
	"strings"
	"testing"

	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/text"
	"PaintedSkin/theme"
)

// P2：outline（不占布局的焦点环）、Dropdown 弹层补框、Dialog.SetBorder、
// TextInput/ProgressBar 用 BorderStyle 枚举、启用 Clip。

// ---- outline 接线 ----

// 焦点环只换样式：不改字符、不改宽度、不改布局。
func TestFocusRingKeepsContent(t *testing.T) {
	ti := NewTextInput()
	ti.SetText("abc")
	r := geom.Rect{X: 0, Y: 0, W: 10, H: 3}
	ti.OnFocus()
	before := rowOf(paintBounds(ti, r), 0)

	// 开/关焦点环，同一聚焦态下内容必须逐字相同。
	ti.SetFocusRing(false)
	noRing := rowOf(paintBounds(ti, r), 0)
	ti.SetFocusRing(true)
	withRing := rowOf(paintBounds(ti, r), 0)
	if before != noRing || noRing != withRing {
		t.Errorf("focus ring changed content: %q / %q / %q", before, noRing, withRing)
	}
	// Measure 不受影响（不占布局）。
	m := ti.Measure(looseC())
	ti.SetFocusRing(true)
	if m != ti.Measure(looseC()) {
		t.Error("focus ring must not change Measure")
	}
}

// 焦点环覆盖外圈（首末行与首末列）。
func TestFocusRingCoversBorder(t *testing.T) {
	ti := NewTextInput()
	ti.SetText("abc")
	ti.SetFocusRing(true)
	ti.OnFocus()
	buf := paintBounds(ti, geom.Rect{X: 0, Y: 0, W: 10, H: 3})
	ring := func(x, y int) bool {
		return y == 0 || y == 2 || x == 0 || x == 9
	}
	for y := 0; y < 3; y++ {
		for x := 0; x < 10; x++ {
			c, _ := buf.GetCell(x, y)
			if got := c.Style.Underline; got != ring(x, y) {
				t.Errorf("(%d,%d) underline = %v want %v", x, y, got, ring(x, y))
			}
		}
	}
}

// 关掉焦点环后不画（默认关闭，保持既有观感）。
func TestFocusRingOffByDefault(t *testing.T) {
	ti := NewTextInput()
	if ti.FocusRingEnabled() {
		t.Error("focus ring must be opt-in")
	}
	ti.OnFocus()
	buf := paintBounds(ti, geom.Rect{X: 0, Y: 0, W: 10, H: 3})
	if c, _ := buf.GetCell(0, 0); c.Style.Underline {
		t.Error("no ring by default")
	}
	ti.SetFocusRing(true)
	if !ti.FocusRingEnabled() {
		t.Error("SetFocusRing(true) must take effect")
	}
}

// 焦点环的样式随焦点：控件自身 Style 优先，其次主题 FocusColor。
func TestFocusRingUsesFocusStyle(t *testing.T) {
	ti := NewTextInput()
	ti.SetTheme(ThemeFrom(onlyTheme(func(th *theme.Theme) { th.FocusColor = mark })))
	ti.SetFocusRing(true)
	ti.OnFocus()
	buf := paintBounds(ti, geom.Rect{X: 0, Y: 0, W: 10, H: 3})
	c, _ := buf.GetCell(0, 0)
	if c.Style.FG != mark {
		t.Errorf("focus ring style = %+v want FocusColor %v", c.Style, mark)
	}
}

// ---- P2-3：Dropdown 弹层补框 ----

// 弹层必须有边框（此前注释承诺"边框 + 选项"但只画了 Fill）。
func TestDropdownLayerHasBorder(t *testing.T) {
	d := NewDropdown()
	l := newDropdownList([]string{"中文", "b"}, 0, d)
	r := geom.Rect{X: 0, Y: 0, W: 8, H: 3}
	buf := paintBounds(l, r)
	// 四角 + 边
	corners := [][2]rune{{'┌', '┐'}, {'└', '┘'}}
	_ = corners
	if c, _ := buf.GetCell(0, 0); c.Rune != '┌' {
		t.Errorf("top-left = %q want ┌", c.Rune)
	}
	if c, _ := buf.GetCell(7, 0); c.Rune != '┐' {
		t.Errorf("top-right = %q want ┐", c.Rune)
	}
	if c, _ := buf.GetCell(0, 2); c.Rune != '└' {
		t.Errorf("bottom-left = %q want └", c.Rune)
	}
	if c, _ := buf.GetCell(7, 2); c.Rune != '┘' {
		t.Errorf("bottom-right = %q want ┘", c.Rune)
	}
	for x := 1; x < 7; x++ {
		for _, y := range []int{0, 2} {
			c, _ := buf.GetCell(x, y)
			if c.Rune != '─' {
				t.Errorf("h edge (%d,%d) = %q want ─", x, y, c.Rune)
			}
		}
	}
	for y := 1; y < 2; y++ {
		for _, x := range []int{0, 7} {
			c, _ := buf.GetCell(x, y)
			if c.Rune != '│' {
				t.Errorf("v edge (%d,%d) = %q want │", x, y, c.Rune)
			}
		}
	}
}

// 弹层边框吃掉 1 格：选项只能画在框内，且不得越框。
func TestDropdownLayerContentInsideFrame(t *testing.T) {
	d := NewDropdown()
	l := newDropdownList([]string{"中文选项", "b"}, 0, d)
	r := geom.Rect{X: 0, Y: 0, W: 10, H: 3}
	buf := paintBounds(l, r)
	// 中间行必须是框内（首列边框 + 选项 + 空格 + 末列边框）
	mid := rowOf(buf, 1)
	if !strings.HasPrefix(mid, "│") || !strings.HasSuffix(mid, "│") {
		t.Errorf("middle row = %q want │…│", mid)
	}
	if text.Width(mid) != 10 {
		t.Errorf("middle row width = %d want 10 (%q)", text.Width(mid), mid)
	}
	// 边框内不得被选项覆盖
	for y := 0; y < 3; y++ {
		for _, x := range []int{0, 9} {
			c, _ := buf.GetCell(x, y)
			if c.Rune != '│' && c.Rune != '┌' && c.Rune != '┐' && c.Rune != '└' && c.Rune != '┘' {
				t.Errorf("frame cell (%d,%d) = %q overwritten", x, y, c.Rune)
			}
		}
	}
}

// 弹层 Measure 要为边框留出 2 格（否则框会吃掉选项）。
func TestDropdownLayerMeasureReservesFrame(t *testing.T) {
	d := NewDropdown()
	l := newDropdownList([]string{"中文", "abcd"}, 0, d)
	got := l.Measure(looseC())
	wantW := text.Width("abcd") + 2 // 原来 +2 → 现在左右边框各 1
	if got.W != wantW {
		t.Errorf("width = %d want %d", got.W, wantW)
	}
}

// 退化尺寸（高 1 / 宽 1）不 panic。
func TestDropdownLayerDegenerate(t *testing.T) {
	d := NewDropdown()
	l := newDropdownList([]string{"a"}, 0, d)
	for _, size := range []geom.Size{{W: 1, H: 1}, {W: 6, H: 1}, {W: 1, H: 3}, {W: 0, H: 0}} {
		buf := paintBounds(l, geom.Rect{X: 0, Y: 0, W: size.W, H: size.H})
		_ = buf
	}
}

// ---- P2-3：Dialog.SetBorder ----

func TestDialogSetBorder(t *testing.T) {
	d := NewDialog("t", "m", nil, nil)
	if d.Border() != BorderSingle {
		t.Errorf("default border = %v want BorderSingle", d.Border())
	}
	d.SetBorder(BorderDouble)
	if d.Border() != BorderDouble {
		t.Errorf("Border = %v want BorderDouble", d.Border())
	}
	d.Layout(geom.Rect{X: 0, Y: 0, W: 24, H: 10})
	buf := screen.NewBuffer(24, 10)
	d.Paint(buf)
	if c, _ := buf.GetCell(d.Bounds.X, d.Bounds.Y); c.Rune != '╔' {
		t.Errorf("dialog top-left = %q want ╔", c.Rune)
	}
	// 逐边开关
	d.SetBorderEdges(screen.BorderEdges{Top: true})
	if d.BorderEdges() != (screen.BorderEdges{Top: true}) {
		t.Errorf("BorderEdges = %+v", d.BorderEdges())
	}
	buf2 := screen.NewBuffer(24, 10)
	d.Paint(buf2)
	if c, _ := buf2.GetCell(d.Bounds.X, d.Bounds.Bottom()-1); c.Rune == '╚' {
		t.Error("bottom edge must be off")
	}
	// 任意字形集
	d.SetBorderSpec(screen.BorderThick)
	if c := d.BorderSpec(); !c.Visible() || c.Style != "thick" {
		t.Errorf("BorderSpec = %+v want thick", c)
	}
}

// ---- P2-3：TextInput / ProgressBar 用 BorderStyle 枚举 ----

// TextInput / ProgressBar 的边框走 BorderStyle 枚举（此前直接传 style.AttrBold）。
func TestTextInputAndProgressUseBorderEnum(t *testing.T) {
	ti := NewTextInput()
	ti.SetBorder(BorderRound)
	if ti.Border() != BorderRound {
		t.Errorf("textinput Border = %v", ti.Border())
	}
	buf := paintBounds(ti, geom.Rect{X: 0, Y: 0, W: 8, H: 3})
	if c, _ := buf.GetCell(0, 0); c.Rune != '╭' {
		t.Errorf("textinput top-left = %q want ╭", c.Rune)
	}

	pb := NewProgressBar()
	pb.SetBorder(BorderDouble)
	if pb.Border() != BorderDouble {
		t.Errorf("progress Border = %v", pb.Border())
	}
	buf2 := paintBounds(pb, geom.Rect{X: 0, Y: 0, W: 8, H: 3})
	if c, _ := buf2.GetCell(0, 0); c.Rune != '╔' {
		t.Errorf("progress top-left = %q want ╔", c.Rune)
	}
}

// 默认边框与改动前视觉一致（TextInput/ProgressBar 都是粗线）。
func TestTextInputAndProgressDefaultBorder(t *testing.T) {
	ti := NewTextInput()
	buf := paintBounds(ti, geom.Rect{X: 0, Y: 0, W: 8, H: 3})
	if c, _ := buf.GetCell(0, 0); c.Rune != screen.BorderHeavy.At(0, 0) {
		t.Errorf("textinput default border = %q", c.Rune)
	}
	pb := NewProgressBar()
	buf2 := paintBounds(pb, geom.Rect{X: 0, Y: 0, W: 8, H: 3})
	if c, _ := buf2.GetCell(0, 0); c.Rune != screen.BorderHeavy.At(0, 0) {
		t.Errorf("progress default border = %q", c.Rune)
	}
}

// TextInput 的 Measure 必须把边框宽度算进去（换边框不改变行数）。
func TestTextInputBorderDoesNotChangeMeasure(t *testing.T) {
	ti := NewTextInput()
	before := ti.Measure(looseC())
	ti.SetBorder(BorderDouble)
	after := ti.Measure(looseC())
	if before != after {
		t.Errorf("Measure changed with border: %+v -> %+v", before, after)
	}
	_ = after
}

// ---- P2-4：启用 Clip，绘制受 Bounds 约束 ----

// Table 横向滚动时最后一列不得超出 Bounds。
func TestTableHorizontalScrollStaysInBounds(t *testing.T) {
	tb := NewTable()
	tb.SetColumns([]TableColumn{
		{Title: "A", Width: 6}, {Title: "B", Width: 6}, {Title: "C", Width: 6},
	})
	tb.SetRows([][]string{{"a1", "b1", "c1"}})
	tb.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 3})
	tb.OnFocus()
	// 横向滚动走公开按键路径（Left/Right）。
	for i := 0; i < 2; i++ {
		tb.Handle(event.Key{Code: event.KeyRight, Press: true})
	}
	if tb.HOffset() == 0 {
		t.Fatal("precondition: table must be scrolled horizontally")
	}
	r := geom.Rect{X: 0, Y: 0, W: 10, H: 3}
	buf := paintBounds(tb, r)
	// 区域外一列必须没被画（Clip 生效）
	for y := 0; y < 3; y++ {
		for x := r.Right(); x < buf.Width(); x++ {
			c, _ := buf.GetCell(x, y)
			if c.Rune != 0 && c.Rune != ' ' {
				t.Errorf("(%d,%d) outside bounds was painted: %q", x, y, c.Rune)
			}
		}
	}
	// 行宽不变量：每行不得超过区域宽
	for y := 0; y < 3; y++ {
		row := rowOf(buf, y)
		if w := text.Width(row); w > r.W {
			t.Errorf("row %d width = %d > %d (%q)", y, w, r.W, row)
		}
	}
}

// Tabs 标签超出区域时不得画到界外。
func TestTabsOverflowStaysInBounds(t *testing.T) {
	tb := NewTabs()
	tb.SetTabs([]string{"第一页", "第二页", "第三页", "第四页"})
	tb.SetContent(NewLabel("x"))
	r := geom.Rect{X: 0, Y: 0, W: 8, H: 3}
	buf := paintBounds(tb, r)
	for y := 0; y < 3; y++ {
		row := rowOf(buf, y)
		if w := text.Width(row); w > r.W {
			t.Errorf("row %d width = %d > %d (%q)", y, w, r.W, row)
		}
	}
}

// Clip 生效后，任何控件都不得画出 Bounds 之外。
func TestNoWidgetPaintsOutsideBounds(t *testing.T) {
	build := func() []struct {
		name string
		w    Widget
		r    geom.Rect
	} {
		lbl := NewLabel("very long label text")
		btn := NewButton("very long button label")
		chk := NewCheckbox("very long checkbox label")
		rad := NewRadio("very long radio label")
		ti := NewTextInput()
		ti.SetText("some long value here")
		dd := NewDropdown()
		dd.SetItems([]string{"very long option text"})
		lst := NewList()
		lst.SetItems([]string{"very long list item"})
		sl := NewSlider()
		pb := NewProgressBar()
		pb.SetShowPercent(true)
		sep := NewSeparator()
		sb := NewScrollbar()
		sb.SetTotal(100)
		sb.SetView(5)
		tb := NewTable()
		tb.SetColumns([]TableColumn{{Title: "T", Width: 30}})
		tb.SetRows([][]string{{"cell"}})
		tabs := NewTabs()
		tabs.SetTabs([]string{"一", "二", "三", "四", "五", "六"})
		pn := NewPanel("很长的面板标题")
		pn.SetChild(NewLabel("content"))
		return []struct {
			name string
			w    Widget
			r    geom.Rect
		}{
			{"Label", lbl, geom.Rect{X: 0, Y: 0, W: 6, H: 2}},
			{"Button", btn, geom.Rect{X: 0, Y: 0, W: 6, H: 1}},
			{"Checkbox", chk, geom.Rect{X: 0, Y: 0, W: 6, H: 1}},
			{"Radio", rad, geom.Rect{X: 0, Y: 0, W: 6, H: 1}},
			{"TextInput", ti, geom.Rect{X: 0, Y: 0, W: 6, H: 1}},
			{"Dropdown", dd, geom.Rect{X: 0, Y: 0, W: 6, H: 1}},
			{"List", lst, geom.Rect{X: 0, Y: 0, W: 6, H: 2}},
			{"Slider", sl, geom.Rect{X: 0, Y: 0, W: 6, H: 3}},
			{"ProgressBar", pb, geom.Rect{X: 0, Y: 0, W: 6, H: 3}},
			{"Separator", sep, geom.Rect{X: 0, Y: 0, W: 6, H: 1}},
			{"Scrollbar", sb, geom.Rect{X: 0, Y: 0, W: 1, H: 4}},
			{"Table", tb, geom.Rect{X: 0, Y: 0, W: 6, H: 3}},
			{"Tabs", tabs, geom.Rect{X: 0, Y: 0, W: 6, H: 3}},
			{"Panel", pn, geom.Rect{X: 0, Y: 0, W: 10, H: 6}},
		}
	}
	// Clip 由 Root 施加（Root.Paint 把每个控件裁剪到自己的 Bounds）。
	for _, tc := range build() {
		root := NewRoot()
		root.Layout(geom.Rect{X: 0, Y: 0, W: 40, H: 20})
		root.Add(tc.w)
		tc.w.Layout(tc.r)

		// 画一块比 Bounds 大的缓冲，界外任何落笔都是越界。
		buf := screen.NewBuffer(tc.r.Right()+4, tc.r.Bottom()+2)
		root.Paint(buf)
		for y := 0; y < buf.Height(); y++ {
			for x := 0; x < buf.Width(); x++ {
				inside := x >= tc.r.X && x < tc.r.Right() && y >= tc.r.Y && y < tc.r.Bottom()
				if inside {
					continue
				}
				c, _ := buf.GetCell(x, y)
				if c.Rune != 0 {
					t.Errorf("%s painted outside Bounds at (%d,%d): %q", tc.name, x, y, c.Rune)
				}
			}
		}
	}
}

// Clip 与铺底不冲突：区域外仍是未绘制的零值格（不是被填的空格）。
func TestClipDoesNotFillOutside(t *testing.T) {
	lbl := NewLabel("hi")
	r := geom.Rect{X: 0, Y: 0, W: 4, H: 1}
	buf := screen.NewBuffer(8, 3)
	lbl.Layout(r)
	lbl.Paint(buf)
	if c, _ := buf.GetCell(7, 2); c.Rune != 0 {
		t.Errorf("outside must stay untouched: %+v", c)
	}
}

// ---- helpers ----

// rowOf 把一行还原成字符串（续格跳过，零值格算空格）。
func rowOf(buf *screen.Buffer, y int) string {
	var sb strings.Builder
	for x := 0; x < buf.Width(); x++ {
		c, _ := buf.GetCell(x, y)
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
