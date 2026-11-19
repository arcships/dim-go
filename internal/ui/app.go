package ui

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	gui "github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"

	"github.com/arcships/dimagent-mygo/internal/dirpick"
	"github.com/arcships/dimagent-mygo/internal/shortcut"
)

// State 是首片骨架要看的五种界面状态，对应 docs/domains/U1-ia.md §5
// 的可见性矩阵。它不是业务状态：业务事实由 App Server 提供，这里只切样例。
type State int

const (
	// StateDefault 默认：三轮历史，过程摘要行折叠，输入位可用。
	StateDefault State = iota
	// StateRunning 运行中：回流行在生成、摘要行进形态、输入位上沿出现队列。
	StateRunning
	// StateWaiting 等待你同意：审批卡占据输入位，输入区不存在。
	StateWaiting
	// StateFailed 失败 / 结果不确定：本轮下方一行说明，摘要行转失败形态。
	StateFailed
	// StateEmpty 空态：没有消息，输入位仍可用。
	StateEmpty
)

var stateLabels = [...]string{
	StateDefault: "默认",
	StateRunning: "运行中",
	StateWaiting: "等待你同意",
	StateFailed:  "失败 / 结果不确定",
	StateEmpty:   "空态",
}

var stateArgs = [...]string{
	StateDefault: "default",
	StateRunning: "running",
	StateWaiting: "waiting",
	StateFailed:  "failed",
	StateEmpty:   "empty",
}

func (s State) String() string {
	if int(s) < 0 || int(s) >= len(stateLabels) {
		return "未知状态"
	}
	return stateLabels[s]
}

// ParseState 解析 -state 的取值：名称或 1–5。
func ParseState(v string) (State, error) {
	v = strings.ToLower(strings.TrimSpace(v))
	if v == "" {
		return StateDefault, nil
	}
	for i, name := range stateArgs {
		if v == name {
			return State(i), nil
		}
		if n, err := strconv.Atoi(v); err == nil && n == i+1 {
			return State(i), nil
		}
	}
	return StateDefault, fmt.Errorf("未知状态 %q：可用 default、running、waiting、failed、empty，或 1–5", v)
}

// 设置的分节标识（19 节，顺序与名称见 settings_sections.go）。

// Screen 是当前屏：主界面或设置（P1/P2，U1-ia §4）。设置独立成屏，返回时恢复
// 离开前的主界面状态（草稿、滚动位置、展开状态都由 App 持有，切屏不重建它们）。
type Screen int

const (
	// ScreenMain 主界面。
	ScreenMain Screen = iota
	// ScreenSettings 设置屏。
	ScreenSettings
)

// Variant 是骨架的第三种维度：在同一个状态上打开分支菜单、目标编辑或后台任务，
// 用来走查三类新信息（不改变业务事实，只切样例里的展开态）。
type Variant int

const (
	// VariantNone 不额外展开。
	VariantNone Variant = iota
	// VariantBranch 打开标题行的分支菜单。
	VariantBranch
	// VariantGoal 让输入区上沿的目标行进入编辑态。
	VariantGoal
	// VariantTasks 打开右栏的「后台」标签并显示运行中的任务。
	VariantTasks
	// VariantBranchEnd 打开分支菜单并停在「结束分支」的确认态。
	VariantBranchEnd
	// VariantRich 只显示那条富文本回复（Markdown 块级样式走查）。
	VariantRich
	// VariantTermDisconnected / TermRecovered 是终端的断线与恢复（F31 的退化）。
	VariantTermDisconnected
	VariantTermRecovered
	// VariantShell / ShellRun 是输入区命令模式（`!`）的空闲与运行中。
	VariantShell
	VariantShellRun
	// VariantSessionMenu / Rename / SessionDelete / Search 是左栏目录动作的四个走查态。
	VariantSessionMenu
	VariantSessionRename
	VariantSessionDelete
	VariantSearch
	// VariantTerm / TermRunning / TermExited 打开右栏的终端标签，三种状态。
	VariantTerm
	VariantTermRunning
	VariantTermExited
	// VariantSettings 进入设置屏（首节：账户）；VariantSettingsProviders 进模型
	// 供应节并打开「添加自定义提供商」表单（表单走查与截图用）；
	// VariantSettingsKeybindings 进快捷键节（全局唤起那一项要能截图）。
	VariantSettings
	VariantSettingsProviders
	VariantSettingsKeybindings
	VariantSettingsNotifications
	// 输入区 / 标题栏的浮层走查态：⋯ 菜单、打开方式、模型、推理强度、技能、
	// 模式、权限、工作区。
	VariantChatMenu
	VariantOpenWith
	VariantModel
	VariantEffort
	VariantSkills
	VariantMode
	VariantPermission
	VariantWorkspace
	// 右栏的五个走查态：文件页、改动页、范围下拉、＋新建菜单、展开面板。
	VariantRightPanel
	VariantReview
	VariantReviewScope
	VariantAddTab
	VariantRightExpanded
	// VariantPalette / PaletteEmpty 打开命令面板，后者停在无匹配项的空态。
	VariantPalette
	VariantPaletteEmpty
	// VariantTimelineHover 把第一轮用户消息的悬停动作钉在屏上（截图没有指针）；
	// VariantTimelineToolDetail 展开第一轮工具分组里的失败工具，走查三段详情。
	VariantTimelineHover
	VariantTimelineToolDetail
)

// variantNames 是 -variant 的取值。
var variantNames = map[string]Variant{
	"": VariantNone, "none": VariantNone, "branch": VariantBranch,
	"goal": VariantGoal, "tasks": VariantTasks,
	"branch-end": VariantBranchEnd, "rich": VariantRich,
	"term-disconnected": VariantTermDisconnected, "term-recovered": VariantTermRecovered,
	"shell": VariantShell, "shell-run": VariantShellRun,
	"session-menu": VariantSessionMenu, "rename": VariantSessionRename,
	"session-delete": VariantSessionDelete, "search": VariantSearch,
	"term": VariantTerm, "term-running": VariantTermRunning,
	"term-exited": VariantTermExited, "settings": VariantSettings,
	"settings-providers":   VariantSettingsProviders,
	"settings-keybindings": VariantSettingsKeybindings,
	"settings-notifications": VariantSettingsNotifications,
	"chat-menu":            VariantChatMenu, "open-with": VariantOpenWith,
	"model": VariantModel, "effort": VariantEffort,
	"skills": VariantSkills, "mode": VariantMode,
	"permission": VariantPermission, "workspace": VariantWorkspace,
	"right-panel": VariantRightPanel, "review": VariantReview,
	"review-scope": VariantReviewScope, "add-tab": VariantAddTab,
	"right-expanded":  VariantRightExpanded,
	"command-palette": VariantPalette, "command-palette-empty": VariantPaletteEmpty,
	"timeline-hover": VariantTimelineHover, "timeline-tool-detail": VariantTimelineToolDetail,
}

