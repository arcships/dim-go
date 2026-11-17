# 全局面板与首次运行：元素清单

作用：按 `docs/project-organization.md` §9.16 二，穷举参考产品 dim-electron 的全局面板（搜索、命令、弹窗卡片）与首次运行/登录路径上的界面元素。单位固定为 **屏 → 区域 → 分组 → 元素**，每元素六项：元素 ID / 名称（文案原文）/ 控件 / 默认 / 条件 / 定位。

边界：只登记"有什么元素、文案是什么、什么条件下在场"。不写行为需求（点后做什么、失败如何重试、顺序如何），不写我方命名，不自造参考产品没有的元素。

## 0 来源与取证状态

- 事实源与证据：本版为**参考产品源码逐元素抽取**。定位一律给 `组件文件:行号`（相对 `packages/desktop/packages/renderer/src/`）并附 i18n 键；文案取 `i18n/zh-CN.ts` 原文。
- 行号口径：指向承载该元素的那一行——模板里有对应节点时指模板行；文案由计算属性产出时指该键所在的行（同组件内一跳可达）。同一行号出现在连续多行是正常的（该分支一次渲染多个元素）。
- 未附截图裁片。§9.16 七禁止"未跑参考产品就凭源码写用户看到什么"，故本清单只承担"元素在不在、文案是什么、由哪段模板产生"；证据列待各屏跑通参考产品后按屏补，补齐前该屏不进入行为需求阶段。
- 覆盖口径：一个面板 = 参考产品里一次出现的独立浮层/弹窗/卡片。同一元素的多状态（禁用、选中、加载、错误、按类型切换文案）记在该元素的「默认 / 条件」里，不另立元素；只有**文案完全不同且由不同分支渲染**的形态才分列，以免实现时漏文案。

## 1 屏与区域

屏：全局面板（Global panels）。区域按参考产品界面结构派生：

| 区域 | 覆盖 | 分组 |
|---|---|---|
| A 搜索 | 会话内搜索、会话搜索对话框、`@` 提及、`$` 技能提及、命中高亮 | 5 |
| B 命令 | 斜杠命令弹层、`ui/command` 基元 | 2 |
| C 审批 | 工具权限卡、沙箱越权卡、子代理模型卡、审批输入区 | 4 |
| D 提问 | 提问卡 | 1 |
| E 确认 | 编辑运行中消息、删除会话、归档会话、删除工作区 | 4 |
| F 重命名 | 会话、工作区 | 2 |
| G 导出与派生 | 导出对话、Fork 会话 | 2 |
| H 导入与更新 | 插件安装（3）、更新清单（2，插件+技能）、Chrome Cookie 导入、浏览器数据清理 | 7 |
| I 附件与选择 | 图片预览、文本附件、选中文本气泡、宠物活动 | 4 |
| J Git 与定时 | 提交、新分支、切换分支、撤销/重新应用、创建定时任务 | 5 |
| K 账户与首次运行 | 企业邮箱登录、微信扫码、飞书扫码、账户入口与退出确认、首次运行欢迎、更新提示、Appshots 引导与权限 | 7 |

合计 43 分组。

## 2 区域 × 元素数

派复核按区域进行，一区域的数字独立可验（逐组数字见文末「计数汇总」）。

| 区域 | 分组数 | 元素数 | 复核范围 |
|---|---|---|---|
| A 搜索 | 5 | 41 | A1–A5 |
| B 命令 | 2 | 29 | B1–B2 |
| C 审批 | 4 | 63 | C1–C4 |
| D 提问 | 1 | 22 | D1 |
| E 确认 | 4 | 21 | E1–E4 |
| F 重命名 | 2 | 16 | F1–F2 |
| G 导出与派生 | 2 | 22 | G1–G2 |
| H 导入与更新 | 7 | 56 | H1–H7 |
| I 附件与选择 | 4 | 31 | I1–I4 |
| J Git 与定时 | 5 | 59 | J1–J5 |
| K 账户与首次运行 | 7 | 65 | K1–K7 |
| **合计** | **43** | **425** | — |

## 3 边界说明

三条边界已定，后续复核按此口径，不再讨论：

1. **设置类弹窗归设置域**。`settings/DataCleanupDialog.vue`、`settings/ProxyConfigDialog.vue`、`settings/Modality*.vue` 以及 `ui/dialog/*`、`ui/popover/*`、`ui/alert-dialog/*` 这些表单基元，元素明细记在设置域（`docs/domains/D11a-settings.md` 方向），本清单只按需引用（B2 引 `ui/command`，H6/H7 是设置页内的导入与清理对话框，已登记），不重复计数。
2. **浏览器权限请求不登记**。参考产品对浏览器会话的系统权限请求在 `electron-main/src/services/preview-security.ts` 里是静默判定（默认拒绝，仅放行受信任的剪贴板写入），不弹任何 UI，因此没有可登记元素。应用内唯一的权限类对话框是 Appshots 的录入/录屏引导（K7），系统级授权由 macOS 自己的弹窗承担，不在参考产品界面内。
3. **运行时包裹不单列**。`TimelineSearchDomText.vue` 只是把命中片段包一层 `<span>` 的运行时实现，与 `TimelineSearchHighlightedText.vue` 同一套 class、同一视觉效果，合并计入 A5，不另立元素。

---

## A 搜索

### A1 会话内搜索浮层（ThreadSearchBar）

条件：主区有活跃会话时按 `Cmd/Ctrl+F` 打开；浮层绝对定位在标题栏右侧，点浮层外关闭。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-A1-01 | （无文案）浮层容器 | 浮层（420px，圆角 + 边框 + 阴影 + 背景模糊，两行网格：44px + 内容） | 关闭 | 会话内搜索打开 | `components/chat-workspace/timeline/ThreadSearchBar.vue:112` |
| G-A1-02 | — 搜索图标 | 图标（20px） | 放大镜 | 不在搜索中；搜索中该位换成旋转的加载图标 | `ThreadSearchBar.vue:118`、`:123` |
| G-A1-03 | 「搜索聊天...」 | 输入框（单行，placeholder，无边框） | 空；时间线有选中文本时预填该文本 | 打开即聚焦并全选 | `ThreadSearchBar.vue:135`；i18n `chatHeader.searchChatPlaceholder` |
| G-A1-04 | 「搜索聊天消息」 | 输入框的无障碍名 | — | 常驻 | `ThreadSearchBar.vue:134`；i18n `chatHeader.searchChatLabel` |
| G-A1-05 | 「关闭搜索」 | 图标按钮（X，32px） | 可用 | 常驻 | `ThreadSearchBar.vue:146`；i18n `chatHeader.closeSearch` |
| G-A1-06 | （无文案）结果行 | 行（36px，上边框，左右分布） | 隐藏 | 查询非空 | `ThreadSearchBar.vue:155` |
| G-A1-07 | 「上一个搜索结果」 | 图标按钮（↑，28px） | 禁用 | 结果数为 0 时禁用 | `ThreadSearchBar.vue:166`；i18n `chatHeader.previousSearchResult` |
| G-A1-08 | 「下一个搜索结果」 | 图标按钮（↓，28px） | 禁用 | 结果数为 0 时禁用 | `ThreadSearchBar.vue:178`；i18n `chatHeader.nextSearchResult` |
| G-A1-09 | 「0 个结果」 / 「{current} / {total} 个结果」 | 文字（结果行右，`aria-live=polite`） | 空字符串 | 查询非空且不在搜索中 | `ThreadSearchBar.vue:35`；i18n `chatHeader.searchNoResults`、`chatHeader.searchResults` |

元素数 9。

### A2 搜索聊天对话框（ChatSearchDialog）

条件：全局入口唤起，模态，无右上关闭按钮；最多 8 条结果，按置顶/最近分组。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-A2-01 | （无文案）对话框容器 | 模态（`max-w-xl`，无内边距，无关闭按钮） | 关闭 | 唤起后 | `components/sidebar/ChatSearchDialog.vue:186` |
| G-A2-02 | 「搜索聊天」 / 「搜索并打开已有聊天」 | 标题 + 描述（`sr-only`，屏上不可见） | — | 常驻 | `ChatSearchDialog.vue:192`；i18n `chatSearch.title`、`chatSearch.description` |
| G-A2-03 | — 搜索图标 | 图标（16px） | 放大镜 | 常驻 | `ChatSearchDialog.vue:197` |
| G-A2-04 | 「搜索聊天」 | 输入框（placeholder，无边框，高 56px） | 空 | 常驻；打开即聚焦 | `ChatSearchDialog.vue:201`；i18n `chatSearch.placeholder` |
| G-A2-05 | 「正在加载...」 | 状态行（旋转图标 + 文字，`role=status`） | 隐藏 | 当前页结果未就绪 | `ChatSearchDialog.vue:209`；i18n `common.loading` |
| G-A2-06 | 「上一页」 | 文字按钮（整行，32px） | 隐藏 | 存在上一页 | `ChatSearchDialog.vue:210`；i18n `sidebar.previousPage` |
| G-A2-07 | 「显示更多」 | 文字按钮（整行，32px） | 隐藏 | 还有下一页 | `ChatSearchDialog.vue:211`；i18n `sidebar.showMore` |
| G-A2-08 | 「置顶聊天」 / 「最近聊天」 | 分组标题（12px 灰字） | — | 该组有结果 | `ChatSearchDialog.vue:107`；i18n `chatSearch.pinnedChats`、`chatSearch.recentChats` |
| G-A2-09 | （会话标题） | 结果行主行（14px 半粗，截断） | — | 有结果 | `ChatSearchDialog.vue:242` |
| G-A2-10 | （项目显示名） | 结果行副行（12px 灰字，截断） | — | 有结果 | `ChatSearchDialog.vue:245` |
| G-A2-11 | — 行首图标 | 图标（32px 底托） | 消息图标 | 错误→红色告警圈；加载中→旋转图标；未读→实心圆点；置顶→琥珀星标；否则→消息图标 | `ChatSearchDialog.vue:237`–`:241` |
| G-A2-12 | 「⌘{n}」 / 「Ctrl+{n}」 | 快捷键提示（12px 灰字） | 序号 1–8 | 有结果 | `ChatSearchDialog.vue:248` |
| G-A2-13 | （行选中态） | 行状态（强调底 + 强调前景） | 第 1 行为选中 | 上下键或悬停移动 | `ChatSearchDialog.vue:228` |
| G-A2-14 | 「未找到聊天」 | 空态文字（居中，上下留白 32px） | 隐藏 | 无任何结果 | `ChatSearchDialog.vue:259`；i18n `chatSearch.noResults` |

元素数 14。

### A3 `@` 提及搜索弹层（文件 / 提交 / 技能）

条件：输入框可用且有工作区目录，键入 `@` 后出现；查询非空时合并文件、提交、技能三类候选。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-A3-01 | （无文案）弹层容器 | 浮层（`max-h-72`，宽 `min(35rem,100vw-2rem)`，锚在光标上方或下方） | 隐藏 | `@` 提及激活 | `components/chat-workspace/composer/tiptap/ComposerEditor.vue:1386` |
| G-A3-02 | （无文案）加载骨架 | 两条脉冲占位（32px） | — | 候选未返回 | `ComposerEditor.vue:1393` |
| G-A3-03 | 「未找到文件或目录」 / 「当前无工作区目录」 / 「文件搜索失败。」 | 空态文字 | — | 依次为：查询无命中 / 会话无 cwd / 该次搜索 RPC 失败 | `ComposerEditor.vue:333`；i18n `composer.noFileMatches`、`composer.noProjectDirectory`、`contextPanel.searchFailed` |
| G-A3-04 | 「文件」 / 「技能」 | 分组标题（12px 灰字） | — | 该组有候选 | `ComposerEditor.vue:291`；i18n `composer.groupFiles`、`composer.groupSkills` |
| G-A3-05 | （文件名 / 目录名） | 候选行主字（加粗；目录名追加 `/`） | — | 文件候选 | `ComposerEditor.vue:1424`、`:271` |
| G-A3-06 | （所在目录） | 候选行次字（12px 灰字，截断） | 无 | 文件候选且有上级目录 | `ComposerEditor.vue:1426` |
| G-A3-07 | （提交短 hash，7 位） | 候选行主字（半粗） | — | 提交候选 | `ComposerEditor.vue:262` |
| G-A3-08 | （提交标题 subject） | 候选行次字（12px 灰字，截断） | — | 提交候选 | `ComposerEditor.vue:263` |
| G-A3-09 | （作者 · 日期） | 候选行右尾小字（12px，70% 不透明） | — | 提交候选 | `ComposerEditor.vue:264`、`:1432` |
| G-A3-10 | （技能名称 + 描述） | 候选行主字 + 次字 | — | 技能候选；描述缺省时只显示名称 | `ComposerEditor.vue:268`、`:269` |
| G-A3-11 | （行选中态） | 行状态（强调底 + 强调前景） | 第 0 行 | 上下键或悬停移动 | `ComposerEditor.vue:1421` |

元素数 11。

### A4 `$` 技能提及弹层

条件：输入框可用且有 suggestion provider，键入 `$` 后出现；只列技能（同名按工作区/用户/插件优先级去重，禁用的不列）。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-A4-01 | （无文案）弹层容器 | 浮层（`max-h-72`，宽 `min(24rem,100vw-2rem)`） | 隐藏 | `$` 提及激活 | `ComposerEditor.vue:1442` |
| G-A4-02 | （无文案）加载骨架 | 两条脉冲占位 | — | 技能列表未返回 | `ComposerEditor.vue:1449` |
| G-A4-03 | 「未找到技能」 | 空态文字 | — | 无匹配技能 | `ComposerEditor.vue:380`；i18n `composer.noSkill` |
| G-A4-04 | （技能名称 + 描述） | 候选行主字 + 右尾小字 | — | 有技能；描述缺省时只显示名称 | `ComposerEditor.vue:1481`、`:1482` |
| G-A4-05 | （行选中态） | 行状态（强调底 + 强调前景） | 第 0 行 | 上下键或悬停移动 | `ComposerEditor.vue:1478` |

