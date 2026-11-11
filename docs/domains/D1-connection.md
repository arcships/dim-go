# D1 连接（protocol v3 wire、daemon 生命周期、会话流）

Status: 第六版（第一版 + v3 wire + 管理面/权限档位 + 卡点调查 + 宿主侧调查 + Go UI 叶子验证，P1-1 候选结果包）
Owner: D1 领域 agent
验证: 可执行命令 + 可观察输出见第 6 节；真实 daemon 证据在 6.3–6.5，调查结论在 10
依据: `docs/domains/R.md` 第 1 部分功能块 1、`docs/domains/A.md`（连接状态机、会话能力归属、四条总规矩）、`docs/project-organization.md` §6 F3
范围: 协议编解码、本地传输、daemon 发现与归属、握手、会话流与监听、重连、错误分类。不含界面投影、目录、审批、写登记、账号、文件、终端。

---

## 1. 输入

| 输入 | 来源 | 本版使用方式 |
|---|---|---|
| protocol v3 wire 定义 | 产品基线 `app-server-protocol/src/{wire,ndjson,client}.ts`、`dim-app-server/src/{protocol,transport,tree_protocol}.rs` | 在 `internal/appserver/protocol` 中只定义一次；其余包只用这些类型 |
| 连接与会话流生命周期 | `dim-electron` 的 `app-server/{supervisor,transport,session-reader,daemon-link}.ts`，`docs/architecture/app-server.md` | 端点、自启参数、hello 校验、attach/detach、按序号重读 |
| 错误语义 | 产品基线 `app-server/error-mapping.ts` | 映射为连接领域的五种分类，供上层失败关闭 |
| 状态机 | `docs/domains/A.md` 的「连接」「启动与退出」、多窗口共用一条订阅 | 连接状态与订阅生命周期按此实现 |
| 本地证据 | 本机 `~/.dimcode/rust/dimcode-server`（`dimcode app-server` 子命令，0.1.0-alpha.7）与仓库外二进制 `/tmp/dim-2-target/debug/dim-app-server`（0.1.0-alpha.18） | 临时数据目录 + mock 模型跑真实握手、打开会话与回收，覆盖新旧能力分支 |

## 2. 产物

### 2.1 包结构

| 包 | 一句话职责 |
|---|---|
| `internal/appserver/protocol` | JSON-RPC 2.0 over NDJSON：帧编解码与上限、消息解析、请求关联与超时、通知分发、关闭语义；v3 wire 类型（审批、会话通知、未完成事项、提交核对、附件、方法清单、管理面、权限档位、shell、唤醒、计划任务、模态、UI 宿主）只在这里出现 |
| `internal/appserver/transport` | 本地传输入口：Unix domain socket 实现；Windows 命名管道保留接口并明确返回不支持 |
| `internal/appserver/daemon` | 数据目录 → 端点解析、自启与归属、回收、hello 握手与版本校验、控制连接与流连接 |
| `internal/appserver/session` | 每会话一条流、一份订阅；订阅先于 attach 与历史；持有者计数；断线按序号重读 |
| `internal/rutis` | UI 叶子的最小 rutis 会话：私有 Unix socket、hello/握手、newline 帧、保持连接；不提供插件目录与业务方法 |
| `internal/appserver/failure` | 五种错误分类与 `errors.Is` 支持，供上层决定重试或失败关闭 |
| `cmd/dimagent-conncheck` | 真实 daemon 验收程序：握手、打开会话、观察通知、回收自启实例 |

### 2.2 关键约定

- **wire 只出现一次**：方法名、参数与回执形状、通知参数、`RawEvent`、错误对象都在 `protocol` 包；`daemon` 与 `session` 不重复声明 wire 结构。
- **不存事实**：本领域不读 daemon 的 SQLite、JSONL、session 文件与 `daemon.log`，不在本地保存会话或事件历史；自启子进程的 stdout/stderr 直接丢弃，失败只带退出状态。
- **订阅唯一**：同一会话无论多少个持有者只有一份订阅与一条连接；最后一个持有者释放才 `session.detach` 并断开。
- **订阅先于历史**：连接建立后先登记通知，再 `session.attach`，再读历史；attach 重放的通知在历史读取期间缓冲，不丢。
- **按序号重读**：断线后以 `deliveredThrough` 为起点，用 `session.history {after}` 重读；序号不倒退、不重复交付。
- **失败关闭**：能力缺失、参数错误、未知 `appError` 一律不可重试；只有超时、连接被打断、服务忙、订阅变化可重试。

## 3. 设计

### 3.1 帧与消息

- NDJSON 增量解码：任意分块边界、空行忽略、行尾 `\r` 去掉；单帧上限默认 16 MiB。
- 坏帧处理：超限或非法 JSON 的整行被丢弃并报错一次，解码器不失步，后续帧继续交付；超限的未结束行进入丢弃状态直到换行，避免无界缓冲。
- 消息校验与产品基线一致：`jsonrpc` 必须是 `2.0`；带 `method` 的消息是通知且不得带 `id`；回执必须恰好有 `result` 或 `error` 之一；错误对象的 `code` 与 `message` 必填，`data.appError` 可选但类型必须正确。
- 未知通知方法原样保留并交给监听器；未知回执标识静默忽略，与基线一致。

### 3.2 请求关联与超时

- 标识是 wire 原始文本（数字或字符串），`1` 与 `"1"` 不混淆；请求先登记再写出，快速回执不会漏。
- 默认单请求超时 30 秒；`NoTimeout` 关闭超时。超时返回 `*protocol.TimeoutError`，可被 `errors.Is(err, protocol.ErrRequestTimeout)` 识别。
- 连接关闭时所有在途请求以 `*protocol.ClosedError` 结束；关闭原因统一包成该类型，保证分类稳定。

### 3.3 端点、自启与归属

- 数据目录 → `<dataDir>/rust` 为 App Server home：非 Windows 要求 0700 且属于当前用户，路径按真实路径解析；Unix 端点固定为 `<home>/daemon.sock`。
- Windows 管道名与产品基线同一规则：`\\.\pipe\dim-rust-<sha256("dimcode:"+home)[:32]>`。
- 发现顺序：先连已有实例；只有「端点不存在/无人监听」才拉起自己的实例。权限错误等非缺失错误直接上报，不擅自替换别人的服务。
- 自启参数：`--unix <sock> --no-ws --home <nativeHome>`；mock 测试追加 `--mock-model`，否则默认 `--product-home <dataDir>`，显式模型配置用 `--model-config`。子进程脱离子进程组、标准输入输出丢弃。
- 归属：只有自己拉起的实例才会被回收。回收优先用 `daemon.shutdown {ifIdle:true}`（服务端声明 `daemon.shutdown.ifIdle` 时）；旧服务端用 `daemon.ping` 确认没有其他客户端后再 `daemon.shutdown {force:false}`。其他客户端、忙、非自己拉起都如实返回 `Reason`，不接管。
- 重连：丢弃控制连接重新握手；自己拥有的实例消失时重新拉起；`instanceId` 变化返回 `Restarted`，上层据此重建而不是续接旧序号。

### 3.4 握手校验

`hello {protocolVersion:3, clientKind:"other", clientName}` 的回执必须满足：

1. `protocolVersion == 3`，否则 `version_incompatible`，并记为终态（之后不再拨号）；
2. `instanceId` 非空；
3. `daemonProfile.kind` 是 `product` / `mock` / `configured`，`configured` 必须有非空 `modelConfigSha256`。

服务端以 `appError=protocol_version` 拒绝时同样归为版本不兼容。

### 3.5 会话流

打开顺序：独立连接 → hello → 登记通知 → `session.attach` → 读历史。

- 头部序号取 attach 回执的 `sequence`；旧服务端没有该字段时补一页 `session.history {after:0, limit:1}` 取 `through`。
- 初始历史：服务端声明 `session.history.latestTurn` 时读最新一页（`through=head`）；否则从起点读一页，后续窗口与锚点归会话投影。
- 历史之后开始投递实时通知；attach 重放中序号已被历史覆盖的 `session.event` 被丢弃，之后的按到达顺序交付。
- 断线：调用一次 `OnResync`（带上次交付序号），按 `after=FromSequence` 重读并交给 `OnHistory`，然后恢复实时投递；版本不兼容或不可重试错误进入 `OnFatal` 终态。
- 持有者：`Acquire` 递增计数并返回 `Lease`；首个持有者提供监听器，后续持有者共享事实分发；全部释放后 detach（2 秒上限，尽力而为）并关闭连接。

### 3.6 错误分类

| 分类 | 触发 | 上层动作 |
|---|---|---|
| `version_incompatible` | hello 版本不符、`appError=protocol_version/session_protocol_version` | 停止重试，直到进程重启或服务被替换 |
| `connection_missing` | 端点不存在、连接被拒 | 重新发现或拉起服务，不盲目重发请求 |
| `timeout` | 请求超时、`context.DeadlineExceeded`、`appError=request_timeout` | 恢复连接后可重试；写操作结果未知由写登记处理 |
| `retryable` | 连接关闭/中断、`service_reloading`、`connection_busy`、`subscription_changed` | 重连后重试 |
| `non_retryable` | 能力缺失（`unknown_method`）、参数错误、未识别错误 | 失败关闭并向用户报缺失 |

### 3.7 v3 wire 形状（第二轮补齐）

方法名与字段以真实服务端为准：本机 `/tmp/dim-2-target/debug/dim-app-server`（0.1.0-alpha.18）与其同版本源码 `dim-2` 的 `crates/dim-app-server/src/{protocol,tree_protocol,storage/metadata}.rs`、`crates/dim-agent-{questions,attachments}`、`crates/dim-agent-capabilities/src/{input_queue,workspace_tools}/pending.rs`。能触发的形状用真实 daemon 跑过（见 6.4），不能触发的在 6.6 标 blocked。

**审批通知**（`approval.request` / `approval.cancelled` / `approval.resolved`）

