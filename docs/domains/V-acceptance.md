# V 验收设施（假服务端、固定样例与验收口径）

Status: 第一版实现完成（P1 可用；P1 三条集成路径已跑通）
Owner: V 领域 agent
依据: `docs/domains/R.md` 第 4 部分验收口径；`docs/domains/A.md` 的连接状态机与四条总规矩；`docs/domains/D1-connection.md` 第 6.4 节把"版本不兼容"分支交给 fake；`docs/project-organization.md` §4.1 与 §6 F6。
范围: 可编程假 App Server、故障注入、固定样例、验收脚本与证据记录。不含界面、真实服务、生产代码路径。
自包含: 需要真实服务才能得出的结论不在本文；假服务端与固定样例只用于路径验证与回归，不能替代真实服务证据。

---

## 0. 结果包

| 项 | 内容 |
|---|---|
| 领域 | V 验收设施 |
| 输入 | `docs/domains/R.md` 第 4 部分验收口径；`docs/domains/A.md` 的连接状态机与四条总规矩；`docs/domains/D1-connection.md` 的公开类型与订阅接口；`docs/project-organization.md` §4.1、§6 F6 |
| 产物 | `internal/fakeappserver`（可编程假服务端、故障注入、样例加载、验收测试）；`testdata/history-basic.json`、`testdata/notifications-basic.json`；本文 |
| 验收 | `go test ./internal/fakeappserver/...`、`go vet ./internal/fakeappserver/...`、`go test -race ./internal/fakeappserver/...`；三条集成路径与注入断言见第 4 节 |
| 依赖 | D1 保持 `protocol`、`transport` 的公开类型稳定；D2 冻结事件词表与翻页游标后扩展历史样例；D3/D4/D5 在 `protocol` 定义目录/提交/停止/审批等方法形状后收紧脚本回执 |
| 上下文需求 | 各领域把需要的固定样例与故障场景提给 V；组织侧确认 F6 的样例格式与接口 |

---

## 1. 产物

### 1.1 包结构

| 文件 | 职责 |
|---|---|
| `internal/fakeappserver/doc.go` | 包说明与三行用法 |
| `internal/fakeappserver/script.go` | `Script`、`Session`、`Reply`、`Fault` 与脚本默认值 |
| `internal/fakeappserver/server.go` | 监听本地 socket、按脚本应答、记录收发帧、主动推送、断线控制 |
| `internal/fakeappserver/fixture.go` | 固定样例的加载与结构校验 |
| `internal/fakeappserver/testdata/history-basic.json` | 历史事件序列样例（两轮 6 条） |
| `internal/fakeappserver/testdata/notifications-basic.json` | 通知序列样例（重复、乱序旧序号、未知方法） |
| `internal/fakeappserver/fakeappserver_test.go` | 用真实客户端跑通的验收测试 |

只依赖标准库与 `internal/appserver` 的公开类型：

- 协议帧一律由 `internal/appserver/protocol` 编解码；
- 端点类型用 `internal/appserver/transport`；
- 验收测试用 `daemon`、`session`、`failure` 的公开 API，不接触内部实现。

### 1.2 对外接口（F6）

```go
server, err := fakeappserver.Start(fakeappserver.Script{
    Features: fixture.Features,
    Sessions: []fakeappserver.Session{fixture.Session()},
})
defer server.Close()
supervisor, err := daemon.New(daemon.Config{DataDir: server.DataDir()})
```

`Start` 在 `<临时目录>` 下建数据目录并监听 `<dataDir>/rust/daemon.sock`，与真实客户端解析出的端点规则一致；`Close` 删除目录与 socket。需要自己管理目录时用 `StartIn(dataDir, script)`，`Close` 只删 socket 不删目录。

## 2. 脚本指令

### 2.1 `Script`

| 字段 | 作用 |
|---|---|
| `InstanceID` | hello 回执的实例标识；空时固定为 `fake-instance` |
| `ServerVersion` | hello 回执的服务端版本；空时 `fake` |
| `ProtocolVersion` | hello 回执的协议版本；0 表示当前版本，设为其他值即注入版本不符 |
| `Profile` | hello 回执的服务端画像；零值按 `mock` |
| `Features` | hello 声明的可选能力；缺项按能力缺失处理 |
| `HelloError` | 非空时 hello 直接回错误回执，用于 `appError=protocol_version` |
| `Sessions` | 会话固定事实（历史、头部、运行状态、最新一轮起点、attach 重放） |
| `Replies` | 按方法名给出的脚本回执；同名后者覆盖前者 |
| `Faults` | 故障注入规则（见 2.3） |
| `StopOnShutdown` | 真时 `daemon.shutdown` 回执后停止监听并断开全部连接 |

