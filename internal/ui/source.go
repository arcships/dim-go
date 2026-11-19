package ui

import (
	"context"
	"time"
)

// Source 是视图读取事实的唯一入口（A 领域边界）：视图只认这个接口，
// 不 import internal/app。适配器在 internal/uiwire。
//
// 约定：每帧从 Source 读；view 不缓存事实（只在 Source 返回的快照上读），
// 可丢的视图状态（展开、焦点、草稿、滚动）仍留 view。
type Source interface {
	// Connection 是连接阶段；Phase 为 "connected" 时其余读才有意义。
	Connection() Connection
	// SessionNames 是左栏的会话列表（按目录顺序）。
	SessionNames() []string
	// SessionTitles 是与 SessionNames 同序的显示名：标题优先，其次首条消息预览，
	// 都没有时退到短标识（参考产品不下发标题的会话不会把整串 uuid 摆出来）。
	SessionTitles() []string
	// SessionPlaces 是与 SessionNames 同序的会话归属：工作目录（左栏按它分组）
	// 与最近活动时间（会话数相同的组之间按它排序）。Cwd 为空表示服务端没有给
	// 工作目录，这些会话归左栏内置的「工作区」组。
	SessionPlaces() []SessionPlaceSnapshot
	// Workspaces 是服务端授权的工作区（项目根）清单；空表示没有读到或没有授权。
	// 新建会话的 cwd 只从这里取，不自己拼路径。
	Workspaces() []WorkspaceSnapshot
	// Session 是当前会话的快照；ok 为假表示还没打开会话。
	Session() (SessionSnapshot, bool)
	// Approval 是当前待决审批；ok 为假表示没有。
	Approval() (ApprovalSnapshot, bool)
	// Question 是当前待答提问；ok 为假表示没有。
	Question() (QuestionSnapshot, bool)
	// Account 是页脚的身份与连接态。
	Account() AccountSnapshot
	// Usage 是账号用量的聚合项（与 account.Usage 的聚合字段对齐）。
	Usage() UsageSnapshot
	// Settings 是设置里来自事实的几项。
	Settings() SettingsSnapshot
	// Extensions 是扩展四类的读数（与 extensions.Snapshot[T] 对齐）。
	Extensions() ExtensionsSnapshot
	// Schedules 是定时任务读数（与 extensions.Snapshot[ScheduledTask] 对齐）。
	Schedules() SchedulesSnapshot
	// Terminal 是当前会话终端的状态；Active 为假表示没有终端实例。
	Terminal() TerminalSnapshot
	// Preferences 是客户端本地偏好（主题、动效、时区、侧栏宽度）。
	Preferences() PreferencesSnapshot
	// GlobalShortcut 是全局唤起快捷键在设置里的状态（本端事实，不来自 App Server）。
	GlobalShortcut() GlobalShortcutSnapshot
}

// PanelFacts 是右栏文件页与改动页要读的事实（D7 的只读能力在适配器里的映射）。
//
// 为什么单列成一个接口、而不是加进 Source：这三项读的不是"已就绪的内存快照"，
// 而是本机磁盘与仓库（列目录、扫工作区、读 git 状态），可能耗时。适配器按请求
// 缓存并在后台刷新，视图每帧用同一份请求读快照：读态、截断与错误都写在快照里，
// 既不阻塞帧，也不抛错让页面空白。只实现 Source 的适配器仍可编译。
//
// 事实口径与 docs/domains/D7-workspace.md 一致，未覆盖的部分不在这里补样例：
//
//   - 目录与条目：真实（含隐藏条目、截断说明、git 合并状态）；
//   - git 状态：只有合并后的单列状态字母（M/A/D/?/R/U）——D7 的 git 只读口径
//     不区分暂存列（§4.5），也没有 ± 行数与 diff 文本（§1「不做」）；
//   - 搜索：内容搜索（路径 + 行号 + 列号 + 片段），不是文件名模糊匹配；
//   - git 写动作（stage / unstage / revert / commit / push / 切分支）：既没有
//     协议方法，也不在 D7 的能力里，所以本接口不提供发起入口——不设一个永远
//     失败的写方法。
type PanelFacts interface {
	// FileTree 读一个目录的条目；同一请求每帧读的是同一份快照。
	FileTree(request FileTreeRequest) FileTreeSnapshot
	// FileSearch 读一次工作区内容搜索的结果。
	FileSearch(request FileSearchRequest) FileSearchSnapshot
	// Changes 读一次改动清单（范围见 ChangeScope*）。
	Changes(request ChangesRequest) ChangeSet
	// Preview 读一个文件的前缀内容（文件预览页，D7 的 Preview）。
	Preview(request PreviewRequest) PreviewSnapshot
}

