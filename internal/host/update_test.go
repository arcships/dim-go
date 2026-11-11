package host

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func channelManifest(version string, content []byte) string {
	return fmt.Sprintf("version: %s\nfiles:\n  - url: /artifact/app.bin\n    sha512: %s\n    size: %d\n",
		version, testDigest(content), len(content))
}

// newUpdateServer 提供版本清单与产物；artifactStatus 非 0 时产物返回该状态码。
func newUpdateServer(t *testing.T, manifest string, artifactStatus int, content []byte) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "latest-mac.yml") {
			_, _ = io.WriteString(w, manifest)
			return
		}
		if r.URL.Path == "/artifact/app.bin" {
			if artifactStatus != 0 && artifactStatus != http.StatusOK {
				w.WriteHeader(artifactStatus)
				return
			}
			_, _ = w.Write(content)
			return
		}
		http.NotFound(w, r)
	}))
	t.Cleanup(server.Close)
	return server
}

func manifestURLFor(server *httptest.Server) string {
	return server.URL + "/updates/stable/mac/arm64/latest-mac.yml"
}

func newTestUpdater(t *testing.T, manifestURL string, cfg UpdateConfig) (*Updater, *syncBuffer) {
	t.Helper()
	diag, buf := newTestDiag(t)
	cfg.ManifestURL = manifestURL
	cfg.Diag = diag
	cfg.TempDir = t.TempDir()
	if cfg.CurrentVersion == "" && cfg.ReadVersion == nil {
		cfg.CurrentVersion = "1.0.0"
	}
	// 这些用例用临时文件模拟安装产物，显式选择单文件模式，避免在 macOS 上误走 .app 适配器。
	if cfg.InstallMode == "" {
		cfg.InstallMode = InstallFile
	}
	defaults := UpdateTimeouts{
		Check:    time.Second,
		Download: time.Second,
		Settle:   time.Second,
		Switch:   time.Second,
	}
	if cfg.Timeouts.Check == 0 {
		cfg.Timeouts.Check = defaults.Check
	}
	if cfg.Timeouts.Download == 0 {
		cfg.Timeouts.Download = defaults.Download
	}
	if cfg.Timeouts.Settle == 0 {
		cfg.Timeouts.Settle = defaults.Settle
	}
	if cfg.Timeouts.Switch == 0 {
		cfg.Timeouts.Switch = defaults.Switch
	}
	updater, err := NewUpdater(cfg)
	if err != nil {
		t.Fatalf("NewUpdater: %v", err)
	}
	t.Cleanup(func() { _ = updater.Close() })
	return updater, buf
}

func writeTarget(t *testing.T, dir, content string) string {
	t.Helper()
	target := filepath.Join(dir, "DimAgent.bin")
	if err := os.WriteFile(target, []byte(content), 0o600); err != nil {
		t.Fatalf("写安装产物: %v", err)
	}
	return target
}

func readText(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读 %s: %v", path, err)
	}
	return string(data)
}

func TestUpdaterCheckFindsNewVersion(t *testing.T) {
	server := newUpdateServer(t, channelManifest("2.1.0", []byte("2.1.0")), 0, []byte("2.1.0"))
	updater, buf := newTestUpdater(t, manifestURLFor(server), UpdateConfig{})

	status, err := updater.Check(context.Background())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if status.State != UpdateAvailable {
		t.Fatalf("状态 = %s，期望 available", status.State)
	}
	if status.AvailableVersion != "2.1.0" || status.CurrentVersion != "1.0.0" {
		t.Fatalf("状态快照 = %+v", status)
	}
	if status.LastError != nil {
		t.Fatalf("成功后 LastError = %v", status.LastError)
	}
	if status.CheckedAt.IsZero() {
		t.Fatal("缺少检查时间")
	}

	records := recordsOf(decodeRecords(t, buf.String()), EventUpdateCheck)
	if len(records) != 1 {
		t.Fatalf("检查记录条数 = %d", len(records))
	}
	if records[0]["outcome"] != "ok" || records[0]["update_state"] != "available" ||
		records[0]["update_version"] != "2.1.0" {
		t.Fatalf("检查记录 = %v", records[0])
	}

	// 同一进程内再次检查属于不合法迁移：丢弃并记诊断，状态不变。
	if _, err := updater.Check(context.Background()); !errors.Is(err, ErrUpdateState) {
		t.Fatalf("再次检查错误 = %v，期望 ErrUpdateState", err)
	}
	if updater.State() != UpdateAvailable {
		t.Fatalf("丢弃后状态 = %s", updater.State())
	}
	rejected := recordsOf(decodeRecords(t, buf.String()), EventUpdateTransitionRejected)
	if len(rejected) != 1 || rejected[0]["update_state"] != "available" || rejected[0]["outcome"] != "skipped" {
		t.Fatalf("丢弃记录 = %v", rejected)
	}
}

