package workspace

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// GitState 是单个目录条目的 git 跟踪状态。
type GitState string

const (
	// GitNone 表示没有可报告的 git 状态。
	GitNone GitState = ""
	// GitUntracked 表示未跟踪，对应 porcelain 的 ??。
	GitUntracked GitState = "untracked"
	// GitModified 表示已跟踪但有改动，对应 M、T 等。
	GitModified GitState = "modified"
	// GitAdded 表示已暂存的新增。
	GitAdded GitState = "added"
	// GitDeleted 表示已跟踪但被删除。
	GitDeleted GitState = "deleted"
	// GitRenamed 表示重命名或复制。
	GitRenamed GitState = "renamed"
	// GitConflicted 表示合并冲突。
	GitConflicted GitState = "conflicted"
	// GitIgnored 预留：默认不列出被忽略的条目。
	GitIgnored GitState = "ignored"
)

// GitStatus 是工作区的 git 只读概览。
// 只读分支与是否有未提交改动，不含提交、切换、暂存等写操作。
type GitStatus struct {
	// IsRepo 表示工作区是否位于 git 工作树中。
	IsRepo bool
	// Root 是仓库根目录的绝对路径。
	Root string
	// Branch 是当前分支名；分离头指针时为空。
	Branch string
	// Detached 表示当前处于分离头指针状态。
	Detached bool
	// Dirty 表示存在未提交改动（含未跟踪文件）。
	Dirty bool
	// ChangedFiles 是 porcelain 输出的改动条目数。
	ChangedFiles int
}

// GitStatus 读取工作区所在的仓库状态。工作区不在仓库内返回 IsRepo=false 且不报错；
// 本机没有 git 返回 ErrGitUnavailable。
func (w *Workspace) GitStatus(ctx context.Context) (GitStatus, error) {
	if err := ctx.Err(); err != nil {
		return GitStatus{}, err
	}
	if _, err := exec.LookPath("git"); err != nil {
		return GitStatus{}, ErrGitUnavailable
	}
	rootOut, rootStderr, err := runGitContext(ctx, w.root, "rev-parse", "--show-toplevel")
	if err != nil {
		if ctx.Err() != nil {
			return GitStatus{}, ctx.Err()
		}
		if isNotRepoError(rootStderr) {
			return GitStatus{}, nil
		}
		return GitStatus{}, fmt.Errorf("判断 git 仓库失败: %w（%s）", err, strings.TrimSpace(string(rootStderr)))
	}
	status := GitStatus{IsRepo: true, Root: strings.TrimSpace(string(rootOut))}

	out, stderr, err := runGitContext(ctx, w.root, "status", "--porcelain=v1", "--branch", "--untracked-files=normal")
	if err != nil {
		if ctx.Err() != nil {
			return GitStatus{}, ctx.Err()
		}
		return GitStatus{}, fmt.Errorf("读取 git 状态失败: %w（%s）", err, strings.TrimSpace(string(stderr)))
	}
	for i, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSuffix(line, "\r")
		if line == "" {
			continue
		}
		if i == 0 && strings.HasPrefix(line, "##") {
			status.Branch, status.Detached = parseBranchHeader(line)
			continue
		}
		status.Dirty = true
		status.ChangedFiles++
	}
	return status, nil
}

// isNotRepoError 判断 git 的报错是不是"不在仓库内"。
func isNotRepoError(stderr []byte) bool {
	return bytes.Contains(stderr, []byte("not a git repository")) ||
		bytes.Contains(stderr, []byte("not a git work tree"))
}

// parseBranchHeader 解析 porcelain 的分支头行，例如：
//
//	## main...origin/main [ahead 1]
//	## HEAD (no branch)
//	## No commits yet on main
func parseBranchHeader(header string) (branch string, detached bool) {
	s := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(header), "##"))
	switch {
	case s == "":
		return "", false
	case strings.HasPrefix(s, "HEAD (no branch)"):
		return "", true
	}
	if strings.HasPrefix(s, "No commits yet on ") {
		return strings.TrimSpace(strings.TrimPrefix(s, "No commits yet on ")), false
	}
	if strings.HasPrefix(s, "Initial commit on ") {
		return strings.TrimSpace(strings.TrimPrefix(s, "Initial commit on ")), false
	}
	if i := strings.Index(s, "..."); i >= 0 {
		s = s[:i]
	}
	if i := strings.Index(s, " ["); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s), false
}

// gitSnapshot 是一次列目录期间使用的 git 状态快照，不跨调用缓存。
type gitSnapshot struct {
	available bool
	reason    string
	// files 与 dirs 的键是相对工作区根目录的路径。
	files map[string]GitState
	dirs  map[string]GitState
}

