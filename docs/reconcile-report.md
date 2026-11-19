# 需求实现状态：机械对账报告

Status: 对账报告（脚本生成，可重跑；`python3 scripts/reconcile.py`）
Scope: 四份需求表的全部条目（`docs/domains/requirements/requirements-*.md`）；状态列写回的是 §4「按元素映射判定」的结论，§1–§3 是第一轮记录
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
| `requirements-settings.md` | 8325 | 670 | 875 | 5 | 633 | 6142 | 8.0% | 73.8% |
| `requirements-main.md` | 3520 | 423 | 1394 | 97 | 384 | 1222 | 12.0% | 34.7% |
| `requirements-global.md` | 2348 | 358 | 1033 | 25 | 197 | 735 | 15.2% | 31.3% |
| `requirements-panels.md` | 2152 | 341 | 186 | 56 | 170 | 1399 | 15.8% | 65.0% |
| **合计** | **16345** | **1792** | **3488** | **183** | **1384** | **9498** | **11.0%** | **58.1%** |

「未实现」里有一半以上是「期望列没有可逐字文案」造成的（无可用字面量 13925 条，其中判未实现 9077 条）；这类条目只能靠元素符号/测试或界面取证，见 §3。

## 2. 第一轮随机 40 条复核

抽样种子固定（`20261120`），每份 10 条，优先覆盖「逐字」与「未实现」两档；复核结论由脚本按证据反向核验，人工只需看标记为「有分歧」的条目。

### requirements-settings.md

| 需求 ID | 元素 ID | 维度 | 机器判定 | 证据 | 复核 |
|---|---|---|---|---|---|
| R-MCP-025 | S-mcp-04 | 内容 | 逐字 | `internal/app/extensions.go:extensionReaders.ReadToolServers`「服务器」 | 通过（字面量在 internal/app/extensions.go:extensionReaders.ReadToolServers 内） |
| R-EXT-049 | S-externalAgents-07 | 内容 | 逐字 | `internal/ui/settings_external_agents_screen.go:eaCopyUnavailDesc`「App Server 尚未实现外部代理。此模式下 Desktop 无法检测或配置它们。」 | 通过（字面量在 internal/ui/settings_external_agents_screen.go:eaCopyUnavailDesc 内） |
| R-PRM-209 | S-permissions-27 | 内容 | 逐字 | `internal/ui/settings_permissions_screen.go:permCopySaveRules`「保存权限」 | 通过（字面量在 internal/ui/settings_permissions_screen.go:permCopySaveRules 内） |
| R-MOD-073 | S-modality-10 | 内容 | 逐字 | `internal/settings/modality_realtime.go:Manager.OpenRealtime`「实时」 | 通过（字面量在 internal/settings/modality_realtime.go:Manager.OpenRealtime 内） |
| R-GEN-644 | S-general-81 | 关联 | 逐字 | `internal/ui/app.go:App.actionErrorView`「重试」 | 通过（字面量在 internal/ui/app.go:App.actionErrorView 内） |
| R-SHL-044 | S-shell-06 | 高DPI | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-PER-430 | S-personalization-45 | 内容 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-ABT-243 | S-about-31 | 状态 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-USG-337 | S-usage-42 | 内容 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-CFG-144 | S-configuration-18 | 深色 | 未实现 | — | 通过（文案/符号/测试均无命中） |

### requirements-main.md

| 需求 ID | 元素 ID | 维度 | 机器判定 | 证据 | 复核 |
|---|---|---|---|---|---|
| R-LB-903 | LB-103 | 内容 | 逐字 | `internal/account/login.go:Account.waitBrowser`「取消」 | 通过（字面量在 internal/account/login.go:Account.waitBrowser 内） |
| R-TL-668 | TL-68 | 状态 | 逐字 | `internal/ui/timeline_compaction.go:compactionWord`「已取消」 | 通过（字面量在 internal/ui/timeline_compaction.go:compactionWord 内） |
| R-TL-383 | TL-39 | 无障碍 | 逐字 | `internal/ui/timeline.go:App.userActions`「从此处分叉」 | 通过（字面量在 internal/ui/timeline.go:App.userActions 内） |
| R-LB-741 | LB-47 | 记忆 | 逐字 | `internal/ui/sidebar.go:App.workspaceGroup`「暂无会话」 | 通过（字面量在 internal/ui/sidebar.go:App.workspaceGroup 内） |
| R-LB-522 | LB-110 | 样式 | 逐字 | `internal/ui/sidebar.go:App.sessionMenu`「在新窗口中打开」 | 通过（字面量在 internal/ui/sidebar.go:App.sessionMenu 内） |
| R-LB-575 | LB-87 | 关联 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-LB-353 | LB-55 | 记忆 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-IN-203 | IN-14 | 高DPI | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-LB-230 | LB-34 | 无障碍 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-IN-228 | IN-17 | 内容 | 未实现 | — | 通过（文案/符号/测试均无命中） |

