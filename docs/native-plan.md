# Native UI 执行计划

Status: active（Wave 1 静态壳与组件验证、Wave 2 visible-disabled WIP 入口已实现，证据等级 `headless`；纵向闭环和真实窗口手测未开始）

## Milestone 0：组件验证

已实现（headless/fake）：

- MyGo native window 入口和静态 shell；
- `ui` 基础布局；
- `ui.Text` / `ui.TextArea` composer；`TextBuffer` 未单独接入；
- `ui.List` / `ui.Scroll`、keyed rows 和 `FollowEnd`；
- `ui.Markdown`；
- theme tokens、键盘导航、状态行 live region 和 accessibility labels；
- headless `ui.NewTester` fixtures。

未完成（blocker）：

- 真实窗口的视觉和交互手测；
- 中文/emoji IME、composition 光标、复制粘贴、撤销、字体 fallback；
- VoiceOver 等辅助技术朗读；
- `ui.Editor` 的代码编辑能力；当前代码块仍由 `ui.Markdown` 渲染；
- Windows/Linux GUI 运行时验证。

## Milestone 1：静态聊天壳

已实现：

- AppFrame；
- SessionSidebar；
- ConversationView（`ui.List` + `FollowEnd`）；
- ComposerView（`ui.TextArea`）；
- Markdown/代码块 fixture；
- 长列表和滚动行为的 headless fixture；
- visible-disabled WIP 入口：`New session`、`Reconnect`（仅非 ready 时显示）、`Attach file`；入口保留位置和可访问名称，标记 disabled、dimmed，并以描述说明等待的上游方法，不处理点击、不报告成功；
- approval 的 `Approve`/`Deny` 已接通 `approval.respond`：有 pending request 和回调时是真实按钮，无回调或无 requestID 时退回 visible-disabled WIP；点击只发送一次 decision，`-32602`/cancelled 显示为 notice 而不是成功；
- Tab 顺序覆盖上述入口，screen reader/keyboard 用户能发现并读到不可用原因。

未完成：真实窗口截图验收、流式增长下的人工视觉检查、真实 IME/AX 验收。

## Milestone 2：协议和投影

已实现（`headless` + `fake-server`）：

- protocol v3 framing、client correlation/timeout/cancel、wire DTO 和错误映射；
- Supervisor endpoint/dial/hello/spawn 与只关闭自有 daemon 的退出路径，以及 Wave 2 的 dial/spawn/close/endpoint QA；
- TimelineFold、transient projection 和 14 个 golden fixtures（含 user content blocks、shell-input raw block）；
- session catalog/history read helpers；
- `SessionStream`：订阅先行 attach、head 兜底、通知缓冲/路由、事件幂等、detach/closed/lost 生命周期；
- submit/stop/approval.respond/session.delete mutation helpers 和 unknown-result 分类；
- fake App Server（`net.Pipe`）和 fake daemon 测试；
- `internal/session` 组合层：首屏 latest-turn/兼容扫描/history 降级、按 branch 的 durable/transient projection、bounded pending buffer、terminal/close 语义；`BranchDirectory` 已按公开 Desktop protocol contract 的 `session.branches` capability-gated 读取并暴露只读 snapshot；`Branches()` 只含从事件观察到的 branch id，尚未接入 UI 切换或 cut point。
- `internal/viewmodel`：projection snapshot → native 行（user/assistant/tool）；
- `internal/integration`：supervisor → session → viewmodel 的 `SessionController`（Open/Catalog/State/Close），fake daemon 端到端测试；
- Supervisor reconnect：单飞、generation、旧 attachment 失效、transport 终止原因和 close 竞争处理。

未完成：UI branch switching/cut point（`BranchDirectory` 已有只读 snapshot，`Branches()` 仅含事件观察到的 branch id）、`session.methods` 插件入口接线、approval replay 集合替换、附件上传、`session.create` 与 unknown-result UI 接线。`Reattach` 已实现：只在 transport 终止时重建 attachment 和投影，replay gap 时只读路径单次重试、写操作 fail-closed，不再依赖上层临时编排。`session.create`、`session.methods`、`session.input_lookup` 已有 Go client/helper；`session.methods` 和 `session.input_lookup` 有公开合同依据（后者响应 `{status:"accepted",event}` 或 `{status:"unknown"}`）。`session.keys`/`session.branches` 已有 decoder/types 和 `SessionKeys`/`SessionBranches` helper；`internal/session.Open` 已 capability-gated 读取并暴露只读 snapshot。UI branch switching/cut point 仍未完成。真实 daemon 已有 opt-in 的 `TestRealDaemonSmoke` 和 `TestRealDaemonSessionFlow`，后者覆盖 create → attach → history → submit → stop → cleanup；两者都不覆盖审批 replay、`subscription_changed` 恢复、GUI 或 IME。

## Milestone 3：第一个纵向闭环

启动路径、session 切换、submit/stop/reconnect 和 approval 决策已接入（只有 `headless` + `fake-server` 测试级证据，real-daemon session 流仅覆盖协议路径，**产品验收未通过**）；持续 live ticker、真实窗口手测、IME/VoiceOver 和设计与组件 1:1 对齐都未完成。

- open session（`cmd/dimagent` 在 `DIMAGENT_PRODUCT_HOME` 下经 supervisor → integration 打开首个 session）
- render history（首屏 history + viewmodel 渲染快照）
- submit（`SubmitWithID` 接到窗口，未知结果只读回不重发）
- live event（stream 路由和 session drain 已有，窗口没有持续 ticker，按动作/选中后刷新）
- stop（`session.stop` 接到窗口，只在 running 时出现）
- reconnect/resync（supervisor `Reconnect`/generation 已接入窗口；`SessionController.Reattach` 在 transport 终止时重建 attachment，`subscription_changed` 写操作 fail-closed）

`cmd/dimagent` 未设置 `DIMAGENT_PRODUCT_HOME` 时显示未连接，不再使用手写 fixture。

## Milestone 4：核心能力

未开始。

- branch/side chat
- approval/question（`approval.respond` 已接到 UI：`allow`/`deny` 折叠、`-32602` 按 cancelled 处理并移除 pending、不重放；question/elicitation 和 replay 集合替换未实现）
- model progress（Go transient projection 已实现，未接入 UI）
- goal/title
- tasks/workflows

## Milestone 5：发布基线

未开始。

- macOS arm64
- Windows/Linux 后端验证
- 打包、签名、崩溃诊断
- 与 Electron 结果对比

## Visible-disabled WIP 入口

| 入口 | 当前行为 | 等待的能力 |
|---|---|---|
| `New session` | 已接线：有 `OnCreateSession` 时是真实按钮，创建中禁用；无回调时可见、dimmed、disabled | 无（`session.create` + catalog 刷新已实现） |
| `Reconnect` | 仅在连接非 ready 时出现；有 `OnReconnect` 时是真实按钮 | 持续 resync 语义（replay gap 已 fail-closed） |
| `Attach file` | 可见但不可用，描述 `attachment.put` 未接通 | attachment 上传 |
| `Approve` / `Deny` | 已接线：有 pending request 时是真实按钮，`-32602` 按 cancelled 处理 | question/always-allow、replay |
| approval 行本身 | 已接线：`viewmodel.BuildApprovals` 从 daemon pending approvals 生成行 | approval replay 集合替换 |

这些入口在 headless 测试中验证：存在、accessible name 保留、disabled 状态和原因可读、点击不产生成功状态。真实窗口、IME、VoiceOver 和视觉检查仍未完成。
