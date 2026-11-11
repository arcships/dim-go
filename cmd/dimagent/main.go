// Command dimagent 是集成层的可执行入口。
//
// 本版不提供界面：只提供无界面的自检子命令，用一次完整启停验证
// 「单实例 → 装载本地状态 → 连服务 → 订阅 → 首轮历史 → 允许写 → 打开会话并打印行
// → 提交意图（经写登记）→ 收到回执 → 五阶段退出」。
//
// 用法：
//
//	dimagent selfcheck -data-dir /tmp/dimagent-home \
//	  -server /path/to/dim-app-server -mock -session <id> -lines 10
//
// 缺少真实 App Server 时命令以非零码退出并写明 blocked 的依赖；二次启动把参数
// 转交给已有实例后立即退出，不进入退出五阶段。
package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/arcships/dimagent-mygo/internal/app"
	"github.com/arcships/dimagent-mygo/internal/appserver/failure"
	"github.com/arcships/dimagent-mygo/internal/appserver/protocol"
	"github.com/arcships/dimagent-mygo/internal/catalog"
	"github.com/arcships/dimagent-mygo/internal/host"
	"github.com/arcships/dimagent-mygo/internal/interaction"
	"github.com/arcships/dimagent-mygo/internal/projection"
	"github.com/arcships/dimagent-mygo/internal/write"
)

// 退出码：0 成功；1 运行失败；2 用法错误；3 缺少环境（blocked）。
const (
	exitFailed  = 1
	exitUsage   = 2
	exitBlocked = 3
)

// defaultVersion 只用于诊断；它必须是 x.y.z 形式，外壳才会写进日志。
const defaultVersion = "0.1.0"

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(exitUsage)
	}
	switch os.Args[1] {
	case "selfcheck":
		os.Exit(runSelfCheck(os.Args[2:]))
	case "update":
		os.Exit(runUpdate(os.Args[2:]))
	case "-h", "--help", "help":
		usage()
	default:
		fmt.Fprintf(os.Stderr, "dimagent: 未知子命令 %q\n", os.Args[1])
		usage()
		os.Exit(exitUsage)
	}
}

func usage() {
	fmt.Print(`dimagent：DimAgent Go 客户端（集成层自检）

用法：
  dimagent update check|download|switch [参数]   （见 dimagent update -h）
  dimagent selfcheck [参数]

参数：
  -data-dir <目录>    客户端数据目录（必填；建议临时目录）
  -server <可执行文件> App Server 可执行文件；缺省取 DIMCODE_APP_SERVER_BIN
  -server-arg <参数>  追加给服务端的参数，可重复
  -mock               用 mock 模型启动服务端（验收用）
  -model-config <文件> 显式模型配置（configured profile；管理面方法只在非 mock 实例上可用）
  -defaults <文件>    默认插件/运行时预设（--defaults）；缺失时按能力缺失处理
  -ui-attach          在打开的会话上注册一个普通界面（ui.attach，kind=gui）
  -ui-close           注册后立即关闭（ui.close）
  -ui-status          读取界面清单（ui.status）
  -session <标识>     打开已有会话；省略时取目录里的第一条
  -create             目录为空时新建一个会话
  -permission-preset <档位> 新建会话的权限档位；缺省为空＝按服务端默认，不下发 permissionPolicy
                          需要 full-access 的证据用例必须显式传参
  -cwd <目录>         新建会话的工作目录；缺省取当前目录
  -lines <条数>       打印最近多少行（默认 10）
  -older <页数>       打开后向前翻多少页更早的轮次（默认 0）
  -wait <时长>        提交后等多久再重读视图（默认 1s）
  -submit <文本>      提交一条输入并等待写登记收敛
  -attachment <文本>  用 attachment.put 存成会话文件并等待收敛
  -shell <命令>       用 shell.run 执行一条会话内命令并等待收敛
  -shell-stop         在 -shell 之后用 shell.stop 停止该次运行
  -shell-list         读取 shell.list 并打印运行摘要（不含命令与输出）
  -rename <标题>      用 session.rename 改会话标题并打印回执
  -permission <档位>  用 session.permission.update 切换权限档位（如 full-access）
  -short              只打印汇总，不打印行
  -log <文件>         进程级诊断（宿主外壳白名单字段）写到该文件；缺省丢弃

退出码：0 成功；1 运行失败；2 用法错误；3 缺少环境（blocked）。
`)
}

