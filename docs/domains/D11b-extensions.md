# D11b 扩展与后台（插件、技能、钩子、MCP、计划任务、工作流、唤醒）

Status: 第一版实现完成（按 alpha.21 实测方法改名并落地，待组织评审）
Owner: 扩展与后台领域 agent
验证: 可执行命令与覆盖见第 6 节；真实 alpha.21 证据见 6.4，blocked 见 6.5
依据: `docs/domains/R.md` 功能块 8、9 与「块 8 扩展管理」「块 9 计划任务、运行组件、存储读数、唤醒与模态」的沿用条目；`docs/domains/A.md` 管理面读数、四条总规矩与状态归属；`docs/domains/D1-connection.md` §3.8 管理面实测方法；`docs/domains/D3-write.md` 的写登记意图与状态
范围: 插件、技能、钩子、MCP（外部工具服务器）、计划任务、工作流定义与运行、会话级唤醒的读取、写入意图与事件跟随。不含界面、连接实现、protocol wire 定义、会话折叠、设置与模型、账号、运行组件。

---

## 1. 结果包

**领域**：D11b 扩展与后台

**输入**

| 输入 | 来源 | 本版使用方式 |
|---|---|---|
| 功能块 8「扩展管理」 | `docs/domains/R.md` | 插件列表与安装来源、技能安装与启停、钩子按来源的写入范围、MCP 增删改与探测、配置导入预览与按勾选写入 |
| 功能块 9「计划任务、…、唤醒」 | `docs/domains/R.md` | 计划任务列表与事件跟随、唤醒记录读取/取消/状态跟随 |
| 管理面读数、四条总规矩 | `docs/domains/A.md` | 每项读数只有一个驱动者；只由读取与变更通知驱动；缺能力报缺失；不做乐观更新；不保存第二份事实 |
| alpha.21 管理面实测 | `docs/domains/D1-connection.md` §3.8、§6.5 | 真实方法名：`plugin.*`、`skills.*`、`hooks.*`、`mcp.*`、`schedule.*`、`session.wakeup.*`；`workflow.*`、`runtime.*` 在 alpha.21 不注册 |
| D1 管理面类型 | `internal/appserver/protocol/management_*.go`、`notices.go` | 集成层按这些类型映射；本包不复制 wire 结构 |
| 连接错误分类 | `docs/domains/D1-connection.md`、`internal/appserver/failure` | 能力缺失归一成 `CapabilityError`；超时、连接失败、参数错误原样上报 |
| 写登记 | `docs/domains/D3-write.md`、`internal/write` | 所有变更登记为 `KindManagementWrite`，一次发送、不自行重试、结果由写登记收敛 |
| 假服务端与真实服务端 | `internal/fakeappserver`、`internal/appserver/protocol`、`internal/appserver/daemon`、`~/.dimcode-alpha/rust/dimcode-server`（0.1.0-alpha.21） | 本机 Unix socket + 真实 NDJSON；假服务端覆盖全部分支，真实服务端覆盖读取、插件/技能启停、唤醒闭环 |
| 实现约束 | 本仓库 `AGENTS.md` | 只用 Go 标准库；不改 `go.mod`；只写本领域包与本文档 |

**产物**

| 路径 | 职责 |
|---|---|
| `internal/extensions/extensions.go` | 对外类型、错误与能力缺失分类、读镜像、写意图判别名、真实方法名归一、管理写入参数、写登记调用 |
| `internal/extensions/follow.go` | 变更通知主题、单一跟随协程、同作用域合并、重读计数与日志 |
| `internal/extensions/plugins.go` | 插件清单、按来源安装、启停、更新 |
| `internal/extensions/skills.go` | 技能清单、安装、启停、来源区分 |
| `internal/extensions/hooks.go` | 钩子清单与用户来源写入范围 |
| `internal/extensions/toolservers.go` | MCP 目录、增删改、显式探测、导入预览与按勾选写入 |
| `internal/extensions/scheduler.go` | 计划任务清单与镜像读数 |
| `internal/extensions/workflows.go` | 工作流类型、定义浏览、启动/停止、运行读数；服务端没有方法时只报能力缺失 |
| `internal/extensions/wakeups.go` | 唤醒读取、取消与状态跟随；取消/过期记录不出现在读数里 |
| `internal/extensions/extensions_test.go` | 注入替身覆盖能力分类、钩子来源、导入勾选、通知跟随、真实方法名归一、失败保留等分支 |
| `internal/extensions/appserver_test.go` | 假服务端 + 真实 NDJSON 的插件/技能、钩子、MCP、计划任务、唤醒、工作流能力缺失证据 |
| `internal/extensions/daemon_evidence_test.go` | 真实 alpha.21 daemon 证据：管理面读取、插件/技能启停、唤醒 create/list/cancel 与通知 |
| `docs/domains/D11b-extensions.md` | 本文件 |

