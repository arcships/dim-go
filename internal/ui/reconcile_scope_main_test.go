package ui

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是主窗口「标题栏开关 / 消息内搜索 / 状态条」与全局面板「提问卡」这一批
// 的可执行证据：每条断言落在屏上看得见的结果或视图状态上，不看实现细节。

// —— 消息内搜索 ——

// TestSessionSearchLongQueryKeepsWidth：查询词极长时横向滚动，浮层与输入框宽度
// 都不变（R-TB-061）。
func TestSessionSearchLongQueryKeepsWidth(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	openSessionSearchForTest(t, tt)
	panel := func() gui.Rect {
		rect, ok := tt.Find("会话内搜索")
		if !ok {
			t.Fatalf("找不到搜索浮层：%v", tt.Texts())
		}
		return rect
	}
	before := panel()
	inputBefore, ok := tt.Find(sessionSearchLabel)
	if !ok {
		t.Fatalf("找不到搜索输入框：%v", tt.Texts())
	}

	long := strings.Repeat("很长的查询词", 40) // 240 字，超过 200
	searchFor(t, tt, long)
	if len([]rune(app.sessionSearch.query)) <= 200 {
		t.Fatalf("查询没到 200 字：%d", len([]rune(app.sessionSearch.query)))
	}
	// 宽度不变（高度会因结果行出现而变化，那是另一条需求）。
	after := panel()
	if after.W != before.W {
		t.Fatalf("长查询改变了浮层宽度：%v → %v", before, after)
	}
	inputAfter, ok := tt.Find(sessionSearchLabel)
	if !ok {
		t.Fatalf("长查询后找不到输入框：%v", tt.Texts())
	}
	if inputAfter.W != inputBefore.W || inputAfter.H != inputBefore.H {
		t.Fatalf("长查询把输入框撑开了：%v → %v", inputBefore, inputAfter)
	}
}

// TestSessionSearchCloseClearsHighlights：关闭浮层清掉全部高亮与当前项，下一次
// 打开是干净状态（R-TB-071）。
func TestSessionSearchCloseClearsHighlights(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	openSessionSearchForTest(t, tt)
	searchFor(t, tt, "排队")
	if len(app.sessionSearch.byUnit) == 0 {
		t.Fatal("查询应有高亮区间")
	}
	if err := tt.Click("关闭搜索"); err != nil {
		t.Fatalf("点不到关闭搜索：%v", err)
	}
	tt.Frame()
	if app.sessionSearch.open {
		t.Fatal("关闭键没有关闭浮层")
	}
	if len(app.sessionSearch.byUnit) != 0 || len(app.sessionSearch.hits) != 0 ||
		app.sessionSearch.current != -1 {
		t.Fatalf("关闭后仍留着高亮：units=%d current=%d",
			len(app.sessionSearch.byUnit), app.sessionSearch.current)
	}
	// 再打开：没有残留高亮，查询从空开始。
	openSessionSearchForTest(t, tt)
	if len(app.sessionSearch.byUnit) != 0 {
		t.Fatalf("重开后仍有高亮区间：%d", len(app.sessionSearch.byUnit))
	}
	if app.sessionSearch.query != "" {
		t.Fatalf("重开后查询 = %q，期望空", app.sessionSearch.query)
	}
}

// TestSessionSearchQueryResetsIndex：换查询词后序号归零并定位首个命中，不沿用
// 上一次的序号（R-TB-088）。
func TestSessionSearchQueryResetsIndex(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	openSessionSearchForTest(t, tt)
	searchFor(t, tt, "排队")
	if len(app.sessionSearch.hits) < 2 {
		t.Fatalf("命中数不足：%d", len(app.sessionSearch.hits))
	}
	tt.Key(0, gui.KeyEnter)
	tt.Frame()
	if app.sessionSearch.current != 1 {
		t.Fatalf("回车后当前项 = %d，期望 1", app.sessionSearch.current)
	}
	searchFor(t, tt, "发送")
	if app.sessionSearch.current != 0 {
		t.Fatalf("换查询后当前项 = %d，期望归零", app.sessionSearch.current)
	}
	if len(app.sessionSearch.hits) > 0 && !tt.HasText(fmt.Sprintf("1 / %d 个结果", len(app.sessionSearch.hits))) {
		t.Fatalf("换查询后计数没有从 1 开始：%v", tt.Texts())
	}
}

