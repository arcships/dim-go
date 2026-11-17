# 主窗口清单（模板 ∪ 运行时 ∪ 截图 联表）

范围：主窗口五区（标题栏 / 左栏 / 时间线 / 输入区 / 状态条）＋空态（NewThread）＋运行时出现的覆盖层与独立页。本文件只做元素对账，不写行为需求、不写验收。

## 0. 取证与口径

### 0.1 材料

| 类 | 位置 |
|---|---|
| 模板 | 参考产品源码；仓库根记 `<ref>`，定位写作 `<ref>/<相对路径>:<行号>`，相对 `packages/desktop/packages/renderer/src` |
| 运行时（空/面板/路由） | `docs/domains/requirements/runtime/main/{empty,new-session,right-panel,search,scheduled,skills}.json` |
| 运行时（会话态） | `docs/domains/requirements/runtime/main/{session,session-approval,session-after,session-done}.json` |
| 运行时（多工具并行） | `docs/domains/requirements/runtime/session/{tools-1..5,tools-rich,tools-done,tools-final}.json` |
| 运行时（本轮补采·主窗口） | `docs/domains/requirements/runtime/main/`（`C1`–`C45`，见 §0.2b） |
| 运行时（本轮补采·右栏面板） | `docs/domains/requirements/runtime/panels/`（`P1`–`P23`，见 §0.2b） |
| 截图 | 与 JSON 同目录同名 `.png`（DPR=2，1030×725 CSS 视口） |
| 节点口径 | 每项 = `{chain, tag, role, text, label, icon, x, y, w, h}`，单位 CSS px |
| 文案 | 参考产品 `zh-CN` 语言包（`<ref>/i18n/zh-CN.ts`）；`{name}` 为插值占位 |

### 0.2 状态码

| 码 | 文件 | 屏上内容 |
|---|---|---|
| E | `empty` | 左栏展开 + 空会话（「开始构建」） |
| N | `new-session` | 与 E 逐节点相同（`items` 完全相同，PNG 同 MD5），下文合并记 `E/N` |
| R | `right-panel` | 左栏收起 + 空会话 + 右栏「文件」面板打开 |
| Q | `search` | R 的底色（左栏收起、右栏打开）+「搜索聊天」弹窗（采集只含弹窗层） |
| S | `scheduled` | 定时任务页（左栏收起）＋「搜索聊天」弹窗 |
| K | `skills` | 技能/插件页（左栏收起）＋「搜索聊天」弹窗 |
| A | `session` ≡ `session-approval` | 会话进行中并停在审批卡上（两份 `items` 完全相同，PNG 不同：审批卡外观） |
| P | `session-after` | 审批放行后继续跑（输入区恢复，发送键变停止） |
| D | `session-done` | 该轮完成（助手输出 + 轮末动作） |
| T | `tools-rich` | 多工具并行轮次（读/查找/grep/执行/网搜/写/编辑/抓取/ask_user） |

`T` 代表 `tools-1`、`tools-2`、`tools-3`、`tools-4`、`tools-5`、`tools-done`、`tools-final`、`tools-rich` 八个采集：节点集相同，只有计时文案与滚动位置不同（见 §3.6 备注）。

### 0.2b 追加状态码（本轮补采，CDP 驱动）

下表状态一律由 CDP（`Runtime.evaluate` 派发 `pointerdown/mousedown/pointerup/mouseup/click`、`Input.dispatchMouseEvent`、`Input.dispatchKeyEvent`、`Input.dispatchDragEvent`）在本机参考产品上真实触发后导出，一个状态一份 JSON + 同名 PNG：`C*` 在 `docs/domains/requirements/runtime/main/`，`P*` 在 `docs/domains/requirements/runtime/panels/`。

| 码 | 文件 | 触发方式 | 屏上内容 |
|---|---|---|---|
| C1 | `session-left-collapsed-right-file` | 会话中收起左栏 + 右栏打开文件 | 会话态 + 左栏收起 + 右栏文件页签 |
| C2 | `menu-chat-more` | 点标题栏 `⋯` | 置顶 / 重命名 / 在新窗口中打开 / 导出 / 归档 / 三条调试复制项 |
| C3 | `menu-chat-more-pinned` | 已置顶时点 `⋯` | 同上，首项变「取消置顶」 |
| C4 | `menu-open-with` | 点「更多打开方式」 | 应用列表 + 在 Finder 中显示 + 用系统默认方式打开 |
| C5 | `menu-permission` | 点状态条权限按钮 | 完全访问权限 / 默认权限 |
| C6 | `dialog-full-access` | 权限菜单选「完全访问权限」 | 确认开启完全放开？对话框（勾选框 + 取消/确认） |
| C7 | `popover-reasoning` | 点输入区「最高」 | 思考强度滑杆浮层 |
| C8 | `popover-model` | 点模型按钮 | 模型浮层（模型/子代理/外部 agent 页签、搜索、最近使用、列表） |
| C9 | `popover-skills` | 点输入区技能按钮 | 搜索技能 + 技能列表 |
| C10 | `menu-mode` | 点输入区模式按钮 | Agent / Goal（Goal 灰显带不可用说明） |
| C11 | `menu-view` | 点列表头视图按钮 | 排序（最近活跃/最近创建）+ 视图（按工作区分组/平铺/折叠全部） |
| C12 | `sidebar-flat-view` | 视图菜单选「平铺」 | 日期分组 + 会话行（无工作区分组） |
| C13 | `menu-session-row-context` | 会话行右键 | 置顶 / 重命名 / 在新窗口中打开 / 导出 / 归档 / 删除 |
| C14 | `sidebar-pinned` | 置顶一条会话后 | 「已置顶」组 + 置顶会话行（带钉住图标） |
| C15 | `sidebar-pinned-workspace` | 置顶一个工作区后 | 已置顶组内出现工作区行 |
| C16 | `menu-workspace-item` | 工作区行悬停 → `⋯` | 置顶工作区 / 在文件管理器中显示 / 重命名 / 移除 |
| C17 | `menu-pinned-workspace-item` | 置顶工作区行悬停 → `⋯` | 首项变「取消置顶工作区」 |
| C18 | `hover-session-row` | 悬停会话行 | 钉住按钮 + 归档按钮 + hover 卡（标题/所属工作区） |
| C19 | `hover-search-entry` | 悬停左栏「搜索」行 | 行尾 `⌘G` 键帽 |
| C20 | `mod-meta` | 按住 ⌘ 并悬停页签 | 页签显示 `⌘2` 快捷键徽标 |
| C21 | `hover-rightpanel-tab` | 悬停右栏页签 | 页签关闭按钮（`aria-label=关闭`） |
| C22 | `menu-rightpanel-add-tab` | 会话中右栏 `+` | 侧边对话 / 终端 |
| C23 | `timeline-scrolled-up` | 时间线滚离底部 | 悬浮「回到最新」按钮 |
| C24 | `timeline-search-empty` | 会话中 `⌘F` | 消息内搜索条（输入框 + 关闭搜索） |
| C25 | `timeline-search-noresult` | 搜索条输入无命中词 | `0 个结果` |
| C26 | `timeline-search-results` | 搜索条输入命中词 | 上一个/下一个结果按钮 + `{current} / {total} 个结果` |
| C27 | `timeline-compression` | 滚到压缩分隔行并展开工具行 | `上下文已压缩` 分隔行 +「无法加载摘要。+ 重试」+ 工具详情表（输出/说明/日志） |
| C28 | `timeline-inline-editor` | 悬停用户消息 → 点铅笔 | 行内编辑器（取消 ✕ / 发送 ↑） |
| C29 | `menu-usermsg-context` | 用户消息右键 | 从此处开始侧边对话 / 复制 / 复制整个消息 |
| C30 | `menu-assistant-context` | 助手消息右键 | 复制 / 复制整个消息 |
| C31 | `sidebar-drop-hint` | 把文件夹拖到整栏（`Input.dispatchDragEvent`） | 松开以添加工作区 / 仅限文件夹 · 同时新建会话 |
| C32 | `menu-account` | 点左栏账户行 | 身份项 / 用量（剩余 61%）/ 显示 Pet / 邀请朋友 / 设置 / 退出登录 |
| C33 | `dialog-logout-confirm` | 账户菜单选「退出登录」 | 退出 Dim？对话框（描述 + 取消/退出登录） |
| C34 | `popover-usage` | 点状态条「~2%」 | 本轮上下文 / 会话消耗 / 缓存命中 / 套餐用量（账户 ↗）/ 明细行 |
| C35 | `select-workspace-empty` | 空态点工作区选择器 | 搜索工作区… / 内置工作区 / 带路径的工作区 / 添加新工作区 |
| C36 | `select-branch-empty` | 空态点分支选择器 | 搜索分支… / 分支项（当前分支带勾） |
| C37 | `composer-slash-menu` | 输入框输入 `/` | 指令菜单（/agent /goal /pet /settings /models /mcp /skills /reload-skills） |
| C38 | `session-streaming` | 发一条会触发工具审批的消息 | 运行中 + 审批卡出现瞬间 + 宿主通知横幅 |
| C39 | `session-approval-timeout` | 审批卡停留时 | 审批卡（含 `timeoutMs: 60 s`）+ 拒绝/仅次/始终允许此命令 |
| C40 | `session-running-approved` | 审批放行后 | 输入区恢复、发送键变停止、`已经工作了 {duration}` |
| C41 | `session-running-queue` | 运行中再发一条 | 第二轮消息直接进入时间线，出现 `等待` 汇总行 |
| C42 | `window-narrow` | `window.resizeTo(760,720)` | 窄窗替换布局：主区让位给右栏，输入区浮动，头部出现「返回」 |
| C43 | `window-dark-session` | 外观切深色 + 打开会话 | 暗色会话态（左栏 + 时间线 + 右栏改动页） |
| C44 | `window-dark-empty` | 外观切深色 + 空态 | 暗色空态 |
| C45 | `menu-workspace-with-sessions` | 工作区行 `⋯`（该组有会话） | 在文件管理器中显示 / 归档会话 / 重命名 |
| P1 | `panel-review-git` | 工作区是 git 仓库时打开「Review」 | 未提交改动下拉 + 刷新 + 已暂存(1)/未暂存(2) + 底部合计 |
| P2 | `menu-review-row` | 改动行右键 | 暂存 / 还原 / 预览 / 用外部工具打开 / 复制路径 |
| P3 | `panel-review-preview-file` | 改动行右键选「预览」 | 在右栏新页签打开该文件内容 |
| P4 | `panel-full-width` | 点右栏「展开面板」 | 右栏占满窗口的替换布局（页签栏置顶） |
| P5 | `menu-rightpanel-add-tab-fullwidth` | 全宽时点 `+` | 浏览器 / 终端 |
| P6 | `panel-review-state` | 全宽下切到 Review 页签 | 全宽改动面板（已暂存/未暂存 + 合计） |
| P7 | `panel-review-all-staged` | 点「全部暂存」进行中 | 按钮转加载态（转圈 + 置灰） |
| P8 | `panel-review-all-staged-done` | 全部暂存完成 | 只剩「已暂存 (3)」 |
| P9 | `panel-review-clean` | 工作区无改动 | 空态：没有未提交的改动 / 工作区是干净的 |
| P10 | `menu-review-scope` | 点「未提交改动」下拉 | 未提交改动 / 当前分支改动 / Last Run 改动（各带说明） |
| P11 | `panel-review-lastrun-empty` | 选「Last Run 改动」且无记录 | 空态：没有 Last Run 改动 |
| P12 | `panel-review-branch-diff` | 选「当前分支改动」且与基线一致 | 空态：当前分支没有改动 |
| P13 | `panel-files-root` | 打开「文件」页签 | 目录树根（目录折叠 + 根下文件） |
| P14 | `panel-files-bulk-expanded` | 展开一个含 250 文件的目录 | 文件列表逐行（含 +0/−0） |
| P15 | `panel-files-search-empty` | 筛选文件输入无命中词 | 无匹配文件 / 没有找到包含“…”的文件 |
| P16 | `panel-files-search-limit` | 筛选词命中 250 个文件 | `仅显示前 200 个匹配项` + 高亮结果 |
| P17 | `panel-terminal` | 右栏 `+` → 终端 | 面板内终端（cwd = 工作区，`git:(main)`） |
| P18 | `menu-terminal-context` | 终端内右键 | 复制选中内容 / 粘贴 / 清屏 / 新建终端 / 关闭进程 |
| P19 | `hover-tab-close` | 悬停页签 | 页签上的关闭 ✕ |
| P20 | `panel-terminal-closed` | 关闭终端页签 | 回到改动页；分支改动范围下出现 253 条未暂存 |
| P21 | `panel-review-branch-untracked253` | 当前分支改动 + 大量未跟踪文件 | `未暂存 (253)` 长列表 |
| P22 | `panel-browser-empty` | 右栏 `+` → 浏览器 | 浏览器工具栏 + 空态「预览你的成果」 |
| P23 | `menu-rightpanel-add-tab-empty` | 空会话下右栏 `+` | 浏览器 / 终端 |
| P24 | `hover-rightpanel-tab` | 悬停右栏页签（与 `C21` 同一状态，另存一份在 `panels/`） | 页签关闭 ✕ |
| P25 | `panel-side-chat-empty` | 会话中右栏 `+` → 侧边对话 | 侧边对话空态（「就当前 thread 提问…」+ 底部「问一个旁支问题...」输入框） |

另有 7 份同族补充状态（`sidebar-open`、`hover-user-message`、`hover-usermsg-actions`、`session-running-waiting`、`session-running-third-turn`、`timeline-tool-detail`、`window-dark-session-sidebar`），与上表某个主码指同一元素集，只作交叉证据，元素归属仍记主码。

实测结论一条：运行中再投递的消息**不进入排队面板**，而是直接排成下一轮并开跑（见 `C41`），因此 `IN-35`–`IN-42` 仍判「需数据」，需要 App Server 侧真的产生排队态。

### 0.3 列义

- **名称（文案原文）**：`t('key')` 的中文原文；`{x}` 为插值占位；无文案记 `（无文案）`。
- **控件**：按钮 / 图标按钮 / 菜单项 / 输入框 / 文本 / 容器 / 状态位 / 浮层。
- **图标**：模板里实际渲染的图标（lucide 组件名或 `ThemeIcon:<name>`）；无则 `—`。
- **条件**：祖先条件 ∧ 本节点 `v-if`/`v-else` 分支条件。无条件的常显节点写 `常显`。
- **在状态**：该元素在哪些采集状态里出现（E/N/R/S/K/Q/A/P/D/T）。不带括注 = 屏上直接可见；`（悬停可见）` = 节点在 DOM 里但只在悬停时显形（如用户消息动作行）；`（不可见）` = 占位或 `opacity-0`。`—（原因）` = 十态均未出现，原因见 §0.5。
- **位置**：运行时包围盒 `x,y w×h`，前缀为状态码；`≈` 表示 JSON 未采集该节点（无文案或纯 svg），坐标按截图量取。
- **定位**：模板出处；同一文案多处渲染时列多处。

### 0.4 采集局限（判「未命中」前必读）

1. **无文案且非 svg 的图标按钮会被采集跳过**：如右栏 `+ 添加页签`（`RightPanelBar.vue:381`，无 `aria-label`/`title`）、输入区 `+ 附件`、`筛选文件...` 输入框在 `right-panel.json` 里都缺失，但截图可见。故「JSON 无」≠「屏上无」，本表用截图补位并标 `≈`。
2. **opacity-0 的隐藏标签仍会被采集**：`empty.json` 里 `Agent`（463,456）、`技能`（491,455）是 Agent 模式/技能选择器展开时才出现文字的隐藏 `<span>`（按钮本身可见）。本表只记按钮（`IN-06`、`IN-07`），不把这两个标签当作独立命中。
3. **S、K、Q 三态的左栏与主区被折叠/替换**：S/K 是独立路由页（整窗页面），Q 只采到弹窗层；三者都不能用来判断五区元素的去留。
4. **`search.json` 的文本宽度失真**：弹窗标题与描述的 `w` 采成 1（`搜索聊天`、`搜索并打开已有聊天`），宽度不可用，只取 x/y。
5. **会话态的左栏始终展开、右栏始终关闭**：A/P/D/T 四态都是「左栏开 + 右栏关 + 有会话」，因此标题栏里「左栏收起」与「右栏打开」两个分支仍无法覆盖（需另采两态）。
6. **审批卡会把输入区整个换掉**：A 态没有输入框、没有发送/停止按钮，底部只剩审批卡 + 状态条；P/D/T 才有输入区。
7. **滚动位置不同**：D/T 的时间线已滚到底部，部分节点 `y` 为负值（在视口上方），坐标不能当作屏上位置使用。
8. **模态浮层会 aria-hidden 掉背景**：`C6`/`C33`/`P10` 这类 Radix 模态浮层打开时，会把应用根节点整体置 `aria-hidden=true`，采集脚本按 §0.4 的口径跳过它们，因此这类状态只包含浮层本身（节点数 6–16），不含背景元素；判读时以浮层节点 + PNG 为准。

