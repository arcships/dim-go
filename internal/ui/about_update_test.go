package ui

import (
	"context"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// updateTestSource 给关于页一份更新事实；其余事实走 fakeSource。
type updateTestSource struct {
	*fakeSource
	snapshot UpdateSnapshot
}

func (s *updateTestSource) Update() UpdateSnapshot { return s.snapshot }

// updateTestIntents 记录关于页发出的更新动作；其余入口留在嵌入的 nil 接口上。
type updateTestIntents struct {
	Intents
	checked    chan struct{}
	downloaded chan struct{}
	installed  chan struct{}
	feed       chan string
}

func newUpdateTestIntents() *updateTestIntents {
	return &updateTestIntents{
		checked:    make(chan struct{}, 1),
		downloaded: make(chan struct{}, 1),
		installed:  make(chan struct{}, 1),
		feed:       make(chan string, 1),
	}
}

func (u *updateTestIntents) CheckUpdate(context.Context) error {
	u.checked <- struct{}{}
	return nil
}

func (u *updateTestIntents) DownloadUpdate(context.Context) error {
	u.downloaded <- struct{}{}
	return nil
}

func (u *updateTestIntents) InstallUpdate(context.Context) error {
	u.installed <- struct{}{}
	return nil
}

func (u *updateTestIntents) SetUpdateFeed(_ context.Context, feedURL string) error {
	u.feed <- feedURL
	return nil
}

// openAboutForTest 重置本屏草稿并渲染一帧关于页。
func openAboutForTest(t *testing.T, app *App) *gui.Tester {
	t.Helper()
	aboutScreen.feedDraft, aboutScreen.feedDirty, aboutScreen.feedError = "", false, ""
	app.screen, app.settingsSection = ScreenSettings, secAbout
	return gui.NewTester(app.View, WindowWidth, WindowHeight)
}

// TestAboutUnconfiguredUpdateSourceDisabled 核对没有配置更新源时显示
// 「未配置更新源」并禁用检查入口：不假装已是最新。
func TestAboutUnconfiguredUpdateSourceDisabled(t *testing.T) {
	src := &updateTestSource{
		fakeSource: &fakeSource{},
		snapshot:   UpdateSnapshot{Configured: false, Phase: "idle"},
	}
	app := NewApp(Options{State: StateDefault, Source: src, Intents: newUpdateTestIntents()})
	tt := openAboutForTest(t, app)

	if !tt.HasText(abtCopyNoSource) {
		t.Fatalf("未配置更新源时没有显示 %q", abtCopyNoSource)
	}
	if tt.HasText(abtCopyLatest) {
		t.Fatal("未配置更新源时不该显示「已是最新版本」")
	}
	info, ok := tt.Access(abtCopyCheck)
	if !ok {
		t.Fatal("未配置更新源时检查入口应保留")
	}
	if !info.Disabled {
		t.Fatal("未配置更新源时检查入口应禁用")
	}
}

// TestAboutAvailableUpdateOffersDownload 核对有新版本时显示版本、说明与下载入口。
func TestAboutAvailableUpdateOffersDownload(t *testing.T) {
	src := &updateTestSource{
		fakeSource: &fakeSource{},
		snapshot: UpdateSnapshot{
			Configured:       true,
			FeedURL:          "https://example.com/updates/latest.json",
			Phase:            "available",
			CurrentVersion:   "1.0.0",
			AvailableVersion: "1.4.0",
			Notes:            "修了下载校验",
		},
	}
	intents := newUpdateTestIntents()
	app := NewApp(Options{State: StateDefault, Source: src, Intents: intents})
	tt := openAboutForTest(t, app)

	if !tt.HasText(abtCopyAvailable) {
		t.Fatal("有新版本时没有显示状态行")
	}
	if !tt.HasText("1.4.0") {
		t.Fatal("没有显示可用版本")
	}
	if !tt.HasText(abtCopyNotes + ": 修了下载校验") {
		t.Fatal("没有显示发布说明")
	}
	if err := tt.Click(abtCopyDownload); err != nil {
		t.Fatalf("点不到下载入口：%v", err)
	}
	select {
	case <-intents.downloaded:
	default:
		t.Fatal("点下载没有发出下载动作")
	}
}

// TestAboutDownloadedUpdateInstallDisabled 核对已下载但没有宿主安装能力时
// 「重启安装」置灰并写原因，不写假回执。
func TestAboutDownloadedUpdateInstallDisabled(t *testing.T) {
	src := &updateTestSource{
		fakeSource: &fakeSource{},
		snapshot: UpdateSnapshot{
			Configured:       true,
			Phase:            "downloaded",
			CurrentVersion:   "1.0.0",
			AvailableVersion: "1.4.0",
			Installable:      false,
			InstallReason:    NoteUpdateInstall,
		},
	}
	intents := newUpdateTestIntents()
	app := NewApp(Options{State: StateDefault, Source: src, Intents: intents})
	tt := openAboutForTest(t, app)

	info, ok := tt.Access(abtCopyInstall)
	if !ok {
		t.Fatal("已下载时「重启安装」入口应保留")
	}
	if !info.Disabled {
		t.Fatal("没有宿主安装能力时「重启安装」应禁用")
	}
	if !tt.HasText(NoteUpdateInstall) {
		t.Fatal("安装不可用时没有写原因")
	}
	_ = tt.Click(abtCopyInstall)
	select {
	case <-intents.installed:
		t.Fatal("置灰的安装入口仍发出了动作")
	default:
	}
}

// TestAboutErrorOffersRetry 核对检查失败时显示原因并给重试入口。
func TestAboutErrorOffersRetry(t *testing.T) {
	src := &updateTestSource{
		fakeSource: &fakeSource{},
		snapshot: UpdateSnapshot{
			Configured: true,
			Phase:      "error",
			Err:        "更新源返回 500",
		},
	}
	intents := newUpdateTestIntents()
	app := NewApp(Options{State: StateDefault, Source: src, Intents: intents})
	tt := openAboutForTest(t, app)

	if !tt.HasText("更新源返回 500") {
		t.Fatal("检查失败没有显示原因")
	}
	if err := tt.Click(abtCopyRetry); err != nil {
		t.Fatalf("点不到重试入口：%v", err)
	}
	select {
	case <-intents.checked:
	default:
		t.Fatal("点重试没有重新检查")
	}
}

// TestAboutDemoKeepsAutoUpdateDisabled 核对没有实现 UpdateFacts 的 Source 仍按
// 「自动更新已禁用」渲染，不显示更新源输入。
func TestAboutDemoKeepsAutoUpdateDisabled(t *testing.T) {
	app := NewApp(Options{State: StateDefault, Source: &fakeSource{}})
	tt := openAboutForTest(t, app)

	if !tt.HasText(abtCopyDisabled) {
		t.Fatal("没有更新能力时没有显示「自动更新已禁用」")
	}
	if _, ok := tt.AccessByRole(gui.RoleTextField, abtCopySourceLabel); ok {
		t.Fatal("没有更新能力时不该显示更新源输入")
	}
}
