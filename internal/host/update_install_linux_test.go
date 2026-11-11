package host

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

// TestLinuxInstallerReplacesSymlinkTarget 覆盖符号链接安装位置：替换真实文件，链接本身保留。
func TestLinuxInstallerReplacesSymlinkTarget(t *testing.T) {
	content := []byte("2.1.0")
	dir := t.TempDir()
	realTarget := filepath.Join(dir, "DimAgent.real.AppImage")
	if err := os.WriteFile(realTarget, []byte("1.0.0"), 0o700); err != nil {
		t.Fatalf("写真实 AppImage: %v", err)
	}
	link := filepath.Join(dir, "DimAgent.AppImage")
	if err := os.Symlink(realTarget, link); err != nil {
		t.Fatalf("建符号链接: %v", err)
	}

	spawnName := ""
	hooks := InstallHooks{
		Spawn: func(_ context.Context, name string, _ []string, _ []string) error {
			spawnName = name
			return nil
		},
	}
	updater, _ := newReadyUpdater(t, "2.1.0", content, UpdateConfig{
		InstallMode:  InstallLinuxAppImage,
		TargetPath:   link,
		InstallHooks: hooks,
	})
	if err := updater.Switch(context.Background()); err != nil {
		t.Fatalf("Switch: %v", err)
	}
	if got := readText(t, realTarget); got != "2.1.0" {
		t.Fatalf("真实文件内容 = %q", got)
	}
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatalf("Lstat 链接: %v", err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("符号链接被替换成了普通文件")
	}
	resolved, err := filepath.EvalSymlinks(link)
	if err != nil {
		t.Fatalf("解析链接: %v", err)
	}
	if spawnName != resolved {
		t.Fatalf("重启路径 = %q，期望 %q", spawnName, resolved)
	}
}
