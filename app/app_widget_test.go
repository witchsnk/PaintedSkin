package app

import (
	"bytes"
	"errors"
	"io"
	"sync"
	"testing"
	"time"

	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/input"
	"PaintedSkin/screen"
	"PaintedSkin/term"
	"PaintedSkin/theme"
	"PaintedSkin/widget"
)

var got []event.Event

// newTestApp 构造不接真终端的 App（内存缓冲 + 内存 writer）。
func newTestApp(w, h int) (*App, *bytes.Buffer) {
	buf := &bytes.Buffer{}
	a := New(Config{
		Width:  w,
		Height: h,
		Writer: buf,
	})
	return a, buf
}

// 初始帧应绘制并输出全量重绘。
func TestAppFirstFrame(t *testing.T) {
	a, out := newTestApp(20, 5)
	lbl := widget.NewLabel("hi")
	root := a.Root()
	root.Add(lbl)
	lbl.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 1})
	a.Layout()
	a.Draw()
	if out.Len() == 0 {
		t.Fatal("first frame must emit output")
	}
}

// 事件经 Root 分发：点击按钮触发回调。
func TestAppDispatchClick(t *testing.T) {
	a, _ := newTestApp(20, 5)
	clicks := 0
	btn := widget.NewButton("OK")
	btn.SetOnClick(func() { clicks++ })
	a.Root().Add(btn)
	btn.Layout(geom.Rect{X: 0, Y: 0, W: 6, H: 1})
	a.Layout()

	a.Dispatch(event.Mouse{X: 1, Y: 0, Kind: event.MousePress, Button: event.ButtonLeft})
	a.Dispatch(event.Mouse{X: 1, Y: 0, Kind: event.MouseRelease, Button: event.ButtonLeft})
	if clicks != 1 {
		t.Errorf("clicks = %d want 1", clicks)
	}
}

// Tab 遍历焦点。
func TestAppTabFocus(t *testing.T) {
	a, _ := newTestApp(20, 5)
	b1 := widget.NewButton("A")
	b2 := widget.NewButton("B")
	a.Root().Add(b1)
	a.Root().Add(b2)
	b1.Layout(geom.Rect{X: 0, Y: 0, W: 4, H: 1})
	b2.Layout(geom.Rect{X: 0, Y: 1, W: 4, H: 1})
	a.Layout()

	a.Dispatch(event.Key{Code: event.KeyTab, Press: true})
	if a.Root().Focused() != widget.Widget(b1) {
		t.Errorf("tab1 = %v", a.Root().Focused())
	}
	a.Dispatch(event.Key{Code: event.KeyTab, Press: true})
	if a.Root().Focused() != widget.Widget(b2) {
		t.Errorf("tab2 = %v", a.Root().Focused())
	}
}

// Resize 改变尺寸并重排。
func TestAppResize(t *testing.T) {
	a, _ := newTestApp(20, 5)
	a.Layout()
	a.Dispatch(event.Resize{W: 40, H: 10})
	if a.Size() != (geom.Size{W: 40, H: 10}) {
		t.Errorf("size = %+v", a.Size())
	}
	if a.Root().Bounds.W != 40 {
		t.Errorf("root bounds = %+v", a.Root().Bounds)
	}
}

// Close 幂等；重复调用不 panic。
func TestAppCloseIdempotent(t *testing.T) {
	a, _ := newTestApp(10, 3)
	if err := a.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	if err := a.Close(); err != nil {
		t.Errorf("second close must be nil: %v", err)
	}
	if !a.Closed() {
		t.Error("Closed must be true")
	}
}

// Post 投递的闭包在下一次 Flush 执行（UI 线程串行）。
func TestAppPost(t *testing.T) {
	a, _ := newTestApp(10, 3)
	ran := false
	a.Post(func() { ran = true })
	if ran {
		t.Error("Post must defer execution")
	}
	a.Flush()
	if !ran {
		t.Error("Flush must run posted funcs")
	}
}

// Dialog 模态：打开后屏蔽下层输入。
func TestAppDialogModal(t *testing.T) {
	a, _ := newTestApp(40, 20)
	btn := widget.NewButton("under")
	a.Root().Add(btn)
	btn.Layout(geom.Rect{X: 0, Y: 0, W: 8, H: 1})
	a.Layout()

	dlg := widget.NewDialog("T", "m", []string{"OK"}, nil)
	a.Root().PushDialog(dlg)
	dlg.Layout(a.Root().Bounds)

	clicks := 0
	btn.SetOnClick(func() { clicks++ })
	a.Dispatch(event.Mouse{X: 1, Y: 0, Kind: event.MousePress, Button: event.ButtonLeft})
	a.Dispatch(event.Mouse{X: 1, Y: 0, Kind: event.MouseRelease, Button: event.ButtonLeft})
	if clicks != 0 {
		t.Errorf("modal must shield: clicks=%d", clicks)
	}
	a.Root().PopDialog()
}

