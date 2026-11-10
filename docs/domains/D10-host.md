# D10 宿主外壳（单实例、启动编排、退出顺序、诊断、进程级状态、应用更新）

Status: 第一版实现完成（P1-3 候选结果包；P3 应用更新已补齐）
Owner: D10 领域 agent
验证: 可执行命令 + 可观察结果见第 6 节
依据: `docs/domains/R.md` 第 1 部分功能块 14（进程级行为）；`docs/domains/A.md` 的「启动与退出」迁移表、应用级状态表（单实例事实、启动阶段、窗口与偏好、更新状态）与四条总规矩；`docs/project-organization.md` §4.2 D10、§6 F3
范围: 单实例与二次启动接管、启动六阶段编排与失败收敛、退出五阶段与逐段超时、诊断记录（字段白名单）、进程级状态暴露、应用更新的检查与切换（P3）。不含界面、连接实现（D1）、各领域内部逻辑；窗口/菜单/托盘/通知/快捷键/深链/主题字体/i18n 属界面阶段；更新只做检查、下载校验与切换编排，不做签名工具链与渠道发布。

---

## 1. 输入

| 输入 | 来源 | 本版使用方式 |
|---|---|---|
| 进程级行为要求 | `R.md` 功能块 14 及其完成标志 | 二次启动进程数仍为 1、退出先交互后回收、日志含连接事件与版本且不含正文与凭据 |
| 启动与退出状态机 | `A.md`「启动与退出」「连接」迁移表 | 六阶段启动、五阶段退出；非法推进丢弃并写诊断 |
| 单实例与诊断归属 | `A.md` 四、应用级客户端状态；五、组织表 | 单实例事实与启动阶段归进程框架；诊断只写不读，不被任何模块依赖 |
| 连接领域接口（F3 候选） | `docs/domains/D1-connection.md` §5 | 只作为外部步骤接入：`Connect` / `OpenStream` / `Shutdown` / `failure.KindOf`；本包不 import 连接包、不重复 wire 定义 |
| 错误分类白名单 | D1 的五种错误分类 | 通过 `ErrorClass` 与 `Classified` 接口收敛，未知一律记 `other` |
| 更新状态机 | `A.md` 的「更新」迁移表；`R.md` 功能块 14 完成标志⑤、R9 | 五条迁移逐条实现；检查失败按已是最新呈现；下载/校验失败保留当前版本 |
| 更新源形状（只读参考） | dim-electron `electron-main/src/services/update-service.ts`、`src/config/distribution.ts`、`~/.dimcode-alpha/config.json` | 基址 `https://dimcode.echooai.com`、通道 `updates/stable`、平台路径 `mac/<arch>`/`win/x64`/`linux/x64`、清单 `latest*.yml`（electron-builder 通用）；只复制读取，不改参考仓库 |
| 本机证据环境 | macOS arm64、Go 1.24 | 端点互斥、强杀残留、超时、更新检查与切换均在本机实测（更新源用本地 HTTP 服务模拟） |

## 2. 产物

### 2.1 包结构

| 文件 | 职责 |
|---|---|
| `internal/host/doc.go` | 包边界与四条硬约定 |
| `internal/host/host.go` | `Shell`：把单实例、启动阶段、退出顺序、诊断与状态编排到一起；`Config`/`StartupSteps`/`StartOutcome` |
| `internal/host/instance.go` | 单实例端点：`TryAcquire`、`Forward`、`Instance`、接管队列与报文 |
| `internal/host/endpoint_unix.go` | Unix 端点：临时名监听 + 硬链接原子发布 + 残留清理 |
| `internal/host/endpoint_windows.go` | Windows 端点：回环 TCP + 端口文件（编译通过，未在 Windows 验证） |
| `internal/host/stage.go` | `Stage` 六阶段、`Store` 进程级状态、写闸门 `Writable` |
| `internal/host/startup.go` | 启动编排：顺序校验、失败停住、不允许跳过阶段 |
| `internal/host/shutdown.go` | 退出编排：五阶段、逐段超时、超时后继续、幂等 |
| `internal/host/diag.go` | 诊断记录：字段白名单、取值形状、红acted、并发写入、冲刷 |
| `internal/host/update.go` | 更新状态机与编排：检查、下载校验、待切换到切换；超时与错误分类 |
| `internal/host/update_manifest.go` | 版本源地址推导、electron-builder 清单的最小 YAML 解析、摘要形状校验 |
| `internal/host/update_version.go` | semver 解析与比较（含预发布段） |
| `internal/host/update_install.go` | 单文件产物的原子替换 |

