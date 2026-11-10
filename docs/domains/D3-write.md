# D3 写登记（在途写、一次性发送、未知核对、草稿与待发）

Status: active（第一版实现完成，待组织评审）
Owner: D3 写登记领域
依据: `docs/domains/R.md` 功能块 3、6、7、8、9、10、12 的写入侧与 R1/R8；`docs/domains/A.md` 的「在途写」「草稿」「四条总规矩」；`docs/domains/D1-connection.md` 的错误分类与发送接口；`docs/domains/D2-projection.md` 只读核对可用视图；`docs/domains/D8-attachments.md` 的暂存与引用约定。
范围: 出站写意图的登记、一次性发送、结果收敛、未知核对、草稿与待发保留、在途表与状态通知。不含界面、连接实现、协议 wire 定义、投影内部、写入内容的业务校验、任何持久化。

---

## 1. 结果包

**领域**：D3 写登记

**输入**

| 输入 | 来源 | 本版使用方式 |
|---|---|---|
| 功能块 3「发送与运行控制」 | `docs/domains/R.md` | 提交、投递方式、停止、失败保留、结果未知核对、幂等 |
| 功能块 6/7 的写入侧 | `docs/domains/R.md` | 审批应答、长期允许、提问应答各作为一次写意图；顺序与一次一条由 D4 负责 |
| 功能块 8/9/10/12 的写入 | `docs/domains/R.md` | 管理写入、账号动作、附件资源写入作为同一种受控写入 |
| 功能块 5 的会话生命周期写入 | `docs/domains/R.md` | 新建、改名、删除、开分支、结束分支用 `KindSessionLifecycle`，参数里的 `op` 区分动作 |
| 功能块 13 的会话内命令写入 | `docs/domains/R.md` | 执行与停止用 `KindShellRun`，`op` 取 `run`/`stop`；`!` 由服务端执行，不是本机终端 |
| 功能块 9 的模态任务与实时传输写入 | `docs/domains/R.md` | 提交、取消、打开、关闭用 `KindModalityWrite`，`op` 取 `task.submit`/`task.cancel`/`realtime.open`/`realtime.close` |
| 在途写状态机、草稿、四条总规矩 | `docs/domains/A.md` | 状态迁移逐条对应；草稿与待发归属本领域；启动时校验迁移表 |
| 错误分类 | `docs/domains/D1-connection.md`、`internal/appserver/failure` | 超时与连接中断转未知；能力缺失、参数错误、连接缺失按明确失败 |
| 只读核对视图 | `docs/domains/D2-projection.md` | 由调用方实现注入的 `Inspector`：按输入标识查归属、运行状态、历史是否已含该输入 |
| 附件暂存语义 | `docs/domains/D8-attachments.md` | 本包只承载资源写入意图与参数，不释放暂存引用、不解析附件内容 |
| R1 默认投递方式 | `docs/domains/R.md` | 空投递方式按排队（queue） |
| R8 本地状态边界 | `docs/domains/R.md` | 幂等键、草稿、在途表都只在内存；重启不补发 |

**产物**

| 路径 | 职责 |
|---|---|
| `internal/write/write.go` | 对外类型与接口：意图、请求、回执、核对读取、视图、变化通知、草稿、配置、管理器 |
| `internal/write/state.go` | 「在途写」迁移表、查表与启动校验 |
| `internal/write/verify.go` | 未知核对的三选一判定与归属一致性 |
| `internal/write/write_test.go` | 注入假发送方与假核对接口的核心行为测试 |
| `internal/write/appserver_test.go` | 用 `fakeappserver` 与 D1 连接走真实 NDJSON 的一次发送、超时、断线、明确失败与重启测试 |
| `internal/write/state_test.go` | 迁移表与 A.md 逐条对应、坏表拒绝、核对判定表 |
| `docs/domains/D3-write.md` | 本文件 |

**验收**

