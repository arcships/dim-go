# D2 会话投影（行身份、折叠、未落库增量、分页、按轮导出来源、未完成事项、目标）

Status: 第一版实现完成（P1-2 候选结果包；含分页、按轮导出来源与真实 daemon 证据）
Owner: D2 领域 agent
验证: 可执行命令与覆盖表见第 6 节；真实 daemon 证据见 6.3；真实模型下三类未完成事项见 6.6；任务输出、任务停止与提示建议见 6.7；用量、撤回与目标控制见 6.8；上下文预算与压缩的调查结论见 6.9；分支、插话、审批重放与 @ 搜索见 6.10；blocked 见 6.5
依据: `docs/domains/R.md` 第 1 部分功能块 2、块 4 的读取侧；`docs/domains/A.md`（会话视图、会话级状态含分页锚点、四条总规矩、启动校验）；`docs/domains/D1-connection.md`（wire 类型与会话流）
范围: 历史与实时折叠、行模型与身份、轮次归属、未落库增量、分页锚点与向前翻页、按轮历史来源（供导出）、四类未完成事项、后台任务输出分页与停止意图、提示建议、上下文用量与压缩摘要、排队输入撤回与目标暂停恢复的意图、目标、分支分桶、打开与装载/分支选择两张状态机。不含界面、连接、写路径的登记与发送、磁盘存储、审批应答、目录与导出文件本身。

---

## 1. 输入

| 输入 | 来源 | 本版使用方式 |
|---|---|---|
| 历史页 `protocol.SessionHistoryResult` | D1 `session.history`、`session.Snapshot.History`、`session.HistoryReplay.History` | 按序号升序逐条折叠；`through` 推进"已落库"位置；`turnStart` / `nextBefore` / `done` 驱动分页 |
| 实时通知 `session.event` | D1 `session.Listener.OnEvent` | 与历史走同一个折叠入口，按到达顺序追加 |
| 实时预览 `session.model.progress` | 同上 | 只进"末快照层"，不落库、不建正式行 |
| 未完成事项清单 | 服务端各截止点读取方法（`session.input.pending`、`session.question.pending`、`session.exec.pending`、`session.collaboration.pending`） | 由调用方按固定截止点读好后 `SeedPending*` 装进来，之后按事件结算 |
| 事件词表 | `dim-agent-types` 的 `EventKind::key()`（50 个内建键 + 扩展事件），形状取自产品基线 `timeline-fold.ts`、`goal-fold.ts`、`interaction-service.ts`、`task-inventory.ts` | 见 3.9（冻结版） |
| 状态机与总规矩 | `docs/domains/A.md` 的「打开与装载」「分支选择」「四条总规矩」 | 迁移表逐条对应，启动时校验 |

不读服务端存储（SQLite/JSONL/session 文件/daemon.log），不把历史写盘，不生成正式标识。

## 2. 产物

### 2.1 包结构

| 文件 | 一句话职责 |
|---|---|
| `internal/projection/view.go` | 只读视图类型：行、行类型与状态、轮次、目标、四类未完成事项、分支目录、装载/选择状态、丢弃与诊断类型 |
| `internal/projection/fold.go` | `Folder`：分支分桶、唯一折叠入口（`ApplyHistory` / `ApplyNotice`）、去重与乱序重建、清单装载、断线清尾、两张状态机的驱动、视图输出 |
| `internal/projection/rows.go` | 行与轮次的内部状态、行到只读视图的转换（未落库行不带标识）、折叠状态与重建 |
| `internal/projection/events.go` | 事件词表到事实生命周期的唯一实现（输入、轮次、文本、思考、工具、压缩、后台任务、协作请求、提问） |
| `internal/projection/paging.go` | 分页：最近一轮首屏、向前翻页请求、锚点推进、越界与重复页处理、`hasOlder` 判定 |
| `internal/projection/turnhistory.go` | 按轮历史来源（导出用）：读原始历史页 → 现有折叠 → 按轮切分，给出稳定的轮次号与游标 |
| `internal/projection/cut.go` | 切点查询：父分支某一轮结束处的序号，只读已加载事实，未结束/未加载都明确报出 |
| `internal/projection/execoutput.go` | 后台任务输出：按字节游标分页读，只记游标与单页上限，运行中可读、结束后读到 EOF |
| `internal/projection/execstop.go` | 后台任务停止：停止意图（给写登记）+ 发送 + "正在停止"标记，结算仍由结束事实负责 |
| `internal/projection/suggestion.go` | 提示建议：折叠 `session.notice` 的 offer / cleared，给出当前建议；顺便提供任务运行查询与停止标记入口 |
| `internal/projection/usage.go` | 上下文与用量：`model/usage` 累加、`context/budget` 取最新、`context/compaction` 摘要与次数，以及按需读数的 `ContextReading` |
| `internal/projection/inputcontrol.go` | 排队输入撤回与目标暂停/恢复：方法探测、写登记意图、发送接口 |
| `internal/projection/inventory.go` | 四类未完成事项：截止点装载、按事件结算、截止点以下的结算不生效 |
| `internal/projection/goal.go` | 目标折叠：目标块的解析、最新记录决定状态、删除后重算 |
| `internal/projection/preview.go` | 末快照层：`session.model.progress` 按偏移裁剪、按整行替换、与落库行对齐 |
| `internal/projection/state.go` | 两张迁移表与启动校验（状态与事件集合、目标可达、同一（状态，事件）不冲突） |
| `internal/projection/helpers.go` | JSON 取值、输入文本抽取（用户文本 / 普通文本 / 会话内命令 raw 块）、状态映射 |
| `internal/projection/d1.go` | 与 D1 会话流的交界：`Listen`、`ApplyListenerEvent/History`、`ApplySnapshot*` |

### 2.2 对外接口（第一版）

```go
folder, err := projection.New(projection.Config{SessionID: id}) // 校验迁移表，不合格即启动失败

// 折叠（唯一入口）
folder.ApplyHistory(branchID, page)      []Rejection  // 原始历史页（重读、回放）
folder.ApplyFirstPage(branchID, page)    []Rejection  // 最近一轮首屏，同时设分页位置
folder.ApplyOlderPage(branchID, page)    []Rejection  // 向前翻页，按请求游标裁掉越界事实
folder.ApplyNotice(method, params)       []Rejection  // 实时通知

// 分页
folder.OlderPageRequest(branchID, limit) (protocol.SessionHistoryParams, bool)
folder.Window(branchID)                  PageWindow    // {Anchor, Cursor, HasOlder}
folder.HasOlder(branchID)                bool
folder.ResetWindow(branchID)                          // 重设锚点重读

// 按轮历史（导出用）：一个实例服务一个会话的一个分支
source, err := projection.NewTurnHistory(sessionID, reader) // reader = D1 的 *session.Subscription
page, err := source.ReadHistory(ctx, projection.HistoryRequest{
    SessionID: sessionID, BranchID: "main", Before: 0, Limit: 100,
})
// page.Turns[i] = {Index, Data}；page.NextBefore / page.Done 驱动下一页

// 后台任务输出：按字节游标分页读，不缓存整份输出
output, err := projection.NewExecOutput(folder, branchID, taskID, reader)
output.SetLimit(64 << 10)
chunk, err := output.Next(ctx)   // {Text, Bytes, Cursor, EOF, Truncated}

// 后台任务停止：意图交给写登记，发送方由集成层实现
stop, err := projection.NewExecTaskStop(folder, branchID, taskID, sender)
intent := stop.Intent()          // Op()/Key()/Params() 给写登记
result, err := stop.Stop(ctx)    // {TaskID, Status}；Status=cancelled 表示已取消

// 提示建议：折叠 session.notice，读 View().Suggestion
folder.ApplyNotice("session.notice", params)

// 上下文与用量：直接读视图
view.Usage      // {Requests, PromptTokens, CompletionTokens, TotalTokens, CacheRead/WriteTokens, Cost}
view.Context    // {EstimatedInputTokens, ContextWindow, OutputTokens, ...}；Remaining()/Percent()
view.Compaction // {Sequence, Status, Summary, Strategy, RetainFrom, At}；view.Compactions 是次数
reading := projection.ContextReading{...}  // session.context.usage 的读数：Remaining()/Percent()/OverThreshold()

// 撤回排队输入 / 暂停恢复目标：先探测方法名，再登记意图
withdraw, err := projection.NewQueuedInputWithdraw(folder, branchID, inputID, methods, sender)
goal, err := projection.NewGoalControl(folder, branchID, projection.GoalPause, methods, sender)

// 切点（目录领域开分支用）
cut, ok := folder.TurnCut(sessionID, branchID, turnIndex) // ok=false 表示拿不到，不猜
point := folder.CutPoint(branchID, turnIndex)            // 带原因：ready / not_ended / not_loaded / no_branch

// 未完成事项与断线
folder.SeedPendingInputs/Questions/Tasks/Collaborations(branchID, through, items)
folder.OnDisconnect(branchID)

// 状态机与视图
folder.AdvanceLoad(event)  Transition
folder.SelectBranch(id)    error
folder.View() / BranchView(id)  View
folder.Branches()           []BranchInfo
folder.Diagnostics()        []Diagnostic

listener := projection.Listen(folder, onReject)  // 直接交给 D1 Manager.Acquire
```