**验收**

- 命令：`go test ./internal/extensions/`、`go test -race ./internal/extensions/`、`go test -count=1 ./...`、`go vet ./...`；本版全部通过（2026-11-11，macOS arm64，Go 1.24.3）。
- 本领域测试：22 个顶层测试（21 个默认通过，1 个真实 daemon 证据按环境变量运行）。
- 真实 daemon：alpha.21，临时数据目录 + 复制到 `/tmp` 的离线模型配置；读取、插件/技能启停、唤醒 create/list/cancel 与通知均通过，见 6.4。
- 证据档位：
  - `fakeappserver`：本机 Unix socket 上真实 NDJSON，覆盖插件/技能/钩子/MCP/计划任务/工作流能力缺失。
  - 注入替身：读取方与写登记替身，覆盖能力缺失与恢复、来源拒绝、导入勾选、失败保留、通知时序、真实方法名归一。
  - 真实 alpha.21：管理面读取、插件与技能启停、唤醒闭环；未覆盖的写与探测见 6.5。

**依赖**

| 依赖 | 需要对方冻结/提供 |
|---|---|
| D1 连接 | 管理面读取方法、通知形状与 `protocol` 类型（已按 alpha.21 落地）；集成层实现本包七个读取接口并把通知映射成 `Notice` |
| D3 写登记 | `Register`/`Wait` 与 `KindManagementWrite`；本包只登记一次，不重试 |
| D12 集成层 | 按 §4.3 映射写意图到真实方法；把 `management.progress` 的 `schedule.watch` 与 `session.notice` 的 Scheduler/wakeup 映射成通知；唤醒方法走会话流 |
| D11a 设置与运行环境 | 运行组件、存储读数、模态归 D11a；alpha.21 的 `runtime.*` 能力缺失由 D11a 同步，本包不提供本地替代 |
| U1 原生界面 | 只读 `Snapshot` 与 `write.IntentView`；通过 `Notify`/`WaitIdle` 驱动刷新 |
| V 验收设施 | 假服务端已用于本领域；真实 alpha.21 证据脚本按环境变量运行 |

**上下文需求**

- 真实钩子条目（`hooks.catalog.hooks[]`）与 `hooks.save` 文档形状尚未冻结；本版只按测试适配器的扁平形状验证来源范围，真实写标 blocked。
- `mcp.test`、`mcp.import.preview`、`mcp.save`、`mcp.remove`、`mcp.import.apply` 的参数已冻结，回执形态未全部探测；本版用测试适配器覆盖方法名与参数，真实写/探测标 blocked。
- 计划任务 `schedule.watch` 通过 `management.progress` 推送；本版只在测试适配器验证映射，真实 watch 标 blocked。
- 插件/技能安装的完整参数（`plugin.install` 的 `id`/`enable`、`skills.install` 的 `skills[]` 勾选与 revision）未在真实服务端闭环；本版只验证真实启停。
- 工作流与运行组件在 alpha.21 没有 wire 方法；工作流类型保留但不发请求，运行组件由 D11a 按能力缺失处理。

## 2. 边界

**做**

