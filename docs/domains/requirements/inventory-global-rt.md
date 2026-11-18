# 全局面板运行时清单（模板 ∪ 运行时）

范围：八类全局面板 —— 搜索（会话内搜索 / 搜索聊天 / `@` 提及 / `$` 技能）、命令（斜杠菜单）、审批（工具权限卡 / 沙箱越权卡 / 子代理模型卡 / 审批输入区 10 类渲染器）、提问、确认（删会话 / 归档 / 删工作区 / 编辑运行中）、重命名、导出与 Fork、导入与更新。

本文件只做元素对账：**有什么元素、什么条件下在场、参考产品已采状态里是否真的出现**。不写行为需求、不写我方命名、不写验收判定。

## 0. 取证与口径

### 0.1 材料

| 类 | 位置 |
|---|---|
| 模板 | 参考产品源码；仓库根记 `<ref>`，定位写作 `<ref>/<相对路径>:<行号>`，相对 `packages/desktop/packages/renderer/src` |
| 模板行来源 | 本文件的行集合沿用 `docs/domains/requirements/inventory-global.md` §A–§H（元素 ID、文案、控件、默认、条件、定位逐行继承），只新增「已采状态」列并按图标分支拆 3 处（见 §0.4） |
| 运行时（主窗口补采） | `docs/domains/requirements/runtime/main/` 60 份 `*.json`（`E/N/R/Q/S/K/A/P/D` 八码 + `C1`–`C45` 补采 + 7 份同族补充态），同目录同名 `.png`（1030×725 CSS 视口，DPR=2） |
| 运行时（本轮补采，`X1`–`X13` / `Y1`–`Y18`） | `docs/domains/requirements/runtime/global/` 34 份 `*.json` + 同名 `.png`（见 §0.2b） |
| 运行时（旁证，不计入判定） | `docs/domains/requirements/runtime/settings/通用.json`（浏览器数据入口）、`docs/domains/requirements/runtime/session/turns/11-tool-ask-user-pending.png`（提问卡仅截图、无 DOM 导出） |
| 文案 | 参考产品 `zh-CN` 语言包（`<ref>/i18n/zh-CN.ts`）；`{x}` 为插值占位 |
| 节点口径 | 每项 = `{chain, tag, role, text, label, x, y, w, h}`，单位 CSS px；`label` 即 `aria-label`/可见标签文本 |

### 0.2 状态码（本文件用到的）

| 码 | 文件 | 屏上内容 | 与本文件的关系 |
|---|---|---|---|
| C24 | `timeline-search-empty` | 会话内搜索条，查询为空 | 会话内搜索默认态 |
| C25 | `timeline-search-noresult` | 搜索条 + `0 个结果` | 无命中态 |
| C26 | `timeline-search-results` | 搜索条 + 上一个/下一个 + `1 / 35 个结果` + 命中高亮 | 有命中态 |
| Q | `search` | 「搜索聊天」弹窗（`未找到聊天`）；背景是 R | A2 唯一有节点的态（只含弹窗层） |
| S | `scheduled`、K | 独立路由页 + 同一弹窗 | 与 Q 同族，仅背景不同 |
| C37 | `composer-slash-menu` | 输入框键入 `/` 的指令菜单（10 行） | 斜杠命令弹层 |
| A | `session` ≡ `session-approval` | 会话停在 exec 权限卡上（左栏展开） | 工具权限卡 + 命令渲染器 |
| C38 | `session-streaming` | 同上，左栏收起 + 宿主通知横幅 | 审批卡第二证据（x 平移 −356） |
| C39 | `session-approval-timeout` | 审批卡带 `timeoutMs: 60 s` | 审批卡参数行第二分支 |
| C2/C13/C16/C17/C45/C19 | 会话行/标题栏/工作区菜单、搜索行悬停 | 入口菜单与悬停态 | **只作入口证据**，元素本身在 `inventory-main-rt.md`，不重复计数 |

`A` 的两份 `items` 完全相同（PNG 不同），合并记 `A`。

### 0.2b 本轮补采状态码（全部导出到 `docs/domains/requirements/runtime/global/`）

| 码 | 文件 | 触发方式 | 屏上内容 |
|---|---|---|---|
| X1 | `X1-search-pinned-group` | 置顶一条会话后打开「搜索聊天」（空查询） | `置顶聊天` 分组 + 星标结果行 + `工作区` 副行 + `⌘1` + 首行选中 |
| X2 | `X2-search-query-result` | 同一弹窗键入 `文件` | 查询词 + 置顶结果行（过滤后仍命中） |
| X3 | `X3-search-recent-results` / `X3b-search-recent-query` | 取消置顶后重开弹窗（空查询 / 键入 `文件`） | `最近聊天` 分组 + 消息图标结果行 |
| X4 | `X4-slash-empty` | 输入框键入 `/zzzz` | 弹层空态 `未找到命令或技能` |
| X5 | `X5-slash-selected` | 键入 `/` 后按 ↓ | `/goal` 行强调底（选中态） |
| X6 | `X6-slash-scrolled` / `X6b-slash-skills-group` / `X6c-slash-skills-header` | 把弹层滚到底 / 滚到 `技能` 分组 | `/compact`、`/branch`、`/workflow`、`技能` 分组与技能候选行 |
| X7 | `X7-mention-open` | 输入框键入 `@` | `文件` 分组 + 顶层目录候选（`adr-audit/` 等）+ 首行选中 |
| X8 | `X8-mention-empty` / `X8b-mention-files` | `@zzz` / 删回 `@` | 技能候选组（含描述）/ 文件候选组 |
| X9 | `X9-mention-noresult` | `@qqqqq` | `未找到文件或目录` |
| X10 | `X10-dollar-skills` | 输入框键入 `$` | `技能` 分组 + 技能候选（名称 + 描述）+ 首行选中 |
| X11 | `X11-dollar-noresult` | `$qqqq` | `未找到技能` |
| X12 | `X12-export-dialog` | 会话行右键 → 导出 | 导出对话框（导出内容三项默认勾选、消息、搜索回合、原始错误行、取消/导出置灰） |
| X13 | `X13-delete-session-dialog` | 会话行右键 → 删除 | `删除会话` 确认框（单条文案 + 取消/删除） |
| Y1 | `Y1-terminal-open` | 右栏 `+` → 终端 | 面板内终端（`(base) → dim-agent` 提示符） |
| Y2 | `Y2-terminal-command` | 在终端里执行 `echo terminal-ok` | 命令行 + 回显 `terminal-ok` + 新提示符 |
| Y3 | `Y3-terminal-context-menu` | 终端内右键 | 复制选中内容 / 粘贴 / 清屏 / 新建终端 / 关闭进程 |
| Y4 | `Y4-terminal-last-closed` | 关闭最后一个终端页签 | 页签消失并切回上一页签；**没有**残留提示行 |
| Y6 | `Y6-plugins-tab` | 技能/插件页 → `插件` → `插件市场` | `Marketplace 暂不可用 / 需要上游 API` |
| Y7 | `Y7-my-plugins` | 同上 → `我的插件` | 内置 / Dim 插件卡片 + `＋ 添加插件` |
| Y8 | `Y8-plugin-install-dialog` | `我的插件` → 添加插件 | 从 Git 添加插件（本地文件夹模式：来源、插件 id、安装后启用、取消/添加插件） |
| Y9 | `Y9-plugin-install-git` | 对话框内切到 `Git 仓库` | Git 仓库 / Git ref（main）字段 |
| Y10 | `Y10-settings-general-browser` | 设置 → 通用 → 浏览器数据 | `Chrome 数据` 的 `导入…` 与 `浏览器存储` 的 `清理数据…` 入口 |
| Y11 | `Y11-browser-import-dialog` | 点 `导入…` 后立刻采集 | `正在扫描 Chrome 配置…` + 取消/导入（置灰） |
| Y12 | `Y12-browser-import-scanned` | 扫描结束 | `Chrome 配置` 三行（`可导入`）+ 错误块（`无法访问 Chrome 加密密钥…`）+ 诊断行 + 重试 |
| Y13 | `Y13-browser-clear-dialog` | 点 `清理数据…` | `清理浏览器数据` 确认框（描述 + 取消/危险按钮） |
| Y14 | `Y14-delete-workspace-dialog` | 工作区行 `⋯` → 移除 | `删除工作区会话`（条数 + 路径 + 取消/删除） |
| Y15 | `Y15-rename-workspace-dialog` | 工作区行 `⋯` → 重命名 | `重命名工作区`（名称标签、预填 input、路径、取消/确认） |
| Y16 | `Y16-askuser-pending` | 发一条要求 `ask_user` 单选的消息 | 提问卡：标题 `单选测试`、正文、A/B/C 单选、`拒绝`/`提交`（未选时置灰）、会话行 `等待操作` |
| Y17 | `Y17-askuser-collapsed` | 点 `收起` | 收起态 `待选择 · 3 个选项` + `展开` |
| Y18 | `Y18-askuser-selected` | 选 A 后 | 选中行主色边框 + `提交` 转实心 |
| Y3b | `Y3b-terminal-context-menu` | 终端里执行过命令后再右键 | 同一菜单五项（复制选中内容为禁用态） |
| Y5b | `Y5b-terminal-process-closed` | 右键 → 关闭进程 | 终端页签直接关闭、切回上一页签；**无**［进程已关闭］提示行 |
| Z2 | `Z2-timeline-changes-collapsed` | 一轮里改了一个文件的会话 | 轮末 `1 个文件改动` 折叠行 + 活动条同名 chip |
| Z3 | `Z3-timeline-changes-expanded` | 点开该折叠行 | 展开出改动文件行（`rt-approval-demo.txt`） |

### 0.3 列义

- **元素 ID / 名称 / 控件 / 默认 / 条件 / 定位**：与 `inventory-global.md` 同值（条件列已写全祖先链与 `v-if` 分支）。
- **已采状态**：五种取值——
  - `可见 <码> x,y w×h`：在运行时 JSON 命中，坐标为包围盒。
  - `可见（随行；…）`：节点在场但 JSON 未单列（容器、无文案 svg、被文本切分吞掉的片段），括注反推方式。
  - `可见（仅视觉；…）`：节点在场，但该行关心的差异只体现在底色/不透明度/旋转，JSON 无此维度，须读同名 PNG。
  - `需交互（…）`：要人工操作或键入才出现，括注具体操作。
  - `需数据（…）`：要特定数据 / App Server 状态才出现，括注具体条件。

### 0.4 行粒度与计数规则

1. 一行 = 模板一个节点；同一节点的纯文案三元（`loading ? A : B`）合成一行写 `A / B`，不同**交互状态**（未触发 / 已触发）各成一行。
2. **图标分支拆行**：`G-A1-02`（放大镜 / 旋转加载）、`G-A2-11`（5 个行首图标分支）、`G-H6-09`（对勾 / 告警）按分支拆成 `-a`、`-b`… 后缀行；其余行集合与 `inventory-global.md` 逐行一致，便于对照（§9）。
3. `v-for` 只记一行（模板节点），实例数写进「已采状态」列（如斜杠菜单 10 行）。
4. 「可见」按模板节点计，不按实例计。
5. 判定「不可观察」时先读 §0.5 的采集局限；本文件**不使用「死代码」标记**——八类里没有一份态能让宿主整体在场，缺节点一律先归「需交互 / 需数据」。

### 0.5 采集局限（判「未出现」前必读）

1. **五类原先零覆盖，本轮补掉四类半**：补采后提问卡（`D1`）、删除会话 / 删除工作区 / 重命名工作区 / 导出对话 / 添加插件 / Chrome Cookie 导入 / 浏览器数据清理都有态；仍缺的是「归档（E3）」「会话重命名（F1，`SessionTitle` 不可用）」「Fork（G2，需同一会话在两窗口打开）」与插件市场系四框（上游 API 未接入）。
2. **同族的两个确认框不属本文件**：`dialog-full-access`（完全访问确认）、`dialog-logout-confirm`（退出登录）已采，但归账户/权限域，不在八类内，不得拿来充当 E 类证据。
3. **模态与弹层会 aria-hidden 掉背景**：Q/S/K 三态只有弹窗层（5–6 节点），`dialog-*` 同理（6–16 节点）；因此「弹窗背后少元素」不能判缺。
4. **`search.json` 的标题/描述宽度失真**：`搜索聊天`、`搜索并打开已有聊天` 的 `w` 采成 1（`sr-only`），宽度不可用；PNG 证明二者屏上不可见。
5. **placeholder 不入 JSON**：采集只取 `text`/`label`，输入框占位文案只能读 PNG（A1-03、A2-04、D1-10 受影响）。
6. **C24/C25/C26 的时间线内容不同**（分别停在 21:54 / 21:22 / 21:22）：搜索条以外的节点不能跨态互相补位。
7. **审批卡坐标随左栏状态平移**：A 在 x≈386，C38/C39 在 x≈30（差 356），坐标不可跨态混用。
8. **`SessionTitle` 在本机不可用**：C2/C13 的重命名项直接渲染成「重命名（SessionTitle 不可用）」且禁用，F1 在本环境无条件通达。
9. **提问卡已补到 DOM**：本轮用 `ask_user` 触发 `Y16`–`Y18`（待答 / 收起 / 已选），单字段单选全组可取；旧旁证 `runtime/session/turns/11-tool-ask-user-pending.png` 只作历史对照。多字段、倒计时、多选与自定义答案仍缺（见 §10.1 D）。

---

## 1. A 搜索

### A1 会话内搜索浮层（ThreadSearchBar）

