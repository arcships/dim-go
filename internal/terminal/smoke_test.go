package terminal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// smokeTimeout 是冒烟用例每一步的严格上限。冒烟步骤只用「等标记出现」同步，
// 不用固定 sleep：固定 sleep 在并联跑包时可能还没等到判据就断言，造成偶发失败。
const smokeTimeout = 2 * time.Second

// lenBytes 返回已收集输出的字节数，用作「这一步之后的新输出」的基线。
func (c *capture) lenBytes() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.buf.Len()
}

// since 返回基线之后的输出；基线越界时按当前末尾处理。
func (c *capture) since(off int) string {
	c.mu.Lock()
	defer c.mu.Unlock()
	data := c.buf.Bytes()
	if off < 0 {
		off = 0
	}
	if off > len(data) {
		off = len(data)
	}
	return string(data[off:])
}

// waitAfter 等待基线之后的新输出里出现 want；超时带出这一步已读到的片段，
// 便于分辨「程序没起来」与「断言太早」。
func (c *capture) waitAfter(t *testing.T, off int, want string) string {
	t.Helper()
	deadline := time.Now().Add(smokeTimeout)
	for {
		out := c.since(off)
		if strings.Contains(out, want) {
			return out
		}
		if time.Now().After(deadline) {
			t.Fatalf("冒烟步骤等待 %q 超时（上限 %s）；该步骤已读到 %q",
				want, smokeTimeout, tail(out, 800))
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// tail 取字符串末尾 n 个字节。
func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}

// TestSmokeFullScreenPrograms 是全屏程序与中文输入的灌烟用例。
//
// 终端渲染结果没有可自动比对的判据，所以这里只断言：程序能起来（等到它确实画出内容）、
// 运行中变更尺寸不崩、能退出、退出后 shell 仍可执行命令，且多字节输入没有被改写。
// 用例依赖本机真实的 vim/less/top，默认跳过；设置 DIMAGENT_TERMINAL_SMOKE=1 运行：
//
//	DIMAGENT_TERMINAL_SMOKE=1 go test ./internal/terminal -run TestSmokeFullScreenPrograms -v
func TestSmokeFullScreenPrograms(t *testing.T) {
	if os.Getenv("DIMAGENT_TERMINAL_SMOKE") == "" {
		t.Skip("需要本机真实全屏程序，设置 DIMAGENT_TERMINAL_SMOKE=1 运行")
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "smoke.txt")
	const content = "中文内容"
	if err := os.WriteFile(file, []byte(content+"\n"), 0o600); err != nil {
		t.Fatalf("建冒烟文件: %v", err)
	}
	term := startTerm(t, "sess-smoke", dir, Options{})
	c := startCapture(t, term)

	// vim：文件内容出现在屏幕上即视为已进入；随后改尺寸、退出，
	// 用 shell 打印的 VIM-OUT 确认已经回到 shell。
	off := c.lenBytes()
	term.Write([]byte("vim " + file + "; printf 'VIM-%s\\n' OUT\r"))
	c.waitAfter(t, off, content)
	if err := term.Resize(120, 40); err != nil {
		t.Fatalf("vim 运行中变更尺寸: %v", err)
	}
	// ESC 与命令分两次写：vim 需要把 ESC 当作独立按键处理。
	term.Write([]byte("\x1b"))
	term.Write([]byte(":q!\r"))
	c.waitAfter(t, off, "VIM-OUT")

	// less：文件内容出现即视为已进入；q 退出后等 LES-OUT。
	off = c.lenBytes()
	term.Write([]byte("less " + file + "; printf 'LES-%s\\n' OUT\r"))
	c.waitAfter(t, off, content)
	term.Write([]byte("q"))
	c.waitAfter(t, off, "LES-OUT")

	// top：出现列头 PID 即视为已画出第一屏；q 退出后等 TOP-OUT。
	off = c.lenBytes()
	term.Write([]byte("top; printf 'TOP-%s\\n' OUT\r"))
	c.waitAfter(t, off, "PID")
	term.Write([]byte("q"))
	c.waitAfter(t, off, "TOP-OUT")

	// 中文输入灌烟：命令回显出现即视为该行已被 shell 读完，之后写入的多字节内容
	// 会排进终端输入队列，由 cat 按原始字节读走；Ctrl-D 结束后再等 CAT-OUT。
	off = c.lenBytes()
	term.Write([]byte("cat > /dev/null; printf 'CAT-%s\\n' OUT\r"))
	c.waitAfter(t, off, "cat > /dev/null")
	term.Write([]byte("中文输入\r"))
	term.Write([]byte("\x04"))
	c.waitAfter(t, off, "CAT-OUT")

	if strings.Contains(c.String(), "\uFFFD") {
		t.Fatalf("输出出现替换字符，说明多字节输入被改写")
	}
}
