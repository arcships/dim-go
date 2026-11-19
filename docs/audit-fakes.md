# 打假审核报告（界面事实、空壳交互、证据与测试）

- 审核对象：`HEAD` = `ad61f44`（`docs(organization): add an independent audit against fake data and empty actions`）
- 审核方式：静态阅读 + 在 HEAD 的临时副本上注入假 `Source` 渲染取证（不改仓库代码）
- 审核范围：`internal/ui`、`internal/uiwire`、`cmd/dimagent-ui`，以及
  `docs/domains/requirements/requirements-{main,panels,settings}.md` 的「已实现」行
- 说明：本机工作区在本轮审核期间出现了**未提交的并发改动**（`internal/ui/terminal.go`、
  `rightpanel_*.go`、`app.go`、`source.go`、`uiwire/*`）。下文所有行号都按 `HEAD`
  的 `git archive` 副本核对，不按工作区当前文件。

判定口径（来自审核要求）：

- P0 界面显示假事实：界面上的数据点无法追到 `Source` / `PanelFacts` / `Intents`，
  且发生在 `source != nil`（连接模式）。
- P1 空壳交互：按钮/菜单项点了只改本地状态，不发 Intent，也不禁用并写原因。
- P2 证据错位：`requirements-*.md` 标「已实现」的 `文件:行` 指向的不是实现该元素该维度的行。

约束：本报告只出结论与复现方式，不改实现、不提交。

---

## 0. 先说结论

1. **样例并没有被完全关在 `-demo` 里。** 终端、标题栏菜单、审批卡、设置 › Git /
   宠物 / 插件设置 / 运行环境 / 多模态 / 外观 / 通用 / 个性化 / 权限、侧聊、子代理，
   在连接模式下都会画样例常量或屏内全局初值，其中至少 5 处是有事实可读、却根本
   没读（`Settings().Runtimes`、`Settings().Modality*`、`ModaliyTasks`、
   `RuntimesKnown`），另 2 处是"界面自己编一串回执"。
2. **最严重的一条不是"样例上屏"，是审批卡**：真实审批的命令来自事实，但命令下方
   的 `cwd` 是常量 `~/dim-agent`，命令为空时还会回落成 `ls -la`。用户在这一屏上
   做的是允许/拒绝执行的决定。
3. **空壳交互集中在设置区与标题栏**：14 条里有 9 条是"开关/下拉能拨、能选、能打字，
   但不落盘也不发意图，也没有任何禁用或原因"。其中"提交后自动推送"这类安全相关项
   也只留在内存。
4. **证据层面比实现层面更糟**：7017 条「已实现」里有 875 条没有 `文件:行`；
   有行号的 6142 条里 41.7% 指向注释/空行，42.1% 的带「」期望文案在所引文件里
   根本找不到。抽查 60 条严格可用 1 条（1.7%）。
5. **两个校准例复核**：
   - 终端 `newTerminal` 不启 pty 只清状态：**成立**（`terminal.go:430`），且不止它，
     同屏的"关闭进程/清屏/模拟断线"同样只在本地造状态（见 P0-11）。
   - 文件页"无会话/无 cwd 时回落样例树"：**在 HEAD 上不成立**。`source != nil` 时
     样例树根本不会被建（`rightpanel_parts.go:170-172`）。实际问题是另一条：
     无会话时静默改读**工作区根目录**，界面不说明这不是会话目录（见 P1-14）。

---

## 1. P0：界面显示假事实

