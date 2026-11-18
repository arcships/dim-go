# 右栏七面板运行时清单（模板 ∪ 运行时）

范围：右栏内的七个面板 —— 文件 / 改动 / 产物 / 终端 / 任务 / 侧聊 / 子代理。本文件只做元素对账（有什么元素、什么条件下在场、参考产品右栏态里是否真的出现），不写行为需求、不写我方命名、不写验收。

## 0. 取证与口径

### 0.1 材料

| 类 | 位置 |
|---|---|
| 模板 | 参考产品源码；仓库根记 `<ref>`，定位写作 `<ref>/<相对路径>:<行号>`，相对 `packages/desktop/packages/renderer/src` |
| 模板（目录） | `<ref>/components/context-panel/**`、`<ref>/components/chat-workspace/{TerminalDrawer,SessionTasksPanel*,TaskStopButton,ProcessDetail*,ProcessStatusIcon,BackgroundTaskTerminal,WakeupDetail,SideChatPanel,SubagentChatPanel}.vue`、`<ref>/components/chat-workspace/timeline/TimelineArtifacts*.vue`、`TimelineFileChanges*.vue`、`<ref>/components/chat-workspace/git/UndoChangesDialog.vue` |
| 运行时 | `docs/domains/requirements/runtime/main/right-panel.json`（状态 R，157 节点，1030×725 CSS 视口，DPR=2）＋同目录 `right-panel.png` |
| 文案 | 参考产品 `zh-CN` 语言包（`<ref>/i18n/zh-CN.ts`）；模板内硬编码照抄；`{x}` 为插值占位 |
| 宿主条件 | 面板挂载与祖先条件取自 `<ref>/components/chat-workspace/ChatWorkspace.vue:6753`–`:7070`（`<aside data-testid="right-panel-shell">` 内的 `v-if` / `:class` 可见性翻转） |

### 0.2 状态 R 里有什么

R = 左栏收起 + **空会话**（「开始构建」）+ 右栏打开，唯一标签为「文件」。

```
右栏标签条：拖拽手柄 · 「文件」标签 · 无文案图标按钮（x=966）· 「展开面板」
文件面板：筛选输入框 · 33 个目录行（全部收起）· 31 个根级文件行
```

因此 R 只覆盖「文件」面板的**默认非搜索、无改动、无展开**一路；其余六个面板在 R 里根本没有被打开（标签位存在、面板未挂载或未激活）。这是本清单的主要局限，见 §9 补采清单。

### 0.3 列义

- **名称（文案原文）**：`t('key')` 的中文原文；`{x}` 为插值占位；无文案记 `（无文案）`；同一控件二态文案记 `A / B`。
- **控件**：按钮 / 图标按钮 / 菜单项 / 子菜单项 / 输入框 / 文本 / 容器 / 行 / 标签 / 徽标 / 骨架 / 分隔线 / 浮层。
- **图标**：该节点实际渲染的图标（lucide 组件名 / `FileIcon` / `ThemeIcon:<name>`）；无则 `—`。**图标单列为独立一行**（当图标有自己的 `v-if` 分支时）或用本列标注宿主行的图标，位置列注明「随行」。
- **截断**：文案被截断的方式（`truncate` = 单行省略；`truncate + title` = 悬停可见完整值；`line-clamp` = 多行）；不截断记 `—`。
- **默认**：无需用户交互即出现记「显示」；仅悬停 / 点击 / 快捷键出现记「隐藏」。
- **条件（分支 ∧ 祖先）**：本节点 `v-if`/`v-else-if`/`v-else`/`v-show` 的分支条件，∧ 面板祖先条件。祖先用方括号缩写：
  - `[文件]` = 右栏打开 ∧ 激活标签 `kind=files` ∧ ContextPanel 已挂载（`contextPanelMounted`）
  - `[改动]` = 右栏打开 ∧ 激活标签 `kind=diff` ∧ `availableTabIds` 含 `diff`
  - `[终端]` = 右栏打开 ∧ 激活标签 `kind=terminal` ∧ TerminalDrawer 已挂载（`terminalPanelWorkspaceCwds` 含 `activePanelCwd`）
  - `[任务]` = 右栏打开 ∧ 激活标签 `kind=background-processes` 或 `session-wakeup`
  - `[侧聊]` = 右栏打开 ∧ 激活标签 `side-chat` ∧ `presentationActive`
  - `[子代理]` = 右栏打开 ∧ 激活标签 `subagent-chat`
  - `[时间线·最新轮]` = 会话时间线有助手回合 ∧ 非 presentation ∧ 非只读 ∧ `turnRunId≠null` ∧ 该回合是最新回合
- **定位**：模板出处；右栏标签条与外壳（标签、展开面板、拖拽手柄）不在本文件范围（见 `inventory-main-rt.md` 与外壳清单），本文件只在下文用「标签位」指代。
- **右栏态（R）**：`可见 <x>,<y> <w>×<h>` = 在 `right-panel.json` 命中，坐标为运行时包围盒；`可见（随行）` = 节点随宿主行出现，但 JSON 未单列（无文案 / 纯 svg / 采集跳过）；`—（条件）` = 未出现且分支条件已解释；`—（死代码）` = 未出现且**没有任何条件可解释**，附行号。

### 0.4 计数与判定规则

- 一行 = 一个模板节点；`v-if` / `v-else-if` / `v-else` 每个分支各成一行；同一节点的纯文案三元（如 `loading ? A : B`）合成一行写 `A / B`，但 `armed` 前后属于不同交互状态，按「未触发 / 已触发」各成一行；图标分支一律单列。
- `v-for` 只记一行（模板节点），实例数写进右栏态列（如 33 个目录行）。
- 「可见」按模板节点计，不按实例计；图标 / 占位 / 无文案节点标「随行」，不计入运行时可对齐的节点数。
- **死代码**判定：该面板已在 R 中渲染（或其祖先唯一宿主恒定成立）后，节点仍缺失且没有分支条件可解释。布局型分支（`show-tab-bar=false` 等）单列「布局性不可达」，不计入死代码。

### 0.5 采集局限（判「未出现」前必读）

1. **R 是空会话**：时间线没有任何回合，故「本轮改动块」「本轮产物块」这两块在 R 里天然不存在，不能据此判死。
2. **右栏只开了「文件」标签**：改动 / 终端 / 任务 / 侧聊 / 子代理五个面板未打开，其内部节点全部只能标条件。
3. **无文案节点被采集跳过**：`+ 新建标签`（`RightPanelBar.vue:386`）、筛选框前置 `Search` 图标、目录箭头、`FileIcon`、行内 `span.w-3.5` 占位在 JSON 里都没有独立节点，只能按宿主行补位并标「随行」。
4. **空会话 → 面板宽度受限**：R 中右栏宽 624（x 405–1029），文件树行宽 616；表内坐标只在这一列宽下成立。
5. **opacity-0 / inert 面板仍会被采集**：任务、侧聊、子代理面板容器常驻 DOM（`v-for` + 可见性翻转），若标签曾打开过，其节点会以 `opacity-0` 出现在后续采集里；R 未触发，故本文件不涉及。
6. **会话态的右栏始终关闭**：`session*.json` 五态都是「右栏关」，因此所有面板在会话态同样不可见（见 §9）。

---

## 1. 面板一：文件（FilesView + FileTreeNode）

