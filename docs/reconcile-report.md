# 需求实现状态：机械对账报告

Status: 对账报告（脚本生成，可重跑；`python3 scripts/reconcile.py`）
Scope: 四份需求表的全部条目（`docs/domains/requirements/requirements-*.md`）；状态列写回的是 §4「按元素映射判定」的结论，§1–§3 是第一轮记录，§5 是设置 19 节的五档分流与本轮做掉的条目，§6 是主窗口与全局面板的五档分流（核对状态列，不看口头说明）
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
| `requirements-settings.md` | 8325 | 681 | 1057 | 6 | 633 | 5948 | 8.2% | 71.4% |
| `requirements-main.md` | 3520 | 489 | 1732 | 106 | 384 | 809 | 13.9% | 23.0% |
| `requirements-global.md` | 2348 | 372 | 1213 | 74 | 197 | 492 | 15.8% | 21.0% |
| `requirements-panels.md` | 2152 | 352 | 414 | 171 | 170 | 1045 | 16.4% | 48.6% |
| **合计** | **16345** | **1894** | **4416** | **357** | **1384** | **8294** | **11.6%** | **50.7%** |

「未实现」里有一半以上是「期望列没有可逐字文案」造成的（无可用字面量 13923 条，其中判未实现 7996 条）；这类条目只能靠元素符号/测试或界面取证，见 §3。

## 2. 第一轮随机 40 条复核

抽样种子固定（`20261120`），每份 10 条，优先覆盖「逐字」与「未实现」两档；复核结论由脚本按证据反向核验，人工只需看标记为「有分歧」的条目。

### requirements-settings.md

| 需求 ID | 元素 ID | 维度 | 机器判定 | 证据 | 复核 |
|---|---|---|---|---|---|
| R-RUN-181 | S-runtime-23 | 关联 | 逐字 | `internal/ui/settings_runtime_screen.go:rtCopyInstall`「下载安装」 | 通过（字面量在 internal/ui/settings_runtime_screen.go:rtCopyInstall 内） |
| R-PLG-161 | S-pluginSettings-21 | 内容 | 逐字 | `internal/ui/app.go:App.persistPreferencesThen`「保存」 | 通过（字面量在 internal/ui/app.go:App.persistPreferencesThen 内） |
| R-PRM-089 | S-permissions-12 | 内容 | 逐字 | `internal/ui/settings_permissions_screen.go:permCopyNetworkIn`「允许沙箱命令访问网络。」 | 通过（字面量在 internal/ui/settings_permissions_screen.go:permCopyNetworkIn 内） |
| R-CFG-633 | S-configuration-80 | 状态 | 逐字 | `internal/ui/settings_configuration_screen.go:App.configModelList`「没有匹配的模型」 | 通过（字面量在 internal/ui/settings_configuration_screen.go:App.configModelList 内） |
| R-GEN-594 | S-general-75 | 内容 | 逐字 | `internal/ui/settings_general_screen.go:generalCleanupDialog`「0 B」 | 通过（字面量在 internal/ui/settings_general_screen.go:generalCleanupDialog 内） |
| R-SHL-256 | S-shell-32 | 深色 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-PER-430 | S-personalization-45 | 内容 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-SHL-189 | S-shell-24 | 无障碍 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-USG-337 | S-usage-42 | 内容 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-CFG-299 | S-configuration-38 | 交互 | 未实现 | — | 通过（文案/符号/测试均无命中） |

### requirements-main.md

| 需求 ID | 元素 ID | 维度 | 机器判定 | 证据 | 复核 |
|---|---|---|---|---|---|
| R-LB-712 | LB-33 | 内容 | 逐字 | `internal/ui/sidebar.go:App.viewMenu`「折叠全部」 | 通过（字面量在 internal/ui/sidebar.go:App.viewMenu 内） |
| R-TL-285 | TL-28 | 无障碍 | 逐字 | `internal/ui/timeline_extra.go:App.mentionChip`「打开文件」 | 通过（字面量在 internal/ui/timeline_extra.go:App.mentionChip 内） |
| R-LB-869 | LB-94 | 关联 | 逐字 | `internal/ui/sidebar.go:App.accountMenu`「显示 Pet」 | 通过（字面量在 internal/ui/sidebar.go:App.accountMenu 内） |
| R-LB-613 | LB-96 | 动画 | 逐字 | `internal/ui/sidebar.go:App.accountMenu`「邀请朋友」 | 通过（字面量在 internal/ui/sidebar.go:App.accountMenu 内） |
| R-LB-361 | LB-54 | 样式 | 逐字 | `internal/ui/sidebar.go:App.workspaceGroup`「暂无会话」 | 通过（字面量在 internal/ui/sidebar.go:App.workspaceGroup 内） |
| R-NT-052 | NT-05 | 异常 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-LB-344 | LB-53 | 交互 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-NT-030 | NT-03 | 异常 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-LB-202 | LB-31 | 交互 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-LB-813 | LB-75 | 无障碍 | 未实现 | — | 通过（文案/符号/测试均无命中） |

### requirements-global.md

| 需求 ID | 元素 ID | 维度 | 机器判定 | 证据 | 复核 |
|---|---|---|---|---|---|
| R-GR-066 | G-F1-08 | 无障碍 | 逐字 | `internal/account/login.go:Account.waitBrowser`「取消」 | 通过（字面量在 internal/account/login.go:Account.waitBrowser 内） |
| R-GI-098 | G-H1-12 | 无障碍 | 逐字 | `internal/ui/plugin_install.go:pluginDialogTitle`「添加插件」 | 通过（字面量在 internal/ui/plugin_install.go:pluginDialogTitle 内） |
| R-GF-097 | G-G1-12 | 内容 | 逐字 | `internal/ui/export.go:App.exportTurnPicker`「全部保留」 | 通过（字面量在 internal/ui/export.go:App.exportTurnPicker 内） |
| R-GC-125 | G-E4-01 | 内容 | 逐字 | `internal/ui/confirm.go:confirmCopyWorkspace`「删除工作区会话」 | 通过（字面量在 internal/ui/confirm.go:confirmCopyWorkspace 内） |
| R-GQ-109 | G-D1-14 | 状态 | 逐字 | `internal/ui/question.go:questionCopyFreeform`「每行输入一个值」 | 通过（字面量在 internal/ui/question.go:questionCopyFreeform 内） |
| R-GI-169 | G-H2-09 | 异常 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-GS-188 | G-A2-11c | 异常 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-GI-161 | G-H2-08 | 样式 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-GS-052 | G-A1-05 | 关联 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-GM-235 | G-B2-09 | 样式 | 未实现 | — | 通过（文案/符号/测试均无命中） |

### requirements-panels.md

| 需求 ID | 元素 ID | 维度 | 机器判定 | 证据 | 复核 |
|---|---|---|---|---|---|
| R-RA-138 | A-17 | 状态 | 逐字 | `internal/ui/rightpanel_changes.go:App.rpChangeMenu`「在文件管理器中显示」 | 通过（字面量在 internal/ui/rightpanel_changes.go:App.rpChangeMenu 内） |
| R-TK-216 | K-27 | 样式 | 逐字 | `internal/ui/rightpanel_tasks.go:App.taskCopyCommand`「复制命令」 | 通过（字面量在 internal/ui/rightpanel_tasks.go:App.taskCopyCommand 内） |
| R-TK-001 | K-01 | 状态 | 逐字 | `internal/ui/rightpanel_tasks.go:App.tasksPanel`「后台进程」 | 通过（字面量在 internal/ui/rightpanel_tasks.go:App.tasksPanel 内） |
| R-RD-893 | D-97 | 状态 | 逐字 | `internal/ui/rightpanel_changes.go:App.rpChangeMenu`「复制相对路径」 | 通过（字面量在 internal/ui/rightpanel_changes.go:App.rpChangeMenu 内） |
| R-RD-511 | D-54 | 无障碍 | 逐字 | `internal/ui/rightpanel_changes.go:App.rpChangeGroups`「未暂存」 | 通过（字面量在 internal/ui/rightpanel_changes.go:App.rpChangeGroups 内） |
| R-RD-400 | D-42 | 交互 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-RD-123 | D-12 | 状态 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-TM-126 | T-15 | 记忆 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-RD-013 | D-01 | 状态 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-TM-144 | T-18 | 交互 | 未实现 | — | 通过（文案/符号/测试均无命中） |

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
| `requirements-settings.md` | 8325 | 681 | 4769 | 1858 | 384 | 633 | 57.3% |
| `requirements-main.md` | 3520 | 489 | 1785 | 714 | 148 | 384 | 50.7% |
| `requirements-global.md` | 2348 | 372 | 982 | 435 | 362 | 197 | 41.8% |
| `requirements-panels.md` | 2152 | 352 | 1115 | 374 | 141 | 170 | 51.8% |
| **合计** | **16345** | **1894** | **8651** | **3381** | **1035** | **1384** | **52.9%** |

