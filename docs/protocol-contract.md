# Protocol v3 Wire 合同

Status: active（合同已冻结；Wave 1 实现 transport/client/DTO 子集，Wave 2 实现 session stream/attach、detach、四个 mutation helper 和 supervisor reconnect；real-daemon 控制面已验证，session re-attach/resync、UI 接线与真实 session 流仍未验证）
Last updated: 2026-11-09
协议依据：upstream 公开设计合同（`session-boundary.md`、`session-access.md`）与 public protocol v3；不引用 Rust 内部源码或 Desktop private wire 作为实现合同。
Owner: protocol

本文冻结 `dimagent-mygo` 第一阶段（Milestone 2 + Milestone 3 最小闭环）需要的 protocol v3 最小合同。它描述 wire 事实和 Go 侧类型边界；transport/client、supervisor 和 projection 的实现分别在 `internal/appserver`、`internal/supervisor`、`internal/projection`，本文不重复实现细节。

实现或扩展能力前请先读本文的"边界"与"未决问题"两节；合同之外的能力一律 fail-closed，不用本地存储或推测补齐。

## 1. 边界

必须遵守：

1. Rust `dim-app-server` 是 session、branch、timeline、approval、model、permission 的唯一事实源。
2. 只通过公开 socket/pipe + protocol v3 访问；不读 App Server 的 SQLite/JSONL/session 文件/`daemon.log`，不调 plugin 实例或私有 IPC。
3. 不建第二份 session/message/run/approval 存储，不写本地 session 数据库。
4. 能力缺失时 visible-disabled 并说明原因；不伪造成功、不返回空列表冒充真相。
5. wire 类型以公开 protocol v3 合同为准；公开合同没有声明的字段/方法默认 fail-closed，不用 Rust 内部源码或 Desktop private wire 反推合同。

本阶段不做：branch/side chat、question、MCP、skills、workflow、scheduler、account、plugin、UI extensions、modality、runtime 管理。

## 2. 公开协议依据

本合同只把 public protocol 和公开 Desktop protocol contract 作为实现依据：

- `session-boundary.md`：`session.methods` 是 public surface；`session.methods {sessionId}` → `{methods:[{name:string, mutating:bool}]}`，`mutating` 必填。
- `session-access.md`：`session.input_lookup`/`lookup` 的提交标识查找回答 `accepted` 或 `unknown`，不写入；`accepted` 找到已提交输入，不等于 turn 成功。
- `session.keys` / `session.branches`：公开 Desktop protocol contract 已有；`dim-electron` 的 `session-reader.ts:1041-1074` 正式调用，`app-server-protocol/src/wire.ts:267-303` 定义 DTO。MyGo helper 和 session 读取已落地，UI 接线与真实验收仍未完成。

本仓库的 `headless`/`fake-server` 测试只证明本地实现行为，不构成上游合同；real-daemon 目前只有控制面 smoke 证据，不覆盖真实 session 流。本文不引用 Rust 内部源码；Desktop 证据仅限公开 client 调用和 public protocol DTO。

## 3. 传输与帧

- 一行一个 JSON-RPC 2.0 对象，LF 结尾。读端容忍 CRLF 与空行，空行不产生消息。
- 单帧上限 **16 MiB**（`16 * 1024 * 1024`）。未结束的残留 buffer 也按同一上限检查，超限即协议错误，不继续累积。
- 写端不产生空行、不写 BOM；用标准 JSON 序列化即可，不做额外的非 ASCII 转义（控制字符按 JSON 规则转义）。`params` 为 nil 时**省略该字段**，不发 `"params": null`。
- Go 侧不要用默认 `bufio.Scanner`（64 KiB 上限）；应自建按行读取 + 长度上限，且能在任意 chunk 边界拼接。

## 4. JSON-RPC 信封

```text
request       { jsonrpc:"2.0", id:<number|string>, method:string, params?:object }
success       { jsonrpc:"2.0", id:<number|string>, result:<any> }
failure       { jsonrpc:"2.0", id:<number|string|null>, error:{ code:int, message:string, data?:{ appError:string, ... } } }
notification  { jsonrpc:"2.0", method:string, params?:object }   // 不允许带 id
```

校验规则（必须与 public protocol v3 一致）：

1. `jsonrpc` 必须是 `"2.0"`。
2. 带 `method` 的消息若带 `id` → 协议错误。
3. reply 必须**恰好**有 `result` 或 `error` 之一（用 key 存在性判断，`result: null` 合法）。
4. `error.data` 可缺省；存在时 `data.appError` 必须是字符串。
5. `id: null` 的 error reply 不属于任何请求：上报错误，不崩溃、不断连接。
6. 未知通知方法必须原样透传（保留 `params` 全部未知字段）。

请求关联：

- 请求 id 由客户端生成，从 1 递增的整数即可；接收端同时支持 number 与 digit-string。
- 默认单请求超时 **30s**；可按请求覆盖；长生命周期调用用"无超时 + context 取消"。
- 关闭连接时必须拒绝所有 in-flight 请求（`connection_closed`），并关闭 transport。

