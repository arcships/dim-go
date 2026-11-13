# U4 技术设计

Status: 现行（v1；C4 定稿，2026-11-13）
Owner: U4 技术设计（界面领域）
依据: `docs/project-organization.md` §9.0（A0–A4、D0–D6）、§9.8（X1–X22）、§9.9（UI-1…UI-23）、§9.14（0–5 层）、§9.15（变更通道、v1 冻结）；`docs/domains/U1-ia.md`、`U2-forms.md`、`U2-elements.json`、`U3-interaction.md`、`U0-framework-baseline.md`
范围: 界面怎么落到 view 上——映射、分层、预算、虚拟化单位、Markdown 路线、键位/焦点/读屏映射、校正清单。
不做: 不新增形式（在 U2），不新增交互规则（在 U3），不改公理与判定。

判据（§9.14 第 1 层）：**每个可见元素都能指到一个组件、每份状态都有一处权威来源、每一帧都有上界**。

## 1 view 映射表

单一来源是 [`U2-elements.json`](U2-elements.json)（33 条，含 `component` 与 `symbol`）；本表由 `scripts/render-lists.py` 渲染。覆盖率由检查项 **L19** 守：每条元素的 `component`/`symbol` 非空，且 `symbol` 是 `internal/ui` 里真实存在的标识符——指不到即缺，缺了就红。

<!-- BEGIN:items -->
| # | 可见元素 | 组件（框架 / 自绘） | 代码符号 | 出现的态 |
|---|---|---|---|---|
| EL-01 | 左栏入口 · 新建 | gui.ButtonBase + 自绘图标 | `iconButton` | `always` |
| EL-02 | 左栏入口 · 搜索 | gui.ButtonBase + 自绘图标 | `iconButton` | `always` |
| EL-03 | 左栏入口 · 收起左栏 | gui.ButtonBase + 自绘图标 | `iconButton` | `always` |
| EL-04 | 连接状态 | 自绘 Box + DrawOver 圆 | `statusDot` | `always` |
| EL-05 | 账号 | 自绘 Box + DrawOver 首字母 | `avatar` | `always` |
| EL-06 | 标题行会话名 | 自绘 Row + gui.Text | `header` | `always` |
| EL-07 | 右栏入口 | gui.ButtonBase + 自绘图标 | `iconMenuButton` | `always` |
| EL-08 | 分支标记 | gui.ButtonBase + PopoverMenuButton（原生菜单） | `branchChip` | `branch` |
| EL-09 | 轮次组头 | CollapsibleBase + 自绘行 | `roundHeader` | `always` |
| EL-10 | 过程条目 | 自绘行 | `opRow` | `always` |
| EL-11 | 结果不确定 | 自绘 Row + iconGlyph | `note` | `failed` |
| EL-12 | 审批决定 · 允许一次 | 自绘 ButtonBase（textIconButton） | `textIconButton` | `waiting` |
| EL-13 | 审批决定 · 一直允许 | 自绘 ButtonBase（textIconButton） | `textIconButton` | `waiting` |
| EL-14 | 审批决定 · 拒绝 | 自绘 ButtonBase（textIconButton） | `textIconButton` | `waiting` |
| EL-15 | 队列条 · 撤回 | gui.ButtonBase + 自绘图标 | `iconButton` | `running` |
| EL-16 | 上下文用量 | CollapsibleBase + 自绘行 | `contextMeter` | `always` |
| EL-17 | 附件入口 | gui.ButtonBase + 自绘图标 | `iconButton` | `always` |
| EL-18 | 发送槽（发送） | 自绘 ButtonBase（iconAction） | `iconAction` | `always` |
| EL-19 | 发送槽（停止） | 自绘 ButtonBase（iconAction） | `iconAction` | `running` |
| EL-20 | 读取更早 | gui.ButtonBase + 自绘图标 | `iconButton` | `always` |
| EL-21 | 空态 | 自绘 Row + iconGlyph | `timeline` | `empty` |
| EL-22 | 目标行 | 自绘 Row + TextAreaBase | `goalRow` | `always` |
| EL-23 | 目标行 · 修改入口 | gui.ButtonBase + 自绘图标 | `iconButton` | `always` |
| EL-24 | 目标行 · 编辑态保存 | gui.PrimaryButton | `actionButton` | `goal-edit` |
| EL-25 | 后台任务行 | 自绘 Row + iconGlyph | `taskRow` | `tasks` |
| EL-26 | 后台任务 · 取消 | gui.ButtonBase + 自绘图标 | `iconButton` | `tasks` |
| EL-27 | 用户消息 | 自绘 Box（气泡） | `round` | `always` |
| EL-28 | 助手正文 | ui/markdown.View | `markdown.View` | `always` |
| EL-29 | 设置入口 | gui.ButtonBase + 自绘图标 | `iconButton` | `always` |
| EL-30 | 设置分节导航 | gui.ButtonBase（自绘行） | `navRow` | `settings` |
| EL-31 | 设置项（标签 + 控件） | gui.Select / gui.Switch | `settingsRow` | `settings` |
| EL-32 | 危险动作 · 清空命令规则 | 自绘 ButtonBase（textIconButton） | `textIconButton` | `settings-rules` |
| EL-33 | 目标行 · 编辑态取消 | gui.Button | `actionButton` | `goal-edit` |
| EL-34 | 扩展分节 | 自绘行（ButtonBase） | `navRow` | `settings-ext` |
| EL-35 | 扩展页签 | gui.Tabs | `settingsExtensions` | `settings-ext` |
| EL-36 | 扩展列表行 | 自绘行 + gui.Switch | `extRow` | `settings-ext` |
| EL-37 | 扩展空态 | 自绘行 + iconGlyph | `settingsExtensions` | `settings-ext-empty` |
| EL-38 | 扩展读取失败 | 自绘行 + iconGlyph + iconButton | `settingsExtensions` | `settings-ext-failed` |
| EL-39 | 终端表面 | 右栏标签（gui.Tabs 第五项） | `rightPanel` | `term` |
| EL-40 | 终端状态点 | 自绘 Box + DrawOver 点 | `terminalPane` | `term` |
| EL-41 | 终端输出区 | gui.Scroll + 等宽 Text | `terminalPane` | `term-exited` |
| EL-42 | 终端中止入口 | 自绘图标入口 | `iconButton` | `term-running` |
| EL-43 | 终端清屏入口 | 自绘图标入口 | `iconButton` | `term` |
<!-- END:items -->

