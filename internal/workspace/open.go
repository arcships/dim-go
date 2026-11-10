package workspace

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"runtime"
)

// openCommand 是一次外部打开对应的命令。
type openCommand struct {
	name string
	args []string
}

// openCommandFor 按平台选择用系统默认处理程序打开路径的命令。
// 不支持的系统返回 ErrUnsupportedPlatform；不做终端兜底。
func openCommandFor(goos, path string) (openCommand, error) {
	switch goos {
	case "darwin":
		return openCommand{name: "open", args: []string{path}}, nil
	case "linux", "freebsd", "openbsd", "netbsd":
		return openCommand{name: "xdg-open", args: []string{path}}, nil
	case "windows":
		// start 的第一个空参数是窗口标题，避免带空格路径被当成标题。
		return openCommand{name: "cmd", args: []string{"/c", "start", "", path}}, nil
	default:
		return openCommand{}, fmt.Errorf("%w: %s", ErrUnsupportedPlatform, goos)
	}
}

// externalRunner 是可替换的外部命令执行器，测试用注入避免真的打开桌面程序。
var externalRunner = runExternalCommand

// runExternalCommand 启动子进程并返回退出码。
// 启动失败时没有退出码，返回 -1 与包装了 ErrOpenFailed 的错误。
func runExternalCommand(name string, args ...string) (int, error) {
	cmd := exec.Command(name, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	err := cmd.Run()
	if err == nil {
		return 0, nil
	}
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		code := exitErr.ExitCode()
		return code, fmt.Errorf("%w: %s 退出码 %d", ErrOpenFailed, name, code)
	}
	return -1, fmt.Errorf("%w: 无法启动 %s: %v", ErrOpenFailed, name, err)
}

// Open 用系统默认处理程序打开工作区内的文件或目录，并返回子进程退出码。
// 路径不存在返回 ErrPathNotExist，越界返回 ErrOutsideWorkspace。
// 注意：打开命令通常在启动桌面程序后立即返回，退出码属于该命令本身。
func (w *Workspace) Open(rel string) (int, error) {
	abs, err := w.Resolve(rel)
	if err != nil {
		return -1, err
	}
	if _, err := os.Stat(abs); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return -1, fmt.Errorf("%w: %s", ErrPathNotExist, rel)
		}
		return -1, fmt.Errorf("读取路径 %s: %w", rel, err)
	}
	command, err := openCommandFor(runtime.GOOS, abs)
	if err != nil {
		return -1, err
	}
	return externalRunner(command.name, command.args...)
}
