// Package examples 提供示例共用的启动与降级处理。
package examples

import (
	"errors"
	"fmt"
	"os"

	"PaintedSkin/app"
)

// Run 启动示例；无 TTY 时给出可读提示而不是 panic。
//
//	examples.Run(paintedskin.Column(...))
func Run(setup func(a *app.App), argv ...string) {
	if err := app.Run(app.Config{Setup: setup, Args: argv}); err != nil {
		if errors.Is(err, app.ErrNoTerminal) {
			fmt.Fprintln(os.Stderr, "提示：当前环境没有可用的终端（TTY）。")
			fmt.Fprintln(os.Stderr, "      请在真实终端里运行，例如：go run ./examples/minimal")
			fmt.Fprintln(os.Stderr, "      或用无头渲染查看画面：go run ./examples/preview")
			return
		}
		fmt.Fprintln(os.Stderr, "运行出错:", err)
		os.Exit(1)
	}
}
