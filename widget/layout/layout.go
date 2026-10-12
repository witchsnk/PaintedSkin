package layout

import (
	"PaintedSkin/geom"
)

// Constraints 描述父容器对子控件期望尺寸的钳制区间（docs/07）。
//
// Measure 返回值必须落在 [Min, Max] 内；Min > Max 冲突时以裁剪优先
// （Max 胜出），且结果永不为负、不 panic。
type Constraints struct {
	MinW, MinH, MaxW, MaxH int
}

// Clamp 把期望尺寸钳制进约束区间。
func (c Constraints) Clamp(want geom.Size) geom.Size {
	w := want.W
	if w < c.MinW {
		w = c.MinW
	}
	if c.MaxW >= 0 && w > c.MaxW {
		w = c.MaxW
	}
	h := want.H
	if h < c.MinH {
		h = c.MinH
	}
	if c.MaxH >= 0 && h > c.MaxH {
		h = c.MaxH
	}
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	return geom.Size{W: w, H: h}
}

// Align 是 Flex 交叉轴的摆放方式。
type Align int

const (
	// AlignStart 交叉轴起点对齐。
	AlignStart Align = iota
	// AlignCenter 交叉轴居中。
	AlignCenter
	// AlignEnd 交叉轴末端对齐。
	AlignEnd
	// AlignStretch 交叉轴拉伸填满。
	AlignStretch
)

// FlexItem 是 Flex 容器的一个子项。
type FlexItem struct {
	// Min 是主轴/交叉轴均可压缩到的下限（各自维度取对应分量）。
	Min geom.Size
	// Pref 是子项 Measure 所得的理想尺寸。
	Pref geom.Size
	// Grow 是主轴有剩余时分配权重的分子（<=0 表示不参与分配）。
	Grow float64
	// Shrink 是主轴不足时压缩权重（<=0 表示不可压缩，只能被裁剪）。
	Shrink float64
}

// FlexRow 在 area 内沿水平主轴排列子项，返回每个子项的最终 Rect。
// spacing 为兄弟间距（docs/07 的 Margin 由此承担）。
func FlexRow(area geom.Rect, items []FlexItem, align Align, spacing int) []geom.Rect {
	widths := flexAxis(area.W, len(items), spacing,
		func(i int) (pref, min int, grow, shrink float64) {
			it := items[i]
			return it.Pref.W, it.Min.W, it.Grow, it.Shrink
		})
	out := make([]geom.Rect, len(items))
	x := area.X
	for i, w := range items {
		_ = w
		cw := widths[i]
		ch, y := crossAxis(area.Y, area.H, items[i].Pref.H, align)
		r := geom.Rect{X: x, Y: y, W: cw, H: ch}
		out[i] = r.Intersect(area)
		x += cw + spacing
	}
	return out
}

// FlexColumn 在 area 内沿垂直主轴排列子项，返回每个子项的最终 Rect。
func FlexColumn(area geom.Rect, items []FlexItem, align Align, spacing int) []geom.Rect {
	heights := flexAxis(area.H, len(items), spacing,
		func(i int) (pref, min int, grow, shrink float64) {
			it := items[i]
			return it.Pref.H, it.Min.H, it.Grow, it.Shrink
		})
	out := make([]geom.Rect, len(items))
	y := area.Y
	for i := range items {
		ch := heights[i]
		cw, x := crossAxis(area.X, area.W, items[i].Pref.W, align)
		// crossAxis 返回 (size, pos)，此处复用：cw 为交叉轴宽，x 为起点
		r := geom.Rect{X: x, Y: y, W: cw, H: ch}
		out[i] = r.Intersect(area)
		y += ch + spacing
	}
	return out
}

