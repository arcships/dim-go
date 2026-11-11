# D12 集成层（装配、宿主外壳接线、通知与清单路由、写登记适配器、自检入口）

Status: 第三版实现完成（P1 集成候选结果包，待组织评审）
Owner: D12 集成层领域
验证: 可执行命令与可观察输出见第 7 节；真实 daemon 证据见 7.3，blocked 见第 9 节
依据: `docs/domains/A.md`（两个平面、四条总规矩、启动与退出、模块分工）、`docs/domains/R.md`（块 1–3、6、7、12、14 与 R8）、`docs/domains/D10-host.md`（六阶段启动、五阶段退出、写闸门、诊断白名单、二次启动）、`docs/domains/D1-connection.md` §3.7 的 v3 wire 形状、各领域文档的"接口形状"章节
范围: 把已落地的领域包装配成可运行程序，并接到宿主外壳上：连接与订阅、投影、通知与清单路由、写登记的 Sender 与 Inspector、目录、交互、账号、设置、扩展、工作区（含变更读取来源）、附件、终端；启动与退出分别挂到 D10 的阶段函数；一个只读视图与意图入口；一个走完整启停的自检子命令。不含界面、各领域内部逻辑。

---

## 1. 结果包

**领域**：D12 集成层

**输入**

| 输入 | 来源 | 本版使用方式 |
|---|---|---|
| 两个平面、四条总规矩、启动与退出 | `docs/domains/A.md` §4、§5 | 阶段划分、状态归属、退出顺序逐条对应 |
| 六阶段启动、五阶段退出、写闸门、诊断白名单、二次启动 | `docs/domains/D10-host.md` §3、§5 | `NewHost` 把本包的步骤函数接到外壳；写闸门取 `Shell.Writable`；诊断只走 `host.Recorder` |
| v3 wire 形状（审批、标题、核对、四类清单、附件、方法清单） | `docs/domains/D1-connection.md` §3.7 与 `internal/appserver/protocol` | 通知解码、清单读取、核对与附件写入只在这里映射一次 |
| 写意图类型与两个注入点 | `docs/domains/D3-write.md` §3–§4 | 实现 `write.Sender` 与 `write.Inspector`；十一类意图全覆盖（含命令、模态、分支） |
| 折叠、分页、清单注入 | `docs/domains/D2-projection.md` §2.2、§4 | 首屏折叠、向前翻页、重读尾部、`SeedPending*` 四类清单 |
| 审批推进、提问清单与结算 | `docs/domains/D5-interaction.md` §4 与 `question.go` | 审批事实、`SeedQuestions`/`ApplyAskUser`/`SettleQuestion`、长期允许的两步顺序 |
| 目录读取与标题合并 | `docs/domains/D4-catalog.md` §3 | `catalog.Reader`、`TitleUpdate` 按修订号合并 |
| 账号、设置、扩展的读取方形状 | `docs/domains/D6-account.md`、`D11a-settings.md`、`D11b-extensions.md` | 逐项映射 wire 回执为领域类型 |
| 本地能力 | `docs/domains/D7-workspace.md`、`D8-attachments.md`、`D9-terminal.md` | 工作区、暂存区、终端登记表在本地状态阶段与退出阶段打开与收尾；变更读取只注入来源、不扫描 |
| 变更读取接口 | `docs/domains/D7-workspace.md` §4.7 的 `ChangeSource`/`RunChanges` | 实现来源适配器并暴露 `App.RunChanges` |
| fake 服务端 | `internal/fakeappserver`（V） | 装配链路、通知路由、清单注入、核对与能力门禁测试 |

**产物**

| 路径 | 职责 |
|---|---|
| `internal/app/doc.go` | 包说明与装配图 |
| `internal/app/app.go` | `Config`、`App`、`New`、五个启动阶段步骤、四个退出阶段步骤、写闸门、阶段与诊断 |
| `internal/app/host.go` | `NewHost`：把 App 的步骤函数接进 `host.Shell`；`ClassOf` 收敛错误分类 |
| `internal/app/session.go` | 订阅与首屏折叠分离、只读视图、翻页、重读尾部、意图入口、终端启动、会话方法清单缓存 |
| `internal/app/approvals.go` | 会话流通知路由：审批三类、标题通知、时间线 ask_user 与工具结果；审批事实与长期允许口径 |
| `internal/app/pending.go` | 四类未完成事项的截止点读取与注入；提问清单优先、失败退回历史推导 |
| `internal/app/sender.go` | `write.Sender`：写意图类型 → 服务端方法；发送前过写闸门；会话贡献方法做能力门禁；附件、命令、模态、分支走会话流 |
| `internal/app/writes.go` | 命令执行/停止、模态写入的意图入口（先过写闸门，再登记） |
| `internal/app/inspector.go` | `write.Inspector`：优先 `session.input_lookup`，失败或未广告时退回投影 + 历史扫描 |
| `internal/app/catalog.go` | `catalog.Reader`（`session.catalog`）、订阅释放、导出用按轮历史来源 |
| `internal/app/account.go` | 账号状态/用量/模型读取、浏览器登录的发起与完成 |
| `internal/app/settings.go` | 设置来源/项目/文档/模型目录/存储/运行组件/模态能力读取、能力声明转发 |
| `internal/app/extensions.go` | 插件、技能、钩子、工具服务器、计划任务、工作流、唤醒的读取方 |
| `internal/app/notices.go` | 控制连接的通知路由（扩展事件、标题兜底、审批丢弃） |
| `internal/app/changes.go` | `workspace.ChangeSource` 适配器：从工具结果事实提取声明路径；`App.RunChanges` 入口 |
| `internal/app/providers.go` | `settings.ProviderAdmin`/`CollaborationReader`/`AgentAdmin` 适配器：provider 清单/详情/探测、协作策略、agent 档案 |
| `internal/app/account_sources.go` | `account.ProviderSource`/`ProviderAuthorizationSource`/`DeviceAuthorizationSource`：按 provider 分发登录 |
| `internal/app/rules.go` | `interaction.RuleSource`：`permission.rules` 读取、`permission.set_rules` 写入（冲突不重试） |
| `internal/app/update.go` | 应用更新装配：当前版本、版本源、单文件目标、会话空闲探针、退出五阶段、重启回调、诊断 |
| `cmd/dimagent/update.go` | `update check|download|switch` 子命令：状态与版本号输出，不打印下载地址 |
| `internal/app/app_test.go` | 装配链路、宿主接线、单实例、阶段失败、退出超时、诊断白名单、写闸门 |
| `internal/app/wire_test.go` | 审批路由、标题合并、清单注入、提问时间线路由、核对优先级、附件写入与能力门禁 |
| `internal/app/writes_test.go` | 命令执行/停止、模态四种写入、分支开/结束的映射与门禁 |
| `internal/app/changes_test.go` | 变更读取：多工具合并、去重、越界、按 run 过滤、空声明、来源错误、截断 |
| `cmd/dimagent/main.go` | 无界面自检子命令 `selfcheck`：完整六阶段启动与五阶段退出 |
| `docs/domains/D12-integration.md` | 本文件 |

**验收**（详见第 7 节）

- 命令：`go test ./...`、`go vet ./...`；本版全部通过（2026-11-10，macOS arm64）。`go test -race ./internal/app/` 通过。
- 本领域测试：67 个顶层测试与 8 个子测试全部通过。
- 证据档位：`fakeappserver`（本机 Unix socket 上真实 NDJSON，覆盖通知路由、清单注入、核对与附件门禁）；真实 App Server（0.1.0-alpha.7 与 0.1.0-alpha.18，临时数据目录 + mock 模型）。
- 真实事件里能触发的已触发（清单读取、提交、核对方法可探测）；不能触发的（审批、标题、ask_user）如实标 blocked。

**依赖**

| 依赖 | 需要对方冻结/提供 | 本版处理 |
|---|---|---|
| D1 连接 | 控制连接、订阅、会话方法、input_lookup、pending、attachment.put 的 wire 形状 | 已按 §3.7 接线；缺方法的实例按能力缺失处理 |
| D2 会话投影 | 折叠入口、分页、按轮来源、`SeedPending*`（已冻结） | 直接用 |
| D3 写登记 | `Sender`/`Inspector` 契约与十一类意图（已冻结） | 命令、模态、分支三类新意图已映射；核对优先 `session.input_lookup` |
| D4 目录 | `Reader`、`TitleUpdate`、`SubscriptionReleaser`、`HistorySource` | 标题通知按修订号合并 |
| D5 交互裁决 | `Fact`、`SeedQuestions`、`ApplyAskUser`、`SettleQuestion`、`Settle` | 审批事实与提问清单/时间线均接入 |
| D6/D11a/D11b | 读取方接口与回执形状（已冻结） | 逐项映射 |
| D10 宿主外壳 | `Shell`/`StartupSteps`/`ShutdownSteps`/`Recorder`（已冻结） | 已接线 |
| U1 原生界面 | 只读视图、待决队首与意图入口的消费方式 | 本包只提供视图与入口 |

**上下文需求**

- 需要需求侧确认长期允许的口径：本版按"tool 来源 + 输入只有绝对路径一个字段"判定可长期允许（比 R 块 6 的"结构简单的命令"更窄），其余一律不可长期允许。
- 需要用户确认附件写入的文本上限是否由集成层预检（服务端上限 10 MiB，本版只依赖服务端拒绝）。
- 需要界面阶段确认提问的应答格式：本版把 `ask_user` 输入原始保留，问题与候选项由交互裁决解析，应答原文由界面给出。

## 2. 边界

**做**

| 能力 | 说明 |
|---|---|
| 装配 | 连接、订阅、投影、目录、交互、账号、设置、扩展、工作区、附件、终端按依赖顺序接起来 |
| 宿主接线 | 五个启动阶段与四个退出阶段分别指向本包步骤函数；写闸门取外壳；诊断走 `host.Recorder` |
| 通知路由 | 审批三类 → 交互裁决；标题通知 → 目录；时间线 ask_user 与工具结果 → 提问集合 |
| 清单注入 | 四类未完成事项按固定截止点读取后注入投影；提问同时注入交互裁决；读取失败退回历史推导 |
| 写登记适配 | `Sender` 按 `write.Kind` 选方法、合入会话/分支/投递方式/输入标识；附件、命令、模态、分支走会话流并做能力门禁；发送前过写闸门 |
| 核对 | `Inspector` 优先 `session.input_lookup`（accepted/unknown），失败或未广告时退回投影 + 历史扫描 |
| 变更读取 | 从工具结果事实提取声明路径，按 run（一轮）过滤，交给工作区规范化；不扫描、不推断 diff |
| 只读视图 | `SessionView`：行、轮次、四类未完成事项、目标、分支、装载阶段、待决计数；不含可写引用 |
| 自检 | `dimagent selfcheck`：六阶段启动 → 打开会话 → 打印行 → 提交 → 重读 → 五阶段退出 |

**不做**

| 不做 | 归属/原因 |
|---|---|
| 界面、渲染、输入法、快捷键 | U1 |
| 各领域内部逻辑（折叠算法、状态机、暂存实现、伪终端实现） | D1–D11 |
| 单实例端点、阶段推进、逐段超时、诊断白名单 | D10 宿主外壳 |
| 会话内容缓存、事实落盘 | A.md：客户端不保存第二份事实 |
| 长期允许的业务口径最终裁定 | 需求侧；本版给出更窄的可执行口径并注明 |
| 自动重连后自动重开会话 | 需要 U1/D10 决定交互；本包提供 `Reconnect` 与重新打开的能力 |

本包只用 Go 标准库与仓库内各领域包；不引第三方依赖，不改 `go.mod`。

## 3. 装配图

```text
   cmd/dimagent selfcheck
        │  ① 建 App（不装载、不连接）  ② 建 Recorder  ③ NewHost 接线
        ▼
   ┌──────────────────────────────── host.Shell（D10）────────────────────────────────┐
   │ 单实例端点互斥 → 六阶段推进 → 写闸门 → 五阶段退出 → 诊断（白名单字段）            │
   └───────────┬─────────────────────────────────────────────────────┬──────────────┘
               │ StartupSteps（五个阶段函数）                          │ ShutdownSteps（四个阶段函数）
               ▼                                                       ▼
        internal/app.App                                     StopIntents / EndInteraction /
        ├─ LoadLocalState  归档隐藏列表 + 附件暂存区 + 工作区  CloseConnection / ReclaimService
        ├─ Connect         Supervisor.Connect → 控制连接 + 各领域装配
        ├─ Subscribe       会话订阅（session.Manager.Acquire）
        ├─ FirstHistory    首屏折叠 + 四类清单注入（projection.SeedPending*、interaction.SeedQuestions）
        └─ AllowWrites     打开写入闸门

   会话流通知 → routeSessionEvent：
        ├─ approval.request   → interaction.Fact（长期允许按路径口径）→ Apply
        ├─ approval.resolved  ┐
        ├─ approval.cancelled ┴→ Settle（服务端结算，移出并推进）
        ├─ session.notice     → protocol 标题变更 → catalog.ApplyTitle（按修订号合并）
        ├─ session.event      → 折叠（投影）+ ask_user/工具结果 → ApplyAskUser / SettleQuestion
        └─ 其余               → 记诊断，不解释

   控制连接上的读适配器与发送方：
        ├─ catalogReader / account*Reader / settings*Reader / extensionReaders（见 §4.4）
        ├─ appSender          写意图方法（见 §4.1–4.2）；附件走会话流
        └─ 通知路由           scheduler.changed / wakeup.changed / workflow.changed / 标题兜底

   只读视图 SessionView ◄ Folder.View + Arbiter.Counts/Current
   意图入口              → write.Manager.Register
```