// Intents 是视图发起写的唯一入口：写走既有的 app 方法，不新开协议通道。
//
// 没有对应 app 方法的动作不在这里：见 docs/domains/D14-delivery.md 的「未接动作」。
type Intents interface {
	OpenSession(ctx context.Context, name string) error
	Submit(ctx context.Context, text string) error
	// SubmitWithContent 提交文本 + 内容块（文本 / 图片）。
	SubmitWithContent(ctx context.Context, text string, blocks []ContentBlockSnapshot) error
	Queue(ctx context.Context, text string) error
	Stop(ctx context.Context) error
	RespondApproval(ctx context.Context, decision string) error
	RespondQuestion(ctx context.Context, answer string) error
	// SkipQuestion 是提问卡的「跳过」：按取消提交，不当作拒绝。
	SkipQuestion(ctx context.Context) error
	RetrySubmit(ctx context.Context) error
	CreateSession(ctx context.Context, cwd string) error

	// 会话目录。
	RenameSession(ctx context.Context, sessionID, title string) error
	ArchiveSession(ctx context.Context, sessionID string, archived bool) error
	DeleteSession(ctx context.Context, sessionID string) error
	// ExportSession 把会话导出到本地文件，返回写出的路径。
	ExportSession(ctx context.Context, sessionID string) (string, error)

	// 设置。
	SetPermissionTier(ctx context.Context, tier string) error
	SelectModel(ctx context.Context, providerID, modelID string) error
	WriteCommandRules(ctx context.Context, rules []RuleSnapshot) error

	// 上下文：手动压缩当前会话的上下文（斜杠命令 /compact）。压缩结果以时间线
	// 事实 context/compaction 为准，界面不另存。
	CompactContext(ctx context.Context) error

	// 扩展：重新扫描磁盘上的技能目录（斜杠命令 /reload-skills）。条目与版本由
	// 随后的技能读数带回，界面不缓存第二份。
	RefreshSkills(ctx context.Context) error

	// 分支。
	SelectBranch(ctx context.Context, branchID string) error
	CreateBranch(ctx context.Context, parentID string, turnIndex int, prompt string) error
	EndBranch(ctx context.Context, branchID string) error

	// 附件：写入文本附件，返回服务端回执里的标识与路径。
	WriteAttachment(ctx context.Context, name, text string) (AttachmentRefSnapshot, error)

	// 供应商管理。CreateProvider 带上表单里填的模型条目（模型清单是表单字段，
	// 新建时随 provider.create 一起下发）。
	CreateProvider(ctx context.Context, id, name, baseURL, apiKey string, models []ProviderModelDraft) error
	UpdateProvider(ctx context.Context, id, name, baseURL, apiKey string) error
	ProbeProvider(ctx context.Context, id string) (int, error)
	SetProviderModelEnabled(ctx context.Context, providerID, modelID string, enabled bool) error

	// 账号登录：method 取 browser / device。
	BeginLogin(ctx context.Context, method string) (LoginPromptSnapshot, error)
	WaitLogin(ctx context.Context) error
	CancelLogin(ctx context.Context) error
	Logout(ctx context.Context) error

	// 扩展：kind 取 "plugin" / "skill" / "hook" / "mcp"。
	SetExtensionEnabled(ctx context.Context, kind, id string, enabled bool) error
	UninstallExtension(ctx context.Context, kind, id string) error
	// InstallPlugin 从 sourceKind（local / git）与 sourceRef 安装一个插件；
	// gitRef 只在 Git 来源下有值；id 留空表示按 manifest 自动生成，enable 是
	// 「安装后启用」。
	InstallPlugin(ctx context.Context, sourceKind, sourceRef, gitRef, id string, enable bool) error
	// UpdatePlugin 更新一个已安装插件；具体版本由服务端决定。
	UpdatePlugin(ctx context.Context, pluginID string) error
	// TestMCP 探测一个工具服务器；返回是否可达与原因。
	TestMCP(ctx context.Context, id string) (bool, string, error)
	// ImportMCP 先预览再应用一个导入来源，返回写入的条目数。
	ImportMCP(ctx context.Context, sourceID string) (int, error)

	// 定时任务。
	SetScheduleEnabled(ctx context.Context, jobID string, enabled bool) error
	DeleteSchedule(ctx context.Context, jobID string) error
	// CancelModalityTask 取消一条运行中的模态任务。
	CancelModalityTask(ctx context.Context, taskID string) error
	// StopExecTask 停止一条运行中的后台任务。
	StopExecTask(ctx context.Context, taskID string) error

	// 目标：set 新建/覆盖，edit 改文本，pause/resume，clear；read 走 get。
	ReadGoal(ctx context.Context) (string, error)
	SetGoal(ctx context.Context, objective string) error
	EditGoal(ctx context.Context, objective string) error
	PauseGoal(ctx context.Context) error
	ResumeGoal(ctx context.Context) error
	ClearGoal(ctx context.Context) error

	// 终端：RunShell 返回运行标识，供 StopShell 使用。
	RunShell(ctx context.Context, command string) (string, error)
	StopShell(ctx context.Context, runID string) error
	// 右栏终端的本机 pty（不经服务端）：StartTerminal 在会话的工作目录里启动
	// shell，StopTerminal 停止它，WriteTerminal 把原样字节写回 pty。
	StartTerminal(ctx context.Context) error
	StopTerminal(ctx context.Context) error
	WriteTerminal(ctx context.Context, data string) error
	// Reconnect 重连控制连接（断线后的「重新连接」）。
	Reconnect(ctx context.Context) error
	// SetPreferences 落盘本地偏好。
	SetPreferences(ctx context.Context, prefs PreferencesSnapshot) error
	// SetGlobalShortcut 保存全局唤起快捷键的开关与键位，并按新值重新注册；
	// 注册失败（键位被占用）返回原因。
	SetGlobalShortcut(ctx context.Context, enabled bool, accelerator string) error
}

// ContentBlockSnapshot 是一次提交里的内容块。
type ContentBlockSnapshot struct {
	Kind string // "text" / "image"
	Text string
	// Image 是图片块来源；Kind 为 image 时必填。
	Image *ImageBlockSnapshot
}

// ImageBlockSnapshot 是图片块来源。
type ImageBlockSnapshot struct {
	FileID     string
	SourceType string // base64 / url / file
	MediaType  string
	Data       string
	URL        string
}

// AttachmentRefSnapshot 是 attachment.put 回执里的标识与路径。
type AttachmentRefSnapshot struct {
	ID   string
	Path string
	Name string
}

// PreferencesSnapshot 是本地偏好；它不来自 App Server。
type PreferencesSnapshot struct {
	Theme        string
	MotionOff    bool
	TimeZone     string
	SidebarWidth int
	// Workspaces 是本端登记的附加工作区目录（左栏「添加工作区」）；它不是服务端
	// 授权的项目根，只用于左栏分组。
	Workspaces []string
	// HooksProjectPath 是设置 › Hooks 里选中的项目目录。
	HooksProjectPath string
	// UpdateFeedURL 是设置 › 关于里填写的更新源（latest.json 地址或基址）；
	// 空表示未配置：自动更新关闭。
	UpdateFeedURL string
	// PluginFeedURL 是插件市场源的地址（返回插件清单的 JSON）；空表示未配置：
	// 插件市场整块禁用并写「未配置插件源」。
	PluginFeedURL string
	// NotifyTaskDone / NotifySound / NotifyNeedsResponse 是设置 › 通知的三个开关；
	// 落盘在本地偏好里，宿主的系统通知按它们决定发不发、带不带声音。
	NotifyTaskDone      bool
	NotifySound         bool
	NotifyNeedsResponse bool
	// View 是跨重启要记住的视图状态：左栏收起、视图口径、分组折叠、右栏开合与
	// 标签页、两层筛选。它不是事实，重启后按上次的样子渲染。
	View ViewPreferencesSnapshot
	// Appearance 是「设置 › 外观」里会真正影响渲染的几项（界面字体与字号、代码
	// 字体与字号、代码配色）。它也是本地偏好，不是服务端事实。
	Appearance AppearancePreferences
}

