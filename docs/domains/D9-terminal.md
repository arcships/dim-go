# D9 本机终端

Status: active（第一版实现完成：启动、读写、尺寸变更、停止回收、退出状态、按会话登记）
Owner: D9 本机终端领域
依据: `docs/domains/R.md` 功能块 13 与块 13「必须改变」；`docs/domains/A.md` 的「本机终端」模块与伪终端方案附录
范围: 只做本机伪终端会话本身。不做界面渲染、命令历史、`!` 命令（由服务端执行）、审批、会话事实。

本领域不保存任何业务事实，也没有落盘状态：只持有一个正在运行的 shell 进程与它的伪终端设备。
进程一停，本领域的全部状态随之消失。

## 1 结果包

**领域**：D9 本机终端

**输入**

- `docs/domains/R.md` 功能块 13（会话内命令由服务端执行且只有一条路径；本机 shell 的启动、工作目录、停止），
  完成标志③ `pwd` 输出等于会话工作目录、④ 停止后进程列表无该进程；块 13「必须改变」指定
  「启动独立子进程并把它绑定到伪终端，工作目录取会话目录，停止时回收进程树」。
- `docs/domains/A.md`「本机终端」模块（拥有的对象：本机 shell 进程；拥有的状态：终端进程生命周期；
  调用方：用户操作与进程框架）与附录「本机终端的伪终端方案」：已批准只引一个伪终端库、只供本模块使用，
  其余模块保持零第三方依赖。
- 本领域实现约束：只用标准库加一个伪终端库；工作目录由调用方给出、不自行推测；停止必须回收整棵进程树；
  macOS 行为先行为准，其他平台至少能编译。

**产物**

| 路径 | 职责 |
|---|---|
| `internal/terminal/terminal.go` | 对外类型与生命周期：`Start`、`Options`、`ExitStatus`、`Terminal`（读写、尺寸、停止、等待）；工作目录与 shell 解析 |
| `internal/terminal/registry.go` | `Registry`：按会话登记实例，保证一个会话一个运行中的终端 |
| `internal/terminal/pty_unix.go` | darwin/linux/BSD 的伪终端实现：创建、尺寸、进程组信号、控制终端快照、残留核对 |
| `internal/terminal/pty_other.go` | 其余平台的编译桩：一律返回 `ErrUnsupportedPlatform`，不提供替代实现 |
| `internal/terminal/terminal_test.go` | 自动化验收测试 |
| `internal/terminal/smoke_test.go` | 全屏程序与中文输入的灌烟用例（默认跳过，需真实 vim/less/top） |
| `docs/domains/D9-terminal.md` | 本文件 |

**验收**

- 命令：`go test ./internal/terminal -count=1`、`go test ./internal/terminal -race -count=1`、`go vet ./...`，全部通过。
- 本领域测试（macOS 14 arm64，2026-11-10）：14 个顶层测试（13 通过、1 个灌烟用例默认跳过）与 16 个子测试全部通过；
  `go test ./...` 与 `go vet ./...` 全仓通过。
- 自动化断言覆盖：`pwd` 等于会话工作目录、停止后无残留进程、尺寸变更、UTF-8 多字节往返、退出码、
  重复启动被拒绝、停止后读写与尺寸变更的明确结果、参数与 `TERM` 传递、前置条件拒绝。
- 证据档位：本机真实 shell 与真实伪终端。全屏程序与中文输入只做过灌烟，见 §11、§12。
- 不用于宣称：界面渲染正确、输入法（IME）可用、会话内命令执行归属（属服务端与工具调用）。

**依赖**

- 上游：D3 会话（会话标识与会话工作目录，由调用方传入）；进程框架（何时启动、何时收尾，调用 `StopAll`）。
- 下游：无。本领域不被其他模块依赖，也不调用其他模块（不直接读目录模块，见 §10 的待确认项）。
- 第三方：`github.com/creack/pty v1.1.24`，只在 `pty_unix.go` 里使用；`go.mod` 增加一条直接依赖，
  `go.sum` 两条校验行；该库本身没有依赖，其他模块仍然是零第三方依赖。

**上下文需求**

