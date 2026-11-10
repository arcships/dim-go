package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestListDirEntries(t *testing.T) {
	ws, root := newTestWorkspace(t)
	writeFile(t, filepath.Join(root, "a.txt"), []byte("abc"))
	writeFile(t, filepath.Join(root, "b.txt"), []byte("12345"))
	writeFile(t, filepath.Join(root, "sub", "inner.go"), []byte("package sub"))
	writeFile(t, filepath.Join(root, ".hidden"), []byte("h"))

	listing, err := ws.ListDir("", ListOptions{SkipGit: true})
	if err != nil {
		t.Fatalf("列目录失败: %v", err)
	}
	if listing.Path != "." {
		t.Fatalf("根目录 Path = %q，期望 \".\"", listing.Path)
	}
	if listing.Total != 4 || len(listing.Entries) != 4 {
		t.Fatalf("条目数 = %d/%d，期望 4", len(listing.Entries), listing.Total)
	}
	names := make([]string, 0, len(listing.Entries))
	for _, entry := range listing.Entries {
		names = append(names, entry.Name)
	}
	want := []string{".hidden", "a.txt", "b.txt", "sub"}
	for i, name := range want {
		if names[i] != name {
			t.Fatalf("条目顺序 = %v，期望 %v", names, want)
		}
	}
	if listing.Entries[0].Kind != EntryFile {
		t.Fatalf("隐藏文件类型 = %s，期望 file", listing.Entries[0].Kind)
	}
	if listing.Entries[1].Size != 3 {
		t.Fatalf("a.txt 大小 = %d，期望 3", listing.Entries[1].Size)
	}
	if listing.Entries[1].ModTime.IsZero() {
		t.Fatal("修改时间不应为零值")
	}
	if listing.Entries[3].Kind != EntryDir {
		t.Fatalf("sub 类型 = %s，期望 dir", listing.Entries[3].Kind)
	}
	if listing.GitAvailable {
		t.Fatal("SkipGit 时不应报告 git 可用")
	}

	sub, err := ws.ListDir("sub", ListOptions{SkipGit: true})
	if err != nil {
		t.Fatalf("列子目录失败: %v", err)
	}
	if sub.Path != "sub" || len(sub.Entries) != 1 || sub.Entries[0].Name != "inner.go" {
		t.Fatalf("子目录列表 = %+v", sub)
	}
}

func TestListDirLimitsAndErrors(t *testing.T) {
	ws, root := newTestWorkspace(t)
	writeFile(t, filepath.Join(root, "a.txt"), []byte("a"))
	writeFile(t, filepath.Join(root, "b.txt"), []byte("b"))
	writeFile(t, filepath.Join(root, "c.txt"), []byte("c"))

	listing, err := ws.ListDir("", ListOptions{MaxEntries: 2, SkipGit: true})
	if err != nil {
		t.Fatalf("列目录失败: %v", err)
	}
	if len(listing.Entries) != 2 || listing.Total != 3 || !listing.Truncated {
		t.Fatalf("截断结果 = %+v", listing)
	}
	if listing.Note == "" {
		t.Fatal("截断时必须有说明")
	}

	all, err := ws.ListDir("", ListOptions{MaxEntries: -1, SkipGit: true})
	if err != nil || len(all.Entries) != 3 || all.Truncated {
		t.Fatalf("不限制时结果 = %+v, %v", all, err)
	}

	if _, err := ws.ListDir("missing", ListOptions{SkipGit: true}); !errors.Is(err, ErrPathNotExist) {
		t.Fatalf("缺失目录应返回 ErrPathNotExist，得到 %v", err)
	}
	if _, err := ws.ListDir("a.txt", ListOptions{SkipGit: true}); !errors.Is(err, ErrNotDirectory) {
		t.Fatalf("文件应返回 ErrNotDirectory，得到 %v", err)
	}
	if _, err := ws.ListDir("..", ListOptions{SkipGit: true}); !errors.Is(err, ErrOutsideWorkspace) {
		t.Fatalf("越界目录应返回 ErrOutsideWorkspace，得到 %v", err)
	}
}

