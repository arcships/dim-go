# Native UI 执行计划

Status: active（Wave 1 静态壳与组件验证已实现，证据等级 `headless`；纵向闭环和真实窗口手测未开始）

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
- 长列表和滚动行为的 headless fixture。

未完成：真实窗口截图验收、流式增长下的人工视觉检查、真实 IME/AX 验收。

## Milestone 2：协议和投影

已实现（`headless` + `fake-server`）：

- protocol v3 framing、client correlation/timeout/cancel、wire DTO 和错误映射；
- Supervisor endpoint/dial/hello/spawn 与只关闭自有 daemon 的退出路径；
- TimelineFold 与 transient projection；
- session catalog/history read helpers；
- fake App Server（`net.Pipe`）和 fake daemon 测试。

未完成：reconnect/re-attach/domain resync、`session.create`、`session.keys`/`session.branches`、`session.input_lookup`、approval mutation、attachment 上传和真实 daemon 验证。

## Milestone 3：第一个纵向闭环

未开始。

- open session
- render history
- submit
- live event
- stop
- reconnect/resync

`cmd/dimagent` 仍使用手写 fixture，尚未接线 supervisor → appserver → projection → UI。

## Milestone 4：核心能力

未开始。

- branch/side chat
- approval/question
- model progress（Go transient projection 已实现，未接入 UI）
- goal/title
- tasks/workflows

## Milestone 5：发布基线

未开始。

- macOS arm64
- Windows/Linux 后端验证
- 打包、签名、崩溃诊断
- 与 Electron 结果对比
