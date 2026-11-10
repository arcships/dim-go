# D11b 扩展与后台（插件、技能、钩子、工具服务器、计划任务、工作流、唤醒）

Status: 第一版实现完成（D11 拆分后的扩展与后台部分，待组织评审）
Owner: 扩展与后台领域 agent
验证: 可执行命令与覆盖表见第 6 节；证据档位与 blocked 见 6.3、6.4
依据: `docs/domains/R.md` 功能块 8、9 与「块 8 扩展管理」「块 9 计划任务、运行组件、存储读数、唤醒与模态」的沿用条目；`docs/domains/A.md` 管理面读数、四条总规矩与状态归属；`docs/domains/D1-connection.md` 的错误分类；`docs/domains/D3-write.md` 的写登记意图与状态
范围: 插件、技能、钩子、外部工具服务器、计划任务、工作流定义与运行、会话级唤醒的读取、写入意图与事件跟随。不含界面、连接实现、protocol wire 定义、会话折叠、设置与模型、账号。

---

## 1. 结果包

**领域**：D11b 扩展与后台

**输入**

| 输入 | 来源 | 本版使用方式 |
|---|---|---|
| 功能块 8「扩展管理」 | `docs/domains/R.md` | 插件列表与安装来源、技能安装与启停、钩子按来源的写入范围、工具服务器增删改与探测、配置导入预览与按勾选写入 |
| 功能块 9「计划任务、…、唤醒」 | `docs/domains/R.md` | 计划任务列表与事件跟随、唤醒记录读取/取消/状态跟随 |
| 管理面读数、四条总规矩 | `docs/domains/A.md` | 每项读数只有一个驱动者；只由读取与变更通知驱动；缺能力报缺失；不做乐观更新；不保存第二份事实 |
| 连接错误分类 | `docs/domains/D1-connection.md`、`internal/appserver/failure` | 能力缺失归一成 `CapabilityError`；超时、连接失败、参数错误原样上报，不冒充缺能力 |
| 写登记 | `docs/domains/D3-write.md`、`internal/write` | 所有变更登记为 `KindManagementWrite`，一次发送、不自行重试、结果由写登记收敛 |
| 假服务端与真实客户端 | `internal/fakeappserver`、`internal/appserver/protocol`、`internal/appserver/daemon` | 本机 Unix socket + 真实 NDJSON，验证读取、写入只发一次、通知跟随、能力缺失与恢复 |
| 实现约束 | 本仓库 `AGENTS.md` | 只用 Go 标准库；不改 `go.mod`；只写本领域包与本文档 |

**产物**

| 路径 | 职责 |
|---|---|
| `internal/extensions/extensions.go` | 对外类型、错误与能力缺失分类、读镜像与写入键、写登记调用、管理写入参数形状 |
| `internal/extensions/follow.go` | 变更通知主题、单一跟随协程、同作用域合并、重读计数与日志 |
| `internal/extensions/plugins.go` | 插件清单、按来源安装、启停、更新 |
| `internal/extensions/skills.go` | 技能清单、安装、启停、来源区分 |
| `internal/extensions/hooks.go` | 钩子清单与用户来源写入范围 |
| `internal/extensions/toolservers.go` | 工具服务器增删改、显式探测、导入预览与按勾选写入 |
| `internal/extensions/scheduler.go` | 计划任务清单与镜像读数 |
| `internal/extensions/workflows.go` | 工作流定义浏览、启动/停止、运行读数与跟随 |
| `internal/extensions/wakeups.go` | 唤醒读取、取消与状态跟随 |
| `internal/extensions/extensions_test.go` | 注入替身覆盖能力分类、钩子来源、导入勾选、通知跟随、失败保留等分支 |
| `internal/extensions/appserver_test.go` | 假服务端 + 真实 NDJSON 的插件/技能、钩子、工具服务器、计划任务、唤醒、工作流与能力缺失证据 |
| `docs/domains/D11b-extensions.md` | 本文件 |

**验收**