`View` 只含行序列、轮索引、四类未完成事项、目标、分支状态、装载阶段、选择状态、锚点与 `hasOlder`；不含临时键、不含可写引用。

## 3. 设计

### 3.1 行模型与身份

行是视图的最小单元，一条消息可以拆成多个行。

| 行类型 | 来自哪些事实 | 身份（服务端标识） |
|---|---|---|
| 用户消息 | `input/message`（有用户或可见文本） | 该输入的 `event.id` |
| 助手文本 | `step/text` 的连续增量 | 该段首条事实的 `event.id` |
| 思考 | `step/thinking` 的连续增量 | 同上 |
| 工具调用与结果 | `step/tool-use-start`/`tool-use`/`tool-call` 与 `tool-completed`/`tool-result` | 调用的 `call_id` |
| 系统提示 | `context/compaction` | 该事实的 `event.id` |
| 目标变更 | 只带目标块的 `input/message` | 承载它的输入的 `event.id` |

规则：

1. 已落库的行用服务端标识；客户端不生成正式标识。
2. 未落库（尾部）的行在视图里 `Pending=true`、`ID` 为空；包内用行的内部键（预览行是"轮次+请求"）定位，这个键不出包。落库后**原地**换成服务端标识，行的位置与内容不变。
3. 一行只有一个服务端标识：目标写入不额外占用一条用户消息行，因此不会出现两行同标识。
4. 文本/思考的行边界由种类切换决定：连续同种增量共用一行，换种类（文本↔思考↔工具）关闭上一行；再次换回会开新行（新首事实标识）。
5. 工具的调用与结果合成一行；结果先于调用到达时先暂存，调用到达后合并，不产生第二个工具行。
6. 视图只给只读拷贝：工具参数字节与提问原始输入都做复制，调用方改不动折叠状态。

### 3.2 顺序、去重与乱序

- 顺序键是事实序号：行按其创建事实的序号在序列中占位，后到的增量只改这一行，不改位置。
- 去重按 `(分支, 序号)`：attach 重放与断线重读带来的同一序号只折叠一次。
- 乱序：一条序号低于已折叠位置的实时事实会让当前折叠结果失效，下次读取时按"清单 + 序号升序事件"整体重建。因此乱序与重复都不会产生重复行，也不会因为到达顺序不同而得到不同视图。
- `loadedThrough` 只由历史页推进（`through`），实时事实不推进它：分页锚点与"已落库"的位置不被增量移动。

### 3.3 折叠入口只有一处

```
历史页:  排序(序号升序) → applyPage → applyFact → (需要时重建后再) foldState.apply
实时通知:                            applyFact → foldState.apply
```

`ApplyHistory` / `ApplyFirstPage` / `ApplyOlderPage` 都落到 `applyPageLocked`，后者逐条交给 `applyFactLocked`，再交给同一个 `foldState.apply`。三个历史入口的差别只有窗口边界与分页位置，实时入口没有窗口。历史与实时没有第二套语义。

### 3.4 未落库增量（末快照层）

两层，互不重叠：

1. **已落库层**：被历史覆盖（序号 ≤ `loadedThrough`）的事实折叠出的行，带服务端标识。
2. **末快照层**：
   - 实时事实里序号大于 `loadedThrough` 的部分：同一折叠算出来的行，标 `Pending`、不带标识；
   - `session.model.progress` 的预览：只留偏移之后的尾部（`text[textOffset:]`），一条助手消息一份整行快照，新的整体覆盖旧的。

裁剪与替换按整行做：落库后预览整行消失，不会和落库文本逐字拼接，也不会重复；同一条预览反复到达只是整行覆盖。断线时清空这一层（`OnDisconnect` 或 `AdvanceLoad(LoadConnectionLost)`），重读历史后重新折叠进来。

### 3.5 未完成事项（四类）

| 类 | 打开事实 | 结算事实 | 备注 |
|---|---|---|---|
| 排队输入 | `input/message`（有用户或文本） | `input/consumed`、`input/deleted`、`input/withdraw` | `input/applied` 只标"已应用不可撤回"，仍在清单里 |
| 待答提问 | `step/tool-use`/`tool-call` 里名字是 `ask_user` 的调用 | 该 `call_id` 的 `step/tool-completed`/`tool-result` | 新一轮 `turn/start` 与 `turn/end` 会清掉上一轮残留 |
| 未结束后台任务 | `exec/started` | `exec/ended` | 按 `task_id` 结算 |
| 协作请求 | `collaboration/request` | `collaboration/result`/`cancel`/`close` | `delivered` 只标已接入，不结算 |

装载按固定截止点：`SeedPending*(branch, through, items)` 用服务端在 `through` 那一刻读到的清单替换本地这一类清单。之后：

1. 序号大于 `through` 的事实照常结算；
2. 序号不大于 `through` 的结算事实不生效——那一时刻的清单已经反映过它们；
3. 清单晚于结算事实装进来也不会把已结算项装回来（重建时先装清单再按序号重放）。

### 3.6 目标

- 目标由输入块承载：`blocks[].key == "goal:state"` 且正文以 `<dim-goal ...>` 开头；属性 `status`/`judge`/`reason`，正文里 `<objective>`、`<blocker>`、`<result>` 三段按行读回并解一层实体。
- 本分支最新的一条记录决定整个目标；记录退休（`retire`）或 `status=cleared/none`、正文没有目标，都表示没有目标。
- 承载记录的输入被 `input/deleted` 删除后，按剩下的事实重算（回到上一条记录，或清空）；目标变更行同时消失。
- 目标变更行给出这次变更的动作（`set`/`pause`/`resume`/`completed`/`blocked`/`cleared`）与变更后的状态。

### 3.7 分支分桶与结束

- 每个分支一份事实集合、一份折叠结果、一份清单、一份末快照、一份分页位置；行、轮次、清单、目标都按分支隔离。
- 目录事实：`branch/create`（在父分支里声明新分支、父标识、切点序号）、`branch/start`（分支自己声明父与切点）、`branch/end`/`branch/ended`（自己的分支结束）。主线不会结束。
- 分支结束后：序号大于结束事实的该分支事实一律丢弃并带原因 `branch_ended`，头部不再变；已结束分支仍可只读查看。
- 实时事实先于 `branch/create` 到达时按事实建桶，父标识与切点在目录事实到达后补上。

### 3.8 两张状态机与启动校验

「打开与装载」（6 条）与「分支选择」（3 条）逐条对应 `docs/domains/A.md` 的迁移表，见 `state.go`。补充约定：

- 非法迁移：丢弃该事件、记一条带状态机名的诊断、状态不变、不向用户报错（`Transition.OK=false`）。
- `就绪 + 连接中断 → 装载首轮` 同时清空未落库尾部并重设分页位置；`任意 + 关闭会话` 也清尾。
- 分支选择：未知分支拒绝并记诊断；从分支 A 切到分支 B 走"切回主线"+"选某个分支"两步，不新增表外迁移；选已结束的分支允许（只读）。
- `New` 调用 `validateTransitionTables`：当前状态、目标状态、事件三项任一越界，或同一（状态，事件）声明两个目标，即启动失败并点名状态机与条目。

### 3.9 事件词表（冻结版 v1，2026-11-10）

词表以服务端的 `EventKind::key()` 为准（50 个内建键 + 扩展事件）。本表**冻结**：V 的 fixture、D3、D4 以它为准；词表发生变化（服务端新增或改名）时先改本表再改实现与 fixture，不在消费端猜别名。

**本包折叠（31 条）**

| `type/name` | 折叠结果 |
|---|---|
| `input/message` | 用户行（有用户或可见文本）；目标块 → 目标 + 目标变更行；上下文块不进时间线 |
| `input/applied` | 清单标记"已应用不可撤回"，用户行仍在 |
| `input/consumed` | 结算输入 + 用户行收口 |
| `input/deleted` | 结算输入 + 删除用户行/目标变更行 + 目标重算 |
| `input/withdraw` | 与删除同形状：结算输入 + 删除用户行 |
| `turn/start` | 开一轮，把尚未归属的用户输入归入本轮；清掉上一轮提问 |
| `turn/end` | 结束一轮，收口块并清掉提问 |
| `step/model-start` | 不建行；确定该轮该请求的助手消息边界 |
| `step/text` | 助手文本行（同消息连续增量共用一行） |
| `step/thinking` | 思考行（同上） |
| `step/model-output-end` | 收口该助手消息的块（`failed`→error，`cancelled`→cancelled） |
| `step/tool-use-start` | 工具行（可按调用标识后合并 name/input） |
| `step/tool-use` | 工具行；`ask_user` 同时开一条待答提问 |
| `step/tool-call` | 批量工具行；`ask_user` 同样开提问 |
| `step/tool-completed` | 工具行结束（合并结果）；结算提问 |
| `step/tool-result` | 批量工具行结束；结算提问 |
| `context/compaction` | 系统提示行 + 最近一次压缩摘要与次数（`View.Compaction`/`Compactions`） |
| `model/usage` | 累加会话用量：请求数、prompt/completion/total、缓存读写、费用（`View.Usage`） |
| `context/budget` | 记录最近一次上下文占用：估算输入、窗口、输出预留、基线与增量（`View.Context`） |
| `exec/started` | 未结束后台任务加入清单 |
| `exec/ended` | 后台任务结算 |
| `collaboration/request` | 协作请求加入清单 |
| `collaboration/delivered` | 标记已接入，不结算 |
| `collaboration/result` | 协作请求结算 |
| `collaboration/cancel` | 协作请求结算 |
| `collaboration/close` | 协作请求结算 |
| `branch/create` | 分支目录项（父、切点） |
| `branch/start` | 分支目录项（父、切点），状态 open |
| `branch/resume` | 记录事实；本版不重开已结束分支 |
| `branch/end` | 分支结束，之后的事实丢弃 |
| `branch/ended` | 同上（服务端的结束事实名） |

