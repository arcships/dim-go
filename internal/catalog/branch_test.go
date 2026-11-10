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

	"github.com/arcships/dimagent-mygo/internal/write"
)

// fakeBranchSource 是会话投影分支事实的替身。
type fakeBranchSource struct {
	mu       sync.Mutex
	branches []Branch
	complete bool
	err      error
	points   map[string]CutPoint
	read     int
	cutCalls int
}

func (f *fakeBranchSource) ReadBranches(
	ctx context.Context,
	sessionID string,
) ([]Branch, bool, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.read++
	if f.err != nil {
		return nil, false, f.err
	}
	return append([]Branch(nil), f.branches...), f.complete, nil
}

func (f *fakeBranchSource) TurnCut(sessionID, branchID string, turnIndex int) CutPoint {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cutCalls++
	key := fmt.Sprintf("%s#%d", branchID, turnIndex)
	if point, ok := f.points[key]; ok {
		point.TurnIndex = turnIndex
		return point
	}
	return CutPoint{Status: CutNotLoaded, TurnIndex: turnIndex}
}

func (f *fakeBranchSource) readCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.read
}

func (f *fakeBranchSource) cutCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.cutCalls
}

func (f *fakeBranchSource) setError(err error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.err = err
}

// newBranchCatalog 建一个带分支来源与空目录的目录。
func newBranchCatalog(t *testing.T, source BranchSource, sender *recordSender) *Catalog {
	t.Helper()
	reader := &fakeReader{}
	reader.setPage("", Page{})
	return newTestCatalog(t, Config{
		Reader:   reader,
		Writes:   newWriteManager(t, sender),
		Branches: source,
	})
}

// decodeBranchCreate 解出分支创建意图的参数。
func decodeBranchCreate(t *testing.T, req write.Request) branchCreatePayload {
	t.Helper()
	var payload branchCreatePayload
	if err := json.Unmarshal(req.Params, &payload); err != nil {
		t.Fatalf("分支创建参数无法解析: %v（%s）", err, req.Params)
	}
	return payload
}

// branchCall 找出指定 op 的写请求。
func branchCall(t *testing.T, sender *recordSender, op string) (write.Request, bool) {
	t.Helper()
	for _, req := range sender.requestList() {
		var head struct {
			Op string `json:"op"`
		}
		if err := json.Unmarshal(req.Params, &head); err != nil {
			continue
		}
		if head.Op == op {
			return req, true
		}
	}
	return write.Request{}, false
}

// TestCreateBranchMergesIntoDirectoryWithoutRebuild 覆盖：创建成功并入目录、只改
// 新条目、已完成条目（含固定头）不变、来源只读一次、切点按投影给的轮结束序号。
func TestCreateBranchMergesIntoDirectoryWithoutRebuild(t *testing.T) {
	source := &fakeBranchSource{
		branches: []Branch{
			{ID: "main", Status: BranchOpen, Head: 10},
			{ID: "old", ParentID: "main", CutSequence: 4, Status: BranchEnded, Head: 7},
		},
		complete: true,
		points:   map[string]CutPoint{"main#2": {Status: CutReady, Sequence: 6}},
	}
	sender := &recordSender{handle: func(req write.Request) (write.Reply, error) {
		if req.Kind != write.KindSessionLifecycle {
			return write.Reply{}, fmt.Errorf("分支创建用了错误的意图类型: %s", req.Kind)
		}
		return write.Reply{Result: json.RawMessage(
			`{"branch":{"branchId":"b-new","parentBranchId":"main","branchPoint":6,"createEvent":11}}`,
		)}, nil
	}}
	catalog := newBranchCatalog(t, source, sender)

	before, err := catalog.RefreshBranches(context.Background(), "s1")
	if err != nil {
		t.Fatalf("读取分支目录: %v", err)
	}
	if !before.Complete || !before.Loaded || len(before.Branches) != 2 {
		t.Fatalf("初始目录不符：%+v", before)
	}

	created, err := catalog.CreateBranch(context.Background(), BranchCreateRequest{
		SessionID:      "s1",
		ParentBranchID: "main",
		TurnIndex:      2,
		Prompt:         "换一条路试试",
	})
	if err != nil {
		t.Fatalf("创建分支: %v", err)
	}
	if created.ID != "b-new" || created.ParentID != "main" ||
		created.CutSequence != 6 || created.Status != BranchOpen {
		t.Fatalf("新分支条目不符：%+v", created)
	}

	after := catalog.Branches("s1")
	if !after.Complete || len(after.Branches) != 3 {
		t.Fatalf("并入后的目录不符：%+v", after)
	}
	if !reflect.DeepEqual(after.Branches[0], before.Branches[0]) {
		t.Fatalf("已有分支被改动：%+v → %+v", before.Branches[0], after.Branches[0])
	}
	if !reflect.DeepEqual(after.Branches[1], before.Branches[1]) {
		t.Fatalf("已结束分支被改动：%+v → %+v", before.Branches[1], after.Branches[1])
	}
	if after.Branches[2].ID != "b-new" {
		t.Fatalf("新分支没有追加到目录末尾：%+v", after.Branches)
	}
	if got := source.readCount(); got != 1 {
		t.Fatalf("创建分支不应重建整个目录：读取次数 %d", got)
	}

	request, ok := branchCall(t, sender, opBranchCreate)
	if !ok {
		t.Fatal("没有登记分支创建意图")
	}
	if request.Kind != write.KindSessionLifecycle || request.SessionID != "s1" {
		t.Fatalf("分支创建意图的归属不符：%+v", request)
	}
	payload := decodeBranchCreate(t, request)
	if payload.ParentBranchID != "main" || payload.At != 6 || payload.Prompt != "换一条路试试" {
		t.Fatalf("分支创建参数不符：%+v", payload)
	}
}

