# 一致性检查：措辞、交互路径、四态

Status: 已执行（2026-11-20）
依据: `docs/cleanliness.md` §3
范围: `internal/ui` 的屏上文案与状态分支；不改功能范围、不改协议与适配器事实。

## 1 措辞清单

同一事实只留一种叫法。表里的「统一为」是现在代码与 `docs/domains/U3-interaction.md` 的写法。

| 事实 | 原有叫法 | 统一为 | 落点 | 仍不一致及原因 |
|---|---|---|---|---|
| 进程退出码 | 终端提示行「代码 {n}」；命令面板脚注「进程退出码 {n}」 | 退出码 | `internal/ui/terminal.go` 的 `termExitNotice`；`docs/domains/U3-interaction.md` §1.2、§13.5 | 需求表仍按参考原文写「代码」（`requirements-panels.md` R-TM-087/088/093、`inventory-panels*.md` 的 T-11 / TM-I-04）。这批行归 panels 领域，需其同步期望列 |
| 会话的目录 | 文件/改动页空态标题「没有工作区」，同屏说明与终端页写「工作目录」 | 没有工作目录 | `internal/ui/rightpanel_files.go`；`docs/domains/U3-interaction.md` §13.1、§13.2 | 「工作区」保留给注册的项目根 / 左栏分组 / 空态选择器（「添加工作区」「工作区是干净的」为参考原文），两者不是同一对象 |
| 会话所属的项目分组 | 左栏、空态选择器、标题行叫「工作区」；归档屏的筛选与批量删除叫「项目」 | 按屏沿用参考原文 | — | 两套都是参考原文（LB-49 与 S-archived-08）；同一后台对象在两个屏两个名字，用户可见。统一要改需求证据列，本轮不动 |
| 权限档名称 | 默认权限 / 智能权限 / 完全访问权限 / 只读 | 同上（单一定义 `permissionLevels`、`permissionReadOnlyLabel`） | `internal/ui/composer.go`、`internal/ui/statusbar.go` | 无 |
| 模型名 | 快照名称；样例表照抄参考（含「Qwen: Qwen3.8 27B」这类带提供方前缀的写法） | 原样显示快照给的名称 | `internal/ui/composer.go` 的 `currentModel`、`settings_configuration_screen.go` 的模型表 | 样例表是参考产品 DOM 的逐行实录，前缀不统一属参考原文；接上事实后不走样例 |
| 日期 | 归档行「11 月 20 日」；账户节「2026年12月13日」；状态条「1月2日」 | 数字与年月日之间不插空格 | `internal/ui/settings_archived_screen.go` 的 `arcDateLabel` | 无 |
| 时间 | 会话行相对时间（刚刚 / 16 分钟前 / 1 小时前）；任务与唤醒行钟点（15:04） | 列表相对、详情绝对 | `internal/ui/sidebar.go` 的 `timePhrase`、`internal/ui/app.go` 的 `taskClock` | 会话行的相对时间只在样例里出现：连接模式没有会话活动时间事实，全部留空，不编造 |
| 会话标识 | 目录名缺失时回落「未命名会话」；导出文件头带会话标识 | 屏上不出现标识；导出文件是用户显式要的文件，保留 | `internal/uiwire/adapter.go` 的 `unnamedSession` | 导出的 Markdown / 纯文本行与 JSON 字段含会话标识，且面板预览会显示前 20 行（含该行）。要彻底不出标识得改导出格式（属功能变更），未做 |
| 删除类状态词 | 会话「删除 / 已删除」；工作区菜单「移除工作区」；插件「移除 / 已移除」；Pet「删除」 | 按对象沿用参考原文；同一对象的回执与对话用同一个量词 | `internal/ui/confirm.go` | E4 的描述是参考原文「N 条会话」，删除回执统一成「N 个会话」（与其他删除回执一致）。同一个动作在菜单叫「移除」、在确认框叫「删除」是参考原文，未改 |
| 未打开会话（右栏空态标题） | 「未打开会话」（六个面板）/「还没有打开会话」（子代理） | 未打开会话 | `internal/ui/rightpanel_subagent.go`；`docs/domains/U3-interaction.md` §13.7 | 斜杠命令的置灰原因仍是「还没有打开会话」（参考原文，句法不同） |
| 骨架与重试的读屏名 | 文件页与改动页共用一个「正在加载文件树」「重试读取文件树」 | 各页说自己读的是什么 | `internal/ui/rightpanel_files.go`、`internal/ui/rightpanel_changes.go`；`docs/domains/U3-interaction.md` §13.2 | 无 |
| 省略号 | 有的写「…」（导出中…、保存中…），有的写「...」（搜索模型...、删除中...、正在加载...） | 不统一 | — | 两套都是参考产品原文，逐条写在需求证据列里；统一要同时改参考证据，属参考文案变更，本轮不动 |
| 输入区 / 输入框 | 快捷键屏的动作名「聚焦输入框」，同一屏的全局唤起说明写「聚焦输入区」；命令面板与目标面板的自写文案用「输入区」 | 区域叫「输入区」，文本框叫「输入框」 | `internal/ui/settings_keybindings_screen.go` 的 `globalShortcutHint` | 两个词对应两个对象（区域 vs 文本框），同一屏已统一到文本框的说法；其余屏的「输入区」指区域，不改 |

## 2 同一交互只有一条主路径