测试：`diag_test.go`、`stage_test.go`、`startup_test.go`、`shutdown_test.go`、`instance_test.go`、`instance_unix_test.go`、`host_test.go`、`update_test.go`、`update_manifest_test.go`、`update_version_test.go`、`helpers_test.go`。

### 2.2 关键约定

- **只服务进程**：本包不读会话、不存事实、不发协议请求；它只决定什么时候允许别人发。
- **单实例即端点互斥**：端点上出现套接字就代表一个可以连上的实例。二次启动把参数转交后立即退出；不写锁文件、不读状态文件。
- **阶段只能按序完成**：六个启动阶段的顺序在构造时校验，缺一个或顺序不对直接报错，不会静默跳过；失败停在失败阶段并如实暴露。
- **写闸门三条同时成立才放行**：阶段为 `ready`、没有连接探针说未连接、未开始退出。
- **退出不无限等**：五段各有超时，任一段超时或失败都记录后继续；父上下文取消也不跳过收尾。
- **诊断没有自由文本入口**：字段只能由本包构造函数生成，字符串取值只有枚举与 `x.y.z` 版本号；凭据与会话正文没有可写入的位置。

## 3. 设计

### 3.1 单实例与二次启动接管

**端点位置**：未显式给端点时取 `<数据目录>/host/instance.sock`（Unix）。父目录 0700，套接字 0600。Windows 取 `<数据目录>/host/instance.port`。

**抢占（Unix）**：

1. 在端点同目录用临时名 `.<端点名>.<pid>.<序号>` 先 `listen`，此时套接字已可接受连接。
2. 用 `os.Link(临时名, 端点)` 原子发布：链接成功即抢到；失败 `EEXIST` 说明端点已被占。
3. 端点已存在时先拨号（最多 3 次、每次间隔 5ms）：能连上说明是活实例，返回 `ErrInstanceRunning`；连不上才当作强杀残留，删除后重试。
4. 抢到后删除临时名并清理同目录其他残留临时套接字。

这样端点路径一旦存在就一定可连接，"能连即活、连不上即残留"的判定没有 bind 与 listen 之间的空窗；并发启动时只有一个进程能链接成功。

**转交**：二次启动向端点写一行 JSON `{"action":"activate","args":[...],"working_dir":"..."}` 并等一行回执 `{"ok":true,"delivered":true}`；主实例把参数放进接管队列（默认 8 条，满了丢弃并只记参数个数）。转交成功返回 `StartForwarded`，调用方直接退出。转交失败会再抢一次端点（已有实例可能刚好退出），仍失败则返回 `ErrForwardFailed`。

**强杀后的收敛**：进程被 `SIGKILL` 后端点套接字留在磁盘上，没有监听者。下次启动按上面的"拨号失败即残留"清理并重新占用；除此之外不留下任何锁文件或状态文件。若端点位置被普通文件占用，本包拒绝删除并如实报错。

**Windows**：没有可原子抢占的本地套接字，改用回环 TCP 并把端口写进端点文件；先探测、再清理、再监听。两个进程同时启动存在竞态窗口，本阶段未在 Windows 验证，见第 8 节。

### 3.2 启动阶段与写闸门

| 阶段 | 名字 | 完成含义 | 未完成时 `Writable()` |
|---|---|---|---|
| 1 | `single_instance` | 已抢到端点（转交成功时本进程不推进阶段、直接退出，状态停在 `idle`） | `ErrNotReady` |
| 2 | `local_state` | 本地状态装载完成（失败按空处理由实现方决定） | `ErrNotReady` |
| 3 | `connect` | 握手完成 | `ErrNotReady` |
| 4 | `subscribe` | 订阅已建立 | `ErrNotReady` |
| 5 | `first_history` | 首轮历史已到达 | `ErrNotReady` |
| 6 | `allow_writes` | 允许写 | `ErrNotReady` |
| 终态 | `ready` | 六阶段全部完成 | 探针已连接时放行 |

