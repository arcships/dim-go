# D11a 设置与运行环境（设置来源、模型可用性、存储读数、运行组件、模态任务与实时传输）

Status: 第一版实现完成，待组织评审
Owner: D11a 设置与运行环境领域
验证: 可执行命令与可观察输出见 §6；真实模型工具调用证据见 §6.5；真实模态任务证据见 §6.6；其余真实服务端证据见 §6.4
依据: `docs/domains/R.md` 功能块 9、10「模型可用性过滤」与块 9「沿用」、`docs/domains/A.md` 的「管理面读数」与四条总规矩、`docs/domains/D1-connection.md`、`docs/domains/D3-write.md`、`docs/domains/D6-account.md`、`docs/project-organization.md` §4.2 D11
范围: 设置来源与项目作用域读写、模型与 provider 目录及选择读取与写入、存储占用读数、运行组件状态与安装转达、模态能力状态、模态任务提交/查询/取消、实时传输打开/发送/关闭与事件接收。不含界面、连接实现、协议 wire 定义、写登记内部、插件与技能、MCP、钩子、计划任务、唤醒、账号本身、存储清理；模态方法的路由（hello 能力、控制连接或会话流）由 D12 集成层实现。

---

## 1. 结果包

**领域**：D11a 设置与运行环境（D11 管理面的设置与运行环境部分）

**输入**

| 输入 | 来源 | 本版使用方式 |
|---|---|---|
| 功能块 9「计划任务、运行组件、存储读数、唤醒与模态」 | `docs/domains/R.md` | 运行组件状态与安装、存储只读读数、模态能力状态与不可用原因、模态任务提交/查询/取消、实时传输打开/发送/关闭与事件接收 |
| 功能块 10「模型可用性过滤」 | `docs/domains/R.md` | 模型可选范围按账号状态过滤；模型档位选择（选择写入）归本领域 |
| 块 9「沿用」与四条总规矩 | `docs/domains/R.md`、`docs/domains/A.md` | 能力缺失即报缺失、组件获取由服务端完成、存储只读、读数单一驱动者 |
| 设置来源、项目根、版本前置条件 | 产品基线的设置服务语义：来源声明 `scopes`/`userOnly`/`writeOnly`，项目以进程内根编号定位，写入带 `expectedRevision` | 本包只表达领域形状，不复制 wire；项目作用域与版本冲突由本包先行校验 |
| provider 与模型目录、当前选择 | 产品基线的模型管理语义：`providers[].models`、`enabledModelIds`、`selection`、`provider.select` | 目录整份读取；可选性由账号可用模型清单过滤；选择写入经写登记转达 |
| 运行组件与模态能力 | 产品基线的运行组件与模态能力语义：状态枚举、能力类别与原因、任务记录与实时连接 | 本包只映射为领域类型并校验结构；wire 映射由 D12 实现 |
| 会话方法表 | D1 `session.methods` 与 hello 能力 | 模态任务与实时传输在发出任何请求或写登记之前，按 hello 能力与会话方法表做门禁 |
| provider/模型管理事实 | 真实 wire：`provider.list`、`provider.management`、`provider.probe`、`provider.create/update/models.*/preferences.update` | 本包只表达领域形状与校验，不复制 wire；D12 映射回执 |
| 错误分类 | `internal/appserver/failure`、`internal/appserver/protocol` | 能力缺失归为 `ErrCapabilityMissing`；其他错误按读取失败分类 |
| 写意图与不重试 | `internal/write`、`docs/domains/D3-write.md` | 设置写入与组件安装登记 `KindManagementWrite`，模态任务与实时开关登记 `KindModalityWrite`；都不自行重试 |
| 账号可用模型视图 | `internal/account`、`docs/domains/D6-account.md` | 只读 `Models() account.ModelList`，不自行判断账号状态 |

**产物**

| 路径 | 职责 |
|---|---|
| `internal/settings/settings.go` | 包说明、能力名与错误分类、`CapabilityError`、`Config`/`Manager`/`New`、读取错误分类、JSON 对象校验 |
| `internal/settings/scope.go` | 用户与项目作用域、作用域校验与缓存键 |
| `internal/settings/sources.go` | 设置来源清单读取、声明校验、作用域/只读边界 |
| `internal/settings/projects.go` | 已授权项目根读取、按目录解析、授权校验 |
| `internal/settings/documents.go` | 设置文档读取与缓存、版本前置条件、经写登记的写入、冲突判定 |
| `internal/settings/models.go` | provider 与模型目录、当前选择、按账号可用模型过滤、选择写入 |
| `internal/settings/providers.go` | provider 清单/详情/显式探测、新建与更新、启用状态、模型启用集合、模型编辑/重置、模型偏好 |
| `internal/settings/collaboration.go` | 协作模型策略读取、版本前置写入与冲突判定 |
| `internal/settings/agents.go` | agent 档案清单、会话当前 agent 读取、会话 agent 更新 |
| `internal/settings/storage.go` | 存储占用只读读数与结构校验 |
| `internal/settings/runtimes.go` | 运行组件清单、安装转达、安装后重读 |
| `internal/settings/modality.go` | 模态能力状态、类别、操作与不可用原因 |
| `internal/settings/modality_session.go` | 会话模态接口、方法名常量、能力门禁、模态写登记与公共校验 |
| `internal/settings/modality_tasks.go` | 模态任务提交/取消/读取/清单/结果分页 |
| `internal/settings/modality_realtime.go` | 实时打开/发送/关闭、连接状态、事件订阅与投递 |
| `internal/settings/*_test.go` | 单元测试、`fakeappserver` 集成测试与会话模态替身测试（见 §6.2） |
| `scripts/verify-real-model.sh` | 真实模型验证脚本：读取 `/tmp` 夹具，起 daemon、跑会话、断言真实工具调用与工具结果 |
| `scripts/verify-modality.py` | 真实模态验证脚本：查询能力、提交 OCR 任务、轮询状态、分页读结果、取消；记录实时门禁 |
| `scripts/verify-provider-admin.py` | 真实 provider/模型管理验证脚本：列表、新建、探测、更新、启用、模型开关、编辑、重置、偏好与版本冲突 |
| `scripts/verify-collaboration-agent.py` | 真实协作模型策略与会话 agent 档案证据：策略版本/冲突、agent 方法表、采用与读回 |
| `docs/domains/D11a-settings.md` | 本文件 |

**验收**

- 命令：`go test ./internal/settings`、`go test -race ./internal/settings`、`go test ./...`、`go vet ./...`；本版全部通过（2026-11-10，macOS arm64）。
- 本领域：104 个顶层测试、53 个子测试，共 157 个通过；语句覆盖率 82.1%。
- 真实模型：`scripts/verify-real-model.sh --fixture /tmp/dimagent-real-model` 通过，观察到真实工具调用与工具结果（§6.5）。
- 真实模态：`scripts/verify-modality.py --fixture /tmp/dimagent-real-model --server ~/.dimcode-alpha/rust/dimcode-server --image /tmp/dimagent-modality-ocr.png` 通过；OCR 任务 `queued → running → succeeded`，结果分页读完；取消 `queued → cancelled`；实时因 adapter 不可用标 blocked（§6.6）。
- 真实 provider/模型管理：`scripts/verify-provider-admin.py --fixture /tmp/dimagent-real-model --server ~/.dimcode-alpha/rust/dimcode-server` 通过；新建→探测→更新→启用→模型开关→编辑→重置→偏好→版本冲突全部在真实 alpha.21 上跑通（§6.7）。
- 真实协作策略与 agent 档案：`scripts/verify-collaboration-agent.py --fixture /tmp/dimagent-real-model --server ~/.dimcode-alpha/rust/dimcode-server` 通过；策略 revision 1→4、过期写入返回冲突；会话 agent get/update/adopt/unchanged 均跑通（§6.8）。
- 证据档位：
  - `fakeappserver`：本机 Unix socket 上的真实 NDJSON，经 D1 连接与真实 `write.Manager` 覆盖设置来源/项目/读写/冲突、模型与账号过滤、存储读数、运行组件安装、模态能力。
  - 注入替身：各读取接口、写发送方与会话模态面在单元测试里替换，覆盖能力缺失、事实不合法、缓存隔离、冲突、重试边界、任务分页、实时事件与失败原因。
  - 真实模型：本机 `dimcode-server` + `/tmp` 夹具 + 真实 Dim 账号，观察到一次真实工具调用（`exec`）与工具结果，见 §6.5。
  - 真实模态：真实服务端执行 OCR 任务并分页读完结果；取消成功；实时因能力不可用标 blocked，见 §6.6。
  - 真实 provider/模型管理：真实 alpha.21 上由脚本直连验证（§6.7）；本包 Go API 仍依赖 D12 的 `ProviderAdmin` 适配器。
  - 真实协作策略与 agent 档案：真实 alpha.21 上由脚本直连验证（§6.8）；本包 Go API 仍依赖 D12 的 `CollaborationReader`/`AgentAdmin` 适配器。
  - 本包 Go API 对真实 App Server 的设置/模态/provider/协作/agent 方法、真实模态任务与实时传输：**blocked**，见 §6.4。

**依赖**

| 依赖 | 需要对方冻结/提供 |
|---|---|
| D1 连接 | 设置、模型、存储、运行组件、模态能力的读取方法与回执形状；`hello.features` 中的能力名；会话方法表与模态任务/实时方法；由集成层实现本包的读取接口与会话模态接口。`*daemon.Connection` 直接满足 `Features` |
| D3 写登记 | `*write.Manager` 直接满足 `Writer`；设置写入与组件安装使用 `KindManagementWrite`，模态任务与实时开关使用 `KindModalityWrite`；明确失败与未知不重发 |
| D12 集成层 | 实现 `ModalitySession`：会话方法门禁、任务状态/清单/结果读取、实时帧发送；把 `modality.realtime.event`/`ended` 通知投递给本包；实现 `ProviderAdmin`：`provider.list`/`management`/`probe` 的映射 |
| provider 管理依赖 | `*daemon.Connection` 的 hello 能力、控制连接 `protocol.Client.Call` 与写发送方 | D12 按 `KindManagementWrite` 的 op 映射 provider 写方法；本包只登记意图 |
| 协作策略依赖 | hello 的 `collaboration.model_policy`/`collaboration.set_model_policy`、控制连接与写发送方 | D12 映射读取与写入；本包只做版本前置与冲突判定 |
| agent 档案依赖 | `agent.list`、`session.methods`、`session.agent.get`、`session.agent.update` | D12 实现 `AgentAdmin`；`agent.list` 是内建方法，真实 hello 不一定声明 |
| D6 账号 | `Models() account.ModelList` 作为唯一可用性来源；模型标识需与 provider 目录的模型标识可对齐（见 §7） |
| V 验收设施 | `fakeappserver` 的脚本回执与故障注入；本版已用于设置、模型、存储、运行组件与模态能力的集成测试；模态任务与实时传输用会话模态替身验证 |
| 组织 | 确认 D11 内部分工：本领域只管设置与运行环境，插件、技能、MCP、钩子、计划任务、唤醒归管理面的其余部分 |

**上下文需求**