**不折叠（19 条，逐条写明归谁）**

| `type/name` | 归属 | 理由 |
|---|---|---|
| `stop/interrupt` | 不处理（写登记 D3） | 控制事实，不在版本 1 的 wire 事件联合里；停止结果由 `turn/end` 收敛 |
| `approval/request` | D4 交互裁决 | 审批集合与队列归 D4，本包不读审批引用 |
| `approval/respond` | D4 | 同上 |
| `approval/cancelled` | D4 | 同上 |
| `permission/changed` | D4 | 权限档位归 D4 |
| `prompt/changed` | 不处理（D5 管理面） | 提示词配置不属于会话视图 |
| `prompt/configure-request` | 不处理 | 控制事实，不进时间线 |
| `prompt/configure-result` | 不处理 | 同上 |
| `context/configured` | 不处理（D5/账号） | 模型选择归管理面与账号 |
| `context/scope` | 不处理 | 上下文管理内部事实 |
| `context/prepare` | 不处理 | 同上 |
| `context/compact` | 不处理 | 同上 |
| `context/compaction-start` | 不处理 | 本版只提示"压缩完成"，开始事实不建行 |
| `model/call` | 不处理 | 模型调用读数，不属视图 |
| `model/call` 之外的模型读数 | 不处理（账号/管理面） | 逐次调用的原始请求与配置不进视图；累计用量已折叠 |
| `step/model-content` | 不处理 | 模型原始内容（含图片等），本版不建行 |
| `step/tool-start` | 不处理 | 工具执行开始；工具行已由调用事实收敛为 running |
| `branch/create-failed` | 不处理 | 失败的分支不建目录项 |
| `collaboration/message` | 不处理（D4/U1） | 成员消息不属于主分支时间线 |
| `session/forked` | 不处理 | fork 出的新会话由新折叠器承担，本包不共享状态 |
| 扩展事件（非保留域） | 不处理（D1 原样转发） | 词表外的扩展事实不猜含义 |

非法事实（缺标识/序号/类型/名字、别的会话的通知、坏通知、已结束分支的新事实、翻页越界事实）被丢弃并通过 `Rejection` 带原因返回，怎么报由调用方决定。

### 3.10 分页：锚点、游标与 hasOlder

打开会话只读最近一轮，更早的轮次按页向前读。

| 概念 | 规则 |
|---|---|
| 首屏 | `ApplyFirstPage`：折叠最近一轮的页，锚点设成这一页的 `turnStart`（没有则取页内最小序号） |
| 锚点 Anchor | 最早已装载位置的序号；**只由历史页向更早推进**，实时事实与断线重读都不会把它往前挪 |
| 游标 Cursor | 下一次向前翻页的 `before`：优先用服务端给的 `nextBefore`，否则用锚点 |
| hasOlder | 首屏页带 `turnStart` 时按"这一轮起点是否在分支起点之后"判定（服务端的 `done` 在窗口页只表示这一轮读完）；向前翻页的页不带 `turnStart`，按 `!done && cursor > 1` 判定 |
| 越界 | 序号不小于本次请求游标的事实不属于这一页：丢弃为 `out_of_window`，不进视图、不动锚点 |
| 重复页 | 不产生重复行；这一页没有带来任何新事实时停止继续翻页并记一条 `向前翻页` 诊断（避免同一游标原地打转） |
| 断线 | `LoadConnectionLost` 清空尾部并重设分页位置（锚点/游标/hasOlder 归零），重开时按 A 文档"重设锚点重读" |

请求参数由本包给出：`OlderPageRequest(branchID, limit)` 返回 `{branchId, before: cursor, limit}`；服务端返回后交给 `ApplyOlderPage`。分页位置按分支各存一份，不影响其它分支。

### 3.11 按轮历史（导出用）

目录领域（D4）导出时按游标逐页读全部轮次；本包提供这个来源，不新建折叠：

```go
source, err := projection.NewTurnHistory(sessionID, reader) // reader 只需实现 History(ctx, params)
page, err := source.ReadHistory(ctx, projection.HistoryRequest{SessionID, BranchID, Before, Limit})
```

| 项 | 规则 |
|---|---|
| 请求 | `SessionID`、`BranchID`（空表示主线）、`Before`（向前游标，0 = 从最近一轮开始）、`Limit`（本轮最多几轮） |
| 返回 | `Turns [{Index int64, Data json.RawMessage}]`（页内按轮次号升序）、`NextBefore`（下一页游标，0 = 没有更早的）、`Done` |
| 轮次号 | 会话内稳定的 1..N，按轮次起点排序；不随翻页顺序或装载顺序变化 |
| 数据 | `Data` 是一轮的 JSON：`{index, turnId?, status, outcome?, startedAt, endedAt?, rows}`，`rows` 就是视图行（字段名见 `Row` 的 JSON 标签） |
| 复用折叠 | 每页原始历史都交给 `applyHistoryWindow` → `applyPageLocked` → `applyFactLocked` → `foldState.apply`；没有第二套折叠 |
| 装载时机 | 第一次读取把该分支历史读全（导出本来就要读全部）：轮次号只有读全才能是绝对序号，服务端 wire 上不带轮次序号 |
| 行为 | 只导出服务端已确认的行（`Pending` 行不进文件）；未归属轮次的行归前一轮，出现在第一轮之前的归第一轮 |
| 边界 | 会话/分支与来源不一致时明确报错；历史页数超过上限报错，不产出半份历史；被丢弃的事实可经 `Rejections()` 取出 |

与目录领域的接法：本包不导入 `internal/catalog`（避免反向依赖）。D4 把三个类型改成指向本包的别名即可，`HistorySource` 直接成立，无需转换函数：

```go
type HistoryRequest = projection.HistoryRequest
type HistoryPage    = projection.HistoryPage
type Turn           = projection.HistoryTurn
```

`shape_test.go`（外部测试包）逐字抄了 D4 的三个形状做字段级双向赋值检查，并断言别名替换后 `*TurnHistory` 满足 `ReadHistory` 的形状；D4 换别名时若字段漂移，本包测试先红。

### 3.12 切点（开分支用）

目录领域从父分支的第 N 轮结束处开分支，需要一个序号；本包只回答"已加载事实里有没有这个序号"。

```go
cut, ok := folder.TurnCut(sessionID, branchID string, turnIndex int) (int64, bool) // 与目录接口同形
point := folder.CutPoint(branchID string, turnIndex int) CutPoint                  // 带原因
```

| 状态 | 含义 | 调用方动作 |
|---|---|---|
| `ready` | 该轮有 `turn/end` 事实，`Sequence` 就是它 | 用这个序号开分支 |
| `not_ended` | 该轮没有结束事实（还在进行，或只被下一轮隐式收口） | 明确拒绝：这一轮还没结束 |
| `not_loaded` | 该轮不在已加载范围内 | 明确拒绝：先向前翻页到该轮 |
| `no_branch` | 分支不在折叠里（未知分支或不是本会话） | 明确拒绝 |

规则：

- 只读现有视图的轮次与它们的结束事实，不新增折叠、不触发任何读取；查询前后 `View`、头部、分页位置都不变。
- 切点只认 `turn/end` 的序号：没有就说不可用，不拿本轮起点、行序号或时间凑。
- 轮次序号取自当前视图（与界面显示的是同一份），视图只覆盖已加载范围；已结束分支的轮次与切点保持固定，不随后续事实变化。

与目录领域的接法：`TurnCut(sessionID, branchID string, turnIndex int) (int64, bool)` 与 `catalog.BranchSource` 里同名方法同形（`shape_test.go` 编译期断言），集成层一行转发即可，本包不导入 `internal/catalog`。

### 3.13 后台任务输出

任务输出是服务端的事实，按字节游标分页读；本包不缓存整份输出、不落盘。

| 项 | 规则 |
|---|---|
| 读取接口 | `ExecOutputReader.ReadExecOutput(ctx, ExecOutputRequest) (ExecOutputPage, error)`；集成层用 `session.exec.output.page` 实现 |
| 请求 | 会话、分支、任务标识、字节游标 `Cursor`、单页字节上限 `Limit` |
| 返回 | `Text`（可能比字节数短：服务端丢掉控制字符）、`Bytes`（本页消费的字节数）、`NextCursor`（服务端给了就以它为准）、`Truncated` |
| 游标 | 按 `Bytes` 前进，**不按文本长度**；服务端给的 `NextCursor` 更大时采用它 |
| 上限 | 默认 64 KiB，最大 4 MiB（与服务端一致）；`SetLimit` 会夹紧 |
| EOF | 任务已结束（不在未结束清单里）且这一页没有字节；任务还在跑时空页只表示"暂时没有新输出" |
| 不缓存 | `ExecOutput` 只持有游标与上限；每页文本交出去就丢，`Next` 在 EOF 之后不再向服务端要页 |