### requirements-global.md

| 需求 ID | 元素 ID | 维度 | 机器判定 | 证据 | 复核 |
|---|---|---|---|---|---|
| R-GR-116 | G-F2-06 | 状态 | 逐字 | `internal/account/login.go:Account.waitBrowser`「取消」 | 通过（字面量在 internal/account/login.go:Account.waitBrowser 内） |
| R-GI-158 | G-H2-08 | 状态 | 逐字 | `internal/account/login.go:Account.waitBrowser`「取消」 | 通过（字面量在 internal/account/login.go:Account.waitBrowser 内） |
| R-GI-007 | G-H1-01 | 内容 | 逐字 | `internal/ui/plugin_install.go:pluginDialogTitle`「从 Git 添加插件」 | 通过（字面量在 internal/ui/plugin_install.go:pluginDialogTitle 内） |
| R-GR-018 | G-F1-03 | 内容 | 逐字 | `internal/ui/rename_dialog.go:renameCopySessionDesc`「会话名称」 | 通过（字面量在 internal/ui/rename_dialog.go:renameCopySessionDesc 内） |
| R-GQ-166 | G-D1-20 | 内容 | 逐字 | `internal/ui/question.go:questionCopyNext`「下一题」 | 通过（字面量在 internal/ui/question.go:questionCopyNext 内） |
| R-GS-189 | G-A2-11c | 状态 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-GI-447 | G-H7-03 | 无障碍 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-GS-092 | G-A2-01 | 异常 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-GM-213 | G-B2-07 | 关联 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-GS-048 | G-A1-05 | 样式 | 未实现 | — | 通过（文案/符号/测试均无命中） |

### requirements-panels.md

| 需求 ID | 元素 ID | 维度 | 机器判定 | 证据 | 复核 |
|---|---|---|---|---|---|
| R-RA-130 | A-16 | 状态 | 逐字 | `internal/ui/rightpanel_changes.go:App.rpChangeMenu`「用默认应用打开」 | 通过（字面量在 internal/ui/rightpanel_changes.go:App.rpChangeMenu 内） |
| R-SC-007 | S-01 | 窄窗 | 逐字 | `internal/ui/app.go:App.actionErrorView`「重试」 | 通过（字面量在 internal/ui/app.go:App.actionErrorView 内） |
| R-TK-015 | K-02 | 无障碍 | 逐字 | `internal/ui/rightpanel_tasks.go:App.tasksPanel`「唤醒」 | 通过（字面量在 internal/ui/rightpanel_tasks.go:App.tasksPanel 内） |
| R-RD-885 | D-96 | 状态 | 逐字 | `internal/ui/rightpanel_changes.go:App.rpChangeMenu`「复制路径」 | 通过（字面量在 internal/ui/rightpanel_changes.go:App.rpChangeMenu 内） |
| R-RD-494 | D-53 | 状态 | 逐字 | `internal/ui/rightpanel_changes.go:App.rpChangeGroups`「全部暂存」 | 通过（字面量在 internal/ui/rightpanel_changes.go:App.rpChangeGroups 内） |
| R-RD-274 | D-30 | 状态 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-RD-046 | D-03 | 记忆 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-RD-807 | D-83 | 无障碍 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-RF-347 | F-41 | 异常 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-RD-823 | D-88 | 无障碍 | 未实现 | — | 通过（文案/符号/测试均无命中） |

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
| `requirements-settings.md` | 8325 | 670 | 4578 | 2040 | 404 | 633 | 55.0% |
| `requirements-main.md` | 3520 | 423 | 1484 | 690 | 539 | 384 | 42.2% |
| `requirements-global.md` | 2348 | 358 | 784 | 428 | 581 | 197 | 33.4% |
| `requirements-panels.md` | 2152 | 341 | 918 | 344 | 379 | 170 | 42.7% |
| **合计** | **16345** | **1792** | **7764** | **3502** | **1903** | **1384** | **47.5%** |