- 六阶段顺序固定；`NewStartup` 要求阶段列表正好覆盖当前阶段之后的全部阶段，缺项、错序、空实现都返回 `ErrStageUnwired`。
- 任一阶段返回错误：`Store.Fail` 把当前阶段置为该阶段并保留错误，写 `startup.failed`（阶段、错误分类、耗时），后续阶段不执行。
- 阶段完成写 `startup.stage`（阶段、结果、距上一阶段的毫秒数）；六阶段完成后再推进到 `ready`，`Shell` 追加一条 `startup.ready`（含版本号）。
- 非法推进（跳跃、失败后继续、退出后推进）不改状态，写 `startup.stage_rejected`（起始阶段、目标阶段、结果 `skipped`），返回 `ErrInvalidStage`。
- 二次启动转交成功后本进程不推进任何阶段，状态停在 `idle`：它的职责只是把参数送出去然后退出。

**连接状态**：由连接领域持有，本包只读。`Config.Connected` 是一个只读探针；`Store.ObserveConnection` 只写一条 `connection.changed` 诊断，不改状态。断线后写闸门返回 `ErrNotConnected`，不会把"未连接"当成"没有数据"继续发请求。

### 3.3 退出顺序与超时

| 顺序 | 阶段 | 默认超时 | 典型接线 |
|---|---|---|---|
| 1 | `stop_intents` | 1s | 写登记拒绝新意图，交互裁决不再接收新应答 |
| 2 | `end_interaction` | 3s | 结束打开的会话与交互（与"停止后 3 秒内非运行"对齐） |
| 3 | `close_connection` | 2s | 关闭会话流与控制连接 |
| 4 | `reclaim_service` | 5s | 只回收自己拉起的服务；外部实例不动 |
| 5 | `flush_diagnostics` | 1s | 冲刷诊断（未接线时用记录自身的 `Flush`） |

- 进入第一阶段之前 `Store.BeginExit()`：从这一刻起 `Writable()` 一律 `ErrExiting`，新意图被拒。
- 每段在独立 goroutine 里执行，`select` 等待完成或本段超时；超时写 `shutdown.timeout`（阶段、已耗时、超时值）后进入下一段。实现方不响应上下文也不会拖住退出。
- 某段返回错误同样记录 `shutdown.phase`（结果 `failed`）后继续；五段走完后写 `shutdown.done`，再冲刷一次诊断，保证收尾记录落盘。
- 报告 `ShutdownReport` 给出每段结果、耗时、超时/失败清单与合并错误；`Run` 重复或并发调用都只执行一次并返回同一报告。
- 五段之外的端点释放由 `Shell.Close()` 完成（退出五段走完后），不属于五段。

### 3.4 诊断字段白名单

记录头固定三个键：`time`、`level`（info/warn/error）、`event`（形状校验，最多 64 字符）。

可写字段只有这些：

| 字段 | 取值形状 | 说明 |
|---|---|---|
| `stage` / `from_stage` / `next_stage` | 六个启动阶段 + `idle` + `ready` | 阶段推进与拒绝 |
| `phase` | 五个退出阶段 | 退出进度 |
| `outcome` | `ok` / `timeout` / `failed` / `skipped` | 阶段结果 |
| `classification` | `version_incompatible` / `connection_missing` / `timeout` / `retryable` / `non_retryable` / `other` | 错误分类，未知值收敛为 `other` |
| `connected` / `exiting` | 布尔 | 连接观察与退出标记 |
| `version` | `x.y.z[-pre][+build]` | 客户端版本；不合形状写 `[redacted]` |
| `elapsed_ms` / `timeout_ms` | 非负整数 | 耗时与超时 |
| `args` | 非负整数 | 只记参数个数，不记内容 |
| `dropped` | 非负整数 | 本条记录被丢弃的字段个数 |
| `update_state` | `unchecked` / `checking` / `available` / `up_to_date` / `ready_to_switch` / `switching` | 更新状态机当前状态 |
| `update_version` | `x.y.z[-pre]` | 版本源给出的新版本；不合形状写 `[redacted]` |