读的循环由调用方驱动（`for { chunk := Next(); 写出 chunk.Text; if chunk.EOF { break } }`），本包不做定时轮询。

### 3.14 后台任务停止

停止是一次写入：本包给登记内容与发送接口，不自己登记、不自己重发。

| 项 | 规则 |
|---|---|
| 意图 | `ExecStopIntent{SessionID, BranchID, TaskID}`：`Op()` 返回 `exec.stop`，`Key()` 按会话+分支+任务派生幂等键，`Params()` 给出 `{op, branchId, taskId}` |
| 写登记 | 复用已有的 `write.KindStop`（"停止会话或分支上的运行"）：集成层把 `write.Intent{Kind: KindStop, SessionID, BranchID, Params: intent.Params()}` 登记，不需要新增意图类型 |
| 发送 | `ExecStopSender.StopExecTask(ctx, ExecStopRequest)`；集成层用写登记的发送方实现（它决定走控制连接还是会话流） |
| 收敛 | 服务端在任务真正结束后才回执，回执带结束状态（`cancelled` 表示已取消）；回执先到时本地把任务标成"正在停止"，**清单成员仍由 `exec/ended` 结算** |
| 幂等 | 同一任务重复点击只登记一次（同一个 `Key()`）；发送失败不标记、不重发 |

### 3.15 提示建议

服务端在某轮完成后给出"用户接下来可能会说的话"；它不进时间线，没有序号，也不重放。

| 项 | 规则 |
|---|---|
| 通知 | `session.notice {source: "PromptSuggestion", kind: "offer"\|"cleared", data: {turnId, text}\|{turnId, reason}}`；`ApplyNotice` 折叠，`View().Suggestion` 给出当前一条 |
| 结算 | `cleared` 只清同一轮的建议；没带轮次标识时一律清；断线清空（服务端不重放） |
| 隔离 | 别的来源的通知不看；别的会话的通知拒绝；建议按分支分开 |
| 请求与应答 | 集成层发 `session.suggestion.request {sessionId, turnId}` 与 `session.suggestion.respond {sessionId, turnId, action: accept\|dismiss}`；应答是写入，走写登记（目前没有专门的意图类型，集成层用 `KindManagementWrite` 带 op；需要时由 D3 增类型） |
| 开关 | 由设置源 `prompt-suggestion` 控制（默认关）；关着时请求返回 `accepted=false, reason=disabled` |

### 3.16 上下文用量与压缩摘要

只折叠事实、只做累加与取最新，不推导服务端没给的数字。

| 视图 | 事实 | 规则 |
|---|---|---|
| `View.Usage` | `model/usage`（`usage.{promptTokens, completionTokens, totalTokens, cacheReadTokens?, cacheWriteTokens?, cost?}`） | 逐条累加：请求数、各 token、费用；token 四舍五入成整数。没有事实时为 nil，不是零值 |
| `View.Context` | `context/budget`（`estimatedInputTokens`、`contextWindow`、`outputTokens`、`baselinePromptTokens?`、`deltaTokens?`） | 后来的覆盖先来的；`Remaining()` 与 `Percent()` 只按事实给出的窗口与估算计算，窗口缺失时为 0 |
| `View.Compaction` / `Compactions` | `context/compaction`（`status` 为 completed / failed / cancelled，完成时带 `replacement` 摘要、`strategy`） | 下一次覆盖上一次；次数累加；失败与取消没有摘要。摘要取 `replacement_parts.strategy` 指出的策略消息（`Message.content` 是 untagged 的：字符串或内容块数组都认）。系统提示行仍按行折叠 |
| `ContextReading` | 会话方法 `session.context.usage`（不是事件） | `context/budget` **不被协议 v3 发给客户端**（见 6.9），占用只能按需读；`Remaining()`/`Percent()`/`OverThreshold()` 只按读数里的数字算 |

### 3.17 排队输入撤回与目标暂停/恢复

两者都是"先探测方法名，再发一次写入"：方法表里没有就不发请求，报能力缺失。

| 项 | 撤回排队输入 | 暂停/恢复目标 |
|---|---|---|
| 探测 | `session.input.withdraw` | `session.goal.pause` / `session.goal.resume` |
| 意图 | `InputWithdrawIntent{SessionID, BranchID, InputID}`：`Op()=input.withdraw`、稳定幂等键、参数 `{op, branchId, inputId}` | `GoalControlIntent{SessionID, BranchID, Action}`：`Op()=goal.pause`/`goal.resume`、稳定幂等键、参数 `{op, branchId}` |
| 写登记 | 复用 `write.KindStop`（会话内的一次运行/动作停止）之外的既有类型不合语义：撤回属于会话内输入管理，建议 D3 增 `KindInputWithdraw`；在那之前集成层用 `KindManagementWrite` 带 op | 同上，建议 D3 增 `KindGoalControl`；在那之前用 `KindManagementWrite` 带 op |
| 发送 | `InputWithdrawSender.WithdrawInput` | `GoalControlSender.ControlGoal` |
| 回执 | `{status: "deleted"\|"notHeld"}`；`notHeld` 表示已被消费或已删除，什么也没写 | `{sequence}`：控制事实的序号 |
| 结算 | `input/withdraw`（可能跟一条 `input/deleted`）到达后，清单与用户行一起消失；**回执不做本地预删** | 服务端写下一条 `goal:state` 记录，目标折叠按它更新状态与暂停原因；**回执不做本地预改** |
| 零请求 | 方法表缺项或没探测过（空表）都返回能力缺失，不发请求 | 同上 |

## 4. 状态映射（与 A 文档对上）

| A.md 的状态 | 本领域实现 | 驱动 |
|---|---|---|
| 会话视图（轮次、消息、块、工具调用） | `foldState.rows` / `turns`，只由 `foldState.apply` 改 | `ApplyHistory` / `ApplyFirstPage` / `ApplyOlderPage` / `ApplyNotice` |
| 未落库尾部（末快照） | 未落库事实折叠出的行 + `branchState.preview`，标 `Pending` | 实时通知；断线清空 |
| 目标状态 | `goalFold`，删除后重算 | 输入块与 `input/deleted` |
| 分支目录 | `Folder.branches`（状态、父、切点、头部） | `branch/*` 事实 |
| 分支选择 | `Folder.selection` + 迁移表 | `SelectBranch`、已结束分支的新事实 |
| 打开与装载 | `Folder.loadPhase` + 迁移表 | `AdvanceLoad` |
| 分页锚点 | `branchState.anchor/cursor/hasOlder` + `paging.go` | 历史页（首屏与向前翻页）；断线重设 |
| 后台任务清单 | `inventory.tasks`（含"正在停止"标记） | `exec/started`、`exec/ended`；停止只加标记，不加成员 |
| 提示建议 | `branchState.suggestion`（可丢的展示状态） | `session.notice` 的 offer / cleared；断线清空 |
| 上下文用量 | `foldState.usage/context/compaction` | `model/usage`、`context/budget`、`context/compaction` 事实 |
| 对话之外的审批/提问答复 | 不在本包（D4） | — |

四条总规矩的落点：镜像状态只由事实驱动（行只能由 `apply` 改）；客户端状态只提交意图、不写事实（本包不写任何服务端状态）；每个状态只有一个驱动者（折叠、清单、目标、分页、两张状态机各有唯一入口）；不做乐观更新（未确认的内容只能以 `Pending` 出现，提交与应答不在本包）。

## 5. 错误与诊断

| 现象 | 处置 |
|---|---|
| 事实缺标识/序号/类型/名字 | 丢弃，`Rejection.Reason=invalid_event` |
| 通知属于别的会话 | 丢弃，`session_mismatch` |
| 通知参数无法解码 | 丢弃，`malformed_notice` |
| 已结束分支上的新事实 | 丢弃，`branch_ended`；若该分支正被查看，走表内自环 |
| 翻页页里的越界事实 | 丢弃，`out_of_window` |
| 选择未知分支 | 拒绝并记 `unknown_branch` 诊断 |
| 状态机非法迁移 | 丢弃、状态不变、记诊断（带状态机名、来源状态、事件） |
| 向前翻页没有进展（重复页/空页） | 停止翻页并记一条 `向前翻页` 诊断 |

## 6. 验收

### 6.1 命令与结果

```bash
go test ./...
go vet ./...
go test -race ./internal/projection/
DIM_APP_SERVER_BIN=<App Server> go test -run TestRealDaemonLiveAndHistoryAgree -v ./internal/projection/
```

本版结果（2026-11-10，本机 macOS arm64）：

- `go test ./...`：全部通过（含 `internal/projection` 与同期其它领域的包）。
- `go vet ./...`：无输出。
- `go test -race ./internal/projection/`：通过。
- projection 用例 72 个：65 个 fixture 用例通过；7 个真实用例在未设环境变量时跳过（`DIM_APP_SERVER_BIN` 见 6.3，`DIM_REAL_MODEL_FIXTURE` 见 6.6–6.10），设了就通过。

