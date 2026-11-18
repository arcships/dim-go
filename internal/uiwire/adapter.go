package uiwire

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/arcships/dimagent-mygo/internal/account"
	"github.com/arcships/dimagent-mygo/internal/app"
	"github.com/arcships/dimagent-mygo/internal/catalog"
	"github.com/arcships/dimagent-mygo/internal/extensions"
	"github.com/arcships/dimagent-mygo/internal/interaction"
	"github.com/arcships/dimagent-mygo/internal/settings"
	"github.com/arcships/dimagent-mygo/internal/ui"
)

// App 把 internal/app 的集成层包成界面要的 Source 与 Intents。
//
// 边界（A 领域）：ui 不 import app；这个包同时 import 两边，只做映射与转调，
// 不新增协议通道、不攒第二份事实（每次读都问 app）。
type App struct {
	A *app.App
	// OnChange 在订阅收到回执或事件后调用，视图据此重绘；可为 nil。
	OnChange func()
	// SessionID 是当前打开的会话 id；空表示还没打开。
	SessionID string

	mu        sync.RWMutex
	phase     ui.Connection
	sessionMu sync.RWMutex
	// importMu 保护最近一次读取的 MCP 导入来源（界面每帧读，不能每帧发请求）。
	importMu      sync.Mutex
	importSources []ui.MCPImportSourceSnapshot
	importErr     string
	// rulesMu 保护最近一次读取的命令规则（界面每帧读，不能每帧发请求）。
	rulesMu       sync.Mutex
	rules         []ui.RuleSnapshot
	rulesRevision string
	rulesKnown    bool
	rulesErr      string
	// accountMu 保护 provider 登录方式与最近一次登录提示。
	accountMu    sync.Mutex
	loginMethods []string
	loginPrompt  ui.LoginPromptSnapshot
	// providerMu 保护 provider 管理清单。
	providerMu      sync.Mutex
	providerRecords []ui.ProviderRecordSnapshot
	providerErr     string
	// tasksMu 保护最近一次读取的模态任务清单。
	tasksMu  sync.Mutex
	tasks    []ui.ModalityTaskSnapshot
	tasksErr string
	// changesMu 保护按轮次缓存的工具声明改动（界面每帧读，不能每帧发请求）。
	changesMu    sync.Mutex
	changesByRun map[string][]ui.RowSnapshot
	changesAsked map[string]bool
	// changeSlots 限制同时在读的改动数量（见 refreshRunChanges）。
	changeSlots chan struct{}
	// panel 是右栏文件页与改动页的读数缓存（列目录 / 内容搜索 / git 状态），
	// 见 panel.go：界面每帧读快照，后台按缓存期重读。
	panel panelCache
}

// New 包住一个已创建的集成层。
func New(a *app.App) *App {
	return &App{
		A:            a,
		phase:        ui.Connection{Phase: "connecting"},
		changesByRun: make(map[string][]ui.RowSnapshot),
		changesAsked: make(map[string]bool),
		changeSlots:  make(chan struct{}, changeReadSlots),
		panel:        newPanelCache(),
	}
}

// currentSession 返回当前打开的会话标识；读写由 sessionMu 保护。
func (w *App) currentSession() string {
	w.sessionMu.RLock()
	defer w.sessionMu.RUnlock()
	return w.SessionID
}

// setSession 设置当前会话标识。
func (w *App) setSession(id string) {
	w.sessionMu.Lock()
	w.SessionID = id
	w.sessionMu.Unlock()
}

// Connect 走连接与订阅，并把阶段写进 Connection（失败时带上原因）。
func (w *App) Connect(ctx context.Context) error {
	w.setPhase("connecting", "")
	// 集成层要先装载本地状态（归档、暂存、工作区），否则拒绝新意图。
	if err := w.A.LoadLocalState(ctx); err != nil {
		w.setPhase("disconnected", err.Error())
		return err
	}
	if err := w.A.Connect(ctx); err != nil {
		w.setPhase("disconnected", err.Error())
		return err
	}
	// 先订阅（拿到首轮历史），再打开写入闸门：宿主启动里 allow_writes 在 history 之后。
	if err := w.A.Subscribe(ctx); err != nil {
		w.setPhase("disconnected", err.Error())
		return err
	}
	// 首轮历史阶段：没有打开的会话时它直接标记完成，之后才能打开写入闸门。
	if err := w.A.FirstHistory(ctx); err != nil {
		w.setPhase("disconnected", err.Error())
		return err
	}
	if err := w.A.AllowWrites(ctx); err != nil {
		w.setPhase("disconnected", err.Error())
		return err
	}
	// 目录是独立的读：没有它，"有没有会话"都无从判断。
	if err := w.refreshCatalog(ctx); err != nil {
		w.setPhase("disconnected", err.Error())
		return err
	}
	if names := w.SessionNames(); len(names) > 0 && w.currentSession() == "" {
		w.setSession(names[0])
	}
	// 设置、扩展、定时任务与终端的读数：只在连接成功后读一次，失败留空由界面
	// 显示"未读取"，不用样例顶上。
	w.loadFacts(ctx)
	w.setPhase("connected", "")
	w.watchConnection()
	w.changed()
	return nil
}

// watchConnection 在控制连接关闭后把阶段改成 disconnected，并带上关闭原因。
// 只观察一次：重连不在本批范围内（见 D14 未接清单）。
func (w *App) watchConnection() {
	done := w.A.ConnectionClosed()
	if done == nil {
		return
	}
	go func() {
		<-done
		reason := "连接已关闭"
		if err := w.A.ConnectionError(); err != nil {
			reason = err.Error()
		}
		w.setPhase("disconnected", reason)
		w.changed()
	}()
}

// loadFacts 读取设置、扩展与定时任务的初始读数。
//
// 每一项都是可失败的可选能力：失败不影响连接阶段，只让对应的快照保持"未读取"
// 或带上服务端原文。这里不发写请求，也不探测服务端不存在的方法。
func (w *App) loadFacts(ctx context.Context) {
	if account := w.A.Account(); account != nil {
		_, _ = account.Refresh(ctx)
		_, _ = account.RefreshUsage(ctx)
		_, _ = account.RefreshModels(ctx)
	}
	if settings := w.A.Settings(); settings != nil {
		_, _ = settings.RefreshModels(ctx)
		_, _ = settings.RefreshStorage(ctx)
		_, _ = settings.RefreshRuntimes(ctx)
		_, _ = settings.RefreshModality(ctx)
	}
	if extensions := w.A.Extensions(); extensions != nil {
		_, _ = extensions.ListPlugins(ctx)
		_, _ = extensions.ListSkills(ctx)
		_, _ = extensions.ListHooks(ctx)
		_, _ = extensions.ListToolServers(ctx)
		_, _ = extensions.ListScheduledTasks(ctx)
	}
	w.refreshImportSources(ctx)
	w.refreshRules(ctx)
	w.refreshModalityTasks(ctx)
	w.refreshAccountProviders(ctx)
	w.refreshProviderRecords(ctx)
}

