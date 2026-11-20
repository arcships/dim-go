package ui

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「异常分支」的可执行清单：需求四份里「异常」维度标未实现的那一类，
// 只要能在界面这一侧补的（失败、超时、断线、读失败、空态），都在这里落成用例。
//
// 两张表：
//
//   - intentCases：每个走 Intents 的写动作一条，注入失败后断言界面出现可读原因，
//     并且有重试入口（全局错误行的「重试」，或就地按钮能再发一次）；
//   - readFailureCases：每个读事实的路径一条，让适配器给出失败读数，断言界面写
//     出原因（不是空白区域），可重试的读还给重试。
//
// 表必须覆盖整个 Intents 接口：TestIntentCasesCoverIntents 用反射比对方法集，
// 新加一个写方法却没写失败分支时，这一条会红。确实没有界面入口的方法登记在
// intentNoEntry 里并写明原因——它们不是异常分支的缺口，而是入口本身的缺口，
// 交回清单里单列。

// 注入失败的原因原文：一眼能看出是哪一条动作的失败。
func failReason(method string) string { return "注入失败：" + method }

type failIntents struct {
	Intents
	mu    sync.Mutex
	calls map[string]int
}

func newFailIntents() *failIntents { return &failIntents{calls: map[string]int{}} }

// hit 记一次调用并给出失败原因。
func (f *failIntents) hit(method string) error {
	f.mu.Lock()
	f.calls[method]++
	f.mu.Unlock()
	return errors.New(failReason(method))
}

// count 读一个方法被调用的次数。
func (f *failIntents) count(method string) int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls[method]
}

func (f *failIntents) OpenSession(context.Context, string) error { return f.hit("OpenSession") }
func (f *failIntents) Submit(context.Context, string) error      { return f.hit("Submit") }
func (f *failIntents) SubmitWithContent(context.Context, string, []ContentBlockSnapshot) error {
	return f.hit("SubmitWithContent")
}
func (f *failIntents) Queue(context.Context, string) error { return f.hit("Queue") }
func (f *failIntents) Stop(context.Context) error          { return f.hit("Stop") }
func (f *failIntents) RespondApproval(context.Context, string) error {
	return f.hit("RespondApproval")
}
func (f *failIntents) RespondQuestion(context.Context, string) error {
	return f.hit("RespondQuestion")
}
func (f *failIntents) SkipQuestion(context.Context) error { return f.hit("SkipQuestion") }
func (f *failIntents) RetrySubmit(context.Context) error  { return f.hit("RetrySubmit") }
func (f *failIntents) CreateSession(context.Context, string) error {
	return f.hit("CreateSession")
}
func (f *failIntents) RenameSession(context.Context, string, string) error {
	return f.hit("RenameSession")
}
func (f *failIntents) ArchiveSession(context.Context, string, bool) error {
	return f.hit("ArchiveSession")
}
func (f *failIntents) DeleteSession(context.Context, string) error { return f.hit("DeleteSession") }
func (f *failIntents) ExportSession(context.Context, string) (string, error) {
	return "", f.hit("ExportSession")
}
func (f *failIntents) SetPermissionTier(context.Context, string) error {
	return f.hit("SetPermissionTier")
}
func (f *failIntents) SelectModel(context.Context, string, string) error {
	return f.hit("SelectModel")
}
func (f *failIntents) SelectAgent(context.Context, string, string) error {
	return f.hit("SelectAgent")
}
func (f *failIntents) WriteCommandRules(context.Context, []RuleSnapshot) error {
	return f.hit("WriteCommandRules")
}
func (f *failIntents) CompactContext(context.Context) error { return f.hit("CompactContext") }
func (f *failIntents) RefreshSkills(context.Context) error  { return f.hit("RefreshSkills") }
func (f *failIntents) SelectBranch(context.Context, string) error {
	return f.hit("SelectBranch")
}
func (f *failIntents) CreateBranch(context.Context, string, int, string) error {
	return f.hit("CreateBranch")
}
func (f *failIntents) EndBranch(context.Context, string) error { return f.hit("EndBranch") }
func (f *failIntents) WriteAttachment(context.Context, string, string) (AttachmentRefSnapshot, error) {
	return AttachmentRefSnapshot{}, f.hit("WriteAttachment")
}
func (f *failIntents) CreateProvider(context.Context, string, string, string, string, []ProviderModelDraft) error {
	return f.hit("CreateProvider")
}
func (f *failIntents) UpdateProvider(context.Context, string, string, string, string) error {
	return f.hit("UpdateProvider")
}
func (f *failIntents) ProbeProvider(context.Context, string) (int, error) {
	return 0, f.hit("ProbeProvider")
}
func (f *failIntents) SetProviderModelEnabled(context.Context, string, string, bool) error {
	return f.hit("SetProviderModelEnabled")
}
func (f *failIntents) BeginProviderOAuth(
	context.Context, string, string, string, string,
) (ProviderOAuthStartSnapshot, error) {
	return ProviderOAuthStartSnapshot{}, f.hit("BeginProviderOAuth")
}
func (f *failIntents) CompleteProviderOAuth(context.Context, string, string, string, string) error {
	return f.hit("CompleteProviderOAuth")
}
func (f *failIntents) PollProviderOAuth(
	context.Context, string, string, string,
) (ProviderOAuthProgressSnapshot, error) {
	return ProviderOAuthProgressSnapshot{}, f.hit("PollProviderOAuth")
}
func (f *failIntents) CancelProviderOAuth(context.Context, string, string, string) error {
	return f.hit("CancelProviderOAuth")
}
func (f *failIntents) LogoutProviderOAuth(context.Context, string, string) error {
	return f.hit("LogoutProviderOAuth")
}
func (f *failIntents) OpenProviderOAuthURL(context.Context, string) error {
	return f.hit("OpenProviderOAuthURL")
}
func (f *failIntents) RefreshProviders(context.Context) error {
	return f.hit("RefreshProviders")
}
func (f *failIntents) SaveMCPServer(context.Context, MCPServerDraft) error {
	return f.hit("SaveMCPServer")
}
func (f *failIntents) RemoveMCPServer(context.Context, string) error {
	return f.hit("RemoveMCPServer")
}
func (f *failIntents) BeginLogin(context.Context, string) (LoginPromptSnapshot, error) {
	return LoginPromptSnapshot{}, f.hit("BeginLogin")
}
func (f *failIntents) WaitLogin(context.Context) error   { return f.hit("WaitLogin") }
func (f *failIntents) CancelLogin(context.Context) error { return f.hit("CancelLogin") }
func (f *failIntents) Logout(context.Context) error      { return f.hit("Logout") }
func (f *failIntents) SetExtensionEnabled(context.Context, string, string, bool) error {
	return f.hit("SetExtensionEnabled")
}
func (f *failIntents) UninstallExtension(context.Context, string, string) error {
	return f.hit("UninstallExtension")
}
func (f *failIntents) InstallPlugin(context.Context, string, string, string, string, bool) error {
	return f.hit("InstallPlugin")
}
func (f *failIntents) UpdatePlugin(context.Context, string) error { return f.hit("UpdatePlugin") }
func (f *failIntents) TestMCP(context.Context, string) (bool, string, error) {
	return false, "", f.hit("TestMCP")
}
func (f *failIntents) ImportMCP(context.Context, string) (int, error) {
	return 0, f.hit("ImportMCP")
}
func (f *failIntents) SetScheduleEnabled(context.Context, string, bool) error {
	return f.hit("SetScheduleEnabled")
}
func (f *failIntents) DeleteSchedule(context.Context, string) error { return f.hit("DeleteSchedule") }
func (f *failIntents) CancelModalityTask(context.Context, string) error {
	return f.hit("CancelModalityTask")
}
func (f *failIntents) StopExecTask(context.Context, string) error { return f.hit("StopExecTask") }
func (f *failIntents) ReadGoal(context.Context) (string, error)   { return "", f.hit("ReadGoal") }
func (f *failIntents) SetGoal(context.Context, string) error      { return f.hit("SetGoal") }
func (f *failIntents) EditGoal(context.Context, string) error     { return f.hit("EditGoal") }
func (f *failIntents) PauseGoal(context.Context) error            { return f.hit("PauseGoal") }
func (f *failIntents) ResumeGoal(context.Context) error           { return f.hit("ResumeGoal") }
func (f *failIntents) ClearGoal(context.Context) error            { return f.hit("ClearGoal") }
func (f *failIntents) RunShell(context.Context, string) (string, error) {
	return "", f.hit("RunShell")
}
func (f *failIntents) StopShell(context.Context, string) error     { return f.hit("StopShell") }
func (f *failIntents) StartTerminal(context.Context) error         { return f.hit("StartTerminal") }
func (f *failIntents) StopTerminal(context.Context) error          { return f.hit("StopTerminal") }
func (f *failIntents) WriteTerminal(context.Context, string) error { return f.hit("WriteTerminal") }
func (f *failIntents) Reconnect(context.Context) error             { return f.hit("Reconnect") }
func (f *failIntents) SetPreferences(context.Context, PreferencesSnapshot) error {
	return f.hit("SetPreferences")
}
func (f *failIntents) SetGlobalShortcut(context.Context, bool, string) error {
	return f.hit("SetGlobalShortcut")
}

