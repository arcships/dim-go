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

// Options 是骨架的启动参数。
type Options struct {
	// State 是起始状态。
	State State
	// StateChanged 在调试键切换状态后回调，供入口更新窗口标题。
	StateChanged func(State)
}

// App 只保存可丢的界面状态：当前样例状态、左栏与右栏的开合、摘要展开、
// 草稿文本与滚动位置。它不保存任何事实，也不发起任何写入。
type App struct {
	state   State
	onState func(State)

	sidebarCollapsed bool
	rightOpen        bool
	rightTab         int
	selectedSession  string

	draft       string
	rounds      []round
	summaryOpen []bool

	feed gui.ScrollState
}

// NewApp 建立骨架。状态切换只改界面样例，不连任何服务。
func NewApp(o Options) *App {
	a := &App{
		state:           o.State,
		onState:         o.StateChanged,
		selectedSession: "s1",
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
}

// View 是每帧重建的界面。
func (a *App) View(c *gui.Context) {
	a.debugKeys(c)
	a.layout(c)
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
	panel := float32(0)
	if a.rightOpen {
		maxPanel := windowWidth - sidebar - RightPanelMainMinWidth
		if maxPanel >= RightPanelMinWidth {
			panel = clamp(RightPanelPreferredWidth, RightPanelMinWidth, maxPanel)
		}
	}

	gui.Row(c).Fill().AlignItems(gui.Stretch).
		Background(t.Background).Children(func() {
		a.sidebar(c)
		gui.Column(c).Grow(1).MinWidth(MainMinWidth).Children(func() {
			a.header(c)
			gui.Row(c).Grow(1).AlignItems(gui.Stretch).Children(func() {
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
