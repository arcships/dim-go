# 待裁决 18 条的落地裁定

Status: 现行（2026-11-19）
Scope: 界面入口的可用性、权限档位映射、未接能力的处理与记录
Input: `docs/domains/requirements/gaps.md` 的「需裁决」类，以及界面走查中同形的未接入口（本批共 18 条）
Owner: 界面实现（裁定记录）；组织侧只做索引与追溯

## 0 口径

- 编号用 `DEC-01`…`DEC-18`，与面板元素的 `D-01`…`D-110` 区分。
- **能做的不留空**：用已有读数、已有 `Intents` 或本地可丢状态能完成的，本批直接做。
- **不能做的不留假入口**：需要协议方法或宿主能力、而本版没有通道的，入口保留但禁用，原因写在 tooltip 或入口旁。禁用以外的处理只有两种：隐藏（无入口价值）或只读展示（有事实可看）。
- **界面用词不带内部术语**（UI-3）：技术口径（缺哪个方法、哪个领域边界）写在本表；界面只说「服务端没有提供」「当前版本不支持」「请手动输入」。
- 本表只裁产品语义与入口行为，不改协议、不改宿主框架，也不把样例结果当事实。

## 1 十八条裁定

| 编号 | 条目 | 裁定 | 理由 | 影响面 |
|---|---|---|---|---|
| DEC-01 | 权限三档映射 | 默认权限 → `workspace-write`；智能权限 → `workspace-write`；完全访问权限 → `full-access`；`read-only` 只在服务端回执明确给出时显示为「只读」，不出现在可选菜单里 | 协议只有三档 preset；参考产品的「默认 / 智能」只差审批策略，而协议档位不表达审批策略；`read-only` 不是用户可选档，硬塞会伪造档位 | `internal/ui/composer.go` 的 `permissionLevels` / `permissionLabel`；状态条与输入区两处入口；`TestPermissionTierEntriesSendIntent` 与只读显示测试 |
| DEC-02 | 权限「智能」档 | 并入 `workspace-write`，不新造档位；将来协议提供审批策略字段再单开 | 没有协议字段可写；本地新造一档会在回执后回弹，用户看到的值与事实不一致 | `permissionChoice` 记住点过的那一档；本表与 D5 的映射说明 |
| DEC-03 | git 写：暂存 / 取消暂存 / 还原 / 提交 / 提交并推送 / 推送 / 切分支 / 查看差异 | **不做**。连着真实读数时入口一律禁用，菜单项副行与 tooltip 写原因；`-demo` 保留样例交互 | protocol v3 没有 git 写方法，D7 §1 也明确不做 git 写；留一个点了会“像生效”的入口会让用户误判仓库状态 | `internal/ui/rightpanel_files.go`（`rpGitWriteNote`、文件右键菜单）；`internal/ui/rightpanel_changes.go`（提交菜单、分组动作、行内动作、改动右键菜单）；`TestRightPanelGitWritesDisabled` |
| DEC-04 | MCP 写：保存 / 删除 / 移动 / 查看连接 / 断开连接 / 导入预览 | **不做**。保存与导入入口禁用；查看连接、断开连接图标禁用；原因写进 tooltip 与编辑器/导入面板说明 | 没有 MCP 写入、inspect、disconnect、import 的 `Intents`，协议也未注册这些能力；界面不能自己造写入事实 | `internal/ui/settings_mcp_screen.go`（`mcpIconAction` 带原因、保存按钮、导入面板、编辑器说明） |
| DEC-05 | 消息分叉与消息级删除 / 撤回 | **不做**。三处入口置灰并在读屏名、tooltip、菜单项里写原因 | `Intents` 只有按轮次切分支的 `CreateBranch`，没有消息级 fork；`RowSnapshot` 不带这一行的轮次，不能拿别的轮次顶替；删除与撤回同样没有消息级写方法 | `internal/ui/timeline.go` 的悬停动作、用户/助手右键菜单；`TestUserActionsDoWhatTheySay`、`TestLiveUserMessageActionsAndMenu` |
| DEC-06 | artifact / 产物 | **不做打开**。时间线产物块只在投影给出 `RowArtifact` 且有子行时出现；产物卡上的「打开」按钮禁用并写原因 | 协议没有 artifact 概念；没有事实就不画块，有事实也不留一个打不开却像能点的入口 | `internal/ui/timeline.go`（`artifactsBlock`、`artifactCard`）；投影侧不产生该行 |
| DEC-07 | 附件 / 引用 chip | **等事实**。`RowSnapshot` 未带附件、引用、标注、定时来源字段；真实消息先渲染一行占位「附件与引用还没有进快照（等事实）」，不画假 chip；字段进快照后走现有渲染分支 | 输入区的附件提交已经可用，但消息级事实还没进投影；占位让用户知道边界，也不伪造已发送的附件 | `internal/ui/sample.go` 的 `round.factsPending`；`internal/ui/timeline.go` 的 `userAttachmentArea` / `userRound`；新增占位测试 |
| DEC-08 | 终端多标签 | **连线时维持单终端**：会话事实只给一个终端实例，标签条只显示一个页签，「新建终端」复用同一页签；样例模式可出现多个标签，每个标签一层画布（T-09 的层叠与切换按此实现），走查多层的切换 | 宿主框架没有多实例终端标签；连线时做伪多标签会与真实终端实例分叉。分层画布由视图承担，不宣称有第二个真实终端 | `internal/ui/terminal.go` 的 `termToolbar` / `termTab` / `termCanvas` / `switchTerminal` / `newTerminal` |
| DEC-09 | 预览另存 / 外部打开 | **不做**另存为、用外部工具打开、在文件管理器中显示；入口禁用并写原因。预览与复制路径保留可用 | 另存为、文件管理器与外部工具要宿主能力；协议也没有对应方法。预览与复制路径是本地安全动作，按「能做就做」保留 | `internal/ui/rightpanel_files.go` 文件右键菜单；`internal/ui/rightpanel_changes.go` 改动右键菜单；菜单测试 |
| DEC-10 | 系统目录选择器 | **用文本输入替代**。Worktree 路径输入占位「手动输入路径」，旁注「手动输入路径（系统目录选择器未接入）」；「选择目录」按钮禁用并写原因 | 宿主没有目录选择器；文本输入可逆、可校验，不会卡住流程 | `internal/ui/settings_git_screen.go` 的 `gitWorktreeGroup` / `gitIconButton` |
| DEC-11 | 供应商新增 / 编辑表单 | **做**。设置 › 模型供应补新增与编辑表单，保存走 `CreateProvider` / `UpdateProvider`；不设禁用 | 协议已有 provider 管理方法，`Intents` 也已实现，属于「能做」的一批；不因为表单长得慢就保留禁用 | `internal/ui/settings_configuration_screen.go`（设置线实现）；`internal/uiwire` 的 provider Intents |
| DEC-12 | 插件设置保存 / 重载 | **不做**。保存、重载、根目录重载、错误重试入口禁用并写原因；不再显示本屏校验后的「已保存」假回执 | 插件设置写入与重载没有 `Intents`；屏内校验只能说明输入合法，不能说明已写回服务端 | `internal/ui/settings_plugin_settings_screen.go`（保存、重载、错误行、页内说明） |
| DEC-13 | 运行环境 · 条目区 / 下载安装 | **不做**。刷新、重试、下载安装入口禁用并写原因；条目区保持错误分支（`runtime.list` 未注册） | protocol v3 没有 `runtime.*` 读取方法；下载安装属于宿主能力，客户端不保留第二份运行时 | `internal/ui/settings_runtime_screen.go` |
| DEC-14 | 企业邮箱登录提交 | **不做**。邮箱对话框保留，提交按钮禁用并写原因；浏览器登录与设备码登录保留（已有 `Intents`） | `account.login.begin` 支持浏览器与设备码，邮箱 + 验证码这一段没有协议方法 | `internal/ui/settings_account_screen.go` 的 `accountEmailDialog` |
| DEC-15 | 思考强度写入 | **不做写入**。滑杆禁用，只显示当前档位；原因写在浮层内 | 协议没有 thinking level 写入口；滑杆只改本地值会与当前会话事实分叉 | `internal/ui/composer.go` 的 `effortSelector`；`variant-effort` 截图 |
| DEC-16 | 定时任务创建与唤醒页 | **不做创建**。左栏「定时任务」入口禁用并写原因；唤醒页维持空态，并写明唤醒事实尚未提供 | 定时任务 create 未冻结；唤醒事实归服务端，会话快照不含它。只读清单已有，但不足以支撑新建与唤醒页 | `internal/ui/sidebar.go` 的 `sidebarEntries` / `sidebarEntry`；`internal/ui/rightpanel_tasks.go` |
| DEC-17 | 权限设置保存 / 刷新 / 离开拦截 | **不做写入**。保存隔离设置、保存权限、刷新入口禁用并写原因；离开拦截只在本地草稿成立时保留，不冒充服务端保存 | 进程隔离设置没有写方法；规则写入在设置壳里拿不到 `Intents`；刷新也没有读取方法 | `internal/ui/settings_permissions_screen.go` |
| DEC-18 | 会话置顶 | **本地登记**。保持进程内 `pinned` 映射，不做服务端事实，也不在本批引入跨重启持久化 | 没有置顶事实；本地偏好快照当前只有主题、动效、时区、侧栏宽度四项，扩项要动落盘契约 | `internal/ui/sidebar_state.go` 的 `pinned` / `togglePin`；本文登记「重启后丢失」 |

