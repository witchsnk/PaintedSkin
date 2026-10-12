package widget

import (
	"testing"

	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/style"
	"PaintedSkin/widget/layout"
)

// Measure：宽=列宽和，高=min(1+行数, MaxH)。
func TestTableMeasure(t *testing.T) {
	tb := NewTable()
	tb.SetColumns([]TableColumn{
		{Title: "A", Width: 6},
		{Title: "B", Width: 4},
	})
	tb.SetRows([][]string{{"a1", "b1"}, {"a2", "b2"}})
	got := tb.Measure(layout.Constraints{MaxW: 80, MaxH: 24})
	if got.W != 10 || got.H != 3 {
		t.Errorf("fixed widths = %+v", got)
	}
	// 高度受 MaxH 钳制（表头 + 2 行）
	got2 := tb.Measure(layout.Constraints{MaxW: 80, MaxH: 2})
	if got2.H != 2 {
		t.Errorf("clamped = %+v", got2)
	}
	// 空表
	if got3 := NewTable().Measure(layout.Constraints{MaxW: 80, MaxH: 5}); got3.H != 0 {
		t.Errorf("empty = %+v", got3)
	}
}

// 自动均分列宽。
func TestTableAutoWidth(t *testing.T) {
	tb := NewTable()
	tb.SetColumns([]TableColumn{
		{Title: "A"},
		{Title: "B"},
		{Title: "C"},
	})
	tb.SetRows([][]string{{"x", "y", "z"}})
	tb.Layout(geom.Rect{X: 0, Y: 0, W: 30, H: 5})
	ws := tb.ColumnWidths()
	if len(ws) != 3 {
		t.Fatalf("widths = %v", ws)
	}
	total := 0
	for _, w := range ws {
		total += w
		if w <= 0 {
			t.Errorf("auto width must be positive: %v", ws)
		}
	}
	if total != 30 {
		t.Errorf("auto widths must fill area: %d want 30 (%v)", total, ws)
	}
}

// 上下键选行 + 滚入视。
func TestTableSelectionScroll(t *testing.T) {
	tb := NewTable()
	tb.SetColumns([]TableColumn{{Title: "A"}})
	tb.SetRows([][]string{{"r0"}, {"r1"}, {"r2"}, {"r3"}, {"r4"}})
	tb.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 3}) // 表头 1 + 2 行视口
	tb.OnFocus()
	if tb.SelectedRow() != 0 {
		t.Errorf("initial row = %d", tb.SelectedRow())
	}
	for i := 0; i < 3; i++ {
		tb.Handle(event.Key{Code: event.KeyDown, Press: true})
	}
	if tb.SelectedRow() != 3 {
		t.Fatalf("row = %d want 3", tb.SelectedRow())
	}
	if tb.Offset() == 0 {
		t.Error("must scroll into view")
	}
	for i := 0; i < 10; i++ {
		tb.Handle(event.Key{Code: event.KeyUp, Press: true})
	}
	if tb.SelectedRow() != 0 || tb.Offset() != 0 {
		t.Errorf("back to head: row=%d off=%d", tb.SelectedRow(), tb.Offset())
	}
}

// 垂直滚轮。
func TestTableWheel(t *testing.T) {
	tb := NewTable()
	tb.SetColumns([]TableColumn{{Title: "A"}})
	tb.SetRows([][]string{{"0"}, {"1"}, {"2"}, {"3"}})
	tb.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 2})
	tb.OnFocus()
	if !tb.Handle(event.Mouse{Kind: event.MouseWheelDown}) {
		t.Error("wheel must consume")
	}
	if tb.Offset() == 0 {
		t.Error("wheel must scroll")
	}
	tb.Handle(event.Mouse{Kind: event.MouseWheelUp})
	if tb.Offset() != 0 {
		t.Errorf("wheel up = %d", tb.Offset())
	}
}