// AppearancePreferences 是「设置 › 外观」里会真正影响渲染的本地偏好：读它改
// 渲染，改它落盘。空串 / 0 表示用默认档。
//
// 语言、皮肤、代码配色里除默认之外的主题本版没有实现，因此不在这里：那几项在
// 屏上是禁用 + 原因（见 unavailable.go），不写一个永远不会生效的值。
type AppearancePreferences struct {
	UIFont   string  // 界面字体族；空 = 系统 UI 字体栈
	CodeFont string  // 代码字体族；空 = 系统等宽
	UIScale  float32 // 界面字号倍率；0 = 平台默认
	CodeSize float32 // 代码字号（DIP）；0 = 13
	// CodeTheme 是代码配色名："" 或 "GitHub" 是内置的一套，其余见 codeThemePalette。
	CodeTheme string
}

// ViewPreferencesSnapshot 是一组可丢但要跨重启记住的视图选择。
type ViewPreferencesSnapshot struct {
	// SidebarCollapsed 是左栏收起态（LB-10 / TB-13）。
	SidebarCollapsed bool
	// SidebarFlat / SidebarByCreated 是左栏视图口径（LB-28–LB-33）。
	SidebarFlat      bool
	SidebarByCreated bool
	// CollapsedGroups 是收起的工作区分组标识（LB-34）；PinnedGroups 是置顶的
	// 工作区分组（LB-44）；WorkspaceNames 是改过的分组显示名（LB-48）。
	CollapsedGroups []string
	PinnedGroups    []string
	WorkspaceNames  map[string]string
	// RightOpen / RightTab / RightTabs 是右栏的开合、当前页与已打开的标签页。
	RightOpen bool
	RightTab  int
	RightTabs []int
	// RightExpanded / RightWidth 是「展开面板」与拖出来的宽度。
	RightExpanded bool
	RightWidth    int
	// FilesFilter 是文件页的筛选词（F-01）；ChangesScope 是改动页的范围（D-01）；
	// ChangesGroupsClosed 是改动页收起的分组（D-33）。
	FilesFilter         string
	ChangesScope        int
	ChangesGroupsClosed []string
}

// GlobalShortcutSnapshot 是全局唤起快捷键在设置里的状态（本端事实）。
type GlobalShortcutSnapshot struct {
	// Enabled 是设置里的开关。
	Enabled bool
	// Accelerator 是规范化后的键位（如 CmdOrCtrl+Shift+Space）。
	Accelerator string
	// Label 是给人看的键位（macOS 上是 ⌘⇧Space）。
	Label string
	// Registered 表示系统已经接受了这次注册。
	Registered bool
	// Reason 是注册失败的原因（例如键位被别的应用占用）；空表示没有失败。
	Reason string
}

// UpdateSnapshot 是设置 › 关于读到的更新事实（宿主本端，不来自 App Server）。
//
// 没有实现 UpdateFacts 的 Source 表示这次运行没有更新能力：界面按「自动更新
// 已禁用」渲染。已实现但 FeedURL 为空时 Configured 为 false，界面显示「未配置
// 更新源」并禁用检查入口，不假装已是最新。
type UpdateSnapshot struct {
	// Configured 表示已经配置更新源。
	Configured bool
	// FeedURL 是规范化后的 latest.json 地址；未配置时为空。
	FeedURL string
	// ConfigError 是配置本身的问题（例如地址不是 http/https）。
	ConfigError string
	// Phase 取 idle / checking / up_to_date / available / downloading /
	// downloaded / preparing / error。
	Phase string
	// CurrentVersion / AvailableVersion 是当前版本与更新源给出的版本。
	CurrentVersion   string
	AvailableVersion string
	// Notes 是更新源给出的发布说明（只展示，不参与判定）。
	Notes string
	// Percent 是下载进度（0–100）。
	Percent int
	// Err 是检查、下载或安装失败的原因；空表示没有失败。
	Err string
	// Installable 表示宿主有安装能力；为假时「重启安装」置灰并给出 Reason。
	Installable   bool
	InstallReason string
}

// UpdateFacts 是可选的更新事实接口。没有实现它的 Source（样例、单屏截图）
// 仍显示「自动更新已禁用」；实现它的适配器从宿主本端读相位。
type UpdateFacts interface {
	Update() UpdateSnapshot
}

// UpdateIntents 是可选的更新动作接口。动作在后台执行，界面每帧从 UpdateFacts
// 读回相位，不在点击回调里等网络。
type UpdateIntents interface {
	// CheckUpdate 检查更新源；失败原因写在 UpdateFacts 的 Err 里。
	CheckUpdate(ctx context.Context) error
	// DownloadUpdate 下载并校验产物。
	DownloadUpdate(ctx context.Context) error
	// InstallUpdate 走宿主安装能力；没有能力时返回原因，不写假回执。
	InstallUpdate(ctx context.Context) error
	// SetUpdateFeed 保存更新源地址（空串表示关闭自动更新）。
	SetUpdateFeed(ctx context.Context, feedURL string) error
}

// PluginMarketEntrySnapshot 是插件市场源返回的一个条目。它只是上游清单的内容，
// 不是 App Server 的事实，也不代表可安装：协议没有市场安装来源。
type PluginMarketEntrySnapshot struct {
	Name        string
	Version     string
	Description string
	Source      string
}

// PluginMarketplaceSnapshot 是插件市场的只读快照。
type PluginMarketplaceSnapshot struct {
	// Configured 表示本地偏好里配置了插件源地址。
	Configured bool
	// FeedURL 是当前生效的源地址；未配置时为空。
	FeedURL string
	// ConfigError 是地址本身的问题（不是 http/https 等）。
	ConfigError string
	// Loading 表示正在拉取。
	Loading bool
	// Entries 是最近一次成功拉取的条目；失败时保留旧条目。
	Entries []PluginMarketEntrySnapshot
	// Err 是拉取失败的原因；空表示没有失败。
	Err string
}

// PluginMarketplaceFacts 是可选的插件市场读数。没有实现它的 Source（样例、
// 单屏截图）不画市场卡片，只写「未配置插件源」。
type PluginMarketplaceFacts interface {
	PluginMarketplace() PluginMarketplaceSnapshot
}

