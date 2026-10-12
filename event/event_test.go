package event

import "testing"

func TestKeyRuneString(t *testing.T) {
	k := Key{Code: KeyRune, Rune: 'a'}
	if k.String() != "a" {
		t.Errorf("String()=%q want %q", k.String(), "a")
	}
	kc := Key{Code: KeyRune, Rune: '中', Mod: ModCtrl}
	if kc.String() != "Ctrl+中" {
		t.Errorf("String()=%q want Ctrl+中", kc.String())
	}
}

func TestKeyNamed(t *testing.T) {
	cases := map[KeyCode]string{
		KeyEnter:     "Enter",
		KeyEsc:       "Esc",
		KeyTab:       "Tab",
		KeyUp:        "Up",
		KeyDown:      "Down",
		KeyLeft:      "Left",
		KeyRight:     "Right",
		KeyBackspace: "Backspace",
		KeyDelete:    "Delete",
		KeyHome:      "Home",
		KeyEnd:       "End",
		KeyPgUp:      "PgUp",
		KeyPgDn:      "PgDn",
		KeyF1:        "F1",
		KeyF12:       "F12",
		KeySpace:     "Space",
	}
	for code, want := range cases {
		if got := (Key{Code: code}).String(); got != want {
			t.Errorf("Key{Code:%v}.String()=%q want %q", code, got, want)
		}
	}
}

func TestKeyModPrefix(t *testing.T) {
	k := Key{Code: KeyUp, Mod: ModShift | ModCtrl}
	if k.String() != "Ctrl+Shift+Up" {
		t.Errorf("String()=%q want Ctrl+Shift+Up", k.String())
	}
}

func TestKeyOrdering(t *testing.T) {
	// Tab 必须先于普通可打印键参与焦点遍历判断——用字段语义做基本断言。
	if KeySpace != 0x20 {
		t.Errorf("KeySpace=%#x want 0x20", KeySpace)
	}
	if KeyRune == KeyEnter {
		t.Error("KeyRune must differ from KeyEnter")
	}
}

func TestMouseEventKinds(t *testing.T) {
	m := Mouse{X: 5, Y: 6, Kind: MousePress, Button: ButtonLeft}
	if m.X != 5 || m.Y != 6 {
		t.Errorf("position wrong: %+v", m)
	}
	if m.String() != "Press(Left)@5,6" {
		t.Errorf("String()=%q", m.String())
	}
	w := Mouse{Kind: MouseWheelUp, X: 1, Y: 2}
	if w.String() != "WheelUp@1,2" {
		t.Errorf("wheel String()=%q", w.String())
	}
	rel := Mouse{Kind: MouseRelease, Button: ButtonRight, X: 3, Y: 3}
	if rel.String() != "Release(Right)@3,3" {
		t.Errorf("release String()=%q", rel.String())
	}
}

func TestModString(t *testing.T) {
	if (ModCtrl | ModShift).String() != "Ctrl+Shift" {
		t.Errorf("got %q", (ModCtrl | ModShift).String())
	}
	if ModNone.String() != "" {
		t.Errorf("ModNone should be empty, got %q", ModNone.String())
	}
}
