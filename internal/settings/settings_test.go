package settings

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/write"
)

func executionSources() Sources {
	return Sources{
		Dialect: "json",
		Sources: []Source{
			{
				Name:     "execution",
				Scopes:   []ScopeKind{ScopeUser, ScopeProject},
				UserOnly: []string{"font"},
				Schema:   json.RawMessage(`{"type":"object"}`),
			},
			{
				Name:      "plugins",
				Scopes:    []ScopeKind{ScopeUser},
				WriteOnly: false,
			},
		},
	}
}

func TestRefreshSourcesReadsAndCaches(t *testing.T) {
	reader := &stubSources{value: executionSources()}
	manager := mustNewManager(t, Config{Sources: reader})

	if _, ok := manager.Sources(); ok {
		t.Fatal("未读取时不应有来源缓存")
	}
	sources, err := manager.RefreshSources(context.Background())
	if err != nil {
		t.Fatalf("读取来源: %v", err)
	}
	if len(sources.Sources) != 2 || sources.Sources[0].Name != "execution" {
		t.Fatalf("来源清单不符：%+v", sources)
	}
	if reader.calls != 1 {
		t.Fatalf("来源读取次数不符：%d", reader.calls)
	}

	// 返回值是副本：改调用方手里的切片不影响缓存。
	sources.Sources[0].Scopes[0] = ScopeProject
	cached, ok := manager.Sources()
	if !ok || cached.Sources[0].Scopes[0] != ScopeUser {
		t.Fatalf("缓存被调用方改动了：%+v", cached)
	}
}

func TestRefreshSourcesRejectsMalformed(t *testing.T) {
	src := func(name string, scopes ...ScopeKind) Source {
		return Source{Name: name, Scopes: scopes}
	}
	cases := []struct {
		name    string
		sources Sources
	}{
		{"缺名称", Sources{Sources: []Source{src("", ScopeUser)}}},
		{"重名", Sources{Sources: []Source{src("a", ScopeUser), src("a", ScopeUser)}}},
		{"未知作用域", Sources{Sources: []Source{src("a", ScopeKind("global"))}}},
		{"重复作用域", Sources{Sources: []Source{src("a", ScopeUser, ScopeUser)}}},
		{"空只读键", Sources{Sources: []Source{{Name: "a", Scopes: []ScopeKind{ScopeUser}, UserOnly: []string{" "}}}}},
		{"重复只读键", Sources{Sources: []Source{{Name: "a", Scopes: []ScopeKind{ScopeUser}, UserOnly: []string{"font", "font"}}}}},
		{"结构描述非法", Sources{Sources: []Source{{Name: "a", Scopes: []ScopeKind{ScopeUser}, Schema: json.RawMessage(`{`)}}}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			manager := mustNewManager(t, Config{Sources: &stubSources{value: testCase.sources}})
			_, err := manager.RefreshSources(context.Background())
			if !errors.Is(err, ErrMalformedSources) {
				t.Fatalf("应拒绝不合法来源：%v", err)
			}
		})
	}
}

func TestRefreshSourcesCapabilityAndReadFailure(t *testing.T) {
	manager := mustNewManager(t, Config{Features: stubFeatures{}})
	if _, err := manager.RefreshSources(context.Background()); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("缺能力应分类：%v", err)
	}

	manager = mustNewManager(t, Config{Sources: &stubSources{err: errors.New("连接中断")}})
	_, err := manager.RefreshSources(context.Background())
	if !errors.Is(err, ErrSourcesRead) {
		t.Fatalf("读取失败应分类：%v", err)
	}
	if errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("普通读取失败不应归为能力缺失：%v", err)
	}

	manager = mustNewManager(t, Config{Sources: &stubSources{err: rpcError(-32601, "unknown_method", "unknown method settings.sources")}})
	if _, err := manager.RefreshSources(context.Background()); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("unknown_method 应归为能力缺失：%v", err)
	}
	if _, err := manager.RefreshSources(context.Background()); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("重复读取仍应分类：%v", err)
	}
}