客户端本地错误（与服务端错误分开建模）：

| appError | code | 可重试 |
|---|---|---|
| `connection_closed` | -32000 | 是（重连） |
| `request_timeout` | -32000 | 是（读操作） |
| `request_aborted` | -32000 | 否 |

注意：-32000 也是服务端通用错误码，语义只能靠 `appError` 区分，不能只看 code。

## 5. 连接模型

```text
control connection (1)
  hello / daemon.ping / session.catalog / session.create / session.delete / management.*

session stream (每打开 session 1 条)
  session.attach / session.detach / session.history / session.submit / session.stop / session.*
  → session.event / session.state / approval.* / session.notice / session.closed
```

- control 连接不 attach，也不接收 session 通知；列表只是快照。
- 每条连接（含每条 stream）都要先 `hello`，各自有一份 `hello.features`。
- 同一 session 多窗口共享一个 attachment；最后一个 holder 释放才 detach。
- 同一连接对同一 session 只有一个 attachment；重复 attach 会替换旧订阅，客户端不主动重复 attach。

## 6. `hello`

请求：

```json
{"protocolVersion":3,"clientKind":"other","clientName":"dimagent-mygo"}
```

`clientKind` 合法值只有 `"cli" | "web" | "other"`；Go 客户端固定发 `"other"`。

响应：

| 字段 | Go 类型 | 说明 |
|---|---|---|
| `instanceId` | `string` | 必填非空；daemon 重启后变化，是"是否重启"的唯一依据 |
| `serverVersion` | `string` | 展示用 |
| `protocolVersion` | `int` | 必须等于 3 |
| `daemonProfile` | union | `{kind:"product"}` / `{kind:"configured",modelConfigSha256:string}`（非空）/ `{kind:"mock"}` |
| `features` | `[]string?` | 连接级能力快照 |

校验失败即断开并停止重试（不 spawn、不降级）：版本不是 3、`instanceId` 为空、`daemonProfile` 形状非法。

辅助方法：`daemon.ping` → `{sessions, connections, serverVersion}`；`daemon.shutdown {force:false}`。二者只用于 supervisor 的替换/退出判定，不用于列表或运行状态。

## 7. 列表：`session.catalog` 与 `session.summaries`

**不存在 `session.list`。** 列表只走下面两条路。

### 7.1 `session.catalog`（首选）

参数：

| 字段 | 类型 | 说明 |
|---|---|---|
| `limit` | `int?` | Go 第一阶段固定 200 |
| `cursor` | `string?` | 不透明分页游标，原样回传 |
| `query` | `string?` | 服务端搜索 |
| `cwd` | `string \| []string?` | 单个 workspace 用 string；数组需要 `session.catalog.workspaces` |
| `withPreview` | `bool?` | |
| `withInternal` | `bool?` | 需要 `session.catalog.origins` |
| `withAcpSessions` | `bool?` | 需要 `session.catalog.origins` |

响应：`{ sessions: SessionSummary[], nextCursor?: string }`。

`SessionSummary` 最小字段：`sessionId`、`cwd`、`model{providerId,modelId}`、`createdAt`、`running`、`sequence`、`attachedClients`、`updatedAt`；可选 `agentId`、`origin`(`client|acp|internal|unknown`)、`parentSessionId`、`title`(可 null)、`titleSource`(`auto|manual|null`)、`titleRevision`、`preview`(可 null)、`closed`。

规则：

- 缺 `session.catalog` feature → 列表不可用，明确报错；**不要**回退到 summaries，也不要本地伪造列表。
- 缺 `session.catalog.origins` → 删除 `withInternal` / `withAcpSessions` 后重发。
- 分页循环到 `nextCursor` 缺失或为空字符串；两者都表示结束，不得再发请求。并发刷新合并成一次遍历（单飞）。

### 7.2 `session.summaries`（仅例外）

参数 `{includeClosed?: bool}`；响应是**裸数组**，不是对象。

旧 daemon 对 `includeClosed` 回 `-32602`；此时**只重试一次**不带该字段的请求，其他错误照抛。第一阶段只在需要 closed 行时才用，不用于普通列表。

### 7.3 创建

`session.create` → `{sessionId, model?, modelPreferences?}`；参数含 `cwd`、`model{provider,modelId}`、`modelPreferences`、`permissionPolicy{preset}`。缺 `sessionId` 视为 INTERNAL。

## 8. attach：订阅先行

顺序不能变：

