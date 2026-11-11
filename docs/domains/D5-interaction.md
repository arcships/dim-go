# D5 交互裁决（审批、提问、长期允许、权限档位、成员停止）

Status: 第一版实现完成，待组织评审
Owner: 交互裁决领域 agent
依据: `docs/domains/R.md` 功能块 6、功能块 7；`docs/domains/A.md` 的「交互裁决」模块、「审批推进」「提问应答」「四条总规矩」；`docs/domains/D3-write.md`；`docs/domains/D2-projection.md`
范围: 审批与提问的待决集合、推进顺序、应答与长期允许、重放替换、权限档位读写与事实更新、命令规则管理、成员停止。不含界面、连接实现、协议 wire 定义、会话折叠内部、目录、账号、管理面。

---

## 1. 结果包

**领域**：D5 交互裁决

**输入**

| 输入 | 来源 | 本版使用方式 |
|---|---|---|
| 功能块 6「审批处理语义」 | `docs/domains/R.md` | 审批接收与去重、同意与拒绝、长期允许先写后答、重放集合替换 |
| 功能块 7「提问处理与权限档位」 | `docs/domains/R.md` | 一次一个、应答/取消/拒绝、结算后移除且不重答、档位读取/写入与时间线事实、成员停止 |
| 审批推进、提问应答、四条总规矩 | `docs/domains/A.md` | 两张迁移表逐条实现并在启动时校验；队列一次只推一条；不做乐观更新 |
| 写意图、长期允许与审批应答两条独立意图 | `docs/domains/D3-write.md` | 所有应答、长期规则与成员停止都经写登记；成功/失败/未知/需人工确认直接消费 |
| 审批通知 | `docs/domains/D1-connection.md` | `approval.request`/`approval.resolved`/`approval.cancelled` 映射成审批 `Fact`，重放用 `Snapshot` |
| 命令规则 | `permission.rules` / `permission.set_rules` | 读取 revision 与规则集合；新增、编辑、撤销都以带 revision 的整集重写实现；只读缓存不落盘 |
| 提问清单与工具事实 | D1 `session.question.pending`、`step/tool-use`/`tool-call`/`tool-completed`/`tool-result`；D2 折叠结果 | 清单按固定截止点装载；截止点之后的 `ask_user` 调用与工具结果驱动提问集合；**协议没有提问通知** |
| 事件词表 | `docs/domains/D2-projection.md` §3.9 | `approval/*`、`permission/changed` 归本领域；本包不读折叠内部 |
| 错误分类与连接方法 | `docs/domains/D1-connection.md` | 由集成层把通知/列表/工具事实映射成 `Fact`、`Snapshot`、`QuestionSeed`、`QuestionItem`，把 `write.Kind` 映射成 wire 方法 |

**产物**

| 路径 | 职责 |
|---|---|
| `internal/interaction/interaction.go` | 对外类型（`Fact`、`Snapshot`、`Item`、`Decision`、`Response`、`Outcome`、错误）、`New`、`Apply`/`Replay`/`OnDisconnect`/`Settle`、队列与视图 |
| `internal/interaction/state.go` | 「审批推进」「提问应答」两张迁移表与启动校验 |
| `internal/interaction/question.go` | 提问来源：清单/推导装载、`ask_user` 事实、工具结果结算、问题解析 |
| `internal/interaction/rules.go` | 命令规则：读取、新增、编辑、撤销、只读缓存与 revision 冲突 |
| `internal/interaction/respond.go` | 应答流程：队首约束、长期允许先写后答、写意图参数与幂等键、重复与不重发 |
| `internal/interaction/permission.go` | 按会话的权限档位状态；无值拒绝提交；提交参数不被污染 |
| `internal/interaction/member.go` | 成员停止意图 |
| `internal/interaction/helpers_test.go` | 写登记测试替身与事实构造 |
| `internal/interaction/interaction_test.go` | 审批重放、队列推进、结算、重复应答、断线与视图 |
| `internal/interaction/question_test.go` | 提问清单顺序、批次结算、清单失败退回推导、截止点与重复结算 |
| `internal/interaction/rules_test.go` | 规则读取、写入、编辑、撤销、冲突与能力缺失 |
| `internal/interaction/rules_real_test.go` | 真实 daemon 的规则读取/写入/编辑/撤销证据 |
| `internal/interaction/state_test.go` | 迁移表与文档逐条对应、坏表拒绝 |
| `internal/interaction/respond_test.go` | 长期允许顺序、写入失败不发应答、重试与不重发 |
| `internal/interaction/permission_test.go` | 档位取值、无值拒绝提交、按会话隔离、提交参数不被污染 |
| `internal/interaction/member_test.go` | 成员停止只停成员 |
| `internal/interaction/appserver_test.go` | `fakeappserver` + 真实写登记的请求顺序、失败与一次一条 |
| `docs/domains/D5-interaction.md` | 本文件 |

**验收**

- 命令：`go test ./...`、`go vet ./...`、`go test -race ./internal/interaction/`；本版全部通过（2026-11-11，`<platform>`）。
- 本领域：57 个顶层测试、20 个子测试通过；4 个真实环境用例按环境跳过，设置夹具后通过；覆盖见 §6.2。
- 证据档位：
  - `fakeappserver`：本机 Unix socket 上的真实 NDJSON，经 D1 连接与真实 `write.Manager` 发出长期规则、审批应答与提问应答，核对服务端收到的请求顺序与参数。
  - 注入写登记替身：覆盖成功、明确失败、需人工确认、结果未知、并发应答等分支。
  - 真实模型审批链路：已在本机 daemon + 夹具账号上验证（§6.5）；真实非空提问清单与成员停止仍 blocked（§6.6）。

**依赖**

| 依赖 | 需要对方冻结/提供 |
|---|---|
| D1 连接 | 审批请求/结算/取消、`permission/changed` 时间线事实、成员状态变化的通知形状；`session.permission.get/update`；`permission.rules/set_rules`；`session.question.pending` 清单与时间线工具事实；各写意图的方法名与参数形状；由集成层实现 `Fact`/`QuestionSeed`/`RuleSource` 映射与 `Sender` |
| D3 写登记 | `*write.Manager` 直接满足本包 `Writer`（`Register`/`Wait`）；长期允许、审批应答、提问应答各为一条独立意图；明确失败与未发生可重提，未知与需人工确认不重发 |
| D2 投影 | 不导入本包；D2 已声明 `approval/*`、`permission/changed`、`collaboration/message` 不折叠，事件路由需把这些通知转给本领域 |
| 集成层（宿主） | 每个打开的会话创建一个裁决器；权限档位随裁决器按会话一份持有；实现 `RuleSource`（把 `permission.rules/set_rules` 映射成 `RuleSet`/`CommandRule`）；把审批事实与快照交给本包；按截止点读提问清单（失败时注入推导结果），把工具事实与工具结果交给本包 |
| U1 原生界面 | 只读 `Current`/`Items`/`Approvals`/`Questions`/`Counts`，只调 `Respond` 与 `StopMember`；不得自行发送应答或推进队列 |

**上下文需求**

1. 需要 D1 明确审批/权限/成员通知的 wire 形状与提问清单/工具事实形状，本包只接受归一化 `Fact`、`QuestionSeed`、`QuestionItem`，不复制 wire。
2. 需要 D1/D2 给出「成员」在事实里的标识与停止语义，本包只按成员标识发停止意图。
3. 提问没有通知：清单读取与时间线事实由集成层提供，本包不等待、也不伪造提问通知。

**已裁决结论**

