package widget

import (
	"testing"

	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/style"
	"PaintedSkin/text"
	"PaintedSkin/theme"
	"PaintedSkin/widget/layout"
)

// widgetTheme 走生产用的转换路径（theme.Theme → widget.Theme）。
func widgetTheme(th theme.Theme) *Theme { return ThemeFrom(th) }

func eventMouseEnter() event.Mouse { return event.Mouse{Kind: event.MouseEnter} }

// theme 由 app 层注入（widget 不 import theme 会成环，故用接口注入）。
// 这里用最小接口验证"未设 Style 时回落主题"。
type styler interface{ SetTheme(*theme.Theme) }

// Button 未设自身 Style → 绘制使用注入的主题（ButtonNormal 可为零值）。
func TestButtonFallsBackToTheme(t *testing.T) {
	th := theme.Dark()
	th.ButtonNormal = style.Style{}.WithForeground(style.Cyan)
	b := NewButton("X")
	b.SetTheme(widgetTheme(th))
	b.Layout(geom.Rect{X: 0, Y: 0, W: 8, H: 1})

	mc := newMemCanvas(8, 1)
	b.Paint(mc)
	if len(mc.texts) != 1 {
		t.Fatalf("texts = %+v", mc.texts)
	}
	if mc.texts[0].st.FG != style.Cyan {
		t.Errorf("normal must use theme ButtonNormal: %+v", mc.texts[0].st)
	}
}

// hover 态走主题 ButtonHover。
func TestButtonHoverUsesTheme(t *testing.T) {
	th := theme.Dark()
	b := NewButton("X")
	b.SetTheme(widgetTheme(th))
	b.Layout(geom.Rect{X: 0, Y: 0, W: 8, H: 1})
	b.Handle(eventMouseEnter())
	mc := newMemCanvas(8, 1)
	b.Paint(mc)
	if !mc.texts[0].st.Bold {
		t.Errorf("hover must use theme hover style: %+v", mc.texts[0].st)
	}
}

// 控件自身 Style 优先于主题。
func TestOwnStyleOverridesTheme(t *testing.T) {
	th := theme.Dark()
	b := NewButton("X")
	b.SetTheme(widgetTheme(th))
	b.Style = style.Style{}.WithItalic(true)
	b.Layout(geom.Rect{X: 0, Y: 0, W: 8, H: 1})
	mc := newMemCanvas(8, 1)
	b.Paint(mc)
	if !mc.texts[0].st.Italic {
		t.Errorf("own style must win: %+v", mc.texts[0].st)
	}
}

// 换肤后新样式生效（SetDefault → 新控件使用新主题）。
func TestThemeSwapAffectsNewWidgets(t *testing.T) {
	orig := theme.Current()
	defer theme.SetDefault(orig)

	th := theme.Dark()
	th.ButtonNormal = style.Style{}.WithUnderline(true)
	b := NewButton("X")
	b.SetTheme(widgetTheme(th))
	b.Layout(geom.Rect{X: 0, Y: 0, W: 8, H: 1})
	mc := newMemCanvas(8, 1)
	b.Paint(mc)
	if !mc.texts[0].st.Underline {
		t.Errorf("themed style must apply: %+v", mc.texts[0].st)
	}
}

// List 选中行走主题 ListSelected。
func TestListSelectedUsesTheme(t *testing.T) {
	th := theme.Light()
	l := NewList()
	l.SetItems([]string{"a", "b"})
	l.SetTheme(widgetTheme(th))
	l.Layout(geom.Rect{X: 0, Y: 0, W: 8, H: 2})
	l.SetSelected(0)
	l.OnFocus()
	mc := newMemCanvas(8, 2)
	l.Paint(mc)
	if len(mc.texts) == 0 {
		t.Fatal("no rows")
	}
	if mc.texts[0].st != th.ListSelected {
		t.Errorf("selected row = %+v want %+v", mc.texts[0].st, th.ListSelected)
	}
}