要点：

1. **启动与退出的驱动者只有宿主外壳**：本包只提供每个阶段的执行体；阶段重复调用是空操作，顺序由 D10 校验。
2. **连接只有一个来源**：控制连接由 `Supervisor.Connect` 产生；会话流由 `session.Manager` 按会话共享一份。
3. **折叠只有一处**：历史、实时、向前翻页、重读尾部全部落到 `projection.Folder`；本包只做通知分类，不解析时间线语义。
4. **写只有一处**：所有写意图都经 `write.Manager` 登记；登记入口先过 `App.Writable()`（外壳闸门），`Sender` 只做映射、不重试。
5. **事实只有一份**：本包不落盘会话内容；本地状态只有归档隐藏列表与附件暂存区（R8 允许的两项）。

## 4. 适配器映射表

### 4.1 `write.Sender`：写意图 → 服务端方法

| `write.Kind` | 服务端方法 | 参数来源 | 证据档位 |
|---|---|---|---|
| `KindSubmitInput` | `session.submit` | 透明参数 + `sessionId` + `branchId`（空按 `main`）+ `mode`（排队 `queue`／插话 `steer`）+ `submissionId`（UUID） | 真实 daemon（两版）通过 |
| `KindStop` | `session.stop` | 透明参数 + `sessionId` + `branchId`；`memberId` 在透明参数里 | D3/D5 假服务端证据 |
| `KindApprovalResponse` | `approval.respond` | 透明参数（`decision`）+ `sessionId` + `branchId` + `requestId` | D5 假服务端证据 |
| `KindLongTermAllow` | `permission.rules`（读）+ `permission.set_rules`（写） | 事实里的长期规则参数 → 一条命令规则 `{command, match, decision:"allow"}`；先读 `{revision,commandRules}`，再带 `expectedRevision` 写回完整规则表；`permission_rules_conflict` 时重读重试（最多 3 次），其它错误原样上抛 | 假服务端；真实服务端未验证 |
| `KindQuestionAnswer` | `question.respond` | 透明参数（`decision`、`answer`）+ `sessionId` + `branchId` + `requestId` | D5 假服务端证据 |
| `KindManagementWrite` | 见 4.2 | `op` + 该操作的载荷 | 设置与扩展领域的假服务端证据 |
| `KindAccountAction` | `account.<op>` | 透明参数里的 `op`（`logout`、`reset`） | D6 假服务端证据（真实登出/重置会影响真实授权，未执行） |
| `KindAttachmentWrite` | `attachment.put` | `{sessionId, name?, text}`；**走会话流**，发送前按 `session.methods` 判定能力，缺方法报能力缺失且不发请求 | 假服务端全链路；真实服务端本机不可用（两个二进制都不贡献该方法） |
| `KindSessionLifecycle` | `session.create` / `session.rename` / `session.delete` / `session.branch_create` / `session.branch_end` | `op` 取 `create`/`rename`/`delete`/`branch.create`/`branch.end`（兼容 `catalog.*` 前缀）；**`session.rename` 与分支动作走会话流**（会话必须已打开），回执 `{title,titleSource,revision}` 直接交给目录合并 | 新建/删除/改名：真实 daemon（alpha.21 configured profile）通过；分支参数与目录领域的真实分支证据一致 |
| `KindShellRun` | `shell.run`（`op=run`）/ `shell.stop`（`op=stop`） | `run`：`{sessionId, branchId, command, runId?}`；`stop`：`{sessionId, branchId, runId}`（缺 `runId` 时取意图的输入标识）；走会话流，缺方法报能力缺失 | 映射与门禁：假服务端；真实服务端两个二进制都不贡献 `shell.*`（见 7.3） |
| `KindModalityWrite` | `modality.task.submit` / `modality.task.cancel` / `modality.realtime.open` / `modality.realtime.close` | 载荷原样透传（`taskId`、模型引用、`connectionId` 等）+ `sessionId` + `branchId`；`op` 是意图级字段，不下发；走会话流，缺方法报能力缺失 | 映射与门禁：假服务端；真实服务端不贡献这些方法，名称按 D3 的 op 命名与 D11a 的 `modality.capabilities` 约定，**未在真实服务端验证** |

回执映射：`session.submit` 的 `inputId` 覆盖意图的输入标识；`attachment.put` 的回执原文（`attachmentId`/`path`/`bytes`）原样交回写登记；其余方法沿用意图的输入标识。

### 4.2 管理写入（`KindManagementWrite`）→ 服务端方法

provider、协作策略、agent 与新增扩展写入都先过 hello 能力（控制面）或会话方法表（`session.agent.update`）；缺声明零请求。

管理面的真实命名以 alpha.21 实测为准（D1 §3.7）：工具服务器是 `mcp.*`、计划任务是 `schedule.*`、唤醒是会话方法 `session.wakeup.*`、技能与钩子的读取是 `skills.catalog` / `hooks.catalog`。`workflow.*` 与 `runtime.*` 在 alpha.21 不注册：**按能力缺失处理，零请求**。

| 参数 `op`（领域包取值，兼容旧写法） | 服务端方法 | 载荷字段 |
|---|---|---|
| `plugin.install` | `plugin.install` | `source{path}` 或 `source{git,ref?}` + `enable:true` |
| `plugin.enable` / `plugin.disable`（旧 `plugin.set_enabled`） | `plugin.enable` / `plugin.disable` | `{name}` |
| `plugin.update` | `plugin.update` | `{id}` |
| `skills.install`（旧 `skill.install`） | `skills.install` | `source{path|git,ref?}` + `skills:[]` |
| `skills.set_enabled`（旧 `skill.set_enabled`） | `skills.set_enabled` | `{id, enabled}` |
| `hooks.save`（旧 `hook.write`） | `hooks.save` | `{expectedRevision, document:{hooks:[…]}}`；`expectedRevision` 为空时先读 `hooks.catalog` 的 `userRevision` 补齐 |
| `mcp.import.apply` | `mcp.import.apply` | `{cwd?, source, from?/document?, expectedVersion, expectedSourceRevision?, resolutions[{id,action,newId?}]}`；`mcp_settings_conflict` 由扩展领域归为 `ErrImportConflict` |
| `mcp.save` / `tool_server.add` / `tool_server.update` | `mcp.save` | `{id, source, expectedVersion, configuration}`；配置文档优先用载荷里的 `configuration`，否则按 transport/command/args/url/env 写最小文档 |
| `mcp.set_enabled` | `mcp.set_enabled` | `{id, source, expectedVersion, enabled}` |
| `mcp.remove` / `tool_server.remove` | `mcp.remove` | `{id, source:"user", expectedVersion}`；版本为空时读 `mcp.catalog` 的 `userRevision` 补齐 |
| `hooks.set_enabled` | `hooks.set_enabled` | `{cwd, enabled, expectedRevision}`（项目作用域） |
| `skills.refresh` | `skills.refresh` | 无参数；回执 `{version,issues}` 原样交回 |
| `skills.reload_catalog` | `skills.reload_catalog` | 无参数；alpha.21 hello 未声明 → 能力缺失、零请求 |
| `mcp.auth.start` | `mcp.auth.start` | `{id, source, cwd?, clientSecret?}`；回执里的授权地址不回显、不日志 |
| `mcp.auth.complete` / `.cancel` | 同名 | `{attemptId, callbackUrl}` / `{attemptId}` |
| `mcp.auth.logout` | `mcp.auth.logout` | `{id, source, cwd?}` |
| `session.wakeup.cancel`（旧 `wakeup.cancel`） | `session.wakeup.cancel`（**会话流**） | `{sessionId, wakeupId}`；会话标识取载荷，缺失时用意图的会话标识 |
| `workflow.start` / `workflow.stop` | —（不注册） | 能力缺失，零请求 |
| `settings.update` | `settings.update` | `source`、`scope`（用户 `"user"`／项目 `{project}`）、`expectedRevision`、`document` |
| `provider.select` | `provider.select` | `providerId`、`modelId`、`expectedVersion`（取 `expectedRevision`） |
| `runtime.install`（旧写法保留） | —（不注册） | 能力缺失，零请求 |
| `provider.create` / `provider.update` | 同名 | 载荷已是 wire 形状：`providerId`（规范小写 UUID）、`expectedVersion`、`patch` 等；去掉 `op` 后原样下发 |
| `provider.models.set_enabled` / `.edit` / `.reset` | 同名 | `providerId`、`expectedVersion`、`modelIds`/`modelId`、`edit` |
| `provider.preferences.update` | `provider.preferences.update` | `providerId`、`expectedVersion`、`modelId`、`patch` |
| `collaboration.set_model_policy` | 同名 | `{expectedRevision, policy:{mode, model?}}` |
| `session.agent.update` | `session.agent.update`（**会话流**） | `{sessionId, branchId, agentId}`；**`expectedAgentId` 是本领域前置条件，不下发**（服务端拒绝未知字段） |
| `session.context.compact` | `session.context.compact`（**会话流**） | `{sessionId, branchId}`；回执 `{sequence,status}` 原样交回；压缩结果以时间线 `context/compaction` 为准 |
| `session.collaboration.cancel` | `session.collaboration.cancel`（**会话流**） | `{sessionId, branchId?, memberSessionId}`；回执 `{status:"cancelled"\|"idle"}`，`idle` 视为已停 |

### 4.3 `write.Inspector`：核对读取

| 顺序 | 情况 | 判定 | 依据 |
|---|---|---|---|
| 1 | 提交类意图（`KindSubmitInput`）且会话方法清单声明了 `session.input_lookup`（或 hello 声明了同名能力） | 调 `session.input_lookup {sessionId, branchId, submissionId}`：`accepted` → 命中；`unknown` → 确认未发生 | 服务端查询，结果确定 |
| 2 | 查询失败（能力缺失、超时、解析失败） | 记诊断，退回下面两条 | 不把读失败当成"发生/未发生" |
| 3 | 投影的行里有该标识 | 命中，归属取该行分支 | `Folder.BranchView` 的行标识（服务端标识） |
| 4 | 历史按页读完仍没有 | 确认未发生（允许重提） | `session.history {after}` 读到 `done` |
| 5 | 读不完（出错或超过 50 页） | 无法确认 | 不把"没读全"当成"没发生" |
| 6 | 意图没有会话标识（会话新建） | 目录里有该标识即命中 | `catalog.Entry`；目录没读过则不确认 |
| 7 | 意图没有可核对标识 | 无法确认 | 收敛到需人工确认，不自动重发 |

方法清单按会话缓存一次（`session.methods`），失败按能力缺失处理；不会对每个未知结果重复读取。

### 4.4 读取方映射（节选）

