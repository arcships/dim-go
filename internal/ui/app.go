package ui

import (
	"fmt"
	"strconv"
	"strings"

	gui "github.com/egoist/mygo/ui"
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
	// VariantSettings 进入设置屏（默认分节）。
	VariantSettings
	// VariantSettingsRules 进入设置屏并停在命令规则的清空确认态。
	VariantSettingsRules
)

// variantNames 是 -variant 的取值。
var variantNames = map[string]Variant{
	"": VariantNone, "none": VariantNone, "branch": VariantBranch,
	"goal": VariantGoal, "tasks": VariantTasks,
	"settings": VariantSettings, "settings-rules": VariantSettingsRules,
}

// ParseVariant 解析 -variant。
func ParseVariant(v string) (Variant, error) {
	if got, ok := variantNames[v]; ok {
		return got, nil
	}
	return VariantNone, fmt.Errorf("未知变体 %q：可用 none、branch、goal、tasks、settings、settings-rules", v)
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
	summaryOpen []bool

	// connected 与 accountInitial 是样例里的连接态与账号首字母：只用于画状态点
	// 与头像（UI-23），不是事实来源。
	connected      bool
	accountInitial string

	// 分支、目标、后台任务是样例里的展示数据（不是事实）：分支名、目标文本、
	// 任务名都属于内容字符串（U1-content.json）。
	branch         string
	branches       []string
	branchMenuOpen bool
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
	a.summaryOpen = make([]bool, len(a.rounds))
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
	a.goal = "把发送路径改成排队"
	a.goalDone = false
	a.goalEditing = false
	a.tasks = taskSamples()

	a.screen = ScreenMain
	a.settingsSection = 0
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

	switch a.variant {
	case VariantSettings:
		a.screen = ScreenSettings
		a.settingsSection = 0 // 外观：主题与动效
	case VariantSettingsRules:
		a.screen = ScreenSettings
		a.settingsSection = 2 // 命令规则：清空确认态
		a.clearRulesConfirm = true
	case VariantBranch:
		a.branch, a.branches = branchSamples()
		a.branchMenuOpen = true
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
