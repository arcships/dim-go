# 需求实现状态：机械对账报告

Status: 对账报告（脚本生成，可重跑；`python3 scripts/reconcile.py`）
Scope: 四份需求表的全部条目（`docs/domains/requirements/requirements-*.md`），状态列由本报告写回
Owner: 组织 agent（验收判定侧）；需求文件与元素清单归 R 与需求分析领域

## 0. 判定口径

按顺序取第一个命中的证据：

| 判定 | 证据 | 含义 |
|---|---|---|
| 逐字 | 「期望」里「」/“”字面量在 `internal/**/*.go` 的字符串字面量出现 | 强证据，能指到具体函数 |
| 弱 | 清单符号、函数内元素 ID 注释、或字面量只出现在源码注释里 | 有对应实现点，但不证明逐字行为 |
| 测试 | `*_test.go` 断言该文案或提及该元素 ID | 有测试覆盖点，未证明实现 |
| 需负向取证 | 期望/验收为负向断言（不出现、无响应、出现次数 = 0 等） | 单独一档，不计入未实现 |
| 未实现 | 文案、符号、测试都不命中 | 缺 = 未找到文案/符号/测试 |

证据只写 `文件:函数`，不写行号；函数按顶层 `func` 与列 0 的 `}` 切分；包级常量/变量记声明名，找不到声明名才写 `包级`。

## 1. 命中分布（每份）

| 文件 | 总条数 | 逐字 | 弱 | 测试 | 需负向取证 | 未实现 | 逐字占比 | 未实现占比 |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| `requirements-settings.md` | 8325 | 670 | 109 | 5 | 633 | 6908 | 8.0% | 83.0% |
| `requirements-main.md` | 3520 | 423 | 96 | 195 | 384 | 2422 | 12.0% | 68.8% |
| `requirements-global.md` | 2348 | 358 | 60 | 67 | 197 | 1666 | 15.2% | 71.0% |
| `requirements-panels.md` | 2152 | 341 | 101 | 56 | 170 | 1484 | 15.8% | 69.0% |
| **合计** | **16345** | **1792** | **366** | **323** | **1384** | **12480** | **11.0%** | **76.4%** |

「未实现」里有一半以上是「期望列没有可逐字文案」造成的（无可用字面量 13925 条，其中判未实现 11986 条）；这类条目只能靠元素符号/测试或界面取证，见 §3。

## 2. 随机 40 条复核

抽样种子固定（`20261120`），每份 10 条，优先覆盖「逐字」与「未实现」两档；复核结论由脚本按证据反向核验，人工只需看标记为「有分歧」的条目。

### requirements-settings.md

| 需求 ID | 元素 ID | 维度 | 机器判定 | 证据 | 复核 |
|---|---|---|---|---|---|
| R-MCP-025 | S-mcp-04 | 内容 | 逐字 | `internal/app/extensions.go:extensionReaders.ReadToolServers`「服务器」 | 通过（字面量在 internal/app/extensions.go:extensionReaders.ReadToolServers 内） |
| R-EXT-049 | S-externalAgents-07 | 内容 | 逐字 | `internal/ui/settings_external_agents_screen.go:eaCopyUnavailDesc`「App Server 尚未实现外部代理。此模式下 Desktop 无法检测或配置它们。」 | 通过（字面量在 internal/ui/settings_external_agents_screen.go:eaCopyUnavailDesc 内） |
| R-PRM-209 | S-permissions-27 | 内容 | 逐字 | `internal/ui/settings_permissions_screen.go:permCopySaveRules`「保存权限」 | 通过（字面量在 internal/ui/settings_permissions_screen.go:permCopySaveRules 内） |
| R-MOD-073 | S-modality-10 | 内容 | 逐字 | `internal/settings/modality_realtime.go:Manager.OpenRealtime`「实时」 | 通过（字面量在 internal/settings/modality_realtime.go:Manager.OpenRealtime 内） |
| R-GEN-644 | S-general-81 | 关联 | 逐字 | `internal/ui/app.go:App.actionErrorView`「重试」 | 通过（字面量在 internal/ui/app.go:App.actionErrorView 内） |
| R-HOK-183 | S-hooks-23 | 窄窗 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-PER-430 | S-personalization-45 | 内容 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-HOK-111 | S-hooks-14 | 键盘 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-USG-337 | S-usage-42 | 内容 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-KEY-011 | S-keybindings-02 | 内容 | 未实现 | — | 通过（文案/符号/测试均无命中） |

### requirements-main.md