1. 领域编号：D1 连接、D2 投影、D3 写登记、D4 目录、D5 交互、D6 账号、D7 工作区、D8 附件、D9 终端、D10 宿主外壳、D11 管理面；本领域为 D5。
2. 队列作用域：按会话。每个打开的会话一条待决队列，跨会话互不影响；同一条队列内一次只推进一条；跨会话并行的最高条数等于打开的会话数。
3. 权限档位：归交互裁决，作用域按会话。读取 `session.permission.get`、写入 `session.permission.update`；变更来源是时间线 `permission/changed`，没有通知。preset 只有 `read-only`/`workspace-write`/`full-access` 三档，不本地自造枚举。未选择时不猜默认值，并拒绝提交；提交本身不携带协议没有的 permission 字段，档位由会话分支 policy 携带。
4. 提问来源：协议没有 `question.*`/`elicitation.*` 通知；提问由 `session.question.pending` 清单（固定截止点 + 不透明游标）与时间线 `ask_user` 事实构成，结算依据是工具结果/结算事实。清单优先，清单读取失败才退回历史推导。

## 2. 边界

**做**

| 能力 | 说明 |
|---|---|
| 待决集合 | 审批接受通知事实与重放；提问按清单/推导装载并接受截止点后的工具事实；同一请求标识只保留一条；视图按类型可取子集 |
| 推进队列 | 两类按到达顺序排成一条队列，一次只把队首给用户；只有队首可应答 |
| 重放替换 | `Replay` 整体替换审批集合；`SeedQuestions` 整体替换提问集合；两者都不累加、不与旧集合合并 |
| 断线 | `OnDisconnect` 清空待决，重连后以快照为准；已答记录保留，防重发 |
| 结算 | 审批用 `Settle`，提问用 `SettleQuestion`（工具结果/结算事实）；移除并推进下一条；重复结算无操作 |
| 应答 | 审批同意/拒绝；提问应答/拒绝/取消；全部经写登记 |
| 长期允许 | 先写长期规则，成功后才发审批应答；规则失败或无法确认时不发应答 |
| 重复保护 | 同一时刻只允许一次应答；已答与已结算的请求拒绝再次应答 |
| 权限档位 | 按会话保存服务端读到的 preset；从 `permission/changed` 更新；无值时拒绝提交，不猜默认值 |
| 成员停止 | 只对指定成员发停止意图，主线与待决集合不受影响 |

**不做**

| 不做 | 归属/原因 |
|---|---|
| 界面、按钮状态、弹窗 | U1 |
| 连接、socket、NDJSON、通知解码、方法名 | D1；本包只吃归一化 `Fact` |
| 写意图的发送、重试、核对与在途表 | D3；本包只登记并等终态 |
| 会话历史与工具行的折叠 | D2；本包不读折叠内部 |
| 会话目录、草稿、导出 | 目录领域与 D3 |
| 账号、模型、插件、技能、钩子、计划任务 | 账号与管理面 |
| 权限档位读写 wire | `session.permission.get`/`update` 的方法名与参数由连接/集成层持有；本包只保存 preset 并校验提交前置条件 |
| 等待提问通知 | 协议没有 `question.*`/`elicitation.*`；提问改由清单 + 时间线事实驱动 |
| 应答超时语义 | R 块 7：不引入，只提供应答、取消、拒绝 |
| 任何持久化 | R8 与 A.md：幂等与去重标识只在内存 |

本包只使用 Go 标准库与仓库内的 `internal/write`，不引第三方依赖，不改 `go.mod`。

## 3. 包结构与对外接口

```text
internal/interaction/
├── interaction.go     Fact、Snapshot、Item、Decision、Response、Outcome、错误、
│                      New、Apply、Replay、OnDisconnect、Settle、Current、Item、
│                      Items、Approvals、Questions、Counts
├── state.go           两张推进迁移表、nextState、validateTransitionTables
├── question.go        提问来源：QuestionSeed、QuestionItem、SeedQuestions、
│                      ApplyAskUser、SettleQuestion、QuestionSeedState
├── respond.go         Respond、长期允许两步、参数与幂等键、重复与不重发
├── permission.go      PermissionTier（真实三档）、Set/Clear/PermissionTier、WithPermission
├── member.go          MemberRef、StopMember
└── *_test.go          §1 产物列出的测试
```

对外接口（第一版）：

```go
arbiter, err := interaction.New(interaction.Config{
    SessionID: "s1",      // 必填：一个打开的会话一个裁决器
    Writer:    writeMgr,  // 必填：*write.Manager 直接满足 interaction.Writer
})

// 审批事实与重放
rejections := arbiter.Apply(interaction.Fact{...})                 // 审批通知
rejections = arbiter.Replay(interaction.Snapshot{Pending: [...]})  // 整体替换审批集合
settled := arbiter.Settle("approval-1")                            // 审批回执/取消

// 提问：清单装载（或清单失败后的历史推导）
rejections = arbiter.SeedQuestions(interaction.QuestionSeed{
    BranchID: "main",
    Through:  cutoff,                                  // 固定截止点
    Source:   interaction.QuestionSourceList,          // 或 QuestionSourceDerived
    Items:    []interaction.QuestionItem{...},
})
rejections = arbiter.ApplyAskUser(interaction.QuestionItem{...})   // 截止点之后的 ask_user 调用
settled = arbiter.SettleQuestion("question-1", "call-1", eventSeq) // 工具结果/结算事实
seed, ok := arbiter.QuestionSeedState()

arbiter.OnDisconnect()                                      // 断线清空待决与提问截止点

// 视图
current, ok := arbiter.Current()
item, ok := arbiter.Item("request-1")
items := arbiter.Items()
approvals := arbiter.Approvals()
questions := arbiter.Questions()
counts := arbiter.Counts()   // {Approvals, Questions}

// 应答
outcome, err := arbiter.Respond(ctx, "request-1", interaction.Response{
    Decision: interaction.DecisionApprove, // 或 Reject / AllowLongTerm / Cancel / Answer
    Answer:   json.RawMessage(`{"choice":["继续"]}`), // DecisionAnswer 时必填
})

// 成员停止
view, err := arbiter.StopMember(interaction.MemberRef{MemberID: "m1", BranchID: "b1"})

// 权限档位（每个会话一份，随裁决器持有）
err = arbiter.SetPermissionTier(interaction.PermissionWorkspaceWrite)
tier, ok := arbiter.PermissionTier()
arbiter.ClearPermissionTier()
intent, err := arbiter.WithPermission(write.Intent{
    Kind: write.KindSubmitInput, SessionID: "s1", ...
})

// 命令规则（RuleSource 由集成层实现，wire 不进本包）
set, err := arbiter.LoadRules(ctx)          // 读取 revision 与规则集合
rules, ok := arbiter.Rules()                // 只读缓存拷贝
set, err = arbiter.AddRule(ctx, interaction.CommandRule{
    Command: "printf ok", Match: interaction.CommandMatchExact,
    Decision: interaction.CommandDecisionAllow,
})
set, err = arbiter.EditRule(ctx, 0, rule)   // 以整集重写实现编辑
set, err = arbiter.RevokeRule(ctx, 0)       // 以整集重写实现撤销
unavailable := arbiter.RulesUnavailable()   // 能力缺失后不再发请求
```

`Item` 只含业务可观察字段：类型、请求标识、会话/分支/轮次、动作描述、问题与候选项、提问来源与调用清单、可长期允许标记与可用性、推进状态、已发出的决定、队列位置。不含长期规则原文、幂等键、写意图标识等内部字段。

## 4. 设计

### 4.1 事实输入与归一化

审批与提问走两条来源，各自只有一个驱动者：

| 来源 | 输入 | 归一化 |
|---|---|---|
| 审批 | `approval.request`/`approval.resolved`/`approval.cancelled` 通知，以及重放快照 | `Fact{Kind: PendingApproval, ...}`，含请求标识、会话/分支、动作描述、可长期允许标记与长期规则参数 |
| 提问 | `session.question.pending` 清单（或清单失败后的历史推导） | `QuestionSeed{BranchID, Through, Source, Items}`，每条 `QuestionItem` 对应一个 `ask_user` 调用 |
| 提问（截止点之后） | `step/tool-use`/`tool-call` 里的 `ask_user` 调用 | `QuestionItem{RequestID, CallID, EventID, Sequence, ToolName, Input}` |
| 提问（结算） | `step/tool-completed`/`tool-result` | `SettleQuestion(requestID, callID, sequence)` |

