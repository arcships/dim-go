# Go Projection Golden Fixture 合同

Status: active（核心 timeline 和 input blocks 已实现并接入 session/viewmodel/UI；真实运行与非核心投影仍未验收）
Last updated: 2026-11-09
Owner: projection

本文冻结 `dimagent-mygo` 的纯 projection 语义和 golden fixture 格式。Go projection 是可丢的 view state，不是 session/message/run/approval 的事实存储；Rust `dim-app-server` 仍是唯一事实源。fixture 只描述输入 raw event 与期望输出，不能读取 App Server 的 SQLite/JSONL/session 文件或 `daemon.log`。

## 1. 证据与范围

本合同基于 `dim-electron` 既有实现的只读调研（合同证据等级 `source-read`）。核心 timeline 已按本合同实现在 `internal/projection`，并由 14 个 golden fixtures 在 `headless` 等级验证；其中 `user-content-blocks` 和 `shell-input-raw-block` 于 Wave 2 补齐。real-daemon 当前只有控制面 smoke 证据，不覆盖 projection data shape。

| 语义 | 证据 |
|---|---|
| history 与 live 都调用同一 fold，按 sequence 排 history | `packages/desktop/packages/electron-main/src/app-server/timeline-fold.ts:389-445` |
| sequence 幂等、更新时钟和 branch fold | `timeline-fold.ts:511-523` |
| user input ID、环境 raw block 隔离、consumed 完成 | `timeline-fold.ts:628-670`；`packages/desktop/packages/electron-main/tests/unit/timeline-fold.test.ts:83-156` |
| input 文本优先级和 shell-input raw block 解码 | `timeline-fold.ts:289-335` |
| assistant ID 为 `a:<branch>:<turn>:<request>` | `timeline-fold.ts:718-735`；`timeline-fold.test.ts:158-193` |
| text/thinking 按工作边界形成 block，block ID 使用首个 raw event ID | `timeline-fold.ts:738-823`；`timeline-fold.test.ts:195-224` |
| tool 以 call ID 配对，start/use 合并，result 结束生命周期 | `timeline-fold.ts:825-938`；`timeline-fold.test.ts:226-298` |
| message `orderKey` 为 zero-padded sequence | `timeline-fold.ts:1070-1091`；共享 message 形状见 `packages/desktop/packages/shared/src/protocol/rpc.ts:3332-3353` |
| RawEventBridge 的 live 生命周期（run/message/text/thinking/tool） | `packages/desktop/packages/electron-main/src/app-server/raw-event-bridge.ts:59-169`；`packages/desktop/packages/electron-main/tests/unit/raw-event-bridge.test.ts:14-54,175-226` |
| approval/question、model progress 不进入 TimelineFold | 本仓库 [development-plan.md](development-plan.md) 的 Milestone 3 |
| Go 实现与 golden runner | `internal/projection/projection.go`、`transient.go`、`fixture_test.go`、`testdata/projection/*.json` |

`TimelineFold` 的现有代码还包含 goal、usage、branch、exec/background task 等事实投影；本文件先冻结核心 timeline（input、text、thinking、tool），其余能力在各自 fixture 扩展中沿用相同 reducer 和时钟规则。

## 2. 核心模型与不变量

### 2.1 IDs

- raw event 的 `id` 是去重和 provenance ID，fixture 中必须稳定且唯一；重复 `id` 不足以替代 sequence 去重测试。
- user message ID = `RawEvent.id`（`input/message`）。
- assistant message ID = `a:<branchId>:<turnId>:<requestId>`；同一 turn/request 永远只有一条 assistant message。
- tool message ID = `callId`；tool event ID 固定为 `tool:<callId>:call`（call）和 `tool:<callId>:result`（result），与结果 raw event 的 provenance `id` 分离。
- text/thinking block ID = 该连续 block 首个 raw event 的 `id`；后续 delta 只追加 content，不产生新 block。
- ID 生成必须纯函数、与输入顺序（history 先或 live 先）无关；不得使用随机数或本地持久化 ID。

### 2.2 orderKey 与顺序

- message `orderKey = decimal(sequence).padStart(20, "0")`，例如 sequence `7` → `"00000000000000000007"`。
- `orderKey` 是排序键，不是 message ID；message 按事件首次创建的 sequence 排序。相同 sequence 的输入属于非法 fixture（除非专门验证 fail-fast）。
- history 输入先按 ascending `sequence`；live 输入按到达顺序交给同一个 `Apply`。Reducer 必须对重复事件幂等，并在允许的乱序生命周期事实下最终得到相同 durable snapshot。
- 不使用 wall clock 排序；`create_at` 仅用于 created/updated 和 block end 时间。

### 2.3 text 生命周期

`step/model-start` 创建 assistant（状态 `streaming`）；`step/text`：

1. 关闭当前 running thinking block；
2. 创建或复用当前 running `text_delta` block；
3. 将 `data.delta` 追加到 block 和 assistant `content`。