- 需要组织确认 A 的模块表里「本机终端 → 调目录（取工作目录）」是否改为「调用方传入会话工作目录」。
  本版按后者实现：终端不该为拿 cwd 反向依赖目录模块。
- 需要进程框架明确：会话关闭与进程退出时调用 `Terminal.Stop` / `Registry.StopAll` 的时机，以及是否要在界面上
  提示「终端将结束」；本包默认 SIGHUP 宽限 500 ms 后强杀，不阻塞。
- 需要用户确认：已经自然退出的终端实例是否需要在登记表里保留（当前保留，便于读取退出状态，由调用方 `Forget` 移除）。

## 2 与需求、架构的对应

| 需求 / 架构条目 | 本领域责任 | 不在本领域 |
|---|---|---|
| R 块 13 完成标志③：本机 shell 的 `pwd` 等于会话工作目录 | `Start` 把会话目录作为 cwd（解析符号链接），`Dir()` 返回同一路径 | 会话目录的来源与生命周期（D3） |
| 完成标志④：停止后进程列表无该进程 | `Stop` 回收整棵进程树并核对 | 界面上的停止入口 |
| 块 13「必须改变」：独立子进程 + 伪终端 + 按会话回收 | 伪终端以 `setsid` 启动，停止按进程组与控制终端回收 | — |
| A 模块「本机终端」：拥有本机 shell 进程与终端进程生命周期 | `Terminal` 的启动/存活/停止/退出状态 | 窗口与偏好（进程框架） |
| A 的三类状态：本机终端只有进程生命周期，没有持久状态 | 不落盘、不写数据目录 | 恢复策略（无状态可恢复） |
| R 块 13 沿用项：会话内命令由服务端执行，客户端不留第二条执行路径 | 本包不执行任何会话内命令，只跑用户自己的 shell | `!` 命令（服务端） |
| 需求外但影响判据：终端里程序能正常交互 | 提供真实终端属性、窗口尺寸与字节流；全屏程序只灌烟 | 渲染、滚动、复制粘贴、输入法 |

## 3 对象与状态

### 3.1 对象

| 对象 | 业务含义 | 身份 | 构成 | 始终成立 |
|---|---|---|---|---|
| 终端实例 | 一个会话的本机 shell | 会话标识（调用方给出） | shell 路径与参数、工作目录、环境、行列数、主设备、进程号与进程组号、控制终端名、退出状态 | 一个会话最多一个运行中的实例 |
| 退出状态 | shell 结束的方式 | 属于该实例 | 退出码或终止信号 | 只产生一次，产生后不再变 |

实例只存在于内存中；进程结束即消失，进程框架不依赖它恢复任何东西。

### 3.2 状态与迁移

对应 A「终端进程生命周期」：无持久状态机，只有以下一组互斥事实。

| 当前状态 | 事件 | 下一个状态 | 本领域的表现 |
|---|---|---|---|
| 未启动 | `Start` 且前置条件满足 | 运行中 | 伪终端创建，shell 已 `exec`，`Alive()` 为真 |
| 未启动 | `Start` 且前置条件不满足 | 未启动 | 返回 `ErrEmptySessionID` / `ErrEmptyWorkingDir` / `ErrWorkingDirNotExist` / `ErrWorkingDirNotDir` / `ErrShellNotFound` / `ErrUnsupportedPlatform` |
| 运行中 | shell 自行退出（用户输入 `exit`） | 已退出 | `Wait()` 返回，`ExitStatus()` 有值，`Alive()` 为假 |
| 运行中 | 调用 `Stop` | 已退出 | 回收进程树后 `Wait()` 返回 |
| 已退出 | 调用 `Stop` | 已退出 | 幂等，只做收尾核对，不报错 |
| 已退出 | `Write` / `Resize` | 已退出 | `Write` 返回 `ErrTerminalClosed`，`Resize` 返回 `ErrTerminalClosed` |
| 已退出 | `Read` | 已退出 | 返回 `io.EOF` |

登记表（`Registry`）的状态：

