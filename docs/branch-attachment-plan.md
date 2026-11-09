# Branch switching / cut point / side chat 与附件：公开合同核对与接线计划

Status: active（只读调研；本文不改源码）
Last updated: 2026-11-09
Owner: protocol / ui
Scope: 只回答两件事——(1) branch switching / cut point / side chat 有没有公开请求；(2) 附件有没有公开上传方法；并给出 MyGo 现状、可安全接线的最小范围、必须保持 WIP 的入口。

证据根（下文用代号引用，行号以本次读取为准）：

- **A** = `/Users/eric8810/Code/dim-electron`（Desktop public client 与它的公开文档）
- **B** = `/Users/eric8810/Code/dimagent-mygo`（本仓库）
- **C** = `/Users/eric8810/Code/dim-2`（Rust App Server 的公开设计合同）

证据等级按 [acceptance.md](acceptance.md)：本文全部结论目前是 **L0 source-read**；没有任何 L1–L5 证据。

一条硬规则：wire 类型以 A 的 [`wire.ts`](/Users/eric8810/Code/dim-electron/packages/desktop/packages/app-server-protocol/src/wire.ts) 和 C 的公开设计合同为准。**`wire.ts` 里没有 DTO 的方法**（`session.branch_create`、`session.branch_end`、`attachment.put`）只有「public client 调用点 + 上游设计文档」证据，MyGo 必须先补 DTO/校验/fixture（S0 门禁），不能凭记忆直接发请求。

## 0. 一句话结论

1. **有公开请求，而且不是"只能开新 session"**：同一 Session 内的 side chat 就是 **host branch**。创建用 `session.branch_create {sessionId, parentBranchId, at, prompt}`，结束用 `session.branch_end {sessionId, branchId}`，目录读 `session.branches {sessionId}`；"切换 branch" 没有独立 RPC，而是用带 `branchId` 的 `session.history` / `session.keys` 读该 branch 的历史并折叠；"cut point" 也没有 RPC，是客户端算出的父分支 sequence，由 `session.branch_create` 校验。
2. **附件有上传方法，但只收文本**：`attachment.put {sessionId, name?, text}` → `{attachmentId, path, bytes}`，文本非空且 ≤10 MiB UTF-8，单帧 ≤16 MiB；图片**没有**上传方法，走 `session.submit` 的 `user.content` inline base64 image block。

## 1. 问题 1：branch switching / cut point / side chat

### 1.1 已确认存在的公开请求

| 用途 | 方法 | 参数（已证） | 响应（已证） | 证据 |
|---|---|---|---|---|
| 订阅 Session（不含 branch） | `session.attach` | `{sessionId}` **只有 sessionId** | `{sequence?, running?, model?, modelPreferences?}` | A `wire.ts:151-153`、`172-181`；A `session-reader.ts:776` |
| 读某 branch 的历史（= 切换的数据来源） | `session.history` | `{sessionId, branchId?, after?, limit?, through?, latestTurn?, before?}` | `{events, through, nextAfter, done, nextBefore?, turnStart?}` | A `wire.ts:244-265`；A `session-reader.ts:941-949` |
| 读某 branch 的 keyed block | `session.keys` | `{sessionId, branchId, through}`（feature `session.keys`） | `{through, entries:[{key, sequence, inputId, text}]}` | A `wire.ts:267-278`；A `session-reader.ts:1035-1047` |
| 读 branch 目录 | `session.branches` | `{sessionId}`（feature `session.branches`） | `{branches:[{branchId, parentBranchId?, branchPoint?, createEvent?, source?, state, head, endEvent?}]}` | A `wire.ts:280-304`；A `session-reader.ts:1067-1084`；C `design/timeline-branches.md:512-520` |
| 创建 host line（side chat） | `session.branch_create` | `{sessionId, parentBranchId, at, prompt}`（`at` 是父分支 sequence） | public client 解码 `{branch?:{branchId?, parentBranchId?, branchPoint?, createEvent?}}`；**缺 `branchId` 视为错误** | C `design/plugins/Hooks.md:27`、`design/web-workspace.md:200-201`；A `service.ts:4968-4984` |
| 结束某 branch | `session.branch_end` | `{sessionId, branchId}` | 响应体被客户端忽略；已结束再结束报 `branch_ended`；不支持时 `unknown_method` | A `service.ts:5049-5059`；C `design/timeline-branches.md:301`、`323`、`654` |
| 向 branch 发消息 | `session.submit` | `{sessionId, submissionId, branchId, text?, user?, mode?}`，`mode: queue\|steer\|append` | 至少 `{event}` | A `wire.ts` 无 DTO（client 调用点 `service.ts:4915-4922`）；C `design/web-workspace.md:200`、`design/plugins/Hooks.md:27` |
| 停止 branch 的 turn | `session.stop` | `{sessionId, branchId, reason?}` | 客户端不解析 | A `service.ts:5040-5047` |
| branch 的审批 | `approval.respond` | `{sessionId, branchId, requestId, decision}` | — | A `interaction-service.ts:497`；C `design/timeline-branches.md:292-297` |
| branch 事件/状态 | `session.event` / `session.state` / `approval.*` / `session.model.progress` | 通知里带 `branchId` | — | A `wire.ts:306-317` |

