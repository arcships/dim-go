# D11b 扩展与后台（插件、技能、钩子、MCP、计划任务、工作流、唤醒）

Status: 第一版实现完成（alpha.21 实测方法 + hooks/MCP 真实回执 + 夜间授权三项，待组织评审）
Owner: 扩展与后台领域 agent
验证: 可执行命令与覆盖见第 6 节；真实 alpha.21 证据见 6.4（含 hooks/MCP 回执与夜间授权），blocked 见 6.5
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
| 夜间授权补充项 | `docs/domains/R.md` 附 E 第 12–14 项 | 钩子启停、技能刷新、MCP 认证 |
| Scheduler 默认预设 | 随包默认插件 `scheduler`（`<bundled-default-plugins>`），复制到 `<tmp>` 后通过 `--defaults` 装配 | 真实 schedule.* 与 watch 证据 |
| 管理面读数、四条总规矩 | `docs/domains/A.md` | 每项读数只有一个驱动者；只由读取与变更通知驱动；缺能力报缺失；不做乐观更新；不保存第二份事实 |
| alpha.21 管理面实测 | `docs/domains/D1-connection.md` §3.8、§6.5 | 真实方法名：`plugin.*`、`skills.*`、`hooks.*`、`mcp.*`、`schedule.*`、`session.wakeup.*`；`workflow.*`、`runtime.*` 在 alpha.21 不注册 |
| D1 管理面类型 | `internal/appserver/protocol/management_*.go`、`notices.go` | 集成层按这些类型映射；本包不复制 wire 结构 |
| 连接错误分类 | `docs/domains/D1-connection.md`、`internal/appserver/failure` | 能力缺失归一成 `CapabilityError`；超时、连接失败、参数错误原样上报 |
| 写登记 | `docs/domains/D3-write.md`、`internal/write` | 所有变更登记为 `KindManagementWrite`，一次发送、不自行重试、结果由写登记收敛 |
| 假服务端与真实服务端 | `internal/fakeappserver`、`internal/appserver/protocol`、`internal/appserver/daemon`、`<app-server-binary>`（0.1.0-alpha.21） | 本机 Unix socket + 真实 NDJSON；假服务端覆盖全部分支，真实服务端覆盖读取、插件/技能启停、唤醒闭环、hooks 写回/冲突、MCP 保存/配置/探测/删除与 Scheduler 预设下的 schedule.watch |
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
| `internal/extensions/daemon_evidence_hooks_mcp_test.go` | 真实 alpha.21 证据：hooks.save 写回与版本冲突、mcp.save/configuration/test/remove |
| `internal/extensions/daemon_evidence_night_test.go` | 真实 alpha.21 证据：项目钩子启停/冲突、skills.refresh/reload_catalog、mcp.auth.* 方法接入 |
| `internal/extensions/daemon_evidence_mcp_import_test.go` | 真实 alpha.21 证据：MCP 导入来源、preview/apply 形状、逐条写入与冲突 |
| `internal/extensions/daemon_evidence_schedule_test.go` | 真实 alpha.21 证据：Scheduler 默认预设、schedule.watch 序号、list/history 重读、删除与缺口 |
| `docs/domains/D11b-extensions.md` | 本文件 |

**验收**

- 命令：`go test ./internal/extensions/`、`go test -race ./internal/extensions/`、`go test -count=1 ./...`、`go vet ./...`；本版全部通过（2026-11-11，macOS arm64，Go 1.24.3）。
- 本领域测试：35 个顶层测试（30 个默认通过，5 个真实 daemon 证据按环境变量运行）。
- 真实 daemon：alpha.21，临时数据目录 + 离线模型配置；读取、插件/技能启停、唤醒闭环、hooks 写回/冲突、MCP 保存/配置/探测/删除、项目钩子启停/冲突、skills.refresh/reload_catalog、mcp.auth.* 方法接入、MCP 配置导入与 schedule.watch 跟随均通过，见 6.4。
- 证据档位：
  - `fakeappserver`：本机 Unix socket 上真实 NDJSON，覆盖插件/技能/钩子/MCP/计划任务/工作流能力缺失。
  - 注入替身：读取方与写登记替身，覆盖能力缺失与恢复、来源拒绝、导入勾选、失败保留、通知时序、真实方法名归一。
  - 真实 alpha.21：管理面读取、插件与技能启停、唤醒闭环；未覆盖的写与探测见 6.5。

**依赖**

| 依赖 | 需要对方冻结/提供 |
|---|---|
| D1 连接 | 管理面读取方法、通知形状与 `protocol` 类型（已按 alpha.21 落地）；集成层实现本包七个读取接口并把通知映射成 `Notice` |
| D3 写登记 | `Register`/`Wait` 与 `KindManagementWrite`；本包只登记一次，不重试 |
| D12 集成层 | 按 §4.3 映射写意图到真实方法；实现 `MCPImports` 读取方（来源与真实 preview）；把 `management.progress` 的 `schedule.watch` 与 `session.notice` 的 Scheduler/wakeup 映射成通知；唤醒方法走会话流 |
| D11a 设置与运行环境 | 运行组件、存储读数、模态归 D11a；alpha.21 的 `runtime.*` 能力缺失由 D11a 同步，本包不提供本地替代 |
| U1 原生界面 | 只读 `Snapshot` 与 `write.IntentView`；通过 `Notify`/`WaitIdle` 驱动刷新 |
| V 验收设施 | 假服务端已用于本领域；真实 alpha.21 证据脚本按环境变量运行 |

