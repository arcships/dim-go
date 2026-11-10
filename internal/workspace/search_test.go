package workspace

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// prepareSearchTree 构造搜索测试用的目录树。
func prepareSearchTree(t *testing.T) (*Workspace, string) {
	t.Helper()
	ws, root := newTestWorkspace(t)
	writeFile(t, filepath.Join(root, ".hidden.txt"), []byte("hello hidden\n"))
	writeFile(t, filepath.Join(root, "a.txt"), []byte("Hello World\nsecond hello line\n"))
	writeFile(t, filepath.Join(root, "b.go"), []byte("package b\n"))
	writeFile(t, filepath.Join(root, "binary.bin"), []byte{'h', 'e', 'l', 'l', 'o', 0})
	writeFile(t, filepath.Join(root, "big.txt"), []byte(strings.Repeat("hello ", 100)))
	writeFile(t, filepath.Join(root, "sub", "c.txt"), []byte("hello deep\n"))
	writeFile(t, filepath.Join(root, ".git", "config"), []byte("hello in git\n"))
	return ws, root
}

func TestSearchMatches(t *testing.T) {
	ws, _ := prepareSearchTree(t)
	result, err := ws.Search("", SearchOptions{Query: "hello"})
	if err != nil {
		t.Fatalf("搜索失败: %v", err)
	}
	if result.StopReason != StopNone || result.Truncated {
		t.Fatalf("完整扫描不应标记截断: %+v", result)
	}
	paths := make(map[string]int)
	for _, match := range result.Matches {
		paths[match.Path]++
	}
	want := map[string]int{
		".hidden.txt": 1,
		"a.txt":       2,
		"big.txt":     1,
		"sub/c.txt":   1,
	}
	if len(paths) != len(want) {
		t.Fatalf("命中路径 = %v，期望 %v", paths, want)
	}
	for path, count := range want {
		if paths[path] != count {
			t.Errorf("%s 命中数 = %d，期望 %d", path, paths[path], count)
		}
	}
	if _, ok := paths["binary.bin"]; ok {
		t.Error("二进制文件不应命中")
	}
	if _, ok := paths[".git/config"]; ok {
		t.Error(".git 目录不应被扫描")
	}

	for _, match := range result.Matches {
		if match.Path == "a.txt" && match.Line == 2 {
			if match.Column != 8 {
				t.Errorf("第二行命中列 = %d，期望 8", match.Column)
			}
			if match.Snippet != "second hello line" {
				t.Errorf("片段 = %q", match.Snippet)
			}
		}
	}
}

func TestSearchCaseInsensitive(t *testing.T) {
	ws, _ := prepareSearchTree(t)
	result, err := ws.Search("", SearchOptions{Query: "HELLO"})
	if err != nil {
		t.Fatalf("搜索失败: %v", err)
	}
	if len(result.Matches) != 5 {
		t.Fatalf("大小写不敏感命中数 = %d，期望 5", len(result.Matches))
	}
}

func TestSearchLimits(t *testing.T) {
	ws, _ := prepareSearchTree(t)

	limited, err := ws.Search("", SearchOptions{Query: "hello", MaxMatches: 2})
	if err != nil {
		t.Fatalf("搜索失败: %v", err)
	}
	if len(limited.Matches) != 2 || limited.StopReason != StopMaxMatches || !limited.Truncated {
		t.Fatalf("条数上限结果 = %+v", limited)
	}

	files, err := ws.Search("", SearchOptions{Query: "hello", MaxFiles: 1})
	if err != nil {
		t.Fatalf("搜索失败: %v", err)
	}
	if files.StopReason != StopMaxFiles || files.FilesScanned != 1 || len(files.Matches) != 1 {
		t.Fatalf("文件数上限结果 = %+v", files)
	}

	bytesLimited, err := ws.Search("", SearchOptions{Query: "hello", MaxScanBytes: 30})
	if err != nil {
		t.Fatalf("搜索失败: %v", err)
	}
	if bytesLimited.StopReason != StopMaxScanBytes || bytesLimited.FilesScanned != 1 {
		t.Fatalf("字节上限结果 = %+v", bytesLimited)
	}
	if bytesLimited.BytesScanned > 30 {
		t.Fatalf("字节上限被突破: %d", bytesLimited.BytesScanned)
	}

	small, err := ws.Search("", SearchOptions{Query: "hello", MaxFileBytes: 20})
	if err != nil {
		t.Fatalf("搜索失败: %v", err)
	}
	for _, match := range small.Matches {
		if match.Path != ".hidden.txt" && match.Path != "sub/c.txt" {
			t.Fatalf("大文件不应参与匹配: %+v", match)
		}
	}
	if len(small.Matches) != 2 {
		t.Fatalf("单文件上限命中数 = %d，期望 2", len(small.Matches))
	}
}