约定：

1. 审批：类型必须是审批、请求标识与会话标识非空且会话匹配、长期规则参数若非空必须是合法 JSON。
2. 提问：请求标识与调用标识非空、工具名是 `ask_user`（大小写与下划线不敏感）、`Input` 能解析出至少一个问题。
3. 提问事实不能从审批通道（`Apply`/`Replay`）进入；违反时按 `RejectionQuestionSource` 丢弃，避免出现第二个驱动者。
4. 不合格事实丢弃并通过 `Rejection` 带原因（`invalid_fact`、`session_mismatch`、`duplicate_request`、`question_source`、`below_cutoff`、`already_settled`）；本包不写诊断日志，怎么记录由调用方决定。

### 4.2 两个集合与一条队列

- 审批与提问是两类待决项：决定集合、可长期允许、结算语义都不同，视图分别用 `Approvals()`/`Questions()` 取。
- 推进只有一条队列：两类按到达顺序入队，`Current()` 只返回队首。这样满足 A.md「后到的不越前」与「一次只把队首给用户」，也不会出现审批与提问各自弹一个、用户同时面对两条。
- `Counts()` 统计两类条数，已答但尚未结算的项仍计入（服务端仍视其为待决）。
- 请求标识按审批与提问共用一把键：服务端标识跨类型唯一；同一标识第二次出现按重复丢弃。
- 队列作用域按会话：每个打开的会话一条待决队列，跨会话互不影响；同一条队列内一次只推进一条。跨会话并行的最高条数等于打开的会话数。

### 4.3 审批重放、提问装载、断线与结算

**审批重放**：`Replay` 用快照的 `Pending` 整体替换审批集合，旧审批里不在快照中的项一律消失，快照里的审批按给出顺序排在新提问之前；不累加、不与旧集合去重合并。快照里的提问事实按 `RejectionQuestionSource` 丢弃；提问集合由 `SeedQuestions` 单独整体替换，审批重放不动它。

**提问装载**：`SeedQuestions` 用清单（`QuestionSourceList`）或历史推导（`QuestionSourceDerived`）整体替换提问集合，并记录固定截止点 `Through`、分支与来源：

1. **清单优先**：集成层先按截止点读 `session.question.pending`（分页由不透明游标完成，本包只接收汇总后的条目）；清单成功即装载 `list`。
2. **读取失败才退回推导**：清单读取失败时，集成层用 D2 已折叠出的待答提问（历史里的 `ask_user` 调用减去工具结果）装载 `derived`，并把分支当前头部作为截止点。
3. 同一请求标识的多个 `ask_user` 调用合成一批提问：一个条目、按调用顺序汇总问题；作答仍以请求标识为单位，结算按调用进行，全部调用结算后整批移除。
4. 装载顺序就是清单/推导给出的顺序；本包不按序号重排。
5. 截止点以下的工具事实不改动提问集合：清单已经反映过那个时刻的未决状态；只有序号晚于截止点的 `ask_user` 调用（开）与工具结果（结算）才生效。
6. 装载会重置「已结算调用」记录；已答记录（`answered`）保留，因此清单里仍待决、但本端已应答过的提问显示为「已答」，不再接受应答。

**断线**：`OnDisconnect` 清空审批与提问，作废提问截止点与已结算调用记录；重连后审批由 `Replay`、提问由清单（失败则推导）重建。已答记录与规则已写入记录保留，避免重连后重复应答或重写规则。

**结算**：

- 审批用 `Settle(requestID)` 应用服务端回执/取消；提问用 `SettleQuestion(requestID, callID, sequence)` 应用工具结果/结算事实。
- 重复结算按无操作处理；序号不晚于截止点的结算不改动清单。
- 应答写入成功**不会**自行移除条目——A.md 要求「应答发出且服务端结算」才到无待决。

### 4.4 两张推进状态机与启动校验

逐条对应 A.md，并补两条服务端提前结算的出边（审批被取消、提问被取消）：

| 审批推进 | 事件 |
|---|---|
| 无待决 → 待呈现 | 审批事件或重放快照 |
| 待呈现 → 待应答 | 用户选同意或拒绝 |
| 待呈现 → 待写长期 | 用户选长期允许 |
| 待写长期 → 待应答 | 长期允许写入成功 |
| 待写长期 → 待呈现 | 长期允许写入失败或无法确认（且不发应答） |
| 待应答 → 无待决 | 应答发出且服务端结算 |
| 待呈现/待写长期 → 无待决 | 服务端在作答前结算（补充出边） |

| 提问应答 | 事件 |
|---|---|
| 无待决 → 待答 | 清单/推导装载，或截止点之后的 `ask_user` 调用（只取队首） |
| 待答 → 已答 | 用户应答、取消或拒绝 |
| 已答 → 无待决 | 工具结果/结算事实（批次最后一个调用结算后推进） |
| 待答 → 无待决 | 工具结果/结算事实在作答前结算（补充出边） |

约定：

- 状态名：`StatePresented`（待呈现/待答）、`StateAwaitingLongTerm`、`StateAwaitingResponse`、`StateAnswered`；条目不存在即「无待决」。
- 写入明确失败时视为事件没有发生：审批留在/回到待呈现，提问留在待答；不做表外迁移。
- 提问没有通知：进入「待答」只由清单/推导装载与截止点后的 `ask_user` 事实驱动，结算只由工具结果/结算事实驱动。批次里部分调用结算不改变条目状态，最后一个调用结算才移除。
- 非法迁移丢弃事件、状态不变、不向用户报错（A.md 统一处理）。
- `New` 启动时校验两张表：来源与目标在状态集合内、事件在事件集合内、`无待决` 只能作为入口、本表可达的每个状态都有服务端结算出边；不合法即失败并点名状态机与条目。

「提问已答」但写入明确失败的处理：A.md 的表没有失败出边。本实现把「已答」定义在**写登记受理之后**，因此失败时提问仍是待答、可以重试；这样既守住「不做乐观更新」，也不会把一条没发出去的应答永久标成已答（见 §7 限制 7）。

### 4.5 应答与写意图

| 决定 | 适用 | 写意图 `Kind` | 参数（透明） |
|---|---|---|---|
| `DecisionApprove` | 审批 | `KindApprovalResponse` | `{"decision":"approve"}` |
| `DecisionReject` | 审批、提问 | `KindApprovalResponse` / `KindQuestionAnswer` | `{"decision":"reject"}` |
| `DecisionAllowLongTerm` | 审批（可长期允许且有规则参数） | 先 `KindLongTermAllow`，后 `KindApprovalResponse` | 规则：`{"command":<简单命令>,"match":"exact","decision":"allow"}`；应答：`{"decision":"allow"}` |
| `DecisionAnswer` | 提问 | `KindQuestionAnswer` | `{"decision":"answer","answer":<原文>}` |
| `DecisionCancel` | 提问 | `KindQuestionAnswer` | `{"decision":"cancel"}` |

- `SessionID` 为本裁决器的会话；`BranchID` 审批取自事实、提问取自清单分支。`InputID` 取审批或提问的请求标识，供写登记核对与发送方引用。
- 每次尝试生成新的幂等键：`interaction/<step>/<session>/<requestID>/<decision>/<attempt>`；只有上一次写入明确失败后才会产生下一次尝试，因此新键不会掩盖重复发送。
- 发送方（集成层）负责把 `Kind` 映射成 wire 方法，并合入会话、分支与请求标识；本包不复制 wire 定义。审批应答的 `approve`/`reject` 在集成层映射成服务端的 `allow`/`deny`。
- 服务端没有 `approval.allow_always`：长期允许由 `permission.rules` 读、`permission.set_rules` 写用户命令规则承载，规则写成功后才发 `approval.respond {decision:"allow"}`。