| 编号 | 位置 | 现象 | 为什么是假 | 怎么验 |
|---|---|---|---|---|
| P0-1 | `internal/ui/composer.go:406`、`composer.go:389-392` | 审批卡第三行恒为 `cwd: ~/dim-agent`；待决审批的 `Command` 为空时，卡片显示 `$ ls -la` | `ApprovalSnapshot` 只有 `ID/Command/Impact/LongTerm`（`source.go:433-438`），**根本没有 cwd 字段**，这一行不可能是事实；空命令回落到样例命令，等于替用户编了一条要执行的命令 | 注入 `ApprovalSnapshot{ID:"a1", Command:""}` 渲染：屏上同时出现 `ls -la` 与 `~/dim-agent`（探针 `TestAuditApprovalCardCwd` 输出两项均为 true） |
| P0-2 | `internal/ui/app.go:1719-1727`；显示点 `composer.go:521,528` | 连接模式下模型入口显示 `seed-2.1-pro` | `currentModel()` 在 `Settings().Model == ""` 时直接 `return sampleModel`。连接早期、或服务端还没报选择、或无可用模型时，这里就是样例品牌名。`liveSettings()` 只要 `source != nil` 就返回 ok（`app.go:1746-1751`），所以这个回落必然命中 | 注入源把 `Settings().Model/ModelID/Models` 置空后渲染，屏上出现 `seed-2.1-pro`（探针 `TestAuditNoModelFact`）；或 `dimagent-ui -connect` 在模型清单到达前截图 |
| P0-3 | `internal/ui/settings_git_screen.go:92-93,108-125` | 设置 › Git 在连接模式显示远程 `origin/upstream`、基线分支 `main/develop/release`，两个下拉、两个开关、全部输入都是屏内状态 | 整屏没有任何 `Source` / `Intents` 读取；`gitRemotes`/`gitBranches` 是写死的样例切片，`var gitScreen = gitState{}` 在进程内。文件头自认"接线时把七处读写换成 Source / Intents"，但没有任何"未接线"的界面提示 | 用注入 Source 打开 `secGit`：屏上仍有 `origin`、`main`（探针 `TestAuditGitScreenSamples`） |
| P0-4 | `internal/ui/settings_pets_screen.go:141-149` | 设置 › 宠物显示两个内置 Pet（`Dim`、`dim-fox`）且 `dim-fox` 已选中 | 描述符是码里写死的初值，Pet 领域既没有 `Source` 也没有 `Intents`；界面把它当"当前可选的 Pet 清单 + 当前选择"呈现 | 注入 Source 打开 `secPets`：屏上有 `dim-fox`（探针 `TestAuditSettingsScreensWithSource` 输出 true） |
| P0-5 | `internal/ui/settings_plugin_settings_screen.go:92-120` | 设置 › 插件设置显示 3 张插件卡（`Model log` 等）与 6 个字段实例 | 同样是包级初值；没有任何插件 schema 读取路径 | 注入 Source 打开 `secPluginSettings`：屏上有 `Model log`（探针输出 true） |
| P0-6 | `internal/ui/settings_runtime_screen.go:76-133`；文案 `:166-176` | 设置 › 运行环境恒显示错误卡「服务端未提供该能力：The App Server does not advertise runtime.list…」，功能组件恒显示「未安装」「配套版本 1.0.135-desktop.1 · 下载 13.0 MB」 | 适配器**已经**把 `settings.Runtimes()` 读进 `SettingsSnapshot.Runtimes/RuntimesKnown`（`uiwire/adapter.go:646-655`），但 `internal/ui` 里 `.Runtimes`/`RuntimesKnown` 除定义处（`source.go:509-511`）**没有任何读取点**。也就是说：有事实、不显示，还反过来断言服务端没提供 | 注入源给 `RuntimesKnown=true` + 一条 `server-real v9.0.0 ready`，打开 `secRuntime`：屏上仍是那句"未声明 runtime.list"（探针输出 true） |
| P0-7 | `internal/ui/settings_modality_screen.go:48-60,88-89`；`mdGroupCount` | 设置 › 多模态恒显示「服务端未提供该能力：…modality.capabilities」、分组计数恒 `0 / N`、任务区恒「没有任务。」 | 适配器已读 `settings.Modality()` 与 `ListTasks`（`uiwire/adapter.go:657-663,255-280`），快照里有 `Modality.Supported/TaskSupported` 与 `ModalityTasks`；`internal/ui` 里这两项除定义外**零读取**（`source.go:514-518`）。服务端声明了能力、给了任务，界面照样说没声明 | 注入 `Modality{Supported:true,TaskSupported:true,TaskKnown:true}` + 一条 `ModalityTasks`，打开 `secModality`：横幅与「没有任务」仍在（探针 `TestAuditModalityIgnoresFacts` 两项 true） |
| P0-8 | `internal/ui/settings_appearance_screen.go:456-480`；`app.go:1390-1405` | 设置 › 外观 › 主题卡有"选中"角标，点另一张卡角标会移动 | 点卡只写包级 `appearanceScreen.theme`（`:479`），**没有任何代码读它**；真正决定主题的是 `a.themePref`，而全仓只有 ctor / reset / 读盘三处写它（`app.go:588,799,857`），**没有任何 UI 入口能改**。所以：卡片显示的"当前主题"是假的，主题也改不动 | 在连接模式下点「主题：深色」：`app.themePref` 仍为 `"跟随系统"`，`appearanceScreen.theme` 变成 1（探针 `TestAuditThemeCardDoesNotApply`）；界面配色不变 |
| P0-9 | `internal/ui/rightpanel_sidechat.go:143-150` | 侧聊输入一句话点发送，消息立即出现在只读时间线上 | `sendSideChat()` 只做 `a.panels.sideChatSent = append(...)` 的本地回声；`Source` / `Intents` 都没有侧聊事实与方法（全仓 `sideChat` 只出现在 `internal/ui`）。用户会以为侧分支真的收到了问题 | 连接模式下把 `panels.sideChatDraft` 设为文本并调用 `sendSideChat()`：`len(sideChatSent)==1`，无任何请求发出（探针 `TestAuditSideChatEcho`） |
| P0-10 | `internal/ui/rightpanel_subagent.go:35-70`；`rightpanel.go:71` | 右栏「子代理」页在连接模式永远停在「正在加载…」 | `panels.subagentReady` 没有任何写入点（全仓只有读取与测试里赋值），`subagentLines` 也没人填。它是**恒假状态**，不是加载中 | 连接模式打开子代理页：一直转圈；`grep -n "subagentReady =" internal/ui/*.go` 只有测试命中 |
| P0-11 | `internal/ui/terminal.go:430-446`（新建）、`:449-454`（关闭进程）、`:220-224`（清屏）、`:232-241`（模拟断线/重连） | 连接模式下：点「关闭进程」内容区变成「［进程已关闭］」；点「新建终端」清空画布回到空闲；点「清屏」清本地行；点「模拟断线」改本地状态 | `TerminalSnapshot` 只读；上述动作都不发 `Intents`（`Intents` 里也确实没有 create/close/reconnect 终端的方法）。真实进程还在跑，界面却声称已关闭；"清屏"下一帧事实又把行画回来（每帧从 Source 读），"模拟断线"被 `terminalView()` 的快照覆盖成完全无效果 | 注入 `Terminal{Active:true,Running:true,Lines:["real output"]}`，调 `newTerminal()`：`terminalView()` 返回 `wired=true, lines=0, state=idle`，而源里 `Running` 仍为 true（探针 `TestAuditTerminalActions`） |
| P0-12 | `internal/ui/header.go:21-29,349-370` | 标题栏「更多打开方式」列出 VS Code / Zed / Terminal / Ghostty 四款，注释称"采集到本机可用的四款应用"；选一项只写一行回执「正在用 X 打开」；「在 Finder 中显示」写「已在 Finder 中显示」；「用系统默认方式打开」写「正在用系统默认方式打开」 | 四个应用是写死的切片，没有探测；三条动作与选择应用都没有任何打开调用（`openWithSystem` 只用于打开日志，`header.go` 里没有）。回执是编的 | 在没装 Ghostty 的机器上连接模式打开该菜单：仍列出 Ghostty；点它后标题下出现"正在用 Ghostty 打开"，系统里什么都没发生。`grep -rn "openWithSystem" internal/ui` 只有日志一处在用 |

