# 打假审核（第二轮）：P0/P1 复验、新增六块、证据抽样

- 审核对象：`774d244`（`feat(ui): rename dialog and plugin install, update and marketplace`）。
  审核开始时工作区还有 6 个未提交改动（`requirements-global.md`、`app.go`、`plugin_install.go`、
  `plugin_manage.go`、`plugin_manage_test.go`、`rename_dialog.go`），审核期间它们被提交成 `774d244`；
  本报告的行号与结论按 `774d244` 的工作区内容核对。
- 审核方式：静态阅读 + 注入 `Source` 渲染取证（包内临时探针，跑完删除，不入库）+ 全量证据机检
  + 80 条人工回读。命令：`go test ./...`、`go vet ./...`、`go test ./internal/ui -run <探针>`。
- 口径（沿用第一轮，见 `docs/audit-fakes.md` §0 与附录 D.1）：
  - **假事实**：界面上的数据点追不到 `Source` / `PanelFacts` / `Intents`，且发生在 `source != nil`；
  - **空壳**：点了只改本地状态、不发 Intent，也不禁用并写原因；
  - **证据不合格**：所引 `文件:函数/常量` 不存在、指向注释/空行、所引函数里没有那段文案、
    或实现只在 `-demo` 路径而需求声称「已实现」；行号形态按 D.1 本身就不合格。
- 约束：本报告只出结论与复现方式，不改实现、不提交。

---

## 0. 先说结论

1. **第一轮 12 条 P0：修好 9 条，部分修 1 条，未修 2 条。**
   - 未修：**P0-9 侧聊本地回声**（连接模式仍能把一句话"发出去"并画进只读时间线）、
     **P0-10 子代理恒「正在加载...」**（`subagentReady` 至今没有任何写入点）。
   - 部分修：**P0-1 审批卡**——`cwd: ~/dim-agent` 与空命令回落 `ls -la` 都删了，但
     「有工具名、没有入参」时卡片不画输入区、也不写"读不到"，而「拒绝 / 仅这次」仍可点：
     用户可以在**看不到要执行什么**的情况下放行。
2. **第一轮 14 条 P1：13 条真的改成"动作或禁用+原因"，1 条改成了另一种假。**
   P1-12「立即重试」现在禁用了，但它所在的整行（`RowRetry`）在连接模式下**根本不会被生产**
   （`internal/uiwire/rows.go` 不产这个行种），而回归用例是手工塞一行进去的；同一行的
   `Duration` 契约（`source.go` 写"剩余时间"）与唯一的生产口径（`rowDuration` 是"已耗时"）相反。
3. **新增六块里最硬的假事实不是渲染，是"界面替用户写的话"**：
   Fork「在新分支继续」与状态条「创建新分支」在草稿为空时，自造一句 prompt
   （`从第 N 轮继续` / `从当前分支切出`）提交给服务端——协议要求非空 prompt，
   于是这句话会作为一条**用户从没写过的消息**落进新分支。
4. **两个新面板把"装载窗口"说成了"全量"**：导出面板的「整条会话」只导出 `SessionSnapshot.Rows`，
   会话内搜索的「N / M 个结果」只在同一批行里找；而 `SessionSnapshot` 没有 `HasOlder`
   （投影层的 `HasOlder` 表示还有更早轮次没装载），界面无从知道、也不提示自己只看到一部分。
5. **斜杠命令里的 `/agent`、`/goal` 是纯装饰**：两者只写 `a.mode`，全仓除模式菜单自身没有任何
   读取点——切模式不改变任何行为，却回"已执行：/goal"；`/branch` 则把用户送进会说谎的侧聊。
6. **证据比第一轮好很多，但仍有一半站不住**：四份需求 2403 条「已实现」，
   逐字文案就在所引函数里的只有 470 条（19.6%）；878 条「」在仓库任何 `.go` 里都搜不到
   （是描述性中文而不是文案）；180 条用行号当证据（D.1 明令禁止），其中 23 行指向注释/空行。
   80 条抽样：合格 24（30%）、弱 37（46%）、错位 18（22.5%）、无证据 1（1.25%）。
7. **提交信息与内容不符**（影响溯源）：审批十类渲染器（`approval.go` 621 行）、会话内搜索
   （`search.go` 950 行）、斜杠命令（`slash_commands.go` 570 行）都落在 `fa2bedb` 这个
   **标题是 `docs(requirements): evidence rebuilt…`** 的提交里；六个 `feat(ui)` 提交只是在其上加壳。
   按提交信息回溯这六块，会找不到实现。

---

## 1. P0 / P1 复验（逐条）

判定：`已修`＝现在真的是事实或禁用+原因；`部分`＝主症状没了但仍有一条路径给假/给不出；
`未修`＝第一轮描述的现象仍在。

### 1.1 P0（12 条）