元素数 5。

### A5 时间线命中高亮

条件：会话内搜索有查询时，时间线正文内的命中片段被包一层行内标记。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-A5-01 | （命中片段） | 行内高亮（圆角 3px，黄色 45% 底） | — | 命中但不是当前项 | `TimelineSearchHighlightedText.vue:78`、`:84` |
| G-A5-02 | （当前命中片段） | 行内高亮（橙色 70% 底 + 1px 描边） | — | 是当前定位项 | `TimelineSearchHighlightedText.vue:83`、`:89` |

元素数 2。

区域 A 合计 41。

---

## B 命令

### B1 斜杠命令弹层

条件：输入框可用，行首键入 `/` 后出现；`new`、`login` 两个命令被显式排除，不在应用内出现。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-B1-01 | （无文案）弹层容器 | 浮层（`max-h-72`，宽 `min(24rem,100vw-2rem)`） | 隐藏 | `/` 激活 | `ComposerEditor.vue:1442` |
| G-B1-02 | （无文案）加载骨架 | 两条脉冲占位 | — | 命令/技能未返回 | `ComposerEditor.vue:1449` |
| G-B1-03 | 「未找到命令或技能」 | 空态文字 | — | 无匹配项 | `ComposerEditor.vue:367`；i18n `composer.noCommandOrSkill` |
| G-B1-04 | 「指令」 / 「技能」 | 分组标题（12px 灰字） | — | 该组有项 | `ComposerEditor.vue:290`；i18n `composer.groupCommands`、`composer.groupSkills` |
| G-B1-05 | 「/{命令名}」 + （命令描述） | 候选行（主字 + 右尾 12px 灰字，截断） | — | 命令项；无描述时只显示名称 | `ComposerEditor.vue:1481`、`:1482` |
| G-B1-06 | （技能名称） + （技能描述） | 候选行（主字 + 右尾小字） | — | 技能项；无描述时只显示名称 | `ComposerEditor.vue:1481`、`:1482` |
| G-B1-07 | （行选中态） | 行状态（强调底 + 强调前景） | 第 0 行 | 上下键或悬停移动 | `ComposerEditor.vue:1478` |
| G-B1-08 | 「/agent」「切换至 Agent 模式」 | 固定命令行 | — | 常驻候选 | `components/chat-workspace/composer/desktop-slash-commands.ts:37`；i18n `slashCommands.switchToAgent` |
| G-B1-09 | 「/goal」「切换至 Goal 模式」 | 固定命令行 | — | 常驻候选 | `desktop-slash-commands.ts:43`；i18n `slashCommands.switchToGoal` |
| G-B1-10 | 「/pet」「唤醒或收起 Pet」 | 固定命令行 | — | 常驻候选 | `desktop-slash-commands.ts:49`；i18n `slashCommands.togglePet` |
| G-B1-11 | 「/settings」「打开设置」 | 固定命令行 | — | 非独立会话窗口 | `desktop-slash-commands.ts:55`；i18n `slashCommands.openSettings` |
| G-B1-12 | 「/models」「打开模型设置」 | 固定命令行 | — | 非独立会话窗口 | `desktop-slash-commands.ts:61`；i18n `slashCommands.openModelSettings` |
| G-B1-13 | 「/mcp」「打开 MCP 设置」 | 固定命令行 | — | 非独立会话窗口 | `desktop-slash-commands.ts:67`；i18n `slashCommands.openMcpSettings` |
| G-B1-14 | 「/skills」「打开技能」 | 固定命令行 | — | 非独立会话窗口 | `desktop-slash-commands.ts:73`；i18n `slashCommands.openSkills` |
| G-B1-15 | 「/reload-skills」「重新扫描磁盘上的技能」 | 固定命令行 | — | 常驻候选 | `desktop-slash-commands.ts:79`；i18n `slashCommands.reloadSkills` |
| G-B1-16 | 「/reload-plugins」「重新扫描已安装插件」 | 固定命令行 | — | 常驻候选 | `desktop-slash-commands.ts:85`；i18n `slashCommands.reloadPlugins` |
| G-B1-17 | 「/compact」「压缩当前上下文」 | 固定命令行 | — | 常驻候选 | `desktop-slash-commands.ts:91`；i18n `slashCommands.compactContext` |
| G-B1-18 | 「/fast」「Toggle fast mode」 | 固定命令行（描述未本地化，英文原文） | — | 快速模式可用时 | `desktop-slash-commands.ts:95` |
| G-B1-19 | 「/branch」「侧边对话」 | 固定命令行 | — | 常驻候选 | `desktop-slash-commands.ts:103`；i18n `sideChat.title` |
| G-B1-20 | 「/workflow」「Browse workflows」 | 固定命令行（描述未本地化，英文原文） | — | 常驻候选 | `desktop-slash-commands.ts:107` |

元素数 20。

### B2 命令面板基元（ui/command）

条件：参考产品内只有"设置 → 添加模型"调色板使用这套基元；斜杠命令与会话搜索各自手写浮层，不复用本组。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-B2-01 | 「Command Palette」 / 「Search for a command to run...」 | 标题 + 描述（`sr-only`） | 上述英文默认值 | 调用方未覆盖；实际使用处为「添加模型」/「搜索模型...」 | `components/ui/command/CommandDialog.vue:12`；i18n `configurationSettings.addModel`、`inputArea.searchModels` |
| G-B2-02 | （无文案）面板容器 | 容器（弹出底，圆角，`h-full w-full`） | — | 面板打开 | `components/ui/command/Command.vue:80` |
| G-B2-03 | （无文案）搜索输入 | 输入框（左置放大镜图标，自动聚焦） | 空 | 面板打开 | `CommandInput.vue:31` |
| G-B2-04 | （分组标题） | 分组标题（12px 灰字） | 无 | 传入 `heading` | `CommandGroup.vue:40` |
| G-B2-05 | （命令项） | 行（可选图标 + 文字，选中态强调底） | — | 有项且匹配筛选 | `CommandItem.vue:21` |
| G-B2-06 | （快捷键） | 行尾小字（字距加大） | 无 | 传入 shortcut | `components/ui/command/CommandShortcut.vue:14` |
| G-B2-07 | （分隔线） | 分隔线（1px） | 无 | 传入时 | `components/ui/command/CommandSeparator.vue:15` |
| G-B2-08 | （空态） | 空态容器（居中，上下 24px） | 隐藏 | 有搜索词且匹配数为 0 | `CommandEmpty.vue:20` |
| G-B2-09 | （列表容器） | 滚动列表（最大高 300px） | — | 面板打开 | `CommandList.vue:16` |

元素数 9。

区域 B 合计 29。

---

## C 审批

### C1 工具权限卡（内联，占输入位上方）

条件：会话收到 `tool_call` 权限请求；卡片贴输入区上沿，不遮时间线。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-C1-01 | （无文案）卡片容器 | 卡片（边框 + 内边距 16px，背景与页面同色） | 隐藏 | 有权限请求 | `components/chat-workspace/interactions/PermissionPrompt.vue:222` |
| G-C1-02 | 「DimAgent 想要：{action}」 | 标题（16px 半粗） | — | 常驻；`{action}` 为工具动作标签（运行命令、读取文件等） | `PermissionPrompt.vue:81`、`:78`；i18n `permissionPrompt.actionRequest` |
| G-C1-03 | 「使用工具」 | 标题动作回退 | — | 工具无专用标签或来自 MCP | `PermissionPrompt.vue:63`、`approval-panels/approval-tool-display.ts:38`；i18n `permissionPrompt.useTool` |
| G-C1-04 | （无文案）成员审批上下文块 | 区块（浅底 30%，边框 60%，内边距 12px） | 隐藏 | 请求来自团队成员分支 | `PermissionPrompt.vue:231` |
| G-C1-05 | 「该成员想要：{action}」 / 「正在加载成员操作...」 / 「无法加载成员操作」 / 「当前客户端无法审查成员操作」 | 区块标题（14px 半粗） | — | 依次为：成员动作就绪 / 加载中 / 加载失败 / 客户端与服务端都无法给出原始动作 | `PermissionPrompt.vue:80`–`:74`；i18n `permissionPrompt.memberApproval.actionRequest`/`loading`/`errorTitle`/`unavailableTitle` |
| G-C1-06 | 「成员 {agentId} · 分支 {branchId}」 | 文字行（14px 灰） | 隐藏 | 成员标识存在 | `PermissionPrompt.vue:87`；i18n `permissionPrompt.memberApproval.member` |
| G-C1-07 | 「主分支 {branchId}」 | 文字行（14px 灰） | 隐藏 | 存在 leader 分支 | `PermissionPrompt.vue:96`；i18n `permissionPrompt.memberApproval.leaderBranch` |
| G-C1-08 | 「App Server 未能返回原始操作。请重试或拒绝。」 | 错误文字（14px 危险色） | 隐藏 | 成员动作加载失败 | `PermissionPrompt.vue:238`；i18n `permissionPrompt.memberApproval.error` |
| G-C1-09 | 「当前客户端与 App Server 无法显示原始操作，只能拒绝。」 | 说明文字（14px 灰） | 隐藏 | 成员动作不可审查 | `PermissionPrompt.vue:241`；i18n `permissionPrompt.memberApproval.unavailable` |
| G-C1-10 | 「审批 {approvalId}」 | 等宽文字（12px 灰，可断行） | 隐藏 | 成员动作不可审查 | `PermissionPrompt.vue:244`；i18n `permissionPrompt.memberApproval.approvalId` |
| G-C1-11 | （审批原因） | 说明文字（14px 灰） | 隐藏 | 成员动作就绪且有 reason | `PermissionPrompt.vue:250` |
| G-C1-12 | 「重试」 | 次级小按钮（`outline`） | 隐藏 | 加载失败且可重试 | `PermissionPrompt.vue:257`；i18n `permissionPrompt.memberApproval.retry` |
| G-C1-13 | （无文案）审批输入区 | 区块（见 C4，最大高 240px） | 隐藏 | 工具展示区解析成功 | `PermissionPrompt.vue:260` |
| G-C1-14 | 「需要你判断」 + （交回原因） | 提示块（浅底 50%，内边距 12px，标题 14px 半粗） | 隐藏 | 自动审批交回人工（`reviewer_uncertain`） | `PermissionPrompt.vue:274`；i18n `permissionPrompt.needsYourDecision` |
| G-C1-15 | 「自动审批没有完成，请由你决定。」 | 提示块（浅底 50%，14px 灰） | 隐藏 | 自动审批失败（`reviewer_failed`） | `PermissionPrompt.vue:283`；i18n `permissionPrompt.autoReviewFailed` |
| G-C1-16 | （提交错误） | 错误文字（14px 危险色） | 隐藏 | 选择动作失败 | `PermissionPrompt.vue:286` |
| G-C1-17 | （无文案）动作按钮组 | 按钮组（右对齐，间距 8px） | — | 常驻 | `PermissionPrompt.vue:288` |
| G-C1-18 | 「仅这次」 | 主按钮（`default`，小号，28px 高） | — | 选项含一次性允许；成员审批时 `allow_all` 也显示此文案 | `PermissionPrompt.vue:137`；i18n `permissionPrompt.allowOnce` |
| G-C1-19 | 「拒绝」 / 「不允许」 | 次按钮（`outline`，小号） | — | 普通工具为「拒绝」，沙箱场景为「不允许」 | `PermissionPrompt.vue:134`；i18n `permissionPrompt.reject`、`permissionPrompt.dontAllow` |
| G-C1-20 | 「始终允许此命令」 / 「允许此会话期间访问」 / 「始终允许文件访问」 | 次按钮（`outline`，小号） | — | `allow_all` 时按工具切换：exec/execute → 命令；read/write/edit → 会话期间；其余 → 文件 | `PermissionPrompt.vue:147`–`:151`；i18n `permissionPrompt.alwaysAllowCommand`、`permissionPrompt.allowForSession`、`permissionPrompt.alwaysAllowFile` |
| G-C1-21 | （按钮禁用态） | 按钮状态（50% 不透明） | 可用 | 提交中或整体禁用 | `PermissionPrompt.vue:293` |

元素数 21。

### C2 沙箱越权卡（内联）

条件：权限请求变体为 `sandbox`，工具被沙箱拦下后请求放宽访问范围。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-C2-01 | （无文案）卡片容器 | 卡片（边框 + 内边距 16px） | 隐藏 | 有沙箱请求 | `PermissionPrompt.vue:302` |
| G-C2-02 | 「网络访问已拦截」 / 「文件读取已拦截」 / 「文件写入已拦截」 / 「文件执行已拦截」 | 标题（16px 半粗） | — | 按资源类型与访问方式切换 | `PermissionPrompt.vue:104`、`:101`；i18n `permissionPrompt.sandboxNetworkBlocked`/`sandboxFileReadBlocked`/`sandboxFileWriteBlocked`/`sandboxFileExecuteBlocked` |
| G-C2-03 | （域名或路径） | 等宽文字（14px，可断行） | — | 常驻 | `PermissionPrompt.vue:307` |
| G-C2-04 | 「{tool} 需要访问该域名。」 / 「需要读取该路径。」 / 「需要写入该路径。」 / 「需要执行该路径。」 | 说明文字（14px 灰） | — | 按资源类型与访问方式切换 | `PermissionPrompt.vue:116`；i18n `permissionPrompt.sandboxNetworkMessage`/`sandboxReadMessage`/`sandboxWriteMessage`/`sandboxExecuteMessage` |
| G-C2-05 | （提交错误） | 错误文字 | 隐藏 | 选择失败 | `PermissionPrompt.vue:310` |
| G-C2-06 | 「不允许」 | 次按钮 | — | 常驻 | `PermissionPrompt.vue:134`；i18n `permissionPrompt.dontAllow` |
| G-C2-07 | 「仅这次」 | 主按钮 | — | 常驻 | `PermissionPrompt.vue:137`；i18n `permissionPrompt.allowOnce` |
| G-C2-08 | 「始终允许网络访问」 / 「始终允许文件访问」 | 次按钮 | — | 按资源类型切换 | `PermissionPrompt.vue:143`；i18n `permissionPrompt.alwaysAllowNetwork`、`permissionPrompt.alwaysAllowFile` |

