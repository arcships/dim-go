# D6 账号（状态、浏览器登录、登出、用量、可用模型）

Status: 第一版实现完成；真实账号夹具验证通过，真实登录完成仍 blocked；待组织评审
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
| 真实账号夹具 | 本机 alpha 产品 home 的 `auth.json`（只读复制到 `/tmp`）与真实 `dimcode-server` 0.1.0-alpha.21 | 在临时 home 上验证状态、用量、模型清单与登录发起；不进入产品代码，不读取来源存储 |

**产物**

| 路径 | 职责 |
|---|---|
| `internal/account/account.go` | 对外类型（`Status`、`Identity`、`Fact`、`Snapshot`、读取接口、`Config`、`New`、`Refresh`）、账号镜像状态与错误 |
| `internal/account/login.go` | 浏览器登录：环回回调地址、授权请求、系统浏览器、回调校验、完成交换、超时与取消 |
| `internal/account/browser.go` | 按平台打开系统默认处理程序；只启动，不等待、不读取浏览器输出 |
| `internal/account/state.go` | 登录尝试状态机与启动校验；账号状态只读不推断 |
| `internal/account/usage.go` | 逐项用量、聚合项不可用的明确表达、读取失败保留上次确认值 |
| `internal/account/models.go` | 模型清单读取与按账号状态过滤 |
| `internal/account/actions.go` | 登出与重置：经写登记、`KindAccountAction`、不自动重发 |
| `internal/account/fixture_test.go` | 真实账号夹具验证：状态、用量、模型清单、登录发起与取消；未设置夹具环境变量时跳过 |
| `internal/account/*_test.go` | 单元测试与 `fakeappserver` 集成测试（见 §6.2） |
| `docs/domains/D6-account.md` | 本文件 |

**验收**

- 命令：`go test ./...`、`go vet ./...`、`go test -race ./internal/account/`；本版全部通过（2026-11-10，macOS arm64）。
- 本领域：45 个顶层测试、14 个子测试通过；另有 2 个真实账号夹具测试在未设置夹具环境变量时跳过。覆盖见 §6.2。
- 证据档位：
  - `fakeappserver`：本机 Unix socket 上的真实 NDJSON，经 D1 连接与真实 `write.Manager` 覆盖状态读取、登录回调、登出与重置。
  - 注入替身：状态/用量/模型读取、授权来源、系统浏览器、写登记在单元测试里替换，覆盖超时、取消、拒绝、校验不一致、聚合不可用与不重发。
  - 真实账号夹具：临时产品 home + 真实 `dimcode-server`（alpha.21）验证账号状态、逐项用量、模型清单与登录发起/取消；见 §6.4。
  - 真实浏览器登录完成：**blocked**，需要真人在授权页完成登录；见 §6.4。

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
├── login.go          授权请求、环回回调、系统浏览器、超时/取消、完成交换、登录错误
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
    Auth:         authSource,     // 可选：BeginAuthorization / CompleteAuthorization（可再实现 CancelAuthorization）
    Browser:      browser,        // 可选：nil 时用系统默认处理程序
    Actions:      writeManager,   // 可选：*write.Manager 直接满足 Writer
    LoginTimeout: 5 * time.Minute,
})

snapshot, err := acct.Refresh(ctx)             // 读账号状态
usage, err := acct.RefreshUsage(ctx)           // 逐项读用量
models, err := acct.RefreshModels(ctx)         // 读模型清单并按账号过滤
snapshot = acct.Snapshot()
models = acct.Models()
snapshot, err = acct.Login(ctx)                // 发起并等待浏览器登录
acct.CancelLogin()
state := acct.LoginState()
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

### 4.2 浏览器登录

流程：