去掉逐字与负向后可判 13067 条：元素映射到位 12032 条（92.1%），其中缺该维度标记 3381 条；找不到渲染点 1035 条（7.9%）。

「无渲染点」是映射的缺口，不是实现的结论：元素 ID 没在 Go 注释里标注，就查不到渲染函数。按域看元素映射覆盖率：

| 文件 | 元素数 | 有渲染点 | 覆盖率 |
|---|---:|---:|---:|
| `requirements-settings.md` | 992 | 936 | 94.4% |
| `requirements-main.md` | 344 | 326 | 94.8% |
| `requirements-global.md` | 276 | 220 | 79.7% |
| `requirements-panels.md` | 244 | 220 | 90.2% |

缺标注最多的元素族（按条数）：`S-configuration` 152 条、`S-general` 112 条、`A` 99 条、`G-B2` 72 条、`G-H2` 72 条、`NT` 62 条、`TL` 58 条、`S-mcp` 56 条。

### 4.3 按维度

| 维度 | 条数 | 逐字 | 映射命中 | 映射到位无标记 | 无渲染点 | 负向 | 标记组来源 |
|---|---:|---:|---:|---:|---:|---:|---|
| 状态 | 2570 | 504 | 1088 | 275 | 149 | 554 | 本轮补充 |
| 样式 | 1984 | 55 | 1673 | 72 | 171 | 13 | 领导给定 |
| 内容 | 1924 | 688 | 965 | 12 | 124 | 135 | 领导给定 |
| 无障碍 | 1893 | 308 | 1377 | 42 | 147 | 19 | 领导给定 |
| 异常 | 1745 | 95 | 691 | 552 | 99 | 308 | 领导给定 |
| 交互 | 1436 | 66 | 873 | 259 | 86 | 152 | 领导给定 |
| 关联 | 1380 | 136 | 928 | 132 | 64 | 120 | 本轮补充 |
| 窄窗 | 858 | 11 | 87 | 670 | 62 | 28 | 领导给定 |
| 记忆 | 688 | 20 | 76 | 529 | 36 | 27 | 领导给定 |
| 深色 | 569 | 0 | 541 | 0 | 28 | 0 | 本轮补充 |
| 键盘 | 482 | 5 | 182 | 264 | 25 | 6 | 本轮补充 |
| 高DPI | 292 | 0 | 24 | 255 | 10 | 3 | 本轮补充 |
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

逐字 1894 条中：字面量落在元素映射函数内 1400 条；元素有映射但字面量不在映射函数内 372 条；元素没有映射点 122 条。

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
| R-CFG-408 | S-configuration-51 | 窄窗 | 映射到位无标记 | `internal/ui/settings_configuration_screen.go:configUnbuiltBody` | Theme, Text, TextColor | 待复核 |
| R-USG-478 | S-usage-59 | 状态 | 映射到位无标记 | `internal/ui/settings_usage_screen.go:renderSettingsUsageScreen` | Theme, Column, Grow, Children, Scroll, Gap | 有分歧（映射错位）：映射到用量屏根，用量屏只渲染标题与不可用卡；S-usage-59 模型用量在代码里没有渲染点 |
| R-PER-604 | S-personalization-66 | 异常 | 映射到位无标记 | `internal/ui/settings_personalization_screen.go:renderSettingsPersonalizationScreen` | Theme, Column, Grow, Children, Scroll, Gap, Text, TextColor | 确认：该屏无长度校验与保存提示（psCustomInstructionsSection 只有说明与打开文件按钮）；映射点偏到屏根 |
| R-APP-323 | S-appearance-37 | 异常 | 映射到位无标记 | `internal/ui/settings_appearance_screen.go:appearanceSizeSection` | Theme, Column, Shrink, Gap, Children, Row, AlignItems, Spacer | 待复核 |
| R-HOK-103 | S-hooks-13 | 窄窗 | 映射到位无标记 | `internal/ui/settings_hooks_screen.go:App.hkSourceCard` | Row, Shrink, AlignItems, Padding, Label, Children, Column, Grow | 待复核 |
| R-MOD-446 | S-modality-56 | 异常 | 映射到位无标记 | `internal/ui/settings_modality_screen.go:mdRealtimeSection` | Column, Shrink, Margin, Children, Padding, Text, TextColor, Level | 待复核 |
| R-ACC-1064 | S-account-104 | 动画 | 映射到位无标记 | `internal/ui/settings_account_screen.go:accountCopy` | Services, WriteClipboard, Toast | 有分歧（映射错位）：映射到动作函数 accountCopy（写剪贴板 + Toast）；按下反馈与图标变对勾在按钮层 |
| R-CFG-570 | S-configuration-72 | 交互 | 映射命中 | `internal/ui/settings_provider_detail.go:App.configModelDraftPanel` 标记 Clicked | Theme, Modal, Sprintf, Row, AlignItems, Gap, Children, CheckboxStateBase | 待复核 |
| R-ACC-1008 | S-account-97 | 深色 | 映射命中 | `internal/ui/settings_account_screen.go:App.accountReferralCard` 标记 Theme | Theme, Column, Shrink, Clip, Label, Children, Row, AlignItems | 待复核 |
| R-ACC-240 | S-account-20 | 样式 | 无渲染点 | — | — | 待复核 |

#### requirements-main.md

| 需求 ID | 元素 ID | 维度 | 判定 | 映射 | 函数体信号 | 人工复核 |
|---|---|---|---|---|---|---|
| R-IN-790 | IN-74 | 键盘 | 映射到位无标记 | `internal/ui/approval.go:App.approvalActions` | Row, WidthPercent, Gap, AlignItems, Wrap, Children, Spacer, PrimaryButton | 待复核 |
| R-TL-020 | TL-02 | 窄窗 | 映射到位无标记 | `internal/ui/timeline.go:App.emptyTimeline` | Theme, Box, Grow, Center, Label, Role, Children, Column | 待复核 |
| R-TL-542 | TL-55 | 动画 | 映射到位无标记 | `internal/ui/timeline.go:App.toolDetail` | Theme, Column, Gap, PaddingStart, Children | 待复核 |
| R-IN-014 | IN-01 | 动画 | 映射到位无标记 | `internal/ui/composer.go:App.composerCard` | Theme, Column, WidthPercent, MaxWidth, MinHeight, Border, Clip, Label | 待复核 |
| R-SB-208 | SB-18 | 状态 | 映射到位无标记 | `internal/ui/statusbar.go:App.statusUsage` | Theme, MenuBase, Height, Padding, Radius, AlignItems, Label, Sprintf | 待复核 |
| R-LB-164 | LB-25 | 状态 | 映射到位无标记 | `internal/ui/sidebar.go:App.listHeader` | Theme, Row, Height, Shrink, Padding, Gap, Children, Text | 待复核 |
| R-TL-739 | TL-74 | 窄窗 | 映射到位无标记 | `internal/ui/timeline.go:App.roundOps` | Theme, Row, Grow, Gap, Children, Box, Width, Shrink | 待复核 |
| R-LB-006 | LB-01 | 记忆 | 映射命中 | `internal/ui/sidebar.go:App.sidebarHeader` 标记 persist | TitleBar, Row, Height, Shrink, DragWindow, Label, AlignItems, Gap | 待复核 |
| R-TB-045 | TB-06 | 样式 | 映射命中 | `internal/ui/search.go:App.sessionSearchOverlay` 标记 Width | Theme, Size, Overlay, Box, Absolute, Fill, AccessHidden, OnClick | 待复核 |
| R-TL-688 | TL-70 | 动画 | 无渲染点 | — | — | 待复核 |

#### requirements-global.md

