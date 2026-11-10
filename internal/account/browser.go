package account

import (
	"fmt"
	"os/exec"
	"runtime"
)

// systemBrowser 用系统默认处理程序打开授权地址。它只负责启动，不等待浏览器退出；
// 浏览器关掉之后登录仍可能完成，退出码也不改变登录状态。
type systemBrowser struct{}

// Open 按平台启动系统浏览器。失败只说明没能拉起处理程序，不代表授权被拒绝。
func (systemBrowser) Open(url string) error {
	var command *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		command = exec.Command("open", url)
	case "windows":
		command = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		command = exec.Command("xdg-open", url)
	}
	if err := command.Start(); err != nil {
		return fmt.Errorf("%w: %v", ErrBrowserOpen, err)
	}
	// 不等待退出：浏览器可能长期驻留；也没有需要从它读取的输出。
	go func() { _ = command.Wait() }()
	return nil
}
