package ui

import (
	"context"
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「设置 › 权限」的重建屏（参考产品里这一节的组件叫 SandboxSettings）：
// 元素按 docs/domains/requirements/inventory-settings.md §S-sandbox（源码穷举
// S-sandbox-01 … 40，40 个）逐行对应，可见态与位置取 inventory-settings-rt.md
// §13（S-permissions-01 … 30）；行为对应 requirements-settings.md 的
// R-PRM-001 … 240。
//
// 数值取 docs/domains/requirements/design-baseline.md（页面标题 17.5、分组标题 16、
// 小标题 13、标签 13、说明 12.25、开关 28×16、表头高 28、控件高 28/32、
// 卡片圆角 8.75）与运行时截图的实测（标题下 27、卡内 15、行距 14、分隔线上 29 /
// 下 22、表头列宽 217、规则行高 36）。
//
// 只保留可丢的界面状态（permissionsScreen）：两个开关与其已保存值、加载 / 后端
// 不可用的分支、规则草稿与行内校验、反馈与离开确认框。事实（进程隔离设置与
// Exec command 规则）归 App Server：读取经 Source、写入经 Intents
// （WriteCommandRules 已有，隔离设置尚无对应方法）；本屏不写事实。
//
// # 与参考实现的差异
//
//   - 表头列宽：参考实现的三列在 741 宽下各 217（第四列「规则」只给读屏）。本屏
//     取 217 的定宽 + 两列均分剩余宽度；窄窗（内容列 < SettingsRuleRowMinWidth）
//     按 U2-forms F23 的退化规则堆叠成两行（R-PRM-192）。
//   - 规则行的输入框用系统等宽字体（U2 §1.3），参考实现同。
//   - 沙箱不可用提示条（S-permissions-07）用警示色文字，没有琥珀底：与账户屏的
//     到期条同口径（不新增第二套警示面）。
//
// # 接了什么
//
//   - 规则表（S-permissions-18 … 31）：增删改都是草稿，**保存权限**（S-permissions-27）
//     接 Intents.WriteCommandRules，成功写「已保存。」，失败把原因写进错误行；
//     规则与版本前置条件由 Source.Settings().Rules 给（读到就按它铺草稿，读失败
//     就写原因）。冲突（服务端版本变了）不重试，直接把原因摆出来重读。
//   - 保存成功后重新按事实铺一次草稿：屏上留下的是服务端接受的规则，不是本地那份。
//
// # 未实现分支
//
//   - 进程隔离开关与它的保存（S-permissions-08 … 16）：服务端不提供这组设置的
//     读取，读不到当前值就不摆一个会漂的开关；连接态置灰并写原因。
//   - 刷新（S-permissions-06）：规则读数由连接层按需重读，本屏没有手动刷新通道，
//     那个入口只在读失败时出现，置灰并写原因。
//   - 离开拦截（S-permissions-36 … 40）：对话框在，但"切到别的分节前拦一次"
//     要有设置壳的离开钩子（settings.go 的 settingsContent 直接换屏），本屏
//     触发不到；判定保持原样，等接线时由壳调用。
//   - 语言包缺失的回落（R-PRM-005/021 …）：文案直接取参考产品中文，无 i18n 层。

const (
	// 权限屏几何：design-baseline 只给常量，元素级包围盒取运行时 png 的边框行
	// 与 inventory-settings-rt.md §13 的 DOM 实测。
	permTitleGap   = 27 // 页面标题 → 首卡（58 → 85）
	permCardPad    = 15 // 卡内边距：图标 x=272 − 卡左 257
	permRowGap     = 14 // 卡内各行之间（139 → 153、184 → 195、223 → 237）
	permLineGap    = 3  // 标签 → 说明（100+18 → 121）
	permSectionGap = 29 // 卡 → 分隔线（306.5 → 335.5）
	permHeadingGap = 22 // 分隔线 → 分组标题（335.5 → 358）
	permIconSize   = 18
	permTableH     = 28  // 表头高（473 → 502，含上下 0.5 描边）
	permColW       = 217 // 表头首列宽（命令 258 → 匹配 475）
	permRowH       = 36  // 规则行高
	permRemoveW    = 56  // 行尾「移除」按钮的列宽（与表头的读屏列对齐）

	permCopyTitle       = "权限"
	permCopyIsolation   = "进程隔离"
	permCopyIsolationIn = "控制命令是否在 App Server 沙箱中运行，以及沙箱命令是否可以使用网络。"
	permCopyLoading     = "正在加载..."
	permCopyRefresh     = "刷新"
	permCopyUnavailable = "沙箱后端不可用："
	permCopySandbox     = "在沙箱中运行命令"
	permCopySandboxIn   = "使用宿主沙箱后端运行命令。"
	permCopyNetwork     = "允许网络访问"
	permCopyNetworkIn   = "允许沙箱命令访问网络。"
	permCopySaveIso     = "保存隔离设置"
	permCopyScope       = "新命令使用这些设置；正在运行的进程保持启动时的隔离边界。"

	permCopyExecTitle = "权限"
	permCopyExecLabel = "Exec command 权限"
	permCopyExecDesc  = "App Server 命令规则作用于 exec 和 execute 调用。"
	permCopyRules     = "规则"
	permCopyRulesDesc = "deny 优先于 allow。exact 匹配完整命令词；prefix 匹配开头的词。"
	permCopyColCmd    = "命令"
	permCopyColMatch  = "匹配"
	permCopyColAction = "决策"
	permCopyMatcher   = "命令匹配式"
	permCopyAddRule   = "添加规则"
	permCopyRemove    = "移除"
	permCopySaveRules = "保存权限"
	permCopySaved     = "已保存。"
	// permCopyRulesAuto 是没有手动刷新通道时的说明：规则读数随连接自动重读。
	permCopyRulesAuto = "命令规则会随连接自动重新读取，这里没有手动刷新"

	permCopyErrEmpty     = "请输入命令匹配式。"
	permCopyErrDuplicate = "命令匹配式不能重复。"

	permCopyLeaveTitle = "命令权限尚未保存"
	permCopyLeaveDesc  = "离开前保存更改，或丢弃更改。"
	permCopyCancel     = "取消"
	permCopyDiscard    = "丢弃"
	permCopySave       = "保存"

	permMatchExact    = "精确"
	permMatchPrefix   = "前缀"
	permDecisionAllow = "允许"
	permDecisionDeny  = "拒绝"
)

// 本屏图标：Heroicons v2 outline（24 网格、1.5 描边、MIT）的内联路径，与
// icons.go 同一套约定。导航里的权限图标是 shield，这里要的是 shield-check 与
// save（保存按钮），形状不同，单独放。
var (
	permIconShieldCheck = mustSVG(`<path d="M9 12.75 11.25 15 15 9.75m-3-7.036A11.959 11.959 0 0 1 3.598 6 11.99 11.99 0 0 0 3 9.749c0 5.592 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.31-.21-2.571-.598-3.751h-.152c-3.196 0-6.1-1.248-8.25-3.285Z"/>`)
	permIconSave        = mustSVG(`<path d="M19.5 14.25v-2.625a3.375 3.375 0 0 0-3.375-3.375h-1.5A1.125 1.125 0 0 1 13.5 7.125v-1.5a3.375 3.375 0 0 0-3.375-3.375H8.25m.75 12 3 3m0 0 3-3m-3 3v-6m-1.5-9H5.625c-.621 0-1.125.504-1.125 1.125v17.25c0 .621.504 1.125 1.125 1.125h12.75c.621 0 1.125-.504 1.125-1.125V11.25a9 9 0 0 0-9-9Z"/>`)
)

// sandboxRule 是一条 Exec command 规则草稿（S-sandbox-27 … 31）。
type sandboxRule struct {
	matcher  string
	match    int // 0 精确 / 1 前缀
	decision int // 0 允许 / 1 拒绝
	// touched 为真表示这一行被改过：空匹配式只在改过之后才报校验错（参考实现
	// 在 blur 时校验；这里按"填过又清空"判，避免新加的空行一出现就报错）。
	touched bool
	err     string
}

// permissionsState 是本屏可丢的界面状态。不是事实。
type permissionsState struct {
	loading bool
	loaded  bool
	// loadError 是读取进程隔离设置失败时的原文；空表示没有失败。
	loadError string
	// backendReason 非空表示沙箱后端不可用；空表示可用。
	backendReason string

	sandbox bool
	network bool
	// savedSandbox / savedNetwork 是"上一次保存成功"的值，用来判有没有改动。
	savedSandbox bool
	savedNetwork bool
	saving       bool
	feedback     string
	saveError    string

	rules []sandboxRule
	// rulesSeeded 表示草稿已经按服务端的规则铺过一次：之后再读到的规则不覆盖用户
	// 正在改的草稿，保存成功后由 permSaveRules 主动清掉它再铺一次。
	rulesSeeded    bool
	rulesReadErr   string
	permissionFeed string
	permissionErr  string

	// leaveOpen 是「命令权限尚未保存」离开确认框。
	leaveOpen bool
}

// permissionsScreen 的初值取运行时截图：进程隔离已加载、后端可用、两个开关都在
// 关位、没有规则。
var permissionsScreen = permissionsState{loaded: true}

// renderPermissions 是设置 › 权限这一屏（设置壳的一个分节）。
//
// 连着 Source 时进程隔离这一卡里的开关照画（参考形态），但整钮置灰，行下与
// tooltip 写原因（noteIsolationUnread）：读不到当前的沙箱与网络设置，所以开关
// 的位置不代表现状，也拨不动；保存本来就没有写入通道。命令规则表照旧：它是本屏
// 的草稿，不含任何"现状"。
func renderPermissions(c *gui.Context, a *App) {
	live := a != nil && a.wired()
	permSyncRules(a, live)
	permValidateRules()
	gui.Column(c).Grow(1).Children(func() {
		gui.Scroll(c).Grow(1).Label(permCopyTitle).Children(func() {
			gui.Column(c).Shrink(0).Gap(permTitleGap).Children(func() {
				// S-permissions-01 分节标题「权限」
				screenTitle(c, permCopyTitle)
				gui.Column(c).Shrink(0).Gap(permSectionGap).Children(func() {
					permIsolationCard(c, live)
					gui.Column(c).Shrink(0).Gap(permHeadingGap).Children(func() {
						screenRule(c)
						permExecSection(c, a, live)
					})
				})
			})
		})
	})
	permLeaveDialog(c)
}

// permIsolationCard 是「进程隔离」卡（S-sandbox-02 … 17）：盾牌 + 标题 + 说明，
// 两个开关，保存按钮与范围说明；加载中 / 加载失败 / 后端不可用各一支。
func permIsolationCard(c *gui.Context, live bool) {
	t := c.Theme()
	screenCard(c, func() {
		gui.Column(c).Padding(permCardPad, permCardPad).Gap(permRowGap).
			Children(func() {
				gui.Column(c).Shrink(0).Gap(permLineGap).Children(func() {
					gui.Row(c).Gap(screenSpace15).AlignItems(gui.Center).Children(func() {
						// S-permissions-02 盾牌图标
						screenIcon(c, permIconShieldCheck, permIconSize, t.TextMuted, "shield-check")
						// S-permissions-03 「进程隔离」
						permGroupTitle(c, permCopyIsolation)
					})
					// S-permissions-04 进程隔离的说明句
					permSubhint(c, permCopyIsolationIn)
				})
				if live {
					permIsolationControlsDisabled(c)
					return
				}
				switch {
				case permissionsScreen.loading:
					permLoadingRow(c)
				case permissionsScreen.loadError != "":
					gui.Row(c).Gap(screenSpace2).AlignItems(gui.Center).Children(func() {
						permAlert(c, permissionsScreen.loadError)
						// 重新拉取要有 Source 的读取方法（DEC-17），本屏不造"重试
						// 成功"的假象：按钮置灰并写原因。
						settingsActionButton(c, settingsActionSpec{
							// S-permissions-06 读失败后的「刷新」
							text: permCopyRefresh, label: permCopyRefresh, height: rowRegular,
							tip: notePermissionWrite, disabled: true,
						})
					})
				case !permissionsScreen.loaded:
					// 还没读到设置、也还没报错：按加载中画，卡片不留空白。
					permLoadingRow(c)
				case permissionsScreen.backendReason != "":
					// S-permissions-07 「沙箱后端不可用：{reason}」
					permAlert(c, permCopyUnavailable+permissionsScreen.backendReason)
				default:
					permIsolationControls(c)
				}
			})
	})
}

// permIsolationControlsDisabled 是连接模式下的同一组控件：两个开关与保存按钮
// 都在场，全部置灰，原因写在开关行下。
func permIsolationControlsDisabled(c *gui.Context) {
	// S-permissions-08 / S-permissions-09 在沙箱中运行命令（开关 + 说明，连接态置灰）
	permSwitchRowDisabled(c, permCopySandbox, permCopySandboxIn, &permissionsScreen.sandbox)
	// S-permissions-11 / S-permissions-12 允许网络访问（开关 + 说明，连接态置灰）
	permSwitchRowDisabled(c, permCopyNetwork, permCopyNetworkIn, &permissionsScreen.network)
	settingsActionButton(c, settingsActionSpec{
		variant: settingsActionPrimary, icon: permIconSave, mark: "save",
		// S-permissions-14 保存隔离设置（连接态置灰）
		text: permCopySaveIso, label: permCopySaveIso, height: rowRegular,
		tip: noteIsolationUnread, disabled: true,
	})
	permSubhint(c, noteIsolationUnread)
	// S-permissions-17 隔离设置的作用范围说明
	permSubhint(c, permCopyScope)
}

// permSwitchRowDisabled 是同款开关行的禁用态：开关照画、置灰，原因进行下说明。
func permSwitchRowDisabled(c *gui.Context, label, hint string, on *bool) {
	gui.Row(c).AlignItems(gui.Center).Gap(screenSpace2).Children(func() {
		gui.Column(c).Grow(1).MinWidth(0).Gap(permLineGap).Children(func() {
			permLabel(c, label)
			permSubhint(c, hint)
			permSubhint(c, noteIsolationUnread)
		})
		gui.Switch(c, on).Size(screenSwitchW, screenSwitchH).
			Radius(screenSwitchH / 2).Label(label).
			Disabled(true).Tooltip(noteIsolationUnread).Opacity(noteDisabledAlpha)
	})
}

// permIsolationControls 是已加载时的三行：两个开关 + 保存按钮 + 范围说明。
func permIsolationControls(c *gui.Context) {
	// S-permissions-10 在沙箱中运行命令（开关）
	permSwitchRow(c, permCopySandbox, permCopySandboxIn, &permissionsScreen.sandbox)
	// S-permissions-13 允许网络访问（开关）
	permSwitchRow(c, permCopyNetwork, permCopyNetworkIn, &permissionsScreen.network)
	if settingsActionButton(c, settingsActionSpec{
		variant: settingsActionPrimary, icon: permIconSave, mark: "save",
		// S-permissions-15 保存隔离设置与保存回执
		text: permCopySaveIso, label: permCopySaveIso, height: rowRegular,
		tip: notePermissionWrite, disabled: true,
	}) {
		// 写入走 App Server 的进程隔离设置：Intents 里还没有对应方法（DEC-17），
		// 入口置灰并写原因，这里不造"已保存"的假象。
	}
	permSubhint(c, notePermissionWrite)
	if permissionsScreen.feedback != "" {
		permNote(c, permissionsScreen.feedback, c.Theme().Success)
	}
	if permissionsScreen.saveError != "" {
		permAlert(c, permissionsScreen.saveError)
	}
	// S-permissions-17 隔离设置的作用范围说明
	permSubhint(c, permCopyScope)
}

// permIsolationDirty 判有没有改动：与上一次保存成功的值比。
func permIsolationDirty() bool {
	return permissionsScreen.sandbox != permissionsScreen.savedSandbox ||
		permissionsScreen.network != permissionsScreen.savedNetwork
}

// permSwitchRow 是一行开关：标签 + 说明在左，28×16 开关在右（S-sandbox-08 … 13）。
func permSwitchRow(c *gui.Context, label, hint string, on *bool) {
	gui.Row(c).AlignItems(gui.Center).Gap(screenSpace2).Children(func() {
		gui.Column(c).Grow(1).MinWidth(0).Gap(permLineGap).Children(func() {
			permLabel(c, label)
			permSubhint(c, hint)
		})
		gui.Switch(c, on).Size(screenSwitchW, screenSwitchH).
			Radius(screenSwitchH / 2).Label(label).Disabled(permissionsScreen.saving)
	})
}

// permExecSection 是规则区（S-sandbox-18 … 35）：分组标题、标签、说明、
// 规则表与两个按钮。
func permExecSection(c *gui.Context, a *App, live bool) {
	gui.Column(c).Shrink(0).Children(func() {
		gui.Column(c).Shrink(0).Gap(permLineGap).Children(func() {
			// S-permissions-18 「权限」分组标题
			permGroupTitle(c, permCopyExecTitle)
			// S-permissions-19 「Exec command 权限」
			permLabel(c, permCopyExecLabel)
			// S-permissions-20 命令规则的说明句
			permSubhint(c, permCopyExecDesc)
			gui.Column(c).Shrink(0).Gap(screenSpace05).Children(func() {
				// S-permissions-21 「规则」
				permRulesTitle(c, permCopyRules)
				// S-permissions-22 deny 优先于 allow。exact / prefix 的匹配口径
				permSubhint(c, permCopyRulesDesc)
			})
		})
		gui.Column(c).Shrink(0).Gap(screenSpace35).Children(func() {
			gui.Column(c).Shrink(0).Gap(screenSpace2).Children(func() {
				permRuleTable(c)
				if settingsActionButton(c, settingsActionSpec{
					// S-permissions-26 添加规则
					text: permCopyAddRule, label: permCopyAddRule,
					height: rowRegular, disabled: permissionsScreen.saving,
				}) {
					permissionsScreen.rules = append(permissionsScreen.rules, sandboxRule{})
				}
			})
			// 读失败：写原因；刷新入口保留但置灰（规则读数由连接层按需重读）。
			if live && permissionsScreen.rulesReadErr != "" {
				gui.Row(c).Gap(screenSpace2).AlignItems(gui.Center).Children(func() {
					permAlert(c, permissionsScreen.rulesReadErr)
					settingsActionButton(c, settingsActionSpec{
						text: permCopyRefresh, label: permCopyRefresh, height: rowRegular,
						tip: permCopyRulesAuto, disabled: true,
					})
				})
			}
			permRulesActions(c, a, live)
		})
	})
}

// permRulesActions 是保存权限与它的反馈（S-sandbox-33 … 35）。
//
// 连着 Source 时保存权限是**真写入**：整份规则集合交给 Intents.WriteCommandRules，
// 成功写「已保存。」并按回执重铺草稿，失败把原因写进错误行（冲突不重试）。
// 没有连接时（-demo）入口置灰并写原因，不造「已保存」的假象。
func permRulesActions(c *gui.Context, a *App, live bool) {
	disabled := !live || permRulesBlocked()
	tip := ""
	switch {
	case !live:
		tip = notePermissionWrite
	case permRulesBlocked():
		tip = permCopyErrEmpty
	}
	if settingsActionButton(c, settingsActionSpec{
		// S-permissions-27 保存权限
		variant: settingsActionPrimary, icon: permIconSave, mark: "save",
		text: permCopySaveRules, label: permCopySaveRules,
		height: rowRegular, tip: tip, disabled: disabled,
	}) {
		permSaveRules(a)
	}
	// S-permissions-28 保存权限的反馈
	if permissionsScreen.permissionFeed != "" {
		permNote(c, permissionsScreen.permissionFeed, c.Theme().Success)
	}
	// S-permissions-29 保存权限的错误
	if permissionsScreen.permissionErr != "" {
		permAlert(c, permissionsScreen.permissionErr)
	}
}

// permSyncRules 按事实铺一次规则草稿：连着 Source、读到规则、且还没铺过时铺，
// 之后不覆盖用户正在改的草稿。读失败把原因留在 permRulesBlocked 之外的一行里。
func permSyncRules(a *App, live bool) {
	if !live {
		return
	}
	st, ok := a.liveSettings()
	if !ok {
		return
	}
	permissionsScreen.rulesReadErr = st.RulesError
	if !st.RulesKnown || permissionsScreen.rulesSeeded {
		return
	}
	permissionsScreen.loaded = true
	rules := make([]sandboxRule, 0, len(st.Rules))
	for _, rule := range st.Rules {
		match := 0
		if rule.Match == "prefix" {
			match = 1
		}
		decision := 0
		if rule.Decision == "deny" {
			decision = 1
		}
		rules = append(rules, sandboxRule{matcher: rule.Command, match: match, decision: decision, touched: true})
	}
	permissionsScreen.rules = rules
	permissionsScreen.rulesSeeded = true
}

// permSaveRules 把草稿整份写给服务端；成功写回执，失败写原因。
//
// 空匹配式的行在保存前被丢掉（校验已经拦住了"填过又清空"的行，这里兜住从没填过
// 的空行）：服务端不接受空命令，发过去只会换一个语焉不详的错误。
func permSaveRules(a *App) {
	if a == nil || a.intents == nil {
		return
	}
	payload := make([]RuleSnapshot, 0, len(permissionsScreen.rules))
	for _, rule := range permissionsScreen.rules {
		matcher := strings.TrimSpace(rule.matcher)
		if matcher == "" {
			continue
		}
		match := "exact"
		if rule.match == 1 {
			match = "prefix"
		}
		decision := "allow"
		if rule.decision == 1 {
			decision = "deny"
		}
		payload = append(payload, RuleSnapshot{Command: matcher, Decision: decision, Match: match})
	}
	permissionsScreen.permissionFeed = ""
	permissionsScreen.permissionErr = ""
	permissionsScreen.saving = true
	a.dispatchQuiet("保存权限", func(ctx context.Context) error {
		return a.intents.WriteCommandRules(ctx, payload)
	}, func() {
		permissionsScreen.saving = false
		permissionsScreen.permissionFeed = permCopySaved
		permissionsScreen.permissionErr = ""
		// 回执里的规则是服务端接受的版本：清掉播种标记，下一帧按事实重铺。
		permissionsScreen.rulesSeeded = false
	}, func(reason string) {
		permissionsScreen.saving = false
		// 错误行写服务端原文（S-permissions-29 的「错误原文」），不加前缀：
		// 前缀会把「逐字等于接口返回」这条判据改成两句话。
		permissionsScreen.permissionErr = reason
	})
}

// permRulesBlocked 判保存权限是否不可用：有行内错误或正在保存（R-PRM-197）。
func permRulesBlocked() bool {
	if permissionsScreen.saving {
		return true
	}
	for _, rule := range permissionsScreen.rules {
		if rule.err != "" {
			return true
		}
	}
	return false
}

// permRuleTable 是规则表：三列并排时表头常显（S-sandbox-23 … 26），行是有规则
// 时才出现（S-sandbox-24）。窄窗里规则行按行堆叠（U2-forms F23），每一格自带小
// 标签，表头就不再画——表头只描述并排的三列，堆叠后它对不上任何一行。
func permRuleTable(c *gui.Context) {
	gui.Column(c).Shrink(0).Children(func() {
		if !permNarrow(c) {
			permTableHeader(c)
		}
		for i := range permissionsScreen.rules {
			permRuleRow(c, i)
		}
	})
}

// permTableHeader 是表头：命令 / 匹配 / 决策三列可见，第四列「规则」只给读屏。
// S-permissions-23 表头（命令 / 匹配 / 决策 / 规则）
func permTableHeader(c *gui.Context) {
	t := c.Theme()
	box := gui.Row(c).Shrink(0).MinHeight(permTableH).Padding(0, screenSpace2).
		AlignItems(gui.Center).Radius(screenRadiusCtl).Background(t.Background).
		Role(gui.RoleRow).Label("命令规则表头")
	box.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, screenRadiusCtl, screenStroke)
	})
	box.Children(func() {
		permHeaderCell(c, permCopyColCmd, permColW, false, false)
		permHeaderCell(c, permCopyColMatch, 0, true, false)
		permHeaderCell(c, permCopyColAction, 0, true, false)
		permHeaderCell(c, permCopyRules, permRemoveW, false, true)
	})
}