## 2 与既有文档的关系

| 文档 | 关系 |
|---|---|
| `docs/domains/requirements/gaps.md` | 本表把其中判定为「需裁决」的条目落成可执行裁定；「不能」类（协议/宿主缺失）沿用同一原则：禁用并写原因，不伪造方法 |
| `docs/domains/D7-workspace.md` | DEC-03 / DEC-09 / DEC-10 的边界依据：D7 只做 git 只读口径，不做 git 写、diff 与目录选择器 |
| `docs/domains/D14-delivery.md` | 「未接动作」的统一处理：保留入口时必须有原因；本表把原因文案收敛到 `internal/ui/unavailable.go` |
| `docs/project-organization.md` §9.9 | UI-3（界面不用内部术语）、UI-6（未接能力不伪装可用）、UI-9（默认值不展示）；本表按「可见但禁用 + 原因」执行，工具口径写在这里 |

## 3 实现约定

1. **原因集中在一处**：`internal/ui/unavailable.go` 的 `note*` 常量是这些入口的唯一文案来源；屏文件不再各写一套句子。
2. **禁用不是隐藏**：入口保留在原地，置灰并给 tooltip；没有入口价值的才隐藏。读屏名与 tooltip 都带原因。
3. **可逆的本地动作照做**：预览、复制路径、本地校验、本地草稿、demo 样例交互不因「服务端没接」一起停掉。
4. **不伪造回执**：本屏校验通过、按钮点击、样例数据都不写成「已保存」「已断开」「已提交」。
5. **事实到了就换实现**：DEC-04 / 07 / 12 / 13 / 14 / 15 / 16 / 17 的禁用分支都在原入口上，投影或协议补字段后直接改走事实与 `Intents`，不新增第二套界面。

