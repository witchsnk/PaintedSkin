// Package event 定义框架内流转的全部事件类型。
// 事件是不可变值：解码器产出后不再修改，主循环串行消费。
package event

import "fmt"

// Event 是所有事件的根接口。
type Event interface {
	isEvent()
}

// ---- 键盘 ----

// KeyCode 是按键的符号编码。
// KeyRune 表示一个可打印字符（值存于 Key.Rune），其余为具名键。
type KeyCode int

const (
	// KeyRune 表示可打印字符，具体字符在 Key.Rune。
	KeyRune KeyCode = iota
	KeyEnter
	KeyEsc
	KeyTab
	KeyBackspace
	KeyInsert
	KeyDelete
	KeyUp
	KeyDown
	KeyLeft
	KeyRight
	KeyHome
	KeyEnd
	KeyPgUp
	KeyPgDn
	KeyF1
	KeyF2
	KeyF3
	KeyF4
	KeyF5
	KeyF6
	KeyF7
	KeyF8
	KeyF9
	KeyF10
	KeyF11
	KeyF12
)

// KeySpace 是空格键。取其字符值 0x20，便于与解码出的空格字节直接比较；
// 也因为空格既是要录入 TextInput 的字符，又是 Button 的激活键，需要独立编码。
const KeySpace KeyCode = 0x20

// String 返回键名，用于快捷键展示与测试断言。
func (c KeyCode) String() string {
	switch c {
	case KeyRune:
		return "Rune"
	case KeyEnter:
		return "Enter"
	case KeyEsc:
		return "Esc"
	case KeyTab:
		return "Tab"
	case KeyBackspace:
		return "Backspace"
	case KeyInsert:
		return "Insert"
	case KeyDelete:
		return "Delete"
	case KeyUp:
		return "Up"
	case KeyDown:
		return "Down"
	case KeyLeft:
		return "Left"
	case KeyRight:
		return "Right"
	case KeyHome:
		return "Home"
	case KeyEnd:
		return "End"
	case KeyPgUp:
		return "PgUp"
	case KeyPgDn:
		return "PgDn"
	case KeySpace:
		return "Space"
	case KeyF1:
		return "F1"
	case KeyF2:
		return "F2"
	case KeyF3:
		return "F3"
	case KeyF4:
		return "F4"
	case KeyF5:
		return "F5"
	case KeyF6:
		return "F6"
	case KeyF7:
		return "F7"
	case KeyF8:
		return "F8"
	case KeyF9:
		return "F9"
	case KeyF10:
		return "F10"
	case KeyF11:
		return "F11"
	case KeyF12:
		return "F12"
	}
	return fmt.Sprintf("Key(%d)", int(c))
}

// Key 是键盘事件。
// Press 为 false 表示释放（仅 kitty 协议启用时出现）。
type Key struct {
	Code  KeyCode
	Rune  rune // Code==KeyRune 时有效
	Mod   Mod  // 修饰键掩码
	Press bool // 释放事件为 false
}

func (Key) isEvent() {}

// String 返回可读形式，如 "Ctrl+Shift+Up"、"a"、"Esc"。
func (k Key) String() string {
	s := k.Mod.String()
	if k.Code == KeyRune {
		if s != "" {
			s += "+"
		}
		return s + string(k.Rune)
	}
	if s != "" {
		s += "+"
	}
	return s + k.Code.String()
}

// CtrlRune 返回 Ctrl+字母对应的控制字符 rune（如 Ctrl+C → 0x03）。
// 输入解码器用它把控制字节归一化为带 ModCtrl 的字符键。
func CtrlRune(r rune) rune { return r & 0x1F }

// ---- 修饰键 ----

// Mod 是修饰键位掩码。
type Mod uint8

const (
	ModNone  Mod = 0
	ModShift Mod = 1 << iota
	ModCtrl
	ModAlt
	ModMeta
)

// String 按 Ctrl+Shift+Alt+Meta 顺序返回前缀，无修饰返回空串。
func (m Mod) String() string {
	var parts []string
	if m&ModCtrl != 0 {
		parts = append(parts, "Ctrl")
	}
	if m&ModShift != 0 {
		parts = append(parts, "Shift")
	}
	if m&ModAlt != 0 {
		parts = append(parts, "Alt")
	}
	if m&ModMeta != 0 {
		parts = append(parts, "Meta")
	}
	out := ""
	for i, p := range parts {
		if i > 0 {
			out += "+"
		}
		out += p
	}
	return out
}

// ---- 鼠标 ----

// MouseKind 区分鼠标动作类别。
type MouseKind uint8

const (
	// MousePress 按下按键。
	MousePress MouseKind = iota
	// MouseRelease 释放按键。
	MouseRelease
	// MouseMove 移动。按下时为拖拽，未按键时为悬停。
	MouseMove
	// MouseWheelUp 滚轮向上。
	MouseWheelUp
	// MouseWheelDown 滚轮向下。
	MouseWheelDown
	// MouseEnter 悬停进入（widget 层由 Move 合成，input 解码器永不产出）。
	MouseEnter
	// MouseLeave 悬停离开（widget 层由 Move 合成，按下捕获期间不发）。
	MouseLeave
)

func (k MouseKind) String() string {
	switch k {
	case MousePress:
		return "Press"
	case MouseRelease:
		return "Release"
	case MouseMove:
		return "Move"
	case MouseWheelUp:
		return "WheelUp"
	case MouseWheelDown:
		return "WheelDown"
	case MouseEnter:
		return "Enter"
	case MouseLeave:
		return "Leave"
	}
	return "Mouse?"
}

// MouseButton 是鼠标按键。
type MouseButton uint8

const (
	// ButtonNone 表示无按键（悬停移动）。
	ButtonNone MouseButton = iota
	ButtonLeft
	ButtonMiddle
	ButtonRight
)

func (b MouseButton) String() string {
	switch b {
	case ButtonLeft:
		return "Left"
	case ButtonMiddle:
		return "Middle"
	case ButtonRight:
		return "Right"
	}
	return "None"
}

// Mouse 是鼠标事件，坐标为 0-based 单元格坐标。
type Mouse struct {
	X, Y   int
	Kind   MouseKind
	Button MouseButton
	Mod    Mod
	// Drag 为 true 表示按住移动（由 MouseMove + Button 推导时由解码器填充）。
	Drag bool
}

func (Mouse) isEvent() {}

// String 返回可读形式，如 "Press(Left)@5,6"。
func (m Mouse) String() string {
	switch m.Kind {
	case MouseMove:
		if m.Drag {
			return fmt.Sprintf("Drag(%s)@%d,%d", m.Button, m.X, m.Y)
		}
		return fmt.Sprintf("Move@%d,%d", m.X, m.Y)
	case MouseWheelUp, MouseWheelDown:
		return fmt.Sprintf("%s@%d,%d", m.Kind, m.X, m.Y)
	case MouseEnter:
		return fmt.Sprintf("Enter@%d,%d", m.X, m.Y)
	case MouseLeave:
		return fmt.Sprintf("Leave@%d,%d", m.X, m.Y)
	default:
		return fmt.Sprintf("%s(%s)@%d,%d", m.Kind, m.Button, m.X, m.Y)
	}
}

// ---- 其他事件 ----

// Resize 是终端尺寸变化事件。
type Resize struct {
	W, H int
}

func (Resize) isEvent() {}

// Paste 是括号粘贴事件（一次完整粘贴内容）。
type Paste struct {
	Text string
}

func (Paste) isEvent() {}

// Focus 是终端焦点变化事件。
type Focus struct {
	Gained bool
}

func (Focus) isEvent() {}
