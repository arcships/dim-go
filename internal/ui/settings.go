package ui

import (
	"context"
	"fmt"
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// tierChoices 是界面上的权限档位选项。
func tierChoices() []string { return []string{"只读", "工作区内自动", "全自动"} }

// tierFromLabel 把界面语换回服务端取值；认不出的返回空（适配器会拒绝）。
func tierFromLabel(label string) string {
	switch label {
	case "只读":
		return "read-only"
	case "工作区内自动":
		return "workspace-write"
	case "全自动":
		return "full-access"
	}
	return ""
}

// tierLabel 把服务端的权限档位取值换成界面语；认不出的原样显示。
func tierLabel(tier string) string {
	switch tier {
	case "full-access":
		return "全自动"
	case "workspace-write":
		return "工作区内自动"
	case "read-only":
		return "只读"
	}
	return dashIfEmpty(tier)
}

// settings 是 P2 设置屏（独立成屏，U1-ia §4）：左导航 224 + 内容区（上限 1008）。
// 本片只做三个分节——外观、供应商、命令规则；其余分节在后续批次追加节点。
//
// 顶部让位与标题栏一致：窗口控件（红绿灯）落在导航列上方的那条里（UI-19），
// 导航列顶部留出 c.TitleBar().Height，空白区可拖动窗口。
func (a *App) settings(c *gui.Context) {
	t := c.Theme()
	gui.Row(c).Fill().AlignItems(gui.Stretch).
		Background(t.Background).Label("设置").Children(func() {
		a.settingsNav(c)
		gui.Column(c).Grow(1).MinWidth(MainMinWidth).Children(func() {
			gui.Column(c).Grow(1).MaxWidth(SettingsContentMaxWidth).
				Padding(space8, space8).Gap(space4).Children(func() {
				a.actionErrorView(c)
				switch a.settingsSection {
				case secAccount:
					a.settingsAccount(c)
				case secProviders:
					a.settingsProviders(c)
				case secRules:
					a.settingsRules(c)
				case secExtensions:
					a.settingsExtensions(c)
				case secCron:
					a.settingsCron(c)
				case secRuntime:
					a.settingsRuntime(c)
				case secStorage:
					a.settingsStorage(c)
				default:
					a.settingsAppearanceKeys(c)
				}
			})
		})
	})
}

// settingsNav 是左导航：返回主界面的图标入口 + 分节列表。
func (a *App) settingsNav(c *gui.Context) {
	t := c.Theme()
	sections := []string{"账号与套餐", "模型与供应商", "命令规则", "扩展", "定时任务", "运行时与模态", "存储与诊断", "外观与键位"}
	gui.Column(c).Width(SettingsNavWidth).Shrink(0).
		Background(t.Surface).Children(func() {
		if inset := titleBarInset(c.TitleBar()); inset > 0 {
			gui.Row(c).Height(inset).Shrink(0).DragWindow().Label("窗口控件让位")
		}
		gui.Row(c).Height(sidebarHeaderHeight).Shrink(0).
			Padding(space1, space2).AlignItems(gui.Center).Gap(space1).Children(func() {
			if iconButton(c, iconButtonSpec{icon: iconChevronL, label: "返回主界面", size: rowCompact}) {
				a.screen = ScreenMain
			}
			gui.Spacer(c)
		})
		for i, name := range sections {
			a.navRow(c, i, name)
		}
		gui.Spacer(c)
	})
}

// navRow 是导航里的一节：自绘行（无外观 ButtonBase），选中态由底色与字重表达。
func (a *App) navRow(c *gui.Context, index int, name string) {
	t := c.Theme()
	row := gui.ButtonBase(c).Padding(space1, space3).Radius(radius).
		Label(name).Cursor(gui.CursorPointer).Justify(gui.Start)
	selected := a.settingsSection == index
	switch {
	case selected:
		row = row.Background(t.SurfacePressed)
	case row.Hovered():
		row = row.Background(t.SurfaceHover)
	}
	if row.Clicked() {
		a.settingsSection = index
	}
	row.Children(func() {
		style := typeControl
		if selected {
			style = typeControlStrong
		}
		style.style(t, gui.Text(c, name)).TextColor(t.Text)
	})
}

// settingsRow 是一行设置项：左侧标签（界面语）+ 右侧控件。标签列宽固定，
// 让同一分节内的控件左缘对齐（UI-22 的"同类元素同一条左缘"）。
func settingsRow(c *gui.Context, label string, control func()) {
	t := c.Theme()
	gui.Row(c).Gap(space4).AlignItems(gui.Center).Shrink(0).Children(func() {
		typeControl.style(t, gui.Text(c, label)).Width(SettingsLabelWidth).TextColor(t.Text)
		gui.Row(c).Grow(1).AlignItems(gui.Center).Children(control)
	})
}

// settingsSectionTitle 是分节标题：标题层级由字重与字号给（U2 §1）。
func settingsSectionTitle(c *gui.Context, title string) {
	t := c.Theme()
	typeH3.style(t, gui.Text(c, title)).TextColor(t.Text)
}

// settingsAppearance 是「外观」：主题与动效两项。
func (a *App) settingsAppearance(c *gui.Context) {
	settingsSectionTitle(c, "账号与套餐")
	settingsRow(c, "主题", func() {
		gui.Select(c, &a.themePref, []string{"跟随系统", "明", "暗"}).Label("主题选择")
	})
	settingsRow(c, "动效", func() {
		gui.Switch(c, &a.motionOff).Label("关闭动效")
	})
	a.persistPreferences()
}

// settingsProviders 是「供应商」：当前供应商与模型，下面是目录信息（内容）。
func (a *App) settingsProviders(c *gui.Context) {
	t := c.Theme()
	settingsSectionTitle(c, "供应商")
	if st, ok := a.liveSettings(); ok {
		providerNames := make([]string, 0, len(st.Providers))
		for _, provider := range st.Providers {
			providerNames = append(providerNames, provider.Name)
		}
		gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Children(func() {
			typeControl.style(t, gui.Text(c, "当前供应商")).Width(SettingsLabelWidth).TextColor(t.Text)
			for _, provider := range st.Providers {
				provider := provider
				primary := provider.Name == st.Provider
				if actionButton(c, provider.Name, "切换到 "+provider.Name, primary, provider.Name == st.Provider).Clicked() && a.intents != nil {
					models := modelsForProvider(st.Models, provider.ID)
					if len(models) > 0 {
						model := models[0].ID
						a.dispatch("切换供应商", func(ctx context.Context) error {
							return a.intents.SelectModel(ctx, provider.ID, model)
						}, nil)
					}
				}
			}
			gui.Spacer(c)
		})
		gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Children(func() {
			typeControl.style(t, gui.Text(c, "模型")).Width(SettingsLabelWidth).TextColor(t.Text)
			for _, model := range modelsForProvider(st.Models, st.ProviderID) {
				model := model
				primary := model.Name == st.Model
				if actionButton(c, model.Name, "选择模型 "+model.Name, primary, model.Name == st.Model).Clicked() && a.intents != nil {
					a.dispatch("切换模型", func(ctx context.Context) error {
						return a.intents.SelectModel(ctx, model.ProviderID, model.ID)
					}, nil)
				}
			}
			gui.Spacer(c)
		})
		if !st.TierKnown {
			settingsValueRow(c, "权限档位", "权限档位", "—")
			return
		}
		current := tierLabel(st.Tier)
		if a.tierDraft == "" {
			a.tierDraft = current
		}
		gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Children(func() {
			typeControl.style(t, gui.Text(c, "权限档位")).Width(SettingsLabelWidth).TextColor(t.Text)
			gui.Select(c, &a.tierDraft, tierChoices()).Label("权限档位选择")
			gui.Spacer(c)
		})
		if a.tierDraft != current {
			gui.Row(c).Gap(space1).AlignItems(gui.Center).Shrink(0).
				Label("权限档位确认").Children(func() {
				iconGlyph(c, iconWarn, t.Warning, "exclamation")
				typeSecondary.style(t, gui.Text(c, "切到「"+a.tierDraft+"」会影响之后所有操作。")).
					TextColor(t.TextMuted)
				gui.Spacer(c)
			})
			gui.Row(c).Gap(space2).Children(func() {
				if actionButton(c, "确认切换", "确认切换权限档位", true, false).Clicked() && a.intents != nil {
					tier := tierFromLabel(a.tierDraft)
					a.dispatch("权限档位", func(ctx context.Context) error {
						return a.intents.SetPermissionTier(ctx, tier)
					}, func() { a.tierDraft = "" })
				}
				if actionButton(c, "取消", "取消切换权限档位", false, false).Clicked() {
					a.tierDraft = current
				}
				gui.Spacer(c)
			})
		}
		a.providerManagement(c, st)
		return
	}
	settingsRow(c, "当前供应商", func() {
		gui.Select(c, &a.provider, a.providers).Label("当前供应商")
	})
	settingsRow(c, "模型", func() {
		gui.Select(c, &a.modelChoice, a.models).Label("模型")
	})
	// 权限档位：改变后续所有操作的授权范围（高影响），因此两步确认（AC-09）。
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Children(func() {
		typeControl.style(t, gui.Text(c, "权限档位")).Width(SettingsLabelWidth).TextColor(t.Text)
		gui.Select(c, &a.tierDraft, []string{"工作区内自动", "每次询问", "只读"}).
			Label("权限档位选择")
		gui.Spacer(c)
	})
	if a.tierDraft != a.tier && a.tierPending == "" {
		a.tierPending = a.tierDraft
	}
	if a.tierPending != "" && a.tierPending != a.tier {
		gui.Row(c).Gap(space1).AlignItems(gui.Center).Shrink(0).
			Label("权限档位确认").Children(func() {
			iconGlyph(c, iconWarn, t.Warning, "exclamation")
			typeSecondary.style(t, gui.Text(c, "切到「"+a.tierPending+"」会影响之后所有操作。")).
				TextColor(t.TextMuted)
			gui.Spacer(c)
		})
		gui.Row(c).Gap(space2).Children(func() {
			if actionButton(c, "确认切换", "确认切换权限档位", true, false).Clicked() {
				a.tier = a.tierPending
				a.tierDraft = a.tier
				a.tierPending = ""
			}
			if actionButton(c, "取消", "取消切换权限档位", false, false).Clicked() {
				a.tierDraft = a.tier
				a.tierPending = ""
			}
			gui.Spacer(c)
		})
	}

	// 目录单价是内容（G6）：只读一行，不在主界面出现。
	typeSecondary.style(t, gui.Text(c, a.modelPrice)).TextColor(t.TextMuted).
		Label("目录单价")
}

// settingsRules 是「命令规则」：每条规则一行（命令 + 决定 + 匹配方式），下面是
// 高后果动作「清空命令规则」（AC-13，两步确认）。
func (a *App) settingsRules(c *gui.Context) {
	t := c.Theme()
	settingsSectionTitle(c, "命令规则")
	typeSecondary.style(t, gui.Text(c, "规则对所有会话生效；拒绝先于允许。")).
		TextColor(t.TextMuted).Label("规则说明")
	if st, ok := a.liveSettings(); ok {
		a.settingsRulesLive(c, st)
		return
	}

	// 窄窗退化（U2-forms F23）：三列排不下一行时，一条规则两行——先命令，
	// 再「决定 + 匹配方式」。表头只在三列并排时出现（堆叠后每列带自己的标签）。
	width, _ := c.Size()
	inner := width - SettingsNavWidth - 2*space8
	stacked := inner < SettingsRuleRowMinWidth

	if !stacked {
		gui.Row(c).Gap(space4).Shrink(0).Children(func() {
			typeSecondary.style(t, gui.Text(c, "命令")).Width(SettingsRuleCommandWidth).
				TextColor(t.TextMuted).Label("命令列")
			typeSecondary.style(t, gui.Text(c, "决定")).Width(SettingsRuleSelectWidth).
				TextColor(t.TextMuted).Label("决定列")
			typeSecondary.style(t, gui.Text(c, "匹配方式")).Width(SettingsRuleSelectWidth).
				TextColor(t.TextMuted).Label("匹配列")
		})
	}
	for i := range a.rules {
		r := &a.rules[i]
		if !stacked {
			gui.Row(c).Gap(space4).AlignItems(gui.Center).Shrink(0).Children(func() {
				typeCode.style(t, gui.RichText(c, gui.Span{
					Text: r.command, Size: typeCode.sizeIn(t), Weight: typeCode.weight, Color: t.Text,
				})).Width(SettingsRuleCommandWidth).SingleLine()
				gui.Select(c, &r.decision, []string{"允许", "拒绝"}).
					Width(SettingsRuleSelectWidth).Label("决定：" + r.command)
				gui.Select(c, &r.match, []string{"精确", "前缀"}).
					Width(SettingsRuleSelectWidth).Label("匹配方式：" + r.command)
			})
			continue
		}
		gui.Column(c).Shrink(0).Gap(space1).Children(func() {
			typeCode.style(t, gui.RichText(c, gui.Span{
				Text: r.command, Size: typeCode.sizeIn(t), Weight: typeCode.weight, Color: t.Text,
			})).SingleLine()
			gui.Row(c).Gap(space3).AlignItems(gui.Center).Shrink(0).Children(func() {
				typeSecondary.style(t, gui.Text(c, "决定")).TextColor(t.TextMuted)
				gui.Select(c, &r.decision, []string{"允许", "拒绝"}).
					Label("决定：" + r.command)
				typeSecondary.style(t, gui.Text(c, "匹配方式")).TextColor(t.TextMuted)
				gui.Select(c, &r.match, []string{"精确", "前缀"}).
					Label("匹配方式：" + r.command)
				gui.Spacer(c)
			})
		})
	}
	gui.Spacer(c)
	// 高后果动作：图标 + 文字，两步确认（U3-actions AC-13）。
	label, tip := "清空命令规则", "删除所有会话的全部命令规则"
	if a.clearRulesConfirm {
		label, tip = "确认清空", "再点一次：清空全部规则，无法撤销"
	}
	gui.Row(c).Shrink(0).Children(func() {
		if textIconButton(c, textIconButtonSpec{
			icon: iconXMark, mark: "x-mark", text: label, tip: tip,
		}) {
			if a.clearRulesConfirm {
				a.rules = nil
				a.clearRulesConfirm = false
			} else {
				a.clearRulesConfirm = true
			}
		}
	})
	if a.clearRulesConfirm {
		typeSecondary.style(t, gui.Text(c, "清空后所有会话立即生效，无法撤销。")).
			TextColor(t.Danger).Label("清空确认说明")
	}
}

// settingsRulesLive 是接上事实后的命令规则：读数来自 permission.rules，改动与
// 清空都发 permission.set_rules。规则语义（拒绝先于允许）由服务端解释。
func (a *App) settingsRulesLive(c *gui.Context, st SettingsSnapshot) {
	t := c.Theme()
	if !st.RulesKnown {
		reason := "尚未读取"
		if st.RulesError != "" {
			reason = st.RulesError
		}
		a.stateNote(c, "命令规则", iconClock, "clock", reason, t.TextMuted)
		return
	}
	if len(st.Rules) == 0 {
		a.emptyNote(c, "命令规则空态", "还没有命令规则")
		return
	}
	width, _ := c.Size()
	inner := width - SettingsNavWidth - 2*space8
	stacked := inner < SettingsRuleRowMinWidth
	if !stacked {
		gui.Row(c).Gap(space4).Shrink(0).Children(func() {
			typeSecondary.style(t, gui.Text(c, "命令")).Width(SettingsRuleCommandWidth).
				TextColor(t.TextMuted).Label("命令列")
			typeSecondary.style(t, gui.Text(c, "决定")).Width(SettingsRuleSelectWidth).
				TextColor(t.TextMuted).Label("决定列")
			typeSecondary.style(t, gui.Text(c, "匹配方式")).Width(SettingsRuleSelectWidth).
				TextColor(t.TextMuted).Label("匹配列")
		})
	}
	for index := range st.Rules {
		rule := st.Rules[index]
		decision := decisionLabel(rule.Decision)
		match := matchLabel(rule.Match)
		changed := false
		nextDecision, nextMatch := rule.Decision, rule.Match
		if !stacked {
			gui.Row(c).Gap(space4).AlignItems(gui.Center).Shrink(0).Children(func() {
				typeCode.style(t, gui.RichText(c, gui.Span{
					Text: rule.Command, Size: typeCode.sizeIn(t), Weight: typeCode.weight, Color: t.Text,
				})).Width(SettingsRuleCommandWidth).SingleLine()
				if actionButton(c, "允许", "允许："+rule.Command, decision == "允许", decision == "允许").Clicked() {
					nextDecision, changed = "allow", true
				}
				if textIconButton(c, textIconButtonSpec{
					icon: iconXMark, mark: "x-mark", text: "拒绝", label: "拒绝：" + rule.Command,
					primary: decision == "拒绝", disabled: decision == "拒绝",
				}) {
					nextDecision, changed = "deny", true
				}
				if actionButton(c, "精确", "精确："+rule.Command, match == "精确", match == "精确").Clicked() {
					nextMatch, changed = "exact", true
				}
				if actionButton(c, "前缀", "前缀："+rule.Command, match == "前缀", match == "前缀").Clicked() {
					nextMatch, changed = "prefix", true
				}
			})
		} else {
			gui.Column(c).Shrink(0).Gap(space1).Children(func() {
				typeCode.style(t, gui.RichText(c, gui.Span{
					Text: rule.Command, Size: typeCode.sizeIn(t), Weight: typeCode.weight, Color: t.Text,
				})).SingleLine()
				gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Children(func() {
					if actionButton(c, "允许", "允许："+rule.Command, decision == "允许", decision == "允许").Clicked() {
						nextDecision, changed = "allow", true
					}
					if textIconButton(c, textIconButtonSpec{
						icon: iconXMark, mark: "x-mark", text: "拒绝", label: "拒绝：" + rule.Command,
						primary: decision == "拒绝", disabled: decision == "拒绝",
					}) {
						nextDecision, changed = "deny", true
					}
					if actionButton(c, "精确", "精确："+rule.Command, match == "精确", match == "精确").Clicked() {
						nextMatch, changed = "exact", true
					}
					if actionButton(c, "前缀", "前缀："+rule.Command, match == "前缀", match == "前缀").Clicked() {
						nextMatch, changed = "prefix", true
					}
					gui.Spacer(c)
				})
			})
		}
		if a.intents == nil || !changed {
			continue
		}
		next := append([]RuleSnapshot(nil), st.Rules...)
		next[index].Decision = nextDecision
		next[index].Match = nextMatch
		a.dispatch("修改命令规则", func(ctx context.Context) error {
			return a.intents.WriteCommandRules(ctx, next)
		}, nil)
		return
	}
	gui.Spacer(c)
	label, tip := "清空命令规则", "删除所有会话的全部命令规则"
	if a.clearRulesConfirm {
		label, tip = "确认清空", "再点一次：清空全部规则，无法撤销"
	}
	gui.Row(c).Shrink(0).Children(func() {
		if textIconButton(c, textIconButtonSpec{
			icon: iconXMark, mark: "x-mark", text: label, tip: tip,
		}) {
			if a.clearRulesConfirm {
				a.clearRulesConfirm = false
				if a.intents != nil {
					a.dispatch("清空命令规则", func(ctx context.Context) error {
						return a.intents.WriteCommandRules(ctx, nil)
					}, nil)
				}
			} else {
				a.clearRulesConfirm = true
			}
		}
	})
	if a.clearRulesConfirm {
		typeSecondary.style(t, gui.Text(c, "清空后所有会话立即生效，无法撤销。")).
			TextColor(t.Danger).Label("清空确认说明")
	}
}

// decisionLabel / decisionWire 在界面语与 wire 取值之间转换。
func decisionLabel(value string) string {
	if value == "deny" {
		return "拒绝"
	}
	return "允许"
}

func decisionWire(label string) string {
	if label == "拒绝" {
		return "deny"
	}
	return "allow"
}

// matchLabel / matchWire 同上。
func matchLabel(value string) string {
	if value == "prefix" {
		return "前缀"
	}
	return "精确"
}

func matchWire(label string) string {
	if label == "前缀" {
		return "prefix"
	}
	return "exact"
}

// settingsSummary 是入口语（样例）：提供给导航与测试读取的当前值摘要。
func (a *App) settingsSummary() string {
	return fmt.Sprintf("%s · %s", a.themePref, a.provider)
}

// settingsAccount 是「账号与套餐」分节（U1-ia §4 第 1 节，T24/T27）：身份与套餐
// 读数都是事实，只读；登录/退出属后续批次。
func (a *App) settingsAccount(c *gui.Context) {
	t := c.Theme()
	settingsSectionTitle(c, "账号与套餐")
	if snap, ok := a.liveAccount(); ok {
		a.settingsAccountLive(c, snap)
		return
	}
	rows := []struct{ label, value string }{
		{"账号", "E · eric@example.com"},
		{"套餐", "Pro · 每月 $20"},
		{"本月额度", "34% 已用（约 680 万 token）"},
		{"用量重置", "12 月 1 日"},
	}
	for _, r := range rows {
		gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).
			Label("账号信息：" + r.label).Children(func() {
			typeControl.style(t, gui.Text(c, r.label)).Width(SettingsLabelWidth).
				TextColor(t.Text)
			typeSecondary.style(t, gui.Text(c, r.value)).TextColor(t.TextMuted)
			gui.Spacer(c)
		})
	}
}