| 编号 | 位置（`774d244`） | 本轮判定 | 依据 | 复现 |
|---|---|---|---|---|
| P0-1 审批卡 cwd / 空命令回落 | `internal/ui/approval.go:165-205`（`approvalCard`/`toolApprovalBody`）、`:131-143`（`approvalBusy`） | **部分** | `cwd:` 与 `ls -la` 已删；命令只从 `ApprovalSnapshot.Command` / `Input` 读。但「工具名非空、入参为空」分支既不画输入区也不写原因是空的，决定按钮仍可点 | 注入 `ApprovalSnapshot{ID:"a9",Tool:"exec",Input:""}`：屏上 `noteApprovalCommand=false`、输入区标签=false，`拒绝/仅这次` 均 `disabled=false`（探针 A-1） |
| P0-2 模型槽回落 `seed-2.1-pro` | `internal/ui/app.go:1878-1892`（`currentModel`）、`composer.go:512-520`（`modelOptions`） | 已修 | `liveSettings()` 命中且 Model/ModelID 皆空时返回 `modelUnset`；浮层候选只读 `Settings().Models`，样例清单只在 `source==nil` | `TestModelSlotShowsUnsetWithoutModelFact`；`go test ./internal/ui -run TestModelSlot` |
| P0-3 设置›Git 画样例远程/分支 | `internal/ui/settings_git_screen.go:114-140`（`renderGit` 的 `live` 分支）、`:167-176`（`gitSelectGroup`）、`:203-210`（`gitSelectDisabled`） | 已修 | 连接模式：原因条 + 下拉取值位写「未读取」+ 全部置灰；`gitRemotes/gitBranches` 只在 `!live` 用 | `TestGitScreenKeepsFormWithoutSampleRemotes` |
| P0-4 设置›宠物画样例 Pet | `internal/ui/settings_pets_screen.go:160-175`（`renderPets`）、`:269-286`（`petsChooseSection` 的 `live` 分支） | 已修 | `live` 时列表区是原因卡，`dim-fox`/「已选择」不上屏，动作按钮置灰 | `TestPetsScreenKeepsFormWithoutSamplePets` |
| P0-5 设置›插件设置画样例卡 | `internal/ui/settings_plugin_settings_screen.go:196-216`、`:220-240`（`plgUnavailableCard`） | 已修 | 连接模式走 `pluginManageView` + 不可用卡，样例插件卡与字段不画 | `TestPluginSettingsScreenKeepsFormWithoutSamplePlugins` |
| P0-6 运行环境恒「服务端未提供」 | `internal/ui/settings_runtime_screen.go:60-92`（`rtLiveSection`）、`:105-135`（`rtRefresher/rtReasonCard`）、`:199-231`（`rtFeatureCard` 的 `live` 分支） | 已修 | 读 `Settings().Runtimes/RuntimesKnown/RuntimesReason`；未读到时写适配器给的原因；功能组件的「未安装/配套版本」只在 `!live` 画 | `TestRuntimeScreenReadsRuntimes`、`TestRuntimeScreenWritesReadReasonWhenUnread` |
| P0-7 多模态恒「未声明/没有任务」 | `internal/ui/settings_modality_screen.go:57-110`（`renderModality` → `mdLiveScreen`）、`:112-135`（`mdCapabilityBanner`） | 已修 | 连接模式只画能力横幅 + 任务清单 + 一句"服务端没有分组事实"；`0 / N` 与样例能力名不再上屏 | `TestModalityScreenReadsFacts`、`TestModalityScreenCapabilityBranches` |
| P0-8 外观主题卡是假选中 | `internal/ui/settings_appearance_screen.go:474-547`（`selectTheme`/`appearanceThemeSection`） | 已修 | 卡片选中态读 `a.themePref`，点卡改 `themePref` 并 `persistPreferences()`；主题真的换 | `TestAppearanceThemeCardAppliesAndFollowsPreference` |
| P0-9 侧聊本地回声 | `internal/ui/rightpanel_sidechat.go:106-171`（`sideChatComposer` / `sendSideChat`） | **未修** | 输入框 `Disabled(closed || sending)`，发送按钮 `disabled: draft=="" || closed`——与是否连接无关；`sendSideChat` 只 `append(a.panels.sideChatSent, text)`，没有任何 Intent。屏上照旧出现"已发送"的气泡 | 连接模式：`sideChatDraft="这条消息会发给谁"` → 调 `sendSideChat()` → `sideChatSent=[这条消息会发给谁]`，屏上出现该文本（探针 A-2）。复现：`dimagent-ui -connect -variant right-panel`，在侧聊输入一行回车 |
| P0-10 子代理恒「正在加载...」 | `internal/ui/rightpanel.go:71-76`、`rightpanel_subagent.go:34-76` | **未修** | `subagentReady` 全仓**仍只有定义与读取**（`grep -rn subagentReady internal/` 仅 `rightpanel.go:71` 与 `rightpanel_subagent.go:35`），`subagentLines` 也没有写入点；加载态不是"加载中"，是恒假状态 | 连接模式打开子代理页连跑 5 帧：`subagentReady=false`、屏上恒有「正在加载...」（探针 A-3） |
| P0-11 终端四个本地假动作 | `internal/ui/terminal.go:140`（`termLive`）、`:200-220`（`syncLiveTerminal`）、`:590-616`（`newTerminal`/`startLiveTerminal`）、`:628-660`（`closeTermProcess`/`stopLiveTerminal`）、`:434-461`（`termMenu`） | 已修 | 终端改成本机 pty，经 `Intents.StartTerminal/WriteTerminal/StopTerminal` 起停；「新建 / 关闭进程 / 清屏 / 断线」按有通道/无通道分支，断线入口只在 `!wired` 出现；清屏只移可见基线（`termCleared`） | `go test ./internal/terminal ./internal/ui -run Terminal`；静态：`grep -n "StartTerminal" internal/ui/terminal.go` |
| P0-12 标题栏自编打开回执 | `internal/ui/header.go:388-425`（`openWithGroup`）、`:428-476`（`openWithMenu`） | 已修 | `wired()` 时主按钮置灰 + `noteOpenWith`，「更多打开方式」只写"没有可用的外部应用"，四款写死应用不上屏，Finder / 系统默认两项置灰 | `TestHeaderOpenWithDisabledWithoutChannel` |

### 1.2 P1（14 条）

