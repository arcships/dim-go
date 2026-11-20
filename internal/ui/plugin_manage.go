package ui

import (
	"context"
	"fmt"
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是设置 › 插件设置里的插件管理块（§8 导入与更新）：
//
//	我的插件（已装清单 + 添加 / 重新加载 + 逐项更新 / 移除）
//	可用更新（逐项更新与全部更新）
//	插件市场（源地址配置；未配置时整块禁用并写「未配置插件源」，不画样例卡片）
//
// 读的是 Extensions().Plugins 的事实；写走 Intents（plugin.install / update /
// remove）。没有写通道时入口置灰并写原因，不假装生效。市场清单来自本地偏好里
// 配置的上游源（ui.PluginMarketplaceFacts），协议没有市场安装来源，所以市场
// 条目的「安装」入口置灰并写原因。
//
// 与参考产品的差异：参考产品的插件管理在「技能/插件」路由页的「插件」页签
// （PG-12…PG-29），本版没有重建该路由页；左栏「技能/插件」落到设置 › 插件
// 设置，因此管理块画在这一节里。插件设置本身的 schema 表单仍按 S-pluginSettings
// 的「不可用」分支处理。

const (
	pluginManageTitle       = "我的插件"
	pluginManageAdd         = "添加插件"
	pluginManageReload      = "重新加载"
	pluginManageEmpty       = "没有已安装插件"
	pluginManageNoFacts     = "未连接 App Server，没有插件清单"
	pluginManageReadError   = "读取插件清单失败"
	pluginManageBuiltin     = "内置插件不能移除"
	pluginManageLocked      = "服务端锁定，不能移除"
	pluginManageEnabled     = "已启用"
	pluginManageDisabled    = "已禁用"
	pluginManageUnavail     = "不可用"
	pluginManageUpdates     = "可用更新"
	pluginManageUpdateAll   = "全部更新（%d）"
	pluginManageUpdate      = "更新"
	pluginManageUpdating    = "更新中…"
	pluginManageAllUpdating = "正在更新…"
	pluginManageRemove      = "移除"
	pluginManageAllDone     = "已更新 %d 个插件"
	pluginManageUpdateFail  = "更新失败"
	pluginManageNoUpdates   = "所有插件均为最新版本"
	// pluginUpdateNoProgress 是 G-H4-05 下载进度条的缺项原因：插件清单里没有
	// 下载进度字段（protocol v3 也没有进度事件），因此更新中不画进度条，只写原因。
	pluginUpdateNoProgress = "服务端没有下载进度读数，不画进度条"

	pluginMarketTitle       = "插件市场"
	pluginMarketFeedLabel   = "插件源"
	pluginMarketFeedPlace   = "https://example.com/plugins.json"
	pluginMarketSave        = "保存"
	pluginMarketSaving      = "正在保存…"
	pluginMarketUnset       = "未配置插件源"
	pluginMarketUnavailable = "市场暂不可用"
	// pluginMarketNoFacts 是「市场暂不可用」的原因：这条连接没有市场读数。
	pluginMarketNoFacts   = "这条连接没有插件市场读数（Source 没有提供市场快照）"
	pluginMarketLoading   = "正在加载..."
	pluginMarketEmpty     = "插件源没有返回条目"
	pluginMarketRetry     = "重试"
	pluginMarketRefresh   = "刷新"
	pluginMarketInstall   = "安装"
	pluginMarketNoInstall = "当前版本不支持从市场安装"
	// G-H2-01 安装详情的标题与 G-H2-02 描述（逐字取参考产品中文）。
	pluginMarketInstallTitleFmt = "安装 %s"
	pluginMarketInstallDesc     = "选择要启用的内置技能。默认全选。"
	// G-H2-06 / G-H2-07 技能清单的缺项原因：市场条目没有技能字段（协议也没有
	// 市场安装来源），因此不画勾选行，也不写「此插件不包含技能」那种断言。
	pluginMarketNoSkills = "插件源没有给出这个条目的技能清单，无法勾选技能"

	pluginInstalledFmt = "已安装 %s"
	pluginRemovedFmt   = "已移除 %s"
	pluginUpdatedFmt   = "已更新 %s"

	pluginRemoveTitle   = "移除插件？"
	pluginRemoveDescFmt = "将移除「%s」及其技能。此操作无法撤销。"
	pluginRemoveBusy    = "移除中…"
)

// pluginManageState 是插件管理块的可丢视图状态。
type pluginManageState struct {
	notice string
	// updating 按插件标识记「这一行正在更新」；rowErr 是逐行失败原因。
	updating    map[string]bool
	rowErr      map[string]string
	updatingAll bool
	updateErr   string
	// 移除确认框：目标、进行中与失败原因。
	removeID   string
	removeName string
	removeBusy bool
	removeErr  string
	// 市场源草稿与保存回执。
	feedDraft  string
	feedLoaded bool
	feedSaving bool
	feedErr    string
	// 市场安装详情（G-H2）：打开的是哪一个条目。条目来自上游清单的只读快照。
	marketOpen  bool
	marketEntry PluginMarketEntrySnapshot
}

// pluginManage 返回本轮要画的插件管理状态（惰性建映射）。
func (a *App) pluginManageState() *pluginManageState {
	if a.pluginManage.updating == nil {
		a.pluginManage.updating = map[string]bool{}
	}
	if a.pluginManage.rowErr == nil {
		a.pluginManage.rowErr = map[string]string{}
	}
	return &a.pluginManage
}

// pluginList 取插件清单读数；ok 为假表示没有连接事实。
func (a *App) pluginList() (ExtListSnapshot, bool) {
	if a.source == nil {
		return ExtListSnapshot{}, false
	}
	return a.source.Extensions().Plugins, true
}

// pluginMarketFacts 取插件市场读数：Source 实现可选接口时才有。
func (a *App) pluginMarketFacts() (PluginMarketplaceSnapshot, bool) {
	if a.source == nil {
		return PluginMarketplaceSnapshot{}, false
	}
	facts, ok := a.source.(PluginMarketplaceFacts)
	if !ok {
		return PluginMarketplaceSnapshot{}, false
	}
	return facts.PluginMarketplace(), true
}

// pluginMarketIntents 取插件市场动作：Intents 实现可选接口时才有。
func (a *App) pluginMarketIntents() (PluginMarketplaceIntents, bool) {
	intents, ok := a.intents.(PluginMarketplaceIntents)
	return intents, ok
}

// pluginManageView 画插件管理块（设置 › 插件设置的最上面一段）。
func (a *App) pluginManageView(c *gui.Context) {
	t := c.Theme()
	st := a.pluginManageState()
	plugins, facts := a.pluginList()
	canWrite := a.intents != nil
	writeNote := ""
	if !facts {
		writeNote = pluginManageNoFacts
	} else if !canWrite {
		writeNote = noteNoWriteChannel
	}
	updates := pluginUpdateItems(plugins.Items)

	gui.Column(c).Shrink(0).Gap(plgCardGap).Children(func() {
		screenCard(c, func() {
			gui.Column(c).Padding(plgCardPad, plgCardPad).Gap(plgCardInner).
				Children(func() {
					gui.Row(c).AlignItems(gui.Center).Gap(space2).Children(func() {
						plgCardTitle.style(t, gui.Text(c, pluginManageTitle)).
							TextColor(t.Text).SingleLine().
							Role(gui.RoleHeading).Level(2)
						gui.Spacer(c)
						if pluginSmallButton(c, t, pluginManageReload, false, true, notePluginReload) {
							// 没有重扫通道：按钮恒禁用，这里不会到达。
						}
						if pluginSmallButtonLabeled(c, t, pluginManageAdd, "打开添加插件弹窗",
							true, false, !canWrite || !facts, manageNote(facts, writeNote)) {
							a.openPluginInstall()
						}
					})
					if st.notice != "" {
						typeSecondaryStrong.style(t, gui.Text(c, st.notice)).
							TextColor(t.Success).Label(st.notice)
					}
					if writeNote != "" {
						plgNote(c, t, writeNote)
					}
					if facts && plugins.ReadError != "" {
						typeSecondary.style(t, gui.Text(c, pluginManageReadError+"："+plugins.ReadError)).
							TextColor(t.Danger).Role(gui.RoleAlert).
							Label(pluginManageReadError + "：" + plugins.ReadError)
					}
					if facts && len(plugins.Items) == 0 {
						plgNote(c, t, pluginManageEmpty)
					}
					for _, item := range plugins.Items {
						a.pluginRow(c, t, item, canWrite)
					}
					a.pluginUpdatesSection(c, t, updates, canWrite, facts && plugins.ReadError == "")
					a.pluginMarketSection(c, t)
				})
		})
	})
}

// manageNote 取「添加插件」置灰的原因。
func manageNote(facts bool, writeNote string) string {
	if !facts {
		return pluginManageNoFacts
	}
	if writeNote != "" {
		return writeNote
	}
	return ""
}

// pluginUpdateItems 取有可用更新的插件（服务端声明，本端不比较版本号）。
func pluginUpdateItems(items []ExtItemSnapshot) []ExtItemSnapshot {
	out := make([]ExtItemSnapshot, 0, len(items))
	for _, item := range items {
		if item.UpdateAvailable {
			out = append(out, item)
		}
	}
	return out
}

// pluginRow 是一行已装插件：名称 / 版本 / 状态 / 来源，加移除。更新入口只在
// 「可用更新」区里（G-H4），避免同一动作在同一屏出现两遍。
func (a *App) pluginRow(c *gui.Context, t *gui.Theme, item ExtItemSnapshot, canWrite bool) {
	st := a.pluginManageState()
	gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(space2).Children(func() {
		gui.Column(c).Shrink(0).Grow(1).Gap(spaceUnit).Children(func() {
			typeSecondaryStrong.style(t, gui.Text(c, item.Name)).TextColor(t.Text).
				SingleLine().Label(item.Name)
			meta := pluginStatus(item)
			if item.Version != "" {
				meta += " · " + pluginVersionLabel(item.Version)
			}
			if item.Use == "builtin" || item.Source == "builtin" {
				meta += " · builtin"
			}
			typeSecondary.style(t, gui.Text(c, meta)).TextColor(t.TextMuted).
				SingleLine().Label(meta)
		})
		removable := !item.Locked && item.Use != "builtin" && item.Source != "builtin"
		removeNote := pluginWriteNote(canWrite)
		if !removable {
			if item.Locked {
				removeNote = pluginManageLocked
			} else {
				removeNote = pluginManageBuiltin
			}
		}
		if pluginSmallButton(c, t, pluginManageRemove, false,
			!canWrite || !removable || st.removeBusy, removeNote) {
			a.openPluginRemove(item)
		}
	})
	if item.Error != "" {
		typeSecondary.style(t, gui.Text(c, item.Error)).TextColor(t.Danger).
			Role(gui.RoleAlert).Label(item.Name + "：" + item.Error)
	}
	if msg := st.rowErr[item.ID]; msg != "" {
		typeSecondary.style(t, gui.Text(c, msg)).TextColor(t.Danger).
			Role(gui.RoleAlert).Label(msg)
	}
}

// pluginStatus 是插件行的状态词：服务端错误 → 不可用；有 status 用原文；
// 否则按 enabled 写已启用 / 已禁用。
func pluginStatus(item ExtItemSnapshot) string {
	switch {
	case item.Error != "":
		return pluginManageUnavail
	case item.Status != "":
		return item.Status
	case item.Enabled:
		return pluginManageEnabled
	default:
		return pluginManageDisabled
	}
}

// pluginWriteNote 是写入口置灰的原因。
func pluginWriteNote(canWrite bool) string {
	if canWrite {
		return ""
	}
	return noteNoWriteChannel
}

// pluginSmallButton 是管理块里的行内小按钮：高 28，可选主按钮与禁用原因。
func pluginSmallButton(c *gui.Context, t *gui.Theme, text string, primary, disabled bool, tip string) bool {
	return pluginSmallButtonLabeled(c, t, text, text, primary, false, disabled, tip)
}

// pluginSmallButtonLabeled 与 pluginSmallButton 相同，可另给读屏名与「进行中」
// （按钮文字前加一个旋转图标）。
func pluginSmallButtonLabeled(c *gui.Context, t *gui.Theme, text, label string, primary, busy, disabled bool, tip string) bool {
	b := gui.ButtonBase(c).Shrink(0).Height(plgBtnH).
		Padding(plgAddPadV, controlPadH).Radius(radius).Label(label).
		Background(t.Background)
	if primary {
		b = b.Background(t.Accent)
	}
	if tip != "" {
		b = b.Tooltip(tip)
	}
	if disabled {
		b = b.Disabled(true)
	}
	switch {
	case disabled:
		b = b.Opacity(0.5)
	case !primary && b.Pressed():
		b = b.Background(t.SurfacePressed)
	case !primary && b.Hovered():
		b = b.Background(t.SurfaceHover)
	}
	color := t.Text
	if primary {
		color = t.AccentText
	}
	b.Children(func() {
		gui.Row(c).Gap(space1).AlignItems(gui.Center).Children(func() {
			if busy {
				gui.Spinner(c).Size(plgIconSize, plgIconSize)
			}
			plgBtnTextSty.style(t, gui.Text(c, text)).TextColor(color).SingleLine()
		})
	})
	return b.Clicked()
}

// pluginUpdatesSection 是「可用更新」：清单读到时总是画标题；没有可更新项时
// 给空态「所有插件均为最新版本」，不把「读不到清单」说成「都最新」。
// G-H4-01 标题 / G-H4-02 空态 / G-H4-05 进度条缺项原因
func (a *App) pluginUpdatesSection(c *gui.Context, t *gui.Theme, updates []ExtItemSnapshot,
	canWrite, known bool) {
	if !known {
		return
	}
	st := a.pluginManageState()
	gui.Column(c).Shrink(0).Gap(plgCardInner).Children(func() {
		hairlineBox(c, true, "")
		gui.Row(c).AlignItems(gui.Center).Gap(space2).Children(func() {
			// G-H4-01 「可用更新」
			plgCardTitle.style(t, gui.Text(c, pluginManageUpdates)).TextColor(t.Text).
				SingleLine().Role(gui.RoleHeading).Level(3)
			gui.Spacer(c)
			if len(updates) == 0 {
				return
			}
			// G-H4-07 全部更新（{count}）
			label := fmt.Sprintf(pluginManageUpdateAll, len(updates))
			if st.updatingAll {
				label = pluginManageAllUpdating
			}
			if pluginSmallButton(c, t, label, true,
				!canWrite || st.updatingAll, pluginWriteNote(canWrite)) {
				a.updateAllPlugins(updates)
			}
		})
		if len(updates) == 0 {
			// G-H4-02 空态（14px 次要色；读屏名就是这一句）
			plgNote(c, t, pluginManageNoUpdates)
			return
		}
		if st.updateErr != "" {
			typeSecondary.style(t, gui.Text(c, st.updateErr)).TextColor(t.Danger).
				Role(gui.RoleAlert).Label(st.updateErr)
		}
		for _, item := range updates {
			gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(space2).Children(func() {
				gui.Column(c).Shrink(0).Grow(1).Gap(spaceUnit).Children(func() {
					// G-H4-03 插件图标 + 名称
					typeSecondaryStrong.style(t, gui.Text(c, item.Name)).TextColor(t.Text).
						SingleLine().Label(item.Name)
					// G-H4-04 v{旧} → v{新} / v{新}
					versions := pluginVersionLine(item)
					if versions != "" {
						typeSecondary.style(t, gui.Text(c, versions)).TextColor(t.TextMuted).
							Label(versions)
					}
					// G-H4-05 更新中：没有进度事实，就写清为什么没有进度条
					if st.updating[item.ID] {
						typeMicro.style(t, gui.Text(c, pluginUpdateNoProgress)).
							TextColor(t.TextMuted).Label(pluginUpdateNoProgress).SingleLine()
					}
				})
				// G-H4-06 更新
				label := pluginManageUpdate
				if st.updating[item.ID] {
					label = pluginManageUpdating
				}
				if pluginSmallButtonLabeled(c, t, label, label, false,
					st.updating[item.ID], !canWrite || st.updating[item.ID] || st.updatingAll,
					pluginWriteNote(canWrite)) {
					a.updatePlugin(item)
				}
			})
		}
	})
}

// pluginVersionLine 是版本行：有旧版本时「v旧 → v新」，否则只写「v新」。
func pluginVersionLine(item ExtItemSnapshot) string {
	latest := strings.TrimSpace(item.LatestVersion)
	if latest == "" {
		return ""
	}
	if strings.TrimSpace(item.Version) == "" {
		return pluginVersionLabel(latest)
	}
	return pluginVersionLabel(item.Version) + " → " + pluginVersionLabel(latest)
}

// pluginVersionLabel 给版本号补一个 v：服务端已经带 v 时不重复。
func pluginVersionLabel(version string) string {
	version = strings.TrimSpace(version)
	if version == "" {
		return ""
	}
	if strings.HasPrefix(version, "v") || strings.HasPrefix(version, "V") {
		return version
	}
	return "v" + version
}

// updatePlugin 更新一个插件；失败原因留在该行。
func (a *App) updatePlugin(item ExtItemSnapshot) {
	if a.intents == nil {
		return
	}
	st := a.pluginManageState()
	if st.updating[item.ID] {
		return
	}
	st.updating[item.ID] = true
	delete(st.rowErr, item.ID)
	a.dispatchQuiet("更新插件", func(ctx context.Context) error {
		return a.intents.UpdatePlugin(ctx, item.ID)
	}, func() {
		delete(a.pluginManageState().updating, item.ID)
		a.pluginManageState().notice = fmt.Sprintf(pluginUpdatedFmt, item.Name)
	}, func(msg string) {
		delete(a.pluginManageState().updating, item.ID)
		a.pluginManageState().rowErr[item.ID] = msg
	})
}

// updateAllPlugins 依次更新全部可更新项；部分失败时保留失败行并给汇总原因。
func (a *App) updateAllPlugins(items []ExtItemSnapshot) {
	if a.intents == nil {
		return
	}
	st := a.pluginManageState()
	if st.updatingAll {
		return
	}
	st.updatingAll = true
	st.updateErr = ""
	targets := append([]ExtItemSnapshot(nil), items...)
	a.dispatchQuiet("更新插件", func(ctx context.Context) error {
		var first error
		for _, item := range targets {
			if err := a.intents.UpdatePlugin(ctx, item.ID); err != nil && first == nil {
				first = err
			}
		}
		return first
	}, func() {
		a.pluginManageState().updatingAll = false
		a.pluginManageState().notice = fmt.Sprintf(pluginManageAllDone, len(targets))
	}, func(msg string) {
		a.pluginManageState().updatingAll = false
		a.pluginManageState().updateErr = pluginManageUpdateFail + "：" + msg
	})
}

// openPluginRemove 打开移除确认框。
func (a *App) openPluginRemove(item ExtItemSnapshot) {
	st := a.pluginManageState()
	st.removeID, st.removeName = item.ID, item.Name
	st.removeErr = ""
}

// confirmPluginRemove 卸载一个插件；失败原因留在确认框里。
func (a *App) confirmPluginRemove() {
	st := a.pluginManageState()
	if st.removeID == "" || st.removeBusy || a.intents == nil {
		return
	}
	id, name := st.removeID, st.removeName
	st.removeBusy = true
	st.removeErr = ""
	a.dispatchQuiet("移除插件", func(ctx context.Context) error {
		return a.intents.UninstallExtension(ctx, "plugin", id)
	}, func() {
		a.pluginManageState().removeBusy = false
		a.pluginManageState().removeID = ""
		a.pluginManageState().removeName = ""
		a.pluginManageState().notice = fmt.Sprintf(pluginRemovedFmt, name)
	}, func(msg string) {
		a.pluginManageState().removeBusy = false
		a.pluginManageState().removeErr = msg
	})
}

// pluginRemoveView 画移除确认框。
func (a *App) pluginRemoveView(c *gui.Context) {
	st := a.pluginManageState()
	if st.removeID == "" {
		return
	}
	t := c.Theme()
	name := st.removeName
	if strings.TrimSpace(name) == "" {
		name = st.removeID
	}
	open := true
	m := modalShell(gui.Modal(c, &open, func() {
		panel := gui.Column(c).Gap(space3).Role(gui.RoleAlertDialog).Label(pluginRemoveTitle)
		panel.Children(func() {
			typeDialogTitle.style(t, gui.Text(c, pluginRemoveTitle)).TextColor(t.Text).
				SingleLine().Role(gui.RoleHeading).Level(2)
			typeBody.style(t, gui.Text(c, fmt.Sprintf(pluginRemoveDescFmt, name))).
				TextColor(t.TextMuted)
			if st.removeErr != "" {
				typeSecondary.style(t, gui.Text(c, st.removeErr)).TextColor(t.Danger).
					Role(gui.RoleAlert).Label(st.removeErr)
			}
			gui.Row(c).Justify(gui.End).Gap(space2).Children(func() {
				if pluginDialogButton(c, t, pluginCancel, false, false, st.removeBusy, "") &&
					!st.removeBusy {
					st.removeID, st.removeName, st.removeErr = "", "", ""
				}
				label := pluginManageRemove
				if st.removeBusy {
					label = pluginRemoveBusy
				}
				if pluginDialogButtonLabeled(c, t, label, "确认移除插件", true,
					st.removeBusy, st.removeBusy, "") {
					a.confirmPluginRemove()
				}
			})
		})
		panel.Width(renameDialogWidth(c))
	}))

	if !open || m.Dismissed() {
		if st.removeBusy {
			return
		}
		st.removeID, st.removeName, st.removeErr = "", "", ""
	}
}

// pluginMarketSection 是插件市场块：源地址配置 + 上游清单。未配置源时整块禁用
// 并写「未配置插件源」，不画任何条目。
func (a *App) pluginMarketSection(c *gui.Context, t *gui.Theme) {
	st := a.pluginManageState()
	facts, hasFacts := a.pluginMarketFacts()
	market, canAct := a.pluginMarketIntents()
	if !st.feedLoaded {
		if hasFacts {
			st.feedDraft = facts.FeedURL
		}
		st.feedLoaded = true
	}
	gui.Column(c).Shrink(0).Gap(plgCardInner).Children(func() {
		hairlineBox(c, true, "")
		plgCardTitle.style(t, gui.Text(c, pluginMarketTitle)).TextColor(t.Text).
			SingleLine().Role(gui.RoleHeading).Level(3)
		gui.Column(c).Gap(plgLabelGap).Children(func() {
			plgLabelLine(c, t, pluginMarketFeedLabel, false)
			gui.Row(c).AlignItems(gui.Center).Gap(space2).Children(func() {
				gui.TextInput(c, &st.feedDraft).Label(pluginMarketFeedLabel).
					Placeholder(pluginMarketFeedPlace).Grow(1).Height(plgCtlH).
					Padding(screenInputPadV, screenSpace25)
				label := pluginMarketSave
				if st.feedSaving {
					label = pluginMarketSaving
				}
				if pluginSmallButtonLabeled(c, t, label, "保存插件源", true,
					st.feedSaving, !canAct || st.feedSaving, pluginMarketWriteNote(canAct)) {
					a.savePluginFeed(market)
				}
			})
			if st.feedErr != "" {
				typeSecondary.style(t, gui.Text(c, st.feedErr)).TextColor(t.Danger).
					Role(gui.RoleAlert).Label(st.feedErr)
			}
		})
		switch {
		case !hasFacts:
			// 没有市场读数：整块换成「市场暂不可用」+原因，不画样例条目。
			plgNote(c, t, pluginMarketUnavailable)
			plgNote(c, t, pluginMarketNoFacts)
			return
		case strings.TrimSpace(facts.FeedURL) == "":
			plgNote(c, t, pluginMarketUnset)
			return
		case facts.ConfigError != "":
			typeSecondary.style(t, gui.Text(c, facts.ConfigError)).TextColor(t.Danger).
				Role(gui.RoleAlert).Label(facts.ConfigError)
			return
		}
		gui.Row(c).AlignItems(gui.Center).Gap(space2).Children(func() {
			if facts.Loading {
				plgNote(c, t, pluginMarketLoading)
			} else if pluginSmallButton(c, t, pluginMarketRefresh, false, !canAct,
				pluginMarketWriteNote(canAct)) {
				a.refreshPluginMarket(market)
			}
			gui.Spacer(c)
		})
		if facts.Err != "" {
			gui.Row(c).AlignItems(gui.Center).Gap(space2).Children(func() {
				typeSecondary.style(t, gui.Text(c, facts.Err)).TextColor(t.Danger).
					Role(gui.RoleAlert).Label(facts.Err)
				if pluginSmallButton(c, t, pluginMarketRetry, false, !canAct,
					pluginMarketWriteNote(canAct)) {
					a.refreshPluginMarket(market)
				}
			})
		}
		if len(facts.Entries) == 0 && !facts.Loading && facts.Err == "" {
			plgNote(c, t, pluginMarketEmpty)
			return
		}
		for _, entry := range facts.Entries {
			a.pluginMarketRow(c, t, entry)
		}
	})
}

// pluginMarketRow 是一个市场条目：名称 / 版本 / 说明；「安装」打开详情
// （G-H2）。条目来自上游清单的只读快照，安装动作本身没有协议通道，详情里的
// 主按钮置灰并写原因。
func (a *App) pluginMarketRow(c *gui.Context, t *gui.Theme, entry PluginMarketEntrySnapshot) {
	gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(space2).Children(func() {
		gui.Column(c).Shrink(0).Grow(1).Gap(spaceUnit).Children(func() {
			name := entry.Name
			if entry.Version != "" {
				name += " · " + pluginVersionLabel(entry.Version)
			}
			typeSecondaryStrong.style(t, gui.Text(c, name)).TextColor(t.Text).
				SingleLine().Label(name)
			if entry.Description != "" {
				typeSecondary.style(t, gui.Text(c, entry.Description)).
					TextColor(t.TextMuted).SingleLine()
			}
			if entry.Source != "" {
				typeMicro.style(t, gui.Text(c, entry.Source)).TextColor(t.TextMuted).
					SingleLine()
			}
		})
		// G-H2 市场条目的「安装」入口：有条目事实就打开详情
		if pluginSmallButton(c, t, pluginMarketInstall, false, false, "") {
			st := a.pluginManageState()
			st.marketEntry = entry
			st.marketOpen = true
		}
	})
}

// pluginMarketDetailView 画市场安装详情：标题与描述来自条目与固定文案；技能
// 勾选与安装动作缺事实/协议，按块写原因，不画假的勾选行。
// G-H2-01 标题 / G-H2-02 描述 / G-H2-06 技能清单缺项 / G-H2-08 取消 / G-H2-09 安装
func (a *App) pluginMarketDetailView(c *gui.Context) {
	st := a.pluginManageState()
	if !st.marketOpen {
		return
	}
	t := c.Theme()
	entry := st.marketEntry
	name := strings.TrimSpace(entry.Name)
	if name == "" {
		st.marketOpen = false
		return
	}
	title := fmt.Sprintf(pluginMarketInstallTitleFmt, name)
	open := true
	m := modalShell(gui.Modal(c, &open, func() {
		panel := gui.Column(c).Gap(space3).Role(gui.RoleDialog).Label(title)
		panel.Children(func() {
			// G-H2-01 「安装 {name}」
			typeDialogTitle.style(t, gui.Text(c, title)).TextColor(t.Text).
				SingleLine().Role(gui.RoleHeading).Level(2)
			// G-H2-02 描述
			typeBody.style(t, gui.Text(c, pluginMarketInstallDesc)).TextColor(t.TextMuted)
			meta := entry.Name
			if entry.Version != "" {
				meta += " · " + pluginVersionLabel(entry.Version)
			}
			if entry.Source != "" {
				meta += " · " + entry.Source
			}
			typeSecondary.style(t, gui.Text(c, meta)).TextColor(t.TextMuted).
				Label(meta).SingleLine()
			gui.Column(c).Gap(plgLabelGap).Children(func() {
				// G-H2-05 技能分组：条目没有技能清单，分组标题与勾选行都不画
				plgNote(c, t, pluginMarketNoSkills)
			})
			gui.Row(c).Justify(gui.End).Gap(space2).Children(func() {
				// G-H2-08 取消
				if pluginDialogButton(c, t, pluginCancel, false, false, false, "") {
					st.marketOpen = false
					open = false
				}
				// G-H2-09 添加插件（没有市场安装来源：置灰并写原因）
				pluginDialogButton(c, t, pluginAdd, true, false, true, pluginMarketNoInstall)
			})
		})
		panel.Width(pluginInstallDialogWidth(c))
	}))

	if !open || m.Dismissed() {
		st.marketOpen = false
	}
}

// pluginMarketWriteNote 是市场写入口置灰的原因。
func pluginMarketWriteNote(canAct bool) string {
	if canAct {
		return ""
	}
	return "当前版本不支持安装插件"
}

// savePluginFeed 保存插件源地址并触发一次拉取。
func (a *App) savePluginFeed(market PluginMarketplaceIntents) {
	st := a.pluginManageState()
	if st.feedSaving {
		return
	}
	st.feedSaving = true
	st.feedErr = ""
	feed := st.feedDraft
	a.dispatchQuiet("保存插件源", func(ctx context.Context) error {
		return market.SetPluginFeed(ctx, feed)
	}, func() {
		a.pluginManageState().feedSaving = false
	}, func(msg string) {
		a.pluginManageState().feedSaving = false
		a.pluginManageState().feedErr = msg
	})
}

// refreshPluginMarket 重新拉取一次上游清单。
func (a *App) refreshPluginMarket(market PluginMarketplaceIntents) {
	a.dispatchQuiet("刷新插件市场", func(ctx context.Context) error {
		return market.RefreshPluginMarketplace(ctx)
	}, nil, func(msg string) {
		a.pluginManageState().feedErr = msg
	})
}
