package ui

import (
	"strings"

	gui "github.com/egoist/mygo/ui"
)

// `@` 文件提及与 `$` 技能提及弹层（requirements-global §2.3、§2.4，
// G-A3-01–G-A3-11、G-A4-01–G-A4-05）。
//
// 触发只看草稿：行中任意位置键入 `@` 或 `$`，其后的那段前缀不含空白就弹出；
// 两者互斥——取靠后的那一个触发位，同一时刻只开一个弹层。弹层贴在输入卡上沿
// （与斜杠命令弹层同一位置），↑↓ 选择、↵ 插入、Esc 收起。
//
// 候选来自既有读数，不新增协议方法，也不新增 Source 接口：
//
//   - 文件：**目录投影为主、内容搜索的命中路径补足**。主源是右栏与命令面板用的
//     同一份目录投影（PanelFacts.FileTree 的已读页面：名字、目录 / 文件、路径都在），
//     补足源是 PanelFacts.FileSearch 的命中路径（一次工作区扫描，能把尚未展开的
//     目录里的同名文件带进来）。
//
//     这是刻意的判断：`@` 是**文件名选择器**，不是内容检索。只用 FileSearch 的话，
//     空前缀列不出任何文件（内容搜索要求非空关键词），`@composer` 也只能列出
//     「内容里含 composer」的文件而漏掉按名字要选的那个——功能等于不做。反过来，
//     只用目录投影则只覆盖已读到的分页（根目录 20 条 + 已展开的子目录），深层
//     文件选不到。两者合起来各补一半，缺口写在候选行的上限说明与 pickCapNote 里。
//
//   - 技能：Extensions().Skills（与设置 › 插件设置、输入区技能浮层同一份读数）。
//
// 与另外两个搜索面的分工：⌘F 是**会话内**搜索（search.go，浮在标题栏），⌘G 是
// **跨会话**的搜索聊天对话框（search_chat.go，模态），`@` / `$` 只作用于输入区
// 草稿。三者的触发源互不相同（键位 / 草稿前缀），不会同时开两个。
//
// 视图状态（开合、前缀、选中行、Esc 记住的前缀）全部可丢，不落盘。

const (
	// 弹层宽：`@` 取 min(35rem, 100vw−2×7)=560，`$` 取 min(24rem, …)=384。
	pickFileWidth  = 560
	pickSkillWidth = 384
	// 候选行 32 高，一屏最多 8 行，多出来的在弹层内滚动。
	pickRowHeight    = 32
	pickMaxRows      = 8
	pickMaxHeight    = pickRowHeight * pickMaxRows
	pickSkeletonRows = 2
	pickPad          = space2
	pickGap          = space1
	// pickFileCap 是文件候选的扫描上限（与命令面板的文件组同一口径）。
	pickFileCap = 5000
	// pickDetailMax 是右尾（目录 / 描述）的最大宽：主字完整，右尾截断。
	pickDetailMax = 220
	// pickMinWidth 是窄窗里弹层的下限：再窄下去候选的目录与名字都读不出。
	pickMinWidth = 240

	// 文案。
	pickGroupFile       = "文件"
	pickGroupSkill      = "技能"
	pickNoFileText      = "未找到文件或目录"
	pickNoWorkspaceText = "当前无工作区目录"
	pickFileFailedText  = "文件搜索失败。"
	pickNoSkillText     = "未找到技能"
	pickSkillFailedText = "读取技能失败："
	pickRetryText       = "重试"
	pickRetryFileText   = "重试文件搜索"
	pickRetrySkillText  = "重试技能读取"
	// pickRetryNoChannel 是技能读数没有重试通道时的原因（没有刷新技能的读数入口
	// 之外的路径：刷新走 /reload-skills，这里不给第二个写入口）。
	pickRetryNoChannel = "技能清单没有单独的刷新入口，用 /reload-skills 重新扫描"
	pickSearchingText  = "正在搜索"
	// pickCapNote 是候选到扫描上限时的说明（G-A3 的文件数量上限）。
	pickCapNote = "工作区文件超过 5000 个，候选只取前 5000 个"
)

// 候选的二级说明（读屏名里带上「文件 / 目录」）。
const (
	pickWordFile  = "文件"
	pickWordDir   = "目录"
	pickWordSkill = "技能"
)