- 命令：`go test ./internal/extensions/`、`go test -race ./internal/extensions/`、`go vet ./...`。本版全部通过（2026-11-10，macOS arm64，Go 1.24.3）。
- 本领域测试：19 个顶层测试全部通过。
- 证据档位：
  - `fakeappserver`：本机 Unix socket 上真实 NDJSON，覆盖插件/技能/钩子/工具服务器/计划任务/工作流/唤醒的关键分支。
  - 注入替身：读取方与写登记替身，覆盖能力缺失与恢复、来源拒绝、导入勾选、失败保留、通知与时序。
  - 真实 App Server 上的管理面方法：**blocked**，wire 方法名、回执字段与通知名尚未冻结（见 6.4）。

**依赖**

| 依赖 | 需要对方冻结/提供 |
|---|---|
| D1 连接 | 集成层实现本包的七个读取接口；把服务端通知映射成 `Notice`；提供管理面各方法的 wire 适配。本包不定义方法名与回执形状 |
| D3 写登记 | `Register`/`Wait` 与 `KindManagementWrite`；错误分类让超时/断线转未知、能力缺失按明确失败处理；本包不重试 |
| D11a 设置与模型 | 与本包无状态重叠：插件/技能的配置内容、模型与 provider 不在本包 |
| D6 账号 | 本包不读账号状态，也不按账号过滤任何管理读数 |
| U1 原生界面 | 只读 `Snapshot` 与 `write.IntentView`；通过 `Notify`/`WaitIdle` 驱动刷新，不直接改镜像 |
| V 验收设施 | 假服务端已可用于本领域；管理面真实方法冻结后补 wire golden |

**上下文需求**

- 需要 D1/管理面冻结以下方法名、参数与回执字段：插件清单/安装/启停/更新，技能清单/安装/启停，钩子清单/写入，工具服务器清单/增删改/探测/导入预览，计划任务清单，工作流清单/详情/运行/启动/停止，唤醒清单/取消。
- 需要冻结服务端通知名与参数：计划任务变更、唤醒变更、工作流运行变更；插件/技能/钩子/工具服务器变更通知本版已保留主题入口。
- 需要确认工作流运行状态、唤醒状态、计划任务状态的服务端枚举取值；本版按字符串原样保留，未知值不猜测语义。
- 需要确认工具服务器导入配置的来源形态（配置路径、分类名或服务端标识）与候选键规则；本版只把来源当不透明标识，并要求候选键在一份预览里唯一。
- 需要确认钩子是否支持删除；本版只做清单读取与写入（upsert）。
- 需要确认随包可选插件的安装入参：本版用 `bundled` + 插件标识，服务端若要求其他形状由集成层适配。

## 2. 边界

**做**

| 能力 | 说明 |
|---|---|
| 插件 | 清单（含未安装的可选插件）、按本地路径/Git/随包来源安装、启停、更新；状态由服务端保存 |
| 技能 | 清单与来源、安装、启停；本端不留独立记录，重启后重读仍一致 |
| 钩子 | 清单、来源区分、只有用户来源可写；项目/插件/未知来源只读 |
| 外部工具服务器 | 清单、新增、修改、删除、显式探测、导入预览、按勾选逐条写入 |
| 计划任务 | 清单读取；服务端通知到达时重读一次，无定时轮询 |
| 工作流 | 定义清单与详情浏览、启动、停止、运行状态读取与通知跟随 |
| 会话级唤醒 | 按会话/分支读取、取消；读取过的范围接收通知并只重读一次 |
| 缺能力 | 每个读取入口返回带能力名的 `CapabilityError`，能力恢复后同一实例可再次读到列表 |
| 观察 | 每个读数的最近确认值、读取时间、读取错误、能力缺失标记；跟随计数与日志 |

**不做**

