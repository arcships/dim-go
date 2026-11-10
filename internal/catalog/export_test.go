package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

// fakeHistory 按 Before 游标返回预先准备的历史页。
type fakeHistory struct {
	mu    sync.Mutex
	pages map[int64]HistoryPage
	errs  map[int64]error
	calls []HistoryRequest
}

func (h *fakeHistory) ReadHistory(ctx context.Context, req HistoryRequest) (HistoryPage, error) {
	h.mu.Lock()
	h.calls = append(h.calls, req)
	page, ok := h.pages[req.Before]
	err := h.errs[req.Before]
	h.mu.Unlock()
	if err != nil {
		return HistoryPage{}, err
	}
	if !ok {
		return HistoryPage{}, errTest
	}
	return page, nil
}

func (h *fakeHistory) callList() []HistoryRequest {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]HistoryRequest(nil), h.calls...)
}

func (h *fakeHistory) setPage(before int64, page HistoryPage) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.pages == nil {
		h.pages = make(map[int64]HistoryPage)
	}
	h.pages[before] = page
}

func (h *fakeHistory) setError(before int64, err error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.errs == nil {
		h.errs = make(map[int64]error)
	}
	h.errs[before] = err
}

// exportCatalog 建一个带历史来源的目录，并放入一条标题已知的会话。
func exportCatalog(t *testing.T, history HistorySource, archive *Archive) *Catalog {
	t.Helper()
	reader := &fakeReader{}
	reader.setPage("", Page{})
	catalog := newTestCatalog(t, Config{
		Reader:  reader,
		Writes:  newWriteManager(t, &recordSender{}),
		Archive: archive,
		History: history,
	})
	seedCatalogEntries(t, catalog, "s1")
	catalog.mu.Lock()
	catalog.entries[0].Title = "导出标题"
	catalog.mu.Unlock()
	return catalog
}

// TestExportReadsAllTurnsAndWritesVersionedFile 确认按游标读完所有页，文件轮次等于
// 读到的历史总轮次，且带格式版本号。
func TestExportReadsAllTurnsAndWritesVersionedFile(t *testing.T) {
	history := &fakeHistory{}
	history.setPage(0, HistoryPage{
		Turns:      []Turn{{Index: 3, Data: json.RawMessage(`{"text":"第三轮"}`)}, {Index: 4, Data: json.RawMessage(`{"text":"第四轮"}`)}},
		NextBefore: 2,
	})
	history.setPage(2, HistoryPage{
		Turns:      []Turn{{Index: 1, Data: json.RawMessage(`{"text":"第一轮"}`)}, {Index: 2, Data: json.RawMessage(`{"text":"第二轮"}`)}},
		NextBefore: 0,
		Done:       true,
	})
	catalog := exportCatalog(t, history, nil)

	path := filepath.Join(t.TempDir(), "session.json")
	result, err := catalog.Export(context.Background(), ExportRequest{
		SessionID: "s1",
		BranchID:  "main",
		Path:      path,
	})
	if err != nil {
		t.Fatalf("导出: %v", err)
	}
	if result.Turns != 4 || result.Pages != 2 || result.Path != path {
		t.Fatalf("导出结果不符：%+v", result)
	}

	calls := history.callList()
	if len(calls) != 2 || calls[0].Before != 0 || calls[1].Before != 2 {
		t.Fatalf("历史读取游标不符：%+v", calls)
	}
	if calls[0].Limit != defaultExportPageLimit || calls[0].SessionID != "s1" || calls[0].BranchID != "main" {
		t.Fatalf("历史请求参数不符：%+v", calls[0])
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取导出文件: %v", err)
	}
	var document exportDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("导出文件不是合法 JSON: %v", err)
	}
	if document.Format != ExportFormat || document.FormatVersion != ExportFormatVersion {
		t.Fatalf("格式或版本不符：%+v", document)
	}
	if document.SessionID != "s1" || document.BranchID != "main" || document.Title != "导出标题" {
		t.Fatalf("文件头不符：%+v", document)
	}
	if document.TurnCount != 4 || len(document.Turns) != document.TurnCount {
		t.Fatalf("轮次数不符：turnCount=%d，turns=%d", document.TurnCount, len(document.Turns))
	}
	for i, turn := range document.Turns {
		want := int64(i + 1)
		if turn.Index != want {
			t.Fatalf("第 %d 个轮次序号不符：got %d，want %d", i, turn.Index, want)
		}
	}
	var first map[string]string
	if err := json.Unmarshal(document.Turns[0].Data, &first); err != nil {
		t.Fatalf("轮次内容无法解析: %v", err)
	}
	if first["text"] != "第一轮" {
		t.Fatalf("轮次内容不符：%+v", first)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("查看导出文件属性: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Fatalf("导出文件权限不符：%o", perm)
	}

	// 没有临时文件残留。
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("读取导出目录: %v", err)
	}
	for _, item := range entries {
		if item.Name() != filepath.Base(path) {
			t.Fatalf("导出后留下临时文件: %s", item.Name())
		}
	}
}

