package uiwire

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/app"
	"github.com/arcships/dimagent-mygo/internal/host"
	"github.com/arcships/dimagent-mygo/internal/ui"
)

func newUpdateWire(t *testing.T) (*App, *app.App) {
	t.Helper()
	base, err := app.New(app.Config{
		DataDir:    t.TempDir(),
		ClientName: "update-wire-test",
		Version:    "1.0.0",
	})
	if err != nil {
		t.Fatalf("创建集成层：%v", err)
	}
	return New(base), base
}

// waitUpdatePhase 等后台动作把相位写回快照。
func waitUpdatePhase(t *testing.T, wire *App, want string) ui.UpdateSnapshot {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		snapshot := wire.Update()
		if snapshot.Phase == want {
			return snapshot
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("相位没有到 %s，最后是 %+v", want, wire.Update())
	return ui.UpdateSnapshot{}
}

// TestUpdateUnconfigured 核对没有配置更新源时读到的相位与拒绝动作：界面据此
// 显示「未配置更新源」并禁用检查入口。
func TestUpdateUnconfigured(t *testing.T) {
	wire, _ := newUpdateWire(t)
	snapshot := wire.Update()
	if snapshot.Configured {
		t.Fatalf("没有配置更新源却报告已配置：%+v", snapshot)
	}
	if snapshot.Phase != string(host.FeedUpdateIdle) {
		t.Fatalf("相位 = %q，想要 idle（未配置由 Configured 表达）", snapshot.Phase)
	}
	if err := wire.CheckUpdate(context.Background()); !errors.Is(err, host.ErrFeedNotConfigured) {
		t.Fatalf("CheckUpdate 错误 = %v，想要 ErrFeedNotConfigured", err)
	}
}

// TestUpdateFeedEndToEnd 用本地 httptest 版本源走完整条链路：保存更新源 →
// 检查 → 下载校验落到 `<数据目录>/updates/` → 安装不可用给原因。
func TestUpdateFeedEndToEnd(t *testing.T) {
	artifact := []byte("dimagent 1.2.0")
	sum := sha256.Sum256(artifact)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest.json":
			fmt.Fprintf(w, `{"version":"1.2.0","notes":"修了下载校验","url":"/app.zip","sha256":%q}`,
				hex.EncodeToString(sum[:]))
		case "/app.zip":
			_, _ = w.Write(artifact)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	wire, base := newUpdateWire(t)

	if err := wire.SetUpdateFeed(context.Background(), server.URL); err != nil {
		t.Fatalf("保存更新源：%v", err)
	}
	if got := base.Preferences().UpdateFeedURL; got != server.URL+"/latest.json" {
		t.Fatalf("更新源没有按 latest.json 规范化落盘：%q", got)
	}
	if snapshot := wire.Update(); !snapshot.Configured || snapshot.FeedURL != server.URL+"/latest.json" {
		t.Fatalf("快照没有跟上偏好：%+v", snapshot)
	}

	if err := wire.CheckUpdate(context.Background()); err != nil {
		t.Fatalf("检查更新：%v", err)
	}
	snapshot := waitUpdatePhase(t, wire, string(host.FeedUpdateAvailable))
	if snapshot.AvailableVersion != "1.2.0" || snapshot.Notes != "修了下载校验" {
		t.Fatalf("检查结果 = %+v", snapshot)
	}

	if err := wire.DownloadUpdate(context.Background()); err != nil {
		t.Fatalf("下载更新：%v", err)
	}
	snapshot = waitUpdatePhase(t, wire, string(host.FeedUpdateDownloaded))
	if snapshot.Percent != 100 {
		t.Fatalf("下载完成进度 = %d，想要 100", snapshot.Percent)
	}
	entries, err := os.ReadDir(filepath.Join(base.DataDir(), "updates"))
	if err != nil || len(entries) == 0 {
		t.Fatalf("下载没有落在 <数据目录>/updates：%v %v", entries, err)
	}
	written, err := os.ReadFile(filepath.Join(base.DataDir(), "updates", entries[0].Name()))
	if err != nil || string(written) != string(artifact) {
		t.Fatalf("产物内容不符：%v", err)
	}

	if snapshot.Installable {
		t.Fatal("本版没有宿主安装能力，Installable 应为 false")
	}
	if err := wire.InstallUpdate(context.Background()); err == nil {
		t.Fatal("没有安装能力时 InstallUpdate 应返回原因")
	} else if !strings.Contains(err.Error(), "自动安装") {
		t.Fatalf("安装失败原因不清楚：%v", err)
	}
	if got := wire.Update(); got.Phase != string(host.FeedUpdateDownloaded) {
		t.Fatalf("安装不可用不该改相位：%+v", got)
	}
}

// TestSetUpdateFeedRejectsBadAddress 核对不是 http/https 的地址不落盘。
func TestSetUpdateFeedRejectsBadAddress(t *testing.T) {
	wire, base := newUpdateWire(t)
	if err := wire.SetUpdateFeed(context.Background(), "ftp://example.com/updates"); err == nil {
		t.Fatal("ftp 地址应被拒绝")
	}
	if got := base.Preferences().UpdateFeedURL; got != "" {
		t.Fatalf("被拒绝的地址改动了偏好：%q", got)
	}
	if err := wire.SetUpdateFeed(context.Background(), ""); err != nil {
		t.Fatalf("清空更新源：%v", err)
	}
	if snapshot := wire.Update(); snapshot.Configured {
		t.Fatalf("清空后仍报告已配置：%+v", snapshot)
	}
}
