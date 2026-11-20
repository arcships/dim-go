package ui

import (
	"context"
	"fmt"
	"strings"
	"time"

	gui "github.com/egoist/mygo/ui"
)

// 本文件是设置 › 归档会话的重建实现：元素按
// docs/domains/requirements/inventory-settings-rt.md §18（S-archived-01…17，
// 17 个元素）逐行对应，文案逐字取参考产品中文，数值取
// docs/domains/requirements/design-baseline.md：
//
//	页面标题 17.5/600、行标题 13、元信息 11、空态主文 15、卡片圆角 8、
//	行高 56、搜索框与筛选高 32、描边按钮高 28、空态虚线框 220 高。
//
// # 接了什么
//
//   - 列表读事实：连着 Source 且它实现 ArchivedFacts 时，行来自
//     ArchivedSessions()——本端归档登记（归档时间）+ 服务端目录条目（标题、
//     首条预览、工作目录）。样例与单屏截图没有这份读数，空白态。
//   - 恢复（S-archived-13）：Intents.ArchiveSession(id, false)，即取消本端归档；
//     成功后这一行从事实里消失，不留在屏上。
//   - 删除（S-archived-14）与两种批量删除（S-archived-04、S-archived-11）：
//     Intents.DeleteSession，一条一条发；失败把原因写在页首错误行，成功的行由
//     事实驱动消失。
//   - App Server 没有归档事实：归档=本端登记（catalog 的隐藏列表）。所以「全部
//     删除」同时意味着把这些会话从服务端删掉，对话框的「不可撤销」是字面意思。
//
// # 未实现分支
//
//	S-archived-08  项目筛选下拉展开后的选项：选项由 threads 的项目集合生成，
//	               当前没有任何归档会话，下拉本身不渲染（R-ARC-050）。
//	其余 13 个条件元素（对话框、骨架、搜索、分组、行、恢复、删除）都已实现，
//	只是当前状态不成立。

// archivedThread 是一条归档会话。
type archivedThread struct {
	id       string
	title    string
	project  string
	archived string
}

// archivedState 是本屏可丢的界面状态：不是事实。
type archivedState struct {
	loading   bool
	loadError string
	threads   []archivedThread

	query         string
	projectFilter string
	rootSelection string

	deleteAllOpen     bool
	deleteProjectName string
	deleteThreadID    string
	// opErr 是本屏动作（恢复 / 删除）失败的原因；空表示没有失败。它不是事实。
	opErr string
}

// archivedScreen 的首屏取值与参考产品运行时一致：没有归档会话。
var archivedScreen = archivedState{projectFilter: arcCopyAllProjects}

// 本屏几何：参考的 Tailwind 量换算成 design-baseline §1.1 的档位。
const (
	arcToolGap     = 10.5 // 搜索框与筛选之间
	arcGroupGap    = 8    // 分组头与行卡之间、分组之间
	arcRowGap      = 8    // 行内元素之间
	arcCardR       = 8.75 // 卡片圆角
	arcRowH        = 56   // 会话行高
	arcSkeletonH   = 56   // 骨架行高
	arcSkeletonGap = 10.5
	arcRowPadV     = 10
	arcRowPadH     = 14
	arcCtlH        = 32
	arcBtnH        = 28
	arcIconHit     = 24
	arcIconSize    = 14
	arcTrashIcon   = 14
	arcFilterW     = 200
	arcEmptyH      = 220
	arcEmptyIcon   = 24
	arcEmptyBox    = 48
	arcEmptyGap    = 8
	arcBadgePadV   = 3.5
	arcBadgePadH   = 7
	arcBadgeR      = 4.75
	arcChipGap     = 7
	arcTextGap     = 2
	arcDialogW     = 448
	arcDialogMinW  = 160
	arcAlpha50     = 0.5
)

// 字阶（design-baseline §2）。
var (
	arcTitleStyle  = typePageTitle
	arcRowTitle    = fontStep{size: 13, line: 19.5, weight: 500}
	arcGroupTitle  = fontStep{size: 13, line: 19.5, weight: 600}
	arcMetaStyle   = fontStep{size: 11, line: 14, weight: 400}
	arcEmptyTitle  = fontStep{size: 15, line: 21, weight: 500}
	arcBtnStyle    = fontStep{size: 12.25, line: 17.5, weight: 400}
	arcNoteStyle   = fontStep{size: 12.25, line: 17.5, weight: 400}
	arcSearchStyle = fontStep{size: 12.25, line: 17.5, weight: 400}
)

