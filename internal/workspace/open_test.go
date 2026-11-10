package workspace

import (
	"errors"
	"fmt"
	"path/filepath"
	"runtime"
	"testing"
)

// useFakeRunner 替换外部命令执行器，并返回记录到的调用参数。
func useFakeRunner(t *testing.T, code int, err error) *[][]string {
	t.Helper()
	old := externalRunner
	t.Cleanup(func() { externalRunner = old })
	calls := make([][]string, 0, 1)
	externalRunner = func(name string, args ...string) (int, error) {
		call := append([]string{name}, args...)
		calls = append(calls, call)
		return code, err
	}
	return &calls
}

func TestOpenSuccess(t *testing.T) {
	ws, root := newTestWorkspace(t)
	writeFile(t, filepath.Join(root, "a.txt"), []byte("a"))
	calls := useFakeRunner(t, 0, nil)

	code, err := ws.Open("a.txt")
	if err != nil {
		t.Fatalf("外部打开失败: %v", err)
	}
	if code != 0 {
		t.Fatalf("退出码 = %d，期望 0", code)
	}
	callsValue := *calls
	if len(callsValue) != 1 {
		t.Fatalf("调用次数 = %d，期望 1", len(callsValue))
	}
	wantSpec, specErr := openCommandFor(runtime.GOOS, filepath.Join(root, "a.txt"))
	if specErr != nil {
		t.Skipf("当前平台没有外部打开命令: %v", specErr)
	}
	if callsValue[0][0] != wantSpec.name {
		t.Fatalf("命令 = %q，期望 %q", callsValue[0][0], wantSpec.name)
	}
	if got := callsValue[0][len(callsValue[0])-1]; got != filepath.Join(root, "a.txt") {
		t.Fatalf("打开参数 = %q，期望规范化后的绝对路径", got)
	}
}

func TestOpenExitCode(t *testing.T) {
	ws, root := newTestWorkspace(t)
	writeFile(t, filepath.Join(root, "a.txt"), []byte("a"))
	useFakeRunner(t, 7, fmt.Errorf("%w: 退出码 7", ErrOpenFailed))

	code, err := ws.Open("a.txt")
	if code != 7 {
		t.Fatalf("退出码 = %d，期望 7", code)
	}
	if !errors.Is(err, ErrOpenFailed) {
		t.Fatalf("错误应包装 ErrOpenFailed，得到 %v", err)
	}
}

func TestOpenRejectsMissingAndOutside(t *testing.T) {
	ws, _ := newTestWorkspace(t)
	calls := useFakeRunner(t, 0, nil)

	if _, err := ws.Open("missing.txt"); !errors.Is(err, ErrPathNotExist) {
		t.Fatalf("缺失路径应返回 ErrPathNotExist，得到 %v", err)
	}
	if _, err := ws.Open("../outside.txt"); !errors.Is(err, ErrOutsideWorkspace) {
		t.Fatalf("越界路径应返回 ErrOutsideWorkspace，得到 %v", err)
	}
	if len(*calls) != 0 {
		t.Fatalf("被拒绝的路径不应启动外部命令: %v", *calls)
	}
}

func TestOpenDirectory(t *testing.T) {
	ws, root := newTestWorkspace(t)
	calls := useFakeRunner(t, 0, nil)
	if code, err := ws.Open(""); err != nil || code != 0 {
		t.Fatalf("打开目录 = %d, %v", code, err)
	}
	if len(*calls) != 1 {
		t.Fatalf("打开目录应调用一次外部命令")
	}
	if got := (*calls)[0][len((*calls)[0])-1]; got != root {
		t.Fatalf("目录参数 = %q，期望 %q", got, root)
	}
}

func TestOpenCommandFor(t *testing.T) {
	cases := []struct {
		goos string
		name string
		ok   bool
	}{
		{"darwin", "open", true},
		{"linux", "xdg-open", true},
		{"freebsd", "xdg-open", true},
		{"windows", "cmd", true},
		{"plan9", "", false},
	}
	for _, tc := range cases {
		spec, err := openCommandFor(tc.goos, "/tmp/file.txt")
		if tc.ok {
			if err != nil || spec.name != tc.name {
				t.Errorf("openCommandFor(%q) = %+v, %v", tc.goos, spec, err)
			}
			if spec.args[len(spec.args)-1] != "/tmp/file.txt" {
				t.Errorf("openCommandFor(%q) 参数 = %v", tc.goos, spec.args)
			}
			continue
		}
		if !errors.Is(err, ErrUnsupportedPlatform) {
			t.Errorf("openCommandFor(%q) 应返回 ErrUnsupportedPlatform，得到 %v", tc.goos, err)
		}
	}
}

func TestRunExternalCommand(t *testing.T) {
	if code, err := runExternalCommand("sh", "-c", "exit 0"); err != nil || code != 0 {
		t.Fatalf("成功命令 = %d, %v", code, err)
	}
	code, err := runExternalCommand("sh", "-c", "exit 7")
	if code != 7 || !errors.Is(err, ErrOpenFailed) {
		t.Fatalf("失败命令 = %d, %v，期望 7 与 ErrOpenFailed", code, err)
	}
	code, err = runExternalCommand("dimagent-no-such-command")
	if code != -1 || !errors.Is(err, ErrOpenFailed) {
		t.Fatalf("无法启动的命令 = %d, %v，期望 -1 与 ErrOpenFailed", code, err)
	}
}
