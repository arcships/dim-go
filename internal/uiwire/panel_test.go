package uiwire

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/fakeappserver"
	"github.com/arcships/dimagent-mygo/internal/ui"
)

// 右栏两页的读数：文件树（目录 + git 合并状态 + 分页）、内容搜索（命中 + 上限）、
// 改动（git 只读概览 + Last Run 路径清单），以及读不到时的错误态。
//
// 用例用真实临时工作区与真实临时仓库（不 mock 文件系统、不 mock git），假服务端
// 只负责目录里的会话 cwd 与历史事实；没有 git 的机器上相关用例 t.Skip，不把跳过
// 当通过。

// requireGit 在没有 git 的机器上跳过（D7 的 git 读依赖本机 git）。
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("本机没有 git：跳过工作区 git 读数用例")
	}
}

// runPanelGit 在测试仓库里执行一条 git 命令。
func runPanelGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0", "GIT_CONFIG_NOSYSTEM=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
}

// writePanelFile 在工作区里写一个文件（自动建目录）。
func writePanelFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("建目录 %s: %v", name, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("写文件 %s: %v", name, err)
	}
}

// newPanelRepo 造一个真实仓库：一个已暂存的新增、一个未暂存的修改、一个未暂存
// 的删除、一个未跟踪文件，外加一处可搜索的内容。
func newPanelRepo(t *testing.T) string {
	t.Helper()
	requireGit(t)
	dir := t.TempDir()
	writePanelFile(t, dir, "notes.md", "panel notes\nPANEL_SEARCH_TOKEN\n")
	writePanelFile(t, dir, "main.go", "package main\n\nfunc main() {}\n")
	writePanelFile(t, dir, "docs/guide.md", "guide\n")
	writePanelFile(t, dir, "docs/old.md", "old\n")
	runPanelGit(t, dir, "init", "-b", "main")
	runPanelGit(t, dir, "config", "user.email", "panel@example.com")
	runPanelGit(t, dir, "config", "user.name", "panel")
	runPanelGit(t, dir, "add", ".")
	runPanelGit(t, dir, "commit", "-m", "initial")

	// 已暂存的新增。
	writePanelFile(t, dir, "staged.go", "package main\n")
	runPanelGit(t, dir, "add", "staged.go")
	// 未暂存的修改。
	writePanelFile(t, dir, "notes.md", "panel notes\nPANEL_SEARCH_TOKEN\n更多一行\n")
	// 未暂存的删除：删掉的文件不在目录列表里，只能从所在目录的聚合状态看出 D。
	if err := os.Remove(filepath.Join(dir, "docs", "old.md")); err != nil {
		t.Fatalf("删除 docs/old.md: %v", err)
	}
	// 未跟踪。
	writePanelFile(t, dir, "scratch.txt", "scratch\n")
	return dir
}

// panelEvents 是一轮收口的历史：两次写入声明，其中一次声明的文件不在磁盘上。
func panelEvents() []protocol.RawEvent {
	return []protocol.RawEvent{
		e2eRawEvent("p1", 1, "input", "message",
			`{"mode":"queue","user":{"content":[{"type":"text","text":"改两个文件"}]}}`),
		e2eRawEvent("p2", 2, "turn", "start", `{"turn_id":"p-t1"}`),
		e2eRawEvent("p3", 3, "step", "tool-use",
			`{"turn_id":"p-t1","request_id":"r1","call":{"id":"c-write","name":"write","input":{"path":"internal/ui/source.go"}}}`),
		e2eRawEvent("p4", 4, "step", "tool-completed",
			`{"turn_id":"p-t1","request_id":"r1","call_id":"c-write","result":{"structuredContent":{"path":"internal/ui/source.go"}}}`),
		e2eRawEvent("p5", 5, "step", "tool-use",
			`{"turn_id":"p-t1","request_id":"r1","call":{"id":"c-gone","name":"write","input":{"path":"gone/file.go"}}}`),
		e2eRawEvent("p6", 6, "step", "tool-completed",
			`{"turn_id":"p-t1","request_id":"r1","call_id":"c-gone","result":{"structuredContent":{"path":"gone/file.go"}}}`),
		e2eRawEvent("p7", 7, "turn", "end", `{"turn_id":"p-t1","status":"completed"}`),
	}
}