// PluginMarketplaceIntents 是插件市场的动作：保存源地址与重新拉取。
type PluginMarketplaceIntents interface {
	SetPluginFeed(ctx context.Context, feedURL string) error
	RefreshPluginMarketplace(ctx context.Context) error
}

// Connection 是连接阶段与原因（断线要能看见原因，不是白屏）。
type Connection struct {
	Phase  string // "connecting" / "connected" / "disconnected"
	Reason string
}

// WorkspaceSnapshot 是一个授权的工作区（项目根，与 settings.Project 对齐）。
//
// Root 是服务端进程内的项目根编号；Path 是服务端给出的目录，只用于展示与
// 新建会话的 cwd，不用于拼路径。
type WorkspaceSnapshot struct {
	Root int
	Path string
}

// SessionPlaceSnapshot 是一条会话在左栏的归属：工作目录与最近活动时间。
//
// 分组只用这两个事实：Cwd 决定进哪一组（组名取目录末段），UpdatedAt 只在会话数
// 相同时决定组的先后。空 Cwd 表示服务端没有给工作目录（归内置「工作区」）。
type SessionPlaceSnapshot struct {
	Cwd       string
	UpdatedAt time.Time
}

// SessionSnapshot 是首片要用的会话事实（与 app.SessionView 对齐，不多造）。
type SessionSnapshot struct {
	ID      string
	Title   string
	Running bool
	Rows    []RowSnapshot
	Pending int
	Goal    string
	// GoalStatus 是 active/paused/complete/blocked；空表示没有状态。
	GoalStatus string
	// Cwd 是目录里的会话工作目录（绝对路径）；空表示服务端没有给。
	// 右栏文件页与改动页按它决定读哪个工作区。
	Cwd string
	// Branch 是当前查看的分支；Branches 是分支清单（名字即标识）。
	Branch   string
	Branches []BranchSnapshot
	// TurnCount 是已加载轮次数，用于从当前分支切出时选切点。
	TurnCount int
	// HasOlder 表示还有更早的轮次没装载（投影层的分页窗口）。为真时 Rows 只是
	// 已加载的那一段：导出与搜索都只能覆盖这一段，界面必须写明，不把它说成全量。
	HasOlder bool
	// LastTurn 是最后一个已结束的轮：Ended 为假表示还没有结束的轮，Outcome 是
	// 服务端给的结果词。宿主通知按它区分「完成」与「失败」。
	LastTurnEnded   bool
	LastTurnIndex   int
	LastTurnOutcome string
	// Tasks 是未结束的后台任务。
	Tasks    []TaskSnapshot
	OpenedAt time.Time
	// Queued 是已提交、服务端还没消费的输入（排队消息面板 IN-35–IN-42）：
	// 标识、投递方式、正文与「已应用未消费」。只读事实，撤回与编辑都不在这里。
	Queued []QueuedMessageSnapshot
	// ShellRun 是最近一次会话内命令运行；nil 表示没有运行记录（IN-45–IN-52）。
	ShellRun *ShellRunSnapshot
	// Compaction 是最近一次上下文压缩的事实（TL-68 / TL-69）；nil 表示没有。
	Compaction *CompactionSnapshot
}

// QueuedMessageSnapshot 是一条排队输入（与 projection.PendingInput 对齐）。
//
// Mode 取 queue / steer / append：steer 表示要插进当前轮次，界面上写「转向」；
// Applied 为真表示已经应用到上下文但还没被消费，因而不可撤回。
type QueuedMessageSnapshot struct {
	ID      string
	Mode    string
	Text    string
	Applied bool
}

// ShellRunSnapshot 是一次会话内命令运行的读数（shell.run 回执的字段）。
//
// Status 取 running / completed / failed / stopped。Output 是回执里的输出原文，
// Truncated 表示服务端截断过输出；ExitCode 只在 HasExitCode 时有意义。
// Recorded 表示 record 模式下结果已经写入时间线；Error 是服务端给的失败原文。
type ShellRunSnapshot struct {
	RunID       string
	Command     string
	Mode        string
	Status      string
	Output      string
	Truncated   bool
	ExitCode    int
	HasExitCode bool
	Error       string
	Reason      string
	Recorded    bool
}

// CompactionSnapshot 是最近一次上下文压缩（与 projection.Compaction 对齐）。
//
// Status 取服务端给的词：running / completed / failed / skipped / cancelled。
// Summary 是压缩后的摘要正文；取消或失败时可能为空。
type CompactionSnapshot struct {
	Status   string
	Summary  string
	Strategy string
}

// TaskSnapshot 是一条后台任务。
type TaskSnapshot struct {
	ID          string
	Description string
	Stopping    bool
}

// BranchSnapshot 是一个分支。
type BranchSnapshot struct {
	ID     string
	Name   string
	Status string
	// Ended 为真表示分支已结束（只读）。
	Ended bool
}

// 时间线行的种类。视图按种类找对应的块；认不出的种类仍按正文文本画，
// 所以新增种类不会让内容消失（纯文本回退）。
const (
	// RowUser 是用户的一次输入。
	RowUser = "user"
	// RowAssistant 是助手正文的一段（Markdown）。
	RowAssistant = "assistant"
	// RowThinking 是一段思考，正文是思考原文。
	RowThinking = "thinking"
	// RowTool 是一次工具调用；Title 是动作，Args 是参数摘要，Text 是结果原文。
	RowTool = "tool"
	// RowToolGroup 是连续工具调用的折叠组，Children 是其中的工具行。
	RowToolGroup = "toolGroup"
	// RowError 是一轮以失败收口的事实；Text 是服务端给的结果原文（可能为空）。
	RowError = "error"
	// RowRetry 是一次自动重试的倒计时；剩余时间取 RetryIn（不用 Duration）。
	//
	// 协议 v3 没有重试事实：投影与 adapter 都不产这种行，界面在 RetryIn 为 0 时
	// 也不画（拿不到剩余时间就不显示，绝不把已耗时当成"还剩"）。
	RowRetry = "retry"
	// RowArtifact 是本轮产物，Children 是产物行（Text 是名字）。
	//
	// 工具与协议都不声明产物：种类与画法先留着，投影不产生这种行。
	RowArtifact = "artifact"
	// RowFileChanges 是本轮改动，Children 是改动文件行（Text 是路径）。
	RowFileChanges = "fileChanges"
	// RowFileChange 是改动块里的一个文件（只作 Children 出现）。
	RowFileChange = "fileChange"
	// RowCompression 是压缩/窗口截断的分隔行，Text 是服务端的摘要原文。
	RowCompression = "compression"
)