// permHeaderCell 是一个表头格：width 为 0 时均分剩余宽度；hidden 为真时只给
// 读屏（参考实现的第四列「规则」，屏上不出现字）。
func permHeaderCell(c *gui.Context, text string, width float32, grow, hidden bool) {
	cell := gui.Box(c)
	switch {
	case width > 0:
		cell = cell.Width(width).Shrink(0)
	case grow:
		cell = cell.Grow(1)
	}
	if hidden {
		cell.Label(text).AccessHidden(true)
		return
	}
	cell.Children(func() {
		screenTick(c, text)
	})
}

// permRuleRow 是一条规则（S-sandbox-24 … 31）：命令输入 + 匹配 / 决策下拉 +
// 移除；窄窗按 U2-forms F23 堆叠成两行。
// S-permissions-24 规则行（命令输入 + 匹配 / 决策下拉 + 移除）
func permRuleRow(c *gui.Context, index int) {
	rule := &permissionsScreen.rules[index]
	screenRule(c)
	gui.Column(c).Shrink(0).Gap(screenSpace15).Padding(screenSpace1, 0).
		Children(func() {
			if permNarrow(c) {
				gui.Row(c).AlignItems(gui.Center).Children(func() {
					permMatcherInput(c, rule)
				})
				gui.Row(c).Gap(screenSpace2).AlignItems(gui.Center).Children(func() {
					permRuleSelects(c, rule)
					permRemoveButton(c, index)
				})
			} else {
				gui.Row(c).MinHeight(permRowH).AlignItems(gui.Center).Children(func() {
					gui.Box(c).Width(permColW).Shrink(0).Children(func() {
						permMatcherInput(c, rule)
					})
					permRuleSelects(c, rule)
					permRemoveButton(c, index)
				})
			}
			// S-permissions-25 规则行的校验错误
			if rule.err != "" {
				permAlert(c, rule.err)
			}
		})
}