| 领域接口 | 服务端方法 | 文档/证据 |
|---|---|---|
| `catalog.Reader` | `session.catalog`（`limit`/`cursor`/`query`/`cwd` → `sessions[]`/`nextCursor`） | D4 假服务端 + 真实 daemon |
| `account` 三个读取方 | `account.providers`（解析 providerId 与默认目录）、`account.status {providerId}`、`account.usage {providerId}`、`provider.catalog {providerId}` | D6 假服务端 + D6 真实夹具（alpha.21） |
| `account.AuthorizationSource` | `account.login.browser.begin`（`providerId,directory,redirectUri` → `attemptId,authorizationUrl`）、`account.login.browser.complete`（`providerId,attemptId,callbackUrl`）、`account.login.cancel`（`providerId,attemptId`） | D6 §6.3 + 真实夹具（发起与取消；完成需真人登录） |
| `settings.*Reader` | `settings.sources`/`projects`/`read`、`models.providers.list`、`storage.report`、`modality.capabilities`；`runtime.*` 不注册 → 能力缺失 | D11a 假服务端 |
| `extensions` 读取方 | `plugin.list`（数组）、`skills.catalog`、`hooks.catalog`、`mcp.catalog`/`mcp.test`/`mcp.import.preview`、`schedule.list`、`session.wakeup.list`（会话流）；`workflow.*` 不注册 → 能力缺失 | D11b 假服务端 + D1 §3.7 |
| 四类未完成事项 | `session.input.pending` / `session.exec.pending` / `session.question.pending` / `session.collaboration.pending`（固定 `through`、`next` 原样带回，最多 5 页） | 真实 daemon（alpha.18/alpha.21 的 input/collaboration 返回真实条目；exec/question 按实例退回推导） |
| 控制连接通知 | `scheduler.changed`、`wakeup.changed`、`workflow.changed`、`session.notice`（标题兜底） | D11b 假服务端 + 本版测试 |
| `settings.ModalitySession` | `modality.task.status`/`list`/`result`、`modality.realtime.send`（会话流）；门禁 hello → 会话方法表 | 真实 daemon：OCR 任务 `queued→running→succeeded`、结果分页读取；假服务端覆盖映射 |
| `settings.ProviderAdmin` | `provider.list`、`provider.management {providerId}`、`provider.probe` | 真实 daemon（alpha.21 夹具）清单 127 条、详情与连接映射跑通；探测映射用假服务端 |
| `settings.CollaborationReader` | `collaboration.model_policy` → `{revision, policy{mode, model?}}` | 真实 daemon：revision 与 mode 有值 |
| `settings.AgentAdmin` | `agent.list`（内建，不按 hello 预判）、`session.agent.get`（会话流）；`SupportsSessionAgentGet/Update` 按 hello → 会话方法表 | 真实 daemon：清单 1 条、方法可用 |
| `account.ProviderSource` / `DeviceAuthorizationSource` | `account.providers`；`account.login.begin` / `account.login.status`（设备码） | 真实 daemon：browser/device 发起、轮询与取消（见证据 K） |
| `interaction.RuleSource` | `permission.rules`（`{revision,commandRules[]}`）、`permission.set_rules`（`{expectedRevision,commandRules}`） | 真实 daemon：读取、原样写回推进 revision、旧 revision 冲突（见证据 M）；假服务端覆盖缺能力零请求 |
| 上下文占用 | `session.context.usage`（会话流）→ `projection.ContextReading`（窗口/输出预留/阈值/估算输入/基线/through） | 真实 daemon：窗口 258000、输出预留 128000、阈值 114000、估算输入实测（见证据 O）；协议 v3 不推 `context/budget` |
| 运行组件 | `runtime.list` / `runtime.install`；hello 未声明 `runtime.*` 时能力缺失零请求 | 真实 daemon（带 `--defaults` 预设）：node/python/uv 状态 `off`（见证据 P） |

### 4.5 会话流通知 → 领域

| 通知 | 去向 | 处理 |
|---|---|---|
| `approval.request` | 交互裁决 `Apply` | 映射成 `Fact`：`RequestID`/`SessionID`/`BranchID`/`TurnID`（`source.turn_id`）/`Action`（工具名 + 原因）；长期允许按 4.7 口径 |
| `approval.resolved` | 交互裁决 `Settle` | 服务端结算：移出并推进下一条 |
| `approval.cancelled` | 交互裁决 `Settle` | 调用没有开始，按结算处理 |
| `session.notice`（`SessionStore/title`） | 目录 `ApplyTitle` | `titleRevision` 更高才覆盖；`title` 为 null 表示清空 |
| `session.event`（`step/tool-use`、`step/tool-call`） | 交互裁决 `ApplyAskUser` | 只认 `ask_user`；序号不晚于清单截止点的调用由清单负责，自动丢弃 |
| `session.event`（`step/tool-completed`、`step/tool-result`） | 交互裁决 `SettleQuestion` | 按 `request_id` + `call_id` 结算，批次空了整条移出 |
| `session.event` / `session.model.progress` | 投影折叠 | 本包不解析内容 |
| `session.state` 与其余 | — | 不改变本版持有的状态；其余只记一条诊断 |

### 4.6 会话命令与模态调用（真实形状）

**门禁顺序（两处共用）**：先看 `hello.features`，再看会话方法表 `session.methods`；两者都没有才算缺失。真实服务端的情况正好相反：`shell.*`/`attachment.put`/`session.rename` 只出现在**会话方法表**里，`modality.*` 只出现在 **hello** 里——只查一边都会误判。

| 项 | 口径 |
|---|---|
| 命令归属 | `shell.run`/`shell.stop` 由服务端执行；`mode=record` 时结果回写成 `input/message`（append），本机终端（D9）是另一条路径 |
| `shell.run` 参数 | `{sessionId, branchId, runId, command, mode}`：服务端要求 `runId` 与 `mode` 必填；缺 `runId` 用意图输入标识（UUID），缺 `mode` 按 `record` |
| `shell.stop` 参数 | 只接受 `{sessionId, runId}`（服务端拒绝未知字段，不能带 `branchId`） |
| `shell.list` 参数 | `{sessionId}`；回执只取运行标识、分支、模式与开始时间，**不保留命令正文** |
| 模态方法 | `modality.task.submit|status|list|cancel|result`、`modality.realtime.open|send|close` 都由会话流承载，参数里**不带** `sessionId`/`branchId` |
| 模态参数映射 | 领域用 camelCase 表达载荷（`clientTaskId`、`files[].mediaType`/`dataBase64`/`nodePath`、`model{providerId,modelId}`），发送方映射成服务端 snake_case（`client_task_id`、`files[].media_type`/`data_base64`/`node_path`）；`op` 不下发 |
| 模态标识 | 回执 `{task:{task_id}}` / `{connectionId}` 覆盖意图的输入标识；`task.cancel` 只发 `{task_id}`，`realtime.close` 只发 `{connectionId}` |
| 模态读取面 | 集成层实现 `settings.ModalitySession`：任务状态/清单/结果与实时帧发送都走会话流；任务记录里的 `progress{ fraction,message }`、`result{ files[], data }`（文件字段兼容 snake_case 与 camelCase）、`error{ code,message }` 原样映射 |
| 能力缺失 | 缺方法一律零请求；实时传输打不开（例如 `no_default_model`）按失败原样上报，自检里记 blocked，不伪造连接 |

### 4.7 长期允许的判定口径（集成层给出）

服务端 wire 里没有「可长期允许」标记，`approval.request` 只给出 `source` 与 `call`；长期允许写成服务端的**命令规则**。

- 只认 `source.kind == "tool"` 且工具名是 `exec`/`execute`（启动进程的命令工具）。
- `call.input.command` 必须是非空字符串，且不含 `;&|\`()<>$` 与换行这类命令分隔符——只放行单条简单命令，复杂命令不进长期规则。
- 规则为 `{command, match:"exact", decision:"allow"}`；`approvalFacts` 把它放进事实的 `LongTermParams`，发送方据此走 `permission.rules` → `permission.set_rules`。
- 写入顺序仍是「先写规则、成功后才发审批应答」；规则写入失败时不发应答，条目留在待决集合（D5 的两步式）。
- 冲突：`permission.set_rules` 回 `permission_rules_conflict` 时重读规则表并重试，最多 3 次；仍冲突则明确失败，不静默通过。

### 4.8 变更读取（`workspace.ChangeSource` 适配器）

| 项 | 口径 |
|---|---|
| run 的含义 | `runID` 是一轮的标识（事实里的 `turn_id`）；只取该轮工具结果里的声明，别轮一律不看 |
| 来源读取 | 读本会话历史里的工具结果事实（`step/tool-completed`、`step/tool-result`），与投影同源；折叠后的工具行只保留文本，因此这里取原始载荷。一次读取最多 40 页 × 500 条，超限报错而不是返回看似完整的清单 |
| 声明字段 | 只认 `structuredContent.affectedPaths`（兼容 snake_case）、`write`/`edit` 工具的 `structuredContent.path`、`fileWritten` effect（`structuredContent.effects[]`、顶层 `effects[]`/`fileWritten`，字段 `type`/`kind`/`name` 为 `fileWritten`，路径取 `path`/`filePath`/`file_path`/`file`/`paths`） |
| 没有声明 | 返回空清单，不把"没有声明"当成"没有改动"，也不扫描任何目录 |
| 规范化 | 交给 `workspace.ReadRunChanges`：去空白、展开 `~/`、相对路径拼工作区根、解析符号链接后判内/外、按 `scope+path` 去重、工作区内在前、超过 256 条截断并标记 |
| 覆盖级别 | 恒为 `tool_reports`：工具声明级的部分覆盖，不声称覆盖会话内命令（exec）、外部修改、未声明的写入 |
| 不做的 | 不扫描工作区、不推断 diff、不读 git 历史、不执行工具、不落盘、不缓存 |
| 错误 | 会话未打开 → 报错（`ErrSessionNotOpen`）；历史读取失败/游标不前进 → 报错；没有工作区根 → `ErrWorkspaceUnavailable` |

来源适配器只在 `App` 上存一份实例引用（`app.changes`），不新增包级全局状态；`App.RunChanges(ctx, sessionID, runID)` 调用 `Workspace.ReadRunChanges`，不在集成层另做规范化。

### 4.9 账号（真实 wire 形状，alpha.21）

| 领域接口 | 服务端方法 | 映射 |
|---|---|---|
| provider 解析（内部） | `account.providers` | 取 `providerId` 与 `defaultDirectory`，优先选 `loginMethods` 含 `browser` 的提供者；结果按实例缓存一次 |
| `StatusReader` | `account.status {providerId}` | `account` 为空 → `signed_out`；否则 `signed_in`，`subject/email/name/plan` → `Identity`。真实回执没有独立的「过期」字段 |
| `UsageReader` | `account.usage {providerId}` | 全量映射：`plan{name,status,priceAmount,priceCurrency,priceInterval,expiresAt,currentTermStart}` → `Plan`；`credits{usedUnits,totalUnits,remainingUnits,unlimited,resetsAt}` → `Credits`（`Unlimited` 为真时 `Aggregate` 只给原因）；`windows[]` → `Windows` 并拼进 `Lines`；`features[]{key,used,allowance,remaining,unit,unlimited,periodEnd}` → `Features` 并拼进 `Lines`；`subscriptionUrl`/`pricingUrl` 原样保留，`subscriptionId` > 0 时置 `SubscriptionID`/`HasSubscriptionID`。字段缺失保持 nil / 零值，不填 0；`PriceAmount` 是货币最小单位，原样透传，不做货币换算 |
| `ModelReader` | `provider.catalog {providerId}` | `catalog.models[{id,name,unavailableReason}]` → `Model{ID, DisplayName, RequiresAccount:true, Unavailable: reason != nil}` |
| `ProviderSource` | `account.providers` | `providerId/displayName/loginMethods/defaultDirectory` → `account.Provider`；登录方式只取服务端声明 |
| `ProviderAuthorizationSource` | `account.login.browser.begin` / `complete`；取消用 `account.login.cancel` | begin 带 `providerId,directory,redirectUri`，回执 `attemptId,authorizationUrl`；complete 带 `providerId,attemptId,callbackUrl`；尝试到 provider 的对应只在内存里保存 |
| `DeviceAuthorizationSource` | `account.login.begin` / `account.login.status` | begin 带 `providerId,directory`，回执 `attemptId,verificationUri,verificationUriComplete,userCode,expiresAt,interval`；轮询带 `providerId,attemptId` → `{status,error?}`；未知状态按事实不合法拒绝 |

### 4.10 权限档位（`session.permission.*`）

| 项 | 口径 |
|---|---|
| 读取 | 打开会话、首轮历史之后调 `session.permission.get {sessionId,branchId}` → `{version,policy}`；`policy.preset` 原样装进交互裁决（`PermissionTier`），同时记下 `version` |
| 切换 | `App.SetPermissionTier(ctx, session, preset)` 先过写闸门，再调 `session.permission.update {sessionId,branchId,expectedVersion,policy{preset,allowedTools:[],disallowedTools:[]}}`；策略列表显式给空数组（服务端不接受 null），不发的后续字段由服务端保留 |
| 状态驱动 | 以回执与时间线事实为准：回执更新 `version` 与档位；`session.event` 的 `permission/changed`（`data.policy.preset`）同样更新档位。**没有 `permission.changed` 通知**，本包不订阅、不轮询 |
| 透传 | 提交意图经 `WithPermission` 带上当前档位；没有值时原样返回，不补默认 |
| 能力缺失 | `get` 失败只记诊断（不阻塞打开会话）；`update` 回 `unknown_method`／不可用时按能力缺失报错，零请求由发送方门禁保证 |

### 4.11 自启参数（统一二进制）

`dimcode-server` 是统一二进制，需要 `app-server` 子命令（D1 §3.7）。`App.Connect` 在 `ServerArgs` 为空且可执行文件名为 `dimcode-server` 时自动补 `["app-server"]`；调用方显式给了参数就原样使用（例如 `-server-arg app-server`）。`dim-app-server` 这类独立二进制不补参数。

### 4.12 应用更新（装配 `host.Updater`）