### 0.5 未命中标记的四种原因

| 标记 | 含义 |
|---|---|
| `—（需交互）` | 悬停 / 点击 / 快捷键才出现（菜单、浮层、tooltip、hover 动作） |
| `—（需数据）` | 依赖具体数据或状态（置顶项、出错会话、压缩行、图片附件、更新态…） |
| `—（需平台）` | 平台或模式不符（非 macOS、办公模式、平铺视图、弹窗窗口） |
| `—（需另采）` | 需要另一组采集状态才能覆盖（左栏收起、右栏打开、消息内搜索、空会话） |

---

## 1. 区：标题栏（主区聊天卡片顶部）

主区标题栏有两个互斥变体：空态用 `NewThreadHeader`，会话态用 `ChatHeaderFrame`（内含 `ChatHeader` 的插槽）。两变体在 E/N/R 下都与左栏栏头同高（`--titlebar-height`）。

### 1.1 空态变体（NewThreadHeader，`v-if="isNewThread"`）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| TB-01 | （无文案） | 占位 | — | 空态 ∧ macOS ∧ 非全屏 ∧ 左栏收起 | R/Q（不可见） | R ≈0,0 80×52 | `<ref>/components/chat-workspace/NewThreadHeader.vue:24` |
| TB-02 | 展开侧边栏 | 图标按钮 | PanelLeftOpen | 空态 ∧ 左栏收起 | R/Q | R 95,15 28×28 | `<ref>/components/chat-workspace/NewThreadHeader.vue:47`–`:62` |
| TB-03 | 打开上下文面板 | 图标按钮 | PanelRight | 空态 ∧ 右栏未开 | E/N | E 987,15 28×28 | `<ref>/components/chat-workspace/NewThreadHeader.vue:74`–`:88` |
| TB-04 | 关闭上下文面板 | 图标按钮 | PanelRightClose | 空态 ∧ 右栏已开 | R/Q | R 362,15 28×28 | `<ref>/components/chat-workspace/NewThreadHeader.vue:74`–`:88` |

元素数：4

### 1.2 会话变体外壳（ChatHeaderFrame）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| TB-05 | （无文案） | 占位 | — | 会话 ∧ macOS ∧ 非全屏 ∧ 左栏收起 | C1（占位） | C1 ≈0,0 80×52 | `<ref>/components/chat-workspace/ChatHeaderFrame.vue:10` |
| TB-06 | （无文案） | 浮层容器 | — | 会话 ∧ 消息内搜索打开 | C24 | C24 ≈355,4 375×86 | `<ref>/components/chat-workspace/ChatHeaderFrame.vue:7` |
| TB-07 | 搜索聊天... | 输入框 | search（ThemeIcon） | 会话 ∧ 消息内搜索打开 | C24 | C24 343,23 338×18（placeholder 为「搜索聊天...」，aria 为「搜索聊天消息」） | `<ref>/components/chat-workspace/timeline/ThreadSearchBar.vue:19`–`:30` |
| TB-08 | 关闭搜索 | 图标按钮 | X | 会话 ∧ 消息内搜索打开 | C24 | C24 688,18 28×28 | `<ref>/components/chat-workspace/timeline/ThreadSearchBar.vue:31`–`:41` |
| TB-09 | 上一个搜索结果 | 图标按钮 | ArrowUp | 会话 ∧ 搜索打开 ∧ 有查询词 | C26 | C26 322,61 25×25 | `<ref>/components/chat-workspace/timeline/ThreadSearchBar.vue:50`–`:60` |
| TB-10 | 下一个搜索结果 | 图标按钮 | ArrowDown | 会话 ∧ 搜索打开 ∧ 有查询词 | C26 | C26 357,61 25×25 | `<ref>/components/chat-workspace/timeline/ThreadSearchBar.vue:61`–`:71` |
| TB-11 | 0 个结果 | 文本 | — | 会话 ∧ 搜索打开 ∧ 有查询词 ∧ 命中 0 | C25 | C25 ≈640,64 71×18（`0 个结果`） | `<ref>/components/chat-workspace/timeline/ThreadSearchBar.vue:77`–`:80` |
| TB-12 | `{current} / {total} 个结果` | 文本 | — | 会话 ∧ 搜索打开 ∧ 有查询词 ∧ 命中 >0 | C26 | C26 641,64 71×18（`1 / 35 个结果`） | `<ref>/components/chat-workspace/timeline/ThreadSearchBar.vue:77`–`:80` |
| TB-13 | 展开侧边栏 ⌘B | 图标按钮 | PanelLeftOpen | 会话 ∧ 左栏收起 ∧ 非弹窗窗口 | C1 | C1 95,15 28×28 | `<ref>/components/chat-workspace/WorkspaceHeaderNavigation.vue:31`–`:49` |
| TB-14 | 上一个工作区会话 | 图标按钮 | ArrowLeft | 会话 ∧ 左栏收起 ∧ 非弹窗窗口 ∧ 未隐藏会话导航 | C1 | C1 127,15 28×28 | `<ref>/components/chat-workspace/WorkspaceHeaderNavigation.vue:51`–`:67` |
| TB-15 | 下一个工作区会话 | 图标按钮 | ArrowRight | 同上 | C1 | C1 158,15 28×28 | `<ref>/components/chat-workspace/WorkspaceHeaderNavigation.vue:69`–`:85` |
| TB-16 | 在工作区中新建会话 | 图标按钮 | SquarePen | 会话 ∧ 左栏收起 ∧ 非弹窗窗口 | C1 | C1 190,15 28×28 | `<ref>/components/chat-workspace/ChatHeaderFrame.vue:18`–`:23` |
| TB-17 | `{workspaceTitle}` | 文本 | — | 会话 ∧ 显示工作区标题 | A/P/D/T | A 381,20 60×18（`dim-agent`） | `<ref>/components/chat-workspace/ChatHeaderFrame.vue:27`–`:28` |
| TB-18 | `/` | 文本 | — | 会话 ∧ 显示工作区标题 | A/P/D/T | A ≈455,20 10×18 | `<ref>/components/chat-workspace/ChatHeaderFrame.vue:29` |
| TB-19 | `{title}` | 标题文本（h1） | — | 会话 | A/P/D/T | A 456,19 49×20（`Untitled`） | `<ref>/components/chat-workspace/ChatHeaderFrame.vue:30` |

元素数：15

### 1.3 标题栏动作（`ChatHeader` 插槽 + 右栏开关）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| TB-20 | （无文案） | 图标按钮（菜单触发器） | MoreHorizontal | 会话 | A/P/D/T | A ≈517,22 16×16（截图；采集无标签） | `<ref>/components/chat-workspace/ChatHeader.vue:37`–`:41` |
| TB-21 | 置顶 | 菜单项 | Pin | 会话 ∧ 菜单打开 ∧ 非弹窗窗口 ∧ 未置顶 | C2 | C2 362,48 159×28 | `<ref>/components/chat-workspace/ChatHeader.vue:43`–`:46` |
| TB-22 | 取消置顶 | 菜单项 | PinOff | 会话 ∧ 菜单打开 ∧ 非弹窗窗口 ∧ 已置顶 | C3 | C3 609,48 159×28 | `<ref>/components/chat-workspace/ChatHeader.vue:43`–`:46` |
| TB-23 | 重命名 | 菜单项 | ChevronRight | 会话 ∧ 菜单打开 | C2 | C2 ≈362,76 159×28（禁用，文案成 `重命名（SessionTitle 不可用）`） | `<ref>/components/chat-workspace/ChatHeader.vue:47`–`:54` |
| TB-24 | 在新窗口打开 | 菜单项 | ExternalLink | 会话 ∧ 菜单打开 ∧ 非弹窗窗口 | C2 | C2 ≈362,104 159×28 | `<ref>/components/chat-workspace/ChatHeader.vue:55` |
| TB-25 | 导出 | 菜单项 | Download | 会话 ∧ 菜单打开 | C2 | C2 ≈362,132 159×28 | `<ref>/components/chat-workspace/ChatHeader.vue:59` |
| TB-26 | 归档 | 菜单项 | Archive | 会话 ∧ 菜单打开 | C2 | C2 ≈362,160 159×28 | `<ref>/components/chat-workspace/ChatHeader.vue:63` |
| TB-27 | 复制会话链接 | 菜单项 | Link | 会话 ∧ 菜单打开 ∧ 调试菜单项可见 | C2 | C2 ≈362,196 159×28 | `<ref>/components/chat-workspace/ChatHeader.vue:68` |
| TB-28 | 复制会话 ID | 菜单项 | FileCode | 同上 | C2 | C2 ≈362,224 159×28 | `<ref>/components/chat-workspace/ChatHeader.vue:72` |
| TB-29 | 复制会话路径 | 菜单项 | FolderOpen | 会话 ∧ 菜单打开 ∧ 有工作目录 | C2 | C2 ≈362,252 159×28 | `<ref>/components/chat-workspace/ChatHeader.vue:76` |
| TB-30 | 复制调试信息 | 菜单项 | FileCode | 会话 ∧ 菜单打开 ∧ 调试菜单项可见 | C2 | C2 ≈362,280 159×28 | `<ref>/components/chat-workspace/ChatHeader.vue:80` |
| TB-31 | `{打开方式}` | 图标按钮 | 运行期解析（iconify / 图片 / lucide） | 会话 ∧ 「打开方式」组可见 | A/P/D/T | A 926,16 25×25（`在 VS Code 中打开`） | `<ref>/components/chat-workspace/ChatHeader.vue:92`–`:105` |
| TB-32 | 更多打开方式 | 图标按钮 | ChevronDown | 会话 ∧ 「打开方式」组可见 | A/P/D/T | A 950,16 18×25 | `<ref>/components/chat-workspace/ChatHeader.vue:106`–`:112` |
| TB-33 | 正在检测已安装应用… | 菜单项（禁用） | — | 会话 ∧ 打开方式菜单 ∧ 应用列表加载中 | —（需交互） | — | `<ref>/components/chat-workspace/ChatHeader.vue:116`–`:118` |
| TB-34 | 未检测到可用应用 | 菜单项（禁用） | — | 会话 ∧ 打开方式菜单 ∧ 列表为空 | —（需交互） | — | `<ref>/components/chat-workspace/ChatHeader.vue:139`–`:141` |
| TB-35 | `{应用名}` | 菜单项 | 应用图标 | 会话 ∧ 打开方式菜单 ∧ 每款可用应用 | C4 | C4 501,115 152×28（每款可用应用一行：VS Code / Zed / Terminal / Ghostty） | `<ref>/components/chat-workspace/ChatHeader.vue:127`–`:138` |
| TB-36 | 在文件夹中显示 | 菜单项 | FolderOpen | 会话 ∧ 打开方式菜单 | C4 | C4 501,171 152×28（本机文案为「在 Finder 中显示」） | `<ref>/components/chat-workspace/ChatHeader.vue:143`–`:146` |
| TB-37 | （无文案，系统默认程序） | 菜单项 | 运行期解析 | 会话 ∧ 打开方式菜单 | C4 | C4 501,199 152×28（本机文案为「用系统默认方式打开」） | `<ref>/components/chat-workspace/ChatHeader.vue:147`–`:150` |
| TB-38 | （无文案） | 竖分隔线 | — | 会话 ∧ 「打开方式」组可见 | A/P/D/T | A ≈943,10 1×28（截图） | `<ref>/components/chat-workspace/ChatHeader.vue:153` |
| TB-39 | 打开上下文面板 | 图标按钮 | PanelRight | 会话 ∧ 右栏未开 | A/P/D/T | A 987,15 28×28 | `<ref>/components/chat-workspace/ChatHeader.vue:583`–`:598` |
| TB-40 | 关闭上下文面板 | 图标按钮 | PanelRightClose | 会话 ∧ 右栏已开 | C1 | C1 676,15 28×28 | `<ref>/components/chat-workspace/ChatHeader.vue:583`–`:598` |

元素数：21

**区 1 元素数：40**

---

## 2. 区：左栏

### 2.1 栏头（SidebarHeader）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| LB-01 | （无文案） | 占位 | — | macOS ∧ 非全屏 | E/N/A/P/D/T（不可见） | E ≈0,0 80×12（仅占位，无可见像素） | `<ref>/components/sidebar/SidebarHeader.vue:92`–`:96` |
| LB-02 | （无文案，品牌图形） | 图标 | 品牌 SVG（明暗两套） | 非 macOS | —（需平台/模式） | — | `<ref>/components/sidebar/SidebarHeader.vue:103`–`:106` |
| LB-03 | `DESKTOP_PRODUCT_NAME` | 文本 | — | 非 macOS | —（需平台/模式） | — | `<ref>/components/sidebar/SidebarHeader.vue:107` |
| LB-04 | 更新 | 按钮 | Download | 更新态 `available` ∧ 无错误消息 | —（需数据） | — | `<ref>/components/sidebar/SidebarHeader.vue:58`、`:114`–`:134` |
| LB-05 | `{downloadProgressPercent}%` | 按钮（禁用） | — | 更新态 `downloading` | —（需数据） | — | `<ref>/components/sidebar/SidebarHeader.vue:55`、`:126` |
| LB-06 | 正在准备… | 按钮（禁用） | Loader2 | 更新态 `preparing` | —（需数据） | — | `<ref>/components/sidebar/SidebarHeader.vue:56`、`:130` |
| LB-07 | 重启安装 | 按钮 | ArrowDown | 更新态 `downloaded` | —（需数据） | — | `<ref>/components/sidebar/SidebarHeader.vue:57`、`:131` |
| LB-08 | 重试更新 | 按钮（危险色） | Download | 更新态 `available` ∧ 有错误消息 | —（需数据） | — | `<ref>/components/sidebar/SidebarHeader.vue:58`、`:121`–`:125` |
| LB-09 | `{状态} · v{版本} · {错误消息}` | 浮层（tooltip / aria-label） | — | 悬停更新按钮 | —（需数据） | — | `<ref>/components/sidebar/SidebarHeader.vue:60`–`:69`、`:136` |
| LB-10 | 收起侧边栏 ⌘B | 图标按钮 | PanelLeftClose | 允许显示切换 ∧ 左栏展开 | E/N/A/P/D/T | E 325,14 28×28 | `<ref>/components/sidebar/SidebarHeader.vue:146`–`:164` |

元素数：10

### 2.2 全局入口（GlobalEntries）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| LB-11 | 新建会话 | 菜单行（图标 + 文本） | ThemeIcon:new-session | 常显（禁用由 `disableNewThread`） | E/N/A/P/D/T | E 7,59 346×28；文本 35,64 49×18 | `<ref>/components/sidebar/GlobalEntries.vue:65`–`:76` |
| LB-12 | 搜索 | 菜单行（图标 + 文本） | ThemeIcon:search | 常显 | E/N/A/P/D/T | E 7,91 346×28；文本 35,96 311×18 | `<ref>/components/sidebar/GlobalEntries.vue:79`–`:92` |
| LB-13 | ⌘G（Windows/Linux 为 Ctrl+G） | 快捷键徽标 | — | 悬停或键盘聚焦该行 | C19 | C19 314,96 32×18 | `<ref>/components/sidebar/GlobalEntries.vue:88`–`:90` |
| LB-14 | 定时任务 | 菜单行（图标 + 文本） | Clock3 | 常显（路由为定时任务时高亮） | E/N/A/P/D/T | E 7,122 346×28；文本 35,127 49×18 | `<ref>/components/sidebar/GlobalEntries.vue:95`–`:106` |
| LB-15 | 技能/插件 | 菜单行（图标 + 文本） | LayoutGrid | 常显（路由为技能时高亮） | E/N/A/P/D/T | E 7,154 346×28；文本 35,159 53×18 | `<ref>/components/sidebar/GlobalEntries.vue:109`–`:120` |

元素数：5

### 2.3 置顶区（PinnedThreads）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| LB-16 | （无文案） | 容器 | — | 有置顶会话 ∨ 有置顶工作区 ∨ 置顶分页存在 | C14 | C14 ≈7,189 346×87 | `<ref>/components/sidebar/PinnedThreads.vue:180`–`:182` |
| LB-17 | 已置顶 | 分组标题 | — | LB-16 | C14 | C14 11,197 36×14 | `<ref>/components/sidebar/PinnedThreads.vue:192` |
| LB-18 | （置顶会话行） | 行（复用 TB/LB 会话行组） | 同会话行 | LB-16 ∧ 每条置顶会话 | C14 | C14 ≈7,224 346×28（一条置顶会话行） | `<ref>/components/sidebar/PinnedThreads.vue:213`–`:233` |
| LB-19 | （无文案） | 拖拽落点指示器 | — | 会话拖拽悬停于置顶列表项半区 | —（需数据） | — | `<ref>/components/sidebar/PinnedThreads.vue:197`、`:225` |
| LB-20 | 上一页 | 文字按钮 | — | 存在更早的置顶会话页 | —（需数据） | — | `<ref>/components/sidebar/PinnedThreads.vue:239` |
| LB-21 | 显示更多 | 文字按钮 | — | 存在更多置顶会话页 | —（需数据） | — | `<ref>/components/sidebar/PinnedThreads.vue:240` |
| LB-22 | （置顶工作区分组） | 分组（复用工作区组） | 同工作区行 | LB-16 ∧ 存在置顶工作区 | C15 | C15 ≈7,264 346×28（置顶工作区行） | `<ref>/components/sidebar/PinnedThreads.vue:251`–`:275` |
| LB-23 | （无文案） | 容器态 | — | 会话拖拽进入置顶区 | —（需数据） | — | `<ref>/components/sidebar/PinnedThreads.vue:180` |