祖先：`[文件]`（右栏打开 ∧ 激活标签 kind=files ∧ ContextPanel 已挂载）。面板外壳与标签条由右栏承担。

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 截断 | 默认 | 条件（分支 ∧ 祖先） | 定位 | 右栏态（R） |
|---|---|---|---|---|---|---|---|---|
| F-01 | 筛选文件... | 输入框（占位即文案） | Search（前置） | — | 显示 | `effectiveIsDiffActive ∨ effectiveIsFilesActive` ∧ `[文件]` | `<ref>/components/context-panel/ContextPanel.vue:541`–`:551` | 可见 445,60 563×25 |
| F-02 | （无文案） | 图标 | Search | — | 显示 | 同 F-01 | `ContextPanel.vue:543` | 可见（随行；JSON 未单列） |
| F-03 | （无文案） | 骨架（3 条 32px） | — | — | 显示 | `¬searchMode ∧ loading ∧ sourceFileTree.length===0` ∧ `[文件]` | `<ref>/components/context-panel/FilesView.vue:329`–`:331` | —（条件） |
| F-04 | 正在搜索整个工作区… | 状态文本（次要色） | — | — | 隐藏 | `searchMode ∧ searchStatus='searching'` | `FilesView.vue:333`–`:340` | —（条件） |
| F-05 | 文件搜索失败。 | 错误文本（危险色） | — | — | 隐藏 | `searchMode ∧ searchStatus='error'` | `FilesView.vue:341`–`:350` | —（条件） |
| F-06 | 重试 | 文字按钮 | — | — | 隐藏 | 同 F-05 | `FilesView.vue:345`–`:349` | —（条件） |
| F-07 | 部分文件夹无法搜索，结果可能不完整。 | 警告文本（琥珀色） | — | — | 隐藏 | `searchMode ∧ searchStatus='partial'` | `FilesView.vue:351`–`:358` | —（条件） |
| F-08 | 仅显示前 {count} 个匹配项（`{count}=200`） | 提示文本 | — | — | 隐藏 | `searchMode ∧ searchHasMoreMatches`（独立 `v-if`，非 el 链） | `FilesView.vue:359`–`:366`、`:45` | —（条件） |
| F-09 | （无文案） | 图标（圆形底 48px） | FileIcon | — | 隐藏 | `searchMode ∧ searchStatus='complete' ∧ searchNodes.length===0` | `FilesView.vue:367`–`:373` | —（条件） |
| F-10 | 无匹配文件 | 空态标题 | — | — | 隐藏 | 同 F-09 | `FilesView.vue:374` | —（条件） |
| F-11 | 没有找到包含“{query}”的文件 | 空态说明文本 | — | — | 隐藏 | 同 F-09 | `FilesView.vue:375`–`:377` | —（条件） |
| F-12 | （搜索命中行：图标 + 文件名 + 次级目录 + 改动统计） | 行（`v-for` searchNodes） | FileIcon | 文件名 truncate + title | 显示 | `searchMode ∧ searchNodes.length>0` | `FilesView.vue:380`–`:397` → `FileTreeNode.vue:194` | —（条件） |
| F-13 | 无法加载此文件夹。 | 根级错误行（危险色） | — | — | 隐藏 | `¬searchMode ∧ directoryErrors.has('')` | `FilesView.vue:398`–`:414` | —（条件） |
| F-14 | 重试 / 正在加载... | 文字按钮（同按钮二态） | — | — | 隐藏 | 同 F-13 | `FilesView.vue:407`–`:413` | —（条件） |
| F-15 | （无文案） | 图标（圆形底 48px） | FileIcon | — | 隐藏 | `¬searchMode ∧ sourceFileTree.length===0 ∧ ¬partialDirectories.has('')` | `FilesView.vue:415`–`:421` | —（条件） |
| F-16 | 无匹配文件 | 空态标题 | — | — | 隐藏 | 同 F-15 | `FilesView.vue:422` | —（条件） |
| F-17 | （目录行：箭头 + 文件夹图标 + 目录名） | 行按钮（整行可点、可拖拽） | FileIcon | 目录名 truncate | 显示 | FilesView 主树分支（`v-else`：`¬searchMode ∧ ¬根目录错误 ∧ ¬(树空 ∧ ¬根 partial)`）∧ `node.type='directory'` | `FileTreeNode.vue:112`–`:146` | 可见 410,95 616×28（JSON 共 33 行） |
| F-18 | （无文案） | 图标：已展开箭头 | ChevronDown | — | 隐藏 | `isExpanded` ∧ F-17 | `FileTreeNode.vue:124` | —（条件：R 中目录全部收起） |
| F-19 | （无文案） | 图标：已收起箭头 | ChevronRight | — | 显示 | `¬isExpanded` ∧ F-17 | `FileTreeNode.vue:124` | 可见（随行；JSON 未单列） |
| F-20 | （无文案） | 图标：文件夹 / 打开的文件夹 | FileIcon（`folder` 或 `folder-open`） | — | 显示 | F-17 | `FileTreeNode.vue:127`–`:132` | 可见（随行；JSON 未单列） |
| F-21 | （目录名） | 文本（`.` 开头灰化；有改动后代转琥珀） | — | truncate | 显示 | F-17 | `FileTreeNode.vue:133`–`:141` | 可见 462,100 553×18 |
| F-22 | （无文案） | 容器（子级缩进 `ml-4`） | — | — | 隐藏 | `isExpanded ∧ node.children` ∧ F-17 | `FileTreeNode.vue:146`–`:159` | —（条件） |
| F-23 | （子级目录 / 文件行） | 行（`v-for` children） | 同 F-17、F-27 | — | 显示 | `isExpanded ∧ node.children` ∧ F-17 | `FileTreeNode.vue:147`–`:157`、`:165` | —（条件） |
| F-24 | 加载更多 / 正在加载... | 目录内文字按钮（同按钮二态） | — | — | 隐藏 | `isExpanded ∧ node.children ∧ partialDirs.has(node.path)` ∧ F-17 | `FileTreeNode.vue:165`–`:176` | —（条件） |
| F-25 | 无法加载此文件夹。 | 目录内错误行（危险色；与 F-22 的 `<ul>` 互斥） | — | truncate | 隐藏 | `isExpanded ∧ directoryErrors.has(node.path)` ∧ F-17 | `FileTreeNode.vue:178`–`:187` | —（条件） |
| F-26 | 重试 / 正在加载... | 目录内错误行尾文本 | — | — | 隐藏 | 同 F-25 | `FileTreeNode.vue:188`–`:190` | —（条件） |
| F-27 | （文件行：图标 + 文件名 + 统计 + 暂存点） | 行按钮（整行可点、可拖拽、带右键菜单） | FileIcon | 文件名 truncate | 显示 | 同 F-17 的主树分支 ∨ 搜索命中行 ∧ `node.type='file'` | `FileTreeNode.vue:194`–`:235` | 可见 410,1020 616×28（JSON 共 31 行） |
| F-28 | （无文案） | 占位（对齐目录箭头，14px） | — | — | 显示 | F-27 | `FileTreeNode.vue:206` | 可见（随行；无文案未采集） |
| F-29 | （无文案） | 图标 | FileIcon | — | 显示 | F-27 | `FileTreeNode.vue:208`–`:211` | 可见（随行；JSON 未单列） |
| F-30 | （文件名） | 文本（`.` 开头 / ignored 灰化；有改动转琥珀） | — | truncate | 显示 | F-27 | `FileTreeNode.vue:212`–`:221` | 可见 461,1025 555×18 |
| F-31 | （次级目录路径） | 次级文本（11px） | — | truncate | 隐藏 | `node.secondaryPath` ∧ F-27 | `FileTreeNode.vue:223`–`:225` | —（条件） |
| F-32 | +{added}/-{removed} | 统计文本（等宽） | — | — | 隐藏 | `getFileChangeStats(node)≠null` ∧ F-27 | `FileTreeNode.vue:227`–`:231`、`:70` | —（条件） |
| F-33 | ○ | 徽标（蓝底，已暂存） | — | — | 隐藏 | `isStaged(node)` ∧ F-27 | `FileTreeNode.vue:232`–`:236`、`:75` | —（条件） |
| F-34 | 暂存 | 菜单项 | Plus | — | 隐藏 | `node.change ∧ ¬node.change.staged ∧ uiMode.visibility.fileChanges` ∧ F-27 | `FileTreeNode.vue:240`–`:247` | —（条件） |
| F-35 | 取消暂存 | 菜单项 | Minus | — | 隐藏 | `node.change ∧ node.change.staged ∧ uiMode.visibility.fileChanges` ∧ F-27 | `FileTreeNode.vue:248`–`:255` | —（条件） |
| F-36 | （无文案） | 分隔线 | — | — | 隐藏 | 随 F-34 块 | `FileTreeNode.vue:256` | —（条件） |
| F-37 | 查看差异 | 菜单项 | GitCompare | — | 隐藏 | 同 F-34 | `FileTreeNode.vue:257`–`:260` | —（条件） |
| F-38 | （无文案） | 分隔线 | — | — | 隐藏 | 随 F-34 块 | `FileTreeNode.vue:261` | —（条件） |
| F-39 | 预览 | 菜单项 | Eye | — | 隐藏 | 右键菜单展开 ∧ F-27 | `FileTreeNode.vue:263`–`:266` | —（条件） |
| F-40 | 用外部工具打开 | 菜单项 | ExternalLink | — | 隐藏 | 右键菜单展开 ∧ F-27 | `FileTreeNode.vue:267`–`:270` | —（条件） |
| F-41 | 在文件管理器中显示 | 菜单项 | FolderOpen | — | 隐藏 | `linkInteractions` 已注入 ∧ F-27 | `FileTreeNode.vue:271`–`:278` | —（条件） |
| F-42 | 另存为… | 菜单项 | Download | — | 隐藏 | 同 F-41 | `FileTreeNode.vue:279`–`:286` | —（条件） |
| F-43 | （无文案） | 分隔线 | — | — | 隐藏 | 右键菜单展开 | `FileTreeNode.vue:287` | —（条件） |
| F-44 | 复制路径 | 菜单项 | Copy | — | 隐藏 | 右键菜单展开 | `FileTreeNode.vue:288`–`:294` | —（条件） |
| F-45 | 加载更多 / 正在加载... | 树上卷末尾文字按钮（同按钮二态） | — | — | 隐藏 | `partialDirectories.has('')` ∧ `[文件]` | `FilesView.vue:446`–`:457` | —（条件） |
| F-46 | 没有写入权限，部分操作不可用 | 警告条（琥珀底） | — | — | 隐藏 | `permissionDenied` ∧ `[文件]`（捕获到 EPERM / EACCES） | `ContextPanel.vue:554`–`:559` | —（条件） |
| F-47 | （无文案） | 图标 | AlertTriangle | — | 隐藏 | 同 F-46 | `ContextPanel.vue:558` | —（条件） |
| F-48 | 重试 | 警告条尾部文字按钮 | — | — | 隐藏 | 同 F-46 | `ContextPanel.vue:560`–`:567` | —（条件） |

元素数：48　可见：10（F-01、F-02、F-17、F-19、F-20、F-21、F-27、F-28、F-29、F-30）　未出现（条件）：38　死代码：0

## 2. 面板二：改动（Review）

包含三处：右栏「改动」标签页（ContextPanel 顶部范围栏 + DiffView + 底部统计）、时间线「本轮改动」块、撤销 / 重新应用确认弹窗。

祖先：`[改动]`；时间线块另用 `[时间线·最新轮]`；弹窗挂在本轮改动块内。

