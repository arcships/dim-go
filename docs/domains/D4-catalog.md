# D4 会话目录（读取与过滤分页、新建、改名与自动标题、删除、归档隐藏、导出、分支目录与创建、会话分叉）

Status: 第一版实现完成（P1-2 候选结果包，待组织评审）
Owner: D4 目录领域
验证: 可执行命令与覆盖表见第 6 节；真实 daemon 证据见 6.3，blocked 见 6.5
依据: `docs/domains/R.md` 功能块 4、5、R3、R4、R5、附 E 第 4 项；`docs/domains/A.md` 的「目录页」「归档隐藏列表」「四条总规矩」；`docs/domains/D1-connection.md` 的控制连接与错误分类；`docs/domains/D2-projection.md` 的行与轮次视图；`docs/domains/D3-write.md` 的写登记意图接口
范围: 目录读取与过滤分页、新建、改名与自动标题、删除判定、本地归档隐藏列表、导出，分支目录的读取、合并、状态收敛与从指定轮次结束处创建分支，以及从某个切点分叉出新会话。不含界面、连接实现、协议 wire 定义、会话内容折叠、分支内提交与停止、审批语义、账号与管理面。

---

## 1. 结果包

**领域**：D4 会话目录

**输入**

| 输入 | 来源 | 本版使用方式 |
|---|---|---|
| 功能块 5「会话目录与生命周期」 | `docs/domains/R.md` | 目录读取、新建、改名与自动标题、删除、归档、导出逐条对应 |
| 功能块 4「分支会话」与 R3 | `docs/domains/R.md` | 分支目录读取与完整性标记、从轮结束处开分支、状态收敛逐条对应 |
| 附 E 第 4 项「会话分叉」 | `docs/domains/R.md` | 从某个切点分叉出新会话；与分支（留在原会话内）区分 |
| R4 归档语义 | `docs/domains/R.md` | 只维护本地隐藏列表，不写回服务端；跨客户端归档记为服务端缺口 |
| R5 导出内容范围 | `docs/domains/R.md` | 按游标读完全部历史后生成文件，不写当前已加载的部分 |
| 目录页、归档隐藏列表、四条总规矩 | `docs/domains/A.md` | 目录页只由目录读取与目录变更通知驱动；归档只由用户显式操作驱动；不做乐观更新 |
| 控制连接与错误分类 | `docs/domains/D1-connection.md` | 目录读取与写意图由集成层经控制连接发送；能力缺失按非可重试失败上报 |
| 行、轮次与分页、分支事实 | `docs/domains/D2-projection.md` | 导出按轮读取；分支目录与切点沿用投影已有的推导结果；本包只保存、合并与校验，不另建折叠 |
| 写登记意图接口 | `docs/domains/D3-write.md` | 新建、改名、删除、分支创建与结束都登记为 `KindSessionLifecycle` 写意图；一次发送、不自行重试、结果由写登记收敛 |
| 假服务端与 D1 客户端 | `internal/fakeappserver`、`internal/appserver/protocol` | 本地 Unix socket 上真实 NDJSON 的目录读取、新建、改名、删除证据 |

**产物**

| 路径 | 职责 |
|---|---|
| `internal/catalog/catalog.go` | 对外类型与接口：条目、过滤、页、读取方、目录、分页推进、标题合并、归档可见性 |
| `internal/catalog/mutate.go` | 新建、改名、删除：写意图的形状、幂等键、结果形状、拒绝与冲突错误、删除前的订阅释放 |
| `internal/catalog/archive.go` | 本地归档隐藏列表：版本化格式、原子写、损坏重建为空 |
| `internal/catalog/export.go` | 导出：按游标收集轮次、去重与冲突判定、格式版本、最后才写文件 |
| `internal/catalog/branch.go` | 分支目录：读取、完整性标记、状态收敛与固定头、切点校验、创建与结束 |
| `internal/catalog/fork.go` | 会话分叉：切点选择、写意图、成功并目录、单条读回与待补齐 |
| `internal/catalog/{catalog,archive,mutate,export,branch,fork}_test.go` | 注入读取方/发送方/历史来源/分支来源/单条读回的领域行为测试 |
| `internal/catalog/appserver_test.go` | 假服务端 + D1 客户端的目录读取、新建后可提交、改名、删除闭环 |
| `internal/catalog/branch_appserver_test.go` | 假服务端 + D1 客户端的分支创建与结束闭环 |
| `internal/catalog/export_projection_test.go` | 用会话投影的按轮来源跑一次多轮导出（端到端，接 D2） |
| `internal/catalog/daemon_evidence_test.go` | 真实 App Server 的目录生命周期证据（缺可执行文件时跳过） |
| `internal/catalog/daemon_export_evidence_test.go` | 真实 App Server 两轮对话后导出的证据（缺可执行文件时跳过） |
| `internal/catalog/fork_appserver_test.go` | 假服务端 + D1 客户端的会话分叉闭环 |
| `internal/catalog/daemon_branch_evidence_test.go` | 真实 App Server 分支创建、结束与固定头的证据（缺可执行文件时跳过） |
| `internal/catalog/daemon_fork_evidence_test.go` | 真实 App Server 会话分叉与单条读回的证据（缺可执行文件时跳过） |
| `docs/domains/D4-catalog.md` | 本文件 |

**验收**

- 命令：`go test ./...`、`go vet ./...`、`go test -race ./internal/catalog/`；本版全部通过（2026-11-11，macOS arm64）。
- 本领域测试：63 个顶层测试（另含 11 个子测试）全部通过；覆盖见 §6.2。
- 证据档位：
  - `fakeappserver`：本地 Unix socket 上真实 NDJSON，覆盖过滤参数透传、新建、提交到新建标识、改名、删除、分支创建与结束、会话分叉、能力缺失。
  - 注入替身：`Reader`、`write.Sender`、`HistorySource`、`SubscriptionReleaser`、`BranchSource`、`EntryReader` 为测试替身，覆盖分页、标题合并、删除被拒与结果未定、导出失败路径、分支切点与状态收敛、分叉读回与待补齐。
  - 真实 App Server：alpha.21 上目录读取、新建、改名与标题通知、非法标题拒绝、会话分叉、删除已跑通；alpha.18 上用 mock 模型跑通两轮对话后的导出、分支创建/结束与分叉（6.3）。旧版本（alpha.7 / alpha.18）的缺能力路径仍按 blocked 记录（6.5）。

**依赖**

| 依赖 | 需要对方冻结/提供 |
|---|---|
| D1 连接 | 控制连接上的目录读取、新建、改名、删除、分叉（alpha.21 实测 `session.rename` / `session.fork` 在控制连接可用）；把标题通知（`session.notice`）原样交给集成层 |
| D2 会话投影 | 按轮提供只读快照（轮次序号 + 该轮内容）与向前翻页游标，供导出的 `HistorySource` 实现；提供分支目录与带原因的切点查询（`Folder.CutPoint`：ready/not_ended/not_loaded/no_branch）；不要求本包折叠 |
| D3 写登记 | `Register`/`Wait` 与 `KindSessionLifecycle`；支持 create/rename/delete、branch.create/branch.end 与 fork；新建不要求 `SessionID`，其余要求；删除未知结果的核对由 `Inspector` 用目录读取实现 |
| U1 原生界面 | 只读 `Snapshot`/`Entry`/`BranchDirectory`，用 `Hide`/`Unhide` 改归档，用 `Create`/`Rename`/`Delete`/`CreateBranch`/`EndBranch`/`Fork` 提交意图；不直接改本地集合 |

