package ui

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「导出对话」面板的回归断言：三种格式的内容、范围选择、剪贴板与
// 文件两条通路，以及可核对回执（字符数 / 字节数）与失败原因。

// exportTestSession 是一份含两轮的事实会话：第二轮带思考与工具，用来验格式。
func exportTestSession() SessionSnapshot {
	return SessionSnapshot{
		ID: "s1", Title: "真实会话", Branch: "main", TurnCount: 2,
		LastTurnEnded: true, LastTurnIndex: 2,
		Branches: []BranchSnapshot{{ID: "main", Name: "main", Status: "open"}},
		Rows: []RowSnapshot{
			{Kind: RowUser, Text: "第一条消息", Status: RowStatusDone, BranchID: "main", TurnIndex: 1},
			{Kind: RowAssistant, Text: "第一条回复", Status: RowStatusDone, BranchID: "main", TurnIndex: 1},
			{Kind: RowUser, Text: "第二条消息", Status: RowStatusDone, BranchID: "main", TurnIndex: 2},
			{Kind: RowThinking, Text: "先看看事实", Status: RowStatusDone, BranchID: "main", TurnIndex: 2},
			{Kind: RowTool, Title: "读取", Args: "internal/ui/export.go", Text: "读到了",
				Status: RowStatusDone, BranchID: "main", TurnIndex: 2},
			{Kind: RowAssistant, Text: "第二条回复", Status: RowStatusDone, BranchID: "main", TurnIndex: 2},
		},
	}
}

// exportTestApp 建一个连着事实的界面，并打开导出面板。
func exportTestApp(t *testing.T, snap SessionSnapshot) (*App, *gui.Tester) {
	t.Helper()
	src := wiredSource()
	src.names = []string{snap.ID}
	src.titles = []string{snap.Title}
	src.session = snap
	src.hasSess = true
	app := NewApp(Options{State: StateDefault, Source: src})
	app.openExportPanel(snap.ID, snap.Title)
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()
	return app, tt
}

// waitExport 反复渲染，直到条件成立；用于等后台文件写入交回渲染线程。
func waitExport(t *testing.T, tt *gui.Tester, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		tt.Frame()
		if cond() {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatalf("等待 %s 超时", what)
}

// TestExportPanelFormatsAndPreview：面板给出三种格式、三种范围、说明文案与预览，
// 切格式后预览跟着换。
func TestExportPanelFormatsAndPreview(t *testing.T) {
	app, tt := exportTestApp(t, exportTestSession())

	for _, want := range []string{
		exportTitle, "Markdown", "JSON", "纯文本",
		"整条会话", "仅当前分支", "选中的轮次", exportDescPreview,
	} {
		if _, ok := tt.Find(want); !ok {
			t.Fatalf("面板缺 %q：%v", want, tt.Texts())
		}
	}
	if !tt.HasText("导出「真实会话」的会话对话。不包含原始 Provider 请求。") {
		t.Fatalf("说明文案不是逐字那句：%v", tt.Texts())
	}
	if !tt.HasText("## 第 1 轮") || !tt.HasText("## 第 2 轮") {
		t.Fatalf("Markdown 预览没有两轮：%v", tt.Texts())
	}
	if !tt.HasText("### 你") || !tt.HasText("第一条消息") {
		t.Fatalf("Markdown 预览没有用户消息：%v", tt.Texts())
	}

	// 切到 JSON：预览换成结构化的文档，字段名与版本都在。
	if err := tt.Click("JSON"); err != nil {
		t.Fatalf("点不到 JSON：%v", err)
	}
	tt.Frame()
	if app.export.format != exportFormatJSON {
		t.Fatalf("格式没有切到 JSON：%q", app.export.format)
	}
	if !tt.HasText(`"format": "dimagent.dialogue-export"`) ||
		!tt.HasText(`"turnCount": 2`) {
		t.Fatalf("JSON 预览不对：%v", tt.Texts())
	}

	// 切到纯文本：没有 Markdown 标记，轮次用小节行分隔。
	if err := tt.Click("纯文本"); err != nil {
		t.Fatalf("点不到纯文本：%v", err)
	}
	tt.Frame()
	if app.export.format != exportFormatText {
		t.Fatalf("格式没有切到纯文本：%q", app.export.format)
	}
	if !tt.HasText("--- 第 1 轮 ---") || tt.HasText("## 第 1 轮") {
		t.Fatalf("纯文本预览不对：%v", tt.Texts())
	}
}

// TestExportFormatsCarryTheSameTurns：三种格式装的是同一份回合内容；JSON 结构
// 可解析，Markdown / 纯文本保留工具与思考。
func TestExportFormatsCarryTheSameTurns(t *testing.T) {
	app, _ := exportTestApp(t, exportTestSession())
	turns, branch := app.exportTurns()
	if len(turns) != 2 {
		t.Fatalf("回合数 = %d，想要 2：%+v", len(turns), turns)
	}
	if turns[1].User != "第二条消息" || len(turns[1].Tools) != 1 || turns[1].Tools[0].Action != "读取" {
		t.Fatalf("第二轮内容不对：%+v", turns[1])
	}
	if len(turns[1].Thinking) != 1 || turns[1].Thinking[0] != "先看看事实" {
		t.Fatalf("思考没有进导出：%+v", turns[1])
	}

	app.export.format = exportFormatJSON
	content := app.exportContent(turns, branch)
	var doc exportDocument
	if err := json.Unmarshal([]byte(content), &doc); err != nil {
		t.Fatalf("JSON 导出无法解析：%v\n%s", err, content)
	}
	if doc.SessionID != "s1" || doc.Title != "真实会话" || doc.Branch != "main" ||
		doc.TurnCount != 2 || len(doc.Turns) != 2 {
		t.Fatalf("JSON 文件头不对：%+v", doc)
	}
	if doc.Turns[1].Tools[0].Args != "internal/ui/export.go" {
		t.Fatalf("JSON 工具参数不对：%+v", doc.Turns[1].Tools)
	}

	app.export.format = exportFormatMarkdown
	markdown := app.exportContent(turns, branch)
	for _, want := range []string{"# 真实会话", "## 第 2 轮", "### 你", "### 助手", "> 思考：先看看事实", "> 工具：读取 internal/ui/export.go — 读到了"} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("Markdown 缺 %q：\n%s", want, markdown)
		}
	}

	app.export.format = exportFormatText
	text := app.exportContent(turns, branch)
	for _, want := range []string{"真实会话", "--- 第 2 轮 ---", "用户：", "助手：", "工具：读取 internal/ui/export.go — 读到了"} {
		if !strings.Contains(text, want) {
			t.Fatalf("纯文本缺 %q：\n%s", want, text)
		}
	}
}

