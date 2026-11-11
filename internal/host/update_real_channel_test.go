package host

import (
	"context"
	"os"
	"testing"
	"time"
)

// TestRealChannelCheck 只读取真实更新渠道的版本清单，验证检查路径。
// 默认跳过；设置 D10_REAL_CHANNEL=1 才访问网络。不下载产物，也不改任何安装文件。
func TestRealChannelCheck(t *testing.T) {
	if os.Getenv("D10_REAL_CHANNEL") != "1" {
		t.Skip("设置 D10_REAL_CHANNEL=1 才访问真实更新渠道")
	}
	platforms := []struct {
		platform string
		arch     string
	}{
		{"darwin", "arm64"},
		{"darwin", "amd64"},
		{"windows", "amd64"},
		{"linux", "amd64"},
	}
	for _, target := range platforms {
		target := target
		t.Run(target.platform+"/"+target.arch, func(t *testing.T) {
			manifestURL, err := UpdateManifestURL("", "", target.platform, target.arch)
			if err != nil {
				t.Fatalf("推导清单地址: %v", err)
			}
			updater, err := NewUpdater(UpdateConfig{
				CurrentVersion: "0.0.1",
				ManifestURL:    manifestURL,
				Platform:       target.platform,
				Arch:           target.arch,
				TempDir:        t.TempDir(),
				Timeouts:       UpdateTimeouts{Check: 20 * time.Second},
			})
			if err != nil {
				t.Fatalf("NewUpdater: %v", err)
			}
			defer updater.Close()

			status, err := updater.Check(context.Background())
			if err != nil {
				t.Fatalf("Check: %v", err)
			}
			if status.State != UpdateAvailable {
				t.Fatalf("状态 = %s，期望 available", status.State)
			}
			if status.AvailableVersion == "" {
				t.Fatal("缺少版本源给出的版本号")
			}
			if _, err := parseSemver(status.AvailableVersion); err != nil {
				t.Fatalf("版本源版本号不合法: %v", err)
			}
			if status.ArtifactPath != "" {
				t.Fatalf("检查阶段不应下载产物: %q", status.ArtifactPath)
			}
		})
	}
}
