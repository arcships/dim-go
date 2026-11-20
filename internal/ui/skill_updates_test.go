package ui

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是 G-H5（技能可更新清单，G-H5-01…G-H5-06）的回归：入口与摘要、弹窗的
// 标题与空态、逐行技能与版本行、来源与变更说明、逐项更新、全部更新、更新中、
// 失败与重试，以及缺读数 / 缺通道时置灰并写原因。
//
// 读数是可选的 ui.SkillUpdatesFacts（protocol v3 没有「技能有新版本」的读取
// 方法），写走可选的 ui.SkillUpdateIntents（skills.update，按安装记录标识）。

// skillUpdatesSource 是带技能更新读数的连接。
type skillUpdatesSource struct {
	Source
	updates SkillUpdatesSnapshot
}

func (s skillUpdatesSource) SkillUpdates() SkillUpdatesSnapshot { return s.updates }

// skillUpdatesIntents 接住 skills.update 的写入；fail 里的标识返回原因。
type skillUpdatesIntents struct {
	Intents
	updated chan string
	fail    map[string]error
}

func (s *skillUpdatesIntents) UpdateSkill(_ context.Context, id string) error {
	if err, ok := s.fail[id]; ok && err != nil {
		return err
	}
	if s.updated != nil {
		s.updated <- id
	}
	return nil
}

// skillUpdateEntry 是一条可更新技能。
func skillUpdateEntry(id, name, version, latest, source, notes string) SkillUpdateSnapshot {
	return SkillUpdateSnapshot{
		ID: id, Name: name, Version: version, LatestVersion: latest,
		Source: source, Notes: notes,
	}
}

// skillUpdatesTester 把界面停在设置 › 插件设置，并带上技能更新读数。
func skillUpdatesTester(src SkillUpdatesSnapshot, intents Intents) (*App, *gui.Tester) {
	return pluginSettingsTester(skillUpdatesSource{Source: wiredSource(), updates: src}, intents)
}

// openSkillUpdatesDialog 点入口打开「可用更新」弹窗。
func openSkillUpdatesDialog(t *testing.T, app *App, tt *gui.Tester) {
	t.Helper()
	clickAccessRole(t, tt, gui.RoleButton, skillUpdatesOpenLabel)
	if !app.skillUpdates.open {
		t.Fatalf("技能更新弹窗没有打开：%v", tt.Texts())
	}
}

// TestSkillUpdatesNeedsFacts：没有技能更新读数时入口置灰并写原因，不画样例行
// （G-H5 的前置块）。
func TestSkillUpdatesNeedsFacts(t *testing.T) {
	// 只拿 Source 接口本身：不实现可选的 SkillUpdatesFacts，读数就没有。
	_, tt := pluginSettingsTester(marketlessSource{Source: wiredSource()}, nil)
	if !tt.HasText(skillUpdatesSectionTitle) {
		t.Fatalf("没有画「技能更新」：%v", tt.Texts())
	}
	if !tt.HasText(skillUpdatesNoFacts) {
		t.Fatalf("没有写缺读数的原因：%v", tt.Texts())
	}
	info, ok := tt.AccessByRole(gui.RoleButton, skillUpdatesOpenLabel)
	if !ok || !info.Disabled {
		t.Fatalf("没有读数时入口没有置灰：%+v（找到 %v）", info, ok)
	}
	if tt.HasText(skillUpdatesEmpty) {
		t.Fatal("没有读数时说成「所有技能均为最新版本」")
	}
}

