package host

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// feedJSON 拼一份 latest.json。artifact 用相对地址，验证按版本源地址解析。
func feedJSON(version, minVersion string, artifact []byte) string {
	return fmt.Sprintf(`{"version":%q,"notes":"修了下载校验","url":"/artifact/app.zip","sha256":%q,"minVersion":%q}`,
		version, sha256Hex(artifact), minVersion)
}

// newFeedServer 提供 latest.json 与产物；latest 为函数以便用例中途换版本源。
func newFeedServer(t *testing.T, latest func() string, artifact []byte) *httptest.Server {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/latest.json":
			_, _ = io.WriteString(w, latest())
		case "/artifact/app.zip":
			_, _ = w.Write(artifact)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(server.Close)
	return server
}

func newFeedUpdaterForTest(t *testing.T, feedURL string, cfg FeedUpdateConfig) *FeedUpdater {
	t.Helper()
	cfg.FeedURL = feedURL
	cfg.Dir = t.TempDir()
	cfg.CheckTimeout = 2 * time.Second
	cfg.DownloadTimeout = 2 * time.Second
	updater, err := NewFeedUpdater(cfg)
	if err != nil {
		t.Fatalf("NewFeedUpdater: %v", err)
	}
	t.Cleanup(func() { _ = updater.Close() })
	return updater
}

func TestNormalizeFeedURL(t *testing.T) {
	cases := []struct {
		raw  string
		want string
		bad  bool
	}{
		{"", "", false},
		{"  ", "", false},
		{"https://example.com/updates", "https://example.com/updates/latest.json", false},
		{"https://example.com/updates/", "https://example.com/updates/latest.json", false},
		{"https://example.com/latest.json", "https://example.com/latest.json", false},
		{"http://127.0.0.1:8080", "http://127.0.0.1:8080/latest.json", false},
		{"ftp://example.com/updates", "", true},
		{"example.com/updates", "", true},
		{"file:///tmp/latest.json", "", true},
	}
	for _, tc := range cases {
		got, err := NormalizeFeedURL(tc.raw)
		if tc.bad {
			if err == nil {
				t.Fatalf("NormalizeFeedURL(%q) 应失败", tc.raw)
			}
			continue
		}
		if err != nil {
			t.Fatalf("NormalizeFeedURL(%q): %v", tc.raw, err)
		}
		if got != tc.want {
			t.Fatalf("NormalizeFeedURL(%q) = %q，想要 %q", tc.raw, got, tc.want)
		}
	}
}

func TestParseFeed(t *testing.T) {
	artifact := []byte("dimagent 1.2.0")
	feed, err := ParseFeed("https://example.com/updates/latest.json", []byte(feedJSON("1.2.0", "1.0.0", artifact)))
	if err != nil {
		t.Fatalf("ParseFeed: %v", err)
	}
	if feed.Version != "1.2.0" || feed.MinVersion != "1.0.0" {
		t.Fatalf("版本字段不符：%+v", feed)
	}
	if feed.URL != "https://example.com/artifact/app.zip" {
		t.Fatalf("相对产物地址没有按版本源解析：%q", feed.URL)
	}
	if feed.SHA256 != sha256Hex(artifact) {
		t.Fatalf("sha256 = %q，想要 %q", feed.SHA256, sha256Hex(artifact))
	}
	if feed.Notes != "修了下载校验" {
		t.Fatalf("notes = %q", feed.Notes)
	}
}

func TestParseFeedRejectsIncompleteFacts(t *testing.T) {
	digest := sha256Hex([]byte("x"))
	cases := map[string]string{
		"缺版本号":            fmt.Sprintf(`{"url":"/a.zip","sha256":%q}`, digest),
		"版本号不合法":          fmt.Sprintf(`{"version":"1.2","url":"/a.zip","sha256":%q}`, digest),
		"缺产物地址":           fmt.Sprintf(`{"version":"1.2.0","sha256":%q}`, digest),
		"缺摘要":             `{"version":"1.2.0","url":"/a.zip"}`,
		"摘要不是 64 位":       `{"version":"1.2.0","url":"/a.zip","sha256":"abc"}`,
		"摘要不是十六进制":        `{"version":"1.2.0","url":"/a.zip","sha256":"` + strings.Repeat("z", 64) + `"}`,
		"minVersion 不合法":  `{"version":"1.2.0","url":"/a.zip","sha256":"` + digest + `","minVersion":"1.0"}`,
		"minVersion 高于版本": `{"version":"1.2.0","url":"/a.zip","sha256":"` + digest + `","minVersion":"2.0.0"}`,
		"不是 JSON":         `{not json`,
	}
	for name, body := range cases {
		if _, err := ParseFeed("https://example.com/latest.json", []byte(body)); err == nil {
			t.Fatalf("%s：应拒绝", name)
		}
	}
	if _, err := ParseFeed("https://example.com/latest.json", []byte(`{"version":"1.2.0","url":"file:///tmp/a.zip","sha256":"`+digest+`"}`)); err == nil {
		t.Fatal("file:// 产物地址应拒绝")
	}
}