能力发现：

- `session.branches` 由 `hello.features` 广告（A `session-reader.ts:1071`；C `design/timeline-branches.md:512-520`、`557`）。
- `session.branch_end` 作为 v3 host 入口出现在 `hello.features`（C `design/timeline-branches.md:557`、`658`）。`session.branch_create` 没有文档化的 feature 名（C 只说明它已存在，`design/timeline-branches.md:48`），客户端按调用结果判定：`unknown_method` → 入口 visible-disabled WIP。
- `session.methods` 列出该 Session 已注册 namespace 声明的方法（C `design/contracts/session-boundary.md:247-248`），可用于发现插件方法；文档只说它按 namespace 列出并额外加入一个 built-in 名，没有说明包含 `branch_create`，所以**不能**拿它当 branch_create 的 gate。

### 1.2 三个概念各自对应什么（不是三个 RPC）

| 概念 | 真实机制 | 证据 |
|---|---|---|
| **Branch switching** | 不重开连接、不换 attach。在已 attach 的同一 stream 上用该 `branchId` 读 `session.history`（ended line 用固定 `head` 作 `through`），再读 `session.keys`，折叠成该 branch 的独立投影；live `session.event` 按 `branchId` 分流。 | A `session-reader.ts:925-1008`、`941-949`、`953-983`、`1067-1084`；A `docs/architecture/timeline.md:17-18`、`91` |
| **Cut point** | 客户端从 history 里算父分支 sequence：目标 user input 的 `sequence - 1`，且该 input 必须是已完成 turn 的第一个 input；默认切点是最近一次 `turn/end` 或 `context/configured`。`session.branch_create` 是最终校验者。 | A `session-reader.ts:862-896`、`903-923`；C `design/web-workspace.md:201` |
| **Side chat** | UI 概念，协议上是同一 Session 的 **host branch**：create → 读该 line 历史 → `submit {branchId}`（运行中可 `mode:"steer"`）→ `stop {branchId}` → `branch_end`；审批带 `branchId`。branch 事件走同一个 `session.event`；Desktop 的 `session.branch.event` 只是 **Electron IPC 事件名**，不是 wire 方法。 | A `docs/architecture/upstream-readiness.md:137`、`143`；A `bootstrap.ts:867-872`、`944-951`；C `design/web-workspace.md:184`、`200-202` |

为什么不是"开新 session"：branch 是**同一个 agent、同一个 Session、同一份身份/工具/模型/权限/工作区**的另一条线；fork 才是复制出的新 Session，collaboration member 是另一个 agent 的 Session（C `design/timeline-branches.md:30-40`）。新 session 会多出 catalog 行、独立历史、独立 branch 树，语义不同。

### 1.3 明确不存在 / 不能假设