| 编号 | 位置 | 本轮判定 | 依据 / 复现 |
|---|---|---|---|
| P1-1 标题栏「归档」假回执 | `internal/ui/header.go:196-215` | 已修 | 走 `ArchiveSession`；无写通道置灰写原因 | `TestHeaderMenuUsesRealSessionActions` |
| P1-2 标题栏「置顶」只翻布尔 | `internal/ui/header.go:146-160`、`sidebar_state.go:422-445` | 已修 | 与左栏同一份视图状态，按会话标识记 | `TestHeaderPinPinsSidebarSession`（含重排） |
| P1-3 标题栏「重命名」假禁用 | `internal/ui/header.go:162-176`、`rename_dialog.go:82-100` | 已修 | 有会话即打开弹窗，`RenameSession` 提交 | `TestHeaderRenameIsAvailable` |
| P1-4 左栏置顶按行号 | `internal/ui/sidebar_state.go:422-445`（`threadPinned/pinned/togglePin`） | 已修 | `pinned map[string]bool` 按会话标识；重排后跟随 | 同上用例的重排段 |
| P1-5 状态条「创建新分支」无反应 | `internal/ui/statusbar.go:162-190` | 已修（但同一段写通道引入一条新假：见 §2.3 F-2） | 连接模式发 `CreateBranch`；无通道置灰 | `TestStatusBarCreateBranchSendsIntent` |
| P1-6 空态「添加新工作区」只改名 | `internal/ui/composer.go:772-777` | 已修 | 走 `startDirPick(dirPickWorkspace)`，登记并落盘 | `TestComposerAddWorkspaceOpensDirectoryPicker` |
| P1-7 设置›通用开关/根目录/代理/浏览器数据 | `internal/ui/settings_general_screen.go:60-80`、`:120-160`、`:190-210`、`:250-270` | 已修 | 连接模式整组置灰 + `noteDesktopSettings`；存储报告读 `Settings().Storage` | `TestGeneralScreenDisablesSampleToggles`、`TestGeneralScreenDesktopActionsDisabled`、`TestGeneralStorageReportReadsFacts` |
| P1-8 设置›外观字体/字号/皮肤不落盘 | `internal/ui/settings_appearance_screen.go:602-620`、`:795-820`、`:940-960`；`app.go:1500-1535` | 已修 | 字体/字号/代码配色写 `AppearancePreferences` 并每帧 `SetTheme`/`applyCodeLook`；皮肤与语言按"没有素材/没有语言包"置灰写原因 | `TestAppearancePreferencesRestore` / `TestAppearanceFontChoiceIsPersisted` / `TestAppearanceUISizeSegmentIsPersisted` / `TestAppearanceLanguageAndSkinDisabled` |
| P1-9 设置›个性化只改屏内 | `internal/ui/settings_personalization_screen.go:88-140` | 已修 | 连接模式卡片与开关置灰 + `notePersonalizationWrite`，点击不改值 | `TestPersonalizationControlsDisabledWithoutChannel` |
| P1-10 设置›权限两个开关白拨 | `internal/ui/settings_permissions_screen.go:219-256`（`permIsolationControlsDisabled`/`permSwitchRowDisabled`） | 已修 | 连接模式开关置灰 + 原因，保存按钮同样禁用并写原因 | `TestPermissionsIsolationDisablesSampleSwitches` |
| P1-11 设置›Git 安全开关只改内存 | `internal/ui/settings_git_screen.go:114-160` | 已修 | 同 P0-3，两个开关置灰 | `TestGitScreenKeepsFormWithoutSampleRemotes` |
| P1-12 「立即重试」纯装饰 | `internal/ui/timeline.go:845-857`、`:1514-1515`；`source.go:503-511`；`uiwire/rows.go:168-172` | **改成禁用，但底下的行在连接模式根本不存在** | 入口已禁用 + `noteRetryNow`。问题：全仓只有测试产生 `RowRetry`（`grep -rn RowRetry internal/ \| grep -v _test` 只命中 `timeline.go:1514` 与 `source.go:510`），adapter 的 `rowSnapshot` 不产这个行种；即便产，`Duration` 是 `UpdatedAt-CreatedAt`（已耗时），界面按「%ds 后重试」渲染成"还剩"。回归用例 `TestTimelineRetryNowDisabledWithoutChannel` 是手工塞 `Rows: []RowSnapshot{{Kind: RowRetry, Duration: 5s}}` 才成立 | 静态：同上两条 grep + `rows.go:104-130` |
| P1-13 侧聊停止/结束分支假布尔 | `internal/ui/rightpanel_sidechat.go:106-171`、`:176-200`（`sideChatEndDialog`） | 已修 | 两个入口置灰 + `noteSideChatWrite`，确认框在 `wired()` 时不出现 | `TestSideChatStopAndEndDisabledWithoutChannel` |
| P1-14 无会话读工作区根 | `internal/ui/rightpanel_files.go:88-105`（`rpNoWorkspace`）、`:150-170` | 已修 | 空态分成「未打开会话」「没有工作区」，不再拿工作区根顶替 | `TestRightPanelUnopenedEmptyState` |

### 1.3 第一轮的「假测试」条目

| 编号 | 本轮状态 | 依据 |
|---|---|---|
| T-1 `TestConcurrentUIInteractions` 在 HEAD 稳定失败 | 已修 | `go test ./internal/uiwire -run TestConcurrentUIInteractions -count=2` 通过；`go test ./...` 全绿 |
| T-2 `samplePhrases` 没盖住 P0 样例 | 部分修 | 补进了 `~/dim-agent`、`ls -la`、`seed-2.1-pro`、`dim-fox`、`Model log`、`origin`（`wired_samples_test.go:60-70`）；仍只覆盖"事实齐全"的一份注入源 |
| T-4 把子代理恒加载断言成正确 | **未修** | `internal/ui/panel_elements_test.go:600-630` 仍断言"重试后回到加载态"，而 `subagentReady` 无写入点（P0-10） |
| T-3 设置屏没有连接模式用例 | 已修（覆盖换了文件） | 新增 `internal/ui/p0_facts_test.go`，用 `openSettingsSection` 覆盖 Git / 宠物 / 插件设置 / 运行环境 / 多模态 / 外观 / 通用 / 权限 / 个性化；`settings_wired_test.go` 仍是 MCP / Hooks / Git |
| T-5 样例常量进 UI 断言 | 已修（口径变了） | `app_test.go:1204-1212` 仍断言 `seed-2.1-pro`，但那是 `-demo` 用例；连接模式由 `TestModelSlotShowsUnsetWithoutModelFact` 单独守 |
| T-6 截图只比 -demo | 未修 | `internal/ui/snapshot_test.go` 仍全部基于 -demo，需求侧也仍把截图当证据 |
| （新增）仓库里有调试残留 | 建议清理 | `internal/ui/zz_dbg_test.go`（`TestSlashDebug`，随 `103e16e` 入库）只有 `t.Logf`、没有断言，是斜杠弹层调试用的临时用例 |

---

## 2. 新增六块（会话内搜索、斜杠命令、审批十类渲染器、提问卡与确认框、导出与 Fork、重命名与插件管理）

### 2.1 样例冒充（连接模式仍画样例）