| 当前状态 | 事件 | 下一个状态 | 说明 |
|---|---|---|---|
| 无实例 | `Start` | 有运行中实例 | 新建并登记 |
| 有运行中实例 | `Start` 同一会话 | 不变 | 明确拒绝：`ErrAlreadyRunning` |
| 有运行中实例 | `Stop(sessionID)` | 有已停止实例 | 停止但保留登记，供读取退出状态 |
| 有已停止实例 | `Start` 同一会话 | 有运行中实例 | 先回收旧实例残留，再替换 |
| 有已停止实例 | `Forget(sessionID)` | 无实例 | 移除登记 |
| 任意 | `StopAll` | 全部已停止 | 进程退出时的收尾入口，返回合并错误 |

**重复启动的取舍**：选「明确拒绝」而不是「返回同一实例」。理由：设备句柄与进程归属只能有一个所有者，
返回同一实例会让两个调用方都以为自己拥有停止权，一处停止另一处立刻失效且原因不明；明确拒绝让第二个调用方
必须自己决定是复用、等待还是报错。

## 4 伪终端方案与依赖例外

### 4.1 选型

按 A 附录的推荐：只供本模块使用一个伪终端库。选用 `github.com/creack/pty`，因为它只做伪终端创建、
尺寸设置与设备读写，正是附录限定的用途范围；没有传递依赖。

实际用到的 API 只有三处，都在 `pty_unix.go`：

| API | 用途 |
|---|---|
| `pty.StartWithSize(cmd, &pty.Winsize{...})` | 创建伪终端、设置初始行列数、以 `setsid` + 控制终端启动 shell |
| `pty.Setsize(f, &pty.Winsize{...})` | 运行中变更行列数 |
| `pty.Winsize` | 行列数载体 |

没有使用该库的其它能力（不做 I/O 复制、不做 inherit、不碰 `Open`）。

### 4.2 依赖记录

- `go.mod`：`require github.com/creack/pty v1.1.24`（直接依赖，无 `// indirect`）。
- `go.sum`：`github.com/creack/pty v1.1.24` 的模块与校验行各一条。
- 该库自身 `go.mod` 只声明 `go 1.18`，无其它依赖。
- 引入范围：只有 `internal/terminal` 的 unix 实现文件 import 它；`pty_other.go` 不 import，因此
  Windows 等平台的编译不经过该库。
- 理由与代价见 A 附录「引一个只供终端模块使用的伪终端库」；风险仍是库停更与平台覆盖，本版以
  macOS 实测 + 其他平台编译验证控制风险，见 §11。

## 5 启动

### 5.1 前置条件（失败即拒绝，不做兜底）

| 条件 | 失败返回 |
|---|---|
| 会话标识非空 | `ErrEmptySessionID` |
| 平台有伪终端实现 | `ErrUnsupportedPlatform` |
| 工作目录非空 | `ErrEmptyWorkingDir` |
| 工作目录存在且是目录 | `ErrWorkingDirNotExist` / `ErrWorkingDirNotDir` |
| shell 可执行 | `ErrShellNotFound` |

工作目录先 `filepath.Abs`，再 `EvalSymlinks`：shell 的 `pwd` 报物理路径，只有按同一路径登记，
「`pwd` 等于会话目录」才是可直接比对的等式。`Dir()` 返回解析后的路径；调用方不要拿原始字符串直接比较。

### 5.2 shell 解析顺序

1. `Options.Shell`（显式指定）；
2. 环境变量 `SHELL`；
3. 平台缺省：macOS 为 `/bin/zsh`，其余平台 `/bin/sh`。

解析结果再经 `exec.LookPath` 与可执行位校验；不可执行、是目录、不存在都按 `ErrShellNotFound` 拒绝。

参数由 `Options.ShellArgs` 给出，默认为空。shell 的 stdin 是伪终端，因此无需 `-i` 也会进入交互模式；
是否加载用户启动文件由调用方决定（例如传 `-f` 跳过）。本包不替调用方决定这件事。

### 5.3 环境

- `Options.Env` 为空时继承当前进程环境。
- `TERM` 缺失时补 `xterm-256color`（`Options.Term` 可改，置空字符串表示不补）。伪终端里的程序靠
  `TERM` 选择终端能力；缺了它，全屏程序会退化成行式输出。
- 工作目录通过子进程 cwd 设置，改动只影响该子进程，不影响本进程。

### 5.4 初始尺寸