// permMatcherInput 是命令匹配式的输入框（等宽）。
// S-permissions-24 命令匹配式输入
func permMatcherInput(c *gui.Context, rule *sandboxRule) {
	screenInput(c, &rule.matcher, permCopyMatcher, permCopyMatcher).
		Font(codeFamily()).Grow(1)
}

// permRuleSelects 是两个下拉：匹配（精确 / 前缀）与决策（允许 / 拒绝）。
// S-permissions-24 匹配与决策两个下拉
func permRuleSelects(c *gui.Context, rule *sandboxRule) {
	gui.Row(c).Grow(1).Gap(screenSpace2).Children(func() {
		screenIndexSelect(c, &rule.match, permMatchOptions, permCopyColMatch)
		screenIndexSelect(c, &rule.decision, permDecisionOptions, permCopyColAction)
	})
}

// permRemoveButton 是行尾的「移除」（S-sandbox-30）。
// S-permissions-24 行尾的「移除」
func permRemoveButton(c *gui.Context, index int) {
	if settingsActionButton(c, settingsActionSpec{
		text: permCopyRemove, label: permCopyRemove + "：第 " + permOrdinal(index) + " 条",
		height: rowRegular, disabled: permissionsScreen.saving,
	}) {
		permissionsScreen.rules = append(permissionsScreen.rules[:index],
			permissionsScreen.rules[index+1:]...)
	}
}

