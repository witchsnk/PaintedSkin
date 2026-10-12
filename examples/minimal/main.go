// Command minimal 是最小可运行示例（docs/01 目标：30 行内跑起来）。
//
//	go run ./examples/minimal
//
// 鼠标点击按钮或 Tab 聚焦后按 Enter/Space 触发；Esc 退出。
package main

import (
	"fmt"

	"PaintedSkin"
	"PaintedSkin/app"
	"PaintedSkin/examples"
	"PaintedSkin/geom"
)

func main() {
	label := paintedskin.Label("按 Tab 聚焦按钮，或直接点击")
	clicks := 0
	button := paintedskin.Button("点我", func() {
		clicks++
		label.SetText(fmt.Sprintf("点了 %d 次", clicks))
	})

	examples.Run(func(a *app.App) {
		sz := a.Size()
		root := a.Root()
		root.Add(label)
		root.Add(button)
		// 留出内边距，避免文字贴边
		label.Layout(geom.Rect{X: 3, Y: sz.H / 2, W: sz.W - 6, H: 1})
		button.Layout(geom.Rect{X: 3, Y: sz.H/2 + 2, W: 12, H: 1})
	})
}
