package catalog

import (
	"context"
	"errors"
	"testing"
)

// TestRefreshAndLoadMoreCoverAllPagesWithoutDuplicates 用三页带重叠的目录核对：
// 本地集合按页推进、重复条目去重、无遗漏、顺序与服务端页一致。
func TestRefreshAndLoadMoreCoverAllPagesWithoutDuplicates(t *testing.T) {
	reader := &fakeReader{}
	reader.setPage("", Page{Entries: []Entry{entry("s1", "一", 1), entry("s2", "二", 1)}, NextCursor: "c1"})
	reader.setPage("c1", Page{Entries: []Entry{entry("s2", "二", 1), entry("s3", "三", 1)}, NextCursor: "c2"})
	reader.setPage("c2", Page{Entries: []Entry{entry("s3", "三", 1), entry("s4", "四", 1)}, NextCursor: ""})

	catalog := newTestCatalog(t, Config{
		Reader: reader,
		Writes: newWriteManager(t, &recordSender{}),
	})

	result, err := catalog.Refresh(context.Background(), Query{})
	if err != nil {
		t.Fatalf("读取第一页: %v", err)
	}
	if result.Added != 2 || result.Duplicates != 0 || !result.HasMore {
		t.Fatalf("第一页结果不符：%+v", result)
	}

	result, err = catalog.LoadMore(context.Background())
	if err != nil {
		t.Fatalf("读取第二页: %v", err)
	}
	if result.Added != 1 || result.Duplicates != 1 || !result.HasMore {
		t.Fatalf("第二页结果不符：%+v", result)
	}

	result, err = catalog.LoadMore(context.Background())
	if err != nil {
		t.Fatalf("读取第三页: %v", err)
	}
	if result.Added != 1 || result.Duplicates != 1 || result.HasMore {
		t.Fatalf("第三页结果不符：%+v", result)
	}

	// 没有下一页时再推进是空操作，不会重复请求。
	if _, err := catalog.LoadMore(context.Background()); err != nil {
		t.Fatalf("推进到底后再读: %v", err)
	}

	snapshot := catalog.Snapshot()
	mustEqualStrings(t, "条目顺序", entryIDs(snapshot.Entries), []string{"s1", "s2", "s3", "s4"})
	if snapshot.HasMore {
		t.Fatal("到底后仍报告还有下一页")
	}

	calls := reader.callList()
	cursors := make([]string, 0, len(calls))
	for _, call := range calls {
		cursors = append(cursors, call.Cursor)
	}
	mustEqualStrings(t, "请求游标", cursors, []string{"", "c1", "c2"})
}

// TestQueryFiltersArePassedThrough 确认过滤参数原样交给读取方，后续页沿用同一组过滤。
func TestQueryFiltersArePassedThrough(t *testing.T) {
	reader := &fakeReader{}
	reader.setPage("", Page{Entries: []Entry{entry("s1", "", 0)}, NextCursor: "next"})
	reader.setPage("next", Page{Entries: []Entry{entry("s2", "", 0)}})

	catalog := newTestCatalog(t, Config{
		Reader: reader,
		Writes: newWriteManager(t, &recordSender{}),
	})

	query := Query{Search: "构建失败", Workspace: "/work/repo", Limit: 7}
	if _, err := catalog.Refresh(context.Background(), query); err != nil {
		t.Fatalf("读取第一页: %v", err)
	}
	if _, err := catalog.LoadMore(context.Background()); err != nil {
		t.Fatalf("读取第二页: %v", err)
	}

	calls := reader.callList()
	if len(calls) != 2 {
		t.Fatalf("读取次数不符：%d", len(calls))
	}
	if calls[0].Query != query {
		t.Fatalf("第一页过滤参数被改写：got %+v，want %+v", calls[0].Query, query)
	}
	if calls[1].Query != query {
		t.Fatalf("第二页过滤参数被改写：got %+v，want %+v", calls[1].Query, query)
	}
	if calls[1].Cursor != "next" {
		t.Fatalf("第二页游标不符：%q", calls[1].Cursor)
	}
	if got := catalog.Snapshot().Query; got != query {
		t.Fatalf("快照里的过滤参数不符：%+v", got)
	}
}

// TestDefaultPageSize 确认未给页大小时使用默认值，但已给的页大小不被改写。
func TestDefaultPageSize(t *testing.T) {
	reader := &fakeReader{}
	reader.setPage("", Page{})
	catalog := newTestCatalog(t, Config{
		Reader: reader,
		Writes: newWriteManager(t, &recordSender{}),
	})
	if _, err := catalog.Refresh(context.Background(), Query{}); err != nil {
		t.Fatalf("读取目录: %v", err)
	}
	calls := reader.callList()
	if len(calls) != 1 || calls[0].Query.Limit != DefaultPageSize {
		t.Fatalf("默认页大小不符：%+v", calls)
	}
}