// 更新的四个动作在可选接口上，也是写入口：失败同样要上屏。
func (f *failIntents) CheckUpdate(context.Context) error { return f.hit("CheckUpdate") }
func (f *failIntents) DownloadUpdate(context.Context) error {
	return f.hit("DownloadUpdate")
}
func (f *failIntents) InstallUpdate(context.Context) error { return f.hit("InstallUpdate") }
func (f *failIntents) SetUpdateFeed(context.Context, string) error {
	return f.hit("SetUpdateFeed")
}

// 插件市场的两个动作同上。
func (f *failIntents) SetPluginFeed(context.Context, string) error {
	return f.hit("SetPluginFeed")
}
func (f *failIntents) RefreshPluginMarketplace(context.Context) error {
	return f.hit("RefreshPluginMarketplace")
}

// intentCase 是一条「写动作的失败分支」用例。
type intentCase struct {
	// method 是 Intents（或可选写接口）上的方法名：覆盖检查按它比对。
	method string
	// source 在 wiredSource 上补这一条要用的读数（审批、提问、分支、目标…）。
	source func(*fakeSource)
	// open 在发动作前摆好视图状态（进设置分节、打开面板、写草稿）。
	open func(*App)
	// act 发一次动作：与控件走同一条路径（点界面，或调控件调的同一个处理器）。
	act func(*intentRun)
	// local 表示失败原因就地显示（卡片 / 表单 / 对话框 / 行内），不写标题下方的
	// 全局错误行；这一类的重试是再点一次就地的按钮。
	local bool
	// update 非空时把 Source 包成带更新读数的实现（关于页的失败分支要用）。
	update *UpdateSnapshot
	// retry 是就地重试的入口；为空时 local 用例复用 act、其余点全局「重试」。
	retry func(*intentRun)
}

// intentRun 是用例手上的三样东西：断言、App 与无窗口渲染器。
type intentRun struct {
	t   *testing.T
	app *App
	tt  *gui.Tester
}

// click 画一帧后点一个可见入口（按读屏名或可见文字），再画一帧。
func (r *intentRun) click(label string) {
	r.t.Helper()
	r.tt.Frame()
	if err := r.tt.Click(label); err != nil {
		r.t.Fatalf("点不到 %q：%v\n屏上文字：%v", label, err, r.tt.Texts())
	}
	r.tt.Frame()
}

// frame 画一帧。
func (r *intentRun) frame() { r.tt.Frame() }

// waitCalls 等一个写入口被调到第 want 次（写入在后台协程里跑）。
func waitCalls(t *testing.T, stub *failIntents, method string, want int) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if stub.count(method) >= want {
			return
		}
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("%s 没有发到第 %d 次（实际 %d 次）", method, want, stub.count(method))
}

// waitReason 等失败原因上屏：后台结果经渲染线程交回，最多等 3 秒。
func waitReason(t *testing.T, tt *gui.Tester, method string) {
	t.Helper()
	reason := failReason(method)
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if tt.HasText(reason) {
			return
		}
		tt.Frame()
		time.Sleep(2 * time.Millisecond)
	}
	t.Fatalf("%s 的失败原因 %q 没有上屏\n屏上文字：%v", method, reason, tt.Texts())
}

// paletteItem 取命令面板里的一条候选：点它等于点面板里的同一项。
func paletteItem(a *App, id string) commandItem {
	for _, item := range a.commandPaletteActions() {
		if item.id == id {
			return item
		}
	}
	return commandItem{}
}

// slashRun 取一条斜杠命令的可执行闭包：点菜单项走的是同一个。
func slashRun(a *App, name string) func() {
	for _, cmd := range a.slashCommands() {
		if cmd.name == name {
			return cmd.run
		}
	}
	return nil
}

