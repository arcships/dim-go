# D6 账号（状态、浏览器登录、登出、用量、可用模型）

Status: 第一版实现完成；按 provider 分发 browser/device；与原 Electron 客户端字段/接口核对完成，登录完成、登出、重置按等价核对通过；待组织评审
Owner: 账号领域 agent
验证: 可执行命令与可观察输出见 §6；真实账号侦察、夹具与脱敏证据见 §6.4
依据: `docs/domains/R.md` 功能块 10 与块 10「沿用」、`docs/domains/A.md` 的「账号与用量」镜像状态与四条总规矩、`docs/domains/D1-connection.md`、`docs/domains/D3-write.md`、`docs/project-organization.md` §4.2 与 §6 F5
范围: 账号状态读取、浏览器登录的发起与完成、登出、用量读取、按账号状态过滤的可用模型。不含界面、连接实现、协议 wire 定义、管理面设置写入、模型档位选择。

---

## 1. 结果包

**领域**：D6 账号

**输入**

| 输入 | 来源 | 本版使用方式 |
|---|---|---|
| 功能块 10「账号与凭据边界」 | `docs/domains/R.md` | 未登录即未登录、登录后返回身份、登出后清空、用量逐项与聚合不可用、重置未知不重发、模型按账号过滤 |
| 块 10「沿用」四条 | `docs/domains/R.md` | 账号状态只有服务端一个来源；登录由服务端完成；用量不派生；重置幂等只在内存 |
| R8「允许保留哪些客户端本地状态」 | `docs/domains/R.md` | 本包只在内存保留登录尝试状态、动作意图与最近一次读取结果，不落盘 |
| 账号与用量镜像状态、四条总规矩 | `docs/domains/A.md` | 状态只由读取驱动；未登录即未登录；不做乐观更新；本包是该状态的唯一驱动者 |
| 写意图、幂等键、未知核对、重启语义 | `docs/domains/D3-write.md` | 登出与重置各为一条 `KindAccountAction` 意图；只发一次；未知不重发 |
| 错误分类与连接方法 | `docs/domains/D1-connection.md` | 由集成层把 wire 映射成 `Fact`/`Usage`/`Model`，并实现授权来源与发送方；本包不复制 wire |
| F5「账号状态与 provider 列表」 | `docs/project-organization.md` §6 | 本包输出 `Snapshot` 与 `ModelList`，供 D11 与管理面、U1 消费 |
| 真实账号夹具 | 本机 alpha 产品 home 的 `auth.json`（只读复制到 `/tmp`）与真实 `dimcode-server` 0.1.0-alpha.21 | 在临时 home 上验证状态、用量、模型清单与 browser/device 登录发起；不进入产品代码，不读取来源存储 |
| provider 登录方式 | 真实服务端 `account.providers` 的 `providerId`、`loginMethods`、`defaultDirectory` | 登录方式只按声明分发；缺声明或本端缺能力都明确报不可用 |

**产物**

| 路径 | 职责 |
|---|---|
| `internal/account/account.go` | 对外类型（`Status`、`Identity`、`Fact`、`Snapshot`、读取接口、`Config`、`New`、`Refresh`）、账号镜像状态与错误 |
| `internal/account/providers.go` | provider 声明、登录方式选择、provider 错误与 `LoginStart` |
| `internal/account/login.go` | 按 provider 分发 browser/device 登录：环回回调、系统浏览器、设备码轮询、尝试释放、超时与取消 |
| `internal/account/browser.go` | 按平台打开系统默认处理程序；只启动，不等待、不读取浏览器输出 |
| `internal/account/state.go` | 登录尝试状态机与启动校验；账号状态只读不推断 |
| `internal/account/usage.go` | 逐项用量、聚合项不可用的明确表达、读取失败保留上次确认值 |
| `internal/account/models.go` | 模型清单读取与按账号状态过滤 |
| `internal/account/actions.go` | 登出与重置：经写登记、`KindAccountAction`、不自动重发 |
| `internal/account/fixture_test.go` | 真实账号夹具验证：状态、用量、模型清单、登录发起与取消；未设置夹具环境变量时跳过 |
| `internal/account/*_test.go` | 单元测试与 `fakeappserver` 集成测试（见 §6.2） |
| `docs/domains/D6-account.md` | 本文件 |

**验收**

- 命令：`go test ./internal/account/`、`go test -race ./internal/account/`、`go vet ./...` 通过；`go test ./...` 当前被 `internal/app` 的 modality 门禁测试阻塞（非账号领域，相关包正在并行修改），账号领域无失败（2026-11-11，macOS arm64）。
- 本领域：77 个顶层测试（含 3 个真实账号夹具测试）、17 个子测试；未设置夹具环境变量时夹具测试跳过，其余 74 个通过。覆盖见 §6.2。
- 证据档位：
  - `fakeappserver`：本机 Unix socket 上的真实 NDJSON，经 D1 连接与真实 `write.Manager` 覆盖状态读取、登录回调、登出与重置。
  - 注入替身：状态/用量/模型读取、授权来源、系统浏览器、写登记在单元测试里替换，覆盖超时、取消、拒绝、校验不一致、聚合不可用与不重发。
  - 真实账号夹具：临时产品 home + 真实 `dimcode-server`（alpha.21）验证账号状态、逐项用量、模型清单与登录发起/取消；见 §6.4。
  - 真实 browser/device 登录完成：真实运行未执行（需真人操作）；字段与接口按 §6.5 等价核对通过，不列为阻塞。

**依赖**

| 依赖 | 需要对方冻结/提供 |
|---|---|
| D1 连接 | 账号状态、用量、模型清单的读取方法与回执形状；浏览器登录的发起与完成交换方法；各方法的错误语义；由集成层实现 `StatusReader`/`UsageReader`/`ModelReader`/`AuthorizationSource`/`AuthorizationCanceler` 与写发送方。真实方法名与回执字段见 §6.3 |
| D3 写登记 | `*write.Manager` 直接满足本包 `Writer`；账号动作使用 `KindAccountAction`；明确失败与未发生可重提，未知与需人工确认不重发 |
| D11 管理面 | 消费 `ModelList` 做模型可选范围；账号设置入口只调用本包，不自行发账号请求 |
| U1 原生界面 | 只读 `Snapshot`/`Models`/`LoginState`；登录、登出、重置只调本包方法 |
| D10 宿主外壳 | 诊断记录由 D10 落盘；本包不写文件、不写日志、不记录授权码 |

**上下文需求**

1. 需要 D1 冻结浏览器登录的 wire 形状：真实服务端已按「尝试标识 + 完整回调地址」工作（§6.3）；本版接口已对齐，仍需 D1 把方法名与参数收敛到正式 wire 定义。
2. 需要 D1 明确账号状态回执里「过期」如何表达；本包按 `signed_in`/`signed_out`/`expired` 三值读取，未知取值按事实不合法拒绝。
3. 需要服务端明确用量项的数值类型（整数或小数）与单位；本版按 `float64` 承载，集成层负责映射。
4. 需要 D11/U1 确认登录入口的呈现位置与取消时机；本包只提供 `Login`/`CancelLogin`。
5. 需要组织确认「重置」对应哪个服务端能力（用量重置或账号重置），本包只按 `KindAccountAction` 与操作名 `reset` 表达。

## 2. 边界

**做**

