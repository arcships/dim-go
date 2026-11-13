# U2 检查器：规则 → 检查项 → 断言 → 修法

Status: 现行（v1 冻结期内的**追加**产物：只新增检查项，不改公理与已确认形式）
Owner: U2 界面视觉
依据: `docs/project-organization.md` §9.0（A0–A4、D0–D6、形式的选择）、§9.8（X1–X22）、§9.9（UI-1…UI-23）、§9.14（0–5 层、排版基线、动效从属、交付物）、§9.15（变更通道、v1 冻结、检查器门禁）
运行: `scripts/ui-lint`（工作区）、`scripts/ui-lint --seeds`（种子自检）、`scripts/ui-lint --all`
断言: `internal/ui/lint_test.go`（`TestUILint` / `TestUILintSeeds`）

本文件是检查器的唯一对照表：**一条规则 = 一个检查项 = 一个断言 = 一条修法**；指不回 §9.0 / §9.8 / §9.9 编号的不进检查器。

## 0 用法与失败输出

```bash
scripts/ui-lint            # 工作区必须零违例（退出码非 0 即失败）
scripts/ui-lint --seeds    # 五类种子违例必须全部报出并带位置
```

失败时每条给三样东西：`file:line`、检查项编号、修法。例：

```
composer.go:57 [L1] §9.14 排版基线 3 / X16：Padding(13) 用了档位外的数值
    修法：换成 tokens.go 的档位（space-1…12 / row-compact / icon-inline / hit-min），或新增常量并写明推导
sidebar.go:32 [L2] UI-21、X15：字符串里用字符当图标：搜索 ›
    修法：删掉字符，用 internal/ui/icons.go 的图标元素
```

## 1 检查项（16 条）

| # | 检查什么 | 规则 | 断言（`lint_test.go`） | 不通过时的修法 |
|---|---|---|---|---|
| L1 | 档位外的间距 / 尺寸 / 圆角 | §9.14 排版基线 3、X16、UI-22 | `lintSource` → `layoutCalls`：`Padding/Gap/Width/Height/MinHeight/MaxHeight/Radius/BorderWidth` 的字面量参数只允许 0 | 换成 `tokens.go` 的档位，或新增常量并写明推导 |
| L2 | 字符冒充图标（`＋ ‹ › × ⋯`） | UI-21、X15 | `lintSource` → `glyphIcons` | 删掉字符，改用 `icons.go` 的图标元素 |
| L3 | 图标入口缺可访问名 | UI-21、D4 | `lintSource`：`iconButtonSpec` 必须有 `label` | 补 `label`（读屏名） |
| L4 | 纯图标入口缺 tooltip（名称 / 快捷键） | UI-23 选择表、F17、§9.0「形式的选择」 | 静态：`iconButtonSpec` 至少有 `tip` 或 `label`；渲染：发送/停止的 `Access().Description` 必须含名称与 `Enter` | 补 `tip`：名称 · 快捷键（如「发送 · Enter」） |
| L5 | 高后果动作缺图标 / 可见文字 / 确认 | UI-23、F18、D5 | 静态：① 清单里的高后果动作不得走 `actionButton`；② `textIconButtonSpec` 的 `text` 字段必须存在且**不得是空串**（只留 `label` 可访问名会被静默绕过）。渲染：③ `U3-actions.json` 里 `form=icon+text` 的动作必须能用 `Find` 找到可见文本，且按钮宽于纯图标盒（`iconInline + 2*space3`）——只留 label 时宽度只有图标那么宽；④ 盒内有图标墨迹；⑤ `form=icon` 的动作不得出现可见文字（盒子不得宽于纯图标盒）；⑥ `confirm=true` 的第一次点击不得生效 | 改用 `textIconButton` 并写可见文字；需要确认的补两步 |
| L6 | 时间线状态位把状态画成文字 | A2/D2、UI-11、X20 | `lintSource`：`gui.Text/RichText` 的字面量不得是状态词（完成 / 正在运行 / 等待你同意 / 失败）；报错时按 `U1-content.json` 给出对应的信息项编号 | 状态用图标 + 颜色；读屏名可保留状态词 |
| L7 | 五种状态同屏两两可分 | D4 | `lintStatesDistinct`：状态读屏名集合不同 + 整帧像素指纹不同 | 让两个状态在图标、颜色或形态上至少差两项 |
| L8 | 三档宽度下内容左缘相等 | UI-22、§9.14 排版基线 2 | `lintBaselines`（1440/900/720）：标题行 / 时间线 / 输入卡同一左缘（±1.5） | 统一 `space-4` 外边距 + `space-2` 卡片内边距 |
| L9 | 组间间距 > 组内间距 | A3、§9.14 第 3 层 | `lintRoundSpacing`：下一轮用户消息与上一组头之间的间距 > 组内回复与组头之间的间距 | 轮次之间 `space-6`，轮内 `space-1/2/3` |
| L10 | 自绘描边只用主题 `Border` | UI-18、X14 | `lintSource`：`.Border(...)` 的色参必须是 `t.Border`，`BorderColor` 不得用字面色值 | 换 `t.Border`；需要更强区分时用面差 |
| L11 | 分隔线 / 自绘描边处数白名单 | UI-18、X14 | `lintDividerAllowlist`：每个文件的 `Divider(` + `.Border(` 处数等于白名单 | 能靠留白或面差分开的不画线；确需线时登记理由 |
| L12 | 关动效后信息与布局不变 | UI-5、§9.14 动效从属 | `lintMotionInvariance`：`ReduceMotion` 前后文字集合一致、标题行/对话流/输入框位置一致 | 把过渡换成瞬时，不动布局 |
| L13 | 命中区 ≥ 24 | UI-22 §6.5、`hit-min` | `lintHitAreas`：七个图标入口的 `Access().Bounds` 不小于 24×24 | 命中区放大到 `hit-min` |
| L14 | 自绘可交互元素有悬停 / 按下 / 禁用态 | X17、§9.14 第 4 层 | `lintInteractiveStates`：组头悬停 `surface-hover`、按下 `surface-pressed`；发送槽在无草稿时 `Disabled` | 补缺的那一态 |
| L15 | 200% 文本缩放下不越界 | §9.14 第 3 层、U2 §7.4 | `lintTextScale`：五个状态在 `TextScale: 2` 下所有可见文字不出窗口 | 调字号、允许单行截断 |
| L16 |清单引用完整性：U1 清单每条能指回信息项 | §9.14 第 1 层、U1-information | `lintListReferences`：`U1-content.json` 的 `item` 必须作为一行出现在 `U1-information.md` | 改信息项编号，或把条目从清单里删掉 |

