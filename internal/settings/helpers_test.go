package settings

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/account"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/write"
)

type stubFeatures map[string]bool

func (f stubFeatures) HasFeature(name string) bool { return f[name] }

func allFeatures() stubFeatures {
	return stubFeatures{
		FeatureSettingsSources:           true,
		FeatureSettingsProjects:          true,
		FeatureSettingsRead:              true,
		FeatureSettingsUpdate:            true,
		FeatureModelProviders:            true,
		FeatureModelSelection:            true,
		FeatureProviderList:              true,
		FeatureProviderManagement:        true,
		FeatureProviderCreate:            true,
		FeatureProviderUpdate:            true,
		FeatureProviderProbe:             true,
		FeatureProviderModelsSetEnabled:  true,
		FeatureProviderModelsEdit:        true,
		FeatureProviderModelsReset:       true,
		FeatureProviderPreferencesUpdate: true,
		FeatureCollaborationPolicy:       true,
		FeatureCollaborationSetPolicy:    true,
		FeatureAgentList:                 true,
		FeatureSessionAgentGet:           true,
		FeatureSessionAgentUpdate:        true,
		FeatureStorageReport:             true,
		FeatureRuntimeList:               true,
		FeatureRuntimeInstall:            true,
		FeatureModalityCapabilities:      true,
	}
}

type stubSources struct {
	value Sources
	err   error
	calls int
}

func (s *stubSources) ReadSettingsSources(context.Context) (Sources, error) {
	s.calls++
	if s.err != nil {
		return Sources{}, s.err
	}
	return s.value, nil
}

type stubProjects struct {
	value []Project
	err   error
	calls int
}

func (s *stubProjects) ReadSettingsProjects(context.Context) ([]Project, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.value, nil
}

type stubSettings struct {
	read func(context.Context, string, Scope) (Document, error)
	// documents 按作用域缓存键返回固定文档，便于多作用域测试。
	documents map[string]Document
	err       error
	calls     int
}

func (s *stubSettings) ReadSettings(ctx context.Context, source string, scope Scope) (Document, error) {
	s.calls++
	if s.err != nil {
		return Document{}, s.err
	}
	if s.read != nil {
		return s.read(ctx, source, scope)
	}
	document, ok := s.documents[scope.cacheKey(source)]
	if !ok {
		return Document{}, errors.New("测试没有准备该设置文档")
	}
	return document, nil
}

type stubCatalog struct {
	value ModelCatalog
	err   error
	calls int
}

func (s *stubCatalog) ReadModelCatalog(context.Context) (ModelCatalog, error) {
	s.calls++
	if s.err != nil {
		return ModelCatalog{}, s.err
	}
	return s.value, nil
}

type stubAccount struct {
	list  account.ModelList
	calls int
}

func (s *stubAccount) Models() account.ModelList {
	s.calls++
	return s.list
}

type stubStorage struct {
	value StorageReading
	err   error
	calls int
}

func (s *stubStorage) ReadStorage(context.Context) (StorageReading, error) {
	s.calls++
	if s.err != nil {
		return StorageReading{}, s.err
	}
	return s.value, nil
}

type stubRuntimes struct {
	value []RuntimeEntry
	err   error
	calls int
}

func (s *stubRuntimes) ReadRuntimes(context.Context) ([]RuntimeEntry, error) {
	s.calls++
	if s.err != nil {
		return nil, s.err
	}
	return s.value, nil
}

type stubModality struct {
	value ModalityCapabilities
	err   error
	calls int
}

func (s *stubModality) ReadModalityCapabilities(context.Context) (ModalityCapabilities, error) {
	s.calls++
	if s.err != nil {
		return ModalityCapabilities{}, s.err
	}
	return s.value, nil
}

// stubInspector 表示单元测试不做未知核对。
type stubInspector struct{}

func (stubInspector) LookupInput(context.Context, write.InputQuery) (write.InputState, error) {
	return write.InputState{}, nil
}

// senderFunc 是写登记发送方的测试替身；它记录每一次请求。
type senderFunc struct {
	mu       sync.Mutex
	requests []write.Request
	fn       func(context.Context, write.Request) (write.Reply, error)
}

func (s *senderFunc) Send(ctx context.Context, req write.Request) (write.Reply, error) {
	s.mu.Lock()
	s.requests = append(s.requests, req)
	s.mu.Unlock()
	if s.fn == nil {
		return write.Reply{}, errors.New("测试没有准备发送应答")
	}
	return s.fn(ctx, req)
}

func (s *senderFunc) snapshot() []write.Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]write.Request(nil), s.requests...)
}

func newTestWriter(t *testing.T, send *senderFunc) *write.Manager {
	t.Helper()
	manager, err := write.New(write.Config{
		Sender:      send,
		Inspector:   stubInspector{},
		SendTimeout: 500 * time.Millisecond,
		Verify: write.VerifyPolicy{
			MaxAttempts: 1,
			Interval:    time.Millisecond,
			Timeout:     200 * time.Millisecond,
		},
	})
	if err != nil {
		t.Fatalf("创建写登记: %v", err)
	}
	t.Cleanup(manager.Close)
	return manager
}

func mustNewManager(t *testing.T, cfg Config) *Manager {
	t.Helper()
	manager, err := New(cfg)
	if err != nil {
		t.Fatalf("创建管理面: %v", err)
	}
	return manager
}

func decodeParams[T any](t *testing.T, raw json.RawMessage) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(raw, &value); err != nil {
		t.Fatalf("解析写意图参数: %v", err)
	}
	return value
}

func rpcError(code int, appError, message string) *protocol.RPCError {
	data, _ := json.Marshal(struct {
		AppError string `json:"appError"`
	}{AppError: appError})
	return &protocol.RPCError{Code: code, Message: message, Data: data}
}

func int64Ptr(value int64) *int64 { return &value }

func float64Ptr(value float64) *float64 { return &value }

func boolPtr(value bool) *bool { return &value }

func stringPtr(value string) *string { return &value }