// ParseVariant 解析 -variant。
func ParseVariant(v string) (Variant, error) {
	if got, ok := variantNames[v]; ok {
		return got, nil
	}
	return VariantNone, fmt.Errorf("未知变体 %q：可用 none、term-disconnected、term-recovered、shell、shell-run、session-menu、rename、session-delete、search、branch、branch-end、rich、goal、tasks、settings、settings-providers、settings-keybindings、settings-notifications、term、term-running、term-exited、chat-menu、open-with、model、effort、skills、mode、permission、workspace、right-panel、review、review-scope、add-tab、right-expanded、command-palette、command-palette-empty、timeline-hover、timeline-tool-detail", v)
}

// Options 是骨架的启动参数。
type Options struct {
	// State 是起始状态。
	State State
	// Variant 是额外的走查态（分支菜单 / 目标编辑 / 后台任务）。
	Variant Variant
	// Source 非空时视图从它读事实（-connect）；为空时用样例（-demo / 截图）。
	Source Source
	// Intents 非空时视图上的动作会发起真实写入；为空时动作只留在视图里
	// （样例走查）。
	Intents Intents
	// LogPath 是客户端启动日志的路径（`<数据目录>/logs/dimagent.log`），由入口
	// 在启动时给出。为空表示这次运行没有日志：断线卡的「查看日志」会写明这一点，
	// 不自己猜数据目录（视图不推导入口的文件位置）。
	LogPath string
	// Changed 在后台意图完成后回调，供入口重绘；可为 nil。
	Changed func()
	// StateChanged 在调试键切换状态后回调，供入口更新窗口标题。
	StateChanged func(State)
	// OnFrame 在每帧视图构建后调用（渲染线程），供宿主外壳（托盘与系统通知）
	// 读取当前会话状态并刷新菜单；它只读快照，不改视图。可为 nil。
	OnFrame func(HostSnapshot)
}

// HostSnapshot 是宿主外壳每一帧要读的会话事实：托盘菜单、图标状态与系统通知
// 都由它决定。它不是第二份事实：字段每次从 Source 快照现读。
type HostSnapshot struct {
	// SessionID 是当前打开的会话；空表示还没有打开。
	SessionID    string
	SessionTitle string
	// Running 表示当前有一轮在跑。
	Running bool
	// Pending 表示有审批或提问待决；PendingID 是待决项的标识（通知去重用）。
	Pending   bool
	PendingID string
	// LastTurn 是最后一个已结束的轮。
	LastTurnEnded   bool
	LastTurnIndex   int
	LastTurnOutcome string
	// Recent 是按最近活动排序的会话（最多 5 条），给托盘「最近会话」用。
	Recent []HostSession
}

// HostSession 是托盘里的一条会话。
type HostSession struct {
	ID    string
	Title string
}