1. 新建一条连接并完成 `hello`；
2. **先**注册通知监听（approval replay 会紧贴 attach 到达）；
3. 发 `session.attach {sessionId}`；
4. 取 head：`attach.sequence` 存在则用它，否则 `session.history {branchId:"main", after:0, limit:1}` 的 `through`；
5. 读最新 turn window（见 §9）；
6. `session.branches` / `session.keys` 的公开 Desktop protocol contract 已有（feature `session.branches` / `session.keys`）；MyGo 在 `internal/session.Open` 读取可选只读 snapshot，UI 接线与真实验收仍未完成。
7. 建 projection；把 attach 到 projection 就绪之间到达的事件/状态按**到达顺序**补放。pending buffer 按 `event.id` 去重；进入 TimelineFold 后必须再按 `(branchId, sequence)` 做幂等，不能只依赖 event.id。若同一 sequence 出现不同 id，按协议错误记录并停止盲目合并，等待上游语义确认。

Go 实现：`internal/appserver/session_stream.go` 的 `OpenSessionStream` 假定 supervisor 已完成步骤 1 的连接与 hello，实现步骤 2–4 以及步骤 7 的订阅先行、到达顺序缓冲和 `event.id`/`(branchId, sequence)` 冲突检查；它不构建 projection，也不读 latest turn 或 `session.branches`/`session.keys`。`Detach` 与 `Release` 按本节超时语义发送 `session.detach` 并关闭连接。

`internal/session` 在 stream 之上实现步骤 5–7 的会话组合：首屏 history（latest-turn / 兼容扫描）、按 branch 的 projection、attach 期间有界 pending buffer，以及 remote closed / transport lost / local release 的区分。Open 在 `hello.features` capability gate 下读取可选的 `session.branches`/`session.keys`，失败只降级为空并报告，不合成 snapshot；`Session.Keys`/`BranchDirectory` 暴露只读副本，`Session.Branches()` 只返回从 attach/event 观察到的 branch id，不是 daemon directory。公开 Desktop protocol contract 已有，仍未完成的是 UI branch switching/cut point 和真实验收。它不自动 reconnect/resync，也不把投影写入本地存储。

`session.attach` 响应：

| 字段 | 类型 | 说明 |
|---|---|---|
| `sequence` | `int?` | 主分支 head；**老 daemon 可能不返回**，必须兜底 |
| `running` | `bool?` | |
| `model` | `{providerId?,modelId?}?` | |
| `modelPreferences` | `{reasoningEffort?,serviceTier?}?` | |

其他规则：

- attach 没有"快照完成"标记；`session.event` 可能早于 `session.state`，必须乱序合并。
- `session.detach {sessionId}`：删除路径要等回复（2s），普通释放可 fire-and-forget（500ms）后关闭连接；无论成败都要关连接，否则 daemon 侧 attachment 计数不释放。
- `session.closed {sessionId, reason?}` 是终态：释放全部 holder、清理 interaction/branch 状态、关闭连接，且**不**发 resync。
- Go `SessionStream` 把 `session.closed` 记为 `closed` 并关闭连接；transport 意外结束时记为 `lost` 并带 `transport_closed`，由上层触发 domain resync，stream 自己不会重开。

## 9. `session.history`

参数：

| 字段 | 类型 | 说明 |
|---|---|---|
| `sessionId` | `string` | |
| `branchId` | `string?` | 默认 `main` |
| `after` | `int?` | 前向页下界（不含） |
| `limit` | `int?` | |
| `through` | `int?` | 上界（含）；传了则回包 `through` 必须一致 |
| `before` | `int?` | 前向游标的镜像：取该序列之前的事件 |
| `latestTurn` | `bool?` | 需要 feature `session.history.latestTurn` |

响应：

| 字段 | 类型 | 说明 |
|---|---|---|
| `events` | `RawEvent[]` | |
| `through` | `int` | 本页上界 |
| `nextAfter` | `int` | 下一页前向游标 |
| `done` | `bool` | |
| `nextBefore` | `int?` | 仅 latestTurn 反向翻页用 |
| `turnStart` | `int?` | latestTurn 读里最新 `turn/start` 的 sequence |

`RawEvent`：`id:string`、`sequence:int`、`create_at:int`（epoch 毫秒）、`tags?:[]string`、`type:string`、`name:string`、`data?:any`。

分页不变量（Go fake server 与 client 校验都要实现；普通 history、latest-turn、兼容扫描和 attach fallback 均适用）：

1. 每个事件 `after < sequence <= through`；
2. `after <= nextAfter <= through`；
3. 未 `done` 时 `nextAfter > after`；
4. 请求带了 `through`，响应 `through` 必须相同。

首屏"最新 turn"策略（沿用 protocol v3 公开语义）：