### 4.6 长期允许：先写后答

1. 校验：服务端声明可长期允许，且事实带长期规则参数；否则 `ErrLongTermUnavailable`，不写任何意图。
2. 状态进入「待写长期」，登记长期规则意图并等待终态。
3. 只有 `StatusSucceeded` 才算写入成功；**明确失败、未发生、需人工确认、登记失败或等待被取消都按未成功处理**：回到待呈现、不发应答、条目留在待决集合（`ErrLongTermWriteFailed`）。
4. 规则成功后记录「规则已写入」，进入待应答，再登记审批应答并等待终态。
5. 若应答写入明确失败：条目停在待应答、允许**只重试应答**，不重写规则，也不允许改口成拒绝；重试用新幂等键。规则已写入的记录跨重放与断线保留，重连后仍只重试应答。
6. 若应答写入成功、需人工确认或结果未收敛：按已答处理，不重发。

「先写后答」在 `fakeappserver` 上核对的是服务端收到的请求顺序：`permission.rules` → `permission.set_rules` → `approval.respond`；规则失败时 `approval.respond` 的请求数为 0。规则只覆盖 `exec`/`execute` 的简单命令（不含分号、`&`、`|`、反引号、括号、重定向、`$` 与换行），其他工具暂不支持长期允许。

### 4.7 重复应答与不重发

| 情况 | 处理 |
|---|---|
| 不是队首 | `ErrNotCurrent`，不写任何意图 |
| 已答（含已结算） | `ErrAlreadyAnswered`，不写 |
| 同一项正在写应答 | `ErrResponseInProgress`，不写 |
| 应答成功 / 需人工确认 | 记入已答记录；不重发 |
| 等待结束但结果未收敛（在途/未知） | 按已答处理，`ErrResponseUnsettled`；不重发 |
| 明确失败 / 未发生 | 不记已答；允许重试（新幂等键） |
| 审批结算 / 提问工具结果 | 移除并推进；已答记录保留，之后仍拒绝应答 |

### 4.8 权限档位：状态、读写与不变量

**状态**：每个打开的会话在自己的裁决器里持有一个 preset，可以没有值。preset 只有服务端的三档：`read-only`、`workspace-write`、`full-access`；本包不定义别名、不猜默认值。版本号由集成层随会话持有（打开时读到、写入回执更新），交互裁决只持有 preset。

**读写与变更来源**

| 动作 | 服务端事实 | 本领域/集成层 |
|---|---|---|
| 打开会话时读取 | `session.permission.get {sessionId,branchId}` → `{version,policy}` | 集成层装入 preset 与版本 |
| 用户切换档位 | `session.permission.update {sessionId,branchId,expectedVersion,policy}` → `{version,policy}` | 集成层写入；成功以回执更新 preset 与版本 |
| 其他客户端改动 | 时间线 `permission/changed`（`event.type=permission`、`event.name=changed`、`data.policy`） | 集成层把 preset 装入本领域；**没有 `permission.changed` 通知，不订阅、不轮询** |
| 提交 | `session.submit` | 提交前要求本会话已有 preset；协议没有提交 permission 字段，档位由分支 policy 携带 |

**不变量**

1. **无值不猜默认**：没有读到/选择过就没有值，`PermissionTier()` 返回 `false`；不得补任何本地默认档位。
2. **无值拒绝提交**：`WithPermission` 在本会话没有 preset 时返回 `ErrPermissionUnset`，提交路径必须先经过它；有值时只校验、不向提交参数添加字段。
3. **只作用于本会话**：`WithPermission` 只接受 `KindSubmitInput`，且意图的 `SessionID` 必须等于本裁决器的会话；别的类型报 `ErrNotSubmitIntent`，别的会话报 `ErrPermissionSession`。
4. **真实三档、原样保存**：`SetPermissionTier` 只要求非空，preset 原样保存；三档以服务端取值为准，不本地自造枚举。
5. **版本冲突不覆盖**：集成层写入带 `expectedVersion`；服务端回 `permission_conflict` 时返回 `ErrPermissionConflict`，本地 preset 与版本都不变，由调用方重读后再写。
6. **会话隔离**：两个会话的 preset 与版本互不影响；清空一个会话的档位不改变另一个。

**协议事实（alpha.21 实测与源码）**

- `session.submit` 的参数没有 permission 字段；带该字段会被服务端按 `deny_unknown_fields` 回 `-32602 invalid_params`。因此本包不向提交参数注入 `permission`，只用 `WithPermission` 做“必须已选”的前置校验；真正生效的是会话分支的 policy。
- `permission/changed` 的 data 只有 `policy`，没有版本号。集成层更新 preset；版本号不伪造，下一次 get/update 以服务端返回的 version 为准；若期间被其他客户端改过，update 会以 `permission_conflict` 显式失败，不覆盖。
- 权限档位的读写方法名与参数形状只出现在集成层，本包不复制 wire。

### 4.9 成员停止

- `StopMember(MemberRef{MemberID, BranchID})` 登记一条 `KindStop` 意图，参数 `{"memberId":"..."}`，`SessionID` 为本裁决器会话，`BranchID` 取成员所在分支。
- 幂等键固定为 `interaction/member-stop/<session>/<branch>/<member>`：重复点击返回同一条意图，不重复发送。
- 停止作用范围由发送方按 `memberId` 收窄到该成员；本包不发会话级停止，也不改待决集合与主线运行状态。主线是否继续由服务端事实与 D2 的读取决定。

### 4.10 并发与锁

- `Arbiter` 用一把互斥锁串行化事实、视图与状态；写登记调用（`Register`/`Wait`）在锁外进行，避免阻塞其它操作。
- `inProgress` 在锁内占位、写完后释放：同一请求同一时刻只允许一次应答跨过校验。
- 重放或结算可能发生在一次应答写入期间：写入完成后按请求标识重新查找条目，找不到就只记已答记录，不复活条目；条目被替换时把结果落到新条目上。

### 4.11 命令规则管理

**来源与形状**

- 读：`permission.rules` → `{revision, commandRules:[{command, match, decision}]}`；`revision` 是不透明字符串，规则集合是用户级、跨会话。
- 写：`permission.set_rules {expectedRevision, commandRules}`；新增、编辑、撤销都是**重写整个集合**，不是增量操作。
- `match` 只有 `exact`/`prefix`，`decision` 只有 `allow`/`deny`；命令必须非空且不含服务端定义的分隔符（`;&|` 反引号、括号、重定向、`$`、换行）。
- wire 形状只由集成层实现 `RuleSource`；本包只消费 `RuleSet` 与 `CommandRule`。

**缓存**

- `RuleSource.ReadRules` 的结果写入内存只读缓存；`Rules()` 返回深拷贝，调用方改不动缓存；不落盘。
- 缓存按 `revision` 更新：revision 相同保留原缓存，revision 变化整体替换。
- 成功写入后用回执 revision 更新缓存；真实 `permission.set_rules` 回执可能只带 revision，此时规则集合以本次提交的集合为准（写入语义就是整集替换）。
- 读取或写入返回 `ErrRulesUnavailable` 后标记能力缺失；此后的读取/新增/编辑/撤销都直接报 `ErrRulesUnavailable`，不再发任何请求。
- 写入返回 `ErrRulesConflict` 时返回错误、**不覆盖缓存、不自动重试**；调用方先 `LoadRules` 再决定是否重写。

**新增、编辑、撤销**

