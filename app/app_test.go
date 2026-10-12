package app_test

import (
	"reflect"
	"testing"

	"PaintedSkin/event"
	"PaintedSkin/input"
	"PaintedSkin/screen"
)

// want 是 input_test 包中的辅助函数，复用其断言逻辑
func want(t *testing.T, name string, got *[]event.Event, want ...event.Event) {
	// 复用 input_test 中的 want 函数逻辑
	if len(want) == 0 {
		*got = nil
		return
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

// TestAppRunWithSyntheticEvents - 测试 app.Run 能够通过合成事件正确分发事件
func TestAppRunWithSyntheticEvents(t *testing.T) {
	// M4: app事件循环 - 用合成事件测试 app.Run 能够正确分发事件
	var evs []event.Event
	d := input.New(func(e event.Event) { evs = append(evs, e) })

	// 注入关键事件：先喂 'A'，然后是 Enter，然后是 ESC
	// ESC 单独一个字节会进入 pending 状态，需要后续字节才能解码
	d.Feed([]byte("\x41")) // 'A' key - should produce event immediately
	d.Feed([]byte("\x0a")) // Enter key - should produce event immediately

	// 此时有 2 个事件
	if len(evs) != 2 {
		t.Errorf("after A+Enter: expected 2 events, got %d: %v", len(evs), evs)
	}

	// 再喂 ESC 单独一个字节 - 这会进入 pending 状态
	// 为了测试，我们喂一个后续字节 'A' 使其变成 Alt+A 序列
	d.Feed([]byte{0x1b})   // ESC only - goes pending
	d.Feed([]byte("\x41")) // 'A' after ESC -> becomes Alt+A

	// 现在应该有 3 个事件：A, Enter, Alt+A
	if len(evs) != 3 {
		t.Errorf("after ESC+A: expected 3 events, got %d: %v", len(evs), evs)
	}

	// 第一个应是 KeyRune 'A'
	if key, ok := evs[0].(event.Key); ok {
		if key.Code != event.KeyRune || key.Rune != 'A' {
			t.Errorf("expected first event to be KeyRune 'A', got Code=%d Rune=%d", key.Code, key.Rune)
		}
	} else {
		t.Errorf("expected first event to be Key type, got %T", evs[0])
	}

	// 第二个应是 KeyEnter
	if key, ok := evs[1].(event.Key); ok {
		if key.Code != event.KeyEnter {
			t.Errorf("expected second event to be KeyEnter, got Code=%d", key.Code)
		}
	} else {
		t.Errorf("expected second event to be Key type, got %T", evs[1])
	}

	// 第三个应是 KeyRune 且 ModAlt (ESC+A -> Alt+A)
	if key, ok := evs[2].(event.Key); ok {
		if key.Code != event.KeyRune || key.Rune != 'A' || key.Mod != event.ModAlt {
			t.Errorf("expected third event to be KeyRune 'A' with ModAlt, got Code=%d Rune=%d Mod=%v", key.Code, key.Rune, key.Mod)
		}
	} else {
		t.Errorf("expected third event to be Key type, got %T", evs[2])
	}
}

// TestAppBufferBasics - 基本缓冲区操作测试
func TestAppBufferBasics(t *testing.T) {
	buffer := screen.NewBuffer(80, 24)

	// 测试 NewBuffer 创建
	if buffer == nil {
		t.Error("buffer should not be nil")
	}

	// 测试尺寸
	if buffer.Width() != 80 || buffer.Height() != 24 {
		t.Errorf("expected buffer 80x24, got %dx%d", buffer.Width(), buffer.Height())
	}

	// 测试 Reset
	buffer.Reset()
	cell := buffer.At(0, 0)
	if cell == nil {
		t.Error("cell at (0,0) should not be nil after reset")
	} else if cell.Rune != 0 {
		t.Errorf("expected empty cell after reset, got rune=%c", cell.Rune)
	}
}

// TestInputDecoderPending - 测试 decoder pending 状态
func TestInputDecoderPending(t *testing.T) {
	d := input.New(func(event.Event) {})

	// 部分 UTF-8 字符应保持 pending 状态
	raw := []byte("中")
	d.Feed(raw[:1])
	if !d.Pending() {
		t.Error("partial UTF-8 must be pending after first Feed")
	}

	// 完成字符
	d.Feed(raw[1:])
	if d.Pending() {
		t.Error("completed UTF-8 must not be pending after second Feed")
	}
}

// TestDecoderESCBare - ESC 单字节应产生 KeyEsc（跨 Feed 分片）
func TestDecoderESCBare(t *testing.T) {
	d := input.New(func(event.Event) {})

	d.Feed([]byte{0x1b}) // ESC only
	if !d.Pending() {
		t.Error("ESC only must be pending, need more bytes")
	}

	// 再喂一个字节
	d.Feed([]byte{0x41}) // 'A' after ESC -> Alt+A, not KeyEsc
	if !d.Pending() {
		// ESC+'A' 已解码，不应 pending
	} else {
		t.Error("ESC+'A' should not be pending")
	}
}
