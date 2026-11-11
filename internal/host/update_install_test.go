package host

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func newReadyUpdater(t *testing.T, serverManifestVersion string, content []byte, cfg UpdateConfig) (*Updater, *syncBuffer) {
	t.Helper()
	server := newUpdateServer(t, channelManifest(serverManifestVersion, content), 0, content)
	updater, buf := newTestUpdater(t, manifestURLFor(server), cfg)
	if _, err := updater.Check(context.Background()); err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, err := updater.Download(context.Background()); err != nil {
		t.Fatalf("Download: %v", err)
	}
	return updater, buf
}

func directoryEntries(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("读目录 %s: %v", dir, err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		names = append(names, entry.Name())
	}
	return names
}

func TestInstallModeForPlatform(t *testing.T) {
	cases := map[string]InstallMode{
		"darwin":  InstallMacBundle,
		"linux":   InstallLinuxAppImage,
		"windows": InstallWindowsNSIS,
		"freebsd": InstallFile,
	}
	for platform, want := range cases {
		if got := installModeForPlatform(platform); got != want {
			t.Fatalf("%s 的安装模式 = %s，期望 %s", platform, got, want)
		}
	}
}

func cfgMac(bundle string, hooks InstallHooks, verify func(context.Context, string, string) error, shutdown func(context.Context) error) UpdateConfig {
	return UpdateConfig{
		InstallMode:     InstallMacBundle,
		TargetPath:      bundle,
		InstallHooks:    hooks,
		VerifySignature: verify,
		Shutdown:        shutdown,
	}
}

