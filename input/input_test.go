package input_test

import (
	"reflect"
	"testing"

	"PaintedSkin/event"
	"PaintedSkin/input"
)

// collect 返回解码器与事件收集器。
func collect() (*input.Decoder, *[]event.Event) {
	var evs []event.Event
	d := input.New(func(e event.Event) { evs = append(evs, e) })
	return d, &evs
}

func want(t *testing.T, name string, got *[]event.Event, want ...event.Event) {
	t.Helper()
	if len(*got) == 0 && len(want) == 0 {
		// 零事件断言：nil 与空切片视为等价（DeepEqual 会区分二者，
		// 但"无产出"时收集器恒为 nil，任何正确实现都无法满足旧比较）。
		*got = nil
		return
	}
	if want == nil {
		want = []event.Event{}
	}
	if !reflect.DeepEqual(*got, want) {
		t.Errorf("%s: got %v want %v", name, *got, want)
		t.Logf("got  = %#v", *got)
		t.Logf("want = %#v", want)
	}
	*got = nil
}

func key(code event.KeyCode, r rune, mod event.Mod) event.Event {
	return event.Key{Code: code, Rune: r, Mod: mod, Press: true}
}

// ---- 可打印字符 / UTF-8 ----

func TestPlainASCII(t *testing.T) {
	d, evs := collect()
	d.Feed([]byte("ab"))
	want(t, "ab", evs,
		key(event.KeyRune, 'a', event.ModNone),
		key(event.KeyRune, 'b', event.ModNone),
	)
	if d.Pending() {
		t.Error("complete ASCII must not be pending")
	}
}

func TestUTF8SplitAcrossFeeds(t *testing.T) {
	d, evs := collect()
	raw := []byte("中")
	d.Feed(raw[:1])
	if !d.Pending() {
		t.Error("partial UTF-8 must be pending")
	}
	if len(*evs) != 0 {
		t.Errorf("partial UTF-8 must not emit, got %v", *evs)
	}
	d.Feed(raw[1:])
	want(t, "utf8 中", evs, key(event.KeyRune, '中', event.ModNone))
	if d.Pending() {
		t.Error("completed UTF-8 must not be pending")
	}
}

func TestSpaceKey(t *testing.T) {
	d, evs := collect()
	d.Feed([]byte(" "))
	want(t, "space", evs, key(event.KeySpace, ' ', event.ModNone))
}

// ---- 控制字符 ----

func TestControlKeys(t *testing.T) {
	cases := []struct {
		in   string
		want event.Event
	}{
		{"\r", key(event.KeyEnter, 0, event.ModNone)},
		{"\n", key(event.KeyEnter, 0, event.ModNone)},
		{"\t", key(event.KeyTab, 0, event.ModNone)},
		{"\x7f", key(event.KeyBackspace, 0, event.ModNone)},
		{"\x08", key(event.KeyBackspace, 0, event.ModNone)},
		{"\x03", key(event.KeyRune, 'c', event.ModCtrl)},
		{"\x01", key(event.KeyRune, 'a', event.ModCtrl)},
		{"\x1a", key(event.KeyRune, 'z', event.ModCtrl)},
		{"\x00", key(event.KeySpace, ' ', event.ModCtrl)},
		{"\x1c", key(event.KeyRune, '\\', event.ModCtrl)},
		{"\x1f", key(event.KeyRune, '_', event.ModCtrl)},
	}
	for _, c := range cases {
		d, evs := collect()
		d.Feed([]byte(c.in))
		want(t, "ctrl "+c.in, evs, c.want)
	}
}

// ---- ESC 歧义 ----

func TestEscBareFlush(t *testing.T) {
	d, evs := collect()
	d.Feed([]byte{0x1b})
	if !d.Pending() {
		t.Error("bare ESC must be pending until flush")
	}
	if len(*evs) != 0 {
		t.Errorf("bare ESC must not emit immediately, got %v", *evs)
	}
	d.Flush()
	want(t, "flush esc", evs, key(event.KeyEsc, 0, event.ModNone))
	if d.Pending() {
		t.Error("flush must clear pending")
	}
	// 再次 flush 不得重复产出。
	d.Flush()
	want(t, "flush again", evs)
}