元素数 8。

### C3 子代理模型卡（内联）

条件：权限请求变体为 `subagent_model`，子任务要换用更贵的模型。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-C3-01 | （无文案）卡片容器 | 卡片（边框 + 内边距 16px） | 隐藏 | 有子代理模型请求 | `PermissionPrompt.vue:324` |
| G-C3-02 | 「DimAgent 想要为子任务使用更贵的模型」 | 标题（16px 半粗） | — | 常驻 | `PermissionPrompt.vue:330`；i18n `permissionPrompt.subagentModel.title` |
| G-C3-03 | 「子任务 · {taskTitle}」 | 说明行（14px 灰） | 隐藏 | 请求带任务标题 | `PermissionPrompt.vue:332`；i18n `permissionPrompt.subagentModel.task` |
| G-C3-04 | — 提供方图标 + （模型显示名）+ （提供方名称） | 区块（边框 + 浅底 30%，内边距 12px）内一行（20px 图标 + 名称 + 供方名） | — | 常驻 | `PermissionPrompt.vue:333`–`:341` |
| G-C3-05 | 「价格高于当前模型」 | 标记（12px 琥珀色） | — | 常驻 | `PermissionPrompt.vue:346`；i18n `permissionPrompt.subagentModel.pricier` |
| G-C3-06 | 「拒绝时子任务将使用当前模型 {model} 继续」 | 说明文字（12px 灰） | — | 常驻 | `PermissionPrompt.vue:351`；i18n `permissionPrompt.subagentModel.fallback` |
| G-C3-07 | （提交错误） | 错误文字 | 隐藏 | 选择失败 | `PermissionPrompt.vue:354` |
| G-C3-08 | 「使用当前模型」 | 次按钮 | — | 拒绝类选项（`reject_once`/`reject_always`） | `PermissionPrompt.vue:133`、`:126`；i18n `permissionPrompt.subagentModel.reject` |
| G-C3-09 | 「本会话允许」 | 主按钮 | — | 允许类选项 | `PermissionPrompt.vue:136`；i18n `permissionPrompt.subagentModel.allow` |

元素数 9。

### C4 审批输入区（按工具类型分渲染器）

条件：工具权限卡内嵌，按工具名与入参选择渲染器（命令 / 搜索 / 文件 / 网页 / 技能 / 子代理 / 浏览器 / CUA / 等待 / 通用）；旧版纯文本入参一律走通用渲染器。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-C4-01 | （无文案）容器 | 区块（最大高 240px，可滚动，边框 + 卡片底） | — | 渲染器命中 | `approval-panels/CommandApprovalPanel.vue:18`（各渲染器同构） |
| G-C4-02 | 「输入」 | 顶部标签（11px 灰，滚动时吸附） | — | 命令 / 搜索 / 网页搜索 / 行列表 / 通用渲染器 | `CommandApprovalPanel.vue:24`；i18n `timelineTool.input` |
| G-C4-03 | 「$ {command}」 | 等宽代码行（`$` 灰色前缀，可断行） | — | 命中命令字段 | `CommandApprovalPanel.vue:26` |
| G-C4-04 | 「cwd: {cwd}」 / 「timeoutMs: {值}」 / 「runInBackground: {true/false}」 | 等宽参数行组（缩进两格，70% 不透明） | 无 | 对应字段存在 | `CommandApprovalPanel.vue:27`–`:29` |
| G-C4-05 | （文件名 / `patch.diff` / 工具名） | 顶部标签（11px 灰，截断） | 工具名 | 文件渲染器 | `FileApprovalPanel.vue:36` |
| G-C4-06 | （编辑前/后） | 双栏代码块（无行号） | — | `edit` 且有 oldText 与 newText | `FileApprovalPanel.vue:38` |
| G-C4-07 | （写入内容） | 代码块（带行号） | — | `write` 且有 content | `FileApprovalPanel.vue:45` |
| G-C4-08 | （补丁文本） | 代码块（无行号） | — | `apply_patch`/`patch` 且有 patch | `FileApprovalPanel.vue:50` |
| G-C4-09 | 「replaceAll: {值}」 / 「offset/limit/encoding: {值}」 | 等宽参数行组 | 无 | 对应字段存在 | `FileApprovalPanel.vue:56`、`:58` |
| G-C4-10 | 「pattern: {值}」 / 「query: {值}」 / 「path: {值}」 | 等宽主值行（90% 不透明） | — | 搜索/列举渲染器 | `SearchApprovalPanel.vue:33`–`:38` |
| G-C4-11 | 「path: {值}」 | 等宽次要行（70% 不透明） | 无 | 有路径且主值不是路径 | `SearchApprovalPanel.vue:34` |
| G-C4-12 | 「include/glob/outputMode/beforeContext/afterContext/context/-B/-A/-C/lineNumbers/-n/literal/limit: {值}」 | 等宽参数行组 | 无 | 对应字段存在 | `SearchApprovalPanel.vue:35` |
| G-C4-13 | 「query: {值}」 | 等宽主值行 | — | 网页搜索 | `WebApprovalPanel.vue:30` |
| G-C4-14 | 「page/pageSize: {值}」 | 等宽参数行组 | 无 | 网页搜索且带分页 | `WebApprovalPanel.vue:31`、`:44` |
| G-C4-15 | （URL） | 顶部标签（11px 灰，截断） | — | 网页抓取 | `WebApprovalPanel.vue:36` |
| G-C4-16 | 「startIndex/contextMaxCharacters/mode: {值}」 | 等宽参数行组 | 无 | 网页抓取且有附加参数 | `WebApprovalPanel.vue:39` |
| G-C4-17 | 「skill_name: {名称}」 | 顶部标签 + 等宽行 | — | 技能渲染器 | `SkillApprovalPanel.vue:11` |
| G-C4-18 | （子代理创建头行：图标 + 名称 + 任务标题 + 模型 id） | 行（14px 图标 + 三段文字） | — | 子代理创建类动作（`create`/`create_external`） | `AgentApprovalPanel.vue:30`–`:38` |
| G-C4-19 | （子代理 prompt） | 等宽文本块（可断行） | — | 有 prompt | `AgentApprovalPanel.vue:41` |
| G-C4-20 | 「effort/forkContext: {值}」 | 等宽参数行组 | 无 | 有对应字段 | `AgentApprovalPanel.vue:42` |
| G-C4-21 | 「action/taskId/agentName/taskTitle/agentType/providerId/modelId/effort/reason: {值}」 | 顶部标签 + 行列表 | — | 非创建类子代理动作 | `AgentApprovalPanel.vue:46`、`ApprovalInputLines.vue:18` |
| G-C4-22 | 「action/url/ref/value/key/direction/amount/type/timeout/expression/method/params/selector/interactiveOnly/compact/fullPage/clear: {值}」 | 顶部标签 + 行列表 | — | 浏览器渲染器 | `BrowserApprovalPanel.vue:8` |
| G-C4-23 | 「tool: {tool}」 + 全部入参行 | 顶部标签 + 行列表 | — | CUA 渲染器（工具名的 `cua-driver__` 前缀被去掉） | `CuaApprovalPanel.vue:24`–`:27` |
| G-C4-24 | 「taskId: {id}」 / 「timeoutMs: {值}」 | 顶部标签 + 行列表 | — | 等待渲染器 | `WaitApprovalPanel.vue:16` |
| G-C4-25 | （工具名 + 入参 JSON） | 顶部标签（等宽 11px）+ 等宽代码块（两空格缩进） | — | 通用渲染器；旧版纯文本入参以 `{input: 文本}` 走同一块 | `GenericApprovalPanel.vue:17`、`:21`、`ApprovalInputPanel.vue:24` |

元素数 25。

区域 C 合计 63。

---

## D 提问

### D1 提问卡（elicitation，内联）

条件：会话收到 `elicitation` 请求；多字段时逐题展示；贴输入区上沿，可收起；带截止时间时显示倒计时。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-D1-01 | （无文案）卡片容器 | 卡片（圆角 10px，边框，纵向三段：头 / 可滚动体 / 脚） | 隐藏 | 有提问请求 | `ElicitationPrompt.vue:436`、`:438` |
| G-D1-02 | （当前字段标题） | 标题（14px 半粗，截断） | — | 取字段 title → schema title → 回退 | `ElicitationPrompt.vue:451` |
| G-D1-03 | 「需要你的回答」 | 标题回退文案 | — | 字段与 schema 都无标题 | `ElicitationPrompt.vue:165`；i18n `elicitation.defaultTitle` |
| G-D1-04 | 「{当前}/{总数}」 | 进度徽标（灰底 10px 圆角小字） | — | 字段数 > 1 | `ElicitationPrompt.vue:453` |
| G-D1-05 | 「待选择」 / 「待选择 · {count} 个选项」 / 「已选：{answer}」 / 「已选 {count} 项」 | 收起态状态行（12px 灰，截断） | 隐藏 | 卡片收起时按已答情况切换 | `ElicitationPrompt.vue:171`；i18n `elicitation.awaitingAnswer`/`optionCount`/`selectedAnswer`/`selectedCount` |
| G-D1-06 | 「剩余 {m:ss}」 | 倒计时（12px 等宽数字，`role=timer`） | 无 | 请求带 `deadlineAt` 且未被作答清除 | `ElicitationPrompt.vue:246`；i18n `elicitation.timeLeft` |
| G-D1-07 | 「收起」 / 「展开」 | 文字小按钮（带下箭头，收起时旋转 180°） | 「收起」 | 常驻 | `ElicitationPrompt.vue:487`；i18n `elicitation.collapse`、`elicitation.expand` |
| G-D1-08 | （提问正文） | 多行文字（13px，保留换行） | 无 | 请求带 message | `ElicitationPrompt.vue:499` |
| G-D1-09 | （字段描述） | 说明文字（14px 灰，保留换行） | 无 | 字段带 description | `ElicitationPrompt.vue:507` |
| G-D1-10 | 「筛选选项…」 | 输入框（placeholder） | 空 | 当前字段选项数 > 8 | `ElicitationPrompt.vue:522`；i18n `elicitation.filterPlaceholder` |
| G-D1-11 | （单选项条目） | 行 + 单选圈（选中：主色边框 + 浅主色底） | 未选 | 字段为 string 且带 `oneOf` | `ElicitationPrompt.vue:527`、`:544` |
| G-D1-12 | （多选项条目） | 行 + 方形勾选框（选中显示对勾） | 未选 | 字段为 array 且带 `anyOf` | `ElicitationPrompt.vue:598`、`:603` |
| G-D1-13 | 「没有匹配的选项」 | 空态文字（14px 灰） | 隐藏 | 筛选后无匹配 | `ElicitationPrompt.vue:561`、`:616`；i18n `elicitation.noMatch` |
| G-D1-14 | 「每行输入一个值」 | 多行输入框（1 行起，最多 4 行后内部滚动） | 空 | 字段为无选项数组 | `ElicitationPrompt.vue:630`；i18n `elicitation.freeformPlaceholder` |
| G-D1-15 | 「自定义答案」 + 「输入自定义答案」 | 标签（12px 灰）+ 多行输入框 | 空 | 字段允许自定义且有选项 | `ElicitationPrompt.vue:650`、`:665`；i18n `elicitation.customAnswer`、`elicitation.customAnswerPlaceholder` |
| G-D1-16 | 「自定义选项」 + 「每行输入一个自定义值」 | 标签 + 多行输入框 | 空 | 数组字段允许自定义 | `ElicitationPrompt.vue:673`、`:687`；i18n `elicitation.customOptions`、`elicitation.customOptionsPlaceholder` |
| G-D1-17 | 「请先回答所有必填问题」 / （服务端错误） | 错误文字（14px 危险色） | 隐藏 | 必填未答时提交 / 提交失败 | `ElicitationPrompt.vue:353`；i18n `elicitation.requiredMissing` |
| G-D1-18 | 「拒绝」 | 次按钮（`outline`，小号，左置） | — | 常驻 | `ElicitationPrompt.vue:708`；i18n `elicitation.decline` |
| G-D1-19 | 「上一题」 | 次按钮（小号） | 隐藏 | 多字段且非首题 | `ElicitationPrompt.vue:718`；i18n `elicitation.previous` |
| G-D1-20 | 「下一题」 | 主按钮（小号） | 隐藏 | 多字段且非末题 | `ElicitationPrompt.vue:726`；i18n `elicitation.next` |
| G-D1-21 | 「提交」 | 主按钮（小号） | — | 单字段或已是末题 | `ElicitationPrompt.vue:734`；i18n `elicitation.submit` |
| G-D1-22 | （按钮禁用态） | 按钮状态（50% 不透明） | 可用 | 提交中；单字段且必填未满足时「提交」禁用 | `ElicitationPrompt.vue:717`、`:731` |

元素数 22。

区域 D 合计 22。

---

## E 确认

### E1 编辑运行中消息确认

条件：会话仍在运行时，提交对已发送消息的编辑。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-E1-01 | 「会话正在运行」 | 标题 | — | 打开时 | `components/chat-workspace/interactions/EditRunningMessageDialog.vue:13`；i18n `chatWorkspace.editBusyTitle` |
| G-E1-02 | 「当前会话仍在运行中。是否停止本次运行并发送编辑后的消息？」 | 描述 | — | 打开时 | `EditRunningMessageDialog.vue:15`；i18n `chatWorkspace.editBusyDescription` |
| G-E1-03 | 「取消」 | 次按钮（`alert-dialog` 取消位） | — | 常驻 | `EditRunningMessageDialog.vue:20`；i18n `common.cancel` |
| G-E1-04 | 「停止并发送」 | 主按钮 | — | 常驻 | `EditRunningMessageDialog.vue:23`；i18n `chatWorkspace.editBusyStopAndSend` |