元素数：8

### 2.4 列表头（ThreadListHeader）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| LB-24 | 会话 | 文本 | — | 常显（左栏展开） | E/N/A/P/D/T | E 11,197 25×14 | `<ref>/components/sidebar/ThreadListHeader.vue:94` |
| LB-25 | （无文案） | 图标按钮 | FolderPlus | 常显（选择目录中禁用） | E/N/A/P/D/T | E 297,192 25×25 | `<ref>/components/sidebar/ThreadListHeader.vue:98`–`:106` |
| LB-26 | （无文案） | 图标按钮（菜单触发器） | List（分组视图）/ Grid3X3（平铺） | 办公模式隐藏该切换器 | E/N/A/P/D/T | E 325,192 25×25 | `<ref>/components/sidebar/ThreadListHeader.vue:109`–`:119` |
| LB-27 | 排序 | 菜单分组标题 | — | 视图菜单打开 | C11 | C11 200,225 145×25 | `<ref>/components/sidebar/ThreadListHeader.vue:127`–`:129` |
| LB-28 | 最近活跃 | 菜单项 | Check（选中时） | 视图菜单打开 | C11 | C11 200,250 145×28 | `<ref>/components/sidebar/ThreadListHeader.vue:130`–`:137` |
| LB-29 | 最近创建 | 菜单项 | Check（选中时） | 视图菜单打开 | C11 | C11 200,278 145×28 | `<ref>/components/sidebar/ThreadListHeader.vue:138`–`:145` |
| LB-30 | 视图 | 菜单分组标题 | — | 视图菜单打开 | C11 | C11 200,314 145×25 | `<ref>/components/sidebar/ThreadListHeader.vue:149`–`:151` |
| LB-31 | 按工作区分组 | 菜单项 | Check（选中时） | 视图菜单打开 | C11 | C11 200,338 145×28 | `<ref>/components/sidebar/ThreadListHeader.vue:152`–`:159` |
| LB-32 | 平铺 | 菜单项 | Check（选中时） | 视图菜单打开 | C11 | C11 200,366 145×28 | `<ref>/components/sidebar/ThreadListHeader.vue:160`–`:167` |
| LB-33 | 折叠全部 | 菜单项 | ChevronsDownUp | 视图菜单打开 ∧ 当前为分组视图 | C11 | C11 200,402 145×28 | `<ref>/components/sidebar/ThreadListHeader.vue:171`–`:181` |

元素数：10

### 2.5 工作区组（ProjectItem）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| LB-34 | （无文案） | 行按钮（折叠触发器） | — | 该工作区控件已渲染 ∧ 行未向上滚动出视口 | E/N/A/P/D/T | E 14,229 288×18 | `<ref>/components/sidebar/ProjectItem.vue:300`–`:343` |
| LB-35 | （无文案） | 图标（悬停出现） | ChevronRight（展开时旋转 90°） | LB-34 ∧ 行被悬停 | E/N/A/P/D/T（不可见） | E 14,231 14×14 | `<ref>/components/sidebar/ProjectItem.vue:307`–`:317` |
| LB-36 | （无文案） | 图标槽容器 | — | LB-34 | E/N/A/P/D/T | E 14,231 14×14 | `<ref>/components/sidebar/ProjectItem.vue:319`–`:336` |
| LB-37 | （无文案） | 图标 | ThemeIcon:workspace（🏠） | LB-36 ∧ 内置工作区 | E/N/A/P/D/T | E 14,231 14×14 | `<ref>/components/sidebar/ProjectItem.vue:323`–`:327` |
| LB-38 | （无文案） | 图标 | ThemeIcon:folder-open（展开）/ folder（收起） | LB-36 ∧ 非内置工作区 | E/N/A/P/D/T | E 14,375 14×14 | `<ref>/components/sidebar/ProjectItem.vue:328`–`:335` |
| LB-39 | `{工作区名}` | 文本 | — | LB-34 | E/N/A/P/D/T | E 33,229 37×18 | `<ref>/components/sidebar/ProjectItem.vue:338`–`:341` |
| LB-40 | `{工作区路径}` | 浮层（tooltip） | — | 悬停名称 ∧ 该工作区有路径 | —（需交互） | — | `<ref>/components/sidebar/ProjectItem.vue:433`–`:444` |
| LB-41 | （无文案） | 状态位（旋转） | Loader2 | 工作区收起 ∧ 有会话在跑 | —（需数据） | — | `<ref>/components/sidebar/ProjectItem.vue:349`–`:356` |
| LB-42 | （无文案） | 图标按钮 | ThemeIcon:new-session | 行被悬停 ∨ 菜单打开 | E/N（不可见） | 工作区 302,227 21×21；year 302,303 21×21 | `<ref>/components/sidebar/ProjectItem.vue:361`–`:370` |
| LB-43 | （无文案） | 图标按钮（菜单触发器） | MoreHorizontal | 行被悬停 ∨ 菜单打开 | E/N（不可见） | 工作区 325,227 21×21；year 325,303 21×21 | `<ref>/components/sidebar/ProjectItem.vue:372`–`:383` |
| LB-44 | 置顶工作区 | 菜单项 | Pin | 工作区菜单打开 ∧ 非内置工作区 ∧ 未置顶 | C16 | C16 169,386 173×28 | `<ref>/components/sidebar/ProjectItem.vue:390`–`:397` |
| LB-45 | 取消置顶工作区 | 菜单项 | Pin | 同上 ∧ 已置顶 | C17 | C17 169,301 173×28 | `<ref>/components/sidebar/ProjectItem.vue:390`–`:397` |
| LB-46 | 在文件夹中显示 | 菜单项 | ThemeIcon:folder | 工作区菜单打开 ∧ （内置 ∨ 有路径） | C16 | C16 169,414 173×28（本机文案为「在文件管理器中显示」） | `<ref>/components/sidebar/ProjectItem.vue:398`–`:405` |
| LB-47 | 归档会话 | 菜单项 | Archive | 工作区菜单打开 ∧ 该组会话数 > 0 | C45 | C45 169,406 173×28 | `<ref>/components/sidebar/ProjectItem.vue:406`–`:413` |
| LB-48 | 重命名 | 菜单项 | Edit3 | 工作区菜单打开 | C16 | C16 169,450 173×28 | `<ref>/components/sidebar/ProjectItem.vue:415`–`:418` |
| LB-49 | 移除工作区 | 菜单项（危险色） | Trash2 | 工作区菜单打开 ∧ 非内置工作区 | C16 | C16 169,478 173×28 | `<ref>/components/sidebar/ProjectItem.vue:419`–`:427` |
| LB-50 | 暂无会话 | 文本 | — | 组展开 ∧ 会话数 0 | E/N/A/P/D/T | E 33,266 42×13 | `<ref>/components/sidebar/ProjectItem.vue:457`–`:462` |
| LB-51 | （无文案） | 状态行（旋转 + 加载中） | Loader2 | 组展开 ∧ 该组会话分页加载中 | —（需数据） | — | `<ref>/components/sidebar/ProjectItem.vue:455` |
| LB-52 | （会话行实例） | 行 | 见 2.7 | 组展开 ∧ 每条会话 | 全部含会话的状态（A/C14/C12…） | C14 ≈7,224 346×28（置顶组内一行）；分组视图下同 2.7 | `<ref>/components/sidebar/ProjectItem.vue:465`–`:468` |
| LB-53 | 上一页 | 文字按钮 | — | 组展开 ∧ 该组存在更早会话页 | —（需数据） | — | `<ref>/components/sidebar/ProjectItem.vue:470` |
| LB-54 | 显示更多 | 文字按钮 | ChevronRight | 组展开 ∧ 该组尚有未展示会话 | —（需数据） | — | `<ref>/components/sidebar/ProjectItem.vue:473`–`:489` |
| LB-55 | 显示更少 | 文字按钮 | ChevronRight（旋转 90°） | LB-54 ∧ 已超过阈值 ∧ 无隐藏项 ∧ 无更多页 | —（需数据） | — | `<ref>/components/sidebar/ProjectItem.vue:481`–`:484` |
| LB-56 | `{工作区名}` | 占位（虚拟化代替行） | — | 控件已滚出视口（不渲染控件） | —（需数据） | — | `<ref>/components/sidebar/ProjectItem.vue:494`–`:508` |
| LB-57 | （无文案） | 分隔线 | — | 内置工作区组之下 | E/N/A/P/D/T | E ≈7,295 346×1 | `<ref>/components/sidebar/ProjectItem.vue:511`–`:515` |

元素数：24

### 2.6 列表尾部与平铺视图（ThreadList）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| LB-58 | 显示更多 | 文字按钮 | — | 分组视图 ∧ 存在更多工作区页 | E/N | 7,1652 346×28 | `<ref>/components/sidebar/ThreadList.vue:240` |
| LB-59 | 上一页 | 文字按钮 | — | 分组视图 ∧ 存在更早工作区页 | —（需数据） | — | `<ref>/components/sidebar/ThreadList.vue:239` |
| LB-60 | （无文案） | 拖拽落点指示线（两处） | — | 分组视图 ∧ 工作区拖拽落到第 0 位 / 第 n+1 位 | —（需交互） | — | `<ref>/components/sidebar/ThreadList.vue:190`、`:216` |
| LB-61 | （会话行） | 行（复用 2.7） | — | 平铺视图 ∧ 每行 | C12 | C12 35,264 149×18（行本身即 2.7 会话行） | `<ref>/components/sidebar/ThreadList.vue:249` |
| LB-62 | `{日期分组}` | 分组标题 | — | 平铺视图 ∧ 每个日期段（今天/昨天/更早/`{日期}`） | C12 | C12 7,227 346×23（`今天`） | `<ref>/components/sidebar/ThreadList.vue:250` |
| LB-63 | （无文案） | 状态行（旋转） | Loader2 | 平铺视图 ∧ 会话分页加载中 | —（需平台/模式） | — | `<ref>/components/sidebar/ThreadList.vue:246` |
| LB-64 | 显示更多 | 文字按钮 | ChevronRight | 平铺视图 ∧ 尚有未展示项或更多页 | —（需平台/模式） | — | `<ref>/components/sidebar/ThreadList.vue:254`–`:269` |
| LB-65 | 显示更少 | 文字按钮 | ChevronRight（旋转 90°） | LB-64 ∧ 已超过阈值 ∧ 无隐藏项 ∧ 无更多页 | —（需平台/模式） | — | `<ref>/components/sidebar/ThreadList.vue:261`–`:264` |

元素数：8

### 2.7 会话行（ThreadItem）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| LB-66 | （无文案） | 状态位 | AlertCircle（红色） | 会话行 ∧ 该会话出错 | —（需数据） | — | `<ref>/components/sidebar/ThreadItem.vue:260`–`:263` |
| LB-67 | （无文案） | 状态位（旋转） | Loader2 | 会话行 ∧ 运行中 ∧ 未悬停 | A/P/T | A 11,259 21×21 | `<ref>/components/sidebar/ThreadItem.vue:264`–`:269` |
| LB-68 | （无文案） | 按钮 | Pin | 会话行 ∧ 悬停 ∧ 允许置顶切换 | C18（悬停可见） | C18 ≈294,250 21×21 | `<ref>/components/sidebar/ThreadItem.vue:270`–`:289` |
| LB-69 | （无文案） | 图标 | Pin | 会话行 ∧ 置顶列表内（`showUnpinOnHover`）∧ 非悬停 | C14 | C14 ≈35,229 14×14 | `<ref>/components/sidebar/ThreadItem.vue:281`–`:288` |
| LB-70 | （无文案） | 状态点 | — | 会话行 ∧ 未读 ∧ 不在上述分支 | —（需数据） | — | `<ref>/components/sidebar/ThreadItem.vue:291`–`:294` |
| LB-71 | （无文案） | 图标 | Star | 会话行 ∧ 已置顶 ∧ 不隐藏置顶图标 ∧ 不在上述分支 | C14 | C14 ≈35,229 14×14 | `<ref>/components/sidebar/ThreadItem.vue:295`–`:299` |
| LB-72 | `{会话标题}` | 文本（跑马灯） | — | 会话行 | A/P/D/T | A 35,260 59×18（`Untitled`） | `<ref>/components/sidebar/ThreadItem.vue:301`–`:304` |
| LB-73 | 等待操作 | 标签 | — | 会话行 ∧ 有待处理交互 | A | A 167,260 40×18 | `<ref>/components/sidebar/ThreadItem.vue:308`–`:313` |
| LB-74 | `{+a −b}` | 文本 | — | 会话行 ∧ 有改动摘要 ∧ 办公模式外 | —（需数据） | — | `<ref>/components/sidebar/ThreadItem.vue:317`–`:322` |
| LB-75 | `{⌘1..9}` | 快捷键徽标 | — | 会话行 ∧ 修饰键按住 | —（需交互） | — | `<ref>/components/sidebar/ThreadItem.vue:329`–`:332` |
| LB-76 | （无文案） | 图标 | Clock3 | 会话行 ∧ 来源为定时任务 ∧ 未悬停 ∧ 无快捷键徽标 | —（需数据） | — | `<ref>/components/sidebar/ThreadItem.vue:337`–`:345` |
| LB-77 | `{相对时间}` | 文本 | — | 会话行 ∧ 非定时来源 ∧ 未悬停 ∧ 无快捷键徽标 | A/P/D/T | A 328,260 22×18（`now`） | `<ref>/components/sidebar/ThreadItem.vue:346`–`:352` |
| LB-78 | （无文案） | 按钮 | Archive | 会话行 ∧ 悬停 ∧ 无快捷键徽标 | C18（悬停可见） | C18 329,259 21×21（aria：归档会话） | `<ref>/components/sidebar/ThreadItem.vue:355`–`:364` |
| LB-79 | `{会话标题}` | 浮层（hover 卡） | — | 悬停会话行 | C18（悬停可见） | C18 ≈290,170 120×20（hover 卡标题） | `<ref>/components/sidebar/ThreadItem.vue:376`–`:383` |
| LB-80 | `{所属工作区}` | 浮层（hover 卡） | ThemeIcon:workspace / folder | 悬停会话行 | C18（悬停可见） | C18 ≈300,196 40×13（所属工作区） | `<ref>/components/sidebar/ThreadItem.vue:386`–`:392` |
| LB-81 | `{git 分支}` | 浮层（hover 卡） | GitBranch | 悬停会话行 ∧ 有分支 ∧ 办公模式外 | —（需交互） | — | `<ref>/components/sidebar/ThreadItem.vue:393`–`:400` |
| LB-82 | `{分支不匹配警告}` | 浮层（hover 卡） | AlertCircle | 悬停会话行 ∧ 分支不一致 ∧ 办公模式外 | —（需交互） | — | `<ref>/components/sidebar/ThreadItem.vue:401`–`:410` |

元素数：17

### 2.7.1 会话行右键菜单（ThreadContextMenu，本轮补登记）

上一版漏登记该组件（`ThreadItem.vue:225` 引用的独立组件），本轮按 `C13` 采集补入。

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| LB-107 | 置顶 / 取消置顶 | 菜单项 | — | 会话行右键 ∧ （未置顶 ∨ 已置顶） | C13、C3 | C13 157,268 187×28 | `<ref>/components/sidebar/ThreadContextMenu.vue:100`–`:112` |
| LB-108 | 重命名 | 菜单项（禁用态带原因） | — | 会话行右键 ∧ 非多选 | C13 | C13 157,296 187×28（禁用，文案成 `重命名（SessionTitle 不可用）`） | `<ref>/components/sidebar/ThreadContextMenu.vue:114`–`:127` |
| LB-109 | 在新窗口中打开 | 菜单项 | — | 会话行右键 ∧ 非多选 | C13 | C13 157,324 187×28 | `<ref>/components/sidebar/ThreadContextMenu.vue:129`–`:138` |
| LB-110 | 导出 | 菜单项 | — | 会话行右键 ∧ 非多选 | C13 | C13 157,352 187×28 | `<ref>/components/sidebar/ThreadContextMenu.vue:140`–`:148` |
| LB-111 | 归档 | 菜单项 | — | 会话行右键 ∧ 非多选 | C13 | C13 157,388 187×28 | `<ref>/components/sidebar/ThreadContextMenu.vue:156`–`:162` |
| LB-112 | 删除 | 菜单项 | — | 会话行右键 | C13 | C13 157,416 187×28 | `<ref>/components/sidebar/ThreadContextMenu.vue:164`–`:172` |

