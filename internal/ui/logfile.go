package ui

// 本文件只做一件事：把客户端日志文件交给系统默认处理程序打开。
//
// 为什么在视图里做：断线卡的「查看日志」是本地动作——它不读 App Server 的
// 任何事实，也不经过 Intents 的写路径；日志路径由入口（`cmd/dimagent-ui`）
// 在启动时建好日志后通过 `Options.LogPath` 交过来，视图不自己推导数据目录，
// 免得与入口对日志位置的判断分叉。
//
// 平台约定与 `internal/workspace` 的外部打开一致（darwin 的 open、Linux 系
// 的 xdg-open、Windows 的 start）；不支持的系统返回原因，由调用方把路径留在
// 原位供复制，不静默失败。

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"
)

// openLogCommand 按平台给出打开路径的命令；不支持的系统返回错误。
func openLogCommand(goos, path string) (string, []string, error) {
	switch goos {
	case "darwin":
		return "open", []string{path}, nil
	case "linux", "freebsd", "openbsd", "netbsd":
		return "xdg-open", []string{path}, nil
	case "windows":
		// start 的第一个空参数是窗口标题，避免带空格的路径被当成标题。
		return "cmd", []string{"/c", "start", "", path}, nil
	default:
		return "", nil, fmt.Errorf("不支持的平台：%s", goos)
	}
}

// logOpenRunner 是可替换的外部打开执行器：测试注入，避免真的拉起桌面程序。
var logOpenRunner = runOpenCommand

// runOpenCommand 启动打开命令；打开命令通常在拉起桌面程序后立即返回。
func runOpenCommand(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("无法启动 %s：%w", name, err)
	}
	return nil
}

// openWithSystem 用系统默认处理程序打开一个已存在的路径。
func openWithSystem(path string) error {
	name, args, err := openLogCommand(runtime.GOOS, path)
	if err != nil {
		return err
	}
	return logOpenRunner(name, args...)
}
