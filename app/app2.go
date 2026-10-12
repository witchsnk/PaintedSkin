package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"PaintedSkin/event"
	"PaintedSkin/geom"
	"PaintedSkin/input"
	"PaintedSkin/screen"
	"PaintedSkin/term"
	"PaintedSkin/theme"
	"PaintedSkin/widget"
)

// Config 是 App 的构造参数。零值经 withDefaults 补全为可用配置。
type Config struct {
	// Width/Height 是初始尺寸；<=0 时取 80x24。
	Width, Height int
	// Writer 是 ANSI 输出目标；nil 时用 os.Stdout。
	Writer io.Writer
	// Terminal 是终端后端；nil 时用 term.New()（真实终端）。
	Terminal term.Terminal
	// Mouse/Paste/FocusReports 决定是否开启对应终端模式。
	Mouse        bool
	Paste        bool
	FocusReports bool
	// Setup 在 Run 接管终端后调用，用于构建控件树（拿到真实尺寸）。
	Setup func(a *App)
	// Args 透传给示例自身（惯例：os.Args[1:]）。
	Args []string
}

// withDefaults 补全零值字段。
func (c Config) withDefaults() Config {
	if c.Width <= 0 {
		c.Width = 80
	}
	if c.Height <= 0 {
		c.Height = 24
	}
	if c.Writer == nil {
		c.Writer = os.Stdout
	}
	if !c.Mouse && !c.Paste && !c.FocusReports {
		// 三者全未显式关闭时默认全开（真实终端体验）。
		c.Mouse, c.Paste, c.FocusReports = true, true, true
	}
	return c
}

// CursorRequester 是"上报硬件光标位置"的控件接口（docs/06）。
// TextInput 聚焦时实现它，把插入位换算为屏幕坐标；宿主据此定位硬件光标。
type CursorRequester interface {
	CursorScreenPos() (x, y int, ok bool)
}

// App 是应用宿主：持有控件树（Root）、双缓冲与渲染器，负责事件分发、
// 布局、绘制与终端生命周期。控件树只在 UI goroutine 访问。
type App struct {
	mu       sync.Mutex
	cfg      Config
	root     *widget.Root
	buf      *screen.Buffer
	front    *screen.Buffer
	renderer *screen.Renderer
	post     []func()
	closed   bool
	once     sync.Once
}

// New 创建 App（不打开终端；Run 时才接管）。
func New(cfg Config) *App {
	c := cfg.withDefaults()
	a := &App{
		cfg:      c,
		root:     widget.NewRoot(),
		buf:      screen.NewBuffer(c.Width, c.Height),
		renderer: screen.NewRenderer(c.Writer),
	}
	a.root.Layout(geom.Rect{X: 0, Y: 0, W: c.Width, H: c.Height})
	// 只挂主题、不重绘：构造阶段不得产生任何输出副作用
	// （终端尚未接管，见 ADR-016）。
	a.root.ApplyTheme(widget.ThemeFrom(theme.Current()))
	return a
}

// Root 返回控件树根（把它当 Host 用：PushDialog/SetOverlay 等）。
func (a *App) Root() *widget.Root { return a.root }

// SetTheme 换肤但不重绘（下一帧生效）。
func (a *App) SetTheme(th theme.Theme) {
	theme.SetDefault(th)
	a.root.ApplyTheme(widget.ThemeFrom(th))
	a.root.Invalidate()
}

// ApplyTheme 换肤并立即重绘。
func (a *App) ApplyTheme(th theme.Theme) {
	a.SetTheme(th)
	a.Draw()
}

// Buffer 返回后缓冲（测试与调试用）。
func (a *App) Buffer() *screen.Buffer { return a.buf }

// Config 返回生效配置。
func (a *App) Config() Config { return a.cfg }

// Size 返回当前缓冲尺寸。
func (a *App) Size() geom.Size { return a.buf.Size() }

// Closed 报告是否已关闭。
func (a *App) Closed() bool {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.closed
}

// Post 投递闭包到 UI 线程执行（外部 goroutine 改控件属性的唯一入口）。
func (a *App) Post(fn func()) {
	if fn == nil {
		return
	}
	a.mu.Lock()
	a.post = append(a.post, fn)
	a.mu.Unlock()
}

// Flush 执行已投递的闭包并按需重绘。
func (a *App) Flush() {
	a.mu.Lock()
	fns := a.post
	a.post = nil
	a.mu.Unlock()
	for _, fn := range fns {
		fn()
	}
	if a.root.NeedPaint() {
		a.Draw()
		a.root.Invalidate()
	}
}

// Layout 按当前尺寸重排控件树（Root 自身边界 + 子控件由调用方布局）。
func (a *App) Layout() {
	s := a.Size()
	a.root.Layout(geom.Rect{X: 0, Y: 0, W: s.W, H: s.H})
}

// Draw 清空后缓冲 → 控件树绘制 → diff 输出 → 硬件光标定位。
func (a *App) Draw() {
	a.buf.Reset()
	a.buf.ResetClip()
	a.root.Paint(a.buf)
	data := a.buf.Present(a.front)
	data = append(data, a.cursorSeq()...)
	if len(data) > 0 {
		if _, err := a.renderer.Write(data); err != nil {
			return
		}
	}
	a.front = a.buf
}