// TestCreateBranchFailureKeepsDirectory 覆盖：失败保留原目录并给原因。
func TestCreateBranchFailureKeepsDirectory(t *testing.T) {
	source := &fakeBranchSource{
		branches: []Branch{{ID: "main", Status: BranchOpen, Head: 10}},
		complete: true,
		points:   map[string]CutPoint{"main#2": {Status: CutReady, Sequence: 6}},
	}
	sender := &recordSender{handle: func(req write.Request) (write.Reply, error) {
		return write.Reply{}, errors.New("服务端拒绝创建分支")
	}}
	catalog := newBranchCatalog(t, source, sender)
	before, err := catalog.RefreshBranches(context.Background(), "s1")
	if err != nil {
		t.Fatalf("读取分支目录: %v", err)
	}

	_, err = catalog.CreateBranch(context.Background(), BranchCreateRequest{
		SessionID: "s1",
		TurnIndex: 2,
		Prompt:    "换一条路试试",
	})
	if !errors.Is(err, ErrRejected) {
		t.Fatalf("创建失败应返回拒绝错误：%v", err)
	}
	var rejected *RejectedError
	if !errors.As(err, &rejected) || rejected.Op != OpBranchCreate {
		t.Fatalf("拒绝原因不完整：%+v", err)
	}
	if !strings.Contains(err.Error(), "服务端拒绝创建分支") {
		t.Fatalf("拒绝错误没有带回原因：%v", err)
	}
	after := catalog.Branches("s1")
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("创建失败改动了分支目录：%+v → %+v", before, after)
	}
}

// TestCreateBranchRejectsUnloadedCut 覆盖：切点不在已加载范围内时明确拒绝且不发请求。
func TestCreateBranchRejectsUnloadedCut(t *testing.T) {
	source := &fakeBranchSource{
		branches: []Branch{{ID: "main", Status: BranchOpen, Head: 10}},
		complete: true,
		points:   map[string]CutPoint{"main#2": {Status: CutReady, Sequence: 6}},
	}
	sender := &recordSender{}
	catalog := newBranchCatalog(t, source, sender)
	if _, err := catalog.RefreshBranches(context.Background(), "s1"); err != nil {
		t.Fatalf("读取分支目录: %v", err)
	}

	_, err := catalog.CreateBranch(context.Background(), BranchCreateRequest{
		SessionID: "s1",
		TurnIndex: 5, // 只有第 2 轮在已加载范围
		Prompt:    "换一条路试试",
	})
	if !errors.Is(err, ErrCutNotLoaded) {
		t.Fatalf("未加载的切点应被拒绝：%v", err)
	}
	if !strings.Contains(err.Error(), "第 5 轮") || !strings.Contains(err.Error(), "向前翻页") {
		t.Fatalf("未加载的拒绝原因没有说明要翻页：%v", err)
	}
	if strings.Contains(err.Error(), "还没结束") {
		t.Fatalf("把未加载说成了未结束：%v", err)
	}
	if calls := sender.requestList(); len(calls) != 0 {
		t.Fatalf("切点被拒绝后不应发请求：%d", len(calls))
	}
}

