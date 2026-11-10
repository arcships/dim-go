package projection

import (
	"context"
	"encoding/json"
	"reflect"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// fakeHistoryReader 模仿服务端的历史窗口：首屏读最近一轮（窗口从该轮起点开始），
// 向前翻页读游标之前的整段窗口。数据量小，所以不做字节裁剪。
type fakeHistoryReader struct {
	events   []protocol.RawEvent
	requests []protocol.SessionHistoryParams
}

func newFakeHistoryReader(events []protocol.RawEvent) *fakeHistoryReader {
	return &fakeHistoryReader{events: append([]protocol.RawEvent(nil), events...)}
}

func (f *fakeHistoryReader) History(
	_ context.Context,
	params protocol.SessionHistoryParams,
) (protocol.SessionHistoryResult, error) {
	f.requests = append(f.requests, params)
	head := f.events[len(f.events)-1].Sequence
	result := protocol.SessionHistoryResult{Through: head, NextAfter: head}

	if params.LatestTurn != nil && *params.LatestTurn {
		start := int64(0)
		for _, event := range f.events {
			if event.Type == "turn" && event.Name == "start" {
				start = event.Sequence
			}
		}
		for _, event := range f.events {
			if event.Sequence >= start {
				result.Events = append(result.Events, event)
			}
		}
		result.TurnStart = &start
		result.NextBefore = &start
		result.Done = start <= 1
		return result, nil
	}

	before := int64(0)
	if params.Before != nil {
		before = *params.Before
	}
	for _, event := range f.events {
		if event.Sequence < before {
			result.Events = append(result.Events, event)
		}
	}
	if len(result.Events) == 0 {
		result.Done = true
		return result, nil
	}
	next := result.Events[0].Sequence
	result.NextBefore = &next
	result.Done = next <= 1
	return result, nil
}

// TestTurnHistoryPagesEveryTurnOnce 连续向前翻页直到到底：轮次连续、不重复、不遗漏，
// 每轮的行数据与一次读全的折叠结果一致。
func TestTurnHistoryPagesEveryTurnOnce(t *testing.T) {
	const turns = 6
	events := turnFixture(turns)
	const through = int64(42)

	reference := newFolder(t)
	mustHistory(t, reference, mainBranch, page(through, events...))
	referenceView := reference.View()
	referenceRows := testTurnBuckets(referenceView)

	reader := newFakeHistoryReader(events)
	source, err := NewTurnHistory("s-1", reader)
	if err != nil {
		t.Fatalf("新建按轮历史来源: %v", err)
	}

	var pages [][]int64
	before := int64(0)
	pageCount := 0
	for {
		if pageCount > turns+2 {
			t.Fatalf("翻页没有收敛: 已经 %d 页", pageCount)
		}
		got, err := source.ReadHistory(context.Background(), HistoryRequest{
			SessionID: "s-1",
			BranchID:  mainBranch,
			Before:    before,
			Limit:     2,
		})
		if err != nil {
			t.Fatalf("读取第 %d 页: %v", pageCount+1, err)
		}
		pageCount++

		pageIndexes := make([]int64, 0, len(got.Turns))
		for index, turn := range got.Turns {
			if turn.Index <= 0 {
				t.Fatalf("轮次序号必须为正: %+v", turn)
			}
			if !json.Valid(turn.Data) {
				t.Fatalf("第 %d 轮的内容不是合法 JSON: %s", turn.Index, turn.Data)
			}
			if index > 0 && got.Turns[index-1].Index >= turn.Index {
				t.Fatalf("一页内轮次必须升序: %+v", got.Turns)
			}
			pageIndexes = append(pageIndexes, turn.Index)
			assertTurnDocument(t, turn, referenceView, referenceRows)
		}
		pages = append(pages, pageIndexes)
		if got.Done {
			break
		}
		if got.NextBefore == 0 {
			t.Fatalf("没到头却没有下一页游标")
		}
		if before != 0 && got.NextBefore >= before {
			t.Fatalf("游标没有向前推进: %d -> %d", before, got.NextBefore)
		}
		before = got.NextBefore
	}

	// 每页内部升序，页与页之间向更早的轮次推进：三页正好覆盖 1..6，无重复无遗漏。
	wantPages := [][]int64{{5, 6}, {3, 4}, {1, 2}}
	if !reflect.DeepEqual(pages, wantPages) {
		t.Fatalf("翻页顺序应为 %v，实际 %v", wantPages, pages)
	}
	seenIndexes := make(map[int64]int, turns)
	for _, page := range pages {
		for _, index := range page {
			seenIndexes[index]++
		}
	}
	for index := int64(1); index <= turns; index++ {
		if seenIndexes[index] != 1 {
			t.Fatalf("轮次 %d 出现 %d 次，应为 1 次", index, seenIndexes[index])
		}
	}
	for _, request := range reader.requests {
		if request.Limit == nil || *request.Limit != turnHistoryServerPageLimit {
			t.Fatalf("发给服务端的单页上限不对: %+v", request)
		}
	}
	if len(source.Rejections()) != 0 {
		t.Fatalf("按轮读取不该丢弃事实: %+v", source.Rejections())
	}
}

// testTurnBuckets 把一次读全的视图按轮归桶，键是导出序号（与折叠里的轮序号一致）。
func testTurnBuckets(view View) map[int64][]Row {
	ordered := orderedTurns(view.Turns)
	index := make(map[int]int64, len(ordered))
	for position, turn := range ordered {
		index[turn.Index] = int64(position + 1)
	}
	return bucketRowsByTurn(view.Rows, ordered, index)
}

// assertTurnDocument 校验一轮的行数据与一次读全的折叠结果一致。
func assertTurnDocument(
	t *testing.T,
	turn HistoryTurn,
	reference View,
	referenceRows map[int64][]Row,
) {
	t.Helper()
	var document turnDocument
	if err := json.Unmarshal(turn.Data, &document); err != nil {
		t.Fatalf("第 %d 轮内容无法解码: %v", turn.Index, err)
	}
	if document.Index != turn.Index {
		t.Fatalf("轮次序号不一致: 文档 %d，外层 %d", document.Index, turn.Index)
	}
	want := referenceRows[turn.Index]
	if len(document.Rows) != len(want) {
		t.Fatalf("第 %d 轮行数应为 %d，实际 %d", turn.Index, len(want), len(document.Rows))
	}
	for index, row := range document.Rows {
		if row.Kind != want[index].Kind || row.Text != want[index].Text {
			t.Fatalf("第 %d 轮第 %d 行不一致: %+v vs %+v", turn.Index, index, row, want[index])
		}
		if row.ID != want[index].ID {
			t.Fatalf("第 %d 轮第 %d 行标识不一致: %q vs %q", turn.Index, index, row.ID, want[index].ID)
		}
	}
	// 轮索引里也该有这一轮。
	found := false
	for _, ref := range reference.Turns {
		if int64(ref.Index) == turn.Index {
			found = true
		}
	}
	if !found {
		t.Fatalf("第 %d 轮不在轮索引里", turn.Index)
	}
}

// TestTurnHistoryStartsFromNewest 验证首屏只给最近一轮，而且同一游标重复读取得到同一页。
func TestTurnHistoryStartsFromNewest(t *testing.T) {
	events := turnFixture(3)
	source, err := NewTurnHistory("s-1", newFakeHistoryReader(events))
	if err != nil {
		t.Fatalf("新建按轮历史来源: %v", err)
	}

	got, err := source.ReadHistory(context.Background(), HistoryRequest{SessionID: "s-1", Limit: 1})
	if err != nil {
		t.Fatalf("读取首屏: %v", err)
	}
	if len(got.Turns) != 1 || got.Turns[0].Index != 3 {
		t.Fatalf("首屏应只有最近一轮: %+v", got.Turns)
	}
	if got.Done {
		t.Fatalf("还有更早的轮次，不该说到头了")
	}

	again, err := source.ReadHistory(context.Background(), HistoryRequest{SessionID: "s-1", Limit: 1})
	if err != nil {
		t.Fatalf("重复读取首屏: %v", err)
	}
	if !reflect.DeepEqual(got, again) {
		t.Fatalf("同一游标重复读取应得到同一页\n第一次: %+v\n第二次: %+v", got, again)
	}
}

// TestTurnHistoryRejectsForeignRequest 验证会话与分支不匹配时明确拒绝。
func TestTurnHistoryRejectsForeignRequest(t *testing.T) {
	source, err := NewTurnHistory("s-1", newFakeHistoryReader(turnFixture(1)))
	if err != nil {
		t.Fatalf("新建按轮历史来源: %v", err)
	}
	if _, err := source.ReadHistory(context.Background(), HistoryRequest{SessionID: "s-2"}); err != ErrTurnHistorySessionMismatch {
		t.Fatalf("别的会话应被拒绝，实际 %v", err)
	}
	if _, err := source.ReadHistory(context.Background(), HistoryRequest{SessionID: "s-1", BranchID: "main"}); err != nil {
		t.Fatalf("首次读取主线应成功: %v", err)
	}
	if _, err := source.ReadHistory(context.Background(), HistoryRequest{BranchID: "side"}); err != ErrTurnHistoryBranchMismatch {
		t.Fatalf("换分支应被拒绝，实际 %v", err)
	}
}

// TestTurnHistoryRequiresReaderAndSession 验证构造参数缺失时明确失败。
func TestTurnHistoryRequiresReaderAndSession(t *testing.T) {
	if _, err := NewTurnHistory("", newFakeHistoryReader(nil)); err != ErrTurnHistoryNoSession {
		t.Fatalf("缺会话标识应失败: %v", err)
	}
	if _, err := NewTurnHistory("s-1", nil); err != ErrTurnHistoryNoReader {
		t.Fatalf("缺读取来源应失败: %v", err)
	}
}