## 2. P1：空壳交互（点了像生效）

| 编号 | 位置 | 现象 | 为什么算空壳 | 怎么验 |
|---|---|---|---|---|
| P1-1 | `internal/ui/header.go:156-159` | 标题栏 ⋯ ›「归档」点击后标题下出现「已归档」 | 只写 `headerNote`，没有 `Intents.ArchiveSession`；左栏同名菜单（`sidebar.go:900`）和命令面板（`command_palette.go:721`）都是真发，这一处是假回执 | 连接模式点该菜单项；服务端收不到 `session.archive`，列表不变，标题下方多出"已归档" |
| P1-2 | `internal/ui/header.go:140-145` | 标题栏 ⋯ ›「置顶 / 取消置顶」 | 只翻 `a.chatPinned`，既不发意图也不落盘（`viewPreferences()` 里没有这一项），重启即丢 | 点一次 → 重启 → 回到「置顶」 |
| P1-3 | `internal/ui/header.go:146-148` | 标题栏 ⋯ ›「重命名」恒禁用，括注「（SessionTitle 不可用）」 | 假原因：`Intents.RenameSession` 存在（`source.go:97`）且左栏行内改名在用（`sidebar.go:865-885`）。同一能力在左栏可用、在标题栏说不可用 | 左栏改名成功；同会话标题栏同一项灰色不可点 |
| P1-4 | `internal/ui/sidebar_state.go:416-421` | 左栏会话行「置顶」 | `map[int]bool` 按**行号**记，不落盘（`ViewPreferencesSnapshot` 只有 `PinnedGroups`），列表按活跃时间重排后会置顶到别的行 | 置顶第 1 行 → 有新消息使排序变化 → 第 1 行换人，置顶跟着换；重启全丢 |
| P1-5 | `internal/ui/statusbar.go:159-168` | 状态条分支菜单「创建新分支」连接模式下可点，点了没反应 | `if !wired {…}` 之外没有任何 else；`Intents.CreateBranch` 存在（`source.go:111`）。既不禁用也不写原因 | 连接模式点该项：菜单关闭、无请求、无提示 |
| P1-6 | `internal/ui/composer.go:837-839` | 空态工作区菜单「添加新工作区」 | 只执行 `a.workspaceName = "新工作区"`；没有目录选择、没有落盘、没有提示。菜单关掉后界面上看不出发生过什么 | 点它并重开菜单：选项不变 |
| P1-7 | `internal/ui/settings_general_screen.go:39-46,58-80` | 设置 › 通用：「保持电脑唤醒」「性能日志」「登录时自动启动」三个开关、会话数据根目录、代理三档与自定义地址、导入会话、清理浏览器数据全部可操作 | 全是 `var generalScreen` 进程内字段，没有任何 `Intents`；`PreferencesSnapshot` 里也没有这些键。开关位置本身也是样例初值（`keepAwake:true, launch:true`），界面把它当"当前设置" | 拨「登录时自动启动」→ 重启客户端 → 回到"开"；系统登录项没有变化 |
| P1-8 | `internal/ui/settings_appearance_screen.go:409-420,479,496-620,697-800` | 设置 › 外观：主题卡、语言、UI/代码字体、UI 字号、代码字号、代码主题、皮肤全部可点可选 | 除主题外全部只写包级 `appearanceScreen`；主题见 P0-8。`SetPreferences` 只覆盖主题/动效/时区/侧栏宽/工作区/Hooks 目录/通知三项/视图状态（`app.go:1220-1229`），这些项一个都没落盘，也没有一个影响渲染 | 把 UI 字号从"中"改到最大：任何一屏的字都没变 |
| P1-9 | `internal/ui/settings_personalization_screen.go:15-55` | 设置 › 个性化：界面模式、展示 ACP 会话、默认打开目标应用、子代理模型/上限、自定义指令都可操作 | 文件头自认"点击不写事实；变更只留在屏内状态里"，但界面没有任何"未保存/不可保存"的标记，控件也不禁用 | 改「界面模式」→ 切节再回来 → 回到"经典" |
| P1-10 | `internal/ui/settings_permissions_screen.go:156,219-220,246-257` | 设置 › 权限「在沙箱中运行命令」「允许网络访问」两个开关可自由拨动；保存按钮禁用且写原因 | 开关状态与事实无关（初值 `sandbox=false,network=false` 是码里写的，不是 `Settings().Tier` 之类的读数）；因为保存被禁用，拨动**永远不可能**生效，却没有任何提示告诉用户"这两下白拨" | 连接模式拨开关：界面显示"开"，`permIsolationDirty()` 变 true，但保存不可点、没有任何"未落盘"提示 |
| P1-11 | `internal/ui/settings_git_screen.go:111-145` | 设置 › Git 的「始终强制推送」「提交后自动推送」「Commit/PR 指令补充」「Worktree 根目录」都能改 | 只改 `gitScreen`；其中两项是安全相关（强制推送、自动推送），界面上看起来是已生效的策略 | 打开开关 → 重启 → 回到关 |
| P1-12 | `internal/ui/timeline.go:796-804` | 自动重试行有「立即重试」按钮，但没有任何点击处理 | `gui.Button(c,"立即重试").Label(...)` 之后没有 `.Clicked()` 分支，是纯装饰 | 点它：无任何请求、无反馈 |
| P1-13 | `internal/ui/rightpanel_sidechat.go:121-130,160-170` | 侧聊「停止」把 `sideChatSending=false`；「结束分支」把 `sideChatEnded=true` | 只有布尔；没有任何结束分支/停止的通道 | 点「结束分支」→ 确认：状态条出现"此分支已结束"，服务端分支照旧 |
| P1-14 | `internal/ui/rightpanel_files.go:78-83`、`uiwire/panel.go:708-721` | 没有打开会话（`rpCwd()==""`）时，文件页改读**工作区根目录** | 代码注释里这是有意的回落（"空表示按工作区根读"），但**界面上没有任何提示**说这不是会话目录；用户会把工作区根当成会话目录。（这是对校准例 2 的更正：不是"回落样例"，是"回落工作区根"） | 连接模式不开会话、打开右栏文件页：列出的是仓库根，而不是"还没有打开会话" |