// 水平滚动（列总宽超视口）。
func TestTableHorizontalScroll(t *testing.T) {
	tb := NewTable()
	tb.SetColumns([]TableColumn{
		{Title: "AAAA", Width: 10},
		{Title: "BBBB", Width: 10},
	})
	tb.SetRows([][]string{{"11", "22"}})
	tb.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 3})
	tb.OnFocus()
	if tb.HOffset() != 0 {
		t.Errorf("initial hoff = %d", tb.HOffset())
	}
	tb.Handle(event.Key{Code: event.KeyRight, Press: true})
	if tb.HOffset() == 0 {
		t.Error("right must hscroll")
	}
	tb.Handle(event.Key{Code: event.KeyLeft, Press: true})
	if tb.HOffset() != 0 {
		t.Errorf("left back = %d", tb.HOffset())
	}
}

// Enter 激活；点击选行。
func TestTableActivateClick(t *testing.T) {
	tb := NewTable()
	tb.SetColumns([]TableColumn{{Title: "A"}})
	tb.SetRows([][]string{{"a"}, {"b"}, {"c"}})
	tb.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 4})
	tb.OnFocus()
	act := -1
	tb.SetOnActivate(func(i int) { act = i })
	tb.Handle(event.Key{Code: event.KeyEnter, Press: true})
	if act != 0 {
		t.Errorf("activate = %d", act)
	}
	// 表头占第 0 行，数据行从第 1 行起；点 y=2 → 第 1 个数据行
	tb.Handle(pressAt(2, 2))
	if tb.SelectedRow() != 1 {
		t.Errorf("click row = %d want 1", tb.SelectedRow())
	}
}

// 禁用吞事件。
func TestTableDisabled(t *testing.T) {
	tb := NewTable()
	tb.SetColumns([]TableColumn{{Title: "A"}})
	tb.SetRows([][]string{{"a"}, {"b"}})
	tb.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 3})
	tb.OnFocus()
	tb.SetDisabled(true)
	if !tb.Handle(event.Key{Code: event.KeyDown, Press: true}) {
		t.Error("disabled key must swallow")
	}
	if !tb.Handle(event.Mouse{Kind: event.MouseWheelDown}) {
		t.Error("disabled wheel must swallow")
	}
	if tb.SelectedRow() != 0 {
		t.Errorf("disabled must not change: %d", tb.SelectedRow())
	}
}

// Paint golden：表头粗体固定，主体行，列对齐。
func TestTablePaint(t *testing.T) {
	tb := NewTable()
	tb.SetColumns([]TableColumn{
		{Title: "AA", Width: 4},
		{Title: "B", Width: 4},
	})
	tb.SetRows([][]string{{"r0c0", "r0c1"}, {"r1c0", "r1c1"}})
	tb.Layout(geom.Rect{X: 0, Y: 0, W: 8, H: 3})
	tb.OnFocus()
	tb.SetSelectedRow(1)
	mc := newMemCanvas(8, 3)
	tb.Paint(mc)
	// 按 (x,y) 收集绘制文本
	at := func(x, y int) (string, style.Style, bool) {
		for _, tc := range mc.texts {
			if tc.x == x && tc.y == y {
				return tc.s, tc.st, true
			}
		}
		return "", style.Style{}, false
	}
	if s, st, ok := at(0, 0); !ok || s != "AA  " || !st.Bold {
		t.Errorf("header A = %q bold=%v ok=%v", s, st.Bold, ok)
	}
	if s, _, ok := at(4, 0); !ok || s != "B   " {
		t.Errorf("header B = %q ok=%v", s, ok)
	}
	if s, _, ok := at(0, 1); !ok || s != "r0c0" {
		t.Errorf("row0 col0 = %q ok=%v", s, ok)
	}
	// 选中行（第 1 个数据行，y=2）反色
	s2, st2, ok := at(0, 2)
	if !ok || s2 != "r1c0" || !st2.Reverse {
		t.Errorf("selected row = %q reverse=%v ok=%v", s2, st2.Reverse, ok)
	}
}
