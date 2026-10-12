//go:build linux

package term

import "golang.org/x/sys/unix"

// linux 的 termios ioctl 请求码（docs/03：linux 用 TCGETS/TCSETS）。
const (
	ioctlRead  = unix.TCGETS
	ioctlWrite = unix.TCSETS
)
