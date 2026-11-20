package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 提问卡（全局 D1、U3 §5.5）：elicitation 内联在输入区上沿，纵分头 / 体 / 脚三段。
//
// 事实来自 Source.Question()：一个请求带一批问题，界面逐题展示、一次提交。Go 侧
// 只保留可丢的视图状态（当前题、每题勾选与自定义草稿、筛选词、收起、提交态与
// 错误），不缓存事实、不重放已答。
//
// 提交走 Intents.RespondQuestion：wire 上的 answer 只有一段原文，批次里多题时按
// 「题目：答案」逐行拼接（见 questionBatchAnswer）；逐题结构化的应答形状 protocol
// v3 没有定义，界面不自己发明一个。

const (
	questionCopyTitleFallback = "需要你的回答"
	questionCopyFilter        = "筛选选项…"
	questionCopyNoMatch       = "没有匹配的选项"
	questionCopyCustom        = "自定义答案"
	questionCopyCustomHint    = "输入自定义答案"
	questionCopyCustomOption  = "自定义选项"
	questionCopyCustomOptHint = "每行输入一个自定义值"
	questionCopyFreeform      = "每行输入一个值"
	questionCopyRequired      = "请先回答所有必填问题"
	// questionCopyPending 是收起态状态行的未答文案（G-D1-05）。
	questionCopyPending = "待选择"
	// questionCopyNoItems 是空清单时的原因：提问请求在，却没有可作答的内容。
	questionCopyNoItems = "这条提问没有可作答的内容"
	// questionCopyStale 是已经答过 / 已经结算的提问再提交时的原因。
	questionCopyStale    = "这条提问已经提交过，不能再答"
	questionCopyDecline  = "拒绝"
	questionCopyPrevious = "上一题"
	questionCopyNext     = "下一题"
	questionCopySubmit   = "提交"
	questionCopyCollapse = "收起"
	questionCopyExpand   = "展开"
	questionCopyTimeout  = "已到作答截止时间"
	questionCardLabel    = "提问卡"

	// questionFilterThreshold 是出现筛选框的选项数下限（G-D1-10：>8 才出现，
	// 选项降到 8 个时它随之消失）。
	questionFilterThreshold = 8
	// questionBodyHeight 是卡片体部的固定高：切题时卡片不跳动（G-D1-01）。
	questionBodyHeight = 132
	// questionInputMinHeight / questionInputMaxHeight 是自定义答案框的高档：
	// 1 行起，4 行后内部滚动（G-D1-14）。
	questionInputMinHeight = 35
	questionInputMaxHeight = 79
	// questionCardWidth 是卡片内容的宽档：与审批卡、输入卡同一条内容列。
	questionCardWidth = composerMaxWidth
)

// questionCardState 是提问卡的可丢视图状态。整批换请求时清零；同一批里切题、
// 勾选与草稿都保留（G-D1-19/G-D1-20 的「已答内容保留」）。
type questionCardState struct {
	// requestID 是这份状态属于哪一次提问请求。
	requestID string
	// index 是当前题在批次里的位置。
	index int
	// selected 是每题的勾选项，custom 是每题的「自定义答案」，extra 是每题的
	// 「自定义选项」（多行，一行一个值）。
	selected []map[string]bool
	custom   []string
	extra    []string
	// filter 是筛选框的词；切题时清空。
	filter string
	// collapsed 是收起态：收起后只剩头部（G-D1-07）。
	collapsed bool
	// busy 是一次提交或拒绝正在写：脚部整组置灰。
	busy bool
	// err 是卡片自己的错误行（本地校验或服务端原文），在脚部之上。
	err string
	// buttonFocused 记上一帧焦点是否落在卡里的按钮上：落在按钮上时回车交给按钮，
	// 不再被窗口级的「提交」抢走。
	buttonFocused bool
}