- 有 `session.history.latestTurn`：`{latestTurn:true, limit:4096, through:head}`，用 `nextBefore` 往前翻到 `done`；多页的 `turnStart` 必须一致，否则放弃窗口改走兼容扫描。
- 无该 feature：从 head 往前 512 条/页、最多 64 页，用 `{after, limit, through}` 扫到 turn 边界，并尽量带上该 turn 的 prompting input。
- `appError === "history_too_large"`：该批原子事件无法塞进一帧。首屏降级为"空窗口 + hasOlder=true"，Session 仍可打开；不要反复重试同一批。错误映射必须显式覆盖该值。
- `through`、`nextAfter`、`nextBefore`、`turnStart` 若缺失、非整数或违反不变量，必须 fail-fast 为 protocol/schema error；不能依赖 `NaN` 比较或静默走 fallback。多页 latest-turn 的 `turnStart` 必须存在且恒定。RawEvent 的 `id`、`sequence`、`create_at`、`type`、`name` 也需在普通 history 路径运行时校验。

往回翻用 `before`，直到页首是 "turn 前的 user input" 或 `sequence === 1`。

Go 实现：`internal/session/history.go` 实现 latest-turn 反向分页、`turnStart` 漂移回退兼容扫描、`history_too_large` 和 page-cap 降级；缺 `turnStart`/`nextBefore` 或 `nextBefore` 非递减仍是 protocol error。首屏窗口只覆盖 main branch，side branch 的 durable event 走各自 reducer。

可选读（公开 Desktop protocol contract 已有）：

- `session.keys` → `{through, entries:[{key, sequence, inputId, text}]}`。
- `session.branches` → `{branches:[{branchId, parentBranchId?, branchPoint?, createEvent?, source?, state:"open"|"ending"|"ended", head, endEvent?}]}`。

MyGo 已有 decoder/types 和 `SessionKeys`/`SessionBranches` helper；`internal/session.Open` 已按 `hello.features` capability-gated 读取，失败只降级为空并报告，不合成 snapshot。UI branch switching/cut point 和真实验收仍未完成。

## 10. `session.submit` / `session.stop`

`session.submit`（在 session stream 上）：

```json
{"sessionId":"…","submissionId":"<client uuid>","branchId":"main","text":"…","user":{…},"mode":"queue|steer|append"}
```

- `submissionId` 是幂等键，客户端生成 UUID；`text` 与 `user` 至少一个。
- `mode` 仅分支/steering 使用；`steer` 不等待 `turn/start`。
- 响应至少含 `{event: RawEvent}`。`turnId` **不是** wire 字段：从随后 `session.event` 的 `turn/start` 取 `data.turn_id ?? data.turnId`，最多等 2s，超时则不带。
- 附件：文本走 `attachment.put {sessionId, name, text}` → `{path}` 后拼进 text；图片作为 inline `image` base64 block 放进 `user.content`；两者不能与显式 `user` 混用。第一阶段只做 text 即可。
- 结果未知（超时/断线）时用 `session.input_lookup {sessionId, branchId, submissionId}` 读回，**不重发**。公开 `lookup` 请求包含 `branchId`；`Session.LookupInput` 在 Go 调用侧默认当前 branch。公开合同确认响应为 `{status:"accepted", event:...}` 或 `{status:"unknown"}`。`accepted` 只表示按 `submissionId` 找到已接受的输入，不代表 turn 成功；`unknown` 是最终未找到，不能据此自动重试 submit。

`session.stop {sessionId, branchId, reason?}`，响应内容忽略；停止状态由 `session.state` 收敛。

Go 实现：`internal/appserver/mutations.go` 的 `Submit`/`Stop` 发送这两个方法。`Submit` 校验 `submissionId`、`branchId` 和 `text|user`，按最小 `{event}` 解码；`Submit` 遇到 timeout、断线、`request_aborted`、`submission_unknown`、`submission_ambiguous` 或畸形结果时返回 unknown-result 并注明读回方法，绝不自动重发。`internal/session` 的 `SubmitWithID`/`SubmitBranchWithID` 额外返回已发送的 UUID，供调用方在 unknown-result 时读取 `submissionId`；`appserver.InputLookupOutcome` 已按上述 tagged union 解码并校验；`appserver.InputLookup`/`Session.LookupInput` 仍原样返回 JSON。v3 调用必须带 `branchId`；`InputLookupParams` 的空值会省略该字段，`Session.LookupInput` 默认当前 branch。attachment 上传尚未实现。

## 11. 通知（只出现在 session stream 上）

| 方法 | 参数要点 | 处理 |
|---|---|---|
| `session.event` | `{sessionId, branchId, event:RawEvent}` | durable Timeline 唯一事实；按 `(branchId, sequence)` 归并；按 `event.id` 幂等 |
| `session.state` | `{sessionId, branchId, running, stopping}` | 运行态；与 event 无序，按到达顺序合并 |
| `approval.request` | `{sessionId, branchId, requestId, source{kind,callId?}, call{name,input}, reason}` | 重放时**整体替换**该 session 的 pending 集合 |
| `approval.resolved` | `{sessionId, branchId, requestId, decision:"allow"\|"deny"}` | 外部或本端已决 |
| `approval.cancelled` | `{sessionId, branchId, requestId, reason}` | 已撤销 |
| `session.notice` | `{sessionId, source, kind, data}` | 按 source+kind 分派；第一阶段只记录 title 可选 |
| `session.closed` | `{sessionId, reason?}` | 终态，见 §8 |
| `session.model.progress` | `{sessionId, branchId, turnId?, requestId?, textPartId?, thinkingPartId?, text, textOffset, thinking, thinkingOffset, tools[]}` | transient，不进 Timeline；第一阶段可只存不透传 |