元素数：6

### 2.8 账户行（AccountEntry）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| LB-83 | （无文案） | 行按钮（菜单触发器） | — | 常显（贴底） | E/N/A/P/D/T | E 7,688 316×30 | `<ref>/components/sidebar/AccountEntry.vue:211`–`:228` |
| LB-84 | `{首字母}` | 头像文本 | — | LB-83 ∧ 已登录 | E/N/A/P/D/T | E 17,697 11×13 | `<ref>/components/sidebar/AccountEntry.vue:220` |
| LB-85 | （无文案） | 头像图标 | UserIcon | LB-83 ∧ 未登录 ∧ 非登录中 | —（需数据） | — | `<ref>/components/sidebar/AccountEntry.vue:219` |
| LB-86 | （无文案） | 头像状态（旋转） | Loader2 | LB-83 ∧ 登录中 | —（需数据） | — | `<ref>/components/sidebar/AccountEntry.vue:218` |
| LB-87 | （无文案） | 错误点 | — | LB-83 ∧ 有错误 ∧ 非登录中 | A/P/D/T | A ≈20,690 8×8（截图红点） | `<ref>/components/sidebar/AccountEntry.vue:221`–`:225` |
| LB-88 | `{账户名}` / 登录 Dim / 正在登录… | 文本 | — | LB-83 | E/N/A/P/D/T | E 39,694 281×18 | `<ref>/components/sidebar/AccountEntry.vue:69`–`:78`、`:227` |
| LB-89 | （无文案） | 菜单 | — | 点击账户行 | C32 | C32 ≈7,492 316×186 | `<ref>/components/sidebar/AccountEntry.vue:230`–`:360` |
| LB-90 | `{首字母}` + `{账户名}` | 菜单项（身份） | — | 账户菜单打开 | C32 | C32 ≈42,494 214×20（`ER` + `eric`） | `<ref>/components/sidebar/AccountEntry.vue:242`–`:251` |
| LB-91 | 用量 | 菜单项 | Gauge | 账户菜单打开 ∧ 已登录 | C32 | C32 42,523 153×18 | `<ref>/components/sidebar/AccountEntry.vue:256`–`:270` |
| LB-92 | 剩余 {pct}% | 菜单项尾注 | — | LB-91 ∧ 已知剩余额度 | C32 | C32 200,523 56×18（`剩余 61%`） | `<ref>/components/sidebar/AccountEntry.vue:264`–`:266` |
| LB-93 | 显示 Pet / 收起 Pet | 菜单项 | Smile | 账户菜单打开 ∧ 已登录 | C32 | C32 42,547 214×18（菜单项文案「显示 Pet」） | `<ref>/components/sidebar/AccountEntry.vue:271`–`:289` |
| LB-94 | `{Pet 不可用说明}` | 菜单内说明 | — | 账户菜单打开 ∧ 浮动 Pet 不可用 | —（需交互） | — | `<ref>/components/sidebar/AccountEntry.vue:290`–`:296` |
| LB-95 | 邀请朋友 | 菜单项 | Send | 账户菜单打开 ∧ 已登录 ∧ 邀请可用 | C32 | C32 42,603 214×18 | `<ref>/components/sidebar/AccountEntry.vue:298`–`:308` |
| LB-96 | 加载中 | 菜单项（禁用） | Loader2 | 账户菜单打开 ∧ 邀请信息加载中 | —（需交互） | — | `<ref>/components/sidebar/AccountEntry.vue:310`–`:320` |
| LB-97 | 设置 | 菜单项 + 快捷键 | ThemeIcon:settings | 账户菜单打开 ∧ 已登录 | C32 | C32 42,628 196×18（`设置 ⌘,`） | `<ref>/components/sidebar/AccountEntry.vue:321`–`:331` |
| LB-98 | 退出登录 | 菜单项 | LogOut | 账户菜单打开 ∧ 已登录 | C32 | C32 42,657 214×18 | `<ref>/components/sidebar/AccountEntry.vue:336`–`:346` |
| LB-99 | 登录 Dim | 菜单项 | UserIcon | 账户菜单打开 ∧ 未登录 | —（需交互） | — | `<ref>/components/sidebar/AccountEntry.vue:348`–`:359` |
| LB-100 | 设置 | 图标按钮 | ThemeIcon:settings | 常显（贴底右侧） | E/N/A/P/D/T | E 323,688 30×30 | `<ref>/components/sidebar/AccountEntry.vue:362`–`:371` |
| LB-101 | 退出 Dim？ | 对话框标题 | — | 请求退出确认 | C33 | C33 313,315 404×25 | `<ref>/components/sidebar/AccountEntry.vue:376`–`:380` |
| LB-102 | 退出后，需要重新登录才能使用 Dim 账户。 | 对话框描述 | — | LB-101 | C33 | C33 313,347 404×18 | `<ref>/components/sidebar/AccountEntry.vue:381`–`:383` |
| LB-103 | 取消 | 对话框按钮 | — | LB-101 | C33 | C33 579,378 55×32 | `<ref>/components/sidebar/AccountEntry.vue:386`–`:388` |
| LB-104 | 退出登录 | 对话框主按钮（危险色） | — | LB-101 | C33 | C33 640,378 77×32 | `<ref>/components/sidebar/AccountEntry.vue:389`–`:396` |

元素数：22

### 2.9 整栏拖入提示（AppSidebar）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| LB-105 | 松开以添加工作区 | 提示文本 | FolderPlus | 从文件管理器拖入文件夹 ∧ 提示可见 | C31 | C31 131,357 98×18 | `<ref>/components/sidebar/AppSidebar.vue:112`–`:121` |
| LB-106 | 仅限文件夹 · 同时新建会话 | 提示副文本 | — | LB-105 | C31 | C31 118,382 124×14 | `<ref>/components/sidebar/AppSidebar.vue:120` |

元素数：2

**区 2 元素数：112**

---

## 3. 区：时间线

E/N/R 三态没有会话，时间线容器被 `NewThread`（空态）占位；A/P/D/T 四态有会话，本区大部分元素据此命中。仍未命中的，都是需要特定数据（图片/引用/标注/压缩/错误/产物）或特定交互（展开详情、滚离底部、消息内搜索）的分支，逐行标出原因。

### 3.1 外壳与空态

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| TL-01 | （无文案） | 容器（滚动区） | — | 常显（空态时由空态节填充） | E/N/R | E ≈356,0 674×725 | `<ref>/components/chat-workspace/ChatWorkspace.vue:6585`–`:6616` |
| TL-02 | 当前会话还没有消息 | 文本 | — | 会话 ∧ 消息数 0 ∧ 未运行 ∧ 未处于恢复遮罩 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/PlainTimeline.vue:42`–`:44` |
| TL-03 | （无文案，仅读屏） | 加载条 | — | 会话 ∧ 正在加载更早历史 | —（需交互） | — | `<ref>/components/chat-workspace/timeline/PlainTimeline.vue:48`–`:66` |
| TL-04 | （无文案） | 容器（用户消息轨道） | — | 会话 ∧ 轨道可用 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/PlainTimeline.vue:68`–`:81` |
| TL-05 | （无文案） | 标记（每条用户消息） | — | TL-04 ∧ 每条用户消息 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessageRail.vue` |
| TL-06 | （无文案） | 占位（虚拟化上/下） | — | 会话 ∧ 渲染窗口外仍有行 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/PlainTimeline.vue:90`–`:96`、`:137`–`:143` |
| TL-07 | （无文案） | 行外壳 | — | 会话 ∧ 每条渲染行 | A/P/D/T | A ≈356,60 674×560（行区） | `<ref>/components/chat-workspace/timeline/PlainTimeline.vue:97`–`:110` |
| TL-08 | （无文案） | 图标按钮 | ChevronDown | 会话 ∧ 视口距底部超过阈值 | C23 | C23 ≈550,529 24×24 | `<ref>/components/chat-workspace/timeline/PlainTimeline.vue:3610`–`:3617` |
| TL-09 | （无文案） | 骨架屏 | — | 会话切换/恢复中 ∧ 加载遮罩可见 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineSkeleton.vue`；挂载点 `ChatWorkspace.vue:6657`–`:6666` |
| TL-10 | 该消息渲染失败。 | 兜底文本 | — | 某行渲染抛错 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineErrorBoundary.vue` |

元素数：10

### 3.2 行类型与轮结构（TimelineRow）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| TL-11 | （无文案） | 轮容器 | — | 行类型 = 轮 | A/P/D/T | A 385,142 617×21 | `<ref>/components/chat-workspace/timeline/TimelineRow.vue:4`–`:5` |
| TL-12 | （无文案） | 助手区容器 | — | 轮 ∧ （有内容块 ∨ 有终端信息） | A/P/D/T | A 385,142 617×21 | `<ref>/components/chat-workspace/timeline/TimelineRow.vue:15`–`:18` |
| TL-13 | （无文案） | 轮脚容器 | — | 轮（运行中时整体 inert） | A/P/D/T | A ≈385,470 617×40 | `<ref>/components/chat-workspace/timeline/TimelineRow.vue:48`–`:54` |
| TL-14 | （本轮产物块） | 组件 | — | 轮 ∧ 非演示模式 ∧ 非只读 ∧ 有 runId ∧ 是最新一轮 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineRow.vue:56`–`:65`（元素见 `inventory-panels.md` G） |
| TL-15 | （本轮改动块） | 组件 | — | 轮 ∧ 非演示模式 ∧ 非只读 ∧ 办公模式外 ∧ 有 runId ∧ 有用户消息 ∧ 是最新一轮 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineRow.vue:67`–`:77`（元素见 `inventory-panels.md` F） |
| TL-16 | （无文案） | 分隔行 | — | 行类型 = 显示 ∧ 窗口截断 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineRow.vue:110`–`:113` |
| TL-17 | （压缩分隔线） | 分隔行 | — | 行类型 = 显示 ∧ 压缩；或角色 = divider | C27 | C27 509,348 84×21 | `<ref>/components/chat-workspace/timeline/TimelineRow.vue:115`–`:119` |
| TL-18 | `{系统通知文本}` | 文本块 | — | 消息 ∧ 系统通知消息 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineRow.vue:135`–`:144` |
| TL-19 | `{消息正文}` | 文本块 | — | 消息 ∧ 其它角色 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineRow.vue:146`–`:148` |

元素数：9

### 3.3 用户消息（TimelineUserMessage）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| TL-20 | （无文案） | 气泡容器 | — | 用户消息 ∧ 有内容 ∧ 非编辑中 | A/P/D/T | A ≈810,64 190×50 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:952` |
| TL-21 | （无文案） | 图片廊 | — | 用户消息 ∧ 有图片附件 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:902`–`:906` |
| TL-22 | （无文案） | 附件 chip 架 | — | 用户消息 ∧ 有 chip 附件 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:909`–`:919` |
| TL-23 | （无文案） | 引用列表 | — | 用户消息 ∧ 有已发送引用 ∧ 非演示 ∧ 非编辑 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:921`–`:926` |
| TL-24 | `mark {num}` + 选择器 | 标注卡 | — | 用户消息 ∧ 有标注 ∧ 非演示 ∧ 非编辑 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:928`–`:950` |
| TL-25 | 定时唤醒来源 | 文本 | — | 用户消息 ∧ 元数据 `scheduleWakeup` | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:955`–`:961` |
| TL-26 | `{消息正文}` | 文本 | — | TL-20 | A/P/D/T | A 833,83 154×17 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:963`–`:1053` |
| TL-27 | `{技能名}` | 行内 chip | BoxIcon | 正文中技能提及 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1004`–`:1012` |
| TL-28 | `{文件/目录名}` | 行内按钮 | 文件图标（图片） | 正文中文件提及 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1013`–`:1030` |
| TL-29 | `{commit hash}` | 行内 chip | GitCommitHorizontal | 正文中提交提及 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1031`–`:1039` |
| TL-30 | `{链接标题}` | 行内链接 | Globe / 站点图标 | 正文中链接提及 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1040`–`:1052` |
| TL-31 | 展示全部 / 收起 | 文字按钮 | ChevronDown（展开时旋转） | TL-20 ∧ 内容溢出 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1055`–`:1067` |
| TL-32 | （无文案） | 行内编辑器 | — | 用户消息 ∧ 有文本 ∧ （编辑中 ∨ 无内容） | C28 | C28 ≈685,404 240×56 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1073`–`:1108` |
| TL-33 | 取消 | 图标按钮 | X | TL-32 | C28 | C28 661,434 21×21 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1090`–`:1096` |
| TL-34 | 发送消息... | 图标按钮 | ArrowUp | TL-32 | C28 | C28 685,434 21×21 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1097`–`:1106` |
| TL-35 | `{发送时间}` | 文本 | — | 用户消息 ∧ （有内容 ∨ 有附件）∧ 非编辑 | A/P/D/T（悬停可见） | A 810,123 75×12 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1118`–`:1123` |
| TL-36 | 取消（撤回） | 图标按钮 | X | `canWithdraw` 只在消息状态为 `sending` 时成立；A/P/D/T 的消息都已发出 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1124`–`:1135` |
| TL-37 | 复制整个消息 | 图标按钮 | Copy / Check | 同 TL-35 | A/P/D/T（悬停可见） | A 893,117 25×25 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1136`–`:1153` |
| TL-38 | 编辑消息 | 图标按钮 | Pencil | 同 TL-35 ∧ 可编辑 ∧ 有文本 | A/P/D/T（悬停可见） | A 921,117 25×25 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1154`–`:1170` |
| TL-39 | 从此处分叉 | 图标按钮 | GitFork | 同 TL-35 ∧ 可分叉 ∧ 非弹窗窗口 | A/P/D/T（悬停可见） | A 949,117 25×25 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1171`–`:1186` |
| TL-40 | 从此处开始侧边对话 | 图标按钮 | GitBranch | 同 TL-35 ∧ 可分支 ∧ 非弹窗窗口 | A/P/D/T（悬停可见） | A 977,117 25×25 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1187`–`:1202` |
| TL-41 | 从此处开始侧边对话 | 右键菜单项 | GitBranch | 用户消息右键 ∧ 可分支 | C29 | C29 593,356 110×18 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1206`–`:1213` |
| TL-42 | 复制 | 右键菜单项 | Copy | 用户消息右键 ∧ 已选中文本 | C29 | C29 ≈593,384 74×18 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1214`–`:1221` |
| TL-43 | 复制整个消息 | 右键菜单项 | Copy | 用户消息右键 | C29 | C29 593,412 74×18 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1222`–`:1229` |
| TL-44 | （无文案） | 选区引用浮层 | — | 非只读 ∧ 选中文本 ∧ 浮层打开 | —（需交互） | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1237`–`:1244` |

元素数：25