// flexAxis 计算主轴上每项的最终长度。
// total 为主轴可用长度，get 返回每项 (理想, 最小, grow, shrink)。
func flexAxis(total, n, spacing int, get func(i int) (pref, min int, grow, shrink float64)) []int {
	out := make([]int, n)
	if n == 0 {
		return out
	}
	if spacing < 0 {
		spacing = 0
	}
	avail := total - spacing*(n-1)
	if avail < 0 {
		avail = 0
	}
	prefs := make([]int, n)
	mins := make([]int, n)
	sum := 0
	for i := 0; i < n; i++ {
		p, m, _, _ := get(i)
		if p < 0 {
			p = 0
		}
		if m < 0 {
			m = 0
		}
		prefs[i] = p
		mins[i] = m
		sum += p
	}
	switch {
	case sum == avail:
		copy(out, prefs)
	case sum < avail:
		// 按 Grow 权重分配剩余空间。
		extra := avail - sum
		totalGrow := 0.0
		for i := 0; i < n; i++ {
			_, _, g, _ := get(i)
			if g > 0 {
				totalGrow += g
			}
		}
		copy(out, prefs)
		if totalGrow > 0 && extra > 0 {
			adds := make([]int, n)
			frac := make([]float64, n)
			base := 0
			for i := 0; i < n; i++ {
				_, _, g, _ := get(i)
				if g <= 0 {
					continue
				}
				exact := float64(extra) * g / totalGrow
				adds[i] = int(exact)
				frac[i] = exact - float64(adds[i])
				base += adds[i]
			}
			// 最大余数法分发剩余格，保证总和精确。
			rest := extra - base
			for rest > 0 {
				best, bestF := -1, -1.0
				for i := 0; i < n; i++ {
					_, _, g, _ := get(i)
					if g <= 0 {
						continue
					}
					if frac[i] > bestF {
						bestF, best = frac[i], i
					}
				}
				if best < 0 {
					break
				}
				adds[best]++
				frac[best] = -1
				rest--
			}
			for i := range out {
				out[i] += adds[i]
			}
		}
	default:
		// 主轴不足：先按 Shrink 权重压缩到各自 Min，仍不够则裁剪。
		deficit := sum - avail
		weights := make([]float64, n)
		totalW := 0.0
		for i := 0; i < n; i++ {
			_, _, _, s := get(i)
			reducible := prefs[i] - mins[i]
			if reducible < 0 {
				reducible = 0
			}
			if s > 0 {
				weights[i] = s * float64(reducible)
				totalW += weights[i]
			}
		}
		copy(out, prefs)
		if totalW > 0 && deficit > 0 {
			cuts := make([]int, n)
			frac := make([]float64, n)
			base := 0
			for i := 0; i < n; i++ {
				if weights[i] <= 0 {
					continue
				}
				exact := float64(deficit) * weights[i] / totalW
				maxCut := prefs[i] - mins[i]
				if maxCut < 0 {
					maxCut = 0
				}
				c := int(exact)
				if c > maxCut {
					c = maxCut
				}
				cuts[i] = c
				frac[i] = exact - float64(int(exact))
				base += c
			}
			// 压缩量不足（被 Min 截顶）时，剩余缺口转入裁剪阶段。
			cut := 0
			for _, c := range cuts {
				cut += c
			}
			for i := range out {
				out[i] -= cuts[i]
				if out[i] < mins[i] {
					out[i] = mins[i]
				}
			}
			deficit -= cut
		}
		now := 0
		for _, v := range out {
			now += v
		}
		if now > avail {
			// 裁剪：按当前比例压缩（保底，不崩溃）。
			scaled := make([]int, n)
			acc := 0
			for i, v := range out {
				scaled[i] = v * avail / now
				acc += scaled[i]
			}
			rest := avail - acc
			for i := 0; rest > 0 && i < n; i++ {
				scaled[i]++
				rest--
			}
			out = scaled
		}
	}
	for i, v := range out {
		if v < 0 {
			out[i] = 0
		}
	}
	return out
}

// crossAxis 计算交叉轴的 (尺寸, 起点)。
func crossAxis(origin, total, pref int, align Align) (int, int) {
	if pref < 0 {
		pref = 0
	}
	if total < 0 {
		total = 0
	}
	switch align {
	case AlignStretch:
		return total, origin
	case AlignCenter:
		if pref >= total {
			return total, origin
		}
		return pref, origin + (total-pref)/2
	case AlignEnd:
		if pref >= total {
			return total, origin
		}
		return pref, origin + total - pref
	default: // AlignStart
		if pref > total {
			pref = total
		}
		return pref, origin
	}
}

// StackPos 是 Stack 叠放的子项定位方式。
type StackPos int

const (
	// StackTopLeft 左上定位。
	StackTopLeft StackPos = iota
	// StackCenter 居中定位。
	StackCenter
	// StackBottomRight 右下定位。
	StackBottomRight
	// StackFill 填充整个区域。
	StackFill
)

// Stack 把一个子项放入 area，全部子项共享同一块 Rect（Panel 内容叠加、
// Dropdown 弹层、Dialog 覆盖均用此定位）。超大子项裁剪保底。
func Stack(area geom.Rect, want geom.Size, pos StackPos) geom.Rect {
	if pos == StackFill {
		return area
	}
	w, h := want.W, want.H
	if w < 0 {
		w = 0
	}
	if h < 0 {
		h = 0
	}
	if w > area.W {
		w = area.W
	}
	if h > area.H {
		h = area.H
	}
	x, y := area.X, area.Y
	switch pos {
	case StackCenter:
		x = area.X + (area.W-w)/2
		y = area.Y + (area.H-h)/2
	case StackBottomRight:
		x = area.X + area.W - w
		y = area.Y + area.H - h
	}
	return geom.Rect{X: x, Y: y, W: w, H: h}
}

// ContentRect 从外框扣除内边距与边框宽度得到内容区。
func ContentRect(outer geom.Rect, pad geom.Insets, border int) geom.Rect {
	if border < 0 {
		border = 0
	}
	return outer.Inflate(geom.Insets{
		Top:    pad.Top + border,
		Right:  pad.Right + border,
		Bottom: pad.Bottom + border,
		Left:   pad.Left + border,
	})
}

// ClampScroll 把滚动偏移钳制在 [0, content-view]；内容小于视口时恒为 0。
func ClampScroll(offset, content, view int) int {
	max := content - view
	if max < 0 {
		max = 0
	}
	if offset < 0 {
		return 0
	}
	if offset > max {
		return max
	}
	return offset
}