### 6.2 覆盖

| 要求 | 测试 |
|---|---|
| 同一 fixture 走历史与逐条实时结果一致；覆盖后身份一致、无第二条 | `TestHistoryAndLiveConverge` |
| 乱序与重复通知不产生重复行 | `TestDuplicateAndOutOfOrderNotifications` |
| 工具结果先于调用到达仍只有一行 | `TestToolResultBeforeCallKeepsOneRow` |
| 行身份两次读取一致（同折叠器、同 fixture 两个折叠器） | `TestRowIdentityStableAcrossReads` |
| 未落库增量叠加、落库后原地换标识且不重复 | `TestUnlandedIncrementLandsWithoutDuplication` |
| 断线清空尾部落、已落库内容不受影响 | `TestDisconnectClearsTail` |
| 末快照按偏移裁剪、按整行替换 | `TestProgressPreviewIsReplacedByLandedRows` |
| 四类未完成事项按截止点装载、按事件结算到零 | `TestPendingInventoriesUseCutoffAndSettleByFacts` |
| 成员接入不等于结算 | `TestDeliveredBeforeSettleKeepsRequestPending` |
| 晚到的清单不把已结算项装回来 | `TestLateSeedDoesNotResurrectSettledItems` |
| 截止点以下的结算事实不改清单 | `TestSeedAboveCutoffSettlementIsIgnored` |
| 新一轮开始清掉上一轮提问 | `TestTurnStartClosesPreviousQuestions` |
| 已应用的输入仍在清单里 | `TestAppliedInputStaysPendingButNotWithdrawable` |
| 撤回的输入从清单与视图一起消失 | `TestWithdrawnInputLeavesViewAndInventory` |
| 目标删除后重算、清空 | `TestGoalDeleteRecomputes` |
| 退休与清空记录表示没有目标 | `TestGoalRecordRetireAndCleared` |
| 阻塞与达成的摘要 | `TestGoalBlockedAndCompletedSummary` |
| 上下文输入不进时间线 | `TestGoalIgnoresContextOnlyInputs` |
| 分支事实各归各的桶、目录项正确 | `TestFactsAreBucketedByBranch` |
| 已结束分支丢弃新事实、头部不变 | `TestEndedBranchDropsFactsAndKeepsHead` |
| 选择未知分支被拒绝并记诊断 | `TestSelectBranchRejectsUnknown` |
| 分支间切换走两步表内迁移 | `TestSwitchBranchFromBranch` |
| 装载状态机合法路径 | `TestLoadPhaseMachineFollowsTable` |
| 非法迁移丢弃、状态不变、记诊断 | `TestIllegalLoadTransitionIsDropped` |
| 任意状态可关闭会话 | `TestCloseFromAnyState` |
| 迁移表与 A 文档逐条对应 | `TestTransitionTablesMatchDocument` |
| 不合格迁移表导致启动失败并点名状态机 | `TestBrokenTableIsRejected` |
| 缺会话标识即启动失败 | `TestNewRequiresSessionID` |
| 非法事实带原因丢弃 | `TestIllegalFactsAreRejectedWithReason` |
| 连续向前翻三页：行与一次读全一致、无重复无缺口、锚点停最早一轮 | `TestBackwardPagingCoversAllTurnsWithoutDuplicates` |
| 越界事实被丢弃、不改锚点 | `TestOlderPageDropsEventsOutsideWindow` |
| 重复页不产生重复行、不挪锚点、停止翻页 | `TestRepeatedOlderPageStopsWithoutDuplicates` |
| 首屏就说完时不再有向前翻页 | `TestFirstPageWithoutOlderTurns` |
| 锚点只由历史页推进，实时事实不动它 | `TestAnchorDoesNotMoveWithLiveFacts` |
| 断线重开重设分页位置 | `TestDisconnectResetsWindow` |
| 真实 daemon：实时与历史、落库、真实分页、导出式按轮读取一致 | `TestRealDaemonLiveAndHistoryAgree`，证据见 6.3 |
| 真实模型：后台任务、排队输入、提问三类的截止点读取与结算 | `TestRealDaemonPendingInventories`，证据见 6.6 |
| 任务输出按字节游标分页、运行中不算 EOF、结束后读到 EOF、EOF 后不再请求 | `TestExecOutputPagesByBytes` |
| 任务输出只留游标不缓存整份内容 | `TestExecOutputKeepsOnlyCursor` |
| 单页上限的默认值与服务端最大值 | `TestExecOutputLimitClamp` |
| 输出读取的构造与读取错误 | `TestExecOutputReportsErrors` |
| 停止后标记"正在停止"，任务仍由结束事实结算 | `TestExecTaskStopMarksStoppingUntilFact` |
| 停止意图的 op、幂等键与参数 | `TestExecTaskStopIntent` |
| 发送失败不标记；缺任务或发送方明确失败 | `TestExecTaskStopFailures` |
| 提示建议的 offer / cleared 结算 | `TestSuggestionOfferAndClear` |
| 别的来源、别的会话、断线都不留建议 | `TestSuggestionIsolation` |
| 建议按分支分开 | `TestSuggestionPerBranch` |
| 真实模型：任务输出分页、任务停止、提示建议 | `TestRealDaemonExecOutputAndStop`，证据见 6.7 |
| 用量按事实累加、没有事实时为空 | `TestUsageAccumulates` |
| 上下文占用取最新一条并给出剩余与比例 | `TestContextBudgetKeepsLatest` |
| 压缩摘要、状态与次数 | `TestCompactionSummaryAndCount` |
| 撤回先探测方法名、缺方法零请求 | `TestInputWithdrawNeedsMethod` |
| 撤回发一次请求，清单与视图由事实同步移除 | `TestInputWithdrawSettlesByFact` |
| 撤回意图的 op、幂等键与参数 | `TestInputWithdrawIntent` |
| 目标控制先探测方法名、非法动作被拒 | `TestGoalControlNeedsMethod` |
| 暂停/恢复各发一次，状态由记录结算 | `TestGoalPauseAndResume` |
| 真实模型：用量、撤回、目标暂停恢复 | `TestRealDaemonContextAndControls`，证据见 6.8 |
| 压缩摘要在 `Message.content` 为字符串时也能取到 | `TestCompactionSummaryTextVariant` |
| 失败的压缩保留状态、不给摘要、仍计次 | `TestCompactionFailedStatus` |
| 按需读数的剩余量、占比与阈值判定 | `TestContextReadingHelpers` |
| 真实模型：占用读数与手动压缩（含成功配方） | `TestRealDaemonContextAndCompaction`，证据见 6.9 |
| 真实模型：分支内容只落该分支、主线不变、插话、分支未完成事项 | `TestRealDaemonBranchIsolation`，证据见 6.10 |
| 真实模型：审批重放对投影的影响面、@ 搜索口径 | `TestRealDaemonApprovalReplayAndFileSearch`，证据见 6.10 |
| 多轮 fixture 连续翻页到底：轮次连续、无重复、无遗漏，行数据与一次读全一致 | `TestTurnHistoryPagesEveryTurnOnce` |
| 首屏从最近一轮开始，同一游标重复读取得到同一页 | `TestTurnHistoryStartsFromNewest` |
| 会话/分支与来源不一致时拒绝 | `TestTurnHistoryRejectsForeignRequest` |
| 缺读取来源或会话标识时明确失败 | `TestTurnHistoryRequiresReaderAndSession` |
| 按轮形状与目录领域的导出接口一致（别名即可满足） | `shape_test.go`（编译期检查） |
| 两轮会话各取切点，都等于该轮结束事实的序号 | `TestTurnCutPointsForEachTurn` |
| 轮次不在已加载范围：返回未加载、不改视图、不推进头部；补齐后可用 | `TestTurnCutOutsideLoadedRange` |
| 当前轮没有结束事实：明确不可用，收到结束事实后可用 | `TestTurnCutWithoutEndFact` |
| 已结束分支上仍能取到固定切点，头部不变 | `TestTurnCutOnEndedBranchStaysFixed` |
| 会话或分支不匹配时不可用 | `TestTurnCutRejectsForeignSessionAndUnknownBranch` |
| 切点方法与目录领域分支接口同形 | `shape_test.go`（编译期检查） |

### 6.3 真实 daemon 证据

环境：macOS arm64；App Server 为仓库外构建的 `dim-app-server` 0.1.0-alpha.18，临时数据目录 + `--mock-model`，不触碰任何用户数据目录。

```bash
DIM_APP_SERVER_BIN=<App Server 可执行文件> \
  go test -run TestRealDaemonLiveAndHistoryAgree -v -count=1 ./internal/projection/
```

观察到的输出：