| 能力 | 说明 |
|---|---|
| 状态读取 | 读取账号状态与身份字段；未登录、已登录、过期三值直接取自服务端；结构不合法即拒绝 |
| 本地镜像 | 保存最近一次确认的状态、身份、用量与读取时间；读取失败只记录错误，不改事实 |
| 浏览器登录 | 生成环回回调地址；请服务端给出授权请求；打开系统浏览器；接收回调；校验回调校验值；完成交换；随后重新读取状态 |
| 超时与取消 | 单次登录有默认 5 分钟上限；`CancelLogin` 结束等待；一次只允许一个登录尝试 |
| 登出 | 经写登记发一次 `KindAccountAction`；明确成功才清空本地状态 |
| 重置 | 经写登记发一次 `KindAccountAction`；未知与需人工确认不重复提交 |
| 用量 | 逐项读取并原样保留；聚合项只能采用服务端值，缺失时明确不可用并给原因 |
| 模型过滤 | 按当前账号状态过滤服务端清单；未登录/未知为空清单并给原因；过期隐藏需要账号的模型 |
| 观察 | `Snapshot`、`Usage`、`Models`、`LoginState`、动作 `write.IntentView` |

**不做**

| 不做 | 归属/原因 |
|---|---|
| 界面、按钮状态、登录弹窗 | U1 |
| 连接、socket、NDJSON、方法名与 wire 类型 | D1 |
| 内嵌浏览器、抓取凭据、保存授权码/令牌 | R 块 10 与 A.md：客户端不持有凭据副本 |
| 本地派生聚合用量、缓存用量并当成事实 | R 块 10：聚合项返回不可用，不出现本地派生数字 |
| 登出/重置的自动重试与自动重发 | D3：未知先核对，不重发 |
| 持久化账号状态、幂等键、动作表 | R8：这些只在内存 |
| 模型档位的具体选择 | R 块 10「不包含」；归管理面设置写入 |
| 管理面设置写入、provider 探测 | D11 |
| 诊断落盘、日志 | D10；本包不写日志，避免泄露授权码 |

本包只使用 Go 标准库与仓库内写登记包，不引第三方依赖，不改 `go.mod`。

## 3. 包结构与对外接口

```text
internal/account/
├── account.go        状态、身份、快照、读取接口、Config、New、Refresh、Usage、Status
├── providers.go      provider 声明、登录方式选择、LoginRequest/LoginStart
├── login.go          BeginLogin/WaitLogin/CancelLogin：browser 回调与 device 轮询
├── browser.go        系统默认处理程序（darwin/windows/其他）
├── state.go          登录尝试状态机与启动校验
├── usage.go          用量项、聚合不可用、RefreshUsage
├── models.go         模型清单、按账号状态过滤、RefreshModels/Models
├── actions.go        登出与重置（写登记、不重发）
└── *_test.go         本领域测试
```

对外接口：

```go
acct, err := account.New(account.Config{
    Status:       statusReader,   // 必填：ReadAccount(ctx) (Fact, error)
    Usage:        usageReader,    // 可选：ReadUsage(ctx) (Usage, error)
    Models:       modelReader,    // 可选：ReadModels(ctx) ([]Model, error)
    Providers:    providerSource, // 可选：ReadProviders(ctx) ([]Provider, error)；nil 表示不提供 provider 登录
    Auth:         authSource,     // 可选：AuthorizationSource 或 ProviderAuthorizationSource
    DeviceAuth:   deviceSource,   // 可选：BeginDeviceAuthorization / PollDeviceAuthorization
    Browser:      browser,        // 可选：nil 时用系统默认处理程序
    Actions:      writeManager,   // 可选：*write.Manager 直接满足 Writer
    LoginTimeout: 5 * time.Minute,
})

snapshot, err := acct.Refresh(ctx)             // 读账号状态
usage, err := acct.RefreshUsage(ctx)           // 逐项读用量
models, err := acct.RefreshModels(ctx)         // 读模型清单并按账号过滤
snapshot = acct.Snapshot()
models = acct.Models()
providers, err := acct.Providers(ctx)          // 读服务端声明的 provider 与登录方式
start, err := acct.BeginLogin(ctx, account.LoginRequest{ProviderID: "..."}) // 按声明发起
snapshot, err = acct.WaitLogin(ctx)            // 等 browser 回调或 device 轮询完成
acct.CancelLogin()                             // 取消并交回服务端释放
state := acct.LoginState()                     // 含 provider 与方式
snapshot, err = acct.Login(ctx, request)       // BeginLogin + WaitLogin 的便捷入口
view, err := acct.Logout(ctx)                  // 经写登记
view, err = acct.Reset(ctx)                    // 经写登记
```

关键约定：

- **未读取就是未知**：`New` 后状态为 `StatusUnknown`，不是 `signed_out`；只有服务端读取结果能把状态改成未登录、已登录或过期。
- **读取失败不清事实**：失败只写 `Snapshot.ReadError`/`Usage.ReadError`，状态、身份、用量与读取时间保持上一次确认值。
- **只在内存**：登录尝试、动作意图、模型清单、用量都随进程存在；重启后从零开始，不补发。
- **不持有凭据**：授权码只在 `Login` 的栈上短暂存在，完成交换后清空引用；`Account` 没有任何令牌或授权码字段，也没有日志输出。

## 4. 设计

### 4.1 状态读取与本地镜像

`Fact{Status, Identity}` 是服务端读取结果的归一化形状，由集成层在交界处映射。校验规则只有结构性的三条：

1. 状态必须是 `signed_in`、`signed_out`、`expired` 之一；空值或未知取值返回 `ErrMalformedFact`，不降级成未登录。
2. `signed_in` 与 `expired` 必须带非空账号标识；缺标识视为事实不合法。
3. `signed_out` 不保留身份字段：读取结果为准，旧身份与旧用量一起清空。

读取成功后 `ReadAt` 是本次读取时间；读取失败不推进它。调用方据 `ReadAt` 与 `ReadError` 决定是否提示数据陈旧，本包不替它判断。

### 4.2 按 provider 分发的登录

登录方式只来自服务端 `account.providers` 的声明，本包不推断、不探测。

**provider 与方式选择**

| 情形 | 结果 |
|---|---|
| provider ID 指定且存在 | 使用该 provider |
| provider ID 指定但不存在 | `ErrProviderUnknown` |
| 未指定且只有一个 provider | 使用它 |
| 未指定且有多个 provider | `ErrProviderSelectionRequired`，不猜 |
| 服务端没有声明 provider | `ErrProvidersUnavailable` |
| 显式指定方式但 provider 未声明 | `ErrLoginMethodUnavailable`，错误里列出已声明方式 |
| 未指定方式、声明含 browser | 使用 browser（桌面端优先） |
| 未指定方式、只声明 device | 使用 device |
| 未声明任何 browser/device | `ErrLoginMethodUnavailable` |
| 声明了方式但本端缺对应来源 | `ErrLoginMethodUnavailable`，说明缺哪种能力 |

provider 声明结构不合法（缺标识、标识重复）时整份拒绝（`ErrMalformedProviders`）；不保留半份。

**浏览器流程**

