package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// appServerHarness 是「连接 + 写登记」集成形状的测试替身：目录读取与写发送都走
// 真实 NDJSON；方法名与 wire 形状只出现在这里，本包不复制它们。
type appServerHarness struct {
	client *protocol.Client
	// renameClient 是改名所用的连接。真实的 session.rename 只在会话流上可用；
	// 集成层据此选择连接，本包不关心。
	renameClient *protocol.Client

	mu    sync.Mutex
	steps []string
}

func (h *appServerHarness) add(step string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.steps = append(h.steps, step)
}

func (h *appServerHarness) stepList() []string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]string(nil), h.steps...)
}

// ReleaseSession 实现目录的订阅释放接口。
func (h *appServerHarness) ReleaseSession(sessionID string) error {
	h.add("release:" + sessionID)
	return nil
}

// catalogWireSession 是 session.catalog 回执里的条目形状。
type catalogWireSession struct {
	SessionID     string  `json:"sessionId"`
	Cwd           string  `json:"cwd"`
	CreatedAt     int64   `json:"createdAt"`
	UpdatedAt     int64   `json:"updatedAt"`
	Running       bool    `json:"running"`
	Title         *string `json:"title"`
	TitleSource   *string `json:"titleSource"`
	TitleRevision *int64  `json:"titleRevision"`
}