// refreshProviderRecords 读取 provider 清单与每个 provider 的详情。
func (w *App) refreshProviderRecords(ctx context.Context) {
	manager := w.A.Settings()
	if manager == nil {
		return
	}
	list, err := manager.RefreshProviders(ctx)
	w.providerMu.Lock()
	defer w.providerMu.Unlock()
	if err != nil {
		w.providerErr = err.Error()
		return
	}
	w.providerErr = ""
	w.providerRecords = w.providerRecords[:0]
	for _, summary := range list.Providers {
		record := ui.ProviderRecordSnapshot{
			ID: summary.ID, Name: firstNonEmpty(summary.DisplayName, summary.ID),
			BaseURL: summary.BaseURL, DriverKind: summary.DriverKind,
			AuthType: summary.AuthType, Enabled: summary.Enabled, Version: summary.Version,
		}
		if detail, readErr := manager.ReadProvider(ctx, summary.ID); readErr == nil {
			for _, model := range detail.Models {
				record.Models = append(record.Models, ui.ProviderModelSnapshot{
					ID: model.ID, Name: firstNonEmpty(model.DisplayName, model.ID),
					Enabled: model.Enabled, Status: model.Status,
				})
			}
		}
		w.providerRecords = append(w.providerRecords, record)
	}
}

// refreshAccountProviders 读一次 provider 登录方式，供账号分节显示登录入口。
func (w *App) refreshAccountProviders(ctx context.Context) {
	accountDomain := w.A.Account()
	if accountDomain == nil {
		return
	}
	providers, err := accountDomain.Providers(ctx)
	if err != nil {
		return
	}
	methods := make([]string, 0)
	seen := make(map[string]bool)
	for _, provider := range providers {
		for _, method := range provider.LoginMethods {
			if !seen[string(method)] {
				seen[string(method)] = true
				methods = append(methods, string(method))
			}
		}
	}
	w.accountMu.Lock()
	w.loginMethods = methods
	w.accountMu.Unlock()
}

// refreshModalityTasks 读一页模态任务并留在适配器里；界面每帧只读这份快照。
func (w *App) refreshModalityTasks(ctx context.Context) {
	manager := w.A.Settings()
	if manager == nil || w.currentSession() == "" {
		return
	}
	page, err := manager.ListTasks(ctx, w.currentSession(), settings.ModalityTaskQuery{})
	w.tasksMu.Lock()
	defer w.tasksMu.Unlock()
	if err != nil {
		w.tasksErr = err.Error()
		return
	}
	w.tasksErr = ""
	w.tasks = w.tasks[:0]
	for _, task := range page.Tasks {
		progress := ""
		if task.Progress != nil {
			progress = task.Progress.Message
		}
		w.tasks = append(w.tasks, ui.ModalityTaskSnapshot{
			ID: task.ID, Category: task.Category, Operation: task.Operation,
			State: string(task.State), Progress: progress,
		})
	}
}

// refreshRules 读一次用户级命令规则并留在适配器里；界面每帧只读这份快照。
func (w *App) refreshRules(ctx context.Context) {
	set, err := w.A.ReadCommandRules(ctx)
	w.rulesMu.Lock()
	defer w.rulesMu.Unlock()
	if err != nil {
		w.rulesErr = err.Error()
		return
	}
	w.rulesErr = ""
	w.rulesKnown = true
	w.rulesRevision = set.Revision
	w.rules = w.rules[:0]
	for _, rule := range set.Rules {
		w.rules = append(w.rules, ui.RuleSnapshot{
			Command:  rule.Command,
			Decision: string(rule.Decision),
			Match:    string(rule.Match),
		})
	}
}

// refreshImportSources 读一次 MCP 导入来源并留在适配器里；界面每帧只读这份快照。
func (w *App) refreshImportSources(ctx context.Context) {
	service := w.A.Extensions()
	if service == nil {
		return
	}
	sources, err := service.ListMCPImportSources(ctx, w.sessionWorkspace(w.currentSession()))
	w.importMu.Lock()
	defer w.importMu.Unlock()
	if err != nil {
		w.importErr = err.Error()
		return
	}
	w.importErr = ""
	w.importSources = w.importSources[:0]
	for _, source := range sources {
		entries := 0
		if source.Entries != nil {
			entries = *source.Entries
		}
		w.importSources = append(w.importSources, ui.MCPImportSourceSnapshot{
			ID: source.ID, Client: source.Client, Scope: source.Scope,
			Path: source.Path, State: string(source.State), Entries: entries,
		})
	}
}

func (w *App) setPhase(phase, reason string) {
	w.mu.Lock()
	w.phase = ui.Connection{Phase: phase, Reason: reason}
	w.mu.Unlock()
}

func (w *App) changed() {
	if w.OnChange != nil {
		w.OnChange()
	}
}

// ---- ui.Source ----

func (w *App) Connection() ui.Connection {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.phase
}

// SessionNames 是目录里的会话（不是"已打开"的会话）：读 Catalog 的快照。
func (w *App) SessionNames() []string {
	c := w.A.Catalog()
	if c == nil {
		return nil
	}
	entries := c.Snapshot().Entries
	out := make([]string, 0, len(entries))
	for _, e := range entries {
		out = append(out, e.ID)
	}
	return out
}

// refreshCatalog 刷新目录（读之后才知道有没有会话）。
func (w *App) refreshCatalog(ctx context.Context) error {
	c := w.A.Catalog()
	if c == nil {
		return nil
	}
	_, err := c.Refresh(ctx, catalog.Query{})
	w.changed()
	return err
}

func (w *App) Session() (ui.SessionSnapshot, bool) {
	if w.currentSession() == "" {
		return ui.SessionSnapshot{}, false
	}
	view, ok := w.A.View(w.currentSession())
	if !ok {
		return ui.SessionSnapshot{}, false
	}
	// 已收口的轮次排一次改动读取（同一轮只排一次，结果到了再重绘）。
	w.requestRunChanges(view)
	return w.snapshot(view), true
}

// snapshot 把 app.SessionView 映射成界面要的快照（只取首片用到的字段）。
func (w *App) snapshot(v app.SessionView) ui.SessionSnapshot {
	out := ui.SessionSnapshot{
		ID:        v.SessionID,
		Title:     v.SessionID,
		Running:   v.Running,
		Pending:   v.PendingCounts.Approvals + v.PendingCounts.Questions,
		Branch:    v.BranchID,
		TurnCount: len(v.Turns),
	}
	for _, pending := range v.Pending.Tasks {
		out.Tasks = append(out.Tasks, ui.TaskSnapshot{
			ID: pending.TaskID, Description: pending.Description, Stopping: pending.Stopping,
		})
	}
	for _, branch := range v.Branches {
		out.Branches = append(out.Branches, ui.BranchSnapshot{
			ID:     branch.BranchID,
			Name:   branch.BranchID,
			Status: string(branch.Status),
			Ended:  string(branch.Status) == "ended",
		})
	}
	// 标题是目录事实；目录没有给标题时才退回会话标识。
	if catalog := w.A.Catalog(); catalog != nil {
		if entry, ok := catalog.Entry(v.SessionID); ok && entry.Title != "" {
			out.Title = entry.Title
		}
	}
	if v.Goal != nil {
		out.Goal = v.Goal.Objective
		out.GoalStatus = string(v.Goal.Status)
	}
	out.Rows = w.rows(v)
	return out
}