// questionItems 取一批问题：快照带整批时用它，否则把单题字段当成一批。
func questionItems(snap QuestionSnapshot) []QuestionItemSnapshot {
	if len(snap.Questions) > 0 {
		return snap.Questions
	}
	if strings.TrimSpace(snap.Text) == "" && len(snap.Options) == 0 {
		return nil
	}
	return []QuestionItemSnapshot{{Text: snap.Text, Options: snap.Options}}
}

// questionTitle 是当前题的标题：字段标题 → 回退文案（G-D1-03）。
func questionTitle(item QuestionItemSnapshot) string {
	if title := strings.TrimSpace(item.Title); title != "" {
		return title
	}
	return questionCopyTitleFallback
}

// questionText 是卡片正文：显式给了 message 用它，没有时用题目本身。
func questionText(item QuestionItemSnapshot) string {
	text := strings.TrimSpace(item.Text)
	if text != "" {
		return text
	}
	return strings.TrimSpace(item.Description)
}

// questionAnswered 判断这一批是否已经本地答过：答过的请求不再画卡（等结算时
// 输入区回到正常形态，不闪一张空白卡）。
func (a *App) questionAnswered(requestID string) bool {
	return requestID != "" && a.questionAnsweredID == requestID
}

// syncQuestionCard 把视图状态对齐到这一批问题：换请求就清零，题数变了补槽位。
func (a *App) syncQuestionCard(snap QuestionSnapshot) []QuestionItemSnapshot {
	items := questionItems(snap)
	q := &a.question
	if q.requestID != snap.ID {
		*q = questionCardState{requestID: snap.ID}
	}
	for len(q.selected) < len(items) {
		q.selected = append(q.selected, map[string]bool{})
		q.custom = append(q.custom, "")
		q.extra = append(q.extra, "")
	}
	if q.index >= len(items) {
		q.index = len(items) - 1
	}
	if q.index < 0 {
		q.index = 0
	}
	return items
}

// questionSlot 保证第 index 题的三份槽位（勾选 / 自定义答案 / 自定义选项）都在，
// 并报告 index 是否落在 [0, count) 里。槽位缺就补齐：下标越界只让调用方给原因，
// 不让索引落到切片外。
func (q *questionCardState) questionSlot(index, count int) bool {
	if index < 0 || index >= count {
		return false
	}
	for len(q.selected) <= index {
		q.selected = append(q.selected, map[string]bool{})
	}
	for len(q.custom) <= index {
		q.custom = append(q.custom, "")
	}
	for len(q.extra) <= index {
		q.extra = append(q.extra, "")
	}
	return true
}

// questionCard 画提问卡：头（标题 / 进度 / 倒计时 / 收起）+ 体 + 脚。
func (a *App) questionCard(c *gui.Context, snap QuestionSnapshot) {
	t := c.Theme()
	items := a.syncQuestionCard(snap)
	q := &a.question
	q.buttonFocused = false
	if len(items) == 0 {
		// 有请求没有题：给一行可读原因，不留一张空卡（也不 panic）。
		gui.Column(c).WidthPercent(100).MaxWidth(questionCardWidth).Shrink(0).
			Padding(space4).Gap(space2).Background(t.Surface).
			Border(strokeHairline, t.Border).Radius(radiusWindowCard).
			Role(gui.RoleRegion).Label(questionCardLabel).Children(func() {
			typeSecondary.style(t, gui.Text(c, questionCopyNoItems)).TextColor(t.TextMuted)
		})
		return
	}
	if !q.questionSlot(q.index, len(items)) {
		// 题号不在这一批里：给原因，不拿别的题顶替。
		gui.Column(c).WidthPercent(100).MaxWidth(questionCardWidth).Shrink(0).
			Padding(space4).Gap(space2).Background(t.Surface).
			Border(strokeHairline, t.Border).Radius(radiusWindowCard).
			Role(gui.RoleRegion).Label(questionCardLabel).Children(func() {
			typeSecondary.style(t, gui.Text(c, questionCopyNoItems)).TextColor(t.TextMuted)
		})
		return
	}
	// IN-44 / G-D1-01 提问卡容器
	gui.Column(c).WidthPercent(100).MaxWidth(questionCardWidth).Shrink(0).
		Padding(space4).Gap(space2).Background(t.Surface).
		Border(strokeHairline, t.Border).Radius(radiusWindowCard).
		Role(gui.RoleRegion).Label(questionCardLabel).Children(func() {
		a.questionHead(c, snap, items)
		if q.collapsed {
			return
		}
		a.questionBody(c, items[q.index])
		a.questionFoot(c, snap, items)
	})
}