## 3. P2：证据错位（requirements 的「已实现」）

### 3.1 全量统计（三个 requirements 文件）

| 项 | 数量 | 占比 |
|---|---|---|
| 标「已实现」的行 | 7017 | 100% |
| 有可解析 `文件:行` 证据 | 6142 | 87.5% |
| 无行号（只给文件名或"屏级：xxx"） | 875 | 12.5% |
| 有行号、但所引行是注释或空行 | 2561 | 41.7%（占 6142） |
| 带「」逐字期望文案的行 | 4427 | — |
| ├ 期望文案就在所引那一行 | 1730 | 39.1% |
| ├ 在 ±5 行内 | 275 | 6.2% |
| ├ 在 ±25 行内 | 207 | 4.7% |
| ├ 在同一文件别处 | 351 | 7.9% |
| └ 所引文件里根本没有这段文案 | 1864 | **42.1%** |
| 证据指向样例/屏内初值代码 | 51 | 0.8% |

### 3.2 抽查 60 条（随机种子 `20261119`，三个文件按行均匀抽）

| 分类 | 定义 | 条数 | 占比 |
|---|---|---|---|
| A 可用 | 所引行就是实现该元素该维度的代码行 | 1 | 1.7% |
| B 弱 | 所引行是该元素所在函数/段的注释，等于"在这一段里" | 22 | 36.7% |
| C 错位 | 所引行与元素无关（常量、括号、别的段、样例数据、根本没实现的分支） | 25 | 41.7% |
| D 无证据 | 该行没有 `文件:行` | 12 | 20.0% |