- 命令：`go test ./...`、`go vet ./...`、`go test -race ./internal/write/`；本版全部通过（2026-11-10，macOS arm64）。
- 本领域测试：34 个顶层测试、68 个子测试全部通过；覆盖见 §6.2。
- 证据档位：
  - `fakeappserver`：本机 Unix socket 上真实 NDJSON 收发，覆盖一次发送、无回执超时、回执前断线、能力缺失、重启不补发。
  - 注入替身：`Sender` 与 `Inspector` 为测试替身，覆盖核对命中/未发生/无法确认/归属不一致。
  - 真实 App Server 的提交、停止、审批、提问与会话生命周期写入：**blocked**，缺 D1/D4 等域冻结的具体 wire 方法适配与真实服务端事实。

**依赖**

| 依赖 | 需要对方冻结/提供 |
|---|---|
| D1 连接 | `Sender` 适配器：把 `write.Kind` 映射到具体方法，只调用一次、不自行重试、把错误原样返回，让 `failure.KindOf` 可分类；提供 `session.input_lookup` 对应的核对读取所需事实 |
| D2 会话投影 | `Inspector` 实现：按输入标识回答 `Present`、实际 `SessionID`/`BranchID`、加载是否可信；运行状态可由 D1 的 attach 结果补入 |
| D4 交互与审批 | 长期允许先写后答：只有本包对长期允许意图给出成功，才登记审批应答；写入失败时不得发出应答；提问一次只登记一条 |
| 会话目录与分支领域 | 新建/改名/删除/开分支/结束分支用 `KindSessionLifecycle`；`op` 可用 `create`/`rename`/`delete`/`branch.create`/`branch.end`，会话动作兼容现有的 `catalog.*` 前缀；新建会话给客户端生成的 `InputID` 供核对；`branch.end` 的分支标识放在 `Intent.BranchID` 或参数 `branchId` |
| 会话命令执行领域 | 执行/停止用 `KindShellRun`；`op` 取 `run`/`stop`，`run` 带命令、`stop` 带 `runId`；建议把 `runId` 也填进 `Intent.InputID` 供未知核对 |
| 模态能力领域 | 任务提交/取消与实时传输开/关用 `KindModalityWrite`；`op` 取 `task.submit`/`task.cancel`/`realtime.open`/`realtime.close`；建议把 `taskId`/`connectionId` 或客户端请求标识填进 `Intent.InputID` 供未知核对 |
| D5 管理面、D6 账号、D8 附件 | 各自提供稳定的幂等键；附件资源写入的键应包含内容摘要与会话，保证重复挂载只写一次 |
| U1 原生界面 | 只读 `IntentView`/`Change`，用 `SetDraft`/`Draft` 保存草稿，不直接改状态 |

**上下文需求**

- 需要 D1 明确各写意图对应的方法名、参数形状与回执里的输入标识字段；本版发送方由调用方注入，包内不复制 wire 定义。
- 需要 D2 明确「输入标识」在折叠视图里如何定位：用户行标识、分支桶、是否已落库；本版只约定读取结果的语义，不依赖 D2 内部结构。
- 需要用户确认核对次数与间隔的默认值（本版 3 次 × 1 秒，单次读取 5 秒；R 附 B 的 Q-3 仍未定数值，已配置化）。
- 需要 D4/D8 确认失败、未知、未发生、需人工确认四种结果下附件暂存引用的释放时机；本包只保留参数，不释放。

## 2. 边界

**做**

| 能力 | 说明 |
|---|---|
| 意图登记 | 十一类意图；目标会话与分支、投递方式、输入标识、透明参数 |
| 一次性发送 | 每条意图只在登记后发送一次；发送在登记之后、异步进行 |
| 结果收敛 | 明确成功、明确失败、超时/断线转未知 |
| 未知核对 | 按输入标识查归属与历史；命中且归属一致才成功；确认未发生允许重提；无法确认按上限再核对 |
| 草稿与待发 | 草稿按会话保存；失败与未知保留待发内容；成功清空内容并只清空匹配的草稿 |
| 观察 | 在途表、全表、单条结果、等待终态、状态变化订阅 |
| 重启语义 | 幂等键只在内存；新实例没有旧意图，不补发任何请求 |