```text
服务端 0.1.0-alpha.18（protocol 3，profile mock，owned=true）
会话 <session-id>
首屏: head=1 events=1 through=1 done=true
提交成功: 输入事件 <input-id>（序号 5）
提交成功: 输入事件 <input-id>（序号 18）
实时折叠: 4 行（未落库 4），2 轮
历史折叠: 4 行（23 条事实，through=27）
向前翻页: 5 页，4 行，锚点 1，hasOlder=false
导出式读取: 2 页，2 轮
证据: {"eventKinds":["branch/start","input/applied","input/consumed","input/message",
  "step/model-output-end","step/model-start","step/text","turn/end","turn/start"],
  "exportPages":2,"exportTurns":2,"historyEvents":23,"landedRows":4,"pages":5,
  "pendingTotal":0,"rows":4,"rowKinds":["user_message","assistant_text","user_message","assistant_text"],
  "turns":2,"serverVersion":"0.1.0-alpha.18"}
回收自启服务端: stopped=true
```

这条证据同时验证了：

1. **实时与历史一致**：实时折叠（首屏 + 逐条通知）与"另开折叠器只吃历史"的行内容、轮索引、目标、四类未完成事项全部相同。
2. **落库不重复**：把同一份历史再喂给实时折叠器后，行数不变（4），未落库行归零，行标识补齐且与纯历史视图一致。
3. **真实分页**：最近一轮打开（1 页）后连续向前翻 5 页到分支起点，锚点落到 1、`hasOlder=false`，行与一次读全逐条相同。
4. **导出式按轮读取**：`TurnHistory` 以同一条会话流为来源，每页 1 轮读到底（2 页 2 轮），轮次号 1..2 各出现一次，每轮行数据与该轮的行逐条相同。
5. **真实事件词表**：服务端在这一轮里发出 `branch/start`、`input/applied`、`input/consumed`、`input/message`、`step/model-*`、`step/text`、`turn/*`，都按 3.9 的表折叠。
6. **回收**：先释放持有者再回收自启实例，`stopped=true`，临时目录清空，无残留进程。

同一次验证也确认了服务端的 `session.submit` 契约：`submissionId` 必须是 UUID（`dimcode-server` 0.1.0-alpha.7 与 `dim-app-server` 0.1.0-alpha.18 都按此校验），否则回 `-32602`；缺省时服务端自己生成。

### 6.4 与 D1 的接口对齐

| D1 提供 | 本包消费方式 |
|---|---|
| `protocol.SessionHistoryResult`（`events`/`through`/`nextBefore`/`turnStart`/`done`） | `ApplyHistory`（回放）、`ApplyFirstPage`（首屏）、`ApplyOlderPage`（向前翻页） |
| `session.Snapshot.History`（首屏窗口） | `ApplySnapshot` / `ApplySnapshotBranch` → `ApplyFirstPage` |
| `session.HistoryReplay.History`（断线重读） | `ApplyListenerHistory`（D1 会话流固定主线） |
| `session.Event{Method, Params}` | `ApplyListenerEvent` → `ApplyNotice`；只认 `session.event` 与 `session.model.progress` |
| `session.Listener.OnResync` | `Listen` 里清空未落库尾部 |
| `protocol.SessionHistoryParams` | `OlderPageRequest` 直接产出，调用方原样发给 `Subscription.History` |
| `*session.Subscription.History(ctx, params)` | 直接满足本包的 `HistoryReader`，`NewTurnHistory(sessionID, subscription)` 即可 |
| 目录领域的 `BranchSource.TurnCut(sessionID, branchID string, turnIndex int) (int64, bool)` | 本包 `*Folder` 的同名方法同形，集成层直接转发；未加载与未结束都返回 `ok=false` |

D1 待确认：`OnHistory` 不带分支标识（本包按主线折叠）。

### 6.5 blocked

| 项 | 状态 | 缺什么 |
|---|---|---|
| 工具、思考、审批、提问、后台任务、协作请求的真实证据 | blocked | `--mock-model` 只产生文本回复，没有工具调用与思考；需要真实模型账号（R 的验收口径要求真实服务事实） |
| 协作请求的真实读取 | blocked | 需要真实会话上出现 `collaboration/request`（成员协作）；后台任务、排队输入、提问三类已在真实模型上验证（6.6） |
| 审批集合"整体替换"的条数验收 | blocked（归 D5） | 本包不持有审批；6.10 只证明重放会重发同一 requestId 且不影响提问清单 |
| 分支内提问/协作清单的真实读取 | blocked | 6.10 验证了分支内的排队输入与后台任务；提问与协作需要模型或成员协作配合 |
| 任务输出的真实大流量与截断 | blocked | 需要一段超过服务端 64 MiB 输出上限的命令；本版只验证了多页小输出与 EOF |
| `context/budget` 的真实事实 | 不适用（服务端不发） | 结论见 6.9：该事实被协议 v3 的可见性白名单排除，任何客户端都收不到；占用改用 `session.context.usage` 读取（已有真实证据） |
| `context/compaction` 的真实事实 | 已取得 | 见 6.9：`gpt-5.6-terra` + 4 轮大输出后手动压缩成功（completed、替换 19 条、摘要 1177 字符） |
| 分支会话的真实证据 | blocked | 需要在真实服务端上开分支并读分支历史 |
| Windows 平台 | blocked | 没有 Windows 主机 |

### 6.6 真实模型：三类未完成事项（2026-11-11）

前提：真实 daemon 0.1.0-alpha.21 + 本机临时夹具账号（真实 provider `deepseek-v4.1-flash`），
夹具只复制不修改、不入库。会话用 `session.create` 建，**不传 `tools`**（传空数组等于一个工具都不挂），
`permissionPolicy.preset = "full-access"`（否则工具调用会停在审批上）。

```bash
DIM_REAL_MODEL_FIXTURE=<夹具目录> \
  go test -run TestRealDaemonPendingInventories -v -count=1 ./internal/projection/
```

观察到的输出（只留事件类型、标识、序号与状态）：

```text
服务端 0.1.0-alpha.21（profile product）
会话 <session-id>
后台任务开始: sequence=39 task=<task-id>
未结束任务: task=<task-id> sequence=39 through=39
后台任务结束: sequence=49 task=<task-id>
运行中提交排队输入: input=<input-id>
未处理输入: input=<input-id> sequence=51 mode=queue applied=false
未处理输入: input=<input-id> sequence=56 mode=queue applied=false
排队输入被消费: sequence=88 input=<input-id>
提问工具调用: sequence=109
待答提问: request=<request-id> call=<call-id> sequence=109 through=109
已应答提问: request=<request-id> call=<call-id>
提问被工具结果结算: sequence=110 call=<call-id>
证据: events=[input/message input/applied turn/start input/consumed step/model-start
  step/thinking step/tool-use-start step/tool-use model/usage step/model-output-end
  step/tool-call exec/started step/tool-result step/text turn/end exec/ended
  notice:session.state notice:session.model.progress notice:session.mcp.state notice:session.notice]
回收自启服务端: stopped=true
```

三类各自结论：

| 类 | 读取 | 结算 | 结果 |
|---|---|---|---|
| 未结束后台任务 | `session.exec.pending` 在 `exec/started`（序号 39）之后列出该任务（`through=39`）；本包按服务端清单装载后为 1 | `exec/ended`（序号 49）之后清单为空，本包清单结算为 0 | 通过 |
| 排队输入 | 运行中提交的输入（序号 56，`mode=queue`，`applied=false`）出现在 `session.input.pending`；本包按清单装载后能查到 | `input/consumed`（序号 88）之后清单不再列出它，本包清单同步消失 | 通过 |
| 待答提问 | 模型调用 `ask_user`（`step/tool-call` 序号 109）后 `session.question.pending` 给出批次（`through=109`）；本包装载后能查到 | 用 `session.question.respond` 应答（选中模型给的第一个选项），`step/tool-result`（序号 110）结算，清单为空，本包清单同步消失 | 通过 |

试过什么、为什么这样：

1. 一开始给 `session.create` 传 `tools: []`，模型拿不到任何工具：整轮只有文本，没有 `exec`、没有 `ask_user`。改成不传 `tools` 后工具才挂上。
2. 会话默认权限策略下 `exec` 会先发 `approval.request`，整轮停在那里（观察到 `notice:approval.request`）。改成 `permissionPolicy.preset=full-access` 后不再需要审批——审批语义归 D5，本证据不碰。
3. 后台任务用 `exec` 的 `runInBackground: true` + `timeoutMs: -1`；排队输入在 `turn/start` 之后立刻以 `mode=queue` 提交；提问要求模型"先问再答"并给两个选项，应答用 `session.question.respond`。
4. 三项判据都是"事件 + 清单"，不看模型文案：模型措辞、选哪个选项都不影响结论。

数据纪律：夹具只在机器上的临时目录里，只在复制出来的副本上跑，未写回来源；日志只含事件类型、标识、序号、状态，没有提示词、命令正文、模型输出与凭据；临时目录与自启服务端在用例结束时清掉（`stopped=true`，无残留进程）。

### 6.7 真实模型：任务输出、任务停止、提示建议（2026-11-11）

前提同 6.6（真实 daemon 0.1.0-alpha.21 + 临时夹具复制）。提示建议默认关闭，用例只在**夹具副本**上把设置源 `prompt-suggestion` 打开（`settings.update`，revision 1 → 2），来源夹具不动。

```bash
DIM_REAL_MODEL_FIXTURE=<夹具目录> \
  go test -run TestRealDaemonExecOutputAndStop -v -count=1 ./internal/projection/
```

观察到的输出（只留事件类型、标识、序号、状态、字节数与分页数）：