| 需求 ID | 元素 ID | 维度 | 机器判定 | 证据 | 复核 |
|---|---|---|---|---|---|
| R-LB-903 | LB-103 | 内容 | 逐字 | `internal/account/login.go:Account.waitBrowser`「取消」 | 通过（字面量在 internal/account/login.go:Account.waitBrowser 内） |
| R-TL-668 | TL-68 | 状态 | 逐字 | `internal/ui/timeline_compaction.go:compactionWord`「已取消」 | 通过（字面量在 internal/ui/timeline_compaction.go:compactionWord 内） |
| R-TL-383 | TL-39 | 无障碍 | 逐字 | `internal/ui/timeline.go:App.userActions`「从此处分叉」 | 通过（字面量在 internal/ui/timeline.go:App.userActions 内） |
| R-LB-741 | LB-47 | 记忆 | 逐字 | `internal/ui/sidebar.go:App.workspaceGroup`「暂无会话」 | 通过（字面量在 internal/ui/sidebar.go:App.workspaceGroup 内） |
| R-LB-522 | LB-110 | 样式 | 逐字 | `internal/ui/sidebar.go:App.sessionMenu`「在新窗口中打开」 | 通过（字面量在 internal/ui/sidebar.go:App.sessionMenu 内） |
| R-TL-102 | TL-12 | 样式 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-LB-561 | LB-84 | 内容 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-IN-581 | IN-55 | 记忆 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-LB-363 | LB-54 | 异常 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-IN-619 | IN-59 | 无障碍 | 未实现 | — | 通过（文案/符号/测试均无命中） |

### requirements-global.md

| 需求 ID | 元素 ID | 维度 | 机器判定 | 证据 | 复核 |
|---|---|---|---|---|---|
| R-GR-116 | G-F2-06 | 状态 | 逐字 | `internal/account/login.go:Account.waitBrowser`「取消」 | 通过（字面量在 internal/account/login.go:Account.waitBrowser 内） |
| R-GI-158 | G-H2-08 | 状态 | 逐字 | `internal/account/login.go:Account.waitBrowser`「取消」 | 通过（字面量在 internal/account/login.go:Account.waitBrowser 内） |
| R-GI-007 | G-H1-01 | 内容 | 逐字 | `internal/ui/plugin_install.go:pluginDialogTitle`「从 Git 添加插件」 | 通过（字面量在 internal/ui/plugin_install.go:pluginDialogTitle 内） |
| R-GR-018 | G-F1-03 | 内容 | 逐字 | `internal/ui/rename_dialog.go:renameCopySessionDesc`「会话名称」 | 通过（字面量在 internal/ui/rename_dialog.go:renameCopySessionDesc 内） |
| R-GQ-166 | G-D1-20 | 内容 | 逐字 | `internal/ui/question.go:questionCopyNext`「下一题」 | 通过（字面量在 internal/ui/question.go:questionCopyNext 内） |
| R-GI-218 | G-H4-02 | 样式 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-AP-564 | G-C4-23 | 内容 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-GI-188 | G-H3-02 | 异常 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-AP-392 | G-C4-02 | 高DPI | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-GQ-009 | G-D1-01 | 记忆 | 未实现 | — | 通过（文案/符号/测试均无命中） |

### requirements-panels.md

| 需求 ID | 元素 ID | 维度 | 机器判定 | 证据 | 复核 |
|---|---|---|---|---|---|
| R-RA-130 | A-16 | 状态 | 逐字 | `internal/ui/rightpanel_changes.go:App.rpChangeMenu`「用默认应用打开」 | 通过（字面量在 internal/ui/rightpanel_changes.go:App.rpChangeMenu 内） |
| R-SC-007 | S-01 | 窄窗 | 逐字 | `internal/ui/app.go:App.actionErrorView`「重试」 | 通过（字面量在 internal/ui/app.go:App.actionErrorView 内） |
| R-TK-015 | K-02 | 无障碍 | 逐字 | `internal/ui/rightpanel_tasks.go:App.tasksPanel`「唤醒」 | 通过（字面量在 internal/ui/rightpanel_tasks.go:App.tasksPanel 内） |
| R-RD-885 | D-96 | 状态 | 逐字 | `internal/ui/rightpanel_changes.go:App.rpChangeMenu`「复制路径」 | 通过（字面量在 internal/ui/rightpanel_changes.go:App.rpChangeMenu 内） |
| R-RD-494 | D-53 | 状态 | 逐字 | `internal/ui/rightpanel_changes.go:App.rpChangeGroups`「全部暂存」 | 通过（字面量在 internal/ui/rightpanel_changes.go:App.rpChangeGroups 内） |
| R-SG-053 | G-07 | 状态 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-RD-261 | D-28 | 样式 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-SC-135 | S-17 | 交互 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-RD-025 | D-01 | 无障碍 | 未实现 | — | 通过（文案/符号/测试均无命中） |
| R-RD-794 | D-81 | 记忆 | 未实现 | — | 通过（文案/符号/测试均无命中） |

抽样复核：40 条中 40 条与证据一致，0 条标记有分歧。

## 3. 已知局限

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