**上下文需求**

- 真实 `hooks` 已冻结本次用到的部分：`hooks.save` 文档 `{hooks:{event:[{matcher,hooks:[{type,command}]}]}}`，回执 `{revision}`，旧版本返回 `hooks_conflict` 且不覆盖；`hooks.catalog.hooks[]` 是扁平条目，无 `id`，本端按来源/事件/位置派生稳定标识。更多钩子字段（args、if、commandWindows）待补。
- 真实 `mcp.save/remove/test/configuration` 已按本地无凭据 stdio 夹具核对；原始配置文档、HTTP/SSE 与鉴权服务器仍待真实证据。`mcp.import.sources/preview/apply` 已在本机真实来源上跑通，领域 API 已按真实形状落地；旧便捷方法保留为薄包装。
- 计划任务 `schedule.watch` 通过 `management.progress` 推送；真实 Scheduler 默认预设下已验证 watching/jobs/delete 通知与重读。真实 run 通知未触发（任务安排在几分钟后），history 重读用 run 型通知驱动验证。
- 插件/技能安装的完整参数（`plugin.install` 的 `id`/`enable`、`skills.install` 的 `skills[]` 勾选与 revision）未在真实服务端闭环；本版只验证真实启停。
- 工作流与运行组件在 alpha.21 没有 wire 方法；工作流类型保留但不发请求，运行组件由 D11a 按能力缺失处理。

## 2. 边界

**做**

| 能力 | 说明 |
|---|---|
| 插件 | 清单（`plugin.list` 数组，含未安装/停用条目）、按本地路径/Git/随包来源安装、启停、更新 |
| 技能 | 清单（`skills.catalog`）、来源、安装、启停；本端不留独立记录 |
| 钩子 | 清单（`hooks.catalog`）、来源区分、只有用户来源可写、项目作用域启停（`hooks.set_enabled`）与读回 |
| MCP | 目录（`mcp.catalog`）、新增/修改/删除、显式探测、导入预览、按勾选逐条写入、认证开始/完成/取消/登出 |
| 技能刷新 | `skills.refresh` 触发目录刷新并观察版本变化；`skills.reload_catalog` 按 hello 门禁，缺能力零请求 |
| 计划任务 | `schedule.list` 清单与 `schedule.history` 历史读取；`schedule.watch` 的 watching/jobs/run、序号缺口与去重；无定时轮询 |
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
    MCPImports:  mcpImportReader,  // 读取来源与真实 preview；apply 走写登记
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

`Config.Features` 接入 hello 能力声明：新增的项目钩子启停、技能刷新/重载与 MCP 认证入口先查 hello，声明缺失即返回能力缺失且零请求；`Features` 为 nil 时不做前置门禁，交给调用结果分类。alpha.21 的 hello 声明 `hooks.set_enabled`、`skills.refresh` 与 `mcp.auth.*`，未声明 `skills.reload_catalog`。

### 4.3 写意图与真实方法

所有变更都经注入的写登记发出，类型固定为 `write.KindManagementWrite`。`WriteOp` 是**判别名，不是 wire 方法名**；集成层按它映射到 alpha.21 实测方法：

| WriteOp（判别名） | alpha.21 实际方法 | 说明 |
|---|---|---|
| `hooks.set_enabled` | `hooks.set_enabled` | `{cwd,enabled,expectedRevision}`；项目作用域 |
| `skills.refresh` | `skills.refresh` | 无参数；回执 `{version,issues}` |
| `skills.reload_catalog` | `skills.reload_catalog` | alpha.21 hello 未声明：能力缺失零请求 |
| `mcp.auth.start` / `.complete` / `.cancel` / `.logout` | 同名 | start `{id,source,cwd?,clientSecret?}`；complete `{attemptId,callbackUrl}`；cancel `{attemptId}`；logout `{id,source,cwd?}` |
| `plugin.install` | `plugin.install` | 来源映射成 `{path}` 或 `{git,ref?}` |
| `plugin.set_enabled` | `plugin.enable` / `plugin.disable` | 由载荷 `Enabled` 选择 |
| `plugin.update` | `plugin.update` | `{id,ref?}` |
| `skill.install` | `skills.install` | 参数含来源与勾选技能 |
| `skill.set_enabled` | `skills.set_enabled` | `{id,enabled}` |
| `hook.write` | `hooks.save` | `{expectedRevision,document}`；文档用 `HookDocument` 生成真实嵌套形状 |
| `tool_server.add`（手工） | `mcp.save` | `{id,source,expectedVersion,configuration}` |
| `mcp.import.apply` | `mcp.import.apply` | `{cwd?,source,from?|document?,expectedVersion,expectedSourceRevision?,resolutions[]}`；逐条动作，不再逐条 `tool_server.add` |
| `tool_server.update` | `mcp.save` | 同手工新增，`id` 为已有服务器 |
| `tool_server.remove` | `mcp.remove` | `{id,source,expectedVersion}`；`RemoveToolServerWithVersion` 带目录版本 |
| `workflow.start` / `workflow.stop` | 无 | alpha.21 不注册；能力缺失零请求 |
| `wakeup.cancel` | `session.wakeup.cancel` | `{sessionId,wakeupId}`；需要会话标识 |