**结论：严格可用 1.7%（1/60）；错位 + 无证据 61.7%（37/60）。** 这与 3.1 的机器统计
（41.7% 指向注释、42.1% 文案对不上）互相印证。

### 3.3 具体例子（都按 HEAD 行号）

| 编号 | 证据位置 | 现象 | 为什么假 | 怎么验 |
|---|---|---|---|---|
| P2-1 | `requirements-settings.md:971` R-ACC-719「有额度数值 → 显示 10.8万 / 30万」→ 证据 `internal/ui/settings_account_screen.go:366` | 所引行是 `accountSampleData()` 里的 `Amount: "10.8万 / 30万"` | 拿 **-demo 样例数值**当"已实现"证据。连接模式下这一屏读 `Account()/Usage()`，该字符串永远不会上屏 | 打开该行看引用；连接模式渲染账户屏，屏上无此串 |
| P2-2 | `requirements-settings.md:3656` R-APP-028「断开网络改主题卡，等保存返回失败 → 角标回滚」 | 标已实现 | 外观屏没有任何保存路径（见 P0-8），"保存失败回滚"这条支不存在 | 读 `settings_appearance_screen.go:456-480`：只有 `appearanceScreen.theme = index` |
| P2-3 | `requirements-settings.md:9463` R-GIT-077「无分支 → 禁用并提示」→ `settings_git_screen.go:152` | 所引行是 `gitSelectGroup` 的注释 | 代码里没有"无分支"分支；`gitBranches` 恒为非空样例，下拉永远可点 | 读 `settings_git_screen.go:111-145` |
| P2-4 | `requirements-main.md:1896` R-TL-415 → `internal/ui/timeline.go:384` | 所引行是一个 `})` | 括号不是实现 | `sed -n 384p internal/ui/timeline.go` |
| P2-5 | `requirements-main.md:198` R-TB-143 → `internal/ui/header.go:57` | 所引行是空行 | 空行不是实现 | `sed -n 57p internal/ui/header.go` |
| P2-6 | `requirements-settings.md:4485` R-APP-636「皮肤卡的 FolderOpen 图标」→ `settings_appearance_screen.go:696` | 所引行是"4.3 我的皮肤"段注释；同一行被 **151 条**需求共用 | 一个段注释当 151 个元素/维度的证据，等于没有证据 | 统计脚本：`grep -c` 该行号在 requirements-settings.md 的出现次数 |
| P2-7 | `requirements-settings.md:3623-3624` R-APP-001/002 → `settings_appearance_screen.go:417` | 所引行是 `appearanceScreen` 初值块里的 `codeFont: "默认",`（`:417`） | 用"屏内初值"当"已实现"，与 P0-8 直接冲突 | 同 P2-6 |
| P2-8 | `requirements-main.md:191` R-TB-136 → `internal/ui/sidebar.go:324` | 期望文案「在工作区中新建会话」在 `sidebar.go` 里不存在 | 文案对不上，说明要么元素在别处、要么没做 | `grep -n "在工作区中新建会话" internal/ui/sidebar.go` 无输出 |
| P2-9 | 三个文件里 875 行（例：「已实现（屏级：settingsAccountScreen）」「已实现（证据 = internal/ui/sidebar.go）」） | 没有行号 | 无法核对，且与"可验收"冲突 | 见 3.1 统计脚本 |
| P2-10 | 需求行与实现相互矛盾的一类 | 例：`requirements-settings.md:9463`(R-GIT-077) 与同文件 `R-GIT-075/079` 一致地写「未实现：缺协议」；同一屏同一元素组里"显示"标已实现、"保存"标未实现，但界面并不告诉用户这些值只是内存 | 用户按文档验收会得到错误结论 | 对照 P0-3 / P1-11 |

## 4. P2：假测试 / 坏测试