1. 需要 D1 冻结设置来源与项目根的 wire：`settings.sources`、`settings.projects`、`settings.read`、`settings.update` 的方法名、作用域形状与回执字段。本版只在测试适配器里出现这些方法名，不代表协议冻结。
2. 需要 D1 冻结 provider 目录与当前选择的读取形状：`providers[].enabled`、`models[].id`、`enabledModelIds`、`selection`。本版按此形状映射，缺失的必填布尔值由集成层拒绝。
3. 需要 D1 冻结存储读数、运行组件、模态能力、模态任务与实时传输的方法名与字段；本版的领域类型已按产品语义固定，会话流映射由 D12 实现。
4. 需要 D6 明确模型标识是否带 provider 限定。本版按模型标识完全匹配，见 §7 第 3 条。
5. 需要 D1 明确选择读取与写入是否带项目目录 `cwd`。本版不带 `cwd`，作用于服务端默认作用域，见 §7 第 16 条。
6. 需要 D12 明确模态方法的路由：任务查询与实时发送走会话流；任务提交/取消、实时打开/关闭由写发送方按 `KindModalityWrite` 映射；实时事件与结束事件由 D12 路由到 `DeliverRealtimeEvent`/`DeliverRealtimeEnded`。
7. 需要 D1 确认 `modality.task.submit`/`realtime.open` 回执里的资源标识可以由写发送方放进 `write.Reply.InputID`，供本包在成功后重读。

## 2. 边界

**做**

| 能力 | 说明 |
|---|---|
| 设置来源 | 读取来源清单与每项的 `scopes`、`userOnly`、`writeOnly`、`schema`；来源必须声明作用域才能读写 |
| 项目作用域 | 读取已授权项目根；按目录解析项目根；项目根必须已授权；项目写入拒绝 `userOnly` 键 |
| 设置读取 | 按来源与作用域读取完整文档与版本；结构不合法即拒绝；最近一次确认值留在内存 |
| 设置写入 | 必须带 `expectedRevision`；本地缓存发现版本已变时直接返回冲突且不发请求；经写登记只发一次；成功后采用回执文档 |
| 模型与 provider | 读取 provider、模型、启用模型清单与当前选择；按账号可用模型清单过滤可选性并说明原因；写入当前选择，成功后重读回填 |
| provider 管理 | 列表与详情；新建；更新显示名/地址/启用状态；显式探测（只读，不写配置）；显式全量设置启用模型；模型编辑/重置；模型偏好写入 |
| 协作模型策略 | 读取策略与 revision；按 revision 写入 agent/inherit/fixed；本地与服务端冲突判定 |
| agent 档案 | 读取 agent 清单；读取会话当前 agent；按当前 agent 前置条件更新会话 agent，写入后重读 |
| 模态任务 | 提交、状态读取、清单分页、结果分页、取消；提交与取消经写登记，读取走集成层模态调用面（本机为 daemon 级控制方法） |
| 实时传输 | 打开、发送帧、关闭；打开与关闭经写登记，发送走集成层模态调用面，接收按事件；维护本进程内的连接状态与订阅 |
| 会话方法门禁 | 每个模态调用先查该会话的方法表；缺方法或方法表不可读即返回能力缺失，零请求 |
| 存储读数 | 读取数据库、WAL 与可回收字节；只读，没有清理入口，不做本地估算 |
| 运行组件 | 读取状态、来源、版本、路径与原因；安装只转达给服务端，成功后重读清单 |
| 模态能力 | 读取整体支持位、可配置位、任务/实时支持位、版本、类别、操作、选项与不可用原因 |
| 观察 | 每类读数的最近一次确认值与缓存访问；设置文档按来源与作用域缓存 |

**不做**

| 不做 | 归属/原因 |
|---|---|
| 界面、设置页、按钮状态、提示文案 | U1 |
| 连接、socket、NDJSON、方法名与 wire 类型 | D1；本包只在错误分类时读取 `protocol.RPCError` |
| 写登记内部、重试策略、未知核对 | D3；本包只登记一次并等待终态 |
| 插件、技能、MCP、钩子、计划任务、唤醒 | D11 管理面的其余部分 |
| 账号状态、登录、登出、用量 | D6；本包只消费 `Models()` |
| 存储清理、删除、本地容量估算、离线指令 | R 块 9：存储只读，清理交给离线指令 |
| 运行组件的下载、校验、落盘 | 服务端负责；本包只转达安装并读取状态 |
| 模态任务的等待/轮询策略、音频编解码、媒体播放 | 本包只提供提交/查询/取消与事件投递；等待与播放由调用方按 `ReadTask` 自行收敛 |
| 凭据存储、凭据回显、凭据日志 | 本包只在显式写入请求里透传凭据，不缓存、不回显、不写日志；详情只给 `CredentialConfigured` |
| provider 删除、导入旧配置、远程刷新模型 | 不在本版；本版只做列表/新建/更新/探测/启用/模型配置/偏好 |
| agent 档案创建/删除、agent 档案字段编辑 | 本版只做会话采用的 agent 更新与 agent 清单读取；`agent.create` 由调用方/其他领域负责 |
| 实时事件的路由与连接选择 | D12：本包只接收 `DeliverRealtimeEvent`/`DeliverRealtimeEnded` 并按连接分发 |
| 模态方法的 wire 参数与回执 | D1/D12：本包不定义 wire 结构，类型由集成层映射；本机服务端把模态作为 daemon 级控制方法 |
| 设置键的业务含义、schema 解释、默认值补齐 | 服务端拥有设置事实；本包只做结构校验 |
| 任何持久化、跨进程幂等表、重启恢复 | R8：缓存与幂等键只在内存 |

本包只使用 Go 标准库与仓库内的连接、写登记、账号包，不引第三方依赖，不改 `go.mod`。

## 3. 包结构与对外接口

```text
internal/settings/
├── settings.go        能力名、错误分类、Config/Manager/New、读取错误分类、结构校验
├── scope.go           用户/项目作用域与缓存键
├── sources.go         来源清单与声明校验
├── projects.go        项目根与按目录解析
├── documents.go       设置文档读取、写入、版本冲突
├── models.go          provider/模型目录、账号过滤与选择写入
├── storage.go         存储只读读数
├── runtimes.go        运行组件清单与安装转达
├── modality.go        模态能力状态
├── modality_session.go 会话模态接口、方法名、门禁与模态写登记
├── modality_tasks.go  模态任务提交/取消/读取/清单/结果分页
├── modality_realtime.go 实时打开/发送/关闭、连接状态与事件订阅
└── *_test.go          单元测试、fakeappserver 与会话模态替身测试
```

对外接口：

```go
manager, err := settings.New(settings.Config{
    Features: conn,           // 可选：*daemon.Connection，读 hello.features
    Sources:  sourcesReader,  // ReadSettingsSources(ctx) (Sources, error)
    Projects: projectsReader, // ReadSettingsProjects(ctx) ([]Project, error)
    Settings: settingsReader, // ReadSettings(ctx, source, scope) (Document, error)
    Models:   modelReader,    // ReadModelCatalog(ctx) (ModelCatalog, error)
    Account:  acct,           // *account.Account，提供 Models() account.ModelList
    Storage:  storageReader,  // ReadStorage(ctx) (StorageReading, error)
    Runtimes: runtimeReader,  // ReadRuntimes(ctx) ([]RuntimeEntry, error)
    Modality: modalityReader, // ReadModalityCapabilities(ctx) (ModalityCapabilities, error)
    ModalitySession: sessionModality, // 会话方法门禁、任务读取、实时发送
    Providers: providerAdmin, // provider 列表/详情/探测
    Collaboration: collaborationReader, // collaboration.model_policy
    Agents:       agentAdmin,           // agent.list / session.agent.*
    Writer:   writeManager,   // *write.Manager，登记 KindManagementWrite / KindModalityWrite
})

sources, err := manager.RefreshSources(ctx)
projects, err := manager.RefreshProjects(ctx)
project, err := manager.ResolveProject(ctx, cwd)
doc, err := manager.Read(ctx, "execution", settings.UserScope())
doc, err = manager.Read(ctx, "execution", settings.ProjectScope(project.Root))
doc, err = manager.Update(ctx, settings.UpdateRequest{
    Source:           "execution",
    Scope:            settings.ProjectScope(project.Root),
    ExpectedRevision: doc.Revision,
    Document:         json.RawMessage(`{"sandbox":false}`),
})

models, err := manager.RefreshModels(ctx)
models, err = manager.SelectModel(ctx, settings.SelectModelRequest{
    ProviderID:       "cloud",
    ModelID:          "c1",
    ExpectedRevision: models.Selection.Revision,
})
report, err := manager.RefreshStorage(ctx)
entries, err := manager.RefreshRuntimes(ctx)
entry, err := manager.InstallRuntime(ctx, "uv")
caps, err := manager.RefreshModality(ctx)

providers, err := manager.RefreshProviders(ctx)
providerDetail, err := manager.ReadProvider(ctx, providerID)
probe, err := manager.ProbeProvider(ctx, settings.ProviderProbeRequest{
    ProviderID: "01900000-0000-4000-8000-000000000002",
    Connection: &settings.ProviderConnectionInput{
        DriverKind: "openai-compatible",
        Auth:       settings.ProviderAuth{Type: "anonymous"},
        BaseURL:    "http://127.0.0.1:9000/v1",
    },
})
providerDetail, err = manager.CreateProvider(ctx, settings.ProviderCreateRequest{
    ID:          "01900000-0000-4000-8000-000000000001",
    DisplayName: "示例供应商",
    Enabled:     true,
    Connection: settings.ProviderConnectionInput{
        DriverKind: "openai-compatible",
        Auth:       settings.ProviderAuth{Type: "anonymous"},
        BaseURL:    "http://127.0.0.1:9000/v1",
    },
    Models: []settings.ProviderModelInput{
        {ID: "model-1", DisplayName: "模型一", Status: "active"},
    },
})
providerDetail, err = manager.UpdateProvider(ctx, settings.ProviderUpdateRequest{
    ProviderID: providerID, ExpectedVersion: 1,
    Patch: settings.ProviderPatch{DisplayName: &name, Enabled: &enabled},
})
providerDetail, err = manager.SetModelsEnabled(ctx, providerID, 2, []string{"model-1"})
providerDetail, err = manager.SetModelEnabled(ctx, providerID, "model-1", 3, false)
providerDetail, err = manager.EditModel(ctx, settings.ProviderModelEditRequest{
    ProviderID: providerID, ModelID: "model-1", ExpectedVersion: 4,
    Edit: settings.ProviderModelEdit{DisplayName: &name},
})
providerDetail, err = manager.ResetModel(ctx, providerID, "model-1", 5)
providerDetail, err = manager.UpdateModelPreferences(ctx, settings.ModelPreferenceRequest{
    ProviderID: providerID, ModelID: "model-1", ExpectedVersion: 6,
    Patch: settings.ModelPreferencePatch{
        ReasoningEffort: &settings.PreferenceUpdate{Action: "set", Value: "high"},
    },
})

policy, err := manager.RefreshCollaborationPolicy(ctx)
policy, err = manager.SetCollaborationPolicy(ctx, settings.CollaborationPolicyRequest{
    ExpectedRevision: policy.Revision,
    Policy: settings.CollaborationModelPolicy{
        Mode: settings.CollaborationPolicyFixed,
        ModelProviderID: "dimcode-api-oauth", ModelID: "deepseek-v4.1-flash",
    },
})
agents, err := manager.RefreshAgents(ctx)
sessionAgent, err := manager.ReadSessionAgent(ctx, "s1", "main")
updatedAgent, err := manager.UpdateSessionAgent(ctx, settings.SessionAgentUpdateRequest{
    SessionID: "s1", BranchID: "main",
    AgentID: "01900000-0000-4000-8000-000000000003",
    ExpectedAgentID: sessionAgent.AgentID,
})

task, err := manager.SubmitTask(ctx, settings.ModalityTaskSubmitRequest{
    SessionID: "s1", Category: "image", Operation: "generate", ClientTaskID: "local-1",
    Params: json.RawMessage(`{"prompt":"猫"}`),
})
task, err = manager.ReadTask(ctx, "s1", task.ID)
page, err := manager.ListTasks(ctx, "s1", settings.ModalityTaskQuery{State: settings.ModalityTaskQueued, Limit: 20})
chunk, err := manager.ReadTaskResult(ctx, "s1", settings.ModalityTaskResultQuery{
    TaskID: task.ID, Name: "out.png", Offset: 0, Length: 4096,
})
task, err = manager.CancelTask(ctx, "s1", task.ID)

conn, err := manager.OpenRealtime(ctx, settings.ModalityRealtimeOpenRequest{
    SessionID: "s1", Operation: "realtime.voice",
})
cancel, err := manager.WatchRealtime("s1", conn.ConnectionID, settings.ModalityRealtimeListener{
    OnEvent: func(settings.ModalityRealtimeEvent) {},
    OnEnded: func(settings.ModalityRealtimeEnded) {},
})
err = manager.SendRealtime(ctx, "s1", conn.ConnectionID, settings.ModalityRealtimeFrame{
    Kind: "text", Data: json.RawMessage(`"你好"`),
})
err = manager.CloseRealtime(ctx, "s1", conn.ConnectionID)

// 最近一次确认值；第二个返回值表示是否读过。
sources, ok := manager.Sources()
projects, ok := manager.Projects()
doc, ok := manager.Cached("execution", settings.UserScope())
models, ok := manager.Models()
report, ok := manager.Storage()
entries, ok := manager.Runtimes()
entry, ok := manager.Runtime("uv")
caps, ok := manager.Modality()
providers, ok := manager.Providers()
providerDetail, ok := manager.Provider(providerID)
policy, ok := manager.CollaborationPolicy()
agents, ok := manager.Agents()
task, ok := manager.Task("s1", "task-1")
conn, ok := manager.RealtimeConnection("s1", "conn-1")
```