去掉逐字与负向后可判 13169 条：元素映射到位 11266 条（85.5%），其中缺该维度标记 3502 条；找不到渲染点 1903 条（14.5%）。

「无渲染点」是映射的缺口，不是实现的结论：元素 ID 没在 Go 注释里标注，就查不到渲染函数。按域看元素映射覆盖率：

| 文件 | 元素数 | 有渲染点 | 覆盖率 |
|---|---:|---:|---:|
| `requirements-settings.md` | 992 | 933 | 94.1% |
| `requirements-main.md` | 344 | 272 | 79.1% |
| `requirements-global.md` | 276 | 187 | 67.8% |
| `requirements-panels.md` | 244 | 184 | 75.4% |

缺标注最多的元素族（按条数）：`IN` 208 条、`TL` 203 条、`K` 176 条、`S-configuration` 160 条、`G-A2` 147 条、`LB` 129 条、`D` 122 条、`S-general` 112 条。

### 4.3 按维度

| 维度 | 条数 | 逐字 | 映射命中 | 映射到位无标记 | 无渲染点 | 负向 | 标记组来源 |
|---|---:|---:|---:|---:|---:|---:|---|
| 状态 | 2570 | 463 | 952 | 326 | 275 | 554 | 本轮补充 |
| 样式 | 1984 | 50 | 1544 | 77 | 300 | 13 | 领导给定 |
| 内容 | 1924 | 671 | 876 | 12 | 230 | 135 | 领导给定 |
| 无障碍 | 1893 | 285 | 1289 | 58 | 242 | 19 | 领导给定 |
| 异常 | 1745 | 85 | 567 | 614 | 171 | 308 | 领导给定 |
| 交互 | 1436 | 63 | 747 | 294 | 180 | 152 | 领导给定 |
| 关联 | 1380 | 133 | 836 | 132 | 159 | 120 | 本轮补充 |
| 窄窗 | 858 | 11 | 50 | 681 | 88 | 28 | 领导给定 |
| 记忆 | 688 | 20 | 57 | 486 | 98 | 27 | 领导给定 |
| 深色 | 569 | 0 | 538 | 3 | 28 | 0 | 本轮补充 |
| 键盘 | 482 | 5 | 159 | 263 | 49 | 6 | 本轮补充 |
| 高DPI | 292 | 0 | 19 | 246 | 24 | 3 | 本轮补充 |
| 动画 | 285 | 1 | 25 | 213 | 38 | 8 | 领导给定 |
| 性能 | 88 | 0 | 10 | 62 | 13 | 3 | 本轮补充 |
| 语言 | 69 | 4 | 57 | 0 | 4 | 4 | 本轮补充 |
| 输入法 | 30 | 0 | 2 | 26 | 0 | 2 | 本轮补充 |
| 多窗口 | 14 | 0 | 8 | 4 | 2 | 0 | 本轮补充 |
| 多语言 | 13 | 0 | 12 | 0 | 0 | 1 | 本轮补充 |
| 位置 | 9 | 1 | 7 | 0 | 1 | 0 | 本轮补充 |
| 拖拽 | 4 | 0 | 0 | 4 | 0 | 0 | 本轮补充 |
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

