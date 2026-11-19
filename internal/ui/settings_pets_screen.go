package ui

import (
	"fmt"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是「设置 › 宠物」的重建屏：元素按
// docs/domains/requirements/inventory-settings.md §S-pets（源码穷举 S-pets-01 …
// 36，36 个）逐行对应，可见态与位置取 inventory-settings-rt.md §12
// （S-pets-01 … 26）；行为对应 requirements-settings.md 的 R-PET-001 … 208。
//
// 数值取 docs/domains/requirements/design-baseline.md（页面标题 17.5、分组标题 14、
// 行标签 14、说明 10.5、开关 28×16、按钮 28/32、卡片圆角 8.75）与运行时截图的
// 实测（标题下 27、分组间 30、分组标题下 14、空态上 13；卡内 15；列表行 85、
// 头像 48、行内边距 15/16；空态 48 高）。
//
// 只保留可丢的界面状态（petsScreen）：显示开关、大小、描述符与自定义列表、
// 当前选中、删除对话框。事实（petStore 的快照与描述符、选择的写入、安装与删除）
// 归宠物领域；本屏不写事实，接线时把选择 / 刷新 / 删除换成 Intents。
//
// # 与参考实现的差异
//
//   - 头像：Pet 素材（`pet-<id>.png`）不在仓库，本屏用等尺寸的圆角块 + 名字首字母
//     占位（与账户屏 G-01 头像同一策略），不冒充素材。
//   - 预览区（S-pets-16）：参考实现是 PetSettingsPreview 的动画预览；本屏没有
//     渲染 Pet 的组件，预览位就是头像块，`previewUnavailable` 时改画一行警示
//     （S-pets-22）。
//   - 禁用态：参考实现用框架按钮的 disabled（整棵子树 0.5）；本屏的按钮走
//     settingsActionButton，同一口径（见 §共用按钮）。
//
// # 未实现分支
//
//   - Pet 显示组件本身：本屏只画「显示 Pet」的开关与不可用提示，没有浮窗渲染器，
//     开关恒为禁用（主窗口 / 组件不可用 / 忙，R-PET-025 … 034）。
//   - 写入：选择（R-PET-115/119/178）、刷新（R-PET-154）、安装（R-PET-146/152）、
//     创建（R-PET-138）、打开文件夹（R-PET-162）、打开保留素材（R-PET-066）、
//     删除（R-PET-202/208）都要宠物领域或宿主能力；本屏只改可丢的界面状态。
//   - 失败反馈：operationError（S-pets-07）、恢复说明与「打开保留素材」
//     （S-pets-08 … 10）、行级 {code}:{message}（S-pets-20/30）、预览失败
//     （S-pets-16/22）的渲染分支都在，但造出这些态的数据来自 Pet 领域，本屏
//     默认不进入。
//   - 语言包缺失的回落（R-PET-005/014/022 …）：文案直接取参考产品中文，无 i18n 层。

const (
	// 宠物屏几何：design-baseline 只给常量，元素级包围盒取运行时 png 的边框行
	// 与 inventory-settings-rt.md §12 的 DOM 实测。
	petsTitleGap   = 27  // 页面标题 → 首卡（58 → 85）
	petsSectionGap = 30  // 卡 → 分组标题（196.5 → 226）、列表 → 分组标题（430.5 → 460）
	petsGroupGap   = 14  // 分组标题 → 内容（226+21 → 260.5；481 → 495）
	petsBodyGap    = 13  // 按钮行 → 空态（527 → 540）
	petsCardPad    = 15  // 卡内边距：标签 x=272 − 卡左 257
	petsNoteGap    = 14  // 行说明 → 提示条（139 → 149 减行盒 14）
	petsNotePadY   = 7.5 // 提示条上下：32.5 = 17.5（行盒）+ 2×7.5
	petsLineGap    = 3   // 标签 → 说明（100+18 → 121）
	petsNotePadX   = 10
	petsRowH       = 85  // 列表行高（260.5 → 345.5）
	petsAvatarSize = 48  // 头像块（截图 565–660 设备像素 / 2）
	petsRowGap     = 16  // 头像 → 名字（320 → 337 减行内边距）
	petsSizeW      = 112 // Pet 大小滑杆 w-32 = 8rem = 112px
	petsEmptyH     = 48  // 空态虚线框（540 → 588）
	petsDisabled   = 0.5

	petsCopyTitle    = "宠物"
	petsCopyShown    = "显示 Pet"
	petsCopyFloating = "Pet 会保持在其他窗口上方。"
	petsCopySize     = "Pet 大小"
	petsCopyChoose   = "选择 Pet"
	petsCopyCustom   = "自定义 Pets"
	petsCopyCreate   = "创建"
	petsCopyInstall  = "从文件夹安装"
	petsCopyRefresh  = "刷新"
	petsCopyOpenDir  = "打开 Pets 文件夹"
	petsCopyEmpty    = "尚未安装自定义 Pet。"
	// petsCopyEmptyNote 是连接模式下自定义列表的说明：读不到清单，不写"尚未安装"。
	petsCopyEmptyNote = "服务端没有提供自定义 Pet 清单（本版不做）"
	petsCopyLoading   = "正在加载..."
	petsCopyRetry     = "重试"
	petsCopySelect    = "选择"
	petsCopySelected  = "已选择"
	petsCopyDelete    = "删除"
	petsCopyDeleting  = "正在删除…"
	petsCopyCancel    = "取消"
	petsCopyOpenKeep  = "打开保留素材"
	petsCopyPreviewNA = "无法渲染这个 Pet 的预览。"

	petsCopyNotePlatform = "当前平台不支持显示 Pet。你仍可选择、预览和管理 Pet 文件。"
	petsCopyNoteRestart  = "Pet 显示失败，本次运行已停用；重启应用即可恢复。"
	petsCopyNoteAddon    = "Pet 显示组件不可用，应用安装可能不完整或已损坏；重新安装或更新应用可恢复。"
	petsCopyRecovery     = "打开素材位置并备份。修复目录权限后，刷新并从备份重新安装。不要覆盖相同 ID 的 Pet。"
	petsCopyDeleteKept   = "已保存的选择将切换回内置 Pet。"
	petsCopyDeleteOnly   = "这个自定义 Pet 将从托管的 Pets 文件夹中移除。"
)

// petsNote 是提示条的三选一（inventory-settings.md §S-pets-05）：按不可用原因取。
type petsNote int

const (
	petsNoteNone petsNote = iota
	petsNotePlatform
	petsNoteRestart
	petsNoteAddon
)

// petDescriptor 是一个可选的 Pet（参考实现的 PetDescriptor）：名字与描述来自
// 描述符列表，problem 是该 Pet 不可用时的 {code}: {message}。
type petDescriptor struct {
	id          string
	name        string
	description string
	available   bool
	problem     string
}

// petsState 是本屏可丢的界面状态。不是事实。
type petsState struct {
	shown bool
	size  float64

	// mainWindow 为真时隐藏「Pet 大小」（参考实现只在浮窗里给）。
	mainWindow bool
	busy       bool
	note       petsNote

	// descriptors 为 nil 表示描述符还没加载（S-pets-13/17）。
	descriptors []petDescriptor
	selected    string
	custom      []petDescriptor

	operationError     string
	recoveryPetID      string
	selectionMissing   bool
	selectionError     string
	previewUnavailable bool

	deleteFor string
	deleting  bool
}

// petsScreen 的初值取运行时截图：主窗口、组件不可用（addon 缺）、两个内置 Pet
// 可选、dim-fox 已选中、没有自定义 Pet。
var petsScreen = petsState{
	mainWindow: true,
	size:       1,
	note:       petsNoteAddon,
	selected:   "dim-fox",
	descriptors: []petDescriptor{
		{id: "dim", name: "Dim", available: true},
		{id: "dim-fox", name: "dim-fox", available: true},
	},
}

// renderPets 是设置 › 宠物这一屏（设置壳的一个分节）。
//
// 连着 Source 时没有任何 Pet 事实（清单、当前选择、显示开关都没有通道）：参考图里
// 的控件形态照留——开关、列表卡、四个动作按钮——但全部置灰，原因写在 tooltip 与
// 卡片里（notePetSettings）。列表区没有行可画，所以是一个写着原因的虚线空态，
// 不拿样例的两个内置 Pet 与「dim-fox 已选中」当现状。样例只在 -demo 出现。
func renderPets(c *gui.Context, a *App) {
	live := a != nil && a.wired()
	gui.Column(c).Grow(1).Children(func() {
		gui.Scroll(c).Grow(1).Label(petsCopyTitle).Children(func() {
			gui.Column(c).Shrink(0).Gap(petsSectionGap).Children(func() {
				gui.Column(c).Shrink(0).Gap(petsTitleGap).Children(func() {
					screenTitle(c, petsCopyTitle)
					petsStatusCard(c, live)
				})
				petsChooseSection(c, live)
				petsCustomSection(c, live)
			})
		})
	})
	petsDeleteDialog(c)
}

// petsStatusCard 是「显示 Pet」卡（S-pets-02 … 11）：标签 + 说明 + 开关，
// 下面是不可用提示条；浮窗里再加「Pet 大小」，出错时加错误行与恢复入口，
// 描述符未加载时给「重试」。
func petsStatusCard(c *gui.Context, live bool) {
	screenCard(c, func() {
		gui.Column(c).Padding(petsCardPad, petsCardPad).Gap(petsNoteGap).
			Children(func() {
				gui.Row(c).AlignItems(gui.Center).Gap(screenSpace2).Children(func() {
					gui.Column(c).Grow(1).MinWidth(0).Gap(petsLineGap).Children(func() {
						petsLabel(c, petsCopyShown)
						petsSubhint(c, petsCopyFloating)
						if live {
							petsSubhint(c, notePetSettings)
						}
					})
					sw := gui.Switch(c, &petsScreen.shown).
						Size(screenSwitchW, screenSwitchH).
						Radius(screenSwitchH / 2).
						Label(petsCopyShown).
						Disabled(petsShownDisabled())
					if live {
						sw = sw.Disabled(true).Tooltip(notePetSettings).Opacity(noteDisabledAlpha)
					}
				})
				if live {
					// 连接模式：没有 Pet 快照，样例的"组件不可用"提示条也不画。
					return
				}
				if note := petsNoteCopy(petsScreen.note); note != "" {
					petsNoteBar(c, note)
				}
				if !petsScreen.mainWindow {
					petsSizeRow(c)
				}
				if petsScreen.operationError != "" {
					petsAlert(c, petsScreen.operationError)
				}
				if petsScreen.recoveryPetID != "" {
					gui.Column(c).Shrink(0).Gap(screenSpace2).Children(func() {
						petsAlert(c, petsCopyRecovery)
						settingsActionButton(c, settingsActionSpec{
							icon: screenIconFolderOpen, mark: "folder-open",
							text: petsCopyOpenKeep, label: petsCopyOpenKeep,
						})
					})
				}
				if petsScreen.descriptors == nil && !petsScreen.busy {
					settingsActionButton(c, settingsActionSpec{
						text: petsCopyRetry, label: petsCopyRetry,
					})
				}
			})
	})
}

// petsShownDisabled 是「显示 Pet」的禁用条件：主窗口、组件不可用、忙、
// 快照未加载（参考实现的 `:disabled`）。
func petsShownDisabled() bool {
	return petsScreen.mainWindow || petsScreen.note != petsNoteNone ||
		petsScreen.busy || petsScreen.descriptors == nil
}

// petsNoteCopy 是提示条的文案（S-pets-05 的三选一）。
func petsNoteCopy(note petsNote) string {
	switch note {
	case petsNotePlatform:
		return petsCopyNotePlatform
	case petsNoteRestart:
		return petsCopyNoteRestart
	case petsNoteAddon:
		return petsCopyNoteAddon
	}
	return ""
}

// petsSizeRow 是「Pet 大小」：标签 + 滑杆（0.5–2，步长 0.1，宽 112）。
func petsSizeRow(c *gui.Context) {
	gui.Column(c).Shrink(0).Gap(screenSpace15).Children(func() {
		petsLabel(c, petsCopySize)
		gui.Slider(c, &petsScreen.size, petsSizeMin, petsSizeMax).Step(petsSizeStep).
			Width(petsSizeW).Label(petsCopySize).Disabled(petsScreen.busy)
	})
}

const (
	petsSizeMin  = 0.5
	petsSizeMax  = 2
	petsSizeStep = 0.1
)

// petsChooseSection 是「选择 Pet」（S-pets-12 … 22）：分组标题 + 列表卡。
// 连着 Source 时没有清单可列：列表区是写着原因的虚线空态（形态在，值不编）。
func petsChooseSection(c *gui.Context, live bool) {
	gui.Column(c).Shrink(0).Gap(petsGroupGap).Children(func() {
		petsGroupTitle(c, petsCopyChoose)
		if live {
			petsEmptyState(c, notePetSettings)
			return
		}
		if petsScreen.selectionMissing {
			petsMissingSelection(c)
		}
		switch {
		case petsScreen.descriptors == nil:
			petsLoadingRow(c)
		default:
			petsListCard(c)
		}
	})
}

// petsMissingSelection 是「已保存的 Pet 不可用」提示（S-pets-13 … 16）。
func petsMissingSelection(c *gui.Context) {
	petsAlert(c, fmt.Sprintf("已保存的 Pet 不可用：%s", petsScreen.selected))
	if petsScreen.selectionError != "" {
		petsAlert(c, petsScreen.selectionError)
	}
	if name := petsFallbackName(); name != "" {
		petsSubhint(c, fmt.Sprintf("在已保存的 Pet 恢复可用前，将显示 %s。", name))
	}
}

// petsFallbackName 是选择缺失时的回落 Pet：第一个可用的描述符。
func petsFallbackName() string {
	for _, pet := range petsScreen.descriptors {
		if pet.available {
			return pet.name
		}
	}
	return ""
}

// petsLoadingRow 是「正在加载...」（S-pets-17）：转圈 + 文案。
func petsLoadingRow(c *gui.Context) {
	gui.Row(c).Shrink(0).Gap(screenSpace2).AlignItems(gui.Center).Children(func() {
		gui.Spinner(c).Size(screenIconInline, screenIconInline).Label(petsCopyLoading)
		screenHint(c, petsCopyLoading)
	})
}

// petsListCard 是内置 Pet 的列表卡：一行一个 Pet，行间 1px 细线。
func petsListCard(c *gui.Context) {
	screenCard(c, func() {
		gui.Column(c).Shrink(0).Children(func() {
			for i, pet := range petsScreen.descriptors {
				if i > 0 {
					screenRule(c)
				}
				petsListRow(c, pet)
			}
		})
	})
}

// petsListRow 是一个 Pet：头像 + 名字（+ 描述 / 错误）+ 选择按钮。
// 选中行的底色是主色的极浅档（截图 #f6f9f6），不是描边。
func petsListRow(c *gui.Context, pet petDescriptor) {
	t := c.Theme()
	selected := pet.id == petsScreen.selected
	row := gui.Row(c).Shrink(0).MinHeight(petsRowH).Padding(0, petsCardPad).
		AlignItems(gui.Center).Gap(petsRowGap).Label(pet.name)
	if selected {
		row = row.Background(t.Accent.Alpha(petsSelectedAlpha))
	}
	row.Children(func() {
		petsAvatar(c, pet)
		gui.Column(c).Grow(1).MinWidth(0).Children(func() {
			petsLabel(c, pet.name)
			if pet.description != "" {
				petsSubhint(c, pet.description)
			}
			if pet.problem != "" {
				petsAlert(c, pet.problem)
			}
			if selected && petsScreen.previewUnavailable {
				petsAlert(c, petsCopyPreviewNA)
			}
		})
		if petsRowButton(c, pet, selected) {
			petsScreen.selected = pet.id
		}
	})
}

// petsSelectedAlpha 是选中行的底色透明度（截图 #f6f9f6 对白底 ≈ 主色 5%）。
const petsSelectedAlpha = 0.05

// petsAvatar 是 Pet 的位置：素材不在仓库，用圆角块 + 名字首字母占位。
func petsAvatar(c *gui.Context, pet petDescriptor) {
	t := c.Theme()
	gui.Box(c).Shrink(0).Size(petsAvatarSize, petsAvatarSize).
		Radius(screenRadiusCard).Background(t.SurfaceHover).Center().Children(func() {
		screenStrongStyle.style(t, gui.Text(c, petsInitial(pet.name))).
			TextColor(t.TextMuted).Label("Pet 预览：" + pet.name).SingleLine()
	})
}

// petsInitial 取名字的第一个字（大写），作为占位标记。
func petsInitial(name string) string {
	for _, r := range name {
		if r >= 'a' && r <= 'z' {
			return string(r - 32)
		}
		return string(r)
	}
	return "?"
}

// petsRowButton 是一行的动作：可选且未选中时是主按钮「选择」，已选中与不可用
// 都是禁用的描边按钮「已选择」/「选择」。
func petsRowButton(c *gui.Context, pet petDescriptor, selected bool) bool {
	switch {
	case selected:
		settingsActionButton(c, settingsActionSpec{
			text: petsCopySelected, label: petsCopySelected + "：" + pet.name,
			height: rowRegular, disabled: true,
		})
		return false
	case !pet.available:
		settingsActionButton(c, settingsActionSpec{
			text: petsCopySelect, label: petsCopySelect + "：" + pet.name,
			height: rowRegular, disabled: true,
		})
		return false
	}
	return settingsActionButton(c, settingsActionSpec{
		variant: settingsActionPrimary,
		text:    petsCopySelect, label: petsCopySelect + "：" + pet.name,
		height: rowRegular, tip: petsCopySelect,
	})
}

// petsCustomSection 是「自定义 Pets」（S-pets-23 … 32）：分组标题 + 按钮行 +
// 空态 / 列表。
func petsCustomSection(c *gui.Context, live bool) {
	gui.Column(c).Shrink(0).Gap(petsBodyGap).Children(func() {
		gui.Column(c).Shrink(0).Gap(petsGroupGap).Children(func() {
			petsGroupTitle(c, petsCopyCustom)
			gui.Row(c).Wrap().Gap(screenSpace2).AlignItems(gui.Center).Children(func() {
				loaded := petsScreen.descriptors != nil
				if live {
					// 四个动作都要 Pet 领域或宿主能力：形态照留，逐个置灰并写原因。
					for _, spec := range []settingsActionSpec{
						{variant: settingsActionPrimary, icon: iconPlus, mark: "plus", text: petsCopyCreate, label: petsCopyCreate},
						{icon: screenIconFolderOpen, mark: "folder-open", text: petsCopyInstall, label: petsCopyInstall},
						{icon: screenIconRefresh, mark: "refresh", text: petsCopyRefresh, label: petsCopyRefresh},
						{icon: screenIconFolderOpen, mark: "folder-open", text: petsCopyOpenDir, label: petsCopyOpenDir},
					} {
						spec.height, spec.tip, spec.disabled = screenControlH, notePetSettings, true
						settingsActionButton(c, spec)
					}
					return
				}
				settingsActionButton(c, settingsActionSpec{
					variant: settingsActionPrimary, icon: iconPlus, mark: "plus",
					text: petsCopyCreate, label: petsCopyCreate,
					height: screenControlH, disabled: petsScreen.busy,
				})
				settingsActionButton(c, settingsActionSpec{
					icon: screenIconFolderOpen, mark: "folder-open",
					text: petsCopyInstall, label: petsCopyInstall,
					height: screenControlH, disabled: petsScreen.busy || !loaded,
				})
				settingsActionButton(c, settingsActionSpec{
					icon: screenIconRefresh, mark: "refresh",
					text: petsCopyRefresh, label: petsCopyRefresh,
					height: screenControlH, disabled: petsScreen.busy || !loaded,
				})
				settingsActionButton(c, settingsActionSpec{
					icon: screenIconFolderOpen, mark: "folder-open",
					text: petsCopyOpenDir, label: petsCopyOpenDir,
					height: screenControlH, disabled: petsScreen.busy,
				})
			})
		})
		if live {
			petsEmptyState(c, petsCopyEmptyNote)
			return
		}
		switch {
		case petsScreen.descriptors == nil:
			// 未加载：列表区不渲染（重试在上面的卡里）。
		case len(petsScreen.custom) == 0:
			petsEmptyState(c, petsCopyEmpty)
		default:
			petsCustomList(c)
		}
	})
}

// petsEmptyState 是虚线框里的空态（S-pets-28）。
func petsEmptyState(c *gui.Context, text string) {
	t := c.Theme()
	box := gui.Row(c).Shrink(0).MinHeight(petsEmptyH).Padding(0, petsCardPad).
		AlignItems(gui.Center).Radius(screenRadiusCard)
	box.Draw(func(p *gui.Painter, r gui.Rect) {
		p.StrokeDashed(r, t.Border, screenRadiusCard, screenStroke)
	})
	box.Children(func() {
		screenHint(c, text)
	})
}

// petsCustomList 是自定义 Pet 的列表（S-pets-29 … 32）：名字 + 错误 + 选择 + 删除。
func petsCustomList(c *gui.Context) {
	screenCard(c, func() {
		gui.Column(c).Shrink(0).Children(func() {
			for i, pet := range petsScreen.custom {
				if i > 0 {
					screenRule(c)
				}
				petsCustomRow(c, pet)
			}
		})
	})
}

// petsCustomRow 是一条自定义 Pet：名字与错误在左，选择与删除在右。
func petsCustomRow(c *gui.Context, pet petDescriptor) {
	row := gui.Row(c).Shrink(0).MinHeight(petsRowH).Padding(0, petsCardPad).
		AlignItems(gui.Center).Gap(petsRowGap)
	if pet.id == petsScreen.selected {
		row = row.Background(c.Theme().Accent.Alpha(petsSelectedAlpha))
	}
	row.Children(func() {
		gui.Column(c).Grow(1).MinWidth(0).Gap(petsLineGap).Children(func() {
			petsLabel(c, pet.name)
			if pet.problem != "" {
				petsAlert(c, pet.problem)
			}
		})
		selected := pet.id == petsScreen.selected
		if petsRowButton(c, pet, selected) {
			petsScreen.selected = pet.id
		}
		if settingsActionButton(c, settingsActionSpec{
			variant: settingsActionDanger, text: petsCopyDelete,
			label: petsCopyDelete + "：" + pet.name, height: rowRegular,
			disabled: petsScreen.busy,
		}) {
			petsScreen.deleteFor = pet.id
		}
	})
}

// petsDeleteDialog 是「删除 {name}？」确认框（S-pets-33 … 36）：选中项的说明与
// 其他项不同（删的是当前选择 → 选择切回内置）。
func petsDeleteDialog(c *gui.Context) {
	if petsScreen.deleteFor == "" {
		return
	}
	name := petsDeleteName(petsScreen.deleteFor)
	open := true
	m := gui.Modal(c, &open, func() {
		screenDialogPanel(c, "删除 "+name+"？", func() {
			hint := petsCopyDeleteOnly
			if petsScreen.deleteFor == petsScreen.selected {
				hint = petsCopyDeleteKept
			}
			screenHint(c, hint)
			gui.Row(c).Justify(gui.End).Gap(screenSpace2).Children(func() {
				if settingsActionButton(c, settingsActionSpec{
					text: petsCopyCancel, label: petsCopyCancel, height: rowRegular,
				}) {
					petsScreen.deleteFor = ""
					open = false
				}
				label := petsCopyDelete
				if petsScreen.deleting {
					label = petsCopyDeleting
				}
				if settingsActionButton(c, settingsActionSpec{
					variant: settingsActionDanger, text: label, label: label,
					height: rowRegular, disabled: petsScreen.deleting,
				}) {
					petsRemoveCustom(petsScreen.deleteFor)
					petsScreen.deleteFor = ""
					open = false
				}
			})
		})
	})
	if !open || m.Dismissed() {
		petsScreen.deleteFor = ""
		petsScreen.deleting = false
	}
}

// petsDeleteName 取要删的 Pet 名字；找不到时回落 id。
func petsDeleteName(id string) string {
	for _, pet := range petsScreen.custom {
		if pet.id == id {
			return pet.name
		}
	}
	return id
}

// petsRemoveCustom 从本屏的列表里去掉一个自定义 Pet；删的是当前选择时回落到
// 第一个内置 Pet（参考实现：选择切回内置）。
func petsRemoveCustom(id string) {
	kept := petsScreen.custom[:0]
	for _, pet := range petsScreen.custom {
		if pet.id != id {
			kept = append(kept, pet)
		}
	}
	petsScreen.custom = kept
	if petsScreen.selected == id {
		petsScreen.selected = ""
		for _, pet := range petsScreen.descriptors {
			if pet.available {
				petsScreen.selected = pet.id
				break
			}
		}
	}
}

// ---- 本屏与权限屏共用的小构件 ----

// petsLabel 是行标签：14、中粗、前景，行盒 18（参考实现的 text-sm 行高，
// 比共用 screenLabel 的 21 更紧，列表行高才落得准）。
func petsLabel(c *gui.Context, text string) {
	t := c.Theme()
	petsLabelStyle.style(t, gui.Text(c, text)).TextColor(t.Text).Label(text)
}

// petsLabelStyle 是行标签的字阶。
var petsLabelStyle = fontStep{size: 14, line: 18, weight: 500}

// petsGroupTitle 是卡片之外的分组标题：14、半粗、前景，三级标题（R-PET-081 的
// 无障碍判据：分组标题播报）。
func petsGroupTitle(c *gui.Context, text string) {
	t := c.Theme()
	typeH4.style(t, gui.Text(c, text)).TextColor(t.Text).
		Role(gui.RoleHeading).Level(3).SingleLine()
}

// petsSubhint 是行说明：10.5、次级前景、可复制（R-PET-020 的选中复制）。
func petsSubhint(c *gui.Context, text string) {
	t := c.Theme()
	typeMicro.style(t, gui.Text(c, text)).TextColor(t.TextMuted).Selectable()
}

// petsAlert 是一行警示或操作错误：10.5、警示色、role=alert（R-PET-... 的
// "role=alert" 判据）。文字可复制，不吞原文。
func petsAlert(c *gui.Context, text string) {
	t := c.Theme()
	typeMicro.style(t, gui.Text(c, text)).TextColor(t.Danger).
		Selectable().Label(text)
}

// petsNoteBar 是提示条：次要面 + 1px 主题描边 + 控件圆角，内边距 7/10。
func petsNoteBar(c *gui.Context, text string) {
	t := c.Theme()
	box := gui.Column(c).Shrink(0).Background(t.SurfaceHover).
		Radius(screenRadiusCtl)
	box.Draw(func(p *gui.Painter, r gui.Rect) {
		p.Stroke(r, t.Border, screenRadiusCtl, screenStroke)
	})
	box.Children(func() {
		gui.Row(c).Shrink(0).Padding(petsNotePadY, petsNotePadX).Children(func() {
			screenHint(c, text)
		})
	})
}

// settingsActionVariant 是动作按钮的三态。
type settingsActionVariant int

const (
	settingsActionOutline settingsActionVariant = iota
	settingsActionPrimary
	settingsActionDanger
)

// settingsActionSpec 是一个自绘动作按钮：文字 +（可选）图标，可禁用。
type settingsActionSpec struct {
	variant  settingsActionVariant
	text     string
	icon     *gui.SVG
	mark     string
	label    string
	tip      string
	height   float32
	disabled bool
}

// settingsActionButton 画一个动作按钮并返回它是否被按下。五态：默认 / 悬停 /
// 按下 / 聚焦（框架焦点环）/ 禁用（整棵子树 0.5，与框架一致）。
//
// 它放在本文件里，是因为宠物屏与权限屏都要「带图标 + 可禁用」的按钮，而这两个
// 屏的文件之外没有合适的共用位置（settings_*_screen.go 里的构件按屏归属）。
// 下一个用到它的屏可以把它挪进三屏共用构件。
func settingsActionButton(c *gui.Context, spec settingsActionSpec) bool {
	t := c.Theme()
	height := spec.height
	if height == 0 {
		height = screenControlH
	}
	label := spec.label
	if label == "" {
		label = spec.text
	}
	var b gui.Element
	// 外面套一层行：按钮在任何容器里都按内容宽度排，不被列拉成通栏
	// （设置屏的按钮都是"文字 + 内边距"宽，不是卡片宽）。
	gui.Row(c).Shrink(0).Children(func() {
		b = gui.ButtonBase(c).Shrink(0).Height(height).Padding(controlPadV, controlPadH).
			Radius(screenRadiusCtl).AlignItems(gui.Center).
			Label(label).Disabled(spec.disabled)
		if spec.tip != "" {
			b = b.Tooltip(spec.tip)
		}
		base := t.Background
		switch spec.variant {
		case settingsActionPrimary:
			base = t.Accent
		case settingsActionDanger:
			base = t.Danger
		}
		bg := base
		switch {
		case spec.disabled:
		case spec.variant == settingsActionPrimary && b.Pressed():
			bg = t.AccentPressed
		case spec.variant == settingsActionPrimary && b.Hovered():
			bg = t.AccentHover
		case b.Pressed():
			bg = t.SurfacePressed
		case b.Hovered():
			bg = t.SurfaceHover
		}
		b = b.Background(bg)
		if spec.disabled {
			b = b.Opacity(petsDisabled)
		}
		if spec.variant == settingsActionOutline {
			b.Draw(func(p *gui.Painter, r gui.Rect) {
				p.Stroke(r, t.Border, screenRadiusCtl, screenStroke)
			})
		}
		ink := t.Text
		if spec.variant != settingsActionOutline {
			ink = t.AccentText
		}
		b.Children(func() {
			gui.Row(c).Gap(screenSpace15).AlignItems(gui.Center).Children(func() {
				if spec.icon != nil {
					screenIcon(c, spec.icon, screenIconInline, ink, spec.mark)
				}
				screenValueStyle.style(t, gui.Text(c, spec.text)).TextColor(ink).SingleLine()
			})
		})
	})
	return b.Clicked()
}