// TestCreateBranchRejectsEndedParent 覆盖：已结束分支拒绝再开分支。
func TestCreateBranchRejectsEndedParent(t *testing.T) {
	source := &fakeBranchSource{
		branches: []Branch{
			{ID: "main", Status: BranchOpen, Head: 10},
			{ID: "old", ParentID: "main", CutSequence: 4, Status: BranchEnded, Head: 7},
		},
		complete: true,
		points:   map[string]CutPoint{"old#1": {Status: CutReady, Sequence: 3}},
	}
	sender := &recordSender{}
	catalog := newBranchCatalog(t, source, sender)
	if _, err := catalog.RefreshBranches(context.Background(), "s1"); err != nil {
		t.Fatalf("读取分支目录: %v", err)
	}

	_, err := catalog.CreateBranch(context.Background(), BranchCreateRequest{
		SessionID:      "s1",
		ParentBranchID: "old",
		TurnIndex:      1,
		Prompt:         "再开一条",
	})
	if !errors.Is(err, ErrBranchEnded) {
		t.Fatalf("已结束分支应拒绝创建：%v", err)
	}
	if cuts := source.cutCount(); cuts != 0 {
		t.Fatalf("父分支已结束时不应再取切点：%d", cuts)
	}
	if calls := sender.requestList(); len(calls) != 0 {
		t.Fatalf("父分支已结束时不应发请求：%d", len(calls))
	}
}

// TestCreateBranchRejectsNotEndedCut 覆盖：该轮还没结束时不报“翻页”，而是要求等这一轮结束。
func TestCreateBranchRejectsNotEndedCut(t *testing.T) {
	source := &fakeBranchSource{
		branches: []Branch{{ID: "main", Status: BranchOpen, Head: 10}},
		complete: true,
		points:   map[string]CutPoint{"main#1": {Status: CutNotEnded}},
	}
	sender := &recordSender{}
	catalog := newBranchCatalog(t, source, sender)
	if _, err := catalog.RefreshBranches(context.Background(), "s1"); err != nil {
		t.Fatalf("读取分支目录: %v", err)
	}

	_, err := catalog.CreateBranch(context.Background(), BranchCreateRequest{
		SessionID: "s1",
		TurnIndex: 1,
		Prompt:    "等这轮说完再开",
	})
	if !errors.Is(err, ErrCutNotEnded) {
		t.Fatalf("未结束的轮次应被拒绝：%v", err)
	}
	if !strings.Contains(err.Error(), "还没结束") {
		t.Fatalf("拒绝原因没有说明未结束：%v", err)
	}
	if strings.Contains(err.Error(), "翻页") {
		t.Fatalf("未结束的拒绝原因不应让人去翻页：%v", err)
	}
	if calls := sender.requestList(); len(calls) != 0 {
		t.Fatalf("未结束的切点不应发请求：%d", len(calls))
	}
}

// TestCreateBranchRejectsUnknownBranchCut 覆盖：切点查询说分支不存在时报未知分支错误。
func TestCreateBranchRejectsUnknownBranchCut(t *testing.T) {
	source := &fakeBranchSource{
		points: map[string]CutPoint{"ghost#1": {Status: CutNoBranch}},
	}
	sender := &recordSender{}
	catalog := newBranchCatalog(t, source, sender)

	_, err := catalog.CreateBranch(context.Background(), BranchCreateRequest{
		SessionID:      "s1",
		ParentBranchID: "ghost",
		TurnIndex:      1,
		Prompt:         "开一条",
	})
	if !errors.Is(err, ErrUnknownBranch) {
		t.Fatalf("未知分支应报未知分支错误：%v", err)
	}
	if calls := sender.requestList(); len(calls) != 0 {
		t.Fatalf("未知分支不应发请求：%d", len(calls))
	}
}