// pickKind 是弹层当前列的是什么。
type pickKind int

const (
	pickNone pickKind = iota
	pickFile
	pickSkill
)

// pickItem 是一条候选。
type pickItem struct {
	kind pickKind
	// path 是文件的相对路径（'/' 分隔，相对会话工作目录）；技能为空。
	path string
	// name 是文件 / 目录名，或技能名。
	name string
	// dir 表示目录候选（主字尾部补 '/'，回车后留在弹层里继续追问子目录）。
	dir bool
	// detail 是右尾：文件候选给所在目录，技能候选给描述。
	detail string
	// nameStart / nameEnd 与 detailStart / detailEnd 是查询命中的字节区间；
	// -1 表示这一段没有命中。
	nameStart, nameEnd     int
	detailStart, detailEnd int
	// insert 是插进草稿的文本（含 `@` / `$`，不含尾部空白）。
	insert string
}

// pickRead 是一组候选的读态。
type pickRead struct {
	loading bool
	failed  bool
	reason  string
	// empty 是这一组为空时的说明（「未找到文件或目录」等）。
	empty string
	// note 是候选的来源说明（截断、扫描上限）。
	note string
}

// pickState 是弹层的全部视图状态（可丢）。
type pickState struct {
	open bool
	kind pickKind
	// at 是触发字符（`@` / `$`）在草稿里的字节下标；query 是其后到草稿末尾的前缀。
	at    int
	query string

	// dismissed / dismissedFor：Esc 收起后，同样的「类 + 前缀」不再自动弹回来。
	dismissed    bool
	dismissedFor string

	// files / skills 是两组的候选；items 是两组合起来的键盘顺序（= 渲染顺序）。
	files  []pickItem
	skills []pickItem
	items  []pickItem
	// fileRead / skillRead 是两组的读态。
	fileRead  pickRead
	skillRead pickRead

	selected int
	// refresh 是这一帧要用「跳过缓存重读」的文件读数（重试按钮）。
	refresh bool
}

// reset 收起弹层并清掉本次的全部视图状态（不含 Esc 记住的前缀）。
func (s *pickState) reset() {
	s.open = false
	s.kind = pickNone
	s.at = 0
	s.query = ""
	s.files, s.skills, s.items = nil, nil, nil
	s.fileRead, s.skillRead = pickRead{}, pickRead{}
	s.selected = 0
}

// dismiss 是 Esc：收起弹层，记住当前「类 + 前缀」，草稿不动（不吞字）。
func (s *pickState) dismiss() {
	s.dismissed = true
	s.dismissedFor = pickKey(s.kind, s.query)
	s.reset()
}

// pickKey 是 Esc 记住的键：同一类、同一前缀才不重新弹出。
func pickKey(kind pickKind, query string) string {
	switch kind {
	case pickFile:
		return "f:" + query
	case pickSkill:
		return "s:" + query
	}
	return ""
}

// pickTriggerOf 取草稿末尾的触发位：`@` 或 `$`，其前是行首或空白、其后到末尾
// 不含空白。两者都在时取靠后的那一个（互斥，见 G-A3-01 / G-A4-01 的「弹层计数为 1」）。
func pickTriggerOf(draft string) (kind pickKind, at int, query string, ok bool) {
	for i := len(draft) - 1; i >= 0; i-- {
		switch draft[i] {
		case ' ', '\t', '\n':
			// 触发位到末尾之间出现空白：这一段已经写完了，弹层不该再开。
			return pickNone, 0, "", false
		case '@', '$':
			if i > 0 {
				switch draft[i-1] {
				case ' ', '\t', '\n':
				default:
					// 例如 `a@b`：不是提及的触发位，继续往前找。
					continue
				}
			}
			kind = pickFile
			if draft[i] == '$' {
				kind = pickSkill
			}
			return kind, i, draft[i+1:], true
		}
	}
	return pickNone, 0, "", false
}

