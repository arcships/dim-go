package ui

import (
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// 本文件是左栏重建后的视图状态与数据视图：工作区分组、每组的会话行、置顶、
// 折叠、视图菜单与账户菜单。全部是可丢的界面状态，不是事实：会话名、状态标记
// 与时间仍然每帧从 Source（或样例）读，这里只记"谁被折起来、谁被置顶、菜单
// 开没开"这类视图选择。

// sidebarViewState 是左栏的视图状态。
type sidebarViewState struct {
	// collapsed 按工作区标识记折叠态；没有记录时按默认值（内置工作区展开，
	// 其余收起）。
	collapsed map[string]bool
	// pinned 按会话标识（目录里的 id）记置顶；置顶的会话进「已置顶」，不再重复
	// 出现在工作区组里。按标识而不是行号记：列表按活跃时间重排后行号会换人，
	// 按行号记会把置顶跟到别的会话上。
	//
	// 进程内登记（DEC-18、gaps M-05）：不做服务端事实，也不进本地偏好——重启后
	// 丢失是已登记的口径，不是漏写。
	pinned map[string]bool
	// pinnedGroups 按工作区标识记置顶；置顶的工作区分组排在其余分组之前。
	// 与 pinned 一样是本地视图状态：没有置顶事实，也不写服务端。
	pinnedGroups map[string]bool
	// groupNames 按工作区标识记改过的显示名（LB-48 的重命名）；没有记录时取
	// 目录末段。
	groupNames map[string]string
	// hiddenGroups 按目录记「移除工作区」后不再显示的组：本端登记的目录同时从
	// workspaces 里删掉，有会话的目录再记一条隐藏，列表里不再出现它。
	hiddenGroups map[string]bool
	// flat 是平铺视图（按日期分组，不按工作区分组）；byCreated 是排序口径
	// （最近创建；默认最近活跃）。
	flat      bool
	byCreated bool
	// viewMenu / accountMenu 是两处菜单的开合（PopoverMenuButton 的 Open）。
	viewMenu    bool
	accountMenu bool
	// groupMenus 按工作区标识记行末菜单的开合。
	groupMenus map[string]*bool
	// hoveredGroup 是上一帧指针停留的工作区分组：F2 用它决定行内改名改哪一组。
	hoveredGroup string
}

// sidebarWorkspace 是左栏里的一个分组：标识、显示名、目录路径、是否内置（🏠）、
// 组内会话数与最近活动时间（排序用）。
type sidebarWorkspace struct {
	id      string
	name    string
	builtin bool
	// path 是这一组的目录（内置「工作区」与样例空组没有目录）。
	path string
	// count 是组里显示的会话数（置顶的不算，它们在「已置顶」里）；active 是
	// 组里最近的活动时间。
	count  int
	active time.Time
}

// sidebarThread 是一条会话行要显示的内容：目录里的序号、标题、状态标记与相对
// 时间（相对时间只在样例里有；连着 Source 时为空，不编造）；cwd / active 是
// 分组用的事实（工作目录与最近活动时间），不上屏。
type sidebarThread struct {
	index  int
	name   string
	marker string
	when   string
	cwd    string
	active time.Time
}

// 内置工作区的标识与显示名：参考产品的第一个分组是内置工作区（🏠「工作区」）。
const (
	sidebarBuiltinID   = "builtin"
	sidebarBuiltinName = "工作区"
)

// 本端登记的附加工作区分组的标识前缀：与内置工作区和样例分组区分开。
const workspaceGroupPrefix = "local:"

// workspaceGroupID 是一个工作区分组的标识（按目录路径，折叠态按它记）。
func workspaceGroupID(path string) string { return workspaceGroupPrefix + path }

// workspaceGroupName 是工作区分组的显示名：取目录最后一段，根目录用整条路径。
func workspaceGroupName(path string) string {
	name := filepath.Base(path)
	if name == "" || name == "." || name == string(filepath.Separator) {
		return path
	}
	return name
}

// cleanGroupPath 把会话的 cwd 归一成分组的键：去掉首尾空白后取干净路径；空表示
// 没有工作目录（归内置「工作区」）。
func cleanGroupPath(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	return filepath.Clean(path)
}

// expandGroup 把一个分组标成展开（新登记的工作区直接露出「暂无会话」）。
func (a *App) expandGroup(id string) {
	if a.sidebarView.collapsed == nil {
		a.sidebarView.collapsed = map[string]bool{}
	}
	a.sidebarView.collapsed[id] = false
}

// sidebarSampleGroups 是样例里的两个空目录分组（默认收起）：走查文件夹图标、
// 折叠与「暂无会话」；接上 Source 时不出现。
func sidebarSampleGroups() []sidebarWorkspace {
	return []sidebarWorkspace{
		{id: "p1", name: sampleProjectA},
		{id: "p2", name: sampleProjectB},
	}
}

// sidebarWorkspaces 建左栏的分组：内置「工作区」在前（没有 cwd 的会话），其余
// 按会话的工作目录分组，组名取目录末段。本端登记的目录（「添加工作区」）即使
// 没有会话也留一组，与有会话的同名目录合成一组。
//
// 排序：内置「工作区」永远第一；其余按会话数（多在前）、再按最近活动时间；
// 没有会话的空组排在后面（同档内保持登记 / 出现的先后）。
func (a *App) sidebarWorkspaces(all []sidebarThread) []sidebarWorkspace {
	builtin := sidebarWorkspace{id: sidebarBuiltinID, name: sidebarBuiltinName, builtin: true}
	groups := make([]sidebarWorkspace, 0, len(a.workspaces)+1)
	index := make(map[string]int, len(a.workspaces)+1)
	add := func(path string) *sidebarWorkspace {
		path = cleanGroupPath(path)
		if path == "" {
			return nil
		}
		if i, ok := index[path]; ok {
			return &groups[i]
		}
		groups = append(groups, sidebarWorkspace{
			id: workspaceGroupID(path), name: workspaceGroupName(path), path: path,
		})
		index[path] = len(groups) - 1
		return &groups[len(groups)-1]
	}
	// 本端登记的目录先建组：没有会话也常显；被「移除工作区」藏起来的目录不再建组。
	for _, path := range a.workspaces {
		if a.groupHiddenPath(path) {
			continue
		}
		add(path)
	}
	for _, thread := range all {
		// 置顶的会话在「已置顶」里，不算进目录组的会话数。
		if a.threadPinned(thread) {
			continue
		}
		if cleanGroupPath(thread.cwd) == "" {
			builtin.count++
			if thread.active.After(builtin.active) {
				builtin.active = thread.active
			}
			continue
		}
		// 工作区被移除后，它名下的会话不再出现在列表里（不伪造归档）。
		if a.groupHiddenPath(thread.cwd) {
			continue
		}
		if group := add(thread.cwd); group != nil {
			group.count++
			if thread.active.After(group.active) {
				group.active = thread.active
			}
		}
	}
	if !a.wired() {
		groups = append(groups, sidebarSampleGroups()...)
	}
	// 行内改名改的是显示名，不碰目录：按标识取改过的名字。
	for i := range groups {
		if name, ok := a.sidebarView.groupNames[groups[i].id]; ok && name != "" {
			groups[i].name = name
		}
	}
	// 置顶的工作区分组排在其余分组之前，组内相对顺序不变。
	sort.SliceStable(groups, func(i, j int) bool {
		if pinI, pinJ := a.groupPinned(groups[i]), a.groupPinned(groups[j]); pinI != pinJ {
			return pinI
		}
		if groups[i].count != groups[j].count {
			return groups[i].count > groups[j].count
		}
		if !groups[i].active.Equal(groups[j].active) {
			return groups[i].active.After(groups[j].active)
		}
		return false
	})
	return append([]sidebarWorkspace{builtin}, groups...)
}

// groupPinned 报告一个工作区分组是否置顶。
func (a *App) groupPinned(ws sidebarWorkspace) bool {
	return a.sidebarView.pinnedGroups[ws.id]
}

// pinnedGroupsOf 挑出置顶的工作区分组，顺序沿用分组顺序（内置组不参与置顶）。
func (a *App) pinnedGroupsOf(groups []sidebarWorkspace) []sidebarWorkspace {
	out := make([]sidebarWorkspace, 0, len(groups))
	for _, ws := range groups {
		if !ws.builtin && a.groupPinned(ws) {
			out = append(out, ws)
		}
	}
	return out
}

// toggleGroupPin 置顶 / 取消置顶一个工作区分组（本地视图状态）。
func (a *App) toggleGroupPin(ws sidebarWorkspace) {
	if ws.builtin || ws.id == "" {
		return
	}
	if a.sidebarView.pinnedGroups == nil {
		a.sidebarView.pinnedGroups = map[string]bool{}
	}
	a.sidebarView.pinnedGroups[ws.id] = !a.sidebarView.pinnedGroups[ws.id]
	a.persistViewState()
}

// groupHiddenPath 报告一个目录是否被「移除工作区」藏起来。
func (a *App) groupHiddenPath(path string) bool {
	path = cleanGroupPath(path)
	if path == "" || len(a.sidebarView.hiddenGroups) == 0 {
		return false
	}
	return a.sidebarView.hiddenGroups[path]
}

// renameGroup 保存组头的行内改名（LB-48）：空输入回落原名称，不留空标题。
// 显示名是本地视图状态，不改目录。
func (a *App) renameGroup(ws sidebarWorkspace) {
	name := strings.TrimSpace(a.groupRenameDraft)
	a.renamingGroup, a.groupRenameDraft = "", ""
	if name == "" || ws.builtin || ws.id == "" {
		return
	}
	if a.sidebarView.groupNames == nil {
		a.sidebarView.groupNames = map[string]string{}
	}
	a.sidebarView.groupNames[ws.id] = name
	a.persistViewState()
}

// removeWorkspaceGroup 移除一个工作区分组（LB-49）：本端登记的目录同时从
// workspaces 里删掉；组里还有会话时把该目录记成隐藏，列表里不再出现它的行。
// 当前会话若属于它，主区退回空态（不留在已移除的目录上）。
func (a *App) removeWorkspaceGroup(ws sidebarWorkspace) {
	if ws.builtin || ws.path == "" {
		return
	}
	path := cleanGroupPath(ws.path)
	kept := a.workspaces[:0]
	for _, registered := range a.workspaces {
		if cleanGroupPath(registered) == path {
			continue
		}
		kept = append(kept, registered)
	}
	a.workspaces = kept
	if a.sidebarView.hiddenGroups == nil {
		a.sidebarView.hiddenGroups = map[string]bool{}
	}
	a.sidebarView.hiddenGroups[path] = true
	delete(a.sidebarView.pinnedGroups, ws.id)
	delete(a.sidebarView.collapsed, ws.id)
	if current := a.currentWorkspaceCwd(); cleanGroupPath(current) == path {
		a.SetState(StateEmpty)
	}
	a.persistViewState()
}

// groupCollapsed 取分组的折叠态：没有显式记录时有会话的组展开、空组收起
// （内置「工作区」默认展开，参考产品的第一个分组总是露出来的）。搜索命中时
// 由调用方强制展开，不改这里记下的选择。
func (a *App) groupCollapsed(ws sidebarWorkspace) bool {
	if collapsed, ok := a.sidebarView.collapsed[ws.id]; ok {
		return collapsed
	}
	return !ws.builtin && ws.count == 0
}

// toggleGroup 折叠 / 展开一个分组（视图状态；收起态随偏好落盘）。
func (a *App) toggleGroup(ws sidebarWorkspace) {
	if a.sidebarView.collapsed == nil {
		a.sidebarView.collapsed = map[string]bool{}
	}
	a.sidebarView.collapsed[ws.id] = !a.groupCollapsed(ws)
	a.persistViewState()
}

// sidebarSearching 表示搜索面开着并且有关键字：命中跨组显示，组自动展开。
func (a *App) sidebarSearching() bool { return a.searchOpen && a.searchQuery != "" }

// collapseAllGroups 收起全部工作区分组（视图菜单的「折叠全部」）。
func (a *App) collapseAllGroups() {
	if a.sidebarView.collapsed == nil {
		a.sidebarView.collapsed = map[string]bool{}
	}
	for _, ws := range a.sidebarWorkspaces(a.sidebarThreads()) {
		a.sidebarView.collapsed[ws.id] = true
	}
	a.persistViewState()
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
// 名称匹配的行（空白查询给全部）。顺序沿用目录顺序，并带上分组用的事实
// （工作目录 / 最近活动时间）。
func (a *App) sidebarThreads() []sidebarThread {
	names := a.sessionNames()
	titles := a.sessionTitles()
	markers := a.sessionMarkers()
	times := a.sidebarTimes(len(names))
	places := a.sessionPlaces()
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
		thread := sidebarThread{
			index: i, name: name, marker: marker, when: when,
		}
		if i < len(places) {
			thread.cwd = places[i].Cwd
			thread.active = places[i].UpdatedAt
		}
		// 被「移除工作区」藏起来的目录，其会话在任何视图（分组 / 平铺 / 置顶）
		// 里都不再出现。
		if a.groupHiddenPath(thread.cwd) {
			continue
		}
		out = append(out, thread)
	}
	return sortThreads(out, a.sidebarView.byCreated)
}

// unpinnedThreads 是没置顶的会话行：置顶的会话进「已置顶」，不在目录组里重复。
func (a *App) unpinnedThreads(all []sidebarThread) []sidebarThread {
	out := make([]sidebarThread, 0, len(all))
	for _, thread := range all {
		if a.threadPinned(thread) {
			continue
		}
		out = append(out, thread)
	}
	return out
}

// sidebarGroupThreads 取某个目录组里的会话行：内置「工作区」是没有 cwd 的会话，
// 其余组按 cwd 归到同一目录（路径先归一，末尾分隔符与相对写法不影响归属）。
func (a *App) sidebarGroupThreads(ws sidebarWorkspace, all []sidebarThread) []sidebarThread {
	path := cleanGroupPath(ws.path)
	if !ws.builtin && path == "" {
		return nil
	}
	threads := a.unpinnedThreads(all)
	out := make([]sidebarThread, 0, len(threads))
	for _, thread := range threads {
		cwd := cleanGroupPath(thread.cwd)
		if ws.builtin {
			if cwd != "" {
				continue
			}
		} else if cwd != path {
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
		if a.threadPinned(thread) {
			out = append(out, thread)
		}
	}
	return out
}

// threadPinned 报告一条会话行是否置顶：换行号找会话标识，不按行号记。
func (a *App) threadPinned(thread sidebarThread) bool {
	id := a.sessionIDAt(thread.index)
	return id != "" && a.sidebarView.pinned[id]
}

// togglePin 置顶 / 取消置顶一条会话（视图状态，不改会话本身）。它按会话标识
// 记，重排后仍指同一条会话；没有标识（目录越界）时不动状态。
func (a *App) togglePin(index int) {
	id := a.sessionIDAt(index)
	if id == "" {
		return
	}
	if a.sidebarView.pinned == nil {
		a.sidebarView.pinned = map[string]bool{}
	}
	a.sidebarView.pinned[id] = !a.sidebarView.pinned[id]
}

// pinned 报告一条会话是否置顶。
func (a *App) pinned(index int) bool {
	id := a.sessionIDAt(index)
	return id != "" && a.sidebarView.pinned[id]
}

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
// 名字与邮箱都为空时给占位，不留空行（LB-90）。
func (a *App) accountNameValue() string {
	if a.source != nil {
		if name := a.source.Account().Name; name != "" {
			return name
		}
		if email := a.source.Account().Email; email != "" {
			return email
		}
		return "未命名账户"
	}
	return "eric"
}

// accountStatusPhrase 是账户行的读屏状态说明：登录状态与账户名。头像只是首字母，
// 不单独播报。
func (a *App) accountStatusPhrase() string {
	if a.source != nil && !a.source.Account().Connected {
		return "未登录，打开账户菜单可登录"
	}
	return "已登录 " + a.accountNameValue()
}
