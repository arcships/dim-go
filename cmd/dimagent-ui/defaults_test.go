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
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	explicit := fakeServer(t, filepath.Join(dir, "explicit", serverName))
	fromEnv := fakeServer(t, filepath.Join(dir, "env", serverName))
	t.Setenv(envAppServer, fromEnv)
	t.Setenv(legacyEnvAppServer, "")

	plan, err := findServer(explicit, dir, false)
	if err != nil {
		t.Fatalf("显式指定应成功: %v", err)
	}
	if plan.Path != explicit || plan.Source != "参数 -server" {
		t.Fatalf("应命中参数给出的可执行文件: %+v", plan)
	}

	missing := filepath.Join(dir, "missing", serverName)
	plan, err = findServer(missing, dir, false)
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
	t.Setenv("HOME", shortDir(t))
	fromEnv := fakeServer(t, filepath.Join(dir, "env", serverName))
	t.Setenv(envAppServer, fromEnv)
	t.Setenv(legacyEnvAppServer, "")

	plan, err := findServer("", dir, false)
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
	plan, err = findServer("", dir, false)
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
	t.Setenv("HOME", shortDir(t))
	binDir := filepath.Join(dir, "bin")
	found := fakeServer(t, filepath.Join(binDir, serverName))
	t.Setenv(envAppServer, "")
	t.Setenv(legacyEnvAppServer, "")
	t.Setenv("PATH", binDir)
	staleEndpoint(t, dir)

	plan, err := findServer("", dir, false)
	if err != nil {
		t.Fatalf("残留端点不该挡住 PATH 里命中的可执行文件: %v", err)
	}
	if plan.Path != found || plan.Source != "PATH" {
		t.Fatalf("应继续命中 PATH 里的可执行文件: %+v", plan)
	}

	// 连可执行文件也没有：原因里要写明端点残留在、连不上，而不是含混地说找不到。
	t.Setenv("PATH", "")
	plan, err = findServer("", dir, false)
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
	t.Setenv("HOME", t.TempDir())
	dir := t.TempDir()
	t.Setenv(envAppServer, "")
	t.Setenv(legacyEnvAppServer, "")
	t.Setenv("PATH", "")

	plan, err := findServer("", dir, false)
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

// TestFindServerReusesExistingDaemon 核对发现顺序：已有 daemon 先于拉起自带 server，
// 多个 home 都有 daemon 时按 home 顺序取第一个，本端数据目录排在最后。
func TestFindServerReusesExistingDaemon(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 用命名管道，不做端点探测")
	}
	home := shortDir(t)
	t.Setenv("HOME", home)
	t.Setenv(envAppServer, "")
	t.Setenv(legacyEnvAppServer, "")
	// PATH 里放一个可执行文件：已有 daemon 仍然优先，不拉起。
	binDir := filepath.Join(home, "bin")
	fakeServer(t, filepath.Join(binDir, serverName))
	t.Setenv("PATH", binDir)

	dataDir := shortDir(t)
	own := liveEndpoint(t, dataDir)
	alpha := liveEndpoint(t, filepath.Join(home, ".dimcode-alpha"))

	plan, err := findServer("", dataDir, true)
	if err != nil {
		t.Fatalf("已有 daemon 时应成功: %v", err)
	}
	if !plan.Reuse || plan.DaemonSocket != alpha || plan.Path != "" {
		t.Fatalf("应复用 ~/.dimcode-alpha 的 daemon，不拉起: %+v", plan)
	}
	if plan.Endpoint != own {
		t.Fatalf("本端端点应保持不变: %q", plan.Endpoint)
	}
	if !strings.Contains(plan.Source, ".dimcode-alpha") {
		t.Fatalf("来源里应写明命中的 home: %q", plan.Source)
	}
}

// TestFindServerOwnEndpointLast 核对本端数据目录的 daemon 排在产品 home 之后。
func TestFindServerOwnEndpointLast(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 用命名管道，不做端点探测")
	}
	home := shortDir(t)
	t.Setenv("HOME", home)
	t.Setenv(envAppServer, "")
	t.Setenv(legacyEnvAppServer, "")
	t.Setenv("PATH", "")

	dataDir := shortDir(t)
	own := liveEndpoint(t, dataDir)
	alpha := liveEndpoint(t, filepath.Join(home, ".dimcode-alpha"))

	plan, err := findServer("", dataDir, true)
	if err != nil {
		t.Fatalf("已有 daemon 时应成功: %v", err)
	}
	if plan.DaemonSocket != alpha {
		t.Fatalf("产品 home 的 daemon 应优先于本端数据目录: %+v", plan)
	}

	// 产品 home 的 daemon 不在了：退回本端数据目录里还活着的那个。
	if err := os.Remove(alpha); err != nil {
		t.Fatalf("移除产品 home 端点: %v", err)
	}
	plan, err = findServer("", dataDir, true)
	if err != nil {
		t.Fatalf("本端端点应命中: %v", err)
	}
	if !plan.Reuse || plan.DaemonSocket != own || plan.Source != "已有服务端点" {
		t.Fatalf("应复用本端端点: %+v", plan)
	}
}