### 2.1 范围栏与 Git 动作（`ContextPanel.vue`）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 截断 | 默认 | 条件（分支 ∧ 祖先） | 定位 | 右栏态（R） |
|---|---|---|---|---|---|---|---|---|
| D-01 | （当前范围标签：三态见 D-02–D-04） | 下拉触发器（`data-scope`） | — | — | 显示 | `effectiveIsDiffActive` ∧ `[改动]` | `ContextPanel.vue:443`–`:450`、`:188` | —（条件） |
| D-02 | 未提交改动 | 触发器文本 | — | — | 显示 | `effectiveDiffScope='uncommitted'` | `ContextPanel.vue:449`、`:188` | —（条件） |
| D-03 | 当前分支改动 | 触发器文本 | — | — | 隐藏 | `effectiveDiffScope='branch'` | 同上 | —（条件） |
| D-04 | Last Run 改动 | 触发器文本 | — | — | 隐藏 | `effectiveDiffScope='lastRun'` | 同上 | —（条件） |
| D-05 | 未提交改动 当前工作区改动 | 下拉选项（两段文本） | — | — | 隐藏 | 下拉展开 | `ContextPanel.vue:452`–`:455` | —（条件） |
| D-06 | 当前分支改动 分支与基线差异 | 下拉选项（两段文本） | — | — | 隐藏 | 下拉展开 | `ContextPanel.vue:456`–`:459` | —（条件） |
| D-07 | Last Run 改动 最近一次 agent run 触达的文件 | 下拉选项（两段文本） | — | — | 隐藏 | 下拉展开 | `ContextPanel.vue:460`–`:463` | —（条件） |
| D-08 | 提交或推送 | 按钮（图标 + 文本 + 下拉箭头） | CloudUpload、ChevronDown | — | 显示 | `showGitAction`（有 sessionId ∧ 范围≠lastRun） | `ContextPanel.vue:468`–`:481`、`:265` | —（条件） |
| D-09 | （当前分支名） | 子菜单触发器 | GitBranch | truncate | 隐藏 | `showGitAction ∧ 菜单展开` | `ContextPanel.vue:483`–`:487` | —（条件） |
| D-10 | （本地分支名） | 子菜单项（`v-for` localBranches；当前分支带勾且禁用） | GitBranch、Check | truncate | 隐藏 | 子菜单展开 | `ContextPanel.vue:489`–`:499` | —（条件） |
| D-11 | （无文案） | 分隔线 | — | — | 隐藏 | 菜单展开 | `ContextPanel.vue:502` | —（条件） |
| D-12 | 提交 | 菜单项（无已暂存时禁用） | GitCommit | — | 隐藏 | 菜单展开 | `ContextPanel.vue:503`–`:510` | —（条件） |
| D-13 | 提交并推送 | 菜单项（无已暂存时禁用） | CloudUpload | — | 隐藏 | 菜单展开 | `ContextPanel.vue:511`–`:518` | —（条件） |
| D-14 | 推送 | 菜单项 | CloudUpload | — | 隐藏 | 菜单展开 | `ContextPanel.vue:519`–`:525` | —（条件） |
| D-15 | （无文案） | 图标按钮（加载中旋转） | RefreshCw | — | 显示 | `effectiveIsDiffActive` ∧ `[改动]` | `ContextPanel.vue:528`–`:536` | —（条件） |

### 2.2 改动列表（`DiffView.vue`）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 截断 | 默认 | 条件（分支 ∧ 祖先） | 定位 | 右栏态（R） |
|---|---|---|---|---|---|---|---|---|
| D-16 | （无文案） | 骨架（3 条 32px） | — | — | 显示 | `loading` ∧ `[改动]` | `DiffView.vue:491`–`:493` | —（条件） |
| D-17 | （无文案） | 图标（圆形底 48px） | CheckCircle2 | — | 隐藏 | `¬loading ∧ filteredFiles.length===0` | `DiffView.vue:497`–`:504` | —（条件） |
| D-18 | 没有未提交的改动 | 空态标题（默认） | — | — | 隐藏 | 空态 ∧ `emptyTitle=undefined`（范围=未提交） | `DiffView.vue:505` | —（条件） |
| D-19 | 没有 Last Run 改动 | 空态标题（覆盖） | — | — | 隐藏 | 空态 ∧ 范围=lastRun ∧ `changesUnavailableReason=undefined` | `DiffView.vue:505` → `ChatWorkspace.vue:3948` | —（条件） |
| D-20 | Last Run 改动不可用 | 空态标题（覆盖） | — | — | 隐藏 | 空态 ∧ 范围=lastRun ∧ `changesUnavailableReason≠undefined` | `DiffView.vue:505` → `ChatWorkspace.vue:3949` | —（条件） |
| D-21 | 当前分支没有改动 | 空态标题（覆盖） | — | — | 隐藏 | 空态 ∧ 范围=branch ∧ 非 Git 读取错误 | `DiffView.vue:505` → `ChatWorkspace.vue:3954` | —（条件） |
| D-22 | Git 改动不可用 | 空态标题（覆盖） | — | — | 隐藏 | 空态 ∧ `gitChangesLoadErrorScope===scope`（优先于 D-21） | `DiffView.vue:505` → `ChatWorkspace.vue:3952` | —（条件） |
| D-23 | 工作区是干净的 | 空态说明（默认） | — | — | 隐藏 | 同 D-18 | `DiffView.vue:506` | —（条件） |
| D-24 | 最近一次 agent run 没有记录改动文件。 | 空态说明（覆盖） | — | — | 隐藏 | 同 D-19 | `DiffView.vue:506` → `ChatWorkspace.vue:3960` | —（条件） |
| D-25 | 当前 agent runtime 无法报告最近一次 run 的改动文件。 | 空态说明（覆盖） | — | — | 隐藏 | 同 D-20 | `DiffView.vue:506` → `ChatWorkspace.vue:3961` | —（条件） |
| D-26 | 当前分支与其基线一致。 | 空态说明（覆盖） | — | — | 隐藏 | 同 D-21 | `DiffView.vue:506` → `ChatWorkspace.vue:3966` | —（条件） |
| D-27 | 无法加载 Git 改动，请刷新后重试。 | 空态说明（覆盖） | — | — | 隐藏 | 同 D-22 | `DiffView.vue:506` → `ChatWorkspace.vue:3964` | —（条件） |
| D-28 | 已暂存（{count}） | 分组标题（`role=button`，整行可点） | — | — | 显示 | `stagedFiles.length>0` ∧ 非空态 | `DiffView.vue:513`–`:525` | —（条件） |
| D-29 | （无文案） | 图标：分组已展开 | ChevronDown | — | 隐藏 | `groups[0].expanded` ∧ D-28 | `DiffView.vue:526` | —（条件） |
| D-30 | （无文案） | 图标：分组已收起 | ChevronRight | — | 显示 | `¬groups[0].expanded` ∧ D-28 | `DiffView.vue:527` | —（条件） |
| D-31 | 全部取消暂存 | 分组行尾小按钮（执行中旋转） | Loader2 / Minus | — | 显示 | `groups[0].expanded` ∧ D-28；全 clean / 权限错 / 动作中时禁用 | `DiffView.vue:530`–`:543` | —（条件） |
| D-32 | （无文案） | 容器（`v-show`） | — | — | 显示 | `groups[0].expanded` ∧ D-28 | `DiffView.vue:546` | —（条件） |
| D-33 | （已暂存文件行：图标 + 文件名 + 统计 + 行按钮） | 行（`v-for` stagedFiles） | — | — | 显示 | D-32 | `DiffView.vue:547`–`:577` | —（条件） |
| D-34 | （无文案） | 图标 | FileIcon | — | 显示 | D-33 | `DiffView.vue:564`–`:567` | —（条件） |
| D-35 | （文件名） | 文本（`title` = 完整路径） | — | truncate + title | 显示 | D-33 | `DiffView.vue:568`–`:570` | —（条件） |
| D-36 | +{added} | 统计文本（绿） | — | — | 显示 | D-33 | `DiffView.vue:572` | —（条件） |
| D-37 | -{removed} | 统计文本（红） | — | — | 显示 | D-33 | `DiffView.vue:573` | —（条件） |
| D-38 | 取消暂存 | 行内图标按钮（`aria-label`/`title`） | Minus | — | 显示 | D-33；clean / 权限错 / 动作中时禁用 | `DiffView.vue:578`–`:590` | —（条件） |
| D-39 | 取消暂存 | 右键菜单项 | Minus | — | 隐藏 | 右键菜单展开 ∧ D-33 | `DiffView.vue:592`–`:595` | —（条件） |
| D-40 | （无文案） | 分隔线 | — | — | 隐藏 | 右键菜单展开 ∧ D-33 | `DiffView.vue:596` | —（条件） |
| D-41 | 还原 | 右键菜单项 | Undo2 | — | 隐藏 | 右键菜单展开 ∧ D-33 | `DiffView.vue:597`–`:600` | —（条件） |
| D-42 | （无文案） | 分隔线 | — | — | 隐藏 | 右键菜单展开 ∧ D-33 | `DiffView.vue:601` | —（条件） |
| D-43 | 预览 | 右键菜单项 | Eye | — | 隐藏 | 右键菜单展开 ∧ D-33 | `DiffView.vue:602`–`:605` | —（条件） |
| D-44 | 用外部工具打开 | 右键菜单项 | ExternalLink | — | 隐藏 | 右键菜单展开 ∧ D-33 | `DiffView.vue:606`–`:609` | —（条件） |
| D-45 | （无文案） | 分隔线 | — | — | 隐藏 | 右键菜单展开 ∧ D-33 | `DiffView.vue:610` | —（条件） |
| D-46 | 复制路径 | 右键菜单项 | Copy | — | 隐藏 | 右键菜单展开 ∧ D-33 | `DiffView.vue:611`–`:614` | —（条件） |
| D-47 | （无文案） | 容器（行内联 diff 展开区） | — | — | 隐藏 | `shouldRenderExpandedDiff(file.id)` ∧ D-33 | `DiffView.vue:619`–`:623` | —（条件） |
| D-48 | （无文案） | 代码块（unified diff，隐藏头部 / 复制 / 展开） | — | 横向滚动 | 隐藏 | `file.content!==undefined` ∧ D-47 | `DiffView.vue:624`–`:646` | —（条件） |
| D-49 | 已截断 | 行内提示文本 | — | — | 隐藏 | `file.diffTruncated` ∧ D-47 | `DiffView.vue:625`–`:627` | —（条件） |
| D-50 | 未暂存（{count}） | 分组标题（`role=button`，整行可点） | — | — | 显示 | `unstagedFiles.length>0` ∧ 非空态 | `DiffView.vue:654`–`:666` | —（条件） |
| D-51 | （无文案） | 图标：分组已展开 | ChevronDown | — | 隐藏 | `groups[1].expanded` ∧ D-50 | `DiffView.vue:667` | —（条件） |
| D-52 | （无文案） | 图标：分组已收起 | ChevronRight | — | 显示 | `¬groups[1].expanded` ∧ D-50 | `DiffView.vue:668` | —（条件） |
| D-53 | 全部暂存 | 分组行尾小按钮（执行中旋转） | Loader2 / Check | — | 显示 | `groups[1].expanded` ∧ D-50 | `DiffView.vue:671`–`:684` | —（条件） |
| D-54 | （无文案） | 容器（`v-show`） | — | — | 显示 | `groups[1].expanded` ∧ D-50 | `DiffView.vue:687` | —（条件） |
| D-55 | （未暂存文件行：图标 + 文件名 + 统计 + 行按钮） | 行（`v-for` unstagedFiles） | — | — | 显示 | D-54 | `DiffView.vue:688`–`:718` | —（条件） |
| D-56 | （无文案） | 图标 | FileIcon | — | 显示 | D-55 | `DiffView.vue:705`–`:708` | —（条件） |
| D-57 | （文件名） | 文本（`title` = 完整路径） | — | truncate + title | 显示 | D-55 | `DiffView.vue:709`–`:711` | —（条件） |
| D-58 | +{added} | 统计文本（绿） | — | — | 显示 | D-55 | `DiffView.vue:713` | —（条件） |
| D-59 | -{removed} | 统计文本（红） | — | — | 显示 | D-55 | `DiffView.vue:714` | —（条件） |
| D-60 | 暂存 | 行内图标按钮（`aria-label`/`title`） | Plus | — | 显示 | D-55 | `DiffView.vue:719`–`:729` | —（条件） |
| D-61 | 还原 | 行内图标按钮（`aria-label`/`title`） | Undo2 | — | 显示 | D-55 | `DiffView.vue:730`–`:741` | —（条件） |
| D-62 | 暂存 | 右键菜单项 | Plus | — | 隐藏 | 右键菜单展开 ∧ D-55 | `DiffView.vue:744`–`:747` | —（条件） |
| D-63 | （无文案） | 分隔线 | — | — | 隐藏 | 右键菜单展开 ∧ D-55 | `DiffView.vue:748` | —（条件） |
| D-64 | 还原 | 右键菜单项 | Undo2 | — | 隐藏 | 右键菜单展开 ∧ D-55 | `DiffView.vue:749`–`:752` | —（条件） |
| D-65 | （无文案） | 分隔线 | — | — | 隐藏 | 右键菜单展开 ∧ D-55 | `DiffView.vue:753` | —（条件） |
| D-66 | 预览 | 右键菜单项 | Eye | — | 隐藏 | 右键菜单展开 ∧ D-55 | `DiffView.vue:754`–`:757` | —（条件） |
| D-67 | 用外部工具打开 | 右键菜单项 | ExternalLink | — | 隐藏 | 右键菜单展开 ∧ D-55 | `DiffView.vue:758`–`:761` | —（条件） |
| D-68 | （无文案） | 分隔线 | — | — | 隐藏 | 右键菜单展开 ∧ D-55 | `DiffView.vue:762` | —（条件） |
| D-69 | 复制路径 | 右键菜单项 | Copy | — | 隐藏 | 右键菜单展开 ∧ D-55 | `DiffView.vue:763`–`:766` | —（条件） |
| D-70 | （无文案） | 容器（行内联 diff 展开区） | — | — | 隐藏 | `shouldRenderExpandedDiff(file.id)` ∧ D-55 | `DiffView.vue:769`–`:773` | —（条件） |
| D-71 | （无文案） | 代码块（unified diff，行内嵌） | — | 横向滚动 | 隐藏 | `file.content!==undefined` ∧ D-70 | `DiffView.vue:774`–`:796` | —（条件） |
| D-72 | 已截断 | 行内提示文本 | — | — | 隐藏 | `file.diffTruncated` ∧ D-70 | `DiffView.vue:776`–`:778` | —（条件） |