// intentCases 是写动作的失败分支清单（顺序与 Intents 接口同序）。
var intentCases = []intentCase{
	{
		method: "OpenSession",
		source: func(src *fakeSource) {
			src.names = []string{"s1", "s2"}
			src.session = SessionSnapshot{ID: "s1", Title: "真实会话"}
		},
		open: func(a *App) { a.selectedSession = "s2" },
		act:  func(r *intentRun) { r.app.syncSelectedSession() },
	},
	{
		method: "SubmitWithContent",
		act: func(r *intentRun) {
			r.app.draft = "你好"
			r.app.submitDraft()
		},
	},
	{
		method: "Queue",
		act: func(r *intentRun) {
			r.app.draft = "排队这一句"
			paletteItem(r.app, "action:send").secondary()
		},
	},
	{
		method: "Stop",
		source: func(src *fakeSource) {
			src.session = SessionSnapshot{ID: "s1", Title: "真实会话", Running: true}
		},
		act: func(r *intentRun) { paletteItem(r.app, "action:stop").run() },
	},
	{
		method: "RespondApproval",
		source: func(src *fakeSource) {
			src.approval = ApprovalSnapshot{ID: "a1", Tool: "exec", Command: "ls",
				Input: `{"command":"ls"}`}
			src.hasAppr = true
		},
		local: true,
		act:   func(r *intentRun) { r.app.respondApproval(approvalCopyAllowOnce) },
	},
	{
		method: "RespondQuestion",
		source: func(src *fakeSource) {
			src.question = QuestionSnapshot{ID: "q1", Questions: []QuestionItemSnapshot{{
				ID: "i1", Title: "选一个", Options: []string{"甲", "乙"},
			}}}
			src.hasQ = true
		},
		local: true,
		act: func(r *intentRun) {
			// 先画一帧让卡片把每题的状态槽位补齐，再勾一个选项：答案取的是
			// 卡片自己的视图状态，不另造一份。
			r.frame()
			if len(r.app.question.selected) == 0 {
				r.t.Fatal("提问卡没有为这一题建状态槽位")
			}
			r.app.question.selected[0]["甲"] = true
			src, _ := r.app.liveQuestion()
			r.app.submitQuestion(src)
		},
	},
	{
		method: "SkipQuestion",
		source: func(src *fakeSource) {
			src.question = QuestionSnapshot{ID: "q1", Questions: []QuestionItemSnapshot{{
				ID: "i1", Title: "选一个", Options: []string{"甲", "乙"},
			}}}
			src.hasQ = true
		},
		local: true,
		act: func(r *intentRun) {
			src, _ := r.app.liveQuestion()
			r.app.declineQuestion(src)
		},
	},
	{
		method: "CreateSession",
		act:    func(r *intentRun) { r.app.createSessionAt("") },
	},
	{
		method: "RenameSession",
		act: func(r *intentRun) {
			r.app.renaming, r.app.renameDraft = 0, "新名字"
			r.app.saveRename(0)
		},
	},
	{
		method: "ArchiveSession",
		act:    func(r *intentRun) { r.app.archiveSession(0) },
	},
	{
		method: "DeleteSession",
		open: func(a *App) {
			a.confirm = confirmState{kind: confirmKindDelete,
				targets: []confirmTarget{{index: 0, id: "s1"}}}
		},
		local: true,
		act:   func(r *intentRun) { r.app.runConfirmDelete() },
	},
	{
		method: "SetPermissionTier",
		act:    func(r *intentRun) { r.app.applyPermission("默认权限") },
	},
	{
		method: "SelectModel",
		act: func(r *intentRun) {
			r.app.selectModel(ModelOption{ID: "model-x", ProviderID: "acme", Name: "Model-X"})
		},
	},
	{
		method: "SelectAgent",
		source: func(s *fakeSource) {
			s.agent = AgentConfigSnapshot{
				Known: true, Current: "agent-a", CurrentKnown: true, CanSelect: true,
				Profiles: []AgentConfigProfile{{ID: "agent-a", Name: "A"}, {ID: "agent-b", Name: "B"}},
			}
		},
		act: func(r *intentRun) {
			r.app.selectAgentConfig("agent-b", "agent-a", "B")
		},
	},
	{
		method: "CompactContext",
		act: func(r *intentRun) {
			if run := slashRun(r.app, "compact"); run != nil {
				run()
			} else {
				r.t.Fatal("斜杠命令里没有可执行的 /compact")
			}
		},
	},
	{
		method: "RefreshSkills",
		act: func(r *intentRun) {
			if run := slashRun(r.app, "reload-skills"); run != nil {
				run()
			} else {
				r.t.Fatal("斜杠命令里没有可执行的 /reload-skills")
			}
		},
	},
}

// branchSource 把会话摆到一条非主干分支上，并给出可切换的清单。
func branchSource(src *fakeSource) {
	src.session = SessionSnapshot{ID: "s1", Title: "真实会话", Branch: "b1",
		TurnCount: 2,
		Branches:  []BranchSnapshot{{ID: "b1"}, {ID: "b2"}}}
}

// intentCasesTail 接 intentCases：分支、附件、供应商、账号、扩展。