| 方法 | 字段 |
|---|---|
| `approval.request` | `sessionId`,`branchId`,`requestId`,`source`,`call`,`reason`；`source.kind=tool` 时带 `turn_id`,`request_id`,`call_id`，`kind=host` 时带 `operation_id`；`call={name,input}` |
| `approval.cancelled` | `sessionId`,`branchId`,`requestId`,`reason`（调用未开始） |
| `approval.resolved` | `sessionId`,`branchId`,`requestId`,`decision`（`allow`/`deny`）；同一 `requestId` 的第一条是被接受的决定 |

**会话通知与标题**（`session.notice`）

`{sessionId, source, kind, data}`；标题变更是 `source="SessionStore"`、`kind="title"`、`data={title|null, titleSource|null, titleRevision>=1}`。与 v3 会话摘要的标题字段相同；`titleRevision` 更高才覆盖本地条目。

**提交核对**（`session.input_lookup`）

请求 `{sessionId, branchId, submissionId}`：v3 要求 `branchId` 必填，`submissionId` 必须是 UUID。回执二选一：`{status:"accepted", event}` 或 `{status:"unknown"}`。写登记据此结束「结果未知」。

**四类未完成事项**（共同的截止点与游标）

请求 `{sessionId, branchId, through?, after?}`；`through` 是固定截止点，`after` 必须把上一页的 `next` 原样带回。回执共同部分 `{branchId, through, next}`，`next` 为 null 表示没有下一页。

| 方法 | 列表字段 | 条目 |
|---|---|---|
| `session.input.pending` | `inputs` | `inputId`,`sequence`,`mode`(queue/steer/append),`user`(可 null，`content[]` 原样),`applied` |
| `session.exec.pending` | `tasks` | `eventId`,`sequence`,`startedAt`,`taskId`,`turnId`,`requestId`,`callId`,`description` |
| `session.question.pending` | `batches` | `eventId`,`sequence`,`requestId`,`calls[]`（`id`,`name`,`input`） |
| `session.collaboration.pending` | `requests` | `eventId`,`sequence`,`startedAt`,`request`,`delivery`(可 null)；`request` 为 `turn_id`,`request_id`,`call_id`,`target`,`prompt`，`target` 以 `kind` 标记（`agent`/`member`/`temporary`）；`delivery` 以 `status` 标记（`ok`/`error`），字段 snake_case |

**提问与征询没有通知**：真实服务端的通知表里没有 `question.*`/`elicitation.*`。提问是时间线事实（`step/tool-call`，`data.calls[].name="ask_user"`）加 `session.question.pending` 读取；「已决」「超时清理」属于交互裁决按事件与本地期限推导，连接领域不伪造通知。

**附件**（`attachment.put`）

请求 `{sessionId, name?, text}`（`name` 省略时服务端用 `paste.txt`；文本上限 10 MiB）；回执 `{attachmentId, path, bytes}`。错误码 `attachment_invalid` / `attachment_too_large` / `node_unavailable` / `storage_failed`。该方法是会话贡献方法：只在提供本地执行与原生存储的实例上存在，必须以 `session.methods` 判定，不能按版本猜。

**能力清单**（`session.methods`）

回执 `{methods:[{name, mutating}]}`；上表所有贡献方法（`attachment.put`、四类 pending、`session.question.respond` 等）都应先在这里判定。

### 3.8 管理面方法（alpha.21 实测 + 同版本/上游源码核对）

本机真实服务端为 `~/.dimcode-alpha/rust/dimcode-server`（0.1.0-alpha.21）。能调用的方法都实际调过（临时数据目录 + 复制到 `/tmp` 的离线模型配置），失败或不存在的方法记录原始错误。

**存在的方法（按家族）**

| 家族 | 实际 wire 方法 | 备注 / 类型 |
|---|---|---|
| 插件 | `plugin.list` / `install` / `enable` / `disable` / `reload` / `update` / `remove` | list 返回**数组**；条目 `{name,use,off,locked,candidates[],status?,error?,refused?}`；install 参数 `{source:{path}|{git,ref?},id?,enable?}`；enable/disable/reload `{name,use?}`；update/remove `{id,ref?}`；enable 回执 `{status}`，disable `{}` |
| 技能 | `skills.catalog` / `install` / `installations` / `preview` / `refresh` / `remove` / `set_enabled` / `update` | catalog `{version,entries[],issues[]}`；entry `{id,source,name,description,filePath,allowedTools[],enabled,shadowedBy}`；installations `{installations[]}` |
| 钩子 | `hooks.catalog` / `save` / `set_enabled` | catalog `{userRevision,userDocument,projectsRevision,disableAllHooks,sources[],hooks[]}`；save `{expectedRevision,document}` |
| 外部工具服务器 | **实际命名是 `mcp.*`**：`mcp.catalog` / `configuration` / `inspect` / `save` / `remove` / `move` / `update` / `set_enabled` / `test` / `import.sources` / `import.preview` / `import.apply` / `auth.*` / `disconnect` | catalog `{userRevision,projectRevision,userApplication,pluginApplication,projectApplication,servers[],invalid[]}`；server `{id,source,enabled,transport,auth?}`；import.sources `{sources[]}` |
| 计划任务 | `schedule.list` / `save` / `delete` / `set_enabled` / `history` / `run_now` / `watch` | list `{jobs[]}`，job 为 snake_case；save `{job,revision?}` → `{transition?,job}`；delete `{job_id}`；set_enabled `{job_id,enabled}` → `{job}`；history `{job_id,limit?}` → `{runs[]}`；watch 走 `management.progress` |
| 存储 | `storage.report` | `{databaseBytes,walBytes,reclaimableBytes}` |
| 会话标题 | `session.rename` | 参数 `{sessionId,title|null}`；回执 `{title,titleSource,revision}` |
| 本机 shell | `shell.list` / `run` / `stop`（会话贡献） | run `{sessionId,branchId,runId,command,mode}` → `{runId,status,output,truncated,durationMs,exitCode?,signal?,reason?,recorded?}`；list `{sessionId}` → `{runs[]}`；stop `{sessionId,runId}` → `{stopped}` |
| 唤醒记录 | `session.wakeup.create` / `list` / `cancel`（会话贡献） | create `{sessionId,branchId,at,prompt}` → `{wakeupId}`；list `{sessionId,branchId?}` → `{wakeups[{wakeupId,at,prompt,state}]}`；cancel `{sessionId,wakeupId}` → `{}` 或 `{state}` |
| 模态 | `modality.capabilities` / `configure` / `task.submit|status|list|wait|cancel|result` / `realtime.open|send|close` | capabilities `{revision,categories[]}`；task 记录 `{version,task_id,client_task_id,revision,category,operation,executor,params,state,remote_handle,progress,result,error,remote_cancel,created_at,updated_at,finished_at}`；realtime open `{connectionId,transport,connection?}` |
| 附件 | `attachment.put` | 见 3.7 |

对应 Go 类型（均在 `internal/appserver/protocol`）：

- 权限：`PermissionPolicy`、`PermissionResult`、`PermissionGetParams`、`PermissionUpdateParams`、`PermissionChanged`
- 标题：`SessionRenameParams`、`SessionRenameResult`
- 存储/进度：`StorageReportResult`、`ManagementProgressNotice`
- 插件：`PluginListResult`、`PluginEntry`、`PluginCandidate`、`PluginSource`、`PluginInstallParams`、`PluginNameParams`、`PluginIDParams`、`PluginOperationResult`
- 技能：`SkillsCatalogResult`、`SkillEntry`、`SkillIssue`、`SkillsRefreshResult`、`SkillsInstallationsResult`、`SkillInstallation`、`SkillInstallationSkill`、`SkillsSetEnabledParams`、`SkillsPreviewParams`、`SkillsInstallParams`、`SelectedSkill`、`SkillsInstallationParams`
- 钩子：`HooksCatalogResult`、`HookSource`、`HooksSaveParams`、`HooksSaveResult`、`HooksSetProjectEnabledParams`、`HooksSetProjectEnabledResult`
- MCP：`MCPCatalogResult`、`MCPCatalogEntry`、`MCPAuthStatus`、`MCPInvalidCatalogEntry`、`MCPImportSourcesResult`、`MCPImportSource`、`MCPConfigurationResult`、`MCPTargetParams`、`MCPSetEnabledParams`、`MCPRemoveParams`、`MCPMoveParams`、`MCPSaveParams`、`MCPImportPreviewParams`、`MCPImportApplyParams`
- 计划任务：`ScheduleListResult`、`ScheduleJob`、`ScheduleSaveParams`、`ScheduleSaveResult`、`ScheduleTransition`、`ScheduleDeleteParams/Result`、`ScheduleSetEnabledParams/Result`、`ScheduleHistoryParams/Result`、`ScheduleRun`、`ScheduleRunNowParams`、`ScheduleWatchParams`、`ScheduleWatchItem`
- 模态：`ModalityCapabilitiesResult`、`ModalityCategory/Operation/Option/Effective/Reason`、`ModalityConfigureParams`、`ModalityTaskSubmitParams`、`ModalityTaskStatusParams`、`ModalityTaskWaitParams`、`ModalityTaskResultParams`、`ModalityTaskRecord`、`ModalityTaskReply`、`ModalityTaskListParams/Result`、`ModalityTaskResultReply`、`ModalityRealtimeOpenParams/Result`、`ModalityRealtimeSendParams`、`ModalityRealtimeCloseParams`、`ModalityRealtimeEventNotice`、`ModalityRealtimeEndedNotice`
- shell：`ShellRunParams`、`ShellRunResult`、`ShellRunRecorded`、`ShellListParams`、`ShellListResult`、`ShellRunInfo`、`ShellStopParams`、`ShellStopResult`
- 唤醒：`SessionWakeupCreateParams`、`SessionWakeupCreateResult`、`SessionWakeupListParams`、`SessionWakeup`、`SessionWakeupListResult`、`SessionWakeupCancelParams`、`SessionWakeupCancelResult`、`WakeupNoticeData`
- 运行时（仅带默认预设时存在）：`RuntimeListResult`、`RuntimeEntry`、`RuntimeInstallParams`、`RuntimeConfigureParams`、`RuntimeWatchItem`
- 协作成员停止：`SessionCollaborationCancelParams`、`SessionCollaborationCancelResult`、`CollaborationDeliveredData`
- UI 宿主：`UIOpenParams`、`UIOpened`、`UIAttachParams`、`UIAttached`、`UICloseParams`、`UIReplaceParams`、`UIShutdownParams`、`UIShutdownNotice`、`UIStatusResult`、`UIInstance`