| 项 | 口径 |
|---|---|
| 当前版本 | `UpdateOptions.CurrentVersion`（x.y.z），或 `ReadVersion` 每次读取安装产物里的版本；CLI 缺省从构建信息取，读不到用 `0.0.0` |
| 版本源 | 完整 `ManifestURL` 优先；否则按现有客户端的渠道约定推导：基址 `https://dimcode.echooai.com`、通道 `updates/stable`、平台路径 `mac/<arch>`（或 `win/x64`、`linux/x64`）、清单 `latest-mac.yml`/`latest.yml`。渠道地址只读确认自现有客户端配置，不在本包硬编码新值 |
| 单文件目标 | `TargetPath` 默认是当前可执行文件；平台目录/安装器不在地支持范围（见 10） |
| 会话空闲探针 | `App.SessionsIdle()`：本实例没有打开中的会话即为空闲 |
| 退出五阶段 | 接到宿主外壳后，更新编排的 `Shutdown` 直接调用外壳的 `Shutdown`（不复制阶段顺序） |
| 重启 | `Restart` 由调用方给出；CLI 用 `exec.Command(target)` 启动新进程，不接管输出 |
| 诊断 | 复用宿主外壳的白名单 `Recorder`；只记录结果、状态与版本字段 |
| 检查失败 | 状态进 `up_to_date`，原因放 `Status.LastError` 与诊断；**不阻塞使用** |
| 切换推迟 | 有会话未收尾时返回 `host.ErrSessionsBusy`，状态保持待切换，产物不替换 |
| 输出纪律 | 只打印状态与版本号；不打印清单地址、产物地址或其中的查询串 |

CLI：

```text
dimagent update check      # 检查；失败打印原因并退出 0
dimagent update download   # 检查 + 下载校验；失败退出 1
dimagent update switch     # 检查 + 下载 + 切换；会话未收尾时明确报并退出 1
```

`switch` 与 `download` 在同一进程内完成状态机（检查 → 待下载 → 待切换），因为更新状态只在内存，不落盘。

### 4.13 界面注册（`ui.*`）

| 项 | 口径 |
|---|---|
| 注册 | `AttachUI(ctx, session)` 在已 attach 的会话流上调 `ui.attach {sessionId, kind:"gui"}`（普通路径不带 `uiId/token`）；保存回执的 `uiId` 与 `generation`，状态 `online` |
| 关闭 | `CloseUI(ctx, session, reason)` 调 `ui.close {uiId, reason?}`，随后清本地注册；没有注册时不发请求 |
| 清单 | `UIStatus(ctx)` 读控制连接 `ui.status`（instances/webConnections），只用于自检与诊断 |
| 建议性关闭 | `ui.shutdown {uiId, reason}` 只记录进 `DrainUIShutdowns` 并把注册标成 `shutdown_requested`；不自动发 `ui.close`、不杀进程 |
| detach/reattach | 会话流断开时注册标成 `detached`；重连重读后自动重新 `ui.attach`（同一连接+会话幂等），恢复 `online` |
| 清理 | 会话关闭与退出五阶段清理本地注册；不额外请求 |
| 失败码 | `UIErrorCode(err)` 取稳定 appError（`ui_in_use`、`session_not_attached`、`ui_detached` 等），调用方按码分类 |
| 范围 | `ui.*` 只对本机连接；叶子路径（`ui.open` + rutis）不在 Go 侧实现 |

### 4.14 上下文、成员与默认预设

| 项 | 口径 |
|---|---|
| 上下文占用 | `App.ReadContextUsage(ctx, session)` 走 `session.context.usage`，映射成 `projection.ContextReading` 并缓存在会话内（内存）；打开会话后尽力读一次，缺方法零请求 |
| 手动压缩 | `App.CompactContext(ctx, session)` 登记一次 `KindManagementWrite`，发送方转到 `session.context.compact`；结果以 `context/compaction` 事实与投影的 `Compaction` 为准 |
| 成员停止 | `App.CollaborationMembers` 从 `session.collaboration.pending` 的 `delivery.session_id` 与历史 `collaboration/delivered` 收集成员会话；`App.StopTeamMember` 登记一次写，发送方转到 `session.collaboration.cancel`；`idle` 视为已停 |
| 默认预设 | `Config.ServerDefaults` 追加 `--defaults <文件>`；文件缺失记诊断并按能力缺失处理（不传参数、不崩）。带预设的实例才在 hello 声明 `runtime.*` |
| 门禁 | 上下文/成员/运行组件都按 hello-first；缺方法零请求 |

### 4.15 新建会话的权限档位

| 项 | 口径 |
|---|---|
| 入口 | `App.CreateSession(ctx, cwd, preset)` → `catalog.Create`，参数 `permissionPolicy:{preset}`；`preset` 为空表示按服务端默认，**不猜** |
| 为什么不默认 full-access | 新建会话的档位是用户安全选择的一部分。集成层不能为了让工具调用“跑通”就静默提权：默认策略等待审批是服务端的正确行为，客户端应当呈现审批，而不是绕过 |
| 自检与自动化 | 需要工具调用的证据/自检必须**显式传参**（例如 `--permission-preset full-access`）；缺省为空，不下发 `permissionPolicy` |
| selfcheck | `-permission-preset <档位>`：缺省为空＝按服务端默认；文档与证据命令都显式写出档位 |
| 正常创建路径 | 调用方给出用户当前选择的档位；没有选择时传空，由服务端默认策略决定；集成层不补默认，也不在本地记录 |

**成员权限策略（只查协议与文档，不改实现）**

- `session.collaboration.pending` 的请求形状是 `{turn_id, request_id, call_id, target, prompt}`；`target` 只有 `kind`（`agent`/`member`/`temporary`）、`agent_id`、`session_id`、`profile`，**没有权限策略字段**。
- `agent` 工具的 `profile` 结构只接受 `name`、`description`、`systemPrompt`、`tools`、`model`（`additionalProperties:false`），同样没有权限字段。
- 服务端在派发协作请求时按**请求分支当前的权限策略**采样（`PermissionState` → `member` 的 `permission_policy`），成员不能由客户端单独指定策略。
- 结论：**成员策略由服务端决定，客户端不可指定**；客户端能做的只有先设置好发起会话/分支的档位。

## 5. 启动与退出（接线点）

### 5.1 启动：五阶段执行体 → 宿主六阶段

| D10 阶段 | 本包步骤函数 | 完成含义 | 失败处理 |
|---|---|---|---|
| `single_instance` | —（外壳自己） | 抢到实例端点；二次启动转交后直接退出 | 外壳记录并停在 `single_instance` |
| `local_state` | `App.LoadLocalState` | 归档隐藏列表（失败按空处理）、附件暂存区打开并清理、工作区根解析 | 暂存区或工作区不可用即返回错误 |
| `connect` | `App.Connect` | `Supervisor.Connect` 握手；各领域依次装配；登记控制连接通知路由 | 任一步失败即释放已建对象并关闭连接 |
| `subscribe` | `App.Subscribe` | 为 `Config.InitialSessionID` 建唯一订阅；未指定时读目录第一条；都没有时空转并记诊断 | 订阅失败停在该阶段；没有会话不算失败 |
| `first_history` | `App.FirstHistory` | 折叠首屏，并读四类清单注入投影与交互裁决 | 订阅不存在时报错；没有会话时空转 |
| `allow_writes` | `App.AllowWrites` | 打开本包写入闸门，阶段进入 `ready` | 首轮历史未完成则拒绝 |

- 阶段函数都幂等；`App.Phase()` 只反映本包进度，进程级阶段以 `shell.State().Stage` 为准。
- 连接探针 `App.Connected()` 交给外壳；`ObserveConnection` 只在连上/连不上时写 `connection.changed`。

### 5.2 退出：四阶段执行体 → 宿主五阶段

| D10 阶段 | 本包步骤函数 | 动作 |
|---|---|---|
| `stop_intents` | `App.StopIntents` | 置退出标记（此后 `Submit`/`Respond`/`StopMember` 一律 `ErrExiting`），把引用交给后续阶段 |
| `end_interaction` | `App.EndInteraction` | 释放全部会话订阅、关闭写登记（未结项按"需人工确认"收尾）、关闭扩展事件跟随、停全部终端 |
| `close_connection` | `App.CloseConnection` | 注销通知路由，关闭会话流管理器与控制连接 |
| `reclaim_service` | `App.ReclaimService` | `Supervisor.Shutdown`：只回收自己拉起的实例 |
| `flush_diagnostics` | —（外壳用 `Recorder.Flush`） | 冲刷诊断 |

- 每段由外壳在独立 goroutine 中执行并逐段超时；超时或失败只记诊断，继续下一段。
- 二次启动（`StartForwarded`）不进入退出五阶段：本进程没有装载本地状态、没有连接、没有订阅。

### 5.3 写闸门

`App.Writable()`：`closed` → `ErrClosed`；退出第一阶段已执行 → `ErrExiting`；`allow_writes` 未执行 → `ErrNotReady`；外壳闸门（`Shell.Writable`）→ `ErrNotReady`/`ErrNotConnected`/`ErrExiting`。

两处使用：意图入口（`Submit`/`Respond`/`StopMember`/`RunShell`/`StopShell`/`ModalityWrite`）在登记前检查，被拒即不登记；**发送方也检查一次**，因此即使调用方绕过入口直接登记（例如目录领域的分支意图），未就绪或退出中也不会发出请求。读路径不受影响。

### 5.4 诊断

- 进程级诊断只走 `host.Recorder`（白名单字段，无自由文本），本包通过 `app.ObserveConnection` 追加 `connection.changed`。
- `App.Diagnostics()` 是给人看的自由文本（例如"标题已合并""读取待答提问失败，退回历史推导"），**不写进** `host.Recorder`；`selfcheck` 默认打印到标准输出，`-log` 只装白名单记录。

## 6. 只读视图与意图入口

```go
base, err := app.New(app.Config{
    DataDir: ..., ServerPath: ..., WorkspaceRoot: ...,
    InitialSessionID: "s-1",     // 订阅阶段要打开的会话；空则由目录第一条决定
    Version: "0.1.0",
})
shell, err := app.NewHost(base, app.HostOptions{DataDir: ..., Args: os.Args[1:], Version: "0.1.0", Diag: host.NewRecorder(file)})
outcome, err := shell.Start(ctx)               // 六阶段；StartForwarded 时本进程直接退出

view, ok := base.View("s-1")                   // 只读视图（不发起请求）；含四类清单与待决队首
intent, err := base.Submit(ctx, app.SubmitRequest{SessionID: "s-1", Text: "继续"})
outcome2, err := base.Respond(ctx, "s-1", "req-1", interaction.Response{Decision: interaction.DecisionApprove})
run, err := base.RunShell(ctx, app.ShellRunRequest{SessionID: "s-1", Command: "pwd"})
stop, err := base.StopShell(ctx, app.ShellStopRequest{SessionID: "s-1", RunID: "run-1"})
task, err := base.ModalityWrite(ctx, app.ModalityWriteRequest{SessionID: "s-1", Op: app.ModalityTaskSubmit, Params: map[string]any{"model": "m1"}})
changes, err := base.RunChanges(ctx, "s-1", "turn-1") // 该轮工具声明的改动路径（部分覆盖）
report, err := shell.Shutdown(ctx)             // 五阶段退出（含逐段超时）
```

`SessionView` 只含服务端事实与可丢状态：行（含服务端标识、是否未落库）、轮次、四类未完成事项、目标、分支目录、装载阶段、分页锚点、待决计数与队首。不含包内临时键、不含可写引用。

## 7. 验收

### 7.1 命令与结果

```bash
go test ./...
go vet ./...
go test -race ./internal/app/
```

本版结果：三条命令全部通过（2026-11-11，macOS arm64）。`internal/app` 的 67 个顶层测试与 8 个子测试全绿。

### 7.2 覆盖

