package ui

import (
	"context"
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// stubIntents 只实现本批接线要断言的写入口；其余方法走嵌入的 nil 接口，
// 被调用会直接 panic：说明测试点到了一个没接的动作。
type stubIntents struct {
	Intents
	models  chan [2]string
	tiers   chan string
	created chan string
}

func (s *stubIntents) SelectModel(_ context.Context, providerID, modelID string) error {
	s.models <- [2]string{providerID, modelID}
	return nil
}

func (s *stubIntents) SetPermissionTier(_ context.Context, tier string) error {
	s.tiers <- tier
	return nil
}

func (s *stubIntents) CreateSession(_ context.Context, cwd string) error {
	s.created <- cwd
	return nil
}

// waitString 等一个写入意图的实参，超时即失败（不 sleep 轮询）。
func waitString(t *testing.T, ch chan string, what string) string {
	t.Helper()
	select {
	case got := <-ch:
		return got
	case <-time.After(2 * time.Second):
		t.Fatalf("%s 没有发出", what)
		return ""
	}
}

// 模型浮层读 Settings().Models：列出真实模型、点选后发 SelectModel（provider.select），
// 当前模型标记跟着事实走；样例清单不上屏。
func TestModelOverlaySelectsModel(t *testing.T) {
	src := wiredSource()
	src.settings.Models = []ModelOption{
		{ID: "model-x", ProviderID: "acme", Name: "Model-X"},
		{ID: "model-y", ProviderID: "acme", Name: "Model-Y"},
	}
	src.settings.Model, src.settings.ModelID = "Model-X", "model-x"
	stub := &stubIntents{models: make(chan [2]string, 1)}
	app := NewApp(Options{State: StateDefault, Source: src, Intents: stub})
	app.modelOpen = true
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	if !tt.HasText("模型：Model-Y") {
		t.Fatalf("模型浮层没有列出 Source 里的模型：\n%s", strings.Join(tt.Texts(), "\n"))
	}
	if tt.HasText("GPT-5.6 Terra") {
		t.Fatal("连上事实后仍列出样例模型")
	}
	if app.currentModel() != "Model-X" {
		t.Fatalf("当前模型 = %q，想要 Model-X", app.currentModel())
	}
	if err := tt.Click("模型：Model-Y"); err != nil {
		t.Fatalf("点模型：%v", err)
	}
	select {
	case got := <-stub.models:
		if got != [2]string{"acme", "model-y"} {
			t.Fatalf("SelectModel 收到 %v，想要 acme/model-y", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("点模型没有发 SelectModel（provider.select）")
	}
	// 服务端确认新选择后，触发按钮上的模型名跟着事实走。
	src.settings.Model, src.settings.ModelID = "Model-Y", "model-y"
	tt.Frame()
	if app.currentModel() != "Model-Y" {
		t.Fatalf("选择写入后当前模型 = %q，想要 Model-Y", app.currentModel())
	}
	if !tt.HasText("模型：Model-Y") {
		t.Fatalf("选择写入后按钮没有显示新模型：\n%s", strings.Join(tt.Texts(), "\n"))
	}
}

// 技能浮层读 Extensions().Skills：列出真实技能与描述；点一条把 `$名字` 写进草稿。
func TestSkillsOverlayUsesSource(t *testing.T) {
	src := wiredSource()
	src.extensions.Skills = ExtListSnapshot{Items: []ExtItemSnapshot{
		{ID: "skill-1", Name: "dispatch-work", Description: "真实技能说明", Enabled: true},
	}}
	app := NewApp(Options{State: StateDefault, Source: src})
	app.skillsOpen = true
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	if !tt.HasText("dispatch-work") {
		t.Fatalf("技能浮层没有列出 Source 里的技能：\n%s", strings.Join(tt.Texts(), "\n"))
	}
	if !tt.HasText("真实技能说明") {
		t.Fatalf("技能浮层没有显示真实描述：\n%s", strings.Join(tt.Texts(), "\n"))
	}
	if tt.HasText("agents-sdk") {
		t.Fatal("连上事实后仍列出样例技能")
	}
	if err := tt.Click("技能：dispatch-work"); err != nil {
		t.Fatalf("点技能：%v", err)
	}
	if got := strings.TrimSpace(app.draft); got != "$dispatch-work" {
		t.Fatalf("点技能后草稿 = %q，想要 $dispatch-work", got)
	}
}

// 权限档位：状态条与输入区两处入口都发 SetPermissionTier；完全放开仍是两步确认。
func TestPermissionTierEntriesSendIntent(t *testing.T) {
	t.Run("状态条", func(t *testing.T) {
		src := wiredSource()
		src.settings.Tier = "workspace-write"
		stub := &stubIntents{tiers: make(chan string, 1)}
		app := NewApp(Options{State: StateDefault, Source: src, Intents: stub})
		tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

		if !tt.HasText("权限：默认权限") {
			t.Fatalf("状态条没有读会话档位：\n%s", strings.Join(tt.Texts(), "\n"))
		}
		if err := tt.Click("权限：默认权限"); err != nil {
			t.Fatalf("点权限选择器：%v", err)
		}
		tt.Frame()
		if err := tt.Click("权限：智能权限"); err != nil {
			t.Fatalf("点智能权限：%v", err)
		}
		if got := waitString(t, stub.tiers, "权限切换"); got != "workspace-write" {
			t.Fatalf("SetPermissionTier 收到 %q，想要 workspace-write", got)
		}
	})

	t.Run("完全访问两步确认", func(t *testing.T) {
		src := wiredSource()
		src.settings.Tier = "workspace-write"
		stub := &stubIntents{tiers: make(chan string, 1)}
		app := NewApp(Options{State: StateDefault, Source: src, Intents: stub})
		tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

		if err := tt.Click("权限：默认权限"); err != nil {
			t.Fatalf("点权限选择器：%v", err)
		}
		tt.Frame()
		if err := tt.Click("权限：完全访问权限"); err != nil {
			t.Fatalf("点完全访问权限：%v", err)
		}
		tt.Frame()
		if !app.permissionArmOpen {
			t.Fatal("完全访问权限没有先出确认框")
		}
		select {
		case tier := <-stub.tiers:
			t.Fatalf("确认前就发了 %q", tier)
		default:
		}
		if err := tt.Click("我已了解风险，并愿意继续"); err != nil {
			t.Fatalf("点风险勾选：%v", err)
		}
		tt.Frame()
		if err := tt.Click("确认开启"); err != nil {
			t.Fatalf("点确认开启：%v", err)
		}
		if got := waitString(t, stub.tiers, "完全放开确认"); got != "full-access" {
			t.Fatalf("SetPermissionTier 收到 %q，想要 full-access", got)
		}
	})

	t.Run("输入区控制行：没有会话时与状态条同一条判断", func(t *testing.T) {
		src := wiredSource()
		src.settings.Tier = "workspace-write"
		src.names, src.hasSess, src.session = nil, false, SessionSnapshot{}
		stub := &stubIntents{tiers: make(chan string, 1)}
		app := NewApp(Options{State: StateDefault, Source: src, Intents: stub})
		tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

		if !tt.HasText("权限：默认权限") {
			t.Fatalf("空态输入区没有权限入口：\n%s", strings.Join(tt.Texts(), "\n"))
		}
		if err := tt.Click("权限：默认权限"); err != nil {
			t.Fatalf("点权限选择器：%v", err)
		}
		tt.Frame()
		// 空态没有可写的会话：档位入口与状态条共用 permissionControlLock，
		// 置灰并写原因，不发出注定失败的 SetPermissionTier。
		info, ok := tt.Access("权限：智能权限")
		if !ok || !info.Disabled {
			t.Fatalf("空态里权限档位没有置灰写原因：%+v（找到 %v）", info, ok)
		}
		if info.Description != notePermissionNoSession {
			t.Fatalf("置灰原因 = %q，想要 %q", info.Description, notePermissionNoSession)
		}
		select {
		case tier := <-stub.tiers:
			t.Fatalf("没有会话却发了 SetPermissionTier(%q)", tier)
		default:
		}
	})
}

// 工作区清单进 Source：空态选择器列服务端授权的工作区；新建会话（左栏与命令
// 面板是同一条路径）用选中的工作区作 cwd。
func TestWorkspaceSelectorListsSourceWorkspaces(t *testing.T) {
	src := wiredSource()
	src.names, src.hasSess, src.session = nil, false, SessionSnapshot{}
	src.workspaces = []WorkspaceSnapshot{
		{Root: 1, Path: "/tmp/dimagent-test/ws-alpha"},
		{Root: 2, Path: "/tmp/dimagent-test/ws-beta"},
	}
	stub := &stubIntents{created: make(chan string, 2)}
	app := NewApp(Options{State: StateDefault, Source: src, Intents: stub})
	app.workspaceOpen = true
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	if !tt.HasText("ws-alpha") || !tt.HasText("ws-beta") {
		t.Fatalf("工作区选择器没有列出 Source 里的工作区：\n%s", strings.Join(tt.Texts(), "\n"))
	}
	if err := tt.Click("工作区：ws-beta"); err != nil {
		t.Fatalf("点工作区：%v", err)
	}
	if app.workspacePath != "/tmp/dimagent-test/ws-beta" {
		t.Fatalf("选中的工作区 = %q", app.workspacePath)
	}

	// 命令面板的「新建会话」不再置灰，执行后把当前工作区当 cwd 发出去。
	var newSession *commandItem
	for _, item := range app.commandPaletteActions() {
		if item.id == "action:new-session" {
			copied := item
			newSession = &copied
		}
	}
	if newSession == nil {
		t.Fatal("命令面板缺少「新建会话」")
	}
	if newSession.disabled {
		t.Fatalf("「新建会话」仍置灰：%s", newSession.reason)
	}
	newSession.run()
	if got := waitString(t, stub.created, "新建会话"); got != "/tmp/dimagent-test/ws-beta" {
		t.Fatalf("CreateSession 收到 cwd %q，想要 /tmp/dimagent-test/ws-beta", got)
	}
}

// Goal 模式不再置灰：切到 Goal 且还没有目标时打开目标行的编辑框（写入仍走
// SetGoal 那条既有路径）。
func TestGoalModeEntryIsWired(t *testing.T) {
	src := wiredSource()
	app := NewApp(Options{State: StateDefault, Source: src, Intents: &stubIntents{}})
	app.modeOpen = true
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	if tt.HasText("缺少目标事实") {
		t.Fatal("Goal 模式仍写着「缺少目标事实」")
	}
	if err := tt.Click("Goal 模式"); err != nil {
		t.Fatalf("点 Goal 模式：%v", err)
	}
	if app.mode != "Goal" {
		t.Fatalf("模式 = %q，想要 Goal", app.mode)
	}
	if !app.goalEditing {
		t.Fatal("还没有目标时没有打开目标编辑框")
	}
	tt.Frame()
	if !tt.HasText("目标") {
		t.Fatalf("目标编辑框没有出现：\n%s", strings.Join(tt.Texts(), "\n"))
	}
}

// 只读只在服务端回执明确给出时显示（DEC-01）：它被翻译成「只读」，且不出现在
// 可选的三档菜单里。
func TestPermissionReadOnlyOnlyFromReceipt(t *testing.T) {
	src := wiredSource()
	src.settings.Tier = "read-only"
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)

	if !tt.HasText("权限：只读") {
		t.Fatalf("回执给出 read-only 时没有显示「只读」：\n%s", strings.Join(tt.Texts(), "\n"))
	}
	if err := tt.Click("权限：只读"); err != nil {
		t.Fatalf("点权限选择器：%v", err)
	}
	tt.Frame()
	for _, label := range []string{"权限：默认权限", "权限：智能权限", "权限：完全访问权限"} {
		if _, ok := tt.Find(label); !ok {
			t.Fatalf("三档菜单缺 %q：%v", label, tt.Texts())
		}
	}
	// 菜单里不出现可选的「只读」项：三档以外的取值不可写。
	if _, ok := tt.Find("权限：只读"); !ok {
		t.Fatal("触发按钮的只读文案不应随菜单打开消失")
	}
}

// 思考强度没有写入口（DEC-15）：滑杆只显示当前档，拖动不改本地值，原因在浮层里。
func TestEffortSliderIsDisplayOnly(t *testing.T) {
	app, tt := newTester(t, StateDefault)
	app.effortOpen = true
	tt.Frame()
	if !tt.HasText(noteThinking) {
		t.Fatalf("思考强度浮层没有写原因：%v", tt.Texts())
	}
	if info, ok := tt.Access("思考强度滑杆（" + noteThinking + "）"); !ok || !info.Disabled {
		t.Fatalf("思考强度滑杆没有禁用：%+v（找到 %v）", info, ok)
	}
	// 禁用控件不接收拖动：Tester 的指针事件不会改值。
	before := app.effort
	if box, ok := tt.Find("思考强度滑杆（" + noteThinking + "）"); ok {
		tt.Press(box.X+box.W-2, box.Y+box.H/2)
		tt.Move(box.X+2, box.Y+box.H/2)
		tt.Release(box.X+2, box.Y+box.H/2)
	}
	if app.effort != before {
		t.Fatalf("禁用滑杆改了本地值：%v → %v", before, app.effort)
	}
}
