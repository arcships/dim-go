# MyGo Native GUI 组件调查

调查对象：[egoist/mygo](https://github.com/egoist/mygo) 当前主分支（2026-11-08）。本地 checkout 位于 `/tmp/mygo-research`，其 `go.mod` 当前声明 Go `1.27.3`；本机 Go `1.24.3` 尚不能直接按主分支构建。

## 结论

MyGo 并不是只有窗口和几个基础控件。它已经包含一套相当完整的 native UI toolkit：布局、文本、输入法、虚拟列表、代码编辑器、Markdown、富文本、可访问性和测试基础设施都在官方仓库内。DimAgent 最应该复用这些公开 `ui` API，而不是复制 MyGo 的 `internal` 包。

但是，MyGo 不提供 Dim 专属的 session/timeline/streaming/approval/tool card 领域组件。它提供的是渲染和交互基础设施；这些领域组件应在项目自己的 `internal/ui` 和 `internal/projection` 中实现。

## 可直接优先利用的组件

| 能力 | MyGo 组件/入口 | DimAgent 用法 | 结论 |
|---|---|---|---|
| 基础布局和控件 | `ui` 基础 widgets/layout，见 [`ui/api_gen.go`](https://github.com/egoist/mygo/blob/main/ui/api_gen.go) | AppFrame、Sidebar、Toolbar、Panel、Button、Popover、Tabs | 直接复用 |
| 聊天消息列表 | [`ui.List` / `ListState`](https://github.com/egoist/mygo/blob/main/ui/list.go) | keyed rows、动态高度、`FollowEnd`、增量消息、sticky header、键盘选择 | 最关键，直接验证 |
| 大型网格/资源面板 | [`ui.GridView`](https://github.com/egoist/mygo/blob/main/ui/gridview.go) | model/provider/plugin 列表、图片/附件面板 | 按需复用 |
| 滚动 | `ui.Scroll` 及 List 内建滚动 | Conversation、代码块、设置页 | 直接复用 |
| 文本输入 | `ui.TextInput`、`ui.TextArea`、`ui.TextBuffer` | Composer、搜索、单行设置输入 | 直接复用，重点验证 IME |
| 原生编辑器 | [`ui/editor`](https://github.com/egoist/mygo/tree/main/ui/editor) | 代码编辑、diff、未来 workspace 编辑器 | 直接复用编辑器能力；接入 LSP 需自建 adapter |
| Markdown | [`ui/markdown`](https://github.com/egoist/mygo/blob/main/docs/ui/markdown.md) | assistant 消息、thinking、tool 输出、代码块、链接、图片 | 直接复用，再补 Dim mention/tool card hooks |
| 富文本 | [`ui/richtext`](https://github.com/egoist/mygo/tree/main/ui/richtext) | 若 Composer 需要富文本/格式化输入 | 暂不作为首片，先用 TextArea |
| 可访问性 | `ui/access*`、roles/relations/live regions | session list、message、approval、progress 状态 | 直接接入并做平台验收 |
| SVG / 图标 | `ParseSVG`、`Icon`，另有 Darwin `sfsymbols` plugin | toolbar、状态、工具类型、平台图标 | 复用；建立项目图标语义表 |
| 主题 | `ui/theme`、颜色/字体/spacing API | Dim design tokens、dark/light theme | 复用 API，tokens 自己定义 |
| 动画/反馈 | motion/transition、loading 等 plugin | streaming 状态、popover、toast、progress | 少量使用，尊重 reduced motion |
| 测试 | fake/headless/automation 测试设施 | View 行为、键盘、projection fixture、E2E | 先用 headless/fake；automation 仅测试环境 |

## 重点组件详情

### 1. `ui.List` 是聊天首选

源码文档明确支持：

- 只构建视口内行，适合长聊天记录；
- keyed row，消息新增或历史分页时保持行状态；
- 动态行高；
- `FollowEnd`，适合 streaming chat/log；用户滚离底部后停止自动跟随；
- selected/multi-selection、键盘导航、first-letter search；
- sticky section header；
- drag/reorder、accessibility label。

建议实现：

```text
ConversationView
  └─ ui.List(ListState{Key, FollowEnd, Label})
       └─ MessageRow / TurnRow
            └─ ui.Markdown / ToolCard / ApprovalCard
```

注意：`List` 解决的是虚拟化和滚动，不解决 TimelineFold、事件合并或消息语义。

### 2. Markdown 适合 assistant 内容，但不是完整聊天 renderer

官方 markdown view/options 覆盖 CommonMark、代码块、自定义链接、图片、mention、math、HTML、reveal 和 copy 等能力。它可以作为 assistant message 的基础 renderer。

需要项目自建：

- tool call/result 卡片；
- approval/question 卡片；
- streaming 增量更新策略；
- message actions；
- branch/source 标记；
- 安全的外部链接和图片策略；
- 与 TimelineFold 的 message block 映射。

不应为了方便把不可信 HTML 直接放进主 native tree；HTML/外部内容要经过明确策略，必要时隔离到 WebView。

### 3. Editor / RichText 的边界

`ui/editor` 已包含 Buffer、语法高亮、fold、多光标、snippet、diff/patch 相关能力，并有 code-editor example。它适合代码编辑器和未来的 workspace patch UI。

`ui/richtext` 提供 Document/Editor、IME、undo、Markdown/HTML/RTF 相关能力。它可能适合富文本 Composer，但输入法、粘贴、撤销和序列化复杂度较高，不应阻塞首个聊天闭环。

首片建议：

```text
Composer = ui.TextArea/TextBuffer
Assistant = ui.Markdown
Code block = ui.Markdown（必要时再升级为 editor）
```

### 4. Text 和 IME 是必须做 spike 的部分

MyGo 内建 TextBuffer/TextInput/TextArea，底层使用 `go-text/typesetting`（当前依赖 v0.3.5）。这对中英文、emoji、字体 fallback 和换行很有帮助，但不能仅凭 API 判断生产质量。

必须在 macOS、Windows、Linux 分别验证：

- 中文/日文/韩文 IME composition；
- composition 中的光标和候选词；
- emoji 和组合字符；
- 复制粘贴；
- Markdown 长行换行；
- 字体缺失 fallback；
- accessibility text exposure。

### 5. Chart 和 Video 属于后续能力

官方 `ui/chart` 支持 scales、Line/Area/Pie/Stack、selection、min/max 和 downsampling，可用于 usage、model progress 或运行统计，但不属于聊天首片。

官方 `ui/video` 通过平台后端支持播放，另外有 GIF/APNG 纯 Go 能力。附件/视频预览需要定义安全、权限和生命周期策略，不应进入第一阶段。

## 官方依赖与第三方库

MyGo 主模块当前直接依赖：

- `github.com/ebitengine/purego`：平台调用；
- `github.com/go-text/typesetting`：字体和文本排版；
- 音频：`go-mp3`、`oggvorbis`、`flac`、`go-aac`；
- `golang.org/x/image`、`x/net`、`x/text`。

Markdown、editor、chart 主要是 MyGo 自实现，不应在 DimAgent 中再引入第二套同类 UI 框架，除非 spike 证明官方组件不足。

## 需要自行实现的 Dim 组件

以下不是 MyGo 现成组件：

- SessionSidebar 的 session/catalog 业务投影；
- TimelineFold 和 history/live 合并；
- Turn/Message/Thinking/ToolCall/ToolResult 领域模型；
- ToolCard、ApprovalCard、QuestionCard；
- streaming 增量和 late event 处理；
- Branch/SideChat；
- model progress 与 capability gate；
- protocol v3 client、Supervisor、reconnect/resync；
- mention/file search/attachment 的 App Server 适配；
- LSP/backend adapter；
- Dim design tokens 和 icon semantics；
- 与 Electron 版本的 golden fixture/黑盒验收。

## 依赖策略

1. 优先使用 MyGo 公开 `ui` API 和官方 examples。
2. 不复制或依赖 `github.com/egoist/mygo/internal/...`；internal 只作为行为/实现研究参考。
3. 暂不引入 React-like 或其他 Go GUI toolkit，以免出现第二套布局、文本和事件模型。
4. 首片只引入 MyGo 核心模块；chart、video、richtext、automation、platform plugin 按 capability 延后。
5. automation plugin 只用于测试，不能在生产构建中开启；不把测试 socket 暴露给外部。

## 第一轮验证矩阵

| Spike | 验证内容 | 通过标准 |
|---|---|---|
| Native shell | window、layout、theme | macOS 可启动，窗口布局稳定 |
| Text | TextArea、TextBuffer、IME | 中英文/emoji composition 正确 |
| List | keyed dynamic rows、FollowEnd | streaming 增长无跳动，离底后不强拉 |
| Markdown | CommonMark、代码、链接、copy | fixture 与 Electron 视觉/语义可接受 |
| Accessibility | roles、labels、live regions | session/message/approval 可被辅助技术识别 |
| Editor | syntax/fold/diff | 代码块和 patch 不阻塞主聊天 |
| Headless | fake backend、键盘和状态 | 无 GUI 也能跑确定性测试 |

## 推荐采用顺序

```text
ui.List + ui.TextArea/TextBuffer
  → ui.Markdown
  → accessibility/theme/SVG
  → editor（代码/patch）
  → richtext（若 Composer 需要）
  → chart/video/plugins（按需求）
```