逐字 1792 条中：字面量落在元素映射函数内 1247 条；元素有映射但字面量不在映射函数内 345 条；元素没有映射点 200 条。

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
| R-ABT-127 | S-about-16 | 键盘 | 映射到位无标记 | `internal/ui/settings_about_screen.go:abtUpdateAction` | Sprintf, Row, Shrink, Children, ButtonBase, Height, Radius, Disabled | 待复核 |
| R-KEY-193 | S-keybindings-25 | 状态 | 映射到位无标记 | `internal/ui/settings_keybindings_screen.go:keybindingButton` | Theme, ButtonBase, Shrink, Height, MinWidth, Radius, Label, Background | 待复核 |
| R-USG-478 | S-usage-59 | 状态 | 映射到位无标记 | `internal/ui/settings_usage_screen.go:renderSettingsUsageScreen` | Theme, Column, Grow, Children, Scroll, Gap | 有分歧（映射错位）：映射到用量屏根，用量屏只渲染标题与不可用卡；S-usage-59 模型用量在代码里没有渲染点 |
| R-PER-604 | S-personalization-66 | 异常 | 映射到位无标记 | `internal/ui/settings_personalization_screen.go:renderSettingsPersonalizationScreen` | Theme, Column, Grow, Children, Scroll, Gap, Text, TextColor | 确认：该屏无长度校验与保存提示（psCustomInstructionsSection 只有说明与打开文件按钮）；映射点偏到屏根 |
| R-APP-209 | S-appearance-24 | 动画 | 映射到位无标记 | `internal/ui/settings_appearance_screen.go:appearanceFontSection` | Theme, Column, Shrink, Gap, Children | 有分歧（负向期望）：期望是「无逐项淡入」，函数里没有动效标记恰好满足；机器按未实现计是口径问题 |
| R-PET-154 | S-pets-20 | 交互 | 映射到位无标记 | `internal/ui/settings_pets_screen.go:petsChooseSection` | Column, Shrink, Gap, Children | 待复核 |
| R-CFG-599 | S-configuration-75 | 窄窗 | 映射到位无标记 | `internal/ui/settings_provider_form.go:App.configFormModels` | Theme, Column, Shrink, Gap, Children, Row, AlignItems, Text | 待复核 |
| R-USG-011 | S-usage-02 | 内容 | 映射命中 | `internal/ui/settings_usage_screen.go:usUnavailableCard` 标记 文案「不可用」等 2 条 | Row, Shrink, Wrap, AlignItems, Padding, Role, Label, Children | 待复核 |
| R-MOD-197 | S-modality-25 | 关联 | 映射命中 | `internal/ui/settings_modality_screen.go:renderModality` 标记 Screen | Theme, Column, Grow, Children, Scroll, Text, TextColor, Level | 待复核 |
| R-GEN-611 | S-general-77 | 样式 | 无渲染点 | — | — | 待复核 |

#### requirements-main.md

| 需求 ID | 元素 ID | 维度 | 判定 | 映射 | 函数体信号 | 人工复核 |
|---|---|---|---|---|---|---|
| R-IN-843 | IN-66 | 记忆 | 映射到位无标记 | `internal/ui/approval.go:App.approvalCard` | Theme, Column, WidthPercent, MaxWidth, Padding, Background, Border, Role | 待复核 |
| R-TL-061 | TL-07 | 高DPI | 映射到位无标记 | `internal/ui/timeline.go:App.round` | Column, Shrink, Gap, Children | 待复核 |
| R-TL-582 | TL-58 | 窄窗 | 映射到位无标记 | `internal/ui/timeline.go:App.retryNotice` | Theme, Row, Gap, AlignItems, Children, Text, Sprintf, Button | 待复核 |
| R-IN-114 | IN-09 | 记忆 | 映射到位无标记 | `internal/ui/composer.go:App.modelSelector` | Theme, MenuBase, Height, Padding, Radius, Tooltip, Hovered, Background | 待复核 |
| R-NT-024 | NT-02 | 高DPI | 映射到位无标记 | `internal/ui/composer.go:App.emptyThread` | Theme, Box, Grow, Center, Padding, Label, Children, Column | 待复核 |
| R-LB-156 | LB-23 | 异常 | 映射到位无标记 | `internal/ui/sidebar.go:App.sidebar` | Shortcut, Theme, Column, Width, Shrink, Role, Focusable, TabStop | 待复核 |
| R-IN-028 | IN-01 | 记忆 | 映射到位无标记 | `internal/ui/composer.go:App.composerCard` | Theme, Column, WidthPercent, MaxWidth, MinHeight, Border, Clip, Label | 待复核 |
| R-LB-092 | LB-13 | 样式 | 映射命中 | `internal/ui/sidebar.go:App.sidebarEntries` 标记 Padding | Column, Shrink, Padding, Gap, Children | 待复核 |
| R-TB-045 | TB-06 | 样式 | 映射命中 | `internal/ui/search.go:App.sessionSearchOverlay` 标记 Width | Theme, Size, Overlay, Box, Absolute, Fill, AccessHidden, OnClick | 待复核 |
| R-TL-275 | TL-27 | 关联 | 无渲染点 | — | — | 待复核 |

#### requirements-global.md