// questionHead 是头部：当前题标题、进度徽标、倒计时与收起/展开。
func (a *App) questionHead(c *gui.Context, snap QuestionSnapshot, items []QuestionItemSnapshot) {
	t := c.Theme()
	q := &a.question
	title := questionTitle(items[q.index])
	gui.Row(c).AlignItems(gui.Center).Gap(space2).Children(func() {
		// G-D1-02 当前字段标题
		typeH4.style(t, gui.Text(c, title)).TextColor(t.Text).SingleLine().Label(title)
		if len(items) > 1 {
			label := fmt.Sprintf("第 %d 题，共 %d 题", q.index+1, len(items))
			gui.Row(c).Shrink(0).Padding(0, space1).Radius(radiusSmall).
				Background(t.SurfacePressed).Label(label).Children(func() {
				typeMicro.style(t, gui.Text(c, fmt.Sprintf("%d/%d", q.index+1, len(items)))).
					TextColor(t.TextMuted).SingleLine()
			})
		}
		gui.Spacer(c)
		if !snap.Deadline.IsZero() {
			a.questionTimer(c, snap.Deadline)
		}
		label, icon, mark := questionCopyCollapse, iconChevronDown, "chevron-down"
		if q.collapsed {
			label, icon, mark = questionCopyExpand, iconChevronUp, "chevron-up"
		}
		// G-D1-07 收起 / 展开
		clicked, focused := questionCollapseButton(c, t, label, icon, mark, q.busy && !q.collapsed)
		if focused {
			q.buttonFocused = true
		}
		if clicked {
			q.collapsed = !q.collapsed
		}
	})
	if q.collapsed {
		a.questionCollapsedStatus(c, items[q.index])
	}
}

// questionCollapsedStatus 是收起态的状态行（G-D1-05）：未答写「待选择」或
// 「待选择 · N 个选项」，已答单值写「已选：{answer}」，多值写「已选 N 项」。
// 文案只读当前题，不把整批的进度混进来；超长单行截断，完整值在 tooltip 里。
func (a *App) questionCollapsedStatus(c *gui.Context, item QuestionItemSnapshot) {
	t := c.Theme()
	q := &a.question
	answer := q.answerAt(q.index, item)
	text := questionCopyPending
	enabledOptions := len(item.Options)
	if enabledOptions > 0 {
		enabledOptions = len(q.filteredOptions(item))
	}
	switch {
	case answer == "" && enabledOptions > 0:
		text = fmt.Sprintf("待选择 · %d 个选项", enabledOptions)
	case answer == "":
		text = questionCopyPending
	case item.Multiple:
		text = fmt.Sprintf("已选 %d 项", len(questionLines(answer)))
	default:
		text = "已选：" + answer
	}
	// G-D1-05 收起态状态行
	typeMicro.style(t, gui.Text(c, text)).TextColor(t.TextMuted).SingleLine().
		Label(text).Tooltip(text).Role(gui.RoleStatus)
}