1. `net.Listen("tcp", "127.0.0.1:0")` 只绑 IPv4 环回，端口由系统分配；回调地址为 `http://127.0.0.1:<port>/auth/callback`。真实服务端只接受 `localhost`/`127.0.0.1`/`::1` 上的 `/auth/callback`，因此路径固定成它。
2. 调浏览器授权来源换一次授权请求（授权地址、校验值、可选尝试标识）。实现 `ProviderAuthorizationSource` 时带上 `providerId`、`directory` 与可选 `email`；只实现 `AuthorizationSource` 时退回单 provider 兼容调用。服务端回执没有独立校验值时，本包从授权地址的 `state` 参数读取；仍缺则发起失败。URL 由服务端给出，本包只打开、不修改。
3. 交给 `Browser.Open` 启动系统浏览器；只启动，不等待退出、不读取浏览器输出。
4. 等待环回回调，三种结束方式：收到回调、`LoginTimeout` 到期、`ctx`/`CancelLogin` 取消。
5. 回调校验：用校验值做常量时间比较；不一致直接失败。回调带 `error` 时按拒绝处理。没有授权码按不完整处理。
6. 把 `{Code, State, CallbackURL}` 与尝试标识交给完成交换；成功后立即清掉手里的授权码引用。
7. 重新读取账号状态，返回快照。账号状态只来自这次读取；登录回执本身不解析身份字段。

**设备码流程**

1. 调 `DeviceAuthorizationSource.BeginDeviceAuthorization(ctx, providerID, directory)`，拿到尝试标识、验证地址、用户码与过期时间。
2. 服务端没有给尝试标识、验证地址或用户码时明确失败，不展示半份提示。
3. `BeginLogin` 立即返回 `LoginStart`，界面据此展示验证地址与用户码；`WaitLogin` 按 `Interval`（默认 5 秒）轮询。
4. 轮询状态：

| 状态 | 处理 |
|---|---|
| `pending` | 继续等；到达过期时间报 `ErrLoginExpired` |
| `completed` | 结束等待并重新读取账号状态 |
| `failed` | 保留服务端原因，返回 `ErrDeviceLogin` |
| `expired` | 返回 `ErrLoginExpired`，登录尝试记为超时 |
| `canceled` | 返回 `ErrLoginCanceled` |
| 未知取值 | 明确失败，不当作 pending |

5. 设备码尝试同样只在内存；取消、超时、失败都按尝试标识交回服务端释放。

**取消与超时**

- 进行中的尝试保存在 `Account` 内存里：provider、方式、尝试标识、授权请求/设备信息、取消句柄。
- `CancelLogin` 取消本地等待；有 `WaitLogin` 在等待时由它收尾，没有等待者时直接结束尝试并释放服务端。
- 发起后、等待前被取消时，`WaitLogin` 返回 `ErrLoginCanceled`，不会退化成「没有尝试」。
- 未完成的尝试（发起失败、回执失败、取消、超时、失败）在授权来源实现 `AuthorizationCanceler` 时按尝试标识交回服务端释放；取消失败不改变本地失败原因。成功完成的尝试不释放，由服务端结算。
- 授权码、用户码与尝试标识不落盘；尝试结束时清空引用。

**登录尝试状态机（客户端本地，不是账号事实）**

| 当前状态 | 事件 | 下一个状态 |
|---|---|---|
| 空闲 | 发起 | 等待登录 |
| 等待登录 | browser 完成交换或 device 轮询完成 | 已完成 |
| 等待登录 | 发起、回调校验、完成交换或轮询失败 | 已失败 |
| 等待登录 | 调用方取消 | 已取消 |
| 等待登录 | 等待超时或设备码过期 | 已超时 |
| 已完成/已失败/已取消/已超时 | 再次发起 | 等待登录 |

`New` 时校验这张表：状态与事件都在集合内、空闲只能作为入口、等待登录必须有四种结束方式、结束态只能重新发起；不合法即拒绝创建并点名条目。非法迁移按 A.md 统一处理：丢弃事件、状态不变、不向用户报错。

回调应答页面是固定文案，不回显授权码或任何请求参数。

真实服务端（alpha.21）的登录形状与对齐结果：

| 服务端方法 | 参数 | 回执 | 本领域对应 |
|---|---|---|---|
| `account.login.browser.begin` | `providerId`、`directory`、`redirectUri`、可选 `email` | `attemptId`、`authorizationUrl`、`expiresAt` | `ProviderAuthorizationSource.BeginProviderAuthorization` |
| `account.login.browser.complete` | `providerId`、`attemptId`、`callbackUrl` | 账号快照 | `ProviderAuthorizationSource.CompleteProviderAuthorization` |
| `account.login.begin` | `providerId`、`directory` | `attemptId`、`verificationUri`、`verificationUriComplete`、`userCode`、`expiresAt` | `DeviceAuthorizationSource.BeginDeviceAuthorization` |
| `account.login.status` | `providerId`、`attemptId` | `{status, error?}` | `DeviceAuthorizationSource.PollDeviceAuthorization` |
| `account.login.cancel` | `providerId`、`attemptId` | 空 | 可选 `AuthorizationCanceler` |

由此得到三条硬约束：回调路径必须是 `/auth/callback`；browser 完成交换需要完整回调地址与尝试标识；browser 与 device 都必须按 provider 发起。真实服务端的发起回执可能没有独立 `state`，本包会从授权地址的 `state` 参数读取；仍缺则拒绝发起。

### 4.3 登出与重置

两条动作都登记成 `write.Intent{Kind: KindAccountAction}`，字段与原客户端对齐：

| 动作 | 写意图幂等键 | 透明参数 | 成功后的本地影响 |
|---|---|---|---|
| 登出 | `account:logout:<n>` | `{"op":"logout","providerId"?}` | 状态回到未知，身份、用量与重置键清空 |
| 重置 | `account:reset:<type>:<n>` | `{"op":"apply_reset","providerId"?,"resetType","subscriptionId","idempotencyKey"}` | 不动账号状态；由调用方重新读取用量 |

要点：

- **只发一次**：每次调用只 `Register` 一次，再 `Wait` 到终态；本包不自行发送、不自行重试。
- **登出不重发**：最近一次登出视图保存在内存。状态是未知、需人工确认、已成功或尚未收敛时，下一次登出只对同一条意图 `Wait`，不再登记；只有明确失败或确认未发生才用新的 `<n>` 重新登记。重新登录后旧登出记录清空，允许新的登出。
- **重置参数与原客户端一致**：必须已登录；`resetType` 只能是 `window` / `monthly_full`；`subscriptionId` 取请求值，缺省用最近一次用量里的订阅标识，仍缺则拒绝；透明参数里带 `providerId` 与内存幂等键。
- **重置幂等键生命周期**：按「订阅标识:重置类型」保存在内存。结果未知或需人工确认时保留，后续显式重试复用同一个键；成功、明确失败或确认未发生后删除，下一次重试生成新键。始终不自动重发；重启后没有键，也不会补发。
- **乐观更新禁止**：只有 `StatusSucceeded` 才清空账号状态；失败、未知、需人工确认都不清空。
- **登出的本地结果**：成功只说明服务端确认了登出，不等于「服务端确认未登录」。因此状态回到 `StatusUnknown` 并写明原因；调用方重新 `Refresh` 后才显示未登录。与原客户端直接置 `signed-out` 的差异见 §6.5。
- **重置键清理**：登出成功、服务端确认未登录、或重新进入已登录会话时清空；不跨进程、不落盘。
- **重启不补发**：动作视图与幂等键只在内存；新实例没有旧意图，也不会在创建时自动提交任何动作。
- **并发**：登出与重置各自串行化；同一动作不会被两个并发调用登记两次。

### 4.4 用量