| 能力 | 说明 |
|---|---|
| 插件 | 清单（`plugin.list` 数组，含未安装/停用条目）、按本地路径/Git/随包来源安装、启停、更新 |
| 技能 | 清单（`skills.catalog`）、来源、安装、启停；本端不留独立记录 |
| 钩子 | 清单（`hooks.catalog`）、来源区分、只有用户来源可写 |
| MCP | 目录（`mcp.catalog`）、新增/修改/删除、显式探测、导入预览、按勾选逐条写入 |
| 计划任务 | `schedule.list` 清单读取；通知到达时重读一次，无定时轮询 |
| 工作流 | 类型、定义浏览、运行读数；服务端没有 `workflow.*` 时读取与写都报能力缺失，不发请求 |
| 会话级唤醒 | `session.wakeup.list` 读取、`session.wakeup.cancel` 取消、通知后重读一次；取消/过期记录不出现在读数里 |
| 缺能力 | 每个读取入口返回带能力名的 `CapabilityError`；能力恢复后同一实例可再读 |
| 观察 | 每个读数的最近确认值、读取时间、读取错误、能力缺失标记；跟随计数与日志 |

**不做**

| 不做 | 归属/原因 |
|---|---|
| 界面、列表渲染、按钮状态、表单校验提示 | U1 |
| 连接、socket、NDJSON、wire 类型 | D1 |
| 设置项、模型与 provider、权限档位、运行组件、存储、模态 | D11a |
| 账号、登录、用量 | D6 |
| 会话折叠、分支、审批、提问 | D2/D5 |
| 计划任务的执行、补跑、本地调度、启停 | 服务端；本版只读清单 |
| 自动探测 MCP、自动连接外部进程 | R 块 8：探测只由显式调用触发 |
| 为 `workflow.*`/`runtime.*` 发明本地替代 | alpha.21 不注册；按能力缺失处理 |
| 插件市场、外部内容容器 | R 附 C：不做 |
| 自动重试、自动重发、乐观更新 | A 四条总规矩；D3 写登记 |
| 持久化镜像、导入预览、跟随订阅 | R8：这些是可由用户丢弃的本地状态；重启后重读 |

本包只使用 Go 标准库与仓库内已有的错误分类/写登记包，不引第三方依赖，不改 `go.mod`。

## 3. 包结构与对外接口

```text
internal/extensions/
├── extensions.go    Snapshot、错误与 CapabilityError、来源、WriteOp、管理写入参数、
│                    真实方法名归一、Config/New/Close、readInto、写键与 write
├── follow.go        Topic/Notice、FollowStats、Notify/WaitIdle、唯一跟随协程
├── plugins.go       Plugin、PluginReader、ListPlugins、InstallPlugin、SetPluginEnabled、UpdatePlugin
├── skills.go        Skill、SkillReader、ListSkills、InstallSkill、SetSkillEnabled
├── hooks.go         Hook、HookReader、ListHooks、WriteHook
├── toolservers.go   ToolServer*、探测、导入预览、ApplyToolServerImport
├── scheduler.go     ScheduledTask、SchedulerReader、ListScheduledTasks
├── workflows.go     WorkflowDefinition/Run、WorkflowReader、启动/停止/运行读数
├── wakeups.go       Wakeup、WakeupRef、WakeupReader、ListWakeups、CancelWakeup
├── extensions_test.go      替身驱动的领域分支测试
├── appserver_test.go       假服务端 + 真实 NDJSON 的集成证据
└── daemon_evidence_test.go 真实 alpha.21 daemon 证据（缺可执行文件时跳过）
```

对外接口（节选）：

```go
service, err := extensions.New(extensions.Config{
    Plugins:     pluginReader,     // 七个读取方按能力配置；nil 表示本端没有该能力
    Skills:      skillReader,
    Hooks:       hookReader,
    ToolServers: mcpReader,
    Scheduler:   scheduleReader,
    Workflows:   nil,              // alpha.21 不注册 workflow.*：保留字段但不配置
    Wakeups:     wakeupReader,
    Writer:      writeManager,     // *write.Manager 直接满足
    Logf:        logf,
})
defer service.Close()

// 读取：返回最近确认快照与本次错误；失败时保留上次 Items。
plugins, err := service.ListPlugins(ctx)
scheduled, err := service.ListScheduledTasks(ctx)

// 写：全部经写登记；集成层按 §4.3 映射到真实方法。
view, err := service.SetPluginEnabled(ctx, "Attachments", true)
view, err = service.WriteHook(ctx, extensions.Hook{Event: "pre", Source: extensions.SourceUser, ...})
view, err = service.CancelWakeup(ctx, extensions.WakeupRef{SessionID: "s-1", WakeupID: "w-1"})

// MCP 探测与导入：探测只读；应用按勾选逐条写。
probe, err := service.ProbeToolServer(ctx, "server-id")
preview, err := service.PreviewToolServerImport(ctx, "claude-code:user")
result, err := service.ApplyToolServerImport(ctx, "claude-code:user", []string{"cand-b", "cand-c"})

// 事件跟随：集成层把服务端通知映射成 Notice。
service.Notify(extensions.Notice{Topic: extensions.TopicSchedule})
service.Notify(extensions.Notice{Topic: extensions.TopicWakeups, SessionID: "s-1", BranchID: "main"})
err = service.WaitIdle(ctx)
```