func TestRefreshProjectsAndResolution(t *testing.T) {
	reader := &stubProjects{value: []Project{{Root: 7, CWD: "/workspace/project"}}}
	manager := mustNewManager(t, Config{Projects: reader})

	if _, err := manager.ResolveProject(context.Background(), "/workspace/project"); err != nil {
		t.Fatalf("解析项目目录: %v", err)
	}
	if reader.calls != 1 {
		t.Fatalf("尚未读取时应只读一次：%d", reader.calls)
	}
	if _, err := manager.ResolveProject(context.Background(), "/workspace/project"); err != nil {
		t.Fatalf("再次解析: %v", err)
	}
	if reader.calls != 1 {
		t.Fatalf("已读取后不应重复请求：%d", reader.calls)
	}
	if _, err := manager.ResolveProject(context.Background(), "/workspace/other"); !errors.Is(err, ErrProjectNotAuthorized) {
		t.Fatalf("未授权目录应拒绝：%v", err)
	}
	if _, err := manager.ResolveProject(context.Background(), "  "); !errors.Is(err, ErrProjectNotAuthorized) {
		t.Fatalf("空目录应拒绝：%v", err)
	}
	projects, ok := manager.Projects()
	if !ok || len(projects) != 1 || projects[0].Root != 7 {
		t.Fatalf("项目缓存不符：%+v", projects)
	}
}

func TestRefreshProjectsRejectsMalformed(t *testing.T) {
	cases := []struct {
		name     string
		projects []Project
	}{
		{"根编号为零", []Project{{Root: 0, CWD: "/a"}}},
		{"根编号重复", []Project{{Root: 1, CWD: "/a"}, {Root: 1, CWD: "/b"}}},
		{"目录为空", []Project{{Root: 1, CWD: " "}}},
		{"目录重复", []Project{{Root: 1, CWD: "/a"}, {Root: 2, CWD: "/a"}}},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			manager := mustNewManager(t, Config{Projects: &stubProjects{value: testCase.projects}})
			if _, err := manager.RefreshProjects(context.Background()); !errors.Is(err, ErrMalformedProjects) {
				t.Fatalf("应拒绝不合法项目根：%v", err)
			}
		})
	}
}

func TestReadRequiresSourcesAndDeclaredScope(t *testing.T) {
	manager := mustNewManager(t, Config{
		Settings: &stubSettings{documents: map[string]Document{
			UserScope().cacheKey("execution"): {Revision: "r1", Document: json.RawMessage(`{}`)},
		}},
	})
	if _, err := manager.Read(context.Background(), "execution", UserScope()); !errors.Is(err, ErrSourcesNotLoaded) {
		t.Fatalf("未读来源前应拒绝：%v", err)
	}
	if _, err := manager.RefreshSources(context.Background()); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("缺来源读取接口应分类：%v", err)
	}

	manager = mustNewManager(t, Config{
		Sources: &stubSources{value: executionSources()},
		Settings: &stubSettings{documents: map[string]Document{
			UserScope().cacheKey("execution"): {Revision: "r1", Document: json.RawMessage(`{}`)},
		}},
	})
	if _, err := manager.RefreshSources(context.Background()); err != nil {
		t.Fatalf("读取来源: %v", err)
	}
	if _, err := manager.Read(context.Background(), "plugins", ProjectScope(7)); !errors.Is(err, ErrScopeNotAllowed) {
		t.Fatalf("用户作用域来源不应支持项目作用域：%v", err)
	}
	if _, err := manager.Read(context.Background(), "missing", UserScope()); !errors.Is(err, ErrInvalidSource) {
		t.Fatalf("未声明来源应拒绝：%v", err)
	}
	if _, err := manager.Read(context.Background(), "execution", Scope{Kind: "tenant"}); !errors.Is(err, ErrInvalidScope) {
		t.Fatalf("未知作用域应拒绝：%v", err)
	}
	if _, err := manager.Read(context.Background(), "execution", ProjectScope(7)); !errors.Is(err, ErrProjectsNotLoaded) {
		t.Fatalf("项目作用域要求项目根已读取：%v", err)
	}
}

func TestReadProjectScopeRequiresAuthorizedRoot(t *testing.T) {
	manager := mustNewManager(t, Config{
		Sources:  &stubSources{value: executionSources()},
		Projects: &stubProjects{value: []Project{{Root: 7, CWD: "/workspace/project"}}},
		Settings: &stubSettings{documents: map[string]Document{
			ProjectScope(7).cacheKey("execution"): {Revision: "r1", Document: json.RawMessage(`{"sandbox":true}`)},
		}},
	})
	if _, err := manager.RefreshSources(context.Background()); err != nil {
		t.Fatalf("读取来源: %v", err)
	}
	if _, err := manager.RefreshProjects(context.Background()); err != nil {
		t.Fatalf("读取项目根: %v", err)
	}
	document, err := manager.Read(context.Background(), "execution", ProjectScope(7))
	if err != nil {
		t.Fatalf("读取项目设置: %v", err)
	}
	if document.Scope.Project != 7 || document.Revision != "r1" {
		t.Fatalf("项目设置文档不符：%+v", document)
	}
	if _, err := manager.Read(context.Background(), "execution", ProjectScope(9)); !errors.Is(err, ErrProjectNotAuthorized) {
		t.Fatalf("未授权项目根应拒绝：%v", err)
	}
}