| 编号 | 位置 | 现象 | 为什么是假证据 | 怎么验 |
|---|---|---|---|---|
| T-1 | `internal/uiwire/intents_test.go:362-395` `TestConcurrentUIInteractions` | **在 HEAD 上稳定失败**：等 `"附件名"` 超时（约 950 帧） | `AGENTS.md §5` 要求 `go test ./...` 通过；HEAD 不通过，任何"测试全绿"的验收陈述都不成立 | `go test ./internal/uiwire -run TestConcurrentUIInteractions -count=1`（HEAD 副本上两次均 FAIL） |
| T-2 | `internal/ui/wired_samples_test.go:50-56` `samplePhrases` | 声称"接上事实后任何一条都不该出现在视图里"，但清单里没有 `seed-2.1-pro`、`~/dim-agent`、`ls -la`、`dim-fox`、`Model log`、`origin` | 它挡住的只是第一片样例，且用例只用一份"事实齐全"的注入源（`wiredSource()` 给了 `Model="Model-X"`）。P0-1/2/3/4/5 的样例一个都不在清单里；一旦某一项事实缺失，P0-2 立刻上屏而测试仍绿 | 用同一用例但把 `src.settings.Model` 置空：屏上出现 `seed-2.1-pro`，现有断言不会失败 |
| T-3 | `internal/ui/settings_wired_test.go` 全套 | "wired"用例只覆盖 `secMCP` / `secHooks` / `secProviders` / `secGit`（其中 `secGit` 只在 -demo） | 恰好是全样例驱动的 Git/宠物/插件/运行环境/多模态/外观/通用/权限/归档，**没有一条连接模式用例**。测试命名给了"设置屏已接线"的错觉 | `grep -n "openSettingsSection(t, app, sec" internal/ui/settings_wired_test.go` |
| T-4 | `internal/ui/rightpanel_test.go:232-237` | 断言子代理页出现「正在加载...」 | 把恒加载的假状态断言成正确行为（P0-10），等于给假状态上锁 | 同上；且 `subagentReady` 无写入点 |
| T-5 | `internal/ui/app_test.go:1207` `TestComposerShowsModel` | 断言屏上有 `seed-2.1-pro` | 把样例常量固化成 UI 期望值；连接模式复用同一常量时反而没人拦 | 同 T-2 |
| T-6 | `internal/ui/snapshot_test.go` 系列 | 截图比对全部基于 -demo 样例 | 截图通过≠连接模式正确；需求文档又把截图当"已实现"证据（见 3.3） | 读文件头：`图里只有样例，没有事实` |

---

## 5. 统计

| 类别 | 条数 |
|---|---|
| P0 假事实 | 12 |
| P1 空壳交互 | 14 |
| P2 证据错位（条目） | 10 |
| P2 假测试（条目） | 6 |
| 需人工确认 | 5（见附录 A） |
| 「已实现」行（全量） | 7017，其中 875 无行号、41.7% 指向注释、42.1% 文案对不上 |
| 抽查 60 条严格可用率 | 1.7% |

按屏分布（P0+P1 合计）：

| 区域 | 条数 | 备注 |
|---|---|---|
| 设置区 | 11 | 全部集中在 Git / 通用 / 外观 / 个性化 / 权限 / 宠物 / 插件 / 运行环境 / 多模态 |
| 标题栏 | 4 | 归档、置顶、重命名、打开方式 |
| 右栏 | 4 | 侧聊、子代理、文件页、侧聊停止/结束 |
| 输入区 | 3 | 审批卡、模型槽、工作区菜单 |
| 终端 | 1 | 新建/关闭进程/清屏/断线 |
| 状态条 | 1 | 创建新分支 |
| 左栏 | 1 | 会话置顶 |
| 时间线 | 1 | 立即重试 / 失败原文 |

---

## 6. 最该先修的 10 条

排序依据：假事实是否影响用户决策 → 是否可能造成安全/数据误解 → 修复成本。

1. **P0-1 审批卡**：删掉 `cwd: <样例>` 这一行（没有事实就不画），命令为空时写"命令不可读"并禁用三个决定按钮。
2. **P0-11 终端关闭进程**：连接模式下没有关闭通道，应禁用并写原因；"新建/清屏/断线"同理（清屏可以留，但必须说"只清本屏，输出会被服务端读数覆盖"）。
3. **P0-6 运行环境**：接 `Settings().Runtimes/RuntimesKnown`；读不到时用 `RuntimesReason` 写真实原因，不要写死"服务端未提供"。
4. **P0-7 多模态**：接 `Modality` 与 `ModalityTasks`；只有 `Supported=false` 时才画横幅。
5. **P0-2 模型槽**：`Settings().Model` 为空时显示"未选择/未读到"，禁止回落 `sampleModel`（这是全应用最容易被当成"事实"的单值）。
6. **P0-8 外观主题**：主题卡改成写 `a.themePref` 并 `persistPreferences()`；其余外观项要么接上落盘，要么整块置灰。
7. **P0-9 侧聊**：没有分支/消息事实时，输入区禁用并写"服务端没有提供侧聊"，删掉本地回声。
8. **P0-3 + P1-11 设置 › Git**：整屏按"未接"处理（占位或禁用 + 原因），不要画样例远程/分支和可拨开关。
9. **P1-1 / P1-3 标题栏 ⋯**：归档改走 `ArchiveSession`；重命名恢复可用（`RenameSession` 已存在）。
10. **P2（证据线）**：把「已实现」的 `文件:行` 全部换到实现行，取消"段注释/屏级/样例数据"三类证据；同时修 T-1，让 `go test ./...` 回到绿。

---

## 附录 A：需人工确认（缺证据）