- 字段只能用 `StageField`、`ClassField`、`VersionField` 等构造函数生成；`Field` 的字段不导出，外部无法拼出"任意键 + 任意文本"。
- 事件名形状不合法时收敛为 `event.invalid`；字段形状不合法（未知阶段、未知分类、零值字段）被丢弃并只累加 `dropped`。
- 因此凭据与会话正文没有入口：没有自由文本字段，唯一接受字符串的 `version` 只接受版本形状，`classification` 与阶段都是枚举。`TestRecorderRejectsCredentialsAndBodies` 用真实形态的凭据串与正文验证输出里不含原文。

示例（实测字段集合，时间为示例值）：

```json
{"time":"2026-11-10T12:00:00.123456Z","level":"info","event":"startup.stage","stage":"connect","outcome":"ok","elapsed_ms":37}
{"time":"2026-11-10T12:00:00.200000Z","level":"warn","event":"shutdown.timeout","phase":"end_interaction","elapsed_ms":3000,"timeout_ms":3000}
```

### 3.5 进程级状态

`Store.Snapshot()` 一次给出五项，全部只读：

| 字段 | 含义 |
|---|---|
| `Stage` | 当前阶段（已完成的最后一个阶段，或停在的失败阶段） |
| `Failed` | 导致启动停在当前阶段的错误 |
| `Connected` | 连接领域探针的当前值（本包不写） |
| `Exiting` | 退出已开始 |
| `Exited` | 退出五段已走完 |

`Store` 是启动阶段、失败、退出标记的唯一驱动者；连接状态仍是连接领域的唯一驱动者，本包只读。

### 3.6 并发与幂等

- 诊断写入持锁，一行一次写入，多协程不会串行；`Flush` 依次调用底层 `Flush` 与 `Sync`。
- 启动编排只能运行一次；退出编排 `sync.Once`，并发调用共享同一报告。
- 接管队列满时丢弃新参数并写 `activation.dropped`，不阻塞主实例；主实例关闭后队列关闭。
- `Instance.Close`、`Shell.Close` 幂等。

### 3.7 应用更新（P3）

状态机与 `A.md` 的「更新」迁移表逐条对应：

| 当前状态 | 事件 | 下一个状态 | 实现 |
|---|---|---|---|
| 未检查 | 检查版本 | 检查中 | `Check`：置检查中后读取版本清单 |
| 检查中 | 有新版 | 待下载 | 清单版本高于当前版本 |
| 检查中 | 已是最新或检查失败 | 已是最新 | 网络、HTTP、清单、版本比较失败都进这里；写诊断并保留 `LastError` |
| 待下载 | 下载并校验完成 | 待切换 | `Download`：落临时目录、比对大小与 sha512 |
| 待切换 | 用户确认且会话已收尾 | 切换中 | `Switch`：先探测/收尾会话，再退出、替换、重启 |

- 非法迁移（未检查就下载、就绪前切换、切换后重复检查等）一律丢弃、写 `update.transition_rejected`（`update_state` + `skipped`），状态不变、不向用户报错。
- 下载或校验失败不迁移状态：停在待下载，写 `update.download`（`failed` + 分类），当前版本不变，可重新下载。会话未收尾时停在待切换，写 `update.switch_deferred`，不替换、不重启。
- 同一进程只检查一次（迁移表没有从已是最新回到检查中的边）；重启后回到未检查。
- 版本源沿用现有客户端形状：`<基址>/<通道>/<平台路径>/latest*.yml`，默认基址 `https://dimcode.echooai.com`、通道 `updates/stable`，平台路径 `mac/<arch>`、`win/x64`、`linux/x64`。清单解析 electron-builder 的 `version`、`files[].url/sha512/size` 与顶层 `path/sha512/size`；相对产物地址按清单地址解析。
- 校验只有 sha512（base64 或十六进制），缺失摘要的清单视为失败；不做签名校验。
- 切换顺序：会话收尾探测 → 可选退出五阶段 → 单文件原子替换（同目录临时文件 + `rename`）→ 重启步骤。任何一步失败都写 `update.switch`（`failed` + 分类）并保留当前版本；替换失败不会留下半截文件。
- 默认超时：检查 15 秒、下载 10 分钟、收尾 30 秒、切换 5 分钟；均可由 `UpdateTimeouts` 覆盖。
- 切换一旦开始就是终态：退出、替换或重启任一步失败都保留当前版本，但本进程不再重试，重启后回到未检查。
- 更新只写诊断、不读会话事实，也不驱动连接；界面阶段只消费 `Updater.Status()`。