| 验收要求 | 测试 |
|---|---|
| 六阶段启动；订阅阶段打开指定会话；写闸门放行；退出五阶段按序走完 | `TestHostWiringStartupAndShutdown` |
| 连接 → 打开会话 → 折叠历史行 → 提交一次并收到回执 | `TestAssemblyOpenSessionAndSubmit` |
| 实时事实进入同一份折叠；重读后尾部行换成服务端标识且不重复 | `TestLiveEventLandsIntoRows` |
| 写意图类型 → 服务端方法（8 类） | `TestSenderMapsIntentKinds`（8 个子测试） |
| 核对：命中、确认未发生、会话未打开时无法确认 | `TestInspectorLooksUpInputThroughProjection` |
| 目录读取、订阅阶段按目录决定会话、生命周期核对 | `TestCatalogAssembles` |
| 单实例二次启动：转交成功、阶段停在 idle、不进入退出五阶段 | `TestSecondLaunchForwardsAndSkipsShutdown` |
| 启动阶段失败：停在 connect、写闸门保持拒绝 | `TestStartupStopsAtFailingStage` |
| 退出超时：记录后继续走完五段 | `TestShutdownTimeoutContinues` |
| 日志白名单：凭据与正文进不来、字段集合固定 | `TestDiagnosticsWhitelistOnly` |
| 写闸门：未就绪与退出中都拒绝且不发请求 | `TestWriteGateBeforeReadyAndWhileExiting` |
| 读路径不受写闸门影响、各领域已装配 | `TestReadPathsWorkAfterReady` |
| 审批三类通知：请求入队（含长期允许口径）、结算与取消出队 | `TestApprovalNoticesRouteToArbiter` |
| 标题通知按修订号合并、低修订号不覆盖 | `TestTitleNoticeMergesIntoCatalog` |
| 四类清单注入投影；提问同时进交互裁决；单类读失败退回推导 | `TestPendingSeedingOnAttach` |
| 核对优先 `input_lookup`（accepted/unknown），方法缺失时退回推导 | `TestInspectorPrefersInputLookup` |
| 附件写入走 `attachment.put`；缺方法报能力缺失且不发请求 | `TestAttachmentWriteUsesPutAndChecksMethods` |
| 时间线 ask_user 进队列、工具结果结算 | `TestAskUserTimelineRoutesToArbiter` |
| 命令执行/停止：`shell.run`/`shell.stop`、会话与主线、命令/运行标识、回执里的运行标识 | `TestShellRunAndStopMapping` |
| 模态四种写入：方法名、载荷透传、`op` 不下发、回执里的任务/连接标识 | `TestModalityWriteMapping` |
| 开分支/结束分支：`session.branch_create`（会话+切点+首条输入）、`session.branch_end`（会话+分支） | `TestBranchLifecycleMapping` |
| 三类新意图的门禁：缺方法零请求、退出中零请求（入口拒绝 + 发送方兜底） | `TestNewWriteKindsRespectGates` |
| 变更读取：多工具合并、去重、越界、按 run 过滤 | `TestRunChangesMergesToolDeclarations` |
| 变更读取：声明缺失返回空、未知 run 为空、会话未打开报错、没有工作区根报错 | `TestRunChangesEmptyAndErrors` |
| 变更读取：超过上限截断并保留总数 | `TestRunChangesTruncates` |
| 管理面读取用 `plugin.list` / `skills.catalog` / `hooks.catalog` / `mcp.catalog` / `schedule.list` / `session.wakeup.list`，旧名字零请求 | `TestManagementReadsUseRealMethods` |
| 管理面写入落到 `plugin.enable` / `skills.set_enabled` / `hooks.save` / `mcp.*` / `session.wakeup.cancel`；workflow 能力缺失零请求 | `TestManagementWritesUseRealMethods` |
| `workflow.*` 与 `runtime.*` 能力缺失：读与写都零请求 | `TestWorkflowAndRuntimeMissingWithoutRequests` |
| 改名走会话流并按回执修订号合并 | `TestRenameGoesThroughSessionStream` |
| 权限：打开时读、切换写（带版本）、时间线 `permission/changed` 驱动状态 | `TestPermissionGetUpdateAndTimeline` |
| 统一二进制自启参数：`dimcode-server` 自动补 `app-server`，显式参数不覆盖 | `TestServerArgsForUnifiedBinary` |
| 长期允许写命令规则：`permission.rules` 读、`permission.set_rules` 带版本写 | `TestSenderWritesCommandRuleForLongTerm` |
| 新增扩展写入：项目钩子启停、技能刷新、MCP 认证四步；`reload_catalog` 缺能力零请求 | `TestNewExtensionWritesUseRealMethods` |
| 新增扩展写入的气门：hello 未声明时全部零请求 | `TestNewExtensionWritesMissingCapability` |
| provider 清单/详情/探测、协作策略、agent 清单与会话 agent 的读取映射 | `TestProviderAdminReads` |
| provider 管理/协作策略/会话 agent 写入：参数映射、`op` 不下发、`expectedAgentId` 不下发 | `TestProviderAndAgentWritesUseRealMethods` |
| provider 写入缺能力：零请求且明确失败 | `TestProviderWritesMissingCapability` |
| 上下文占用读取映射、缓存与缺能力零请求 | `TestContextUsageReadMapsReading` |
| 手动压缩走写登记与会话流、缺能力零请求 | `TestCompactContextMapsWrite` |
| 成员标识来源（pending + delivered）与停止映射、`idle` 视为已停 | `TestCollaborationMembersAndStop` |
| 带默认预设时 `runtime.list` 读取与 `runtime.install` 映射 | `TestRuntimeListReadsWhenDeclared` |
| 账号邮箱透传（browser/device）与设备码取消 | `TestAccountEmailAndDeviceCancel` |
| 登出成功后清 provider 解析缓存 | `TestLogoutClearsProviderBinding` |
| MCP 导入新形状（resolutions/expectedVersion）与冲突映射 | `TestMCPImportApplyMapsResolutions` |
| 新建会话显式带权限档位；空档位不带字段 | `TestCreateSessionPassesPermissionPreset` |
| 界面注册成功保存 uiId/generation、关闭清理、ui.status 映射 | `TestUIAttachCloseAndStatus` |
| `ui.shutdown` 只记录为建议性关闭、不自动 ui.close | `TestUIShutdownIsAdvisory` |
| 失败码 `ui_in_use`/`session_not_attached` 分类与 detach/reattach 注册跟随 | `TestUIErrorCodesAndReattach` |
| 界面注册缺能力零请求 | `TestUIAttachMissingCapability` |
| 断线恢复、共享订阅、未知核对（真实 daemon，环境变量开启） | `TestRealResilienceEvidence` |
| 成员停止真实端到端：显式 full-access 创建 → 成员创建 → 停止 `cancelled`（环境变量开启） | `TestRealCollaborationMemberStopEvidence` |
| 应用更新：检查到新版、已是最新 | `TestUpdaterCheckFindsNewVersion`、`TestUpdaterUpToDate` |
| 应用更新：下载摘要不一致留在待下载 | `TestUpdaterDownloadChecksumFailure` |
| 应用更新：有会话时切换推迟、产物不替换 | `TestUpdaterSwitchDeferredWithOpenSession` |
| 应用更新：切换替换产物并读到新版本、重启一次 | `TestUpdaterSwitchReplacesTargetAndVersion` |
| 命令规则来源：读取映射、写入映射（expectedRevision 与规则字段） | `TestRuleSourceReadAndWrite` |
| 命令规则冲突：映射成 `ErrRulesConflict` 且不重试 | `TestRuleSourceConflict` |
| 命令规则缺能力：零请求、裁决记住缺失 | `TestRuleSourceMissingCapability` |
| 账号用量新字段：plan/credits/windows/features/链接/订阅标识透传 | `TestAccountUsageMapsNewFields` |
| 账号用量缺失字段保持 nil / 聚合不可用 | `TestAccountUsageMissingFieldsStayNil` |
| 账号用量两次读取互不共享内存 | `TestAccountUsageDeepCopiesValues` |
| 账号提供者映射 | `TestAccountProviderSource` |
| 按 provider 的浏览器登录发起/完成/取消参数 | `TestAccountProviderAuthorization` |
| 设备码登录发起与轮询状态映射、未知状态拒绝 | `TestAccountDeviceAuthorization` |
| 模态门禁 hello 优先、其次会话方法表；两边都没有才缺失 | `TestModalityGateChecksHelloBeforeSessionMethods` |
| 模态适配器：任务状态/清单/结果与实时帧的映射；缺能力零请求 | `TestModalitySessionMapsTasks` |
| `shell.list` 摘要不含命令正文 | `TestShellRunsReadsListWithoutContent` |

### 7.3 真实 daemon 证据

真实证据用例只从环境变量取路径：`DIM_APP_SERVER_BIN`（服务端可执行文件）、`DIM_REAL_MODEL_FIXTURE`（夹具根，含 `home/`）、`DIM_MODEL_CONFIG`（离线模型配置）、`DIM_MODALITY_IMAGE`（模态图片）、`DIM_RUNTIME_PRESET`（现有客户端的运行时预设目录）。缺变量时用例直接 `t.Skip` 并点名缺哪个，不写任何本机默认路径。

服务端在仓库之外，使用临时数据目录与 mock 模型。两版都不能产生工具调用，因此审批、标题与 ask_user 的真实事件**未触发**（见 7.5）。

**证据 A：完整启停与清单降级（0.1.0-alpha.18，`<app-server-binary>`）**

```bash
D=$(mktemp -d)   # 临时数据目录
go run ./cmd/dimagent selfcheck -data-dir "$D" \
  -server <app-server-binary> -mock \
  -create -submit "通知与清单接线自检" -lines 10 -log "$D/diag.log"
```

```text
startup: ready（单实例 → 本地状态 → 连服务 → 订阅 → 首轮历史 → 允许写）
session: <session-id>  branch: main  load: ready  rows: 0  turns: 0  hasOlder: false
pending: inputs=0 tasks=0 questions=0 collaborations=0  approvals=0 待答=0
submit: status=succeeded  input=<input-id>
after submit: rows: 2  load: ready
[turn 1] user_message   <event-id> 通知与清单接线自检
[turn 1] assistant_text <event-id> Echo: 通知与清单接线自检
diagnostics:
  - 读取未结束后台任务失败，退回历史推导: session.exec.pending: ... unknown_method
  - 读取待答提问失败，退回历史推导: session.question.pending: ... unknown_method
shutdown: 5 段全部完成（5 ms）
```

**证据 B：启动阶段就带会话，清单读到真实条目（同一数据目录，`-session`）**

```bash
go run ./cmd/dimagent selfcheck -data-dir "$D" -server <app-server-binary> \
  -mock -session <session-id> -lines 10 -log "$D/c.log"
```

```text
startup: ready（单实例 → 本地状态 → 连服务 → 订阅 → 首轮历史 → 允许写）
session: <session-id>  branch: main  load: ready  rows: 2  turns: 1  hasOlder: true
[turn 1] user_message   <event-id> 通知与清单接线自检
[turn 1] assistant_text <event-id> Echo: 通知与清单接线自检
pending: inputs=4 tasks=0 questions=0 collaborations=0  approvals=0 待答=0
```

`inputs=4` 是从服务端 `session.input.pending` 读回并注入投影的真实条目（四条未消费的状态块输入）；`tasks`/`questions` 在该版服务端是 `unknown_method`，按设计退回历史推导（此时为 0）。阶段耗时：`connect` 52ms、`subscribe` 71ms、`first_history` 6ms。

**证据 C：旧服务端全部降级（0.1.0-alpha.7，`dimcode app-server`）**

```bash
go run ./cmd/dimagent selfcheck -data-dir "$D" -server "<server-binary>" \
  -server-arg app-server -mock -session <会话标识> -submit "旧服务端 wire 自检"
```

四类清单方法全部 `unknown_method`，逐类退回历史推导，会话与提交仍正常：

```text
submit: status=succeeded  input=<input-id>
pending: inputs=0 tasks=0 questions=0 collaborations=0  approvals=0 待答=0
diagnostics:
  - 读取未消费输入失败，退回历史推导: session.input.pending: ... unknown_method
  - 读取未结束后台任务失败，退回历史推导: session.exec.pending: ... unknown_method
  - 读取待答提问失败，退回历史推导: session.question.pending: ... unknown_method
  - 读取未结束协作请求失败，退回历史推导: session.collaboration.pending: ... unknown_method
```

**证据 D：附件写入的能力门禁（同一 alpha.18）**

```bash
go run ./cmd/dimagent selfcheck ... -session <会话标识> -attachment "要存成会话文件的一段文本"
```

```text
attachment: status=failed reason="发送明确失败: attachment.put: non_retryable: 服务端缺少该能力: attachment.put" result=
```

`session.methods` 在该二进制上没有 `attachment.put`，集成层据此判定能力缺失、**不发请求**，写登记按明确失败收敛。同一台机器上用 D1 的探针程序（`dimagent-conncheck -probe`）复核，`attachment.put`、`session.exec.pending`、`session.question.pending` 三项都回 `unknown_method`，与该判定一致。

**证据 E：命令执行的能力门禁（同一 alpha.18）**

```bash
go run ./cmd/dimagent selfcheck ... -session <会话标识> -shell "pwd"
```

```text
shell: status=failed reason="发送明确失败: shell.run: non_retryable: 服务端缺少该能力: shell.run" run= result=
```

该二进制的 hello 与 `session.methods` 都没有 `shell.*`（也没有 `modality.*`），集成层据此判定能力缺失、**不发请求**。

同一二进制在 hello 里声明了 `session.branch_create`/`session.branch_end`。目录领域用同一对方法、同一组参数在同一服务端上跑通了真实分支生命周期（本包 Sender 的参数与之一致）：

```text
$ DIM_APP_SERVER_BIN=<app-server-binary> \
    go test ./internal/catalog -run TestRealDaemonBranchLifecycle -count=1 -v
初始目录: [{ID:main ParentID: CutSequence:0 Status:open Head:1}]
新分支:   {ID:<branch-id> ParentID:main CutSequence:4 Status:open Head:0}
分支收敛：state=ended head=9
回收自启服务端: stopped=true
```

**证据 F：两个进程并发启动，只有一个赢家**

