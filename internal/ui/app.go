package ui

import (
	"fmt"
	"strconv"
	"strings"

	gui "github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/markdown"
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

// 设置的分节序号（U1-ia §4 的 8 节，顺序与文档一致）。
const (
	secAccount    = 0 // 账号与套餐
	secProviders  = 1 // 模型与供应商
	secRules      = 2 // 命令规则
	secExtensions = 3 // 扩展
	secCron       = 4 // 定时任务
	secRuntime    = 5 // 运行时与模态
	secStorage    = 6 // 存储与诊断
	secAppearance = 7 // 外观与键位
)

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
	// VariantTierConfirm 是权限档位切换的两步确认态。
	VariantTierConfirm
	// VariantExtDetail / ExtSelect / MCPTest / MCPFail 是扩展的详情、选择态与 MCP 测试三态。
	VariantExtDetail
	VariantExtSelect
	VariantMCPTest
	VariantMCPFail
	// VariantShell / ShellRun 是输入区命令模式（`!`）的空闲与运行中。
	VariantShell
	VariantShellRun
	// VariantStorage / AppearanceKeys 是设置第 7、8 节的走查态。
	VariantStorage
	VariantAppearanceKeys
	// VariantCron / CronDelete / Runtime / RuntimeSubmit 是设置第 5、6 节的走查态。
	VariantCron
	VariantCronDelete
	VariantRuntime
	VariantRuntimeSubmit
	// VariantSessionMenu / Rename / SessionDelete / Search 是左栏目录动作的四个走查态。
	VariantSessionMenu
	VariantSessionRename
	VariantSessionDelete
	VariantSearch
	// VariantTerm / TermRunning / TermExited 打开右栏的终端标签，三种状态。
	VariantTerm
	VariantTermRunning
	VariantTermExited
	// VariantExt 设置 · 扩展（列表）；后三个是同一节的加载 / 空 / 失败态。
	VariantExt
	VariantExtFailed
	VariantExtEmpty
	VariantExtLoading
	// VariantSettings 进入设置屏（默认分节）。
	VariantSettings
	// VariantSettingsRules 进入设置屏并停在命令规则的清空确认态。
	VariantSettingsRules
)

// variantNames 是 -variant 的取值。
var variantNames = map[string]Variant{
	"": VariantNone, "none": VariantNone, "branch": VariantBranch,
	"goal": VariantGoal, "tasks": VariantTasks,
	"branch-end": VariantBranchEnd, "rich": VariantRich,
	"term-disconnected": VariantTermDisconnected, "term-recovered": VariantTermRecovered,
	"tier-confirm": VariantTierConfirm,
	"ext-detail": VariantExtDetail, "ext-select": VariantExtSelect,
	"mcp-test": VariantMCPTest, "mcp-fail": VariantMCPFail,
	"shell": VariantShell, "shell-run": VariantShellRun,
	"storage": VariantStorage, "appearance-keys": VariantAppearanceKeys,
	"cron": VariantCron, "cron-delete": VariantCronDelete,
	"runtime": VariantRuntime, "runtime-submit": VariantRuntimeSubmit,
	"session-menu": VariantSessionMenu, "rename": VariantSessionRename,
	"session-delete": VariantSessionDelete, "search": VariantSearch,
	"term": VariantTerm, "term-running": VariantTermRunning,
	"term-exited": VariantTermExited,
	"ext": VariantExt, "ext-failed": VariantExtFailed,
	"ext-empty": VariantExtEmpty, "ext-loading": VariantExtLoading, "settings": VariantSettings, "settings-rules": VariantSettingsRules,
}

// ParseVariant 解析 -variant。
func ParseVariant(v string) (Variant, error) {
	if got, ok := variantNames[v]; ok {
		return got, nil
	}
	return VariantNone, fmt.Errorf("未知变体 %q：可用 none、term-disconnected、term-recovered、tier-confirm、ext-detail、ext-select、mcp-test、mcp-fail、shell、shell-run、storage、appearance-keys、cron、cron-delete、runtime、runtime-submit、session-menu、rename、session-delete、search、branch、branch-end、rich、goal、tasks、settings、settings-rules、term、term-running、term-exited、ext、ext-failed、ext-empty、ext-loading", v)
}