## 4. 设计

### 4.1 读镜像与快照

- 每个能力有一份 `Snapshot[T]`：最近一次成功读取的 `Items`、`ReadAt`、最近一次读取的 `ReadError`、`CapabilityMissing`。
- 成功读取整体替换 `Items` 并清除错误；失败读取**保留**上一次确认的 `Items`，只更新错误与能力缺失标记。
- 全部读取共用 `readMu` 串行执行，保证先发出的旧读取不会覆盖后完成的新读取。
- `*Snapshot()` 访问器只返回副本，不发起请求；镜像只由读取与跟随入口更新。

### 4.2 能力缺失分类

`CapabilityError` 带能力名，并同时满足：

```go
errors.Is(err, extensions.ErrCapabilityMissing)
errors.Is(err, &extensions.CapabilityError{Feature: "schedule.list"})
```

识别两类能力缺失：

1. 本端没有配置读取方（`Config` 中对应字段为 nil）；不发请求，直接报缺失并记录日志。
2. 服务端返回稳定的 `appError`：`unknown_method`、`method_not_found`、`capability_missing`、`not_supported`。

超时、连接失败、参数错误与未知错误不会被冒充成能力缺失。`CapabilityError` 保留底层错误，因此 `failure.KindOf` 仍会把它归为不可重试。计划任务能力恢复后再次读取即可得到列表；同一实例不需要重建。

### 4.3 写意图与真实方法

所有变更都经注入的写登记发出，类型固定为 `write.KindManagementWrite`。`WriteOp` 是**判别名，不是 wire 方法名**；集成层按它映射到 alpha.21 实测方法：

| WriteOp（判别名） | alpha.21 实际方法 | 说明 |
|---|---|---|
| `plugin.install` | `plugin.install` | 来源映射成 `{path}` 或 `{git,ref?}` |
| `plugin.set_enabled` | `plugin.enable` / `plugin.disable` | 由载荷 `Enabled` 选择 |
| `plugin.update` | `plugin.update` | `{id,ref?}` |
| `skill.install` | `skills.install` | 参数含来源与勾选技能 |
| `skill.set_enabled` | `skills.set_enabled` | `{id,enabled}` |
| `hook.write` | `hooks.save` | `{expectedRevision,document}`；真实文档形状待冻结 |
| `tool_server.add`（手工） | `mcp.save` | `{id,source,expectedVersion,configuration}` |
| `tool_server.add`（导入） | `mcp.import.apply` | 载荷带 `importSource` 时逐条应用；每次只带一个候选 |
| `tool_server.update` | `mcp.save` | 同手工新增，`id` 为已有服务器 |
| `tool_server.remove` | `mcp.remove` | `{id,source,expectedVersion}` |
| `workflow.start` / `workflow.stop` | 无 | alpha.21 不注册；能力缺失零请求 |
| `wakeup.cancel` | `session.wakeup.cancel` | `{sessionId,wakeupId}`；需要会话标识 |

- `DecodeManagementWrite` 同时接受真实方法名（`skills.*`、`hooks.save`、`mcp.*`、`plugin.enable/disable`、`session.wakeup.cancel`），并把它们归一到上面的判别名与发送方载荷；旧判别名原样通过。
- 幂等键由「判别名 + 载荷摘要」生成。同一内容重复登记不会产生第二次请求；上一次明确失败或确认未发生时换新键允许重试；工作流启动在成功后再次调用也换键。
- 写成功后不自动重读清单：管理面事实只来自读取与变更通知；调用方显式读取或等待通知。
- 状态以 `write.IntentView.Status` 为准；本包不自行重试、不自动补发。

### 4.4 插件与技能

