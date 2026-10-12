package widget

import (
	"testing"

	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/style"
	"PaintedSkin/widget/layout"
)

// memCanvas 是 Paint golden 用的内存画布：记录全部绘制调用。
type memCanvas struct {
	size      geom.Size
	texts     []textCall
	boxes     []boxCall
	fills     []fillCall
	cellCalls []cellCall
	cells     int
	outlines  []outlineCall
	hatches   []hatchCall
}

type textCall struct {
	x, y int
	s    string
	st   style.Style
}

type boxCall struct {
	r  geom.Rect
	bs screen.Border
	st style.Style
}

type hatchCall struct {
	r  geom.Rect
	h  screen.Hatch
	st style.Style
}

type outlineCall struct {
	r     geom.Rect
	edges screen.BorderEdges
	st    style.Style
}

type fillCall struct {
	r  geom.Rect
	ch rune
	st style.Style
}

func newMemCanvas(w, h int) *memCanvas { return &memCanvas{size: geom.Size{W: w, H: h}} }

func (m *memCanvas) SetCell(x, y int, r rune, s style.Style) {
	m.cellCalls = append(m.cellCalls, cellCall{x, y, r, s})
	m.cells++
}

type cellCall struct {
	x, y int
	r    rune
	st   style.Style
}

func (m *memCanvas) Fill(rect geom.Rect, r rune, s style.Style) {
	m.fills = append(m.fills, fillCall{rect, r, s})
}
func (m *memCanvas) FillStyle(rect geom.Rect, s style.Style) {
	m.fills = append(m.fills, fillCall{rect, ' ', s})
}
func (m *memCanvas) Text(x, y int, s string, st style.Style) {
	m.texts = append(m.texts, textCall{x, y, s, st})
}
func (m *memCanvas) Box(rect geom.Rect, bs screen.Border, s style.Style) {
	m.boxes = append(m.boxes, boxCall{rect, bs, s})
}
func (m *memCanvas) Outline(rect geom.Rect, e screen.BorderEdges, s style.Style) {
	m.outlines = append(m.outlines, outlineCall{rect, e, s})
}
func (m *memCanvas) Hatch(rect geom.Rect, h screen.Hatch, s style.Style) {
	m.hatches = append(m.hatches, hatchCall{rect, h, s})
}
func (m *memCanvas) Clip(rect geom.Rect) {}
func (m *memCanvas) PopClip()            {}
func (m *memCanvas) Size() geom.Size     { return m.size }

// ---- Label ----

func TestLabelMeasure(t *testing.T) {
	loose := layout.Constraints{MaxW: 80, MaxH: 24}
	l := NewLabel("中文ab") // 2+2+1+1 = 6
	if got := l.Measure(loose); got != (geom.Size{W: 6, H: 1}) {
		t.Errorf("single = %+v", got)
	}
	// 换行：宽 4 → ["中文","ab"]
	wrap := NewLabel("中文ab")
	wrap.SetWrap(true)
	if got := wrap.Measure(layout.Constraints{MaxW: 4, MaxH: 24}); got != (geom.Size{W: 4, H: 2}) {
		t.Errorf("wrap = %+v", got)
	}
	// 边界：空文本
	if got := NewLabel("").Measure(loose); got.H != 1 || got.W != 0 {
		t.Errorf("empty = %+v", got)
	}
	// 钳制：MaxW=2
	if got := NewLabel("中文ab").Measure(layout.Constraints{MaxW: 2, MaxH: 24}); got.W != 2 {
		t.Errorf("clamp = %+v", got)
	}
}

func TestLabelAlignPaint(t *testing.T) {
	mk := func(a layout.Align) *memCanvas {
		l := NewLabel("ab")
		l.SetAlign(a)
		l.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 1})
		mc := newMemCanvas(10, 1)
		l.Paint(mc)
		return mc
	}
	if mc := mk(layout.AlignStart); len(mc.texts) != 1 || mc.texts[0].x != 0 {
		t.Errorf("left = %+v", mc.texts)
	}
	if mc := mk(layout.AlignCenter); len(mc.texts) != 1 || mc.texts[0].x != 4 {
		t.Errorf("center = %+v", mc.texts)
	}
	if mc := mk(layout.AlignEnd); len(mc.texts) != 1 || mc.texts[0].x != 8 {
		t.Errorf("right = %+v", mc.texts)
	}
}