// panelScript 是面板用例的服务端脚本：目录里的会话 cwd 指向真实工作区。
func panelScript(cwd string) fakeappserver.Script {
	return fakeappserver.Script{
		Features: []string{"session.history.latestTurn"},
		Sessions: []fakeappserver.Session{{
			ID:              e2eSession,
			LatestTurnStart: 1,
			Events:          panelEvents(),
		}},
		Replies: []fakeappserver.Reply{
			{Method: "session.catalog", Result: map[string]any{"sessions": []any{map[string]any{
				"sessionId": e2eSession, "cwd": cwd, "createdAt": 1, "updatedAt": 2,
				"running": false, "title": "面板读数会话",
			}}, "nextCursor": nil}},
			{Method: "session.permission.get", Result: map[string]any{
				"version": 1, "policy": map[string]any{"preset": "full-access"},
			}},
		},
	}
}

// panelEntry 在快照里按名字找条目。
func panelEntry(snapshot ui.FileTreeSnapshot, name string) (ui.FileEntrySnapshot, bool) {
	for _, entry := range snapshot.Entries {
		if entry.Name == name {
			return entry, true
		}
	}
	return ui.FileEntrySnapshot{}, false
}

// TestPanelFileTreeReadsWorkspace 断言文件树来自真实目录与真实 git 状态。
func TestPanelFileTreeReadsWorkspace(t *testing.T) {
	root := newPanelRepo(t)
	// 历史声明里用到的工作区文件（Last Run 用例共用）。
	writePanelFile(t, root, "internal/ui/source.go", "package ui\n")
	server := startE2EServer(t, panelScript(root))
	w := New(newRowsApp(t, server, root))

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := w.Connect(ctx); err != nil {
		t.Fatalf("连接失败：%v", err)
	}

	request := ui.FileTreeRequest{Cwd: root, Limit: 200}
	waitFor(t, "目录读回", func() bool {
		return w.FileTree(request).Phase == ui.PanelReadReady
	})
	snapshot := w.FileTree(request)
	if snapshot.Error != "" {
		t.Fatalf("目录读取报错：%s", snapshot.Error)
	}
	if !snapshot.GitKnown {
		t.Fatal("这次读没有拿到 git 状态")
	}
	// 隐藏条目也在：D7 的列目录含隐藏条目。
	if _, ok := panelEntry(snapshot, ".git"); !ok {
		t.Fatal("目录里没有 .git（隐藏条目应当列出）")
	}
	// 目录在前：所有目录都排在文件之前。
	seenFile := false
	for _, entry := range snapshot.Entries {
		if !entry.Dir {
			seenFile = true
			continue
		}
		if seenFile {
			t.Fatalf("目录排在文件之后：%+v", snapshot.Entries)
		}
	}
	cases := map[string]string{
		"notes.md":    "M", // 未暂存的修改
		"staged.go":   "A", // 已暂存的新增
		"scratch.txt": "?", // 未跟踪
		"main.go":     "",  // 干净文件没有状态字母
	}
	for name, want := range cases {
		entry, ok := panelEntry(snapshot, name)
		if !ok {
			t.Fatalf("目录里缺少 %s（条目：%+v）", name, snapshot.Entries)
		}
		if entry.Change != want {
			t.Fatalf("%s 的 git 状态字母 = %q，想要 %q", name, entry.Change, want)
		}
	}
	// 目录的字母由后代聚合：docs 里删掉了一个已提交的文件，所以 docs 带 D。
	if docs, ok := panelEntry(snapshot, "docs"); !ok || docs.Change != "D" {
		t.Fatalf("docs = %+v（想要聚合出 D）", docs)
	}
	// 删掉的文件不在目录列表里：列目录列的是磁盘上还在的条目。
	// 改动页要显示它，需要条目级改动读数（D7 目前没有，见交回说明）。
	if _, ok := panelEntry(snapshot, "old.md"); ok {
		t.Fatal("已删除的文件不该出现在目录列表里")
	}

	// 同一请求第二次读的是同一份快照（界面每帧读不该每帧扫目录）。
	again := w.FileTree(request)
	if !again.ReadAt.Equal(snapshot.ReadAt) {
		t.Fatalf("同一请求的两次读不是同一份快照：%v / %v", again.ReadAt, snapshot.ReadAt)
	}
}