```text
A exit=0   B exit=0
=== A out ===
startup: ready（单实例 → 本地状态 → 连服务 → 订阅 → 首轮历史 → 允许写）
=== B out ===
instance: forwarded（已有实例在运行，本次启动参数已转交）
=== instance events ===
{"event":"instance.primary"}
{"event":"instance.forwarded","args":13}
```

只有 A 推进了六阶段并写出 `startup.*` 与 `shutdown.done`；B 停在 `idle`，没有 `shutdown.*` 记录。

**证据 G：改名（alpha.21 configured profile）**

mock profile 不注册管理面方法（`Management method is unavailable`），改名与权限要在 configured profile 上验证：服务端用复制到临时目录的离线模型配置启动（`-model-config`），数据目录也是临时目录。

```bash
D=$(mktemp -d)   # 临时数据目录
go run ./cmd/dimagent selfcheck -data-dir "$D" \
  -server "<server-binary>" \
  -model-config <model-config> -create -rename "集成层改名验证" -lines 4
```

```text
startup: ready（单实例 → 本地状态 → 连服务 → 订阅 → 首轮历史 → 允许写）
session: <session-id>  branch: main  load: ready  rows: 0  turns: 0
pending: inputs=5 tasks=0 questions=0 collaborations=0  approvals=0 待答=0
rename: title="集成层改名验证" source="manual" revision=1
shutdown: 5 段全部完成（4 ms）
```

**证据 H：权限档位读取与切换（同一会话，alpha.21）**

```bash
go run ./cmd/dimagent selfcheck -data-dir "$D" \
  -server "<server-binary>" \
  -model-config <model-config> -session <session-id> -permission read-only
```

```text
startup: ready（单实例 → 本地状态 → 连服务 → 订阅 → 首轮历史 → 允许写）
permission: preset="read-only" ok=true
diagnostics:
  - 已建立会话订阅: <session-id> 头部=35
  - 权限档位=full-access: 会话=<session-id>        # 打开时 session.permission.get
  - 权限档位=read-only: 会话=<session-id>          # session.permission.update 的回执
shutdown: 5 段全部完成（4 ms）
```

两次运行都回收了自启的服务实例；临时数据目录与临时模型配置在取证后删除。

**证据 I：会话内命令真实执行（alpha.21 configured profile）**

环境同证据 G。命令正文与输出不进入证据，只记运行标识、状态、耗时、退出码与历史事实类型增量。

```bash
go run ./cmd/dimagent selfcheck -data-dir "$D" \
  -server "<server-binary>" \
  -model-config <model-config> -create -shell "<一条短命令>" -shell-list
```

```text
shell.run: status=succeeded run=<run-id> state=completed durationMs=24 exitCode=0
history: rows=0 kinds=
history.events.added: input/applied=+2 input/message=+2
shell.list: runs=0
```

`mode=record` 的结果确实回写成 `input/message`（append）与 `input/applied` 两条事实；`shell.list` 在运行结束后为空。

运行中停止（命令换成长命令，同样不打印正文）：

```text
shell.stop: status=succeeded run=<run-id> stopped=true
shell.run: status=succeeded run=<run-id> state=stopped durationMs=69
history.events.added: input/applied=+3 input/message=+2
shell.list: runs=0
```

**证据 J：模态任务真实执行（alpha.21 configured profile）**

`DIMAGENT_MODALITY_EVIDENCE=1` 时运行 `TestRealModalityAndShellEvidence`，输入图片来自 `DIM_MODALITY_IMAGE`；证据只打印状态序列、任务标识、文件字节数与页数。

```text
shell.run: run=<run-id> status=succeeded
modality.task: id=<modality-task-id> states=[queued running succeeded]
modality.result: file=result.json bytes=94 pages=1 eof=true
modality.result: file=text.txt bytes=0 pages=1 eof=true
modality.cancel: id=<modality-task-id> state=cancelled
modality.realtime: blocked（打开失败）: 实时传输打开失败: ... no_default_model: No model serves realtime.transcribe
```

- 任务提交、状态、结果分页、取消四条链路都走本包的 `ModalitySession` 与会话流。
- 实时传输：打开被服务端拒绝（该账号没有 `realtime.transcribe` 的默认模型），按 blocked 记录，不伪造连接；没有进入帧发送。
- 服务端能力回执里没有单独的能力标识，集成层用 `类别.操作` 派生本地标识（见 10）。

**证据 K：账号按 provider 分发（alpha.21 夹具，只验证发起与取消）**

`DIMAGENT_ACCOUNT_EVIDENCE=1` 时运行 `TestRealAccountLoginBeginEvidence`；夹具 home 从只读夹具目录复制到新的临时目录。证据只留状态、标识与主机名。

```text
account.providers: count=1 methods=[browser device] directorySet=true
browser.begin: attemptSet=true urlHost=dimagent.cn
browser.cancel: ok
device.begin: attemptSet=true userCodeSet=true verificationHost=dimagent.cn expiresSet=true interval=5s
device.status: status=pending reasonSet=false
device.cancel: ok
```

- 两种登录都按 `account.providers` 的声明分发；browser 与 device 的发起、轮询与取消都走本包的来源适配器。
- 没有完成真实登录交换（需要真人在授权页完成），真实登出/重置仍未执行。
- 证据里没有授权码、用户码原文或回调查询串，只保留「已给出」与主机名。

**证据 L：provider / 协作策略 / agent 读取（alpha.21 夹具）**

用本包适配器在 fixture home 副本上读取（只读）：

```text
provider.list via adapter: providers=127 errors=0
provider.management via adapter: versionSet=true connectionSet=true
collaboration via adapter: revisionSet=true modeSet=true
agent.list via adapter: count=1
```

真实 `provider.management` 的模型条目里 `model` 是对象（`modelId/displayName/status/capabilities`），回执里没有 `status` 的条目按 `active` 展示；这些映射选择见 §10。

**证据 M：命令规则来源（alpha.21 夹具，写入只影响副本）**

`DIMAGENT_RULES_EVIDENCE=1` 时运行 `TestRealRuleSourceEvidence`；夹具 home 复制到临时目录，证据只留 revision 是否给出与规则条数。

```text
permission.rules: revisionSet=true count=0
permission.set_rules: revisionChanged=true count=0
permission.set_rules conflict: mapped=true
```

- 读取回执的 `{revision,commandRules}` 映射成 `RuleSet`；写入带 `expectedRevision` 与完整规则集合。
- 原样写回推进了 revision；用旧 revision 再写返回 `permission_rules_conflict`，本包映射成 `interaction.ErrRulesConflict` 且只请求一次。
- 真实回执只带 revision，规则集合以本次写入的集合为准；只读缓存由交互裁决持有，不落盘。

**证据 N：更新检查失败不阻塞（本机离线）**

```bash
dimagent update check -version 0.0.1 \
  -manifest http://127.0.0.1:9/updates/stable/mac/arm64/latest-mac.yml
```

```text
update: state=up_to_date current=0.0.1 available=-
dimagent update: 检查失败（不阻塞使用）: Get "...": connect: connection refused
exit=0
```

真实渠道 `https://dimcode.echooai.com` **没有请求**（离线环境），见 9。检查成功、下载校验、切换与版本变化由本地 HTTP 版本源在测试里覆盖（证据见 7.2）。

**证据 O：上下文占用与手动压缩（alpha.21 夹具，真实模型）**

`DIMAGENT_COMPACTION_EVIDENCE=1` 时运行 `TestRealCompactionEvidence`：用 gpt-5.6-terra（258k 窗口 / 128k 输出）建会话，连续 4 轮大输出把上下文推过保留目标，再手动压缩。证据只留数字与状态。

```text
占用读数: 窗口=258000 输出预留=128000 阈值=114000 估算输入=3062 基线=full_estimate
第 1 轮后: 估算输入=15044 阈值=114000
第 2 轮后: 估算输入=21205 阈值=114000
第 3 轮后: 估算输入=27385 阈值=114000
第 4 轮后: 估算输入=33543 阈值=114000
压缩回执: err=<nil> status=completed sequence=117
折叠结果: counters=1 status=completed strategy=summary summaryLength=995
压缩后占用: 估算输入=8656（压缩前 33543）
```

- `session.context.usage` 给出与 `context/budget` 等价的读数；协议 v3 的视图里没有 `Context`（事实不下发）。
- 手动压缩回执 `completed`，`context/compaction` 折叠成摘要（只记录摘要长度，不打印正文）。
- 压缩后估算输入从 33543 降到 8656。

**证据 P：默认预设（`--defaults`，alpha.21）**

`DIMAGENT_DEFAULTS_EVIDENCE=1` 时运行 `TestRealRuntimeDefaultsEvidence`：把 `DIM_RUNTIME_PRESET` 指向的运行时预设复制到临时目录后启动。

```text
hello.runtime: [runtime.configure runtime.install runtime.list runtime.watch]
plugin.list: count=78 runtimeManager=true
runtime.list: count=3 [node=off python=off uv=off]
```

- 带预设的实例在 hello 里声明 `runtime.*`；不带预设时这些方法不注册（按能力缺失零请求）。
- `plugin.list` 出现 `RuntimeManager`；`runtime.list` 给出 node/python/uv 三个组件的 `off` 状态。

**证据 Q：断线恢复、共享订阅与未知核对（alpha.21 夹具）**

```bash
D=$(mktemp -d)   # 临时数据目录；测试内复制夹具 home
DIMAGENT_RESILIENCE_EVIDENCE=1 DIM_REAL_MODEL_FIXTURE=<fixture-root> \
  DIM_APP_SERVER_BIN=<server-binary> \
  go test ./internal/app -run TestRealResilienceEvidence -count=1 -v
```

```text
连接数: 初始 control+stream=2；第二个持有者后=2；释放一个后=2
已杀掉 daemon: pidSet=true
恢复: head=1->1 rows=4->4 唯一行=4 input/message=8->11
未知核对: status=succeeded reasonSet=false 新增 input/message=1 该 submissionId 出现=1
连接数: 释放会话后控制连接已重连中，未能取到计数: app server connection closed: EOF
```

- 多窗口共享：同一会话两个持有者时连接数不变（仍 control+stream=2），释放一个仍在；两个都释放后控制连接进入回收（EOF 是自启实例被回收的表现）。
- 断线恢复：`kill -9` daemon 后自动重连并重读；行数与唯一行数一致（4/4），没有重复或缺失。期间模型继续产生的新事实会让 `input/message` 总数增加，重复以行标识为准。
- 未知核对：极短发送超时让结算先转未知，`session.input_lookup` 核对后收敛为 `succeeded`，该 `submissionId` 在历史里只出现 1 次（不重发）。

**证据 R：附件真实写入与运行中停止（configured profile）**

```bash
D=$(mktemp -d)   # 临时数据目录
go run ./cmd/dimagent selfcheck -data-dir "$D" \
  -server <server-binary> \
  -model-config <model-config> -create -attachment "attachment evidence" -short
```

```text
attachment: status=succeeded reason="" result={"attachmentId":"<attachment-id>","path":"<临时数据目录>/rust/dimcode.sqlite.session-files/..."}
```

回执只记录 `attachmentId` 与路径前缀；字节数与文件内容未打印（CLI 输出长度有限，且证据不需要正文）。

```text
shell.stop: status=succeeded run=<run-id> stopped=true
shell.run: status=succeeded run=<run-id> state=stopped durationMs=70
history.events.added: input/applied=+3 input/message=+2
shell.list: runs=0
```

- `attachment.put` 在 configured profile 上真实可用，门禁放行并拿到 `attachmentId` 与路径；mock profile 会按能力缺失拒绝。
- 运行中停止：`shell.stop` 返回 `stopped=true`，运行状态收敛为 `stopped`，清单清空。

**证据 S：界面注册（alpha.21 configured profile）**

```bash
D=$(mktemp -d)   # 临时数据目录
go run ./cmd/dimagent selfcheck -data-dir "$D" \
  -server <server-binary> \
  -model-config <model-config> -create -ui-attach -ui-close -ui-status -short
```

```text
ui.attach: uiId=<ui-id> generation=1 state=online
ui.close: uiId=<ui-id> state=closed
ui.status: instances=0 webConnections=0
```

- 普通路径注册成功，拿到 `uiId` 与 `generation=1`；关闭后本地注册清理，`ui.status` 里没有残留界面。
- 证据只留标识、代数与状态；不打印界面地址、token 或会话正文。

**证据 T：成员停止的真实端到端（根因已定位，已跑通）**

第一次尝试失败的现象与断点：

```text
leader 事实类型: [... step/model-start=1 step/tool-call=1 step/tool-use=1 step/tool-use-start=1 turn/start=1]
leader 工具调用: [agent=1]
pending 协作请求数=0
daemon.log 最后一行: model.invoke{...} outcome="returned"
daemon.ping: sessions=1 connections=1（没有成员会话）
```

