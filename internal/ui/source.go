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
}

// Intents 是视图发起写的唯一入口：写走既有的 app 方法，不新开协议通道。
//
// 没有对应 app 方法的动作不在这里：见 docs/domains/D14-delivery.md 的「未接动作」。
type Intents interface {
	OpenSession(ctx context.Context, name string) error
	Submit(ctx context.Context, text string) error
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

	// 附件：文本先做，返回写入的附件名。
	WriteAttachment(ctx context.Context, name, text string) (string, error)

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

	// 终端：RunShell 返回运行标识，供 StopShell 使用。
	RunShell(ctx context.Context, command string) (string, error)
	StopShell(ctx context.Context, runID string) error
	// Reconnect 重连控制连接（断线后的「重新连接」）。
	Reconnect(ctx context.Context) error
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

// RowSnapshot 是时间线的一行（用户 / 助手 / 过程组）。
type RowSnapshot struct {
	Kind string // "user" / "assistant" / "group"
	Text string
	Ops  int
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