### 2.3 还原确认弹窗与底部统计

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 截断 | 默认 | 条件（分支 ∧ 祖先） | 定位 | 右栏态（R） |
|---|---|---|---|---|---|---|---|---|
| D-73 | 确认还原未暂存改动？ | 弹窗标题 | — | — | 隐藏 | `pendingUnstagedRevertFile≠null`（点未暂存行「还原」）∧ `[改动]` | `DiffView.vue:807` | —（条件） |
| D-74 | 这会永久丢弃“{path}”中的未暂存改动。 | 弹窗说明文本 | — | — | 隐藏 | 同 D-73 | `DiffView.vue:808`–`:810` | —（条件） |
| D-75 | 取消 | 弹窗次按钮（执行中禁用） | — | — | 隐藏 | 同 D-73 | `DiffView.vue:813`–`:815` | —（条件） |
| D-76 | 还原 | 弹窗主按钮（危险色） | — | — | 隐藏 | 同 D-73 | `DiffView.vue:816`–`:822` | —（条件） |
| D-77 | {count} 个文件改动 | 底部统计左侧文本 | — | — | 显示 | `effectiveIsDiffActive ∧ files.length>0` ∧ `[改动]` | `ContextPanel.vue:660`–`:666` | —（条件） |
| D-78 | +{totalAdded} | 底部统计文本（绿） | — | — | 显示 | 同 D-77 | `ContextPanel.vue:668` | —（条件） |
| D-79 | -{totalRemoved} | 底部统计文本（红） | — | — | 显示 | 同 D-77 | `ContextPanel.vue:669` | —（条件） |

### 2.4 时间线「本轮改动」块（TimelineFileChanges）

祖先：`[时间线·最新轮]` ∧ `uiMode.visibility.fileChanges` ∧ `rows.length>0 ∧ summaryCount>0`。整块位于时间线的折叠页脚区，助手回合运行中时该区高度收拢（`collapse-grid-open = ¬assistantIsRunning`）。

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 截断 | 默认 | 条件（分支 ∧ 祖先） | 定位 | 右栏态（R） |
|---|---|---|---|---|---|---|---|---|
| D-80 | （无文案） | 容器（块根） | — | — | 显示 | `rows.length>0 ∧ summaryCount>0 ∧ uiMode.visibility.fileChanges ∧ [时间线·最新轮]` | `<ref>/components/chat-workspace/timeline/TimelineFileChangesView.vue:51` | 可见 Z2（折叠态，随行；由折叠头 429,452 反推）与 Z3（展开态 ≈410,440 620×90） |
| D-81 | （折叠头：箭头 + 「{count} 个文件改动」） | 按钮（整行可点） | — | — | 显示 | D-80 | `TimelineFileChangesView.vue:54`–`:63` | 可见 Z2 429,452（收起，箭头朝右）；Z3 同位置（已展开，箭头朝下） |
| D-82 | （无文案） | 图标：已展开 | ChevronDown | — | 隐藏 | `expanded` ∧ D-81 | `TimelineFileChangesView.vue:61` | 可见（仅视觉）Z3（PNG：折叠头箭头朝下） |
| D-83 | （无文案） | 图标：已收起 | ChevronRight | — | 显示 | `¬expanded` ∧ D-81 | `TimelineFileChangesView.vue:61` | 可见（仅视觉）Z2（PNG：箭头朝右） |
| D-84 | {count} 个文件改动 | 折叠头文本 | — | truncate | 显示 | `¬isUndone` ∧ D-81 | `TimelineFileChangesView.vue:62`、`:15` | 可见 Z2/Z3（`1 个文件改动`） |
| D-85 | {count} 个文件改动（已撤销） | 折叠头文本（灰化） | — | truncate | 隐藏 | `isUndone` ∧ D-81 | `TimelineFileChangesView.vue:15`、`:57` | —（条件） |
| D-86 | 撤销 | 链接按钮（读取预览中旋转） | Loader2 / RotateCcw | — | 显示 | `showUndo`（`¬isRunning ∧ ¬isUndone ∧ ¬appServerRead ∧ previewStatus≠'unavailable'`） | `TimelineFileChangesView.vue:65`–`:77`、`TimelineFileChanges.vue:74`–`:79` | —（条件） |
| D-87 | 重新应用 | 链接按钮（执行中追加旋转圈） | Redo2、Loader2 | — | 隐藏 | `showReapply`（`isUndone ∧ paths>0 ∧ ¬isRunning ∧ ¬appServerRead`） | `TimelineFileChangesView.vue:79`–`:91`、`TimelineFileChanges.vue:81`–`:86` | —（条件） |
| D-88 | （无文案） | 容器（展开态列表，圆角描边） | — | — | 隐藏 | `expanded` ∧ D-80 | `TimelineFileChangesView.vue:95`–`:100` | 可见 Z3 ≈410,470 620×42 |
| D-89 | （改动文件行：状态字母 + 路径 + 统计） | 行按钮（`v-for` rows，带右键菜单） | — | — | 显示 | D-88 | `TimelineFileChangesView.vue:103`–`:120` | 可见 Z3 ≈410,470 620×34（`rt-approval-demo.txt`） |
| D-90 | M / A / D / ? | 状态字母（等宽，按类型着色） | — | — | 显示 | D-89；无 `statusKind` 时为 `?` | `TimelineFileChangesView.vue:111`–`:115`、`:20`–`:24` | 可见（仅视觉）Z3：本机无 `statusKind`（工作区非 git 仓库），渲染为 `?` 蓝色字 |
| D-91 | （文件路径） | 行内文本（`title` = 完整路径） | — | truncate + title | 显示 | D-89 | `TimelineFileChangesView.vue:116`–`:118` | 可见 Z3（`rt-approval-demo.txt`） |
| D-92 | +{insertions}/-{deletions} | 行内统计（等宽；reportedOnly 时为空串；无增删行数时为 binary） | — | — | 显示 | D-89 | `TimelineFileChangesView.vue:119`、`:32`–`:47` | —（条件） |
| D-93 | 用默认应用打开 | 右键菜单项 | ExternalLink | — | 隐藏 | `localActionsAvailable`（宿主恒传 true）∧ 右键菜单展开 | `TimelineFileChangesView.vue:123`–`:129` | —（条件） |
| D-94 | 在文件管理器中显示 | 右键菜单项 | FolderOpen | — | 隐藏 | 同 D-93 | `TimelineFileChangesView.vue:130`–`:136` | —（条件） |
| D-95 | （无文案） | 分隔线 | — | — | 隐藏 | `localActionsAvailable` | `TimelineFileChangesView.vue:137` | —（条件） |
| D-96 | 复制路径 | 右键菜单项 | Copy | — | 隐藏 | `row.absolutePath` ∧ 右键菜单展开 | `TimelineFileChangesView.vue:138`–`:144` | —（条件） |
| D-97 | 复制相对路径 | 右键菜单项 | Copy | — | 隐藏 | `row.scope='workspace'` ∧ 右键菜单展开 | `TimelineFileChangesView.vue:145`–`:152` | —（条件） |

