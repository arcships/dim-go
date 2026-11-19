package ui

import (
	"context"
	"fmt"
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// 确认框（全局 E1–E4）：四类高后果动作共用一个对话框骨架。
//
//   - E1 编辑运行中消息：会话还在跑时提交已发消息的编辑；
//   - E2 删除会话：一条或多条；
//   - E3 归档会话：一次归档多条（单条归档可恢复，不弹框）；
//   - E4 删除工作区会话：一个工作区分组下的全部会话。
//
// 结构固定为标题 + 说明（删除类含不可撤销警告）+ 可选路径行 + 错误行 + 取消/确认；
// 删除类用危险色，归档用主色（可恢复）。危险动作的默认焦点落在「取消」，回车交给
// 焦点上的那枚按钮，Esc 关闭（执行中 Esc 无效）。
//
// 执行结果：成功关闭弹窗并写一行回执；失败留在弹窗里给原因，可再点一次重试。

const (
	confirmKindNone            = ""
	confirmKindEditRunning     = "edit-running"
	confirmKindDelete          = "delete"
	confirmKindArchive         = "archive"
	confirmKindDeleteWorkspace = "delete-workspace"
)

const (
	confirmCopyEditTitle   = "会话正在运行"
	confirmCopyEditDesc    = "当前会话仍在运行中。是否停止本次运行并发送编辑后的消息？"
	confirmCopyStopSend    = "停止并发送"
	confirmCopyCancel      = "取消"
	confirmCopyDeleteOne   = "删除会话"
	confirmCopyDeleteMany  = "删除 %d 个会话"
	confirmCopyDeleteDesc  = "确定要永久删除“%s”吗？删除后会话将立即从侧边栏移除，且无法恢复。"
	confirmCopyDeleteList  = "确定要永久删除以下会话吗？%s"
	confirmCopyArchive     = "归档 %d 个会话"
	confirmCopyArchiveSub  = "确定要归档以下会话吗？%s"
	confirmCopyArchiveRun  = "正在运行的会话会被中断，其队列中的后续消息也会被丢弃。"
	confirmCopyWorkspace   = "删除工作区会话"
	confirmCopyWorkspaceD  = "将删除“%s”下的 %d 条会话。"
	confirmCopyWorkspaceF  = "该工作区"
	confirmCopyDelete      = "删除"
	confirmCopyDeleteBusy  = "删除中..."
	confirmCopyArchiveBtn  = "归档"
	confirmCopyArchiveBusy = "归档中..."
	confirmCopyStopBusy    = "停止中..."
	confirmCopyUnnamed     = "未命名会话"
	confirmCopyMore        = " 等 %d 个"

	// confirmNamesMax 是描述里逐个列出的会话名个数，超出以「等 N 个」收尾。
	confirmNamesMax = 3
	// confirmWidth 是对话框的内容宽（参考产品对话框的常用宽）。
	confirmWidth = 420
	// confirmQuoteOpen / confirmQuoteClose 是描述里引用会话名的中文直角引号。
	confirmQuoteOpen  = "“"
	confirmQuoteClose = "”"
)

// confirmTarget 是一个目标会话：视图里的行序号（样例态用）、事实标识与标题。
type confirmTarget struct {
	index int
	id    string
	title string
}

// confirmState 是确认框的视图状态。打开时把目标与「打开时是否在运行」定格：
// 弹窗期间集合或运行态变化不改描述（R-GC-038 / R-GC-097），确认时按当前目标执行。
type confirmState struct {
	kind string
	// targets 是删除/归档的目标会话（E2、E3）。
	targets []confirmTarget
	// workspaceID / workspaceName / workspacePath 是 E4 的目标工作区。
	workspaceID   string
	workspaceName string
	workspacePath string
	// running 表示打开时目标里有正在运行的会话（E3 的警告按它定格）。
	running bool
	// editText / editRow 是 E1 待发送的编辑正文与行号。
	editText string
	editRow  int
	// busy 是一次执行正在写：按钮整组置灰，Esc 不关闭。
	busy bool
	// err 是执行失败的原始原因，在按钮行之上。
	err string
}

// openConfirmEditRunning 打开「会话正在运行」确认框：会话还在跑时保存编辑。
func (a *App) openConfirmEditRunning(row int, text string) {
	a.confirm = confirmState{kind: confirmKindEditRunning, editRow: row, editText: text}
}

// openDeleteConfirm 打开删除会话确认框；目标为空时不开。
func (a *App) openDeleteConfirm(indexes []int) {
	targets := a.confirmTargets(indexes)
	if len(targets) == 0 {
		return
	}
	a.confirm = confirmState{kind: confirmKindDelete, targets: targets}
}

// openArchiveConfirm 打开归档确认框：目标里有正在运行的会话时带警告。
func (a *App) openArchiveConfirm(indexes []int) {
	targets := a.confirmTargets(indexes)
	if len(targets) == 0 {
		return
	}
	a.confirm = confirmState{
		kind: confirmKindArchive, targets: targets, running: a.anyRunning(indexes),
	}
}

// openWorkspaceDeleteConfirm 打开「删除工作区会话」确认框。
func (a *App) openWorkspaceDeleteConfirm(ws sidebarWorkspace, threads []sidebarThread) {
	indexes := make([]int, 0, len(threads))
	for _, thread := range threads {
		indexes = append(indexes, thread.index)
	}
	a.confirm = confirmState{
		kind:          confirmKindDeleteWorkspace,
		targets:       a.confirmTargets(indexes),
		workspaceID:   ws.id,
		workspaceName: ws.name,
		workspacePath: ws.path,
	}
}

// confirmTargets 把行序号换成确认框要的目标：名称与事实标识都取目录里的读数。
func (a *App) confirmTargets(indexes []int) []confirmTarget {
	out := make([]confirmTarget, 0, len(indexes))
	names := a.sessionNames()
	for _, index := range indexes {
		if index < 0 || index >= len(names) {
			continue
		}
		out = append(out, confirmTarget{
			index: index,
			id:    names[index],
			title: a.sessionName(index, names[index]),
		})
	}
	return out
}

// anyRunning 判断这些行里是否有正在跑的会话：样例看行首标记，连着事实时看
// 当前会话的 Running（其余会话的运行态没有读数）。
func (a *App) anyRunning(indexes []int) bool {
	markers := a.sessionMarkers()
	session, hasSession := a.liveSession()
	for _, index := range indexes {
		if index >= 0 && index < len(markers) && markers[index] == "●" {
			return true
		}
		if hasSession && index >= 0 && index < len(a.sessionNames()) &&
			a.sessionNames()[index] == session.ID && session.Running {
			return true
		}
	}
	return false
}

// confirmTitle 是弹窗标题（E1–E4 各自的文案）。
func (a *App) confirmTitle() string {
	st := a.confirm
	switch st.kind {
	case confirmKindEditRunning:
		return confirmCopyEditTitle
	case confirmKindDelete:
		if len(st.targets) == 1 {
			return confirmCopyDeleteOne
		}
		return fmt.Sprintf(confirmCopyDeleteMany, len(st.targets))
	case confirmKindArchive:
		return fmt.Sprintf(confirmCopyArchive, len(st.targets))
	default:
		return confirmCopyWorkspace
	}
}

// confirmDescription 是弹窗说明（含不可撤销的那半句）。
func (a *App) confirmDescription() string {
	st := a.confirm
	switch st.kind {
	case confirmKindEditRunning:
		return confirmCopyEditDesc
	case confirmKindDelete:
		if len(st.targets) == 1 {
			return fmt.Sprintf(confirmCopyDeleteDesc, st.targets[0].title)
		}
		return fmt.Sprintf(confirmCopyDeleteList, confirmNames(st.targets))
	case confirmKindArchive:
		return fmt.Sprintf(confirmCopyArchiveSub, confirmNames(st.targets))
	default:
		name := strings.TrimSpace(st.workspaceName)
		if name == "" {
			name = confirmCopyWorkspaceF
		}
		return fmt.Sprintf(confirmCopyWorkspaceD, name, len(st.targets))
	}
}

// confirmNames 把目标拼成「“a”、“b”、“c” 等 2 个」：最多三名，超出给剩余数。
func confirmNames(targets []confirmTarget) string {
	quoted := make([]string, 0, confirmNamesMax)
	for i, target := range targets {
		if i == confirmNamesMax {
			break
		}
		title := strings.TrimSpace(target.title)
		if title == "" {
			title = confirmCopyUnnamed
		}
		quoted = append(quoted, confirmQuoteOpen+title+confirmQuoteClose)
	}
	out := strings.Join(quoted, "、")
	if rest := len(targets) - len(quoted); rest > 0 {
		out += fmt.Sprintf(confirmCopyMore, rest)
	}
	return out
}

// confirmPrimaryText 是确认按钮的文案：执行中换成「…中」，带数量时带括号。
func (a *App) confirmPrimaryText() string {
	st := a.confirm
	switch st.kind {
	case confirmKindEditRunning:
		if st.busy {
			return confirmCopyStopBusy
		}
		return confirmCopyStopSend
	case confirmKindDelete:
		if st.busy {
			return confirmCopyDeleteBusy
		}
		if len(st.targets) > 1 {
			return fmt.Sprintf("%s (%d)", confirmCopyDelete, len(st.targets))
		}
		return confirmCopyDelete
	case confirmKindArchive:
		if st.busy {
			return confirmCopyArchiveBusy
		}
		return fmt.Sprintf("%s (%d)", confirmCopyArchiveBtn, len(st.targets))
	default:
		if st.busy {
			return confirmCopyDeleteBusy
		}
		return confirmCopyDelete
	}
}

// confirmDangerous 判断这次确认是不是危险动作：危险动作默认焦点落在取消，回车
// 绝不替用户按下删除；归档可恢复，默认焦点在「归档」上。
func confirmDangerous(kind string) bool {
	return kind != confirmKindArchive
}

// confirmDialog 画四类确认框共用的对话框。
func (a *App) confirmDialog(c *gui.Context) {
	if a.confirm.kind == confirmKindNone {
		return
	}
	t := c.Theme()
	open := true
	kind := a.confirm.kind
	title := a.confirmTitle()
	dialog := gui.Modal(c, &open, func() {
		panel := gui.Column(c).Gap(space3).Role(gui.RoleAlertDialog).Label(title)
		panel.Children(func() {
			typeH3.style(t, gui.Text(c, title)).TextColor(t.Text).SingleLine()
			typeSecondary.style(t, gui.Text(c, a.confirmDescription())).
				TextColor(t.TextMuted).Selectable()
			if kind == confirmKindArchive && a.confirm.running {
				typeSecondary.style(t, gui.Text(c, confirmCopyArchiveRun)).
					TextColor(t.Warning).Selectable()
			}
			if kind == confirmKindDeleteWorkspace {
				a.confirmPathRow(c)
			}
			if a.confirm.err != "" {
				typeSecondary.style(t, gui.Text(c, a.confirm.err)).TextColor(t.Danger).
					Selectable().Label(a.confirm.err).Role(gui.RoleAlert)
			}
			gui.Row(c).Justify(gui.End).Gap(space2).Children(func() {
				cancel := gui.Button(c, confirmCopyCancel)
				if confirmDangerous(kind) {
					// 危险动作的默认焦点在「取消」：回车不会替用户按下删除。
					cancel = cancel.AutoFocus()
				}
				if cancel.Clicked() && !a.confirm.busy {
					a.confirm = confirmState{}
				}
				if confirmDangerous(kind) {
					if confirmDangerButton(c, t, a.confirmPrimaryText(), a.confirm.busy) {
						a.runConfirm()
					}
					return
				}
				accept := gui.PrimaryButton(c, a.confirmPrimaryText())
				if a.confirm.busy {
					accept = accept.Disabled(true).Opacity(0.5)
				} else {
					accept = accept.AutoFocus()
				}
				if accept.Clicked() {
					a.runConfirm()
				}
			})
		})
		panel.Width(confirmWidthFor(c))
	})
	// 执行中按 Esc / 点遮罩不关闭；关闭后状态整份清掉。
	if dialog.Dismissed() && !a.confirm.busy {
		a.confirm = confirmState{}
	}
}

// confirmPathRow 是 E4 的工作区路径行：没有路径时不渲染（不留空行）。
func (a *App) confirmPathRow(c *gui.Context) {
	t := c.Theme()
	path := strings.TrimSpace(a.confirm.workspacePath)
	if path == "" {
		return
	}
	typeSecondary.style(t, gui.Text(c, path)).TextColor(t.TextMuted).Selectable().
		Label("工作区路径：" + path)
}

// confirmDangerButton 是危险色实心按钮：危险底 + 反色字、高 28；执行中 0.5 且不
// 接收指针与键盘。用框架的 ButtonBase 承担命中、焦点与四态。
func confirmDangerButton(c *gui.Context, t *gui.Theme, text string, disabled bool) bool {
	b := gui.ButtonBase(c).Height(rowRegular).Shrink(0).
		Padding(controlPadV, controlPadH).Radius(radius).
		Background(t.Danger).Label(text).Disabled(disabled)
	switch {
	case disabled:
		b = b.Opacity(0.5)
	case b.Pressed():
		b = b.Background(t.Danger.Alpha(0.85))
	case b.Hovered():
		b = b.Background(t.Danger.Alpha(0.92))
	}
	b.Children(func() {
		typeControlStrong.style(t, gui.Text(c, text)).TextColor(t.InverseText).SingleLine()
	})
	return b.Clicked()
}

// confirmWidthFor 是对话框内容宽：窗口更窄时收进来（R-GC-039）。
func confirmWidthFor(c *gui.Context) float32 {
	width, _ := c.Size()
	limit := width - 2*space8
	if limit < confirmWidth {
		if limit < MainMinWidth/2 {
			return MainMinWidth / 2
		}
		return limit
	}
	return confirmWidth
}

// runConfirm 执行当前确认框的动作。
func (a *App) runConfirm() {
	if a.confirm.busy || a.confirm.kind == confirmKindNone {
		return
	}
	switch a.confirm.kind {
	case confirmKindEditRunning:
		a.runConfirmEditRunning()
	case confirmKindDelete:
		a.runConfirmDelete()
	case confirmKindArchive:
		a.runConfirmArchive()
	case confirmKindDeleteWorkspace:
		a.runConfirmDeleteWorkspace()
	}
}

// runConfirmEditRunning 停止当前回合，再保存编辑后的正文。
func (a *App) runConfirmEditRunning() {
	text := strings.TrimSpace(a.confirm.editText)
	row := a.confirm.editRow
	if a.intents == nil {
		a.saveInlineEditText(row, text)
		a.confirm = confirmState{}
		return
	}
	a.confirm.busy = true
	a.confirm.err = ""
	a.dispatchQuiet("停止并发送", func(ctx context.Context) error {
		return a.intents.Stop(ctx)
	}, func() {
		a.confirm = confirmState{}
		a.saveInlineEditText(row, text)
	}, func(msg string) {
		a.confirm.busy = false
		a.confirm.err = msg
	})
}

// runConfirmDelete 删除目标会话：样例态只改视图，连着事实时逐条删并汇总失败。
func (a *App) runConfirmDelete() {
	targets := append([]confirmTarget(nil), a.confirm.targets...)
	if a.intents == nil {
		for _, target := range targets {
			a.archived[target.index] = true
		}
		a.confirm = confirmState{}
		a.headerNote = confirmReceipt("已删除", len(targets), "个会话")
		return
	}
	a.confirm.busy = true
	a.confirm.err = ""
	a.dispatchQuiet("删除会话", func(ctx context.Context) error {
		return a.deleteTargets(ctx, targets)
	}, func() {
		a.confirm = confirmState{}
		a.headerNote = confirmReceipt("已删除", len(targets), "个会话")
	}, func(msg string) {
		a.confirm.busy = false
		a.confirm.err = msg
	})
}

// runConfirmArchive 归档目标会话（可恢复，主色按钮）。
func (a *App) runConfirmArchive() {
	targets := append([]confirmTarget(nil), a.confirm.targets...)
	if a.intents == nil {
		for _, target := range targets {
			a.archived[target.index] = true
		}
		a.confirm = confirmState{}
		a.headerNote = confirmReceipt("已归档", len(targets), "个会话")
		return
	}
	a.confirm.busy = true
	a.confirm.err = ""
	a.dispatchQuiet("归档会话", func(ctx context.Context) error {
		var failed int
		var firstErr error
		for _, target := range targets {
			if target.id == "" {
				continue
			}
			if err := a.intents.ArchiveSession(ctx, target.id, true); err != nil {
				failed++
				if firstErr == nil {
					firstErr = err
				}
			}
		}
		if failed > 0 {
			return fmt.Errorf("%d 个会话归档失败：%v", failed, firstErr)
		}
		return nil
	}, func() {
		a.confirm = confirmState{}
		a.headerNote = confirmReceipt("已归档", len(targets), "个会话")
	}, func(msg string) {
		a.confirm.busy = false
		a.confirm.err = msg
	})
}

// runConfirmDeleteWorkspace 删除一个工作区下的全部会话；非内置工作区连同本端
// 登记的目录一起去掉（会话没了，空组不再留着）。
func (a *App) runConfirmDeleteWorkspace() {
	targets := append([]confirmTarget(nil), a.confirm.targets...)
	workspace := sidebarWorkspace{
		id: a.confirm.workspaceID, name: a.confirm.workspaceName,
		path: a.confirm.workspacePath, builtin: a.confirm.workspaceID == sidebarBuiltinID,
	}
	done := func() {
		a.removeWorkspaceGroup(workspace)
		a.confirm = confirmState{}
		a.headerNote = confirmReceipt("已删除", len(targets), "条会话")
	}
	if a.intents == nil {
		for _, target := range targets {
			a.archived[target.index] = true
		}
		done()
		return
	}
	a.confirm.busy = true
	a.confirm.err = ""
	a.dispatchQuiet("删除工作区会话", func(ctx context.Context) error {
		return a.deleteTargets(ctx, targets)
	}, done, func(msg string) {
		a.confirm.busy = false
		a.confirm.err = msg
	})
}

// deleteTargets 逐条删除目标会话，把失败条数与第一条原因汇总成一行。
func (a *App) deleteTargets(ctx context.Context, targets []confirmTarget) error {
	var failed int
	var firstErr error
	for _, target := range targets {
		if target.id == "" {
			continue
		}
		if err := a.intents.DeleteSession(ctx, target.id); err != nil {
			failed++
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	if failed > 0 {
		return fmt.Errorf("%d 个会话删除失败：%v", failed, firstErr)
	}
	return nil
}

// confirmReceipt 拼执行回执：标题栏下方的一行（「已删除 2 个会话」）。
func confirmReceipt(verb string, count int, unit string) string {
	return fmt.Sprintf("%s %d %s", verb, count, unit)
}
