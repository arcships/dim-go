// Package settings 实现管理面的设置与运行环境领域：设置来源与项目作用域的读取与
// 写入、模型与 provider 目录、存储读数、运行组件状态与安装、模态能力状态。
//
// 范围内：
//   - 读取服务端声明的设置来源（含作用域与只读/只写边界）与已授权项目根；
//   - 按用户作用域或项目作用域读取设置文档，写入时带版本前置条件；
//   - 读取 provider 与模型目录、当前选择，并按账号可用模型清单过滤可选范围；
//   - 读取存储占用与可回收字节数（只读，不触发删除）；
//   - 读取运行组件状态，安装只转达给服务端；
//   - 读取模态能力状态与不可用原因。
//
// 不包含：界面、连接实现与 wire 定义、写登记内部、插件与技能、账号本身、
// 存储清理、模态任务提交与实时传输。所有写入都经写登记（KindManagementWrite）
// 登记一次，本包不自行重试；缺能力、缺读取接口时返回可识别的能力缺失错误，
// 不做本地兜底，也不返回零值冒充事实。
package settings

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// 服务端声明的可选能力名。缺项按能力缺失处理，不猜。
const (
	// FeatureSettingsSources 是设置来源清单。
	FeatureSettingsSources = "settings.sources"
	// FeatureSettingsProjects 是已授权项目根清单。
	FeatureSettingsProjects = "settings.projects"
	// FeatureSettingsRead 是设置文档读取。
	FeatureSettingsRead = "settings.read"
	// FeatureSettingsUpdate 是设置文档写入。
	FeatureSettingsUpdate = "settings.update"
	// FeatureModelProviders 是 provider 与模型目录。
	FeatureModelProviders = "models.providers.list"
	// FeatureModelSelection 是模型选择的写入。
	FeatureModelSelection = "provider.select"
	// FeatureProviderList 是 provider 概要清单。
	FeatureProviderList = "provider.list"
	// FeatureProviderManagement 是 provider 管理详情。
	FeatureProviderManagement = "provider.management"
	// FeatureProviderCreate 是新建 provider。
	FeatureProviderCreate = "provider.create"
	// FeatureProviderUpdate 是更新 provider。
	FeatureProviderUpdate = "provider.update"
	// FeatureProviderProbe 是显式探测 provider 的模型列表。
	FeatureProviderProbe = "provider.probe"
	// FeatureProviderModelsSetEnabled 是设置 provider 的启用模型集合。
	FeatureProviderModelsSetEnabled = "provider.models.set_enabled"
	// FeatureProviderModelsEdit 是编辑 provider 的模型配置。
	FeatureProviderModelsEdit = "provider.models.edit"
	// FeatureProviderModelsReset 是重置 provider 的模型配置。
	FeatureProviderModelsReset = "provider.models.reset"
	// FeatureProviderPreferencesUpdate 是写入模型偏好。
	FeatureProviderPreferencesUpdate = "provider.preferences.update"
	// FeatureCollaborationPolicy 是协作模型策略读取。
	FeatureCollaborationPolicy = "collaboration.model_policy"
	// FeatureCollaborationSetPolicy 是协作模型策略写入。
	FeatureCollaborationSetPolicy = "collaboration.set_model_policy"
	// FeatureAgentList 是 agent 档案清单。
	FeatureAgentList = "agent.list"
	// FeatureSessionAgentGet 是会话当前采用的 agent 档案读取。
	FeatureSessionAgentGet = "session.agent.get"
	// FeatureSessionAgentUpdate 是会话 agent 档案更新。
	FeatureSessionAgentUpdate = "session.agent.update"
	// FeatureStorageReport 是存储读数。
	FeatureStorageReport = "storage.report"
	// FeatureRuntimeList 是运行组件清单。
	FeatureRuntimeList = "runtime.list"
	// FeatureRuntimeInstall 是运行组件安装。
	FeatureRuntimeInstall = "runtime.install"
	// FeatureModalityCapabilities 是模态能力清单。
	FeatureModalityCapabilities = "modality.capabilities"

	// capabilityAccountModels 不是服务端方法名，而是账号领域提供的可用模型视图。
	capabilityAccountModels = "account.models"
)