条件：主区有活跃会话时按 `Cmd/Ctrl+F` 打开；浮层绝对定位在标题栏右侧，点浮层外关闭。已采三态：C24（空）/ C25（无命中）/ C26（有命中）。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 | 已采状态 |
|---|---|---|---|---|---|---|
| G-A1-01 | （无文案）浮层容器 | 浮层（420px，圆角 + 边框 + 阴影 + 背景模糊，两行网格：44px + 内容） | 关闭 | 会话内搜索打开 | `<ref>/components/chat-workspace/timeline/ThreadSearchBar.vue:112` | 可见 C24/C25/C26（随行；容器未单列，按输入框 343,23 + 关闭键 688,18 + 结果行按钮 322,61 反推 ≈310,10 406×80；模板声明 420，实测窄 14） |
| G-A1-02a | — 搜索图标（放大镜） | 图标（20px） | 放大镜 | 不在搜索中 | `ThreadSearchBar.vue:118` | 可见 C24/C25/C26（随行；svg 无节点，JSON 无坐标；PNG 见其贴在输入框左端内侧，与占位文字同一行） |
| G-A1-02b | — 加载图标（旋转） | 图标（20px） | 隐藏 | 搜索进行中 | `ThreadSearchBar.vue:123` | 需数据（`pending` 态窗口 <1s，本机检索瞬时返回，三态均未捕获） |
| G-A1-03 | 「搜索聊天...」 | 输入框（单行，placeholder，无边框） | 空；时间线有选中文本时预填该文本 | 打开即聚焦并全选 | `ThreadSearchBar.vue:135`；i18n `chatHeader.searchChatPlaceholder` | 可见 C24 343,23 338×18（空）；C25/C26 同框有查询词（`value` 不入 JSON，PNG 见光标在首字符前） |
| G-A1-04 | 「搜索聊天消息」 | 输入框的无障碍名 | — | 常驻 | `ThreadSearchBar.vue:134`；i18n `chatHeader.searchChatLabel` | 可见 C24/C25/C26（`label` 字段 = `搜索聊天消息`，随 G-A1-03） |
| G-A1-05 | 「关闭搜索」 | 图标按钮（X，32px） | 可用 | 常驻 | `ThreadSearchBar.vue:146`；i18n `chatHeader.closeSearch` | 可见 C24/C25/C26 688,18 28×28（`label` = `关闭搜索`） |
| G-A1-06 | （无文案）结果行 | 行（36px，上边框，左右分布） | 隐藏 | 查询非空 | `ThreadSearchBar.vue:155` | 可见 C25/C26（随行；y=61 行，由两侧按钮 322,61/357,61 反推；C24 空查询时不存在） |
| G-A1-07 | 「上一个搜索结果」 | 图标按钮（↑，28px） | 禁用 | 结果数为 0 时禁用 | `ThreadSearchBar.vue:166`；i18n `chatHeader.previousSearchResult` | 可见 C25/C26 322,61 25×25（`label` = 上一个搜索结果；可用/禁用差异只体现在图标颜色，属视觉） |
| G-A1-08 | 「下一个搜索结果」 | 图标按钮（↓，28px） | 禁用 | 结果数为 0 时禁用 | `ThreadSearchBar.vue:178`；i18n `chatHeader.nextSearchResult` | 可见 C25/C26 357,61 25×25（`label` = 下一个搜索结果） |
| G-A1-09 | 「0 个结果」 / 「{current} / {total} 个结果」 | 文字（结果行右，`aria-live=polite`） | 空字符串 | 查询非空且不在搜索中 | `ThreadSearchBar.vue:35`；i18n `chatHeader.searchNoResults`、`chatHeader.searchResults` | 可见 C25「0 个结果」664,64 48×18；C26「1 / 35 个结果」641,64 71×18；C24 无 |

元素数 10（原 9，`G-A1-02` 拆两行）。

### A2 搜索聊天对话框（ChatSearchDialog）

条件：全局入口唤起（左栏「搜索」行 / `⌘G`），模态，无右上关闭按钮；最多 8 条结果，按置顶/最近分组。已采：Q（空结果），S/K 同族。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 | 已采状态 |
|---|---|---|---|---|---|---|
| G-A2-01 | （无文案）对话框容器 | 模态（`max-w-xl`，无内边距，无关闭按钮） | 关闭 | 唤起后 | `<ref>/components/sidebar/ChatSearchDialog.vue:186` | 可见 Q 291,293 448×139（`role=dialog`）；S/K 同坐标 |
| G-A2-02 | 「搜索聊天」 / 「搜索并打开已有聊天」 | 标题 + 描述（`sr-only`，屏上不可见） | — | 常驻 | `ChatSearchDialog.vue:192`；i18n `chatSearch.title`、`chatSearch.description` | 可见 Q（h2 291,293 1×16、p 291,316 1×18；`w=1` 即 sr-only，PNG 屏上无此两行文案） |
| G-A2-03 | — 搜索图标 | 图标（16px） | 放大镜 | 常驻 | `ChatSearchDialog.vue:197` | 可见 Q（随行；svg 无节点，PNG 位于占位文字左 ≈300,312） |
| G-A2-04 | 「搜索聊天」 | 输入框（placeholder，无边框，高 56px） | 空 | 常驻；打开即聚焦 | `ChatSearchDialog.vue:201`；i18n `chatSearch.placeholder` | 可见 Q 331,294 394×48（placeholder 不入 JSON，PNG 显示「搜索聊天」；实测高 48，模板声明 56） |
| G-A2-05 | 「正在加载...」 | 状态行（旋转图标 + 文字，`role=status`） | 隐藏 | 当前页结果未就绪 | `ChatSearchDialog.vue:209`；i18n `common.loading` | 需数据（结果首页未返回；本机会话列表瞬时返回） |
| G-A2-06 | 「上一页」 | 文字按钮（整行，32px） | 隐藏 | 存在上一页 | `ChatSearchDialog.vue:210`；i18n `sidebar.previousPage` | 需数据（已翻到第 2 页之后） |
| G-A2-07 | 「显示更多」 | 文字按钮（整行，32px） | 隐藏 | 还有下一页 | `ChatSearchDialog.vue:211`；i18n `sidebar.showMore` | 需数据（结果 >8 条） |
| G-A2-08 | 「置顶聊天」 / 「最近聊天」 | 分组标题（12px 灰字） | — | 该组有结果 | `ChatSearchDialog.vue:107`；i18n `chatSearch.pinnedChats`、`chatSearch.recentChats` | 可见 X1 299,357 432×18（`置顶聊天`）；X3 299,357 432×18（`最近聊天`） |
| G-A2-09 | （会话标题） | 结果行主行（14px 半粗，截断） | — | 有结果 | `ChatSearchDialog.vue:242` | 可见 X1/X3 345,380 355×18（会话标题） |
| G-A2-10 | （项目显示名） | 结果行副行（12px 灰字，截断） | — | 有结果 | `ChatSearchDialog.vue:245` | 可见 X1/X3 345,398 355×14（`工作区`） |
| G-A2-11a | — 行首图标（错误） | 图标（32px 底托，红色告警圈） | 消息图标 | 该行 RPC 失败 | `ChatSearchDialog.vue:237`–`:241` | 需数据（单行加载失败） |
| G-A2-11b | — 行首图标（加载中） | 图标（32px 底托，旋转） | 消息图标 | 该行数据加载中 | `ChatSearchDialog.vue:237`–`:241` | 需数据（单行加载中，窗口极短） |
| G-A2-11c | — 行首图标（未读） | 图标（32px 底托，实心圆点） | 消息图标 | 该会话未读 | `ChatSearchDialog.vue:237`–`:241` | 需数据（存在未读会话） |
| G-A2-11d | — 行首图标（置顶） | 图标（32px 底托，琥珀星标） | 消息图标 | 该会话已置顶 | `ChatSearchDialog.vue:237`–`:241` | 可见（仅视觉）X1 行首 32×32 底托内为琥珀星标（PNG） |
| G-A2-11e | — 行首图标（默认） | 图标（32px 底托，消息图标） | 消息图标 | 以上皆否 | `ChatSearchDialog.vue:237`–`:241` | 可见（仅视觉）X3 行首 32×32 底托内为消息图标（PNG） |
| G-A2-12 | 「⌘{n}」 / 「Ctrl+{n}」 | 快捷键提示（12px 灰字） | 序号 1–8 | 有结果 | `ChatSearchDialog.vue:248` | 可见 X1/X3 710,389 14×14（`⌘1`） |
| G-A2-13 | （行选中态） | 行状态（强调底 + 强调前景） | 第 1 行为选中 | 上下键或悬停移动 | `ChatSearchDialog.vue:228` | 可见（仅视觉）X1/X3 首行强调底（PNG；第 1 行默认选中） |
| G-A2-14 | 「未找到聊天」 | 空态文字（居中，上下留白 32px） | 隐藏 | 无任何结果 | `ChatSearchDialog.vue:259`；i18n `chatSearch.noResults` | 可见 Q 299,350 432×74（Q/S/K 三态均为该分支） |

元素数 18（原 14，`G-A2-11` 拆五行）。

### A3 `@` 提及搜索弹层（文件 / 提交 / 技能）

条件：输入框可用且有工作区目录，键入 `@` 后出现；查询非空时合并文件、提交、技能三类候选。已采态里没有任何一份捕获，全部为操作型。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 | 已采状态 |
|---|---|---|---|---|---|---|
| G-A3-01 | （无文案）弹层容器 | 浮层（`max-h-72`，宽 `min(35rem,100vw-2rem)`，锚在光标上方或下方） | 隐藏 | `@` 提及激活 | `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue:1386` | 可见 X7（弹层 ≈390,330 481×276，由 `文件` 分组 390,328 与候选行反推）；X8b/X9 同容器 |
| G-A3-02 | （无文案）加载骨架 | 两条脉冲占位（32px） | — | 候选未返回 | `ComposerEditor.vue:1393` | 需数据（文件搜索 RPC 未返回，窗口 <1s） |
| G-A3-03 | 「未找到文件或目录」 / 「当前无工作区目录」 / 「文件搜索失败。」 | 空态文字 | — | 依次为：查询无命中 / 会话无 cwd / 该次搜索 RPC 失败 | `ComposerEditor.vue:333`；i18n `composer.noFileMatches`、`composer.noProjectDirectory`、`contextPanel.searchFailed` | 可见 X9 390,539 481×32（`未找到文件或目录`）；另两分支仍需数据（无 cwd / RPC 失败） |
| G-A3-04 | 「文件」 / 「技能」 | 分组标题（12px 灰字） | — | 该组有候选 | `ComposerEditor.vue:291`；i18n `composer.groupFiles`、`composer.groupSkills` | 可见 X7 390,328（`文件` 分组）；X8b ≈390,330（`技能` 分组） |
| G-A3-05 | （文件名 / 目录名） | 候选行主字（加粗；目录名追加 `/`） | — | 文件候选 | `ComposerEditor.vue:1424`、`:271` | 可见 X7 401,358 58×18（`adr-audit/` 等目录候选，目录名带 `/`） |
| G-A3-06 | （所在目录） | 候选行次字（12px 灰字，截断） | 无 | 文件候选且有上级目录 | `ComposerEditor.vue:1426` | 需数据（候选位于子目录） |
| G-A3-07 | （提交短 hash，7 位） | 候选行主字（半粗） | — | 提交候选 | `ComposerEditor.vue:262` | 需数据（`@` 查询命中 git 提交） |
| G-A3-08 | （提交标题 subject） | 候选行次字（12px 灰字，截断） | — | 提交候选 | `ComposerEditor.vue:263` | 需数据（同上） |
| G-A3-09 | （作者 · 日期） | 候选行右尾小字（12px，70% 不透明） | — | 提交候选 | `ComposerEditor.vue:264`、`:1432` | 需数据（同上） |
| G-A3-10 | （技能名称 + 描述） | 候选行主字 + 次字 | — | 技能候选；描述缺省时只显示名称 | `ComposerEditor.vue:268`、`:269` | 可见 X8b ≈401,358（`baoyu-translate` + 英文描述） |
| G-A3-11 | （行选中态） | 行状态（强调底 + 强调前景） | 第 0 行 | 上下键或悬停移动 | `ComposerEditor.vue:1421` | 可见（仅视觉）X7/X8b 首行强调底（PNG） |

元素数 11。

### A4 `$` 技能提及弹层

条件：输入框可用且有 suggestion provider，键入 `$` 后出现；只列技能（同名按工作区/用户/插件优先级去重，禁用的不列）。

注：C9 `popover-skills` **不是**本组——它是输入区技能按钮的浮层（外层 `role=menu`、宽 252、无 `$` 触发），元素归 `inventory-main-rt.md`，不能拿来充 A4 的证据。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 | 已采状态 |
|---|---|---|---|---|---|---|
| G-A4-01 | （无文案）弹层容器 | 浮层（`max-h-72`，宽 `min(24rem,100vw-2rem)`） | 隐藏 | `$` 提及激活 | `ComposerEditor.vue:1442` | 可见 X10（弹层 ≈390,320 327×221，同 B1 容器规格） |
| G-A4-02 | （无文案）加载骨架 | 两条脉冲占位 | — | 技能列表未返回 | `ComposerEditor.vue:1449` | 需数据（技能列表 RPC 未返回） |
| G-A4-03 | 「未找到技能」 | 空态文字 | — | 无匹配技能 | `ComposerEditor.vue:380`；i18n `composer.noSkill` | 可见 X11 390,539 327×32（`未找到技能`） |
| G-A4-04 | （技能名称 + 描述） | 候选行主字 + 右尾小字 | — | 有技能；描述缺省时只显示名称 | `ComposerEditor.vue:1481`、`:1482` | 可见 X10 401,358 65×18（`agents-sdk` + 描述） |
| G-A4-05 | （行选中态） | 行状态（强调底 + 强调前景） | 第 0 行 | 上下键或悬停移动 | `ComposerEditor.vue:1478` | 可见（仅视觉）X10 首行强调底（PNG） |

元素数 5。

### A5 时间线命中高亮

条件：会话内搜索有查询时，时间线正文内的命中片段被包一层行内标记。已采：C26 同屏两个命中（当前项 + 非当前项）。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 | 已采状态 |
|---|---|---|---|---|---|---|
| G-A5-01 | （命中片段） | 行内高亮（圆角 3px，黄色 45% 底） | — | 命中但不是当前项 | `<ref>/components/chat-workspace/timeline/TimelineSearchHighlightedText.vue:78`、`:84` | 可见 C26 538,217 28×17（`文件`，在助手正文「当前目录 … 文件列表：」内，PNG 为纯黄底） |
| G-A5-02 | （当前命中片段） | 行内高亮（橙色 70% 底 + 1px 描边） | — | 是当前定位项 | `TimelineSearchHighlightedText.vue:83`、`:89` | 可见 C26 675,83 28×17（用户消息内第一个 `文件`，PNG 带描边；文本被命中切分成两段：549,83 + 675,83） |

元素数 2。

区域 A 合计 46（原 41；+1 `G-A1-02`、+4 `G-A2-11`）。

---

## 2. B 命令

### B1 斜杠命令弹层