func TestReadRejectsMalformedDocument(t *testing.T) {
	manager := mustNewManager(t, Config{
		Sources: &stubSources{value: Sources{Sources: []Source{{Name: "a", Scopes: []ScopeKind{ScopeUser}}}}},
		Settings: &stubSettings{read: func(context.Context, string, Scope) (Document, error) {
			return Document{Revision: "", Document: json.RawMessage(`{}`)}, nil
		}},
	})
	if _, err := manager.RefreshSources(context.Background()); err != nil {
		t.Fatalf("读取来源: %v", err)
	}
	if _, err := manager.Read(context.Background(), "a", UserScope()); !errors.Is(err, ErrMalformedDocument) {
		t.Fatalf("缺版本应拒绝：%v", err)
	}

	manager = mustNewManager(t, Config{
		Sources: &stubSources{value: Sources{Sources: []Source{{Name: "a", Scopes: []ScopeKind{ScopeUser}}}}},
		Settings: &stubSettings{read: func(context.Context, string, Scope) (Document, error) {
			return Document{Revision: "r1", Document: json.RawMessage(`[]`)}, nil
		}},
	})
	if _, err := manager.RefreshSources(context.Background()); err != nil {
		t.Fatalf("读取来源: %v", err)
	}
	if _, err := manager.Read(context.Background(), "a", UserScope()); !errors.Is(err, ErrMalformedDocument) {
		t.Fatalf("非对象文档应拒绝：%v", err)
	}
}

func TestUpdateWritesOnceThroughManagementWrite(t *testing.T) {
	sender := &senderFunc{fn: func(context.Context, write.Request) (write.Reply, error) {
		return write.Reply{Result: json.RawMessage(`{"revision":"r2","document":{"sandbox":false}}`)}, nil
	}}
	writer := newTestWriter(t, sender)
	manager := mustNewManager(t, Config{
		Features: allFeatures(),
		Sources:  &stubSources{value: executionSources()},
		Writer:   writer,
	})
	if _, err := manager.RefreshSources(context.Background()); err != nil {
		t.Fatalf("读取来源: %v", err)
	}

	document, err := manager.Update(context.Background(), UpdateRequest{
		Source:           "execution",
		Scope:            UserScope(),
		ExpectedRevision: "r1",
		Document:         json.RawMessage(`{"sandbox":false}`),
	})
	if err != nil {
		t.Fatalf("写入设置: %v", err)
	}
	if document.Revision != "r2" || !strings.Contains(string(document.Document), `"sandbox":false`) {
		t.Fatalf("写入回执不符：%+v", document)
	}
	requests := sender.snapshot()
	if len(requests) != 1 {
		t.Fatalf("写入应只发一次：%d", len(requests))
	}
	if requests[0].Kind != write.KindManagementWrite {
		t.Fatalf("写入应经管理写入类型：%q", requests[0].Kind)
	}
	params := decodeParams[settingsUpdateParams](t, requests[0].Params)
	if params.Op != "settings.update" || params.Source != "execution" || params.ExpectedRevision != "r1" {
		t.Fatalf("写入参数不符：%+v", params)
	}
	if params.Scope != UserScope() {
		t.Fatalf("写入作用域不符：%+v", params.Scope)
	}

	cached, ok := manager.Cached("execution", UserScope())
	if !ok || cached.Revision != "r2" {
		t.Fatalf("成功写入应更新缓存：%+v", cached)
	}
}

