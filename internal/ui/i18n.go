// This file owns the shell's interface language: the Locale type, the message
// table of each language and the lookup every region uses.
//
// It translates the shell's own text only. Session titles, tool output, App
// Server error strings and protocol method names are facts of another layer and
// stay as they came; a WIP reason names the wire method it waits on literally in
// every language. English is the default and the fallback: a key a language does
// not carry is drawn in English, so a missing translation never leaves a blank
// row or an internal key in the window.
package ui

import "strings"

// Locale is the language of the shell's own text. It is a desktop preference,
// not a session fact, and it changes no layout: only the text of a region.
type Locale string

const (
	// LocaleEN is English, the default and the fallback language.
	LocaleEN Locale = "en"
	// LocaleZHCN is Simplified Chinese.
	LocaleZHCN Locale = "zh-CN"
	// DefaultLocale is the language a window opens in.
	DefaultLocale Locale = LocaleEN
)

// String reports the locale's tag, for a caller's log or saved preference.
func (l Locale) String() string {
	return string(l)
}

// Name is the language's own name, as the language chooser shows it: a reader
// looking for their own language finds it written the way they write it.
func (l Locale) Name() string {
	if resolveLocale(l) == LocaleZHCN {
		return "简体中文"
	}
	return "English"
}

// resolveLocale folds an empty or unknown preference onto the default, so a
// hand-edited or not-yet-saved value draws the shell in English rather than in
// no language at all.
func resolveLocale(l Locale) Locale {
	if l == LocaleZHCN {
		return LocaleZHCN
	}
	return DefaultLocale
}

// Strings is one language's message table: a key to the text drawn for it.
type Strings map[string]string

// text resolves one key in this table. A key this language does not carry falls
// back to the English table and then to the key itself, so a translation gap is
// visible in the window and is never a silent empty string.
func (s Strings) text(key string) string {
	if text, ok := s[key]; ok && text != "" {
		return text
	}
	if text, ok := stringsEN[key]; ok && text != "" {
		return text
	}
	return key
}

// localeStrings is the table of each language the shell carries.
var localeStrings = map[Locale]Strings{
	LocaleEN:   stringsEN,
	LocaleZHCN: stringsZHCN,
}

// stringsFor returns the table for a locale; an unknown or empty locale gets
// English.
func stringsFor(l Locale) Strings {
	if table, ok := localeStrings[resolveLocale(l)]; ok {
		return table
	}
	return stringsEN
}

// tr is the lookup a region calls: the text of key in locale l.
func tr(l Locale, key string) string {
	return stringsFor(l).text(key)
}

