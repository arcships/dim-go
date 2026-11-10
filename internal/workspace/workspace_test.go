package workspace

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestNewWorkspace(t *testing.T) {
	if _, err := NewWorkspace(""); !errors.Is(err, ErrEmptyPath) {
		t.Fatalf("空路径应返回 ErrEmptyPath，得到 %v", err)
	}
	if _, err := NewWorkspace(filepath.Join(t.TempDir(), "missing")); !errors.Is(err, ErrPathNotExist) {
		t.Fatalf("不存在的根应返回 ErrPathNotExist，得到 %v", err)
	}
	file := filepath.Join(t.TempDir(), "file.txt")
	writeFile(t, file, []byte("x"))
	if _, err := NewWorkspace(file); !errors.Is(err, ErrNotDirectory) {
		t.Fatalf("文件根应返回 ErrNotDirectory，得到 %v", err)
	}
}

func TestResolveInside(t *testing.T) {
	ws, root := newTestWorkspace(t)
	writeFile(t, filepath.Join(root, "sub", "a.txt"), []byte("abc"))

	abs, err := ws.Resolve("sub/a.txt")
	if err != nil {
		t.Fatalf("解析工作区内路径失败: %v", err)
	}
	if abs != filepath.Join(root, "sub", "a.txt") {
		t.Fatalf("解析结果 = %s，期望 %s", abs, filepath.Join(root, "sub", "a.txt"))
	}

	// 绝对路径与带 .. 的路径都要先规范化再判断。
	abs, err = ws.Resolve(filepath.Join(root, "sub", "..", "sub", "a.txt"))
	if err != nil {
		t.Fatalf("解析带 .. 的路径失败: %v", err)
	}
	if abs != filepath.Join(root, "sub", "a.txt") {
		t.Fatalf("规范化结果 = %s", abs)
	}

	// 不存在的目标也允许解析，只要落在工作区内。
	missing, err := ws.Resolve("new/deep/file.txt")
	if err != nil {
		t.Fatalf("解析不存在的目标失败: %v", err)
	}
	if missing != filepath.Join(root, "new", "deep", "file.txt") {
		t.Fatalf("不存在目标的解析结果 = %s", missing)
	}

	rel, err := ws.Relative("")
	if err != nil || rel != "." {
		t.Fatalf("根目录的相对路径 = %q, %v", rel, err)
	}
	rel, err = ws.Relative("sub/a.txt")
	if err != nil || rel != filepath.Join("sub", "a.txt") {
		t.Fatalf("相对路径 = %q, %v", rel, err)
	}
}

func TestResolveRejectsEscape(t *testing.T) {
	ws, root := newTestWorkspace(t)
	outside := t.TempDir()
	writeFile(t, filepath.Join(outside, "secret.txt"), []byte("secret"))

	cases := []string{
		"..",
		"../secret.txt",
		filepath.Join(root, "..", "secret.txt"),
		filepath.Join(outside, "secret.txt"),
	}
	for _, rel := range cases {
		if _, err := ws.Resolve(rel); !errors.Is(err, ErrOutsideWorkspace) {
			t.Errorf("Resolve(%q) 应返回 ErrOutsideWorkspace，得到 %v", rel, err)
		}
	}
}

func TestResolveRejectsPrefixSibling(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "ws")
	sibling := filepath.Join(parent, "ws-other")
	writeFile(t, filepath.Join(root, "a.txt"), []byte("a"))
	writeFile(t, filepath.Join(sibling, "b.txt"), []byte("b"))
	ws, err := NewWorkspace(root)
	if err != nil {
		t.Fatalf("创建工作区: %v", err)
	}
	if _, err := ws.Resolve(sibling); !errors.Is(err, ErrOutsideWorkspace) {
		t.Fatalf("同前缀兄弟目录应被拒绝，得到 %v", err)
	}
}

func TestResolveSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 上创建符号链接需要额外权限")
	}
	ws, root := newTestWorkspace(t)
	writeFile(t, filepath.Join(root, "real", "inside.txt"), []byte("inside"))
	if err := os.Symlink(filepath.Join(root, "real"), filepath.Join(root, "link")); err != nil {
		t.Fatalf("创建目录符号链接: %v", err)
	}
	abs, err := ws.Resolve("link/inside.txt")
	if err != nil {
		t.Fatalf("工作区内的符号链接应可用: %v", err)
	}
	if abs != filepath.Join(root, "real", "inside.txt") {
		t.Fatalf("符号链接解析结果 = %s", abs)
	}

	outside := t.TempDir()
	writeFile(t, filepath.Join(outside, "secret.txt"), []byte("secret"))
	if err := os.Symlink(outside, filepath.Join(root, "escape")); err != nil {
		t.Fatalf("创建越界目录符号链接: %v", err)
	}
	if _, err := ws.Resolve("escape/secret.txt"); !errors.Is(err, ErrOutsideWorkspace) {
		t.Fatalf("目录符号链接越界应被拒绝，得到 %v", err)
	}
	if err := os.Symlink(filepath.Join(outside, "secret.txt"), filepath.Join(root, "secret-link")); err != nil {
		t.Fatalf("创建越界文件符号链接: %v", err)
	}
	if _, err := ws.Resolve("secret-link"); !errors.Is(err, ErrOutsideWorkspace) {
		t.Fatalf("文件符号链接越界应被拒绝，得到 %v", err)
	}
}

func TestResolveIntermediateFile(t *testing.T) {
	ws, root := newTestWorkspace(t)
	writeFile(t, filepath.Join(root, "a.txt"), []byte("abc"))
	if _, err := ws.Resolve("a.txt/child"); !errors.Is(err, ErrNotDirectory) {
		t.Fatalf("普通文件下的路径应返回 ErrNotDirectory，得到 %v", err)
	}
}