断点：`step/tool-call`（序号 41）之后没有 `tool-completed`/`tool-result`，也没有 `collaboration/*`，`turn/start` 没有 `turn/end`；daemon 线程全部空闲。同一现象在 `exec` 工具上也能复现——不是 `agent` 工具特有问题。

根因：**新建会话没有显式请求权限档位**（当时自检新建路径缺少这个参数）。夹具 `config.json` 里的 `permission=full-access` 不等于会话策略；`session.create` 不带 `permissionPolicy` 时服务端用默认策略，工具调用转去等待审批（审批是通知，不进入历史事实），没有人应答就一直等。对照实验：同一夹具、同一提示词，`SetPermissionTier(..., "full-access")` 后成员立即创建。

修复后（证据用例显式传 `--permission-preset full-access`；selfcheck 缺省仍为空）的完整证据：

```bash
D=$(mktemp -d)   # 临时数据目录
DIMAGENT_COLLAB_EVIDENCE=1 DIM_REAL_MODEL_FIXTURE=<fixture-root> \
  DIM_APP_SERVER_BIN=<server-binary> \
  go test ./internal/app -run TestRealCollaborationMemberStopEvidence -count=1 -v
```

```text
leader session: <session-id>
成员会话: [<member-session-id>]
停止回执: intent=succeeded status=cancelled stopped=true err=<nil> 耗时=3ms
成员停止后新增事实: turn/end=+1 input/applied=+1 step/model-output-end=+1
成员停止后事实增量: 3
主线事实类型数: 19
```

- 成员创建成功并出现在 `session.collaboration.pending` 的 delivery 里。
- `session.collaboration.cancel` 回执 `cancelled`，`idle` 也算已停；停止后成员只留下收尾事实（`turn/end` 等），不再产生新轮次。
- 主线会话仍可读、写闸门正常。

结论：#1405 不应写成“服务端未创建成员”。真实断点是客户端新建会话未显式请求权限档位，默认策略下工具调用等待审批；显式 full-access 后成员创建与停止都正常。服务端“审批无超时”可以作为独立议题讨论，但不是成员丢失。

### 7.4 自检子命令

```bash
dimagent selfcheck -data-dir <目录> [-server <可执行文件>] [-server-arg <参数>]... [-mock] \
  [-session <标识>] [-create] [-cwd <目录>] [-lines <条数>] [-older <页数>] [-wait <时长>] \
  [-submit <文本>] [-attachment <文本>] [-short] [-log <文件>]
```

| 参数 | 作用 |
|---|---|
| `-data-dir` | 客户端数据目录（必填；单实例端点、服务端端点、归档隐藏列表、附件暂存区都在这里） |
| `-server` / `-server-arg` / `-mock` | App Server 可执行文件、追加参数、mock 模型；缺省取 `DIMCODE_APP_SERVER_BIN` |
| `-session` | 订阅阶段打开的会话；省略时由目录第一条决定，目录为空需 `-create` |
| `-create` / `-cwd` | 目录为空时在允许写之后新建一个会话，工作目录缺省取当前目录 |
| `-lines` / `-older` | 打印最近多少行；打开后向前翻多少页更早的轮次 |
| `-submit` / `-wait` | 提交一条文本并等待写登记收敛；提交后等多久再重读尾部 |
| `-attachment` | 用 `attachment.put` 存一段文本成会话文件（缺方法时报能力缺失） |
| `-shell` / `-shell-stop` / `-shell-list` | 命令执行（record 模式）、运行中停止、读取运行摘要（都不打印命令与输出） |
| `-rename` / `-permission` | 改名、权限档位切换（需 configured profile） |
| `-model-config` | 显式模型配置（configured profile；管理面方法只在非 mock 实例上可用） |
| `-short` | 只打印汇总 |
| `-log` | 进程级白名单诊断写到该文件；缺省丢弃 |

退出码：0 成功（含二次启动转交成功）；1 运行失败；2 用法错误；3 blocked（缺少真实 App Server）。

### 7.5 未验证

| 项 | 状态 | 缺什么 |
|---|---|---|
| 真实 `approval.request/cancelled/resolved` 事件 | blocked（真实事件未触发） | 形状已按 D1 冻结并接线；本机 mock 模型不发工具调用，产生不了审批。需要能产生需审批工具调用的真实模型 |
| 真实标题通知（`session.notice`） | blocked（真实事件未触发） | 需要服务端自动生成标题并推送通知；mock 模型下不产生 |
| 真实 `ask_user` 提问事件 | blocked（真实事件未触发） | 需要模型调用 `ask_user`；本机 mock 模型不调用 |
| 真实 `session.exec.pending` / `session.question.pending` 的非空页 | blocked（能力不可用） | 两个本机二进制都不提供这两个方法；条目形状按 wire 冻结，只在假服务端验证过 |
| 真实 `attachment.put` | blocked（能力不可用） | 两个本机二进制都不贡献该方法；门禁行为已在真实服务端验证（证据 D），写入本身只在假服务端验证 |
| 真实 `session.input_lookup` 由本包发出的核对 | 未验证 | D1 已用探针验证 accepted/unknown；本包的调用路径只有假服务端证据（需要一次超时或断线写入才能走到核对） |
| 真实审批/提问写入（应答、长期允许） | blocked | 同"真实审批事件"：没有真实待决项可答 |
| 真实改名 | blocked | 两版服务端都没有 `session.rename` |
| 插话（`mode: steer`） | 未验证 | wire 取值只有投影词表证据 |
| Windows | 未验证 | 无 Windows 主机（继承 D1/D9/D10） |

## 8. 与 A/R 的对应

| 需求或架构条目 | 本领域实现 | 不在本领域 |
|---|---|---|
| A 启动与退出（阶段与超时） | 五个启动阶段步骤 + 四个退出阶段步骤，接进 D10 `Shell` | 单实例判定、阶段推进、逐段超时、诊断白名单（D10） |
| A 连接状态机、多窗口共用一份订阅 | 控制连接唯一；`Subscribe`/`OpenSession` 复用同一份订阅 | 连接内部（D1） |
| A 两个平面、四条总规矩 | 视图只读、意图只登记、折叠唯一、发送唯一 | 各领域状态机 |
| A「审批集合、提问集合」镜像状态 | 审批由通知驱动、提问由清单 + 时间线驱动，各自单向 | 集合与队列本身（D5） |
| R 块 2 未完成事项与服务端一致 | 四类清单按截止点读取注入；断线后按事件结算 | 清单语义（D2/D5） |
| R 块 3 提交与停止 | `Submit` + `Sender` 映射；写闸门拦住未就绪与退出中 | 运行状态收敛（D2） |
| R 块 5 目录与标题 | `catalog.Reader` + 标题通知按修订号合并 | 目录页状态（D4） |
| R 块 6 审批接收与长期允许 | 三类通知路由；长期允许按 4.7 口径给出规则参数 | 长期允许的最终业务口径（需求侧） |
| R 块 7 提问与结算 | 清单注入 + `ask_user` 时间线 + 工具结果结算 | 应答内容（界面） |
| R 块 4 分支 | 开分支/结束分支的写意图映射（`session.branch_create`/`session.branch_end`） | 切点选取、分支目录与内容（D4/D2） |
| R 块 12 附件 | 暂存区阶段打开清理；`attachment.put` 写入并做能力门禁 | 暂存引用生命周期（D8） |
| R 块 13 会话内命令 | `shell.run`/`shell.stop` 映射，命令由服务端执行 | 本机终端进程（D9） |
| R 块 9 模态任务与实时传输 | `modality.*` 四种写入映射 | 模态能力读数（D11a） |
| R 块 11 本次运行的变更读取 | 工具结果声明 → 路径清单（`App.RunChanges`），只读本 run | 规范化、上限与 git 只读（D7） |
| R 块 14 日志与诊断 | 进程级诊断只写白名单字段 | 落盘与冲刷（D10） |

## 9. blocked

| 项 | 状态 | 卡在哪个依赖、由谁解除 |
|---|---|---|
| 真实审批事件（request/cancelled/resolved） | blocked（真实事件未触发） | 需要能产生工具审批的真实模型；形状已冻结、接线与假服务端测试已完成 |
| 真实标题通知 | blocked（真实事件未触发） | 需要服务端自动标题与推送；接线与合并规则已测试 |
| 真实 ask_user 提问 | blocked（真实事件未触发） | 需要模型调用 `ask_user`；清单与时间线两条路都已接 |
| 非空 `exec.pending` / `question.pending` | blocked（能力不可用） | 本机两个二进制都没有这两个方法 |
| 真实 `attachment.put` 写入 | blocked（能力不可用） | 本机两个二进制都不贡献该方法；需要提供本地执行与原生存储的实例 |
| 真实 `shell.run`/`shell.stop`/`shell.list` | 已真实跑通（alpha.21 configured profile） | 见证据 I；mock profile 不注册这些会话方法，按能力缺失处理 |
| 真实模态任务（submit/status/result/cancel） | 已真实跑通（alpha.21 configured profile） | 见证据 J；OCR 任务 `queued→running→succeeded`，取消收敛为 `cancelled` |
| 真实模态实时传输 | blocked（服务端没有可用模型） | 打开回 `no_default_model: No model serves realtime.transcribe`；帧发送路径因此未验证，不伪造连接 |
| 真实分支开/结束（本包发出的写） | 未验证 | 参数形状与目录领域的真实分支证据一致；本包没有在真实服务端上直接发过 `session.branch_create`/`session.branch_end` |
| 真实工具结果里的路径声明 | blocked（真实事件未触发） | 需要模型真的调用 `write`/`edit` 等工具并给出 `structuredContent`；本机 mock 模型不产生工具调用，因此变更读取的真实声明只验证到"空清单" |
| 真实核对调用 | 未验证 | 需要一次超时/断线后才进入核对；D1 已单独验证方法本身 |
| 自动标题通知 | blocked（真实事件未触发） | `session.rename` 与标题合并已真实跑通；自动生成的标题通知仍未触发 |
| 真实登录完成交换 | blocked | 需要真人在授权页完成登录；发起与取消已由 D6 真实夹具覆盖，本包按同一形状实现 |
| 真实登出/重置 | blocked | 会吊销真实授权；需要一次性测试账号 |
| `hooks.save` / `mcp.save` 的真实回执与冲突语义 | 未验证 | D1 只冻结了参数形状；回执与版本冲突只在假服务端验证 |
| 真实 `mcp.set_enabled` / `mcp.import.apply` | 未验证 | 方法与参数按 D1 冻结形状实现，未在真实服务端上执行 |
| 真实 MCP OAuth 认证成功 | blocked | 需要真实 OAuth 认证服务器；本机只验证到方法接入与错误分类（`mcp_auth_unsupported` 等） |
| 真实 provider 写入（create/update/models/preferences） | 未验证（本包） | D11a 已用脚本直接跑通真实 wire；本包的发送方只做了映射与假服务端测试 |
| 真实 `session.agent.update` 写入 | 未验证（本包） | D11a 脚本已验证 `adopted/unchanged`；本包只验证映射与 `expectedAgentId` 不下发 |
| 真实设备码/浏览器登录完成 | blocked | 需要真人在授权页完成；本包只验证发起、轮询与取消 |
| 真实 `permission.set_rules`（长期允许） | 未验证 | 只有假服务端证据；需要一次真实工具审批（本机模型不产生） |
| 重连后自动重开会话 | 未验证 | 需要 U1/D10 决定重开策略；本包只提供 `Reconnect` 与 `OpenSession` |
| 真实 UI 叶子路径（`ui.open` + rutis 会话） | blocked | 需要 rutis 客户端实现；本版只做普通连接注册（`ui.attach`） |
| 真实成员运行被停止 | 已真实跑通 | 显式 full-access 创建会话后成员正常创建，`session.collaboration.cancel` 回 `cancelled`，成员停止后只剩收尾事实；见证据 T |
| 真实更新渠道 | blocked（离线环境） | `https://dimcode.echooai.com` 未请求；真实渠道提供的是 macOS `.app`/安装器，本版只实现单文件替换 |
| 真实安装包上的切换与重启 | blocked | 需要真实单文件产物；测试用假文件模拟替换与版本变化 |

## 10. 已知限制

