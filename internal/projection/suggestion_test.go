package projection

import (
	"encoding/json"
	"testing"
)

// suggestionNotice 造一条提示建议的会话通知。
func suggestionNotice(t *testing.T, branchID, kind, data string) json.RawMessage {
	t.Helper()
	return noticeRaw(t, map[string]any{
		"sessionId": "s-1",
		"branchId":  branchID,
		"source":    "PromptSuggestion",
		"kind":      kind,
		"data":      jsonRaw(data),
	})
}

// TestSuggestionOfferAndClear 验证提示建议从 offer 通知来、由 cleared 通知结算。
func TestSuggestionOfferAndClear(t *testing.T) {
	folder := newFolder(t)

	rejections := folder.ApplyNotice("session.notice", suggestionNotice(t, mainBranch, "offer", `{"turnId":"t1","text":"要不要继续"}`))
	if len(rejections) > 0 {
		t.Fatalf("提示建议通知被丢弃: %+v", rejections[0])
	}
	view := folder.View()
	if view.Suggestion == nil || view.Suggestion.TurnID != "t1" || view.Suggestion.Text != "要不要继续" {
		t.Fatalf("没有折出提示建议: %+v", view.Suggestion)
	}

	// 另一轮的 cleared 不清理这一条。
	folder.ApplyNotice("session.notice", suggestionNotice(t, mainBranch, "cleared", `{"turnId":"t2","reason":"turn_started"}`))
	if folder.View().Suggestion == nil {
		t.Fatalf("别的轮的 cleared 不该清掉这一条")
	}

	// 同一轮的 cleared 清掉。
	folder.ApplyNotice("session.notice", suggestionNotice(t, mainBranch, "cleared", `{"turnId":"t1","reason":"accepted"}`))
	if folder.View().Suggestion != nil {
		t.Fatalf("cleared 之后不该再有建议: %+v", folder.View().Suggestion)
	}
}

// TestSuggestionIsolation 验证别的来源、别的会话与断线都不留建议。
func TestSuggestionIsolation(t *testing.T) {
	folder := newFolder(t)

	// 别的来源的通知本包不看。
	other := noticeRaw(t, map[string]any{
		"sessionId": "s-1",
		"branchId":  mainBranch,
		"source":    "OtherCapability",
		"kind":      "offer",
		"data":      jsonRaw(`{"turnId":"t1","text":"不该进来"}`),
	})
	folder.ApplyNotice("session.notice", other)
	if folder.View().Suggestion != nil {
		t.Fatalf("别的来源不该产生建议")
	}

	// 别的会话的通知被拒绝。
	foreign := noticeRaw(t, map[string]any{
		"sessionId": "s-2",
		"source":    "PromptSuggestion",
		"kind":      "offer",
		"data":      jsonRaw(`{"turnId":"t1","text":"x"}`),
	})
	rejections := folder.ApplyNotice("session.notice", foreign)
	if len(rejections) != 1 || rejections[0].Reason != ReasonSessionMismatch {
		t.Fatalf("别的会话的通知应被拒绝: %+v", rejections)
	}
	if folder.View().Suggestion != nil {
		t.Fatalf("被拒绝的通知不该进视图")
	}

	// 断线清空：服务端不重放建议。
	folder.ApplyNotice("session.notice", suggestionNotice(t, mainBranch, "offer", `{"turnId":"t3","text":"再来一条"}`))
	if folder.View().Suggestion == nil {
		t.Fatalf("应先有建议")
	}
	folder.OnDisconnect(mainBranch)
	if folder.View().Suggestion != nil {
		t.Fatalf("断线后建议应清空")
	}
}

// TestSuggestionPerBranch 验证建议按分支分开。
func TestSuggestionPerBranch(t *testing.T) {
	folder := newFolder(t)
	mustNotice(t, folder, mainBranch, ev("b1", 1, "branch", "create", `{"branch":"side","at":{"sequence":1}}`))
	folder.ApplyNotice("session.notice", suggestionNotice(t, "side", "offer", `{"turnId":"t9","text":"分支的建议"}`))

	if folder.View().Suggestion != nil {
		t.Fatalf("主线不该看到分支的建议")
	}
	side, ok := folder.BranchView("side")
	if !ok || side.Suggestion == nil || side.Suggestion.Text != "分支的建议" {
		t.Fatalf("分支视图没有建议: %+v", side.Suggestion)
	}
}
