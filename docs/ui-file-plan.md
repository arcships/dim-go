# internal/ui 文件规划

Status: active（文件划分记录；§2 已补入后续新增文件，§4 的行数是拆分当时的快照）
Last updated: 2026-11-09
适用范围：`internal/ui` 的源码文件划分与并行写权限边界。流程与角色见 [workflow.md](workflow.md)，布局与组件约定见 [ui-layout-spec.md](ui-layout-spec.md)。

## 1. 拆分原则

1. **一个区域一个文件**：同一个界面区域的渲染函数、它自己的小工具函数放在同一个文件里；文件名就是区域名。
2. **纯移动**：拆分只搬函数，不改实现、不改注释文字、不改测试断言。同包内移动不改变可见性，因此没有导出面变化。
3. **类型留在 `shell.go`**：`State` 和视图模型类型（`Session`、`Branch`、`Author`、`Message`、`Approval`、`Notice`、`ModelProgress`）是被所有区域共享的合同，集中在 `shell.go`，区域文件只读不定义。
4. **`shell.go` 保持精简**：只保留包文档、`State`、`View`、主布局 `main` 和跨区域共享的常量/类型。
5. **文件头写职责**：每个文件顶部一句注释说明它拥有哪个区域，方便按文件分派任务。

## 2. 区域 → 文件 → 测试 → 所有权

| 区域 | 文件 | 职责 | 测试文件 | owner |
|---|---|---|---|---|
| 壳与布局 | [shell.go](../internal/ui/shell.go) | 包文档、`State`、视图模型类型、`View`、主布局 `main` | `shell_test.go`、`screenshot_test.go`、`follow_end_test.go` | `ui-shell` |
| 顶栏 | [header.go](../internal/ui/header.go) | `header`、`runStatus`、`runLabel`、`status`、`banner`、`selectedTitle` | `status_test.go`、`run_test.go` | `ui-header` |
| 会话区 | [conversation.go](../internal/ui/conversation.go) | `conversationList`、`loadingConversation`、`emptyConversation`、`messageRow` 按 author 分发 | `conversation_test.go` | `ui-conversation` |
| 审批面板 | [approval_panel.go](../internal/ui/approval_panel.go) | `renderApprovalPanel`：composer 槽位的待决请求卡片（call/reason/source/input variant + 决策）；审批行已并入面板，原 `approval.go` 在 `ffa54c7` 删除 | `approval_test.go`、`approval_panel_test.go` | `ui-approval` |
| 输入区 | [composer.go](../internal/ui/composer.go) | `composer`：自适应多行、Enter 发送 / Shift+Enter 换行、提示行、工具行与底部 context row；`canSubmit`、`submitDraft`、`stopRun`、`reconnect` | `composer_p0_test.go`、`actions_test.go`、`wip_test.go`、`run_test.go` | `ui-composer` |
| 会话动作 | [actions.go](../internal/ui/actions.go) | `createSession`、`selectSession`、`selectBranch`、`endBranch`、`refreshBranches`、`toggleKeys` | `create_test.go`、`actions_test.go` | `ui-actions` |
| 侧栏 | [sidebar.go](../internal/ui/sidebar.go) | `sidebar`、线程列表、本地搜索、键盘选择、`chooseSession`、`branchList`、侧栏页脚 | `sidebar_test.go`、`keyboard_test.go` | `ui-sidebar` |
| 时间线 | [timeline.go](../internal/ui/timeline.go) | 行渲染（user/assistant/error/retry）、block、文件变更、`blockOpen` | `timeline_test.go` | `ui-timeline` |
| 工具卡 | [tool.go](../internal/ui/tool.go) | `toolRow`、`toolCard`、disclosure 与状态 | `tool_test.go` | `ui-tool` |
| 工具面板 | [toolrender.go](../internal/ui/toolrender.go) | `toolKindOf`、command/file/search/generic 面板，缺失事实不伪造 | `toolrender_test.go` | `ui-tool` |
| Markdown 内容层 | [markdown_render.go](../internal/ui/markdown_render.go) | `MarkdownOptions`、`renderMarkdownContent`、代码块头与 Copy、内容层 token | `markdown_render_test.go` | `ui-markdown` |
| 上下文面板 | [context_panel.go](../internal/ui/context_panel.go) | `ContextPanelView`、文件树、diff/预览、空态与 WIP；已作为右侧 context panel 绘制并可由工具栏切换，仍无 changed-file 数据源 | `context_panel_test.go` | `ui-context` |
| Keys 面板 | [keys_panel.go](../internal/ui/keys_panel.go) | `KeysPanelView`、本地过滤、只读行、未加载 WIP | `keys_panel_test.go` | `ui-keys` |
| 设计 token | [tokens.go](../internal/ui/tokens.go) | 颜色/间距/字号 token、`newTokens`、`k.*` 取色辅助 | `tokens_test.go` | `ui-tokens` |
| 消息层 | [i18n.go](../internal/ui/i18n.go) | `Locale`、en/zh-CN 消息表、`tr`、`resolveLocale` | `i18n_test.go` | `ui-i18n` |
| 图标 | [icons.go](../internal/ui/icons.go) | 侧栏与工具卡片使用的矢量 glyph | `icons_test.go` | `ui-icons` |
| 区域合同 | [protocols.go](../internal/ui/protocols.go) | `Actions`、各区域 View struct、渲染入口包装 | 由各区域测试覆盖 | `ui-shell` |
| WIP 入口 | [wip.go](../internal/ui/wip.go) | `wipAction`、WIP 徽标与 reason 文本 | `wip_test.go` | `ui-wip` |
| 外观设置 | [settings.go](../internal/ui/settings.go) | `SettingsView`、`renderSettings`、17 分区导航、主题/字号/字体/宽度/密度/代码字号/语言；skin/chrome WIP | `settings_test.go` | `ui-settings` |
| 状态栏 | [statusbar.go](../internal/ui/statusbar.go) | `StatusBarView`、`renderStatusBar`、context 用量条、窄宽度省略 | `statusbar_test.go` | `ui-statusbar` |
| 能力卡 | [capability.go](../internal/ui/capability.go) | `wipCapabilities`、`capabilityCards`、`capabilityCard` | `capability_test.go` | `ui-capability` |
| Run 状态 | [run.go](../internal/ui/run.go) | `RunState` 及其 `String` | `run_test.go` | `ui-run` |
| 连接状态 | [connection.go](../internal/ui/connection.go) | `Connection` 及其 `String` | `status_test.go` | `ui-connection` |

