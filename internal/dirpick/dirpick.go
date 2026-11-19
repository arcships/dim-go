// Package dirpick 取一个目录路径：macOS 上用系统的「选择目录」对话框，其他平台
// 明确回答"没有"。
//
// 它只做一件事：把宿主能力（osascript 的 choose folder）包成一个可注入执行器的
// 函数，返回选中目录的绝对路径。调用方据此决定界面回退（非 macOS 手输路径）与
// 失败原因。取消不是失败：返回 ErrCancelled，调用方什么都不改。
//
// 不读 App Server 的任何事实，也不落盘：路径归调用方的偏好通道。
package dirpick

import (
	"context"
	"errors"
	"fmt"
	"os/exec"
	"strings"
)

// ErrCancelled 表示用户取消了选择：调用方不改状态，也不报错。
var ErrCancelled = errors.New("已取消选择")

// ErrUnsupported 表示当前平台没有系统目录选择器：调用方回退为手输路径。
var ErrUnsupported = errors.New("当前平台没有系统目录选择器")

// Runner 执行一次外部命令，返回合并后的标准输出与标准错误。
//
// 它可注入：测试用它返回固定输出与错误，不真的弹窗。
type Runner func(ctx context.Context, name string, args ...string) (string, error)

// Run 是 Runner 的真实实现：拉起命令并等它结束。
func Run(ctx context.Context, name string, args ...string) (string, error) {
	out, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return string(out), err
}

// darwinScript 是 macOS 的选择目录脚本：POSIX path 给出 '/' 分隔的绝对路径。
// 提示语是界面语，与界面的其余文案同为中文。
const darwinScript = `POSIX path of (choose folder with prompt "选择目录")`

// darwinCancelMarks 是 osascript 在对话框被取消时写进输出的标记（含 -128 退出码
// 文案与中文系统下的「用户已取消」）。
var darwinCancelMarks = []string{"(-128)", "User canceled", "用户已取消"}

// Choose 弹系统目录选择器并返回选中的目录（去掉末尾的分隔符）。
//
// 返回：
//   - 路径 + nil：选好了；
//   - ErrCancelled：用户取消（调用方不改状态、不报错）；
//   - ErrUnsupported：当前平台没有系统选择器（goos 不是 darwin）；
//   - 其他错误：一句可读原因（无权限、命令不存在、超时…）。
func Choose(ctx context.Context, run Runner, goos string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if goos != "darwin" {
		return "", ErrUnsupported
	}
	if run == nil {
		run = Run
	}
	out, err := run(ctx, "osascript", "-e", darwinScript)
	text := strings.TrimSpace(out)
	if err != nil {
		if ctx.Err() != nil || errors.Is(err, context.DeadlineExceeded) ||
			errors.Is(err, context.Canceled) {
			return "", errors.New("选择目录超时，请重试")
		}
		if isCancel(text) {
			return "", ErrCancelled
		}
		return "", fmt.Errorf("系统目录选择器运行失败：%s", commandReason(text, err))
	}
	// 退出码为 0 但没有输出：同样按取消处理（脚本没给出路径）。
	if text == "" {
		return "", ErrCancelled
	}
	return normalizePath(text), nil
}

// isCancel 判输出是不是"用户取消"。
func isCancel(out string) bool {
	for _, mark := range darwinCancelMarks {
		if strings.Contains(out, mark) {
			return true
		}
	}
	return false
}

// commandReason 取一句原因：优先命令自己的第一行非空输出，其次错误文本。
func commandReason(out string, err error) string {
	for _, line := range strings.Split(out, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	if err != nil {
		return err.Error()
	}
	return "未知原因"
}

// normalizePath 去掉末尾的分隔符（根目录 '/' 保留）。
func normalizePath(path string) string {
	for len(path) > 1 && strings.HasSuffix(path, "/") {
		path = strings.TrimSuffix(path, "/")
	}
	return path
}
