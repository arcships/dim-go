package ui_test

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/arcships/dimagent-mygo/internal/app"
	"github.com/arcships/dimagent-mygo/internal/ui"
	"github.com/arcships/dimagent-mygo/internal/uiwire"

	gui "github.com/egoist/mygo/ui"
)

// 界面级端到端：终端不是假的。
//
// 这一条把三层都接起来跑一遍：真 App Server（`DIM_APP_SERVER_BIN` 门禁）→
// 适配器把本机 pty 的起停与读写接到界面的事实与写入口上 → `ui.Tester` 点按钮、
// 往提示行里打字。断言只看观察值（画布上的文本、适配器给出的终端快照），
// 不数帧、不看内部字段。
//
// 模型一律用服务端的 `--mock-model`：终端这一路不经过模型，整条用例不产生任何
// 模型调用。
//
// 入口参数：
//
//	DIM_APP_SERVER_BIN  真实 App Server 可执行文件（必需；缺失则跳过）
const (
	// termE2EDirPrefix 是临时 home 的前缀。目录要短：macOS 上 unix socket 的
	// 路径上限约 104 字节，而 App Server 的端点就在 `<home>/rust/daemon.sock`。
	termE2EDirPrefix = "dimterm-e2e-"
	// termE2EShell 是交互 shell：固定用 /bin/sh，让画布上出现什么不取决于谁的
	// 机器（本机 zsh 的 rc 会打印自己的东西）。pty 仍是真的。
	termE2EShell     = "/bin/sh"
	termE2EWait      = 10 * time.Second
	termE2EPollEvery = 25 * time.Millisecond
	// termE2ECommand 是敲进提示行的命令，termE2EOutput 是它唯一的输出行。
	termE2ECommand = "echo dim-e2e-ok"
	termE2EOutput  = "dim-e2e-ok"
	// 画布上的两句：没启动时的说明行、进程被关掉后的提示行（全角方括号）。
	termE2ENotStarted = "还没有启动终端，点「新建终端」启动本机 shell"
	termE2EClosed     = "［进程已关闭］"
	// termE2EPromptMark 是提示行的箭头：只有运行中才画。
	termE2EPromptMark = "→"
	// termE2EInputLabel 是提示行输入框的读屏名。
	termE2EInputLabel = "终端输入"
)

// termE2ESamples 是样例（-demo）里的输出片段：连着 pty 时一片都不该出现。
var termE2ESamples = []string{"$ go test", "npm run build", "(base)", "internal/ui/composer.go"}

