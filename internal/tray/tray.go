// Package tray 是宿主外壳的托盘模型：菜单项、图标状态与提示文字。
//
// 它只做纯计算，不碰平台 API：界面层每帧把事实（窗口可见、运行中、待审批、
// 最近会话、关窗偏好）交给 Menu / Icon / Tooltip，入口再把这些结果转成 mygo 的
// 托盘菜单与图标。这样菜单项与状态标记能在没有窗口的测试里逐条核对。
//
// 事实边界：本包不存事实、不读会话；RecentSessions 的标题与顺序由界面层给出，
// 这里只截断到前 RecentLimit 条。
package tray

import (
	"strconv"
	"strings"
)

// RecentLimit 是托盘「最近会话」最多列几条（参考产品的最近列表取前 5）。
const RecentLimit = 5

// Action 是一个菜单项对应的动作；入口按它分发，不靠标签字符串比较。
type Action string

const (
	// ActionShowWindow 显示窗口并聚焦。
	ActionShowWindow Action = "show-window"
	// ActionHideWindow 隐藏窗口（进程留在托盘）。
	ActionHideWindow Action = "hide-window"
	// ActionNewSession 新建会话。
	ActionNewSession Action = "new-session"
	// ActionOpenSession 打开一条最近会话；Item.SessionID 是会话标识。
	ActionOpenSession Action = "open-session"
	// ActionToggleCloseToTray 切换「关闭窗口时最小化到托盘」。
	ActionToggleCloseToTray Action = "toggle-close-to-tray"
	// ActionQuit 退出应用。
	ActionQuit Action = "quit"
)

// RecentSession 是托盘里的一条最近会话。
type RecentSession struct {
	ID    string
	Title string
}

// State 是托盘要读的界面事实。
type State struct {
	// WindowVisible 表示主窗口当前是否显示；菜单第一项据此在显示/隐藏之间切换。
	WindowVisible bool
	// Running 表示当前会话有一轮正在运行。
	Running bool
	// WaitingApproval 表示当前会话有审批或提问待决。
	WaitingApproval bool
	// Recent 是按最近活动排序的会话；只取前 RecentLimit 条。
	Recent []RecentSession
	// CloseToTray 是关窗偏好，决定勾选项的状态。
	CloseToTray bool
}

// Status 是托盘图标要表达的状态。等待审批优先于运行中：用户要回应的状态更急。
type Status int

const (
	// StatusIdle 没有运行、也没有待回应。
	StatusIdle Status = iota
	// StatusRunning 有一轮在运行。
	StatusRunning
	// StatusWaiting 有审批或提问待决。
	StatusWaiting
)

// StatusOf 取托盘状态：等待审批 > 运行中 > 空闲。
func StatusOf(state State) Status {
	switch {
	case state.WaitingApproval:
		return StatusWaiting
	case state.Running:
		return StatusRunning
	default:
		return StatusIdle
	}
}

// Item 是一个托盘菜单项。Separator 为真时其余字段无意义。
type Item struct {
	Action    Action
	Label     string
	SessionID string
	Checked   bool
	Disabled  bool
	Separator bool
}

// Menu 生成托盘菜单：显示/隐藏窗口 → 新建会话 → 最近会话（前 5）→ 关窗偏好 →
// 退出。分隔线把「动作」与「设置/退出」分开，与参考产品的菜单分组一致。
func Menu(state State) []Item {
	items := []Item{windowItem(state.WindowVisible), {Action: ActionNewSession, Label: "新建会话"}}
	if recent := recentItems(state.Recent); len(recent) > 0 {
		items = append(items, Item{Separator: true}, Item{Label: "最近会话", Disabled: true})
		items = append(items, recent...)
	}
	items = append(items,
		Item{Separator: true},
		Item{Action: ActionToggleCloseToTray, Label: "关闭窗口时最小化到托盘", Checked: state.CloseToTray},
		Item{Separator: true},
		Item{Action: ActionQuit, Label: "退出"},
	)
	return items
}

// windowItem 是第一项：窗口显示时给「隐藏」，隐藏时给「显示」。
func windowItem(visible bool) Item {
	if visible {
		return Item{Action: ActionHideWindow, Label: "隐藏主窗口"}
	}
	return Item{Action: ActionShowWindow, Label: "显示主窗口"}
}

// recentItems 截断到 RecentLimit 条，跳过没有标识的条目（打不开的菜单项不摆）。
func recentItems(recent []RecentSession) []Item {
	out := make([]Item, 0, RecentLimit)
	for _, session := range recent {
		if len(out) == RecentLimit {
			break
		}
		id := strings.TrimSpace(session.ID)
		if id == "" {
			continue
		}
		label := strings.TrimSpace(session.Title)
		if label == "" {
			label = id
		}
		out = append(out, Item{Action: ActionOpenSession, Label: label, SessionID: id})
	}
	return out
}

// Tooltip 是托盘图标的悬停文字：只在有状态时追加标记，空闲时只有产品名。
func Tooltip(status Status) string {
	switch status {
	case StatusRunning:
		return "DimAgent · 运行中"
	case StatusWaiting:
		return "DimAgent · 等待你的回应"
	default:
		return "DimAgent"
	}
}

// Signature 是一份菜单的指纹：入口只在它变化时重建系统菜单，免得每帧重装。
func Signature(items []Item) string {
	var b strings.Builder
	for _, item := range items {
		if item.Separator {
			b.WriteString("-|")
			continue
		}
		b.WriteString(string(item.Action))
		b.WriteByte('|')
		b.WriteString(item.Label)
		b.WriteByte('|')
		b.WriteString(item.SessionID)
		b.WriteByte('|')
		b.WriteString(strconv.FormatBool(item.Checked))
		b.WriteByte('|')
		b.WriteString(strconv.FormatBool(item.Disabled))
		b.WriteByte('\n')
	}
	return b.String()
}