- `DecodeManagementWrite` 同时接受真实方法名（`skills.*`、`hooks.save`、`mcp.*`、`plugin.enable/disable`、`session.wakeup.cancel`），并把它们归一到上面的判别名与发送方载荷；旧判别名原样通过。
- `HookDocument` 与 `ToolServerConfiguration` 生成 alpha.21 实际接受的文档形状；集成层已改用这两个构造函数，`mcp.remove` 也带上版本前置条件。
- 幂等键由「判别名 + 载荷摘要」生成。数据型写入成功后同一内容不重复；项目钩子启停、技能刷新、MCP 认证这类命令型操作在成功后再次调用会换新键，保证旧版本冲突与重复认证尝试都能真实发出。上一次明确失败或确认未发生时同样换新键。
- 写成功后不自动重读清单：管理面事实只来自读取与变更通知；调用方显式读取或等待通知。
- 状态以 `write.IntentView.Status` 为准；本包不自行重试、不自动补发。

### 4.4 插件与技能

- 插件清单按 `plugin.list` 的数组映射：`id` 取 `name`，`enabled` 取 `!off`，并保留 `use`、`locked`、`status`、`error`；未安装/停用条目不被过滤。
- 安装来源只接受 `local`、`git`、`bundled`；引用不能为空。真实安装参数由集成层映射成 `{path}` 或 `{git,ref?}`。
- 技能清单按 `skills.catalog` 映射：`id`、`source`、`name`、`description`、`enabled`、`installed=true`；启停走 `skills.set_enabled`。
- `skills.catalog` 的 `version` 与 `issues` 由实现 `SkillCatalogReader` 的读取方一并带回；`ListSkills` 后可用 `SkillCatalogVersion` 观察版本。真实证据中刷新前版本 0、刷新后 1，条目数不变。
- `skills.refresh` 是命令型写意图，经写登记只发一次；回执 `{version,issues}`，刷新后调用 `ListSkills` 即可观察条目与版本变化。
- `skills.reload_catalog` 已按真实方法名接入，但 alpha.21 的 hello 未声明该能力；入口直接返回能力缺失且零请求。
- 本端不留独立记录；重启后重新读取即可得到同一状态。真实证据验证了技能启停往返与刷新版本变化。

### 4.5 钩子的来源范围、写回与冲突

- `Hook.Writable()` 只在来源为用户时返回真；未知来源按只读处理。
- `WriteHook` 在调用写登记**之前**校验来源：项目、插件、未知或空来源一律返回 `ErrSourceNotWritable`，不产生写意图，也不发出任何请求。
- 读取按 `hooks.catalog`：`userRevision` 是写入前置版本；`hooks[]` 是扁平条目，字段为 `source`、`event`、`matcher`、`command`、`timeout`、`onFailure`、`async`、`state`，没有 `id`，本端按来源/事件/位置派生稳定标识。
- 写入按 `hooks.save`：文档形状是 `{"hooks": {<event>: [{"matcher": <string|null>, "hooks": [{"type":"command", "command": ...}]}]}}`，由 `HookDocument` 生成；`WriteHookWithRevision` 把 `userRevision` 放进 `expectedRevision`。
- 真实回执是 `{"revision":"native:user/hooks:N"}`。旧版本再次写入返回 `appError=hooks_conflict`；写登记把它收敛成明确失败，文档不被覆盖。调用方检查 `write.IntentView.Status` 与 `Reason`。真实写入应使用 `WriteHookWithRevision` 并传入最近一次 `userRevision`；不带版本可能被服务端拒绝或进入冲突。
- 项目作用域：`hooks.catalog {cwd}` 返回项目来源状态 `{source:"project",path,enabled,state}`；`SetProjectHooksEnabled(cwd,enabled,projectsRevision)` 走 `hooks.set_enabled`，回执 `{revision}`。真实证据：初始 false → 启用 true（revision 1→2）→ 停用 false（revision 2→3）；用初始 revision 再启用返回 `hooks_conflict`，读回仍为 false。

### 4.6 MCP：保存、配置、探测、删除与导入

- 目录读取按 `mcp.catalog`：`userRevision` 是保存/删除的前置版本；`servers[]` 形如 `{id,source,enabled,transport,referencedVariables,unsetVariables}`，真实回执没有 name/command/url，本端用 `id` 作为显示名，详细配置走 `mcp.configuration`。
- 保存按 `mcp.save`：参数 `{id,source,expectedVersion,configuration}`。配置文档由 `ToolServerConfiguration` 生成：stdio 只带 `command/args/env`，HTTP/SSE 只带 `url/env`，空字段必须省略；否则服务端返回 `mcp_invalid_config`。回执是空对象 `{}`。
- 读回配置按 `mcp.configuration`：回执 `{revision,enabled,secretsInUserScope,configuration:{type,command,argsCount,cwd,envNames,referencedVariables,unsetVariables}}`。`envNames` 只是键名，不含值。
- 探测按 `mcp.test`：参数 `{id,source}`，回执 `{status:"connected",latencyMs,tools:[{name,description,inputSchema}]}`；探测只在显式调用时发生，不写回配置。
- 删除按 `mcp.remove`：参数 `{id,source,expectedVersion}`，回执空对象 `{}`；`RemoveToolServerWithVersion` 带目录版本。
- 环境变量只出现在写入参数中；清单只返回键名，不返回值；本包日志不记录参数或环境变量内容。
- 认证按真实方法接入：`mcp.auth.start {id,source,cwd?,clientSecret?}` 回执含 `{attemptId,authorizationUrl,redirectUri,resource,issuer,expiresAt}`；`mcp.auth.complete {attemptId,callbackUrl}`、`mcp.auth.cancel {attemptId}`、`mcp.auth.logout {id,source,cwd?}`（回执 `{revoked}`）。这些入口先查 hello 能力，缺声明即能力缺失零请求。
- 真实证据用无凭据的本地 Python stdio 服务器覆盖保存/配置/探测/删除；认证对 stdio 服务器返回 `mcp_auth_unsupported`，complete/cancel 对不存在的 attempt 返回 `mcp_auth_attempt_closed`。成功认证需要真实 OAuth 认证服务器，仍 blocked。