**16 条与组织编号的对应**：A2/D2 → L6；A3 → L9；D4 → L3/L7；UI-5 → L12；UI-11 → L6；UI-18 → L10/L11；UI-21 → L2/L3；UI-22 → L1/L8/L13；UI-23 → L4/L5；X14 → L10/L11；X15 → L2；X16 → L1；X17 → L14；X20 → L6；§9.14 第 1 层 → L16。

## 2 白名单（逐条给理由）

**L1 的档位**：所有排布数值都走 `tokens.go` 的常量；检查器只放行字面量 `0`（无内边距 / 无间距）。

**L2 的内容放行**：内容字符串里出现 `＋ ‹ › × ⋯` 不算违例（U1 清单里 `kind=content` 的字符串整串放行）；当前清单没有这类字符串，所以放行列表为空，规则仍生效。

**L11 的分隔线与自绘描边**（每个文件的允许处数）：

| 文件 | 处数 | 为什么允许 |
|---|---:|---|
| `internal/ui/controls.go` | 1 | 附件 chip 落在 `surface` 上，面差为零，需要一条 `t.Border` 才看得出是「一个条目」 |
| `internal/ui/composer.go` | 1 | 审批卡：输入位被替换成一张决策卡，是唯一需要硬分开的自绘面（边界三档的第三档） |
| `internal/ui/icons.go` | 1 | 审批的次按钮（一直允许 / 拒绝）落在卡面上，需要一条 `t.Border` 轮廓 |
| `internal/ui/sidebar.go` | 1 | 框架 `gui.Divider`：入口区与会话列表是两个模块 |
| `internal/ui/rightpanel.go` | 1 | 框架 `gui.Divider`：标签栏与内容 |

**内容字符串与高后果动作的清单不在本文**（单一来源）：

| 清单 | 位置 | 检查项用它做什么 |
|---|---|---|
| 内容 vs 界面语 | [`U1-content.md`](U1-content.md) + [`U1-content.json`](U1-content.json) | L2 放行内容里的特殊字符；L6 报错时给出对应信息项编号 |
| 高后果动作 | [`U3-actions.md`](U3-actions.md) + [`U3-actions.json`](U3-actions.json) | L5 按 `form=icon+text` 判"必须有图标与可见文字"，按 `confirm=true` 判"必须两步" |