// Options 是骨架的启动参数。
type Options struct {
	// State 是起始状态。
	State State
	// Variant 是额外的走查态（分支菜单 / 目标编辑 / 后台任务）。
	Variant Variant
	// StateChanged 在调试键切换状态后回调，供入口更新窗口标题。
	StateChanged func(State)
}

// App 只保存可丢的界面状态：当前样例状态、左栏与右栏的开合、摘要展开、
// 草稿文本与滚动位置、上下文用量的展开，外加几个只用于样例的展示字段
// （非默认模型标记、审批命令原文、用量样例）。它不保存任何事实，也不发起
// 任何写入。
type App struct {
	state   State
	variant Variant
	onState func(State)

	sidebarCollapsed bool
	rightOpen        bool
	rightTab         int
	selectedSession  string

	draft       string
	rounds      []round
	// summaryOpen 是每一轮的展开态（可丢的视图状态）：按轮次索引存一份 *bool，
	// 指针稳定、按需创建——轮次变多时不必重建数组，view 里也不攒第二份事实。
	summaryOpen map[int]*bool

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
	goal           string
	goalDone       bool
	goalEditing    bool

	// 设置屏：屏幕、当前分节、三个分节的样例值。全部是可丢的界面状态。
	screen             Screen
	settingsSection    int
	themePref          string
	motionOff          bool
	provider           string
	providers          []string
	modelChoice        string
	models             []string
	modelPrice         string
	rules              []commandRule
	clearRulesConfirm  bool

	// 定时任务（设置第 5 节）与运行时（第 6 节）。
	cron          []cronTask
	cronState     extState
	cronDeleting  int
	runtime       []runtimePart
	runtimeState  extState
	modalTask     string
	modalSubmitted string

	// 第 8 节：时区；第 7 节：诊断导出/打开的回执。
	timeZone            string
	diagnosticsExported bool
	diagnosticsOpened   bool

	// 扩展：页签、加载态与四类样例。
	extTab   int
	extState extState

	// 扩展的详情（二级页）、选择态与 MCP 的认证 / 导入 / 连接测试。
	extDetail          int
	extUninstallConfirm bool
	selectMode         bool
	extSelected        map[string]bool
	mcpTest            map[string]mcpState
	mcpTestDone        map[string]bool
	importMCP          string

	// 待发附件（样例）与权限档位（提交值与草稿值分开：切换要两步确认）。
	attachments []string
	tier        string
	tierDraft   string
	tierPending string

	// 会话目录：搜索面、行内重命名、删除确认、归档与导出反馈（都属可丢的视图状态）。
	searchOpen bool
	searchQuery string
	renaming   int
	renameDraft string
	deleting   int
	archived   map[int]bool
	renamed    map[int]string
	exported   string
	menuOpen   map[int]*bool

	// 终端（右栏第五个标签）：状态、输出行、提示行与"已复制"反馈。
	termState  termState
	termLines  []termLine
	termInput  string
	termCopied bool
	plugins  []extItem
	skills   []extItem
	mcps     []extItem
	hooks    []extItem
	tasks          []task

	// contextUsed / contextLimit 是样例里的上下文用量，百分比由二者算；
	// contextOpen 是明细的展开状态。三者都是可丢的界面状态（UI-8）。
	contextUsed  int
	contextLimit int
	contextOpen  bool

	// model 是非默认模型在标题行末尾的淡标记；空表示取默认值，不上屏（UI-9）。
	model string
	// approvalCommand 是审批卡上的命令原文；为空时拿不到原文，不提供「一直允许」。
	approvalCommand string
	// confirmAlways 是「一直允许」的确认态：第一次点击先换成确认，再点才生效。
	confirmAlways bool
	// approvalDecision 是样例里点过的决定（只用于界面反馈，不发起任何写入）。
	approvalDecision string

	feed gui.ScrollState
}