`Session` 字段：`ID`、`BranchID`（默认 `main`）、`Running`、`Head`（0 时取最大事件序号）、`LatestTurnStart`、`Events`、`AttachReplay`。

### 2.2 内置应答

未在 `Replies` 里覆盖时，以下方法由假服务端直接实现；回执形状全部引用 `protocol` 的公开定义：

| 方法 | 回执 |
|---|---|
| `hello` | `protocol.HelloResult`；`HelloError` 非空时改回错误 |
| `daemon.ping` | `protocol.PingResult`（会话数、当前连接数、服务端版本） |
| `daemon.shutdown` | `result: null`；`StopOnShutdown` 为真时随后停止 |
| `session.attach` | `protocol.SessionAttachResult`（会话标识、头部序号、运行状态），并先把 `AttachReplay` 逐条作为 `session.event` 发出 |
| `session.detach` | `result: null` |
| `session.history` | `protocol.SessionHistoryResult`；按 `after`（不含）、`through`（含）、`limit`（默认 200）、`latestTurn` 过滤；`latestTurn` 从 `Session.LatestTurnStart` 开始；`done` 表示该窗口之后没有更多事实 |
| 其他方法 | `appError=unknown_method`（能力缺失） |

尚未在 `protocol` 包定义形状的方法（会话目录、提交、停止等）由调用方在 `Replies` 里给出回执值，本包不复制 wire 类型：

```go
Replies: []fakeappserver.Reply{
    {Method: "session.catalog", Result: map[string]any{"sessions": []any{...}}},
    {Method: "session.submit", Result: map[string]any{"inputId": "input-1", "sessionId": "s1"}},
    {Method: "session.stop", Result: map[string]any{"stopped": true}},
}
```

错误回执用 `fakeappserver.AppError(code, appError, message)` 构造。脚本回执与内置应答都可以被 `Faults` 拦截（见下）。

### 2.3 故障注入

| 字段 | 注入的故障 |
|---|---|
| `Method` + `Occurrence` | 命中哪个方法的第几次请求；`Occurrence=0` 表示每次 |
| `Delay` | 应答前等待，制造"后到的请求先回"的乱序回执 |
| `NoReply` | 不回执，让调用方进入超时分支 |
| `DisconnectBefore` / `DisconnectAfter` | 应答前/后断开该连接 |
| `UnknownMethod` | 忽略正常应答，回 `unknown_method`（即使脚本给了回执） |
| `OversizeFrameBytes` | 正常应答前插入一条指定大小的合法帧；客户端需把 `protocol.Options.MaxFrameBytes` 调到该值以下 |

版本不符有两种注入方式：`Script.ProtocolVersion` 给旧版本号，或 `Script.HelloError` 给 `protocol_version` 错误。

断线也可以主动触发，不依赖脚本：

```go
server.DisconnectAll()        // 断开全部连接，继续监听
server.DisconnectStreams()    // 只断 attach 过会话的连接，控制连接不受影响
server.DisconnectSession(id)  // 只断某个会话的订阅
```

### 2.4 主动推送

| 方法 | 用途 |
|---|---|
| `Notify(method, params)` / `NotifyRaw(method, raw)` | 向所有连接推送任意通知 |
| `PushEvent(sessionID, event)` | 追加事实到历史，并向已 attach 该会话的连接推送 `session.event` |
| `PushEventTimes(sessionID, event, n)` | 同一事实连发 n 次，注入重复通知 |
| `AppendEvent(sessionID, event)` | 只追加历史并推进头部，不推送；用于构造"断线期间已发生、重连后重读才能拿到"的事实 |

### 2.5 记录与观察

| 方法 | 返回 |
|---|---|
| `Frames()` | 全部收发帧（方向、连接号、方法、标识、原始字节），按发生顺序 |
| `ReceivedFrames()` | 只含客户端发来的帧；坏帧与客户端通知也记录 |
| `Requests()` / `RequestsFor(method)` | 已解析请求（方法、参数、标识、第几次出现） |
| `Methods()` | 收到的方法名序列（含重复），用于核对请求次数 |
| `Accepted()` / `Connections()` | 累计接受连接数 / 当前跟踪连接数 |
| `AttachedSessions()` | 当前 attach 过的会话标识 |
| `Head(sessionID)` | 会话当前头部序号 |

## 3. 固定样例

样例都是 JSON，用 `description` 说明整份样例验证什么，用每条记录的 `comment` 说明这一条验证什么。加载时校验会话标识、序号严格递增、`head` 不小于最大序号、`latestTurnStart` 必须是某条事件的序号、通知的 `expect` 只能是 `delivered` 或 `dropped`。