// syncPick 每帧按草稿决定弹层开合。草稿是唯一的触发来源：没有另一个「打开
// 弹层」的入口，也就不存在弹层与输入不一致的状态。
func (a *App) syncPick() {
	s := &a.pick
	kind, at, query, ok := pickTriggerOf(a.draft)
	// 斜杠命令弹层占着输入框时不再开提及弹层：两者是同一处的两个候选表。
	// 判据与斜杠弹层同源（都从草稿算），不依赖两个同步函数的先后。
	if _, slashOpen := slashMenuPrefix(a.draft); !ok || slashOpen {
		s.reset()
		return
	}
	if s.open {
		if s.kind != kind || s.query != query {
			// 继续输入：候选取新前缀，选中行回到第一行（G-A3-11 / G-A4-05）。
			s.kind, s.at, s.query = kind, at, query
			s.selected = 0
			s.dismissed, s.dismissedFor = false, ""
		}
		// 每帧重取候选：目录投影与技能读数都在别的帧里变化（读态、分页、刷新），
		// 只在开合时取一次会把「还没读到的失败」留在屏上。
		a.loadPickItems()
		return
	}
	if s.dismissed && s.dismissedFor == pickKey(kind, query) {
		// Esc 关掉之后，同样的前缀不再自动弹出；再键入字符才会回来。
		return
	}
	s.open = true
	s.kind, s.at, s.query = kind, at, query
	s.selected = 0
	s.dismissed, s.dismissedFor = false, ""
	a.loadPickItems()
}

// pickFrame 每帧先做复位：离开主界面或断线时弹层不在场，状态一并清掉。
func (a *App) pickFrame() {
	s := &a.pick
	if !s.open {
		return
	}
	if a.screen != ScreenMain || a.disconnected() {
		s.reset()
		return
	}
	// 连着 Source 但没有打开的会话：草稿不属于任何会话，弹层收起来。
	if _, ok := a.liveSession(); !ok && a.wired() {
		s.reset()
	}
}

// loadPickItems 取两组候选并按前缀过滤。
func (a *App) loadPickItems() {
	s := &a.pick
	if s.kind == pickFile {
		files, read := a.pickFileState(s.query, s.refresh)
		s.files, s.fileRead = files, read
		skills, skillRead := a.pickSkillState(s.query)
		s.skills, s.skillRead = skills, skillRead
	} else {
		skills, read := a.pickSkillState(s.query)
		s.skills, s.skillRead = skills, read
		s.files, s.fileRead = nil, pickRead{}
	}
	s.items = append(append([]pickItem(nil), s.files...), s.skills...)
	if len(s.items) == 0 {
		s.selected = 0
		return
	}
	if s.selected < 0 || s.selected >= len(s.items) {
		s.selected = 0
	}
}

// movePick 上下移动选中行，越界不跨组回卷。
func (a *App) movePick(delta int) {
	s := &a.pick
	n := len(s.items)
	if n == 0 {
		return
	}
	next := s.selected + delta
	if next < 0 {
		next = 0
	}
	if next >= n {
		next = n - 1
	}
	s.selected = next
}

// pickTextKey 接管输入框里的键：↑↓ 选择、↵ 插入、Esc 收起。
func (a *App) pickTextKey(k gui.TextKey) bool {
	s := &a.pick
	if !s.open {
		return false
	}
	switch {
	case k.Mods == 0 && k.Key == gui.KeyEscape:
		s.dismiss()
		return true
	case k.Mods == 0 && k.Key == gui.KeyUp:
		a.movePick(-1)
		return true
	case k.Mods == 0 && k.Key == gui.KeyDown:
		a.movePick(1)
		return true
	case k.Mods == 0 && k.Key == gui.KeyEnter:
		if a.pickLoading() || len(s.items) == 0 || s.selected < 0 ||
			s.selected >= len(s.items) {
			// 骨架期间没有可选项：回车不插入任何内容（G-A3-02 / G-A4-02）。
			return false
		}
		a.insertPick(s.items[s.selected])
		return true
	}
	return false
}

// pickLoading 表示这一组候选还没读回来。
func (a *App) pickLoading() bool {
	s := &a.pick
	if s.kind == pickFile {
		return s.fileRead.loading
	}
	return s.skillRead.loading
}

// insertPick 把候选插进草稿：替换触发位到草稿末尾的那一段（`@前缀` / `$前缀`）。
// 目录候选插完仍留在弹层里，可以接着追问子目录；文件与技能插完就收起。
func (a *App) insertPick(item pickItem) {
	s := &a.pick
	at := s.at
	if at < 0 || at > len(a.draft) {
		// 触发位已经不在草稿里（例如同时被清空）：收起来，不写半截文本。
		s.reset()
		return
	}
	a.draft = a.draft[:at] + item.insert
	if item.dir {
		// 留在弹层里：下一帧的触发位就在刚插入的目录上，候选换成它的子项。
		s.selected = 0
		s.dismissed, s.dismissedFor = false, ""
		return
	}
	s.reset()
}

