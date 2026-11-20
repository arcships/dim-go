# 需求实现状态：机械对账报告

Status: 对账报告（脚本生成，可重跑；`python3 scripts/reconcile.py`）
Scope: 四份需求表的全部条目（`docs/domains/requirements/requirements-*.md`）；状态列写回的是 §4「按元素映射判定」的结论，§1–§3 是第一轮记录，§5 是设置 19 节的五档分流与本轮做掉的条目（核对状态列，不看口头说明）
Owner: 组织 agent（验收判定侧）；需求文件与元素清单归 R 与需求分析领域

## 0. 判定口径（第一轮：逐字/符号/测试）

按顺序取第一个命中的证据：

| 判定 | 证据 | 含义 |
|---|---|---|
| 逐字 | 「期望」里「」/“”字面量在 `internal/**/*.go` 的字符串字面量出现 | 强证据，能指到具体函数 |
| 弱 | 清单符号、函数内元素 ID 注释、或字面量只出现在源码注释里 | 有对应实现点，但不证明逐字行为 |
| 测试 | `*_test.go` 断言该文案或提及该元素 ID | 有测试覆盖点，未证明实现 |
| 需负向取证 | 期望/验收为负向断言（不出现、无响应、出现次数 = 0 等） | 单独一档，不计入未实现 |
| 未实现 | 文案、符号、测试都不命中 | 缺 = 未找到文案/符号/测试 |

证据只写 `文件:函数`，不写行号；函数按顶层 `func` 与列 0 的 `}` 切分；包级常量/变量记声明名，找不到声明名才写 `包级`。

## 1. 第一轮命中分布（每份；状态列已由 §4 覆盖）

| 文件 | 总条数 | 逐字 | 弱 | 测试 | 需负向取证 | 未实现 | 逐字占比 | 未实现占比 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| `requirements-settings.md` | 8325 | 665 | 892 | 5 | 633 | 6130 | 8.0% | 73.6% |
| `requirements-main.md` | 3520 | 484 | 1664 | 107 | 384 | 881 | 13.8% | 25.0% |
| `requirements-global.md` | 2348 | 371 | 1178 | 48 | 197 | 554 | 15.8% | 23.6% |
| `requirements-panels.md` | 2152 | 351 | 378 | 100 | 170 | 1153 | 16.3% | 53.6% |
| **合计** | **16345** | **1871** | **4112** | **260** | **1384** | **8718** | **11.4%** | **53.3%** |

「未实现」里有一半以上是「期望列没有可逐字文案」造成的（无可用字面量 13925 条，其中判未实现 8382 条）；这类条目只能靠元素符号/测试或界面取证，见 §3。

## 2. 第一轮随机 40 条复核

抽样种子固定（`20261120`），每份 10 条，优先覆盖「逐字」与「未实现」两档；复核结论由脚本按证据反向核验，人工只需看标记为「有分歧」的条目。

### requirements-settings.md

| 需求 ID | 元素 ID | 维度 | 机器判定 | 证据 | 复核 |
|---|---|---|---|---|---|
| R-MCP-066 | S-mcp-09 | 内容 | 逐字 | `internal/ui/settings_mcp_screen.go:mcpCopyScopeAll`「全部范围」 | 通过（字面量在 internal/ui/settings_mcp_screen.go:mcpCopyScopeAll 内） |
| R-EXT-113 | S-externalAgents-15 | 内容 | 逐字 | `internal/ui/settings_external_agents_screen.go:eaCopyFootNote`「更改将在新对话中生效。」 | 通过（字面量在 internal/ui/settings_external_agents_screen.go:eaCopyFootNote 内） |
| R-GIT-025 | S-git-04 | 内容 | 逐字 | `internal/ui/settings_git_screen.go:gitCopyRemote`「默认远程仓库」 | 通过（字面量在 internal/ui/settings_git_screen.go:gitCopyRemote 内） |
| R-MOD-201 | S-modality-26 | 状态 | 逐字 | `internal/ui/composer_extra.go:modelEmptyState`「无可用模型」 | 通过（字面量在 internal/ui/composer_extra.go:modelEmptyState 内） |
| R-GEN-652 | S-general-82 | 关联 | 逐字 | `internal/ui/app.go:App.actionErrorView`「重试」 | 通过（字面量在 internal/ui/app.go:App.actionErrorView 内） |
| R-SHL-057 | S-shell-08 | 内容 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-PER-430 | S-personalization-45 | 内容 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-ABT-256 | S-about-32 | 深色 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-USG-337 | S-usage-42 | 内容 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-CFG-150 | S-configuration-19 | 无障碍 | 未实现 | — | 通过（文案/符号/测试均无命中） |

### requirements-main.md

| 需求 ID | 元素 ID | 维度 | 机器判定 | 证据 | 复核 |
|---|---|---|---|---|---|
| R-LB-737 | LB-46 | 内容 | 逐字 | `internal/ui/header.go:App.openWithMenu`「在 Finder 中显示」 | 通过（字面量在 internal/ui/header.go:App.openWithMenu 内） |
| R-TL-314 | TL-31 | 无障碍 | 逐字 | `internal/ui/timeline_inline.go:App.bubbleExpandButton`「展示全部」 | 通过（字面量在 internal/ui/timeline_inline.go:App.bubbleExpandButton 内） |
| R-LB-878 | LB-96 | 关联 | 逐字 | `internal/ui/sidebar.go:App.accountMenu`「邀请朋友」 | 通过（字面量在 internal/ui/sidebar.go:App.accountMenu 内） |
| R-LB-623 | LB-98 | 样式 | 逐字 | `internal/ui/confirm.go:confirmCopyDeleteOne`「删除」 | 通过（字面量在 internal/ui/confirm.go:confirmCopyDeleteOne 内） |
| R-LB-380 | LB-59 | 样式 | 逐字 | `internal/ui/sidebar_extra.go:App.sidebarGroupPager`「显示更多」 | 通过（字面量在 internal/ui/sidebar_extra.go:App.sidebarGroupPager 内） |
| R-IN-804 | IN-75 | 记忆 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-LB-279 | LB-43 | 状态 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-IN-755 | IN-71 | 记忆 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-LB-136 | LB-19 | 交互 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-LB-695 | LB-23 | 关联 | 未实现 | — | 通过（文案/符号/测试均无命中） |

### requirements-global.md