// 错误值。调用方按 errors.Is 判断，不匹配错误文本。
var (
	// ErrCapabilityMissing 是所有「服务端或集成层没有这项能力」的共同分类。
	// 具体缺失项见 *CapabilityError。
	ErrCapabilityMissing = errors.New("能力缺失")
	// ErrInvalidScope 表示作用域取值不合法。
	ErrInvalidScope = errors.New("作用域不合法")
	// ErrInvalidSource 表示设置来源名为空。
	ErrInvalidSource = errors.New("设置来源不合法")
	// ErrSourcesNotLoaded 表示尚未读取设置来源清单。
	ErrSourcesNotLoaded = errors.New("设置来源尚未读取")
	// ErrProjectsNotLoaded 表示尚未读取已授权项目根。
	ErrProjectsNotLoaded = errors.New("设置项目尚未读取")
	// ErrScopeNotAllowed 表示该来源没有声明此作用域。
	ErrScopeNotAllowed = errors.New("该设置来源不支持此作用域")
	// ErrSourceWriteOnly 表示该来源只接受写入，不提供可读文档。
	ErrSourceWriteOnly = errors.New("该设置来源只支持写入")
	// ErrProjectNotAuthorized 表示项目根不在服务端授权清单中。
	ErrProjectNotAuthorized = errors.New("项目根未授权")
	// ErrUserOnlySetting 表示把只能用户作用域写入的键写进了项目作用域。
	ErrUserOnlySetting = errors.New("该设置只能在用户作用域写入")
	// ErrRevisionMissing 表示写入缺少版本前置条件。
	ErrRevisionMissing = errors.New("缺少设置版本前置条件")
	// ErrRevisionConflict 表示版本前置条件不成立。冲突时不覆盖、不重试。
	ErrRevisionConflict = errors.New("设置版本冲突")
	// ErrSelectionUnavailable 表示当前账号状态下没有可写的模型选择。
	ErrSelectionUnavailable = errors.New("模型选择不可用")
	// ErrSelectionFailed 表示模型选择写入明确失败。
	ErrSelectionFailed = errors.New("模型选择写入失败")
	// ErrInvalidProviderRequest 表示 provider 或模型管理请求结构不合法。
	ErrInvalidProviderRequest = errors.New("供应商请求不合法")
	// ErrProviderReadFailed 表示 provider 清单或详情读取失败。
	ErrProviderReadFailed = errors.New("供应商读取失败")
	// ErrProviderConflict 表示 provider 版本前置条件不成立。
	ErrProviderConflict = errors.New("供应商版本冲突")
	// ErrProviderWriteFailed 表示 provider 写入明确失败。
	ErrProviderWriteFailed = errors.New("供应商写入失败")
	// ErrModelWriteFailed 表示 provider 模型写入明确失败。
	ErrModelWriteFailed = errors.New("模型配置写入失败")
	// ErrProbeFailed 表示显式探测失败。
	ErrProbeFailed = errors.New("供应商探测失败")
	// ErrProviderNotEditable 表示该 provider 没有可写的版本或动作。
	ErrProviderNotEditable = errors.New("供应商不可编辑")
	// ErrMalformedProvider 表示 provider 事实结构不合法。
	ErrMalformedProvider = errors.New("供应商事实不合法")
	// ErrInvalidCollaborationRequest 表示协作模型策略请求不合法。
	ErrInvalidCollaborationRequest = errors.New("协作策略请求不合法")
	// ErrCollaborationReadFailed 表示协作模型策略读取失败。
	ErrCollaborationReadFailed = errors.New("协作策略读取失败")
	// ErrCollaborationConflict 表示协作模型策略版本前置条件不成立。
	ErrCollaborationConflict = errors.New("协作策略版本冲突")
	// ErrCollaborationWriteFailed 表示协作模型策略写入明确失败。
	ErrCollaborationWriteFailed = errors.New("协作策略写入失败")
	// ErrMalformedCollaboration 表示协作模型策略事实不合法。
	ErrMalformedCollaboration = errors.New("协作策略事实不合法")
	// ErrInvalidAgentRequest 表示 agent 档案请求不合法。
	ErrInvalidAgentRequest = errors.New("agent 档案请求不合法")
	// ErrAgentReadFailed 表示 agent 档案读取失败。
	ErrAgentReadFailed = errors.New("agent 档案读取失败")
	// ErrAgentConflict 表示当前采用的 agent 与期望不一致。
	ErrAgentConflict = errors.New("agent 档案冲突")
	// ErrAgentWriteFailed 表示 agent 档案写入明确失败。
	ErrAgentWriteFailed = errors.New("agent 档案写入失败")
	// ErrMalformedAgent 表示 agent 档案事实不合法。
	ErrMalformedAgent = errors.New("agent 档案事实不合法")
	// ErrInvalidModalityRequest 表示模态任务或实时传输的请求结构不合法。
	ErrInvalidModalityRequest = errors.New("模态请求不合法")
	// ErrTaskSubmitFailed 表示模态任务提交明确失败。
	ErrTaskSubmitFailed = errors.New("模态任务提交失败")
	// ErrTaskCancelFailed 表示模态任务取消明确失败。
	ErrTaskCancelFailed = errors.New("模态任务取消失败")
	// ErrTaskReadFailed 表示模态任务状态读取失败。
	ErrTaskReadFailed = errors.New("模态任务读取失败")
	// ErrTaskListFailed 表示模态任务清单读取失败。
	ErrTaskListFailed = errors.New("模态任务清单读取失败")
	// ErrTaskResultFailed 表示模态任务结果读取失败。
	ErrTaskResultFailed = errors.New("模态任务结果读取失败")
	// ErrRealtimeOpenFailed 表示实时传输打开明确失败。
	ErrRealtimeOpenFailed = errors.New("实时传输打开失败")
	// ErrRealtimeSendFailed 表示实时帧发送失败。
	ErrRealtimeSendFailed = errors.New("实时帧发送失败")
	// ErrRealtimeCloseFailed 表示实时传输关闭明确失败。
	ErrRealtimeCloseFailed = errors.New("实时传输关闭失败")
	// ErrRealtimeNotOpen 表示本进程没有该连接，或连接已经结束。
	ErrRealtimeNotOpen = errors.New("实时连接未打开")
	// ErrWriteUnavailable 表示没有可用的写登记。
	ErrWriteUnavailable = errors.New("设置写入不可用")
	// ErrUpdateFailed 表示设置写入明确失败。
	ErrUpdateFailed = errors.New("设置写入失败")
	// ErrMalformedWriteResult 表示写入成功回执不是可采用的设置文档。
	ErrMalformedWriteResult = errors.New("设置写入回执不合法")
	// ErrMalformedSources 表示设置来源事实不合法。
	ErrMalformedSources = errors.New("设置来源事实不合法")
	// ErrMalformedProjects 表示项目根事实不合法。
	ErrMalformedProjects = errors.New("项目根事实不合法")
	// ErrMalformedDocument 表示设置文档不合法。
	ErrMalformedDocument = errors.New("设置文档不合法")
	// ErrMalformedModels 表示 provider 与模型目录不合法。
	ErrMalformedModels = errors.New("模型目录不合法")
	// ErrMalformedStorage 表示存储读数不合法。
	ErrMalformedStorage = errors.New("存储读数不合法")
	// ErrMalformedRuntimes 表示运行组件清单不合法。
	ErrMalformedRuntimes = errors.New("运行组件清单不合法")
	// ErrMalformedModality 表示模态能力事实不合法。
	ErrMalformedModality = errors.New("模态能力事实不合法")
	// ErrSourcesRead 表示设置来源读取失败。
	ErrSourcesRead = errors.New("设置来源读取失败")
	// ErrProjectsRead 表示项目根读取失败。
	ErrProjectsRead = errors.New("项目根读取失败")
	// ErrSettingsRead 表示设置文档读取失败。
	ErrSettingsRead = errors.New("设置读取失败")
	// ErrModelsRead 表示模型目录读取失败。
	ErrModelsRead = errors.New("模型目录读取失败")
	// ErrStorageRead 表示存储读数读取失败。
	ErrStorageRead = errors.New("存储读数读取失败")
	// ErrRuntimesRead 表示运行组件读取失败。
	ErrRuntimesRead = errors.New("运行组件读取失败")
	// ErrModalityRead 表示模态能力读取失败。
	ErrModalityRead = errors.New("模态能力读取失败")
	// ErrInvalidRuntimeName 表示运行组件名为空。
	ErrInvalidRuntimeName = errors.New("运行组件名不合法")
	// ErrInstallFailed 表示运行组件安装明确失败。
	ErrInstallFailed = errors.New("运行组件安装失败")
	// ErrRuntimeEntryMissing 表示安装成功后清单里没有该组件。
	ErrRuntimeEntryMissing = errors.New("安装后运行组件未出现在清单")
)