## 4. 状态映射

| `A.md` 状态机 / 状态 | 本领域实现 |
|---|---|
| 初始 → 单实例判定通过 → 装载本地状态 → 连服务 → 订阅 → 可用 | `Stage` 六阶段 + `ready`；`Startup` 按序推进，`Store` 记录 |
| 启动阶段「停在当前阶段并如实提示，不假装可用」 | `Store.Fail` 保留失败阶段与错误；写闸门保持拒绝 |
| 单实例事实「已有实例则转交参数后本进程退出」 | `TryAcquire` → `Forward` → `StartOutcome=StartForwarded` |
| 可用 → 退出指令 → 收尾交互（有超时） → 断连接 → 回收自启服务（有超时） → 结束 | `Shutdown` 五阶段（多出"停止新意图"与"冲刷诊断"，见下） |
| 非法迁移统一处理：丢弃、写诊断、不改状态、不向用户报错 | `Store.Advance` → `EventStageRejected` + `ErrInvalidStage` |
| 窗口与偏好（归进程框架） | 本版只留"装载本地状态"阶段位，未实现偏好存储（第 8 节） |
| 诊断记录（只写、不被依赖） | `Recorder`：白名单字段、无自由文本 |
| 更新状态：未检查 → 检查中 → 待下载 → 待切换 → 切换中 | `Updater`；检查失败进已是最新；下载/校验失败留在待下载；会话未收尾时切换推迟 |

退出五阶段相对 `A.md` 的"收尾交互 → 断连接 → 回收自启服务"多了前后两段：前段"停止新意图"落实"退出中收到新意图一律拒绝"，后段"冲刷诊断"落实"退出后日志可交付"；`A.md` 的"结束"对应 `ShutdownReport` 与 `Exited`。

## 5. 接口

```go
// 构造与启动
shell, err := host.New(host.Config{
    DataDir:  dataDir,              // 或 Endpoint 显式指定
    Args:     os.Args[1:],          // 二次启动转交
    Version:  "0.1.0-dev",          // 只进诊断
    Diag:     host.NewRecorder(file),
    Connected: func() bool { ... }, // 连接领域的只读探针
    Startup: host.StartupSteps{
        LocalState:   func(ctx context.Context) error { ... },
        Connect:      func(ctx context.Context) error { ... }, // D1: Supervisor.Connect
        Subscribe:    func(ctx context.Context) error { ... }, // D1/D2: 订阅
        FirstHistory: func(ctx context.Context) error { ... }, // D2: 首轮历史
        AllowWrites:  func(ctx context.Context) error { ... }, // D3/D5 打开写入
    },
    Shutdown: host.ShutdownSteps{
        StopIntents:     func(ctx context.Context) error { ... }, // D3/D5
        EndInteraction:  func(ctx context.Context) error { ... }, // D2/D5
        CloseConnection: func(ctx context.Context) error { ... }, // D1
        ReclaimService:  func(ctx context.Context) error { ... }, // D1: Supervisor.Shutdown
    },
    Timeouts: host.DefaultTimeouts(),
})
outcome, err := shell.Start(ctx)
if outcome == host.StartForwarded { return 0 }   // 参数已转交，本进程退出

// 进程级状态与写闸门
state := shell.State()        // Stage / Failed / Connected / Exiting / Exited
err = shell.Writable()        // ErrNotReady / ErrNotConnected / ErrExiting
for act := range shell.Activations() { ... } // 二次启动参数，交给界面
shell.ObserveConnection(false, host.ClassConnectionMissing) // 只写诊断

// 退出
report, err := shell.Shutdown(ctx)

// 应用更新（P3）
updater, err := host.NewUpdater(host.UpdateConfig{
    CurrentVersion: "0.1.0",                 // 或 ReadVersion 从安装产物读
    TargetPath:     appArtifactPath,         // 本版只支持单文件
    SessionsIdle:   func() bool { ... },     // 会话是否已收尾
    Settle:         func(ctx context.Context) error { ... },
    Shutdown:       func(ctx context.Context) error { _, err := shell.Shutdown(ctx); return err },
    Restart:        func(ctx context.Context) error { ... }, // 启动新进程
    Diag:           diag,
})
status, err := updater.Check(ctx)      // 失败也返回状态：up_to_date + LastError
artifact, err := updater.Download(ctx)
err = updater.Switch(ctx)              // 调用即用户确认
defer updater.Close()
```