1. `net.Listen("tcp", "127.0.0.1:0")` 只绑 IPv4 环回，端口由系统分配；回调地址为 `http://127.0.0.1:<port>/auth/callback`。真实服务端只接受 `localhost`/`127.0.0.1`/`::1` 上的 `/auth/callback`，因此路径固定成它。
2. 调 `AuthorizationSource.BeginAuthorization(ctx, redirectURI)` 换一次授权请求：授权地址、可选校验值、可选尝试标识（`Attempt`）。URL 由服务端给出，本包只打开、不修改。
3. 交给 `Browser.Open` 启动系统浏览器；只启动，不等待退出、不读取浏览器输出。
4. 等待环回回调，三种结束方式：收到回调、`LoginTimeout` 到期、`ctx`/`CancelLogin` 取消。
5. 回调校验：服务端给出校验值时用常量时间比较；不一致直接失败。回调带 `error` 时按拒绝处理。没有授权码按不完整处理。
6. 把 `{Code, State, CallbackURL}` 与尝试标识交给 `CompleteAuthorization(ctx, attempt, result)` 完成交换；成功后立即清掉手里的授权码引用。
7. 重新读取账号状态，返回快照。账号状态只来自这次读取；登录回执本身不解析身份字段。
8. 未完成的尝试（发起失败、回执失败、超时、取消）在授权来源实现 `AuthorizationCanceler` 时按尝试标识交回服务端释放；取消失败不改变本地失败原因。

状态机（客户端本地的登录尝试，不是账号事实）：

| 当前状态 | 事件 | 下一个状态 |
|---|---|---|
| 空闲 | 发起 | 等待回调 |
| 等待回调 | 完成交换成功 | 已完成 |
| 等待回调 | 发起、回调校验或完成交换失败 | 已失败 |
| 等待回调 | 调用方取消 | 已取消 |
| 等待回调 | 等待超时 | 已超时 |
| 已完成/已失败/已取消/已超时 | 再次发起 | 等待回调 |

`New` 时校验这张表：状态与事件都在集合内、空闲只能作为入口、等待回调必须有四种结束方式、结束态只能重新发起；不合法即拒绝创建并点名条目。非法迁移按 A.md 统一处理：丢弃事件、状态不变、不向用户报错。

回调应答页面是固定文案，不回显授权码或任何请求参数。

真实服务端（alpha.21）的登录形状与对齐结果：

| 服务端方法 | 参数 | 回执 | 本领域对应 |
|---|---|---|---|
| `account.login.browser.begin` | `providerId`、`directory`、`redirectUri`、可选 `email` | `attemptId`、`authorizationUrl`、`expiresAt` | `BeginAuthorization` → `AuthorizationRequest{URL, Attempt}` |
| `account.login.browser.complete` | `providerId`、`attemptId`、`callbackUrl` | 账号快照 | `CompleteAuthorization(ctx, attempt, result)`；`CallbackURL` 就是 `callbackUrl` |
| `account.login.cancel` | `providerId`、`attemptId` | 空 | 可选 `CancelAuthorization` |

由此得到两条硬约束：回调路径必须是 `/auth/callback`；完成交换需要完整回调地址与尝试标识，不能只交授权码。真实服务端不在发起回执里给出 `state`，回调校验由服务端按尝试标识完成；本包的本地校验只在服务端给出 `State` 时启用。

### 4.3 登出与重置

两条动作都登记成 `write.Intent{Kind: KindAccountAction}`：

| 动作 | 幂等键 | 透明参数 | 成功后的本地影响 |
|---|---|---|---|
| 登出 | `account:logout:<n>` | `{"op":"logout"}` | 状态回到未知，身份与用量清空 |
| 重置 | `account:reset:<n>` | `{"op":"reset"}` | 不动账号状态；由调用方重新读取用量 |

要点：

- **只发一次**：每次调用只 `Register` 一次，再 `Wait` 到终态；本包不自行发送、不自行重试。
- **未知不重发**：最近一次动作视图保存在内存。状态是未知、需人工确认、已成功或尚未收敛时，下一次调用只对同一条意图 `Wait`，不再登记；只有明确失败或确认未发生才用新的 `<n>` 重新登记。
- **乐观更新禁止**：只有 `StatusSucceeded` 才清空账号状态；失败、未知、需人工确认都不清空。
- **登出的本地结果**：成功只说明服务端确认了登出，不等于「服务端确认未登录」。因此状态回到 `StatusUnknown` 并写明原因；调用方重新 `Refresh` 后才显示未登录。
- **重启不补发**：动作视图只在内存；新实例没有旧意图，也不会在创建时自动提交任何动作。
- **并发**：登出与重置各自串行化；同一动作不会被两个并发调用登记两次。

### 4.4 用量