// —— 文件候选 ——

// pickFileState 摊平已读到的目录页，再用内容搜索的命中路径补足未展开的目录。
func (a *App) pickFileState(query string, refresh bool) ([]pickItem, pickRead) {
	read := pickRead{empty: pickNoFileText}
	if a.wired() {
		if title, _ := a.rpNoWorkspace(); title != "" {
			// 会话没有工作目录：不派发读数，空态写清原因（G-A3-03）。
			read.empty = pickNoWorkspaceText
			return nil, read
		}
	}
	q := strings.ToLower(strings.TrimSpace(query))
	items, scanned := pickTreeItems(a.rp().tree, "", q, pickFileCap)
	seen := make(map[string]bool, len(items))
	for _, item := range items {
		seen[item.path] = true
	}
	// 目录投影只覆盖已经读到的页：再用一次工作区内容搜索的命中路径补足。
	if facts, ok := a.rpFacts(); ok {
		if q != "" {
			snapshot := facts.FileSearch(FileSearchRequest{
				Cwd: a.rpCwd(), Query: query, Limit: rpSearchLimit, Refresh: refresh,
			})
			switch snapshot.Phase {
			case PanelReadLoading, PanelReadIdle:
				// 内容搜索还在跑：已经读到的目录页照常给，不把这一组说成加载中。
			case PanelReadFailed:
				if len(items) == 0 {
					read.failed = true
					read.reason = snapshot.Error
				}
			default:
				for _, path := range pickHitPaths(snapshot.Hits) {
					for _, item := range pickPathItems(path, q) {
						if seen[item.path] {
							continue
						}
						seen[item.path] = true
						items = append(items, item)
						scanned++
					}
				}
				if snapshot.Truncated && snapshot.StopReason == "max-files" {
					read.note = pickCapNote
				}
			}
			if snapshot.FilesScanned > pickFileCap && read.note == "" {
				read.note = pickCapNote
			}
		}
	}
	if read.failed {
		return nil, read
	}
	if a.wired() && a.rp().rootFailed && len(items) == 0 {
		read.failed = true
		read.reason = a.rp().rootError
		return nil, read
	}
	if a.wired() && a.rp().loading && len(items) == 0 && q == "" {
		read.loading = true
		return nil, read
	}
	if scanned >= pickFileCap && read.note == "" {
		read.note = pickCapNote
	}
	if len(items) == 0 {
		return nil, read
	}
	return items, read
}

// pickTreeItems 把一棵目录投影摊平成候选：目录先于它的子项，最多扫 limit 个。
func pickTreeItems(nodes []*rpNode, dir, q string, limit int) ([]pickItem, int) {
	var items []pickItem
	scanned := 0
	var walk func(nodes []*rpNode, dir string)
	walk = func(nodes []*rpNode, dir string) {
		for _, node := range nodes {
			if scanned >= limit {
				return
			}
			scanned++
			path := rpJoinPath(dir, node.name)
			if item, ok := pickPathItem(path, node.name, node.dir, q); ok {
				items = append(items, item)
			}
			if node.dir && len(node.children) > 0 {
				walk(node.children, path)
			}
		}
	}
	walk(nodes, dir)
	return items, scanned
}

// pickPathItems 是一条命中路径带来的候选：路径本身，加上路径上的各级目录。
func pickPathItems(path, q string) []pickItem {
	var items []pickItem
	name := rpBaseName(path)
	if item, ok := pickPathItem(path, name, false, q); ok {
		items = append(items, item)
	}
	dir := rpDirName(path)
	for dir != "" && dir != "." {
		if item, ok := pickPathItem(dir, rpBaseName(dir), true, q); ok {
			items = append(items, item)
		}
		dir = rpDirName(dir)
	}
	return items
}