**不做**

| 不做 | 归属/原因 |
|---|---|
| 界面、列表渲染、按钮状态 | U1 |
| 连接、socket、NDJSON、方法名与 wire 类型 | D1 |
| 历史折叠、输入归属的具体读取实现 | D2 |
| 审批与提问的先后顺序、一次一条 | D4 |
| 写入内容的业务校验（文本长度、附件上限、参数含义） | 调用方；本包只校验结构 |
| 附件的选择、暂存、上限与引用释放 | D8 |
| 任何持久化、磁盘上的幂等表、重启恢复 | R8 与 A.md：幂等标识只存在内存 |
| 自动重发、自动重试写入 | A.md：结果未知先核对，不自动重发 |

本包只使用 Go 标准库与仓库内的连接错误分类包，不引第三方依赖，不改 `go.mod`。

## 3. 包结构与对外接口

```text
internal/write/
├── write.go          对外类型、Config、New、Register、Result、InFlight、Table、Wait、
│                     Subscribe、SetDraft、Draft、ClearDraft、Close
├── state.go          「在途写」迁移表、transition、启动校验
├── verify.go         核对判定与归属一致性
├── write_test.go     假发送方/假核对接口的核心测试
├── appserver_test.go fakeappserver + D1 连接的一次性发送、超时、断线、失败、重启
└── state_test.go     迁移表与核对判定测试
```

对外接口：

```go
manager, err := write.New(write.Config{
    Sender:      sender,      // 必填：只发一次，不重试
    Inspector:   inspector,   // 必填：只读核对
    SendTimeout: 30 * time.Second,
    Verify:      write.VerifyPolicy{MaxAttempts: 3, Interval: time.Second, Timeout: 5 * time.Second},
})

view, err := manager.Register(write.Intent{
    Key:       "submit:s1:main:<digest>",
    Kind:      write.KindSubmitInput,
    SessionID: "s1",
    BranchID:  "main",
    Delivery:  write.DeliveryQueue, // 空也按 queue
    Params:    json.RawMessage(`{"text":"你好"}`),
})
view, err = manager.Wait(ctx, view.ID)       // 取得终态结果
view, ok = manager.Result(view.ID)           // 取当前结果
inFlight := manager.InFlight()               // 已登记 + 在途 + 未知
all := manager.Table()                       // 本进程登记过的全部意图
cancel := manager.Subscribe(func(change write.Change) { ... })
manager.SetDraft(write.Draft{SessionID: "s1", Text: "草稿", AttachmentIDs: []string{"stg-1"}})
draft, ok := manager.Draft("s1")
manager.Close()
```

`IntentView` 只含业务可观察字段：意图标识、幂等键、类型、目标会话与分支、投递方式、输入标识、状态、原因、待发内容、成功回执原文、时间。不暴露迁移事件、核对次数、内部归属判断等状态机字段。

## 4. 设计

### 4.1 意图类型

| 类型 | 常量 | 目标会话 | 说明 |
|---|---|---|---|
| 提交输入 | `KindSubmitInput` | 必填 | 可带投递方式；输入标识为空时由本包生成 |
| 停止 | `KindStop` | 必填 | 只停目标会话/分支，作用范围由调用方给 |
| 审批应答 | `KindApprovalResponse` | 必填 | 同意或拒绝；长期允许是另一条意图 |
| 长期允许 | `KindLongTermAllow` | 必填 | D4 必须先让这条成功，再登记审批应答 |
| 提问应答 | `KindQuestionAnswer` | 必填 | 一次只登记一条由 D4 保证 |
| 管理写入 | `KindManagementWrite` | 可选 | 插件、技能、钩子、计划任务等设置 |
| 账号动作 | `KindAccountAction` | 可选 | 登录、登出、重置等 |
| 附件资源写入 | `KindAttachmentWrite` | 必填 | 只承载写入意图；暂存与引用归 D8 |
| 会话生命周期 | `KindSessionLifecycle` | 新建会话不需要，其余必填 | 参数 `op` 取 `create`/`rename`/`delete`/`branch.create`/`branch.end`，会话动作兼容 `catalog.create`/`catalog.rename`/`catalog.delete`；`branch.create` 按指定轮次开分支；`branch.end` 还需要分支标识 |
| 会话内命令执行 | `KindShellRun` | 必填 | `op=run` 必须有命令；`op=stop` 必须有运行标识；`!` 由服务端执行 |
| 模态写入 | `KindModalityWrite` | 必填 | `op` 取 `task.submit`/`task.cancel`/`realtime.open`/`realtime.close` |

