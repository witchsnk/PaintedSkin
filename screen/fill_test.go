package screen

import (
	"testing"

	"PaintedSkin/geom"
	"PaintedSkin/style"
)

// FillStyle 用空格（而非任意字符）铺满矩形：这是控件"先铺自身背景、
// 再写内容"的原语（docs/05 尾部空档纪律）。
func TestCanvasFillStyle(t *testing.T) {
	st := style.Style{}.WithBackground(style.ANSIColor(238))
	b := NewBuffer(6, 2)
	b.FillStyle(geom.Rect{X: 1, Y: 0, W: 3, H: 1}, st)
	for x := 1; x < 4; x++ {
		c, ok := b.GetCell(x, 0)
		if !ok {
			t.Fatalf("cell %d missing", x)
		}
		if c.Rune != ' ' || c.Width != 1 {
			t.Errorf("cell %d = %+v want blank width-1 cell", x, c)
		}
		if !c.Style.Equal(st) {
			t.Errorf("cell %d style = %+v want %+v", x, c.Style, st)
		}
	}
	if c, _ := b.GetCell(4, 0); c.Rune != 0 {
		t.Errorf("fill must not exceed rect: %+v", c)
	}
	if c, _ := b.GetCell(1, 1); c.Rune != 0 {
		t.Errorf("fill must not exceed rect vertically: %+v", c)
	}
}

// FillStyle 覆盖宽字符左半时必须同步清掉右半占位，否则留下孤儿续格
// （diff 阶段会被当成半格残留，整行错位）。
func TestFillStyleClearsWidePair(t *testing.T) {
	b := NewBuffer(6, 1)
	b.Text(0, 0, "中文", style.Default) // 占 0..3：0/2 左格，1/3 续格
	b.FillStyle(geom.Rect{X: 0, Y: 0, W: 2, H: 1}, style.Default)
	for x := 0; x < 2; x++ {
		c, _ := b.GetCell(x, 0)
		if c.IsContinuation() {
			t.Errorf("orphan continuation left at column %d", x)
		}
		if c.Rune != ' ' {
			t.Errorf("column %d = %+v want blank", x, c)
		}
	}
	// 宽字符对的后一格（2..3）不受影响。
	if c, _ := b.GetCell(2, 0); c.Rune != '文' {
		t.Errorf("column 2 = %+v want 文 untouched", c)
	}
}

// FillStyle 覆盖宽字符**右半**占位所在列时，左半必须降级成空格。
//
// 覆盖续格而不动左半，会留下一个"没有配对的宽字符左格"：diff 写出它时会
// 吃掉两列，而缓冲只打算写一格 → 整行右移一列（docs/05 宽字符处理）。
func TestFillStyleOverContinuationOnly(t *testing.T) {
	b := NewBuffer(6, 1)
	b.Text(0, 0, "中", style.Default) // 0=左格，1=续格
	b.FillStyle(geom.Rect{X: 1, Y: 0, W: 2, H: 1}, style.Default)
	if c, _ := b.GetCell(1, 0); c.IsContinuation() {
		t.Error("must not keep continuation at covered column")
	}
	// 左半不得保持 Width==2（那会让它多写一列）。
	if c, _ := b.GetCell(0, 0); c.Width == 2 {
		t.Errorf("unpaired wide left cell: %+v", c)
	}
	if c, _ := b.GetCell(0, 0); c.Rune != ' ' {
		t.Errorf("left half must degrade to a blank: %+v", c)
	}
	assertNoUnpairedWide(t, b)
}

// FillStyle 遵守裁剪区。
func TestFillStyleRespectsClip(t *testing.T) {
	b := NewBuffer(6, 2)
	b.Clip(geom.Rect{X: 0, Y: 0, W: 2, H: 1})
	b.FillStyle(geom.Rect{X: 0, Y: 0, W: 6, H: 2}, style.Default)
	if c, _ := b.GetCell(2, 0); c.Rune != 0 {
		t.Errorf("outside clip must stay untouched: %+v", c)
	}
	if c, _ := b.GetCell(0, 1); c.Rune != 0 {
		t.Errorf("outside clip must stay untouched: %+v", c)
	}
}

// 退化矩形不得 panic。
func TestFillStyleDegenerate(t *testing.T) {
	b := NewBuffer(4, 2)
	b.FillStyle(geom.Rect{X: 0, Y: 0, W: 0, H: 2}, style.Default)
	b.FillStyle(geom.Rect{X: 3, Y: 3, W: 2, H: 2}, style.Default)
	b.FillStyle(geom.Rect{X: -1, Y: -1, W: 99, H: 99}, style.Default)
	for y := 0; y < 2; y++ {
		for x := 0; x < 4; x++ {
			c, _ := b.GetCell(x, y)
			if c.IsContinuation() {
				t.Errorf("(%d,%d) dangling continuation", x, y)
			}
		}
	}
}

// Fill 与 FillStyle 必须共用同一套成对处理（单一真相源）。
func TestFillAndFillStyleShareWidePairHandling(t *testing.T) {
	for name, paint := range map[string]func(*Buffer){
		"Fill":      func(b *Buffer) { b.Fill(geom.Rect{X: 0, Y: 0, W: 2, H: 1}, '.', style.Default) },
		"FillStyle": func(b *Buffer) { b.FillStyle(geom.Rect{X: 0, Y: 0, W: 2, H: 1}, style.Default) },
	} {
		b := NewBuffer(6, 1)
		b.Text(0, 0, "中文", style.Default)
		paint(b)
		for x := 0; x < 2; x++ {
			if c, _ := b.GetCell(x, 0); c.IsContinuation() {
				t.Errorf("%s: orphan continuation at %d", name, x)
			}
		}
	}
}