// settingsAccountLive 是接上事实后的账号分节：缺项显示「—」，不用样例顶替。
func (a *App) settingsAccountLive(c *gui.Context, snap AccountSnapshot) {
	t := c.Theme()
	identity := snap.Email
	if identity == "" {
		identity = snap.Name
	}
	quota := "—"
	if usage, ok := a.liveUsage(); ok && usage.Available {
		quota = fmt.Sprintf("%g / %g %s", usage.Used, usage.Limit, usage.Unit)
	} else if usage.Reason != "" {
		quota = usage.Reason
	}
	reset := "—"
	if usage, ok := a.liveUsage(); ok && usage.HasResets {
		reset = usage.ResetsAt.Local().Format("2006-01-02")
	}
	rows := []struct{ label, value string }{
		{"账号", dashIfEmpty(identity)},
		{"套餐", dashIfEmpty(snap.Plan)},
		{"本月额度", quota},
		{"用量重置", reset},
	}
	for _, r := range rows {
		settingsValueRow(c, "账号信息："+r.label, r.label, r.value)
	}
	if a.intents == nil {
		return
	}
	if snap.Status == "signed_in" {
		if actionButton(c, "退出登录", "退出登录", false, false).Clicked() {
			a.dispatch("退出登录", func(ctx context.Context) error {
				return a.intents.Logout(ctx)
			}, nil)
		}
		return
	}
	if snap.LoginPhase == "waiting" || snap.LoginPhase == "failed" {
		if snap.LoginUserCode != "" {
			settingsValueRow(c, "登录验证码", "验证码", snap.LoginUserCode)
		}
		if snap.LoginURL != "" {
			settingsValueRow(c, "登录地址", "地址", snap.LoginURL)
		}
		if snap.LoginReason != "" {
			a.stateNote(c, "登录状态", iconWarn, "exclamation", snap.LoginReason, t.Danger)
		}
		gui.Row(c).Gap(space2).Children(func() {
			if actionButton(c, "等待完成", "等待登录完成", true, false).Clicked() {
				a.dispatch("等待登录", func(ctx context.Context) error {
					return a.intents.WaitLogin(ctx)
				}, nil)
			}
			if actionButton(c, "取消登录", "取消登录", false, false).Clicked() {
				a.dispatch("取消登录", func(ctx context.Context) error {
					return a.intents.CancelLogin(ctx)
				}, nil)
			}
			gui.Spacer(c)
		})
		return
	}
	gui.Row(c).Gap(space2).Children(func() {
		for _, method := range snap.LoginMethods {
			method := method
			label := "浏览器登录"
			if method == "device" {
				label = "设备码登录"
			}
			if actionButton(c, label, label, true, false).Clicked() {
				a.dispatch(label, func(ctx context.Context) error {
					_, err := a.intents.BeginLogin(ctx, method)
					return err
				}, nil)
			}
		}
		gui.Spacer(c)
	})
}