检查器直接读这两份 JSON（`loadJSON`）；读不到就让检查器失败，不许退化成空表。

## 3 种子自检（证明会报）

`TestUILintSeeds` 把违例注入到**合成源码 / 合成量测**里，断言每条都报出并带位置。当前输出：

```
种子「档位外间距」    → seed.go:2 [L1] §9.14 排版基线 3 / X16：Padding(13) 用了档位外的数值
种子「字符冒充图标」  → seed.go:2 [L2] UI-21、X15：字符串里用字符当图标：右栏 ›
种子「缺 tooltip」    → seed.go:2 [L4] UI-23 选择表、F17：纯图标入口缺 tooltip 与名称
种子「状态词上屏」    → seed.go:2 [L6] A2/D2、UI-11、X20：状态词被当成文字画到屏幕上：完成
种子「决定缺可见文字」→ seed.go:2 [L5] UI-23、F18：图标 + 文字的入口把可见文字写成了空串
种子「基线不等」      → seed [L8] UI-22、§9.14 排版基线 2：输入卡 在 900 档的左缘 275.0 与 1440 档的 272.0 不同
```

六类种子覆盖：档位外间距、字符冒充图标、缺 tooltip、状态词上屏、高后果动作缺可见文字、基线不等。

## 4 手工注入（评审用的复现步骤）

每条都能用"改一个常数 / 加一个字符 / 删一行"触发，检查器给 `file:line`：

| 检查项 | 怎么注入 | 期望输出 |
|---|---|---|
| L1 | `internal/ui/composer.go` 里把某一处 `Padding(space2)` 改成 `Padding(13)` | `composer.go:<行> [L1] … Padding(13) 用了档位外的数值` |
| L2 | 给任一图标入口的 `label` 末尾加 `›` | `sidebar.go:<行> [L2] … 搜索 ›`（并导致 L13 找不到原名入口） |
| L3 | 删掉某个 `iconButtonSpec{...}` 的 `label` 字段 | `[L3] … 图标入口缺可访问名` |
| L4 | 把 `sendSlot` 的 `tip: tip` 改成 `tip: ""` | `[L4] … 发送/停止的 tooltip 缺名称或快捷键`（渲染侧） |
| L5 | 把 `允许一次` 的 `textIconButton` 换成 `actionButton` | `[L5] … 高后果动作「允许一次」用了纯文字按钮` |
| L5 | 把 `允许一次` 的 `text: "允许一次"` 改成 `text: ""`（只留 label） | `composer.go:<行> [L5] … 图标 + 文字的入口把可见文字写成了空串` |
| L6 | 在 `timeline.go` 里加 `gui.Text(c, "完成")` | `[L6] … 状态词被当成文字画到屏幕上：完成` |
| L8 | 把输入卡的内边距从 `space2` 改成 `space3` | `[L8] … 输入卡 在 1440 档的左缘 … 不同` |
| L13 | 把 `iconButtonSpec` 的 `size` 改成 `20` | `[L13] … 命中区 20×20 小于 24` |

## 5 本轮顺带修的控件级问题

| 问题 | 改法 | 证据 |
|---|---|---|
| 审批卡里 `rm -rf ./tmp/*` 与影响说明粘连 | 命令单独放进代码块（`surface-pressed` 面 + 内边距），影响另起一行；块内文字有内边距，不参与区域文字列 | `window-waiting.png` / `state-waiting.png` |
| failed 态同时出现「结果不确定」与一条失败操作行 | 该轮的操作为「读取 tmp/ 完成」，不再有失败的 `✕` 行；失败操作由默认态的第一轮承担 | `window-failed.png` / `state-failed.png` |
| 未知提交不再画组头 | 未收到事实前不生成轮次：只保留占位与「结果不确定 · 正在核对」说明行，组头与操作行都不出现（U3 §6.1「未知的呈现边界」）。不再需要第五种轮次状态图标 | `window-failed.png` / `state-failed.png` |

## 6 未覆盖

| 项 | 缺什么 |
|---|---|
| 真实窗口的悬停 / tooltip 截图 | 本会话进程无法成为前台（`activate` 返回 true 而 `isActive=false`），hover 事件进不来；以断言代替（L4/L14） |
| Windows / Linux | 无目标机器（§7.1） |
| U1 内容白名单的正式版 | 现按 `internal/ui` 现状列出（§2），待 U1 出清单后替换并保留来源列 |
| U3 高后果动作清单的正式版 | 现按 U3 §5.2 列出（§2），待 U3 单列后补 |