| 不存在的请求 | 结论与替代 | 证据 |
|---|---|---|
| `session.attach {branchId}` | 不存在。attach 只按 Session；分支靠 `branchId` 参数读。 | A `wire.ts:151-153`；A `session-reader.ts:776` |
| branch **switch/select** RPC | 不存在。切换是客户端读该 branch 的 history/keys。 | A `session-reader.ts:925-1008` |
| branch **cut point** RPC | 不存在。切点是客户端算的 sequence。 | A `session-reader.ts:862-923`；C `design/web-workspace.md:201` |
| `side_chat.*` 方法 | 不存在。side chat = `branch_create` + 带 `branchId` 的既有方法。 | 同上；A `docs/architecture/upstream-readiness.md:137` |
| branch 级 detach / resync / attach | 不存在；detach 与 resync 是 Session 级。 | B `docs/protocol-contract.md:181-183`、`305-309`；A `wire.ts:151-153` |
| `session.branch.get` | 上游设计文档存在（返回 `purpose`、`cutMessages`、`end`），但 A 的 `wire.ts` 无 DTO，A 的 client 也没有任何调用点（本次 grep 为 0）。不要作为第一期依据。 | C `design/timeline-branches.md:521-530` |
| branch compare / adopt | 上游设计明确"以后再说"（S3 later）。 | C `design/timeline-branches.md:48`；A `docs/architecture/upstream-readiness.md:239` |
| branch delete / rename、per-branch model | 不在本次公开合同证据内；A 的当前状态文档把它们列为 WIP/upstream 边界。 | A `docs/architecture/timeline.md:92`；A `docs/architecture/upstream-readiness.md:137` |

小结：branch switching / cut point / side chat **都不 blocked**，也不需要新 session；它们由 `session.branch_create`、带 `branchId` 的 `session.history`/`keys`/`submit`/`stop`/`approval.respond` 和 `session.branch_end` 组合而成。真正 blocked 上游的是 `session.branch.get` 的展示详情、compare/adopt、branch delete/rename 和 per-branch model。

## 2. 问题 2：attachment

### 2.1 `attachment.put`（文本附件）

注册方式：Session 方法，namespace `attachment`，`MethodDecl {name:"attachment.put", mutating:true, interruption:drain, reply:direct}`（C `design/plugins/Attachments.md:245-247`）。

| 项 | 合同 |
|---|---|
| 请求 | `{sessionId, name?, text}`（`name` 默认 `paste.txt`） |
| 响应 | `{attachmentId, path, bytes}`；`path` 是 Session 文件绝对路径，`bytes` 是存储长度 |
| `text` 限制 | 非空；≤ **10 MiB UTF-8** |
| 帧限制 | 单请求 ≤ **16 MiB** JSON；超帧在 Rpc 分发前就断连，客户端必须自己检查编码后大小 |
| `name` 规则 | 由 SessionStore 校验：`[A-Za-z0-9._-]{1,64}`、不以点开头、不能是 `auth.json`/`auth.json.lock`、单一路径分量 |
| 错误 | `invalid_params`(-32602)、`attachment_invalid`、`attachment_too_large`、`session_closed`、`node_unavailable`、`service_reloading`（unload 取消且未 commit）、`storage_failed` |
| 不可用 | host 没装 Attachments（无 native storage）时 `unknown_method` |

证据：C `design/plugins/Attachments.md:245-257`、`261-283`、`44`、`155`；A 的实际调用点 `service.ts:4905-4914`（`attachment.put {sessionId,name,text}`，只解 `{path}`），客户端限制 `attachments.ts:8-9`（10 MiB 文本 / 8 MiB 图片）、`50-98`（文本走 put 后以 Markdown 链接拼进正文）。Desktop 的 8 MiB 图片上限是**客户端策略**，不是 wire 限制。

结果未知的处理：`attachment.put` 没有 `submissionId` 这类幂等键，也没有任何方法能列出附件（C `design/plugins/Attachments.md:392`）；C 明确把 `attachment.put` 列入"不得自动重发"的写操作（C `design/acp-gateway-implementation.md:302`）。所以超时/断线只能报告"未知，可能已存储"，由用户决定是否重试。

### 2.2 图片：没有上传方法

图片不需要 Attachments：客户端把它作为 **inline base64 image content block** 放进 `session.submit` 的 `user.content`（C `design/plugins/Attachments.md:5`、`177-180`、`228`）。限制来自输入侧：每次提交最多 **20 张**、整个请求 ≤16 MiB 帧；超过最长边 2048 px 的图由 ModelProvider 缩放而不是拒收（C `design/plugins/Attachments.md:228`、`378`）。A 的实现：`attachments.ts:76-85` 生成 `{type:'image', source:{type:'base64', mediaType, data}}`；文本与显式 `user` 不能混用（`attachments.ts:66-68`）。