func TestUpdaterCheckUpToDate(t *testing.T) {
	for _, version := range []string{"1.0.0", "0.9.9"} {
		server := newUpdateServer(t, channelManifest(version, []byte("payload")), 0, []byte("payload"))
		updater, buf := newTestUpdater(t, manifestURLFor(server), UpdateConfig{})
		status, err := updater.Check(context.Background())
		if err != nil {
			t.Fatalf("版本 %s: Check: %v", version, err)
		}
		if status.State != UpdateUpToDate || status.AvailableVersion != "" {
			t.Fatalf("版本 %s 状态 = %+v", version, status)
		}
		records := recordsOf(decodeRecords(t, buf.String()), EventUpdateCheck)
		if len(records) != 1 || records[0]["update_state"] != "up_to_date" || records[0]["outcome"] != "ok" {
			t.Fatalf("版本 %s 检查记录 = %v", version, records)
		}
	}
}

func TestUpdaterCheckFailureIsRecordedAndDoesNotBlock(t *testing.T) {
	t.Run("版本源 500", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		t.Cleanup(server.Close)
		updater, buf := newTestUpdater(t, manifestURLFor(server), UpdateConfig{})
		status, err := updater.Check(context.Background())
		if err == nil {
			t.Fatal("检查失败应返回错误")
		}
		if status.State != UpdateUpToDate || status.LastError == nil {
			t.Fatalf("失败后状态 = %+v", status)
		}
		records := recordsOf(decodeRecords(t, buf.String()), EventUpdateCheck)
		if len(records) != 1 || records[0]["outcome"] != "failed" ||
			records[0]["update_state"] != "up_to_date" || records[0]["classification"] != "retryable" {
			t.Fatalf("失败记录 = %v", records)
		}
		// 失败后仍可继续使用进程，状态是已是最新，不会卡在检查中。
		if updater.State() != UpdateUpToDate {
			t.Fatalf("失败后状态 = %s", updater.State())
		}
	})

	t.Run("清单不可解析", func(t *testing.T) {
		server := newUpdateServer(t, "version: nope\npath: app.bin\nsha512: bad", 0, nil)
		updater, buf := newTestUpdater(t, manifestURLFor(server), UpdateConfig{})
		if _, err := updater.Check(context.Background()); err == nil {
			t.Fatal("坏清单应返回错误")
		}
		records := recordsOf(decodeRecords(t, buf.String()), EventUpdateCheck)
		if len(records) != 1 || records[0]["classification"] != "non_retryable" {
			t.Fatalf("失败记录 = %v", records)
		}
	})

	t.Run("检查超时", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			<-r.Context().Done()
		}))
		t.Cleanup(server.Close)
		updater, buf := newTestUpdater(t, manifestURLFor(server), UpdateConfig{
			Timeouts: UpdateTimeouts{Check: 30 * time.Millisecond},
		})
		if _, err := updater.Check(context.Background()); err == nil {
			t.Fatal("超时应返回错误")
		}
		records := recordsOf(decodeRecords(t, buf.String()), EventUpdateCheck)
		if len(records) != 1 || records[0]["classification"] != "timeout" {
			t.Fatalf("失败记录 = %v", records)
		}
	})
}

func TestUpdaterDownloadVerifiesArtifact(t *testing.T) {
	content := []byte("2.1.0")
	server := newUpdateServer(t, channelManifest("2.1.0", content), 0, content)
	updater, buf := newTestUpdater(t, manifestURLFor(server), UpdateConfig{})

	if _, err := updater.Check(context.Background()); err != nil {
		t.Fatalf("Check: %v", err)
	}
	artifact, err := updater.Download(context.Background())
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if got := readText(t, artifact); got != string(content) {
		t.Fatalf("产物内容 = %q", got)
	}
	status := updater.Status()
	if status.State != UpdateReady || status.ArtifactPath != artifact {
		t.Fatalf("下载后状态 = %+v", status)
	}
	records := recordsOf(decodeRecords(t, buf.String()), EventUpdateDownload)
	if len(records) != 1 || records[0]["outcome"] != "ok" || records[0]["update_version"] != "2.1.0" {
		t.Fatalf("下载记录 = %v", records)
	}
}