| 需求 ID | 元素 ID | 维度 | 机器判定 | 证据 | 复核 |
|---|---|---|---|---|---|
| R-GR-067 | G-F1-09 | 状态 | 逐字 | `internal/ui/composer.go:App.goalRow`「确认」 | 通过（字面量在 internal/ui/composer.go:App.goalRow 内） |
| R-GI-100 | G-H1-12 | 内容 | 逐字 | `internal/ui/plugin_install.go:pluginDialogTitle`「添加插件」 | 通过（字面量在 internal/ui/plugin_install.go:pluginDialogTitle 内） |
| R-GF-108 | G-G1-14 | 交互 | 逐字 | `internal/ui/command_palette.go:App.commandPaletteActions`「导出」 | 通过（字面量在 internal/ui/command_palette.go:App.commandPaletteActions 内） |
| R-GC-134 | G-E4-02 | 内容 | 逐字 | `internal/ui/confirm.go:confirmCopyWorkspaceF`「该工作区」 | 通过（字面量在 internal/ui/confirm.go:confirmCopyWorkspaceF 内） |
| R-GQ-116 | G-D1-14 | 状态 | 逐字 | `internal/ui/question.go:questionCopyStale`「提交」 | 通过（字面量在 internal/ui/question.go:questionCopyStale 内） |
| R-GM-178 | G-B2-02 | 内容 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-GS-170 | G-A2-11a | 样式 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-GI-164 | G-H2-08 | 状态 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-GS-060 | G-A1-06 | 状态 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-GI-175 | G-H3-01 | 状态 | 未实现 | — | 通过（文案/符号/测试均无命中） |

### requirements-panels.md

| 需求 ID | 元素 ID | 维度 | 机器判定 | 证据 | 复核 |
|---|---|---|---|---|---|
| R-RA-130 | A-16 | 状态 | 逐字 | `internal/ui/rightpanel_changes.go:App.rpChangeMenu`「用默认应用打开」 | 通过（字面量在 internal/ui/rightpanel_changes.go:App.rpChangeMenu 内） |
| R-TK-217 | K-27 | 无障碍 | 逐字 | `internal/ui/rightpanel_tasks.go:App.taskDetail`「打开日志」 | 通过（字面量在 internal/ui/rightpanel_tasks.go:App.taskDetail 内） |
| R-TK-006 | K-01 | 无障碍 | 逐字 | `internal/ui/rightpanel_tasks.go:App.tasksPanel`「后台进程」 | 通过（字面量在 internal/ui/rightpanel_tasks.go:App.tasksPanel 内） |
| R-RD-885 | D-96 | 状态 | 逐字 | `internal/ui/rightpanel_changes.go:App.rpChangeMenu`「复制路径」 | 通过（字面量在 internal/ui/rightpanel_changes.go:App.rpChangeMenu 内） |
| R-RD-494 | D-53 | 状态 | 逐字 | `internal/ui/rightpanel_changes.go:App.rpChangeGroups`「全部暂存」 | 通过（字面量在 internal/ui/rightpanel_changes.go:App.rpChangeGroups 内） |
| R-RD-302 | D-33 | 内容 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-RD-060 | D-05 | 样式 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-RA-083 | A-10 | 状态 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-RF-356 | F-42 | 样式 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-RA-105 | A-12 | 记忆 | 未实现 | — | 通过（文案/符号/测试均无命中） |

抽样复核：40 条中 40 条与证据一致，0 条标记有分歧。

## 3. 第一轮已知局限

- **动态拼接文案**：期望列里带 `{}`、`…` 的字面量（如「已使用 {pct}%」「已于 … 到期」）被跳过，无法逐字命中，只会落到弱/未实现。
- **没有引号的期望**：期望列用叙述而不是「」/“”给文案时抽不出字面量，这类条目只能走符号/测试，统计见 §1 的「无可用字面量」一行。
- **尾部标点**：先按原字面量查，查不到再去掉尾部 `。！？；：` 重查，证据记录实际命中的写法。
- **同文案多处出现**：只记第一次命中的 `文件:函数`，不证明该函数属于本元素；同名文案跨节复用时证据可能指错实现点。
- **无 i18n 层**：代码未建 i18n，期望为英文键名（如 `settings.account.title`）时无法逐字命中，按未实现计。
- **注释提及不算逐字**：字面量只出现在 Go 注释里时记「已实现（弱）」；这样「未实现」的含义是：字符串字面量、注释、符号、测试都搜不到。
- **弱证据的来源**：清单没有 symbol 列；名称/类型/模板/定位列派生的 Vue 组件名在 Go 里几乎不出现（本轮只有个别命中），实际起作用的是「函数内注释标注元素 ID」。同名符号与顺带提及的注释都可能误命中。
- **负向要求不判实现**：只标「需负向取证」，需要界面驱动或代码反向事实，不计入未实现。
- **静态匹配不看分支**：字面量命中的函数即使处在不可达分支，也会判逐字；不替代界面验收证据。
- **函数区间是启发式**：顶层 `func` 到列 0 的 `}`，闭包与生成代码可能切错；证据写函数名而非行号，切错时函数名仍可用。
- **只扫 `internal/**/*.go`**：`cmd/`、生成文件与前端资源不在扫描范围。

## 4. 第二轮：按元素映射判定

第一轮把「期望列抽不出字面量」的叙述型需求压成「未实现」，那是下限，不是结论。第二轮改按**元素 → 渲染函数 → 维度标记**判。

### 4.1 判定口径

| 判定 | 条件 | 状态列写法 |
|---|---|---|
| 逐字 | 第一轮逐字命中（不重判） | 沿用 `已实现（逐字）：…` |
| 映射命中 | 元素映射到渲染函数，函数体内有该维度的标记 | `已实现（映射 = 元素→文件:函数；标记 = 标记）` |
| 映射到位无标记 | 映射到位，函数体内没有该维度的标记 | `未实现（映射到位但没有该维度的标记）：元素→文件:函数` |
| 无渲染点 | 元素 ID 在 `internal/**/*.go` 的注释里反查不到渲染函数 | `未实现（找不到元素渲染点）` |
| 负向 | 第一轮负向断言（不重判） | 沿用 `需负向取证：…` |

映射来源（按优先级）：函数 doc 注释里的元素 ID（同一处注释覆盖的元素越少越优先）→ 函数体内注释里的元素 ID → 清单 `symbol` 派生的 Go 标识符。注释里的 `S-account-02–07`、`S-account-10/11/13`、`TB-20–TB-30`、`S-about-01…18` 按区间/并列展开，`A1-03` 这类短写还原成 `G-A1-03`，展开结果必须落在已知元素 ID 集合里。跨域误命中按元素族过滤：设置元素只认 `settings*.go`，单字母面板元素不认设置屏。

函数体 = 去注释、去字符串的代码行 + 函数区间内的字符串字面量，并向下展开一跳同包（同目录）被调函数，方法按裸函数名解析（`a.rpGroup(` 能解析到 `App.rpGroup`）；命中记 `标记 = X@被调函数`。