func (w *App) Approval() (ui.ApprovalSnapshot, bool) {
	if w.currentSession() == "" {
		return ui.ApprovalSnapshot{}, false
	}
	view, ok := w.A.View(w.currentSession())
	if !ok || !view.HasCurrent || view.Current.Kind != interaction.PendingApproval {
		return ui.ApprovalSnapshot{}, false
	}
	item := view.Current
	return ui.ApprovalSnapshot{
		ID:       item.RequestID,
		Command:  item.Action,
		LongTerm: item.LongTermAvailable,
	}, true
}

func (w *App) Question() (ui.QuestionSnapshot, bool) {
	if w.currentSession() == "" {
		return ui.QuestionSnapshot{}, false
	}
	view, ok := w.A.View(w.currentSession())
	if !ok || !view.HasCurrent || view.Current.Kind != interaction.PendingQuestion ||
		len(view.Current.Questions) == 0 {
		return ui.QuestionSnapshot{}, false
	}
	q := view.Current.Questions[0]
	return ui.QuestionSnapshot{ID: view.Current.RequestID, Text: q.Text, Options: q.Options}, true
}

func (w *App) Account() ui.AccountSnapshot {
	out := ui.AccountSnapshot{Connected: w.A.Connected()}
	account := w.A.Account()
	if account == nil {
		return out
	}
	snapshot := account.Snapshot()
	out.Name = firstNonEmpty(snapshot.Identity.DisplayName, snapshot.Identity.PreferredUsername, snapshot.Identity.Email)
	out.Email = snapshot.Identity.Email
	out.Plan = snapshot.Identity.Plan
	out.Status = string(snapshot.Status)
	out.Initial = avatarInitial(out.Name, snapshot.Identity.Email, snapshot.Identity.AccountID)
	state := account.LoginState()
	out.LoginPhase = string(state.Phase)
	out.LoginReason = state.Reason
	w.accountMu.Lock()
	out.LoginMethods = append([]string(nil), w.loginMethods...)
	out.LoginUserCode = w.loginPrompt.UserCode
	out.LoginURL = w.loginPrompt.URL
	w.accountMu.Unlock()
	return out
}

// Usage 是账号用量的聚合项（逐项求和不在本端做）。
func (w *App) Usage() ui.UsageSnapshot {
	account := w.A.Account()
	if account == nil {
		return ui.UsageSnapshot{}
	}
	usage := account.Usage()
	out := ui.UsageSnapshot{
		Available: usage.Aggregate.Available,
		Used:      usage.Aggregate.Used,
		Limit:     usage.Aggregate.Limit,
		Unit:      usage.Aggregate.Unit,
		Reason:    usage.Aggregate.Reason,
	}
	if usage.Credits != nil && !usage.Credits.ResetsAt.IsZero() {
		out.ResetsAt = usage.Credits.ResetsAt
		out.HasResets = true
	}
	return out
}

// Settings 把设置领域最近确认的读数映射成界面快照。
func (w *App) Settings() ui.SettingsSnapshot {
	out := ui.SettingsSnapshot{}
	if w.currentSession() != "" {
		if tier, ok := w.A.PermissionTier(w.currentSession()); ok {
			out.Tier = string(tier)
			out.TierKnown = true
		}
	}
	settings := w.A.Settings()
	if settings == nil {
		return out
	}
	if models, ok := settings.Models(); ok {
		for _, provider := range models.Providers {
			out.Providers = append(out.Providers, ui.ProviderOption{
				ID: provider.ID, Name: firstNonEmpty(provider.Name, provider.ID),
			})
			for _, model := range provider.Models {
				out.Models = append(out.Models, ui.ModelOption{
					ID: model.ID, ProviderID: provider.ID,
					Name: firstNonEmpty(model.Name, model.ID),
				})
			}
		}
		if selection := models.Selection; selection != nil {
			out.ProviderID = selection.ProviderID
			out.ModelID = selection.ModelID
			out.Provider = providerLabel(models, selection.ProviderID)
			out.Model = modelLabel(models, selection.ProviderID, selection.ModelID)
			out.ModelRevision = selection.Revision
		}
	}
	w.rulesMu.Lock()
	out.Rules = append([]ui.RuleSnapshot(nil), w.rules...)
	out.RulesKnown = w.rulesKnown
	out.RulesError = w.rulesErr
	w.rulesMu.Unlock()
	w.tasksMu.Lock()
	out.ModalityTasks = append([]ui.ModalityTaskSnapshot(nil), w.tasks...)
	out.TasksError = w.tasksErr
	w.tasksMu.Unlock()
	w.providerMu.Lock()
	out.ProviderRecords = append([]ui.ProviderRecordSnapshot(nil), w.providerRecords...)
	out.ProviderError = w.providerErr
	w.providerMu.Unlock()
	if report, ok := settings.Storage(); ok {
		out.Storage = ui.StorageSnapshot{
			DatabaseBytes:    report.DatabaseBytes,
			WALBytes:         report.WALBytes,
			ReclaimableBytes: report.ReclaimableBytes,
		}
		out.StorageKnown = true
	}
	if entries, ok := settings.Runtimes(); ok {
		out.RuntimesKnown = true
		for _, entry := range entries {
			out.Runtimes = append(out.Runtimes, ui.RuntimeSnapshot{
				Name:    entry.Name,
				Version: entry.Version,
				Status:  string(entry.Status),
				Reason:  entry.Reason,
			})
		}
	}
	if capabilities, ok := settings.Modality(); ok {
		out.Modality.Supported = capabilities.Supported
		out.Modality.Reason = capabilities.UnavailableReason
		if capabilities.TaskSupported != nil {
			out.Modality.TaskSupported = *capabilities.TaskSupported
			out.Modality.TaskKnown = true
		}
	}
	return out
}

// Extensions 把扩展领域四类最近确认的读数映射成界面快照。
func (w *App) Extensions() ui.ExtensionsSnapshot {
	service := w.A.Extensions()
	if service == nil {
		return ui.ExtensionsSnapshot{}
	}
	out := ui.ExtensionsSnapshot{}
	plugins := service.PluginSnapshot()
	out.Plugins = ui.ExtListSnapshot{ReadError: plugins.ReadError, CapabilityMissing: plugins.CapabilityMissing}
	for _, item := range plugins.Items {
		out.Plugins.Items = append(out.Plugins.Items, ui.ExtItemSnapshot{
			ID:      item.ID,
			Name:    firstNonEmpty(item.Name, item.ID),
			Version: item.Version,
			Enabled: item.Enabled,
			Status:  item.Status,
			Error:   item.Error,
		})
	}
	skills := service.SkillSnapshot()
	out.Skills = ui.ExtListSnapshot{ReadError: skills.ReadError, CapabilityMissing: skills.CapabilityMissing}
	for _, item := range skills.Items {
		out.Skills.Items = append(out.Skills.Items, ui.ExtItemSnapshot{
			ID:      item.ID,
			Name:    firstNonEmpty(item.Name, item.ID),
			Enabled: item.Enabled,
		})
	}
	hooks := service.HookSnapshot()
	out.Hooks = ui.ExtListSnapshot{ReadError: hooks.ReadError, CapabilityMissing: hooks.CapabilityMissing}
	for _, item := range hooks.Items {
		out.Hooks.Items = append(out.Hooks.Items, ui.ExtItemSnapshot{
			ID:      item.ID,
			Name:    firstNonEmpty(item.Name, item.ID),
			Enabled: item.Enabled,
			Event:   item.Event,
			Command: item.Command,
			Source:  string(item.Source),
		})
	}
	servers := service.ToolServerSnapshot()
	out.MCP = ui.ExtListSnapshot{ReadError: servers.ReadError, CapabilityMissing: servers.CapabilityMissing}
	for _, item := range servers.Items {
		out.MCP.Items = append(out.MCP.Items, ui.ExtItemSnapshot{
			ID:      item.ID,
			Name:    firstNonEmpty(item.Name, item.ID),
			Enabled: item.Enabled,
			Status:  item.AuthStatus,
		})
	}
	w.importMu.Lock()
	out.MCPImportSources = append([]ui.MCPImportSourceSnapshot(nil), w.importSources...)
	out.ImportError = w.importErr
	w.importMu.Unlock()
	return out
}