### 2.3 明确不存在

| 不存在的请求 | 说明 |
|---|---|
| `attachment.upload` / 图片二进制上传 | 不存在；图片走 `session.submit.user` inline base64 |
| `attachment.list` / `attachment.get` / `attachment.read` / `attachment.delete` | 不存在；`path` 由 put 返回，模型用 `read` 工具读；文件生命周期归 Session/SessionStore（C `design/plugins/Attachments.md:360`、`392`） |
| 客户端指定存储路径 | 不存在；`name` 只是文件名，路径由 SessionStore 决定（C `design/plugins/Attachments.md:184`） |
| `session.submit` 的 attachment 字段 | 不存在；提交只有 `text` / `user` |

小结：**文本附件不 blocked**——`attachment.put` 已确认存在且有完整请求/响应/限额；**图片附件 blocked**——没有上传方法，只能把 base64 放进 `session.submit.user.content`。

## 3. MyGo 现状

| 层 | 已有 | 证据（B） | 缺口 |
|---|---|---|---|
| `internal/appserver` | `session.branches` helper：`BranchesParams{sessionId}`、`BranchesResult`、逐项 Validate、`hello.features` gate、单次读不重试、不合成目录 | [`branches.go`](../internal/appserver/branches.go) `:16-23`、`:52-111`、`:122-147` | 无 `branch_create`/`branch_end`/`attachment.put` DTO 与 helper |
| 同上 | `session.keys` helper；`session.history` 分页不变量；`session.methods` / `input_lookup` / submit / stop / approval / delete | [`keys.go`](../internal/appserver/keys.go) `:13`；[`wire.go`](../internal/appserver/wire.go) `:418-426`、`:559-563`、`:639-665`、`:700-705`；[`methods.go`](../internal/appserver/methods.go) `:59-71` | `session.methods` 的 helper 注释（`methods.go:53-57`）说没有可支持的 gate，但 `session.Open` 仍按 `hello.features` 门禁（见下行），两处说明不一致 |
| `internal/session` | `Open` 在 `hello.features` 门禁下读 `session.methods` / `session.branches` / `session.keys`；branch/keys 失败只降级并记录原因 | [`session.go`](../internal/session/session.go) `:225-256`；[`optional_reads.go`](../internal/session/optional_reads.go) `:20-88` | `HistoryStatus`/首屏窗口只有 main 的；没有"读某个 branch 的 history"入口 |
| 同上 | 只读快照：`Keys()`、`KeysStatus()`、`BranchDirectory()`、`BranchDirectoryStatus()`、`Branches()`（只含观察到的 branch id）、`View(branchID)`、`BranchSnapshot`、`BranchTransient`、`BranchState` | `session.go:821-901`、`:895-901`、`:920-940` | `View(branchID)` 读不到历史：未加载的 branch 返回空 reducer（`snapshotLocked`），所以"切过去"会看到空会话 |
| 同上 | `SubmitBranch`/`SubmitBranchWithID`/`StopBranch` 已带 `branchId`；submission UUID 供 unknown-result 读回 | `session.go:1078-1125` | 没有 create/end/attachment 方法 |
| `internal/integration` | `StateForBranch(summary, branchID)` 按 branch 出 UI state；`BranchDirectory()` 转发 session 快照 | [`integration.go`](../internal/integration/integration.go) `:269-335`、`:460-471` | 注释 `:460-461` 仍写 "BranchDirectory is always empty / no verified upstream RPC"，与实现和公开合同矛盾；`:325` 用非合同的 `Purpose` 作 label |
| `internal/ui` | 只读 branch 列表（id/state/head/current），无切换/切点操作 | [`shell.go`](../internal/ui/shell.go) `:41-47`、`:273-292` | 没有 branch 选择、branch 发送、end branch；"Attach file" 仍是 WIP 按钮（`:490`） |
| WIP 文案 | `reasonBranches`、`reasonAttachment`、capability 卡片 | [`wip.go`](../internal/ui/wip.go) `:19-25`；[`capability.go`](../internal/ui/capability.go) `:31`、`:40`、`:48` | `capability.go:31/40` 的 "no verified switch/cut-point RPC" 是**错误结论**：公开合同与 public client 都有 `branch_create`/`branch_end`，缺的只是 MyGo helper/接线 |
| 文档 | `protocol-contract.md` 的 branch/attachment 段；`development-plan.md:49-51` 的并行待办；`acceptance.md:38-39` 的 F12/F13 | [`protocol-contract.md`](protocol-contract.md) `:162`、`:167`、`:236`、`:254`、`:329`、`:334`、`:367`、`:377`、`:423`、`:429-435` | `development-plan.md:49` "没有已验证的切换 RPC" 同样需要改成"helper 未接" |