| 维度 | 标记组 | 来源 |
|---|---|---|
| 交互 | `Clicked|Pressed|Key|OnSubmit|Menu` | 领导给定 |
| 位置 | `Row|Column|Align|Gap|Padding|Place|Offset` | 本轮补充 |
| 关联 | `Switch|Navigate|Screen|Section|Tab|Select|Open|Show` | 本轮补充 |
| 内容 | 函数内中文字面量 | 领导给定 |
| 动画 | `Animate|Duration|ease|Ease` | 领导给定 |
| 多窗口 | `Window` | 本轮补充 |
| 多语言 | 函数内中文字面量 | 本轮补充 |
| 密度 | `spaceUnit|Density|Compact|Gap` | 本轮补充 |
| 展开后 | `Expand|Collapsed|Visible` | 本轮补充 |
| 异常 | `(?i)err|错误|失败|重试|Reason` | 领导给定 |
| 性能 | `Duration|Throttle|Debounce|Cache|time\.` | 本轮补充 |
| 悬停 | `Hover|Hovered` | 本轮补充 |
| 拖拽 | `Drag|Drop` | 本轮补充 |
| 持久化 | `(?i)prefs|persist|View\.` | 领导给定 |
| 数据量 | `Limit|Count|len\(|Max` | 本轮补充 |
| 文案 | 函数内中文字面量 | 领导给定 |
| 无障碍 | `Label|Role|AccessHidden|Aria` | 领导给定 |
| 暗色 | `Theme|Dark|Light` | 本轮补充 |
| 样式 | `TextColor|Radius|Padding|Width|Height|Font|Type` | 领导给定 |
| 深色 | `Theme|Dark|Light` | 本轮补充 |
| 状态 | `State|Disabled|Enabled|Visible|Hidden|Loading|Invalidate` | 本轮补充 |
| 空间 | `spaceUnit|Width|Height|Gap` | 本轮补充 |
| 窄窗 | `narrow|Narrow|Compact|compact` | 领导给定 |
| 记忆 | `(?i)prefs|persist|View\.` | 领导给定 |
| 语言 | 函数内中文字面量 | 本轮补充 |
| 输入法 | `IME|Composition|Compose|Marked` | 本轮补充 |
| 键盘 | `Key|Shortcut|Escape|Enter|Cmd|Ctrl|Tab` | 本轮补充 |
| 高DPI | `Scale|Dpi|DPI|Px|Pixel` | 本轮补充 |

未列出的维度按「内容」判（函数内中文字面量）。八组之外的维度用补充标记组，因为需求里出现的维度只有一半落在八组内；记忆/持久化与异常两组按代码习惯写成大小写不敏感（`a.persistPreferences()`、`ErrMsg`），表里给的是实际正则。

### 4.2 分布

| 文件 | 总条数 | 逐字 | 映射命中 | 映射到位无标记 | 无渲染点 | 负向 | 映射命中占比 |
|---|---:|---:|---:|---:|---:|---:|---:|
| `requirements-settings.md` | 8325 | 665 | 4632 | 2004 | 391 | 633 | 55.6% |
| `requirements-main.md` | 3520 | 484 | 1763 | 709 | 180 | 384 | 50.1% |
| `requirements-global.md` | 2348 | 371 | 932 | 450 | 398 | 197 | 39.7% |
| `requirements-panels.md` | 2152 | 351 | 1113 | 382 | 136 | 170 | 51.7% |
| **合计** | **16345** | **1871** | **8440** | **3545** | **1105** | **1384** | **51.6%** |

去掉逐字与负向后可判 13090 条：元素映射到位 11985 条（91.6%），其中缺该维度标记 3545 条；找不到渲染点 1105 条（8.4%）。

「无渲染点」是映射的缺口，不是实现的结论：元素 ID 没在 Go 注释里标注，就查不到渲染函数。按域看元素映射覆盖率：

| 文件 | 元素数 | 有渲染点 | 覆盖率 |
|---|---:|---:|---:|
| `requirements-settings.md` | 992 | 935 | 94.3% |
| `requirements-main.md` | 344 | 322 | 93.6% |
| `requirements-global.md` | 276 | 214 | 77.5% |
| `requirements-panels.md` | 244 | 222 | 91.0% |

缺标注最多的元素族（按条数）：`S-configuration` 160 条、`S-general` 112 条、`A` 99 条、`G-B2` 72 条、`G-H2` 72 条、`NT` 62 条、`TL` 58 条、`S-mcp` 56 条。

### 4.3 按维度

| 维度 | 条数 | 逐字 | 映射命中 | 映射到位无标记 | 无渲染点 | 负向 | 标记组来源 |
|---|---:|---:|---:|---:|---:|---:|---|
| 状态 | 2570 | 490 | 1067 | 296 | 163 | 554 | 本轮补充 |
| 样式 | 1984 | 55 | 1660 | 75 | 181 | 13 | 领导给定 |
| 内容 | 1924 | 684 | 958 | 12 | 135 | 135 | 领导给定 |
| 无障碍 | 1893 | 304 | 1359 | 56 | 155 | 19 | 领导给定 |
| 异常 | 1745 | 94 | 640 | 598 | 105 | 308 | 领导给定 |
| 交互 | 1436 | 66 | 822 | 305 | 91 | 152 | 领导给定 |
| 关联 | 1380 | 136 | 918 | 134 | 72 | 120 | 本轮补充 |
| 窄窗 | 858 | 11 | 72 | 683 | 64 | 28 | 领导给定 |
| 记忆 | 688 | 20 | 70 | 532 | 39 | 27 | 领导给定 |
| 深色 | 569 | 0 | 538 | 3 | 28 | 0 | 本轮补充 |
| 键盘 | 482 | 5 | 167 | 277 | 27 | 6 | 本轮补充 |
| 高DPI | 292 | 0 | 23 | 255 | 11 | 3 | 本轮补充 |
| 动画 | 285 | 1 | 31 | 224 | 21 | 8 | 领导给定 |
| 性能 | 88 | 0 | 16 | 62 | 7 | 3 | 本轮补充 |
| 语言 | 69 | 4 | 57 | 0 | 4 | 4 | 本轮补充 |
| 输入法 | 30 | 0 | 2 | 26 | 0 | 2 | 本轮补充 |
| 多窗口 | 14 | 0 | 9 | 4 | 1 | 0 | 本轮补充 |
| 多语言 | 13 | 0 | 12 | 0 | 0 | 1 | 本轮补充 |
| 位置 | 9 | 1 | 8 | 0 | 0 | 0 | 本轮补充 |
| 拖拽 | 4 | 0 | 2 | 2 | 0 | 0 | 本轮补充 |
| 密度 | 3 | 0 | 3 | 0 | 0 | 0 | 本轮补充 |
| 暗色 | 2 | 0 | 1 | 0 | 0 | 1 | 本轮补充 |
| 悬停 | 2 | 0 | 2 | 0 | 0 | 0 | 本轮补充 |
| 数据量 | 1 | 0 | 0 | 1 | 0 | 0 | 本轮补充 |
| 空间 | 1 | 0 | 1 | 0 | 0 | 0 | 本轮补充 |
| 分工 | 1 | 0 | 1 | 0 | 0 | 0 | 未定义（按内容） |
| 执行 | 1 | 0 | 0 | 0 | 1 | 0 | 未定义（按内容） |
| 展开后 | 1 | 0 | 1 | 0 | 0 | 0 | 本轮补充 |