// pickPathItem 把一条路径变成候选：查询不在路径里就不算候选。
func pickPathItem(path, name string, dir bool, q string) (pickItem, bool) {
	if name == "" {
		return pickItem{}, false
	}
	item := pickItem{
		kind: pickFile, path: path, name: name, dir: dir,
		detail:      rpDirName(path),
		nameStart:   -1,
		nameEnd:     -1,
		detailStart: -1,
		detailEnd:   -1,
	}
	if item.detail == "." {
		// 根目录下的候选没有次级行（G-A3-06：`.` 不显示）。
		item.detail = ""
	}
	if q != "" {
		if spans := searchFindAll(name, q); len(spans) > 0 {
			item.nameStart, item.nameEnd = spans[0][0], spans[0][1]
		} else if item.detail != "" {
			if spans := searchFindAll(item.detail, q); len(spans) > 0 {
				item.detailStart, item.detailEnd = spans[0][0], spans[0][1]
			} else {
				return pickItem{}, false
			}
		} else {
			return pickItem{}, false
		}
	}
	item.insert = "@" + path
	if dir {
		item.insert += "/"
	} else {
		item.insert += " "
	}
	return item, true
}

// pickHitPaths 取内容搜索命中里的去重路径（按扫描顺序）。
func pickHitPaths(hits []FileHitSnapshot) []string {
	seen := make(map[string]bool, len(hits))
	paths := make([]string, 0, len(hits))
	for _, hit := range hits {
		path := strings.TrimSpace(hit.Path)
		if path == "" || seen[path] {
			continue
		}
		seen[path] = true
		paths = append(paths, path)
	}
	return paths
}

// rpBaseName 取路径的末段（'/' 分隔，不碰文件系统）。
func rpBaseName(path string) string {
	if at := strings.LastIndexByte(path, '/'); at >= 0 {
		return path[at+1:]
	}
	return path
}

// rpDirName 取路径的目录部分；没有目录时返回 "."。
func rpDirName(path string) string {
	if at := strings.LastIndexByte(path, '/'); at >= 0 {
		return path[:at]
	}
	return "."
}

// —— 技能候选 ——

// pickSkillState 读 Extensions().Skills 并按前缀过滤：禁用的不列、同名只留
// 第一条（服务端给的顺序即优先级）、没有描述的只显示名称。
func (a *App) pickSkillState(query string) ([]pickItem, pickRead) {
	read := pickRead{empty: pickNoSkillText}
	skills, readErr := a.skillEntries()
	if readErr != "" {
		read.failed = true
		read.reason = readErr
		return nil, read
	}
	if skills == nil {
		// 读数还没到：给骨架，而不是「未找到技能」。
		read.loading = true
		return nil, read
	}
	q := strings.ToLower(strings.TrimSpace(query))
	seen := make(map[string]bool, len(skills))
	var items []pickItem
	for _, skill := range skills {
		name := strings.TrimSpace(skill.Name)
		if name == "" || !skill.Enabled || seen[name] {
			continue
		}
		seen[name] = true
		item := pickItem{
			kind: pickSkill, name: name, detail: strings.TrimSpace(skill.Description),
			nameStart: -1, nameEnd: -1, detailStart: -1, detailEnd: -1,
			insert: "$" + name + " ",
		}
		if q != "" {
			if spans := searchFindAll(name, q); len(spans) > 0 {
				item.nameStart, item.nameEnd = spans[0][0], spans[0][1]
			} else if item.detail != "" {
				if spans := searchFindAll(item.detail, q); len(spans) > 0 {
					item.detailStart, item.detailEnd = spans[0][0], spans[0][1]
				} else {
					continue
				}
			} else {
				continue
			}
		}
		items = append(items, item)
	}
	return items, read
}

// —— 渲染 ——