## 4. 可安全接线的最小范围

原则：只接公开合同已经证明的字段；每个 mutation 都按"unknown result = 读回，不重发"；不新增本地 branch/附件事实。

### 4.1 `internal/appserver`：三组新 DTO + helper（S0 先补合同）

| 新增 | 形状 | 校验/失败语义 |
|---|---|---|
| `BranchCreateParams` | `{sessionId, parentBranchId, at, prompt}` | MyGo 侧全部必填（`at` 总是显式发送）；`at >= 0`；`prompt` 非空。失败：`invalid_params`(-32602)、`input_blocked`、`unknown_method`（→ WIP） |
| `BranchCreateResult` | `{branch:{branchId, parentBranchId?, branchPoint?, createEvent?}}` | `branchId` 缺失按 protocol error（对齐 A `service.ts:4982-4984`）；响应形状只有 public client 证据，**在注释里写明** |
| `BranchEndParams` | `{sessionId, branchId}` | 响应忽略；`branch_ended`（已结束）、`unknown_branch`（创建尚未发布）、`session_closed` 是确定拒绝（C `design/timeline-branches.md:323-324`）；`unknown_method` → WIP |
| `AttachmentPutParams/Result` | `{sessionId, name?, text}` → `{attachmentId, path, bytes}` | `text` 非空；发送前按 UTF-8 字节数检查 ≤10 MiB，并检查编码后不超过 `MaxFrameSize`（[`framing.go`](../internal/appserver/framing.go) `:13-18`）；`path` 必填非空 |

共同要求：

- 三个方法都在已 attach 的 session stream client 上调用（A `service.ts:4908`、`4975`、`5052` 都用 `opened.stream.client`），不新开连接、不走 control 连接。
- `branch_create` / `branch_end` / `attachment.put` 都进 `MutationMethod` 的 unknown-result 视图：timeout/断线 → `ErrUnknownResult`，读回手段分别是 `session.branches`（只能观察是否多出一条 line，无法归因，见 §7）、`session.branches`（看 state 是否 `ending`/`ended`）、无（附件没有列表方法，只能如实报未知）。
- **不自动重发**（C `design/acp-gateway-implementation.md:302`；B `docs/protocol-contract.md:310-313`）。
- fake server 覆盖：成功、`-32602`、`unknown_method`、`branch_ended`、timeout/断线 unknown-result。

### 4.2 `internal/session`：branch 历史读取（"切换"的真正实现）

新增 `OpenBranch(ctx, branchID)`（或 `LoadBranch`），只做读，不重开连接：

1. ended line 用目录里的固定 `head` 作 `through`；否则先 `session.history {branchId, after:0, limit:1}` 取 `through`（A `session-reader.ts:939-949`；`branch-directory.ts:108-110`）。
2. 复用现有 latest-turn/兼容扫描逻辑（[`history.go`](../internal/session/history.go)）读该 branch 的窗口；side chat 需要整条 line 时向后翻到 line 起点（A `session-reader.ts:953-978`），`history_too_large` 保留已读部分并报 `hasOlder`。
3. `session.keys {branchId, through}`（feature gate 不变）喂 keyed fold。
4. 把事件喂给该 branch 的 reducer（`branchLocked`/`applyEventLocked` 已有 branch 隔离），并给该 branch 单独记录 history 状态，不挪用 main 的 `HistoryStatus`。
5. 同一 stream 的 live `session.event` 已按 `branchId` 路由，加载期间用与 main 相同的 pending buffer 顺序补放。