Timeline 的 `type/name` 由 App Server 事件定义；projection 需要的组合见另外的 projection 合同文档，wire 层不校验 `data` 形状。

## 12. Approval

`approval.respond`：`{sessionId, branchId, requestId, decision:"allow"|"deny"}`（在 session stream 上）。

- UI option 折叠为 decision：`allow_once|allow_all → allow`，其余 `deny`。
- 回 `-32602` 表示该 approval 已不存在：按 cancelled 处理，不报错、不重试。
- 持久化授权（always allow）走 `session.grants.add`，且必须先由 `session.methods` advertise；本阶段可不做。
- pending question 不在 attach replay 里，需从当前 turn 的 `step/tool-call` 缺少对应 `step/tool-result` 推导；本阶段不做。

Go 实现：`ApprovalRespond` 发送折叠后的 `allow`/`deny`，`-32602` 返回 cancelled 而不报错或重试；always-allow（`session.grants.add`）和 question 仍未实现。UI 的 approve/deny 仍是 visible-disabled WIP，不报告未发送的决策。

## 13. 错误

判定顺序：先看 `code === -32602`（一律 invalid params / BAD_REQUEST），再按 `appError` 分派，未知 `appError` 降级 INTERNAL 且不可重试。

| appError | 行为 |
|---|---|
| `unknown_method` | 能力不存在；入口 disabled/WIP，不重试 |
| `service_reloading` / `connection_busy` | 可重试（有界退避） |
| `service_unavailable` | 不可重试，提示用户 |
| `subscription_changed` | REPLAY_GAP：只读操作强制 re-attach 并**只重试一次**；写操作不重放 |
| `session_closed` | 刷新列表，丢弃该 attachment |
| `session_active` | 删除被拒；保持行，不重试 |
| `protocol_version` / `session_protocol_version` | 不兼容，停止重试并提示升级 |
| `submission_unknown` | 用 `session.input_lookup` 读回 |
| `submission_ambiguous` | 不重发，提示并刷新 |
| `permission_conflict` / `permission_rules_conflict` | 重读 permission，让用户确认 |
| `question_not_pending` | 用 tool-result 判断是否已 settle |
| `operation_interrupted` | 先读当前状态，不盲重试 |
| `local_transport_required` | 本地启动/配置错误 |
| `history_too_large` | 见 §9 降级 |

## 14. unknown mutation 与 resync 边界

- **`session.resync` 不是 wire 方法。** wire 层只有"transport 关闭"和 `subscription_changed`。
- Go 侧定义 domain event `session.resync {sessionId}`，触发条件：session stream transport 意外关闭，或 `subscription_changed` 恢复失败。
- 收到 resync 的动作固定为：重新 `openStream` + `hello` + `attach` + 重读最新 turn window，**按 branch sequence 重建**；不要用 renderer/VM 的 cursor 当 resume token。
- `session.closed` 不触发 resync（它是终态，不是流丢失）。
- `instanceId` 变化视为 daemon 重启：按新 daemon 的 history 重建，旧 attachment 全部丢弃。
- mutation（submit/stop/approval.respond/session.delete）结果未知时：只读回或提示，绝不自动重发。四类 mutation 都必须各有 unknown-result fixture；不能只用 submit 覆盖。
- 重连单飞：并发重连共享同一个进行中的尝试；关闭中的 supervisor 拒绝新重连。

Go 实现现状：四个 mutation helper 已实现，fake-server 表驱动测试覆盖四类的 timeout/断线 unknown-result，`submit` 另有 `submission_unknown`/`submission_ambiguous`；`internal/session.SubmitWithID` 返回已发送的 submission UUID，`Session.LookupInput`/`appserver.InputLookup` 可按该 UUID 原样读回 `{status:"accepted",event}` 或 `{status:"unknown"}`，绝不自动重发。supervisor 已实现 `Reconnect`（单飞、重新解析 endpoint、关闭旧 attachment、按 backoff 重试、generation/stale 标记、transport 终止原因和 close 竞争处理）；`Client.Done`、`SessionStream` 的 `lost`/`transport_closed` 和 `Connection.Done`/`Err` 提供 resync 触发信号。session 层的 re-attach/domain resync 仍由上层编排，`internal/session` 不会自己重开或重建 attachment。

## 15. 当前合同缺口与上游确认项

以下项目已进入实现门禁，未确认前不得用本地推测补齐：