// Schedules 把定时任务最近确认的读数映射成界面快照。
func (w *App) Schedules() ui.SchedulesSnapshot {
	service := w.A.Extensions()
	if service == nil {
		return ui.SchedulesSnapshot{}
	}
	snapshot := service.ScheduledTaskSnapshot()
	out := ui.SchedulesSnapshot{ReadError: snapshot.ReadError, CapabilityMissing: snapshot.CapabilityMissing}
	for _, item := range snapshot.Items {
		out.Items = append(out.Items, ui.ScheduleSnapshot{
			ID:         item.ID,
			Name:       item.Name,
			Schedule:   item.Schedule,
			Enabled:    item.Enabled,
			Running:    item.Running,
			NextRunAt:  item.NextRunAt,
			LastRunAt:  item.LastRunAt,
			LastStatus: item.LastStatus,
		})
	}
	return out
}

// Preferences 是客户端本地偏好；不从服务端读。
func (w *App) Preferences() ui.PreferencesSnapshot {
	prefs := w.A.Preferences()
	return ui.PreferencesSnapshot{
		Theme: prefs.Theme, MotionOff: prefs.MotionOff,
		TimeZone: prefs.TimeZone, SidebarWidth: prefs.SidebarWidth,
	}
}

// SetPreferences 落盘本地偏好。
func (w *App) SetPreferences(ctx context.Context, prefs ui.PreferencesSnapshot) error {
	current := w.A.Preferences()
	current.Theme = prefs.Theme
	current.MotionOff = prefs.MotionOff
	current.TimeZone = prefs.TimeZone
	current.SidebarWidth = prefs.SidebarWidth
	if err := w.A.UpdatePreferences(current); err != nil {
		return err
	}
	w.changed()
	return nil
}

// Terminal 是当前会话终端的状态；输出由 pty 流式读取，本批不缓存。
func (w *App) Terminal() ui.TerminalSnapshot {
	registry := w.A.Terminals()
	if registry == nil || w.currentSession() == "" {
		return ui.TerminalSnapshot{}
	}
	terminal, ok := registry.Get(w.currentSession())
	if !ok {
		return ui.TerminalSnapshot{}
	}
	out := ui.TerminalSnapshot{Active: true, Running: terminal.Alive()}
	if !out.Running {
		if status, ok := terminal.ExitStatus(); ok {
			out.Exited = true
			out.ExitCode = status.Code
		}
	}
	return out
}

// providerLabel 把 provider 标识换成展示名（没有展示名时用标识）。
func providerLabel(models settings.ModelSnapshot, id string) string {
	for _, provider := range models.Providers {
		if provider.ID == id {
			return firstNonEmpty(provider.Name, provider.ID)
		}
	}
	return id
}

// modelLabel 把模型标识换成展示名（没有展示名时用标识）。
func modelLabel(models settings.ModelSnapshot, providerID, modelID string) string {
	for _, provider := range models.Providers {
		if providerID != "" && provider.ID != providerID {
			continue
		}
		for _, model := range provider.Models {
			if model.ID == modelID {
				return firstNonEmpty(model.Name, model.ID)
			}
		}
	}
	return modelID
}

// modelLabels 返回当前可选模型的名字；同一标识只出现一次。
func modelLabels(models settings.ModelSnapshot) []string {
	seen := make(map[string]struct{})
	var out []string
	for _, provider := range models.Providers {
		for _, model := range provider.Models {
			name := firstNonEmpty(model.Name, model.ID)
			if name == "" {
				continue
			}
			if _, ok := seen[name]; ok {
				continue
			}
			seen[name] = struct{}{}
			out = append(out, name)
		}
	}
	return out
}

// avatarInitial 取身份字段里第一个字母或数字作为头像首字母；都没有时为空。
func avatarInitial(values ...string) string {
	for _, value := range values {
		for _, r := range value {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return strings.ToUpper(string(r))
			}
		}
	}
	return ""
}

// firstNonEmpty 返回第一个非空白字符串。
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// ---- ui.Intents ----

func (w *App) OpenSession(ctx context.Context, name string) error {
	if _, err := w.A.OpenSession(ctx, name); err != nil {
		return fmt.Errorf("打开会话 %s：%w", name, err)
	}
	w.setSession(name)
	w.changed()
	return nil
}

func (w *App) Submit(ctx context.Context, text string) error {
	return w.SubmitWithContent(ctx, text, nil)
}

// SubmitWithContent 提交文本 + 内容块；超过 4KB 的长文本先 attachment.put，
// 再以 `[Pasted text: path (N lines)]` 文本块提交。
func (w *App) SubmitWithContent(
	ctx context.Context,
	text string,
	blocks []ui.ContentBlockSnapshot,
) error {
	if w.currentSession() == "" {
		return fmt.Errorf("没有打开的会话")
	}
	if strings.TrimSpace(text) == "" && len(blocks) == 0 {
		return fmt.Errorf("提交文本为空")
	}
	content := make([]app.SubmitContentBlock, 0, len(blocks)+1)
	if len(text) > 4096 {
		ref, err := w.WriteAttachment(ctx, "pasted-text.txt", text)
		if err != nil {
			return err
		}
		lines := strings.Count(text, "\n") + 1
		content = append(content, app.SubmitContentBlock{
			Kind: "text", Text: fmt.Sprintf("[Pasted text: %s (%d lines)]", ref.Path, lines),
		})
		text = ""
	}
	for _, block := range blocks {
		mapped := app.SubmitContentBlock{Kind: block.Kind, Text: block.Text}
		if block.Image != nil {
			mapped.Image = &app.SubmitImage{
				FileID: block.Image.FileID, SourceType: block.Image.SourceType,
				MediaType: block.Image.MediaType, Data: block.Image.Data, URL: block.Image.URL,
			}
		}
		content = append(content, mapped)
	}
	_, err := w.A.Submit(ctx, app.SubmitRequest{
		SessionID: w.currentSession(), Text: text, Content: content,
	})
	w.changed()
	return err
}

