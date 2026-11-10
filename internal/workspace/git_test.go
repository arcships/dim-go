package workspace

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestGitStatusNotRepo(t *testing.T) {
	requireGit(t)
	ws, _ := newTestWorkspace(t)
	status, err := ws.GitStatus(context.Background())
	if err != nil {
		t.Fatalf("非仓库目录不应报错: %v", err)
	}
	if status.IsRepo {
		t.Fatal("临时目录不应该是 git 仓库")
	}
}

func TestGitStatusCleanAndDirty(t *testing.T) {
	requireGit(t)
	root := initRepo(t)
	writeFile(t, filepath.Join(root, "a.txt"), []byte("one"))
	runGitT(t, root, "add", "a.txt")
	runGitT(t, root, "commit", "-q", "-m", "init")
	ws, err := NewWorkspace(root)
	if err != nil {
		t.Fatalf("创建工作区: %v", err)
	}

	clean, err := ws.GitStatus(context.Background())
	if err != nil {
		t.Fatalf("读取 git 状态失败: %v", err)
	}
	if !clean.IsRepo || clean.Branch != "main" || clean.Detached || clean.Dirty {
		t.Fatalf("干净仓库状态 = %+v", clean)
	}
	if clean.Root == "" {
		t.Fatal("仓库根不应为空")
	}

	writeFile(t, filepath.Join(root, "a.txt"), []byte("two"))
	writeFile(t, filepath.Join(root, "b.txt"), []byte("new"))
	dirty, err := ws.GitStatus(context.Background())
	if err != nil {
		t.Fatalf("读取 git 状态失败: %v", err)
	}
	if !dirty.Dirty || dirty.ChangedFiles != 2 {
		t.Fatalf("改动状态 = %+v，期望 2 项改动", dirty)
	}
}

func TestGitStatusDetached(t *testing.T) {
	requireGit(t)
	root := initRepo(t)
	writeFile(t, filepath.Join(root, "a.txt"), []byte("one"))
	runGitT(t, root, "add", "a.txt")
	runGitT(t, root, "commit", "-q", "-m", "init")
	runGitT(t, root, "checkout", "-q", "--detach")
	ws, err := NewWorkspace(root)
	if err != nil {
		t.Fatalf("创建工作区: %v", err)
	}
	status, err := ws.GitStatus(context.Background())
	if err != nil {
		t.Fatalf("读取 git 状态失败: %v", err)
	}
	if !status.IsRepo || !status.Detached || status.Branch != "" {
		t.Fatalf("分离头指针状态 = %+v", status)
	}
}

func TestGitStatusEmptyRepo(t *testing.T) {
	requireGit(t)
	root := initRepo(t)
	ws, err := NewWorkspace(root)
	if err != nil {
		t.Fatalf("创建工作区: %v", err)
	}
	status, err := ws.GitStatus(context.Background())
	if err != nil {
		t.Fatalf("读取 git 状态失败: %v", err)
	}
	if !status.IsRepo || status.Branch != "main" || status.Dirty {
		t.Fatalf("空仓库状态 = %+v", status)
	}
}

func TestGitStatusCanceled(t *testing.T) {
	requireGit(t)
	ws, _ := newTestWorkspace(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ws.GitStatus(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("取消的上下文应返回 context.Canceled，得到 %v", err)
	}
}

func TestGitStatusUnavailable(t *testing.T) {
	t.Setenv("PATH", "")
	ws, _ := newTestWorkspace(t)
	if _, err := ws.GitStatus(context.Background()); !errors.Is(err, ErrGitUnavailable) {
		t.Fatalf("没有 git 时应返回 ErrGitUnavailable，得到 %v", err)
	}
}

func TestParseBranchHeader(t *testing.T) {
	cases := []struct {
		header   string
		branch   string
		detached bool
	}{
		{"## main", "main", false},
		{"## main...origin/main [ahead 1]", "main", false},
		{"## main...origin/main [ahead 1, behind 2]", "main", false},
		{"## HEAD (no branch)", "", true},
		{"## No commits yet on main", "main", false},
		{"## Initial commit on trunk", "trunk", false},
		{"## feature/foo...origin/feature/foo", "feature/foo", false},
		{"##", "", false},
	}
	for _, tc := range cases {
		branch, detached := parseBranchHeader(tc.header)
		if branch != tc.branch || detached != tc.detached {
			t.Errorf("parseBranchHeader(%q) = (%q, %v)，期望 (%q, %v)",
				tc.header, branch, detached, tc.branch, tc.detached)
		}
	}
}

func TestParsePorcelainZ(t *testing.T) {
	out := []byte(" M changed.txt\x00?? new.txt\x00A  added.txt\x00R  moved.txt\x00old.txt\x00")
	states := parsePorcelainZ(out)
	want := map[string]GitState{
		"changed.txt": GitModified,
		"new.txt":     GitUntracked,
		"added.txt":   GitAdded,
		"moved.txt":   GitRenamed,
	}
	if len(states) != len(want) {
		t.Fatalf("解析结果 = %v，期望 %v", states, want)
	}
	for path, state := range want {
		if states[path] != state {
			t.Errorf("%s 状态 = %q，期望 %q", path, states[path], state)
		}
	}
}