func TestLabelBubbles(t *testing.T) {
	l := NewLabel("x")
	l.Layout(geom.Rect{X: 0, Y: 0, W: 5, H: 1})
	if l.Handle(pressAt(1, 0)) {
		t.Error("label mouse must bubble (false)")
	}
	if l.Handle(event.Key{Code: event.KeyEnter, Press: true}) {
		t.Error("label key must bubble (false)")
	}
}

// ---- Button ----

func TestButtonClickSequence(t *testing.T) {
	b := NewButton("OK")
	b.Layout(geom.Rect{X: 0, Y: 0, W: 6, H: 1})
	clicks := 0
	b.SetOnClick(func() { clicks++ })

	b.Handle(pressAt(1, 0))
	if !b.Pressed() {
		t.Error("press must set pressed")
	}
	b.Handle(releaseAt(1, 0))
	if clicks != 1 {
		t.Errorf("release inside must click: %d", clicks)
	}
	if b.Pressed() {
		t.Error("release must reset pressed")
	}
	// 界外释放不触发
	b.Handle(pressAt(1, 0))
	b.Handle(releaseAt(30, 15))
	if clicks != 1 {
		t.Errorf("release outside must not click: %d", clicks)
	}
}

func TestButtonKeyboard(t *testing.T) {
	b := NewButton("OK")
	clicks := 0
	b.SetOnClick(func() { clicks++ })
	b.OnFocus()
	b.Handle(event.Key{Code: event.KeyEnter, Press: true})
	b.Handle(event.Key{Code: event.KeySpace, Press: true})
	if clicks != 2 {
		t.Errorf("enter+space = %d", clicks)
	}
	b.OnBlur()
	b.Handle(event.Key{Code: event.KeyEnter, Press: true})
	if clicks != 2 {
		t.Errorf("blurred must not trigger: %d", clicks)
	}
}

func TestButtonDisabledSwallows(t *testing.T) {
	b := NewButton("OK")
	b.Layout(geom.Rect{X: 0, Y: 0, W: 6, H: 1})
	clicks := 0
	b.SetOnClick(func() { clicks++ })
	b.SetDisabled(true)
	if !b.Handle(pressAt(1, 0)) {
		t.Error("disabled press must be swallowed")
	}
	if !b.Handle(releaseAt(1, 0)) {
		t.Error("disabled release must be swallowed")
	}
	b.OnFocus()
	if !b.Handle(event.Key{Code: event.KeyEnter, Press: true}) {
		t.Error("disabled key must be swallowed")
	}
	if clicks != 0 {
		t.Error("disabled must not click")
	}
}

func TestButtonHoverStates(t *testing.T) {
	b := NewButton("OK")
	b.Layout(geom.Rect{X: 0, Y: 0, W: 6, H: 1})
	b.Handle(event.Mouse{Kind: event.MouseEnter})
	if !b.Hover() {
		t.Error("enter must hover")
	}
	b.Handle(event.Mouse{Kind: event.MouseLeave})
	if b.Hover() {
		t.Error("leave must unhover")
	}
}

func TestButtonMeasurePaint(t *testing.T) {
	if got := NewButton("OK").Measure(layout.Constraints{MaxW: 80, MaxH: 24}); got.W != 6 || got.H != 1 {
		t.Errorf("measure = %+v", got)
	}
	fw := NewButton("OK")
	fw.SetWidth(10)
	if got := fw.Measure(layout.Constraints{MaxW: 80, MaxH: 24}); got.W != 10 {
		t.Errorf("fixed = %+v", got)
	}
	// golden：pressed 叠反色、disabled 叠暗淡
	b := NewButton("OK")
	b.Layout(geom.Rect{X: 0, Y: 0, W: 6, H: 1})
	b.Handle(pressAt(1, 0))
	mc := newMemCanvas(6, 1)
	b.Paint(mc)
	if len(mc.texts) != 1 || !mc.texts[0].st.Reverse {
		t.Errorf("pressed paint = %+v", mc.texts)
	}
	b2 := NewButton("OK")
	b2.Layout(geom.Rect{X: 0, Y: 0, W: 6, H: 1})
	b2.SetDisabled(true)
	mc2 := newMemCanvas(6, 1)
	b2.Paint(mc2)
	if len(mc2.texts) != 1 || !mc2.texts[0].st.Dim {
		t.Errorf("disabled paint = %+v", mc2.texts)
	}
}