```text
服务端 0.1.0-alpha.21（profile product）
会话 <session-id>
后台任务开始: sequence=39 task=<task-id>
运行中读输出: bytes=0 cursor=0 eof=false（任务仍在跑）
任务输出: task=<task-id> 分页=6 字节=30
后台任务开始: sequence=76 task=<task-id>
任务停止: task=<task-id> status=cancelled cancelled=true
停止的结束事实: sequence=78 status=cancelled
提示建议设置已在夹具副本上打开: revision=native:user/prompt-suggestion:2
最近一轮已结束: turn=<turn-id> sequence=86
提示建议请求已受理: turn=<turn-id>
提示建议已到达并折叠: turn=<turn-id> 文本长度=53
提示建议已结算: turn=<turn-id>
回收自启服务端: stopped=true
```

三项结论：

| 项 | 观察 | 结论 |
|---|---|---|
| 任务输出 | 运行中读一页（当时还没有输出，`bytes=0`、未判 EOF）；结束后按 8 字节一页读到 EOF，共 6 页 30 字节 | 运行中可读、结束后读到 EOF；游标按字节前进；没有整份缓存 |
| 任务停止 | `session.exec.stop` 回执 `status=cancelled`；`exec/ended`（序号 78）状态同为 `cancelled`；`session.exec.pending` 与本包清单都不再列出该任务 | 按任务标识停止，状态收敛为非运行 |
| 提示建议 | 打开开关后请求受理；`offer` 通知到达并折成建议（只记文本长度 53，不打印正文）；`dismiss` 应答后 `cleared` 通知到达，建议清空 | 请求、通知与结算都走通；没碰界面 |

数据纪律：日志只有事件类型、标识、序号、状态、字节数与分页数；没有命令输出、建议正文、提示词或凭据；夹具复制运行，用例结束 `stopped=true`，无残留进程。

### 6.8 真实模型：用量、撤回、目标暂停/恢复（2026-11-11）

前提同 6.6（真实 daemon 0.1.0-alpha.21 + 临时夹具复制，full-access，不传 `tools`）。

```bash
DIM_REAL_MODEL_FIXTURE=<夹具目录> \
  go test -run TestRealDaemonContextAndControls -v -count=1 ./internal/projection/
```

观察到的输出（只留状态、标识与数字）：

```text
服务端 0.1.0-alpha.21（profile product）
会话 <session-id>
用量: 请求=1 prompt=11325 completion=2 total=11327 缓存读=9600
上下文占用: 这一轮没有 context/budget 事实（服务端未发布）
压缩摘要: 这一轮没有触发压缩（compaction 由上下文策略决定，不可人为触发）
撤回前: input=<input-id> mode=queue applied=false
撤回回执: status=deleted（服务端方法 session.input.withdraw）
撤回事实: type=input/withdraw sequence=43
目标: 已设置，状态=active
目标: 已暂停，状态=paused 原因=user
目标: 已恢复，状态=active
证据: events=[input/message input/applied turn/start input/consumed step/model-start step/text
  model/usage step/model-output-end turn/end input/withdraw input/deleted step/thinking
  step/tool-use-start step/tool-use step/tool-call step/tool-result]
回收自启服务端: stopped=true
```

| 项 | 观察 | 结论 |
|---|---|---|
| 用量 | 一次真实调用后 `model/usage` 折出 `requests=1`、prompt 11325、completion 2、total 11327、缓存读 9600 | 通过 |
| 上下文占用 | 这一轮服务端没有发 `context/budget`，`View.Context` 为 nil，只记日志 | blocked（服务端未发布该事实；折叠与视图由 fixture 覆盖） |
| 压缩摘要 | 没有触发压缩（由上下文策略与环境决定，不可人为触发） | blocked（fixture 覆盖 completed/cancelled 两种摘要） |
| 撤回 | `session.input.withdraw` 回执 `deleted`；`input/withdraw`（序号 43）与随后的 `input/deleted` 到达后，服务端清单与本包清单都不再列出该输入 | 通过 |
| 目标 | `session.goal.set` → active；`GoalPause` → paused（原因 user）；`GoalResume` → active | 通过 |

数据纪律：日志只留状态、标识与数字；没有目标正文、输入正文、模型输出与凭据；夹具复制运行，`stopped=true`，无残留进程。

### 6.9 调查：上下文预算与压缩为什么拿不到真实事实（2026-11-11）

**结论一：`context/budget` 是服务端内部事实，v3 客户端永远收不到。**

依据（服务端源码，只读）：`dim-app-server/src/tree_protocol.rs` 的 `ProductEvent::new` 按事件类型决定是否发给客户端；`ContextBudget`、`ContextPrepare`、`ModelCall`、`Prompt*`、`Approval*` 落在 `false` 分支（"not visible"），而 `ContextCompaction`/`ContextCompactionStart`/`ContextCompact` 要求 `version >= 3`（可见）。所以无论会话多长、用哪个 provider、开哪个设置，`session.event` 与 `session.history` 都不会出现 `context/budget`——这不是环境条件，是协议的可见性决定。

可用的等价读法是会话方法 **`session.context.usage`**，同一份估算事实按需返回。真实读数：

```text
占用读数(session.context.usage): 窗口=258000 输出预留=128000 阈值=114000 估算输入=3062 基线=full_estimate
占用事实(context/budget): 客户端未收到（服务端 v3 可见性白名单排除）
```

**结论二：压缩可以稳定触发，但要先让策略"有东西可切"。**

触发路径有两条：

1. **手动**：会话方法 `session.context.compact`（`session.methods` 列出，mutating）写 `context/compact` 并等配套的 `context/compaction`；回执是压缩结果。
2. **自动**：上下文准备时估算输入 ≥ `thresholdTokens` 才会自动压缩；该阈值随模型：`deepseek-v4.1-flash`（1M 窗口）是 782,035，`gpt-5.6-terra`（258k 窗口）是 114,000。

但 summary 策略还要过一道：切掉之后必须**真的变小**，保留目标 = `min(0.5 × 输出预留, 0.1 × (窗口 − 输出预留))`。两次失败的尝试都卡在这里：

| 尝试 | 模型 | 估算输入 | 保留目标 | 结果 |
|---|---|---|---|---|
| 1 轮短会话 | deepseek-v4.1-flash（1M/131k） | 11,305 | 65,536 | `status=failed`，`entriesReplaced=0`，错误 `Compacted context does not reduce input.` |
| 1 轮大输出 | gpt-5.6-terra（258k/128k） | 15,009 | 13,000 | 仍 `failed`：策略自己的估算（8,396）低于保留目标 |

**成功配方**（可复现）：选保留目标最小的模型 + 把上下文推过它。

```text
会话：model = {provider: "dimcode-api-oauth", modelId: "gpt-5.6-terra"}（258k 窗口 / 128k 输出 → 保留目标 13,000）
轮次：4 轮，每轮让模型用 exec 跑 `seq 1 20000`（约 30k 字符内联，约 6k token）
读数：估算输入 3,062 → 15,040 → 21,201 → 27,383 → 33,569
压缩：session.context.compact → status=completed，策略 summary，替换 19 条
压缩后：估算输入 8,730（压缩前 33,569）
```

观察到的证据（只留状态、标识与数字）：

```text
服务端 0.1.0-alpha.21（profile product）
第 4 轮后: 估算输入=33569 阈值=114000 窗口=258000
压缩回执: err=<nil> status=completed（耗时 10.2s）
压缩事实: seq=118 status=completed 策略=summary 替换条目=19 估算输入=19414 估算输出=8731
压缩事实形状: keys=[command context_baseline diagnostics digest_entries replacement replacement_parts
  retain_from start status strategy through] replacement=2 parts=map[digest:1 strategy:1]
折叠结果: status=completed 策略=summary 摘要长度=1177
压缩后占用: 估算输入=8730（压缩前 33569）
结论: 手动压缩成功
回收自启服务端: stopped=true
```

可复现命令：

```bash
DIM_REAL_MODEL_FIXTURE=<夹具目录> \
  go test -run TestRealDaemonContextAndCompaction -v -count=1 ./internal/projection/
```

这条用例会自己把上下文推过保留目标再压缩，并断言：压缩事实对 v3 可见、折叠状态与事实一致、成功时摘要非空且压缩后占用变小。

**本任务的小改（已说明）**：

1. `compactionSummary` 现在认得 `Message.content` 的**字符串**形式（真实 wire 里 `MessageContent` 是 untagged 的 `Text(String)|Parts(Vec<MessagePart>)`）；此前只认内容块数组，导致真实成功的压缩摘要为空。
2. 新增只读类型 `ContextReading` 与 `Remaining()/Percent()/OverThreshold()`：`View.Context` 仍只由 `context/budget` 事件驱动（v3 下恒为 nil），客户端该用 `session.context.usage` 的读数。

### 6.10 真实模型：分支、插话、审批重放与 @ 搜索（2026-11-11）

前提同 6.6（真实 daemon 0.1.0-alpha.21 + 临时夹具复制）。两段证据各一条命令：