- `session.history` 的 `through` 必填性和 `through < after` 的服务端行为仍缺 public protocol 确认；`session.summaries` 裸数组形状在公开合同中未明确，fake server 只证明本地行为。
- `RawEvent` 的 `create_at`、`type`、`name` 已在普通 history 路径做运行时校验（`HistoryResult.Validate`）；TS interface 不能作为运行时验证。
- TimelineFold 的 durable 幂等以 `(branchId, sequence)` 为准，attach pending buffer 以 `event.id` 去重；同 sequence 不同 id 的冲突处理仍需要上游语义确认。`SessionStream` 已在路由层把该冲突报告为协议错误并丢弃第二个事件；projection reducer 仍按 sequence 丢弃重复项，fixture 会拒绝该输入。
- unknown `appError` 已统一映射为不可重试 INTERNAL；`history_too_large` 已显式映射，但 §9 的首屏降级窗口未实现。

## 16. Capability

两层门禁都要进 capability snapshot：

1. 连接级 `hello.features`（每条连接各一份）。第一阶段相关：
   `session.catalog`、`session.catalog.origins`、`session.catalog.workspaces`、`session.history`、`session.history.latestTurn`、`session.keys`、`session.branches`。`session.keys` / `session.branches` 有公开 Desktop protocol contract；MyGo session 已 capability-gated 读取只读 snapshot，UI 接线未完成。
2. session 级 `session.methods {sessionId}` 是 public surface，响应 `{methods:[{name:string, mutating:bool}]}`（`mutating` 必填，见公开合同 `session-boundary.md`），用于插件方法（`session.rename`、`shell.run`、`session.goal.*`、`session.grants.add`、`session.collaboration.*`）。客户端在 session stream 上调用。

规则：缺 feature 时功能降级或 disabled，并说明原因；不写本地替代事实。`session.catalog` 缺失是"列表不可用"，不是"空列表"。

当前 Go 实现有连接级 `hello.features` 门禁（`HasFeature`/`CapabilityError`）；`appserver.SessionMethods`/`Session.Methods` 按公开合同实现 `session.methods` 读取（目前仍先检查 `hello.features` 是否声明 `session.methods`），插件方法入口保持 WIP。`session.keys` / `session.branches` 的公开 Desktop protocol contract 已有；Go 侧已有 decoder/types 和 `SessionKeys`/`SessionBranches` helper，`session.Open` 已按 capability gate 读取并暴露只读 snapshot，失败降级为空；UI 接线与真实验收未完成。

## 17. Go DTO（`internal/appserver`）

DTO 已实现在 `internal/appserver/wire.go`，编译并由 `headless` 测试覆盖。请求信封与响应关联在 `client.go`、`framing.go`：

```go
// internal/appserver/client.go
type Notification struct {
    Method string
    Params json.RawMessage
}

type RPCError struct {
    Code    int
    Message string
    Data    json.RawMessage // 解析出 AppError 后仍保留原始 data
}
```

请求由 client 内部构造：id 是逐渐递增的 `int64`，`params` 为 nil 时省略字段；接收端用 `responseKey` 同时接受 JSON number 和 digit string。

DTO 规则：`json.RawMessage` 承载未知/未用字段；整数字段用 `int64`；可空字段用指针；不做"宽容别名"（例如 `provider_id` 与 `providerId` 的兼容只允许在明确的 legacy 读路径里，wire 层不加）。

| Go 类型 | wire 方法 | 状态 |
|---|---|---|
| `HelloParams` / `HelloResult` | `hello` | 已实现；含 `daemonProfile`、`instanceId`、`features` 校验 |
| `CatalogParams` / `CatalogResult` / `SessionSummary` | `session.catalog` | 已实现 |
| `SummariesParams` | `session.summaries` | 已实现；响应 `[]SessionSummary` |
| `CreateParams` / `CreateResult` | `session.create` | 已实现；`internal/integration.CreateSession` 负责创建后打开专用 stream |
| `AttachParams` / `AttachResult` / `SessionStream` / `StreamState` | `session.attach` | 已实现；`OpenSessionStream` 负责订阅先行、head 兜底和通知路由 |
| `DetachParams` | `session.detach` | 已实现；`Detach`/`Release` 都关闭连接 |
| `HistoryParams` / `HistoryResult` / `RawEvent` | `session.history` | 已实现；分页不变量运行时校验 |
| `KeysResult` / `BranchesResult` | `session.keys` / `session.branches` | 公开 Desktop protocol contract 已有；Go 侧已有 decoder/types 和 `SessionKeys`/`SessionBranches` helper；`internal/session.Open` 已 capability-gated 读取并暴露只读 snapshot；UI branch switching/cut point 与真实验收未完成 |
| `MethodsParams` / `MethodsResult` / `MethodDeclaration` | `session.methods` | 已实现；built-in，`{methods:[{name, mutating}]}`，`mutating` 必填 |
| `SubmitParams` / `SubmitResult` + `Submit` | `session.submit` | 已实现；unknown-result 不重发 |
| `StopParams` + `Stop` | `session.stop` | 已实现 |
| `ApprovalRespondParams` + `ApprovalRespond` | `approval.respond` | 已实现；`-32602` 映射为 cancelled |
| `DeleteParams` + `Delete` | `session.delete` | 已实现；响应体被忽略，需用 catalog 读回 |
| `InputLookupParams` | `session.input_lookup` | 已实现；响应 `{status:"accepted",event}` 或 `{status:"unknown"}`；`InputLookupOutcome` 解码校验，`InputLookup` 原样返回 JSON，不自动重发 |
| `EventNotice` / `StateNotice` / `ApprovalRequest` / `ApprovalResolved` / `ApprovalCancelled` / `ClosedNotice` / `SessionNotice` / `ModelProgress` | 通知 | 已实现，`DecodeNotification` 按方法解码；未知方法原样透传 |
| `LocalError` / `MappedError` | 客户端本地错误 / `appError` 映射 | 已实现 |