// CapabilityError 说明缺的是哪一项能力。它同时满足 errors.Is(err, ErrCapabilityMissing)。
type CapabilityError struct {
	// Capability 是缺失的能力名（服务端方法名或集成层接口名）。
	Capability string
	// Reason 是缺失原因；不解释成可用状态。
	Reason string
}

func (e *CapabilityError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Reason == "" {
		return fmt.Sprintf("%s: %s", ErrCapabilityMissing, e.Capability)
	}
	return fmt.Sprintf("%s: %s: %s", ErrCapabilityMissing, e.Capability, e.Reason)
}

// Is 让所有缺失项都归到 ErrCapabilityMissing 这一分类。
func (e *CapabilityError) Is(target error) bool { return target == ErrCapabilityMissing }

func missing(capability, reason string) error {
	return &CapabilityError{Capability: capability, Reason: reason}
}

// SelectionUnavailableError 说明为什么不能写入模型选择。它同时满足
// errors.Is(err, ErrSelectionUnavailable)。
type SelectionUnavailableError struct {
	ProviderID string
	ModelID    string
	// Reason 是账号状态或模型可选性给出的原因。
	Reason string
}

func (e *SelectionUnavailableError) Error() string {
	if e == nil {
		return "<nil>"
	}
	if e.Reason == "" {
		return fmt.Sprintf("%s: %s/%s", ErrSelectionUnavailable, e.ProviderID, e.ModelID)
	}
	return fmt.Sprintf("%s: %s/%s: %s", ErrSelectionUnavailable, e.ProviderID, e.ModelID, e.Reason)
}