### 2.5 撤销 / 重新应用确认弹窗（UndoChangesDialog）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 截断 | 默认 | 条件（分支 ∧ 祖先） | 定位 | 右栏态（R） |
|---|---|---|---|---|---|---|---|---|
| D-98 | 撤销改动 | 弹窗标题（undo 模式） | — | — | 隐藏 | `dialogOpen ∧ mode='undo'`（D-86 点击且预览可用） | `<ref>/components/chat-workspace/git/UndoChangesDialog.vue:103`、`:60`–`:62` | —（条件） |
| D-99 | 重新应用改动 | 弹窗标题（reapply 模式） | — | — | 隐藏 | `dialogOpen ∧ mode='reapply'`（D-87 点击） | 同上 | —（条件） |
| D-100 | 撤销本轮的文件改动？ | 弹窗问句 | — | — | 隐藏 | 同 D-98 | `UndoChangesDialog.vue:106`、`:60`–`:62` | —（条件） |
| D-101 | 重新应用本轮的文件改动？ | 弹窗问句 | — | — | 隐藏 | 同 D-99 | 同上 | —（条件） |
| D-102 | （弹窗文件行：状态字母 + 路径 + 统计） | 行（`v-for` files，只读） | — | — | 隐藏 | 弹窗打开 | `UndoChangesDialog.vue:110`–`:124` | —（条件） |
| D-103 | M / A / D | 状态字母（等宽，按类型着色） | — | — | 隐藏 | D-102 ∧ `file.kind` 有值 | `UndoChangesDialog.vue:116`–`:119` | —（条件） |
| D-104 | （文件路径） | 行内文本（`title` = 完整路径） | — | truncate + title | 隐藏 | D-102 | `UndoChangesDialog.vue:120`–`:122` | —（条件） |
| D-105 | +{insertions} / -{deletions} | 行统计（等宽） | — | — | 隐藏 | D-102（有增删行数） | `UndoChangesDialog.vue:123`、`:78`–`:85` | —（条件） |
| D-106 | binary | 行统计文本 | — | — | 隐藏 | D-102 ∧ 增删行数均缺失 | `UndoChangesDialog.vue:76` | —（条件） |
| D-107 | （删除） | 行统计文本 | — | — | 隐藏 | `mode='undo' ∧ file.kind='added'` | `UndoChangesDialog.vue:73` | —（条件） |
| D-108 | +{count} 更多 | 弹窗底部提示文本 | — | — | 隐藏 | `omittedFiles` ∧ 弹窗打开 | `UndoChangesDialog.vue:128` | —（死代码：唯一调用点 `TimelineFileChanges.vue:361` 从不传 `omittedFiles`） |
| D-109 | 取消 | 弹窗次按钮（执行中禁用） | — | — | 隐藏 | 弹窗打开 | `UndoChangesDialog.vue:131`–`:133` | —（条件） |
| D-110 | 撤销改动 / 重新应用 | 弹窗主按钮（危险色 / 主色，执行中旋转） | Loader2 | — | 隐藏 | 弹窗打开（文案与配色随 `mode`） | `UndoChangesDialog.vue:134`–`:142`、`:63`–`:65` | —（条件） |

元素数：110　可见：0　未出现（条件）：109　死代码：1

## 3. 面板三：产物（TimelineArtifacts）

