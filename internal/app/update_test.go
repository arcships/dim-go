package app

import (
	"context"
	"crypto/sha512"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
	"github.com/arcships/dimagent-mygo/internal/host"
)

// updateManifest 生成 electron-builder 形状的清单。
func updateManifest(version string, content []byte) string {
	sum := sha512.Sum512(content)
	return fmt.Sprintf("version: %s\nfiles:\n  - url: /artifact/app.bin\n    sha512: %s\n    size: %d\n",
		version, base64.StdEncoding.EncodeToString(sum[:]), len(content))
}

// startUpdateSource 提供清单与产物；artifact 为实际返回的字节。
func startUpdateSource(t *testing.T, manifest string, artifact []byte) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "latest-mac.yml"):
			_, _ = io.WriteString(w, manifest)
		case r.URL.Path == "/artifact/app.bin":
			_, _ = w.Write(artifact)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

// updateTestApp 建一个不连服务端的 App，并把更新编排指到本地版本源。
func updateTestApp(t *testing.T, server *httptest.Server, targetPath string, restart func(context.Context) error) *App {
	t.Helper()
	base, err := New(Config{DataDir: t.TempDir(), ClientName: "dimagent-update-test", Version: "1.0.0"})
	if err != nil {
		t.Fatalf("创建集成层: %v", err)
	}
	t.Cleanup(func() { _ = base.CloseUpdate() })
	if _, err := base.EnableUpdate(UpdateOptions{
		CurrentVersion: "1.0.0",
		ReadVersion: func() (string, error) {
			raw, err := os.ReadFile(targetPath)
			if err != nil {
				return "", err
			}
			return strings.TrimSpace(string(raw)), nil
		},
		ManifestURL: server.URL + "/updates/stable/mac/arm64/latest-mac.yml",
		TempDir:     t.TempDir(),
		TargetPath:  targetPath,
		Restart:     restart,
	}); err != nil {
		t.Fatalf("装配更新: %v", err)
	}
	return base
}

// TestUpdaterCheckFindsNewVersion 核对检查到新版。
func TestUpdaterCheckFindsNewVersion(t *testing.T) {
	artifact := []byte("2.0.0")
	server := startUpdateSource(t, updateManifest("2.0.0", artifact), artifact)
	target := filepath.Join(t.TempDir(), "dimagent")
	if err := os.WriteFile(target, []byte("1.0.0"), 0o700); err != nil {
		t.Fatalf("写入假产物: %v", err)
	}
	base := updateTestApp(t, server, target, nil)
	status, err := base.Updater().Check(context.Background())
	if err != nil {
		t.Fatalf("检查版本: %v", err)
	}
	if status.State != host.UpdateAvailable || status.AvailableVersion != "2.0.0" {
		t.Fatalf("应检查到新版: %+v", status)
	}
}

// TestUpdaterUpToDate 核对已是最新。
func TestUpdaterUpToDate(t *testing.T) {
	artifact := []byte("1.0.0")
	server := startUpdateSource(t, updateManifest("1.0.0", artifact), artifact)
	target := filepath.Join(t.TempDir(), "dimagent")
	if err := os.WriteFile(target, []byte("1.0.0"), 0o700); err != nil {
		t.Fatalf("写入假产物: %v", err)
	}
	base := updateTestApp(t, server, target, nil)
	status, err := base.Updater().Check(context.Background())
	if err != nil {
		t.Fatalf("检查版本: %v", err)
	}
	if status.State != host.UpdateUpToDate || status.AvailableVersion != "" {
		t.Fatalf("应是最新: %+v", status)
	}
}

// TestUpdaterDownloadChecksumFailure 核对摘要不一致时保留待下载状态。
func TestUpdaterDownloadChecksumFailure(t *testing.T) {
	expected := []byte("2.0.0")
	server := startUpdateSource(t, updateManifest("2.0.0", expected), []byte("2.0.0-tampered"))
	target := filepath.Join(t.TempDir(), "dimagent")
	if err := os.WriteFile(target, []byte("1.0.0"), 0o700); err != nil {
		t.Fatalf("写入假产物: %v", err)
	}
	base := updateTestApp(t, server, target, nil)
	ctx := context.Background()
	if _, err := base.Updater().Check(ctx); err != nil {
		t.Fatalf("检查版本: %v", err)
	}
	if _, err := base.Updater().Download(ctx); err == nil {
		t.Fatal("摘要不一致应下载失败")
	}
	if state := base.Updater().State(); state != host.UpdateAvailable {
		t.Fatalf("下载失败应保留待下载，实际 %s", state)
	}
}