// Is 让所有不可写原因归到 ErrSelectionUnavailable 这一分类。
func (e *SelectionUnavailableError) Is(target error) bool {
	return target == ErrSelectionUnavailable
}

// Features 是服务端声明的可选能力。连接领域的 *daemon.Connection 直接满足它。
type Features interface {
	HasFeature(name string) bool
}

// Writer 是写登记的对外入口；*write.Manager 直接满足它。
type Writer interface {
	Register(write.Intent) (write.IntentView, error)
	Wait(context.Context, string) (write.IntentView, error)
}

// Config 是管理面设置领域的依赖。各项能力可以独立缺失；缺失时对应方法返回
// 能力缺失错误，而不是空值。
type Config struct {
	// Features 是服务端声明的可选能力；nil 时不做声明检查，只按调用结果分类。
	Features Features
	// Sources 读取设置来源清单。
	Sources SourcesReader
	// Settings 读取设置文档。
	Settings SettingsReader
	// Projects 读取已授权项目根。
	Projects ProjectsReader
	// Models 读取 provider 与模型目录。
	Models ModelCatalogReader
	// Account 提供按账号状态过滤后的可用模型清单。
	Account AccountAvailability
	// Storage 读取存储占用。
	Storage StorageReader
	// Runtimes 读取运行组件清单。
	Runtimes RuntimeReader
	// Modality 读取模态能力。
	Modality ModalityReader
	// ModalitySession 是会话流上的模态任务读取与实时帧发送面。
	ModalitySession ModalitySession
	// Providers 是 provider 清单、详情与显式探测的读取面。
	Providers ProviderAdmin
	// Collaboration 是协作模型策略的读取面。
	Collaboration CollaborationReader
	// Agents 是 agent 清单与会话当前 agent 的读取面。
	Agents AgentAdmin
	// Writer 是写登记；设置写入与组件安装都经它登记一次。
	Writer Writer
}