条件：输入框可用，行首键入 `/` 后出现；`new`、`login` 两个命令被显式排除，不在应用内出现。已采：C37（10 行，弹层滚动停在 `/compact`）。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 | 已采状态 |
|---|---|---|---|---|---|---|
| G-B1-01 | （无文案）弹层容器 | 浮层（`max-h-72`，宽 `min(24rem,100vw-2rem)`） | 隐藏 | `/` 激活 | `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue:1442` | 可见 C37（随行；容器未单列，按分组标题 390,328 与 PNG 底边 ≈541 反推 ≈390,320 327×221——高度受输入框上方可用空间限制，未达 `max-h-72` 的 288） |
| G-B1-02 | （无文案）加载骨架 | 两条脉冲占位 | — | 命令/技能未返回 | `ComposerEditor.vue:1449` | 需数据（命令与技能列表首次拉取未返回，窗口 <1s） |
| G-B1-03 | 「未找到命令或技能」 | 空态文字 | — | 无匹配项 | `ComposerEditor.vue:367`；i18n `composer.noCommandOrSkill` | 可见 X4 390,539 327×32（`未找到命令或技能`） |
| G-B1-04 | 「指令」 / 「技能」 | 分组标题（12px 灰字） | — | 该组有项 | `ComposerEditor.vue:290`；i18n `composer.groupCommands`、`composer.groupSkills` | 可见 `指令` 由 C37 命中；`技能` 分支见 X6c ≈390,300（滚动后 `技能` 分组标题 + 技能行） |
| G-B1-05 | 「/{命令名}」 + （命令描述） | 候选行（主字 + 右尾 12px 灰字，截断） | — | 命令项；无描述时只显示名称 | `ComposerEditor.vue:1481`、`:1482` | 可见 C37（DOM 内 10 行，行按钮 390,352/380/408/436/464/492/520/548/576/604 327×28；PNG 实际露出到 `/reload-skills`，后两行落在弹层裁剪区内） |
| G-B1-06 | （技能名称） + （技能描述） | 候选行（主字 + 右尾小字） | — | 技能项；无描述时只显示名称 | `ComposerEditor.vue:1481`、`:1482` | 可见 X6c 401,354 65×18（`agents-sdk`）起，随后 background-check / baoyu-* 等技能行 |
| G-B1-07 | （行选中态） | 行状态（强调底 + 强调前景） | 第 0 行 | 上下键或悬停移动 | `ComposerEditor.vue:1478` | 可见（仅视觉）X5 `/goal` 行强调底（↑/↓ 后第 2 行为选中态；C37 无选中行） |
| G-B1-08 | 「/agent」「切换至 Agent 模式」 | 固定命令行 | — | 常驻候选 | `<ref>/components/chat-workspace/composer/desktop-slash-commands.ts:37`；i18n `slashCommands.switchToAgent` | 可见 C37 401,358 / 619,359 |
| G-B1-09 | 「/goal」「切换至 Goal 模式」 | 固定命令行 | — | 常驻候选 | `desktop-slash-commands.ts:43`；i18n `slashCommands.switchToGoal` | 可见 C37 401,386 / 625,387 |
| G-B1-10 | 「/pet」「唤醒或收起 Pet」 | 固定命令行 | — | 常驻候选 | `desktop-slash-commands.ts:49`；i18n `slashCommands.togglePet` | 可见 C37 401,414 / 635,415 |
| G-B1-11 | 「/settings」「打开设置」 | 固定命令行 | — | 非独立会话窗口 | `desktop-slash-commands.ts:55`；i18n `slashCommands.openSettings` | 可见 C37 401,442 / 665,443 |
| G-B1-12 | 「/models」「打开模型设置」 | 固定命令行 | — | 非独立会话窗口 | `desktop-slash-commands.ts:61`；i18n `slashCommands.openModelSettings` | 可见 C37 401,470 / 644,471 |
| G-B1-13 | 「/mcp」「打开 MCP 设置」 | 固定命令行 | — | 非独立会话窗口 | `desktop-slash-commands.ts:67`；i18n `slashCommands.openMcpSettings` | 可见 C37 401,498 / 635,499 |
| G-B1-14 | 「/skills」「打开技能」 | 固定命令行 | — | 非独立会话窗口 | `desktop-slash-commands.ts:73`；i18n `slashCommands.openSkills` | 可见 C37 401,526 / 665,527 |
| G-B1-15 | 「/reload-skills」「重新扫描磁盘上的技能」 | 固定命令行 | — | 常驻候选 | `desktop-slash-commands.ts:79`；i18n `slashCommands.reloadSkills` | 可见 C37 401,554 / 602,555 |
| G-B1-16 | 「/reload-plugins」「重新扫描已安装插件」 | 固定命令行 | — | 常驻候选 | `desktop-slash-commands.ts:85`；i18n `slashCommands.reloadPlugins` | 可见 C37 401,582 / 612,583（DOM 命中；PNG 中在弹层裁剪区内，需滚动露出） |
| G-B1-17 | 「/compact」「压缩当前上下文」 | 固定命令行 | — | 常驻候选 | `desktop-slash-commands.ts:91`；i18n `slashCommands.compactContext` | 可见 C37 401,610 / 633,611（DOM 命中；PNG 中在弹层裁剪区内，需滚动露出） |
| G-B1-18 | 「/fast」「Toggle fast mode」 | 固定命令行（描述未本地化，英文原文） | — | 快速模式可用时 | `desktop-slash-commands.ts:95` | 需数据（`/fast` 仅在快速模式可用；X6b 已滚到末项仍未出现该项） |
| G-B1-19 | 「/branch」「侧边对话」 | 固定命令行 | — | 常驻候选 | `desktop-slash-commands.ts:103`；i18n `sideChat.title` | 可见 X6c 401,274 45×18（`/branch`，滚动后坐标） |
| G-B1-20 | 「/workflow」「Browse workflows」 | 固定命令行（描述未本地化，英文原文） | — | 常驻候选 | `desktop-slash-commands.ts:107` | 可见 X6c 401,302 58×18（`/workflow`，同上） |

元素数 20。

### B2 命令面板基元（ui/command）

条件：参考产品内只有「设置 → 添加模型」调色板使用这套基元；斜杠命令与会话搜索各自手写浮层，不复用本组。已采 60 份态里没有任何一份打开该调色板。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 | 已采状态 |
|---|---|---|---|---|---|---|
| G-B2-01 | 「Command Palette」 / 「Search for a command to run...」 | 标题 + 描述（`sr-only`） | 上述英文默认值 | 调用方未覆盖；实际使用处为「添加模型」/「搜索模型...」 | `<ref>/components/ui/command/CommandDialog.vue:12`；i18n `configurationSettings.addModel`、`inputArea.searchModels` | 需交互（设置 → 添加模型调色板；本机设置态 15 份均为页面级，无该浮层） |
| G-B2-02 | （无文案）面板容器 | 容器（弹出底，圆角，`h-full w-full`） | — | 面板打开 | `Command.vue:80` | 需交互（同上） |
| G-B2-03 | （无文案）搜索输入 | 输入框（左置放大镜图标，自动聚焦） | 空 | 面板打开 | `CommandInput.vue:31` | 需交互（同上） |
| G-B2-04 | （分组标题） | 分组标题（12px 灰字） | 无 | 传入 `heading` | `CommandGroup.vue:40` | 需数据（调用方传 `heading`；该调色板是否传未验证） |
| G-B2-05 | （命令项） | 行（可选图标 + 文字，选中态强调底） | — | 有项且匹配筛选 | `CommandItem.vue:21` | 需交互（面板打开后键入筛选词） |
| G-B2-06 | （快捷键） | 行尾小字（字距加大） | 无 | 传入 shortcut | `<ref>/components/ui/command/CommandShortcut.vue:14` | 需数据（调用方传 `shortcut`） |
| G-B2-07 | （分隔线） | 分隔线（1px） | 无 | 传入时 | `<ref>/components/ui/command/CommandSeparator.vue:15` | 需数据（调用方使用分隔线） |
| G-B2-08 | （空态） | 空态容器（居中，上下 24px） | 隐藏 | 有搜索词且匹配数为 0 | `CommandEmpty.vue:20` | 需交互（面板打开后键入无命中词） |
| G-B2-09 | （列表容器） | 滚动列表（最大高 300px） | — | 面板打开 | `CommandList.vue:16` | 需交互（同上） |

元素数 9。

区域 B 合计 29。

---

## 3. C 审批

### C1 工具权限卡（内联，占输入位上方）

条件：会话收到 `tool_call` 权限请求；卡片贴输入区上沿，不遮时间线。已采：A（x≈386）、C38/C39（左栏收起，x≈30），三份都是 `exec` 的权限请求。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 | 已采状态 |
|---|---|---|---|---|---|---|
| G-C1-01 | （无文案）卡片容器 | 卡片（边框 + 内边距 16px，背景与页面同色） | 隐藏 | 有权限请求 | `<ref>/components/chat-workspace/interactions/PermissionPrompt.vue:222` | 可见 A/C38/C39（随行；容器未单列，按标题 386,519 + 按钮行底 660 反推 ≈370,503 647×173） |
| G-C1-02 | 「DimAgent 想要：{action}」 | 标题（16px 半粗） | — | 常驻；`{action}` 为工具动作标签（运行命令、读取文件等） | `PermissionPrompt.vue:81`、`:78`；i18n `permissionPrompt.actionRequest` | 可见 A「DimAgent 想要：执行」386,519 615×21；C38/C39 30,519 659×21 |
| G-C1-03 | 「使用工具」 | 标题动作回退 | — | 工具无专用标签或来自 MCP | `PermissionPrompt.vue:63`、`approval-panels/approval-tool-display.ts:38`；i18n `permissionPrompt.useTool` | 需数据（MCP / 未登记工具的权限请求；本次为 `exec`，有专用标签「执行」） |
| G-C1-04 | （无文案）成员审批上下文块 | 区块（浅底 30%，边框 60%，内边距 12px） | 隐藏 | 请求来自团队成员分支 | `PermissionPrompt.vue:231` | 需数据（请求来自团队成员分支） |
| G-C1-05 | 「该成员想要：{action}」 / 「正在加载成员操作...」 / 「无法加载成员操作」 / 「当前客户端无法审查成员操作」 | 区块标题（14px 半粗） | — | 依次为：成员动作就绪 / 加载中 / 加载失败 / 客户端与服务端都无法给出原始动作 | `PermissionPrompt.vue:80`–`:74`；i18n `permissionPrompt.memberApproval.actionRequest`/`loading`/`errorTitle`/`unavailableTitle` | 需数据（成员分支请求；四分支各需对应加载结果） |
| G-C1-06 | 「成员 {agentId} · 分支 {branchId}」 | 文字行（14px 灰） | 隐藏 | 成员标识存在 | `PermissionPrompt.vue:87`；i18n `permissionPrompt.memberApproval.member` | 需数据（成员请求且带 agentId/branchId） |
| G-C1-07 | 「主分支 {branchId}」 | 文字行（14px 灰） | 隐藏 | 存在 leader 分支 | `PermissionPrompt.vue:96`；i18n `permissionPrompt.memberApproval.leaderBranch` | 需数据（同上且有 leader 分支） |
| G-C1-08 | 「App Server 未能返回原始操作。请重试或拒绝。」 | 错误文字（14px 危险色） | 隐藏 | 成员动作加载失败 | `PermissionPrompt.vue:238`；i18n `permissionPrompt.memberApproval.error` | 需数据（成员动作 RPC 失败） |
| G-C1-09 | 「当前客户端与 App Server 无法显示原始操作，只能拒绝。」 | 说明文字（14px 灰） | 隐藏 | 成员动作不可审查 | `PermissionPrompt.vue:241`；i18n `permissionPrompt.memberApproval.unavailable` | 需数据（协议缺原始动作字段） |
| G-C1-10 | 「审批 {approvalId}」 | 等宽文字（12px 灰，可断行） | 隐藏 | 成员动作不可审查 | `PermissionPrompt.vue:244`；i18n `permissionPrompt.memberApproval.approvalId` | 需数据（同上） |
| G-C1-11 | （审批原因） | 说明文字（14px 灰） | 隐藏 | 成员动作就绪且有 reason | `PermissionPrompt.vue:250` | 需数据（成员动作带 reason） |
| G-C1-12 | 「重试」 | 次级小按钮（`outline`） | 隐藏 | 加载失败且可重试 | `PermissionPrompt.vue:257`；i18n `permissionPrompt.memberApproval.retry` | 需交互（成员动作加载失败后点「重试」） |
| G-C1-13 | （无文案）审批输入区 | 区块（见 C4，最大高 240px） | 隐藏 | 工具展示区解析成功 | `PermissionPrompt.vue:260` | 可见 A/C38/C39（随行；见 C4 各组件的包围盒） |
| G-C1-14 | 「需要你判断」 + （交回原因） | 提示块（浅底 50%，内边距 12px，标题 14px 半粗） | 隐藏 | 自动审批交回人工（`reviewer_uncertain`） | `PermissionPrompt.vue:274`；i18n `permissionPrompt.needsYourDecision` | 需数据（自动审批返回 `reviewer_uncertain`） |
| G-C1-15 | 「自动审批没有完成，请由你决定。」 | 提示块（浅底 50%，14px 灰） | 隐藏 | 自动审批失败（`reviewer_failed`） | `PermissionPrompt.vue:283`；i18n `permissionPrompt.autoReviewFailed` | 需数据（自动审批返回 `reviewer_failed`） |
| G-C1-16 | （提交错误） | 错误文字（14px 危险色） | 隐藏 | 选择动作失败 | `PermissionPrompt.vue:286` | 需交互（点任一动作后提交失败） |
| G-C1-17 | （无文案）动作按钮组 | 按钮组（右对齐，间距 8px） | — | 常驻 | `PermissionPrompt.vue:288` | 可见 A/C38/C39（随行；三个按钮同在 y=632 行） |
| G-C1-18 | 「仅这次」 | 主按钮（`default`，小号，28px 高） | — | 选项含一次性允许；成员审批时 `allow_all` 也显示此文案 | `PermissionPrompt.vue:137`；i18n `permissionPrompt.allowOnce` | 可见 A 827,632 58×28；C38/C39 516,632 58×28 |
| G-C1-19 | 「拒绝」 / 「不允许」 | 次按钮（`outline`，小号） | — | 普通工具为「拒绝」，沙箱场景为「不允许」 | `PermissionPrompt.vue:134`；i18n `permissionPrompt.reject`、`permissionPrompt.dontAllow` | 可见「拒绝」A 772,632 48×28；C38/C39 461,632 48×28；「不允许」需数据（沙箱变体，见 C2） |
| G-C1-20 | 「始终允许此命令」 / 「允许此会话期间访问」 / 「始终允许文件访问」 | 次按钮（`outline`，小号） | — | `allow_all` 时按工具切换：exec/execute → 命令；read/write/edit → 会话期间；其余 → 文件 | `PermissionPrompt.vue:147`–`:151`；i18n `permissionPrompt.alwaysAllowCommand`、`permissionPrompt.allowForSession`、`permissionPrompt.alwaysAllowFile` | 可见「始终允许此命令」A 891,632 109×28；C38/C39 580,632；另两文案需数据（read/write/edit 与其他工具的 `allow_all` 请求） |
| G-C1-21 | （按钮禁用态） | 按钮状态（50% 不透明） | 可用 | 提交中或整体禁用 | `PermissionPrompt.vue:293` | 需交互（点动作后提交中） |

元素数 21。

### C2 沙箱越权卡（内联）