| 需求 ID | 元素 ID | 维度 | 判定 | 映射 | 函数体信号 | 人工复核 |
|---|---|---|---|---|---|---|
| R-GS-242 | G-A3-02 | 动画 | 映射到位无标记 | `internal/ui/mention.go:App.pickSkeletons` | Theme, Column, Shrink, Gap, Label, Children, Box, Height | 待复核 |
| R-AP-281 | G-C2-07 | 关联 | 映射到位无标记 | `internal/ui/approval.go:approvalActionsFor` | — | 待复核 |
| R-AP-417 | G-C4-04 | 记忆 | 映射到位无标记 | `internal/ui/approval_panels.go:approvalCommandPanel` | TrimSpace | 待复核 |
| R-AP-487 | G-C4-13 | 异常 | 映射到位无标记 | `internal/ui/approval_panels.go:approvalWebPanel` | — | 待复核 |
| R-GC-127 | G-E4-01 | 样式 | 映射到位无标记 | `internal/ui/confirm.go:App.confirmTitle` | Sprintf | 待复核 |
| R-GM-109 | G-B1-14 | 交互 | 映射到位无标记 | `internal/ui/slash_commands.go:App.slashCommands` | — | 待复核 |
| R-AP-156 | G-C1-15 | 窄窗 | 映射到位无标记 | `internal/ui/approval.go:App.handoffBlock` | Theme, Column, WidthPercent, Padding, Gap, Radius, Label, Text | 待复核 |
| R-GS-248 | G-A3-02 | 交互 | 映射命中 | `internal/ui/mention.go:App.pickSkeletons` 标记 Pressed | Theme, Column, Shrink, Gap, Label, Children, Box, Height | 待复核 |
| R-AP-168 | G-C1-17 | 交互 | 映射命中 | `internal/ui/approval.go:App.approvalActions` 标记 Clicked | Row, WidthPercent, Gap, AlignItems, Wrap, Children, Spacer, PrimaryButton | 待复核 |
| R-GS-022 | G-A1-02b | 异常 | 无渲染点 | — | — | 待复核 |

#### requirements-panels.md

| 需求 ID | 元素 ID | 维度 | 判定 | 映射 | 函数体信号 | 人工复核 |
|---|---|---|---|---|---|---|
| R-TM-164 | T-20 | 异常 | 映射到位无标记 | `internal/ui/terminal.go:App.termMenu` | Item, Disabled, Subtitle, Chosen, ReadClipboard, Separator, Danger | 待复核 |
| R-RD-159 | D-16 | 关联 | 映射到位无标记 | `internal/ui/rightpanel_files.go:App.rpSkeleton` | Theme, Column, Label, Role, Children, Row, Height, Padding | 待复核 |
| R-RD-477 | D-50 | 性能 | 映射到位无标记 | `internal/ui/rightpanel_changes.go:App.rpChangeGroups` | Sprintf | 待复核 |
| R-RD-707 | D-73 | 窄窗 | 映射到位无标记 | `internal/ui/rightpanel_changes.go:App.rpRevertDialog` | Modal, Theme, Column, Gap, Padding, Width, Label, Radius | 待复核 |
| R-TK-214 | K-27 | 交互 | 映射到位无标记 | `internal/ui/rightpanel_tasks.go:App.taskDetail` | Theme, Scroll, Grow, Children, Column, Padding, Gap, Row | 待复核 |
| R-RF-238 | F-27 | 记忆 | 映射到位无标记 | `internal/ui/rightpanel_files.go:App.rpFileRow` | Theme, Sprintf, ButtonBase, Height, Padding, AlignItems, Radius, Level | 待复核 |
| R-RD-671 | D-70 | 记忆 | 映射到位无标记 | `internal/ui/rightpanel_changes.go:App.rpChangeRow` | Theme, Sprintf, Row, Height, Padding, AlignItems, Role, Background | 待复核 |
| R-RF-281 | F-33 | 关联 | 映射命中 | `internal/ui/rightpanel_files.go:App.rpFileRow` 标记 Select | Theme, Sprintf, ButtonBase, Height, Padding, AlignItems, Radius, Level | 待复核 |
| R-RF-012 | F-01 | 异常 | 映射命中 | `internal/ui/rightpanel_files.go:App.rpFilterRow` 标记 erR | Theme, Column, Shrink, Padding, Children, Row, Height, Gap | 待复核 |
| R-RA-035 | A-04 | 异常 | 无渲染点 | — | — | 待复核 |

人工复核 3 条：与机器判定一致 1 条，有分歧 2 条。分歧构成：映射错位 2。

### 4.6 「缺标记」里有多少是标记组漏词

对「映射到位但没有该维度的标记」的条目，再查一遍同维度在 mygo 里的其它写法（`DIM_ADJACENT`，只作参考、不改判定）。这类条目更像标记组不够宽，不像元素缺实现。

| 维度 | 缺标记条数 | 其中函数体有同类写法 | 例子 |
|---|---:|---:|---|
| 窄窗 | 670 | 608（90.7%） | R-ACC-033 `Wrap` |
| 异常 | 552 | 121（21.9%） | R-ACC-029 `Banner` |
| 记忆 | 529 | 219（41.4%） | R-ACC-354 `.accountEmailOpen =` |
| 状态 | 275 | 31（11.3%） | R-RUN-033 `Empty@rtLiveSection` |
| 键盘 | 264 | 7（2.7%） | R-ACC-603 `Focus` |
| 交互 | 259 | 130（50.2%） | R-ACC-074 `Selectable` |
| 高DPI | 255 | 189（74.1%） | R-ACC-027 `Size@fontStep.style` |
| 动画 | 224 | 102（45.5%） | R-ACC-017 `Opacity@accButton` |
| 关联 | 132 | 4（3.0%） | R-IN-482 `Close` |
| 样式 | 72 | 8（11.1%） | R-TL-006 `Fill` |
| 性能 | 62 | 0（0.0%） | — |
| 无障碍 | 42 | 0（0.0%） | — |
| 输入法 | 26 | 0（0.0%） | — |
| 内容 | 12 | 0（0.0%） | — |
| 多窗口 | 4 | 0（0.0%） | — |
| 拖拽 | 2 | 0（0.0%） | — |
| 数据量 | 1 | 0（0.0%） | — |

### 4.7 最该先修的 10 个「映射到位但缺标记」元素

排序用「真缺」＝映射到位、缺该维度的标记，且函数体里连同维度的其它写法（§4.6 的 `DIM_ADJACENT`）也没有；「缺标记条数」是全量。维度缺口列是该元素缺标记的维度构成（按条数）。

| # | 元素 ID | 名称 | 真缺条数 | 缺标记条数 | 覆盖需求 | 维度缺口 | 映射点 |
|---:|---|---|---:|---:|---:|---|---|
| 1 | IN-66 | （无文案） | 14 | 15 | 32 | 关联 4、键盘 3、记忆 3、动画 1 | `internal/ui/approval.go:App.approvalCard` |
| 2 | G-C1-01 | （无文案）卡片容器 | 12 | 13 | 30 | 关联 4、动画 2、记忆 2、交互 1 | `internal/ui/approval.go:App.approvalCard` |
| 3 | TL-08 | （无文案） | 12 | 12 | 15 | 状态 3、交互 3、样式 1、动画 1 | `internal/ui/timeline.go:App.backToLatest` |
| 4 | TL-67 | 已压缩 / 自动压缩 + `(部分)` | 9 | 9 | 14 | 状态 2、交互 1、样式 1、动画 1 | `internal/ui/timeline_compaction.go:compactionWord` |
| 5 | G-C1-20 | 「始终允许此命令」 / 「允许此会话期间访问」  | 8 | 8 | 16 | 状态 2、关联 1、无障碍 1、键盘 1 | `internal/ui/approval.go:approvalLongTermCopy` |
| 6 | IN-35 | （排队消息面板） | 8 | 8 | 14 | 状态 5、关联 1、异常 1、记忆 1 | `internal/ui/composer_panels.go:App.queuePanel` |
| 7 | TL-55 | （工具调用详情） | 8 | 8 | 21 | 状态 4、动画 1、记忆 1、高DPI 1 | `internal/ui/timeline.go:App.toolDetail` |
| 8 | G-A5-02 | （当前命中片段） | 7 | 7 | 10 | 交互 2、关联 1、无障碍 1、异常 1 | `internal/ui/search.go:App.searchReveal` |
| 9 | G-D1-16 | 「自定义选项」 + 「每行输入一个自定义值」 | 7 | 7 | 9 | 内容 2、样式 2、交互 1、无障碍 1 | `internal/ui/question.go:questionCardState.answerAt` |
| 10 | S-personalization-30 | 子代理总数限制（滑块） | 7 | 9 | 20 | 交互 4、动画 1、异常 1、记忆 1 | `internal/ui/settings_personalization_screen.go:psSubagentCountSection` |

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

