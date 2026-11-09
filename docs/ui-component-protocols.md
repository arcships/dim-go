# internal/ui 组件协议

Status: active（纯契约提取：渲染结果、视觉和测试断言均未改动）
Last updated: 2026-11-09
适用范围：`internal/ui` 各区域之间、区域与 `shell.go` 之间的输入与动作合同。文件划分见 [ui-file-plan.md](ui-file-plan.md)，布局与设计 token 见 [ui-layout-spec.md](ui-layout-spec.md)，整体流程见 [workflow.md](workflow.md)。

## 1. 结构

- [`protocols.go`](../../internal/ui/protocols.go) 冻结合同：`Actions`、五个 View struct、`blockFolds`，以及每个区域的渲染入口。
- [`shell.go`](../../internal/ui/shell.go) 是唯一装配点：把 `State` 复制成 View struct，用 `State` 的回调构造 `Actions`，再按绘制顺序调用入口。它不画任何区域。
- 区域文件只读三样东西：自己的 View struct、`tokens`、`Actions`。不引用 `State`，不读 App Server/supervisor/projection/存储/日志，不改会话事实。

入口签名（`protocols.go` 中是薄包装，实现在对应区域文件）：

```go
func renderSidebar(c *ui.Context, k tokens, v SidebarView, a Actions)
func renderHeader(c *ui.Context, k tokens, v HeaderView, a Actions)
func renderTimeline(c *ui.Context, k tokens, v TimelineView, a Actions)
func renderApproval(c *ui.Context, k tokens, v ApprovalView, a Actions)
func renderComposer(c *ui.Context, k tokens, v ComposerView, a Actions)
```

包装函数是刻意的：区域可以改自己的 helper 和内部结构，但入口名、参数顺序和 `Actions` 的类型不能在区域文件里单方面改掉；要改先改 `protocols.go` 和本文档。

## 2. 区域合同

| 区域 | 入口 | 输入 | 调用方 | 可调用的 Actions |
|---|---|---|---|---|
| 侧栏 | `renderSidebar` | `SidebarView` | `shell.View`（`ui.Split` 左格） | `OnSelect`、`OnCreateSession` |
| 顶栏 + banner | `renderHeader` | `HeaderView` | `shell.View`（主列第一项） | `OnStop`、`OnReconnect` |
| 时间线 | `renderTimeline` | `TimelineView` | `shell.View`（主列第二项） | `OnApproval`（经审批行） |
| 审批行 | `renderApproval` | `ApprovalView` | `renderTimeline` 的按 author 分发 | `OnApproval` |
| 输入区 | `renderComposer` | `ComposerView` | `shell.View`（主列第三项） | `OnSubmit` |

### 2.1 侧栏 `SidebarView`

| 字段 | 类型 | 含义与不变量 |
|---|---|---|
| `Sessions` | `[]Session` | App Server catalog 的顺序；`Session.Group` 是窗口从 catalog 事实派生的展示分组，不发明项目名。全空时是一个平铺列表 |
| `Selected` | `*string` | 选中的 session id。指针是因为侧栏拥有“显示上的选择”：点击或方向键写入这里，再调 `OnSelect`；这是区域唯一会写的字段 |
| `SelectedBranch` | `string` | 观察到的当前 branch，只读展示；切换和 cut point 仍是 WIP |
| `Branches` | `[]Branch` | App Server 的只读 branch 元数据；为空时不画分支区 |
| `Creating` | `bool` | `session.create` 在途的窗口记录，不包含会话事实；为真时入口禁用，不能重复创建 |
| `Capabilities` | `[]wipCapability` | 只读 WIP 能力卡。shell 按已接线的 `Actions` 生成（`wipCapabilities`），区域原样绘制，不提供控件 |
| `state` | `*sidebarState` | 区域的列表位置、choice、上次映射的选中项和行布局；窗口生命周期内归 shell 持有，不含会话事实 |

调用方：`shell.View`。动作：选择行 → 写 `Selected` 再 `OnSelect(sessionID)`；“New session” → `OnCreateSession()`。

### 2.2 顶栏 `HeaderView`