// 本屏图标：Heroicons v2 outline（24 网格、1.5 描边、MIT）的内联路径。
var (
	arcIconRotate = mustSVG(`<path d="M9 15 3 9m0 0 6-6M3 9h12a6 6 0 0 1 0 12h-3"/>`)
)

// 本屏文案：逐字取自参考产品中文。
const (
	arcCopyTitle          = "归档会话"
	arcCopyDeleteAll      = "全部删除"
	arcCopyDeleteAllTitle = "删除所有归档会话"
	arcCopyDeleteAllDesc  = "确定要永久删除 %d 条归档会话吗？此操作不可撤销。"
	arcCopyDeleteProject  = "删除项目内全部会话"
	arcCopyDeleteProjDesc = "确定要永久删除「%s」中的 %d 条归档会话吗？此操作不可撤销。"
	arcCopyDeleteThread   = "删除归档"
	arcCopyDeleteOneDesc  = "确定要永久删除「%s」吗？此操作不可撤销。"
	arcCopyCancel         = "取消"
	arcCopyDelete         = "删除"
	arcCopyRestore        = "恢复"
	arcCopySearch         = "搜索归档会话"
	arcCopySearchHolder   = "搜索归档会话…"
	arcCopyAllProjects    = "全部项目"
	arcCopyNoProject      = "无项目"
	arcCopyClosed         = "已关闭"
	arcCopyArchivedAt     = "%s 归档"
	arcCopySessions       = "%d 条会话"
	arcCopyEmpty          = "暂无归档会话"
	arcCopyNoMatch        = "没有匹配的会话"
	arcCopyNoMatchDesc    = "请尝试其他搜索词或项目筛选条件"
	arcCopyMore           = "更多操作"
	arcCopyLoading        = "正在加载归档会话"
	arcCopyGroupLabel     = "会话"
)

// archivedView 是本帧要画的归档行：连着 Source 时来自事实，样例模式取屏内表。
// 它每帧重算一次，不跨帧缓存（事实变了下一帧就跟上）。
var archivedView struct {
	threads []archivedThread
	wired   bool
}

// renderArchived 是设置 › 归档会话这一屏（设置壳的一个分节）。
func renderArchived(c *gui.Context, a *App) {
	t := c.Theme()
	archivedView.wired = a != nil && a.wired()
	archivedView.threads = arcThreads(a)
	gui.Column(c).Grow(1).Children(func() {
		gui.Scroll(c).Grow(1).Children(func() {
			gui.Column(c).Gap(arcGroupGap).Children(func() {
				arcHeader(c, t)
				if archivedScreen.opErr != "" {
					arcNoteStyle.style(t, gui.Text(c, archivedScreen.opErr)).
						TextColor(t.Danger).Role(gui.RoleAlert).Label(archivedScreen.opErr)
				}
				arcBody(c, a, t)
			})
		})
	})
	arcDeleteAllDialog(c, t, a)
	arcDeleteProjectDialog(c, t, a)
	arcDeleteThreadDialog(c, t, a)
}

// arcThreads 取这一帧的归档行：连着 Source 且它提供归档读数时读事实，否则取
// 屏内表（样例模式只有空表）。
func arcThreads(a *App) []archivedThread {
	if a == nil || !a.wired() {
		return archivedScreen.threads
	}
	facts, ok := a.source.(ArchivedFacts)
	if !ok {
		return nil
	}
	sessions := facts.ArchivedSessions()
	out := make([]archivedThread, 0, len(sessions))
	for _, session := range sessions {
		title := strings.TrimSpace(session.Title)
		if title == "" {
			title = strings.TrimSpace(session.Preview)
		}
		out = append(out, archivedThread{
			id:       session.ID,
			title:    title,
			project:  session.Workspace,
			archived: arcDateLabel(session.ArchivedAt),
		})
	}
	return out
}