// blockedError 表示缺少环境，依赖不在本仓库内。
type blockedError struct {
	what string
	need string
}

func (e *blockedError) Error() string {
	return fmt.Sprintf("blocked：缺少%s；%s", e.what, e.need)
}

// repeatedFlag 收集可重复出现的参数。
type repeatedFlag []string

func (r *repeatedFlag) String() string { return strings.Join(*r, " ") }

func (r *repeatedFlag) Set(value string) error {
	*r = append(*r, value)
	return nil
}

func runSelfCheck(args []string) int {
	var serverArgs repeatedFlag
	flags := flag.NewFlagSet("selfcheck", flag.ExitOnError)
	dataDir := flags.String("data-dir", "", "客户端数据目录")
	serverPath := flags.String("server", os.Getenv("DIMCODE_APP_SERVER_BIN"), "App Server 可执行文件")
	flags.Var(&serverArgs, "server-arg", "追加给服务端的参数，可重复")
	sessionID := flags.String("session", "", "要打开的会话；省略时取目录第一条")
	create := flags.Bool("create", false, "目录为空时新建一个会话")
	permissionPreset := flags.String("permission-preset", "", "新建会话的权限档位；空表示按服务端默认")
	cwd := flags.String("cwd", "", "新建会话的工作目录")
	lines := flags.Int("lines", 10, "打印最近多少行")
	older := flags.Int("older", 0, "打开后向前翻多少页")
	wait := flags.Duration("wait", time.Second, "提交后等多久再重读视图")
	submit := flags.String("submit", "", "提交一条输入")
	attachment := flags.String("attachment", "", "用 attachment.put 存成会话文件")
	shellCommand := flags.String("shell", "", "用 shell.run 执行一条会话内命令")
	shellStop := flags.Bool("shell-stop", false, "在 -shell 之后停止该次运行")
	shellList := flags.Bool("shell-list", false, "读取 shell.list 打印运行摘要")
	rename := flags.String("rename", "", "用 session.rename 改会话标题")
	permission := flags.String("permission", "", "用 session.permission.update 切换权限档位")
	short := flags.Bool("short", false, "只打印汇总")
	mockModel := flags.Bool("mock", false, "用 mock 模型启动服务端")
	modelConfig := flags.String("model-config", "", "显式模型配置路径")
	defaults := flags.String("defaults", "", "默认插件/运行时预设路径")
	uiAttach := flags.Bool("ui-attach", false, "注册普通界面")
	uiClose := flags.Bool("ui-close", false, "注册后立即关闭")
	uiStatus := flags.Bool("ui-status", false, "读取界面清单")
	logPath := flags.String("log", "", "进程级诊断写到该文件")
	if err := flags.Parse(args); err != nil {
		return exitUsage
	}
	if *dataDir == "" {
		fmt.Fprintln(os.Stderr, "selfcheck: 必须给出 -data-dir")
		return exitUsage
	}
	if *cwd == "" {
		dir, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "selfcheck: 取当前目录: %v\n", err)
			return exitFailed
		}
		*cwd = dir
	}

	// 进程级诊断按宿主外壳的白名单字段写；自由文本诊断不写这里。
	diag, closeDiag, err := openDiagnostics(*logPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "selfcheck: 打开诊断文件: %v\n", err)
		return exitFailed
	}
	defer closeDiag()

	base, err := app.New(app.Config{
		DataDir:          *dataDir,
		ServerPath:       *serverPath,
		ServerArgs:       serverArgs,
		MockModel:        *mockModel,
		ModelConfig:      *modelConfig,
		ServerDefaults:   *defaults,
		ClientName:       "dimagent-mygo-selfcheck",
		InitialSessionID: *sessionID,
		Version:          defaultVersion,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "selfcheck: %v\n", err)
		return exitFailed
	}
	shell, err := app.NewHost(base, app.HostOptions{
		DataDir:    *dataDir,
		Args:       os.Args[1:],
		WorkingDir: *cwd,
		Version:    defaultVersion,
		Diag:       diag,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "selfcheck: 接上宿主外壳: %v\n", err)
		return exitFailed
	}

	startCtx, cancelStart := context.WithTimeout(context.Background(), 90*time.Second)
	outcome, startErr := shell.Start(startCtx)
	cancelStart()
	if startErr != nil {
		// 启动停在失败阶段：连接观察值与分类进诊断，再按退出五阶段收尾。
		app.ObserveConnection(shell, base.Connected(), app.ClassOf(startErr))
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		report, _ := shell.Shutdown(shutdownCtx)
		cancel()
		fmt.Fprintf(os.Stderr, "selfcheck: 启动失败（阶段 %s）: %v%s\n",
			shell.State().Stage, startErr, shutdownNote(report))
		if failure.Is(startErr, failure.KindConnectionMissing) || *serverPath == "" {
			fmt.Fprintln(os.Stderr, (&blockedError{
				what: "真实的 App Server",
				need: "用 -server 指定可执行文件，或设 DIMCODE_APP_SERVER_BIN",
			}).Error())
			return exitBlocked
		}
		return exitFailed
	}
	if outcome == host.StartForwarded {
		// D10：二次启动把参数转交后直接退出，不进入退出五阶段。
		fmt.Printf("instance: forwarded（已有实例在运行，本次启动参数已转交）\n")
		return 0
	}
	app.ObserveConnection(shell, true, "")
	if state := shell.State(); state.Stage != host.StageReady {
		fmt.Fprintf(os.Stderr, "selfcheck: 启动未就绪: 阶段 %s %v\n", state.Stage, state.Failed)
		return exitFailed
	}
	fmt.Printf("startup: ready（单实例 → 本地状态 → 连服务 → 订阅 → 首轮历史 → 允许写）\n")

	code := runCheck(base, checkOptions{
		cwd:        *cwd,
		preset:     *permissionPreset,
		create:     *create,
		lines:      *lines,
		older:      *older,
		wait:       *wait,
		submit:     *submit,
		attachment: *attachment,
		shell:      *shellCommand,
		shellStop:  *shellStop,
		shellList:  *shellList,
		uiAttach:   *uiAttach,
		uiClose:    *uiClose,
		uiStatus:   *uiStatus,
		rename:     *rename,
		permission: *permission,
		short:      *short,
	})

	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 30*time.Second)
	report, shutdownErr := shell.Shutdown(shutdownCtx)
	cancelShutdown()
	if shutdownErr != nil {
		fmt.Fprintf(os.Stderr, "selfcheck: 退出未全部完成: %v\n", shutdownErr)
		if code == 0 {
			code = exitFailed
		}
	} else if !*short {
		fmt.Printf("shutdown: %d 段全部完成（%.0f ms）\n", len(report.Results), float64(report.Elapsed.Microseconds())/1000)
	}
	return code
}