// App 只保存可丢的界面状态：当前样例状态、左栏与右栏的开合、摘要展开、
// 草稿文本与滚动位置、上下文用量的展开，外加几个只用于样例的展示字段
// （非默认模型标记、审批命令原文、用量样例）。它不保存任何事实，也不发起
// 任何写入。
type App struct {
	state   State
	variant Variant
	onState func(State)

	sidebarCollapsed    bool
	sidebarWidth        float32
	lastPrefs           PreferencesSnapshot
	rightOpen           bool
	rightTab            int
	selectedSession     string
	lastSelectedSession string

	draft  string
	rounds []round
	// summaryOpen 是每一轮的展开态（可丢的视图状态）：按轮次索引存一份 *bool，
	// 指针稳定、按需创建——轮次变多时不必重建数组，view 里也不攒第二份事实。
	summaryOpen map[int]*bool
	// blockOpen 是思考块、工具行、产物与改动块的开合（键是块标识）。
	blockOpen map[string]*bool
	// timelineHover 是走查变体钉住的悬停轮次（-1 表示没有）：自动截图没有指针，
	// 只影响用户消息的悬停动作是否显形，不改任何事实。
	timelineHover int

	// connected 与 accountInitial 是样例里的连接态与账号首字母：只用于画状态点
	// 与头像（UI-23），不是事实来源。
	connected      bool
	accountInitial string

	// md 是 Markdown 视图的配置（主题与扩展），一份实例跨帧复用（解析缓存随它）；
	// 主题每帧在 applyTokens 里换成当前外观（U0 §3.1 的 M 系列）。
	md markdown.Options

	// 分支、目标、后台任务是样例里的展示数据（不是事实）：分支名、目标文本、
	// 任务名都属于内容字符串（U1-content.json）。
	branch           string
	branches         []string
	branchMenuOpen   bool
	endBranchConfirm bool
	goal             string
	goalDone         bool
	goalEditing      bool
	goalDraft        string
	goalClearConfirm bool

	// 设置屏：屏幕、当前分节（19 节见 settings_sections.go）、外观偏好与时区。
	// 全部是可丢的界面状态。
	screen          Screen
	settingsSection settingsSectionID
	themePref       string
	motionOff       bool
	timeZone        string
	// 样例模式下的全局唤起快捷键（接上 Source 时读注册状态，这两项不参与）。
	globalShortcutOn    bool
	globalShortcutAccel string
	// focusComposer 请求下一帧把焦点放到输入区（全局快捷键唤起时由入口调用）。
	focusComposer atomic.Bool

	// 设置 · 账户（重建屏）：邮箱登录对话框的开合与草稿、重置确认框的类型、
	// 用量刷新的起点。都是可丢的界面状态，不是事实。
	accountEmailOpen  bool
	accountEmailDraft string
	accountResetType  string
	accountRefreshAt  time.Time

	// source 非空时，视图从它读事实（A 领域的 Source 接口）；为空时用样例数据
	// （-demo 与截图）。视图不缓存事实：每次读的都是 Source 的快照。
	source Source
	// onFrame 在每帧构建后收到宿主外壳要的会话快照（托盘与通知），可为 nil。
	onFrame func(HostSnapshot)
	// 设置 › 通知的三个开关（本地偏好，接上 Source 时读写事实）。
	notifyTaskDone      bool
	notifySound         bool
	notifyNeedsResponse bool

	// 目录选择（左栏「添加工作区」与设置 › Hooks 的项目路径）：两处入口各留一份
	// 可丢的界面状态，后台结果经 pickResults 交回渲染线程（见 dirpick.go）。
	addWorkspacePick dirPickState
	hooksPick        dirPickState
	// 本端登记的附加工作区与 Hooks 项目目录：随本地偏好落盘，不是服务端授权的
	// 项目根。
	workspaces       []string
	hooksProjectPath string
	// dirPickRunner / dirPickGOOS 让测试注入选择器与平台；nil / 空表示真实环境。
	dirPickRunner dirpick.Runner
	dirPickGOOS   string
	pickMu        sync.Mutex
	pickResults   []dirPickOutcome

	// intents 非空时，视图上的动作发起真实写入；写入在后台协程执行，结果回到
	// intentMu 保护的两个槽位，下一帧在渲染线程上应用。
	intents    Intents
	changed    func()
	intentMu   sync.Mutex
	intentErr  string
	intentDone []func()
	// intentFail 是后台意图的失败回调（例如表单要就地显示失败原因）：与
	// intentDone 一样只在渲染线程上执行，不在后台协程里改视图。
	intentFail []func()
	// questionAnswer 是提问卡的回答草稿（可丢的视图状态）。
	questionAnswer string
	// logPath 是入口交来的客户端日志路径（Options.LogPath）；logNote 是「查看日志」
	// 打不开时留在断线卡里的路径或原因。两者都是可丢的视图状态。
	logPath string
	logNote string
	// lastShellRun 是最近一次已发出的命令运行标识，供「中止」使用。
	lastShellRun string

	// 待发附件（样例）。
	attachments       []attachmentChipData
	attachmentEditing bool
	attachmentKind    string
	attachmentName    string
	attachmentText    string
	attachmentMedia   string
	attachmentData    string

	// 会话目录：搜索面、行内重命名、删除确认、归档与导出反馈（都属可丢的视图状态）。
	searchOpen  bool
	searchQuery string
	renaming    int
	renameDraft string
	deleting    int
	archived    map[int]bool
	renamed     map[int]string
	exported    string
	menuOpen    map[int]*bool

	// 左栏重建后的可丢视图状态：工作区分组折叠、置顶、视图菜单与账户菜单
	// （见 sidebar_state.go）。它不是事实，切会话不重建。
	sidebarView sidebarViewState

	// 终端（右栏第五个标签）：状态、输出行、提示行与"已复制"反馈。
	termState  termState
	termLines  []termLine
	termInput  string
	termCopied bool
	tasks      []task

	// panels 是右栏四个重建面板（后台任务 / 终端 / 侧聊 / 子代理）的可丢视图
	// 状态：选中的行、草稿、确认框与面板内的加载态。它不是事实。
	panels panelView

	// right 是右栏外壳与文件 / 改动两页的可丢视图状态（标签页、筛选词、
	// 目录展开、浮层与拖出来的宽度），见 rightpanel_parts.go。
	right rightPanelState

	// contextUsed / contextLimit 是样例里的上下文用量，百分比由二者算；
	// contextOpen 是明细的展开状态。三者都是可丢的界面状态（UI-8）。
	contextUsed  int
	contextLimit int
	contextOpen  bool

	// 重建后的输入区 / 标题栏视图状态：推理强度、模式、权限、工作区、选择器
	// 浮层的开合与查询词。全部可丢，不承载事实。
	effort         float64
	effortOpen     bool
	modelOpen      bool
	modelQuery     string
	skillsOpen     bool
	skillQuery     string
	mode           string
	modeOpen       bool
	workspaceName  string
	workspaceOpen  bool
	workspaceQuery string
	// workspacePath 是空态工作区选择器选中的目录（可丢）：新建会话把它当 cwd。
	workspacePath string
	permission    string
	// permissionChoice 是权限菜单里点过的那一档文案（可丢）：协议档位在「默认
	// 权限」「智能权限」上取值相同（见 permissionLevels），记住点的是哪一档才不
	// 至于回弹。
	permissionChoice  string
	permissionOpen    bool
	permissionArmOpen bool
	permissionArmAck  bool
	statusBranchOpen  bool
	statusBranchQuery string
	chatMenuOpen      bool
	openWithOpen      bool
	openWithLoading   bool
	openWithApp       string
	chatPinned        bool
	headerNote        string

	// model 是当前模型名（输入区工具行与标题栏共用）。
	model string
	// approvalCommand 是审批卡上的命令原文；为空时拿不到原文，不提供「一直允许」。
	approvalCommand string
	// confirmAlways 是「一直允许」的确认态：第一次点击先换成确认，再点才生效。
	confirmAlways bool
	// approvalDecision 是样例里点过的决定（只用于界面反馈，不发起任何写入）。
	approvalDecision string

	feed gui.ScrollState

	// palette 是命令面板的视图状态（查询、选中、历史与最近使用）。
	palette commandPaletteState
}

// NewApp 建立骨架。状态切换只改界面样例，不连任何服务。
func NewApp(o Options) *App {
	a := &App{
		source:          o.Source,
		intents:         o.Intents,
		changed:         o.Changed,
		onFrame:         o.OnFrame,
		state:           o.State,
		variant:         o.Variant,
		onState:         o.StateChanged,
		selectedSession: "s1",
		connected:       true,
		// 样例账号是「eric」：首字母与设置 › 账户卡一致（ER），不另写一个 E。
		accountInitial: "ER",
		logPath:        o.LogPath,
	}
	// 时间线的助手区自己承担可选区与右键菜单（消息级复制）：Markdown 不再自建
	// 选区与菜单，把段落交给外层容器（TL-65）。
	a.md.ShareSelection = true
	a.loadSample()
	return a
}

// SetChanged 设置后台意图完成后的重绘回调。
func (a *App) SetChanged(changed func()) { a.changed = changed }

// State 返回当前样例状态。
func (a *App) State() State { return a.state }

// SetState 切换样例状态。
func (a *App) SetState(s State) {
	if a.state == s {
		return
	}
	a.state = s
	a.loadSample()
	if a.onState != nil {
		a.onState(s)
	}
}