func TestCompareVersionsPrerelease(t *testing.T) {
	cases := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"2.0.0", "1.9.9", 1},
		{"1.0.0-alpha", "1.0.0", -1},
		{"1.0.0", "1.0.0-beta", 1},
		{"1.0.0-alpha", "1.0.0-beta", -1},
		{"1.0.0-alpha.2", "1.0.0-alpha.10", -1},
		{"1.0.0-alpha", "1.0.0-alpha.1", -1},
		{"1.0.0-1", "1.0.0-alpha", -1},
		{"1.0.0+build.9", "1.0.0", 0},
	}
	for _, tc := range cases {
		got, err := CompareVersions(tc.a, tc.b)
		if err != nil {
			t.Fatalf("CompareVersions(%q, %q): %v", tc.a, tc.b, err)
		}
		if got != tc.want {
			t.Fatalf("CompareVersions(%q, %q) = %d，想要 %d", tc.a, tc.b, got, tc.want)
		}
	}
	if _, err := CompareVersions("1.0", "1.0.0"); err == nil {
		t.Fatal("不合法版本号应报错")
	}
}

// TestFeedUpdaterEndToEnd 用本地 httptest 起一个版本源：检查 → 下载 →
// sha256 校验 → 安装不可用。不访问外网。
func TestFeedUpdaterEndToEnd(t *testing.T) {
	artifact := []byte("dimagent 1.2.0 artifact")
	server := newFeedServer(t, func() string {
		return feedJSON("1.2.0", "", artifact)
	}, artifact)
	updater := newFeedUpdaterForTest(t, server.URL, FeedUpdateConfig{CurrentVersion: "1.0.0"})

	status, err := updater.Check(context.Background())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if status.State != FeedUpdateAvailable || status.AvailableVersion != "1.2.0" {
		t.Fatalf("检查结果 = %+v", status)
	}
	if status.Notes != "修了下载校验" {
		t.Fatalf("发布说明 = %q", status.Notes)
	}

	artifactPath, err := updater.Download(context.Background())
	if err != nil {
		t.Fatalf("Download: %v", err)
	}
	if filepath.Dir(artifactPath) != updater.dir {
		t.Fatalf("产物落在 %q，想要下载目录 %q", filepath.Dir(artifactPath), updater.dir)
	}
	got, err := os.ReadFile(artifactPath)
	if err != nil {
		t.Fatalf("读产物: %v", err)
	}
	if string(got) != string(artifact) {
		t.Fatal("产物内容与版本源不一致")
	}
	status = updater.Status()
	if status.State != FeedUpdateDownloaded || status.Percent != 100 {
		t.Fatalf("下载后相位 = %+v", status)
	}

	if status.Installable {
		t.Fatal("没有宿主安装能力时 Installable 应为 false")
	}
	if status.InstallReason == "" {
		t.Fatal("安装不可用时没有原因")
	}
	if err := updater.Install(context.Background()); !errors.Is(err, ErrFeedInstallUnavailable) {
		t.Fatalf("Install 错误 = %v，想要 ErrFeedInstallUnavailable", err)
	}
	if updater.State() != FeedUpdateDownloaded {
		t.Fatalf("安装不可用不该改相位：%s", updater.State())
	}
}