**形式库的 24 条**落在这些组件上（一条形式可能对应多个元素）：

| 形式 | 组件 | 元素 |
|---|---|---|
| F1 会话身份 | `header` + 框架 `Sidebar` | EL-06 |
| F2 轮次组头 | `CollapsibleBase` + `roundHeader` | EL-09 |
| F3 过程条目 | `opRow` | EL-10 |
| F4 轮次状态 | `statusMark`（自绘 DrawOver） | 组头行首 |
| F5/F6/F18 审批与提问 | `approvalCard` + `textIconButton` | EL-12…14 |
| F7 结果不确定 | `note` + `iconGlyph` | EL-11 |
| F8 队列 | `queueBar` + `iconButton` | EL-15 |
| F9 附件 chip | `actionButton` 之外的 `Box`（控件的 chip） | — |
| F10 上下文用量 | `contextMeter` | EL-16 |
| F11 连接 / F12 账号 | `statusDot` / `avatar` | EL-04 / EL-05 |
| F13 对象引用 | 等宽 `RichText` | 组头/条目内 |
| F14 空态 | `timeline` 的空态行 | EL-21 |
| F15 错误 | `note` | 同 EL-11 |
| F16 图标入口 | `iconButton` / `iconMenuButton` | EL-01…03、07、15、17、20、23、26、29 |
| F17 发送 / 停止 | `iconAction` | EL-18/19 |
| F19 分支 | `branchChip`（ButtonBase + `PopoverMenuButton`） | EL-08 |
| F20 目标 | `goalRow`（含 `actionButton` 的保存/取消） | EL-22/23/24/33 |
| F21 后台任务 | `taskRow` | EL-25/26 |
| F22 设置导航 | `settingsNav` + `navRow` | EL-30 |
| F23 设置项 | `settingsRow` + `gui.Select`/`gui.Switch` | EL-31 |
| F24 危险动作 | `textIconButton` + `actionButton` 之外的确认态 | EL-32 |