| 不做 | 归属/原因 |
|---|---|
| 界面、列表渲染、按钮状态、表单校验提示 | U1 |
| 连接、socket、NDJSON、方法名与 wire 类型 | D1 |
| 设置项、模型与 provider、权限档位 | D11a |
| 账号、登录、用量 | D6 |
| 会话折叠、分支、审批、提问 | D2/D5 |
| 计划任务的执行、补跑与本地调度 | 服务端；本端只读清单与状态 |
| 自动探测工具服务器、自动连接外部进程 | R 块 8：探测只由显式调用触发 |
| 插件市场、外部内容容器 | R 附 C：不做 |
| 自动重试、自动重发、乐观更新 | A 四条总规矩；D3 写登记 |
| 持久化镜像、导入预览、跟随订阅 | R8：这些是可由用户丢弃的本地状态；重启后重读 |

本包只使用 Go 标准库与仓库内已有的错误分类/写登记包，不引第三方依赖，不改 `go.mod`。

## 3. 包结构与对外接口

```text
internal/extensions/
├── extensions.go    Snapshot、错误与 CapabilityError、来源、WriteOp、ManagementWrite、
│                    Config/New/Close、readInto、写键与 write
├── follow.go        Topic/Notice、FollowStats、Notify/WaitIdle、唯一跟随协程
├── plugins.go       Plugin、PluginReader、ListPlugins、InstallPlugin、SetPluginEnabled、UpdatePlugin
├── skills.go        Skill、SkillReader、ListSkills、InstallSkill、SetSkillEnabled
├── hooks.go         Hook、HookReader、ListHooks、WriteHook
├── toolservers.go   ToolServer*、探测、导入预览、ApplyToolServerImport
├── scheduler.go     ScheduledTask、SchedulerReader、ListScheduledTasks
├── workflows.go     WorkflowDefinition/Run、WorkflowReader、启动/停止/运行读数
├── wakeups.go       Wakeup、WakeupReader、ListWakeups、CancelWakeup
├── extensions_test.go      替身驱动的领域分支测试
└── appserver_test.go       假服务端 + 真实 NDJSON 的集成证据
```

对外接口（节选）：

```go
service, err := extensions.New(extensions.Config{
    Plugins:     pluginReader,     // 七个读取方按能力配置；nil 表示本端没有该能力
    Skills:      skillReader,
    Hooks:       hookReader,
    ToolServers: toolServerReader,
    Scheduler:   schedulerReader,
    Workflows:   workflowReader,
    Wakeups:     wakeupReader,
    Writer:      writeManager,     // *write.Manager 直接满足
    Logf:        logf,             // 能力缺失与跟随重读；不写凭据与参数
})
defer service.Close()

// 读取：返回最近确认快照与本次错误；失败时保留上次 Items。
plugins, err := service.ListPlugins(ctx)
scheduled, err := service.ListScheduledTasks(ctx)

// 写：全部经写登记，返回写意图视图；状态以 write.Status 为准。
view, err := service.InstallPlugin(ctx, extensions.InstallSource{Kind: extensions.SourceLocal, Ref: "/path/to/plugin"})
view, err = service.SetPluginEnabled(ctx, "plugin-id", true)
view, err = service.WriteHook(ctx, extensions.Hook{Event: "pre", Source: extensions.SourceUser, ...})

// 工具服务器：探测与预览只读；应用按勾选逐条写。
probe, err := service.ProbeToolServer(ctx, "server-id")
preview, err := service.PreviewToolServerImport(ctx, "config-a")
result, err := service.ApplyToolServerImport(ctx, "config-a", []string{"candidate-b", "candidate-c"})

// 事件跟随：集成层把服务端通知映射成 Notice。
service.Notify(extensions.Notice{Topic: extensions.TopicScheduler})
service.Notify(extensions.Notice{Topic: extensions.TopicWakeups, SessionID: "s-1", BranchID: "main"})
err = service.WaitIdle(ctx)
```

管理写入参数形状（供集成层发送方解析）：

```go
managed, err := extensions.DecodeManagementWrite(request.Params)
// managed.Op 取 plugin.install / plugin.set_enabled / ... / wakeup.cancel
// managed.Payload 是该操作自己的 JSON 载荷
```

## 4. 设计

### 4.1 读镜像与快照

