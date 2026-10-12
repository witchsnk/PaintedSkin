# PaintedSkin

从零实现的 Go TUI 底层框架：在终端里提供与 GUI 同级的控件界面（鼠标全支持、焦点管理、模态对话框、约束布局），同时保持命令行的简洁。

## 核心优势

| 优势 | 说明 |
|---|---|
| **零依赖** | 仅 `golang.org/x/sys` + `github.com/mattn/go-runewidth`，其余全部自研 |
| **鼠标全支持** | 悬停、点击、拖拽、滚轮、修饰键；SGR(1006) 协议 |
| **15 个控件** | Label / Button / Checkbox / Radio / TextInput / Slider / ProgressBar / List / Table / Tabs / Dropdown / Scrollbar / Panel / Dialog / StatusBar / Separator |
| **GUI 级交互** | z 序命中、鼠标捕获、事件冒泡、hover 合成、焦点陷阱、模态对话框 |
| **约束布局** | Measure → Layout → Paint 三阶段；Flex（Grow/Shrink/对齐）、Stack、裁剪与滚动钳制 |
| **双缓冲渲染** | 逐格 diff → 最小 ANSI 序列；宽字符步进、色彩按能力降级 |
| **跨平台** | macOS / Linux / Windows（VT console 模式），交叉编译验证 |
| **中文优化** | CJK 全角字符按 2 列宽计算，截断不劈字，中英混排正确 |
| **测试驱动** | 全部行为有单测覆盖，golden 字节序列回归，文档示例可执行校验 |

## 快速上手

```bash
go run ./examples/minimal     # 最小示例
go run ./examples/gallery     # 控件总览
go run ./examples/preview     # 无头渲染（无需 TTY）
```

### 30 行跑起来

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
			label := paintedskin.Label("按 Tab 聚焦按钮")
			btn := paintedskin.Button("点我", func() { label.SetText("clicked") })

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

### 构造器糖容器（推荐）

```go
col := paintedskin.Column(
	paintedskin.Gap(0, paintedskin.Label("标题")),
	paintedskin.Gap(1, paintedskin.Button("确定", onOK)),
	paintedskin.Gap(1, paintedskin.HSlider(40)),
	paintedskin.Gap(1, paintedskin.Fixed(8, paintedskin.List(items...))),
	paintedskin.Grow(1, paintedskin.TextInput()),
)
col.Layout(area)
```

### 线程安全更新

```go
go func() {
	for i := 0; i <= 100; i += 5 {
		v := i
		a.Post(func() { bar.SetValue(v) }) // 投递到 UI 线程
		time.Sleep(80 * time.Millisecond)
	}
}()
```

## 包结构

```
PaintedSkin/
├── geom/    Point/Size/Rect/Insets
├── style/   Color（default/16/256/rgb）、Attrs、Style
├── event/   Key/Mouse/Resize/Paste/Focus
├── term/    终端后端（unix / windows）
├── input/   字节流 → 事件 的增量解码状态机
├── screen/  Cell、双缓冲、diff 渲染、Canvas
├── widget/  Widget 接口、Base、FocusManager、Root 路由
│   └── layout/  约束布局：Flex、Stack、ClampScroll
├── theme/   默认主题（Dark / Light）
├── app/     事件循环、Post/Flush、生命周期与恢复
└── examples/ minimal、gallery、preview
```

依赖方向单向无环：`geom ← style ← event → input → screen ← widget ← theme`，`app` 在最上层。

## 测试

```bash
./go.sh vet ./...      # 静态检查
./go.sh test ./...     # 全量单测
```

沙箱内 `openpty` 被拒绝，因此 term 层用内存假终端注入；真实 PTY 集成测试写成可跳过形式。跨平台验证：

```bash
GOOS=linux   ./go.sh build ./...
GOOS=windows ./go.sh build ./...
```

## 文档

- [中文使用文档](docs/usage.zh-CN.md) — 快速开始、事件处理、布局、15 个控件、主题、中英混排
- [English usage guide](docs/usage.en.md) — same content in English