**横向维度的实现层级在元素函数之上**：窄窗、深色、记忆、高 DPI、键盘、动画这一类，代码把判定放在屏根 / `App` / 主题层（如 `isCompact(width)`、`persistViewState`、主题 token），元素函数本身不出现标记。这些维度的「映射到位但没有标记」多数是层级问题，不是元素没实现；要取证得看屏根函数或界面实拍。

### 4.4 逐字档的映射一致性

逐字 1871 条中：字面量落在元素映射函数内 1370 条；元素有映射但字面量不在映射函数内 381 条；元素没有映射点 120 条。

映射不一致的前 10 条（逐字证据可能指错实现点）：

| 需求 ID | 元素 ID | 逐字证据 | 元素映射 |
|---|---|---|---|
| R-ACC-010 | S-account-02 | `internal/ui/settings_account_screen.go:accCopyTitle`「账户」 | `internal/ui/settings_account_screen.go:App.accountExpiryBanner` |
| R-ACC-041 | S-account-05 | `internal/ui/settings_plugin_settings_screen.go:plgCopyRootNote`「已打开」 | `internal/ui/settings_account_screen.go:App.accountExpiryBanner` |
| R-ACC-083 | S-account-08 | `internal/ui/confirm.go:confirmCopyUnnamed`「未命名」 | `internal/ui/settings_account_screen.go:App.accountIdentityCard` |
| R-ACC-191 | S-account-16 | `internal/ui/settings_account_screen.go:accCopyRenew`「续订」 | `internal/ui/settings_account_screen.go:App.accountViewPlanButton` |
| R-ACC-398 | S-account-32 | `internal/ui/settings_account_screen.go:accUsedLabel`「已使用」 | `internal/ui/settings_account_screen.go:App.accountPlanHeader` |
| R-ACC-439 | S-account-36 | `internal/ui/settings_account_screen.go:accCopyLoginDetail`「DimAgent」 | `internal/ui/settings_account_screen.go:accountBrandMark` |
| R-ACC-444 | S-account-37 | `internal/ui/settings_account_screen.go:accCopyLoginDetail`「DimAgent」 | `internal/ui/settings_account_screen.go:App.accountServiceRow` |
| R-ACC-446 | S-account-37 | `internal/ui/settings_account_screen.go:accCopyLoginDetail`「DimAgent」 | `internal/ui/settings_account_screen.go:App.accountServiceRow` |
| R-ACC-452 | S-account-38 | `internal/ui/settings_account_screen.go:accountSampleData`「法至内部使用 · ¥0.10 / 月」 | `internal/ui/settings_account_screen.go:App.accountServiceRow` |
| R-ACC-501 | S-account-43 | `internal/ui/settings_account_screen.go:accCopyResetUsage`「重置当月用量」 | `internal/ui/settings_account_screen.go:App.accountResetPill` |

### 4.5 随机 40 条复核（重点：映射到位但没有标记）

抽样种子 `20261121`，每份 10 条 = 7 条「映射到位无标记」+ 2 条「映射命中」+ 1 条「无渲染点」。人工结论存 `scripts/reconcile-review-mapping.json`（需求 ID → 结论），脚本重跑时自动带出；「函数体信号」是该函数直接调用到的构件。

#### requirements-settings.md

| 需求 ID | 元素 ID | 维度 | 判定 | 映射 | 函数体信号 | 人工复核 |
|---|---|---|---|---|---|---|
| R-ABT-295 | S-about-37 | 异常 | 映射到位无标记 | `internal/ui/settings_about_screen.go:abtDebugSection` | Column, Shrink, Gap, Radius, Label, BorderWidth, BorderColor, BorderStyle | 待复核 |
| R-CFG-139 | S-configuration-18 | 状态 | 映射到位无标记 | `internal/ui/settings_configuration_screen.go:configBaseURL` | Theme, Column, Shrink, Gap, Children, Text, TextColor, SingleLine | 待复核 |
| R-USG-478 | S-usage-59 | 状态 | 映射到位无标记 | `internal/ui/settings_usage_screen.go:renderSettingsUsageScreen` | Theme, Column, Grow, Children, Scroll, Gap | 有分歧（映射错位）：映射到用量屏根，用量屏只渲染标题与不可用卡；S-usage-59 模型用量在代码里没有渲染点 |
| R-PER-604 | S-personalization-66 | 异常 | 映射到位无标记 | `internal/ui/settings_personalization_screen.go:renderSettingsPersonalizationScreen` | Theme, Column, Grow, Children, Scroll, Gap, Text, TextColor | 确认：该屏无长度校验与保存提示（psCustomInstructionsSection 只有说明与打开文件按钮）；映射点偏到屏根 |
| R-APP-209 | S-appearance-24 | 动画 | 映射到位无标记 | `internal/ui/settings_appearance_screen.go:appearanceFontSection` | Theme, Column, Shrink, Gap, Children | 有分歧（负向期望）：期望是「无逐项淡入」，函数里没有动效标记恰好满足；机器按未实现计是口径问题 |
| R-PRM-095 | S-permissions-12 | 窄窗 | 映射到位无标记 | `internal/ui/settings_permissions_screen.go:permIsolationControlsDisabled` | — | 待复核 |
| R-MOD-055 | S-modality-07 | 异常 | 映射到位无标记 | `internal/ui/settings_modality_screen.go:mdGroupTabs` | Size, TabsBase, Grow, AlignItems, Label, Children, Tab, Shrink | 待复核 |
| R-USG-011 | S-usage-02 | 内容 | 映射命中 | `internal/ui/settings_usage_screen.go:usUnavailableCard` 标记 文案「不可用」等 2 条 | Row, Shrink, Wrap, AlignItems, Padding, Role, Label, Children | 待复核 |
| R-MOD-112 | S-modality-14 | 深色 | 映射命中 | `internal/ui/settings_modality_screen.go:mdCapabilityPanel` 标记 Theme | Column, Shrink, Gap, Label, Children, Row, AlignItems, Text | 待复核 |
| R-GEN-611 | S-general-77 | 样式 | 无渲染点 | — | — | 待复核 |

#### requirements-main.md