func TestSearchSnippetAndScope(t *testing.T) {
	ws, _ := prepareSearchTree(t)
	result, err := ws.Search("sub", SearchOptions{Query: "hello", SnippetRunes: 5})
	if err != nil {
		t.Fatalf("搜索子目录失败: %v", err)
	}
	if len(result.Matches) != 1 {
		t.Fatalf("子目录命中数 = %d，期望 1", len(result.Matches))
	}
	match := result.Matches[0]
	if match.Path != "sub/c.txt" {
		t.Fatalf("路径 = %q，期望 sub/c.txt", match.Path)
	}
	if match.Snippet != "hello…" {
		t.Fatalf("片段 = %q，期望 hello…", match.Snippet)
	}

	if _, err := ws.Search("", SearchOptions{Query: " \t "}); !errors.Is(err, ErrEmptyQuery) {
		t.Fatalf("空白关键词应返回 ErrEmptyQuery，得到 %v", err)
	}
	if _, err := ws.Search("..", SearchOptions{Query: "hello"}); !errors.Is(err, ErrOutsideWorkspace) {
		t.Fatalf("越界搜索应返回 ErrOutsideWorkspace，得到 %v", err)
	}
	if _, err := ws.Search("a.txt", SearchOptions{Query: "hello"}); !errors.Is(err, ErrNotDirectory) {
		t.Fatalf("文件起点应返回 ErrNotDirectory，得到 %v", err)
	}
}

func TestSearchSkipsSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 上创建符号链接需要额外权限")
	}
	ws, root := prepareSearchTree(t)
	outside := t.TempDir()
	outsideFile := filepath.Join(outside, "outside.txt")
	writeFile(t, outsideFile, []byte("hello outside\n"))
	if err := os.Symlink(outsideFile, filepath.Join(root, "outside-link.txt")); err != nil {
		t.Fatalf("创建符号链接: %v", err)
	}
	result, err := ws.Search("", SearchOptions{Query: "hello outside"})
	if err != nil {
		t.Fatalf("搜索失败: %v", err)
	}
	if len(result.Matches) != 0 {
		t.Fatalf("符号链接不应被扫描: %+v", result.Matches)
	}
}

func TestSearchCreatesNoIndex(t *testing.T) {
	ws, root := prepareSearchTree(t)
	before := treeSnapshot(t, root)
	if _, err := ws.Search("", SearchOptions{Query: "hello"}); err != nil {
		t.Fatalf("搜索失败: %v", err)
	}
	after := treeSnapshot(t, root)
	if strings.Join(before, "\n") != strings.Join(after, "\n") {
		t.Fatalf("搜索不应创建索引或缓存文件\nbefore=%v\nafter=%v", before, after)
	}
}

// treeSnapshot 记录目录树的相对路径，用于确认搜索没有落盘。
func treeSnapshot(t *testing.T, root string) []string {
	t.Helper()
	var paths []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		paths = append(paths, rel)
		return nil
	})
	if err != nil {
		t.Fatalf("遍历测试目录: %v", err)
	}
	return paths
}

func TestSearchUnicodeFold(t *testing.T) {
	ws, root := newTestWorkspace(t)
	// 开尔文符号（3 字节）与长 s（2 字节）都会在大小写转换时改变字节长度。
	writeFile(t, filepath.Join(root, "fold.txt"), []byte("\u212a hello\n\u017fun\n"))
	result, err := ws.Search("", SearchOptions{Query: "hello"})
	if err != nil {
		t.Fatalf("搜索失败: %v", err)
	}
	if len(result.Matches) != 1 || result.Matches[0].Column != 3 {
		t.Fatalf("多字节折叠后的列号 = %+v，期望第 1 行第 3 列", result.Matches)
	}

	result, err = ws.Search("", SearchOptions{Query: "s"})
	if err != nil {
		t.Fatalf("搜索失败: %v", err)
	}
	if len(result.Matches) != 1 || result.Matches[0].Line != 2 || result.Matches[0].Column != 1 {
		t.Fatalf("长 s 匹配结果 = %+v", result.Matches)
	}
}