// permMatchOptions / permDecisionOptions 是两个下拉的取值，顺序与参考实现一致。
var (
	permMatchOptions    = []string{permMatchExact, permMatchPrefix}
	permDecisionOptions = []string{permDecisionAllow, permDecisionDeny}
)

// permOrdinal 是规则行的序号文字（第 1 条…），只用于读屏名。
func permOrdinal(index int) string {
	digits := "0123456789"
	n := index + 1
	if n < 10 {
		return string(digits[n])
	}
	return permOrdinal(n/10-1) + string(digits[n%10])
}

// permValidateRules 是行内校验（S-sandbox-31、R-PRM-190/193）：填过又清空报
// 「请输入命令匹配式」，与前面的行重复报「命令匹配式不能重复」。
func permValidateRules() {
	seen := map[string]bool{}
	for i := range permissionsScreen.rules {
		rule := &permissionsScreen.rules[i]
		matcher := strings.TrimSpace(rule.matcher)
		if matcher != "" {
			rule.touched = true
		}
		switch {
		case rule.touched && matcher == "":
			rule.err = permCopyErrEmpty
		case matcher != "" && seen[matcher]:
			rule.err = permCopyErrDuplicate
		default:
			rule.err = ""
		}
		if matcher != "" {
			seen[matcher] = true
		}
	}
}

