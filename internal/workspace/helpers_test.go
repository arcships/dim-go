package workspace

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// requireGit 在没有 git 的本机上跳过依赖 git 的测试。
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("本机没有 git，跳过依赖 git 的测试")
	}
}

// runGitT 在指定目录执行 git，失败即终止测试。
func runGitT(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_OPTIONAL_LOCKS=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v 失败: %v\n%s", args, err, out)
	}
	return string(out)
}

// initRepo 初始化一个分支名为 main 的空仓库。
func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGitT(t, dir, "init", "-q", "-b", "main")
	runGitT(t, dir, "config", "user.email", "workspace-test@example.com")
	runGitT(t, dir, "config", "user.name", "workspace test")
	return dir
}

// writeFile 写入测试文件并自动创建父目录。
func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("创建目录 %s: %v", filepath.Dir(path), err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("写入文件 %s: %v", path, err)
	}
}

// newTestWorkspace 创建临时工作区。
func newTestWorkspace(t *testing.T) (*Workspace, string) {
	t.Helper()
	root := t.TempDir()
	ws, err := NewWorkspace(root)
	if err != nil {
		t.Fatalf("创建测试工作区: %v", err)
	}
	// macOS 的 /var 是指向 /private/var 的符号链接，统一使用解析后的根。
	return ws, ws.Root()
}