| 操作 | 语义 |
|---|---|
| `AddRule` | 缓存末尾追加；完全相同的规则已存在时直接返回缓存，不写 |
| `EditRule(index, rule)` | 用新规则替换第 index 条；index 越界报 `ErrRuleIndex` |
| `RevokeRule(index)` | 删除第 index 条；允许删到空集合；越界报 `ErrRuleIndex` |

三者都先用缓存里的 revision 作 `expectedRevision`，写成功后更新缓存；校验失败（空命令、分隔符、未知 match/decision）在发请求前返回 `ErrRuleInvalid`。

### 4.12 审批待答期间的提交

**结论：允许。** 审批待答不阻塞 `session.submit`；`mode:"queue"` 与 `mode:"steer"` 都会被服务端接受，不需要先结算审批。两者的差别在消费方式：

| 模式 | 待答期间 | 审批同意后 |
|---|---|---|
| `queue` | 接受；输入进入队列（`input/pending` 出现，`applied=false`） | 当前轮结束后被消费；实测轮次数 1→2（开新轮），队列回到 0 |
| `steer` | 接受；输入进入队列（`input/pending` 出现，`applied=false`） | 被应用进当前轮；实测轮次数保持 1，队列回到 0 |

条件与边界：

- 审批所属轮仍在进行时实测成立；提交带合法的会话、分支与 mode。
- 提交不会改变待决审批：审批条数仍为 1，当前审批项不被替换。
- 两种模式都先进入输入队列；`queue` 等当前轮结束再开下一轮，`steer` 不新开轮。
- 没有观察到错误码；本场景不是“能力缺失”或“需要先审批才能输入”。

对界面的影响：审批面板出现时输入框可以保持可用，并允许用户选择排队/插话；界面不能把“有待决审批”当作禁止提交的条件，也不应把提交当作对审批的回答。提交后应显示队列条数/模式；审批结算后按队列与当前轮的消费结果显示，而不是本地推断。

### 4.13 始终允许：规则形状、匹配与作用域

**服务端形状**

- 读：`permission.rules {}` → `{revision, commandRules:[{command, match, decision}]}`。
- 写：`permission.set_rules {expectedRevision, commandRules}`，整集重写；回执只带新的 `revision`。
- `match` 只有两个取值：`exact`、`prefix`；服务端枚举不接受 `glob`、`regex` 或其他值。
- `decision` 只有 `allow`、`deny`。
- `command` 必须非空且不含服务端定义的分隔符（分号、`&`、`|`、反引号、括号、重定向、`$`、换行）；规则本身不能是一条管道/复合命令。

**匹配语义**

- 命令按空白切成词（不是 shell 解析，不处理引号/转义）；`exact` 要求词序列完全相同，`prefix` 要求命令以规则的词序列开头。
- 精确匹配包含参数：多一个参数就不匹配。
- 前缀匹配按整词前缀：规则 `printf ok` 可以匹配 `printf ok extra`，但不会匹配 `printf okay`。
- 允许规则只在命令**不含**分隔符时参与判定；带管道的命令不会被 allow 规则自动放行，仍会触发审批。
- 拒绝规则在判定更早的位置生效：命令按分隔符切成段，任意一段匹配 deny 规则即拒绝；deny 在三个 preset 下都生效。
- allow 规则只把“本会询问（Ask）”改成允许，不会覆盖 preset 的拒绝（例如 read-only 下启动进程仍被拒绝）。

**作用域与修改权**

- 规则是**用户级**设置（`permission` source、user scope），全局适用所有会话与分支；不是会话级、分支级。
- 项目/会话配置不能授予规则；没有项目级规则。
- 连接 daemon 的客户端可调用 `permission.set_rules`；方法参数里没有 sessionId/branchId。本机 socket 就是访问边界。
- 撤销没有单条删除方法：读当前集合、去掉目标、带 `expectedRevision` 重写；冲突不覆盖。
- 审批请求本身不带规则作用域/规则标识；服务端只在 deny 的 `reason` 里带“由哪条规则拒绝”（含 command 与 match）。

**审批卡可展示的字段**

| 来源 | 可展示 |
|---|---|
| `approval.request` | `source.kind`、`call.name`、`call.input`（可显示工具与命令）、`reason` |
| 审批卡上的作用域说明 | 服务端没有 scope/revision/ruleId 字段；“写入用户级命令规则、所有会话生效”只能是客户端按 API 语义做的说明 |
| `permission.rules` | `revision`、完整规则集合，可做规则管理视图 |
| `permission.set_rules` | 只有 `revision`；不返回规则集 |
| deny 的 `reason` | 规则命令与 match，可用于解释拒绝原因 |

**默认该用哪种**

默认 `exact`。事实依据：prefix 会连同后续参数一起放行，范围更宽；exact 只在命令词序列完全一致时生效。需要覆盖不同参数时，必须由用户显式选择 prefix 并看到“所有会话生效”的说明，不能由客户端根据一次审批自动推断。

## 5. 与 A/R 的对应

| 需求或架构条目 | 本领域实现 | 不在本领域 |
|---|---|---|
| R 块 6 审批接收与去重 | 按请求标识去重；重复事实丢弃 | 通知解码（D1） |
| R 块 6 同意与拒绝 | `Respond` + `KindApprovalResponse` | 工具执行结果（D2） |
| R 块 6 长期允许先写后答、写入失败不发应答 | 两步流程、成功门槛、待呈现回退 | 长期规则的业务内容与真实方法（D1） |
| R 块 6 重放集合替换、条数一致 | `Replay` 整体替换、快照内去重 | 快照的来源（D1/D2） |
| R 块 7 两个提问只答一个、结算后才答下一个 | 一条队列、队首约束、`SettleQuestion` 推进 | 问题候选项解释（界面） |
| R 块 7 提问来源与结算 | 清单/推导装载、截止点后的 `ask_user` 调用、工具结果结算 | 清单读取与分页（D1/集成层） |
| R 块 7 结算后移除且不再应答 | `SettleQuestion` 移除；已答记录拒绝重发 | 工具结果/结算事实（D1/D2） |
| R 块 7 权限档位取值与传递 | 按会话保存 preset；`WithPermission` 校验提交前置条件；读写方法由集成层执行 | `session.permission.get/update` 的 wire（集成层/D1） |
| R 块 7 成员停止只停成员 | `StopMember` 带 `memberId` 的停止意图 | 成员运行状态读取（D2/D1） |
| R 块 3/6 审批待答期间的输入 | 允许 `queue`/`steer`；不替换审批；queue 开新轮、steer 进当前轮 | 输入正文与轮次消费（D3/D2） |
| R 块 6 始终允许的作用域 | 用户级命令规则；exact/prefix；所有会话；deny 全 preset，allow 仅 Ask→Allow | 规则匹配 wire（集成层），审批卡无 scope 字段 |
| R 块 6/7 命令规则管理 | `LoadRules`/`AddRule`/`EditRule`/`RevokeRule`；整集重写、revision 冲突不覆盖 | `permission.rules/set_rules` wire（集成层） |
| A 审批推进 / 提问应答 | `state.go` 两张表、启动校验 | — |
| A 「审批集合、提问集合」镜像状态 | 审批只由 `Apply`/`Replay`/`Settle` 改；提问只由 `SeedQuestions`/`ApplyAskUser`/`SettleQuestion` 改；各自整体替换 | — |
| A 「审批推进到哪一条」「提问推进到哪一条」 | 队列与 `Position`，只由本包推进 | — |
| A 四条总规矩 | 镜像状态只由事实驱动；只提交写意图、不写事实；状态只有一个驱动者；应答受理前不显示为已答 | 投影与界面 |
| A 检查项「重放后待决条数不多于服务端」「已结算的审批不再应答」 | 重放替换与已答记录 | — |

## 6. 验收

### 6.1 命令与结果

```bash
go test ./...
go vet ./...
go test -race ./internal/interaction/
```