// checkOptions 是自检里真正的读写步骤参数。
type checkOptions struct {
	cwd        string
	create     bool
	preset     string
	lines      int
	older      int
	wait       time.Duration
	submit     string
	attachment string
	shell      string
	shellStop  bool
	shellList  bool
	uiAttach   bool
	uiClose    bool
	uiStatus   bool
	rename     string
	permission string
	short      bool
}

// runCheck 在启动就绪后打开会话、打印行，并按需提交一条输入。
func runCheck(base *app.App, options checkOptions) int {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// 写闸门：未就绪、未连接或正在退出时一律拒绝，不把"未连接"当成空数据。
	if err := base.Writable(); err != nil {
		fmt.Fprintf(os.Stderr, "selfcheck: 写闸门拒绝: %v\n", err)
		return exitFailed
	}

	target := base.StartupSession()
	if target == "" {
		id, err := firstCatalogEntry(ctx, base, options.create, options.cwd, options.preset)
		if err != nil {
			fmt.Fprintf(os.Stderr, "selfcheck: %v\n", err)
			return exitFailed
		}
		target = id
	}

	view, err := base.OpenSession(ctx, target)
	if err != nil {
		fmt.Fprintf(os.Stderr, "selfcheck: 打开会话: %v\n", err)
		return exitFailed
	}
	for page := 0; page < options.older && view.HasOlder; page++ {
		view, err = base.LoadOlder(ctx, target)
		if err != nil {
			fmt.Fprintf(os.Stderr, "selfcheck: 向前翻页: %v\n", err)
			return exitFailed
		}
	}
	fmt.Printf("session: %s  branch: %s  load: %s  rows: %d  turns: %d  hasOlder: %t\n",
		view.SessionID, view.BranchID, view.LoadPhase, len(view.Rows), len(view.Turns), view.HasOlder)
	if !options.short {
		printRows(view, options.lines)
	}

	if options.submit != "" {
		intent, err := base.Submit(ctx, app.SubmitRequest{SessionID: view.SessionID, Text: options.submit})
		if err != nil {
			fmt.Fprintf(os.Stderr, "selfcheck: 登记提交意图: %v\n", err)
			return exitFailed
		}
		fmt.Printf("submit: intent=%s input=%s status=%s\n", intent.ID, intent.InputID, intent.Status)
		settled, err := base.Wait(ctx, intent.ID)
		fmt.Printf("submit: status=%s reason=%q input=%s result=%s\n",
			settled.Status, settled.Reason, settled.InputID, compact(string(settled.Result)))
		if err != nil {
			fmt.Fprintf(os.Stderr, "selfcheck: 等待提交收敛: %v\n", err)
			return exitFailed
		}
		if settled.Status != write.StatusSucceeded {
			fmt.Fprintf(os.Stderr, "selfcheck: 提交未成功：%s（%s）\n", settled.Status, settled.Reason)
			return exitFailed
		}
		// 提交后稍等再重读尾部：已经落库的行原地换成服务端标识。
		if options.wait > 0 {
			time.Sleep(options.wait)
		}
		updated, err := base.ReloadLatest(ctx, view.SessionID)
		if err != nil {
			fmt.Printf("after submit: 重读最近一轮失败: %v\n", err)
		} else {
			fmt.Printf("after submit: rows: %d  load: %s\n", len(updated.Rows), updated.LoadPhase)
			if !options.short {
				printRows(updated, options.lines)
			}
		}
	}

	if options.attachment != "" {
		if code := writeAttachment(ctx, base, view.SessionID, options.attachment); code != 0 {
			return code
		}
	}
	if options.shell != "" {
		beforeHistory, _ := base.HistoryKinds(ctx, view.SessionID)
		if options.shellStop {
			if code := runAndStopShell(ctx, base, view.SessionID, options.shell); code != 0 {
				return code
			}
		} else if _, code := runShellCommand(ctx, base, view.SessionID, options.shell); code != 0 {
			return code
		}
		// 证据只给行数与事实类型增量：确认命令结果写回了会话历史（record 模式）。
		if updated, err := base.ReloadLatest(ctx, view.SessionID); err == nil {
			fmt.Printf("history: rows=%d kinds=%s\n", len(updated.Rows), rowKindSummary(updated.Rows))
		}
		if afterHistory, err := base.HistoryKinds(ctx, view.SessionID); err == nil {
			fmt.Printf("history.events.added: %s\n", kindDelta(beforeHistory, afterHistory))
		}
	}
	if options.shellList {
		if code := listShellRuns(ctx, base, view.SessionID); code != 0 {
			return code
		}
	}
	if options.uiAttach || options.uiClose {
		if code := runUIAttach(ctx, base, view.SessionID, options.uiClose); code != 0 {
			return code
		}
	}
	if options.uiStatus {
		if code := printUIStatus(ctx, base); code != 0 {
			return code
		}
	}
	if options.rename != "" {
		if code := renameSession(ctx, base, view.SessionID, options.rename); code != 0 {
			return code
		}
	}
	if options.permission != "" {
		if code := switchPermission(ctx, base, view.SessionID, options.permission); code != 0 {
			return code
		}
	}

	if diagnostics := base.Diagnostics(); len(diagnostics) > 0 && !options.short {
		fmt.Println("diagnostics:")
		for _, line := range diagnostics {
			fmt.Printf("  - %s\n", line)
		}
	}
	return 0
}

