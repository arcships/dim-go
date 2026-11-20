package ui

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是 docs/cleanliness.md 第 1、2 节的可执行断言：连着事实时屏上不出现
// 样例，用户可见文案里不出现开发口径。两件事都是「指着代码说没有」，所以写成
// 检查而不是约定。

// sampleMarkers 是样例独有的内容：只在 -demo（没有 Source）时才会出现在屏上。
// 取的是各样例入口里最短、最不可能与真实会话重名的一段。
var sampleMarkers = []string{
	sampleWorkspaceTitle,            // 「dim-agent」工作区名
	sampleModel,                     // 「seed-2.1-pro」样例模型
	sampleRecentProvider,            // 「DimAgent OAuth」最近使用
	sampleProjectA,                  // 「项目 A」
	sampleProjectB,                  // 「项目 B」
	"修发送路径",                         // sessionSamples 的第一条
	"旧实现对照",                         // sessionSamples 的第三条
	"/example/dim-agent",            // sampleSessionPlaces 的工作目录
	"feature/queue-rework",          // branchSamples 的分支
	"看看这次的失败",                       // sampleRounds 的第一轮
	"把发送路径改成排队",                     // 样例目标
	"run-sample",                    // sampleShellRun 的运行标识
	"hello from the preview sample", // rpSamplePreview 的正文
	"再补一条错误分支的测试",                   // sampleQueued 的排队消息
	"临时输出：不要了",                      // termSamples 的样例输出
	"sample-approval",               // 样例审批标识
}

// TestConnectedModeShowsNoSampleData 把连着 Source 的界面逐屏画一遍：主界面、
// 右栏每一种标签、设置每一节、命令面板，都不许出现样例内容。
//
// 这一条替代「逐个入口读代码判断」：样例入口有三十多处，只有把所有面都画出来
// 才能证明「连上事实时样例不上屏」。
func TestConnectedModeShowsNoSampleData(t *testing.T) {
	app, tt := cleanlinessWiredApp(t)

	check := func(where string) {
		t.Helper()
		texts := strings.Join(tt.Texts(), "\n")
		for _, marker := range sampleMarkers {
			if strings.Contains(texts, marker) {
				t.Fatalf("%s 出现样例内容 %q：\n%s", where, marker, texts)
			}
		}
	}

	check("主界面")
	// 右栏：每一种标签都画一遍（文件、改动、任务、子代理、侧聊、终端、MCP）。
	for tab := 0; tab < tabCount; tab++ {
		app.rightOpen, app.rightTab = true, tab
		tt.Frame()
		check("右栏标签 " + strconv.Itoa(tab))
	}
	app.rightOpen = false
	// 设置：19 节逐节画。
	app.screen = ScreenSettings
	for _, section := range settingsSectionDefs {
		app.settingsSection = section.id
		tt.Frame()
		check("设置 · " + section.label)
	}
	// 命令面板（文件组读的是右栏同一份投影）。
	app.screen = ScreenMain
	app.palette.open = true
	tt.Frame()
	check("命令面板")
	app.palette.open = false
	// 提及弹层的文件候选与「会话」分组：同一份目录投影。
	app.draft = "@"
	tt.Frame()
	check("提及弹层")
	app.draft = ""
}

// cleanlinessWiredApp 建一个连着事实的 App（没有 intents，动作只留在视图里）。
func cleanlinessWiredApp(t *testing.T) (*App, *gui.Tester) {
	t.Helper()
	resetScreens(t)
	app := NewApp(Options{State: StateDefault, Source: wiredSource()})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()
	return app, tt
}

// devTerms 是开发口径：这些词属于实现内部，不该出现在用户可见的文案里。
// 「快照 / 宿主 / Source」也在内，但规范逐字给的中文文案里出现时走 verbatimCopy。
var devTerms = []string{
	"读数", "通道", "快照", "Source", "宿主", "protocol", "协议方法",
	"本版不做", "暂未", "界面标记", "尚未影响发送", "TODO", "FIXME",
}

// verbatimCopy 是与需求逐字对齐的文案（TX-EXACT 行）：它们里的「快照」「宿主」
// 是参考产品的原话，改一个字就与需求对不上，所以不按开发口径处理。
var verbatimCopy = map[string]bool{
	exportStaleNotice:    true, // R-GF-107 会话快照已过期…
	permCopySandboxIn:    true, // R-PRM-065 使用宿主沙箱后端运行命令。
	psCopyAgentsFileNote: true, // R-PER-573 PromptSources 会读取此 AGENTS.md…
	"TODO.md":            true, // -demo 文件树里的一行文件名
}

// TestUserFacingCopyHasNoDevTerms 扫包内全部字符串字面量：用户可见的文案里不
// 出现开发口径（缺哪个协议方法、Source 有没有读数、本版不做）。
//
// 技术口径写在注释里不受影响：只看字面量，不看注释。
func TestUserFacingCopyHasNoDevTerms(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatalf("列举包内文件：%v", err)
	}
	scanned := 0
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatalf("读 %s：%v", name, err)
		}
		file, err := parser.ParseFile(token.NewFileSet(), name, src, 0)
		if err != nil {
			t.Fatalf("解析 %s：%v", name, err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			lit, ok := node.(*ast.BasicLit)
			if !ok || lit.Kind != token.STRING {
				return true
			}
			value, err := strconv.Unquote(lit.Value)
			if err != nil || verbatimCopy[value] {
				return true
			}
			scanned++
			for _, term := range devTerms {
				if strings.Contains(value, term) {
					line := strings.Count(string(src[:lit.Pos()-1]), "\n") + 1
					t.Fatalf("%s:%d 的文案带开发口径 %q：%q", name, line, term, value)
				}
			}
			return true
		})
	}
	if scanned == 0 {
		t.Fatal("没有扫到任何文案：检查器本身失效了")
	}
}

// TestNoOpenSessionCopyIsUniform 同一件事实（没有打开会话）在每一屏用同一句：
// 面板、命令面板、斜杠命令与输入区不再一处写「未打开会话」、一处写「还没有打开
// 会话」。
func TestNoOpenSessionCopyIsUniform(t *testing.T) {
	if noteNoOpenSession != "未打开会话" {
		t.Fatalf("共用文案 = %q，想要「未打开会话」", noteNoOpenSession)
	}
	resetScreens(t)
	src := wiredSource()
	src.hasSess, src.session = false, SessionSnapshot{}
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	tt.Frame()
	screens := map[string]func(){
		"右栏 · 文件":  func() { app.rightOpen, app.rightTab = true, tabFiles },
		"右栏 · 任务":  func() { app.rightOpen, app.rightTab = true, tabTasks },
		"右栏 · 子代理": func() { app.rightOpen, app.rightTab = true, tabSubagent },
		"右栏 · 侧聊":  func() { app.rightOpen, app.rightTab = true, tabSideChat },
		"命令面板":     func() { app.palette.open = true },
		"斜杠命令":     func() { app.draft = "/" },
	}
	for where, open := range screens {
		app.rightOpen, app.palette.open, app.draft = false, false, ""
		open()
		tt.Frame()
		texts := strings.Join(tt.Texts(), "\n")
		if strings.Contains(texts, "还没有打开会话") {
			t.Fatalf("%s 仍写「还没有打开会话」，与面板的「未打开会话」不一致：\n%s", where, texts)
		}
	}
}
