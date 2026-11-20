package ui

import (
	"fmt"
	"strings"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件补输入区此前缺的元素：
//
//   - 活动条：待办 / 子代理 / 分支 / 唤醒 / 进程 / 文件改动六个 chip（IN-29–IN-34）；
//   - `+` 附件菜单：添加附件、Agent / 目标子菜单、技能（IN-17–IN-22）；
//   - 模型浮层的四种状态与子代理页（IN-53 / IN-54 / IN-56）；
//   - ZDR / priority 标记（IN-10 / IN-11）；
//   - 停止确认的 Esc 键帽（IN-14）与发送不可用原因行（IN-15 / IN-16）。
//
// 事实口径：分文、任务、唤醒、文件改动、服务档与模型目录都读快照；待办与子代理
// 没有独立清单，只从工具行认（认不出就不画 chip，原因见 unavailable.go）。

// ---- 活动条（IN-29 – IN-34） ----

// activityChipFacts 是活动条要画的事实：每一项只在有事实时出现。
type activityChipFacts struct {
	todo        bool
	subagent    bool
	branches    int
	wakes       int
	processes   int
	fileChanges int
}

// activityFacts 收集活动条的事实。待办与子代理没有清单通道，只能从工具行认；
// 分支、进程来自会话快照；唤醒来自定时任务读数；文件改动来自最新一轮的改动块。
func (a *App) activityFacts(snap SessionSnapshot) activityChipFacts {
	var facts activityChipFacts
	// 工具行可能在工具组里：待办与子代理两种事实都往下看一层。
	var scan func(row RowSnapshot)
	scan = func(row RowSnapshot) {
		if row.Kind == RowTool {
			name := strings.ToLower(row.Title)
			if strings.Contains(name, "todo") || strings.Contains(name, "待办") {
				facts.todo = true
			}
			if subagentTool(row.Title) && (row.Status == RowStatusRunning || row.Status == RowStatusPending) {
				facts.subagent = true
			}
		}
		if row.Kind == RowFileChanges && len(row.Children) > 0 {
			facts.fileChanges = len(row.Children)
		}
		for _, child := range row.Children {
			scan(child)
		}
	}
	for _, row := range snap.Rows {
		scan(row)
	}
	for _, branch := range snap.Branches {
		if !branch.Ended {
			facts.branches++
		}
	}
	if a.source != nil {
		for _, item := range a.source.Schedules().Items {
			if item.Enabled || item.Running {
				facts.wakes++
			}
		}
	}
	facts.processes = len(snap.Tasks)
	if facts.fileChanges == 0 && len(a.rounds) > 0 {
		// 样例模式没有行事实：最新一轮的改动块就是它的文件改动事实。
		facts.fileChanges = len(a.rounds[len(a.rounds)-1].fileChanges)
	}
	return facts
}

// activityBar 是输入区上沿的活动条：只列有事实的 chip，没有事实就不占位置。
// TL/IN 的六项挨着排，点击各自去到能看明细的地方。
// IN-27 底部栈里的活动条（IN-29 – IN-34）
func (a *App) activityBar(c *gui.Context, snap SessionSnapshot) {
	facts := a.activityFacts(snap)
	type chip struct {
		icon  *gui.SVG
		text  string
		tip   string
		open  func()
		count int
	}
	chips := make([]chip, 0, 6)
	if facts.todo {
		// IN-29 待办活动 chip
		chips = append(chips, chip{icon: iconList, text: "待办", tip: noteTodoFacts})
	}
	if facts.subagent {
		// IN-30 子代理活动 chip
		chips = append(chips, chip{icon: iconBot, text: "子代理", tip: noteSubagentActivity,
			open: func() {
				a.rightOpen = true
				a.rpOpen(tabSubagent)
			}})
	}
	if facts.branches > 0 {
		// IN-31 分支活动 chip
		chips = append(chips, chip{icon: iconBranch, text: fmt.Sprintf("分支 %d", facts.branches),
			tip: "当前分支：" + snap.Branch})
	}
	if facts.wakes > 0 {
		// IN-32 唤醒活动 chip
		chips = append(chips, chip{icon: iconClock, text: fmt.Sprintf("唤醒 %d", facts.wakes),
			tip: "定时唤醒"})
	}
	if facts.processes > 0 {
		// IN-33 后台进程活动 chip
		chips = append(chips, chip{icon: iconTerminal, text: fmt.Sprintf("进程 %d", facts.processes),
			tip: "后台任务",
			open: func() {
				a.rightOpen = true
				a.rpOpen(tabTasks)
			}})
	}
	if facts.fileChanges > 0 {
		// IN-34 文件改动活动 chip
		chips = append(chips, chip{icon: iconDiff, text: fmt.Sprintf("%d 个文件改动", facts.fileChanges),
			tip: "本轮文件改动",
			open: func() {
				a.rightOpen = true
				a.rpOpen(tabChanges)
			}})
	}
	if len(chips) == 0 {
		return
	}
	gui.Row(c).Height(QueueHeight).Shrink(0).Padding(0, space4).Gap(space2).
		AlignItems(gui.Center).Label("活动条").Children(func() {
		for _, item := range chips {
			item := item
			btn := gui.ButtonBase(c).Shrink(0).Height(mentionChipH+3).Gap(space1).
				Padding(0, space2).Radius(radiusSmall).Background(c.Theme().SurfaceHover).
				AlignItems(gui.Center).Cursor(gui.CursorPointer).
				Label("活动条：" + item.text).Tooltip(item.tip)
			btn.Children(func() {
				iconGlyph(c, item.icon, c.Theme().TextMuted, "activity")
				typeMicro.style(c.Theme(), gui.Text(c, item.text)).TextColor(c.Theme().Text)
			})
			if btn.Clicked() && item.open != nil {
				item.open()
			}
		}
	})
}

// ---- `+` 附件菜单（IN-17 – IN-22） ----

// attachMenu 是工具行首个 `+` 的附件菜单：添加附件、Agent / 目标子菜单、技能。
// IN-22 紧凑态 `+` 与 IN-17 … IN-21 的六个入口。
func (a *App) attachMenu(c *gui.Context) {
	t := c.Theme()
	m := gui.MenuBase(c, &a.attachOpen)
	// IN-22 紧凑态 `+` 图标按钮
	tr := m.Trigger.Width(composerToolSize).Height(composerToolSize).Radius(radius).
		Label("附件").Tooltip("添加附件").Cursor(gui.CursorPointer)
	if tr.Hovered() {
		tr = tr.Background(t.SurfaceHover)
	}
	tr.Children(func() { iconBox(c, iconPlus, t.TextMuted, "plus") })
	m.Popup(func(panel gui.Element) {
		panel.Width(composerMenuWidth).Padding(space1).Radius(radiusCard).Background(t.Surface).
			Border(strokeHairline, t.Border).Shadow(0, 4, 16, 0, colorShadow.in(t))
		// IN-17 添加附件
		if menuItem(c, m, iconPaperclip, "添加附件", "添加附件", "", false, false) {
			a.attachmentEditing = true
		}
		// IN-18 Agent / 目标子菜单容器
		sub := m.SubmenuItem().MinHeight(rowRegular).Padding(space1, space2).
			Radius(radius).Gap(space2).AlignItems(gui.Center).Label("Agent / 目标")
		if sub.Highlighted() {
			sub = sub.Background(t.SurfaceHover)
		}
		sub.Children(func() {
			iconBox(c, iconChevronsR, t.TextMuted, "mode")
			typeSecondary.style(t, gui.Text(c, "Agent / 目标")).TextColor(t.Text).Grow(1).SingleLine()
			iconBox(c, iconChevronR, t.TextMuted, "submenu")
		})
		m.Submenu(sub, func(panel gui.Element) {
			panel.Width(composerModeMenuWidth).Padding(space1).Radius(radiusCard).
				Background(t.Surface).Border(strokeHairline, t.Border).
				Shadow(0, 4, 16, 0, colorShadow.in(t))
			// IN-19 Agent
			if menuItem(c, m, iconChevronsR, "Agent", "Agent 模式", "", false, a.mode == "Agent") {
				a.switchMode("Agent")
			}
			// IN-20 目标
			_, hasSession := a.liveSession()
			goalReason := ""
			if a.wired() && !hasSession {
				goalReason = noteNoSessionGoal
			}
			if menuItem(c, m, iconTarget, "目标", "目标模式", goalReason,
				goalReason != "", a.mode == "Goal") {
				a.switchMode("Goal")
			}
			if goalReason != "" {
				typeMicro.style(t, gui.Text(c, goalReason)).
					TextColor(t.TextMuted).Padding(space1, space2)
			}
		})
		// IN-21 技能
		if menuItem(c, m, iconCube, "技能", "技能", "", false, a.skillsOpen) {
			a.skillsOpen = true
		}
	})
}

// ---- 模型浮层状态、页签与标记（IN-10 / IN-11 / IN-54 / IN-56） ----

// modelFacts 是模型浮层要用的读数：目录是否读到、失败原因、服务档与候选。
type modelFacts struct {
	options   []ModelOption
	currentID string
	known     bool
	readErr   string
	providers int
	tier      string
}

// modelOverlayFacts 读模型浮层的读数：连着 Source 时只读快照，样例里用静态清单
// （样例没有「加载中 / 失败」这两种状态）。
func (a *App) modelOverlayFacts() modelFacts {
	if a.source == nil {
		options, currentID := a.modelOptions()
		return modelFacts{options: options, currentID: currentID, known: true}
	}
	settings := a.source.Settings()
	return modelFacts{
		options: settings.Models, currentID: settings.ModelID,
		known: settings.ModelsKnown, readErr: settings.ModelsError,
		providers: len(settings.Providers), tier: settings.ServiceTier,
	}
}

// modelEmptyState 是模型列表的四态之一：还没读到、读失败、未配置、读到空。
// 返回空串表示有可列出的模型。
// IN-56 模型浮层的四种状态文案
func modelEmptyState(facts modelFacts, wired bool) string {
	if !wired {
		return ""
	}
	switch {
	case facts.readErr != "":
		return "加载失败"
	case !facts.known:
		return "模型加载中…"
	case len(facts.options) == 0 && facts.providers == 0:
		return "未配置"
	case len(facts.options) == 0:
		return "暂无可用模型"
	}
	return ""
}

// modelTabs 是浮层顶部的三个页签：对话、子代理、外部 Agent。页签只在有模型
// 目录时显示；点「子代理」把浮层切到子代理页，页内写清没有子代理策略通道。
// IN-54 浮层页签（模型 / 子代理 / 外部 agent）
func (a *App) modelTabs(c *gui.Context, m gui.MenuParts) {
	t := c.Theme()
	tabs := []struct {
		index int
		text  string
	}{
		{0, "模型"}, {1, "子代理"}, {2, "外部 agent"},
	}
	gui.Row(c).Gap(space3).AlignItems(gui.Center).Children(func() {
		for _, tab := range tabs {
			tab := tab
			color := t.TextMuted
			if a.modelTab == tab.index {
				color = t.Text
			}
			item := gui.ButtonBase(c).Shrink(0).Height(rowCompact).Padding(0, 0).
				Radius(radiusSmall).Cursor(gui.CursorPointer).Label(tab.text)
			item.Children(func() {
				style := typeSecondary
				if a.modelTab == tab.index {
					style = typeSecondaryStrong
				}
				style.style(t, gui.Text(c, tab.text)).TextColor(color)
			})
			if item.Clicked() {
				a.modelTab = tab.index
			}
		}
	})
}

// modelTabBody 是页签下的正文：模型页列候选，子代理页与外部 Agent 页写清没有
// 对应通道（不拿模型列表顶替）。
func (a *App) modelTabBody(c *gui.Context, m gui.MenuParts, facts modelFacts) {
	t := c.Theme()
	switch a.modelTab {
	case 1:
		// IN-54 子代理页：没有子代理策略事实，写原因。
		typeMicro.style(t, gui.Text(c, noteSubagentFacts)).TextColor(t.TextMuted).
			Padding(space1, space2)
		return
	case 2:
		// IN-54 外部 Agent 页：App Server 模式下不可用（与设置屏同一句）。
		typeMicro.style(t, gui.Text(c, "App Server 尚未实现外部代理。此模式下 Desktop 无法检测或配置它们。")).
			TextColor(t.TextMuted).Padding(space1, space2)
		return
	}
	if state := modelEmptyState(facts, a.wired()); state != "" {
		typeMicro.style(t, gui.Text(c, state)).TextColor(t.TextMuted).Padding(space1, space2)
		if state == "加载失败" {
			gui.Row(c).Gap(space1).AlignItems(gui.Center).Children(func() {
				btn := gui.ButtonBase(c).Shrink(0).Height(rowCompact).Padding(0, space2).
					Radius(radiusSmall).Label("重试").Tooltip(noteModelsRetry).
					Disabled(true).Opacity(noteDisabledAlpha)
				btn.Children(func() {
					typeMicro.style(t, gui.Text(c, "重试")).TextColor(t.TextMuted)
				})
			})
			typeMicro.style(t, gui.Text(c, noteModelsRetry)).TextColor(t.TextMuted).
				Padding(space1, space2)
		}
		return
	}
	options := facts.options
	currentID := facts.currentID
	gui.Row(c).Gap(space2).AlignItems(gui.Start).Children(func() {
		// 最近使用：服务端没有「最近用过哪些模型」的事实，能解析出来的只有当前
		// 选择，所以这一栏最多一项；没有当前选择时整栏不画。
		if recent, ok := a.recentModel(options, currentID); ok {
			gui.Column(c).Width(composerRecentWidth).Gap(space1).Children(func() {
				gui.Row(c).Gap(space1).AlignItems(gui.Center).Children(func() {
					iconBox(c, iconClock, t.TextMuted, "recent")
					typeMicro.style(t, gui.Text(c, "最近使用")).TextColor(t.TextMuted)
				})
				if a.modelOpen {
					if menuItem(c, m, iconProvider, recent.Name, "最近使用："+recent.Name, "", false, true) {
						a.selectModel(recent)
					}
				}
			})
		}
		gui.Column(c).Grow(1).Gap(space1).Children(func() {
			for _, option := range options {
				option := option
				if !strings.Contains(strings.ToLower(option.Name), strings.ToLower(a.modelQuery)) {
					continue
				}
				if menuItem(c, m, nil, option.Name, "模型："+option.Name, "", false, option.ID == currentID) {
					a.selectModel(option)
				}
			}
		})
	})
}

// modelFactorIcons 是模型槽上的两个标记：服务档 priority 画 Zap；ZDR 没有事实，
// 只在浮层底部写明本版不显示。
// IN-11 priority 服务档图标
func (a *App) modelFactorIcons(c *gui.Context, facts modelFacts) {
	if !strings.EqualFold(strings.TrimSpace(facts.tier), "priority") {
		return
	}
	iconGlyph(c, iconZap, colorAccent.in(c.Theme()), "priority")
}

// zdrNote 是模型浮层底部的 ZDR 说明：没有该事实时不装一个假的盾牌图标。
// IN-10 ZDR 徽标位（缺事实，写原因）
func (a *App) zdrNote(c *gui.Context) {
	if !a.wired() {
		return
	}
	typeMicro.style(c.Theme(), gui.Text(c, noteZDR)).TextColor(c.Theme().TextMuted).
		Padding(space1, space2)
}

// ---- 停止确认的 Esc 键帽（IN-14）与发送原因行（IN-15 / IN-16） ----

// stopConfirmWindow 是 Esc 两段确认的窗口：首次 Esc 武装，3 秒内第二次 Esc
// 才停止，超时撤除（IN-14，参考 STOP_CONFIRMATION_TIMEOUT_MS = 3000）。
const stopConfirmWindow = 3 * time.Second

// escKeycap 是 Esc 武装后出现的键帽：键帽只说明「再按一次停止」，不显示倒计时。
// IN-14 Esc 键帽
func (a *App) escKeycap(c *gui.Context) {
	t := c.Theme()
	// 出现与撤除都是直接换，不做淡入淡出、不显示倒计时（R-IN-195）。
	gui.Row(c).Shrink(0).Height(rowCompact).Padding(0, space1).Gap(space1).
		Radius(radiusSmall).Background(t.SurfacePressed).Border(strokeHairline, t.Border).
		AlignItems(gui.Center).Label("Esc 键：再按一次停止").Tooltip(noteStopArm).Children(func() {
		typeMicro.style(t, gui.Text(c, "Esc")).TextColor(t.TextMuted)
	})
}

// stopKeys 是 Esc 的两段确认：首次武装、3 秒内第二次停止、超时撤除；武装中
// 点停止键由 sendSlot 一点即停，与这条路径互不影响（R-IN-163 / R-IN-191）。
func (a *App) stopKeys(c *gui.Context) {
	if a.stopArmed && c.Now().Sub(a.stopArmedAt) > stopConfirmWindow {
		a.stopArmed, a.stopArmedAt = false, time.Time{}
	}
	if !a.liveRunning() {
		return
	}
	if !c.Shortcut(0, gui.KeyEscape) {
		return
	}
	if a.stopArmed && c.Now().Sub(a.stopArmedAt) <= stopConfirmWindow {
		a.stopArmed, a.stopArmedAt = false, time.Time{}
		if a.intents != nil {
			a.dispatch("停止", a.intents.Stop, nil)
		}
		return
	}
	a.stopArmed, a.stopArmedAt = true, c.Now()
	// 3 秒后要一帧把武装态撤掉：不靠用户下一次输入才更新。
	c.After(stopConfirmWindow)
}

// sendUnavailableReason 是发送不可用且不可排队时的原因；空表示没有要写的。
// 发送槽在草稿为空时的禁用是常态，不写原因。
func (a *App) sendUnavailableReason() string {
	if a.liveRunning() {
		if a.wired() && a.intents == nil {
			return noteNoWriteChannel
		}
		return ""
	}
	if a.wired() && a.intents == nil && strings.TrimSpace(a.draft) != "" {
		return noteSendUnknown
	}
	return ""
}

// sendReasonRow 是发送不可用原因行：原因 + 一个不可用的「重试」（没有重试通道，
// 原因写在 tooltip 与读屏名里）。
// IN-15 发送不可用原因 / IN-16 重试
func (a *App) sendReasonRow(c *gui.Context) {
	reason := a.sendUnavailableReason()
	if reason == "" {
		return
	}
	t := c.Theme()
	gui.Row(c).Grow(1).Gap(space2).AlignItems(gui.Center).Label("发送不可用：" + reason).
		Children(func() {
			iconGlyph(c, iconInfo, t.TextMuted, "send-reason")
			typeMicro.style(t, gui.Text(c, reason)).TextColor(t.TextMuted).Grow(1).SingleLine()
			btn := gui.ButtonBase(c).Shrink(0).Height(rowCompact).Padding(0, space2).
				Radius(radiusSmall).Label("重试发送").Tooltip(noteNoWriteChannel).
				Disabled(true).Opacity(noteDisabledAlpha)
			btn.Children(func() {
				typeMicro.style(t, gui.Text(c, "重试")).TextColor(t.TextMuted)
			})
		})
}