// ProgressBar 绘制走主题底色。
func TestProgressUsesTheme(t *testing.T) {
	th := theme.Dark()
	p := NewProgressBar()
	p.SetTheme(widgetTheme(th))
	p.Layout(geom.Rect{X: 0, Y: 0, W: 12, H: 1})
	mc := newMemCanvas(12, 1)
	p.Paint(mc)
	if len(mc.texts) != 1 {
		t.Fatalf("texts = %+v", mc.texts)
	}
	if mc.texts[0].st != th.TextInputNormal && mc.texts[0].st != style.Default {
		t.Errorf("progress style = %+v", mc.texts[0].st)
	}
}

// Panel 默认内边距必须为 1（内容不贴边框）。
func TestPanelDefaultPadding(t *testing.T) {
	p := NewPanel("t")
	if p.Padding() != (geom.SymInsets(1)) {
		t.Errorf("default padding = %+v want 1 on all sides", p.Padding())
	}
	// Measure = 内容 + 标题行(1) + padding*2 + border*2
	p.SetChild(NewLabel("ab")) // 2x1
	if got := p.Measure(looseC()); got != (geom.Size{W: 6, H: 6}) {
		t.Errorf("measure = %+v want 6x6", got)
	}
	// 无标题时不占标题行
	p2 := NewPanel("")
	p2.SetChild(NewLabel("ab"))
	if got := p2.Measure(looseC()); got != (geom.Size{W: 6, H: 5}) {
		t.Errorf("no-title measure = %+v want 6x5", got)
	}
}

func looseC() layout.Constraints { return layout.Constraints{MaxW: 80, MaxH: 24} }

// Panel 绘制不得破坏边框：顶行标题两侧仍是横线，角不被覆盖。
func TestPanelTitleKeepsBorder(t *testing.T) {
	p := NewPanel("中文标题")
	p.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 5})
	mc := newMemCanvas(20, 5)
	p.Paint(mc)
	if len(mc.boxes) != 1 {
		t.Fatalf("must draw one box: %+v", mc.boxes)
	}
	// 标题写在内容行（顶行之下），不覆盖边框
	for _, tc := range mc.texts {
		if tc.y == 0 {
			t.Errorf("title must not paint on border row: %+v", tc)
		}
	}
}

// Panel 内容宽度不得超过内容区（否则会吃掉右边框）。
func TestPanelContentFitsInside(t *testing.T) {
	p := NewPanel("t")
	p.SetChild(NewLabel("中文混排mixed"))
	p.Layout(geom.Rect{X: 0, Y: 0, W: 20, H: 5})
	child := p.Child().(*Label)
	b := child.Bounds
	if b.X < 1 || b.Y < 1 {
		t.Errorf("content overlaps border: %+v", b)
	}
	if b.Right() > p.Bounds.Right()-1 {
		t.Errorf("content overlaps right border: %+v vs panel %+v", b, p.Bounds)
	}
	if b.Bottom() > p.Bounds.Bottom()-1 {
		t.Errorf("content overlaps bottom border: %+v", b)
	}
}

// 中英混排下的整框完整性：Panel 每一行的最右可见字符都必须是右边框。
// 这是"中英混排 + 边框规整"的可执行验收。
func TestPanelFrameIntactWithMixedText(t *testing.T) {
	cases := []string{
		"中文标题 mixed",
		"ascii only",
		"中a文b混排",
		"",
		"emoji 😀 行",
	}
	for _, s := range cases {
		p := NewPanel("标题中文")
		p.SetChild(NewLabel(s))
		bounds := geom.Rect{X: 0, Y: 0, W: 20, H: 6}
		p.Layout(bounds)

		buf := newMemCanvas(20, 6)
		// 先铺一层空格，模拟真实帧
		buf.Fill(bounds, ' ', style.Default)
		p.Paint(buf)

		if len(buf.boxes) != 1 || buf.boxes[0].r != bounds {
			t.Fatalf("%q: box = %+v", s, buf.boxes)
		}
		// 边框右侧一列（含四个角）必须被 Box 覆盖，且没有被文字越界
		for _, tc := range buf.texts {
			if tc.x+text.Width(tc.s) > bounds.Right()-1 {
				t.Errorf("%q: text %q@%d overruns right border", s, tc.s, tc.x)
			}
			if tc.y >= bounds.Bottom()-1 {
				t.Errorf("%q: text %q@y=%d overruns bottom border", s, tc.s, tc.y)
			}
		}
	}
}
