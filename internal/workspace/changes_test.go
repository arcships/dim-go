package workspace

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fakeChangeSource 是测试用的工具结果声明来源。
type fakeChangeSource struct {
	declarations []ToolResultDeclaration
	err          error
	called       bool
}

var _ ChangeSource = (*fakeChangeSource)(nil)

func (f *fakeChangeSource) ToolResultDeclarations(ctx context.Context, sessionID, runID string) ([]ToolResultDeclaration, error) {
	f.called = true
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return f.declarations, f.err
}

func TestReadRunChangesMergesAndClassifies(t *testing.T) {
	ws, root := newTestWorkspace(t)
	writeFile(t, filepath.Join(root, "a.txt"), []byte("a"))
	writeFile(t, filepath.Join(root, "sub", "b.txt"), []byte("b"))
	outsideDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("解析外部临时目录: %v", err)
	}
	outsideFile := filepath.Join(outsideDir, "outside.txt")
	writeFile(t, outsideFile, []byte("outside"))
	missingOutside := filepath.Join(filepath.Dir(root), "outside-relative.txt")

	source := &fakeChangeSource{declarations: []ToolResultDeclaration{
		{ToolCallID: "c1", ToolName: "write", Paths: []string{
			"a.txt", "./sub/../a.txt", "sub/b.txt", filepath.Join(root, "a.txt"), "",
		}},
		{ToolCallID: "c2", ToolName: "edit", Paths: []string{
			outsideFile, "../outside-relative.txt", "missing.txt",
		}},
	}}
	result, err := ws.ReadRunChanges(context.Background(), source, "session-1", "run-1")
	if err != nil {
		t.Fatalf("读取变更失败: %v", err)
	}
	if result.SessionID != "session-1" || result.RunID != "run-1" || result.Root != root {
		t.Fatalf("结果标识 = %+v", result)
	}
	if result.Coverage != CoverageToolReports || result.Declarations != 2 {
		t.Fatalf("覆盖信息 = %+v", result)
	}
	if result.Total != 5 || len(result.Paths) != 5 || result.Truncated {
		t.Fatalf("路径清单 = %+v", result)
	}

	wantWorkspace := []struct {
		path   string
		exists bool
	}{
		{"a.txt", true},
		{"missing.txt", false},
		{"sub/b.txt", true},
	}
	for i, item := range wantWorkspace {
		got := result.Paths[i]
		if got.Path != item.path || got.Scope != ChangeWorkspace || got.Exists != item.exists {
			t.Errorf("第 %d 条 = %+v，期望 workspace path=%q exists=%v",
				i, got, item.path, item.exists)
		}
		if !filepath.IsAbs(got.AbsPath) {
			t.Errorf("第 %d 条缺少绝对路径: %q", i, got.AbsPath)
		}
	}
	// 工作区内的相对路径以 "/" 分隔，且 AbsPath 已解析符号链接。
	if result.Paths[2].AbsPath != filepath.Join(root, "sub", "b.txt") {
		t.Errorf("相对路径的绝对形态 = %q", result.Paths[2].AbsPath)
	}
	// 工作区外按绝对路径排序；两条外部路径的相对顺序不写死，按集合核对。
	wantExternal := map[string]bool{outsideFile: true, missingOutside: false}
	for _, got := range result.Paths[3:] {
		if got.Scope != ChangeExternal || !filepath.IsAbs(got.Path) {
			t.Errorf("外部条目 = %+v", got)
			continue
		}
		exists, ok := wantExternal[got.Path]
		if !ok {
			t.Errorf("意外外部路径 = %q", got.Path)
			continue
		}
		if got.Exists != exists || got.AbsPath != got.Path {
			t.Errorf("外部条目 %q = %+v，期望 exists=%v 且 AbsPath=Path", got.Path, got, exists)
		}
	}
}

func TestReadRunChangesEmptyAndNilSource(t *testing.T) {
	ws, root := newTestWorkspace(t)
	writeFile(t, filepath.Join(root, "not-a-declared-change.txt"), []byte("x"))

	result, err := ws.ReadRunChanges(context.Background(), nil, "s", "r")
	if err != nil {
		t.Fatalf("nil source 不应报错: %v", err)
	}
	if result.Total != 0 || len(result.Paths) != 0 || result.Truncated {
		t.Fatalf("nil source 结果 = %+v", result)
	}
	if result.Coverage != CoverageToolReports {
		t.Fatalf("覆盖级别 = %q", result.Coverage)
	}

	empty := &fakeChangeSource{}
	result, err = ws.ReadRunChanges(context.Background(), empty, "s", "r")
	if err != nil {
		t.Fatalf("空声明不应报错: %v", err)
	}
	if result.Total != 0 || len(result.Paths) != 0 || result.Declarations != 0 {
		t.Fatalf("空声明结果 = %+v", result)
	}
	if !empty.called {
		t.Fatal("应调用一次声明来源")
	}
}