结构校验：幂等键非空、类型已知、会话要求满足、投递方式只能是 queue/interject（空按 queue）、参数是合法 JSON。参数内容不解释、不校验。各类型的 op 与必需字段：

- 会话生命周期：参数必须是非空对象；`op` 取上表五个动作或三个目录兼容写法。`create` 不要求 `SessionID`；`rename`/`delete`/`branch.create`/`branch.end` 要求 `SessionID`；`branch.end` 还要求分支标识，可用 `Intent.BranchID` 或参数 `branchId`，两处都给时必须一致，登记时统一归一到 `BranchID`。
- 会话内命令：`op` 只能是 `run`/`stop`；`run` 的 `command` 去掉空白后不能为空；`stop` 的 `runId` 不能为空。
- 模态写入：`op` 只能是 `task.submit`/`task.cancel`/`realtime.open`/`realtime.close`。

`InputID` 是核对用的标识：提交类为空时由本包生成；审批、提问、停止、会话命令与模态写入建议由调用方填入服务端请求标识或运行/任务/连接标识。新建会话没有会话标识，建议由调用方给一个客户端生成的新建标识；没有可核对标识的意图一旦进入未知，只能按核对上限收敛到需人工确认。

### 4.2 状态机

逐条对应 `docs/domains/A.md` 的「在途写」：

| 当前状态 | 事件 | 下一个状态 | 本领域表现 |
|---|---|---|---|
| 已登记 | 发出请求 | 在途 | `Register` 先入表，异步 `Sender.Send` 前查表迁移 |
| 在途 | 明确成功 | 成功 | 回执到达；保留回执原文，清空待发内容与匹配草稿 |
| 在途 | 明确失败 | 失败 | 错误不可重试或连接缺失（请求未发出）；保留内容与草稿 |
| 在途 | 超时、断线 | 未知 | 超时与连接中断可能在发出之后发生；不改内容与草稿 |
| 未知 | 核对命中且归属一致 | 成功 | 输入在历史中，且会话/分支与意图一致；没有回执原文 |
| 未知 | 核对确认没有这次写入 | 未发生 | 终态；允许用新的幂等键重提；内容与草稿保留 |
| 未知 | 核对无法确认 | 未知 | 自环，状态与原因不变，按上限继续核对 |
| 未知 | 核对次数到上限 | 需人工确认 | 终态；不自动重发；内容与草稿保留 |

补充：

- 「已登记 → 未发生」只在关闭时发生：请求还没发出就关闭，视为未发生。
- 「在途/未知 → 需人工确认」也只在关闭时发生：已发出但未结项，按 A.md「重启后一律按需人工确认」处理。
- 启动时校验迁移表：来源与目标都在状态集合内、事件在事件集合内、终态无出边、非终态都有「关闭」出边；不合法即 `New` 失败并点名条目。
- 非法迁移按 A.md 统一处理：丢弃事件、状态不变、不发通知、不向用户报错。

### 4.3 一次性与幂等

1. 每次登记生成一个意图标识；发送协程由 `Register` 启动且只启动一次。
2. 幂等键由调用方给，只在内存中。同一键重复登记返回同一意图，不重复发送；键相同但类型、会话、分支、投递方式或参数不同，返回 `ErrKeyConflict`，不发送。
3. 提交类意图没有输入标识时，本包生成一个并写入 `Request.InputID`；发送方负责把它作为提交去重标识（例如 `submissionId`）下发。服务端回执给出输入标识时以服务端为准。
4. 提交失败或核对确认未发生后允许重提，但必须使用新的幂等键；同一旧键返回旧的终态记录，不会再次发送。这样「同一意图只发一次」与「失败可重提」不冲突。
5. 审批应答与长期允许是两条独立意图；本包不自动串行、不自动重发，顺序由 D4 通过「先等长期允许成功，再登记应答」保证。