| 需求 ID | 元素 ID | 维度 | 判定 | 映射 | 函数体信号 | 人工复核 |
|---|---|---|---|---|---|---|
| R-IN-796 | IN-74 | 记忆 | 映射到位无标记 | `internal/ui/approval.go:App.approvalActions` | Row, WidthPercent, Gap, AlignItems, Wrap, Children, Spacer, PrimaryButton | 待复核 |
| R-TL-065 | TL-08 | 交互 | 映射到位无标记 | `internal/ui/timeline.go:App.backToLatest` | Row, Absolute, Bottom, Right, Children, ScrollTo | 待复核 |
| R-TL-560 | TL-56 | 记忆 | 映射到位无标记 | `internal/ui/timeline_extra.go:App.subagentChip` | Theme, ButtonBase, Shrink, Height, Gap, Radius, Background, Cursor | 待复核 |
| R-IN-082 | IN-06 | 记忆 | 映射到位无标记 | `internal/ui/composer.go:App.modeSelector` | Theme, MenuBase, Width, Height, Radius, Tooltip, Cursor, Hovered | 待复核 |
| R-SB-224 | SB-19 | 记忆 | 映射到位无标记 | `internal/ui/statusbar.go:App.statusUsage` | Theme, MenuBase, Height, Padding, Radius, AlignItems, Label, Sprintf | 待复核 |
| R-LB-228 | LB-34 | 异常 | 映射到位无标记 | `internal/ui/sidebar.go:App.workspaceGroup` | Theme, Column, Shrink, Children, Row, PaddingStart, ButtonBase, Height | 待复核 |
| R-TL-767 | TL-77 | 记忆 | 映射到位无标记 | `internal/ui/timeline.go:App.opRow` | Theme, Sprintf, ButtonBase, Grow, Height, Gap, AlignItems, Padding | 待复核 |
| R-LB-032 | LB-05 | 状态 | 映射命中 | `internal/ui/sidebar_extra.go:App.sidebarUpdateButton` 标记 Disabled | Theme, ButtonBase, Shrink, Height, Justify, Radius, Gap, AlignItems | 待复核 |
| R-TB-045 | TB-06 | 样式 | 映射命中 | `internal/ui/search.go:App.sessionSearchOverlay` 标记 Width | Theme, Size, Overlay, Box, Absolute, Fill, AccessHidden, OnClick | 待复核 |
| R-TL-090 | TL-10 | 样式 | 无渲染点 | — | — | 待复核 |

#### requirements-global.md

| 需求 ID | 元素 ID | 维度 | 判定 | 映射 | 函数体信号 | 人工复核 |
|---|---|---|---|---|---|---|
| R-GS-164 | G-A2-10 | 异常 | 映射到位无标记 | `internal/ui/search_chat.go:App.chatSearchHitRow` | Theme, Row, Shrink, Height, Padding, AlignItems, Radius, Role | 待复核 |
| R-AP-277 | G-C2-06 | 异常 | 映射到位无标记 | `internal/ui/approval.go:approvalActionsFor` | — | 待复核 |
| R-AP-402 | G-C4-03 | 记忆 | 映射到位无标记 | `internal/ui/approval_panels.go:approvalCommandPanel` | TrimSpace | 待复核 |
| R-AP-469 | G-C4-11 | 样式 | 映射到位无标记 | `internal/ui/approval_panels.go:approvalSearchPanel` | — | 待复核 |
| R-GC-075 | G-E3-01 | 状态 | 映射到位无标记 | `internal/ui/confirm.go:App.confirmTitle` | Sprintf | 待复核 |
| R-GM-088 | G-B1-11 | 样式 | 映射到位无标记 | `internal/ui/slash_commands.go:App.slashCommands` | — | 待复核 |
| R-AP-148 | G-C1-14 | 窄窗 | 映射到位无标记 | `internal/ui/approval.go:App.handoffBlock` | Theme, Column, WidthPercent, Padding, Gap, Radius, Label, Text | 待复核 |
| R-GS-317 | G-A3-11 | 交互 | 映射命中 | `internal/ui/mention.go:App.syncPick` 标记 Menu | — | 待复核 |
| R-AP-179 | G-C1-18 | 样式 | 映射命中 | `internal/ui/approval.go:approvalActionsFor` 标记 Type | — | 待复核 |
| R-GS-013 | G-A1-02a | 样式 | 无渲染点 | — | — | 待复核 |

#### requirements-panels.md

| 需求 ID | 元素 ID | 维度 | 判定 | 映射 | 函数体信号 | 人工复核 |
|---|---|---|---|---|---|---|
| R-TM-140 | T-17 | 高DPI | 映射到位无标记 | `internal/ui/terminal.go:App.termMenu` | Item, Disabled, Chosen, ReadClipboard, Separator, Subtitle, Danger | 待复核 |
| R-RD-156 | D-15 | 窄窗 | 映射到位无标记 | `internal/ui/rightpanel_changes.go:App.rpScopeBar` | Theme, Row, Height, Shrink, Padding, Gap, Children, MenuBase | 待复核 |
| R-RD-475 | D-50 | 记忆 | 映射到位无标记 | `internal/ui/rightpanel_changes.go:App.rpChangeGroups` | Sprintf | 待复核 |
| R-RD-697 | D-73 | 记忆 | 映射到位无标记 | `internal/ui/rightpanel_changes.go:App.rpRevertDialog` | Modal, Theme, Column, Gap, Padding, Width, Label, Radius | 有分歧（负向期望）：期望是「标题不变、说明换路径」，属于稳定性断言；无记忆标记不构成缺失证据 |
| R-TK-161 | K-20 | 动画 | 映射到位无标记 | `internal/ui/rightpanel_tasks.go:App.taskStateGlyph` | Box, Width, Height, Shrink, AccessHidden, DrawOver, Rotate, Loop | 待复核 |
| R-RF-237 | F-27 | 键盘 | 映射到位无标记 | `internal/ui/rightpanel_files.go:App.rpFileRow` | Theme, Sprintf, ButtonBase, Height, Padding, AlignItems, Radius, Level | 确认：rpFileRow 没有键盘处理（只有 Pressed/Hovered/Clicked），逐行焦点与 Enter 预览未实现 |
| R-RD-664 | D-70 | 高DPI | 映射到位无标记 | `internal/ui/rightpanel_changes.go:App.rpChangeRow` | Theme, Sprintf, Row, Height, Padding, AlignItems, Role, Hovered | 待复核 |
| R-RF-273 | F-32 | 内容 | 映射命中 | `internal/ui/rightpanel_files.go:App.rpFileRow` 标记 文案「文件，」等 5 条 | Theme, Sprintf, ButtonBase, Height, Padding, AlignItems, Radius, Level | 待复核 |
| R-RF-012 | F-01 | 异常 | 映射命中 | `internal/ui/rightpanel_files.go:App.rpFilterRow` 标记 erR | Theme, Column, Shrink, Padding, Children, Row, Height, Gap | 待复核 |
| R-RA-046 | A-06 | 异常 | 无渲染点 | — | — | 待复核 |

