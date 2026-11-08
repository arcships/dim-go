# DimAgent MyGo 开发计划与 Subagent 编排

Status: active proposal
Last updated: 2026-11-08

## 目标与边界

本项目用 MyGo native UI 重实现 DimAgent Desktop 主界面；Rust `dim-app-server` 仍是 session、branch、timeline、approval、model、permission 的唯一事实源。Go 项目不读 App Server 私有 SQLite/JSONL/daemon.log，不建立本地 session/run/approval 真相，也不把 Electron IPC 搬进来。

主界面优先使用 MyGo 公开 `ui` API：`ui.List`、`ui.TextArea/TextBuffer`、`ui.Markdown`、accessibility、theme、SVG；`editor`、`richtext`、chart、video 和 plugins 按阶段启用。调查细节见 [mygo-native-components.md](mygo-native-components.md)。

## 关键技术前置

- MyGo 当前主分支 `go.mod` 要求 Go 1.27.3；本机 Go 1.24.3。第一任务不是盲目添加依赖，而是完成 Go toolchain/MyGo 版本决策：升级本机 Go，或选择兼容的 MyGo commit/tag，并记录证据。
- 不能复制 `github.com/egoist/mygo/internal/...`；只使用公开 API。
- 所有跨 agent 的领域合同以 Go package 和 golden fixtures 为准，不能靠口头约定。
- protocol v3 的 session 列表方法是 `session.catalog`（首选）或受限的 `session.summaries`，不是 `session.list`。
- `session.resync` 是 Desktop domain event，不是 wire RPC；触发后必须重新 attach 并重读窗口，不按 cursor 伪造重放。
- Projection 必须同时覆盖 durable `TimelineFold` 和 live `RawEventBridge` 的一致性；approval/question 与 model progress 是独立 transient projection，不塞进 TimelineFold。
- Go golden fixture 要注入时钟，避免 `GoalFoldState` 等逻辑使用 `Date.now()` 造成跨语言不确定性。

## Milestone 0：工具链和组件 Spike

**目标：**确认 MyGo 在当前 macOS 环境可构建、窗口可启动、核心组件能承载聊天。

任务：

1. 确认 Go 版本策略并锁定 MyGo commit/version。
2. 添加 MyGo 依赖，运行最小窗口。
3. 建立 `internal/ui` 的 AppFrame、Sidebar、Content 静态布局。
4. 用 `ui.List` 做 keyed dynamic rows + `FollowEnd` fixture。
5. 用 `ui.TextArea/TextBuffer` 做 Composer，验证中文/英文/emoji IME、复制、撤销。
6. 用 `ui.Markdown` 做 assistant、代码块、链接和 copy fixture。
7. 加入 headless/fake 测试和基本 accessibility labels。

退出条件：macOS 可启动；上述 fixture 有可复现实测结果；依赖和版本已锁定；未通过的能力明确记录为 blocker。

## Milestone 1：Native UI 静态壳

组件：

```text
AppFrame
├─ SessionSidebar
├─ ConversationView (ui.List)
│  └─ MessageRow (ui.Markdown / native cards)
├─ ComposerView (ui.TextArea)
└─ Status/Inspector
```

任务：design tokens、dark/light theme、字体 fallback、MessageRow、ToolCard/ApprovalCard 占位、窗口状态、loading/empty/error、keyboard navigation、accessibility。

退出条件：fixture 可截图；长列表和流式增长不跳动；UI 不依赖 WebView；所有能力均是可丢 view state。

## Milestone 2：协议合同、Transport 和 Supervisor

任务：

1. 从 `dim-electron` protocol v3 wire 合同提取最小 Go 类型，不复制 Electron IPC；列表首选 `session.catalog`，不虚构 `session.list`。
2. NDJSON framing、request correlation、notification dispatch、malformed frame/size/timeout/cancel。
3. control connection、每 session 独立 stream、attach/detach；attach 前先订阅通知。
4. Supervisor：发现/启动 Desktop 自己拥有的 daemon，hello 校验，退出时只停止自己启动者。
5. reconnect、re-attach、domain-level resync、instanceId 变化和 capability unavailable；resync 不作为 wire 方法。
6. fake App Server：断线、乱序、重复、协议错误、history 分页不变量、subscription_changed 和未知 mutation 结果。

退出条件：transport/supervisor/reconnect 单测；fake server 可确定性重放；没有任何私有存储读取。

## Milestone 3：Projection 与 Session 最小闭环

先建立两层 golden fixtures：

- 纯 projection：history/live raw events、sequence 幂等、ID/block 生命周期、branch、closed；
- 协议会话：catalog、attach 顺序、history 分页、approval replay、model progress、subscription_changed、resync。

其中 TimelineFold 与 RawEventBridge 必须对相同事实维持一致的 ID/block 边界；approval/question 和 model progress 保持独立 transient projection。

任务：

- TimelineFold；
- history/live 共用 reducer；
- session catalog/history；
- native ViewModel；
- session list/open/attach；
- render bounded history；
- submit；
- streaming text/thinking；
- tool call/result；
- stop、draft 保留、close、reconnect。

退出条件：

```text
启动 → session list → open → history
→ submit → streaming → tool 状态 → stop
→ reconnect/resync 后继续
```

## Milestone 4：核心 Agent 能力

按风险顺序：

1. title、goal、prompt suggestion、model progress、preferences；
2. branch directory、side chat、cut point、branch stream；
3. approval/question/permission（approve/deny/always allow、replay、顺序）；
4. background task/process inventory、completed preview、member cancel。

缺上游方法必须 visible-disabled/WIP；mutation unknown 先 read-back，不盲目重试。

