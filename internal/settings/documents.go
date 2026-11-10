package settings

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// Document 是一份设置文档及其版本。Revision 是服务端给出的前置条件，写入时
// 必须原样带回；Document 是服务端返回的键值对象，本包不解释键的业务含义。
type Document struct {
	Source   string
	Scope    Scope
	Revision string
	Document json.RawMessage
}

// SettingsReader 按来源与作用域读取设置文档。
type SettingsReader interface {
	ReadSettings(ctx context.Context, source string, scope Scope) (Document, error)
}

// UpdateRequest 是一次设置写入。ExpectedRevision 必填，为空直接拒绝；服务端
// 或本地缓存发现版本已变时返回 ErrRevisionConflict，不覆盖、不重试。
type UpdateRequest struct {
	Source           string
	Scope            Scope
	ExpectedRevision string
	Document         json.RawMessage
}

// settingsUpdateParams 是交给写登记的透明参数形状。集成层按 Op 映射到服务端
// 方法，并把 Scope 映射成服务端的 user / {project} 形状。
type settingsUpdateParams struct {
	Op               string          `json:"op"`
	Source           string          `json:"source"`
	Scope            Scope           `json:"scope"`
	ExpectedRevision string          `json:"expectedRevision"`
	Document         json.RawMessage `json:"document"`
}

// settingsUpdateReply 是写登记成功回执里的设置文档形状。
type settingsUpdateReply struct {
	Revision string          `json:"revision"`
	Document json.RawMessage `json:"document"`
}

// Read 读取一份设置文档并缓存最近一次确认值。
//
// 前置条件：先调用 RefreshSources。来源必须声明目标作用域；项目作用域还要求
// 项目根在 RefreshProjects 的授权清单里。能力缺失、事实不合法或来源未声明都
// 返回分类错误，不返回空文档。
func (m *Manager) Read(ctx context.Context, source string, scope Scope) (Document, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	source, err := normalizeSourceName(source)
	if err != nil {
		return Document{}, err
	}
	if err := scope.Validate(); err != nil {
		return Document{}, err
	}
	if err := m.requireFeature(FeatureSettingsRead); err != nil {
		return Document{}, err
	}
	declared, err := m.declaredSource(source)
	if err != nil {
		return Document{}, err
	}
	if !declared.Allows(scope.Kind) {
		return Document{}, fmt.Errorf("%w: 来源 %q 未声明 %s", ErrScopeNotAllowed, source, scope.Kind)
	}
	if declared.WriteOnly {
		return Document{}, fmt.Errorf("%w: 来源 %q", ErrSourceWriteOnly, source)
	}
	if scope.Kind == ScopeProject {
		if err := m.authorizeProject(scope.Project); err != nil {
			return Document{}, err
		}
	}
	return m.readDocument(ctx, source, scope)
}