// The message keys. They are grouped by region and are the only names a region
// uses to read text, so no region passes a display string as a lookup key.
const (
	keyNoSession           = "shell.noSession"
	keyComposerPlaceholder = "composer.placeholder"

	keySidebarNavigation    = "sidebar.navigation"
	keySidebarUpdate        = "sidebar.update"
	keySidebarNewSession    = "sidebar.newSession"
	keySidebarSearch        = "sidebar.search"
	keySidebarKeys          = "sidebar.keys"
	keySidebarSessions      = "sidebar.sessions"
	keySidebarGroup         = "sidebar.group"
	keySidebarEmpty         = "sidebar.empty"
	keySidebarUngrouped     = "sidebar.ungrouped"
	keySidebarBranches      = "sidebar.branches"
	keySidebarBranchesRO    = "sidebar.branches.readOnly"
	keySidebarBranchCurrent = "sidebar.branch.current"
	keySidebarBranchSuffix  = "sidebar.branch.currentSuffix"
	keySidebarRefresh       = "sidebar.branch.refresh"
	keySidebarEndBranch     = "sidebar.branch.end"
	keySidebarThreadActions = "sidebar.threadActions"
	keySidebarPin           = "sidebar.pin"
	keySidebarRename        = "sidebar.rename"
	keySidebarArchive       = "sidebar.archive"
	keySidebarDelete        = "sidebar.delete"
	keyReasonDelete         = "reason.delete"

	keyReasonCreateSession   = "reason.createSession"
	keyReasonApproval        = "reason.approval"
	keyReasonAttachment      = "reason.attachment"
	keyReasonReconnect       = "reason.reconnect"
	keyReasonStop            = "reason.stop"
	keyReasonBranches        = "reason.branches"
	keyReasonUpdate          = "reason.update"
	keyReasonSearch          = "reason.search"
	keyReasonGroup           = "reason.group"
	keyReasonPin             = "reason.pin"
	keyReasonRename          = "reason.rename"
	keyReasonArchive         = "reason.archive"
	keyReasonKeysEntry       = "reason.keysEntry"
	keyReasonEndBranch       = "reason.endBranch"
	keyReasonRefreshBranches = "reason.refreshBranches"
	keyReasonModel           = "reason.model"
	keyReasonMode            = "reason.mode"
	keyReasonOpenInEditor    = "reason.openInEditor"
	keyReasonSkin            = "reason.skin"
	keyReasonChrome          = "reason.chrome"

	keyHeaderStop          = "header.stop"
	keyHeaderReconnect     = "header.reconnect"
	keyHeaderSettings      = "header.settings"
	keyHeaderConversation  = "header.conversation"
	keyHeaderRegion        = "header.region"
	keyHeaderMenu          = "header.menu"
	keyHeaderMenuRefresh   = "header.menu.refreshBranches"
	keyHeaderOpenInEditor  = "header.openInEditor"
	keyHeaderDeleteTitle   = "header.delete.title"
	keyHeaderDeleteMessage = "header.delete.message"
	keyHeaderDeleteConfirm = "header.delete.confirm"
	keyHeaderDeleteCancel  = "header.delete.cancel"
	keyHeaderSessionClosed = "header.sessionClosed"

	keyComposerHint   = "composer.hint"
	keyComposerLabel  = "composer.label"
	keyComposerAttach = "composer.attach"
	keyComposerMode   = "composer.mode"
	keyComposerModel  = "composer.model"
	keyComposerSend   = "composer.send"

	// keyComposerUnconfigured is the muted label beside Send: the shell has no
	// model fact, so the composer states that none is configured instead of
	// naming one. keyComposerStatus names the conversation's status band.
	keyComposerUnconfigured = "composer.model.unconfigured"
	keyComposerBusy         = "composer.busy"
	keyComposerStatus       = "composer.status"

	keySettingsRegion           = "settings.region"
	keySettingsTitle            = "settings.title"
	keySettingsLocalNote        = "settings.localNote"
	keySettingsBack             = "settings.back"
	keySettingsThemeGroup       = "settings.theme.group"
	keySettingsSystem           = "settings.theme.system"
	keySettingsLight            = "settings.theme.light"
	keySettingsDark             = "settings.theme.dark"
	keySettingsLanguage         = "settings.language"
	keySettingsLanguageNote     = "settings.language.note"
	keySettingsLanguageRow      = "settings.language.row"
	keySettingsText             = "settings.text"
	keySettingsFontFamily       = "settings.fontFamily"
	keySettingsFontNote         = "settings.font.note"
	keySettingsFontSystem       = "settings.font.system"
	keySettingsFontMonospace    = "settings.font.monospace"
	keySettingsCodeFont         = "settings.codeFont"
	keySettingsCodeFontSame     = "settings.codeFont.same"
	keySettingsCodeFontGap      = "settings.codeFont.gap"
	keySettingsFontSize         = "settings.fontSize"
	keySettingsSizeXS           = "settings.size.xs"
	keySettingsSizeSmall        = "settings.size.small"
	keySettingsSizeMedium       = "settings.size.medium"
	keySettingsSizeLarge        = "settings.size.large"
	keySettingsSizeXL           = "settings.size.xl"
	keySettingsCodeFontSize     = "settings.codeFontSize"
	keySettingsCodeTheme        = "settings.codeTheme"
	keySettingsCodeThemeNote    = "settings.codeTheme.note"
	keySettingsCodeThemeSame    = "settings.codeTheme.same"
	keySettingsCodePreviewLight = "settings.codeTheme.preview.light"
	keySettingsCodePreviewDark  = "settings.codeTheme.preview.dark"
	keySettingsPreviewGap       = "settings.preview.gap"
	keySettingsSkinSection      = "settings.skin.section"
	keySettingsSkinNote         = "settings.skin.note"
	keySettingsSkinCard         = "settings.skin.card"
	keySettingsSkinNew          = "settings.skin.new"
	keySettingsSkinOpen         = "settings.skin.open"

	keyCapabilityGroup             = "capability.group"
	keyCapabilityCreate            = "capability.create.action"
	keyCapabilityCreateGap         = "capability.create.gap"
	keyCapabilityBranchSwitch      = "capability.branchSwitch.action"
	keyCapabilityBranchSwitchGap   = "capability.branchSwitch.gap"
	keyCapabilityBranchEnd         = "capability.branchEnd.action"
	keyCapabilityBranchEndGap      = "capability.branchEnd.gap"
	keyCapabilityBranchRefresh     = "capability.branchRefresh.action"
	keyCapabilityBranchRefreshGap  = "capability.branchRefresh.gap"
	keyCapabilityKeys              = "capability.keys.action"
	keyCapabilityKeysGap           = "capability.keys.gap"
	keyCapabilityApprovals         = "capability.approvals.action"
	keyCapabilityApprovalsGap      = "capability.approvals.gap"
	keyCapabilityAttachments       = "capability.attachments.action"
	keyCapabilityAttachmentsGap    = "capability.attachments.gap"
	keyCapabilityConnectionRecover = "capability.connectionRecovery.action"
	keyCapabilityConnectionGap     = "capability.connectionRecovery.gap"
	keyCapabilityTurnCancel        = "capability.turnCancellation.action"
	keyCapabilityTurnCancelGap     = "capability.turnCancellation.gap"

	// Settings navigation: the section names and the WIP reason of each page the
	// shell cannot open yet.
	keySettingsNavigation = "settings.navigation"
	keySettingsBackToList = "settings.backToList"
	keyContextPanelToggle = "header.contextPanel"

	// Layout pieces the shell draws around the conversation: the empty-state
	// hero, the composer's context row and the sidebar's account footer.
	keyHeroTitle            = "hero.title"
	keyHeroSubtitle         = "hero.subtitle"
	keyComposerWorkspace    = "composer.workspace"
	keyComposerPermission   = "composer.permission"
	keyComposerBranch       = "composer.branch"
	keyComposerContextUsage = "composer.contextUsage"
	keySidebarScheduled     = "sidebar.scheduled"
	keySidebarSkills        = "sidebar.skills"
	keySidebarAccount       = "sidebar.account"
	keySidebarSettingsEntry = "sidebar.settings"

	// The centered search palette: its input's placeholder and accessible
	// name, the recent-chats heading and the empty state.
	keySearchPlaceholder = "search.placeholder"
	keySearchRecent      = "search.recent"
	keySearchEmpty       = "search.empty"
)