### 3.4 助手消息（TimelineAssistantMessage 及块组件）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| TL-45 | 工作过程 / `工作过程 · {actions}` | 文字按钮（折叠行） | ChevronRight（展开旋转） | 助手消息 ∧ 有被收起的内容块 ∧ 允许显示折叠 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:895`–`:915` |
| TL-46 | `已经工作了 {duration}` / `已经工作了 {duration} · {actions}` | 同上 | 同上 | 同上（运行中/完成） | C40 | C40 261,156 179×14（`已经工作了 48.5s · 执行命令`） | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:560`–`:574` |
| TL-47 | 已停止 / `已停止 · {duration}` | 同上 | 同上 | 同上（被停止） | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:556`–`:560` |
| TL-48 | `{duration} 后失败` / `{duration} 后失败 · {actions}` | 同上（警告色） | 同上 | 同上（失败） | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:542`–`:550` |
| TL-49 | 出错前的工作过程 / `出错前的工作过程 · {actions}` | 同上 | 同上 | 同上（出错且无终端信息） | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:551`–`:554` |
| TL-50 | （无文案） | 块容器 | — | 助手消息 | A/P/D/T | A ≈385,130 617×400 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:917` |
| TL-51 | （工具调用分组） | 折叠组 | — | 连续工具调用被合并 | A/P/D/T | A 385,142 617×21 | `<ref>/components/chat-workspace/timeline/TimelineBlockGroup.vue` |
| TL-52 | 正在思考 / `思考了 {duration}` | 折叠行 | — | 思考块 | A/P/D/T | A 421,168 62×21（`思考了 0s`）；P 显示 `正在思考 · 0s` | `<ref>/components/chat-workspace/timeline/TimelineThinkingItem.vue` |
| TL-53 | （markdown 正文） | 富文本 | — | 文本块 | D/T | D 385,-1237 554×17（已滚出视口）；T 385,126 554×17 | `<ref>/components/chat-workspace/timeline/TimelineTextItem.vue` |
| TL-54 | （工具调用标题行） | 行按钮 | 工具图标 | 工具调用块 | A/P/D/T | A 401,202 62×21（`执行 ls`）；T 401,184 155×21（`读取 package.json`） | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue` |
| TL-55 | （工具调用详情） | 展开区 | ChevronRight | 工具调用块 ∧ 已展开 | C27 | C27 ≈400,600 620×300（输出 / 说明 / 日志 详情表） | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue` |
| TL-56 | `{子代理名}` + `{通知文本}` | 行按钮（chip） | Bot + ProviderIcon | 通知块 ∧ 子代理通知 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:986`–`:1045` |
| TL-57 | 运行失败 / 本次运行失败。 | 文本 | — | 块状态失败 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:773`–`:776` |
| TL-58 | `{seconds}s 后重试` / 重试中 / 已重试 / 已取消 | 提示 + 按钮 | — | 重试通知块 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineRetryNotice.vue:36`–`:47` |
| TL-59 | （终端错误详情标题）+ 复制 | 折叠区 + 图标按钮 | Copy | 终端失败 ∧ 有调试详情 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1247`–`:1262` |
| TL-60 | 用户中断 | 通知文本 | — | 运行被中断 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:776` |
| TL-61 | 复制整个消息 | 图标按钮 | Copy / Check | 轮末动作条 ∧ 有可复制内容 | D（悬停可见） | D 385,504 25×25 | `<ref>/components/chat-workspace/timeline/TimelineAssistantActions.vue:13`–`:20` |
| TL-62 | 从此处分叉 | 图标按钮 | GitFork | 轮末动作条 ∧ 有分叉锚点 ∧ 非弹窗窗口 | D（悬停可见） | D 416,504 25×25 | `<ref>/components/chat-workspace/timeline/TimelineAssistantActions.vue:21`–`:28` |
| TL-63 | 重试（或自定义重试文案） | 图标按钮 | RotateCcw | 轮末动作条 ∧ 可重试 | D（悬停可见） | D 448,504 25×25 | `<ref>/components/chat-workspace/timeline/TimelineAssistantActions.vue:29`–`:36` |
| TL-64 | `{完成时间}` | 文本 | — | 轮末动作条 ∧ 有完成时间 | D（悬停可见） | D 479,510 75×12（`星期二 21:23`） | `<ref>/components/chat-workspace/timeline/TimelineAssistantActions.vue:37`–`:41` |
| TL-65 | 复制 / 复制整个消息 | 右键菜单项（2 条） | Copy | 助手消息右键 | C30 | C30 635,345 23×23（复制）+ 同级「复制整个消息」 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1298`–`:1315` |

元素数：21

### 3.5 分隔与压缩

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| TL-66 | （无文案） | 分隔线 | — | 窗口截断行 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineWindowTruncationDivider.vue` |
| TL-67 | 已压缩 / 自动压缩 + `(部分)` | 分隔行文本 | — | 压缩行（完成态） | C27 | C27 509,348 84×21（本机文案为「上下文已压缩」） | `<ref>/components/chat-workspace/timeline/TimelineCompressionDivider.vue:75`–`:78` |
| TL-68 | 正在压缩 / 压缩失败 / 已跳过 / 已取消 | 分隔行文本 | Loader2 | 压缩行（进行/失败/跳过/取消） | —（需数据） | — | `<ref>/components/chat-workspace/timeline/TimelineCompressionDivider.vue:63`–`:73` |
| TL-69 | 摘要生成中 / 摘要不可用 / 加载失败 + 重试 | 分隔行内文本/按钮 | — | 压缩行展开 | C27 | C27 400,395 86×18（`无法加载摘要。`）+ 654,390 48×28（`重试`） | `<ref>/components/chat-workspace/timeline/TimelineCompressionDivider.vue:192`–`:217` |

元素数：4

### 3.6 运行态才出现的行内元素（本次新增观测）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| TL-70 | `正在处理` / `正在处理 · {时长}` | 行内文本（微光） | — | 会话 ∧ 该轮运行中 ∧ 存在活动起点 | P/T | T 385,492 118×21（`正在处理 · 1m 28s`）；P `正在处理 · 39s` | `<ref>/components/chat-workspace/timeline/PlainTimeline.vue:3577`–`:3587`；文案 `<ref>/i18n/zh-CN.ts:1201` |
| TL-71 | （无文案） | 恢复遮罩 | BrandMark（动画） | 会话切换 / 历史位置恢复中 | —（需数据） | — | `<ref>/components/chat-workspace/timeline/PlainTimeline.vue:3589`–`:3599` |
| TL-72 | `{分组汇总}`，如 `思考1轮 · 改2次文件、搜1次、抓取2次、其他1次` | 分组头文本 | Workflow | 轮内连续工具/思考被合并成组 | A/P/D/T | A 385,142 617×21（`思考1轮·执行1次命令`）；T 404,157 349×21 | `<ref>/components/chat-workspace/timeline/TimelineBlockGroup.vue:6`–`:28`；聚合文案 `<ref>/i18n/zh-CN.ts:1313`–`:1327` |
| TL-73 | （无文案） | 分组折叠箭头 | ChevronRight（展开旋转 90°） | TL-72 所在分组 | A/P/D/T | A ≈600,148 16×16 | `<ref>/components/chat-workspace/timeline/TimelineBlockGroup.vue:31`–`:35` |
| TL-74 | （无文案） | 分组内容轨道 | — | 分组展开 | T | T ≈400,175 620×330（左竖线 + 缩进） | `<ref>/components/chat-workspace/timeline/TimelineBlockGroup.vue:38`–`:44` |
| TL-75 | `+{新增}` / `-{删除}` | 行内数字 | — | 工具调用 ∧ 有 diff 统计 | T | T `+19` 582,287 26×14；`+2`/`-2` 582,384 / 602,384 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:28`–`:34` |
| TL-76 | （无文案，站点图标） | 图标批次 | 站点 favicon（`FaviconImg`） | 网页搜索/抓取工具调用 ∧ 有主机名 | T | T `E` 480,318 16×16；`H` 480,414 16×16 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:35`–`:66` |
| TL-77 | `✗` | 状态标记 | — | 网页类工具调用失败 | T | T 688,313 9×14（搜索网络失败）；591,319（获取网页失败） | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:67`–`:72` |
| TL-78 | （无文案） | 展开箭头 | ChevronRight（展开旋转 90°） | 工具调用 ∧ 有详情 | A/P/D/T | A ≈470,208 16×16 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:73`–`:78` |

备注（`T` 的八个采集）：`tools-1..5`、`tools-done`、`tools-final`、`tools-rich` 的节点集相同；差异只在计时文案（`思考了 0.3s` / `0.5s`、`正在处理 · 1m 28s`）与滚动位置（`tools-rich` 部分节点 `y` 为负），故合并记为 `T`。

元素数：9

**区 3 元素数：78**

---

## 4. 区：输入区

E/N/R 的输入区由空态（`NewThread`）挂载 `InputArea`；P/D/T 由 `ChatWorkspace` 底部栈挂载同一个 `InputArea`；A 态则被审批卡整体替换（见 §4.5）。包装差异见第 6 节。

### 4.1 输入框与工具行

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| IN-01 | （无文案） | 容器（圆角描边输入块） | — | 常显（空态整块带投影，控制行灰底） | E/N/R/P/D/T | E ≈396,372 582×110；D 386,581 615×60 | `<ref>/components/chat-workspace/composer/ConversationComposer.vue:59`–`:66` |
| IN-02 | 发送消息... | 占位文本 | — | 常显（草稿为空） | E/N/R/P/D/T | E ≈425,399；D ≈390,588（`发送消息...`） | `<ref>/components/chat-workspace/composer/InputArea.vue:2334` |
| IN-03 | 斜杠命令菜单 | 浮层 | — | 输入 `/` 且编辑器聚焦 | C37 | C37 390,328 327×25（分组标题「指令」）+ 首项 401,358（`/agent`） | `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue:178`–`:183` |
| IN-04 | 提示建议（接受/拒绝） | 行内浮层 | — | 有提示建议 | —（需交互） | — | `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue` |
| IN-05 | （无文案） | 图标按钮 | Plus | 常显（工具行首个） | E/N/R/P/D/T | E ≈424,461 28×28；D 410,650 25×25 | `<ref>/components/chat-workspace/composer/InputArea.vue:2460`–`:2467` |
| IN-06 | （无文案） | 图标按钮（模式选择器） | ChevronsRight / Target | 常显 | E/N/R/P/D/T | E ≈462,461 28×28（隐藏标签 463,456）；D 438,650 25×25 | `<ref>/components/chat-workspace/composer/AgentModeSelector.vue:4`–`:30` |
| IN-07 | （无文案） | 图标按钮（技能选择器） | Box | 常显 | E/N/R/P/D/T | E ≈480,461 28×28（隐藏标签 491,455）；D 463,650 25×25（标签 463,653） | `<ref>/components/chat-workspace/composer/ComposerSkillsSelector.vue:9`–`:18` |
| IN-08 | 高（推理强度短标签） | 按钮（四格条 + 文本） | 四格条（自绘） | 当前模型有推理强度选项 | E/N/R/P/D/T | E 802,455 25×18；D 831,653 25×18（`高`） | `<ref>/components/chat-workspace/model-selector/EffortControl.vue:6`–`:32` |
| IN-09 | `{模型名}` | 按钮（模型选择器） | ProviderIcon | 模型选择器可打开 | E/N/R/P/D/T | E 837,451 110×25；D 866,650 110×25（文本 893,653） | `<ref>/components/chat-workspace/model-selector/ModelSelector.vue:22`–`:48` |
| IN-10 | （无文案） | 图标按钮（ZDR 徽标位） | ShieldCheck 图标（ZdrBadge） | 模型 `zdr === true` ∧ 非纯图标模式 | —（需数据） | — | `<ref>/components/chat-workspace/model-selector/ModelSelector.vue:41`–`:44` |
| IN-11 | （无文案） | 图标 | Zap | 模型服务档 = priority | —（需数据） | — | `<ref>/components/chat-workspace/model-selector/ModelSelector.vue:45`–`:49` |
| IN-12 | 发送消息... | 图标按钮（发送） | ArrowUp | 发送动作为「发送」 | E/N/R/D | E 951,451 25×25；D 979,650 25×25（`发送消息...`） | `<ref>/components/chat-workspace/composer/ConversationComposer.vue:86`–`:90` |
| IN-13 | 停止回复 | 图标按钮（停止） | Square | 发送动作为「停止」 | P/T | P 979,650 25×25；T 979,650 25×25（`停止回复`） | `<ref>/components/chat-workspace/composer/ConversationComposer.vue:88`–`:90` |
| IN-14 | Esc | 键帽 | — | 停止动作 ∧ 停止确认已武装 | —（需交互） | — | `<ref>/components/chat-workspace/composer/ConversationComposer.vue:84`–`:86` |
| IN-15 | `{发送不可用原因}` | 文本 | — | 不允许发送 ∧ 不允许排队 ∧ 发送态已确定 | —（需数据） | — | `<ref>/components/chat-workspace/composer/InputArea.vue:2309`–`:2310` |
| IN-16 | 重试 | 文字按钮 | — | IN-15 ∧ 有活动会话 ∧ 发送能力未知 | —（需数据） | — | `<ref>/components/chat-workspace/composer/InputArea.vue:2311` |

元素数：16

### 4.2 附件菜单、紧凑态与上抬区

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| IN-17 | 添加附件 | 菜单项 | Paperclip | 紧凑浮层 ∧ 展开 `+` 菜单 | —（需交互） | — | `<ref>/components/chat-workspace/composer/InputArea.vue:2371`–`:2374` |
| IN-18 | Agent / 目标 | 菜单项（子菜单容器） | ChevronsRight / Target | IN-17 所在菜单 ∧ 悬停该子菜单 | —（需交互） | — | `<ref>/components/chat-workspace/composer/InputArea.vue:2375`–`:2406` |
| IN-19 | Agent | 菜单项 | ChevronsRight | IN-18 ∧ 子菜单展开 | —（需交互） | — | `<ref>/components/chat-workspace/composer/InputArea.vue:2383`–`:2391` |
| IN-20 | 目标 | 菜单项 | Target | IN-18 ∧ 子菜单展开（可用性见「检查中/不可用」提示） | —（需交互） | — | `<ref>/components/chat-workspace/composer/InputArea.vue:2392`–`:2404` |
| IN-21 | 技能 | 菜单项 | Box | IN-17 所在菜单 | —（需交互） | — | `<ref>/components/chat-workspace/composer/InputArea.vue:2407`–`:2409` |
| IN-22 | （无文案） | 图标按钮（紧凑态 `+`） | Plus | 紧凑浮层 | —（需交互） | — | `<ref>/components/chat-workspace/composer/InputArea.vue:2358`–`:2368` |
| IN-23 | 正在编辑排队消息 | 文本 | — | 编辑排队消息 ∧ 非紧凑浮层 | —（需数据） | — | `<ref>/components/chat-workspace/composer/InputArea.vue:2414`–`:2420` |
| IN-24 | （无文案） | 占位（应用快照展开高度） | — | 应用快照区域高度已确定 | —（需数据） | — | `<ref>/components/chat-workspace/composer/InputArea.vue:2421`–`:2426` |
| IN-25 | （无文案） | 图片附件廊 | — | 有图片附件 | —（需数据） | — | `<ref>/components/chat-workspace/composer/InputArea.vue:2427`–`:2432` |
| IN-26 | （无文案） | chip 架 | — | 有引用或 chip 附件 | —（需数据） | — | `<ref>/components/chat-workspace/composer/InputArea.vue:2433`–`:2453` |

元素数：10

### 4.3 底部栈（会话态，`ChatWorkspace`）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| IN-27 | （无文案） | 容器（底部栈） | — | 会话 ∧ 非全宽面板聊天 | A/P/D/T | A ≈356,480 674×210 | `<ref>/components/chat-workspace/ChatWorkspace.vue:6671`–`:6743` |
| IN-28 | （活动条：目标） | chip | — | 会话 ∧ 目标进行中 | —（需数据） | — | `<ref>/components/chat-workspace/composer/activity/GoalActivityChip.vue` |
| IN-29 | （活动条：待办） | chip | — | 会话 ∧ 有待办项 | —（需数据） | — | `<ref>/components/chat-workspace/composer/activity/TodoActivityChip.vue` |
| IN-30 | （活动条：子代理） | chip | — | 会话 ∧ 有运行中子代理 | —（需数据） | — | `<ref>/components/chat-workspace/composer/activity/AgentActivityChip.vue` |
| IN-31 | （活动条：分支） | chip | — | 会话 ∧ 有分支项 | —（需数据） | — | `<ref>/components/chat-workspace/composer/activity/BranchActivityChip.vue` |
| IN-32 | （活动条：唤醒） | chip | — | 会话 ∧ 有唤醒项 ∨ 面板打开 | —（需数据） | — | `<ref>/components/chat-workspace/composer/activity/TaskActivityChip.vue:21` |
| IN-33 | （活动条：后台进程） | chip | — | 会话 ∧ 有进程 ∨ 面板打开 | —（需数据） | — | `<ref>/components/chat-workspace/composer/activity/TaskActivityChip.vue:29` |
| IN-34 | （活动条：文件改动） | chip | — | 会话 ∧ 有文件改动摘要 | T | T 371,533 129×28（文本 `1 个文件改动`） | `<ref>/components/chat-workspace/composer/activity/FileChangesActivityChip.vue` |
| IN-35 | （排队消息面板） | 容器 | — | 会话 ∧ 排队消息数 > 0 | —（需数据） | — | `<ref>/components/chat-workspace/composer/FollowUpMessagesView.vue:2`–`:8` |
| IN-36 | `{count} 条消息`（计数文案） | 文本 | List | IN-35 | —（需数据） | — | `<ref>/components/chat-workspace/composer/FollowUpMessagesView.vue:6` |
| IN-37 | `{状态}` + `{消息文本}` | 行（转向中） | Loader2 | IN-35 ∧ 该条为转向中 | —（需数据） | — | `<ref>/components/chat-workspace/composer/FollowUpMessagesView.vue:12`–`:19` |
| IN-38 | `{消息文本}` | 行（待发） | — | IN-35 ∧ 该条待发 | —（需数据） | — | `<ref>/components/chat-workspace/composer/FollowUpMessagesView.vue:20`–`:26` |
| IN-39 | （无文案） | 拖拽把手 | GripVertical | IN-38 ∧ 未锁序 ∧ 悬停该行 | —（需数据） | — | `<ref>/components/chat-workspace/composer/FollowUpMessagesView.vue:28`–`:33` |
| IN-40 | 编辑 | 图标按钮 | Pencil | IN-38 ∧ 允许编辑 ∧ 悬停该行 | —（需数据） | — | `<ref>/components/chat-workspace/composer/FollowUpMessagesView.vue:40`–`:45` |
| IN-41 | 移除 | 图标按钮 | Trash2 | IN-38 ∧ 悬停该行 | —（需数据） | — | `<ref>/components/chat-workspace/composer/FollowUpMessagesView.vue:46`–`:51` |
| IN-42 | 转向 | 文字按钮 | — | IN-38 ∧ 运行中 | —（需数据） | — | `<ref>/components/chat-workspace/composer/FollowUpMessagesView.vue:52`–`:55` |
| IN-43 | （无文案） | 权限请求面板 | — | 会话 ∧ 有权限交互 | A | A ≈386,519 615×141 | `<ref>/components/chat-workspace/interactions/InlinePermissionPrompt.vue:1`–`:4` |
| IN-44 | （无文案） | 澄清请求面板 | — | 会话 ∧ 无权限交互 ∧ 有澄清交互 | —（需数据） | — | `<ref>/components/chat-workspace/interactions/InlineElicitationPrompt.vue` |
| IN-45 | （无文案） | 面板容器（Shell 运行） | Terminal | 会话 ∧ 有活动 shell 运行 | —（需数据） | — | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:2`–`:10` |
| IN-46 | `{命令}` | 等宽文本 | — | IN-45 | —（需数据） | — | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:7` |
| IN-47 | `{模式}` | 标签 | — | IN-45 | —（需数据） | — | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:8`–`:10` |
| IN-48 | `{状态}` | 状态文本 | Loader2 | IN-45 | —（需数据） | — | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:11`–`:14` |
| IN-49 | 停止 | 按钮 | Square | IN-45 ∧ 运行中 | —（需数据） | — | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:15`–`:23` |
| IN-50 | 关闭 | 按钮 | X | IN-45 ∧ 非运行中 | —（需数据） | — | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:24`–`:31` |
| IN-51 | `{输出}` | 等宽块 | — | IN-45 ∧ 有输出 | —（需数据） | — | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:34`–`:36` |
| IN-52 | `{跳过数}` / `{输出已截断}` / `{进程退出码}` / `{错误}` / `{原因}` / `{录制结果}` | 脚注文本（6 条） | — | IN-45 ∧ 各自条件 | —（需数据） | — | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:37`–`:59` |

