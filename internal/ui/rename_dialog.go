package ui

import (
	"context"
	"strings"
	"unicode"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是重命名弹窗（G-F1 / G-F2）：会话与工作区共用一套骨架——
// 标题 + 一句描述 + 字段标签 + 输入框 + 校验提示 + 错误行 + 取消/确认。
//
// 入口与「行内快捷」并存：
//   - 标题栏的会话名双击、标题栏 ⋯ 菜单的「重命名」、左栏会话行菜单的
//     「重命名」打开这个弹窗；
//   - 左栏会话行 / 工作区分组行双击仍是行内快捷编辑（renameRow /
//     groupRenameRow），不弹窗。
//
// 弹窗只发起已有写入路径：会话走 Intents.RenameSession，工作区写本地视图状态
// （groupNames）并落盘本地偏好。校验四条互斥：超长 → 与原名相同 → 控制字符 →
// 去空白后为空；任一成立时「确认」禁用。回车保存、Esc 取消（提交中无效）。

const (
	renameControlMax = 80

	renameCopySessionTitle  = "重命名会话"
	renameCopySessionDesc   = "输入新的会话名称"
	renameCopySessionField  = "会话名称"
	renameCopySessionPlace  = "输入会话名称"
	renameCopyWorkspaceT    = "重命名工作区"
	renameCopyWorkspaceDesc = "输入新的工作区名称"
	renameCopyWorkspaceFld  = "工作区名称"
	renameCopyWorkspacePlc  = "输入工作区名称"
	renameCopyTooLong       = "最多使用 80 个字符"
	renameCopySameSession   = "请输入不同的会话名称"
	renameCopySameWorkspace = "请输入不同的工作区名称"
	renameCopyCtrlSession   = "会话名称中不能包含控制字符"
	renameCopyCtrlWorkspace = "工作区名称中不能包含控制字符"
	renameCopyEmptySession  = "会话名称不能为空"
	renameCopyEmptyWksp     = "工作区名称不能为空"
	renameCopyCancel        = "取消"
	renameCopyConfirm       = "确认"
	renameCopySaving        = "保存中…"
	renameCopyBuiltinWksp   = "内置工作区不能重命名"
)

// renameKind 是重命名弹窗改的对象。
type renameKind int

const (
	renameNone renameKind = iota
	renameSession
	renameWorkspace
)

// renameDialogState 是弹窗的可丢视图状态。
type renameDialogState struct {
	open   bool
	kind   renameKind
	index  int    // 会话：目录里的行序号
	wkspID string // 工作区：分组标识
	// original 是打开时的名称（判断「与当前相同」）；path 是工作区路径行。
	original string
	path     string
	builtin  bool
	draft    string
	// focus / selectAll 让输入框在打开那一帧拿到焦点并全选。
	focus     bool
	selectAll bool
	// submitting 是一次写入正在进行：两个按钮禁用，Esc 不关闭。
	submitting bool
	err        string
}

// openSessionRenameDialog 打开会话重命名弹窗。
func (a *App) openSessionRenameDialog(index int) {
	if index < 0 {
		a.headerNote = noteNoSessionID
		return
	}
	name := a.sessionName(index, a.sessionIDAt(index))
	a.renameDialog = renameDialogState{
		open: true, kind: renameSession, index: index,
		original: name, draft: name, focus: true, selectAll: true,
	}
}

// openWorkspaceRenameDialog 打开工作区重命名弹窗。内置工作区不能改名：弹窗仍
// 打开，但输入框与确认禁用，并写清原因（不静默无反应）。
func (a *App) openWorkspaceRenameDialog(ws sidebarWorkspace) {
	a.renameDialog = renameDialogState{
		open: true, kind: renameWorkspace, wkspID: ws.id,
		original: ws.name, draft: ws.name, path: ws.path, builtin: ws.builtin,
		focus: !ws.builtin, selectAll: !ws.builtin,
	}
}

// closeRenameDialog 收起弹窗并清掉草稿与错误。
func (a *App) closeRenameDialog() { a.renameDialog = renameDialogState{} }

// renameDialogCopy 取当前对象的文案组。
func (a *App) renameDialogCopy() (title, desc, field, placeholder, same, ctrl, empty string) {
	if a.renameDialog.kind == renameWorkspace {
		return renameCopyWorkspaceT, renameCopyWorkspaceDesc, renameCopyWorkspaceFld,
			renameCopyWorkspacePlc, renameCopySameWorkspace, renameCopyCtrlWorkspace,
			renameCopyEmptyWksp
	}
	return renameCopySessionTitle, renameCopySessionDesc, renameCopySessionField,
		renameCopySessionPlace, renameCopySameSession, renameCopyCtrlSession,
		renameCopyEmptySession
}

// renameDialogProblem 按三条校验的固定顺序取第一条：超长 → 与原名相同 →
// 控制字符 → 去空白后为空。返回空串表示可以提交。
func (a *App) renameDialogProblem() string {
	_, _, _, _, same, ctrl, empty := a.renameDialogCopy()
	draft := a.renameDialog.draft
	if len([]rune(draft)) > renameControlMax {
		return renameCopyTooLong
	}
	trimmed := strings.TrimSpace(draft)
	if trimmed == "" {
		return empty
	}
	if trimmed == a.renameDialog.original {
		return same
	}
	for _, r := range draft {
		if unicode.IsControl(r) {
			return ctrl
		}
	}
	return ""
}

// renameDialogBlocked 返回确认按钮被禁用的原因：空串表示可点。
func (a *App) renameDialogBlocked() string {
	st := &a.renameDialog
	switch {
	case st.submitting:
		return "正在保存"
	case st.builtin:
		return renameCopyBuiltinWksp
	case a.renameDialogProblem() != "":
		return a.renameDialogProblem()
	case st.kind == renameSession && a.wired() && a.sessionIDAt(st.index) == "":
		return noteNoSessionID
	case st.kind == renameSession && a.wired() && a.intents == nil:
		return noteNoWriteChannel
	}
	return ""
}

// confirmRenameDialog 提交一次重命名。
func (a *App) confirmRenameDialog() {
	st := &a.renameDialog
	if a.renameDialogBlocked() != "" {
		return
	}
	title := strings.TrimSpace(st.draft)
	if st.kind == renameWorkspace {
		if st.builtin || st.wkspID == "" {
			return
		}
		if a.sidebarView.groupNames == nil {
			a.sidebarView.groupNames = map[string]string{}
		}
		a.sidebarView.groupNames[st.wkspID] = title
		a.persistViewState()
		a.closeRenameDialog()
		return
	}
	if a.intents == nil {
		// 样例：只改视图里的标题映射。
		a.renamed[st.index] = title
		a.closeRenameDialog()
		return
	}
	sessionID := a.sessionIDAt(st.index)
	if sessionID == "" {
		return
	}
	index := st.index
	st.submitting = true
	st.err = ""
	a.dispatchQuiet("重命名", func(ctx context.Context) error {
		return a.intents.RenameSession(ctx, sessionID, title)
	}, func() {
		a.renamed[index] = title
		a.closeRenameDialog()
	}, func(msg string) {
		a.renameDialog.submitting = false
		a.renameDialog.err = msg
	})
}

// renameDialog 画重命名弹窗。
func (a *App) renameDialogView(c *gui.Context) {
	st := &a.renameDialog
	if !st.open {
		return
	}
	t := c.Theme()
	title, desc, field, placeholder, _, _, _ := a.renameDialogCopy()
	open := true
	problem := a.renameDialogProblem()
	blocked := a.renameDialogBlocked()
	m := modalShell(gui.Modal(c, &open, func() {
		panel := gui.Column(c).Gap(space3).Role(gui.RoleDialog).Label(title)
		panel.Children(func() {
			typeDialogTitle.style(t, gui.Text(c, title)).TextColor(t.Text).
				SingleLine().Role(gui.RoleHeading).Level(2)
			typeBody.style(t, gui.Text(c, desc)).TextColor(t.TextMuted)
			gui.Column(c).Gap(space2).Children(func() {
				// G-F1-01 / G-F2-01 弹窗标题（重命名会话 / 重命名工作区）
				typeBodyStrong.style(t, gui.Text(c, field)).TextColor(t.Text).Label(field)
				input := gui.TextInputBase(c, &st.draft).
					// G-F1-02 / G-F2-02 弹窗说明
					Label(field).Placeholder(placeholder).
					Height(plgCtlH).Padding(screenInputPadV, screenSpace25).
					Background(fieldFill(t)).BorderWidth(strokeHairline).BorderColor(t.Border).
					// G-F1-03 / G-F2-03 字段标签
					Disabled(st.builtin || st.submitting)
				// G-F1-04 / G-F2-04 输入框占位
				if st.focus {
					input = input.Focus()
					st.focus = false
				}
				if st.selectAll {
					input.SetTextSelection(0, len([]rune(st.draft)))
					st.selectAll = false
				}
				input.OnSubmit(func() { a.confirmRenameDialog() })
				if st.kind == renameWorkspace && strings.TrimSpace(st.path) != "" {
					// G-F2-05 工作区路径
					typeBody.style(t, gui.Text(c, st.path)).TextColor(t.TextMuted).
						Selectable().Label("工作区路径：" + st.path)
				}
			})
			switch {
			case st.builtin:
				typeSecondary.style(t, gui.Text(c, renameCopyBuiltinWksp)).
					TextColor(t.TextMuted).Label(renameCopyBuiltinWksp)
			case problem != "":
				// G-F1-05 / G-F2-05 校验提示（超长 / 同名 / 控制字符 / 为空）
				typeSecondary.style(t, gui.Text(c, problem)).
					TextColor(t.TextMuted).Label("校验提示：" + problem)
			case a.wired() && st.kind == renameSession && a.intents == nil:
				// G-F1-06 SessionTitle 不可用 / 无写通道
				typeSecondary.style(t, gui.Text(c, noteNoWriteChannel)).
					TextColor(t.TextMuted).Label(noteNoWriteChannel)
			}
			// G-F1-07 重命名失败原因
			if st.err != "" {
				typeSecondary.style(t, gui.Text(c, st.err)).TextColor(t.Danger).
					Role(gui.RoleAlert).Label(st.err)
			}
			// G-F1-08 / G-F2-06 取消
			gui.Row(c).Justify(gui.End).Gap(space2).Children(func() {
				cancel := gui.Button(c, renameCopyCancel)
				if st.submitting {
					cancel = cancel.Disabled(true).Opacity(0.5)
				}
				if cancel.Clicked() && !st.submitting {
					a.closeRenameDialog()
				}
				confirmText := renameCopyConfirm
				if st.submitting {
					confirmText = renameCopySaving
				}
				confirm := gui.PrimaryButton(c, confirmText).
					Label(confirmText)
				// G-F1-09 / G-F2-07 确认
				if blocked != "" {
					confirm = confirm.Disabled(true).Opacity(0.5).Tooltip(blocked)
				}
				if confirm.Clicked() {
					a.confirmRenameDialog()
				}
			})
		})
		panel.Width(renameDialogWidth(c))
	}))

	switch {
	case !open || m.Dismissed():
		if st.submitting {
			// 提交中 Esc / 点遮罩无效：下一帧仍打开。
			st.open = true
			return
		}
		a.closeRenameDialog()
	}
}

// renameDialogWidth 是重命名弹窗的内容宽：参考 sm:max-w-md = 448，窗口更窄时
// 收进窗口内（左右各留 28）。
func renameDialogWidth(c *gui.Context) float32 {
	return dialogWidth(c, 399)
}
