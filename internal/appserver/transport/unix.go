package transport

import (
	"context"
	"fmt"
	"net"
	"time"
)

// dialUnix 连接 Unix domain socket。
func dialUnix(ctx context.Context, address string, timeout time.Duration) (Transport, error) {
	dialer := net.Dialer{Timeout: timeout}
	conn, err := dialer.DialContext(ctx, "unix", address)
	if err != nil {
		return nil, fmt.Errorf("connect unix socket %s: %w", address, err)
	}
	return conn, nil
}