人工复核 5 条：与机器判定一致 2 条，有分歧 3 条。分歧构成：负向期望 2、映射错位 1。

### 4.6 「缺标记」里有多少是标记组漏词

对「映射到位但没有该维度的标记」的条目，再查一遍同维度在 mygo 里的其它写法（`DIM_ADJACENT`，只作参考、不改判定）。这类条目更像标记组不够宽，不像元素缺实现。

| 维度 | 缺标记条数 | 其中函数体有同类写法 | 例子 |
|---|---:|---:|---|
| 窄窗 | 683 | 619（90.6%） | R-ACC-033 `Wrap` |
| 异常 | 598 | 124（20.7%） | R-ACC-029 `Banner` |
| 记忆 | 532 | 210（39.5%） | R-ACC-354 `.accountEmailOpen =` |
| 交互 | 305 | 138（45.2%） | R-ACC-074 `Selectable` |
| 状态 | 296 | 31（10.5%） | R-RUN-033 `Empty@rtLiveSection` |
| 键盘 | 277 | 11（4.0%） | R-ACC-603 `Focus` |
| 高DPI | 255 | 187（73.3%） | R-ACC-027 `Size@fontStep.style` |
| 动画 | 224 | 101（45.1%） | R-ACC-017 `Opacity@accButton` |
| 关联 | 134 | 4（3.0%） | R-IN-482 `Close` |
| 样式 | 75 | 8（10.7%） | R-TL-006 `Fill` |
| 性能 | 62 | 0（0.0%） | — |
| 无障碍 | 56 | 0（0.0%） | — |
| 输入法 | 26 | 0（0.0%） | — |
| 内容 | 12 | 0（0.0%） | — |
| 多窗口 | 4 | 0（0.0%） | — |
| 深色 | 3 | 0（0.0%） | — |
| 拖拽 | 2 | 0（0.0%） | — |
| 数据量 | 1 | 0（0.0%） | — |

### 4.7 最该先修的 10 个「映射到位但缺标记」元素

排序用「真缺」＝映射到位、缺该维度的标记，且函数体里连同维度的其它写法（§4.6 的 `DIM_ADJACENT`）也没有；「缺标记条数」是全量。维度缺口列是该元素缺标记的维度构成（按条数）。

| # | 元素 ID | 名称 | 真缺条数 | 缺标记条数 | 覆盖需求 | 维度缺口 | 映射点 |
|---:|---|---|---:|---:|---:|---|---|
| 1 | IN-66 | （无文案） | 14 | 15 | 32 | 关联 4、键盘 3、记忆 3、动画 1 | `internal/ui/approval.go:App.approvalCard` |
| 2 | G-C1-01 | （无文案）卡片容器 | 12 | 13 | 30 | 关联 4、动画 2、记忆 2、交互 1 | `internal/ui/approval.go:App.approvalCard` |
| 3 | TL-08 | （无文案） | 12 | 12 | 15 | 状态 3、交互 3、样式 1、动画 1 | `internal/ui/timeline.go:App.backToLatest` |
| 4 | D-77 | {count} 个文件改动 | 9 | 10 | 26 | 关联 3、窄窗 1、记忆 1、交互 1 | `internal/ui/rightpanel_changes.go:App.rpChangesFooter` |
| 5 | G-C4-01 | （无文案）容器 | 9 | 11 | 16 | 状态 3、交互 2、关联 2、记忆 1 | `internal/ui/approval_panels.go:approvalInputPanel` |
| 6 | TL-67 | 已压缩 / 自动压缩 + `(部分)` | 9 | 9 | 14 | 状态 2、交互 1、样式 1、动画 1 | `internal/ui/timeline_compaction.go:compactionWord` |
| 7 | G-C1-20 | 「始终允许此命令」 / 「允许此会话期间访问」  | 8 | 8 | 16 | 状态 2、关联 1、无障碍 1、键盘 1 | `internal/ui/approval.go:approvalLongTermCopy` |
| 8 | IN-35 | （排队消息面板） | 8 | 8 | 14 | 状态 5、关联 1、异常 1、记忆 1 | `internal/ui/composer_panels.go:App.queuePanel` |
| 9 | TL-55 | （工具调用详情） | 8 | 8 | 21 | 状态 4、动画 1、记忆 1、高DPI 1 | `internal/ui/timeline.go:App.toolDetail` |
| 10 | G-A5-02 | （当前命中片段） | 7 | 7 | 10 | 交互 2、关联 1、无障碍 1、异常 1 | `internal/ui/search.go:App.searchReveal` |

### 4.8 第二轮已知局限

- **映射靠注释**：清单没有 `symbol` 列，映射实际依赖 Go 注释里的元素 ID；注释没写的元素一律「无渲染点」，那是缺映射，不等于缺实现。
- **一处注释覆盖多元素**：`S-account-02–07` 这类区间把整段元素挂到同一函数，函数里的标记会算到区间内每个元素头上。
- **「内容」几乎必命中**：渲染函数里只要有一个中文字面量就算命中，所以内容档的「映射命中」只说明有文案，不说明是本元素那句。
- **`err` 标记过宽**：异常档的 `(?i)err` 会匹配 `error`/`ErrMsg` 一类通用名字，异常档命中偏松；异常档缺标记反而更可信。
- **标记组窄于实现词汇**：样式档只认 `TextColor|Radius|Padding|Width|Height|Font|Type`，用 `Background`/`BorderColor`/`Opacity`/`Gap`/`AlignItems` 写的样式会被判「无标记」。
- **只展开一跳**：被调函数再往下调用的辅助函数不查，深层复用的标记会漏。
- **横向维度按元素函数查**：窄窗 / 深色 / 记忆 / 高 DPI / 键盘的实现层级在屏根与主题层，这类「映射到位但缺标记」不能当作元素缺陷读。
- **按维度判，不看数值**：命中只说明函数里出现过该类标记，不证明取值等于期望（字号、间距、颜色仍要界面取证）。
- **逐字档不重判**：第一轮逐字证据可能指错实现点（§4.4 给了不一致条数与样例）。

## 5. 附录：设置 19 节的未实现分流（五档）

本节只分 `requirements-settings.md`（19 节 + 壳层）。分流的对象是**现行状态列**的未实现条目（§4 口径：映射到位无标记 + 无渲染点），共 2395 条；第一轮逐字口径的 6,140 条是同一批需求的下限计数，两者不是两批条目，差别在判定口径（§1 的「未实现」含「期望列抽不出字面量」的叙述型条目）。

### 5.1 分档口径