func (w *App) Queue(ctx context.Context, text string) error { return w.Submit(ctx, text) }

func (w *App) Stop(ctx context.Context) error {
	if w.currentSession() == "" {
		return fmt.Errorf("没有打开的会话")
	}
	_, err := w.A.Stop(ctx, w.currentSession())
	w.changed()
	return err
}

func (w *App) RespondApproval(ctx context.Context, decision string) error {
	if w.currentSession() == "" {
		return fmt.Errorf("没有打开的会话")
	}
	view, ok := w.A.View(w.currentSession())
	if !ok || !view.HasCurrent {
		return fmt.Errorf("没有待决审批")
	}
	_, err := w.A.Respond(ctx, w.currentSession(), view.Current.RequestID, approvalResponse(decision))
	w.changed()
	return err
}

func (w *App) RespondQuestion(ctx context.Context, answer string) error {
	if w.currentSession() == "" {
		return fmt.Errorf("没有打开的会话")
	}
	view, ok := w.A.View(w.currentSession())
	if !ok || !view.HasCurrent {
		return fmt.Errorf("没有待答提问")
	}
	_, err := w.A.Respond(ctx, w.currentSession(), view.Current.RequestID, questionResponse(answer))
	w.changed()
	return err
}

// SkipQuestion 是提问卡的「跳过」：按取消提交，不当作拒绝。
func (w *App) SkipQuestion(ctx context.Context) error {
	if w.currentSession() == "" {
		return fmt.Errorf("没有打开的会话")
	}
	view, ok := w.A.View(w.currentSession())
	if !ok || !view.HasCurrent {
		return fmt.Errorf("没有待答提问")
	}
	_, err := w.A.Respond(ctx, w.currentSession(), view.Current.RequestID,
		interaction.Response{Decision: interaction.DecisionCancel})
	w.changed()
	return err
}

func (w *App) RetrySubmit(ctx context.Context) error { return w.Submit(ctx, "") }

func (w *App) CreateSession(ctx context.Context, cwd string) error {
	if strings.TrimSpace(cwd) == "" {
		// 界面没有目录选择器：沿用当前会话的工作目录；没有打开会话时取目录首项。
		cwd = w.sessionWorkspace(w.currentSession())
		if cwd == "" {
			if names := w.SessionNames(); len(names) > 0 {
				cwd = w.sessionWorkspace(names[0])
			}
		}
	}
	if cwd == "" {
		return fmt.Errorf("新建会话缺少工作目录")
	}
	if _, err := w.A.CreateSession(ctx, cwd, ""); err != nil {
		return err
	}
	// 新建之后目录要刷新一次，否则列表还是旧的。
	return w.refreshCatalog(ctx)
}

// ---- 会话目录 ----

// RenameSession 走目录领域改名；成功后目录快照里的标题会更新。
func (w *App) RenameSession(ctx context.Context, sessionID, title string) error {
	domain := w.A.Catalog()
	if domain == nil {
		return fmt.Errorf("目录未就绪")
	}
	if _, err := domain.Rename(ctx, catalog.RenameRequest{SessionID: sessionID, Title: title}); err != nil {
		return err
	}
	w.changed()
	return nil
}

// ArchiveSession 是本地归档隐藏：只改本地目录列表，不写服务端。
func (w *App) ArchiveSession(ctx context.Context, sessionID string, archived bool) error {
	domain := w.A.Catalog()
	if domain == nil {
		return fmt.Errorf("目录未就绪")
	}
	var err error
	if archived {
		err = domain.Hide(sessionID)
	} else {
		err = domain.Unhide(sessionID)
	}
	if err != nil {
		return err
	}
	w.changed()
	return nil
}

// DeleteSession 先释放订阅再删除；成功后当前会话切到列表首项（没有则清空）。
func (w *App) DeleteSession(ctx context.Context, sessionID string) error {
	domain := w.A.Catalog()
	if domain == nil {
		return fmt.Errorf("目录未就绪")
	}
	if err := domain.Delete(ctx, catalog.DeleteRequest{SessionID: sessionID}); err != nil {
		return err
	}
	if w.currentSession() == sessionID {
		if names := w.SessionNames(); len(names) > 0 {
			w.setSession(names[0])
		} else {
			w.setSession("")
		}
	}
	w.changed()
	return nil
}

// ExportSession 把会话导出到工作目录下的一个带时间戳的 Markdown 文件。
//
// 界面没有路径选择器；这里不覆盖已有文件，路径由会话工作目录与时间戳决定，
// 返回值交给界面显示。没有工作目录时明确失败，不在任意目录落盘。
func (w *App) ExportSession(ctx context.Context, sessionID string) (string, error) {
	domain := w.A.Catalog()
	if domain == nil {
		return "", fmt.Errorf("目录未就绪")
	}
	entry, ok := domain.Entry(sessionID)
	if !ok {
		return "", fmt.Errorf("目录里没有该会话")
	}
	if strings.TrimSpace(entry.Workspace) == "" {
		return "", fmt.Errorf("会话没有工作目录，无法选择导出位置")
	}
	name := "dimagent-export-" + time.Now().Format("20060102-150405") + ".md"
	path := filepath.Join(entry.Workspace, name)
	result, err := domain.Export(ctx, catalog.ExportRequest{
		SessionID: sessionID,
		Path:      path,
		Title:     entry.Title,
	})
	if err != nil {
		return "", err
	}
	return result.Path, nil
}

// ---- 设置 ----

// SetPermissionTier 走会话方法切换权限档位；服务端没有该方法时按能力缺失返回。
func (w *App) SetPermissionTier(ctx context.Context, tier string) error {
	if w.currentSession() == "" {
		return fmt.Errorf("没有打开的会话")
	}
	if strings.TrimSpace(tier) == "" {
		return fmt.Errorf("权限档位为空")
	}
	if err := w.A.SetPermissionTier(ctx, w.currentSession(), interaction.PermissionTier(tier)); err != nil {
		return err
	}
	w.changed()
	return nil
}

// SelectModel 切换当前模型；ExpectedRevision 取最近一次读取的选择版本。
func (w *App) SelectModel(ctx context.Context, providerID, modelID string) error {
	manager := w.A.Settings()
	if manager == nil {
		return fmt.Errorf("设置领域未就绪")
	}
	revision := ""
	if models, ok := manager.Models(); ok && models.Selection != nil {
		revision = models.Selection.Revision
	}
	_, err := manager.SelectModel(ctx, settings.SelectModelRequest{
		ProviderID:       providerID,
		ModelID:          modelID,
		ExpectedRevision: revision,
	})
	if err != nil {
		return err
	}
	w.changed()
	return nil
}