func TestUpdaterDownloadFailureKeepsAvailable(t *testing.T) {
	t.Run("摘要不一致", func(t *testing.T) {
		manifest := fmt.Sprintf("version: 2.1.0\nfiles:\n  - url: /artifact/app.bin\n    sha512: %s\n    size: %d\n",
			testDigest([]byte("other")), len("2.1.0"))
		server := newUpdateServer(t, manifest, 0, []byte("2.1.0"))
		updater, buf := newTestUpdater(t, manifestURLFor(server), UpdateConfig{})
		if _, err := updater.Check(context.Background()); err != nil {
			t.Fatalf("Check: %v", err)
		}
		if _, err := updater.Download(context.Background()); err == nil {
			t.Fatal("摘要不一致应失败")
		}
		if updater.State() != UpdateAvailable {
			t.Fatalf("失败后状态 = %s，期望 available", updater.State())
		}
		if status := updater.Status(); status.LastError == nil || status.ArtifactPath != "" {
			t.Fatalf("失败后状态 = %+v", status)
		}
		entries, err := os.ReadDir(updater.tempDir)
		if err != nil {
			t.Fatalf("读临时目录: %v", err)
		}
		if len(entries) != 0 {
			t.Fatalf("失败后残留产物: %v", entries)
		}
		records := recordsOf(decodeRecords(t, buf.String()), EventUpdateDownload)
		if len(records) != 1 || records[0]["outcome"] != "failed" || records[0]["classification"] != "non_retryable" {
			t.Fatalf("下载记录 = %v", records)
		}
	})

	t.Run("产物 404", func(t *testing.T) {
		server := newUpdateServer(t, channelManifest("2.1.0", []byte("2.1.0")), http.StatusNotFound, []byte("2.1.0"))
		updater, _ := newTestUpdater(t, manifestURLFor(server), UpdateConfig{})
		if _, err := updater.Check(context.Background()); err != nil {
			t.Fatalf("Check: %v", err)
		}
		if _, err := updater.Download(context.Background()); err == nil {
			t.Fatal("404 应失败")
		}
		if updater.State() != UpdateAvailable {
			t.Fatalf("失败后状态 = %s", updater.State())
		}
	})

	t.Run("未检查就下载", func(t *testing.T) {
		server := newUpdateServer(t, channelManifest("2.1.0", []byte("2.1.0")), 0, []byte("2.1.0"))
		updater, buf := newTestUpdater(t, manifestURLFor(server), UpdateConfig{})
		if _, err := updater.Download(context.Background()); !errors.Is(err, ErrUpdateState) {
			t.Fatalf("错误 = %v，期望 ErrUpdateState", err)
		}
		rejected := recordsOf(decodeRecords(t, buf.String()), EventUpdateTransitionRejected)
		if len(rejected) != 1 || rejected[0]["update_state"] != "unchecked" {
			t.Fatalf("丢弃记录 = %v", rejected)
		}
	})
}