// pickPopover 画弹层：贴在输入卡上沿（与斜杠命令弹层同一处），宽度按内容
// 类型取 560 / 384，超过 8 行在弹层内滚动。
// G-A3-01 / G-A4-01 提及弹层（上层底色 + 描边 + 阴影，候选列表）不画时留一个零尺寸占位，输入框不会
// 因为弹层出现而换节点（换节点就会丢焦点）。
func (a *App) pickPopover(c *gui.Context) {
	s := &a.pick
	if !s.open {
		gui.Row(c).Shrink(0).Height(0)
		return
	}
	t := c.Theme()
	width := float32(pickFileWidth)
	if s.kind == pickSkill {
		width = pickSkillWidth
	}
	// 窄窗里跟着主区收窄，不溢出窗口（min(w, 100vw − 2×14)）。
	if windowWidth, _ := c.Size(); windowWidth-2*space4 < width {
		width = windowWidth - 2*space4
		if width < pickMinWidth {
			width = pickMinWidth
		}
	}
	label := pickGroupFile
	if s.kind == pickSkill {
		label = pickGroupSkill
	}
	// G-A3-01 / G-A4-01 提及弹层（上层底色 + 描边 + 阴影）
	gui.Column(c).Shrink(0).Width(width).Padding(pickPad).Gap(pickGap).
		Background(t.SurfaceHover).Border(strokeHairline, t.Border).Radius(radiusCard).
		Clip().Shadow(0, 4, 16, 0, colorShadow.in(t)).
		Label("候选列表：" + label).Role(gui.RoleList).Children(func() {
		switch {
		case a.pickLoading():
			a.pickSkeletons(c)
		case s.fileRead.failed || s.skillRead.failed:
			a.pickFailedRow(c)
		case len(s.items) == 0:
			a.pickEmptyRow(c)
		default:
			gui.Scroll(c).Shrink(0).MaxHeight(pickMaxHeight).Children(func() {
				gui.Column(c).Gap(pickGap).Children(func() {
					index := 0
					if len(s.files) > 0 {
						pickGroupHeader(c, pickGroupFile, len(s.files))
						for _, item := range s.files {
							a.pickRow(c, index, item)
							index++
						}
					}
					if len(s.skills) > 0 {
						pickGroupHeader(c, pickGroupSkill, len(s.skills))
						for _, item := range s.skills {
							a.pickRow(c, index, item)
							index++
						}
					}
					if note := a.pickNote(); note != "" {
						typeMicro.style(t, gui.Text(c, note)).TextColor(t.TextMuted).
							Padding(0, space2).SingleLine()
					}
				})
			})
		}
	})
}

// pickNote 是候选的来源说明：候选到上限、或内容搜索被截断时写一句。
func (a *App) pickNote() string {
	s := &a.pick
	if s.fileRead.note != "" {
		return s.fileRead.note
	}
	return s.skillRead.note
}

// pickGroupHeader 是分组标题（文件 / 技能）：12px 次要色，与候选主字左缘对齐。
// G-A3-04 分组标题（文件在前、技能在后；没有候选的组不画标题）
func pickGroupHeader(c *gui.Context, label string, count int) {
	t := c.Theme()
	gui.Row(c).Shrink(0).Height(rowCompact).Padding(0, space2).AlignItems(gui.Center).
		Label("候选分组：" + label).Children(func() {
		typeSecondary.style(t, gui.Text(c, label)).TextColor(t.TextMuted).SingleLine()
	})
}

// pickRow 是一条候选：图标 + 主字（14 半粗，命中高亮）+ 右尾（12 次要，目录 /
// 描述）；目录主字尾部补 '/'，选中行用悬停面。
// G-A3-05 文件候选（主字加粗、目录补 '/'）/ G-A3-06 次级目录 /
// G-A3-10 / G-A4-04 技能候选（名称 + 描述）
func (a *App) pickRow(c *gui.Context, index int, item pickItem) {
	t := c.Theme()
	selected := index == a.pick.selected
	name := item.name
	if item.dir {
		name += "/"
	}
	word := pickWordFile
	icon := iconDocument
	switch {
	case item.dir:
		word, icon = pickWordDir, iconFolderOpen
	case item.kind == pickSkill:
		word, icon = pickWordSkill, iconSparkles
	}
	label := name
	if item.detail != "" {
		label += "，" + item.detail
	}
	label += "，" + word
	row := gui.Row(c).Shrink(0).Height(pickRowHeight).Padding(0, space2).
		Gap(space2).AlignItems(gui.Center).Radius(radiusSmall).
		Label(label).Role(gui.RoleOption).AccessSelected(selected).
		Cursor(gui.CursorPointer)
	if selected {
		row = row.ScrollIntoView().Background(t.SurfacePressed)
	} else if row.Hovered() {
		row = row.Background(t.SurfacePressed)
	}
	row = row.OnClick(func() { a.insertPick(item) })
	row.Children(func() {
		iconGlyph(c, icon, t.TextMuted, word)
		if item.nameStart >= 0 {
			// G-A3-05 候选行主字（命中片段高亮）
			spans := searchSpans(t, name,
				[]searchRange{{start: item.nameStart, end: item.nameEnd}},
				typeBodyStrong, t.Text)
			gui.RichText(c, spans...).Label(name).SingleLine()
		} else {
			typeBodyStrong.style(t, gui.Text(c, name)).TextColor(t.Text).SingleLine()
		}
		if item.detail != "" {
			detail := item.detail
			if item.detailStart >= 0 {
				// G-A3-06 候选行右尾（所在目录，命中片段高亮）
				spans := searchSpans(t, detail,
					[]searchRange{{start: item.detailStart, end: item.detailEnd}},
					typeMicro, t.TextMuted)
				gui.RichText(c, spans...).Label(detail).MaxWidth(pickDetailMax).
					TextAlign(gui.End).SingleLine()
				return
			}
			// G-A3-10 技能候选右尾（描述）
			typeMicro.style(t, gui.Text(c, detail)).TextColor(t.TextMuted).
				MaxWidth(pickDetailMax).TextAlign(gui.End).SingleLine()
		}
	})
}

