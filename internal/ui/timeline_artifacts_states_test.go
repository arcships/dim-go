package ui

// 本文件验收时间线「本轮产物」块的三处真缺：块头「提交或推送」菜单
// （A-03–A-06）、已删除标签（A-11）与无法预览替代文本（A-12）。
//
// git 写通道在协议 v3 里没有：三项菜单保留、置灰，并把原因写进项里；产物
// 不存在 / 不可打开按快照事实画，不给「打开」。

import (
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// artifactBool 是产物事实的取值指针（nil 表示快照没有给）。
func artifactBool(v bool) *bool { return &v }

// openArtifactsBlock 展开样例里的产物块，返回界面与测试器。
func openArtifactsBlock(t *testing.T, arts []artifact) (*App, *gui.Tester) {
	t.Helper()
	app, tt := newTester(t, StateDefault)
	app.branch = "main"
	app.rounds[0].artifacts = arts
	*app.blockOf("artifacts:0", false) = true
	tt.Frame()
	return app, tt
}

// TestArtifactsSubmitMenuHasThreeItems：块头「提交或推送」在场；单击展开菜单，
// 三项按 提交 / 提交并推送 / 推送 的顺序排列，且都因缺 git 写通道置灰并写原因
// （A-03–A-06）。
func TestArtifactsSubmitMenuHasThreeItems(t *testing.T) {
	_, tt := openArtifactsBlock(t, []artifact{{name: "说明.md"}})
	trigger, ok := tt.Access("提交或推送")
	if !ok {
		t.Fatalf("产物块头没有「提交或推送」：%v", tt.Texts())
	}
	if trigger.Disabled {
		t.Fatal("「提交或推送」本身应当可展开菜单")
	}
	if err := tt.Click("提交或推送"); err != nil {
		t.Fatalf("点不到「提交或推送」：%v", err)
	}
	tt.Frame()
	trigger, ok = tt.Access("提交或推送")
	if !ok || !trigger.Expanded {
		t.Fatalf("菜单没有展开：%+v（找到 %v）", trigger, ok)
	}
	items := []string{"提交", "提交并推送", "推送"}
	var last float32
	for i, item := range items {
		info, ok := tt.Access(item)
		if !ok {
			t.Fatalf("菜单缺 %q：%v", item, tt.Texts())
		}
		if !info.Disabled {
			t.Fatalf("缺 git 写通道时 %q 没有置灰：%+v", item, info)
		}
		if info.Description != noteGitWrite {
			t.Fatalf("%q 的禁用原因 = %q，想要 %q", item, info.Description, noteGitWrite)
		}
		if i > 0 && info.Bounds.Y < last {
			t.Fatalf("菜单项顺序不对：%q 在上一项上方", item)
		}
		last = info.Bounds.Y
	}
}

// TestArtifactsSubmitMenuHiddenWithoutBranch：当前分支为空时块头只剩计数文本，
// 不出现「提交或推送」（A-03 的条件）。
func TestArtifactsSubmitMenuHiddenWithoutBranch(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	app.branch = ""
	app.rounds[0].artifacts = []artifact{{name: "说明.md"}}
	*app.blockOf("artifacts:0", false) = true
	tt.Frame()
	if _, ok := tt.Access("提交或推送"); ok {
		t.Fatalf("没有分支仍出现「提交或推送」：%v", tt.Texts())
	}
	if _, ok := tt.Find("1 个产物"); !ok {
		t.Fatalf("块头计数不在：%v", tt.Texts())
	}
}

// TestArtifactDeletedRowMarksAndHidesOpen：产物不存在时行内标「已删除」，不再给
// 「打开」；同组存在的产物仍有「打开」（A-11 / A-13）。
func TestArtifactDeletedRowMarksAndHidesOpen(t *testing.T) {
	_, tt := openArtifactsBlock(t, []artifact{
		{name: "删掉了.md", missing: true},
		{name: "还在.md"},
	})
	if !tt.HasText("已删除") {
		t.Fatalf("已删除的产物行没有标签：%v", tt.Texts())
	}
	if _, ok := tt.Access("打开产物 删掉了.md（" + noteArtifact + "）"); ok {
		t.Fatal("已删除的产物仍给了「打开」")
	}
	if info, ok := tt.Access("打开产物 还在.md（" + noteArtifact + "）"); !ok || !info.Disabled {
		t.Fatalf("存在的产物缺置灰的「打开」：%+v（找到 %v）", info, ok)
	}
}

// TestArtifactUnopenableReplacesOpen：条目存在但不可打开时，在「打开」的位置
// 显示「无法预览文件」，不是按钮（A-12）。
func TestArtifactUnopenableReplacesOpen(t *testing.T) {
	_, tt := openArtifactsBlock(t, []artifact{
		{name: "看不到.bin", canOpen: artifactBool(false)},
	})
	if _, ok := tt.Find("无法预览文件"); !ok {
		t.Fatalf("不可打开的产物没有替代文本：%v", tt.Texts())
	}
	if _, ok := tt.AccessByRole(gui.RoleButton, "无法预览文件"); ok {
		t.Fatal("「无法预览文件」被画成了按钮")
	}
	if _, ok := tt.Access("打开产物 看不到.bin（" + noteArtifact + "）"); ok {
		t.Fatal("不可打开的产物仍给了「打开」")
	}
}

// TestArtifactFactsFromSnapshot：快照给出的事实直接决定「已删除」与「无法预览
// 文件」，界面不自己猜（A-11 / A-12）。
func TestArtifactFactsFromSnapshot(t *testing.T) {
	src := wiredSource()
	src.conn = Connection{Phase: "connected"}
	src.names = []string{"产物"}
	src.hasSess = true
	src.session = SessionSnapshot{
		ID: "s1", Title: "产物", Branch: "main",
		Rows: []RowSnapshot{
			{Kind: RowUser, Text: "生成文件"},
			{Kind: RowArtifact, Children: []RowSnapshot{
				{Kind: RowArtifact, Text: "没了.md", ArtifactExists: artifactBool(false)},
				{Kind: RowArtifact, Text: "打不开.bin", ArtifactCanOpen: artifactBool(false)},
				{Kind: RowArtifact, Text: "正常.md"},
			}},
		},
	}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if err := tt.Click("3 个产物"); err != nil {
		t.Fatalf("点不到产物块：%v", err)
	}
	tt.Frame()
	if !tt.HasText("已删除") || !tt.HasText("无法预览文件") {
		t.Fatalf("快照事实没有画出来：%v", tt.Texts())
	}
	if _, ok := tt.Access("打开产物 没了.md（" + noteArtifact + "）"); ok {
		t.Fatal("快照说不存在的产物仍给了「打开」")
	}
	if _, ok := tt.Access("打开产物 打不开.bin（" + noteArtifact + "）"); ok {
		t.Fatal("快照说不可打开的产物仍给了「打开」")
	}
	if info, ok := tt.Access("打开产物 正常.md（" + noteArtifact + "）"); !ok || !info.Disabled {
		t.Fatalf("正常产物缺置灰的「打开」：%+v（找到 %v）", info, ok)
	}
}

// TestArtifactDeletedRowRestores：文件重新出现后「已删除」消失，「打开」回到行上
// （A-11 的异常与记忆口径：按当帧事实重建，不沿用旧标签）。
func TestArtifactDeletedRowRestores(t *testing.T) {
	app, tt := openArtifactsBlock(t, []artifact{{name: "回来了.md", missing: true}})
	if !tt.HasText("已删除") {
		t.Fatalf("先给的存在事实没有画成已删除：%v", tt.Texts())
	}
	app.rounds[0].artifacts = []artifact{{name: "回来了.md"}}
	tt.Frame()
	if tt.HasText("已删除") {
		t.Fatalf("文件回来后仍留着「已删除」：%v", tt.Texts())
	}
	if info, ok := tt.Access("打开产物 回来了.md（" + noteArtifact + "）"); !ok || !info.Disabled {
		t.Fatalf("文件回来后没有恢复「打开」：%+v（找到 %v）", info, ok)
	}
}

// TestArtifactUnopenableRestoresOpen：能力恢复后「无法预览文件」换成「打开」
// （A-12 的异常与记忆口径）。
func TestArtifactUnopenableRestoresOpen(t *testing.T) {
	app, tt := openArtifactsBlock(t, []artifact{
		{name: "恢复了.bin", canOpen: artifactBool(false)},
	})
	if _, ok := tt.Find("无法预览文件"); !ok {
		t.Fatalf("先给的不可打开事实没有画成替代文本：%v", tt.Texts())
	}
	app.rounds[0].artifacts = []artifact{{name: "恢复了.bin", canOpen: artifactBool(true)}}
	tt.Frame()
	if tt.HasText("无法预览文件") {
		t.Fatalf("能力恢复后仍留着替代文本：%v", tt.Texts())
	}
	if info, ok := tt.Access("打开产物 恢复了.bin（" + noteArtifact + "）"); !ok || !info.Disabled {
		t.Fatalf("能力恢复后没有恢复「打开」：%+v（找到 %v）", info, ok)
	}
}

// TestArtifactsSubmitMenuNarrow：窄窗下计数与「提交或推送」同排不重叠，菜单三项
// 仍在窗口内（A-03 的窄窗口径）。
func TestArtifactsSubmitMenuNarrow(t *testing.T) {
	app := NewApp(Options{State: StateDefault})
	app.branch = "main"
	app.rounds[0].artifacts = []artifact{{name: "说明.md"}}
	*app.blockOf("artifacts:0", false) = true
	tt := gui.NewTester(app.View, CompactWidth, WindowHeight)
	tt.Frame()
	count, ok := tt.Find("1 个产物")
	if !ok {
		t.Fatalf("窄窗缺计数：%v", tt.Texts())
	}
	button, ok := tt.Access("提交或推送")
	if !ok {
		t.Fatalf("窄窗缺「提交或推送」：%v", tt.Texts())
	}
	if count.X+count.W > button.Bounds.X+0.5 && button.Bounds.X+button.Bounds.W > count.X+0.5 &&
		count.Y+count.H > button.Bounds.Y+0.5 && button.Bounds.Y+button.Bounds.H > count.Y+0.5 {
		t.Fatalf("计数与按钮重叠：计数 %+v，按钮 %+v", count, button.Bounds)
	}
	if err := tt.Click("提交或推送"); err != nil {
		t.Fatalf("窄窗点不到菜单：%v", err)
	}
	tt.Frame()
	for _, item := range []string{"提交", "提交并推送", "推送"} {
		info, ok := tt.Access(item)
		if !ok {
			t.Fatalf("窄窗菜单缺 %q：%v", item, tt.Texts())
		}
		if info.Bounds.X < 0 || info.Bounds.X+info.Bounds.W > CompactWidth {
			t.Fatalf("窄窗下 %q 越出窗口：%+v", item, info.Bounds)
		}
	}
}

// TestArtifactsSubmitMenuClosesOutside：点菜单外部关闭，箭头恢复（A-03 的交互）。
func TestArtifactsSubmitMenuClosesOutside(t *testing.T) {
	app, tt := openArtifactsBlock(t, []artifact{{name: "说明.md"}})
	if err := tt.Click("提交或推送"); err != nil {
		t.Fatalf("点不到「提交或推送」：%v", err)
	}
	tt.Frame()
	if !*app.artifactsGitMenuOpen("artifacts:0") {
		t.Fatal("菜单没有展开")
	}
	tt.ClickAt(WindowWidth/2, 60)
	tt.Frame()
	if *app.artifactsGitMenuOpen("artifacts:0") {
		t.Fatal("点外部没有关闭菜单")
	}
}