// questionCollapseButton 是头上的「收起 / 展开」文字小按钮：12px 次要色、无边框，
// 箭头随状态换向。它同时报告焦点，窗口级的回车据此判断是否该让给按钮。
func questionCollapseButton(
	c *gui.Context, t *gui.Theme, label string, icon *gui.SVG, mark string, disabled bool,
) (clicked, focused bool) {
	b := gui.ButtonBase(c).Shrink(0).Height(rowCompact).Padding(0, space1).
		Radius(radius).Label(label).Tooltip(label + "：回答这张提问卡").
		Cursor(gui.CursorPointer).Disabled(disabled)
	switch {
	case disabled:
		b = b.Opacity(0.5)
	case b.Pressed():
		b = b.Background(t.SurfacePressed)
	case b.Hovered():
		b = b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		gui.Row(c).Gap(space1).AlignItems(gui.Center).Children(func() {
			iconGlyph(c, icon, t.TextMuted, mark)
			typeMicro.style(t, gui.Text(c, label)).TextColor(t.TextMuted).SingleLine()
		})
	})
	return b.Clicked(), b.Focused()
}

// questionTimer 是倒计时行（G-D1-06）：12px 等宽数字，剩余时间按 m:ss 显示；
// 到点后换成一句状态，不显示负数。
func (a *App) questionTimer(c *gui.Context, deadline time.Time) {
	t := c.Theme()
	gui.Row(c).Shrink(0).Gap(space1).AlignItems(gui.Center).Children(func() {
		iconGlyph(c, iconClock, t.TextMuted, "clock")
		text := questionClock(time.Until(deadline))
		typeMicro.style(t, gui.Text(c, text)).Font(codeFamily()).
			TextColor(t.TextMuted).SingleLine().Label(text).Role(gui.RoleTimer)
	})
}

// questionClock 把剩余时间写成「剩余 m:ss」；不足 1 分钟仍是 m:ss 形态，到点换成
// 「已到作答截止时间」。
func questionClock(left time.Duration) string {
	if left <= 0 {
		return questionCopyTimeout
	}
	seconds := int((left + time.Second - 1) / time.Second)
	return fmt.Sprintf("剩余 %d:%02d", seconds/60, seconds%60)
}

// questionBody 是体部：正文、说明、筛选框、选项或自由输入、自定义答案。体部固定
// 高度并内部滚动，头与脚不随之滚动（G-D1-04）。
func (a *App) questionBody(c *gui.Context, item QuestionItemSnapshot) {
	t := c.Theme()
	q := &a.question
	// 体部只画当前这一题：进来先把这一题的槽位补齐，后面按 index 取值不会越界。
	q.questionSlot(q.index, q.index+1)
	gui.Scroll(c).Shrink(0).Height(questionBodyHeight).Children(func() {
		gui.Column(c).WidthPercent(100).Gap(space2).Children(func() {
			if text := questionText(item); text != "" {
				// G-D1-08 提问正文
				typeBody.style(t, gui.Text(c, text)).TextColor(t.Text).Selectable()
			}
			if desc := strings.TrimSpace(item.Description); desc != "" && desc != questionText(item) {
				// G-D1-09 字段描述
				typeSecondary.style(t, gui.Text(c, desc)).TextColor(t.TextMuted).Selectable()
			}
			if len(item.Options) > questionFilterThreshold {
				// G-D1-10 筛选选项…
				gui.TextInputBase(c, &q.filter).Label(questionCopyFilter).
					Placeholder(questionCopyFilter).Padding(space1, space2).Radius(radius).
					Background(fieldFill(t)).SingleLine()
			}
			if len(item.Options) == 0 {
				// G-D1-14 每行输入一个值（自由输入）
				gui.TextAreaBase(c, &q.custom[q.index]).Grow(1).
					MinHeight(questionInputMinHeight).MaxHeight(questionInputMaxHeight).
					Label(questionCopyFreeform).Placeholder(questionCopyFreeform).
					Padding(space1, space2).Radius(radius).Background(fieldFill(t))
				return
			}
			a.questionOptions(c, item)
			if item.AllowCustom {
				label, hint := questionCopyCustom, questionCopyCustomHint
				if item.Multiple {
					label, hint = questionCopyCustomOption, questionCopyCustomOptHint
				}
				a.questionCustomField(c, item, label, hint)
			}
		})
	})
}