func TestUpdaterSwitchDefersWhileSessionsBusy(t *testing.T) {
	content := []byte("2.1.0")
	server := newUpdateServer(t, channelManifest("2.1.0", content), 0, content)
	dir := t.TempDir()
	target := writeTarget(t, dir, "1.0.0")

	idle := false
	var order []string
	restartVersion := ""
	readVersion := func() (string, error) {
		data, err := os.ReadFile(target)
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(data)), nil
	}

	updater, buf := newTestUpdater(t, manifestURLFor(server), UpdateConfig{
		ReadVersion:  readVersion,
		TargetPath:   target,
		SessionsIdle: func() bool { return idle },
		Settle: func(context.Context) error {
			order = append(order, "settle")
			return nil
		},
		Shutdown: func(context.Context) error {
			order = append(order, "shutdown")
			if got := readText(t, target); got != "1.0.0" {
				t.Fatalf("退出五阶段时产物已变化: %q", got)
			}
			return nil
		},
		Restart: func(context.Context) error {
			order = append(order, "restart")
			restartVersion = readText(t, target)
			return nil
		},
	})
	if _, err := updater.Check(context.Background()); err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, err := updater.Download(context.Background()); err != nil {
		t.Fatalf("Download: %v", err)
	}
	if got := updater.Status().CurrentVersion; got != "1.0.0" {
		t.Fatalf("切换前版本 = %q", got)
	}

	// 会话始终收不干净：推迟，不替换、不重启。
	if err := updater.Switch(context.Background()); !errors.Is(err, ErrSessionsBusy) {
		t.Fatalf("Switch 错误 = %v，期望 ErrSessionsBusy", err)
	}
	if updater.State() != UpdateReady {
		t.Fatalf("推迟后状态 = %s", updater.State())
	}
	if got := readText(t, target); got != "1.0.0" {
		t.Fatalf("推迟时产物被替换: %q", got)
	}
	if len(order) != 1 || order[0] != "settle" {
		t.Fatalf("推迟时执行步骤 = %v", order)
	}
	deferred := recordsOf(decodeRecords(t, buf.String()), EventUpdateSwitchDeferred)
	if len(deferred) != 1 || deferred[0]["update_state"] != "ready_to_switch" ||
		deferred[0]["outcome"] != "skipped" {
		t.Fatalf("推迟记录 = %v", deferred)
	}

	// 会话收尾后再次确认：退出 → 替换 → 重启。
	idle = true
	if err := updater.Switch(context.Background()); err != nil {
		t.Fatalf("Switch: %v", err)
	}
	if updater.State() != UpdateSwitching {
		t.Fatalf("切换后状态 = %s", updater.State())
	}
	if got := readText(t, target); got != "2.1.0" {
		t.Fatalf("替换后产物 = %q", got)
	}
	if restartVersion != "2.1.0" {
		t.Fatalf("重启步骤读到的版本 = %q", restartVersion)
	}
	if got := updater.Status().CurrentVersion; got != "2.1.0" {
		t.Fatalf("切换后当前版本 = %q，期望 2.1.0", got)
	}
	wantOrder := []string{"settle", "shutdown", "restart"}
	if len(order) != len(wantOrder) {
		t.Fatalf("切换步骤 = %v", order)
	}
	for i := range wantOrder {
		if order[i] != wantOrder[i] {
			t.Fatalf("切换步骤 = %v，期望 %v", order, wantOrder)
		}
	}
	switched := recordsOf(decodeRecords(t, buf.String()), EventUpdateSwitch)
	if len(switched) != 1 || switched[0]["outcome"] != "ok" || switched[0]["update_version"] != "2.1.0" {
		t.Fatalf("切换记录 = %v", switched)
	}
}

func TestUpdaterSwitchRequiresWiring(t *testing.T) {
	content := []byte("2.1.0")
	server := newUpdateServer(t, channelManifest("2.1.0", content), 0, content)
	dir := t.TempDir()
	target := writeTarget(t, dir, "1.0.0")

	newReady := func(t *testing.T, cfg UpdateConfig) (*Updater, *syncBuffer) {
		t.Helper()
		updater, buf := newTestUpdater(t, manifestURLFor(server), cfg)
		if _, err := updater.Check(context.Background()); err != nil {
			t.Fatalf("Check: %v", err)
		}
		if _, err := updater.Download(context.Background()); err != nil {
			t.Fatalf("Download: %v", err)
		}
		return updater, buf
	}

	t.Run("缺少目标路径", func(t *testing.T) {
		updater, buf := newReady(t, UpdateConfig{
			Restart: func(context.Context) error { return nil },
		})
		if err := updater.Switch(context.Background()); err == nil {
			t.Fatal("缺少安装路径应失败")
		}
		if updater.State() != UpdateReady {
			t.Fatalf("未接线失败后状态 = %s，期望 ready_to_switch", updater.State())
		}
		records := recordsOf(decodeRecords(t, buf.String()), EventUpdateSwitch)
		if len(records) != 1 || records[0]["outcome"] != "failed" {
			t.Fatalf("切换记录 = %v", records)
		}
	})

	t.Run("缺少重启步骤", func(t *testing.T) {
		updater, _ := newReady(t, UpdateConfig{TargetPath: target})
		if err := updater.Switch(context.Background()); err == nil {
			t.Fatal("缺少重启步骤应失败")
		}
		if updater.State() != UpdateReady {
			t.Fatalf("未接线失败后状态 = %s", updater.State())
		}
		if got := readText(t, target); got != "1.0.0" {
			t.Fatalf("失败后产物被替换: %q", got)
		}
	})
}