// loadSample 按状态重建静态样例。
func (a *App) loadSample() {
	// 连着真实事实时不灌样例（-connect 下任何样例都不得进视图），只恢复可丢的
	// 视图状态默认值；事实全部每帧从 Source 读。
	if a.source != nil {
		a.resetWiredViewState()
		return
	}
	a.rounds = sampleRounds(a.state)
	a.summaryOpen = make(map[int]*bool)
	a.blockOpen = make(map[string]*bool)
	a.timelineHover = -1
	// 运行中的最新一轮把工具分组默认展开：参考产品在输出时就是展开态。
	if a.state == StateRunning && len(a.rounds) >= 2 {
		*a.openOf(len(a.rounds) - 2) = true
	}
	a.approvalCommand = ""
	// 上下文用量的样例值：12%，与线框一致；上限只是样例，不是事实。
	a.contextUsed, a.contextLimit = 24_600, 200_000
	a.contextOpen = false
	a.confirmAlways = false
	a.approvalDecision = ""
	// 设置 › 通知的三个开关：样例与参考产品的出厂状态一致（全开）。
	a.notifyTaskDone, a.notifySound, a.notifyNeedsResponse = true, true, true
	a.lastPrefs = PreferencesSnapshot{}

	// 分支、目标、后台任务：默认取"常规"值（分支 main 不上屏，UI-9）。
	a.branch = "main"
	a.branches = []string{"main"}
	a.branchMenuOpen = false
	a.endBranchConfirm = false
	a.goal = "把发送路径改成排队"
	a.goalDone = false
	a.goalEditing = false
	a.tasks = taskSamples()

	a.screen = ScreenMain
	a.settingsSection = secAccount
	a.themePref = "跟随系统"
	a.motionOff = false
	a.timeZone = "跟随系统"
	// 全局唤起快捷键的样例值（接上 Source 时改读注册状态）。
	a.globalShortcutOn = true
	a.globalShortcutAccel = shortcut.DefaultAccelerator
	globalShortcutEdit = globalShortcutEditState{}
	// 目录选择的两处界面状态与本地登记的目录：样例里从零开始。
	a.addWorkspacePick = dirPickState{}
	a.hooksPick = dirPickState{}
	a.workspaces = nil
	a.hooksProjectPath = ""
	a.attachments = nil
	a.effort = 2
	a.effortOpen = false
	a.modelOpen = false
	a.modelQuery = ""
	a.skillsOpen = false
	a.skillQuery = ""
	a.mode = "Agent"
	a.modeOpen = false
	a.workspaceName = sampleWorkspaceTitle
	a.workspaceOpen = false
	a.workspaceQuery = ""
	a.workspacePath = ""
	a.permission = "默认权限"
	a.permissionChoice = ""
	a.permissionOpen = false
	a.permissionArmOpen = false
	a.permissionArmAck = false
	a.statusBranchOpen = false
	a.statusBranchQuery = ""
	a.chatMenuOpen = false
	a.openWithOpen = false
	a.openWithLoading = false
	a.openWithApp = "VS Code"
	a.chatPinned = false
	a.headerNote = ""
	a.model = sampleModel
	a.searchOpen, a.searchQuery = false, ""
	a.renaming, a.deleting = -1, -1
	a.renameDraft = ""
	a.archived, a.renamed = map[int]bool{}, map[int]string{}
	a.menuOpen = map[int]*bool{}
	a.exported = ""
	a.termState = termIdle
	a.termLines = termSamples(termIdle)
	a.termCopied = false
	a.panels = newPanelView()

	switch a.variant {
	case VariantSettings:
		a.screen = ScreenSettings
		a.settingsSection = secAccount // 设置：首节（账户）
	case VariantSettingsProviders:
		// 设置 › 模型供应 + 打开「添加自定义提供商」表单（走查与截图用）。
		a.screen = ScreenSettings
		a.settingsSection = secProviders
		configScreen.form = configFormNew()
	case VariantSettingsKeybindings:
		// 设置 › 快捷键（全局唤起那一项要能单独截图）。
		a.screen = ScreenSettings
		a.settingsSection = secKeybindings
	case VariantSettingsNotifications:
		// 设置 › 通知（三个开关要能单独截图）。
		a.screen = ScreenSettings
		a.settingsSection = secNotifications
	case VariantTermDisconnected:
		a.rightOpen, a.rightTab = true, 4
		a.termState = termDisconnected
		a.termLines = append(termSamples(termExited), termLine{text: "connection lost", color: 1})
	case VariantTermRecovered:
		a.rightOpen, a.rightTab = true, 4
		a.termState = termIdle
		a.termLines = append(termSamples(termExited), termLine{text: "connection lost", color: 1},
			termLine{text: "reconnected", color: 2})
	case VariantShell:
		a.draft = "!go test ./..."
	case VariantShellRun:
		a.draft = "!npm run build"
		a.rightOpen, a.rightTab = true, 4
		a.termState = termRunning
		a.termLines = append(termSamples(termRunning), termLine{text: "$ npm run build", color: -1})
	case VariantSessionMenu:
		*a.menuOpenPtr(0) = true
	case VariantSessionRename:
		a.renaming = 0
		if names := a.sessionNames(); len(names) > 0 {
			a.renameDraft = names[0]
		}
	case VariantSessionDelete:
		a.deleting = 0
	case VariantSearch:
		a.searchOpen = true
		a.searchQuery = ""
	case VariantTerm, VariantTermRunning, VariantTermExited:
		a.rightOpen, a.rightTab = true, 4
		switch a.variant {
		case VariantTermRunning:
			a.termState = termRunning
		case VariantTermExited:
			a.termState = termExited
		}
		a.termLines = termSamples(a.termState)
	case VariantRich:
		a.rounds = []round{{
			user:   "把排队的实现讲清楚",
			reply:  richReply,
			status: roundDone,
			ops:    []op{{action: "读取", target: "internal/ui/composer.go", result: "完成"}},
		}}
	case VariantBranch:
		a.branch, a.branches = branchSamples()
		a.branchMenuOpen = true
	case VariantBranchEnd:
		a.branch, a.branches = branchSamples()
		a.branchMenuOpen = true
		a.endBranchConfirm = true
	case VariantGoal:
		a.goalEditing = true
	case VariantTasks:
		a.rightOpen = true
		a.rightTab = 2
	case VariantChatMenu:
		a.chatMenuOpen = true
	case VariantOpenWith:
		a.openWithOpen = true
	case VariantModel:
		a.modelOpen = true
	case VariantEffort:
		a.effortOpen = true
	case VariantSkills:
		a.skillsOpen = true
	case VariantMode:
		a.modeOpen = true
	case VariantPermission:
		a.state = StateEmpty
		a.permissionOpen = true
	case VariantWorkspace:
		a.state = StateEmpty
		a.workspaceOpen = true
	case VariantRightPanel:
		a.rightOpen, a.rightTab = true, tabFiles
	case VariantReview:
		a.rightOpen, a.rightTab = true, tabChanges
	case VariantReviewScope:
		a.rightOpen, a.rightTab = true, tabChanges
		a.rp().scopeOpen = true
	case VariantAddTab:
		a.rightOpen, a.rightTab = true, tabFiles
		a.rp().newMenu = true
	case VariantRightExpanded:
		a.rightOpen, a.rightTab = true, tabFiles
		a.rp().expanded = true
	case VariantTimelineHover:
		// 截图没有指针：把第一轮的悬停动作钉住，只改显形，不改事实。
		a.timelineHover = 0
	case VariantTimelineToolDetail:
		// 第一轮工具分组展开，并展开其中的失败工具，走查输入 / 输出 / 失败原因。
		if len(a.rounds) > 0 {
			*a.openOf(0) = true
			*a.blockOf("tool:0:3", true) = true
		}
	}

	if a.state == StateWaiting {
		a.approvalCommand = "ls -la"
	}
	a.applyCommandPaletteVariant()
}

