package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/failure"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// fakeEntryLookup 是单条目录读回的替身。
type fakeEntryLookup struct {
	mu      sync.Mutex
	entries map[string]Entry
	err     error
	calls   []string
}

func (f *fakeEntryLookup) ReadEntry(ctx context.Context, sessionID string) (Entry, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, sessionID)
	if f.err != nil {
		return Entry{}, false, f.err
	}
	entry, ok := f.entries[sessionID]
	return entry, ok, nil
}

func (f *fakeEntryLookup) callList() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...)
}

// forkSender 只回执分叉请求。
func forkSender(result string) *recordSender {
	return &recordSender{handle: func(req write.Request) (write.Reply, error) {
		var head struct {
			Op string `json:"op"`
		}
		if err := json.Unmarshal(req.Params, &head); err != nil {
			return write.Reply{}, err
		}
		if head.Op != opFork {
			return write.Reply{}, fmt.Errorf("意外意图 %q", head.Op)
		}
		return write.Reply{Result: json.RawMessage(result)}, nil
	}}
}

// newForkCatalog 建一个带两条条目与可选读回的目录。
func newForkCatalog(t *testing.T, sender *recordSender, lookup EntryReader) *Catalog {
	t.Helper()
	reader := &fakeReader{}
	reader.setPage("", Page{Entries: []Entry{
		entry("s1", "源会话", 1),
		entry("s2", "另一个会话", 1),
	}})
	return newTestCatalog(t, Config{
		Reader: reader,
		Writes: newWriteManager(t, sender),
		Lookup: lookup,
	})
}

// TestForkAddsNewSessionAndReadsBackOneRow 覆盖：分叉成功只并入新会话、单条读回
// 补齐标题与来源、已有条目与目录读取都不被重建。
func TestForkAddsNewSessionAndReadsBackOneRow(t *testing.T) {
	sender := forkSender(`{"sessionId":"s-fork","sequence":9}`)
	lookup := &fakeEntryLookup{entries: map[string]Entry{
		"s-fork": {ID: "s-fork", Title: "分叉标题", TitleSource: TitleSourceManual, TitleRevision: 2, Workspace: "/w"},
	}}
	reader := &fakeReader{}
	reader.setPage("", Page{Entries: []Entry{
		entry("s1", "源会话", 1),
		entry("s2", "另一个会话", 1),
	}})
	catalog := newTestCatalog(t, Config{
		Reader: reader,
		Writes: newWriteManager(t, sender),
		Lookup: lookup,
	})
	if _, err := catalog.Refresh(context.Background(), Query{}); err != nil {
		t.Fatalf("初始化目录: %v", err)
	}
	before := catalog.Snapshot()
	readsBefore := len(reader.callList())

	forked, err := catalog.Fork(context.Background(), ForkRequest{SessionID: "s1", At: 7})
	if err != nil {
		t.Fatalf("分叉: %v", err)
	}
	if forked.ID != "s-fork" || forked.Title != "分叉标题" ||
		forked.TitleSource != TitleSourceManual || forked.TitleRevision != 2 || forked.Incomplete {
		t.Fatalf("读回后的分叉条目不符：%+v", forked)
	}

	after := catalog.Snapshot()
	if len(after.Entries) != 3 {
		t.Fatalf("目录条目数不符：%+v", after.Entries)
	}
	if !reflect.DeepEqual(after.Entries[0], before.Entries[0]) ||
		!reflect.DeepEqual(after.Entries[1], before.Entries[1]) {
		t.Fatalf("已有条目被改动：%+v → %+v", before.Entries, after.Entries)
	}
	if after.Entries[2].ID != "s-fork" {
		t.Fatalf("分叉会话没有追加：%+v", after.Entries)
	}
	if got := len(reader.callList()); got != readsBefore {
		t.Fatalf("分叉不应重读整个目录：%d → %d", readsBefore, got)
	}
	mustEqualStrings(t, "单条读回", lookup.callList(), []string{"s-fork"})

	calls := sender.requestList()
	if len(calls) != 1 {
		t.Fatalf("分叉应只发一次：%d", len(calls))
	}
	request := calls[0]
	if request.Kind != write.KindSessionLifecycle || request.SessionID != "s1" {
		t.Fatalf("分叉意图的归属不符：%+v", request)
	}
	var payload struct {
		Op string `json:"op"`
		At int64  `json:"at"`
	}
	if err := json.Unmarshal(request.Params, &payload); err != nil {
		t.Fatalf("分叉参数无法解析: %v", err)
	}
	if payload.Op != opFork || payload.At != 7 {
		t.Fatalf("分叉参数不符：%+v", payload)
	}
}