### 3.1 `testdata/history-basic.json`

两轮共 6 条事实，`head=6`，`latestTurnStart=4`，声明 `session.history.latestTurn`。

验证：

1. 声明 `latestTurn` 时初始历史只读最近一轮（4、5、6）；
2. attach 回执的头部序号等于 `head`；
3. 断线后按 `after` 游标重读只拿游标之后的条目，不重复；
4. `through`/`limit` 过滤不改变事件顺序。

`latestTurnStart` 是样例元数据，不用事件类型猜轮次边界：事件词表尚未冻结，样例里的 `type`/`name` 只是占位。

加载后 `fixture.Session()` 直接得到 `Session`；`fixture.LatestTurnSequences()` 给出最近一轮的序号，测试用它当期望值。

### 3.2 `testdata/notifications-basic.json`

与上一份样例的序号对齐（初始历史覆盖到 6），依次发送：

| 通知 | 期望 | 验证 |
|---|---|---|
| `session.event` 序号 7 | delivered | 实时事实正常交付 |
| 重复的序号 7 | dropped | 已交付序号挡掉重复 |
| 乱序到达的序号 6 | dropped | 序号不倒退 |
| `session.event` 序号 8 | delivered | 后续事实继续交付 |
| `vendor.unknown` | delivered | 未知方法原样交给上层 |

测试按 `expect` 逐条断言，最后确认没有样例之外的交付。

### 3.3 新增样例的约定

- 历史样例不改已有序号；新增一轮时同时更新 `head` 与 `latestTurnStart`；
- 通知样例的序号必须与历史样例的头部衔接，否则"旧序号被丢弃"的期望不成立；
- 每条记录都写 `comment`，说明它想验证什么；
- 样例只表达"这个假服务端会这样发"，不代表服务端事件词表已经冻结。

## 4. 验收

### 4.1 命令与结果

```bash
go test ./...
go vet ./...
go test -race ./internal/fakeappserver/...
```

本版结果（macOS arm64，Go 1.24）：

- `go test ./...` 全部 `ok`，其中 `internal/fakeappserver` 为 `ok`；
- `go vet ./...` 无输出；
- `go test -race -count=5 ./internal/fakeappserver/...` 为 `ok`，无数据竞争。

本领域测试逐条：

```text
--- PASS: TestNormalHandshakeAndSession (0.00s)
--- PASS: TestDisconnectAndReconnect (0.12s)
--- PASS: TestVersionMismatchStopsRetry (0.00s)
    --- PASS: TestVersionMismatchStopsRetry/hello_回执版本号不符
    --- PASS: TestVersionMismatchStopsRetry/hello_以_appError_拒绝
--- PASS: TestOutOfOrderRepliesAreCorrelated (0.30s)
--- PASS: TestOversizedFrameIsDroppedAndConnectionContinues (0.00s)
--- PASS: TestPushEventTimesInjectsDuplicateNotification (0.11s)
--- PASS: TestNotificationFixtureDuplicateAndOutOfOrder (0.15s)
--- PASS: TestInjectedUnknownMethodOverridesScriptedReply (0.00s)
```

### 4.2 三条集成路径

全部用真实客户端 `daemon.Supervisor` + `session.Manager`，只把服务端换成假服务端。

| 路径 | 测试 | 断言 |
|---|---|---|
| 正常握手与会话 | `TestNormalHandshakeAndSession` | hello 协议版本与能力；控制连接与会话流各一次 hello；目录、提交、停止按脚本回执；attach 头部等于 6；初始历史只读最近一轮；主动通知到达；最后一个持有者释放后 detach；收发帧都有记录 |
| 断线后重连 | `TestDisconnectAndReconnect` | 实时事实 7 交付后断开会话流；重同步起点为 7；按 `after=7` 重读只拿到断线期间的 8；重连后实时事实 9 到达；7 与 8 不重复出现在实时通道 |
| 版本不符停止重试 | `TestVersionMismatchStopsRetry` | 回执版本号不符、`appError=protocol_version` 两种注入都归为 `version_incompatible`；第二次连接不再拨号（接受连接与 hello 请求都只有 1 次） |

### 4.3 注入项的断言