// 绘制到内存 Buffer：Label 文本可见。
func TestAppPaintVisible(t *testing.T) {
	a, _ := newTestApp(10, 2)
	lbl := widget.NewLabel("XY")
	a.Root().Add(lbl)
	lbl.Layout(geom.Rect{X: 0, Y: 0, W: 10, H: 1})
	a.Layout()
	a.Draw()
	found := false
	for y := 0; y < a.Buffer().Height(); y++ {
		for x := 0; x < a.Buffer().Width(); x++ {
			if c := a.Buffer().At(x, y); c != nil && c.Rune == 'X' {
				found = true
			}
		}
	}
	if !found {
		t.Error("painted text must reach buffer")
	}
}

// Config 缺省值补全。
func TestConfigDefaults(t *testing.T) {
	c := Config{}.withDefaults()
	if c.Width <= 0 || c.Height <= 0 {
		t.Errorf("size defaults = %dx%d", c.Width, c.Height)
	}
	if c.Writer == nil {
		t.Error("writer default must be set")
	}
	var _ screen.Canvas = screen.NewBuffer(1, 1)
}

// fakeTerm 是满足 term.Terminal 的假后端（不触碰真实 tty）。
type fakeTerm struct {
	mu     sync.Mutex
	in     *bytes.Reader
	out    *bytes.Buffer
	size   term.Size
	caps   term.Caps
	closed int
	events chan event.Event
}

func newFakeTerm(in string) *fakeTerm {
	return &fakeTerm{
		in:     bytes.NewReader([]byte(in)),
		out:    &bytes.Buffer{},
		size:   term.Size{W: 40, H: 20},
		caps:   term.Caps{ColorDepth: "256", HasMouse: true},
		events: make(chan event.Event, 8),
	}
}

func (f *fakeTerm) Events() <-chan event.Event { return f.events }
func (f *fakeTerm) Input() io.Reader {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.in
}
func (f *fakeTerm) Writer() io.Writer { return f.out }
func (f *fakeTerm) Size() term.Size   { return f.size }
func (f *fakeTerm) Capabilities() term.Caps {
	return f.caps
}
func (f *fakeTerm) Close() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.closed++
	return nil
}
func (f *fakeTerm) closeCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.closed
}

var _ term.Terminal = (*fakeTerm)(nil)

// 端到端：字节流 → 解码 → 控件状态（回车触发按钮）。
func TestEndToEndKeyToWidget(t *testing.T) {
	// "\r" = Enter；解码后应触发聚焦按钮的 OnClick
	ft := newFakeTerm("\r")
	clicks := 0
	btn := widget.NewButton("Go")
	btn.SetOnClick(func() { clicks++ })

	a := New(Config{Width: 40, Height: 20, Writer: io.Discard})
	a.Root().Add(btn)
	btn.Layout(geom.Rect{X: 0, Y: 0, W: 8, H: 1})
	a.Layout()
	a.Root().Focus(btn)
	a.Dispatch(event.Key{Code: event.KeyEnter, Press: true})
	if clicks != 1 {
		t.Errorf("enter must click: %d", clicks)
	}

	// 解码器确实把 \r 解析为 KeyEnter
	dec := input.New(func(e event.Event) { got = append(got, e) })
	dec.Feed([]byte("\r"))
	if len(got) != 1 {
		t.Fatalf("decode \\r = %v", got)
	}
	if k, ok := got[0].(event.Key); !ok || k.Code != event.KeyEnter {
		t.Errorf("decoded = %#v", got[0])
	}
	_ = ft
}

// readInput 把整段字节喂进解码器并产出事件。
func TestReadInputFeedsDecoder(t *testing.T) {
	var got []event.Event
	dec := input.New(func(e event.Event) { got = append(got, e) })
	readInput(bytes.NewReader([]byte("a\r")), dec)
	if len(got) != 2 {
		t.Fatalf("events = %v", got)
	}
	if k := got[0].(event.Key); k.Code != event.KeyRune || k.Rune != 'a' {
		t.Errorf("first = %#v", got[0])
	}
	if k := got[1].(event.Key); k.Code != event.KeyEnter {
		t.Errorf("second = %#v", got[1])
	}
}