### 4.3 Cut point（最小版）

- P1 只做"从当前静止点开 side chat"：idle 时 `at = head`（A `session-reader.ts:906-909`），turn 运行中则提示"等当前 turn 结束"，不发请求。
- "从这条消息开"需要父 branch 的 raw event + turn 边界索引（A `branchCutBeforeMessage` 依赖 `eventsById`）；MyGo 目前只留 reducer 快照，没有 raw event 索引。这是第二步，先按 WIP 标出，不要用快照猜 sequence。

### 4.4 Side chat 最小闭环（UI）

- branch 列表行可选中：新增 `OnSelectBranch(branchID)`，选中后调用 4.2；发送/停止用现有 branch-aware 方法；非 `main` 且 `state != open` 时禁用发送（C `design/timeline-branches.md:332`、`654`）。
- end branch 只对非 `main` 且 `state` 为 open/ending 的行出现，二次确认后发 `session.branch_end`，等 `session.branches`/`session.state` 收敛，不本地改状态。
- 右侧多 tab 面板、branch 内联审批、模型/effort 继承展示属于 UI parity 阶段，本次不做；先让"切过去能读、能发、能停、能结束"成立。

### 4.5 附件（只做文本）

- 入口可用：`mygo.Dialog.Open`（框架公开 API，`mygo@v0.8.3/dialog.go:140`）选文件 → 读取 → 只接受文本（UTF-8/扩展名白名单；对齐 A `attachments.ts:36-40`）→ 检查 10 MiB → `attachment.put` → 把 `[name](path)` 拼进 draft → 走现有 `session.submit`。
- gate：`Session.Methods()` 里出现 `attachment.put` 才启用；没有 `session.methods` 或没声明 → 保持 WIP（host 无 Attachments 会是 `unknown_method`，C `design/plugins/Attachments.md:44`、`155`）。
- 图片、URL、非文本文件保持 WIP：没有上传方法，且 inline image block 会触碰 `user.content` 构造与模型图片能力（C `design/plugins/Attachments.md:5`、`228`、`378`）。

### 4.6 验收与证据门禁

- 全部新 helper：fake-server 表驱动 + `-race`；`gofmt`/`go vet`/`go test ./...` 通过只能算 L2。
- 至少一条 real-daemon 流程（create branch → history → submit → stop → end → 目录收敛）才能写 L3；L4（真实窗口）另计。
- 不允许用 `Purpose` 或任何本地字段补 branch 展示事实。

## 5. 必须保持 WIP 的入口

| 入口 | 为什么必须 WIP |
|---|---|
| `session.branch.get` 的 `purpose` / `cutMessages` / `end` 详情 | 上游设计文档有，`wire.ts` 无 DTO，public client 无调用点；目录里没有 `purpose` |
| 每消息 "Side chat from here" 的精确 cut | 需要父 branch raw event 索引；先做静止点切分 |
| branch 内联审批 / 右面板多 tab / per-branch 模型选择 | UI parity 与上游模型能力未就绪；branch 继承 main 的 model/effort（A `upstream-readiness.md:137`） |
| branch compare / adopt | 上游 S3 仍 "Later"（C `design/timeline-branches.md:48`） |
| branch delete / rename | 本次公开合同证据里没有对应方法 |
| transport 丢失后 branch 级 resync | 目前 `Reattach` 只重建 main attachment；branch 需要重新 `OpenBranch`，未验证前不宣称 |
| 图片 / URL / 二进制附件 | 无上传方法；inline base64 需要 `user.content` 构造与模型能力判定 |
| 附件列表 / 读取 / 删除 / 预览 | 没有公开方法；`path` 只由 put 返回 |
| 任何本地 branch/附件事实 | 违反单一事实源 |

## 6. 结论

### P0（现在就能做，且不引入新事实）