连续 text delta 共用一个 block；thinking/tool 边界会关闭该 block。`step/model-output-end` 将 message 和所有 running blocks 置为 `complete`（失败映射 `error`，取消映射 `cancelled`）。缺少 `model-start` 时，若事件携带完整 turn/request，reducer 可按同一确定性 ID 隐式创建 assistant；不能创建匿名消息。

### 2.4 thinking 生命周期

`step/thinking` 首个 delta 创建 `thinking` block（`status=running`），随后 delta 追加；text、tool-use/tool-call/tool-use-start 或 model-output-end 关闭当前 thinking。text 后再次 thinking 必须生成新 block（新首事件 ID）。thinking 内容不进入 assistant `content`，只存在 `message.events`。

### 2.5 tool 生命周期

- `step/tool-use-start` 可先创建只有 call ID/name 的 running tool call；后续 `step/tool-use` 的 authoritative name/input 合并到同一 message/event，不得重复创建 call。
- `step/tool-use` 或 `step/tool-call` 创建 tool message（ID=call ID，role=`tool`）和 `tool_call` event（status=`running`）。
- `step/tool-completed` / `step/tool-result` 通过 `call_id` 配对，写入 `tool_result`，并将 tool message/call/result status 结束；结果先于 call 的乱序必须暂存或在 call 到达后收敛，不能生成第二个 tool。
- tool 开始会关闭 thinking；tool 结果不自动拼入 assistant text。
- 未知 tool/event 类型保留 raw 输入可诊断，但不伪造 timeline message。

### 2.6 input 文本与 raw block

`input/message` 的可见文本按 upstream `inputTextFromData` 的优先级提取：

1. `data.user.content` 中 `type: "text"` 的 block 按顺序拼接；image 等其他 block 不贡献 timeline 文本；
2. 没有用户文本时，`data.blocks` 中 `type: "text"` 的 block 按顺序拼接；
3. 否则把 `data.blocks` 中 `type: "raw"` 且匹配 `<dim-shell-input ...>...</dim-shell-input>` 的 body 解码后，用换行拼接。实体只解码一层，按 `&lt;`、`&gt;`、`&quot;`、`&#39;`、`&amp;` 的顺序替换，并去掉首尾空白。

其他 raw block（例如 `<dim-state>`、branch parent）保持模型上下文，不进入 timeline message。没有非 null `user` 且文本为空的 input 不产生 message；不得回退到未文档化的顶层 `data.content` 字段，也不得用本地内容补齐。显式非 null 的 `user` 仍按输入消息处理。

## 3. history/live 共用 reducer

Go 实现必须只有一个 `Reducer.Apply(RawEvent)` 事实入口：

```text
history page: sort(sequence asc) → Apply(event) → Snapshot
live notice:  subscribe first → Apply(event) in arrival order
attach overlap: Apply(event) → sequence/idempotency guard
```

`ApplyHistory` 不能有另一套 history-only 语义。attach 到 projection 就绪之间收到的通知按到达顺序补放；随后 history 重叠事件必须被幂等忽略。至少以 golden case 验证：history→live、live→history、重复同 event、tool result 乱序、text/thinking/tool 交错，最终 durable message IDs/orderKey/events 相同。

现有基线按 `sequence` 去重（`TimelineFold:511-513`）；Go fixture 必须明确采用同一规则，并要求同一 session/branch 的 sequence 唯一。若未来上游改变去重键，应先更新本合同和证据，不在消费端增加猜测别名。

## 4. transient 隔离

approval、question/elicitation、permission request、model progress、connection/reconnect、loading/error banners 是独立 transient projections。它们：

- 不写入 `TimelineFold` durable messages/events；
- 不改变 message ID、orderKey、text/thinking/tool block 边界；
- 可 replay、替换、过期或丢弃；其 fixture 输出放在 `transient`，不放 `messages`；
- approval/question 的响应仍通过 App Server mutation，projection 不自行判定成功。

golden fixture 应包含一个 timeline event 与 approval/model-progress 通知交错的 case，并断言 timeline 输出与“无 transient 通知”完全相同，transient 输出只包含独立状态。

## 5. 时钟注入

Reducer 不得调用 `time.Now()`、`Date.now()` 或依赖测试运行时当前时间。构造函数接收：

```go
type Clock interface { NowMillis() int64 }
```

raw event 的 `create_at` 是 App Server 事实，用于 message `createdAt`/`updatedAt` 和 block `startTime`/`endTime`；clock 只用于本地观察时间（例如 transient received/expiry、缺失时间的诊断）及需要“当前时刻”的 fold。fixture 的 `clock` 必须固定，输出不得含未声明的当前时间。所有时间比较使用 epoch milliseconds；输出 JSON 中时间字段类型和单位固定。

## 6. Golden fixture schema（v1）

文件实际放在 `internal/projection/testdata/projection/*.json`，一文件一个 case（当前 14 个）；schema 版本必须显式：