## 3. 所有权与并行边界

- **同一文件同一时间单一写者**。并行任务的 worktree 以文件为单位划分，不以包为单位；两个任务同时改同一文件即视为冲突，必须由 leader 合并或串行化。
- 一个任务的写权限 = 表内对应的实现文件 + 该行列出的测试文件。不得顺带修改其他区域的文件。
- `shell.go` 是共享类型文件：新增或修改 `State` 字段属于跨区域改动，由 leader 统一收口；区域文件只读 `State` 字段。
- `tokens.go` 同理：新增 token 或改 `tokens` 结构会影响所有区域，需 leader 先行落地再分派。
- 跨文件改动（改函数签名、改共享字段、改 `wipAction` 的调用约定）无法靠分文件隔离，因为同包共享 `State` 和 `tokens`；此时按“先改 `shell.go`/`tokens.go`，再按文件分派”的顺序串行。
- 本规划只解决写冲突，不承诺行为边界：拆分不改变任何行为、视觉或断言；行为改动单独提交并补对应测试。

## 4. 拆分前后行数

只统计 `internal/ui` 下的非测试源码。下表是**拆分当时（2026-11-09 早段）的快照**，之后新增的 `i18n.go`、`icons.go`、`protocols.go`、`keys_panel.go`、`settings.go`、`statusbar.go` 等没有回填；当前文件与行数以 `ls internal/ui/*.go | grep -v _test` 和 `wc -l` 为准。

| 文件 | 拆分前 | 拆分后 | 说明 |
|---|---|---|---|
| `shell.go` | 590 | 302 | 只保留包文档、类型、`State`、`View`、`main` |
| `header.go` | — | 125 | 新增 |
| `conversation.go` | — | 76 | 新增 |
| `composer.go` | — | 63 | 新增 |
| `approval.go` | — | 36 | 新增；已在 `ffa54c7` 删除，审批行并入 `approval_panel.go` |
| `actions.go` | — | 24 | 新增 |
| `sidebar.go` | 314 | 314 | 未动；后续接管 `branchList` 并扩展到搜索/分支/页脚 |
| `timeline.go` | 345 | 345 | 未动 |
| `tokens.go` | 498 | 498 | 未动 |
| `tool.go` | 192 | 192 | 未动 |
| `capability.go` | 98 | 98 | 未动 |
| `wip.go` | 46 | 46 | 未动 |
| `connection.go` | 36 | 36 | 未动 |
| `run.go` | 30 | 30 | 未动 |
| 合计 | 2149 | 2149 + 36（新文件的 package/import/职责注释开销） | — |

## 5. 待收敛项

- `branchList` 已从 `shell.go` 搬到 `sidebar.go`，与侧栏语义一致。
- 视图模型类型集中在 `shell.go`；如果将来类型继续变大，可以再拆 `model.go`（同样属于纯移动），但不要按区域拆，避免各区域互相 import 式的重定义。

## 6. 验证

拆分后在工作树执行，全部通过：

```bash
gofmt -l .                 # 无输出
go build ./...
go vet ./...
go test ./internal/ui
go test -count=1 ./...
```

验证等级：`headless` 测试级，不涉及真实窗口（见 [acceptance.md](acceptance.md)）。