// writeAttachment 用 attachment.put 存一段文本，验证附件写入链路。
func writeAttachment(ctx context.Context, base *app.App, sessionID, text string) int {
	writes := base.Writes()
	if writes == nil {
		fmt.Fprintln(os.Stderr, "selfcheck: 写登记不可用")
		return exitFailed
	}
	params, err := json.Marshal(map[string]string{"name": "selfcheck.txt", "text": text})
	if err != nil {
		fmt.Fprintf(os.Stderr, "selfcheck: 编码附件参数: %v\n", err)
		return exitFailed
	}
	registered, err := writes.Register(write.Intent{
		Key:       "selfcheck:attachment:" + sessionID,
		Kind:      write.KindAttachmentWrite,
		SessionID: sessionID,
		Params:    params,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "selfcheck: 登记附件写入: %v\n", err)
		return exitFailed
	}
	settled, err := base.Wait(ctx, registered.ID)
	fmt.Printf("attachment: status=%s reason=%q result=%s\n",
		settled.Status, settled.Reason, compact(string(settled.Result)))
	if err != nil {
		fmt.Fprintf(os.Stderr, "selfcheck: 等待附件写入收敛: %v\n", err)
		return exitFailed
	}
	if settled.Status != write.StatusSucceeded {
		fmt.Fprintf(os.Stderr, "selfcheck: 附件写入未成功：%s（%s）\n", settled.Status, settled.Reason)
		for _, line := range base.Diagnostics() {
			fmt.Fprintf(os.Stderr, "  - %s\n", line)
		}
		return exitFailed
	}
	return 0
}