| # | 位置 | 现象 | 判定 | 复现 |
|---|---|---|---|---|
| S-1 | `internal/ui/settings_modality_screen.go:57-68`、`:210-232`（`modalityGroups` 与 `mdGroupCount`） | 分组的固有上限 `4/3/4/3/3` 与「0 / N」只服务 -demo；连接模式不画。需求侧却把「创作与工具 · 0 / 3」标成「已实现」（R-MOD-065），读者会以为连接模式有这一行 | 样例只在 -demo，**但需求把它当已实现**（证据错位，见 §3） | 连接模式打开 `secModality`：无分组页签；-demo 有 |
| S-2 | `internal/ui/settings_runtime_screen.go:199-231` | 「未安装」「配套版本 1.0.135-desktop.1 · 下载 13.0 MB」只在 `!live` 画 | 同上：样例已关进 -demo，但 R-RUN-193 仍标「已实现」 | 同上 |
| S-3 | `internal/ui/settings_general_screen.go:440-455`（`generalCleanupDialog`） | 存储报告在连接模式只画读数或「未读到」；「0 B」只在不连接时画 | 同上：R-GEN-594「显示 0 B」标已实现 | 连接模式打开清理框：无 `0 B` |
| S-4 | `internal/ui/settings_modality_screen.go:437-460`（`mdRealtimeFormCard`） | 实时会话分节整块只在 -demo 画；`mdLiveScreen` 不画它 | R-MOD-512/514 仍以该函数为「已实现」证据 | 连接模式该屏无「打开会话/关闭会话」 |

**六块里的新界面（搜索浮层、斜杠弹层、审批渲染器、提问卡、确认框、导出面板、Fork、重命名弹层、插件管理）没有发现新的样例常量上屏**：这批文件的样例串只有 `demoModels()`（`composer.go:509-521`，`source==nil` 才用）与 `subagentSamples()`。

### 2.2 空壳交互（点了像生效）

| # | 位置 | 现象 | 为什么算空壳 | 复现 |
|---|---|---|---|---|
| N-1 | `internal/ui/slash_commands.go:248-250`（`/agent`）、`:217-231`（`/goal`）；`composer.go:639-669`（模式菜单） | 选 Agent / Goal 只写 `a.mode`；全仓除模式菜单自身**没有任何读取点**（`grep -rn "a\.mode\b" internal/ui \| grep -v _test`：只有菜单的 Label、Checked 与两处赋值） | 切模式不改变发送、权限、目标、任何行为；弹层仍回「已执行：/goal」。用户会把"模式：Goal"当成生效的现状 | 连接模式键入 `/goal` 回车：屏上「已执行：/goal」、模式标签变 Goal，再发消息行为与 Agent 完全一致 |
| N-2 | `internal/ui/slash_commands.go:203-215`（`/branch`）→ `rightpanel_sidechat.go:153-171` | `/branch` 打开侧聊；侧聊发送是本地回声（P0-9） | 命令把用户送进一个"发了但没发"的面板 | `/branch` 后输入一句话回车：气泡出现，服务端无请求 |
| N-3 | `internal/ui/plugin_manage.go:151-153` | 「重新加载」按钮恒禁用（`disabled=true`），点击分支里只写注释"这里不会到达" | 禁用的写法是诚实的（原因在 tooltip 与 `notePluginReload`）；问题只在观感：它与可用的「添加插件」同排、`-demo` 里也一样灰，容易被读成"重扫功能坏了" | 连接模式/-demo 打开插件设置：按钮恒灰，tooltip 写 `notePluginReload` |
| N-4 | `internal/ui/rightpanel_subagent.go:44-64` | 子代理「重试」只清 `subagentError/subagentFailed`，没有任何重读 | 与 P0-10 同因：清了错误仍回到恒加载 | 点「重试」：回到「正在加载...」 |

### 2.3 假事实（界面上的值追不到事实，或界面替事实写值）

| # | 位置 | 现象 | 判定 | 复现 |
|---|---|---|---|---|
| F-1 | `internal/ui/fork.go:91-96`（`forkPrompt`）、`:99-130`（`forkAtTurn`） | 草稿为空时，Fork 用 `"从第 %d 轮继续"` 当分支起始 prompt 提交；协议要求非空（`internal/catalog/branch.go:88-89`），这句话会变成新分支里的**第一条用户消息** | **假事实**：会话里出现用户没写过的文本 | 连接模式、草稿为空、在某轮菜单点「从此处分叉」：`CreateBranch` 收到 `prompt="从第 3 轮继续"`（探针 A-4） |
| F-2 | `internal/ui/statusbar.go:174-179`（`statusBranchDefaultPrompt`） | 同上：草稿为空时用 `"从当前分支切出"` 当起始 prompt | 同上 | `CreateBranch` 收到 `prompt="从当前分支切出"`（探针 A-5） |
| F-3 | `internal/ui/export.go:84`、`:447-462`（`exportTurns`） | 范围选「整条会话」，内容却是 `SessionSnapshot.Rows`——投影层有 `HasOlder`（`internal/projection/view.go:315-316`）表示还有更早轮次未装载，`ui.SessionSnapshot` 没有这个字段，界面既不提示也不警告 | **口径假**：把"已装载的窗口"说成"整条会话" | 真实长会话（>一页）导出，与 `dimcode` 侧历史条数比对会少；静态：`exportTurns` 只读 `snap.Rows` |
| F-4 | `internal/ui/search.go:423-455`（`liveSearchUnits`）、`:920-957`（结果计数） | 同上：只在已装载行里搜索，计数写作「N / M 个结果」，没有"只搜了已加载部分"的说明 | 口径假（同因） | 长会话搜索一个只存在于更早轮次的词：0 个结果 |
| F-5 | `internal/ui/rename_dialog.go:162-171`（工作区分支）、`sidebar_state.go:182`、`app.go:1215-1217` | 工作区重命名只写本地 `groupNames` 并落盘本地偏好；服务端工作区名不变，界面上不写"仅本机" | 轻度假事实：屏幕上显示的名字不是服务端的工作区名 | 重命名工作区 → 关掉重开（仍生效）→ 同一工作区在别的客户端/导出里仍是原名 |
| F-6 | `internal/ui/approval.go:187-205`（同 P0-1 残留） | 有工具名、无入参时，卡片没有任何内容区，也不写"读不到"，决定按钮可点 | 假事实（缺省即允许） | 见 P0-1 复现 |
| F-7 | `internal/ui/approval_panels.go:639-645`（`approvalImageSize`） | 图片尺寸拼成 `"640 x 480"`，而函数注释与需求写作「宽 × 高」 | 文案不一致（小） | `approvalImagePanel` 渲染入参含 width/height 的请求：屏上 `640 x 480` |