关键约定：

- **能力缺失统一分类**：所有缺失项返回 `*CapabilityError`，`errors.Is(err, settings.ErrCapabilityMissing)` 成立，`err.Capability` 给出方法名或接口名。
- **不猜默认值**：读取失败保留上一次确认值；写入成功只采用回执里的文档；任何缺失字段都不补 0、不补空字符串、不补假值。
- **只在内存**：来源、项目、文档、模型、存储、运行组件、模态能力、provider 列表/详情、任务缓存与实时连接都随进程存在；重启不补发任何写入。
- **不存凭据**：provider 详情只保留 `CredentialConfigured`；新建/更新请求里的凭据只在本次调用中透传，不缓存、不回显、不写日志。
- **agent 无服务端版本**：真实 `session.agent.update` 没有 revision；本包用 `ExpectedAgentID` 作为前置条件，并在写入后重读确认。
- **写入只登记一次**：`Update`、`SelectModel`、`InstallRuntime` 与全部 provider/模型写入经 `KindManagementWrite`，`SubmitTask`/`CancelTask`/`OpenRealtime`/`CloseRealtime` 经 `KindModalityWrite`；都只登记一次并等待终态，没有重发路径。
- **模态先过方法门禁**：任务与实时调用在发出任何请求或写登记之前，先查 `hello.features` 与会话方法表；缺方法或方法表不可读时返回能力缺失。
- **返回值是副本**：所有缓存访问与读取结果都深拷贝，调用方改不到内部状态。

## 4. 设计

### 4.1 设置来源与作用域

`Sources` 是服务端声明的来源清单：

| 字段 | 用法 |
|---|---|
| `Dialect` | 文档方言；空表示服务端没有声明，本包不推断 |
| `Source.Name` | 来源标识，必填且唯一 |
| `Source.Scopes` | `user` / `project`；没有声明的作用域不可读写 |
| `Source.UserOnly` | 只能写入用户作用域的键；项目写入时客户端先行拒绝 |
| `Source.WriteOnly` | 只接受写入；读取返回 `ErrSourceWriteOnly` |
| `Source.Schema` | 服务端结构描述；只校验是合法 JSON，不解释业务含义 |

`Scope` 只有两种：

```go
settings.UserScope()          // Kind=user, Project=0
settings.ProjectScope(root)   // Kind=project, root 必须来自 settings.projects
```

校验规则：

1. `user` 不能带项目根；`project` 必须带正数项目根；其他种类拒绝。
2. 读取与写入都要求先 `RefreshSources`，否则返回 `ErrSourcesNotLoaded`；来源未声明返回 `ErrInvalidSource`。
3. 来源必须声明目标作用域，否则 `ErrScopeNotAllowed`。
4. 项目作用域要求 `RefreshProjects` 已完成，且项目根在授权清单中；否则 `ErrProjectsNotLoaded` / `ErrProjectNotAuthorized`。
5. `ResolveProject(ctx, cwd)` 把目录解析成项目根：目录必须与服务端返回的 `CWD` 完全一致，不做路径归一化猜测；未读取项目根时先读一次。

### 4.2 设置读取与版本前置条件

`Read` 返回 `Document{Source, Scope, Revision, Document}`。`Revision` 非空、`Document` 必须是 JSON 对象；否则整份拒绝（`ErrMalformedDocument`），不缓存半份文档。

`Update` 的顺序固定：

1. 校验来源声明、作用域、项目授权、`ExpectedRevision` 非空、文档是对象；项目作用域额外拒绝 `userOnly` 键（`ErrUserOnlySetting`）。
2. **本地版本前置条件**：若同一来源与作用域有缓存，且缓存版本不等于 `ExpectedRevision`，直接返回 `ErrRevisionConflict`，不发出任何写请求。
3. 经写登记登记一条 `KindManagementWrite` 并等待终态。透明参数形状：

```json
{
  "op": "settings.update",
  "source": "execution",
  "scope": {"kind": "project", "project": 7},
  "expectedRevision": "r1",
  "document": {"sandbox": false}
}
```

4. 成功：采用回执里的 `{revision, document}`；回执缺版本或文档不是对象时返回 `ErrMalformedWriteResult`，不采用本地拼出的替代值。
5. 明确失败：只读一次当前文档判断冲突。若当前版本不等于 `ExpectedRevision` → `ErrRevisionConflict`；否则 → `ErrUpdateFailed`。这一步只读，不发写请求。
6. 未知或需人工确认：与明确失败同样处理；本包不自动核对、不重发。

幂等键是进程内的 `management:<n>`，每次登记递增；同一进程内不复用。重启后写登记表为空，不会补发。

同一 `Manager` 的管理写入串行化：一次只登记一条设置写入或组件安装。这样两个并发写入即使都成功，也不会因为回执到达顺序不同而把较旧的文档写回缓存。

### 4.3 模型与 provider

`ModelCatalog` 是 provider 目录与当前选择：

- `Provider{ID, Name, Enabled, Models, EnabledModelIDs}`；标识必填且唯一，启用模型必须在该 provider 的模型清单里。
- `Selection{ProviderID, ModelID, Revision, ReasoningEffort, ServiceTier}`；可为空，表示服务端没有给出当前选择。`Revision` 是服务端给出的选择版本；空表示服务端没有给。

可选性判定（`ModelView.Selectable`）依次为：

1. provider 未启用 → 不可选，原因「provider 已停用」；
2. 模型不在 `EnabledModelIDs` → 不可选，原因「模型未启用」；
3. 账号可用模型清单尚未读取 → 不可选，原因取账号给出或「账号可用模型尚未读取」；
4. 模型标识不在账号可用清单里 → 不可选，原因取账号给出或「当前账号状态不包含该模型」；
5. 其余可选。

`SelectionView` 在目录里找不到对应 provider 或模型时 `Known=false`，只说明「无法确认」，不改变选择、不伪造选择。当前选择不可选时 `Selectable=false` 并带原因。

账号可用模型视图缺失（`Config.Account == nil`）时返回能力缺失；清单未就绪或为空时返回带原因的模型快照，所有模型不可选。本包不探测 provider、不自行判断账号状态、不派生可用模型。

### 4.4 模型选择写入

`SelectModel` 把当前选择写成服务端事实，流程与设置写入一致。调用方应从最近一次 `ModelSnapshot.Selection.Revision` 取版本前置条件；没有快照时本包会先读一次目录，若读到版本而调用方没给前置条件，返回 `ErrRevisionMissing`。

1. 校验 provider 与模型标识非空；要求服务端声明 `provider.select`；要求账号可用模型视图与写登记都存在。
2. 读取当前模型快照（没有缓存时先读一次），并用**此刻**的 `account.Models()` 重算可选性，不依赖上一次刷新留下的可选性。
3. 未登录、账号可用清单未就绪或为空、目标模型不在目录里、目标模型不可选，都返回 `*SelectionUnavailableError`（`errors.Is(err, ErrSelectionUnavailable)`）并带原因，不发出写请求。
4. 版本前置条件：
   - 快照里的选择版本非空、请求的 `ExpectedRevision` 为空 → `ErrRevisionMissing`；
   - 两者都非空且不同 → `ErrRevisionConflict`；
   - 两者都为空（服务端没有给出选择版本）→ 不带前置条件继续。
5. 经写登记登记一条 `KindManagementWrite`，透明参数：

```json
{
  "op": "provider.select",
  "providerId": "cloud",
  "modelId": "c1",
  "expectedRevision": "41"
}
```

6. 成功 → 重读模型目录，返回服务端确认的选择（`SelectionView.Revision` 与 `SelectionView.ProviderID/ModelID`），不做乐观更新；重读失败返回错误并说明状态未确认。
7. 未成功 → 只读一次模型目录判断版本：当前版本与期望不同 → `ErrRevisionConflict`；否则 → `ErrSelectionFailed` 并保留写登记给出的原因。不重发。

服务端的 `provider.select` 使用 `expectedVersion`；集成层把透明参数里的 `expectedRevision` 映射过去，并在为空时省略。选择写入与设置写入、组件安装共用同一条串行通道。

### 4.5 存储读数

`StorageReader` 返回 `StorageReading`，三个字段都是 `*int64`：

```go
type StorageReading struct {
    DatabaseBytes    *int64
    WALBytes         *int64
    ReclaimableBytes *int64
}
```

这样「服务端没有给出」与「服务端给了 0」在类型上就分得开：任一字段为 `nil` → `ErrMalformedStorage`；任一字段为负 → `ErrMalformedStorage`。只有三项都齐备且非负时才生成 `StorageReport` 并缓存。

本包没有清理、删除、容量估算的导出方法；存储读数只调用服务端的读数接口。回收空间由离线指令完成，属于运行手册而不是本包。

### 4.6 运行组件

`RuntimeEntry` 的 `Source` 只接受 `download` / `system`，`Status` 只接受 `off` / `acquiring` / `failed` / `ready` / `starting` / `down`；名称必填且唯一、来源与状态未知即整份拒绝。

`InstallRuntime` 只转达：

1. 名称为空 → `ErrInvalidRuntimeName`；
2. 服务端未声明 `runtime.install` → `ErrCapabilityMissing`，不发请求；
3. 经写登记登记 `KindManagementWrite`，参数 `{"op":"runtime.install","name":"uv"}`；
4. 等待终态；未成功 → `ErrInstallFailed`；
5. 成功 → 重读运行组件清单并返回该项目；清单里没有该项目 → `ErrRuntimeEntryMissing`；重读失败则返回错误并说明「状态未确认」，不冒充已就绪。

