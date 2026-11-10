package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/failure"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// decodePayload 解出目录意图参数里的 op 与字段。
func decodePayload(t *testing.T, req write.Request) map[string]string {
	t.Helper()
	var payload map[string]string
	if err := json.Unmarshal(req.Params, &payload); err != nil {
		t.Fatalf("意图参数无法解析: %v（%s）", err, req.Params)
	}
	return payload
}

// TestCreateReturnsServerIDAndAcceptsSubmit 确认新建只用服务端返回的会话标识，
// 并且该标识随后可以直接承载提交意图。
func TestCreateReturnsServerIDAndAcceptsSubmit(t *testing.T) {
	sender := &recordSender{handle: func(req write.Request) (write.Reply, error) {
		switch req.Kind {
		case write.KindSessionLifecycle:
			return write.Reply{Result: json.RawMessage(`{"sessionId":"s-new"}`)}, nil
		case write.KindSubmitInput:
			return write.Reply{InputID: "input-1", Result: json.RawMessage(`{"inputId":"input-1"}`)}, nil
		}
		return write.Reply{}, nil
	}}
	writes := newWriteManager(t, sender)

	reader := &fakeReader{}
	reader.setPage("", Page{})
	catalog := newTestCatalog(t, Config{Reader: reader, Writes: writes})

	created, err := catalog.Create(context.Background(), CreateRequest{Cwd: "/work/repo"})
	if err != nil {
		t.Fatalf("新建会话: %v", err)
	}
	if created.ID != "s-new" {
		t.Fatalf("新建返回的标识不符：%q", created.ID)
	}
	// 新建回执只有标识；标题与时间等服务端目录页补齐，不替服务端编造。
	if created.Title != "" || created.TitleRevision != 0 || !created.CreatedAt.IsZero() || !created.UpdatedAt.IsZero() {
		t.Fatalf("新建行不应带未经服务端确认的字段：%+v", created)
	}
	if !created.Incomplete {
		t.Fatal("新建行应标记待补齐")
	}
	if _, ok := catalog.Entry("s-new"); !ok {
		t.Fatal("新建后的会话不在本地目录集合里")
	}

	createCalls := sender.requestList()
	if len(createCalls) != 1 {
		t.Fatalf("新建请求次数不符：%d", len(createCalls))
	}
	if got := decodePayload(t, createCalls[0]); got["op"] != opCreate || got["cwd"] != "/work/repo" {
		t.Fatalf("新建意图参数不符：%v", got)
	}

	// 用新建返回的标识提交，确认该标识就是后续写入的目标。
	intent, err := writes.Register(write.Intent{
		Key:       "submit-after-create",
		Kind:      write.KindSubmitInput,
		SessionID: created.ID,
		Params:    json.RawMessage(`{"text":"你好"}`),
	})
	if err != nil {
		t.Fatalf("登记提交: %v", err)
	}
	settled, err := writes.Wait(context.Background(), intent.ID)
	if err != nil {
		t.Fatalf("等待提交: %v", err)
	}
	if settled.Status != write.StatusSucceeded {
		t.Fatalf("提交没有成功：%+v", settled)
	}

	var submitSessionID string
	for _, call := range sender.requestList() {
		if call.Kind == write.KindSubmitInput {
			submitSessionID = call.SessionID
		}
	}
	if submitSessionID != created.ID {
		t.Fatalf("提交没有落到新建标识：got %q，want %q", submitSessionID, created.ID)
	}
}

// TestCreateWithoutCwdIsRejectedBeforeSending 确认缺工作目录时不登记意图。
func TestCreateWithoutCwdIsRejectedBeforeSending(t *testing.T) {
	sender := &recordSender{}
	catalog := withEmptyCatalog(t, sender)
	if _, err := catalog.Create(context.Background(), CreateRequest{Cwd: "  "}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("缺工作目录应被拒绝：%v", err)
	}
	if calls := sender.requestList(); len(calls) != 0 {
		t.Fatalf("被拒绝的新建不应发出请求：%d", len(calls))
	}
}

