package catalog

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
)

// TestAppServerBranchLifecycle 用假服务端核对分支创建与结束：切点按投影给的轮次
// 结束序号下发，新分支并入目录；结束请求带源会话与分支标识，状态先收敛为结束中。
func TestAppServerBranchLifecycle(t *testing.T) {
	server, client := startFakeServer(t, fakeappserver.Script{
		Replies: []fakeappserver.Reply{
			{Method: "session.branch_create", Result: map[string]any{
				"branch": map[string]any{
					"branchId":       "b-1",
					"parentBranchId": "main",
					"branchPoint":    int64(6),
					"createEvent":    int64(11),
				},
			}},
			{Method: "session.branch_end", Result: nil},
		},
	})
	harness := &appServerHarness{client: client}
	source := &fakeBranchSource{
		branches: []Branch{
			{ID: "main", Status: BranchOpen, Head: 10},
			{ID: "old", ParentID: "main", CutSequence: 4, Status: BranchEnded, Head: 7},
		},
		complete: true,
		points:   map[string]CutPoint{"main#2": {Status: CutReady, Sequence: 6}},
	}
	directory := newTestCatalog(t, Config{
		Reader:   harness,
		Writes:   newWriteManager(t, harness),
		Branches: source,
	})
	ctx := context.Background()
	if _, err := directory.RefreshBranches(ctx, "s1"); err != nil {
		t.Fatalf("读取分支目录: %v", err)
	}

	created, err := directory.CreateBranch(ctx, BranchCreateRequest{
		SessionID:      "s1",
		ParentBranchID: "main",
		TurnIndex:      2,
		Prompt:         "换一条路试试",
	})
	if err != nil {
		t.Fatalf("创建分支: %v", err)
	}
	if created.ID != "b-1" || created.CutSequence != 6 || created.Status != BranchOpen {
		t.Fatalf("新分支条目不符：%+v", created)
	}
	merged := directory.Branches("s1")
	if !reflect.DeepEqual(merged.Branches[1], source.branches[1]) {
		t.Fatalf("创建分支改动了已有条目：%+v", merged.Branches)
	}

	createRequests := server.RequestsFor("session.branch_create")
	if len(createRequests) != 1 {
		t.Fatalf("分支创建请求次数不符：%d", len(createRequests))
	}
	var createParams map[string]any
	if err := json.Unmarshal(createRequests[0].Params, &createParams); err != nil {
		t.Fatalf("分支创建参数无法解析: %v", err)
	}
	if createParams["sessionId"] != "s1" || createParams["parentBranchId"] != "main" ||
		createParams["at"] != float64(6) || createParams["prompt"] != "换一条路试试" {
		t.Fatalf("分支创建参数不符：%+v", createParams)
	}

	ended, err := directory.EndBranch(ctx, BranchEndRequest{SessionID: "s1", BranchID: "b-1"})
	if err != nil {
		t.Fatalf("结束分支: %v", err)
	}
	if ended.Status != BranchEnding {
		t.Fatalf("结束受理后的状态不符：%+v", ended)
	}
	endRequests := server.RequestsFor("session.branch_end")
	if len(endRequests) != 1 {
		t.Fatalf("结束分支请求次数不符：%d", len(endRequests))
	}
	var endParams map[string]any
	if err := json.Unmarshal(endRequests[0].Params, &endParams); err != nil {
		t.Fatalf("结束分支参数无法解析: %v", err)
	}
	if endParams["sessionId"] != "s1" || endParams["branchId"] != "b-1" {
		t.Fatalf("结束分支参数不符：%+v", endParams)
	}

	// 服务端收敛为已结束后，目录固定头部并拒绝再提交。
	if _, err := directory.ApplyBranchUpdate("s1", Branch{ID: "b-1", Status: BranchEnded, Head: 12}); err != nil {
		t.Fatalf("收敛分支状态: %v", err)
	}
	if err := directory.CheckBranchWritable("s1", "b-1"); !errors.Is(err, ErrBranchEnded) {
		t.Fatalf("已结束分支应拒绝提交：%v", err)
	}
}