| 字段 | 类型 | 含义与不变量 |
|---|---|---|
| `Title` | `string` | shell 已解析的选中 session 标题；没有选中时是 `No session`，区域不自己查 |
| `HasSession` | `bool` | 是否有选中 session；Stop / Reconnect 的前提 |
| `Connection` / `ConnectionError` | `Connection` / `string` | supervisor 最后一次报告及最后错误，纯展示 |
| `Run` | `RunState` | supervisor 报告的会话状态；不从行内容推断 |
| `Closed` / `ClosedReason` | `bool` / `string` | 终态 `session.closed`；优先于 run、连接错误和 action error，且关闭后不提供 submit/stop/reconnect |
| `ModelProgress` | `*ModelProgress` | 瞬时、非持久的模型预览 |
| `Notices` / `ActionNotices` | `[]Notice` | projection 通知与窗口自己的动作结果；状态行只显示总数 |
| `ActionError` | `string` | 窗口自己动作的失败。只在没有终态关闭和连接错误时显示；失败就是失败，不显示成成功 |
| `ActionStatus` | `string` | App Server 已确认、但没有会话事实可展示的结果（如回读确认的提交） |
| `Loading` | `bool` | 历史或重连在途；隐藏 Reconnect，避免同一个动作发起两次 |

调用方：`shell.View`。动作：Stop → `OnStop()`；Reconnect → `OnReconnect()`；只在 `HasSession && !Closed` 且各自条件满足时出现。banner 由同一区域画在工具栏下方。

### 2.3 时间线 `TimelineView`

| 字段 | 类型 | 含义与不变量 |
|---|---|---|
| `Messages` | `[]Message` | projection 顺序的行；行是展示副本，identity/流式/工具状态归 projection |
| `Loading` | `bool` | 历史仍在加载；为真时只画 loading 状态，不画行 |
| `Closed` | `bool` | 终态 `session.closed`。区域不因此增删行（行是 projection 事实，历史仍可读）；当前没有分支逻辑，留给将来需要停止“live tail”语义时使用 |
| `list` | `*ui.ListState` | 会话列表的滚动位置和按行 identity 的状态，窗口生命周期内保留 |
| `folds` | `*blockFolds` | thinking / tool block 的折叠状态，按 `blockKey(row, block)` 索引；属于窗口视图状态，不是 timeline 事实 |
| `markdown` | `*markdown.Options` | 所有 assistant 行共享的 Markdown 选项，跟随窗口主题 |

调用方：`shell.View`。动作：只有审批行会调 `OnApproval`。

### 2.4 审批行 `ApprovalView`

| 字段 | 类型 | 含义与不变量 |
|---|---|---|
| `Text` | `string` | transient projection 给的行文本；区域不补原因 |
| `Pending` | `[]Approval` | 该行仍待决的请求。当前 projection 一行一个请求；没有请求或请求缺 App Server identity 时，两个决策保持可见、禁用并说明原因 |

调用方：`renderTimeline` 的按 author 分发（`AuthorApproval`）。动作：每个可寻址请求给出 Approve/Deny 各一个按钮，分别调 `OnApproval(branchID, requestID, allow/deny)`；行本身只在之后的 projection 帧里消失。

### 2.5 输入区 `ComposerView`

| 字段 | 类型 | 含义与不变量 |
|---|---|---|
| `Draft` | `*string` | 就地编辑的草稿。提交成功后由窗口清空；失败保留文本 |
| `CanSubmit` | `bool` | 窗口的提交闸门：已接线的提交回调、ready 连接、历史已加载、会话未关闭且有选中 session。**不含草稿文本**：区域在点击/快捷键时实时读取草稿，草稿为空或全空白就不发 |
| `Sending` | `bool` | 提交在途的窗口记录。当前没有事实来源，shell 恒传 `false`；为真时只禁用 Send，不改变其他内容 |
| `Placeholder` | `string` | 空草稿时的占位文本 |
| `Shortcut` | `KeyBinding` | 提交草稿的修饰键 + 键（当前 `ComposerShortcutMods` + Enter）。shell 自己的聚焦快捷键不属于该 View |
| `Focus` | `*ui.Handle` | composer 的身份，供 shell 的聚焦快捷键使用 |

调用方：`shell.View`。动作：Send 或提交快捷键 → `OnSubmit(trimmed draft)`。

## 3. 不变量

