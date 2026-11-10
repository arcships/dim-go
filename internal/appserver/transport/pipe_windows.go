//go:build windows

package transport

import (
	"context"
	"time"
)

// dialPipe 是 Windows 命名管道的接口占位。
//
// 后续实现需要直接用 Win32 命名管道句柄打开 `\\.\pipe\...`，或使用 App Server
// 的 `--connect-pipe` 桥接经 stdio 传输。两种方式都必须保留 Transport 接口，
// 让连接领域之外没有平台分支。
func dialPipe(_ context.Context, _ string, _ time.Duration) (Transport, error) {
	return nil, ErrUnsupportedPlatform
}