本版结果（2026-11-11，本机 `<platform>`）：三条命令全部通过；`internal/interaction` 57 个顶层测试、20 个子测试全绿；`TestRealModelApproval`、`TestRealCommandRules`、`TestRealSubmitDuringApproval` 与 `TestRealRuleMatching` 未设置夹具时跳过，设置后通过（§6.5）。

### 6.2 覆盖

| 验收要求 | 测试 |
|---|---|
| 审批重放整体替换，不累加、不合并；提问集合不被重放改动 | `TestReplayReplacesApprovalSetKeepsQuestions` |
| 重放按快照顺序建审批队列，队首是快照第一条 | `TestReplayReplacesApprovalSetKeepsQuestions` |
| 快照里的坏事实、重复标识与提问事实被丢弃并带原因 | `TestReplayRejectsBadAndDuplicateFacts` |
| 坏审批事实不进入集合；重复事实不累加；提问事实不能走审批通道 | `TestApplyRejectsBadFacts` |
| 提问清单注入后按清单顺序呈现 | `TestQuestionSeedPresentsInListOrder` |
| 同一请求的多个调用合批、逐个结算、全部结算后移除 | `TestQuestionSeedGroupsBatchCalls` |
| 工具结果结算后移除并推进 | `TestQuestionSettlementByToolResultAdvances` |
| 重复结算不重复应答、条目不再出现 | `TestDuplicateSettlementDoesNotReanswer` |
| 清单读取失败退回历史推导，之后工具事实继续生效 | `TestQuestionListFailureFallsBackToFacts` |
| 截止点及以下的调用与结算不改动清单 | `TestQuestionBelowCutoffFactsAreIgnored` |
| 已结算调用不被旧事实装回来 | `TestSettledCallIsNotReadded` |
| 清单坏项丢弃、装载参数校验、重复调用去重 | `TestQuestionSeedRejectsInvalidItems`、`TestSeedQuestionsRequiresSourceAndBranch`、`TestApplyAskUserRejectsDuplicateCall` |
| 一次只推一条；非队首应答被拒 | `TestQueueAdvancesOneAtATime` |
| 两个提问只应答一个，结算后才发第二个 | `TestQueueAdvancesOneAtATime`、`TestAppServerQuestionsOneAtATime` |
| 应答成功不自行移除；结算后才推进 | `TestQueueAdvancesOneAtATime`、`TestAppServerLongTermWritesRuleBeforeResponse` |
| 结算后移除且不再应答；重复结算无操作 | `TestSettlementBeforeAnswerRemovesItem`、`TestQueueAdvancesOneAtATime`、`TestDuplicateSettlementDoesNotReanswer` |
| 已答但未结算的项重新装载后仍显示已答、不重答 | `TestAnsweredItemStaysAnsweredAcrossSeed` |
| 断线清空待决，重连后审批重放、提问清单重建 | `TestDisconnectClearsUntilReload` |
| 同一待决项同一时刻只允许一次应答 | `TestDuplicateResponseWhileWritingIsRejected` |
| 已答/已结算的重复应答被拒，且不产生新意图 | `TestQueueAdvancesOneAtATime`、`TestAppServerLongTermWritesRuleBeforeResponse` |
| 长期允许先写规则、后发应答（服务端请求顺序） | `TestLongTermWritesRuleBeforeResponse`、`TestAppServerLongTermWritesRuleBeforeResponse` |
| 长期规则写入失败/需人工确认/登记失败时不发应答、保留待决 | `TestLongTermRuleFailureSendsNoResponse`、`TestAppServerLongTermFailureSendsNoResponse` |
| 没有规则参数或服务端不允许时不写任何意图 | `TestLongTermRuleUnavailable` |
| 长期规则已写入后应答失败：只重试应答、不改口 | `TestLongTermResponseFailureRetriesOnlyResponse` |
| 规则已写入后重放：停在待应答、不重写规则 | `TestRuleWrittenStateSurvivesReplay` |
| 普通应答明确失败可重试，且换新幂等键 | `TestResponseFailureAllowsRetry` |
| 需人工确认按已发出处理、不重发 | `TestManualConfirmationCountsAsAnswered` |
| 等待未收敛按已发出处理、不重发 | `TestUnsettledResponseIsAnsweredNotResent` |
| 提问应答需要合法内容；类型与决定匹配 | `TestQuestionAnswerValidation`、`TestDecisionMatrix` |
| 提问应答与取消的参数形状 | `TestQuestionAnswerAndCancelParams` |
| 审批应答参数与请求标识 | `TestLongTermWritesRuleBeforeResponse`、`TestAppServerLongTermWritesRuleBeforeResponse` |
| 未知请求被拒 | `TestRespondUnknownRequest` |
| 视图是拷贝，不暴露内部可写引用 | `TestViewsAreCopies` |
| 两张迁移表与 A.md 逐条对应 | `TestTransitionTablesMatchDocument` |
| 迁移表启动校验，坏表点名状态机并失败 | `TestTransitionTablesAreValid`、`TestBrokenTransitionTablesAreRejected`、`TestNewRequiresConfig` |
| 权限档位无值时不猜默认值，并拒绝提交 | `TestPermissionHasNoDefault` |
| 提交不携带协议没有的 permission 字段，原参数不被改写 | `TestPermissionKeepsSubmitParams` |
| 档位取值原样保存、空值被拒 | `TestPermissionValuesAreOpaque`、`TestPermissionRejectsEmptyTier` |
| 档位只作用于提交意图，且只作用于本会话 | `TestPermissionScopeMatchesSession` |
| 每个会话各有一份档位、互不影响 | `TestPermissionIsPerSession` |
| 清空档位回到没有值，提交再次被拒 | `TestPermissionClear` |
| 服务端读取、带版本写入、工具列表保留、时间线 `permission/changed` 驱动同一状态 | `TestPermissionGetUpdateAndTimeline`（`internal/app`） |
| 版本冲突按冲突处理、不覆盖本地档位 | `TestPermissionUpdateConflictIsNotOverwritten`（`internal/app`） |
| 规则读取按 revision 缓存，视图是拷贝 | `TestLoadRulesCachesByRevisionAndReturnsCopies` |
| 新增、编辑、撤销都以整集重写，带上一 revision | `TestAddEditRevokeRewriteRuleSet` |
| 规则写入冲突返回错误、不覆盖、不自动重试 | `TestRulesConflictDoesNotOverwrite` |
| 能力缺失后零请求 | `TestRulesUnavailableMakesZeroRequests` |
| 坏规则与越界下标在发请求前被拒 | `TestRuleValidationAndIndex` |
| 真实 set_rules 回执只带 revision 时使用提交集合 | `TestWriteReplyWithoutRulesUsesSentSet` |
| 规则管理不经过写登记 | `TestRulesDoNotUseWriteRegistry` |
| 真实 daemon 的规则读取、新增、编辑、撤销 | `TestRealCommandRules`（§6.5） |
| 规则 exact/prefix、带参数、管道与 deny 的真实匹配行为 | `TestRealRuleMatching`（§6.5） |
| 审批待答期间 `queue`/`steer` 均被接受，审批不被替换 | `TestRealSubmitDuringApproval/queue`、`/steer`（§6.5） |
| 审批结算后 queue 开新轮、steer 进当前轮 | `TestRealSubmitDuringApproval/queue`、`/steer`（§6.5） |
| 成员停止只停成员，主线与待决集合不变 | `TestStopMemberOnlyStopsMember` |
| 成员停止幂等，缺成员标识被拒 | `TestStopMemberIsIdempotent`、`TestStopMemberRequiresMember` |

证据档位：`fakeappserver`（本机 Unix socket + NDJSON + 真实 `write.Manager`）三个用例；写登记替身覆盖其余分支；真实模型审批链路见 §6.5。

### 6.3 与 D3/D2/D1 的接口形状