// TestTerminalEndToEndRealPty 用真 App Server + 真 pty 走完终端的五种形状：
// 未启动 → 启动 → 敲命令 → 关进程 → 收尾。
func TestTerminalEndToEndRealPty(t *testing.T) {
	if testing.Short() {
		t.Skip("短模式跳过终端端到端（需要真实 App Server 进程与本机 pty）")
	}
	serverPath := os.Getenv("DIM_APP_SERVER_BIN")
	if serverPath == "" {
		t.Skip("缺少 DIM_APP_SERVER_BIN（真实 App Server 可执行文件）；跳过终端端到端")
	}
	if info, err := os.Stat(serverPath); err != nil || info.IsDir() {
		t.Fatalf("DIM_APP_SERVER_BIN 不可用: %v", err)
	}
	if runtime.GOOS == "windows" {
		// 本机 pty 只在类 unix 平台有实现（internal/terminal/pty_other.go 明确失败）。
		t.Skip("Windows 没有本机 pty 实现；跳过终端端到端")
	}
	// 交互 shell 固定成 /bin/sh（见 termE2EShell）：pty 是真的，rc 不是这台机器的。
	t.Setenv("SHELL", termE2EShell)

	// ① 临时 home（0700）+ 真 App Server。
	home, err := os.MkdirTemp("/tmp", termE2EDirPrefix)
	if err != nil {
		t.Fatalf("建临时 home: %v", err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(home) })
	if info, err := os.Stat(home); err != nil {
		t.Fatalf("临时 home 不可用: %v", err)
	} else if info.Mode().Perm() != 0o700 {
		t.Fatalf("临时 home 不是 0700: %v", info.Mode().Perm())
	}
	work := t.TempDir()

	base, err := app.New(app.Config{
		DataDir:      home,
		ServerPath:   serverPath,
		MockModel:    true,
		ClientName:   "dimagent-mygo-terminal-e2e",
		HistoryLimit: 20,
		SendTimeout:  30 * time.Second,
		CloseTimeout: 10 * time.Second,
		Logf:         func(format string, args ...any) { t.Logf("app: "+format, args...) },
	})
	if err != nil {
		t.Fatalf("创建集成层: %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	t.Cleanup(func() {
		closeCtx, closeCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer closeCancel()
		if err := base.Close(closeCtx); err != nil {
			t.Logf("收尾: %v", err)
		}
	})

	// ② 装配：装载本地状态 → 连接 → 建会话（工作目录是 t.TempDir()）→ 打开。
	w := uiwire.New(base)
	if err := base.LoadLocalState(ctx); err != nil {
		t.Fatalf("装载本地状态: %v", err)
	}
	if err := w.Connect(ctx); err != nil {
		t.Fatalf("连接: %v", err)
	}
	if conn := w.Connection(); conn.Phase != "connected" {
		t.Fatalf("连接阶段 = %q（%s）", conn.Phase, conn.Reason)
	}
	if err := w.CreateSession(ctx, work); err != nil {
		t.Fatalf("新建会话: %v", err)
	}
	names := w.SessionNames()
	if len(names) == 0 {
		t.Fatal("目录里没有会话")
	}
	if err := w.OpenSession(ctx, names[0]); err != nil {
		t.Fatalf("打开会话: %v", err)
	}
	snap, ok := w.Session()
	if !ok {
		t.Fatal("打开后没有会话快照")
	}
	if strings.TrimSpace(snap.Cwd) == "" {
		t.Fatalf("会话没有工作目录: %+v", snap)
	}
	t.Logf("② 已连接；会话 %q cwd=%q", snap.ID, snap.Cwd)

	// ③ 界面：Source / Intents 都接真实适配器，⌘J 打开右栏终端页。
	screen := ui.NewApp(ui.Options{
		State:   ui.StateDefault,
		Source:  w,
		Intents: w,
		Changed: func() {},
		LogPath: filepath.Join(home, "logs", "dimagent.log"),
	})
	tt := gui.NewTester(screen.View, ui.WindowWidth, ui.WindowHeight)
	tt.Frame()
	tt.Key(gui.Cmd, gui.KeyJ)
	tt.Frame()

	// ④ 初始态：说明行在、提示行不在、「新建终端」可点。
	if !tt.HasText(termE2ENotStarted) {
		t.Fatalf("④ 未启动时画布没有说明行：%s", screenText(tt))
	}
	if tt.HasText(termE2EPromptMark) {
		t.Fatalf("④ 未启动时画出了提示行：%s", screenText(tt))
	}
	start, ok := tt.Access("新建终端")
	if !ok {
		t.Fatalf("④ 找不到「新建终端」：%s", screenText(tt))
	}
	if start.Disabled {
		t.Fatalf("④ 「新建终端」是灰的：%q", start.Description)
	}
	t.Logf("④ 初始态：说明行=%q 提示行=无 新建终端=可点", termE2ENotStarted)

	// ⑤ 点「新建终端」：等到画布上是真 pty 的输出（提示符）且提示行出现。
	if err := tt.Click("新建终端"); err != nil {
		t.Fatalf("点不到「新建终端」: %v", err)
	}
	waitCanvas(t, w, tt, "终端启动后画布出现 pty 输出与提示行", func() bool {
		live := w.Terminal()
		if !live.Active || !live.Running || len(live.Lines) == 0 {
			return false
		}
		if !tt.HasText(termE2EPromptMark) || tt.HasText(termE2ENotStarted) {
			return false
		}
		// 画布要画出 pty 真打出来的那一行（提示符），不是视图自己编的内容。
		prompt := firstText(live.Lines)
		return prompt == "" || tt.HasText(prompt)
	})
	live := w.Terminal()
	prompt := firstText(live.Lines)
	if prompt == "" {
		t.Fatalf("⑤ pty 起来了但一行输出都没有：%+v", live)
	}
	t.Logf("⑤ 已启动：running=%t 行数=%d pty 提示符=%q（画布上能看到）",
		live.Running, len(live.Lines), prompt)

	// ⑥ 提示行里敲命令回车：等到画布上出现命令的输出行（整行就是输出，不是回显）。
	if err := tt.Click(termE2EInputLabel); err != nil {
		t.Fatalf("点不到提示行输入框: %v", err)
	}
	tt.Type(termE2ECommand)
	tt.Frame()
	if input, ok := tt.Access(termE2EInputLabel); !ok || input.Value != termE2ECommand {
		t.Fatalf("⑥ 命令没有进提示行: value=%q ok=%t", input.Value, ok)
	}
	tt.Key(0, gui.KeyEnter)
	waitCanvas(t, w, tt, "回车后画布出现命令输出 "+termE2EOutput, func() bool {
		_, found := tt.Find(termE2EOutput)
		return found
	})
	t.Logf("⑥ 命令已执行：画布出现整行 %q；pty 行数=%d", termE2EOutput, len(w.Terminal().Lines))

	// ⑦ 点「关闭进程」：画布换成关闭提示，提示行收起。
	if err := tt.Click("关闭进程"); err != nil {
		t.Fatalf("点不到「关闭进程」: %v", err)
	}
	waitCanvas(t, w, tt, "关闭进程后画布出现关闭提示", func() bool {
		return tt.HasText(termE2EClosed)
	})
	if tt.HasText(termE2EPromptMark) {
		t.Fatalf("⑦ 进程关了提示行还在：%s", screenText(tt))
	}
	if _, ok := tt.Access(termE2EInputLabel); ok {
		t.Fatalf("⑦ 进程关了仍能往提示行里打字：%s", screenText(tt))
	}
	if live := w.Terminal(); live.Running {
		t.Fatalf("⑦ 画布说已关闭，pty 仍在运行：%+v", live)
	}
	t.Logf("⑦ 已关闭：画布=%q 提示行=无", termE2EClosed)

	// ⑧ 全程没有样例文案：画布上是这台机器上真跑出来的东西。
	for _, sample := range termE2ESamples {
		if tt.HasText(sample) {
			t.Fatalf("⑧ 画布出现样例文案 %q：%s", sample, screenText(tt))
		}
	}
	t.Logf("⑧ 样例文案 0 处（查了 %d 条）", len(termE2ESamples))
}