// ReadCatalog 读取一页目录，把回执转成目录领域的只读条目。
func (h *appServerHarness) ReadCatalog(ctx context.Context, query Query, cursor string) (Page, error) {
	params := map[string]any{"limit": query.Limit}
	if cursor != "" {
		params["cursor"] = cursor
	}
	if query.Search != "" {
		params["query"] = query.Search
	}
	if query.Workspace != "" {
		params["cwd"] = query.Workspace
	}
	raw, err := h.client.Call(ctx, "session.catalog", params)
	if err != nil {
		return Page{}, err
	}
	var result struct {
		Sessions   []catalogWireSession `json:"sessions"`
		NextCursor *string              `json:"nextCursor"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return Page{}, fmt.Errorf("目录回执无法解析: %w", err)
	}
	page := Page{Entries: make([]Entry, 0, len(result.Sessions))}
	for _, session := range result.Sessions {
		entry := Entry{
			ID:        session.SessionID,
			Workspace: session.Cwd,
			Running:   session.Running,
			CreatedAt: time.UnixMilli(session.CreatedAt).UTC(),
			UpdatedAt: time.UnixMilli(session.UpdatedAt).UTC(),
		}
		if session.Title != nil {
			entry.Title = *session.Title
		}
		if session.TitleSource != nil {
			entry.TitleSource = TitleSource(*session.TitleSource)
		}
		if session.TitleRevision != nil {
			entry.TitleRevision = *session.TitleRevision
		}
		page.Entries = append(page.Entries, entry)
	}
	if result.NextCursor != nil {
		page.NextCursor = *result.NextCursor
	}
	return page, nil
}

// Send 实现写登记的发送方：按目录意图的操作名映射到服务端方法，只调用一次。
func (h *appServerHarness) Send(ctx context.Context, req write.Request) (write.Reply, error) {
	switch req.Kind {
	case write.KindSessionLifecycle:
		var payload struct {
			Op             string `json:"op"`
			Cwd            string `json:"cwd"`
			Title          string `json:"title"`
			ParentBranchID string `json:"parentBranchId"`
			At             int64  `json:"at"`
			Prompt         string `json:"prompt"`
		}
		if err := json.Unmarshal(req.Params, &payload); err != nil {
			return write.Reply{}, err
		}
		switch payload.Op {
		case opCreate:
			h.add("send:create")
			raw, err := h.client.Call(ctx, "session.create", map[string]any{"cwd": payload.Cwd})
			if err != nil {
				return write.Reply{}, err
			}
			return write.Reply{Result: raw}, nil
		case opRename:
			h.add("send:rename:" + req.SessionID)
			params := map[string]any{"sessionId": req.SessionID}
			if payload.Title == "" {
				params["title"] = nil
			} else {
				params["title"] = payload.Title
			}
			client := h.client
			if h.renameClient != nil {
				client = h.renameClient
			}
			raw, err := client.Call(ctx, "session.rename", params)
			if err != nil {
				return write.Reply{}, err
			}
			return write.Reply{Result: raw}, nil
		case opDelete:
			h.add("send:delete:" + req.SessionID)
			if _, err := h.client.Call(ctx, "session.delete", map[string]any{"sessionId": req.SessionID}); err != nil {
				return write.Reply{}, err
			}
			return write.Reply{}, nil
		case opBranchCreate:
			h.add("send:branch_create:" + req.SessionID)
			raw, err := h.client.Call(ctx, "session.branch_create", map[string]any{
				"sessionId":      req.SessionID,
				"parentBranchId": payload.ParentBranchID,
				"at":             payload.At,
				"prompt":         payload.Prompt,
			})
			if err != nil {
				return write.Reply{}, err
			}
			return write.Reply{Result: raw}, nil
		case opBranchEnd:
			h.add("send:branch_end:" + req.BranchID)
			if _, err := h.client.Call(ctx, "session.branch_end", map[string]any{
				"sessionId": req.SessionID,
				"branchId":  req.BranchID,
			}); err != nil {
				return write.Reply{}, err
			}
			return write.Reply{}, nil
		case opFork:
			h.add("send:fork:" + req.SessionID)
			raw, err := h.client.Call(ctx, "session.fork", map[string]any{
				"sessionId": req.SessionID,
				"at":        payload.At,
			})
			if err != nil {
				return write.Reply{}, err
			}
			return write.Reply{Result: raw}, nil
		}
		return write.Reply{}, fmt.Errorf("未知目录意图 %q", payload.Op)
	case write.KindSubmitInput:
		params := map[string]any{}
		if len(req.Params) > 0 {
			if err := json.Unmarshal(req.Params, &params); err != nil {
				return write.Reply{}, err
			}
		}
		params["sessionId"] = req.SessionID
		params["mode"] = string(req.Delivery)
		raw, err := h.client.Call(ctx, "session.submit", params)
		if err != nil {
			return write.Reply{}, err
		}
		var result struct {
			InputID string `json:"inputId"`
		}
		if err := json.Unmarshal(raw, &result); err != nil {
			return write.Reply{}, err
		}
		return write.Reply{InputID: result.InputID, Result: raw}, nil
	}
	return write.Reply{}, fmt.Errorf("未支持的写意图 %s", req.Kind)
}

// startFakeServer 在临时数据目录里启动假服务端并连上控制连接。
func startFakeServer(t *testing.T, script fakeappserver.Script) (*fakeappserver.Server, *protocol.Client) {
	t.Helper()
	server, err := fakeappserver.Start(script)
	if err != nil {
		t.Fatalf("启动假服务端: %v", err)
	}
	supervisor, err := daemon.New(daemon.Config{
		DataDir:        server.DataDir(),
		RequestTimeout: 2 * time.Second,
	})
	if err != nil {
		_ = server.Close()
		t.Fatalf("创建连接监督者: %v", err)
	}
	conn, err := supervisor.Connect(context.Background())
	if err != nil {
		supervisor.Close(nil)
		_ = server.Close()
		t.Fatalf("连接假服务端: %v", err)
	}
	t.Cleanup(func() {
		supervisor.Close(nil)
		_ = server.Close()
	})
	return server, conn.Client
}

// TestAppServerCatalogLifecycle 用假服务端走一遍目录的关键分支：过滤分页参数透传、
// 新建后可提交、改名一致、删除先释放订阅且在确认后移除。
func TestAppServerCatalogLifecycle(t *testing.T) {
	title := "构建失败"
	source := "auto"
	revision := int64(2)
	server, client := startFakeServer(t, fakeappserver.Script{
		Replies: []fakeappserver.Reply{
			{
				Method: "session.catalog",
				Result: map[string]any{
					"sessions": []any{
						map[string]any{
							"sessionId":     "s-1",
							"cwd":           "/work/repo",
							"createdAt":     int64(1700000000000),
							"updatedAt":     int64(1700000500000),
							"running":       false,
							"title":         title,
							"titleSource":   source,
							"titleRevision": revision,
						},
						map[string]any{
							"sessionId": "s-2",
							"cwd":       "/work/other",
							"createdAt": int64(1700000100000),
							"updatedAt": int64(1700000200000),
							"running":   true,
							"title":     nil,
						},
					},
					"nextCursor": nil,
				},
			},
			{Method: "session.create", Result: map[string]any{"sessionId": "s-new-1"}},
			{Method: "session.rename", Result: map[string]any{
				"title":       "新标题",
				"titleSource": "manual",
				"revision":    int64(3),
			}},
			{Method: "session.delete", Result: nil},
			{Method: "session.submit", Result: map[string]any{"inputId": "input-1"}},
		},
	})

	harness := &appServerHarness{client: client}
	archive, err := OpenArchive(filepath.Join(t.TempDir(), "session-archive.json"))
	if err != nil {
		t.Fatalf("打开归档隐藏列表: %v", err)
	}
	catalog := newTestCatalog(t, Config{
		Reader:   harness,
		Writes:   newWriteManager(t, harness),
		Archive:  archive,
		Releaser: harness,
	})

	// 1. 目录读取：过滤与分页参数透传到服务端。
	if _, err := catalog.Refresh(context.Background(), Query{Search: "构建", Workspace: "/work/repo", Limit: 7}); err != nil {
		t.Fatalf("读取目录: %v", err)
	}
	snapshot := catalog.Snapshot()
	mustEqualStrings(t, "目录条目", entryIDs(snapshot.Entries), []string{"s-1", "s-2"})
	first, _ := catalog.Entry("s-1")
	if first.Title != "构建失败" || first.TitleSource != TitleSourceAuto || first.TitleRevision != 2 {
		t.Fatalf("目录标题不符：%+v", first)
	}
	if first.Workspace != "/work/repo" || !first.CreatedAt.Equal(time.UnixMilli(1700000000000).UTC()) {
		t.Fatalf("目录字段不符：%+v", first)
	}
	if _, ok := catalog.Entry("s-2"); !ok {
		t.Fatal("无标题会话没有进入目录")
	}

	catalogRequests := server.RequestsFor("session.catalog")
	if len(catalogRequests) != 1 {
		t.Fatalf("目录请求次数不符：%d", len(catalogRequests))
	}
	var catalogParams map[string]any
	if err := json.Unmarshal(catalogRequests[0].Params, &catalogParams); err != nil {
		t.Fatalf("目录请求参数无法解析: %v", err)
	}
	if catalogParams["limit"] != float64(7) || catalogParams["query"] != "构建" || catalogParams["cwd"] != "/work/repo" {
		t.Fatalf("过滤参数没有透传：%+v", catalogParams)
	}

	// 2. 新建：用返回的标识直接提交。
	created, err := catalog.Create(context.Background(), CreateRequest{Cwd: "/work/new"})
	if err != nil {
		t.Fatalf("新建会话: %v", err)
	}
	if created.ID != "s-new-1" {
		t.Fatalf("新建标识不符：%q", created.ID)
	}
	createRequests := server.RequestsFor("session.create")
	if len(createRequests) != 1 {
		t.Fatalf("新建请求次数不符：%d", len(createRequests))
	}
	var createParams map[string]any
	if err := json.Unmarshal(createRequests[0].Params, &createParams); err != nil {
		t.Fatalf("新建请求参数无法解析: %v", err)
	}
	if createParams["cwd"] != "/work/new" {
		t.Fatalf("新建没有带上工作目录：%+v", createParams)
	}

	submitView, err := harness.clientSubmit(t, catalog, created.ID)
	if err != nil {
		t.Fatalf("向新建会话提交: %v", err)
	}
	if submitView.Status != write.StatusSucceeded {
		t.Fatalf("提交没有成功：%+v", submitView)
	}
	submitRequests := server.RequestsFor("session.submit")
	if len(submitRequests) != 1 {
		t.Fatalf("提交请求次数不符：%d", len(submitRequests))
	}
	var submitParams map[string]any
	if err := json.Unmarshal(submitRequests[0].Params, &submitParams); err != nil {
		t.Fatalf("提交请求参数无法解析: %v", err)
	}
	if submitParams["sessionId"] != created.ID {
		t.Fatalf("提交没有落到新建标识：%+v", submitParams)
	}

	// 3. 改名：再次读取时名称等于新名称。
	renamed, err := catalog.Rename(context.Background(), RenameRequest{SessionID: "s-1", Title: "新标题"})
	if err != nil {
		t.Fatalf("改名: %v", err)
	}
	if renamed.Title != "新标题" || renamed.TitleRevision != 3 || renamed.TitleSource != TitleSourceManual {
		t.Fatalf("改名后的条目不符：%+v", renamed)
	}
	renameRequests := server.RequestsFor("session.rename")
	if len(renameRequests) != 1 {
		t.Fatalf("改名请求次数不符：%d", len(renameRequests))
	}
	var renameParams map[string]any
	if err := json.Unmarshal(renameRequests[0].Params, &renameParams); err != nil {
		t.Fatalf("改名请求参数无法解析: %v", err)
	}
	if renameParams["sessionId"] != "s-1" || renameParams["title"] != "新标题" {
		t.Fatalf("改名请求参数不符：%+v", renameParams)
	}

	// 4. 自动标题通知按修订号合并。
	if !catalog.ApplyTitle(TitleUpdate{SessionID: "s-1", Title: "自动标题", Source: TitleSourceAuto, Revision: 4}) {
		t.Fatal("更高修订号的自动标题没有合并")
	}

	// 5. 删除：先释放订阅，服务端确认后才移除本地行。
	if err := catalog.Delete(context.Background(), DeleteRequest{SessionID: "s-2"}); err != nil {
		t.Fatalf("删除会话: %v", err)
	}
	if _, ok := catalog.Entry("s-2"); ok {
		t.Fatal("删除成功后本地行仍在")
	}
	if _, ok := catalog.Entry("s-1"); !ok {
		t.Fatal("删除一个会话影响了其他行")
	}
	deleteRequests := server.RequestsFor("session.delete")
	if len(deleteRequests) != 1 {
		t.Fatalf("删除请求次数不符：%d", len(deleteRequests))
	}
	var deleteParams map[string]any
	if err := json.Unmarshal(deleteRequests[0].Params, &deleteParams); err != nil {
		t.Fatalf("删除请求参数无法解析: %v", err)
	}
	if deleteParams["sessionId"] != "s-2" {
		t.Fatalf("删除请求参数不符：%+v", deleteParams)
	}
	mustEqualStrings(t, "删除顺序", harness.stepList(), []string{"send:create", "send:rename:s-1", "release:s-2", "send:delete:s-2"})

	mustEqualStrings(t, "删除后的目录", entryIDs(catalog.Snapshot().Entries), []string{"s-1", "s-new-1"})
}

// clientSubmit 用与目录相同的写登记把一个提交意图发到指定会话。
func (h *appServerHarness) clientSubmit(t *testing.T, catalog *Catalog, sessionID string) (write.IntentView, error) {
	t.Helper()
	intent, err := catalog.cfg.Writes.Register(write.Intent{
		Key:       "submit-" + sessionID,
		Kind:      write.KindSubmitInput,
		SessionID: sessionID,
		Params:    json.RawMessage(`{"text":"你好"}`),
	})
	if err != nil {
		return write.IntentView{}, err
	}
	return catalog.cfg.Writes.Wait(context.Background(), intent.ID)
}

// TestAppServerMissingCatalogCapability 确认服务端没有目录能力时报错，而不是空目录。
func TestAppServerMissingCatalogCapability(t *testing.T) {
	_, client := startFakeServer(t, fakeappserver.Script{})
	harness := &appServerHarness{client: client}
	catalog := newTestCatalog(t, Config{
		Reader: harness,
		Writes: newWriteManager(t, harness),
	})

	if _, err := catalog.Refresh(context.Background(), Query{}); err == nil {
		t.Fatal("缺目录能力时应报错")
	} else if errors.Is(err, context.Canceled) {
		t.Fatalf("错误分类不符：%v", err)
	}
	if entries := catalog.Snapshot().Entries; len(entries) != 0 {
		t.Fatalf("缺能力时不应得到条目：%+v", entries)
	}
}