元素数：26

### 4.4 选择器浮层（模型 / 推理强度 / 技能 / 模式）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| IN-53 | 模型选择浮层 | 浮层容器 | — | 模型选择器可打开 ∧ 浮层打开 | C8 | C8 ≈276,340 305×300 | `<ref>/components/chat-workspace/model-selector/ModelSelector.vue:66`–`:80` |
| IN-54 | 标签页（对话 / 子代理策略 / 外部 Agent） | 文本标签（多于一个时才显示） | — | IN-53 ∧ 子代理页 ∨ 外部 Agent 页可用 | C8 | C8 276,345 25×23 / 315,345 37×23 / 365,345 61×23 | `<ref>/components/chat-workspace/model-selector/ModelSelector.vue:82`–`:105` |
| IN-55 | 搜索模型... | 输入框 | search | IN-53 | C8 | C8 ≈290,375 270×24（`搜索模型...`） | `<ref>/components/chat-workspace/model-selector/ModelSelector.vue:115` 起 |
| IN-56 | 未配置 / 模型加载中… / 加载失败 · 重试 / 暂无可用模型 | 文本（4 条） | — | IN-53 ∧ 各自状态 | —（需交互） | — | `<ref>/components/chat-workspace/model-selector/ModelSelector.vue:121` 起 |
| IN-57 | 思考强度 | 浮层（滑杆） | — | 点开推理强度按钮 | C7 | C7 404,575 49×12（`思考强度`）+ 滑杆 397,596 187×34 | `<ref>/components/chat-workspace/model-selector/EffortControl.vue:34`–`:47`；`SmartnessSlider.vue` |
| IN-58 | Fast 模式 | 浮层（选择） | — | 模型支持服务档 | —（需交互） | — | `<ref>/components/chat-workspace/model-selector/ServiceTierControl.vue` |
| IN-59 | 搜索技能 | 输入框 | search（ThemeIcon） | 技能选择器浮层打开 | C9 | C9 ≈155,425 260×24（`搜索技能...`） | `<ref>/components/chat-workspace/composer/ComposerSkillsSelector.vue:25`–`:26` |
| IN-60 | 加载中 | 文本 | — | 技能浮层打开 ∧ 加载中 | —（需交互） | — | `<ref>/components/chat-workspace/composer/ComposerSkillsSelector.vue:29` |
| IN-61 | `{错误}` | 文本（错误色） | — | 技能浮层打开 ∧ 出错 | —（需交互） | — | `<ref>/components/chat-workspace/composer/ComposerSkillsSelector.vue:30` |
| IN-62 | `{无技能文案}` | 文本 | — | 技能浮层打开 ∧ 列表为空 | —（需交互） | — | `<ref>/components/chat-workspace/composer/ComposerSkillsSelector.vue:31` |
| IN-63 | `{技能名}` + `{技能描述}` | 菜单项 | — | 技能浮层打开 ∧ 每个技能 | C9 | C9 94,438 66×18（首行 agents-sdk，含描述行） | `<ref>/components/chat-workspace/composer/ComposerSkillsSelector.vue:33`–`:37` |
| IN-64 | Agent / 目标 | 菜单项 | ChevronsRight / Target | 模式选择器菜单打开 | C10 | C10 94,590 34×18（`Agent`） | `<ref>/components/chat-workspace/composer/AgentModeSelector.vue:34`–`:63` |
| IN-65 | `{检查中}` / `{不可用}` | 菜单项提示 | Loader2（检查中） | IN-64 的「目标」项 ∧ 可用性未定 | C10 | C10 59,613 159×28（`此会话暂不可用 Goal 模式…`） | `<ref>/components/chat-workspace/composer/AgentModeSelector.vue:52`–`:62` |

元素数：13

### 4.5 审批卡（PermissionPrompt，替换输入区）

A 态（`session` / `session-approval`）里 `InlinePermissionPrompt` 占据输入区位置，`InputArea` 不渲染——这是 A 态没有输入框、发送键的原因。

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| IN-66 | （无文案） | 卡片容器 | — | 会话 ∧ 有工具审批交互 | A | A ≈386,519 615×141 | `<ref>/components/chat-workspace/interactions/PermissionPrompt.vue:4`–`:13` |
| IN-67 | `DimAgent 想要：{action}`（此处 `执行`） | 标题 | — | IN-66 | A | A 386,519 615×21 | `<ref>/components/chat-workspace/interactions/PermissionPrompt.vue:14`；文案 `<ref>/i18n/zh-CN.ts:1514` |
| IN-68 | `{成员}` / `{主分支}` / `审批 {approvalId}` + 重试 | 上下文块 | — | IN-66 ∧ 成员审批请求 | —（需数据） | — | `<ref>/components/chat-workspace/interactions/PermissionPrompt.vue:17`–`:49` |
| IN-69 | （无文案） | 审批输入面板（按工具类型选） | — | IN-66 ∧ 有工具展示信息 | A | A ≈387,545 613×80 | `<ref>/components/chat-workspace/interactions/PermissionPrompt.vue:51`–`:57`；`approval-panels/*.vue` |
| IN-70 | 输入 | 面板小标题 | — | 命令类审批面板 | A | A 387,552 613×27 | `<ref>/components/chat-workspace/interactions/approval-panels/CommandApprovalPanel.vue:8`–`:10`；文案 `<ref>/i18n/zh-CN.ts:1329`（`timelineTool.input`） |
| IN-71 | `$ {命令}` | 等宽行 | — | 同上 ∧ 有命令 | A | A 397,579 592×18（`ls -la`，`$` 397,581 6×14） | `<ref>/components/chat-workspace/interactions/approval-panels/CommandApprovalPanel.vue:13`–`:14` |
| IN-72 | `cwd: {路径}` | 等宽行 | — | 同上 ∧ 有工作目录 | A | A 397,596 592×18 | `<ref>/components/chat-workspace/interactions/approval-panels/CommandApprovalPanel.vue:15` |
| IN-73 | `timeoutMs: {值}` / `runInBackground: {值}` | 等宽行（2 条） | — | 同上 ∧ 有超时/后台标记 | C39 | C39 42,596 636×18（`timeoutMs: 60 s`） | `<ref>/components/chat-workspace/interactions/approval-panels/CommandApprovalPanel.vue:16`–`:17` |
| IN-74 | 拒绝 / 仅这次 / 始终允许此命令 / 始终允许网络访问 / 始终允许文件访问 / 允许此会话期间访问 / 不允许 | 选项按钮组（按请求与权限级出现） | — | IN-66 | A（`拒绝` `仅这次` `始终允许此命令`） | A 拒绝 772,632 48×28；仅这次 827,632 58×28；始终允许此命令 891,632 109×28 | `<ref>/components/chat-workspace/interactions/PermissionPrompt.vue:87`–`:95`；文案 `<ref>/i18n/zh-CN.ts:1522`、`:1534`–`:1546` |
| IN-75 | 网络访问已拦截 / 文件读取已拦截 / 文件写入已拦截 / 文件执行已拦截 + `{资源}` + `{工具} 需要…` | 沙箱拦截卡 | — | 审批请求来自沙箱范围升级 | —（需数据） | — | `<ref>/components/chat-workspace/interactions/PermissionPrompt.vue:97`–`:117` |
| IN-76 | `{子代理模型审批标题}` / `{任务} · {标题}` / `{模型}` / `{供应商}` / `{更贵标记}` / `{回退说明}` | 子代理模型审批卡 | ProviderIcon | 审批请求为子代理换模型 | —（需数据） | — | `<ref>/components/chat-workspace/interactions/PermissionPrompt.vue:118`–`:158` |
| IN-77 | `{需要你决定}` + `{原因}` / `{自动审核失败}` | 提示块（2 条） | — | 审批请求带 handoff（不确定 / 审核失败） | —（需数据） | — | `<ref>/components/chat-workspace/interactions/PermissionPrompt.vue:59`–`:76` |
| IN-78 | `{错误}` | 错误行 | — | 审批提交失败 | —（需数据） | — | `<ref>/components/chat-workspace/interactions/PermissionPrompt.vue:78`–`:80` |

元素数：13

**区 4 元素数：78**

---

## 5. 区：状态条（StatusBar）

`StatusBar` 只在会话工作区渲染（`ChatWorkspace.vue:6738`–`:6741`），故 E/N/R 无、A/P/D/T 有。屏上可见的只有三件：容器、权限选择器、上下文用量表（其余是点击后才出的浮层）。空态里的权限选择器挂在空态控制行（见第 6 节 `NT-14`），属同一组件、不同挂载点。

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| SB-01 | （无文案） | 容器（状态条） | — | 会话 ∧ 非紧凑浮层 ∧ 非全宽面板聊天 | A/P/D/T | A ≈356,686 674×30 | `<ref>/components/chat-workspace/StatusBar.vue:3`–`:5` |
| SB-02 | 默认权限 / 智能权限 / 完全访问权限 | 按钮（权限选择器） | 盾形图标（随权限变） | 会话（工具栏左） | A/P/D/T | A 378,693 77×21（文本 404,696 42×14 `默认权限`） | `<ref>/components/chat-workspace/ToolPermissionSelector.vue:3`–`:6` |
| SB-03 | 默认权限 / 智能权限 / 完全访问权限 | 菜单项（3 条） | 各自图标 | 权限菜单打开 | C5 | C5 55,633 124×18（`完全访问权限`）+ 同菜单 `默认权限` | `<ref>/components/chat-workspace/ToolPermissionSelector.vue:10`–`:28` |
| SB-04 | 确认开启完全放开？ | 对话框标题 | AlertTriangle | 选择「完全访问权限」 | C6 | C6 299,263 432×25 | `<ref>/components/chat-workspace/ToolPermissionSelector.vue:40`–`:44` |
| SB-05 | 开启后，AI 将减少确认步骤… | 对话框描述 | — | SB-04 | C6 | C6 299,294 432×74 | `<ref>/components/chat-workspace/ToolPermissionSelector.vue:45`–`:47` |
| SB-06 | 我已了解风险，并愿意继续 | 复选框 | — | SB-04 | C6 | C6 299,392 14×14 | `<ref>/components/chat-workspace/ToolPermissionSelector.vue:49`–`:54` |
| SB-07 | 取消 / 确认开启 | 对话框按钮（2 条） | — | SB-04 | C6 | C6 取消 593,431 55×32；确认开启 654,431 77×32 | `<ref>/components/chat-workspace/ToolPermissionSelector.vue:56`–`:63` |
| SB-08 | Agent 配置 | 按钮（配置选择器） | — | 会话 ∧ 该会话提供多套 Agent 配置 | —（需数据） | — | `<ref>/components/chat-workspace/StatusBar.vue:12`–`:21` |
| SB-09 | `{当前分支}` | 按钮（分支选择器） | GitBranch | 会话 ∧ 有分支 ∧ 非办公模式 | C36 | C36 201,498 24×14（`main`） | `<ref>/components/chat-workspace/StatusBar.vue:28`–`:35` |
| SB-10 | 搜索分支 | 输入框 | search（ThemeIcon） | 分支菜单打开 | C36 | C36 ≈160,533 177×24（`搜索分支...`） | `<ref>/components/chat-workspace/StatusBar.vue:38`–`:46` |
| SB-11 | 分支 | 分组标题 | — | 分支菜单打开 | —（需交互） | — | `<ref>/components/chat-workspace/StatusBar.vue:50`–`:52` |
| SB-12 | `{分支名}` | 菜单项 | GitBranch + Check | 分支菜单打开 ∧ 每个分支 | C36 | C36 212,574 177×18（`main`） | `<ref>/components/chat-workspace/StatusBar.vue:55`–`:71` |
| SB-13 | 创建新分支 | 菜单项 | Plus | 分支菜单打开 | —（需交互） | — | `<ref>/components/chat-workspace/StatusBar.vue:75`–`:78` |
| SB-14 | （无文案） | 用量表（上下文用量触发器） | 环形/条形（自绘） | 会话 ∧ 上下文用量可见 | A/P/D/T | A 929,693 79×21（文本 977,696 25×15 `—`）；D 同框，文本 `~6%` | `<ref>/components/chat-workspace/ContextUsageMeter.vue` |
| SB-15 | 本轮上下文 / 会话消耗：{tokens} tokens / 缓存命中 {pct}% / 速率：{rate} tok/s | 浮层文本（4 条） | — | 悬停或点开用量表 | C34 | C34 455,500 53×14（`本轮上下文`）+ 明细 455,540（`~19.7K / 1.0M`、`缓存命中 98%`） | `<ref>/i18n/zh-CN.ts:1193`–`:1196`；`ContextUsageMeter.vue` |
| SB-16 | 计划 | 浮层分组标题 | — | 用量浮层 ∧ 计划用量可见 | C34 | C34 455,600 42×14（`套餐用量`） | `<ref>/components/chat-workspace/StatusBar.vue:91`–`:94` |
| SB-17 | 账户设置 | 浮层文字按钮 | ArrowUpRight | 同上 ∧ 非弹窗窗口 | C34 | C34 671,597 41×20（`账户 ↗`） | `<ref>/components/chat-workspace/StatusBar.vue:96`–`:105` |
| SB-18 | `{用量错误}` | 文本（错误色） | — | 用量浮层 ∧ 有错误 | —（需交互） | — | `<ref>/components/chat-workspace/StatusBar.vue:106`–`:108` |
| SB-19 | `{无限额度说明}` | 文本 | — | 用量浮层 ∧ 无限额 ∧ 无明细行 | —（需交互） | — | `<ref>/components/chat-workspace/StatusBar.vue:110`–`:115` |
| SB-20 | `{加载中 / 空文案}` | 文本 | — | 用量浮层 ∧ 无明细行 ∧ 无错误 | —（需交互） | — | `<ref>/components/chat-workspace/StatusBar.vue:116`–`:118` |
| SB-21 | `{行标签}` + `{pct}%` + `{重置时间}` + 进度条 | 明细行 | — | 用量浮层 ∧ 有明细行 | C34 | C34 455,624 257×13（`月度用量 39%`）+ 进度条 | `<ref>/components/chat-workspace/StatusBar.vue:120`–`:143` |

元素数：21

**区 5 元素数：21**

---

## 6. 空态（NewThread，单独成节）

