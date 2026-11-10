//go:build !windows

package host

import (
	"errors"
	"fmt"
	"io/fs"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"
)

// instanceFileName 是 Unix 上实例端点的文件名。
const instanceFileName = "instance.sock"

// tempSeq 给同一进程内的并发抢占生成不同的临时文件名。
var tempSeq atomic.Uint64

// acquireLocal 用 Unix 域套接字做实例互斥。
//
// 关键点：套接字先以临时名监听，完全就绪后用硬链接原子发布到端点路径。
// 端点路径一旦出现就一定是可以连上的实例，因此"能连上即占用、连不上即残留"的
// 判定没有 bind 与 listen 之间的空窗；被强杀留下的套接字能被安全清理。
// 端点被占用时返回 ErrInstanceRunning。
func acquireLocal(path string, probeTimeout time.Duration) (net.Listener, bool, error) {
	const attempts = 4
	stale := false
	var lastErr error

	for attempt := 0; attempt < attempts; attempt++ {
		ln, temp, err := listenTemp(path)
		if err != nil {
			return nil, false, fmt.Errorf("host: 创建实例端点失败: %w", err)
		}
		if unixListener, ok := ln.(*net.UnixListener); ok {
			// 端点路径由本包显式删除，交给监听器只会删错名字。
			unixListener.SetUnlinkOnClose(false)
		}

		linkErr := os.Link(temp, path)
		switch {
		case linkErr == nil:
			if err := os.Chmod(path, 0o600); err != nil {
				_ = ln.Close()
				_ = os.Remove(path)
				_ = os.Remove(temp)
				return nil, false, fmt.Errorf("host: 设置实例端点权限失败: %w", err)
			}
			_ = os.Remove(temp)
			sweepStaleTemps(path)
			return ln, stale, nil

		case errors.Is(linkErr, fs.ErrNotExist):
			// 临时套接字被并发启动的其他进程清理了，重建后重试。
			_ = ln.Close()
			lastErr = linkErr
			continue

		case errors.Is(linkErr, fs.ErrExist):
			_ = ln.Close()
			_ = os.Remove(temp)
			if endpointAlive(path, probeTimeout) {
				return nil, false, fmt.Errorf("%w: %s", ErrInstanceRunning, path)
			}
			if err := removeStaleEndpoint(path); err != nil {
				return nil, false, fmt.Errorf("host: 清理实例端点失败: %w", err)
			}
			stale = true
			lastErr = linkErr
			continue

		default:
			_ = ln.Close()
			_ = os.Remove(temp)
			return nil, false, fmt.Errorf("host: 发布实例端点失败: %w", linkErr)
		}
	}
	return nil, false, fmt.Errorf("host: 占用实例端点失败: %w", lastErr)
}

// listenTemp 在端点同目录下用临时名监听，保证发布前就已经可以接受连接。
func listenTemp(path string) (net.Listener, string, error) {
	dir := filepath.Dir(path)
	temp := filepath.Join(dir, fmt.Sprintf(".%s.%d.%d", filepath.Base(path), os.Getpid(), tempSeq.Add(1)))
	_ = os.Remove(temp)
	ln, err := net.Listen("unix", temp)
	if err != nil {
		return nil, "", err
	}
	return ln, temp, nil
}

// endpointAlive 连续探测端点，避免把刚发布还没被调度到的实例误判为残留。
func endpointAlive(path string, probeTimeout time.Duration) bool {
	for attempt := 0; attempt < 3; attempt++ {
		if conn, err := dialLocal(path, probeTimeout); err == nil {
			_ = conn.Close()
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}

// sweepStaleTemps 清理被强杀后残留的临时套接字。抢到端点的进程不可能与另一个
// 活实例并存，因此这里留下的临时文件都是残留。
func sweepStaleTemps(path string) {
	dir := filepath.Dir(path)
	prefix := "." + filepath.Base(path) + "."
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}
		_ = os.Remove(filepath.Join(dir, entry.Name()))
	}
}

func dialLocal(path string, timeout time.Duration) (net.Conn, error) {
	return net.DialTimeout("unix", path, timeout)
}

// removeStaleEndpoint 只删除套接字文件；端点位置若是普通文件则如实报错，不删用户数据。
func removeStaleEndpoint(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return err
	}
	if info.Mode()&os.ModeSocket == 0 {
		return fmt.Errorf("host: 实例端点位置被非套接字文件占用: %s", path)
	}
	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