| 注入 | 测试 | 断言 |
|---|---|---|
| 乱序回执 | `TestOutOfOrderRepliesAreCorrelated` | 先发的历史请求被延迟 300ms，后发的 ping 先回；客户端按标识关联，两次回执各自正确，服务端记录的到达顺序为 history、ping |
| 超大帧 | `TestOversizedFrameIsDroppedAndConnectionContinues` | 客户端上限 1024 字节，收到 4096 字节帧后报 `protocol.ErrFrameTooLarge`（字节数大于上限）；同一连接上的 hello、主动通知与 ping 都照常完成 |
| 重复通知 | `TestPushEventTimesInjectsDuplicateNotification`、`TestNotificationFixtureDuplicateAndOutOfOrder` | 同一事实连发两次只交付一次，头部只推进一次；样例里重复的序号 7 只交付一次 |
| 乱序通知 | `TestNotificationFixtureDuplicateAndOutOfOrder` | 旧序号 6 被丢弃，后续 8 正常交付 |
| 未知方法 | `TestNormalHandshakeAndSession`、`TestInjectedUnknownMethodOverridesScriptedReply` | 未声明的方法回 `unknown_method`，分类为不可重试；故障命中时即使脚本有回执也按未知方法返回，下一次恢复正常 |
| 断线 | `TestDisconnectAndReconnect` | 主动断开只影响会话流，重连后按序号续读 |
| 版本不符 | `TestVersionMismatchStopsRetry` | 进入终态，不再拨号 |

### 4.4 证据档位与不可宣称项

- 假服务端与固定样例属于"替身"档位：只能证明客户端在给定输入下的行为，不能宣称"与真实服务可用"。
- 真实服务的事实证据仍由 D1 提供（`docs/domains/D1-connection.md` 第 6.3 节）；本包不替代。
- 本机路径、帧编解码、会话流生命周期可在本档位验证；真实 daemon 版本不符、真实 Windows 命名管道仍为 blocked。

## 5. 依赖与上下文需求

| 对象 | 需要什么 | 现状 |
|---|---|---|
| D1 连接 | `protocol` 的公开类型与 `transport.Endpoint` 保持稳定；新增方法形状先落到 `protocol` | 本版只引用既有公开定义；目录/提交/停止的回执由调用方给，等 D3/D4 冻结后再收紧 |
| D2 会话投影 | 事件词表与向后翻页（`before`）语义冻结后，扩展历史样例与假服务端过滤 | 当前只实现向前游标；`latestTurn` 边界用样例元数据表达 |
| D3/D4/D5 | 需要更多方法形状与通知序列时，提结果包要求，在本包加脚本回执或新样例 | 未开始 |
| 组织侧 | F6 冻结：样例格式与假服务端接口 | 本版可评审 |

## 6. 已知限制

1. 只支持 Unix socket；Windows 命名管道未实现，`Start`/`StartIn` 在 Windows 上返回明确错误。
2. `Start` 用 `<临时目录>` 建数据目录以避开 socket 路径上限；`StartIn` 的完整路径超过 100 字节时直接报错。部分平台上 `<临时目录>` 会解析为更长的真实路径，因此实际路径还要再留余量。
3. 会话目录、提交、停止等方法的回执形状未在 `protocol` 冻结，由调用方用 `Replies` 提供任意 JSON；假服务端不校验形状，也不保证与真实服务端一致。
4. 历史只实现向前游标（`after`/`through`/`limit`/`latestTurn`）；`before` 返回参数错误。向后翻页等 D2 冻结语义后再补。
5. `latestTurn` 的轮次边界由样例元数据 `latestTurnStart` 给出，不解析事件类型；事件词表变化时需要同步样例与实现。
6. 超大帧注入需要调用方把客户端 `MaxFrameBytes` 调到注入帧以下；经 `daemon.Supervisor` 的默认 16 MiB 上限下不会触发，因此该断言用 `transport.Dial` + `protocol.NewClient` 直接构造。
7. 不模拟多客户端语义：`daemon.ping` 的连接数只反映假服务端的当前连接，审批、权限、账号等业务语义都没有。
8. 收发帧与请求记录无上限，长时间运行会持续占用内存；它面向测试与单次验收，不适合作为长驻服务。
9. `daemon.shutdown` 不模拟 `ifIdle` 的忙判定；需要该分支时用脚本回执 `daemon_busy` 或在测试里注入错误。
10. 断线注入按"关闭连接"实现，没有模拟半包、丢包、网络延迟抖动；乱序只覆盖回执延迟与通知序号。

## 7. 下一步

1. 等 D2 冻结事件词表与翻页游标，补齐 `before` 与按轮样例。
2. 等 D3/D4/D5 在 `protocol` 定义目录、提交、停止、审批等方法形状，把 `Replies` 的任意 JSON 收紧为类型化回执。
3. 需要时增加"乱序通知调度"（按序号故意倒序发送）与"半包/慢写"注入，用于更细的连接层验收。
4. 把各领域的固定样例需求收进本包 `testdata`，保持"一个样例只说明一类事实"。
