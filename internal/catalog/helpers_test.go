package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/write"
)

// fakeReader 按请求游标返回预先准备的页，并记录每次读取的过滤条件与游标。
type fakeReader struct {
	mu    sync.Mutex
	pages map[string]Page
	errs  map[string]error
	calls []readCall
}

type readCall struct {
	Query  Query
	Cursor string
}

func (r *fakeReader) ReadCatalog(ctx context.Context, query Query, cursor string) (Page, error) {
	r.mu.Lock()
	r.calls = append(r.calls, readCall{Query: query, Cursor: cursor})
	page, ok := r.pages[cursor]
	err := r.errs[cursor]
	r.mu.Unlock()
	if err != nil {
		return Page{}, err
	}
	if !ok {
		return Page{}, fmt.Errorf("未准备的游标 %q", cursor)
	}
	return page, nil
}

func (r *fakeReader) callList() []readCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]readCall(nil), r.calls...)
}

func (r *fakeReader) setPage(cursor string, page Page) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.pages == nil {
		r.pages = make(map[string]Page)
	}
	r.pages[cursor] = page
}

func (r *fakeReader) setError(cursor string, err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.errs == nil {
		r.errs = make(map[string]error)
	}
	r.errs[cursor] = err
}

// recordSender 是写登记的发送方替身：记录每次发送，并按 handle 回执。
type recordSender struct {
	mu     sync.Mutex
	calls  []write.Request
	order  *orderLog
	handle func(write.Request) (write.Reply, error)
}

func (s *recordSender) Send(ctx context.Context, req write.Request) (write.Reply, error) {
	req.Params = append(json.RawMessage(nil), req.Params...)
	s.mu.Lock()
	s.calls = append(s.calls, req)
	handle := s.handle
	order := s.order
	s.mu.Unlock()
	if order != nil {
		order.add("send:" + string(req.Kind) + ":" + req.SessionID)
	}
	if handle == nil {
		return write.Reply{}, nil
	}
	return handle(req)
}

func (s *recordSender) requestList() []write.Request {
	s.mu.Lock()
	defer s.mu.Unlock()
	calls := make([]write.Request, len(s.calls))
	for i, call := range s.calls {
		call.Params = append(json.RawMessage(nil), call.Params...)
		calls[i] = call
	}
	return calls
}

// noLookupInspector 是核对需要的占位实现：总是无法确认。
type noLookupInspector struct{}

func (noLookupInspector) LookupInput(ctx context.Context, query write.InputQuery) (write.InputState, error) {
	return write.InputState{}, nil
}

// orderLog 记录本地动作顺序，用来核对「先释放订阅，再发删除意图」。
type orderLog struct {
	mu    sync.Mutex
	steps []string
}

func (o *orderLog) add(step string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.steps = append(o.steps, step)
}

func (o *orderLog) snapshot() []string {
	o.mu.Lock()
	defer o.mu.Unlock()
	return append([]string(nil), o.steps...)
}

// recordReleaser 是订阅释放的替身。
type recordReleaser struct {
	order *orderLog
	err   error

	mu       sync.Mutex
	released []string
}

func (r *recordReleaser) ReleaseSession(sessionID string) error {
	r.mu.Lock()
	r.released = append(r.released, sessionID)
	r.mu.Unlock()
	if r.order != nil {
		r.order.add("release:" + sessionID)
	}
	return r.err
}

func (r *recordReleaser) releasedIDs() []string {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]string(nil), r.released...)
}

// newWriteManager 建一个用于测试的写登记：核对为占位实现，间隔很短。
func newWriteManager(t *testing.T, sender write.Sender) *write.Manager {
	t.Helper()
	manager, err := write.New(write.Config{
		Sender:      sender,
		Inspector:   noLookupInspector{},
		SendTimeout: 2 * time.Second,
		Verify: write.VerifyPolicy{
			MaxAttempts: 1,
			Interval:    time.Millisecond,
			Timeout:     time.Second,
		},
	})
	if err != nil {
		t.Fatalf("创建写登记: %v", err)
	}
	t.Cleanup(manager.Close)
	return manager
}

// newTestCatalog 建一个目录；Reader 与 Writes 必填，其余按调用方给。
func newTestCatalog(t *testing.T, cfg Config) *Catalog {
	t.Helper()
	catalog, err := New(cfg)
	if err != nil {
		t.Fatalf("创建目录: %v", err)
	}
	return catalog
}

// entry 造一条目录条目。
func entry(id, title string, revision int64) Entry {
	return Entry{
		ID:            id,
		Title:         title,
		TitleRevision: revision,
		Workspace:     "/w",
	}
}

// entryIDs 取条目标识序列。
func entryIDs(entries []Entry) []string {
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}
	return ids
}

// seedCatalog 用一页条目初始化目录。
func seedCatalog(t *testing.T, reader *fakeReader, writes *write.Manager, entries ...Entry) *Catalog {
	t.Helper()
	reader.setPage("", Page{Entries: entries})
	catalog := newTestCatalog(t, Config{Reader: reader, Writes: writes})
	if _, err := catalog.Refresh(context.Background(), Query{}); err != nil {
		t.Fatalf("初始化目录: %v", err)
	}
	return catalog
}

// mustEqualStrings 比较两个字符串序列。
func mustEqualStrings(t *testing.T, label string, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s 长度不符：got %v，want %v", label, got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("%s 第 %d 项不符：got %q，want %q（整体 %v）", label, i, got[i], want[i], got)
		}
	}
}

// newArchive 在临时目录里打开一个归档隐藏列表。
func newArchive(t *testing.T) *Archive {
	t.Helper()
	archive, err := OpenArchive(t.TempDir() + "/session-archive.json")
	if err != nil {
		t.Fatalf("打开归档隐藏列表: %v", err)
	}
	return archive
}

// errTest 是一个普通的服务端拒绝错误。
var errTest = errors.New("服务端拒绝")