// TestExportScopeBranchAndTurns：范围筛回合；勾选变化实时更新计数，0 个回合时
// 导出按钮禁用。
func TestExportScopeBranchAndTurns(t *testing.T) {
	snap := exportTestSession()
	// 第三轮在 side 分支上：当前分支仍是 main，用来分辨「整条」与「仅当前分支」。
	snap.Rows = append(snap.Rows,
		RowSnapshot{Kind: RowUser, Text: "侧分支消息", Status: RowStatusDone, BranchID: "side", TurnIndex: 3},
	)
	snap.Branches = append(snap.Branches, BranchSnapshot{ID: "side", Name: "side", Status: "open"})
	app, tt := exportTestApp(t, snap)

	all, branch := app.exportTurns()
	if len(all) != 3 {
		t.Fatalf("整条会话回合数 = %d，想要 3", len(all))
	}
	app.export.scope = exportScopeBranch
	onlyMain := app.exportScopeTurns(all, branch)
	if len(onlyMain) != 2 {
		t.Fatalf("仅当前分支回合数 = %d，想要 2：%+v", len(onlyMain), onlyMain)
	}
	for _, turn := range onlyMain {
		if turn.Branch != "main" {
			t.Fatalf("仅当前分支混进了 %+v", turn)
		}
	}

	// 选中的轮次：默认全选，取消第一轮后计数变成 1 / 3。
	app.export.scope = exportScopeTurns
	app.export.turns = nil
	tt.Frame()
	if !tt.HasText("保留 3 / 3 个回合") {
		t.Fatalf("没有默认全选计数：%v", tt.Texts())
	}
	*app.exportTurnPtr(1) = false
	tt.Frame()
	if !tt.HasText("保留 2 / 3 个回合") {
		t.Fatalf("取消一轮后计数不对：%v", tt.Texts())
	}
	selected := app.exportScopeTurns(all, branch)
	if len(selected) != 2 || selected[0].Index != 2 {
		t.Fatalf("排除后的回合不对：%+v", selected)
	}

	// 全部排除：计数 0，导出按钮禁用，且按钮旁不给可点的导出动作。
	if err := tt.Click("全部排除"); err != nil {
		t.Fatalf("点不到全部排除：%v", err)
	}
	tt.Frame()
	if !tt.HasText("保留 0 / 3 个回合") {
		t.Fatalf("全部排除后计数不是 0：%v", tt.Texts())
	}
	if info, ok := tt.Access("导出到剪贴板"); !ok || !info.Disabled {
		t.Fatalf("保留 0 时剪贴板按钮没有禁用：%+v（找到 %v）", info, ok)
	}
	// 全部保留：回到 3 / 3。
	if err := tt.Click("全部保留"); err != nil {
		t.Fatalf("点不到全部保留：%v", err)
	}
	tt.Frame()
	if !tt.HasText("保留 3 / 3 个回合") {
		t.Fatalf("全部保留后计数不对：%v", tt.Texts())
	}
}