- 每个能力有一份 `Snapshot[T]`：最近一次成功读取的 `Items`、`ReadAt`、最近一次读取的 `ReadError`、`CapabilityMissing`。
- 成功读取整体替换 `Items` 并清除错误；失败读取**保留**上一次确认的 `Items`，只更新错误与能力缺失标记。界面因此能区分「空列表」与「读不到」。
- 全部读取共用一把 `readMu` 串行执行，保证先发出的旧读取不会覆盖后完成的新读取。
- `*Snapshot()` 访问器只返回副本，不发起请求；镜像只由读取与跟随入口更新，符合「每个状态只有一个驱动者」。

### 4.2 能力缺失分类

`CapabilityError` 带能力名，并同时满足：

```go
errors.Is(err, extensions.ErrCapabilityMissing)
errors.Is(err, &extensions.CapabilityError{Feature: "scheduler.list"})
```

识别两类能力缺失：

1. 本端没有配置读取方（`Config` 中对应字段为 nil）；不发请求，直接报缺失并记录日志。
2. 服务端返回稳定的 `appError`：`unknown_method`、`method_not_found`、`capability_missing`、`not_supported`。

超时、连接失败、参数错误与未知错误不会被冒充成能力缺失。`CapabilityError` 保留底层错误，因此 `failure.KindOf` 仍会把它归为不可重试。计划任务能力恢复后再次读取即可得到列表；同一实例不需要重建。

### 4.3 写意图

所有变更都经注入的写登记发出，类型固定为 `write.KindManagementWrite`：

| WriteOp | 含义 |
|---|---|
| `plugin.install` / `plugin.set_enabled` / `plugin.update` | 插件安装、启停、更新 |
| `skill.install` / `skill.set_enabled` | 技能安装与启停 |
| `hook.write` | 用户来源钩子写入 |
| `tool_server.add` / `tool_server.update` / `tool_server.remove` | 工具服务器增删改；导入应用也走 `tool_server.add` |
| `workflow.start` / `workflow.stop` | 工作流启动与停止 |
| `wakeup.cancel` | 唤醒取消 |

- 参数形状是 `{"op": "...", "payload": {...}}`，由 `ManagementParams` 生成、`DecodeManagementWrite` 解析；本包不定义 wire 方法名。
- 幂等键由「操作名 + 载荷摘要」生成。同一内容重复登记不会产生第二次请求；上一次明确失败或确认未发生时换一个新键允许重试；工作流启动在成功后再次调用也会换键，允许用户再启动一次。
- 本包只调用一次 `Register`，随后等待终态；不自行重试、不自动补发。写失败、未知、需人工确认都以 `write.IntentView.Status` 表达，交给界面与人工处理。
- 写成功后不自动重读清单：管理面事实只来自读取与变更通知；调用方在需要时显式读取，或等待服务端通知。

### 4.4 插件与技能

- 插件清单保留服务端给出的全部条目；未安装的可选插件以 `Installed=false, Optional=true` 出现，不做过滤。
- 安装来源只接受 `local`、`git`、`bundled`；引用不能为空。是否真的可安装由服务端判定，本包只拒绝结构非法的请求。
- 技能清单带来源；安装与启停都写服务端。本端不留独立记录，重启后重新读取即可得到同一状态。
- 插件更新是命令型操作：每次显式调用都会在成功后允许再登记；同一未结项期间重复调用只等同一条意图。

### 4.5 钩子的来源范围

- `Hook.Writable()` 只在来源为用户时返回真；未知来源按只读处理。
- `WriteHook` 在调用写登记**之前**校验来源：项目、插件、未知或空来源一律返回 `ErrSourceNotWritable`，不产生写意图，也不发出任何请求。
- 写入前还要求事件非空；钩子的其他字段由服务端解释，本包不展开业务校验。

### 4.6 工具服务器：显式探测与按勾选导入