**上下文需求**

- 需要 D1 冻结目录读取的参数映射（关键字、工作目录、页大小、游标）与回执字段，以及 `session.rename` 究竟在控制连接还是会话流上发送。本版通过注入的 `Reader` 与 `write.Sender` 隔离，不复制 wire 形状。
- 需要 D3 确认删除的核对读取如何表达“该会话已不在目录里”。本版把写登记的成功视为服务端确认，并在文档中写明集成责任。
- 需要 D2 明确导出用的轮次快照形状（本版只约定 `Turn{Index, Data}` 为合法 JSON，不解释内容）。
- 需要 D2 提供带原因的切点查询（`Folder.CutPoint`，已落地）。本包按 ready/not_ended/not_loaded/no_branch 分派，不拿轮起点或行序号凑切点。
- 需要用户确认：导出文件默认位置与命名（本版只接受调用方给出的路径）；新建/分叉后未命名行在界面上的占位（U1 决定）；分支创建后是否立即切到新分支（界面行为，本版只给目录）；分叉后是否自动打开新会话（界面行为）。

## 2. 边界

**做**

| 能力 | 说明 |
|---|---|
| 目录读取 | 页大小、游标、关键字与工作目录过滤；过滤参数原样交给服务端 |
| 本地集合 | 按服务端页顺序合并；页内与跨页重复标识去重；游标重复则停止推进 |
| 标题合并 | 目录页、自动标题通知、改名结果三处按修订号取大 |
| 新建 | 经写登记提交，返回服务端会话标识；本地只增加一行标识，不编造标题与时间 |
| 改名 | 经写登记提交；成功后本地标题与回执一致；回执修订号落后时报告冲突并可重读 |
| 删除 | 先释放订阅，再经写登记提交；服务端确认成功后才移除本地行；被拒保留并给原因 |
| 归档隐藏 | 本地隐藏列表：原子写、带版本、损坏重建为空；只影响列表可见性 |
| 导出 | 按游标读完全部轮次后生成带格式版本的文件；中途失败不产出文件 |
| 分支目录 | 读取并保存每个会话的目录；完整性标记；状态只前进（open/ending/ended）；已结束分支头部固定 |
| 分支创建 | 从父分支某一轮结束处经写登记创建；成功后只并入新条目，不重建目录 |
| 分支结束 | 经写登记结束非主线分支；成功先收敛为 ending，等目录给出 ended |
| 结束分支的拦截 | 已结束分支拒绝再开分支；提交路径可经 `CheckBranchWritable` 拒绝已结束分支 |
| 会话分叉 | 从某个切点（某条输入或某轮结束处）经写登记分叉出新会话；新会话并入目录，源会话不变 |
| 分叉读回 | 回执只有标识时按标识读回一条目录条目补齐字段；读回失败保留待补齐标记 |

**不做**

| 不做 | 归属/原因 |
|---|---|
| 界面、列表渲染、归档入口的交互 | U1 |
| 连接、socket、NDJSON、方法名、参数与回执的 wire 定义 | D1 |
| 历史折叠、轮次切分、行身份、分支内容与分支历史 | D2 |
| 分支内的提交与停止 | D3（目录只给可写性判断） |
| 分叉出的新会话内容与打开 | D2（内容折叠）、U1（打开） |
| 分叉后自动切换/打开 | U1（本包只并目录） |
| 写意图的发送、重发、核对与在途表 | D3 |
| 审批、提问、权限档位 | D4 之外的交互裁决领域 |
| 账号、模型选择、管理面设置 | D6、D5 |
| 服务端目录的索引、缓存、本地过滤与本地排序 | 远程事实由服务端决定；本地只合并页 |
| 归档写回服务端、跨客户端同步 | R4 选 A；服务端没有对应事实 |
| 导出内容的渲染（Markdown、HTML）、图片与附件打包 | R5 选 A：只导出轮次快照；表达层由后续任务决定 |
| 任何持久化目录页或分支目录缓存 | A.md：目录页由读取驱动，坏了重取 |

本包只使用 Go 标准库与仓库内的写登记、会话投影公开类型，不引第三方依赖，不改 `go.mod`。

## 3. 包结构与对外接口

```text
internal/catalog/
├── catalog.go        条目、过滤、页、读取方、目录、分页、标题合并、归档可见性
├── mutate.go         新建/改名/删除的写意图、幂等键、结果形状、错误、订阅释放
├── archive.go        归档隐藏列表、版本化落盘、原子写、损坏重建
├── export.go         轮次收集、去重与冲突、格式版本、最后写文件
├── branch.go         分支目录、完整性、切点校验、创建与结束、状态收敛
├── fork.go           会话分叉、切点选择、并目录、单条读回、待补齐
├── catalog_test.go   分页、过滤透传、重复页、标题合并、归档可见性
├── archive_test.go   跨重启、损坏、未知版本、原子写与并发
├── mutate_test.go    新建、改名冲突、删除顺序与拒绝、归档记录惰性
├── export_test.go    轮次计数、失败不产出、游标重复、冲突与上限
├── branch_test.go    分支目录、切点、创建/结束、状态与固定头
├── fork_test.go      分叉成功/失败、切点缺失、未知结果、读回与待补齐
├── appserver_test.go 假服务端 + D1 客户端的目录闭环与能力缺失
├── branch_appserver_test.go 假服务端 + D1 客户端的分支创建与结束
├── fork_appserver_test.go 假服务端 + D1 客户端的分叉
├── export_projection_test.go 投影按轮来源的多轮导出
├── daemon_evidence_test.go 真实 App Server 目录证据（按环境变量跳过）
├── daemon_export_evidence_test.go 真实 App Server 导出证据（按环境变量跳过）
├── daemon_branch_evidence_test.go 真实 App Server 分支证据（按环境变量跳过）
└── daemon_fork_evidence_test.go 真实 App Server 分叉证据（按环境变量跳过）
```

对外接口：

