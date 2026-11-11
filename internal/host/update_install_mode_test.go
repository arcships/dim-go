package host

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallModeResolution(t *testing.T) {
	dir := t.TempDir()
	bundle := filepath.Join(dir, "DimAgent.app")
	if err := os.MkdirAll(bundle, 0o755); err != nil {
		t.Fatalf("建 .app: %v", err)
	}
	singleFile := filepath.Join(dir, "dimagent")
	if err := os.WriteFile(singleFile, []byte("binary"), 0o755); err != nil {
		t.Fatalf("写单文件: %v", err)
	}
	restart := func(context.Context) error { return nil }

	cases := []struct {
		name string
		cfg  UpdateConfig
		want any
	}{
		{
			name: "darwin 目录按 .app 适配器",
			cfg:  UpdateConfig{Platform: "darwin", TargetPath: bundle},
			want: &macInstaller{},
		},
		{
			name: "darwin 提供 Restart 视为单文件接线",
			cfg:  UpdateConfig{Platform: "darwin", TargetPath: singleFile, Restart: restart},
			want: &fileInstaller{},
		},
		{
			name: "linux 单文件默认 AppImage 适配器",
			cfg:  UpdateConfig{Platform: "linux", TargetPath: singleFile},
			want: &linuxInstaller{},
		},
		{
			name: "windows 默认 NSIS 适配器",
			cfg:  UpdateConfig{Platform: "windows"},
			want: &windowsInstaller{},
		},
		{
			name: "显式 file 覆盖平台",
			cfg:  UpdateConfig{Platform: "linux", InstallMode: InstallFile, TargetPath: singleFile, Restart: restart},
			want: &fileInstaller{},
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			cfg := testCase.cfg
			cfg.CurrentVersion = "1.0.0"
			cfg.ManifestURL = "https://example.test/latest-mac.yml"
			cfg.TempDir = t.TempDir()
			updater, err := NewUpdater(cfg)
			if err != nil {
				t.Fatalf("NewUpdater: %v", err)
			}
			defer updater.Close()
			install, err := updater.newInstaller()
			if err != nil {
				t.Fatalf("newInstaller: %v", err)
			}
			switch testCase.want.(type) {
			case *macInstaller:
				if _, ok := install.(*macInstaller); !ok {
					t.Fatalf("适配器 = %T，期望 *macInstaller", install)
				}
			case *fileInstaller:
				if _, ok := install.(*fileInstaller); !ok {
					t.Fatalf("适配器 = %T，期望 *fileInstaller", install)
				}
			case *linuxInstaller:
				if _, ok := install.(*linuxInstaller); !ok {
					t.Fatalf("适配器 = %T，期望 *linuxInstaller", install)
				}
			case *windowsInstaller:
				if _, ok := install.(*windowsInstaller); !ok {
					t.Fatalf("适配器 = %T，期望 *windowsInstaller", install)
				}
			}
		})
	}
}