#### 4.6.1 MCP 配置导入：真实来源、回执与领域 API

**服务端扫描的已知来源**（alpha.21 实测 + 同版本上游源码 `mcp/discovery.rs`）：

| 来源 id | client | scope | 路径 | 条件 |
|---|---|---|---|---|
| `claude-code:user` | claude-code | user | `<claude-user-config>` | 读取顶层 `mcpServers` |
| `claude-code:local` | claude-code | local | `<claude-user-config>` 的 `projects[<authorized-workspace>]` | 需要已授权工作区 |
| `codex:user` | codex | user | `<codex-user-config>` | 读取 `mcp_servers` 表 |
| `codex:project` | codex | project | `<codex-project-config>` | 需要已授权工作区 |
| `omp:user` | omp | user | `<omp-user-config>` | 读取 `mcpServers`（或等价表） |
| `omp:project` | omp | project | `<omp-project-config>` | 需要已授权工作区 |

- `mcp.import.sources` 参数 `{cwd?}`；回执 `{sources:[{id,client,scope,path,state,entries?,error?}]}`。`state` 为 `ready` / `absent` / `unreadable` / `too-large` / `malformed`；单文件上限 32 MiB，条目上限由导入模块控制；**只读**，不返回值。
- `mcp.import.preview` 参数 `{cwd?,source,from?|document?}`：`source` 是写入目标作用域（`user`/`project`），`from` 是上面的来源 id；`document` 用于客户端自带的 JSON 文本，二者只能给一个。回执：
  ```text
  {revision, entries:[{id,transport?,status,changes:[{field,kind}],unsupported:[{field,effect}],omitted?,error?}],
   unknownTopLevel:[...], sourceRevision?, store?:{id,client,scope,path}}
  ```
  `status` 为 `identical` / `changed` / `conflict` / `invalid`；`changes` 只给字段名与类型，不给值；`unsupported.effect` 为 `kept-unused` / `dropped`；`sourceRevision` 是来源文档摘要，用于 apply 时确认来源未变。
- `mcp.import.apply` 参数 `{cwd?,source,from?|document?,expectedVersion,expectedSourceRevision?,resolutions:[{id,action,newId?}]}`：`action` 为 `import` / `skip` / `replace` / `rename`；`expectedVersion` 是目标 MCP 目录版本，`expectedSourceRevision` 来自预览。回执 `{revision}`；目标已变化返回 `mcp_settings_conflict`，来源已变化返回同名冲突；逐条动作在 `resolutions` 里表达，一次写入选择过的条目，不是每条一次 `mcp.save`。
- **领域 API**：`MCPImportReader` 提供 `ReadMCPImportSources` 与 `PreviewMCPImport`；`ApplyMCPImport` 经写登记只发一次 `mcp.import.apply`，`resolutions` 逐条动作、目标或来源冲突返回 `ErrImportConflict` 且不重写。旧的 `PreviewToolServerImport(source)` / `ApplyToolServerImport(source, keys)` 保留为薄包装：内部按 `SourceUser` + `From=source` 转真实请求，并把真实预览转回扁平候选；旧包装不表达 project 目标或 `document`，新代码应使用真实 API。

### 4.7 计划任务与 schedule.watch

- 清单按 `schedule.list` 映射；job 是 snake_case，保留 `revision`、`prompt`、`cwd`、`mode` 等字段。状态与时间由服务端给出，本端不推算下次运行时间。
- `schedule.watch` 是长驻调用，进度以 `management.progress {method:"schedule.watch", data}` 推送；`data` 为 `{seq,kind}`，`kind` 取 `watching` / `jobs` / `run`；`run` 项另带 `jobId`、`runId`、`state`、`terminal`。集成层把它们映射成 `Notice{Topic: TopicSchedule, Sequence, SequenceSet, Kind, JobID}`。
- 跟随规则：`watching` 后读一次 list；`jobs` 变化读一次 list；`run` 通知读一次 list 并读一次该任务的 `schedule.history`；序号缺口（`next > last+1`）记录日志并重读；同一序号重复到达不重读。没有定时器、没有全量轮询。
- `ListScheduleHistory` 在读取方实现 `ScheduleHistoryReader` 时可用；否则按能力缺失处理。
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
| 夜间授权第 13 项 钩子启停 | `hooks.set_enabled` 项目作用域启停、读回与冲突 | 钩子执行与审批 |
| 夜间授权第 14 项 技能刷新 | `skills.refresh` 版本观察；`skills.reload_catalog` 能力缺失零请求 | 技能内容执行 |
| 夜间授权第 12 项 MCP 认证 | `mcp.auth.start/complete/cancel/logout`、hello 门禁与敏感输入不落日志 | 真实 OAuth 认证服务器 |
| R 块 8 MCP 增删改与探测 | `mcp.catalog`、`mcp.save`、`mcp.remove`、显式 `mcp.test` | 外部进程连接实现 |
| R 块 8 导入预览与按勾选写入 | `mcp.import.sources/preview/apply`；`resolutions` 逐条动作、冲突不重写 | 配置文件解析 |
| R 块 9 计划任务列表与变更 | `schedule.list/history` + `schedule.watch` 序号跟随、缺口重读、无轮询 | 任务执行、补跑、启停 |
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