`SessionStream` 不构建 projection，也不自动 reconnect/resync；它只把意外结束报告为 `lost`。latest-turn、`session.input_lookup`、`session.methods` 的读取已按公开合同实现；`session.keys`/`session.branches` 有公开 Desktop protocol contract，Go 侧已有 decoder/types 和 `SessionKeys`/`SessionBranches` helper，`session.Open` 已读取只读 snapshot；UI branch switching/cut point 与真实验收未完成；插件方法入口仍保持 WIP。

## 18. Transport 与 Supervisor 参数（已实现，real-daemon 控制面已验证）

以下参数已实现在 `internal/supervisor`，由 `headless` 测试、fake daemon 和 real-daemon 控制面 smoke 覆盖；真实 smoke 仅验证 spawn/hello/catalog/close。Wave 2 增补了 dial/hello 超时、dial 错误分类、spawn 失败不声明 ownership、等待 listen 上限、关闭时先停 stream 再 ping、按单连接重试、endpoint 相对路径/symlink 和 fake daemon 按次 ping 的 QA 测试；Windows named pipe 运行时仍未验证。

- Unix：`<productHome>/rust/daemon.sock`；`<productHome>/rust` 必须 0700、属于当前用户、canonicalize 后使用。
- Windows：`\\.\pipe\dim-rust-<sha256("dimcode:"+canonicalNativeHome)[:32]>`。
- 连接超时 5s；只有 `ENOENT`/`ECONNREFUSED` 才算"daemon 不存在"并允许 spawn，其他错误直接上报。
- spawn 参数：`--unix <endpoint>`（或 `--pipe`）、`--no-ws`、`--home <nativeHome>`，可选 `--mock-model`、`--defaults <path>`、`--model-config <path>`，否则 `--product-home <productHome>`。
- 等待 listen：10s 上限、50ms 重试。
- 退出：只停自己 spawn 的 daemon；先 `daemon.ping` 确认 `connections == 1`，再 `daemon.shutdown {force:false}`，否则保留。
- POSIX 以 detached 方式 spawn；子进程 stdout/stderr 只重定向到 `<nativeHome>/daemon.log`，**只写不读**。

## 19. Golden fixtures 现状

实际目录是 `internal/appserver/testdata/wire/`，当前只有 `notifications.ndjson`（6 条，与 `dim-electron/packages/desktop/packages/app-server-protocol/tests/fixtures/notifications.ndjson` 逐字节相同）。其余场景目前由表驱动 `headless`/`fake-server` 测试覆盖，没有独立 fixture 文件。

| 场景 | 现状 |
|---|---|
| 任意 chunk 边界、CRLF、空行 | `framing_fixture_test.go`、`framing_test.go` |
| 16 MiB 上限和 +1 拒绝 | `framing_fixture_test.go`、`framing_test.go` |
| 非 JSON、reply 同时有 result/error、notification 带 id | `client_test.go`、`framing_test.go` |
| hello 的 product / configured / mock / 反例 | `wire_test.go` |
| catalog 多页、空页、缺 `origins` 裁参 | `read_helpers_test.go` + fake server |
| summaries 裸数组和 `-32602` 只回退一次 | `read_helpers_test.go` |
| attach 有/无 `sequence` 的 head 读取 | `wire_test.go`、`session_stream_test.go` |
| attach 订阅先行、通知缓冲/路由、跨 session 过滤、未知通知透传 | `session_stream_test.go` |
| 事件 `id` 去重、`(branchId, sequence)` 冲突、closed 终态、transport loss | `session_stream_test.go` |
| detach 回复等待、release 忽略回复、closed 后 no-op | `session_stream_test.go` |
| history 分页不变量 | `wire_test.go` |
| latest-turn、`turnStart` 漂移、`history_too_large`/page-cap 降级 | `internal/session/session_test.go`（session 组合层，fake history） |
| submit 成功/未知结果 | `mutations_test.go` + fake server；含 `submission_unknown`/`submission_ambiguous` |
| stop 成功/拒绝 | `mutations_test.go` + fake server |
| approval `-32602` 映射；attach 期间 approval 到达顺序 | `mutations_test.go`、`session_stream_test.go`；完整 replay 集合替换未实现 |
| delete 成功/`session_active` | `mutations_test.go` + fake server |
| unknown_method、timeout、断线的 visible WIP/unknown-result | `mutations_test.go` |
| 每个 `appError` 的语义和可重试性 | `errors_test.go` |
| `dim-electron` 通知 fixture | `testdata/wire/notifications.ndjson` + `wire_test.go` |