// runShellCommand 用 shell.run 执行一条会话内命令，验证命令写入链路。
//
// 证据口径：只打印运行标识、状态、截断标记、耗时与退出码；命令正文与输出内容
// 一律不打印。
func runShellCommand(ctx context.Context, base *app.App, sessionID, command string) (string, int) {
	registered, err := base.RunShell(ctx, app.ShellRunRequest{SessionID: sessionID, Command: command})
	if err != nil {
		fmt.Fprintf(os.Stderr, "selfcheck: 登记命令执行: %v\n", err)
		return "", exitFailed
	}
	settled, err := base.Wait(ctx, registered.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "selfcheck: 等待命令执行收敛: %v\n", err)
		return "", exitFailed
	}
	runID := settled.InputID
	fmt.Printf("shell.run: status=%s run=%s %s\n", settled.Status, runID, summarizeShellResult(settled.Result))
	if settled.Status != write.StatusSucceeded {
		fmt.Fprintf(os.Stderr, "selfcheck: 命令执行未成功：%s（%s）\n", settled.Status, settled.Reason)
		for _, line := range base.Diagnostics() {
			fmt.Fprintf(os.Stderr, "  - %s\n", line)
		}
		return "", exitFailed
	}
	return runID, 0
}

// summarizeShellResult 只取回执里的状态字段，不回显输出。
func summarizeShellResult(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	result, err := protocol.ParseShellRunResult(raw)
	if err != nil {
		return "result=unparsable"
	}
	parts := []string{fmt.Sprintf("state=%s", result.Status)}
	if result.Truncated {
		parts = append(parts, "truncated=true")
	}
	parts = append(parts, fmt.Sprintf("durationMs=%d", result.DurationMS))
	if result.ExitCode != nil {
		parts = append(parts, fmt.Sprintf("exitCode=%d", *result.ExitCode))
	}
	if result.Signal != nil {
		parts = append(parts, "signal=set")
	}
	if result.Reason != nil {
		parts = append(parts, "reason=set")
	}
	return strings.Join(parts, " ")
}