// TestSkillUpdatesDialogRowsShowsVersions：弹窗标题「可用更新」，逐行给技能名、
// 版本行（v旧 → v新 / 只给 v新）、来源与变更说明；缺项写原因（G-H5-01 / 03 / 04）。
func TestSkillUpdatesDialogRowsShowsVersions(t *testing.T) {
	app, tt := skillUpdatesTester(SkillUpdatesSnapshot{Items: []SkillUpdateSnapshot{
		skillUpdateEntry("inst-1", "alpha", "1.0.0", "1.2.0", "owner/alpha", "修了勾选"),
		skillUpdateEntry("inst-2", "beta", "", "2.0.0", "", ""),
	}}, &skillUpdatesIntents{})
	openSkillUpdatesDialog(t, app, tt)

	if _, ok := tt.AccessByRole(gui.RoleDialog, skillUpdatesTitle); !ok {
		t.Fatalf("对话框名不是「可用更新」：%v", tt.Texts())
	}
	for _, want := range []string{
		skillUpdatesTitle,
		"alpha",
		"v1.0.0 → v1.2.0",
		"来源：owner/alpha",
		"变更：修了勾选",
		"beta",
		"v2.0.0",
		skillUpdatesNoSource,
		skillUpdatesNoNotes,
		"全部更新（2）",
	} {
		if !tt.HasText(want) {
			t.Fatalf("弹窗缺少 %q：%v", want, tt.Texts())
		}
	}
	if tt.HasText("v1.0.0 → v2.0.0") {
		t.Fatal("两个条目的版本串到了一起")
	}
	// 逐项按钮的读屏名带技能名（G-H5-05 无障碍）。
	for _, name := range []string{"更新 alpha", "更新 beta"} {
		if info, ok := tt.AccessByRole(gui.RoleButton, name); !ok || info.Disabled {
			t.Fatalf("找不到可点的 %q：%+v（找到 %v）", name, info, ok)
		}
	}
	if _, ok := tt.AccessByRole(gui.RoleButton, "全部更新，共 2 个"); !ok {
		t.Fatalf("「全部更新」的读屏名不对：%v", tt.Texts())
	}
}

// TestSkillUpdatesEmptyState：没有待更新项时入口仍可打开，弹窗里给空态句，且
// 没有「全部更新」与逐项按钮（G-H5-02）。
func TestSkillUpdatesEmptyState(t *testing.T) {
	app, tt := skillUpdatesTester(SkillUpdatesSnapshot{}, &skillUpdatesIntents{})
	if !tt.HasText(skillUpdatesEmpty) {
		t.Fatalf("没有待更新项时没写空态：%v", tt.Texts())
	}
	openSkillUpdatesDialog(t, app, tt)
	if info, ok := tt.AccessByRole(gui.RoleText, skillUpdatesEmpty); !ok || info.Disabled {
		t.Fatalf("弹窗里的空态不是一行文字：%+v（找到 %v）", info, ok)
	}
	if _, ok := tt.AccessByRole(gui.RoleButton, "全部更新，共 0 个"); ok {
		t.Fatal("没有待更新项时仍画了「全部更新」")
	}
	if _, ok := tt.AccessByRole(gui.RoleButton, "更新 alpha"); ok {
		t.Fatal("没有待更新项时仍画了逐项「更新」")
	}
}

// TestSkillUpdatesNeedsChannel：有读数但没有写通道时，逐项与全部更新都置灰并
// 写原因。
func TestSkillUpdatesNeedsChannel(t *testing.T) {
	app, tt := skillUpdatesTester(SkillUpdatesSnapshot{Items: []SkillUpdateSnapshot{
		skillUpdateEntry("inst-1", "alpha", "1.0.0", "1.2.0", "owner/alpha", ""),
	}}, nil)
	openSkillUpdatesDialog(t, app, tt)

	if info, ok := tt.AccessByRole(gui.RoleButton, "更新 alpha"); !ok || !info.Disabled {
		t.Fatalf("没有写通道时逐项更新没有置灰：%+v（找到 %v）", info, ok)
	}
	if info, ok := tt.AccessByRole(gui.RoleButton, "全部更新，共 1 个"); !ok || !info.Disabled {
		t.Fatalf("没有写通道时全部更新没有置灰：%+v（找到 %v）", info, ok)
	}
	if !tt.HasText(skillUpdatesNoChannel) {
		t.Fatalf("没有写缺通道的原因：%v", tt.Texts())
	}
}