// cursorSeq 按焦点控件的请求输出硬件光标定位（docs/06）。
// 焦点控件实现 CursorRequester 且能给出位置 → 显示并定位；否则隐藏。
func (a *App) cursorSeq() []byte {
	if f := a.root.Focused(); f != nil {
		if req, ok := f.(CursorRequester); ok {
			if x, y, valid := req.CursorScreenPos(); valid {
				seq := []byte("\x1b[?25h") // 显示光标
				seq = append(seq, []byte(fmt.Sprintf("\x1b[%d;%dH", y+1, x+1))...)
				return seq
			}
		}
	}
	return []byte("\x1b[?25l") // 无请求：隐藏
}

// Dispatch 分发一个事件到控件树并触发重绘，返回是否被控件树消费。
func (a *App) Dispatch(e event.Event) bool {
	handled := false
	switch ev := e.(type) {
	case event.Resize:
		a.resize(ev.W, ev.H)
		handled = true
	default:
		handled = a.root.Dispatch(e)
	}
	a.Flush()
	return handled
}

// resize 重建双缓冲（全量重绘，docs/05）。
func (a *App) resize(w, h int) {
	if w <= 0 || h <= 0 {
		return
	}
	a.buf = screen.NewBuffer(w, h)
	a.front = nil
	a.cfg.Width, a.cfg.Height = w, h
	a.Layout()
	a.Draw()
}

// Close 释放终端（幂等）。
func (a *App) Close() error {
	var err error
	a.once.Do(func() {
		a.mu.Lock()
		a.closed = true
		t := a.cfg.Terminal
		a.mu.Unlock()
		if t != nil {
			err = t.Close()
		}
	})
	return err
}

// ErrNoTerminal 表示无法打开真实终端。
var ErrNoTerminal = errors.New("app: cannot open terminal")

// Run 接管终端并进入事件循环，直到 Quit 或终端关闭。
//
// 终端恢复遵循 AGENT.md 2.4：defer / 信号 / panic 三路径共用同一恢复逻辑。
func Run(cfg Config) (err error) {
	a := New(cfg)
	t := cfg.Terminal
	if t == nil {
		// 按配置探测能力：TERM=dumb 时禁用鼠标（docs/03 降级路径）。
		env := map[string]string{"TERM": os.Getenv("TERM"), "COLORTERM": os.Getenv("COLORTERM")}
		mouse := cfg.Mouse
		if !term.HasMouse(env) {
			mouse = false
		}
		t, err = term.New(term.Options{
			Mouse:        mouse,
			Paste:        cfg.Paste,
			FocusReports: cfg.FocusReports,
			AltScreen:    true,
		})
		if err != nil {
			return errors.Join(ErrNoTerminal, err)
		}
	}
	a.cfg.Terminal = t
	if s := t.Size(); s.W > 0 && s.H > 0 && (cfg.Width <= 0 || cfg.Height <= 0) {
		a.buf = screen.NewBuffer(s.W, s.H)
		a.cfg.Width, a.cfg.Height = s.W, s.H
		a.front = nil
	}
	a.Layout()
	a.Draw()

	if cfg.Setup != nil {
		cfg.Setup(a)
		a.Draw()
	}

	restore := func() {
		// 唯一恢复路径：幂等 Close 会还原 raw mode / 备用屏 / 光标 /
		// 鼠标 / 括号粘贴 / 焦点事件。
		_ = t.Close()
	}

	defer func() {
		if r := recover(); r != nil {
			restore()
			panic(r) // 恢复终端后再上抛
		}
		restore()
	}()

	quit := make(chan struct{})
	var closeOnce sync.Once
	stop := func() { closeOnce.Do(func() { close(quit) }) }
	signalStop(stop)

	// 输入解码：键盘/鼠标字节流 → 事件（与后端事件合并到同一通道）。
	decoded := make(chan event.Event, 256)
	dec := input.New(func(e event.Event) {
		select {
		case decoded <- e:
		default:
		}
	})
	readDone := make(chan struct{})
	go func() {
		defer close(readDone)
		readInput(t.Input(), dec)
	}()

	backend := t.Events()
	for {
		select {
		case <-quit:
			return nil
		case ev := <-decoded:
			if ev == nil {
				continue
			}
			if isCtrlC(ev) {
				return nil
			}
			handled := a.Dispatch(ev)
			if !handled && isEsc(ev) {
				return nil
			}
		case ev, ok := <-backend:
			if !ok {
				return nil
			}
			if ev == nil {
				continue
			}
			a.Dispatch(ev)
		}
	}
}

// readInput 从输入流持续读取并喂给解码器；EOF 或读错误时收尾。
// 收到裸 ESC 时启动 50ms 定时器解决歧义（docs/04）。
func readInput(r io.Reader, dec *input.Decoder) {
	buf := make([]byte, 4096)
	var escTimer *time.Timer
	for {
		n, err := r.Read(buf)
		if n > 0 {
			if escTimer != nil {
				escTimer.Stop()
			}
			dec.Feed(buf[:n])
			if dec.Pending() {
				escTimer = time.AfterFunc(50*time.Millisecond, dec.Flush)
			}
		}
		if err != nil {
			if escTimer != nil {
				escTimer.Stop()
			}
			dec.Flush()
			return
		}
	}
}

// isQuit 报告是否为退出事件（Esc 或 Ctrl+C）。
func isQuit(e event.Event) bool {
	return isEsc(e) || isCtrlC(e)
}

func isEsc(e event.Event) bool {
	k, ok := e.(event.Key)
	return ok && k.Press && k.Code == event.KeyEsc
}

func isCtrlC(e event.Event) bool {
	k, ok := e.(event.Key)
	if !ok || !k.Press {
		return false
	}
	if k.Code == event.KeyRune && k.Rune == 0x03 {
		return true
	}
	return (k.Rune == 'c' || k.Rune == 'C') && k.Mod&event.ModCtrl != 0
}