// 行状态的界面词：前三个是视图按它们选样式的那三种，其余两词是投影原样给出
// 的服务端状态（视图按结束状态画）。
const (
	RowStatusRunning   = "running"
	RowStatusDone      = "done"
	RowStatusFailed    = "failed"
	RowStatusPending   = "pending"
	RowStatusCancelled = "cancelled"
)

// RowSnapshot 是时间线的一行。字段只装事实：种类、正文、短标签、参数摘要、
// 状态、时长、差异计数与分组子行；呈现（字号、图标、颜色）由视图决定。
type RowSnapshot struct {
	// Kind 取上面 Row* 之一。
	Kind string
	// BranchID 是这一行所属的分支；空表示快照没有给。
	//
	// 导出按范围取内容时用它（整条会话 / 仅当前分支）；只读映射，不改投影。
	BranchID string
	// TurnIndex 是这一行所属的轮次序号（分支内从 1 开始）；0 表示还没有轮次事实。
	//
	// 导出按轮取内容、以及「以某一轮结束处为切点建分支」都用它：没有轮次序号时
	// 不拿别的轮次顶替（见 unavailable.go）。
	TurnIndex int
	// Text 是这一行的正文：用户消息、助手正文、思考原文、工具结果、压缩摘要。
	Text string
	// Title 是行首的短标签（工具动作）；空表示该种类没有短标签。
	Title string
	// Args 是工具行的参数摘要（命令、路径、查询），不是整段输入。
	Args string
	// Status 取 RowStatus* 之一；空表示这一行没有状态事实。
	Status string
	// Duration 是这一行已经用掉的时间（思考、工具）；0 表示没有。
	//
	// 它是"已耗时"，不是"还剩"：倒计时一律取 RetryIn，两个口径不混用。
	Duration time.Duration
	// RetryIn 是自动重试行还剩多久（RowRetry 专用）；0 表示拿不到剩余时间。
	//
	// 剩余时间是另一条事实，不能由 Duration 推出：RetryIn 为 0 时界面不画这一
	// 行，也不写一个推算出来的秒数。
	RetryIn time.Duration
	// DiffAdd / DiffRemove 是改动行数；0 表示没有差异事实（不是"没有改动"）。
	DiffAdd    int
	DiffRemove int
	// Expanded 是这一行默认是否展开；视图仍可在本地覆盖（可丢的界面状态）。
	//
	// 目前没有一条事实给出默认开合，所以投影一律给 false，视图按收起画。
	Expanded bool
	// Children 是分组里的行：工具组里的工具行、改动块里的文件行、产物行。
	Children []RowSnapshot
}

// ApprovalSnapshot 是一条待决审批。
//
// 卡片的输入区按 Tool + Input 挑渲染器（命令 / 文件 / 检索 / 网页 / 技能 /
// 子代理 / MCP / 图片 / 待办 / 通用）；Tool 与 Input 都是服务端原文，界面不补
// 字段。Command 是要执行的命令原文的短路径（样例态与只有命令的场景用）。
type ApprovalSnapshot struct {
	ID       string
	Command  string
	Impact   string
	LongTerm bool
	// Tool 是工具名原文（如 exec / read / grep / web_fetch）。
	Tool string
	// Input 是工具入参 JSON 原文；空表示服务端没有给入参。
	Input string
	// SourceKind 是审批来源类别（tool / host）。
	SourceKind string
	// Pending 表示决定已发出、还没有结算：按钮保持禁用直到请求结算。
	Pending bool
	// Member 是成员审批的上下文：请求工具为 member_approval_allow 时由连接层填。
	// protocol v3 没有成员审批明细通道（session.collaboration.approval.get），
	// 所以状态恒为 unavailable：能画出来的只有审批 ID 与"只能拒绝"。
	Member *ApprovalMemberSnapshot
	// Sandbox / SubagentModel / Handoff 是另外三种审批形状。protocol v3 的
	// approval.request 没有这些字段，连接层不会填；nil 表示没有该事实，界面不画
	// 那张卡，也不拿工具卡顶替。
	Sandbox       *ApprovalSandboxSnapshot
	SubagentModel *ApprovalSubagentModelSnapshot
	Handoff       *ApprovalHandoffSnapshot
}

// 成员审批区块的状态，与参考产品 InteractionMemberApproval.state 同义。
const (
	// ApprovalMemberLoading 是成员动作还在加载。
	ApprovalMemberLoading = "loading"
	// ApprovalMemberReady 是成员动作已就绪，可以审查。
	ApprovalMemberReady = "ready"
	// ApprovalMemberError 是成员动作读取失败，可重试。
	ApprovalMemberError = "error"
	// ApprovalMemberUnavailable 是本端与 App Server 都给不出原始动作。
	ApprovalMemberUnavailable = "unavailable"
)

// ApprovalMemberSnapshot 是成员审批区块的事实。
type ApprovalMemberSnapshot struct {
	State          string
	AgentID        string
	BranchID       string
	LeaderBranchID string
	ApprovalID     string
	Reason         string
	Error          string
	CanRetry       bool
}

// ApprovalSandboxSnapshot 是沙箱越权卡的事实：被拦的是域名还是路径、哪类访问。
type ApprovalSandboxSnapshot struct {
	// ResourceType 取 domain / path；空按 domain 画（参考产品的回退）。
	ResourceType string
	// Access 取 read / write / execute；domain 时为空。
	Access string
	// Resource 是被拦的域名或路径原文。
	Resource string
	// Tool 是发起这次访问的工具名，用于说明句。
	Tool string
}

// ApprovalSubagentModelSnapshot 是子代理换模型卡的事实。
type ApprovalSubagentModelSnapshot struct {
	TaskTitle     string
	ModelName     string
	ProviderName  string
	FallbackModel string
	// Pricier 表示这个模型比当前模型贵；false 时不画价格标记。
	Pricier bool
}

// ApprovalHandoffSnapshot 是自动审批交回人工的提示块事实。
type ApprovalHandoffSnapshot struct {
	// Kind 取 reviewer_uncertain / reviewer_failed。
	Kind   string
	Reason string
}