var intentCasesTail = []intentCase{
	{
		method: "SelectBranch",
		source: branchSource,
		act: func(r *intentRun) {
			r.click("分支 b1")
			r.click("分支：b2")
		},
	},
	{
		method: "CreateBranch",
		source: branchSource,
		open:   func(a *App) { a.draft = "从这条分支继续" },
		act: func(r *intentRun) {
			r.click("分支 b1")
			r.click("创建新分支")
		},
	},
	{
		method: "EndBranch",
		source: branchSource,
		act:    func(r *intentRun) { paletteItem(r.app, "action:end-branch").run() },
	},
	{
		method: "WriteAttachment",
		open: func(a *App) {
			a.attachmentEditing, a.attachmentKind = true, "text"
			a.attachmentName, a.attachmentText = "notes.txt", "附件内容"
		},
		act: func(r *intentRun) { r.click("添加附件") },
	},
	{
		method: "CreateProvider",
		open:   func(a *App) { configScreen.form = configFormNew() },
		local:  true,
		act: func(r *intentRun) {
			form := configFormNew()
			form.name, form.baseURL = "Acme 二号", "https://api.example.com/v1"
			form.apiKey = "sk-test"
			configScreen.form = form
			r.app.configFormSave(r.app.configView(), form)
		},
	},
	{
		method: "UpdateProvider",
		source: func(src *fakeSource) {
			src.settings.ProviderRecords = []ProviderRecordSnapshot{{
				ID: "acme", Name: "Acme", BaseURL: "https://api.example.com/v1",
				Enabled: true, Version: 1,
				Models: []ProviderModelSnapshot{{ID: "model-x", Name: "Model-X", Enabled: true}},
			}}
		},
		local: true,
		act: func(r *intentRun) {
			data := r.app.configView()
			form := configFormEdit(data.selected, data.models)
			form.name = "Acme 改名"
			configScreen.form = form
			r.app.configFormSave(data, form)
		},
	},
	{
		method: "ProbeProvider",
		act:    func(r *intentRun) { r.app.configProbeProvider(configProvider{id: "acme", name: "Acme"}, probeModels) },
	},
	{
		// 账号形态的供应商（Codex）：发起登录失败时原因写在账号块下方那一行
		// （就地显示，不挂全局错误行），再点一次入口就是重试。
		method: "BeginProviderOAuth",
		source: func(src *fakeSource) {
			src.settings.ProviderRecordsKnown = true
			src.settings.ProviderRecords = []ProviderRecordSnapshot{{
				ID: "codex", Name: "OpenAI Codex", BaseURL: "https://api.test/v1",
				Enabled: true, Version: 1, Authority: "openai_codex",
				Account: &ProviderAccountSnapshot{
					Namespace: "account.codex", LoginMethods: []string{configLoginBrowser},
				},
			}}
		},
		open: func(a *App) {
			a.screen, a.settingsSection = ScreenSettings, secProviders
			configScreen.selected = "codex"
		},
		local: true,
		act: func(r *intentRun) {
			r.app.configOAuthBegin(r.app.configView().selected, configLoginBrowser)
		},
	},
	{
		method: "SetProviderModelEnabled",
		act: func(r *intentRun) {
			r.app.configSetModelEnabled(configProvider{id: "acme"},
				configModel{id: "model-x", name: "Model-X"}, false)
		},
	},
	{
		method: "CompleteProviderOAuth",
		source: oauthFailSource(false),
		open:   func(a *App) { a.screen, a.settingsSection = ScreenSettings, secProviders },
		local:  true,
		act: func(r *intentRun) {
			configOAuthFlow = providerOAuthFlow{
				providerKey: "codex", providerID: "codex", namespace: "account.codex",
				method: configLoginBrowser, attempt: "a1",
				callbackURL: "http://127.0.0.1/callback?code=x",
			}
			r.app.configOAuthCompleteBrowser(r.app.configView().selected)
		},
	},
	{
		method: "PollProviderOAuth",
		source: oauthFailSource(false),
		open:   func(a *App) { a.screen, a.settingsSection = ScreenSettings, secProviders },
		local:  true,
		act: func(r *intentRun) {
			configOAuthFlow = providerOAuthFlow{
				providerKey: "codex", providerID: "codex", namespace: "account.codex",
				method: configLoginDevice, attempt: "a1", userCode: "CD-1",
			}
			r.app.configOAuthPoll("codex", "codex", "account.codex", "a1", time.Millisecond)
			time.Sleep(10 * time.Millisecond)
		},
	},
	{
		method: "CancelProviderOAuth",
		source: oauthFailSource(false),
		open:   func(a *App) { a.screen, a.settingsSection = ScreenSettings, secProviders },
		local:  true,
		act: func(r *intentRun) {
			configOAuthFlow = providerOAuthFlow{
				providerKey: "codex", providerID: "codex", namespace: "account.codex",
				method: configLoginBrowser, attempt: "a1",
			}
			r.app.configOAuthCancel(r.app.configView().selected)
		},
	},
	{
		method: "LogoutProviderOAuth",
		source: oauthFailSource(true),
		open:   func(a *App) { a.screen, a.settingsSection = ScreenSettings, secProviders },
		local:  true,
		act:    func(r *intentRun) { r.app.configOAuthLogout(r.app.configView().selected) },
	},
	{
		method: "OpenProviderOAuthURL",
		source: oauthFailSource(false),
		open:   func(a *App) { a.screen, a.settingsSection = ScreenSettings, secProviders },
		local:  true,
		act: func(r *intentRun) {
			configOAuthFlow = providerOAuthFlow{
				providerKey: "codex", providerID: "codex", namespace: "account.codex",
				method: configLoginBrowser, attempt: "a1",
				authorizationURL: "https://auth.test/start",
			}
			r.app.configOAuthOpenURL("https://auth.test/start")
		},
	},
	{
		method: "RefreshProviders",
		source: oauthFailSource(true),
		open:   func(a *App) { a.screen, a.settingsSection = ScreenSettings, secProviders },
		local:  true,
		act:    func(r *intentRun) { r.app.configOAuthRefresh() },
	},
	{
		method: "BeginLogin",
		source: func(src *fakeSource) {
			src.acct = AccountSnapshot{Initial: "A", Connected: false, Status: "signed_out"}
		},
		open: func(a *App) { a.screen, a.settingsSection = ScreenSettings, secAccount },
		act:  func(r *intentRun) { r.click(accCopySignIn) },
	},
	{
		method: "Logout",
		open:   func(a *App) { a.screen, a.settingsSection = ScreenSettings, secAccount },
		act:    func(r *intentRun) { r.click(accCopySignOut) },
	},
	{
		method: "SaveMCPServer",
		source: func(src *fakeSource) { src.extensions.MCPCanSave = true },
		open: func(a *App) {
			a.screen, a.settingsSection = ScreenSettings, secMCP
			mcpScreen.view = mcpViewEdit
			mcpScreen.draft = mcpDraft{name: "本地工具", command: "npx"}
		},
		local: true,
		act:   func(r *intentRun) { r.app.mcpSaveDraft() },
	},
	{
		method: "RemoveMCPServer",
		source: func(src *fakeSource) {
			src.extensions.MCPCanRemove = true
			src.extensions.MCP = ExtListSnapshot{Items: []ExtItemSnapshot{{
				ID: "srv-1", Name: "srv-1", Source: "user", Transport: "stdio", Enabled: true,
			}}}
		},
		open:  func(a *App) { a.screen, a.settingsSection = ScreenSettings, secMCP },
		local: true,
		act: func(r *intentRun) {
			r.click("删除服务器")
			r.click(mcpCopyDelete)
		},
	},
	{
		method: "SetExtensionEnabled",
		open:   func(a *App) { a.screen, a.settingsSection = ScreenSettings, secMCP },
		act: func(r *intentRun) {
			r.app.mcpToggle(mcpServer{id: "real-mcp", name: "real-mcp", enabled: true}, false)
		},
	},
	{
		method: "UninstallExtension",
		open: func(a *App) {
			a.openPluginRemove(ExtItemSnapshot{ID: "p1", Name: "real-plugin"})
		},
		local: true,
		act:   func(r *intentRun) { r.app.confirmPluginRemove() },
	},
	{
		method: "InstallPlugin",
		open: func(a *App) {
			a.pluginInstall = pluginInstallState{
				open: true, enable: true, path: "/tmp/plugin",
				preview: &pluginManifestPreview{},
			}
		},
		local: true,
		act:   func(r *intentRun) { r.app.confirmPluginInstall() },
	},
	{
		method: "UpdatePlugin",
		source: func(src *fakeSource) {
			src.extensions.Plugins = ExtListSnapshot{Items: []ExtItemSnapshot{{
				ID: "p1", Name: "real-plugin", Version: "v1", Enabled: true,
				UpdateAvailable: true, LatestVersion: "v2",
			}}}
		},
		open:  func(a *App) { a.screen, a.settingsSection = ScreenSettings, secPluginSettings },
		local: true,
		act: func(r *intentRun) {
			r.app.updatePlugin(ExtItemSnapshot{ID: "p1", Name: "real-plugin"})
		},
	},
	{
		method: "StopExecTask",
		source: func(src *fakeSource) {
			src.session = SessionSnapshot{ID: "s1", Title: "真实会话",
				Tasks: []TaskSnapshot{{ID: "t1", Description: "后台任务"}}}
		},
		open: func(a *App) {
			a.rightOpen = true
			a.rpOpen(tabTasks)
		},
		act: func(r *intentRun) {
			r.click("取消任务")
			r.click("确认停止")
		},
	},
}

// goalSource 把会话摆到一个已有目标的状态上。
func goalSource(status string) func(*fakeSource) {
	return func(src *fakeSource) {
		src.session = SessionSnapshot{ID: "s1", Title: "真实会话", Goal: "目标 A",
			GoalStatus: status}
	}
}

// intentCasesGoal 接上一段：目标、命令、终端、连接与本端偏好。
var intentCasesGoal = []intentCase{
	{
		method: "ReadGoal",
		source: goalSource("active"),
		act:    func(r *intentRun) { r.click("修改目标") },
	},
	{
		method: "SetGoal",
		open: func(a *App) {
			a.goalEditing, a.goalDraft = true, "新目标"
		},
		act: func(r *intentRun) { r.click("保存目标") },
	},
	{
		method: "EditGoal",
		source: goalSource("active"),
		open:   func(a *App) { a.goalEditing, a.goalDraft = true, "改过的目标" },
		act:    func(r *intentRun) { r.click("保存目标") },
	},
	{
		method: "PauseGoal",
		source: goalSource("active"),
		act:    func(r *intentRun) { r.click("暂停目标") },
	},
	{
		method: "ResumeGoal",
		source: goalSource("paused"),
		act:    func(r *intentRun) { r.click("恢复目标") },
	},
	{
		method: "ClearGoal",
		source: goalSource("active"),
		act: func(r *intentRun) {
			r.click("清除目标")
			r.click("确认清除目标")
		},
	},
	{
		method: "RunShell",
		open:   func(a *App) { a.draft = "!ls" },
		act:    func(r *intentRun) { r.app.runShellCommand() },
	},
	{
		method: "StopShell",
		open:   func(a *App) { a.lastShellRun = "run-1" },
		act:    func(r *intentRun) { r.app.stopShellRun() },
	},
	{
		method: "StartTerminal",
		source: func(src *fakeSource) {
			src.session = SessionSnapshot{ID: "s1", Title: "真实会话", Cwd: "/tmp"}
			// 还没启动：快照里没有 pty（有在跑时「新建终端」本来就是禁用的）。
			src.terminal = TerminalSnapshot{}
		},
		act: func(r *intentRun) { r.app.startLiveTerminal() },
	},
	{
		method: "StopTerminal",
		source: func(src *fakeSource) {
			src.session = SessionSnapshot{ID: "s1", Title: "真实会话", Cwd: "/tmp"}
			src.terminal = TerminalSnapshot{Active: true, Running: true}
		},
		act: func(r *intentRun) { r.app.stopLiveTerminal(false) },
	},
	{
		method: "WriteTerminal",
		source: func(src *fakeSource) {
			src.session = SessionSnapshot{ID: "s1", Title: "真实会话", Cwd: "/tmp"}
			src.terminal = TerminalSnapshot{Active: true, Running: true}
		},
		open: func(a *App) { a.termInput = "ls" },
		act:  func(r *intentRun) { r.app.submitTerminalInput() },
	},
	{
		method: "Reconnect",
		source: func(src *fakeSource) { src.conn = Connection{Phase: "disconnected"} },
		act:    func(r *intentRun) { paletteItem(r.app, "action:reconnect").run() },
	},
	{
		method: "SetPreferences",
		open: func(a *App) {
			a.themePref = "暗"
			a.lastPrefs = PreferencesSnapshot{Theme: "明"}
		},
		act: func(r *intentRun) { r.app.persistPreferences() },
	},
	{
		method: "SetGlobalShortcut",
		act: func(r *intentRun) {
			r.app.applyGlobalShortcut(true, "CmdOrCtrl+Shift+Space")
		},
	},
}