1. **App Server 是唯一事实源。** View 里只有展示副本和窗口自己的视图状态。区域不读 `internal/appserver`、`internal/supervisor`、`internal/projection`、SQLite/JSONL/session 文件或 `daemon.log`，也不建第二份 session/run/approval 存储。
2. **点击是请求，不是结果。** 区域最多调用一个 `Actions` 字段一次，自己不修改会话事实，也不移除审批行。成功由后续 projection 帧体现；失败由窗口写进 `HeaderView.ActionError` / `ActionStatus`。不允许任何区域把失败画成成功。
3. **WIP 入口保留。** `Actions` 字段为 nil 表示窗口没接线：入口仍然可见、禁用、用 `wipAction` + reason 说明缺的能力，并且不注册点击。不隐藏、不伪造。
4. **视图状态的归属。** `Selected *string`、`sidebarState`、`blockFolds`、`ui.ListState`、`markdown.Options` 和 `ui.Handle` 都是窗口/视图状态，放在 `State` 里由窗口保活，通过 View 以字段或指针传入区域。它们永远不是会话事实，也不参与投影。
5. **区域不反向引用 `State`。** 唯一的例外是 `capability.go` 的 `wipCapabilities(s *State)`：它把“哪些回调已接线”映射成只读能力卡，是 shell 的装配输入而不是渲染器；`capability_test.go` 直接以 `*State` 调用它，所以签名保持不动。区域渲染函数本身不接收 `State`。
6. **颜色、间距、字号只来自 `tokens`。** 区域不得硬编码视觉常量；新增 token 属于共享合同（见 §4）。
7. **入口包装不可绕过。** `shell.go` 只调 `renderX`；区域之间不互相调用，审批行是唯一例外（时间线把 `AuthorApproval` 行交给 `renderApproval`，因为审批行的位置由时间线决定）。

## 4. 并行 worktree 规则

- **写权限按文件划分，不按包划分。** 一个任务只写自己的区域文件及其测试文件；同包不等于可以顺手改别人的文件。
- **共享合同文件单一写者：** `protocols.go`（合同）与 `shell.go`（装配）由同一个 owner 串行落地。任何改 View 字段、入口签名、`Actions` 语义或点击行为的任务，必须先在这个 owner 的分支上改 `protocols.go` + 本文档，再分派给区域。
- **`tokens.go` 是共享的：** 新增 token 或改 `tokens` 结构会影响所有区域，先落地再分派，不在区域分支里顺手加。
- **区域内部自由改动：** 区域文件里的 helper、折行、注释和小工具函数可以随便改，只要不动合同、不动视觉，并保持自己的测试断言不变。
- **行为改动单独提交：** 协议提取只搬输入和动作边界，不改行为、不改断言。真正改变行为（显示条件、点击结果、视觉）必须单独 PR 并补对应测试，同时更新本文档的字段表。
- **合并前至少跑：** `gofmt -l .`（无输出）、`go build ./...`、`go vet ./...`、`go test -count=1 ./...`。
- **冲突处理：** 两个分支同时改了同一个区域文件即视为冲突，由 leader 合并或串行化；不要用“同包会自动合并”当理由。

## 5. 本次验证（2026-11-09）

```bash
gofmt -l .                 # 无输出
go build ./...
go vet ./...
go test -count=1 ./internal/ui
go test -count=1 ./...
```

结果：全部通过；8 个包测试通过。

视觉零变化证据：在 `HEAD` 和本次工作树上分别用 `DIMAGENT_UI_SCREENSHOT_DIR` 生成 `shell-light.png` / `shell-dark.png`（1440×900），四个 PNG 逐字节相同（light `4d4f74…`，dark `e8c850…`）。测试文件未改动。

验证等级：`headless` 测试级，不涉及真实窗口（见 [acceptance.md](acceptance.md)）。

## 6. 相关文档与待同步

- [`ui-file-plan.md`](ui-file-plan.md) 的区域表记录的是拆分时的函数名（`main`、`branchList` 在 `shell.go`）；契约提取后 `main` 并入 `View`、`branchList` 移到侧栏区域、新增 `protocols.go`。该表的同步是纯文档改动，留给 owner 统一处理。
- 新文件需要在 [README](../../README.md) 的“参考”列表加一条入口（本次未改，避免与并行任务冲突）。
