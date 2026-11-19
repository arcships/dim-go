package main

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/app"
)

// TestCleanupEndpointRemovesStaleSocket 核对退出清理：没人监听的残留端点删掉，
// 再清一次是空操作。
func TestCleanupEndpointRemovesStaleSocket(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 用命名管道，没有可删的 socket 文件")
	}
	dir := shortDir(t)
	staleEndpoint(t, dir)
	socket := endpointPath(dir)

	cleanupEndpoint(socket, nil)
	if _, err := os.Stat(socket); !os.IsNotExist(err) {
		t.Fatalf("残留端点应被删除：%v", err)
	}
	// 端点已经不在：不报错、不留东西。
	cleanupEndpoint(socket, nil)
}

// TestCleanupEndpointKeepsLiveInstance 核对外部实例：还在监听的端点不是残留，
// 不能删（那是它的 socket）。
func TestCleanupEndpointKeepsLiveInstance(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 用命名管道，没有可删的 socket 文件")
	}
	dir := shortDir(t)
	socket := liveEndpoint(t, dir)

	cleanupEndpoint(socket, nil)
	if _, err := os.Stat(socket); err != nil {
		t.Fatalf("外部实例还在监听的端点不该被删：%v", err)
	}
}

// TestQuitGateCleansStaleEndpoint 核对退出路径真的接上了清理：四阶段结束后由
// tidy 删掉没人监听的残留端点。
func TestQuitGateCleansStaleEndpoint(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 用命名管道，没有可删的 socket 文件")
	}
	dir := shortDir(t)
	staleEndpoint(t, dir)
	socket := endpointPath(dir)

	base, err := app.New(app.Config{DataDir: dir, ClientName: "quitgate-cleanup-test"})
	if err != nil {
		t.Fatalf("创建集成层: %v", err)
	}
	gate := newQuitGate(base, nil)
	gate.tidy = func() { cleanupEndpoint(socket, nil) }
	if !gate.settle(10 * time.Second) {
		t.Fatal("退出四阶段应在上限内结束")
	}
	if _, err := os.Stat(socket); !os.IsNotExist(err) {
		t.Fatalf("退出后残留端点应被删除：%v", err)
	}
}