### 4.4 发送与错误分类

`Sender.Send` 的契约：

- 只发一次，不自行重试；
- 尊重 `ctx`，返回明确回执或错误；
- 错误原样返回，本包用 `failure.KindOf` 分类。

| 错误分类 | 触发 | 本包处理 |
|---|---|---|
| `timeout` | 请求超时、`context.DeadlineExceeded` | 未知，开始核对 |
| `retryable` | 连接关闭/中断、服务忙 | 未知，开始核对（可能已经写入） |
| `connection_missing` | 端点不存在、连接被拒 | 明确失败（请求未发出） |
| `non_retryable` | 能力缺失、参数错误、未识别错误、版本不符 | 明确失败 |

发送超时由 `Config.SendTimeout` 控制，默认 30 秒；超时只结束等待，不重发。

### 4.5 未知核对

`Inspector` 是唯一核对入口，最小读取接口：

```go
type InputState struct {
    Confirmed bool   // 这次读取是否可信
    Present   bool   // 服务端历史是否已含该输入
    SessionID string // 实际归属
    BranchID  string
    Running   bool   // 目标当前是否在运行，作为上下文
}
```

判定：

1. 读取返回错误或 `Confirmed=false` → 无法确认。
2. `Present=true` 且归属一致 → 成功。归属一致要求：意图声明了会话，则实际会话必须等于它；意图声明了分支，则实际分支必须等于它；读取结果为空视为无法确认。会话生命周期的新建意图登记时没有会话标识，核对允许读到服务端新建出的会话标识；其他无会话意图仍要求读取结果没有会话标识。
3. `Confirmed=true` 且 `Present=false` → 未发生，终态，允许重提。
4. 命中但归属不一致 → 无法确认，继续核对。
5. 每次核对之间等待 `Verify.Interval`，最多 `Verify.MaxAttempts` 次；用尽进入需人工确认。

约定与限制：

- `Running` 只作为核对上下文，本版不单独用它判断某次写是否发生；运行状态不能证明一次停止或应答已到达。
- 只有历史已含该输入且归属一致才算成功；服务端返回的别的输入标识不会被当作本次命中。
- 核对只读，不发送任何写请求，因此核对本身不会造成重复写入。
- `Inspector` 必须只在读取反映确定状态时返回 `Confirmed=true`；首次核对前的间隔用于给服务端结算留出时间。

### 4.6 草稿与待发

- 草稿按会话保存在内存：文本与待发附件暂存标识。`SetDraft` 每次递增该会话的修订号，清空后重新输入也不复用旧修订号。
- 提交意图登记时，若未显式给 `DraftRevision`，自动取当前草稿修订号。
- 成功时：清空该意图的待发内容；仅当会话当前草稿修订号仍等于登记时的修订号，才清空草稿。用户后来输入的新草稿不会被旧提交抹掉。
- 失败、未知、未发生、需人工确认时：待发内容与草稿都保留。未发生与需人工确认的意图需要人工决定是否重提。
- 附件暂存引用不归本包释放；调用方按 D8 的引用计数约定，在成功或明确放弃后再释放。

### 4.7 观察

| 接口 | 语义 |
|---|---|
| `InFlight()` | 已登记、在途、未知的意图，按登记顺序；发送结束后不再出现 |
| `Table()` | 本进程登记过的全部意图，含终态，按登记顺序；供人工核对 |
| `Result(id)` | 单条当前视图；不存在返回 `false` |
| `Wait(ctx, id)` | 等到终态；ctx 结束返回当前视图与原因；关闭时返回 `ErrClosed` |
| `Subscribe(fn)` | 登记、在途、终结时同步回调；未知 → 未知的自环不发通知；取消函数幂等 |

