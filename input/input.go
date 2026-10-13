package input

import (
	"bytes"
	"strconv"
	"strings"
	"sync"

	"PaintedSkin/event"
)

// 缓冲区与序列最大长度限制（防 DoS / OOM）。
const (
	// MaxBufSize 是解码器缓冲区的最大字节数上限。
	MaxBufSize = 64 * 1024
	// MaxCsiParamsLen 是 CSI 参数的最大容忍长度。
	MaxCsiParamsLen = 64
	// MaxOscLen 是 OSC / DCS 序列的最大容忍长度。
	MaxOscLen = 4096
	// MaxPasteLen 是括号粘贴单次最大容忍长度。
	MaxPasteLen = 64 * 1024
)

// Decoder 将终端字节流增量解码为结构化的 event.Event。
type Decoder struct {
	mu       sync.Mutex
	callback func(event.Event)

	// buf 缓冲区：保存尚未处理的字节（跨 Feed 分片）
	buf []byte

	// flushCount 记录连续 Flush 次数，用于超时丢弃不完整或损坏序列
	flushCount int
}

// New 创建一个新的 Decoder，cb 接收解码出的事件。
func New(cb func(event.Event)) *Decoder {
	return &Decoder{callback: cb, buf: nil}
}

// Pending 返回是否有未完成的字节序列（跨 read 分片需要等待下一次 Feed）。
func (d *Decoder) Pending() bool {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.buf) > 0
}

// BufferLen 返回当前缓冲区未消费的字节长度（供测试与诊断使用）。
func (d *Decoder) BufferLen() int {
	d.mu.Lock()
	defer d.mu.Unlock()
	return len(d.buf)
}

// Flush 处理输入超时。
// 裸 ESC（ESC 歧义超时）结束并视为 KeyEsc。
// 若连续多次 Flush（超时且无新字节到达）仍存在无法解析的不完整序列（如半截 CSI/OSC/DCS/未闭合粘贴），
// 强制丢弃损坏序列头部以恢复输入状态机，防止状态机永久挂死。
func (d *Decoder) Flush() {
	d.mu.Lock()
	defer d.mu.Unlock()

	if len(d.buf) == 0 {
		d.flushCount = 0
		return
	}

	if len(d.buf) == 1 && d.buf[0] == 0x1B {
		d.buf = d.buf[:0]
		d.flushCount = 0
		d.emit(event.Key{Code: event.KeyEsc, Mod: event.ModNone, Press: true})
		return
	}

	d.flushCount++
	// 连续第二次 Flush（无新数据进入且仍处于 pending）说明序列被截断或损坏
	if d.flushCount >= 2 {
		d.flushCount = 0
		// 丢弃损坏且超时的未完成序列，恢复输入状态机
		d.buf = d.buf[:0]
	}
}

// emit 通过回调产出一个事件。
func (d *Decoder) emit(ev event.Event) {
	if d.callback != nil {
		d.callback(ev)
	}
}

// Feed 输入一批字节，按状态机解码并通过 callback 回调回收事件。
// 返回已消费的字节数（可能小于 len(b)，表示剩余字节需要下次 Feed）。
func (d *Decoder) Feed(b []byte) int {
	d.mu.Lock()
	defer d.mu.Unlock()

	d.flushCount = 0

	// 缓冲区上限防护（防 DoS / OOM）：若超过容量上限，强制丢弃最早的无法解析的数据
	if len(d.buf)+len(b) > MaxBufSize {
		overflow := (len(d.buf) + len(b)) - MaxBufSize
		if overflow >= len(d.buf) {
			d.buf = d.buf[:0]
		} else {
			d.buf = d.buf[overflow:]
		}
	}

	// 将新字节追加到缓冲区
	d.buf = append(d.buf, b...)

	// 循环处理缓冲区中的所有完整事件
	consumed := 0
	for len(d.buf) > 0 {
		n := d.tryDecodeOne()
		if n <= 0 {
			// 无法解码一个完整字节/序列，保留当前字节待下次 Feed
			break
		}
		consumed += n
		// 移除已消费的字节
		d.buf = d.buf[n:]
	}

	return consumed
}

