# PaintedSkin 使用文档（中文）

> 英文版：[usage.en.md](usage.en.md)

一个从零实现的 Go TUI 底层框架：在终端里提供与 GUI 同级的控件界面（鼠标全支持、焦点管理、模态对话框、约束布局），同时保持命令行的简洁。

```bash
go run ./examples/minimal    # 最小示例（需真实终端）
go run ./examples/gallery    # 控件总览（需真实终端）
go run ./examples/preview    # 无头渲染：把画面打印到 stdout（无 TTY 也能看）
```

---

## 目录

1. [快速开始](#1-快速开始)
2. [核心概念](#2-核心概念)
3. [统一事件处理](#3-统一事件处理)
4. [布局](#4-布局)
5. [控件参考](#5-控件参考)
6. [弹层：Dialog 与覆盖层](#6-弹层dialog-与覆盖层)
7. [主题](#7-主题)
8. [文本宽度与中英混排](#8-文本宽度与中英混排)
9. [样式与颜色](#9-样式与颜色)
10. [线程模型](#10-线程模型)
11. [终端能力与降级](#11-终端能力与降级)
12. [无 TTY 环境](#12-无-tty-环境)
13. [自定义控件](#13-自定义控件)
14. [测试与调试](#14-测试与调试)
15. [完整示例](#15-完整示例)

---

## 1. 快速开始

### 1.1 最小应用

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

`Setup` 在终端接管之后、`事件循环开始之前` 调用一次，因此里面可以安全地
`a.Size()` 拿真实终端尺寸。

### 1.2 三种运行形态

| 形态 | 入口 | 说明 |
|---|---|---|
| 交互式 | `app.Run(cfg)` | 接管终端，事件循环，Esc 退出 |
| 无头渲染 | `app.New(cfg)` + `a.Draw()` | 不接管终端，把画面写进 `a.Buffer()` |
| 嵌入测试 | `app.New(cfg)` + `a.Dispatch(e)` | 注入合成事件，断言状态 |

无头渲染示例见 [§12](#12-无-tty-环境)。

---

## 2. 核心概念

### 2.1 保留模式对象树

控件是**持久对象**，不是每帧重建的函数：

```go
btn := paintedskin.Button("确定", onConfirm)
btn.SetDisabled(true)          // 随时改状态
btn.SetText("提交中…")
label.SetText("已提交")        // Label 会被就地更新
```

### 2.2 四阶段生命周期

```
Measure(Constraints) → Size     自底向上：报告"我要多大"
Layout(Rect)                     自顶向下：分配"给你这块地"
Paint(Canvas)                    绘制（先铺自身背景，越界由 Canvas/上层裁剪）
Handle(Event) bool               处理事件，返回 true = 已消费
```

对应 `docs/06-widget-model.md`。

> **尾部空档纪律**：控件 `Paint` 的第一步是铺自身背景（`Canvas.FillStyle`），
> 因此 `Bounds` 内每格要么是内容、要么是控件自己的背景色空格 —— 不会出现
> "换肤底色到不了空档""高亮停在文字末尾""框体内部露出终端底色"这类缺口。
> 状态高亮（聚焦/选中/按下）一律覆盖整个控件宽度。**全部 15 个控件**都遵守。

### 2.3 语义样式（焦点/悬停/选中/禁用）

这四种状态走主题，代码里**没有**硬编码的 `AttrReverse` / `WithDim`：

| 语义 | 主题字段 | 无主题兜底 |
|---|---|---|
| 焦点 | `FocusColor`（前景）| `AttrReverse` |
| 悬停 | `HoverBG`（背景）| `Bold` |
| 选中 | `SelectionBG`（背景）| `AttrReverse` |
| 禁用 | `DisabledColor`（前景）| `Dim` |

优先级：**控件自身 `Style` > 主题语义色 > 内置兜底**。

### 2.4 焦点环（outline）

```go
ti.SetFocusRing(true)      // 聚焦时套一圈环（默认关闭）
ti.FocusRingEnabled()
```

环只**换样式**、不动字符，因此不吃内容也不占布局（`Bounds` 不变）。
环画在 `Paint` 末尾（晚于内容），所以整圈都可见。
无框控件的首末字符会被环盖住 —— 那种场景建议留在"语义色"而不是开环。

### 2.5 包结构与依赖方向

```
geom ← style ← event
        ↑        ↑          text（宽度口径，被 screen/widget 共用）
      screen   input
        ↑
    widget ← theme      layout
        ↑
       app  ←  根包 paintedskin（构造器糖）
```

依赖单向无环：`widget` **不得** import `term`/`app`；`screen` **不得** import
`widget`/`layout`。

### 2.6 常用类型

```go
// geom：单元格坐标系，(0,0) 在左上，X 右增 Y 下增
type Point struct{ X, Y int }
type Size  struct{ W, H int }
type Rect  struct{ X, Y, W, H int }   // 右/下边界开区间
type Insets struct{ Top, Right, Bottom, Left int }

// event
type Key   struct{ Code KeyCode; Rune rune; Mod Mod; Press bool }
type Mouse struct{ X, Y int; Kind MouseKind; Button MouseButton; Mod Mod; Drag bool }
type Resize struct{ W, H int }
type Paste  struct{ Text string }
type Focus  struct{ Gained bool }
```

---

## 3. 统一事件处理

**所有输入都从 `Root.Dispatch` 进入**，框架负责命中测试、捕获、冒泡、焦点路由，
控件只需实现 `Handle`。

### 3.1 事件从哪来

```
终端字节流 ──► input.Decoder ──┐
                               ├──► Root.Dispatch ──► 控件树
SIGWINCH / 轮询 ─► event.Resize┘
```

应用层通常**不直接调 `Dispatch`**，由 `app.Run` 的事件循环驱动。若要手动注入
（测试、无头渲染），直接调即可：

```go
a.Dispatch(event.Key{Code: event.KeyEnter, Press: true})
a.Dispatch(event.Mouse{X: 3, Y: 5, Kind: event.MousePress, Button: event.ButtonLeft})
```

### 3.2 路由规则（docs/06）

**鼠标**

1. 从根 `HitTest`，**z 序最顶层优先**（后加入 / 后绘制的先命中）
2. `Press` 命中的控件获得**鼠标捕获**：后续 `Move`/`Release` 直达它，
   即使光标已移出其边界（拖拽 Slider、拖选文本的基础）
3. `Move` 产生 hover：进入发 `MouseEnter`，离开发 `MouseLeave`；
   **按下期间不发 Leave**
4. 未被子级消费的鼠标事件**冒泡到父**

```go
// 捕获语义：Press 后移出边界，Move/Release 仍到原控件
r.Dispatch(press(0, 0))
r.Dispatch(move(30, 15))      // 仍在拖 a
r.Dispatch(release(30, 15))   // 界外释放 → 不触发 OnClick
```

**键盘**

1. 路由给**当前焦点控件**
2. 焦点控件不消费 → 沿父链冒泡（模态 Dialog 内不外泄）
3. 全局键（Tab / Shift+Tab）由 FocusManager 先拦截

### 3.3 事件类型速查

| 类别 | 类型 | 说明 |
|---|---|---|
| 键盘 | `event.Key` | `Code` + `Rune` + `Mod` + `Press`；解码器恒置 `Press:true` |
| 鼠标 | `event.Mouse` | `Kind` 决定动作；`X/Y` 为 0 基单元格坐标 |
| 尺寸 | `event.Resize` | 窗口变化，双缓冲全量失效后重绘 |
| 粘贴 | `event.Paste` | 括号粘贴的完整内容 |
| 焦点 | `event.Focus` | 终端焦点获得/失去 |

`MouseKind`：`MousePress`、`MouseRelease`、`MouseMove`、`MouseWheelUp`、
`MouseWheelDown`、`MouseEnter`、`MouseLeave`。

> `MouseEnter/Leave` **仅由框架合成**，`input.Decoder` 永不产出。

常用 `KeyCode`：`KeyRune`（可打印字符看 `Rune`）、`KeyEnter`、`KeyEsc`、
`KeyTab`、`KeySpace`（=0x20）、`KeyBackspace`、`KeyDelete`、`KeyUp/Down/Left/Right`、
`KeyHome/End`、`KeyPgUp/PgDn`、`KeyF1..KeyF12`。

`Mod`：`ModShift`、`ModCtrl`、`ModAlt`、`ModMeta`。

### 3.4 焦点管理

```go
r := a.Root()

r.Dispatch(event.Key{Code: event.KeyTab, Press: true})          // 下一个
r.Dispatch(event.Key{Code: event.KeyTab, Mod: event.ModShift})  // 上一个（成环）

r.Focus(btn)            // 主动聚焦
r.Focused()             // 查询当前焦点（可能是 nil）
```

规则：

- 遍历序 = 树序，只收集 `Focusable && 有效可见 && 有效可用` 的控件，**成环**
- 点击聚焦：命中的控件若可聚焦则夺取焦点；不可聚焦或点空白**焦点不变**
- 聚焦变化 → 旧控件 `OnBlur`、新控件 `OnFocus`，各自打脏重绘
- 控件是否入遍历由 `SetFocusable(bool)` 控制；`Radio` 成员默认不可聚焦
  （由 `RadioGroup` 占一个 Tab 位）

### 3.5 脏标记与重绘

```go
w.Invalidate()          // 自身打脏 + 请求宿主重绘
root.NeedPaint()        // 是否有待重绘请求
a.Draw()                // 手动重绘一帧
a.Flush()               // 执行 Post 队列 + 按需重绘
```

---

## 4. 布局

### 4.1 三种用法

**① 显式坐标**（简单场景、可控）

```go
label.Layout(geom.Rect{X: 2, Y: 2, W: 30, H: 1})
```

**② 构造器糖容器**（推荐，响应式）

```go
col := paintedskin.Column(
    paintedskin.Gap(0, paintedskin.Label("标题")),
    paintedskin.Gap(1, paintedskin.Button("确定", onOK)),
    paintedskin.Gap(1, paintedskin.HSlider(40)),
    paintedskin.Gap(1, paintedskin.Fixed(8, paintedskin.List(items...))),  // 固定 8 行
    paintedskin.Grow(1, paintedskin.TextInput()),                    // 占据剩余空间
)
col.Layout(area)
```

- `paintedskin.Column(...)` / `paintedskin.Row(...)`：主轴依次排布
- `paintedskin.Gap(n, children...)`：设置兄弟间距（`n` 列/行）
- `paintedskin.Fixed(h, w)`：固定高度（`Row` 中为固定宽度）
- `paintedskin.Grow(weight, w)`：按权重分配剩余空间
- `paintedskin.Centered(w)`：交叉轴居中

`Column` / `Row` / `Root` **默认透明**（不铺自己的底色），子控件各铺各的；
要整块染色就设样式（**零值 = 透明**）：

```go
col := paintedskin.Column(paintedskin.Label("x"))
col.SetStyle(style.Style{}.WithBackground(style.ANSIColor(238)))
```

未设样式的子控件不会戳穿这层底色（"零样式 = 透明叠加"），所以分层背景可用。

**③ 布局引擎直接调用**（自定义排布）

```go
rects := layout.FlexRow(area, []layout.FlexItem{
    {Pref: geom.Size{W: 4, H: 1}},              // 理想 4
    {Pref: geom.Size{W: 4, H: 1}, Grow: 1},     // 拿走剩余
}, layout.AlignStretch, 0)
```

### 4.2 Flex 算法（docs/07）

1. 先给所有子项其 `Measure` 所得的理想尺寸
2. 主轴有剩余 → 按 `Grow` 权重分配
3. 主轴不足 → 按 `Shrink` 权重压缩到各自 `Min`
4. 仍不够 → 按比例**裁剪保底**（不崩溃、不越界）
5. 交叉轴按 `Align` 摆放

`Align`：`AlignStart`、`AlignCenter`、`AlignEnd`、`AlignStretch`。

### 4.3 约束钳制

```go
c := layout.Constraints{MinW: 4, MaxW: 20, MinH: 1, MaxH: 5}
c.Clamp(geom.Size{W: 100, H: 0})   // → {W:20, H:1}
```

- 结果必落在 `[Min, Max]`；`Min > Max` 冲突时**以 Max 胜**（裁剪优先）
- 永不为负
- `MaxW < 0` 表示**无上界**

### 4.4 Stack（叠放定位）

```go
layout.Stack(area, geom.Size{W: 20, H: 7}, layout.StackCenter)
```

`StackTopLeft`、`StackCenter`、`StackBottomRight`、`StackFill`。
子项全部共享同一块 Rect——Panel 内容叠加、Dropdown 弹层、Dialog 覆盖都用它。

### 4.5 内容区与滚动

```go
// 外框 → 内容区（扣掉内边距与边框）
content := layout.ContentRect(bounds, geom.SymInsets(1), 1)

// 滚动偏移钳制在 [0, content-view]
off := layout.ClampScroll(offset, contentLen, viewLen)
```

### 4.6 布局约束

- 每次 `Layout` 给子控件的 Rect 不得超出父的内容区
- 窗口过小：Min 冲突时裁剪保底，不崩溃、不越界
- `Slider` 默认**垂直**（`Measure` 要 7 行）；行内使用请用 `paintedskin.HSlider(v)`
  或 `SetHorizontal(true)`

---

## 5. 控件参考

全部 15 个控件。根包糖与 `widget` 包构造器一一对应（如 `paintedskin.Label` ↔ `widget.NewLabel`）。

### 5.1 速查表

| 控件 | 构造器 | 可聚焦 | 主要回调 |
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
| Dialog | `paintedskin.Dialog(t, m, labels, cb)` | 内部按钮 | `SetOnButton` / `SetOnClose` |
| StatusBar | `paintedskin.StatusBar(l, c, r)` | ✗ | — |
| Separator | `paintedskin.Separator()` | ✗ | — |

**通用属性**（全部控件都有，继承自 `widget.Base`）：

```go
w.SetVisible(bool)
w.SetEnabled(bool)      // 等价 SetDisabled(!v)
w.SetStyle(style.Style) // 覆盖主题
w.SetFocusable(bool)
w.SetTheme(*widget.Theme)
b := w.BoundsOf()       // 当前边界
w.Invalidate()
```

> 禁用态（`SetDisabled(true)`）的控件**吞掉全部事件**，不会误触发回调。

### 5.2 Label — 文本

```go
lbl := paintedskin.Label("中文 mixed 文本")
lbl.SetAlign(layout.AlignCenter)   // layout.AlignStart(默认) / Center / End
lbl.SetWrap(true)           // 超宽自动折行（CJK 按列宽折，不劈字）
lbl.SetText("新文本")
lbl.Text()                  // 读取
```

- 测量：单行 = 文本列宽 × 1；折行 = 约束宽内多行最大宽 × 行数
- 无交互：不可聚焦，鼠标与键盘事件**冒泡给父**
- 示例：[`label_test.go`](widget/widget_test.go)

### 5.3 Button — 按钮（五态）

```go
btn := paintedskin.Button("确定", func() { submit() })
btn.SetWidth(20)            // 固定宽（0 = 文本 + padding 自适应）
btn.SetDisabled(true)
btn.Hover(); btn.Pressed(); btn.Focused(); btn.Disabled()
```

五态：`normal / hover / pressed / focused / disabled`
（hover 与 focus 可叠加；样式优先级 disabled > pressed/focused > hover）。

交互：

| 输入 | 行为 |
|---|---|
| `MouseEnter/Leave` | 切 hover |
| `Press`（界内左键） | 切 pressed + 请求鼠标捕获 |
| `Release`（界内） | 触发 `OnClick` |
| `Release`（界外） | **不触发**（按下-移出-移回的语义） |
| `Enter` / `Space`（聚焦时） | 触发 `OnClick` |
| 禁用 | 全吞 |

测量：文本宽 + 左右 padding（共 4 格）；可设固定宽。
绘制：`[ 文本 ]` 形式，**补齐到控件宽**（焦点/按下时反色带覆盖整个按钮）。

### 5.4 Checkbox — 复选框

```go
cb := paintedskin.Checkbox("启用功能")
cb.SetChecked(true)
cb.SetOnChange(func(on bool) { apply(on) })
cb.Checked()
```

交互：点击方框**或标签**均切换；`Space`/`Enter`（聚焦时）切换；v1 不做三态不确定。
绘制：`[X] 文字`，补齐到控件宽（焦点反色带覆盖整宽）。

### 5.5 RadioGroup — 单选组

```go
g := paintedskin.RadioGroup("theme", []string{"Dark", "Light"})
g.SetSelected(0)
g.SetOnChange(func(i int) { switch i { case 0: /* dark */ } })
g.Radio(1).Checked()   // 成员状态
```

- 绘制：`(x) 文字`，补齐到控件宽
- 组内**互斥**：选中一个，其余自动取消
- 聚焦时 `Up/Down/Left/Right` 在组内移动选择
- **组占一个 Tab 位**，成员 `Radio` 不可聚焦（避免逐个占位）
- `SetItems` 重建成员并保持已选项有效

### 5.6 TextInput — 单行输入

```go
in := paintedskin.TextInput()
in.SetPlaceholder("请输入…")
in.SetMaxLength(40)
in.SetOnSubmit(func(s string) { login(s) })     // Enter
in.SetOnConfirm(func(s string) { /* 别名 */ })
in.SetOnChange(func(s string) { /* 每次编辑 */ })
in.Cursor()                 // rune 索引
x, y, ok := in.CursorScreenPos()   // 硬件光标位置（宿主使用）
_, _, _ = x, y, ok
in.Selection()              // (start, end)；无选区为 (-1,-1)
in.Placeholder(); in.MaxLength()
in.SetBorder(widget.BorderRound)   // 默认 BorderHeavy（粗线）
in.SetFocusRing(true)              // 聚焦时套一圈焦点环（默认关闭）
in.Border(); in.FocusRingEnabled()
```

| 类别 | 按键 | 行为 |
|---|---|---|
| 光标 | `Left/Right` | 移动一位 |
| 光标 | `Ctrl+Left/Right` | 词跳 |
| 光标 | `Home/End` | 行首/行尾 |
| 编辑 | 可打印字符 | 插入（受 `MaxLength` 约束） |
| 编辑 | `Backspace` | 删前一个；若有选区则删选区 |
| 编辑 | `Delete` | 删后一个；若有选区则删选区 |
| 编辑 | `Ctrl+A` | 全选 |
| 编辑 | `Paste` 事件 | 批量插入（括号粘贴） |
| 提交 | `Enter` | 触发 `OnSubmit` |
| 鼠标 | 点击 | 定位光标（落在字符上 = 插到该字符**之前**） |
| 鼠标 | 按住拖动 | 选区 |
| 视图 | 自动 | 内容宽于视口时横向滚动，**光标始终可见** |
| 硬件光标 | 聚焦时 | 框架定位终端光标到插入位；失焦自动隐藏 |

### 5.7 Slider — 滑块

```go
s := paintedskin.HSlider(40)              // 水平；paintedskin.Slider() 是垂直
s.SetRange(0, 100)
s.SetStep(5)
s.SetOnChange(func(v int) { fmt.Println(v) })
s.Value()
```

交互：

| 输入 | 行为 |
|---|---|
| `Press`（轨道任意处） | 捕获 + **跳转**到点击位置 |
| `Move`（按住） | 连续换算新值 |
| `Release` | 结束拖拽 |
| `Left/Right/Up/Down` | ±`Step`（聚焦时） |
| `Home/End` | 到端点 |

方向：`SetHorizontal(bool)`；`Measure` 水平 30×3、垂直 15×7。
绘制（**两个方向都铺满全部行**，字形取自 `screen.Bar`）：

- 水平：`[===>   ]`（端帽 + 填充 + 手柄 `>`）。手柄恒在轨道内，
  满值时贴右端而不是消失。
- 垂直：逐行块条 —— 手柄行之前是已填充块 `█`，之后是轨道 `░`，
  手柄 `▐` 落在行中央。

### 5.8 ProgressBar — 进度条

```go
pb := paintedskin.ProgressBar()
pb.SetRange(0, 100)
pb.SetValue(40)
pb.SetShowPercent(true)
pb.SetBorder(widget.BorderRound)   // 默认 BorderHeavy（粗线）
pb.SetFormat("40%")
```

**交互：无**（不可聚焦、不消费按键、禁用时吞事件）。
三行高画框 + 填充块；单行高退化为 `[===   ]`。

### 5.9 List — 列表

```go
l := paintedskin.List("第一项", "第二项", "第三项")
l.SetItems(items)                     // 替换数据
l.SetSelected(2)
l.SetOnChange(func(i int) {})
l.SetOnActivate(func(i int) {})       // Enter
l.Selected(); l.Offset(); l.Len()
```

- **虚拟滚动**：只 Paint 可见行
- 选中初始为首项；`SetSelected` 越界自动钳制
- 交互：滚轮 ±1 行；`Up/Down` 移动选择并**自动滚入视**；
  `Home/End` 首尾；`PgUp/PgDn` 翻页；`Enter` 激活；点击选中
- 右侧 1 格宽滚动条指示（非独立交互控件）
- 绘制：先铺满整个区域（项数少于视口时空行也铺底）；行文本补齐到 `W-1`
  （右侧 1 列留给滚动条），选中行的背景/反色一直盖到行尾

### 5.10 Table — 表格

```go
tb := paintedskin.TableRows(
    paintedskin.Table(
        widget.TableColumn{Title: "Name", Width: 10},
        widget.TableColumn{Title: "N", Width: 6, Align: layout.AlignEnd},
        // Width: 0 → 自动均分剩余宽度
    ),
    [][]string{{"alice", "1"}, {"bob", "2"}},
)
tb.SetSelectedRow(1)
tb.SetOnActivate(func(row int) {})
tb.ColumnWidths()      // 当前生效列宽
tb.Offset(); tb.HOffset()
```

- **表头固定**（粗体，不随滚动），主体滚动
- 列宽：`Width > 0` 固定；`Width == 0` 均分剩余（最大余数法）
- 交互：滚轮垂直；`Up/Down` 选行 + 滚入视；`Enter` 激活；点击选行；
  `Left/Right` **水平滚动**（列总宽超视口时）
- 行选择只占用 `Up/Down`；水平滚动独立

### 5.11 Tabs — 标签页

```go
tabs := paintedskin.Tabs("One", "Two", "Three")
tabs.SetContent(paintedskin.Label("第一个页面的内容"))
tabs.SetCurrent(1)
tabs.SetOnChange(func(i int) {})
tabs.Labels(); tabs.Current(); tabs.Content()
```

- 点击标签切换；聚焦时 `Left/Right` 切页
- **标签条视为一组**：占一个 Tab 位，`Tab` 键进入内容区（内容控件入遍历）
- `Measure` 高 = 1（标签行）+ 内容高；`Layout` 内容区为 bounds 去掉首行

### 5.12 Dropdown — 下拉选择

```go
d := paintedskin.Dropdown("Go", "Rust", "Zig")
d.SetSelected(1)
d.SetOnChange(func(i int) {})
d.IsOpen(); d.Open(); d.Close(); d.Pick(i)
```

- 点击展开 → `Host.SetOverlay` 弹层；**下方优先，空间不足翻转到上方**
  （最多显示 6 项）
- 点选项 → 选择并收起；`Esc` 收起；**点外部收起**（`Root` 自动处理）
- 展开时 `Up/Down` 预选、`Enter` 确认收起
- 绘制：`<当前项> ▼`，**补齐到 `Measure` 报出的宽度**（空选时也不会留下缺口）

### 5.13 Scrollbar — 滚动条

```go
sb := widget.NewScrollbar()
sb.SetHorizontal(false)   // 默认垂直（贴右缘、宽 1 格）
sb.SetTotal(100)          // 内容总长
sb.SetView(20)            // 视口长度
sb.SetOffset(40)          // 直接设置（不触发 OnChange）
sb.SetOnChange(func(off int) {})
```

- 偏移自动钳制在 `[0, Total-View]`；内容不足视口时恒为 0
- 点击轨道跳页（±`View`）；**拖拽滑块**（捕获 → 按比例换算）
- **滚轮不消费**（转发给宿主，由列表/表格处理）
- setter 不触发 `OnChange`；只有交互触发
- 绘制：轨道 `screen.Bar.Track` + 滑块 `screen.Bar.Filled`；样式走主题
  `ScrollbarTrack` / `ScrollbarThumb`（后者未设时回落 `ScrollbarThumbColor`）

### 5.14 Panel — 容器

```go
p := paintedskin.Panel(" 标题 ", paintedskin.Column(
    paintedskin.Gap(1, paintedskin.Label("内容")),
))
p.SetBorder(widget.BorderDouble)         // None/Single(默认)/Double/Round/Heavy
p.SetBorderSpec(screen.BorderDashed)    // 任意字形集（内置 9 套）
p.SetBorderEdges(screen.BorderEdges{Top: true})   // 逐边开关（不改变布局）
p.SetHatch(screen.HatchCross)           // 内区底纹（~20% 斜纹）
p.BorderSpec(); p.Hatch()               // 生效的边框与底纹
p.SetPadding(geom.SymInsets(1))          // 默认 1
p.SetTitle("新标题")
p.Child()                                 // 取内容控件
// 注：Panel/Dialog 等容器不支持 SetDisabled（用 SetVisible 控制显隐）
```

- **默认内边距 1**：内容不贴边框
- 先铺满整个区域（内区 + padding 环落控件自身背景），再叠加边框
- 标题画在**边框内侧首行**，占 1 行并计入 `Measure`（不覆盖边框），
  并按 `框宽 - 4` 截断（左右各留 1 格空格），**长标题不会盖掉右边框**
- `Measure` = max(内容, 标题) + padding×2 + border×2
- 边框字形来自 `screen.Border`（3×3 字形表）：`BorderDouble` 双线 `╔═╗`、
  `BorderHeavy` 粗线 `┏━┓`、`BorderRound` 圆角、`BorderDashed` 虚线、
  `BorderThick` 半块粗边框；`SetBorderSpec` 可直接给任意字形集，
  `SetBorderEdges` 逐边开关（关掉的边不落笔，布局仍保留该边间距）
- 容器不消费事件（未被子控件消费的向上冒泡）
- `HitTest` 先探子控件，再回落自身

### 5.15 Dialog — 模态对话框

```go
dlg := paintedskin.Dialog("确认", "要继续吗？", []string{"确定", "取消"}, func(i int) {
    // i == 0 确定，1 == 取消
    root.PopDialog()
})
dlg.SetOnClose(func() { root.PopDialog() })   // Esc 路径
dlg.SetDim(true)                               // 遮罩
dlg.SetSize(40, 10)                            // 0 = 自适应
dlg.Layout(root.Bounds)                        // 居中于所给区域
dlg.SetBorder(widget.BorderRound)              // 枚举档
dlg.SetBorderSpec(screen.BorderThick)          // 任意字形集
dlg.SetBorderEdges(screen.BorderEdges{Top: true})  // 逐边开关
dlg.Border(); dlg.BorderSpec(); dlg.BorderEdges()
```

行为：

- 经 `Root.PushDialog` 入栈 → **模态**（栈下控件收不到鼠标与键盘）
- **焦点陷阱**：`Tab` 只在 Dialog 子树内循环；焦点默认落第一个按钮
- `Esc` → `Close()` → 触发 `OnClose` → `PopDialog`，**恢复原焦点**
- 命中顺序：覆盖层 > 栈顶 Dialog > 普通子控件
- 绘制：**总是铺满自身区域**（`SetDim` 只决定铺什么底，不再决定是否铺），
  再叠加边框；标题与 `Panel` 同一策略 —— 落在框内首行、按 `框宽 - 4` 截断；
  正文长行同样截断，长消息不会吃掉右边框

> `Dialog.Close()` 内部已调 `PopDialog`，`OnClose` 里**不要**再调一次。

### 5.16 StatusBar — 状态栏

```go
bar := paintedskin.StatusBar("app", "提示信息", "v1.0")
bar.SetLeft("x"); bar.SetCenter("y"); bar.SetRight("z")
bar.SetLeftStyle(style.Style{}.WithBold(true))    // 段级样式（零值回落控件 Style）
bar.Left(); bar.Center(); bar.Right()
```

- 高恒为 1；左段左对齐、中段居中、右段右对齐（按**列宽**计算，CJK 正确）
- 三段各自截断到互不重叠的区间（保位优先级 **右 > 中 > 左**）：
  段间不留空洞，三段也不会互相压过
- 先铺满整行（自身背景），空段不绘制；常驻底行

### 5.17 Separator — 分隔线

```go
sep := paintedskin.Separator()      // 默认水平
sep.SetHorizontal(false)    // 垂直
sep.SetLine('=')            // 自定义线字符
```

水平高恒 1、垂直宽恒 1（不随约束退化）；固定 1 格的那一维不会被压缩。

---

## 6. 弹层：Dialog 与覆盖层

两种机制对应两类弹层（`docs/05` 弹层模型）：

| 类型 | 机制 | 命中顺序 | 关闭方式 |
|---|---|---|---|
| Dialog | `Root.PushDialog/PopDialog` 模态栈 | 栈顶优先 | `Esc` / 按钮回调 `PopDialog` |
| Dropdown / 弹层 | `Host.SetOverlay(w)` 单实例覆盖层 | 覆盖层优先 | 点外部 / `Esc` / 选择后 |

```go
// 模态
root.PushDialog(dlg)
root.PopDialog()

// 覆盖层
root.SetOverlay(w)     // 非 nil 即显示；命中优先于普通子控件
root.SetOverlay(nil)   // 收起
root.Overlay()         // 查询
```

`Root` 实现 `Host` 接口，控件通过 `HostOf()` 拿到它，因此**控件不依赖 `app` 包**：

```go
type Host interface {
    Invalidate()             // 打脏请求重绘
    Focus(w Widget)          // 主动聚焦
    PushDialog(w Widget)     // 模态入栈
    PopDialog()              // 模态出栈 + 恢复焦点
    SetOverlay(w Widget)     // 单实例覆盖层
    Capture(w Widget)        // 显式请求鼠标捕获
}
```

---

## 7. 主题

```go
// 全局换肤（立即重绘）
a.ApplyTheme(theme.Light())

// 只换肤，下一帧生效
a.SetTheme(th)

// 自定义
th := theme.Dark()
th.FocusColor = style.Blue
th.ButtonHover = style.Style{}.WithBold(true)
theme.SetDefault(th)
```

优先级（`docs/08`）：**控件自身 `Style` > 主题 > 内置默认**。

```go
btn.SetStyle(myStyle)                       // 最高优先级
btn.SetTheme(widget.ThemeFrom(theme.Dark())) // 其次
// 都没有时用内置默认（保证五态视觉可区分）
```

`Theme` 字段：基础色 `FocusColor`、`BorderColor`、`DisabledColor`、`HoverBG`、
`SelectionBG`、`ScrollbarThumbColor`；控件样式 `ButtonNormal/Hover/Pressed/
Focused/Disabled`、`ListSelected`、`TableHeader`、`DialogMask`、`ScrollbarThumb/
Track`、`Separator`、`StatusBar`、`TextInputNormal/Focus`。

**全部字段都被控件读取**（`TestEveryThemeFieldIsWired` 逐字段验证）。

内置主题：`theme.Dark()`（默认）、`theme.Light()`、`theme.Default()`。
默认配色下焦点用 `BrightCyan` 前景、选中用背景色（Dark 为主题反色，
Light 为 `ANSIColor(153)`）—— 想恢复"反色带"风格把
`FocusColor`/`SelectionBG` 置零即可（会回落到内置 `AttrReverse`）。

---

## 8. 文本宽度与中英混排

面向中文用户，**列宽口径全框架统一**在 `text` 包（ADR-017）：

```go
text.Width("中文abc")        // 7  （中=2 文=2 a=1 b=1 c=1）
text.RuneWidth('中')          // 2
text.Truncate("中文abc", 4)   // "中文"（不劈开宽字符）
text.Fit("ab", 6, text.Center)
text.PadEnd("中文", 6)
text.OffsetAt(s, charIdx)    // 字符索引 → 屏幕列
text.IndexAtColumn(s, col)   // 屏幕列 → 字符索引
```

规则：

- CJK / 全角 = **2 列**，ASCII = **1 列**，组合附标 / 零宽连接符 = **0 列**
  （并入前一格）
- 截断只在字符边界进行，**绝不把宽字符劈成两半**
- 宽字符在缓冲里占两格：左格存字（`Width=2`），右格写占位（`Width=-1`）
- 宽字符若卡在行尾无处放右格 → **退化为空格**，绝不越界（否则终端换行、边框被撑破）
- 组合附标（`é` = `e` + U+0301）与 ZWJ 连接符（U+200D）**不占列**，并入基字符那一格；
  渲染时整串发出，终端才会合成一个字素
  > **已知限制**：ZWJ emoji（👨‍👩‍👧）的**列宽**仍按 3 个宽字符算 6 列，
  > 而支持 ZWJ 的终端渲染成 2 列 —— 差 4 列。彻底修正需要字素簇切分。

**已写成测试的不变量**：

1. 任何内容写入后，该行占用的终端列数 == 缓冲宽度
2. 中英混排（含全角、半角片假名、emoji、组合字符）后，每行最右可见字符仍是右边框
3. 截断不劈字
4. Panel 的子控件与标题都落在边框内侧

> **已知限制**：emoji 的 ZWJ 序列（如 👨‍👩‍👧）按码点求和，与部分终端实际渲染
> 可能差 1 列。彻底修正需字素簇切分，超出 v1 范围。

---

## 9. 样式与颜色

```go
st := style.Style{}.
    WithForeground(style.Cyan).
    WithBackground(style.ANSIColor(238)).
    WithBold(true)

st.WithDim(true).WithItalic(true).WithUnderline(true).
   WithBlink(true).WithReverse(true).WithStrike(true)
st.WithAttrs(style.AttrBold | style.AttrUnderline)   // 整体替换
st.AttrMask()      // 掩码
st.Equal(other)    // 视觉等价（忽略 Attrs 镜像）
st.Swap()          // 交换前景/背景
```

颜色：

```go
style.RGB(30, 144, 255)      // 真彩
style.Hex(0x1E90FF)           // #RRGGBB
style.ANSIColor(6)            // 16 色（Cyan）
style.Red / BrightRed / ...  // 具名常量
```

发射规则（按终端能力自动降级）：

| 深度 | 前景 | 背景 |
|---|---|---|
| `ColorANSI(0..7)` | `30–37` | `40–47` |
| `ColorANSI(8..15)` | `90–97` | `100–107` |
| `Color256` | `38;5;n` | `48;5;n` |
| `ColorRGB` | `38;2;r;g;b` | `48;2;r;g;b` |
| `ColorDefault` | 不发参数 | 不发参数 |

属性位：`AttrBold`、`AttrDim`、`AttrItalic`、`AttrUnderline`、`AttrBlink`、
`AttrReverse`、`AttrStrike`。

边框与条形字形（**全框架唯一真相源**在 `screen/border.go`）：

```go
screen.BorderSingle / Double / Round / Heavy / Ascii / Dashed / Thick
screen.BorderNone                       // 零值 = 无边框
screen.BorderByName("round")            // 按名查表
b := screen.BorderSingle.WithEdges(screen.BorderEdges{Top: true})
b.At(0, 0)        // 3×3 字形矩阵：[行 上/中/下][列 左/横/右]
b.HLine(); b.VLine()                    // 横线 / 竖线字形
screen.Bar.Track / Filled / KnobH / KnobV / Pad / CapL / CapR
screen.Bar.Line(screen.Bar.Track, 3)    // 重复 n 次
```

内置 9 套：`Single` `Double` `Round` `Heavy` `Ascii` `Dashed` `Thick`
（半块粗线 `█▀▄`）`Inner`（内贴半块）`Outer`（外扩半块）。
`BorderDouble` 画双线 `╔═╗`，`BorderHeavy` 画粗线 `┏━┓`。

底纹（~20% 密度，只落在空格上，绝不覆盖内容）：

```go
screen.HatchNone / HatchLeft / HatchRight / HatchCross
c.Hatch(rect, screen.HatchCross, st)    // Canvas 原语
```

几何：

```go
r := geom.Rect{X: 0, Y: 0, W: 20, H: 5}
r.Contains(geom.Point{X: 3, Y: 1})   // 右/下开区间
r.Right(); r.Bottom(); r.Size(); r.Empty()
r.Intersect(o); r.Union(o)
r.Inflate(geom.SymInsets(1))          // 收缩（padding）
r.Size().Area()
```

---

## 10. 线程模型

控件树**只在 UI goroutine 可变**。`Handle` / `Measure` / `Layout` / `Paint`
全部在主循环内串行执行，无需加锁。

外部 goroutine 改状态必须走 `Post`：

```go
go func() {
    // 正确：投递到 UI 线程
    a.Post(func() { label.SetText("done") })
}()

// 主循环内
a.Flush()      // 执行 Post 队列 + 按需重绘
```

`Dispatch` 返回后会自动 `Flush`。**不要**从 goroutine 直接调控件方法。

终端尺寸变化也只在主循环处理（`app.Run` 已保证），避免与绘制并发。

---

## 11. 终端能力与降级

进入时写入的模式（`docs/03`）：

| 序列 | 含义 |
|---|---|
| `CSI ?1049h` | 备用屏（退出恢复原屏幕） |
| `CSI ?25l` / `?25h` | 隐藏 / 显示硬件光标 |
| `CSI ?1002h` `?1003h` `?1006h` | 鼠标按键跟踪 + 任意移动 + SGR 扩展 |
| `CSI ?2004h` | 括号粘贴 |
| `CSI ?1004h` | 焦点事件 |
| raw mode | `ECHO/ICANON/ISIG/IEXTEN/IXON` 关闭，`VMIN=1 VTIME=0` |

退出时逆向恢复，**`?1049l` 必须最后输出**（其余序列在备用屏内失效）。

```go
app.Config{
    Width: 0, Height: 0,     // <=0 时取终端真实尺寸
    Writer: os.Stdout,       // ANSI 输出目标
    Mouse: true,             // 开启鼠标模式
    Paste: true,             // 开启括号粘贴
    FocusReports: true,      // 开启焦点事件
    Setup: func(a *app.App) { /* 构建控件树 */ },
}
```

降级：

- `TERM=dumb` 或未设 `TERM` → **自动禁用鼠标**，程序仍可用键盘操作
- 无 `COLORTERM=truecolor` → 256 色 → 16 色
- Windows：走 VT console mode（`ENABLE_VIRTUAL_TERMINAL_INPUT/OUTPUT`），
  输入即标准 VT 字节流，复用同一解码器；无 SIGWINCH，用 300ms 轮询产生 resize

终端恢复（AGENT.md 2.4，硬性不变量）：

1. `Terminal.Close()` **幂等**（`sync.Once`）
2. `defer` / 信号 / `panic recover` **三路径共用**同一恢复逻辑
3. 退出后还原：raw mode、备用屏、光标、鼠标模式、括号粘贴、焦点事件

---

## 12. 无 TTY 环境

CI / 沙箱里 `openpty` 被拒，`app.Run` 会返回 `app.ErrNoTerminal`。两种做法：

**① 用 `examples.Run` 优雅降级**

```go
func main() {
	examples.Run(func(a *app.App) { /* ... */ })
}
```

**② 无头渲染：把画面打印出来**

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
			continue // 空 / 宽字符右格占位（保留列位）
		}
		fmt.Print(string(c.Rune))
	}
	fmt.Println()
}
```

`app.New` **无输出副作用**（终端未接管时不写任何字节）。

---

## 13. 自定义控件

嵌入 `widget.Base` 获得全部默认实现，只写差异逻辑：

```go
type LED struct {
	widget.Base
	on bool
}

func NewLED() *LED {
	l := &LED{}
	l.Base = widget.NewBase()
	l.Base.Bind(l)      // 必须：让 HitTest 返回外层控件而非 *Base
	return l
}

func (l *LED) Measure(c layout.Constraints) geom.Size {
	return c.Clamp(geom.Size{W: 3, H: 1})
}

func (l *LED) Handle(e event.Event) bool {
	if !l.EffectivelyEnabled() {
		return true            // 禁用吞事件
	}
	if m, ok := e.(event.Mouse); ok && m.Kind == event.MousePress {
		l.on = !l.on
		l.Invalidate()
		return true
	}
	return false               // 不消费 → 冒泡给父
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

要点：

- **`Bind` 必须调用**：Go 嵌入无法让 `Base.HitTest` 返回外层控件，
  不调用会导致焦点/捕获/冒泡全部错位
- 嵌入后自动获得：`Handle`/`HitTest`/`Measure`/`Layout`/`Paint`/`Children`/
  `SetHost` 等默认实现
- 需要子控件时覆盖 `Children()`（并让容器递归 `SetHost`）
- 想跟随 Tab：构造时 `l.Focusable = true`（或 `SetFocusable(true)`）

容器实现 `Children()` 后会自动参与：焦点遍历、命中测试、主题注入、`SetHost` 递归。

---

## 14. 测试与调试

```bash
./go.sh vet ./...      # 静态检查
./go.sh test ./...     # 全量单测
GOOS=windows ./go.sh build ./...   # 交叉编译
```

> 沙箱内全局 Go 模块缓存不可写，统一用 `./go.sh` 包装器。

### 合成事件断言

```go
r := widget.NewRoot()
r.Layout(geom.Rect{X: 0, Y: 0, W: 40, H: 20})
btn := widget.NewButton("OK")
clicks := 0
btn.SetOnClick(func() { clicks++ })
r.Add(btn)
btn.Layout(geom.Rect{X: 0, Y: 0, W: 6, H: 1})

// 序列：按下 → 释放
r.Dispatch(event.Mouse{X: 1, Y: 0, Kind: event.MousePress, Button: event.ButtonLeft})
r.Dispatch(event.Mouse{X: 1, Y: 0, Kind: event.MouseRelease, Button: event.ButtonLeft})
if clicks != 1 { t.Fatalf("clicks = %d", clicks) }

// 焦点
r.Dispatch(event.Key{Code: event.KeyTab, Press: true})
if r.Focused() != btn { t.Fatalf("focus = %v", r.Focused()) }
```

### 应用层断言

```go
a := app.New(app.Config{Width: 40, Height: 10, Writer: io.Discard})
// ... 挂控件 ...
a.Layout()
a.Draw()
// 从 a.Buffer() 断言画面
```

`a.Buffer().At(x, y)` 返回 `*screen.Cell`（`Rune` / `Width` / `Style`）。

### 文档示例校验

`docs/examples/compile_test.go` 编译并运行文档里的每段示例，改文档时同步改它——
`go test ./docs/examples` 会挡住"文档写了但跑不通"的代码。

---

## 15. 完整示例

### 15.1 登录表单

```go
func build(a *app.App) {
	root := a.Root()
	sz := a.Size()

	username := paintedskin.TextInput()
	username.SetPlaceholder("用户名")
	password := paintedskin.TextInput()
	password.SetPlaceholder("密码")

	msg := paintedskin.Label("")
	submit := paintedskin.Button("登录", func() {
		if username.Value() == "" {
			msg.SetText("请输入用户名")
			return
		}
		dlg := paintedskin.Dialog("欢迎", "你好，"+username.Value(), []string{"好"}, func(int) {
			root.PopDialog()
		})
		root.PushDialog(dlg)
		dlg.Layout(root.Bounds)
	})

	form := paintedskin.Panel(" 登录 ", paintedskin.Column(
		paintedskin.Gap(1, paintedskin.Label("用户名")),
		paintedskin.Gap(1, username),
		paintedskin.Gap(1, paintedskin.Label("密码")),
		paintedskin.Gap(1, password),
		paintedskin.Gap(2, submit),
		paintedskin.Gap(1, msg),
	))

	bar := paintedskin.StatusBar("demo", "", "Tab 切换焦点")
	root.Add(form)
	root.Add(bar)

	y := (sz.H - 8) / 2
	form.Layout(geom.Rect{X: (sz.W - 34) / 2, Y: y, W: 34, H: 8})
	bar.Layout(geom.Rect{X: 0, Y: sz.H - 1, W: sz.W, H: 1})
}
```

### 15.2 列表 + 详情联动

```go
func build(a *app.App) {
	root := a.Root()
	sz := a.Size()

	items := []string{"alpha", "beta", "gamma", "delta", "epsilon"}
	list := paintedskin.List(items...)
	detail := paintedskin.Label("选中：alpha")

	list.SetOnChange(func(i int) {
		if i >= 0 && i < len(items) {
			detail.SetText("选中：" + items[i])
		}
	})
	list.SetOnActivate(func(i int) {
		dlg := paintedskin.Dialog("确认", "打开 "+items[i]+" ?", []string{"打开", "取消"}, func(int) {
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

### 15.3 后台任务更新（线程安全）

```go
func build(a *app.App) {
	bar := paintedskin.ProgressBar()
	bar.SetRange(0, 100)
	bar.SetShowPercent(true)
	a.Root().Add(bar)
	bar.Layout(geom.Rect{X: 2, Y: 2, W: 40, H: 3})

	// 模拟下载：后台协程算进度，UI 线程更新控件
	go func() {
		for i := 0; i <= 100; i += 5 {
			v := i
			a.Post(func() { bar.SetValue(v) })
			time.Sleep(80 * time.Millisecond)
		}
	}()
}
```

### 15.4 响应式双栏

```go
func build(a *app.App) {
	root := a.Root()
	sz := a.Size()

	left := paintedskin.Panel(" 左栏 ", paintedskin.Column(
		paintedskin.Gap(1, paintedskin.Label("内容 A")),
		paintedskin.Gap(1, paintedskin.Checkbox("选项 A")),
	))
	right := paintedskin.Panel(" 右栏 ", paintedskin.Column(
		paintedskin.Gap(1, paintedskin.Label("内容 B")),
		paintedskin.Gap(1, paintedskin.HSlider(30)),
	))

	root.Add(left)
	root.Add(right)

	// 用 FlexRow 平分宽度，随窗口尺寸自适应
	body := geom.Rect{X: 0, Y: 0, W: sz.W, H: sz.H}
	rects := layout.FlexRow(body, []layout.FlexItem{
		{Pref: geom.Size{W: sz.W / 2, H: sz.H}},
		{Pref: geom.Size{W: sz.W - sz.W/2, H: sz.H}},
	}, layout.AlignStretch, 1)
	left.Layout(rects[0])
	right.Layout(rects[1])
}
```

### 15.5 监听窗口尺寸变化

```go
func build(a *app.App) {
	panel := paintedskin.Panel("可调整", paintedskin.Label("拖动窗口边缘试试"))
	root := a.Root()
	root.Add(panel)

	// 首次布局
	relayout := func() {
		sz := a.Size()
		panel.Layout(geom.Rect{X: 2, Y: 2, W: sz.W - 4, H: sz.H - 4})
	}
	relayout()

	// 后续 resize 事件由 app.Run 触发；需手动重排时：
	// 监听 a.Root().NeedPaint() 或包装一层
}
```

> 提示：`app.Run` 收到 `event.Resize` 会重建缓冲并重绘。控件若需按新尺寸重新
> 排布，把布局逻辑写成幂等函数，在 `Setup` 与尺寸变化后各调一次。

---

## 附：相关文档

| 文档 | 内容 |
|---|---|
| [01-overview](docs/01-overview.md) | 项目目标、验收标准 |
| [02-architecture](docs/02-architecture.md) | 分层、数据流、线程模型 |
| [03-terminal](docs/03-terminal.md) | 终端后端、raw mode、能力探测 |
| [04-input](docs/04-input.md) | 输入解码状态机 |
| [05-render](docs/05-render.md) | 双缓冲、diff、宽字符、色彩降级 |
| [06-widget-model](docs/06-widget-model.md) | 控件模型、事件路由、焦点管理 |
| [07-layout](docs/07-layout.md) | 约束布局算法 |
| [08-widgets](docs/08-widgets.md) | 15 个控件的行为规格 |
| [09-roadmap](docs/09-roadmap.md) | 里程碑与验收清单 |
| [10-decisions](docs/10-decisions.md) | ADR 决策记录 |