// questionOptions 画选项：单选是一组单选项（↑/↓ 选、Space 选中），多选是一列
// 勾选框（Space 切换）。筛选后没有匹配时给空态句。
func (a *App) questionOptions(c *gui.Context, item QuestionItemSnapshot) {
	t := c.Theme()
	q := &a.question
	q.questionSlot(q.index, q.index+1)
	options := q.filteredOptions(item)
	if len(options) == 0 {
		if strings.TrimSpace(q.filter) != "" {
			// G-D1-13 没有匹配的选项
			typeSecondary.style(t, gui.Text(c, questionCopyNoMatch)).
				TextColor(t.TextMuted).Selectable()
		}
		return
	}
	if item.Multiple {
		for _, option := range options {
			option := option
			on := q.selected[q.index][option]
			// G-D1-12 多选项条目：空格勾选 / 取消，动作随勾选框一起报给读屏。
			gui.Checkbox(c, &on, option).KeyShortcuts(0, gui.KeySpace)
			q.selected[q.index][option] = on
		}
		return
	}
	// 自定义答案非空时接管答案，选项选中态随之清掉（G-D1-118）；清空自定义后
	// 回到选项选择。
	choice := q.singleChoice()
	if strings.TrimSpace(q.custom[q.index]) != "" {
		choice = ""
		q.selected[q.index] = map[string]bool{}
	}
	gui.RadioGroup(c, func() {
		for _, option := range options {
			option := option
			// G-D1-11 单选项条目：空格选中，单选项互斥。
			gui.Radio(c, &choice, option, option).KeyShortcuts(0, gui.KeySpace)
		}
	}).Label(questionTitle(item))
	q.setSingleChoice(choice)
}

// questionCustomField 是「自定义答案 / 自定义选项」：一行灰标签 + 多行输入框。
// 标签与输入框成对：输入框以这行可见标签为可读名（LabelledBy），读屏先念标签
// 再念输入内容，提示语留在 placeholder 里。
func (a *App) questionCustomField(c *gui.Context, item QuestionItemSnapshot, label, hint string) {
	t := c.Theme()
	q := &a.question
	// G-D1-15 自定义答案 / G-D1-16 自定义选项
	labelEl := typeMicro.style(t, gui.Text(c, label)).TextColor(t.TextMuted).SingleLine()
	draft := &q.custom[q.index]
	if item.Multiple {
		draft = &q.extra[q.index]
	}
	input := gui.TextAreaBase(c, draft).Grow(1).
		MinHeight(questionInputMinHeight).MaxHeight(questionInputMaxHeight).
		Label(label).Placeholder(hint).Padding(space1, space2).Radius(radius).
		Background(fieldFill(t))
	if labelEl != (gui.Element{}) {
		input.LabelledBy(labelEl)
	}
}

// questionFoot 是脚部：错误行 + 拒绝 / 上一题 / 下一题 / 提交。
func (a *App) questionFoot(c *gui.Context, snap QuestionSnapshot, items []QuestionItemSnapshot) {
	t := c.Theme()
	q := &a.question
	// 补答之后本地校验错误自动消失（G-D1-138）。
	if q.err == questionCopyRequired && q.complete(items) {
		q.err = ""
	}
	if q.err != "" {
		// G-D1-17 错误行：14px 危险色、脚部按钮行之上、role=alert 播报一次
		gui.Row(c).Children(func() {
			typeSecondary.style(t, gui.Text(c, q.err)).TextColor(t.Danger).
				Selectable().Label(q.err).Role(gui.RoleAlert)
		})
	}
	last := q.index == len(items)-1
	gui.Row(c).AlignItems(gui.Center).Gap(space2).Children(func() {
		// G-D1-18 拒绝
		decline := actionButton(c, questionCopyDecline, "", false, q.busy)
		if decline.Focused() {
			q.buttonFocused = true
		}
		if decline.Clicked() {
			a.declineQuestion(snap)
		}
		gui.Spacer(c)
		if q.index > 0 {
			// G-D1-19 上一题
			previous := actionButton(c, questionCopyPrevious, "上一题", false, q.busy)
			if previous.Focused() {
				q.buttonFocused = true
			}
			if previous.Clicked() {
				a.goQuestion(q.index - 1)
			}
		}
		if !last {
			// G-D1-20 下一题
			next := actionButton(c, questionCopyNext, "下一题", true, q.busy)
			if next.Focused() {
				q.buttonFocused = true
			}
			if next.Clicked() {
				a.goQuestion(q.index + 1)
			}
			return
		}
		// 单题且未答时「提交」直接禁用（G-D1-22）；多题交给提交时的校验，
		// 因为「下一题」不阻断前进。
		empty := len(items) == 1 && q.answerAt(0, items[0]) == ""
		// G-D1-21 提交（实心强调色、高 28、脚部最右）；执行中与未答时禁用
		submit := actionButton(c, questionCopySubmit, questionCopySubmit, true, q.busy || empty)
		if submit.Focused() {
			q.buttonFocused = true
		}
		if submit.Clicked() {
			a.submitQuestion(snap)
		}
	})
}