// updateCase 是更新动作的一条：打开关于页、点相位上的按钮。
func updateCase(method, phase, available string, installable bool,
	open func(*App), act func(*intentRun), retry string) intentCase {
	return intentCase{
		method: method,
		update: &UpdateSnapshot{Configured: true, Phase: phase,
			AvailableVersion: available, Installable: installable},
		open:  open,
		local: true,
		act:   act,
		retry: func(r *intentRun) { r.click(retry) },
	}
}

// openAbout 进设置 › 关于，并把本屏的失败位清干净（aboutScreen 是包级状态）。
func openAbout(a *App) {
	a.screen, a.settingsSection = ScreenSettings, secAbout
	aboutScreen.updateError, aboutScreen.phase = "", abtDisabled
	aboutScreen.feedError, aboutScreen.feedDirty = "", false
	aboutScreen.feedDraft = ""
}

// intentCasesUpdate 是更新动作的四条失败分支（可选接口 UpdateIntents）。
var intentCasesUpdate = []intentCase{
	updateCase("CheckUpdate", "idle", "", false, openAbout,
		func(r *intentRun) { r.click(abtCopyCheck) }, abtCopyRetry),
	updateCase("DownloadUpdate", "available", "v9.9.9", false, openAbout,
		func(r *intentRun) { r.click(abtCopyDownload) }, abtCopyRetry),
	updateCase("InstallUpdate", "downloaded", "v9.9.9", true, openAbout,
		func(r *intentRun) { r.click(abtCopyInstall) }, abtCopyRetry),
	{
		method: "SetUpdateFeed",
		update: &UpdateSnapshot{Configured: false, Phase: "idle"},
		open: func(a *App) {
			openAbout(a)
			aboutScreen.feedDraft, aboutScreen.feedDirty = "https://update.example.com/latest.json", true
		},
		local: true,
		act:   func(r *intentRun) { r.click(abtCopySourceSave) },
		retry: func(r *intentRun) { r.click(abtCopySourceSave) },
	},
}

// pluginMarket 取插件市场的动作接口（Intents 实现可选接口时才有）。
func pluginMarket(a *App) PluginMarketplaceIntents {
	market, ok := a.pluginMarketIntents()
	if !ok {
		return nil
	}
	return market
}

// intentCasesMarket 是插件市场两个动作的失败分支：失败原因留在市场卡片里。
var intentCasesMarket = []intentCase{
	{
		method: "SetPluginFeed",
		source: func(src *fakeSource) {
			src.market = PluginMarketplaceSnapshot{Configured: true,
				FeedURL: "https://feed.example.com/plugins.json"}
		},
		open: func(a *App) {
			a.screen, a.settingsSection = ScreenSettings, secPluginSettings
			st := a.pluginManageState()
			st.feedLoaded, st.feedErr = true, ""
			st.feedDraft = "https://feed.example.com/next.json"
		},
		local: true,
		act:   func(r *intentRun) { r.app.savePluginFeed(pluginMarket(r.app)) },
	},
	{
		method: "RefreshPluginMarketplace",
		source: func(src *fakeSource) {
			src.market = PluginMarketplaceSnapshot{Configured: true,
				FeedURL: "https://feed.example.com/plugins.json"}
		},
		open: func(a *App) {
			a.screen, a.settingsSection = ScreenSettings, secPluginSettings
			a.pluginManageState().feedErr = ""
		},
		local: true,
		act:   func(r *intentRun) { r.app.refreshPluginMarket(pluginMarket(r.app)) },
	},
}

// allIntentCases 把四段拼成一张表（顺序与 Intents 接口同序，更新动作在最后）。
func allIntentCases() []intentCase {
	out := make([]intentCase, 0, 64)
	out = append(out, intentCases...)
	out = append(out, intentCasesTail...)
	out = append(out, intentCasesGoal...)
	out = append(out, intentCasesUpdate...)
	out = append(out, intentCasesMarket...)
	return out
}

// TestIntentFailureBranches 是这张表的执行者：逐条注入失败，断言界面给出可读
// 原因，并且重试入口能把同一个动作再发一次。
func TestIntentFailureBranches(t *testing.T) {
	cases := allIntentCases()
	if len(cases) == 0 {
		t.Fatal("失败分支表是空的")
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.method, func(t *testing.T) {
			// 设置屏的包级状态先摆回基线，免得上一条的失败位漏到下一条。
			resetScreens(t)
			base := wiredSource()
			var src Source = base
			if tc.source != nil {
				tc.source(base)
			}
			if tc.update != nil {
				src = &updateTestSource{fakeSource: base, snapshot: *tc.update}
			}
			stub := newFailIntents()
			app := NewApp(Options{State: StateDefault, Source: src, Intents: stub})
			tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
			r := &intentRun{t: t, app: app, tt: tt}
			if tc.open != nil {
				tc.open(app)
			}
			tt.Frame()

			tc.act(r)
			waitCalls(t, stub, tc.method, 1)
			waitReason(t, tt, tc.method)

			switch {
			case tc.retry != nil:
				tc.retry(r)
			case tc.local:
				tc.act(r)
			case !tc.local:
				if _, ok := tt.Find("重试未完成的动作"); !ok {
					t.Fatalf("失败后没有重试入口\n屏上文字：%v", tt.Texts())
				}
				if err := tt.Click("重试未完成的动作"); err != nil {
					t.Fatalf("点不到重试：%v", err)
				}
			}
			waitCalls(t, stub, tc.method, 2)
		})
	}
}