// TestTwoCreatesAreIndependent 确认空幂等键的新建各自产生一条意图。
func TestTwoCreatesAreIndependent(t *testing.T) {
	var counter atomic.Int64
	sender := &recordSender{handle: func(req write.Request) (write.Reply, error) {
		n := counter.Add(1)
		result, _ := json.Marshal(createResult{SessionID: fmt.Sprintf("s-%d", n)})
		return write.Reply{Result: result}, nil
	}}
	writes := newWriteManager(t, sender)
	reader := &fakeReader{}
	reader.setPage("", Page{})
	catalog := newTestCatalog(t, Config{Reader: reader, Writes: writes})

	first, err := catalog.Create(context.Background(), CreateRequest{Cwd: "/w"})
	if err != nil {
		t.Fatalf("第一次新建: %v", err)
	}
	second, err := catalog.Create(context.Background(), CreateRequest{Cwd: "/w"})
	if err != nil {
		t.Fatalf("第二次新建: %v", err)
	}
	if first.ID == second.ID {
		t.Fatalf("两次新建返回了同一标识：%q", first.ID)
	}
	calls := sender.requestList()
	if len(calls) != 2 {
		t.Fatalf("两次新建应发出两次请求：%d", len(calls))
	}
	table := writes.Table()
	if len(table) != 2 {
		t.Fatalf("写登记里的新建意图数不符：%d", len(table))
	}
	if table[0].Key == table[1].Key {
		t.Fatalf("两次新建复用了幂等键：%q", table[0].Key)
	}
	if table[0].Key == "" || table[1].Key == "" {
		t.Fatal("新建意图必须有幂等键")
	}
}

// TestCreateRejectedAddsNoRow 确认新建被拒时不产生本地行，并带回原因。
func TestCreateRejectedAddsNoRow(t *testing.T) {
	sender := &recordSender{handle: func(req write.Request) (write.Reply, error) {
		return write.Reply{}, errTest
	}}
	catalog := withEmptyCatalog(t, sender)

	_, err := catalog.Create(context.Background(), CreateRequest{Cwd: "/w"})
	if !errors.Is(err, ErrRejected) {
		t.Fatalf("新建被拒应返回拒绝错误：%v", err)
	}
	var rejected *RejectedError
	if !errors.As(err, &rejected) || rejected.Op != OpCreate || rejected.Status != write.StatusFailed {
		t.Fatalf("拒绝原因不完整：%+v", err)
	}
	if _, ok := catalog.Entry("s-new"); ok {
		t.Fatal("被拒的新建不应产生本地行")
	}
	if entries := catalog.Snapshot().Entries; len(entries) != 0 {
		t.Fatalf("被拒的新建不应改变目录：%+v", entries)
	}
}

