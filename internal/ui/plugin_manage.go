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
	pluginUpdateNoProgress = "当前版本看不到下载进度"

	pluginMarketTitle       = "插件市场"
	pluginMarketFeedLabel   = "插件源"
	pluginMarketFeedPlace   = "https://example.com/plugins.json"
	pluginMarketSave        = "保存"
	pluginMarketSaving      = "正在保存…"
	pluginMarketUnset       = "未配置插件源"
	pluginMarketUnavailable = "市场暂不可用"
	// pluginMarketNoFacts 是「市场暂不可用」的原因：这条连接没有市场读数。
	pluginMarketNoFacts = "当前版本不支持插件市场"
	pluginMarketLoading = "正在加载..."
	pluginMarketEmpty   = "插件源没有返回条目"
	pluginMarketRetry   = "重试"
	// pluginMarketRetryDetail 是详情失败块里「重试」的读屏名：同一屏上的市场块
	// 也有一个「重试」，读屏要能分开这两个入口。
	pluginMarketRetryDetail = "重试加载详情"
	pluginMarketRefresh     = "刷新"
	pluginMarketInstall     = "安装"
	// G-H2-01 安装详情的标题与 G-H2-02 描述（逐字取参考产品中文）。
	pluginMarketInstallTitleFmt = "安装 %s"
	pluginMarketInstallDesc     = "选择要启用的内置技能。默认全选。"
	// G-H2-06 / G-H2-07 技能清单的两种情况分开写：源没有给清单（不画勾选行，写
	// 原因），源给了空清单（画参考产品的空态句「此插件不包含技能。」）。
	pluginMarketNoSkills  = "插件源没有给出这个条目的技能清单，无法勾选技能"
	pluginMarketSelectAll = "全选"
	// 详情的一组只读字段（G-H2-01 标题之下）：名称、版本、作者、来源、安装位置。
	pluginMarketFieldName     = "名称"
	pluginMarketFieldVersion  = "版本"
	pluginMarketFieldAuthor   = "作者"
	pluginMarketFieldSource   = "来源"
	pluginMarketFieldLocation = "安装位置"
	pluginMarketFieldDesc     = "说明"
	// 字段缺失时写原因：不猜、不留空行。
	pluginMarketNoVersion     = "插件源没有给出这个条目的版本"
	pluginMarketNoAuthor      = "插件源没有给出这个条目的作者"
	pluginMarketNoSource      = "插件源没有给出这个条目的来源"
	pluginMarketNoLocation    = "插件源没有给出安装位置"
	pluginMarketNoDescription = "插件源没有给出这个条目的说明"
	// 安装通道的两条门槛：条目没有 Git 地址；同名插件已装（替换确认 G-H3 未建）。
	pluginMarketNoInstallSource = "插件源没有给出可安装的 Git 地址（plugin.install 只接受本地目录或 Git 地址）"
	pluginMarketNameTakenFmt    = "已安装同名插件「%s」，当前版本不支持替换"
	// 详情读不到时主按钮置灰的原因（读失败时框里另有失败块与「重试」）。
	pluginMarketDetailFailed = "插件源读取失败"
	// 安装中的进度：protocol v3 的 plugin.install 只有阶段事件，没有字节读数。
	pluginMarketInstallProgress = "当前版本看不到安装进度"
	// 安装中「取消」置灰的原因（与「不允许半途取消」同口径）。
	pluginMarketCancelBusy = "安装中不能取消"
	// 勾选只决定这次安装的启用清单：plugin.install 没有技能参数。
	pluginMarketSkillPickNote = "勾选只决定安装后启用哪些技能；安装本身装整包。"
	// pluginMarketSkillsMaxH 是技能勾选列表的高度上限：4 行（每行两行文字），
	// 再多就在框里滚，不把弹窗顶出屏幕。
	pluginMarketSkillsMaxH = rowCompact * 6

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
	// 市场安装详情（G-H2）：打开的是哪一个条目、勾选了哪些技能、安装中与失败
	// 原因。条目来自上游清单的只读快照；勾选位是本端可丢的界面状态。
	marketOpen       bool
	marketEntry      PluginMarketEntrySnapshot
	marketSkills     map[string]bool
	marketInstalling bool
	marketErr        string
}