// TestSessionSearchCountFollowsNewContent：时间线新增内容后计数按当前命中集合
// 重算，不保留打开搜索时的快照（R-TB-100）。
func TestSessionSearchCountFollowsNewContent(t *testing.T) {
	src := wiredSource()
	src.session.Rows = []RowSnapshot{
		{Kind: RowUser, Text: "排队中的那条消息"},
	}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()

	openSessionSearchForTest(t, tt)
	searchFor(t, tt, "排队")
	total := len(app.sessionSearch.hits)
	if total == 0 {
		t.Fatalf("首帧应有命中：%v", tt.Texts())
	}
	if !tt.HasText(fmt.Sprintf("1 / %d 个结果", total)) {
		t.Fatalf("首个计数不对：%v", tt.Texts())
	}

	// 新消息进来（含同一个词）：总数跟着变。
	src.session.Rows = append(src.session.Rows, RowSnapshot{Kind: RowUser, Text: "又一条排队"})
	tt.Frame()
	if got := len(app.sessionSearch.hits); got != total+1 {
		t.Fatalf("新增内容后命中 = %d，期望 %d", got, total+1)
	}
	if !tt.HasText(fmt.Sprintf("1 / %d 个结果", total+1)) {
		t.Fatalf("计数没有重算：%v", tt.Texts())
	}
}

// —— 状态条 ——

// permFailIntents 让切档失败，用来验证「文案回退到切换前档位并提示」。
type permFailIntents struct {
	stubBaseIntents
	tries chan string
}

func (s *permFailIntents) SetPermissionTier(_ context.Context, tier string) error {
	s.tries <- tier
	return fmt.Errorf("权限服务暂不可用")
}

// TestStatusPermissionFailureKeepsTier：切档失败时按钮文案留在切换前的档位，并在
// 标题下方给出原因（R-SB-028）。
func TestStatusPermissionFailureKeepsTier(t *testing.T) {
	src := wiredSource()
	stub := &permFailIntents{tries: make(chan string, 2)}
	app := NewApp(Options{State: StateDefault, Source: src, Intents: stub})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()

	before := app.permissionLabel()
	if err := tt.Click("权限：" + before); err != nil {
		t.Fatalf("点不到权限按钮：%v", err)
	}
	tt.Frame()
	if err := tt.Click("权限：默认权限"); err != nil {
		t.Fatalf("点不到菜单项：%v", err)
	}
	tt.Frame()
	select {
	case <-stub.tries:
	case <-time.After(2 * time.Second):
		t.Fatal("没有发出 SetPermissionTier")
	}
	waitFrame(t, tt, "切档失败的提示落地", func() bool {
		return app.intentErr != ""
	})
	if got := app.permissionLabel(); got != before {
		t.Fatalf("失败后档位文案 = %q，期望留在 %q", got, before)
	}
	if !strings.Contains(app.intentErr, "权限服务暂不可用") {
		t.Fatalf("失败原因没有写出：%q", app.intentErr)
	}
}

// TestStatusUsagePanelStates：用量浮层的四分支——有明细（百分比 + 进度条）、无限
// 额度、无明细、错误（R-SB-208 / R-SB-218 / R-SB-228 / R-SB-238）。
func TestStatusUsagePanelStates(t *testing.T) {
	cases := []struct {
		name  string
		usage UsageSnapshot
		want  string
	}{
		{"有明细", UsageSnapshot{Available: true, Used: 39, Limit: 100}, "月度用量 39%"},
		{"无限额度", UsageSnapshot{Available: true, Used: 12, Limit: 0}, "无限额度"},
		{"无明细", UsageSnapshot{Available: false}, "暂无用量明细"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			src := wiredSource()
			src.usage = tc.usage
			app := NewApp(Options{State: StateDefault, Source: src})
			tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
			tt.Frame()
			if err := tt.Click("上下文用量 —"); err != nil {
				t.Fatalf("点不到用量表：%v", err)
			}
			tt.Frame()
			if !tt.HasText(tc.want) {
				t.Fatalf("用量浮层缺少 %q：%v", tc.want, tt.Texts())
			}
		})
	}

	// 有明细时浮层里还有一条进度条（RoleProgress）。
	src := wiredSource()
	src.usage = UsageSnapshot{Available: true, Used: 39, Limit: 100}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()
	if err := tt.Click("上下文用量 —"); err != nil {
		t.Fatalf("点不到用量表：%v", err)
	}
	tt.Frame()
	if info, ok := tt.AccessByRole(gui.RoleProgress, "套餐用量"); !ok || info.Value != "~39%" {
		t.Fatalf("套餐用量进度条不对：%+v（ok=%v）", info, ok)
	}
}

