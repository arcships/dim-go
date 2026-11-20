package uiwire

import (
	"context"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
	"github.com/arcships/dimagent-mygo/internal/ui"
)

// 本文件是排队面板只画用户输入的回归断言。真实 `session.input.pending` 会把服务端
// 自己注入的环境/状态上下文块一起列出来：投递方式 append、`user` 为 null、没有
// 正文也没有附件、多数已 applied，每接入一次会话就多一两条，空闲会话里能累积到
// 十几条。它们不是用户排队的一条消息——旧实现把它们逐条画成「没有正文」，输入区
// 上方出现「18 条消息」。这里用同一份 wire 形状钉住修法：上下文块不进排队快照，
// 面板因为清单为空而不出现；真实排队两条时只画两行，正文来自回执。

// contextPendingInput 造一条服务端注入的环境/状态上下文块输入（真实回执同形）。
func contextPendingInput(id string, sequence int64) map[string]any {
	return map[string]any{
		"inputId": id, "sequence": sequence, "mode": "append",
		"user": nil, "applied": true,
	}
}

// contextQueuedScript 造一个排队回执：prefix 条上下文块在前，real 条真实排队在后。
//
// 历史用一段已经结算完的一轮（用户输入在序号 6 被消费）：排队清单只该反映服务端
// 此刻报的未消费输入，不再从历史里多推导出一条。
func contextQueuedScript(real int) fakeappserver.Script {
	events := append(e2eEvents(),
		e2eRawEvent("e6", 6, "input", "consumed", `{"input_id":"e1"}`),
	)
	inputs := make([]any, 0, 3+real)
	for index := 0; index < 3; index++ {
		inputs = append(inputs, contextPendingInput(
			"c"+string(rune('1'+index)),
			int64(6+index),
		))
	}
	if real > 0 {
		inputs = append(inputs,
			map[string]any{
				"inputId": "q1", "sequence": 9, "mode": "queue", "applied": false,
				"user": map[string]any{"content": []any{
					map[string]any{"type": "text", "text": "再补一条错误分支的测试"},
				}},
			},
			map[string]any{
				"inputId": "q2", "sequence": 10, "mode": "steer", "applied": true,
				"user": map[string]any{"content": []any{
					map[string]any{"type": "input_text", "text": "先把编译过的包跑一遍"},
				}},
			},
		)
	}
	return fakeappserver.Script{
		Features: []string{"session.history.latestTurn"},
		Sessions: []fakeappserver.Session{{
			ID:              e2eSession,
			LatestTurnStart: 1,
			Events:          events,
		}},
		Replies: []fakeappserver.Reply{
			{Method: "session.catalog", Result: map[string]any{
				"sessions": []any{}, "nextCursor": nil,
			}},
			{Method: "session.permission.get", Result: map[string]any{
				"version": 1, "policy": map[string]any{"preset": "full-access"},
			}},
			{Method: "session.input.pending", Result: map[string]any{
				"branchId": "main", "through": 10, "next": nil, "inputs": inputs,
			}},
		},
	}
}

// openQueuedSnapshot 打开会话并取快照。
func openQueuedSnapshot(t *testing.T, script fakeappserver.Script) ui.SessionSnapshot {
	t.Helper()
	server := startE2EServer(t, script)
	w := New(newE2EApp(t, server))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := w.Connect(ctx); err != nil {
		t.Fatalf("连接失败：%v", err)
	}
	pin := w.Pin("")
	if err := pin.OpenSession(ctx, e2eSession); err != nil {
		t.Fatalf("打开会话：%v", err)
	}
	snap, ok := pin.Session()
	if !ok {
		t.Fatal("会话已打开却没有快照")
	}
	return snap
}

// TestQueuedPanelHidesWhenOnlyContextInputsPending 断言待处理输入只有上下文块时
// 排队快照为空（界面据此不画面板），而不是列出若干条「没有正文」。
func TestQueuedPanelHidesWhenOnlyContextInputsPending(t *testing.T) {
	snap := openQueuedSnapshot(t, contextQueuedScript(0))
	if len(snap.Queued) != 0 {
		t.Fatalf("只有上下文块时排队快照应为空，实际 %d 条：%+v", len(snap.Queued), snap.Queued)
	}
}

// TestQueuedPanelKeepsOnlyRealQueueRows 断言混在同一张回执里的两条真实排队输入
// 留下来，顺序与正文都与回执一致；上下文块一条都不进快照。
func TestQueuedPanelKeepsOnlyRealQueueRows(t *testing.T) {
	snap := openQueuedSnapshot(t, contextQueuedScript(2))
	if len(snap.Queued) != 2 {
		t.Fatalf("排队快照应为 2 条，实际 %d 条：%+v", len(snap.Queued), snap.Queued)
	}
	if got := snap.Queued[0]; got.ID != "q1" || got.Mode != "queue" || got.Text != "再补一条错误分支的测试" {
		t.Fatalf("第一条不对：%+v", got)
	}
	if got := snap.Queued[1]; got.ID != "q2" || got.Mode != "steer" || got.Text != "先把编译过的包跑一遍" {
		t.Fatalf("第二条不对：%+v", got)
	}
	for _, queued := range snap.Queued {
		if queued.Text == "" && queued.Attachments == 0 {
			t.Fatalf("没有正文也没有附件的行不该出现在排队快照里：%+v", queued)
		}
	}
}