- 清单读取、新增、修改、删除都不会触发探测；只有 `ProbeToolServer` 会发一次探测请求。探测结果只是预览，存为最近一次探测读数，不写回配置。
- 导入先 `PreviewToolServerImport`：读取候选并缓存在内存中，按来源键保存；预览会校验候选键非空且唯一、服务器配置结构合法。
- `ApplyToolServerImport`：
  1. 要求该来源已有预览，否则 `ErrPreviewRequired`；
  2. 要求勾选非空、无重复、每个键都在预览里；任一不合法都不写任何条目；
  3. 按预览顺序为每条被勾选候选登记一条 `tool_server.add`，写入条数等于勾选条数；
  4. 任一条没有明确成功就停止，返回已有结果与 `ErrWriteNotSucceeded`。
- 工具服务器的环境变量只出现在写入参数中；清单只返回键名，不返回值；本包日志不记录参数或环境变量内容。

### 4.7 事件跟随

- 集成层把服务端通知映射成 `Notice{Topic, SessionID, BranchID, WorkflowID}`；本包不解释通知方法名。`Notify` 只入队、不阻塞、不发请求，因此可以直接在连接的通知回调里调用；实际读取在跟随协程里进行，不会卡住连接的读循环。
- 只有一个跟随协程，没有定时器，也没有周期性全量读取。同主题同作用域的通知在读取开始前合并，最多重读一次；读取进行中到达的通知作为下一批再重读一次。
- 计划任务：一次通知重读一次清单。
- 工作流：只重读已经通过 `ListWorkflowRuns` 或 `StartWorkflow` 关注过的工作流；未知工作流不触发读取。
- 唤醒：只重读已经通过 `ListWakeups` 关注过的会话/分支；通知带范围时只读该范围，不带范围时对每个已关注范围各读一次（不是一次全局读取）。
- `FollowStats` 与日志给出通知数、重读数、唤醒每范围重读数，验收可直接核对「通知后只重读一次、之后不再增加」。
- `WaitIdle` 只用于等待当前通知批次处理完，不轮询；生产代码不需要周期性调用它。

### 4.8 唤醒取消

- `CancelWakeup` 只登记一条 `wakeup.cancel` 写意图，不因取消成功而自行重读。
- 记录从读数中消失由服务端通知触发的那一次重读确认；若没有通知，调用方显式 `ListWakeups` 也能得到最新事实。
- 因此“取消 + 通知”在证据里表现为：写请求一次、读取总次数只增加一次。

### 4.9 并发与关闭

- 读取、跟随、写意图登记可并发调用；镜像更新与跟随计数由内部互斥保护。
- `Close` 停止跟随协程并拒绝后续读取/写入；跟随中的读取共享可取消上下文。显式读取使用调用方上下文，关闭不会中断已经开始的外部请求，这是本版已知限制。

## 5. 与 A/R 的对应

| 需求或架构条目 | 本领域实现 | 不在本领域 |
|---|---|---|
| R 块 8 插件列表与安装来源 | 清单保留可选未安装项；local/git/bundled 三类安装来源 | 插件市场；插件内部能力 |
| R 块 8 插件启停、更新 | `plugin.set_enabled` / `plugin.update` 写意图 | 插件运行进程与日志 |
| R 块 8 技能安装与启停 | 清单、安装、启停写意图；状态只在服务端 | 技能内容执行 |
| R 块 8 钩子写入范围 | `Hook.Writable` 与写入口的来源前置校验 | 钩子执行与审批 |
| R 块 8 工具服务器增删改与探测 | 清单、增删改写意图、显式探测 | 外部进程连接实现 |
| R 块 8 导入预览与按勾选写入 | 预览缓存、勾选校验、逐条写入 | 配置文件解析 |
| R 块 9 计划任务列表与变更 | 清单读取与通知后重读一次 | 任务执行、补跑 |
| R 块 9 唤醒读取、取消与状态跟随 | 按范围读取、取消写意图、每范围一次重读 | 定时器触发与提醒展示 |
| R 块 8/9 能力缺失 | `CapabilityError` + 日志 + 恢复后可再读 | 服务端能力实现 |
| A 管理面读数只由读取与通知驱动 | 镜像更新只经 `readInto` 与跟随入口 | 界面状态 |
| A 四条总规矩 | 不写事实、不做乐观更新、一个状态一个驱动者、缺能力报缺失 | 其他模块状态 |
| A 检查项「缺能力就报缺失」 | 每个读取入口分类错误，不用空列表冒充 | — |