// status 返回相对工作区路径的条目状态。
func (s gitSnapshot) status(relKey string, kind EntryKind) GitState {
	if !s.available {
		return GitNone
	}
	if kind == EntryDir {
		return s.dirs[relKey]
	}
	return s.files[relKey]
}

// gitSnapshot 读取当前工作区的条目状态。git 缺失或不在仓库内时 available=false，
// 由调用方把原因写进说明，不让列表本身失败。
func (w *Workspace) gitSnapshot() gitSnapshot {
	snap := gitSnapshot{
		reason: "不在 git 仓库内，未提供跟踪状态",
		files:  make(map[string]GitState),
		dirs:   make(map[string]GitState),
	}
	if _, err := exec.LookPath("git"); err != nil {
		snap.reason = "git 不可用，未提供跟踪状态"
		return snap
	}
	repoRootOut, err := runGit(w.root, "rev-parse", "--show-toplevel")
	if err != nil {
		return snap
	}
	repoRoot := strings.TrimSpace(string(repoRootOut))
	if repoRoot == "" {
		return snap
	}
	if real, evalErr := filepath.EvalSymlinks(repoRoot); evalErr == nil {
		repoRoot = real
	}
	out, err := runGit(w.root, "status", "--porcelain=v1", "-z", "--untracked-files=all")
	if err != nil {
		snap.reason = "读取 git 状态失败，未提供跟踪状态"
		return snap
	}
	for relPath, state := range parsePorcelainZ(out) {
		abs := filepath.Join(repoRoot, filepath.FromSlash(relPath))
		if !within(w.root, abs) {
			continue
		}
		rel, relErr := filepath.Rel(w.root, abs)
		if relErr != nil {
			continue
		}
		key := filepath.Clean(rel)
		snap.files[key] = state
		// 目录状态由子条目聚合，方便在列表里看到"这个目录下有改动"。
		for dir := filepath.Dir(key); dir != "."; dir = filepath.Dir(dir) {
			if existing, ok := snap.dirs[dir]; !ok || gitPriority(state) > gitPriority(existing) {
				snap.dirs[dir] = state
			}
		}
	}
	snap.available = true
	snap.reason = ""
	return snap
}

// parsePorcelainZ 解析 `git status --porcelain=v1 -z` 的输出。
// 重命名与复制的条目后面紧跟一个原路径字段，需要一并跳过。
func parsePorcelainZ(out []byte) map[string]GitState {
	result := make(map[string]GitState)
	fields := bytes.Split(out, []byte{0})
	for i := 0; i < len(fields); i++ {
		field := fields[i]
		if len(field) < 4 {
			continue
		}
		x, y := field[0], field[1]
		path := string(field[3:])
		result[path] = classifyGitState(x, y)
		if x == 'R' || x == 'C' || y == 'R' || y == 'C' {
			i++
		}
	}
	return result
}

// classifyGitState 把 porcelain 的两列状态码归并成一个可展示状态。
func classifyGitState(x, y byte) GitState {
	switch {
	case x == '?' && y == '?':
		return GitUntracked
	case x == '!' && y == '!':
		return GitIgnored
	case x == 'U' || y == 'U' || (x == 'A' && y == 'A') || (x == 'D' && y == 'D'):
		return GitConflicted
	case x == 'R' || y == 'R' || x == 'C' || y == 'C':
		return GitRenamed
	case x == 'A' || y == 'A':
		return GitAdded
	case x == 'D' || y == 'D':
		return GitDeleted
	default:
		return GitModified
	}
}

// gitPriority 用于目录聚合时保留更醒目的状态。
func gitPriority(state GitState) int {
	switch state {
	case GitConflicted:
		return 7
	case GitModified:
		return 6
	case GitAdded:
		return 5
	case GitDeleted:
		return 4
	case GitRenamed:
		return 3
	case GitUntracked:
		return 2
	case GitIgnored:
		return 1
	default:
		return 0
	}
}

// runGit 执行只读 git 命令。GIT_OPTIONAL_LOCKS=0 避免 status 顺带刷新索引。
func runGit(dir string, args ...string) ([]byte, error) {
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	out, err := cmd.Output()
	if err != nil {
		return out, fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return out, nil
}

// runGitContext 是带取消的 git 调用，返回 stdout 与 stderr。
func runGitContext(ctx context.Context, dir string, args ...string) (stdout, stderr []byte, err error) {
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	out, err := cmd.Output()
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return out, exitErr.Stderr, err
		}
		return out, nil, err
	}
	return out, nil, nil
}