// pluginManage 返回本轮要画的插件管理状态（惰性建映射）。
func (a *App) pluginManageState() *pluginManageState {
	if a.pluginManage.updating == nil {
		a.pluginManage.updating = map[string]bool{}
	}
	if a.pluginManage.rowErr == nil {
		a.pluginManage.rowErr = map[string]string{}
	}
	if a.pluginManage.marketSkills == nil {
		a.pluginManage.marketSkills = map[string]bool{}
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
					a.skillUpdatesSection(c, t)
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
// G-H4-01 标题 / G-H4-02 空态
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
			// G-H4-02 空态（14px 次要色、居中；读屏名就是这一句）
			gui.Row(c).Shrink(0).Justify(gui.Center).Children(func() {
				typeBody.style(t, gui.Text(c, pluginManageNoUpdates)).
					TextColor(t.TextMuted).Label(pluginManageNoUpdates)
			})
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
					// 更新中：没有进度事实，就写清为什么没有进度条（不画假进度）
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

// pluginMarketRow 是一个市场条目：名称 / 版本 / 说明 / 来源；「安装」打开详情
// （G-H2）。条目来自上游清单的只读快照，可安装与否看条目有没有给出 Git 地址。
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
			a.openMarketDetail(entry)
		}
	})
}

// openMarketDetail 打开条目的安装详情：技能勾选位的初值是「全选」。
func (a *App) openMarketDetail(entry PluginMarketEntrySnapshot) {
	st := a.pluginManageState()
	st.marketEntry = entry
	st.marketSkills = map[string]bool{}
	for _, skill := range entry.Skills {
		st.marketSkills[skill.Name] = true
	}
	st.marketInstalling = false
	st.marketErr = ""
	st.marketOpen = true
}

// closeMarketDetail 关掉详情：勾选位、安装中与失败原因一起丢掉（都是可丢状态）。
func (a *App) closeMarketDetail() {
	st := a.pluginManageState()
	st.marketOpen = false
	st.marketEntry = PluginMarketEntrySnapshot{}
	st.marketSkills = map[string]bool{}
	st.marketInstalling = false
	st.marketErr = ""
}

// pluginMarketLookup 在读数里按名字取同一个条目：重拉之后条目内容会变，详情读
// 最新的一份；取不到（重拉中 / 失败）时保留打开时的快照，标题不空。
func pluginMarketLookup(facts PluginMarketplaceSnapshot, name string) (PluginMarketEntrySnapshot, bool) {
	for _, entry := range facts.Entries {
		if entry.Name == name {
			return entry, true
		}
	}
	return PluginMarketEntrySnapshot{}, false
}

// marketInstallProblem 返回「添加插件」被禁用的原因：空串表示可点。
//
// 市场条目没有自己的安装方法：条目给出 Git 地址时走 plugin.install 的 Git 来源；
// 没给地址、没有写通道、同名插件已装（替换确认 G-H3 未建）都禁用并写原因。
func (a *App) marketInstallProblem(entry PluginMarketEntrySnapshot, facts PluginMarketplaceSnapshot,
	hasFacts bool) string {
	st := a.pluginManageState()
	switch {
	case st.marketInstalling:
		return pluginAdding
	case a.intents == nil:
		return noteNoWriteChannel
	case !hasFacts:
		return pluginMarketNoFacts
	case facts.Loading:
		return pluginMarketLoading
	case facts.Err != "":
		return pluginMarketDetailFailed
	case strings.TrimSpace(entry.Git) == "":
		return pluginMarketNoInstallSource
	}
	if plugins, ok := a.pluginList(); ok {
		for _, item := range plugins.Items {
			if item.Name == entry.Name {
				return fmt.Sprintf(pluginMarketNameTakenFmt, entry.Name)
			}
		}
	}
	return ""
}

// confirmMarketInstall 装一个市场条目：走 plugin.install 的 Git 来源，装后启用。
// 技能勾选没有对应的写参数（plugin.install 没有技能参数），所以勾选只决定这次
// 安装的启用清单，界面在勾选区写清这一点。
func (a *App) confirmMarketInstall(entry PluginMarketEntrySnapshot, facts PluginMarketplaceSnapshot,
	hasFacts bool) {
	if a.marketInstallProblem(entry, facts, hasFacts) != "" {
		return
	}
	st := a.pluginManageState()
	name := entry.Name
	st.marketInstalling = true
	st.marketErr = ""
	a.dispatchQuiet("添加插件", func(ctx context.Context) error {
		return a.intents.InstallPlugin(ctx, "git", strings.TrimSpace(entry.Git),
			strings.TrimSpace(entry.GitRef), "", true)
	}, func() {
		a.closeMarketDetail()
		a.pluginManageState().notice = fmt.Sprintf(pluginInstalledFmt, name)
	}, func(msg string) {
		st := a.pluginManageState()
		st.marketInstalling = false
		st.marketErr = msg
	})
}