**抽查三条（元素 → 代码）**：
- EL-08 分支标记 → `internal/ui/header.go` 的 `branchChip`（`PopoverMenuButton` 承载原生菜单，确认态在同一位置）。
- EL-16 上下文用量 → `internal/ui/composer.go` 的 `contextMeter`（`CollapsibleBase`，展开走 `contextDetail`）。
- EL-28 助手正文 → `internal/ui/timeline.go` 的 `reply` 调 `markdown.View`（流式用 `markdown.Stream`）。

## 2 分层与状态归属

三层，状态只能从上往下流：

```
D 领域事实（App Server，经 protocol v3）
      │  只读快照 + 事件增量
      ▼
可丢的视图状态（App 里的字段；退出即失，不落盘）
      │  每帧读出
      ▼
view 树（框架元素；每帧重建，框架自己缓存布局）
```

**允许存在 view 里的（UI-8，全部可丢）**：

| 类别 | 字段 | 说明 |
|---|---|---|
| 草稿 | `draft` | 未提交的输入；切屏、切会话不丢（本片不切会话） |
| 展开态 | `summaryOpen map[int]*bool`、`contextOpen` | 按轮次索引；轮次变多不重建数组 |
| 焦点 | 框架持有（`c.Focus`），view 只声明可聚焦 | 不另存焦点字段 |
| 滚动 | `feed gui.ScrollState` | 框架滚动容器持有 |
| 列宽 / 面板开合 | `sidebarCollapsed`、`rightOpen`、`rightTab` | 屏幕档位与用户的开关 |
| 屏与分节 | `screen`、`settingsSection` | 设置屏是独立屏（P2） |
| 本地偏好 | `themePref`、`motionOff` | 主题与动效；只影响渲染 |
| 瞬态确认 | `confirmAlways`、`clearRulesConfirm`、`endBranchConfirm`、`approvalDecision` | 两步确认的中间态 |
| 走查变体 | `variant` | 只用于截图与种子，不参与业务 |

**必须来自事实的（view 不许攒第二份）**：会话与标题、分支清单与当前分支、轮次与操作、审批与决定、目标文本与完成态、后台任务与状态、命令规则、上下文用量、连接态、供应商与模型。

**落地约定**：
1. view 只按事实的「索引 / id」记视图状态（如 `summaryOpen[i]`），不复制事实文本。
2. 任何需要"记住"的事实都回 App Server 取；本地只有一个可丢的缓存（`sample.go` 的样例、后续接入后的只读快照）。
3. 断言：`TestPerfBudgets` 的 200 轮样例只改 `rounds`（事实侧）而不改任何视图字段，界面仍能画——视图状态不依赖事实数量。

## 3 性能预算

| 项 | 预算 | 实测（Apple M4 Max，1x，`ui.Tester`） | 测法 |
|---|---|---|---|
| 首帧（建窗口 + 第一帧） | ≤ 250 ms | **34.1 ms** | `TestPerfBudgets`：`NewApp` + `gui.NewTester` 计时 |
| 稳态帧 | ≤ 16 ms（60 fps 余量） | **0.16 ms** | 连画 20 帧取中位数 |
| 长列表（200 轮） | ≤ 60 ms | **2.96 ms** | 200 轮样例，连画 10 帧取中位数 |
| 常驻堆（200 轮） | ≤ 96 MB | **2.3 MB** | `TestPerfMemory`：GC 后 `HeapAlloc` |

可复跑命令：

```bash
go test ./internal/ui -run 'TestPerf' -v                     # 断言预算并打印实测
go test ./internal/ui -run '^$' -bench . -benchtime 20x -benchmem
```

基准输出（同一台机器，20 次）：

```
BenchmarkFirstFrame-16      20   4096398 ns/op   6039563 B/op   589 allocs/op
BenchmarkSteadyFrame-16     20    159031 ns/op      3681 B/op    88 allocs/op
BenchmarkLongFeed200-16     20   1662125 ns/op    144981 B/op  3465 allocs/op
```

读法：首帧 4.1 ms 主要是首次字体与主题初始化；稳态 159 µs；200 轮 1.66 ms，分配随轮次线性（每轮约 17 次分配，来自组头与消息节点）。**没有做虚拟化**，200 轮仍远在预算内；到 1000 轮需要（见 §4 的触发条件）。

