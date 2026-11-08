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
- visible-disabled WIP 入口：`New session`、`Reconnect`（仅非 ready 时显示）、`Attach file`、approval 的 `Approve`/`Deny`；入口保留位置和可访问名称，标记 disabled、dimmed，并以描述说明等待的上游方法，不处理点击、不报告成功；
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
- `internal/session` 组合层：首屏 latest-turn/兼容扫描/history 降级、按 branch 的 durable/transient projection、bounded pending buffer、terminal/close 语义；
- `internal/viewmodel`：projection snapshot → native 行（user/assistant/tool）；
- Supervisor reconnect：单飞、generation、旧 attachment 失效、transport 终止原因和 close 竞争处理。

未完成：session 层 re-attach/domain resync、`session.create`、`session.keys`/`session.branches`、`session.input_lookup`、`session.methods`、approval replay 集合替换、附件上传、approval UI 接线、session/viewmodel UI 接线和真实 daemon 验证。

## Milestone 3：第一个纵向闭环

底层组件已实现（`headless` + `fake-server`），UI 接线未开始。

- open session（`SessionStream` + `internal/session` 已有，未接入 UI）
- render history（projection、首屏 history 和 viewmodel 已有，未接入 UI）
- submit（`internal/session.Submit`/`SubmitWithID` 已有，未接入 UI）
- live event（stream 路由和 session drain 已有，未接入 UI）
- stop（`internal/session.Stop` 已有，未接入 UI）
- reconnect/resync（supervisor `Reconnect`/generation 已有；session re-attach/rebuild 未实现）

`cmd/dimagent` 仍使用手写 fixture，尚未接线 supervisor → session → viewmodel → UI。

## Milestone 4：核心能力

未开始。

- branch/side chat
- approval/question（`approval.respond` client helper 和 `-32602` 映射已实现；UI 的 `Approve`/`Deny` 仍是 visible-disabled WIP，replay 集合替换未实现）
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
| `New session` | 可见、dimmed、标记 disabled，描述 `session.create` 未接通 | `session.create` client helper 和 UI 接线 |
| `Reconnect` | 仅在连接非 ready 时出现，不处理点击 | reconnect/re-attach/domain resync |
| `Attach file` | 可见但不可用，描述 `attachment.put` 未接通 | attachment 上传 |
| `Approve` / `Deny` | approval 行保留两个决策，不可用，不报告决策成功 | `approval.respond` UI 接线和 replay |
| approval 行本身 | UI 有 `AuthorApproval` fixture；transient projection 只有通用 notice，未接线 | approval 专用 transient 投影和接线 |

这些入口在 headless 测试中验证：存在、accessible name 保留、disabled 状态和原因可读、点击不产生成功状态。真实窗口、IME、VoiceOver 和视觉检查仍未完成。