本包不下载、不校验、不写本地组件文件，也不轮询；状态跟随（`runtime.watch`）留给后续版本。

### 4.7 模态能力状态

`ModalityCapabilities` 是只读快照：

| 字段 | 说明 |
|---|---|
| `Supported` / `CanConfigure` | 服务端给出的能力门 |
| `TaskSupported` / `RealtimeSupported` | `*bool`；保留「没有给」与「给了 false」的区别 |
| `Revision` | `*string`；保留「没有给」与「给了空字符串」的区别 |
| `Categories[].Operations[]` | 类别、能力标识、操作名、可用性与原因 |
| `Default` / `Effective` / `Options` | 当前默认与有效模型、可选模型；原因原样保留 |
| `UnavailableReason` | 服务端给出的整体不可用原因；空表示没有给 |

服务端声明 `supported=false` 时返回快照与原因，这是事实而不是客户端错误；服务端没有声明 `modality.capabilities` 时返回 `ErrCapabilityMissing`，不发出请求。类别名、能力标识、操作名、模型引用与原因维度都做结构校验，缺失或重复即整份拒绝。

### 4.8 模态任务

任务面由 `ModalitySession` 的读取方法与 `KindModalityWrite` 的提交/取消组成：

| 操作 | 方法 | 走哪里 |
|---|---|---|
| 提交 | `SubmitTask` → `KindModalityWrite`，`op=task.submit` | 写登记；成功后用回执的 `InputID` 重读任务 |
| 读取状态 | `ReadTask` | 会话读取 `modality.task.status` |
| 清单分页 | `ListTasks` | 会话读取 `modality.task.list`；`Next` 原样带回 |
| 结果分页 | `ReadTaskResult` | 会话读取 `modality.task.result`；按 `Offset`/`Length` 递增到 `EOF` |
| 取消 | `CancelTask` → `KindModalityWrite`，`op=task.cancel` | 写登记；成功后重读任务 |

提交、取消的透明参数形状：

```json
{
  "op": "task.submit",
  "sessionId": "s1",
  "category": "image",
  "operation": "generate",
  "params": {"prompt": "猫"},
  "files": [{"name": "in.png", "mediaType": "image/png", "dataBase64": "..."}],
  "model": {"providerId": "p1", "modelId": "m1"},
  "node": "node-a",
  "clientTaskId": "local-1"
}
```

```json
{"op": "task.cancel", "sessionId": "s1", "taskId": "task-1"}
```

任务要求：

1. **能力门禁先于请求**：提交前要求服务端同时声明 `modality.task.submit` 与 `modality.task.status`；取消前要求 `modality.task.cancel` 与 `modality.task.status`。缺任一方法即返回能力缺失，不发写请求。读取、清单、结果分别要求对应方法。门禁先看 `hello.features`，再看会话方法表；本机真实服务端只在 `hello.features` 声明 `modality.*`。
2. **不猜默认参数**：`Category`、`Operation`、任务标识、结果文件名必填；`Params` 为空表示没有提供，不补 `{}`；输入文件必须且只能给 `dataBase64` 或 `nodePath`；模型引用必须同时有 provider 与模型标识。
3. **回执只给标识**：提交与取消的写登记回执用 `InputID` 传任务标识；本包据此重读任务。回执缺标识时返回 `ErrMalformedWriteResult`，不伪造任务。
4. **任务事实校验**：标识、类别、操作、状态、创建/更新时间必填；状态只能是 `queued`/`running`/`succeeded`/`failed`/`cancelled`/`interrupted`；结果文件与错误对象的必填字段都校验；缺字段即整份拒绝。
5. **分页不省略**：清单的 `State`/`Category`/`Limit`/`After` 只在调用方给出时下发；结果读取的 `Length` 限制在 `1..4 MiB`，偏移为负直接拒绝。
6. **只在内存缓存**：成功读取或提交后按会话与任务标识缓存最近确认值；重启后不保留，也不补发。

任务提交与取消都经同一条管理写入串行通道，不自行重试；失败时返回分类错误并保留写登记给出的原因。

### 4.9 实时传输

实时面由写登记与 `ModalitySession` 的发送方法组成：

| 操作 | 方法 | 走哪里 |
|---|---|---|
| 打开 | `OpenRealtime` → `KindModalityWrite`，`op=realtime.open` | 写登记；回执的 `InputID` 是连接标识 |
| 发送 | `SendRealtime` | 会话读取 `modality.realtime.send`；不经过写登记 |
| 关闭 | `CloseRealtime` → `KindModalityWrite`，`op=realtime.close` | 写登记；成功后移除本地连接 |
| 接收 | `DeliverRealtimeEvent` / `DeliverRealtimeEnded` | D12 把 `modality.realtime.event`/`ended` 通知投递给本包 |
| 订阅 | `WatchRealtime` | 本进程内按连接分发事件与结束通知 |

打开、关闭的透明参数形状：

```json
{"op": "realtime.open", "sessionId": "s1", "operation": "realtime.voice", "params": {"lang": "zh"}}
```

```json
{"op": "realtime.close", "sessionId": "s1", "connectionId": "conn-1"}
```

要求：

1. **能力门禁**：打开、发送、关闭分别要求服务端声明 `modality.realtime.open`、`modality.realtime.send`、`modality.realtime.close`；缺方法即返回能力缺失，零请求。门禁先看 `hello.features`，再看会话方法表。
2. **连接状态**：只有打开成功且回执带连接标识才建立本地状态；发送与关闭都要求本地状态存在，否则 `ErrRealtimeNotOpen`，不向服务端猜连接。
3. **不猜参数**：`Operation` 必填；`Params` 为空不补默认对象；帧类型只能是 `text`/`audio`/`control`，帧数据必须是合法 JSON，音频帧必须是合法 base64 字符串。
4. **不乐观更新**：打开成功后才登记连接；关闭成功后才移除并通知订阅者。打开或关闭失败保留原状态并返回 `ErrRealtimeOpenFailed`/`ErrRealtimeCloseFailed` 与原因。
5. **接收按事件**：本包不轮询实时事件；D12 收到通知后调用投递方法。未知连接的事件返回 `ErrRealtimeNotOpen`；结束事件幂等，重复到达不报错。
6. **订阅回调同步**：回调里不要阻塞，也不要调用本包的管理写入（与写登记共用串行通道）。

### 4.10 供应商与模型目录管理

provider 读取面：

| 接口 | 用途 |
|---|---|
| `RefreshProviders` / `Providers` | `provider.list` 概要清单与缓存 |
| `ReadProvider` / `Provider` | `provider.management` 详情与缓存；模型与偏好从这里读 |
| `ProbeProvider` | `provider.probe` 显式探测；**只读，不写配置** |

provider 写入面（全部 `KindManagementWrite`，先过 `provider.management` 与具体方法的能力门禁）：

| 方法 | op | 关键参数 |
|---|---|---|
| `CreateProvider` | `provider.create` | `providerId`（规范小写 UUID）、`displayName`、`connection`、`models`、`enabled` |
| `UpdateProvider` | `provider.update` | `providerId`、`expectedVersion`、`patch` |
| `SetProviderEnabled` | `provider.update` | `patch.enabled` |
| `SetModelsEnabled` | `provider.models.set_enabled` | `providerId`、`expectedVersion`、完整 `modelIds` |
| `SetModelEnabled` | `provider.models.set_enabled` | 先读详情计算完整集合，再写；已是目标状态则不写 |
| `EditModel` | `provider.models.edit` | `providerId`、`expectedVersion`、`modelId`、`edit` |
| `ResetModel` | `provider.models.reset` | `providerId`、`expectedVersion`、`modelId` |
| `UpdateModelPreferences` | `provider.preferences.update` | `providerId`、`expectedVersion`、`modelId`、`patch` |

规则：

1. **版本前置**：`expectedVersion` 必须为正；本地缓存版本不符直接返回 `ErrProviderConflict` 且不发请求；服务端拒绝后只读一次 `provider.management` 判断版本，版本变了报冲突，否则报对应写入失败并保留原因。
2. **不覆盖**：冲突只读不写；不自动重试。
3. **写入只传显式字段**：`ProviderPatch` 的 nil 字段不下发；模型编辑同样；凭据用 `keep/clear/set/set_ref` 显式表达。
4. **不存凭据**：`ProviderSummary`/`ProviderDetail` 只有 `CredentialConfigured`；请求里的 `Value`/`Reference` 只透传，不缓存、不回显、不写日志。
5. **新建标识**：真实服务端要求新建 provider ID 是规范小写 UUID；本包在发请求前拒绝其他形状。已有 provider 的读取/更新不受此限制。
6. **探测只读**：`ProbeProvider` 不登记写意图、不创建 provider；它把 `ProviderConnectionInput` 或 `ProviderPatch` 交给集成层调用 `provider.probe`。
7. **偏好**：`ModelPreferencePatch` 的 `set` 必须带值、`clear` 不带值；服务端拒绝（例如模型不支持推理偏好）时保留原因，不本地伪造偏好。

### 4.11 协作模型策略

```go
type CollaborationModelPolicy struct {
    Mode            CollaborationPolicyMode // agent / inherit / fixed
    ModelProviderID string                 // 仅 fixed
    ModelID         string                 // 仅 fixed
}
```

- `RefreshCollaborationPolicy` 读 `collaboration.model_policy`，返回 `{Revision, Policy}`；`CollaborationPolicy()` 返回最近确认值。
- `SetCollaborationPolicy` 必须带 `ExpectedRevision`；本地缓存 revision 不符时不发请求；服务端拒绝后只读一次 `collaboration.model_policy` 判断冲突；成功后重读，不做乐观更新。
- 透明参数：`{"op":"collaboration.set_model_policy","expectedRevision":"...","policy":{"mode":"fixed","model":{"providerId":"...","modelId":"..."}}}`。
- 缺 `collaboration.model_policy` 或 `collaboration.set_model_policy` 时返回能力缺失，零请求。
- `agent`/`inherit` 不得带模型；`fixed` 必须同时给 provider 与模型标识。

### 4.12 agent 档案更新

- `RefreshAgents` 读 `agent.list`，`ReadSessionAgent` 读 `session.agent.get`，`UpdateSessionAgent` 写 `session.agent.update`。
- 真实 `session.agent.update` 是会话贡献方法，**没有服务端 revision**；本包用 `ExpectedAgentID` 作为前置条件：写入前读 `session.agent.get`，当前 AgentID 必须等于 `ExpectedAgentID`（空表示未采用任何档案），否则 `ErrAgentConflict`，不发请求。
- 缺 `session.agent.get` 或 `session.agent.update`（按 `session.methods` 判定）时返回能力缺失，零请求。
- 写入经 `KindManagementWrite`，op `session.agent.update`；成功后重读 `session.agent.get`，返回服务端确认的 AgentID 与 `Changed`。
- 透明参数里的 `expectedAgentId` 是本地前置条件；真实服务端 `deny_unknown_fields` 会拒绝该字段，D12 写发送方必须**不把它下发**，只发 `sessionId`、`branchId`、`agentId`。
- `agent.list` 是内建方法，真实 hello.features 不一定声明；本包不按 hello 预判，直接调用，`unknown_method` 归为能力缺失。

### 4.13 能力缺失与错误分类

