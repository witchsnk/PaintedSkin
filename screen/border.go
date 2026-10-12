package screen

import "strings"

// 本文件是全框架**边框与块元素字形的唯一真相源**。
//
// 设计抄自 Textual 的 `_border.py:20-125`：每套边框是一个 3 行 × 3 列的
// 字形矩阵，而不是六个局部变量：
//
//	行 0 = 上（顶边）    列 0 = 左角/左侧   列 1 = 横线    列 2 = 右角/右侧
//	行 1 = 中（侧边）    列 0 = 竖线       列 1 = 内部    列 2 = 竖线
//	行 2 = 下（底边）    列 0 = 左下角     列 1 = 横线    列 2 = 右下角
//
// 上下共用同一个"横线"字形，左右列在顶/底行当角、在中行当竖边，
// 因此一张 3×3 表就能推导全部 9 个位置。
//
// 边框语义层与文本属性层（style.Attr）**解耦**：边框样式只由字形表达。
// 旧实现用 `style.AttrBold` / `AttrUnderline` / `AttrBold|AttrReverse`
// 当"边框线型"的传输格式，于是 `BorderDouble` 与 `BorderHeavy` 的字形
// 互相对调，且这些属性同时又是文字强调属性，两种关切混在一个位掩码里。

// BorderGlyphs 是 3×3 边框字形矩阵（下标 [行][列]，见文件头说明）。
type BorderGlyphs [3][3]rune

// BorderEdges 是四条边的开关，对应 Textual 的
// border-top / border-right / border-bottom / border-left。
type BorderEdges struct {
	Top, Right, Bottom, Left bool
}

// AllEdges 是四条边全开的快捷值。
var AllEdges = BorderEdges{Top: true, Right: true, Bottom: true, Left: true}

// Border 是一套边框：可读的样式名 + 字形矩阵 + 逐边开关。
//
// 零值 Border 即"无边框"（字形全空、四边全关），因此不需要额外的
// None 枚举分支 —— 布局侧只需 Visible() 一个真值判断。
type Border struct {
	// Style 是样式名（"single"/"double"/…），仅用于日志与调试。
	Style string
	// Glyphs 是 3×3 字形矩阵。
	Glyphs BorderGlyphs
	// Edges 是逐边开关。
	Edges BorderEdges
}

// 边框表。上下共用横线字形；左右列在顶/底行为角、在中行为侧边。
var (
	// BorderNone 无边框（零值）。
	BorderNone = Border{}

	// BorderSingle 单线框（默认）。
	BorderSingle = Border{Style: "single", Edges: AllEdges, Glyphs: BorderGlyphs{
		{'┌', '─', '┐'},
		{'│', ' ', '│'},
		{'└', '─', '┘'},
	}}

	// BorderDouble 双线框。
	BorderDouble = Border{Style: "double", Edges: AllEdges, Glyphs: BorderGlyphs{
		{'╔', '═', '╗'},
		{'║', ' ', '║'},
		{'╚', '═', '╝'},
	}}

	// BorderRound 圆角框。
	BorderRound = Border{Style: "round", Edges: AllEdges, Glyphs: BorderGlyphs{
		{'╭', '─', '╮'},
		{'│', ' ', '│'},
		{'╰', '─', '╯'},
	}}

	// BorderHeavy 粗线框。
	BorderHeavy = Border{Style: "heavy", Edges: AllEdges, Glyphs: BorderGlyphs{
		{'┏', '━', '┓'},
		{'┃', ' ', '┃'},
		{'┗', '━', '┛'},
	}}

	// BorderAscii 纯 ASCII 框（`+--+` / `|`），终端字形缺失时的兜底。
	BorderAscii = Border{Style: "ascii", Edges: AllEdges, Glyphs: BorderGlyphs{
		{'+', '-', '+'},
		{'|', ' ', '|'},
		{'+', '-', '+'},
	}}

	// BorderDashed 虚线框。
	BorderDashed = Border{Style: "dashed", Edges: AllEdges, Glyphs: BorderGlyphs{
		{'┌', '╌', '┐'},
		{'╎', ' ', '╎'},
		{'└', '╌', '┘'},
	}}

	// BorderThick 半块粗框：一行之内表达"加粗且带 padding"的边框
	// （上半块 ▀ 做顶边、下半块 ▄ 做底边、█ 做角与侧边）。
	BorderThick = Border{Style: "thick", Edges: AllEdges, Glyphs: BorderGlyphs{
		{'█', '▀', '█'},
		{'█', ' ', '█'},
		{'█', '▄', '█'},
	}}

	// BorderInner 内贴粗线：上下边用"贴着内侧"的那半块
	// （顶边 ▄ 画在格子下半、底边 ▀ 画在上半），看起来像从框内贴边。
	BorderInner = Border{Style: "inner", Edges: AllEdges, Glyphs: BorderGlyphs{
		{'▗', '▄', '▖'},
		{'▐', ' ', '▌'},
		{'▝', '▀', '▘'},
	}}

	// BorderOuter 外扩粗线：与 BorderInner 相反 —— 顶边 ▀ 在上半、
	// 底边 ▄ 在下半，且左右半块对调，看起来像向外扩边。
	BorderOuter = Border{Style: "outer", Edges: AllEdges, Glyphs: BorderGlyphs{
		{'▛', '▀', '▜'},
		{'▌', ' ', '▐'},
		{'▙', '▄', '▟'},
	}}
)

