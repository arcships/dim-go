# DimAgent MyGo Native

MyGo native UI 重实现实验项目。

## 目标

- 使用 [egoist/mygo](https://github.com/egoist/mygo) 的 Go native UI 构建 DimAgent 主界面。
- 通过 protocol v3 连接 Rust `dim-app-server`。
- 不在本项目建立 session、message、run 或 approval 的第二份事实存储。
- WebView 只作为未来隔离的插件/外部内容容器，不作为主界面。

## 当前状态

当前实现（截至 2026-11-10，HEAD 375 commits）：

- **协议与运行时**：protocol v3 framing/client、supervisor spawn/hello/reconnect/generation、session stream + attach、首屏 history、按 branch 隔离的 projection、14 个 golden fixture；mutation（submit/stop/approval.respond/session.delete/session.rename/session.model.update/branch_create）与 unknown-result 读回。
- **主界面**：隐藏标题栏聊天壳、空态 hero、侧栏（项目分组、相对时间、置顶/归档）、composer（461/520 宽度、图标工具栏、context 状态带）、时间线（user/assistant/thinking/tool/file changes）、审批面板、keys 浏览器、右 context 面板、状态/通知条。
- **搜索与路由**：Setting 是全窗口路由（19 分区，默认个性化；Appearance 含 6 套 chrome 主题、主题/字体/字号/密度/语言）；⌘G 全局搜索浮层；⌘F 会话内查找；Scheduled / Skills 为 WIP 路由页。
- **动作**：创建/切换/删除会话、重命名、Pin/Archive（桌面本地隐藏列表）、Copy ID/debug/link（`dimagent://thread/<id>`）、模型切换、branch 切换/结束/刷新、cut point（Branch from here）、断线重连、提交/停止/审批、草稿与窗口状态持久化。
- **视觉层**：语义 token、Lucide 0.577 图标集、6 套 chrome 主题、elevation 阴影、150/250ms 过渡动画；字体栅格、渐变/3D/vibrancy、自定义 skin 仍是已知差异。

验证：`gofmt -l .`、`git diff --check` 空；`go build ./...`、`go vet ./...`、`go test -count=1 ./...` 通过（10 个包、937 个顶层测试）；`go test -race -count=1 ./...` 通过（`internal/ui` 约 340s，已列性能项）。opt-in real-daemon `TestRealDaemonSmoke` / `TestRealDaemonSessionFlow` 通过（create → attach → history → submit → stop；daemon 声明 branches 时覆盖 branch_create → open → refresh → end）。真实窗口证据在 `docs/window-evidence.md` 与 `docs/evidence/`，UI 并排对照在 `docs/evidence/ui-parity/`。

**产品验收：12 通过 / 16 未通过**（F1–F5、F10、F12a、F12b、F14a、F14b、A1、A4 为 L4；F6–F8、F13 的 AX 窗口记录保留 provider 描述，但实际 provider/daemon 未独立复核，不作协议/daemon 验收依据；其余见[验收矩阵](docs/acceptance.md)）。测试通过不等于产品验收；视觉项要求同尺寸同状态截图 + 逐区域视觉复核，当前 D1–D7 全部未通过。`cmd/dimagent` 默认不连 daemon；设置 `DIMAGENT_PRODUCT_HOME` 才启动 supervisor。`scripts/package-macos.sh` 可生成带 `dimagent` scheme 的 macOS .app（ad-hoc 签名，未公证）。剩余阻塞：真实模型/审批流量（F6/F7/F8）、真实 `subscription_changed`（F11）、unknown-result 读回展示（F9）、附件运行来源复核及图片选择（F13）、VoiceOver/IME/Windows/Linux（A2/A3/A5）、字体/图标/皮肤全量像素对齐（D1–D7）。

## 目录

```text
cmd/dimagent/       应用入口（DIMAGENT_PRODUCT_HOME opt-in 连 daemon）
internal/appserver/ protocol v3 framing、client、wire DTO、read helpers、
                    session stream/attach 和 mutation helpers
internal/config/    桌面本地外观与语言偏好（settings.json 读写，不存 session 事实）
internal/projection/ TimelineFold、transient projection 和 14 个 golden fixtures
internal/session/   stream + 首屏 history + 按 branch 隔离的内存会话视图
internal/viewmodel/ projection snapshot → native 行（user/assistant/tool）
internal/integration/ supervisor → session → viewmodel 的会话桥
internal/supervisor/ daemon 发现/启动、hello、reconnect 和连接生命周期
internal/ui/        native view tree（设置页、状态栏、en/zh-CN 消息层）；
                    未接通动作以 visible-disabled WIP 呈现
scripts/seed-session/ 用公开 session.create 预置窗口证据会话
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

## 打包

macOS 本地 `.app`（开发件，未公证）：

```bash
scripts/package-macos.sh --out /tmp/dimagent-app --version 0.1.0
open -a /tmp/dimagent-app/DimAgent.app
```

产物在 `/tmp/dimagent-app/DimAgent.app`，`Info.plist` 声明 `dimagent://thread/<session-id>` 深链。默认 ad-hoc 签名，没有 Developer ID、公证和安装器，不能当发布件；如果系统里已安装同样声明 `dimagent` 的 DimAgent，默认 handler 仍归它，本地 bundle 用 `open -a <路径>` 验证。命令参数、产物结构、深链注册方式和本地验证步骤见[发布与打包](docs/release.md)。

## 参考

- [MyGo Native UI 组件调查](docs/mygo-native-components.md)
- [发布与打包](docs/release.md)
- [项目流程](docs/workflow.md)
- [Agent 组织与开发方案](docs/organization-model.md)
- [验收矩阵](docs/acceptance.md)
- [开发计划](docs/development-plan.md)
- [Native UI 执行计划](docs/native-plan.md)
- [UI 1:1 对齐审计](docs/ui-parity.md)
- [真实窗口截图证据](docs/window-evidence.md)
- [Protocol v3 Wire 合同](docs/protocol-contract.md)
- [Projection / TimelineFold 合同](docs/projection-contract.md)
- [进度归档](docs/archive/progress-2026-11.md)
- [dim-electron 产品基线](https://github.com/arcships/dim-electron)
