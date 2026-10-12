# PaintedSkin

A from-scratch Go TUI framework: GUI-grade widgets in the terminal (full mouse support, focus management, modal dialogs, constraint layout) while keeping the terseness of command-line tools.

## Core Advantages

| Advantage | Description |
|---|---|
| **Zero dependencies** | Only `golang.org/x/sys` + `github.com/mattn/go-runewidth`; everything else is self-built |
| **Full mouse support** | Hover, click, drag, scroll wheel, modifier keys; SGR(1006) protocol |
| **15 widgets** | Label / Button / Checkbox / Radio / TextInput / Slider / ProgressBar / List / Table / Tabs / Dropdown / Scrollbar / Panel / Dialog / StatusBar / Separator |
| **GUI-grade interaction** | z-order hit testing, mouse capture, event bubbling, hover synthesis, focus trap, modal dialogs |
| **Constraint layout** | Measure → Layout → Paint pipeline; Flex (Grow/Shrink/Align), Stack, clipping & scroll clamping |
| **Double-buffered rendering** | Cell-level diff → minimal ANSI sequences; wide-char stepping, color depth auto-fallback |
| **Cross-platform** | macOS / Linux / Windows (VT console mode), cross-compilation verified |
| **CJK optimized** | Full-width chars count as 2 columns, no mid-char truncation, correct mixed CJK/ASCII layout |
| **Test-driven** | Every behavior covered by unit tests; golden byte-sequence regression; doc examples validated |

## Quick Start

```bash
go run ./examples/minimal     # minimal demo
go run ./examples/gallery     # full widget gallery
go run ./examples/preview     # headless render (no TTY needed)
```

### Up and running in 30 lines

```go
package main

import (
	"PaintedSkin"
	"PaintedSkin/app"
	"PaintedSkin/geom"
)

func main() {
	err := app.Run(app.Config{
		Setup: func(a *app.App) {
			label := paintedskin.Label("Press Tab to focus the button")
			btn := paintedskin.Button("Click me", func() { label.SetText("clicked") })

			a.Root().Add(label)
			a.Root().Add(btn)
			label.Layout(geom.Rect{X: 2, Y: 2, W: 30, H: 1})
			btn.Layout(geom.Rect{X: 2, Y: 4, W: 10, H: 1})
		},
	})
	if err != nil {
		panic(err)
	}
}
```

### Constructor-sugar containers (recommended)

```go
col := paintedskin.Column(
	paintedskin.Gap(0, paintedskin.Label("Title")),
	paintedskin.Gap(1, paintedskin.Button("OK", onOK)),
	paintedskin.Gap(1, paintedskin.HSlider(40)),
	paintedskin.Gap(1, paintedskin.Fixed(8, paintedskin.List(items...))),
	paintedskin.Grow(1, paintedskin.TextInput()),
)
col.Layout(area)
```

### Thread-safe updates

```go
go func() {
	for i := 0; i <= 100; i += 5 {
		v := i
		a.Post(func() { bar.SetValue(v) }) // marshal to UI thread
		time.Sleep(80 * time.Millisecond)
	}
}()
```

## Package Layout

```
PaintedSkin/
├── geom/    Point/Size/Rect/Insets
├── style/   Color (default/16/256/rgb), Attrs, Style
├── event/   Key/Mouse/Resize/Paste/Focus
├── term/    terminal backend (unix / windows)
├── input/   incremental byte-stream to event decoder
├── screen/  Cell, double buffer, diff rendering, Canvas
├── widget/  Widget interface, Base, FocusManager, Root routing
│   └── layout/  constraint layout: Flex, Stack, ClampScroll
├── theme/   built-in themes (Dark / Light)
├── app/     event loop, Post/Flush, lifecycle & restore
└── examples/ minimal, gallery, preview
```

Unidirectional dependency graph: `geom ← style ← event → input → screen ← widget ← theme`, with `app` at the top.

## Testing

```bash
./go.sh vet ./...      # static analysis
./go.sh test ./...     # full unit tests
```

`openpty` is denied in sandbox, so the term layer uses an injected in-memory fake terminal. Cross-platform verification:

```bash
GOOS=linux   ./go.sh build ./...
GOOS=windows ./go.sh build ./...
```

## Documentation

- [English usage guide](usage.en.md) — getting started, event handling, layout, 15 widgets, theming, CJK/ASCII
- [中文使用文档](usage.zh-CN.md) — 快速开始、事件处理、布局、15 个控件、主题、中英混排