// applyCommandPaletteVariant 把走查变体落到面板开合上（截图用）。
func (a *App) applyCommandPaletteVariant() {
	switch a.variant {
	case VariantPalette:
		a.openCommandPalette()
	case VariantPaletteEmpty:
		a.openCommandPalette()
		a.palette.query = "zzzz"
		a.palette.filter = "zzzz"
		a.palette.lastSeen = "zzzz"
	}
}

// resetWiredViewState 只重置可丢的界面状态：不写任何样例事实，事实每帧从
// Source 读。连着 Source 时 loadSample 走这条路径。
func (a *App) resetWiredViewState() {
	a.rounds = nil
	a.summaryOpen = make(map[int]*bool)
	a.blockOpen = make(map[string]*bool)
	a.timelineHover = -1
	a.approvalCommand = ""
	a.contextUsed, a.contextLimit = 0, 0
	a.contextOpen = false
	a.confirmAlways = false
	a.approvalDecision = ""

	a.branch = ""
	a.branches = nil
	a.branchMenuOpen = false
	a.endBranchConfirm = false
	a.goal = ""
	a.goalDone = false
	a.goalEditing = false
	a.goalDraft = ""
	a.goalClearConfirm = false
	a.tasks = nil

	a.screen = ScreenMain
	a.settingsSection = secAccount
	a.themePref = "跟随系统"
	a.motionOff = false
	a.timeZone = "跟随系统"
	a.sidebarWidth = 0
	a.lastPrefs = PreferencesSnapshot{}
	// 目录选择的两处界面状态与本地登记的目录：从本地偏好里恢复。
	a.addWorkspacePick = dirPickState{}
	a.hooksPick = dirPickState{}
	a.workspaces = nil
	a.hooksProjectPath = ""

	a.attachments = nil
	a.attachmentEditing = false
	a.attachmentName, a.attachmentText = "", ""
	a.attachmentKind = "text"
	a.attachmentMedia, a.attachmentData = "", ""
	a.effort = 2
	a.effortOpen = false
	a.modelOpen = false
	a.modelQuery = ""
	a.skillsOpen = false
	a.skillQuery = ""
	a.mode = "Agent"
	a.modeOpen = false
	a.workspaceName = ""
	a.workspaceOpen = false
	a.workspaceQuery = ""
	a.workspacePath = ""
	a.permission = "默认权限"
	a.permissionChoice = ""
	a.permissionOpen = false
	a.permissionArmOpen = false
	a.permissionArmAck = false
	a.statusBranchOpen = false
	a.statusBranchQuery = ""
	a.chatMenuOpen = false
	a.openWithOpen = false
	a.openWithLoading = false
	a.openWithApp = "VS Code"
	a.chatPinned = false
	a.headerNote = ""
	a.searchOpen, a.searchQuery = false, ""
	a.renaming, a.deleting = -1, -1
	a.renameDraft = ""
	a.archived, a.renamed = map[int]bool{}, map[int]string{}
	a.menuOpen = map[int]*bool{}
	a.exported = ""
	a.termState = termIdle
	a.termLines = nil
	a.termCopied = false
	a.termInput = ""
	a.panels = newPanelView()
	a.model = ""

	if a.source != nil {
		prefs := a.source.Preferences()
		if prefs.Theme != "" {
			a.themePref = prefs.Theme
		}
		if prefs.TimeZone != "" {
			a.timeZone = prefs.TimeZone
		}
		a.motionOff = prefs.MotionOff
		a.sidebarWidth = float32(prefs.SidebarWidth)
		a.workspaces = append([]string(nil), prefs.Workspaces...)
		a.hooksProjectPath = prefs.HooksProjectPath
		a.notifyTaskDone = prefs.NotifyTaskDone
		a.notifySound = prefs.NotifySound
		a.notifyNeedsResponse = prefs.NotifyNeedsResponse
		a.lastPrefs = prefs
	}
	a.applyCommandPaletteVariant()
	a.applyPanelVariant()
}

// applyPanelVariant 把"停在右栏哪一页"这条视图状态落到走查变体上。
//
// 只切视图状态（打开哪一页），不造任何事实：连接模式（-connect）下右栏两页
// 读的是 PanelFacts 的真实读数，走查与截图需要能直接停在那一页上。每帧不做，
// 只在重置视图状态时落一次，用户关掉右栏不会被重新打开。
func (a *App) applyPanelVariant() {
	switch a.variant {
	case VariantRightPanel, VariantAddTab, VariantRightExpanded:
		a.rightOpen, a.rightTab = true, tabFiles
	case VariantReview, VariantReviewScope:
		a.rightOpen, a.rightTab = true, tabChanges
	}
}

// View 是每帧重建的界面。
func (a *App) View(c *gui.Context) {
	a.applyIntentResults()
	a.applyDirPickResults()
	a.syncSelectedSession()
	a.applyTokens(c)
	// 右栏文件页的目录投影每帧做一次：文件页与命令面板的文件组读同一份。
	a.syncPanelTree(false)
	a.commandPaletteKeys(c)
	a.debugKeys(c)
	a.layout(c)
	if a.screen == ScreenMain {
		a.dirPickDialog(c, dirPickWorkspace)
	}
	a.commandPalette(c)
	a.hostFrame()
}

// hostFrame 把这一帧要读的会话事实交给宿主外壳（托盘与系统通知）。只在接上
// OnFrame 时做；读的是 Source 的快照，不新增事实。
func (a *App) hostFrame() {
	if a.onFrame == nil {
		return
	}
	a.onFrame(a.hostSnapshot())
}

