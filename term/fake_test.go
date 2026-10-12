package term

import (
	"bytes"
)

// newFakeBase 构造只依赖共享 base 的假终端状态：注入内存后端验证 Enter/Leave
// 序列与 Close 幂等，全程不触碰真实 tty。这是终端层测试的**默认且唯一**路径，
// 沙箱与真机行为一致，不依赖运行环境是否提供 PTY。
func newFakeBase(out *bytes.Buffer, o Options) *base {
	return &base{opts: o, out: out}
}
