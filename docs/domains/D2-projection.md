# D2 会话投影（行身份、折叠、未落库增量、分页、按轮导出来源、未完成事项、目标）

Status: 第一版实现完成（P1-2 候选结果包；含分页、按轮导出来源与真实 daemon 证据）
Owner: D2 领域 agent
验证: 可执行命令与覆盖表见第 6 节；真实 daemon 证据见 6.3，blocked 见 6.5
依据: `docs/domains/R.md` 第 1 部分功能块 2、块 4 的读取侧；`docs/domains/A.md`（会话视图、会话级状态含分页锚点、四条总规矩、启动校验）；`docs/domains/D1-connection.md`（wire 类型与会话流）
范围: 历史与实时折叠、行模型与身份、轮次归属、未落库增量、分页锚点与向前翻页、按轮历史来源（供导出）、四类未完成事项、目标、分支分桶、打开与装载/分支选择两张状态机。不含界面、连接、写路径、磁盘存储、审批应答、目录与导出文件本身。

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

**本包折叠（29 条）**

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
| `context/compaction` | 系统提示行（摘要在 `summary` 或 `replacement` 的 strategy 段） |
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

**不折叠（21 条，逐条写明归谁）**

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
| `context/budget` | 不处理 | 上下文预算读数 |
| `model/call` | 不处理 | 模型调用读数，不属视图 |
| `model/usage` | 不处理（账号/管理面） | 用量归账号与管理面 |
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
- projection 用例 45 个：44 个 fixture 用例通过；1 个真实 daemon 用例在未设 `DIM_APP_SERVER_BIN` 时跳过，设了就通过（见 6.3）。

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

环境：macOS arm64；App Server 为仓库外构建的 `dim-app-server` 0.1.0-alpha.18（`/tmp/dim-2-target/debug/dim-app-server`），临时数据目录 + `--mock-model`，不触碰任何用户数据目录。

```bash
DIM_APP_SERVER_BIN=/tmp/dim-2-target/debug/dim-app-server \
  go test -run TestRealDaemonLiveAndHistoryAgree -v -count=1 ./internal/projection/
```

观察到的输出：

```text
服务端 0.1.0-alpha.18（protocol 3，profile mock，owned=true）
会话 1165b29c-5670-42c0-b91a-9aec9f70bfdd
首屏: head=1 events=1 through=1 done=true
提交成功: 输入事件 65d0cb98-...（序号 5）
提交成功: 输入事件 bdf0e1f4-...（序号 18）
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
| 四类未完成事项的真实读取 | blocked | 需要真实会话上出现排队输入、提问与后台任务；本版按 wire 形状自造 fixture 并逐条结算 |
| 分支会话的真实证据 | blocked | 需要在真实服务端上开分支并读分支历史 |
| Windows 平台 | blocked | 没有 Windows 主机 |

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

## 8. 下一步

1. 接真实模型跑一次带工具调用与思考的会话，补 6.5 的 blocked 项（工具行、思考行、审批）。
2. 旧键目标兼容（`goal:objective`/`goal:status`）与目标判定者（evaluator）例外规则。
3. 与 D3 对齐导出：导出按轮读取全部历史，本包的 `ApplyHistory` 已可按页喂入，需要补按轮切分。
4. 与 D4 对齐"待答提问"的语义边界：本包只维持清单，提问的候选项解释与应答顺序归 D4。
5. 把 3.9 的冻结表交给 V 做成 fixture 词表，服务端词表变化时先改表再改实现。
6. 等 D4 接上别名后，跑一次真实导出落盘，把导出文件的轮次与该会话的轮索引逐条对照。