// ---- Panel ----

func TestPanelMeasureLayout(t *testing.T) {
	p := NewPanel("T")
	p.SetChild(NewLabel("中文ab")) // 6×1
	// 默认 padding 1 + 单线边框 + 标题行 1：内容 6×1 → 高 1+1+2+2=6
	got := p.Measure(layout.Constraints{MaxW: 80, MaxH: 24})
	if got != (geom.Size{W: 10, H: 6}) {
		t.Errorf("measure = %+v", got)
	}
	p.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 10})
	// 内容区 = 外框 - padding(1) - border(1) - 标题行(1)
	want := geom.Rect{X: 2, Y: 3, W: 16, H: 5}
	if p.Child() == nil {
		t.Fatal("no child")
	}
	childBounds := p.Child().(*Label).Bounds
	if childBounds != want {
		t.Errorf("child = %+v want %+v", childBounds, want)
	}
}

func TestPanelHitBubble(t *testing.T) {
	p := NewPanel("T")
	lbl := NewLabel("hi")
	p.SetChild(lbl)
	p.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 10})
	// 子区命中子控件
	if hit := p.HitTest(geom.Point{X: 5, Y: 5}); hit != Widget(lbl) {
		t.Errorf("hit = %v want label", hit)
	}
	// 边框命中 panel 自身
	if hit := p.HitTest(geom.Point{X: 0, Y: 0}); hit != Widget(p) {
		t.Errorf("border hit = %v want panel", hit)
	}
	if p.Handle(pressAt(5, 5)) {
		t.Error("panel must not consume")
	}
	// golden：Box + 标题
	mc := newMemCanvas(20, 10)
	p.Paint(mc)
	if len(mc.boxes) != 1 || mc.boxes[0].r != (geom.Rect{X: 0, Y: 0, W: 20, H: 10}) {
		t.Errorf("box = %+v", mc.boxes)
	}
	found := false
	for _, tc := range mc.texts {
		if tc.s == "T" {
			found = true
		}
	}
	if !found {
		t.Errorf("title missing: %+v", mc.texts)
	}
}

// ---- Dialog ----

func TestDialogCenterButtons(t *testing.T) {
	d := NewDialog("T", "msg", []string{"OK"}, nil)
	area := geom.Rect{X: 0, Y: 0, W: 40, H: 20}
	d.Layout(area)
	// 自适应 20×7 → 居中 (10,6)
	if d.Bounds != (geom.Rect{X: 10, Y: 6, W: 20, H: 7}) {
		t.Errorf("centered = %+v", d.Bounds)
	}
	if len(d.Buttons()) != 1 {
		t.Fatalf("buttons = %d", len(d.Buttons()))
	}
	bb := d.Buttons()[0].Bounds
	if bb.Y != 11 || bb.W != 6 {
		t.Errorf("button = %+v", bb)
	}
	if bb.X < d.Bounds.X || bb.Right() > d.Bounds.Right() {
		t.Errorf("button out of dialog: %+v", bb)
	}
}

func TestDialogButtonClick(t *testing.T) {
	r := mkRoot()
	var got = -1
	d := NewDialog("T", "msg", []string{"Yes", "No"}, func(i int) { got = i })
	r.Add(d)
	d.Layout(geom.Rect{X: 0, Y: 0, W: 40, H: 20})
	b0 := d.Buttons()[0].Bounds
	cx, cy := b0.X+b0.W/2, b0.Y
	r.Dispatch(pressAt(cx, cy))
	r.Dispatch(releaseAt(cx, cy))
	if got != 0 {
		t.Errorf("click = %d want 0", got)
	}
}

func TestDialogEscCloses(t *testing.T) {
	r := mkRoot()
	closed := false
	d := NewDialog("T", "msg", []string{"OK"}, nil)
	d.SetOnClose(func() { closed = true })
	r.PushDialog(d)
	if len(r.dialogs) != 1 {
		t.Fatal("not pushed")
	}
	r.Dispatch(event.Key{Code: event.KeyEsc, Press: true})
	if len(r.dialogs) != 0 {
		t.Error("esc must pop")
	}
	if !closed {
		t.Error("esc must call onClose")
	}
}