// runAndStopShell 在运行仍进行时停止它，验证“停止后状态收敛”。
//
// 证据只给状态与标识：命令正文、输出都不打印。
func runAndStopShell(ctx context.Context, base *app.App, sessionID, command string) int {
	registered, err := base.RunShell(ctx, app.ShellRunRequest{SessionID: sessionID, Command: command})
	if err != nil {
		fmt.Fprintf(os.Stderr, "selfcheck: 登记命令执行: %v\n", err)
		return exitFailed
	}
	runID := registered.InputID
	done := make(chan write.IntentView, 1)
	go func() {
		settled, _ := base.Wait(ctx, registered.ID)
		done <- settled
	}()
	// 等这次运行出现在 shell.list 里再停止，避免在服务端登记之前就发 stop。
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		runs, listErr := base.ShellRuns(ctx, sessionID)
		if listErr == nil {
			found := false
			for _, run := range runs {
				if run.RunID == runID {
					found = true
				}
			}
			if found {
				break
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	if code := stopShellRun(ctx, base, sessionID, runID); code != 0 {
		return code
	}
	settled := <-done
	fmt.Printf("shell.run: status=%s run=%s %s\n", settled.Status, runID, summarizeShellResult(settled.Result))
	return 0
}

// kindDelta 比较两次事实类型统计，只给出新增的类型与条数。
func kindDelta(before, after map[string]int) string {
	parts := make([]string, 0, len(after))
	for kind, count := range after {
		if delta := count - before[kind]; delta > 0 {
			parts = append(parts, fmt.Sprintf("%s=+%d", kind, delta))
		}
	}
	sort.Strings(parts)
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, " ")
}

// rowKindSummary 汇总行类型与条数，不含正文。
func rowKindSummary(rows []projection.Row) string {
	counts := make(map[string]int, len(rows))
	for _, row := range rows {
		counts[string(row.Kind)]++
	}
	parts := make([]string, 0, len(counts))
	for kind, count := range counts {
		parts = append(parts, fmt.Sprintf("%s=%d", kind, count))
	}
	sort.Strings(parts)
	return strings.Join(parts, " ")
}

// stopShellRun 停止一次运行并打印是否真的停下（不含输出）。
func stopShellRun(ctx context.Context, base *app.App, sessionID, runID string) int {
	registered, err := base.StopShell(ctx, app.ShellStopRequest{SessionID: sessionID, RunID: runID})
	if err != nil {
		fmt.Fprintf(os.Stderr, "selfcheck: 登记命令停止: %v\n", err)
		return exitFailed
	}
	settled, err := base.Wait(ctx, registered.ID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "selfcheck: 等待命令停止收敛: %v\n", err)
		return exitFailed
	}
	stopped := ""
	if len(settled.Result) > 0 {
		var reply struct {
			Stopped bool `json:"stopped"`
		}
		if err := json.Unmarshal(settled.Result, &reply); err == nil {
			stopped = fmt.Sprintf(" stopped=%t", reply.Stopped)
		}
	}
	fmt.Printf("shell.stop: status=%s run=%s%s\n", settled.Status, runID, stopped)
	if settled.Status != write.StatusSucceeded {
		return exitFailed
	}
	return 0
}