// TestForkReadBackFailureKeepsIncompleteRow 覆盖：读回失败不阻塞分叉，保留标识行。
func TestForkReadBackFailureKeepsIncompleteRow(t *testing.T) {
	for name, lookup := range map[string]EntryReader{
		"读回失败":  &fakeEntryLookup{err: errTest},
		"读回未命中": &fakeEntryLookup{entries: map[string]Entry{}},
		"未配置读回": nil,
	} {
		t.Run(name, func(t *testing.T) {
			catalog := newForkCatalog(t, forkSender(`{"sessionId":"s-fork","sequence":9}`), lookup)
			if _, err := catalog.Refresh(context.Background(), Query{}); err != nil {
				t.Fatalf("初始化目录: %v", err)
			}
			forked, err := catalog.Fork(context.Background(), ForkRequest{SessionID: "s1", At: 7})
			if err != nil {
				t.Fatalf("分叉: %v", err)
			}
			if forked.ID != "s-fork" || !forked.Incomplete {
				t.Fatalf("回执只有标识时应标记待补齐：%+v", forked)
			}
			if _, ok := catalog.Entry("s-fork"); !ok {
				t.Fatal("分叉会话没有保留标识行")
			}
		})
	}
}

// TestForkFailureKeepsCatalog 覆盖：分叉失败保留原目录并给原因。
func TestForkFailureKeepsCatalog(t *testing.T) {
	sender := &recordSender{handle: func(req write.Request) (write.Reply, error) {
		return write.Reply{}, errors.New("源会话不可分叉")
	}}
	lookup := &fakeEntryLookup{entries: map[string]Entry{}}
	catalog := newForkCatalog(t, sender, lookup)
	if _, err := catalog.Refresh(context.Background(), Query{}); err != nil {
		t.Fatalf("初始化目录: %v", err)
	}
	before := catalog.Snapshot()

	_, err := catalog.Fork(context.Background(), ForkRequest{SessionID: "s1", At: 7})
	if !errors.Is(err, ErrRejected) {
		t.Fatalf("分叉失败应返回拒绝错误：%v", err)
	}
	var rejected *RejectedError
	if !errors.As(err, &rejected) || rejected.Op != OpFork {
		t.Fatalf("拒绝原因不完整：%+v", err)
	}
	if !strings.Contains(err.Error(), "源会话不可分叉") {
		t.Fatalf("拒绝错误没有带回原因：%v", err)
	}
	if after := catalog.Snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatalf("分叉失败改动了目录：%+v → %+v", before, after)
	}
	if calls := lookup.callList(); len(calls) != 0 {
		t.Fatalf("分叉失败不应读回：%v", calls)
	}
}

// TestForkRejectsMissingOrAmbiguousCut 覆盖：切点缺失或同时给两个时在登记前拒绝。
func TestForkRejectsMissingOrAmbiguousCut(t *testing.T) {
	sender := &recordSender{}
	catalog := newForkCatalog(t, sender, nil)
	cases := []ForkRequest{
		{SessionID: "s1"},
		{SessionID: "s1", At: 7, TurnIndex: 1},
		{At: 7},
	}
	for _, request := range cases {
		if _, err := catalog.Fork(context.Background(), request); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("切点不合法应被拒绝：%+v → %v", request, err)
		}
	}
	if calls := sender.requestList(); len(calls) != 0 {
		t.Fatalf("切点不合法不应发请求：%d", len(calls))
	}
}