```go
catalog, err := catalog.New(catalog.Config{
    Reader:   reader,    // 必填：读取服务端目录页
    Writes:   writes,    // 必填：D3 写登记
    Archive:  archive,   // 可选：本地归档隐藏列表
    Releaser: releaser,  // 可选：删除前释放会话订阅
    History:  history,   // 可选：导出的按轮历史来源
    Branches: branches,  // 可选：D2 的分支目录与切点来源
    Lookup:   lookup,    // 可选：按标识读回一条目录条目
})

// 目录读取
result, err := catalog.Refresh(ctx, catalog.Query{Search: "构建", Workspace: "<workspace>", Limit: 20})
result, err = catalog.LoadMore(ctx)             // 用服务端给的游标推进一页
snapshot := catalog.Snapshot()                  // 可见条目 + 隐藏计数 + HasMore + 过滤
entry, ok := catalog.Entry(sessionID)
hidden := catalog.HiddenEntries()

// 标题
applied := catalog.ApplyTitle(catalog.TitleUpdate{SessionID: id, Title: "自动标题", Source: catalog.TitleSourceAuto, Revision: 3})

// 目录变更（都经写登记）
entry, err = catalog.Create(ctx, catalog.CreateRequest{Cwd: "<workspace>"})
entry, err = catalog.Rename(ctx, catalog.RenameRequest{SessionID: id, Title: "新名字"})
err = catalog.Delete(ctx, catalog.DeleteRequest{SessionID: id})

// 归档（只改本地）
err = catalog.Hide(id)
err = catalog.Unhide(id)

// 导出
result, err = catalog.Export(ctx, catalog.ExportRequest{SessionID: id, Path: "<export-path>"})

// 分支目录与生命周期
directory, err := catalog.RefreshBranches(ctx, sessionID) // 从 D2 读目录，整次替换该会话
directory = catalog.Branches(sessionID)                   // Loaded / Complete / Branches
branch, err := catalog.CreateBranch(ctx, catalog.BranchCreateRequest{
    SessionID: sessionID, ParentBranchID: "main", TurnIndex: 2, Prompt: "换一条路",
})
branch, err = catalog.EndBranch(ctx, catalog.BranchEndRequest{SessionID: sessionID, BranchID: branch.ID})
branch, err = catalog.ApplyBranchUpdate(sessionID, catalog.Branch{ID: branch.ID, Status: catalog.BranchEnded, Head: 12})
err = catalog.CheckBranchWritable(sessionID, branchID)     // 已结束分支拒绝再提交

// 会话分叉（产生新会话，源会话不变）
forked, err := catalog.Fork(ctx, catalog.ForkRequest{
    SessionID: sessionID, At: 17, // 或 TurnIndex: 2
})
```

`Snapshot`/`Entry`/`BranchDirectory` 只含业务可观察字段：标识、标题与来源与修订号、工作目录、创建/更新时间、运行状态、是否可见、是否待补齐、分支状态与头部、目录是否完整。不暴露游标、页边界、幂等键与写意图标识。

## 4. 设计

### 4.1 目录读取与分页

| 概念 | 规则 |
|---|---|
| 过滤 | `Query{Search, Workspace, Limit}` 原样交给 `Reader.ReadCatalog`；本包不解释、不在本地二次过滤 |
| 页 | `Page{Entries, NextCursor}`；`NextCursor` 为空表示没有更多 |
| 首屏 | `Refresh` 从第一页重取，并用这一页重建本地集合；过滤条件变化必须 `Refresh` |
| 推进 | `LoadMore` 只用服务端返回的游标；读取失败不推进游标，重试仍是同一页 |
| 去重 | 页内重复标识只留一条；跨页已存在的标识跳过；两者都计入 `LoadResult.Duplicates` |
| 顺序 | 与服务端页顺序一致；本地不排序 |
| 游标环 | 服务端把游标指回已经请求过的位置时停止推进并标 `Stopped`，不产生重复行 |
| 上限 | 一次加载最多 10000 页，超过即停止推进 |
| 坏页 | 条目缺少会话标识即整页拒绝（`ErrMalformedPage`），本地集合保持上一版 |
| 恢复 | 页面损坏或过滤变化时重取第一页重建；更早的页由 `LoadMore` 重新推进 |

`LoadResult{Added, Duplicates, HasMore, Stopped}` 是不进界面的读数，供调用方诊断“为什么没有更多”。

本地只有一个进程内的标识索引，用于去重与按标识查找；它不落盘、不跨重启，也不是服务端目录的缓存副本。重启后从第一页重新读取。

### 4.2 标题：目录页、自动标题与改名

三处标题事实走同一条合并规则：**修订号大的胜**。

1. 目录页里的标题：重取页面时，若本地条目的 `TitleRevision` 更高，保留本地标题，避免并发通知被更旧的页面回退。
2. 自动标题通知：集成层解码后调用 `ApplyTitle`；未知会话忽略，不凭空新增条目；修订号不大于当前值忽略。
3. 改名结果：回执 `{title, titleSource, revision}` 与本地比较——
   - 修订号更大：采用回执的标题、来源与修订号；
   - 修订号相等且内容相同：视为重放，不产生变化；
   - 修订号更小，或相等但内容不同：返回 `TitleConflictError`，本地标题保持不变，调用方重读目录。

标题清空（回执 `title: null`）按空标题合并；来源与标题同时为空。

### 4.3 新建

1. 校验工作目录非空，否则在登记前拒绝。
2. 幂等键：调用方给了就用；没给就生成唯一键——**两次新建是两次独立意图**，不会被去重成同一个会话。
3. 以 `KindSessionLifecycle` 登记写意图，参数为 `{op: "catalog.create", cwd}`；新建不需要目标会话，发送与重试都由写登记负责。
4. 成功后从 `Reply.Result` 读 `{sessionId}`；缺失或为空即 `ErrMalformedResult`，不产生本地行。
5. 本地只加入 `Entry{ID, Incomplete:true}`：标题、工作目录、时间、运行状态都等服务端目录页补齐。这样“新建后可提交到该标识”成立，同时不出现乐观更新；下一次目录刷新会清掉待补齐标记。
6. 被拒、未发生或需人工确认：返回 `RejectedError`，目录不变。

### 4.4 改名

- 只对本地目录里已有的会话生效；未知会话在登记前拒绝。
- 以 `KindSessionLifecycle` 登记写意图，参数为 `{op: "catalog.rename", title}`，目标会话由写登记的 `SessionID` 承载（写登记按 op 要求改名必须有目标会话）。
- 幂等键默认按 `sha256(sessionId + "\x00" + title)` 派生，不把标题原文写进键；完全相同的改名复用同一意图，不重复发送。
- 结果按 4.2 的规则合并；冲突时返回 `TitleConflictError`，本地行保留。
- 本包不校验标题长度、空白字符与换行，这些是服务端规则；空标题表示清除。

### 4.5 删除

固定顺序：

```text
检查会话在本地目录里
  → Releaser.ReleaseSession（释放订阅；失败即中止，不登记删除）
  → 写登记登记 KindSessionLifecycle + {op: "catalog.delete"}（目标会话必填）
  → 等待终态
     成功 → 从本地集合移除该行
     被拒 / 未发生 / 需人工确认 / 未结项 → 保留该行并带回原因
```

- 写登记的“成功”指服务端明确回执或核对命中；删除的核对（结果未知时确认目录里已经没有该标识）由集成层的 `Inspector` 实现。本包不自行发读请求。
- 删除只移除目录行；归档隐藏列表里关于它的记录保留（A.md：按“条目已不存在”清理，不阻塞删除），该记录对不存在的条目不产生可见结果。
- 删除不影响其他行，也不回滚已加载的页。

### 4.6 归档隐藏列表