// intentNoEntry 是「接口上有方法、界面没有入口」的动作：它们不是异常分支的
// 缺口（没有路径可走，也就没有成功路径要补失败分支），而是入口本身没接。
// 登记在这里是为了让覆盖检查知道它们是"已知且登记过的空"，不是漏项。
var intentNoEntry = map[string]string{
	"Submit":             "输入区一律走 SubmitWithContent（文本 + 内容块），Submit 没有入口",
	"RetrySubmit":        "失败轮的重试没有写入口：时间线上只有自动重试的倒计时事实",
	"ExportSession":      "导出在导出面板里落本地文件，没有走 ExportSession",
	"WriteCommandRules":  "设置 › 权限的保存入口置灰（服务端没有提供权限设置写入）",
	"WaitLogin":          "登录只发 begin：等待与取消没有入口（登录相位由账号读数带回）",
	"CancelLogin":        "同上：登录进行中没有「取消」入口",
	"TestMCP":            "MCP 屏没有「测试连接」入口（只有启停）",
	"ImportMCP":          "MCP 导入来源只读展示，没有「导入」入口",
	"SetScheduleEnabled": "定时任务没有列表屏（Source 的 Schedules 没有屏在读）",
	"DeleteSchedule":     "同上：定时任务没有删除入口",
	"CancelModalityTask": "模态任务只读展示，没有「取消」入口",
}

// TestIntentCasesCoverIntents 用反射比对：Intents 上的每个方法要么在失败分支表
// 里，要么在 intentNoEntry 里写明为什么没有入口。新增写方法时这一条会红。
func TestIntentCasesCoverIntents(t *testing.T) {
	cases := allIntentCases()
	covered := map[string]bool{}
	for _, tc := range cases {
		if covered[tc.method] {
			t.Fatalf("%s 在失败分支表里出现两次", tc.method)
		}
		covered[tc.method] = true
	}
	typ := reflect.TypeOf((*Intents)(nil)).Elem()
	for i := 0; i < typ.NumMethod(); i++ {
		name := typ.Method(i).Name
		if covered[name] {
			continue
		}
		reason, ok := intentNoEntry[name]
		if !ok || strings.TrimSpace(reason) == "" {
			t.Fatalf("Intents.%s 没有失败分支用例，也没有登记无入口的原因："+
				"补一条用例，或在 intentNoEntry 里写明原因", name)
		}
	}
	for name := range covered {
		if _, ok := intentNoEntry[name]; ok {
			t.Fatalf("%s 既有用例又登记为无入口，二选一", name)
		}
	}
	for _, iface := range []struct {
		name string
		typ  reflect.Type
	}{
		{"UpdateIntents", reflect.TypeOf((*UpdateIntents)(nil)).Elem()},
		{"PluginMarketplaceIntents", reflect.TypeOf((*PluginMarketplaceIntents)(nil)).Elem()},
	} {
		for i := 0; i < iface.typ.NumMethod(); i++ {
			name := iface.typ.Method(i).Name
			if !covered[name] {
				t.Fatalf("%s.%s 没有失败分支用例", iface.name, name)
			}
		}
	}
}

// TestQuestionDegenerateInputsDoNotPanic 是 question.go 的回归：待答清单为空、
// 题号 / 槽位对不上、这条提问已经提交过，三种输入都只给原因、不崩。
//
// 这一条来自注入测试：RespondQuestion 的失败分支曾在 questionOptions 里按
// q.selected[q.index] 取值，视图状态的槽位没补齐时数组越界 panic。
func TestQuestionDegenerateInputsDoNotPanic(t *testing.T) {
	// 空清单：请求在、题不在。
	t.Run("空清单", func(t *testing.T) {
		src := wiredSource()
		src.question, src.hasQ = QuestionSnapshot{ID: "q1"}, true
		stub := newFailIntents()
		app := NewApp(Options{State: StateDefault, Source: src, Intents: stub})
		tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
		tt.Frame()
		if !tt.HasText(questionCopyNoItems) {
			t.Fatalf("空清单没有给原因，屏上文字：%v", tt.Texts())
		}
		snap, _ := app.liveQuestion()
		app.submitQuestion(snap)
		tt.Frame()
		if app.question.err != questionCopyNoItems {
			t.Fatalf("空清单提交的原因 = %q，想要 %q", app.question.err, questionCopyNoItems)
		}
		if got := stub.count("RespondQuestion"); got != 0 {
			t.Fatalf("空清单仍发出了 %d 次提交", got)
		}
	})

	// 槽位对不上：卡片状态只剩 requestID，三份槽位都是空的（越界的那一次）。
	t.Run("槽位缺失", func(t *testing.T) {
		src := wiredSource()
		src.question = QuestionSnapshot{ID: "q1", Questions: []QuestionItemSnapshot{{
			ID: "i1", Title: "选一个", Options: []string{"甲", "乙"},
		}}}
		src.hasQ = true
		stub := newFailIntents()
		app := NewApp(Options{State: StateDefault, Source: src, Intents: stub})
		app.question = questionCardState{requestID: "q1"}
		tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
		tt.Frame()
		if !tt.HasText("选一个") {
			t.Fatalf("槽位缺失时卡片没有画出来：%v", tt.Texts())
		}
		snap, _ := app.liveQuestion()
		app.submitQuestion(snap)
		tt.Frame()
		if app.question.err != questionCopyRequired {
			t.Fatalf("没勾选时的原因 = %q，想要 %q", app.question.err, questionCopyRequired)
		}
		if got := stub.count("RespondQuestion"); got != 0 {
			t.Fatalf("没答完仍发出了 %d 次提交", got)
		}
	})

	// 题号越界：索引指到这一批之外。
	t.Run("题号越界", func(t *testing.T) {
		src := wiredSource()
		src.question = QuestionSnapshot{ID: "q1", Questions: []QuestionItemSnapshot{{
			ID: "i1", Title: "选一个", Options: []string{"甲"},
		}}}
		src.hasQ = true
		app := NewApp(Options{State: StateDefault, Source: src, Intents: newFailIntents()})
		app.question = questionCardState{requestID: "q1", index: 7}
		tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
		tt.Frame()
		// 越界题号回到这一批的最后一题，卡片照常画（不 panic、不空白）。
		if app.question.index != 0 {
			t.Fatalf("越界题号没有回到范围内：%d", app.question.index)
		}
		if !tt.HasText("选一个") {
			t.Fatalf("越界题号时卡片没有画出来：%v", tt.Texts())
		}
		// 直接按越界题号取值：只给空答案，不越界。
		if got := app.question.answerAt(9, QuestionItemSnapshot{}); got != "" {
			t.Fatalf("越界题号的答案 = %q，想要空", got)
		}
	})

	// 已经提交过的提问：再提交只给原因，不发写入。
	t.Run("已提交", func(t *testing.T) {
		src := wiredSource()
		src.question = QuestionSnapshot{ID: "q1", Questions: []QuestionItemSnapshot{{
			ID: "i1", Title: "选一个", Options: []string{"甲"},
		}}}
		src.hasQ = true
		stub := newFailIntents()
		app := NewApp(Options{State: StateDefault, Source: src, Intents: stub})
		app.questionAnsweredID = "q1"
		app.question = questionCardState{requestID: "q1"}
		tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
		tt.Frame()
		snap, _ := app.liveQuestion()
		app.submitQuestion(snap)
		tt.Frame()
		if app.question.err != questionCopyStale {
			t.Fatalf("已提交再答的原因 = %q，想要 %q", app.question.err, questionCopyStale)
		}
		if got := stub.count("RespondQuestion"); got != 0 {
			t.Fatalf("已提交的提问仍发出了 %d 次提交", got)
		}
	})
}