- 插件清单按 `plugin.list` 的数组映射：`id` 取 `name`，`enabled` 取 `!off`，并保留 `use`、`locked`、`status`、`error`；未安装/停用条目不被过滤。
- 安装来源只接受 `local`、`git`、`bundled`；引用不能为空。真实安装参数由集成层映射成 `{path}` 或 `{git,ref?}`。
- 技能清单按 `skills.catalog` 映射：`id`、`source`、`name`、`description`、`enabled`、`installed=true`；启停走 `skills.set_enabled`。
- 本端不留独立记录；重启后重新读取即可得到同一状态。真实证据验证了技能启停往返。

### 4.5 钩子的来源范围

- `Hook.Writable()` 只在来源为用户时返回真；未知来源按只读处理。
- `WriteHook` 在调用写登记**之前**校验来源：项目、插件、未知或空来源一律返回 `ErrSourceNotWritable`，不产生写意图，也不发出任何请求。
- 读取方法按 `hooks.catalog`；写入方法按 `hooks.save`。真实 `hooks.catalog` 的条目形状尚未冻结，本版只在测试适配器里验证扁平条目；真实写标 blocked。

### 4.6 MCP：显式探测与按勾选导入

- 目录读取按 `mcp.catalog`；服务器条目映射 `id`、`source`、`enabled`、`transport`、`auth`。清单读取、新增、修改、删除都不会触发探测；只有 `ProbeToolServer` 会发一次 `mcp.test`。探测结果只是预览，不写回配置。
- 导入先 `PreviewToolServerImport`（`mcp.import.preview`）：读取候选并缓存在内存中，按来源键保存；预览会校验候选键非空且唯一、服务器配置结构合法。
- `ApplyToolServerImport`：
  1. 要求该来源已有预览，否则 `ErrPreviewRequired`；
  2. 要求勾选非空、无重复、每个键都在预览里；任一不合法都不写任何条目；
  3. 按预览顺序为每条被勾选候选登记一条 `tool_server.add`，载荷带 `ImportSource` 与 `CandidateKey`；集成层映射成 `mcp.import.apply`，每次一个候选标识；
  4. 任一条没有明确成功就停止，返回已有结果与 `ErrWriteNotSucceeded`。
- 环境变量只出现在写入参数中；清单只返回键名，不返回值；本包日志不记录参数或环境变量内容。
- 真实证据只验证 `mcp.catalog` 读取；探测、导入与增删改标 blocked（没有真实 MCP 服务器与冻结的回执形状）。

### 4.7 计划任务

- 清单按 `schedule.list` 映射；job 是 snake_case，保留 `revision`、`prompt`、`cwd`、`mode` 等字段。状态与时间由服务端给出，本端不推算下次运行时间。
- 通知由集成层从 `management.progress`（`schedule.watch`）映射成 `Notice{Topic: TopicSchedule}`；一次通知重读一次清单，无定时器、无全量轮询。
- 能力缺失分类并记录日志；能力恢复后同一实例再次读取即可得到列表。

### 4.8 工作流：保留类型，不发请求

- alpha.21 不注册 `workflow.*`；`WorkflowReader`、定义、运行与写意图类型保留，但集成层不配置读取方。
- 读取、启动、停止都返回 `CapabilityError{Feature: "workflow.*"}`，写入口在登记之前返回，不产生写意图。
- 本版不提供本地工作流替代，也不把 Electron 的类型清单当成服务端能力。

### 4.9 会话级唤醒

- 读取按 `session.wakeup.list`；取消按 `session.wakeup.cancel`，参数必须带会话标识（`WakeupRef.SessionID`），因为这是会话贡献方法。
- 真实服务端在取消后仍保留 `cancelled` 状态的记录；A.md 要求「取消后不再出现」，因此本包在读数里过滤 `cancelled` 与 `expired`，保留 `scheduled` 与 `fired`。取消结果由写回执与通知确认。
- 集成层把 `session.notice`（Source `Scheduler`、Kind `wakeup`）映射成 `Notice{Topic: TopicWakeups, SessionID, BranchID}`；通知到达后只重读一次，取消后记录消失。
- `CancelWakeup` 本身不重读；重读由通知或调用方显式读取触发。

### 4.10 运行组件

