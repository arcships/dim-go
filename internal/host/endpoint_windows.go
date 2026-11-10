//go:build windows

package host

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"
)

// instanceFileName 是 Windows 上实例端点文件的文件名。
const instanceFileName = "instance.port"

// acquireLocal 在 Windows 上没有可原子抢占的本地套接字，改用回环 TCP：
// 端口写进端点文件。先探测已有实例；进程被强杀留下的端口文件在读不到响应时清理。
// 代价是两个同时启动的进程之间存在竞态窗口，本阶段未在 Windows 验证。
func acquireLocal(path string, probeTimeout time.Duration) (net.Listener, bool, error) {
	if conn, err := dialLocal(path, probeTimeout); err == nil {
		_ = conn.Close()
		return nil, false, fmt.Errorf("%w: %s", ErrInstanceRunning, path)
	}
	stale := false
	if _, err := os.Stat(path); err == nil {
		stale = true
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, false, fmt.Errorf("host: 清理实例端点失败: %w", err)
		}
	}
	ln, err := listenLocal(path)
	if err != nil {
		return nil, false, fmt.Errorf("host: 占用实例端点失败: %w", err)
	}
	return ln, stale, nil
}

func listenLocal(path string) (net.Listener, error) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	addr, ok := ln.Addr().(*net.TCPAddr)
	if !ok {
		_ = ln.Close()
		return nil, errors.New("host: 回环监听地址类型不符")
	}
	if err := os.WriteFile(path, []byte(strconv.Itoa(addr.Port)), 0o600); err != nil {
		_ = ln.Close()
		return nil, err
	}
	return ln, nil
}

func dialLocal(path string, timeout time.Duration) (net.Conn, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	port, err := strconv.Atoi(strings.TrimSpace(string(raw)))
	if err != nil || port <= 0 || port > 65535 {
		return nil, fmt.Errorf("host: 端点文件内容不是合法端口: %s", path)
	}
	return net.DialTimeout("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)), timeout)
}