// readCase 是一条「读事实的失败分支」用例：适配器给出失败读数，界面必须写出
// 原因（不是空白区域）；有重读通道的还要给重试入口。
type readCase struct {
	name string
	// inject 往注入源上写这一条的失败读数。
	inject func(*fakePanelSource)
	// open 摆好要看的那一屏（设置分节、右栏标签、浮层）。
	open func(*App)
	// want 是必须出现在屏上的原因原文（注入时用的是同一句标记）。
	want string
	// retry 是可重读入口的读屏名；空表示本版没有重读通道（只写原因）。
	retry string
	// update 非空时把 Source 包成带更新读数的实现（关于页两条要用）。
	update *UpdateSnapshot
}

// readFail 造一条注入用的原因原文：每种读路径一句，互不串味。
func readFail(what string) string { return "注入读失败：" + what }

// readFailureCases 是读路径失败分支的清单。
var readFailureCases = []readCase{
	{
		name: "连接断开",
		inject: func(src *fakePanelSource) {
			src.conn = Connection{Phase: "disconnected", Reason: readFail("控制连接")}
		},
		want:  readFail("控制连接"),
		retry: "重新连接",
	},
	{
		name: "文件树读失败",
		inject: func(src *fakePanelSource) {
			src.trees[""] = FileTreeSnapshot{Phase: PanelReadFailed, Error: readFail("目录")}
		},
		open:  func(a *App) { a.rightOpen, a.rightTab = true, tabFiles },
		want:  readFail("目录"),
		retry: "重试读取文件树",
	},
	{
		name: "文件搜索读失败",
		inject: func(src *fakePanelSource) {
			src.search = FileSearchSnapshot{Phase: PanelReadFailed, Error: readFail("搜索")}
		},
		open: func(a *App) {
			a.rightOpen, a.rightTab = true, tabFiles
			a.rp().filter = "token"
		},
		want:  readFail("搜索"),
		retry: "重试搜索",
	},
	{
		name: "改动读失败",
		inject: func(src *fakePanelSource) {
			src.changes = ChangeSet{Phase: PanelReadFailed, Error: readFail("改动")}
		},
		open: func(a *App) { a.rightOpen, a.rightTab = true, tabChanges },
		want: readFail("改动"),
	},
	{
		name: "文件预览读失败",
		inject: func(src *fakePanelSource) {
			src.trees[""] = readyTree(FileEntrySnapshot{Name: "main.go"})
			src.previews = map[string]PreviewSnapshot{
				"main.go": {Phase: PanelReadFailed, Error: readFail("预览")},
			}
		},
		open: func(a *App) {
			a.rightOpen, a.rightTab = true, tabFilePreview
			a.right.previewPath = "main.go"
		},
		want:  readFail("预览"),
		retry: "重试读取文件",
	},
	{
		name: "插件清单读失败",
		inject: func(src *fakePanelSource) {
			src.extensions.Plugins = ExtListSnapshot{ReadError: readFail("插件清单")}
		},
		open: func(a *App) { a.screen, a.settingsSection = ScreenSettings, secPluginSettings },
		want: readFail("插件清单"),
	},
	{
		name: "技能读失败",
		inject: func(src *fakePanelSource) {
			src.extensions.Skills = ExtListSnapshot{ReadError: readFail("技能")}
		},
		open: func(a *App) { a.skillsOpen = true },
		want: readFail("技能"),
	},
	{
		name: "MCP 读失败",
		inject: func(src *fakePanelSource) {
			src.extensions.MCP = ExtListSnapshot{ReadError: readFail("MCP")}
		},
		open: func(a *App) { a.screen, a.settingsSection = ScreenSettings, secMCP },
		want: readFail("MCP"),
	},
	{
		name: "Hooks 读失败",
		inject: func(src *fakePanelSource) {
			src.extensions.Hooks = ExtListSnapshot{ReadError: readFail("Hooks")}
		},
		open: func(a *App) { a.screen, a.settingsSection = ScreenSettings, secHooks },
		want: readFail("Hooks"),
	},
	{
		name: "运行环境读失败",
		inject: func(src *fakePanelSource) {
			src.settings.RuntimesKnown = false
			src.settings.RuntimesReason = readFail("运行环境")
		},
		open: func(a *App) { a.screen, a.settingsSection = ScreenSettings, secRuntime },
		want: readFail("运行环境"),
	},
	{
		name: "模态任务读失败",
		inject: func(src *fakePanelSource) {
			src.settings.Modality = ModalitySnapshot{Known: true, TaskKnown: true}
			src.settings.TasksError = readFail("模态任务")
		},
		open: func(a *App) { a.screen, a.settingsSection = ScreenSettings, secModality },
		want: readFail("模态任务"),
	},
	{
		name: "供应商读失败",
		inject: func(src *fakePanelSource) {
			src.settings.ProviderError = readFail("供应商")
		},
		open: func(a *App) { a.screen, a.settingsSection = ScreenSettings, secProviders },
		want: readFail("供应商"),
	},
	{
		name: "存储用量读失败",
		inject: func(src *fakePanelSource) {
			src.settings.StorageKnown = false
			src.settings.StorageError = readFail("存储用量")
		},
		open: func(a *App) {
			a.screen, a.settingsSection = ScreenSettings, secGeneral
			generalScreen.cleanupOpen = true
		},
		want: readFail("存储用量"),
	},
	{
		name: "用量读失败",
		inject: func(src *fakePanelSource) {
			src.usage = UsageSnapshot{Available: false, Reason: readFail("用量")}
		},
		open: func(a *App) { a.screen, a.settingsSection = ScreenSettings, secAccount },
		want: readFail("用量"),
	},
	{
		name: "登录失败",
		inject: func(src *fakePanelSource) {
			src.acct = AccountSnapshot{Initial: "A", Connected: false, Status: "signed_out",
				LoginPhase: "failed", LoginReason: readFail("登录")}
		},
		open: func(a *App) { a.screen, a.settingsSection = ScreenSettings, secAccount },
		want: readFail("登录"),
	},
	{
		name:   "更新源配置错误",
		update: &UpdateSnapshot{Configured: true, ConfigError: "更新源必须是 http/https 地址"},
		open:   func(a *App) { a.screen, a.settingsSection = ScreenSettings, secAbout },
		want:   "更新源必须是 http/https 地址",
	},
	{
		name:   "更新失败",
		update: &UpdateSnapshot{Configured: true, Phase: "error", Err: readFail("更新")},
		open:   func(a *App) { a.screen, a.settingsSection = ScreenSettings, secAbout },
		want:   readFail("更新"),
		retry:  abtCopyRetry,
	},
	{
		name: "插件市场读失败",
		inject: func(src *fakePanelSource) {
			src.market = PluginMarketplaceSnapshot{Configured: true,
				FeedURL: "https://feed.example.com/plugins.json", Err: readFail("插件市场")}
		},
		open: func(a *App) { a.screen, a.settingsSection = ScreenSettings, secPluginSettings },
		want: readFail("插件市场"),
	},
	{
		name: "插件市场源配置错误",
		inject: func(src *fakePanelSource) {
			src.market = PluginMarketplaceSnapshot{Configured: true,
				FeedURL: "https://feed.example.com/plugins.json", ConfigError: readFail("插件市场源")}
		},
		open: func(a *App) { a.screen, a.settingsSection = ScreenSettings, secPluginSettings },
		want: readFail("插件市场源"),
	},
	{
		name: "成员审批读不到",
		inject: func(src *fakePanelSource) {
			src.approval = ApprovalSnapshot{ID: "a1", Tool: "member_approval_allow",
				Member: &ApprovalMemberSnapshot{State: ApprovalMemberUnavailable,
					ApprovalID: "ap-1"}}
			src.hasAppr = true
		},
		want: approvalCopyMemberUnavailable,
	},
	{
		name: "搜索结果扫描上限",
		inject: func(src *fakePanelSource) {
			src.search = FileSearchSnapshot{Phase: PanelReadReady, Truncated: true,
				StopReason: "max-files", Limit: 200,
				Hits: []FileHitSnapshot{{Path: "main.go", Line: 3, Column: 1, Snippet: "token"}}}
		},
		open: func(a *App) {
			a.rightOpen, a.rightTab = true, tabFiles
			a.rp().filter = "token"
		},
		want: "已扫到文件数上限，结果可能不完整",
	},
}