// goQuestion 切到另一题：已答内容保留，筛选词与错误行清掉。
func (a *App) goQuestion(index int) {
	q := &a.question
	q.index = index
	q.filter = ""
	q.err = ""
}

// filteredOptions 取当前题在筛选词下的选项：不筛选时原样返回。
func (q *questionCardState) filteredOptions(item QuestionItemSnapshot) []string {
	needle := strings.ToLower(strings.TrimSpace(q.filter))
	if needle == "" {
		return item.Options
	}
	out := make([]string, 0, len(item.Options))
	for _, option := range item.Options {
		if strings.Contains(strings.ToLower(option), needle) {
			out = append(out, option)
		}
	}
	return out
}

// singleChoice 取当前题的单选值：多选或未选时为空。
func (q *questionCardState) singleChoice() string {
	if q.index < 0 || q.index >= len(q.selected) {
		return ""
	}
	for option := range q.selected[q.index] {
		return option
	}
	return ""
}

// setSingleChoice 记下单选值：单选互斥，只留一个。
func (q *questionCardState) setSingleChoice(choice string) {
	if q.index < 0 || q.index >= len(q.selected) {
		return
	}
	if q.selected[q.index] == nil {
		q.selected[q.index] = map[string]bool{}
	}
	if choice == "" {
		return
	}
	if q.selected[q.index][choice] {
		return
	}
	q.selected[q.index] = map[string]bool{choice: true}
}

// answerAt 取某题的答案原文：自定义答案优先（接管选项），其次勾选项，多选时
// 勾选在前、自定义选项在后（G-D1-16）。
func (q *questionCardState) answerAt(index int, item QuestionItemSnapshot) string {
	// 槽位缺（例如这一批刚换过、视图状态还没补齐）按"没答"处理，不越界取值。
	if index < 0 || index >= len(q.custom) {
		return ""
	}
	if custom := strings.TrimSpace(q.custom[index]); custom != "" {
		return custom
	}
	if len(item.Options) == 0 {
		return ""
	}
	var picked []string
	for _, option := range item.Options {
		if q.selected[index][option] {
			picked = append(picked, option)
		}
	}
	if item.Multiple {
		picked = append(picked, questionLines(q.extra[index])...)
	}
	if len(picked) == 0 {
		return ""
	}
	if item.Multiple {
		// 勾选在前、自定义选项在后；同一值只留第一次出现（G-D1-16）。
		seen := make(map[string]bool, len(picked))
		merged := picked[:0]
		for _, value := range picked {
			if seen[value] {
				continue
			}
			seen[value] = true
			merged = append(merged, value)
		}
		return strings.Join(merged, "\n")
	}
	return picked[0]
}

// complete 判断这一批的必填是否都答了：没有必填事实，按「每题都有答案」算。
func (q *questionCardState) complete(items []QuestionItemSnapshot) bool {
	for i, item := range items {
		if q.answerAt(i, item) == "" {
			return false
		}
	}
	return true
}

