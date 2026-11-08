# DimAgent MyGo Native

MyGo native UI 重实现实验项目。

## 目标

- 使用 [egoist/mygo](https://github.com/egoist/mygo) 的 Go native UI 构建 DimAgent 主界面。
- 通过 protocol v3 连接 Rust `dim-app-server`。
- 不在本项目建立 session、message、run 或 approval 的第二份事实存储。
- WebView 只作为未来隔离的插件/外部内容容器，不作为主界面。

## 当前状态

Wave 1 已实现：`internal/appserver` 的 protocol v3 framing、JSON-RPC client、wire DTO、catalog/summaries/history helpers 和错误映射；`internal/supervisor` 的 endpoint、dial/hello、spawn 和连接生命周期；`internal/projection` 的 TimelineFold、transient projection 和 golden runner；`internal/ui` 的静态 shell、`ui.List`/`FollowEnd`、Markdown、composer、键盘和 accessibility fixtures。

Wave 2 已实现底层组件：`internal/appserver` 的 `SessionStream`（订阅先行 attach、head 兜底、通知路由、事件幂等和 detach/closed/lost 生命周期）与 submit/stop/approval.respond/session.delete mutation helpers（含 unknown-result 读回提示和可回读的 submission UUID）；`internal/session` 把 stream、首屏 history（latest-turn、兼容扫描、`history_too_large`/page-cap 降级）和按 branch 隔离的 projection 组合成可重建的内存会话视图；`internal/viewmodel` 把 projection snapshot 映射成 native 行（user/assistant/tool）；`internal/integration` 把 supervisor → session → viewmodel 组合成 `SessionController`（开 session、首屏、catalog、状态与 detach）；`cmd/dimagent` 已支持 opt-in 的 daemon 启动路径；`internal/projection` 增加 user content text blocks 和 `dim-shell-input` raw block 解码，golden fixtures 增至 14 个；`internal/supervisor` 增加 reconnect、generation、transport 终止原因和断线测试；`internal/ui` 增加 tool 卡片、run 状态，并为 create session、reconnect、attachment 和 approval 决策提供 visible-disabled WIP 入口。

验证目前以 `headless` + `fake-server` 为主：`go build ./...`、`go test -race -count=1 ./...`、`go vet ./...`、`gofmt -l .` 通过，8 个包（cmd、appserver、integration、session、projection、supervisor、ui、viewmodel）测试通过，14 个 projection golden fixture 和 fake server/fake daemon 测试通过。`../dim-2` 可以构建独立的 `dim-app-server`；项目新增了默认跳过的 real-daemon 控制面 smoke。已用 `/Users/eric8810/Code/dim-2/target/debug/dim-app-server` 实测 spawn/hello/catalog/close 通过；这只提升控制面证据，不覆盖真实 session 流、GUI 或 IME。`cmd/dimagent` 默认不连 daemon、也不使用 fixture；只有设置 `DIMAGENT_PRODUCT_HOME` 时才启动 supervisor 并打开第一个 session。窗口目前只渲染一次快照，没有 session 切换、live 刷新、submit/stop 或审批决策接线；session 层的 re-attach/resync、session methods/keys/branches/input_lookup、approval UI、真实 IME/VoiceOver、复制粘贴/撤销和 Windows/Linux 运行时均未验证。证据等级约定见[开发计划](docs/development-plan.md#qa-基线2026-11-08)。

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

版本决策和实测结果见 [开发计划](docs/development-plan.md#milestone-0-进展toolchain-和最小窗口2026-11-08)。

## 参考

- [MyGo Native UI 组件调查](docs/mygo-native-components.md)
- [开发计划与 Subagent 编排](docs/development-plan.md)
- [Native UI 执行计划](docs/native-plan.md)
- [Protocol v3 Wire 合同](docs/protocol-contract.md)
- [Projection / TimelineFold 合同](docs/projection-contract.md)
- [dim-electron 产品基线](https://github.com/arcships/dim-electron)