```json
{
  "schemaVersion": 1,
  "name": "text-thinking-tool-lifecycle",
  "sessionId": "s-1",
  "branchId": "main",
  "clock": { "nowMillis": 1700000000999 },
  "input": {
    "history": [
      {
        "id": "e-1", "sequence": 1, "create_at": 1700000000001,
        "type": "step", "name": "model-start",
        "data": { "turn_id": "t-1", "request_id": "r-1" }
      }
    ],
    "live": [],
    "transient": []
  },
  "expect": {
    "messages": [],
    "timelineUserMessageIds": [],
    "transient": []
  }
}
```

规范：

- 必填：`schemaVersion`, `name`, `sessionId`, `branchId`, `clock`, `input`, `expect`。
- 每个 raw event 必填 `id`, `sequence`, `create_at`, `type`, `name`；`data` 缺省等同空 object，但 fixture 优先显式写出。
- `history` 表示服务端页；`live` 是通知到达序列。测试 runner 必须分别执行两者，也应提供 `historyAndLiveEquivalent` case。
- `input.transient` 只供独立 transient reducer，禁止 timeline reducer 消费；建议形状 `{kind, id, sequence?, payload}`。
- `expect.messages` 使用共享字段的最小稳定子集：`id`, `orderKey`, `role`, `content`, `status`, `createdAt`, `updatedAt`, `events`；events 至少含 `id`, `type`, `status`, `content?`, `startTime`, `endTime?`。无关可选字段不要写入 golden，避免 UI 重构造成噪声。
- `expect.transient` 明确断言 transient projection；若 case 不含 transient，必须为 `[]`，证明 durable 输出没有偷偷承载 transient。
- JSON key 顺序不构成语义；比较器按结构比较，messages/events 按数组顺序比较，maps 按 key 比较。

## 7. 必备 fixture 矩阵

1. user input → consumed：sending→complete，环境 state/raw block 不出现在 messages。（`user-input-consumed.json`）
2. input content blocks：`user.content` 的多个 text block 拼接，image 不贡献文本，环境 raw block 隔离。（`user-content-blocks.json`）
3. shell-input raw block：单个/多个 `<dim-shell-input>` body 解码并用换行拼接，只有已记录输入进入 timeline。（`shell-input-raw-block.json`）
4. one assistant：text 连续 delta；thinking→text→thinking 分成三个 block，首 event ID 稳定。（`text-thinking-blocks.json`、`two-assistants-interleaved.json`）
5. tool：tool-use-start→tool-use→tool-completed；call ID 配对、input 合并、result 完成。（`tool-lifecycle.json`、`tool-result-before-call.json`）
6. duplicate/overlap：同一 sequence 重放不产生第二条 message/block。（`duplicate-sequence-idempotent.json`）
7. history/live permutation：相同事实以分页和通知两种路径输入，snapshot 相等。（`history-live-equivalence.json`，另有 `live-attach-mid-text`、`live-attach-overlap` 子 case）
8. transient isolation：approval/question/model progress 交错但 durable snapshot 不变。（`transient-isolation.json`）
9. injected clock：重复运行得到完全相同 JSON；不允许 wall-clock 泄漏。（`clock-determinism.json`）
10. branch isolation：不同 `branchId` 生成不同 assistant ID/order scope，不串消息。（`branch-isolation.json`）

## 8. 验收与 blocker

验收命令：`go test ./...`、`go vet ./...`；projection 专项用 `go test -run TestGoldenFixtures -v ./internal/projection` 运行并逐个报告 fixture 名称。

当前状态：Go projection、Go `RawEvent` 类型和 golden runner 已实现；`internal/session` 已按 branch 持有独立的 durable/transient reducer，并把 history 与 live 事件送入同一个 `Apply`；`internal/viewmodel` 把 projection snapshot 映射成 native 行（user/assistant/tool），只保留有界预览，`internal/integration` 再把 supervisor/session/viewmodel 组合成会话桥；`go test -count=1 ./...`、`go test -race -count=1 ./...`、`go vet ./...`、`gofmt -l .` 通过，14 个 fixture 全部通过（`headless`）。

剩余 blocker：

- `internal/session` 和 `internal/viewmodel` 已组合 stream、history、projection 和行映射，并已接到 UI；submit/live 端到端和真实运行验收仍未验证；
- goal、usage、branch directory、exec/background task 等非核心投影未实现，本合同只冻结 input/text/thinking/tool；
- 上游 wire `RawEvent` 的完整 data 形状仍以 `dim-electron` protocol v3 和 Rust App Server 为准；real-daemon 控制面 smoke 已通过，但不覆盖 projection data shape，因此不能宣称 real-daemon projection 语义通过；
- approval/question/permission 只有通用 transient notice fixture（`transient-isolation.json`）；`approval.respond` 已接到 UI 决策面板，但 replay 集合替换未实现；
- 同一 sequence 不同 id 的冲突已由 `SessionStream` 在路由层报告为协议错误并丢弃第二个事件；projection reducer 仍按 sequence 丢弃重复项，fixture 会拒绝该输入；上游语义仍待确认。