- `runtime.*` 在 alpha.21 不注册；运行组件归 D11a 设置与运行环境，本包不提供类型、不发请求，也不做本地替代。设置领域按能力缺失同步。

### 4.11 并发与关闭

- 读取、跟随、写意图登记可并发调用；镜像更新与跟随计数由内部互斥保护。
- `Close` 停止跟随协程并拒绝后续读取/写入；跟随中的读取共享可取消上下文。显式读取使用调用方上下文，关闭不会中断已经开始的外部请求，这是本版已知限制。

## 5. 与 A/R 的对应

| 需求或架构条目 | 本领域实现 | 不在本领域 |
|---|---|---|
| R 块 8 插件列表与安装来源 | `plugin.list` 数组；local/git/bundled 安装来源 | 插件市场；插件内部能力 |
| R 块 8 插件启停、更新 | `plugin.enable`/`plugin.disable`/`plugin.update` 写意图 | 插件运行进程与日志 |
| R 块 8 技能安装与启停 | `skills.catalog`、`skills.install`、`skills.set_enabled` | 技能内容执行 |
| R 块 8 钩子写入范围 | `hooks.catalog` 读取；用户来源 `hooks.save`，其余来源零请求 | 钩子执行与审批 |
| R 块 8 MCP 增删改与探测 | `mcp.catalog`、`mcp.save`、`mcp.remove`、显式 `mcp.test` | 外部进程连接实现 |
| R 块 8 导入预览与按勾选写入 | `mcp.import.preview`、逐条 `mcp.import.apply` | 配置文件解析 |
| R 块 9 计划任务列表与变更 | `schedule.list` + 事件后重读一次 | 任务执行、补跑、启停 |
| R 块 9 唤醒读取、取消与状态跟随 | `session.wakeup.list/cancel`、每范围一次重读、取消后过滤 | 定时器触发与提醒展示 |
| R 块 8/9 能力缺失 | `CapabilityError` + 日志 + 恢复后可再读 | 服务端能力实现 |
| alpha.21 工作流/运行组件不存在 | 工作流保留类型零请求；运行组件交 D11a | 服务端能力实现 |
| A 管理面读数只由读取与通知驱动 | 镜像更新只经 `readInto` 与跟随入口 | 界面状态 |
| A 四条总规矩 | 不写事实、不做乐观更新、一个状态一个驱动者、缺能力报缺失 | 其他模块状态 |

## 6. 验收

### 6.1 命令与结果

```bash
go test ./internal/extensions/
go test -race ./internal/extensions/
go test -count=1 ./...
go vet ./...
```

本版结果（2026-11-11，macOS arm64，Go 1.24.3）：

- `go test ./internal/extensions/`：`ok`（22 个顶层测试；21 个默认通过，真实 daemon 证据在无环境变量时跳过）。
- `go test -race ./internal/extensions/`：`ok`。
- `go test -count=1 ./...`：全部包 `ok`（含 `internal/app` 集成层）。
- `go vet ./...`：无输出。

### 6.2 覆盖