func TestReadRunChangesSourceErrorAndCancel(t *testing.T) {
	ws, _ := newTestWorkspace(t)
	source := &fakeChangeSource{err: errors.New("投影读取失败")}
	if _, err := ws.ReadRunChanges(context.Background(), source, "s", "r"); err == nil {
		t.Fatal("来源错误应向上返回")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	source = &fakeChangeSource{declarations: []ToolResultDeclaration{{Paths: []string{"a.txt"}}}}
	if _, err := ws.ReadRunChanges(ctx, source, "s", "r"); !errors.Is(err, context.Canceled) {
		t.Fatalf("取消的上下文应返回 context.Canceled，得到 %v", err)
	}
	if source.called {
		t.Fatal("上下文已取消时不应调用声明来源")
	}
}

func TestReadRunChangesTruncates(t *testing.T) {
	ws, _ := newTestWorkspace(t)
	paths := make([]string, 0, MaxChangePaths+20)
	for i := 0; i < MaxChangePaths+20; i++ {
		paths = append(paths, fmt.Sprintf("p%03d.txt", i))
	}
	source := &fakeChangeSource{declarations: []ToolResultDeclaration{{Paths: paths}}}
	result, err := ws.ReadRunChanges(context.Background(), source, "s", "r")
	if err != nil {
		t.Fatalf("读取变更失败: %v", err)
	}
	if !result.Truncated || result.Total != len(paths) || len(result.Paths) != MaxChangePaths {
		t.Fatalf("截断结果 = total %d, kept %d, truncated %v", result.Total, len(result.Paths), result.Truncated)
	}
	if result.Note == "" || !strings.Contains(result.Note, "截断") {
		t.Fatalf("截断说明 = %q", result.Note)
	}
	if result.Paths[len(result.Paths)-1].Path != fmt.Sprintf("p%03d.txt", MaxChangePaths-1) {
		t.Fatalf("截断后最后一条 = %q", result.Paths[len(result.Paths)-1].Path)
	}
}

func TestReadRunChangesSkipsRootAndBlank(t *testing.T) {
	ws, root := newTestWorkspace(t)
	source := &fakeChangeSource{declarations: []ToolResultDeclaration{{Paths: []string{"", "  ", ".", "./", root}}}}
	result, err := ws.ReadRunChanges(context.Background(), source, "s", "r")
	if err != nil {
		t.Fatalf("读取变更失败: %v", err)
	}
	if result.Total != 0 {
		t.Fatalf("根目录与空路径不应计入改动: %+v", result.Paths)
	}
}

func TestReadRunChangesHomeExpansion(t *testing.T) {
	tempHome, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("解析临时 home: %v", err)
	}
	t.Setenv("HOME", tempHome)
	writeFile(t, filepath.Join(tempHome, "notes.md"), []byte("home"))
	ws, _ := newTestWorkspace(t)
	source := &fakeChangeSource{declarations: []ToolResultDeclaration{{Paths: []string{"~/notes.md"}}}}
	result, err := ws.ReadRunChanges(context.Background(), source, "s", "r")
	if err != nil {
		t.Fatalf("读取变更失败: %v", err)
	}
	if len(result.Paths) != 1 {
		t.Fatalf("路径数 = %d", len(result.Paths))
	}
	got := result.Paths[0]
	want := filepath.Join(tempHome, "notes.md")
	if got.Scope != ChangeExternal || got.Path != want || got.AbsPath != want || !got.Exists {
		t.Fatalf("home 展开结果 = %+v，期望 %q", got, want)
	}
}

func TestReadRunChangesSymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 上创建符号链接需要额外权限")
	}
	ws, root := newTestWorkspace(t)
	outsideDir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatalf("解析外部临时目录: %v", err)
	}
	outsideFile := filepath.Join(outsideDir, "secret.txt")
	writeFile(t, outsideFile, []byte("secret"))
	if err := os.Symlink(outsideDir, filepath.Join(root, "link")); err != nil {
		t.Fatalf("创建符号链接: %v", err)
	}
	source := &fakeChangeSource{declarations: []ToolResultDeclaration{{Paths: []string{"link/secret.txt"}}}}
	result, err := ws.ReadRunChanges(context.Background(), source, "s", "r")
	if err != nil {
		t.Fatalf("读取变更失败: %v", err)
	}
	if len(result.Paths) != 1 {
		t.Fatalf("路径数 = %d", len(result.Paths))
	}
	if got := result.Paths[0]; got.Scope != ChangeExternal || got.Path != outsideFile {
		t.Fatalf("符号链接越界应归为 external: %+v", got)
	}
}