// TestPanelFileTreePages 断言分页：目录在前，按 Offset/Limit 切片，剩余可见。
func TestPanelFileTreePages(t *testing.T) {
	root := newPanelRepo(t)
	server := startE2EServer(t, panelScript(root))
	w := New(newRowsApp(t, server, root))
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := w.Connect(ctx); err != nil {
		t.Fatalf("连接失败：%v", err)
	}

	first := ui.FileTreeRequest{Cwd: root, Offset: 0, Limit: 2}
	waitFor(t, "第一页读回", func() bool {
		return w.FileTree(first).Phase == ui.PanelReadReady
	})
	page1 := w.FileTree(first)
	if len(page1.Entries) != 2 {
		t.Fatalf("第一页 %d 条，想要 2：%+v", len(page1.Entries), page1.Entries)
	}
	for _, entry := range page1.Entries {
		if !entry.Dir {
			t.Fatalf("第一页里出现了文件：%+v", entry)
		}
	}
	if !page1.HasMore {
		t.Fatalf("第一页之后还有条目，HasMore 应当是 true（Total=%d）", page1.Total)
	}

	second := ui.FileTreeRequest{Cwd: root, Offset: 2, Limit: 2}
	waitFor(t, "第二页读回", func() bool {
		return w.FileTree(second).Phase == ui.PanelReadReady
	})
	page2 := w.FileTree(second)
	if len(page2.Entries) != 2 {
		t.Fatalf("第二页 %d 条，想要 2", len(page2.Entries))
	}
	firstNames := map[string]bool{}
	for _, entry := range page1.Entries {
		firstNames[entry.Name] = true
	}
	for _, entry := range page2.Entries {
		if firstNames[entry.Name] {
			t.Fatalf("第二页与第一页重复：%q", entry.Name)
		}
	}
	if page2.Total != page1.Total {
		t.Fatalf("两页的 Total 不一致：%d / %d", page1.Total, page2.Total)
	}
}

// TestPanelFileSearchReadsWorkspace 断言搜索命中来自真实内容扫描，且有上限标记。
func TestPanelFileSearchReadsWorkspace(t *testing.T) {
	root := newPanelRepo(t)
	server := startE2EServer(t, panelScript(root))
	w := New(newRowsApp(t, server, root))
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := w.Connect(ctx); err != nil {
		t.Fatalf("连接失败：%v", err)
	}

	request := ui.FileSearchRequest{Cwd: root, Query: "PANEL_SEARCH_TOKEN", Limit: 50}
	waitFor(t, "搜索读回", func() bool {
		return w.FileSearch(request).Phase == ui.PanelReadReady
	})
	snapshot := w.FileSearch(request)
	if snapshot.Error != "" {
		t.Fatalf("搜索报错：%s", snapshot.Error)
	}
	if len(snapshot.Hits) != 1 {
		t.Fatalf("命中 %d 处，想要 1：%+v", len(snapshot.Hits), snapshot.Hits)
	}
	hit := snapshot.Hits[0]
	if hit.Path != "notes.md" || hit.Line != 2 {
		t.Fatalf("命中 = %+v，想要 notes.md 第 2 行", hit)
	}
	if !strings.Contains(hit.Snippet, "PANEL_SEARCH_TOKEN") {
		t.Fatalf("命中片段 = %q", hit.Snippet)
	}
	if snapshot.Truncated || snapshot.StopReason != "" {
		t.Fatalf("完整扫描不该标截断：%+v", snapshot)
	}

	// 上限 1：命中被截断并给出停止原因。
	limited := ui.FileSearchRequest{Cwd: root, Query: "package", Limit: 1}
	waitFor(t, "限量搜索读回", func() bool {
		return w.FileSearch(limited).Phase == ui.PanelReadReady
	})
	capped := w.FileSearch(limited)
	if len(capped.Hits) != 1 || !capped.Truncated || capped.StopReason != "max-matches" {
		t.Fatalf("限量搜索 = %+v，想要 1 条 + 截断 + max-matches", capped)
	}

	// 无命中是读到的空，不是错误。
	empty := ui.FileSearchRequest{Cwd: root, Query: "ZZZ_NOT_IN_ANY_FILE", Limit: 50}
	waitFor(t, "无命中搜索读回", func() bool {
		return w.FileSearch(empty).Phase == ui.PanelReadReady
	})
	if got := w.FileSearch(empty); len(got.Hits) != 0 || got.Error != "" {
		t.Fatalf("无命中 = %+v，想要空且无错", got)
	}

	// 关键词为空是空闲态：不发搜索。
	if got := w.FileSearch(ui.FileSearchRequest{Cwd: root}); got.Phase != ui.PanelReadIdle {
		t.Fatalf("空关键词的相位 = %q，想要 %q", got.Phase, ui.PanelReadIdle)
	}
}

