package screen

import (
	"io"
)

// Renderer 是负责将 Buffer 的 diff 输出为最小 ANSI 序列，并写入终端 Writer。
type Renderer struct {
	writer io.Writer
}

// NewRenderer 创建一个新的 Renderer，使用 writer 输出 ANSI 序列。
func NewRenderer(writer io.Writer) *Renderer {
	return &Renderer{writer: writer}
}

// Present 计算 buffer 与 old buffer 的 diff，并输出最小 ANSI 序列到 writer。
// 返回本次呈现输出的字节数（可能为 0，表示无变化）。
func (r *Renderer) Present(buffer *Buffer, old *Buffer) int {
	data := buffer.Present(old)
	if len(data) == 0 {
		return 0
	}
	n, err := r.writer.Write(data)
	if err != nil {
		return 0
	}
	return n
}

// Write 写入原始 ANSI 序列到终端。
func (r *Renderer) Write(data []byte) (int, error) {
	return r.writer.Write(data)
}
