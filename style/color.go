// Package style 定义终端颜色与文本属性。
// Color 是携带"深度种类"的值类型：渲染层根据终端能力决定发射哪种 SGR，
// 因此控件只声明想要的颜色，不关心降级细节。
package style

// ColorKind 表示颜色的表示深度。
type ColorKind uint8

const (
	// ColorDefault 使用终端默认前景/背景。
	ColorDefault ColorKind = iota
	// ColorANSI 是 0..15 的基础 16 色。
	ColorANSI
	// Color256 是 0..255 的 256 色。
	Color256
	// ColorRGB 是 24 位真彩。
	ColorRGB
)

// Color 是一个终端颜色值。
type Color struct {
	Kind    ColorKind
	N       uint8 // ColorANSI / Color256 使用
	R, G, B uint8 // ColorRGB 使用
}

// DefaultColor 表示"终端默认色"。
var DefaultColor = Color{Kind: ColorDefault}

// RGB 构造 24 位真彩颜色。
func RGB(r, g, b uint8) Color { return Color{Kind: ColorRGB, R: r, G: g, B: b} }

// ANSIColor 构造基础 16 色（0..15）。
func ANSIColor(n uint8) Color { return Color{Kind: ColorANSI, N: n} }

// Hex 按 #RRGGBB 构造真彩颜色。
func Hex(v uint32) Color {
	return RGB(uint8(v>>16), uint8(v>>8), uint8(v))
}

// 具名 16 色常量，主题与控件统一使用。
var (
	Black         = ANSIColor(0)
	Red           = ANSIColor(1)
	Green         = ANSIColor(2)
	Yellow        = ANSIColor(3)
	Blue          = ANSIColor(4)
	Magenta       = ANSIColor(5)
	Cyan          = ANSIColor(6)
	White         = ANSIColor(7)
	BrightBlack   = ANSIColor(8)
	BrightRed     = ANSIColor(9)
	BrightGreen   = ANSIColor(10)
	BrightYellow  = ANSIColor(11)
	BrightBlue    = ANSIColor(12)
	BrightMagenta = ANSIColor(13)
	BrightCyan    = ANSIColor(14)
	BrightWhite   = ANSIColor(15)
)

// to256Cube 把 0..255 分量量化到 6 级立方 (0,95,135,175,215,255)。
func to256Cube(v uint8) (level int) {
	switch {
	case v < 48:
		return 0
	case v < 115:
		return 1
	default:
		return int((int(v) - 35) / 40)
	}
}

var cubeLevels = [6]uint8{0, 95, 135, 175, 215, 255}

func cubeDist(v, level uint8) int {
	d := int(v) - int(cubeLevels[level])
	return d * d
}

// To256 把真彩量化为 256 色索引（16..231 立方 + 232..255 灰阶）。
// 亮度接近灰阶梯度时优先用灰阶，避免彩色噪点。
func (c Color) To256() uint8 {
	if c.Kind == ColorDefault {
		return 7
	}
	if c.Kind == ColorANSI {
		return c.N
	}
	if c.Kind == Color256 {
		return c.N
	}
	// 灰阶带：r==g==b 时考虑 24 级灰阶。
	if c.R == c.G && c.G == c.B {
		if c.R < 8 {
			return 16 // 立方黑
		}
		if c.R > 248 {
			return 231 // 立方白
		}
		idx := (int(c.R)-8)/10 + 232
		if idx > 255 {
			idx = 255
		}
		return uint8(idx)
	}
	ri, gi, bi := to256Cube(c.R), to256Cube(c.G), to256Cube(c.B)
	best := 16 + 36*ri + 6*gi + bi
	bestDist := cubeDist(c.R, uint8(ri)) + cubeDist(c.G, uint8(gi)) + cubeDist(c.B, uint8(bi))

	// 同时比较最接近的灰阶色。
	grayIdx := (int(c.R) + int(c.G) + int(c.B)) / 3
	grayLevel := (grayIdx-3)/10 + 232
	if grayLevel >= 232 && grayLevel <= 255 {
		gv := 8 + (grayLevel-232)*10
		d := (int(c.R)-gv)*(int(c.R)-gv) + (int(c.G)-gv)*(int(c.G)-gv) + (int(c.B)-gv)*(int(c.B)-gv)
		if d < bestDist {
			return uint8(grayLevel)
		}
	}
	return uint8(best)
}

// To16 把任意颜色量化为基础 16 色（供 ColorDepth 仅 16 色的终端使用）。
func (c Color) To16() uint8 {
	if c.Kind == ColorDefault {
		return 7
	}
	if c.Kind == ColorANSI {
		return c.N & 0x0F
	}
	r, g, b := c.R, c.G, c.B
	if c.Kind == Color256 && c.N >= 16 {
		r, g, b = from256(c.N)
	}
	// 先判亮度极值，保证纯黑/纯白/纯三原色落在确定端点。
	sum := int(r) + int(g) + int(b)
	switch {
	case sum < 64:
		return 0 // black
	case r > 200 && g < 96 && b < 96:
		return 9 // bright red
	case r < 96 && g > 200 && b < 96:
		return 10
	case r < 96 && g < 96 && b > 200:
		return 12
	case r > 200 && g > 200 && b < 96:
		return 11
	case r > 200 && g < 96 && b > 200:
		return 13
	case r < 96 && g > 200 && b > 200:
		return 14
	case r > 200 && g > 200 && b > 200:
		return 15
	}
	// 一般情形：按亮度 + 主导通道选 8 色。
	bright := sum >= 384
	base := uint8(0)
	mx := max3(r, g, b)
	switch {
	case mx == int(r) && mx == int(g):
		base = 3 // yellow dominant
	case mx == int(r) && mx == int(b):
		base = 5 // magenta
	case mx == int(g) && mx == int(b):
		base = 6 // cyan
	case mx == int(r):
		base = 1
	case mx == int(g):
		base = 2
	default:
		base = 4
	}
	if int(r)+int(g)+int(b) < 96 {
		// 很暗时判为黑。
		return 0
	}
	if bright {
		return base + 8
	}
	return base
}

func max3(r, g, b uint8) int {
	m := int(r)
	if int(g) > m {
		m = int(g)
	}
	if int(b) > m {
		m = int(b)
	}
	return m
}

// from256 把 256 色索引还原为近似 RGB（供向 16 色降级时判亮度）。
func from256(n uint8) (r, g, b uint8) {
	switch {
	case n < 16:
		return ansiRGB(n)
	case n < 232:
		i := int(n - 16)
		r = cubeLevels[i/36]
		g = cubeLevels[(i/6)%6]
		b = cubeLevels[i%6]
		return
	default:
		v := uint8(8 + (int(n)-232)*10)
		return v, v, v
	}
}

var ansi16 = [16][3]uint8{
	{0, 0, 0}, {205, 49, 49}, {13, 188, 121}, {229, 229, 16},
	{36, 114, 200}, {188, 63, 188}, {17, 168, 205}, {229, 229, 229},
	{102, 102, 102}, {241, 76, 76}, {35, 209, 139}, {245, 245, 67},
	{59, 142, 234}, {214, 112, 214}, {41, 184, 219}, {255, 255, 255},
}

func ansiRGB(n uint8) (r, g, b uint8) {
	if n > 15 {
		return 229, 229, 229
	}
	c := ansi16[n]
	return c[0], c[1], c[2]
}