| 错误 | 触发 | 调用方动作 |
|---|---|---|
| `*CapabilityError` / `ErrCapabilityMissing` | 服务端未声明能力、会话方法表缺方法或读不到、集成层未提供读取接口、服务端回 `unknown_method`/`capability_unavailable`/`storage_unavailable` | 明说该能力缺失，不本地兜底 |
| `ErrSourcesNotLoaded` / `ErrProjectsNotLoaded` | 尚未读取来源或项目根 | 先刷新对应声明 |
| `ErrScopeNotAllowed` / `ErrProjectNotAuthorized` / `ErrUserOnlySetting` | 作用域或项目作用域的写入限制不成立 | 修正作用域或改用用户作用域 |
| `ErrRevisionMissing` / `ErrRevisionConflict` | 缺版本前置条件或版本已变 | 重新读取后再决定是否写入 |
| `ErrSelectionUnavailable` | 未登录、没有可用模型或目标模型不可选 | 明说原因，不猜默认值 |
| `ErrInvalidModalityRequest` | 模态任务/实时参数结构不合法 | 修正请求，不发请求 |
| `ErrRealtimeNotOpen` | 本地没有该连接或已结束 | 重新打开或刷新连接状态 |
| `ErrTaskSubmitFailed` / `ErrTaskCancelFailed` / `ErrTaskReadFailed` / `ErrTaskListFailed` / `ErrTaskResultFailed` | 任务写入或会话读取失败 | 读原因后由调用方决定是否重试 |
| `ErrRealtimeOpenFailed` / `ErrRealtimeSendFailed` / `ErrRealtimeCloseFailed` | 实时写入或发送失败 | 读原因后由调用方决定是否重试 |
| `ErrUpdateFailed` / `ErrSelectionFailed` / `ErrInstallFailed` | 写登记明确失败 | 读结果与原因后由调用方决定是否重试 |
| `ErrMalformedSources/Projects/Document/Models/Storage/Runtimes/Modality` | 服务端事实结构不合法 | 报事实不合法，不采用半份 |
| `ErrMalformedWriteResult` | 写入成功回执不是可采用的文档或缺少资源标识 | 保留原状态，由调用方重新读取 |
| `ErrWriteUnavailable` | 没有写登记 | 集成层补齐写登记后重试动作 |

能力名常量：`FeatureSettingsSources`、`FeatureSettingsProjects`、`FeatureSettingsRead`、`FeatureSettingsUpdate`、`FeatureModelProviders`、`FeatureModelSelection`、`FeatureStorageReport`、`FeatureRuntimeList`、`FeatureRuntimeInstall`、`FeatureModalityCapabilities`。会话模态方法名常量：`ModalityMethodTaskSubmit`、`ModalityMethodTaskStatus`、`ModalityMethodTaskList`、`ModalityMethodTaskCancel`、`ModalityMethodTaskResult`、`ModalityMethodRealtimeOpen`、`ModalityMethodRealtimeSend`、`ModalityMethodRealtimeClose`。`Config.Features` 为 nil 时不做声明检查，只按调用结果分类；集成层提供 `*daemon.Connection` 即可获得声明检查；会话方法表由 `ModalitySession` 的 `SupportsModalityMethod` 回答。

## 5. 与 A/R 的对应

| 需求或架构条目 | 本领域实现 | 不在本领域 |
|---|---|---|
| R 块 9 运行组件首次获取与安装 | 状态读取、`InstallRuntime` 转达、安装后重读 | 组件的下载与校验（服务端） |
| R 块 9 存储读数与清理边界 | 只读 `RefreshStorage`；无可回收动作的方法 | 离线清理指令、容量估算 |
| R 块 9 模态能力状态 | 只读 `RefreshModality`，声明不支持时给原因 | 能力配置写入（不在本版） |
| R 块 9 模态任务提交与结果 | `SubmitTask`/`CancelTask` 经 `KindModalityWrite`；`ReadTask`/`ListTasks`/`ReadTaskResult` 走会话读取与分页 | 任务等待策略、媒体落盘 |
| R 块 9 模态实时传输 | `OpenRealtime`/`CloseRealtime` 经 `KindModalityWrite`；`SendRealtime` 走会话；`DeliverRealtimeEvent`/`Ended` 按事件接收 | 事件路由（D12）、音频编解码与播放 |
| R 块 10 模型可用性过滤 | `RefreshModels` 按 `account.ModelList` 过滤 | — |
| R 块 10 模型档位选择 | `SelectModel` 经写登记转达 `provider.select`，成功后重读回填 | 界面选择入口 |
| provider 列表/新建/更新/启用/探测 | `RefreshProviders`/`ReadProvider`/`ProbeProvider`、`CreateProvider`/`UpdateProvider`/`SetProviderEnabled` | 远程模型刷新、provider 删除、导入旧配置 |
| 模型列表/编辑/启用/重置/偏好 | `ProviderDetail.Models`、`SetModelsEnabled`/`SetModelEnabled`/`EditModel`/`ResetModel`/`UpdateModelPreferences` | 模型删除、模型新建、账户策略 |
| 协作模型策略 | `RefreshCollaborationPolicy`/`SetCollaborationPolicy`，revision 前置与冲突 | 子代理运行时的模型选择（执行归服务端） |
| agent 档案更新 | `RefreshAgents`/`ReadSessionAgent`/`UpdateSessionAgent`，当前 agent 前置与读回 | agent 创建/删除、档案字段编辑 |
| R 块 8 设置写入（版本前置条件、冲突） | 来源声明、项目作用域、`ExpectedRevision`、本地与服务端冲突判定 | 插件、技能、钩子的具体设置键 |
| A 管理面读数镜像状态 | 只有本包的刷新方法与写登记回执能改读数 | 界面展示 |
| A 四条总规矩 | 只提交意图、不写事实；不做乐观更新；读数单一驱动者；不显示未确认内容 | — |
| A 检查项「缺能力就报缺失」 | 统一 `CapabilityError`，不构造替代读数 | — |
| A 检查项「一个状态只有一个驱动者」 | 缓存只在 `Manager` 内部改写 | — |
| R8 本地状态边界 | 所有缓存与幂等键只在内存 | — |

## 6. 验收

### 6.1 命令与结果

```bash
go test ./internal/settings
go test -race ./internal/settings
go test ./...
go vet ./...
```

本版结果：`go test ./internal/settings`、`go test -race ./internal/settings`、`go vet ./internal/settings` 全部通过；`internal/settings` 104 个顶层测试、53 个子测试全绿，语句覆盖率 82.1%（2026-11-11，本机 macOS arm64）。
全仓 `go test ./...` 当前在 `internal/account` 的 `TestLoginRejectsConcurrentAttempt` 失败（账号领域正在改，不在本领域）；`go vet ./...` 通过。该失败与本领域无关，单独说明。

### 6.2 覆盖