- `Usage.Lines` 逐项来自服务端，顺序与服务端一致；本包不改写数值、不合并、不补齐缺失项。
- `Usage.Aggregate` 只能采用服务端给的聚合值。服务端没有给时 `Available=false`，`Used/Limit/Unit` 一律没有意义；服务端没给原因时本包补一句「服务端未提供聚合用量」。**本包从不把逐项相加**。
- 真实服务端的 `account.usage` 给出 `features[]`（逐项）、`windows[]`（时间窗令牌用量）与 `credits`（聚合额度）；集成层把前两者映射成 `Usage.Lines`、把 `credits` 映射成 `Usage.Aggregate`。`credits.unlimited` 没有上限数值，本版按聚合不可用并给原因，不伪造上限。
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
- 错误都带稳定语义：`ErrStatusRead`、`ErrUsageRead`、`ErrUsageUnavailable`、`ErrModelsRead`、`ErrModelsUnavailable`、`ErrMalformedFact/Usage/Models`、`ErrLogin*`、`ErrActionsUnavailable`。

## 5. 与 A/R 的对应

| 需求或架构条目 | 本领域实现 | 不在本领域 |
|---|---|---|
| R 块 10 ① 未登录返回未登录 | `Refresh` 按 `signed_out` 填充，身份与用量清空 | 不依赖账号的会话读取由 D1/D2 保证，本包不拦截 |
| R 块 10 ② 登录后返回已登录与身份字段 | 登录完成后重新读取；`signed_in` 必须有账号标识 | 登录页与浏览器交互（系统浏览器） |
| R 块 10 ③ 登出后未登录且本地无凭据 | 登出经写登记，成功后清空本地状态；重新读取得到未登录 | 凭据删除由服务端完成 |
| R 块 10 ④ 用量逐项一致、聚合不可用 | `RefreshUsage` 原样保留逐项；`Aggregate.Available=false` 加原因 | 用量展示（U1） |
| R 块 10 ⑤ 重置未知不重复提交、重启后提交 0 次 | 动作视图只在内存；未知/需人工确认只 `Wait` 不 `Register`；新实例不提交 | 人工确认流程 |
| A 账号与用量镜像状态 | 只有读取与已确认动作能改状态；`clearConfirmedLogout` 是唯一本地清空入口 | — |
| A 四条总规矩 | 只提交意图、不写事实；不做乐观更新；状态单一驱动者；客户端不显示未确认内容 | — |
| A 检查项「缺能力就报缺失」 | 无读取/登录/动作能力时返回稳定错误，不构造替代事实 | — |
| A 检查项「一个状态只有一个驱动者」 | 状态只在 `Account` 内部改写 | — |
| R8 本地状态边界 | 登录尝试、动作、清单、用量都只在内存 | — |
| F5 账号状态与 provider 列表 | `Snapshot` 与 `ModelList` 是冻结候选输出 | 模型档位选择（D11） |

## 6. 验收

### 6.1 命令与结果

```bash
go test ./...
go vet ./...
go test -race ./internal/account/
```

本版结果：三条命令全部通过；`internal/account` 的 45 个顶层测试、14 个子测试全绿（2026-11-10，本机 macOS arm64）。

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

证据档位：`fakeappserver`（本机 Unix socket + NDJSON + 真实 `write.Manager`）与注入替身；真实账号、真实登录与真实用量未验证。

### 6.3 与 D1/D3 的接口形状

| 提供方 | 形状 | 本包消费方式 |
|---|---|---|
| D1 连接 | `protocol.Client.Call` / `CallInto` | 集成层用它实现 `StatusReader`、`UsageReader`、`ModelReader` 与 `AuthorizationSource`，把 wire 回执映射成 `Fact`/`Usage`/`Model`，不把 wire 类型带进本包 |
| D1 错误分类 | `failure.KindOf(err)` | 读取失败原样返回；写动作的错误由 D3 分类 |
| D1 会话流 | 与本领域无关 | 账号状态不依赖会话是否打开；本包不订阅通知、不轮询 |
| D3 写登记 | `Register(write.Intent)` / `Wait(ctx, id)` | `*write.Manager` 直接满足 `Writer`；账号动作用 `KindAccountAction`，透明参数 `{"op":"logout"|"reset"}`，幂等键 `account:<op>:<n>` |
| D3 结果语义 | `StatusSucceeded` / `Failed` / `NotOccurred` / `Unknown` / `NeedsManualConfirmation` | 只有成功清空登录状态；只有失败与未发生允许重提；其余只等待同一条意图 |
| D11 管理面 / U1 界面 | `Snapshot`、`Usage`、`ModelList`、`LoginState`、`write.IntentView` | 只读；动作只调 `Login`/`CancelLogin`/`Logout`/`Reset` |