空态与右栏同时存在时（R/Q），输入区宽度压缩但元素集合不变；坐标随卡片宽度变化。

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| NT-01 | （无文案） | 品牌图形 | BrandMark（品牌绿） | 空态 | E/N/R | E ≈651,247 56×56 | `<ref>/components/chat-workspace/NewThread.vue:298` |
| NT-02 | 开始构建 | 标题（h1） | — | 空态 | E/N/R | E 651,313 84×28；R 160,313 84×28 | `<ref>/components/chat-workspace/NewThread.vue:299` |
| NT-03 | 加载中 | 状态文本 | — | 空态 ∧ 会话目录初次选择进行中 | —（需数据） | — | `<ref>/components/chat-workspace/NewThread.vue:300` |
| NT-04 | （无文案） | 容器（输入块 + 阴影） | — | 空态 | E/N/R | E ≈396,372 582×110 | `<ref>/components/chat-workspace/NewThread.vue:306`–`:308` |
| NT-05 | （无文案） | 容器（控制行，圆角底 + 灰底） | — | 空态 | E/N/R | E ≈396,482 582×40 | `<ref>/components/chat-workspace/NewThread.vue:314`–`:325` |
| NT-06 | `{工作区名}` / 选择工作区... | 按钮（工作区选择器） | ThemeIcon:workspace / folder | 空态 | E/N/R | E 433,498 32×14；R 77,498 32×14（文本均为 `工作区`） | `<ref>/components/chat-workspace/NewThread.vue:326`–`:343` |
| NT-07 | 搜索工作区... | 输入框 | search（ThemeIcon） | 工作区下拉打开 | C35 | C35 ≈78,216 271×26（`搜索工作区...`） | `<ref>/components/chat-workspace/NewThread.vue:346`–`:353` |
| NT-08 | `{内置工作区名}` | 菜单项 | ThemeIcon:workspace + 单选圈 | 工作区下拉 ∧ 每个内置工作区 | C35 | C35 134,258 37×18（内置「工作区」行） | `<ref>/components/chat-workspace/NewThread.vue:360`–`:380` |
| NT-09 | `{工作区名}` + `{路径}` | 菜单项 | 单选圈 | 工作区下拉 ∧ 每个普通工作区 | C35 | C35 ≈78,290 271×40（`year` + 路径行） | `<ref>/components/chat-workspace/NewThread.vue:385`–`:412` |
| NT-10 | 上一页 / 显示更多 | 文字按钮（2 条） | — | 工作区下拉 ∧ 各自分页条件 | —（需交互） | — | `<ref>/components/chat-workspace/NewThread.vue:419`–`:420` |
| NT-11 | 添加新工作区 / 正在打开... | 菜单项 | FolderPlus | 工作区下拉 | C35 | C35 78,458 271×28 | `<ref>/components/chat-workspace/NewThread.vue:425`–`:431` |
| NT-12 | 选择分支... / `{分支名}` | 按钮（分支选择器） | GitBranchIcon | 空态 ∧ 支持分支选择 ∧ 非办公模式 | C36 | C36 201,498 24×14（`main`） | `<ref>/components/chat-workspace/NewThread.vue:434`–`:452` |
| NT-13 | 搜索分支... / 无分支 / `{分支名}` | 输入框 + 菜单项 | search + GitBranchIcon + Check | 分支下拉打开 | C36 | C36 ≈160,533 177×24（`搜索分支...`）/ 212,574 177×18（`main`） | `<ref>/components/chat-workspace/NewThread.vue:457`–`:499` |
| NT-14 | 默认权限 / 智能权限 / 完全访问权限 | 按钮（权限选择器） | 盾形图标 | 空态（控制行第三项） | E/N/R | E 521,498 42×14；R 165,498 42×14 | `<ref>/components/chat-workspace/NewThread.vue:499`（组件 `ToolPermissionSelector.vue`） |
| NT-15 | 总结 PDF / 改写 Word 文档 / 做一版 PPT / 整理成表格 / 帮我调研 / 翻译 | 推荐卡（6 张） | 各自图标 | 空态 ∧ 办公模式 | —（需平台/模式） | — | `<ref>/components/chat-workspace/NewThread.vue:505`–`:521` |

元素数：15

---

## 7. 运行时出现、但不属五区的区域

这些区域在采集状态里真实可见，模板也有对应节点；它们不是五区元素，单列以避免与第 1–6 节重复计数。

### 7.1 右栏（R、Q）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| OV-01 | （无文案） | 容器（右栏外壳） | — | 右栏打开 | R/Q | R ≈400,0 630×725 | `<ref>/components/chat-workspace/ChatWorkspace.vue:6753`–`:6770` |
| OV-02 | 拖拽调整面板宽度 | 拖拽手柄 | — | 右栏打开 ∧ 非全宽替换布局 | R | R 405,5 4×716 | `<ref>/components/chat-workspace/ChatWorkspace.vue:6771`–`:6776` |
| OV-03 | 文件 | 标签（页签） | Folders | 右栏标签含「文件」页 | R | R 462,19 25×18（图标 + 文本整块 ≈412,10 78×25） | `<ref>/components/right-panel/RightPanelBar.vue:65`–`:70` |
| OV-04 | （无文案） | 关闭页签按钮 | X | 页签可关闭 ∧ 悬停该页签 | C21（悬停可见） | C21 733,21 14×14（另有 905,21 / 1059,21） | `<ref>/components/right-panel/RightPanelBar.vue:100`–`:112` |
| OV-05 | `{⌘n}` | 快捷键徽标 | — | 修饰键按住 ∧ 右栏打开 ∧ 页签 > 1 | C20 | C20 ≈788,21 24×16（按住 ⌘ 时页签显示 `⌘2`） | `<ref>/components/right-panel/RightPanelBar.vue:93`–`:98` |
| OV-06 | （无文案） | 图标按钮（添加页签） | Plus | 有可添加的页签类型 | R | R ≈953,16 28×28 | `<ref>/components/right-panel/RightPanelBar.vue:381`–`:397` |
| OV-07 | 文件 / 改动 / 浏览器 / 终端 / 侧边对话 / 界面扩展 | 菜单项（按可用性出现） | Folders / GitCompare / Globe / SquareTerminal / MessagesSquare / Blocks | OV-06 菜单打开 | C22 | C22 终端 841,77 145×28；侧边对话 841,105 145×28 | `<ref>/components/right-panel/RightPanelBar.vue:133`–`:152`、`:399`–`:410` |
| OV-08 | 展开面板 | 图标按钮 | Maximize2 | 非紧凑布局 ∧ 面板未展开 | R | R 994,16 25×25 | `<ref>/components/right-panel/RightPanelBar.vue:419`–`:428` |
| OV-09 | 恢复面板宽度 | 图标按钮 | Minimize2 | 面板已展开 | —（需交互） | — | `<ref>/components/right-panel/RightPanelBar.vue:419`–`:428` |
| OV-10 | （无文案） | 返回按钮 | ArrowLeft | 全宽替换布局 | C42 | C42 ≈116,33 25×25；P5 127,16 25×25（aria「返回」） | `<ref>/components/right-panel/RightPanelBar.vue:29`–`:38` |
| OV-11 | 筛选文件... | 输入框 | Search | 文件页 ∨ 改动页激活 | R | R ≈400,62 600×24（占位文字左上 ≈425,69） | `<ref>/components/context-panel/ContextPanel.vue:539`–`:553` |
| OV-12 | `{目录名}` | 树节点行（可折叠） | ChevronRight + 文件夹图标 | 文件页激活 ∧ 每个目录 | R | R 462,100 553×18（首项） | `<ref>/components/context-panel/FileTreeNode.vue` |
| OV-13 | `{文件名}` | 树节点行 | 文件图标 | 文件页激活 ∧ 每个文件 | R | R 461,1025 555×18（首项） | `<ref>/components/context-panel/FileTreeNode.vue` |
| OV-14 | （无文案） | 状态块 | — | 文件页 ∧ 首次加载 | —（需数据） | — | `<ref>/components/context-panel/FilesView.vue:3`–`:5` |
| OV-15 | 正在搜索工作区 / 搜索失败 + 重试 / 搜索不完整 / 结果条数上限 / 无匹配文件 + 说明 | 文本（6 条） | FileIcon | 文件页 ∧ 搜索模式 ∧ 各自状态 | P15 | P15 484,189 61×18（`无匹配文件`）；P16 5,95 1021×18（`仅显示前 200 个匹配项`） | `<ref>/components/context-panel/FilesView.vue:8`–`:60` |
| OV-16 | 无权限提示 + 重试 | 警告条 | AlertTriangle | 目录读取被拒 | —（需数据） | — | `<ref>/components/context-panel/ContextPanel.vue:556`–`:576` |
| OV-17 | `{范围}` | 下拉（改动范围） | — | 改动页激活 | P1 | P1 742,64 53×14（`未提交改动` 下拉）；P10 27,100 / 27,128 / 27,156（三个范围项） | `<ref>/components/context-panel/ContextPanel.vue:440`–`:461` |
| OV-18 | 提交或 Push | 按钮（菜单） | CloudUpload + ChevronDown | 改动页 ∧ 可提交 | P1 | P1 ≈745,41 84×25（`提交或推送`） | `<ref>/components/context-panel/ContextPanel.vue:465`–`:478` |
| OV-19 | （无文案） | 图标按钮（刷新） | RefreshCw | 改动页激活 | P1 | P1 ≈920,64 25×25（刷新） | `<ref>/components/context-panel/ContextPanel.vue:523`–`:531` |

元素数：19

### 7.2 「搜索聊天」弹窗（Q、S、K）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| OV-20 | （无文案） | 对话框容器（遮罩 + 弹窗） | — | 全局搜索打开 | Q/S/K | Q 291,293 448×139 | `<ref>/components/sidebar/ChatSearchDialog.vue:185`–`:190` |
| OV-21 | 搜索聊天 | 标题（视觉隐藏） | — | OV-20 | Q/S/K | Q 291,293（宽度采集失真） | `<ref>/components/sidebar/ChatSearchDialog.vue:192` |
| OV-22 | 搜索并打开已有聊天 | 描述（视觉隐藏） | — | OV-20 | Q/S/K | Q 291,316（宽度采集失真） | `<ref>/components/sidebar/ChatSearchDialog.vue:193` |
| OV-23 | 搜索聊天（占位） | 输入框 | search（ThemeIcon） | OV-20 | Q/S/K | Q 331,294 394×48 | `<ref>/components/sidebar/ChatSearchDialog.vue:196`–`:205` |
| OV-24 | 未找到聊天 | 文本 | — | OV-20 ∧ 无结果 | Q/S/K | Q 299,350 432×74 | `<ref>/components/sidebar/ChatSearchDialog.vue:255`–`:261` |
| OV-25 | 加载中 | 状态行 | Loader2 | OV-20 ∧ 结果分页加载中 | —（需数据） | — | `<ref>/components/sidebar/ChatSearchDialog.vue:209` |
| OV-26 | 上一页 / 显示更多 | 文字按钮（2 条） | — | OV-20 ∧ 各自分页条件 | —（需数据） | — | `<ref>/components/sidebar/ChatSearchDialog.vue:210`–`:211` |
| OV-27 | 置顶聊天 / 最近聊天 | 分组标题 | — | OV-20 ∧ 有结果 | —（需数据） | — | `<ref>/components/sidebar/ChatSearchDialog.vue:105`–`:112`、`:217`–`:220` |
| OV-28 | `{会话标题}` + `{工作区名}` + `{快捷键}` | 结果行 | 状态图标（错误/加载/未读/置顶/普通） | OV-20 ∧ 每条结果 | —（需数据） | — | `<ref>/components/sidebar/ChatSearchDialog.vue:221`–`:252` |

元素数：9

### 7.3 定时任务页（S）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| PG-01 | 展开侧边栏 | 图标按钮 | PanelLeftOpen | 路由 = 定时任务 ∧ 左栏收起 | S | S 99,14 28×28 | `<ref>/views/ScheduledView.vue`（页头） |
| PG-02 | 筛选任务 | 分组（role=group） | — | 路由 = 定时任务 | S | S 127,14 174×28 | `<ref>/components/scheduled/ScheduledFilterTabs.vue` |
| PG-03 | 全部 | 分段按钮 | — | PG-02 ∧ 当前筛选非「全部」时高亮切换 | S | S 130,17 47×22 | `<ref>/components/scheduled/ScheduledFilterTabs.vue` |
| PG-04 | 已启用 | 分段按钮 | — | PG-02 | S | S 177,17 60×22 | `<ref>/components/scheduled/ScheduledFilterTabs.vue` |
| PG-05 | 已暂停 | 分段按钮 | — | PG-02 | S | S 237,17 60×22 | `<ref>/components/scheduled/ScheduledFilterTabs.vue` |
| PG-06 | 创建 | 按钮 | — | 路由 = 定时任务（调度器可用时可点） | S | S 966,14 46×28 | `<ref>/views/ScheduledView.vue` |
| PG-07 | 搜索定时任务 | 输入框 | — | 路由 = 定时任务 | S | S 221,63 588×32 | `<ref>/views/ScheduledView.vue` |
| PG-08 | 定时任务暂不可用 | 空态标题 | 日历/时钟图形 | 路由 = 定时任务 ∧ 调度器插件未加载 | S | S 466,187 98×18 | `<ref>/i18n/zh-CN.ts:3307`；`ScheduledView.vue:183` |
| PG-09 | 当前 App Server 未加载 Scheduler 插件。请关闭其他客户端或会话后重启 DimAgent。 | 空态描述 | — | PG-08 | S | S 375,208 280×35 | `<ref>/i18n/zh-CN.ts:3308`；`ScheduledView.vue:186` |
| PG-10 | `{空列表标题}` / `{无匹配标题}` + 说明 | 空态 | — | 路由 = 定时任务 ∧ 调度器可用 ∧ 列表为空/无匹配 | —（需数据） | — | `<ref>/views/ScheduledView.vue:183`–`:187` |
| PG-11 | （任务行） | 列表行 | — | 路由 = 定时任务 ∧ 每条任务 | —（需数据） | — | `<ref>/components/scheduled/ScheduledJobList.vue` |

元素数：11

### 7.4 技能 / 插件页（K）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| PG-12 | 技能 | 分段按钮 | — | 路由 = 技能 | K | K 133,17 47×22 | `<ref>/views/SkillsView.vue` |
| PG-13 | 插件 | 分段按钮 | — | 路由 = 技能 | K | K 180,17 47×22 | `<ref>/views/SkillsView.vue` |
| PG-14 | 创建技能 | 按钮 | — | 路由 = 技能 | K | K 907,14 70×28 | `<ref>/views/SkillsView.vue` |
| PG-15 | 更多操作 | 图标按钮（菜单） | ChevronDown | 路由 = 技能 | K | K 984,14 28×28 | `<ref>/views/SkillsView.vue` |
| PG-16 | 技能商店 | 页签 | — | 路由 = 技能 | K | K 60,60 82×32 | `<ref>/views/SkillsView.vue` |
| PG-17 | 我的技能 | 页签 | — | 路由 = 技能 | K | K 163,63 72×28 | `<ref>/views/SkillsView.vue` |
| PG-18 | 从来源安装 | 分区标题（h2） | — | 路由 = 技能 ∧ 我的技能页签 | K | K 79,141 873×22 | `<ref>/components/skills/SkillsSourceInstallPanel.vue:430` 起 |
| PG-19 | 从绝对路径或 Git 仓库安装选中的技能。预览不会写入任何内容。 | 说明文本 | — | PG-18 | K | K 79,167 873×19 | `<ref>/components/skills/SkillsSourceInstallPanel.vue` |
| PG-20 | 来源 | 标签 | — | PG-18 | K | K 79,200 873×17 | `<ref>/components/skills/SkillsSourceInstallPanel.vue` |
| PG-21 | /绝对路径 或 https://example.com/skills | 输入框（占位） | — | PG-18 | K | K ≈80,220 858×28 | `<ref>/components/skills/SkillsSourceInstallPanel.vue` |
| PG-22 | 安装 ID（可选） | 标签 | — | PG-18 | K | K 79,265 873×17 | `<ref>/components/skills/SkillsSourceInstallPanel.vue` |
| PG-23 | 根据来源生成 | 输入框（占位） | — | PG-18 | K | K ≈80,285 858×28 | `<ref>/components/skills/SkillsSourceInstallPanel.vue` |
| PG-24 | 预览 | 按钮 | Loader2（预览中） | PG-18 ∧ 已填来源 | K | K 79,332 67×28 | `<ref>/components/skills/SkillsSourceInstallPanel.vue` |
| PG-25 | 已安装来源 | 分区标题（h2） | — | 路由 = 技能 ∧ 我的技能页签 | K | K 60,407 340×22 | `<ref>/components/skills/SkillsSourceInstallPanel.vue` |
| PG-26 | 通过 App Server 安装的来源。插件提供的技能仍由对应插件管理。 | 说明文本 | — | PG-25 | K | K 60,433 340×17 | `<ref>/components/skills/SkillsSourceInstallPanel.vue` |
| PG-27 | 刷新 | 按钮 | RefreshCw | PG-25 | K | K 902,414 69×28 | `<ref>/components/skills/SkillsSourceInstallPanel.vue:455`–`:459` |
| PG-28 | 尚无来源安装。 | 空态文本 | — | PG-25 ∧ 无来源安装 | K | K 60,464 910×89 | `<ref>/components/skills/SkillsSourceInstallPanel.vue` |
| PG-29 | （技能卡 / 搜索技能 / 空态 / 详情面板） | 列表与面板 | — | 路由 = 技能 ∧ 商店或我的技能有内容 | —（需数据） | — | `<ref>/components/skills/SkillCard.vue`、`SkillDetailPanel.vue`、`SkillsMarketplacePanel.vue` |