// permNarrow 判内容列是否窄到放不下一整行规则（U2-forms F23 的退化阈值）。
// 可用宽取 settingsUsableWidth（窄窗没有 224 的常驻导航，不能按窗口直接减），
// 再减去设置壳的窗框留白与内容列两侧内边距。
func permNarrow(c *gui.Context) bool {
	content := settingsUsableWidth(c) - 2*settingsShellInset - 2*space8
	return content < SettingsRuleRowMinWidth
}

// permLeaveDialog 是「命令权限尚未保存」确认框（S-sandbox-36 … 40）。
// S-permissions-30 命令权限尚未保存 · 取消 / 丢弃 / 保存
func permLeaveDialog(c *gui.Context) {
	if !permissionsScreen.leaveOpen {
		return
	}
	open := true
	m := modalShell(gui.Modal(c, &open, func() {
		screenDialogPanel(c, permCopyLeaveTitle, func() {
			screenHint(c, permCopyLeaveDesc)
			gui.Row(c).Justify(gui.End).Gap(screenSpace2).Children(func() {
				if settingsActionButton(c, settingsActionSpec{
					text: permCopyCancel, label: permCopyCancel, height: rowRegular,
				}) {
					permissionsScreen.leaveOpen = false
					open = false
				}
				if settingsActionButton(c, settingsActionSpec{
					text: permCopyDiscard, label: permCopyDiscard, height: rowRegular,
				}) {
					permissionsScreen.leaveOpen = false
					open = false
				}
				if settingsActionButton(c, settingsActionSpec{
					variant: settingsActionPrimary, text: permCopySave,
					label: permCopySave, height: rowRegular,
				}) {
					permissionsScreen.leaveOpen = false
					open = false
				}
			})
		})
	}))

	if !open || m.Dismissed() {
		permissionsScreen.leaveOpen = false
	}
}