// QuestionSnapshot 是一条待答提问。一个请求带一批问题，界面逐题展示、一次提交；
// Questions 为空时按 Text / Options 当成单题（旧读数兼容）。
type QuestionSnapshot struct {
	ID      string
	Text    string
	Options []string
	// Questions 是整批问题，按服务端给的顺序。
	Questions []QuestionItemSnapshot
	// Deadline 是作答截止时间；零值表示请求没有给。protocol v3 的待答清单目前
	// 不带这个事实，界面只在快照有时画倒计时。
	Deadline time.Time
}

// QuestionItemSnapshot 是批次里的一道题：题目、说明、候选项与两个开关。
type QuestionItemSnapshot struct {
	ID          string
	Title       string
	Description string
	Text        string
	Options     []string
	Multiple    bool
	AllowCustom bool
}

// AccountSnapshot 是页脚身份与设置里的账号读数（与 account.Snapshot 对齐）。
type AccountSnapshot struct {
	Initial   string
	Connected bool
	// Name/Email/Plan/Status 来自服务端最近一次确认的账号事实；空表示服务端没有给。
	Name   string
	Email  string
	Plan   string
	Status string
	// LoginPhase/Reason 是当前登录尝试；LoginUserCode/LoginURL 是设备码或浏览器提示。
	LoginPhase    string
	LoginReason   string
	LoginUserCode string
	LoginURL      string
	// LoginMethods 是 provider 声明的登录方式（browser / device）。
	LoginMethods []string
}

// LoginPromptSnapshot 是一次登录发起的提示。
type LoginPromptSnapshot struct {
	ProviderID string
	Method     string
	Attempt    string
	URL        string
	UserCode   string
	VerifyURL  string
}

// UsageSnapshot 是账号用量的聚合项；Available 为假时数值没有意义。
type UsageSnapshot struct {
	Available bool
	Used      float64
	Limit     float64
	Unit      string
	Reason    string
	ResetsAt  time.Time
	HasResets bool
}

// SettingsSnapshot 是设置里来自事实的几项（与 settings 领域视图对齐）。
type SettingsSnapshot struct {
	Provider   string
	ProviderID string
	Providers  []ProviderOption
	Model      string
	ModelID    string
	Models     []ModelOption
	// ModelRevision 是当前选择的版本前置条件；空表示服务端没有给。
	ModelRevision string
	// Tier 是当前会话的权限档位；TierKnown 为假表示还没读到。
	Tier      string
	TierKnown bool

	// Rules 是命令规则的只读视图；RulesKnown 为假表示还没读到。
	Rules      []RuleSnapshot
	RulesKnown bool
	RulesError string

	Storage      StorageSnapshot
	StorageKnown bool
	StorageError string

	Runtimes       []RuntimeSnapshot
	RuntimesKnown  bool
	RuntimesReason string

	Modality ModalitySnapshot
	// ModalityTasks 是模态任务清单；TasksError 非空表示读取失败。
	ModalityTasks []ModalityTaskSnapshot
	TasksError    string
	// ProviderRecords 是 provider 管理清单；ProviderError 非空表示读取失败。
	ProviderRecords []ProviderRecordSnapshot
	ProviderError   string
}

// ProviderRecordSnapshot 是一个 provider 及其模型。
type ProviderRecordSnapshot struct {
	ID         string
	Name       string
	BaseURL    string
	DriverKind string
	AuthType   string
	Enabled    bool
	Version    int
	Models     []ProviderModelSnapshot
}

// ProviderModelSnapshot 是 provider 下的一个模型。
type ProviderModelSnapshot struct {
	ID      string
	Name    string
	Enabled bool
	Status  string
}

// ProviderModelDraft 是新建供应商表单里的一个模型条目：表单只收模型标识，
// 展示名取标识本身、状态取 active（表单没有第二个字段可给）。
type ProviderModelDraft struct {
	ID string
}

// ModalityTaskSnapshot 是一条模态任务。
type ModalityTaskSnapshot struct {
	ID        string
	Category  string
	Operation string
	State     string
	Progress  string
}

// ProviderOption 是一个 provider（标识 + 展示名）。
type ProviderOption struct {
	ID   string
	Name string
}

// ModelOption 是一个模型（标识 + 展示名 + 所属 provider）。
type ModelOption struct {
	ID         string
	ProviderID string
	Name       string
}

// RuleSnapshot 是一条命令规则。
type RuleSnapshot struct {
	Command  string
	Decision string
	Match    string
}

// StorageSnapshot 是存储占用读数，单位是字节。
type StorageSnapshot struct {
	DatabaseBytes    int64
	WALBytes         int64
	ReclaimableBytes int64
}

// RuntimeSnapshot 是一个运行时组件。
type RuntimeSnapshot struct {
	Name    string
	Version string
	Status  string
	Reason  string
}

// ModalitySnapshot 是模态能力里界面要用的几项。
type ModalitySnapshot struct {
	// Known 为假表示还没读到服务端的能力声明：这时不画「未声明」的横幅，
	// 也不列任何能力。
	Known         bool
	Supported     bool
	TaskSupported bool
	TaskKnown     bool
	Reason        string
}

// ExtensionsSnapshot 是扩展四类的读数。
type ExtensionsSnapshot struct {
	Plugins ExtListSnapshot
	Skills  ExtListSnapshot
	MCP     ExtListSnapshot
	Hooks   ExtListSnapshot
	// MCPImportSources 是服务端扫描到的导入来源；ImportError 是非空时的原因。
	MCPImportSources []MCPImportSourceSnapshot
	ImportError      string
}

// MCPImportSourceSnapshot 是一个可导入的客户端配置来源。
type MCPImportSourceSnapshot struct {
	ID      string
	Client  string
	Scope   string
	Path    string
	State   string
	Entries int
}

// ExtListSnapshot 是一类扩展的读数；ReadError 非空表示最近一次读取失败。
type ExtListSnapshot struct {
	Items             []ExtItemSnapshot
	ReadError         string
	CapabilityMissing bool
}