- `Usage.Lines` 逐项来自服务端，顺序与服务端一致；本包不改写数值、不合并、不补齐缺失项。
- `Usage.Aggregate` 只能采用服务端给的聚合值。服务端没有给时 `Available=false`，`Used/Limit/Unit` 一律没有意义；服务端没给原因时本包补一句「服务端未提供聚合用量」。**本包从不把逐项相加**。
- 真实服务端的 `account.usage` 给出 `features[]`（逐项）、`windows[]`（时间窗令牌用量）、`credits`（聚合额度）与 `subscriptionId`；集成层把前两者映射成 `Usage.Lines`、把 `credits` 映射成 `Usage.Aggregate`，并保留 `subscriptionId` 供重置使用。`credits.unlimited` 没有上限数值，本版按聚合不可用并给原因，不伪造上限。
- 读取失败：`Ready` 与数据保持上一次确认值，`ReadError` 记录原因，`RefreshUsage` 返回错误；观察时间不推进。
- 没有用量读取能力：返回 `ErrUsageUnavailable`，`ReadError` 说明原因，不返回「零用量」。
- 缺少标识的用量项按事实不合法处理，整次读取不采用。

### 4.5 可用模型

`Model` 的字段全部来自服务端：标识、展示名、是否需要账号、是否不可用。过滤规则：

| 账号状态 | 输出 | 原因 |
|---|---|---|
| 已登录/过期但清单尚未读取 | 空清单 | 「模型清单尚未读取」或最近一次读取失败原因 |
| 未登录 | 空清单 | 「未登录：可用模型需要账号」 |
| 状态未知 | 空清单 | 「账号状态未知：不能确定可用模型」 |
| 登录过期 | 只保留不需要账号的模型 | 「登录已过期：需要账号的模型不可用」；为空时说明没有免账号模型 |
| 已登录 | 去掉服务端标为不可用的模型 | 为空时说明服务端没有声明可用模型 |

- 清单整份读取、整份采用；缺标识或标识重复按事实不合法拒绝，不保留半份。
- 真实服务端用 `provider.catalog {providerId}` 返回账号的模型目录，模型带 `unavailableReason`；集成层映射为 `Model{ID, DisplayName, RequiresAccount: true, Unavailable: reason != nil}`。
- `Models()` 每次按当前账号状态重新过滤，因此登出清空状态后，之前读到的清单不会再被当成可用模型返回。
- 清单只由 `RefreshModels` 推进；账号状态变化不会去重取清单。

### 4.6 凭据边界

- 客户端不持有凭据副本：`Account` 没有令牌、刷新令牌、授权码字段；授权码只在完成交换调用期间存在，调用后清空局部引用。
- 登录走系统浏览器；本包不内嵌浏览器、不注入脚本、不读取浏览器存储。
- 不记录授权码、令牌或回调查询串：本包没有日志输出，回调应答也不回显参数。
- 登出只经服务端动作完成，客户端没有需要删除的本地凭据文件。

### 4.7 错误与并发

- 所有读取方法可并发调用；内部状态只在锁内改写。
- 每次调用返回快照副本，调用方改不动内部数据。
- 登录同一时刻只允许一次；动作按动作类型串行。
- 错误都带稳定语义：`ErrStatusRead`、`ErrUsageRead`、`ErrUsageUnavailable`、`ErrModelsRead`、`ErrModelsUnavailable`、`ErrMalformedFact/Usage/Models`、`ErrProvidersUnavailable/Read/Malformed`、`ErrProviderUnknown`、`ErrProviderSelectionRequired`、`ErrLoginMethodUnavailable`、`ErrLogin*`、`ErrDeviceLogin`、`ErrLoginExpired`、`ErrActionsUnavailable`。

## 5. 与 A/R 的对应

| 需求或架构条目 | 本领域实现 | 不在本领域 |
|---|---|---|
| R 块 10 ① 未登录返回未登录 | `Refresh` 按 `signed_out` 填充，身份与用量清空 | 不依赖账号的会话读取由 D1/D2 保证，本包不拦截 |
| R 块 10 ② 登录后返回已登录与身份字段 | 按 provider 声明分发 browser/device，完成后重新读取；`signed_in` 必须有账号标识 | 登录页与浏览器交互（系统浏览器）、设备码提示展示（U1） |
| R 块 10 ③ 登出后未登录且本地无凭据 | 登出经写登记，成功后清空本地状态；重新读取得到未登录 | 凭据删除由服务端完成 |
| R 块 10 ④ 用量逐项一致、聚合不可用 | `RefreshUsage` 原样保留逐项；`Aggregate.Available=false` 加原因 | 用量展示（U1） |
| R 块 10 ⑤ 重置未知不重复提交、重启后提交 0 次 | 不自动重发；未知/需人工确认保留幂等键，显式重试复用同一键；新实例不提交 | 人工确认流程 |
| A 账号与用量镜像状态 | 只有读取与已确认动作能改状态；`clearConfirmedLogout` 是唯一本地清空入口 | — |
| A 四条总规矩 | 只提交意图、不写事实；不做乐观更新；状态单一驱动者；客户端不显示未确认内容 | — |
| A 检查项「缺能力就报缺失」 | 无读取/登录/动作能力时返回稳定错误，不构造替代事实 | — |
| A 检查项「一个状态只有一个驱动者」 | 状态只在 `Account` 内部改写 | — |
| R8 本地状态边界 | 登录尝试、动作、清单、用量都只在内存 | — |
| F5 账号状态与 provider 列表 | `Snapshot`、`ModelList` 与 `Provider` 是冻结候选输出 | 模型档位选择（D11） |

## 6. 验收

### 6.1 命令与结果

```bash
go test ./...
go vet ./...
go test -race ./internal/account/

# 真实账号夹具（需要临时 home 与真实 App Server；未设置时自动跳过）
DIMAGENT_ACCOUNT_FIXTURE=/tmp/<fixture-home> \
DIMAGENT_APP_SERVER_BIN=/path/to/dimcode-server \
  go test ./internal/account -run TestRealAccountFixture -v
```

本版结果（2026-11-11，本机 macOS arm64）：

- `go test ./internal/account/`、`go test -race ./internal/account/`、`go vet ./...` 通过；`internal/account` 74 个顶层测试、17 个子测试全绿。
- `go test ./...` 当前被 `internal/app` 的 `TestModalityGateChecksHelloBeforeSessionMethods` 阻塞（modality 门禁顺序，非账号领域；相关包仍在并行修改）；账号领域包与依赖编译均无失败。

### 6.2 覆盖