### 2.4 六块里**没有**发现问题的部分（逐块结论）

| 块 | 结论 |
|---|---|
| 会话内搜索（`search.go`） | 语料口径见 F-4；命中、当前项、折叠展开、滚动还原都是本帧状态，没有编造值 |
| 斜杠命令（`slash_commands.go`） | 13 条命令：固定置灰 4 条且原因具体（`/pet`、`/reload-plugins`、`/fast`、`/workflow`）；`/compact`、`/branch`、`/goal` 只在无会话/无写通道时置灰并写原因；`/compact`→`CompactContext`、`/reload-skills`→`RefreshSkills` 都是既有 Intent；问题只在 N-1/N-2 |
| 审批十类渲染器（`approval_panels.go`） | 13 个渲染器全部从 `ApprovalSnapshot.Input`（服务端 `ToolInput` 原文）取值，无样例回落；`cwd` 现在是入参里的真实字段（`approval_panels.go:97`）。唯一文案偏差 F-7 |
| 提问卡（`question.go`） | 事实来自 `Source.Question()`；提交 `RespondQuestion`、拒绝 `SkipQuestion`；`Deadline` 零值时不出倒计时（协议 v3 目前没有），没有编造截止时间 |
| 确认框（`confirm.go`） | E1–E4 都走真实 Intent（`Stop`/`DeleteSession`/`ArchiveSession`）；连接模式无写通道时置灰写原因；失败留在弹框给原文 |
| 导出与 Fork | 导出写文件是真写（`os.WriteFile` + 真实字节数回执，`export.go:868-883`）；Fork 走 `CreateBranch`。问题见 F-1/F-2/F-3 |
| 重命名与插件管理 | 会话重命名走 `RenameSession`；插件安装/更新/移除/启停都走 `Intents`，市场安装入口置灰写原因（`plugin_manage.go:60`）。问题见 F-5 |

---

## 3. 证据错误率与抽样

### 3.1 规则（判定前先固定口径）

1. 合格证据：所引 `文件:函数/常量` 存在，且（a）`「」`里是该函数里逐字的界面文案或代码标记，或
   （b）引用的常量被该函数使用；并且该元素在**连接模式**会上屏。
2. 不合格：
   - **弱**：引用可回读，但 `「」`里是中文描述（不是文案也不是代码），或只有行号/字段名；
   - **错位**：所引行是注释/空行/括号，或名字不存在，或所引函数里没有那段文案，或实现只在 `-demo`；
   - **无证据**：状态列没有任何 `文件:函数`。
3. 行号形态（`证据 = x.go:123`）按附录 D.1 本身不合格（D.1 第 3 条：不写行号）。

### 3.2 全量机器统计（四份需求，2403 条「已实现」）

（下表的类别互有重叠——一行可以既用行号、文案又不在函数里——所以不求和。）

| 项 | 数量 | 占比 |
|---|---:|---:|
| 「已实现」行（main 250 / panels 184 / settings 780 / global 1189） | 2403 | 100% |
| 「」逐字文案就在所引函数里 | 470 | 19.6% |
| 「」在所引文件别处（常量或别的函数） | 431 | 17.9% |
| 「」在仓库任何 `.go` 里都搜不到（描述性中文/不存在的文案） | 878 | 36.5% |
| 用行号当证据（D.1 禁止） | 180 | 7.5% |
| ├ 其中指向注释/空行 | 23 | — |
| └ 其中行号越界 | 1 | — |
| 所引函数/常量名在文件里不存在 | 29 处 | 1.2% |
| 完全没有证据 | 153 | 6.4% |
| 所引文件不存在 | 1 | 0.04% |

> 全量 2403 里，「严格合格」上限是 470/2403 = **19.6%**；把"常量在别处、函数对"算弱合格，
> 上限 901/2403 = 37.5%。

### 3.3 抽样 80 条（随机种子 `20261120`，四份文件各 20 条，按行均匀抽）

| 分类 | 定义 | 条数 | 占比 |
|---|---|---:|---:|
| 合格 | 见 3.1-1 | 24 | 30.0% |
| 弱 | 引用可回读，但「」是中文描述 / 只有行号 / 只有字段名 | 37 | 46.3% |
| 错位 | 注释、空行、括号、名字不存在、文案不在该函数、-demo 专属 | 18 | 22.5% |
| 无证据 | 状态列没有 `文件:函数` | 1 | 1.25% |

**合格率 30%（24/80）；错位+无证据 23.75%（19/80）；不合格（弱+错位+无证据）70%。**

判据：第一轮同口径为 1.7% 严格可用；本轮 19.6%（全量）/30%（抽样）——有实质改善，
但四分之三的「已实现」仍然只有"能猜到在哪"的证据，不是"能回读"的证据。

### 3.4 抽样明细（80 条）

`文件:行` 是需求行号。判定后括号里是理由。