| 验收要求 | 测试 |
|---|---|
| 设置来源读取与缓存、返回值隔离 | `TestRefreshSourcesReadsAndCaches` |
| 来源事实不合法（缺名、重名、未知/重复作用域、空/重复只读键、非法 schema） | `TestRefreshSourcesRejectsMalformed` |
| 缺来源能力、普通读取失败、`unknown_method` 分类 | `TestRefreshSourcesCapabilityAndReadFailure` |
| 项目根读取、按目录解析、未授权目录、空目录 | `TestRefreshProjectsAndResolution` |
| 项目根事实不合法 | `TestRefreshProjectsRejectsMalformed` |
| 未读来源拒绝读取、来源未声明、未知作用域、用户作用域来源拒绝项目作用域 | `TestReadRequiresSourcesAndDeclaredScope` |
| 项目作用域要求授权根 | `TestReadProjectScopeRequiresAuthorizedRoot` |
| 设置文档缺版本、非对象拒绝 | `TestReadRejectsMalformedDocument` |
| 只写来源不可读 | `TestReadRejectsWriteOnlySource` |
| 写入经管理写入只发一次、参数形状、缓存更新 | `TestUpdateWritesOnceThroughManagementWrite` |
| 本地版本冲突不发请求、不覆盖缓存 | `TestUpdateLocalRevisionConflictDoesNotWrite` |
| 服务端版本冲突不重发、只读一次当前文档 | `TestUpdateServerConflictIsNotRetried` |
| 失败但版本未变 → 写入失败而非冲突 | `TestUpdateFailureWithoutRevisionChange` |
| 缺版本前置条件、项目作用域写用户专属键 | `TestUpdateRequiresPreconditionAndRejectsUserOnlyKeys` |
| 缺写登记、缺写入能力 | `TestUpdateWithoutWriterIsCapabilityStyle`、`TestUpdateRequiresAdvertisedCapability` |
| 写入回执不合法不采用 | `TestUpdateMalformedReplyIsNotAdopted` |
| 模型可选性按账号清单过滤、当前选择可选性 | `TestRefreshModelsFiltersByAccount` |
| 账号可用清单未就绪：全部不可选并带原因 | `TestRefreshModelsAvailabilityNotReady` |
| 选择指向未启用模型或目录外模型 | `TestRefreshModelsSelectionEdgeCases` |
| 缺目录/缺账号视图/目录读取失败 | `TestRefreshModelsCapabilityErrors` |
| 目录不合法（provider/模型标识、启用清单、选择） | `TestRefreshModelsRejectsMalformedCatalog` |
| 模型快照隔离 | `TestModelsSnapshotIsDetached` |
| 存储读数正确且只读一次 | `TestRefreshStorageIsReadOnlyReport` |
| 存储读数缺项或负数拒绝、不缓存 | `TestRefreshStorageRejectsMissingOrNegative` |
| 存储能力缺失、`storage_unavailable`、读取失败分类 | `TestStorageCapabilityAndReadFailure` |
| 运行组件读取、单个查询 | `TestRefreshRuntimesAndLookup` |
| 运行组件事实不合法（缺名、重名、未知来源/状态） | `TestRefreshRuntimesRejectsMalformed` |
| 安装转达只发一次、经管理写入、安装后重读 | `TestInstallRuntimeRelaysThroughWrite` |
| 安装失败、空名称、缺写登记、缺安装能力 | `TestInstallRuntimeFailureAndCapabilities` |
| 安装成功但清单缺失 | `TestInstallRuntimeMissingEntryAfterSuccess` |
| 模态能力状态、原因、可选位与版本 | `TestRefreshModalityReadsStateAndReasons` |
| 服务端声明不支持是快照不是错误 | `TestRefreshModalityUnsupportedIsNotAnError` |
| 模态能力缺失与事实不合法 | `TestRefreshModalityCapabilityAndMalformed` |
| 所有缺失项共享一个分类且带能力名 | `TestCapabilityErrorsShareOneClassification` |
| 任务提交只发一次、参数形状、提交后重读并回填 | `TestSubmitTaskWritesOnceAndReadsBack` |
| 同一客户端任务标识只提交一次 | `TestSubmitTaskDuplicateClientIDIsSentOnce` |
| 任务取消只发一次、取消后重读 | `TestCancelTaskWritesOnceAndReadsBack` |
| 提交与取消失败保留写登记原因 | `TestSubmitTaskFailureKeepsReason`、`TestCancelTaskFailureKeepsReason` |
| 缺提交/状态方法或方法表不可读时零请求 | `TestSubmitTaskCapabilityGate` |
| 任务清单按游标分页、回填缓存 | `TestListTasksPaginates` |
| 任务清单非法查询零调用 | `TestListTasksRejectsInvalidQueryWithoutCall` |
| 任务结果按偏移/长度分页 | `TestReadTaskResultPaginates` |
| 任务结果非法查询零调用 | `TestReadTaskResultRejectsInvalidQueryWithoutCall` |
| 任务状态读取与事实校验 | `TestReadTaskAndMalformedFacts` |
| 任务读取失败分类与非法 base64 拒绝 | `TestModalitySessionReadFailuresAndBadBase64` |
| 任务事实不合法（缺字段、未知状态、结果文件缺摘要） | `TestModalityTaskValidationRejectsMalformedFacts` |
| 提交/打开回执缺资源标识拒绝 | `TestModalityWriteResultRequiresResourceID` |
| 实时打开、发送、关闭、事件与结束通知 | `TestRealtimeOpenSendClose` |
| 实时打开/发送/关闭缺方法时零请求 | `TestRealtimeCapabilityGate` |
| 实时发送失败保留原因与连接状态 | `TestRealtimeSendFailureKeepsReason` |
| 实时关闭失败保留连接状态 | `TestRealtimeCloseFailureKeepsState` |
| 未知连接的事件与结束事件边界 | `TestRealtimeDeliverUnknownAndEndedAfterClose` |
| 非法实时帧零请求 | `TestRealtimeRejectsInvalidFramesWithoutCall` |
| 实时订阅未知连接、空回调与关闭未知连接 | `TestRealtimeWatchAndCloseEdgeCases` |
| 选择写入成功并回填、参数形状、重读一次 | `TestSelectModelWritesAndRefreshes` |
| 选择写入本地版本冲突不发请求 | `TestSelectModelLocalConflictDoesNotWrite` |
| 选择写入服务端冲突只读一次、不重发 | `TestSelectModelServerConflictReadsOnce` |
| 未登录、目标模型不在账号清单、目录里没有该模型时不得写入 | `TestSelectModelRejectsUnavailable` |
| 选择写入失败保留原因、不乐观更新 | `TestSelectModelFailureKeepsReason` |
| 已知选择版本时必须给前置条件 | `TestSelectModelRequiresRevisionWhenKnown` |
| 选择写入的缺能力/缺账号/缺写登记分类 | `TestSelectModelCapabilities` |
| provider 清单/详情读取、缓存隔离、事实校验 | `TestRefreshProvidersAndReadDetail`、`TestProviderFactsRejectMalformed` |
| 显式探测只读、能力门禁与参数校验 | `TestProbeProviderIsReadOnly`、`TestProbeProviderCapabilityAndValidation` |
| provider 新建一次写入、显式字段、重读 | `TestCreateProviderWritesOnceAndReadsBack`、`TestCreateProviderValidationAndCapability` |
| provider 本地/服务端版本冲突、失败保留原因 | `TestUpdateProviderLocalConflictDoesNotWrite`、`TestUpdateProviderServerConflictReadsOnce`、`TestUpdateProviderFailureKeepsReason` |
| provider 启用状态、显式启用集合 | `TestSetProviderEnabledAndModels` |
| 单模型启用/停用与无变化不写 | `TestSetModelEnabledToggles` |
| 模型编辑、重置、偏好写入 | `TestEditResetAndPreferencesWrites` |
| provider/模型写入缺能力零请求 | `TestProviderWritesRequireCapabilities` |
| provider 显式字段与凭据只透传 | `TestProviderPatchFieldsAndCredentialsAreExplicit`、`TestProviderPatchValidation` |
| 模型编辑多字段与偏好 clear/set | `TestModelEditAndPreferenceFields` |
| 新建 UUID 与托管 provider 不可编辑 | `TestProviderCreateRequiresUUIDAndManagedNotEditable` |
| provider 读取缺能力分类 | `TestProviderReadCapabilities` |
| 协作策略读取、缓存与事实校验 | `TestRefreshCollaborationPolicy` |
| 协作策略写入一次、重读与参数形状 | `TestSetCollaborationPolicyWritesOnceAndRefreshes` |
| 协作策略本地/服务端版本冲突 | `TestSetCollaborationPolicyConflicts` |
| 协作策略失败原因与参数校验 | `TestSetCollaborationPolicyFailureAndValidation` |
| 协作策略缺能力零请求 | `TestCollaborationPolicyCapabilities` |
| agent 清单读取、缓存与事实校验 | `TestRefreshAgentsAndValidation` |
| 会话当前 agent 读取与参数校验 | `TestReadSessionAgent` |
| 会话 agent 更新一次、参数与重读 | `TestUpdateSessionAgentWritesOnceAndReadsBack` |
| agent 当前值冲突、缺方法与方法表失败零请求 | `TestUpdateSessionAgentConflictAndCapability` |
| agent 写入失败保留原因、服务端变化转冲突、回读不一致 | `TestUpdateSessionAgentFailureKeepsReasonAndReadsOnce`、`TestUpdateSessionAgentServerChangedBecomesConflict`、`TestUpdateSessionAgentReadbackMismatch` |
| agent 缺接口/缺写登记分类 | `TestAgentCapabilitiesAndWriter` |
| fakeappserver：设置来源/项目/读取/写入/本地冲突/用户专属键 | `TestAppServerSettingsProjectScopeAndRevision` |
| fakeappserver：服务端版本冲突不覆盖 | `TestAppServerSettingsServerConflictNotOverwritten` |
| fakeappserver：模型与 provider 目录、账号过滤、当前选择 | `TestAppServerModelsFilteredByAccount` |
| fakeappserver：选择写入成功并回填、只发一次 | `TestAppServerSelectModelWritesAndRefreshes` |
| fakeappserver：选择写入服务端冲突不重发 | `TestAppServerSelectModelServerConflict` |
| fakeappserver：未登录时不得写入选择 | `TestAppServerSelectModelSignedOut` |
| 真实 alpha.21：provider 列表、新建、探测、更新、启用、模型开关、编辑、重置、偏好、版本冲突 | `scripts/verify-provider-admin.py`（§6.7） |
| 真实 alpha.21：协作策略 revision/冲突、agent 方法表/采用/读回 | `scripts/verify-collaboration-agent.py`（§6.8） |
| fakeappserver：未登录时无可用模型并带原因 | `TestAppServerModelsSignedOut` |
| fakeappserver：存储读数只读且无删除动作；缺能力不发请求 | `TestAppServerStorageReportIsReadOnly` |
| fakeappserver：运行组件状态与安装转达、安装后重读 | `TestAppServerRuntimeInstallRelay` |
| fakeappserver：模态能力状态、原因、不支持快照、缺能力不发请求 | `TestAppServerModalityCapabilities` |

### 6.3 与 D1/D3/D6 的接口形状

| 提供方 | 形状 | 本包消费方式 |
|---|---|---|
| D1 连接 | `protocol.Client.Call` / `CallInto` | 集成层实现 `SourcesReader`、`SettingsReader`、`ProjectsReader`、`ModelCatalogReader`、`StorageReader`、`RuntimeReader`、`ModalityReader`，把 wire 回执映射成领域类型；方法名不出现在本包；选择写入的 `provider.select` 由写发送方映射 |
| D1 连接 | `hello.features` / `protocol.Client.Call` | D12 用 `hello.features` 实现 `ModalitySession.SupportsModalityMethod`（本机真实服务端只在 hello 声明 `modality.*`），用控制连接调用任务读取、清单、结果与实时发送；若未来服务端把模态贡献为会话方法，再走 `Subscription.Methods/Call`。本包不定义 wire 结构 |
| D1 能力声明 | `*daemon.Connection.HasFeature(name)` | 直接作为 `Config.Features`；缺失项在发请求前返回 `CapabilityError` |
| D1 错误分类 | `protocol.RPCError.AppError()` / `Code` | 只把 `unknown_method`、`capability_unavailable`、`storage_unavailable` 归为能力缺失；其他错误保持读取失败 |
| D3 写登记 | `*write.Manager.Register` / `Wait` | 直接满足 `Writer`；设置写入参数 `{"op":"settings.update",...}`，选择写入参数 `{"op":"provider.select",...}`，安装参数 `{"op":"runtime.install","name":...}`；provider 管理用 `{"op":"provider.create"/"provider.update"/"provider.models.set_enabled"/"provider.models.edit"/"provider.models.reset"/"provider.preferences.update",...}`；模态写入用 `KindModalityWrite`；只登记一次，不重试 |
| D12 集成层 | `ModalitySession` 与模态通知路由 | D12 实现 hello 能力门禁、任务读取与实时发送；把 `modality.realtime.event` 与 `ended` 通知投递给 `DeliverRealtimeEvent`/`DeliverRealtimeEnded`；写发送方按 `KindModalityWrite` 路由到会话流并回填 `Reply.InputID` |
| D3 结果语义 | `StatusSucceeded` / `Failed` / `Unknown` / `NeedsManualConfirmation` | 只有成功采用回执；其余按失败处理，失败后只读一次文档判断冲突 |
| D6 账号 | `*account.Account.Models() account.ModelList` | 作为唯一可用性来源；不在本包重复读取账号状态、不派生模型 |
| D6 模型标识 | `account.Model.ID` | 与 provider 目录的 `models[].id` 完全匹配；匹配口径见 §7 第 3 条 |
| 选择版本 | `provider.selection` 的 `version` | 集成层映射成 `Selection.Revision`；选择写入把同一值通过 `expectedRevision` 带回 |
| U1 界面 | `Sources`、`Projects`、`Document`、`ModelSnapshot`、`StorageReport`、`[]RuntimeEntry`、`ModalityCapabilities`、`ModalityTask`/`ModalityTaskPage`、`ModalityRealtimeConnection` | 只读；写入只调 `Update` / `InstallRuntime` / `SelectModel` / `SubmitTask` / `CancelTask` / `OpenRealtime` / `SendRealtime` / `CloseRealtime` |

本包只导入 `internal/write`、`internal/account` 与 `internal/appserver/protocol`（仅错误分类）；不导入 D1 的业务包。读取接口与模态调用接口都由 D12 实现，本包不感知 socket、控制连接、会话流或 wire。

### 6.4 blocked

| 项 | 状态 | 缺什么 |
|---|---|---|
| 本包 Go API 对真实 App Server 的设置来源、项目根、设置读写与版本冲突 | blocked | 需要冻结的 wire 方法名、作用域形状与回执字段；本版用 `fakeappserver` 验证。真实模型工具调用已另行验证（§6.5） |
| 真实账号与真实 provider 目录的可用性过滤 | blocked | 需要真实账号、真实 provider 目录与 D6 的模型标识口径；本版用假服务端与真实 `account` 包验证 |
| 真实模型选择写入与版本冲突 | blocked | 需要真实账号、真实 provider 与 `provider.select` 的版本语义；本版用假服务端验证 |
| 真实存储读数与离线回收边界 | blocked | 需要真实服务端原生存储；本版验证只读与结构校验，未执行任何清理 |
| 真实运行组件安装 | blocked | 需要真实服务端与组件包；本版只验证转达、状态重读与失败分类 |
| 本包 Go API 对真实 provider/模型管理 | blocked | 真实 alpha.21 的列表/新建/探测/更新/启用/模型配置/偏好/冲突已由脚本直连验证（§6.7）；本包 Go API 仍依赖 D12 的 `ProviderAdmin` 与写发送方适配器 |
| 本包 Go API 对真实协作策略/agent 档案 | blocked | 真实 alpha.21 的协作 revision/冲突与 agent get/update 已由脚本直连验证（§6.8）；本包 Go API 仍依赖 D12 的 `CollaborationReader`/`AgentAdmin` 与写发送方适配器 |
| 本包 Go API 对真实模态能力/任务/实时 | blocked | 真实 daemon 的能力、OCR 任务与取消已由脚本验证（§6.6）；但本包 Go API 仍依赖 D12 的 `ModalitySession` 适配器，尚未接通 |
| 实时传输 | blocked | 当前 provider/adapter 没有可用实时模型：`realtime.voice/transcribe/avatar` 均为 adapter 不可用；无法验证打开→发送→关闭 |
| 模态方法在会话流上的路由 | blocked | 本机真实服务端把 `modality.*` 声明在 `hello.features`，不贡献会话方法；D12 需按 hello 能力实现门禁与路由 |