条件：权限请求变体为 `sandbox`，工具被沙箱拦下后请求放宽访问范围。已采三份审批请求都是普通 `exec` 权限，不是沙箱拦下后的放宽请求，故本组全不可观察。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 | 已采状态 |
|---|---|---|---|---|---|---|
| G-C2-01 | （无文案）卡片容器 | 卡片（边框 + 内边距 16px） | 隐藏 | 有沙箱请求 | `PermissionPrompt.vue:302` | 需数据（权限请求 `variant=sandbox`） |
| G-C2-02 | 「网络访问已拦截」 / 「文件读取已拦截」 / 「文件写入已拦截」 / 「文件执行已拦截」 | 标题（16px 半粗） | — | 按资源类型与访问方式切换 | `PermissionPrompt.vue:104`、`:101`；i18n `permissionPrompt.sandboxNetworkBlocked`/`sandboxFileReadBlocked`/`sandboxFileWriteBlocked`/`sandboxFileExecuteBlocked` | 需数据（沙箱拦下网络 / 读 / 写 / 执行四种之一） |
| G-C2-03 | （域名或路径） | 等宽文字（14px，可断行） | — | 常驻 | `PermissionPrompt.vue:307` | 需数据（同上，带被拦的域名/路径） |
| G-C2-04 | 「{tool} 需要访问该域名。」 / 「需要读取该路径。」 / 「需要写入该路径。」 / 「需要执行该路径。」 | 说明文字（14px 灰） | — | 按资源类型与访问方式切换 | `PermissionPrompt.vue:116`；i18n `permissionPrompt.sandboxNetworkMessage`/`sandboxReadMessage`/`sandboxWriteMessage`/`sandboxExecuteMessage` | 需数据（同上） |
| G-C2-05 | （提交错误） | 错误文字 | 隐藏 | 选择失败 | `PermissionPrompt.vue:310` | 需交互（沙箱卡上点任一动作后提交失败） |
| G-C2-06 | 「不允许」 | 次按钮 | — | 常驻 | `PermissionPrompt.vue:134`；i18n `permissionPrompt.dontAllow` | 需数据（沙箱卡在场） |
| G-C2-07 | 「仅这次」 | 主按钮 | — | 常驻 | `PermissionPrompt.vue:137`；i18n `permissionPrompt.allowOnce` | 需数据（沙箱卡在场） |
| G-C2-08 | 「始终允许网络访问」 / 「始终允许文件访问」 | 次按钮 | — | 按资源类型切换 | `PermissionPrompt.vue:143`；i18n `permissionPrompt.alwaysAllowNetwork`、`permissionPrompt.alwaysAllowFile` | 需数据（沙箱卡在场，网络 / 文件两类） |

元素数 8。

### C3 子代理模型卡（内联）

条件：权限请求变体为 `subagent_model`，子任务要换用更贵的模型。已采态里没有任何子代理换模型请求。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 | 已采状态 |
|---|---|---|---|---|---|---|
| G-C3-01 | （无文案）卡片容器 | 卡片（边框 + 内边距 16px） | 隐藏 | 有子代理模型请求 | `PermissionPrompt.vue:324` | 需数据（权限请求 `variant=subagent_model`） |
| G-C3-02 | 「DimAgent 想要为子任务使用更贵的模型」 | 标题（16px 半粗） | — | 常驻 | `PermissionPrompt.vue:330`；i18n `permissionPrompt.subagentModel.title` | 需数据（同上） |
| G-C3-03 | 「子任务 · {taskTitle}」 | 说明行（14px 灰） | 隐藏 | 请求带任务标题 | `PermissionPrompt.vue:332`；i18n `permissionPrompt.subagentModel.task` | 需数据（请求带 taskTitle） |
| G-C3-04 | — 提供方图标 + （模型显示名）+ （提供方名称） | 区块（边框 + 浅底 30%，内边距 12px）内一行（20px 图标 + 名称 + 供方名） | — | 常驻 | `PermissionPrompt.vue:333`–`:341` | 需数据（同上，且模型有 provider 元数据） |
| G-C3-05 | 「价格高于当前模型」 | 标记（12px 琥珀色） | — | 常驻 | `PermissionPrompt.vue:346`；i18n `permissionPrompt.subagentModel.pricier` | 需数据（同上） |
| G-C3-06 | 「拒绝时子任务将使用当前模型 {model} 继续」 | 说明文字（12px 灰） | — | 常驻 | `PermissionPrompt.vue:351`；i18n `permissionPrompt.subagentModel.fallback` | 需数据（同上） |
| G-C3-07 | （提交错误） | 错误文字 | 隐藏 | 选择失败 | `PermissionPrompt.vue:354` | 需交互（点选项后提交失败） |
| G-C3-08 | 「使用当前模型」 | 次按钮 | — | 拒绝类选项（`reject_once`/`reject_always`） | `PermissionPrompt.vue:133`、`:126`；i18n `permissionPrompt.subagentModel.reject` | 需数据（同上） |
| G-C3-09 | 「本会话允许」 | 主按钮 | — | 允许类选项 | `PermissionPrompt.vue:136`；i18n `permissionPrompt.subagentModel.allow` | 需数据（同上） |

元素数 9。

### C4 审批输入区（按工具类型分渲染器）

条件：工具权限卡内嵌，按工具名与入参选择渲染器（命令 / 搜索 / 文件 / 网页 / 技能 / 子代理 / 浏览器 / CUA / 等待 / 通用）；旧版纯文本入参一律走通用渲染器。已采只命中**命令**渲染器（A 为 `ls -la` + `cwd`，C38/C39 为 `sleep 25` + `timeoutMs: 60 s`），其余九类需各自工具的权限请求。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 | 已采状态 |
|---|---|---|---|---|---|---|
| G-C4-01 | （无文案）容器 | 区块（最大高 240px，可滚动，边框 + 卡片底） | — | 渲染器命中 | `<ref>/components/chat-workspace/interactions/approval-panels/CommandApprovalPanel.vue:18`（各渲染器同构） | 可见 A（随行；容器未单列，按标签 387,552、命令行 397,579、参数行 397,596 反推 ≈387,545 613×70） |
| G-C4-02 | 「输入」 | 顶部标签（11px 灰，滚动时吸附） | — | 命令 / 搜索 / 网页搜索 / 行列表 / 通用渲染器 | `CommandApprovalPanel.vue:24`；i18n `timelineTool.input` | 可见 A 387,552 613×27；C38/C39 31,552 657×27 |
| G-C4-03 | 「$ {command}」 | 等宽代码行（`$` 灰色前缀，可断行） | — | 命中命令字段 | `CommandApprovalPanel.vue:26` | 可见 A 397,579 592×18（`ls -la`，`$` 在 397,581 6×14）；C38/C39 42,579（`sleep 25`） |
| G-C4-04 | 「cwd: {cwd}」 / 「timeoutMs: {值}」 / 「runInBackground: {true/false}」 | 等宽参数行组（缩进两格，70% 不透明） | 无 | 对应字段存在 | `CommandApprovalPanel.vue:27`–`:29` | 可见 `cwd` A 397,596 592×18；`timeoutMs` C38/C39 42,596 636×18（`timeoutMs: 60 s`）；`runInBackground` 需数据（后台执行的权限请求） |
| G-C4-05 | （文件名 / `patch.diff` / 工具名） | 顶部标签（11px 灰，截断） | 工具名 | 文件渲染器 | `FileApprovalPanel.vue:36` | 需数据（`read`/`write`/`edit`/`apply_patch` 的权限请求） |
| G-C4-06 | （编辑前/后） | 双栏代码块（无行号） | — | `edit` 且有 oldText 与 newText | `FileApprovalPanel.vue:38` | 需数据（`edit` 请求带 oldText+newText） |
| G-C4-07 | （写入内容） | 代码块（带行号） | — | `write` 且有 content | `FileApprovalPanel.vue:45` | 需数据（`write` 请求带 content） |
| G-C4-08 | （补丁文本） | 代码块（无行号） | — | `apply_patch`/`patch` 且有 patch | `FileApprovalPanel.vue:50` | 需数据（`apply_patch`/`patch` 请求带 patch） |
| G-C4-09 | 「replaceAll: {值}」 / 「offset/limit/encoding: {值}」 | 等宽参数行组 | 无 | 对应字段存在 | `FileApprovalPanel.vue:56`、`:58` | 需数据（文件类请求带这些字段） |
| G-C4-10 | 「pattern: {值}」 / 「query: {值}」 / 「path: {值}」 | 等宽主值行（90% 不透明） | — | 搜索/列举渲染器 | `SearchApprovalPanel.vue:33`–`:38` | 需数据（`grep`/`glob` 的权限请求） |
| G-C4-11 | 「path: {值}」 | 等宽次要行（70% 不透明） | 无 | 有路径且主值不是路径 | `SearchApprovalPanel.vue:34` | 需数据（同上且带 path） |
| G-C4-12 | 「include/glob/outputMode/beforeContext/afterContext/context/-B/-A/-C/lineNumbers/-n/literal/limit: {值}」 | 等宽参数行组 | 无 | 对应字段存在 | `SearchApprovalPanel.vue:35` | 需数据（检索请求带这些参数） |
| G-C4-13 | 「query: {值}」 | 等宽主值行 | — | 网页搜索 | `WebApprovalPanel.vue:30` | 需数据（`web_search` 的权限请求） |
| G-C4-14 | 「page/pageSize: {值}」 | 等宽参数行组 | 无 | 网页搜索且带分页 | `WebApprovalPanel.vue:31`、`:44` | 需数据（同上带分页） |
| G-C4-15 | （URL） | 顶部标签（11px 灰，截断） | — | 网页抓取 | `WebApprovalPanel.vue:36` | 需数据（`web_fetch` 的权限请求） |
| G-C4-16 | 「startIndex/contextMaxCharacters/mode: {值}」 | 等宽参数行组 | 无 | 网页抓取且有附加参数 | `WebApprovalPanel.vue:39` | 需数据（同上带附加参数） |
| G-C4-17 | 「skill_name: {名称}」 | 顶部标签 + 等宽行 | — | 技能渲染器 | `SkillApprovalPanel.vue:11` | 需数据（`skill` 的权限请求） |
| G-C4-18 | （子代理创建头行：图标 + 名称 + 任务标题 + 模型 id） | 行（14px 图标 + 三段文字） | — | 子代理创建类动作（`create`/`create_external`） | `AgentApprovalPanel.vue:30`–`:38` | 需数据（`agent` 创建类动作的权限请求） |
| G-C4-19 | （子代理 prompt） | 等宽文本块（可断行） | — | 有 prompt | `AgentApprovalPanel.vue:41` | 需数据（同上带 prompt） |
| G-C4-20 | 「effort/forkContext: {值}」 | 等宽参数行组 | 无 | 有对应字段 | `AgentApprovalPanel.vue:42` | 需数据（同上带这些字段） |
| G-C4-21 | 「action/taskId/agentName/taskTitle/agentType/providerId/modelId/effort/reason: {值}」 | 顶部标签 + 行列表 | — | 非创建类子代理动作 | `AgentApprovalPanel.vue:46`、`ApprovalInputLines.vue:18` | 需数据（`agent` 的 list/cancel/query 类动作的权限请求） |
| G-C4-22 | 「action/url/ref/value/key/direction/amount/type/timeout/expression/method/params/selector/interactiveOnly/compact/fullPage/clear: {值}」 | 顶部标签 + 行列表 | — | 浏览器渲染器 | `BrowserApprovalPanel.vue:8` | 需数据（`browser` 工具存在并被请求；本机浏览器能力经 MCP 暴露） |
| G-C4-23 | 「tool: {tool}」 + 全部入参行 | 顶部标签 + 行列表 | — | CUA 渲染器（工具名的 `cua-driver__` 前缀被去掉） | `CuaApprovalPanel.vue:24`–`:27` | 需数据（接入 cua-driver 工具） |
| G-C4-24 | 「taskId: {id}」 / 「timeoutMs: {值}」 | 顶部标签 + 行列表 | — | 等待渲染器 | `WaitApprovalPanel.vue:16` | 需数据（`wait` 工具的权限请求） |
| G-C4-25 | （工具名 + 入参 JSON） | 顶部标签（等宽 11px）+ 等宽代码块（两空格缩进） | — | 通用渲染器；旧版纯文本入参以 `{input: 文本}` 走同一块 | `GenericApprovalPanel.vue:17`、`:21`、`ApprovalInputPanel.vue:24` | 需数据（`recall`/`settings`/`plugin`/`exec_stop`/MCP 等未登记工具的权限请求） |

元素数 25。

区域 C 合计 63。

---

## 4. D 提问

### D1 提问卡（elicitation，内联）

条件：会话收到 `elicitation` 请求；多字段时逐题展示；贴输入区上沿，可收起；带截止时间时显示倒计时。