// waitCanvas 等观察成立：每帧重画一次（后台意图的成功回调在画帧时生效），最多
// termE2EWait；超时算失败，并把画布最后 20 行打印出来。
func waitCanvas(t *testing.T, w *uiwire.App, tt *gui.Tester, what string, ok func() bool) {
	t.Helper()
	deadline := time.Now().Add(termE2EWait)
	for {
		tt.Frame()
		if ok() {
			return
		}
		if time.Now().After(deadline) {
			t.Fatalf("等 %s 超时（%s）\n画布最后 20 行：\n%s\n整屏文本：%s",
				what, termE2EWait, tailLines(w.Terminal().Lines, 20), screenText(tt))
		}
		time.Sleep(termE2EPollEvery)
	}
}

// tailLines 取最后 n 行，供失败时打印。
func tailLines(lines []string, n int) string {
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	if len(lines) == 0 {
		return "（空）"
	}
	shown := make([]string, 0, len(lines))
	for _, line := range lines {
		shown = append(shown, "  "+line)
	}
	return strings.Join(shown, "\n")
}

// firstText 取第一行有内容的输出（pty 的提示符通常就是它）。
func firstText(lines []string) string {
	for _, line := range lines {
		if text := strings.TrimSpace(line); text != "" {
			return text
		}
	}
	return ""
}

// screenText 把整屏文本压成一行，失败信息里看得见画的是什么。
func screenText(tt *gui.Tester) string {
	texts := tt.Texts()
	trimmed := make([]string, 0, len(texts))
	for _, text := range texts {
		if text = strings.TrimSpace(text); text != "" {
			trimmed = append(trimmed, text)
		}
	}
	return strings.Join(trimmed, " | ")
}