func TestUpdateLocalRevisionConflictDoesNotWrite(t *testing.T) {
	sender := &senderFunc{}
	writer := newTestWriter(t, sender)
	reader := &stubSettings{documents: map[string]Document{
		UserScope().cacheKey("execution"): {Revision: "r2", Document: json.RawMessage(`{"sandbox":false}`)},
	}}
	manager := mustNewManager(t, Config{
		Features: allFeatures(),
		Sources:  &stubSources{value: executionSources()},
		Settings: reader,
		Writer:   writer,
	})
	if _, err := manager.RefreshSources(context.Background()); err != nil {
		t.Fatalf("读取来源: %v", err)
	}
	if _, err := manager.Read(context.Background(), "execution", UserScope()); err != nil {
		t.Fatalf("读取设置: %v", err)
	}

	_, err := manager.Update(context.Background(), UpdateRequest{
		Source:           "execution",
		Scope:            UserScope(),
		ExpectedRevision: "r1",
		Document:         json.RawMessage(`{"sandbox":true}`),
	})
	if !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("本地版本已变应返回冲突：%v", err)
	}
	if len(sender.snapshot()) != 0 {
		t.Fatal("冲突时不应发出写请求")
	}
	if cached, _ := manager.Cached("execution", UserScope()); cached.Revision != "r2" {
		t.Fatalf("冲突不应覆盖缓存：%+v", cached)
	}
}

func TestUpdateServerConflictIsNotRetried(t *testing.T) {
	sender := &senderFunc{fn: func(context.Context, write.Request) (write.Reply, error) {
		return write.Reply{}, rpcError(-32000, "revision_conflict", "设置版本已变")
	}}
	writer := newTestWriter(t, sender)
	reader := &stubSettings{documents: map[string]Document{
		UserScope().cacheKey("execution"): {Revision: "r2", Document: json.RawMessage(`{"sandbox":true}`)},
	}}
	manager := mustNewManager(t, Config{
		Features: allFeatures(),
		Sources:  &stubSources{value: executionSources()},
		Settings: reader,
		Writer:   writer,
	})
	if _, err := manager.RefreshSources(context.Background()); err != nil {
		t.Fatalf("读取来源: %v", err)
	}

	_, err := manager.Update(context.Background(), UpdateRequest{
		Source:           "execution",
		Scope:            UserScope(),
		ExpectedRevision: "r1",
		Document:         json.RawMessage(`{"sandbox":false}`),
	})
	if !errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("服务端拒绝且版本已变应返回冲突：%v", err)
	}
	requests := sender.snapshot()
	if len(requests) != 1 {
		t.Fatalf("冲突不应重发：%d", len(requests))
	}
	if reader.calls != 1 {
		t.Fatalf("冲突判定只应读一次当前文档：%d", reader.calls)
	}
}

func TestUpdateFailureWithoutRevisionChange(t *testing.T) {
	sender := &senderFunc{fn: func(context.Context, write.Request) (write.Reply, error) {
		return write.Reply{}, errors.New("参数错误")
	}}
	writer := newTestWriter(t, sender)
	reader := &stubSettings{documents: map[string]Document{
		UserScope().cacheKey("execution"): {Revision: "r1", Document: json.RawMessage(`{"sandbox":true}`)},
	}}
	manager := mustNewManager(t, Config{
		Features: allFeatures(),
		Sources:  &stubSources{value: executionSources()},
		Settings: reader,
		Writer:   writer,
	})
	if _, err := manager.RefreshSources(context.Background()); err != nil {
		t.Fatalf("读取来源: %v", err)
	}

	_, err := manager.Update(context.Background(), UpdateRequest{
		Source:           "execution",
		Scope:            UserScope(),
		ExpectedRevision: "r1",
		Document:         json.RawMessage(`{"sandbox":false}`),
	})
	if !errors.Is(err, ErrUpdateFailed) {
		t.Fatalf("版本未变时的失败应归写入失败：%v", err)
	}
	if errors.Is(err, ErrRevisionConflict) {
		t.Fatalf("版本未变不应报冲突：%v", err)
	}
	if len(sender.snapshot()) != 1 {
		t.Fatal("失败不应重发")
	}
}

