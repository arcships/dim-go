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

	// 分支。
	SelectBranch(ctx context.Context, branchID string) error
	CreateBranch(ctx context.Context, parentID string, turnIndex int, prompt string) error
	EndBranch(ctx context.Context, branchID string) error

	// 附件：写入文本附件，返回服务端回执里的标识与路径。
	WriteAttachment(ctx context.Context, name, text string) (AttachmentRefSnapshot, error)

	// 供应商管理。
	CreateProvider(ctx context.Context, id, name, baseURL, apiKey string) error
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
	// Reconnect 重连控制连接（断线后的「重新连接」）。
	Reconnect(ctx context.Context) error
	// SetPreferences 落盘本地偏好。
	SetPreferences(ctx context.Context, prefs PreferencesSnapshot) error
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
}

// Connection 是连接阶段与原因（断线要能看见原因，不是白屏）。
type Connection struct {
	Phase  string // "connecting" / "connected" / "disconnected"
	Reason string
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
	// Branch 是当前查看的分支；Branches 是分支清单（名字即标识）。
	Branch   string
	Branches []BranchSnapshot
	// TurnCount 是已加载轮次数，用于从当前分支切出时选切点。
	TurnCount int
	// Tasks 是未结束的后台任务。
	Tasks    []TaskSnapshot
	OpenedAt time.Time
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
	// RowRetry 是一次自动重试的倒计时；Duration 是剩余时间。
	//
	// 协议目前没有重试事实：种类与画法先留着，投影不产生这种行。
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
	// Text 是这一行的正文：用户消息、助手正文、思考原文、工具结果、压缩摘要。
	Text string
	// Title 是行首的短标签（工具动作）；空表示该种类没有短标签。
	Title string
	// Args 是工具行的参数摘要（命令、路径、查询），不是整段输入。
	Args string
	// Status 取 RowStatus* 之一；空表示这一行没有状态事实。
	Status string
	// Duration 是这一行已经用掉的时间（思考、工具、重试倒计时）；0 表示没有。
	Duration time.Duration
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
type ApprovalSnapshot struct {
	ID       string
	Command  string
	Impact   string
	LongTerm bool
}

// QuestionSnapshot 是一条待答提问。
type QuestionSnapshot struct {
	ID      string
	Text    string
	Options []string
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
	// Status/Error 是服务端给出的原始状态与错误；空表示没有。
	Status string
	Error  string
	// Event/Command/Source 是钩子的原始事实，用于单项启停时重建文档。
	Event   string
	Command string
	Source  string
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
// 终端输出由 pty 流式读取，本批只接状态，不缓存输出行；Lines 为空。
type TerminalSnapshot struct {
	Active   bool
	Running  bool
	Exited   bool
	ExitCode int
	Lines    []string
}