// TestPanelChangesReadsGitSummary 断言改动页拿到的 git 概览是真实读数，
// 且没有把「D7 没有条目级事实」画成「没有改动」。
func TestPanelChangesReadsGitSummary(t *testing.T) {
	root := newPanelRepo(t)
	server := startE2EServer(t, panelScript(root))
	w := New(newRowsApp(t, server, root))
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := w.Connect(ctx); err != nil {
		t.Fatalf("连接失败：%v", err)
	}

	uncommitted := ui.ChangesRequest{Cwd: root, Scope: ui.ChangeScopeUncommitted}
	waitFor(t, "改动读回", func() bool {
		return w.Changes(uncommitted).Phase == ui.PanelReadReady
	})
	changes := w.Changes(uncommitted)
	if changes.Error != "" {
		t.Fatalf("改动读取报错：%s", changes.Error)
	}
	if !changes.Repo || changes.Branch != "main" || changes.Detached {
		t.Fatalf("仓库读数 = %+v，想要 main 分支的普通仓库", changes)
	}
	if !changes.Dirty {
		t.Fatal("工作区有改动，Dirty 应当是 true")
	}
	// 四条：已暂存的新增、未暂存的修改、未暂存的删除、未跟踪各一。
	if changes.ChangedFileCount != 4 {
		t.Fatalf("改动条目数 = %d，想要 4", changes.ChangedFileCount)
	}
	if changes.Unavailable == "" {
		t.Fatal("没有条目级事实时必须给出原因，不能画成「没有改动」")
	}
	if len(changes.Files) != 0 {
		t.Fatalf("未提交范围不该有路径清单：%+v", changes.Files)
	}

	// 当前分支范围：没有基线差异读取，原因要说清楚，不是空列表。
	branch := ui.ChangesRequest{Cwd: root, Scope: ui.ChangeScopeBranch}
	waitFor(t, "分支范围读回", func() bool {
		return w.Changes(branch).Phase == ui.PanelReadReady
	})
	if got := w.Changes(branch); got.Repo != true || !strings.Contains(got.Unavailable, "diff") {
		t.Fatalf("分支范围 = %+v，想要仓库读数 + 说明缺 diff 读取", got)
	}
}