// Cached 返回最近一次读取到的设置文档；第二个返回值表示是否有缓存。
func (m *Manager) Cached(source string, scope Scope) (Document, bool) {
	if err := scope.Validate(); err != nil {
		return Document{}, false
	}
	source, err := normalizeSourceName(source)
	if err != nil {
		return Document{}, false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	document, ok := m.documents[scope.cacheKey(source)]
	if !ok {
		return Document{}, false
	}
	return cloneDocument(document), true
}

// readDocument 只做读取、校验与缓存，不重复检查来源声明；调用前已校验过。
func (m *Manager) readDocument(ctx context.Context, source string, scope Scope) (Document, error) {
	if m.cfg.Settings == nil {
		return Document{}, missing(FeatureSettingsRead, "未提供设置读取接口")
	}
	document, err := m.cfg.Settings.ReadSettings(ctx, source, scope)
	if err != nil {
		return Document{}, readFailure(FeatureSettingsRead, ErrSettingsRead, err)
	}
	document.Source = source
	document.Scope = scope
	normalized, err := normalizeDocument(document)
	if err != nil {
		return Document{}, err
	}
	m.mu.Lock()
	m.documents[scope.cacheKey(source)] = normalized
	m.mu.Unlock()
	return cloneDocument(normalized), nil
}

// Update 写入一份设置文档。
//
// 流程固定为：结构校验 → 本地版本前置条件 → 经写登记登记一次 KindManagementWrite
// → 等待终态。本地缓存显示版本已变时直接返回冲突且不发出请求；服务端拒绝后
// 只再读一次当前文档来判断冲突，不做任何重发。成功后采用回执里的设置文档，
// 回执不合法则返回 ErrMalformedWriteResult，不采用本地拼出的替代值。
func (m *Manager) Update(ctx context.Context, request UpdateRequest) (Document, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	// 一次只做一条管理写入：并发写入若各自成功，回执写回缓存的顺序无法保证。
	m.writeMu.Lock()
	defer m.writeMu.Unlock()
	source, err := normalizeSourceName(request.Source)
	if err != nil {
		return Document{}, err
	}
	if err := request.Scope.Validate(); err != nil {
		return Document{}, err
	}
	if err := m.requireFeature(FeatureSettingsUpdate); err != nil {
		return Document{}, err
	}
	declared, err := m.declaredSource(source)
	if err != nil {
		return Document{}, err
	}
	if !declared.Allows(request.Scope.Kind) {
		return Document{}, fmt.Errorf("%w: 来源 %q 未声明 %s", ErrScopeNotAllowed, source, request.Scope.Kind)
	}
	if request.Scope.Kind == ScopeProject {
		if err := m.authorizeProject(request.Scope.Project); err != nil {
			return Document{}, err
		}
	}
	expected, ok := trimNonEmpty(request.ExpectedRevision)
	if !ok {
		return Document{}, ErrRevisionMissing
	}
	normalized, object, err := normalizeObject(request.Document)
	if err != nil {
		return Document{}, err
	}
	if request.Scope.Kind == ScopeProject {
		for key := range object {
			if containsKey(declared.UserOnly, key) {
				return Document{}, fmt.Errorf("%w: 键 %q 只能写入用户作用域", ErrUserOnlySetting, key)
			}
		}
	}
	// 本地版本前置条件：缓存里的版本已经变了就不发请求，避免覆盖。
	if cached, ok := m.Cached(source, request.Scope); ok && cached.Revision != expected {
		return Document{}, fmt.Errorf("%w: 期望版本 %q，本地最近确认版本 %q", ErrRevisionConflict, expected, cached.Revision)
	}
	if m.writer() == nil {
		return Document{}, fmt.Errorf("%w: %s", ErrWriteUnavailable, "未提供写登记")
	}

	view, err := m.registerWrite(ctx, settingsUpdateParams{
		Op:               "settings.update",
		Source:           source,
		Scope:            request.Scope,
		ExpectedRevision: expected,
		Document:         normalized,
	})
	if err != nil {
		return Document{}, err
	}
	if view.Status != write.StatusSucceeded {
		return Document{}, m.updateFailure(ctx, source, request.Scope, expected, view)
	}
	reply, err := parseUpdateReply(view.Result)
	if err != nil {
		return Document{}, err
	}
	document := Document{
		Source:   source,
		Scope:    request.Scope,
		Revision: reply.Revision,
		Document: reply.Document,
	}
	m.mu.Lock()
	m.documents[request.Scope.cacheKey(source)] = document
	m.mu.Unlock()
	return cloneDocument(document), nil
}

// updateFailure 在写入没有成功时区分版本冲突与普通失败。它只读一次当前文档，
// 不发送任何写请求，也不重试。
func (m *Manager) updateFailure(ctx context.Context, source string, scope Scope, expected string, view write.IntentView) error {
	if current, err := m.readDocument(ctx, source, scope); err == nil && current.Revision != expected {
		return fmt.Errorf("%w: 期望版本 %q，服务端当前版本 %q", ErrRevisionConflict, expected, current.Revision)
	}
	reason := view.Reason
	if reason == "" {
		reason = string(view.Status)
	}
	return fmt.Errorf("%w: %s", ErrUpdateFailed, reason)
}

// parseUpdateReply 解析写登记成功回执。只接受带版本与对象文档的回执；不补值。
func parseUpdateReply(raw json.RawMessage) (settingsUpdateReply, error) {
	var reply settingsUpdateReply
	if len(raw) == 0 {
		return reply, fmt.Errorf("%w: 回执为空", ErrMalformedWriteResult)
	}
	if err := json.Unmarshal(raw, &reply); err != nil {
		return reply, fmt.Errorf("%w: %v", ErrMalformedWriteResult, err)
	}
	revision, ok := trimNonEmpty(reply.Revision)
	if !ok {
		return reply, fmt.Errorf("%w: 回执缺少版本", ErrMalformedWriteResult)
	}
	document, _, err := normalizeObject(reply.Document)
	if err != nil {
		return reply, fmt.Errorf("%w: %v", ErrMalformedWriteResult, err)
	}
	reply.Revision = revision
	reply.Document = document
	return reply, nil
}

// registerWrite 把一次管理写入登记到写登记并等待终态。它只登记一次，不重试。
func (m *Manager) registerWrite(ctx context.Context, params any) (write.IntentView, error) {
	raw, err := json.Marshal(params)
	if err != nil {
		return write.IntentView{}, fmt.Errorf("编码管理写入参数: %w", err)
	}
	m.mu.Lock()
	m.writeSeq++
	seq := m.writeSeq
	m.mu.Unlock()
	writer := m.writer()
	if writer == nil {
		return write.IntentView{}, ErrWriteUnavailable
	}
	view, err := writer.Register(write.Intent{
		Key:    fmt.Sprintf("management:%d", seq),
		Kind:   write.KindManagementWrite,
		Params: raw,
	})
	if err != nil {
		return write.IntentView{}, fmt.Errorf("登记管理写入: %w", err)
	}
	view, err = writer.Wait(ctx, view.ID)
	if err != nil && view.Status == "" {
		return write.IntentView{}, err
	}
	return view, nil
}

func (m *Manager) writer() Writer {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.cfg.Writer
}

// normalizeDocument 校验读取到的设置文档并返回副本。
func normalizeDocument(document Document) (Document, error) {
	revision, ok := trimNonEmpty(document.Revision)
	if !ok {
		return Document{}, fmt.Errorf("%w: 缺少版本", ErrMalformedDocument)
	}
	normalized, _, err := normalizeObject(document.Document)
	if err != nil {
		return Document{}, err
	}
	document.Revision = revision
	document.Document = normalized
	return document, nil
}

func cloneDocument(document Document) Document {
	document.Document = cloneRaw(document.Document)
	return document
}