// NewApp 建立骨架。状态切换只改界面样例，不连任何服务。
func NewApp(o Options) *App {
	a := &App{
		state:           o.State,
		variant:         o.Variant,
		onState:         o.StateChanged,
		selectedSession: "s1",
		connected:       true,
		accountInitial:  "E",
	}
	a.loadSample()
	return a
}

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
	a.rounds = sampleRounds(a.state)
	a.summaryOpen = make(map[int]*bool)
	a.approvalCommand = ""
	// 上下文用量的样例值：12%，与线框一致；上限只是样例，不是事实。
	a.contextUsed, a.contextLimit = 24_600, 200_000
	a.contextOpen = false
	a.confirmAlways = false
	a.approvalDecision = ""

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
	a.settingsSection = secAppearance
	a.themePref = "跟随系统"
	a.motionOff = false
	a.providers = []string{"Anthropic", "OpenAI"}
	a.provider = "Anthropic"
	a.models = []string{"Opus 4.6", "Sonnet 4.6"}
	a.modelChoice = "Opus 4.6"
	a.modelPrice = "$3 / 百万 token · $15 / 百万 token（输出）"
	a.rules = []commandRule{
		{command: "go test ./...", decision: "允许", match: "精确"},
		{command: "rm -rf", decision: "拒绝", match: "前缀"},
	}
	a.clearRulesConfirm = false
	a.cron = cronSamples()
	a.cronState, a.cronDeleting = extLoaded, -1
	a.runtime = runtimeSamples()
	a.runtimeState = extLoaded
	a.modalTask, a.modalSubmitted = "", ""
	a.timeZone = "跟随系统"
	a.diagnosticsExported, a.diagnosticsOpened = false, false
	a.extTab, a.extState = 0, extLoaded
	a.extDetail, a.extUninstallConfirm = -1, false
	a.selectMode = false
	a.extSelected = map[string]bool{}
	a.mcpTest, a.mcpTestDone = map[string]mcpState{}, map[string]bool{}
	a.importMCP = ""
	a.attachments = []string{"发送路径.md"}
	a.tier, a.tierDraft, a.tierPending = "工作区内自动", "工作区内自动", ""
	a.searchOpen, a.searchQuery = false, ""
	a.renaming, a.deleting = -1, -1
	a.renameDraft = ""
	a.archived, a.renamed = map[int]bool{}, map[int]string{}
	a.menuOpen = map[int]*bool{}
	a.exported = ""
	a.termState = termIdle
	a.termLines = termSamples(termIdle)
	a.termCopied = false
	a.skills = extSamples(extSkills)
	a.plugins = extSamples(extPlugins)
	a.mcps = extSamples(extMCP)
	a.hooks = extSamples(extHooks)

	switch a.variant {
	case VariantSettings:
		a.screen = ScreenSettings
		a.settingsSection = secAppearance // 外观：主题与动效
	case VariantSettingsRules:
		a.screen = ScreenSettings
		a.settingsSection = secRules // 命令规则：清空确认态
		a.clearRulesConfirm = true
	case VariantTermDisconnected:
		a.rightOpen, a.rightTab = true, 4
		a.termState = termDisconnected
		a.termLines = append(termSamples(termExited), termLine{text: "connection lost", color: 1})
	case VariantTermRecovered:
		a.rightOpen, a.rightTab = true, 4
		a.termState = termIdle
		a.termLines = append(termSamples(termExited), termLine{text: "connection lost", color: 1},
			termLine{text: "reconnected", color: 2})
	case VariantTierConfirm:
		a.screen, a.settingsSection = ScreenSettings, secProviders
		a.tierDraft, a.tierPending = "每次询问", "每次询问"
	case VariantExtDetail:
		a.screen, a.settingsSection, a.extTab = ScreenSettings, secExtensions, 1
		a.extDetail = 0
	case VariantExtSelect:
		a.screen, a.settingsSection, a.extTab = ScreenSettings, secExtensions, 1
		a.selectMode = true
		a.extSelected = map[string]bool{"GitHub": true, "Linear": true}
	case VariantMCPTest:
		a.screen, a.settingsSection, a.extTab = ScreenSettings, secExtensions, 2
		a.mcpTest["filesystem"] = mcpTesting
		a.mcpTest["postgres"] = mcpOK
	case VariantMCPFail:
		a.screen, a.settingsSection, a.extTab = ScreenSettings, secExtensions, 2
		a.mcpTest["postgres"] = mcpFail
		a.importMCP = "filesystem"
	case VariantShell:
		a.draft = "!go test ./..."
	case VariantShellRun:
		a.draft = "!npm run build"
		a.rightOpen, a.rightTab = true, 4
		a.termState = termRunning
		a.termLines = append(termSamples(termRunning), termLine{text: "$ npm run build", color: -1})
	case VariantStorage:
		a.screen, a.settingsSection = ScreenSettings, secStorage
	case VariantAppearanceKeys:
		a.screen, a.settingsSection = ScreenSettings, secAppearance
	case VariantCron:
		a.screen, a.settingsSection = ScreenSettings, secCron
	case VariantCronDelete:
		a.screen, a.settingsSection, a.cronDeleting = ScreenSettings, secCron, 0
	case VariantRuntime:
		a.screen, a.settingsSection = ScreenSettings, secRuntime
	case VariantRuntimeSubmit:
		a.screen, a.settingsSection = ScreenSettings, secRuntime
		a.modalTask = "重跑全部语言测试"
		a.modalSubmitted = "重跑全部语言测试"
	case VariantSessionMenu:
		*a.menuOpenPtr(0) = true
	case VariantSessionRename:
		a.renaming = 0
		a.renameDraft = "修发送路径"
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
	case VariantExt, VariantExtFailed, VariantExtEmpty, VariantExtLoading:
		a.screen = ScreenSettings
		a.settingsSection = secExtensions
		switch a.variant {
		case VariantExtFailed:
			a.extState = extFailed
		case VariantExtEmpty:
			a.extState = extEmpty
		case VariantExtLoading:
			a.extState = extLoading
		}
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
	}

	if a.state == StateWaiting {
		a.approvalCommand = "rm -rf ./tmp/*"
	}
}