- 本地状态，文件路径由调用方给（例如应用数据目录下的 `session-archive.json`）。
- 格式带版本：`{"version":1,"hidden":[{"sessionId":"...","hiddenAt":"..."}]}`。
- 只有 `Hide`/`Unhide` 会改它；任何读取、刷新、删除都不会改归档。
- 写入是原子替换：同目录临时文件 → `0600` → 写 → 同步 → 替换；任一步失败都不动原文件。
- 损坏（非法 JSON）或版本不认识：状态重建为空，`Rebuilt()` 为真，磁盘文件在下一次用户显式操作时才覆盖。
- 隐藏只影响 `Snapshot`/`HiddenEntries` 的归类，不改变 `Entry` 事实，也不写回服务端。
- 不存在的会话标识可以隐藏（它只是不可见），不会阻塞删除，也不会凭空产生行。

### 4.7 导出

1. 校验会话标识与目标路径；没有历史来源即 `ErrExportUnavailable`。
2. 来源是会话投影（D2）的按轮历史：类型用别名直接对齐（`HistoryRequest`、`HistoryPage`、`Turn` 指向 `projection` 的公开类型），`*projection.TurnHistory` 直接满足 `HistorySource`，没有转换函数，也没有第二套折叠。构造方式为 `projection.NewTurnHistory(sessionID, reader)`，`reader` 是 D1 的会话订阅（`History(ctx, params)`）。
3. 从最近一页开始（`Before=0`），按 `HistorySource.ReadHistory` 的游标逐页向前读，`Limit` 控制每页轮数，直到 `Done` 或游标为 0。
4. 每页轮次做校验与去重：
   - 序号必须为正，内容必须是合法 JSON；
   - 同一序号出现两次且内容相同只算一次；内容不同即 `ErrConflictingTurn`；
   - 游标回到已请求过的位置即 `ErrRepeatedCursor`；页数超过上限即 `ErrExportLimit`。
5. 全部读完后按轮次序号升序排列，生成文件：

```json
{
  "format": "dimagent.session-export",
  "formatVersion": 1,
  "exportedAt": "2026-11-10T10:00:00Z",
  "sessionId": "s-1",
  "branchId": "main",
  "title": "会话标题",
  "turnCount": 3,
  "turns": [{"index": 1, "data": {...}}]
}
```

6. 文件同样走原子写（`0600`）。读页失败、轮次冲突、游标重复、页数超限都在写文件之前返回：**目标路径不存在时仍然不存在，已存在的旧文件保持原样**。
7. `turnCount` 恒等于 `turns` 的长度；二者都等于去重后实际读到的轮次数。

### 4.8 分支目录、创建与结束

分工：分支内容的折叠、行身份与分支历史归 D2；目录领域只维护“有哪些分支、什么状态、头部在哪、切点在哪”，以及创建/结束的写意图与结果合并。

**读取**

- 来源是注入的 `BranchSource`：`ReadBranches` 返回目录与 `complete` 标记，`TurnCut` 返回某一轮结束处的父分支序号。
- `complete=false` 表示服务端没有提供目录、内容按投影事实推导：目录照样可用，但标记为不完整，不冒充完整目录。
- `RefreshBranches` 只替换该会话的目录；读取失败保留原目录并把错误带回。

**合并**

- 目录按分支标识索引；`ApplyBranchUpdate` 只改一条，不重建整个目录。
- 状态只前进 `open → ending → ended`；已结束分支的 `Head` 固定，迟到的 `open` 或更大的 `Head` 都不生效。
- 创建成功只把新条目追加到目录；已有条目（含已结束分支的固定头）保持原样，也不为了合并再读一次目录。

**切点**

- 以轮结束为界：请求给父分支与轮次序号，切点由 `BranchSource.TurnCut` 返回的 `CutPoint` 给出。状态直接使用 D2 的公开类型（`ready` / `not_ended` / `not_loaded` / `no_branch`）。
- `ready`：切点可用；序号非正按 `not_ended` 处理，不拿轮起点或行序号凑。
- `not_ended`：返回 `ErrCutNotEnded`，文案为“第 N 轮还没结束，等这一轮结束后再开分支”，不引导翻页。
- `not_loaded`：返回 `ErrCutNotLoaded`，文案为“不在已加载范围，请先向前翻页到该轮”。
- `no_branch`：返回现有 `ErrUnknownBranch`。
- 未知状态按来源契约错误 `ErrBranchDirectoryMalformed` 拒绝，不发请求。
- 父分支已结束返回 `ErrBranchEnded`；父分支不在目录里返回 `ErrUnknownBranch`（在切点查询之前）。
- 服务端回执里的 `branchPoint` 是提交后的切点，以它为准写回目录。

**创建与结束**

- 创建：`KindSessionLifecycle` + `op=branch.create`，要求 `SessionID`，参数 `{op, parentBranchId, at, prompt}`；只发一次，结果由写登记收敛。幂等键按会话、父分支、切点轮次与提示词派生，不把提示词原文写进键。
- 结束：`KindSessionLifecycle` + `op=branch.end`，要求 `SessionID` 与分支标识；主线拒绝结束。
- 创建成功后返回 `Branch`（open）；失败或未结项保留目录并返回原因（`RejectedError` / `PendingError`）。
- 结束成功先把本地状态收敛为 `ending`；之后目录读取或事实更新给出 `ended` 与固定头；已结束分支再次结束不发请求。
- 已结束分支拒绝再开分支；提交路径可调用 `CheckBranchWritable` 咨询，已结束返回 `ErrBranchEnded`，目录还没读取时不拦截。

### 4.9 会话分叉

**与分支的区别**

- 分支：留在原会话内，切点是父分支某一轮的结束序号，结果进入原会话的分支目录；源会话内容不变。
- 分叉：复制出一个新会话；切点是“要继承的最后一条主线序号”（`session.fork` 的 `at`）；新会话作为独立目录行进入目录，源会话不变。

**切点**

- 显式 `At`：调用方从某条用户输入（会话投影里该行的序号）或已知边界给出。服务端要求该位置是静止点（本轮已结束、没有未消费输入与未结束后台任务），否则拒绝。
- `TurnIndex`：用 D2 的 `CutPoint` 取该轮结束序号；四种原因与分支切点共用（ready / not_ended / not_loaded / no_branch），错误按分叉语义描述。
- 两个都给或都不给，在登记写意图之前拒绝（`ErrInvalidRequest`）。
- 切点不可用时不发请求，不猜。

**写入**

- `KindSessionLifecycle` + `op=fork`，要求 `SessionID`（源会话），参数 `{op, at}`；只发一次，失败与未结项都保留原目录并带回原因。
- 未知结果：写登记核对后仍无法确认时进入需人工确认（`RejectedError`），不把未确认的新会话加进目录。
- 写登记确认成功但回执没有新会话标识（例如核对命中）：返回 `ErrForkResultUnknown`，提示重读目录，不猜标识。

**并目录与单条读回**