// tryDecodeOne 尝试从缓冲区解码一个事件。
// 返回已消费的字节数（1 表示单字节，>1 表示序列）。
// 如果返回 0，表示需要更多字节（pending 状态）。
// 注意：tryDecodeOne 返回消耗的字节数，由 Feed 循环根据返回值来 slice d.buf。
func (d *Decoder) tryDecodeOne() int {
	if len(d.buf) == 0 {
		return 0
	}

	first := d.buf[0]

	// 情况 1: ESC 序列起始 - MUST check this BEFORE printable range
	if first == 0x1B {
		return d.tryDecodeEsc()
	}

	// 情况 2: UTF-8 多字节字符处理
	if first >= 0xC0 && first <= 0xF7 {
		// 确定所需的 continuation byte 数
		var need int
		switch {
		case first >= 0xF0:
			need = 3 // 4-byte sequence
		case first >= 0xE0:
			need = 2 // 3-byte sequence
		default:
			need = 1 // 2-byte sequence
		}

		if len(d.buf) < need+1 {
			// 包括 leading byte 在内总共 need+1 个字节不足
			return 0 // pending
		}

		// 校验 continuation byte，损坏则吞掉 leading byte
		for i := 1; i <= need; i++ {
			if d.buf[i] < 0x80 || d.buf[i] > 0xBF {
				return 1
			}
		}

		// 有足够的字节，解码 rune
		r := d.decodeRune(d.buf[:need+1])
		d.emit(event.Key{
			Code:  event.KeyRune,
			Rune:  r,
			Mod:   event.ModNone,
			Press: true,
		})
		return need + 1 // 返回需要消费的字节数
	}

	// 情况 3: 可打印 ASCII (0x20-0x7E)
	if first >= 0x20 && first <= 0x7E {
		// 空格键 (0x20) 使用 KeySpace 编码，便于与解码出的空格字节直接比较
		code := event.KeyRune
		if first == 0x20 {
			code = event.KeySpace
		}
		d.emit(event.Key{
			Code:  code,
			Rune:  rune(first),
			Mod:   event.ModNone,
			Press: true,
		})
		return 1
	}

	// 情况 4: 控制字符 (0x00-0x1F, 0x7F)
	if ev, ok := controlKey(first, event.ModNone); ok {
		d.emit(ev)
		return 1
	}

	// 未知字节（孤立 continuation 0x80-0xBF、非法 0xF8-0xFF）：吞掉
	return 1
}