func TestListDirSymlinkKind(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 上创建符号链接需要额外权限")
	}
	ws, root := newTestWorkspace(t)
	writeFile(t, filepath.Join(root, "a.txt"), []byte("a"))
	if err := os.Symlink(filepath.Join(root, "a.txt"), filepath.Join(root, "link.txt")); err != nil {
		t.Fatalf("创建符号链接: %v", err)
	}
	listing, err := ws.ListDir("", ListOptions{SkipGit: true})
	if err != nil {
		t.Fatalf("列目录失败: %v", err)
	}
	for _, entry := range listing.Entries {
		if entry.Name == "link.txt" && entry.Kind != EntrySymlink {
			t.Fatalf("符号链接类型 = %s，期望 symlink", entry.Kind)
		}
	}
}

func TestListDirGitStatus(t *testing.T) {
	requireGit(t)
	root := initRepo(t)
	writeFile(t, filepath.Join(root, "tracked.txt"), []byte("one"))
	writeFile(t, filepath.Join(root, "dir", "tracked-in-dir.txt"), []byte("one"))
	runGitT(t, root, "add", "tracked.txt", "dir/tracked-in-dir.txt")
	runGitT(t, root, "commit", "-q", "-m", "init")

	writeFile(t, filepath.Join(root, "tracked.txt"), []byte("two"))
	writeFile(t, filepath.Join(root, "dir", "tracked-in-dir.txt"), []byte("two"))
	writeFile(t, filepath.Join(root, "untracked.txt"), []byte("new"))
	writeFile(t, filepath.Join(root, "sub", "deep.txt"), []byte("deep"))

	ws, err := NewWorkspace(root)
	if err != nil {
		t.Fatalf("创建工作区: %v", err)
	}
	listing, err := ws.ListDir("", ListOptions{})
	if err != nil {
		t.Fatalf("列目录失败: %v", err)
	}
	if !listing.GitAvailable {
		t.Fatal("git 仓库应报告 GitAvailable")
	}
	statuses := make(map[string]GitState)
	for _, entry := range listing.Entries {
		statuses[entry.Name] = entry.Git
	}
	if statuses["tracked.txt"] != GitModified {
		t.Errorf("tracked.txt 状态 = %q，期望 modified", statuses["tracked.txt"])
	}
	if statuses["untracked.txt"] != GitUntracked {
		t.Errorf("untracked.txt 状态 = %q，期望 untracked", statuses["untracked.txt"])
	}
	if statuses["dir"] != GitModified {
		t.Errorf("dir 聚合状态 = %q，期望 modified", statuses["dir"])
	}
	if statuses["sub"] != GitUntracked {
		t.Errorf("sub 聚合状态 = %q，期望 untracked", statuses["sub"])
	}

	skipped, err := ws.ListDir("", ListOptions{SkipGit: true})
	if err != nil {
		t.Fatalf("列目录失败: %v", err)
	}
	if skipped.GitAvailable {
		t.Fatal("SkipGit 时不应读取 git")
	}
	for _, entry := range skipped.Entries {
		if entry.Git != GitNone {
			t.Fatalf("SkipGit 时 %s 状态应为空", entry.Name)
		}
	}
}

func TestListDirOutsideGitRepo(t *testing.T) {
	ws, root := newTestWorkspace(t)
	writeFile(t, filepath.Join(root, "a.txt"), []byte("a"))
	listing, err := ws.ListDir("", ListOptions{})
	if err != nil {
		t.Fatalf("列目录失败: %v", err)
	}
	if listing.GitAvailable {
		t.Fatal("非仓库目录不应报告 git 可用")
	}
	if listing.Note == "" {
		t.Fatal("git 不可用时应给出说明")
	}
	if listing.Entries[0].Git != GitNone {
		t.Fatalf("非仓库条目状态 = %q，期望空", listing.Entries[0].Git)
	}
}