| 提供方 | 形状 | 本包消费方式 |
|---|---|---|
| D3 写登记 | `*write.Manager` 的 `Register(write.Intent) (write.IntentView, error)`、`Wait(ctx, id) (write.IntentView, error)` | `Writer` 接口直接由 `*write.Manager` 满足；集成层可直接传 |
| D3 写意图 | `KindApprovalResponse`、`KindLongTermAllow`、`KindQuestionAnswer`、`KindStop`；`Intent.Params` 透明；`InputID` 核对 | 长期允许与应答各登记一条；参数形状见 §4.5；不解释回执内容 |
| D3 写状态 | `Succeeded`/`Failed`/`NotOccurred`/`NeedsManualConfirmation`/`Unknown` | 成功与需人工确认按已发出；失败与未发生允许重试；未知按不重发 |
| D3 幂等 | 相同键返回同一意图；失败可换新键重提 | 每次尝试生成新键；重复应答在本包先被拒绝 |
| D1 通知与事实 | 审批三类通知、`permission/changed`、成员通知；提问没有通知，改用 `session.question.pending` 清单与时间线 `ask_user`/工具结果 | 由集成层映射为 `Fact`、`Snapshot`、`QuestionSeed`、`QuestionItem`；本包不 import 连接包、不复制 wire |
| D1 发送 | `protocol.Client.CallInto(ctx, method, params, out)` | 集成层实现 `write.Sender`：按 `Kind` 选方法，合入会话、分支、`requestId`，只发一次 |
| D2 投影 | 事件词表已声明 `approval/*`、`permission/changed` 不归投影；提问清单在投影里有推导结果 | 事件路由把审批与工具事实转给本包；清单读取失败时集成层把投影的待答提问转成推导 `QuestionSeed`；本包不 import 投影包 |
| 集成层规则 | `RuleSource.ReadRules(ctx)` / `WriteRules(ctx, expectedRevision, rules)` | 把 `permission.rules/set_rules` 映射成 `RuleSet`/`CommandRule`；能力缺失映射 `ErrRulesUnavailable`，版冲突映射 `ErrRulesConflict` |

本包只 import `internal/write`，与 D1/D2 无代码依赖；两侧可独立演进，集成层负责拼接。

### 6.4 fakeappserver 证据

三个用例都在本机 Unix socket 上走真实 NDJSON：`fakeappserver` 提供端点，`daemon` 连接并握手，`write.Manager` 作为真实写登记，`interaction.Arbiter` 只通过 `Writer` 使用它。

| 用例 | 观察到的服务端请求 |
|---|---|
| `TestAppServerLongTermWritesRuleBeforeResponse` | `permission.rules` 读 revision，`permission.set_rules` 追加 `{command:"printf DIMAGENT_APPROVAL_OK",match:"exact",decision:"allow"}`，随后 `approval.respond {decision:"allow"}`；结算后应答请求数仍为 1 |
| `TestAppServerLongTermFailureSendsNoResponse` | `permission.rules` 回能力缺失；`approval.respond` 请求数为 0；审批仍在待呈现 |
| `TestAppServerQuestionsOneAtATime` | 提问由清单装载；第二条越前应答产生 0 个请求；第一条应答后按工具结果结算，第二条才发；两个 `question.respond` 的 `requestId` 分别为 `q1`、`q2` |

### 6.5 真实模型审批证据

环境：`<platform>`；夹具 `<fixture>`、daemon `<server>`（`0.1.0-alpha.21`，managed provider，`deepseek-v4.1-flash`，账号已登录）；夹具只复制到新的临时目录，来源未修改、未提交。复制件把 `config.json` 的 `toolApproval` 改为 `ask`、`permission` 改为 `workspace-write`；会话打开后再用 `session.permission.update` 固定为 `workspace-write`。

```bash
DIM_REAL_MODEL_FIXTURE=<fixture> DIM_REAL_MODEL_SERVER=<server> \
  go test -run TestRealModelApproval -v -count=1 -timeout 12m ./internal/interaction/
```

提示词：

```text
Use the shell tool to run exactly: printf DIMAGENT_APPROVAL_OK. You must call the shell tool; do not answer from memory. Then report the tool output.
```

观察到的序列（2026-11-11；只记录类型、请求标识、决定与布尔结果，未打印凭据或会话正文）：

| 场景 | 观察 |
|---|---|
| 同意 | 新建会话 → 提交 → `approval.request` 一条（`pendingApprovals=1`，动作是 `exec`）→ 决定 `allow` → 工具结果含 `DIMAGENT_APPROVAL_OK`，`toolResult=true` |
| 拒绝 | 新建会话 → 提交 → `approval.request` 一条 → 决定 `deny` → 无成功执行结果，`toolResult=false` |
| 长期允许 | 新建会话 → `approval.request`（`allowLongTerm=true`）→ `permission.rules` 读 revision → `permission.set_rules` 写 `{command:"printf DIMAGENT_APPROVAL_OK",match:"exact",decision:"allow"}` → `approval.respond {decision:"allow"}` → 工具结果成功；随后**新会话**发同一命令，未出现 `approval.request`，工具结果成功 |

同一次验证还确认了两条协议事实：

1. `session.submit` 没有 permission 字段；带 `"permission"` 提交会被服务端按 `-32602 invalid_params` 拒绝。档位由会话分支 policy 携带，切换走 `session.permission.update`。
2. 审批应答的决定是 `allow`/`deny`，不是 `approve`/`reject`；长期允许没有 `approval.allow_always` 方法，必须写 `permission.rules`/`permission.set_rules` 命令规则。

**命令规则真实证据**（同一夹具、独立临时 home；只记录 revision 与规则条数，不打印命令全文）：

```bash
DIM_REAL_MODEL_FIXTURE=<fixture> DIM_REAL_MODEL_SERVER=<server> \
  go test -run TestRealCommandRules -v -count=1 -timeout 6m ./internal/interaction/
```

```text
rules: read   revision=native:user/permission:1 count=0
rules: add    revision=native:user/permission:2 count=1
rules: edit   revision=native:user/permission:3 count=1
rules: revoke revision=native:user/permission:4 count=0
rules: cache  revision=native:user/permission:4 count=0
```

读、写、编辑、撤销都在真实 daemon 上成功；`permission.set_rules` 的回执只带 revision，规则集合以本次提交的集合为准。夹具复制件已删除，来源未修改。

**审批待答期间提交真实证据**（同一夹具；只记录状态、序号、模式、错误码与条数，不打印输入正文）：

```bash
DIM_REAL_MODEL_FIXTURE=<fixture> DIM_REAL_MODEL_SERVER=<server> \
  go test -run TestRealSubmitDuringApproval -v -count=1 -timeout 12m ./internal/interaction/
```

```text
queue: before  approvals=1 queuedInputs=0 turns=1
queue: submit  status=succeeded code=none
queue: after   approvals=1 queuedInputs=1 turns=1 mode=queue applied=false seq=41
queue: consume approvals=0 queuedInputs=0 turns=2

steer: before  approvals=1 queuedInputs=0 turns=1
steer: submit  status=succeeded code=none
steer: after   approvals=1 queuedInputs=1 turns=1 mode=steer applied=false seq=41
steer: consume approvals=0 queuedInputs=0 turns=1
```

结论：审批待答期间 queue 与 steer 都被接受；审批保持待决；同意后 queue 开新轮、steer 进当前轮。

**规则匹配真实证据**（同一夹具；只记录 revision、条数与匹配结果，不打印命令正文）：

```bash
DIM_REAL_MODEL_FIXTURE=<fixture> DIM_REAL_MODEL_SERVER=<server> \
  go test -run TestRealRuleMatching -v -count=1 -timeout 12m ./internal/interaction/
```