// ---- 本屏的小构件 ----

// permGroupTitle 是分组标题：16、半粗、三级标题（R-PRM-020 的播报判据）。
func permGroupTitle(c *gui.Context, text string) {
	t := c.Theme()
	typeH3.style(t, gui.Text(c, text)).TextColor(t.Text).
		Role(gui.RoleHeading).Level(3).SingleLine()
}

// permRulesTitle 是小标题「规则」：13、半粗、四级标题（R-PRM-164）。
func permRulesTitle(c *gui.Context, text string) {
	t := c.Theme()
	typeH5.style(t, gui.Text(c, text)).TextColor(t.Text).
		Role(gui.RoleHeading).Level(4).SingleLine()
}

// permLabel 是一行标签：13、中粗、前景（R-PRM-058 的 13 字号）。
func permLabel(c *gui.Context, text string) {
	t := c.Theme()
	permLabelStyle.style(t, gui.Text(c, text)).TextColor(t.Text).Label(text)
}

// permSubhint 是说明：12.25、次级前景、可复制（R-PRM-028/068/132）。
func permSubhint(c *gui.Context, text string) {
	t := c.Theme()
	typeSecondary.style(t, gui.Text(c, text)).TextColor(t.TextMuted).Selectable()
}

// permAlert 是一行警示 / 错误：12.25、警示色、role=alert（R-PRM-193/198）。
func permAlert(c *gui.Context, text string) {
	t := c.Theme()
	typeSecondary.style(t, gui.Text(c, text)).TextColor(t.Danger).
		Selectable().Label(text)
}

// permNote 是一行成功反馈：12.25、成功色（R-PRM 的"已保存"两支）。
func permNote(c *gui.Context, text string, color gui.Color) {
	t := c.Theme()
	typeSecondary.style(t, gui.Text(c, text)).TextColor(color).
		Selectable().Label(text)
}

// permLoadingRow 是「正在加载...」（S-sandbox-04）：转圈 + 文案。
// S-permissions-05 正在加载...
func permLoadingRow(c *gui.Context) {
	gui.Row(c).Shrink(0).Gap(screenSpace2).AlignItems(gui.Center).Children(func() {
		gui.Spinner(c).Size(screenIconInline, screenIconInline).Label(permCopyLoading)
		screenHint(c, permCopyLoading)
	})
}

// permLabelStyle 是操作行标签：13、中粗。
var permLabelStyle = fontStep{size: 13.3, line: 20, weight: 500}
