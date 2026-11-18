package uiwire

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/arcships/dimagent-mygo/internal/app"
	"github.com/arcships/dimagent-mygo/internal/projection"
	"github.com/arcships/dimagent-mygo/internal/ui"
	"github.com/arcships/dimagent-mygo/internal/workspace"
)

// 右栏文件页与改动页的读数（ui.PanelFacts 的适配器实现）。
//
// 数据来源只有 D7 的只读能力（internal/workspace）：列目录（含 git 合并状态）、
// 关键词内容搜索、git 只读状态、按 run 的声明级改动。这一层不跑 git 命令、
// 不解析仓库文件、不推断 diff——那属于 D7 的口径，缺的能力由 D7 补，不在这里
// 补一份看起来完整的读数。
//
// 读法：界面每帧用同一份请求读快照，这里只在缓存缺失或过期时派发一次后台读，
// 结果到了触发重绘。读槽有限（panelReadSlots），占不到槽的这一帧不派发、下一帧
// 再试，避免搜索、列目录堆积。
const (
	// panelTreePage 是一页条目的默认大小（与右栏的分页大小一致）。
	panelTreePage = 20
	// panelTreeEntryLimit 是单次列目录的条目上限；超过即截断并说明。
	panelTreeEntryLimit = 2000
	// panelSearchLimit 是搜索命中的默认上限（右栏清单 F-08：200）。
	panelSearchLimit = 200
	// panelTreeTTL / panelChangesTTL 是目录与改动的缓存期：仓库被 agent 改动后
	// 最迟这么久内反映到面板。
	panelTreeTTL    = 2 * time.Second
	panelChangesTTL = 2 * time.Second
	// panelSearchTTL 是搜索结果的缓存期（同一关键词不重复扫工作区）。
	panelSearchTTL = 5 * time.Second
	// panelReadTimeout 是一次读的上限：超时按失败报给视图，不无限挂住读槽。
	panelReadTimeout = 8 * time.Second
	// panelReadSlots 是同时在跑的读数上限。
	panelReadSlots = 2
	// panelCacheKeep 是缓存项的最长保留期（过期项在派发新读时清掉）。
	panelCacheKeep = 5 * time.Minute
)

// 适配器负责实现面板读数；页面接线时只认 ui.PanelFacts。
var _ ui.PanelFacts = (*App)(nil)

// panelCache 是面板读数的缓存：三类请求各一张表，值里带快照与读取时间。
type panelCache struct {
	mu       sync.Mutex
	trees    map[string]*panelTreeEntry
	searches map[string]*panelSearchEntry
	changes  map[string]*panelChangeEntry
	slots    chan struct{}
}

// panelTreeEntry 是一次目录读的缓存项。
type panelTreeEntry struct {
	snapshot ui.FileTreeSnapshot
	readAt   time.Time
	fetching bool
}

// panelSearchEntry 是一次搜索读的缓存项。
type panelSearchEntry struct {
	snapshot ui.FileSearchSnapshot
	readAt   time.Time
	fetching bool
}

// panelChangeEntry 是一次改动读的缓存项。
type panelChangeEntry struct {
	snapshot ui.ChangeSet
	readAt   time.Time
	fetching bool
}

// newPanelCache 建一份空缓存。
func newPanelCache() panelCache {
	return panelCache{
		trees:    make(map[string]*panelTreeEntry),
		searches: make(map[string]*panelSearchEntry),
		changes:  make(map[string]*panelChangeEntry),
		slots:    make(chan struct{}, panelReadSlots),
	}
}

// tryAcquire 不阻塞地占一个读槽；占不到返回 false，调用方下一帧再试。
func (c *panelCache) tryAcquire() bool {
	select {
	case c.slots <- struct{}{}:
		return true
	default:
		return false
	}
}

// release 归还一个读槽。
func (c *panelCache) release() { <-c.slots }

// prune 清掉长时间没被读到的缓存项（调用方持有锁）。
func (c *panelCache) prune(now time.Time) {
	for key, entry := range c.trees {
		if !entry.fetching && now.Sub(entry.readAt) > panelCacheKeep {
			delete(c.trees, key)
		}
	}
	for key, entry := range c.searches {
		if !entry.fetching && now.Sub(entry.readAt) > panelCacheKeep {
			delete(c.searches, key)
		}
	}
	for key, entry := range c.changes {
		if !entry.fetching && now.Sub(entry.readAt) > panelCacheKeep {
			delete(c.changes, key)
		}
	}
}

