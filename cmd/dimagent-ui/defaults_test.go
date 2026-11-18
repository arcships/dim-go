package main

import (
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// TestResolveDataDirPrecedence 核对数据目录的优先级：参数 → DIM_DATA_DIR → 平台默认。
func TestResolveDataDirPrecedence(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv(envDataDir, filepath.Join(home, "from-env"))

	got, err := resolveDataDir("/tmp/explicit-data-dir")
	if err != nil {
		t.Fatalf("解析数据目录: %v", err)
	}
	if got != "/tmp/explicit-data-dir" {
		t.Fatalf("参数应高于环境变量: %q", got)
	}

	got, err = resolveDataDir("")
	if err != nil {
		t.Fatalf("解析数据目录: %v", err)
	}
	if want := filepath.Join(home, "from-env"); got != want {
		t.Fatalf("环境变量应高于平台默认: %q, want %q", got, want)
	}

	t.Setenv(envDataDir, "")
	got, err = resolveDataDir("")
	if err != nil {
		t.Fatalf("解析数据目录: %v", err)
	}
	if runtime.GOOS == "darwin" {
		want := filepath.Join(home, "Library", "Application Support", "DimAgent")
		if got != want {
			t.Fatalf("macOS 默认数据目录: %q, want %q", got, want)
		}
	} else if !strings.HasSuffix(got, "DimAgent") {
		t.Fatalf("默认数据目录应以 DimAgent 结尾: %q", got)
	}
}

// TestExpandHome 核对 ~ 展开。
func TestExpandHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if got, want := expandHome("~/data"), filepath.Join(home, "data"); got != want {
		t.Fatalf("展开 ~: %q, want %q", got, want)
	}
	if got := expandHome("/absolute/path"); got != "/absolute/path" {
		t.Fatalf("绝对路径应原样返回: %q", got)
	}
}

// TestFindServerExplicitWins 核对显式指定优先，且不可用时立即失败。
func TestFindServerExplicitWins(t *testing.T) {
	dir := t.TempDir()
	explicit := fakeServer(t, filepath.Join(dir, "explicit", serverName))
	fromEnv := fakeServer(t, filepath.Join(dir, "env", serverName))
	t.Setenv(envAppServer, fromEnv)
	t.Setenv(legacyEnvAppServer, "")

	plan, err := findServer(explicit, dir)
	if err != nil {
		t.Fatalf("显式指定应成功: %v", err)
	}
	if plan.Path != explicit || plan.Source != "参数 -server" {
		t.Fatalf("应命中参数给出的可执行文件: %+v", plan)
	}

	missing := filepath.Join(dir, "missing", serverName)
	plan, err = findServer(missing, dir)
	if err == nil {
		t.Fatal("显式指定不可用时应失败")
	}
	if !plan.Explicit {
		t.Fatal("显式指定的失败应标出 Explicit，供启动时快速失败")
	}
}

// TestFindServerEnvAndEndpoint 核对环境变量档与「已有端点」档。
func TestFindServerEnvAndEndpoint(t *testing.T) {
	// 端点是真的 Unix socket：路径有长度上界，用短目录。
	dir := shortDir(t)
	fromEnv := fakeServer(t, filepath.Join(dir, "env", serverName))
	t.Setenv(envAppServer, fromEnv)
	t.Setenv(legacyEnvAppServer, "")

	plan, err := findServer("", dir)
	if err != nil {
		t.Fatalf("环境变量应命中: %v", err)
	}
	if plan.Path != fromEnv || plan.Source != "环境变量 "+envAppServer {
		t.Fatalf("应命中环境变量给出的可执行文件: %+v", plan)
	}

	// 没有可执行文件、没有 PATH、端点真的在监听：不拉起，直接连已有实例。
	t.Setenv(envAppServer, "")
	t.Setenv("PATH", "")
	socket := liveEndpoint(t, dir)
	plan, err = findServer("", dir)
	if err != nil {
		t.Fatalf("已有端点时应成功: %v", err)
	}
	if plan.Path != "" || plan.Endpoint != socket || plan.Source != "已有服务端点" {
		t.Fatalf("已有端点时不应拉起服务端: %+v", plan)
	}
}