// TestExportToClipboardReceipt：剪贴板通路写的是屏上那份内容，回执给字符数。
func TestExportToClipboardReceipt(t *testing.T) {
	app, tt := exportTestApp(t, exportTestSession())
	turns, branch := app.exportTurns()
	app.export.scope = exportScopeSession
	want := app.exportContent(app.exportScopeTurns(turns, branch), branch)

	if err := tt.Click("导出到剪贴板"); err != nil {
		t.Fatalf("点不到导出到剪贴板：%v", err)
	}
	tt.Frame()
	if got := tt.Clipboard(); got != want {
		t.Fatalf("剪贴板内容与预览不一致：\n--- 剪贴板 ---\n%s\n--- 想要 ---\n%s", got, want)
	}
	receipt := app.export.receipt
	if !strings.HasPrefix(receipt, "已复制到剪贴板 · ") {
		t.Fatalf("回执不是字符数：%q", receipt)
	}
	wantReceipt := "已复制到剪贴板 · " + strconv.Itoa(utf8.RuneCountInString(want)) + " 个字符"
	if receipt != wantReceipt {
		t.Fatalf("回执 = %q，想要 %q", receipt, wantReceipt)
	}
	if !tt.HasText(receipt) {
		t.Fatalf("回执没有上屏：%v", tt.Texts())
	}
}

// TestExportClipboardFailure：剪贴板写失败给原因，不给「已复制」回执。
func TestExportClipboardFailure(t *testing.T) {
	app, tt := exportTestApp(t, exportTestSession())
	tt.FailClipboard(errors.New("剪贴板被占用"))
	if err := tt.Click("导出到剪贴板"); err != nil {
		t.Fatalf("点不到导出到剪贴板：%v", err)
	}
	tt.Frame()
	if app.export.receipt != "" {
		t.Fatalf("写失败却有成功回执：%q", app.export.receipt)
	}
	if !strings.Contains(app.export.failure, "复制失败：剪贴板被占用") {
		t.Fatalf("失败原因不对：%q", app.export.failure)
	}
	if info, ok := tt.Access(app.export.failure); !ok || info.Role != gui.RoleAlert {
		t.Fatalf("失败原因不是 alert：%+v（找到 %v）", info, ok)
	}
}

// TestExportToFileWithSavePanel：系统保存面板给路径后写文件，成功后收起面板并
// 在标题行给字节数回执。
func TestExportToFileWithSavePanel(t *testing.T) {
	app, tt := exportTestApp(t, exportTestSession())
	dir := t.TempDir()
	target := filepath.Join(dir, "export.md")
	app.saveFile = func(suggested string) (string, error) {
		if !strings.HasSuffix(suggested, ".md") {
			t.Errorf("建议路径扩展名不对：%q", suggested)
		}
		return target, nil
	}
	turns, branch := app.exportTurns()
	want := app.exportContent(app.exportScopeTurns(turns, branch), branch)

	if err := tt.Click("导出到文件"); err != nil {
		t.Fatalf("点不到导出到文件：%v", err)
	}
	waitExport(t, tt, "文件写出", func() bool {
		if app.export.open {
			return false
		}
		_, err := os.Stat(target)
		return err == nil
	})
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("读不到导出的文件：%v", err)
	}
	if string(data) != want {
		t.Fatalf("文件内容与预览不一致：\n--- 文件 ---\n%s\n--- 想要 ---\n%s", data, want)
	}
	wantReceipt := "已导出到 " + target + " · " + strconv.Itoa(len(data)) + " 字节"
	if app.headerNote != wantReceipt {
		t.Fatalf("回执 = %q，想要 %q", app.headerNote, wantReceipt)
	}
	if tt.HasText(exportTitle) {
		t.Fatalf("成功后面板没有收起：%v", tt.Texts())
	}
}

