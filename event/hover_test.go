package event

import "testing"

// M5 widget 核心需要 hover 合成事件：Root 在 Move 越界时向新旧控件
// 分别派发 MouseEnter / MouseLeave（按下期间不发 Leave）。
func TestMouseHoverKinds(t *testing.T) {
	enter := Mouse{X: 5, Y: 6, Kind: MouseEnter, Button: ButtonNone}
	if enter.String() != "Enter@5,6" {
		t.Errorf("enter String()=%q want %q", enter.String(), "Enter@5,6")
	}
	leave := Mouse{X: 5, Y: 6, Kind: MouseLeave, Button: ButtonNone}
	if leave.String() != "Leave@5,6" {
		t.Errorf("leave String()=%q want %q", leave.String(), "Leave@5,6")
	}
	if MouseEnter == MouseMove || MouseLeave == MouseMove {
		t.Error("hover kinds must differ from MouseMove")
	}
	if MouseEnter == MousePress || MouseLeave == MouseRelease {
		t.Error("hover kinds must not alias press/release")
	}
}