// providerIDByName 把展示名换回 provider 标识。
func providerIDByName(providers []ProviderOption, name string) string {
	for _, provider := range providers {
		if provider.Name == name {
			return provider.ID
		}
	}
	return ""
}

// modelsForProvider 返回某个 provider 下的模型。
func modelsForProvider(models []ModelOption, providerID string) []ModelOption {
	out := make([]ModelOption, 0, len(models))
	for _, model := range models {
		if model.ProviderID == providerID {
			out = append(out, model)
		}
	}
	return out
}

// modelIDByName 把模型展示名换回模型标识。
func modelIDByName(models []ModelOption, providerID, name string) string {
	for _, model := range models {
		if model.ProviderID == providerID && model.Name == name {
			return model.ID
		}
	}
	return ""
}

// settingsValueRow 是只读的一行：左侧标签 + 右侧值（值缺失时显示「—」）。
func settingsValueRow(c *gui.Context, label, name, value string) {
	t := c.Theme()
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Label(label).Children(func() {
		typeControl.style(t, gui.Text(c, name)).Width(SettingsLabelWidth).TextColor(t.Text)
		typeSecondary.style(t, gui.Text(c, dashIfEmpty(value))).TextColor(t.TextMuted)
		gui.Spacer(c)
	})
}

// dashIfEmpty 把空值显示成「—」：不发明内容，也不留空。
func dashIfEmpty(value string) string {
	if strings.TrimSpace(value) == "" {
		return "—"
	}
	return value
}