| 验收要求 | 测试 |
|---|---|
| 插件清单保留可选/停用未安装条目 | `TestPluginListKeepsOptionalUninstalled`、`TestAppServerPluginsAndSkills` |
| 按来源安装、启停、更新；写入只发一次 | `TestAppServerPluginsAndSkills`、`TestInstallSourceAcceptsEverySupportedKind` |
| 非法安装来源在写之前拒绝 | `TestInvalidInstallSourceIsRejectedBeforeWrite` |
| 技能启停后重读保持；重启客户端后仍保持 | `TestAppServerPluginsAndSkills` |
| 只有用户来源的钩子可写；其他来源不发请求、不登记意图 | `TestHookSourceIsTheOnlyWritableOne`、`TestAppServerHookSourceScope` |
| MCP 目录、增删改、清单不探测 | `TestAppServerToolServersAndImport` |
| 显式探测返回工具预览且不写配置 | `TestAppServerToolServersAndImport` |
| 导入预览 3 条；应用只写勾选条数 | `TestAppServerToolServersAndImport`、`TestImportPreviewSelectsOnlyCheckedCandidates` |
| 未预览、空勾选、未知候选都不写入 | `TestImportPreviewSelectsOnlyCheckedCandidates`、`TestAppServerToolServersAndImport` |
| 计划任务能力缺失分类并记录；恢复后返回列表 | `TestAppServerSchedulerFollowing`、`TestCapabilityMissingIsClassifiedAndRecovers` |
| 计划任务通知后只重读一次、无轮询 | `TestAppServerSchedulerFollowing`、`TestSchedulerFollowsNotificationOnceWithoutPolling` |
| 工作流在 alpha.21 能力缺失且零请求 | `TestAlpha21WorkflowCapabilityMissingMakesNoRequest` |
| 唤醒读取、取消后消失、通知只重读一次 | `TestAppServerWakeupFollowAndCancel`、`TestWakeupFollowsCancellationOnce`、`TestWakeupReadingsHideCancelledAndExpired` |
| 读取失败保留上一次确认事实 | `TestSnapshotKeepsConfirmedFactsOnReadFailure` |
| 十一个读取入口缺能力都带能力名 | `TestEveryCapabilityReportsMissingWhenReaderIsAbsent` |
| 写登记缺失时写入口明确不可用 | `TestWriteRequiresWriter` |
| 失败后用新幂等键重试，不重复发送 | `TestWriteRetriesWithNewKeyAfterFailure` |
| 真实方法名归一到稳定判别名 | `TestDecodeRealMethodNamesNormalizes` |
| wire 层能力缺失不冒充空列表；写入明确失败 | `TestAppServerMissingCapabilityOnWire` |
| 真实 alpha.21 管理面读取与唤醒闭环 | `TestRealDaemonExtensionReadsAndWakeup`（需环境变量） |

### 6.3 证据档位

- **fakeappserver + 真实客户端**：本机 Unix socket、真实 NDJSON 与 `daemon` 握手；wire 形状只出现在 `appserver_test.go` 的适配器里。覆盖七个范围的读取、写入只发一次、通知跟随、能力缺失与恢复。
- **注入替身**：读取方与写登记替身，覆盖时序、来源拒绝、勾选校验、失败保留、真实方法名归一与关闭语义。
- **真实 alpha.21 daemon**：临时数据目录 + `--model-config` 离线配置；验证 `plugin.list`、`skills.catalog`、`hooks.catalog`、`mcp.catalog`、`schedule.list`、`session.wakeup.list/cancel` 与 `session.notice` 唤醒通知。

### 6.4 真实 alpha.21 证据

命令（模型配置复制到 `/tmp`，来源目录只读；数据目录临时）：

```bash
cp ~/Code/dim/crates/dim-agent-aimux/examples/offline.json /tmp/d11b-offline.json
DIM_APP_SERVER_BIN=$HOME/.dimcode-alpha/rust/dimcode-server \
DIM_APP_SERVER_ARGS=app-server \
DIM_APP_SERVER_MODEL_CONFIG=/tmp/d11b-offline.json \
  go test -run TestRealDaemonExtensionReadsAndWakeup -v ./internal/extensions/
```

观察到的输出（节选）：

```text
服务端 0.1.0-alpha.21（protocol 3，profile configured，owned=true）
插件 71 个（样例 [AgentDirectory Agents Aimux Attachments BranchTool]）
技能 32 个
钩子 0 条
MCP 服务器 0 个
计划任务 0 条
extensions: 能力缺失 feature=workflow.list reason=no_reader
extensions: 能力缺失 feature=workflow.start reason=no_reader
唤醒读取：1 条（...）
唤醒通知：state=scheduled
唤醒通知：state=cancelled
唤醒重读计数：map[<session>\x00main:2]
插件 Attachments 启停往返成功：enable=false，恢复=succeeded
技能 user:agents-sdk 启停往返成功：enable=false，恢复=succeeded
回收自启服务端: stopped=true
```

说明：

- 配置 profile 下 `skills.catalog` 可用；mock profile 下该能力不注册，证据必须用离线模型配置，不能用 mock 代替。
- 真实服务端取消唤醒后仍返回 `cancelled` 记录；本包按 A.md 在读数里过滤，因此取消后读取为空。
- 唤醒 create 与 cancel 各触发一次 `session.notice`，重读计数为 2（每次通知一次），没有定时轮询。
- 插件启停可能触发服务端整体 reload；读取证据先完成，写尝试放在最后。

### 6.5 blocked