// ExtItemSnapshot 是一行扩展（与 extensions.Plugin/Skill/Hook/ToolServer 对齐）。
type ExtItemSnapshot struct {
	// ID 是服务端标识；Name 是展示名，可能不同。
	ID      string
	Name    string
	Version string
	Enabled bool
	// Description 是服务端给的说明（技能的描述）；空表示没有。
	Description string
	// Status/Error 是服务端给出的原始状态与错误；空表示没有。
	Status string
	Error  string
	// UpdateAvailable / LatestVersion 由服务端声明（插件清单）：本端不比较版本号。
	// 服务端没有给出时不冒充「已是最新」。
	UpdateAvailable bool
	LatestVersion   string
	// Source 是插件的来源（local / git / bundled）；Use 是服务端声明的使用方式
	// （例如 builtin）；Locked 表示服务端锁定，用户不能启停或移除。
	Source   string
	Use      string
	Locked   bool
	Optional bool
	// Event/Command/Source 是钩子的原始事实，用于单项启停时重建文档；
	// Matcher/Type/Timeout/Async/OnFailure 是钩子目录的其余扁平字段
	// （S-hooks-20 的行）。
	Event     string
	Command   string
	Matcher   string
	Type      string
	Timeout   int
	Async     bool
	OnFailure string
	// Transport/Args/URL/Tools/Connected 是工具服务器（MCP）的事实；
	// 其余四类为空（Command 两者共用：钩子命令 / stdio 启动命令）。
	Transport string
	Args      []string
	URL       string
	Tools     int
	Connected bool
}

// SchedulesSnapshot 是定时任务读数。
type SchedulesSnapshot struct {
	Items             []ScheduleSnapshot
	ReadError         string
	CapabilityMissing bool
}

// ScheduleSnapshot 是一条定时任务（与 extensions.ScheduledTask 对齐）。
type ScheduleSnapshot struct {
	ID         string
	Name       string
	Schedule   string
	Enabled    bool
	Running    bool
	NextRunAt  time.Time
	LastRunAt  time.Time
	LastStatus string
}

// TerminalSnapshot 是当前会话终端的状态。
//
// Lines 是适配器从 pty 流里读到的输出（按换行切开，末尾还没换行的半行也给出
// ——提示符通常不带换行）；连着 Source 但还没有启动终端时 Active 为假。视图只
// 读这份快照，不自己读 pty。
type TerminalSnapshot struct {
	Active   bool
	Running  bool
	Exited   bool
	ExitCode int
	Lines    []string
}

// ---- 右栏文件页与改动页的读数 ----

// 面板读的三态与两种空态（视图按它选骨架 / 内容 / 错误行 / 空态）。
const (
	// PanelReadIdle 表示还没读（例如没有可读的工作目录，屏幕尚无请求）。
	PanelReadIdle = "idle"
	// PanelReadLoading 表示首次读取还没回来（视图画骨架或「正在搜索…」）。
	PanelReadLoading = "loading"
	// PanelReadReady 表示最近一次读取成功（含"读到了空"）。
	PanelReadReady = "ready"
	// PanelReadFailed 表示最近一次读取失败，原因在 Error。
	PanelReadFailed = "failed"
)

// FileTreeRequest 是一次目录读取的请求。
//
// Cwd 与 Dir 来自会话事实（目录里的 cwd）与视图状态（展开到哪一层），
// Offset / Limit 是视图的分页位置：同一请求在同一帧里读到的总是同一份快照。
type FileTreeRequest struct {
	// Cwd 是会话工作目录（绝对路径）；空表示按工作区根读。
	Cwd string
	// Dir 是相对 Cwd 的目录路径，"" 表示根；分隔符用 '/'。
	Dir string
	// Offset 是本页的起点条数（「加载更多」传已加载条数）；0 表示第一页。
	Offset int
	// Limit 是本页条数；0 用适配器的默认页大小。
	Limit int
	// Refresh 为真时跳过缓存立刻重读（刷新 / 重试按钮用）。
	Refresh bool
}

// FileTreeSnapshot 是一次目录读取的结果。
type FileTreeSnapshot struct {
	// Cwd / Dir / Offset / Limit 回显请求，视图据此判断快照是否属于当前请求。
	Cwd    string
	Dir    string
	Offset int
	Limit  int
	// Entries 是本页条目：目录在前、文件在后（按名称字节序分组）。
	Entries []FileEntrySnapshot
	// Total 是磁盘上该目录的实际条目数（含被上限截掉的部分）。
	Total int
	// HasMore 表示还有没读到的条目（Offset+len(Entries) < Total）。
	HasMore bool
	// Truncated 表示本次读取被适配器的条目上限截断（说明在 Note）。
	Truncated bool
	// Note 是读取方给出的说明（截断、git 不可用）；空表示没有。
	Note string
	// GitKnown 表示这次读取拿到了 git 状态。
	GitKnown bool
	// Phase 取 PanelRead* 之一。
	Phase string
	// Error 是失败原因原文；空表示没有失败。
	Error string
	// Denied 为真表示失败原因是权限不足（EPERM / EACCES）。
	Denied bool
	// ReadAt 是这次快照的读取时间；零值表示还没读过。
	ReadAt time.Time
}

// FileEntrySnapshot 是一个目录条目。
type FileEntrySnapshot struct {
	Name string
	// Dir 表示目录（不跟随符号链接）。
	Dir bool
	// Symlink 表示符号链接；视图按文件画，不进入。
	Symlink bool
	// Size 是字节数，ModTime 是修改时间（不跟随链接）。
	Size    int64
	ModTime time.Time
	// Change 是 git 合并状态字母：M 修改 / A 新增 / D 删除 / ? 未跟踪 /
	// R 重命名 / U 冲突；空表示没有改动事实（或这次没读到 git）。
	//
	// 目录的字母由后代聚合而来（D7 §4.2）。没有暂存列：D7 只给合并状态。
	Change string
}

// FileSearchRequest 是一次工作区内容搜索的请求。
type FileSearchRequest struct {
	// Cwd 是搜索起点（绝对路径）；空表示按工作区根。
	Cwd string
	// Query 是关键词；空表示不搜索（快照为 PanelReadIdle）。
	Query string
	// Limit 是命中条数上限；0 用适配器的默认值。
	Limit int
	// Refresh 为真时跳过缓存立刻重扫（刷新 / 重试按钮用）。
	Refresh bool
}