// pluginMarketDetailView 画市场安装详情（G-H2）：标题、描述、字段、加载与失败
// 块、技能勾选行、安装 / 取消按钮，以及安装中与成功、失败回执。
// G-H2-01 标题 / G-H2-02 描述 / G-H2-08 取消 / G-H2-09 添加插件
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
	facts, hasFacts := a.pluginMarketFacts()
	if hasFacts {
		if current, ok := pluginMarketLookup(facts, name); ok {
			entry = current
		}
	}
	problem := a.marketInstallProblem(entry, facts, hasFacts)
	title := fmt.Sprintf(pluginMarketInstallTitleFmt, name)
	open := true
	m := modalShell(gui.Modal(c, &open, func() {
		panel := gui.Column(c).Gap(space3).Role(gui.RoleDialog).Label(title)
		panel.Children(func() {
			// G-H2-01 「安装 {name}」：标题只读条目名，不等详情
			typeDialogTitle.style(t, gui.Text(c, title)).TextColor(t.Text).
				SingleLine().Role(gui.RoleHeading).Level(2)
			// G-H2-02 描述
			typeBody.style(t, gui.Text(c, pluginMarketInstallDesc)).TextColor(t.TextMuted)
			a.pluginMarketDetailFields(c, t, entry)
			a.pluginMarketDetailState(c, t, facts, hasFacts)
			a.pluginMarketSkills(c, t, entry)
			// 安装中：主按钮前置旋转图标；进度条没有读数，写原因
			if st.marketInstalling {
				typeMicro.style(t, gui.Text(c, pluginMarketInstallProgress)).
					TextColor(t.TextMuted).Label(pluginMarketInstallProgress).SingleLine()
			}
			// 失败回执：原因留在框里，按钮恢复可点（重试就是再点一次）
			if st.marketErr != "" {
				typeSecondary.style(t, gui.Text(c, st.marketErr)).TextColor(t.Danger).
					Role(gui.RoleAlert).Label(st.marketErr)
			}
			// 主按钮置灰的原因：加载中与安装中已经在别处写清，其余原因在这里写一行
			if problem != "" && problem != pluginMarketLoading && problem != pluginAdding {
				plgNote(c, t, problem)
			}
			gui.Row(c).Justify(gui.End).Gap(space2).Children(func() {
				// G-H2-08 取消（安装中禁用，不允许半途取消）
				if pluginDialogButton(c, t, pluginCancel, false, false, st.marketInstalling,
					pluginMarketCancelBusy) && !st.marketInstalling {
					a.closeMarketDetail()
					open = false
				}
				// G-H2-09 添加插件
				label := pluginAdd
				if st.marketInstalling {
					label = pluginAdding
				}
				if pluginDialogButton(c, t, label, true, st.marketInstalling,
					problem != "", problem) {
					a.confirmMarketInstall(entry, facts, hasFacts)
				}
			})
		})
		panel.Width(pluginInstallDialogWidth(c))
	}))

	// G-H2-08 键盘：Modal 内建 Esc，焦点不在按钮上也照样关框；安装中不生效
	// （与「取消」置灰同口径）。这里只在弹窗自己身上接一次，不吃全局键位。
	if m.OverlayShortcut(0, gui.KeyEscape) && !st.marketInstalling {
		open = false
	}
	if !open || m.Dismissed() {
		if st.marketInstalling {
			return
		}
		a.closeMarketDetail()
	}
}

