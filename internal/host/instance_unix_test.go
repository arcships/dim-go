//go:build !windows

package host

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

// TestStaleSocketIsReclaimed 模拟进程被强杀：端点文件留着但无人监听。
// 重新启动必须清理后占用，且不依赖任何额外的锁文件或状态文件。
func TestStaleSocketIsReclaimed(t *testing.T) {
	endpoint := shortEndpoint(t)
	ln, err := net.Listen("unix", endpoint)
	if err != nil {
		t.Fatalf("建残留套接字: %v", err)
	}
	unixListener, ok := ln.(*net.UnixListener)
	if !ok {
		t.Fatalf("监听器类型 = %T", ln)
	}
	unixListener.SetUnlinkOnClose(false)
	if err := ln.Close(); err != nil {
		t.Fatalf("关闭监听器: %v", err)
	}
	if _, err := os.Stat(endpoint); err != nil {
		t.Fatalf("残留套接字应存在: %v", err)
	}

	diag, buf := newTestDiag(t)
	instance, err := TryAcquire(InstanceConfig{Endpoint: endpoint, Diag: diag})
	if err != nil {
		t.Fatalf("残留端点应可重新占用: %v", err)
	}
	defer instance.Close()

	records := recordsOf(decodeRecords(t, buf.String()), EventInstanceStaleRemoved)
	if len(records) != 1 {
		t.Fatalf("残留清理记录条数 = %d，期望 1", len(records))
	}

	entries, err := os.ReadDir(filepath.Dir(endpoint))
	if err != nil {
		t.Fatalf("读端点目录: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != instanceFileName {
		t.Fatalf("端点目录出现额外状态文件: %v", entries)
	}

	if _, err := TryAcquire(InstanceConfig{Endpoint: endpoint}); !errors.Is(err, ErrInstanceRunning) {
		t.Fatalf("清理后应能正常互斥，实际 %v", err)
	}
}

// TestRegularFileEndpointIsNotDeleted：端点位置被普通文件占用时不删用户数据，如实报错。
func TestRegularFileEndpointIsNotDeleted(t *testing.T) {
	endpoint := shortEndpoint(t)
	if err := os.WriteFile(endpoint, []byte("不要删我"), 0o600); err != nil {
		t.Fatalf("写普通文件: %v", err)
	}
	if _, err := TryAcquire(InstanceConfig{Endpoint: endpoint}); err == nil {
		t.Fatal("普通文件占用端点时应报错")
	}
	if _, err := os.Stat(endpoint); err != nil {
		t.Fatalf("普通文件被删除: %v", err)
	}
}

func TestConcurrentAcquireOnlyOneWins(t *testing.T) {
	endpoint := shortEndpoint(t)
	const contenders = 8

	var wg sync.WaitGroup
	instances := make(chan *Instance, contenders)
	failures := make(chan error, contenders)
	for i := 0; i < contenders; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			instance, err := TryAcquire(InstanceConfig{Endpoint: endpoint})
			if err != nil {
				failures <- err
				return
			}
			instances <- instance
		}()
	}
	wg.Wait()
	close(instances)
	close(failures)

	var winners []*Instance
	for instance := range instances {
		winners = append(winners, instance)
	}
	if len(winners) != 1 {
		t.Fatalf("赢家数量 = %d，期望 1", len(winners))
	}
	for err := range failures {
		if !errors.Is(err, ErrInstanceRunning) {
			t.Fatalf("落选原因应为 ErrInstanceRunning，实际 %v", err)
		}
	}
	_ = winners[0].Close()
}

// TestShellReportsForwardFailure 覆盖"端点有人但不接受转交"：转交失败要如实报，
// 不能假装已接管，也不能把本进程当作主实例继续启动。
func TestShellReportsForwardFailure(t *testing.T) {
	endpoint := shortEndpoint(t)
	raw, err := net.Listen("unix", endpoint)
	if err != nil {
		t.Fatalf("占用端点: %v", err)
	}
	defer raw.Close()
	go func() {
		for {
			conn, err := raw.Accept()
			if err != nil {
				return
			}
			// 立即关闭：二次启动拿不到回执。
			_ = conn.Close()
		}
	}()

	var order []string
	cfg := Config{
		Endpoint: endpoint,
		Startup:  wiredStartup(&order, "", nil),
		Shutdown: ShutdownSteps{
			StopIntents:     func(context.Context) error { return nil },
			EndInteraction:  func(context.Context) error { return nil },
			CloseConnection: func(context.Context) error { return nil },
			ReclaimService:  func(context.Context) error { return nil },
		},
	}
	shell, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	if _, err := shell.Start(context.Background()); !errors.Is(err, ErrForwardFailed) {
		t.Fatalf("Start 错误 = %v，期望 ErrForwardFailed", err)
	}
	if state := shell.State(); state.Stage != StageIdle {
		t.Fatalf("转交失败不应推进启动阶段，实际 %s", state.Stage)
	}
	if len(order) != 0 {
		t.Fatalf("转交失败不应执行启动阶段: %v", order)
	}
}