// View 是每帧重建的界面。
func (a *App) View(c *gui.Context) {
	a.applyTokens(c)
	a.debugKeys(c)
	a.layout(c)
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
	if c.FocusTakesText() {
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
	if a.searchOpen && c.Shortcut(0, gui.KeyEscape) {
		a.searchOpen = false
		a.searchQuery = ""
	}
}

// terminalKeys 是终端的键位（U3 T17）：Ctrl-C 中止、Cmd-K 清屏、Esc 收回右栏
// （回到主界面）。Ctrl-C 在终端输入框里仍是"复制"，所以只在焦点不在输入框时接管。
func (a *App) terminalKeys(c *gui.Context) {
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
	sidebar := float32(SidebarWidth)
	if a.sidebarCollapsed {
		sidebar = SidebarRailWidth
	}
	windowWidth, _ := c.Size()

	// 右栏只在窗口宽到能并排时展开：主区保 400，右栏保 280。
	// 更窄的窗口属于 compact，右栏应整屏替代主区，本批次未做。
	panel := panelWidth(windowWidth, sidebar, a.rightOpen)

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
			gui.Row(c).Grow(1).AlignItems(gui.Stretch).Children(func() {
				// 窄到并排不下时（compact），右栏整屏替代主区（U1-ia §3.1）：
				// 同一时刻只显示一处，关闭右栏后主区回来。
				if a.rightOpen && panel == 0 {
					remaining, _ := c.Size()
					a.rightPanel(c, remaining-sidebar)
					return
				}
				gui.Column(c).Grow(1).Children(func() {
					a.timeline(c)
					a.composer(c)
				})
				if panel > 0 {
					a.rightPanel(c, panel)
				}
			})
		})
	})
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