| 编号 | 位置 | 存疑点 | 缺什么证据 |
|---|---|---|---|
| A-1 | `internal/ui/timeline.go:1450`、`timeline.go:796-804` | 重试行把 `row.Duration` 当"还有 N 秒"显示（`%ds 后重试`）。若 `Duration` 是"已等待时长"而不是"剩余"，文案就是反向的 | 需要 `RowSnapshot.Duration` 在 `RowRetry` 上的语义定义（投影领域） |
| A-2 | `internal/ui/timeline.go:785-794` | `RowError` 的 `row.Text`（真实错误原文）被丢弃，恒画「本次运行失败。」 | 需确认服务端是否在这一行给错误原文；若给了，这是信息丢失而非造假 |
| A-3 | `internal/ui/sidebar.go:838-861`、`app.go:1066-1090` | 左栏选中项用 `"s"+index`（行号）映射到会话标识，列表按活跃时间/置顶重排后是否还指向同一会话 | 需一次真实多会话（含置顶 + 新消息重排）的端到端观察 |
| A-4 | `internal/ui/rightpanel_files.go:78-83` | 无会话时读工作区根目录（P1-14）：是"合理的默认目录"还是"误导" | 需产品口径：空态应不应该直接读工作区根 |
| A-5 | `cmd/dimagent-ui/windows.go:621-645` | 每个窗口 `Source = uiwire.Session`，`PanelFacts` 靠内嵌 `*App` 继承；多窗口时 `FileTree` 的 `Cwd` 是否按窗口隔离 | 需要多窗口 + 不同 cwd 会话的实机观察（本次未跑真实服务端） |

## 附录 B：本次取证用的探针（放在 HEAD 副本里临时加，不入库）

```go
// 在 internal/ui 下临时加一个包内测试文件即可复现 P0-2 / P0-1 / P0-3..8。
func TestAuditNoModelFact(t *testing.T) {
	src := wiredSource()          // 现有测试里的注入源
	src.settings.Model = ""       // 事实里没有模型选择
	app := NewApp(Options{State: StateDefault, Source: src})
	tt := gui.NewTester(app.View, WindowWidth, WindowHeight)
	if strings.Contains(strings.Join(tt.Texts(), "\n"), "seed-2.1-pro") {
		t.Log("连接模式屏上出现样例模型")
	}
}
```

真实服务端验证（需 dimoauth `deepseek v4.1 flash`，AGENTS §3.1）：

```bash
dimagent-ui -connect -data-dir <tmp>            # 看审批卡 cwd 行、模型槽、标题栏 ⋯ 归档
dimagent-ui -connect -variant right-panel       # 看文件页/终端/侧聊/子代理
```

## 附录 C：未覆盖

- `internal/app`、`internal/projection`、`internal/settings`、`internal/extensions` 的
  事实生产端（本次只审"界面拿事实的方式"）。
- `cmd/dimagent`（更新器）与托盘/通知路径。
- 真实 App Server 端到端：本机未接入，全部结论为静态阅读 + 注入源渲染。
- 设置 › 账户 / 用量 / MCP / Hooks / 快捷键 / 通知 / 模型供应：抽查未见样例上屏，
  但未逐条核对需求行。

---

## 附录 D：需求状态列重标（2026-11-19）

§3 的结论（「已实现」的证据只有 1.7% 站得住）已按下面的规则重标三个需求文件的
`状态` 列。本条与附录 A–C 同属打假范围：只改状态列与证据，不改实现、不改其它列。

### D.1 规则（本轮判定口径）

1. **合格证据只有两种形态**：
   - `已实现（证据 = <文件>:<函数/常量>「<逐字文案>」）`——文案必须是该函数体（或它
     引用的界面文案常量）里的原样字符串；
   - `已实现（证据 = <文件>:<函数> + 标记 <代码行为标记>）`——标记是该函数里能证明该
     维度行为的**真实代码行**（如 `.FontSize(12.25)`、`.Label(text)`、
     `Disabled(true)`、`m.Item("折叠全部").Chosen()`、`c.Theme()`）；
     必要时附 `（常量 X「文案」）` 说明文案出处。
2. 拿不出上面任一形态的，一律改成 `未实现（缺：…）`，并写明缺什么（缺文案、缺取值、
   缺交互落点、缺状态分支、缺读屏标注、缺协议事实等）。
3. **不写行号**：行号会腐烂；证据只到「文件 + 函数/常量名 + 逐字文案或行为标记」。
   三个文件里已无 `证据 = …go:<行号>` 形态（`requirements-global.md` 本来就没有状态列）。
4. 已在 `未实现` / `不适用` 的行本轮不动（它们本来就没声明「已实现」）。

### D.2 判定方法

- 用 Go AST 建索引：全仓非测试函数的区间、函数体、文档注释、字符串字面量、
  包级常量/变量声明、同文件调用关系。
- 元素 → 渲染点：优先取注释里带元素 ID 的函数；否则在元素所属屏文件里，按清单
  第二列的元素名/图标名→界面文案常量→引用该常量的函数定位。
- 文案比对是逐字的；`{count}` / `%d` 这类占位按同形匹配，匹配到后证据抄**代码里的
  原样字符串**，不抄需求里的占位写法；跨字符串拼接的文案只在整段函数里做归一化匹配，
  并降级为「标记 + 文案」写法。
