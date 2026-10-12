# PaintedSkin Usage Guide (English)

> 中文版：[usage.zh-CN.md](usage.zh-CN.md)

A from-scratch Go TUI framework: GUI-grade widgets in the terminal (full mouse
support, focus management, modal dialogs, constraint layout) while keeping the
terseness of command-line tools.

```bash
go run ./examples/minimal    # minimal demo (requires a real terminal)
go run ./examples/gallery    # full widget gallery (requires a real terminal)
go run ./examples/preview    # headless render: prints the frame to stdout
```

---

## Table of Contents

1. [Getting Started](#1-getting-started)
2. [Core Concepts](#2-core-concepts)
3. [Unified Event Handling](#3-unified-event-handling)
4. [Layout](#4-layout)
5. [Widget Reference](#5-widget-reference)
6. [Overlays: Dialog and Overlay](#6-overlays-dialog-and-overlay)
7. [Theming](#7-theming)
8. [Text Width and CJK/ASCII Mixed Content](#8-text-width-and-cjkascii-mixed-content)
9. [Style and Color](#9-style-and-color)
10. [Threading Model](#10-threading-model)
11. [Terminal Capabilities and Degradation](#11-terminal-capabilities-and-degradation)
12. [Headless / No TTY](#12-headless-no-tty)
13. [Custom Widgets](#13-custom-widgets)
14. [Testing and Debugging](#14-testing-and-debugging)
15. [Complete Examples](#15-complete-examples)

---

## 1. Getting Started

### 1.1 Minimal application

```go
package main

import (
	"PaintingSkin"
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

`Setup` runs exactly once, after the terminal is claimed and before the event
loop starts, so `a.Size()` inside it returns the real terminal dimensions.

### 1.2 Three ways to run

| Mode | Entry point | Purpose |
|---|---|---|
| Interactive | `app.Run(cfg)` | Claim terminal, run event loop, Esc to quit |
| Headless | `app.New(cfg)` + `a.Draw()` | No terminal; frame lands in `a.Buffer()` |
| Embedded test | `app.New(cfg)` + `a.Dispatch(e)` | Inject synthetic events, assert state |

See [§12](#12-headless-no-tty) for headless rendering.

---

## 2. Core Concepts

### 2.1 Retained-mode object tree

Widgets are **persistent objects**, not per-frame closures:

```go
btn := paintedskin.Button("OK", onConfirm)
btn.SetDisabled(true)        // mutate any time
btn.SetText("Submitting…")
label.SetText("Done")        // Label updated in place
```

### 2.2 Four-phase lifecycle

```
Measure(Constraints) → Size     bottom-up:   "how big do I want to be?"
Layout(Rect)                     top-down:    "here is your area"
Paint(Canvas)                    draw (fill own background first; clipped by Canvas/parent)
Handle(Event) bool                               return true = consumed
```

See `docs/06-widget-model.md`.

> **Trailing-gap rule**: every `Paint` starts by filling its own background
> (`Canvas.FillStyle`), so inside `Bounds` each cell is either real content or
> a space carrying the widget's own background. No theme background gaps, no
> highlight stopping at the end of the text, no terminal background leaking
> through a container. State highlights (focus/selected/pressed) always span
> the full widget width. **All 15 widgets** follow this rule.

### 2.3 Semantic states (focus / hover / selected / disabled)

These four states come from the theme — there is **no** hardcoded `AttrReverse`
or `WithDim` left in the widgets:

| State | Theme field | Fallback without a theme |
|---|---|---|
| focus | `FocusColor` (foreground) | `AttrReverse` |
| hover | `HoverBG` (background) | `Bold` |
| selected | `SelectionBG` (background) | `AttrReverse` |
| disabled | `DisabledColor` (foreground) | `Dim` |

Priority: **widget's own `Style` > theme semantic color > built-in fallback**.

### 2.4 Focus ring (outline)

```go
ti.SetFocusRing(true)      // ring the Bounds when focused (off by default)
ti.FocusRingEnabled()
```

The ring only **changes styles** — characters and widths are untouched, so it
eats no content and takes no layout space (`Bounds` is unchanged). It is drawn
at the end of `Paint` (after the content), so the whole ring is visible. On a
widget without a frame the ring covers the first/last characters — leave those
on the semantic color instead of enabling the ring.

### 2.5 Package layout and dependency direction

```
geom ← style ← event
        ↑        ↑          text  (single width authority, shared by screen/widget)
      screen   input
        ↑
    widget ← theme      layout
        ↑
       app  ←  root package paintedskin (constructor sugar)
```

Dependencies are acyclic: `widget` must **not** import `term`/`app`; `screen`
must **not** import `widget`/`layout`.

### 2.6 Core types

```go
// geom: cell grid, (0,0) top-left, X right, Y down
type Point struct{ X, Y int }
type Size  struct{ W, H int }
type Rect  struct{ X, Y, W, H int }   // right/bottom are exclusive
type Insets struct{ Top, Right, Bottom, Left int }

// event
type Key   struct{ Code KeyCode; Rune rune; Mod Mod; Press bool }
type Mouse struct{ X, Y int; Kind MouseKind; Button MouseButton; Mod Mod; Drag bool }
type Resize struct{ W, H int }
type Paste  struct{ Text string }
type Focus  struct{ Gained bool }
```

---

## 3. Unified Event Handling

**All input enters through `Root.Dispatch`.** The framework owns hit testing,
mouse capture, bubbling, and focus routing; a widget only implements `Handle`.

### 3.1 Where events come from

```
terminal bytes ──► input.Decoder ──┐
                                 ├──► Root.Dispatch ──► widget tree
SIGWINCH / polling ─► event.Resize┘
```

Application code normally never calls `Dispatch` directly — the loop in
`app.Run` drives it. Call it directly for tests or headless rendering:

```go
a.Dispatch(event.Key{Code: event.KeyEnter, Press: true})
a.Dispatch(event.Mouse{X: 3, Y: 5, Kind: event.MousePress, Button: event.ButtonLeft})
```

### 3.2 Routing rules (`docs/06`)

**Mouse**

1. `HitTest` from the root, **topmost z-order first** (later-added / later-painted wins)
2. The widget hit by `Press` gains **mouse capture**: subsequent `Move`/`Release`
   go straight to it even if the cursor left its bounds (the basis of Slider
   dragging and text selection)
3. `Move` produces hover: entering emits `MouseEnter`, leaving emits
   `MouseLeave`; **no Leave is emitted while a button is held**
4. Events not consumed by a child **bubble to the parent**

```go
// capture semantics: press, move far outside, release
r.Dispatch(press(0, 0))
r.Dispatch(move(30, 15))      // still dragging a
r.Dispatch(release(30, 15))   // released outside → OnClick NOT fired
```

**Keyboard**

1. Delivered to the **currently focused widget**
2. If not consumed, bubbles along the parent chain (never escapes a modal Dialog)
3. Global keys (Tab / Shift+Tab) are intercepted by the FocusManager first

### 3.3 Event type reference

| Kind | Type | Notes |
|---|---|---|
| Key | `event.Key` | `Code` + `Rune` + `Mod` + `Press`; decoder always sets `Press:true` |
| Mouse | `event.Mouse` | `Kind` selects the action; `X/Y` are 0-based cell coords |
| Resize | `event.Resize` | both buffers invalidated, then full repaint |
| Paste | `event.Paste` | one bracketed-paste payload |
| Focus | `event.Focus` | terminal focus gained/lost |

`MouseKind`: `MousePress`, `MouseRelease`, `MouseMove`, `MouseWheelUp`,
`MouseWheelDown`, `MouseEnter`, `MouseLeave`.

> `MouseEnter`/`MouseLeave` are **synthesized by the framework only**; the
> input decoder never emits them.

Common `KeyCode`s: `KeyRune` (printable chars use `Rune`), `KeyEnter`, `KeyEsc`,
`KeyTab`, `KeySpace` (=0x20), `KeyBackspace`, `KeyDelete`, `KeyUp/Down/Left/Right`,
`KeyHome/End`, `KeyPgUp/PgDn`, `KeyF1`..`KeyF12`.

`Mod`: `ModShift`, `ModCtrl`, `ModAlt`, `ModMeta`.

### 3.4 Focus management

```go
r := a.Root()

r.Dispatch(event.Key{Code: event.KeyTab, Press: true})          // next
r.Dispatch(event.Key{Code: event.KeyTab, Mod: event.ModShift})  // previous (wraps)

r.Focus(btn)            // focus programmatically
r.Focused()             // current focus, may be nil
```

Rules:

- Traversal order is tree order, collecting only
  `Focusable && effectively visible && effectively enabled`, **wrapping around**
- Click-to-focus: a focusable widget takes focus; clicking a non-focusable
  widget or empty space **does not change focus**
- Focus change → old widget `OnBlur`, new widget `OnFocus`, both marked dirty
- Membership is controlled by `SetFocusable(bool)`; `Radio` members are
  non-focusable by default (`RadioGroup` occupies one Tab slot)

### 3.5 Dirty tracking and repaint

```go
w.Invalidate()          // mark dirty + request host repaint
root.NeedPaint()        // is a repaint pending?
a.Draw()                // force one frame
a.Flush()               // run Post queue + repaint if needed
```

---

## 4. Layout

### 4.1 Three approaches

**① Explicit coordinates** (simple, fully controlled)

```go
label.Layout(geom.Rect{X: 2, Y: 2, W: 30, H: 1})
```

**② Constructor-sugar containers** (recommended, responsive)

```go
col := paintedskin.Column(
    paintedskin.Gap(0, paintedskin.Label("Title")),
    paintedskin.Gap(1, paintedskin.Button("OK", onOK)),
    paintedskin.Gap(1, paintedskin.HSlider(40)),
    paintedskin.Gap(1, paintedskin.Fixed(8, paintedskin.List(items...))),  // exactly 8 rows
    paintedskin.Grow(1, paintedskin.TextInput()),                    // takes the remainder
)
col.Layout(area)
```

- `paintedskin.Column(...)` / `paintedskin.Row(...)`: sequential main-axis placement
- `paintedskin.Gap(n, children...)`: sibling spacing (columns/rows)
- `paintedskin.Fixed(h, w)`: fixed height (fixed width inside a `Row`)
- `paintedskin.Grow(weight, w)`: share leftover space by weight
- `paintedskin.Centered(w)`: center on the cross axis

`Column` / `Row` / `Root` are **transparent by default** (they do not fill their
own background); every leaf fills its own. To tint a whole block, set a style
(**zero = transparent**):

```go
col := paintedskin.Column(widget.NewLabel("x"))
col.SetStyle(style.Style{}.WithBackground(style.ANSIColor(238)))
```

Children without their own style do not punch through that background
("zero style = transparent overlay"), so layered backgrounds work.

**③ Layout engine directly** (custom arrangement)

```go
rects := layout.FlexRow(area, []layout.FlexItem{
    {Pref: geom.Size{W: 4, H: 1}},              // ideal 4
    {Pref: geom.Size{W: 4, H: 1}, Grow: 1},     // takes the rest
}, layout.AlignStretch, 0)
```

### 4.2 Flex algorithm (`docs/07`)

1. Give every child its ideal size from `Measure`
2. Leftover on the main axis → distribute by `Grow`
3. Shortage on the main axis → shrink by `Shrink` down to each `Min`
4. Still not enough → proportional **clipping** (never crash, never overflow)
5. Place on the cross axis per `Align`

`Align`: `AlignStart`, `AlignCenter`, `AlignEnd`, `AlignStretch`.

### 4.3 Constraint clamping

```go
c := layout.Constraints{MinW: 4, MaxW: 20, MinH: 1, MaxH: 5}
c.Clamp(geom.Size{W: 100, H: 0})   // → {W:20, H:1}
```

- The result always lands in `[Min, Max]`; if `Min > Max`, **Max wins** (clip-first)
- Never negative
- `MaxW < 0` means **no upper bound**

### 4.4 Stack

```go
layout.Stack(area, geom.Size{W: 20, H: 7}, layout.StackCenter)
```

`StackTopLeft`, `StackCenter`, `StackBottomRight`, `StackFill`. All children share
the same rect — used for Panel content overlays, Dropdown popups, Dialog cover.

### 4.5 Content rect and scrolling

```go
// outer box → content area (minus padding and border)
content := layout.ContentRect(bounds, geom.SymInsets(1), 1)

// scroll offset clamped to [0, content-view]
off := layout.ClampScroll(offset, contentLen, viewLen)
```

### 4.6 Layout constraints

- Every rect handed to a child must stay inside the parent's content area
- Tiny windows: Min conflicts are clipped safely — no crash, no overflow
- `Slider` is **vertical by default** (`Measure` wants 7 rows); use `paintedskin.HSlider(v)`
  or `SetHorizontal(true)` in rows

---

## 5. Widget Reference

All 15 widgets. Every root-package sugar maps 1:1 to a `widget` constructor
(e.g. `paintedskin.Label` ↔ `widget.NewLabel`).

### 5.1 Cheat sheet

| Widget | Constructor | Focusable | Main callbacks |
|---|---|:--:|---|
| Label | `paintedskin.Label(text)` | ✗ | — |
| Button | `paintedskin.Button(text, onClick)` | ✓ | `SetOnClick(func())` |
| Checkbox | `paintedskin.Checkbox(text)` | ✓ | `SetOnChange(func(bool))` |
| RadioGroup | `paintedskin.RadioGroup(name, items)` | ✓ | `SetOnChange(func(int))` |
| TextInput | `paintedskin.TextInput()` | ✓ | `SetOnSubmit` / `SetOnChange` |
| Slider | `paintedskin.HSlider(v)` / `paintedskin.Slider()` | ✓ | `SetOnChange(func(int))` |
| ProgressBar | `paintedskin.ProgressBar()` | ✗ | `SetOnChange(func(int))` |
| List | `paintedskin.List(items...)` | ✓ | `SetOnChange` / `SetOnActivate` |
| Table | `paintedskin.Table(cols...)` + `paintedskin.TableRows` | ✓ | `SetOnChange` / `SetOnActivate` |
| Tabs | `paintedskin.Tabs(labels...)` | ✓ | `SetOnChange(func(int))` |
| Dropdown | `paintedskin.Dropdown(items...)` | ✓ | `SetOnChange(func(int))` |
| Scrollbar | `widget.NewScrollbar()` | ✗ | `SetOnChange(func(int))` |
| Panel | `paintedskin.Panel(title, child)` | ✗ | — |
| Dialog | `paintedskin.Dialog(t, m, labels, cb)` | internal buttons | `SetOnButton` / `SetOnClose` |
| StatusBar | `paintedskin.StatusBar(l, c, r)` | ✗ | — |
| Separator | `paintedskin.Separator()` | ✗ | — |

**Common properties** (every widget, inherited from `widget.Base`):

```go
w.SetVisible(bool)
w.SetEnabled(bool)      // same as SetDisabled(!v)
w.SetStyle(style.Style) // overrides the theme
w.SetFocusable(bool)
w.SetTheme(*widget.Theme)
b := w.BoundsOf()       // current bounds
w.Invalidate()
```

> A disabled widget (`SetDisabled(true)`) **swallows every event**, so callbacks
> can never fire by accident.

### 5.2 Label — text

```go
lbl := paintedskin.Label("中文 mixed text")
lbl.SetAlign(layout.AlignCenter)   // layout.AlignStart (default) / Center / End
lbl.SetWrap(true)           // wrap when wider than the constraint
lbl.SetText("new text")
lbl.Text()
```

- Measure: single line = column width × 1; wrapped = max line width × line count
- No interaction: not focusable, mouse and key events **bubble to the parent**

### 5.3 Button — five states

```go
btn := paintedskin.Button("OK", func() { submit() })
btn.SetWidth(20)            // fixed width (0 = text + padding)
btn.SetDisabled(true)
btn.Hover(); btn.Pressed(); btn.Focused(); btn.Disabled()
```

States: `normal / hover / pressed / focused / disabled`
(hover and focus can combine; precedence disabled > pressed/focused > hover).

| Input | Behavior |
|---|---|
| `MouseEnter/Leave` | toggle hover |
| `Press` (left, inside) | set pressed + request mouse capture |
| `Release` (inside) | fire `OnClick` |
| `Release` (outside) | **does not fire** (press-move-out-move-back semantics) |
| `Enter` / `Space` (focused) | fire `OnClick` |
| disabled | swallow all |

Measure: text width + left/right padding (4 cells total); fixed width optional.
Paint: `[ text ]`, **padded to the widget width** so the focus/pressed reverse
band spans the whole button instead of stopping at the text.

### 5.4 Checkbox

```go
cb := paintedskin.Checkbox("Enable feature")
cb.SetChecked(true)
cb.SetOnChange(func(on bool) { apply(on) })
cb.Checked()
```

Clicking either the box **or the label** toggles; `Space`/`Enter` when focused
toggles; v1 has no indeterminate state.

Paint: `[X] text`, padded to the widget width (the focus reverse band spans the
full width).

### 5.5 RadioGroup

```go
g := paintedskin.RadioGroup("theme", []string{"Dark", "Light"})
g.SetSelected(0)
g.SetOnChange(func(i int) { /* ... */ })
g.Radio(1).Checked()
```

- Paint: `(x) text`, padded to the widget width
- **Mutually exclusive**: selecting one deselects the rest
- `Up/Down/Left/Right` move the selection when focused
- The **group occupies one Tab slot**; member `Radio` widgets are not focusable
- `SetItems` rebuilds members and keeps the selection valid when possible

### 5.6 TextInput

```go
in := paintedskin.TextInput()
in.SetPlaceholder("type here…")
in.SetMaxLength(40)
in.SetOnSubmit(func(s string) { login(s) })    // Enter
in.SetOnConfirm(func(s string) { /* alias */ })
in.SetOnChange(func(s string) { /* every edit */ })
in.Cursor()                 // rune index
x, y, ok := in.CursorScreenPos()   // hardware cursor position (used by the host)
_, _, _ = x, y, ok
in.Selection()              // (start, end); (-1,-1) when none
in.Placeholder(); in.MaxLength()
in.SetBorder(widget.BorderRound)   // default BorderHeavy
in.SetFocusRing(true)              // focus ring (off by default)
in.Border(); in.FocusRingEnabled()
```

| Category | Keys | Behavior |
|---|---|---|
| Cursor | `Left/Right` | move one rune |
| Cursor | `Ctrl+Left/Right` | word jump |
| Cursor | `Home/End` | start / end |
| Edit | printable char | insert (respects `MaxLength`) |
| Edit | `Backspace` | delete before; deletes selection if present |
| Edit | `Delete` | delete after; deletes selection if present |
| Edit | `Ctrl+A` | select all |
| Edit | `Paste` event | bulk insert (bracketed paste) |
| Submit | `Enter` | fires `OnSubmit` |
| Mouse | click | place cursor (clicking a char inserts *before* it) |
| Mouse | drag | selection |
| View | automatic | horizontal scroll keeps the **cursor always visible** |
| Hardware cursor | when focused | framework positions the terminal cursor at the insertion point; hidden on blur |

### 5.7 Slider

```go
s := paintedskin.HSlider(40)              // horizontal; paintedskin.Slider() is vertical
s.SetRange(0, 100)
s.SetStep(5)
s.SetOnChange(func(v int) {})
s.Value()
```

| Input | Behavior |
|---|---|
| `Press` anywhere on the track | capture + **jump** to that position |
| `Move` (held) | continuous value update |
| `Release` | end drag |
| `Left/Right/Up/Down` | ±`Step` when focused |
| `Home/End` | jump to the ends |

Orientation: `SetHorizontal(bool)`; `Measure` is 30×3 horizontal, 15×7 vertical.
Paint (**both directions fill every row**, glyphs from `screen.Bar`):

- horizontal: `[===>   ]` (caps + pad + knob `>`). The knob always stays inside
  the track; at full value it hugs the right end instead of disappearing
- vertical: a per-row block bar — `█` above the knob row, `░` below it, with the
  `▐` knob centered in the knob row

### 5.8 ProgressBar

```go
pb := paintedskin.ProgressBar()
pb.SetBorder(widget.BorderRound)   // default BorderHeavy
pb.SetRange(0, 100)
pb.SetValue(40)
pb.SetShowPercent(true)
pb.SetFormat("40%")
```

**No interaction** (not focusable, consumes no keys, swallows when disabled).
Three rows draw a box plus a fill bar; a single row degrades to `[===   ]`.

### 5.9 List

```go
l := paintedskin.List("first", "second", "third")
l.SetItems(items)
l.SetSelected(2)
l.SetOnChange(func(i int) {})
l.SetOnActivate(func(i int) {})   // Enter
l.Selected(); l.Offset(); l.Len()
```

- **Virtual scrolling**: only visible rows are painted
- The first item is selected initially; `SetSelected` clamps out-of-range values
- Wheel scrolls ±1 row; `Up/Down` move the selection and **auto-scroll it into
  view**; `Home/End` jump; `PgUp/PgDn` page; `Enter` activates; click selects
- A 1-cell scrollbar indicator is drawn on the right edge (not independently interactive)
- Paint fills the whole area first (rows below the last item included); row text
  is padded to `W-1` (1 column reserved for the scrollbar), so the selected
  row's background/highlight reaches the end of the row

### 5.10 Table

```go
tb := paintedskin.TableRows(
    paintedskin.Table(
        widget.TableColumn{Title: "Name", Width: 10},
        widget.TableColumn{Title: "N", Width: 6, Align: layout.AlignEnd},
        // Width: 0 → share the remaining width automatically
    ),
    [][]string{{"alice", "1"}, {"bob", "2"}},
)
tb.SetSelectedRow(1)
tb.SetOnActivate(func(row int) {})
tb.ColumnWidths()
tb.Offset(); tb.HOffset()
```

- The **header is fixed** (bold, does not scroll); the body scrolls
- Column widths: `Width > 0` fixed, `Width == 0` splits the rest (largest remainder)
- Wheel scrolls vertically; `Up/Down` select a row and scroll it into view;
  `Enter` activates; click selects; **`Left/Right` scroll horizontally** when the
  total column width exceeds the viewport
- Row selection uses `Up/Down` only; horizontal scrolling is independent

### 5.11 Tabs

```go
tabs := paintedskin.Tabs("One", "Two", "Three")
tabs.SetContent(paintedskin.Label("page one content"))
tabs.SetCurrent(1)
tabs.SetOnChange(func(i int) {})
tabs.Labels(); tabs.Current(); tabs.Content()
```

- Click a tab to switch; `Left/Right` switch pages when focused
- **The tab strip is one group**: it occupies a single Tab slot, and `Tab` moves
  into the content widget (which joins traversal)
- `Measure` height = 1 (strip) + content height; `Layout` gives the content
  everything below the first row

### 5.12 Dropdown

```go
d := paintedskin.Dropdown("Go", "Rust", "Zig")
d.SetSelected(1)
d.SetOnChange(func(i int) {})
d.IsOpen(); d.Open(); d.Close(); d.Pick(i)
```

- Click to expand → `Host.SetOverlay` popup; **below by default, flipping above
  when there is not enough room** (at most 6 items shown)
- Click an option → select and close; `Esc` closes; **clicking outside closes**
  (handled by `Root`)
- While open, `Up/Down` pre-select and `Enter` confirms and closes
- Paint: `<current> ▼`, **padded to the width `Measure` reports** so an empty
  selection never leaves a gap

### 5.13 Scrollbar

```go
sb := widget.NewScrollbar()
sb.SetHorizontal(false)   // default vertical (right edge, 1 cell wide)
sb.SetTotal(100)
sb.SetView(20)
sb.SetOffset(40)          // set directly (does not fire OnChange)
sb.SetOnChange(func(off int) {})
```

- The offset is clamped to `[0, Total-View]`; it is 0 when content < viewport
- Clicking the track pages by ±`View`; **dragging the thumb** captures and
  recomputes proportionally
- **The wheel is not consumed** — it is forwarded to the host so List/Table handle it
- Setters never fire `OnChange`; only interaction does
- Paint: track `screen.Bar.Track` + thumb `screen.Bar.Filled`; styles come from the
  theme's `ScrollbarTrack` / `ScrollbarThumb` (the latter falls back to
  `ScrollbarThumbColor` when unset)

### 5.14 Panel

```go
p := paintedskin.Panel(" Title ", paintedskin.Column(
    paintedskin.Gap(1, paintedskin.Label("content")),
))
p.SetBorder(widget.BorderDouble)     // None/Single(default)/Double/Round/Heavy
p.SetBorderSpec(screen.BorderDashed) // any glyph set (9 built-in)
p.SetBorderEdges(screen.BorderEdges{Top: true})  // per-edge switches (no relayout)
p.SetHatch(screen.HatchCross)       // interior texture (~20% crosshatch)
p.BorderSpec(); p.Hatch()           // effective border and texture
p.SetPadding(geom.SymInsets(1))      // default 1
p.SetTitle("New title")
p.Child()
// Note: containers like Panel/Dialog have no SetDisabled (use SetVisible)
```

- **Default padding 1**: content never touches the border
- The whole area (interior + padding ring) is filled with the widget's own
  background first, then the border is drawn on top
- The title is drawn on the **first row inside the border**, occupies 1 row, and
  is included in `Measure` (it never overwrites the border); it is truncated to
  `width - 4` (one blank column on each side), so a long title can never eat
  the right border
- `Measure` = max(content, title) + padding×2 + border×2
- Border glyphs come from `screen.Border` (3×3 glyph table): `BorderDouble`
  `╔═╗`, `BorderHeavy` `┏━┓`, `BorderRound`, `BorderDashed`, `BorderThick`
  (half-block). `SetBorderSpec` takes any glyph set, `SetBorderEdges` switches
  edges individually (a disabled edge is not drawn but still reserves its spacing)
- The container consumes nothing (unconsumed child events bubble up)
- `HitTest` probes the child first, then falls back to itself

### 5.15 Dialog

```go
dlg := paintedskin.Dialog("Confirm", "Continue?", []string{"OK", "Cancel"}, func(i int) {
    // i == 0 OK, 1 Cancel
    root.PopDialog()
})
dlg.SetOnClose(func() { root.PopDialog() })   // Esc path
dlg.SetDim(true)                               // mask
dlg.SetSize(40, 10)                            // 0 = auto
dlg.Layout(root.Bounds)                        // centered in the given area
dlg.SetBorder(widget.BorderRound)              // enum preset
dlg.SetBorderSpec(screen.BorderThick)          // any glyph set
dlg.SetBorderEdges(screen.BorderEdges{Top: true})  // per-edge switches
dlg.Border(); dlg.BorderSpec(); dlg.BorderEdges()
```

Behavior:

- `Root.PushDialog` makes it **modal** (widgets below receive no mouse or keys)
- **Focus trap**: `Tab` cycles only inside the Dialog; focus starts on the first button
- `Esc` → `Close()` → fires `OnClose` → `PopDialog`, **restoring the previous focus**
- Hit order: overlay > topmost Dialog > ordinary children
- Paint: **always fills its own area** (`SetDim` only chooses what to fill, not
  whether to fill), then draws the border; the title follows the same rule as
  `Panel` (first row inside the border, truncated to `width - 4`), and long
  message lines are truncated too so they never eat the right border

> `Dialog.Close()` already calls `PopDialog`; do **not** call it again in `OnClose`.

### 5.16 StatusBar

```go
bar := paintedskin.StatusBar("app", "message", "v1.0")
bar.SetLeft("x"); bar.SetCenter("y"); bar.SetRight("z")
bar.SetLeftStyle(style.Style{}.WithBold(true))   // per-segment (zero falls back)
bar.Left(); bar.Center(); bar.Right()
```

- Height is always 1; left segment left-aligned, center centered, right
  right-aligned (measured in **columns**, so CJK is correct)
- Each segment is truncated to non-overlapping spans (reservation priority
  **right > center > left**): no gaps between segments, and segments can never
  overwrite each other
- The whole row is filled with the widget's own background; empty segments are
  not drawn; typically pinned to the bottom row

### 5.17 Separator

```go
sep := paintedskin.Separator()      // horizontal by default
sep.SetHorizontal(false)    // vertical
sep.SetLine('=')            // custom glyph
```

A horizontal separator is always 1 row tall, a vertical one always 1 cell wide
(the fixed dimension never degrades under constraints).

---

## 6. Overlays: Dialog and Overlay

Two mechanisms cover two classes of popup (`docs/05` overlay model):

| Type | Mechanism | Hit order | Dismissed by |
|---|---|---|---|
| Dialog | `Root.PushDialog/PopDialog` modal stack | topmost first | `Esc` / button callback `PopDialog` |
| Dropdown / popup | `Host.SetOverlay(w)` single overlay | overlay first | outside click / `Esc` / selection |

```go
// modal
root.PushDialog(dlg)
root.PopDialog()

// overlay
root.SetOverlay(w)     // non-nil shows it; hit-tested before ordinary children
root.SetOverlay(nil)   // dismiss
root.Overlay()         // query
```

`Root` implements `Host`, and widgets reach it via `HostOf()`, so **widgets never
depend on the `app` package**:

```go
type Host interface {
    Invalidate()             // mark dirty, request repaint
    Focus(w Widget)          // focus programmatically
    PushDialog(w Widget)     // push modal
    PopDialog()              // pop modal + restore focus
    SetOverlay(w Widget)     // single overlay
    Capture(w Widget)        // request mouse capture
}
```

---

## 7. Theming

```go
a.ApplyTheme(theme.Light())   // swap theme and repaint now
a.SetTheme(th)                // swap only; takes effect next frame

th := theme.Dark()
th.FocusColor = style.Blue
th.ButtonHover = style.Style{}.WithBold(true)
theme.SetDefault(th)
```

Precedence (`docs/08`): **widget's own `Style` > theme > built-in default**.

```go
btn.SetStyle(myStyle)                         // highest
btn.SetTheme(widget.ThemeFrom(theme.Dark()))  // then
// otherwise the built-in default (keeps the five states distinguishable)
```

`Theme` fields: base colors `FocusColor`, `BorderColor`, `DisabledColor`,
`HoverBG`, `SelectionBG`, `ScrollbarThumbColor`; widget styles
`ButtonNormal/Hover/Pressed/Focused/Disabled`, `ListSelected`, `TableHeader`,
`DialogMask`, `ScrollbarThumb/Track`, `Separator`, `StatusBar`,
`TextInputNormal/Focus`.

**Every field is read by the widgets** (`TestEveryThemeFieldIsWired` verifies them
one by one).

Built-in themes: `theme.Dark()` (default), `theme.Light()`, `theme.Default()`.
With the default themes, focus uses the `BrightCyan` foreground and selection a
background color (reverse in Dark, `ANSIColor(153)` in Light) — zero out
`FocusColor`/`SelectionBG` to get the reverse-band look back (they fall back to
the built-in `AttrReverse`).

---

## 8. Text Width and CJK/ASCII Mixed Content

For Chinese-speaking users, **column measurement has a single authority**: the
`text` package (ADR-017).

```go
text.Width("中文abc")        // 7  (中=2 文=2 a=1 b=1 c=1)
text.RuneWidth('中')          // 2
text.Truncate("中文abc", 4)   // "中文" (never splits a wide char)
text.Fit("ab", 6, text.Center)
text.PadEnd("中文", 6)
text.OffsetAt(s, charIdx)    // char index → screen column
text.IndexAtColumn(s, col)   // screen column → char index
```

Rules:

- CJK / full-width = **2 columns**, ASCII = **1 column**, combining marks /
  zero-width joiners = **0 columns** (merged into the preceding cell's `Tail`)
- A combining mark (`é` = `e` + U+0301) and a ZWJ (U+200D) are emitted as one
  **contiguous run** with their base character — that is what makes the terminal
  compose a single glyph instead of a stray tofu box
- Truncation only happens at character boundaries — a wide char is **never split**
- In the buffer a wide char occupies two cells: left holds the char (`Width=2`),
  right holds a placeholder (`Width=-1`)
- A wide char that would land on the last column with no room for its second
  cell **degrades to a space** rather than overflowing (overflowing makes the
  terminal wrap and breaks the frame)

**Invariants covered by tests**:

1. After any write, the row's occupied column count equals the buffer width
2. With mixed CJK/ASCII content (including full-width, half-width kana, emoji,
   and combining chars), the rightmost visible character of every row is still
   the right border
3. Truncation never splits a character
4. Panel children and titles always stay inside the border

> **Known limitation**: the joiners of a ZWJ emoji sequence (👨‍👩‍👧) are now
> forwarded to the terminal intact, so a ZWJ-aware terminal *does* compose one
> glyph — but the **column budget** is still 3 wide chars = 6 columns while such a
> terminal renders 2, so the layout is off by 4 columns. A full fix requires
> grapheme-cluster segmentation, which is out of scope for v1.

---

## 9. Style and Color

```go
st := style.Style{}.
    WithForeground(style.Cyan).
    WithBackground(style.ANSIColor(238)).
    WithBold(true)

st.WithDim(true).WithItalic(true).WithUnderline(true).
   WithBlink(true).WithReverse(true).WithStrike(true)
st.WithAttrs(style.AttrBold | style.AttrUnderline)   // replace wholesale
st.AttrMask()      // bitmask
st.Equal(other)    // visual equality (ignores the Attrs mirror)
st.Swap()          // swap foreground/background
```

Colors:

```go
style.RGB(30, 144, 255)      // truecolor
style.Hex(0x1E90FF)           // #RRGGBB
style.ANSIColor(6)            // 16-color (Cyan)
style.Red / BrightRed / ...  // named constants
```

Emission rules (degraded automatically to match terminal capabilities):

| Depth | Foreground | Background |
|---|---|---|
| `ColorANSI(0..7)` | `30–37` | `40–47` |
| `ColorANSI(8..15)` | `90–97` | `100–107` |
| `Color256` | `38;5;n` | `48;5;n` |
| `ColorRGB` | `38;2;r;g;b` | `48;2;r;g;b` |
| `ColorDefault` | no parameters | no parameters |

Attribute bits: `AttrBold`, `AttrDim`, `AttrItalic`, `AttrUnderline`,
`AttrBlink`, `AttrReverse`, `AttrStrike`.

Border and bar glyphs (the **single source of truth** is `screen/border.go`):

```go
screen.BorderSingle / Double / Round / Heavy / Ascii / Dashed / Thick
screen.BorderNone                       // zero value = no border
screen.BorderByName("round")            // lookup by style name
b := screen.BorderSingle.WithEdges(screen.BorderEdges{Top: true})
b.At(0, 0)        // 3×3 glyph matrix: [row top/mid/bottom][col left/hline/right]
b.HLine(); b.VLine()                    // horizontal / vertical line glyph
screen.Bar.Track / Filled / KnobH / KnobV / Pad / CapL / CapR
screen.Bar.Line(screen.Bar.Track, 3)    // repeat n times
```

9 built-in sets: `Single` `Double` `Round` `Heavy` `Ascii` `Dashed` `Thick`
(half-block `█▀▄`) `Inner` (inset half-block) `Outer` (outset half-block).
`BorderDouble` draws `╔═╗` (double), `BorderHeavy` draws `┏━┓` (heavy).

Textures (~20% density, only on blanks, never overwrites content):

```go
screen.HatchNone / HatchLeft / HatchRight / HatchCross
c.Hatch(rect, screen.HatchCross, st)    // Canvas primitive
```

Geometry:

```go
r := geom.Rect{X: 0, Y: 0, W: 20, H: 5}
r.Contains(geom.Point{X: 3, Y: 1})   // right/bottom exclusive
r.Right(); r.Bottom(); r.Size(); r.Empty()
r.Intersect(o); r.Union(o)
r.Inflate(geom.SymInsets(1))          // shrink (padding)
r.Size().Area()
```

---

## 10. Threading Model

The widget tree is **mutable only on the UI goroutine**. `Handle`, `Measure`,
`Layout`, and `Paint` all run serially inside the main loop — no locking needed.

External goroutines must go through `Post`:

```go
go func() {
    // correct: hop onto the UI thread
    a.Post(func() { label.SetText("done") })
}()

// on the main loop
a.Flush()      // run the Post queue + repaint if needed
```

`Dispatch` calls `Flush` automatically. **Never** call widget mutators directly
from a goroutine.

Terminal resize is also handled on the main loop only (guaranteed by
`app.Run`), avoiding races with painting.

---

## 11. Terminal Capabilities and Degradation

Modes enabled on entry (`docs/03`):

| Sequence | Meaning |
|---|---|
| `CSI ?1049h` | alternate screen (original content restored on exit) |
| `CSI ?25l` / `?25h` | hide / show hardware cursor |
| `CSI ?1002h` `?1003h` `?1006h` | button tracking + any-motion + SGR extension |
| `CSI ?2004h` | bracketed paste |
| `CSI ?1004h` | focus events |
| raw mode | `ECHO/ICANON/ISIG/IEXTEN/IXON` cleared, `VMIN=1 VTIME=0` |

On exit everything is undone in reverse order, and **`?1049l` must be emitted
last** (the other sequences would be lost inside the alternate screen).

```go
app.Config{
    Width: 0, Height: 0,     // <=0 → use the real terminal size
    Writer: os.Stdout,       // ANSI output sink
    Mouse: true,             // enable mouse reporting
    Paste: true,             // enable bracketed paste
    FocusReports: true,      // enable focus events
    Setup: func(a *app.App) { /* build the widget tree */ },
}
```

Degradation:

- `TERM=dumb` or unset → **mouse is disabled automatically**; the app stays
  fully usable from the keyboard
- No `COLORTERM=truecolor` → 256 colors → 16 colors
- Windows: VT console mode (`ENABLE_VIRTUAL_TERMINAL_INPUT/OUTPUT`), so input is
  a standard VT byte stream fed to the same decoder; there is no SIGWINCH, so
  size changes are detected by a 300 ms poll

Terminal restoration (AGENT.md §2.4, hard invariants):

1. `Terminal.Close()` is **idempotent** (`sync.Once`)
2. `defer` / signal / `panic` recovery **share one restore path**
3. On exit the following are restored: raw mode, alternate screen, cursor, mouse
   mode, bracketed paste, focus events

---

## 12. Headless / No TTY

In CI or a sandbox `openpty` is denied and `app.Run` returns
`app.ErrNoTerminal`. Two options:

**① Degrade gracefully with `examples.Run`**

```go
func main() {
	examples.Run(func(a *app.App) { /* ... */ })
}
```

**② Headless rendering: print the frame**

```go
a := app.New(app.Config{Width: 64, Height: 24, Writer: io.Discard})
root := a.Root()
panel := paintedskin.Panel("demo", paintedskin.Column(paintedskin.Gap(1, paintedskin.Label("hello"))))
root.Add(panel)

a.Layout()
panel.Layout(geom.Rect{X: 0, Y: 0, W: 64, H: 23})
a.Draw()

for y := 0; y < a.Buffer().Height(); y++ {
	for x := 0; x < a.Buffer().Width(); x++ {
		c := a.Buffer().At(x, y)
		if c == nil || c.Rune == 0 || c.IsContinuation() {
			continue // empty / wide-char continuation (keeps the column)
		}
		fmt.Print(string(c.Rune))
	}
	fmt.Println()
}
```

`app.New` has **no output side effects** — it writes nothing before a terminal is
claimed.

---

## 13. Custom Widgets

Embed `widget.Base` to inherit every default and write only the difference:

```go
type LED struct {
	widget.Base
	on bool
}

func NewLED() *LED {
	l := &LED{}
	l.Base = widget.NewBase()
	l.Base.Bind(l)      // required: makes HitTest return the outer widget
	return l
}

func (l *LED) Measure(c layout.Constraints) geom.Size {
	return c.Clamp(geom.Size{W: 3, H: 1})
}

func (l *LED) Handle(e event.Event) bool {
	if !l.EffectivelyEnabled() {
		return true            // disabled: swallow
	}
	if m, ok := e.(event.Mouse); ok && m.Kind == event.MousePress {
		l.on = !l.on
		l.Invalidate()
		return true
	}
	return false               // not consumed → bubbles to the parent
}

func (l *LED) Paint(c screen.Canvas) {
	if l.Bounds.Empty() {
		return
	}
	s := style.Style{}
	if l.on {
		s = s.WithReverse(true)
	}
	c.Text(l.Bounds.X, l.Bounds.Y, "[o]", s)
}
```

Key points:

- **`Bind` must be called**: Go embedding cannot make `Base.HitTest` return the
  outer widget; skipping it breaks focus, capture, and bubbling
- After embedding you inherit: `Handle` / `HitTest` / `Measure` / `Layout` /
  `Paint` / `Children` / `SetHost` and friends
- Override `Children()` when you own children (and make the container recurse in
  `SetHost`)
- To join Tab traversal: set `l.Focusable = true` in the constructor (or
  `SetFocusable(true)`)

A container that implements `Children()` automatically participates in focus
traversal, hit testing, theme injection, and recursive `SetHost`.

---

## 14. Testing and Debugging

```bash
./go.sh vet ./...      # static checks
./go.sh test ./...     # full test suite
GOOS=windows ./go.sh build ./...   # cross compile
```

> The global Go module cache is not writable in the sandbox, so always use the
> `./go.sh` wrapper.

### Asserting with synthetic events

```go
r := widget.NewRoot()
r.Layout(geom.Rect{X: 0, Y: 0, W: 40, H: 20})
btn := widget.NewButton("OK")
clicks := 0
btn.SetOnClick(func() { clicks++ })
r.Add(btn)
btn.Layout(geom.Rect{X: 0, Y: 0, W: 6, H: 1})

r.Dispatch(event.Mouse{X: 1, Y: 0, Kind: event.MousePress, Button: event.ButtonLeft})
r.Dispatch(event.Mouse{X: 1, Y: 0, Kind: event.MouseRelease, Button: event.ButtonLeft})
if clicks != 1 { t.Fatalf("clicks = %d", clicks) }

r.Dispatch(event.Key{Code: event.KeyTab, Press: true})
if r.Focused() != btn { t.Fatalf("focus = %v", r.Focused()) }
```

### Asserting the rendered frame

```go
a := app.New(app.Config{Width: 40, Height: 10, Writer: io.Discard})
// ... attach widgets ...
a.Layout()
a.Draw()
// assert against a.Buffer()
```

`a.Buffer().At(x, y)` returns `*screen.Cell` (`Rune` / `Width` / `Style`).

### Doc example verification

`docs/examples/compile_test.go` compiles and runs every snippet in these docs.
Update it alongside the docs — `go test ./docs/examples` blocks
"documented but does not compile" code.

---

## 15. Complete Examples

### 15.1 Login form

```go
func build(a *app.App) {
	root := a.Root()
	sz := a.Size()

	username := paintedskin.TextInput()
	username.SetPlaceholder("username")
	password := paintedskin.TextInput()
	password.SetPlaceholder("password")

	msg := paintedskin.Label("")
	submit := paintedskin.Button("Login", func() {
		if username.Value() == "" {
			msg.SetText("Please enter a username")
			return
		}
		dlg := paintedskin.Dialog("Welcome", "Hello, "+username.Value(), []string{"OK"}, func(int) {
			root.PopDialog()
		})
		root.PushDialog(dlg)
		dlg.Layout(root.Bounds)
	})

	form := paintedskin.Panel(" Login ", paintedskin.Column(
		paintedskin.Gap(1, paintedskin.Label("Username")),
		paintedskin.Gap(1, username),
		paintedskin.Gap(1, paintedskin.Label("Password")),
		paintedskin.Gap(1, password),
		paintedskin.Gap(2, submit),
		paintedskin.Gap(1, msg),
	))

	bar := paintedskin.StatusBar("demo", "", "Tab cycles focus")
	root.Add(form)
	root.Add(bar)

	y := (sz.H - 8) / 2
	form.Layout(geom.Rect{X: (sz.W - 34) / 2, Y: y, W: 34, H: 8})
	bar.Layout(geom.Rect{X: 0, Y: sz.H - 1, W: sz.W, H: 1})
}
```

### 15.2 List with a detail pane

```go
func build(a *app.App) {
	root := a.Root()
	sz := a.Size()

	items := []string{"alpha", "beta", "gamma", "delta", "epsilon"}
	list := paintedskin.List(items...)
	detail := paintedskin.Label("selected: alpha")

	list.SetOnChange(func(i int) {
		if i >= 0 && i < len(items) {
			detail.SetText("selected: " + items[i])
		}
	})
	list.SetOnActivate(func(i int) {
		dlg := paintedskin.Dialog("Confirm", "Open "+items[i]+"?", []string{"Open", "Cancel"}, func(int) {
			root.PopDialog()
		})
		root.PushDialog(dlg)
		dlg.Layout(root.Bounds)
	})

	root.Add(list)
	root.Add(detail)
	list.Layout(geom.Rect{X: 2, Y: 1, W: 30, H: 10})
	detail.Layout(geom.Rect{X: 2, Y: 12, W: 40, H: 1})
}
```

### 15.3 Background task (thread-safe)

```go
func build(a *app.App) {
	bar := paintedskin.ProgressBar()
	bar.SetRange(0, 100)
	bar.SetShowPercent(true)
	a.Root().Add(bar)
	bar.Layout(geom.Rect{X: 2, Y: 2, W: 40, H: 3})

	// simulated download: progress computed off-thread, widget updated on UI thread
	go func() {
		for i := 0; i <= 100; i += 5 {
			v := i
			a.Post(func() { bar.SetValue(v) })
			time.Sleep(80 * time.Millisecond)
		}
	}()
}
```

### 15.4 Responsive two-column layout

```go
func build(a *app.App) {
	root := a.Root()
	sz := a.Size()

	left := paintedskin.Panel(" Left ", paintedskin.Column(
		paintedskin.Gap(1, paintedskin.Label("content A")),
		paintedskin.Gap(1, paintedskin.Checkbox("option A")),
	))
	right := paintedskin.Panel(" Right ", paintedskin.Column(
		paintedskin.Gap(1, paintedskin.Label("content B")),
		paintedskin.Gap(1, paintedskin.HSlider(30)),
	))

	root.Add(left)
	root.Add(right)

	// FlexRow splits the width and adapts to the window
	body := geom.Rect{X: 0, Y: 0, W: sz.W, H: sz.H}
	rects := layout.FlexRow(body, []layout.FlexItem{
		{Pref: geom.Size{W: sz.W / 2, H: sz.H}},
		{Pref: geom.Size{W: sz.W - sz.W/2, H: sz.H}},
	}, layout.AlignStretch, 1)
	left.Layout(rects[0])
	right.Layout(rects[1])
}
```

### 15.5 Reacting to window resize

```go
func build(a *app.App) {
	panel := paintedskin.Panel("resizable", paintedskin.Label("drag the window edge"))
	root := a.Root()
	root.Add(panel)

	// idempotent layout function; call it on Setup and after size changes
	relayout := func() {
		sz := a.Size()
		panel.Layout(geom.Rect{X: 2, Y: 2, W: sz.W - 4, H: sz.H - 4})
	}
	relayout()
}
```

> `app.Run` rebuilds the buffers and repaints on `event.Resize`. If widgets need
> re-arranging for the new size, keep the layout logic in an idempotent function
> and call it from `Setup` and whenever the size changes.

---

## Appendix: Related Documents

| Document | Contents |
|---|---|
| [01-overview](docs/01-overview.md) | Goals and acceptance criteria |
| [02-architecture](docs/02-architecture.md) | Layering, data flow, threading |
| [03-terminal](docs/03-terminal.md) | Terminal backend, raw mode, capability probing |
| [04-input](docs/04-input.md) | Input decoding state machine |
| [05-render](docs/05-render.md) | Double buffering, diff, wide chars, color degradation |
| [06-widget-model](docs/06-widget-model.md) | Widget model, event routing, focus |
| [07-layout](docs/07-layout.md) | Constraint layout algorithm |
| [08-widgets](docs/08-widgets.md) | Behavior spec for all 15 widgets |
| [09-roadmap](docs/09-roadmap.md) | Milestones and acceptance checklist |
| [10-decisions](docs/10-decisions.md) | ADR decision records |