**需要其他领域提供（本包不实现）**：

- D1：`Connect`/`OpenStream`/`Supervisor.Shutdown`；连接状态只读探针；错误实现 `ErrorClass() string` 或由调用方包装成 `host.ErrorClassFrom(分类名)`。
- D2：首轮历史读取完成信号（阶段 5）；"结束交互"时清空未落库尾部。
- D3：停止新意图（拒绝新写）、在途写的处理归属。
- D5：结束待决审批与提问的交互侧收尾。

本包对上述领域的唯一要求是"提供可超时的步骤函数"；不 import 任何领域包，也不重复声明协议类型。

## 6. 验收

### 6.1 命令与结果

```bash
go test ./...          # 全仓库通过；internal/host ok
go vet ./...           # 无输出
go test -race -count=3 ./internal/host/   # 通过
GOOS=windows go vet ./internal/host/      # 通过（仅编译验证）
```

本机（macOS arm64，Go 1.24.3）实测：`go test ./...` 全通过，`go vet ./...` 无输出，`internal/host` 竞态检测通过。

### 6.2 覆盖

| 要求 | 测试 |
|---|---|
| 单实例：第二次启动拿不到端点 | `TestSecondLaunchForwardsArgsAndPrimaryKeepsRunning`、`TestConcurrentAcquireOnlyOneWins` |
| 二次启动把参数转交给已有实例后退出（含工作目录） | `TestSecondLaunchForwardsArgsAndPrimaryKeepsRunning`、`TestShellLifecycle` |
| 转交失败如实报错；已有实例拒绝时不静默 | `TestForwardWithoutPrimaryFails`、`TestShellReportsForwardFailure`、`TestShellLifecycle` |
| 接管队列满时丢弃且不记内容 | `TestActivationOverflowDropsWithoutLoggingContent` |
| 被强杀后收敛：清理残留端点、重新独占、无额外状态文件 | `TestStaleSocketIsReclaimed` |
| 端点被普通文件占用时不删用户数据 | `TestRegularFileEndpointIsNotDeleted` |
| 端点关闭后删除，可重新抢占 | `TestSecondLaunchForwardsArgsAndPrimaryKeepsRunning` |
| 六阶段顺序推进、每阶段可观察 | `TestStartupRunsSixStagesInOrder`、`TestStoreAdvancesStagesInOrder` |
| 阶段失败停在当前阶段、如实报、不推进后续 | `TestStartupStopsAtFailingStage`、`TestShellStartStopsAtFailingStage` |
| 缺实现/错序/缺阶段不允许启动 | `TestNewStartupRequiresFullSuffix`、`TestShellNewRequiresConfiguration` |
| 启动取消即停 | `TestStartupStopsOnCanceledContext` |
| 连上之前与就绪之前不允许写 | `TestStoreWritableStates`、`TestShellLifecycle`（首轮历史阶段仍 `ErrNotReady`） |
| 断线不算可写、不把未连接当空数据 | `TestShellLifecycle`（断线后 `ErrNotConnected`） |
| 非法阶段推进丢弃并写诊断 | `TestStoreAdvancesStagesInOrder` |
| 退出五阶段顺序 | `TestShutdownRunsFivePhasesInOrder`、`TestShellLifecycle` |
| 退出中拒绝新意图 | `TestShutdownRunsFivePhasesInOrder`、`TestShellLifecycle` |
| 某段超时记录后继续，不无限等待 | `TestShutdownTimeoutContinues`、`TestShellLifecycle` |
| 某段失败记录后继续 | `TestShutdownFailureContinues` |
| 退出幂等、并发只跑一次 | `TestShutdownIsIdempotent` |
| 父上下文取消仍走完退出 | `TestShutdownSurvivesCanceledParent` |
| 退出缺阶段不允许构造 | `TestNewShutdownRequiresSteps` |
| 诊断只写白名单字段 | `TestRecorderOnlyWritesWhitelistFields`、`TestRecorderDropsZeroValueField` |
| 凭据与正文进不来 | `TestRecorderRejectsCredentialsAndBodies`、`TestSecondLaunchForwardsArgsAndPrimaryKeepsRunning`、`TestActivationOverflowDropsWithoutLoggingContent` |
| 事件名与版本形状校验 | `TestRecorderEveryDeclaredEventMatchesShape`、`TestVersionFieldValidatesSemver` |
| 多协程写入不串行、冲刷调用底层 Flush/Sync | `TestRecorderIsConcurrentAndLineAtomic`、`TestRecorderFlushAndWriteFailures` |
| 进程级状态：阶段/失败/已连接/退出中 | `TestStoreAdvancesStagesInOrder`、`TestStoreWritableStates`、`TestStoreFailStopsProgress`、`TestShellLifecycle` |
| 更新检查：读当前版本、有新版、已是最新（含更低版本） | `TestUpdaterCheckFindsNewVersion`、`TestUpdaterCheckUpToDate`、`TestCompareVersions` |
| 更新检查失败按已是最新呈现、不阻塞使用 | `TestUpdaterCheckFailureIsRecordedAndDoesNotBlock` |
| 版本源形态：渠道地址推导、清单解析、缺摘要/坏版本拒绝 | `TestUpdateManifestURLMatchesExistingChannel`、`TestParseManifestElectronBuilderShape`、`TestParseManifestFallsBackToTopLevelFields`、`TestParseManifestRejectsIncompleteManifests` |
| 下载落临时目录并校验；校验失败保留当前版本且不留产物 | `TestUpdaterDownloadVerifiesArtifact`、`TestUpdaterDownloadFailureKeepsAvailable` |
| 非法更新迁移丢弃并记诊断 | `TestUpdaterCheckFindsNewVersion`、`TestUpdaterDownloadFailureKeepsAvailable` |
| 有会话时推迟切换，不替换不重启 | `TestUpdaterSwitchDefersWhileSessionsBusy` |
| 会话收尾后按"退出 → 替换 → 重启"切换，失败保留当前版本 | `TestUpdaterSwitchDefersWhileSessionsBusy`、`TestUpdaterSwitchInstallFailureKeepsCurrentVersion` |
| 切换后版本号变化（假文件模拟） | `TestUpdaterSwitchDefersWhileSessionsBusy`（`ReadVersion` 从假文件读，切换后为 2.1.0） |
| 切换缺目标路径/重启步骤时不迁移、不替换 | `TestUpdaterSwitchRequiresWiring`、`TestUpdaterSwitchWithoutShutdownWiring` |
| 更新诊断字段也在白名单内 | `TestRecorderOnlyWritesWhitelistFields`、`TestRecorderEveryDeclaredEventMatchesShape` |