- `go test ./internal/extensions/`：`ok`（35 个顶层测试；30 个默认通过，5 个真实 daemon 证据在无环境变量时跳过）。
- `go test -race ./internal/extensions/`：`ok`。
- `go test ./internal/app`：`ok`（集成层）。
- `go vet ./internal/extensions/`：无输出。
- `go test -count=1 ./...`：全部包通过。
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
| 十二个读取入口缺能力都带能力名 | `TestEveryCapabilityReportsMissingWhenReaderIsAbsent` |
| 写登记缺失时写入口明确不可用 | `TestWriteRequiresWriter` |
| 失败后用新幂等键重试，不重复发送 | `TestWriteRetriesWithNewKeyAfterFailure` |
| 真实方法名归一到稳定判别名 | `TestDecodeRealMethodNamesNormalizes` |
| 钩子文档符合 alpha.21 嵌套形状 | `TestHookDocumentMatchesAlpha21Shape` |
| 钩子写入携带 revision 与文档 | `TestWriteHookWithRevisionCarriesVersionAndDocument` |
| MCP 配置省略不兼容字段 | `TestToolServerConfigurationOmitsIncompatibleFields` |
| wire 层能力缺失不冒充空列表；写入明确失败 | `TestAppServerMissingCapabilityOnWire` |
| 真实 alpha.21 管理面读取与唤醒闭环 | `TestRealDaemonExtensionReadsAndWakeup`（需环境变量） |
| 真实 hooks 写回/冲突、MCP 保存/配置/探测/删除 | `TestRealDaemonHooksAndMCPWrites`（需环境变量） |
| 项目钩子启停与状态读回 | `TestSkillCatalogVersionAndProjectHooksRead`、`TestRealDaemonNightAuthorization`（需环境变量） |
| 项目钩子版本冲突不覆盖 | `TestRealDaemonNightAuthorization`（需环境变量） |
| 技能目录版本与问题清单 | `TestSkillCatalogVersionAndProjectHooksRead` |
| skills.refresh 版本变化、reload_catalog 缺能力零请求 | `TestRealDaemonNightAuthorization`（需环境变量） |
| hello 能力门禁、缺能力零写意图 | `TestNewCapabilitiesGateOnHello` |
| 新增写入口的载荷与只发一次 | `TestNewManagementWritesCarryPayloads` |
| MCP 认证方法接入与无认证服务器错误 | `TestRealDaemonNightAuthorization`（需环境变量） |
| MCP 导入来源扫描、preview/apply 形状、逐条写入与冲突 | `TestRealDaemonMCPImport`（需环境变量） |
| 导入来源状态、preview 字段与 unknownTopLevel 透传 | `TestMCPImportSourcesAndPreviewMapping` |
| 导入旧版本冲突返回 ErrImportConflict 且不重写 | `TestMCPImportApplyConflict` |
| 旧便捷方法转真实形状与只应用勾选条数 | `TestImportPreviewSelectsOnlyCheckedCandidates`、`TestAppServerToolServersAndImport` |

### 6.3 证据档位

- **fakeappserver + 真实客户端**：本机 Unix socket、真实 NDJSON 与 `daemon` 握手；wire 形状只出现在 `appserver_test.go` 的适配器里。覆盖七个范围的读取、写入只发一次、通知跟随、能力缺失与恢复。
- **注入替身**：读取方与写登记替身，覆盖时序、来源拒绝、勾选校验、失败保留、真实方法名归一与关闭语义。
- **真实 alpha.21 daemon**：临时数据目录 + `--model-config` 离线配置；验证 `plugin.list`、`skills.catalog/refresh/reload_catalog`、`hooks.catalog/save/set_enabled`、`mcp.catalog/save/configuration/test/remove/auth.*`、`schedule.list`、`session.wakeup.list/cancel` 与 `session.notice` 唤醒通知。

### 6.4 真实 alpha.21 证据

两组证据都使用临时数据目录；模型配置从只读来源复制到临时目录后交给服务端。日志只留状态、标识、字段名与布尔结果，不打印文档内容、命令环境变量或任何凭据。

#### 6.4.1 管理面读取与唤醒

```bash
cp <offline-model-config-source> <offline-model-config>
DIM_APP_SERVER_BIN=<app-server-binary> \
DIM_APP_SERVER_ARGS=app-server \
DIM_APP_SERVER_MODEL_CONFIG=<offline-model-config> \
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

#### 6.4.2 hooks 写回/冲突与 MCP 保存/配置/探测/删除

```bash
DIM_APP_SERVER_BIN=<app-server-binary> \
DIM_APP_SERVER_ARGS=app-server \
DIM_APP_SERVER_MODEL_CONFIG=<offline-model-config> \
  go test -run TestRealDaemonHooksAndMCPWrites -v ./internal/extensions/
