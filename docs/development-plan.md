# DimAgent MyGo 开发计划与 Subagent 编排

Status: active（Wave 1 已实现；Wave 2 底层组件已实现，纵向闭环和真实运行验收未开始）
Last updated: 2026-11-08

## 目标与边界

本项目用 MyGo native UI 重实现 DimAgent Desktop 主界面；Rust `dim-app-server` 仍是 session、branch、timeline、approval、model、permission 的唯一事实源。Go 项目不读 App Server 私有 SQLite/JSONL/daemon.log，不建立本地 session/run/approval 真相，也不把 Electron IPC 搬进来。

主界面优先使用 MyGo 公开 `ui` API：`ui.List`、`ui.TextArea/TextBuffer`、`ui.Markdown`、accessibility、theme、SVG；`editor`、`richtext`、chart、video 和 plugins 按阶段启用。调查细节见 [mygo-native-components.md](mygo-native-components.md)。

## 关键技术前置

- Toolchain 已定：`go.mod` 固定 Go 1.27.3 和 MyGo `v0.8.3`。MyGo 所有已发布 tag 都要求 Go 1.27.1 以上，没有能配 Go 1.24.3 的版本；本机 `/usr/local/go` 仍是 1.24.3，靠 `GOTOOLCHAIN=auto` 按 `go.mod` 自动下载 `go1.27.3`。证据见 [Milestone 0 进展](#milestone-0-进展toolchain-和最小窗口2026-11-08)。
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

### Milestone 0 进展：Toolchain 和最小窗口（2026-11-08）

Task / owner：Wave 0 Toolchain/UI agent，主 agent 已审查并合并。提交：`7f89890 feat(ui): add mygo native shell spike`。

Changed files：`go.mod`、`go.sum`、`cmd/dimagent/main.go`、`internal/ui/shell.go`、`internal/ui/shell_test.go`、`docs/development-plan.md`、`README.md`、`docs/mygo-native-components.md`。

**版本结论**

- MyGo 所有已发布 tag（`v0.1.0` 到 `v0.8.3`）的 `go.mod` 都要求 Go 1.27.1 以上；`v0.8.3` 和 `main`（`c8e911e`，2026-11-08）要求 `go 1.27.3`。因此"选一个能用 Go 1.24.3 构建的 MyGo 版本"不成立，唯一可行路径是让构建使用 Go 1.27.3。
- 本机 `/usr/local/go` 仍是 Go 1.24.3，`GOTOOLCHAIN=auto`（默认）按 `go.mod` 自动下载 toolchain 到模块缓存：`~/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.27.3.darwin-arm64`（243 MB）。系统 Go 安装未被修改。
- 锁定 `go 1.27.3` + `github.com/egoist/mygo v0.8.3`（tag，不跟 `main` commit）。不需要 `toolchain` 行：`go get` 会因为 `go` 指令已达到所需版本而删掉它。
- 依赖面：`go list -m all` 共 17 个模块，`go.mod` 的直接依赖只有 MyGo；`purego`、`go-text/typesetting`、`x/image`、`x/net`、`x/text` 是间接依赖。`go-mp3`、`oggvorbis`、`vorbis` 只出现在模块图里，没有写进 `go.mod` 的 require，也没有参与当前构建。

**Verification（实际命令和结果）**

| 命令 | 结果 |
|---|---|
| `go version`（仓库内） | `go version go1.27.3 darwin/arm64`（自动切换后） |
| `GOTOOLCHAIN=local /usr/local/go/bin/go build ./...` | 失败：`go: go.mod requires go >= 1.27.3 (running go 1.24.3; GOTOOLCHAIN=local)` |
| `go mod tidy` | 成功，写出 `go.mod`/`go.sum` |
| `go mod verify` | `all modules verified` |
| `go vet ./...` | 通过，退出码 0 |
| `go test ./...` | `ok github.com/arcships/dimagent-mygo/internal/ui 0.393s`；`cmd/dimagent` 无测试文件 |
| `go build -o /tmp/dimagent-spike ./cmd/dimagent` | 成功，14 MB 单文件 |
| 运行 `/tmp/dimagent-spike` 3 秒 | 进程存活；`lsappinfo list` 显示 GUI app `dimagent-spike`（in front）；`SIGTERM` 后退出码 0，stderr 为空 |

**实现范围**

- `cmd/dimagent/main.go`：`mygo.App.WhenReady` + `mygo.NewWindow`，窗口 1080×720，内容为 `ui.View(state.View)`。
- `internal/ui/shell.go`：静态 AppFrame（`ui.Sidebar` 会话列 + content 列 + `ui.TextArea` composer），只持有可丢 view state，不放 session/run/approval 事实。
- `internal/ui/shell_test.go`：3 个 headless 用例（`ui.NewTester`）：静态文本渲染、sidebar 选择驱动 header、composer 输入写入 draft。
- 只使用公开 `ui` API：`Row`、`Column`、`Box`、`Sidebar`、`SidebarItem`、`Scroll`、`Text`、`Textf`、`TextArea`、`View`、`NewTester`。没有 import `github.com/egoist/mygo/internal/...`。

**Contract impact**：无。没有 socket/App Server 访问，没有本地 session 存储，没有 self-defined 协议类型。

**Known gaps/blockers**

- 真实窗口的视觉和交互没有人工验收，目前证据只有"进程启动 + 窗口注册"加 headless 断言。
- 中文/emoji IME、复制粘贴、撤销、字体 fallback、accessibility 朗读均未验证；IME 必须真机手测。
- Milestone 0 任务 4–6 已由 Wave 1 实现（`ui.List`/`FollowEnd`、`ui.Markdown`、`TextArea` composer），见 [Wave 1 进展](#wave-1-进展协议supervisorprojection-和静态壳2026-11-08)；`TextBuffer` 没有单独接入。
- `mygo.App.Run()` 必须在 main goroutine 调用，真实窗口不能写成 `go test`；目前只能 `go run ./cmd/dimagent` 手测。
- 离线环境或 CI 没有 `GOTOOLCHAIN=auto` 时会在 `go.mod` 版本检查处失败；需要预装 Go 1.27.3。
- 平台运行覆盖只有 macOS arm64；六个目标只做过交叉编译和 vet，未运行。

**Next action**：Wave 1 由主 agent 审查后进入纵向闭环；真实窗口、IME 和 accessibility 手测在纵向闭环前单独记录。

## Wave 1 进展：协议、supervisor、projection 和静态壳（2026-11-08）

Task / owner：Wave 1 Protocol、Projection、Native UX agents；由 merge reviewer 按 appserver/supervisor/projection/ui/docs 分 scope 提交。

Changed files：`internal/appserver/`（wire、framing、client、catalog、summaries、history、errors、catalog_refresher、fake tests、`testdata/wire/notifications.ndjson`）、`internal/supervisor/`（endpoint、dial、spawn、supervisor、wire、fake daemon/pipe tests）、`internal/projection/`（projection、transient、golden runner、12 个 `testdata/projection/*.json`）、`internal/ui/`（shell、tokens、connection、List/Markdown/composer、keyboard/status/accessibility/follow-end tests）。

**实现范围**

- Protocol：16 MiB NDJSON framing、JSON-RPC 2.0 信封校验、请求关联/超时/取消、hello 校验、catalog/summaries/history read helpers、capability fail-closed、`appError` 映射、已知通知类型解码。
- Supervisor：Unix socket/Windows pipe endpoint、5s dial + hello、detached spawn、`daemon.log` 只写、listen 等待、只关闭自己启动的 daemon、control/stream 连接。
- Projection：单一 reducer 覆盖 history/live、sequence 幂等、text/thinking/tool 生命周期、tool-result 乱序、branch 隔离和注入时钟；transient 只做 model progress/notice，不进入 durable timeline。
- UI：keyed `ui.List` + `FollowEnd`、`ui.Markdown`、`TextArea` composer、侧栏键盘导航和状态行 live region；全部是 fixture/view state。

**Verification（实际命令和结果）**

| 命令 | 结果 |
|---|---|
| `go build ./...` | 通过 |
| `go test -count=1 ./...` | 4 个包全部 `ok`（appserver、projection、supervisor、ui） |
| `go test -race -count=1 ./...` | 4 个包全部 `ok` |
| `go vet ./...`（darwin/arm64、darwin/amd64、linux/amd64、linux/arm64、windows/amd64、windows/arm64） | 六个目标全部通过 |
| `go build ./...`、`GOOS=windows GOARCH=amd64 go build ./...`、`GOOS=linux GOARCH=amd64 go build ./...` | 通过（交叉目标未运行） |
| `go test -run TestGoldenFixtures -v ./internal/projection` | 12 个 golden fixture 全部通过 |
| `gofmt -l .`、`git diff --check` | 无输出、通过 |

**Contract impact**：`protocol-contract.md` 的 wire/DTO/错误状态和 `projection-contract.md` 的核心 timeline 语义已按实现更新；没有新增本地事实存储，没有读取 App Server 私有文件。

**Known gaps/blockers**

- 纵向闭环未开始：`cmd/dimagent/main.go` 仍是手写 fixture，没有 supervisor → appserver → projection → UI 接线（Milestone 3）。
- reconnect/re-attach/domain resync 未实现；Milestone 2 任务 5 仍未完成。
- `session.create`、`session.keys`、`session.branches`、`session.input_lookup`、approval mutation 和 attachment 上传没有 Go 实现；对应入口保持 WIP。
- 没有 real-daemon 测试：本机未找到 `dim-app-server`（`target/{debug,release}`、`../dim/target`、runtime 和 `.tmp` 路径均不存在，`DIMCODE_APP_SERVER_BIN` 未设置），不宣称 real-daemon 通过。
- 真实窗口、中文/emoji IME、复制粘贴/撤销、VoiceOver 和字体 fallback 未验证；`internal/ui` 测试是 headless。
- 交叉编译只证明能编译，Windows named pipe 和 Linux GUI 运行时未验证。

**Next action**：主 agent 审查 Wave 1 diff；通过后只由一个 integration agent 接通 Wave 2 闭环。UI 手测和 real-daemon 验收在二进制和目标平台可用后单独记录。

## Wave 2 进展：Session stream、mutation helpers、input blocks 和 WIP 入口（2026-11-08）

Task / owner：Wave 2 App Server、Session、Viewmodel、Projection、Supervisor QA 和 Native UX agents。Projection、Supervisor QA、Native UX 与 App Server hardening 已分别提交 `6dd3be3`、`892d7b6`、`bf74da2`、`b0f853f`/`3ebfea3`；`internal/session` 为 `8c4e7e8`/`064c58a`，viewmodel 为 `36703e9`，supervisor reconnect 为 `f41f276`/`415908c`/`2eb89f8`，UI tool/run 为 `be341f5`，appserver 流所有权修复为 `7b0e728`；UI 接线仍待同步。

Changed files：`internal/appserver/`（`session_stream.go`、`mutations.go`、`client.go`、`wire.go`、`fake_test.go`、对应 tests）、`internal/session/`（`session.go`、`history.go`、`adapter.go`、`fake_test.go`、`session_test.go`）、`internal/viewmodel/`（`viewmodel.go`、`viewmodel_test.go`）、`internal/projection/`（`projection.go`、`projection_test.go`、新增 `testdata/projection/user-content-blocks.json` 和 `shell-input-raw-block.json`）、`internal/supervisor/`（`supervisor.go`、`dial.go`、`watch.go`、`reconnect_unix_test.go`、`watch_test.go`、`fake_daemon_unix_test.go` 等）、`internal/ui/`（`shell.go`、`wip.go`、`run.go`、`tool.go`、`tokens.go` 及对应 tests）、`README.md` 和 `docs/` 的四份合同/计划。

**实现范围**

- Session stream：`OpenSessionStream` 先订阅通知再发 `session.attach`；`attach.sequence` 缺失时用 `session.history {branchId:"main", after:0, limit:1}` 的 `through` 兜底，缺 feature 时返回 capability error；attach 期间的通知按到达顺序缓冲，返回后按顺序补放；按 `event.id` 去重，并拒绝同一 `(branchId, sequence)` 映射到不同 id；`session.closed` 为终态；transport 意外结束时进入 `lost` 状态，提示调用方需要 domain resync；`Detach` 等待 detach 回复（默认 2s），`Release` 最多等 500ms 并忽略回复，两者都关闭连接。
- Mutation helpers：`Submit`、`Stop`、`ApprovalRespond`、`Delete` 覆盖 wire 调用、参数校验和 §13 错误映射；timeout、断线、`request_aborted`、`submission_unknown`、`submission_ambiguous` 以及 `Submit` 的畸形结果都返回 unknown-result（绝不自动重发），并注明应读回的协议方法；`unknown_method` 作为 visible-disabled/WIP 情况暴露；`approval.respond` 的 `-32602` 映射为已经 cancelled，不报错、不重试。
- Session vertical slice：`internal/session` 组合 `SessionStream`、首屏 history 和 projection；支持 latest-turn 反向分页、`turnStart` 漂移时回退兼容扫描、`history_too_large`/page-cap 降级、按 branch 隔离的 durable/transient reducer、有限且按 `event.id` 去重的 attach 期间缓冲、approval/model progress/未知通知的显式暴露，以及 remote closed / transport lost / local release 的区分。`Head`/`Through`/`View` 提供一致快照；重连和 resync 仍由上层触发。
- Projection input blocks：`input/message` 的文本按 upstream 优先级提取——`user.content` 的 `text` blocks 优先，其次 `blocks` 里的 `text` blocks，最后拼接已记录的 `dim-shell-input` raw block；其他 raw block 保持为模型上下文，不进入 timeline；不再读取未文档化的顶层 `content` 字段。新增 `user-content-blocks` 和 `shell-input-raw-block` 两个 golden fixture，fixture 总数从 12 增至 14。
- Supervisor QA：补 dial/hello 超时、已分类 dial 错误保留、换行包装的 connect 错误分类、错误 network 拒绝、spawn 失败不声明 ownership、等待 listen 上限、只关闭自有 daemon、关闭时先停 stream 再 ping、按单连接重试、shutdown 错误、context 截止、endpoint 相对路径和 symlink canonicalize、按次返回 ping 连接数的 fake daemon。
- Supervisor reconnect：`Reconnect` 单飞、每次 reconnect 重新解析 endpoint、先关闭旧 attachment、只在自有 daemon 缺失时 respawn 一次、按 backoff 重试；hello 的 `instanceId` 变化推进 generation，旧 connection 标记 stale；`watch.go` 记录 transport 终止原因；Close 期间允许在飞连接注册并在随后关闭。
- Viewmodel：`internal/viewmodel` 把 projection snapshot 映射为 native 行（user/assistant/tool），工具卡片只保留有界 params/result 预览，`RunStateFrom` 只作为状态行提示；不读 daemon、不持有事实。
- UI tool/run：assistant、user、approval 之外增加 tool 卡片和 `RunRunning`/`RunStopping` 状态行；Stop 入口只在 running 时以 visible-disabled WIP 出现。
- Session follow-up：`SubmitWithID`/`SubmitBranchWithID` 返回 submission UUID，供 unknown-result 时用 `session.input_lookup` 读回而不是重发；`openFailure` 区分 closed/lost/detached，history 期间 transport loss 仍 fail-fast。
- UI WIP affordances：`New session`、`Reconnect`（仅在非 ready 时出现）、`Attach file` 和 approval 的 `Approve`/`Deny` 保持可见、dimmed、accessibility 标记 disabled，并用描述说明等待的上游方法；WIP 入口没有点击处理，不会报告 App Server 未执行的成功。Tab 顺序包含这些入口，headless 测试验证其可发现且不伪造成功。

**Verification（实际命令和结果）**

| 命令 | 结果 |
|---|---|
| `go build ./...` | 通过 |
| `go test -count=1 ./...` | 6 个包全部 `ok`（appserver、session、projection、supervisor、ui、viewmodel） |
| `go test -race -count=1 ./...` | 6 个包全部 `ok`（一次全量 race 曾出现 supervisor spawn 测试时序 flake，单包与复跑通过） |
| `go test -count=20 ./internal/session`、`go test -race -count=5 ./internal/session` | 通过 |
| `go vet ./...` | 通过 |
| `gofmt -l .`、`git diff --check` | 无输出、通过 |
| `go test -run TestGoldenFixtures -v ./internal/projection` | 14 个 golden fixture 全部通过 |

**Contract impact**：`protocol-contract.md` 增加 session stream/attach、detach、四个 mutation、首屏 history 组合和 unknown-result 边界的实现状态；`projection-contract.md` 增加 input text/block 提取语义、14 个 fixture 矩阵和 session 组合状态；没有新增本地 session/run/approval 事实，没有读取 App Server 私有文件。未实现的方法保持 WIP，不以本地状态补齐。

**Known gaps/blockers**

- 纵向闭环仍未接通：`cmd/dimagent` 仍使用手写 fixture，`internal/session` 和 `internal/viewmodel` 还没有接到 supervisor/UI；这是下一步 integration 的范围。
- session 层 re-attach/domain resync 未实现：supervisor 已提供 `Reconnect`、generation 和 `Connection.Done`/`Err`，但 `internal/session` 不会自己重开或重建 attachment，需要上层编排。
- `session.methods` 能力门禁、`session.keys`/`session.branches`、`session.input_lookup` 和 `session.create` 没有 Go 实现；unknown-result 已能返回 submission UUID，但读回方法本身仍是 WIP。
- approval mutation 已有 client helper，但 UI 的 approve/deny 仍是 visible-disabled WIP；attachment 上传和 approval replay 集合替换未实现。
- `internal/session` 的 branch 支持是“每个 branch 独立 reducer/transient/state”，没有 branch directory、cut point 或 side-chat 创建流程。
- 没有 real-daemon 测试：本机仍缺 `dim-app-server`，不宣称 real-daemon 通过。
- 真实窗口、中文/emoji IME、复制粘贴/撤销、VoiceOver 和字体 fallback 未验证；`internal/ui` 证据仍是 `headless`。
- `subscription_changed` 恢复和 Windows/Linux GUI 运行时仍未验证。
- `internal/supervisor` 的 `TestDefaultSpawnRunsDetachedAndLogs` 在并行全量 race 下出现过一次 spawn child 未退出的时序 flake；单包和复跑通过，仍应加宽等待或隔离。

**Next action**：只由一个 integration agent 把 `internal/session` + `internal/viewmodel` 接到 supervisor → UI 并补端到端 fake-server 场景；真实窗口手测和 real-daemon 验收在二进制和目标平台可用后单独记录，不因底层组件通过而提前标记 preview 可用。

## Milestone 1：Native UI 静态壳

状态：静态壳、headless fixtures 和 visible-disabled WIP 入口已实现；真实窗口视觉、IME 和 accessibility 手测未完成。

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

状态：任务 1–4 已实现（`headless` + `fake-server`）；Wave 2 补上 session stream/attach、detach、四个 mutation helper 和 supervisor reconnect/generation，并扩展 fake server 的 attach/detach、断线、事件冲突、unknown_method 和 unknown-result 场景；任务 5 的 session re-attach/domain resync 仍由上层编排；任务 6 的 latest-turn 已由 `internal/session` 覆盖，`subscription_changed` 恢复仍未覆盖。

任务：

1. 从 `dim-electron` protocol v3 wire 合同提取最小 Go 类型，不复制 Electron IPC；列表首选 `session.catalog`，不虚构 `session.list`。
2. NDJSON framing、request correlation、notification dispatch、malformed frame/size/timeout/cancel。
3. control connection、每 session 独立 stream、attach/detach；attach 前先订阅通知。
4. Supervisor：发现/启动 Desktop 自己拥有的 daemon，hello 校验，退出时只停止自己启动者。
5. reconnect、re-attach、domain-level resync、instanceId 变化和 capability unavailable；resync 不作为 wire 方法。
6. fake App Server：断线、乱序、重复、协议错误、history 分页不变量、subscription_changed 和未知 mutation 结果。

退出条件：transport/supervisor/reconnect 单测；fake server 可确定性重放；没有任何私有存储读取。

## Milestone 3：Projection 与 Session 最小闭环

状态：纯 projection（14 个 golden fixtures，含 user content blocks 和 `dim-shell-input` raw block）、session stream/attach 和 mutation helpers 已实现（`headless` + `fake-server`）；`cmd/dimagent` 接线、latest-turn、history 分页端到端和 submit/stop 闭环未开始。

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

### Wave 1：合同冻结后（已实现并提交）

- Native UX：已实现静态 AppFrame、Sidebar、Conversation fixture、Composer；
- Protocol：已实现 framing/client/read helpers、Supervisor 和受限 fake server；
- Projection：已实现纯 Go TimelineFold/transient projection 和 12 个 golden fixtures（Wave 2 增至 14）；
- QA：已补 headless/fake tests；真实窗口和 real-daemon 验收仍未做。

### Wave 2：纵向闭环组件（底层已实现，接线未开始）

已实现的底层组件：

- App Server：`SessionStream` 覆盖订阅先行 attach、head 兜底、通知缓冲/路由、事件幂等和 detach/closed/lost 生命周期；`Submit`/`Stop`/`ApprovalRespond`/`Delete` mutation helpers 覆盖成功、拒绝和 unknown-result 分类；
- Projection：input text blocks 和 `dim-shell-input` raw block 解码，14 个 golden fixtures；
- Supervisor QA：dial、spawn、close、endpoint 和 fake daemon 的失败路径与生命周期测试；
- Native UX：create session、reconnect、attachment 和 approval 决策的 visible-disabled WIP 入口。

仍未开始：只由一个 integration agent 接通

```text
supervisor → protocol → projection → ViewModel → native UI
```

`cmd/dimagent` 仍是手写 fixture。其他 agent 负责回归测试、错误注入、性能和可访问性，不再并行改同一条链。

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
- Wave 0 Toolchain/UI agent（deepseek-v4.1-flash）：已完成 Go 1.27.3 / MyGo v0.8.3 版本决策、最小窗口和 headless shell；提交 `7f89890` 已合并。
- Wave 0 Protocol research agent（deepseek-v4.1-flash）：已完成只读合同调查；`docs/protocol-contract.md` 已审阅并合并（`1977952`）。
- Wave 0 Projection research agent（deepseek-v4.1-flash）：已完成 TimelineFold 调查；`docs/projection-contract.md` 已提交（`bafef10`）。
- Wave 0 QA agent（deepseek-v4.1-flash）：已完成只读验收矩阵和 blocker 分类；报告已审阅。
- Wave 1 Protocol/Supervisor agent：已实现 `internal/appserver` + `internal/supervisor` 的 headless/fake 子集，见 [Wave 1 进展](#wave-1-进展协议supervisorprojection-和静态壳2026-11-08)。
- Wave 1 Projection agent：已实现 `internal/projection` 和 12 个 golden fixtures；Wave 2 增至 14 个，见 [Wave 1 进展](#wave-1-进展协议supervisorprojection-和静态壳2026-11-08)。
- Wave 1 Native UX agent：已实现 `internal/ui` 静态壳和 fixtures，见 [Wave 1 进展](#wave-1-进展协议supervisorprojection-和静态壳2026-11-08)。
- Wave 2 App Server agent：已实现 `SessionStream`、detach 和四个 mutation helper，以及流所有权/pre-attach 丢弃计数修复（`b0f853f`/`3ebfea3`/`7b0e728`）；未实现 create、keys/branches 和 input_lookup，见 [Wave 2 进展](#wave-2-进展session-streammutation-helpersinput-blocks-和-wip-入口2026-11-08)。
- Wave 2 Session agent：已实现 `internal/session` 组合层（首屏 history + 按 branch projection + transient/terminal 状态）和 `SubmitWithID`，提交 `8c4e7e8`/`064c58a`；尚未接到 UI。
- Wave 2 Viewmodel agent：已实现 projection snapshot → native 行映射，提交 `36703e9`；尚未接到 UI。
- Wave 2 Supervisor reconnect agent：已实现 reconnect 单飞、generation、transport 终止原因和 close 竞争修复，提交 `f41f276`/`415908c`/`2eb89f8`。
- Wave 2 Projection agent：已补 input content blocks 和 shell-input raw block，golden fixtures 增至 14 个。
- Wave 2 Supervisor QA agent：已补 dial、spawn、close、endpoint 和 fake daemon 的失败路径测试。
- Wave 2 Native UX agent：已补 visible-disabled WIP 入口、tool 卡片和 run 状态。
- Wave 2 integration agent：未开始；等 Wave 2 diff 审查结束后只由一人把 `internal/session` + `internal/viewmodel` 接到 supervisor/UI。

### QA 基线（2026-11-08）

- 已通过（`headless`）：MyGo v0.8.3 List/FollowEnd、Markdown、accessibility、IME 基础测试；项目 `go build ./...`、`go test -count=1 ./...`、`go test -race -count=1 ./...`、六目标 `go vet ./...`、Windows/Linux 交叉编译、`gofmt -l .` 和 `git diff --check`。
- 已通过（`headless` + `fake-server`）：项目内 FollowEnd/Markdown/键盘/状态/AX fixtures、WIP visible-disabled fixtures、Go fake server/fake daemon、session stream/attach、四个 mutation 的成功/拒绝/unknown-result 场景、`internal/session` 的 latest-turn/兼容扫描/降级/branch 隔离/terminal/close 测试、viewmodel 行映射、supervisor dial/spawn/close/endpoint/reconnect QA、14 个 projection golden fixtures。
- 仍是底层证据：session stream 只到 fake peer，supervisor 只到 fake daemon；没有 `real-daemon` 证据，也不能替代真实窗口、IME 或 VoiceOver 手测。
- 已通过 Electron 行为基线：protocol 16、shared 4、renderer IME 7、FollowEnd 64、Markdown 53，以及 typecheck/lint/boundary/docs。
- 阻断而非产品失败：electron-main 测试缺 Electron binary；real-daemon 缺 `dim-app-server`；两个浏览器 E2E 存在 bridge harness 漂移。
- 尚未满足 Wave 0 退出：真实窗口手测、真实 IME/VoiceOver、字体 fallback 和跨平台 GUI 运行时验证。
- 证据等级固定为 `headless`、`fake-server`、`real-daemon`、`manual`、`packaged`；低等级证据不得宣称高等级能力通过。

## Wave 0 研究结论（已收录）

- MyGo 组件：`ui.List` 的 keyed/dynamic rows 与 `FollowEnd` 是聊天首选；`ui.TextArea/TextBuffer` 是 Composer 首选；`ui.Markdown` 是 assistant 基础 renderer；`ui/editor` 延后用于代码/diff；richtext/chart/video/plugins 延后。
- 协议：不存在 `session.list`；`session.catalog` 是首选列表；`session.resync` 不是 wire method；attach 必须先订阅；history/live、approval replay、subscription_changed 和未知 mutation 结果都要有 fixtures。
- Projection：TimelineFold 是 durable transcript fold；RawEventBridge 是并行 live projection；approval/question/model progress 不应塞进 fold；Go golden 需要注入时钟。

Wave 0 研究已收录，Wave 1 和 Wave 2 底层组件已通过 `headless` + `fake-server` 验证；在纵向闭环、真实窗口手测和 real-daemon 验收完成前，不把 preview 标成可用。
