package ui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是提问卡（全局 D1）的可执行验收：头/体/脚三段、进度徽标、逐题切换、
// 选项与自由输入、必填校验、提交失败的重试、键位与倒计时行。
//
// 事实走 Source.Question()：一批问题逐题展示、一次提交；提交走 Intents。

// stubQuestionIntents 接住提问的提交与拒绝，其余方法走嵌入的 nil 接口（被调用
// 即 panic，说明测试点到了一个没接的动作）。
type stubQuestionIntents struct {
	stubBaseIntents
	answers  chan string
	declines chan struct{}
	failNext bool
}

func (s *stubQuestionIntents) RespondQuestion(_ context.Context, answer string) error {
	if s.failNext {
		s.failNext = false
		return errors.New("写入没有完成")
	}
	s.answers <- answer
	return nil
}

func (s *stubQuestionIntents) SkipQuestion(context.Context) error {
	s.declines <- struct{}{}
	return nil
}

// questionTester 连上 Source 造一张提问卡；intents 为 nil 时只验渲染侧。
func questionTester(
	t *testing.T, snap QuestionSnapshot, intents Intents,
) (*App, *gui.Tester) {
	t.Helper()
	src := wiredSource()
	src.hasQ = true
	src.question = snap
	app := NewApp(Options{State: StateDefault, Source: src, Intents: intents})
	return app, gui.NewTester(app.View, WindowWidth, WindowHeight)
}

// threeQuestionBatch 是一批三题：单选、多选、自由输入各一道。
func threeQuestionBatch() QuestionSnapshot {
	return QuestionSnapshot{ID: "q-batch", Questions: []QuestionItemSnapshot{
		{
			ID: "env", Title: "选择部署目标", Text: "这次部署到哪个环境？",
			Options: []string{"暂存", "生产"},
		},
		{
			ID: "checks", Title: "选择检查项", Options: []string{"构建", "单测", "静态检查"},
			Multiple: true,
		},
		{ID: "note", Text: "补充说明"},
	}}
}

// questionSettled 等一次提交的写入结果回到渲染线程：busy 落回 false 且有错误
// 行时返回真（失败分支）。
func questionSettled(tt *gui.Tester, app *App, limit time.Duration) bool {
	deadline := time.Now().Add(limit)
	for time.Now().Before(deadline) {
		tt.Frame()
		if !app.question.busy && app.question.err != "" {
			return true
		}
		time.Sleep(5 * time.Millisecond)
	}
	return false
}

// TestQuestionCardBatch：三段结构、进度徽标、逐题切换、必填校验与一次提交。
func TestQuestionCardBatch(t *testing.T) {
	stub := &stubQuestionIntents{answers: make(chan string, 1), declines: make(chan struct{}, 1)}
	app, tt := questionTester(t, threeQuestionBatch(), stub)

	// 头 / 体 / 脚：标题、进度徽标与第一题的选项都在；首题没有「上一题」。
	for _, want := range []string{"选择部署目标", "1/3", "这次部署到哪个环境？", "暂存", "生产", "下一题"} {
		if !tt.HasText(want) {
			t.Fatalf("提问卡缺少 %q：\n%s", want, p0WindowText(tt))
		}
	}
	if tt.HasText("上一题") {
		t.Fatalf("首题不该有「上一题」：\n%s", p0WindowText(tt))
	}
	// 单题且未答时「提交」禁用；这里是多题，末题才出现「提交」。
	if tt.HasText("提交") {
		t.Fatalf("非末题不该有「提交」：\n%s", p0WindowText(tt))
	}

	// 选一项，切到第二题：单选互斥、进度加一、已答内容保留。
	if err := tt.Click("暂存"); err != nil {
		t.Fatalf("点不到选项：%v", err)
	}
	tt.Frame()
	if got := app.question.selected[0]; !got["暂存"] || len(got) != 1 {
		t.Fatalf("单选状态 = %v，想要只有「暂存」", got)
	}
	if err := tt.Click("下一题"); err != nil {
		t.Fatalf("点不到下一题：%v", err)
	}
	tt.Frame()
	if !tt.HasText("2/3") || !tt.HasText("上一题") {
		t.Fatalf("第二题的头不对：\n%s", p0WindowText(tt))
	}
	// 多选：两项可以同时勾上。
	if err := tt.Click("构建"); err != nil {
		t.Fatalf("点不到复选项：%v", err)
	}
	tt.Frame()
	if err := tt.Click("单测"); err != nil {
		t.Fatalf("点不到复选项：%v", err)
	}
	tt.Frame()
	if got := app.question.selected[1]; !got["构建"] || !got["单测"] {
		t.Fatalf("多选状态 = %v，想要「构建」与「单测」都勾上", got)
	}

	// 末题：自由输入为空时提交给本地校验错误，不发写入。
	if err := tt.Click("下一题"); err != nil {
		t.Fatalf("点不到下一题：%v", err)
	}
	tt.Frame()
	if !tt.HasText("3/3") || !tt.HasText(questionCopyTitleFallback) {
		t.Fatalf("末题的标题回退不对：\n%s", p0WindowText(tt))
	}
	if err := tt.Click("提交"); err != nil {
		t.Fatalf("点不到提交：%v", err)
	}
	tt.Frame()
	if !tt.HasText(questionCopyRequired) {
		t.Fatalf("必填未答时没有本地校验错误：\n%s", p0WindowText(tt))
	}
	select {
	case got := <-stub.answers:
		t.Fatalf("必填未答时不该发写入，实际发出 %q", got)
	default:
	}

	// 自由输入填上后提交：一次发出整批，多题按「题目：答案」逐行拼接。
	if err := tt.Click(questionCopyFreeform); err != nil {
		t.Fatalf("点不到自由输入框：%v", err)
	}
	tt.Type("值班同学")
	tt.Frame()
	if err := tt.Click("提交"); err != nil {
		t.Fatalf("点不到提交：%v", err)
	}
	tt.Frame()
	select {
	case got := <-stub.answers:
		for _, want := range []string{
			"选择部署目标：暂存", "选择检查项：构建\n单测", questionCopyTitleFallback + "：值班同学",
		} {
			if !strings.Contains(got, want) {
				t.Fatalf("提交的答案缺 %q：%q", want, got)
			}
		}
	case <-time.After(2 * time.Second):
		t.Fatal("提交后没有发出写入")
	}
	// 答过之后卡片不再画（等结算时输入区回到正常形态）。
	tt.Frame()
	if tt.HasText(questionCopyTitleFallback) {
		t.Fatalf("答过的提问仍在屏上：\n%s", p0WindowText(tt))
	}
}