```

观察到的输出（节选；`...` 表示省略的临时标识）：

```text
hooks.catalog: userRevision=native:user/hooks:1 userDocumentKeys=[disableAllHooks hooks] hooks=0
hooks.save 回执字段=[revision] revision=native:user/hooks:2
hooks.catalog 写后: userRevision=native:user/hooks:2 hooks=1
读回钩子: source=user event=SessionStart state=ok
领域钩子读取: hooks=1 writable=true
hooks.save 旧版本: status=failed conflict=true
冲突后 hooks.catalog: userRevision=native:user/hooks:2 commands=[<hook-command>]
mcp.catalog: userRevision=native:user/mcp:1 servers=0
mcp.save 回执字段=[]
读回 MCP: id=<test-server-id> source=user enabled=true transport=stdio
mcp.configuration: revision=native:user/mcp:2 enabled=true secretsInUserScope=false configurationFields=[type command argsCount cwd envNames referencedVariables unsetVariables] type=stdio argsCount=1 envNames=0
mcp.test: status=connected reachable=true latencyMs=25 tools=[echo]
mcp.remove 回执字段=[]
mcp.remove 后 servers=0
回收自启服务端: stopped=true
```

回执与语义（alpha.21 实测）：

| 方法 | 参数要点 | 回执形状 |
|---|---|---|
| `hooks.catalog` | 无 | `{userRevision, userDocument, projectsRevision, disableAllHooks, sources[], hooks[]}`；`hooks[]` 条目是扁平对象，无 `id` |
| `hooks.save` | `{expectedRevision, document:{hooks:{<event>:[{matcher,hooks:[{type,command,...}]}]}}}` | `{revision}`；旧版本返回 `appError=hooks_conflict`，文档不覆盖 |
| `mcp.catalog` | 无 | `{userRevision, projectRevision, userApplication, pluginApplication, projectApplication, servers[], invalid[]}`；server 含 `{id,source,enabled,transport,referencedVariables,unsetVariables}` |
| `mcp.save` | `{id,source,expectedVersion,configuration}`；stdio 不带 url，http/sse 不带 command | 空对象 `{}` |
| `mcp.configuration` | `{id,source}` | `{revision,enabled,secretsInUserScope,configuration:{type,command,argsCount,cwd,envNames,referencedVariables,unsetVariables}}` |
| `mcp.test` | `{id,source}` | `{status:"connected",latencyMs,tools:[{name,description,inputSchema}]}` |
| `mcp.remove` | `{id,source,expectedVersion}` | 空对象 `{}` |

MCP 夹具是无凭据、只响应 `initialize` 与 `tools/list` 的本地 Python stdio 进程；它不访问网络、不读写外部文件。

#### 6.4.3 夜间授权三项：项目钩子、技能刷新、MCP 认证

```bash
DIM_APP_SERVER_BIN=<app-server-binary> \
DIM_APP_SERVER_ARGS=app-server \
DIM_APP_SERVER_MODEL_CONFIG=<offline-model-config> \
  go test -run TestRealDaemonNightAuthorization -v ./internal/extensions/
```

观察到的输出（节选）：

```text
服务端 0.1.0-alpha.21（profile configured）
hello hooks.set_enabled=true
hello skills.refresh=true
hello skills.reload_catalog=false
hello mcp.auth.start=true
hello mcp.auth.complete=true
hello mcp.auth.cancel=true
hello mcp.auth.logout=true
项目钩子初始: projectsRevision=native:user/hooks-projects:1 enabled=false sources=2
项目钩子启用后: projectsRevision=native:user/hooks-projects:2 enabled=true
项目钩子停用后: projectsRevision=native:user/hooks-projects:3 enabled=false
项目钩子旧版本: status=failed conflict=true 当前enabled=false
技能刷新前: version=0 entries=32
skills.refresh: replyVersion=1 issues=0; catalog version=0 -> 1 entries=32
skills.reload_catalog: capability_missing=true zero_request=true
mcp.auth.start: status=failed reason=mcp_auth_unsupported（无 OAuth 认证服务器）
mcp.auth.complete: status=failed reason=...mcp_auth_attempt_closed...
mcp.auth.cancel: status=failed reason=...mcp_auth_attempt_closed...
mcp.auth.logout: status=failed reason=mcp_auth_unsupported
回收自启服务端: stopped=true
```

说明：

- 项目钩子状态通过 `hooks.catalog {cwd}` 的 `sources[]` 读回；启用/停用分别推进 `projectsRevision`，旧 revision 写入返回 `hooks_conflict` 且状态不变。
- 技能刷新回执版本 1；随后 `skills.catalog` 版本从 0 变为 1，条目数不变。
- `skills.reload_catalog` 的 hello 声明为 false，领域入口按能力缺失处理，不登记写意图、不发请求。
- MCP 认证成功链路需要真实 OAuth 认证服务器；本机只有无凭据 stdio 夹具，实测到 `mcp_auth_unsupported` 与 `mcp_auth_attempt_closed` 两类真实错误，成功流程标 blocked。

#### 6.4.4 MCP 配置导入：来源、预览、应用与冲突

```bash
DIM_APP_SERVER_BIN=<app-server-binary> \
DIM_APP_SERVER_ARGS=app-server \
DIM_APP_SERVER_MODEL_CONFIG=<offline-model-config> \
  go test -run TestRealDaemonMCPImport -v ./internal/extensions/
