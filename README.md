# DimAgent MyGo Native

MyGo native UI 重实现实验项目。

## 目标

- 使用 [egoist/mygo](https://github.com/egoist/mygo) 的 Go native UI 构建 DimAgent 主界面。
- 通过 protocol v3 连接 Rust `dim-app-server`。
- 不在本项目建立 session、message、run 或 approval 的第二份事实存储。
- WebView 只作为未来隔离的插件/外部内容容器，不作为主界面。

## 当前状态

Wave 1 已实现：`internal/appserver` 的 protocol v3 framing、JSON-RPC client、wire DTO、catalog/summaries/history helpers 和错误映射；`internal/supervisor` 的 endpoint、dial/hello、spawn 和连接生命周期；`internal/projection` 的 TimelineFold、transient projection 和 golden runner；`internal/ui` 的静态 shell、`ui.List`/`FollowEnd`、Markdown、composer、键盘和 accessibility fixtures。

验证目前只到 `headless` + `fake-server`：`go build ./...`、`go test -race -count=1 ./...`、六目标 `go vet ./...` 通过，12 个 projection golden fixture 和 fake server/fake daemon 测试通过。纵向闭环尚未接通，`cmd/dimagent` 仍使用手写 fixture；没有 real-daemon、真实 IME/VoiceOver、复制粘贴/撤销或 Windows/Linux 运行时验证。证据等级约定见[开发计划](docs/development-plan.md#qa-基线2026-11-08)。

## 目录

```text
cmd/dimagent/       应用入口（当前为 fixture shell）
internal/appserver/ protocol v3 framing、client、wire DTO 和 read helpers
internal/projection/ TimelineFold 与 transient projection
internal/supervisor/ daemon 发现/启动、hello 和连接生命周期
internal/ui/        native view tree
docs/               研究、设计与验收记录
```

## 开发前提

`go.mod` 固定 Go 1.27.3 和 MyGo `v0.8.3`。MyGo 所有已发布 tag 都要求 Go 1.27.1 以上，所以本项目不能用 Go 1.24.x 构建。

本机 `/usr/local/go` 可以是更老的版本：`GOTOOLCHAIN=auto`（默认）会按 `go.mod` 自动下载 `go1.27.3` toolchain 到模块缓存（`~/go/pkg/mod/golang.org/toolchain@...`，约 243 MB），第一次构建需要网络。CI 或离线环境请预装 Go 1.27.3，或用 `GOTOOLCHAIN=local` 立即暴露版本不足。

```bash
go run ./cmd/dimagent     # 启动最小窗口
go test ./...             # headless/fake 测试，不需要窗口
go vet ./...
```

版本决策和实测结果见 [开发计划](docs/development-plan.md#milestone-0-进展toolchain-和最小窗口2026-11-08)。

## 参考

- [MyGo Native UI 组件调查](docs/mygo-native-components.md)
- [开发计划与 Subagent 编排](docs/development-plan.md)
- [Native UI 执行计划](docs/native-plan.md)
- [Protocol v3 Wire 合同](docs/protocol-contract.md)
- [Projection / TimelineFold 合同](docs/projection-contract.md)
- [dim-electron 产品基线](https://github.com/arcships/dim-electron)