// TestSkillUpdatesPerItemUpdate：点一行「更新」只更新该安装记录，成功后写回执
// （G-H5-05 交互）。
func TestSkillUpdatesPerItemUpdate(t *testing.T) {
	intents := &skillUpdatesIntents{updated: make(chan string, 4)}
	app, tt := skillUpdatesTester(SkillUpdatesSnapshot{Items: []SkillUpdateSnapshot{
		skillUpdateEntry("inst-1", "alpha", "1.0.0", "1.2.0", "owner/alpha", ""),
		skillUpdateEntry("inst-2", "beta", "1.0.0", "1.1.0", "", ""),
	}}, intents)
	openSkillUpdatesDialog(t, app, tt)

	clickAccessRole(t, tt, gui.RoleButton, "更新 alpha")
	select {
	case id := <-intents.updated:
		if id != "inst-1" {
			t.Fatalf("UpdateSkill 收到 %q，期望 inst-1", id)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("没有发出 skills.update")
	}
	waitPluginFrame(t, tt, func() bool { return !app.skillUpdates.updating["inst-1"] })
	if !tt.HasText("已更新 alpha") {
		t.Fatalf("逐项更新没有写回执：%v", tt.Texts())
	}
}

// TestSkillUpdatesFailureKeepsRowAndRetries：更新失败时该行保留并写原因，按钮
// 回可点；再点一次（重试）走同一条通道（G-H5-05 异常）。
func TestSkillUpdatesFailureKeepsRowAndRetries(t *testing.T) {
	intents := &skillUpdatesIntents{
		updated: make(chan string, 4),
		fail:    map[string]error{"inst-1": errors.New("下载失败")},
	}
	app, tt := skillUpdatesTester(SkillUpdatesSnapshot{Items: []SkillUpdateSnapshot{
		skillUpdateEntry("inst-1", "alpha", "1.0.0", "1.2.0", "", ""),
	}}, intents)
	openSkillUpdatesDialog(t, app, tt)

	clickAccessRole(t, tt, gui.RoleButton, "更新 alpha")
	waitPluginFrame(t, tt, func() bool {
		return strings.Contains(app.skillUpdates.rowErr["inst-1"], "下载失败")
	})
	if !tt.HasText("更新技能：下载失败") {
		t.Fatalf("失败原因没有上屏：%v", tt.Texts())
	}
	if info, ok := tt.AccessByRole(gui.RoleButton, "更新 alpha"); !ok || info.Disabled {
		t.Fatalf("失败后按钮没有回可点：%+v（找到 %v）", info, ok)
	}
	// 重试：清掉原因后再点一次，这次成功。
	intents.fail = nil
	clickAccessRole(t, tt, gui.RoleButton, "更新 alpha")
	waitPluginFrame(t, tt, func() bool {
		return app.skillUpdates.rowErr["inst-1"] == "" && app.skillUpdates.notice != ""
	})
	if !tt.HasText("已更新 alpha") {
		t.Fatalf("重试成功后没有回执：%v", tt.Texts())
	}
}

// TestSkillUpdatesUpdateAllPartialFailure：全部更新逐条发出；失败的行留着并写
// 原因，其余照发，最后给汇总（G-H5-06 交互 / 异常）。
func TestSkillUpdatesUpdateAllPartialFailure(t *testing.T) {
	intents := &skillUpdatesIntents{
		updated: make(chan string, 8),
		fail:    map[string]error{"inst-2": errors.New("仓库不可达")},
	}
	app, tt := skillUpdatesTester(SkillUpdatesSnapshot{Items: []SkillUpdateSnapshot{
		skillUpdateEntry("inst-1", "alpha", "1.0.0", "1.2.0", "", ""),
		skillUpdateEntry("inst-2", "beta", "1.0.0", "1.1.0", "", ""),
		skillUpdateEntry("inst-3", "gamma", "1.0.0", "1.1.0", "", ""),
	}}, intents)
	openSkillUpdatesDialog(t, app, tt)

	clickAccessRole(t, tt, gui.RoleButton, "全部更新，共 3 个")
	waitPluginFrame(t, tt, func() bool { return !app.skillUpdates.updatingAll })
	// 三条都发过：失败的那条不发成功回执，但排到了。
	sent := map[string]bool{}
	deadline := time.After(2 * time.Second)
	for len(sent) < 2 {
		select {
		case id := <-intents.updated:
			sent[id] = true
		case <-deadline:
			t.Fatalf("全部更新只发出了 %v", sent)
		}
	}
	if !sent["inst-1"] || !sent["inst-3"] {
		t.Fatalf("成功行没有逐条发出：%v", sent)
	}
	if !tt.HasText("1 个技能更新失败") {
		t.Fatalf("没有写失败汇总：%v", tt.Texts())
	}
	if !tt.HasText("更新失败：仓库不可达") {
		t.Fatalf("失败行没有保留原因：%v", tt.Texts())
	}
}

// TestSkillUpdatesUpdatingDisablesButtons：更新进行中逐项按钮换成「更新中…」并
// 禁用，全部更新同时禁用（G-H5-05 / G-H5-06 状态）。
func TestSkillUpdatesUpdatingDisablesButtons(t *testing.T) {
	app, tt := skillUpdatesTester(SkillUpdatesSnapshot{Items: []SkillUpdateSnapshot{
		skillUpdateEntry("inst-1", "alpha", "1.0.0", "1.2.0", "", ""),
		skillUpdateEntry("inst-2", "beta", "1.0.0", "1.1.0", "", ""),
	}}, &skillUpdatesIntents{})
	openSkillUpdatesDialog(t, app, tt)

	app.skillUpdatesStateOf().updating["inst-1"] = true
	app.skillUpdatesStateOf().updatingAll = true
	tt.Frame()
	if !tt.HasText(skillUpdatesUpdating) {
		t.Fatalf("更新中没有把按钮文案换成「更新中…」：%v", tt.Texts())
	}
	if info, ok := tt.Access(skillUpdatesUpdating); !ok || !info.Disabled {
		t.Fatalf("更新中的按钮没有置灰：%+v（找到 %v）", info, ok)
	}
	if info, ok := tt.AccessByRole(gui.RoleButton, "更新 beta"); !ok || !info.Disabled {
		t.Fatalf("全部更新进行中时别的行没有禁用：%+v（找到 %v）", info, ok)
	}
	if info, ok := tt.Access(skillUpdatesAllUpdating); !ok || !info.Disabled {
		t.Fatalf("全部更新进行中按钮没有换成「正在更新…」并禁用：%+v（找到 %v）", info, ok)
	}
}

// TestSkillUpdatesUpdateAllNotice：全部更新成功后写回执，逐行不再有失败原因
// （G-H5-06）。
func TestSkillUpdatesUpdateAllNotice(t *testing.T) {
	intents := &skillUpdatesIntents{updated: make(chan string, 8)}
	app, tt := skillUpdatesTester(SkillUpdatesSnapshot{Items: []SkillUpdateSnapshot{
		skillUpdateEntry("inst-1", "alpha", "1.0.0", "1.2.0", "", ""),
		skillUpdateEntry("inst-2", "beta", "1.0.0", "1.1.0", "", ""),
	}}, intents)
	openSkillUpdatesDialog(t, app, tt)

	clickAccessRole(t, tt, gui.RoleButton, "全部更新，共 2 个")
	waitPluginFrame(t, tt, func() bool { return !app.skillUpdates.updatingAll })
	if !tt.HasText("已更新 2 个技能") {
		t.Fatalf("全部更新没有写回执：%v", tt.Texts())
	}
	if app.skillUpdates.allErr != "" {
		t.Fatalf("全部成功却留了失败汇总：%q", app.skillUpdates.allErr)
	}
}

// TestSkillUpdatesReadError：读取失败时弹窗里写原因，不画样例行（G-H5-01 异常）。
func TestSkillUpdatesReadError(t *testing.T) {
	app, tt := skillUpdatesTester(SkillUpdatesSnapshot{ReadError: "skills.update 读取超时"},
		&skillUpdatesIntents{})
	if !tt.HasText("skills.update 读取超时") {
		t.Fatalf("没有写读取失败的原因：%v", tt.Texts())
	}
	openSkillUpdatesDialog(t, app, tt)
	if !tt.HasText(skillUpdatesReadError) {
		t.Fatalf("弹窗里没有写读取失败：%v", tt.Texts())
	}
	if _, ok := tt.AccessByRole(gui.RoleButton, "全部更新，共 0 个"); ok {
		t.Fatal("读取失败时仍画了「全部更新」")
	}
}

// TestSkillUpdatesNarrowWindowKeepsTitleAndButtons：窗口收到最窄档时，标题单行、
// 行与按钮仍完整在屏上（列表自己滚，不把按钮顶出去）（G-H5-01 窄窗）。
func TestSkillUpdatesNarrowWindowKeepsTitleAndButtons(t *testing.T) {
	src := skillUpdatesSource{Source: wiredSource(), updates: SkillUpdatesSnapshot{Items: []SkillUpdateSnapshot{
		skillUpdateEntry("inst-1", "alpha", "1.0.0", "1.2.0", "owner/alpha", "一条很长的变更说明，用来把行撑满"),
		skillUpdateEntry("inst-2", "beta", "1.0.0", "1.1.0", "", ""),
		skillUpdateEntry("inst-3", "gamma", "1.0.0", "1.1.0", "", ""),
	}}}
	app := NewApp(Options{State: StateDefault, Source: src, Intents: &skillUpdatesIntents{}})
	app.screen = ScreenSettings
	app.settingsSection = secPluginSettings
	tt := gui.NewTester(app.View, CompactWidth-40, WindowHeight)
	openSkillUpdatesDialog(t, app, tt)

	width := float32(CompactWidth - 40)
	title, ok := tt.AccessByRole(gui.RoleHeading, skillUpdatesTitle)
	if !ok || title.Bounds.X < 0 || title.Bounds.X+title.Bounds.W > width {
		t.Fatalf("窄窗里标题越界：%+v（找到 %v）", title, ok)
	}
	for _, name := range []string{"更新 alpha", "关闭", "全部更新，共 3 个"} {
		info, ok := tt.AccessByRole(gui.RoleButton, name)
		if !ok {
			t.Fatalf("窄窗里找不到按钮 %q", name)
		}
		r := info.Bounds
		if r.X < 0 || r.Y < 0 || r.X+r.W > width || r.Y+r.H > WindowHeight {
			t.Fatalf("窄窗里按钮 %q 越界：%+v", name, r)
		}
	}
}

// TestSkillUpdatesFormattingHelpers：版本行只按读数写，不自己比较版本号。
func TestSkillUpdatesFormattingHelpers(t *testing.T) {
	cases := []struct {
		item SkillUpdateSnapshot
		want string
	}{
		{skillUpdateEntry("i", "a", "1.0.0", "1.2.0", "", ""), "v1.0.0 → v1.2.0"},
		{skillUpdateEntry("i", "a", "", "1.2.0", "", ""), "v1.2.0"},
		{skillUpdateEntry("i", "a", "1.0.0", "", "", ""), ""},
		{skillUpdateEntry("i", "a", "v2", "v3", "", ""), "v2 → v3"},
	}
	for _, c := range cases {
		if got := skillUpdateVersionLine(c.item); got != c.want {
			t.Fatalf("版本行 %q → %q，期望 %q", c.item.Version, got, c.want)
		}
	}
	if got := skillUpdatesChannelNote(true); got != "" {
		t.Fatalf("有写通道时仍给原因：%q", got)
	}
	if got := skillUpdatesChannelNote(false); got != skillUpdatesNoChannel {
		t.Fatalf("缺通道的原因不对：%q", got)
	}
	if msg := fmt.Sprintf(skillUpdatesUpdateFmt, "alpha"); msg != "更新 alpha" {
		t.Fatalf("逐项按钮的读屏名不对：%q", msg)
	}
}
