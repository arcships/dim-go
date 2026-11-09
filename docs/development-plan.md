# DimAgent MyGo 开发计划

Status: active（Wave 1/2 的底层组件已实现并有测试级证据；**产品验收未通过**，见 [acceptance.md](acceptance.md)）
Last updated: 2026-11-09
流程与角色见 [workflow.md](workflow.md)；逐日记录见 [archive/progress-2026-11.md](archive/progress-2026-11.md)。

## 目标与边界

本项目用 MyGo native UI 重实现 DimAgent Desktop 主界面；Rust `dim-app-server` 仍是 session、branch、timeline、approval、model、permission 的唯一事实源。Go 项目不读 App Server 私有 SQLite/JSONL/daemon.log，不建立本地 session/run/approval 真相，也不把 Electron IPC 搬进来。

主界面优先使用 MyGo 公开 `ui` API：`ui.List`、`ui.TextArea/TextBuffer`、`ui.Markdown`、accessibility、theme、SVG；`editor`、`richtext`、chart、video 和 plugins 按阶段启用。调查细节见 [mygo-native-components.md](mygo-native-components.md)。

## 关键技术前置

- Toolchain：`go.mod` 固定 Go 1.27.3 和 MyGo `v0.8.3`；`GOTOOLCHAIN=auto` 自动下载 toolchain。实测证据见 [archive/progress-2026-11.md](archive/progress-2026-11.md#milestone-0-进展toolchain-和最小窗口2026-11-08)。
- 不能复制 `github.com/egoist/mygo/internal/...`；只使用公开 API。
- 跨 agent 的领域合同以 Go package、wire DTO 和 golden fixtures 为准，不靠口头约定。
- protocol v3 的 session 列表是 `session.catalog`（首选）或受限的 `session.summaries`，不是 `session.list`。
- `session.resync` 是 Desktop domain event，不是 wire RPC；触发后重新 attach 并重读窗口，不按 cursor 伪造重放。
- Projection 必须同时覆盖 durable `TimelineFold` 和 live `RawEventBridge` 的一致性；approval/question 与 model progress 是独立 transient projection，不塞进 TimelineFold。
- Go golden fixture 注入时钟，避免 `GoalFoldState` 等逻辑使用 `Date.now()` 造成跨语言不确定性。

## 里程碑路线图

| 里程碑 | 状态 | 退出条件 |
|---|---|---|
| M0 工具链与组件 Spike | 已完成 | 窗口可启动；核心组件有可复现实测；版本锁定 |
| M1 Native UI 静态壳 | 已完成（headless） | AppFrame/Sidebar/List/Composer/Markdown + 键盘与无障碍 fixture；真实窗口视觉未验收 |
| M2 协议、Transport、Supervisor | 已完成 | framing/correlation/错误映射、dial/hello/spawn/reconnect；fake + real-daemon 控制面证据 |
| M3 Projection 与 Session 纵向闭环 | 已打通（测试级） | 首屏 history、branch 隔离、stream 折叠、submit/stop/approval、reattach；真实窗口 L4 未验收 |
| M4 核心 Agent 能力 | 进行中 | approval（已接 UI）、model progress、goal/title、tasks/workflows；question/always-allow 未做 |
| M5 管理面与 Desktop 服务 | 未开始 | 设置、skills/market、scheduled、多窗口/托盘 |
| M6 发布与并行 | 未开始 | 打包、跨平台运行时、退役 Electron 路径 |

各里程碑的详细组件草图：M1 见 [native-plan.md](native-plan.md)；wire 合同见 [protocol-contract.md](protocol-contract.md)；projection 见 [projection-contract.md](projection-contract.md)。

## 当前波次

**Wave 3：UI 1:1 对齐（进行中）** —— 顺序不能换：

1. ✅ token 语义层与量纲层（`2701701` 之前的 `2c888e1`）：elevation、前景配对、sidebar/tooltip 语义集、侧栏 256/200–360、四级圆角、字号档位。
2. ✅ 布局提取（`docs/ui-layout-spec.md`，596 行）：容器树、flex 方向、宽高/min/max/padding/gap、滚动与虚拟化、Tailwind/变量出处（file:line）、MyGo API 映射与缺口。组件实现以这份规格为验收基线。
3. 🚧 P0 组件：✅ 侧栏（`2701701`）→ ✅ timeline 行模型（`760413e`：thinking 折叠、tool 展开、file changes、error/retry）→ ⏭ composer（自适应/附件 chip/模式选择）与 approval 面板。
4. ✅ 截图证据闭环（`8b05e98` + 每次 UI 改动重出）：`Tester.Image()` light/dark PNG；真实窗口证据见 [window-evidence.md](window-evidence.md)。跨端人工对照仍未做。

差距清单见 [ui-parity.md](ui-parity.md)；布局规格见 [ui-layout-spec.md](ui-layout-spec.md)。

并行待办（不阻塞 Wave 3）：

- branch switching / cut point：目前只有只读 branch 列表，没有已验证的切换 RPC。
- attachment 上传：没有 Go helper，保持 visible-disabled WIP。
- `subscription_changed` 真实 wire 恢复：当前只读单次重试、写操作 fail-closed。
- `session.methods` 插件入口接线。

## 验收与证据

证据等级 L0–L5 与逐项状态见 [acceptance.md](acceptance.md)。规则：

- L0 source-read / L1 headless / L2 fake / L3 real-daemon 只能写成“测试级证据”。
- UI 与视觉项必须同时具备 L4（真实窗口）与 L5（设计对照）才算通过。
- 未通过项保持 visible-disabled WIP，不用本地状态补齐。

## 文档地图

| 想了解 | 读 |
|---|---|
| 项目流程、角色、门禁 | [workflow.md](workflow.md) |
| 验收标准与当前状态 | [acceptance.md](acceptance.md) |
| UI token/组件差距 | [ui-parity.md](ui-parity.md) |
| 协议 wire 合同 | [protocol-contract.md](protocol-contract.md) |
| projection 语义 | [projection-contract.md](projection-contract.md) |
| Native UI 执行计划 | [native-plan.md](native-plan.md) |
| MyGo 组件调查 | [mygo-native-components.md](mygo-native-components.md) |
| 历史进展记录 | [archive/progress-2026-11.md](archive/progress-2026-11.md) |