fake server 是 `fake_test.go` 内的 test-only `net.Pipe` peer，服务 `hello`、`session.catalog`、`session.summaries`、`session.attach`、`session.detach`、`session.history`、`session.submit`、`session.stop`、`approval.respond` 和 `session.delete`，并支持 scripted mutation fault、gate 和主动断线；`subscription_changed` 和完整 history 分页场景仍未从 `dim-electron` 的 `fake-app-server.mjs` 派生。latest-turn 和兼容扫描由 `internal/session` 自己的 fake history 覆盖，不依赖 appserver fake server。

## 20. 验收（当前状态）

- 已通过：`go build ./...`、`go vet ./...`、`go test -count=1 ./...`（8 个包）、`go test -race -count=1 ./...`、`gofmt -l .`、`git diff --check`；
- 已通过：无 `internal/mygo` 复制、无 socket 直连 UI、无本地 session 存储；
- 部分通过：fake server 已覆盖 catalog/summaries/attach/detach/history、session stream 的订阅先行/事件冲突/closed/transport loss、四个 mutation 的拒绝与 unknown-result、`internal/session` 的 latest-turn/兼容扫描/降级/branch 隔离/terminal/close、supervisor reconnect/generation/close 竞争、integration 的 supervisor → session → viewmodel 桥和 cmd 的 opt-in 启动路径，以及 client 层错误；`subscription_changed` 恢复、`session.create` 的 UI 接线、`session.keys`/`session.branches` 的 MyGo UI 接线（公开 Desktop protocol contract 已有；session 已读取只读 snapshot，UI branch switching/cut point 未接）、`session.input_lookup` 的真实 daemon 读回和 session 层 re-attach/resync 未覆盖；
- 已通过（`real-daemon` 控制面）：在 `../dim-2` 执行 `CARGO_TARGET_DIR=/tmp/dim-2-target cargo build --locked -p dim-app-server --bin dim-app-server`，再设置 `DIMAGENT_REAL_APP_SERVER_BIN=/tmp/dim-2-target/debug/dim-app-server` 运行 `TestRealDaemonSmoke`，覆盖 spawn/hello/catalog/close；该证据不覆盖真实 session 流、projection 或 GUI。

## 21. 未决问题与 blocker

1. public protocol contract 已确认 `session.methods` 和 `session.input_lookup` 的公开形状；`session.keys` / `session.branches` 有公开 Desktop protocol contract（`session-reader.ts:1041-1074` 调用，`wire.ts:267-303` 定义 DTO），MyGo session 已 capability-gated 读取；缺的是 UI 接线和真实验收。`SessionAttachResult` 除 `sequence` 外是否必返、`session.submit` 结果是否还有其它字段，公开合同未明确；实现按"可选 + 缺失兜底"，不引用内部源码补齐。
2. `session.summaries` 返回裸数组这一形状未在 public protocol contract 中明确；fake server 只证明本地行为，若上游变化，属 wire 修正。
3. `session.history` 的 `through` 在上游是否必填、`through < after` 是否报错，未验证。
4. `session.question.respond` 的 `answers` 字段级 schema 未展开（本阶段不做）。
5. Windows named pipe 与父进程退出行为在上游文档中标注未验证。
6. 合同结论仍是 `source-read`；Wave 1/2 实现已有 `headless` + `fake-server` 证据，real-daemon 已补充控制面 smoke（spawn/hello/catalog/close），不代表真实 session 流、projection 或 GUI 通过。
7. `session.keys` / `session.branches` 的公开 Desktop protocol contract 已有；`internal/session` 已 capability-gated 读取，`Keys`/`BranchDirectory` 暴露只读 snapshot，`Branches()` 只含观察到的 branch id，UI branch switching/cut point 与真实验收仍未完成。`session.input_lookup` 已有 typed/raw helper（公开合同响应 `accepted`/`unknown`），`session.methods` helper 按公开 `{name, mutating}` 实现，但仍未接到 UI/真实 daemon 验收；session 层 re-attach/resync 未实现。`session.create` client/helper 已实现，但尚未接到 UI。supervisor 的 `Reconnect`/generation 和 `internal/session.SubmitWithID` 已落地，但仍未接到 UI。latest-turn 首屏窗口和兼容扫描已由 `internal/session` 实现，但尚未接到 UI。`approval.respond` 已有 client helper 和 `-32602` 映射，但 UI 决策入口仍是 visible-disabled WIP。
