package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/app"
	"github.com/arcships/dimagent-mygo/internal/uiwire"
)

// envTestRealServer 指向本机真实 App Server：验收里「装上服务端再重连」的用例
// 需要它，缺失时用例跳过并写明依赖，不用替身冒充真实服务端。
const envTestRealServer = "DIM_TEST_REAL_SERVER"

// TestRetryingIntentsRediscoversServer 核对「没有服务端 → 装上服务端 → 重新连接成功」。
//
// 断线卡上的「重新连接」走的就是这条路径：先常规重连，连不上就重跑发现链，
// 把新路径交给集成层，再走一遍连接与订阅。启动时找不到服务端不是定局。
func TestRetryingIntentsRediscoversServer(t *testing.T) {
	server := strings.TrimSpace(os.Getenv(envTestRealServer))
	if server == "" {
		t.Skip("本机没有真实 App Server：用 " + envTestRealServer + " 指过去")
	}
	if ok, reason := executable(server); !ok {
		t.Skipf("%s 指的不是可执行文件：%s（%s）", envTestRealServer, server, reason)
	}

	dir := shortDir(t)
	binDir := filepath.Join(dir, "bin")
	if err := os.MkdirAll(binDir, 0o700); err != nil {
		t.Fatalf("建目录: %v", err)
	}
	// 发现链的后两档都清掉：启动时确实「没有服务端」。
	t.Setenv(envAppServer, "")
	t.Setenv(legacyEnvAppServer, "")
	t.Setenv("PATH", binDir)
	dataDir := filepath.Join(dir, "data")

	base, err := app.New(app.Config{
		DataDir:    dataDir,
		MockModel:  true,
		ClientName: "dimagent-ui-reconnect-test",
	})
	if err != nil {
		t.Fatalf("创建集成层: %v", err)
	}
	wire := uiwire.New(base)
	discover := serverDiscovery{dataDir: dataDir, explicit: "", base: base}
	// 入口侧的重连日志：与真实入口一样写到启动日志，验收从文件里读。
	logPath := filepath.Join(dir, "dimagent.log")
	startup := openStartupLog(logPath)
	defer startup.close()
	intents := retryingIntents{
		App:    wire,
		rescan: func() { discover.apply("重新连接") },
		log:    startup,
	}
	// 与入口一致：启动时先跑一次发现链，把「没有可执行文件」的说明交给集成层。
	discover.apply("启动")

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	// 1. 没有服务端：连不上，断线原因是发现链的说明，不是拉起服务的内部细节。
	if err := wire.Connect(ctx); err == nil {
		t.Fatal("没有 App Server 时不该连上")
	}
	reason := wire.Connection().Reason
	if !strings.Contains(reason, "按顺序都没找到 App Server") {
		t.Fatalf("断线原因应是发现链的说明，实际 %q", reason)
	}
	if strings.Contains(reason, "缺少 App Server 可执行文件路径") {
		t.Fatalf("断线原因不该是拉起服务的内部细节：%q", reason)
	}
	t.Logf("① 没有服务端：%s", reason)

	// 2. 装上服务端：放进 PATH 档（发现链会重新看到它）。
	installed := filepath.Join(binDir, serverName)
	if err := os.Symlink(server, installed); err != nil {
		t.Fatalf("装服务端: %v", err)
	}

	// 3. 点「重新连接」：重跑发现链之后连上。
	if err := intents.Reconnect(ctx); err != nil {
		t.Fatalf("装上服务端之后重新连接应成功: %v", err)
	}
	conn := wire.Connection()
	if conn.Phase != "connected" {
		t.Fatalf("重连后阶段 = %q（%s）", conn.Phase, conn.Reason)
	}
	if !base.Connected() {
		t.Fatal("重连后控制连接应可用")
	}
	t.Logf("② 装上 %s 之后重新连接：阶段 %s", installed, conn.Phase)

	// 4. 重连接要在日志里留一行结果（含耗时）：外部验收不用靠界面猜。
	logText := readText(t, logPath)
	for _, want := range []string{"重新连接：常规重连失败", "重新连接：连接就绪（", "ms）"} {
		if !strings.Contains(logText, want) {
			t.Fatalf("重连日志缺少 %q：\n%s", want, logText)
		}
	}
	t.Logf("③ 重连日志：\n%s", strings.TrimSpace(logText))

	// 5. 标题跟着阶段走：连上之后回到产品标题，不是「连接失败」。
	titles := newWindowTitle(modeConnect, startup)
	if got := titles.forConnection(wire.Connection()); got != titleProduct {
		t.Fatalf("重连后的窗口标题 = %q, want %q", got, titleProduct)
	}
	t.Logf("④ 重连后的窗口标题：%s", titleProduct)

	// 6. 收尾：按退出四阶段回收自启实例，再删掉它留下的端点文件。
	shutdown(base, startup)
	socket := endpointPath(dataDir)
	cleanupEndpoint(socket, startup)
	if _, err := os.Stat(socket); !os.IsNotExist(err) {
		t.Fatalf("退出后端点文件应已删除：%v", err)
	}
	t.Logf("⑤ 退出后端点 %s 已不存在", socket)
}

// readText 读文件内容；读不到时直接失败（验收要看日志原文）。
func readText(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读 %s: %v", path, err)
	}
	return string(data)
}