// pickSkeletons 是候选未返回时的两条骨架（32 高一行的中性浅块）。
// G-A3-02 / G-A4-02 骨架（两条、与候选行同高、读屏不逐条播报）
func (a *App) pickSkeletons(c *gui.Context) {
	t := c.Theme()
	gui.Column(c).Shrink(0).Gap(pickGap).Label(pickSearchingText).
		Role(gui.RoleStatus).Children(func() {
		for i := 0; i < pickSkeletonRows; i++ {
			gui.Box(c).Shrink(0).Height(pickRowHeight - 2*pickGap).Radius(radiusSmall).
				Background(t.SurfacePressed).Label("正在搜索").AccessHidden(true)
		}
	})
}

// pickFailedRow 是读失败：通用句 + 原因原文 + 重试，不停在骨架里。
// G-A3-03 文件搜索失败（换掉骨架，给原因与重试）
func (a *App) pickFailedRow(c *gui.Context) {
	t := c.Theme()
	reason := a.pick.fileRead.reason
	text := pickFileFailedText
	if a.pick.fileRead.failed {
		text = pickFileFailedText
	} else {
		text = pickSkillFailedText
		reason = a.pick.skillRead.reason
	}
	gui.Row(c).Shrink(0).MinHeight(rowRegular).Padding(space1, space2).Gap(space2).
		AlignItems(gui.Center).Children(func() {
		gui.Column(c).Grow(1).Children(func() {
			typeSecondary.style(t, gui.Text(c, text)).TextColor(t.Danger).
				Role(gui.RoleAlert).SingleLine()
			if reason != "" {
				typeMicro.style(t, gui.Text(c, reason)).TextColor(t.Danger).MaxLines(2)
			}
		})
		// 重试：文件读数跳过缓存重读一次；技能读数没有重试入口时置灰写原因。
		// 按钮的读屏名带组名（屏上仍只有一个「重试」，与输入区别处的同名文案
		// 区分开）。
		canRetry := a.pick.fileRead.failed && a.pick.kind == pickFile && a.wired()
		retryLabel := pickRetryFileText
		if a.pick.skillRead.failed {
			retryLabel = pickRetrySkillText
		}
		b := gui.Button(c, pickRetryText).Label(retryLabel).Disabled(!canRetry)
		if !canRetry {
			b = b.Tooltip(pickRetryNoChannel)
		}
		if b.Clicked() && canRetry {
			a.pick.refresh = true
		}
	})
}

// pickEmptyRow 是这一组的空态（未找到文件或目录 / 当前无工作区目录 / 未找到技能）。
// G-A3-03 / G-A4-03 空态（14px 次要色、左内边距与候选主字对齐）
func (a *App) pickEmptyRow(c *gui.Context) {
	t := c.Theme()
	text := pickNoSkillText
	if a.pick.kind == pickFile {
		text = a.pick.fileRead.empty
		if text == "" {
			text = pickNoFileText
		}
	} else if a.pick.skillRead.empty != "" {
		text = a.pick.skillRead.empty
	}
	gui.Row(c).Shrink(0).Height(pickRowHeight).Padding(0, space2).
		AlignItems(gui.Center).Label(text).Children(func() {
		typeSecondary.style(t, gui.Text(c, text)).TextColor(t.TextMuted).SingleLine()
	})
}