// arcDateLabel 把归档时间换成「月 日」：参考产品的归档行只给到日（11 月 20 日）。
// 没有登记时间时返回空串，行上就不写那一段（不造一个日期）。
func arcDateLabel(at time.Time) string {
	if at.IsZero() {
		return ""
	}
	return fmt.Sprintf("%d 月 %d 日", at.Month(), at.Day())
}

// arcHeader 是页首（S-archived-01/02）：标题 + 右端「全部删除」。
func arcHeader(c *gui.Context, t *gui.Theme) {
	gui.Row(c).Shrink(0).AlignItems(gui.Center).Children(func() {
		arcTitleStyle.style(t, gui.Text(c, arcCopyTitle)).TextColor(t.Text).
			Role(gui.RoleHeading).Level(2).SingleLine()
		gui.Spacer(c)
		if arcDeleteAllButton(c, t) {
			archivedScreen.deleteAllOpen = true
		}
	})
}

// arcDeleteAllButton 是「全部删除」（S-archived-02）：危险色描边按钮，
// 没有归档会话时禁用并半透明。
func arcDeleteAllButton(c *gui.Context, t *gui.Theme) bool {
	disabled := len(archivedView.threads) == 0
	b := gui.ButtonBase(c).Shrink(0).Height(arcBtnH).
		Padding(controlPadV, controlPadH).Radius(screenRadiusCtl).
		Label(arcCopyDeleteAll).Disabled(disabled)
	b.BorderWidth(strokeHairline).BorderColor(t.Danger)
	switch {
	case disabled:
		b = b.Opacity(arcAlpha50)
	case b.Pressed():
		b = b.Background(t.Danger.Alpha(arcAlpha08))
	case b.Hovered():
		b = b.Background(t.Danger.Alpha(arcAlpha08))
	}
	b.Children(func() {
		gui.Row(c).Gap(arcChipGap).AlignItems(gui.Center).Children(func() {
			accIcon(c, screenIconTrash, arcTrashIcon, t.Danger, "trash")
			arcBtnStyle.style(t, gui.Text(c, arcCopyDeleteAll)).TextColor(t.Danger).
				SingleLine()
		})
	})
	return b.Clicked() && !disabled
}

// arcAlpha08 是危险色按钮的底色透明度。
const arcAlpha08 = 0.08

// arcBody 按状态分支：加载骨架 / 空态 / 搜索筛选与分组。
func arcBody(c *gui.Context, a *App, t *gui.Theme) {
	switch {
	case archivedScreen.loading:
		arcSkeleton(c, t)
		return
	case archivedScreen.loadError != "":
		arcNoteStyle.style(t, gui.Text(c, archivedScreen.loadError)).
			TextColor(t.Danger).Role(gui.RoleAlert)
		return
	}
	if len(archivedView.threads) == 0 {
		arcEmptyState(c, t, arcCopyEmpty, "")
		return
	}
	arcToolbar(c, t)
	groups := arcGroups()
	if len(groups) == 0 {
		arcEmptyState(c, t, arcCopyNoMatch, arcCopyNoMatchDesc)
		return
	}
	gui.Column(c).Shrink(0).Gap(arcGroupGap).Children(func() {
		for _, group := range groups {
			arcGroup(c, a, t, group)
		}
	})
}

// arcSkeleton 是加载骨架（S-archived-05）：三行等高的面块。
func arcSkeleton(c *gui.Context, t *gui.Theme) {
	gui.Column(c).Shrink(0).Gap(arcSkeletonGap).Label(arcCopyLoading).Children(func() {
		for i := 0; i < arcSkeletonRows; i++ {
			row := gui.Box(c).Shrink(0).Height(arcSkeletonH).
				Background(t.SurfaceHover).Radius(arcCardR)
			row.Label("正在加载").AccessHidden(true)
		}
	})
}

// arcSkeletonRows 是骨架行数（参考 v-for="i in 3"）。
const arcSkeletonRows = 3