// pluginMarketDetailFields 是详情的一组只读字段：名称、版本、作者、来源、安装
// 位置与说明；缺一项就写该项的原因，不猜也不留空行。
func (a *App) pluginMarketDetailFields(c *gui.Context, t *gui.Theme, entry PluginMarketEntrySnapshot) {
	version := ""
	if strings.TrimSpace(entry.Version) != "" {
		version = pluginVersionLabel(entry.Version)
	}
	gui.Column(c).Shrink(0).Gap(spaceUnit).Children(func() {
		pluginMarketField(c, t, pluginMarketFieldName, entry.Name, "")
		pluginMarketField(c, t, pluginMarketFieldVersion, version, pluginMarketNoVersion)
		pluginMarketField(c, t, pluginMarketFieldAuthor, entry.Author, pluginMarketNoAuthor)
		pluginMarketField(c, t, pluginMarketFieldSource, entry.Source, pluginMarketNoSource)
		pluginMarketField(c, t, pluginMarketFieldLocation, entry.Location, pluginMarketNoLocation)
		plgLabelStyle.style(t, gui.Text(c, pluginMarketFieldDesc)).TextColor(t.TextMuted).
			Label(pluginMarketFieldDesc)
		if strings.TrimSpace(entry.Description) == "" {
			typeSecondary.style(t, gui.Text(c, pluginMarketNoDescription)).
				TextColor(t.TextMuted).Label(pluginMarketNoDescription)
			return
		}
		typeSecondary.style(t, gui.Text(c, entry.Description)).TextColor(t.TextMuted).
			Selectable().MaxLines(3).Label(pluginMarketFieldDesc)
	})
}

// pluginMarketField 一行只读字段：12.25 的标签 + 值；值缺失时写原因。
func pluginMarketField(c *gui.Context, t *gui.Theme, label, value, missing string) {
	gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(space2).Children(func() {
		plgLabelStyle.style(t, gui.Text(c, label)).TextColor(t.TextMuted).Label(label)
		if strings.TrimSpace(value) == "" {
			typeSecondary.style(t, gui.Text(c, missing)).TextColor(t.TextMuted).
				Label(missing)
			return
		}
		typeSecondaryStrong.style(t, gui.Text(c, value)).TextColor(t.Text).
			SingleLine().Label(label + "：" + value)
	})
}

// pluginMarketDetailState 是详情里的加载行与失败块（G-H2-03 / G-H2-04）：读不到
// 详情时说清是「正在加载」还是「加载失败」，失败块里的「重试」真的重拉一次清单。
func (a *App) pluginMarketDetailState(c *gui.Context, t *gui.Theme, facts PluginMarketplaceSnapshot,
	hasFacts bool) {
	if !hasFacts || (!facts.Loading && facts.Err == "") {
		return
	}
	if facts.Loading {
		// G-H2-03 「正在加载...」：旋转图标 + 14px 灰字，居中
		gui.Row(c).Shrink(0).Justify(gui.Center).AlignItems(gui.Center).
			Gap(space1).Children(func() {
			gui.Spinner(c).Size(plgIconSize, plgIconSize)
			typeSecondary.style(t, gui.Text(c, pluginMarketLoading)).TextColor(t.TextMuted).
				Label(pluginMarketLoading)
		})
		return
	}
	// G-H2-04 错误块：告警图标 + 错误文案 + 次级按钮「重试」
	market, canAct := a.pluginMarketIntents()
	gui.Row(c).Shrink(0).AlignItems(gui.Center).Wrap().Gap(space2).Children(func() {
		accIcon(c, accIconTriangleAlert, plgIconSize, t.Danger, "triangle-alert")
		typeSecondary.style(t, gui.Text(c, facts.Err)).TextColor(t.Danger).
			Role(gui.RoleAlert).Label(facts.Err)
		gui.Spacer(c)
		if pluginSmallButtonLabeled(c, t, pluginMarketRetry, pluginMarketRetryDetail,
			false, false, !canAct, pluginMarketWriteNote(canAct)) {
			a.refreshPluginMarket(market)
		}
	})
}

// pluginMarketSkills 是详情里的技能清单：分组标签（G-H2-05）、勾选行
// （G-H2-06，默认全选）、无技能的空态（G-H2-07），以及「源没给清单」的说明。
func (a *App) pluginMarketSkills(c *gui.Context, t *gui.Theme, entry PluginMarketEntrySnapshot) {
	st := a.pluginManageState()
	if entry.Skills == nil {
		// 源没给清单：不断言「不包含技能」，写原因
		plgNote(c, t, pluginMarketNoSkills)
		return
	}
	if len(entry.Skills) == 0 {
		// G-H2-07 空态句（14px 灰）
		plgNote(c, t, pluginEmptySkills)
		return
	}
	gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(space2).Children(func() {
		// G-H2-05 「技能」分组标签（12.25 中等灰）
		plgLabelStyle.style(t, gui.Text(c, pluginSkillsEntry)).TextColor(t.TextMuted).
			Label(pluginSkillsEntry)
		gui.Spacer(c)
		pluginMarketSelectAllBox(c, t, entry, st)
	})
	// 勾选行多了就滚：高度上限受弹窗约束（G-H2-06）
	gui.Scroll(c).Shrink(0).MaxHeight(pluginMarketSkillsMaxH).Children(func() {
		gui.Column(c).Shrink(0).Gap(space2).Children(func() {
			for _, skill := range entry.Skills {
				pluginMarketSkillRow(c, t, st, skill)
			}
		})
	})
	plgNote(c, t, pluginMarketSkillPickNote)
}

