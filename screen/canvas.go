package screen

import "PaintedSkin/geom"
import "PaintedSkin/style"

// Canvas 是控件的绘图目标，提供带裁剪的写入操作。
type Canvas interface {
	// SetCell 在指定位置写入单个单元格。坐标超出裁剪区则丢弃。
	SetCell(x, y int, r rune, s style.Style)
	// Fill 用字符和样式填充矩形区域。
	Fill(rect geom.Rect, r rune, s style.Style)
	// FillStyle 用空格和样式铺满矩形：控件 Paint 的第一步，用于铺自身背景，
	// 使区域内每格要么被写入内容、要么被写入控件自己的背景（无空洞）。
	FillStyle(rect geom.Rect, s style.Style)
	// Text 在指定位置写入文本，会按宽字符步进。
	Text(x, y int, s string, st style.Style)
	// Box 绘制带边框的矩形框（逐边开关由 border.Edges 决定）。
	Box(rect geom.Rect, border Border, s style.Style)
	// Outline 在区域外围套一圈只换样式的环：字符与宽度都不动，
	// 因此既不吃内容也不占布局（焦点环）。逐边开关见 BorderEdges。
	Outline(rect geom.Rect, edges BorderEdges, s style.Style)
	// Hatch 在已铺好的背景上按固定密度点缀底纹；只落在空格上，
	// 绝不覆盖真实内容，也不改变任何格子的字符与宽度。
	Hatch(rect geom.Rect, hatch Hatch, s style.Style)
	// Clip 设置当前的裁剪矩形。 thereafter SetCell/Fill/Text 的坐标相对于此矩形。
	Clip(rect geom.Rect)
	// PopClip 移除当前的裁剪。
	PopClip()
	// Size 返回 canvas 的总尺寸。
	Size() geom.Size
}