// questionLines 把多行草稿切成非空行。
func questionLines(text string) []string {
	var out []string
	for _, line := range strings.Split(text, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}
	return out
}

// questionBatchAnswer 拼出这一批要提交的 answer 原文：单题原样给，多题按
// 「题目：答案」逐行拼接（wire 只有一段原文，见文件头）。
func questionBatchAnswer(items []QuestionItemSnapshot, answers []string) string {
	if len(answers) == 1 {
		return answers[0]
	}
	lines := make([]string, 0, len(answers))
	for i, answer := range answers {
		lines = append(lines, questionTitle(items[i])+"："+answer)
	}
	return strings.Join(lines, "\n")
}

// submitQuestion 提交这一批：先做本地必填校验，再经写登记发出。失败时卡片保留、
// 错误行给原因、按钮恢复可点（G-D1-17 / G-D1-21）。
func (a *App) submitQuestion(snap QuestionSnapshot) {
	items := questionItems(snap)
	q := &a.question
	if q.busy {
		return
	}
	if len(items) == 0 {
		// 空清单：给可读原因，不发写入（原来这里静默返回，卡片仍是一块空白）。
		q.err = questionCopyNoItems
		return
	}
	if a.questionAnswered(snap.ID) {
		// 已经答过 / 已经结算的提问：再提交只会把旧答案再发一遍。
		q.err = questionCopyStale
		return
	}
	answers := make([]string, len(items))
	missing := false
	for i, item := range items {
		answers[i] = q.answerAt(i, item)
		if answers[i] == "" {
			missing = true
		}
	}
	if missing {
		q.err = questionCopyRequired
		return
	}
	if a.intents == nil {
		return
	}
	q.busy = true
	q.err = ""
	answer := questionBatchAnswer(items, answers)
	requestID := snap.ID
	a.dispatchQuiet(questionCopySubmit, func(ctx context.Context) error {
		return a.intents.RespondQuestion(ctx, answer)
	}, func() {
		a.questionAnsweredID = requestID
		a.question = questionCardState{requestID: requestID}
	}, func(msg string) {
		a.question.busy = false
		a.question.err = msg
	})
}

// declineQuestion 拒绝这一批（等同「不回答」）：卡片消失、时间线记录拒绝。
func (a *App) declineQuestion(snap QuestionSnapshot) {
	q := &a.question
	if q.busy || a.intents == nil {
		return
	}
	q.busy = true
	q.err = ""
	requestID := snap.ID
	a.dispatchQuiet("拒绝提问", func(ctx context.Context) error {
		return a.intents.SkipQuestion(ctx)
	}, func() {
		a.questionAnsweredID = requestID
		a.question = questionCardState{requestID: requestID}
	}, func(msg string) {
		a.question.busy = false
		a.question.err = msg
	})
}

// questionKeys 是提问卡的窗口级键位：Esc 拒绝、↵ 提交（多题时先逐题前进，末题
// 提交）。焦点在文字输入里时不抢键（输入框里的 ↵ 是换行）；焦点已经在卡里的
// 按钮上时交给那枚按钮（否则一次回车会既点按钮又提交）。
func (a *App) questionKeys(c *gui.Context) {
	if a.palette.open || a.screen != ScreenMain {
		return
	}
	snap, ok := a.liveQuestion()
	if !ok || a.questionAnswered(snap.ID) {
		return
	}
	q := &a.question
	if q.busy {
		return
	}
	items := questionItems(snap)
	if len(items) == 0 {
		return
	}
	if c.Shortcut(0, gui.KeyEscape) {
		a.declineQuestion(snap)
		return
	}
	if q.buttonFocused || c.FocusTakesText() {
		return
	}
	if c.Shortcut(0, gui.KeyEnter) {
		if q.index == len(items)-1 {
			a.submitQuestion(snap)
			return
		}
		a.goQuestion(q.index + 1)
	}
}