- 回执 `{sessionId, sequence}`（标题字段可选）成功后只追加这一条，不重建整个目录，也不重读目录页。
- 配置了 `Lookup` 时，按新会话标识读回一条目录条目：成功则用读回条目补齐标题、来源、工作目录并清掉 `Incomplete`；读回失败或未命中只保留标识行并标记 `Incomplete=true`，不阻塞分叉结果。
- 下一次目录刷新会把待补齐行换成服务端事实（`Incomplete=false`）。

## 5. 与 A/R 的对应

| 需求或架构条目 | 本领域实现 | 不在本领域 |
|---|---|---|
| R 块 5 目录过滤与分页 | `Query` 透传、服务端游标、页合并与去重 | 服务端过滤实现 |
| R 块 5 新建以服务端标识为准 | 写意图 → 回执 `sessionId` → 本地只加标识 | 打开会话（D2） |
| R 块 5 名称与标题只采用服务端值 | `ApplyTitle`/改名结果按修订号合并，页面不回退 | 标题生成规则（服务端） |
| R 块 5 删除只认服务端成功 | 先释放订阅、再登记意图、成功才移除 | 删除的服务端语义与核对读取（D1/D3） |
| R 块 5 / R4 归档 | 本地隐藏列表，版本化、原子写、损坏重建 | 跨客户端归档（服务端缺口） |
| R 块 5 / R5 导出 | 用 D2 的按轮来源读完所有轮次后写文件，失败不产出 | 渲染与打包 |
| R 块 4 分支目录读取与完整性 | `BranchDirectory` + `BranchSource`：服务端目录优先，缺失时沿用投影推导并标记不完整 | 分支事实的推导（D2） |
| R 块 4 / R3 开分支切点 | 以轮结束为界；`TurnCut` 不在已加载范围即 `ErrCutNotLoaded` | 轮次结束边界的推导（D2） |
| R 块 4 分支状态收敛 | `EndBranch` + 目录读取：open → ending → ended，已结束头部固定 | 分支内提交与停止（D3） |
| R 块 4 已结束分支不再接收 | `CreateBranch` 拒绝已结束父分支；`CheckBranchWritable` 拒绝提交 | 服务端侧的执行 |
| R 附 E 会话分叉 | `Fork`：显式 At 或按轮结束切点；新会话并入目录，源会话不变 | 新会话内容与打开（D2/U1） |
| R 附 E 分叉未知与失败 | 未确认不加行；失败保留目录；回执缺失报 `ErrForkResultUnknown` | 写登记核对实现（D3） |
| A 目录页 | 本地集合只由目录页推进；重取第一页恢复；标题按修订号合并 | — |
| A 归档隐藏列表 | 只有用户显式 `Hide`/`Unhide` 才变；损坏重建为空 | — |
| A 四条总规矩 | 只提交意图不写事实；删除成功才移除；不做乐观更新；目录状态只有一个驱动者 | — |
| A 检查项“没有第二份事实” | 归档只留本地；目录不写服务端；新建行不带未经确认的字段 | — |
| A 检查项“缺能力就报缺失” | 目录读取错误原样返回；改名被拒时保留行并给原因；无归档能力时 `ErrArchiveUnavailable` | — |

## 6. 验收

### 6.1 命令与结果

```bash
go test ./...
go vet ./...
go test -race ./internal/catalog/
```

本版结果（2026-11-11，macOS arm64）：

- `go test ./...`：全部通过（含 `internal/catalog`）。
- `go vet ./...`：无输出。
- `go test -race ./internal/catalog/`：通过。
- catalog 用例 63 个顶层测试（另含 11 个子测试）全绿；四个真实 daemon 用例在未设 `DIM_APP_SERVER_BIN` 时跳过。

### 6.2 覆盖