采集现状：60 份态里没有一份停在待答态。唯一旁证是 `runtime/session/turns/11-tool-ask-user-pending.png`（面板外观可见），但该态没有 DOM 导出，无法给出节点、文案与包围盒；`10-tool-ask-user-timeout.png` 是超时后的时间线行，不是本卡。故本组 22 项全判不可观察。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 | 已采状态 |
|---|---|---|---|---|---|---|
| G-D1-01 | （无文案）卡片容器 | 卡片（圆角 10px，边框，纵向三段：头 / 可滚动体 / 脚） | 隐藏 | 有提问请求 | `<ref>/components/chat-workspace/interactions/ElicitationPrompt.vue:436`、`:438` | 可见 Y16 ≈370,425 690×235（由标题 386,452 与按钮行 636 反推） |
| G-D1-02 | （当前字段标题） | 标题（14px 半粗，截断） | — | 取字段 title → schema title → 回退 | `ElicitationPrompt.vue:451` | 可见 Y16 386,452 49×18（`单选测试`） |
| G-D1-03 | 「需要你的回答」 | 标题回退文案 | — | 字段与 schema 都无标题 | `ElicitationPrompt.vue:165`；i18n `elicitation.defaultTitle` | 需数据（同上，且无标题可回退） |
| G-D1-04 | 「{当前}/{总数}」 | 进度徽标（灰底 10px 圆角小字） | — | 字段数 > 1 | `ElicitationPrompt.vue:453` | 需数据（一次 elicitation 带 ≥2 个字段） |
| G-D1-05 | 「待选择」 / 「待选择 · {count} 个选项」 / 「已选：{answer}」 / 「已选 {count} 项」 | 收起态状态行（12px 灰，截断） | 隐藏 | 卡片收起时按已答情况切换 | `ElicitationPrompt.vue:171`；i18n `elicitation.awaitingAnswer`/`optionCount`/`selectedAnswer`/`selectedCount` | 可见 Y17 386,650 258×14（`待选择 · 3 个选项`）；其余三分支按已答情况仍需交互 |
| G-D1-06 | 「剩余 {m:ss}」 | 倒计时（12px 等宽数字，`role=timer`） | 无 | 请求带 `deadlineAt` 且未被作答清除 | `ElicitationPrompt.vue:246`；i18n `elicitation.timeLeft` | 需数据（请求带 `deadlineAt`） |
| G-D1-07 | 「收起」 / 「展开」 | 文字小按钮（带下箭头，收起时旋转 180°） | 「收起」 | 常驻 | `ElicitationPrompt.vue:487`；i18n `elicitation.collapse`、`elicitation.expand` | 可见 Y16/Y17 654,633 61×28（展开/收起切换） |
| G-D1-08 | （提问正文） | 多行文字（13px，保留换行） | 无 | 请求带 message | `ElicitationPrompt.vue:499` | 可见 Y16 386,485 330×18（`请选择一项（A/B/C）。`） |
| G-D1-09 | （字段描述） | 说明文字（14px 灰，保留换行） | 无 | 字段带 description | `ElicitationPrompt.vue:507` | 需数据（字段带 description） |
| G-D1-10 | 「筛选选项…」 | 输入框（placeholder） | 空 | 当前字段选项数 > 8 | `ElicitationPrompt.vue:522`；i18n `elicitation.filterPlaceholder` | 需数据（当前字段 >8 个选项） |
| G-D1-11 | （单选项条目） | 行 + 单选圈（选中：主色边框 + 浅主色底） | 未选 | 字段为 string 且带 `oneOf` | `ElicitationPrompt.vue:527`、`:544` | 可见（仅视觉）Y16 A/B/C 三行单选圈；Y18 选中 A 后为主色边框 + 浅主色底（PNG） |
| G-D1-12 | （多选项条目） | 行 + 方形勾选框（选中显示对勾） | 未选 | 字段为 array 且带 `anyOf` | `ElicitationPrompt.vue:598`、`:603` | 需数据（字段为 `anyOf` 数组） |
| G-D1-13 | 「没有匹配的选项」 | 空态文字（14px 灰） | 隐藏 | 筛选后无匹配 | `ElicitationPrompt.vue:561`、`:616`；i18n `elicitation.noMatch` | 需交互（在筛选框键入无命中词） |
| G-D1-14 | 「每行输入一个值」 | 多行输入框（1 行起，最多 4 行后内部滚动） | 空 | 字段为无选项数组 | `ElicitationPrompt.vue:630`；i18n `elicitation.freeformPlaceholder` | 需数据（数组字段无 `anyOf`） |
| G-D1-15 | 「自定义答案」 + 「输入自定义答案」 | 标签（12px 灰）+ 多行输入框 | 空 | 字段允许自定义且有选项 | `ElicitationPrompt.vue:650`、`:665`；i18n `elicitation.customAnswer`、`elicitation.customAnswerPlaceholder` | 需数据（字段 `allowCustom` 且有选项） |
| G-D1-16 | 「自定义选项」 + 「每行输入一个自定义值」 | 标签 + 多行输入框 | 空 | 数组字段允许自定义 | `ElicitationPrompt.vue:673`、`:687`；i18n `elicitation.customOptions`、`elicitation.customOptionsPlaceholder` | 需数据（数组字段 `allowCustom`） |
| G-D1-17 | 「请先回答所有必填问题」 / （服务端错误） | 错误文字（14px 危险色） | 隐藏 | 必填未答时提交 / 提交失败 | `ElicitationPrompt.vue:353`；i18n `elicitation.requiredMissing` | 需交互（必填未答时点「提交」，或提交失败） |
| G-D1-18 | 「拒绝」 | 次按钮（`outline`，小号，左置） | — | 常驻 | `ElicitationPrompt.vue:708`；i18n `elicitation.decline` | 可见 Y16 386,636 48×28（`拒绝`） |
| G-D1-19 | 「上一题」 | 次按钮（小号） | 隐藏 | 多字段且非首题 | `ElicitationPrompt.vue:718`；i18n `elicitation.previous` | 需数据（≥2 字段且已走到第 2 题之后） |
| G-D1-20 | 「下一题」 | 主按钮（小号） | 隐藏 | 多字段且非末题 | `ElicitationPrompt.vue:726`；i18n `elicitation.next` | 需数据（≥2 字段且未到末题） |
| G-D1-21 | 「提交」 | 主按钮（小号） | — | 单字段或已是末题 | `ElicitationPrompt.vue:734`；i18n `elicitation.submit` | 可见 Y16 670,636 46×28（`提交`） |
| G-D1-22 | （按钮禁用态） | 按钮状态（50% 不透明） | 可用 | 提交中；单字段且必填未满足时「提交」禁用 | `ElicitationPrompt.vue:717`、`:731` | 可见（仅视觉）Y16 未选时 `提交` 置灰；Y18 选中后转实心（PNG） |

元素数 22。

区域 D 合计 22。

---

## 5. E 确认

### E1 编辑运行中消息确认

条件：会话仍在运行时，提交对已发送消息的编辑。入口已采（C28 行内编辑器），确认框本身未采。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 | 已采状态 |
|---|---|---|---|---|---|---|
| G-E1-01 | 「会话正在运行」 | 标题 | — | 打开时 | `<ref>/components/chat-workspace/interactions/EditRunningMessageDialog.vue:13`；i18n `chatWorkspace.editBusyTitle` | 需交互（会话运行中 → 悬停已发消息点铅笔（C28 已采 386,581 行内编辑器）→ 提交编辑） |
| G-E1-02 | 「当前会话仍在运行中。是否停止本次运行并发送编辑后的消息？」 | 描述 | — | 打开时 | `EditRunningMessageDialog.vue:15`；i18n `chatWorkspace.editBusyDescription` | 需交互（同上，且该轮仍在跑） |
| G-E1-03 | 「取消」 | 次按钮（`alert-dialog` 取消位） | — | 常驻 | `EditRunningMessageDialog.vue:20`；i18n `common.cancel` | 需交互（同上） |
| G-E1-04 | 「停止并发送」 | 主按钮 | — | 常驻 | `EditRunningMessageDialog.vue:23`；i18n `chatWorkspace.editBusyStopAndSend` | 需交互（同上） |

元素数 4。

### E2 删除会话

条件：单条删除（任意时刻）或批量删除（多选后）；标题与描述随条数切换。入口已采，对话框未采。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 | 已采状态 |
|---|---|---|---|---|---|---|
| G-E2-01 | 「删除会话」 / 「删除 {count} 个会话」 | 标题 | — | 单条 / 多条 | `<ref>/components/sidebar/session/dialogs/DeleteSessionDialog.vue:68`；i18n `sessionDialogs.deleteSessionTitle`、`deleteSessionsTitle` | 可见 X13 341,311 348×16（`删除会话`） |
| G-E2-02 | 「确定要永久删除“{title}”吗？删除后会话将立即从侧边栏移除，且无法恢复。」 / 「确定要永久删除以下会话吗？{names}」 | 描述 | — | 单条 / 多条；`{names}` 最多三名 + 「等 N 个」 | `DeleteSessionDialog.vue:76`、`:77`；i18n `sessionDialogs.deleteSessionDescription`、`deleteSessionsDescription`、`namesMore` | 可见 X13 341,334 348×35（`确定要永久删除“帮我列出当前目录的文件”吗？…`） |
| G-E2-03 | （失败原因） | 错误文字（14px 危险色） | 隐藏 | 删除失败且为单条 | `DeleteSessionDialog.vue:81` | 需交互（删除请求返回错误） |
| G-E2-04 | 「取消」 | 次按钮（`outline`） | — | 常驻 | `DeleteSessionDialog.vue:85`；i18n `common.cancel` | 可见 X13 575,383 55×32（`取消`） |
| G-E2-05 | 「删除」 / 「删除 ({count})」 / 「删除中...」 | 危险按钮（`destructive`） | — | 单条 / 多条 / 提交中（提交中禁用） | `DeleteSessionDialog.vue:90`；i18n `common.delete`、`deleteMany`、`deleting` | 可见 X13 637,383 53×32（`删除`） |

元素数 5。

### E3 归档会话

条件：批量归档，或归档单条仍在运行的会话（空闲单条不弹此框，直接归档）。入口已采，对话框未采。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 | 已采状态 |
|---|---|---|---|---|---|---|
| G-E3-01 | 「归档 {count} 个会话」 | 标题 | — | 打开时 | `<ref>/components/sidebar/ArchiveThreadDialog.vue:67`；i18n `sessionDialogs.archiveSessionsTitle` | 需交互（C13「归档」157,388 / C45「归档会话」169,406；多条入口） |
| G-E3-02 | 「确定要归档以下会话吗？{names}」 | 描述 | — | 打开时；`{names}` 最多三名 + 「等 N 个」 | `ArchiveThreadDialog.vue:70`；i18n `sessionDialogs.archiveSessionsDescription` | 需交互（同上） |
| G-E3-03 | 「正在运行的会话会被中断，其队列中的后续消息也会被丢弃。」 | 警告文字（14px 警示色） | 隐藏 | 目标中有运行中的会话 | `ArchiveThreadDialog.vue:79`；i18n `sessionDialogs.archiveSessionsLoopingWarning` | 需数据（归档目标里有正在运行的会话） |
| G-E3-04 | （失败原因） | 错误文字 | 隐藏 | 归档失败 | `ArchiveThreadDialog.vue:82` | 需交互（归档请求返回错误） |
| G-E3-05 | 「取消」 | 次按钮 | — | 常驻 | `ArchiveThreadDialog.vue:86`；i18n `common.cancel` | 需交互（同上） |
| G-E3-06 | 「归档 ({count})」 / 「归档中...」 | 主按钮 | — | 常驻 / 提交中（提交中禁用） | `ArchiveThreadDialog.vue:90`、`:89`；i18n `sessionDialogs.archiveMany`、`archiving` | 需交互（同上；提交中为瞬态） |

元素数 6。

### E4 删除工作区会话

条件：在侧栏工作区分组上执行删除；展示该工作区下将被删除的会话条数。入口已采，对话框未采。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 | 已采状态 |
|---|---|---|---|---|---|---|
| G-E4-01 | 「删除工作区会话」 | 标题 | — | 打开时 | `<ref>/components/sidebar/session/dialogs/DeleteProjectDialog.vue:34`；i18n `sessionDialogs.deleteProjectTitle` | 可见 Y14 341,304 348×16（`删除工作区会话`） |
| G-E4-02 | 「将删除“{name}”下的 {count} 条会话。」 | 描述 | — | 打开时 | `DeleteProjectDialog.vue:36`；i18n `sessionDialogs.deleteProjectDescription` | 可见 Y14 341,327 348×18（`将删除“year”下的 0 条会话。`） |
| G-E4-03 | （工作区路径） | 文字行（14px 灰） | — | 打开时 | `DeleteProjectDialog.vue:40` | 可见 Y14 341,358 348×18（`/Users/…/Documents/year`） |
| G-E4-04 | （失败原因） | 错误文字 | 隐藏 | 删除失败 | `DeleteProjectDialog.vue:41` | 需交互（删除请求返回错误） |
| G-E4-05 | 「取消」 | 次按钮 | — | 常驻 | `DeleteProjectDialog.vue:45`；i18n `common.cancel` | 可见 Y14 ≈341,395（`取消`） |
| G-E4-06 | 「删除」 | 危险按钮 | — | 常驻 | `DeleteProjectDialog.vue:48`；i18n `common.delete` | 可见 Y14 ≈341,395（`删除`，危险色） |

元素数 6。

区域 E 合计 21。

---

## 6. F 重命名

### F1 会话重命名

条件：侧栏会话条目菜单；需 SessionTitle 方法就绪。

采集现状有一条硬阻塞：本机 `SessionTitle` 不可用，C2/C13 的重命名菜单项直接渲染成「重命名（SessionTitle 不可用）」（C2 362,76 159×63、C13 157,296 187×28）且不可点，因此本对话框在本环境无法通达，不是"没点进去"。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 | 已采状态 |
|---|---|---|---|---|---|---|
| G-F1-01 | 「重命名会话」 | 标题 | — | 打开时 | `<ref>/components/sidebar/RenameThreadDialog.vue:125`；i18n `sessionDialogs.renameThreadTitle` | 需交互（先恢复 SessionTitle 能力，再从 C13/C2 点「重命名」；当前两项均显示「（SessionTitle 不可用）」） |
| G-F1-02 | 「输入新的会话名称」 | 描述 | — | 打开时 | `RenameThreadDialog.vue:127`；i18n `sessionDialogs.renameThreadDescription` | 需交互（同上，前置 SessionTitle 就绪） |
| G-F1-03 | 「会话名称」 | 字段标签（14px 中等） | — | 打开时 | `RenameThreadDialog.vue:133`；i18n `sessionDialogs.threadName` | 需交互（同上） |
| G-F1-04 | 「输入会话名称」 | 输入框（placeholder，聚焦即全选） | 当前标题 | 打开时 | `RenameThreadDialog.vue:139`；i18n `sessionDialogs.threadNamePlaceholder` | 需交互（同上） |
| G-F1-05 | 「最多使用 {max} 个字符」 / 「请输入不同的会话名称」 / 「会话名称中不能包含控制字符」 / 「会话名称不能为空」 | 校验提示（12px 灰） | 隐藏 | 依次为超 80 字符 / 与当前相同 / 含控制字符或行分隔符 / 去空白后为空 | `RenameThreadDialog.vue:168`–`:148`；i18n `sessionDialogs.renameTooLong`、`renameUnchanged`、`renameControlCharacters`、`rpcErrors.requiredString` | 需交互（对话框打开后键入 81 字符 / 原样 / 控制字符 / 清空，四分支各需一次） |
| G-F1-06 | 「正在检查 SessionTitle…」 / 「SessionTitle 不可用」 | 就绪提示（12px 灰） | 隐藏 | 能力探测中 / 能力不可用 | `RenameThreadDialog.vue:160`；i18n `sessionDialogs.renameChecking`、`renameUnavailable` | 需交互（同上；本机只能验证"不可用"这一结果的旁证——菜单项文案，非本行节点） |
| G-F1-07 | （重命名失败原因） | 错误文字（12px 危险色，`role=alert`） | 隐藏 | 提交失败 | `RenameThreadDialog.vue:152` | 需交互（提交后 RPC 失败） |
| G-F1-08 | 「取消」 | 次按钮（`outline`） | — | 常驻 | `RenameThreadDialog.vue:186`；i18n `common.cancel` | 需交互（同上） |
| G-F1-09 | 「确认」 | 主按钮 | 禁用 | 校验通过且能力就绪且非提交中才可用 | `RenameThreadDialog.vue:195`；i18n `common.confirm` | 需交互（同上） |

元素数 9。

### F2 工作区重命名