// TestRenameAppliesCommittedTitleAndReportsConflict 覆盖：改名后一致、重复改名不再发送、
// 标题按修订号合并、回执修订号落后时提示冲突且本地不变、冲突后可重读对齐。
func TestRenameAppliesCommittedTitleAndReportsConflict(t *testing.T) {
	sender := &recordSender{handle: func(req write.Request) (write.Reply, error) {
		var payload struct {
			Title string `json:"title"`
		}
		if err := json.Unmarshal(req.Params, &payload); err != nil {
			return write.Reply{}, err
		}
		result := renameResult{Title: &payload.Title, TitleSource: string(TitleSourceManual), Revision: 2}
		raw, _ := json.Marshal(result)
		return write.Reply{Result: raw}, nil
	}}
	writes := newWriteManager(t, sender)
	reader := &fakeReader{}
	catalog := seedCatalog(t, reader, writes, entry("s1", "旧标题", 1))

	updated, err := catalog.Rename(context.Background(), RenameRequest{SessionID: "s1", Title: "新标题"})
	if err != nil {
		t.Fatalf("改名: %v", err)
	}
	if updated.Title != "新标题" || updated.TitleSource != TitleSourceManual || updated.TitleRevision != 2 {
		t.Fatalf("改名后的条目不符：%+v", updated)
	}

	// 完全相同的改名复用幂等键：不再发送，且重放同一修订号结果是空操作。
	sent := len(sender.requestList())
	replayed, err := catalog.Rename(context.Background(), RenameRequest{SessionID: "s1", Title: "新标题"})
	if err != nil {
		t.Fatalf("重复改名: %v", err)
	}
	if replayed.Title != "新标题" || replayed.TitleRevision != 2 {
		t.Fatalf("重复改名后的条目不符：%+v", replayed)
	}
	if got := len(sender.requestList()); got != sent {
		t.Fatalf("完全相同的改名不应再发送：%d → %d", sent, got)
	}

	// 自动标题以更高修订号到达：改名的旧回执不能再覆盖。
	if !catalog.ApplyTitle(TitleUpdate{SessionID: "s1", Title: "自动标题", Source: TitleSourceAuto, Revision: 3}) {
		t.Fatal("自动标题没有合并")
	}
	if _, err := catalog.Rename(context.Background(), RenameRequest{SessionID: "s1", Title: "冲突标题"}); !errors.Is(err, ErrTitleConflict) {
		t.Fatalf("落后修订号应报告冲突：%v", err)
	}
	current, _ := catalog.Entry("s1")
	if current.Title != "自动标题" || current.TitleRevision != 3 {
		t.Fatalf("冲突时本地标题被覆盖：%+v", current)
	}

	// 「可重读」：重取目录后以服务端标题为准。
	reader.setPage("", Page{Entries: []Entry{entry("s1", "服务端最新", 4)}})
	if _, err := catalog.Refresh(context.Background(), Query{}); err != nil {
		t.Fatalf("重读目录: %v", err)
	}
	reloaded, _ := catalog.Entry("s1")
	if reloaded.Title != "服务端最新" || reloaded.TitleRevision != 4 {
		t.Fatalf("重读后标题不符：%+v", reloaded)
	}
}

// TestRenameAndDeleteRequireKnownSession 确认未知会话在发出请求前被拒绝。
func TestRenameAndDeleteRequireKnownSession(t *testing.T) {
	sender := &recordSender{}
	catalog := withEmptyCatalog(t, sender)

	if _, err := catalog.Rename(context.Background(), RenameRequest{SessionID: "missing", Title: "x"}); !errors.Is(err, ErrUnknownSession) {
		t.Fatalf("未知会话改名应被拒绝：%v", err)
	}
	if err := catalog.Delete(context.Background(), DeleteRequest{SessionID: "missing"}); !errors.Is(err, ErrUnknownSession) {
		t.Fatalf("未知会话删除应被拒绝：%v", err)
	}
	if calls := sender.requestList(); len(calls) != 0 {
		t.Fatalf("未知会话不应发出请求：%d", len(calls))
	}
}

// TestDeleteReleasesThenRemovesOnlyOnSuccess 确认顺序为「先释放订阅，再发删除意图」，
// 且成功后才移除本地行。
func TestDeleteReleasesThenRemovesOnlyOnSuccess(t *testing.T) {
	order := &orderLog{}
	sender := &recordSender{order: order}
	releaser := &recordReleaser{order: order}
	writes := newWriteManager(t, sender)
	reader := &fakeReader{}
	catalog := newTestCatalog(t, Config{Reader: reader, Writes: writes, Releaser: releaser})
	seedCatalogEntries(t, catalog, "s1", "s2")

	if err := catalog.Delete(context.Background(), DeleteRequest{SessionID: "s1"}); err != nil {
		t.Fatalf("删除会话: %v", err)
	}
	if _, ok := catalog.Entry("s1"); ok {
		t.Fatal("删除成功后本地行仍在")
	}
	if _, ok := catalog.Entry("s2"); !ok {
		t.Fatal("删除一个会话不应影响其他行")
	}
	mustEqualStrings(t, "释放与发送顺序", order.snapshot(), []string{"release:s1", "send:session_lifecycle:s1"})
	mustEqualStrings(t, "释放的会话", releaser.releasedIDs(), []string{"s1"})

	calls := sender.requestList()
	if len(calls) != 1 {
		t.Fatalf("删除请求次数不符：%d", len(calls))
	}
	if calls[0].SessionID != "s1" || decodePayload(t, calls[0])["op"] != opDelete {
		t.Fatalf("删除意图不符：%+v", calls[0])
	}
}