func TestMacInstallerReplacesBundleAfterVerification(t *testing.T) {
	content := []byte("2.1.0")
	dir := t.TempDir()
	bundle := filepath.Join(dir, "DimAgent.app")
	if err := os.MkdirAll(bundle, 0o755); err != nil {
		t.Fatalf("建旧 .app: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "version.txt"), []byte("1.0.0"), 0o644); err != nil {
		t.Fatalf("写旧版本: %v", err)
	}

	var events []string
	restartedVersion := ""
	hooks := InstallHooks{
		Run: func(_ context.Context, name string, args ...string) (string, error) {
			if name != macDitto {
				return "", nil
			}
			staging := args[len(args)-1]
			newApp := filepath.Join(staging, "DimAgent.app")
			if err := os.MkdirAll(newApp, 0o755); err != nil {
				return "", err
			}
			if err := os.WriteFile(filepath.Join(newApp, "version.txt"), []byte("2.1.0"), 0o644); err != nil {
				return "", err
			}
			events = append(events, "unpack")
			return "", nil
		},
		Spawn: func(_ context.Context, name string, args []string, _ []string) error {
			events = append(events, "restart")
			if name != macOpen || len(args) != 2 || args[0] != "-n" || args[1] != bundle {
				t.Fatalf("重启命令 = %s %v", name, args)
			}
			data, err := os.ReadFile(filepath.Join(bundle, "version.txt"))
			restartedVersion = string(data)
			return err
		},
	}
	verify := func(_ context.Context, oldApp, newApp string) error {
		events = append(events, "verify")
		if oldApp != bundle {
			t.Fatalf("校验的旧安装路径 = %q", oldApp)
		}
		if _, err := os.Stat(filepath.Join(newApp, "version.txt")); err != nil {
			return err
		}
		return nil
	}
	shutdown := func(context.Context) error {
		events = append(events, "shutdown")
		if got := readText(t, filepath.Join(bundle, "version.txt")); got != "1.0.0" {
			t.Fatalf("退出五阶段时安装包已被替换: %q", got)
		}
		return nil
	}

	updater, buf := newReadyUpdater(t, "2.1.0", content, cfgMac(bundle, hooks, verify, shutdown))
	if err := updater.Switch(context.Background()); err != nil {
		t.Fatalf("Switch: %v", err)
	}
	wantOrder := []string{"unpack", "verify", "shutdown", "restart"}
	if len(events) != len(wantOrder) {
		t.Fatalf("调用顺序 = %v", events)
	}
	for i := range wantOrder {
		if events[i] != wantOrder[i] {
			t.Fatalf("调用顺序 = %v，期望 %v", events, wantOrder)
		}
	}
	if got := readText(t, filepath.Join(bundle, "version.txt")); got != "2.1.0" {
		t.Fatalf("安装后版本 = %q，期望 2.1.0", got)
	}
	if restartedVersion != "2.1.0" {
		t.Fatalf("重启时读到的版本 = %q", restartedVersion)
	}
	if updater.State() != UpdateSwitching {
		t.Fatalf("切换后状态 = %s", updater.State())
	}
	for _, name := range directoryEntries(t, dir) {
		if strings.HasPrefix(name, ".dimagent-update-") || strings.Contains(name, ".old-") || strings.Contains(name, ".failed-") {
			t.Fatalf("安装后残留临时条目: %v", directoryEntries(t, dir))
		}
	}
	switched := recordsOf(decodeRecords(t, buf.String()), EventUpdateSwitch)
	if len(switched) != 1 || switched[0]["outcome"] != "ok" {
		t.Fatalf("切换记录 = %v", switched)
	}
}

func TestMacInstallerUnpackFailureKeepsOldBundle(t *testing.T) {
	content := []byte("2.1.0")
	dir := t.TempDir()
	bundle := filepath.Join(dir, "DimAgent.app")
	if err := os.MkdirAll(bundle, 0o755); err != nil {
		t.Fatalf("建旧 .app: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "version.txt"), []byte("1.0.0"), 0o644); err != nil {
		t.Fatalf("写旧版本: %v", err)
	}

	verifyCalled := false
	shutdownCalled := false
	hooks := InstallHooks{
		Run: func(_ context.Context, name string, _ ...string) (string, error) {
			if name == macDitto {
				return "", errors.New("坏包")
			}
			return "", nil
		},
		Spawn: func(context.Context, string, []string, []string) error {
			t.Fatal("解包失败不应重启")
			return nil
		},
	}
	cfg := cfgMac(bundle, hooks, func(context.Context, string, string) error {
		verifyCalled = true
		return nil
	}, func(context.Context) error {
		shutdownCalled = true
		return nil
	})

	updater, buf := newReadyUpdater(t, "2.1.0", content, cfg)
	if err := updater.Switch(context.Background()); err == nil {
		t.Fatal("解包失败应返回错误")
	}
	if verifyCalled || shutdownCalled {
		t.Fatalf("解包失败后不应继续: verify=%v shutdown=%v", verifyCalled, shutdownCalled)
	}
	if got := readText(t, filepath.Join(bundle, "version.txt")); got != "1.0.0" {
		t.Fatalf("旧安装包被改动: %q", got)
	}
	if updater.State() != UpdateReady {
		t.Fatalf("准备失败后状态 = %s，期望 ready_to_switch", updater.State())
	}
	for _, name := range directoryEntries(t, dir) {
		if strings.HasPrefix(name, ".dimagent-update-") {
			t.Fatalf("解包失败后残留暂存目录: %v", directoryEntries(t, dir))
		}
	}
	failed := recordsOf(decodeRecords(t, buf.String()), EventUpdateSwitch)
	if len(failed) != 1 || failed[0]["outcome"] != "failed" ||
		failed[0]["update_state"] != "ready_to_switch" ||
		failed[0]["classification"] != "non_retryable" {
		t.Fatalf("切换记录 = %v", failed)
	}
}

func TestMacInstallerSignatureFailureDoesNotReplace(t *testing.T) {
	content := []byte("2.1.0")
	dir := t.TempDir()
	bundle := filepath.Join(dir, "DimAgent.app")
	if err := os.MkdirAll(bundle, 0o755); err != nil {
		t.Fatalf("建旧 .app: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "version.txt"), []byte("1.0.0"), 0o644); err != nil {
		t.Fatalf("写旧版本: %v", err)
	}

	shutdownCalled := false
	hooks := InstallHooks{
		Run: func(_ context.Context, name string, args ...string) (string, error) {
			if name == macDitto {
				staging := args[len(args)-1]
				if err := os.MkdirAll(filepath.Join(staging, "DimAgent.app"), 0o755); err != nil {
					return "", err
				}
			}
			return "", nil
		},
		Spawn: func(context.Context, string, []string, []string) error {
			t.Fatal("签名失败不应重启")
			return nil
		},
	}
	cfg := cfgMac(bundle, hooks, func(context.Context, string, string) error {
		return errors.New("TeamIdentifier 不一致")
	}, func(context.Context) error {
		shutdownCalled = true
		return nil
	})

	updater, _ := newReadyUpdater(t, "2.1.0", content, cfg)
	if err := updater.Switch(context.Background()); err == nil {
		t.Fatal("签名校验失败应返回错误")
	}
	if shutdownCalled {
		t.Fatal("签名失败不应进入退出五阶段")
	}
	if got := readText(t, filepath.Join(bundle, "version.txt")); got != "1.0.0" {
		t.Fatalf("签名失败后旧安装包被改动: %q", got)
	}
	if updater.State() != UpdateReady {
		t.Fatalf("签名失败后状态 = %s", updater.State())
	}
	for _, name := range directoryEntries(t, dir) {
		if strings.HasPrefix(name, ".dimagent-update-") {
			t.Fatalf("签名失败后残留暂存目录: %v", directoryEntries(t, dir))
		}
	}
}

func TestMacInstallerRestartFailureRollsBack(t *testing.T) {
	content := []byte("2.1.0")
	dir := t.TempDir()
	bundle := filepath.Join(dir, "DimAgent.app")
	if err := os.MkdirAll(bundle, 0o755); err != nil {
		t.Fatalf("建旧 .app: %v", err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "version.txt"), []byte("1.0.0"), 0o644); err != nil {
		t.Fatalf("写旧版本: %v", err)
	}

	hooks := InstallHooks{
		Run: func(_ context.Context, name string, args ...string) (string, error) {
			if name == macDitto {
				staging := args[len(args)-1]
				newApp := filepath.Join(staging, "DimAgent.app")
				if err := os.MkdirAll(newApp, 0o755); err != nil {
					return "", err
				}
				if err := os.WriteFile(filepath.Join(newApp, "version.txt"), []byte("2.1.0"), 0o644); err != nil {
					return "", err
				}
			}
			return "", nil
		},
		Spawn: func(context.Context, string, []string, []string) error {
			return errors.New("启动失败")
		},
	}
	updater, buf := newReadyUpdater(t, "2.1.0", content,
		cfgMac(bundle, hooks, func(context.Context, string, string) error { return nil }, nil))
	if err := updater.Switch(context.Background()); err == nil {
		t.Fatal("重启失败应返回错误")
	}
	if got := readText(t, filepath.Join(bundle, "version.txt")); got != "1.0.0" {
		t.Fatalf("重启失败后未回滚: %q", got)
	}
	if updater.State() != UpdateSwitching {
		t.Fatalf("重启失败后状态 = %s", updater.State())
	}
	for _, name := range directoryEntries(t, dir) {
		if strings.Contains(name, ".old-") || strings.Contains(name, ".failed-") || strings.HasPrefix(name, ".dimagent-update-") {
			t.Fatalf("回滚后残留条目: %v", directoryEntries(t, dir))
		}
	}
	failed := recordsOf(decodeRecords(t, buf.String()), EventUpdateSwitch)
	if len(failed) != 1 || failed[0]["outcome"] != "failed" || failed[0]["update_state"] != "switching" {
		t.Fatalf("切换记录 = %v", failed)
	}
}

func TestLinuxInstallerReplacesAppImageAndRestartEnv(t *testing.T) {
	content := []byte("2.1.0")
	target := filepath.Join(t.TempDir(), "DimAgent.AppImage")
	if err := os.WriteFile(target, []byte("1.0.0"), 0o700); err != nil {
		t.Fatalf("写旧 AppImage: %v", err)
	}

	var spawnName string
	var spawnArgs, spawnEnv []string
	hooks := InstallHooks{
		Spawn: func(_ context.Context, name string, args []string, env []string) error {
			spawnName, spawnArgs, spawnEnv = name, args, env
			return nil
		},
	}
	updater, _ := newReadyUpdater(t, "2.1.0", content, UpdateConfig{
		InstallMode:  InstallLinuxAppImage,
		TargetPath:   target,
		InstallHooks: hooks,
	})
	if err := updater.Switch(context.Background()); err != nil {
		t.Fatalf("Switch: %v", err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if info.Mode().Perm() != 0o755 {
		t.Fatalf("AppImage 权限 = %v，期望 0755", info.Mode().Perm())
	}
	if got := readText(t, target); got != "2.1.0" {
		t.Fatalf("AppImage 内容 = %q", got)
	}
	resolved, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatalf("解析目标路径: %v", err)
	}
	if spawnName != resolved || len(spawnArgs) != 0 {
		t.Fatalf("重启命令 = %s %v，期望 %s", spawnName, spawnArgs, resolved)
	}
	found := false
	for _, value := range spawnEnv {
		if value == "APPIMAGE_SILENT_INSTALL=1" {
			found = true
		}
	}
	if !found {
		t.Fatal("重启环境缺少 APPIMAGE_SILENT_INSTALL=1")
	}
	if updater.State() != UpdateSwitching {
		t.Fatalf("切换后状态 = %s", updater.State())
	}
}

func TestNSISArgs(t *testing.T) {
	if got := strings.Join(nsisArgs(false), " "); got != "--updated /S" {
		t.Fatalf("默认参数 = %q", got)
	}
	if got := strings.Join(nsisArgs(true), " "); got != "--updated /S --force-run" {
		t.Fatalf("强制启动参数 = %q", got)
	}
}

func TestWindowsInstallerSpawnsInstallerOnce(t *testing.T) {
	content := []byte("installer")
	var spawns [][]string
	hooks := InstallHooks{
		Spawn: func(_ context.Context, name string, args []string, env []string) error {
			spawns = append(spawns, append([]string{name}, args...))
			return nil
		},
	}
	updater, _ := newReadyUpdater(t, "2.1.0", content, UpdateConfig{
		InstallMode:  InstallWindowsNSIS,
		InstallHooks: hooks,
	})
	if err := updater.Switch(context.Background()); err != nil {
		t.Fatalf("Switch: %v", err)
	}
	if len(spawns) != 1 {
		t.Fatalf("启动次数 = %d，期望 1（安装器负责重启）", len(spawns))
	}
	got := strings.Join(spawns[0][1:], " ")
	if got != "--updated /S" {
		t.Fatalf("安装器参数 = %q", got)
	}
	if !strings.HasSuffix(spawns[0][0], "app.bin") {
		t.Fatalf("安装器路径 = %q", spawns[0][0])
	}
}