## Milestone 5：管理面和 Desktop 服务

按 capability 单独迁移 settings、model/provider、MCP、plugins、skills、workflow、scheduler、runtime、account/usage、hooks，以及文件选择、附件、剪贴板、通知、外链、预览、诊断。外部网页/插件 UI 只能进入隔离 WebView，不进入主 native trusted tree。

## Milestone 6：发布、并行和退役

Electron 继续作为行为基线，MyGo 作为 preview；双方使用同一 App Server，但不双写、不做自动对账。验证 macOS arm64/x64、Windows x64、Linux x64 的字体、IME、GPU fallback、无障碍、daemon 生命周期、签名、更新、卸载、回滚。全部核心验收和回滚证据完成后才退役 Electron。

# Subagent 编排逻辑

## 角色

由主 agent 充当架构 owner、集成者和验收者；所有 subagent 使用 `deepseek-v4.1-flash`，每个任务必须有明确文件边界、验收命令和停止条件。

| 角色 | 责任 | 不允许 |
|---|---|---|
| Toolchain/UI agent | MyGo 版本、最小窗口、UI fixture | 不改协议/projection |
| Protocol agent | wire、transport、fake server、reconnect | 不接真实 UI、不读私有存储 |
| Projection agent | golden fixtures、TimelineFold、ViewModel | 不改 transport、不改 widget |
| Native UX agent | AppFrame、List、Markdown、Composer、a11y | 不定义 App Server 事实 |
| QA/reviewer agent | 测试、边界、性能、跨平台风险 | 不擅自扩大 scope |

## 分配规则

1. **单一 owner**：每个 package/文件在同一时间只能由一个 agent 修改。
2. **合同先行**：Protocol types、projection model、UI model 先由主 agent 审核，才能让下游并行。
3. **小批次**：每个 agent 一次只接一个可在 1–3 小时完成的任务；完成后报告 commit、测试和残余风险。
4. **依赖门**：Milestone 0 不通过，不让 agent 假装接入真实 MyGo；Milestone 2 合同不冻结，不让 UI 直接调用 wire。
5. **只读研究与写代码分开**：研究 agent 不改实现；实现 agent 不擅自解释上游协议。
6. **失败即停**：构建、Go 版本、上游协议或平台能力 blocker 必须报告，不用本地伪实现绕过。
7. **集成前审查**：主 agent 检查 diff、边界、依赖、测试和是否引入第二份事实后，再合并 agent 分支。

## 并行波次

### Wave 0：可并行研究

- Toolchain/UI：Go/MyGo 版本和最小窗口 Spike；
- Protocol research：从 dim-electron 提取 protocol v3 最小合同；
- Projection research：列出现有 TimelineFold 语义和 golden fixture 输入；
- QA：设计 headless、fixture、性能和 IME 验收矩阵。

### Wave 1：合同冻结后

- Native UX：实现静态 AppFrame、Sidebar、Conversation fixture、Composer；
- Protocol：实现 framing/client/fake server；
- Projection：实现纯 Go TimelineFold/ViewModel；
- QA：为三者补测试和 boundary checks。

### Wave 2：纵向闭环

只允许一个 integration agent 接通：

```text
protocol → projection → ViewModel → native UI
```

其他 agent 负责回归测试、错误注入、性能和可访问性，不再并行改同一条链。

### Wave 3：能力切片

每个 capability 独立分配 issue/branch/agent：branch、approval、question、model、tasks、settings。每项都要注明 upstream method、WIP fallback、测试和验收。

# 主 agent 持续监控机制

每次 subagent 报告后检查：

1. `git diff --stat` 和实际文件边界；
2. `go test ./...`、`go vet ./...` 及任务专属测试；
3. 是否引入 `internal/mygo` 复制、私有 IPC、socket 直连、local session storage；
4. 是否符合当前 milestone 和依赖门；
5. 是否需要拆分、返工或升级为 blocker；
6. 合并后更新本文件的状态和下一波任务。

固定状态格式：

```text
Task / owner / branch
Changed files
Verification (actual commands + results)
Contract impact
Known gaps/blockers
Next action
```

## 当前分配

- 主 agent：冻结计划、维护合同、集成和验收。
- Wave 0 Toolchain/UI agent（deepseek-v4.1-flash）：执行 Go 1.27.3 / MyGo 版本与最小窗口 Spike，进行中。
- Wave 0 Protocol research agent（deepseek-v4.1-flash）：已完成只读合同调查；主 agent 待审阅并转化为 Go types/fake server 任务。
- Wave 0 Projection research agent（deepseek-v4.1-flash）：已完成 TimelineFold 调查；主 agent 待审阅并转化为 fixtures 任务。
- Wave 0 QA agent（deepseek-v4.1-flash）：设计验证矩阵，进行中。

## Wave 0 研究结论（已收录）

- MyGo 组件：`ui.List` 的 keyed/dynamic rows 与 `FollowEnd` 是聊天首选；`ui.TextArea/TextBuffer` 是 Composer 首选；`ui.Markdown` 是 assistant 基础 renderer；`ui/editor` 延后用于代码/diff；richtext/chart/video/plugins 延后。
- 协议：不存在 `session.list`；`session.catalog` 是首选列表；`session.resync` 不是 wire method；attach 必须先订阅；history/live、approval replay、subscription_changed 和未知 mutation 结果都要有 fixtures。
- Projection：TimelineFold 是 durable transcript fold；RawEventBridge 是并行 live projection；approval/question/model progress 不应塞进 fold；Go golden 需要注入时钟。

在 Wave 0 全部完成并由主 agent 审核前，不开始大型功能开发。