本节只分 `requirements-settings.md`（19 节 + 壳层）。分流的对象是**现行状态列**的未实现条目（§4 口径：映射到位无标记 + 无渲染点），共 2242 条；第一轮逐字口径的 6,140 条是同一批需求的下限计数，两者不是两批条目，差别在判定口径（§1 的「未实现」含「期望列抽不出字面量」的叙述型条目）。

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
| 本版可做 | 63 | 2.8% | 异常 30、交互 15、状态 9 | `S-configuration` 22、`S-archived` 15、`S-keybindings` 14 |
| 需协议 | 1230 | 54.9% | 窄窗 331、异常 263、交互 120 | `S-personalization` 256、`S-usage` 219、`S-account` 197 |
| 需素材 | 160 | 7.1% | 异常 42、窄窗 32、记忆 23 | `S-appearance` 116、`S-pets` 44 |
| 需宿主能力 | 397 | 17.7% | 窄窗 107、异常 60、状态 49 | `S-general` 249、`S-about` 58、`S-runtime` 53 |
| 取证口径 | 392 | 17.5% | 窄窗 103、键盘 42、记忆 37 | `S-configuration` 190、`S-appearance` 57、`S-keybindings` 47 |
| **合计** | **2242** | **100.0%** | — | — |

未实现条目的维度构成（全量，供对照）：窄窗 573、异常 425、状态 186、交互 185、键盘 165、记忆 160、动画 142、高DPI 134、无障碍 62、关联 57。

### 5.3 每档代表条目（每档 10 条）

#### 本版可做（63 条）

| 需求 ID | 元素 ID | 维度 | 期望 | 判定 | 分档依据 |
|---|---|---|---|---|---|
| R-APP-085 | S-appearance-10 | 交互 | 展开语言列表，当前项带对勾 | 映射到位无标记 | 本地偏好 / 本地草稿 / 已有事实 |
| R-KEY-017 | S-keybindings-03 | 状态 | 空，占位「搜索快捷键…」 | 映射到位无标记 | 本地偏好 / 本地草稿 / 已有事实 |
| R-CFG-354 | S-configuration-45 | 状态 | 留空也能保存 | 映射到位无标记 | 本地偏好 / 本地草稿 / 已有事实 |
| R-PRM-036 | S-permissions-05 | 异常 | 转错误 + 刷新 | 映射到位无标记 | 本地偏好 / 本地草稿 / 已有事实 |
| R-ARC-013 | S-archived-02 | 异常 | 提示 | 映射到位无标记 | 本地偏好 / 本地草稿 / 已有事实 |
| R-APP-086 | S-appearance-10 | 交互 | 界面立即切语言，本节其余文案同步变化 | 映射到位无标记 | 本地偏好 / 本地草稿 / 已有事实 |
| R-APP-098 | S-appearance-11 | 交互 | 收起面板并立即切换界面语言 | 映射到位无标记 | 本地偏好 / 本地草稿 / 已有事实 |
| R-APP-269 | S-appearance-31 | 异常 | 选中回滚到原档 | 映射到位无标记 | 本地偏好 / 本地草稿 / 已有事实 |
| R-APP-278 | S-appearance-32 | 异常 | 回滚 | 映射到位无标记 | 本地偏好 / 本地草稿 / 已有事实 |
| R-APP-287 | S-appearance-33 | 异常 | 回滚 | 映射到位无标记 | 本地偏好 / 本地草稿 / 已有事实 |

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

#### 需宿主能力（397 条）

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

#### 取证口径（392 条）

| 需求 ID | 元素 ID | 维度 | 期望 | 判定 | 分档依据 |
|---|---|---|---|---|---|
| R-APP-015 | S-appearance-02 | 窄窗 | 单行 | 映射到位无标记 | 横向维度 |
| R-KEY-006 | S-keybindings-01 | 记忆 | 回 y=33 | 映射到位无标记 | 横向维度 |
| R-CFG-006 | S-configuration-01 | 窄窗 | 左栏隐藏，详情整宽并出现返回 | 映射到位无标记 | 横向维度 |
| R-PRM-006 | S-permissions-01 | 记忆 | 回 y=33 | 映射到位无标记 | 横向维度 |
| R-ARC-005 | S-archived-01 | 异常 | 回落中文 | 映射到位无标记 | 无 i18n 层 |
| R-SHL-012 | S-shell-02 | 高DPI | 线条清晰 | 映射到位无标记 | 横向维度 |
| R-APP-026 | S-appearance-03 | 动画 | 选中角标 120ms 淡入，主题色 200ms 过渡 | 映射到位无标记 | 横向维度 |
| R-APP-031 | S-appearance-03 | 键盘 | 可聚焦并切换 | 映射到位无标记 | 横向维度 |
| R-APP-036 | S-appearance-04 | 高DPI | 缩略线条不糊 | 映射到位无标记 | 横向维度 |
| R-APP-039 | S-appearance-04 | 动画 | 悬停不放大 | 映射到位无标记 | 横向维度 |

### 5.4 本轮做掉的条目