```bash
DIM_REAL_MODEL_FIXTURE=<夹具目录> \
  go test -run TestRealDaemonBranchIsolation -v -count=1 ./internal/projection/
DIM_REAL_MODEL_FIXTURE=<夹具目录> \
  go test -run TestRealDaemonApprovalReplayAndFileSearch -v -count=1 ./internal/projection/
```

**分支隔离与分支内未完成事项（26 秒）**

```text
主线一轮结束: turn=<turn-id> sequence=36
主线行: 2
分支已创建: branch=<branch-id> 切点=36
分支首轮结束: turn=<turn-id> sequence=33
分支建好: 行=6
分支第二轮结束: turn=<turn-id> sequence=45
主线行未变: 2 行
插话: input=<input-id> sequence=54 清单模式=steer 已应用(seq=55)
插话折叠: 行=<row-id> 轮=3
分支停止: turn=<turn-id> outcome=cancelled sequence=87
分支停止后主线行未变: 2 行
分支未完成事项: 分支=1 主线=0（through=104）
分支任务停止: status=cancelled
证据: 主线行=2 分支行=18 分支任务结算=ok
回收自启服务端: stopped=true
```

| 项 | 事实 | 结论 |
|---|---|---|
| 开分支 | `session.branch_create {sessionId, parentBranchId:"main", at:<主线该轮 turn/end 序号>, prompt}` → `{branch:{branchId,...}}`；服务端用分支提示词自动跑一轮（序号 37–45） | 切点用父分支轮次结束处（与 `TurnCut` 同一口径） |
| 分支历史 | `session.history {branchId}` 读到分支自己的事实；折叠后 8 行，再提交一轮后 17 行 | 分支内容只落在该分支 |
| 主线不受影响 | 主线 `session.history` 两次读取行标识序列不变（2 行）；服务端 `session.branches` 里主线 head 34、分支 head 65 | 分支事实不抬高主线 |
| 插话 | 运行中 `session.submit {branchId, mode:"steer"}`：`session.input.pending` 里这一条 `mode=steer`，随后 `input/applied`（序号 55），折叠后归到运行中的第 3 轮 | 投递方式与服务端行为一致 |
| 分支停止 | 分支内 `exec` 跑 `sleep 60` 时 `session.stop {branchId}`：写 `stop/interrupt`，该轮以 `outcome=cancelled` 结束（序号 87）；主线行标识不变 | 停止只作用于该分支 |
| 分支未完成事项 | 分支内后台任务：`session.exec.pending {branchId}` 分支 1 条、主线 0 条；停止后 `status=cancelled` 且清单清空 | 分区清单按分支隔离并结算 |

**审批重放与 @ 搜索（5 秒）**

```text
@ 搜索: 服务端 5 条，本机扫描 5 条（工作区放了 5 个探针文件与 1 个隐藏文件）
@ 搜索候选: [<dir> <dir>/extra.go <dir>/readme.txt <name>.go <name>.md]
审批请求: request=<request-id>（第一次）
提问集合: 重放前 0 批
审批请求: request=<request-id>（重放）
提问集合: 重放后 0 批（重放只涉及审批）
审批回应: request=<request-id> decision=deny
回收自启服务端: stopped=true
```

| 项 | 事实 | 结论 |
|---|---|---|
| 审批重放 | 默认权限策略下模型调 `exec` → 收到 `approval.request`（request <request-id>）；用第二个 `Manager.Acquire` 重新 attach，服务端**再次交付同一个 requestId** | 重放发的是同一批待决请求，客户端应整体替换而不是累加（**集合语义归 D5**，本包只观察） |
| 提问集合不受影响 | `session.question.pending` 与 `View.Pending.Questions` 在重放前后都是 0 | 审批重放不动提问清单 |
| @ 搜索 | `session.methods` 里有 `session.files.search`；查询词返回 5 条候选，与本机扫描（隐藏文件与隐藏目录排除）逐条一致 | 口径一致；方法缺失时按能力缺失、零请求（**搜索归 D7**，这里是方法探测与口径对照） |

两项边界说明：

1. **审批集合整体替换**是交互裁决（D5）的状态语义；本包不持有审批，只证明重放会重发同一 requestId、且不影响本包的提问清单。要让 D5 的"条数等于服务端"可验收，需要在 D5 的待决集合上重跑同一场景。
2. **会话内 @ 搜索**是工作区领域（D7）的能力；本包只探测方法名与对照一次结果口径，不作为 D7 的验收证据。

## 7. 已知限制

1. 目标只折叠当前 `goal:state` 记录；旧键 `goal:objective`/`goal:status` 只参与变更行归类，不参与状态推导（旧服务端兼容需要补齐）。
2. 系统提示只折叠 `context/compaction`（压缩完成）；其余上下文事实按 3.9 不折叠。
3. 已结束分支不重开：`branch/resume` 不改变状态。
4. 未落库尾部没有条数上限，实际由服务端重放量约束；预览行不带时间与序号（视图里 `Sequence=0`），排序靠"追加在末尾"。
5. 工具调用的审批引用不在行上：审批由 D4 持有，本包不读审批通知。
6. `Folder` 用一把互斥锁串行化折叠与读取；监听器回调里不要阻塞，也不要回调本包做重活。
7. 分页只做"向前翻"：向后翻（`after` 追新）由 D1 的会话流承担，本包不额外维护游标。
8. 真实证据里的模型回复是 mock 的固定文本，只覆盖文本行；工具、思考与审批路径目前只有 fixture 证据（见 6.5）。
9. `input/withdraw` 只做"结算 + 删除行"，撤回原因暂不显示。
10. 按轮历史第一次读取会读完整个分支：轮次号要绝对序号就必须读全（服务端 wire 不带轮次序号）。导出本来就要读全部，界面实时读取不要用它。
11. 导出载荷只含轮次与行；附件、图片、Markdown 渲染由后续表达层决定，不在本包。
12. `TurnHistory` 一个实例只服务一个会话的一个分支；换会话或换分支要新建实例。
13. 切点只覆盖已加载范围：调用方（目录/界面）拿着视图里的轮次序号来问；没翻到那一轮就只能得到"未加载"，本包不会替它读。
14. 只有 `turn/end` 才算结束事实；被下一轮隐式收口的轮次没有切点（按"不猜"处理）。
15. 真实模型证据只覆盖本机一个账号与一个模型（`deepseek-v4.1-flash`）：模型是否调用 `exec`/`ask_user` 由模型决定，判据是事件与清单而不是文案；换账号或换模型需要重跑。
16. 协作请求（`collaboration/request`）的清单仍未在真实会话上验证（需要成员协作场景），见 6.5。
17. 任务输出是拉取式：本包不轮询、不跟随，读的节奏与展示由调用方决定；`Truncated` 只透传，不解释成"内容不全"。
18. 任务停止复用 `write.KindStop`（会话/分支运行停止）：语义上它是"停会话内的一次运行"，参数用 `op: "exec.stop"` + 任务标识区分；D3 若要单独的意图类型，本包意图形状不变。
19. 提示建议的应答没有专门意图类型，集成层暂用 `KindManagementWrite` 带 op；建议本状态可丢，断线即清空，不参与事实判断。
20. 撤回与目标控制都没有专门的写登记意图类型：本包给形状与幂等键，集成层暂用 `KindManagementWrite` 带 op；D3 增类型后只需改调用点。
21. 撤回的收敛依赖事实：回执是 `deleted`，`input/deleted` 可能晚一拍落库；调用方应按事实等待，不要拿回执当已删除。
22. 用量是累计值：只反映本分支已折叠的 `model/usage`，不含服务端账号侧的聚合与计费口径。
23. `View.Context` 在协议 v3 下恒为 nil（`context/budget` 不发给客户端）；要显示占用请读 `session.context.usage`，并把它按 `ContextReading` 换算。若将来协议开放该事件，折叠路径已经就绪。
24. 压缩成功的条件是"策略估算超过保留目标"，与自动压缩阈值不是一回事：手动压缩在上下文很小时会以 `failed` 收口，调用方要把 `status=failed` 当"这次没压成"，不是错误，也不影响已落库内容。
25. 分支建好时服务端会用分支提示词自动跑一轮；客户端不需要为这一轮另外提交，等它结束再提交自己的输入。
26. 插话在 `session.input.pending` 里会以 `mode=steer` 出现一小段时间（已应用未消费），不能把"出现在清单里"当成"排队等下一轮"；判断依据是 `mode` 与 `input/applied`。

## 8. 下一步

1. 接真实模型跑一次带工具调用与思考的会话，补 6.5 的 blocked 项（工具行、思考行、审批）。
2. 旧键目标兼容（`goal:objective`/`goal:status`）与目标判定者（evaluator）例外规则。
3. 与 D3 对齐导出：导出按轮读取全部历史，本包的 `ApplyHistory` 已可按页喂入，需要补按轮切分。
4. 与 D4 对齐"待答提问"的语义边界：本包只维持清单，提问的候选项解释与应答顺序归 D4。
5. 把 3.9 的冻结表交给 V 做成 fixture 词表，服务端词表变化时先改表再改实现。
6. 等 D4 接上别名后，跑一次真实导出落盘，把导出文件的轮次与该会话的轮索引逐条对照。
7. 协作请求清单的真实证据：在真实会话里触发一次成员协作，读 `session.collaboration.pending` 并按 `collaboration/result` 结算。