| 验收要求 | 测试 |
|---|---|
| 未读取时是未知，不给默认值 | `TestNewStartsUnknownWithoutDefaults` |
| 未登录读取清空身份与用量 | `TestRefreshSignedOutClearsIdentityAndUsage`、`TestAppServerLogoutThroughWriteManager` |
| 已登录与过期读取带回身份字段 | `TestRefreshSignedInAndExpired`、`TestAppServerReadsStatusUsageAndModels` |
| 事实不合法不改状态（未知状态、缺账号标识） | `TestMalformedFactRejectedWithoutStateChange` |
| 读取失败保留上次确认事实 | `TestStatusReadFailureKeepsLastConfirmedFact` |
| 快照是副本、读取时间可核对 | `TestSnapshotIsDetached`、`TestReadTimeUsesConfiguredClock` |
| 登录成功：环回地址、系统浏览器、回调、完成交换、重新读取 | `TestLoginOpensBrowserReceivesCallbackAndRefreshes`、`TestAppServerLoginThroughLoopback` |
| 登录回调超时 | `TestLoginTimeout` |
| 登录被拒绝、校验不一致、缺授权码、完成交换失败 | `TestLoginDeniedByAuthorizationServer`、`TestLoginStateMismatch`、`TestLoginIncompleteCallback`、`TestLoginCompletionFailure` |
| 登录取消与并发限制 | `TestLoginCancel`、`TestLoginRejectsConcurrentAttempt` |
| 打不开浏览器、缺授权来源 | `TestLoginBrowserOpenFailure`、`TestLoginRequiresAuthorizationSource` |
| 登录完成但状态读取失败 | `TestLoginStatusUnreadAfterCompletion` |
| 登录尝试迁移表与坏表拒绝 | `TestLoginTransitionTableMatchesDocument`、`TestBrokenLoginTablesAreRejected`、`TestAdvanceLoginDropsIllegalMoves`、`TestNewRejectsMissingStatusReader` |
| 用量逐项一致、聚合不可用、聚合采用服务端值 | `TestUsageItemsAndUnavailableAggregate`、`TestUsageAggregateUsesServerValues` |
| 用量读取失败保留上次数据；无读取能力明确不可用 | `TestUsageReadFailureKeepsConfirmedData`、`TestUsageWithoutReaderIsUnavailable` |
| 用量结构校验与副本隔离 | `TestUsageLineValidation`、`TestUsageIsDetachedFromReader` |
| 未登录/未知空清单并给原因 | `TestModelsSignedOutAndUnknownAreEmptyWithReason` |
| 已登录过滤不可用模型；过期隐藏需要账号的模型 | `TestModelsSignedInFiltersUnavailable`、`TestModelsExpiredHidesAccountModels`、`TestModelsExpiredWithoutFreeModelsIsEmptyWithReason` |
| 无清单读取能力、清单结构校验、状态变化重新过滤 | `TestModelsWithoutReaderIsUnavailable`、`TestModelsValidation`、`TestModelsFollowStatusChanges` |
| 登出经写登记只发一次、成功清空 | `TestLogoutSendsOnceAndClearsOnSuccess`、`TestAppServerLogoutThroughWriteManager` |
| 登出失败保留状态并可重提 | `TestLogoutFailureKeepsState` |
| 登出未知/需人工确认不重发 | `TestLogoutUnknownIsNotResent` |
| 重置未知不重复提交；只在明确失败后重提 | `TestResetUnknownIsNotResent`、`TestResetRetriesOnlyAfterExplicitFailure` |
| 无写登记时动作不可用 | `TestActionsRequireWriter` |
| 假服务端覆盖重置无回执 → 需人工确认；重启不补发 | `TestAppServerResetUnknownIsNotResent` |
| 完成交换带上尝试标识与完整回调地址；未完成时交回服务端取消 | `TestLoginOpensBrowserReceivesCallbackAndRefreshes`、`TestLoginTimeout`、`TestLoginCancel` |
| provider 声明读取、读取失败、结构校验、多 provider 必须指定 | `TestProvidersRequiresSource`、`TestProvidersReadFailure`、`TestProvidersRejectsMalformed`、`TestLoginRequiresProviderSelectionWhenMultiple`、`TestLoginUnknownProvider` |
| 按 provider 声明选择 browser/device；显式方式必须被声明 | `TestLoginPrefersBrowserWhenBothDeclared`、`TestLoginExplicitDeviceWhenBothDeclared`、`TestLoginDeviceWhenOnlyDevice`、`TestLoginExplicitMethodMustBeDeclared`、`TestLoginRequiresProviderDeclaration`、`TestLoginRequiresSourceForDeclaredMethod`、`TestLoginRejectsProviderWithoutUsableMethod` |
| 设备码提示、轮询完成、失败原因、过期、取消、缺提示、轮询错误 | `TestDeviceLoginFailedKeepsReason`、`TestDeviceLoginExpired`、`TestDeviceLoginCancel`、`TestDeviceBeginThenCancelReleases`、`TestDeviceLoginMissingPromptIsRejected`、`TestDevicePollErrorFails`、`TestLoginDeviceWhenOnlyDevice` |
| 没有尝试时等待明确报错 | `TestWaitLoginWithoutAttempt` |
| 假服务端按 provider 走设备码流程 | `TestAppServerDeviceLoginThroughProvider` |
| 真实夹具：账号状态、逐项用量、模型清单三条读取 | `TestRealAccountFixtureReads` |
| 真实夹具：browser 登录发起、环回 `/auth/callback`、尝试取消 | `TestRealAccountFixtureBrowserLoginBeginReleases` |
| 真实夹具：device 登录发起、验证地址/用户码、尝试取消 | `TestRealAccountFixtureDeviceLoginBeginReleases` |
| 登录提示 email 透传（browser/device） | `TestAppServerLoginThroughLoopback`、`TestAppServerDeviceLoginThroughProvider` |
| 浏览器登录 state：从授权地址解析；缺失即拒绝并释放尝试 | `TestLoginParsesStateFromAuthorizationURL`、`TestLoginRequiresAuthorizationState` |
| 重置字段、幂等键复用/更换、前置条件、订阅标识（含按需读用量） | `TestResetReusesIdempotencyKeyAfterUnknown`、`TestResetNewKeyAfterDefinitiveFailure`、`TestResetRequiresSignedIn`、`TestResetRejectsInvalidTypeAndMissingSubscription`、`TestResetExplicitSubscriptionOverridesUsage`、`TestResetLoadsUsageForSubscription` |
| 登出 provider 选择、重新登录后可再次登出、登出后重置键重建 | `TestLogoutProviderSelection`、`TestLogoutAgainAfterRelogin`、`TestResetKeyRegeneratedAfterLogout` |
| 错误码映射到原客户端稳定码 | `TestErrorCodeMapsToOriginalCodes` |
| 与原 Electron 客户端字段/接口核对 | 本文 §6.5（静态对照，无独立测试） |

证据档位：`fakeappserver`（本机 Unix socket + NDJSON + 真实 `write.Manager`）、注入替身，以及临时产品 home 上的真实账号夹具（见 §6.4）；真实 browser/device 登录完成与真实登出/重置未验证。

### 6.3 与 D1/D3 的接口形状

| 提供方 | 形状 | 本包消费方式 |
|---|---|---|
| D1 连接 | `protocol.Client.Call` / `CallInto` | 集成层用它实现 `StatusReader`、`UsageReader`、`ModelReader`、`ProviderSource`、`AuthorizationSource`/`ProviderAuthorizationSource`、`DeviceAuthorizationSource` 与可选的 `AuthorizationCanceler`，把 wire 回执映射成领域类型，不把 wire 类型带进本包 |
| D1 错误分类 | `failure.KindOf(err)` | 读取失败原样返回；写动作的错误由 D3 分类 |
| D1 会话流 | 与本领域无关 | 账号状态不依赖会话是否打开；本包不订阅通知、不轮询 |
| D3 写登记 | `Register(write.Intent)` / `Wait(ctx, id)` | `*write.Manager` 直接满足 `Writer`；账号动作用 `KindAccountAction`，透明参数 `{"op":"logout"|"reset"}`，幂等键 `account:<op>:<n>` |
| D3 结果语义 | `StatusSucceeded` / `Failed` / `NotOccurred` / `Unknown` / `NeedsManualConfirmation` | 只有成功清空登录状态；只有失败与未发生允许重提；其余只等待同一条意图 |
| D11 管理面 / U1 界面 | `Snapshot`、`Usage`、`ModelList`、`LoginState`、`write.IntentView` | 只读；动作只调 `Login`/`CancelLogin`/`Logout`/`Reset` |

真实 App Server（alpha.21）的账号方法与本领域映射：