1. **启动阶段没有会话时空转**：`-session` 为空且目录里没有会话（或目录能力缺失）时，`subscribe`/`first_history` 记录诊断后照常推进；会话在允许写之后由调用方打开。
2. **清单与历史推导的合并口径**：注入清单后，投影仍会把清单未覆盖的事件事实装回去（例如历史里未消费的输入会与清单条目并存）；这一口径由 D2 决定，本包只负责注入。
3. **提问清单整体替换**：`SeedQuestions` 用清单整体替换提问集合；清单读取失败时用已折叠历史推导，截止点取分支当前头部。重连后重新装载可能让已消失的提问重新出现，直到服务端清单或事件再次结算。
4. **长期允许口径偏窄**：只放行"单字段绝对路径"的 tool 调用，`结构简单的命令`尚未实现；需要需求侧确认是否放宽。
5. **核对可能读全量历史**：`input_lookup` 不可用或失败时，确认"未发生"要把历史读完（最多 50 页 × 200 条）；超限按无法确认处理。
6. **附件写入不预检大小**：10 MiB 上限由服务端拒绝，本包只按服务端错误分类。
7. **附件写入要求会话已打开**：`attachment.put` 是会话贡献方法，必须走会话流；会话没打开时按不可重试失败处理。
8. **运行状态只取 attach 回执**：`session.state` 通知不消费，运行状态变化需要重开或后续接入。
9. **命令与模态依赖会话方法**：`shell.*`、`modality.*`、`attachment.put`、`session.branch_*` 都要求会话已打开；没打开时按不可重试失败处理，不发请求。方法名以 `session.methods`/hello 为准，不按版本猜。
10. **模态方法名缺真实服务端证据**：`modality.task.submit` 等四个名称来自意图 op 命名与 `modality.capabilities` 的约定，本机没有贡献这些方法的实例，因此只验证到门禁行为。
11. **管理面只在 configured profile 可用**：alpha.21 的 mock profile 对管理方法回 `Management method is unavailable`；改名、权限与 mcp/hooks 等需要在 configured profile（`-model-config`）上验证。
12. **管理面写形状按 D1 冻结参数实现**：`hooks.save` 写 `{hooks:[…]}` 文档、`mcp.save` 写 `{transport,command,args,url,env}` 最小文档；真实服务端的文档结构与版本冲突语义未在真实服务端验证。`mcp.remove`/`mcp.set_enabled` 的 `source` 取载荷，缺省按 `user`。
13. **唤醒取消需要会话标识**：`session.wakeup.cancel` 是会话方法；领域包的旧写法只带 `wakeupId`，集成层优先用载荷里的 `sessionId`，其次用意图的会话标识，两者都没有时报不可重试错误。
14. **账号没有「过期」字段**：`account.status` 只区分有/无账号对象；过期与恢复仍按 D6 的 blocked 记录处理。
15. **账号用量只做字段映射**：`PriceAmount` 是货币最小单位，除 100 属于界面口径，客户端不换算；`Plan/Credits/Windows/Features` 缺失保持 nil，空数组与缺失区分（指针切片）。账号领域仍未承载 `config`（账号级策略）与 `resets`（重置额度批次），集成层映射不到。
16. **权限档位值是服务端 preset**：集成层把 `policy.preset`（`read-only`/`workspace-write`/`full-access`）原样当档位保存并透传，不再改写成交互裁决早期的本地常量。
16. **模态能力标识是派生值**：真实 `modality.capabilities` 没有单独的能力标识，集成层用 `类别.操作` 派生（`key` 会重复，不能当主键）；该标识只用于本地视图，不写回服务端。
17. **模态参数做一次 camelCase→snake_case 映射**：领域载荷用 `clientTaskId`/`mediaType`/`dataBase64`/`nodePath`，发送方映射成 `client_task_id`/`media_type`/`data_base64`/`node_path`；结果文件两种写法都接受。
18. **命令清单不保留正文**：`shell.list` 回执里的命令文本在集成层被丢弃，只保留标识、分支、模式与开始时间；证据与诊断都不会打印命令或输出。
19. **真实模态/命令证据用例按环境变量开启**：`TestRealModalityAndShellEvidence` 需要 `DIMAGENT_MODALITY_EVIDENCE=1` 与真实服务端/模型配置/图片；默认跳过，不进入 CI。
20. **provider 模型状态与显示名是展示兜底**：真实 `provider.management` 的模型条目可能没有 `status`/`displayName`，设置领域要求 `active/beta/deprecated` 与显示名；集成层按 `active` 与模型标识兜底（只用于展示，不写回服务端）。
21. **provider 模型字段有嵌套与扁平两种写法**：真实回执的 `model` 是对象，旧夹具是字符串；`models` 也可能是数组或按标识索引的对象，映射两种都接受。
22. **agent 清单不按 hello 预判**：`agent.list` 是内建方法，真实 hello 不声明；直接调用，缺失时按 `unknown_method` 归为能力缺失。
23. **账号 provider 解析按实例缓存**：`account.providers` 解析一次；登录方式只取服务端声明，未声明的方法不会本地推断。
24. **设备码状态未知即拒绝**：只映射 pending/completed/failed/expired/canceled（含常见别名），其他取值按事实不合法报错，不改写成 pending。
25. **权限档位不默认提权**：`session.create` 不带 `permissionPolicy` 时按服务端默认策略；默认策略下工具调用会停在 `step/tool-call` 之后等待审批。集成层不替用户选 `full-access`，需要工具调用的自检/证据必须显式传档位，正常路径传用户选择。
26. **成员权限策略不可由客户端指定**：协作请求与 `agent` 工具 profile 都没有权限字段；服务端按发起分支的当前策略采样给成员。
27. **成员停止的真实端到端已跑通**：显式 full-access 下成员创建、`session.collaboration.cancel` 的 `cancelled` 与停止后事实收敛均已验证；`idle` 视为已停。
28. **`ui.*` 只做普通连接注册**：`ui.open` 的叶子路径需要 rutis 会话，不在 Go 侧实现；`ui.attach` 失败按稳定 appError 分类。
29. **`ui.shutdown` 是 advisory**：只记录并标状态，是否关窗由宿主决定；重连后的重新注册只在本地注册存在且未被建议关闭时进行。
30. **断线恢复以行标识判重**：模型可能在断线期间继续产生事实，事实总数会变；是否重复只看行/事实标识，不看总数。
31. **共享订阅的连接数观察依赖 `daemon.ping`**：它只给连接与会话计数，看不到持有者数量；持有者语义由 `session.Manager` 的租约保证。
32. **`context/budget` 永远不下发**：协议 v3 的可见性白名单把它排除；客户端只按需读 `session.context.usage`，视图里的 `Context` 不来自事实。
33. **手动压缩依赖模型窗口**：summary 策略要求压缩后确实变小，小窗口模型 + 足够内容才能成功；不满足时服务端回 `failed`，本包如实透传。
34. **成员标识只来自协作事实**：`agent.list` 只有会话计数，`session.agent.get` 是分支采纳的 agent；停止成员必须从 pending/delivered 取成员会话标识。
35. **默认预设缺失按能力缺失**：`--defaults` 文件不存在时不传给服务端、记诊断；运行组件面回到能力缺失零请求。
36. **应用更新只做单文件替换**：不处理 macOS `.app`、dmg/zip、Windows 安装器与 AppImage；没有代码签名校验，只比对 sha512。
37. **更新状态只在内存**：`switch` 与 `download` 必须在同一进程内完成状态机；进程重启后从「未检查」开始。
38. **CLI 检查失败退出 0**：检查失败按已是最新呈现，只打印原因；`download`/`switch` 的失败才返回非零。
39. **CLI 输出不含地址**：只打印状态与版本号，不打印清单/产物地址及其查询串。
40. **命令规则写入回执只带 revision**：集成层用本次写入的集合补齐 `Rules`，并保留服务端 revision；冲突时不覆盖缓存、不重试。
41. **命令规则缓存只在裁决内存里**：适配器不缓存、不落盘；缺能力确认后由裁决记住，后续不再请求。
42. **变更读取是工具声明级的部分覆盖**：清单只含工具结果明确声明的路径（`coverage=tool_reports`），不含会话内命令（exec）的改动、外部程序修改、没有产生声明的写入；不等于"本次运行改了哪些文件"的权威结论。
43. **变更读取按轮归属**：`runID` 取事实里的 `turn_id`；同一轮之外的工具结果一律不看。历史读取上限 40 页 × 500 条，超出直接报错。
44. **声明字段只认约定形状**：只解析 `structuredContent.affectedPaths`、`write`/`edit` 的 `structuredContent.path` 与 `fileWritten` effect；其它结构不猜，宁可少列也不推断。
45. **重读尾部用 `after` 游标而非 `latestTurn`**：兼容拒绝未知字段的旧服务端；一次重读最多 5 页。
46. **自由文本诊断与白名单诊断分开**：`App.Diagnostics()` 只给人看、不进 `host.Recorder`。
47. **超时阶段的 goroutine 不强制回收**：外壳按时进入下一段；本包步骤尊重上下文，唯一例外是终端停止本身的宽限期。

## 11. 变更记录

| 日期 | 变更 | 原因 |
|---|---|---|
| 2026-11-10 | 第一版：装配、Sender/Inspector 适配器、启动与退出、只读视图、自检 | 集成层结果包首次提交 |
| 2026-11-10 | 接到 D10 宿主外壳：阶段函数、写闸门、诊断、二次启动、完整启停自检 | 组织要求覆盖宿主接线 |
| 2026-11-10 | 接掉连接侧补齐的 wire：审批三类通知、标题通知、四类清单注入（含提问清单与时间线）、`input_lookup` 核对优先级、`attachment.put` 附件写入与能力门禁 | 组织要求把已冻结的 wire 形状接进集成层 |
| 2026-11-10 | 补三类写意图：`KindShellRun`（`shell.run`/`shell.stop`）、`KindModalityWrite`（四种模态写入）、`KindSessionLifecycle` 的 `branch.create`/`branch.end`；发送方加写闸门兜底 | 写登记新增意图，集成层补齐映射与门禁 |
| 2026-11-11 | 接上工作区变更读取：`workspace.ChangeSource` 适配器（工具结果声明、按 run 过滤）+ `App.RunChanges`；来源实例挂在 App 上 | D7 新增变更读取，需要集成层提供来源 |
| 2026-11-11 | 按 alpha.21 实测改真实命名与形状：账号（providers/status/usage/provider.catalog/login.browser.*）、管理面（mcp.*/schedule.*/session.wakeup.*/skills.catalog/hooks.catalog）、改名走会话流、权限 `session.permission.get/update` + 时间线 `permission/changed`、统一二进制自启补 `app-server`；长期允许改为命令规则（`permission.rules`/`permission.set_rules`） | 连接侧实测出真实命名；管理面与权限需要真实接线 |
| 2026-11-11 | 账号用量扩展到 plan/credits/windows/features/订阅链接与标识：字段缺失保持 nil，价格最小单位原样透传 | 账号领域新增用量字段 | 
| 2026-11-11 | 纠正权限默认值：selfcheck `-permission-preset` 缺省为空（不默认提权），工具类证据显式传 full-access；文档补充“成员策略由服务端按发起分支采样，客户端不可指定” | 默认 full-access 是提权隐患 | 
| 2026-11-11 | 成员停止真实端到端跑通：根因是新建会话未显式带权限档位（默认策略等审批）；`catalog.Create`/`App.CreateSession` 增加 `PermissionPreset`，selfcheck 缺省为空、证据用例显式传 full-access；真实 `cancelled` 与成员事实收敛已验证 | 复盘结论收尾 | 
| 2026-11-11 | 成员停止真实尝试：模型调用 `agent` 一次但未产生协作请求/投递，5 分钟轮次未结束，按 blocked 记录 | 收尾 A 组最后一条 | 
| 2026-11-11 | 界面注册：`ui.attach`/`ui.close`/`ui.status`，建议性 `ui.shutdown`，detach/reattach 跟随与失败码分类 | 连接侧查清普通注册路径 | 
| 2026-11-11 | A 组集成层真实证据：断线恢复、多窗口共享订阅、未知结果核对、附件真实写入、运行中停止 | 收尾真实证据 | 
| 2026-11-11 | 上下文占用（`session.context.usage`）与手动压缩（`session.context.compact`）、成员停止（`session.collaboration.cancel`）、默认预设（`--defaults` + `runtime.list` 门禁）；账号邮箱透传/设备码取消/provider 绑定清理；MCP 导入新形状 | 投影与扩展领域查明真实形状；账号与 MCP 导入补齐 | 
| 2026-11-11 | 接上应用更新：装配 `host.Updater`（当前版本、渠道源、单文件目标、会话空闲探针、退出五阶段、重启、诊断），`dimagent update check/download/switch` 子命令 | 更新收尾；真实渠道离线未请求 | 
| 2026-11-11 | 接上命令规则来源 `interaction.RuleSource`：`permission.rules` 读取、`permission.set_rules` 带版本写入、冲突透传不重试 | 交互领域新增命令规则面 | 
| 2026-11-11 | 新增扩展写入（项目钩子启停、技能刷新/重载、MCP 认证四步）与设置适配器（provider 清单/详情/探测、协作策略、agent 档案），账号按 provider 分发（browser/device 来源） | 扩展与设置领域新增接口；账号领域改为按 provider 分发 | 
| 2026-11-11 | 门禁改 hello 优先；命令与模态接真实调用：`shell.run/stop/list`（含 record 回写与停止收敛）、`settings.ModalitySession` 适配器（任务提交/状态/结果/取消 + 实时发送）；证据不含命令与输出 | 真实服务端把 `modality.*` 只声明在 hello、`shell.*` 只声明在会话方法表 |
