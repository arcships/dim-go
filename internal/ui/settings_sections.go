package ui

import gui "github.com/egoist/mygo/ui"

// 设置左导航的 19 节，顺序与名称取自参考产品的 sections.ts：
// 账户 → 用量 → 个性化 → 外观 → 通用 → 快捷键 → 模型供应 → 多模态 → 运行环境 →
// MCP → 通知 → 宠物 → 权限 → Git → Hooks → 插件设置 → 外部 Agent → 归档会话 → 关于。
//
// 一节只有三样东西：标识、界面语、图标。内容由 settingsContent 按标识路由；
// 已重建的分节显示真内容，其余显示「尚未按参考产品重建」的占位，不放控件。

// settingsSectionID 是分节标识：取值本身不重要，顺序由 settingsSectionDefs 定。
type settingsSectionID int

const (
	secAccount settingsSectionID = iota
	secUsage
	secPersonalization
	secAppearance
	secGeneral
	secKeybindings
	secProviders
	secModality
	secRuntime
	secMCP
	secNotifications
	secPets
	secPermissions
	secGit
	secHooks
	secPluginSettings
	secExternalAgents
	secArchived
	secAbout
)

// settingsSectionDef 是导航里的一节：标识、界面语、图标。
type settingsSectionDef struct {
	id    settingsSectionID
	label string
	icon  *gui.SVG
}

// settingsSectionDefs 是导航的唯一来源：顺序即上屏顺序，19 项。
var settingsSectionDefs = []settingsSectionDef{
	{secAccount, "账户", iconUserCircle},
	{secUsage, "用量", iconChartBar},
	{secPersonalization, "个性化", iconSparkles},
	{secAppearance, "外观", iconSwatch},
	{secGeneral, "通用", iconGear},
	{secKeybindings, "快捷键", iconKeyboard},
	{secProviders, "模型供应", iconChip},
	{secModality, "多模态", iconSliders},
	{secRuntime, "运行环境", iconTerminal},
	{secMCP, "MCP", iconWrench},
	{secNotifications, "通知", iconBell},
	{secPets, "宠物", iconPaw},
	{secPermissions, "权限", iconShield},
	{secGit, "Git", iconBranch},
	{secHooks, "Hooks", iconWebhook},
	{secPluginSettings, "插件设置", iconPlug},
	{secExternalAgents, "外部 Agent", iconBot},
	{secArchived, "归档会话", iconArchive},
	{secAbout, "关于", iconInfo},
}

// settingsSectionByID 取一节；标识认不出时回落到第一节，与参考产品
// resolveSettingsSection 的回落一致（不会出现空内容区）。
func settingsSectionByID(id settingsSectionID) settingsSectionDef {
	for _, section := range settingsSectionDefs {
		if section.id == id {
			return section
		}
	}
	return settingsSectionDefs[0]
}