// TestFindServerExplicitDataDirSkipsExternal 核对外部 daemon 的隔离：显式数据目录
// （reuseExternal=false）不碰产品 home 的 daemon，只拉起自己的 server；要复用就
// 显式打开外部探测（-reuse-daemon 走的就是这一档）。
func TestFindServerExplicitDataDirSkipsExternal(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 用命名管道，不做端点探测")
	}
	home := shortDir(t)
	t.Setenv("HOME", home)
	t.Setenv(envAppServer, "")
	t.Setenv(legacyEnvAppServer, "")
	// 外部 daemon 活着，PATH 里也有可执行文件。
	alpha := liveEndpoint(t, filepath.Join(home, ".dimcode-alpha"))
	binDir := filepath.Join(home, "bin")
	found := fakeServer(t, filepath.Join(binDir, serverName))
	t.Setenv("PATH", binDir)

	dataDir := shortDir(t)
	plan, err := findServer("", dataDir, false)
	if err != nil {
		t.Fatalf("给了数据目录时应拉起自己的 server: %v", err)
	}
	if plan.Reuse || plan.DaemonSocket != "" {
		t.Fatalf("显式数据目录不该复用外部 daemon: %+v", plan)
	}
	if plan.Path != found || plan.Source != "PATH" {
		t.Fatalf("应命中自己的可执行文件: %+v", plan)
	}

	// 连可执行文件也没有：原因里只该有本端位置，不该出现被跳过的外部 home。
	t.Setenv("PATH", "")
	plan, err = findServer("", dataDir, false)
	if err == nil {
		t.Fatal("没有可执行文件、也不复用外部 daemon 时应报找不到")
	}
	message := err.Error()
	if strings.Contains(message, ".dimcode-alpha") {
		t.Fatalf("「找不到」的原因里不该出现被跳过的外部 home: %s", message)
	}
	if !strings.Contains(message, endpointPath(dataDir)) {
		t.Fatalf("原因里应列出本端端点: %s", message)
	}

	// 打开外部探测（-reuse-daemon）时能复用同一个外部 daemon。
	plan, err = findServer("", dataDir, true)
	if err != nil {
		t.Fatalf("允许复用时应有命中: %v", err)
	}
	if !plan.Reuse || plan.DaemonSocket != alpha {
		t.Fatalf("打开外部探测后应复用产品 home 的 daemon: %+v", plan)
	}
}

// TestFindServerExplicitDataDirOwnEndpoint 核对隔离语义的边界：调用方自己的数据目录
// 里还有活着的实例时仍然复用——那是本端端点，不是外部 home。
func TestFindServerExplicitDataDirOwnEndpoint(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 用命名管道，不做端点探测")
	}
	home := shortDir(t)
	t.Setenv("HOME", home)
	t.Setenv(envAppServer, "")
	t.Setenv(legacyEnvAppServer, "")
	t.Setenv("PATH", "")

	liveEndpoint(t, filepath.Join(home, ".dimcode-alpha"))
	dataDir := shortDir(t)
	own := liveEndpoint(t, dataDir)

	plan, err := findServer("", dataDir, false)
	if err != nil {
		t.Fatalf("本端端点应命中: %v", err)
	}
	if !plan.Reuse || plan.DaemonSocket != own || plan.Source != "已有服务端点" {
		t.Fatalf("显式数据目录里自己的实例应复用: %+v", plan)
	}
}

// TestBridgeEndpoint 核对复用接入：本端端点变成指向对方 socket 的软链，退出时只删
// 软链、不动对方的 socket。
func TestBridgeEndpoint(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 用命名管道")
	}
	dir := shortDir(t)
	own := endpointPath(dir)
	target := liveEndpoint(t, filepath.Join(dir, "foreign"))
	log := openStartupLog(filepath.Join(dir, "test.log"))
	defer log.close()

	if err := bridgeEndpoint(own, target, log); err != nil {
		t.Fatalf("接入已有 daemon: %v", err)
	}
	if got, err := os.Readlink(own); err != nil || got != target {
		t.Fatalf("本端端点应是指向对方的软链: %q, %v", got, err)
	}
	// 再接一次是幂等的：同一条软链不该被反复改写。
	if err := bridgeEndpoint(own, target, log); err != nil {
		t.Fatalf("重复接入: %v", err)
	}

	cleanupEndpoint(own, log)
	if _, err := os.Lstat(own); !os.IsNotExist(err) {
		t.Fatalf("退出时应移除复用接入: %v", err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("对方的 socket 不该被删: %v", err)
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