// Manager 持有管理面读数的最近一次确认值与设置文档缓存。所有导出方法可并发调用。
// 它不落盘：进程重启后从零开始，不补发任何写入。
type Manager struct {
	cfg Config

	mu sync.Mutex
	// writeMu 串行化管理写入，避免两个并发写入把各自的回执按不同顺序写回缓存。
	writeMu sync.Mutex
	// writeSeq 生成写意图的幂等键序号；同一进程内不复用。
	writeSeq uint64

	sources   Sources
	sourcesOK bool

	projects   []Project
	projectsOK bool

	documents map[string]Document

	models   ModelSnapshot
	modelsOK bool

	storage   StorageReport
	storageOK bool

	runtimes   []RuntimeEntry
	runtimesOK bool

	modality   ModalityCapabilities
	modalityOK bool

	tasks    map[string]ModalityTask
	realtime map[string]*realtimeState

	providers      ProviderList
	providersOK    bool
	providerDetail map[string]ProviderDetail

	collaboration   CollaborationPolicySnapshot
	collaborationOK bool

	agents   []AgentProfile
	agentsOK bool
}

// New 创建管理面设置领域。各读取接口与写登记都可以在之后由集成层补齐；
// 缺失时对应方法返回能力缺失错误。
func New(cfg Config) (*Manager, error) {
	return &Manager{
		cfg:            cfg,
		documents:      make(map[string]Document),
		tasks:          make(map[string]ModalityTask),
		realtime:       make(map[string]*realtimeState),
		providerDetail: make(map[string]ProviderDetail),
	}, nil
}

// requireFeature 在服务端声明了可选能力清单时检查该项；清单缺失时交给调用结果
// 分类，不假装能力存在或不存在。
func (m *Manager) requireFeature(name string) error {
	if m.cfg.Features == nil {
		return nil
	}
	if m.cfg.Features.HasFeature(name) {
		return nil
	}
	return missing(name, "服务端未声明该能力")
}

// classifyReadError 把「方法不存在 / 能力不可用」的服务端错误归到能力缺失；
// 其他错误保持为读取失败。
func classifyReadError(capability string, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, ErrCapabilityMissing) {
		return err
	}
	var rpcErr *protocol.RPCError
	if errors.As(err, &rpcErr) {
		switch rpcErr.AppError() {
		case "unknown_method", "capability_unavailable", "storage_unavailable":
			return missing(capability, rpcErr.Error())
		}
		if rpcErr.Code == -32601 {
			return missing(capability, rpcErr.Error())
		}
	}
	return nil
}

// readFailure 把一次读取错误包装成稳定分类：能力缺失优先，其次是对应读取失败。
func readFailure(capability string, base error, err error) error {
	if capErr := classifyReadError(capability, err); capErr != nil {
		return capErr
	}
	return fmt.Errorf("%w: %v", base, err)
}

// normalizeObject 校验一段 JSON 是对象，并返回压缩后的副本与键集合。
// null、数组、标量与空内容都不算对象。
func normalizeObject(raw json.RawMessage) (json.RawMessage, map[string]json.RawMessage, error) {
	trimmed := bytes.TrimSpace(raw)
	if len(trimmed) == 0 {
		return nil, nil, fmt.Errorf("%w: 内容为空", ErrMalformedDocument)
	}
	var object map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &object); err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrMalformedDocument, err)
	}
	if object == nil {
		return nil, nil, fmt.Errorf("%w: 不是对象", ErrMalformedDocument)
	}
	var compacted bytes.Buffer
	if err := json.Compact(&compacted, trimmed); err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrMalformedDocument, err)
	}
	return append(json.RawMessage(nil), compacted.Bytes()...), object, nil
}

// validJSON 判断非空内容是否是合法 JSON；空内容视为未提供。
func validJSON(raw json.RawMessage) bool {
	return len(bytes.TrimSpace(raw)) == 0 || json.Valid(raw)
}

func cloneRaw(raw json.RawMessage) json.RawMessage {
	if raw == nil {
		return nil
	}
	return append(json.RawMessage(nil), raw...)
}

func trimNonEmpty(value string) (string, bool) {
	trimmed := strings.TrimSpace(value)
	return trimmed, trimmed != ""
}