```text
rules: initial revision=native:user/permission:1 count=0
rules: add exact revision=native:user/permission:2 count=1
match: exact command allowed=true
match: exact with extra argument approval=true
rules: add prefix revision=native:user/permission:3 count=2
match: prefix leading words allowed=true
match: pipeline approval=true
match: pipeline approved and executed=true
rules: add deny revision=native:user/permission:5 count=4
match: deny rule applied_without_approval=true
```

结论：exact 只匹配完整词序列；prefix 按整词前缀匹配并覆盖额外参数；带管道的命令不被 allow 规则放行；deny 规则直接拒绝且不触发审批。

### 6.6 blocked

| 项 | 状态 | 缺什么 |
|---|---|---|
| 真实非空提问清单 | blocked | 需要能产生 `ask_user` 工具调用的真实模型；空页已由 D1 在真实 daemon 上实测，本版按 wire 形状造 fixture |
| 真实 `permission/changed` 事件隔离 | blocked | 真实切换档位已走 `session.permission.get/update`；时间线 `permission/changed` 的独立观测目前由 fakeappserver 覆盖 |
| 成员停止的真实语义 | blocked | 需要服务端真实的成员运行标识与停止方法；本版只验证意图形状 |

已裁决、不再 blocked：

- 跨会话队列：按会话一条队列，跨会话并行上限等于打开的会话数。
- 提问通知：协议没有 `question.*`/`elicitation.*`。提问已改为「清单 + 时间线事实」：`session.question.pending` 按固定截止点装载，`ask_user` 调用与工具结果驱动开与结算；本包不再等待、也不伪造提问通知。
- 真实审批链路：同意、拒绝与长期允许均已在本机真实 daemon + 真实模型上验证（§6.5）。
- 命令规则：真实 daemon 上的读取、新增、编辑、撤销已通过（§6.5）。

## 7. 已知限制

1. **只在内存**：待决集合、已答记录、规则已写入记录、尝试计数都不落盘；重启后这些记录为空，如果服务端快照仍把已应答项列为待决，会再次呈现并可能被再次应答；规则已写入但应答未受理的审批也可能被重写规则。R8 不允许持久化幂等/去重标识，因此本版只能依赖服务端尽快结算。
2. **记录不清理**：已答记录、规则已写入记录与已结算调用记录按标识累积，长进程会线性增长；第一版没有淘汰策略（装载清单会重置已结算调用记录，其余两项不重置）。
3. **清单分页由调用方完成**：`session.question.pending` 是不透明游标分页，本包只接收汇总后的 `QuestionSeed`；漏页会直接导致少提问。
4. **推导来源可能不完整**：清单读取失败时退回 D2 的历史推导；历史未加载到的范围不在推导结果里，`Item.Source` 可区分来源，界面应能说明这是推导而非清单。
5. **批次按请求作答、按调用结算**：同一请求标识下的多个 `ask_user` 调用合成一条待答项，应答一次覆盖整批；部分调用结算时整批仍待答，最后一个调用结算才移除。`Counts` 按请求（批次）计，D2 的待答清单按调用计；两处同时展示时需要对齐口径。
6. **截止点以下事实被丢弃**：序号不晚于清单截止点的 `ask_user` 调用与结算不改动集合，完全依赖清单正确反映当时状态；清单与事实矛盾时以清单为准，直到下一次装载。
7. **快照与清单顺序即队列顺序**：审批按 `Snapshot.Pending` 顺序重放，提问按清单/推导顺序装载；本包不按序号重排，跨审批与提问的先后由装载次序决定。
8. **请求标识跨类型唯一**：审批与提问共用一把键；同一标识同时出现在两类时，后者按重复丢弃。
9. **结算键不同**：审批按请求标识结算；提问按请求标识加调用标识结算。迟到的重复结算按无操作处理；与下一次装载冲突时，以下一次清单为准。
10. **不引入超时语义**：清单里长期未结算的提问会一直存在，直到工具结果/结算事实到达或下一次装载；本包不单方面改服务端状态。
11. **长期规则可能在重试时重写**：规则写入成功但等待被取消时，本包不知道成功，条目回到待呈现；用户重试会以新幂等键再写一次同一规则。规则内容相同，是否幂等取决于服务端。
12. **应答未收敛按已发出处理**：等待被取消或写登记关闭时，应答可能已到达；本包选择不重发（`ErrResponseUnsettled`），必要时由 D3 的需人工确认与人工核对收尾。
13. **档位取值不本地自造**：本包只保存服务端读到的 preset，不定义别名、不猜默认；`SetPermissionTier` 只校验非空，未知取值由服务端拒绝。
14. **成员停止靠发送方收窄**：本包只带 `memberId`；如果发送方把它解释成会话停止，就会影响主线。集成时必须按成员范围映射方法。
15. **无诊断存储**：坏事实与非法迁移通过 `Rejection`/错误返回；是否记录、记到哪里由调用方决定。
16. **应答同步等待**：`Respond` 等写登记收敛，慢服务端会拖住调用方；界面应在后台调用并提供上下文超时。
17. **提交不携带 permission 字段**：协议没有该字段；提交由会话分支 policy 携带档位，`WithPermission` 只保证已选档位并拒绝无值提交。
18. **`permission/changed` 不带版本号**：本包只更新 preset，不伪造版本；版本由下一次 get/update 收敛，过期写入以 `permission_conflict` 显式失败。
19. **长期允许只覆盖简单命令**：只有 `exec`/`execute` 且不含命令分隔符的调用可长期允许；读写文件等其他工具没有对应的服务端长期规则，本版不提供。
20. **命令规则只在内存**：规则缓存不落盘；重启后重新读取。revision 相同即保留缓存，不比较规则内容。
21. **规则冲突不自动重试**：`EditRule`/`RevokeRule`/`AddRule` 遇到 `ErrRulesConflict` 只返回错误，必须先 `LoadRules` 再决定；不合并、不覆盖。
22. **规则回执可能省略集合**：真实 `permission.set_rules` 只回 revision；本包以本次提交的集合更新缓存，若服务端另有裁剪，下一次 `LoadRules` 会纠正。
23. **规则只覆盖用户级命令**：本包保存会话 preset，但不管理 `allowedTools`/`disallowedTools` 或审批 preset 的写回细节；本包只管理 `permission.rules` 的命令规则。
24. **匹配按空白词，不做 shell 解析**：引号、转义、变量展开不参与匹配；规则命令本身也不能含分隔符。
25. **allow 不覆盖管道**：allow 规则只对不含分隔符的命令生效；带管道的命令即使某段匹配也不会自动放行。deny 规则按分隔符切段匹配，可能命中复合命令的一段。
26. **没有单条规则 API**：撤销/编辑都是整集重写并带 revision；冲突时需重读后重写。
27. **审批卡拿不到规则作用域**：`approval.request` 没有 scope/revision/ruleId；“用户级、所有会话生效”只能是客户端说明；规则集合需另读 `permission.rules`。

## 8. 下一步

1. 补真实 `permission/changed` 的独立观测与真实 `approval.cancelled`；目前切换档位已走真实 get/update，长期允许与同意/拒绝已有真实证据。
2. 补真实非空 `session.question.pending` 的证据：包含分页（不透明游标）与真实 `ask_user` 调用、工具结果结算；目前只有空页实测与 wire 形状 fixture。
3. 把版本冲突后的重读与用户提示接入界面：冲突时先 get 最新 version，再由用户决定是否重写；当前只返回 `ErrPermissionConflict`，不自动覆盖。
4. 与 D3 对齐「需人工确认」在界面上的呈现与人工核对入口，避免应答悬空。
5. 评估已答记录的清理与重启窗口：在不违反 R8 的前提下，用服务端清单/快照的结算时序降低重复呈现概率。
6. 补成员停止的真实证据：成员运行标识、停止方法、停止后主线仍运行的读取。
7. 把命令规则管理接到界面入口（列表、编辑、撤销、冲突重读）；本版只提供领域 API 与真实证据。