// hostSnapshot 取宿主外壳要的会话状态：当前会话的运行/待决/最后一轮，加上最近
// 五个会话。没有打开会话时这些字段为空，托盘仍能给出菜单。
func (a *App) hostSnapshot() HostSnapshot {
	out := HostSnapshot{Recent: a.recentSessions(5)}
	session, ok := a.liveSession()
	if !ok {
		return out
	}
	out.SessionID = session.ID
	out.SessionTitle = session.Title
	out.Running = session.Running
	out.Pending = session.Pending > 0
	out.LastTurnEnded = session.LastTurnEnded
	out.LastTurnIndex = session.LastTurnIndex
	out.LastTurnOutcome = session.LastTurnOutcome
	if approval, ok := a.source.Approval(); ok {
		out.PendingID = approval.ID
	} else if question, ok := a.source.Question(); ok {
		out.PendingID = question.ID
	}
	return out
}

// recentSessions 按最近活动时间取前 limit 个会话；时间都未知时按目录顺序。
func (a *App) recentSessions(limit int) []HostSession {
	names := a.sessionNames()
	if len(names) == 0 {
		return nil
	}
	titles := a.sessionTitles()
	places := a.sessionPlaces()
	type entry struct {
		index int
		at    time.Time
	}
	entries := make([]entry, 0, len(names))
	for i := range names {
		at := time.Time{}
		if i < len(places) {
			at = places[i].UpdatedAt
		}
		entries = append(entries, entry{index: i, at: at})
	}
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].at.Equal(entries[j].at) {
			return entries[i].index < entries[j].index
		}
		return entries[i].at.After(entries[j].at)
	})
	if limit <= 0 || limit > len(entries) {
		limit = len(entries)
	}
	out := make([]HostSession, 0, limit)
	for _, e := range entries[:limit] {
		title := names[e.index]
		if e.index < len(titles) && strings.TrimSpace(titles[e.index]) != "" {
			title = titles[e.index]
		}
		out = append(out, HostSession{ID: names[e.index], Title: title})
	}
	return out
}

// FocusSession 回到主界面并把左栏选中定位到该会话：通知点开与托盘「最近会话」
// 都由入口调用，下一帧的 syncSelectedSession 会打开它。
func (a *App) FocusSession(sessionID string) {
	a.screen = ScreenMain
	names := a.sessionNames()
	for i, name := range names {
		if name == sessionID {
			a.selectedSession = "s" + strconv.Itoa(i+1)
			a.lastSelectedSession = ""
			return
		}
	}
}

// NewSession 回到主界面并新建会话：托盘菜单的「新建会话」由入口调用。
func (a *App) NewSession() {
	a.screen = ScreenMain
	a.createSession()
}

// FocusComposer 回到主界面并把焦点放到输入区：全局唤起快捷键按下时由入口调用
// （mygo 的快捷键回调在主线程，与渲染同一线程）。
func (a *App) FocusComposer() {
	a.screen = ScreenMain
	a.focusComposer.Store(true)
}

// globalShortcut 读全局唤起的状态：接上 Source 时读注册状态；样例（-demo）用
// 本地的开关与键位（可丢的界面状态，不注册）。
func (a *App) globalShortcut() GlobalShortcutSnapshot {
	if a.source != nil {
		return a.source.GlobalShortcut()
	}
	return GlobalShortcutSnapshot{
		Enabled:     a.globalShortcutOn,
		Accelerator: a.globalShortcutAccel,
		Label:       shortcut.Label(a.globalShortcutAccel),
	}
}

// syncSelectedSession 在左栏选中项变化时打开对应会话；选中标识是视图里的行
// 标识（s1、s2…），真实会话标识从当前目录列表取。
func (a *App) syncSelectedSession() {
	if a.intents == nil || a.source == nil {
		return
	}
	if a.selectedSession == "" || a.selectedSession == a.lastSelectedSession {
		return
	}
	a.lastSelectedSession = a.selectedSession
	index := sessionIndex(a.selectedSession)
	names := a.sessionNames()
	if index < 0 || index >= len(names) {
		return
	}
	sessionID := names[index]
	if current, ok := a.liveSession(); ok && current.ID == sessionID {
		return
	}
	a.dispatch("打开会话", func(ctx context.Context) error {
		return a.intents.OpenSession(ctx, sessionID)
	}, nil)
}

// persistPreferences 在主题/动效/时区/侧栏宽度/工作区目录变化后落盘；只在接上
// Intents 时。
func (a *App) persistPreferences() {
	if a.intents == nil {
		return
	}
	prefs := PreferencesSnapshot{
		Theme: a.themePref, MotionOff: a.motionOff,
		TimeZone: a.timeZone, SidebarWidth: int(a.sidebarWidth),
		Workspaces:          append([]string(nil), a.workspaces...),
		HooksProjectPath:    a.hooksProjectPath,
		NotifyTaskDone:      a.notifyTaskDone,
		NotifySound:         a.notifySound,
		NotifyNeedsResponse: a.notifyNeedsResponse,
	}
	if samePreferences(prefs, a.lastPrefs) {
		return
	}
	a.lastPrefs = prefs
	a.dispatch("保存偏好", func(ctx context.Context) error {
		return a.intents.SetPreferences(ctx, prefs)
	}, nil)
}

// samePreferences 比两份本地偏好：Workspaces 是切片，逐项比。
func samePreferences(a, b PreferencesSnapshot) bool {
	if a.Theme != b.Theme || a.MotionOff != b.MotionOff || a.TimeZone != b.TimeZone ||
		a.SidebarWidth != b.SidebarWidth || a.HooksProjectPath != b.HooksProjectPath ||
		a.NotifyTaskDone != b.NotifyTaskDone || a.NotifySound != b.NotifySound ||
		a.NotifyNeedsResponse != b.NotifyNeedsResponse {
		return false
	}
	if len(a.Workspaces) != len(b.Workspaces) {
		return false
	}
	for i := range a.Workspaces {
		if a.Workspaces[i] != b.Workspaces[i] {
			return false
		}
	}
	return true
}

// dispatch 在后台执行一次写入意图：失败原因挂到视图的错误行，成功后把视图状态
// 更新闭包交回渲染线程（下一帧执行），不在后台协程里改视图。
func (a *App) dispatch(label string, action func(context.Context) error, onOK func()) {
	a.dispatchThen(label, action, onOK, nil)
}