// TestExportPageFailureLeavesNoFile 确认中途失败不产出文件，已有文件保持原样。
func TestExportPageFailureLeavesNoFile(t *testing.T) {
	history := &fakeHistory{}
	history.setPage(0, HistoryPage{
		Turns:      []Turn{{Index: 2, Data: json.RawMessage(`{"text":"第二轮"}`)}},
		NextBefore: 1,
	})
	history.setError(1, errTest)
	catalog := exportCatalog(t, history, nil)

	dir := t.TempDir()
	missing := filepath.Join(dir, "missing.json")
	if _, err := catalog.Export(context.Background(), ExportRequest{SessionID: "s1", Path: missing}); !errors.Is(err, errTest) {
		t.Fatalf("中途失败应原样返回错误：%v", err)
	}
	if _, err := os.Stat(missing); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("中途失败不应产出文件：%v", err)
	}

	existing := filepath.Join(dir, "existing.json")
	if err := os.WriteFile(existing, []byte("旧内容"), 0o600); err != nil {
		t.Fatalf("准备旧文件: %v", err)
	}
	if _, err := catalog.Export(context.Background(), ExportRequest{SessionID: "s1", Path: existing}); err == nil {
		t.Fatal("中途失败应返回错误")
	}
	raw, err := os.ReadFile(existing)
	if err != nil {
		t.Fatalf("读取旧文件: %v", err)
	}
	if string(raw) != "旧内容" {
		t.Fatalf("中途失败改动了旧文件：%q", raw)
	}
}

// TestExportRejectsRepeatedCursor 确认游标重复时停止并报错，不产出文件。
func TestExportRejectsRepeatedCursor(t *testing.T) {
	history := &fakeHistory{}
	history.setPage(0, HistoryPage{Turns: []Turn{{Index: 1, Data: json.RawMessage(`{}`)}}, NextBefore: 5})
	history.setPage(5, HistoryPage{Turns: []Turn{{Index: 2, Data: json.RawMessage(`{}`)}}, NextBefore: 5})
	catalog := exportCatalog(t, history, nil)

	path := filepath.Join(t.TempDir(), "out.json")
	if _, err := catalog.Export(context.Background(), ExportRequest{SessionID: "s1", Path: path}); !errors.Is(err, ErrRepeatedCursor) {
		t.Fatalf("游标重复应报错：%v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("游标重复时不应产出文件：%v", err)
	}
}

// TestExportDeduplicatesTurnsAndRejectsConflicts 确认重复轮次只算一次，内容不一致即报错。
func TestExportDeduplicatesTurnsAndRejectsConflicts(t *testing.T) {
	history := &fakeHistory{}
	history.setPage(0, HistoryPage{
		Turns:      []Turn{{Index: 2, Data: json.RawMessage(`{"text":"乙"}`)}},
		NextBefore: 1,
	})
	history.setPage(1, HistoryPage{
		Turns:      []Turn{{Index: 1, Data: json.RawMessage(`{"text":"甲"}`)}, {Index: 2, Data: json.RawMessage(`{"text":"乙"}`)}},
		NextBefore: 0,
	})
	catalog := exportCatalog(t, history, nil)

	path := filepath.Join(t.TempDir(), "out.json")
	result, err := catalog.Export(context.Background(), ExportRequest{SessionID: "s1", Path: path})
	if err != nil {
		t.Fatalf("导出: %v", err)
	}
	if result.Turns != 2 {
		t.Fatalf("重复轮次应只算一次：%d", result.Turns)
	}

	// 同一轮次序号、不同内容：视为来源不一致，拒绝导出。
	conflicting := &fakeHistory{}
	conflicting.setPage(0, HistoryPage{Turns: []Turn{{Index: 1, Data: json.RawMessage(`{"text":"甲"}`)}}, NextBefore: 1})
	conflicting.setPage(1, HistoryPage{Turns: []Turn{{Index: 1, Data: json.RawMessage(`{"text":"乙"}`)}}, NextBefore: 0})
	conflictCatalog := exportCatalog(t, conflicting, nil)
	conflictPath := filepath.Join(t.TempDir(), "conflict.json")
	if _, err := conflictCatalog.Export(context.Background(), ExportRequest{SessionID: "s1", Path: conflictPath}); !errors.Is(err, ErrConflictingTurn) {
		t.Fatalf("轮次冲突应报错：%v", err)
	}
	if _, err := os.Stat(conflictPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("轮次冲突时不应产出文件：%v", err)
	}
}

// TestExportRejectsMalformedTurns 确认序号非正或内容不是合法 JSON 的轮次被拒绝。
func TestExportRejectsMalformedTurns(t *testing.T) {
	cases := map[string]Turn{
		"序号为零":     {Index: 0, Data: json.RawMessage(`{}`)},
		"内容非 JSON": {Index: 1, Data: json.RawMessage(`不是 JSON`)},
	}
	for name, turn := range cases {
		t.Run(name, func(t *testing.T) {
			history := &fakeHistory{}
			history.setPage(0, HistoryPage{Turns: []Turn{turn}, NextBefore: 0})
			catalog := exportCatalog(t, history, nil)
			path := filepath.Join(t.TempDir(), "out.json")
			if _, err := catalog.Export(context.Background(), ExportRequest{SessionID: "s1", Path: path}); !errors.Is(err, ErrMalformedTurn) {
				t.Fatalf("坏轮次应被拒绝：%v", err)
			}
			if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("坏轮次时不应产出文件：%v", err)
			}
		})
	}
}