// arcEmptyState 是空态（S-archived-15/16/17）：虚线框、归档图标与一句文案。
func arcEmptyState(c *gui.Context, t *gui.Theme, title, desc string) {
	box := gui.Column(c).Shrink(0).Height(arcEmptyH).Gap(arcEmptyGap).
		Center().Radius(arcCardR).Label(title)
	box.BorderWidth(strokeHairline).BorderColor(t.Border)
	box.BorderStyle(gui.BorderDashed)
	box.Children(func() {
		circle := gui.Box(c).Width(arcEmptyBox).Height(arcEmptyBox).Shrink(0).
			Background(t.SurfaceHover).Radius(arcEmptyBox / 2)
		circle.Children(func() {
			gui.Box(c).Grow(1).Center().Children(func() {
				accIcon(c, iconArchive, arcEmptyIcon, t.TextMuted, "")
			})
		})
		arcEmptyTitle.style(t, gui.Text(c, title)).TextColor(t.TextMuted).
			TextAlign(gui.Center)
		if desc != "" {
			arcNoteStyle.style(t, gui.Text(c, desc)).TextColor(t.TextMuted).
				TextAlign(gui.Center)
		}
	})
}

// arcToolbar 是搜索与项目筛选（S-archived-06/07）：有归档会话时才出现。
func arcToolbar(c *gui.Context, t *gui.Theme) {
	gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(arcToolGap).Children(func() {
		search := gui.Row(c).Grow(1).Height(arcCtlH).AlignItems(gui.Center).
			Gap(screenSpace15).Padding(screenSpace15, screenSpace25).
			Radius(screenRadiusCtl).Background(fieldFill(t)).Label(arcCopySearch)
		search.BorderWidth(strokeHairline).BorderColor(t.Border)
		search.Children(func() {
			accIcon(c, iconSearch, arcIconSize, t.TextMuted, "search")
			gui.TextInputBase(c, &archivedScreen.query).Grow(1).
				Label(arcCopySearch).Placeholder(arcCopySearchHolder)
		})
		options := arcProjectOptions()
		// S-archived-08 项目筛选（全部项目 / 无项目 / 各项目名）：选项由归档行的
		// 工作目录生成，没有归档行时只有前两项。
		gui.Select(c, &archivedScreen.projectFilter, options).
			Width(arcFilterW).Height(arcCtlH).Label("项目筛选")
	})
}

// arcProjectOptions 是筛选下拉的选项：全部项目、无项目 + 各项目名。
func arcProjectOptions() []string {
	out := []string{arcCopyAllProjects, arcCopyNoProject}
	seen := map[string]bool{}
	for _, th := range archivedView.threads {
		if th.project == "" || seen[th.project] {
			continue
		}
		seen[th.project] = true
		out = append(out, th.project)
	}
	return out
}

// arcGroupData 是一个项目分组：项目名与筛选后的行。
type arcGroupData struct {
	project string
	threads []archivedThread
}

// arcGroups 按项目分组并应用搜索与筛选。
func arcGroups() []arcGroupData {
	query := strings.TrimSpace(archivedScreen.query)
	order := make([]string, 0, len(archivedView.threads))
	index := map[string]int{}
	for _, th := range archivedView.threads {
		if !arcMatchFilter(th) {
			continue
		}
		if query != "" && !strings.Contains(th.title, query) {
			continue
		}
		if _, ok := index[th.project]; !ok {
			index[th.project] = len(order)
			order = append(order, th.project)
		}
	}
	out := make([]arcGroupData, len(order))
	for i, project := range order {
		out[i].project = project
	}
	for _, th := range archivedView.threads {
		if !arcMatchFilter(th) {
			continue
		}
		if query != "" && !strings.Contains(th.title, query) {
			continue
		}
		i := index[th.project]
		out[i].threads = append(out[i].threads, th)
	}
	return out
}

// arcMatchFilter 判一条会话是否落在当前项目筛选里。
func arcMatchFilter(th archivedThread) bool {
	switch archivedScreen.projectFilter {
	case arcCopyAllProjects, "":
		return true
	case arcCopyNoProject:
		return th.project == ""
	default:
		return th.project == archivedScreen.projectFilter
	}
}