// TestRepeatedPageStopsWithoutDuplicates 确认服务端游标不前进时停止推进，且不产生重复行。
func TestRepeatedPageStopsWithoutDuplicates(t *testing.T) {
	reader := &fakeReader{}
	page := Page{Entries: []Entry{entry("s1", "", 0)}, NextCursor: "loop"}
	reader.setPage("", page)
	reader.setPage("loop", page)

	catalog := newTestCatalog(t, Config{
		Reader: reader,
		Writes: newWriteManager(t, &recordSender{}),
	})
	if _, err := catalog.Refresh(context.Background(), Query{}); err != nil {
		t.Fatalf("读取第一页: %v", err)
	}
	result, err := catalog.LoadMore(context.Background())
	if err != nil {
		t.Fatalf("读取重复页: %v", err)
	}
	if result.Added != 0 || result.Duplicates != 1 || result.HasMore || !result.Stopped {
		t.Fatalf("重复页结果不符：%+v", result)
	}
	snapshot := catalog.Snapshot()
	mustEqualStrings(t, "条目", entryIDs(snapshot.Entries), []string{"s1"})
	if snapshot.HasMore {
		t.Fatal("重复页之后仍报告还有下一页")
	}

	// 停止之后不再请求。
	before := len(reader.callList())
	if _, err := catalog.LoadMore(context.Background()); err != nil {
		t.Fatalf("停止后再推进: %v", err)
	}
	if after := len(reader.callList()); after != before {
		t.Fatalf("停止后仍然请求了目录：%d → %d", before, after)
	}
}

// TestReadFailureDoesNotAdvanceCursor 确认读取失败不推进游标，重试仍是同一页。
func TestReadFailureDoesNotAdvanceCursor(t *testing.T) {
	reader := &fakeReader{}
	reader.setPage("", Page{Entries: []Entry{entry("s1", "", 0)}, NextCursor: "c1"})
	reader.setError("c1", errTest)
	reader.setPage("c1", Page{Entries: []Entry{entry("s2", "", 0)}})

	catalog := newTestCatalog(t, Config{
		Reader: reader,
		Writes: newWriteManager(t, &recordSender{}),
	})
	if _, err := catalog.Refresh(context.Background(), Query{}); err != nil {
		t.Fatalf("读取第一页: %v", err)
	}
	if _, err := catalog.LoadMore(context.Background()); !errors.Is(err, errTest) {
		t.Fatalf("读取失败应原样返回：%v", err)
	}
	// 失败后清掉故障，同一游标重试成功且不重复。
	reader.mu.Lock()
	delete(reader.errs, "c1")
	reader.mu.Unlock()
	result, err := catalog.LoadMore(context.Background())
	if err != nil {
		t.Fatalf("重试读取: %v", err)
	}
	if result.Added != 1 || result.HasMore {
		t.Fatalf("重试结果不符：%+v", result)
	}
	mustEqualStrings(t, "条目", entryIDs(catalog.Snapshot().Entries), []string{"s1", "s2"})
}

// TestMalformedPageIsRejected 确认没有会话标识的页被拒绝，本地集合保持不变。
func TestMalformedPageIsRejected(t *testing.T) {
	reader := &fakeReader{}
	reader.setPage("", Page{Entries: []Entry{entry("s1", "", 0)}})
	catalog := newTestCatalog(t, Config{
		Reader: reader,
		Writes: newWriteManager(t, &recordSender{}),
	})
	if _, err := catalog.Refresh(context.Background(), Query{}); err != nil {
		t.Fatalf("读取目录: %v", err)
	}

	reader.setPage("", Page{Entries: []Entry{{ID: ""}}})
	if _, err := catalog.Refresh(context.Background(), Query{}); !errors.Is(err, ErrMalformedPage) {
		t.Fatalf("坏页应被拒绝：%v", err)
	}
	mustEqualStrings(t, "坏页后条目", entryIDs(catalog.Snapshot().Entries), []string{"s1"})
}