产物不是右栏标签页，而是时间线最新一轮的「本轮产物」块；宿主条件与改动块相同（`TimelineRow.vue:341`）。祖先：`[时间线·最新轮]` ∧ `entries.length>0`（当前轮的可归类产物，`scope='workspace' ∧ changeType≠'deleted' ∧ classifyArtifact≠undefined`）。

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 截断 | 默认 | 条件（分支 ∧ 祖先） | 定位 | 右栏态（R） |
|---|---|---|---|---|---|---|---|---|
| A-01 | （无文案） | 容器（块根） | — | — | 显示 | `entries.length>0` ∧ `[时间线·最新轮]` | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:22`–`:26` | —（条件） |
| A-02 | {count} 个产物 | 计数文本（块头左） | — | truncate | 显示 | A-01 | `TimelineArtifactsView.vue:30`、`:15` | —（条件） |
| A-03 | 提交或推送 | 按钮（图标 + 文本 + 下拉箭头） | CloudUpload、ChevronDown | — | 显示 | `gitAvailable`（当前分支非空）∧ A-01 | `TimelineArtifactsView.vue:32`–`:45`、`TimelineArtifacts.vue:148` | —（条件） |
| A-04 | 提交 | 菜单项 | GitCommit | — | 隐藏 | 菜单展开 ∧ A-03 | `TimelineArtifactsView.vue:47`–`:53` | —（条件） |
| A-05 | 提交并推送 | 菜单项 | CloudUpload | — | 隐藏 | 菜单展开 ∧ A-03 | `TimelineArtifactsView.vue:54`–`:60` | —（条件） |
| A-06 | 推送 | 菜单项 | CloudUpload | — | 隐藏 | 菜单展开 ∧ A-03 | `TimelineArtifactsView.vue:61`–`:67` | —（条件） |
| A-07 | （无文案） | 容器（产物列表，圆角描边） | — | — | 显示 | A-01 | `TimelineArtifactsView.vue:73`–`:76` | —（条件） |
| A-08 | （产物行） | 行（`v-for` `listExpanded ? entries : previewEntries`，带右键菜单） | — | — | 显示 | A-07；折叠时最多 3 行 | `TimelineArtifactsView.vue:77`–`:148`、`:13` | —（条件） |
| A-09 | （无文案） | 图标（文件类型图标） | `<img>` = `getFileIconUrl(entry.path)` | — | 显示 | A-08 | `TimelineArtifactsView.vue:88`–`:92` | —（条件） |
| A-10 | （文件名） | 行内文本（`title` = 完整路径） | — | truncate + title | 显示 | A-08 | `TimelineArtifactsView.vue:93`–`:95` | —（条件） |
| A-11 | 已删除 | 行内标签文本 | — | — | 隐藏 | `¬entry.exists` ∧ A-08（整行半透明） | `TimelineArtifactsView.vue:96`–`:100`、`:80` | —（条件） |
| A-12 | 无法预览文件 | 行内替代文本 | — | — | 隐藏 | `entry.exists ∧ entry.canOpen===false` ∧ A-08 | `TimelineArtifactsView.vue:103` | —（死代码：唯一调用点 `TimelineArtifacts.vue:213` 未提供 `canOpen`，其 `entries` 类型也无此字段） |
| A-13 | 打开 | 行内按钮 | ExternalLink | — | 显示 | `entry.exists ∧ entry.canOpen!==false` ∧ A-08 | `TimelineArtifactsView.vue:104`–`:114` | —（条件） |
| A-14 | （无文案） | 行内下拉触发器 | ChevronDown | — | 显示 | `localActionsAvailable ∨ downloadAvailable` ∧ A-13 | `TimelineArtifactsView.vue:116`–`:126` | —（条件） |
| A-15 | 下载 | 下拉菜单项 | Download | — | 隐藏 | `downloadAvailable` ∧ 菜单展开 | `TimelineArtifactsView.vue:128`–`:130` | —（死代码：唯一调用点 `TimelineArtifacts.vue:213` 未传 `downloadAvailable`，恒 undefined） |
| A-16 | 用默认应用打开 | 下拉菜单项 | ExternalLink | — | 隐藏 | `localActionsAvailable` ∧ 菜单展开 | `TimelineArtifactsView.vue:131`–`:137` | —（条件） |
| A-17 | 在文件管理器中显示 | 下拉菜单项 | FolderOpen | — | 隐藏 | `localActionsAvailable` ∧ 菜单展开 | `TimelineArtifactsView.vue:138`–`:144` | —（条件） |
| A-18 | 全部 {count} 个产物 | 列表底部展开按钮 | ChevronRight | — | 显示 | `showToggle`（entries>3）∧ `¬listExpanded` ∧ A-01 | `TimelineArtifactsView.vue:152`–`:161`、`:14`–`:16` | —（条件） |
| A-19 | 收起 | 列表底部收起按钮 | ChevronDown | — | 隐藏 | `showToggle ∧ listExpanded` ∧ A-01 | `TimelineArtifactsView.vue:160`、`:159` | —（条件） |

元素数：19　可见：0　未出现（条件）：17　死代码：2

## 4. 面板四：终端（TerminalDrawer）

祖先：`[终端]`。**右栏布局把终端当标签页承载，宿主固定传 `show-tab-bar="false"`（`ChatWorkspace.vue:6923`），因此 H 区工具条在右栏态恒不渲染**；新建 / 关闭终端由右栏标签条承担。

### 4.1 H 工具条（右栏布局不渲染）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 截断 | 默认 | 条件（分支 ∧ 祖先） | 定位 | 右栏态（R） |
|---|---|---|---|---|---|---|---|---|
| T-01 | （终端标签：标题 + 关闭） | 标签（`v-for` terminalTabs） | — | — | 显示 | `showTabBar !== false` ∧ `[终端]` | `<ref>/components/chat-workspace/TerminalDrawer.vue:790`–`:820` | —（条件：布局恒 false） |
| T-02 | 关闭进程 | 标签内圆形图标按钮（`aria-label`=关闭进程） | X | — | 显示 | 同 T-01 | `TerminalDrawer.vue:805`–`:812` | —（条件：布局恒 false） |
| T-03 | （终端标签标题 = shell 上报目录末段 / 「终端」） | 标签标题按钮 | — | truncate | 显示 | 同 T-01 | `TerminalDrawer.vue:813`–`:819`、`:154` | —（条件：布局恒 false） |
| T-04 | 新建终端 | 图标按钮（`aria-label`=新建终端；无 cwd 时禁用） | Plus | — | 显示 | 同 T-01 | `TerminalDrawer.vue:826`–`:836` | —（条件：布局恒 false） |
| T-05 | 新建终端 | 悬停提示 | — | — | 隐藏 | 悬停 T-04 | `TerminalDrawer.vue:838`–`:840` | —（条件：布局恒 false） |
| T-06 | 关闭终端 | 图标按钮（`aria-label`=关闭终端） | X | — | 显示 | 同 T-01 | `TerminalDrawer.vue:848`–`:857` | —（条件：布局恒 false） |
| T-07 | 关闭终端 | 悬停提示 | — | — | 隐藏 | 悬停 T-06 | `TerminalDrawer.vue:859`–`:861` | —（条件：布局恒 false） |

### 4.2 I 终端内容与提示

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 截断 | 默认 | 条件（分支 ∧ 祖先） | 定位 | 右栏态（R） |
|---|---|---|---|---|---|---|---|---|
| T-08 | 终端 | 画布容器（`role=application`，`aria-label`=终端，深底 #1a1a1a） | — | — | 显示 | `[终端]` | `TerminalDrawer.vue:782`–`:788` | 可见 Y1/Y2（右栏终端页签；DOM 只剩 `textarea.xterm-helper-textarea`，画布内容见 PNG） |
| T-09 | （无文案） | 画布层（每标签一层，非激活 `invisible`） | — | — | 显示 | `[终端]` ∧ 至少 1 个终端标签 | `TerminalDrawer.vue:867`–`:879` | 可见（仅视觉）Y1/Y2：单标签时该层占满内容区，Y2 内含命令行与 `terminal-ok` 回显；多标签的非激活 `invisible` 分支未采 |
| T-10 | ［进程已关闭］ | 提示行（占满内容区） | — | — | 隐藏 | `terminalTabs.length===0 ∧ terminalNotice`（主动关闭进程） | `TerminalDrawer.vue:880`–`:885`、`:640`；`t('terminal.processClosed')` | —（条件） |
| T-11 | ［进程已退出，代码 {code}］ | 提示行 | — | — | 隐藏 | 同上（进程退出） | `TerminalDrawer.vue:302`–`:308`；`t('terminal.processExited')` | —（条件） |
| T-12 | ［无法打开终端：需要 cwd］ | 提示行 | — | — | 隐藏 | 同上（缺工作目录） | `TerminalDrawer.vue:525`、`:577`；`t('terminal.cwdRequired')` | —（条件） |
| T-13 | [Failed to open terminal: {message}] | 提示行（硬编码英文） | — | — | 隐藏 | 同上（pty 打开 / adopt 失败） | `TerminalDrawer.vue:596` | —（条件） |
| T-14 | [{message}] / 终端操作失败 | 提示行（帧级错误：有 message 带方括号，否则「终端操作失败」） | — | — | 隐藏 | 同上（流帧 `kind='error'` 且未触发自动重开） | `TerminalDrawer.vue:314`–`:327`；`t('terminal.actionFailed')` | —（条件） |

### 4.3 J 右键菜单

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 截断 | 默认 | 条件（分支 ∧ 祖先） | 定位 | 右栏态（R） |
|---|---|---|---|---|---|---|---|---|
| T-15 | 复制选中内容 | 右键菜单项（无选中时禁用） | Copy | — | 隐藏 | 菜单展开 ∧ `[终端]` | `TerminalDrawer.vue:890`–`:897` | 可见 Y3 856,262 74×18（未选中文本，项为禁用态） |
| T-16 | 粘贴 | 右键菜单项（不可粘贴时禁用） | ClipboardPaste | — | 隐藏 | 菜单展开 ∧ `[终端]` | `TerminalDrawer.vue:898`–`:905` | 可见 Y3 856,290 25×18 |
| T-17 | （无文案） | 分隔线 | — | — | 隐藏 | 菜单展开 | `TerminalDrawer.vue:906` | —（条件） |
| T-18 | 清屏 | 右键菜单项 | Eraser | — | 隐藏 | 菜单展开 ∧ `[终端]` | `TerminalDrawer.vue:907`–`:913` | 可见 Y3 856,326 25×18 |
| T-19 | 新建终端 | 右键菜单项（无 cwd 时禁用） | Plus | — | 隐藏 | 菜单展开 ∧ `[终端]` | `TerminalDrawer.vue:914`–`:921` | 可见 Y3 856,354 49×18（有 cwd，可用） |
| T-20 | （无文案） | 分隔线 | — | — | 隐藏 | 菜单展开 | `TerminalDrawer.vue:922` | —（条件） |
| T-21 | 关闭进程 | 右键菜单项（危险色；无活跃终端时禁用） | Square | — | 隐藏 | 菜单展开 ∧ `[终端]` | `TerminalDrawer.vue:923`–`:931` | 可见 Y3 856,390 49×18（危险色）；Y5b 为其效果：点后整个终端页签关闭 |

元素数：21　可见：9（T-08、T-09、T-15、T-16、T-18、T-19、T-21 + T-10/T-11 的否定证据）　未出现（条件）：12（其中 7 项为布局性不可达）　死代码：0

补充证据：`Y4-terminal-last-closed` 关闭最后一个终端页签、`Y5b-terminal-process-closed` 通过右键「关闭进程」结束进程——两次都直接把终端页签关掉并切回上一页签，**没有**渲染 `T-10`（［进程已关闭］）这一行；因此 T-10/T-11 在本产品的右栏终端形态下判为「条件不可达（页签随进程一起关闭）」，而不是缺采。

## 5. 面板五：任务（后台进程 / 唤醒）

祖先：`[任务]`。列表与详情由 `SessionTasksPanel` → `SessionTasksPanelView` 承载，详情按行类型分发 `ProcessDetail`（进程）或 `WakeupDetail`（唤醒）。

### 5.1 K 任务列表与空态

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 截断 | 默认 | 条件（分支 ∧ 祖先） | 定位 | 右栏态（R） |
|---|---|---|---|---|---|---|---|---|
| K-01 | 后台进程 | 列表容器（`role=list` 的 `aria-label`） | — | — | 显示 | `items.length>0 ∧ kind='background-processes'` ∧ `[任务]` | `<ref>/components/chat-workspace/SessionTasksPanelView.vue:81` | —（条件） |
| K-02 | 唤醒 | 列表容器（`role=list` 的 `aria-label`） | — | — | 显示 | `items.length>0 ∧ kind='session-wakeup'` ∧ `[任务]` | `SessionTasksPanelView.vue:81` | —（条件） |
| K-03 | （无文案） | 行容器（`role=listitem`，底部分隔线） | — | — | 显示 | `v-for` items ∧ K-01 或 K-02 | `SessionTasksPanelView.vue:84`–`:87` | —（条件） |
| K-04 | （行主按钮：状态图标 + 标题 + 分支徽标 + 时间） | 行按钮（选中态有底色） | — | — | 显示 | K-03 | `SessionTasksPanelView.vue:88`–`:105` | —（条件） |
| K-05 | （无文案） | 图标：进程运行中（旋转圈） | Loader2 | — | 显示 | `item.kind='process' ∧ status='running'` | `<ref>/components/chat-workspace/ProcessStatusIcon.vue:9` | —（条件） |
| K-06 | （无文案） | 图标：进程已完成（成功色） | CircleCheck | — | 显示 | `status='completed'` | `ProcessStatusIcon.vue:10` | —（条件） |
| K-07 | （无文案） | 图标：进程失败（危险色） | CircleX | — | 显示 | `status='failed'` | `ProcessStatusIcon.vue:11` | —（条件） |
| K-08 | （无文案） | 图标：进程已停止 | Square | — | 显示 | `status='cancelled'`（其余 status 兜底） | `ProcessStatusIcon.vue:12` | —（条件） |
| K-09 | （无文案） | 图标：时钟（进程无 status 或唤醒行） | Clock | — | 显示 | `¬(item.kind='process' ∧ item.status)` ∧ K-03 | `SessionTasksPanelView.vue:96` | —（条件） |
| K-10 | （任务标题） | 行内文本（`title` = 完整标题） | — | truncate + title | 显示 | K-03 | `SessionTasksPanelView.vue:97` | —（条件） |
| K-11 | （分支标识） | 徽标（等宽 10px） | — | truncate + title | 隐藏 | `item.branchId≠'main'` ∧ K-03 | `SessionTasksPanelView.vue:98`–`:103` | —（条件） |
| K-12 | （任务时间） | 行内文本（等宽数字） | — | — | 显示 | K-03 | `SessionTasksPanelView.vue:104` | —（条件） |
| K-13 | 停止 / 取消唤醒 | 行尾图标按钮（`aria-label`/`title` 即文案；进程为「停止」，唤醒为「取消唤醒」） | Square（进程）/ Trash2（唤醒） | — | 显示 | `item.canStop ∧ ¬armed ∧ ¬pending` ∧ K-03 | `<ref>/components/chat-workspace/TaskStopButton.vue:50`–`:61`、`:17`–`:19`；`SessionTasksPanelView.vue:106` | —（条件） |
| K-14 | 确认停止 / 再次点击取消唤醒 | 行尾图标按钮（armed 态，危险色；3 秒超时或失焦复位） | Check | — | 隐藏 | `item.canStop ∧ armed ∧ ¬pending` | `TaskStopButton.vue:58`、`:18`、`:34` | —（条件） |
| K-15 | （无文案） | 图标：执行中（禁用态，旋转） | Loader2 | — | 隐藏 | `pending` | `TaskStopButton.vue:57` | —（条件） |
| K-16 | （无文案） | 图标：行尾旋转圈（无停止按钮的唤醒行） | Loader2 | — | 隐藏 | `¬item.canStop ∧ item.kind='wakeup'` ∧ K-03 | `SessionTasksPanelView.vue:107` | —（条件） |
| K-17 | （无文案） | 容器（详情区，可滚动，顶部有分隔线） | — | — | 显示 | `selected`（列表有选中行） | `SessionTasksPanelView.vue:110`–`:112` | —（条件） |
| K-18 | 暂无运行进程 | 空态文本 | — | — | 隐藏 | `items.length===0 ∧ kind='background-processes'` | `SessionTasksPanelView.vue:113` | —（条件） |
| K-19 | 暂无待唤醒事项 | 空态文本 | — | — | 隐藏 | `items.length===0 ∧ kind='session-wakeup'` | `SessionTasksPanelView.vue:113` | —（条件） |

### 5.2 M 后台进程详情

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 截断 | 默认 | 条件（分支 ∧ 祖先） | 定位 | 右栏态（R） |
|---|---|---|---|---|---|---|---|---|
| K-20 | （无文案） | 图标：进程状态（四态同 K-05–K-08） | ProcessStatusIcon | — | 显示 | 选中行 `kind='process'` | `<ref>/components/chat-workspace/ProcessDetailView.vue:29`、`ProcessDetail.vue:64` | —（条件） |
| K-21 | 运行中 · {time} | 详情头状态文本 | — | truncate | 显示 | `item.status='running'` | `ProcessDetailView.vue:30`–`:32`、`:10`；`t('processPanel.status.running')` | —（条件） |
| K-22 | 已完成 · {time} | 详情头状态文本 | — | truncate | 显示 | `item.status='completed'` | 同上（`processPanel.status.completed`） | —（条件） |
| K-23 | 失败 · {time} | 详情头状态文本 | — | truncate | 显示 | `item.status='failed'` | 同上（`processPanel.status.failed`） | —（条件） |
| K-24 | 已停止 · {time} | 详情头状态文本 | — | truncate | 显示 | `item.status='cancelled'` | 同上（`processPanel.status.cancelled`） | —（条件） |
| K-25 | 复制命令 | 图标按钮（`aria-label`/`title`=复制命令） | Copy | — | 显示 | `item.command!==undefined` | `ProcessDetailView.vue:33`–`:41` | —（条件） |
| K-26 | 已复制 | 按钮内文本（2 秒内）+ 读屏播报 | — | — | 隐藏 | `copied` | `ProcessDetailView.vue:40`、`:42` | —（条件） |
| K-27 | 打开日志 | 图标按钮（`aria-label`/`title`=打开日志） | ExternalLink | — | 显示 | `logAvailable`（ProcessDetail 恒传） | `ProcessDetailView.vue:43`–`:50`、`ProcessDetail.vue:64` | —（条件） |
| K-28 | {toolName} · {startedAt} · pid {pid} · 结束于 {time} | 元信息行（各段按有无拼接） | — | — | 显示 | 任一段有值 | `ProcessDetailView.vue:53`–`:55` | —（条件） |
| K-29 | $ {command} | 命令行文本（等宽、可换行） | — | `break-all` 换行 | 显示 | `item.command!==undefined` | `ProcessDetailView.vue:57`–`:61` | —（条件） |
| K-30 | （无文案） | 容器（只读输出终端外框，最小高 160） | — | — | 显示 | `¬terminalFailed`（ProcessDetail 内 `v-else`） | `ProcessDetail.vue:76` | —（条件） |
| K-31 | （无文案） | 画布（等宽终端，只读，`data-testid=background-task-terminal`） | — | 纵向滚动 | 显示 | `terminalMounted`（详情激活 ∧ 文档可见 ∧ 终端未失败） | `ProcessDetail.vue:77`–`:85`、`:26`–`:28`；`BackgroundTaskTerminal.vue:76`–`:77` | —（条件） |
| K-32 | 预览已截断 | 终端右下覆盖标签（半透明黑底） | — | — | 隐藏 | `previewTruncated`（结构化最终预览超界） | `BackgroundTaskTerminal.vue:78`–`:82` | —（条件） |
| K-33 | 终端操作失败 | 失败提示行（危险色，`role=alert`） | — | — | 隐藏 | `terminalFailed` | `ProcessDetail.vue:65`–`:66` | —（条件） |
| K-34 | 重新连接 | 失败提示行尾按钮 | RotateCw | — | 隐藏 | 同 K-33 | `ProcessDetail.vue:67`–`:74` | —（条件） |

### 5.3 N 唤醒详情

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 截断 | 默认 | 条件（分支 ∧ 祖先） | 定位 | 右栏态（R） |
|---|---|---|---|---|---|---|---|---|
| K-35 | {time} 触发 | 详情头文本 | — | — | 显示 | 选中行 `kind='wakeup'` | `<ref>/components/chat-workspace/WakeupDetail.vue:11`–`:13`；`t('sessionWakeup.fireAt')` | —（条件） |
| K-36 | （唤醒提示内容） | 正文文本（可换行） | — | `whitespace-pre-wrap` 换行 | 显示 | 同 K-35 | `WakeupDetail.vue:14` | —（条件） |

元素数：36　可见：0　未出现（条件）：36　死代码：0

## 6. 面板六：侧聊（SideChatPanel）

祖先：`[侧聊]`。时间线、审批卡、提问卡按其各自清单登记，这里只算面板自有外壳与输入区。

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 截断 | 默认 | 条件（分支 ∧ 祖先） | 定位 | 右栏态（R） |
|---|---|---|---|---|---|---|---|---|
| S-01 | （无文案） | 横幅（橙底，错误条） | — | — | 隐藏 | `error` ∧ `[侧聊]` | `<ref>/components/chat-workspace/SideChatPanel.vue:565`–`:568` | —（条件） |
| S-02 | （错误文本） | 横幅文本 | — | — | 隐藏 | 同 S-01 | `SideChatPanel.vue:569` | —（条件） |
| S-03 | 重试 | 横幅尾部文字按钮（加载中禁用） | — | — | 隐藏 | `payload===null ∧ branchId!==undefined` ∧ S-01 | `SideChatPanel.vue:570`–`:580` | —（条件） |
| S-04 | 此分支已结束，不再接收消息。历史仍可在此查看；完成后可关闭此标签页。 | 状态条（灰底，顶部分隔线） | — | — | 隐藏 | `branchClosed`（status='ending' ∨ 'ended'） | `SideChatPanel.vue:582`–`:588`、`:154`–`:155` | —（条件） |
| S-05 | （无文案） | 内容区（复用 PlainTimeline，只读） | — | — | 显示 | `payload≠null ∧ messages.length>0` ∧ `[侧聊]` | `SideChatPanel.vue:590`–`:604` | —（条件） |
| S-06 | （无文案） | 图标：空态旋转圈 | Loader2 | — | 隐藏 | `¬S-05 ∧ (loading ∨ sending ∨ branchRunning)` | `SideChatPanel.vue:607` | —（条件） |
| S-07 | 此侧边对话会从所选消息之前开始。 | 空态文本 | — | — | 隐藏 | `¬S-05 ∧ ¬S-06 ∧ branchId===undefined ∧ beforeMessageId!==undefined` | `SideChatPanel.vue:608`–`:613` | —（条件） |
| S-08 | 就当前 thread 提问。侧边对话会继承启动时的上下文。 | 空态文本 | — | — | 隐藏 | `¬S-05 ∧ ¬S-06 ∧ ¬S-07` | `SideChatPanel.vue:614` | —（条件） |
| S-09 | （权限卡位） | 卡位（复用 InlinePermissionPrompt） | — | — | 隐藏 | `¬branchClosed ∧ pendingInteraction.kind='permission'` | `SideChatPanel.vue:618`–`:624` | —（条件） |
| S-10 | （提问卡位） | 卡位（复用 InlineElicitationPrompt） | — | — | 隐藏 | `¬branchClosed ∧ pendingInteraction.kind='elicitation'` | `SideChatPanel.vue:625`–`:631` | —（条件） |
| S-11 | 问一个旁支问题… | 输入框（占位即文案；分支已关闭 / 发送中 / 分支未就绪时禁用） | — | — | 显示 | `[侧聊]`（输入区常驻） | `SideChatPanel.vue:636`–`:642`、`:639` | —（条件） |
| S-12 | 结束分支 | 图标按钮（`title`=结束分支；悬停转危险色） | CircleStop | — | 显示 | `canEndBranch`（激活 ∧ 分支已建 ∧ ≠main ∧ payload≠null ∧ status='open' ∧ ¬ending） | `SideChatPanel.vue:644`–`:654`、`:163`–`:169` | —（条件） |
| S-13 | 停止 | 图标按钮（`title`=停止；停止请求中禁用） | Square | — | 隐藏 | `branchRunning` | `SideChatPanel.vue:655`–`:665`、`:156`–`:161` | —（条件） |
| S-14 | 发送 / 引导 | 图标按钮（`title` 二态：分支运行中为「引导」，否则「发送」；`canSend` 为假时禁用） | Send | — | 显示 | `[侧聊]` | `SideChatPanel.vue:666`–`:675`、`:170`–`:176` | —（条件） |
| S-15 | 结束此分支？ | 确认弹窗标题 | — | — | 隐藏 | `endConfirmOpen`（点 S-12） | `SideChatPanel.vue:682` | —（条件） |
| S-16 | 此操作会永久结束该分支并中断当前回合。分支将不再接收消息，但历史仍可在此标签页中查看。 | 确认弹窗说明文本 | — | — | 隐藏 | 同 S-15 | `SideChatPanel.vue:683` | —（条件） |
| S-17 | 取消 | 确认弹窗次按钮 | — | — | 隐藏 | 同 S-15 | `SideChatPanel.vue:686` | —（条件） |
| S-18 | 确认 | 确认弹窗主按钮（危险色；结束请求中禁用） | — | — | 隐藏 | 同 S-15 | `SideChatPanel.vue:687`–`:694` | —（条件） |

元素数：18　可见：0　未出现（条件）：18　死代码：0

## 7. 面板七：子代理（SubagentChatPanel）

祖先：`[子代理]`。面板本身无输入区；时间线复用只读 `PlainTimeline`。

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 截断 | 默认 | 条件（分支 ∧ 祖先） | 定位 | 右栏态（R） |
|---|---|---|---|---|---|---|---|---|
| G-01 | （无文案） | 横幅（橙底，行内错误条） | — | — | 隐藏 | `streamError ∧ ready` ∧ `[子代理]` | `<ref>/components/chat-workspace/SubagentChatPanel.vue:127`–`:131` | —（条件） |
| G-02 | （错误消息） | 横幅文本 | — | truncate | 隐藏 | 同 G-01 | `SubagentChatPanel.vue:132` | —（条件） |
| G-03 | 重试 | 横幅尾部文字按钮 | RefreshCw | — | 隐藏 | 同 G-01 | `SubagentChatPanel.vue:133`–`:136` | —（条件） |
| G-04 | （错误消息） | 空态错误文本（警告色） | — | — | 隐藏 | `streamError ∧ ¬ready` | `SubagentChatPanel.vue:139`–`:141` | —（条件） |
| G-05 | 重试 | 空态错误按钮（次级样式） | RefreshCw | — | 隐藏 | 同 G-04 | `SubagentChatPanel.vue:142`–`:145` | —（条件） |
| G-06 | （无文案） | 图标：加载旋转圈 | Loader2 | — | 显示 | `¬streamError ∧ ¬ready`（`aria-busy`） | `SubagentChatPanel.vue:148`–`:150` | —（条件） |
| G-07 | 正在加载... | 加载态文本 | — | — | 显示 | 同 G-06 | `SubagentChatPanel.vue:148`–`:152` | —（条件） |
| G-08 | （无文案） | 内容区（复用 PlainTimeline，只读） | — | — | 显示 | `ready` ∧ `[子代理]` | `SubagentChatPanel.vue:154`–`:167` | —（条件） |

元素数：8　可见：0　未出现（条件）：8　死代码：0

## 8. 合计

| 面板 | 章节 | 现清单区域 | 元素数 | 可见 | 未出现（条件） | 死代码 |
|---|---|---|---|---|---|---|
| 文件 | §1 | B | 48 | 10 | 38 | 0 |
| 改动 | §2 | C + F | 110 | 0 | 109 | 1 |
| 产物 | §3 | G | 19 | 0 | 17 | 2 |
| 终端 | §4 | H + I + J | 21 | 0 | 21 | 0 |
| 任务 | §5 | K + L + M + N | 36 | 0 | 36 | 0 |
| 侧聊 | §6 | O | 18 | 0 | 18 | 0 |
| 子代理 | §7 | P | 8 | 0 | 8 | 0 |
| **合计** | 7 章 | 16 区域中的 13 个 | **260** | **10** | **247** | **3** |

恒等式：260 = 10 可见 + 247 未出现（条件） + 3 死代码。

三处死代码：

| 元素 | 文案 | 行号 | 原因 |
|---|---|---|---|
| D-108 | +{count} 更多 | `UndoChangesDialog.vue:128` | 唯一调用点 `TimelineFileChanges.vue:361` 不传 `omittedFiles` |
| A-12 | 无法预览文件 | `TimelineArtifactsView.vue:103` | 唯一调用点 `TimelineArtifacts.vue:213` 不传 `canOpen`，其 `entries` 类型也无该字段 |
| A-15 | 下载 | `TimelineArtifactsView.vue:128` | 唯一调用点不传 `downloadAvailable` |

另有两类「条件恒假但不算死代码」的分支，单列备查：

| 类别 | 元素 | 恒假原因 | 处理 |
|---|---|---|---|
| 布局性不可达 | T-01–T-07（7 项） | 右栏宿主固定 `show-tab-bar=false`（`ChatWorkspace.vue:6923`）；组件仍保留抽屉布局分支 | 计入「未出现（条件）」，不计死代码 |
| 客户端形态相关 | D-86、D-87、D-98–D-110（15 项） | `showUndo`/`showReapply` 要求 `¬appServerRead()`；本产品由 App Server 只读会话，撤销 / 重新应用在本形态下不出现（App Server 模式没有本地 File History RPC，`TimelineFileChanges.vue:328`–`:340` 明说） | 保留为条件项，实施时按「不实现或只读置灰」决策 |

## 9. 待补采状态（按优先级）

R 只覆盖「文件面板 + 空会话」，可见率 10/260。以下状态按面板归类，供统一补采；括号内是它能解锁的行段。

| 序 | 状态 | 前置条件 | 解锁 |
|---|---|---|---|
| 1 | 右栏·改动（有改动，Git 仓库，未提交 + 已暂存混合） | 工作区有 ≥2 个改动文件、其中 1 个已暂存 | D-01–D-15、D-28–D-79（约 67 行） |
| 2 | 右栏·改动·行内 diff 展开 + 未暂存行右键菜单 | 点击未暂存行、再右键同一行 | D-39–D-49、D-62–D-72、D-73–D-76 |
| 3 | 右栏·改动·空态三态（未提交 / 分支 / Last Run） | 干净工作区、切范围下拉 | D-17–D-27 |
| 4 | 右栏·文件·搜索三态 + 限流 + 空结果 | 在筛选框输入命中多 / 无命中 / 触发 200 上限 | F-04–F-12 |
| 5 | 右栏·文件·展开目录 + 加载更多 + 目录错误 | 展开大目录、构造不可读目录 | F-18、F-22–F-26、F-45 |
| 6 | 右栏·文件·改动标记 | 有改动的文件树 | F-32–F-44（含右键菜单） |
| 7 | 右栏·终端（1 个终端 + 右键菜单） | 打开终端标签、右键终端 | T-08、T-09、T-15–T-21 |
| 8 | 右栏·终端·无标签提示（关闭进程后） | 关闭最后一个终端 | T-10 |
| 9 | 右栏·后台进程（运行中 / 完成 / 失败 + 详情） | 触发后台进程工具 | K-01、K-03–K-15、K-17、K-20–K-34 |
| 10 | 右栏·唤醒（有 / 无待唤醒） | 建一条唤醒 | K-02、K-13–K-16、K-19、K-35、K-36 |
| 11 | 右栏·侧聊（空态 / 有消息 / 运行中 / 已结束 / 结束确认） | 从消息处开侧聊并跑一轮 | S-01–S-18 |
| 12 | 右栏·子代理（加载 / 时间线 / 错误） | 派一个子代理任务 | G-01–G-08 |
| 13 | 时间线·本轮改动块（折叠 / 展开 / 已撤销） | 一轮有文件改动的回合 | D-80–D-97 |
| 14 | 时间线·本轮产物块（≤3 行 / 展开 / 已删除） | 一轮产出 ≥4 个产物文件 | A-01–A-19 |
| 15 | 会话态右栏（先开右栏再进会话） | 会话中打开右栏 | 全部面板的「祖先在场」验证，替换本文件所有「未出现」的证伪路径 |
| 16 | 窄窗 replacement 布局（右栏覆盖式） | 收窄窗口至 replacement 断点 | 面板宽度、标签条「返回」、任务/侧聊在覆盖式下的表现 |

## 10. 与既有清单的对照

| 本文件 | `inventory-panels.md` | 差异 |
|---|---|---|
| §1 文件 | B（36 项） | 本文件把目录展开箭头、`FileIcon`、行内占位拆成图标行，并把根级 / 目录级「重试 / 正在加载」合为二态一行 → 48 行 |
| §2 改动 | C（54 项）+ F（26 项） | 合并两区为一面板；空态标题按 `emptyTitle` 覆盖拆成 5 行；撤销弹窗以独立小节登记 → 110 行 |
| §3 产物 | G（17 项） | 拆出图标行与两个下拉菜单项状态 → 19 行；标出 2 处死代码 |
| §4 终端 | H + I + J（17 项） | 补 T-14（帧级错误提示），H 区标为布局性不可达 → 21 行 |
| §5 任务 | K + L + M + N（36 项） | 停止按钮按「kind × armed」二态合为两行、图标分支单列；列表状态图标拆出 → 36 行 |
| §6 侧聊 | O（18 项） | 数量一致，措辞按模板校订 |
| §7 子代理 | P（6 项） | 拆出「正在加载...」与旋转圈两行 → 8 行 |

不在本文件范围（避免重复登记）：

- 右栏标签条与外壳（标签页、＋、选项卡右键菜单、展开面板、拖拽手柄）→ 外壳清单。
- 面板内文件预览 / 文档预览（`FilePreviewView`、`OfficeFilePreview`、`OfficeToolbar`）→ 由预览清单承担；本文件只在「文件标签页」面板层登记到面板内容为止。
- `ContextPanel` 内置标签条（`TabBar.vue`）→ 右栏布局 `show-tab-bar=false`，恒不渲染。
- 侧聊 / 子代理内复用时间线、审批卡、提问卡 → 各自主清单。