// TestPanelChangesCleanAndNoRepo 断言干净工作区是「读到的空」，不是「不可用」；
// 不在仓库里的目录给原因。两例都让工作区根就是仓库根（git 读在根上做）。
func TestPanelChangesCleanAndNoRepo(t *testing.T) {
	requireGit(t)
	// 干净仓库。
	clean := t.TempDir()
	writePanelFile(t, clean, "a.txt", "a\n")
	runPanelGit(t, clean, "init", "-b", "main")
	runPanelGit(t, clean, "config", "user.email", "panel@example.com")
	runPanelGit(t, clean, "config", "user.name", "panel")
	runPanelGit(t, clean, "add", ".")
	runPanelGit(t, clean, "commit", "-m", "initial")
	cleanWire := New(newRowsApp(t, startE2EServer(t, panelScript(clean)), clean))
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := cleanWire.Connect(ctx); err != nil {
		t.Fatalf("连接失败：%v", err)
	}
	request := ui.ChangesRequest{Cwd: clean, Scope: ui.ChangeScopeUncommitted}
	waitFor(t, "干净工作区读回", func() bool {
		return cleanWire.Changes(request).Phase == ui.PanelReadReady
	})
	if got := cleanWire.Changes(request); !got.Repo || got.Dirty || got.Unavailable != "" {
		t.Fatalf("干净工作区 = %+v，想要 Repo=true、Dirty=false 且没有不可用原因", got)
	}

	// 不是仓库：读到了「不在仓库里」，也给原因。
	plain := t.TempDir()
	writePanelFile(t, plain, "b.txt", "b\n")
	plainWire := New(newRowsApp(t, startE2EServer(t, panelScript(plain)), plain))
	if err := plainWire.Connect(ctx); err != nil {
		t.Fatalf("连接失败：%v", err)
	}
	notRepo := ui.ChangesRequest{Cwd: plain, Scope: ui.ChangeScopeUncommitted}
	waitFor(t, "非仓库读回", func() bool {
		return plainWire.Changes(notRepo).Phase == ui.PanelReadReady
	})
	if got := plainWire.Changes(notRepo); got.Repo || got.Unavailable == "" {
		t.Fatalf("非仓库目录 = %+v，想要 Repo=false 且有原因", got)
	}
}

// TestPanelChangesLastRunPaths 断言 Last Run 范围读到这一轮的声明级路径清单：
// 存在的文件没有状态字母，声明了但磁盘上没有的标 D。
func TestPanelChangesLastRunPaths(t *testing.T) {
	root := newPanelRepo(t)
	writePanelFile(t, root, "internal/ui/source.go", "package ui\n")
	server := startE2EServer(t, panelScript(root))
	w := New(newRowsApp(t, server, root))
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := w.Connect(ctx); err != nil {
		t.Fatalf("连接失败：%v", err)
	}

	request := ui.ChangesRequest{Cwd: root, Scope: ui.ChangeScopeLastRun}
	waitFor(t, "Last Run 读回", func() bool {
		return w.Changes(request).Phase == ui.PanelReadReady
	})
	got := w.Changes(request)
	if got.Error != "" {
		t.Fatalf("Last Run 读取报错：%s", got.Error)
	}
	if got.Unavailable != "" {
		t.Fatalf("有收口轮次时不该报不可用：%s", got.Unavailable)
	}
	if got.RunID != "p-t1" {
		t.Fatalf("轮次标识 = %q，想要 p-t1（会话已收口的最后一轮）", got.RunID)
	}
	if len(got.Files) != 2 {
		t.Fatalf("路径清单 = %+v，想要 2 条", got.Files)
	}
	byPath := map[string]ui.ChangeFileSnapshot{}
	for _, file := range got.Files {
		byPath[file.Path] = file
	}
	if existing, ok := byPath["internal/ui/source.go"]; !ok || !existing.Exists || existing.Status != "" {
		t.Fatalf("存在的声明 = %+v，想要 Exists=true 且没有状态字母", existing)
	}
	if gone, ok := byPath["gone/file.go"]; !ok || gone.Exists || gone.Status != "D" {
		t.Fatalf("已删除的声明 = %+v，想要 Exists=false 且状态 D", gone)
	}
	if !strings.Contains(got.Note, "tool_reports") {
		t.Fatalf("说明里没有覆盖级别：%q", got.Note)
	}
}

