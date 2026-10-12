package term

import (
	"bytes"
	"strings"
	"testing"
)

// Options 是打开终端时的模式开关。
func TestEnterSequences(t *testing.T) {
	var buf bytes.Buffer
	o := Options{Mouse: true, Paste: true, FocusReports: true, AltScreen: true}
	Enter(&buf, o)
	got := buf.String()
	for _, want := range []string{
		"\x1b[?1049h", // 备用屏
		"\x1b[?25l",   // 隐藏光标
		"\x1b[?1002h", // 鼠标按键跟踪
		"\x1b[?1003h", // 任意移动
		"\x1b[?1006h", // SGR 扩展
		"\x1b[?2004h", // 括号粘贴
		"\x1b[?1004h", // 焦点事件
	} {
		if !strings.Contains(got, want) {
			t.Errorf("enter missing %q: %q", want, got)
		}
	}
}

// 关闭序列必须与开启一一对应且反向（docs/02 恢复清单）。
func TestLeaveSequences(t *testing.T) {
	var buf bytes.Buffer
	o := Options{Mouse: true, Paste: true, FocusReports: true, AltScreen: true}
	Leave(&buf, o)
	got := buf.String()
	for _, want := range []string{
		"\x1b[?1006l", "\x1b[?1003l", "\x1b[?1002l",
		"\x1b[?2004l", "\x1b[?1004l",
		"\x1b[?25h",   // 显示光标
		"\x1b[?1049l", // 退出备用屏
	} {
		if !strings.Contains(got, want) {
			t.Errorf("leave missing %q: %q", want, got)
		}
	}
	// 备用屏退出必须是最后一条（其余序列在备用屏内失效）
	if !strings.HasSuffix(got, "\x1b[?1049l") {
		t.Errorf("1049l must be last: %q", got)
	}
}

// 选项关闭时不写对应序列（WithMouse(false) 降级路径）。
func TestOptionsGateSequences(t *testing.T) {
	var buf bytes.Buffer
	Enter(&buf, Options{})
	got := buf.String()
	for _, unwanted := range []string{"1002h", "1003h", "1006h", "2004h", "1004h"} {
		if strings.Contains(got, unwanted) {
			t.Errorf("disabled mode must not be written: %q in %q", unwanted, got)
		}
	}
	// 光标隐藏始终写；备用屏可被显式关闭（调试模式画主屏）
	if !strings.Contains(got, "\x1b[?25l") {
		t.Errorf("cursor hide must always be written: %q", got)
	}
	if strings.Contains(got, "1049h") {
		t.Errorf("AltScreen=false must not enter alt screen: %q", got)
	}

	// 显式开启备用屏时应写入
	var buf3 bytes.Buffer
	Enter(&buf3, Options{AltScreen: true})
	if !strings.Contains(buf3.String(), "\x1b[?1049h") {
		t.Errorf("AltScreen=true must enter alt screen: %q", buf3.String())
	}

	var buf2 bytes.Buffer
	Leave(&buf2, Options{})
	if strings.Contains(buf2.String(), "1002l") {
		t.Errorf("disabled mode must not be undone: %q", buf2.String())
	}
}

// 光标显隐由宿主按需控制（TextInput 聚焦时显示）。
func TestCursorShowHide(t *testing.T) {
	var buf bytes.Buffer
	ShowCursor(&buf)
	HideCursor(&buf)
	if got := buf.String(); got != "\x1b[?25h\x1b[?25l" {
		t.Errorf("cursor = %q", got)
	}
}

// 能力探测：真彩 → 256 → 16（docs/03 降级表）。
func TestProbeColorDepth(t *testing.T) {
	env := map[string]string{
		"COLORTERM": "truecolor", "TERM": "xterm-256color",
	}
	if d := ColorDepth(env); d != "truecolor" {
		t.Errorf("truecolor = %q", d)
	}
	if d := ColorDepth(map[string]string{"TERM": "xterm-256color"}); d != "256" {
		t.Errorf("256 = %q", d)
	}
	if d := ColorDepth(map[string]string{"TERM": "xterm-color"}); d != "16" {
		t.Errorf("16 = %q", d)
	}
	if d := ColorDepth(map[string]string{"TERM": "dumb"}); d != "16" {
		t.Errorf("dumb = %q", d)
	}
	if d := ColorDepth(nil); d != "16" {
		t.Errorf("empty = %q", d)
	}
}

// TERM=dumb 或未设 TERM → 判定为无鼠标能力。
func TestProbeMouse(t *testing.T) {
	if HasMouse(map[string]string{"TERM": "dumb"}) {
		t.Error("dumb must disable mouse")
	}
	if HasMouse(map[string]string{}) {
		t.Error("no TERM must disable mouse")
	}
	if !HasMouse(map[string]string{"TERM": "xterm-256color"}) {
		t.Error("xterm must support mouse")
	}
}

// Close 幂等：重复调用只恢复一次（AGENT.md 2.4）。
func TestCloseIdempotent(t *testing.T) {
	var buf bytes.Buffer
	tm := newFakeBase(&buf, Options{Mouse: true, Paste: true, FocusReports: true, AltScreen: true})
	buf.Reset()

	if err := tm.Close(); err != nil {
		t.Fatalf("first close: %v", err)
	}
	first := buf.String()
	if first == "" {
		t.Fatal("first close must restore")
	}
	for i := 0; i < 3; i++ {
		if err := tm.Close(); err != nil {
			t.Fatalf("close #%d: %v", i, err)
		}
	}
	if buf.String() != first {
		t.Errorf("restore must happen exactly once:\nfirst=%q\nnow=%q", first, buf.String())
	}
	if !tm.Closed() {
		t.Error("Closed must be true")
	}
}

// 恢复序列覆盖 docs/02 清单全部 7 项。
func TestRestoreCoversAll(t *testing.T) {
	var buf bytes.Buffer
	tm := newFakeBase(&buf, Options{Mouse: true, Paste: true, FocusReports: true, AltScreen: true})
	tm.Close()
	got := buf.String()
	for _, want := range []string{
		"\x1b[?1049l", "\x1b[?25h",
		"\x1b[?1002l", "\x1b[?1003l", "\x1b[?1006l",
		"\x1b[?2004l", "\x1b[?1004l",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("restore missing %q: %q", want, got)
		}
	}
}
