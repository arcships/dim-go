package ui

import (
	"context"
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件覆盖本批补齐的八块元素的交互断言（Tester 找元素 + 点/键/菜单）。
// 每块都只读事实或可丢的视图状态，不写服务端。

// queuedSource 造一个连着事实、带排队消息的会话。
func queuedSource(items ...QueuedMessageSnapshot) *fakeSource {
	src := wiredSource()
	src.conn = Connection{Phase: "connected"}
	src.names = []string{"排队面板"}
	src.hasSess = true
	src.session = SessionSnapshot{
		ID: "s1", Title: "排队面板", Running: true,
		Rows:   []RowSnapshot{{Kind: RowUser, Text: "第一轮"}},
		Queued: items,
	}
	return src
}

// TestQueuePanelRowsAndCollapse：面板标题是「{n} 条消息」，行显示正文与转向
// 状态；点标题折叠再展开；不可用的排序与撤回置灰并写原因（IN-35–IN-41）。
func TestQueuePanelRowsAndCollapse(t *testing.T) {
	src := queuedSource(
		QueuedMessageSnapshot{ID: "q1", Mode: "queue", Text: "再补一条错误分支的测试"},
		QueuedMessageSnapshot{ID: "q2", Mode: "steer", Text: "先把编译过的包跑一遍"},
	)
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	if _, ok := tt.Find("排队消息面板"); !ok {
		t.Fatalf("有没有排队消息都应该出现面板：%v", tt.Texts())
	}
	if !tt.HasText("2 条消息") {
		t.Fatalf("计数文案不对：%v", tt.Texts())
	}
	if !tt.HasText("再补一条错误分支的测试") {
		t.Fatal("排队消息的正文没有来自事实")
	}
	if !tt.HasText("正在转向") {
		t.Fatal("转向中的那条没有状态词")
	}

	if err := tt.Click("2 条消息"); err != nil {
		t.Fatalf("点不到计数行：%v", err)
	}
	tt.Frame()
	if tt.HasText("再补一条错误分支的测试") {
		t.Fatal("折叠后仍列出消息行")
	}
	if err := tt.Click("2 条消息"); err != nil {
		t.Fatalf("点不到计数行（展开）：%v", err)
	}
	tt.Frame()
	if !tt.HasText("再补一条错误分支的测试") {
		t.Fatal("展开后没有恢复消息行")
	}

	// 悬停待发行：把手与移除在，但置灰；原因写在读屏名里。
	row, ok := tt.Find("再补一条错误分支的测试")
	if !ok {
		t.Fatal("找不到排队消息行")
	}
	tt.Move(row.X+row.W/2, row.Y+row.H/2)
	tt.Frame()
	if info, ok := tt.Access("拖动排序（" + noteQueueReorder + "）"); !ok || !info.Disabled {
		t.Fatalf("排序把手没有按不可用画：%+v（找到 %v）", info, ok)
	}
	if info, ok := tt.Access("移除排队消息（" + noteQueueWithdraw + "）"); !ok || !info.Disabled {
		t.Fatalf("移除没有按不可用画：%+v（找到 %v）", info, ok)
	}
	if info, ok := tt.Access("转向（" + noteQueueSteer + "）"); !ok || !info.Disabled {
		t.Fatalf("转向没有按不可用画：%+v（找到 %v）", info, ok)
	}
}

// TestQueuePanelEditEntersComposer：点「编辑」把正文放进输入块并显示
// 「正在编辑排队消息」；取消后恢复；连线保存走失败分支并保留草稿（IN-40 / IN-436）。
func TestQueuePanelEditEntersComposer(t *testing.T) {
	src := queuedSource(
		QueuedMessageSnapshot{ID: "q1", Mode: "queue", Text: "再补一条错误分支的测试"},
	)
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	row, ok := tt.Find("再补一条错误分支的测试")
	if !ok {
		t.Fatal("找不到排队消息行")
	}
	tt.Move(row.X+row.W/2, row.Y+row.H/2)
	tt.Frame()
	if err := tt.Click("编辑排队消息（再补一条错误分支的测试）"); err != nil {
		t.Fatalf("点不到编辑：%v", err)
	}
	tt.Frame()
	if app.queueEditing != "q1" || app.draft != "再补一条错误分支的测试" {
		t.Fatalf("编辑没有把正文放进输入块：editing=%q draft=%q", app.queueEditing, app.draft)
	}
	if !tt.HasText("正在编辑排队消息") {
		t.Fatalf("输入块没有编辑说明：%v", tt.Texts())
	}
	// 连线时保存没有通道：写原因、草稿保留。
	if err := tt.Click("发送"); err != nil {
		t.Fatalf("点不到保存：%v", err)
	}
	tt.Frame()
	if app.draft == "" {
		t.Fatal("保存失败后草稿被清空")
	}
	if !tt.HasText(noteQueueEdit) {
		t.Fatalf("保存失败没有写原因：%v", tt.Texts())
	}
	if err := tt.Click("取消编辑排队消息"); err != nil {
		t.Fatalf("点不到取消：%v", err)
	}
	tt.Frame()
	if app.queueEditing != "" || app.draft != "" {
		t.Fatalf("取消后没有退出编辑：editing=%q draft=%q", app.queueEditing, app.draft)
	}
}

// TestQueuePanelRowsShowTextOrAttachmentCount：排队行的正文取快照正文的摘要（长正文
// 截断加省略号）；只有附件事实时写「附件 N 个」；两样都没有时才写「没有正文」。
// 没有正文的输入不能被一律标成「附件」——那会把一整列文字输入全说成附件。
func TestQueuePanelRowsShowTextOrAttachmentCount(t *testing.T) {
	long := "先看下构建日志第 1 行\n第二段  有空白的正文" +
		"再补一段很长的正文用来看截断，这段正文会超过八十个字符的上限，后面的内容只在编辑里可见，所以行里必须截断并加省略号，不能把整段都塞进一行"
	src := queuedSource(
		QueuedMessageSnapshot{ID: "q1", Mode: "queue", Text: long},
		QueuedMessageSnapshot{ID: "q2", Mode: "queue", Text: "", Attachments: 3},
		QueuedMessageSnapshot{ID: "q3", Mode: "queue", Text: "   "},
	)
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	summary := queueSummary(long)
	if summary == "" || !strings.HasSuffix(summary, "…") {
		t.Fatalf("长正文没有被截断成摘要：%q", summary)
	}
	if strings.Contains(summary, "\n") {
		t.Fatalf("摘要里还有换行：%q", summary)
	}
	if _, ok := tt.Find("排队消息：" + summary); !ok {
		t.Fatalf("排队行没有显示正文摘要 %q：%v", summary, tt.Texts())
	}
	if _, ok := tt.Find("排队消息：附件"); ok {
		t.Fatalf("有正文的排队行被标成了附件：%v", tt.Texts())
	}

	if !tt.HasText("附件 3 个") {
		t.Fatalf("只有附件事实的排队行没有写「附件 3 个」：%v", tt.Texts())
	}
	if _, ok := tt.Find("排队消息：" + noteQueueNoText); !ok {
		t.Fatalf("没有正文也没有附件的排队行没有写 %q：%v", noteQueueNoText, tt.Texts())
	}
	for _, text := range tt.Texts() {
		if !strings.HasPrefix(text, "排队消息：") {
			continue
		}
		if strings.TrimSpace(strings.TrimPrefix(text, "排队消息：")) == "附件" {
			t.Fatalf("排队行仍然只有「附件」两个字：%v", tt.Texts())
		}
	}
}

// TestQueueLineTextUsesFactsOnly 直接核对排队行的取值：正文摘要优先，附件条数次之，
// 都没有才说没有正文。
func TestQueueLineTextUsesFactsOnly(t *testing.T) {
	cases := []struct {
		name string
		item QueuedMessageSnapshot
		want string
	}{
		{"正文", QueuedMessageSnapshot{Text: "再补一条测试"}, "再补一条测试"},
		{"正文折叠空白", QueuedMessageSnapshot{Text: " 先跑\n一遍  "}, "先跑 一遍"},
		{"只有附件", QueuedMessageSnapshot{Attachments: 2}, "附件 2 个"},
		{"都没有", QueuedMessageSnapshot{}, noteQueueNoText},
	}
	for _, tc := range cases {
		if got := queueLineText(tc.item); got != tc.want {
			t.Errorf("%s：queueLineText = %q，想要 %q", tc.name, got, tc.want)
		}
	}
}

// TestShellRunPanelShowsReceipt：命令、模式、状态、输出、脚注都来自 shell.run
// 回执；折叠隐藏输出；结束后「关闭」收起面板（IN-45–IN-52）。
func TestShellRunPanelShowsReceipt(t *testing.T) {
	src := wiredSource()
	src.conn = Connection{Phase: "connected"}
	src.names = []string{"命令面板"}
	src.hasSess = true
	src.session = SessionSnapshot{
		ID: "s1", Title: "命令面板",
		Rows: []RowSnapshot{{Kind: RowUser, Text: "跑一下测试"}},
		ShellRun: &ShellRunSnapshot{
			RunID: "run-1", Command: "go test ./internal/ui", Mode: "record",
			Status: "failed", Output: "--- FAIL: TestX\npanic: 断言失败",
			ExitCode: 2, HasExitCode: true, Truncated: true,
			Error: "退出码 2", Reason: "测试失败", Recorded: true,
		},
	}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	if _, ok := tt.Find("Shell 运行面板"); !ok {
		t.Fatalf("Shell 运行面板没有出现：%v", tt.Texts())
	}
	if _, ok := tt.Find("命令：go test ./internal/ui"); !ok {
		t.Fatalf("命令行没有来自回执：%v", tt.Texts())
	}
	if !tt.HasText("记录") {
		t.Fatalf("模式标签没有出现：%v", tt.Texts())
	}
	if _, ok := tt.Find("状态：失败"); !ok {
		t.Fatalf("状态文本不对：%v", tt.Texts())
	}
	if !tt.HasText("进程退出码 2") || !tt.HasText("输出已截断") ||
		!tt.HasText("错误：退出码 2") || !tt.HasText("原因：测试失败") {
		t.Fatalf("脚注不全：%v", tt.Texts())
	}
	if !tt.HasText("--- FAIL: TestX") {
		t.Fatalf("输出块没有内容：%v", tt.Texts())
	}
	// 结束后给「关闭」；点它收起面板。
	if _, ok := tt.Access("关闭命令面板"); !ok {
		t.Fatalf("结束后没有关闭入口：%v", tt.Texts())
	}
	if err := tt.Click("关闭命令面板"); err != nil {
		t.Fatalf("点不到关闭：%v", err)
	}
	tt.Frame()
	if _, ok := tt.Find("Shell 运行面板"); ok {
		t.Fatal("关闭后面板仍在")
	}
}

// TestShellRunPanelRunningAndCollapse：运行中给「停止」与旋转；折叠只收起内容。
func TestShellRunPanelRunningAndCollapse(t *testing.T) {
	app := NewApp(Options{State: StateDefault, Variant: VariantShellPanel})
	app.demoShellRun.Status = "running"
	app.demoShellRun.HasExitCode = false
	app.demoShellRun.Truncated = false
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	if _, ok := tt.Find("状态：运行中"); !ok {
		t.Fatalf("运行中的状态文本不对：%v", tt.Texts())
	}
	if _, ok := tt.Access("停止命令"); !ok {
		t.Fatalf("运行中没有停止入口：%v", tt.Texts())
	}
	if _, ok := tt.Access("关闭命令面板"); ok {
		t.Fatal("运行中不该有「关闭」")
	}
	if err := tt.Click("停止命令"); err != nil {
		t.Fatalf("点不到停止：%v", err)
	}
	tt.Frame()
	if _, ok := tt.Find("状态：已停止"); !ok {
		t.Fatalf("停止后状态没有更新：%v", tt.Texts())
	}
	if _, ok := tt.Access("关闭命令面板"); !ok {
		t.Fatalf("停止后没有关闭入口：%v", tt.Texts())
	}
	if err := tt.Click("命令：go test ./internal/ui -run TestComposerPanels -count=1"); err != nil {
		t.Fatalf("点不到命令标题：%v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if tt.HasText("--- FAIL: TestComposerPanels") {
		t.Fatal("折叠后输出仍在")
	}
	if _, ok := tt.Find("状态：已停止"); !ok {
		t.Fatal("折叠把整块面板收掉了")
	}
}

// TestInlineEditorSavesLocally：悬停点「编辑」进入行内编辑；发送保存并在气泡上
// 标「已编辑」；Esc 放弃编辑、正文不变（TL-31–TL-34）。
func TestInlineEditorSavesLocally(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	bubble, ok := tt.Find("我的消息")
	if !ok {
		t.Fatal("找不到用户气泡")
	}
	tt.Move(bubble.X+bubble.W/2, bubble.Y+bubble.H/2)
	tt.Frame()
	if err := tt.Click("编辑消息 1"); err != nil {
		t.Fatalf("点不到编辑：%v", err)
	}
	tt.Frame()
	if !app.inlineEditing() {
		t.Fatalf("编辑没有进入行内编辑器（texts=%v）", tt.Texts())
	}
	if _, ok := tt.Find("编辑消息正文"); !ok {
		t.Fatalf("编辑器没有正文输入区：%v", tt.Texts())
	}
	if _, ok := tt.Access("取消"); !ok {
		t.Fatalf("编辑器没有取消：%v", tt.Texts())
	}
	if _, ok := tt.Access("发送消息..."); !ok {
		t.Fatalf("编辑器没有发送：%v", tt.Texts())
	}
	// Esc 放弃：正文不变，编辑器收起。
	tt.Key(0, gui.KeyEscape)
	tt.Frame()
	if app.inlineEditing() {
		t.Fatal("Esc 没有退出编辑")
	}
	if !tt.HasText(app.rounds[0].user) {
		t.Fatal("放弃编辑改动了正文")
	}

	// 再进编辑，改完后点发送：正文替换并标「已编辑」。
	app.beginInlineEdit(0, app.rounds[0].user)
	tt.Frame()
	app.inlineDraft = "改过的正文"
	tt.Frame()
	if err := tt.Click("发送消息..."); err != nil {
		t.Fatalf("点不到发送：%v", err)
	}
	tt.Frame()
	if app.inlineEditing() {
		t.Fatal("保存后仍在编辑")
	}
	if !tt.HasText("改过的正文") || !tt.HasText("已编辑") {
		t.Fatalf("保存后没有替换正文或标记：%v", tt.Texts())
	}
}

// TestInlineEditorWiredSaveKeepsDraft：连线时保存没有通道，写原因并保留内容
// （TL-339 的失败分支）。
func TestInlineEditorWiredSaveKeepsDraft(t *testing.T) {
	src := wiredSource()
	src.conn = Connection{Phase: "connected"}
	src.names = []string{"会话"}
	src.hasSess = true
	src.session = SessionSnapshot{ID: "s1", Title: "会话",
		Rows: []RowSnapshot{{Kind: RowUser, Text: "原始消息"}}}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	app.beginInlineEdit(0, "原始消息")
	tt.Frame()
	app.inlineDraft = "改过的正文"
	tt.Frame()
	if err := tt.Click("发送消息..."); err != nil {
		t.Fatalf("点不到发送：%v", err)
	}
	tt.Frame()
	if !app.inlineEditing() {
		t.Fatal("保存失败后编辑器关掉了")
	}
	if app.inlineDraft != "改过的正文" {
		t.Fatalf("保存失败后草稿丢了：%q", app.inlineDraft)
	}
	if !tt.HasText(noteInlineEdit) {
		t.Fatalf("保存失败没有写原因：%v", tt.Texts())
	}
}

// compressionSource 造一个带压缩行的会话。
func compressionSource(status string, summary string) *fakeSource {
	src := wiredSource()
	src.conn = Connection{Phase: "connected"}
	src.names = []string{"压缩"}
	src.hasSess = true
	src.session = SessionSnapshot{
		ID: "s1", Title: "压缩",
		Rows: []RowSnapshot{
			{Kind: RowUser, Text: "第一轮"},
			{Kind: RowCompression, Text: summary, Status: status},
		},
		Compaction: &CompactionSnapshot{Status: status, Summary: summary},
	}
	return src
}

// TestCompressionStatesAndSummary：四态文案 + 摘要区三态（TL-665–TL-682）。
func TestCompressionStatesAndSummary(t *testing.T) {
	cases := []struct {
		status string
		want   string
	}{
		{RowStatusRunning, "正在压缩"},
		{RowStatusFailed, "压缩失败"},
		{"skipped", "已跳过"},
		{RowStatusCancelled, "已取消"},
	}
	for _, c := range cases {
		app := NewApp(Options{State: StateDefault, Source: compressionSource(c.status, "")})
		tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
		if !tt.HasText(c.want) {
			t.Fatalf("状态 %s 没有画成 %q：%v", c.status, c.want, tt.Texts())
		}
	}

	// 完成且有摘要：分隔行可点开摘要正文。
	app := NewApp(Options{State: StateDefault, Source: compressionSource("completed", "前 12 轮的摘要")})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if !tt.HasText("上下文已压缩") {
		t.Fatalf("完成的压缩没有默认文案：%v", tt.Texts())
	}
	if tt.HasText("前 12 轮的摘要") {
		t.Fatal("摘要区默认应该是收起的")
	}
	if err := tt.Click("上下文已压缩"); err != nil {
		t.Fatalf("点不到压缩行：%v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if !tt.HasText("前 12 轮的摘要") {
		t.Fatalf("展开后没有摘要正文：%v", tt.Texts())
	}

	// 失败且没有摘要：无法加载摘要 + 置灰的重试。
	app = NewApp(Options{State: StateDefault, Source: compressionSource(RowStatusFailed, "")})
	tt = gui.NewTester(app.View, WindowWidth, WindowHeight)
	if err := tt.Click("压缩失败"); err != nil {
		t.Fatalf("点不到压缩失败行：%v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if !tt.HasText("无法加载摘要。") {
		t.Fatalf("没有摘要的失败态文案不对：%v", tt.Texts())
	}
	info, ok := tt.Access("重试（" + noteCompactionRetry + "）")
	if !ok || !info.Disabled {
		t.Fatalf("重试没有按不可用画：%+v（找到 %v）", info, ok)
	}

	// 进行中：摘要生成中。
	app = NewApp(Options{State: StateDefault, Source: compressionSource(RowStatusRunning, "")})
	tt = gui.NewTester(app.View, WindowWidth, WindowHeight)
	if err := tt.Click("正在压缩"); err != nil {
		t.Fatalf("点不到正在压缩行：%v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if !tt.HasText("摘要生成中") {
		t.Fatalf("进行中没有摘要生成中：%v", tt.Texts())
	}

	// 完成但摘要为空：无摘要，不留空白区。
	app = NewApp(Options{State: StateDefault, Source: compressionSource("completed", "")})
	tt = gui.NewTester(app.View, WindowWidth, WindowHeight)
	if err := tt.Click("上下文已压缩"); err != nil {
		t.Fatalf("点不到压缩行：%v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if !tt.HasText("无摘要") {
		t.Fatalf("空摘要没有兜底文案：%v", tt.Texts())
	}
}

// artifactsSource 造一个带 5 条产物行的会话。
func artifactsSource(n int) *fakeSource {
	children := make([]RowSnapshot, 0, n)
	for i := 0; i < n; i++ {
		children = append(children, RowSnapshot{
			Kind: RowFileChange, Text: string(rune('a'+i)) + ".md", Title: string(rune('a'+i)) + ".md",
		})
	}
	src := wiredSource()
	src.conn = Connection{Phase: "connected"}
	src.names = []string{"产物"}
	src.hasSess = true
	src.session = SessionSnapshot{
		ID: "s1", Title: "产物",
		Rows: []RowSnapshot{
			{Kind: RowUser, Text: "生成文件"},
			{Kind: RowArtifact, Children: children},
		},
	}
	return src
}

// TestArtifactListCapsAndExpands：收起只列 3 行、底部「全部 {n} 个产物」；
// 展开列出全部并换「收起」；文件名单行显示、悬停给全名（A-08 / A-18 / A-19）。
func TestArtifactListCapsAndExpands(t *testing.T) {
	app := NewApp(Options{State: StateDefault, Source: artifactsSource(5)})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	if err := tt.Click("5 个产物"); err != nil {
		t.Fatalf("点不到产物块：%v", err)
	}
	tt.Advance(300 * time.Millisecond)
	if !tt.HasText("全部 5 个产物") {
		t.Fatalf("收起时没有展开入口：%v", tt.Texts())
	}
	for _, name := range []string{"d.md", "e.md"} {
		if tt.HasText(name) {
			t.Fatalf("收起时不该列出第 4 行起的 %q", name)
		}
	}
	if !tt.HasText("5 个产物") {
		t.Fatal("计数不再是全量")
	}
	if err := tt.Click("全部 5 个产物"); err != nil {
		t.Fatalf("点不到展开：%v", err)
	}
	tt.Frame()
	if !tt.HasText("d.md") || !tt.HasText("e.md") {
		t.Fatalf("展开后没有列出全部：%v", tt.Texts())
	}
	if _, ok := tt.Find("收起"); !ok {
		t.Fatalf("展开后没有「收起」：%v", tt.Texts())
	}
	if err := tt.Click("收起"); err != nil {
		t.Fatalf("点不到收起：%v", err)
	}
	tt.Frame()
	if tt.HasText("d.md") {
		t.Fatal("收起后仍列出全部")
	}
}

// TestArtifactRowDropdown：行尾下拉可展开、箭头翻转、同时只开一个；菜单项存在
// 但置灰并写原因（A-14–A-17）。
func TestArtifactRowDropdown(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	app.rounds[0].artifacts = []artifact{{name: "说明.md"}, {name: "稿.pdf"}}
	*app.blockOf("artifacts:0", false) = true
	tt.Frame()

	if _, ok := tt.Access("更多操作：产物 说明.md"); !ok {
		t.Fatalf("产物行没有下拉触发器：%v", tt.Texts())
	}
	if err := tt.Click("更多操作：产物 说明.md"); err != nil {
		t.Fatalf("点不到触发器：%v", err)
	}
	tt.Frame()
	if !*app.artifactMenuOpen("artifacts:0/说明.md") {
		t.Fatal("下拉没有展开")
	}
	if _, ok := tt.Access("更多操作（已展开）：产物 说明.md"); !ok {
		t.Fatal("展开后箭头没有翻转")
	}
	for _, item := range []string{"用默认应用打开", "在文件管理器中显示", "复制路径", "复制相对路径"} {
		info, ok := tt.Access(item)
		if !ok || !info.Disabled {
			t.Fatalf("下拉项 %q 不存在或没有置灰：%+v（找到 %v）", item, info, ok)
		}
		// 屏上的说明也要进读屏：菜单项的 note 设成 Description。
		if info.Description != noteArtifact {
			t.Fatalf("下拉项 %q 的读屏说明 = %q，想要 %q", item, info.Description, noteArtifact)
		}
	}
	// 打开另一行：前一个自动收起（同时只开一个）。
	if err := tt.Click("更多操作：产物 稿.pdf"); err != nil {
		t.Fatalf("点不到第二个触发器：%v", err)
	}
	tt.Frame()
	if *app.artifactMenuOpen("artifacts:0/说明.md") {
		t.Fatal("同时开了两个产物下拉")
	}
	if !*app.artifactMenuOpen("artifacts:0/稿.pdf") {
		t.Fatal("第二个下拉没有展开")
	}
}

// TestChangeRowMenuItems：改动行右键有打开（默认应用 / 文件管理器）与两组复制；
// 均在但不可用的项置灰并写原因；复制相对路径只在工作区内出现（D-93–D-97）。
func TestChangeRowMenuItems(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	app.rightOpen, app.rightTab = true, tabChanges
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	if err := tt.RightClick("composer.go"); err != nil {
		t.Fatalf("右键改动行：%v", err)
	}
	tt.Frame()
	for _, item := range []string{"暂存", "还原", "预览", "用外部工具打开", "用默认应用打开",
		"在文件管理器中显示", "复制路径"} {
		if _, ok := tt.Find(item); !ok {
			t.Fatalf("改动右键菜单缺少 %q（实际 %v）", item, tt.Menu())
		}
	}
	for _, item := range []string{"用外部工具打开", "用默认应用打开", "在文件管理器中显示"} {
		info, ok := tt.Access(item)
		if !ok || !info.Disabled {
			t.Fatalf("%q 没有置灰：%+v（找到 %v）", item, info, ok)
		}
	}
	// 没有工作目录时不算相对路径：这一项不该出现；置灰的复制项要写原因。
	if tt.HasText("复制相对路径") {
		t.Fatal("没有工作目录却给了复制相对路径")
	}
	if info, ok := tt.Access("复制路径"); !ok || info.Disabled || info.Description != "" {
		t.Fatalf("右栏「复制路径」不受工作目录影响：%+v（找到 %v）", info, ok)
	}
	// 两处复制相对路径共用同一个判定：工作区内给相对路径，工作区外与无工作目录
	// 都不给（D-97）。
	if rel, ok := workspaceRelativePath("/work", "/work/internal/ui/composer.go"); !ok ||
		rel != "internal/ui/composer.go" {
		t.Fatalf("相对路径算错：%q（%v）", rel, ok)
	}
	if _, ok := workspaceRelativePath("/work", "/elsewhere/x.go"); ok {
		t.Fatal("工作区外的路径不该给相对路径")
	}
	if _, ok := workspaceRelativePath("/work", "internal/ui/composer.go"); !ok {
		t.Fatal("工作区内的相对路径行应给相对路径")
	}
	if _, ok := workspaceRelativePath("", "internal/ui/composer.go"); ok {
		t.Fatal("没有工作目录时不该给相对路径")
	}
	if _, ok := workspaceRelativePath("/work", "../x.go"); ok {
		t.Fatal("跑出工作区的相对路径不该给")
	}
}

// TestTerminalLayersKeepBuffers：每个终端标签一层画布；切换不重建、层级尺寸不变；
// 非激活层保留自己的输出（TM-070 / TM-074）。
func TestTerminalLayersKeepBuffers(t *testing.T) {
	app, tt := panelTester(t, tabTerminal)
	app.termLines = []termLine{{text: "$ ls", color: -1}, {text: "README.md", color: -1}}
	tt.Frame()

	if got := app.termLayerCount(); got != 1 {
		t.Fatalf("一个标签应有 1 层，实际 %d", got)
	}
	first := app.termLines
	if err := tt.Click("新建终端"); err != nil {
		t.Fatalf("点不到新建终端：%v", err)
	}
	tt.Frame()
	if got := app.termLayerCount(); got != 2 {
		t.Fatalf("新建后应有 2 层，实际 %d", got)
	}
	if app.termActive != 1 {
		t.Fatalf("新层没有激活：%d", app.termActive)
	}
	if _, ok := tt.Find("终端标签：终端 2"); !ok {
		t.Fatalf("新标签没有出现：%v", tt.Texts())
	}
	app.termLines = []termLine{{text: "$ git status", color: -1}}
	tt.Frame()

	// 切回第一层：输出还在。
	if err := tt.Click("终端标签：" + app.termTabs[0].title); err != nil {
		t.Fatalf("点不到第一个标签：%v", err)
	}
	tt.Frame()
	if app.termActive != 0 {
		t.Fatalf("没有切回第一层：%d", app.termActive)
	}
	if len(app.termLines) != len(first) || app.termLines[1].text != "README.md" {
		t.Fatalf("切回后输出丢了：%+v", app.termLines)
	}
	// 尺寸不变：切回去还是同一块画布。
	before, ok := tt.Find("终端画布层")
	if !ok {
		t.Fatal("找不到画布")
	}
	if err := tt.Click("终端标签：" + app.termTabs[1].title); err != nil {
		t.Fatalf("点不到第二个标签：%v", err)
	}
	tt.Frame()
	after, ok := tt.Find("终端画布层")
	if !ok || before != after {
		t.Fatalf("切层后画布包围盒变了：%+v → %+v", before, after)
	}
}

// TestSubagentGenericLoadFailure：错误原文为空时仍画「加载失败」与重试；
// -demo 里重试清掉错误后回到样例时间线（不是回到一个永远转的加载圈）
// （SG-010 / SG-014）。
func TestSubagentGenericLoadFailure(t *testing.T) {
	app, tt := panelTester(t, tabSubagent)
	app.panels.subagentFailed = true
	app.panels.subagentError = ""
	tt.Frame()

	if !tt.HasText("加载失败") {
		t.Fatalf("空错误原文没有通用兜底：%v", tt.Texts())
	}
	if _, ok := tt.Find("重试"); !ok {
		t.Fatalf("错误行没有重试：%v", tt.Texts())
	}
	if err := tt.Click("重试"); err != nil {
		t.Fatalf("点不到重试：%v", err)
	}
	tt.Frame()
	if app.panels.subagentFailed {
		t.Fatal("重试没有清掉错误")
	}
	if !tt.HasText(subagentSamples()[0]) {
		t.Fatalf("重试后没有回到样例时间线：%v", tt.Texts())
	}

	// 已有消息时出错：行内横幅里同样用通用文案，横幅外仍画时间线。
	app.panels.subagentLines = subagentSamples()
	app.panels.subagentFailed = true
	tt.Frame()
	if !tt.HasText("加载失败") {
		t.Fatalf("有消息时的空错误没有兜底：%v", tt.Texts())
	}
	if _, ok := tt.Find("重试"); !ok {
		t.Fatalf("错误横幅没有重试：%v", tt.Texts())
	}
	if !tt.HasText(subagentSamples()[0]) {
		t.Fatalf("错误横幅把时间线挤掉了：%v", tt.Texts())
	}
}

// TestUserBubbleExpandAndCollapse：正文超过折叠行数时底部出现「展示全部」，
// 点开后换「收起」；短正文不出现这个按钮（TL-31）。
func TestUserBubbleExpandAndCollapse(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if _, ok := tt.Find("展示全部"); ok {
		t.Fatal("短正文不该出现「展示全部」")
	}

	long := ""
	for i := 0; i < 9; i++ {
		long += "这一行内容用来把气泡撑到折叠行数以上。\n"
	}
	src := wiredSource()
	src.conn = Connection{Phase: "connected"}
	src.names = []string{"长消息"}
	src.hasSess = true
	src.session = SessionSnapshot{ID: "s1", Title: "长消息",
		Rows: []RowSnapshot{{Kind: RowUser, Text: long}}}
	app = NewApp(Options{State: StateDefault, Source: src})
	tt = gui.NewTester(app.View, WindowWidth, WindowHeight)
	if _, ok := tt.Access("展示全部（当前为折叠）"); !ok {
		t.Fatalf("超长正文没有折叠入口：%v", tt.Texts())
	}
	if r, ok := tt.Find("展示全部"); ok {
		tt.Move(r.X+r.W/2, r.Y+r.H/2)
		tt.Frame()
	}
	if err := tt.Click("展示全部"); err != nil {
		t.Fatalf("点不到展示全部：%v", err)
	}
	tt.Frame()
	if !app.userExpanded[0] {
		t.Fatalf("点了展示全部但展开态没变：%v", app.userExpanded)
	}
	if _, ok := tt.Access("收起"); !ok {
		t.Fatalf("展开后没有「收起」：%v", tt.Texts())
	}
	if err := tt.Click("收起"); err != nil {
		t.Fatalf("点不到收起：%v", err)
	}
	tt.Frame()
	if _, ok := tt.Access("展示全部（当前为折叠）"); !ok {
		t.Fatal("收起后没有回到折叠入口")
	}
}

// shellStopIntents 只实现命令停止与打开会话，其余动作被调用会 panic：
// 说明面板上的「停止」确实走了写入口。
type shellStopIntents struct {
	Intents
	stopped chan string
}

func (s *shellStopIntents) OpenSession(context.Context, string) error { return nil }
func (s *shellStopIntents) StopShell(_ context.Context, runID string) error {
	s.stopped <- runID
	return nil
}

// TestShellRunPanelStopSendsIntent：连线时面板上的「停止」发 StopShell（运行标识
// 来自事实，不靠输入区记的那一份）。
func TestShellRunPanelStopSendsIntent(t *testing.T) {
	src := wiredSource()
	src.conn = Connection{Phase: "connected"}
	src.names = []string{"命令面板"}
	src.hasSess = true
	src.session = SessionSnapshot{
		ID: "s1", Title: "命令面板",
		Rows: []RowSnapshot{{Kind: RowUser, Text: "跑一下"}},
		ShellRun: &ShellRunSnapshot{
			RunID: "run-live-1", Command: "go test ./...", Mode: "record", Status: "running",
		},
	}
	stub := &shellStopIntents{stopped: make(chan string, 1)}
	app := NewApp(Options{State: StateDefault, Source: src, Intents: stub})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if err := tt.Click("停止命令"); err != nil {
		t.Fatalf("点不到停止：%v", err)
	}
	select {
	case got := <-stub.stopped:
		if got != "run-live-1" {
			t.Fatalf("StopShell 收到 %q，想要 run-live-1", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("点停止没有发 StopShell")
	}
}