元素数 4。

### E2 删除会话

条件：单条删除（任意时刻）或批量删除（多选后）；标题与描述随条数切换。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-E2-01 | 「删除会话」 / 「删除 {count} 个会话」 | 标题 | — | 单条 / 多条 | `session/dialogs/DeleteSessionDialog.vue:68`、`:68`；i18n `sessionDialogs.deleteSessionTitle`、`deleteSessionsTitle` |
| G-E2-02 | 「确定要永久删除“{title}”吗？删除后会话将立即从侧边栏移除，且无法恢复。」 / 「确定要永久删除以下会话吗？{names}」 | 描述 | — | 单条 / 多条；`{names}` 最多三名 + 「等 N 个」 | `DeleteSessionDialog.vue:76`、`:77`；i18n `sessionDialogs.deleteSessionDescription`、`deleteSessionsDescription`、`namesMore` |
| G-E2-03 | （失败原因） | 错误文字（14px 危险色） | 隐藏 | 删除失败且为单条 | `DeleteSessionDialog.vue:81` |
| G-E2-04 | 「取消」 | 次按钮（`outline`） | — | 常驻 | `DeleteSessionDialog.vue:85`；i18n `common.cancel` |
| G-E2-05 | 「删除」 / 「删除 ({count})」 / 「删除中...」 | 危险按钮（`destructive`） | — | 单条 / 多条 / 提交中（提交中禁用） | `DeleteSessionDialog.vue:90`–`:90`；i18n `common.delete`、`deleteMany`、`deleting` |

元素数 5。

### E3 归档会话

条件：批量归档，或归档单条仍在运行的会话（空闲单条不弹此框，直接归档）。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-E3-01 | 「归档 {count} 个会话」 | 标题 | — | 打开时 | `components/sidebar/ArchiveThreadDialog.vue:67`；i18n `sessionDialogs.archiveSessionsTitle` |
| G-E3-02 | 「确定要归档以下会话吗？{names}」 | 描述 | — | 打开时；`{names}` 最多三名 + 「等 N 个」 | `ArchiveThreadDialog.vue:70`；i18n `sessionDialogs.archiveSessionsDescription` |
| G-E3-03 | 「正在运行的会话会被中断，其队列中的后续消息也会被丢弃。」 | 警告文字（14px 警示色） | 隐藏 | 目标中有运行中的会话 | `ArchiveThreadDialog.vue:79`；i18n `sessionDialogs.archiveSessionsLoopingWarning` |
| G-E3-04 | （失败原因） | 错误文字 | 隐藏 | 归档失败 | `ArchiveThreadDialog.vue:82` |
| G-E3-05 | 「取消」 | 次按钮 | — | 常驻 | `ArchiveThreadDialog.vue:86`；i18n `common.cancel` |
| G-E3-06 | 「归档 ({count})」 / 「归档中...」 | 主按钮 | — | 常驻 / 提交中（提交中禁用） | `ArchiveThreadDialog.vue:90`、`:89`；i18n `sessionDialogs.archiveMany`、`archiving` |

元素数 6。

### E4 删除工作区会话

条件：在侧栏工作区分组上执行删除；展示该工作区下将被删除的会话条数。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-E4-01 | 「删除工作区会话」 | 标题 | — | 打开时 | `session/dialogs/DeleteProjectDialog.vue:34`；i18n `sessionDialogs.deleteProjectTitle` |
| G-E4-02 | 「将删除“{name}”下的 {count} 条会话。」 | 描述 | — | 打开时 | `DeleteProjectDialog.vue:36`；i18n `sessionDialogs.deleteProjectDescription` |
| G-E4-03 | （工作区路径） | 文字行（14px 灰） | — | 打开时 | `DeleteProjectDialog.vue:40` |
| G-E4-04 | （失败原因） | 错误文字 | 隐藏 | 删除失败 | `DeleteProjectDialog.vue:41` |
| G-E4-05 | 「取消」 | 次按钮 | — | 常驻 | `DeleteProjectDialog.vue:45`；i18n `common.cancel` |
| G-E4-06 | 「删除」 | 危险按钮 | — | 常驻 | `DeleteProjectDialog.vue:48`；i18n `common.delete` |

元素数 6。

区域 E 合计 21。

---

## F 重命名

### F1 会话重命名

条件：侧栏会话条目菜单；需 SessionTitle 方法就绪。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-F1-01 | 「重命名会话」 | 标题 | — | 打开时 | `components/sidebar/RenameThreadDialog.vue:125`；i18n `sessionDialogs.renameThreadTitle` |
| G-F1-02 | 「输入新的会话名称」 | 描述 | — | 打开时 | `RenameThreadDialog.vue:127`；i18n `sessionDialogs.renameThreadDescription` |
| G-F1-03 | 「会话名称」 | 字段标签（14px 中等） | — | 打开时 | `RenameThreadDialog.vue:133`；i18n `sessionDialogs.threadName` |
| G-F1-04 | 「输入会话名称」 | 输入框（placeholder，聚焦即全选） | 当前标题 | 打开时 | `RenameThreadDialog.vue:139`；i18n `sessionDialogs.threadNamePlaceholder` |
| G-F1-05 | 「最多使用 {max} 个字符」 / 「请输入不同的会话名称」 / 「会话名称中不能包含控制字符」 / 「会话名称不能为空」 | 校验提示（12px 灰） | 隐藏 | 依次为超 80 字符 / 与当前相同 / 含控制字符或行分隔符 / 去空白后为空 | `RenameThreadDialog.vue:168`–`:148`；i18n `sessionDialogs.renameTooLong`、`renameUnchanged`、`renameControlCharacters`、`rpcErrors.requiredString` |
| G-F1-06 | 「正在检查 SessionTitle…」 / 「SessionTitle 不可用」 | 就绪提示（12px 灰） | 隐藏 | 能力探测中 / 能力不可用 | `RenameThreadDialog.vue:160`；i18n `sessionDialogs.renameChecking`、`renameUnavailable` |
| G-F1-07 | （重命名失败原因） | 错误文字（12px 危险色，`role=alert`） | 隐藏 | 提交失败 | `RenameThreadDialog.vue:152` |
| G-F1-08 | 「取消」 | 次按钮（`outline`） | — | 常驻 | `RenameThreadDialog.vue:186`；i18n `common.cancel` |
| G-F1-09 | 「确认」 | 主按钮 | 禁用 | 校验通过且能力就绪且非提交中才可用 | `RenameThreadDialog.vue:195`；i18n `common.confirm` |

元素数 9。

### F2 工作区重命名

条件：侧栏工作区分组菜单；改的是本地项目别名。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-F2-01 | 「重命名工作区」 | 标题 | — | 打开时 | `session/dialogs/RenameProjectDialog.vue:81`；i18n `sessionDialogs.renameProjectTitle` |
| G-F2-02 | 「输入新的工作区名称」 | 描述 | — | 打开时 | `RenameProjectDialog.vue:83`；i18n `sessionDialogs.renameProjectDescription` |
| G-F2-03 | 「工作区名称」 | 字段标签 | — | 打开时 | `RenameProjectDialog.vue:89`；i18n `sessionDialogs.projectName` |
| G-F2-04 | 「输入工作区名称」 | 输入框（placeholder，聚焦即全选） | 当前别名 | 打开时 | `RenameProjectDialog.vue:95`；i18n `sessionDialogs.projectNamePlaceholder` |
| G-F2-05 | （工作区路径） | 文字行（14px 灰） | — | 打开时 | `RenameProjectDialog.vue:106` |
| G-F2-06 | 「取消」 | 次按钮 | — | 常驻 | `RenameProjectDialog.vue:110`；i18n `common.cancel` |
| G-F2-07 | 「确认」 | 主按钮 | 禁用 | 非空且与当前不同才可用 | `RenameProjectDialog.vue:118`；i18n `common.confirm` |

元素数 7。

区域 F 合计 16。

---

## G 导出与派生

### G1 导出对话

条件：会话菜单「导出对话」；先取回合大纲，再按回合勾选导出。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-G1-01 | 「导出对话」 | 标题 | — | 打开时 | `session/dialogs/ExportSessionDialog.vue:131`；i18n `sessionDialogs.exportSessionTitle` |
| G-G1-02 | 「导出「{name}」的会话对话。不包含原始 Provider 请求。」 | 描述（最多 3 行，超出截断） | — | 打开时 | `ExportSessionDialog.vue:133`；i18n `sessionDialogs.exportSessionDescription` |
| G-G1-03 | 「导出内容」 | 分组标签（14px 中等） | — | 打开时 | `ExportSessionDialog.vue:138`；i18n `sessionDialogs.exportContent` |
| G-G1-04 | 「压缩记录」 / 「工具详情」 / 「思考」 | 勾选项行（方框 + 14px 文字） | 查实现默认（首屏读代码为未勾选） | 常驻 | `ExportSessionDialog.vue:146`–`:160`；i18n `sessionDialogs.exportIncludeCompaction`、`exportIncludeToolDetails`、`exportIncludeThinking` |
| G-G1-05 | 「消息」 | 分组标签（14px 中等） | — | 打开时 | `ExportSessionDialog.vue:169`；i18n `sessionDialogs.exportMessages` |
| G-G1-06 | 「保留 {kept} / {total} 个回合」 | 计数文字（12px 灰） | 隐藏 | 大纲已载入 | `ExportSessionDialog.vue:171`；i18n `sessionDialogs.exportKeptCount` |
| G-G1-07 | 「搜索回合」 | 搜索输入框（左置放大镜，placeholder） | 空 | 大纲已载入 | `ExportSessionDialog.vue:181`；i18n `sessionDialogs.exportSearchPlaceholder` |
| G-G1-08 | 「排除 {count} 个匹配」 | 次级小按钮（`outline`） | 隐藏 | 搜索有匹配；匹配为 0 时禁用 | `ExportSessionDialog.vue:193`；i18n `sessionDialogs.exportExcludeMatches` |
| G-G1-09 | 「正在读取对话…」 / 「读取会话对话失败」 / 「该对话无法按回合拆分，因此无法过滤。导出文件仅包含对话头信息。」 / 「没有回合匹配此搜索。」 | 状态文字（居中，上下 24px） | — | 依次为：大纲加载中 / 加载失败（危险色）/ 对话不可按回合拆分 / 搜索无匹配 | `ExportSessionDialog.vue:198`–`:211`；i18n `sessionDialogs.exportOutlineLoading`、`exportOutlineFailed`、`exportNotFilterable`、`exportNoMatches` |
| G-G1-10 | （回合条目：序号 + 预览 + 回复摘要） | 勾选行（方框 + 展开箭头 + 序号 + 预览 + 「回复 · {tools} 个工具 · {chars}k」） | 勾选 | 大纲已载入 | `ExportSessionDialog.vue:219`–`:250`；i18n `sessionDialogs.exportReplySummary` |
| G-G1-11 | （回合正文） | 等宽预格式块（最大高 256px） | 隐藏 | 该回合已展开；取文过程中显示「加载中…」 | `ExportSessionDialog.vue:253`；i18n `sessionDialogs.exportTurnLoading` |
| G-G1-12 | 「全部保留」 / 「全部排除」 | 文字按钮（12px 灰） | — | 有回合 | `ExportSessionDialog.vue:264`；i18n `sessionDialogs.exportKeepAll`、`exportKeepNone` |
| G-G1-13 | 「压缩记录会单独描述被折叠的回合，排除某个回合不会将其从压缩记录中移除。」 | 说明文字（12px 灰） | 隐藏 | 勾选压缩记录且存在压缩记录 | `ExportSessionDialog.vue:284`；i18n `sessionDialogs.exportCompactionNote` |
| G-G1-14 | 「会话快照已过期并重新加载。请检查消息和排除项后再次导出。」 | 提示文字（14px，`role=status`） | 隐藏 | 快照过期并重载后 | `ExportSessionDialog.vue:288`；i18n `sessionDialogs.exportOutlineRecovered` |
| G-G1-15 | （导出失败原因） | 错误文字 | 隐藏 | 导出失败 | `ExportSessionDialog.vue:290` |
| G-G1-16 | 「取消」 | 次按钮 | — | 常驻 | `ExportSessionDialog.vue:294`；i18n `common.cancel` |
| G-G1-17 | 「导出」 / 「导出中…」 | 主按钮 | — | 常驻 / 提交中；未取到大纲时禁用 | `ExportSessionDialog.vue:302`；i18n `sessionDialogs.export`、`exporting` |

元素数 17。

### G2 Fork 会话

条件：会话已在另一个窗口打开时尝试打开。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-G2-01 | 「会话正在使用中」 | 标题 | — | 打开时 | `session/dialogs/ForkSessionDialog.vue:51`；i18n `sessionDialogs.forkSessionTitle` |
| G-G2-02 | 「“{title}”当前正在 {heldBy} 中打开。」 | 描述 | — | 打开时 | `ForkSessionDialog.vue:33`；i18n `sessionDialogs.forkSessionDescription` |
| G-G2-03 | 「Fork 将创建一个包含完整消息记录的独立副本，原会话不受影响。」 | 说明文字（14px 灰） | — | 打开时 | `ForkSessionDialog.vue:57`；i18n `sessionDialogs.forkSessionExplanation` |
| G-G2-04 | 「取消」 | 次按钮（`outline`） | — | 常驻 | `ForkSessionDialog.vue:63`；i18n `common.cancel` |
| G-G2-05 | 「Fork」 | 主按钮（提交中前置旋转图标） | — | 常驻 | `ForkSessionDialog.vue:67`；i18n `sessionDialogs.forkSessionAction` |