// TestBranchStatusAdvancesAndEndedHeadIsFixed 覆盖：状态只前进、已结束分支头部固定、
// 已结束分支拒绝再提交。
func TestBranchStatusAdvancesAndEndedHeadIsFixed(t *testing.T) {
	source := &fakeBranchSource{
		branches: []Branch{
			{ID: "main", Status: BranchOpen, Head: 10},
			{ID: "b1", ParentID: "main", CutSequence: 6, Status: BranchOpen, Head: 8},
		},
		complete: true,
	}
	catalog := newBranchCatalog(t, source, &recordSender{})
	if _, err := catalog.RefreshBranches(context.Background(), "s1"); err != nil {
		t.Fatalf("读取分支目录: %v", err)
	}

	if err := catalog.CheckBranchWritable("s1", "b1"); err != nil {
		t.Fatalf("进行中的分支应可提交：%v", err)
	}

	ending, err := catalog.ApplyBranchUpdate("s1", Branch{ID: "b1", Status: BranchEnding, Head: 8})
	if err != nil {
		t.Fatalf("结束中状态: %v", err)
	}
	if ending.Status != BranchEnding || ending.Head != 8 {
		t.Fatalf("结束中状态不符：%+v", ending)
	}

	ended, err := catalog.ApplyBranchUpdate("s1", Branch{ID: "b1", Status: BranchEnded, Head: 9})
	if err != nil {
		t.Fatalf("已结束状态: %v", err)
	}
	if ended.Status != BranchEnded || ended.Head != 9 {
		t.Fatalf("已结束状态不符：%+v", ended)
	}

	// 后续更新不能把状态改回 open，也不能抬高已结束分支的头部。
	reopened, err := catalog.ApplyBranchUpdate("s1", Branch{ID: "b1", Status: BranchOpen, Head: 12})
	if err != nil {
		t.Fatalf("迟到更新: %v", err)
	}
	if reopened.Status != BranchEnded || reopened.Head != 9 {
		t.Fatalf("已结束分支被改动：%+v", reopened)
	}

	if err := catalog.CheckBranchWritable("s1", "b1"); !errors.Is(err, ErrBranchEnded) {
		t.Fatalf("已结束分支应拒绝提交：%v", err)
	}
	if err := catalog.CheckBranchWritable("s1", "missing"); !errors.Is(err, ErrUnknownBranch) {
		t.Fatalf("未知分支应报未知：%v", err)
	}
	if err := catalog.CheckBranchWritable("s1", "main"); err != nil {
		t.Fatalf("主线不应被拦截：%v", err)
	}
}

// TestEndBranchConvergesStatus 覆盖：结束成功先收敛为 ending，已结束不再发请求。
func TestEndBranchConvergesStatus(t *testing.T) {
	source := &fakeBranchSource{
		branches: []Branch{
			{ID: "main", Status: BranchOpen, Head: 10},
			{ID: "b1", ParentID: "main", CutSequence: 6, Status: BranchOpen, Head: 8},
		},
		complete: true,
	}
	sender := &recordSender{}
	catalog := newBranchCatalog(t, source, sender)
	if _, err := catalog.RefreshBranches(context.Background(), "s1"); err != nil {
		t.Fatalf("读取分支目录: %v", err)
	}

	ended, err := catalog.EndBranch(context.Background(), BranchEndRequest{SessionID: "s1", BranchID: "b1"})
	if err != nil {
		t.Fatalf("结束分支: %v", err)
	}
	if ended.Status != BranchEnding {
		t.Fatalf("结束受理后应等待收敛为结束中：%+v", ended)
	}
	request, ok := branchCall(t, sender, opBranchEnd)
	if !ok {
		t.Fatal("没有登记结束分支意图")
	}
	if request.SessionID != "s1" || request.BranchID != "b1" || request.Kind != write.KindSessionLifecycle {
		t.Fatalf("结束分支意图的归属不符：%+v", request)
	}

	// 已经结束的分支：不再发请求。
	before := len(sender.requestList())
	converged, err := catalog.ApplyBranchUpdate("s1", Branch{ID: "b1", Status: BranchEnded, Head: 9})
	if err != nil {
		t.Fatalf("收敛为已结束: %v", err)
	}
	if converged.Status != BranchEnded {
		t.Fatalf("收敛状态不符：%+v", converged)
	}
	again, err := catalog.EndBranch(context.Background(), BranchEndRequest{SessionID: "s1", BranchID: "b1"})
	if err != nil {
		t.Fatalf("已结束分支再次结束应幂等：%v", err)
	}
	if again.Status != BranchEnded {
		t.Fatalf("再次结束后的状态不符：%+v", again)
	}
	if got := len(sender.requestList()); got != before {
		t.Fatalf("已结束分支不应再发请求：%d → %d", before, got)
	}

	if _, err := catalog.EndBranch(context.Background(), BranchEndRequest{SessionID: "s1", BranchID: "main"}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("主线不能结束：%v", err)
	}
}