// WriteCommandRules 写完整规则集合；成功后用回执更新本地缓存。
func (w *App) WriteCommandRules(ctx context.Context, rules []ui.RuleSnapshot) error {
	w.rulesMu.Lock()
	revision := w.rulesRevision
	w.rulesMu.Unlock()
	mapped := make([]interaction.CommandRule, 0, len(rules))
	for _, rule := range rules {
		mapped = append(mapped, interaction.CommandRule{
			Command:  rule.Command,
			Match:    interaction.CommandMatch(rule.Match),
			Decision: interaction.CommandDecision(rule.Decision),
		})
	}
	set, err := w.A.WriteCommandRules(ctx, revision, mapped)
	if err != nil {
		return err
	}
	w.rulesMu.Lock()
	w.rulesRevision = set.Revision
	w.rules = w.rules[:0]
	for _, rule := range set.Rules {
		w.rules = append(w.rules, ui.RuleSnapshot{
			Command: rule.Command, Decision: string(rule.Decision), Match: string(rule.Match),
		})
	}
	w.rulesKnown = true
	w.rulesErr = ""
	w.rulesMu.Unlock()
	w.changed()
	return nil
}

// ---- 分支 ----

// SelectBranch 切换当前查看的分支。
func (w *App) SelectBranch(ctx context.Context, branchID string) error {
	if w.currentSession() == "" {
		return fmt.Errorf("没有打开的会话")
	}
	if _, err := w.A.SelectBranch(ctx, w.currentSession(), branchID); err != nil {
		return err
	}
	w.changed()
	return nil
}

// CreateBranch 从当前分支的指定轮次切出新分支。
func (w *App) CreateBranch(ctx context.Context, parentID string, turnIndex int, prompt string) error {
	if w.currentSession() == "" {
		return fmt.Errorf("没有打开的会话")
	}
	if _, err := w.A.CreateBranch(ctx, w.currentSession(), parentID, turnIndex, prompt); err != nil {
		return err
	}
	w.changed()
	return nil
}

// EndBranch 结束一个非主线分支。
func (w *App) EndBranch(ctx context.Context, branchID string) error {
	if w.currentSession() == "" {
		return fmt.Errorf("没有打开的会话")
	}
	if _, err := w.A.EndBranch(ctx, w.currentSession(), branchID); err != nil {
		return err
	}
	w.changed()
	return nil
}

// ---- 附件 ----

// WriteAttachment 把文本存成会话文件；等到终态后返回写入的附件名。
func (w *App) WriteAttachment(ctx context.Context, name, text string) (ui.AttachmentRefSnapshot, error) {
	if w.currentSession() == "" {
		return ui.AttachmentRefSnapshot{}, fmt.Errorf("没有打开的会话")
	}
	view, err := w.A.WriteAttachment(ctx, w.currentSession(), name, text)
	if err != nil {
		return ui.AttachmentRefSnapshot{}, err
	}
	if settled, waitErr := w.A.Wait(ctx, view.ID); waitErr == nil {
		view = settled
	}
	ref := ui.AttachmentRefSnapshot{Name: name}
	if len(view.Result) > 0 {
		var reply struct {
			AttachmentID string `json:"attachmentId"`
			Path         string `json:"path"`
		}
		if err := json.Unmarshal(view.Result, &reply); err == nil {
			ref.ID = reply.AttachmentID
			ref.Path = reply.Path
		}
	}
	w.changed()
	return ref, nil
}

// Reconnect 重连控制连接。
func (w *App) Reconnect(ctx context.Context) error {
	if err := w.A.Reconnect(ctx); err != nil {
		return err
	}
	w.setPhase("connected", "")
	w.watchConnection()
	w.changed()
	return nil
}

// ---- 目标 ----

// ReadGoal 读当前目标（session.goal.get），返回原始回执文本。
func (w *App) ReadGoal(ctx context.Context) (string, error) {
	if w.currentSession() == "" {
		return "", fmt.Errorf("没有打开的会话")
	}
	raw, err := w.A.ReadGoal(ctx, w.currentSession())
	if err != nil {
		return "", err
	}
	return string(raw), nil
}

// SetGoal 新建或覆盖目标。
func (w *App) SetGoal(ctx context.Context, objective string) error {
	if w.currentSession() == "" {
		return fmt.Errorf("没有打开的会话")
	}
	if err := w.A.SetGoal(ctx, w.currentSession(), objective); err != nil {
		return err
	}
	w.changed()
	return nil
}

// EditGoal 修改目标文本。
func (w *App) EditGoal(ctx context.Context, objective string) error {
	if w.currentSession() == "" {
		return fmt.Errorf("没有打开的会话")
	}
	if err := w.A.EditGoal(ctx, w.currentSession(), objective); err != nil {
		return err
	}
	w.changed()
	return nil
}

// PauseGoal 暂停目标。
func (w *App) PauseGoal(ctx context.Context) error {
	if w.currentSession() == "" {
		return fmt.Errorf("没有打开的会话")
	}
	if err := w.A.PauseGoal(ctx, w.currentSession()); err != nil {
		return err
	}
	w.changed()
	return nil
}

// ResumeGoal 恢复目标。
func (w *App) ResumeGoal(ctx context.Context) error {
	if w.currentSession() == "" {
		return fmt.Errorf("没有打开的会话")
	}
	if err := w.A.ResumeGoal(ctx, w.currentSession()); err != nil {
		return err
	}
	w.changed()
	return nil
}

// ClearGoal 清除目标。
func (w *App) ClearGoal(ctx context.Context) error {
	if w.currentSession() == "" {
		return fmt.Errorf("没有打开的会话")
	}
	if err := w.A.ClearGoal(ctx, w.currentSession()); err != nil {
		return err
	}
	w.changed()
	return nil
}

// StopExecTask 停止一条后台任务。
func (w *App) StopExecTask(ctx context.Context, taskID string) error {
	if w.currentSession() == "" {
		return fmt.Errorf("没有打开的会话")
	}
	if err := w.A.StopExecTask(ctx, w.currentSession(), taskID); err != nil {
		return err
	}
	w.changed()
	return nil
}

// ---- 供应商管理 ----

// CreateProvider 新建一个 provider；driver 取 openai，认证用 bearer + 显式密钥。
func (w *App) CreateProvider(ctx context.Context, id, name, baseURL, apiKey string) error {
	manager := w.A.Settings()
	if manager == nil {
		return fmt.Errorf("设置领域未就绪")
	}
	if !looksLikeUUID(id) {
		id = newUUID()
	}
	connection := settings.ProviderConnectionInput{
		DriverKind: "openai",
		BaseURL:    baseURL,
		Auth:       settings.ProviderAuth{Type: "bearer"},
	}
	if apiKey != "" {
		connection.Credential = &settings.ProviderCredential{Action: "set", Value: apiKey}
	}
	if _, err := manager.CreateProvider(ctx, settings.ProviderCreateRequest{
		ID: id, DisplayName: name, Connection: connection, Enabled: true,
	}); err != nil {
		return err
	}
	w.refreshProviderRecords(ctx)
	w.changed()
	return nil
}

// UpdateProvider 修改 provider 的显示名、Base URL 与密钥。
func (w *App) UpdateProvider(ctx context.Context, id, name, baseURL, apiKey string) error {
	manager := w.A.Settings()
	if manager == nil {
		return fmt.Errorf("设置领域未就绪")
	}
	detail, err := manager.ReadProvider(ctx, id)
	if err != nil {
		return err
	}
	if detail.Version == nil {
		return fmt.Errorf("provider %s 没有可写版本", id)
	}
	patch := settings.ProviderPatch{}
	if name != "" {
		patch.DisplayName = &name
	}
	if baseURL != "" {
		patch.BaseURL = &baseURL
	}
	if apiKey != "" {
		patch.Credential = &settings.ProviderCredentialUpdate{Action: "set", Value: apiKey}
	}
	if _, err := manager.UpdateProvider(ctx, settings.ProviderUpdateRequest{
		ProviderID: id, ExpectedVersion: *detail.Version, Patch: patch,
	}); err != nil {
		return err
	}
	w.refreshProviderRecords(ctx)
	w.changed()
	return nil
}