监听器 panic 会被吞掉，不会带走状态机；回调里不要阻塞，也不要回调 `Close`（会等待当前发送协程而自锁）。

### 4.8 重启

- 没有落盘：幂等键、在途表、草稿都只在本进程内存中。
- 新实例启动时表为空，不会补发任何旧请求。
- 旧实例关闭时：还没发出的登记按「未发生」，已发出或未知的按「需人工确认」；这只是内存里的收尾，不跨进程传播。
- 因此「重启后发现已发出」只能由人工通过服务端事实确认；本包不读取服务端存储，也不在重启后自动核对或重发。

## 5. 与 A/R 的对应

| 需求或架构条目 | 本领域实现 | 不在本领域 |
|---|---|---|
| R 块 3 提交与投递方式 | 提交意图、queue/interject、参数透明下发 | 输入编辑、界面 |
| R 块 3 停止 | 停止意图 | 停止后轮次状态的读取（D2） |
| R 块 3 结果未知核对、不自动重发 | 未知状态、注入核对、次数上限、不重发 | 服务端输入查询的实现（D1/D2） |
| R 块 3 发送失败保留内容 | 失败时保留 `Content` 与草稿 | 界面提示 |
| R 块 3 幂等 | 幂等键去重、重新登记返回同一意图 | 服务端侧去重 |
| R 块 6 长期允许先写后答 | 长期允许与审批应答各为一次意图，结果可等待 | 顺序由 D4 保证 |
| R 块 7 提问一次一条 | 提问应答意图 | 队列与推进由 D4 保证 |
| R 块 8/9/10/12 写入 | 管理写入、账号动作、附件资源写入 | 各自的业务参数与预览 |
| R 块 5 会话生命周期 | 会话生命周期意图：新建、改名、删除 | 目录页的选取、命名来源与删除确认 |
| R 块 4 分支生命周期 | 会话生命周期意图的 `branch.create`/`branch.end` | 切点轮次的选取、分支目录与分支内容 |
| R 块 9 模态任务与实时传输 | 模态写入意图：任务提交/取消、实时开/关 | 任务状态、结果读取与实时事件流 |
| R 块 13 会话内命令 | 命令执行/停止意图 | 命令输出折叠、本机终端 |
| A 在途写状态机 | `state.go` 迁移表与启动校验 | — |
| A 草稿归属写登记 | `Draft`、`SetDraft`、成功时按修订号清空 | 界面输入 |
| A 四条总规矩 | 只提交意图、不写事实；状态只有一个驱动者；不做乐观更新 | 投影与界面 |
| A 检查项「写入不重复」 | 一次发送、未知核对不重发、幂等键去重 | 服务端事实核对 |
| A 检查项「一个状态只有一个驱动者」 | 所有迁移只经 `Manager` 内部入口 | — |

## 6. 验收

### 6.1 命令与结果

```bash
go test ./...
go vet ./...
go test -race ./internal/write/
```

本版结果：三条命令全部通过；`internal/write` 的 34 个顶层测试与 68 个子测试全绿，仓库其余已落地包也全部通过（2026-11-10，本机 macOS arm64）。

### 6.2 覆盖