**不存在的方法（原样调用返回 `-32601 unknown_method`）**

| 被问到的方法 | 结论与源码依据 |
|---|---|
| `toolServer.*`（含 import preview/apply） | 不存在。工具服务器在服务端叫 MCP：alpha.8 源码 `crates/dim-app-server/src/mcp/management.rs` 以命名空间 `mcp` 注册（`methods.register(ctx, "mcp", ...)`）；alpha.21 实测 `mcp.*` 全部可用，`toolServer.*` 全部 unknown_method |
| `workflow.*` | 不存在。alpha.8 全仓 `grep -r "workflow\." crates/dim-app-server/src` 无注册；alpha.21 实测 `workflow.list/status/start/stop/save` 全部 unknown_method。Electron 侧有类型，但当前服务端没有该方法面 |
| `scheduler.*` | 不存在。任务面实际叫 `schedule.*`：`crates/dim-agent-scheduler/src/methods.rs` 注册 `schedule.list/save/delete/set_enabled/history/run_now/watch`；`scheduler.*` 实测 unknown_method |
| `runtime.*`（list/install/configure/watch） | alpha.21 不存在（全部 unknown_method）。alpha.8 源码 `crates/dim-app-server/src/runtime_manager.rs` 注册过 `runtime.list/install/configure`（无 `runtime.watch`），说明该面在 alpha.21 已不再随服务端注册；当前版本按能力缺失处理 |
| 顶层 `wakeup.*` | 不存在。唤醒是会话方法 `session.wakeup.*`：`crates/dim-agent-scheduler/src/session.rs` 注册 `session.wakeup.create/list/cancel`；`wakeup.list` 实测 unknown_method |
| `skills.list` / `hooks.list` | 不存在；实际读取方法是 `skills.catalog` 与 `hooks.catalog`（alpha.8 `skills_management.rs` / `hooks_settings.rs` 的 MethodDecl 与 alpha.21 实测一致） |
| `permission.changed` | **不是通知**，详见 3.9 |

### 3.9 权限档位

- 读取：`session.permission.get {sessionId,branchId}` → `{version,policy}`。
- 切换：`session.permission.update {sessionId,branchId,expectedVersion,policy}` → `{version,policy}`；`policy={preset,allowedTools[],disallowedTools[]}`，`preset` 为 `read-only`/`workspace-write`/`full-access`。
- 通知：**没有** `permission.changed`。服务端通知表（alpha.8 `protocol.rs` 的 `notifications!` 表，含 `approval.*`、`session.*`、`modality.realtime.*` 等，无 permission 项）里没有它；切换档位产生的是时间线事实 `session.event {event.type:"permission", event.name:"changed", data:{policy}}`。alpha.21 实测：update 后控制连接与流连接上没有 `permission.changed`，流上收到 `session.event` 的 `permission/changed`，`data.policy` 为新档位。
- 会话能力归属不变：档位本身由交互裁决持有；连接领域只冻结 wire 形状与事件转发。

## 4. 状态映射

| A.md 状态机 | 本领域实现 |
|---|---|
| 未连接 → 发现中 → 握手中 → 已连接 | `Supervisor.Connect`：先发现，缺失才自启；hello 成功即已连接 |
| 握手中 → 版本不兼容 → 停止 | `setTerminal`；后续 `Connect`/`OpenStream`/`Reconnect` 直接返回同一错误，不再拨号 |
| 已连接 → 断开/被替换 → 重连中 → 已连接（先重订后重读） | `Supervisor.Reconnect`（控制连接）与 `Subscription.onLost`（会话流）；会话流先 attach 再按序号读历史 |
| 多窗口共用一份订阅 | `Manager` + `Lease` 持有者计数；连接数不随窗口增长 |
| 启动与退出：连服务 → 订阅 → 断连接 → 回收自启服务 | `Connect` / `Acquire` / `Lease.Release`、`Supervisor.Close` / `Supervisor.Shutdown` |
| 会话能力由会话视图持有 | 本领域只给出 `hello.features` 与 attach/历史结果；不缓存、不推导能力 |

## 5. 接口（F3 冻结候选）

```go
// 连接状态
sup, err := daemon.New(daemon.Config{DataDir: ..., ServerPath: ..., MockModel: false})
conn, err := sup.Connect(ctx)                  // 控制连接
stream, err := sup.OpenStream(ctx)             // 会话流连接（已 hello）
result, err := sup.Reconnect(ctx)              // Restarted 表示实例被替换
shutdown, err := sup.Shutdown(ctx)             // 只回收自己拉起的实例

// 会话流
manager, err := session.NewManager(session.Config{Opener: sup})
snapshot, lease, err := manager.Acquire(ctx, sessionID, session.Listener{
    OnEvent: func(event session.Event) {
        // event.Method/Params 原样保留；已识别的通知另有类型化字段：
        //   event.ApprovalRequest / ApprovalCancelled / ApprovalResolved
        //   event.SessionNotice / event.Title
        // 解码失败只设置 event.DecodeErr，事实仍交付。
    },
    OnHistory: func(session.HistoryReplay) {}, // 断线后的按序号重读
    OnResync: func(session.Resync) {},         // 断线通知，先于重读
    OnFatal: func(error) {},                   // 终态
})
sub := lease.Subscription()
sub.Methods(ctx)                                // 能力清单
sub.InputLookup(ctx, protocol.InputLookupParams{...}) // 写登记核对
sub.PendingInputs(ctx, protocol.PendingQuery{...})
sub.PendingExecTasks(ctx, ...)                  // 四类未完成事项
sub.PendingQuestions(ctx, ...)
sub.PendingCollaboration(ctx, ...)
sub.PutAttachment(ctx, protocol.AttachmentPutParams{...})
sub.Call(ctx, method, params)                   // 其它会话方法：仍走这条流
lease.Release()                                 // 最后一个持有者才 detach

// 错误分类
failure.KindOf(err) / failure.Is(err, failure.KindTimeout) / failure.Retryable(err)
```

消费方（D2–D6、D10）只需要这些类型与 `protocol` 的 wire 结构；不需要知道 socket、NDJSON 或自启细节。

## 6. 验收

### 6.1 命令与结果

```bash
go test ./...
go vet ./...
```

本版结果：`go test ./...` 全通过（连接领域 5 个包均 `ok`），`go vet ./...` 无输出；`go test -race ./internal/appserver/...` 全通过。

### 6.2 覆盖

| 要求 | 测试 |
|---|---|
| 帧编解码：分块、CRLF、空行、超大整帧、超限未结束行、坏帧后继续 | `protocol/frame_test.go` |
| 消息校验：版本、通知/回执形状、未知通知保留、错误对象 | `protocol/message_test.go` |
| 请求关联与超时：乱序回执、超时、RPC 错误、关闭拒绝在途 | `protocol/client_test.go` |
| 通知分发：未知方法、坏帧不打断 | `protocol/client_test.go`、`session/session_test.go` |
| 通知在订阅前到达不丢、历史覆盖的不重复 | `TestAcquireSubscribesBeforeHistoryAndKeepsEarlyNotification` |
| 持有者计数：两个持有者一条流、逐个释放、最后才 detach、重复释放无效、释放后重开新流 | `TestHoldersKeepOneSubscriptionUntilAllReleased` |
| 断线重连：resync 起点、按序号重读、实时恢复、无重复 | `TestReconnectResumesFromSequence` |
| 会话流协议版本不符停止重试 | `TestVersionIncompatibleStopsReconnecting` |
| 连接协议版本不符停止重试（不再拨号） | `daemon.TestVersionIncompatibleStopsRetrying` |
| 连接缺失 → 自启 → 握手 → 回收自启实例 | `daemon.TestSpawnOwnedDaemonAndReclaimIt` |
| 自启实例在握手始终不完成时被回收 | `daemon.TestSpawnedDaemonIsReclaimedWhenHandshakeNeverCompletes` |
| `daemon.shutdown.ifIdle` 分支：空闲即停、忙则保留并报数量 | `daemon.TestShutdownUsesIdleDecisionWhenAdvertised` |
| 已有实例不归自己、退出不回收外部实例 | `daemon.TestConnectToExistingDaemon`、`TestShutdownLeavesExternalDaemon` |
| 服务被替换后恢复并报告 `Restarted` | `daemon.TestReconnectToReplacedDaemon` |
| 自己拉起的实例退出后重连会重新拉起 | `daemon.TestReconnectRelaunchesOwnedInstance` |
| 端点私有目录、Unix 路径、Windows 管道命名、自启参数 | `daemon/endpoint_test.go` |
| 传输：Unix 往返、管道未实现、未知网络 | `transport/transport_test.go` |
| 错误分类与 `errors.Is` | `failure/failure_test.go` |
| 管理面 wire：权限、改名、存储、进度、插件、技能、钩子、MCP、计划任务、模态、shell、唤醒（未知字段容忍 + 缺字段拒绝） | `protocol/management_test.go` |
| 管理通知映射：permission/changed、唤醒通知、management.progress、模态实时事件 | `session.TestManagementNotificationsAreTyped` |
| 审批三类通知、标题通知、提交核对、四类未完成事项、附件回执、方法清单的解码（未知字段容忍 + 缺字段拒绝） | `protocol/wire_v3_test.go` |
| 通知映射到类型化事件；坏字段保留原始 Params 并报 `DecodeErr` | `session.TestRecognizedNotificationsAreTyped` |
| 会话方法路由：`session.methods`、`input_lookup`、pending、`attachment.put` 都带会话标识走本会话的流 | `session.TestSessionCallsRouteWithSessionID` |

### 6.3 真实 daemon 证据

环境：macOS arm64；均为临时数据目录 + `--mock-model`；服务端路径都在本仓库之外。

**证据 A：旧版服务端（`dimcode app-server`，0.1.0-alpha.7）。**