// —— 提问卡 ——

// TestQuestionMultiSelectKeyboard：多选项用 Tab 移焦、空格勾选/取消，勾选状态
// 落在视图状态上（G-D1-12）。
func TestQuestionMultiSelectKeyboard(t *testing.T) {
	snap := QuestionSnapshot{ID: "q-multi", Questions: []QuestionItemSnapshot{{
		ID: "checks", Title: "选择检查项", Options: []string{"构建", "单测"}, Multiple: true,
	}}}
	app, tt := questionTester(t, snap, nil)
	tt.Frame()

	// 点一下「构建」：勾选框自己拿焦点并勾上。
	if err := tt.Click("构建"); err != nil {
		t.Fatalf("点不到复选项：%v", err)
	}
	tt.Frame()
	if info, ok := tt.Access("构建"); !ok || !info.Focused {
		t.Fatalf("点选后焦点不在勾选框上：%+v（ok=%v）", info, ok)
	}
	// 空格取消。
	tt.Key(0, gui.KeySpace)
	tt.Frame()
	if app.question.selected[0]["构建"] {
		t.Fatal("空格没有取消勾选")
	}
	// 再按空格勾上。
	tt.Key(0, gui.KeySpace)
	tt.Frame()
	if !app.question.selected[0]["构建"] {
		t.Fatalf("空格没有勾上「构建」：%v", app.question.selected[0])
	}
	if info, ok := tt.Access("构建"); !ok || !info.Checked {
		t.Fatalf("勾选后读屏状态不对：%+v（ok=%v）", info, ok)
	}
	// 再按一次空格：取消勾选。
	tt.Key(0, gui.KeySpace)
	tt.Frame()
	if app.question.selected[0]["构建"] {
		t.Fatal("再按空格没有取消勾选")
	}
}

// TestQuestionCustomAnswerOverridesOptions：自定义答案非空时接管答案、选项选中态
// 清掉；清空自定义后回到选项选择（G-D1-15）。
func TestQuestionCustomAnswerOverridesOptions(t *testing.T) {
	snap := QuestionSnapshot{ID: "q-custom", Questions: []QuestionItemSnapshot{{
		ID: "env", Title: "选择部署目标", Options: []string{"暂存", "生产"}, AllowCustom: true,
	}}}
	app, tt := questionTester(t, snap, nil)
	tt.Frame()
	if err := tt.Click("暂存"); err != nil {
		t.Fatalf("点不到选项：%v", err)
	}
	tt.Frame()
	if got := app.question.custom[0]; got != "" {
		t.Fatalf("还没输入自定义答案：%q", got)
	}

	if err := tt.Click(questionCopyCustom); err != nil {
		t.Fatalf("点不到自定义答案框：%v", err)
	}
	tt.Type("灰度环境")
	tt.Frame()
	item := snap.Questions[0]
	if got := app.question.answerAt(0, item); got != "灰度环境" {
		t.Fatalf("自定义答案没有接管：%q", got)
	}
	if len(app.question.selected[0]) != 0 {
		t.Fatalf("自定义生效后选项仍选中：%v", app.question.selected[0])
	}

	// 清空自定义：回到选项选择。
	tt.Command("selectAll")
	tt.Key(0, gui.KeyBackspace)
	tt.Frame()
	if got := app.question.answerAt(0, item); got != "" {
		t.Fatalf("清空自定义后答案 = %q，期望空", got)
	}
	if err := tt.Click("生产"); err != nil {
		t.Fatalf("点不到选项：%v", err)
	}
	tt.Frame()
	if got := app.question.answerAt(0, item); got != "生产" {
		t.Fatalf("回到选项后答案 = %q，期望「生产」", got)
	}
}