```

观察到的输出（节选；路径已替换为占位，不打印任何配置值）：

```text
服务端 0.1.0-alpha.21（profile configured）
import.source id=claude-code:user client=claude-code scope=user state=ready entries=1 path=<claude-user-config>
import.source id=claude-code:local client=claude-code scope=local state=absent entries=-1 path=<claude-user-config> · projects[<authorized-workspace>]
import.source id=codex:user client=codex scope=user state=ready entries=4 path=<codex-user-config>
import.source id=codex:project client=codex scope=project state=absent entries=-1 path=<codex-project-config>
import.source id=omp:user client=omp scope=user state=absent entries=-1 path=<omp-user-config>
import.source id=omp:project client=omp scope=project state=absent entries=-1 path=<omp-project-config>
preview revision=native:user/mcp:1 sourceRevision=<sha256> store=claude-code:user entries=1 unknownTopLevel=[]
preview.entry id=<source-entry-id> status=changed transport=stdio changes=[args:array command:string enabled:boolean type:string] unsupported=0 omitted=0
apply revision=native:user/mcp:2 imported=<source-entry-id>
catalog.server id=<source-entry-id> source=user enabled=false transport=stdio
apply stale err=MCP 导入目标已变化 conflict=true（底层 appError=mcp_settings_conflict）
cleanup mcp.remove shape={}
回收自启服务端: stopped=true
```

判定条件：

- 来源扫描成功：`state=ready` 且 `entries>0`；路径与上表一致；只读来源文件。
- 预览成功：`revision`、`sourceRevision`、`store` 非空；条目带 `status` 与 `changes` 字段名/类型，不含值。
- 逐条写入成功：只应用一条 `resolutions`，apply 后 `mcp.catalog` 只多该条目，`revision` 推进。
- 冲突成功：同一 `expectedVersion` 再次 apply 返回 `mcp_settings_conflict`，目录不重复写入。
- 清理成功：`mcp.remove` 使用 apply 后的 `userRevision` 删除导入条目，回执 `{}`；来源文件未被修改。

#### 6.4.5 计划任务 watch（Scheduler 默认预设）

预设与运行时都在 `<tmp>` 与只读安装目录中；没有复制插件源码进仓库。准备夹具：

```bash
# 随包默认插件目录（只读来源；路径用占位）
rm -rf <tmp>/scheduler-plugin
cp -R <bundled-default-plugins>/scheduler <tmp>/scheduler-plugin
cat > <defaults-file> <<'JSON'
{"plugins":{"scheduler":"<tmp>/scheduler-plugin"},"locked":["PluginManager","Rpc","Transport"],
 "runtimes":{"node":{"source":"download","auto":false},"python":{"source":"download","auto":false}}}
JSON
```

命令示例：

```bash
DIM_APP_SERVER_BIN=<app-server-binary> \
DIM_APP_SERVER_ARGS=app-server \
DIM_APP_SERVER_MODEL_CONFIG=<offline-model-config> \
DIM_APP_SERVER_DEFAULTS=<defaults-file> \
DIM_APP_SERVER_NODE=<node-binary> \
DIM_APP_SERVER_JS_RUNTIME=<js-runtime-dir> \
  go test -run TestRealDaemonScheduleWatch -v ./internal/extensions/