```bash
D=$(mktemp -d /tmp/d1-check-XXXXXX)
go run ./cmd/dimagent-conncheck -data-dir "$D" \
  -server "$HOME/.dimcode/rust/dimcode-server" -server-arg app-server -mock
```

观察到的输出（节选；`<data>` 为临时目录）：

```json
{"event":"hello","endpoint":"unix:/private<data>/rust/daemon.sock","instanceId":"9be62dd2-...","serverVersion":"0.1.0-alpha.7","protocolVersion":3,"profile":"mock","owned":true,"features":["session.summaries","daemon.shutdown.completed","session.history","session.input_lookup","session.create","session.delete","session.model.progress","session.methods","session.fork","web.open","ui.leaf"]}
{"event":"session.created","sessionId":"016d03d5-...","cwd":"<data>/workspace"}
{"event":"session.opened","head":1,"historyEvents":1,"historyThrough":1,"historyDone":true}
{"event":"notification","method":"session.state"}
{"event":"notification","method":"session.model.progress"}
{"event":"session.released","events":2,"resyncs":0,"fatal":null}
{"event":"shutdown","stopped":true}
```

这条证据同时验证了兼容路径：该服务端不声明 `session.history.latestTurn` 与 `daemon.shutdown.ifIdle`，首读走 `after:0` 的一页，回收走 ping 守卫后的 `daemon.shutdown {force:false}`。

**证据 B：较新服务端（仓库外构建的独立二进制，0.1.0-alpha.18）。**

```bash
D=$(mktemp -d /tmp/d1-new-XXXXXX)
go run ./cmd/dimagent-conncheck -data-dir "$D" \
  -server /tmp/dim-2-target/debug/dim-app-server -mock
```

观察到的输出（节选）：

```json
{"event":"hello","instanceId":"2c2a42a2-...","serverVersion":"0.1.0-alpha.18","protocolVersion":3,"profile":"mock","owned":true,"features":["session.summaries","daemon.shutdown.completed","daemon.shutdown.ifIdle","session.catalog",...,"session.history.latestTurn",...]}
{"event":"session.created","sessionId":"155bbe47-..."}
{"event":"session.opened","head":1,"historyEvents":1,"historyThrough":1,"historyDone":true}
{"event":"notification","method":"session.state"}
{"event":"notification","method":"session.model.progress"}
{"event":"session.released","events":6,"resyncs":0,"fatal":null}
{"event":"shutdown","stopped":true}
```

这条证据覆盖了首读 `latestTurn` 与回收 `daemon.shutdown {ifIdle:true}` 分支。

**归属证据。** 同一数据目录第二次运行（服务由第一个客户端拉起并用 `-keep` 留着）：

```json
{"event":"hello",...,"owned":false,...}
{"event":"shutdown","stopped":false,"reason":"not_owned"}
```

结论：非自己拉起的实例不被回收；两次测试结束后，两份临时目录与自启进程都已确认不存在（`ps` 按临时路径过滤为空），未触碰用户 `~/.dimcode` 下的服务与文件。

失败即报：`dimagent-conncheck` 在任一步返回错误时以非零码退出。

### 6.4 第二轮 wire 证据（真实 daemon）

环境同上；`-probe` 会逐项读取方法清单、四类未完成事项、提交核对与附件写入。

```bash
# mock profile
D=$(mktemp -d /tmp/d1-wire-mock-XXXXXX)
go run ./cmd/dimagent-conncheck -data-dir "$D" \
  -server /tmp/dim-2-target/debug/dim-app-server -mock -probe -notice-seconds 0

# configured profile（离线模型配置，只用来建会话，不调用模型）
D=$(mktemp -d /tmp/d1-wire-conf-XXXXXX)
go run ./cmd/dimagent-conncheck -data-dir "$D" \
  -server /tmp/dim-2-target/debug/dim-app-server \
  -model-config ~/Code/dim-2/crates/dim-agent-aimux/examples/offline.json \
  -probe -notice-seconds 0
```

mock profile 输出（节选）：

```json
{"event":"probe.methods","methods":["session.agent.get",...,"session.collaboration.pending","session.input.pending",...]}
{"event":"probe.input_lookup.unknown","status":"unknown"}
{"event":"probe.input_lookup.accepted","eventId":"4868e486-...","sequence":5,"status":"accepted"}
{"event":"probe.input.pending","inputs":[{"inputId":"4868e486-...","sequence":5,"mode":"queue","user":{"content":[{"type":"text","text":"conncheck lookup probe"}]},"applied":false}],"next":null,"through":5}
{"event":"probe.exec.pending","error":"session.exec.pending: non_retryable: rpc error -32601 (unknown_method): Unknown method"}
{"event":"probe.question.pending","error":"session.question.pending: non_retryable: rpc error -32601 (unknown_method): Unknown method"}
{"event":"probe.collaboration.pending","next":null,"requests":[],"through":6}
{"event":"probe.attachment.put","error":"attachment.put: non_retryable: rpc error -32601 (unknown_method): Unknown method"}
```

configured profile 输出（节选）：

```json
{"event":"probe.methods","methods":["attachment.put",...,"session.exec.pending",...,"session.question.pending","session.question.respond",...]}
{"event":"probe.input_lookup.unknown","status":"unknown"}
{"event":"probe.input_lookup.accepted","eventId":"e443ca33-...","sequence":18,"status":"accepted"}
{"event":"probe.input.pending","inputs":[...,{"inputId":"e443ca33-...","sequence":18,"mode":"queue","user":{"content":[{"type":"text","text":"conncheck lookup probe"}]},"applied":false}],"next":null,"through":18}
{"event":"probe.exec.pending","next":null,"tasks":[],"through":19}
{"event":"probe.question.pending","next":null,"batches":[],"through":19}
{"event":"probe.collaboration.pending","next":null,"requests":[],"through":20}
{"event":"probe.attachment.put","attachmentId":"2fa21d1b-...","bytes":15,"path":"/private<data>/rust/dimcode.sqlite.session-files/2fa21d1b-.../conncheck.txt"}
```

证据分级：

| 形状 | 证据 |
|---|---|
| `session.methods`、`input_lookup`（accepted/unknown）、`input.pending`（空页与真实条目）、`exec.pending`/`question.pending`/`collaboration.pending`（空页）、`attachment.put` 成功回执 | 真实 daemon 实测（上两段输出） |
| 审批三类通知、四类条目的完整字段、附件错误码 | 与运行二进制同版本的源码 + 服务端自己的 wire 测试（`transport_flow/{approval,notices,titles}.rs`）；审批事件未触发（见 6.6），标题通知已在 6.5 真实触发 |
| 提问/征询通知 | 不存在：服务端通知表没有 `question.*`/`elicitation.*`；改用 `session.question.pending` + 时间线 `ask_user` 调用（实测空页 + 源码） |

### 6.5 管理面与权限证据（alpha.21）

数据源纪律：`~/Code/dim`、`../dim-electron`、`~/.dimcode-alpha` 只读；模型配置文件复制到 `/tmp` 后交给服务端；所有会话与配置落在临时数据目录。

```bash
# 临时 home + 复制到 /tmp 的离线模型配置；服务端从只读安装目录启动
D=$(mktemp -d /tmp/d1-mgmt-XXXXXX)
cp ~/Code/dim/crates/dim-agent-aimux/examples/offline.json /tmp/d1-offline.json
go run ./cmd/dimagent-conncheck -data-dir "$D" \
  -server ~/.dimcode-alpha/rust/dimcode-server -server-arg app-server \
  -model-config /tmp/d1-offline.json -mgmt -notice-seconds 1
```

探测结果（节选；完整输出已纳入本轮结果包报告）：

```text
HELLO serverVersion=0.1.0-alpha.21 protocolVersion=3 profile=configured
mgmt.plugin.list           bytes=9585  preview=[{"name":"AgentDirectory","use":"builtin","off":false,"locked":false,"candidates":[{"id":"builtin","kinds":["code"]}],"status":"running"},...]
mgmt.skills.catalog        bytes=19880 preview={"version":0,"entries":[{"id":"user:agents-sdk","source":"user",...,"enabled":true,"shadowedBy":null},...],"issues":[]}
mgmt.skills.installations  {"installations":[]}
mgmt.hooks.catalog         {"userRevision":"native:user/hooks:1","userDocument":{...},"projectsRevision":"...","disableAllHooks":false,"sources":[...],"hooks":[]}
mgmt.mcp.catalog           {"userRevision":"native:user/mcp:1",...,"servers":[],"invalid":[]}
mgmt.mcp.import.sources    {"sources":[{"id":"claude-code:user",...},{"id":"codex:user",...},{"id":"omp:user",...,"state":"absent"}]}
mgmt.schedule.list         {"jobs":[]}
mgmt.storage.report        {"databaseBytes":294912,"walBytes":758112,"reclaimableBytes":0}
mgmt.modality.capabilities bytes=4644  preview={"revision":"native:user/modality:1","categories":[{"category":"image",...}]}
mgmt.modality.task.list    {"tasks":[]}
mgmt.permission.rules      {"revision":"native:user/permission:1","commandRules":[]}
mgmt.session.permission.get    {"version":0,"policy":{"preset":"workspace-write","allowedTools":[],"disallowedTools":[]}}
mgmt.session.permission.update {"version":18,"policy":{"preset":"read-only","allowedTools":[],"disallowedTools":[]}}
notice.permission.changed  preset=read-only        # session.event permission/changed，无 permission.changed 通知
mgmt.session.rename        {"title":"conncheck probe","titleSource":"manual","revision":1}
notice.title               title=conncheck probe titleRevision=1 titleSource=manual
mgmt.shell.run             {"runId":"...","status":"completed","output":"conncheck\n","truncated":false,"durationMs":24,"exitCode":0}
mgmt.shell.list            {"runs":[]}
mgmt.shell.stop            {"stopped":false}
mgmt.session.wakeup.create {"wakeupId":"..."}
notice.wakeup              state=scheduled / cancelled
mgmt.session.wakeup.list   {"wakeups":[{"wakeupId":"...","at":"...","prompt":"conncheck wakeup","state":"scheduled"}]}
mgmt.session.wakeup.cancel {}
```