// arcGroup 是一个项目分组（S-archived-09/10/11/12）：分组头 + 行卡。
func arcGroup(c *gui.Context, a *App, t *gui.Theme, group arcGroupData) {
	gui.Column(c).Shrink(0).Gap(arcGroupGap).Children(func() {
		gui.Row(c).Shrink(0).AlignItems(gui.Center).Gap(arcChipGap).Children(func() {
			label := group.project
			if label == "" {
				label = arcCopyNoProject
			}
			arcGroupTitle.style(t, gui.Text(c, label)).TextColor(t.Text).SingleLine()
			arcMetaStyle.style(t, gui.Text(c, arcGroupCount(len(group.threads)))).
				TextColor(t.TextMuted).Label(label)
			gui.Spacer(c)
			arcGroupMenu(c, t, group)
		})
		screenCard(c, func() {
			gui.Column(c).Shrink(0).Children(func() {
				for i := range group.threads {
					if i > 0 {
						screenRule(c)
					}
					arcThreadRow(c, a, t, &group.threads[i])
				}
			})
		})
	})
}

// arcGroupMenu 是分组头的「更多操作」（S-archived-10/11）：原生下拉菜单。
func arcGroupMenu(c *gui.Context, t *gui.Theme, group arcGroupData) {
	project := group.project
	b := gui.ButtonBase(c).Shrink(0).Size(arcIconHit, arcIconHit).
		Radius(radius).Center().Label(arcCopyMore).Tooltip(arcCopyMore)
	switch {
	case b.Pressed():
		b = b.Background(t.SurfacePressed)
	case b.Hovered():
		b = b.Background(t.SurfaceHover)
	}
	b = b.PopoverMenuButton(func(m *gui.Menu) {
		if m.Item(arcCopyDeleteProject).Chosen() {
			archivedScreen.deleteProjectName = project
		}
	})
	b.Children(func() {
		accIcon(c, iconEllipsis, arcIconSize, t.TextMuted, "ellipsis")
	})
}

// arcThreadRow 是一条归档会话（S-archived-12/13/14）：标题、状态、归档时间、
// 恢复与删除。
func arcThreadRow(c *gui.Context, a *App, t *gui.Theme, th *archivedThread) {
	gui.Row(c).Shrink(0).MinHeight(arcRowH).AlignItems(gui.Center).
		Role(gui.RoleListItem).Label(th.title).
		Gap(arcRowGap).Padding(arcRowPadV, arcRowPadH).Children(func() {
		gui.Column(c).Grow(1).MinWidth(0).Gap(arcTextGap).Children(func() {
			arcRowTitle.style(t, gui.Text(c, th.title)).TextColor(t.Text).
				SingleLine()
			gui.Row(c).AlignItems(gui.Center).Gap(arcChipGap).Children(func() {
				arcBadge(c, t, arcCopyClosed)
				if th.archived != "" {
					arcMetaStyle.style(t, gui.Text(c, arcArchivedLabel(th.archived))).
						TextColor(t.TextMuted)
				}
			})
		})
		if arcRestoreButton(c, t) {
			arcRestore(a, th.id)
		}
		if accIconButton(c, accIconSpec{
			icon: screenIconTrash, label: arcCopyDeleteThread,
			tip: arcCopyDeleteThread, size: arcIconSize, hit: arcIconHit,
		}) {
			archivedScreen.deleteThreadID = th.id
		}
	})
}

// arcRestoreButton 是「恢复」（S-archived-13）：描边小按钮 + 回转图标。
func arcRestoreButton(c *gui.Context, t *gui.Theme) bool {
	b := gui.ButtonBase(c).Shrink(0).Height(arcBtnH).
		Padding(controlPadV, controlPadH).Radius(screenRadiusCtl).
		Label(arcCopyRestore).Background(t.Background)
	b.BorderWidth(strokeHairline).BorderColor(t.Border)
	switch {
	case b.Pressed():
		b = b.Background(t.SurfacePressed)
	case b.Hovered():
		b = b.Background(t.SurfaceHover)
	}
	b.Children(func() {
		gui.Row(c).Gap(arcChipGap).AlignItems(gui.Center).Children(func() {
			accIcon(c, arcIconRotate, arcIconSize, t.Text, "rotate-ccw")
			arcBtnStyle.style(t, gui.Text(c, arcCopyRestore)).TextColor(t.Text).
				SingleLine()
		})
	})
	return b.Clicked()
}