### 6.5 真实模型接入验证（2026-11-11）

**侦察（只读）**

- `~/.dimcode-alpha/config.json` 只有外观与权限开关，不含 provider/model。
- `~/.dimcode-alpha/auth.json` 是 Dim 账号 OAuth 记录；账号已登录，subject `63`，issuer `https://dimagent.cn`。令牌不写入本文档。
- 提供 provider 与模型事实的是原生存储 `~/.dimcode-alpha/rust/dimcode.sqlite`：
  - `providers`：`dimcode-api-oauth`，`kind=managed`，`version=5`，模型条目 33 个。
  - `provider_selections`：`scope=global` → `dimcode-api-oauth` / `deepseek-v4.1-flash`。
  - 目标模型声明 `toolCalling=true`、`streaming=true`、`reasoning=true`。
- 侦察只做 SQLite 只读查询与文本读取；未修改、未删除、未写回来源。

**夹具**

- 位置：`/tmp/dimagent-real-model`（只在 `/tmp`，不进入仓库）。
- 内容：`home/config.json`、`home/auth.json`（0600）、`home/rust/dimcode.sqlite`
  （来源 schema + `agent_store_format=36` + 1 条 provider + 1 条 selection +
  6 条非敏感设置文档；**0 条会话/时间线/消息**）、`bin/dimcode-server`
  （0.1.0-alpha.21）、`meta/schema.sql`、`FIXTURE.md`。
- 权限：夹具根、`home`、`home/rust` 为 0700；`auth.json` 为 0600。
- 形状与复制清单见夹具内 `FIXTURE.md`；真实凭据只存在 `/tmp`，不写入仓库、
  不写回来源、不进入产品代码路径。

**验证脚本**

`scripts/verify-real-model.sh` 不读取 `~/.dimcode-alpha` 或任何来源目录，只读
`--fixture`（默认 `/tmp/dimagent-real-model`）。它把 `fixture/home` 复制到新的
临时目录，起 daemon，确认账号已登录，跑一次 `exec`，解析 `session.event`：

- `step/tool-call` 必须出现，且带工具名与输入；
- `step/tool-result` 必须出现，且包含 `--expect` 文本；
- 两者都满足才打印 `REAL_TOOL_CALL_OK` 并以 0 退出；否则以 1 退出。

用法：

```bash
scripts/verify-real-model.sh --fixture /tmp/dimagent-real-model
```

**观察到的结果**

```text
account: Signed in: 63 (https://dimagent.cn)
{"sessionId":"40fe352f-...","toolCalls":[{"tool":"exec","input":{"command":"printf DIMAGENT_REAL_TOOL_OK","timeoutMs":10000}}],"toolResults":["DIMAGENT_REAL_TOOL_OK"],"expected":"DIMAGENT_REAL_TOOL_OK","ok":true}
REAL_TOOL_CALL_OK
```

结论：真实 provider、真实账号、真实 daemon 产生了一次真实工具调用（`exec`
执行 `printf DIMAGENT_REAL_TOOL_OK`，退出码 0，输出 `DIMAGENT_REAL_TOOL_OK`），
不是纯文本回复，也不是 mock。验收通过。

**数据源纪律**

- 来源目录只读；只把必要配置复制到 `/tmp` 夹具，未写回来源。
- 产品代码与领域包不读取 `~/.dimcode-alpha`、Dim 源码或 dim-electron 源码。
- 脚本只读夹具，每次运行把夹具复制到独立临时目录，不修改夹具本身。

### 6.6 真实模态任务与实时证据（2026-11-11）

**能力清单**

`modality.capabilities`（daemon 级，`{}` 参数）返回 9 个类别，可用操作只有两个：

| 操作 | 可用 | 有效模型 | 选项数 |
|---|---|---|---|
| `image.understand` | 是 | `dimcode-api-oauth` / `deepseek-v4.1-flash` | 19 |
| `ocr.recognize` | 是 | `local` / `apple-vision` | 1 |

其余类别不可用的原因：`image.generate/edit/enhance` 为 adapter 未提供图像模型列表；
`video`/`speech`/`music`/`audio`/`asset`/`motion` 为 build 不含；`realtime.*` 为
adapter 没有可用实时适配器。

**方法门禁的真实形状**

- `hello.features` 声明 `modality.capabilities`、`modality.task.submit/status/list/result/cancel`、
  `modality.realtime.open/send/close`。
- `session.methods`（新建会话）**不包含** `modality.*`；本机服务端把模态作为
  daemon 级方法，而不是会话贡献方法。
- 因此 `ModalitySession.SupportsModalityMethod` 必须接受 `hello.features`；
  只查 `session.methods` 会把可用方法误判为缺失。D12 的实现已按“hello 优先、
  会话方法表兜底”接线。

**任务：OCR 成功且结果分页读完**

```bash
scripts/verify-modality.py --fixture /tmp/dimagent-real-model   --server ~/.dimcode-alpha/rust/dimcode-server   --image /tmp/dimagent-modality-ocr.png
```

观察（只记录状态、标识、字节数，未打印 OCR 文本）：

```text
available operations: ['image.understand', 'ocr.recognize']
task: mt_b7a38d15-... states=['queued', 'running', 'succeeded'] terminal=succeeded
  result result.json: bytes=94 declared=94 pages=2 eof=True
  result text.txt: bytes=0 declared=0 pages=1 eof=True
cancel: mt_eebec059-... reply=cancelled final=cancelled
```

- 状态确实经历 `queued → running → succeeded`。
- `result.json` 用 64 字节分页读了两页并到 `EOF`；字节数与任务声明的 94 一致。
- `text.txt` 为 0 字节，一页即 `EOF`。
- 取消：第二个任务提交后立即取消，取消回执与最终状态都是 `cancelled`。

**实时：blocked**

`realtime` 类别不可用，三个操作的原因均为 adapter：

```text
voice:      No available adapter serves realtime.voice
transcribe: No available adapter serves realtime.transcribe
avatar:     No available adapter serves realtime.avatar
```

尝试打开 `realtime.voice` 返回 `modality_unknown_operation`；用 `voice` 打开返回
`no_default_model`（没有模型服务该操作）。因此打开→发送→关闭无法在本机验证，
标 blocked，原因不是客户端缺方法，而是当前 provider/adapter 没有可用实时模型。

**macOS OCR 路径限制**

Apple Vision 权限按可执行文件路径生效。用夹具里复制出来的
`bin/dimcode-server` 跑 OCR 会得到
`executor_error: Apple Vision text recognition failed ...`；用已安装路径
`~/.dimcode-alpha/rust/dimcode-server`（或 npm 安装目录里的同一二进制）成功。
脚本因此提供 `--server`，文档命令显式使用已安装路径；夹具仍是 `/tmp`，来源只读。

**数据纪律**

- 夹具与图像只在 `/tmp`；来源目录只读；未写回来源。
- 证据只含状态、任务/文件标识、字节数、分页数、能力原因；未打印 OCR 文本、
  图像内容或凭据。
- 脚本每次把夹具复制到新的临时目录，运行结束后清理（`--keep` 可保留）。

### 6.7 真实 provider/模型管理证据（2026-11-11）

命令：

```bash
scripts/verify-provider-admin.py --fixture /tmp/dimagent-real-model   --server ~/.dimcode-alpha/rust/dimcode-server
```

真实 alpha.21 环境、匿名自定义 provider、本地 mock `/models` 端点；结果摘要：

```text
provider list: 127 -> 128
created: version=1 modelCount=2 enabled=True
probe: modelCount=2 ids=['probe-1','probe-2'] contextWindows=[32000,16000] createdProvider=False
update: version=2 displayName='Dim Evidence Updated'
enable: version=3 enabled=False -> version=4 enabled=True
models: version=5 evidence-1=True evidence-2=False
edit: version=6 evidence-1 displayName='Evidence One Edited' contextWindow=2000
reset: version=7 evidence-1 displayName='Evidence 1' contextWindow=1000
preferences: version=8 written=True
stale conflict: appError=provider_conflict, name/version unchanged
custom in list: True
```

结论：

- `provider.list` 真实返回 127 个内置 provider；新建自定义 provider 后 128，且清单里能看到它。
- `provider.probe` 真实返回 2 个模型（含 contextWindow/maxOutputTokens），且**没有创建** provider；探测是只读的。
- `provider.update` 真实把显示名从 `Dim Evidence` 改为 `Dim Evidence Updated`，版本 1→2；启用状态 false/true 各推进一版。
- `provider.models.set_enabled` 把 `evidence-2` 停用，`evidence-1` 保持启用。
- `provider.models.edit` 把 `evidence-1` 的显示名与上下文/输出上限改大；`provider.models.reset` 把它们恢复到初始值。
- `provider.preferences.update` 先用 `reasoningEffort=set` 被服务端拒绝（模型不支持推理偏好），再用 `serviceTier=clear` 成功，版本 7→8；偏好保持为空。
- 过期版本更新返回 `appError=provider_conflict`，名称与版本都没有被覆盖。
- 全程使用匿名认证和本地 mock 端点，没有读取/打印真实凭据；证据只保留 provider/模型标识、版本、状态、计数与错误分类。

### 6.8 真实协作策略与 agent 档案证据（2026-11-11）

命令：

```bash
scripts/verify-collaboration-agent.py --fixture /tmp/dimagent-real-model   --server ~/.dimcode-alpha/rust/dimcode-server
```

观察（只保留 revision、状态、agent 标识与错误分类）：

```text
policy initial: revision=native:user/collaboration:1 mode=agent
policy revisions: 1 -> 2 -> 3 -> 4
policy writes: fixed(r2) -> inherit(r3) -> agent(r4)
policy stale conflict: appError=collaboration_model_policy_conflict
  revisionAfter=native:user/collaboration:4 modeAfter=agent
agent methods: get=True update=True
agent list count: 1
agent before: {}
agent: agentId=54026405-... updateOutcome=adopted
  after={'agentId': '54026405-...'} sameOutcome=unchanged
```

结论：

- 协作策略真实读取 revision 与 policy；三次写入分别推进 revision 2/3/4，读回 mode 正确。
- 用旧 revision 写入返回 `collaboration_model_policy_conflict`，当前 revision 与 mode 保持不变。
- `session.methods` 真实包含 `session.agent.get` 与 `session.agent.update`；`agent.list` 可用但**不在 hello.features**，所以本包对 `agent.list` 不用 hello 预判。
- 会话初始未采用 agent（`agent.get` 返回 `{}`）；创建 prompt-only agent 后 `session.agent.update` 返回 `adopted`，`agent.get` 读回该 agentId；重复更新返回 `unchanged`。
- 真实 `session.agent.update` 不接受 `expectedAgentId` 字段（`invalid_params`），所以该前置条件必须只留在本包，D12 写发送方不能下发。
- 全程未打印凭据；只创建临时会话与 prompt-only 档案，运行在 `/tmp` 夹具副本上。

## 7. 已知限制