// dispatchThen 是 dispatch 的带失败回调版本：onFail 在渲染线程上收到失败原因
// （标签已拼好的整句），给需要在原地显示原因的表单用。
func (a *App) dispatchThen(
	label string,
	action func(context.Context) error,
	onOK func(),
	onFail func(string),
) {
	if a.intents == nil || action == nil {
		return
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		err := action(ctx)
		a.intentMu.Lock()
		if err != nil {
			msg := label + "：" + err.Error()
			a.intentErr = msg
			if onFail != nil {
				a.intentFail = append(a.intentFail, func() { onFail(msg) })
			}
		} else {
			a.intentErr = ""
			if onOK != nil {
				a.intentDone = append(a.intentDone, onOK)
			}
		}
		a.intentMu.Unlock()
		if a.changed != nil {
			a.changed()
		}
	}()
}

// applyIntentResults 在渲染线程上执行后台意图交回的成功与失败回调。
func (a *App) applyIntentResults() {
	a.intentMu.Lock()
	done := a.intentDone
	failed := a.intentFail
	a.intentDone, a.intentFail = nil, nil
	a.intentMu.Unlock()
	for _, update := range done {
		update()
	}
	for _, fail := range failed {
		fail()
	}
}

// intentError 是最近一次失败的动作原因；空表示没有。
func (a *App) intentError() string {
	a.intentMu.Lock()
	defer a.intentMu.Unlock()
	return a.intentErr
}

// dismissIntentError 清掉错误行。
func (a *App) dismissIntentError() {
	a.intentMu.Lock()
	a.intentErr = ""
	a.intentMu.Unlock()
}

// actionErrorView 是一行失败原因：只在有错误时出现，关闭入口就在同一行。
func (a *App) actionErrorView(c *gui.Context) {
	reason := a.intentError()
	if reason == "" {
		return
	}
	t := c.Theme()
	gui.Row(c).Shrink(0).Padding(space1, space2).Gap(space2).AlignItems(gui.Center).
		Label("操作未完成").Children(func() {
		iconGlyph(c, iconWarn, t.Danger, "exclamation")
		typeSecondary.style(t, gui.Text(c, reason)).TextColor(t.Danger).Grow(1).SingleLine()
		if iconButton(c, iconButtonSpec{icon: iconXMark, label: "关闭操作失败提示", size: rowCompact}) {
			a.dismissIntentError()
		}
	})
}

// applyTokens 把 U2 的 token 表换成本帧的主题（tokens.go 的 theme）：
// 外观跟随系统，文本缩放与高对比跟随系统设置。
func (a *App) applyTokens(c *gui.Context) {
	dark := false
	if sys := c.Theme(); sys != nil {
		dark = sys.Dark
	}
	// 设置 · 外观 · 主题：跟随系统 / 明 / 暗，改完立即生效（同一帧换主题）。
	switch a.themePref {
	case "明":
		dark = false
	case "暗":
		dark = true
	}
	c.SetTheme(theme(dark, c.Preferences()))
	// Markdown 的块样式跟着当前主题走（每帧换一次指针，解析缓存留在 a.md 里）。
	a.md.Theme = markdownTheme(c.Theme())
}

// debugKeys 是仅用于走查的键位：1–5 切换状态。焦点在输入框里时不抢键，
// 避免把数字从草稿里吃掉（U3 定稿前的临时做法）。
func (a *App) debugKeys(c *gui.Context) {
	if a.palette.open || c.FocusTakesText() {
		return
	}
	switch {
	case c.Shortcut(0, gui.Key1):
		a.SetState(StateDefault)
	case c.Shortcut(0, gui.Key2):
		a.SetState(StateRunning)
	case c.Shortcut(0, gui.Key3):
		a.SetState(StateWaiting)
	case c.Shortcut(0, gui.Key4):
		a.SetState(StateFailed)
	case c.Shortcut(0, gui.Key5):
		a.SetState(StateEmpty)
	}
}

// searchKeys 是搜索面的键位（U3 T18）：Esc 收起搜索面（焦点回到列表）。
func (a *App) searchKeys(c *gui.Context) {
	if a.palette.open {
		return
	}
	if a.searchOpen && c.Shortcut(0, gui.KeyEscape) {
		a.searchOpen = false
		a.searchQuery = ""
	}
}

// terminalKeys 是终端的键位（U3 T17）：Ctrl-C 中止、Cmd-K 清屏、Esc 收回右栏
// （回到主界面）。Ctrl-C 在终端输入框里仍是"复制"，所以只在焦点不在输入框时接管。
func (a *App) terminalKeys(c *gui.Context) {
	if a.palette.open {
		return
	}
	if !a.rightOpen || a.rightTab != 4 {
		return
	}
	if !c.FocusTakesText() && c.Shortcut(gui.Ctrl, gui.KeyC) && a.termState == termRunning {
		a.termState = termExited
	}
	if c.Shortcut(gui.Cmd, gui.KeyK) {
		a.termLines = nil
	}
	if c.Shortcut(0, gui.KeyEscape) {
		a.rightOpen = false
	}
}

// layout 摆六个区域：左栏 +（标题行 / 对话流 / 输入位）+ 右栏。
// 挤占顺序按 U1-ia §3.1：右栏先失空间，主区不缩到 400 以下。
func (a *App) layout(c *gui.Context) {
	t := c.Theme()
	sidebar := a.sidebarWidthValue()
	if a.sidebarCollapsed {
		sidebar = SidebarRailWidth
	}
	windowWidth, _ := c.Size()

	// 右栏只在窗口宽到能并排时展开：主区保 400，右栏保 280。
	// 「展开面板」时右栏占满主区（RP-A-36/37）；更窄的窗口属于 compact，
	// 右栏整屏替代主区，本批次未做。
	panel := a.rpWidth(windowWidth, sidebar)

	a.searchKeys(c)
	a.terminalKeys(c)
	gui.Row(c).Fill().AlignItems(gui.Stretch).
		Background(t.Background).Children(func() {
		if a.screen == ScreenSettings {
			a.settings(c)
			return
		}
		a.sidebar(c)
		gui.Column(c).Grow(1).MinWidth(MainMinWidth).Children(func() {
			a.header(c)
			a.actionErrorView(c)
			gui.Row(c).Grow(1).AlignItems(gui.Stretch).Children(func() {
				// 「展开面板」：右栏占满主区（RP-A-36/37），再按一次恢复。
				if a.rightOpen && a.rp().expanded {
					remaining, _ := c.Size()
					a.rightPanel(c, remaining-sidebar)
					return
				}
				// 窄到并排不下时（compact），右栏整屏替代主区（U1-ia §3.1）：
				// 同一时刻只显示一处，关闭右栏后主区回来。
				if a.rightOpen && panel == 0 {
					remaining, _ := c.Size()
					a.rightPanel(c, remaining-sidebar)
					return
				}
				gui.Column(c).Grow(1).Children(func() {
					// 空态（NewThread）：品牌标记 + 开始构建 + 输入卡整体居中，
					// 不再画时间线的「还没有消息」。
					if a.state == StateEmpty && !a.wired() {
						a.emptyThread(c)
						return
					}
					a.timeline(c)
					a.composer(c)
					if a.statusBarVisible() {
						a.statusBar(c)
					}
				})
				if panel > 0 {
					a.rightPanel(c, panel)
				}
			})
		})
	})
}