// tryDecodeEsc 解码以 ESC 开头的序列。
// 裸 ESC（缓冲区不足）返回 0 进入 pending，由 Flush 超时结束为 KeyEsc。
func (d *Decoder) tryDecodeEsc() int {
	if len(d.buf) < 2 {
		return 0 // 需要更多字节
	}

	second := d.buf[1]

	switch second {
	case 'O':
		// SS3 序列 (ESC O x)
		if len(d.buf) < 3 {
			return 0 // pending，需要更多字节
		}
		if ev, ok := ss3Key(d.buf[2]); ok {
			d.emit(ev)
		}
		// 未知 SS3 整体吞掉
		return 3
	case '[':
		// CSI 序列 (ESC [ ...)
		return d.tryDecodeCsi()
	case ']':
		// OSC 序列 (ESC ] ... BEL/ST)，内容忽略
		return d.tryDecodeOsc()
	case 'P':
		// DCS 序列 (ESC P ... ST)，内容忽略
		return d.tryDecodeDcs()
	case 0x1B:
		// 连按 ESC：第一枚立即解析为 KeyEsc，第二枚保持 pending 待 flush
		d.emit(event.Key{Code: event.KeyEsc, Mod: event.ModNone, Press: true})
		return 1
	}

	// ESC + 可打印字符 -> Alt+该键 (ESC a -> Alt+a)
	if second >= 0x20 && second <= 0x7E {
		code := event.KeyRune
		var r rune
		switch second {
		case ' ':
			code = event.KeySpace
			r = ' '
		case '\r', '\n':
			code = event.KeyEnter
		case '\t':
			code = event.KeyTab
		default:
			r = rune(second)
		}
		d.emit(event.Key{Code: code, Rune: r, Mod: event.ModAlt, Press: true})
		return 2
	}

	// ESC + 控制字节 -> Alt+控制键 (如 ESC+\r -> Alt+Enter)
	if second < 0x20 || second == 0x7F {
		if ev, ok := controlKey(second, event.ModAlt); ok {
			d.emit(ev)
		}
		return 2
	}

	// ESC + 高字节：尝试按 Alt+UTF-8 解码
	if second >= 0x80 {
		var need int
		switch {
		case second >= 0xF0 && second <= 0xF7:
			need = 3
		case second >= 0xE0 && second <= 0xEF:
			need = 2
		case second >= 0xC0 && second <= 0xDF:
			need = 1
		default:
			return 2 // 孤立字节：吞掉 ESC+该字节
		}
		if len(d.buf) < 2+need {
			return 0 // pending
		}
		for i := 2; i < 2+need; i++ {
			if d.buf[i] < 0x80 || d.buf[i] > 0xBF {
				return 2 // 损坏：吞掉 ESC+leading
			}
		}
		d.emit(event.Key{
			Code:  event.KeyRune,
			Rune:  d.decodeRune(d.buf[1 : 2+need]),
			Mod:   event.ModAlt,
			Press: true,
		})
		return 2 + need
	}

	return 1
}

// controlKey 把控制字节映射为按键事件，mod 叠加到结果上（如 Alt）。
func controlKey(b byte, mod event.Mod) (event.Key, bool) {
	withMod := func(m event.Mod) event.Mod { return m | mod }
	switch b {
	case '\r', '\n':
		return event.Key{Code: event.KeyEnter, Mod: withMod(event.ModNone), Press: true}, true
	case '\t':
		return event.Key{Code: event.KeyTab, Mod: withMod(event.ModNone), Press: true}, true
	case 0x08, 0x7F: // Backspace (BS / DEL)
		return event.Key{Code: event.KeyBackspace, Mod: withMod(event.ModNone), Press: true}, true
	case 0x00: // Ctrl+Space / NUL
		return event.Key{Code: event.KeySpace, Rune: ' ', Mod: withMod(event.ModCtrl), Press: true}, true
	case 0x1C:
		return event.Key{Code: event.KeyRune, Rune: '\\', Mod: withMod(event.ModCtrl), Press: true}, true
	case 0x1D:
		return event.Key{Code: event.KeyRune, Rune: ']', Mod: withMod(event.ModCtrl), Press: true}, true
	case 0x1E:
		return event.Key{Code: event.KeyRune, Rune: '^', Mod: withMod(event.ModCtrl), Press: true}, true
	case 0x1F:
		return event.Key{Code: event.KeyRune, Rune: '_', Mod: withMod(event.ModCtrl), Press: true}, true
	}
	if b >= 0x01 && b <= 0x1A {
		return event.Key{
			Code:  event.KeyRune,
			Rune:  rune('a' + b - 1),
			Mod:   withMod(event.ModCtrl),
			Press: true,
		}, true
	}
	return event.Key{}, false
}

// ss3Key 解码 SS3 第三字节（ESC O x）。
func ss3Key(b byte) (event.Key, bool) {
	key := func(code event.KeyCode) event.Key {
		return event.Key{Code: code, Mod: event.ModNone, Press: true}
	}
	switch b {
	case 'P':
		return key(event.KeyF1), true
	case 'Q':
		return key(event.KeyF2), true
	case 'R':
		return key(event.KeyF3), true
	case 'S':
		return key(event.KeyF4), true
	case 'A':
		return key(event.KeyUp), true
	case 'B':
		return key(event.KeyDown), true
	case 'C':
		return key(event.KeyRight), true
	case 'D':
		return key(event.KeyLeft), true
	case 'H':
		return key(event.KeyHome), true
	case 'F':
		return key(event.KeyEnd), true
	}
	return event.Key{}, false
}