// ---- 文件树 ----

// FileTree 读一个目录的条目：目录在前、文件在后，按名称字节序。
func (w *App) FileTree(request ui.FileTreeRequest) ui.FileTreeSnapshot {
	request = normalizeFileTree(request)
	key := treeKey(request)

	c := &w.panel
	c.mu.Lock()
	entry, ok := c.trees[key]
	if !ok {
		entry = &panelTreeEntry{snapshot: ui.FileTreeSnapshot{
			Cwd: request.Cwd, Dir: request.Dir, Offset: request.Offset,
			Limit: request.Limit, Phase: ui.PanelReadLoading,
		}}
		c.trees[key] = entry
	}
	snapshot := entry.snapshot
	dispatch := panelStale(entry.readAt, entry.fetching, panelTreeTTL)
	if dispatch && c.tryAcquire() {
		entry.fetching = true
		c.prune(time.Now())
		go w.refreshFileTree(key, request)
	}
	c.mu.Unlock()
	return snapshot
}

// panelStale 判断缓存项是否该重读：没有读过、过了缓存期，且没有在读。
func panelStale(readAt time.Time, fetching bool, ttl time.Duration) bool {
	if fetching {
		return false
	}
	return readAt.IsZero() || time.Since(readAt) >= ttl
}

// treeKey 是目录读的缓存键。
func treeKey(request ui.FileTreeRequest) string {
	return strings.Join([]string{
		request.Cwd, request.Dir,
		fmt.Sprintf("%d", request.Offset), fmt.Sprintf("%d", request.Limit),
	}, "\x00")
}

// normalizeFileTree 补默认页大小并夹住负值。
func normalizeFileTree(request ui.FileTreeRequest) ui.FileTreeRequest {
	if request.Limit <= 0 {
		request.Limit = panelTreePage
	}
	if request.Offset < 0 {
		request.Offset = 0
	}
	request.Dir = strings.Trim(strings.TrimSpace(request.Dir), "/")
	return request
}

// refreshFileTree 在后台读一次目录并写回缓存。
func (w *App) refreshFileTree(key string, request ui.FileTreeRequest) {
	defer w.panel.release()
	ctx, cancel := context.WithTimeout(context.Background(), panelReadTimeout)
	defer cancel()
	snapshot := w.readFileTree(ctx, request)

	w.panel.mu.Lock()
	if entry, ok := w.panel.trees[key]; ok {
		entry.snapshot = snapshot
		entry.readAt = snapshot.ReadAt
		entry.fetching = false
	}
	w.panel.mu.Unlock()
	w.changed()
}

// readFileTree 是真正的一次目录读（D7 的 ListDir + 分页）。
func (w *App) readFileTree(ctx context.Context, request ui.FileTreeRequest) ui.FileTreeSnapshot {
	snapshot := ui.FileTreeSnapshot{
		Cwd: request.Cwd, Dir: request.Dir, Offset: request.Offset,
		Limit: request.Limit, Phase: ui.PanelReadFailed, ReadAt: time.Now(),
	}
	if err := ctx.Err(); err != nil {
		snapshot.Error = err.Error()
		return snapshot
	}
	ws, rel, err := w.panelWorkspace(request.Cwd)
	if err != nil {
		snapshot.Error = err.Error()
		snapshot.Denied = errors.Is(err, fs.ErrPermission)
		return snapshot
	}
	dir := panelDirArg(rel, request.Dir)

	// 分页用条目上限实现：读 offset+limit 条，再切掉前 offset 条。
	want := request.Offset + request.Limit
	capped := false
	if want > panelTreeEntryLimit {
		want = panelTreeEntryLimit
		capped = true
	}
	listing, err := ws.ListDir(dir, workspace.ListOptions{MaxEntries: want})
	if err != nil {
		snapshot.Error = err.Error()
		snapshot.Denied = errors.Is(err, fs.ErrPermission)
		return snapshot
	}
	entries := listing.Entries
	if request.Offset < len(entries) {
		entries = entries[request.Offset:]
	} else {
		entries = nil
	}
	snapshot.Entries = fileEntries(entries)
	snapshot.Total = listing.Total
	snapshot.HasMore = listing.Total > request.Offset+len(snapshot.Entries)
	snapshot.Truncated = capped && listing.Truncated
	snapshot.Note = listing.Note
	snapshot.GitKnown = listing.GitAvailable
	snapshot.Phase = ui.PanelReadReady
	return snapshot
}