func TestAltKey(t *testing.T) {
	d, evs := collect()
	d.Feed([]byte{0x1b, 'b'})
	want(t, "alt+b", evs, key(event.KeyRune, 'b', event.ModAlt))

	d.Feed([]byte{0x1b, '5'})
	want(t, "alt+5", evs, key(event.KeyRune, '5', event.ModAlt))

	d.Feed([]byte{0x1b, '\r'})
	want(t, "alt+enter", evs, key(event.KeyEnter, 0, event.ModAlt))
}

func TestEscEsc(t *testing.T) {
	// 连按两次 Esc：第一枚立即解析为 Esc，第二枚保持 pending 待 flush。
	d, evs := collect()
	d.Feed([]byte{0x1b, 0x1b})
	want(t, "first esc", evs, key(event.KeyEsc, 0, event.ModNone))
	if !d.Pending() {
		t.Error("second ESC must remain pending")
	}
	d.Flush()
	want(t, "second esc", evs, key(event.KeyEsc, 0, event.ModNone))
}

// ---- CSI：方向 / 修饰 / 波浪号键 ----

func TestCSIArrows(t *testing.T) {
	cases := []struct {
		in   string
		want event.Event
	}{
		{"\x1b[A", key(event.KeyUp, 0, event.ModNone)},
		{"\x1b[B", key(event.KeyDown, 0, event.ModNone)},
		{"\x1b[C", key(event.KeyRight, 0, event.ModNone)},
		{"\x1b[D", key(event.KeyLeft, 0, event.ModNone)},
		{"\x1b[H", key(event.KeyHome, 0, event.ModNone)},
		{"\x1b[F", key(event.KeyEnd, 0, event.ModNone)},
		{"\x1b[Z", key(event.KeyTab, 0, event.ModShift)}, // back-tab
		{"\x1b[1;5A", key(event.KeyUp, 0, event.ModCtrl)},
		{"\x1b[1;2D", key(event.KeyLeft, 0, event.ModShift)},
		{"\x1b[1;3C", key(event.KeyRight, 0, event.ModAlt)},
		{"\x1b[1;6B", key(event.KeyDown, 0, event.ModShift|event.ModCtrl)},
		{"\x1b[1;7A", key(event.KeyUp, 0, event.ModAlt|event.ModCtrl)},
	}
	for _, c := range cases {
		d, evs := collect()
		d.Feed([]byte(c.in))
		want(t, "csi "+c.in, evs, c.want)
	}
}

func TestCSIWaveKeys(t *testing.T) {
	cases := []struct {
		in   string
		want event.Event
	}{
		{"\x1b[1~", key(event.KeyHome, 0, event.ModNone)},
		{"\x1b[2~", key(event.KeyInsert, 0, event.ModNone)},
		{"\x1b[3~", key(event.KeyDelete, 0, event.ModNone)},
		{"\x1b[4~", key(event.KeyEnd, 0, event.ModNone)},
		{"\x1b[5~", key(event.KeyPgUp, 0, event.ModNone)},
		{"\x1b[6~", key(event.KeyPgDn, 0, event.ModNone)},
		{"\x1b[15~", key(event.KeyF5, 0, event.ModNone)},
		{"\x1b[17~", key(event.KeyF6, 0, event.ModNone)},
		{"\x1b[18~", key(event.KeyF7, 0, event.ModNone)},
		{"\x1b[19~", key(event.KeyF8, 0, event.ModNone)},
		{"\x1b[20~", key(event.KeyF9, 0, event.ModNone)},
		{"\x1b[21~", key(event.KeyF10, 0, event.ModNone)},
		{"\x1b[23~", key(event.KeyF11, 0, event.ModNone)},
		{"\x1b[24~", key(event.KeyF12, 0, event.ModNone)},
		{"\x1b[3;5~", key(event.KeyDelete, 0, event.ModCtrl)},
		{"\x1b[5;2~", key(event.KeyPgUp, 0, event.ModShift)},
	}
	for _, c := range cases {
		d, evs := collect()
		d.Feed([]byte(c.in))
		want(t, "csi "+c.in, evs, c.want)
	}
}