// csiMod 把 CSI 修饰参数（1=无，2=Shift，3=Alt，4=Shift+Alt，
// 5=Ctrl，6=Shift+Ctrl，7=Alt+Ctrl，8=全按）译为 Mod 掩码。
func csiMod(m int) event.Mod {
	if m <= 1 {
		return event.ModNone
	}
	m--
	var mod event.Mod
	if m&1 != 0 {
		mod |= event.ModShift
	}
	if m&2 != 0 {
		mod |= event.ModAlt
	}
	if m&4 != 0 {
		mod |= event.ModCtrl
	}
	return mod
}

// splitCsiParams 把 "1;5" 切分为 [1 5]，空串返回空切片。
func splitCsiParams(params string) []int {
	if params == "" {
		return nil
	}
	var out []int
	for _, part := range strings.Split(params, ";") {
		if part == "" || part == "?" {
			out = append(out, 0)
			continue
		}
		// 跳过 '?' 等私有前缀残留，只取数字
		digits := strings.TrimLeft(part, "?")
		n, err := strconv.Atoi(digits)
		if err != nil {
			return nil
		}
		out = append(out, n)
	}
	return out
}

// tryDecodeCsi 处理 CSI 序列 ESC [ 参数 最终字节。
// 未知序列整体吞掉（无事件），不完整返回 0 pending。
func (d *Decoder) tryDecodeCsi() int {
	if len(d.buf) < 3 {
		return 0 // ESC [ 之后等最终字节（分片到达前保持 pending）
	}
	if d.buf[1] != '[' {
		return 1 // 无效序列，消耗 ESC
	}

	rest := d.buf[2:]

	// SGR 鼠标：ESC [ < b ; x ; y M/m
	if len(rest) >= 1 && rest[0] == '<' {
		return d.tryDecodeMouse()
	}

	// 括号粘贴起始：ESC [ 200 ~
	if len(rest) < 4 && strings.HasPrefix("200~", string(rest)) {
		return 0 // 起始符自身被拆开，等待
	}
	if len(rest) >= 4 && string(rest[:4]) == "200~" {
		return d.tryDecodePaste()
	}

	paramEnd := 0
	for paramEnd < len(rest) && rest[paramEnd] >= 0x30 && rest[paramEnd] <= 0x3F {
		paramEnd++
	}

	if paramEnd >= len(rest) {
		if len(rest) > MaxCsiParamsLen {
			return 1 // 参数超长异常，消耗 ESC 避免无限堆积挂起
		}
		return 0 // pending，需要更多字节（最终字节）
	}

	final := rest[paramEnd]
	if final < 0x40 || final > 0x7E {
		return 1 // 无效最终字节，消耗 ESC，由后续 Feed 继续
	}

	params := string(rest[:paramEnd])
	if ev := decodeCsiFinal(final, params); ev != nil {
		d.emit(ev)
	}
	// 未知序列：吞掉整个序列，不产出事件

	return 2 + paramEnd + 1 // ESC + [ + params + final
}