// TestTitleMergesByRevision 确认标题通知与改名结果都按修订号合并，重取页面也不回退。
func TestTitleMergesByRevision(t *testing.T) {
	reader := &fakeReader{}
	reader.setPage("", Page{Entries: []Entry{entry("s1", "旧标题", 1)}})
	catalog := newTestCatalog(t, Config{
		Reader: reader,
		Writes: newWriteManager(t, &recordSender{}),
	})
	if _, err := catalog.Refresh(context.Background(), Query{}); err != nil {
		t.Fatalf("读取目录: %v", err)
	}

	if !catalog.ApplyTitle(TitleUpdate{SessionID: "s1", Title: "自动标题", Source: TitleSourceAuto, Revision: 3}) {
		t.Fatal("更高修订号的标题没有合并")
	}
	if got, _ := catalog.Entry("s1"); got.Title != "自动标题" || got.TitleRevision != 3 || got.TitleSource != TitleSourceAuto {
		t.Fatalf("合并后的标题不符：%+v", got)
	}
	if catalog.ApplyTitle(TitleUpdate{SessionID: "s1", Title: "更旧", Source: TitleSourceManual, Revision: 2}) {
		t.Fatal("更低修订号的标题不应合并")
	}
	if catalog.ApplyTitle(TitleUpdate{SessionID: "s1", Title: "同号", Source: TitleSourceManual, Revision: 3}) {
		t.Fatal("相同修订号的标题不应合并")
	}
	if catalog.ApplyTitle(TitleUpdate{SessionID: "unknown", Title: "无主", Source: TitleSourceAuto, Revision: 9}) {
		t.Fatal("未知会话的标题不应凭空新增条目")
	}

	// 重取页面时服务端仍是旧标题（rev 2）：本地已合并的 rev 3 不能被回退。
	reader.setPage("", Page{Entries: []Entry{entry("s1", "旧标题", 2)}})
	if _, err := catalog.Refresh(context.Background(), Query{}); err != nil {
		t.Fatalf("重取目录: %v", err)
	}
	if got, _ := catalog.Entry("s1"); got.Title != "自动标题" || got.TitleRevision != 3 {
		t.Fatalf("重取页面回退了标题：%+v", got)
	}

	// 服务端给出更高修订号时以服务端为准。
	reader.setPage("", Page{Entries: []Entry{entry("s1", "服务端最新", 4)}})
	if _, err := catalog.Refresh(context.Background(), Query{}); err != nil {
		t.Fatalf("再取目录: %v", err)
	}
	if got, _ := catalog.Entry("s1"); got.Title != "服务端最新" || got.TitleRevision != 4 {
		t.Fatalf("更高修订号没有生效：%+v", got)
	}
}

// TestArchiveHidesEntriesAndKeepsServerFacts 确认归档只影响目录可见性。
func TestArchiveHidesEntriesAndKeepsServerFacts(t *testing.T) {
	reader := &fakeReader{}
	reader.setPage("", Page{Entries: []Entry{entry("s1", "一", 1), entry("s2", "二", 1)}})
	archive := newArchive(t)
	catalog := newTestCatalog(t, Config{
		Reader:  reader,
		Writes:  newWriteManager(t, &recordSender{}),
		Archive: archive,
	})
	if _, err := catalog.Refresh(context.Background(), Query{}); err != nil {
		t.Fatalf("读取目录: %v", err)
	}

	if err := catalog.Hide("s1"); err != nil {
		t.Fatalf("隐藏会话: %v", err)
	}
	snapshot := catalog.Snapshot()
	mustEqualStrings(t, "可见条目", entryIDs(snapshot.Entries), []string{"s2"})
	if snapshot.Hidden != 1 {
		t.Fatalf("隐藏计数不符：%d", snapshot.Hidden)
	}
	mustEqualStrings(t, "隐藏条目", entryIDs(catalog.HiddenEntries()), []string{"s1"})
	if !catalog.Hidden("s1") {
		t.Fatal("隐藏状态没有生效")
	}
	// 隐藏不改服务端事实：条目仍在本地集合里，只是不可见。
	if _, ok := catalog.Entry("s1"); !ok {
		t.Fatal("隐藏不应删除条目")
	}

	if err := catalog.Unhide("s1"); err != nil {
		t.Fatalf("取消隐藏: %v", err)
	}
	mustEqualStrings(t, "取消隐藏后的条目", entryIDs(catalog.Snapshot().Entries), []string{"s1", "s2"})
}

// TestArchiveUnavailable 确认没有配置归档时隐藏入口明确报缺失，而不是静默成功。
func TestArchiveUnavailable(t *testing.T) {
	reader := &fakeReader{}
	reader.setPage("", Page{})
	catalog := newTestCatalog(t, Config{
		Reader: reader,
		Writes: newWriteManager(t, &recordSender{}),
	})
	if err := catalog.Hide("s1"); !errors.Is(err, ErrArchiveUnavailable) {
		t.Fatalf("缺归档时应报缺失：%v", err)
	}
	if err := catalog.Unhide("s1"); !errors.Is(err, ErrArchiveUnavailable) {
		t.Fatalf("缺归档时应报缺失：%v", err)
	}
	if catalog.Hidden("s1") {
		t.Fatal("没有归档时不应报告隐藏")
	}
}