`Options.Cols` / `Options.Rows`，0 表示 80×24。伪终端带真实写入尺寸，shell 与全屏程序据此排版。

### 5.5 会话与进程组

伪终端以 `setsid` 启动：shell 是新会话的首进程，同时是会话标识与进程组标识的来源，
控制终端是伪终端从设备。这两点是停止时能找全进程树的前提（见 §8）。

## 6 读写

- `Write([]byte)` 写终端输入，`Read([]byte)` 读终端输出，**都是原始字节**：不做行缓冲、不做编码转换、
  不解析控制序列、不剥 ANSI。分片边界由调用方按字节处理。
- 伪终端回显由内核终端层完成，本包不模拟、不抑制。
- UTF-8 与其它多字节输入原样往返；测试覆盖「写多字节 → 回显 → 程序读回再写出」三段。
- `Read` 在 shell 退出或终端停止后返回 `io.EOF`；`Write` 在停止后返回 `ErrTerminalClosed`，调用方据此区分
  「程序关闭了终端」与「写入本身失败」。
- **只允许一个读者**：多个 goroutine 并发读会让输出字节在调用方之间交错。`Write` 与 `Resize` 可以并发调用，
  `Stop` 可以与任意读写并发。
- 没有 scrollback、没有输入历史、没有粘贴处理，这些属于界面。

## 7 尺寸变更

`Resize(cols, rows)` 写入新的行列数并记录；内核会向终端前台进程组发 `SIGWINCH`，全屏程序据此重排。
行列数为 0 返回 `ErrInvalidSize`；终端已停止返回 `ErrTerminalClosed`。外部程序读到的值必须等于最近一次成功的
`Resize`（测试用 `stty size` 交叉验证）。

## 8 停止与进程树回收

停止的目标不是「杀掉 shell」，而是「这个终端里的进程一个都不剩」。顺序固定：

1. **取快照**：在 shell 还活着时，按控制终端名（如 `ttys009`）列出该终端下的全部进程。
   shell 一退出，这些进程的控制终端关联就消失（`ps` 显示 `??`），事后再找就找不全，所以快照必须最先做。
2. **先礼后兵**：向 shell 的进程组发 `SIGHUP`，等待宽限期（默认 500 ms，`Options.GracePeriod` 可调）。
   交互 shell 通常在此结束，并把自己的作业一起带走。
3. **强杀进程组**：宽限期内没退出，向进程组发 `SIGKILL`（`kill(-pgid, …)`，不是 `kill(pid, …)`）。
   此时 shell 仍在，补一次快照并入前面的列表。
4. **强杀快照内进程**：逐个 `SIGKILL`。这一步覆盖两类只杀父进程或父进程组都漏掉的情况——
   作业控制把每个作业放进自己的进程组，以及后台作业已经脱离父进程。
5. **关闭主设备**：终端挂断，前台作业收到挂断信号，阻塞中的 `Read` 返回。
6. **等待与核对**：等 shell 退出（上限 2 s）；再核对快照里的进程是否都已消失（上限 3 s）。
   仍在的用 `ErrResidualProcesses` 如实上报，不假装成功。

失败时 `Stop` 返回错误并合并全部原因（`errors.Join`）：停止超时、残留进程、快照枚举失败。
已停止的终端再次 `Stop` 不重复回收、不报错。

为什么按控制终端找而不是按会话或进程组（macOS 实测记录）：

- 进程组不够：交互 shell 打开作业控制后，每个作业自成一个进程组；后台作业那组不在 shell 的组里。
- 会话够用但 macOS 拿不到：macOS 的 `ps` 没有会话列（`-o sess` 恒输出 0），无从枚举会话成员。
- 控制终端正好是会话的标记：同一会话的进程共享同一个控制终端，而一个终端同时只能属于一个会话；
  shell 存活期间取快照，就能把这批进程（含已脱离父进程的）找全。

## 9 生命周期与登记表

- 一个会话一个终端实例；重复启动同一会话返回 `ErrAlreadyRunning`（理由见 §3.2）。
- 已停止的实例保留在登记表里，供调用方读取 `ExitStatus()`；不需要时 `Forget` 移除。
  重新 `Start` 同一会话时，若旧实例已退出，会先回收其残留再替换。