// TestQuestionCardFailureAndKeys：提交失败留在卡片里给原因并可重试；Esc 拒绝、
// ↵ 提交（焦点不在文字输入里）。
func TestQuestionCardFailureAndKeys(t *testing.T) {
	snap := QuestionSnapshot{ID: "q-one", Questions: []QuestionItemSnapshot{{
		ID: "go", Title: "确认继续", Options: []string{"继续", "停下"},
	}}}
	stub := &stubQuestionIntents{answers: make(chan string, 1), declines: make(chan struct{}, 1)}
	stub.failNext = true
	app, tt := questionTester(t, snap, stub)

	if err := tt.Click("继续"); err != nil {
		t.Fatalf("点不到选项：%v", err)
	}
	tt.Frame()
	if err := tt.Click("提交"); err != nil {
		t.Fatalf("点不到提交：%v", err)
	}
	tt.Frame()
	// 失败：卡片保留、错误行给原因、按钮恢复可点。
	if !questionSettled(tt, app, 2*time.Second) {
		t.Fatalf("提交失败后按钮没有恢复：busy=%v err=%q", app.question.busy, app.question.err)
	}
	if !tt.HasText("写入没有完成") {
		t.Fatalf("失败原因没有写进卡片：\n%s", p0WindowText(tt))
	}
	// 再按一次回车重试：同一条请求再发一遍。
	tt.Key(0, gui.KeyEnter)
	tt.Frame()
	select {
	case got := <-stub.answers:
		if got != "继续" {
			t.Fatalf("重试发出的答案 = %q，想要「继续」", got)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("回车没有重试提交")
	}

	// Esc 拒绝：走取消通道，不当作拒绝写入。
	app2, tt2 := questionTester(t, snap, stub)
	tt2.Frame()
	tt2.Key(0, gui.KeyEscape)
	select {
	case <-stub.declines:
	case <-time.After(2 * time.Second):
		t.Fatal("Esc 没有走「拒绝」通道")
	}
	tt2.Frame()
	if !app2.questionAnswered("q-one") {
		t.Fatal("拒绝后没有记下已答请求")
	}
}

// TestQuestionCardTimeoutHint：快照带截止时间时头部有「剩余 m:ss」倒计时行。
func TestQuestionCardTimeoutHint(t *testing.T) {
	snap := QuestionSnapshot{
		ID:        "q-time",
		Questions: []QuestionItemSnapshot{{Title: "限时确认", Options: []string{"好"}}},
		Deadline:  time.Now().Add(95 * time.Second),
	}
	_, tt := questionTester(t, snap, nil)
	if !tt.HasText("剩余 1:3") {
		t.Fatalf("没有倒计时行：\n%s", p0WindowText(tt))
	}
	var timer string
	for _, text := range tt.Texts() {
		if strings.HasPrefix(text, "剩余 ") {
			timer = text
			break
		}
	}
	if info, ok := tt.Access(timer); !ok || info.Role != gui.RoleTimer {
		t.Fatalf("倒计时 %q 的可读项 = %+v（ok=%v），想要 role=timer", timer, info, ok)
	}
	// 已过截止时间：换成一句状态，不显示负数。
	if got := questionClock(-time.Second); got != questionCopyTimeout {
		t.Fatalf("到点的文案 = %q，想要 %q", got, questionCopyTimeout)
	}
}

// TestQuestionCardFilterAndCollapse：选项多于 8 个时出现筛选框，筛掉全部时给
// 空态句；「收起」把体部与脚部藏起来，按钮文案变「展开」。
func TestQuestionCardFilterAndCollapse(t *testing.T) {
	options := []string{"a1", "a2", "a3", "a4", "a5", "a6", "a7", "a8", "b1"}
	snap := QuestionSnapshot{ID: "q-many", Questions: []QuestionItemSnapshot{{
		Title: "挑一个", Options: options,
	}}}
	app, tt := questionTester(t, snap, nil)
	if !tt.HasText(questionCopyFilter) {
		t.Fatalf("选项多于 8 个时没有筛选框：\n%s", p0WindowText(tt))
	}
	if err := tt.Click(questionCopyFilter); err != nil {
		t.Fatalf("点不到筛选框：%v", err)
	}
	tt.Type("zzz")
	tt.Frame()
	if !tt.HasText(questionCopyNoMatch) {
		t.Fatalf("筛选无匹配时没有空态句：\n%s", p0WindowText(tt))
	}
	if err := tt.Click(questionCopyCollapse); err != nil {
		t.Fatalf("点不到收起：%v", err)
	}
	tt.Frame()
	if !app.question.collapsed || !tt.HasText(questionCopyExpand) {
		t.Fatalf("收起后没有换成「展开」：\n%s", p0WindowText(tt))
	}
	if tt.HasText(questionCopyNoMatch) {
		t.Fatalf("收起后体部还画着：\n%s", p0WindowText(tt))
	}
}
