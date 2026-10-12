//go:build darwin

package term

import "golang.org/x/sys/unix"

// darwin 的 termios ioctl 请求码（docs/03：darwin 用 TIOCGETA/TIOCSETA）。
const (
	ioctlRead  = unix.TIOCGETA
	ioctlWrite = unix.TIOCSETA
)