另测（同一 alpha.21，临时 home）：`plugin.enable {name:"Attachments"}` → `{"status":"running"}`，`plugin.disable` → `{}`；`schedule.save`（新建一个 every_minutes 任务）→ `{"transition":...,"job":{"id":"...","name":"probe job","schedule":{"every_minutes":60},...,"created_by":"user"}}`，随后 `schedule.list` 与 `schedule.delete` 正常。

`approval.*`、标题通知、唤醒通知也在这轮被真实事件验证：`session.rename` 触发 `session.notice {source:"SessionStore",kind:"title",data:{title,titleSource,titleRevision}}`；唤醒 create/cancel 触发 `session.notice {source:"Scheduler",kind:"wakeup",data:{branchId,wakeupId,state}}`。

### 6.6 blocked（历史项）



| 项 | 状态 | 缺什么 |
|---|---|---|
| Windows 命名管道真实连接 | blocked | 没有 Windows 主机与 Windows 版 App Server；本版只保留 `transport.Dial(pipe)` 接口并返回 `ErrUnsupportedPlatform` |
| 真实 daemon 版本不兼容 | blocked | 没有协议版本不符的服务端二进制；用 fake 服务端覆盖该分支 |
| 真实 `approval.request/cancelled/resolved` 事件 | blocked | 需要模型发出需要用户同意的工具调用；本机可用的是 mock/离线模型，不产生工具调用。形状按同版本服务端源码与 wire 测试冻结 |
| 真实非空 `exec/question` 清单 | blocked | 同上：mock/离线模型不产生工具调用；空页形状已实测 |

## 7. 依赖与上下文需求

| 依赖 | 冻结/需要 |
|---|---|
| D2 会话投影 | 消费 `Snapshot`、`OnEvent`、`OnHistory`；需要确认历史窗口与翻页锚点由 D2 发起（本版只给一页） |
| D3 会话目录 | 用 `Supervisor.Connect` 的控制连接发 catalog/rename/delete；会话流用 `Manager` |
| D4 交互与审批 | 通知原样转发；审批重放去重归 D4；`approval.respond` 走控制连接 |
| D10 宿主外壳 | 只读连接状态与 `ShutdownResult`；诊断记录由 D10 落盘 |
| V 验收设施 | fake App Server 的 NDJSON 脚本可复用本包 `protocol`；golden 帧样本可加入 |

## 8. 已知限制

1. Windows 命名管道未实现，只留接口；Windows 上没有运行证据。
2. 旧服务端没有 `session.history.latestTurn` 时，首读是从序号 0 开始的一页；证据 A 的服务端就走这条兼容路径。窗口、锚点与向前翻页归 D2。
3. `daemon.ping` 只看连接数；旧服务端无法用它看到已附加会话或运行中的轮次，因此「忙」判定弱于 `daemon.shutdown.ifIdle`。
4. 会话流重连是有界重试（默认 20 次 × 500ms），用尽即终态；不做无限重连。
5. 重连期间的通知缓冲是无上限切片，实际由服务端 attach 重放量约束；没有人为上限。
6. 非 `session.event` 的重放通知（例如 `approval.request`）不做去重，交给上层按快照整体替换。
7. `session.closed` 目前只作为通知交给上层；订阅不会自行结束，需要持有者释放。
8. detach 是尽力而为（2 秒超时），失败不阻塞释放；连接关闭后服务端会自行清理附加关系。
9. 自启子进程的 stdout/stderr 丢弃，启动失败只能看到退出状态与端点未就绪，不能从 `daemon.log` 取原因（约束禁止读写该文件）。
10. 会话流固定主线 `main`；分支流不在本版范围（归 D2/D3 的后续工作）。
11. 未知 `appError` 默认不可重试，属于失败关闭策略；能力扩展需要更新映射表。
12. 审批三类通知的**真实事件**仍未触发（见 6.6）；类型按与运行二进制同版本的源码冻结。标题通知已在 alpha.21 的 `session.rename` 上真实触发并核对。
13. 服务端没有提问/征询通知；提问只能从 `session.question.pending` 与时间线事实推导，连接领域不生成「请求/已决/超时清理」通知。
14. `session.input_lookup` 的 `branchId` 在 v3 必填，`submissionId` 必须是 UUID；写登记应保留这两个约束。
15. `attachment.put`、`session.exec.pending`、`session.question.pending` 等是贡献方法：mock profile 不存在（返回 `unknown_method`），必须先用 `session.methods` 判定能力。附件回执里的 `path` 是服务端路径，本领域不读取它。
16. `CollaborationTarget` 的 `profile` 按不透明 JSON 保留；当前服务端只有 `agent`/`member`/`temporary` 三种 kind，其它 kind 会在解析时按未知目标拒绝。
17. 本机没有 alpha.21 的 Rust 源码（`~/Code/dim` 是 alpha.8，`dim-2` 是 alpha.18）；管理面的存在性与字段以 alpha.21 实测为准，方法注册依据取 alpha.8 源码并标注版本差异。alpha.21 在 `schedule.list` 的 job 上多出 `next_window`/`last_run`，按未知字段容忍。
18. 管理面只冻结了本轮能核对的部分：`plugin.install/update`、`skills.install/preview/update`、`mcp.save/remove/move/set_enabled/import.apply`、`schedule.run_now` 的**回执**未探测成功形态，只冻结了参数；结果按 `json.RawMessage` 或调用方原样处理，不写猜的字段。
19. `runtime.*` 在 alpha.8 源码里注册（`runtime_manager.rs`），alpha.21 已不注册；`workflow.*` 在 alpha.8 与 alpha.21 都没有注册。两者都按能力缺失处理，不能按 Electron 的类型清单实现。
20. 管理方法走控制连接（`daemon.Connection.Client`）；`session.rename`、`shell.*`、`session.wakeup.*`、`attachment.put` 走会话流（`Subscription.Call` 与对应封装）。两者不要混用连接。
21. 本仓库的 `daemon.Config` 还没有默认预设路径，自启不传 `--defaults`，所以 `runtime.*` 在当前客户端不会出现；要解锁需增加该配置（见 10.1）。`runtime.list` 只报告配置与状态，不负责获取；真正 `runtime.install` 仍受网络/解释器约束。
22. `workflow.*` 在 alpha.21 随包发布物里不可用（默认预设没有 Workflow 插件）；带插件的预设能让插件进入 `pending`，但离线环境无法完成 Node/Python 运行时获取，成功注册后的回执未实测（见 10.2）。
23. `modality.realtime.*` 的方法与通知形状已冻结，但产品构建没有 realtime adapter，`available:false` 来自 adapter 维度，provider/model 配置无法解锁（见 10.3）。
24. 成员停止只验证了方法、参数与 `team_member_not_found`；端到端停止需要模型发起协作请求，未实测（见 10.4）。
25. `ui.*` 只冻结了协议形状与连接注册路径；UI 叶子的 rutis 会话没有 Go 实现，`ui.open` 的叶子路未在 Go 侧验证（见 11.1）。`ui.shutdown` 是 advisory；真正的关闭是 daemon 处置 `UiLeaf` 关闭会话，不是宿主进程被 kill。
26. `UiLeave` 结构体存在但未注册、未调度；客户端不要调用 `ui.leave`（见 11.2 末尾）。
27. 插件运行时只支持 Node/Python；没有注册 Go 运行时的公开接口，不能把普通 Go 进程伪装成 runtime（见 11.2）。
28. `internal/rutis` 只实现 UI 叶子的最小会话：Unix socket、hello/握手、newline 帧、保持/关闭；没有 TCP+token 地址、插件目录、服务 offer、入站方法、重连或 session option 限制（见 12.2）。
29. 最小叶子不声明 rutis capabilities；真实 dim 叶子声明 `reentrant-sync/plugins/value-*/node`。alpha.21 接受前者并完成 `ui.*`，但依赖 `node`/`plugins` 的叶子功能需要补实现（见 12.4）。

## 9. 下一步建议

1. D2 接入：用 `Snapshot` + `OnEvent`/`OnHistory` 做折叠，确认 `nextBefore`/`nextAfter` 的翻页接口是否够用；不够则在本包加 `Subscription.History` 的形状约定。
2. 真实「忙」场景证据：有其他客户端或已附加会话时，确认 `ifIdle` 返回 busy 且不回收。
3. Windows 命名管道：直接实现 `\\.\pipe\...` 连接，或在 `transport` 内用 `--connect-pipe` 桥接，保持接口不变。
4. 订阅级 `session.closed` 处理：由 D3 决定是否在终态通知到达时自动释放全部持有者。
5. 诊断记录（D10）：把端点、实例标识、版本、分类结果作为事件交给 D10，不在本包写文件。

## 10. 四项卡点调查

调查时间：2026-11-11。数据源均为只读：`~/Code/dim`（alpha.8 源码）、`~/Code/dim-2`（alpha.18 源码/插件）、`~/.dimcode-alpha`（0.1.0-alpha.21 安装）；脚手架与插件副本只放 `/tmp`。

### 10.1 `runtime.*`：不是移除，而是只在带默认预设时注册

**结论**：`runtime.*` 在 alpha.21 里没有消失。RuntimeManager 是内建插件，只有在 daemon 以 `--defaults <preset>` 启动（并有 home）时才注册；不带默认预设时 `session_typescript` 不存在，RuntimeManager 不装配，`runtime.list` 就是 `unknown_method`。运行时本身改成首次使用/启动时获取：`runtime.install`、`auto`、`source=download|system`，不是另一套接口。

**依据**：