- `StopAll` 用于进程框架退出时统一收尾，返回合并后的错误；单个失败不影响其余实例回收。
- 本领域不落盘、不写应用数据目录；没有重启恢复问题，进程重启后终端本来就应该重新启动。

## 10 Go 接口

```go
// 启动与登记
func Start(sessionID, dir string, opts Options) (*Terminal, error)
func NewRegistry() *Registry
func (r *Registry) Start(sessionID, dir string, opts Options) (*Terminal, error)
func (r *Registry) Get(sessionID string) (*Terminal, bool)
func (r *Registry) Stop(sessionID string) error
func (r *Registry) StopAll() error
func (r *Registry) Forget(sessionID string) error
func (r *Registry) Len() int

// 实例
func (t *Terminal) SessionID() string
func (t *Terminal) Dir() string
func (t *Terminal) Shell() string
func (t *Terminal) Args() []string
func (t *Terminal) PID() int
func (t *Terminal) TTY() string
func (t *Terminal) Size() (cols, rows uint16)
func (t *Terminal) Alive() bool
func (t *Terminal) Read(p []byte) (int, error)
func (t *Terminal) Write(p []byte) (int, error)
func (t *Terminal) Resize(cols, rows uint16) error
func (t *Terminal) ExitStatus() (ExitStatus, bool)
func (t *Terminal) Wait() ExitStatus
func (t *Terminal) Stop() error
```

`Options` 字段：`Shell`、`ShellArgs`、`Env`、`Cols`、`Rows`、`Term`、`GracePeriod`。
错误值：`ErrUnsupportedPlatform`、`ErrEmptySessionID`、`ErrEmptyWorkingDir`、`ErrWorkingDirNotExist`、
`ErrWorkingDirNotDir`、`ErrShellNotFound`、`ErrInvalidSize`、`ErrTerminalClosed`、`ErrAlreadyRunning`、
`ErrNoTerminal`、`ErrStopTimeout`、`ErrResidualProcesses`。

## 11 验收证据

### 11.1 命令与结果

```bash
$ go test ./internal/terminal -count=1
ok  	github.com/arcships/dimagent-mygo/internal/terminal

$ go test ./internal/terminal -race -count=1
ok  	github.com/arcships/dimagent-mygo/internal/terminal

$ go test ./internal/terminal -count=1 -v     # 14 个顶层（13 通过 + 1 跳过），16 个子测试通过
$ go vet ./...                                # 通过
$ go test ./...                               # 全仓通过
```

### 11.2 判据与用例对应

| 验收判据 | 用例 | 怎么断言 |
|---|---|---|
| `pwd` 输出等于会话目录 | `TestPwdEqualsSessionDir`（普通目录、含空格与中文两个子用例） | 写 `pwd`，等输出中出现与 `Dir()` 完全相同的一整行 |
| 停止后无残留进程 | `TestStopReapsProcessGroupTree` | 起一个后台作业、一个已脱离父进程的后台作业、一个前台作业；停止前核对三者都在控制终端快照里；停止后三个进程号都不存活，且 `Stop` 返回 nil |
| 尺寸变更不崩且生效 | `TestResizeWindowSize` | 初始 100×30 用 `stty size` 验证；连续变更到 120×40、20×5、400×200、132×43，再次 `stty size` 得到 `43 132`，之后 shell 仍可执行命令；0 行列数返回 `ErrInvalidSize` |
| UTF-8 多字节往返 | `TestUTF8MultibyteRoundTrip` | 输出方向用 `printf '中文-%s\n' OK`；输入方向把「汉字」写进 `head -c 6`，输出里出现两次且没有替换字符 |
| 退出码 | `TestExitStatusForExitCommand` | `exit 7` 后 `Wait()` 得到 `Code=7`、未被告警为信号终止 |
| 停止后的接口行为与幂等 | `TestStopUnblocksAndIsIdempotent` | 停止后 `Write`/`Resize` 返回 `ErrTerminalClosed`；并发重复 `Stop` 全部返回 nil |
| 一个会话一个终端 | `TestRegistryRejectsDuplicateStart` | 重复启动返回 `ErrAlreadyRunning`；停止后允许重启；`Get` 能取回已停止实例；未知会话 `Stop` 返回 `ErrNoTerminal`；`Forget` 后数量减一 |
| 前置条件拒绝 | `TestStartRejectsInvalidInput` | 空会话标识、空目录、不存在目录、文件当目录、shell 不存在，各自返回对应错误 |
| 参数与 `TERM` 传递 | `TestShellArgsArePassed`、`TestDefaultTermIsInjected` | `-c` 参数被执行；`$TERM` 等于 `xterm-256color` |
| shell 解析顺序 | `TestResolveShell`、`TestBuildEnv` | 表驱动覆盖显式、环境变量、缺省、不可执行、不存在；`TERM` 保留与注入 |