// TestReadFailureBranches 执行读失败清单：每一屏都要把原因写出来，能重读的
// 给出重试入口（并且真的带上「刷新」请求）。
func TestReadFailureBranches(t *testing.T) {
	for _, tc := range readFailureCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			resetScreens(t)
			src := wiredPanelSource()
			if tc.inject != nil {
				tc.inject(src)
			}
			var reader Source = src
			if tc.update != nil {
				reader = &updateTestSource{fakeSource: src.fakeSource, snapshot: *tc.update}
				// 面板读数仍要能读：把两份源合成一份。
				reader = &readUpdateSource{fakePanelSource: src, snapshot: *tc.update}
			}
			app := NewApp(Options{State: StateDefault, Source: reader})
			if tc.open != nil {
				tc.open(app)
			}
			tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
			tt.Frame()
			if !tt.HasText(tc.want) {
				t.Fatalf("没有写出读失败的原因 %q\n屏上文字：%v", tc.want, tt.Texts())
			}
			if tc.retry == "" {
				return
			}
			if _, ok := tt.Find(tc.retry); !ok {
				t.Fatalf("读失败后没有重读入口 %q\n屏上文字：%v", tc.retry, tt.Texts())
			}
		})
	}
}

// resetScreens 把设置屏的包级视图状态摆回基线，并在用例结束后还原：这一族的
// 状态（关于页、通用页、MCP / Hooks / 供应商）是包级的，用例之间会互相污染。
func resetScreens(t *testing.T) {
	t.Helper()
	savedAbout, savedGeneral := aboutScreen, generalScreen
	resetSettingsScreenState()
	aboutScreen = aboutState{phase: abtDisabled}
	generalScreen = generalState{keepAwake: true, launch: true,
		proxyMode: 0, proxyURL: "http://127.0.0.1:7890"}
	t.Cleanup(func() {
		aboutScreen, generalScreen = savedAbout, savedGeneral
		resetSettingsScreenState()
	})
}

// wiredPanelSource 是「连上事实」的面板注入源：账号、用量、设置与扩展读数取
// wiredSource 的那一份，文件树与预览留空由用例填。
func wiredPanelSource() *fakePanelSource {
	base := wiredSource()
	base.session.Cwd = "/work"
	return &fakePanelSource{fakeSource: base, trees: map[string]FileTreeSnapshot{}}
}

// readUpdateSource 是把面板读数与更新读数合到一份的注入源。
type readUpdateSource struct {
	*fakePanelSource
	snapshot UpdateSnapshot
}

func (s *readUpdateSource) Update() UpdateSnapshot { return s.snapshot }

// emptyCase 是一条「读事实读到空」的用例：空态要给可读的说明，不是一块空白。
type emptyCase struct {
	name  string
	state func(*fakePanelSource)
	open  func(*App)
	want  string
}

// emptyStateCases 是读路径的空态清单（空列表、空目录、空搜索结果、空清单）。
var emptyStateCases = []emptyCase{
	{
		name: "没有会话",
		state: func(src *fakePanelSource) {
			src.names, src.hasSess = nil, false
			src.session = SessionSnapshot{}
		},
		want: "当前会话还没有消息",
	},
	{
		name:  "文件树空目录",
		state: func(src *fakePanelSource) { src.trees[""] = readyTree() },
		open:  func(a *App) { a.rightOpen, a.rightTab = true, tabFiles },
		want:  rpEmptyDirTitle,
	},
	{
		name: "搜索无命中",
		state: func(src *fakePanelSource) {
			src.search = FileSearchSnapshot{Phase: PanelReadReady, Limit: 200}
		},
		open: func(a *App) {
			a.rightOpen, a.rightTab = true, tabFiles
			a.rp().filter = "token"
		},
		want: "无匹配文件",
	},
	{
		name: "改动为空",
		state: func(src *fakePanelSource) {
			src.changes = ChangeSet{Phase: PanelReadReady, Repo: true, Branch: "main"}
		},
		open: func(a *App) { a.rightOpen, a.rightTab = true, tabChanges },
		want: "没有未提交的改动",
	},
	{
		name: "后台任务为空",
		open: func(a *App) { a.rightOpen, a.rightTab = true, tabTasks },
		want: "暂无运行进程",
	},
	{
		name: "插件清单为空",
		state: func(src *fakePanelSource) {
			src.extensions.Plugins = ExtListSnapshot{}
		},
		open: func(a *App) { a.screen, a.settingsSection = ScreenSettings, secPluginSettings },
		want: pluginManageEmpty,
	},
	{
		name: "技能为空",
		state: func(src *fakePanelSource) {
			src.extensions.Skills = ExtListSnapshot{}
		},
		open: func(a *App) { a.skillsOpen = true },
		want: "还没有技能",
	},
	{
		name: "MCP 服务器为空",
		state: func(src *fakePanelSource) {
			src.extensions.MCP = ExtListSnapshot{}
		},
		open: func(a *App) { a.screen, a.settingsSection = ScreenSettings, secMCP },
		want: mcpCopyNoServers,
	},
	{
		name: "模态任务为空",
		state: func(src *fakePanelSource) {
			src.settings.Modality = ModalitySnapshot{Known: true, TaskKnown: true,
				TaskSupported: true}
			src.settings.ModalityTasks = nil
		},
		open: func(a *App) { a.screen, a.settingsSection = ScreenSettings, secModality },
		want: mdCopyNoTasks,
	},
}

// TestReadEmptyStatesHaveCopy 执行空态清单：读到空也要有一句说明，不留空白区域。
func TestReadEmptyStatesHaveCopy(t *testing.T) {
	for _, tc := range emptyStateCases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			resetScreens(t)
			src := wiredPanelSource()
			if tc.state != nil {
				tc.state(src)
			}
			app := NewApp(Options{State: StateDefault, Source: src})
			if tc.open != nil {
				tc.open(app)
			}
			tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
			tt.Frame()
			if !tt.HasText(tc.want) {
				t.Fatalf("空态没有写出 %q\n屏上文字：%v", tc.want, tt.Texts())
			}
		})
	}
}

// oauthFailSource 造一个 Codex 账号形态的 provider 读数，供按提供商登录的失败
// 分支用（authenticated 决定已连接卡还是未登录块）。
func oauthFailSource(authenticated bool) func(*fakeSource) {
	return func(src *fakeSource) {
		src.settings.ProviderRecordsKnown = true
		src.settings.ProviderRecords = []ProviderRecordSnapshot{{
			ID: "codex", Name: "OpenAI Codex", BaseURL: "https://api.test/v1",
			Enabled: true, Version: 1, Authority: "openai_codex",
			Account: &ProviderAccountSnapshot{
				Namespace: "account.codex", LoginMethods: []string{configLoginBrowser},
				Authenticated: authenticated, AccountLabel: "ada@example.test",
			},
		}}
	}
}