| 交互 | 主路径 | 快捷方式 | 一致性结论 |
|---|---|---|---|
| 重命名会话 / 工作区 | 菜单 → 重命名弹窗（四条校验：超长 / 同名 / 控制字符 / 空） | 双击行 / F2 行内改名 | 同一条 `RenameSession`（工作区是本地视图状态）；行内与弹窗共用 `hasRenameControlChars`，同名不发请求。差异：弹窗禁用「确认」，行内空名直接退出编辑（行内没有错误行，保留原语义） |
| 删除会话 | 行菜单 → 确认框 → `DeleteSession`；归档屏的批量删除也过确认框 | 无第二条删除入口（命令面板没有「删除」） | 无绕过确认的路径 |
| 导出 | 导出面板一处（格式 / 范围 / 导出内容 / 预览） | 标题栏 ⋯、左栏行菜单、命令面板三处入口 | 三个入口都进同一面板、同一份状态 |
| 权限档位 | 状态条选择器 | 空态控制行的选择器（参考 NT-14 的第二个挂载点） | 两处共用 `permissionMenuItems` 与 `permissionControlLock`：可点性、勾选态、原因一致；没有会话或没有写通道时都置灰写原因，不发出注定失败的 `SetPermissionTier` |

## 3 四态（空 / 读取中 / 错误 / 无权限）

| 数据面 | 空 | 读取中 | 错误 | 无权限 | 证据 |
|---|---|---|---|---|---|
| 会话目录 | 暂无会话 | 正在连接… | 连接已断开 | 没有可用的工作区 | `internal/ui/consistency_states_test.go`；`Source.SessionNames` 没有独立的读失败字段，断线即读不到，用连接阶段区分 |
| 目录（文件页） | 这个文件夹是空的 / 未打开会话 / 没有工作目录 | 三条骨架「正在加载文件树」 | 错误行 + 重试 | 警告条「没有写入权限，部分操作不可用」+ 重试 | `internal/ui/rightpanel_states_test.go`（本轮补了权限格） |
| 变更 | 读到空 / 未打开会话 / 没有工作目录 | 三条骨架「正在加载改动」 | 错误行 + 重试 | 警告条 + 「重试读取改动」 | 同上 |
| 后台任务 | 暂无运行进程 / 暂无待唤醒事项 | 无（会话快照同步给） | 读不到唤醒记录 / 能力缺失 | 不适用 | 面板表测试 |
| 终端 | 11 个互斥状态（未打开 / 没有目录 / 没有写入口 / 没启动 / 启动中 / 读到空 / 有输出 / 关闭失败 / 自行退出 / 已关闭 / 读不到输出） | 启动中（5） | 读不到输出（10）/ 关闭失败（9） | 不适用（无权限概念） | §1.2 状态表与 `terminal_states_test.go` |
| 侧聊 | 读到空 / 未打开会话 | 发送中 | 失败横幅 | 不适用（通道缺口） | 面板表测试 |
| 子代理 | 未打开会话 / 没有子代理事实 | 无（没有加载圈） | 首屏 / 行内错误 + 重试 | 不适用（通道缺口） | 面板表测试 |
| 设置各屏 | 各屏空态 | 加载骨架 / 「正在探测…」/「正在加载...」 | 错误行 / 原因卡 | 不适用（设置读数没有按对象授权的事实；读不到走错误态） | 各屏文件头已登记未实现分支 |
| 外部 Agent | 缺（探测返回 0 个时整块空白） | 正在探测外部 Agent… | 失败原因 + 重试 | 不适用 | 参考产品没有「探测到 0 个」的态，未自造 |

## 4 文档对齐

- `docs/domains/U3-interaction.md` 已随代码改：§1.2 终端结束提示、§13.1 / §13.2 空态标题与骨架 / 权限条读屏名、§13.5 终端提示、§13.7 子代理空态；§12 变更记录补了一条。
- 未同步的需求行（归对应领域，见 §1 表最后两列）。

## 5 本轮改动

- 生产代码：`terminal.go`、`rightpanel_files.go`、`rightpanel_changes.go`、`rightpanel_subagent.go`、`confirm.go`、`settings_archived_screen.go`、`composer.go`、`statusbar.go`、`unavailable.go`、`sidebar.go`、`sidebar_state.go`、`timeline.go`、`rename_dialog.go`、`settings_keybindings_screen.go`、`header.go`（注释）。
- 测试：`panel_gap_test.go`、`terminal_states_test.go`、`rightpanel_states_test.go`、`rightpanel_wired_test.go`、`residual_families_test.go`、`composer_wired_test.go`、`settings_archived_wired_test.go`，新增 `consistency_states_test.go`。
- 文档：`docs/domains/U3-interaction.md`；本文件。

## 6 验收

- `go build ./...`、`go vet ./...`：通过。
- `go test ./... -count=1`：与本轮无关的失败仍在（并行工作中的 OAuth / MCP / 浏览器导入用例：`TestIntentCasesCoverIntents`、`TestMCPSettingsWriteEntriesDisabled`、`TestProviderOAuth*`、`TestGeneralImportReadyShowsProfilesAndCounts`），不在本轮改动的文件里；本轮触及的面板、终端、输入区、重命名、导出、侧栏用例单跑全绿。
- 缺真实 App Server / 真实屏读器的走查项仍按 blocked 记，不用替身结果代替。