- 样例/演示数据（`*sample*`、`*demo*`）一律不作为证据；据此可判「未实现」。
- 维度关口：`样式`（取值必须出现在元素自己的样式行上）、`深色`（元素必须从
  `c.Theme()` 取色，且期望色值要在 `internal/ui/tokens.go` 有对应 token）、
  `无障碍`（标记必须含 Label/Role/Description，或菜单/按钮/文本控件本身）、
  `交互/键盘/动画/记忆`（标记必须在元素那一行，不跨元素借用）、
  `状态`（只有「显示 X」类断言允许用逐字文案，其余要有开/关、选中、禁用、隐藏的取值）、
  `异常`（文案要是错误/失败类，且函数里要有失败分支）、`关联/窄窗/高DPI/性能/输入法/拖拽`
  （没有逐元素落点的一律未实现）。

### D.3 分批统计（每 500 条一批）

| 文件 | 批 | 条数 | 合格证据 | 改判未实现 | 抽检通过 |
|---|---:|---:|---:|---:|---:|
| requirements-settings.md | 1 | 500 | 72 | 428 | 20/20 |
| requirements-settings.md | 2 | 500 | 67 | 433 | 20/20 |
| requirements-settings.md | 3 | 500 | 86 | 414 | 20/20 |
| requirements-settings.md | 4 | 500 | 75 | 425 | 20/20 |
| requirements-settings.md | 5 | 500 | 68 | 432 | 20/20 |
| requirements-settings.md | 6 | 500 | 52 | 448 | 20/20 |
| requirements-settings.md | 7 | 500 | 87 | 413 | 20/20 |
| requirements-settings.md | 8 | 500 | 94 | 406 | 20/20 |
| requirements-settings.md | 9 | 500 | 87 | 413 | 20/20 |
| requirements-settings.md | 10 | 500 | 68 | 432 | 20/20 |
| requirements-settings.md | 11 | 356 | 24 | 332 | 20/20 |
| requirements-main.md | 1 | 500 | 57 | 443 | 20/20 |
| requirements-main.md | 2 | 500 | 32 | 468 | 20/20 |
| requirements-main.md | 3 | 72 | 0 | 72 | 20/20 |
| requirements-panels.md | 1 | 500 | 69 | 431 | 20/20 |
| requirements-panels.md | 2 | 89 | 22 | 67 | 20/20 |
| **合计** | 16 | **7017** | **960** | **6057** | **320/320** |

### D.4 三个文件各自的三个数

| 文件 | 原「已实现」 | 重标后合格证据 | 改判未实现 |
|---|---:|---:|---:|
| `requirements-settings.md` | 5356 | 780 | 4576 |
| `requirements-main.md` | 1072 | 89 | 983 |
| `requirements-panels.md` | 589 | 91 | 498 |
| 合计 | 7017 | 960（13.7%） | 6057（86.3%） |

未被本轮改动的行：`不适用` 724 条（产品形态到不了，不是「已实现」声明）、
原本已标 `未实现` 的 6256 条。

### D.5 抽检

- **机械回读**：每批随机 20 条，按「证据里的文案是否在该函数/常量里、标记是否是
  该函数里的原样代码」回读代码，320/320 通过（见 D.3 最后一列）。
- **人工语义复核**：另抽 20 条逐条读函数，判断「这段代码是不是这个元素、这个维度」，
  合格 17/20（85%）。不合格的三条是三类问题的样本，规则里已写清去向：
  ① 复合行借用「兄弟元素的标记」（同一行的按钮标记被算到旁边的标签上）；
  ② 负向断言（「不单独播报 / 不渲染」）没有对应分支，只能判未实现；
  ③ 框架默认行为（菜单项播报、控件焦点）没有逐元素落点。
  本轮已把 ② 收进规则、把 ③ 留在 D.6 局限里登记。
- 未达标判据（<17/20）：本轮的机械回读 16 批全部 20/20；人工语义复核 17/20，
  刚好在门禁线上，余下 3/20 的成因见上。

### D.6 已知局限（登记，不粉饰）

1. **误判「未实现」的方向**：文案/取值经变量、经服务端下发、或由框架默认行为提供的
   能力，只要代码里没有该元素那一行的落点，本轮一律判 `未实现`，缺什么写在括号里。
   这类条目是「证据不足」，不等于产品没做；要翻案得补一条能回读的代码落点或界面取证。
2. **`不适用` 未复核**：724 条「产品形态到不了」是上一轮的判定，本轮没有重审。
3. **未跑真实服务端**：全部判定是代码回读，没有连接 App Server 逐屏取证；
   `cmd/dimagent-ui` 之外的证据（截图、读屏实测）本轮不采。
4. 需求文件的元素名、场景、期望、验收、异常列一个字未动；本附录与三个状态列是
   本轮唯一改动。
