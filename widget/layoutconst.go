package widget

import (
	"PaintedSkin/text"
	"PaintedSkin/widget/layout"
)

// 本文件是控件绘制/测量路径上的**命名常量**与共享小工具。
//
// 此前这些数字散落在各控件的 Paint/Measure 里（`text+4`、`W-2`、`+2`、
// `H-2`、`min 20`…），含义只能靠上下文猜。同一个数字在多个控件出现时
// （例如三种标记形式都是"文本宽 + 4"）改一处就会漏另一处。

// ---- 标记形式（Button / Checkbox / Radio）----
//
// 三种形式都是"1 格端帽 + 1 格 padding + 文本 + 1 格 padding + 1 格端帽"：
//
//	Button   `[ x ]`      Checkbox `[X] `      Radio  `(x) `
const (
	markCapW = 1 // 每端 1 格方括号/圆括号
	markPadW = 1 // 紧贴文本的 1 格空格
)

// ---- 首选尺寸（Measure 的默认值）----
const (
	progressPrefW  = 20 // ProgressBar 首选宽
	progressPrefH  = 3  // ProgressBar 首选高
	sliderPrefW    = 30 // Slider（水平）首选宽
	sliderPrefH    = 3  // Slider（水平）首选高
	sliderPrefVW   = 15 // Slider（垂直）首选宽
	sliderPrefVH   = 7  // Slider（垂直）首选高
	textInputPrefW = 17 // TextInput 在无宽度约束时的默认宽
	tabPadW        = 1  // Tabs 标签两侧各 1 格空格
)

// ---- Dialog ----
const (
	dialogMinW = 20 // Dialog 自适应时的最小宽
	dialogMinH = 7  // Dialog 自适应时的最小高
	// dialogBtnRowDy 是按钮行相对 Bounds 底边的偏移（-2 = 底边框上一行）。
	dialogBtnRowDy = -2
)

// ---- List ----
const (
	listScrollbarW = 1 // 右侧滚动条列
	listGapW       = 1 // 滚动条与文本之间的留白
)

// ---- 带框控件的退化下限 ----

// 带框控件至少要 3 列（左右边框 + 1 格内容）、1 行。
// TextInput / ProgressBar 此前各自写死 `b.W < 3 || b.H < 1`。
const (
	minFrameW = 3
	minFrameH = 1
)

// textAlign 把布局对齐映射为文本层的对齐（text.Fit 用）。
//
// 两套 Align 是刻意分开的：layout.Align 描述"子项在容器里怎么摆"，
// text.Align 描述"一段文字在给定列宽里怎么补齐"（text 是列宽的唯一口径）。
func textAlign(a layout.Align) text.Align {
	switch a {
	case layout.AlignCenter:
		return text.Center
	case layout.AlignEnd:
		return text.Right
	default:
		return text.Left
	}
}

// fitText 把内容按列宽裁剪并按 align 补齐到恰好 width 列。
// 控件内部的表格单元格/标签段统一走这里（`text.Fit` 是唯一权威）。
func fitText(s string, width int, align layout.Align) string {
	return text.Fit(s, width, textAlign(align))
}