// decodeCsiFinal 根据 CSI 最终字符解码事件，未知返回 nil（吞掉）。
func decodeCsiFinal(final byte, params string) event.Event {
	// 私有参数（?25l / ?1006h / ?2004h 等终端模式应答）：吞掉
	if strings.HasPrefix(params, "?") {
		return nil
	}

	key := func(code event.KeyCode, mod event.Mod) event.Key {
		return event.Key{Code: code, Mod: mod, Press: true}
	}

	// 焦点事件：CSI I / CSI O
	if params == "" || params == "0" {
		switch final {
		case 'I':
			return event.Focus{Gained: true}
		case 'O':
			return event.Focus{Gained: false}
		}
	}

	// 修饰解析："" / "1" / "1;mod"
	modCode := 1
	if params != "" && params != "1" {
		parts := splitCsiParams(params)
		if parts == nil {
			return nil
		}
		if len(parts) == 2 {
			modCode = parts[1]
		} else {
			// "5" 这类单数字对方向键无意义；波浪号键走 '~' 分支，
			// 其余视为未知
			if final != '~' {
				return nil
			}
		}
	}

	switch final {
	case 'A': // 上箭头
		return key(event.KeyUp, csiMod(modCode))
	case 'B': // 下箭头
		return key(event.KeyDown, csiMod(modCode))
	case 'C': // 右箭头
		return key(event.KeyRight, csiMod(modCode))
	case 'D': // 左箭头
		return key(event.KeyLeft, csiMod(modCode))
	case 'H': // Home
		return key(event.KeyHome, csiMod(modCode))
	case 'F': // End
		return key(event.KeyEnd, csiMod(modCode))
	case 'Z': // back-tab
		return key(event.KeyTab, event.ModShift|csiMod(modCode))
	case '~': // 带数字参数的功能键
		parts := splitCsiParams(params)
		if len(parts) == 0 {
			return nil
		}
		num := parts[0]
		mod := event.ModNone
		if len(parts) >= 2 {
			mod = csiMod(parts[1])
		}
		switch num {
		case 1:
			return key(event.KeyHome, mod)
		case 2:
			return key(event.KeyInsert, mod)
		case 3:
			return key(event.KeyDelete, mod)
		case 4:
			return key(event.KeyEnd, mod)
		case 5:
			return key(event.KeyPgUp, mod)
		case 6:
			return key(event.KeyPgDn, mod)
		case 15:
			return key(event.KeyF5, mod)
		case 17:
			return key(event.KeyF6, mod)
		case 18:
			return key(event.KeyF7, mod)
		case 19:
			return key(event.KeyF8, mod)
		case 20:
			return key(event.KeyF9, mod)
		case 21:
			return key(event.KeyF10, mod)
		case 23:
			return key(event.KeyF11, mod)
		case 24:
			return key(event.KeyF12, mod)
		}
		return nil
	}

	return nil
}

// tryDecodeMouse 处理 SGR 鼠标序列 ESC [ < b ; x ; y M/m。
func (d *Decoder) tryDecodeMouse() int {
	rest := d.buf[2:] // 以 '<' 开头

	// 找最终字节 M/m：只允许数字与 ';' 出现在中间
	i := 1
	for i < len(rest) && ((rest[i] >= '0' && rest[i] <= '9') || rest[i] == ';') {
		i++
	}
	if i >= len(rest) {
		return 0 // pending：等最终字节
	}
	final := rest[i]
	if final != 'M' && final != 'm' {
		// 非法 SGR 鼠标：吞掉 ESC（后续字节由状态机继续处理）
		return 1
	}

	fields := strings.Split(string(rest[1:i]), ";")
	if len(fields) != 3 {
		return 2 + i + 1 // 畸形：吞掉整个序列
	}
	b, err1 := strconv.Atoi(fields[0])
	x, err2 := strconv.Atoi(fields[1])
	y, err3 := strconv.Atoi(fields[2])
	if err1 != nil || err2 != nil || err3 != nil {
		return 2 + i + 1
	}

	var mod event.Mod
	if b&4 != 0 {
		mod |= event.ModShift
	}
	if b&8 != 0 {
		mod |= event.ModAlt
	}
	if b&16 != 0 {
		mod |= event.ModCtrl
	}

	button := func() event.MouseButton {
		switch b & 3 {
		case 0:
			return event.ButtonLeft
		case 1:
			return event.ButtonMiddle
		case 2:
			return event.ButtonRight
		}
		return event.ButtonNone
	}

	m := event.Mouse{X: x - 1, Y: y - 1, Mod: mod} // 终端坐标 1-based，转 0-based
	switch {
	case b&64 != 0: // 滚轮：64 上，65 下
		if b&1 == 0 {
			m.Kind = event.MouseWheelUp
		} else {
			m.Kind = event.MouseWheelDown
		}
		m.Button = event.ButtonNone
	case final == 'm': // 释放
		m.Kind = event.MouseRelease
		m.Button = button()
	case b&32 != 0 || b&3 == 3: // 移动/悬停（1002/1003 语义）
		m.Kind = event.MouseMove
		m.Button = button()
		m.Drag = m.Button != event.ButtonNone
	default: // 按下
		m.Kind = event.MousePress
		m.Button = button()
	}

	d.emit(m)
	return 2 + i + 1 // ESC + [ + "<params" + final
}

