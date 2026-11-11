package projection

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/daemon"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/appserver/session"
)

// TestRealDaemonLiveAndHistoryAgree 在真实 App Server 上跑一轮对话，把"实时折叠"
// 与"重读历史后折叠"对照。没有真实服务端时跳过：证据标 blocked，不用替身冒充。
//
// 用法：
//
//	DIM_APP_SERVER_BIN=<App Server 可执行文件> \
//	  go test -run TestRealDaemonLiveAndHistoryAgree -v ./internal/projection/
//
// 老版本服务端需要 `app-server` 子命令，可用 DIM_APP_SERVER_ARGS=app-server 传入。
func TestRealDaemonLiveAndHistoryAgree(t *testing.T) {
	serverPath := os.Getenv("DIM_APP_SERVER_BIN")
	if serverPath == "" {
		t.Skip("未设置 DIM_APP_SERVER_BIN：真实 daemon 证据 blocked（缺 App Server 可执行文件）")
	}
	var serverArgs []string
	if args := os.Getenv("DIM_APP_SERVER_ARGS"); args != "" {
		serverArgs = append(serverArgs, args)
	}

	// 端点路径有长度上限：临时目录由 evidenceTempDir 放在系统临时根下，可用 DIM_EVIDENCE_TMP 收短。
	dataDir := evidenceTempDir(t, "d2proj-")
	t.Cleanup(func() { _ = os.RemoveAll(dataDir) })
	workDir := filepath.Join(dataDir, "workspace")
	if err := os.MkdirAll(workDir, 0o700); err != nil {
		t.Fatalf("创建工作目录: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	supervisor, err := daemon.New(daemon.Config{
		DataDir:    dataDir,
		ServerPath: serverPath,
		ServerArgs: serverArgs,
		MockModel:  true,
	})
	if err != nil {
		t.Fatalf("创建监督者: %v", err)
	}
	conn, err := supervisor.Connect(ctx)
	if err != nil {
		t.Fatalf("连接服务端: %v", err)
	}
	t.Logf("服务端 %s（protocol %d，profile %s，owned=%v）",
		conn.Hello.ServerVersion, conn.Hello.ProtocolVersion, conn.Hello.DaemonProfile.Kind, conn.Owned)

	var created protocol.SessionCreateResult
	if err := conn.Client.CallInto(ctx, "session.create", map[string]any{
		"cwd":   workDir,
		"tools": []string{},
	}, &created); err != nil {
		t.Fatalf("创建会话: %v", err)
	}
	t.Logf("会话 %s", created.SessionID)

	live := newFolderFor(t, created.SessionID)
	manager, err := session.NewManager(session.Config{
		Opener:            supervisor,
		ReconnectAttempts: 3,
		ReconnectDelay:    200 * time.Millisecond,
	})
	if err != nil {
		t.Fatalf("创建会话流管理器: %v", err)
	}
	t.Cleanup(manager.Close)

	turnEnded := make(chan struct{}, 1)
	base := Listen(live, func(rejection Rejection) {
		t.Logf("丢弃事实: %+v", rejection)
	})
	base.OnEvent = func(event session.Event) {
		if rejections := live.ApplyListenerEvent(event); len(rejections) > 0 {
			t.Logf("实时折叠丢弃: %+v", rejections[0])
		}
		if isTurnEnd(t, event) {
			select {
			case turnEnded <- struct{}{}:
			default:
			}
		}
	}
	base.OnResync = func(session.Resync) { live.OnDisconnect(mainBranch) }

	snapshot, lease, err := manager.Acquire(ctx, created.SessionID, base)
	if err != nil {
		t.Fatalf("打开会话: %v", err)
	}
	defer lease.Release()
	if rejections := live.ApplySnapshot(snapshot); len(rejections) > 0 {
		t.Logf("首屏折叠丢弃: %+v", rejections[0])
	}
	t.Logf("首屏: head=%d events=%d through=%d done=%v", snapshot.Head, len(snapshot.History.Events), snapshot.History.Through, snapshot.History.Done)

	for turn := 1; turn <= 2; turn++ {
		submitTurn(t, ctx, conn.Client, created.SessionID,
			fmt.Sprintf("第 %d 轮：用一句话说明这个工作目录里有什么", turn), turnEnded)
		time.Sleep(300 * time.Millisecond)
	}

	liveView := live.View()
	t.Logf("实时折叠: %d 行（未落库 %d），%d 轮", len(liveView.Rows), liveView.Unlanded, len(liveView.Turns))

	// 重读历史：另开一个折叠器，只吃历史，作为对照。
	var after int64
	limit := 1000
	page, err := lease.Subscription().History(ctx, protocol.SessionHistoryParams{
		BranchID: mainBranch,
		After:    &after,
		Limit:    &limit,
	})
	if err != nil {
		t.Fatalf("重读历史: %v", err)
	}
	historyFolder := newFolderFor(t, created.SessionID)
	if rejections := historyFolder.ApplyHistory(mainBranch, page); len(rejections) > 0 {
		t.Logf("历史折叠丢弃: %+v", rejections[0])
	}
	historyView := historyFolder.View()
	t.Logf("历史折叠: %d 行（%d 条事实，through=%d）", len(historyView.Rows), len(page.Events), page.Through)

	if len(liveView.Rows) == 0 {
		t.Fatalf("实时折叠没有得到任何行")
	}
	if !reflect.DeepEqual(fingerprint(liveView), fingerprint(historyView)) {
		t.Fatalf("实时与历史的行内容不一致\n实时: %+v\n历史: %+v", fingerprint(liveView), fingerprint(historyView))
	}
	if !reflect.DeepEqual(liveView.Turns, historyView.Turns) {
		t.Fatalf("轮索引不一致\n实时: %+v\n历史: %+v", liveView.Turns, historyView.Turns)
	}
	if !reflect.DeepEqual(liveView.Goal, historyView.Goal) {
		t.Fatalf("目标不一致\n实时: %+v\n历史: %+v", liveView.Goal, historyView.Goal)
	}
	if !reflect.DeepEqual(liveView.Pending, historyView.Pending) {
		t.Fatalf("未完成事项不一致\n实时: %+v\n历史: %+v", liveView.Pending, historyView.Pending)
	}

	// 同一批事实再走一次历史（相当于断线重读）：尾部落库、身份补齐、不重复。
	if rejections := live.ApplyHistory(mainBranch, page); len(rejections) > 0 {
		t.Logf("重读折叠丢弃: %+v", rejections[0])
	}
	landedView := live.View()
	if len(landedView.Rows) != len(liveView.Rows) {
		t.Fatalf("落库后行数变了: %d -> %d", len(liveView.Rows), len(landedView.Rows))
	}
	if landedView.Unlanded != 0 {
		t.Fatalf("重读后仍有未落库行: %d", landedView.Unlanded)
	}
	assertSameContent(t, "重读后", landedView, historyView)
	for index, id := range rowIDs(historyView) {
		if id == "" {
			t.Fatalf("历史第 %d 行没有服务端标识", index)
		}
	}

	kinds := make(map[string]struct{}, len(page.Events))
	for _, event := range page.Events {
		kinds[event.Type+"/"+event.Name] = struct{}{}
	}
	eventKinds := make([]string, 0, len(kinds))
	for kind := range kinds {
		eventKinds = append(eventKinds, kind)
	}
	sort.Strings(eventKinds)
	rowKinds := make([]string, 0, len(historyView.Rows))
	for _, row := range historyView.Rows {
		rowKinds = append(rowKinds, string(row.Kind))
	}

	// 真实分页：最近一轮打开，再逐页向前翻到分支起点，结果应与一次读全一致。
	paged := newFolderFor(t, created.SessionID)
	latest := true
	pageLimit := 6
	firstPage, err := lease.Subscription().History(ctx, protocol.SessionHistoryParams{
		BranchID:   mainBranch,
		LatestTurn: &latest,
		Limit:      &pageLimit,
	})
	if err != nil {
		t.Fatalf("读取最近一轮: %v", err)
	}
	paged.ApplyFirstPage(mainBranch, firstPage)
	pages := 1
	for pages < 10 {
		request, ok := paged.OlderPageRequest(mainBranch, pageLimit)
		if !ok {
			break
		}
		next, err := lease.Subscription().History(ctx, request)
		if err != nil {
			t.Fatalf("向前翻页: %v", err)
		}
		if rejections := paged.ApplyOlderPage(mainBranch, next); len(rejections) > 0 {
			t.Logf("翻页折叠丢弃: %+v", rejections[0])
		}
		pages++
	}
	pagedView := paged.View()
	t.Logf("向前翻页: %d 页，%d 行，锚点 %d，hasOlder=%v",
		pages, len(pagedView.Rows), pagedView.Anchor, pagedView.HasOlder)
	if pagedView.HasOlder {
		t.Fatalf("翻页没有到达分支起点")
	}
	assertSameContent(t, "翻页", pagedView, historyView)

	// 导出式按轮读取：同一个会话流当历史来源，每页一轮，读到底。
	exporter, err := NewTurnHistory(created.SessionID, lease.Subscription())
	if err != nil {
		t.Fatalf("创建按轮历史来源: %v", err)
	}
	referenceRows := testTurnBuckets(historyView)
	var exported []HistoryTurn
	exportPages := 0
	exportBefore := int64(0)
	for {
		if exportPages > 6 {
			t.Fatalf("导出式读取没有收敛")
		}
		exportPage, err := exporter.ReadHistory(ctx, HistoryRequest{
			SessionID: created.SessionID,
			BranchID:  mainBranch,
			Before:    exportBefore,
			Limit:     1,
		})
		if err != nil {
			t.Fatalf("导出式读取第 %d 页: %v", exportPages+1, err)
		}
		exportPages++
		// 一页之内按轮次号升序；页与页之间往更早的轮次走。
		for index, turn := range exportPage.Turns {
			if index > 0 && exportPage.Turns[index-1].Index >= turn.Index {
				t.Fatalf("一页内轮次必须升序: %+v", exportPage.Turns)
			}
		}
		exported = append(exported, exportPage.Turns...)
		if exportPage.Done {
			break
		}
		if exportPage.NextBefore == 0 {
			t.Fatalf("导出式读取没到头却没有游标")
		}
		exportBefore = exportPage.NextBefore
	}
	t.Logf("导出式读取: %d 页，%d 轮", exportPages, len(exported))
	if len(exported) != len(historyView.Turns) {
		t.Fatalf("导出轮次应为 %d，实际 %d", len(historyView.Turns), len(exported))
	}
	seenTurns := make(map[int64]bool, len(exported))
	for _, turn := range exported {
		if turn.Index < 1 || turn.Index > int64(len(historyView.Turns)) {
			t.Fatalf("导出轮次号越界: %d", turn.Index)
		}
		if seenTurns[turn.Index] {
			t.Fatalf("导出轮次 %d 出现了两次", turn.Index)
		}
		seenTurns[turn.Index] = true
		assertTurnDocument(t, turn, historyView, referenceRows)
	}
	for index := 1; index <= len(historyView.Turns); index++ {
		if !seenTurns[int64(index)] {
			t.Fatalf("导出缺少第 %d 轮", index)
		}
	}

	evidence, err := json.Marshal(map[string]any{
		"serverVersion": conn.Hello.ServerVersion,
		"sessionId":     created.SessionID,
		"historyEvents": len(page.Events),
		"eventKinds":    eventKinds,
		"rows":          len(historyView.Rows),
		"pages":         pages,
		"exportPages":   exportPages,
		"exportTurns":   len(exported),
		"rowIDs":        rowIDs(historyView),
		"rowKinds":      rowKinds,
		"turns":         len(historyView.Turns),
		"landedRows":    len(landedView.Rows),
		"pendingTotal":  historyView.Pending.Total(),
	})
	if err != nil {
		t.Fatalf("证据编码失败: %v", err)
	}
	t.Logf("证据: %s", evidence)

	// 先放掉持有者再回收自启实例：没有附加会话时服务端才会自己停。
	lease.Release()
	manager.Close()
	if result, err := supervisor.Shutdown(ctx); err != nil {
		t.Logf("回收自启服务端失败: %v", err)
	} else {
		t.Logf("回收自启服务端: stopped=%v reason=%q", result.Stopped, result.Reason)
	}
}