登记 126 条：现状为「已实现」123 条，不一致 3 条。登记表在 `scripts/reconcile-triage-done.json`，重跑时逐条核对状态列，做没做不看口头说明。

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
| R-CFG-019 | 搜索框：放大镜颜色不随输入变化，只有清除按钮的可见性跟输入状态走 | TestProviderSettingsReadsRecords | 已实现（映射命中） |
| R-CFG-132 | 编辑态的基础 URL 由代码块变成输入框（表单字段） | TestProviderFormEditPrefilled | 已实现（映射命中） |
| R-CFG-139 | 基础 URL 单行截断，悬停显示全文 | TestProviderSettingsReadsRecords | 已实现（映射命中） |
| R-CFG-145 | 表单校验与保存失败的原因都在详情内的 formError 行 | TestProviderFormValidation | 已实现（映射命中） |
| R-CFG-151 | 连续失败只覆盖同一条 formError，不追加 | TestProviderFormValidation | 已实现（映射命中） |
| R-CFG-160 | 模型全选写入失败后整组开关回滚到改动前的值，并写原因 | TestProviderSelectAllRollsBack | 已实现（映射命中） |
| R-CFG-175 | 模型清单没读到时清空搜索词不会恢复列表：保留搜索词并写加载失败 | TestProviderModelsEmptyState | 已实现（映射命中） |
| R-CFG-198 | 拉取失败把空列表换成错误行 | TestProviderModelsEmptyState | 已实现（映射命中） |
| R-CFG-211 | 探测进行中「刷新」禁用、文案不变 | TestProviderProbeBusyDisablesEntries | 已实现（映射命中） |
| R-CFG-220 | 「查看套餐」始终可点，点了写说明（本屏不拼套餐页地址） | TestProviderProbeBusyDisablesEntries | 已实现（映射命中） |
| R-CFG-346 | 认证方式只显示当前的一个标签（API Key） | TestProviderDetailMasksCredential | 已实现（逐字） |
| R-CFG-362 | 已存密钥只显示掩码，不回显明文 | TestProviderDetailMasksCredential | 已实现（映射命中） |
| R-CFG-363 | 点「修改 API 密钥」进编辑态，保存后替换 | TestProviderDetailMasksCredential | 已实现（映射命中） |
| R-CFG-365 | 没有凭据时行下写「保存前需要填写 API 密钥」 | TestProviderDetailMasksCredential | 已实现（映射命中） |
| R-CFG-366 | 密钥字段是密码型输入，读屏不回读内容 | TestProviderFormSecretToggle | 已实现（映射命中） |
| R-CFG-367 | 粘贴进来的密钥去首尾空白后保存 | TestProviderFormSecretToggle | 已实现（映射命中） |
| R-CFG-394 | 「测试连接」发起一次 provider.probe，结果写在行内 | TestProviderDetailTestConnection | 已实现（映射命中） |
| R-CFG-395 | 探测进行中「测试连接」禁用、「保存」仍可用 | TestProviderProbeBusyDisablesEntries | 已实现（映射命中） |
| R-CFG-397 | 测试失败把服务端原因写在行内，已填字段不动 | TestProviderDetailTestConnection | 已实现（映射命中） |
| R-CFG-398 | 「测试连接」「保存」都有读屏名 | TestProviderDetailTestConnection | 已实现（映射命中） |
| R-CFG-425 | 没有模型数据时写「还没有模型数据，点击「刷新」获取」 | TestProviderModelsEmptyState | 已实现（逐字） |
| R-CFG-428 | 这句点名同行的「刷新」按钮，点了走 provider.probe | TestProviderModelsEmptyState | 已实现（映射命中） |
| R-CFG-429 | 空态句可选中复制 | TestProviderModelsEmptyState | 已实现（映射命中） |
| R-CFG-430 | 空态句有读屏名，只播一次 | TestProviderModelsEmptyState | 已实现（映射命中） |
| R-CFG-431 | 拉取失败换成错误行 | TestProviderModelsEmptyState | 已实现（映射命中） |
| R-CFG-437 | 协议一行只读显示读数，不给改 | TestProviderFormEditPrefilled | 已实现（映射命中） |
| R-CFG-439 | 协议名认不出来时显示服务端给的原始标识 | TestProviderFormEditPrefilled | 已实现（映射命中） |
| R-CFG-442 | 名称回显读数里的当前值 | TestProviderFormEditPrefilled | 已实现（映射命中） |
| R-CFG-443 | 清空名称保存时在框下写「请填写名称」 | TestProviderFormValidation | 已实现（映射命中） |
| R-CFG-445 | 同名供应商（不区分大小写）在框下写「名称已被其他供应商使用」 | TestProviderFormValidation | 已实现（映射命中） |
| R-CFG-446 | 名称是带读屏名的文本框 | TestProviderFormEditPrefilled | 已实现（映射命中） |
| R-CFG-450 | 基础 URL 回显读数（含 /v1 路径） | TestProviderFormEditPrefilled | 已实现（映射命中） |
| R-CFG-451 | 非法 URL 在框下写需 http/https | TestProviderFormValidation | 已实现（映射命中） |
| R-CFG-453 | 保存时基础 URL 末尾多余的斜杠归一化 | TestProviderFormNormalizesBaseURL | 已实现（映射命中） |
| R-CFG-454 | 基础 URL 是带读屏名的文本框 | TestProviderFormEditPrefilled | 已实现（映射命中） |
| R-CFG-458 | 已存密钥显示掩码，输入即为替换 | TestProviderDetailMasksCredential | 已实现（映射命中） |
| R-CFG-459 | 密钥默认密码型，按钮切换显示 | TestProviderFormSecretToggle | 已实现（映射命中） |
| R-CFG-461 | 密钥为空且没有已存凭据时保存被拦下并提示 | TestProviderFormValidation | 已实现（映射命中） |
| R-CFG-462 | 密钥字段读屏不回读内容 | TestProviderFormSecretToggle | 已实现（映射命中） |
| R-CFG-464 | 密钥去首尾空白后保存 | TestProviderFormSecretToggle | 已实现（映射命中） |
| R-CFG-474 | 取消退出编辑，不发请求 | TestProviderFormValidation | 已实现（映射命中） |
| R-CFG-475 | 保存先校验再走 UpdateProvider/CreateProvider，成功后回详情 | TestProviderFormCreateWrites | 已实现（映射命中） |
| R-CFG-476 | 保存中保存键禁用 | TestProviderFormCreateWrites | 已实现（映射命中） |
| R-CFG-478 | 保存失败回 formError 并留在编辑态 | TestProviderFormValidation | 已实现（映射命中） |
| R-CFG-482 | 详情里的「修改 API 密钥」进可编辑密钥态 | TestProviderDetailMasksCredential | 已实现（映射命中） |
| R-CFG-485 | 保存失败写原因、留在编辑态，已存密钥不动 | TestProviderFormValidation | 已实现（映射命中） |
| R-CFG-486 | 读屏读「API 密钥已设置」，不读掩码字符 | TestProviderDetailMasksCredential | 已实现（逐字） |
| R-CFG-489 | 「断开此提供商」弹出同名确认框 | TestProviderDisconnectDialogBlocks | 已实现（逐字） |
| R-CFG-492 | 取消关闭确认框，不改状态 | TestProviderDisconnectDialogBlocks | 已实现（映射命中） |
| R-CFG-495 | 确认框有标题与说明的读屏名 | TestProviderDisconnectDialogBlocks | 已实现（映射命中） |
| R-CFG-497 | 「删除此提供商」弹出同名确认框 | TestProviderDeleteDialogBlocks | 已实现（逐字） |
| R-CFG-500 | 取消关闭删除确认框 | TestProviderDeleteDialogBlocks | 已实现（映射命中） |
| R-CFG-503 | 删除确认框有标题与说明的读屏名 | TestProviderDeleteDialogBlocks | 已实现（映射命中） |
| R-CFG-555 | 「添加模型」草稿的全选 / 全不选 | TestProviderModelDraftValidatesLocally | 已实现（映射命中） |
| R-CFG-560 | 计数以列表为准：可见数按搜索词算 | TestProviderModelDraftValidatesLocally | 已实现（映射命中） |
| R-CFG-564 | 新建表单没有供应商时「刷新」禁用 | TestProviderFormOpensFromAdd | 已实现（映射命中） |
| R-CFG-568 | 协议不支持拉清单时「刷新」禁用并写原因 | TestProviderFormOpensFromAdd | 已实现（映射命中） |
| R-CFG-570 | 草稿模型行按搜索词实时过滤 | TestProviderModelDraftValidatesLocally | 已实现（映射命中） |
| R-CFG-571 | 清空搜索词恢复全部草稿行 | TestProviderModelDraftValidatesLocally | 已实现（映射命中） |
| R-CFG-593 | 没有草稿模型时写「还没有模型。可以手动添加…」 | TestProviderModelDraftValidatesLocally | 已实现（逐字） |
| R-CFG-643 | 读数未到时画骨架，宽高与真实行一致 | TestProviderDetailLoadingSkeleton | 已实现（映射命中） |
| R-CFG-645 | 骨架播报「正在加载供应商」一次 | TestProviderDetailLoadingSkeleton | 已实现（逐字） |
| R-SHL-023 | 该节读数失败时导航项仍可点，切过去后节内自己写错误 | TestSettingsNavKeepsSectionClickableOnReadFailure | 已实现（映射命中） |
| R-SHL-030 | 分节图标缺资源时只留同宽图标位，名称左缘与行高不变 | TestSettingsNavIconMissingKeepsTextColumn | 已实现（映射命中） |
| R-SHL-046 | 同上（用量节图标） | TestSettingsNavIconMissingKeepsTextColumn | 已实现（映射命中） |
| R-SHL-055 | 读数失败不拦切节：仍停留该节并显示错误 | TestSettingsNavKeepsSectionClickableOnReadFailure | 已实现（映射命中） |
| R-SHL-062 | 同上（个性化节图标） | TestSettingsNavIconMissingKeepsTextColumn | 已实现（映射命中） |
| R-SHL-078 | 同上（外观节图标） | TestSettingsNavIconMissingKeepsTextColumn | 已实现（映射命中） |
| R-SHL-087 | 读数失败仍可切并显示错误 | TestSettingsNavKeepsSectionClickableOnReadFailure | 已实现（映射命中） |
| R-SHL-094 | 同上（通用节图标） | TestSettingsNavIconMissingKeepsTextColumn | 已实现（映射命中） |
| R-SHL-110 | 同上（快捷键节图标） | TestSettingsNavIconMissingKeepsTextColumn | 已实现（映射命中） |
| R-SHL-119 | 供应商清单失败仍可切并显示错误 | TestSettingsNavKeepsSectionClickableOnReadFailure | 已实现（映射命中） |
| R-SHL-126 | 同上（模型供应节图标） | TestSettingsNavIconMissingKeepsTextColumn | 已实现（映射命中） |
| R-SHL-142 | 同上（多模态节图标） | TestSettingsNavIconMissingKeepsTextColumn | 已实现（映射命中） |
| R-SHL-151 | 运行环境未声明仍可切，节内显示错误 | TestSettingsNavKeepsSectionClickableOnReadFailure | 已实现（映射命中） |
| R-SHL-158 | 同上（运行环境节图标） | TestSettingsNavIconMissingKeepsTextColumn | 已实现（映射命中） |
| R-SHL-167 | 服务器清单失败仍可切并显示错误 | TestSettingsNavKeepsSectionClickableOnReadFailure | 已实现（映射命中） |
| R-SHL-174 | 同上（MCP 节图标） | TestSettingsNavIconMissingKeepsTextColumn | 已实现（映射命中） |
| R-SHL-190 | 同上（通知节图标） | TestSettingsNavIconMissingKeepsTextColumn | 已实现（映射命中） |
| R-SHL-199 | Pet 组件不可用仍可切 | TestSettingsNavKeepsSectionClickableOnReadFailure | 已实现（映射命中） |
| R-SHL-206 | 同上（宠物节图标） | TestSettingsNavIconMissingKeepsTextColumn | 已实现（映射命中） |
| R-SHL-222 | 同上（权限节图标） | TestSettingsNavIconMissingKeepsTextColumn | 已实现（映射命中） |
| R-SHL-231 | 仓库不可用仍可切 | TestSettingsNavKeepsSectionClickableOnReadFailure | 已实现（映射命中） |
| R-SHL-238 | 同上（Git 节图标） | TestSettingsNavIconMissingKeepsTextColumn | 已实现（映射命中） |
| R-SHL-254 | 同上（Hooks 节图标） | TestSettingsNavIconMissingKeepsTextColumn | 已实现（映射命中） |
| R-SHL-263 | 插件设置不可用仍可切 | TestSettingsNavKeepsSectionClickableOnReadFailure | 已实现（映射命中） |
| R-SHL-270 | 同上（插件设置节图标） | TestSettingsNavIconMissingKeepsTextColumn | 已实现（映射命中） |
| R-SHL-279 | 能力不可用仍可切 | TestSettingsNavKeepsSectionClickableOnReadFailure | 已实现（映射命中） |
| R-SHL-286 | 同上（外部 Agent 节图标） | TestSettingsNavIconMissingKeepsTextColumn | 已实现（映射命中） |
| R-SHL-302 | 同上（归档会话节图标） | TestSettingsNavIconMissingKeepsTextColumn | 已实现（映射命中） |
| R-SHL-318 | 同上（关于节图标） | TestSettingsNavIconMissingKeepsTextColumn | 已实现（映射命中） |
| R-SHL-321 | 窄窗出现返回按钮、常驻侧栏收起 | TestSettingsNarrowBackToSectionList | 已实现（映射命中） |
| R-SHL-323 | 窄窗返回先回节列表，节列表态再返回回主界面 | TestSettingsNarrowBackToSectionList | 已实现（映射命中） |
| R-SHL-334 | 返回图标缺资源时退回同族箭头，命中区仍是 24×24 | TestSettingsNarrowBackIconMissingKeepsHitArea | 已实现（映射命中） |
| R-APP-028 | 主题落盘失败后角标回滚到原主题并提示 | TestAppearanceThemeSaveFailureRollsBack | 已实现（映射命中） |
| R-APP-038 | 主题预览图标缺资源时用等尺寸底色块占位，文字不跳 | TestAppearanceThemePreviewIconFallsBack | 已实现（映射命中） |
| R-APP-047 | 同上（深色卡） | TestAppearanceThemeSaveFailureRollsBack | 已实现（映射命中） |
| R-APP-055 | 系统报不出外观时按浅色渲染、卡仍选中，并写原因 | TestAppearanceSystemUnknownFallsBackLight | 已实现（映射命中） |
| R-APP-065 | 选中角标的对勾缺资源时用纯色圆点兜底 | TestAppearanceThemePreviewIconFallsBack | 已实现（映射命中） |
| R-APP-130 | 字体下拉点开后有搜索框 + 默认项 + 本机字体列表 | TestAppearanceFontPanelSearchesAndApplies | 已实现（映射命中） |
| R-APP-131 | 选中字体立即生效并落盘 | TestAppearanceFontChoiceIsPersisted | 已实现（映射命中） |
| R-APP-164 | 代码字体与 UI 字体同一套面板 | TestAppearanceFontPanelSearchesAndApplies | 已实现（映射命中） |
| R-APP-165 | 代码字体立即进代码字级 | TestAppearanceFontChoiceIsPersisted | 已实现（映射命中） |
| R-APP-173 | 字体列表按族名子串实时过滤（不区分大小写） | TestAppearanceFontPanelSearchesAndApplies | 已实现（映射命中） |
| R-APP-174 | 清空搜索词恢复全部（面板留着） | TestAppearanceFontSearchEmpty | 已实现（映射命中） |
| R-APP-198 | 点一行字体就应用并收起面板 | TestAppearanceFontPanelSearchesAndApplies | 已实现（映射命中） |
| R-APP-224 | 读不到系统字体时写原因并给「重试」，重试后列表回来 | TestAppearanceFontListRetry | 已实现（映射命中） |
| R-APP-264 | UI 字号分段：点「特小」立即切到最小档 | TestAppearanceUISizeSegmentIsPersisted | 已实现（映射命中） |
| R-APP-265 | 分段控件左右键移动、Home/End 到端点 | TestAppearanceUISizeKeyboardHomeEnd | 已实现（映射命中） |
| R-APP-275 | 点「小」切到第二档 | TestAppearanceUISizeSegmentIsPersisted | 已实现（映射命中） |
| R-APP-284 | 点「中」切到默认档 | TestAppearanceUISizeSegmentIsPersisted | 已实现（映射命中） |
| R-APP-294 | 点「大」切到第四档 | TestAppearanceUISizeSegmentIsPersisted | 已实现（映射命中） |
| R-APP-303 | 点「特大」切到最大档 | TestAppearanceUISizeSegmentIsPersisted | 已实现（映射命中） |
| R-APP-336 | 代码字号滑块拖动实时改代码字级与读数 | TestAppearanceCodeSizeChangesCodeText | 已实现（映射命中） |
| R-APP-337 | 点轨道跳到点击位置（框架滑块的拖拽落点） | TestAppearanceCodeSizeChangesCodeText | 已实现（映射命中） |
| R-APP-338 | 滑块左右键每次 ±1，Home/End 到两端 | TestAppearanceCodeSizeKeyboardHomeEnd | 已实现（映射命中） |
| R-APP-384 | 代码主题按分组列出，选中后代码区与预览同步换配色 | TestAppearanceCodeThemeAppliesToPreview | 已实现（映射命中） |
| R-APP-403 | 点一行代码主题就选定并收起面板 | TestAppearanceCodeThemeAppliesToPreview | 已实现（映射命中） |
| R-APP-419 | 深色主题在浅色界面下也给深色代码底 | TestAppearanceCodeThemeAppliesToPreview | 已实现（映射命中） |