| 验收要求 | 测试 |
|---|---|
| 分页无重复无遗漏、顺序与服务端页一致 | `TestRefreshAndLoadMoreCoverAllPagesWithoutDuplicates` |
| 页内与跨页重复条目不产生重复行 | `TestRefreshAndLoadMoreCoverAllPagesWithoutDuplicates` |
| 重复页（游标不前进）停止推进且不重复 | `TestRepeatedPageStopsWithoutDuplicates` |
| 读取失败不推进游标、可重试同一页 | `TestReadFailureDoesNotAdvanceCursor` |
| 过滤参数透传（关键字、工作目录、页大小） | `TestQueryFiltersArePassedThrough`、`TestDefaultPageSize`、`TestAppServerCatalogLifecycle` |
| 坏页被拒绝且不破坏已有集合 | `TestMalformedPageIsRejected` |
| 标题按修订号合并；重取页面不回退；未知会话不新增行 | `TestTitleMergesByRevision`、`TestRenameAppliesCommittedTitleAndReportsConflict` |
| 真实改名后通知与本地一致（同修订幂等、更低修订被忽略） | `TestRealDaemonCatalogLifecycle`（6.3） |
| 新建后用返回标识提交落于该标识 | `TestCreateReturnsServerIDAndAcceptsSubmit`、`TestAppServerCatalogLifecycle` |
| 新建不产生未经服务端确认的字段 | `TestCreateReturnsServerIDAndAcceptsSubmit` |
| 新建被拒不产生本地行 | `TestCreateRejectedAddsNoRow` |
| 两次新建互相独立、幂等键不重复 | `TestTwoCreatesAreIndependent` |
| 缺工作目录时不登记意图 | `TestCreateWithoutCwdIsRejectedBeforeSending` |
| 改名后一致；重复改名不再发送 | `TestRenameAppliesCommittedTitleAndReportsConflict`、`TestAppServerCatalogLifecycle`、`TestRealDaemonCatalogLifecycle` |
| 非法标题被服务端拒绝、本地行保持不变 | `TestRealDaemonCatalogLifecycle`（6.3） |
| 改名回执修订号落后时提示冲突、本地不变、可重读 | `TestRenameAppliesCommittedTitleAndReportsConflict` |
| 未知会话的改名与删除在发请求前被拒绝 | `TestRenameAndDeleteRequireKnownSession` |
| 删除顺序：先释放订阅，再发删除意图 | `TestDeleteReleasesThenRemovesOnlyOnSuccess`、`TestDeleteRejectedKeepsRowWithReason` |
| 删除成功才移除本地行 | `TestDeleteReleasesThenRemovesOnlyOnSuccess`、`TestAppServerCatalogLifecycle`、`TestRealDaemonCatalogLifecycle` |
| 删除被拒保留行并给原因 | `TestDeleteRejectedKeepsRowWithReason` |
| 删除结果需人工确认时保留行并给出原因 | `TestDeleteWithUnknownResultKeepsRow` |
| 删除未结项（等待被取消）时保留行并给出原因 | `TestDeletePendingKeepsRowWithReason` |
| 订阅释放失败时不发删除意图 | `TestDeleteReleaseFailureSendsNothing` |
| 删除不清理归档记录，记录对不存在的条目惰性无效 | `TestDeleteKeepsArchiveRecordInert` |
| 归档跨重启保持 | `TestArchivePersistsAcrossRestart` |
| 归档损坏 / 未知版本重建为空 | `TestArchiveCorruptRebuildsEmpty`、`TestArchiveUnknownVersionRebuildsEmpty` |
| 归档原子写、并发不出半文件、`0600` | `TestArchiveWriteIsAtomic`、`TestArchivePersistsAcrossRestart` |
| 归档只影响目录可见性 | `TestArchiveHidesEntriesAndKeepsServerFacts` |
| 无归档能力时报缺失 | `TestArchiveUnavailable`、`TestArchiveRejectsEmptySessionID` |
| 导出轮次与历史总数一致、文件带格式版本 | `TestExportReadsAllTurnsAndWritesVersionedFile` |
| 导出中途失败不产出文件、旧文件不变 | `TestExportPageFailureLeavesNoFile` |
| 导出游标重复/页数超限/坏轮次/内容冲突都拒绝且不产出 | `TestExportRejectsRepeatedCursor`、`TestExportStopsAtPageLimit`、`TestExportRejectsMalformedTurns`、`TestExportDeduplicatesTurnsAndRejectsConflicts` |
| 导出缺来源或字段时明确拒绝 | `TestExportRequiresConfigurationAndFields`、`TestExportPathMustExist` |
| 导出标题优先取调用方值 | `TestExportTitleCanBeOverridden` |
| 分支创建成功并入目录、不重建已有条目 | `TestCreateBranchMergesIntoDirectoryWithoutRebuild`、`TestAppServerBranchLifecycle` |
| 分支创建失败保留原目录并给原因 | `TestCreateBranchFailureKeepsDirectory` |
| 切点不在已加载范围时拒绝、提示翻页且不发请求 | `TestCreateBranchRejectsUnloadedCut` |
| 切点轮次未结束时返回 `ErrCutNotEnded`，不提示翻页 | `TestCreateBranchRejectsNotEndedCut` |
| 切点查询说分支不存在时返回未知分支错误 | `TestCreateBranchRejectsUnknownBranchCut` |
| 已结束分支拒绝再开分支 | `TestCreateBranchRejectsEndedParent` |
| 分支状态只前进、已结束头部固定、已结束拒绝提交 | `TestBranchStatusAdvancesAndEndedHeadIsFixed` |
| 结束分支成功先收敛为 ending，已结束不再发请求 | `TestEndBranchConvergesStatus`、`TestAppServerBranchLifecycle` |
| 结束分支失败保留目录并给原因 | `TestEndBranchFailureKeepsDirectory` |
| 服务端不提供目录时标记不完整；读取失败保留原目录 | `TestBranchDirectoryDerivedIsMarkedIncomplete` |
| 坏分支目录被拒绝且不写入 | `TestBranchDirectoryRejectsMalformed` |
| 分支请求缺字段时在登记前拒绝 | `TestBranchRequestsRequireFields` |
| 真实分支创建、结束与固定头 | `TestRealDaemonBranchLifecycle`（6.3） |
| 分叉成功只并入新会话、单条读回补齐、已有条目不变 | `TestForkAddsNewSessionAndReadsBackOneRow`、`TestAppServerForkAddsSession` |
| 分叉读回失败/未命中/未配置时保留标识行并标记待补齐 | `TestForkReadBackFailureKeepsIncompleteRow` |
| 分叉失败保留原目录并给原因 | `TestForkFailureKeepsCatalog` |
| 分叉切点缺失或同时给两个时在登记前拒绝 | `TestForkRejectsMissingOrAmbiguousCut` |
| 分叉按轮取切点沿用四种原因 | `TestForkTurnUsesCutReasons` |
| 分叉未知结果只发一次且不加未确认行 | `TestForkUnknownResultKeepsCatalog` |
| 分叉成功但无回执时报需重读目录 | `TestForkSuccessWithoutResultKeepsCatalog` |
| 真实分叉并入目录与读回 | `TestRealDaemonFork`（6.3） |
| 多轮导出接 D2 按轮来源（端到端、两页） | `TestExportThroughProjectionTurnHistory` |
| 真实两轮会话导出（轮次、页数、内容） | `TestRealDaemonExportTurns`（6.3） |
| 服务端缺目录能力时报错而不是空目录 | `TestAppServerMissingCatalogCapability`、`TestRealDaemonCatalogLifecycle` |
| 假服务端上的完整目录闭环 | `TestAppServerCatalogLifecycle` |

### 6.3 真实 daemon 证据

夹具纪律：数据目录、模型配置与导出文件只放执行者提供的临时目录；服务端二进制、模型配置来源与仓库外源码只读，产品代码不读取它们。命令示例中的路径一律用占位符。

**目录、改名与标题通知（alpha.21）**

```bash
# 离线模型配置复制到临时目录，来源目录不动
cp <offline-model-config-source> <offline-model-config>
DIM_APP_SERVER_BIN=<app-server-bin> \
DIM_APP_SERVER_ARGS=app-server \
DIM_APP_SERVER_MODEL_CONFIG=<offline-model-config> \
  go test -run TestRealDaemonCatalogLifecycle -v -count=1 ./internal/catalog/
```

观察到的输出：

```text
服务端 0.1.0-alpha.21（protocol 3，profile configured，owned=true）
新建会话 <session-id>
重读条目：title=""，titleRevision=0，workspace="<workspace>"
改名后重读：title="目录证据标题"，titleRevision=1，titleSource="manual"
标题通知：title="目录证据标题"，revision=1，source="manual"，本地一致
非法标题被拒：目录改名被拒绝（failed）：发送明确失败: rpc error -32602 (invalid_params): Invalid title or control character
回收自启服务端: stopped=true
PASS
```

这条证据验证了：

1. **改名成功**：提交新标题后，回执给出 `title`、`titleSource=manual`、`revision=1`；重读目录（`session.catalog`）后本地行与标题一致，不是只在本地改。
2. **标题通知**：`session.rename` 触发 `session.notice`（`source=SessionStore`、`kind=title`、`data={title,titleSource,titleRevision}`）；通知与回执同修订号同内容，合并后本地状态一致，更低修订号的通知不会覆盖。
3. **非法标题**：带换行的标题被服务端以 `-32602 invalid_params: Invalid title or control character` 拒绝；目录行保持改名后的状态与修订号，写登记把拒绝原样带回。
4. **回收**：`stopped=true`。

**会话分叉（alpha.21）**

```bash
DIM_APP_SERVER_BIN=<app-server-bin> \
DIM_APP_SERVER_ARGS=app-server \
DIM_APP_SERVER_MODEL_CONFIG=<offline-model-config> \
  go test -run TestRealDaemonFork -v -count=1 ./internal/catalog/
```

观察到的输出（离线 provider 回话较慢，这一轮约 107 秒；alpha.18 的 mock 模型跑同一条路径约 0.35 秒）：

```text
服务端 0.1.0-alpha.21（protocol 3，profile configured，owned=true）
投影切点：第 1 轮结束于序号 34（turn <turn-id>）
分叉会话：id=<session-id> title="" titleSource="" titleRevision=0 workspace="<workspace>" incomplete=false
回收自启服务端: stopped=true
PASS
```

这条证据验证了：切点来自 `projection.Folder.CutPoint`（第 1 轮结束于序号 34）；`session.fork` 回执只有 `{sessionId, sequence}`，目录按标识做一次 `session.summaries` 单条读回，补齐工作目录并清掉 `Incomplete`；源会话条目不变；重读目录同时看到源会话与分叉会话；最后两个会话都能删除。alpha.7 因缺 `session.catalog` 跳过（6.5）。