// arcBadge 是「已关闭」徽标：中性面、11 字号。
func arcBadge(c *gui.Context, t *gui.Theme, text string) {
	b := gui.Row(c).Shrink(0).AlignItems(gui.Center).
		Padding(arcBadgePadV, arcBadgePadH).Radius(arcBadgeR).
		Background(t.SurfaceHover).Label(text)
	b.BorderWidth(strokeHairline).BorderColor(t.Border)
	b.Children(func() {
		arcMetaStyle.style(t, gui.Text(c, text)).TextColor(t.TextMuted).SingleLine()
	})
}

// arcArchivedLabel 是「{日期} 归档」。
func arcArchivedLabel(date string) string {
	return strings.Replace(arcCopyArchivedAt, "%s", date, 1)
}

// arcRemoveThread 从屏内表里移除一条：只用于样例模式（没有事实可读，行由屏内
// 状态给）。连着 Source 时行由事实驱动，下一帧就不在了，不在这里改。
func arcRemoveThread(id string) {
	out := archivedScreen.threads[:0]
	for _, th := range archivedScreen.threads {
		if th.id != id {
			out = append(out, th)
		}
	}
	archivedScreen.threads = out
}

// arcRestore 恢复一条归档会话：连着 Source 时写 ArchiveSession(id, false)
// （取消本端归档），样例模式只改屏内表。失败原因写在页首错误行。
func arcRestore(a *App, id string) {
	if a == nil || a.intents == nil || !archivedView.wired {
		arcRemoveThread(id)
		return
	}
	archivedScreen.opErr = ""
	a.dispatch("恢复归档会话", func(ctx context.Context) error {
		return a.intents.ArchiveSession(ctx, id, false)
	}, nil)
}

// arcDelete 删除一条或几条归档会话：一条一条发 DeleteSession，第一条失败就停并
// 写原因（后面的不再发，避免把失败掩在一串成功后面）。样例模式只改屏内表。
func arcDelete(a *App, ids []string) {
	archivedScreen.opErr = ""
	if a == nil || a.intents == nil || !archivedView.wired {
		for _, id := range ids {
			arcRemoveThread(id)
		}
		return
	}
	for _, id := range ids {
		sessionID := id
		a.dispatchQuiet("删除归档会话", func(ctx context.Context) error {
			return a.intents.DeleteSession(ctx, sessionID)
		}, nil, func(reason string) {
			archivedScreen.opErr = reason
		})
	}
}

// arcIDs 取一组归档行的会话标识。
func arcIDs(threads []archivedThread) []string {
	out := make([]string, 0, len(threads))
	for _, th := range threads {
		out = append(out, th.id)
	}
	return out
}

// arcDeleteAllDialog 是「删除所有归档会话」（S-archived-03/04）。
func arcDeleteAllDialog(c *gui.Context, t *gui.Theme, a *App) {
	if !archivedScreen.deleteAllOpen {
		return
	}
	threads := archivedView.threads
	open := true
	arcDialog(c, t, &open, arcCopyDeleteAllTitle,
		arcDeleteAllDesc(len(threads)), func() {
			arcDelete(a, arcIDs(threads))
			archivedScreen.threads = nil
			archivedScreen.deleteAllOpen = false
		})
	if !open {
		archivedScreen.deleteAllOpen = false
	}
}

// arcDeleteProjectDialog 是「删除项目内全部会话」（S-archived-11 的对话框）。
func arcDeleteProjectDialog(c *gui.Context, t *gui.Theme, a *App) {
	if archivedScreen.deleteProjectName == "" {
		return
	}
	project := archivedScreen.deleteProjectName
	count := 0
	var doomed []archivedThread
	for _, th := range archivedView.threads {
		if th.project == project {
			count++
			doomed = append(doomed, th)
		}
	}
	open := true
	arcDialog(c, t, &open, arcCopyDeleteProject,
		arcDeleteProjectDesc(project, count), func() {
			arcDelete(a, arcIDs(doomed))
			out := archivedScreen.threads[:0]
			for _, th := range archivedScreen.threads {
				if th.project != project {
					out = append(out, th)
				}
			}
			archivedScreen.threads = out
			archivedScreen.deleteProjectName = ""
		})
	if !open {
		archivedScreen.deleteProjectName = ""
	}
}