| 档 | 判据（按顺序取第一个命中） | 含义 |
|---|---|---|
| 需素材 | 期望列命中素材词（皮肤、缩略图、素材、封面、波形、预览图、用图片、图片资源、图标资源） | 缺图片素材（皮肤图、缩略图、封面）就画不出，写实现也验不了 |
| 需宿主能力 | 期望/场景/验收列命中宿主接口词（17 个），或元素族属 `S-about`, `S-general`, `S-notifications`, `S-runtime` | 要系统对话框、托盘、通知、休眠、下载安装等宿主通道 |
| 需协议 | 元素族属 `S-account`, `S-externalAgents`, `S-git`, `S-hooks`, `S-mcp`, `S-modality`, `S-personalization`, `S-pluginSettings`, `S-usage` | 事实、写入或清单在 App Server，protocol v3 没有对应方法或字段 |
| 取证口径 | 判定为无渲染点，或维度属横向维度（位置、动画、多窗口、多语言、密度、展开后、性能、悬停、拖拽、数据量、暗色、深色、空间、窄窗、记忆、语言、输入法、键盘、高DPI），或场景是「语言包缺失回落」 | 这一档不是待实现功能，是判定要补证据：补元素锚点后重判，或界面实拍取证 |
| 本版可做 | 其余 | 本地偏好、本地草稿、已有事实能落地；本节列出这些条目 |

表里的词表与元素族是**一次判定**，脚本按它逐条分，不逐条人工过；分档只说「要什么才能做」，不改状态列（状态列仍由 §4 判）。「无渲染点」按 §4.2 的口径归取证档：那是映射缺口，不是实现结论；把它记成「需协议」会把缺注释的元素算成缺服务端方法。

### 5.2 分档计数

| 档 | 条数 | 占比 | 主要维度 | 主要元素族 |
|---|---:|---:|---|---|
| 本版可做 | 192 | 8.0% | 异常 78、交互 57、状态 32 | `S-configuration` 100、`S-shell` 32、`S-appearance` 28 |
| 需协议 | 1230 | 51.4% | 窄窗 331、异常 263、交互 120 | `S-personalization` 256、`S-usage` 219、`S-account` 197 |
| 需素材 | 160 | 6.7% | 异常 42、窄窗 32、记忆 23 | `S-appearance` 116、`S-pets` 44 |
| 需宿主能力 | 392 | 16.4% | 窄窗 107、异常 55、状态 49 | `S-general` 249、`S-about` 58、`S-runtime` 53 |
| 取证口径 | 421 | 17.6% | 窄窗 116、键盘 47、记忆 37 | `S-configuration` 214、`S-appearance` 59、`S-keybindings` 47 |
| **合计** | **2395** | **100.0%** | — | — |

未实现条目的维度构成（全量，供对照）：窄窗 586、异常 471、交互 228、状态 210、键盘 170、记忆 160、动画 142、高DPI 134、无障碍 77、样式 58。

### 5.3 每档代表条目（每档 10 条）

#### 本版可做（192 条）

| 需求 ID | 元素 ID | 维度 | 期望 | 判定 | 分档依据 |
|---|---|---|---|---|---|
| R-APP-028 | S-appearance-03 | 异常 | 角标回滚到原主题 | 映射到位无标记 | 本地偏好 / 本地草稿 / 已有事实 |
| R-KEY-017 | S-keybindings-03 | 状态 | 空，占位「搜索快捷键…」 | 映射到位无标记 | 本地偏好 / 本地草稿 / 已有事实 |
| R-CFG-019 | S-configuration-03 | 状态 | 图标不变色 | 映射到位无标记 | 本地偏好 / 本地草稿 / 已有事实 |
| R-PRM-036 | S-permissions-05 | 异常 | 转错误 + 刷新 | 映射到位无标记 | 本地偏好 / 本地草稿 / 已有事实 |
| R-ARC-013 | S-archived-02 | 异常 | 提示 | 映射到位无标记 | 本地偏好 / 本地草稿 / 已有事实 |
| R-SHL-023 | S-shell-03 | 异常 | 导航项仍可点 | 映射到位无标记 | 本地偏好 / 本地草稿 / 已有事实 |
| R-APP-038 | S-appearance-04 | 异常 | 保留同样高度的底色块 | 映射到位无标记 | 本地偏好 / 本地草稿 / 已有事实 |
| R-APP-047 | S-appearance-05 | 异常 | 角标回滚 | 映射到位无标记 | 本地偏好 / 本地草稿 / 已有事实 |
| R-APP-055 | S-appearance-06 | 异常 | 回落为浅色并保持选中本卡 | 映射到位无标记 | 本地偏好 / 本地草稿 / 已有事实 |
| R-APP-065 | S-appearance-07 | 异常 | 用纯色圆点兜底 | 映射到位无标记 | 本地偏好 / 本地草稿 / 已有事实 |

#### 需协议（1230 条）

| 需求 ID | 元素 ID | 维度 | 期望 | 判定 | 分档依据 |
|---|---|---|---|---|---|
| R-ACC-002 | S-account-01 | 状态 | 标题基线在内容列顶下方 33px 处，左缘与内容列左缘 257 对齐 | 映射到位无标记 | 元素族在服务端 |
| R-USG-004 | S-usage-01 | 交互 | 标题随内容滚出视口，无吸顶 | 映射到位无标记 | 元素族在服务端 |
| R-PER-004 | S-personalization-01 | 交互 | 随内容滚出，无吸顶 | 映射到位无标记 | 元素族在服务端 |
| R-MOD-003 | S-modality-01 | 状态 | 标题仍显示，右侧带旋转图标 | 映射到位无标记 | 元素族在服务端 |
| R-MCP-005 | S-mcp-01 | 异常 | 回落中文 | 映射到位无标记 | 元素族在服务端 |
| R-GIT-002 | S-git-01 | 样式 | 与其它节同级，y=33 | 无渲染点 | 元素族在服务端 |
| R-HOK-003 | S-hooks-01 | 关联 | 同行右端为了解更多与刷新 | 映射到位无标记 | 元素族在服务端 |
| R-PLG-005 | S-pluginSettings-01 | 异常 | 回落中文 | 映射到位无标记 | 元素族在服务端 |
| R-EXT-003 | S-externalAgents-01 | 关联 | 下 4 | 映射到位无标记 | 元素族在服务端 |
| R-ACC-006 | S-account-01 | 交互 | 标题随内容滚出视口，不留吸顶头部；滚回顶部后标题回到 y=33 | 映射到位无标记 | 元素族在服务端 |

#### 需素材（160 条）

