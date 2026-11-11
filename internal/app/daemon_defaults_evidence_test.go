package app

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"
)

// TestRealRuntimeDefaultsEvidence 是默认预设（`--defaults`）的真实证据。
//
// 只有设置 DIMAGENT_DEFAULTS_EVIDENCE=1 时运行；夹具 home 与 runtime 目录从只读
// 来源复制到临时目录。证据只留能力名、插件名与运行组件状态。
func TestRealRuntimeDefaultsEvidence(t *testing.T) {
	if os.Getenv("DIMAGENT_DEFAULTS_EVIDENCE") != "1" {
		t.Skip("未设置 DIMAGENT_DEFAULTS_EVIDENCE=1，跳过默认预设证据")
	}
	fixtureHome := os.Getenv("DIMAGENT_ACCOUNT_FIXTURE")
	if fixtureHome == "" {
		fixtureHome = "/private/tmp/dimagent-real-model/home"
	}
	runtimeSource := os.Getenv("DIMAGENT_RUNTIME_PRESET")
	if runtimeSource == "" {
		runtimeSource = filepath.Join(os.Getenv("HOME"), "Code", "dim-2", "runtime")
	}
	serverPath := os.Getenv("DIMAGENT_APP_SERVER_BIN")
	if serverPath == "" {
		serverPath = filepath.Join(os.Getenv("HOME"), ".dimcode-alpha", "rust", "dimcode-server")
	}
	preset := filepath.Join(runtimeSource, "default-plugins.json")
	if _, err := os.Stat(preset); err != nil {
		t.Fatalf("预设文件不可用: %v", err)
	}

	runDir, err := os.MkdirTemp("/tmp", "d12d-")
	if err != nil {
		t.Fatalf("创建临时目录: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(runDir) })
	if err := copyFixtureTree(fixtureHome, runDir); err != nil {
		t.Fatalf("复制夹具 home: %v", err)
	}
	runtimeCopy := filepath.Join(runDir, "runtime")
	if err := os.MkdirAll(runtimeCopy, 0o700); err != nil {
		t.Fatalf("创建 runtime 副本: %v", err)
	}
	if err := copyFixtureTree(runtimeSource, runtimeCopy); err != nil {
		t.Fatalf("复制 runtime 预设: %v", err)
	}

	base, err := New(Config{
		DataDir:        runDir,
		ServerPath:     serverPath,
		ServerDefaults: filepath.Join(runtimeCopy, "default-plugins.json"),
		ClientName:     "dimagent-mygo-defaults-evidence",
		WorkspaceRoot:  runDir,
		Version:        testVersion,
		SendTimeout:    30 * time.Second,
	})
	if err != nil {
		t.Fatalf("创建集成层: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	if err := base.LoadLocalState(ctx); err != nil {
		t.Fatalf("装载本地状态: %v", err)
	}
	if err := base.Connect(ctx); err != nil {
		t.Fatalf("连接服务端: %v", err)
	}
	defer base.CloseConnection(ctx)

	var runtimeFeatures []string
	if conn := base.connection(); conn != nil {
		for _, feature := range conn.Hello.Features {
			if strings.HasPrefix(feature, "runtime.") {
				runtimeFeatures = append(runtimeFeatures, feature)
			}
		}
	}
	sort.Strings(runtimeFeatures)
	t.Logf("hello.runtime: %v", runtimeFeatures)

	plugins, err := base.Extensions().ListPlugins(ctx)
	if err != nil {
		t.Logf("plugin.list: 读取失败: %v", err)
	} else {
		names := make([]string, 0, len(plugins.Items))
		hasRuntimeManager := false
		for _, plugin := range plugins.Items {
			names = append(names, plugin.ID)
			if plugin.ID == "RuntimeManager" || plugin.Name == "RuntimeManager" {
				hasRuntimeManager = true
			}
		}
		sort.Strings(names)
		t.Logf("plugin.list: count=%d runtimeManager=%t names=%v", len(names), hasRuntimeManager, names)
	}

	entries, err := base.Settings().RefreshRuntimes(ctx)
	if err != nil {
		t.Fatalf("runtime.list: 读取失败: %v", err)
	}
	summaries := make([]string, 0, len(entries))
	for _, entry := range entries {
		summaries = append(summaries, entry.Name+"="+string(entry.Status))
	}
	sort.Strings(summaries)
	t.Logf("runtime.list: count=%d %v", len(entries), summaries)
}