// FileSearchSnapshot 是一次内容搜索的结果。
//
// 命中是内容命中（路径 + 行号 + 列号 + 片段），不是文件名模糊匹配：
// 同一条路可能有多处命中，视图要先按路径归并再画行。
type FileSearchSnapshot struct {
	Query string
	Limit int
	// Hits 是命中，按 D7 的扫描顺序（工作区路径字节序）。
	Hits []FileHitSnapshot
	// Truncated 表示结果不完整，原因在 StopReason。
	Truncated bool
	// StopReason 取 max-matches / max-files / max-scan-bytes；空表示扫完。
	StopReason string
	// FilesScanned / BytesScanned / Skipped 是扫描规模读数。
	FilesScanned int
	BytesScanned int64
	Skipped      int
	// Phase 取 PanelRead* 之一。
	Phase string
	// Error 是失败原因原文；空表示没有失败。
	Error string
	// Denied 为真表示失败原因是权限不足。
	Denied bool
	ReadAt time.Time
}

// FileHitSnapshot 是一处搜索命中（与 workspace.SearchMatch 对齐）。
type FileHitSnapshot struct {
	// Path 是相对工作区根的路径（'/' 分隔）。
	Path   string
	Line   int
	Column int
	// Snippet 是该行去掉首尾空白后的片段，超长时以「…」结尾。
	Snippet string
}

// 改动范围的取值（与改动页的范围栏同序）。
const (
	// ChangeScopeUncommitted 是工作区的未提交改动。
	ChangeScopeUncommitted = "uncommitted"
	// ChangeScopeBranch 是当前分支与基线的差异。
	ChangeScopeBranch = "branch"
	// ChangeScopeLastRun 是最近一次 agent run 触达的文件。
	ChangeScopeLastRun = "lastRun"
)

// ChangesRequest 是一次改动读取的请求。
type ChangesRequest struct {
	// Cwd 是工作区目录（绝对路径）；空表示按工作区根。
	Cwd string
	// Scope 取 ChangeScope* 之一；空按未提交。
	Scope string
	// RunID 是 ChangeScopeLastRun 要读的轮次；空表示取会话已收口的最后一轮。
	RunID string
	// Refresh 为真时跳过缓存立刻重读（刷新 / 重试按钮用）。
	Refresh bool
}

// ChangeSet 是一次改动读取的结果。
//
// 已接到的真实读数：仓库与否、当前分支、是否分离头指针、是否有未提交改动、
// 改动条目数（D7 的 git 只读口径，§4.5）；以及 Last Run 范围的路径清单
// （工具声明级，coverage=tool_reports，§4.7）。
//
// 还没接的读数（不留空位编造）：按暂存态分组的文件行、每行 ± 行数、diff 文本。
// 它们需要一个"按暂存列 + numstat + diff"的读取能力，D7 明确不做（§1「不做」
// 里的 git 写与 diff）。Unavailable 非空时视图画「改动不可用」而不是空列表。
type ChangeSet struct {
	// Cwd / Scope / RunID 回显请求。
	Cwd   string
	Scope string
	RunID string

	// Repo 表示工作区在 git 工作树里；其余 git 字段只在 Repo 为真时有意义。
	Repo     bool
	Branch   string
	Detached bool
	Dirty    bool
	// ChangedFileCount 是 porcelain 列出的改动条目数。
	ChangedFileCount int

	// Files 是 ChangeScopeLastRun 的路径清单（其余范围为空）。
	Files []ChangeFileSnapshot
	// FilesTruncated 表示路径清单被上限截断；Note 给出说明。
	FilesTruncated bool
	Note           string

	// Unavailable 非空表示这个层次的事实拿不到（原因原文），视图据此画不可用态。
	Unavailable string
	// Restricted 表示权限不足（EPERM / EACCES），写动作与部分读不可用。
	Restricted bool

	// Phase 取 PanelRead* 之一。
	Phase  string
	Error  string
	ReadAt time.Time
}

// ChangeFileSnapshot 是一条改动文件。
type ChangeFileSnapshot struct {
	// Path 在工作区内是相对路径（'/' 分隔），在工作区外是绝对路径。
	Path string
	// Status 是 M/A/D/?/R/U 之一；空表示事实里没有状态。
	Status string
	// Exists 表示磁盘上还能 Stat 到（声明删除的文件为 false）。
	Exists bool
	// Added / Removed 是行数；没有行数事实时为 0 且 StatKnown 为假。
	Added     int
	Removed   int
	StatKnown bool
}

// ---- 文件预览（右栏「打开文件」页） ----

// 预览内容的类别（与 workspace.PreviewKind 同一口径）。
const (
	// PreviewEmpty 表示空文件。
	PreviewEmpty = "empty"
	// PreviewText 表示已按识别出的编码解码为文本。
	PreviewText = "text"
	// PreviewBinary 表示二进制或未识别编码，不返回内容。
	PreviewBinary = "binary"
)

// PreviewRequest 是一次文件预览的读取请求。
//
// Cwd 是会话工作目录（绝对路径）；Path 是相对 Cwd 的文件路径（'/' 分隔）。
// 与目录读一样按请求缓存：同一请求每帧读同一份快照，Refresh 跳过缓存重读。
type PreviewRequest struct {
	Cwd  string
	Path string
	// MaxBytes 是读取上限；0 用适配器的默认值。
	MaxBytes int
	Refresh  bool
}

// PreviewSnapshot 是一次文件预览的结果（与 workspace.Preview 对齐）。
//
// 内容类别、截断与说明都来自读的一方：视图不猜「读不到」与「空文件」的区别。
type PreviewSnapshot struct {
	// Cwd / Path 回显请求，视图据此判断快照是否属于当前请求。
	Cwd  string
	Path string
	// Kind 取 PreviewEmpty / PreviewText / PreviewBinary 之一。
	Kind     string
	Encoding string
	Size     int64
	// ReadBytes 是从磁盘实际读到的字节数。
	ReadBytes int
	// Content 只在 Kind 为 text 时有值。
	Content string
	// Truncated 表示文件大于读取上限，Content 只是前缀。
	Truncated bool
	// Note 说明截断、二进制或未识别编码的原因；空表示没有需要说明的情况。
	Note string
	// Phase 取 PanelRead* 之一。
	Phase string
	// Error 是失败原因原文；空表示没有失败。
	Error string
	// Denied 为真表示失败原因是权限不足（EPERM / EACCES）。
	Denied bool
	ReadAt time.Time
}