// ProbeProvider 探测一个已存在的 provider，返回发现的模型数。
func (w *App) ProbeProvider(ctx context.Context, id string) (int, error) {
	manager := w.A.Settings()
	if manager == nil {
		return 0, fmt.Errorf("设置领域未就绪")
	}
	patch := &settings.ProviderPatch{}
	w.providerMu.Lock()
	for _, record := range w.providerRecords {
		if record.ID == id && record.BaseURL != "" {
			base := record.BaseURL
			patch.BaseURL = &base
			break
		}
	}
	w.providerMu.Unlock()
	result, err := manager.ProbeProvider(ctx, settings.ProviderProbeRequest{
		ProviderID: id,
		Patch:      patch,
	})
	if err != nil {
		return 0, err
	}
	return len(result.Models), nil
}

// SetProviderModelEnabled 启用/禁用 provider 下的一个模型。
func (w *App) SetProviderModelEnabled(
	ctx context.Context,
	providerID, modelID string,
	enabled bool,
) error {
	manager := w.A.Settings()
	if manager == nil {
		return fmt.Errorf("设置领域未就绪")
	}
	detail, err := manager.ReadProvider(ctx, providerID)
	if err != nil {
		return err
	}
	if detail.Version == nil {
		return fmt.Errorf("provider %s 没有可写版本", providerID)
	}
	if _, err := manager.SetModelEnabled(ctx, providerID, modelID, *detail.Version, enabled); err != nil {
		return err
	}
	w.refreshProviderRecords(ctx)
	w.changed()
	return nil
}

