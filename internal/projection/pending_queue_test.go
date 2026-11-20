package projection

import "testing"

// 本文件钉住一条真实回执的形状：`session.input.pending` 除了用户排队输入，还会
// 列出服务端自己注入的环境/状态上下文块——投递方式 append、`user` 为 null、
// 没有正文也没有附件、多数已经 applied。它们不是用户排队的一条消息，不能进
// 队列清单（否则输入区上方会出现一整列「没有正文」）。
//
// 回执样例（真实 alpha.18，六次接入后累积到十一条）：
//
//	{"inputId":"…","sequence":2,"mode":"append","user":null,"applied":true}
//
// 同一份事实按历史推导时也不会进队列：折叠 `input/message` 时三样都没有就不
// 算一次用户输入。装载口径必须与推导口径一致，本文件同时钉住两边。

// TestPendingInputsSkipContextBlocks 断言只有上下文块时队列清单为空：面板没有
// 可画的条目（界面因此不出现）。装载前后都是空。
func TestPendingInputsSkipContextBlocks(t *testing.T) {
	folder := newFolder(t)
	folder.SeedPendingInputs(mainBranch, 4, []PendingInput{
		{InputID: "c1", Sequence: 2, Mode: "append", Applied: true},
		{InputID: "c2", Sequence: 3, Mode: "append", Applied: true},
		{InputID: "c3", Sequence: 4, Mode: "append"},
	})
	view := folder.View()
	if len(view.Pending.Inputs) != 0 {
		t.Fatalf("上下文块不该进队列清单: %+v", view.Pending.Inputs)
	}
	if view.Pending.Total() != 0 {
		t.Fatalf("四类未完成事项总条数应为 0，实际 %d", view.Pending.Total())
	}
}

// TestPendingInputsKeepRealQueueRows 断言上下文块与真实排队输入混在一张回执里时，
// 队列清单只留真实的两条，顺序按序号，正文来自回执。
func TestPendingInputsKeepRealQueueRows(t *testing.T) {
	folder := newFolder(t)
	folder.SeedPendingInputs(mainBranch, 9, []PendingInput{
		{InputID: "c1", Sequence: 2, Mode: "append", Applied: true},
		{InputID: "q1", Sequence: 5, Mode: "queue", Text: "再补一条错误分支的测试", HasUser: true},
		{InputID: "c2", Sequence: 6, Mode: "append", Applied: true},
		{InputID: "q2", Sequence: 9, Mode: "steer", Text: "先把编译过的包跑一遍", HasUser: true},
	})
	inputs := folder.View().Pending.Inputs
	if len(inputs) != 2 {
		t.Fatalf("队列清单应为 2 条，实际 %d：%+v", len(inputs), inputs)
	}
	if inputs[0].InputID != "q1" || inputs[0].Text != "再补一条错误分支的测试" || inputs[0].Mode != "queue" {
		t.Fatalf("第一条不对: %+v", inputs[0])
	}
	if inputs[1].InputID != "q2" || inputs[1].Text != "先把编译过的包跑一遍" || inputs[1].Mode != "steer" {
		t.Fatalf("第二条不对: %+v", inputs[1])
	}
}

// TestPendingInputsKeepRowsWithoutText 断言「没有正文」的兜底仍只留给真实的用户
// 输入：带用户内容但正文取不出、只有附件、只有服务端正文摘要，三种都留在清单里。
func TestPendingInputsKeepRowsWithoutText(t *testing.T) {
	folder := newFolder(t)
	folder.SeedPendingInputs(mainBranch, 8, []PendingInput{
		{InputID: "u1", Sequence: 2, Mode: "queue", HasUser: true},
		{InputID: "u2", Sequence: 3, Mode: "queue", Attachments: 2},
		{InputID: "u3", Sequence: 4, Mode: "queue", Text: "完整正文在这里"},
	})
	inputs := folder.View().Pending.Inputs
	if len(inputs) != 3 {
		t.Fatalf("三条用户输入都该留下，实际 %d：%+v", len(inputs), inputs)
	}
	if inputs[0].Text != "" || inputs[0].Attachments != 0 || !inputs[0].HasUser {
		t.Fatalf("带用户内容的那条不对: %+v", inputs[0])
	}
	if inputs[1].Attachments != 2 {
		t.Fatalf("只有附件的那条不对: %+v", inputs[1])
	}
}

// TestDerivedContextBlocksStayOutOfQueue 断言按历史推导的口径与装载一致：受控点
// 之后的上下文块事实（`input/message` + `blocks` 里的 raw 状态块）同样不进队列。
func TestDerivedContextBlocksStayOutOfQueue(t *testing.T) {
	folder := newFolder(t)
	mustHistory(t, folder, mainBranch, page(4,
		ev("e1", 1, "input", "message", `{"blocks":[{"type":"raw","text":"<dim-state source=\"dim/environment/clients\">No client connection is attached.</dim-state>"}]}`),
		ev("e2", 2, "input", "applied", `{"input_id":"e1","mode":"append"}`),
		ev("e3", 3, "input", "message", `{"blocks":[{"type":"raw","text":"<dim-state source=\"dim/environment/clients\">1 client connection attached</dim-state>"}]}`),
		ev("e4", 4, "input", "applied", `{"input_id":"e3","mode":"append"}`),
	))
	if inputs := folder.View().Pending.Inputs; len(inputs) != 0 {
		t.Fatalf("推导出的上下文块不该进队列清单: %+v", inputs)
	}
}