### 11.3 灌烟（不计入自动化断言）

`TestSmokeFullScreenPrograms` 默认跳过，设置 `DIMAGENT_TERMINAL_SMOKE=1` 才运行：

```bash
DIMAGENT_TERMINAL_SMOKE=1 go test ./internal/terminal -run TestSmokeFullScreenPrograms -v
```

2026-11-10 在 macOS 14 arm64 上通过：`vim` 进入、运行中把窗口改到 120×40、`:q!` 退出；`less`、`top`
进入与退出；`cat` 读到中文输入后 `Ctrl-D` 结束；每一步之后 shell 都能继续执行命令，输出不含替换字符。
渲染是否正确、光标位置、颜色、鼠标、输入法候选框都没有判据，只能人眼确认。

**稳定性措施**（并联跑包时曾经偶发超时，原因是用固定 `sleep` 当作「程序已就绪」）：

- **只等标记，不等时间**：每一步先记下已读字节数的基线，写入命令后等基线之后的新输出里出现该步的
  就绪标记，再做下一步。就绪标记取程序自己画出的内容，不取回显：`vim`/`less` 等文件内容 `中文内容`，
  `top` 等列头 `PID`，`cat` 等命令行回显（此时该行已被 shell 读完，后续字节必进终端输入队列）。
  退出也用标记确认：每条命令都写成 `<程序>; printf 'XXX-%s\n' OUT`，等到 `XXX-OUT` 才认定回到 shell。
- **严格上限**：每一步最多等 2 秒（`smokeTimeout`），超时立刻失败，不再用「睡够就好」掩盖问题。
- **失败可诊断**：超时信息带出该步已读到的字节片段（末尾 800 字节，`%q` 转义），能区分「程序没起来」
  与「断言太早」。为了不把超时埋在别处，用例只在自己内部等标记，不再复用生产用例的 8 秒等待。
- **不依赖窗口时序**：尺寸变更在 `vim` 确认画出内容之后才发生，不靠「启动约 1.5 秒」这类经验值。
- **验证**：`-count=5` 单包连跑通过；`DIMAGENT_TERMINAL_SMOKE=1 go test ./... -count=5` 全仓并联通过
  （本包 5 轮共 21 秒）；另外在 8 个 CPU 占满进程下连跑 2 次仍通过，说明 2 秒上限在明显负载下仍有余量。

**平台差异**（灌烟用例依赖的具体标记）：

| 观察点 | macOS（已实测） | 其他平台 |
|---|---|---|
| `vim` 是否已进入 | 文件内容出现在屏幕上（不依赖备用屏幕序列 `?1049h`，用户改过 `.vimrc` 也不受影响） | 同左；Linux 未实测 |
| `less` 是否已进入 | 文件内容出现 | 同左；若 `less` 未安装则失败 |
| `top` 是否已画出第一屏 | 列头 `PID`（macOS 首行是 `Processes:`） | Linux 首行是 `Tasks:`，但列头同样有 `PID`，判据可复用；未实测 |
| 中文输入 | `cat` 读到 6 字节后 `Ctrl-D` 结束，输出不出现替换字符 | 同左；输入法相关行为不覆盖 |
| 未覆盖 | 颜色、光标、鼠标、粘贴、输入法候选框、`TERM` 能力协商 | 同左 |

## 12 跨平台情况