// fileEntries 把 D7 的条目映射成快照：目录在前，同类按名称字节序。
func fileEntries(entries []workspace.Entry) []ui.FileEntrySnapshot {
	out := make([]ui.FileEntrySnapshot, 0, len(entries))
	for _, entry := range entries {
		out = append(out, ui.FileEntrySnapshot{
			Name:    entry.Name,
			Dir:     entry.Kind == workspace.EntryDir,
			Symlink: entry.Kind == workspace.EntrySymlink,
			Size:    entry.Size,
			ModTime: entry.ModTime,
			Change:  gitLetter(entry.Git),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Dir != out[j].Dir {
			return out[i].Dir
		}
		return out[i].Name < out[j].Name
	})
	return out
}

// gitLetter 把 D7 的合并 git 状态换成界面用的状态字母。
//
// D7 的 git 只读口径只有合并后的单列状态（不区分暂存列，见
// docs/domains/D7-workspace.md §4.5），所以这里也只给一个字母。
func gitLetter(state workspace.GitState) string {
	switch state {
	case workspace.GitModified:
		return "M"
	case workspace.GitAdded:
		return "A"
	case workspace.GitDeleted:
		return "D"
	case workspace.GitUntracked:
		return "?"
	case workspace.GitRenamed:
		return "R"
	case workspace.GitConflicted:
		return "U"
	}
	return ""
}

// ---- 内容搜索 ----

// FileSearch 读一次工作区内容搜索的结果；关键词为空时快照是空闲态。
func (w *App) FileSearch(request ui.FileSearchRequest) ui.FileSearchSnapshot {
	request.Query = strings.TrimSpace(request.Query)
	if request.Limit <= 0 {
		request.Limit = panelSearchLimit
	}
	if request.Query == "" {
		return ui.FileSearchSnapshot{Query: "", Limit: request.Limit, Phase: ui.PanelReadIdle}
	}
	key := searchKey(request)

	c := &w.panel
	c.mu.Lock()
	entry, ok := c.searches[key]
	if !ok {
		entry = &panelSearchEntry{snapshot: ui.FileSearchSnapshot{
			Query: request.Query, Limit: request.Limit, Phase: ui.PanelReadLoading,
		}}
		c.searches[key] = entry
	}
	snapshot := entry.snapshot
	dispatch := panelStale(entry.readAt, entry.fetching, panelSearchTTL)
	if dispatch && c.tryAcquire() {
		entry.fetching = true
		c.prune(time.Now())
		go w.refreshFileSearch(key, request)
	}
	c.mu.Unlock()
	return snapshot
}

// searchKey 是搜索读的缓存键。
func searchKey(request ui.FileSearchRequest) string {
	return strings.Join([]string{request.Cwd, request.Query, fmt.Sprintf("%d", request.Limit)}, "\x00")
}

// refreshFileSearch 在后台扫一次工作区并写回缓存。
func (w *App) refreshFileSearch(key string, request ui.FileSearchRequest) {
	defer w.panel.release()
	ctx, cancel := context.WithTimeout(context.Background(), panelReadTimeout)
	defer cancel()
	snapshot := w.readFileSearch(ctx, request)

	w.panel.mu.Lock()
	if entry, ok := w.panel.searches[key]; ok {
		entry.snapshot = snapshot
		entry.readAt = snapshot.ReadAt
		entry.fetching = false
	}
	w.panel.mu.Unlock()
	w.changed()
}

// readFileSearch 是真正的一次搜索（D7 的一次性扫描，不建索引）。
func (w *App) readFileSearch(ctx context.Context, request ui.FileSearchRequest) ui.FileSearchSnapshot {
	snapshot := ui.FileSearchSnapshot{
		Query: request.Query, Limit: request.Limit,
		Phase: ui.PanelReadFailed, ReadAt: time.Now(),
	}
	if err := ctx.Err(); err != nil {
		snapshot.Error = err.Error()
		return snapshot
	}
	ws, rel, err := w.panelWorkspace(request.Cwd)
	if err != nil {
		snapshot.Error = err.Error()
		snapshot.Denied = errors.Is(err, fs.ErrPermission)
		return snapshot
	}
	result, err := ws.Search(rel, workspace.SearchOptions{
		Query: request.Query, MaxMatches: request.Limit,
	})
	if err != nil {
		snapshot.Error = err.Error()
		snapshot.Denied = errors.Is(err, fs.ErrPermission)
		return snapshot
	}
	snapshot.Hits = fileHits(result.Matches)
	snapshot.Truncated = result.Truncated
	snapshot.StopReason = result.StopReason
	snapshot.FilesScanned = result.FilesScanned
	snapshot.BytesScanned = result.BytesScanned
	snapshot.Skipped = result.Skipped
	snapshot.Phase = ui.PanelReadReady
	return snapshot
}

// fileHits 把 D7 的命中映射成快照（路径保持相对工作区根）。
func fileHits(matches []workspace.SearchMatch) []ui.FileHitSnapshot {
	out := make([]ui.FileHitSnapshot, 0, len(matches))
	for _, match := range matches {
		out = append(out, ui.FileHitSnapshot{
			Path: match.Path, Line: match.Line, Column: match.Column, Snippet: match.Snippet,
		})
	}
	return out
}

// ---- 改动 ----

// Changes 读一次改动清单：git 只读概览，外加 Last Run 范围的路径清单。
func (w *App) Changes(request ui.ChangesRequest) ui.ChangeSet {
	if request.Scope == "" {
		request.Scope = ui.ChangeScopeUncommitted
	}
	key := changesKey(request)

	c := &w.panel
	c.mu.Lock()
	entry, ok := c.changes[key]
	if !ok {
		entry = &panelChangeEntry{snapshot: ui.ChangeSet{
			Cwd: request.Cwd, Scope: request.Scope, RunID: request.RunID,
			Phase: ui.PanelReadLoading,
		}}
		c.changes[key] = entry
	}
	snapshot := entry.snapshot
	dispatch := panelStale(entry.readAt, entry.fetching, panelChangesTTL)
	if dispatch && c.tryAcquire() {
		entry.fetching = true
		c.prune(time.Now())
		go w.refreshChanges(key, request)
	}
	c.mu.Unlock()
	return snapshot
}

// changesKey 是改动读的缓存键。
func changesKey(request ui.ChangesRequest) string {
	return strings.Join([]string{request.Cwd, request.Scope, request.RunID}, "\x00")
}

// refreshChanges 在后台读一次改动并写回缓存。
func (w *App) refreshChanges(key string, request ui.ChangesRequest) {
	defer w.panel.release()
	ctx, cancel := context.WithTimeout(context.Background(), panelReadTimeout)
	defer cancel()
	snapshot := w.readChanges(ctx, request)

	w.panel.mu.Lock()
	if entry, ok := w.panel.changes[key]; ok {
		entry.snapshot = snapshot
		entry.readAt = snapshot.ReadAt
		entry.fetching = false
	}
	w.panel.mu.Unlock()
	w.changed()
}

// readChanges 按范围读改动。
//
// 未提交与当前分支两个范围读 D7 的 git 只读概览；条目级事实（按暂存态分组的
// 文件行、± 行数、diff 文本）D7 不提供，所以不画行、不算行数，只把原因写进
// Unavailable——视图据此画「改动不可用」，不把"读不到"画成"没有改动"。
func (w *App) readChanges(ctx context.Context, request ui.ChangesRequest) ui.ChangeSet {
	changes := ui.ChangeSet{
		Cwd: request.Cwd, Scope: request.Scope, RunID: request.RunID,
		Phase: ui.PanelReadFailed, ReadAt: time.Now(),
	}
	if err := ctx.Err(); err != nil {
		changes.Error = err.Error()
		return changes
	}
	if request.Scope == ui.ChangeScopeLastRun {
		return w.readLastRunChanges(ctx, changes)
	}

	ws, _, err := w.panelWorkspace(request.Cwd)
	if err != nil {
		changes.Error = err.Error()
		changes.Restricted = errors.Is(err, fs.ErrPermission)
		return changes
	}
	status, err := ws.GitStatus(ctx)
	if err != nil {
		if errors.Is(err, workspace.ErrGitUnavailable) {
			changes.Phase = ui.PanelReadReady
			changes.Unavailable = "本机没有可用的 git，改动不可读。"
			return changes
		}
		changes.Error = err.Error()
		return changes
	}
	changes.Repo = status.IsRepo
	changes.Branch = status.Branch
	changes.Detached = status.Detached
	changes.Dirty = status.Dirty
	changes.ChangedFileCount = status.ChangedFiles
	changes.Phase = ui.PanelReadReady

	if !status.IsRepo {
		changes.Unavailable = "工作区不在 git 工作树里，没有改动可读。"
		return changes
	}
	switch request.Scope {
	case ui.ChangeScopeBranch:
		changes.Unavailable = "没有分支与基线差异的读取：D7 的 git 只读口径不含 diff 与分支对比。"
	case ui.ChangeScopeUncommitted:
		if !status.Dirty {
			// 干净工作区是读到的空：不写不可用原因，视图画「没有未提交的改动」。
			return changes
		}
		changes.Unavailable = "git 只读口径只给改动计数，没有按文件与暂存态的条目、± 行数与 diff 文本。"
	}
	return changes
}

// readLastRunChanges 读最近一次 run 的声明级改动（D7 的 coverage=tool_reports）。
func (w *App) readLastRunChanges(ctx context.Context, changes ui.ChangeSet) ui.ChangeSet {
	sessionID := w.currentSession()
	if sessionID == "" {
		changes.Phase = ui.PanelReadReady
		changes.Unavailable = "没有打开的会话，读不到最近一次 run 的改动。"
		return changes
	}
	view, ok := w.A.View(sessionID)
	if !ok {
		changes.Phase = ui.PanelReadReady
		changes.Unavailable = "会话还没打开，读不到最近一次 run 的改动。"
		return changes
	}
	runID := changes.RunID
	if runID == "" {
		runID = lastEndedTurn(view)
	}
	if runID == "" {
		changes.Phase = ui.PanelReadReady
		changes.Unavailable = "当前分支还没有收口的轮次。"
		return changes
	}
	result, err := w.A.RunChanges(ctx, sessionID, runID)
	if err != nil {
		if errors.Is(err, workspace.ErrGitUnavailable) {
			changes.Phase = ui.PanelReadReady
			changes.Unavailable = err.Error()
			return changes
		}
		changes.Error = err.Error()
		return changes
	}
	changes.RunID = runID
	changes.Files = runChangeFiles(result.Paths)
	changes.FilesTruncated = result.Truncated
	changes.Note = lastRunNote(result)
	changes.Phase = ui.PanelReadReady
	return changes
}

// lastEndedTurn 取已收口的最后一轮标识。
func lastEndedTurn(view app.SessionView) string {
	id := ""
	index := -1
	for _, turn := range view.Turns {
		if turn.Status != projection.TurnEnded || turn.Index < index {
			continue
		}
		if strings.TrimSpace(turn.ID) == "" {
			continue
		}
		id, index = turn.ID, turn.Index
	}
	return id
}

// runChangeFiles 把声明级改动映射成快照：声明删除（磁盘上已不存在）标 D。
func runChangeFiles(paths []workspace.RunChange) []ui.ChangeFileSnapshot {
	out := make([]ui.ChangeFileSnapshot, 0, len(paths))
	for _, change := range paths {
		file := ui.ChangeFileSnapshot{Path: change.Path, Exists: change.Exists}
		if !change.Exists {
			// 与时间线改动行的「已删除」同一判定：声明了但磁盘上没有。
			file.Status = "D"
		}
		out = append(out, file)
	}
	return out
}

// lastRunNote 说明这次改动读取的覆盖级别与截断。
func lastRunNote(result workspace.RunChanges) string {
	notes := []string{
		fmt.Sprintf("覆盖级别 %s：只含工具结果声明的路径，不含 exec、外部修改与未声明的写入",
			result.Coverage),
	}
	if result.Note != "" {
		notes = append(notes, result.Note)
	}
	return strings.Join(notes, "；")
}

// ---- 共用 ----

// panelWorkspace 取工作区并算出一个目录相对工作区根的位置。
//
// cwd 是会话事实（目录里的 cwd）；空表示按工作区根读。会话目录落在工作区根
// 之外时明确报错，不悄悄换成根目录——那样画出来的树不是用户以为的那一份。
func (w *App) panelWorkspace(cwd string) (*workspace.Workspace, string, error) {
	ws := w.A.Workspace()
	if ws == nil {
		return nil, "", errors.New("工作区未配置：没有可读的工作目录")
	}
	if strings.TrimSpace(cwd) == "" {
		return ws, ".", nil
	}
	rel, err := ws.Relative(cwd)
	if err != nil {
		return nil, "", fmt.Errorf("会话工作目录不在工作区内: %w", err)
	}
	return ws, rel, nil
}

// panelDirArg 拼出要读的目录：相对工作区根的路径 + 视图展开的目录。
func panelDirArg(rel, dir string) string {
	if dir == "" || dir == "." {
		return rel
	}
	if rel == "" || rel == "." {
		return filepath.FromSlash(dir)
	}
	return filepath.Join(rel, filepath.FromSlash(dir))
}