// TestQuestionMergedAnswerDedups：多选提交时勾选在前、自定义选项在后，重复值只
// 出现一次（G-D1-16）。
func TestQuestionMergedAnswerDedups(t *testing.T) {
	snap := QuestionSnapshot{ID: "q-merge", Questions: []QuestionItemSnapshot{{
		ID: "checks", Title: "选择检查项", Options: []string{"构建", "单测"},
		Multiple: true, AllowCustom: true,
	}}}
	app, tt := questionTester(t, snap, nil)
	tt.Frame()
	if err := tt.Click("构建"); err != nil {
		t.Fatalf("点不到选项：%v", err)
	}
	tt.Frame()
	if err := tt.Click(questionCopyCustomOption); err != nil {
		t.Fatalf("点不到自定义选项框：%v", err)
	}
	// 第一行与已勾选项重复，第二行是新值。
	tt.Type("构建\n静态检查")
	tt.Frame()
	item := snap.Questions[0]
	got := app.question.answerAt(0, item)
	want := "构建\n单测\n静态检查"
	if got != want && got != "构建\n静态检查" {
		t.Fatalf("合并后的答案 = %q，期望勾选在前、自定义在后且去重", got)
	}
	if strings.Count(got, "构建") != 1 {
		t.Fatalf("重复值没有去重：%q", got)
	}
	if strings.Index(got, "构建") > strings.Index(got, "静态检查") {
		t.Fatalf("自定义值排在了勾选项之前：%q", got)
	}
}

// —— 标题栏开关 ——

// TestTitleBarPanelAndSidebarToggles：标题栏右端的打开/关闭面板按钮随右栏开合换
// 形与文案；左栏的展开/收起键双向往返，草稿跟着主区走不丢字（R-TB-010 / R-TB-011 /
// R-TB-017 / R-TB-019 / R-TB-027 / R-TB-296 / R-TB-303）。
func TestTitleBarPanelAndSidebarToggles(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	app.draft = "没发出去的草稿"
	tt.Frame()

	if info, ok := tt.Access("打开上下文面板"); !ok || info.Name != "打开上下文面板" {
		t.Fatalf("标题栏缺少打开面板按钮：%+v（ok=%v）", info, ok)
	}
	if err := tt.Click("打开上下文面板"); err != nil {
		t.Fatalf("点不到打开面板：%v", err)
	}
	tt.Frame()
	if !app.rightOpen {
		t.Fatal("点打开面板没有打开右栏")
	}
	if info, ok := tt.Access("关闭上下文面板"); !ok || info.Name != "关闭上下文面板" {
		t.Fatalf("右栏打开后没有换成关闭按钮：%+v（ok=%v）", info, ok)
	}
	if err := tt.Click("关闭上下文面板"); err != nil {
		t.Fatalf("点不到关闭面板：%v", err)
	}
	tt.Frame()
	if app.rightOpen {
		t.Fatal("点关闭面板没有收起右栏")
	}

	// 左栏：收起后是 12px 边条，点它展开；再按 ⌘B 收起，往返都不丢草稿。
	if err := tt.Click("收起侧边栏 ⌘B"); err != nil {
		t.Fatalf("点不到收起侧边栏：%v", err)
	}
	tt.Frame()
	if !app.sidebarCollapsed {
		t.Fatal("点收起侧边栏没有收起")
	}
	if _, ok := tt.Access("展开侧边栏 ⌘B"); !ok {
		t.Fatalf("收起后没有留下展开入口：%v", tt.Texts())
	}
	if err := tt.Click("展开侧边栏 ⌘B"); err != nil {
		t.Fatalf("点不到展开侧边栏：%v", err)
	}
	tt.Frame()
	if app.sidebarCollapsed {
		t.Fatal("点展开侧边栏没有展开")
	}
	tt.Key(gui.Cmd, gui.KeyB)
	tt.Frame()
	if !app.sidebarCollapsed {
		t.Fatal("⌘B 没有收起左栏")
	}
	tt.Key(gui.Cmd, gui.KeyB)
	tt.Frame()
	if app.sidebarCollapsed {
		t.Fatal("再按 ⌘B 没有展开左栏")
	}
	if app.draft != "没发出去的草稿" {
		t.Fatalf("左右栏开合丢了草稿：%q", app.draft)
	}
}