// arcDeleteThreadDialog 是「删除归档」（S-archived-14 的对话框）。
func arcDeleteThreadDialog(c *gui.Context, t *gui.Theme, a *App) {
	if archivedScreen.deleteThreadID == "" {
		return
	}
	id := archivedScreen.deleteThreadID
	title := ""
	for _, th := range archivedView.threads {
		if th.id == id {
			title = th.title
			break
		}
	}
	open := true
	arcDialog(c, t, &open, arcCopyDeleteThread, arcDeleteOneDesc(title), func() {
		arcDelete(a, []string{id})
		archivedScreen.deleteThreadID = ""
	})
	if !open {
		archivedScreen.deleteThreadID = ""
	}
}

// arcDialog 是删除类确认框：标题、说明、取消与危险色「删除」。
func arcDialog(c *gui.Context, t *gui.Theme, open *bool, title, desc string, run func()) {
	m := modalShell(gui.Modal(c, open, func() {
		panel := gui.Column(c).Gap(screenSpace3).
			Role(gui.RoleAlertDialog).Label(title)
		panel.Children(func() {
			typeDialogTitle.style(t, gui.Text(c, title)).TextColor(t.Text).
				Role(gui.RoleHeading).Level(3).SingleLine()
			arcNoteStyle.style(t, gui.Text(c, desc)).TextColor(t.TextMuted).
				Selectable()
			gui.Row(c).Justify(gui.End).Gap(arcChipGap).Children(func() {
				if screenButton(c, arcCopyCancel, arcCopyCancel, "", false, rowRegular) {
					*open = false
				}
				if arcDangerButton(c, t, arcCopyDelete) {
					run()
					*open = false
				}
			})
		})
		panel.Width(arcDialogWidth(c))
	}))

	if m.Dismissed() {
		*open = false
	}
}

// arcDangerButton 是危险色实心按钮。
func arcDangerButton(c *gui.Context, t *gui.Theme, text string) bool {
	b := gui.ButtonBase(c).Shrink(0).Height(rowRegular).
		Padding(controlPadV, controlPadH).Radius(screenRadiusCtl).
		Label(text).Background(t.Danger)
	switch {
	case b.Pressed():
		b = b.Background(t.Danger.Alpha(arcAlphaPressed))
	case b.Hovered():
		b = b.Background(t.Danger.Alpha(arcAlphaHover))
	}
	b.Children(func() {
		arcBtnStyle.style(t, gui.Text(c, text)).TextColor(t.InverseText).
			SingleLine()
	})
	return b.Clicked()
}

// arcAlphaPressed / arcAlphaHover 是危险按钮的两态透明度。
const (
	arcAlphaPressed = 0.85
	arcAlphaHover   = 0.92
)

// arcDeleteAllDesc 是「删除所有归档会话」的说明。
func arcDeleteAllDesc(count int) string {
	return fmt.Sprintf(arcCopyDeleteAllDesc, count)
}

// arcDeleteProjectDesc 是「删除项目内全部会话」的说明。
func arcDeleteProjectDesc(project string, count int) string {
	return fmt.Sprintf(arcCopyDeleteProjDesc, project, count)
}

// arcDeleteOneDesc 是单条删除的说明。
func arcDeleteOneDesc(title string) string {
	return strings.Replace(arcCopyDeleteOneDesc, "%s", title, 1)
}

// arcDialogWidth 是对话框面板宽：窗口更窄时收进来（R-ARC-023）。
func arcDialogWidth(c *gui.Context) float32 {
	width, _ := c.Size()
	limit := width - 2*space8
	if limit < arcDialogW {
		if limit < arcDialogMinW {
			return arcDialogMinW
		}
		return limit
	}
	return arcDialogW
}

// arcGroupCount 是分组头右侧的「· {n} 条会话」。
func arcGroupCount(n int) string {
	return "· " + fmt.Sprintf(arcCopySessions, n)
}