| 项 | 状态 | 缺什么 |
|---|---|---|
| 真实 `hooks.save` 与钩子条目形状 | blocked | `hooks.catalog.hooks[]` 条目与 `hooks.save` 文档形状未冻结；本版只验证 `hooks.catalog` 空列表与来源范围 |
| 真实 `mcp.test` 探测 | blocked | 没有可连接的外部 MCP 服务器；回执形状未冻结 |
| 真实 `mcp.import.preview` / `mcp.import.apply` | blocked | 导入来源与候选回执形状未冻结；本版只验证参数映射与逐条写入 |
| 真实 `mcp.save` / `mcp.remove` | blocked | 配置文档与回执形状未冻结；本版只验证方法名与参数映射 |
| 真实 `plugin.install` / `skills.install` | blocked | 安装参数虽有实测形状，但没有安全的真实插件/技能源与回执闭环；本版只验证真实启停 |
| 真实 `schedule.watch` 通知 | blocked | 需要真实调度器产生进度事件；本版用 `management.progress` 适配器验证映射 |
| 真实 `workflow.*` / `runtime.*` | blocked | alpha.21 不注册；按能力缺失处理，不发明替代 |
| Windows 命名管道与平台差异 | blocked | 没有 Windows 主机与 Windows 版 App Server；本包本身无平台分支 |

## 7. 已知限制

1. **判别名不是 wire 名**：`WriteOp` 保持稳定判别名，真实方法由集成层映射；`DecodeManagementWrite` 已能接受真实方法名并归一。
2. **列表不分页**：每个清单一次读取返回全部条目；服务端若分页，需要集成层在读取方内合并。
3. **读取串行**：全部管理面读取共用 `readMu`；一个慢能力会拖慢其他能力的读取。
4. **关闭不中断显式读取**：`Close` 只保证跟随协程退出；已开始的外部读取由调用方上下文决定何时结束。
5. **快照浅拷贝**：`Snapshot.Items` 复制切片，但条目内部切片/映射仍可能与内部值共享；调用方必须按只读使用。
6. **能力缺失识别有限**：只识别约定的 `appError` 集合与本端 nil 读取方；服务端新增错误名需要扩展映射表。
7. **写入成功不自动重读**：镜像只由读取与通知更新；界面需要在写终态后显式读取或等待通知。
8. **导入预览在内存**：`ApplyToolServerImport` 依赖同一实例里最近一次预览；重启或换实例后必须重新预览。
9. **写重试靠新键**：同一内容成功过就不会再发；未知与需人工确认不会自动重试。
10. **MCP 环境变量经过写参数**：不会进入快照、日志或错误文本，但会留在写登记的内存参数里；不得在集成层打印。
11. **钩子删除未实现**：本版只有清单与写入（新建/更新）；真实写入形状待冻结。
12. **计划任务只读**：本版只做清单与跟随，不提供启停、立即运行或补跑。
13. **唤醒过滤**：读数过滤 `cancelled`/`expired`，保留 `scheduled`/`fired`；历史状态仍可从写回执与通知观察。
14. **工作流无运行时**：类型保留，alpha.21 下任何工作流读取/启动/停止都报能力缺失且零请求。
15. **真实安装未闭环**：插件/技能安装只在 fakeappserver 与单元测试里验证；真实服务端只验证启停。
16. **mcp.save 原始配置**：真实方法名归一后保留 `configuration` 文档，但当前集成层按领域摘要构造最小配置；完整配置文档透传需要 D12 继续补字段。
17. **平台**：只在 macOS arm64 上跑过测试；Windows 未验证。

## 8. 下一步

1. 冻结 `hooks.catalog.hooks[]` 与 `hooks.save` 文档形状，补真实钩子写入证据与删除语义。
2. 准备一个安全的临时 MCP 服务器，补 `mcp.test`、`mcp.save`、`mcp.remove`、`mcp.import.preview/apply` 的真实闭环。
3. 用真实调度器补 `schedule.watch` → `management.progress` 的通知跟随证据。
4. 补真实插件/技能安装夹具，验证安装参数与回执闭环。
5. 评估列表分页与按能力分锁；当前一页读取足够第一版。
6. 与 D11a 对齐运行组件能力缺失的表达，避免两处对 `runtime.*` 给出不同结论。
