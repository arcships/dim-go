package ui

import "path/filepath"

// 本文件是左栏重建后的视图状态与数据视图：工作区分组、每组的会话行、置顶、
// 折叠、视图菜单与账户菜单。全部是可丢的界面状态，不是事实：会话名、状态标记
// 与时间仍然每帧从 Source（或样例）读，这里只记"谁被折起来、谁被置顶、菜单
// 开没开"这类视图选择。

// sidebarViewState 是左栏的视图状态。
type sidebarViewState struct {
	// collapsed 按工作区标识记折叠态；没有记录时按默认值（内置工作区展开，
	// 其余收起）。
	collapsed map[string]bool
	// pinned 按会话序号记置顶；置顶的会话进「已置顶」，不再重复出现在工作区组里。
	pinned map[int]bool
	// flat 是平铺视图（按日期分组，不按工作区分组）；byCreated 是排序口径
	// （最近创建；默认最近活跃）。
	flat      bool
	byCreated bool
	// viewMenu / accountMenu 是两处菜单的开合（PopoverMenuButton 的 Open）。
	viewMenu    bool
	accountMenu bool
	// groupMenus 按工作区标识记行末菜单的开合。
	groupMenus map[string]*bool
}

// sidebarWorkspace 是左栏里的一个工作区分组：标识、显示名、是否内置（🏠）。
type sidebarWorkspace struct {
	id      string
	name    string
	builtin bool
}

// sidebarThread 是一条会话行要显示的内容：目录里的序号、标题、状态标记与相对
// 时间（相对时间只在样例里有；连着 Source 时为空，不编造）。
type sidebarThread struct {
	index  int
	name   string
	marker string
	when   string
}

// 内置工作区的标识与显示名：参考产品的第一个分组是内置工作区（🏠「工作区」）。
const (
	sidebarBuiltinID   = "builtin"
	sidebarBuiltinName = "工作区"
)

// 本端登记的附加工作区分组的标识前缀：与内置工作区和样例分组区分开。
const workspaceGroupPrefix = "local:"

// workspaceGroupID 是一个本端工作区分组的标识。
func workspaceGroupID(path string) string { return workspaceGroupPrefix + path }

// workspaceGroupName 是工作区分组的显示名：取目录最后一段，根目录用整条路径。
func workspaceGroupName(path string) string {
	name := filepath.Base(path)
	if name == "" || name == "." || name == string(filepath.Separator) {
		return path
	}
	return name
}

// expandGroup 把一个分组标成展开（新登记的工作区直接露出「暂无会话」）。
func (a *App) expandGroup(id string) {
	if a.sidebarView.collapsed == nil {
		a.sidebarView.collapsed = map[string]bool{}
	}
	a.sidebarView.collapsed[id] = false
}

// sidebarSampleWorkspaces 是样例里的分组：内置工作区放全部样例会话，另有两个
// 示例项目分组（默认收起），用来走查折叠与「暂无会话」。
func (a *App) sidebarWorkspaces() []sidebarWorkspace {
	builtin := sidebarWorkspace{id: sidebarBuiltinID, name: sidebarBuiltinName, builtin: true}
	out := make([]sidebarWorkspace, 0, len(a.workspaces)+1)
	out = append(out, builtin)
	// 本端登记的工作区（左栏「添加工作区」选中的目录）跟在内置工作区后面。
	for _, path := range a.workspaces {
		out = append(out, sidebarWorkspace{id: workspaceGroupID(path), name: workspaceGroupName(path)})
	}
	if a.wired() {
		// 目录快照只有会话，没有工作区归属：只画内置工作区与本端登记的目录，
		// 不发明分组。
		return out
	}
	return append(out,
		sidebarWorkspace{id: "p1", name: sampleProjectA},
		sidebarWorkspace{id: "p2", name: sampleProjectB},
	)
}

// groupCollapsed 取分组的折叠态：没有显式记录时内置工作区展开，其余收起
// （与参考产品的初始形态一致：只有内置工作区展开）。
func (a *App) groupCollapsed(id string) bool {
	if open, ok := a.sidebarView.collapsed[id]; ok {
		return open
	}
	return id != sidebarBuiltinID
}