// TestFeedUpdaterDownloadVerifiesSHA256 核对摘要不一致时给原因、不落产物。
func TestFeedUpdaterDownloadVerifiesSHA256(t *testing.T) {
	artifact := []byte("真产物")
	server := newFeedServer(t, func() string {
		return fmt.Sprintf(`{"version":"1.2.0","url":"/artifact/app.zip","sha256":%q}`,
			sha256Hex([]byte("别的东西")))
	}, artifact)
	updater := newFeedUpdaterForTest(t, server.URL, FeedUpdateConfig{CurrentVersion: "1.0.0"})

	if _, err := updater.Check(context.Background()); err != nil {
		t.Fatalf("Check: %v", err)
	}
	if _, err := updater.Download(context.Background()); err == nil {
		t.Fatal("摘要不一致应失败")
	} else if !strings.Contains(err.Error(), "sha256") {
		t.Fatalf("失败原因没有提到 sha256：%v", err)
	}
	status := updater.Status()
	if status.State != FeedUpdateError || status.LastError == nil {
		t.Fatalf("失败后相位 = %+v", status)
	}
	if status.AvailableVersion != "1.2.0" {
		t.Fatalf("下载失败后应保留待下载版本，实际 %q", status.AvailableVersion)
	}
	entries, err := os.ReadDir(updater.dir)
	if err != nil {
		t.Fatalf("读下载目录: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("校验失败后目录不该留文件：%v", entries)
	}
}

// TestFeedUpdaterUpToDate 核对版本相等或更低时进入「已是最新」，不摆出可下载版本。
func TestFeedUpdaterUpToDate(t *testing.T) {
	artifact := []byte("artifact")
	server := newFeedServer(t, func() string {
		return feedJSON("1.0.0", "", artifact)
	}, artifact)
	updater := newFeedUpdaterForTest(t, server.URL, FeedUpdateConfig{CurrentVersion: "1.0.0"})

	status, err := updater.Check(context.Background())
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if status.State != FeedUpdateUpToDate {
		t.Fatalf("相位 = %s，想要 up_to_date", status.State)
	}
	if status.AvailableVersion != "" {
		t.Fatalf("已是最新时不该有可下载版本：%q", status.AvailableVersion)
	}
	if _, err := updater.Download(context.Background()); err == nil {
		t.Fatal("已是最新时下载应被拒绝")
	}
}

// TestFeedUpdaterCheckRetryAfterError 核对检查失败后重试能成功。
func TestFeedUpdaterCheckRetryAfterError(t *testing.T) {
	artifact := []byte("artifact")
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/latest.json" {
			http.NotFound(w, r)
			return
		}
		if requests.Add(1) == 1 {
			http.Error(w, "暂时不可用", http.StatusInternalServerError)
			return
		}
		_, _ = io.WriteString(w, feedJSON("1.2.0", "", artifact))
	}))
	t.Cleanup(server.Close)
	updater := newFeedUpdaterForTest(t, server.URL, FeedUpdateConfig{CurrentVersion: "1.0.0"})

	status, err := updater.Check(context.Background())
	if err == nil || status.State != FeedUpdateError {
		t.Fatalf("第一次检查应失败：status=%+v err=%v", status, err)
	}
	status, err = updater.Check(context.Background())
	if err != nil || status.State != FeedUpdateAvailable {
		t.Fatalf("重试应成功：status=%+v err=%v", status, err)
	}
}

// TestFeedUpdaterMinVersionRequiresReinstall 核对低于 minVersion 时不提供增量更新。
func TestFeedUpdaterMinVersionRequiresReinstall(t *testing.T) {
	artifact := []byte("artifact")
	server := newFeedServer(t, func() string {
		return feedJSON("2.0.0", "1.5.0", artifact)
	}, artifact)
	updater := newFeedUpdaterForTest(t, server.URL, FeedUpdateConfig{CurrentVersion: "1.0.0"})

	status, err := updater.Check(context.Background())
	if err == nil {
		t.Fatal("低于 minVersion 应失败")
	}
	if !strings.Contains(err.Error(), "最低版本") {
		t.Fatalf("失败原因没有说明最低版本：%v", err)
	}
	if status.State != FeedUpdateError || status.LastError == nil {
		t.Fatalf("相位 = %+v", status)
	}
}

// TestFeedUpdaterUnconfigured 核对没有更新源时不构造编排。
func TestFeedUpdaterUnconfigured(t *testing.T) {
	_, err := NewFeedUpdater(FeedUpdateConfig{
		CurrentVersion: "1.0.0",
		Dir:            t.TempDir(),
	})
	if !errors.Is(err, ErrFeedNotConfigured) {
		t.Fatalf("NewFeedUpdater 错误 = %v，想要 ErrFeedNotConfigured", err)
	}
}