// pasteTerminator 是括号粘贴结束符。
var pasteTerminator = []byte{0x1B, '[', '2', '0', '1', '~'}

// tryDecodePaste 处理 ESC [ 200 ~ ... ESC [ 201 ~，内容可跨分片、
// 可内嵌 ESC 序列（当作字面文本）。
func (d *Decoder) tryDecodePaste() int {
	const startLen = 2 + 4 // ESC [ 200 ~
	if len(d.buf) < startLen {
		return 0
	}
	idx := bytes.Index(d.buf[startLen:], pasteTerminator)
	if idx < 0 {
		if len(d.buf) >= MaxPasteLen {
			// 超长粘贴且未收到结束符，强制截断并分发当前已接收内容，退出粘贴状态
			text := string(d.buf[startLen:])
			d.emit(event.Paste{Text: text})
			return len(d.buf)
		}
		return 0 // 结束符还没收齐，pending
	}
	text := string(d.buf[startLen : startLen+idx])
	d.emit(event.Paste{Text: text})
	return startLen + idx + len(pasteTerminator)
}

// tryDecodeOsc 处理 OSC 序列 ESC ] ... (BEL | ST)，内容忽略。
func (d *Decoder) tryDecodeOsc() int {
	if len(d.buf) > MaxOscLen {
		return 1 // 超长 OSC 序列，丢弃 ESC 避免状态机挂死
	}
	for i := 2; i < len(d.buf); i++ {
		if d.buf[i] == 0x07 { // BEL
			return i + 1
		}
		if d.buf[i] == 0x1B {
			if i+1 >= len(d.buf) {
				return 0 // ST 可能被拆开，pending
			}
			if d.buf[i+1] == '\\' { // ST
				return i + 2
			}
		}
	}
	return 0 // 结束符还没到，pending
}

// tryDecodeDcs 处理 DCS 序列 ESC P ... ST，内容忽略。
func (d *Decoder) tryDecodeDcs() int {
	if len(d.buf) > MaxOscLen {
		return 1 // 超长 DCS 序列，丢弃 ESC 避免状态机挂死
	}
	for i := 2; i < len(d.buf); i++ {
		if d.buf[i] == 0x1B {
			if i+1 >= len(d.buf) {
				return 0 // ST 可能被拆开，pending
			}
			if d.buf[i+1] == '\\' { // ST
				return i + 2
			}
		}
		if d.buf[i] == 0x07 { // 容忍 BEL 结尾
			return i + 1
		}
	}
	return 0 // 结束符还没到，pending
}

// decodeRune 从 2-4 个字节解码 UTF-8 rune（调用方已校验长度与 continuation）。
func (d *Decoder) decodeRune(buf []byte) rune {
	var r rune
	switch len(buf) {
	case 4:
		// 4-byte: 0xF0-0xF7, 0x80-0xBF ×3
		r = rune(buf[0]&0x07)<<18 | rune(buf[1]&0x3F)<<12 | rune(buf[2]&0x3F)<<6 | rune(buf[3]&0x3F)
	case 3:
		// 3-byte: 0xE0-0xEF, 0x80-0xBF ×2
		r = rune(buf[0]&0x0F)<<12 | rune(buf[1]&0x3F)<<6 | rune(buf[2]&0x3F)
	case 2:
		// 2-byte: 0xC0-0xDF, 0x80-0xBF
		r = rune(buf[0]&0x1F)<<6 | rune(buf[1]&0x3F)
	case 1:
		r = rune(buf[0])
	}
	return r
}