// listShellRuns 打印运行摘要：标识、分支、模式与开始时间；不含命令与输出。
func listShellRuns(ctx context.Context, base *app.App, sessionID string) int {
	runs, err := base.ShellRuns(ctx, sessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "selfcheck: 读取运行清单: %v\n", err)
		for _, line := range base.Diagnostics() {
			fmt.Fprintf(os.Stderr, "  - %s\n", line)
		}
		return exitFailed
	}
	fmt.Printf("shell.list: runs=%d\n", len(runs))
	for _, run := range runs {
		fmt.Printf("  run=%s branch=%s mode=%s startedAt=%d\n",
			run.RunID, run.BranchID, run.Mode, run.StartedAt)
	}
	return 0
}

// runUIAttach 注册普通界面，可选立即关闭；证据只打印 uiId 与状态。
func runUIAttach(ctx context.Context, base *app.App, sessionID string, closeAfter bool) int {
	registration, err := base.AttachUI(ctx, sessionID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "selfcheck: 注册界面: %v（code=%s）\n", err, app.UIErrorCode(err))
		for _, line := range base.Diagnostics() {
			fmt.Fprintf(os.Stderr, "  - %s\n", line)
		}
		return exitFailed
	}
	fmt.Printf("ui.attach: uiId=%s generation=%d state=%s\n",
		registration.UIID, registration.Generation, registration.State)
	if !closeAfter {
		return 0
	}
	if err := base.CloseUI(ctx, sessionID, "selfcheck"); err != nil {
		fmt.Fprintf(os.Stderr, "selfcheck: 关闭界面: %v（code=%s）\n", err, app.UIErrorCode(err))
		return exitFailed
	}
	state := "closed"
	if registration, ok := base.UIRegistration(sessionID); ok {
		state = registration.State
	}
	fmt.Printf("ui.close: uiId=%s state=%s\n", registration.UIID, state)
	return 0
}

// printUIStatus 打印界面清单的条数与连接数；不打印界面地址。
func printUIStatus(ctx context.Context, base *app.App) int {
	status, err := base.UIStatus(ctx)
	if err != nil {
		fmt.Fprintf(os.Stderr, "selfcheck: 读取界面清单: %v（code=%s）\n", err, app.UIErrorCode(err))
		return exitFailed
	}
	fmt.Printf("ui.status: instances=%d webConnections=%d\n", len(status.Instances), status.WebConnections)
	for _, instance := range status.Instances {
		fmt.Printf("  uiId=%s session=%s kind=%s state=%s generation=%d\n",
			instance.UIID, instance.SessionID, instance.Kind, instance.State, instance.Generation)
	}
	return 0
}

// renameSession 用 session.rename 改会话标题，验证改名链路。
func renameSession(ctx context.Context, base *app.App, sessionID, title string) int {
	catalogDomain := base.Catalog()
	if catalogDomain == nil {
		fmt.Fprintln(os.Stderr, "selfcheck: 目录领域不可用")
		return exitFailed
	}
	if _, err := catalogDomain.Refresh(ctx, catalog.Query{}); err != nil {
		fmt.Fprintf(os.Stderr, "selfcheck: 读取目录（改名需要目录条目）: %v\n", err)
		return exitFailed
	}
	entry, err := catalogDomain.Rename(ctx, catalog.RenameRequest{SessionID: sessionID, Title: title})
	if err != nil {
		fmt.Fprintf(os.Stderr, "selfcheck: 改名失败: %v\n", err)
		for _, line := range base.Diagnostics() {
			fmt.Fprintf(os.Stderr, "  - %s\n", line)
		}
		return exitFailed
	}
	fmt.Printf("rename: title=%q source=%q revision=%d\n", entry.Title, entry.TitleSource, entry.TitleRevision)
	return 0
}