// assertSameContent 比较两份视图的内容。分页位置（锚点、hasOlder）不在比较范围内：
// 它随读取路径不同而变化，不属于内容。
func assertSameContent(t *testing.T, label string, got, want View) {
	t.Helper()
	if !reflect.DeepEqual(rowIDs(got), rowIDs(want)) {
		t.Fatalf("%s的行标识不一致\n得到: %v\n期望: %v", label, rowIDs(got), rowIDs(want))
	}
	if !reflect.DeepEqual(fingerprint(got), fingerprint(want)) {
		t.Fatalf("%s的行内容不一致\n得到: %+v\n期望: %+v", label, fingerprint(got), fingerprint(want))
	}
	if !reflect.DeepEqual(got.Turns, want.Turns) {
		t.Fatalf("%s的轮索引不一致\n得到: %+v\n期望: %+v", label, got.Turns, want.Turns)
	}
	if !reflect.DeepEqual(got.Goal, want.Goal) {
		t.Fatalf("%s的目标不一致\n得到: %+v\n期望: %+v", label, got.Goal, want.Goal)
	}
	if !reflect.DeepEqual(got.Pending, want.Pending) {
		t.Fatalf("%s的未完成事项不一致\n得到: %+v\n期望: %+v", label, got.Pending, want.Pending)
	}
	if len(got.Rows) != len(want.Rows) {
		t.Fatalf("%s的行数不一致: %d vs %d", label, len(got.Rows), len(want.Rows))
	}
}