// 全部边框（便于遍历/校验）。
//
//nolint:gochecknoglobals // 只读字形表
var AllBorders = []Border{
	BorderSingle, BorderDouble, BorderRound, BorderHeavy,
	BorderAscii, BorderDashed, BorderThick, BorderInner, BorderOuter,
}

// BorderByName 按样式名查表（未知名返回 BorderNone, false）。
func BorderByName(name string) (Border, bool) {
	for _, b := range AllBorders {
		if b.Style == name {
			return b, true
		}
	}
	return BorderNone, false
}

// Visible 报告该边框是否会画出至少一条边。
//
// 必须**既有字形、边也开着**才算可见：`BorderNone`（零字形）即使被
// `WithEdges` 打开边，字形仍是 0 —— 此时若照画，Box 会往环上写
// `Cell{Rune:0, Width:0}`，也就是把"无边框"变成"一圈空洞"。
func (b Border) Visible() bool {
	return b.Edges != (BorderEdges{}) && b.Glyphs != (BorderGlyphs{})
}

// At 返回矩阵位置 (row, col) 的字形；越界返回 0。
func (b Border) At(row, col int) rune {
	if row < 0 || row > 2 || col < 0 || col > 2 {
		return 0
	}
	return b.Glyphs[row][col]
}

// HLine 返回横线字形（顶边/底边的填充），供只要一根线的控件复用。
func (b Border) HLine() rune { return b.At(0, 1) }

// VLine 返回竖线字形（侧边）。
func (b Border) VLine() rune { return b.At(1, 0) }

// WithEdges 返回逐边开关替换后的边框（字形不变；值语义，不改接收者）。
func (b Border) WithEdges(e BorderEdges) Border {
	b.Edges = e
	return b
}

// BarGlyphs 是条形/滚动条/滑块的块元素字形表：全框架绘制"条"的唯一词汇。
// 所有字形必须是单列宽，否则会撑破框体（见 TestBarGlyphTable）。
type BarGlyphs struct {
	// Track 是未填充的轨道（滚动条轨道、滑块未达部分）。
	Track rune
	// Filled 是已填充部分（进度块、滚动条滑块）。
	Filled rune
	// KnobH 是水平滑块的手柄。
	KnobH rune
	// KnobV 是垂直滑块的手柄。
	KnobV rune
	// Pad 是 ASCII 条（`[===>  ]`）的填充字符。
	Pad rune
	// CapL / CapR 是 ASCII 条的左右端帽。
	CapL rune
	CapR rune
}

// Bar 是默认块元素字形表。
var Bar = BarGlyphs{
	Track:  '░',
	Filled: '█',
	KnobH:  '>',
	KnobV:  '▐',
	Pad:    '=',
	CapL:   '[',
	CapR:   ']',
}

// Line 返回按行重复 n 次的字符串（控件拼条形线用，避免各处各写一套
// O(n) 拼接循环；n <= 0 时为空串）。
func (g BarGlyphs) Line(r rune, n int) string {
	if n <= 0 || r == 0 {
		return ""
	}
	return strings.Repeat(string(r), n)
}

// Hatch 是底纹装饰（Textual 的 `hatch`）：在**已铺好的背景**上按固定密度
// 点缀字形，用来区分"这一块是空的"与"这一块有底纹"。
//
// 底纹只在空格上落笔，绝不覆盖真实内容；字符/宽度都不动，因此不破坏
// 文本宽度口径（docs/05）。
type Hatch int

const (
	// HatchNone 无底纹。
	HatchNone Hatch = iota
	// HatchLeft 左斜纹（20%）。
	HatchLeft
	// HatchRight 右斜纹（20%）。
	HatchRight
	// HatchCross 交叉斜纹（20%，主副字形交替）。
	HatchCross
)

// hatchDensity 是底纹密度：每 5 格点 1 个 ≈ 20%。
const hatchDensity = 5

// hatchGlyphs 是底纹字形表：[0] 主字形（相位 0），[1] 副字形（相位 1，
// 交叉纹用；为 0 表示该模式只有一个字形）。
//
//nolint:gochecknoglobals // 只读字形表
var hatchGlyphs = map[Hatch][2]rune{
	HatchLeft:  {'╱', 0},
	HatchRight: {'╲', 0},
	HatchCross: {'╱', '╲'},
}
