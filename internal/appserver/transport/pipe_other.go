//go:build !windows

package transport

import (
	"context"
	"time"
)

// dialPipe 在非 Windows 平台没有实现。Windows 命名管道按后续项处理。
func dialPipe(_ context.Context, _ string, _ time.Duration) (Transport, error) {
	return nil, ErrUnsupportedPlatform
}