| 验收要求 | 测试 |
|---|---|
| 一次意图只发一次；重复登记返回同一意图 | `TestRegisterSendsOnceAndDuplicateReturnsSameIntent`、`TestAppServerSendsExactlyOnce` |
| 明确成功；成功后清空待发内容与匹配草稿 | `TestRegisterSendsOnceAndDuplicateReturnsSameIntent`、`TestSubmitSuccessClearsOnlyMatchingDraft` |
| 明确失败；失败保留待发内容与草稿 | `TestExplicitFailurePreservesContentAndDraft`、`TestAppServerCapabilityMissingFails` |
| 超时转未知；未知触发核对 | `TestTimeoutGoesUnknownAndVerificationHitSucceeds`、`TestAppServerTimeoutThenConfirmedMiss` |
| 断线转未知 | `TestAppServerDisconnectBecomesUnknownThenNotOccurred` |
| 核对命中且归属一致 → 成功 | `TestTimeoutGoesUnknownAndVerificationHitSucceeds`、`TestJudgeOwnership` |
| 核对确认未发生 → 允许重提 | `TestVerificationConfirmsNotOccurredAllowsResubmit`、`TestAppServerTimeoutThenConfirmedMiss` |
| 核对无法确认 → 继续核对直到上限 → 需人工确认 | `TestVerificationUnconfirmedReachesManualAfterLimit`、`TestOwnershipMismatchReachesManual` |
| 归属不一致不能算命中 | `TestOwnershipMismatchReachesManual`、`TestJudgeOwnership` |
| 未知与需人工确认时保留待发内容 | `TestVerificationUnconfirmedReachesManualAfterLimit`、`TestCloseMarksUnsettledIntentAsManual` |
| 重启后不补发（新实例模拟） | `TestRestartDoesNotResend` |
| 十一类意图都能登记并只发一次 | `TestAllKindsAreSentOnce`、`TestSessionLifecycleIntentIsSentOnce`、`TestShellRunIntentIsSentOnce`、`TestModalityWriteIntentIsSentOnce` |
| 会话生命周期的作用域：新建不要求会话，改名/删除/开分支/结束分支要求会话，op 必须合法 | `TestSessionLifecycleScope` |
| 分支结束的分支标识：`BranchID` 与参数 `branchId` 二选一，冲突拒绝 | `TestSessionLifecycleIntentIsSentOnce`、`TestSessionLifecycleScope` |
| 分支生命周期 op 的未知核对与失败保留 | `TestSessionLifecycleBranchOpsUnknownAndFailure` |
| 会话生命周期未知核对：新建允许读到服务端新会话标识 | `TestSessionLifecycleUnknownVerification`、`TestJudgeOwnership` |
| 会话生命周期失败保留待发内容与草稿 | `TestSessionLifecycleFailurePreservesContentAndDraft` |
| 会话命令执行/停止：只发一次、未知核对、失败保留 | `TestShellRunIntentIsSentOnce`、`TestShellRunUnknownVerification`、`TestShellRunFailurePreservesContentAndDraft` |
| 模态写入：四种 op 各只发一次、未知核对、失败保留 | `TestModalityWriteIntentIsSentOnce`、`TestModalityWriteUnknownVerification`、`TestModalityWriteFailurePreservesContentAndDraft` |
| 会话命令与模态写入的作用域与 op 校验 | `TestShellRunAndModalityScope` |
| 结构校验与幂等键冲突 | `TestInvalidIntentsAreRejected`、`TestKeyConflictIsRejected` |
| 在途表、全表、单条结果与状态变化通知 | `TestInFlightTableAndChangeNotifications` |
| 关闭后拒绝新意图；未结项按需人工确认 | `TestClosedManagerRejectsNewIntents`、`TestCloseMarksUnsettledIntentAsManual` |
| 迁移表与 A.md 逐条对应；坏表启动失败 | `TestTransitionTableMatchesDocument`、`TestTransitionTableIsValid`、`TestBrokenTransitionTablesAreRejected` |
| 核对判定的九种输入（含生命周期新建与其他无会话意图） | `TestJudgeOwnership` |

证据档位：`fakeappserver`（本机 Unix socket + NDJSON）与注入替身；真实 App Server 的写语义未验证。

### 6.3 与 D1/D2 的接口形状

| 提供方 | 形状 | 本包消费方式 |
|---|---|---|
| D1 连接 | `protocol.Client.CallInto(ctx, method, params, out)` | 由集成层实现 `Sender`：按 `Kind` 选方法、合入会话/分支/投递方式/输入标识、只调用一次、错误原样返回 |
| D1 错误分类 | `failure.KindOf(err)` | 超时与 retryable → 未知；connection_missing 与 non_retryable → 失败 |
| D1 会话流 | `session.Snapshot.Attach.Running`（attach 回执运行状态） | 由集成层填入 `InputState.Running`；本包只作上下文 |
| D2 投影 | 行标识、分支桶、已落库位置、装载阶段 | 由集成层实现 `Inspector`：输入标识命中、归属、读取是否可信 |
| D2 历史 | 用户行标识等于输入标识 | `Present` 的来源；本包不直接读 D2 内部 |