本包只导入 `internal/write`，不导入 D1 的业务包；接口实现由集成层提供，两边可独立演进。

### 6.4 blocked

| 项 | 状态 | 缺什么 |
|---|---|---|
| 真实 App Server 的账号状态、用量与模型清单 | blocked | 需要冻结的 wire 方法名与回执形状；本版用集成层接口与假服务端验证 |
| 真实 OAuth 登录（真实提供方、真实浏览器） | blocked | 需要真实账号、真实授权服务端与 D1 冻结的登录 wire；本版用替身浏览器验证环回回调 |
| 系统浏览器真实拉起证据 | blocked | `systemBrowser` 已实现，但没有自动化证据证明 macOS/Windows/Linux 上能拉起授权页 |
| 真实用量逐项与聚合边界 | blocked | 需要真实账号与服务端用量回执；本版只验证结构表达 |
| 登出/重置的真实服务端语义 | blocked | 需要 D1/D11 冻结账号动作方法；本版用假服务端验证一次性发送与不重发 |

## 7. 已知限制

1. **wire 未冻结**：测试适配器里出现的方法名（`account.status`、`account.usage`、`model.list`、`account.login.begin`、`account.login.complete`、`account.logout`、`account.reset`）只是集成测试的约定，不代表协议冻结。
2. **回调校验依赖服务端**：服务端不给校验值时不做 state 校验，只依赖随机环回端口。真实提供方应要求 state。
3. **只绑 IPv4 环回**：回调地址固定 `127.0.0.1`；IPv6 `::1` 未覆盖。
4. **登录回执不解析身份**：身份只来自登录后的状态读取；登录成功但读取失败时返回 `ErrLoginStatusUnread`，状态保持未知。
5. **登出后状态是未知而非未登录**：这是有意的，避免把「动作成功」当成「已确认未登录」；调用方必须重新读取。
6. **动作无核对标识**：账号动作没有可核对输入标识，未知只能按 D3 上限收敛到需人工确认；本包不读取服务端存储自动确认。
7. **动作不跨进程**：重启后旧意图、动作视图全部丢失，也不会补发；人工确认依赖服务端事实与外部流程。
8. **用量数值类型**：`Used/Limit` 用 `float64`；服务端若用字符串或高精度小数，集成层需在映射时决定，本包不猜测。
9. **模型是否需要账号由集成层映射**：本包只消费 `RequiresAccount` 与 `Unavailable`，不探测模型能力。
10. **读取失败保留旧数据**：`Snapshot`/`Usage` 会同时带旧数据与 `ReadError`；调用方需要自己决定是否提示陈旧，本包不替它判过期。
11. **无诊断输出**：本包不写日志、不写文件；登录失败的诊断要由调用方按错误类别交给 D10。
12. **无自动刷新登录**：过期只作为读取结果出现；重新登录必须由用户显式发起。

## 8. 下一步

1. D1 冻结账号与登录 wire：状态、用量、模型清单、授权发起与完成交换、登出与重置的方法名、参数与回执字段，并把测试适配器替换成真实实现。
2. 真实账号证据：用真实 OAuth 提供方完成一次浏览器登录，记录回调、完成交换与登录后状态读取；补登出与重置的真实语义。
3. 系统浏览器：在 macOS/Windows/Linux 各做一次真实拉起，确认失败路径的错误可诊断。
4. 与 D11 对齐 F5：确认 `ModelList` 足以支撑模型可选范围，以及过期状态是否应允许免账号模型。
5. 与 U1 对齐登录入口：确认取消时机、超时提示与「登录已完成但状态未知」的呈现方式。
6. 评估是否需要在服务端支持 state 强制校验；若支持，把「服务端不给 state」从限制里去掉。