func TestUpdaterSwitchInstallFailureKeepsCurrentVersion(t *testing.T) {
	content := []byte("2.1.0")
	server := newUpdateServer(t, channelManifest("2.1.0", content), 0, content)
	dir := t.TempDir()

	restarted := false
	updater, buf := newTestUpdater(t, manifestURLFor(server), UpdateConfig{
		TargetPath: dir, // 目录无法被单文件替换
		Restart: func(context.Context) error {
			restarted = true
			return nil
		},
	})
	if _, err := updater.Check(context.Background()); err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, err := updater.Download(context.Background()); err != nil {
		t.Fatalf("Download: %v", err)
	}
	if err := updater.Switch(context.Background()); err == nil {
		t.Fatal("替换到目录应失败")
	}
	if restarted {
		t.Fatal("替换失败不应重启")
	}
	if updater.State() != UpdateReady {
		t.Fatalf("准备失败后状态 = %s，期望 ready_to_switch（未进入切换中）", updater.State())
	}
	records := recordsOf(decodeRecords(t, buf.String()), EventUpdateSwitch)
	if len(records) != 1 || records[0]["outcome"] != "failed" ||
		records[0]["classification"] != "non_retryable" ||
		records[0]["update_state"] != "ready_to_switch" {
		t.Fatalf("切换记录 = %v", records)
	}
}

func TestUpdaterSwitchWithoutShutdownWiring(t *testing.T) {
	content := []byte("2.1.0")
	server := newUpdateServer(t, channelManifest("2.1.0", content), 0, content)
	target := writeTarget(t, t.TempDir(), "1.0.0")
	restarted := false

	updater, _ := newTestUpdater(t, manifestURLFor(server), UpdateConfig{
		TargetPath: target,
		Restart: func(context.Context) error {
			restarted = true
			return nil
		},
	})
	if _, err := updater.Check(context.Background()); err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, err := updater.Download(context.Background()); err != nil {
		t.Fatalf("Download: %v", err)
	}
	if err := updater.Switch(context.Background()); err != nil {
		t.Fatalf("Switch: %v", err)
	}
	if !restarted || readText(t, target) != "2.1.0" {
		t.Fatal("未接退出步骤时仍应替换并重启")
	}
}

func TestNewUpdaterValidation(t *testing.T) {
	if _, err := NewUpdater(UpdateConfig{}); err == nil {
		t.Fatal("缺少当前版本应报错")
	}
	if _, err := NewUpdater(UpdateConfig{
		CurrentVersion: "v1",
		Platform:       "darwin",
		Arch:           "arm64",
	}); err == nil {
		t.Fatal("当前版本不合法应报错")
	}
	if _, err := NewUpdater(UpdateConfig{
		CurrentVersion: "1.0.0",
		Platform:       "plan9",
		Arch:           "amd64",
	}); err == nil {
		t.Fatal("不支持的平台应报错")
	}
	if _, err := NewUpdater(UpdateConfig{
		CurrentVersion: "1.0.0",
		ManifestURL:    "ftp://example.test/latest-mac.yml",
	}); err == nil {
		t.Fatal("非 http(s) 清单地址应报错")
	}
	updater, err := NewUpdater(UpdateConfig{
		CurrentVersion: "1.0.0",
		Platform:       "darwin",
		Arch:           "arm64",
		TempDir:        t.TempDir(),
	})
	if err != nil {
		t.Fatalf("默认更新源应可用: %v", err)
	}
	defer updater.Close()
	if updater.State() != UpdateUnchecked {
		t.Fatalf("初始状态 = %s", updater.State())
	}
}

func TestUpdaterCloseRemovesOwnedTempDir(t *testing.T) {
	content := []byte("2.1.0")
	server := newUpdateServer(t, channelManifest("2.1.0", content), 0, content)
	updater, err := NewUpdater(UpdateConfig{
		CurrentVersion: "1.0.0",
		ManifestURL:    manifestURLFor(server),
	})
	if err != nil {
		t.Fatalf("NewUpdater: %v", err)
	}
	if _, err := updater.Check(context.Background()); err != nil {
		t.Fatalf("Check: %v", err)
	}
	artifact, err := updater.Download(context.Background())
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	tempDir := updater.tempDir
	if err := updater.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := os.Stat(tempDir); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("临时目录未清理: %v", err)
	}
	if _, err := os.Stat(artifact); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("产物未清理: %v", err)
	}
}