元素数 5。

区域 G 合计 22。

---

## H 导入与更新

### H1 添加插件（本地文件夹 / Git）

条件：设置或插件页触发；两种来源模式共用同一对话框。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-H1-01 | 「从 Git 添加插件」 | 标题 | — | 打开时 | `components/plugins/PluginInstallDialog.vue:79`；i18n `plugins.dialogTitle` |
| G-H1-02 | 「从 GitHub 仓库或 Git URL 添加。」 | 描述 | — | 打开时 | `PluginInstallDialog.vue:80`；i18n `plugins.dialogDescription` |
| G-H1-03 | 「本地文件夹」 / 「Git 仓库」 | 模式切换（两个等宽小按钮，选中为 `secondary`） | 「本地文件夹」 | 常驻 | `PluginInstallDialog.vue:92`–`:96`；i18n `plugins.native.path`、`plugins.native.git` |
| G-H1-04 | 「来源」 + 「/path/to/plugin」 | 字段标签 + 输入框 | 空 | 本地文件夹模式 | `PluginInstallDialog.vue:106`–`:109`；i18n `plugins.sourceLabel`、`plugins.native.pathPlaceholder` |
| G-H1-05 | 「选择文件夹」 | 图标按钮（文件夹图标，`outline`） | — | 本地文件夹模式 | `PluginInstallDialog.vue:114`；i18n `plugins.native.pickPath` |
| G-H1-06 | 「Git 仓库」 + 「owner/repo 或 https://github.com/owner/repo」 | 字段标签 + 输入框 | 空 | Git 模式 | `PluginInstallDialog.vue:122`–`:124`；i18n `plugins.native.git`、`plugins.native.gitPlaceholder` |
| G-H1-07 | 「Git ref」 + 「main」 | 字段标签 + 输入框 | 空 | Git 模式 | `PluginInstallDialog.vue:131`–`:134`；i18n `plugins.refLabel`、`plugins.native.refPlaceholder` |
| G-H1-08 | 「插件 id（可选）」 + 「根据 manifest 自动生成」 | 字段标签 + 输入框 | 空 | 两种模式 | `PluginInstallDialog.vue:142`–`:146`；i18n `plugins.native.idLabel`、`plugins.native.idPlaceholder` |
| G-H1-09 | 「安装后启用」 | 开关（右侧） | 开 | 常驻 | `PluginInstallDialog.vue:152`；i18n `plugins.native.enableAfterInstall` |
| G-H1-10 | 「请选择插件文件夹。」 / 「请输入 Git 仓库。」 / （服务端错误） | 错误文字（14px 危险色） | 隐藏 | 校验失败 / 安装失败 | `PluginInstallDialog.vue:156`；i18n `plugins.native.pathRequired`、`plugins.native.gitRequired` |
| G-H1-11 | 「取消」 | 次按钮（`outline`） | — | 常驻；安装中禁用 | `PluginInstallDialog.vue:161`；i18n `common.cancel` |
| G-H1-12 | 「添加插件」 | 主按钮（安装中前置旋转图标） | — | 常驻；安装中禁用 | `PluginInstallDialog.vue:165`；i18n `plugins.add` |

元素数 12。

### H2 市场安装（选择内置技能）

条件：插件市场条目点安装；先取条目详情。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-H2-01 | 「安装 {name}」 | 标题 | — | 打开时 | `components/plugins/PluginMarketplaceInstallDialog.vue:132`；i18n `plugins.marketplaceInstallTitle` |
| G-H2-02 | 「选择要启用的内置技能。默认全选。」 | 描述 | — | 打开时 | `PluginMarketplaceInstallDialog.vue:133`；i18n `plugins.marketplaceInstallDescription` |
| G-H2-03 | 「正在加载...」 | 加载行（旋转图标 + 14px 灰字，居中） | 隐藏 | 条目详情加载中 | `PluginMarketplaceInstallDialog.vue:138`；i18n `common.loading` |
| G-H2-04 | （错误消息 + 「重试」） | 错误块（告警图标 + 14px 灰字 + 次级按钮） | 隐藏 | 详情加载失败 | `PluginMarketplaceInstallDialog.vue:149`–`:178`；i18n `common.retry` |
| G-H2-05 | 「技能」 | 分组标签（12px 中等灰） | — | 详情已载入 | `PluginMarketplaceInstallDialog.vue:153`；i18n `plugins.marketplaceSkills` |
| G-H2-06 | （技能名称 + 描述） | 勾选行（方框 + 名称 + 12px 描述） | 全选 | 插件含技能 | `PluginMarketplaceInstallDialog.vue:160`–`:199` |
| G-H2-07 | 「此插件不包含技能。」 | 空态文字（14px 灰） | 隐藏 | 插件无技能 | `PluginMarketplaceInstallDialog.vue:174`；i18n `plugins.marketplaceNoSkills` |
| G-H2-08 | 「取消」 | 次按钮（`outline`） | — | 常驻 | `PluginMarketplaceInstallDialog.vue:178`；i18n `common.cancel` |
| G-H2-09 | 「添加插件」 | 主按钮（安装中前置旋转图标） | 禁用 | 详情未载入、加载中或安装中禁用 | `PluginMarketplaceInstallDialog.vue:185`；i18n `plugins.add` |

元素数 9。

### H3 替换现有插件确认

条件：市场安装会替换同目录名的既有插件（来自 git、手动安装或另一市场条目）。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-H3-01 | 「替换现有插件？」 | 标题 | — | 打开时 | `components/plugins/MarketOverwriteDialog.vue:40`；i18n `plugins.marketOverwriteTitle` |
| G-H3-02 | 「"{name}" 将替换一个来自其他来源（git、手动安装或另一个市场条目）的同名插件。此操作无法撤销。」 | 描述 | — | 打开时 | `MarketOverwriteDialog.vue:42`；i18n `plugins.marketOverwriteDescription` |
| G-H3-03 | 「取消」 | 次按钮（`outline`） | — | 常驻；工作禁用 | `MarketOverwriteDialog.vue:47`；i18n `common.cancel` |
| G-H3-04 | 「替换」 | 危险按钮（`destructive`，工作中前置旋转图标） | — | 常驻；工作禁用 | `MarketOverwriteDialog.vue:56`；i18n `plugins.marketOverwriteConfirm` |

元素数 4。

### H4 插件可更新清单

条件：市场已装插件存在更高版本；由商店页「更新」按钮或「我的插件」更新横幅打开。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-H4-01 | 「可用更新」 | 标题 | — | 打开时 | `components/plugins/PluginUpdatesDialog.vue:65`；i18n `plugins.updatesTitle` |
| G-H4-02 | 「所有插件均为最新版本」 | 空态文字（14px 灰，居中） | 隐藏 | 无待更新项 | `PluginUpdatesDialog.vue:74`；i18n `plugins.noUpdates` |
| G-H4-03 | （插件占位图标 + 名称） | 行首图标（40px 底托 + 方块图标）+ 名称（14px 中等，截断） | — | 有可更新项 | `PluginUpdatesDialog.vue:87`–`:95` |
| G-H4-04 | 「v{旧} → v{新}」 / 「v{新}」 | 版本行（12px 灰） | — | 有记录版本 / 旧版本未知 | `PluginUpdatesDialog.vue:101` |
| G-H4-05 | （下载进度条 + 「已下载 {size}…」 / 「{已下载} / {总量}」） | 进度条（1px 高）+ 12px 灰字 | 隐藏 | 该插件安装中 | `PluginUpdatesDialog.vue:39`–`:99`、`:10`–`:13`；i18n `plugins.downloadedBytes` |
| G-H4-06 | 「更新」 | 次级小按钮（刷新图标，安装中旋转） | — | 有可更新项；安装中或全部更新时禁用 | `PluginUpdatesDialog.vue:113`；i18n `plugins.update` |
| G-H4-07 | 「全部更新（{count}）」 | 主按钮（刷新图标，更新中旋转） | — | 有可更新项；更新中禁用 | `PluginUpdatesDialog.vue:126`；i18n `plugins.updateAll` |

元素数 7。

### H5 技能可更新清单

条件：市场已装技能存在更高版本。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-H5-01 | 「可用更新」 | 标题 | — | 打开时 | `components/skills/SkillUpdatesDialog.vue:50`；i18n `skillsMarketplace.updatesTitle` |
| G-H5-02 | 「所有技能均为最新版本」 | 空态文字 | 隐藏 | 无待更新项 | `SkillUpdatesDialog.vue:59`；i18n `skillsMarketplace.noUpdates` |
| G-H5-03 | （技能图标 + 名称） | 行首图标（40px 技能图标）+ 名称 | — | 有可更新项 | `SkillUpdatesDialog.vue:71`–`:77` |
| G-H5-04 | 「v{旧} → v{新}」 / 「v{新}」 | 版本行（12px 灰） | — | 有记录版本 / 旧版本未知 | `SkillUpdatesDialog.vue:80` |
| G-H5-05 | 「更新」 | 次级小按钮（刷新图标，安装中旋转） | — | 有可更新项；安装中或全部更新时禁用 | `SkillUpdatesDialog.vue:89`；i18n `skillsMarketplace.update` |
| G-H5-06 | 「全部更新（{count}）」 | 主按钮 | — | 有可更新项；更新中禁用 | `SkillUpdatesDialog.vue:102`；i18n `skillsMarketplace.updateAll` |

元素数 6。

### H6 导入 Chrome Cookie

条件：设置 → 浏览器数据 → 「导入…」；先扫描 Chrome 配置，再选配置预览计数。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-H6-01 | 「导入 Chrome Cookie」 | 标题 | — | 打开时 | `BrowserDataSettings.vue:273`；i18n `browserData.importTitle` |
| G-H6-02 | 「此操作会替换内置浏览器中的全部非分区 Cookie，并重新加载已打开的页面。每个 Cookie 都会单独尝试导入，因此个别 Cookie 可能失败；部分网站可能需要重新登录。」 | 描述 | — | 可导入数量不为 0 | `BrowserDataSettings.vue:274`；i18n `browserData.replaceWarning` |
| G-H6-03 | 「正在扫描 Chrome 配置…」 | 加载行（旋转图标 + 14px 灰字，最小高 96px） | 隐藏 | 扫描中 | `BrowserDataSettings.vue:279`；i18n `browserData.scanning` |
| G-H6-04 | 「Chrome 配置」 | 分组标签（12px 中等灰） | — | 扫描完成 | `BrowserDataSettings.vue:283`；i18n `browserData.profileLabel` |
| G-H6-05 | （配置名 + 能力标签） | 单选项行（自绘单选圈 + 名称 + 12px 能力标签） | 首个可选配置 | 存在配置 | `BrowserDataSettings.vue:285`–`:300` |
| G-H6-06 | 「未找到 Google Chrome 配置。」 | 空态文字 | 隐藏 | 无配置且无错误 | `BrowserDataSettings.vue:302`；i18n `browserData.noProfiles` |
| G-H6-07 | 「正在读取所选配置…」 | 加载行（旋转图标 + 14px 灰字） | 隐藏 | 读取所选配置中 | `BrowserDataSettings.vue:304`；i18n `browserData.previewing` |
| G-H6-08 | 「可导入 {importable} · 会话 {session} · 已过期 {expired} · 分区 {partitioned}」 | 计数块（边框 + 浅底，12px） | 隐藏 | 已取到预览 | `BrowserDataSettings.vue:313`；i18n `browserData.counts` |
| G-H6-09 | 「已导入 {imported} · 失败 {failed} · 已跳过…」 + 「部分 Cookie 未能导入，部分网站可能需要重新登录。」 | 结果块（对勾或告警图标 + 14px 文字 + 12px 警示色说明） | 隐藏 | 导入结束 | `BrowserDataSettings.vue:317`–`:328`；i18n `browserData.importResultCounts`、`importPartial` |
| G-H6-10 | （错误消息 + 诊断行 + 「重试」） | 错误块（危险色 + 等宽诊断行 + 次级按钮） | 隐藏 | 导入或读取失败 | `BrowserDataSettings.vue:331`–`:347` |
| G-H6-11 | 「取消」 / 「关闭」 | 次按钮（`outline`） | — | 未结束时「取消」，已导入后「关闭」 | `BrowserDataSettings.vue:355`；i18n `common.cancel`、`common.close` |
| G-H6-12 | 「导入」 / 「正在导入 Cookie…」 / （阶段文案：正在关闭浏览器页面… / 正在更新浏览器数据… / 正在验证浏览器数据… / 正在恢复原有登录状态…） | 主按钮（导入中前置旋转图标） | 禁用 | 未取到预览、导入中、读取中或扫描中禁用 | `BrowserDataSettings.vue:358`；i18n `browserData.confirmImport`、`importing`、`phaseQuiescing`/`phaseApplying`/`phaseVerifying`/`phaseRollingBack` |

元素数 12。

### H7 清理浏览器数据确认

条件：设置 → 浏览器数据 → 「清理数据…」。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-H7-01 | 「清理浏览器数据」 | 标题 | — | 打开时 | `BrowserDataSettings.vue:379`；i18n `browserData.clearTitle` |
| G-H7-02 | 「此操作只会清除内置浏览器中的登录状态、Cookie、本地存储、IndexedDB、Service Worker 和缓存。浏览器页面会关闭并以空白页重新打开。」 | 描述 | — | 打开时 | `BrowserDataSettings.vue:380`；i18n `browserData.clearDescription` |
| G-H7-03 | 「内置浏览器数据已清理。」 | 成功行（对勾 + 14px 文字） | 隐藏 | 清理成功 | `BrowserDataSettings.vue:383`；i18n `browserData.clearSuccess` |
| G-H7-04 | （错误消息 + 诊断行） | 错误块（危险色 + 等宽诊断行） | 隐藏 | 清理失败 | `BrowserDataSettings.vue:387` |
| G-H7-05 | 「取消」 / 「关闭」 | 次按钮 | — | 未完成时「取消」，完成后「关闭」 | `BrowserDataSettings.vue:393`；i18n `common.cancel`、`common.close` |
| G-H7-06 | 「清理浏览器数据」 / 「正在清理浏览器数据…」 | 危险按钮（清理中前置旋转图标） | — | 未完成时显示；清理中禁用 | `BrowserDataSettings.vue:396`、`:395`；i18n `browserData.confirmClear`、`clearing` |