// providerManagement 是 provider 管理：新增/编辑/探测与模型启用。
func (a *App) providerManagement(c *gui.Context, st SettingsSnapshot) {
	t := c.Theme()
	gui.Row(c).Gap(space2).AlignItems(gui.Center).Shrink(0).Label("供应商管理").Children(func() {
		typeControl.style(t, gui.Text(c, "供应商管理")).Width(SettingsLabelWidth).TextColor(t.Text)
		if actionButton(c, "新增供应商", "新增供应商", false, false).Clicked() {
			a.providerFormOpen = true
			a.providerEditing = ""
			a.providerIDDraft, a.providerNameDraft, a.providerBaseDraft, a.providerKeyDraft = "", "", "", ""
		}
		gui.Spacer(c)
	})
	if a.providerFormOpen {
		gui.Column(c).Shrink(0).Gap(space1).Label("供应商表单").Children(func() {
			if a.providerEditing == "" {
				gui.TextInputBase(c, &a.providerIDDraft).Label("provider 标识").
					Padding(space1, space2).Radius(radius).Background(t.SurfacePressed)
			}
			gui.TextInputBase(c, &a.providerNameDraft).Label("provider 名称").
				Padding(space1, space2).Radius(radius).Background(t.SurfacePressed)
			gui.TextInputBase(c, &a.providerBaseDraft).Label("Base URL").
				Padding(space1, space2).Radius(radius).Background(t.SurfacePressed)
			gui.TextInputBase(c, &a.providerKeyDraft).Label("API 密钥").
				Padding(space1, space2).Radius(radius).Background(t.SurfacePressed)
			gui.Row(c).Gap(space2).Children(func() {
				gui.Spacer(c)
				label := "创建"
				if a.providerEditing != "" {
					label = "保存"
				}
				if actionButton(c, label, label+" provider", true, false).Clicked() && a.intents != nil {
					id, name, baseURL, key := a.providerIDDraft, a.providerNameDraft, a.providerBaseDraft, a.providerKeyDraft
					editing := a.providerEditing
					edit := editing
					if edit == "" {
						edit = id
					}
					a.dispatch("保存供应商", func(ctx context.Context) error {
						if editing == "" {
							return a.intents.CreateProvider(ctx, id, name, baseURL, key)
						}
						return a.intents.UpdateProvider(ctx, edit, name, baseURL, key)
					}, func() { a.providerFormOpen = false; a.providerEditing = "" })
				}
				if actionButton(c, "取消", "取消供应商编辑", false, false).Clicked() {
					a.providerFormOpen = false
					a.providerEditing = ""
				}
			})
		})
	}
	if st.ProviderError != "" {
		a.stateNote(c, "供应商读取失败", iconWarn, "exclamation", st.ProviderError, t.Danger)
	}
	for _, provider := range st.ProviderRecords {
		provider := provider
		gui.Column(c).Shrink(0).Gap(space1).Label("供应商：" + provider.ID).Children(func() {
			gui.Row(c).Gap(space2).AlignItems(gui.Center).Children(func() {
				statusDotMark(c, statusColor(t, extOK), "provider:"+provider.ID)
				typeBody.style(t, gui.Text(c, provider.Name)).TextColor(t.Text).Width(extNameWidth)
				typeSecondary.style(t, gui.Text(c, provider.BaseURL)).TextColor(t.TextMuted).Grow(1).SingleLine()
				if actionButton(c, "探测", "探测 "+provider.ID, false, false).Clicked() && a.intents != nil {
					id := provider.ID
					var count int
					a.dispatch("探测供应商", func(ctx context.Context) error {
						var err error
						count, err = a.intents.ProbeProvider(ctx, id)
						return err
					}, func() {
						a.providerProbe[id] = fmt.Sprintf("发现 %d 个模型", count)
					})
				}
				if actionButton(c, "编辑", "编辑 "+provider.ID, false, false).Clicked() {
					a.providerFormOpen = true
					a.providerEditing = provider.ID
					a.providerIDDraft = provider.ID
					a.providerNameDraft = provider.Name
					a.providerBaseDraft = provider.BaseURL
					a.providerKeyDraft = ""
				}
			})
			if a.providerProbe[provider.ID] != "" {
				a.stateNote(c, "探测结果", iconCheck, "check", a.providerProbe[provider.ID], t.TextMuted)
			}
			for _, model := range provider.Models {
				model := model
				label := "启用模型：" + model.ID
				if model.Enabled {
					label = "停用模型：" + model.ID
				}
				if actionButton(c, "切换模型", label, false, false).Clicked() && a.intents != nil {
					providerID, modelID, enabled := provider.ID, model.ID, !model.Enabled
					a.dispatch("切换模型启用", func(ctx context.Context) error {
						return a.intents.SetProviderModelEnabled(ctx, providerID, modelID, enabled)
					}, nil)
				}
			}
		})
	}
}