// TestPanelFactsSurfaceErrors 断言读不到时是错误态而不是空白：没有工作区、
// 会话目录越界、目录没有读权限，三种都给出原因（权限不足另带 Denied）。
func TestPanelFactsSurfaceErrors(t *testing.T) {
	// 不配工作区根：读无处落脚，必须报原因。
	bare := New(newRowsApp(t, startE2EServer(t, panelScript("")), ""))
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := bare.Connect(ctx); err != nil {
		t.Fatalf("连接失败：%v", err)
	}
	// 没有工作区根：读无处落脚，异步读回来的必须是带原因的错误态。
	request := ui.FileTreeRequest{Cwd: ""}
	waitFor(t, "没有工作区时读回错误态", func() bool {
		return bare.FileTree(request).Phase == ui.PanelReadFailed
	})
	tree := bare.FileTree(request)
	if tree.Error == "" || tree.Phase != ui.PanelReadFailed {
		t.Fatalf("没有工作区时的文件树 = %+v，想要带原因的错误态", tree)
	}
	if !strings.Contains(tree.Error, "工作区") {
		t.Fatalf("错误原因 = %q，想要说明工作区未配置", tree.Error)
	}
	waitFor(t, "没有工作区时改动读回错误态", func() bool {
		return bare.Changes(ui.ChangesRequest{}).Phase == ui.PanelReadFailed
	})
	if changes := bare.Changes(ui.ChangesRequest{}); changes.Error == "" {
		t.Fatalf("没有工作区时的改动 = %+v，想要带原因的错误态", changes)
	}

	// 会话目录在工作区之外：明确报错，不悄悄换成工作区根。
	root := t.TempDir()
	outside := t.TempDir()
	w := New(newRowsApp(t, startE2EServer(t, panelScript("")), root))
	if err := w.Connect(ctx); err != nil {
		t.Fatalf("连接失败：%v", err)
	}
	outsideRequest := ui.FileTreeRequest{Cwd: outside}
	waitFor(t, "越界读回错误", func() bool {
		return w.FileTree(outsideRequest).Phase == ui.PanelReadFailed
	})
	if got := w.FileTree(outsideRequest); got.Error == "" {
		t.Fatalf("越界目录 = %+v，想要错误原因", got)
	}

	// 没有读权限的子目录：Denied 置位，原因里带权限字样。
	if os.Geteuid() == 0 {
		t.Skip("以 root 运行时权限位不生效，跳过 EACCES 用例")
	}
	locked := filepath.Join(root, "locked")
	if err := os.MkdirAll(locked, 0o755); err != nil {
		t.Fatalf("建目录: %v", err)
	}
	if err := os.Chmod(locked, 0o000); err != nil {
		t.Fatalf("去掉读权限: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	denied := ui.FileTreeRequest{Cwd: root, Dir: "locked"}
	waitFor(t, "无权限目录读回", func() bool {
		return w.FileTree(denied).Phase == ui.PanelReadFailed
	})
	got := w.FileTree(denied)
	if !got.Denied {
		t.Fatalf("无权限目录 = %+v，想要 Denied=true", got)
	}
	if got.Error == "" {
		t.Fatal("无权限目录没有给原因")
	}
}

// TestPanelFactsAfterDisconnect 断言断线后连接阶段带原因；本机读不依赖连接，
// 所以文件树仍是真实内容（不是空白，也不是样例）。
func TestPanelFactsAfterDisconnect(t *testing.T) {
	root := newPanelRepo(t)
	server := startE2EServer(t, panelScript(root))
	w := New(newRowsApp(t, server, root))
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := w.Connect(ctx); err != nil {
		t.Fatalf("连接失败：%v", err)
	}
	request := ui.FileTreeRequest{Cwd: root, Limit: 50}
	waitFor(t, "目录读回", func() bool {
		return w.FileTree(request).Phase == ui.PanelReadReady
	})

	if server.DisconnectAll() == 0 {
		t.Fatal("没有断开任何连接")
	}
	waitFor(t, "连接阶段转为 disconnected", func() bool {
		return w.Connection().Phase == "disconnected"
	})
	if conn := w.Connection(); conn.Reason == "" {
		t.Fatal("断线没有给出原因")
	}
	// 工作区读不经过连接：断线后仍读到真实目录，页面据此既能画内容也能画断线横幅。
	after := w.FileTree(request)
	if after.Phase != ui.PanelReadReady || len(after.Entries) == 0 {
		t.Fatalf("断线后的目录 = %+v，想要仍然读到真实条目", after)
	}
	if _, ok := panelEntry(after, "notes.md"); !ok {
		t.Fatalf("断线后的目录里没有 notes.md：%+v", after.Entries)
	}
}