// pluginMarketSelectAllBox 是技能的全选位：全选 / 部分 / 全不选三态，点一次改全部。
func pluginMarketSelectAllBox(c *gui.Context, t *gui.Theme, entry PluginMarketEntrySnapshot,
	st *pluginManageState) {
	selected := 0
	for _, skill := range entry.Skills {
		if st.marketSkills[skill.Name] {
			selected++
		}
	}
	state := gui.CheckOff
	switch {
	case selected == len(entry.Skills):
		state = gui.CheckOn
	case selected > 0:
		state = gui.CheckMixed
	}
	box := gui.CheckboxStateBase(c, &state).Size(plgIconSize, plgIconSize).
		Shrink(0).Radius(radius).Label(pluginMarketSelectAll)
	if st.marketInstalling {
		box = box.Disabled(true)
	}
	if box.Changed() {
		next := state == gui.CheckOn
		for _, skill := range entry.Skills {
			st.marketSkills[skill.Name] = next
		}
	}
	box.Children(func() {
		box.DrawOver(func(p *gui.Painter, r gui.Rect) {
			switch state {
			case gui.CheckOn, gui.CheckMixed:
				p.Fill(r, t.Accent, radius)
				icon := iconCheck
				if state == gui.CheckMixed {
					icon = iconDot
				}
				paintIcon(p, icon, r, t.AccentText)
			default:
				p.Stroke(r, t.Border, radius, strokeHairline)
			}
		})
	})
}

// pluginMarketSkillRow 是一条技能勾选行（G-H2-06）：方框 + 名称 + 描述；点行
// （不只是方框）就切换该技能的勾选。勾选只决定安装后的启用清单。
//
// 整行是一个复选项：读屏把名称与描述一起读出来（G-H2-06 无障碍），所以描述那
// 一行只给眼睛看（对读屏隐藏），同一个方框也只是画出来——输入由整行接。
func pluginMarketSkillRow(c *gui.Context, t *gui.Theme, st *pluginManageState,
	skill PluginMarketSkillSnapshot) {
	on := st.marketSkills[skill.Name]
	description := strings.TrimSpace(skill.Description)
	label := skill.Name
	if description != "" {
		label += "，" + description
	}
	row := gui.ButtonBase(c).Shrink(0).Grow(1).Gap(space2).AlignItems(gui.Center).
		Role(gui.RoleCheckBox).Checked(on).Label(label).Cursor(gui.CursorPointer)
	if st.marketInstalling {
		row = row.Disabled(true).Opacity(0.5)
	}
	if row.Clicked() {
		on = !on
		st.marketSkills[skill.Name] = on
	}
	if row.Hovered() && !st.marketInstalling {
		row = row.Background(t.SurfaceHover)
	}
	row.Children(func() {
		pluginMarketCheckFace(c, t, on)
		gui.Column(c).Shrink(0).Grow(1).Gap(spaceUnit).Children(func() {
			typeBodyStrong.style(t, gui.Text(c, skill.Name)).TextColor(t.Text).SingleLine()
			if description != "" {
				typeSecondary.style(t, gui.Text(c, description)).TextColor(t.TextMuted).
					MaxLines(2).AccessHidden(true)
			}
		})
	})
}

// pluginMarketCheckFace 是勾选框的方框：勾上时实心主色 + 勾，未勾时描边。它只是
// 外观（输入由整行接），所以对读屏隐藏，也不自己吃点击。
func pluginMarketCheckFace(c *gui.Context, t *gui.Theme, on bool) {
	box := gui.Box(c).Size(plgIconSize, plgIconSize).Shrink(0).
		Radius(radius).AccessHidden(true)
	box.DrawOver(func(p *gui.Painter, r gui.Rect) {
		if !on {
			p.Stroke(r, t.Border, radius, strokeHairline)
			return
		}
		p.Fill(r, t.Accent, radius)
		paintIcon(p, iconCheck, r, t.AccentText)
	})
}

// pluginMarketWriteNote 是市场写入口置灰的原因。
func pluginMarketWriteNote(canAct bool) string {
	if canAct {
		return ""
	}
	return "当前版本不支持修改插件源"
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