- 源码：`~/Code/dim/crates/dim-app-server/src/lib.rs` 中 RuntimeManager 只在 `session_typescript.acquire` 为 `Some` 时注册；`session_typescript::SessionTypeScriptConfig::acquired` 只在 `cli.rs` 的 `--defaults` 分支调用（`acquire: Some`），`--session-rows` 分支是 `acquire: None`。
- alpha.21 实测（无 `--defaults`）：`runtime.list`、`workflow.list` 均为 `unknown_method`。
- alpha.21 实测（`--defaults` + 只含 `runtimes` 的 `/tmp` 预设）：`hello.features` 出现 `runtime.list/runtime.install/runtime.configure/runtime.watch`；`plugin.list` 有 `RuntimeManager use=builtin status=running`；`runtime.list` 返回 `[{"name":"node","source":"download","auto":false,"version":"24.21.0","status":"off"},{"name":"python",...},{"name":"uv",...}]`。
- alpha.21 实测（`--session-rows` + node/js-runtime）：`runtime.list` 仍 `unknown_method`，与 `acquire: None` 一致。

**能否在我们这一侧解锁**：能。给 `daemon.Config` 增加默认预设路径，自启参数追加 `--defaults <file>`（预设内给出 `runtimes`）。方法面即出现；真正安装运行时仍受网络/本机解释器限制。注意 `DIMCODE_DEFAULT_PLUGINS` 环境变量在我们的独立启动里未生效，`--defaults` 有效。

**需上游做什么**：接口不缺。若希望嵌入宿主无需自带预设就能看到运行时，上游需要在文档/发布物里明确默认预设的传递方式（或让独立 `dim-app-server` 也认 `DIMCODE_DEFAULT_PLUGINS`）。

### 10.2 `workflow.*`：默认插件提供，当前发布物没有带

**结论**：`workflow.*` 由 **Workflow 默认插件**（JS）注册，不是内建方法。只有启动时带一个包含该插件的 default-plugins 预设（并具备插件运行时和 `settings`/`workflowFiles` host）才会出现；alpha.21 随包发布的 `default-plugins.json` 没有 workflow，所以默认不可用。

**依据**：

- 安装物：`~/.nvm/.../dimcode-alpha/node_modules/dimcode-darwin-arm64/default-plugins.json` 只有 `plugin-authoring` 与可选 `light-ocr`，没有 workflow；`plugins/` 目录同样没有 workflow。
- 源码/插件：`~/Code/dim-2/staging/dev-plugins/plugins/workflow/index.ts` 通过 `managementMethods.register` 注册 `workflow.list/status/start/stop/save`；`dim-plugin.json` 声明需要 `settings`、`workflowFiles`、`pluginFiles` 等 host；同目录 `default-plugins.json` 把 workflow 列为默认插件。
- alpha.21 实测（`--defaults` 不含 workflow）：`workflow.list/status` 均 `unknown_method`。
- alpha.21 实测（`--defaults` 含 workflow 副本）：`plugin.list` 出现 `workflow use=default status=pending`；在 pending 期间 `workflow.list` 仍 `unknown_method`。
- alpha.21 实测（`--session-rows` 直接加载 workflow/index.ts）：插件报 `still waiting after 10s for settings, workflowFiles: it starts once that is provided`，说明它需要默认预设 profile 的进程 host，而不是 session row。
- 离线环境无法完成 Node/Python 运行时获取，workflow 一直 pending，因此“成功注册后的回执”未实测。

**能否在我们这一侧解锁**：不建议。需要把 workflow 插件与默认预设一起分发，并保证插件运行时（Node/Python 获取）可用；这等于由客户端携带第三方插件，超出连接领域边界。当前应按能力缺失处理。

**需上游做什么**：把 workflow 插件纳入随包默认预设，或提供官方安装路径/发行包说明。

### 10.3 模态实时：adapter 层没有适配器，配置改不动

**结论**：realtime 不可用来自 **adapter 维度**，不是 provider/model 配置。alpha.21 的 `modality.capabilities` 对 `realtime` 给出三条 `adapter` 原因（voice/transcribe/avatar 无可用 adapter），产品构建只带 fixture image adapter，没有 realtime adapter；仅配置 provider/model 无法让它变可用。

**依据**：

- alpha.21 实测 `modality.capabilities`：`realtime available=false reasons=[{dimension:"adapter",detail:"No available adapter serves realtime.voice"},{...transcribe},{...avatar}]`，三个 operation（voice/transcribe/avatar）均 `available:false`；`image` 是 adapter/configuration，`video/speech/music/audio/asset/motion` 是 build，`ocr` 可用。
- 设计：`~/Code/dim/docs/design/plugins/Modality.md` §1/§3.2/附录 A 说明 adapter 操作由 `modalityAdapter` 服务提供；产品构建只有 fixture adapter（image slice），其他类别报 build；`options`/`available` 由 owner 在读取时从“可用的 adapter + Provider 可选择的模型”计算，adapter 的 `operations` 决定它服务哪些模型。
- 安装物 `~/.dimcode-alpha` 的 `runtimes`/`plugins` 无 realtime adapter 插件（只有 light-ocr 供 OCR）。
- 二进制含 `RealtimeDeclaration`、`modality.realtime.*` 与 adapter 原因字符串，证明通路存在，缺的是适配器。

**能否在我们这一侧解锁**：不能靠 provider/model 配置。需要安装/随包一个服务 realtime.voice/transcribe/avatar 的 adapter 插件；当前没有可安装来源。

**需上游做什么**：提供 realtime adapter（或发布 fixture/官方 adapter 插件并说明安装方式）。

### 10.4 成员停止：`session.collaboration.cancel` 存在

**结论**：服务端提供成员停止：`session.collaboration.cancel {sessionId, branchId?, memberSessionId}` → `{status:"cancelled"|"idle"}`。`idle` 是合法结果（没有可停的运行）。成员会话标识来自协作事实，不来自 `agent.list`/`session.agent.get`。

**依据**：

- alpha.21 `session.methods` 含 `session.collaboration.cancel`；实测带伪造 `memberSessionId` 调用返回 `team_member_not_found`，错误文本提示 “use list for agentId + sessionId”，证明参数面与归属校验。
- 源码：`~/Code/dim-2/crates/dim-app-server/src/collaboration.rs` 的 `CancelParams { session_id, branch_id, member_session_id }` 与 `cancel_member` 返回 `{"status":"cancelled"|"idle"}`；客户端调用不写 `collaboration/cancel` op，效果由成员的 `stop/interrupt`、`turn/end` 与 leader 的 `collaboration/result`/report 记录。
- 获取成员会话标识：`collaboration/delivered` 事实的 `delivery.session_id`（status=ok）、`session.collaboration.pending` 请求的 `delivery.session_id`、`collaboration/result` 的 `session_id`、`collaboration/message|cancel|close` 的 peer 数据 `session_id`。`agent.list` 只有 `sessions` 计数；`session.agent.get` 返回分支采纳的 `agentId`，都不是成员会话标识。
- 类型已补：`SessionCollaborationCancelParams`、`SessionCollaborationCancelResult`、`CollaborationDeliveredData`（`ParseCollaborationDeliveredData`）。

**能否在我们这一侧解锁**：能。读 `session.collaboration.pending`（或折叠 `collaboration/delivered`）拿 `memberSessionId`，调用 `Session.Call` 发 `session.collaboration.cancel`；`idle` 视为已停。端到端“成员运行被停止”未实测，因为需要模型发起协作请求；方法、参数、错误码已真实验证。

**需上游做什么**：接口不缺。若希望客户端不依赖事件推导，需要上游在 `session.agent.get`/`agent.list` 一类读取里直接给出成员会话标识（当前没有）。

### 10.5 需上游开的接口/能力

按 #1394/#1395 的格式（Problem / Needed / Owner / Impact / Acceptance）：

1. **Workflow 默认插件未随发布物提供**（能力缺失）
   - Problem：`workflow.*` 只由 Workflow 默认插件注册，alpha.21 随包 `default-plugins.json` 不含该插件；无 `--defaults` 时客户端无法提供工作流。
   - Needed：把 Workflow 插件纳入随包默认预设，或给出官方安装/发行包路径。
   - Owner：默认插件发布物 / PluginManager。
   - Impact：客户端只能把 `workflow.*` 报为能力缺失。
   - Acceptance：用发布包启动 `dim-app-server`（不额外带插件）时 `hello.features`/`session.methods` 出现 `workflow.*`，且 `workflow.list` 返回定义列表。
2. **产品构建没有 realtime adapter**（能力缺失）
   - Problem：`modality.capabilities` 的 realtime 三项因 “No available adapter serves realtime.*” 不可用，provider/model 配置无法改变；构建里没有 realtime adapter 插件。
   - Needed：提供 realtime.voice/transcribe/avatar 的 adapter（内建或可安装插件），并说明获取方式。
   - Owner：Modality / adapter 发布物。
   - Impact：实时语音/转录/头像功能不可用，客户端只能显示 adapter 原因。
   - Acceptance：安装/随包 adapter 后 `modality.capabilities` 的 realtime 至少一项 `available:true`，`modality.realtime.open` 返回 `connectionId` 并能收到 `modality.realtime.event`。

## 11. 宿主侧能力调查

调查时间：2026-11-11。源码只读：`~/Code/dim`（alpha.8）、`~/Code/dim-2`（alpha.18）、`~/.dimcode-alpha`（alpha.21 安装物）；没有写回任何数据源。

### 11.1 宿主协作协议（`ui.*`）

**结论**：`ui.*` 是 daemon 的**界面登记表**，不是 daemon 调用宿主的业务 RPC。宿主有两条接入路：普通连接注册（一条已 attach 到会话的 protocol v3 连接调用 `ui.attach`，无 `uiId/token`），或 UI 叶子（宿主先绑定私有地址，已经 attach 的本机连接调用 `ui.open`，daemon 反向拨号并做一次 rutis 会话握手，随后宿主的 TUI 用 token 调 `ui.attach`）。日常业务仍走 protocol v3；`ui.*` 只负责“谁在展示哪个会话、能否被替换/关闭”。

**规则与形状**（源码：`crates/dim-app-server/src/ui_host.rs`、`ui_leaf.rs`、`protocol.rs`；文档 `docs/design/plugins/Tui.md §3.6`）：