// stringsEN is the English table, the default and the fallback. Its entries are
// the shell's constants, so an English frame draws exactly what those constants
// name and a key added to one table cannot silently drift from the other.
var stringsEN = Strings{
	keyNoSession:           "No session",
	keyComposerPlaceholder: "Send a message…",

	keySidebarNavigation:       "Session sidebar",
	keySidebarUpdate:           "Update",
	keySidebarNewSession:       "New session",
	keySidebarSearch:           "Search",
	keySidebarKeys:             "Session keys",
	keySidebarSessions:         "Sessions",
	keySidebarGroup:            "Group",
	keySidebarEmpty:            "No sessions yet",
	keySidebarUngrouped:        ungroupedLabel,
	keySidebarBranches:         "Branches",
	keySidebarBranchesRO:       "Branches (read-only)",
	keySidebarBranchCurrent:    "current",
	keySidebarBranchSuffix:     " (current)",
	keySidebarRefresh:          "Refresh",
	keySidebarEndBranch:        "End branch",
	keySidebarThreadActions:    "Thread actions (unavailable)",
	keySidebarPin:              "Pin",
	keySidebarRename:           "Rename",
	keySidebarArchive:          "Archive",
	keySidebarDelete:           "Delete session",
	keyReasonDelete:            reasonDelete,
	keySidebarScheduled:        "Scheduled tasks",
	keySidebarSkills:           "Skills and plugins",
	keySidebarAccount:          "Sign in to Dim",
	keySidebarSettingsEntry:    "Settings",
	keyHeroTitle:               "Start building",
	keyHeroSubtitle:            "Ask DimAgent to change code, run a command or explain a session.",
	keyComposerWorkspace:       "Workspace",
	keyComposerPermission:      "Default permissions",
	keyComposerBranch:          "Session branch",
	keyComposerContextUsage:    "Context usage",
	keySettingsNavigation:      "Settings sections",
	keyContextPanelToggle:      "Context panel",
	keySearchPlaceholder:       "Search chats",
	keySearchRecent:            "Recent chats",
	keySearchEmpty:             "No chats found",
	"settings.section.account": "Account", "settings.section.usage": "Usage",
	"settings.section.personalization": "Personalization", "settings.section.general": "General",
	"settings.section.shortcuts": "Shortcuts", "settings.section.models": "Models",
	"settings.section.multimodal": "Multimodal", "settings.section.runtime": "Runtime",
	"settings.section.mcp": "MCP", "settings.section.notifications": "Notifications",
	"settings.section.pet": "Pet", "settings.section.permissions": "Permissions",
	"settings.section.git": "Git", "settings.section.hooks": "Hooks",
	"settings.section.plugins": "Plugins", "settings.section.externalAgent": "External Agent",
	"settings.gap.account":         "Unavailable: the shell has no account client or page yet, so it cannot show an account or sign in.",
	"settings.gap.usage":           "Unavailable: the shell has no usage client or page yet; no App Server method reports usage here.",
	"settings.gap.personalization": "Unavailable: the personalization page is not built yet; the shell has no client for the interface-mode, ACP-session, target-app, assistant-model or subagent-model settings it shows.",
	"settings.gap.general":         "Unavailable: the general settings page is not built yet; the shell has no client for its desktop preferences.",
	"settings.gap.shortcuts":       "Unavailable: the shortcuts page is not built yet; the shell has no keybinding client or editor.",
	"settings.gap.models":          "Unavailable: the models page is not built yet; the shell has no model-provider client or page.",
	"settings.gap.multimodal":      "Unavailable: the multimodal page is not built yet; the shell has no multimodal client or page.",
	"settings.gap.runtime":         "Unavailable: the runtime page is not built yet; the shell has no runtime client or page.",
	"settings.gap.mcp":             "Unavailable: the MCP page is not built yet; the shell has no MCP server client or page.",
	"settings.gap.notifications":   "Unavailable: the notifications page is not built yet; the shell has no notification client or page.",
	"settings.gap.pet":             "Unavailable: the pet page is not built yet; the shell has no pet client or page.",
	"settings.gap.permissions":     "Unavailable: the permissions page is not built yet; the shell shows approvals but has no rule editor, permissions client or page.",
	"settings.gap.git":             "Unavailable: the Git page is not built yet; the shell has no Git client or page.",
	"settings.gap.hooks":           "Unavailable: the hooks page is not built yet; the shell has no hooks client or page.",
	"settings.gap.plugins":         "Unavailable: the plugins page is not built yet; the shell has no plugin client or page.",
	"settings.gap.externalAgent":   "Unavailable: the external agent page is not built yet; the shell has no external-agent client or page.",

	keyReasonCreateSession:   reasonCreateSession,
	keyReasonApproval:        reasonApproval,
	keyReasonAttachment:      reasonAttachment,
	keyReasonReconnect:       reasonReconnect,
	keyReasonStop:            reasonStop,
	keyReasonBranches:        reasonBranches,
	keyReasonUpdate:          reasonUpdate,
	keyReasonSearch:          reasonSearchEntry,
	keyReasonGroup:           reasonGroup,
	keyReasonPin:             reasonPin,
	keyReasonRename:          reasonRename,
	keyReasonArchive:         reasonArchive,
	keyReasonKeysEntry:       reasonKeysEntry,
	keyReasonEndBranch:       reasonEndBranch,
	keyReasonRefreshBranches: reasonRefreshBranches,
	keyReasonModel:           reasonModel,
	keyReasonMode:            reasonMode,
	keyReasonOpenInEditor:    reasonOpenInEditor,
	keyReasonSkin:            reasonSkin,
	keyReasonChrome:          reasonChrome,

	keyHeaderStop:          "Stop",
	keyHeaderReconnect:     "Reconnect",
	keyHeaderSettings:      "Settings",
	keyHeaderConversation:  "Conversation",
	keyHeaderRegion:        "Conversation header",
	keyHeaderMenu:          "Session menu",
	keyHeaderMenuRefresh:   "Refresh branches",
	keyHeaderOpenInEditor:  "Open in VS Code",
	keyHeaderDeleteTitle:   "Delete session?",
	keyHeaderDeleteMessage: "This asks the App Server to delete the selected session. It cannot be undone here.",
	keyHeaderDeleteConfirm: "Delete",
	keyHeaderDeleteCancel:  "Cancel",
	keyHeaderSessionClosed: "Session closed",

	keyComposerHint:         composerHint,
	keyComposerLabel:        "Composer",
	keyComposerAttach:       "Attach file",
	keyComposerMode:         "Mode",
	keyComposerModel:        "Model",
	keyComposerUnconfigured: "Not configured",
	keyComposerStatus:       "Composer status",
	keyComposerSend:         "Send",

	keySettingsRegion:           "Settings",
	keySettingsTitle:            "Appearance",
	keySettingsLocalNote:        settingsLocalNote,
	keySettingsBack:             "Back",
	keySettingsThemeGroup:       "Theme",
	keySettingsSystem:           "System",
	keySettingsLight:            "Light",
	keySettingsDark:             "Dark",
	keySettingsLanguage:         "Language",
	keySettingsLanguageNote:     "The language of the shell's own text.",
	keySettingsLanguageRow:      "Interface language",
	keySettingsText:             "Font",
	keySettingsFontFamily:       "UI font",
	keySettingsFontNote:         "Used for interface text and Markdown.",
	keySettingsFontSystem:       "System font",
	keySettingsFontMonospace:    "Monospace",
	keySettingsCodeFont:         "Code font",
	keySettingsCodeFontSame:     "Follows the UI font",
	keySettingsCodeFontGap:      "Read-only: this shell draws the interface and code in one family; a separate code font needs a theme preference the tokens do not carry.",
	keySettingsFontSize:         "UI size",
	keySettingsSizeXS:           "XS",
	keySettingsSizeSmall:        "S",
	keySettingsSizeMedium:       "M",
	keySettingsSizeLarge:        "L",
	keySettingsSizeXL:           "XL",
	keySettingsCodeFontSize:     "Code size",
	keySettingsCodeTheme:        "Code theme",
	keySettingsCodeThemeNote:    "Unavailable: this shell draws code with the interface theme and keeps no separate code-theme preference.",
	keySettingsCodeThemeSame:    "Same as interface",
	keySettingsCodePreviewLight: "Light code",
	keySettingsCodePreviewDark:  "Dark code",
	keySettingsPreviewGap:       "Preview unavailable: no code theme is stored.",
	keySettingsSkinSection:      "My skin",
	keySettingsSkinNote:         settingsSkinNote,
	keySettingsSkinCard:         "Skin",
	keySettingsSkinNew:          "New from image",
	keySettingsSkinOpen:         "Open folder",

	keyCapabilityGroup:             "Unavailable capabilities",
	keyCapabilityCreate:            "Session creation",
	keyCapabilityCreateGap:         "helper exists; shell has no create callback",
	keyCapabilityBranchSwitch:      "Branch switching",
	keyCapabilityBranchSwitchGap:   "helper exists; shell has no branch selection callback",
	keyCapabilityBranchEnd:         "Branch end",
	keyCapabilityBranchEndGap:      "helper exists; shell has no branch end callback",
	keyCapabilityBranchRefresh:     "Branch refresh",
	keyCapabilityBranchRefreshGap:  "helper exists; shell has no branch refresh callback",
	keyCapabilityKeys:              "Session keys",
	keyCapabilityKeysGap:           "helper exists; shell has no key browser callback",
	keyCapabilityApprovals:         "Approvals",
	keyCapabilityApprovalsGap:      "helper exists; window has no approval callback",
	keyCapabilityAttachments:       "Attachments",
	keyCapabilityAttachmentsGap:    "helper exists; shell has no file picker and keeps unknown upload results disabled",
	keyCapabilityConnectionRecover: "Connection recovery",
	keyCapabilityConnectionGap:     "helper exists; shell not wired",
	keyCapabilityTurnCancel:        "Turn cancellation",
	keyCapabilityTurnCancelGap:     "helper exists; shell not wired",
}

