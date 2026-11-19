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

// settingsSectionDef 是导航里的一节：标识、界面语、图标与深链标识。
type settingsSectionDef struct {
	id    settingsSectionID
	label string
	icon  *gui.SVG
	// slug 是深链里用的稳定标识（dimagent://settings/<slug>）：界面语可能改，
	// 链接不能跟着改。
	slug string
}

// settingsSectionDefs 是导航的唯一来源：顺序即上屏顺序，19 项。
var settingsSectionDefs = []settingsSectionDef{
	{secAccount, "账户", iconUserCircle, "account"},
	{secUsage, "用量", iconChartBar, "usage"},
	{secPersonalization, "个性化", iconSparkles, "personalization"},
	{secAppearance, "外观", iconSwatch, "appearance"},
	{secGeneral, "通用", iconGear, "general"},
	{secKeybindings, "快捷键", iconKeyboard, "keybindings"},
	{secProviders, "模型供应", iconChip, "providers"},
	{secModality, "多模态", iconSliders, "modality"},
	{secRuntime, "运行环境", iconTerminal, "runtime"},
	{secMCP, "MCP", iconWrench, "mcp"},
	{secNotifications, "通知", iconBell, "notifications"},
	{secPets, "宠物", iconPaw, "pets"},
	{secPermissions, "权限", iconShield, "permissions"},
	{secGit, "Git", iconBranch, "git"},
	{secHooks, "Hooks", iconWebhook, "hooks"},
	{secPluginSettings, "插件设置", iconPlug, "plugin-settings"},
	{secExternalAgents, "外部 Agent", iconBot, "external-agents"},
	{secArchived, "归档会话", iconArchive, "archived"},
	{secAbout, "关于", iconInfo, "about"},
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

// DefaultSettingsSectionSlug 是深链省略分节时进的那一节（账户）。
const DefaultSettingsSectionSlug = "account"

// settingsSectionBySlug 按深链标识取一节；空串按第一节处理。
func settingsSectionBySlug(slug string) (settingsSectionDef, bool) {
	if slug == "" {
		return settingsSectionDefs[0], true
	}
	for _, section := range settingsSectionDefs {
		if section.slug == slug {
			return section, true
		}
	}
	return settingsSectionDef{}, false
}

// SettingsSectionKnown 报告深链里的分节标识是否认识。入口在开窗之前先问一次：
// 未知分节只写回执，不开出一个空窗口。
func SettingsSectionKnown(slug string) bool {
	_, ok := settingsSectionBySlug(slug)
	return ok
}