### 6.3 真实更新渠道（未验证）

本版更新只验证到"清单/产物形状与状态机正确"：测试用本地 HTTP 服务模拟版本源与产物，校验 sha512、切换与假文件版本变化；**没有请求真实渠道**（`https://dimcode.echooai.com`），也没有在真实安装包上替换。

原因：真实渠道提供的是 macOS `.app`（dmg/zip）与各平台安装器，本版只实现单文件产物替换；发布与签名工具链不在本领域。解除方式：发布阶段（X1）提供单文件产物或安装器适配后，用真实渠道跑一次"检查 → 下载校验 → 切换 → 重启后版本号变化"。

### 6.4 真实 daemon 证据（未验证）

本版只验证到"可执行步骤接线正确"。与真实 daemon 的接线需要 main 集成（`cmd/dimagent` 目前仍是原型入口），因此以下两项按未验证记录，不记为通过：

1. 启动时无可用服务 → 记录一条连接缺失分类；有服务 → 一次连接成功并读到既有会话标识。
2. 退出后本机自启的服务进程消失、外部启动的仍在（顺序：先交互后回收）。

解除方式：main 集成后按 `testdata` 外的真实数据目录执行，观察 `startup.stage`、`shutdown.phase` 记录与服务端进程列表。

## 7. 依赖与上下文需求