## 6. 附录：主窗口与全局面板的未实现分流（五档）

本节只分 `requirements-main.md`（主窗口五区）与 `requirements-global.md`（全局面板八类）。分流的对象是**现行状态列**的未实现条目（§4 口径：映射到位无标记 + 无渲染点）：主窗口 862 条、全局面板 797 条，合计 1659 条。

领导给的 881 / 554 是 §1 第一轮「未实现」的下限计数（只看字面量、符号、测试），与本节不是同一批：第二轮把「元素映射到位、但函数体里没有该维度标记」也算成未实现，所以全局面板从 554 涨到 797 条；主窗口从 881 条到 862 条，差别很小。

### 6.1 分档口径

| 档 | 判据（按顺序取第一个命中） | 含义 |
|---|---|---|
| 需宿主 | 期望/场景/验收列命中宿主接口词（17 个，另继承 §5 的词表），或维度属多窗口、拖拽、输入法 | 要系统保存面板、窗口拖动、全屏、托盘、下载安装等宿主通道 |
| 需协议 | 元素族属 `G-C2`, `G-C3`, `G-G2`, `G-H2`, `G-H3`, `G-H6`, `G-H7`, `LB-67`, `LB-70`, `LB-74`, `LB-76`, `LB-81`, `LB-82`，或文案命中服务端事实词（未读、改动摘要、定时来源、成员、审批历史、会话复制、快照过期、插件市场、市场安装、下载进度、Cookie、电脑操作、沙箱、子代理模型、分支不一致） | protocol v3 没有对应方法或字段（未读、改动摘要、逐会话分支、成员审批明细、会话 Fork 副本、市场安装、浏览器数据） |
| 需事实 | 文案命中（Agent 配置、多套 Agent 配置） | 协议已有方法（如 `session.agent.get/update`），Go 侧 Source 还没投影出该读数 |
| 取证口径 | 判定为无渲染点，或维度属横向维度（位置、动画、多窗口、多语言、密度、展开后、性能、悬停、拖拽、数据量、暗色、深色、空间、窄窗、记忆、语言、输入法、键盘、高DPI），或场景是「语言包缺失回落」 | 这一档不是待实现功能，是判定要补证据：补元素锚点后重判，或界面实拍取证 |
| 本版可做 | 其余 | 本端视图状态、已有事实或已有写通道能落地 |