| 需求 | ID | 证据（去 `internal/ui/` 前缀） | 判定 |
|---|---|---|---|
| global:61 | R-AP-011 | tokens.go:const | 错位（`const` 不是名字，文件里没有可回读的声明；`strokeHairline` 实际在 tokens.go 常量块） |
| global:228 | R-AP-178 | approval.go:approvalActions「提交中整组 Disabled…」 | 弱（中文描述当文案） |
| global:459 | R-AP-388 | approval_panels.go:approvalPanelLabel「typeMicro 一档…」 | 弱（中文描述） |
| global:567 | R-AP-496 | approval_panels.go:approvalWebPanel「分页值逐字来自入参」 | 弱（中文描述） |
| global:622 | R-AP-551 | approval_panels.go:approvalInputLines「每行都是可读文本」 | 弱（中文描述） |
| global:715 | R-GS-017 | search.go:sessionSearchInputRow「点它等于点输入框」 | 弱（命中在函数里的**注释**，不是代码） |
| global:754 | R-GS-056 | search.go:sessionSearchResultRow「左按钮组、右结果计数」 | 弱（中文描述=文档注释） |
| global:1157 | R-GM-027 | slash_commands.go:slashGroupHeader「12px 次要色分组标题」 | 弱（中文描述） |
| global:1393 | R-GQ-008 | question.go:questionFoot「提交 / 拒绝失败写卡片自己的错误行」 | 弱（中文描述） |
| global:1417 | R-GQ-032 | question.go:questionHead「分母取 len(items)…」 | 弱（中文描述） |
| global:1488 | R-GQ-103 | question.go:questionOptions「只在 len(options)==0…」 | 弱（中文描述） |
| global:1534 | R-GQ-149 | question.go:declineQuestion「SkipQuestion（取消决定）」 | 弱（中文描述） |
| global:1550 | R-GQ-165 | question.go:questionFoot「框架按钮：Enter 等同点击」 | 错位（框架默认行为，没有逐元素落点；D.1 §D.5 已列过这一条） |
| global:1648 | R-GC-056 | confirm.go:confirmDialog「err 为空不画该行」 | 弱（中文描述；行为本身在） |
| global:1731 | R-GC-125 | confirm.go:confirmTitle「confirmCopyWorkspace「删除工作区会话」」 | 弱（嵌套引用，常量 `confirmCopyWorkspace` 实际在 `:50`） |
| global:1807 | R-GR-010 | （无） | 无证据（只写"偏差登记"） |
| global:1896 | R-GR-092 | rename_dialog.go:renameDialogView「标题/描述/字段/校验/取消/确认」 | 弱（中文清单） |
| global:2227 | R-GI-042 | plugin_install.go:pluginInstallView「字段与模式切换」 | 弱（中文描述） |
| global:2656 | R-GI-441 | settings_general_screen.go:generalClearDialog「没有成功行」 | 弱（中文描述） |
| global:2679 | R-GI-464 | settings_general_screen.go:screenButton | 错位（该文件没有 `screenButton`；同名函数在 `settings_appearance_screen.go:229` 且签名不同；「描边次按钮」也不是逐字文案） |
| main:127 | R-TB-072 | search.go:sessionSearchResultRow「上一个搜索结果」 | 合格 |
| main:132 | R-TB-077 | 同上 | 合格 |
| main:135 | R-TB-080 | search.go:reindexSessionSearch「s.current = 0」 | 合格 |
| main:136 | R-TB-081 | sessionSearchResultRow「下一个搜索结果」 | 合格 |
| main:157 | R-TB-102 | sessionSearchResultRow「typeSecondary.style(t, gui.Text(c, count))」 | 合格 |
| main:158 | R-TB-103 | sessionSearchResultRow「"%d / %d 个结果"」 | 合格 |
| main:645 | R-LB-208 | sidebar.go:viewMenu「平铺」 | 合格 |
| main:736 | R-LB-290 | sidebar.go:workspaceMenu「置顶工作区」 | 合格 |
| main:1173 | R-LB-674 | sidebar.go:sidebarEntries「⌘G」 | 合格 |
| main:1291 | R-LB-792 | sidebar_state.go:428 | 错位（该行是注释 `// threadPinned …`，实现从 `:423` 起） |
| main:1318 | R-LB-819 | sidebar.go:workspaceMenu「归档会话」 | 合格 |
| main:1372 | R-LB-873 | sidebar.go:accountMenu「邀请朋友」 | 合格 |
| main:1669 | R-TL-188 | timeline.go:1451 + :1425 | 错位（`:1451` 是 `}`；`:1425` 在 `shortTarget` 里；`liveRow`/`liveTimeline` 在 `:1478`/`:1470`） |
| main:1879 | R-TL-398 | rightpanel.go:69 | 错位（该行是字段声明 `sideChatSent []string`，证明不了"切会话不清空"） |
| main:2279 | R-TL-777 | timeline.go:1181 | 错位（该行是注释） |
| main:2343 | R-IN-044 | composer.go:218 | 错位（该行是注释；placeholder 在 `:213-215`） |
| main:3304 | R-SB-091 | statusbar.go:104 + composer.go:944 | 错位（`:944` 是附件类型按钮，与"每次打开复位"无关） |
| main:3417 | R-SB-204 | statusbar.go:210 | 错位（该行是 `gui.MenuBase(c, &a.contextOpen)`，与"菜单项点击后置 false"不是同一件事） |
| main:3507 | R-NT-033 | composer.go:777 | 错位（该行是 `})`） |
| main:3636 | R-NT-162 | composer.go:choosePermission「完全访问权限」 | 合格 |
| panels:111 | R-RF-043 | rightpanel_files.go:rpSearchResults「文件搜索失败。」 | 合格 |
| panels:188 | R-RF-113 | rightpanel_files.go:rpPermissionBar「重试」 | 合格 |
| panels:290 | R-RF-215 | rightpanel_files.go:rpDirErrorRow「无法加载此文件夹。」 | 合格 |
| panels:385 | R-RF-303 | rightpanel_files.go:210 | 弱（行号指向 `rpEntryChange`，能对上"暂存点来自条目状态"这条，但不是逐字/行为标记） |
| panels:498 | R-RF-409 | rightpanel_files.go:268 | 弱（行号指向函数首行） |
| panels:779 | R-RD-250 | rightpanel_changes.go:123 | 弱（行号指向函数首行） |
| panels:826 | R-RD-292 | rightpanel_changes.go:361 | 弱（同上） |
| panels:888 | R-RD-354 | rightpanel_changes.go:rpChangesFooter + 标记「-%d」 | 合格 |
| panels:978 | R-RD-444 | rightpanel_changes.go:57 | 弱（行号形态；不过该行确是那层 `Scroll`，"没有滚动记忆"这条内容成立） |
| panels:1030 | R-RD-491 | rightpanel_changes.go:370 | 弱（行号） |
| panels:1043 | R-RD-504 | rightpanel_changes.go:361 | 弱（同上，与 R-RD-292 共用同一行） |
| panels:1095 | R-RD-556 | rightpanel_changes.go:321 | 弱（行号） |
| panels:1195 | R-RD-656 | rightpanel_changes.go:412 | 弱（行号） |
| panels:1503 | R-RA-008 | timeline.go:1451 + :1425 | 错位（同 R-TL-188） |
| panels:1857 | R-TM-168 | terminal.go:termMenu「关闭进程」 | 合格 |
| panels:1896 | R-TK-017 | rightpanel_tasks.go:87 | 弱（行号） |
| panels:1939 | R-TK-060 | rightpanel_tasks.go:63 | 弱（行号） |
| panels:2313 | R-SC-117 | rightpanel_sidechat.go:139 | 弱（行号） |
| panels:2369 | R-SG-010 | rightpanel_subagent.go:subagentPanel「加载失败」 | 合格 |
| panels:2415 | R-SG-056 | rightpanel_subagent.go:65 | 错位（"未就绪文案仍在"建立在 `subagentReady` 恒假上，等于拿假状态当证据） |
| settings:459 | R-ACC-326 | settings_account_screen.go:accountSignedOut + 标记 | 弱（标记里的代码片段与函数实文不同形，需人读） |
| settings:549 | R-ACC-395 | settings_account_screen.go:accountPlanHeader「刷新」 | 弱（文案在常量 `accCopyRefresh`，函数确实用了它） |
| settings:3785 | R-APP-122 | settings_appearance_screen.go:appearanceFontSection「用于应用界面和 Markdown 正文」 | 合格 |
| settings:3925 | R-APP-226 | app.go:actionErrorView「重试」 | 合格 |
| settings:4848 | R-GEN-081 | settings_general_screen.go:generalRootDirCard「会话数据根目录」 | 合格 |
| settings:5370 | R-GEN-458 | app.go:actionErrorView「重试」 | 合格 |
| settings:5559 | R-GEN-594 | settings_general_screen.go:generalCleanupDialog「0 B」 | 错位（`0 B` 只在 `!wired` 分支画；连接模式是读数或「未读到」） |
| settings:5859 | R-KEY-089 | settings_keybindings_screen.go:keybindingRows「搜索对话」 | 弱（文案在 `keybindingDefs`，函数按它遍历） |
| settings:6887 | R-CFG-609 | settings_provider_form.go:configFormFooter「保存」 | 合格 |
| settings:7060 | R-MOD-065 | settings_modality_screen.go:mdCopyGroupCreative「创作与工具」 | 错位（常量存在，但分组与「0 / 3」只在 -demo 画） |
| settings:7438 | R-MOD-337 | settings_modality_screen.go:mdTaskListCard「刷新」 | 弱（文案在常量 `mdCopyRefresh`） |
| settings:7678 | R-MOD-512 | settings_modality_screen.go:mdRealtimeFormCard + 标记 | 错位（实时分节在连接模式整块不画） |
| settings:7683 | R-MOD-514 | 同上 | 错位 |
| settings:8016 | R-RUN-193 | settings_runtime_screen.go:rtFeatureCard「配套版本…」 | 错位（只 `!live` 画） |
| settings:8330 | R-MCP-185 | settings_mcp_screen.go:mcpServerCard「配置服务器」 | 弱（常量 `mcpCopyConfigure`，函数用了它） |
| settings:8453 | R-MCP-273 | settings_mcp_screen.go:mcpEditor「名称」 | 弱（常量 `mcpCopyName`） |
| settings:8751 | R-PET-013 | settings_pets_screen.go:petsLabel + 标记 | 弱（标记是描述性的 `.Label(text)`，不是逐字文案） |
| settings:9118 | R-PRM-069 | settings_permissions_screen.go:permSwitchRowDisabled + 标记 | 弱（同上） |
| settings:9253 | R-PRM-168 | settings_permissions_screen.go:permTableHeader「规则」 | 弱（常量 `permCopyRules`） |
| settings:10902 | R-ABT-152 | settings_about_screen.go:abtAnnouncementSection「公告」 | 合格 |

