package catalog

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
)

// TestAppServerForkAddsSession 用假服务端核对分叉：请求带源会话与切点序号，回执
// 只有标识与序号时按标识单条读回，不重读整个目录。
func TestAppServerForkAddsSession(t *testing.T) {
	server, client := startFakeServer(t, fakeappserver.Script{
		Replies: []fakeappserver.Reply{
			{Method: "session.fork", Result: map[string]any{
				"sessionId": "s-fork-1",
				"sequence":  int64(9),
			}},
		},
	})
	harness := &appServerHarness{client: client}
	lookup := &fakeEntryLookup{entries: map[string]Entry{
		"s-fork-1": {
			ID:            "s-fork-1",
			Title:         "分叉标题",
			TitleSource:   TitleSourceManual,
			TitleRevision: 1,
			Workspace:     "/w",
		},
	}}
	catalog := newTestCatalog(t, Config{
		Reader: harness,
		Writes: newWriteManager(t, harness),
		Lookup: lookup,
	})

	forked, err := catalog.Fork(context.Background(), ForkRequest{SessionID: "s1", At: 7})
	if err != nil {
		t.Fatalf("分叉: %v", err)
	}
	if forked.ID != "s-fork-1" || forked.Title != "分叉标题" || forked.Incomplete {
		t.Fatalf("分叉条目不符：%+v", forked)
	}
	if _, ok := catalog.Entry("s-fork-1"); !ok {
		t.Fatal("分叉会话没有并入目录")
	}

	requests := server.RequestsFor("session.fork")
	if len(requests) != 1 {
		t.Fatalf("分叉请求次数不符：%d", len(requests))
	}
	var params map[string]any
	if err := json.Unmarshal(requests[0].Params, &params); err != nil {
		t.Fatalf("分叉参数无法解析: %v", err)
	}
	if params["sessionId"] != "s1" || params["at"] != float64(7) {
		t.Fatalf("分叉参数不符：%+v", params)
	}
	if catalogRequests := server.RequestsFor("session.catalog"); len(catalogRequests) != 0 {
		t.Fatalf("分叉不应重读整个目录：%d", len(catalogRequests))
	}
	mustEqualStrings(t, "单条读回", lookup.callList(), []string{"s-fork-1"})
}
