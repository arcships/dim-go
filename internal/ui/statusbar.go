package ui

import (
	"context"
	"fmt"
	"math"
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// 状态条（参考 StatusBar.vue）：只出现在有会话的工作区，高 30；左边是权限
// 选择器与分支标记，右边是上下文用量表。除这三件外，其余是点开后才有内容
// 的浮层（权限菜单、分支菜单、用量明细）。
const (
	statusBarHeight     = 30
	statusBarPadX       = 17.5
	statusBarTriggerH   = 21
	statusUsageBarWidth = 70
	statusUsageBarH     = 3
	statusUsageMenuW    = 260
	statusBranchMenuW   = 220
	// permissionDialogW 是完全放开确认框的宽（参考 sm:max-w-md = 448）。
	permissionDialogW = 448
)

// statusBar 是会话工作区底部的状态条。
func (a *App) statusBar(c *gui.Context) {
	gui.Row(c).Shrink(0).Padding(0, space4).Justify(gui.Center).Children(func() {
		gui.Row(c).WidthPercent(100).MaxWidth(composerMaxWidth).Height(statusBarHeight).
			Padding(0, statusBarPadX).Gap(space3).AlignItems(gui.Center).
			Label("状态条").Children(func() {
			a.statusPermission(c)
			branch, _, _ := a.branchState()
			if branch != "" && branch != "main" {
				a.statusBranch(c)
			}
			gui.Spacer(c)
			a.statusUsage(c)
		})
	})
	a.fullAccessConfirm(c)
}

// statusPermission 是状态条左侧的权限选择器：盾形图标 + 当前档位 + 菜单。
func (a *App) statusPermission(c *gui.Context) {
	t := c.Theme()
	m := gui.MenuBase(c, &a.permissionOpen)
	tr := m.Trigger.Height(statusBarTriggerH).Padding(0, space1).Radius(radius).
		Gap(space1).AlignItems(gui.Center).Label("权限：" + a.permissionLabel()).
		Tooltip("选择权限").Cursor(gui.CursorPointer)
	if tr.Hovered() {
		tr = tr.Background(t.SurfaceHover)
	}
	tr.Children(func() {
		iconBox(c, iconShield, t.TextMuted, "permission")
		typeMicro.style(t, gui.Text(c, a.permissionLabel())).TextColor(t.TextMuted).FontWeight(500)
	})
	m.Popup(func(panel gui.Element) {
		panel.Width(composerPermissionMenuWidth).Padding(space1).Radius(radiusCard).
			Background(t.Surface).
			Shadow(0, 4, 16, 0, colorShadow.in(t))
		for _, level := range permissionLevels {
			level := level
			if menuItem(c, m, iconShield, level.label, "权限："+level.label, "", false,
				level.label == a.permissionLabel()) {
				// 完全放开是高后果：先确认，不在菜单里一步生效（见 choosePermission）。
				a.choosePermission(level.label)
			}
		}
	})
}

// fullAccessConfirm 是完全放开的确认对话框（SB-04–SB-07）：标题、风险说明、
// 「我已了解风险」勾选与取消/确认两步。未勾选时确认按钮不可用。
func (a *App) fullAccessConfirm(c *gui.Context) {
	if !a.permissionArmOpen {
		return
	}
	t := c.Theme()
	open := true
	m := gui.Modal(c, &open, func() {
		panel := gui.Column(c).Gap(space3).Children(func() {
			gui.Row(c).Gap(space2).AlignItems(gui.Center).Children(func() {
				iconGlyph(c, iconWarn, t.Warning, "warning")
				typeH3.style(t, gui.Text(c, "确认开启完全放开？")).TextColor(t.Text).
					Role(gui.RoleHeading).Level(3)
			})
			typeBody.style(t, gui.Text(c,
				"开启后，AI 将减少确认步骤，可以直接读写文件、执行命令与访问网络。"+
					"请只在你信任当前任务与会话内容时开启。")).TextColor(t.TextMuted)
			gui.Checkbox(c, &a.permissionArmAck, "我已了解风险，并愿意继续").
				Label("我已了解风险，并愿意继续")
			gui.Row(c).Justify(gui.End).Gap(space2).Children(func() {
				if gui.Button(c, "取消").Label("取消").Clicked() {
					a.permissionArmOpen = false
					open = false
				}
				if gui.PrimaryButton(c, "确认开启").Label("确认开启").
					Disabled(!a.permissionArmAck).Clicked() && a.permissionArmAck {
					a.applyPermission("完全访问权限")
					a.permissionArmOpen = false
					open = false
				}
			})
		})
		panel.Width(permissionDialogW)
	})
	if !open || m.Dismissed() {
		a.permissionArmOpen = false
	}
}

// statusBranch 是状态条里的分支标记：分叉图标 + 分支名 + 菜单（搜索、切换、
// 新建）。默认分支 main 不上屏（UI-9）。
func (a *App) statusBranch(c *gui.Context) {
	t := c.Theme()
	current, branches, wired := a.branchState()
	m := gui.MenuBase(c, &a.statusBranchOpen)
	tr := m.Trigger.Height(statusBarTriggerH).Padding(0, space1).Radius(radius).
		Gap(space1).AlignItems(gui.Center).Label("分支 " + current).
		Tooltip("切换分支").Cursor(gui.CursorPointer)
	if tr.Hovered() {
		tr = tr.Background(t.SurfaceHover)
	}
	tr.Children(func() {
		iconGlyph(c, iconBranch, t.TextMuted, "branch")
		typeMicro.style(t, gui.Text(c, current)).TextColor(t.TextMuted).FontWeight(500).SingleLine()
	})
	m.Popup(func(panel gui.Element) {
		panel.Width(statusBranchMenuW).Padding(space1).Radius(radiusCard).
			Background(t.Surface).
			Shadow(0, 4, 16, 0, colorShadow.in(t))
		typeMicro.style(t, gui.Text(c, "分支")).TextColor(t.TextMuted).Padding(space1, space2)
		gui.Row(c).Height(composerSearchRowHeight).Padding(0, space2).Gap(space1).
			Radius(radius).Background(t.SurfaceHover).AlignItems(gui.Center).Children(func() {
			iconBox(c, iconSearch, t.TextMuted, "search")
			gui.TextInputBase(c, &a.statusBranchQuery).Placeholder("搜索分支...").
				Grow(1).Label("搜索分支")
		})
		for _, name := range branches {
			name := name
			if a.statusBranchQuery != "" &&
				!strings.Contains(strings.ToLower(name), strings.ToLower(a.statusBranchQuery)) {
				continue
			}
			if menuItem(c, m, iconBranch, name, "分支："+name, "", false, name == current) {
				if wired && a.intents != nil {
					a.dispatch("切换分支", func(ctx context.Context) error {
						return a.intents.SelectBranch(ctx, name)
					}, func() { a.branch = name })
				} else {
					a.branch = name
				}
			}
		}
		if menuItem(c, m, iconPlus, "创建新分支", "创建新分支", "", false, false) {
			if !wired {
				name := current + "-1"
				if !containsString(a.branches, name) {
					a.branches = append(a.branches, name)
				}
				a.branch = name
			}
		}
	})
}

// statusUsage 是右侧的上下文用量表：一条细进度条 + 百分比；点开是明细。
func (a *App) statusUsage(c *gui.Context) {
	t := c.Theme()
	pct := contextPercent(a.contextUsed, a.contextLimit)
	label := "—"
	if a.contextLimit > 0 {
		label = pctLabel(pct)
	}
	m := gui.MenuBase(c, &a.contextOpen)
	tr := m.Trigger.Height(statusBarTriggerH).Padding(0, space1).Radius(radius).
		Gap(space1).AlignItems(gui.Center).Label(fmt.Sprintf("上下文用量 %s", label)).
		Tooltip("上下文用量").Cursor(gui.CursorPointer)
	if tr.Hovered() {
		tr = tr.Background(t.SurfaceHover)
	}
	tr.Children(func() {
		a.usageBar(c, pct)
		typeMicro.style(t, gui.Text(c, label)).TextColor(t.TextMuted).SingleLine()
	})
	m.Popup(func(panel gui.Element) {
		panel.Width(statusUsageMenuW).Padding(space2).Gap(space1).Radius(radiusCard).
			Background(t.Surface).
			Shadow(0, 4, 18, 0, colorShadow.in(t))
		typeSecondaryStrong.style(t, gui.Text(c, "本轮上下文")).TextColor(t.Text)
		if a.contextLimit > 0 {
			typeSecondary.style(t, gui.Text(c, fmt.Sprintf("~%s / %s",
				formatTokens(a.contextUsed), formatTokens(a.contextLimit)))).TextColor(t.Text)
		} else {
			typeSecondary.style(t, gui.Text(c, "—")).TextColor(t.TextMuted)
		}
		if a.source == nil {
			// 样例读数：接上 Source 后不再出现（视图不掺样例事实）。
			typeMicro.style(t, gui.Text(c, "缓存命中 98%")).TextColor(t.TextMuted)
			typeMicro.style(t, gui.Text(c, "速率：42 tok/s")).TextColor(t.TextMuted)
		}
		gui.Box(c).Height(strokeHairline).Background(t.Border)
		typeMicro.style(t, gui.Text(c, "套餐用量")).TextColor(t.TextMuted)
		if a.source != nil {
			usage := a.source.Usage()
			switch {
			case !usage.Available:
				reason := usage.Reason
				if reason == "" {
					reason = "暂无用量明细"
				}
				typeSecondary.style(t, gui.Text(c, reason)).TextColor(t.TextMuted)
			case usage.Limit > 0:
				p := int(math.Round(usage.Used / usage.Limit * 100))
				typeSecondary.style(t, gui.Text(c, fmt.Sprintf("月度用量 %d%%", p))).TextColor(t.Text)
				a.usageBarSized(c, p, statusUsageMenuW-2*space2)
				if usage.HasResets && !usage.ResetsAt.IsZero() {
					typeMicro.style(t, gui.Text(c, "重置于 "+usage.ResetsAt.Format("1月2日"))).
						TextColor(t.TextMuted)
				}
			default:
				typeSecondary.style(t, gui.Text(c, "无限额度")).TextColor(t.TextMuted)
			}
		} else {
			typeSecondary.style(t, gui.Text(c, "月度用量 39%")).TextColor(t.Text)
		}
		a.usageAccountLink(c)
	})
}

// usageAccountLink 是用量浮层底部的「账户 ↗」：进设置 · 账户，不替换主区内容。
func (a *App) usageAccountLink(c *gui.Context) {
	t := c.Theme()
	btn := gui.ButtonBase(c).Height(controlMini).Padding(0, space1).Radius(radius).
		Gap(space1).AlignItems(gui.Center).Label("账户设置").Cursor(gui.CursorPointer)
	if btn.Hovered() {
		btn = btn.Background(t.SurfaceHover)
	}
	btn.Children(func() {
		typeMicro.style(t, gui.Text(c, "账户")).TextColor(t.TextMuted)
		iconBox(c, iconExternal, t.TextMuted, "account")
	})
	if btn.Clicked() {
		a.screen = ScreenSettings
		a.settingsSection = secAccount
	}
}

// usageBar 是状态条里的上下文用量条：底轨 + 已用比例（将满时换警示色）。
func (a *App) usageBar(c *gui.Context, pct int) {
	a.usageBarSized(c, pct, statusUsageBarWidth)
}

// usageBarSized 是任意宽度的用量条（浮层里的套餐用量也用它）。
func (a *App) usageBarSized(c *gui.Context, pct int, width float32) {
	t := c.Theme()
	box := gui.Box(c).Width(width).Height(statusUsageBarH).Shrink(0).
		Radius(statusUsageBarH / 2).Background(t.Border).Label("用量条")
	fill := t.Success
	if pct >= contextNearPercent {
		fill = t.Warning
	}
	box.DrawOver(func(p *gui.Painter, r gui.Rect) {
		if pct <= 0 {
			return
		}
		w := r.W * float32(pct) / 100
		if w > r.W {
			w = r.W
		}
		p.Fill(gui.Rect{X: r.X, Y: r.Y, W: w, H: r.H}, fill, r.H/2)
	})
}

// contextPercent 把已用/上限换成整数百分比；上限未知时给 0。
func contextPercent(used, limit int) int {
	if limit <= 0 || used <= 0 {
		return 0
	}
	p := used * 100 / limit
	if p > 100 {
		p = 100
	}
	return p
}

// pctLabel 是状态条上的百分比写法：未知上限给「—」，未知总量给「~」。
func pctLabel(pct int) string {
	return fmt.Sprintf("~%d%%", pct)
}

// formatTokens 把 token 数写成 K/M 的短形式。
func formatTokens(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprint(n)
	}
}

// containsString 判断清单里是否已有该名字。
func containsString(list []string, want string) bool {
	for _, item := range list {
		if item == want {
			return true
		}
	}
	return false
}

// statusBarVisible 判断是否该画状态条：有会话（样例态或 Source 已打开会话）。
func (a *App) statusBarVisible() bool {
	if a.source != nil {
		_, ok := a.liveSession()
		return ok
	}
	return a.state != StateEmpty
}