## 4 虚拟化单位

**结论：以「轮次一行」为单位虚拟化**（一轮 = 一个 `round` 块：用户消息 + 回复 + 组头 + 展开内容）。

| 候选 | 代价 | 结论 |
|---|---|---|
| 一条消息一行 | 轮内元素被拆开，展开态跨两个虚拟单位，滚动定位难对齐 | 否 |
| **轮次一行** | 展开后单行高度大（几十到几百 DIP），滚动时可见行数少，估算高度误差集中在一轮内 | **选它** |
| 块一行（markdown 段落级） | 高度最均匀，但要先把 Markdown 解析成块并暴露给虚拟化层，等于自建布局层 | 否（与 §5 复用 `ui/markdown` 冲突） |

依据：U0 §2.8 要求"行高必须可测"——轮次块的高度由 `gui.Measure` 在展开/折叠两态各测一次即可（组头 24 + 展开内容），不需要逐块测量。

**触发条件与代价**：≤ 400 轮不虚拟化（实测 200 轮 1.66 ms/帧，400 轮仍 < 4 ms）；> 400 轮启用，需要事实层给"按游标取一页轮次"（`session.events` 的 range 查询），否则虚拟化拿不到稳定 id。**本批不实现**，列在 §7 校正清单。

## 5 Markdown 路线

| 候选 | 能力 | 依赖 | 体积 | 流式 | 结论 |
|---|---|---|---|---|---|
| **mygo 内建 `ui/markdown`** | CommonMark + 表格 / 代码块（带复制）/ 链接菜单 / 图片 / 数学 / 脚注 / alert | 宿主框架自带（`github.com/egoist/mygo/ui/markdown`），**零新增** | 0（已在框架内） | 有：`markdown.Stream` 逐字增量 + `View` 完成后接管 | **选它** |
| `markstream-go` | 面向流式增量解析 | 新增第三方（AGENTS 里已登记为 `internal/ui/render` 的例外） | 新增一个模块 + 传递依赖 | 有 | 暂不引入 |
| 自写渲染 | 完全可控 | 零第三方 | 自己维护 | 要自己写 | 否（维护成本） |

**结论**：v1 用 **mygo 内建 `ui/markdown`**——零新增依赖（符合"第三方默认禁止"）、有 `Stream`/`View` 两态、与框架主题同一套 token。**不引入 `markstream-go`**，理由：内建已覆盖首片所需能力，引入第二个 Markdown 实现会让"同一段文字两套排版"成为长期负担；若将来需要它的增量解析优势（超长回答 + 低功耗设备），走 `docs/domains/A.md` 的设计选择再引入，届时只需替换 `reply()` 里的一次调用。

**实现**：`internal/ui/timeline.go` 的 `reply()`——完成的回复 `markdown.View(c, r.reply, &a.md)`，正在写的走 `markdown.Stream`（本轮样例只有完成态，`Stream` 的接入点在同一个函数里）。`a.md` 是 `markdown.Options` 的一份实例，跨帧复用（解析缓存随它）。

## 6 键盘、焦点、读屏到 view 的映射

| U3 编号 | 键 / 行为 | view 上的落点 | 代码 |
|---|---|---|---|
| K1 发送 / 停止 | Enter | 输入区 `TextAreaBase` 的 `Shortcut`；槽位 `iconAction` 同动作 | `composer.go` `sendSlot`、`app.go` keyset |
| K2/K3 排队 / 插话 | `Cmd+Enter`、`Cmd+Shift+Enter` | 同上，提交参数不同 | 同上（本片只画槽位） |
| K4/K5 换行 | `Shift+Enter`、`Ctrl+J` | 输入区自身 | 框架 `TextAreaBase` |
| K6/K7/K8 允许 / 始终允许 / 拒绝 | Y / A / N | 审批卡的三个 `textIconButton` | `composer.go` `approvalCard` |
| K9 提交回答 | `Cmd+Enter` | 提问卡（本片未实现提问卡） | — |
| K10 展开 / 收起 | Enter / Space | 组头 `CollapsibleBase.Trigger`（`Focusable`，框架给 Enter/Space） | `timeline.go` `roundHeader` |
| K11 打开会话 | Enter | 左栏 `SidebarItem` | `sidebar.go` |
| K12 区内移动 | ↑ / ↓ | 左栏列表 / 时间线（框架列表与滚动容器） | 框架 |
| K13 焦点进入区 | Tab | 元素树顺序：左栏入口 → 会话行 → 标题行 → 时间线 → 输入区 → 右栏 | 各元素 `Focusable` |
| K14 关闭覆盖层 | Esc | 菜单 / 下拉（框架） | 框架 |