// switchPermission 切换权限档位并从回执读回结果。
func switchPermission(ctx context.Context, base *app.App, sessionID, preset string) int {
	if err := base.SetPermissionTier(ctx, sessionID, interaction.PermissionTier(preset)); err != nil {
		fmt.Fprintf(os.Stderr, "selfcheck: 切换权限档位: %v\n", err)
		for _, line := range base.Diagnostics() {
			fmt.Fprintf(os.Stderr, "  - %s\n", line)
		}
		return exitFailed
	}
	tier, ok := base.PermissionTier(sessionID)
	fmt.Printf("permission: preset=%q ok=%t\n", tier, ok)
	return 0
}

// firstCatalogEntry 读一页目录，返回要打开的会话；需要时先新建一个。
func firstCatalogEntry(ctx context.Context, base *app.App, create bool, cwd, preset string) (string, error) {
	catalogDomain := base.Catalog()
	if catalogDomain == nil {
		return "", app.ErrNotConnected
	}
	result, err := catalogDomain.Refresh(ctx, catalog.Query{})
	if err != nil {
		return "", fmt.Errorf("读取会话目录（服务端需要 session.catalog；缺该能力时用 -session 指定会话）: %w", err)
	}
	entries := catalogDomain.Snapshot().Entries
	if len(entries) > 0 {
		return entries[0].ID, nil
	}
	if !create {
		return "", fmt.Errorf("目录里没有会话（本次读取加入 %d 条）；用 -session 指定或加 -create 新建", result.Added)
	}
	entry, err := base.CreateSession(ctx, cwd, preset)
	if err != nil {
		return "", fmt.Errorf("新建会话: %w", err)
	}
	if entry.ID == "" {
		return "", errors.New("新建会话没有返回会话标识")
	}
	return entry.ID, nil
}

// printRows 打印最近若干行；行标识就是服务端标识，未落库的行为空。
func printRows(view app.SessionView, limit int) {
	if limit <= 0 || limit > len(view.Rows) {
		limit = len(view.Rows)
	}
	start := len(view.Rows) - limit
	for _, row := range view.Rows[start:] {
		identity := row.ID
		if identity == "" {
			identity = "-"
		}
		text := row.Text
		if row.Tool != nil && text == "" {
			text = "tool " + row.Tool.Name
		}
		text = strings.ReplaceAll(text, "\n", " ⏎ ")
		fmt.Printf("[turn %d] %-14s %-20s %s\n", row.TurnIndex, row.Kind, identity, text)
	}
	if view.Goal != nil {
		fmt.Printf("goal: %s (%s)\n", view.Goal.Objective, view.Goal.Status)
	}
	fmt.Printf("pending: inputs=%d tasks=%d questions=%d collaborations=%d  approvals=%d 待答=%d\n",
		len(view.Pending.Inputs), len(view.Pending.Tasks), len(view.Pending.Questions),
		len(view.Pending.Collaborations), view.PendingCounts.Approvals, view.PendingCounts.Questions)
}

func compact(raw string) string {
	text := strings.TrimSpace(strings.ReplaceAll(raw, "\n", " "))
	const limit = 120
	if len(text) > limit {
		return text[:limit] + "…"
	}
	return text
}

// openDiagnostics 打开进程级诊断写入端；路径为空时丢弃全部记录。
func openDiagnostics(path string) (*host.Recorder, func(), error) {
	if path == "" {
		return host.NewRecorder(io.Discard), func() {}, nil
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return nil, nil, err
	}
	return host.NewRecorder(file), func() { _ = file.Close() }, nil
}

// shutdownNote 汇总退出报告里的超时与失败阶段。
func shutdownNote(report host.ShutdownReport) string {
	var parts []string
	for _, phase := range report.TimedOut() {
		parts = append(parts, "超时 "+string(phase))
	}
	for _, phase := range report.Failed() {
		parts = append(parts, "失败 "+string(phase))
	}
	if len(parts) == 0 {
		return ""
	}
	return "（退出阶段：" + strings.Join(parts, "、") + "）"
}
