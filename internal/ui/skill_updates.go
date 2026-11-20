package ui

import (
	"context"
	"fmt"
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是设置 › 插件设置里的「技能更新」块与它的弹窗（G-H5）：
//
//	技能更新（入口 + 摘要 / 空态 / 缺读数的原因）
//	可用更新弹窗（标题、逐行技能与版本、来源、变更说明、逐项更新、全部更新、
//	              更新中、失败与重试）
//
// 读的是可选的 ui.SkillUpdatesFacts；写走可选的 ui.SkillUpdateIntents
// （skills.update，按安装记录标识）。两者都是可选接口：protocol v3 没有「哪些
// 技能有新版本」的读取方法，真实的 uiwire 适配器也没有写通道，因此这两种情况下
// 入口与按钮置灰并写原因，不假装「都最新」，也不发一次不会成功的写入。
//
// 与参考产品的差异：参考的技能更新是技能市场页里的弹窗（SkillUpdatesDialog），
// 本版没有独立的技能路由页（见 plugin_manage.go 的文件头），入口画在设置 ›
// 插件设置的同一张卡里，插件更新区（G-H4）保持原地展开。

const (
	// skillUpdatesSectionTitle 是管理块里这一段的标题（本版新增的落点标题）。
	skillUpdatesSectionTitle = "技能更新"
	// G-H5-01 弹窗标题，逐字取参考产品中文。
	skillUpdatesTitle = "可用更新"
	// G-H5-02 空态，逐字取参考产品中文。
	skillUpdatesEmpty = "所有技能均为最新版本"
	// G-H5-05 逐项更新按钮：可见文字「更新」，读屏名「更新 {技能名}」。
	skillUpdatesUpdate    = "更新"
	skillUpdatesUpdateFmt = "更新 %s"
	// G-H5-06 全部更新（{count}），读屏名「全部更新，共 N 个」。
	skillUpdatesUpdateAllFmt      = "全部更新（%d）"
	skillUpdatesUpdateAllLabelFmt = "全部更新，共 %d 个"
	skillUpdatesUpdating          = "更新中…"
	skillUpdatesAllUpdating       = "正在更新…"
	skillUpdatesClose             = "关闭"
	// 更新中关框会把回执与失败原因一起丢掉：先等这一批结束。
	skillUpdatesCloseBusy = "正在更新，等这一批结束再关"

	// 入口按钮的可见文字与读屏名分开：可见的「可用更新」是参考里这个弹窗的名字，
	// 读屏名说明点它会发生什么。
	skillUpdatesOpenLabel = "打开技能可用更新"

	// 缺读数 / 缺通道的原因，逐块写清。
	skillUpdatesNoFacts   = "这条连接没有技能更新读数（protocol v3 没有「技能有新版本」的读取方法）"
	skillUpdatesNoChannel = "当前连接没有技能更新通道（skills.update）"
	skillUpdatesReadError = "读取技能更新失败"
	skillUpdatesCountFmt  = "%d 个技能有可用更新"

	// 行内三段信息：来源、变更说明、版本缺失时的说明。
	skillUpdatesSourceFmt = "来源：%s"
	skillUpdatesNotesFmt  = "变更：%s"
	skillUpdatesNoSource  = "来源没有给出安装来源"
	skillUpdatesNoNotes   = "来源没有给出变更说明"

	// 回执与失败汇总。
	skillUpdatesDoneFmt    = "已更新 %s"
	skillUpdatesAllDoneFmt = "已更新 %d 个技能"
	skillUpdatesFail       = "更新失败"
	skillUpdatesFailedFmt  = "%d 个技能更新失败"

	// skillUpdateIconBox / skillUpdateIconGlyph 是行首图标：40 的底托 + 16 的占位
	// 图标（没有技能图标时用占位，不显示破图）。
	skillUpdateIconBox   = 40
	skillUpdateIconGlyph = 16
	// skillUpdatesMaxH / skillUpdatesNarrowMaxH 是弹窗里列表的高度上限：超过就滚，
	// 不把弹窗顶出屏幕；窄窗里再矮一档（行与按钮仍完整在屏上）。
	skillUpdatesMaxH       float32 = rowCompact * 8
	skillUpdatesNarrowMaxH         = rowCompact * 5
)

// skillUpdatesState 是这一块的可丢视图状态。
type skillUpdatesState struct {
	open bool
	// updating 按安装记录标识记「这一行正在更新」；rowErr 是逐行失败原因。
	updating    map[string]bool
	rowErr      map[string]string
	updatingAll bool
	allErr      string
	notice      string
}

// skillUpdatesStateOf 返回本轮要画的状态（惰性建映射）。
func (a *App) skillUpdatesStateOf() *skillUpdatesState {
	if a.skillUpdates.updating == nil {
		a.skillUpdates.updating = map[string]bool{}
	}
	if a.skillUpdates.rowErr == nil {
		a.skillUpdates.rowErr = map[string]string{}
	}
	return &a.skillUpdates
}

// skillUpdatesKnown 取技能更新读数：Source 实现可选接口时才有。
func (a *App) skillUpdatesKnown() (SkillUpdatesSnapshot, bool) {
	if a.source == nil {
		return SkillUpdatesSnapshot{}, false
	}
	facts, ok := a.source.(SkillUpdatesFacts)
	if !ok {
		return SkillUpdatesSnapshot{}, false
	}
	return facts.SkillUpdates(), true
}

// skillUpdatesWriter 取技能更新写通道：Intents 实现可选接口时才有。
func (a *App) skillUpdatesWriter() (SkillUpdateIntents, bool) {
	writer, ok := a.intents.(SkillUpdateIntents)
	return writer, ok
}

// openSkillUpdates 打开「可用更新」弹窗；失败原因与回执从干净状态开始。
func (a *App) openSkillUpdates() {
	st := a.skillUpdatesStateOf()
	st.open = true
	st.rowErr = map[string]string{}
	st.allErr = ""
	st.updatingAll = false
}

// skillUpdatesSection 是管理块里的「技能更新」：标题、摘要与打开弹窗的入口；
// 没有读数、读取失败或没有写通道时，入口置灰并写原因。
func (a *App) skillUpdatesSection(c *gui.Context, t *gui.Theme) {
	st := a.skillUpdatesStateOf()
	facts, known := a.skillUpdatesKnown()
	items := len(facts.Items)
	// 入口的禁用只看「有没有读数」：没有可更新项时仍可打开（弹窗里给空态，
	// G-H5-02 的前提就是「打开时无待更新项」），读取失败时也能打开（弹窗里给
	// 失败提示，G-H5-01 异常）。
	problem := ""
	if !known {
		problem = skillUpdatesNoFacts
	}
	gui.Column(c).Shrink(0).Gap(plgCardInner).Children(func() {
		hairlineBox(c, true, "")
		gui.Row(c).AlignItems(gui.Center).Gap(space2).Children(func() {
			plgCardTitle.style(t, gui.Text(c, skillUpdatesSectionTitle)).TextColor(t.Text).
				SingleLine().Role(gui.RoleHeading).Level(3)
			gui.Spacer(c)
			// 入口：可见文字是弹窗的名字，读屏名说明动作
			if pluginSmallButtonLabeled(c, t, skillUpdatesTitle, skillUpdatesOpenLabel,
				false, false, problem != "", problem) {
				a.openSkillUpdates()
			}
		})
		switch {
		case !known:
			plgNote(c, t, skillUpdatesNoFacts)
		case facts.ReadError != "":
			typeSecondary.style(t, gui.Text(c, facts.ReadError)).TextColor(t.Danger).
				Role(gui.RoleAlert).Label(facts.ReadError)
			plgNote(c, t, skillUpdatesReadError)
		case items == 0:
			// G-H5-02 空态句：与弹窗里同一句，不另编一句「没有可更新的技能」
			plgNote(c, t, skillUpdatesEmpty)
		default:
			plgNote(c, t, fmt.Sprintf(skillUpdatesCountFmt, items))
		}
		if st.notice != "" {
			typeSecondaryStrong.style(t, gui.Text(c, st.notice)).TextColor(t.Success).
				Label(st.notice)
		}
	})
}

// skillUpdatesDialog 是「可用更新」弹窗。
// G-H5-01 标题 / G-H5-02 空态 / G-H5-03 行首图标与名称 / G-H5-04 版本行 /
// G-H5-05 更新 / G-H5-06 全部更新
func (a *App) skillUpdatesDialog(c *gui.Context) {
	st := a.skillUpdatesStateOf()
	if !st.open {
		return
	}
	t := c.Theme()
	facts, known := a.skillUpdatesKnown()
	items := facts.Items
	_, canWrite := a.skillUpdatesWriter()
	open := true
	m := modalShell(gui.Modal(c, &open, func() {
		panel := gui.Column(c).Gap(space3).Role(gui.RoleDialog).Label(skillUpdatesTitle)
		panel.Children(func() {
			// G-H5-01 「可用更新」
			typeDialogTitle.style(t, gui.Text(c, skillUpdatesTitle)).TextColor(t.Text).
				SingleLine().Role(gui.RoleHeading).Level(2)
			switch {
			case !known:
				// 没有读数：说清缺什么，不画样例行
				plgNote(c, t, skillUpdatesNoFacts)
			case facts.ReadError != "":
				typeSecondary.style(t, gui.Text(c, facts.ReadError)).TextColor(t.Danger).
					Role(gui.RoleAlert).Label(facts.ReadError)
				plgNote(c, t, skillUpdatesReadError)
			case len(items) == 0:
				// G-H5-02 空态文字（14px 次要色、居中）
				gui.Row(c).Shrink(0).Justify(gui.Center).Children(func() {
					plgNote(c, t, skillUpdatesEmpty)
				})
			}
			if len(items) > 0 {
				// 行多了就滚：高度上限受弹窗约束；窄窗里再矮一档，行与按钮都留在屏上
				maxH := skillUpdatesMaxH
				if narrowWindow(c) {
					maxH = skillUpdatesNarrowMaxH
				}
				gui.Scroll(c).Shrink(0).MaxHeight(maxH).Children(func() {
					gui.Column(c).Shrink(0).Gap(space3).Children(func() {
						for _, item := range items {
							a.skillUpdateRow(c, t, item, canWrite)
						}
					})
				})
			}
			// 失败汇总：逐行原因留在行里，这里只写有几行没过（重试就是再点一次）
			if st.allErr != "" {
				typeSecondary.style(t, gui.Text(c, st.allErr)).TextColor(t.Danger).
					Role(gui.RoleAlert).Label(st.allErr)
			}
			// 没有写通道：按钮置灰，原因也写一行（不只留在 tooltip 里）
			if !canWrite {
				plgNote(c, t, skillUpdatesNoChannel)
			}
			// 成功回执：逐项或全部更新完成后写一句，读数带走已更新的行
			if st.notice != "" {
				typeSecondaryStrong.style(t, gui.Text(c, st.notice)).TextColor(t.Success).
					Label(st.notice)
			}
			gui.Row(c).Justify(gui.End).Gap(space2).Children(func() {
				// 关闭：更新中不许关（与插件的移除确认框同口径）
				if pluginDialogButton(c, t, skillUpdatesClose, false, false,
					st.updatingAll, skillUpdatesCloseBusy) {
					st.open = false
					open = false
				}
				if len(items) > 0 {
					// G-H5-06 「全部更新（{count}）」；读屏名「全部更新，共 N 个」
					text := fmt.Sprintf(skillUpdatesUpdateAllFmt, len(items))
					label := fmt.Sprintf(skillUpdatesUpdateAllLabelFmt, len(items))
					if st.updatingAll {
						text, label = skillUpdatesAllUpdating, skillUpdatesAllUpdating
					}
					if pluginDialogButtonLabeled(c, t, text, label, true, st.updatingAll,
						!canWrite || st.updatingAll, skillUpdatesChannelNote(canWrite)) {
						a.updateAllSkills(items)
					}
				}
			})
		})
		panel.Width(pluginInstallDialogWidth(c))
	}))

	if !open || m.Dismissed() {
		if st.updatingAll {
			return
		}
		st.open = false
	}
}

// skillUpdateRow 是一行可更新技能：40px 图标 + 名称、版本行、来源、变更说明，
// 右端逐项「更新」。
// G-H5-03 图标与名称 / G-H5-04 版本行 / G-H5-05 更新
func (a *App) skillUpdateRow(c *gui.Context, t *gui.Theme, item SkillUpdateSnapshot,
	canWrite bool) {
	st := a.skillUpdatesStateOf()
	busy := st.updating[item.ID]
	gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(space2).Children(func() {
		skillUpdateIcon(c, t)
		gui.Column(c).Shrink(0).Grow(1).Gap(spaceUnit).Children(func() {
			// G-H5-03 名称（14px，单行截断；读屏名是技能名）
			typeH4.style(t, gui.Text(c, item.Name)).TextColor(t.Text).
				SingleLine().Label(item.Name)
			// G-H5-04 版本行：有记录版本时「v{旧} → v{新}」，旧版本未知只写「v{新}」；
			// 新版本也没有就不画这一行。
			version := skillUpdateVersionLine(item)
			if version != "" {
				typeSecondary.style(t, gui.Text(c, version)).TextColor(t.TextMuted).
					Label(version)
			}
			if strings.TrimSpace(item.Source) != "" {
				source := fmt.Sprintf(skillUpdatesSourceFmt, item.Source)
				typeMicro.style(t, gui.Text(c, source)).TextColor(t.TextMuted).
					SingleLine().Label(source)
			} else {
				typeMicro.style(t, gui.Text(c, skillUpdatesNoSource)).TextColor(t.TextMuted).
					SingleLine().Label(skillUpdatesNoSource)
			}
			if strings.TrimSpace(item.Notes) != "" {
				notes := fmt.Sprintf(skillUpdatesNotesFmt, item.Notes)
				typeSecondary.style(t, gui.Text(c, notes)).TextColor(t.TextMuted).
					MaxLines(2).Label(notes)
			} else {
				typeMicro.style(t, gui.Text(c, skillUpdatesNoNotes)).TextColor(t.TextMuted).
					MaxLines(2).Label(skillUpdatesNoNotes)
			}
			if msg := st.rowErr[item.ID]; msg != "" {
				typeSecondary.style(t, gui.Text(c, msg)).TextColor(t.Danger).
					Role(gui.RoleAlert).Label(msg)
			}
		})
		// G-H5-05 「更新」：安装中或全部更新中禁用，更新中前置旋转图标；
		// 读屏名带技能名（同一屏里每行都有这个按钮）。
		label := fmt.Sprintf(skillUpdatesUpdateFmt, item.Name)
		text := skillUpdatesUpdate
		if busy {
			text, label = skillUpdatesUpdating, skillUpdatesUpdating
		}
		if pluginSmallButtonLabeled(c, t, text, label, false, busy,
			!canWrite || busy || st.updatingAll, skillUpdatesChannelNote(canWrite)) {
			a.updateSkill(item)
		}
	})
}

// skillUpdateIcon 是行首 40px 底托 + 16px 占位图标（G-H5-03：没有技能图标时不
// 显示破图，也不需要网络）。
func skillUpdateIcon(c *gui.Context, t *gui.Theme) {
	box := gui.Box(c).Size(skillUpdateIconBox, skillUpdateIconBox).Shrink(0).
		Radius(radiusCard).Background(t.Surface).Label("技能图标").AccessHidden(true)
	box.DrawOver(func(p *gui.Painter, r gui.Rect) {
		inner := gui.Rect{
			X: r.X + (r.W-skillUpdateIconGlyph)/2,
			Y: r.Y + (r.H-skillUpdateIconGlyph)/2,
			W: skillUpdateIconGlyph,
			H: skillUpdateIconGlyph,
		}
		paintIcon(p, iconSparkles, inner, t.TextMuted)
	})
}

// skillUpdateVersionLine 是版本行：有旧版本时「v旧 → v新」，旧版本未知时只写
// 「v新」；服务端没有给新版本时返回空串（不画这一行）。
func skillUpdateVersionLine(item SkillUpdateSnapshot) string {
	latest := strings.TrimSpace(item.LatestVersion)
	if latest == "" {
		return ""
	}
	current := strings.TrimSpace(item.Version)
	if current == "" {
		return pluginVersionLabel(latest)
	}
	return pluginVersionLabel(current) + " → " + pluginVersionLabel(latest)
}

// skillUpdatesChannelNote 是更新入口置灰的原因。
func skillUpdatesChannelNote(canWrite bool) string {
	if canWrite {
		return ""
	}
	return skillUpdatesNoChannel
}

// updateSkill 更新一条安装记录；失败原因留在该行，成功的行由随后的读数带走。
func (a *App) updateSkill(item SkillUpdateSnapshot) {
	writer, ok := a.skillUpdatesWriter()
	st := a.skillUpdatesStateOf()
	if !ok || st.updating[item.ID] || st.updatingAll {
		return
	}
	st.updating[item.ID] = true
	delete(st.rowErr, item.ID)
	st.allErr = ""
	a.dispatchQuiet("更新技能", func(ctx context.Context) error {
		return writer.UpdateSkill(ctx, item.ID)
	}, func() {
		st := a.skillUpdatesStateOf()
		delete(st.updating, item.ID)
		st.notice = fmt.Sprintf(skillUpdatesDoneFmt, item.Name)
	}, func(msg string) {
		st := a.skillUpdatesStateOf()
		delete(st.updating, item.ID)
		st.rowErr[item.ID] = msg
	})
}

// updateAllSkills 依次更新全部可更新项：逐行发出，失败的行留在列表里并写原因，
// 其余的行照样更新（服务端按安装记录逐条写，没有整批回滚，也不假装成功）。
func (a *App) updateAllSkills(items []SkillUpdateSnapshot) {
	writer, ok := a.skillUpdatesWriter()
	st := a.skillUpdatesStateOf()
	if !ok || st.updatingAll {
		return
	}
	st.updatingAll = true
	st.allErr = ""
	st.notice = ""
	st.rowErr = map[string]string{}
	targets := append([]SkillUpdateSnapshot(nil), items...)
	// failed 只在这一条后台协程里写；失败回调在渲染线程上读（runIntent 的锁给出
	// 前后关系）。
	failed := make([]SkillUpdateSnapshot, 0, len(targets))
	reasons := make([]string, 0, len(targets))
	a.dispatchQuiet("更新技能", func(ctx context.Context) error {
		for _, item := range targets {
			if err := writer.UpdateSkill(ctx, item.ID); err != nil {
				failed = append(failed, item)
				reasons = append(reasons, err.Error())
			}
		}
		if len(failed) > 0 {
			return fmt.Errorf(skillUpdatesFailedFmt, len(failed))
		}
		return nil
	}, func() {
		st := a.skillUpdatesStateOf()
		st.updatingAll = false
		st.notice = fmt.Sprintf(skillUpdatesAllDoneFmt, len(targets))
	}, func(msg string) {
		st := a.skillUpdatesStateOf()
		st.updatingAll = false
		st.allErr = msg
		for i, item := range failed {
			st.rowErr[item.ID] = skillUpdatesFail + "：" + reasons[i]
		}
	})
}