// TestDeleteRejectedKeepsRowWithReason 确认删除被拒时保留行并带回原因。
func TestDeleteRejectedKeepsRowWithReason(t *testing.T) {
	order := &orderLog{}
	sender := &recordSender{order: order, handle: func(req write.Request) (write.Reply, error) {
		return write.Reply{}, errors.New("会话正在运行，拒绝删除")
	}}
	releaser := &recordReleaser{order: order}
	catalog := newTestCatalog(t, Config{
		Reader:   &fakeReader{pages: map[string]Page{"": {}}},
		Writes:   newWriteManager(t, sender),
		Releaser: releaser,
	})
	seedCatalogEntries(t, catalog, "s1")

	err := catalog.Delete(context.Background(), DeleteRequest{SessionID: "s1"})
	if !errors.Is(err, ErrRejected) {
		t.Fatalf("删除被拒应返回拒绝错误：%v", err)
	}
	if !strings.Contains(err.Error(), "会话正在运行，拒绝删除") {
		t.Fatalf("拒绝错误没有带回原因：%v", err)
	}
	if _, ok := catalog.Entry("s1"); !ok {
		t.Fatal("删除被拒时本地行被移除了")
	}
	mustEqualStrings(t, "释放后再发送", order.snapshot(), []string{"release:s1", "send:session_lifecycle:s1"})
}

// TestDeleteWithUnknownResultKeepsRow 确认结果无法确认（需人工确认）时保留行。
func TestDeleteWithUnknownResultKeepsRow(t *testing.T) {
	order := &orderLog{}
	sender := &recordSender{order: order, handle: func(req write.Request) (write.Reply, error) {
		return write.Reply{}, failure.New(failure.KindTimeout, "session.delete", errors.New("超时"))
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
		Reader:   &fakeReader{pages: map[string]Page{"": {}}},
		Writes:   manager,
		Releaser: &recordReleaser{order: order},
	})
	seedCatalogEntries(t, catalog, "s1")

	err = catalog.Delete(context.Background(), DeleteRequest{SessionID: "s1"})
	if !errors.Is(err, ErrRejected) {
		t.Fatalf("结果未定应返回拒绝错误：%v", err)
	}
	var rejected *RejectedError
	if !errors.As(err, &rejected) || rejected.Status != write.StatusNeedsManualConfirmation {
		t.Fatalf("未定结果的状态不符：%+v", err)
	}
	if !strings.Contains(rejected.Reason, "核对无法确认") {
		t.Fatalf("未定结果没有带回原因：%q", rejected.Reason)
	}
	if !strings.Contains(err.Error(), "需人工确认") {
		t.Fatalf("错误里看不到原因：%v", err)
	}
	if _, ok := catalog.Entry("s1"); !ok {
		t.Fatal("结果未定时本地行被移除了")
	}
}

// TestDeletePendingKeepsRowWithReason 确认等待期间上下文结束、删除意图还没结项时，
// 本地行保留并把未定状态与原因带回。
func TestDeletePendingKeepsRowWithReason(t *testing.T) {
	release := make(chan struct{})
	sender := &recordSender{handle: func(req write.Request) (write.Reply, error) {
		<-release
		return write.Reply{}, nil
	}}
	catalog := newTestCatalog(t, Config{
		Reader:   &fakeReader{pages: map[string]Page{"": {}}},
		Writes:   newWriteManager(t, sender),
		Releaser: &recordReleaser{},
	})
	// 先释放发送方，再关闭写登记（Cleanup 后进先出）。
	t.Cleanup(func() { close(release) })
	seedCatalogEntries(t, catalog, "s1")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := catalog.Delete(ctx, DeleteRequest{SessionID: "s1"})
	var pending *PendingError
	if !errors.As(err, &pending) {
		t.Fatalf("未结项应返回等待错误：%v", err)
	}
	if pending.Op != OpDelete {
		t.Fatalf("等待错误的操作不符：%+v", pending)
	}
	if pending.Status != write.StatusRegistered && pending.Status != write.StatusInFlight {
		t.Fatalf("等待错误的状态不符：%+v", pending)
	}
	if !strings.Contains(err.Error(), "结果未定") {
		t.Fatalf("等待错误没有说明原因：%v", err)
	}
	if _, ok := catalog.Entry("s1"); !ok {
		t.Fatal("删除未结项时本地行被移除了")
	}
}

