# DimAgent MyGo Native

MyGo native UI 重实现实验项目。

## 目标

- 使用 [egoist/mygo](https://github.com/egoist/mygo) 的 Go native UI 构建 DimAgent 主界面。
- 通过 protocol v3 连接 Rust `dim-app-server`。
- 不在本项目建立 session、message、run 或 approval 的第二份事实存储。
- WebView 只作为未来隔离的插件/外部内容容器，不作为主界面。

## 当前状态

Wave 1 已实现：`internal/appserver` 的 protocol v3 framing、JSON-RPC client、wire DTO、catalog/summaries/history helpers 和错误映射；`internal/supervisor` 的 endpoint、dial/hello、spawn 和连接生命周期；`internal/projection` 的 TimelineFold、transient projection 和 golden runner；`internal/ui` 的静态 shell、`ui.List`/`FollowEnd`、Markdown、composer、键盘和 accessibility fixtures。

Wave 2 已实现底层组件：`internal/appserver` 的 `SessionStream`（订阅先行 attach、head 兜底、通知路由、事件幂等和 detach/closed/lost 生命周期）与 submit/stop/approval.respond/session.delete mutation helpers（含 unknown-result 读回提示和可回读的 submission UUID）；`internal/session` 把 stream、首屏 history（latest-turn、兼容扫描、`history_too_large`/page-cap 降级）和按 branch 隔离的 projection 组合成可重建的内存会话视图；`internal/viewmodel` 把 projection snapshot 映射成 native 行（user/assistant/tool）；`internal/integration` 把 supervisor → session → viewmodel 组合成 `SessionController`（开 session、首屏、catalog、状态与 detach）；`cmd/dimagent` 已支持 opt-in 的 daemon 启动路径；`internal/projection` 增加 user content text blocks 和 `dim-shell-input` raw block 解码，golden fixtures 增至 14 个；`internal/supervisor` 增加 reconnect、generation、transport 终止原因和断线测试；`internal/ui` 增加 tool 卡片、run 状态；approval 的 `Approve`/`Deny` 已接到 `approval.respond`（只有 App Server 接受才更新状态，`-32602` 按 cancelled 处理），create session、reconnect、attachment 仍是 visible-disabled WIP 入口。

验证目前以 `headless` + `fake-server` 为主：`go build ./...`、`go test -race -count=1 ./...`、`go vet ./...`、`gofmt -l .` 通过，8 个包（cmd、appserver、integration、session、projection、supervisor、ui、viewmodel）测试通过，14 个 projection golden fixture 和 fake server/fake daemon 测试通过。`../dim-2` 可以构建独立的 `dim-app-server`；项目新增了默认跳过的 real-daemon 控制面 smoke。已用 `/tmp/dim-2-target/debug/dim-app-server` 实测 spawn/hello/catalog/close 通过，并用 opt-in 的 `TestRealDaemonSessionFlow` 实测 create → attach → history → submit → stop → cleanup；**这些只是测试级证据，不是产品验收**：不打开真实窗口、不验证视觉/交互动效/IME/VoiceOver，也不代表设计 token 与组件已和 Electron 端 1:1 对齐。产品验收标准与当前状态见[验收矩阵](docs/acceptance.md)，目前没有任何一项达到 `accepted`。`cmd/dimagent` 默认不连 daemon、也不使用 fixture；只有设置 `DIMAGENT_PRODUCT_HOME` 时才启动 supervisor 并打开第一个 session。窗口已通过公开 controller 接通 session 切换、创建、submit/stop、approval 决策和 unknown-result 读回；动作只在 App Server 接受后清空草稿/刷新内存投影，不伪造成功。仍没有持续 live ticker，session 层 domain resync 和 `subscription_changed` 真实 wire 恢复仍未验证（`SessionController.Reattach` 已加固：只对 transport 终止的 session 重建，replay gap 时只读路径只重试一次、写操作 fail-closed 不重放），真实 IME/VoiceOver、复制粘贴/撤销和 Windows/Linux 运行时均未验证；`session.methods`/`session.input_lookup` 的 Go 读 helper 已实现且有公开合同依据（`session.input_lookup` 响应为 `{status:"accepted",event}` 或 `{status:"unknown"}`），`session.keys`/`session.branches` 的公开 Desktop protocol contract 已有（`session-reader.ts:1041-1074` 调用，`wire.ts:267-303` 定义 DTO）；MyGo 已有 decoder/`SessionKeys`/`SessionBranches` helper，`session.Open` 已按 `hello.features` capability-gated 读取并暴露只读 snapshot；UI branch switching/cut point 和真实验收仍未完成；UI 与 Electron 端的 1:1 差距见[UI 对齐审计](docs/ui-parity.md)（当前不是 1:1）。证据等级约定见[验收矩阵](docs/acceptance.md)。

## 目录

```text
cmd/dimagent/       应用入口（DIMAGENT_PRODUCT_HOME opt-in 连 daemon）
internal/appserver/ protocol v3 framing、client、wire DTO、read helpers、
                    session stream/attach 和 mutation helpers
internal/projection/ TimelineFold、transient projection 和 14 个 golden fixtures
internal/session/   stream + 首屏 history + 按 branch 隔离的内存会话视图
internal/viewmodel/ projection snapshot → native 行（user/assistant/tool）
internal/integration/ supervisor → session → viewmodel 的会话桥
internal/supervisor/ daemon 发现/启动、hello、reconnect 和连接生命周期
internal/ui/        native view tree，未接通动作以 visible-disabled WIP 呈现
docs/               研究、设计与验收记录
```

## 开发前提

`go.mod` 固定 Go 1.27.3 和 MyGo `v0.8.3`。MyGo 所有已发布 tag 都要求 Go 1.27.1 以上，所以本项目不能用 Go 1.24.x 构建。

本机 `/usr/local/go` 可以是更老的版本：`GOTOOLCHAIN=auto`（默认）会按 `go.mod` 自动下载 `go1.27.3` toolchain 到模块缓存（`~/go/pkg/mod/golang.org/toolchain@...`，约 243 MB），第一次构建需要网络。CI 或离线环境请预装 Go 1.27.3，或用 `GOTOOLCHAIN=local` 立即暴露版本不足。

```bash
go run ./cmd/dimagent     # 启动最小窗口；未设置 DIMAGENT_PRODUCT_HOME 时显示未连接
go test ./...             # headless/fake 测试，不需要窗口
go vet ./...
```

设置 `DIMAGENT_PRODUCT_HOME` 才会启动/连接 `dim-app-server` 并打开 catalog 的第一个 session：

```bash
DIMAGENT_PRODUCT_HOME="$HOME/.dimcode/v2" go run ./cmd/dimagent
```

应用只通过 supervisor 使用公开 socket/pipe；不读 daemon 的 SQLite/JSONL/session 文件或 `daemon.log`。启动、catalog 或首个 session 打开失败时窗口显示连接错误，不会回退到本地方案或伪造会话。

版本决策和实测结果见 [进度归档](docs/archive/progress-2026-11.md#milestone-0-进展toolchain-和最小窗口2026-11-08)。

## 参考

- [MyGo Native UI 组件调查](docs/mygo-native-components.md)
- [项目流程](docs/workflow.md)
- [验收矩阵](docs/acceptance.md)
- [开发计划](docs/development-plan.md)
- [Native UI 执行计划](docs/native-plan.md)
- [UI 1:1 对齐审计](docs/ui-parity.md)
- [Protocol v3 Wire 合同](docs/protocol-contract.md)
- [Projection / TimelineFold 合同](docs/projection-contract.md)
- [进度归档](docs/archive/progress-2026-11.md)
- [dim-electron 产品基线](https://github.com/arcships/dim-electron)