元素数 6。

区域 H 合计 56。

---

## I 附件与选择

### I1 图片附件预览

条件：点开输入区或时间线里的图片附件；全屏黑底灯箱，多图可横向翻页。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-I1-01 | （无文案）灯箱容器 | 全屏模态（黑色 92% 底，无边框无圆角；macOS 覆盖全屏，Windows 让出标题栏） | 关闭 | 打开时 | `components/chat-workspace/attachments/AttachmentImagePreviewDialog.vue:283` |
| G-I1-02 | 「关闭」 | 圆形图标按钮（40px，右上角，X） | — | 常驻 | `AttachmentImagePreviewDialog.vue:300`；i18n `common.close` |
| G-I1-03 | （图片） | 图片（`object-contain`，不可拖动，横向吸附翻页） | — | 图片已就绪 | `AttachmentImagePreviewDialog.vue:351` |
| G-I1-04 | （加载中） | 旋转图标（24px，白色 70%） | 隐藏 | 图片加载中 | `AttachmentImagePreviewDialog.vue:354` |
| G-I1-05 | 「无法加载图片预览」 / 「不支持此图片格式」 + 「重试」 | 错误文字（14px 灰）+ 下划线文字按钮 | 隐藏 | 加载失败 / 格式不支持 | `AttachmentImagePreviewDialog.vue:358`–`:362`；i18n `attachmentPreview.loadFailed`、`attachmentPreview.unsupported`、`common.retry` |
| G-I1-06 | 「复制图片」 / 「复制全部 ({count})」 | 右键菜单项（图标 + 文字） | 隐藏 | 图片上右键；「复制全部」仅在多图时出现 | `AttachmentImagePreviewDialog.vue:337`–`:348`；i18n `attachmentPreview.copyImage`、`attachmentPreview.copyAll` |
| G-I1-07 | 「已复制到剪贴板」 / （复制失败） | 状态胶囊（底部居中，成功绿 / 失败红，`role=status`） | 隐藏 | 复制动作结束后短暂显示 | `AttachmentImagePreviewDialog.vue:372`–`:374`；i18n `attachmentPreview.copySuccess` |
| G-I1-08 | （文件名） / 「图片预览」 | 元数据条（底部居中胶囊，12px 白字，截断） | 「图片预览」 | 常驻；文件名为空时回退 | `AttachmentImagePreviewDialog.vue:385`；i18n `attachmentPreview.title` |
| G-I1-09 | 「{当前} / {总数}」 | 计数文字（元数据条内，12px） | 隐藏 | 多于一张 | `AttachmentImagePreviewDialog.vue:390` |
| G-I1-10 | 「上一张」 | 圆形图标按钮（40px，左侧居中，←） | 禁用 | 多于一张；首张禁用 | `AttachmentImagePreviewDialog.vue:401`；i18n `carousel.previousSlide` |
| G-I1-11 | 「下一张」 | 圆形图标按钮（40px，右侧居中，→） | 禁用 | 多于一张；末张禁用 | `AttachmentImagePreviewDialog.vue:411`；i18n `carousel.nextSlide` |

元素数 11。

### I2 文本附件预览

条件：点开文本类附件；可编辑来源（草稿附件）显示保存，其余只读显示复制。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-I2-01 | （文件名） | 标题（14px 中等，截断）+ 同名 `sr-only` 描述 | — | 打开时 | `TextAttachmentDialog.vue:75` |
| G-I2-02 | 「正在加载...」 | 加载行（旋转图标 + 14px 灰字，`role=status`） | 隐藏 | 内容加载中 | `TextAttachmentDialog.vue:91`；i18n `common.loading` |
| G-I2-03 | （错误文本） | 错误块（居中，14px 危险色，`role=alert`） | 隐藏 | 加载失败 | `TextAttachmentDialog.vue:96` |
| G-I2-04 | （正文） | 等宽文本域（最小高 320px，只读或可编辑，无拼写检查） | — | 内容已就绪 | `TextAttachmentDialog.vue:107` |
| G-I2-05 | 「取消」 | 次按钮（`outline`） | — | 可编辑模式 | `TextAttachmentDialog.vue:115`；i18n `common.cancel` |
| G-I2-06 | 「保存」 | 主按钮（保存中前置旋转图标） | 禁用 | 可编辑模式；未改动或为空时禁用 | `TextAttachmentDialog.vue:123`；i18n `common.save` |
| G-I2-07 | 「复制」 | 次按钮（`outline`，左置复制图标） | — | 只读模式；加载中或出错时禁用 | `TextAttachmentDialog.vue:134`；i18n `common.copy` |
| G-I2-08 | 「关闭」 | 主按钮 | — | 只读模式 | `TextAttachmentDialog.vue:137`；i18n `common.close` |

元素数 8。

### I3 选中文本气泡（标注问题）

条件：在可标注内容里选中文本后浮出；胶囊态两键，展开态为评论输入。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-I3-01 | （无文案）气泡容器 | 浮层（锚在选区底边中点上方 12px；胶囊态全圆角，展开态大圆角） | 隐藏 | 有文本选中 | `components/selection/SelectionPopover.vue:96` |
| G-I3-02 | 「加入对话」 | 胶囊按钮（引用图标 + 文字） | — | 胶囊态 | `SelectionPopover.vue:115`；i18n `markPopover.addToChat` |
| G-I3-03 | 「标注问题」 | 胶囊按钮（闪光图标 + 文字） | — | 胶囊态 | `SelectionPopover.vue:119`；i18n `markPopover.markIssue` |
| G-I3-04 | 「添加可选评论…」 | 多行输入框（1 行起，最多 4 行后内部滚动，展开即聚焦） | 空 | 展开态 | `SelectionPopover.vue:132`；i18n `markPopover.commentPlaceholder` |
| G-I3-05 | 「提交」 | 次级小按钮（引用图标 + 文字） | — | 展开态；发送中禁用 | `SelectionPopover.vue:144`；i18n `markPopover.submit` |
| G-I3-06 | 「发送失败。请重试，或将其添加到输入框。」 | 错误文字（12px 危险色） | 隐藏 | 发送失败 | `SelectionPopover.vue:151`；i18n `markPopover.sendFailed` |

元素数 6。

### I4 宠物活动

条件：从 Pet 浮动窗或侧栏账户菜单打开；列出有活动状态的会话。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-I4-01 | 「活动」 | 标题 + 同名 `sr-only` 描述 | — | 打开时 | `components/pet/PetActivityDialog.vue:165`；i18n `petActivity.title` |
| G-I4-02 | （会话标题 + 「状态 · 相对时间」） | 行（状态图标 16px + 标题 14px 中等 + 12px 灰副行；悬停/聚焦强调底） | — | 有活动会话 | `PetActivityDialog.vue:180`–`:199` |
| G-I4-03 | 「没有活跃对话」 | 空态文字（居中，上下 24px） | 隐藏 | 无活动会话 | `PetActivityDialog.vue:209`；i18n `petActivity.empty` |
| G-I4-04 | 「当前 display server 不支持显示 Pet。」 | 说明块（浅底，12px 灰） | 隐藏 | 主窗口模式 | `PetActivityDialog.vue:217`；i18n `petActivity.platformNote` |
| G-I4-05 | 「打开设置」 | 次按钮（`outline`） | — | 常驻 | `PetActivityDialog.vue:222`；i18n `petActivity.openSettings` |
| G-I4-06 | 「打开 DimAgent」 | 主按钮 | — | 常驻 | `PetActivityDialog.vue:225`；i18n `petActivity.openApp` |

元素数 6。

区域 I 合计 31。

---

## J Git 与定时

### J1 提交改动

条件：Git 面板的提交入口。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-J1-01 | 「提交改动」 | 标题 | — | 打开时 | `components/chat-workspace/git/CommitDialog.vue:105`；i18n `gitDialogs.commitChanges` |
| G-J1-02 | 「{count} 个文件变更 +{a} -{r}」 | 描述（增量为绿色，减量为红色） | — | 打开时 | `CommitDialog.vue:107`；i18n `gitDialogs.filesChanged` |
| G-J1-03 | 「输入提交信息...」 | 多行输入框（最小高 80px，不可缩放） | 空 | 常驻；提交中禁用 | `CommitDialog.vue:116`；i18n `gitDialogs.enterCommitMessage` |
| G-J1-04 | 「同时推送到远程」 | 勾选项行（方框 + 14px 文字） | 跟随默认推送偏好 | 常驻；提交中禁用 | `CommitDialog.vue:131`；i18n `gitDialogs.pushToRemote` |
| G-J1-05 | （提交失败原因） | 错误文字（14px 危险色） | 隐藏 | 提交失败 | `CommitDialog.vue:135` |
| G-J1-06 | 「取消」 | 次按钮（`outline`） | — | 常驻；提交中禁用 | `CommitDialog.vue:142`；i18n `common.cancel` |
| G-J1-07 | 「提交」 | 主按钮（提交中前置旋转图标） | 禁用 | 提交信息为空或提交中禁用 | `CommitDialog.vue:146`；i18n `gitDialogs.commit` |

元素数 7。

### J2 创建新分支

条件：Git 分支入口；打开时加载本地分支列表。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-J2-01 | 「创建新分支」 | 标题 | — | 打开时 | `components/chat-workspace/git/NewBranchDialog.vue:148`；i18n `gitDialogs.createNewBranch` |
| G-J2-02 | 「分支名称」 + 「feature/my-branch」 | 字段标签 + 输入框 | 分支前缀偏好 | 常驻；提交中禁用 | `NewBranchDialog.vue:153`–`:158`；i18n `gitDialogs.branchName` |
| G-J2-03 | 「分支名不能包含空格」 / 「分支名包含无效字符」 / 「分支名格式无效」 / 「分支名不能包含连续的点」 | 校验提示（12px 危险色） | 隐藏 | 依次为含空格 / 非法字符 / 以 `-` 开头或以 `.`、`.lock` 结尾 / 含 `..` | `NewBranchDialog.vue:57`；i18n `gitDialogs.branchNameNoSpaces`、`branchNameInvalidCharacters`、`branchNameInvalidFormat`、`branchNameNoConsecutiveDots` |
| G-J2-04 | 「基于」 + （「加载中...」 / 「选择基础分支」 / 分支名） | 字段标签 + 下拉（placeholder 随状态） | 默认基础分支偏好 | 常驻；提交或加载中禁用 | `NewBranchDialog.vue:167`–`:180`；i18n `gitDialogs.baseBranch`、`loading`、`selectBaseBranch` |
| G-J2-05 | （加载或创建失败原因） | 错误文字（14px 危险色） | 隐藏 | 分支加载失败 / 创建失败 | `NewBranchDialog.vue:87`；i18n `gitDialogs.loadBranchesFailed`、`createBranchFailed` |
| G-J2-06 | 「取消」 | 次按钮（`outline`） | — | 常驻；提交中禁用 | `NewBranchDialog.vue:191`；i18n `common.cancel` |
| G-J2-07 | 「创建」 | 主按钮（提交中前置旋转图标） | 禁用 | 无名称、校验失败或无 git 目录时禁用 | `NewBranchDialog.vue:195`；i18n `common.create` |

元素数 7。

### J3 切换分支（有未提交文件时）

条件：切换分支会带过未提交改动时弹出；工作区干净时直接切换。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-J3-01 | 「切换分支」 | 标题 | — | 打开时 | `components/chat-workspace/git/BranchSwitchDialog.vue:43`；i18n `gitDialogs.switchBranchTitle` |
| G-J3-02 | 「即将从 {current} 切换到 {target}。当前工作区有 {count} 个未提交文件。」 | 描述（可断行） | — | 有挂起请求 | `BranchSwitchDialog.vue:45`；i18n `gitDialogs.switchBranchDescription` |
| G-J3-03 | 「{count} 个文件变更 +{a} -{r}」 | 统计条（12px，上边框） | — | 有挂起请求 | `BranchSwitchDialog.vue:55`；i18n `gitDialogs.filesChanged` |
| G-J3-04 | 「已暂存」 / 「未暂存」 | 行内徽标（描边小胶囊，10px） | — | 有挂起请求 | `BranchSwitchDialog.vue:67`；i18n `gitDialogs.staged`、`unstaged` |
| G-J3-05 | （文件路径） | 行内文字（12px，截断） | — | 有挂起请求 | `BranchSwitchDialog.vue:71` |
| G-J3-06 | 「+{added}」 / 「-{removed}」 | 行内增删计数（绿色 / 红色） | — | 有挂起请求 | `BranchSwitchDialog.vue:76` |
| G-J3-07 | 「输入提交信息后再切换...」 | 多行输入框（最小高 72px，不可缩放） | 空 | 常驻；动作中禁用 | `BranchSwitchDialog.vue:85`；i18n `gitDialogs.commitBeforeSwitchPlaceholder` |
| G-J3-08 | 「会先暂存上面列出的未暂存文件，创建一次提交，然后切换分支。」 | 说明文字（12px 灰） | — | 常驻 | `BranchSwitchDialog.vue:92`；i18n `gitDialogs.commitBeforeSwitchHint` |
| G-J3-09 | （失败原因） | 错误文字（14px 危险色） | 隐藏 | 提交或切换失败 | `BranchSwitchDialog.vue:94` |
| G-J3-10 | 「取消」 | 次按钮（`outline`） | — | 常驻；动作中禁用 | `BranchSwitchDialog.vue:101`；i18n `common.cancel` |
| G-J3-11 | 「直接切换」 | 次按钮（`outline`，切换中前置旋转图标） | — | 常驻；动作中禁用 | `BranchSwitchDialog.vue:105`；i18n `gitDialogs.switchWithoutCommit` |
| G-J3-12 | 「提交后切换」 | 主按钮（提交中前置旋转图标） | 禁用 | 提交信息为空或动作中禁用 | `BranchSwitchDialog.vue:109`；i18n `gitDialogs.commitAndSwitch` |