| 方法（本机 protocol v3） | 参数 | 回执 |
|---|---|---|
| `ui.open` | `{sessionId, kind, peer, address, replace?, expectedGeneration?}`；调用者必须已 attach 到该会话；`kind` 为 `tui`，`peer` 是 `ui-` + 小写字母/数字/`-`，成为 `uiId`；`address` 为 `unix:<path>` 或 `tcp:<host>:<port>#<token>` | `{uiId, generation, token}`；失败只可能是链路/握手（`ui_leaf_failed`），不留注册；20 秒内未 attach 按 `ui.close` 关闭 |
| `ui.attach` | `{sessionId, kind, uiId?, token?}`；普通界面不带后两者；叶子带 `ui.open` 回执的两者 | `{uiId, generation}`；`session_not_attached`、`ui_token`、`ui_in_use` 等错误 |
| `ui.status` | `{}` | `{instances:[{uiId, sessionId, kind, sinceMs, generation, state}], webConnections}` |
| `ui.close` | `{uiId, reason?}` | `{}`；移除注册并向该界面连接发 advisory `ui.shutdown`；叶子同时处置 `UiLeaf`，关闭其会话 |
| `ui.shutdown` | `{uiId, reason?}` | `{}`；只发 advisory 通知；连接在 detach/reattach 之间时为 `ui_detached` |
| `ui.replace` | `{uiId, sessionId, kind, generation?}` | `{uiId, generation}`；用于连接注册的替换，旧界面随后关闭 |

服务端 → 宿主只有一条通知：`ui.shutdown {uiId, reason}`（advisory，不强制终止进程）；叶子的生命周期事实是 **rutis 会话结束**（没有心跳）。daemon 退出时处置 `UiHostPlugin`，等待所有 `UiLeaf` 关闭。

**与 session/branch 的关系**：注册按 `sessionId` 绑定，没有 branch 字段；一个会话可有多个 viewer/连接注册，互不排斥。叶子活到其会话结束，不随调用 `ui.open` 的连接存在；该连接之后没有角色。`ui.open` 的 `replace`/`expectedGeneration` 只允许替换同一会话的旧界面；替换在新叶子 attach 后关闭旧界面，非原子、无回滚。

**失败与重连**：`ui.open` 在 dial/握手失败时 `ui_leaf_failed` 且不留注册；`ui_token`/`ui_in_use`/`ui_unknown`/`session_not_attached`/`ui_generation`/`ui_detached` 是其余稳定错误码。叶子 20 秒未 attach 会被关闭；叶子进程退出或会话结束会移除注册。detach（如 `/new`、resume）不改叶子状态，重新 `ui.attach` 跟到新会话；连接注册在重连后重新 `ui.attach`（同一连接+会话幂等返回同一 `uiId`）。

**一个原生 UI 宿主必须做什么**：

- 选择角色。普通原生窗口建议用**连接注册**：在已 attach 的会话连接上调 `ui.attach {sessionId, kind:"gui"}`（不带 `uiId/token`），保存返回的 `uiId`；窗口关闭调 `ui.close`；收到 `ui.shutdown` 只当作“界面被关闭”的提示并自行收尾；断线重连后重新 attach。
- 若要成为**UI 叶子**（进程/终端与会话同生命周期，可被 daemon 替换/关闭）：宿主必须自己绑定私有地址，提供 rutis 会话的 `LocalId ui-<id>`，接受一次连接并按 rutis 会话协议（0.19.3 为 session protocol 4）与 daemon 握手，会话结束后处置自己的 UI；同时保留一条 protocol v3 连接来 `ui.attach {uiId, token}`。谁发起：**宿主发起 `ui.open` 与 `ui.attach`**；daemon 只反向拨号、登记、关闭。`ui.*` 只对本机连接开放，`hello` 用 `ui.leaf.2` 标记叶子语义。
- 注册身份：普通界面由服务端生成 `uiId`；叶子用宿主提供的 `peer`（`ui-...`）作为 `uiId`，`token` 只由 `ui.open` 发给宿主。`kind` 必须是可打印名字；叶子只接受 `tui`。

**我们能做什么**：连接注册这条路现在就能做：已补 `UIOpenParams/UIOpened/UIAttachParams/UIAttached/UICloseParams/UIReplaceParams/UIShutdownParams/UIShutdownNotice/UIStatusResult/UIInstance` 与解码测试。叶子路需要 rutis 会话协议，Go 侧没有现成 SDK（见 11.2），不建议在连接领域先做。

**需要上游做什么**：`ui.*` 接口不缺。若希望非 Rust 宿主实现叶子，需要 rutis 的 Go 会话协议/SDK 或一份可实现的 wire 规范（当前仓库只写“rutis-bridge 的 framed 通道 + handshake”，没有帧格式）。

### 11.2 插件运行时注册与 Go leaf runtime

**结论**：插件运行时是宿主 loader 的**连接行**（connection row），不是 RPC 注册面。当前只支持 Node 和 Python：`session-node`、`py` 两个连接行，由 `--node/--js-runtime/--python`（开发/测试）或 RuntimeManager 的默认预设 `runtimes` 提供启动命令；插件行按 `<row>:<module>` 由 `SessionResolver::npm` 解析。**没有注册第三种（Go）运行时的公开接口**；UI 叶子虽然也是 rutis 会话，但被刻意设为“无插件目录”，daemon 不能在叶子里跑插件，因此它不是插件运行时。

**源码依据**：

- `docs/design/rutis-usage.md` A.4：每个本地运行时是 `session-node`/`py` 连接行，RuntimeConnection 注入 `nodeRuntime`/`pythonRuntime` 启动命令，`Launcher` 启动进程、等 greeting、挂 `Session::new(channel)`；`SessionResolver::npm(catalog, "session-node")` 解析 `py:<module>` 与其它外部名字。
- `docs/design/rutis-0.19-upgrade.md` M4：运行时是连接行；M8：UI 叶子是插件、其寿命是会话；文中明确 “Not in scope: Go, Swift and Kotlin runtimes, dylib plugins, remote runtimes and node connection rows”。
- `crates/dim-app-server/src/session_typescript.rs`：`SessionTypeScriptConfig` 只有 node/py；`Parts::runtimes()` 只返回这两个 `ForeignRuntime`；启动命令 `node --import tsx ...` / `python -m rutis`；`--session-rows` 只提供行，不提供新的 runtime 类型。
- `crates/dim-app-server/src/runtime_manager.rs`：只计划 Node/Python/uv，`RuntimeEntry` 是这三者；`runtime.*` 的注册仍需默认预设（见 10.1）。
- `ui_leaf.rs`：`UiLeaf` 用 `ctx.isolate(catalog_key(), ...)` 隔离目录，且 LeafNode “offers no plugin catalog”，所以叶子不是插件运行时。

**一个 Go leaf runtime 需要实现什么**：

- 若指 **UI 叶子**（可行但非插件运行时）：实现 rutis 0.19.3 的会话协议（session protocol 4）：私有地址（Unix socket 或 loopback TCP + 首行 token）、`LocalId ui-<id>`、accept 一次、`pipe::framed` 通道、与 `dim-daemon` 握手并校验对端 id；随后用 protocol v3 做业务与 `ui.attach`。前置条件：一份 Go 的 rutis 会话实现（本仓库未找到；上游 docs 只描述 framed/handshake，没有帧格式）。
- 若指 **插件运行时**（让 Go 跑插件行）：还缺服务端接口。需要 host loader 能新增一个连接行：启动命令（`<channel> --id <name> <project>` 形态）、greeting/会话协议、`SessionResolver` 对该行前缀（例如 `go:<module>`）的解析、row 的 `provides`/`instanceNames` 目录声明、`RuntimeStatus`（starting/ready/down）与生命周期（会话结束、卸载杀进程、失败后行回到等待）。这些在 alpha.8/alpha.21 都没有公开注册方式；`--session-rows`/`--defaults` 只接受既有 Node/Python 运行时。
- 能力声明：插件/部件通过 `provides` 与 `instanceNames` 声明，宿主目录在运行时启动前固定；跨语言服务按名字导出（`managementMethods`、`partRegistry`、`settings` 等）。Go 运行时若存在，需实现同样的服务调用/注册与事件总线协议，而不是自定义窄接口。

**我们能做什么**：

- 不注册 Go 插件运行时；在可预见的范围内继续用 Node/Python 插件（由 daemon 的 RuntimeManager/默认预设提供）。
- 原生 UI 用 `ui.attach` 连接注册，不需要 rutis；若将来要做叶子，先补 Go 的 rutis 会话实现（这是一个独立前置项，不应塞进连接领域）。
- 不能靠“宿主自己起一个 Go 插件进程并说它是 runtime”绕过：daemon 的 loader 不会解析它，`plugin.list`/`plugin.install` 也不认。

**需要上游做什么**（按 #1394/#1395 格式）：

1. **缺少通用运行时注册接口**
   - Problem：插件运行时只支持 Node/Python；`SessionTypeScriptConfig`、RuntimeManager、`SessionResolver` 都硬编码这两个，没有公开方式注册第三种连接行。
   - Needed：可配置的运行时连接行（启动命令、行前缀/模块解析、`provides`/`instanceNames` 目录声明、`RuntimeStatus` 与生命周期），或明确说明“只支持 Node/Python，第三方运行时不在范围”。
   - Owner：RuntimeManager / rutis 集成。
   - Impact：非 Rust、非 Node/Python 的宿主（含 Go）无法承载插件。
   - Acceptance：配置一个测试运行时后，`runtime.list` 出现该行，`plugin.list` 的行变为 ready，`<row>:<module>` 的方法/工具在会话中可用；运行时退出后行回到等待且 `runtime.list` 报 down+reason。
2. **缺少 Go 的 rutis 会话协议实现/SDK**
   - Problem：UI 叶子与非 Rust 运行时都需要 rutis 0.19.3 的 framed 通道与 session protocol 4，仓库只有 Rust 实现与文档级描述，没有帧格式或 Go SDK。
   - Needed：Go SDK（或完整 wire 规范 + 一致性测试）。
   - Owner：rutis。
   - Impact：mygo 只能做普通 `ui.attach` 连接注册，不能做叶子，也不能承载插件。
   - Acceptance：Go 进程与 daemon 完成握手、挂载会话、注册/调用一个命名服务并通过 rutis 自带的一致性用例。