// TestExportStopsAtPageLimit 确认页数上限生效。
func TestExportStopsAtPageLimit(t *testing.T) {
	history := &fakeHistory{}
	for before := int64(0); before < 3; before++ {
		history.setPage(before, HistoryPage{
			Turns:      []Turn{{Index: before + 1, Data: json.RawMessage(`{}`)}},
			NextBefore: before + 1,
		})
	}
	catalog := exportCatalog(t, history, nil)
	path := filepath.Join(t.TempDir(), "out.json")
	if _, err := catalog.Export(context.Background(), ExportRequest{
		SessionID: "s1",
		Path:      path,
		MaxPages:  2,
	}); !errors.Is(err, ErrExportLimit) {
		t.Fatalf("页数超限应报错：%v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("页数超限时不应产出文件：%v", err)
	}
}

// TestExportRequiresConfigurationAndFields 确认缺来源或字段时明确拒绝。
func TestExportRequiresConfigurationAndFields(t *testing.T) {
	reader := &fakeReader{}
	reader.setPage("", Page{})
	catalog := newTestCatalog(t, Config{Reader: reader, Writes: newWriteManager(t, &recordSender{})})

	if _, err := catalog.Export(context.Background(), ExportRequest{SessionID: "s1", Path: "out.json"}); !errors.Is(err, ErrExportUnavailable) {
		t.Fatalf("缺历史来源应报缺失：%v", err)
	}

	history := &fakeHistory{}
	catalog = exportCatalog(t, history, nil)
	if _, err := catalog.Export(context.Background(), ExportRequest{SessionID: "", Path: "out.json"}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("缺会话标识应被拒绝：%v", err)
	}
	if _, err := catalog.Export(context.Background(), ExportRequest{SessionID: "s1", Path: "  "}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("缺文件路径应被拒绝：%v", err)
	}
}

// TestExportTitleCanBeOverridden 确认调用方给的标题优先于本地目录标题。
func TestExportTitleCanBeOverridden(t *testing.T) {
	history := &fakeHistory{}
	history.setPage(0, HistoryPage{Done: true})
	catalog := exportCatalog(t, history, nil)

	path := filepath.Join(t.TempDir(), "out.json")
	if _, err := catalog.Export(context.Background(), ExportRequest{
		SessionID: "s1",
		Path:      path,
		Title:     "覆盖标题",
	}); err != nil {
		t.Fatalf("导出: %v", err)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("读取导出文件: %v", err)
	}
	var document exportDocument
	if err := json.Unmarshal(raw, &document); err != nil {
		t.Fatalf("导出文件不是合法 JSON: %v", err)
	}
	if document.Title != "覆盖标题" || document.TurnCount != 0 {
		t.Fatalf("文件头不符：%+v", document)
	}
}

// TestExportPathMustExist 确认目标目录不存在时失败且不产出文件。
func TestExportPathMustExist(t *testing.T) {
	history := &fakeHistory{}
	history.setPage(0, HistoryPage{Done: true})
	catalog := exportCatalog(t, history, nil)
	_, err := catalog.Export(context.Background(), ExportRequest{
		SessionID: "s1",
		Path:      filepath.Join(t.TempDir(), "no-such-dir", "out.json"),
	})
	if err == nil || !strings.Contains(err.Error(), "写入导出文件") {
		t.Fatalf("目标目录不存在应失败：%v", err)
	}
}
