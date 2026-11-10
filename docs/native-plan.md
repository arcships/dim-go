# Native UI 执行计划

Status: proposal

## Milestone 0：组件验证

- MyGo native window
- `ui` 基础布局
- `ui.Text` / `ui.TextField` / `ui.TextArea`
- `ui.List` / `ui.Grid` / `ui.Scroll`
- `ui.Editor` 的代码编辑能力
- 主题、字体、输入法、复制选择
- fake/headless 测试

## Milestone 1：静态聊天壳

- AppFrame
- SessionSidebar
- ConversationView
- ComposerView
- Markdown/代码块 fixture
- 长列表和滚动性能

## Milestone 2：协议和投影

- protocol v3 wire
- Supervisor
- TimelineFold
- session catalog/history
- fake App Server

## Milestone 3：第一个纵向闭环

- open session
- render history
- submit
- live event
- stop
- reconnect/resync

## Milestone 4：核心能力

- branch/side chat
- approval/question
- model progress
- goal/title
- tasks/workflows

## Milestone 5：发布基线

- macOS arm64
- Windows/Linux 后端验证
- 打包、签名、崩溃诊断
- 与 Electron 结果对比