1. **wire 未冻结**：测试适配器里的方法名（`settings.sources`、`settings.projects`、`settings.read`、`settings.update`、`models.providers.list`、`provider.select`、`storage.report`、`runtime.list`、`runtime.install`、`modality.capabilities`、`modality.task.*`、`modality.realtime.*`、`account.status`、`model.list`）只是测试约定，不代表协议冻结。
2. **必须先读声明**：`Read`/`Update` 要求先 `RefreshSources`，项目作用域还要求先 `RefreshProjects`。本包不自动刷新声明，避免一次调用变成多次隐式请求。
3. **模型标识口径未定**：可用性按 `account.Model.ID` 与 provider 目录 `models[].id` 完全匹配。若 D6 的模型标识带 provider 限定，或不同 provider 复用同一模型标识，可选性会偏保守；需要在 D1/D6 冻结口径后调整匹配函数。
4. **布尔与可选字段的缺省**：领域类型里的 `Provider.Enabled`、`RuntimeEntry.Auto` 是普通布尔值，无法区分「服务端没有给」与「给了 false」。集成层必须在映射时拒绝缺失的必填布尔值（本版测试适配器已这样处理）；否则会把缺项当成 false。
5. **存储缺项依赖集成层**：`StorageReading` 用指针保留缺项，但集成层必须把缺失字段映射成 `nil`；若映射成 0，本包无法区分「0 字节」与「没有给」。接口契约已写明，测试适配器按指针解码。
6. **冲突判定依赖一次读**：服务端拒绝写入后，本包只读一次当前事实（设置文档或模型目录）判断冲突；若这次读失败，只能报对应的写入失败错误，无法确认是否冲突。本包不会为了判断冲突而重发写请求。
7. **本地冲突依赖缓存**：设置写入没有缓存时由服务端决定版本前置条件，本包不会为了拿版本先读一次再写。模型选择在没有模型快照时会先读一次目录用于可用性判断，这是读取而不是写入前置条件。
8. **无变更通知**：本版不消费 `runtime.watch` 或管理面通知；设置、模型、存储、运行组件由调用方显式刷新。模态实时事件由 D12 投递，任务状态不订阅事件，由调用方读 `ReadTask`/`ListTasks` 收敛。
9. **写入结果不轮询**：写登记进入未知或需人工确认后，本包不核对、不重发；未知结果只会按版本变化报冲突，不把读到的新文档或新选择当成本次写入成功。由调用方按 D3 的结果语义处理。
10. **不持久化**：来源、项目、文档、模型、存储、运行组件、模态能力、任务缓存、实时连接与订阅、幂等键都只在内存；重启后从零开始，不补发，也不恢复实时连接。
11. **错误文本不解析**：本包不根据错误文本判断冲突或能力缺失；冲突靠版本比较，能力缺失靠 `hello.features`、会话方法表与稳定 `appError`。
12. **模态依赖 D12 路由**：本包只定义会话模态接口与状态；方法表读取、任务读取、实时发送与通知路由由 D12 实现。真实服务端未验证。
13. **存储没有清理入口**：本包刻意不提供清理、删除或容量估算方法；调用方不应在本包之外自行删除服务端存储文件。
14. **管理写入串行**：同一 `Manager` 一次只处理一条设置写入、选择写入、组件安装或模态开关；一条慢写入会挡住其后的管理写入。这是为避免回执乱序写回缓存，第一版不做并发多写。
15. **安装后的状态重读失败**：组件安装已登记成功但重读清单失败时返回错误，调用方应先重读状态再决定是否再次安装，避免重复转达。
16. **选择写入不带项目目录**：本版的模型目录与选择读取都不带 `cwd`，因此选择写入作用于服务端默认作用域；按项目分别选择需要 D1 冻结 `provider.selection`/`provider.select` 的 `cwd` 形状后再补。
17. **选择版本可能缺失**：服务端没有给出选择版本时，本版不带前置条件写入；这时无法在客户端发现并发覆盖，只能由服务端策略保证。若服务端提供版本，调用方必须把 `SelectionView.Revision` 原样带回。
18. **任务提交与取消需要状态方法**：提交前要求 `modality.task.submit` 与 `modality.task.status` 同时存在，取消前要求 `modality.task.cancel` 与 `modality.task.status`；服务端只给其中一个方法时按能力缺失拒绝，不留下读不回状态的任务。
19. **任务提交/实时打开回执用 `InputID` 传资源标识**：提交与打开的意图本身没有资源标识，集成层的写发送方必须把服务端任务/连接标识放进 `write.Reply.InputID`；取消与关闭的意图已带标识，回执为空时沿用请求值。缺标识时返回 `ErrMalformedWriteResult`。
20. **任务等待与结果落盘不在本包**：本版只提供状态/清单读取与结果分页，不实现等待、轮询、base64 解码或媒体落盘；调用方自行按 `EOF` 收敛。
21. **实时状态只在内存**：本进程重启后不记得已打开的连接，`CloseRealtime` 会返回 `ErrRealtimeNotOpen`；这时应由服务端或用户侧清理残留连接。
22. **实时事件原样透传**：事件 JSON 只校验合法性，不解析音频或控制语义；播放与编解码由调用方负责。
23. **订阅回调同步**：`WatchRealtime` 的回调在投递路径上同步执行；慢回调会拖慢通知路由，回调里也不能调用本包的管理写入。
24. **真实模型夹具不入库**：夹具含真实账号凭据，只存在 `/tmp`；仓库只有脚本。换机器或令牌过期后需要人工重新准备夹具或登录。
25. **真实模型验证依赖本机账号与网络**：脚本能证明当前账号/网络下真实工具调用可用，不证明其他账号、其他模型或离线环境；模型输出与工具选择可能变化，脚本以“工具调用 + 期望工具结果”为判据。
26. **真实模型验证用的是 daemon CLI**：`scripts/verify-real-model.sh` 走 `dimcode-server exec` 的完整会话链路，不经过本包 Go API；本包 Go API 对真实 App Server 的验证仍在 §6.4 blocked。
27. **模态方法不是会话方法**：本机真实服务端只在 `hello.features` 声明 `modality.*`，`session.methods` 不含；门禁必须以 hello 能力为准，否则会误报能力缺失。
28. **实时在本机不可用**：`realtime.voice/transcribe/avatar` 均因 adapter 无可用实时模型而不可用；打开→发送→关闭没有真实证据，标 blocked，不用 mock 冒充。
29. **OCR 依赖已安装二进制路径**：macOS Vision 权限按可执行文件路径生效；夹具内复制的二进制跑 OCR 会失败。复现时必须用 `--server ~/.dimcode-alpha/rust/dimcode-server` 或等价安装路径。
30. **模态结果文本未打印**：证据只保留文件名字节数与分页数；OCR 文本属于任务结果内容，未写入文档或仓库。
31. **新建 provider 标识必须是 UUID**：真实服务端拒绝非规范小写 UUID；本包在发请求前拒绝其他形状。已有 provider 的读取/更新不受限制。
32. **provider 详情可能在版本/动作上受限**：账户托管的 provider 可能没有 `update` 动作，`provider.update` 会被服务端拒绝；本包不按动作列表预先猜测，只按版本与错误分类。
33. **凭据只透传**：`CreateProvider`/`UpdateProvider` 可以带显式凭据（set/set_ref/clear），但本包不缓存、不回显、不写日志；`ProviderDetail` 只有 `CredentialConfigured`。真实证据只用匿名认证。
34. **探测会访问网络**：`ProbeProvider` 是显式动作，可能访问用户给的 baseUrl；本包不缓存结果、不写配置。
35. **模型偏好可能被服务端拒绝**：例如模型不支持推理偏好时 `set` 返回 `provider_invalid_config`；本包保留原因，不本地伪造偏好值。
36. **provider 删除/远程刷新/导入旧配置不在本版**：只做列表、新建、更新、探测、启用、模型配置与偏好。
37. **provider 管理真实证据由脚本直连取得**：本包 Go API 的真实 provider 管理仍依赖 D12 的 `ProviderAdmin` 与写发送方适配器，当前 blocked。
38. **provider 列表可能很大**：真实夹具返回 127 个内置 provider；本包整份缓存，不做分页，列表读取耗时与内存随 provider 数量增长。
39. **agent 没有服务端 revision**：真实 `session.agent.update` 没有版本字段；本包用 `ExpectedAgentID` 作为前置条件，并在写入后重读。它不是跨客户端强一致的 revision，但能阻止覆盖已变化的当前 agent。
40. **agent.list 不按 hello 预判**：真实 hello.features 不含 `agent.list`，但方法可用；本包直接调用，缺失时靠 `unknown_method` 归为能力缺失，因此这一项不是零请求门禁。
41. **agent 前置字段不能下发**：`expectedAgentId` 只在本包使用；真实服务端会拒绝未知字段，D12 必须只发 `sessionId`、`branchId`、`agentId`。
42. **协作策略是全局设置**：真实 revision 是 `native:user/collaboration:n`；写入影响所有协作子代理，本包不做项目作用域。
43. **agent 档案创建/删除不在本版**：本版只读取清单与会话采用；`agent.create` 仍由调用方/其他领域负责。
44. **全仓 go test ./... 受账号领域影响**：当前 `internal/account` 的 `TestLoginRejectsConcurrentAttempt` 失败，与本领域无关；`go test ./internal/settings` 与 `go vet ./...` 通过。

## 8. 下一步

1. D1 冻结设置、模型、存储、运行组件与模态能力的方法名、参数与回执字段，把测试适配器替换成真实实现。
2. D6 对齐模型标识口径（是否带 provider 限定），并确认过期状态下免账号模型的可用性口径；本包按对齐后的规则调整匹配。
3. 补按项目目录的模型选择：需要 D1 冻结 `provider.selection`/`provider.select` 的 `cwd` 形状，本包再把 `cwd` 加入选择读取与写入请求。
4. D12 实现 `ModalitySession`：用会话方法表做门禁、用 `Subscription.Call` 读任务/发实时帧；写发送方按 `KindModalityWrite` 路由；把实时事件与结束通知投递给本包。
5. 接入运行组件 `runtime.watch` 与设置变更通知：通知到达后重读一次，不做定时轮询。
6. 补真实服务端证据：设置写入与选择写入的版本冲突、项目作用域、存储读数、运行组件安装、模态能力、任务提交/查询/取消、实时打开/发送/关闭与事件。
7. 与 U1 对齐读数呈现：任务进度、结果分页、实时事件与不可用原因的展示口径；本包只提供数据与原因，不做文案。
8. 把真实模型验证接到本包 Go API：用真实 daemon 执行 `models.providers.list`、`provider.select`、`storage.report`、`runtime.list` 与 `modality.capabilities`，把 §6.5 的“模型可用”推进到“本包读取与选择写入可用”。
9. 为真实模型验证准备专用测试账号或可轮换凭据，避免依赖个人 `~/.dimcode-alpha`；当前只做了本机夹具。
10. 接入可用的实时适配器/模型后，补 `realtime.voice` 打开→发送→关闭的真实证据；当前只能记录能力原因。
11. 让 D12 的 `ModalitySession` 适配器把 `modality.*` 控制连接调用接进来，再用本包 Go API 复跑 §6.6 的任务与分页，闭合“本包 API 真实可用”。
12. 让 D12 的 `ProviderAdmin` 把 `provider.list`/`management`/`probe` 映射进来，写发送方按 op 映射 provider 写方法；再用本包 Go API 复跑 §6.7，闭合“本包 provider API 真实可用”。
13. 补 provider 删除、`provider.refresh` 远程模型刷新与 `provider.import_legacy`；这些不在本版但方法已存在。
14. 让 D12 的 `CollaborationReader`/`AgentAdmin` 接通协作策略与会话 agent，再用本包 Go API 复跑 §6.8；当前真实证据由脚本直连取得。
15. 评估 agent 档案是否需要服务端 revision；当前只能以当前 AgentID 作为前置条件，跨客户端强一致冲突无法表达。