| 能力 | 真实方法 | 参数/回执要点 | 本领域对应 |
|---|---|---|---|
| 提供者 | `account.providers` | 返回 `providerId`、`displayName`、`loginMethods`、`defaultDirectory` | 集成层据此确认账号能力存在 |
| 状态 | `account.status` | `{providerId}` → `{credentialEpoch, generation, account?}`；`account` 为空即未登录 | `ReadAccount` → `Fact`；`account.subject` 映射 `Identity.AccountID` |
| 用量 | `account.usage` | `{providerId}` → `features[]`、`windows[]`、`credits`、`plan` | `ReadUsage` → `Usage`；`features`/`windows` → `Lines`，`credits` → `Aggregate` |
| 模型 | `provider.catalog` | `{providerId}` → `catalog.models[{id,name,unavailableReason}]` | `ReadModels` → `[]Model` |
| 登录发起（browser） | `account.login.browser.begin` | `{providerId, directory, redirectUri}` → `{attemptId, authorizationUrl, expiresAt}` | `ProviderAuthorizationSource.BeginProviderAuthorization` |
| 登录完成（browser） | `account.login.browser.complete` | `{providerId, attemptId, callbackUrl}` | `ProviderAuthorizationSource.CompleteProviderAuthorization`；`CallbackURL` = `callbackUrl` |
| 登录发起（device） | `account.login.begin` | `{providerId, directory}` → `{attemptId, verificationUri, verificationUriComplete, userCode, expiresAt}` | `DeviceAuthorizationSource.BeginDeviceAuthorization` |
| 登录轮询（device） | `account.login.status` | `{providerId, attemptId}` → `{status, error?}` | `DeviceAuthorizationSource.PollDeviceAuthorization` |
| 登录取消 | `account.login.cancel` | `{providerId, attemptId}` | 可选 `AuthorizationCanceler` |

回调路径必须是 `/auth/callback`；真实服务端回执可能没有独立 `state`，本包从授权地址读取，仍缺则拒绝。browser 与 device 都必须按 provider 发起，登录方式只取自 `account.providers` 的 `loginMethods`。统一二进制 `dimcode-server` 需要 `app-server` 子命令（由 D1 的自启参数提供；夹具测试用 `ServerArgs: []string{"app-server"}`）。

本包只导入 `internal/write`，不导入 D1 的业务包；接口实现由集成层提供，两边可独立演进。

### 6.4 真实账号侦察、夹具与证据

**侦察（只读、脱敏）**

- 产品 home 的 `auth.json` 是账号持久化的唯一来源，顶层两个字段：
  - `dimAccountState`：`schemaVersion`、`credentialEpoch`、`generation`、`credentialRevision`、`state`、`identity{issuer,subject,email,name}`、`tokenDigest`；
  - `nextApiOauth`：`type`、`access`、`refresh`、`expires`、`issuer`、`tokenEndpoint`、`authorizationEndpoint`、`deviceAuthorizationEndpoint`、`deviceTokenEndpoint`、`clientId`、`scope`、`relayBaseUrl`。
- 只记录字段名、类型与档位；`access`、`refresh`、`tokenDigest`、`credentialEpoch` 的值没有打印、没有进入证据；身份字段只记录是否存在。
- 来源 home 的 `rust/dimcode.sqlite*`、session 文件与 `daemon.log` 未读取、未复制，产品代码也不含来源路径。
- 真实服务端暴露 `account.providers`、`account.status`、`account.usage`、`provider.catalog`；账号提供者为 `dimcode-api-oauth`（显示名 `Dim OAuth`，登录方式 `browser`/`device`），目录 `https://dimagent.cn`。

**夹具流程**

1. `mktemp -d /tmp/dimagent-account-fixture-XXXXXX`，目录权限 0700。
2. 只复制 `auth.json`（0600）与 `config.json`，新建空的 `rust/`（0700）；不复制来源的 SQLite、session 文件与日志。
3. 设置 `DIMAGENT_ACCOUNT_FIXTURE=<夹具目录>` 与 `DIMAGENT_APP_SERVER_BIN=<真实 dimcode-server>`；夹具测试只从环境变量取路径。
4. 统一二进制用 `ServerArgs: []string{"app-server"}` 启动；`--home` 指向夹具 `rust/`，`--product-home` 指向夹具根。
5. 运行 `go test ./internal/account -run TestRealAccountFixture -v`。
6. 验证后删除夹具（含复制的认证文件）；夹具不进入仓库，也不作为 CI 依赖。本次夹具在取证完成后已删除。

**脱敏证据（alpha.21，本机 macOS arm64）**

```text
服务端版本=0.1.0-alpha.21 能力：account.status=true account.usage=true provider.catalog=true
账号提供者：displayName="Dim OAuth" loginMethods=[browser device] directoryConfigured=true
账号状态：status=signed_in identity=accountIDSet=true emailSet=true nameSet=true planSet=false
用量：lines=1 aggregateAvailable=true subscriptionIDSet=true aggregateReason="" observedAtSet=true
用量项[0]：key="web_search" unit="call" window="" limitSet=true resetSet=true
模型清单：available=21 first=[deepseek-v4-flash seed-2.1-pro seed-2.1-turbo]
浏览器登录发起：host=dimagent.cn redirectLoopback=true provider=dimcode-api-oauth method=browser cancelled=1
设备码登录发起：provider=dimcode-api-oauth method=device attemptIssued=true verificationURISet=true userCodeSet=true cancelled=1
真实服务回收：stopped=true
```

结论：

- **三条读取在真实账号上通过**：状态 `signed_in` 且带身份字段；用量有逐项 `web_search`、服务端聚合额度与 `subscriptionId`；模型目录 21 条可用（登录状态下）。
- **browser 登录发起通过**：按 provider 选择 browser，真实服务端接受环回 `/auth/callback`，返回授权地址与尝试标识；等待回调超时后按尝试标识取消成功。
- **device 登录发起通过**：按 provider 显式选择 device，真实服务端返回尝试标识、验证地址与用户码；取消后按尝试标识交回释放。
- **登录完成未运行**：browser 完成交换需要真人在授权页登录（可能含验证码/二次验证）；device 完成需要真人在验证页输入用户码。本次只验证到发起、提示、回调地址与取消；字段与接口按 §6.5 与原客户端核对后等价通过。
- **真实 `account.status` 没有独立的「过期」字段**：过期与恢复由错误码/恢复流程表达，第一版的 `StatusExpired` 还没有真实 wire 映射。
- **登出/重置未在真实夹具上执行**：登出会调用服务端吊销授权，可能影响来源账号的 refresh token；需要一次性测试账号或测试租户。字段与清理语义按 §6.5 核对后等价通过。

**blocked（剩余环境项）**

| 项 | 状态 | 缺什么 |
|---|---|---|
| 过期/恢复状态映射 | blocked | 真实 `account.status` 无过期字段；需要 D1 冻结「需要恢复」的错误语义与映射 |
| 系统浏览器真实拉起 | blocked | 夹具测试用替身记录授权地址，没有真正弹出浏览器；平台差异未验证（打开真实授权页可能触发自动授权与凭据轮换） |
| token 轮换与写回 | blocked | access token 距过期尚远，未验证 30 秒新鲜窗口内的刷新与 auth.json 写回 |
| Windows/Linux 真实账号 | blocked | 只在 macOS arm64 上取证 |
| 登出/重置真实运行 | blocked | 会吊销来源账号授权；等价核对已通过，真实运行仍需一次性测试账号 |

### 6.5 与原 Electron 客户端的字段与接口核对