本包不导入 D1/D2 的业务包（只导入错误分类包），因此两边的实现可以独立演进；集成层负责把两侧拼成 `Sender` 与 `Inspector`。

### 6.4 blocked

| 项 | 状态 | 缺什么 |
|---|---|---|
| 真实 App Server 上的提交/停止/审批/提问/会话生命周期/会话命令/模态写入 | blocked | D1/D4/D5/D6/D8/D9/D11a 尚未冻结各写意图的方法名与参数；本版发送方由调用方注入 |
| 真实 `session.input_lookup` 核对 | blocked | 需要真实服务端与稳定回执形状；本版只用注入的核对替身 |
| 跨进程重启的人工确认 | blocked | R8 明确不落盘；需要服务端事实与人工流程，不属于本包 |
| 附件资源写入去重 | blocked | 需要 D8 提供稳定内容摘要键与真实资源写入方法 |

## 7. 已知限制

1. **不持久化**：重启后旧意图、幂等键、草稿全部丢失；本包不补发，也无法在新实例里把旧意图标成「需人工确认」。人工确认依赖服务端事实与外部流程。
2. **终态记录不清理**：`Table` 保留本进程所有意图，长时间运行会线性增长；第一版没有淘汰策略。
3. **核对策略单一**：所有类型共用固定间隔与同一上限；`Running` 只作上下文，不参与判定。停止类意图的核对需要集成层把「已停止」映射成可判定的输入事实，否则只能进入人工确认。
4. **归属判定严格**：意图声明了会话/分支时，读取结果必须完全一致；读到空归属视为无法确认，不会乐观算成功。
5. **Inspector 必须由集成层实现**：D1 只提供连接与错误分类，D2 只提供折叠视图；两者都还没有直接产出 `InputState` 的公开方法。
6. **Sender 必须尊重 ctx**：`Close` 会等待发送协程退出；忽略 ctx 的发送方会拖住关闭。
7. **回调同步且吞 panic**：监听器慢会拖慢发送与核对；回调里调用 `Close` 会自锁。
8. **写入内容不校验**：文本、参数、附件上限、审批对象合法性都由调用方负责；本包只拒绝结构非法的意图。
9. **附件引用不释放**：D8 的暂存引用生命周期由调用方按结果处理；本包只保留参数。
10. **提交输入标识由本包生成**：服务端若要求特定格式或必须由调用方提供，集成层需要在 `Sender` 里改写或在登记时显式给 `InputID`。
11. **会话新建不自动生成输入标识**：新建没有会话标识可查，调用方应在 `Intent.InputID` 里给一个客户端生成的新建标识；否则未知只能进入需人工确认。
12. **会话命令与模态写入不自动生成输入标识**：`runId`/`taskId`/`connectionId` 由调用方提供；建议同时填进 `Intent.InputID`，否则未知只能进入需人工确认。

## 8. 下一步

1. D1 冻结写方法适配：提交、停止、审批应答、长期允许、提问应答、管理写入、账号动作、附件资源写入、会话生命周期（`create`/`rename`/`delete`/`branch.create`/`branch.end`）、会话命令（`shell.run`/`shell.stop`）、模态任务与实时传输的方法名、参数与回执字段。
2. D2/D1 提供核对实现：按输入标识查用户行、分支归属与运行状态；补一个真实服务端上的 `session.input_lookup` 证据。
3. D4 接入长期允许与审批应答的两段式顺序，并用本包的 `Wait` 验证「写入失败不发应答」。
4. D8 接入附件资源写入：用内容摘要派生幂等键，成功/失败/未发生/人工确认四种结果各自的引用释放规则。
5. 复核核对默认值（次数、间隔、单次超时）并在真实服务端上验证 3 秒内停止状态收敛。
6. 评估终态记录淘汰策略，避免长进程内存线性增长；淘汰不改变「重启不补发」的结论。