func TestUpdateRequiresPreconditionAndRejectsUserOnlyKeys(t *testing.T) {
	sender := &senderFunc{}
	writer := newTestWriter(t, sender)
	manager := mustNewManager(t, Config{
		Features: allFeatures(),
		Sources:  &stubSources{value: executionSources()},
		Projects: &stubProjects{value: []Project{{Root: 7, CWD: "/workspace/project"}}},
		Writer:   writer,
	})
	if _, err := manager.RefreshSources(context.Background()); err != nil {
		t.Fatalf("读取来源: %v", err)
	}
	if _, err := manager.RefreshProjects(context.Background()); err != nil {
		t.Fatalf("读取项目根: %v", err)
	}

	if _, err := manager.Update(context.Background(), UpdateRequest{
		Source:   "execution",
		Scope:    UserScope(),
		Document: json.RawMessage(`{}`),
	}); !errors.Is(err, ErrRevisionMissing) {
		t.Fatalf("缺版本前置条件应拒绝：%v", err)
	}
	if _, err := manager.Update(context.Background(), UpdateRequest{
		Source:           "execution",
		Scope:            ProjectScope(7),
		ExpectedRevision: "r1",
		Document:         json.RawMessage(`{"font":"mono"}`),
	}); !errors.Is(err, ErrUserOnlySetting) {
		t.Fatalf("项目作用域写只允许用户作用域的键应拒绝：%v", err)
	}
	if len(sender.snapshot()) != 0 {
		t.Fatal("被前置条件拒绝时不应发出写请求")
	}
}

func TestUpdateWithoutWriterIsCapabilityStyle(t *testing.T) {
	manager := mustNewManager(t, Config{
		Features: allFeatures(),
		Sources:  &stubSources{value: executionSources()},
	})
	if _, err := manager.RefreshSources(context.Background()); err != nil {
		t.Fatalf("读取来源: %v", err)
	}
	_, err := manager.Update(context.Background(), UpdateRequest{
		Source:           "execution",
		Scope:            UserScope(),
		ExpectedRevision: "r1",
		Document:         json.RawMessage(`{}`),
	})
	if !errors.Is(err, ErrWriteUnavailable) {
		t.Fatalf("缺写登记应返回写不可用：%v", err)
	}
}

func TestUpdateMalformedReplyIsNotAdopted(t *testing.T) {
	sender := &senderFunc{fn: func(context.Context, write.Request) (write.Reply, error) {
		return write.Reply{Result: json.RawMessage(`{"ok":true}`)}, nil
	}}
	writer := newTestWriter(t, sender)
	manager := mustNewManager(t, Config{
		Features: allFeatures(),
		Sources:  &stubSources{value: executionSources()},
		Writer:   writer,
	})
	if _, err := manager.RefreshSources(context.Background()); err != nil {
		t.Fatalf("读取来源: %v", err)
	}
	if _, err := manager.Update(context.Background(), UpdateRequest{
		Source:           "execution",
		Scope:            UserScope(),
		ExpectedRevision: "r1",
		Document:         json.RawMessage(`{"sandbox":true}`),
	}); !errors.Is(err, ErrMalformedWriteResult) {
		t.Fatalf("回执不合法应拒绝采用：%v", err)
	}
	if _, ok := manager.Cached("execution", UserScope()); ok {
		t.Fatal("回执不合法时不应写入缓存")
	}
	if len(sender.snapshot()) != 1 {
		t.Fatal("回执不合法不应重发")
	}
}

func TestReadRejectsWriteOnlySource(t *testing.T) {
	manager := mustNewManager(t, Config{
		Sources: &stubSources{value: Sources{Sources: []Source{
			{Name: "journal", Scopes: []ScopeKind{ScopeUser}, WriteOnly: true},
		}}},
		Settings: &stubSettings{},
	})
	if _, err := manager.RefreshSources(context.Background()); err != nil {
		t.Fatalf("读取来源: %v", err)
	}
	if _, err := manager.Read(context.Background(), "journal", UserScope()); !errors.Is(err, ErrSourceWriteOnly) {
		t.Fatalf("只写来源不应可读：%v", err)
	}
}

func TestUpdateRequiresAdvertisedCapability(t *testing.T) {
	sender := &senderFunc{}
	writer := newTestWriter(t, sender)
	manager := mustNewManager(t, Config{
		Features: stubFeatures{FeatureSettingsSources: true, FeatureSettingsRead: true},
		Sources:  &stubSources{value: executionSources()},
		Writer:   writer,
	})
	if _, err := manager.RefreshSources(context.Background()); err != nil {
		t.Fatalf("读取来源: %v", err)
	}
	if _, err := manager.Update(context.Background(), UpdateRequest{
		Source:           "execution",
		Scope:            UserScope(),
		ExpectedRevision: "r1",
		Document:         json.RawMessage(`{}`),
	}); !errors.Is(err, ErrCapabilityMissing) {
		t.Fatalf("未声明写入能力应分类：%v", err)
	}
	if len(sender.snapshot()) != 0 {
		t.Fatal("缺能力时不应发出写请求")
	}
}