分档只说「要什么才能做」，不改状态列；状态列仍由 §4 判。

### 6.2 分档计数

| 文件 | 现行未实现 | 本版可做 | 需协议 | 需事实 | 需宿主 | 取证口径 |
|---|---:|---:|---:|---:|---:|---:|
| `requirements-main.md` | 862 | 246 | 4 | 0 | 18 | 594 |
| `requirements-global.md` | 797 | 230 | 223 | 0 | 13 | 331 |
| **合计** | **1659** | **476** | **227** | **0** | **31** | **925** |

「需事实」本轮为 0：唯一的需事实项（状态条的 Agent 配置，`session.agent.get/update` 已存在但 Source 没有投影）已在本轮接上；SB-08 剩下的窄窗细节按横向维度归取证口径。

- **本版可做** 476 条（28.7%）：维度 状态 113、异常 112、关联 78；元素族 `G-C4` 66、`G-B1` 35、`G-D1` 29、`G-C1` 28、`G-A1` 19。
- **需协议** 227 条（13.7%）：维度 样式 36、无障碍 33、状态 31；元素族 `G-H2` 49、`G-G2` 38、`G-C3` 27、`G-H6` 27、`G-C2` 26。
- **需宿主** 31 条（1.9%）：维度 状态 7、内容 7、多窗口 5；元素族 `G-H4` 5、`G-H6` 5、`TL-18` 3、`G-H7` 3、`TB-01` 2。
- **取证口径** 925 条（55.8%）：维度 记忆 246、窄窗 116、键盘 92；元素族 `G-B2` 55、`G-H5` 44、`G-C4` 41、`G-C1` 38、`G-G1` 38。

### 6.3 每档代表条目（每档 10 条）

#### 本版可做（476 条）

| 需求 ID | 元素 ID | 维度 | 期望 | 判定 | 分档依据 |
|---|---|---|---|---|---|
| R-TB-010 | TB-02 | 交互 | 左栏展开，主区卡片让位；按钮消失，换成左栏内的收起按钮（`LB-10`） | 映射到位无标记 | 本地视图状态 / 已有事实 |
| R-TB-018 | TB-03 | 状态 | 标题栏右端出现打开面板按钮（`PanelRight`），987,15 2 | 映射到位无标记 | 本地视图状态 / 已有事实 |
| R-TB-026 | TB-04 | 状态 | 出现关闭面板按钮（`PanelRightClose`），362,15 2 | 映射到位无标记 | 本地视图状态 / 已有事实 |
| R-TB-061 | TB-07 | 异常 | 输入框内横向滚动，浮层宽度不变 | 映射到位无标记 | 本地视图状态 / 已有事实 |
| R-TB-071 | TB-08 | 异常 | 仍关闭浮层，高亮残留由下一次渲染清理，不卡在搜索态 | 映射到位无标记 | 本地视图状态 / 已有事实 |
| R-TB-074 | TB-09 | 状态 | 按钮保持可点：索引按 `(i-1+len)%len` 环绕，跳到最后一个 | 映射到位无标记 | 本地视图状态 / 已有事实 |
| R-TB-083 | TB-10 | 状态 | 按钮保持可点：索引按 `(i+1)%len` 环绕，回到第一个命中 | 映射到位无标记 | 本地视图状态 / 已有事实 |
| R-TB-098 | TB-12 | 状态 | 显示「{current} / {total} 个结果」，实例「1 / 3 | 映射到位无标记 | 本地视图状态 / 已有事实 |
| R-TB-244 | TB-31 | 异常 | 主按钮禁用或隐藏，组内仍可用 `TB-32` 看空态 | 映射到位无标记 | 本地视图状态 / 已有事实 |
| R-TB-265 | TB-34 | 异常 | 以本次检测为准，提供刷新（重开菜单即重测） | 映射到位无标记 | 本地视图状态 / 已有事实 |

#### 需协议（227 条）

