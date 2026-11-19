package ui

import (
	"context"
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// 右栏（R5）：默认收起；展开后是「标签栏 + 分隔线 + 当前面板」。
//
// 七个标签对齐参考产品的右栏面板：文件 / 改动 / 后台（后台进程与唤醒）/
// 目标 / 终端 / 侧聊 / 子代理。文件、改动、目标三面仍是既有实现；后四面按
// docs/domains/requirements 的 inventory-panels-rt 与 requirements-panels 重建。
//
// 面板内的元素只读事实、不写事实：Go 侧只有可丢的视图状态（选中的行、草稿、
// 确认框的开合），事实一律来自 Source；没有 Source 时用样例。

// 右栏标签的下标（与右栏标签条的类型表同序；app.go 里的变体也用这些值）。
const (
	tabFiles = iota
	tabChanges
	tabTasks
	tabGoal
	tabTerminal
	tabSideChat
	tabSubagent
)

// 面板尺寸：参考产品的实测值与需求条目给出的具体值。
const (
	// rightPanelTabHeight 是标签栏高：参考实测约 40（design-baseline §7）。
	rightPanelTabHeight = 40
	// panelRowHeight 是面板列表行高：参考 44–48（R-TK-033），取 44。
	panelRowHeight = 44
	// panelPad 是面板内容内边距：时间线内边距档 10.5（design-baseline §1.3）。
	panelPad = space3
	// panelGap 是面板内的块间距。
	panelGap = space2
	// panelCenterPad 是空态文本的左右留白（R-SC-052：24）。
	panelCenterPad = 24
	// panelBannerPad 是错误横幅的内边距。
	panelBannerPad = space2
	// panelInputHeight 是侧聊输入框高（R-SC-084：行内 25）。
	panelInputHeight = 25
	// panelSendSize 是侧聊发送按钮的直径（R-SC-113：32）。
	panelSendSize = 32
	// panelOutputMinHeight 是进程详情里只读输出框的最小高（K-30：160）。
	panelOutputMinHeight = 160
)

// panelView 是四个重建面板的视图状态：可丢、不承载事实。
type panelView struct {
	// 终端：进程是否已关闭（提示行「［进程已关闭］」）。
	termClosed bool
	// 任务：子标签（0 后台进程 / 1 唤醒）、选中的行、待确认停止的行。
	// 选中的行用下标；-1 表示没有选中（详情区随之不渲染）。
	taskKind  int
	taskSel   int
	taskArmed int
	// 侧聊：草稿、分支是否已结束、结束确认框、发送中、错误行、是否由某条
	// 消息之前发起（决定用哪条空态文案）、本地发出的消息。
	sideChatDraft       string
	sideChatEnded       bool
	sideChatConfirm     bool
	sideChatSending     bool
	sideChatError       string
	sideChatFromMessage bool
	sideChatSent        []string
	// 子代理：错误原文与只读时间线（本地收到的消息）。没有"加载 / 就绪"这一对
	// 布尔：协议没有子代理会话事实，界面不拿一个没有写入点的开关冒充加载中。
	subagentError string
	subagentLines []string
	// subagentFailed 为真表示读取失败但错误原文可能为空（G-02）：界面据此仍画
	// 错误行与重试，文案换成通用的「加载失败」，不留一条空横幅。
	subagentFailed bool
}

// newPanelView 是面板视图状态的默认值：没有选中行（详情区不渲染）。
func newPanelView() panelView {
	return panelView{taskSel: -1, taskArmed: -1}
}

// rightPanel 画右栏：标签栏、分隔线与当前面板。
func (a *App) rightPanel(c *gui.Context, width float32) {
	t := c.Theme()
	// 与主区之间靠面差（surface 对 background）分开，不画竖线（UI-18 第一档）。
	gui.Row(c).Width(width).Shrink(0).FillHeight().
		Background(t.Surface).
		Label("右栏").Role(gui.RoleGroup).Children(func() {
		a.rpHandle(c)
		gui.Column(c).Grow(1).FillHeight().Children(func() {
			a.rpTabBar(c)
			// 块之间的分隔用框架 Divider：方向随容器，颜色取 border（U2 校准 C8）。
			gui.Divider(c)
			a.rightPanelBody(c)
		})
	})
}

// rightPanelBody 按当前标签画面板：文件与改动见 rightpanel_files.go /
// rightpanel_changes.go，目标与后四面见本文件的 goalPanel、rightpanel_tasks.go、
// terminal.go、rightpanel_sidechat.go 与 rightpanel_subagent.go。
func (a *App) rightPanelBody(c *gui.Context) {
	switch a.rightTab {
	case tabFiles:
		a.rpFilesPage(c)
	case tabChanges:
		a.rpChangesPage(c)
	case tabTasks:
		a.tasksPanel(c)
	case tabGoal:
		a.goalPanel(c)
	case tabTerminal:
		a.terminalPane(c)
	case tabSideChat:
		a.sideChatPanel(c)
	case tabSubagent:
		a.subagentPanel(c)
	case tabFilePreview:
		a.rpPreviewPage(c)
	default:
		// 参考产品里浏览器 / 工作流 / 唤醒 / 扩展 / MCP App 由各自的
		// 功能承载；本产品还没有对应能力，页内容留空。
		gui.Box(c).Grow(1)
	}
}

// goalPanel 是目标面板的既有形态（本轮不重建）。
func (a *App) goalPanel(c *gui.Context) {
	t := c.Theme()
	gui.Scroll(c).Grow(1).Children(func() {
		gui.Column(c).Padding(panelPad).Gap(panelGap).Children(func() {
			typeBodyStrong.style(t, gui.Text(c, "把发送路径改成排队"))
			typeSecondary.style(t, gui.Text(c, "运行中的提交先入队，结束后自动发出。")).
				TextColor(t.TextMuted)
		})
	})
}

// panelEmpty 是面板空态：居中、次要色、可换行（K-18/K-19、S-07/S-08）。
func panelEmpty(c *gui.Context, text string) {
	t := c.Theme()
	gui.Column(c).Grow(1).Center().Padding(space4, panelCenterPad).Children(func() {
		typeBody.style(t, gui.Text(c, text)).TextColor(t.TextMuted).TextAlign(gui.Center)
	})
}

// panelBanner 是错误横幅（S-01、G-01）：警示色浅底 + 1px 描边 + 错误原文 +
// 可选的行尾文字按钮。横幅只有一条：多条错误由调用方替换内容。
func panelBanner(c *gui.Context, message, action string, onAction func()) {
	t := c.Theme()
	row := gui.Row(c).Shrink(0).Padding(panelBannerPad).Gap(space2).
		Radius(radius).AlignItems(gui.Center).Role(gui.RoleAlert).
		Background(t.Warning.Alpha(0.12)).Border(strokeHairline, t.Border)
	row.Children(func() {
		iconGlyph(c, iconWarn, t.Warning, "warning")
		typeBody.style(t, gui.Text(c, message)).TextColor(t.Warning).Grow(1).
			Selectable().MaxLines(4)
		if action != "" {
			btn := gui.ButtonBase(c).Height(rowCompact).Padding(space1, space2).
				Radius(radiusSmall).Label(action)
			if btn.Hovered() {
				btn = btn.Background(t.SurfaceHover)
			}
			if btn.Clicked() && onAction != nil {
				onAction()
			}
			btn.Children(func() {
				typeSecondaryStrong.style(t, gui.Text(c, action)).TextColor(t.Text)
			})
		}
	})
}

// panelCommandText 拼命令行：`$ {command}`。命令为空时返回空串（该行不渲染）。
func panelCommandText(command string) string {
	command = strings.TrimSpace(command)
	if command == "" {
		return ""
	}
	return "$ " + command
}

// panelMessage 是只读时间线里的一条消息（侧聊与子代理共用）：用户消息靠右、
// 带底色，助手正文靠左、无底色；都只能选中复制，没有编辑与引用入口。
func panelMessage(c *gui.Context, text string, user bool) {
	t := c.Theme()
	if user {
		gui.Column(c).Shrink(0).AlignItems(gui.End).Children(func() {
			gui.Column(c).Shrink(0).Padding(space2, space3).
				Radius(radiusCard).Background(t.SurfaceHover).Children(func() {
				typeBody.style(t, gui.Text(c, text)).TextColor(t.Text).
					Selectable().MaxLines(8)
			})
		})
		return
	}
	gui.Column(c).Shrink(0).Children(func() {
		typeBody.style(t, gui.Text(c, text)).TextColor(t.Text).
			Selectable().MaxLines(12)
	})
}

// intentStopTask 停止一条后台任务（连着 Source 时才发；样例里只改视图）。
func (a *App) intentStopTask(c *gui.Context, id string, live bool) {
	if live && a.intents != nil && id != "" {
		a.dispatch("停止任务", func(ctx context.Context) error {
			return a.intents.StopExecTask(ctx, id)
		}, nil)
	}
}