// fake 终端的 Close 被 app 调用一次（幂等由后端保证）。
func TestAppClosesTerminal(t *testing.T) {
	ft := newFakeTerm("")
	a := New(Config{Width: 10, Height: 3, Writer: io.Discard, Terminal: ft})
	if err := a.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := a.Close(); err != nil {
		t.Fatalf("close again: %v", err)
	}
	if ft.closeCount() != 1 {
		t.Errorf("terminal close count = %d want 1 (app-level once)", ft.closeCount())
	}
}

// New() 不得有任何输出副作用：终端尚未接管（app_widget_test 的其他用例
// 都靠这一点，Run 失败时也不该污染 stdout）。
func TestNewHasNoOutputSideEffect(t *testing.T) {
	var out bytes.Buffer
	_ = New(Config{Width: 20, Height: 5, Writer: &out})
	if out.Len() != 0 {
		t.Errorf("New must not write to Writer, got %q", out.String())
	}
}

// ApplyTheme 应触发重绘；SetTheme 只换肤不重绘。
func TestSetThemeDoesNotDraw(t *testing.T) {
	var out bytes.Buffer
	a := New(Config{Width: 20, Height: 5, Writer: &out})
	a.Layout()
	a.Draw()
	out.Reset()
	a.SetTheme(theme.Dark())
	if out.Len() != 0 {
		t.Errorf("SetTheme must not draw, got %q", out.String())
	}
	a.ApplyTheme(theme.Light())
	if out.Len() == 0 {
		t.Error("ApplyTheme must redraw")
	}
}

// 无 TTY 时 term.New 失败必须被包装为 ErrNoTerminal（调用方可判断）。
func TestErrNoTerminalIsDistinguishable(t *testing.T) {
	if !errors.Is(ErrNoTerminal, ErrNoTerminal) {
		t.Error("sentinel must be comparable")
	}
	e := errors.Join(ErrNoTerminal, errors.New("boom"))
	if !errors.Is(e, ErrNoTerminal) {
		t.Error("joined error must match sentinel")
	}
}

// 50ms 超时后裸 ESC 必须被 Flush 出 KeyEsc（即使输入流未关闭）。
func TestReadInputEscTimeout(t *testing.T) {
	pr, pw := io.Pipe()
	defer pr.Close()
	defer pw.Close()

	events := make(chan event.Event, 10)
	dec := input.New(func(e event.Event) { events <- e })
	go readInput(pr, dec)

	// 只写一个 0x1B，不关闭 writer
	if _, err := pw.Write([]byte{0x1b}); err != nil {
		t.Fatal(err)
	}

	select {
	case ev := <-events:
		k, ok := ev.(event.Key)
		if !ok || k.Code != event.KeyEsc {
			t.Fatalf("expected KeyEsc from timeout flush, got %#v", ev)
		}
	case <-time.After(300 * time.Millisecond):
		t.Fatal("timed out waiting for ESC flush")
	}
}

// 真实 input.Decoder 解码出的 Ctrl+C (KeyRune 'c' + ModCtrl) 必须被识别为退出。
func TestCtrlCQuitRecognition(t *testing.T) {
	ev := event.Key{Code: event.KeyRune, Rune: 'c', Mod: event.ModCtrl, Press: true}
	if !isQuit(ev) {
		t.Errorf("isQuit should return true for Ctrl+C (KeyRune 'c' + ModCtrl), got false")
	}
}

// 当 Dialog 打开时，Esc 键被 Dialog 消费（关闭 Dialog）并返回 true，未打开 Dialog 时返回 false。
func TestEscClosesDialogWithoutQuitting(t *testing.T) {
	a, _ := newTestApp(40, 20)
	dlg := widget.NewDialog("test", "msg", []string{"OK"}, nil)
	a.Root().PushDialog(dlg)
	dlg.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 10})

	handled := a.Dispatch(event.Key{Code: event.KeyEsc, Press: true})
	if !handled {
		t.Error("Esc should be handled when dialog is open")
	}
	if len(a.Root().Dialogs()) != 0 {
		t.Errorf("dialog should be closed after Esc, remaining: %d", len(a.Root().Dialogs()))
	}

	// 再次按 Esc（没有 Dialog 打开），Dispatch 返回 false（上层将判定为退出）
	handled = a.Dispatch(event.Key{Code: event.KeyEsc, Press: true})
	if handled {
		t.Error("Esc without dialog should not be handled (bubbles to quit)")
	}
}