1. 修正错误结论与陈旧注释：`internal/ui/capability.go:31/40`、`internal/ui/wip.go:24` 的措辞改为"helper 未接"；`internal/integration/integration.go:460-461` 的 "always empty / no verified upstream RPC" 注释删除或改写；`docs/development-plan.md:49` 同步。
2. 只读 branch 目录按合同字段渲染：`branchId`、`parentBranchId`、`state`、`head`、`endEvent`、`source`；label 用 `branchId`，**不再用非合同的 `Purpose`**（`integration.go:325`）。
3. 目录刷新入口：`session.branches` 只在 Open 读一次；补一个显式 `RefreshBranchDirectory(ctx)`，供 create/end 之后与用户手动刷新调用（A 在 open/reattach 读一次并维护 in-memory 目录，`upstream-readiness.md:143`）。
4. `Attach file` 继续 visible-disabled WIP；branch 行保持只读，直到 P1 的历史读取落地。

### P1（需要新 DTO/helper，目标 L2/L3）

1. `internal/appserver`：`branch_create`、`branch_end`、`attachment.put` 三个 DTO + helper（§4.1），包含 unknown-result 建模与 fake-server 覆盖。
2. `internal/session.OpenBranch`（§4.2）：按 branch 读 head/history/keys，喂独立 reducer；`View(branchID)` 对未加载 branch 明确返回 "未加载" 而不是空会话。
3. UI：branch 行可选中 → `OnSelectBranch` → OpenBranch；branch-aware 发送/停止；非 main 的 end branch（二次确认）。先不做右面板 tab。
4. Side chat 最小版：静止点切分（`at = head`）→ `branch_create` → 立即 `OpenBranch` 并选中该 line → 发送/停止/结束；turn 运行中拒绝切分并说明原因。
5. 附件文本版：`session.methods` gate + `mygo.Dialog.Open` + 10 MiB 检查 + Markdown 链接；图片保持 WIP。
6. 验收：`acceptance.md` F12/F13 从"未通过（只有只读列表/无 helper）"改为"L2/L3 测试级证据"，L4 仍写未通过。

### WIP（停在边界）

- 每消息精确 cut、`session.branch.get` 详情、branch compare/adopt、branch delete/rename、per-branch 模型选择。
- branch 级 resync/reload 的可验证语义（transport 丢失后）。
- 图片/URL/二进制附件、附件列表/读取/删除/预览。
- 右面板多 tab side chat 与 branch 内联审批的 1:1 视觉（归 ui-parity）。

## 7. 未决问题（需要上游确认后才推进）

1. `session.branch_create` 的响应字段没有公开设计文档，只有 public client 的解码；若上游返回形状不同，属 wire 修正，不在 MyGo 加兼容别名。
2. `branch_create` 没有幂等键：超时后只能读 `session.branches` 判断是否出现了新 line，无法证明"没有创建"；是否需要上游提供 client 侧 operation id，需在 issue 里提出。
3. `session.branch_create` 的 `at` 是否可省略、省略时的默认切点，公开合同未写；MyGo 一律显式发送。
4. `session.branches` 对旧 daemon 的缺失已由 public client 用 event-derived 不完整目录降级（A `branch-directory.ts:126-207`）；MyGo 是否实现同样的 fallback 需要单独决定（当前 `Session.Branches()` 只给观察到的 id，不合成元数据，属于更保守的边界）。
5. `session.methods` 是否在所有 v3 连接都列出 `attachment.put`：上游合同说 namespace 注册即可见（C `design/contracts/session-boundary.md:247-248`），但 MyGo 的 Open 仍按 `hello.features` 门禁 `session.methods`（`session.go:231`），gate 缺失时附件入口必须 fail-closed。

## 8. 文档同步（本次未改，落地时一起做）

| 文件 | 需要的更新 |
|---|---|
| [protocol-contract.md](protocol-contract.md) | §1 的"不做 branch/side chat"改为指向本文的 P1/WIP 边界；§8/§9/§10/§16/§17 补 `session.branch_create`/`branch_end`/`attachment.put` 与 gate |
| [development-plan.md](development-plan.md) | §"并行待办"第 1、2 条改成"helper 未接"，并链接本文 |
| [acceptance.md](acceptance.md) | F12/F13 的证据描述按 §6 更新 |
| [ui-parity.md](ui-parity.md) | composer 附件、branch 列表/切换的差距行补链接 |
| `internal/ui/capability.go`、`internal/integration/integration.go` | 陈旧注释与 WIP 文案（P0-1） |