**焦点**：view 不存焦点字段；可聚焦性由元素声明（`Focusable`），焦点环由框架画（`c.FocusRing`）。自定义入口统一用 `ButtonBase`（自带 Enter/Space 与焦点环），不自造键盘处理。

**读屏**：每个可见元素有 `Label`（可读名）；装饰性图标用 `AccessHidden(true)`（如 `iconGlyph` 的「图标：<mark>」、方向指示）。组头的可读名 = 「状态：… + 内容短语」（`TestRoundHeaderAriaNameCarriesStatus`），屏幕上不出现状态词（L6）。

**Z 层**：只有原生菜单与下拉弹层（框架持有），没有自造浮层；`layout` 的 `gui.Row/Column` 顺序即层序。

## 7 实现校正清单

C4 定稿后逐条对照实现，标出偏差与处理（形式级 / 控件级）：

| # | 偏差 | 级别 | 处理 | 证据 |
|---|---|---|---|---|
| 1 | 助手正文用 `gui.Text` 画，未走 Markdown（U1 B2、§5） | 控件级 | **已改**：`reply()` 改用 `markdown.View`，流式接入点 `markdown.Stream` | `timeline.go` `reply`；截图无回归，`go test` 全绿 |
| 2 | 展开态是 `[]bool`，按样例轮次数建数组；200 轮时索引越界 panic | 控件级 | **已改**：`summaryOpen map[int]*bool` + `openOf(i)` 按需创建；视图状态不再随事实数量重建 | `TestPerfBudgets` 的 200 轮样例可画 |
| 3 | 长列表无虚拟化（§4 定了"轮次一行"） | 控件级 | **本批不改**：实测 200 轮 1.66 ms/帧、预算 60 ms；触发条件是 > 400 轮且事实层能按游标分页，列入下一批 | `BenchmarkLongFeed200` |
| 4 | 设置屏断线只读只有结构，没有断线事实源 | 控件级 | **本批不改**：等 D1 接入后按 U3 §4.2 的回执规则补 | — |
| 5 | 提问卡（F6、K9）未实现 | 形式级（缺一形式） | **本批不改**：U1 D 组只要求审批；提问卡列在后续批次 | — |
| 6 | 分支菜单「从当前分支切出」单步，无确认 | 控件级 | **符合清单**：AC-12 判为低风险单步，不改 | `U3-actions.json` AC-12 |
| 7 | 目标行与队列条同形相邻 | 形式级 | **不改**：A/B/C 提案已否（F20 记了理由），维持原判 | `U2-forms.md` F20 |

## 8 未覆盖

| 项 | 缺什么 |
|---|---|
| 真机帧率 | `ui.Tester` 是无窗口渲染；真实窗口的 vsync 与合成器开销未测（需可激活的 GUI 会话） |
| 1000 轮以上 | 未采样；虚拟化未实现 |
| Windows / Linux | 无目标机器（§7.1） |
| 断线态 | D1 未接，只读与回执规则未验 |

## 9 变更记录

| 日期 | 变更 | 级别 / 影响面 |
|---|---|---|
| 2026-11-13 | C4 初版：view 映射表（33 条，由 `U2-elements.json` 渲染，L19 守覆盖率）、分层与状态归属、性能预算与实测、虚拟化单位（轮次一行）、Markdown 路线（用 mygo 内建，不引入 markstream-go）、键位/焦点/读屏映射、校正清单 | 形式级补齐（技术设计）+ 控件级落地：新增 `U4-technical.md`、`internal/ui/perf_test.go`、`internal/ui/lint_test.go`（L19）；改 `internal/ui/{timeline,app}.go`（Markdown、展开态）、`docs/domains/{U2-elements.json,U2-checks.md}` |
