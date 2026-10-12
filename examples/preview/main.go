// Command preview 无头渲染示例：真实构建控件树 → 布局 → 绘制，
// 再把画面打印出来。用于在无 PTY 的环境（CI / 沙箱）里验证渲染结果。
//
//	go run ./examples/preview
//
// 交互式体验请运行 examples/minimal 或 examples/gallery（需真实终端）。
package main

import (
	"fmt"
	"io"
	"strings"

	"PaintedSkin"
	"PaintedSkin/app"
	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/screen"
	"PaintedSkin/widget"
)

const (
	width  = 64
	height = 26
)

func main() {
	a := app.New(app.Config{Width: width, Height: height, Writer: io.Discard})
	root := a.Root()

	clicks := 0
	hint := paintedskin.Label("Tab 遍历 · 点击 Button")
	body := paintedskin.Column(
		paintedskin.Gap(0, hint),
		paintedskin.Gap(1, paintedskin.Button("Click me", func() { clicks++; hint.SetText(fmt.Sprintf("clicked %d", clicks)) })),
		paintedskin.Gap(1, paintedskin.Checkbox("checked")),
		paintedskin.Gap(1, paintedskin.HSlider(40)),
		paintedskin.Gap(1, paintedskin.ProgressBar()),
		paintedskin.Gap(1, paintedskin.TextInput()),
		paintedskin.Gap(1, paintedskin.Dropdown("Go")),
		paintedskin.Gap(1, paintedskin.List("row 00", "row 01")),
	)
	body.SetTheme(widget.CurrentTheme())

	panel := paintedskin.Panel(" preview ", body)
	root.Add(panel)

	bar := paintedskin.StatusBar("paintedskin", "无头渲染", fmt.Sprintf("%dx%d", width, height))
	root.Add(bar)

	a.Layout()
	panel.Layout(geom.Rect{X: 0, Y: 0, W: width, H: height - 1})
	bar.Layout(geom.Rect{X: 0, Y: height - 1, W: width, H: 1})
	a.Draw()

	fmt.Print(frame(a.Buffer()))

	fmt.Println("─── 合成事件 ───")
	root.Dispatch(event.Key{Code: event.KeyTab, Press: true})
	fmt.Printf("Tab 后焦点      : %s\n", describe(root.Focused()))
	root.Dispatch(event.Key{Code: event.KeyEnter, Press: true})
	fmt.Printf("Enter 后按钮计数: %d\n", clicks)

	fmt.Printf("命中 (6,3)      : %s\n", describe(root.HitTest(geom.Point{X: 6, Y: 3})))
	fmt.Printf("命中 (60,18)    : %s\n", describe(root.HitTest(geom.Point{X: 60, Y: 18})))
}

// describe 给出控件类型名，便于观察路由结果。
func describe(w widget.Widget) string {
	if w == nil {
		return "<nil>"
	}
	t := fmt.Sprintf("%T", w)
	if i := strings.LastIndex(t, "."); i >= 0 {
		return t[i+1:]
	}
	return t
}

// frame 把缓冲渲染成带边框的文本行。
func frame(b *screen.Buffer) string {
	var sb strings.Builder
	sb.WriteString("┌" + strings.Repeat("─", width) + "┐\n")
	for y := 0; y < b.Height(); y++ {
		sb.WriteString("│")
		for x := 0; x < b.Width(); x++ {
			c := b.At(x, y)
			switch {
			case c == nil:
				sb.WriteRune(' ')
			case c.IsContinuation():
				// 宽字符右格：真实终端里被上一个宽字符吃掉，
				// 这里必须留空以保持列对齐。
				_ = x
			case c.Rune == 0:
				sb.WriteRune(' ')
			default:
				sb.WriteRune(c.Rune)
			}
		}
		sb.WriteString("│\n")
	}
	sb.WriteString("└" + strings.Repeat("─", width) + "┘\n")
	return sb.String()
}