// TestFindServerStaleEndpoint 核对残留端点：socket 文件在但没人监听时不算命中，
// 发现链继续找可执行文件，不拿残留端点掩盖「没有服务端」。
func TestFindServerStaleEndpoint(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 用命名管道，不做端点探测")
	}
	dir := shortDir(t)
	binDir := filepath.Join(dir, "bin")
	found := fakeServer(t, filepath.Join(binDir, serverName))
	t.Setenv(envAppServer, "")
	t.Setenv(legacyEnvAppServer, "")
	t.Setenv("PATH", binDir)
	staleEndpoint(t, dir)

	plan, err := findServer("", dir)
	if err != nil {
		t.Fatalf("残留端点不该挡住 PATH 里命中的可执行文件: %v", err)
	}
	if plan.Path != found || plan.Source != "PATH" {
		t.Fatalf("应继续命中 PATH 里的可执行文件: %+v", plan)
	}

	// 连可执行文件也没有：原因里要写明端点残留在、连不上，而不是含混地说找不到。
	t.Setenv("PATH", "")
	plan, err = findServer("", dir)
	if err == nil {
		t.Fatal("只有残留端点时应视为找不到 App Server")
	}
	if plan.Source != "" {
		t.Fatalf("残留端点不该被当成端点来源: %+v", plan)
	}
	if message := err.Error(); !strings.Contains(message, "端点残留") {
		t.Fatalf("原因里应写明端点残留: %s", message)
	}
}

// TestFindServerMissingListsTried 核对都找不到时给出可读原因（列出试过的位置）。
func TestFindServerMissingListsTried(t *testing.T) {
	dir := t.TempDir()
	t.Setenv(envAppServer, "")
	t.Setenv(legacyEnvAppServer, "")
	t.Setenv("PATH", "")

	plan, err := findServer("", dir)
	if err == nil {
		t.Fatal("都找不到时应失败")
	}
	if plan.Explicit {
		t.Fatal("自动发现的失败不应标出 Explicit")
	}
	message := err.Error()
	for _, want := range []string{"PATH", endpointPath(dir)} {
		if !strings.Contains(message, want) {
			t.Fatalf("原因里应列出 %q: %s", want, message)
		}
	}
}

// fakeServer 建一个可执行的占位文件，模拟 App Server。
func fakeServer(t *testing.T, path string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("建目录: %v", err)
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("写占位可执行文件: %v", err)
	}
	return path
}

// shortDir 建一个短路径的临时目录：Unix socket 路径有长度上界。
func shortDir(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("/tmp", "dim-ui-discovery")
	if err != nil {
		t.Fatalf("建临时目录: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	return dir
}

// liveEndpoint 建一个真的在监听的端点，返回路径。
func liveEndpoint(t *testing.T, dataDir string) string {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dataDir, endpointDirName), 0o700); err != nil {
		t.Fatalf("建端点目录: %v", err)
	}
	socket := endpointPath(dataDir)
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatalf("监听端点: %v", err)
	}
	t.Cleanup(func() { _ = listener.Close() })
	return socket
}

// staleEndpoint 留下一个没人监听的 socket 文件：上一次服务端崩溃后的残留。
func staleEndpoint(t *testing.T, dataDir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dataDir, endpointDirName), 0o700); err != nil {
		t.Fatalf("建端点目录: %v", err)
	}
	listener, err := net.Listen("unix", endpointPath(dataDir))
	if err != nil {
		t.Fatalf("监听端点: %v", err)
	}
	// 关掉时不删文件：留下的就是一个「在、但没人听」的端点。
	listener.(*net.UnixListener).SetUnlinkOnClose(false)
	if err := listener.Close(); err != nil {
		t.Fatalf("关闭端点: %v", err)
	}
}
