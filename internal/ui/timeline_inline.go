package ui

import (
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// 消息行内编辑器（TL-31–TL-34）：双击消息或点「编辑消息」把气泡换成编辑器。
//
// 编辑器里是这一行的原文，右下角是「取消」（X）与「发送消息...」（↑）；Esc 放弃
// 编辑、⌘↵ 保存。保存写回历史需要服务端的消息写入方法，本版没有：连着真实
// 事实时保存会留在失败分支（内容不丢、原因写出来），样例模式里就地替换样例正文
// 并标「已编辑」，走查与测试能走完整个交互。

const (
	// inlineEditMinHeight 是编辑器的正文最小高（参考 ≈240×56）。
	inlineEditMinHeight = 56
	// noteInlineEdit 是连线模式下保存失败的原因。
	noteInlineEdit = "服务端没有提供消息编辑（本版不做）"
)

// inlineEditing 报告当前是否有行内编辑器在场。
func (a *App) inlineEditing() bool { return a.inlineEditRow >= 0 }

// beginInlineEdit 进入某一行的行内编辑：原文进编辑器，焦点落到正文。
func (a *App) beginInlineEdit(i int, text string) {
	a.inlineEditRow = i
	a.inlineDraft = text
	a.inlineNotice = ""
}

// cancelInlineEdit 退出编辑并丢掉草稿（TL-332：不弹二次确认）。
func (a *App) cancelInlineEdit() {
	a.inlineEditRow = -1
	a.inlineDraft = ""
	a.inlineNotice = ""
}

// saveInlineEdit 保存当前编辑。
func (a *App) saveInlineEdit() {
	if a.inlineEditRow < 0 {
		return
	}
	text := strings.TrimSpace(a.inlineDraft)
	if text == "" {
		return
	}
	if a.wired() || a.intents != nil {
		a.inlineNotice = noteInlineEdit
		return
	}
	if a.inlineSaved == nil {
		a.inlineSaved = map[int]string{}
	}
	a.inlineSaved[a.inlineEditRow] = text
	a.inlineEditRow = -1
	a.inlineDraft = ""
	a.inlineNotice = ""
}

// inlineEditor 是编辑器本体：右对齐、与气泡同宽档，正文可多行，右下两个按钮。
func (a *App) inlineEditor(c *gui.Context, i int, r round) {
	t := c.Theme()
	_ = r
	gui.Row(c).Grow(1).Children(func() {
		gui.Spacer(c)
		gui.Column(c).MaxWidthPercent(userBubbleMaxPercent).Gap(space1).Children(func() {
			box := gui.Column(c).Shrink(0).Padding(space2, timelinePadX).
				Radius(radiusCard).Background(t.Surface).
				Border(strokeHairline, t.Border).Label("行内编辑器")
			box.Children(func() {
				gui.TextAreaBase(c, &a.inlineDraft).Grow(1).
					MinHeight(inlineEditMinHeight).AutoFocus().Label("编辑消息正文")
				gui.Row(c).Grow(1).Justify(gui.End).Gap(space1).AlignItems(gui.Center).
					Children(func() {
						if iconButton(c, iconButtonSpec{
							icon: iconXMark, label: "取消", tip: "取消编辑", size: rowCompact,
						}) {
							a.cancelInlineEdit()
						}
						disabled := strings.TrimSpace(a.inlineDraft) == ""
						if iconAction(c, iconButtonSpec{
							icon: iconArrowUp, label: "发送消息...", tip: "保存编辑 · ⌘↵",
							size: rowCompact, disabled: disabled,
						}, true) {
							a.saveInlineEdit()
						}
					})
			})
			if a.inlineNotice != "" {
				gui.Row(c).Grow(1).Children(func() {
					gui.Spacer(c)
					typeMicro.style(t, gui.Text(c, a.inlineNotice)).TextColor(t.Danger).SingleLine()
				})
			}
		})
	})
}