// TestForkTurnUsesCutReasons 覆盖：按轮次分叉时沿用切点查询的四种原因。
func TestForkTurnUsesCutReasons(t *testing.T) {
	cases := []struct {
		name   string
		point  CutPoint
		want   error
		wantIn string
	}{
		{"可用", CutPoint{Status: CutReady, Sequence: 6}, nil, ""},
		{"未结束", CutPoint{Status: CutNotEnded}, ErrCutNotEnded, "还没结束"},
		{"未加载", CutPoint{Status: CutNotLoaded}, ErrCutNotLoaded, "向前翻页"},
		{"无分支", CutPoint{Status: CutNoBranch}, ErrUnknownBranch, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			source := &fakeBranchSource{points: map[string]CutPoint{"main#2": tc.point}}
			sender := forkSender(`{"sessionId":"s-fork","sequence":9}`)
			catalog := newTestCatalog(t, Config{
				Reader:   &fakeReader{},
				Writes:   newWriteManager(t, sender),
				Branches: source,
			})
			_, err := catalog.Fork(context.Background(), ForkRequest{SessionID: "s1", TurnIndex: 2})
			if tc.want == nil {
				if err != nil {
					t.Fatalf("按轮分叉失败：%v", err)
				}
				calls := sender.requestList()
				if len(calls) != 1 {
					t.Fatalf("分叉应只发一次：%d", len(calls))
				}
				var payload struct {
					At int64 `json:"at"`
				}
				if err := json.Unmarshal(calls[0].Params, &payload); err != nil {
					t.Fatalf("分叉参数无法解析: %v", err)
				}
				if payload.At != 6 {
					t.Fatalf("分叉没有采用轮结束切点：%+v", payload)
				}
				return
			}
			if !errors.Is(err, tc.want) {
				t.Fatalf("切点原因不符：%v", err)
			}
			if tc.wantIn != "" && !strings.Contains(err.Error(), tc.wantIn) {
				t.Fatalf("切点原因文案不符：%v", err)
			}
			if calls := sender.requestList(); len(calls) != 0 {
				t.Fatalf("切点不可用不应发请求：%d", len(calls))
			}
		})
	}
}

// TestForkUnknownResultKeepsCatalog 覆盖：未知结果只发一次、进入需人工确认，
// 不把未确认的新会话加进目录。
func TestForkUnknownResultKeepsCatalog(t *testing.T) {
	sender := &recordSender{handle: func(req write.Request) (write.Reply, error) {
		return write.Reply{}, failure.New(failure.KindTimeout, "session.fork", errors.New("超时"))
	}}
	manager, err := write.New(write.Config{
		Sender:      sender,
		Inspector:   noLookupInspector{},
		SendTimeout: 200 * time.Millisecond,
		Verify:      write.VerifyPolicy{MaxAttempts: 1, Interval: time.Millisecond, Timeout: 50 * time.Millisecond},
	})
	if err != nil {
		t.Fatalf("创建写登记: %v", err)
	}
	t.Cleanup(manager.Close)
	catalog := newTestCatalog(t, Config{
		Reader: &fakeReader{pages: map[string]Page{"": {Entries: []Entry{entry("s1", "源会话", 1)}}}},
		Writes: manager,
	})
	if _, err := catalog.Refresh(context.Background(), Query{}); err != nil {
		t.Fatalf("初始化目录: %v", err)
	}
	before := catalog.Snapshot()

	_, err = catalog.Fork(context.Background(), ForkRequest{SessionID: "s1", At: 7})
	if !errors.Is(err, ErrRejected) {
		t.Fatalf("未知结果应返回拒绝错误：%v", err)
	}
	var rejected *RejectedError
	if !errors.As(err, &rejected) || rejected.Status != write.StatusNeedsManualConfirmation {
		t.Fatalf("未知结果的状态不符：%+v", err)
	}
	if calls := sender.requestList(); len(calls) != 1 {
		t.Fatalf("未知结果不应重发：%d", len(calls))
	}
	if after := catalog.Snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatalf("未知结果改动了目录：%+v → %+v", before, after)
	}
}

// TestForkSuccessWithoutResultKeepsCatalog 覆盖：写登记确认成功但没有回执时，
// 不猜新会话标识，按“需重读目录”返回。
func TestForkSuccessWithoutResultKeepsCatalog(t *testing.T) {
	sender := &recordSender{handle: func(req write.Request) (write.Reply, error) {
		return write.Reply{}, nil
	}}
	catalog := newForkCatalog(t, sender, nil)
	if _, err := catalog.Refresh(context.Background(), Query{}); err != nil {
		t.Fatalf("初始化目录: %v", err)
	}
	before := catalog.Snapshot()

	_, err := catalog.Fork(context.Background(), ForkRequest{SessionID: "s1", At: 7})
	if !errors.Is(err, ErrForkResultUnknown) {
		t.Fatalf("缺回执应报需重读目录：%v", err)
	}
	if after := catalog.Snapshot(); !reflect.DeepEqual(before, after) {
		t.Fatalf("缺回执时改动了目录：%+v → %+v", before, after)
	}
	if calls := sender.requestList(); len(calls) != 1 {
		t.Fatalf("缺回执时不应重发：%d", len(calls))
	}
}