元素数 12。

### J4 撤销改动 / 重新应用改动

条件：每轮文件改动的撤销与再应用确认；文件清单只读，动作是全量。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-J4-01 | 「撤销改动」 / 「重新应用改动」 | 标题 | — | 撤销模式 / 再应用模式 | `components/chat-workspace/git/UndoChangesDialog.vue:58`；i18n `fileChanges.undoDialogTitle`、`reapplyDialogTitle` |
| G-J4-02 | 「撤销本轮的文件改动？」 / 「重新应用本轮的文件改动？」 | 提问文字（14px） | — | 按模式切换 | `UndoChangesDialog.vue:61`；i18n `fileChanges.undoDialogQuestion`、`reapplyDialogQuestion` |
| G-J4-03 | 「M」 / 「A」 / 「D」 | 行内状态字母（等宽半粗；改=警示色，增=成功色，删=危险色） | — | 有文件类型；无类型时为空 | `UndoChangesDialog.vue:119` |
| G-J4-04 | （文件路径） | 行内文字（12px，截断） | — | 有文件 | `UndoChangesDialog.vue:120` |
| G-J4-05 | 「+{n} / -{n}」 / 「binary」 / 「（删除）」 | 行内统计（等宽 12px 灰） | — | 有增删数 / 二进制 / 撤销新增文件 | `UndoChangesDialog.vue:123`、`:66`–`:73`；i18n `fileChanges.binary`、`willDelete` |
| G-J4-06 | 「+{count} 更多」 | 省略提示（12px 灰） | 隐藏 | 文件数超出展示上限 | `UndoChangesDialog.vue:128`；i18n `artifacts.moreCount` |
| G-J4-07 | 「取消」 | 次按钮（`outline`） | — | 常驻；进行中禁用 | `UndoChangesDialog.vue:131`；i18n `common.cancel` |
| G-J4-08 | 「撤销改动」 / 「重新应用」 | 主按钮（撤销为危险样式，再应用为默认样式；进行中前置旋转图标） | — | 按模式切换；进行中禁用 | `UndoChangesDialog.vue:134`；i18n `fileChanges.undoConfirm`、`reapplyConfirm` |

元素数 8。

### J5 创建定时任务

条件：定时任务页「创建」；打开时重置表单并重新检查技能可用性。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-J5-01 | 「创建定时任务」 | 标题 | — | 打开时 | `ScheduledCreateDialog.vue:107`；i18n `scheduled.create.title` |
| G-J5-02 | 「为工作区配置一个按计划重复执行的提示词。」 | 描述 | — | 打开时 | `ScheduledCreateDialog.vue:108`；i18n `scheduled.create.description` |
| G-J5-03 | 「定时任务名称」 | 输入框（placeholder） | 空 | 常驻 | `ScheduledCreateDialog.vue:111`；i18n `scheduled.details.namePlaceholder` |
| G-J5-04 | 「描述每次运行时希望 DimAgent 完成的工作...」 | 富文本提示词编辑器（`$skill:` 引用为原子块，最小高 7rem） | 空 | 常驻 | `ScheduledCreateDialog.vue:112`–`:139`；i18n `scheduled.prompt.placeholder` |
| G-J5-05 | 「技能」 + 「添加技能」 | 标签（14px 灰）+ 描边小按钮（+ 图标） | — | 常驻 | `components/scheduled/ScheduledSkillField.vue:134`、`:141`；i18n `scheduled.skills.title`、`scheduled.skills.add` |
| G-J5-06 | 「搜索技能」 | 下拉内搜索输入（左置放大镜） | 空 | 技能下拉打开 | `ScheduledSkillField.vue:158`；i18n `scheduled.skills.search` |
| G-J5-07 | （技能名称 + 描述） | 下拉选项行（方块图标 + 名称 + 最多两行描述） | — | 有可用技能 | `ScheduledSkillField.vue:167`–`:180` |
| G-J5-08 | 「没有匹配的技能。」 / 「该项目暂无可用技能。」 | 下拉空态文字（居中） | 隐藏 | 有技能但无匹配 / 无可用技能 | `ScheduledSkillField.vue:188`；i18n `scheduled.skills.noResults`、`scheduled.skills.empty` |
| G-J5-09 | （技能名 + 「未安装」/「已禁用」 + 移除） | 引用胶囊（圆角 999px；异常态危险色边框 + 告警图标；移除按钮 `aria-label`「移除 {name}」） | — | 提示词里存在 `$skill:` 引用 | `ScheduledSkillField.vue:208`–`:215`；i18n `scheduled.skills.missing`、`scheduled.skills.disabled`、`scheduled.skills.remove` |
| G-J5-10 | 「详情」 | 分组标题（14px 中等灰） | — | 常驻 | `components/scheduled/ScheduledJobForm.vue:158`；i18n `scheduled.details.title` |
| G-J5-11 | 「项目」 + （项目名） | 行（左标签 + 右侧紧凑下拉，49px 行高） | 默认表单的项目 | 常驻 | `ScheduledJobForm.vue:161`；i18n `scheduled.details.project` |
| G-J5-12 | 「模型」 + （模型 · 提供方） | 行 + 下拉（选项为「模型 · 提供方」） | 默认表单的模型 | 常驻 | `ScheduledJobForm.vue:176`；i18n `scheduled.details.model` |
| G-J5-13 | 「推理强度」 + （强度 / 「默认」） | 行 + 下拉（关闭/极低/低/中/高/极高/最高） | 「默认」 | 所选模型支持推理强度；否则禁用 | `ScheduledJobForm.vue:194`；i18n `scheduled.details.reasoning`、`scheduled.details.defaultReasoning` |
| G-J5-14 | 「运行方式」 + 「每次新任务」 / 「同一任务续跑」 | 行 + 下拉 | 默认表单的运行方式 | 常驻 | `ScheduledJobForm.vue:219`、`:228`；i18n `scheduled.details.runsIn`、`newTask`、`sameTask` |
| G-J5-15 | 「频率」 | 分组标题（14px 中等灰） | — | 常驻 | `ScheduledJobForm.vue:237`；i18n `scheduled.frequency.title` |
| G-J5-16 | 「重复」 + 「每小时」/「每天」/「工作日」/「每周」/「自定义」 | 行 + 下拉 | 默认表单的重复方式 | 常驻；不可编辑计划时该行改为只读的 RRULE 文本 | `ScheduledJobForm.vue:244`、`:250`；i18n `scheduled.frequency.repeat`、`everyHour`、`daily`、`weekdays`、`weekly`、`custom` |
| G-J5-17 | 「重复频率」 + 「每分钟」/「每小时」/「每天」/「每周」/「每月」/「每年」 | 行 + 下拉 | 自定义频率 | 重复方式为「自定义」 | `ScheduledJobForm.vue:269`；i18n `scheduled.frequency.repeats`、`freq.*` |
| G-J5-18 | 「间隔」 + 「{n} {单位}」 | 行 + 下拉（数值 1–上限，单位随频率） | 自定义间隔 | 重复方式为「自定义」，且允许自定义间隔或频率为每分钟 | `ScheduledJobForm.vue:297`、`:301`；i18n `scheduled.frequency.every`、`unit.*` |
| G-J5-19 | 「第几分钟」 + （0–59） | 行 + 下拉 | 自定义分钟 | 自定义且频率为每小时 | `ScheduledJobForm.vue:321`；i18n `scheduled.frequency.atMinute` |
| G-J5-20 | 「月份」 + （月名） | 行 + 下拉（1–12 月） | 自定义月份 | 自定义且频率为每年 | `ScheduledJobForm.vue:343`；i18n `scheduled.frequency.inMonth` |
| G-J5-21 | 「日期」 + （1–N） | 行 + 下拉（按月份天数） | 自定义日期 | 自定义且频率为每月/每年 | `ScheduledJobForm.vue:366`；i18n `scheduled.frequency.onDay` |
| G-J5-22 | 「星期」 + （周一…周日） | 行 + 下拉 | 默认表单的星期 | 重复方式为每周，或自定义且频率为每周 | `ScheduledJobForm.vue:389`；i18n `scheduled.frequency.on`、`day.*` |
| G-J5-23 | 「时间」 + （时间选择器） | 行 + 时间选择（`aria-label` 同为「时间」） | 默认表单的时间 | 非每小时、非每分钟的计划 | `ScheduledJobForm.vue:410`；i18n `scheduled.frequency.at` |
| G-J5-24 | 「取消」 | 幽灵按钮 | — | 常驻 | `ScheduledCreateDialog.vue:139`；i18n `common.cancel` |
| G-J5-25 | 「创建」 | 主按钮 | 禁用 | 名称、提示词、项目、模型齐备且引用的技能全部可用才启用；保存中禁用 | `ScheduledCreateDialog.vue:146`；i18n `scheduled.create.action` |

元素数 25。

区域 J 合计 59。

---

## K 账户与首次运行

### K1 企业邮箱登录

条件：首次运行欢迎页的「登录企业账号」，或账户页的企业登录入口。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-K1-01 | 「登录企业账号」 | 标题 | — | 打开时 | `components/account/AccountEmailLoginDialog.vue:60`；i18n `accountSettings.enterpriseLogin` |
| G-K1-02 | 「请输入您的企业邮箱账号」 | 描述 | — | 打开时 | `AccountEmailLoginDialog.vue:62`；i18n `accountSettings.enterpriseLoginDescription` |
| G-K1-03 | 「企业邮箱」 | 字段标签（14px 中等） | — | 打开时 | `AccountEmailLoginDialog.vue:69`；i18n `accountSettings.enterpriseEmailLabel` |
| G-K1-04 | 「name@company.com」 | 邮箱输入框（type=email，自动填充） | 空 | 常驻；登录中禁用 | `AccountEmailLoginDialog.vue:73`–`:81` |
| G-K1-05 | 「请输入有效的企业邮箱。」 / （服务端错误） | 错误文字（12px 危险色，可断行） | 隐藏 | 本地校验失败 / 登录失败 | `AccountEmailLoginDialog.vue:41`；i18n `accountSettings.invalidEnterpriseEmail` |
| G-K1-06 | 「登录企业账号」 / 「正在登录…」 | 主按钮（登录中前置旋转图标） | — | 常驻；登录中禁用 | `AccountEmailLoginDialog.vue:60`–`:93`；i18n `accountSettings.enterpriseLogin`、`accountSettings.signingIn` |

元素数 6。

### K2 微信扫码登录

条件：设置 → 工具 → 微信「连接微信」。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-K2-01 | 「连接微信」 | 标题 | — | 打开时 | `components/plugins/WeChatLoginDialog.vue:86`；i18n `toolsSettings.wechatLoginTitle` |
| G-K2-02 | 「使用「微信」扫描二维码完成授权。」 | 描述 | — | 打开时 | `WeChatLoginDialog.vue:87`；i18n `toolsSettings.wechatLoginDescription` |
| G-K2-03 | （无文案）起始态 | 旋转图标（在 208px 方框内，居中） | — | 会话启动中 | `WeChatLoginDialog.vue:99` |
| G-K2-04 | （二维码 / 「二维码渲染失败，请重试。」） | 图片（208px，圆角边框）/ 失败文字（危险色） | — | 等待扫码或已扫码；无法栅格化时为失败文字 | `WeChatLoginDialog.vue:110`、`:113`；i18n `toolsSettings.qrRenderFailed` |
| G-K2-05 | 「等待扫码…（剩余 {countdown}）」 / 「已扫码，请在微信中确认…（剩余 {countdown}）」 | 状态文字（12px；已扫码为微信绿且中等字重） | — | 等待扫码 / 已扫码 | `WeChatLoginDialog.vue:113`、`:123`；i18n `toolsSettings.wechatWaiting`、`wechatScanned` |
| G-K2-06 | 「微信已连接」 | 成功态（对勾 28px + 14px 中等绿字） | 隐藏 | 连接成功（1 秒后自动关闭） | `WeChatLoginDialog.vue:121`；i18n `toolsSettings.wechatConnected` |
| G-K2-07 | 「二维码已过期」 + 「重试」 | 说明文字（14px 灰）+ 小按钮 | 隐藏 | 二维码过期 | `WeChatLoginDialog.vue:124`；i18n `toolsSettings.wechatExpired`、`wechatRetry` |
| G-K2-08 | 「连接失败：{errorMessage}」 + 「重试」 | 错误文字（14px 危险色）+ 小按钮 | 隐藏 | 连接失败 | `WeChatLoginDialog.vue:129`；i18n `toolsSettings.wechatFailed` |
| G-K2-09 | 「取消」 | 次按钮（`outline`） | — | 常驻 | `WeChatLoginDialog.vue:135`；i18n `common.cancel` |

元素数 9。

### K3 飞书扫码安装