// looksLikeUUID 判断 provider 标识是否已是服务端要求的小写 UUID。
func looksLikeUUID(value string) bool {
	if len(value) != 36 || value[8] != '-' || value[13] != '-' || value[18] != '-' || value[23] != '-' {
		return false
	}
	for _, r := range value {
		if r == '-' {
			continue
		}
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

// newUUID 生成一个小写 UUID v4（provider 标识需要）。
func newUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	text := hex.EncodeToString(b[:])
	return text[0:8] + "-" + text[8:12] + "-" + text[12:16] + "-" + text[16:20] + "-" + text[20:32]
}

// ---- 账号登录 ----

// BeginLogin 发起一次浏览器或设备码登录，返回要显示的提示；不等待完成。
func (w *App) BeginLogin(ctx context.Context, method string) (ui.LoginPromptSnapshot, error) {
	domain := w.A.Account()
	if domain == nil {
		return ui.LoginPromptSnapshot{}, fmt.Errorf("账号领域未就绪")
	}
	providers, err := domain.Providers(ctx)
	if err != nil {
		return ui.LoginPromptSnapshot{}, err
	}
	loginMethod := account.LoginMethod(method)
	providerID := ""
	for _, provider := range providers {
		for _, declared := range provider.LoginMethods {
			if declared == loginMethod {
				providerID = provider.ID
				break
			}
		}
		if providerID != "" {
			break
		}
	}
	if providerID == "" {
		return ui.LoginPromptSnapshot{}, fmt.Errorf("没有声明 %s 登录方式的 provider", method)
	}
	start, err := domain.BeginLogin(ctx, account.LoginRequest{ProviderID: providerID, Method: loginMethod})
	if err != nil {
		return ui.LoginPromptSnapshot{}, err
	}
	prompt := ui.LoginPromptSnapshot{
		ProviderID: start.ProviderID, Method: string(start.Method), Attempt: start.Attempt,
		URL: start.AuthorizationURL, UserCode: start.UserCode, VerifyURL: start.VerificationURIComplete,
	}
	if prompt.VerifyURL == "" {
		prompt.VerifyURL = start.VerificationURI
	}
	w.accountMu.Lock()
	w.loginPrompt = prompt
	w.accountMu.Unlock()
	w.changed()
	return prompt, nil
}

// WaitLogin 等待当前登录尝试完成；成功后重读账号、用量与模型清单。
func (w *App) WaitLogin(ctx context.Context) error {
	domain := w.A.Account()
	if domain == nil {
		return fmt.Errorf("账号领域未就绪")
	}
	if _, err := domain.WaitLogin(ctx); err != nil {
		return err
	}
	_, _ = domain.Refresh(ctx)
	_, _ = domain.RefreshUsage(ctx)
	_, _ = domain.RefreshModels(ctx)
	w.accountMu.Lock()
	w.loginPrompt = ui.LoginPromptSnapshot{}
	w.accountMu.Unlock()
	w.changed()
	return nil
}

// CancelLogin 取消当前登录尝试。
func (w *App) CancelLogin(ctx context.Context) error {
	domain := w.A.Account()
	if domain == nil {
		return fmt.Errorf("账号领域未就绪")
	}
	domain.CancelLogin()
	w.accountMu.Lock()
	w.loginPrompt = ui.LoginPromptSnapshot{}
	w.accountMu.Unlock()
	w.changed()
	return nil
}

// Logout 退出当前账号。
func (w *App) Logout(ctx context.Context) error {
	domain := w.A.Account()
	if domain == nil {
		return fmt.Errorf("账号领域未就绪")
	}
	if _, err := domain.Logout(ctx, account.LogoutRequest{}); err != nil {
		return err
	}
	_, _ = domain.Refresh(ctx)
	w.changed()
	return nil
}

// ---- 扩展 ----

// SetExtensionEnabled 按类型分派：插件、技能、钩子与 MCP 都有启停写入。
func (w *App) SetExtensionEnabled(ctx context.Context, kind, id string, enabled bool) error {
	service := w.A.Extensions()
	if service == nil {
		return fmt.Errorf("扩展领域未就绪")
	}
	switch kind {
	case "plugin":
		if _, err := service.SetPluginEnabled(ctx, id, enabled); err != nil {
			return err
		}
		_, _ = service.ListPlugins(ctx)
	case "skill":
		if _, err := service.SetSkillEnabled(ctx, id, enabled); err != nil {
			return err
		}
		_, _ = service.ListSkills(ctx)
	case "hook":
		var hook extensions.Hook
		found := false
		for _, item := range service.HookSnapshot().Items {
			if item.ID == id {
				hook, found = item, true
				break
			}
		}
		if !found {
			return fmt.Errorf("钩子 %s 不在最近一次读数里", id)
		}
		hook.Enabled = enabled
		if _, err := service.WriteHook(ctx, hook); err != nil {
			return err
		}
		_, _ = service.ListHooks(ctx)
	case "mcp":
		if _, err := service.SetToolServerEnabled(ctx, id, enabled); err != nil {
			return err
		}
		_, _ = service.ListToolServers(ctx)
	default:
		return fmt.Errorf("未接：%s 没有启停写入方法", kind)
	}
	w.changed()
	return nil
}

// UninstallExtension 按类型分派：插件与 MCP 可卸载；技能没有卸载方法。
func (w *App) UninstallExtension(ctx context.Context, kind, id string) error {
	service := w.A.Extensions()
	if service == nil {
		return fmt.Errorf("扩展领域未就绪")
	}
	switch kind {
	case "plugin":
		if _, err := service.RemovePlugin(ctx, id); err != nil {
			return err
		}
		_, _ = service.ListPlugins(ctx)
	case "skill":
		if _, err := service.RemoveSkill(ctx, id); err != nil {
			return err
		}
		_, _ = service.ListSkills(ctx)
	case "mcp":
		if _, err := service.RemoveToolServer(ctx, id); err != nil {
			return err
		}
		_, _ = service.ListToolServers(ctx)
	default:
		return fmt.Errorf("未接：extensions.Service 没有 %s 卸载方法", kind)
	}
	w.changed()
	return nil
}

// TestMCP 显式探测一个工具服务器；只读，不改配置。
func (w *App) TestMCP(ctx context.Context, id string) (bool, string, error) {
	service := w.A.Extensions()
	if service == nil {
		return false, "", fmt.Errorf("扩展领域未就绪")
	}
	probe, err := service.ProbeToolServer(ctx, id)
	if err != nil {
		return false, "", err
	}
	return probe.Reachable, probe.Reason, nil
}

// ImportMCP 先读一次预览，再按预览把可导入的条目应用上去。
//
// 预览里的 identical 与 conflict/invalid 一律跳过；changed 用 replace；其余按
// import。没有可应用条目时不发 apply，返回 0。
func (w *App) ImportMCP(ctx context.Context, sourceID string) (int, error) {
	service := w.A.Extensions()
	if service == nil {
		return 0, fmt.Errorf("扩展领域未就绪")
	}
	if strings.TrimSpace(sourceID) == "" {
		return 0, fmt.Errorf("导入来源为空")
	}
	cwd := w.sessionWorkspace(w.currentSession())
	preview, err := service.PreviewMCPImport(ctx, extensions.MCPImportPreviewRequest{
		Cwd:    cwd,
		Source: extensions.SourceUser,
		From:   sourceID,
	})
	if err != nil {
		return 0, err
	}
	resolutions := make([]extensions.MCPImportResolution, 0, len(preview.Entries))
	for _, entry := range preview.Entries {
		switch entry.Status {
		case extensions.MCPImportIdentical, extensions.MCPImportConflict, extensions.MCPImportInvalid:
			resolutions = append(resolutions, extensions.MCPImportResolution{
				ID: entry.ID, Action: extensions.MCPImportActionSkip,
			})
		case extensions.MCPImportChanged:
			resolutions = append(resolutions, extensions.MCPImportResolution{
				ID: entry.ID, Action: extensions.MCPImportActionReplace,
			})
		default:
			resolutions = append(resolutions, extensions.MCPImportResolution{
				ID: entry.ID, Action: extensions.MCPImportActionImport,
			})
		}
	}
	applied := 0
	for _, resolution := range resolutions {
		if resolution.Action != extensions.MCPImportActionSkip {
			applied++
		}
	}
	if len(resolutions) == 0 {
		return 0, nil
	}
	if _, err := service.ApplyMCPImport(ctx, extensions.MCPImportApplyRequest{
		Cwd:                    cwd,
		Source:                 extensions.SourceUser,
		From:                   sourceID,
		ExpectedVersion:        preview.Revision,
		ExpectedSourceRevision: preview.SourceRevision,
		Resolutions:            resolutions,
	}); err != nil {
		return 0, err
	}
	_, _ = service.ListToolServers(ctx)
	w.refreshImportSources(ctx)
	w.changed()
	return applied, nil
}

// ---- 定时任务 ----

// SetScheduleEnabled 启停一条定时任务；成功后重读清单。
func (w *App) SetScheduleEnabled(ctx context.Context, jobID string, enabled bool) error {
	service := w.A.Extensions()
	if service == nil {
		return fmt.Errorf("扩展领域未就绪")
	}
	if _, err := service.SetScheduledTaskEnabled(ctx, jobID, enabled); err != nil {
		return err
	}
	_, _ = service.ListScheduledTasks(ctx)
	w.changed()
	return nil
}

// DeleteSchedule 删除一条定时任务；成功后重读清单。
func (w *App) DeleteSchedule(ctx context.Context, jobID string) error {
	service := w.A.Extensions()
	if service == nil {
		return fmt.Errorf("扩展领域未就绪")
	}
	if _, err := service.DeleteScheduledTask(ctx, jobID); err != nil {
		return err
	}
	_, _ = service.ListScheduledTasks(ctx)
	w.changed()
	return nil
}

// CancelModalityTask 取消一条运行中的模态任务；成功后重读任务清单。
func (w *App) CancelModalityTask(ctx context.Context, taskID string) error {
	if w.currentSession() == "" {
		return fmt.Errorf("没有打开的会话")
	}
	if strings.TrimSpace(taskID) == "" {
		return fmt.Errorf("任务标识为空")
	}
	view, err := w.A.ModalityWrite(ctx, app.ModalityWriteRequest{
		SessionID: w.currentSession(),
		Op:        app.ModalityTaskCancel,
		Params:    map[string]any{"taskId": taskID},
	})
	if err != nil {
		return err
	}
	_, _ = w.A.Wait(ctx, view.ID)
	w.refreshModalityTasks(ctx)
	w.changed()
	return nil
}

// ---- 终端 ----

// RunShell 登记一次会话内命令执行，返回运行标识供停止使用。
func (w *App) RunShell(ctx context.Context, command string) (string, error) {
	if w.currentSession() == "" {
		return "", fmt.Errorf("没有打开的会话")
	}
	view, err := w.A.RunShell(ctx, app.ShellRunRequest{SessionID: w.currentSession(), Command: command})
	if err != nil {
		return "", err
	}
	// 服务端可能在回执里给运行标识；等到终态后用它的值作为停止依据。
	if settled, waitErr := w.A.Wait(ctx, view.ID); waitErr == nil && settled.InputID != "" {
		view = settled
	}
	w.changed()
	return view.InputID, nil
}

// StopShell 停止一次已经发出的命令执行。
func (w *App) StopShell(ctx context.Context, runID string) error {
	if w.currentSession() == "" {
		return fmt.Errorf("没有打开的会话")
	}
	if strings.TrimSpace(runID) == "" {
		return fmt.Errorf("没有要停止的运行")
	}
	if _, err := w.A.StopShell(ctx, app.ShellStopRequest{SessionID: w.currentSession(), RunID: runID}); err != nil {
		return err
	}
	w.changed()
	return nil
}

// sessionWorkspace 返回目录条目里的工作目录；没有目录或条目时为空。
func (w *App) sessionWorkspace(sessionID string) string {
	if sessionID == "" {
		return ""
	}
	domain := w.A.Catalog()
	if domain == nil {
		return ""
	}
	entry, ok := domain.Entry(sessionID)
	if !ok {
		return ""
	}
	return entry.Workspace
}