**旧版本兼容路径（alpha.18）**

导出与分支证据需要真实模型回话，仍用 alpha.18 的 `--mock-model`；alpha.21 的离线 provider 只用于目录与改名证据。

环境：服务端为仓库外二进制（由 `DIM_APP_SERVER_BIN` 指定）0.1.0-alpha.18；临时数据目录 + `--mock-model`。

```bash
DIM_APP_SERVER_BIN=<app-server-bin> \
  go test -run TestRealDaemonCatalogLifecycle -v -count=1 ./internal/catalog/
```

观察到的输出：

```text
服务端 0.1.0-alpha.18（protocol 3，profile mock，owned=true）
新建会话 <session-id>
重读条目：title=""，titleRevision=0，workspace="<workspace>"
改名 blocked：服务端未提供该方法（目录改名被拒绝（failed）：发送明确失败: rpc error -32601 (unknown_method): Unknown method）
回收自启服务端: stopped=true
PASS
```

这条证据验证了：

1. **空目录**：新建数据目录上第一次读取返回 0 条，不是错误也不是占位。
2. **新建**：`catalog.Create` 经写登记提交，拿到服务端标识。
3. **重读**：本地集合按服务端页补齐工作目录与标题字段；新建行在服务端确认前只有标识。
4. **删除**：服务端确认后本地行消失，重读目录也看不到该会话。
5. **能力缺失**：该版本没有 `session.rename`（`session.methods` 只列 agent/input/tool/turn/ui 等方法），本包把拒绝原样带回并保留本地行，不假装成功。
6. **回收**：`stopped=true`，临时目录清空。

**导出证据（alpha.18）**：两轮真实对话后，用会话投影的按轮来源导出。

```bash
DIM_APP_SERVER_BIN=<app-server-bin> \
  go test -run TestRealDaemonExportTurns -v -count=1 ./internal/catalog/
```

```text
服务端 0.1.0-alpha.18（protocol 3，profile mock，owned=true）
首屏: head=1 events=1
导出：2 轮，2 页，2586 字节
导出文件轮次与内容一致：2 轮
回收自启服务端: stopped=true
PASS
```

这条证据验证了：两轮真实提交（`session.submit` + 等 `turn/end`）之后，`projection.NewTurnHistory(sessionID, lease.Subscription())` 作为来源；每页一轮强制走了 2 页；文件 `turnCount=2` 等于真实轮次；两轮的输入文本都在各自的轮次行里；`projection.TurnHistory` 直接满足目录的 `HistorySource`，中间没有转换层或第二套折叠。

**分支证据（alpha.18）**：读取目录、从切点创建分支、结束后重读。

```bash
DIM_APP_SERVER_BIN=<app-server-bin> \
  go test -run TestRealDaemonBranchLifecycle -v -count=1 ./internal/catalog/
```

```text
服务端 0.1.0-alpha.18（protocol 3，profile mock，owned=true）
投影切点：第 1 轮结束于序号 17（turn <turn-id>）
初始目录: [{ID:main ParentID: CutSequence:0 Status:open Head:17}]
新分支: {ID:<branch-id> ParentID:main CutSequence:18 Status:open Head:0}
分支收敛：state=ended head=9
回收自启服务端: stopped=true
PASS
```

这条证据验证了：先跑一轮真实对话，`projection.Folder.CutPoint` 给出该轮结束事实的序号 17；`session.branches` 的真实目录被读入（主线 open、头部 17）；`session.branch_create` 经写登记只发一次，服务端把切点提交为 `branchPoint=18`，本地只追加新分支、主线条目不变；`session.branch_end` 后重读目录收敛为 `ended`，连续两次读取头部都是 9（已结束分支的固定头）。

旧版服务端（0.1.0-alpha.7，需 `app-server` 子命令）上同一用例的行为：

```bash
DIM_APP_SERVER_BIN=<app-server-bin> DIM_APP_SERVER_ARGS=app-server \
  go test -run TestRealDaemonCatalogLifecycle -v -count=1 ./internal/catalog/
```

```text
服务端 0.1.0-alpha.7（protocol 3，profile mock，owned=true）
回收自启服务端: stopped=true
服务端未提供 session.catalog：真实目录证据 blocked（rpc error -32601 (unknown_method): Unknown method）
SKIP
```

结论：该版本没有目录读取方法，本包按能力缺失上报，不退化成空目录；该版本也没有 `session.history.latestTurn`，导出用例按缺能力跳过；分支用例因 `session.branches` 缺失同样跳过（6.5）。

### 6.4 与 D1/D2/D3 的接口形状

| 提供方 | 形状 | 本包消费方式 |
|---|---|---|
| D1 控制连接 | 目录读取方法与回执 | 集成层实现 `Reader`：`Query` → `{limit, cursor, query, cwd}`，回执 → `Page{Entries, NextCursor}` |
| D1 / 通知层 | 标题通知：`session.notice`，`{sessionId, source:"SessionStore", kind:"title", data:{title, titleSource, titleRevision}}`（alpha.21 实测） | 集成层解码后调用 `ApplyTitle`，本包不定义通知形状 |
| D1 会话流 | 会话订阅与历史读取 | 集成层把订阅交给 `projection.NewTurnHistory`；本包的改名在 alpha.21 走控制连接，不依赖会话流 |
| D1 错误分类 | `failure.KindOf(err)` 交由 D3 使用 | 能力缺失被 D3 归为明确失败，本包看到 `StatusFailed` 与原因 |
| D2 按轮历史 | `projection.HistoryRequest` / `HistoryPage` / `HistoryTurn` | 本包用类型别名对齐（不写转换函数）；`projection.NewTurnHistory(sessionID, subscription)` 直接满足 `HistorySource` |
| D2 分页 | `session.history` 的 `latestTurn` / `before` 游标 | 由 `TurnHistory` 装载与切页；本包只按页推进、去重与统计，不生成轮次 |
| D2 分支目录 | `Folder.Branches()` 的推导结果 + 服务端目录是否存在 | 集成层实现 `BranchSource.ReadBranches`：目录项与 `complete` 标记；本包不另建推导 |
| D2 切点查询 | `projection.CutPoint` / `CutStatus`（本包用类型别名对齐） | 集成层把 `Folder.CutPoint(branchID, turnIndex)` 一行转发为 `BranchSource.TurnCut`，绑定会话后不匹配返回 `no_branch`；本包按 ready/not_ended/not_loaded/no_branch 分派；分叉按轮取切点复用同一查询 |
| D1 单条读回 | `session.summaries` 的条目（alpha.21 实测） | 集成层实现 `EntryReader.ReadEntry`：按标识找到一条并转成 `Entry`；用于新建/分叉后补齐标题、来源与工作目录 |
| D3 写登记 | `Register`、`Wait`、`IntentView`、`Reply.Result`、`KindSessionLifecycle` | 目录、分支与分叉变更都登记为 `KindSessionLifecycle`，参数带 `op`（catalog.create/rename/delete、branch.create/branch.end、fork）；新建不需要 `SessionID`，其余由写登记按 op 要求；结果形状为 `{sessionId}`、`{title,titleSource,revision}`、`{branch:{branchId,...}}` 与 `{sessionId,sequence}` |
| D3 核对 | `Inspector.LookupInput` | 删除结果未知时，集成层用目录读取确认该标识已不存在；写登记返回成功即视为服务端确认 |