## 4 验证

- `go build ./... && go vet ./... && go test ./... -count=1` 全绿。
- 目标断言：
  - `TestPermissionTierEntriesSendIntent`（三档映射、完全访问两步确认）；
  - `TestPermissionReadOnlyOnlyFromReceipt`（`read-only` 显示为「只读」、不出现在菜单）；
  - `TestEffortSliderIsDisplayOnly`（滑杆禁用、浮层写原因、拖动不改值）；
  - `TestRightPanelGitWritesDisabled` / `TestRightPanelSubmitMenuDisabled`（git 写入口禁用并带原因）；
  - `TestRightPanelFileMenu`（外部打开、文件管理器、另存为禁用并带原因）；
  - `TestMCPSettingsWriteEntriesDisabled`（保存、查看连接、断开连接禁用并带原因）；
  - `TestUserActionsDoWhatTheySay` / `TestLiveUserMessageActionsAndMenu`（分叉、删除、撤回置灰带原因）；
  - `TestUserMessageAttachmentPlaceholder`（真实消息的附件占位写明「等事实」，产物打开置灰）；
  - `TestTasksPanelEmptyStates`（唤醒空态带原因）；
  - `TestGitWorktreeManualPath`（目录按钮禁用、输入占位与说明写「手动输入路径」）。

## 5 不在本表的相邻项

- **首次运行欢迎卡**：纯 UI，已按条件（未登录且没有供应商）落在空态两条入口；不属于「缺协议/宿主」的 18 条。
- **样式残余**（输入框底色、其余分节标题字号、正文行高、品牌色、空态标记尺寸）：由样式线处理，不属于入口可用性。
- **供应商表单**（DEC-11）由设置线实现；本表只定范围，不重复记录实现细节。