条件：侧栏工作区分组菜单；改的是本地项目别名（不依赖 SessionTitle）。入口已采，对话框未采。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 | 已采状态 |
|---|---|---|---|---|---|---|
| G-F2-01 | 「重命名工作区」 | 标题 | — | 打开时 | `<ref>/components/sidebar/session/dialogs/RenameProjectDialog.vue:81`；i18n `sessionDialogs.renameProjectTitle` | 可见 Y15 341,258 348×16（`重命名工作区`） |
| G-F2-02 | 「输入新的工作区名称」 | 描述 | — | 打开时 | `RenameProjectDialog.vue:83`；i18n `sessionDialogs.renameProjectDescription` | 可见 Y15 341,280 348×18（`输入新的工作区名称`） |
| G-F2-03 | 「工作区名称」 | 字段标签 | — | 打开时 | `RenameProjectDialog.vue:89`；i18n `sessionDialogs.projectName` | 可见 Y15 341,326 348×12（`工作区名称`） |
| G-F2-04 | 「输入工作区名称」 | 输入框（placeholder，聚焦即全选） | 当前别名 | 打开时 | `RenameProjectDialog.vue:95`；i18n `sessionDialogs.projectNamePlaceholder` | 可见 Y15 ≈341,350 348×40（输入框，预填 `year` 且全选，placeholder `输入工作区名称`） |
| G-F2-05 | （工作区路径） | 文字行（14px 灰） | — | 打开时 | `RenameProjectDialog.vue:106` | 可见 Y15 341,405 348×18（工作区路径） |
| G-F2-06 | 「取消」 | 次按钮 | — | 常驻 | `RenameProjectDialog.vue:110`；i18n `common.cancel` | 可见 Y15 ≈575,436（`取消`） |
| G-F2-07 | 「确认」 | 主按钮 | 禁用 | 非空且与当前不同才可用 | `RenameProjectDialog.vue:118`；i18n `common.confirm` | 可见（仅视觉）Y15 637,436 53×32（`确认`；名称未改时置灰） |

元素数 7。

区域 F 合计 16。

---

## 7. G 导出与 Fork

### G1 导出对话

条件：会话菜单「导出对话」；先取回合大纲，再按回合勾选导出。入口已采（C13 157,352、C2 362,167），对话框未采。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 | 已采状态 |
|---|---|---|---|---|---|---|
| G-G1-01 | 「导出对话」 | 标题 | — | 打开时 | `<ref>/components/sidebar/session/dialogs/ExportSessionDialog.vue:131`；i18n `sessionDialogs.exportSessionTitle` | 可见 X12 243,219 544×16（`导出对话`） |
| G-G1-02 | 「导出「{name}」的会话对话。不包含原始 Provider 请求。」 | 描述（最多 3 行，超出截断） | — | 打开时 | `ExportSessionDialog.vue:133`；i18n `sessionDialogs.exportSessionDescription` | 可见 X12 243,242 544×18（`导出「帮我列出当前目录的文件」的会话对话。不包含原始 Provider 请求。`） |
| G-G1-03 | 「导出内容」 | 分组标签（14px 中等） | — | 打开时 | `ExportSessionDialog.vue:138`；i18n `sessionDialogs.exportContent` | 可见 X12 243,273 544×18（`导出内容`） |
| G-G1-04 | 「压缩记录」 / 「工具详情」 / 「思考」 | 勾选项行（方框 + 14px 文字） | 查实现默认（首屏读代码为未勾选） | 常驻 | `ExportSessionDialog.vue:146`–`:160`；i18n `sessionDialogs.exportIncludeCompaction`、`exportIncludeToolDetails`、`exportIncludeThinking` | 可见 X12 264,298（`压缩记录` 49×18 / `工具详情` 355,298 / `思考` 446,298，三者默认已勾） |
| G-G1-05 | 「消息」 | 分组标签（14px 中等） | — | 打开时 | `ExportSessionDialog.vue:169`；i18n `sessionDialogs.exportMessages` | 可见 X12 243,329 25×18（`消息`） |
| G-G1-06 | 「保留 {kept} / {total} 个回合」 | 计数文字（12px 灰） | 隐藏 | 大纲已载入 | `ExportSessionDialog.vue:171`；i18n `sessionDialogs.exportKeptCount` | 需数据（回合大纲返回） |
| G-G1-07 | 「搜索回合」 | 搜索输入框（左置放大镜，placeholder） | 空 | 大纲已载入 | `ExportSessionDialog.vue:181`；i18n `sessionDialogs.exportSearchPlaceholder` | 需数据（同上） |
| G-G1-08 | 「排除 {count} 个匹配」 | 次级小按钮（`outline`） | 隐藏 | 搜索有匹配；匹配为 0 时禁用 | `ExportSessionDialog.vue:193`；i18n `sessionDialogs.exportExcludeMatches` | 需交互（大纲载入后键入命中词） |
| G-G1-09 | 「正在读取对话…」 / 「读取会话对话失败」 / 「该对话无法按回合拆分，因此无法过滤。导出文件仅包含对话头信息。」 / 「没有回合匹配此搜索。」 | 状态文字（居中，上下 24px） | — | 依次为：大纲加载中 / 加载失败（危险色）/ 对话不可按回合拆分 / 搜索无匹配 | `ExportSessionDialog.vue:198`–`:211`；i18n `sessionDialogs.exportOutlineLoading`、`exportOutlineFailed`、`exportNotFilterable`、`exportNoMatches` | 需数据（大纲加载中/失败/不可拆分）+ 需交互（搜索无匹配） |
| G-G1-10 | （回合条目：序号 + 预览 + 回复摘要） | 勾选行（方框 + 展开箭头 + 序号 + 预览 + 「回复 · {tools} 个工具 · {chars}k」） | 勾选 | 大纲已载入 | `ExportSessionDialog.vue:219`–`:250`；i18n `sessionDialogs.exportReplySummary` | 需数据（大纲返回且含回合） |
| G-G1-11 | （回合正文） | 等宽预格式块（最大高 256px） | 隐藏 | 该回合已展开；取文过程中显示「加载中…」 | `ExportSessionDialog.vue:253`；i18n `sessionDialogs.exportTurnLoading` | 需交互（展开某回合条目） |
| G-G1-12 | 「全部保留」 / 「全部排除」 | 文字按钮（12px 灰） | — | 有回合 | `ExportSessionDialog.vue:264`；i18n `sessionDialogs.exportKeepAll`、`exportKeepNone` | 需数据（大纲含回合） |
| G-G1-13 | 「压缩记录会单独描述被折叠的回合，排除某个回合不会将其从压缩记录中移除。」 | 说明文字（12px 灰） | 隐藏 | 勾选压缩记录且存在压缩记录 | `ExportSessionDialog.vue:284`；i18n `sessionDialogs.exportCompactionNote` | 需交互（勾选「压缩记录」）+ 需数据（会话有压缩记录） |
| G-G1-14 | 「会话快照已过期并重新加载。请检查消息和排除项后再次导出。」 | 提示文字（14px，`role=status`） | 隐藏 | 快照过期并重载后 | `ExportSessionDialog.vue:288`；i18n `sessionDialogs.exportOutlineRecovered` | 需数据（导出时会话快照过期并重载） |
| G-G1-15 | （导出失败原因） | 错误文字 | 隐藏 | 导出失败 | `ExportSessionDialog.vue:290` | 可见 X12 243,392 544×60（原始错误行 `BAD_REQUEST: History cursor exceeds fixed position`） |
| G-G1-16 | 「取消」 | 次按钮 | — | 常驻 | `ExportSessionDialog.vue:294`；i18n `common.cancel` | 可见 X12 673,475 55×32（`取消`） |
| G-G1-17 | 「导出」 / 「导出中…」 | 主按钮 | — | 常驻 / 提交中；未取到大纲时禁用 | `ExportSessionDialog.vue:302`；i18n `sessionDialogs.export`、`exporting` | 可见（仅视觉）X12 735,475 53×32（`导出`；大纲失败时置灰，PNG） |

元素数 17。

### G2 Fork 会话

条件：会话已在另一个窗口打开时尝试打开。入口已采（C13「在新窗口中打开」157,324），对话框未采。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 | 已采状态 |
|---|---|---|---|---|---|---|
| G-G2-01 | 「会话正在使用中」 | 标题 | — | 打开时 | `<ref>/components/sidebar/session/dialogs/ForkSessionDialog.vue:51`；i18n `sessionDialogs.forkSessionTitle` | 需交互（在新窗口打开同一会话 → 在另一窗口再点该会话：C13「在新窗口中打开」157,324 / C2 362,139） |
| G-G2-02 | 「“{title}”当前正在 {heldBy} 中打开。」 | 描述 | — | 打开时 | `ForkSessionDialog.vue:33`；i18n `sessionDialogs.forkSessionDescription` | 需交互（同上；`{heldBy}` 需另一窗口的持有者标识） |
| G-G2-03 | 「Fork 将创建一个包含完整消息记录的独立副本，原会话不受影响。」 | 说明文字（14px 灰） | — | 打开时 | `ForkSessionDialog.vue:57`；i18n `sessionDialogs.forkSessionExplanation` | 需交互（同上） |
| G-G2-04 | 「取消」 | 次按钮（`outline`） | — | 常驻 | `ForkSessionDialog.vue:63`；i18n `common.cancel` | 需交互（同上） |
| G-G2-05 | 「Fork」 | 主按钮（提交中前置旋转图标） | — | 常驻 | `ForkSessionDialog.vue:67`；i18n `sessionDialogs.forkSessionAction` | 需交互（同上） |

元素数 5。

区域 G 合计 22。

---

## 8. H 导入与更新

### H1 添加插件（本地文件夹 / Git）

条件：设置或插件页触发；两种来源模式共用同一对话框。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 | 已采状态 |
|---|---|---|---|---|---|---|
| G-H1-01 | 「从 Git 添加插件」 | 标题 | — | 打开时 | `<ref>/components/plugins/PluginInstallDialog.vue:79`；i18n `plugins.dialogTitle` | 可见 Y8 327,216 376×16（`从 Git 添加插件`） |
| G-H1-02 | 「从 GitHub 仓库或 Git URL 添加。」 | 描述 | — | 打开时 | `PluginInstallDialog.vue:80`；i18n `plugins.dialogDescription` | 可见 Y8 ≈327,242（`从 GitHub 仓库或 Git URL 添加。`） |
| G-H1-03 | 「本地文件夹」 / 「Git 仓库」 | 模式切换（两个等宽小按钮，选中为 `secondary`） | 「本地文件夹」 | 常驻 | `PluginInstallDialog.vue:92`–`:96`；i18n `plugins.native.path`、`plugins.native.git` | 可见 Y8 327,271 184×28（`本地文件夹` 选中）/ `Git 仓库` 并列 |
| G-H1-04 | 「来源」 + 「/path/to/plugin」 | 字段标签 + 输入框 | 空 | 本地文件夹模式 | `PluginInstallDialog.vue:106`–`:109`；i18n `plugins.sourceLabel`、`plugins.native.pathPlaceholder` | 可见 Y8 327,316 25×15（`来源`）+ 输入框 placeholder `/path/to/plugin` |
| G-H1-05 | 「选择文件夹」 | 图标按钮（文件夹图标，`outline`） | — | 本地文件夹模式 | `PluginInstallDialog.vue:114`；i18n `plugins.native.pickPath` | 可见 Y8 ≈560,350（文件夹图标按钮） |
| G-H1-06 | 「Git 仓库」 + 「owner/repo 或 https://github.com/owner/repo」 | 字段标签 + 输入框 | 空 | Git 模式 | `PluginInstallDialog.vue:122`–`:124`；i18n `plugins.native.git`、`plugins.native.gitPlaceholder` | 可见 Y9 ≈327,349（`Git 仓库` 标签 + `owner/repo 或 https://github.com/owner/repo` 输入框） |
| G-H1-07 | 「Git ref」 + 「main」 | 字段标签 + 输入框 | 空 | Git 模式 | `PluginInstallDialog.vue:131`–`:134`；i18n `plugins.refLabel`、`plugins.native.refPlaceholder` | 可见 Y9 327,349 36×15（`Git ref`）+ `main` 输入框 |
| G-H1-08 | 「插件 id（可选）」 + 「根据 manifest 自动生成」 | 字段标签 + 输入框 | 空 | 两种模式 | `PluginInstallDialog.vue:142`–`:146`；i18n `plugins.native.idLabel`、`plugins.native.idPlaceholder` | 可见 Y8 327,383 87×15（`插件 id（可选）` + `根据 manifest 自动生成`） |
| G-H1-09 | 「安装后启用」 | 开关（右侧） | 开 | 常驻 | `PluginInstallDialog.vue:152`；i18n `plugins.native.enableAfterInstall` | 可见 Y8 327,446 61×18（`安装后启用` 开关，默认开） |
| G-H1-10 | 「请选择插件文件夹。」 / 「请输入 Git 仓库。」 / （服务端错误） | 错误文字（14px 危险色） | 隐藏 | 校验失败 / 安装失败 | `PluginInstallDialog.vue:156`；i18n `plugins.native.pathRequired`、`plugins.native.gitRequired` | 需交互（空路径提交 / 空 Git 仓库提交 / 安装 RPC 失败） |
| G-H1-11 | 「取消」 | 次按钮（`outline`） | — | 常驻；安装中禁用 | `PluginInstallDialog.vue:161`；i18n `common.cancel` | 可见 Y8 ≈575,477（`取消`） |
| G-H1-12 | 「添加插件」 | 主按钮（安装中前置旋转图标） | — | 常驻；安装中禁用 | `PluginInstallDialog.vue:165`；i18n `plugins.add` | 可见 Y8 626,477 77×32（`添加插件`） |

元素数 12。

### H2 市场安装（选择内置技能）