| 依赖 | 需要对方冻结什么 | 状态 |
|---|---|---|
| D1 连接 | `Connect`/`OpenStream`/`Supervisor.Shutdown` 与连接状态探针；错误分类名 | F3 候选已提供，接口形状见 D1 §5 |
| D2 会话投影 | 首轮历史完成的回调；结束交互时清空未落库尾部 | 需 D2 确认回调时机 |
| D3 写登记 | 停止新意图的入口；退出中拒绝新写的口径 | 需 D3 确认 |
| D5 交互裁决 | 结束待决审批/提问的入口 | 需 D5 确认 |
| 全部领域 | 诊断走同一 `Recorder`，只传白名单字段 | 需各领域在接线时遵守 |
| V 验收设施 | 真实 daemon 场景的 fixtures 与脚本 | 待 P1-3 集成 |

本领域不修改其他领域实现，也不替其他领域接线；以上接口只描述需要什么。

## 8. 已知限制

1. **Windows 未验证**：单实例用回环 TCP + 端口文件，先探测后监听，同时启动存在竞态窗口；`GOOS=windows` 只验证了编译。
2. **本地状态装载只留阶段位**：窗口与偏好（A.md 归进程框架）的存储未实现；`StartupSteps.LocalState` 由调用方提供。
3. **应用更新只覆盖单文件替换**：macOS `.app` 目录、dmg/zip 安装包、Windows 安装器与 Linux AppImage 的原地替换都未实现；真实渠道未做端到端验证（只用本地 HTTP 服务模拟清单与产物）。
4. **更新没有签名校验与发布能力**：只比对 sha512，不校验代码签名/公证；没有定时检查、渠道切换、增量下载与桌面项修复；版本比较只支持 `x.y.z[-pre]`，当前版本必须能被解析。
5. **替换后重启失败不回滚**：磁盘上已经是新产物，下一次启动会运行新版本；本版不保留旧产物备份，也不做回滚。
4. **超时 goroutine 不强制回收**：某段实现若完全忽略上下文，`Run` 会按时返回并继续，但该 goroutine 会存活到进程结束。这是"不无限等待"的代价，实现方应尊重上下文。
5. **端点路径长度**：Unix 套接字路径受系统上限（macOS 约 104 字节）限制；数据目录过深时启动报错，需换短数据目录。临时名比端点名略长，仍在同一目录。
6. **诊断没有人类可读消息字段**：这是刻意取舍——可诊断性靠事件名 + 白名单字段；需要正文的场景一律不进日志。
7. **转交报文只有一行、无认证**：端点位于 0700 目录且套接字 0600，只允许同一用户连接；不额外做凭据校验。
8. **二次启动转交成功后不再调用退出五阶段**：该进程没有建立任何连接或本地状态，直接退出即可；main 集成时需按此实现。

## 9. 变更记录

| 日期 | 变更 | 原因 |
|---|---|---|
| 2026-11-10 | 第一版：单实例端点互斥与二次启动转交、启动六阶段、退出五阶段逐段超时、诊断白名单、进程级状态 | 结果包要求覆盖范围 1–5 |
| 2026-11-10 | 单实例抢占改为"临时名监听 + 硬链接原子发布" | 并发启动压测暴露 bind 与 listen 之间的空窗会被误判为残留，导致多个赢家 |
| 2026-11-11 | 补齐 P3 应用更新：检查、下载校验、切换与状态机；新增 `update_state`/`update_version` 诊断字段 | 用户拍板 R9 要做更新，此前版本明确未实现 |