### 3.5 五个典型（可以直接拿去当修法样板）

1. **指向括号/注释/别的段**：R-TL-188、R-RA-008（`timeline.go:1451` = `}`）、R-NT-033（`composer.go:777` = `})`）、R-IN-044 / R-TL-777 / R-LB-792（注释行）。
2. **-demo 专属当已实现**：R-MOD-065（`0 / 3`）、R-RUN-193（配套版本）、R-GEN-594（`0 B`）、R-MOD-512/514（实时分节）。
3. **描述性中文塞进「」**：R-AP-178/388/496/551、R-GQ-008/032/103/149 等 25 条——这类引用能"猜"到，但没法回读，也没法在改动后自动复核。
4. **一个段注释/一行被多条需求共用**：`rightpanel_changes.go:361` 同时是 R-RD-292 与 R-RD-504 的证据；`timeline.go:1451/1425` 同时是 R-TL-188 与 R-RA-008 的证据。
5. **拿假状态当证据**：R-SG-056 引 `subagentReady`（恒假，见 P0-10）。

---

## 4. 必须修（按"影响决策 → 成本"排序）

**P0（会让人对事实做错决定）**

1. **P0-9 侧聊**：连接模式下输入区与「发送」置灰 + 写原因（`noteSideChatWrite`），删掉 `sideChatSent` 的本地回声与渲染分支；
   顺带处理 `/branch`（N-2），要么不提供，要么打开后整块是"没有侧聊通道"的空态。
2. **P0-10 子代理**：`subagentReady`/`subagentLines` 要么接上事实（协议没有就说明"没有这条事实"并给空态），
   要么把「正在加载...」改成"这个版本没有子代理流"，并把 `panel_elements_test.go:600-630` 的"重试后回到加载态"断言改掉。
3. **P0-1 残留**：`tool != "" && !hasInput` 时把三个决定按钮一起禁用，并写清"读不到要执行的内容"（复用 `noteApprovalCommand/noteApprovalBlocked`）。
4. **F-1/F-2 自造 prompt**：草稿为空时不要编一句起始提问——按"没有起始提问"禁用「在新分支继续 / 创建新分支」，或要求用户先写一句。

**P1（说法与事实不符）**

5. **F-3/F-4 全量口径**：把 `HasOlder` 暴露到 `ui.SessionSnapshot`，导出面板与搜索浮层在"还有更早的轮次"时
   写清"只覆盖已加载的 N 轮 / 加载全部后再导出"，或提供"加载全部"入口。