本包不导入 D1 的业务包；为导出只导入 D2 的公开类型（别名对齐，D2 不反向导入本包），并导入 D3 的写登记。集成层负责拼装 `Reader`、`write.Sender`、`SubscriptionReleaser`、`BranchSource`，把 D1 的会话订阅交给 `projection.NewTurnHistory`。

### 6.5 blocked

| 项 | 状态 | 缺什么 |
|---|---|---|
| 旧版本上的改名（alpha.7 / alpha.18） | blocked | 这两个版本没有 `session.rename`，按能力缺失处理；alpha.21 的真实改名与标题通知已跑通（6.3） |
| 分叉（alpha.7） | blocked | 该版本没有 `session.catalog`，目录证据不可用；alpha.18（mock）与 alpha.21 的分叉已跑通（6.3） |
| 真实服务端自动标题通知 | blocked | 需要真实模型跑出自动标题并推送标题通知；当前只有 `--mock-model` |
| 真实服务端目录（alpha.7） | blocked | 该版本没有 `session.catalog`，证据记为能力缺失 |
| 真实导出（alpha.7） | blocked | 该版本没有 `session.history.latestTurn`；alpha.18 的两轮导出已跑通（6.3） |
| 真实分支目录/创建/结束（alpha.7） | blocked | 该版本没有 `session.branches`；alpha.18 已跑通（6.3） |
| 删除结果未知时的真实核对 | blocked | 需要注入 `Inspector` 的目录读回实现与真实断线/超时场景；本版只有明确回执路径的真实证据 |
| Windows 平台 | blocked | 没有 Windows 主机 |

## 7. 已知限制

1. **恢复口径**：`Refresh` 从第一页重取并用这一页重建集合，更早的页需要 `LoadMore` 重新推进。A.md 写的“重取当前页、本地集合不回滚”本版没有做成按页精确恢复；重取会丢掉尚未重新推进的旧页。
2. **新建行只有标识**：标题、工作目录、时间与运行状态都等服务端目录页；界面需要先显示未命名占位（由 U1 决定表现）。
3. **不本地过滤与排序**：过滤条件变化必须 `Refresh`；本地集合始终是服务端页顺序。
4. **归档记录不清理**：删除会话不移除归档记录；记录对不存在的条目惰性无效，本版没有提供批量清理入口。
5. **删除的核对未实现**：本包把写登记的成功视为服务端确认；删除在结果未知时依赖集成层用目录读取实现 `Inspector`，本版没有内置实现。
6. **改名可用性依赖服务端版本**：alpha.21 支持 `session.rename` 与标题通知；alpha.7 / alpha.18 没有该方法，只能报拒绝并保留本地行。
7. **导出来源只适合导出**：服务端 wire 没有轮次序号，投影的按轮来源第一次读取会把该分支历史读全，再按轮次起点排序切成稳定的 1..N；因此它只适合导出，不适合界面实时读取。超长会话的一次全量装载是本版已知的内存与耗时成本。
8. **导出内容由投影决定**：`Turn.Data` 的行结构与字段由 D2 的 `turnDocument` 决定；本包只保证轮次计数、去重、排序与文件格式版本。
9. **导出路径由调用方负责**：不创建目录、不做工作区越界校验；目标目录不存在即失败且不产出文件。
10. **归档损坏只重建内存状态**：打开时不立刻覆盖磁盘，下一次显式操作才写新文件；`Rebuilt()` 只供诊断。
11. **空标题等于清除**：需要集成层把空标题映射成服务端语义（null）；长度、空白与换行规则归服务端。
12. **单进程写入**：归档写入用进程内互斥与原子替换；多进程同时写同一文件不在本版范围（单实例应用）。
13. **错误不带用户文案**：目录读取错误原样返回；拒绝错误只带原因字符串，怎么提示由 U1 决定。
14. **标题修订号相等但内容不同被视为冲突**：这能防止两个客户端互相覆盖，但也会让服务端的不一致状态每次改名都报冲突，需要重读才能收敛。
15. **分支目录依赖 D2 来源**：本包不推导分支事实，只保存、合并与校验；目录的完整性与缺失项由 `BranchSource` 给出，服务端目录缺失时 `Complete=false`。
16. **切点只覆盖已结束轮**：D2 的 `CutPoint` 依据已加载的 `turn/end` 事实；未结束轮返回 `not_ended`，本包报 `ErrCutNotEnded`，不会拿轮起点或行序号凑切点。
17. **分支内容与提交不在本领域**：分支历史、行身份与分支内提交/停止归 D2/D3；本包只给目录、创建/结束与可写性判断，`CheckBranchWritable` 需要提交路径主动调用。
18. **结束状态分两步收敛**：`EndBranch` 成功只到 `ending`，`ended` 与固定头要等目录读取或事实更新；结束期间的服务端拒绝仍会保留原目录。
19. **分叉的标题依赖读回或刷新**：alpha 的 `session.fork` 回执只有 `{sessionId, sequence}`；本地先落标识行并标记 `Incomplete`，由单条读回或下一次目录刷新补齐标题与来源。
20. **分叉未知结果拿不到新标识**：写登记若只能确认“发生过”而没有回执，本包无法知道新会话标识，返回 `ErrForkResultUnknown` 让调用方重读目录，不猜。
21. **分叉切点由服务端校验静止点**：显式 `At` 指向的位置若还有未结束轮次、未消费输入或后台任务，服务端会拒绝；本包只负责把拒绝原样带回。

## 8. 下一步

1. D1/D3 冻结目录读取适配器与连接选择：目录读取、新建、删除、分叉走控制连接；改名按服务端方法表选择连接。
2. D3 提供删除的核对实现：结果未知时用目录读取确认“该标识已不存在”，把 `Inspector` 接到真实服务端上跑一次断线删除。
3. 导出已接 D2 的 `TurnHistory`；在超长真实会话上量一次首次全量装载的时延与内存，评估是否需要分片来源。
4. 分叉已接 `session.fork` 与单条读回；与 U1 约定新会话的打开方式，并在真实长会话上验证分叉后的目录与历史一致。
5. 切点已接 D2 的 `CutPoint`；用支持自动标题的服务端与真实模型补自动标题通知的证据（手动标题已由改名触发验证）。
6. 与 U1 约定新建/分叉未命名行的占位、归档入口与恢复方式；约定冲突提示的文案与“重读”按钮。
7. 复核 `Refresh` 的恢复口径：是否改成按已加载页数重读，避免过滤变化之外的场景丢页。
8. 评估归档记录的惰性清理（按整个目录读取结果回收不存在的标识），不阻塞删除、也不增加额外写盘。