| 需求 ID | 元素 ID | 维度 | 期望 | 判定 | 分档依据 |
|---|---|---|---|---|---|
| R-APP-030 | S-appearance-03 | 高DPI | 预览图清晰、边框 1px | 映射到位无标记 | 期望列素材词 / 皮肤·宠物元素 |
| R-PET-002 | S-pets-01 | 样式 | 与其它节同级，y=33 | 无渲染点 | 期望列素材词 / 皮肤·宠物元素 |
| R-APP-440 | S-appearance-50 | 异常 | 回落中文 | 映射到位无标记 | 期望列素材词 / 皮肤·宠物元素 |
| R-APP-441 | S-appearance-50 | 窄窗 | 单行 | 映射到位无标记 | 期望列素材词 / 皮肤·宠物元素 |
| R-APP-446 | S-appearance-51 | 动画 | 静态骨架，不扫光；减少动效下静态 | 映射到位无标记 | 期望列素材词 / 皮肤·宠物元素 |
| R-APP-449 | S-appearance-51 | 窄窗 | 骨架两列换行 | 映射到位无标记 | 期望列素材词 / 皮肤·宠物元素 |
| R-APP-457 | S-appearance-52 | 异常 | 卡显示异常标记（S-appearance-58），点击无效果 | 映射到位无标记 | 期望列素材词 / 皮肤·宠物元素 |
| R-APP-458 | S-appearance-52 | 记忆 | 仍为该皮肤 | 映射到位无标记 | 期望列素材词 / 皮肤·宠物元素 |
| R-APP-465 | S-appearance-53 | 异常 | 显示首字母块（S-appearance-57） | 映射到位无标记 | 期望列素材词 / 皮肤·宠物元素 |
| R-APP-466 | S-appearance-53 | 记忆 | 保持最后选择 | 映射到位无标记 | 期望列素材词 / 皮肤·宠物元素 |

#### 需宿主能力（392 条）

| 需求 ID | 元素 ID | 维度 | 期望 | 判定 | 分档依据 |
|---|---|---|---|---|---|
| R-GEN-006 | S-general-01 | 记忆 | 回到首屏 y=33 | 映射到位无标记 | 宿主接口词 / 元素族 |
| R-RUN-006 | S-runtime-01 | 记忆 | 回 y=33 | 映射到位无标记 | 宿主接口词 / 元素族 |
| R-NTF-002 | S-notifications-01 | 样式 | 与其它节标题同级，y=33 | 无渲染点 | 宿主接口词 / 元素族 |
| R-ABT-006 | S-about-01 | 窄窗 | 单行 | 映射到位无标记 | 宿主接口词 / 元素族 |
| R-GEN-007 | S-general-01 | 窄窗 | 单行 | 映射到位无标记 | 宿主接口词 / 元素族 |
| R-GEN-015 | S-general-02 | 窄窗 | 与开关仍同行 | 映射到位无标记 | 宿主接口词 / 元素族 |
| R-GEN-023 | S-general-03 | 窄窗 | 折三行 | 映射到位无标记 | 宿主接口词 / 元素族 |
| R-GEN-026 | S-general-04 | 交互 | 翻转；开启后运行期间阻止休眠、屏保不启动 | 映射到位无标记 | 宿主接口词 / 元素族 |
| R-GEN-028 | S-general-04 | 动画 | 滑块 120ms 位移 | 映射到位无标记 | 宿主接口词 / 元素族 |
| R-GEN-029 | S-general-04 | 记忆 | 保持开态 | 映射到位无标记 | 宿主接口词 / 元素族 |

#### 取证口径（421 条）

| 需求 ID | 元素 ID | 维度 | 期望 | 判定 | 分档依据 |
|---|---|---|---|---|---|
| R-APP-007 | S-appearance-01 | 异常 | 回落中文 | 映射到位无标记 | 无 i18n 层 |
| R-KEY-006 | S-keybindings-01 | 记忆 | 回 y=33 | 映射到位无标记 | 横向维度 |
| R-CFG-006 | S-configuration-01 | 窄窗 | 左栏隐藏，详情整宽并出现返回 | 映射到位无标记 | 横向维度 |
| R-PRM-006 | S-permissions-01 | 记忆 | 回 y=33 | 映射到位无标记 | 横向维度 |
| R-ARC-005 | S-archived-01 | 异常 | 回落中文 | 映射到位无标记 | 无 i18n 层 |
| R-SHL-012 | S-shell-02 | 高DPI | 线条清晰 | 映射到位无标记 | 横向维度 |
| R-APP-014 | S-appearance-02 | 异常 | 回落中文 | 映射到位无标记 | 无 i18n 层 |
| R-APP-015 | S-appearance-02 | 窄窗 | 单行 | 映射到位无标记 | 横向维度 |
| R-APP-026 | S-appearance-03 | 动画 | 选中角标 120ms 淡入，主题色 200ms 过渡 | 映射到位无标记 | 横向维度 |
| R-APP-031 | S-appearance-03 | 键盘 | 可聚焦并切换 | 映射到位无标记 | 横向维度 |

### 5.4 本轮做掉的条目

登记 7 条：现状为「已实现」4 条，不一致 3 条。登记表在 `scripts/reconcile-triage-done.json`，重跑时逐条核对状态列，做没做不看口头说明。

**「不一致」不等于没做**：§4 的判据只看「元素有没有渲染点 + 函数体里有没有该维度的标记」，不看动作有没有接上写通道。这一轮改的是写通道与本端落盘（改键落盘、权限规则真写入、归档行读事实并接恢复/删除），渲染点与标记本来就在，所以有登记的条目现状仍是「未实现（映射到位但没有该维度的标记）」——那一条缺的是「记忆」「异常」这类横向维度的标记，不是行为。清单里每条都写了证明它的测试名，做没做按测试核。

| 需求 ID | 实现点 | 证明它的测试 | 现状 |
|---|---|---|---|
| R-KEY-055 | 重置写本地偏好：删掉这一项后重启仍是出厂值 | TestKeybindingResetDropsOverride / TestKeybindingPreferencesPersist | 仍为未实现（映射到位无标记） |
| R-KEY-072 | 改键落盘 Preferences.Keybindings，重启后按新键渲染并按新键判定 | TestKeybindingEditPersistsToPreferences / TestKeybindingDrivesMainWindowKeys | 仍为未实现（映射到位无标记） |
| R-KEY-085 | 保存失败回滚到保存前的值，行下写原因 | TestKeybindingSaveFailureRollsBack | 仍为未实现（映射到位无标记） |
| R-PRM-211 | 保存权限走 Intents.WriteCommandRules，成功写「已保存。」并重铺草稿 | TestPermissionsSaveWritesRules | 已实现（映射命中） |
| R-PRM-227 | 写入失败把服务端原文写进错误行（不加前缀） | TestPermissionsSaveFailureShowsReason | 已实现（映射命中） |
| R-ARC-057 | 项目筛选选项由归档行的工作目录生成（全部项目 / 无项目 / 各项目名） | TestArchivedListComesFromFacts | 已实现（逐字） |
| R-ARC-098 | 恢复写 ArchiveSession(id,false)，行由事实驱动消失 | TestArchivedRestoreCallsArchiveSession | 已实现（映射命中） |