// TestUpdaterSwitchDeferredWithOpenSession 核对有会话时切换推迟并明确报。
func TestUpdaterSwitchDeferredWithOpenSession(t *testing.T) {
	artifact := []byte("2.0.0")
	server := startUpdateSource(t, updateManifest("2.0.0", artifact), artifact)
	appServer := startFakeServer(t, fakeappserver.Script{
		Sessions: []fakeappserver.Session{{ID: testSession, Events: testEvents()}},
	})
	assembly := startAssembly(t, appServer, assemblyOptions{session: testSession})
	if assembly.base.SessionsIdle() {
		t.Fatal("打开了会话时不应空闲")
	}
	target := filepath.Join(t.TempDir(), "dimagent")
	if err := os.WriteFile(target, []byte("1.0.0"), 0o700); err != nil {
		t.Fatalf("写入假产物: %v", err)
	}
	if _, err := assembly.base.EnableUpdate(UpdateOptions{
		CurrentVersion: "1.0.0",
		ManifestURL:    server.URL + "/updates/stable/mac/arm64/latest-mac.yml",
		TempDir:        t.TempDir(),
		TargetPath:     target,
		Restart:        func(context.Context) error { return nil },
	}); err != nil {
		t.Fatalf("装配更新: %v", err)
	}
	ctx := context.Background()
	if _, err := assembly.base.Updater().Check(ctx); err != nil {
		t.Fatalf("检查版本: %v", err)
	}
	if _, err := assembly.base.Updater().Download(ctx); err != nil {
		t.Fatalf("下载产物: %v", err)
	}
	err := assembly.base.Updater().Switch(ctx)
	if err == nil || !strings.Contains(err.Error(), "推迟") {
		t.Fatalf("有会话时应推迟切换: %v", err)
	}
	if state := assembly.base.Updater().State(); state != host.UpdateReady {
		t.Fatalf("推迟后应保持待切换，实际 %s", state)
	}
	content, _ := os.ReadFile(target)
	if strings.TrimSpace(string(content)) != "1.0.0" {
		t.Fatalf("推迟时不应替换产物: %q", content)
	}
}

// TestUpdaterSwitchReplacesTargetAndVersion 核对切换后产物与版本号都变化。
func TestUpdaterSwitchReplacesTargetAndVersion(t *testing.T) {
	artifact := []byte("2.0.0")
	server := startUpdateSource(t, updateManifest("2.0.0", artifact), artifact)
	target := filepath.Join(t.TempDir(), "dimagent")
	if err := os.WriteFile(target, []byte("1.0.0"), 0o700); err != nil {
		t.Fatalf("写入假产物: %v", err)
	}
	restarts := 0
	base := updateTestApp(t, server, target, func(context.Context) error {
		restarts++
		return nil
	})
	ctx := context.Background()
	if _, err := base.Updater().Check(ctx); err != nil {
		t.Fatalf("检查版本: %v", err)
	}
	if _, err := base.Updater().Download(ctx); err != nil {
		t.Fatalf("下载产物: %v", err)
	}
	if err := base.Updater().Switch(ctx); err != nil {
		t.Fatalf("切换: %v", err)
	}
	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("读取替换后的产物: %v", err)
	}
	if strings.TrimSpace(string(content)) != "2.0.0" {
		t.Fatalf("产物内容应换成新版本: %q", content)
	}
	if restarts != 1 {
		t.Fatalf("切换后应重启一次，实际 %d", restarts)
	}
	if current := base.Updater().Status().CurrentVersion; current != "2.0.0" {
		t.Fatalf("切换后当前版本应变化，实际 %q", current)
	}
}
