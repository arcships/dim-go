package projection

import (
	"encoding/json"
	"testing"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
)

// goalRecord 造一条承载目标记录的输入事实。
func goalRecord(t *testing.T, id string, sequence int64, body string) protocol.RawEvent {
	t.Helper()
	data, err := json.Marshal(map[string]any{
		"blocks": []any{
			map[string]any{"key": goalStateKey, "text": body},
		},
	})
	if err != nil {
		t.Fatalf("目标块编码失败: %v", err)
	}
	return ev(id, sequence, "input", "message", string(data))
}

// TestGoalDeleteRecomputes 验证目标按输入块折叠，输入被删除后按剩下的事实重算。
func TestGoalDeleteRecomputes(t *testing.T) {
	first := goalRecord(t, "g1", 1, "<dim-goal status=\"active\" judge=\"model\" op=\"set\">\n<objective>\n甲目标\n</objective>\n</dim-goal>")
	second := goalRecord(t, "g2", 2, "<dim-goal status=\"paused\" judge=\"model\" op=\"pause\" reason=\"user\">\n<objective>\n乙目标\n</objective>\n</dim-goal>")

	folder := newFolder(t)
	mustHistory(t, folder, mainBranch, page(2, first, second))

	view := folder.View()
	if view.Goal == nil || view.Goal.Objective != "乙目标" || view.Goal.Status != GoalPaused {
		t.Fatalf("最新记录应决定目标: %+v", view.Goal)
	}
	if view.Goal.Revision != 2 || view.Goal.InputID != "g2" {
		t.Fatalf("目标修订或来源不对: %+v", view.Goal)
	}
	if view.Goal.Reason != "user" {
		t.Fatalf("暂停原因没记下来: %+v", view.Goal)
	}

	// 删除承载最新记录的输入：目标回到上一条记录。
	mustNotice(t, folder, mainBranch, ev("d1", 3, "input", "deleted", `{"input_id":"g2"}`))
	view = folder.View()
	if view.Goal == nil || view.Goal.Objective != "甲目标" || view.Goal.Status != GoalActive {
		t.Fatalf("删除后没有重算到上一条记录: %+v", view.Goal)
	}
	if view.Goal.Revision != 1 {
		t.Fatalf("重算后的修订应为 1: %+v", view.Goal)
	}

	// 目标变更行随输入一起消失。
	goalRows := 0
	for _, row := range view.Rows {
		if row.Kind == RowGoalChange {
			goalRows++
		}
	}
	if goalRows != 1 {
		t.Fatalf("删除后应只剩一项目标变更行，实际 %d", goalRows)
	}

	// 再删掉最后一条：目标为空，本地不再保留。
	mustNotice(t, folder, mainBranch, ev("d2", 4, "input", "deleted", `{"input_id":"g1"}`))
	view = folder.View()
	if view.Goal != nil {
		t.Fatalf("目标应已清空: %+v", view.Goal)
	}
	for _, row := range view.Rows {
		if row.Kind == RowGoalChange {
			t.Fatalf("目标变更行应一并消失: %+v", row)
		}
	}
}

// TestGoalRecordRetireAndCleared 验证退休与清空记录都表示没有目标。
func TestGoalRecordRetireAndCleared(t *testing.T) {
	folder := newFolder(t)
	mustHistory(t, folder, mainBranch, page(1,
		goalRecord(t, "g1", 1, "<dim-goal status=\"active\" op=\"set\">\n<objective>\n甲目标\n</objective>\n</dim-goal>"),
	))
	if folder.View().Goal == nil {
		t.Fatalf("目标应存在")
	}

	cleared := goalRecord(t, "g2", 2, "<dim-goal status=\"cleared\" op=\"clear\">\n<objective>\n甲目标\n</objective>\n</dim-goal>")
	mustNotice(t, folder, mainBranch, cleared)
	if folder.View().Goal != nil {
		t.Fatalf("清空记录表示没有目标: %+v", folder.View().Goal)
	}

	// 被退休的记录同样不出目标，也不回退到更早的记录。
	retired := goalRecord(t, "g3", 3, "<dim-goal status=\"active\" op=\"set\">\n<objective>\n甲目标\n</objective>\n</dim-goal>")
	mustNotice(t, folder, mainBranch, retired)
	if folder.View().Goal == nil {
		t.Fatalf("重新写入后目标应存在")
	}
	retireEvent := retired
	retireEvent.ID = "g4"
	retireEvent.Sequence = 4
	retireEvent.CreateAt = fixtureBaseAt + 4
	retireEvent.Data = json.RawMessage(`{"blocks":[{"key":"goal:state","retire":true,"text":"<dim-goal status=\"active\" op=\"set\">\n<objective>\n甲目标\n</objective>\n</dim-goal>"}]}`)
	mustNotice(t, folder, mainBranch, retireEvent)
	if folder.View().Goal != nil {
		t.Fatalf("退休记录表示没有目标: %+v", folder.View().Goal)
	}
}

// TestGoalBlockedAndCompletedSummary 验证阻塞与达成时带上摘要。
func TestGoalBlockedAndCompletedSummary(t *testing.T) {
	folder := newFolder(t)
	mustHistory(t, folder, mainBranch, page(2,
		goalRecord(t, "g1", 1, "<dim-goal status=\"blocked\" op=\"blocked\">\n<objective>\n修好构建\n</objective>\n<blocker>\n缺依赖\n</blocker>\n</dim-goal>"),
	))
	view := folder.View()
	if view.Goal == nil || view.Goal.Status != GoalBlocked || view.Goal.Blocker != "缺依赖" {
		t.Fatalf("阻塞目标不对: %+v", view.Goal)
	}

	mustNotice(t, folder, mainBranch, goalRecord(t, "g2", 3, "<dim-goal status=\"completed\" op=\"completed\" by=\"evaluator\">\n<objective>\n修好构建\n</objective>\n<result>\n全部通过\n</result>\n</dim-goal>"))
	view = folder.View()
	if view.Goal == nil || view.Goal.Status != GoalComplete || view.Goal.Result != "全部通过" {
		t.Fatalf("达成目标不对: %+v", view.Goal)
	}
}

// TestGoalIgnoresContextOnlyInputs 验证没有目标块的上下文输入不产生目标变更行。
func TestGoalIgnoresContextOnlyInputs(t *testing.T) {
	folder := newFolder(t)
	mustHistory(t, folder, mainBranch, page(1,
		ev("c1", 1, "input", "message", `{"blocks":[{"type":"raw","text":"<dim-state>\n日期\n</dim-state>"}]}`),
	))
	view := folder.View()
	if len(view.Rows) != 0 || view.Goal != nil {
		t.Fatalf("上下文输入不该进时间线: %+v", view)
	}
}