// sidebarWidthValue 是左栏宽度：本地偏好给了非零值就用它。
func (a *App) sidebarWidthValue() float32 {
	if a.sidebarWidth > 0 {
		return a.sidebarWidth
	}
	return SidebarWidth
}

// panelWidth 是右栏实际占的宽度：未打开、或窗口窄到并排不下时为零。
func panelWidth(windowWidth, sidebar float32, rightOpen bool) float32 {
	if !rightOpen {
		return 0
	}
	maxPanel := windowWidth - sidebar - RightPanelMainMinWidth
	if maxPanel < RightPanelMinWidth {
		return 0
	}
	return clamp(RightPanelPreferredWidth, RightPanelMinWidth, maxPanel)
}

// clamp 把 v 收进 [lo, hi]；当 lo > hi 时取 lo。
func clamp(v, lo, hi float32) float32 {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// openOf 取某一轮的展开态指针（视图状态，按需创建并缓存）。默认全部折叠，
// 只影响初始观感，不涉及任何事实。
func (a *App) openOf(i int) *bool {
	if a.summaryOpen == nil {
		a.summaryOpen = make(map[int]*bool)
	}
	if p, ok := a.summaryOpen[i]; ok {
		return p
	}
	v := false
	p := &v
	a.summaryOpen[i] = p
	return p
}

// liveConnection 是连接阶段：有 Source 时读它，否则按样例视为已连接。
func (a *App) liveConnection() Connection {
	if a.source != nil {
		return a.source.Connection()
	}
	return Connection{Phase: "connected"}
}

// disconnected 表示当前连不上（输入区要禁用、原因要可见）。
func (a *App) disconnected() bool { return a.liveConnection().Phase == "disconnected" }

// wired 表示当前由 Source 提供事实。
func (a *App) wired() bool { return a.source != nil }

// sessionNames 是左栏要显示的会话名：连着 Source 就用它的，否则用样例。
func (a *App) sessionNames() []string {
	if a.source != nil {
		return a.source.SessionNames()
	}
	names, _ := sessionSamples(a.state)
	return names
}

// sessionTitles 是左栏会话行的显示名：接上 Source 时用标题/预览，样例用名字。
func (a *App) sessionTitles() []string {
	if a.source != nil {
		if titles := a.source.SessionTitles(); len(titles) > 0 {
			return titles
		}
		return a.source.SessionNames()
	}
	names, _ := sessionSamples(a.state)
	return names
}

// sessionPlaces 是左栏会话的分组事实：工作目录与最近活动时间。连着 Source 时
// 读它，否则用样例（样例里同样是可丢的展示数据）。
func (a *App) sessionPlaces() []SessionPlaceSnapshot {
	if a.source != nil {
		return a.source.SessionPlaces()
	}
	return sampleSessionPlaces(a.state)
}

// liveSession 取 Source 的当前会话快照。
func (a *App) liveSession() (SessionSnapshot, bool) {
	if a.source == nil {
		return SessionSnapshot{}, false
	}
	return a.source.Session()
}

// liveRunning 是"这一轮在跑"：有 Source 时看事实，否则看样例状态。
func (a *App) liveRunning() bool {
	if snap, ok := a.liveSession(); ok {
		return snap.Running
	}
	return a.state == StateRunning
}

// liveApproval 取待决审批（Source 优先）。
func (a *App) liveApproval() (ApprovalSnapshot, bool) {
	if a.source == nil {
		return ApprovalSnapshot{}, false
	}
	return a.source.Approval()
}

// sessionTasks 取后台任务：连着 Source 时用会话快照，否则用样例。
func (a *App) sessionTasks() ([]task, bool) {
	if snap, ok := a.liveSession(); ok {
		tasks := make([]task, 0, len(snap.Tasks))
		for _, item := range snap.Tasks {
			state := taskRunning
			if item.Stopping {
				state = taskDone
			}
			tasks = append(tasks, task{id: item.ID, name: item.Description, state: state})
		}
		return tasks, true
	}
	return a.tasks, false
}

// branchState 返回当前分支与分支清单：连着 Source 时取会话快照，否则用样例。
func (a *App) branchState() (string, []string, bool) {
	if snap, ok := a.liveSession(); ok {
		names := make([]string, 0, len(snap.Branches))
		for _, branch := range snap.Branches {
			names = append(names, branch.ID)
		}
		if len(names) == 0 {
			names = []string{"main"}
		}
		return snap.Branch, names, true
	}
	return a.branch, a.branches, false
}

// currentModel 是标题行末尾要显示的模型：连着 Source 时取当前选择，否则用样例。
func (a *App) currentModel() string {
	if st, ok := a.liveSettings(); ok && st.Model != "" {
		return st.Model
	}
	if a.model != "" {
		return a.model
	}
	return sampleModel
}

// liveQuestion 取待答提问（Source 优先）。
func (a *App) liveQuestion() (QuestionSnapshot, bool) {
	if a.source == nil {
		return QuestionSnapshot{}, false
	}
	return a.source.Question()
}

// accountInitialValue 是页脚头像的首字母：连着 Source 时取账号事实，否则用样例。
func (a *App) accountInitialValue() string {
	if a.source != nil {
		return a.source.Account().Initial
	}
	return a.accountInitial
}

// liveSettings 取设置里的事实快照。
func (a *App) liveSettings() (SettingsSnapshot, bool) {
	if a.source == nil {
		return SettingsSnapshot{}, false
	}
	return a.source.Settings(), true
}

// liveAccount 取账号事实快照。
func (a *App) liveAccount() (AccountSnapshot, bool) {
	if a.source == nil {
		return AccountSnapshot{}, false
	}
	return a.source.Account(), true
}

// liveUsage 取账号用量快照。
func (a *App) liveUsage() (UsageSnapshot, bool) {
	if a.source == nil {
		return UsageSnapshot{}, false
	}
	return a.source.Usage(), true
}

// liveTerminal 取终端状态；第三个返回值表示是否由 Source 提供。
func (a *App) liveTerminal() (TerminalSnapshot, bool) {
	if a.source == nil {
		return TerminalSnapshot{}, false
	}
	return a.source.Terminal(), true
}