// TestEndBranchFailureKeepsDirectory 覆盖：结束失败时目录保持原样并给原因。
func TestEndBranchFailureKeepsDirectory(t *testing.T) {
	source := &fakeBranchSource{
		branches: []Branch{{ID: "main", Status: BranchOpen, Head: 10}, {ID: "b1", Status: BranchOpen, Head: 8}},
		complete: true,
	}
	sender := &recordSender{handle: func(req write.Request) (write.Reply, error) {
		return write.Reply{}, errors.New("分支正在运行")
	}}
	catalog := newBranchCatalog(t, source, sender)
	before, err := catalog.RefreshBranches(context.Background(), "s1")
	if err != nil {
		t.Fatalf("读取分支目录: %v", err)
	}

	_, err = catalog.EndBranch(context.Background(), BranchEndRequest{SessionID: "s1", BranchID: "b1"})
	if !errors.Is(err, ErrRejected) {
		t.Fatalf("结束失败应返回拒绝错误：%v", err)
	}
	if !strings.Contains(err.Error(), "分支正在运行") {
		t.Fatalf("拒绝错误没有带回原因：%v", err)
	}
	if after := catalog.Branches("s1"); !reflect.DeepEqual(before, after) {
		t.Fatalf("结束失败改动了目录：%+v → %+v", before, after)
	}
}

// TestBranchDirectoryDerivedIsMarkedIncomplete 覆盖：服务端不提供目录时标记不完整；
// 读取失败保留原目录。
func TestBranchDirectoryDerivedIsMarkedIncomplete(t *testing.T) {
	source := &fakeBranchSource{
		branches: []Branch{{ID: "main", Status: BranchOpen, Head: 4}},
		complete: false,
	}
	catalog := newBranchCatalog(t, source, &recordSender{})

	derived, err := catalog.RefreshBranches(context.Background(), "s1")
	if err != nil {
		t.Fatalf("读取推导目录: %v", err)
	}
	if derived.Complete {
		t.Fatal("服务端没有目录时应标记为不完整")
	}
	if !derived.Loaded || len(derived.Branches) != 1 {
		t.Fatalf("推导目录内容不符：%+v", derived)
	}

	source.setError(errTest)
	kept, err := catalog.RefreshBranches(context.Background(), "s1")
	if !errors.Is(err, errTest) {
		t.Fatalf("读取失败应原样返回：%v", err)
	}
	if !reflect.DeepEqual(kept, derived) {
		t.Fatalf("读取失败改动了目录：%+v → %+v", derived, kept)
	}
}

// TestBranchDirectoryRejectsMalformed 覆盖：没有标识或状态未知的目录项被拒绝。
func TestBranchDirectoryRejectsMalformed(t *testing.T) {
	for name, branches := range map[string][]Branch{
		"缺少标识": {{ID: " ", Status: BranchOpen}},
		"状态未知": {{ID: "b1", Status: BranchStatus("paused")}},
	} {
		t.Run(name, func(t *testing.T) {
			source := &fakeBranchSource{branches: branches, complete: true}
			catalog := newBranchCatalog(t, source, &recordSender{})
			if _, err := catalog.RefreshBranches(context.Background(), "s1"); !errors.Is(err, ErrBranchDirectoryMalformed) {
				t.Fatalf("坏目录应被拒绝：%v", err)
			}
			if directory := catalog.Branches("s1"); directory.Loaded {
				t.Fatalf("坏目录不应写入本地：%+v", directory)
			}
		})
	}
}

// TestBranchRequestsRequireFields 覆盖：缺会话、切点轮次或提示词时在登记前拒绝。
func TestBranchRequestsRequireFields(t *testing.T) {
	source := &fakeBranchSource{complete: true, points: map[string]CutPoint{"main#1": {Status: CutReady, Sequence: 5}}}
	sender := &recordSender{}
	catalog := newBranchCatalog(t, source, sender)

	cases := []BranchCreateRequest{
		{TurnIndex: 1, Prompt: "x"},
		{SessionID: "s1", TurnIndex: 0, Prompt: "x"},
		{SessionID: "s1", TurnIndex: 1, Prompt: "  "},
	}
	for _, request := range cases {
		if _, err := catalog.CreateBranch(context.Background(), request); !errors.Is(err, ErrInvalidRequest) {
			t.Fatalf("缺字段应被拒绝：%+v → %v", request, err)
		}
	}
	if _, err := catalog.EndBranch(context.Background(), BranchEndRequest{BranchID: "b1"}); !errors.Is(err, ErrInvalidRequest) {
		t.Fatalf("结束分支缺会话应被拒绝：%v", err)
	}
	if calls := sender.requestList(); len(calls) != 0 {
		t.Fatalf("缺字段不应发请求：%d", len(calls))
	}
}