```

观察到的输出（节选；不打印调度表达式正文）：

```text
服务端 0.1.0-alpha.21（profile configured）
hello schedule.list=true
hello schedule.watch=true
hello schedule.save=true
hello schedule.history=true
hello schedule.delete=true
watching: seq=0 kind=watching
创建任务 id=<job-id>
jobs: seq=1 kind=jobs
run 跟随: listReads=3 historyReads=1 historyItems=0
删除任务回执字段=[job_id]
delete: seq=2 kind=jobs
extensions: schedule.watch 序号缺口 last=2 next=5
最终 listReads=5 historyReads=1 stats={Notices:map[schedule:5] Reads:map[schedule:5]}
```

判定条件：

- 预设生效：hello 声明 `schedule.list/watch/save/history/delete`，`schedule.list` 成功。
- ready：`seq=0 kind=watching` 后 list 读取次数从 0 到 1。
- 变化：创建任务后 `seq=1 kind=jobs`，list 读取次数到 2，读数含新任务；等待 300ms 读取次数不变。
- history：domain 的 run 通知路径触发一次 `schedule.history`（list 读取到 3、history 到 1）；本次任务安排在几分钟后，没有真实 run，因此 `historyItems=0`。
- 删除：`schedule.delete` 回执 `{job_id}`，`seq=2 kind=jobs`，list 读取到 4，读数不再含该任务。
- 序号缺口：注入 `last+3` 的 jobs 通知后 list 读取到 5，并记录 `序号缺口`；随后 300ms 无新增读取。
- 不轮询：所有读取都由显式调用或通知驱动，等待窗口内读取次数不增长。

### 6.5 blocked

| 项 | 状态 | 缺什么 |
|---|---|---|
| 钩子扩展字段与删除 | blocked | 已真实验证 `type/command/matcher` 的最小用户钩子；`args/if/commandWindows` 等字段与删除语义未验证 |
| 项目作用域导入与含秘密条目 | blocked | 已验证 user 作用域一条 stdio 条目；project 来源样本缺失，含 env/header 秘密的条目未做过真实导入 |
| 真实 HTTP/SSE/鉴权 MCP 服务器 | blocked | 已用无凭据本地 stdio 夹具验证；网络传输、鉴权与远程服务器未验证 |
| MCP OAuth 成功链路 | blocked | 没有真实 OAuth 认证服务器；只验证了 start/logout 的 `mcp_auth_unsupported`、complete/cancel 的 `mcp_auth_attempt_closed` 与能力门禁 |
| 真实 `skills.reload_catalog` 成功路径 | blocked | alpha.21 的 hello 未声明且实际调用 unknown_method；本版只能按能力缺失零请求 |
| 真实 `mcp.save` 原始配置文档 | blocked | `ToolServerConfiguration` 覆盖最小文档；已有原始文档的完整透传未验证 |
| 真实 `plugin.install` / `skills.install` | blocked | 安装参数虽有实测形状，但没有安全的真实插件/技能源与回执闭环；本版只验证真实启停 |
| 真实 run 通知与终端状态 | blocked | 任务安排在几分钟后，未等待真实执行；`run` 型通知按源码形状用于 history 重读验证，未观察真实状态序列 |
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
9. **写重试靠新键**：数据型写入同一内容成功过就不会再发；项目钩子启停、技能刷新、MCP 认证属于命令型操作，成功后再次调用会换新键。未知与需人工确认不会自动重试。
10. **MCP 环境变量经过写参数**：不会进入快照、日志或错误文本，但会留在写登记的内存参数里；不得在集成层打印。
11. **钩子删除未实现**：本版只有清单与写入（新建/更新）；最小命令钩子的真实写回与冲突已验证，删除未验证。
12. **计划任务只读**：本版只做清单与跟随，不提供启停、立即运行或补跑。
13. **唤醒过滤**：读数过滤 `cancelled`/`expired`，保留 `scheduled`/`fired`；历史状态仍可从写回执与通知观察。
14. **工作流无运行时**：类型保留，alpha.21 下任何工作流读取/启动/停止都报能力缺失且零请求。
15. **真实安装未闭环**：插件/技能安装只在 fakeappserver 与单元测试里验证；真实服务端只验证启停。
16. **mcp.save 原始配置**：`ToolServerConfiguration` 生成最小合法文档并支持传入原始 `Configuration`；原始配置文档的完整透传尚未在真实服务端验证。
17. **冲突表达**：hooks 版本冲突体现为 `write.IntentView.Status=failed` 且 `Reason` 含 `hooks_conflict`，没有单独的错误哨兵；调用方按状态与原因处理。
18. **D12 新增写意图待接线**：集成层已改用 `HookDocument` 与 `ToolServerConfiguration`，`mcp.remove` 已带版本前置条件；新增的 `hooks.set_enabled`、`skills.refresh/reload_catalog`、`mcp.auth.*` 写意图仍需 D12 映射。`mcp.import.apply` 的 D12 分支仍是旧 `{source,ids}` 形状，需改成 `{source,from/expectedSourceRevision,expectedVersion,resolutions}`。
19. **hello 门禁需集成层接线**：`Config.Features` 为 nil 时不做前置门禁；D12 必须把 `*daemon.Connection` 传进 `Features`，并实现 `ProjectHooks` 与 `MCPImports` 读取方，项目钩子、真实导入与新增能力才会按 hello 门禁。
20. **认证成功未验证**：MCP 认证只验证方法接入、能力门禁与无认证服务器错误；成功换取凭据需要真实 OAuth 认证服务器，`clientSecret` 与回调 URL 属敏感输入，日志不得记录。
21. **skills.reload_catalog 不可用**：alpha.21 hello 未声明且调用 unknown_method；本版只保留能力缺失零请求的入口。
22. **旧导入包装范围有限**：`PreviewToolServerImport`/`ApplyToolServerImport` 只支持 `SourceUser` + `From`，不能表达 project 目标或 `document`；完整能力走 `PreviewMCPImport`/`ApplyMCPImport`。
23. **平台**：只在 macOS arm64 上跑过测试；Windows 未验证。

## 8. 下一步

1. 补钩子扩展字段（args/if/commandWindows）与删除语义的真实证据。
2. 补项目作用域导入与含秘密条目的真实样本；验证 `document` 输入与 `resolutions` 的 replace/rename/skip 动作。
3. 用真实调度器触发一次 run，补 `run` 状态的 launching/submitted/running/terminal 真实通知与 history 对照。
4. 补真实插件/技能安装夹具，验证安装参数与回执闭环。
5. 推动 D12 接入 `Features`/`ProjectHooks`/`MCPImports` 与 `SkillCatalogReader`，并映射新增写意图。
6. 准备真实 OAuth MCP 认证服务器，补 `mcp.auth.start/complete/cancel/logout` 成功链路与凭据不落日志的证据。
7. 评估列表分页与按能力分锁；当前一页读取足够第一版。
8. 与 D11a 对齐运行组件能力缺失的表达，避免两处对 `runtime.*` 给出不同结论。