// stringsZHCN is the Simplified Chinese table. Protocol and method names in a
// reason stay literal (session.create, approval.respond, ...): they name the
// wire entry the entry waits on, not a phrase to translate.
var stringsZHCN = Strings{
	keyNoSession:           "未选择会话",
	keyComposerPlaceholder: "发送消息…",

	keySidebarNavigation:       "会话侧栏",
	keySidebarUpdate:           "更新",
	keySidebarNewSession:       "新建会话",
	keySidebarSearch:           "搜索",
	keySidebarKeys:             "会话键",
	keySidebarSessions:         "会话",
	keySidebarGroup:            "分组",
	keySidebarEmpty:            "暂无会话",
	keySidebarUngrouped:        "无项目",
	keySidebarBranches:         "分支",
	keySidebarBranchesRO:       "分支（只读）",
	keySidebarBranchCurrent:    "当前",
	keySidebarBranchSuffix:     "（当前）",
	keySidebarRefresh:          "刷新",
	keySidebarEndBranch:        "结束分支",
	keySidebarThreadActions:    "会话操作（不可用）",
	keySidebarPin:              "置顶",
	keySidebarScheduled:        "定时任务",
	keySidebarSkills:           "技能/插件",
	keySidebarAccount:          "登录 Dim",
	keySidebarSettingsEntry:    "设置",
	keyHeroTitle:               "开始构建",
	keyHeroSubtitle:            "让 DimAgent 改代码、跑命令，或解释一个会话。",
	keyComposerWorkspace:       "工作区",
	keyComposerPermission:      "默认权限",
	keyComposerBranch:          "会话分支",
	keyComposerContextUsage:    "上下文占用",
	keySettingsNavigation:      "设置分区",
	keyContextPanelToggle:      "上下文面板",
	keySearchPlaceholder:       "搜索聊天",
	keySearchRecent:            "最近聊天",
	keySearchEmpty:             "未找到聊天",
	"settings.section.account": "账户", "settings.section.usage": "用量",
	"settings.section.personalization": "个性化", "settings.section.general": "通用",
	"settings.section.shortcuts": "快捷键", "settings.section.models": "模型供应",
	"settings.section.multimodal": "多模态", "settings.section.runtime": "运行环境",
	"settings.section.mcp": "MCP", "settings.section.notifications": "通知",
	"settings.section.pet": "宠物", "settings.section.permissions": "权限",
	"settings.section.git": "Git", "settings.section.hooks": "Hooks",
	"settings.section.plugins": "插件设置", "settings.section.externalAgent": "外部 Agent",
	"settings.gap.account":         "不可用：本 shell 还没有账户客户端或页面，无法显示账户或登录。",
	"settings.gap.usage":           "不可用：本 shell 还没有用量客户端或页面，App Server 也没有报告用量的方法。",
	"settings.gap.personalization": "不可用：个性化页面尚未实现；本 shell 没有界面模式、ACP 会话、目标应用、助手模型或子代理模型的客户端。",
	"settings.gap.general":         "不可用：通用设置页面尚未实现；本 shell 没有对应的桌面偏好客户端。",
	"settings.gap.shortcuts":       "不可用：快捷键页面尚未实现；本 shell 没有键位客户端或编辑器。",
	"settings.gap.models":          "不可用：模型页面尚未实现；本 shell 没有模型供应客户端或页面。",
	"settings.gap.multimodal":      "不可用：多模态页面尚未实现；本 shell 没有多模态客户端或页面。",
	"settings.gap.runtime":         "不可用：运行环境页面尚未实现；本 shell 没有运行时客户端或页面。",
	"settings.gap.mcp":             "不可用：MCP 页面尚未实现；本 shell 没有 MCP 服务器客户端或页面。",
	"settings.gap.notifications":   "不可用：通知页面尚未实现；本 shell 没有通知客户端或页面。",
	"settings.gap.pet":             "不可用：宠物页面尚未实现；本 shell 没有宠物客户端或页面。",
	"settings.gap.permissions":     "不可用：权限页面尚未实现；本 shell 有审批，但没有规则编辑器、权限客户端或页面。",
	"settings.gap.git":             "不可用：Git 页面尚未实现；本 shell 没有 Git 客户端或页面。",
	"settings.gap.hooks":           "不可用：Hooks 页面尚未实现；本 shell 没有 hooks 客户端或页面。",
	"settings.gap.plugins":         "不可用：插件设置页面尚未实现；本 shell 没有插件客户端或页面。",
	"settings.gap.externalAgent":   "不可用：外部 Agent 页面尚未实现；本 shell 没有外部 Agent 客户端或页面。",
	keySidebarRename:               "重命名",
	keySidebarArchive:              "归档",
	keySidebarDelete:               "删除会话",
	keyReasonDelete:                reasonDelete,

	keyReasonCreateSession:   "不可用：窗口还没有 session.create 回调。",
	keyReasonApproval:        "不可用：shell 还没有接到 approval.respond 的审批回调。",
	keyReasonAttachment:      "不可用：attachment.put 还没有接到文件选择器；上传结果可能未知。",
	keyReasonReconnect:       "不可用：shell 还没有接到 supervisor 的重连。",
	keyReasonStop:            "不可用：shell 还没有 session.stop 回调。",
	keyReasonBranches:        "不可用：分支切换/切点还没有接线。",
	keyReasonUpdate:          "不可用：shell 还没有接到应用的更新检查。",
	keyReasonSearch:          "不可用：窗口还没有搜索浮层的打开回调，侧栏入口无法打开搜索对话框。",
	keyReasonGroup:           "不可用：会话行还没有项目或更新时间，shell 无法按项目或日期分组。",
	keyReasonPin:             "不可用：没有经过验证的 App Server RPC 可以置顶会话，shell 也不保留本地置顶列表。",
	keyReasonRename:          "不可用：session.rename 是通过 session.methods 访问的会话方法，shell 还没有调用。",
	keyReasonArchive:         "不可用：shell 还没有桌面本地的隐藏会话列表。",
	keyReasonKeysEntry:       "不可用：没有载入 session.keys 快照时，shell 还没有键浏览器回调。",
	keyReasonEndBranch:       "不可用：shell 还没有 session.branch_end 回调。",
	keyReasonRefreshBranches: "不可用：shell 还没有 session.branches 刷新回调。",
	keyReasonModel:           "不可用：ComposerView 没有模型列表或当前模型；App Server 的模型信息还没有传给输入框。",
	keyReasonMode:            "不可用：ComposerView 没有 agent 模式；shell 还没有向输入框提供模式信息。",
	keyReasonOpenInEditor:    "不可用：shell 还没有用于在 VS Code 中打开会话的编辑器客户端或工作区信息。",
	keyReasonSkin:            "不可用：皮肤需要 dim-electron 的 skin.json 解析器和窗口区域；主题行只能选择 system、light 或 dark。",
	keyReasonChrome:          "不可用：chrome 主题需要 dim-electron 的六套历史样式表；shell 不会绘制假的 chrome。",

	keyHeaderStop:          "停止",
	keyHeaderReconnect:     "重新连接",
	keyHeaderSettings:      "设置",
	keyHeaderConversation:  "会话",
	keyHeaderRegion:        "会话顶栏",
	keyHeaderMenu:          "会话菜单",
	keyHeaderMenuRefresh:   "刷新分支",
	keyHeaderOpenInEditor:  "在 VS Code 中打开",
	keyHeaderDeleteTitle:   "删除会话？",
	keyHeaderDeleteMessage: "这会请求 App Server 删除选中的会话，无法在这里撤销。",
	keyHeaderDeleteConfirm: "删除",
	keyHeaderDeleteCancel:  "取消",
	keyHeaderSessionClosed: "会话已关闭",

	keyComposerHint:         "Enter 发送 · Shift+Enter 换行",
	keyComposerLabel:        "输入框",
	keyComposerAttach:       "添加附件",
	keyComposerMode:         "模式",
	keyComposerModel:        "模型",
	keyComposerUnconfigured: "未配置",
	keyComposerStatus:       "输入框状态",
	keyComposerSend:         "发送",

	keySettingsRegion:           "设置",
	keySettingsTitle:            "外观",
	keySettingsLocalNote:        "这些偏好保存在本机桌面端。它们只改变 shell 的绘制方式，不改变任何会话、运行或审批。",
	keySettingsBack:             "返回",
	keySettingsThemeGroup:       "主题",
	keySettingsSystem:           "跟随系统",
	keySettingsLight:            "浅色",
	keySettingsDark:             "深色",
	keySettingsLanguage:         "程序语言",
	keySettingsLanguageNote:     "shell 自身文字的语言。",
	keySettingsLanguageRow:      "界面语言",
	keySettingsText:             "字体",
	keySettingsFontFamily:       "界面字体",
	keySettingsFontNote:         "用于界面文字与 Markdown 正文。",
	keySettingsFontSystem:       "系统字体",
	keySettingsFontMonospace:    "等宽字体",
	keySettingsCodeFont:         "代码字体",
	keySettingsCodeFontSame:     "跟随界面字体",
	keySettingsCodeFontGap:      "只读：本 shell 的界面与代码共用一种字体族；独立代码字体需要 tokens 尚未携带的主题偏好。",
	keySettingsFontSize:         "UI 字号",
	keySettingsSizeXS:           "特小",
	keySettingsSizeSmall:        "小",
	keySettingsSizeMedium:       "中",
	keySettingsSizeLarge:        "大",
	keySettingsSizeXL:           "特大",
	keySettingsCodeFontSize:     "代码字号",
	keySettingsCodeTheme:        "代码主题",
	keySettingsCodeThemeNote:    "不可用：本 shell 的代码跟随界面主题，没有单独的代码主题偏好。",
	keySettingsCodeThemeSame:    "跟随界面主题",
	keySettingsCodePreviewLight: "浅色代码",
	keySettingsCodePreviewDark:  "深色代码",
	keySettingsPreviewGap:       "预览不可用：没有保存代码主题。",
	keySettingsSkinSection:      "我的皮肤",
	keySettingsSkinNote:         "皮肤尚未实现：shell 只绘制一套语义 token，可选 system、light 或 dark。不提供假皮肤。",
	keySettingsSkinCard:         "皮肤",
	keySettingsSkinNew:          "用图片新建",
	keySettingsSkinOpen:         "打开目录",

	keyCapabilityGroup:             "不可用的能力",
	keyCapabilityCreate:            "创建会话",
	keyCapabilityCreateGap:         "Go 客户端已有 helper；shell 还没有创建回调",
	keyCapabilityBranchSwitch:      "切换分支",
	keyCapabilityBranchSwitchGap:   "Go 客户端已有 helper；shell 还没有分支选择回调",
	keyCapabilityBranchEnd:         "结束分支",
	keyCapabilityBranchEndGap:      "Go 客户端已有 helper；shell 还没有分支结束回调",
	keyCapabilityBranchRefresh:     "刷新分支",
	keyCapabilityBranchRefreshGap:  "Go 客户端已有 helper；shell 还没有分支刷新回调",
	keyCapabilityKeys:              "会话键",
	keyCapabilityKeysGap:           "Go 客户端已有 helper；shell 还没有键浏览器回调",
	keyCapabilityApprovals:         "审批",
	keyCapabilityApprovalsGap:      "Go 客户端已有 helper；窗口还没有审批回调",
	keyCapabilityAttachments:       "附件",
	keyCapabilityAttachmentsGap:    "Go 客户端已有 helper；shell 还没有文件选择器，上传结果未知时保持禁用",
	keyCapabilityConnectionRecover: "连接恢复",
	keyCapabilityConnectionGap:     "Go 客户端已有 helper；shell 还没有接线",
	keyCapabilityTurnCancel:        "取消回合",
	keyCapabilityTurnCancelGap:     "Go 客户端已有 helper；shell 还没有接线",
}

// localeAt is the language a select option stands for.
func localeAt(index int) Locale {
	if index == 1 {
		return LocaleZHCN
	}
	return LocaleEN
}

// hasText reports whether the table carries a non-empty entry for key. It is
// what keeps the two tables in step in the tests.
func (s Strings) hasText(key string) bool {
	text, ok := s[key]
	return ok && strings.TrimSpace(text) != ""
}