| 平台 | 状态 | 证据 |
|---|---|---|
| darwin/arm64（macOS 14） | 实现并验证 | 全部自动化测试与灌烟用例在本机运行通过 |
| linux/amd64、linux/arm64 | 编译通过，未运行测试 | `GOOS=linux GOARCH=amd64 go build ./internal/terminal`、`go vet` 通过；依赖 `ps -o tty=,stat=` 与 `syscall.Kill(-pgid)`，procps 与 Linux 均支持，但未实测 |
| freebsd/openbsd/netbsd/dragonfly amd64 | 编译通过，未运行测试 | 同上，伪终端库支持这些平台 |
| windows/amd64 | 编译通过，功能明确不支持 | `pty_other.go`：`Start` 返回 `ErrUnsupportedPlatform`；不引 `creack/pty`（该库 Windows 实现也只是返回不支持） |
| solaris/plan9/aix 等 | 编译通过，功能明确不支持 | 同 `pty_other.go`；solaris 虽被伪终端库部分支持，本版不承诺，保持明确失败 |

平台隔离方式：`pty_unix.go` 与 `pty_other.go` 用 build tag 分开，共享代码只依赖 `os.File`、`exec.Cmd`
与 `int` 型状态，不出现平台专有类型。

## 13 已知限制与未验证项

1. **全屏程序与中文输入未验证渲染**：只有灌烟结果（vim/less/top 能进能退、尺寸变更不崩、中文输入不出现替换字符）。
   光标与颜色、鼠标事件、粘贴、输入法候选框、`TERM` 能力协商都没有自动判据。
2. **停止的清扫边界**：按「控制终端快照」回收。若进程在快照之后才启动并立刻 `setsid` 自建会话
   （真正的守护进程化），它既不在快照里也不在进程组里，回收不到。这类进程语义上已脱离终端，本版不追。
3. **shell 自然退出后的残留**：用户自己 `exit` 之后快照已无法取得（控制终端关联消失），只能依赖 shell 自身的
   作业回收。主设备此时仍开着，终端名不会被别的会话复用，因此不会误伤无关进程；`Stop` 仍可调用，只做收尾。
4. **停止不做保存协商**：`Stop` 先 `SIGHUP` 再 `SIGKILL`，不给程序「保存后退出」的交互回合。这与关闭终端面板的语义一致，
   但意味着前台编辑中的全屏程序可能丢未保存内容。
5. **固定超时**：强杀后等待 2 s、残留核对 3 s 是常量，只有宽限期可通过 `Options.GracePeriod` 调整。
6. **一个读者**：并发读会交错，接口上只在注释里约定，没有运行时检查。
7. **工作目录解析符号链接**：`Dir()` 是物理路径；调用方拿原始字符串比较会不相等（例如 macOS 的 `/var` 与 `/private/var`）。
8. **`TERM` 缺省值**：`xterm-256color` 是折中，不读取用户终端环境；需要别的值就显式传 `Options.Term` 或 `Env`。
9. **不做的事**：界面渲染、滚动缓冲、命令历史、复制粘贴、输入法集成、`!` 命令（服务端执行）、会话内命令执行归属。
10. **进程识别依赖 `ps`**：`ttyMembers`/`alivePIDs`/`processTTY` 调用系统 `ps`，输出格式按 macOS 与 procps 的列式输出解析；
    没有 `ps` 的极简系统上停止只能退化为进程组回收（快照为空，不报错）。
11. **Windows 无实现**：明确报不支持，不提供管道降级方案（那会变成「看着能用、很多程序跑不起来」的假能力）。
12. **灌烟用例依赖本机程序与固定上限**：需要真实的 `vim`、`less`、`top`；每步上限 2 秒，极慢机器上可能超时。
    该用例默认跳过，`go test ./...` 不依赖它；超时会打印该步已读片段，便于区分环境缺失与断言过早。

## 14 变更记录

| 日期 | 变更 |
|---|---|
| 2026-11-10 | 第一版：启动/读写/尺寸/停止/退出状态/按会话登记；引入 `github.com/creack/pty v1.1.24`；macOS 实测，其余平台编译验证 |
| 2026-11-10 | 灌烟用例改为标记同步：去掉固定 `sleep`，每步 2 秒严格上限，超时输出已读片段；连续 5 轮与全仓并联各 5 轮通过 |
