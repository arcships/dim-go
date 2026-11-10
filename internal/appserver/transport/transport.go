// Package transport 提供连接领域唯一的本地传输入口。
//
// 当前实现是 Unix domain socket；Windows 命名管道先留接口，调用会返回
// ErrUnsupportedPlatform，不假装可用。
package transport

import (
	"context"
	"errors"
	"fmt"
	"io"
	"time"
)

// Transport 是一条已经建立的本地连接。调用方只用它收发字节与关闭。
type Transport interface {
	io.ReadWriteCloser
}

// 端点网络类型。
const (
	NetworkUnix = "unix"
	NetworkPipe = "pipe"
)

// Endpoint 是本地端点：Unix socket 路径或 Windows 命名管道名。
type Endpoint struct {
	Network string
	Address string
}

func (e Endpoint) String() string { return e.Network + ":" + e.Address }

// ErrUnsupportedPlatform 表示该端点类型在当前平台没有实现。
var ErrUnsupportedPlatform = errors.New("local transport is not supported on this platform")

// Dial 按端点类型连接本地传输。timeout 为 0 时使用平台默认连接超时。
func Dial(ctx context.Context, endpoint Endpoint, timeout time.Duration) (Transport, error) {
	switch endpoint.Network {
	case NetworkUnix:
		return dialUnix(ctx, endpoint.Address, timeout)
	case NetworkPipe:
		return dialPipe(ctx, endpoint.Address, timeout)
	default:
		return nil, fmt.Errorf("unknown transport network %q", endpoint.Network)
	}
}
