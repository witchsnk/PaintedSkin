// Package text 统一全框架的文本宽度与截断规则。
//
// 全框架（screen 绘制、layout 对齐、widget 排版、preview 输出）都只经由
// 本包测量列宽，避免各处各写一套导致中英混排时边框错位。
//
// 规则（docs/05 宽字符处理）：
//
//   - 列宽一律按**终端单元格**计算：CJK/全角 = 2，ASCII = 1，
//     组合附标/零宽连接符 = 0（并入前一格）。
//   - 截断只在字符边界进行，绝不把一个宽字符劈成两半。
//   - 填充按列宽补空格，对齐用 Left/Center/Right。
package text

import (
	"strings"

	"github.com/mattn/go-runewidth"
)

// RuneWidth 返回单字符占用的列数（0/1/2）。
func RuneWidth(r rune) int {
	w := runewidth.RuneWidth(r)
	if w < 0 {
		return 0
	}
	return w
}

// Width 返回字符串占用的终端列数。这是全框架唯一的宽度来源。
func Width(s string) int { return runewidth.StringWidth(s) }

// Truncate 按列宽裁剪字符串；不会劈开宽字符。
// budget <= 0 时返回空串。
func Truncate(s string, budget int) string {
	if budget <= 0 {
		return ""
	}
	if Width(s) <= budget {
		return s
	}
	var sb strings.Builder
	acc := 0
	for _, r := range s {
		rw := RuneWidth(r)
		if rw == 0 {
			// 零宽字符不占列；若前面已有内容则并入。
			if acc > 0 {
				sb.WriteRune(r)
			}
			continue
		}
		if acc+rw > budget {
			break
		}
		sb.WriteRune(r)
		acc += rw
	}
	return sb.String()
}

// Fit 裁剪并按 align 填充到恰好 width 列。
func Fit(s string, width int, align Align) string {
	if width <= 0 {
		return ""
	}
	s = Truncate(s, width)
	pad := width - Width(s)
	if pad <= 0 {
		return s
	}
	fill := strings.Repeat(" ", pad)
	switch align {
	case Center:
		l := pad / 2
		return strings.Repeat(" ", l) + s + strings.Repeat(" ", pad-l)
	case Right:
		return fill + s
	default:
		return s + fill
	}
}

// PadEnd 在右侧补空格至 width 列（不裁剪）。
func PadEnd(s string, width int) string {
	if width <= 0 {
		return ""
	}
	pad := width - Width(s)
	if pad <= 0 {
		return s
	}
	return s + strings.Repeat(" ", pad)
}

// PadStart 在左侧补空格至 width 列（不裁剪）。
func PadStart(s string, width int) string {
	if width <= 0 {
		return ""
	}
	pad := width - Width(s)
	if pad <= 0 {
		return s
	}
	return strings.Repeat(" ", pad) + s
}

// OffsetAt 返回前 charIdx 个字符占用的列数（charIdx 为字符数，非字节数）。
func OffsetAt(s string, charIdx int) int {
	if charIdx <= 0 {
		return 0
	}
	w := 0
	n := 0
	for _, r := range s {
		if n >= charIdx {
			break
		}
		w += RuneWidth(r)
		n++
	}
	return w
}

// IndexAtColumn 把列号换算为字符索引。
//
// 语义：返回"光标应停在第几个字符之前"，即列 col 左侧已完整经过的字符数。
// 落在宽字符的**左半格**时返回其左格索引；落在**右半格**时返回其右邻索引
// （光标落在字符中间时按"已越过该字符"处理）。
func IndexAtColumn(s string, col int) int {
	if col <= 0 {
		return 0
	}
	w := 0
	idx := 0
	for _, r := range s {
		rw := RuneWidth(r)
		if w >= col {
			break
		}
		w += rw
		idx++
	}
	return idx
}

// Align 是横向对齐方式。
type Align int

const (
	// Left 左对齐（默认）。
	Left Align = iota
	// Center 居中。
	Center
	// Right 右对齐。
	Right
)