func TestSS3Keys(t *testing.T) {
	cases := []struct {
		in   string
		want event.Event
	}{
		{"\x1bOP", key(event.KeyF1, 0, event.ModNone)},
		{"\x1bOQ", key(event.KeyF2, 0, event.ModNone)},
		{"\x1bOR", key(event.KeyF3, 0, event.ModNone)},
		{"\x1bOS", key(event.KeyF4, 0, event.ModNone)},
		{"\x1bOA", key(event.KeyUp, 0, event.ModNone)}, // 应用光标键模式
		{"\x1bOB", key(event.KeyDown, 0, event.ModNone)},
		{"\x1bOH", key(event.KeyHome, 0, event.ModNone)},
		{"\x1bOF", key(event.KeyEnd, 0, event.ModNone)},
	}
	for _, c := range cases {
		d, evs := collect()
		d.Feed([]byte(c.in))
		want(t, "ss3 "+c.in, evs, c.want)
	}
}

func TestFragmentedCSIByteByByte(t *testing.T) {
	d, evs := collect()
	for _, b := range []byte("\x1b[1;5A") {
		d.Feed([]byte{b})
	}
	want(t, "byte-by-byte csi", evs, key(event.KeyUp, 0, event.ModCtrl))
	if d.Pending() {
		t.Error("completed CSI must not be pending")
	}
}

func TestFlushKeepsPartialCSI(t *testing.T) {
	// 半截 CSI 不是 Esc：flush 只结束裸 ESC，不动序列缓冲。
	d, evs := collect()
	d.Feed([]byte("\x1b["))
	if !d.Pending() {
		t.Fatal("partial CSI must be pending")
	}
	d.Flush()
	want(t, "flush partial csi", evs)
	d.Feed([]byte("A"))
	want(t, "csi completes", evs, key(event.KeyUp, 0, event.ModNone))
}

// ---- 鼠标（SGR1006） ----

func TestMouseSGR(t *testing.T) {
	cases := []struct {
		in   string
		want event.Event
	}{
		{"\x1b[<0;10;5M", event.Mouse{X: 9, Y: 4, Kind: event.MousePress, Button: event.ButtonLeft}},
		{"\x1b[<0;10;5m", event.Mouse{X: 9, Y: 4, Kind: event.MouseRelease, Button: event.ButtonLeft}},
		{"\x1b[<1;3;4M", event.Mouse{X: 2, Y: 3, Kind: event.MousePress, Button: event.ButtonMiddle}},
		{"\x1b[<2;10;5M", event.Mouse{X: 9, Y: 4, Kind: event.MousePress, Button: event.ButtonRight}},
		{"\x1b[<0;1;1M", event.Mouse{X: 0, Y: 0, Kind: event.MousePress, Button: event.ButtonLeft}},
		{"\x1b[<64;10;5M", event.Mouse{X: 9, Y: 4, Kind: event.MouseWheelUp}},
		{"\x1b[<65;10;5M", event.Mouse{X: 9, Y: 4, Kind: event.MouseWheelDown}},
		{"\x1b[<35;10;5M", event.Mouse{X: 9, Y: 4, Kind: event.MouseMove, Button: event.ButtonNone}}, // 悬停
		{"\x1b[<32;10;5M", event.Mouse{X: 9, Y: 4, Kind: event.MouseMove, Button: event.ButtonLeft, Drag: true}},
		{"\x1b[<33;10;5M", event.Mouse{X: 9, Y: 4, Kind: event.MouseMove, Button: event.ButtonMiddle, Drag: true}},
		{"\x1b[<4;10;5M", event.Mouse{X: 9, Y: 4, Kind: event.MousePress, Button: event.ButtonLeft, Mod: event.ModShift}},
		{"\x1b[<8;10;5M", event.Mouse{X: 9, Y: 4, Kind: event.MousePress, Button: event.ButtonLeft, Mod: event.ModAlt}},
		{"\x1b[<16;10;5M", event.Mouse{X: 9, Y: 4, Kind: event.MousePress, Button: event.ButtonLeft, Mod: event.ModCtrl}},
		{"\x1b[<68;5;5M", event.Mouse{X: 4, Y: 4, Kind: event.MouseWheelUp, Mod: event.ModShift}},
		{"\x1b[<3;10;5M", event.Mouse{X: 9, Y: 4, Kind: event.MouseMove, Button: event.ButtonNone}},
	}
	for _, c := range cases {
		d, evs := collect()
		d.Feed([]byte(c.in))
		want(t, "mouse "+c.in, evs, c.want)
	}
}