**核对范围与方法**

- 对照文件（只读）：
  - `dim-electron/packages/desktop/packages/electron-main/src/services/account-service.ts`：账号状态机、登录/登出/重置调用与清理顺序。
  - `.../services/account-rpc-service.ts`：账号 RPC 面（status/login/logout/refreshModels/usage/applyUsageReset）。
  - `.../services/oauth/oauth-browser-flow.ts`、`oauth-callback-url.ts`：浏览器授权、回调参数解析、state 校验、超时/取消。
  - `.../app-server/account-provider.ts`：App Server 托管账号适配（begin/complete/cancel、usage、reset）。
  - `packages/shared/src/protocol/rpc.ts`：`AccountUser`、`AccountState`、`AccountErrorPayload`、`AccountUsagePayload`、`AccountApplyUsageResetParams/Result` 等类型。
  - 服务端侧 `dim-2/crates/dim-app-server/src/managed_provider*`、`dim-oauth/src/store.rs`：真实 wire 字段与方法（已在 §6.3 记录）。
- 方法：逐项读类型定义与调用点，与本包导出类型、写意图透明参数、清理顺序、错误分类对照；只做静态核对与现有夹具验证，不完成真实登录、不改真实账号。
- 判定口径：字段名/类型/调用参数/清理效果一致 → **按等价核对通过**；仅呈现层或客户端状态命名不同 → 语义差异；本包没有的字段/能力 → 缺失。

**逐项结论**

| 项 | 原客户端 | 本包 | 判定 |
|---|---|---|---|
| 账号用户字段 | `uid`、`email`、`name`、`preferredUsername` | `AccountID`(=subject)、`Email`、`DisplayName`、`PreferredUsername`、`Plan` | 部分一致：已补 `PreferredUsername`；`Plan` 是我们多出；`uid` 回退规则见差异 |
| 账号状态 | `signed-out`/`signing-in`/`signed-in`/`refreshing`/`error` + `checked`/`modelSetup` | `Status`（unknown/signed_out/signed_in/expired）+ `LoginPhase` + `ModelList` | 语义差异：分层不同；我们无 `refreshing`/`error` 状态与 `checked`/`modelSetup` |
| 登录发起（browser） | `email?`、`redirectUri`（默认 `localhost:54321/auth/callback`）→ `authorizationUrl`、`redirectUri`；state 从地址解析 | `providerId`、`directory`、`redirectUri`、`email` → `AuthorizationRequest{URL, State, Attempt}`；State 为空时也从授权地址解析 | 字段一致；端口策略不同（随机 vs 固定） |
| 登录完成（browser） | `callbackUrl`（= redirectUri + code + state）→ `authenticated` | `providerId`、`attempt`、`AuthorizationResult{Code, State, CallbackURL}` | **按等价核对通过**：完整回调地址、尝试标识、provider 都对齐 |
| 尝试处理 | provider→attempt 的内存映射 | attempt→provider 的集成映射；尝试在内存，结束释放 | 等价：两边都不落盘，取消/超时都按尝试交回 |
| 登录取消/超时 | 取消服务端尝试；取消时保留已有登录会话 | `CancelLogin` 结束本地等待；授权来源实现 `AuthorizationCanceler` 时按尝试释放；账号状态不变 | 接口等价；现有集成层尚未实现 `AuthorizationCanceler`，需补（见 §8） |
| 登录失败状态 | 置 `error` 状态并清空 user | 登录尝试记 `failed` 与原因，账号状态保持上次事实 | 语义差异：由调用方组合呈现 |
| 登出调用 | `account.logout {providerId}`；随后清 provider 绑定与本地状态 | 写意图 `{"op":"logout","providerId"}`；成功清状态、身份、用量、重置键 | **核心按等价核对通过**；本地状态置 unknown（原客户端置 signed-out），provider 绑定清理在集成层 |
| 登出后重新登录 | 无持久登出记录 | 新会话清空登出视图，允许再次登出 | **等价通过**（修复了跨会话不重发的问题） |
| 重置参数 | `account.apply_reset {providerId, subscriptionId, resetType, idempotencyKey}` | 同名透明参数（`op=apply_reset`），`resetType` 为 `window`/`monthly_full` | **按等价核对通过** |
| 重置前置 | 必须已登录；必须知道 subscriptionId | 同；缺订阅标识拒绝 | **等价通过** |
| 重置幂等 | 按 `subscriptionId:type` 内存键；未定重试复用；明确失败/成功/登出清除 | 同；显式重试复用键，本包不自动重发 | **等价通过** |
| 重置返回 | `{status:'applied', replayed, appliedAt, usage}` | 写意图视图；调用方重新读取用量 | 语义差异：无 `replayed`/合并后的 usage |
| 错误分类 | 服务端 `appError` 转发 + UI 码（`login_failed` 等），`{code,message}` | Go 类型化错误 + `ErrorCode(err)` 稳定码映射 | 部分一致：码可映射，`{code,message}` 载荷仍由集成层组装 |
| 用量字段 | `plan`、`subscription`、`config`、`resets`、`subscriptionId`、`featureMeters` | `Lines`、`Aggregate`、`SubscriptionID`、观察时间与读取错误 | 部分一致：`subscriptionId` 已补；plan/subscription/config/resets 未承载 |
| 模型字段 | `modelSetup`、`unlockedModelCount` | `ModelList{Ready, Reason, Models}` | 语义差异：呈现分层不同 |
| 回调参数来源 | query 或 hash | 环回 HTTP 只能收到 query | 限制：hash 形式回调无法读取（真实提供方使用 query） |
| 登录/登出并发 | `logout` 等登录/刷新结束再执行 | 登出与登录各自独立 | 语义差异：并发顺序由调用方保证 |

**等价通过的范围**

- **浏览器登录完成**：字段、完整回调地址、provider、尝试标识、state 校验与原客户端一致，按等价核对通过；真实完成运行未执行（需要真人），不再作为阻塞。取消释放依赖集成层实现 `AuthorizationCanceler`，当前集成层未实现，是仍需补齐的一项。
- **登出**：服务端调用与参数、成功后的本地清理、重置键清理、重新登录后可再次登出与原客户端一致，按等价核对通过；本地状态写 unknown 与原客户端写 signed-out 是记录的语义差异。
- **重置**：参数、前置条件、幂等键作用域与生命周期、清理时机与原客户端一致，按等价核对通过；返回形状与 usage 刷新方式由集成层补齐。

**分类汇总**

- 一致（按等价核对通过）：browser 登录完成参数与回调、尝试处理、登出参数与清理、重置参数/前置/幂等生命周期、`ErrorCode` 的稳定码。取消释放接口一致，但生产集成仍缺 `AuthorizationCanceler` 实现。
- 我们缺失（已补或仍缺）：`preferredUsername`（已补）、`subscriptionId`（已补）、`AuthorizationCanceler` 生产实现（集成层仍缺）、plan/subscription/config/resets 用量字段（仍缺）、`AccountState` 呈现字段（仍缺）。
- 我们多出：provider 列表与按声明分发（原客户端只有单一托管 provider）、`StatusExpired`、`Usage.Lines/Aggregate` 结构、`LoginPhase`、`ModelList.Reason`、`ErrorCode` 映射。
- 语义差异：登录失败/取消的状态呈现、登出后 unknown vs signed-out、重置返回形状、固定/随机回调端口、query/hash 回调、登录与登出的并发顺序。

**仍不能等价通过的部分**