// submitTurn 提交一轮并等到这一轮结束。
func submitTurn(t *testing.T, ctx context.Context, client interface {
	CallInto(context.Context, string, any, any) error
}, sessionID, text string, turnEnded <-chan struct{}) {
	t.Helper()
	var submitted struct {
		Event protocol.RawEvent `json:"event"`
	}
	if err := client.CallInto(ctx, "session.submit", map[string]any{
		"sessionId":    sessionID,
		"branchId":     mainBranch,
		"submissionId": newUUID(t),
		"mode":         "queue",
		"text":         text,
	}, &submitted); err != nil {
		t.Fatalf("提交一轮: %v", err)
	}
	t.Logf("提交成功: 输入事件 %s（序号 %d）", submitted.Event.ID, submitted.Event.Sequence)
	select {
	case <-turnEnded:
	case <-ctx.Done():
		t.Fatalf("等待 turn/end 超时")
	}
}

// newUUID 造一个 v4 UUID：服务端要求提交标识是 UUID，重复提交以它为幂等键。
func newUUID(t *testing.T) string {
	t.Helper()
	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		t.Fatalf("生成提交标识: %v", err)
	}
	raw[6] = (raw[6] & 0x0f) | 0x40
	raw[8] = (raw[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", raw[0:4], raw[4:6], raw[6:8], raw[8:10], raw[10:16])
}

// isTurnEnd 判断一条通知是不是一轮结束。
func isTurnEnd(t *testing.T, event session.Event) bool {
	t.Helper()
	if event.Method != "session.event" {
		return false
	}
	var params protocol.SessionEventParams
	if err := json.Unmarshal(event.Params, &params); err != nil {
		return false
	}
	return params.Event.Type == "turn" && params.Event.Name == "end"
}

// newFolderFor 建一个指定会话标识的折叠器。
func newFolderFor(t *testing.T, sessionID string) *Folder {
	t.Helper()
	folder, err := New(Config{SessionID: sessionID})
	if err != nil {
		t.Fatalf("新建折叠器: %v", err)
	}
	return folder
}