| 需求 ID | 元素 ID | 维度 | 期望 | 判定 | 分档依据 |
|---|---|---|---|---|---|
| R-LB-416 | LB-67 | 动画 | 指示持续旋转；该轮结束后停止并换成未读点/置顶图标 | 映射到位无标记 | 元素族在 protocol v3 无通道 / 文案指向服务端事实 |
| R-LB-798 | LB-70 | 动画 | 圆点出现无位移动画 | 映射到位无标记 | 元素族在 protocol v3 无通道 / 文案指向服务端事实 |
| R-LB-817 | LB-76 | 高DPI | 图标与文本基线对齐、边缘清晰 | 映射到位无标记 | 元素族在 protocol v3 无通道 / 文案指向服务端事实 |
| R-AP-031 | G-C1-01 | 关联 | 答复后卡片消失，审批记录进入该会话的审批历史（由时间线/审批域承担），不 | 映射到位无标记 | 元素族在 protocol v3 无通道 / 文案指向服务端事实 |
| R-AP-228 | G-C2-01 | 关联 | 与工具权限卡互斥，同一时刻只有一张 | 映射到位无标记 | 元素族在 protocol v3 无通道 / 文案指向服务端事实 |
| R-AP-297 | G-C3-01 | 关联 | 三卡互斥，同一时刻一张 | 映射到位无标记 | 元素族在 protocol v3 无通道 / 文案指向服务端事实 |
| R-GS-169 | G-A2-11a | 关联 | 五分支互斥（失败/加载/未读/置顶/默认） | 映射到位无标记 | 元素族在 protocol v3 无通道 / 文案指向服务端事实 |
| R-GF-107 | G-G1-14 | 内容 | 提示逐字「会话快照已过期并重新加载。请检查消息和排除项后再次导出。」，` | 无渲染点 | 元素族在 protocol v3 无通道 / 文案指向服务端事实 |
| R-GF-142 | G-G2-01 | 内容 | 标题逐字「会话正在使用中」 | 无渲染点 | 元素族在 protocol v3 无通道 / 文案指向服务端事实 |
| R-GI-102 | G-H2-01 | 内容 | 标题逐字「安装 {name}」，name 为条目名 | 无渲染点 | 元素族在 protocol v3 无通道 / 文案指向服务端事实 |

#### 需事实（0 条）

无。

#### 需宿主（31 条）

| 需求 ID | 元素 ID | 维度 | 期望 | 判定 | 分档依据 |
|---|---|---|---|---|---|
| R-TB-001 | TB-01 | 状态 | 标题栏左侧留 80×52 空白占位，无按钮无文字 | 无渲染点 | 宿主接口词 / 多窗口·拖拽·输入法 |
| R-TB-035 | TB-05 | 状态 | 标题栏左侧留 80×52 空白占位，右侧依次为导航按钮组与标题 | 无渲染点 | 宿主接口词 / 多窗口·拖拽·输入法 |
| R-TB-058 | TB-07 | 输入法 | 组合期间不逐键跳转结果（上屏后再定位） | 映射到位无标记 | 宿主接口词 / 多窗口·拖拽·输入法 |
| R-TB-246 | TB-31 | 记忆 | 主按钮图标随之更新 | 映射到位无标记 | 宿主接口词 / 多窗口·拖拽·输入法 |
| R-LB-034 | LB-05 | 动画 | 数字原地替换，不做滚动、不做淡入；按钮宽度随位数变化不抖动左缘 | 映射到位无标记 | 宿主接口词 / 多窗口·拖拽·输入法 |
| R-LB-676 | LB-13 | 异常 | 文案不变（仍显示产品定义的键位），不临时改成其它键 | 映射到位无标记 | 宿主接口词 / 多窗口·拖拽·输入法 |
| R-TL-153 | TL-18 | 状态 | 渲染为文本块，居中或左侧带提示样式 | 无渲染点 | 宿主接口词 / 多窗口·拖拽·输入法 |
| R-IN-153 | IN-12 | 多窗口 | 一个窗口发送后，另一个窗口的按钮不保持“发送中”态，草稿不串 | 映射到位无标记 | 宿主接口词 / 多窗口·拖拽·输入法 |
| R-IN-182 | IN-13 | 多窗口 | 一个窗口停止后，另一个窗口同步退回发送键 | 映射到位无标记 | 宿主接口词 / 多窗口·拖拽·输入法 |
| R-IN-621 | IN-59 | 输入法 | 组合期间列表不逐键抖动 | 映射到位无标记 | 宿主接口词 / 多窗口·拖拽·输入法 |

#### 取证口径（925 条）

| 需求 ID | 元素 ID | 维度 | 期望 | 判定 | 分档依据 |
|---|---|---|---|---|---|
| R-TB-003 | TB-01 | 样式 | 占位不画分割线、不画阴影，两主题下都是纯底色 | 无渲染点 | 无渲染点（先补锚点再判） |
| R-TB-023 | TB-03 | 键盘 | 打开右栏，焦点不跳进右栏 | 映射到位无标记 | 横向维度 |
| R-TB-031 | TB-04 | 键盘 | 关闭右栏 | 映射到位无标记 | 横向维度 |
| R-TB-037 | TB-05 | 样式 | 占位为纯底色，不画线不画阴影 | 无渲染点 | 无渲染点（先补锚点再判） |
| R-TB-046 | TB-06 | 动画 | 原地淡入淡出，无位移动画；标题栏高度不变 | 映射到位无标记 | 横向维度 |
| R-TB-063 | TB-07 | 记忆 | 查询词清空 | 映射到位无标记 | 横向维度 |
| R-TB-080 | TB-09 | 记忆 | 序号从 0 开始，不沿用上次位置 | 映射到位无标记 | 横向维度 |
| R-TB-089 | TB-10 | 记忆 | 搜索结果与序号不跨会话保留 | 映射到位无标记 | 横向维度 |
| R-TB-097 | TB-11 | 记忆 | 该文案消失，浮层只留搜索框与关闭按钮 | 映射到位无标记 | 横向维度 |
| R-TB-105 | TB-12 | 记忆 | 计数从新查询重新计算 | 映射到位无标记 | 横向维度 |

### 6.4 本轮做掉的条目（主窗口 / 全局面板）

登记 40 条：现状为「已实现」40 条，不一致 0 条。做没做按状态列与测试名核。

| 需求 ID | 实现点 | 证明它的测试 | 现状 |
|---|---|---|---|
| R-TB-115 | 左栏收起时标题栏导航组：上一个工作区会话，边界置灰并写原因 | TestSidebarCollapsedNavGroup | 已实现（逐字） |
| R-TB-116 | 点上一个导航：把同工作区的前一条会话交给打开通道 | TestSidebarCollapsedNavGroup | 已实现（映射命中） |
| R-TB-117 | 首条会话时上一个禁用，不回卷 | TestSidebarCollapsedNavGroup | 已实现（映射命中） |
| R-TB-124 | 左栏收起时标题栏导航组：下一个工作区会话 | TestSidebarCollapsedNavGroup | 已实现（逐字） |
| R-TB-125 | 点下一个导航：打开同工作区的后一条会话 | TestSidebarCollapsedNavGroup | 已实现（映射命中） |
| R-TB-126 | 末条会话时下一个禁用 | TestSidebarCollapsedNavGroup | 已实现（映射命中） |
| R-TB-132 | 左栏收起时标题栏导航组：在当前工作区新建会话 | TestSidebarCollapsedNavGroup | 已实现（映射命中） |
| R-TB-133 | 点新建：CreateSession 带当前工作区 cwd，左栏保持收起 | TestSidebarCollapsedNavGroup | 已实现（映射命中） |
| R-LB-484 | 会话行悬停 500ms 出现 hover 卡（标题 + 工作区） | TestSessionHoverCard | 已实现（映射命中） |
| R-LB-485 | 卡内标题完整显示，最多两行，不截成省略号 | TestSessionHoverCard | 已实现（映射命中） |
| R-LB-487 | 卡面：浮层底色 + 1px 描边 + 阴影 + 圆角，宽小于栏宽 | TestSessionHoverCard | 已实现（映射命中） |
| R-LB-490 | 卡内第二行：工作区图标 + 工作区名 | TestSessionHoverCard | 已实现（映射命中） |
| R-LB-491 | 重命名工作区后卡内名称同步（按当前分组名取，不缓存旧名） | TestSessionHoverCard | 已实现（映射命中） |
| R-SB-093 | 会话提供多套 Agent 配置时，状态条权限按钮右侧出现「Agent 配置」 | TestStatusAgentSelector | 已实现（逐字） |
| R-SB-094 | 点 Agent 配置打开菜单，选中项切换走 session.agent.update，带切换前标识 | TestStatusAgentSelector | 已实现（映射命中） |
| R-SB-097 | 与权限按钮同规格：无边框、次要前景色、21 高 | TestStatusAgentSelector | 已实现（映射命中） |
| R-SB-099 | 读屏名与说明带当前配置名与「可更改 Agent 配置」 | TestStatusAgentSelector | 已实现（映射命中） |
| R-GQ-035 | 收起态状态行未答写「待选择」 | TestQuestionCollapsedStatusLine | 已实现（逐字） |
| R-GQ-036 | 收起态未答且选项多：逐字「待选择 · {count} 个选项」 | TestQuestionCollapsedStatusLine | 已实现（映射命中） |
| R-GQ-037 | 收起态已答单值：逐字「已选：{answer}」 | TestQuestionCollapsedStatusLine | 已实现（映射命中） |
| R-GQ-038 | 收起态已答多值：逐字「已选 {count} 项」 | TestQuestionCollapsedStatusLine | 已实现（映射命中） |
| R-GQ-039 | 四分支互斥：同一时刻只画一句 | TestQuestionCollapsedStatusLine | 已实现（映射命中） |
| R-GQ-040 | 状态行：12px 次要色、单行截断、完整值在 tooltip | TestQuestionCollapsedStatusLine | 已实现（映射命中） |
| R-GC-019 | 确认框 Esc 等同取消；执行中不关闭 | TestConfirmDeleteSessionDialog / TestConfirmKeyboardFocusCycle | 已实现（映射命中） |
| R-GC-060 | 删除确认框 Esc 等同取消，不删会话 | TestConfirmDeleteSessionDialog | 已实现（映射命中） |
| R-GC-110 | 归档确认框 Esc 等同取消 | TestConfirmArchiveSessionsDialog | 已实现（映射命中） |
| R-GC-161 | 删除工作区确认框 Esc 等同取消 | TestConfirmDeleteWorkspaceDialog | 已实现（映射命中） |
| R-GF-126 | 导出面板 Esc 等同取消；导出进行中不关闭 | TestExportEscapeCloses | 已实现（映射命中） |
| R-AP-372 | 审批输入区可滚动（双向），卡片位置不动 | TestApprovalLongContentScrolls | 已实现（映射命中） |
| R-AP-378 | 审批输入区按请求标识 key：换请求/切会话滚动回顶 | TestApprovalLongContentScrolls（渲染覆盖；滚动偏移由框架状态保存，Tester 不暴露） | 已实现（映射命中） |
| R-AP-433 | 编辑前后两栏各自双向滚动、各自限高，不互相撑高 | TestApprovalPanelWriteAndEdit（渲染覆盖；两栏各自 ScrollBoth，Tester 不暴露滚动偏移） | 已实现（映射命中） |
| R-AP-450 | 补丁块双向滚动，长行横向滚，不撑破卡片 | TestApprovalPanelWriteAndEdit（渲染覆盖；横向滚动由 ScrollBoth 提供） | 已实现（映射命中） |
| R-AP-587 | 通用代码块双向滚动，长行不撑破卡片 | TestApprovalPanelLegacyTextInput（渲染覆盖；横向滚动由 ScrollBoth 提供） | 已实现（映射命中） |
| R-GS-011 | 输入框左端放大镜 20px，常驻；补元素锚点后重判 | TestSessionSearchMagnifierFocusesInput | 已实现（映射命中） |
| R-GS-013 | 放大镜取次要色、20px；补元素锚点后重判 | TestSessionSearchMagnifierFocusesInput | 已实现（映射命中） |
| R-GS-018 | 查询为空时放大镜仍显示；补元素锚点后重判 | TestSessionSearchOpensOnCommandF | 已实现（映射命中） |
| R-GS-057 | 结果行上边框 1px，横跨浮层内宽；补元素锚点后重判 | TestSessionSearchMatchesTimelineAndCounts | 已实现（映射命中） |
| R-GS-058 | 结果行以工具行读出（按钮组 + 计数）；补元素锚点后重判 | TestSessionSearchArrowButtons | 已实现（映射命中） |
| R-GS-079 | 结果计数逐字「{current} / {total} 个结果」；补元素锚点后重判 | TestSessionSearchMatchesTimelineAndCounts | 已实现（映射命中） |
| R-GS-083 | 计数文本 12px 次要色、Role=Status；补元素锚点后重判 | TestSessionSearchMatchesTimelineAndCounts | 已实现（映射命中） |