条件：插件市场条目点安装；先取条目详情。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 | 已采状态 |
|---|---|---|---|---|---|---|
| G-H2-01 | 「安装 {name}」 | 标题 | — | 打开时 | `<ref>/components/plugins/PluginMarketplaceInstallDialog.vue:132`；i18n `plugins.marketplaceInstallTitle` | 需数据（插件市场不可用：`插件市场` 页显示「Marketplace 暂不可用 / 插件市场需要上游 API」，本机无法打开该对话框；非采集操作缺口） |
| G-H2-02 | 「选择要启用的内置技能。默认全选。」 | 描述 | — | 打开时 | `PluginMarketplaceInstallDialog.vue:133`；i18n `plugins.marketplaceInstallDescription` | 需数据（插件市场不可用：`插件市场` 页显示「Marketplace 暂不可用 / 插件市场需要上游 API」，本机无法打开该对话框；非采集操作缺口） |
| G-H2-03 | 「正在加载...」 | 加载行（旋转图标 + 14px 灰字，居中） | 隐藏 | 条目详情加载中 | `PluginMarketplaceInstallDialog.vue:138`；i18n `common.loading` | 需数据（插件市场不可用：`插件市场` 页显示「Marketplace 暂不可用 / 插件市场需要上游 API」，本机无法打开该对话框；非采集操作缺口） |
| G-H2-04 | （错误消息 + 「重试」） | 错误块（告警图标 + 14px 灰字 + 次级按钮） | 隐藏 | 详情加载失败 | `PluginMarketplaceInstallDialog.vue:149`–`:178`；i18n `common.retry` | 需数据（插件市场不可用：`插件市场` 页显示「Marketplace 暂不可用 / 插件市场需要上游 API」，本机无法打开该对话框；非采集操作缺口） |
| G-H2-05 | 「技能」 | 分组标签（12px 中等灰） | — | 详情已载入 | `PluginMarketplaceInstallDialog.vue:153`；i18n `plugins.marketplaceSkills` | 需数据（插件市场不可用：`插件市场` 页显示「Marketplace 暂不可用 / 插件市场需要上游 API」，本机无法打开该对话框；非采集操作缺口） |
| G-H2-06 | （技能名称 + 描述） | 勾选行（方框 + 名称 + 12px 描述） | 全选 | 插件含技能 | `PluginMarketplaceInstallDialog.vue:160`–`:199` | 需数据（插件市场不可用：`插件市场` 页显示「Marketplace 暂不可用 / 插件市场需要上游 API」，本机无法打开该对话框；非采集操作缺口） |
| G-H2-07 | 「此插件不包含技能。」 | 空态文字（14px 灰） | 隐藏 | 插件无技能 | `PluginMarketplaceInstallDialog.vue:174`；i18n `plugins.marketplaceNoSkills` | 需数据（插件市场不可用：`插件市场` 页显示「Marketplace 暂不可用 / 插件市场需要上游 API」，本机无法打开该对话框；非采集操作缺口） |
| G-H2-08 | 「取消」 | 次按钮（`outline`） | — | 常驻 | `PluginMarketplaceInstallDialog.vue:178`；i18n `common.cancel` | 需数据（插件市场不可用：`插件市场` 页显示「Marketplace 暂不可用 / 插件市场需要上游 API」，本机无法打开该对话框；非采集操作缺口） |
| G-H2-09 | 「添加插件」 | 主按钮（安装中前置旋转图标） | 禁用 | 详情未载入、加载中或安装中禁用 | `PluginMarketplaceInstallDialog.vue:185`；i18n `plugins.add` | 需数据（插件市场不可用：`插件市场` 页显示「Marketplace 暂不可用 / 插件市场需要上游 API」，本机无法打开该对话框；非采集操作缺口） |

元素数 9。

### H3 替换现有插件确认

条件：市场安装会替换同目录名的既有插件（来自 git、手动安装或另一市场条目）。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 | 已采状态 |
|---|---|---|---|---|---|---|
| G-H3-01 | 「替换现有插件？」 | 标题 | — | 打开时 | `<ref>/components/plugins/MarketOverwriteDialog.vue:40`；i18n `plugins.marketOverwriteTitle` | 需数据（插件市场不可用：`插件市场` 页显示「Marketplace 暂不可用 / 插件市场需要上游 API」，本机无法打开该对话框；非采集操作缺口） |
| G-H3-02 | 「"{name}" 将替换一个来自其他来源（git、手动安装或另一个市场条目）的同名插件。此操作无法撤销。」 | 描述 | — | 打开时 | `MarketOverwriteDialog.vue:42`；i18n `plugins.marketOverwriteDescription` | 需数据（插件市场不可用：`插件市场` 页显示「Marketplace 暂不可用 / 插件市场需要上游 API」，本机无法打开该对话框；非采集操作缺口） |
| G-H3-03 | 「取消」 | 次按钮（`outline`） | — | 常驻；工作禁用 | `MarketOverwriteDialog.vue:47`；i18n `common.cancel` | 需数据（插件市场不可用：`插件市场` 页显示「Marketplace 暂不可用 / 插件市场需要上游 API」，本机无法打开该对话框；非采集操作缺口） |
| G-H3-04 | 「替换」 | 危险按钮（`destructive`，工作中前置旋转图标） | — | 常驻；工作禁用 | `MarketOverwriteDialog.vue:56`；i18n `plugins.marketOverwriteConfirm` | 需数据（插件市场不可用：`插件市场` 页显示「Marketplace 暂不可用 / 插件市场需要上游 API」，本机无法打开该对话框；非采集操作缺口） |

元素数 4。

### H4 插件可更新清单

条件：市场已装插件存在更高版本；由商店页「更新」按钮或「我的插件」更新横幅打开。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 | 已采状态 |
|---|---|---|---|---|---|---|
| G-H4-01 | 「可用更新」 | 标题 | — | 打开时 | `<ref>/components/plugins/PluginUpdatesDialog.vue:65`；i18n `plugins.updatesTitle` | 需数据（插件市场不可用：`插件市场` 页显示「Marketplace 暂不可用 / 插件市场需要上游 API」，本机无法打开该对话框；非采集操作缺口） |
| G-H4-02 | 「所有插件均为最新版本」 | 空态文字（14px 灰，居中） | 隐藏 | 无待更新项 | `PluginUpdatesDialog.vue:74`；i18n `plugins.noUpdates` | 需数据（插件市场不可用：`插件市场` 页显示「Marketplace 暂不可用 / 插件市场需要上游 API」，本机无法打开该对话框；非采集操作缺口） |
| G-H4-03 | （插件占位图标 + 名称） | 行首图标（40px 底托 + 方块图标）+ 名称（14px 中等，截断） | — | 有可更新项 | `PluginUpdatesDialog.vue:87`–`:95` | 需数据（插件市场不可用：`插件市场` 页显示「Marketplace 暂不可用 / 插件市场需要上游 API」，本机无法打开该对话框；非采集操作缺口） |
| G-H4-04 | 「v{旧} → v{新}」 / 「v{新}」 | 版本行（12px 灰） | — | 有记录版本 / 旧版本未知 | `PluginUpdatesDialog.vue:101` | 需数据（插件市场不可用：`插件市场` 页显示「Marketplace 暂不可用 / 插件市场需要上游 API」，本机无法打开该对话框；非采集操作缺口） |
| G-H4-05 | （下载进度条 + 「已下载 {size}…」 / 「{已下载} / {总量}」） | 进度条（1px 高）+ 12px 灰字 | 隐藏 | 该插件安装中 | `PluginUpdatesDialog.vue:39`–`:99`、`:10`–`:13`；i18n `plugins.downloadedBytes` | 需数据（插件市场不可用：`插件市场` 页显示「Marketplace 暂不可用 / 插件市场需要上游 API」，本机无法打开该对话框；非采集操作缺口） |
| G-H4-06 | 「更新」 | 次级小按钮（刷新图标，安装中旋转） | — | 有可更新项；安装中或全部更新时禁用 | `PluginUpdatesDialog.vue:113`；i18n `plugins.update` | 需数据（插件市场不可用：`插件市场` 页显示「Marketplace 暂不可用 / 插件市场需要上游 API」，本机无法打开该对话框；非采集操作缺口） |
| G-H4-07 | 「全部更新（{count}）」 | 主按钮（刷新图标，更新中旋转） | — | 有可更新项；更新中禁用 | `PluginUpdatesDialog.vue:126`；i18n `plugins.updateAll` | 需数据（插件市场不可用：`插件市场` 页显示「Marketplace 暂不可用 / 插件市场需要上游 API」，本机无法打开该对话框；非采集操作缺口） |

元素数 7。

### H5 技能可更新清单

条件：市场已装技能存在更高版本。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 | 已采状态 |
|---|---|---|---|---|---|---|
| G-H5-01 | 「可用更新」 | 标题 | — | 打开时 | `<ref>/components/skills/SkillUpdatesDialog.vue:50`；i18n `skillsMarketplace.updatesTitle` | 需数据（插件市场不可用：`插件市场` 页显示「Marketplace 暂不可用 / 插件市场需要上游 API」，本机无法打开该对话框；非采集操作缺口） |
| G-H5-02 | 「所有技能均为最新版本」 | 空态文字 | 隐藏 | 无待更新项 | `SkillUpdatesDialog.vue:59`；i18n `skillsMarketplace.noUpdates` | 需数据（插件市场不可用：`插件市场` 页显示「Marketplace 暂不可用 / 插件市场需要上游 API」，本机无法打开该对话框；非采集操作缺口） |
| G-H5-03 | （技能图标 + 名称） | 行首图标（40px 技能图标）+ 名称 | — | 有可更新项 | `SkillUpdatesDialog.vue:71`–`:77` | 需数据（插件市场不可用：`插件市场` 页显示「Marketplace 暂不可用 / 插件市场需要上游 API」，本机无法打开该对话框；非采集操作缺口） |
| G-H5-04 | 「v{旧} → v{新}」 / 「v{新}」 | 版本行（12px 灰） | — | 有记录版本 / 旧版本未知 | `SkillUpdatesDialog.vue:80` | 需数据（插件市场不可用：`插件市场` 页显示「Marketplace 暂不可用 / 插件市场需要上游 API」，本机无法打开该对话框；非采集操作缺口） |
| G-H5-05 | 「更新」 | 次级小按钮（刷新图标，安装中旋转） | — | 有可更新项；安装中或全部更新时禁用 | `SkillUpdatesDialog.vue:89`；i18n `skillsMarketplace.update` | 需数据（插件市场不可用：`插件市场` 页显示「Marketplace 暂不可用 / 插件市场需要上游 API」，本机无法打开该对话框；非采集操作缺口） |
| G-H5-06 | 「全部更新（{count}）」 | 主按钮 | — | 有可更新项；更新中禁用 | `SkillUpdatesDialog.vue:102`；i18n `skillsMarketplace.updateAll` | 需数据（插件市场不可用：`插件市场` 页显示「Marketplace 暂不可用 / 插件市场需要上游 API」，本机无法打开该对话框；非采集操作缺口） |

元素数 6。

### H6 导入 Chrome Cookie

条件：设置 → 浏览器数据 → 「导入…」；先扫描 Chrome 配置，再选配置预览计数。入口已采（旁证 `runtime/settings/通用.json`：「浏览器数据」272,597、「导入…」924,654），对话框未采。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 | 已采状态 |
|---|---|---|---|---|---|---|
| G-H6-01 | 「导入 Chrome Cookie」 | 标题 | — | 打开时 | `<ref>/components/settings/BrowserDataSettings.vue:273`；i18n `browserData.importTitle` | 可见 Y11 285,250 460×16（`导入 Chrome Cookie`） |
| G-H6-02 | 「此操作会替换内置浏览器中的全部非分区 Cookie，并重新加载已打开的页面。每个 Cookie 都会单独尝试导入，因此个别 Cookie 可能失败；部分网站可能需要重新登录。」 | 描述 | — | 可导入数量不为 0 | `BrowserDataSettings.vue:274`；i18n `browserData.replaceWarning` | 可见 Y12（标题下描述整段，PNG） |
| G-H6-03 | 「正在扫描 Chrome 配置…」 | 加载行（旋转图标 + 14px 灰字，最小高 96px） | 隐藏 | 扫描中 | `BrowserDataSettings.vue:279`；i18n `browserData.scanning` | 可见 Y11（`正在扫描 Chrome 配置…` 行 + 导入按钮置灰） |
| G-H6-04 | 「Chrome 配置」 | 分组标签（12px 中等灰） | — | 扫描完成 | `BrowserDataSettings.vue:283`；i18n `browserData.profileLabel` | 可见 Y12 299,280 432×14（`Chrome 配置`） |
| G-H6-05 | （配置名 + 能力标签） | 单选项行（自绘单选圈 + 名称 + 12px 能力标签） | 首个可选配置 | 存在配置 | `BrowserDataSettings.vue:285`–`:300` | 可见 Y12 335,312 343×18（`Han` 等 3 个配置行，右侧 `可导入`） |
| G-H6-06 | 「未找到 Google Chrome 配置。」 | 空态文字 | 隐藏 | 无配置且无错误 | `BrowserDataSettings.vue:302`；i18n `browserData.noProfiles` | 需数据（本机无 Chrome 或配置不可读） |
| G-H6-07 | 「正在读取所选配置…」 | 加载行（旋转图标 + 14px 灰字） | 隐藏 | 读取所选配置中 | `BrowserDataSettings.vue:304`；i18n `browserData.previewing` | 需数据（切换配置后的预览读取） |
| G-H6-08 | 「可导入 {importable} · 会话 {session} · 已过期 {expired} · 分区 {partitioned}」 | 计数块（边框 + 浅底，12px） | 隐藏 | 已取到预览 | `BrowserDataSettings.vue:313`；i18n `browserData.counts` | 需数据（预览返回计数） |
| G-H6-09a | （成功态：对勾图标 + 「已导入 {imported} · 失败 {failed} · 已跳过…」） | 结果块（对勾图标 + 14px 文字） | 隐藏 | 导入结束且失败数为 0 | `BrowserDataSettings.vue:317`–`:328`；i18n `browserData.importResultCounts` | 需交互（点「导入」并等结束，且失败数为 0） |
| G-H6-09b | （部分失败态：告警图标 + 计数 + 「部分 Cookie 未能导入，部分网站可能需要重新登录。」） | 结果块（告警图标 + 14px 文字 + 12px 警示色说明） | 隐藏 | 导入结束且失败数 >0 | `BrowserDataSettings.vue:317`–`:328`；i18n `browserData.importResultCounts`、`importPartial` | 需数据（导入有失败项） |
| G-H6-10 | （错误消息 + 诊断行 + 「重试」） | 错误块（危险色 + 等宽诊断行 + 次级按钮） | 隐藏 | 导入或读取失败 | `BrowserDataSettings.vue:331`–`:347` | 可见 Y12 299,422 374×28（`无法访问 Chrome 加密密钥。macOS 请解锁钥匙串…`）+ 诊断行 299,454 + 重试 684,440 |
| G-H6-11 | 「取消」 / 「关闭」 | 次按钮（`outline`） | — | 未结束时「取消」，已导入后「关闭」 | `BrowserDataSettings.vue:355`；i18n `common.cancel`、`common.close` | 可见 Y11/Y12 ≈575,477（`取消`） |
| G-H6-12 | 「导入」 / 「正在导入 Cookie…」 / （阶段文案：正在关闭浏览器页面… / 正在更新浏览器数据… / 正在验证浏览器数据… / 正在恢复原有登录状态…） | 主按钮（导入中前置旋转图标） | 禁用 | 未取到预览、导入中、读取中或扫描中禁用 | `BrowserDataSettings.vue:358`；i18n `browserData.confirmImport`、`importing`、`phaseQuiescing`/`phaseApplying`/`phaseVerifying`/`phaseRollingBack` | 可见（仅视觉）Y11/Y12 ≈735,477（`导入`；未取到预览时置灰） |

元素数 13（原 12，`G-H6-09` 拆两行）。

### H7 清理浏览器数据确认