## 6. 验收

### 6.1 命令与结果

```bash
go test ./internal/extensions/
go test -race ./internal/extensions/
go vet ./...
```

本版结果（2026-11-10，macOS arm64，Go 1.24.3）：

- `go test ./internal/extensions/`：`ok`（19 个顶层测试全部通过）。
- `go test -race ./internal/extensions/`：`ok`。
- `go vet ./...`：无输出。
- `go test -count=1 ./...`：本次全部包 `ok`（含 `internal/terminal`；该包不属于本领域，本版未修改）。同一天另一次全量运行中 `internal/terminal.TestSmokeFullScreenPrograms` 偶发超时，单独重跑通过，未作为本领域结论依据。

### 6.2 覆盖

| 验收要求 | 测试 |
|---|---|
| 插件清单保留未安装的可选插件 | `TestPluginListKeepsOptionalUninstalled`、`TestAppServerPluginsAndSkills` |
| 按来源安装、启停、更新；写入只发一次 | `TestAppServerPluginsAndSkills`、`TestInstallSourceAcceptsEverySupportedKind` |
| 非法安装来源在写之前拒绝 | `TestInvalidInstallSourceIsRejectedBeforeWrite` |
| 技能启停后重读保持；重启客户端后仍保持 | `TestAppServerPluginsAndSkills`（新实例重读同一服务端状态） |
| 只有用户来源的钩子可写；其他来源不发请求、不登记意图 | `TestHookSourceIsTheOnlyWritableOne`、`TestAppServerHookSourceScope` |
| 工具服务器增删改、清单不探测 | `TestAppServerToolServersAndImport` |
| 显式探测返回工具预览且不写配置 | `TestAppServerToolServersAndImport` |
| 导入预览 3 条；应用只写勾选条数 | `TestAppServerToolServersAndImport`、`TestImportPreviewSelectsOnlyCheckedCandidates` |
| 未预览、空勾选、未知候选都不写入 | `TestImportPreviewSelectsOnlyCheckedCandidates`、`TestAppServerToolServersAndImport` |
| 计划任务能力缺失分类并记录；恢复后返回列表 | `TestAppServerSchedulerFollowing`、`TestCapabilityMissingIsClassifiedAndRecovers` |
| 计划任务通知后只重读一次、无轮询 | `TestAppServerSchedulerFollowing`、`TestSchedulerFollowsNotificationOnceWithoutPolling` |
| 工作流定义浏览、启动/停止、运行状态与通知跟随 | `TestAppServerWorkflowRuns` |
| 唤醒读取、取消后消失、通知只重读一次 | `TestAppServerWakeupFollowAndCancel`、`TestWakeupFollowsCancellationOnce` |
| 读取失败保留上一次确认事实 | `TestSnapshotKeepsConfirmedFactsOnReadFailure` |
| 十一个读取入口缺能力都带能力名 | `TestEveryCapabilityReportsMissingWhenReaderIsAbsent` |
| 写登记缺失时写入口明确不可用 | `TestWriteRequiresWriter` |
| 失败后用新幂等键重试，不重复发送 | `TestWriteRetriesWithNewKeyAfterFailure` |
| wire 层能力缺失不冒充空列表；写入明确失败 | `TestAppServerMissingCapabilityOnWire` |

### 6.3 证据档位

- **fakeappserver + 真实客户端**：本机 Unix socket、真实 NDJSON 与 `daemon` 握手；wire 形状只出现在 `appserver_test.go` 的适配器里。覆盖七个范围的读取、写入只发一次、通知跟随、能力缺失与恢复。
- **注入替身**：读取方与写登记替身，覆盖时序、来源拒绝、勾选校验、失败保留与关闭语义。
- **真实 App Server / 真实配置 / 真实外部工具服务器**：未验证，见 6.4。