func TestMouseSplitAcrossFeeds(t *testing.T) {
	d, evs := collect()
	d.Feed([]byte("\x1b[<0;1"))
	if !d.Pending() {
		t.Fatal("partial mouse must be pending")
	}
	d.Feed([]byte("0;5M"))
	want(t, "split mouse", evs,
		event.Mouse{X: 9, Y: 4, Kind: event.MousePress, Button: event.ButtonLeft})
}

// ---- 括号粘贴 ----

func TestPaste(t *testing.T) {
	d, evs := collect()
	d.Feed([]byte("\x1b[200~hello\nworld\x1b[201~"))
	want(t, "paste", evs, event.Paste{Text: "hello\nworld"})
	if d.Pending() {
		t.Error("completed paste must not be pending")
	}
}

func TestPasteSplitAcrossFeeds(t *testing.T) {
	d, evs := collect()
	d.Feed([]byte("\x1b[200~hel"))
	d.Feed([]byte("lo"))
	if !d.Pending() {
		t.Error("unclosed paste must be pending")
	}
	if len(*evs) != 0 {
		t.Errorf("paste must not emit before terminator, got %v", *evs)
	}
	// 终止符自身被拆开。
	d.Feed([]byte("\x1b[20"))
	d.Feed([]byte("1~"))
	want(t, "split paste", evs, event.Paste{Text: "hello"})
}

func TestPasteEmptyAndEscInside(t *testing.T) {
	d, evs := collect()
	d.Feed([]byte("\x1b[200~\x1b[201~"))
	want(t, "empty paste", evs, event.Paste{Text: ""})

	// 粘贴体内的 ESC 序列必须当作字面文本，不得解析为按键。
	d.Feed([]byte("\x1b[200~a\x1b[Ab\x1b[201~"))
	want(t, "esc inside paste", evs, event.Paste{Text: "a\x1b[Ab"})
}

func TestPasteThenKey(t *testing.T) {
	d, evs := collect()
	d.Feed([]byte("\x1b[200~x\x1b[201~\x1b[A"))
	want(t, "paste followed by key", evs,
		event.Paste{Text: "x"},
		key(event.KeyUp, 0, event.ModNone),
	)
}

// ---- 焦点 ----

func TestFocusEvents(t *testing.T) {
	d, evs := collect()
	d.Feed([]byte("\x1b[I"))
	want(t, "focus in", evs, event.Focus{Gained: true})
	d.Feed([]byte("\x1b[O"))
	want(t, "focus out", evs, event.Focus{Gained: false})
}

// ---- 未知序列吞掉 ----

func TestUnknownSequencesSwallowed(t *testing.T) {
	cases := []string{
		"\x1b[999~",                        // 未知波浪号键
		"\x1b[?1006h",                      // 终端查询/模式应答
		"\x1b[1;5Q",                        // 未知修饰序列
		"\x1b]0;title\x07",                 // OSC 标题（BEL 结尾）
		"\x1b]11;rgb:0000/0000/0000\x1b\\", // OSC（ST 结尾）
		"\x1bP1;2|payload\x1b\\",           // DCS
	}
	for _, in := range cases {
		d, evs := collect()
		d.Feed([]byte(in))
		want(t, "swallow "+in, evs)
		if d.Pending() {
			t.Errorf("swallowed sequence must not remain pending: %q", in)
		}
	}
}

func TestInvalidByteSwallowed(t *testing.T) {
	d, evs := collect()
	d.Feed([]byte{0xff})
	want(t, "invalid byte", evs)
	if d.Pending() {
		t.Error("invalid byte must be consumed")
	}
}

func TestTrailingGarbageAfterSequence(t *testing.T) {
	d, evs := collect()
	d.Feed([]byte("\x1b[?2004h"))
	want(t, "mode set swallowed", evs)
}