条件：设置 → 工具 → 飞书「二维码安装」。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-K3-01 | 「用飞书扫码安装」 | 标题 | — | 打开时 | `components/plugins/FeishuQrDialog.vue:115`；i18n `toolsSettings.feishuQrDialogTitle` |
| G-K3-02 | 「用飞书 App 扫码，在飞书里完成应用创建与授权。」 | 描述 | — | 打开时 | `FeishuQrDialog.vue:117`；i18n `toolsSettings.feishuQrDialogDescription` |
| G-K3-03 | （无文案）起始态 | 旋转图标（208px 方框内） | — | 注册流程启动中 | `FeishuQrDialog.vue:121` |
| G-K3-04 | （二维码 / 「二维码渲染失败，请重试。」） | 图片（208px）/ 失败文字 | — | 等待扫码；无法栅格化时为失败文字 | `FeishuQrDialog.vue:143`、`:136`；i18n `toolsSettings.qrRenderFailed` |
| G-K3-05 | 「等待扫码…（剩余 {countdown}）」 | 状态文字（12px 灰） | — | 等待扫码 | `FeishuQrDialog.vue:151`；i18n `toolsSettings.feishuQrWaiting` |
| G-K3-06 | 「已完成，正在保存…」 | 成功文字（14px 中等，绿色） | 隐藏 | 扫码完成 | `FeishuQrDialog.vue:159`；i18n `toolsSettings.feishuQrCompleted` |
| G-K3-07 | 「二维码已过期」 + 「重新生成」 | 说明文字 + 小按钮 | 隐藏 | 二维码过期 | `FeishuQrDialog.vue:164`–`:163`；i18n `toolsSettings.feishuQrExpired`、`feishuQrRegenerate` |
| G-K3-08 | 「安装失败：{errorMessage}」 + 「重试」 | 错误文字 + 小按钮 | 隐藏 | 安装失败 | `FeishuQrDialog.vue:173`–`:173`；i18n `toolsSettings.feishuQrFailed`、`feishuQrRetry` |
| G-K3-09 | 「取消」 | 次按钮 | — | 常驻 | `FeishuQrDialog.vue:183`；i18n `common.cancel` |

元素数 9。

### K4 侧栏账户入口与退出确认

条件：侧栏底部常驻；未登录与已登录两态。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-K4-01 | （无文案）账户按钮 | 按钮（30px 高，圆角，悬停强调底） | — | 常驻 | `AccountEntry.vue:211` |
| G-K4-02 | （头像/首字母/用户图标/旋转图标） | 行首圆形标识（18px；未登录显示用户图标，登录中显示旋转图标，已登录显示首字母） | 用户图标 | 常驻 | `AccountEntry.vue:218`–`:226` |
| G-K4-03 | （账号错误红点） | 状态圆点（8px 危险色，右上角） | 隐藏 | 账户错误 | `AccountEntry.vue:221` |
| G-K4-04 | （显示名 / 邮箱 / 「Dim 账户」 / 「登录 Dim」 / 「正在登录…」） | 主行文字（14px，截断） | 「Dim 账户」 | 按登录态与显示名有无切换 | `AccountEntry.vue:227`–`:87`、`:230`；i18n `sidebar.account.defaultName`、`signIn`、`signingIn`、`common.loading` |
| G-K4-05 | （首字母 + 主行） | 菜单首项（身份行，18px 圆标 + 14px 半粗；在账户页时高亮） | — | 菜单打开 | `AccountEntry.vue:241`；i18n 同上 |
| G-K4-06 | 「用量」 + 「剩余 {pct}%」 | 菜单项（图标 + 文字 + 右侧数值；加载时右侧为旋转图标） | — | 已登录 | `AccountEntry.vue:264`；i18n `usageSettings.title`、`sidebar.account.usageRemaining` |
| G-K4-07 | 「显示 Pet」 / 「收起 Pet」 | 菜单项（图标 + 文字） | 依 Pet 当前显隐 | 已登录；Pet 未就绪、切换中或浮动不可用时禁用（未就绪时文字为「正在加载...」） | `AccountEntry.vue:281`–`:283`；i18n `sidebar.account.showPet`、`hidePet` |
| G-K4-08 | （Pet 不可用说明） | 菜单内说明文字（12px 灰，缩进对齐文字列） | 隐藏 | Pet 浮动不可用 | `AccountEntry.vue:291` |
| G-K4-09 | 「邀请朋友」 | 菜单项（图标 + 文字） | — | 已登录且可邀请 | `AccountEntry.vue:306`；i18n `sidebar.account.inviteFriend` |
| G-K4-10 | 「正在加载...」 | 菜单项（旋转图标 + 文字，禁用） | 隐藏 | 邀请信息加载中 | `AccountEntry.vue:315`；i18n `common.loading` |
| G-K4-11 | 「设置」 + （快捷键） | 菜单项（图标 + 文字 + 右侧快捷键；设置页内高亮） | — | 已登录 | `AccountEntry.vue:330`；i18n `sidebar.settings` |
| G-K4-12 | 「退出登录」 | 菜单项（图标 + 文字） | — | 已登录 | `AccountEntry.vue:344`；i18n `sidebar.account.logOut` |
| G-K4-13 | 「登录 Dim」 | 菜单项（图标 + 文字） | — | 未登录 | `AccountEntry.vue:76`；i18n `sidebar.account.signIn` |
| G-K4-14 | 「设置」 | 图标按钮（30px，账户按钮右侧；`aria-label` 与 `title` 同为「设置」） | — | 常驻 | `AccountEntry.vue:362`；i18n `sidebar.settings` |
| G-K4-15 | 「退出 Dim？」 | 确认框标题 | — | 点「退出登录」 | `AccountEntry.vue:379`；i18n `sidebar.account.logOutConfirmationTitle` |
| G-K4-16 | 「退出后，需要重新登录才能使用 Dim 账户。」 | 确认框描述 | — | 同上 | `AccountEntry.vue:381`；i18n `sidebar.account.logOutConfirmationDescription` |
| G-K4-17 | 「取消」 | 确认框取消按钮 | — | 同上 | `AccountEntry.vue:386`；i18n `common.cancel` |
| G-K4-18 | 「退出登录」 | 确认框危险按钮 | — | 同上；退出中禁用 | `AccountEntry.vue:388`；i18n `sidebar.account.logOut` |

元素数 18。

### K5 首次运行欢迎

条件：首次运行、无会话、无可用聊天模型、未登录且未被跳过；账户需要重新授权时也复用本卡。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-K5-01 | （无文案）卡片容器 | 模态（460px，圆角 16px，无关闭按钮，顶部 192px 渐变插画区） | 关闭 | 满足首次运行条件 | `AccountLoginPrompt.vue:118` |
| G-K5-02 | 「欢迎使用 DimAgent」 / 「登录」 | 标题（24px 半粗） | — | 首次运行 / 重新授权 | `AccountLoginPrompt.vue:127`；i18n `onboarding.setup.title`、`accountSettings.login` |
| G-K5-03 | 「登录后即可使用 Dim 提供的模型」 / （服务端错误消息） | 描述（14px，行高 24px） | — | 首次运行 / 重新授权 | `AccountLoginPrompt.vue:130`；i18n `onboarding.setup.signInDesc` |
| G-K5-04 | 「登录 Dim 账号」 / 「登录」 | 主按钮（大号） | — | 首次运行 / 重新授权 | `AccountLoginPrompt.vue:142`；i18n `onboarding.setup.signInLabel`、`accountSettings.login` |
| G-K5-05 | 「登录企业账号」 | 链接样式按钮（居中） | — | 常驻；登录中禁用 | `AccountLoginPrompt.vue:152`；i18n `accountSettings.enterpriseLogin` |
| G-K5-06 | 「跳过」 / 「关闭」 | 幽灵按钮（居中） | — | 首次运行 / 重新授权 | `AccountLoginPrompt.vue:159`；i18n `onboarding.skip`、`common.close` |
| G-K5-07 | 「首次启动时，DimAgent 会在后台下载并校验插件所需的 Node.js 运行环境。……在“设置 → 运行环境”中重试。」 | 脚注（12px，上方分隔线） | 隐藏 | 首次运行（非重新授权） | `AccountLoginPrompt.vue:167`；i18n `onboarding.setup.runtimeHint` |

元素数 7。

### K6 更新提示与安装确认

条件：更新检查返回新版本后，侧栏头部出现更新按钮；点击已下载的更新弹安装确认。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-K6-01 | 「更新」 / 「{n}%」 / 「正在准备…」 / 「重启安装」 / 「重试更新」 | 按钮（带文案标签，位于侧栏头部） | 隐藏 | 有新版本（文字随下载/准备/已下载/失败切换） | `components/sidebar/SidebarHeader.vue:58`–`:59`、`:127`；i18n `sidebar.update`、`aboutSettings.preparingUpdate`、`restartAndInstall`、`retryUpdate` |
| G-K6-02 | 「有新版本可用 · v{版本} · {错误消息}」 | 按钮 tooltip（状态 + 版本 + 消息，缺项省略） | — | 悬停更新按钮 | `SidebarHeader.vue:68`–`:71`；i18n `aboutSettings.updateAvailable`、`downloadingUpdate`、`updateDownloaded` |
| G-K6-03 | 「安装更新」 | 确认框标题 | — | 打开安装确认 | `app/App.vue:321`；i18n `aboutSettings.installUpdateTitle` |
| G-K6-04 | 「DimAgent 会先停止正在运行的任务并保存会话，然后重启安装更新。」 | 确认框描述 | — | 同上 | `app/App.vue:323`；i18n `aboutSettings.installUpdateDescription` |
| G-K6-05 | 「取消」 | 确认框取消按钮 | — | 同上；准备中禁用 | `app/App.vue:328`；i18n `common.cancel` |
| G-K6-06 | 「重启安装」 / 「正在准备…」 | 确认框主按钮 | — | 同上；准备中禁用 | `app/App.vue:336`；i18n `aboutSettings.restartAndInstall`、`preparingUpdate` |

元素数 6。

### K7 Appshots 启用引导与权限请求

条件：首次用双 Command 触发快照时弹启用引导；系统权限缺失时弹权限请求。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| G-K7-01 | （无文案）启用引导图标 | 图标块（88×77，圆角 24px，反色底 + ImagePlus 36px） | — | 启用引导打开 | `session/dialogs/AppshotsDialogsHost.vue:67` |
| G-K7-02 | 「启用 Appshots」 | 标题（18px，居中） | — | 同上 | `AppshotsDialogsHost.vue:70`；i18n `appshotsSettings.enableTitle` |
| G-K7-03 | 「Appshots 会将当前窗口附加到消息，让 DimAgent 看见你正在处理的内容。」 | 描述（14px，行高 20px，居中，最大宽 310px） | — | 同上 | `AppshotsDialogsHost.vue:71`；i18n `appshotsSettings.enableDescription` |
| G-K7-04 | 「取消」 | 幽灵按钮 | — | 同上 | `AppshotsDialogsHost.vue:74`；i18n `common.cancel` |
| G-K7-05 | 「启用」 | 主按钮 | — | 同上；启用中禁用 | `AppshotsDialogsHost.vue:75`；i18n `appshotsSettings.enableAction` |
| G-K7-06 | （无文案）权限请求图标 | 图标块（56px，圆角 16px，浅底 + Camera 28px） | — | 权限请求打开 | `AppshotsDialogsHost.vue:83` |
| G-K7-07 | 「允许输入监控」 / 「允许屏幕录制」 | 标题（18px，居中） | — | 按缺失权限切换 | `AppshotsDialogsHost.vue:84`；i18n `appshotsSettings.inputPermissionTitle`、`screenPermissionTitle` |
| G-K7-08 | 「打开系统设置并启用 DimAgent；若列表中没有它，可将 DimAgent 图标拖入列表。」 | 描述（14px，行高 20px） | — | 同上；宿主给出更具体消息时用该消息 | `AppshotsDialogsHost.vue:85`；i18n `appshotsSettings.permissionDescription` |
| G-K7-09 | 「取消」 | 幽灵按钮 | — | 同上 | `AppshotsDialogsHost.vue:88`；i18n `common.cancel` |
| G-K7-10 | 「打开系统设置」 | 主按钮 | — | 同上；打开中禁用 | `AppshotsDialogsHost.vue:89`；i18n `appshotsSettings.openSettings` |

元素数 10。

区域 K 合计 65。

---

## 计数汇总

| 区域 | 分组 | 元素数 |
|---|---|---|
| A 搜索 | A1 9 / A2 14 / A3 11 / A4 5 / A5 2 | 41 |
| B 命令 | B1 20 / B2 9 | 29 |
| C 审批 | C1 21 / C2 8 / C3 9 / C4 25 | 63 |
| D 提问 | D1 22 | 22 |
| E 确认 | E1 4 / E2 5 / E3 6 / E4 6 | 21 |
| F 重命名 | F1 9 / F2 7 | 16 |
| G 导出与派生 | G1 17 / G2 5 | 22 |
| H 导入与更新 | H1 12 / H2 9 / H3 4 / H4 7 / H5 6 / H6 12 / H7 6 | 56 |
| I 附件与选择 | I1 11 / I2 8 / I3 6 / I4 6 | 31 |
| J Git 与定时 | J1 7 / J2 7 / J3 12 / J4 8 / J5 25 | 59 |
| K 账户与首次运行 | K1 6 / K2 9 / K3 9 / K4 18 / K5 7 / K6 6 / K7 10 | 65 |
| **合计** | 43 分组 | **425** |

计数口径：一行 = 一个元素；同一元素的多状态记在「默认 / 条件」里。按区域汇总见 §2。

## 未收工项

1. **证据列全缺**：全部面板只有源码定位，没有截图裁片。需跑参考产品后按屏补；截图轮由组织侧统一安排，补齐前该屏不进入行为需求阶段（§9.16 七）。在此之前本清单的用途只有两条：按元素核对覆盖率、按文案原文校对实现。
2. **元素数已定，不压**（2026-11-17 组织裁决）：425 保留，口径"一个可指认的控件或一行信息"。同构行组不合并成一个元素——那正是过去漏文案的地方。别再按 120–250 的旧预估回看这份清单。
3. **归属边界**见 §3：设置类弹窗归设置域、浏览器权限静默判定不登记、`TimelineSearchDomText` 作为运行时包裹不单列。