| 需求 ID | 元素 ID | 维度 | 判定 | 映射 | 函数体信号 | 人工复核 |
|---|---|---|---|---|---|---|
| R-GS-375 | G-A5-02 | 关联 | 映射到位无标记 | `internal/ui/search.go:App.searchReveal` | IsZero, Now, Sub, ScrollIntoView, AnimationFrame | 待复核 |
| R-AP-277 | G-C2-06 | 异常 | 映射到位无标记 | `internal/ui/approval.go:approvalActionsFor` | — | 待复核 |
| R-AP-402 | G-C4-03 | 记忆 | 映射到位无标记 | `internal/ui/approval_panels.go:approvalCommandPanel` | TrimSpace | 待复核 |
| R-AP-469 | G-C4-11 | 样式 | 映射到位无标记 | `internal/ui/approval_panels.go:approvalSearchPanel` | — | 待复核 |
| R-GC-172 | G-E4-06 | 键盘 | 映射到位无标记 | `internal/ui/confirm.go:App.confirmPrimaryText` | Sprintf | 待复核 |
| R-GM-137 | G-B1-17 | 无障碍 | 映射到位无标记 | `internal/ui/slash_commands.go:App.slashCommands` | — | 待复核 |
| R-AP-148 | G-C1-14 | 窄窗 | 映射到位无标记 | `internal/ui/approval.go:App.handoffBlock` | Theme, Column, WidthPercent, Padding, Gap, Radius, Label, Text | 待复核 |
| R-GQ-062 | G-D1-08 | 样式 | 映射命中 | `internal/ui/question.go:App.questionBody` 标记 Height | Theme, Scroll, Shrink, Height, Children, Column, WidthPercent, Gap | 待复核 |
| R-AP-179 | G-C1-18 | 样式 | 映射命中 | `internal/ui/approval.go:approvalActionsFor` 标记 Type | — | 待复核 |
| R-GS-017 | G-A1-02a | 交互 | 无渲染点 | — | — | 待复核 |

#### requirements-panels.md

| 需求 ID | 元素 ID | 维度 | 判定 | 映射 | 函数体信号 | 人工复核 |
|---|---|---|---|---|---|---|
| R-TK-105 | K-13 | 样式 | 映射到位无标记 | `internal/ui/rightpanel_tasks.go:App.taskStopButton` | Theme | 待复核 |
| R-RD-155 | D-15 | 记忆 | 映射到位无标记 | `internal/ui/rightpanel_changes.go:App.rpScopeBar` | Theme, Row, Height, Shrink, Padding, Gap, Children, MenuBase | 待复核 |
| R-RD-483 | D-51 | 记忆 | 映射到位无标记 | `internal/ui/rightpanel_changes.go:App.rpChangeGroups` | Sprintf | 待复核 |
| R-RD-710 | D-73 | 性能 | 映射到位无标记 | `internal/ui/rightpanel_changes.go:App.rpRevertDialog` | Modal, Theme, Column, Gap, Padding, Width, Label, Radius | 待复核 |
| R-SC-056 | S-07 | 交互 | 映射到位无标记 | `internal/ui/rightpanel.go:panelEmpty` | Theme, Column, Grow, Center, Padding, Children, Text, TextColor | 待复核 |
| R-RF-230 | F-27 | 拖拽 | 映射到位无标记 | `internal/ui/rightpanel_files.go:App.rpFileRow` | Theme, Sprintf, ButtonBase, Height, Padding, AlignItems, Radius, Level | 待复核 |
| R-RD-681 | D-71 | 性能 | 映射到位无标记 | `internal/ui/rightpanel_changes.go:App.rpChangeRow` | Theme, Sprintf, Row, Height, Padding, AlignItems, Role, Hovered | 待复核 |
| R-RF-148 | F-17 | 无障碍 | 映射命中 | `internal/ui/rightpanel_files.go:App.rpDirRow` 标记 Role | Theme, ButtonBase, Height, Padding, AlignItems, Radius, Expanded, Level | 待复核 |
| R-RF-007 | F-01 | 键盘 | 映射命中 | `internal/ui/rightpanel_files.go:App.rpFilterRow` 标记 Shortcut | Theme, Column, Shrink, Padding, Children, Row, Height, Gap | 确认：rpFilterRow 命中 c.Shortcut(0, gui.KeyEscape) 与清空查询，键盘通路在 |
| R-RD-862 | D-93 | 交互 | 无渲染点 | — | — | 待复核 |