6. **N-1 模式开关**：`a.mode` 要么接上真实行为（提交时带模式），要么把菜单与 `/agent`、`/goal` 置灰写"这一版没有模式切换"。
7. **P1-12 残留**：既然 adapter 不产 `RowRetry`，就删掉这条 UI 分支与回归用例，或在投影层真的产出该行种；
   同时改掉 `source.go:507` 的"Duration 是剩余时间"（与 `rows.go:168` 的"已耗时"冲突）。
8. **F-5 工作区重命名**：界面上注明"仅本机显示名"，或干脆置灰写"服务端没有工作区改名"。
9. **F-7 图片尺寸**：`x` 改成 `×`，与注释/需求一致。

**P2（证据线）**

10. 把 `「」`里的中文描述全部换成逐字文案或代码标记（抽样里 25 条属这一类）；
11. 取消全部行号形态（180 条），其中 23 条指向注释/空行必须重写；29 条名字不存在（例：`screenButton`、`tokens.go:const`）；
12. `-demo` 专属实现一律不许当「已实现」证据（抽样里 5 行：R-MOD-065、R-MOD-512、R-MOD-514、R-RUN-193、R-GEN-594）；
13. 提交信息与内容对齐：`fa2bedb` 里落的是六块的主要实现（`approval.go` 621 行、`search.go` 950 行、`slash_commands.go` 570 行），
    按提交回溯会漏。

---

## 5. 统计

| 类别 | 条数 |
|---|---:|
| P0 复验：已修 / 部分 / 未修 | 9 / 1 / 2 |
| P1 复验：已修 / 改判后仍有问题 | 13 / 1 |
| 第一轮假测试：已修 / 部分 / 未修 | 3 / 1 / 2（T-1、T-3、T-5 已修；T-2 部分；T-4、T-6 未修） |
| 新增六块：样例冒充 | 0 条新的样例上屏；S-1…S-4 这四块样例背后有 **5 行**需求把它们标成「已实现」（R-MOD-065、R-RUN-193、R-GEN-594、R-MOD-512、R-MOD-514） |
| 新增六块：空壳交互 | 4（N-1…N-4） |
| 新增六块：假事实/口径假 | 7（F-1…F-7），其中 3 条要立刻修（F-1、F-2、P0-1 残留） |
| 「已实现」行（全量） | 2403；逐字文案在函数里 470（19.6%）、文案在别处 431、文案在仓库里不存在 878、行号形态 180、无证据 153 |
| 抽样 80 条：合格 / 弱 / 错位 / 无证据 | 24 / 37 / 18 / 1（合格率 30%） |
| 本轮运行证据 | `go test ./...`、`go vet ./...` 全绿；5 个临时探针（已删除，见附录 A） |

按屏分布（P0+P1 遗留 + 六块新增问题）：

| 区域 | 条数 | 备注 |
|---|---:|---|
| 侧聊 / 右栏 | 2 | P0-9 回声、N-2 `/branch` |
| 子代理 | 3 | P0-10 恒加载、T-4 假测试、N-4 「重试」不重读 |
| 审批卡 | 2 | P0-1 残留（F-6）、F-7 尺寸文案 |
| Fork / 分支 | 2 | F-1、F-2 自造 prompt |
| 导出 / 搜索 | 2 | F-3、F-4 全量口径 |
| 斜杠命令 | 1 | N-1 模式开关 |
| 设置 / 重命名 / 插件 | 2 | F-5 工作区改名、N-3 重新加载恒灰 |
| 时间线 | 1 | P1-12 `RowRetry` 不在连接模式存在 |

---

## 附录 A：本轮取证用的探针（临时文件，跑完删除，不入库）

```go
// internal/ui/zz_audit2_probe_test.go（包内，用完删）
// A-1 审批卡：有工具名、无入参
src := wiredSource(); src.hasAppr = true
src.approval = ApprovalSnapshot{ID: "a9", Tool: "exec", Input: ""}
app := NewApp(Options{State: StateDefault, Source: src})
tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
// 实测：拒绝/仅这次 disabled=false；noteApprovalCommand 与输入区标签都不上屏。

// A-2 侧聊回声
app.rightOpen, app.rightTab = true, tabSideChat
app.panels.sideChatDraft = "这条消息会发给谁"
app.sendSideChat()
// 实测：sideChatSent=["这条消息会发给谁"]，屏上出现该文本。

// A-3 子代理恒加载
app.rightOpen, app.rightTab = true, tabSubagent
// 连跑 5 帧：subagentReady=false，屏上恒有「正在加载...」。

// A-4 / A-5 自造 prompt
app.draft = ""
app.forkAtTurn(3, "在新分支继续")
// 实测：CreateBranch(parent="main", turn=3, prompt="从第 3 轮继续")
// 状态条同一路径实测：prompt="从当前分支切出"
```

证据机检脚本（本轮用 Python 实现，思路可直接复现）：
按 `证据 = <file>:<func>「<literal>」` 建索引 → 用花括号匹配取函数体 → 判断
`literal in body`；`<file>:<digits>` 按行取，判"注释/空行/越界"；都不命中时判
`名称不存在`、`无证据`。统计脚本输出即 §3.2，抽样即 §3.3–§3.4（种子 `20261120`）。

## 附录 B：未覆盖 / 缺什么

- **没有真实 App Server**：全部结论是静态阅读 + 注入 `Source` 渲染 + 单元测试，没有连真服务端逐屏截图。
  要补的证据：`dimagent-ui -connect`（只允许 dimoauth `deepseek v4.1 flash`）下的审批卡、侧聊、子代理、终端、导出/Fork 实操截图。
- **没有长会话夹具**：F-3/F-4 的"只导出/只搜索已加载窗口"是按"`SessionSnapshot` 没有 `HasOlder`、
  投影层有"推出来的，没有用一页以上的真实会话实测。
- **没有读屏实测**：无障碍维度（`role=alert`、播报次数）本轮只做代码回读。
- **终端 pty**：`internal/terminal` 的测试全绿，但没有在真实会话里跑过长命令、没有做过 Ctrl-C / 关进程的实机观察。
- `docs/domains/requirements/requirements-global.md` 在审核期间被改动并提交（`774d244`），
  抽样以审核时的文件内容（`774d244`）为准。