元素数：18

**区 7 元素数：57**

---

## 8. 合计与差异

### 8.1 计数

| 项 | 数 |
|---|---|
| 标题栏（区 1） | 40 |
| 左栏（区 2） | 112（含本轮补登记的会话行右键菜单 6 行，见 §2.7.1） |
| 时间线（区 3） | 78 |
| 输入区（区 4） | 78 |
| 状态条（区 5） | 21 |
| **五区合计** | **329** |
| 空态（第 6 节） | 15 |
| 运行时出现、不属五区（第 7 节） | 57 |
| **模板节点总数（元素级行数）** | **401** |
| 可见（屏上直接可见） | 246 |
| 可见（DOM 在，需悬停才显形） | 12 |
| 可见（DOM 在，屏上恒不可见：占位 3 / hover 动作 2 / 悬停箭头 1） | 6 |
| **可见合计** | **259**（上一版 137） |
| 不可见（本轮 70 个采集仍未见） | 142（上一版 258） |
| 运行时独有（无渲染模板） | 2（宿主系统通知的横幅与动作按钮，见 §8.4） |

分区可见数：区 1 = 38、区 2 = 76、区 3 = 42、区 4 = 29、区 5 = 15、空态 = 12、区外 = 47。

口径：
- 「模板节点」按**元素级**计（可交互控件、文案节点、状态位、容器状态），纯布局 `div` 不计。
- 「可见」= 在本文件已登记的全部采集状态（十态 + §0.2b 的 `C*`/`P*`）中至少出现一次；`（悬停可见）` 与 `（不可见）` 已单列，不计入「屏上直接可见」。
- 命中判定优先用截图：JSON 会漏采无文案图标与 svg（§0.4），也会采到悬停才显形的节点（§0.4 第 2 条）。

### 8.2 不可见原因分布（142 条）

按**主因**归类，一条只计一次；顺序 = 先判「是否只有另一种采集状态才渲染」，再判「是否必须交互」，最后判数据/平台。

| 原因 | 说明 | 区 1 | 区 2 | 区 3 | 区 4 | 区 5 | 区 6 | 区 7 | 合计 |
|---|---|---|---|---|---|---|---|---|---|
| 需数据 | 要 App Server/宿主造出状态才渲染：更新流、出错与失败重试、排队与活动条、附件与引用、分页与虚拟化、产物/改动块、沙箱与子代理审批、用量异常 | 0 | 23 | 34 | 36 | 1 | 1 | 9 | **104** |
| 需交互 | 仍未被触发的交互：打开方式加载/空态、工作区路径 tooltip、紧凑浮层附件菜单、选区引用浮层、停止确认、分支/工作区菜单的分组与新建分支、见 §8.2.1 | 2 | 8 | 2 | 13 | 5 | 1 | 1 | **32** |
| 需平台/模式 | 非 macOS（品牌位）、办公模式推荐卡、平铺视图分页 | 0 | 5 | 0 | 0 | 0 | 1 | 0 | **6** |
| 需另采 | 已全部覆盖（`TB-05`/`13`–`16`/`40` 由 `C1` 补上） | 0 | 0 | 0 | 0 | 0 | 0 | 0 | **0** |
| 合计 | | 2 | 36 | 36 | 49 | 6 | 3 | 10 | **142** |

### 8.2.1 结论：本轮未能压到 ≤20（142 条，逐桶原因）

设计门禁是「待补 ≤20」，实测只能压到 142（其中 32 条仍是交互分支，110 条是数据态分支）。原因不是没采：这 110 条**都不由「交互」触发，而由 App Server/宿主的数据态触发**，本机实例造不出来；逐桶如下。

| 桶 | 条数 | 代表元素 | 为什么造不出来 |
|---|---|---|---|
| 更新流 | 6 | `LB-04`–`LB-09` | 需要真实更新 feed（available/downloading/preparing/downloaded/error）；本机已是最新版本 |
| 出错/失败/重试/中断 | 12 | `TL-47`–`TL-49`、`TL-57`–`TL-60`、`TL-68`、`TL-09`/`TL-10`、`LB-66` | 需要模型侧失败或渲染抛错；本机跑的都是正常轮次（停止按钮点击时该轮已结束，见 §9 备注） |
| 排队与活动条 | 14 | `IN-28`–`IN-42`（除已命中） | 运行中再投递的消息直接成为下一轮（`C41` 实测），App Server 未产生排队态 |
| 附件/引用/标注/图片 | 15 | `TL-21`–`TL-26`、`IN-23`–`IN-26`、`TL-44` | 需要带图片/chip/引用的用户消息与选区引用浮层；纯文本投递不产生 |
| 产物/改动块 | 2 | `TL-14`、`TL-15` | 需要「最新一轮 + runId + 有产物/改动」；`dim-agent` 工作区不是 git 仓库，产物块由 `inventory-panels.md` 的 `blk-artifacts-*` 采集覆盖 |
| 分页/虚拟化/加载更多 | 17 | `LB-20`/`LB-21`/`LB-51`–`LB-56`/`LB-59`/`LB-63`–`LB-65`、`IN-03` 附近分页、`OV-25`/`OV-26`、`NT-10`、`PG-*` | 会话/工作区数量太少（1 条会话），分页阈值不触发 |
| 沙箱/子代理/成员审批 | 8 | `IN-68`、`IN-75`–`IN-78`、`IN-44`–`IN-52` | 需要子代理、shell 面板、沙箱拦截审批，属于特定 App Server 能力 |
| 紧凑浮层附件菜单 | 7 | `IN-17`–`IN-22`、`IN-04` | 紧凑浮层只在窄输入区出现；窄窗采集（`C42`）里输入区仍走完整版 |
| 平台/办公模式 | 6 | `LB-02`/`LB-03`、`NT-15`、`LB-63`–`LB-65` | 本机是 macOS、编程模式 |
| 其它（未登录、Pet 不可用、邀请加载、模型/技能空态、用量异常、分支分组） | 12 | `LB-85`/`LB-86`、`LB-94`/`LB-96`、`LB-99`、`IN-56`/`IN-58`、`IN-60`–`IN-62`、`SB-11`/`SB-13`、`SB-18`–`SB-20`、`TB-33`/`TB-34` | 账号已登录、有额度、有技能；这些是异常分支，需登出/断网/改账户状态 |

### 8.3 只在某个状态出现的元素

看点：左栏与标题栏在「空会话」和「会话」两套布局里都是同构复用的，所以真正「只在某态」的元素集中在**审批卡、轮末动作、多工具产物、右栏、路由页**五处。

**只在 A（停在审批上）——9 个**

- `LB-73`（等待操作徽标）
- `IN-43`、`IN-66`、`IN-67`、`IN-69`、`IN-70`、`IN-71`、`IN-72`、`IN-74`（审批卡容器、标题、输入面板、输入标签、命令、cwd、选项按钮）
- 反向证据：A 态没有输入区——`IN-01`、`IN-02`、`IN-05` 至 `IN-09`、`IN-12`、`IN-13` 全部缺席，`IN-43` 顶替其位置。

**只在 D（该轮完成）——4 个**

- `TL-61`、`TL-62`、`TL-63`、`TL-64`（轮末动作条：复制 / 分叉 / 重试 / 完成时间；`opacity-0 group-hover`，DOM 在、屏上需悬停）

**只在 T（多工具并行）——5 个**

- `TL-74`（分组内容轨道）、`TL-75`（diff 统计 `+19` / `+2 -2`）、`TL-76`（站点 favicon 批次 `E` / `H`）、`TL-77`（网页工具失败 `✗`）、`IN-34`（活动条「1 个文件改动」）
- 说明：需要同一轮里出现读/写/编辑/抓取/搜索多类工具；其余采集只跑了一条 `ls`。

**只在 A/P/T（运行中）——1 个**：`LB-67`（会话行运行指示）
**只在 P/T（运行中且有输入区）——2 个**：`TL-70`（`正在处理 · {时长}`）、`IN-13`（停止回复；A 态被审批卡替换、D 态变回发送键）
**只在 D/T——1 个**：`TL-53`（markdown 正文）
**只在 E/N——4 个**：`TB-03`（空态标题栏的打开上下文面板）、`LB-42`、`LB-43`（工作区行内动作，屏上不可见）、`LB-58`（列表底部「显示更多」——会话态里工作区已全部加载）

**只在空会话态（E/N/R）——7 个**：`TL-01`、`NT-01`、`NT-02`、`NT-04`、`NT-05`、`NT-06`、`NT-14`
**只在 R/Q——11 个**：`TB-01`、`TB-02`、`TB-04`、`OV-01`、`OV-02`、`OV-03`、`OV-06`、`OV-08`、`OV-11`、`OV-12`、`OV-13`
**跨 Q/S/K——5 个**：`OV-20` 至 `OV-24`（搜索聊天弹窗）
**只在 S——9 个**：`PG-01` 至 `PG-09`
**只在 K——17 个**：`PG-12` 至 `PG-28`

**左栏同构（E/N/A/P/D/T 六态都在）——21 个**

- `LB-01`、`LB-10` 至 `LB-15`（不含 `LB-13`）、`LB-24` 至 `LB-26`、`LB-34` 至 `LB-39`、`LB-50`、`LB-57`、`LB-83`、`LB-84`、`LB-88`、`LB-100`
- 说明：这些节点在空会话与会话态都一样；差别只在数据（`暂无会话` vs 一条会话行）。

**输入区（除 A）——8 个**

- `IN-01`、`IN-02`、`IN-05` 至 `IN-09`：E/N/R/P/D/T（空态 + 会话态，审批中不渲染）
- `IN-12`（发送键）：E/N/R/D（运行中的 P/T 被 `IN-13` 停止键顶替）

**会话态通用（A/P/D/T）——33 个**

- 标题栏：`TB-17` 至 `TB-20`、`TB-31`、`TB-32`、`TB-38`、`TB-39`
- 左栏：`LB-52`（会话行）、`LB-72`（标题）、`LB-77`（相对时间）、`LB-87`（账户错误点）
- 时间线：`TL-07`、`TL-11`、`TL-12`、`TL-13`、`TL-20`、`TL-26`、`TL-35`、`TL-37` 至 `TL-40`、`TL-50`、`TL-51`、`TL-52`、`TL-54`、`TL-72`、`TL-73`、`TL-78`
- 输入/状态条：`IN-27`（底部栈）、`SB-01`、`SB-02`（默认权限）、`SB-14`（上下文用量表）

**只在某个 `C*`/`P*` 状态出现的元素（本轮新增，挑要点）**

- 只在 `C1`（会话 + 左栏收起 + 右栏打开）：`TB-05`、`TB-13`–`TB-16`、`TB-40` —— 上一版留的「需另采」四类至此清零。
- 只在 `C24`–`C26`（消息内搜索）：`TB-06`–`TB-12`，其中 `C25` 只有 `0 个结果`，`C26` 才有上/下一个结果按钮与 `1 / 35 个结果`。
- 只在 `C14`/`C15`（置顶区）：`LB-16`、`LB-17`、`LB-18`、`LB-22`、`LB-69`、`LB-71`；`C3` 与 `C17` 分别给出「取消置顶」与「取消置顶工作区」。
- 只在 `C12`（平铺视图）：`LB-61`、`LB-62`（`今天` 分组）；`LB-63`–`LB-65` 因只有 1 条会话，分页项仍不出现。
- 只在 `C27`（压缩分隔行展开）：`TL-17`、`TL-67`（本机文案 `上下文已压缩`）、`TL-69`（`无法加载摘要。` + `重试`）、`TL-55`（工具详情表：输出/说明/日志）。
- 只在 `C34`（用量浮层）：`SB-15`、`SB-16`、`SB-17`、`SB-21`。
- 只在 `C39`（审批卡带超时）：`IN-73`（`timeoutMs: 60 s`）。
- 只在 `C42`（窄窗）：`OV-10` 的 `返回` 按钮；主区被替换为右栏，输入区浮在底部。
- 只在 `P1`/`P13`/`P17`/`P22`：右栏四类页签（改动/文件/终端/浏览器）的顶栏与空态。

### 8.4 需要注意的五处差异
1. **运行时独有只有一件：宿主系统通知**。D 态顶部出现 `DimAgent: 当前任务已完成` + `Open` 按钮（`session-done.json` 节点 `DimAgent: 当前任务已完成` 354,43、`Open` 629,41）。它由主进程发出（`<ref>/../electron-main/src/i18n/main-messages.ts:47` `runtimeTaskCompleted`、`:33` `openButton`），渲染进程没有任何模板对应 → 这是本次唯一「运行时有不模板无」的元素，属宿主/系统层，不进五区。
2. **审批卡没有独立分区**：它挂在输入区位置（`ChatWorkspace.vue:6714`–`:6718` 的 `InlinePermissionPrompt`），A 态里 `InputArea` 整个不渲染；`inventory-panels.md` 覆盖的是右栏与时间线内块，审批卡此前两边都没登记。
3. **「工作过程」汇总行在四态都没出现**：屏上出现的是分组头（`TimelineBlockGroup`，文案形如 `思考1轮 · 改2次文件、搜1次、抓取2次、其他1次`），而 `timeline.workSummaryWithActions`（仓库文案 `工作过程 · {actions}`，`TimelineAssistantMessage.vue:895`–`:915`）在 A/P/D/T 均未渲染。要么该行条件未满足，要么采集用的构建与仓库文案不一致——需一次定点核。
4. **无访问名的按钮仍在**：右栏 `+ 添加页签`（`RightPanelBar.vue:381`–`:397`，无 `aria-label`/`title`）、输入区 `+ 附件`（`InputArea.vue:2460`）、右栏 `筛选文件...`（`ContextPanel.vue:547`，只有 placeholder）——采集脚本一律跳过，只能靠截图补。
5. **隐藏标签仍被采集**：`Agent`（P/D/T 435,655）、`技能`（P/D/T 463,653）是 `opacity-0` 的展开标签（A 态没有输入区，故没有这两个节点）；对账时按「按钮可见、标签不可见」处理。

## 9. 本轮自检

- 十态采集回溯比对：E、N、R、Q、S、K、A、P、D、T。
  - `empty` 与 `new-session` 的 `items` 完全相等（同 MD5 的 PNG）→ 合并 E/N。
  - `session` 与 `session-approval` 的 `items` 完全相等（PNG 不同）→ 合并 A。
  - `tools-1..5`、`tools-done`、`tools-final`、`tools-rich` 节点集相同 → 合并 T（仅计时/滚动不同）。
- 五区逐区对齐模板：标题栏（`NewThreadHeader` + `ChatHeaderFrame`/`ChatHeader`）、左栏（`AppSidebar` + 7 个子组件，本轮补上 `ThreadContextMenu`）、时间线（`PlainTimeline` + `TimelineRow`/`TimelineBlockGroup`/`TimelineToolCallItem`）、输入区（`InputArea` + `ConversationComposer` + 底部栈 + `PermissionPrompt`）、状态条（`StatusBar`）。
- 空态（`NewThread`）单列，含 6 张办公模式推荐卡与两处下拉（工作区 / 分支）。
- 运行时可见但不属五区的四块（右栏、搜索聊天弹窗、定时任务页、技能页）单列在第 7 节，未与五区重复计数。
- 本轮补采共 **45 + 25 = 70 份状态**（`C1`–`C45` 在 `runtime/main/`，`P1`–`P25` 在 `runtime/panels/`），全部由 CDP 在参考产品上真实触发后导出，未改渲染代码、未伪造节点。
- 由这 70 份状态改判为「可见」的元素 **122 条**：标题栏 26、左栏 47、时间线 13、输入区 10、状态条 12、空态 6、区外 8（含新登记的 6 条会话行右键菜单项，不计入改判数）。
- **模板行补登记 6 条**：`ThreadContextMenu.vue` 此前整块漏登记，本轮补 `LB-107`–`LB-112`（置顶/重命名/在新窗口中打开/导出/归档/删除），证据是 `C13`。这使区 2 由 106 → 112、模板总数 395 → 401。
- 仍存 142 条未见，逐桶原因见 §8.2 与 §8.2.1；其中只有 32 条属「还能靠交互再试」，其余 110 条需要驱动 App Server/宿主的数据态（更新、失败重试、排队、附件、分页、沙箱/子代理审批、登出与限流）。
- 已实测但**未**出现的分支（据此改判为「需数据」，不再当「需交互」）：运行中再投递不产生排队面板（`C41`）；停止按钮点击时该轮已结束，未捕捉到「用户中断」（因此 `TL-60`、`TL-47` 仍缺）；压缩行只有「上下文已压缩」一种文案（`TL-68` 的进行/失败/跳过/取消未出现）；「工作过程」行仍只有 `已经工作了 …`（`TL-45` 缺）。
- 定点补采结论：会话态 + 左栏收起（`TB-05`/`13`–`16`）、会话态 + 右栏打开（`TB-40`）、消息内搜索（`TB-06`–`TB-12`）三类上一版欠账已由 `C1`/`C24`–`C26` 清零。