// TestExportToFileManualPath：没有系统保存面板时退回手输路径；写失败留在面板里
// 给原因，改对之后能写出。
func TestExportToFileManualPath(t *testing.T) {
	app, tt := exportTestApp(t, exportTestSession())
	app.saveFile = nil
	if err := tt.Click("导出到文件"); err != nil {
		t.Fatalf("点不到导出到文件：%v", err)
	}
	tt.Frame()
	if !app.export.manual {
		t.Fatal("没有保存面板时没有退回手输路径")
	}
	if !tt.HasText(exportManualHint) {
		t.Fatalf("没有写手输原因：%v", tt.Texts())
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "manual.txt")
	app.export.format = exportFormatText
	app.export.path = filepath.Join(dir, "missing", "manual.txt")
	if err := tt.Click("写入文件"); err != nil {
		t.Fatalf("点不到写入文件：%v", err)
	}
	waitExport(t, tt, "失败回执", func() bool { return app.export.failure != "" })
	if !strings.Contains(app.export.failure, "导出失败：目录不存在") {
		t.Fatalf("失败原因不对：%q", app.export.failure)
	}
	if !app.export.open {
		t.Fatal("写失败后面板被收起了")
	}
	if info, ok := tt.Access(app.export.failure); !ok || info.Role != gui.RoleAlert {
		t.Fatalf("失败原因不是 alert：%+v（找到 %v）", info, ok)
	}

	app.export.path = target
	app.export.failure = ""
	// 指针从按钮上移开：第一次点击后悬停的提示层盖在按钮中心；真实用户在读
	// 失败原因时也会移动指针（这里只清命中，不改任何事实）。
	tt.Move(10, 10)
	tt.Frame()
	if err := tt.Click("写入文件"); err != nil {
		t.Fatalf("点不到写入文件：%v", err)
	}
	waitExport(t, tt, "手输路径写出", func() bool {
		if app.export.open {
			return false
		}
		_, err := os.Stat(target)
		return err == nil
	})
	if !strings.Contains(app.headerNote, "字节") {
		t.Fatalf("手输路径写出后没有字节数回执：%q", app.headerNote)
	}
}

// TestExportSavePanelCancelKeepsPanel：保存面板取消不是失败，面板与按钮都回来。
func TestExportSavePanelCancelKeepsPanel(t *testing.T) {
	app, tt := exportTestApp(t, exportTestSession())
	app.saveFile = func(string) (string, error) { return "", nil }
	if err := tt.Click("导出到文件"); err != nil {
		t.Fatalf("点不到导出到文件：%v", err)
	}
	waitExport(t, tt, "取消交回", func() bool { return !app.export.busy })
	if !app.export.open {
		t.Fatal("取消后把面板收起了")
	}
	if app.export.failure != "" {
		t.Fatalf("取消被当成失败：%q", app.export.failure)
	}
	if info, ok := tt.Access("导出到文件"); !ok || info.Disabled {
		t.Fatalf("取消后按钮没有恢复：%+v（找到 %v）", info, ok)
	}
}

// TestExportPreviewCapsAtTwentyLines：预览只给前 20 行，并标出总行数。
func TestExportPreviewCapsAtTwentyLines(t *testing.T) {
	snap := SessionSnapshot{ID: "s1", Title: "长会话", Branch: "main", TurnCount: 40}
	for i := 1; i <= 40; i++ {
		snap.Rows = append(snap.Rows, RowSnapshot{
			Kind: RowUser, Text: "第 " + strconv.Itoa(i) + " 条", Status: RowStatusDone,
			BranchID: "main", TurnIndex: i,
		})
	}
	app, tt := exportTestApp(t, snap)
	turns, branch := app.exportTurns()
	content := app.exportContent(app.exportScopeTurns(turns, branch), branch)
	preview := firstLines(content, previewMaxLines)
	if strings.Count(preview, "\n")+1 != previewMaxLines {
		t.Fatalf("预览行数 = %d，想要 %d", strings.Count(preview, "\n")+1, previewMaxLines)
	}
	if strings.Contains(preview, "第 20 条") {
		t.Fatalf("预览越过了前 20 行：\n%s", preview)
	}
	if !tt.HasText("预览前 20 行") {
		t.Fatalf("没有标出预览上限：%v", tt.Texts())
	}
}

// TestExportEmptySessionState：没有回合时面板仍打开，给空态且导出按钮禁用。
func TestExportEmptySessionState(t *testing.T) {
	snap := SessionSnapshot{ID: "s1", Title: "空会话", Branch: "main"}
	_, tt := exportTestApp(t, snap)
	if !tt.HasText(exportEmptyTurns) {
		t.Fatalf("空会话没有给空态：%v", tt.Texts())
	}
	for _, label := range []string{"导出到剪贴板", "导出到文件"} {
		if info, ok := tt.Access(label); !ok || !info.Disabled {
			t.Fatalf("%q 没有禁用：%+v（找到 %v）", label, info, ok)
		}
	}
}