- `AccountState`/`AccountUser.uid`/`checked`/`modelSetup`/`unlockedModelCount` 等呈现层字段与命名不同。
- 错误载荷 `{code,message}` 与 `appError` 透传未在本包表达；`ErrorCode` 只覆盖本领域错误。
- 用量字段（plan/subscription/config/resets/featureMeters）未完整承载；模型呈现字段不同。
- 固定回调端口、hash 回调、登录/登出并发顺序存在差异。

## 7. 已知限制

1. **wire 未冻结**：真实 alpha.21 的方法名见 §6.3，但仍未由 D1 写进冻结定义；账号动作的 `providerId`、`resetType`、`subscriptionId`、`idempotencyKey` 由本包放进透明参数，发送方必须原样下发。
2. **回调路径与 state**：真实服务端只接受 `/auth/callback`，本包已固定成它；浏览器登录必须有 state，服务端回执没有单独字段时本包从授权地址读取（与原客户端一致），仍缺则拒绝。
3. **只绑 IPv4 环回**：回调地址固定 `127.0.0.1`；真实服务端也接受 `localhost` 与 `::1`，本包未覆盖。
4. **登录回执不解析身份**：身份只来自登录后的状态读取；登录成功但读取失败时返回 `ErrLoginStatusUnread`，状态保持未知。
5. **登出后状态是未知而非未登录**：这是有意的，避免把「动作成功」当成「已确认未登录」；原客户端直接置 `signed-out`，调用方必须重新读取。
6. **动作无核对标识**：账号动作没有可核对输入标识，未知只能按 D3 上限收敛到需人工确认；本包不读取服务端存储自动确认。
7. **动作与重置键不跨进程**：重启后旧意图、登出视图与重置幂等键全部丢失，也不会补发；人工确认依赖服务端事实与外部流程。
8. **用量数值类型**：`Used/Limit` 用 `float64`；服务端若用字符串或高精度小数，集成层需在映射时决定，本包不猜测。
9. **模型是否需要账号由集成层映射**：真实托管提供者的模型一律 `RequiresAccount=true`；本包只消费 `RequiresAccount` 与 `Unavailable`，不探测模型能力。
10. **真实账号标识取 subject**：真实 `account.status` 没有单独的账号标识字段，集成层用 `identity.subject` 映射 `Identity.AccountID`；原客户端在 subject 等于 email 时把 uid 置空，本包不做这层回退。
11. **过期没有真实映射**：真实服务端不返回 `expired`；第一版只能通过读取失败或恢复要求表达，`StatusExpired` 的真实触发条件待 D1 冻结。
12. **用量字段不完整**：本包承载逐项、聚合与 `subscriptionId`；原客户端的 plan/subscription/config/resets 字段尚未承载，由集成层或后续版本补。
13. **provider 声明是登录前提**：没有 `ProviderSource` 时登录明确不可用；多个 provider 必须显式指定，不在候选之间猜。
14. **同时声明两种方式时默认 browser**：这是桌面端的固定偏好，不是能力推断；需要 device 时必须显式指定。
15. **兼容旧授权来源**：只实现 `AuthorizationSource` 时退回单 provider 调用，不能把 `providerId`/`directory`/`email` 交给服务端；provider 专属登录应实现 `ProviderAuthorizationSource`。设备码邮箱提示是可选能力 `DeviceAuthorizationEmailSource`，旧实现忽略它。
16. **设备码提示必须完整**：服务端没有给尝试标识、验证地址或用户码时整次发起失败，不展示半份提示；轮询间隔由服务端给，缺省 5 秒。
17. **读取失败保留旧数据**：`Snapshot`/`Usage` 会同时带旧数据与 `ReadError`；调用方需要自己决定是否提示陈旧，本包不替它判过期。
18. **无诊断输出**：本包不写日志、不写文件；登录失败的诊断要由调用方按错误类别交给 D10。
19. **无自动刷新登录**：过期只作为读取结果出现；重新登录必须由用户显式发起。
20. **夹具含真实凭据**：夹具只用于本机验证，验证后必须删除；不得提交、不得进 CI、不得打印内容。
21. **真实登录完成未运行**：browser 与 device 的完成交换都需要真人操作；字段/接口按 §6.5 等价核对通过，但真实运行仍未执行。
22. **尝试只在内存**：provider、方式、尝试标识、用户码与授权码都随尝试存在内存，不落盘；进程重启后没有可恢复的尝试。
23. **重置显式重试会再次发送**：不自动重发，但用户显式重试会登记新意图并复用同一幂等键；与原客户端一致，是否重复扣减由服务端幂等保证。
24. **重置返回形状**：本包返回写意图视图，不带原客户端的 `replayed`/合并后的 usage；调用方需重新读取用量。
25. **错误载荷**：本包用类型化 Go 错误，并提供 `ErrorCode(err)` 稳定码；不产生完整 `{code,message}`，服务端 `appError` 透传仍需集成层处理。
26. **状态命名与呈现字段**：本包用 `Status`+`LoginPhase`+`ModelList`，没有原客户端的 `signing-in`/`refreshing`/`error`、`checked`、`modelSetup`、`unlockedModelCount`。
27. **回调端口与并发顺序**：原客户端默认固定 `localhost:54321` 并串行等待登录/刷新后再登出；本包随机环回端口，登录与登出由调用方保证顺序。
28. **hash 回调**：原客户端从 query 或 hash 解析回调；环回 HTTP 只能收到 query，hash 形式无法读取（真实提供方使用 query）。
29. **provider 绑定清理不在本领域**：原客户端登出后清 App Server provider 凭据绑定；本包只清账号镜像状态，绑定清理归集成层/D11。

## 8. 下一步

1. D1 冻结真实账号 wire（§6.3 的方法名与参数）；集成层（`internal/app/account_sources.go`）把 `ProviderAuthorizationBegin.Email` 与设备码邮箱提示接到 begin 请求，并把账号动作的 `providerId`/`resetType`/`subscriptionId`/`idempotencyKey` 原样下发（当前适配器还没有转发 email）。
2. 真实运行可选：用一次性测试账号完成 browser/device 登录、登出与重置，验证吊销与幂等效果；等价核对已通过，真实运行不作为当前阻塞。
3. 集成层补齐 `AuthorizationCanceler`（浏览器与设备码尝试的 `account.login.cancel`）、provider 绑定清理（登出后清 App Server provider 凭据）与统一二进制 `app-server` 子命令的自启配置。
4. U1 组合 `Status`+`LoginPhase`+`ModelList` 成原客户端的 `AccountState`（含 signing-in/refreshing/error/checked/modelSetup/unlockedModelCount），并把错误映射成 `AccountErrorPayload{code,message}`。
5. 映射过期/恢复：确认 `account_recovery_required` 等错误如何转为 `StatusExpired` 或 `ErrStatusRead`，并补一条真实证据。
6. 需要 plan/subscription/config/resets 字段时，由集成层读取并承载，或在 D6 下一版补类型；当前只保留逐项、聚合与 `subscriptionId`。
7. 系统浏览器：在 macOS/Windows/Linux 各做一次真实拉起，确认失败路径的错误可诊断。
8. 与 D11 对齐 F5：确认 `ModelList` 足以支撑模型可选范围，以及过期状态是否应允许免账号模型。
9. 决定回调端口策略（随机 vs 固定 54321）、是否支持 hash 回调，以及登录/登出并发顺序是否要像原客户端一样串行等待。
10. 验证 token 新鲜窗口内的刷新与 auth.json 写回（只改夹具副本），确认不会把令牌写进日志或产品状态。