// toggleGroup 折叠 / 展开一个分组。
func (a *App) toggleGroup(id string) {
	if a.sidebarView.collapsed == nil {
		a.sidebarView.collapsed = map[string]bool{}
	}
	a.sidebarView.collapsed[id] = !a.groupCollapsed(id)
}

// collapseAllGroups 收起全部工作区分组（视图菜单的「折叠全部」）。
func (a *App) collapseAllGroups() {
	if a.sidebarView.collapsed == nil {
		a.sidebarView.collapsed = map[string]bool{}
	}
	for _, ws := range a.sidebarWorkspaces() {
		a.sidebarView.collapsed[ws.id] = true
	}
}

// sidebarTimes 是样例里的相对时间；连着 Source 时没有时间事实，全部留空
// （不编造「now」这类内容）。
func (a *App) sidebarTimes(n int) []string {
	out := make([]string, n)
	if a.wired() {
		return out
	}
	sample := [...]string{"now", "1h", "16m"}
	for i := range out {
		out[i] = sample[i%len(sample)]
	}
	return out
}

// sidebarThreads 是当前要显示的全部会话行：已归档的不出现；搜索面开着时只留
// 名称匹配的行（空白查询给全部）。顺序沿用目录顺序。
func (a *App) sidebarThreads() []sidebarThread {
	names := a.sessionNames()
	titles := a.sessionTitles()
	markers := a.sessionMarkers()
	times := a.sidebarTimes(len(names))
	out := make([]sidebarThread, 0, len(names))
	for i := range names {
		if a.archived[i] {
			continue
		}
		display := names[i]
		if i < len(titles) && titles[i] != "" {
			display = titles[i]
		}
		name := a.sessionName(i, display)
		if a.searchOpen && a.searchQuery != "" && !containsFold(name, a.searchQuery) {
			continue
		}
		marker := ""
		if i < len(markers) {
			marker = markers[i]
		}
		when := ""
		if i < len(times) {
			when = times[i]
		}
		out = append(out, sidebarThread{
			index: i, name: name, marker: marker, when: when,
		})
	}
	return sortThreads(out, a.sidebarView.byCreated)
}

// sidebarGroupThreads 取某个工作区组里的会话行。目录没有工作区归属：全部会话
// 归内置工作区，其余分组为空（参考产品里空分组也是常显的形态）。
func (a *App) sidebarGroupThreads(ws sidebarWorkspace, all []sidebarThread) []sidebarThread {
	if ws.id != sidebarBuiltinID {
		return nil
	}
	// 置顶的会话进「已置顶」，不在工作区组里重复。
	out := make([]sidebarThread, 0, len(all))
	for _, thread := range all {
		if a.sidebarView.pinned[thread.index] {
			continue
		}
		out = append(out, thread)
	}
	return out
}

// sidebarPinnedThreads 是「已置顶」里的会话行：按目录顺序，只留仍存在的行。
func (a *App) sidebarPinnedThreads(all []sidebarThread) []sidebarThread {
	out := make([]sidebarThread, 0, len(all))
	for _, thread := range all {
		if a.sidebarView.pinned[thread.index] {
			out = append(out, thread)
		}
	}
	return out
}

// togglePin 置顶 / 取消置顶一条会话（视图状态，不改会话本身）。
func (a *App) togglePin(index int) {
	if a.sidebarView.pinned == nil {
		a.sidebarView.pinned = map[int]bool{}
	}
	a.sidebarView.pinned[index] = !a.sidebarView.pinned[index]
}

// pinned 报告一条会话是否置顶。
func (a *App) pinned(index int) bool { return a.sidebarView.pinned[index] }

// groupMenuPtr 取某个工作区行末菜单的开合指针（按需创建并缓存：每次返回新指针
// 的话，框架把开关写进去后看不到）。
func (a *App) groupMenuPtr(id string) *bool {
	if a.sidebarView.groupMenus == nil {
		a.sidebarView.groupMenus = map[string]*bool{}
	}
	if p, ok := a.sidebarView.groupMenus[id]; ok {
		return p
	}
	v := false
	p := &v
	a.sidebarView.groupMenus[id] = p
	return p
}

// accountNameValue 是账户行显示的名字：连着 Source 时取账号事实，否则用样例。
func (a *App) accountNameValue() string {
	if a.source != nil {
		if name := a.source.Account().Name; name != "" {
			return name
		}
		return a.source.Account().Email
	}
	return "eric"
}
