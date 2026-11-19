package dirpick

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

// fixedRunner 是一个可注入的执行器：记下命令，返回给定的输出与错误。
type fixedRunner struct {
	out  string
	err  error
	name string
	args []string
}

func (f *fixedRunner) run(_ context.Context, name string, args ...string) (string, error) {
	f.name, f.args = name, append([]string(nil), args...)
	return f.out, f.err
}

// 选择成功：osascript 给的路径去掉末尾分隔符，命令是 choose folder 的脚本。
func TestChooseDarwinSuccess(t *testing.T) {
	run := &fixedRunner{out: "/Users/someone/projects/demo/\n"}
	path, err := Choose(context.Background(), run.run, "darwin")
	if err != nil {
		t.Fatalf("选择成功却返回错误：%v", err)
	}
	if path != "/Users/someone/projects/demo" {
		t.Fatalf("路径 = %q，想要去掉末尾分隔符的绝对路径", path)
	}
	if run.name != "osascript" || len(run.args) != 2 || run.args[0] != "-e" ||
		!strings.Contains(run.args[1], "choose folder") {
		t.Fatalf("命令不是 osascript 的 choose folder：%s %v", run.name, run.args)
	}
}

// 取消：osascript 以 -128 失败，或退出码为 0 但没有输出；两种情况都不算失败。
func TestChooseCancel(t *testing.T) {
	cases := []struct {
		name string
		out  string
		err  error
	}{
		{"退出码 -128", "execution error: User canceled. (-128)", errors.New("exit status 1")},
		{"没有输出", "", nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			run := &fixedRunner{out: tc.out, err: tc.err}
			if _, err := Choose(context.Background(), run.run, "darwin"); !errors.Is(err, ErrCancelled) {
				t.Fatalf("取消返回 %v，想要 ErrCancelled", err)
			}
		})
	}
}

// 命令失败：给一句可读原因（取命令自己的第一行输出），不是 ErrCancelled。
func TestChooseCommandFailure(t *testing.T) {
	run := &fixedRunner{
		out: "execution error: Not authorized to send Apple events to System Events. (-1743)",
		err: errors.New("exit status 1"),
	}
	_, err := Choose(context.Background(), run.run, "darwin")
	if err == nil || errors.Is(err, ErrCancelled) {
		t.Fatalf("失败被当成取消：%v", err)
	}
	if !strings.Contains(err.Error(), "Not authorized") {
		t.Fatalf("原因里没有命令的输出：%v", err)
	}
}

// 超时：命令被 context 掐掉时给一句超时原因。
func TestChooseTimeout(t *testing.T) {
	run := Runner(func(ctx context.Context, _ string, _ ...string) (string, error) {
		<-ctx.Done()
		return "", ctx.Err()
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err := Choose(ctx, run, "darwin")
	if err == nil || !strings.Contains(err.Error(), "超时") {
		t.Fatalf("超时返回 %v，想要一句超时原因", err)
	}
}

// 非 macOS：明确回退信号，不执行任何命令。
func TestChooseUnsupportedPlatform(t *testing.T) {
	run := &fixedRunner{out: "/never/used"}
	if _, err := Choose(context.Background(), run.run, "linux"); !errors.Is(err, ErrUnsupported) {
		t.Fatalf("非 darwin 返回 %v，想要 ErrUnsupported", err)
	}
	if run.name != "" {
		t.Fatalf("非 darwin 仍执行了命令：%s", run.name)
	}
}
