// Command gallery 是控件总览示例，覆盖 docs/09 的手动验收清单。
//
//	go run ./examples/gallery
//
// 鼠标：悬停高亮、点击、拖拽、滚轮、拖选。键盘：Tab / Shift+Tab 遍历。
// Esc 退出。
package main

import (
	"strconv"

	"PaintedSkin"
	"PaintedSkin/app"
	"PaintedSkin/examples"
	"PaintedSkin/geom"
	"PaintedSkin/widget"
)

func main() { examples.Run(build) }

func build(a *app.App) {
	root := a.Root()
	sz := a.Size()

	// 底部状态栏贴边（1 行）
	bar := paintedskin.StatusBar("paintedskin gallery", "Esc 退出 · Tab 遍历",
		strconv.Itoa(sz.W)+"x"+strconv.Itoa(sz.H))
	root.Add(bar)
	bar.Layout(geom.Rect{X: 0, Y: sz.H - 1, W: sz.W, H: 1})

	// 主体：左右两个 Panel，中间留 1 格
	body := geom.Rect{X: 0, Y: 0, W: sz.W, H: sz.H - 1}
	colW := (sz.W - 1) / 2
	left, right := buildLeft(root), buildRight(root)
	root.Add(left)
	root.Add(right)
	left.Layout(geom.Rect{X: 0, Y: 0, W: colW, H: body.H})
	right.Layout(geom.Rect{X: sz.W - colW, Y: 0, W: colW, H: body.H})
}

func buildLeft(root widget.Widget) widget.Widget {
	hint := paintedskin.Label("Tab 遍历 · 悬停 / 点击 / 拖拽")
	clicks := 0

	slider := paintedskin.HSlider(40)
	progress := paintedskin.ProgressBar()
	progress.SetRange(0, 100)
	progress.SetShowPercent(true)
	progress.SetValue(40)
	slider.SetOnChange(func(v int) { progress.SetValue(v) })

	input := paintedskin.TextInput()
	input.SetPlaceholder("在这里输入…")

	return paintedskin.Panel(" 交互控件 ", paintedskin.Column(
		paintedskin.Gap(1, hint),
		paintedskin.Gap(1, paintedskin.Button("Button", func() {
			clicks++
			hint.SetText("Button 被点击 " + strconv.Itoa(clicks) + " 次")
		})),
		paintedskin.Gap(1, paintedskin.DisabledButton("Disabled")),
		paintedskin.Gap(1, paintedskin.Checkbox("Checkbox")),
		paintedskin.Gap(1, paintedskin.RadioGroup("theme", []string{"Dark", "Light"})),
		paintedskin.Gap(1, slider),
		paintedskin.Gap(1, progress),
		paintedskin.Gap(1, input),
		paintedskin.Gap(1, paintedskin.Dropdown("Go", "Rust", "Zig")),
	))
}

func buildRight(root widget.Widget) widget.Widget {
	items := make([]string, 0, 30)
	for i := 0; i < 30; i++ {
		items = append(items, "List item "+strconv.Itoa(i))
	}
	rows := make([][]string, 0, 20)
	for i := 0; i < 20; i++ {
		rows = append(rows, []string{"row" + strconv.Itoa(i), strconv.Itoa(i * 3)})
	}
	table := paintedskin.TableRows(
		paintedskin.Table(
			widget.TableColumn{Title: "Name", Width: 10},
			widget.TableColumn{Title: "N", Width: 6},
		), rows)

	tabs := paintedskin.Tabs("One", "Two")
	tabs.SetContent(paintedskin.Label("Tabs 内容区"))

	openBtn := paintedskin.Button("打开 Dialog", func() {
		dlg := widget.NewDialog("确认", "要继续吗？", []string{"确定", "取消"}, func(int) {
			pop(root)
		})
		root.(widget.Host).PushDialog(dlg)
		dlg.Layout(root.(*widget.Root).Bounds)
	})

	return paintedskin.Panel(" 数据控件 ", paintedskin.Column(
		paintedskin.Gap(1, paintedskin.Fixed(8, paintedskin.List(items...))),
		paintedskin.Gap(1, tabs),
		paintedskin.Gap(1, paintedskin.Fixed(8, table)),
		paintedskin.Gap(1, paintedskin.Separator()),
		paintedskin.Gap(1, openBtn),
	))
}

// pop 弹出栈顶 Dialog（焦点由 Root 恢复）。
func pop(root widget.Widget) { root.(widget.Host).PopDialog() }