人工复核 4 条：与机器判定一致 2 条，有分歧 2 条。分歧构成：映射错位 1、负向期望 1。

### 4.6 「缺标记」里有多少是标记组漏词

对「映射到位但没有该维度的标记」的条目，再查一遍同维度在 mygo 里的其它写法（`DIM_ADJACENT`，只作参考、不改判定）。这类条目更像标记组不够宽，不像元素缺实现。

| 维度 | 缺标记条数 | 其中函数体有同类写法 | 例子 |
|---|---:|---:|---|
| 窄窗 | 681 | 619（90.9%） | R-ACC-033 `Wrap` |
| 异常 | 614 | 123（20.0%） | R-ACC-029 `Banner` |
| 记忆 | 486 | 193（39.7%） | R-ACC-354 `.accountEmailOpen =` |
| 状态 | 326 | 32（9.8%） | R-RUN-033 `Empty@rtLiveSection` |
| 交互 | 294 | 134（45.6%） | R-ACC-074 `Selectable` |
| 键盘 | 263 | 10（3.8%） | R-ACC-603 `Focus` |
| 高DPI | 246 | 186（75.6%） | R-ACC-027 `Size@fontStep.style` |
| 动画 | 213 | 89（41.8%） | R-ACC-017 `Opacity@accButton` |
| 关联 | 132 | 6（4.5%） | R-IN-317 `Close@App.shellRunPanel` |
| 样式 | 77 | 8（10.4%） | R-TL-006 `Fill` |
| 性能 | 62 | 0（0.0%） | — |
| 无障碍 | 58 | 0（0.0%） | — |
| 输入法 | 26 | 0（0.0%） | — |
| 内容 | 12 | 0（0.0%） | — |
| 多窗口 | 4 | 0（0.0%） | — |
| 拖拽 | 4 | 0（0.0%） | — |
| 深色 | 3 | 0（0.0%） | — |
| 数据量 | 1 | 0（0.0%） | — |

### 4.7 最该先修的 10 个「映射到位但缺标记」元素

排序用「真缺」＝映射到位、缺该维度的标记，且函数体里连同维度的其它写法（§4.6 的 `DIM_ADJACENT`）也没有；「缺标记条数」是全量。维度缺口列是该元素缺标记的维度构成（按条数）。

| # | 元素 ID | 名称 | 真缺条数 | 缺标记条数 | 覆盖需求 | 维度缺口 | 映射点 |
|---:|---|---|---:|---:|---:|---|---|
| 1 | IN-66 | （无文案） | 14 | 15 | 32 | 关联 4、键盘 3、记忆 3、动画 1 | `internal/ui/approval.go:App.approvalCard` |
| 2 | G-C1-01 | （无文案）卡片容器 | 12 | 13 | 30 | 关联 4、动画 2、记忆 2、交互 1 | `internal/ui/approval.go:App.approvalCard` |
| 3 | TL-08 | （无文案） | 12 | 12 | 15 | 状态 3、交互 3、样式 1、动画 1 | `internal/ui/timeline.go:App.backToLatest` |
| 4 | IN-01 | （无文案） | 10 | 10 | 35 | 状态 5、动画 2、记忆 2、性能 1 | `internal/ui/composer.go:App.composerCard` |
| 5 | D-77 | {count} 个文件改动 | 9 | 10 | 26 | 关联 3、窄窗 1、记忆 1、交互 1 | `internal/ui/rightpanel_changes.go:App.rpChangesFooter` |
| 6 | G-C4-01 | （无文案）容器 | 9 | 11 | 16 | 状态 3、交互 2、关联 2、记忆 1 | `internal/ui/approval_panels.go:approvalInputPanel` |
| 7 | TL-67 | 已压缩 / 自动压缩 + `(部分)` | 9 | 9 | 14 | 状态 2、交互 1、样式 1、动画 1 | `internal/ui/timeline_compaction.go:compactionWord` |
| 8 | G-C1-20 | 「始终允许此命令」 / 「允许此会话期间访问」  | 8 | 8 | 16 | 状态 2、关联 1、无障碍 1、键盘 1 | `internal/ui/approval.go:approvalLongTermCopy` |
| 9 | IN-35 | （排队消息面板） | 8 | 8 | 14 | 状态 5、关联 1、异常 1、记忆 1 | `internal/ui/composer_panels.go:App.queuePanel` |
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