**未找到**：rutis framed 通道的字节级帧格式、Go SDK/Go runtime 实现、任何 `runtime.register`/`runtime.create` 一类的公开方法；`protocol.rs` 里有 `UiLeave` 结构体，但不在 `UI_HOST_METHODS` 或 builtin 方法表里，也没有被 `connection.rs` 调度，未作为接口使用。

## 12. UI 叶子协议与 Go 最小实现

调查时间：2026-11-11。源码只读：`~/Code/rutis`（含 `crates/rutis-bridge`、`rutis-types`、`go/rutis`）、`~/Code/dim`（dim 的叶子实现 `crates/dim-cli/src/leaf.rs`、`crates/dim-app-server/src/ui_leaf.rs`）、`~/.dimcode-alpha`（alpha.21 二进制）。夹具与 socket 只在 `/tmp` 或临时 home。

### 12.1 session protocol 4 要点

| 项 | 内容（源码依据） |
|---|---|
| 帧 | 一条消息一行：JSON 后接 `\n`；发送前禁止消息内出现原始换行（JSON 字符串里的换行会被转义）；接收要求 UTF-8；单帧上限 16 MiB（不含换行），超限即结束通道。来源：`rutis-bridge/src/pipe/lines.rs`（`MAX_MESSAGE`、`refuses`）、`go/rutis/internal/peer/channel.go`。 |
| 握手 | 双方各发一帧 `hello`：`{op:"hello", version:4, endpoint:"<id>", implementation?:{name,version}, capabilities?:[...]}`；`version` 必须为 4；`endpoint` 匹配 `^[a-z0-9-]+$` 且不能等于本端 id；可声明期望对端 id。daemon 端在 10 秒内等不到 greeting 就关闭。来源：`session/plugin/greet.rs`（`Greet::open`、`ENDPOINT_PROTOCOL=4`、`HANDSHAKE=10s`）、`session/plugin/mod.rs`（`Session::handshake`）、`go/rutis/internal/peer/peer.go`（`Start`、`greet`、`versionMismatch`）。 |
| LocalId | 本端 id 由 `rutis_bridge::LocalId` 提供，作为 hello 的 `endpoint` 和调用 id 前缀。UI 叶子用 `ui-<id>`（dim 用 `ui-<uuid-simple>`；peer 只允许小写字母、数字、`-`），daemon 用 `dim-daemon`。daemon 的 `UiLeaf` 用 `expected` 校验叶子 id；dim 的叶子反校验对端为 `dim-daemon`。来源：`crates/dim-cli/src/leaf.rs`、`ui_leaf.rs`。 |
| 管道/地址 | Unix：native home（`<home>/rust`，0700）里的 `ui-<8 hex>` 私有 socket，地址写作 `unix:<path>`；跨平台走 loopback TCP `tcp:<host>:<port>#<token>`，连接后第一行先发一次性 token，再开始握手。来源：`ui_leaf.rs` 的 `dial`、`leaf.rs` 的 `Accept`、`Tui.md §3.6`。 |
| 失败 | 版本/格式不符、非法 endpoint、期望 id 不符会在握手阶段结束会话；daemon 的 `ui.open` 以 `ui_leaf_failed` 返回且不留注册；`ui.attach` 另见 `ui_token`/`ui_in_use`/`ui_unknown`。来源：`greet.rs`、`ui_host.rs`。 |
| 关闭 | 通道结束（EOF/关闭 socket）即会话结束：daemon 处置 `UiLeaf` 并移除界面注册；`ui.close`/替换/daemon 退出先发 advisory `ui.shutdown`，再处置 `UiLeaf` 关闭通道。没有心跳，会话结束就是生命周期事实。来源：`ui_leaf.rs`、`ui_host.rs`、`Tui.md §3.6`。 |

### 12.2 Go 最小实现（`internal/rutis`，只用标准库）

实现范围：

- `Listen(dir)`：在私有目录绑定 `ui-<8 hex>` 的 Unix socket，返回 `peer`（`ui-...`）与 `address`（`unix:<path>`）。
- `Serve(ctx, expectedPeer)`：accept 一次；发送本端 hello（`version=4`、`endpoint=peer`、`implementation.name=mygo-uileaf`、不声明 capabilities）；10 秒内读取对端 hello，校验 `op=hello`、`version=4`、`endpoint=dim-daemon`；之后保持通道，逐帧读取。
- 入站 `invoke/call/get`：以 `throw {name:"BindingError", message:"the UI leaf serves no methods"}` 拒绝；`offer`/通知类帧忽略。叶子没有插件目录、服务或 row，所以不实现 rutis 插件宿主。
- `Close`/通道 EOF：删除 socket、关闭 `Done`；`Serve` 返回 `ErrClosed`。
- 帧编解码：newline、16 MiB 上限、UTF-8 校验、超限帧错误；与 `pipe/lines.rs` 一致。
- 仅实现了 Unix socket 与纯协议 4；没有实现 TCP+token 地址、plugin catalog/services、重连或 session option 限制。

### 12.3 真机结果（alpha.21）

```bash
DIMCODE_APP_SERVER_BIN=~/.dimcode-alpha/rust/dimcode-server \
DIMCODE_APP_SERVER_ARGS=app-server \
go test -count=1 -v ./internal/rutis -run TestRealDaemonUIFLeaf
```

结果（只记标识、状态与帧长度，不打印会话内容）：

```text
leaf peer=ui-9d6ce3ce954767c8
greeting={Endpoint:dim-daemon Implementation:map[name:rutis-bridge version:0.19.3]
  Capabilities:[reentrant-sync plugins value-instant ... value-uuid node]}
hello_bytes=113 far_hello_bytes=302 token_bytes=43 generation=1
ui.status instances=1 state=online generation=1
leaf session ended after ui.close: rutis: session closed
PASS
```

步骤全部通过：宿主绑定 socket → 调 `ui.open {sessionId, kind:"tui", peer, address}` → daemon 反向拨号并握手（对端 `dim-daemon`，protocol 4）→ 宿主拿到 `{uiId, generation, token}` → `ui.attach {uiId, token}` → `ui.status` 看到 `online` → `ui.close` → 注册移除且叶子 session 结束。调用链用现有 `daemon`/`session` 客户端与临时 home、mock 模型，结束后 `Shutdown` 回收自有 daemon。

### 12.4 结论与边界

- **能做**：纯 Go、标准库即可完成 UI 叶子所需的最小 rutis 会话；不需要 rutis 动态库、Node 或 Python。alpha.21 接受不声明 capabilities、无插件目录的最小 hello。
- **不能做**：这个包不是插件运行时，也不承载插件行；需要插件目录/服务/row 的完整叶子（例如在叶子里跑 TUI 或注册 Session part）还要实现 `node`/`plugins` 能力、offer/服务调用与 row 协议，当前未实现。
- **未做/限制**：仅 Unix socket（TCP `tcp:...#token` 未实现）；固定期望 `dim-daemon`；没有重连（会话结束即退出）；入站调用一律拒绝；帧 16 MiB；不声明 capabilities（真实 dim 叶子声明 `reentrant-sync/plugins/value-*/node`）。

**需要上游做什么**：当前无阻塞；若要在 Go 叶子中运行插件/注册 Session part，需要完整的 rutis Go SDK 或等价规范（`~/Code/rutis/go/rutis` 已存在，但版本是 0.21+ 且当前协议仍为 4；接入前要确认与宿主 0.19.3 的兼容性）。本轮已验证最小握手无需该 SDK。

## 13. 变更记录

| 日期 | 变更 | 原因 | 影响 |
|---|---|---|---|
| 2026-11-10 | 第一版：帧、传输、daemon 生命周期、会话流、错误分类与真实 daemon 证据 | 连接领域第一版交付 | D2–D6 可以接入订阅与历史 |
| 2026-11-10 | 补齐 v3 wire：审批三类通知、`session.notice` 标题、`session.input_lookup`、四类未完成事项、`attachment.put`、`session.methods`；会话把通知映射为类型化事件并提供会话方法路由 | 集成层被六处未冻结形状挡住 | 审批、提问、标题、附件与写登记可以按已冻结形状实现；提问没有通知的结论已明确 |

| 2026-11-11 | 管理面方法调查与冻结：plugin/skills/hooks/mcp/schedule/storage/modality/session.rename/shell/wakeup 的类型与解码测试；`permission.changed` 结论为「不存在通知，等同时间线 `permission/changed`」；列出 toolServer/workflow/scheduler/runtime/顶层 wakeup 的不存在依据 | 集成层需要管理面与权限档位形状 | D5 可按已冻结形状实现；未探测的变更回执与运行时方法明确列为未冻结 |

| 2026-11-11 | 四项卡点调查：runtime 注册条件（`--defaults`/`acquire`）、workflow 默认插件缺失、模态实时 adapter 缺失、成员停止方法与成员标识来源 | 集成层需要判定能否在本侧解锁 | runtime 可由客户端补默认预设解锁；workflow 与 realtime 需上游；成员停止可直接接入 |
| 2026-11-11 | 补第四轮调查：runtime 注册条件与客户端解锁方式、workflow 默认插件缺失、模态实时 adapter 缺失、成员停止接口与成员会话标识来源；附两条上游 issue 草案 | 四件卡点需要可否解锁的判定 | runtime/成员停止可接入；workflow/realtime 需上游 |
| 2026-11-11 | 宿主侧调查：`ui.*` 语义/参数/回执/方向/失败与重连；插件运行时只支持 Node/Python，Go leaf runtime 需要上游通用运行时行与 Go rutis 会话实现；补 UI 宿主 wire 类型 | 原生宿主接入需要冻结 ui.* 形状并判定 Go 运行时可不可行 | 普通 UI 连接注册可接入；UI 叶子和 Go 插件运行时需上游 |
| 2026-11-11 | 读 rutis 源码实现最小 Go UI 叶子会话（`internal/rutis`，纯标准库）；alpha.21 真机跑通 ui.open/握手/ui.attach/ui.status/ui.close；记录协议 4 帧、握手、LocalId、地址与关闭语义 | 验证 Go leaf 只缺协议实现而非不可行 | 原生 UI 可用 Go 叶子接入；插件运行时仍未支持 |