// TestDeleteReleaseFailureSendsNothing 确认订阅释放失败时不发删除意图。
func TestDeleteReleaseFailureSendsNothing(t *testing.T) {
	sender := &recordSender{}
	catalog := newTestCatalog(t, Config{
		Reader:   &fakeReader{pages: map[string]Page{"": {}}},
		Writes:   newWriteManager(t, sender),
		Releaser: &recordReleaser{err: errors.New("释放失败")},
	})
	seedCatalogEntries(t, catalog, "s1")

	err := catalog.Delete(context.Background(), DeleteRequest{SessionID: "s1"})
	if err == nil || !strings.Contains(err.Error(), "释放会话订阅失败") {
		t.Fatalf("释放失败应阻止删除：%v", err)
	}
	if calls := sender.requestList(); len(calls) != 0 {
		t.Fatalf("释放失败后仍然发出了删除意图：%d", len(calls))
	}
	if _, ok := catalog.Entry("s1"); !ok {
		t.Fatal("释放失败时本地行被移除了")
	}
}

// TestDeleteKeepsArchiveRecordInert 确认删除会话不清理归档记录，但该记录不再产生
// 可见条目，也不阻塞删除。
func TestDeleteKeepsArchiveRecordInert(t *testing.T) {
	archive := newArchive(t)
	catalog := newTestCatalog(t, Config{
		Reader:  &fakeReader{pages: map[string]Page{"": {}}},
		Writes:  newWriteManager(t, &recordSender{}),
		Archive: archive,
	})
	seedCatalogEntries(t, catalog, "s1")
	if err := catalog.Hide("s1"); err != nil {
		t.Fatalf("隐藏会话: %v", err)
	}
	if err := catalog.Delete(context.Background(), DeleteRequest{SessionID: "s1"}); err != nil {
		t.Fatalf("删除隐藏会话: %v", err)
	}
	if _, ok := catalog.Entry("s1"); ok {
		t.Fatal("删除后本地行仍在")
	}
	if !archive.Hidden("s1") {
		t.Fatal("删除不应清理归档记录")
	}
	if entries := catalog.HiddenEntries(); len(entries) != 0 {
		t.Fatalf("归档记录对不存在的条目不产生可见结果：%+v", entries)
	}
}

// withEmptyCatalog 建一个空目录，共享同一个发送方。
func withEmptyCatalog(t *testing.T, sender write.Sender) *Catalog {
	t.Helper()
	reader := &fakeReader{}
	reader.setPage("", Page{})
	return newTestCatalog(t, Config{Reader: reader, Writes: newWriteManager(t, sender)})
}

// seedCatalogEntries 用一页条目填充目录。
func seedCatalogEntries(t *testing.T, catalog *Catalog, ids ...string) {
	t.Helper()
	entries := make([]Entry, 0, len(ids))
	for _, id := range ids {
		entries = append(entries, entry(id, "标题-"+id, 1))
	}
	// 直接放入本地集合：这些用例只关心变更路径，目录读取由其他用例覆盖。
	catalog.mu.Lock()
	for _, item := range entries {
		if _, exists := catalog.index[item.ID]; exists {
			continue
		}
		catalog.index[item.ID] = len(catalog.entries)
		catalog.entries = append(catalog.entries, item)
	}
	catalog.mu.Unlock()
}