### 6.4 blocked

| 项 | 状态 | 缺什么 |
|---|---|---|
| 真实 App Server 上的插件/技能/钩子/工具服务器方法 | blocked | 管理面 wire 方法名、参数与回执字段尚未冻结；本版由集成层适配 |
| 真实管理面通知名与参数 | blocked | 计划任务、唤醒、工作流变更通知的服务端名称与字段未冻结；本版用注入的 `Notice` 与测试映射验证 |
| 真实插件 Git 仓库与随包插件 | blocked | 没有可用的真实插件源与服务端安装实现 |
| 真实 MCP/外部工具服务器探测 | blocked | 没有真实外部工具服务器进程与凭据；本版只验证请求与预览形状 |
| 真实工具服务器配置文件导入 | blocked | 配置来源形态未冻结；本版要求来源键不透明、候选键在一份预览里唯一 |
| 真实计划任务/工作流服务端 | blocked | 没有可驱动的真实调度器与工作流运行；本版验证读取、写入意图与事件跟随 |

## 7. 已知限制

1. **wire 适配不在本包**：所有方法名、参数与回执都归集成层；真实服务端未验证，方法冻结后需要补 wire 证据。
2. **列表不分页**：每个清单一次读取返回全部条目；服务端若分页，需要集成层在读取方内合并，本包不提供游标。
3. **读取串行**：全部管理面读取共用 `readMu`；一个慢能力会拖慢其他能力的读取。第一版没有按能力分锁。
4. **关闭不中断显式读取**：`Close` 只保证跟随协程退出；已经开始的外部读取由调用方上下文决定何时结束。
5. **快照浅拷贝**：`Snapshot.Items` 复制切片，但条目内部的切片/映射仍可能与内部值共享；调用方必须按只读使用。
6. **能力缺失识别有限**：只识别约定的 `appError` 集合与本端 nil 读取方；服务端新增错误名需要扩展映射表。
7. **写入成功不自动重读**：镜像只由读取与通知更新；界面需要在写终态后显式读取或等待通知。
8. **导入预览在内存**：`ApplyToolServerImport` 依赖同一实例里最近一次预览；重启或换实例后必须重新预览（R8：可丢状态）。
9. **写重试靠新键**：同一内容成功过就不会再发；需要重新执行命令型操作（如工作流启动）时才换键。未知与需人工确认不会自动重试。
10. **工具服务器环境变量经过写参数**：不会进入快照、日志或错误文本，但会留在写登记的内存参数里；不得在集成层打印。
11. **钩子删除未实现**：本版只有清单与写入（新建/更新）；删除需要服务端语义确认后再加。
12. **计划任务只读**：本版只做清单与跟随，不提供启停、立即运行或补跑；它们不在 R 块 9 的列表/变更要求内。
13. **工作流运行标识在回执里**：`StartWorkflow` 只返回写意图视图，运行标识在 `view.Result` 中，由集成层/界面解析；本包不定义回执字段。
14. **唤醒跟随只覆盖已读范围**：没有先 `ListWakeups` 的会话/分支不会因通知自动读取；这是为了避免全量轮询。
15. **平台**：只在 macOS arm64 上跑过测试；Windows 未验证（与 D1 的命名字段限制一致，本包本身无平台分支）。

## 8. 下一步

1. 与管理面负责人冻结 wire 方法、回执与通知名，补一个真实服务端上的完整管理面证据。
2. 为插件/技能/钩子/工具服务器补变更通知的 wire 证据；本版主题入口已保留。
3. 明确工具服务器配置文件导入的来源键与候选键规则，补真实配置样例。
4. 明确钩子删除与批量写入语义，决定是否扩展 `WriteOp`。
5. 评估列表分页与按能力分锁；当前一页读取足够第一版，但大清单会串行阻塞。
6. 管理面清单变大后评估分页与读取并发；当前一页读取与单锁足够第一版。