条件：设置 → 浏览器数据 → 「清理数据…」。入口已采（旁证 `runtime/settings/通用.json`：「清理数据…」899,704），对话框未采。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 | 已采状态 |
|---|---|---|---|---|---|---|
| G-H7-01 | 「清理浏览器数据」 | 标题 | — | 打开时 | `BrowserDataSettings.vue:379`；i18n `browserData.clearTitle` | 可见 Y13 313,292 404×16（`清理浏览器数据`） |
| G-H7-02 | 「此操作只会清除内置浏览器中的登录状态、Cookie、本地存储、IndexedDB、Service Worker 和缓存。浏览器页面会关闭并以空白页重新打开。」 | 描述 | — | 打开时 | `BrowserDataSettings.vue:380`；i18n `browserData.clearDescription` | 可见 Y13 313,314 404×53（`此操作只会清除内置浏览器中的登录状态…`） |
| G-H7-03 | 「内置浏览器数据已清理。」 | 成功行（对勾 + 14px 文字） | 隐藏 | 清理成功 | `BrowserDataSettings.vue:383`；i18n `browserData.clearSuccess` | 需交互（点「清理浏览器数据」并等成功） |
| G-H7-04 | （错误消息 + 诊断行） | 错误块（危险色 + 等宽诊断行） | 隐藏 | 清理失败 | `BrowserDataSettings.vue:387` | 需数据（清理 RPC 失败） |
| G-H7-05 | 「取消」 / 「关闭」 | 次按钮 | — | 未完成时「取消」，完成后「关闭」 | `BrowserDataSettings.vue:393`；i18n `common.cancel`、`common.close` | 可见 Y13 ≈575,402（`取消`） |
| G-H7-06 | 「清理浏览器数据」 / 「正在清理浏览器数据…」 | 危险按钮（清理中前置旋转图标） | — | 未完成时显示；清理中禁用 | `BrowserDataSettings.vue:396`、`:395`；i18n `browserData.confirmClear`、`clearing` | 可见 Y13 603,402 114×32（`清理浏览器数据`，危险按钮） |

元素数 6。

区域 H 合计 57（原 56，`G-H6-09` 拆两行）。

---

## 9. 计数汇总（补采后）

### 9.1 八类数字

| 区域 | 元素数 | 可见 | 不可观察 | 主因 |
|---|---|---|---|---|
| A 搜索 | 46 | 33 | 13 | 结果集太小（无未读/无失败行）、`@` 提交候选需 git 仓库 |
| B 命令 | 29 | 21 | 8 | 设置 → 添加模型调色板（B2）未打开；`/fast` 需快速模式 |
| C 审批 | 63 | 11 | 52 | 沙箱/子代理/成员审批与九类渲染器都要逐工具触发权限请求 |
| D 提问 | 22 | 9 | 13 | 单字段单选已打通；多字段、倒计时、筛选、多选、自定义答案仍缺 |
| E 确认 | 21 | 9 | 12 | 删会话 + 删工作区已采；编辑运行中、归档、批量仍未采 |
| F 重命名 | 16 | 7 | 9 | 工作区重命名已采；**会话重命名被 `SessionTitle` 不可用挡住** |
| G 导出与 Fork | 22 | 8 | 14 | 导出对话框已采（大纲 RPC 报 `BAD_REQUEST`）；Fork 需两窗口持有 |
| H 导入与更新 | 57 | 20 | 37 | 添加插件 + Chrome 导入 + 清理已采；市场系四框被上游 API 挡住 |
| **合计** | **276** | **118** | **158** | 上一版 40 / 236 |

判定口径未变：一行一个模板节点，`可见` 包含「随行」「仅视觉」两种反推。

### 9.2 本轮从「不可观察」翻成「可见」的 78 项

- A2 搜索聊天：`G-A2-08`–`G-A2-13`（分组、结果行、所属工作区、两种行首图标、快捷键、选中态）——`X1`/`X3`
- A3 `@` 提及：`G-A3-01`、`03`、`04`、`05`、`10`、`11`——`X7`/`X8b`/`X9`
- A4 `$` 提及：`G-A4-01`、`03`、`04`、`05`——`X10`/`X11`
- B1 斜杠菜单：`G-B1-03`、`04`(技能分支)、`06`、`07`、`19`、`20`——`X4`/`X5`/`X6c`
- D1 提问卡：`G-D1-01`、`02`、`05`、`07`、`08`、`11`、`18`、`21`、`22`——`Y16`/`Y17`/`Y18`
- E2 删除会话：`G-E2-01`、`02`、`04`、`05`——`X13`（`G-E2-03` 仍缺）
- E4 删除工作区：`G-E4-01`–`03`、`05`、`06`——`Y14`
- F2 重命名工作区：`G-F2-01`–`07`——`Y15`
- G1 导出：`G-G1-01`–`05`、`15`、`16`、`17`——`X12`
- H1 添加插件：`G-H1-01`–`09`、`11`、`12`——`Y8`/`Y9`
- H6 Chrome 导入：`G-H6-01`–`05`、`10`、`11`、`12`——`Y11`/`Y12`
- H7 清理数据：`G-H7-01`、`02`、`05`、`06`——`Y13`

### 9.3 结构性发现（补采新增）

| 发现 | 依据 |
|---|---|
| 插件市场整块不可用 → H2/H3/H4/H5 共 28 项不是采集缺口 | `Y6` 页面正文「Marketplace 暂不可用 / 插件市场需要上游 API」 |
| 导出对话框的大纲 RPC 在本机直接报错 | `X12` 错误行 `BAD_REQUEST: History cursor exceeds fixed position`，`导出` 按钮置灰 |
| Chrome Cookie 导入被钥匙串挡住 | `Y12` 错误块 `无法访问 Chrome 加密密钥…` + 诊断 `key_access_denied` |
| 终端是 canvas 渲染，DOM 里只有 `textarea.xterm-helper-textarea` | `Y1`/`Y2` 的 JSON 只有页签文本与 helper textarea；结论状态以 PNG 为准 |
| 关闭最后一个终端**不会**留提示行 | `Y4` 关闭后页签消失并切回上一页签，面板内无任何残留行 |
| 斜杠菜单在滚动后确实含 `技能` 分组与 `/branch`、`/workflow` | `X6c`：`技能` 分组 + `agents-sdk` 等 + `/branch` 401,274、`/workflow` 401,302 |
| `ask_user` 的 elicitation 卡片可直接在会话内触发 | `Y16`–`Y18`：卡片、`等待操作` 徽标、收起态、提交后自动解除 |
| 审批卡只在 `exec` 上弹：`read`/`write`/`grep` 在工作区内直接放行 | 一轮里连续 read → write → grep，全程没有出现权限卡（`Z2`/`Z3` 为证），因此 C4 的九类渲染器在本机默认权限下不可达 |
| 本轮改动块已可采 | `Z2` 折叠行 `1 个文件改动`（429,452）+ 活动条 chip（435,547）；`Z3` 展开出文件行 |
| 同一会话开第二个窗口不会弹 Fork | 点「在新窗口中打开」直接起新窗口（URL `?sessionId=…`），再在原窗口点该会话也无「会话正在使用中」弹框 |
| 右栏终端关闭即关页签 | `Y4`（关页签）与 `Y5b`（右键关闭进程）都直接回到上一页签，无［进程已关闭］提示行 |

## 10. 仍需补采（158 项）

### 10.1 需数据（102 项，按区域）

| 区域 | 元素 ID |
|---|---|
| A（13） | `G-A1-02b` `G-A2-05` `G-A2-06` `G-A2-07` `G-A2-11a` `G-A2-11b` `G-A2-11c` `G-A3-02` `G-A3-06` `G-A3-07` `G-A3-08` `G-A3-09` `G-A4-02` |
| B（5） | `G-B1-02` `G-B1-18` `G-B2-04` `G-B2-06` `G-B2-07` |
| C（47） | `G-C1-03`–`G-C1-11`、`G-C1-14`、`G-C1-15`、`G-C2-01`–`G-C2-08`（除可见）、`G-C3-01`–`G-C3-09`（除可见）、`G-C4-05`–`G-C4-25` |
| D（11） | `G-D1-03` `G-D1-04` `G-D1-06` `G-D1-09` `G-D1-10` `G-D1-12` `G-D1-14` `G-D1-15` `G-D1-16` `G-D1-19` `G-D1-20` |
| E（1） | `G-E3-03` |
| G（5） | `G-G1-06` `G-G1-07` `G-G1-10` `G-G1-12` `G-G1-14` |
| H（20） | `G-H2-*` / `G-H3-*` / `G-H4-*` / `G-H5-*`（市场不可用，共 28 项中 20 项落此列，另 8 项见 §10.3）、`G-H6-06`–`09b`、`G-H7-04` |

### 10.2 需交互（48 项）

| 区域 | 元素 ID | 一次取证手段 |
|---|---|---|
| B（6） | `G-B2-01` `G-B2-02` `G-B2-03` `G-B2-05` `G-B2-08` `G-B2-09` | 设置 → 添加模型调色板 |
| C（5） | `G-C1-12` `G-C1-16` `G-C1-21` `G-C2-05` `G-C3-07` | 点动作后提交中 / 提交失败、成员动作重试 |
| D（2） | `G-D1-13` `G-D1-17` | 筛选无匹配、必填未答即提交 |
| E（11） | `G-E1-01`–`04`、`G-E2-03`、`G-E3-01` `02` `04` `05` `06`、`G-E4-04` | 运行中提交编辑、删除失败、批量归档 |
| F（9） | `G-F1-01`–`09` | **先恢复 `SessionTitle` 能力**（当前 C2/C13 菜单项渲染成「重命名（SessionTitle 不可用）」并禁用） |
| G（7） | `G-G1-08` `G-G1-11`、`G-G2-01`–`05` | 展开某回合、同会话两窗口后再打开 |
| H（8） | `G-H1-10` `G-H6-09a` `G-H7-03` + `G-H2-01` `02` `08` `09` `G-H4-05`（后 5 项同时受市场阻塞） | 空路径提交、导入成功、清理成功 |

### 10.3 两者兼有（8 项）

`G-G1-09`（大纲失败分支：本机只给出原始 `BAD_REQUEST`，未走 i18n 文案）、`G-G1-13`、`G-H2-04`、`G-H3-04`、`G-H4-06` `G-H4-07` `G-H5-05` `G-H5-06`。

## 11. 九条动作的执行结果

| # | 动作 | 结果 |
|---|---|---|
| 1 | 打开「搜索聊天」并键入命中词 | 完成（`X1`/`X2`/`X3`）；分组与五种行首图标只拿到置顶星标与默认消息两种 |
| 2 | 斜杠菜单空态 / 选中 / 滚动 | 完成（`X4`/`X5`/`X6*`），额外发现 `技能` 分组与 `/branch`、`/workflow` |
| 3 | `@` 与 `$` 提及 | 完成（`X7`–`X11`）；提交候选未测（需 git 仓库工作区） |
| 4 | 提问卡 | 完成单字段单选（`Y16`–`Y18`）；多字段 / 倒计时 / 多选未造出 |
| 5 | 四类确认框 | 删除会话（`X13`）、删除工作区（`Y14`）完成；编辑运行中（E1）、归档（E3）未采 |
| 6 | 重命名 | 工作区重命名完成（`Y15`）；会话重命名被 `SessionTitle` 挡住（**blocked**，原因见 §10.2） |
| 7 | 导出与 Fork | 导出完成（`X12`）；Fork 需同一会话在两窗口打开，本机只允许单窗口持有该会话 |
| 8 | 导入与更新 | 添加插件（`Y8`/`Y9`）、Chrome 导入（`Y11`/`Y12`）、清理（`Y13`）完成；市场与更新四框被上游 API 挡住 |
| 9 | 时间线本轮改动块 / 产物块 | 改动块已采（`Z2` 折叠 / `Z3` 展开）；**已撤销**态与**产物块**三态未采：`dim-agent` 不是 git 仓库（改动面板给 `Git 改动不可用`），且没有 runId 带产物的轮次 |
| 补 | 右栏终端（本轮追加） | 完成（`Y1`/`Y2`/`Y3`/`Y3b`）；关闭最后一个终端**无提示行**（`Y4`/`Y5b` 为证，已同步改判 `inventory-panels-rt.md` 的 T-08/T-09/T-15/T-16/T-18/T-19/T-21） |
| 补 | 沙箱卡 / 子代理模型卡 / 9 类输入渲染器 / 成员审批 | 未采且**本机不可达**：默认权限下只有 `exec` 会弹审批卡，`read`/`write`/`grep` 在工作区内直接放行（见 §9.3），沙箱与子代理变体需要 App Server 侧产生对应 `variant` |
| 补 | 编辑运行中（E1）/ 批量归档（E3）/ Fork（G2） | E1：sleep 45 的一轮在我改完消息前就结束了，两次尝试未卡住运行中窗口；E3：⌘+点击未能进入多选（菜单仍是单条文案），单条空闲会话归档不弹框；G2：新窗口直接打开，不触发 Fork 弹框 |

## 12. 与既有清单的对照

| 本文件 | `inventory-global.md` | 差异 |
|---|---|---|
| §1 A 搜索（46） | A（41） | `G-A1-02` 拆放大镜/转圈两行（+1）；`G-A2-11` 拆 5 个行首图标分支（+4）；其余逐行一致 |
| §2 B 命令（29） | B（29） | 数量一致；B2 补「实际使用处=设置→添加模型」的采不到原因 |
| §3 C 审批（63） | C（63） | 数量一致；C1/C4 补实测坐标，C2/C3 补「整组需对应 variant」 |
| §4 D 提问（22） | D（22） | 数量一致；本轮已采单字段单选，补 `Y16`–`Y18` 证据 |
| §5 E 确认（21） | E（21） | 数量一致；E2/E4 已采，E1/E3 仍缺 |
| §6 F 重命名（16） | F（16） | 数量一致；F2 已采，F1 被 `SessionTitle` 阻塞 |
| §7 G 导出与 Fork（22） | G（22） | 数量一致；G1 已采（大纲 RPC 报错），G2 仍缺 |
| §8 H 导入与更新（57） | H（56） | `G-H6-09` 拆对勾/告警两行（+1）；市场系四框为上游 API 阻塞 |
| **合计 276** | **合计 270** | 差 6 = 3 处图标分支拆行（A1-02 +1、A2-11 +4、H6-09 +1） |

不在本文件范围（避免重复登记）：

- 会话内搜索条与搜索聊天弹窗的**入口**（左栏「搜索」行、`⌘G` 键帽、标题栏按钮）→ `inventory-main-rt.md`。
- 侧栏会话行 / 工作区行 / 标题栏 `⋯` 菜单的四类菜单项本身 → `inventory-main-rt.md`（本文件只把菜单项当入口证据引用坐标）。
- 输入区技能按钮浮层（C9 `popover-skills`）→ `inventory-main-rt.md`；不是 A4 的 `$` 提及弹层。
- 完全访问确认（C6）、退出登录确认（C33）→ 账户 / 权限域。
- 设置页内的表单元素（`BrowserDataSettings` 页面级 `导入…` / `清理数据…` 按钮、插件设置页）→ 设置域清单；本文件只登记点开的六个对话框。
- 右栏终端面板（`T-08`/`T-09`）的判定 → `inventory-panels-rt.md`；本轮只提供 `runtime/panels/` 下的 `Y1`–`Y4` 证据。
