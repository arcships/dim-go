# 主窗口清单（模板 ∪ 运行时 ∪ 截图 联表）

范围：主窗口五区（标题栏 / 左栏 / 时间线 / 输入区 / 状态条）＋空态（NewThread）＋运行时出现的覆盖层与独立页。本文件只做元素对账，不写行为需求、不写验收。

## 0. 取证与口径

### 0.1 材料

| 类 | 位置 |
|---|---|
| 模板 | 参考产品源码；仓库根记 `<ref>`，定位写作 `<ref>/<相对路径>:<行号>`，相对 `packages/desktop/packages/renderer/src` |
| 运行时 | `docs/domains/requirements/runtime/main/{empty,new-session,right-panel,search,scheduled,skills}.json`（节点链 + 标签 + 包围盒，单位 CSS px） |
| 截图 | 同目录同名 `.png`（DPR=2，1030×725 CSS 视口） |
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

### 0.3 列义

- **名称（文案原文）**：`t('key')` 的中文原文；`{x}` 为插值占位；无文案记 `（无文案）`。
- **控件**：按钮 / 图标按钮 / 菜单项 / 输入框 / 文本 / 容器 / 状态位 / 浮层。
- **图标**：模板里实际渲染的图标（lucide 组件名或 `ThemeIcon:<name>`）；无则 `—`。
- **条件**：祖先条件 ∧ 本节点 `v-if`/`v-else` 分支条件。无条件的常显节点写 `常显`。
- **在状态**：该元素在哪些采集状态里出现（E/N/R/S/K/Q）；`—` = 六态均未命中。
- **位置**：运行时包围盒 `x,y w×h`，前缀为状态码；`≈` 表示 JSON 未采集该节点（无文案或纯 svg），坐标按截图量取。
- **定位**：模板出处；同一文案多处渲染时列多处。

### 0.4 采集局限（判「未命中」前必读）

1. **无文案且非 svg 的图标按钮会被采集跳过**：如右栏 `+ 添加页签`（`RightPanelBar.vue:381`，无 `aria-label`/`title`）、输入区 `+ 附件`、`筛选文件...` 输入框在 `right-panel.json` 里都缺失，但截图可见。故「JSON 无」≠「屏上无」，本表用截图补位并标 `≈`。
2. **opacity-0 的隐藏标签仍会被采集**：`empty.json` 里 `Agent`（463,456）、`技能`（491,455）是 Agent 模式/技能选择器展开时才出现文字的隐藏 `<span>`（按钮本身可见）。本表只记按钮（`IN-06`、`IN-07`），不把这两个标签当作独立命中。
3. **S、K、Q 三态的左栏与主区被折叠/替换**：S/K 是独立路由页（整窗页面），Q 只采到弹窗层；三者都不能用来判断五区元素的去留。
4. **`search.json` 的文本宽度失真**：弹窗标题与描述的 `w` 采成 1（`搜索聊天`、`搜索并打开已有聊天`），宽度不可用，只取 x/y。

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
| TB-05 | （无文案） | 占位 | — | 会话 ∧ macOS ∧ 非全屏 ∧ 左栏收起 | — | — | `<ref>/components/chat-workspace/ChatHeaderFrame.vue:10` |
| TB-06 | （无文案） | 浮层容器 | — | 会话 ∧ 消息内搜索打开 | — | — | `<ref>/components/chat-workspace/ChatHeaderFrame.vue:7` |
| TB-07 | 搜索聊天... | 输入框 | search（ThemeIcon） | 会话 ∧ 消息内搜索打开 | — | — | `<ref>/components/chat-workspace/timeline/ThreadSearchBar.vue:19`–`:30` |
| TB-08 | 关闭搜索 | 图标按钮 | X | 会话 ∧ 消息内搜索打开 | — | — | `<ref>/components/chat-workspace/timeline/ThreadSearchBar.vue:31`–`:41` |
| TB-09 | 上一个搜索结果 | 图标按钮 | ArrowUp | 会话 ∧ 搜索打开 ∧ 有查询词 | — | — | `<ref>/components/chat-workspace/timeline/ThreadSearchBar.vue:50`–`:60` |
| TB-10 | 下一个搜索结果 | 图标按钮 | ArrowDown | 会话 ∧ 搜索打开 ∧ 有查询词 | — | — | `<ref>/components/chat-workspace/timeline/ThreadSearchBar.vue:61`–`:71` |
| TB-11 | 0 个结果 | 文本 | — | 会话 ∧ 搜索打开 ∧ 有查询词 ∧ 命中 0 | — | — | `<ref>/components/chat-workspace/timeline/ThreadSearchBar.vue:77`–`:80` |
| TB-12 | `{current} / {total} 个结果` | 文本 | — | 会话 ∧ 搜索打开 ∧ 有查询词 ∧ 命中 >0 | — | — | `<ref>/components/chat-workspace/timeline/ThreadSearchBar.vue:77`–`:80` |
| TB-13 | 展开侧边栏 ⌘B | 图标按钮 | PanelLeftOpen | 会话 ∧ 左栏收起 ∧ 非弹窗窗口 | — | — | `<ref>/components/chat-workspace/WorkspaceHeaderNavigation.vue:31`–`:49` |
| TB-14 | 上一个工作区会话 | 图标按钮 | ArrowLeft | 会话 ∧ 左栏收起 ∧ 非弹窗窗口 ∧ 未隐藏会话导航 | — | — | `<ref>/components/chat-workspace/WorkspaceHeaderNavigation.vue:51`–`:67` |
| TB-15 | 下一个工作区会话 | 图标按钮 | ArrowRight | 同上 | — | — | `<ref>/components/chat-workspace/WorkspaceHeaderNavigation.vue:69`–`:85` |
| TB-16 | 在工作区中新建会话 | 图标按钮 | SquarePen | 会话 ∧ 左栏收起 ∧ 非弹窗窗口 | — | — | `<ref>/components/chat-workspace/ChatHeaderFrame.vue:18`–`:23` |
| TB-17 | `{workspaceTitle}` | 文本 | — | 会话 ∧ 显示工作区标题 | — | — | `<ref>/components/chat-workspace/ChatHeaderFrame.vue:27`–`:28` |
| TB-18 | `/` | 文本 | — | 会话 ∧ 显示工作区标题 | — | — | `<ref>/components/chat-workspace/ChatHeaderFrame.vue:29` |
| TB-19 | `{title}` | 标题文本（h1） | — | 会话 | — | — | `<ref>/components/chat-workspace/ChatHeaderFrame.vue:30` |

元素数：15

### 1.3 标题栏动作（`ChatHeader` 插槽 + 右栏开关）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| TB-20 | （无文案） | 图标按钮（菜单触发器） | MoreHorizontal | 会话 | — | — | `<ref>/components/chat-workspace/ChatHeader.vue:37`–`:41` |
| TB-21 | 置顶 | 菜单项 | Pin | 会话 ∧ 菜单打开 ∧ 非弹窗窗口 ∧ 未置顶 | — | — | `<ref>/components/chat-workspace/ChatHeader.vue:43`–`:46` |
| TB-22 | 取消置顶 | 菜单项 | PinOff | 会话 ∧ 菜单打开 ∧ 非弹窗窗口 ∧ 已置顶 | — | — | `<ref>/components/chat-workspace/ChatHeader.vue:43`–`:46` |
| TB-23 | 重命名 | 菜单项 | ChevronRight | 会话 ∧ 菜单打开 | — | — | `<ref>/components/chat-workspace/ChatHeader.vue:47`–`:54` |
| TB-24 | 在新窗口打开 | 菜单项 | ExternalLink | 会话 ∧ 菜单打开 ∧ 非弹窗窗口 | — | — | `<ref>/components/chat-workspace/ChatHeader.vue:55` |
| TB-25 | 导出 | 菜单项 | Download | 会话 ∧ 菜单打开 | — | — | `<ref>/components/chat-workspace/ChatHeader.vue:59` |
| TB-26 | 归档 | 菜单项 | Archive | 会话 ∧ 菜单打开 | — | — | `<ref>/components/chat-workspace/ChatHeader.vue:63` |
| TB-27 | 复制会话链接 | 菜单项 | Link | 会话 ∧ 菜单打开 ∧ 调试菜单项可见 | — | — | `<ref>/components/chat-workspace/ChatHeader.vue:68` |
| TB-28 | 复制会话 ID | 菜单项 | FileCode | 同上 | — | — | `<ref>/components/chat-workspace/ChatHeader.vue:72` |
| TB-29 | 复制会话路径 | 菜单项 | FolderOpen | 会话 ∧ 菜单打开 ∧ 有工作目录 | — | — | `<ref>/components/chat-workspace/ChatHeader.vue:76` |
| TB-30 | 复制调试信息 | 菜单项 | FileCode | 会话 ∧ 菜单打开 ∧ 调试菜单项可见 | — | — | `<ref>/components/chat-workspace/ChatHeader.vue:80` |
| TB-31 | `{打开方式}` | 图标按钮 | 运行期解析（iconify / 图片 / lucide） | 会话 ∧ 「打开方式」组可见 | — | — | `<ref>/components/chat-workspace/ChatHeader.vue:92`–`:105` |
| TB-32 | 更多打开方式 | 图标按钮 | ChevronDown | 会话 ∧ 「打开方式」组可见 | — | — | `<ref>/components/chat-workspace/ChatHeader.vue:106`–`:112` |
| TB-33 | 正在检测已安装应用… | 菜单项（禁用） | — | 会话 ∧ 打开方式菜单 ∧ 应用列表加载中 | — | — | `<ref>/components/chat-workspace/ChatHeader.vue:116`–`:118` |
| TB-34 | 未检测到可用应用 | 菜单项（禁用） | — | 会话 ∧ 打开方式菜单 ∧ 列表为空 | — | — | `<ref>/components/chat-workspace/ChatHeader.vue:139`–`:141` |
| TB-35 | `{应用名}` | 菜单项 | 应用图标 | 会话 ∧ 打开方式菜单 ∧ 每款可用应用 | — | — | `<ref>/components/chat-workspace/ChatHeader.vue:127`–`:138` |
| TB-36 | 在文件夹中显示 | 菜单项 | FolderOpen | 会话 ∧ 打开方式菜单 | — | — | `<ref>/components/chat-workspace/ChatHeader.vue:143`–`:146` |
| TB-37 | （无文案，系统默认程序） | 菜单项 | 运行期解析 | 会话 ∧ 打开方式菜单 | — | — | `<ref>/components/chat-workspace/ChatHeader.vue:147`–`:150` |
| TB-38 | （无文案） | 竖分隔线 | — | 会话 ∧ 「打开方式」组可见 | — | — | `<ref>/components/chat-workspace/ChatHeader.vue:153` |
| TB-39 | 打开上下文面板 | 图标按钮 | PanelRight | 会话 ∧ 右栏未开 | — | — | `<ref>/components/chat-workspace/ChatHeader.vue:583`–`:598` |
| TB-40 | 关闭上下文面板 | 图标按钮 | PanelRightClose | 会话 ∧ 右栏已开 | — | — | `<ref>/components/chat-workspace/ChatHeader.vue:583`–`:598` |

元素数：21

**区 1 元素数：40**

---

## 2. 区：左栏

### 2.1 栏头（SidebarHeader）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| LB-01 | （无文案） | 占位 | — | macOS ∧ 非全屏 | E/N（不可见） | E ≈0,0 80×12（仅占位，无可见像素） | `<ref>/components/sidebar/SidebarHeader.vue:92`–`:96` |
| LB-02 | （无文案，品牌图形） | 图标 | 品牌 SVG（明暗两套） | 非 macOS | — | — | `<ref>/components/sidebar/SidebarHeader.vue:103`–`:106` |
| LB-03 | `DESKTOP_PRODUCT_NAME` | 文本 | — | 非 macOS | — | — | `<ref>/components/sidebar/SidebarHeader.vue:107` |
| LB-04 | 更新 | 按钮 | Download | 更新态 `available` ∧ 无错误消息 | — | — | `<ref>/components/sidebar/SidebarHeader.vue:58`、`:114`–`:134` |
| LB-05 | `{downloadProgressPercent}%` | 按钮（禁用） | — | 更新态 `downloading` | — | — | `<ref>/components/sidebar/SidebarHeader.vue:55`、`:126` |
| LB-06 | 正在准备… | 按钮（禁用） | Loader2 | 更新态 `preparing` | — | — | `<ref>/components/sidebar/SidebarHeader.vue:56`、`:130` |
| LB-07 | 重启安装 | 按钮 | ArrowDown | 更新态 `downloaded` | — | — | `<ref>/components/sidebar/SidebarHeader.vue:57`、`:131` |
| LB-08 | 重试更新 | 按钮（危险色） | Download | 更新态 `available` ∧ 有错误消息 | — | — | `<ref>/components/sidebar/SidebarHeader.vue:58`、`:121`–`:125` |
| LB-09 | `{状态} · v{版本} · {错误消息}` | 浮层（tooltip / aria-label） | — | 悬停更新按钮 | — | — | `<ref>/components/sidebar/SidebarHeader.vue:60`–`:69`、`:136` |
| LB-10 | 收起侧边栏 ⌘B | 图标按钮 | PanelLeftClose | 允许显示切换 ∧ 左栏展开 | E/N | E 325,14 28×28 | `<ref>/components/sidebar/SidebarHeader.vue:146`–`:164` |

元素数：10

### 2.2 全局入口（GlobalEntries）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| LB-11 | 新建会话 | 菜单行（图标 + 文本） | ThemeIcon:new-session | 常显（禁用由 `disableNewThread`） | E/N | 按钮 E 7,59 346×28；文本 35,64 49×18 | `<ref>/components/sidebar/GlobalEntries.vue:65`–`:76` |
| LB-12 | 搜索 | 菜单行（图标 + 文本） | ThemeIcon:search | 常显 | E/N | 按钮 7,91 346×28；文本 35,96 311×18 | `<ref>/components/sidebar/GlobalEntries.vue:79`–`:92` |
| LB-13 | ⌘G（Windows/Linux 为 Ctrl+G） | 快捷键徽标 | — | 悬停或键盘聚焦该行 | — | — | `<ref>/components/sidebar/GlobalEntries.vue:88`–`:90` |
| LB-14 | 定时任务 | 菜单行（图标 + 文本） | Clock3 | 常显（路由为定时任务时高亮） | E/N | 按钮 7,122 346×28；文本 35,127 49×18 | `<ref>/components/sidebar/GlobalEntries.vue:95`–`:106` |
| LB-15 | 技能/插件 | 菜单行（图标 + 文本） | LayoutGrid | 常显（路由为技能时高亮） | E/N | 按钮 7,154 346×28；文本 35,159 53×18 | `<ref>/components/sidebar/GlobalEntries.vue:109`–`:120` |

元素数：5

### 2.3 置顶区（PinnedThreads）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| LB-16 | （无文案） | 容器 | — | 有置顶会话 ∨ 有置顶工作区 ∨ 置顶分页存在 | — | — | `<ref>/components/sidebar/PinnedThreads.vue:180`–`:182` |
| LB-17 | 已置顶 | 分组标题 | — | LB-16 | — | — | `<ref>/components/sidebar/PinnedThreads.vue:192` |
| LB-18 | （置顶会话行） | 行（复用 TB/LB 会话行组） | 同会话行 | LB-16 ∧ 每条置顶会话 | — | — | `<ref>/components/sidebar/PinnedThreads.vue:213`–`:233` |
| LB-19 | （无文案） | 拖拽落点指示器 | — | 会话拖拽悬停于置顶列表项半区 | — | — | `<ref>/components/sidebar/PinnedThreads.vue:197`、`:225` |
| LB-20 | 上一页 | 文字按钮 | — | 存在更早的置顶会话页 | — | — | `<ref>/components/sidebar/PinnedThreads.vue:239` |
| LB-21 | 显示更多 | 文字按钮 | — | 存在更多置顶会话页 | — | — | `<ref>/components/sidebar/PinnedThreads.vue:240` |
| LB-22 | （置顶工作区分组） | 分组（复用工作区组） | 同工作区行 | LB-16 ∧ 存在置顶工作区 | — | — | `<ref>/components/sidebar/PinnedThreads.vue:251`–`:275` |
| LB-23 | （无文案） | 容器态 | — | 会话拖拽进入置顶区 | — | — | `<ref>/components/sidebar/PinnedThreads.vue:180` |

元素数：8

### 2.4 列表头（ThreadListHeader）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| LB-24 | 会话 | 文本 | — | 常显（左栏展开） | E/N | 11,197 25×14 | `<ref>/components/sidebar/ThreadListHeader.vue:94` |
| LB-25 | （无文案） | 图标按钮 | FolderPlus | 常显（选择目录中禁用） | E/N | 297,192 25×25 | `<ref>/components/sidebar/ThreadListHeader.vue:98`–`:106` |
| LB-26 | （无文案） | 图标按钮（菜单触发器） | List（分组视图）/ Grid3X3（平铺） | 办公模式隐藏该切换器 | E/N | 325,192 25×25 | `<ref>/components/sidebar/ThreadListHeader.vue:109`–`:119` |
| LB-27 | 排序 | 菜单分组标题 | — | 视图菜单打开 | — | — | `<ref>/components/sidebar/ThreadListHeader.vue:127`–`:129` |
| LB-28 | 最近活跃 | 菜单项 | Check（选中时） | 视图菜单打开 | — | — | `<ref>/components/sidebar/ThreadListHeader.vue:130`–`:137` |
| LB-29 | 最近创建 | 菜单项 | Check（选中时） | 视图菜单打开 | — | — | `<ref>/components/sidebar/ThreadListHeader.vue:138`–`:145` |
| LB-30 | 视图 | 菜单分组标题 | — | 视图菜单打开 | — | — | `<ref>/components/sidebar/ThreadListHeader.vue:149`–`:151` |
| LB-31 | 按工作区分组 | 菜单项 | Check（选中时） | 视图菜单打开 | — | — | `<ref>/components/sidebar/ThreadListHeader.vue:152`–`:159` |
| LB-32 | 平铺 | 菜单项 | Check（选中时） | 视图菜单打开 | — | — | `<ref>/components/sidebar/ThreadListHeader.vue:160`–`:167` |
| LB-33 | 折叠全部 | 菜单项 | ChevronsDownUp | 视图菜单打开 ∧ 当前为分组视图 | — | — | `<ref>/components/sidebar/ThreadListHeader.vue:171`–`:181` |

元素数：10

### 2.5 工作区组（ProjectItem）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| LB-34 | （无文案） | 行按钮（折叠触发器） | — | 该工作区控件已渲染 ∧ 行未向上滚动出视口 | E/N | 工作区 E 14,229 288×18；year E 14,305 288×18 | `<ref>/components/sidebar/ProjectItem.vue:300`–`:343` |
| LB-35 | （无文案） | 图标（悬停出现） | ChevronRight（展开时旋转 90°） | LB-34 ∧ 行被悬停 | E/N（不可见） | E 14,231 14×14 | `<ref>/components/sidebar/ProjectItem.vue:307`–`:317` |
| LB-36 | （无文案） | 图标槽容器 | — | LB-34 | E/N | E 14,231 14×14 | `<ref>/components/sidebar/ProjectItem.vue:319`–`:336` |
| LB-37 | （无文案） | 图标 | ThemeIcon:workspace（🏠） | LB-36 ∧ 内置工作区 | E/N | E 14,231 14×14（工作区）；14,307（year 组旁为文件夹） | `<ref>/components/sidebar/ProjectItem.vue:323`–`:327` |
| LB-38 | （无文案） | 图标 | ThemeIcon:folder-open（展开）/ folder（收起） | LB-36 ∧ 非内置工作区 | E/N | E 14,375 14×14（dim-3） | `<ref>/components/sidebar/ProjectItem.vue:328`–`:335` |
| LB-39 | `{工作区名}` | 文本 | — | LB-34 | E/N | 工作区 33,229 37×18；year 33,305 26×18；dim-3 33,373 35×18 | `<ref>/components/sidebar/ProjectItem.vue:338`–`:341` |
| LB-40 | `{工作区路径}` | 浮层（tooltip） | — | 悬停名称 ∧ 该工作区有路径 | — | — | `<ref>/components/sidebar/ProjectItem.vue:433`–`:444` |
| LB-41 | （无文案） | 状态位（旋转） | Loader2 | 工作区收起 ∧ 有会话在跑 | — | — | `<ref>/components/sidebar/ProjectItem.vue:349`–`:356` |
| LB-42 | （无文案） | 图标按钮 | ThemeIcon:new-session | 行被悬停 ∨ 菜单打开 | E/N（不可见） | 工作区 302,227 21×21；year 302,303 21×21 | `<ref>/components/sidebar/ProjectItem.vue:361`–`:370` |
| LB-43 | （无文案） | 图标按钮（菜单触发器） | MoreHorizontal | 行被悬停 ∨ 菜单打开 | E/N（不可见） | 工作区 325,227 21×21；year 325,303 21×21 | `<ref>/components/sidebar/ProjectItem.vue:372`–`:383` |
| LB-44 | 置顶工作区 | 菜单项 | Pin | 工作区菜单打开 ∧ 非内置工作区 ∧ 未置顶 | — | — | `<ref>/components/sidebar/ProjectItem.vue:390`–`:397` |
| LB-45 | 取消置顶工作区 | 菜单项 | Pin | 同上 ∧ 已置顶 | — | — | `<ref>/components/sidebar/ProjectItem.vue:390`–`:397` |
| LB-46 | 在文件夹中显示 | 菜单项 | ThemeIcon:folder | 工作区菜单打开 ∧ （内置 ∨ 有路径） | — | — | `<ref>/components/sidebar/ProjectItem.vue:398`–`:405` |
| LB-47 | 归档会话 | 菜单项 | Archive | 工作区菜单打开 ∧ 该组会话数 > 0 | — | — | `<ref>/components/sidebar/ProjectItem.vue:406`–`:413` |
| LB-48 | 重命名 | 菜单项 | Edit3 | 工作区菜单打开 | — | — | `<ref>/components/sidebar/ProjectItem.vue:415`–`:418` |
| LB-49 | 移除工作区 | 菜单项（危险色） | Trash2 | 工作区菜单打开 ∧ 非内置工作区 | — | — | `<ref>/components/sidebar/ProjectItem.vue:419`–`:427` |
| LB-50 | 暂无会话 | 文本 | — | 组展开 ∧ 会话数 0 | E/N | 工作区 33,266 42×13；year 33,342；dim-3 组 33,466；testxxx 组 33,703；mark 组 33,1079；project 组 33,1203；dim-agent 组 33,1655 | `<ref>/components/sidebar/ProjectItem.vue:457`–`:462` |
| LB-51 | （无文案） | 状态行（旋转 + 加载中） | Loader2 | 组展开 ∧ 该组会话分页加载中 | — | — | `<ref>/components/sidebar/ProjectItem.vue:455` |
| LB-52 | （会话行实例） | 行 | 见 2.7 | 组展开 ∧ 每条会话 | — | — | `<ref>/components/sidebar/ProjectItem.vue:465`–`:468` |
| LB-53 | 上一页 | 文字按钮 | — | 组展开 ∧ 该组存在更早会话页 | — | — | `<ref>/components/sidebar/ProjectItem.vue:470` |
| LB-54 | 显示更多 | 文字按钮 | ChevronRight | 组展开 ∧ 该组尚有未展示会话 | — | — | `<ref>/components/sidebar/ProjectItem.vue:473`–`:489` |
| LB-55 | 显示更少 | 文字按钮 | ChevronRight（旋转 90°） | LB-54 ∧ 已超过阈值 ∧ 无隐藏项 ∧ 无更多页 | — | — | `<ref>/components/sidebar/ProjectItem.vue:481`–`:484` |
| LB-56 | `{工作区名}` | 占位（虚拟化代替行） | — | 控件已滚出视口（不渲染控件） | — | — | `<ref>/components/sidebar/ProjectItem.vue:494`–`:508` |
| LB-57 | （无文案） | 分隔线 | — | 内置工作区组之下 | E/N | E ≈7,295 346×1 | `<ref>/components/sidebar/ProjectItem.vue:511`–`:515` |

元素数：24

### 2.6 列表尾部与平铺视图（ThreadList）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| LB-58 | 显示更多 | 文字按钮 | — | 分组视图 ∧ 存在更多工作区页 | E/N | 7,1652 346×28 | `<ref>/components/sidebar/ThreadList.vue:240` |
| LB-59 | 上一页 | 文字按钮 | — | 分组视图 ∧ 存在更早工作区页 | — | — | `<ref>/components/sidebar/ThreadList.vue:239` |
| LB-60 | （无文案） | 拖拽落点指示线（两处） | — | 分组视图 ∧ 工作区拖拽落到第 0 位 / 第 n+1 位 | — | — | `<ref>/components/sidebar/ThreadList.vue:190`、`:216` |
| LB-61 | （会话行） | 行（复用 2.7） | — | 平铺视图 ∧ 每行 | — | — | `<ref>/components/sidebar/ThreadList.vue:249` |
| LB-62 | `{日期分组}` | 分组标题 | — | 平铺视图 ∧ 每个日期段（今天/昨天/更早/`{日期}`） | — | — | `<ref>/components/sidebar/ThreadList.vue:250` |
| LB-63 | （无文案） | 状态行（旋转） | Loader2 | 平铺视图 ∧ 会话分页加载中 | — | — | `<ref>/components/sidebar/ThreadList.vue:246` |
| LB-64 | 显示更多 | 文字按钮 | ChevronRight | 平铺视图 ∧ 尚有未展示项或更多页 | — | — | `<ref>/components/sidebar/ThreadList.vue:254`–`:269` |
| LB-65 | 显示更少 | 文字按钮 | ChevronRight（旋转 90°） | LB-64 ∧ 已超过阈值 ∧ 无隐藏项 ∧ 无更多页 | — | — | `<ref>/components/sidebar/ThreadList.vue:261`–`:264` |

元素数：8

### 2.7 会话行（ThreadItem）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| LB-66 | （无文案） | 状态位 | AlertCircle（红色） | 会话行 ∧ 该会话出错 | — | — | `<ref>/components/sidebar/ThreadItem.vue:260`–`:263` |
| LB-67 | （无文案） | 状态位（旋转） | Loader2 | 会话行 ∧ 运行中 ∧ 未悬停 | — | — | `<ref>/components/sidebar/ThreadItem.vue:264`–`:269` |
| LB-68 | （无文案） | 按钮 | Pin | 会话行 ∧ 悬停 ∧ 允许置顶切换 | — | — | `<ref>/components/sidebar/ThreadItem.vue:270`–`:289` |
| LB-69 | （无文案） | 图标 | Pin | 会话行 ∧ 置顶列表内（`showUnpinOnHover`）∧ 非悬停 | — | — | `<ref>/components/sidebar/ThreadItem.vue:281`–`:288` |
| LB-70 | （无文案） | 状态点 | — | 会话行 ∧ 未读 ∧ 不在上述分支 | — | — | `<ref>/components/sidebar/ThreadItem.vue:291`–`:294` |
| LB-71 | （无文案） | 图标 | Star | 会话行 ∧ 已置顶 ∧ 不隐藏置顶图标 ∧ 不在上述分支 | — | — | `<ref>/components/sidebar/ThreadItem.vue:295`–`:299` |
| LB-72 | `{会话标题}` | 文本（跑马灯） | — | 会话行 | — | — | `<ref>/components/sidebar/ThreadItem.vue:301`–`:304` |
| LB-73 | 等待操作 | 标签 | — | 会话行 ∧ 有待处理交互 | — | — | `<ref>/components/sidebar/ThreadItem.vue:308`–`:313` |
| LB-74 | `{+a −b}` | 文本 | — | 会话行 ∧ 有改动摘要 ∧ 办公模式外 | — | — | `<ref>/components/sidebar/ThreadItem.vue:317`–`:322` |
| LB-75 | `{⌘1..9}` | 快捷键徽标 | — | 会话行 ∧ 修饰键按住 | — | — | `<ref>/components/sidebar/ThreadItem.vue:329`–`:332` |
| LB-76 | （无文案） | 图标 | Clock3 | 会话行 ∧ 来源为定时任务 ∧ 未悬停 ∧ 无快捷键徽标 | — | — | `<ref>/components/sidebar/ThreadItem.vue:337`–`:345` |
| LB-77 | `{相对时间}` | 文本 | — | 会话行 ∧ 非定时来源 ∧ 未悬停 ∧ 无快捷键徽标 | — | — | `<ref>/components/sidebar/ThreadItem.vue:346`–`:352` |
| LB-78 | （无文案） | 按钮 | Archive | 会话行 ∧ 悬停 ∧ 无快捷键徽标 | — | — | `<ref>/components/sidebar/ThreadItem.vue:355`–`:364` |
| LB-79 | `{会话标题}` | 浮层（hover 卡） | — | 悬停会话行 | — | — | `<ref>/components/sidebar/ThreadItem.vue:376`–`:383` |
| LB-80 | `{所属工作区}` | 浮层（hover 卡） | ThemeIcon:workspace / folder | 悬停会话行 | — | — | `<ref>/components/sidebar/ThreadItem.vue:386`–`:392` |
| LB-81 | `{git 分支}` | 浮层（hover 卡） | GitBranch | 悬停会话行 ∧ 有分支 ∧ 办公模式外 | — | — | `<ref>/components/sidebar/ThreadItem.vue:393`–`:400` |
| LB-82 | `{分支不匹配警告}` | 浮层（hover 卡） | AlertCircle | 悬停会话行 ∧ 分支不一致 ∧ 办公模式外 | — | — | `<ref>/components/sidebar/ThreadItem.vue:401`–`:410` |

元素数：17

### 2.8 账户行（AccountEntry）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| LB-83 | （无文案） | 行按钮（菜单触发器） | — | 常显（贴底） | E/N | 7,688 316×30 | `<ref>/components/sidebar/AccountEntry.vue:211`–`:228` |
| LB-84 | `{首字母}` | 头像文本 | — | LB-83 ∧ 已登录 | E/N | 17,697 11×13 | `<ref>/components/sidebar/AccountEntry.vue:220` |
| LB-85 | （无文案） | 头像图标 | UserIcon | LB-83 ∧ 未登录 ∧ 非登录中 | — | — | `<ref>/components/sidebar/AccountEntry.vue:219` |
| LB-86 | （无文案） | 头像状态（旋转） | Loader2 | LB-83 ∧ 登录中 | — | — | `<ref>/components/sidebar/AccountEntry.vue:218` |
| LB-87 | （无文案） | 错误点 | — | LB-83 ∧ 有错误 ∧ 非登录中 | — | — | `<ref>/components/sidebar/AccountEntry.vue:221`–`:225` |
| LB-88 | `{账户名}` / 登录 Dim / 正在登录… | 文本 | — | LB-83 | E/N | E 39,694 281×18（文本为 `eric`） | `<ref>/components/sidebar/AccountEntry.vue:69`–`:78`、`:227` |
| LB-89 | （无文案） | 菜单 | — | 点击账户行 | — | — | `<ref>/components/sidebar/AccountEntry.vue:230`–`:360` |
| LB-90 | `{首字母}` + `{账户名}` | 菜单项（身份） | — | 账户菜单打开 | — | — | `<ref>/components/sidebar/AccountEntry.vue:242`–`:251` |
| LB-91 | 用量 | 菜单项 | Gauge | 账户菜单打开 ∧ 已登录 | — | — | `<ref>/components/sidebar/AccountEntry.vue:256`–`:270` |
| LB-92 | 剩余 {pct}% | 菜单项尾注 | — | LB-91 ∧ 已知剩余额度 | — | — | `<ref>/components/sidebar/AccountEntry.vue:264`–`:266` |
| LB-93 | 显示 Pet / 收起 Pet | 菜单项 | Smile | 账户菜单打开 ∧ 已登录 | — | — | `<ref>/components/sidebar/AccountEntry.vue:271`–`:289` |
| LB-94 | `{Pet 不可用说明}` | 菜单内说明 | — | 账户菜单打开 ∧ 浮动 Pet 不可用 | — | — | `<ref>/components/sidebar/AccountEntry.vue:290`–`:296` |
| LB-95 | 邀请朋友 | 菜单项 | Send | 账户菜单打开 ∧ 已登录 ∧ 邀请可用 | — | — | `<ref>/components/sidebar/AccountEntry.vue:298`–`:308` |
| LB-96 | 加载中 | 菜单项（禁用） | Loader2 | 账户菜单打开 ∧ 邀请信息加载中 | — | — | `<ref>/components/sidebar/AccountEntry.vue:310`–`:320` |
| LB-97 | 设置 | 菜单项 + 快捷键 | ThemeIcon:settings | 账户菜单打开 ∧ 已登录 | — | — | `<ref>/components/sidebar/AccountEntry.vue:321`–`:331` |
| LB-98 | 退出登录 | 菜单项 | LogOut | 账户菜单打开 ∧ 已登录 | — | — | `<ref>/components/sidebar/AccountEntry.vue:336`–`:346` |
| LB-99 | 登录 Dim | 菜单项 | UserIcon | 账户菜单打开 ∧ 未登录 | — | — | `<ref>/components/sidebar/AccountEntry.vue:348`–`:359` |
| LB-100 | 设置 | 图标按钮 | ThemeIcon:settings | 常显（贴底右侧） | E/N | 323,688 30×30 | `<ref>/components/sidebar/AccountEntry.vue:362`–`:371` |
| LB-101 | 退出 Dim？ | 对话框标题 | — | 请求退出确认 | — | — | `<ref>/components/sidebar/AccountEntry.vue:376`–`:380` |
| LB-102 | 退出后，需要重新登录才能使用 Dim 账户。 | 对话框描述 | — | LB-101 | — | — | `<ref>/components/sidebar/AccountEntry.vue:381`–`:383` |
| LB-103 | 取消 | 对话框按钮 | — | LB-101 | — | — | `<ref>/components/sidebar/AccountEntry.vue:386`–`:388` |
| LB-104 | 退出登录 | 对话框主按钮（危险色） | — | LB-101 | — | — | `<ref>/components/sidebar/AccountEntry.vue:389`–`:396` |

元素数：22

### 2.9 整栏拖入提示（AppSidebar）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| LB-105 | 松开以添加工作区 | 提示文本 | FolderPlus | 从文件管理器拖入文件夹 ∧ 提示可见 | — | — | `<ref>/components/sidebar/AppSidebar.vue:112`–`:121` |
| LB-106 | 仅限文件夹 · 同时新建会话 | 提示副文本 | — | LB-105 | — | — | `<ref>/components/sidebar/AppSidebar.vue:120` |

元素数：2

**区 2 元素数：106**

---

## 3. 区：时间线

E/N/R 三态没有会话，时间线容器被 `NewThread`（空态）占位，故本区除容器外全部未命中；未命中原因统一为「无会话」，个别再叠加数据/交互条件。

### 3.1 外壳与空态

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| TL-01 | （无文案） | 容器（滚动区） | — | 常显（空态时由空态节填充） | E/N/R | E ≈356,0 674×725 | `<ref>/components/chat-workspace/ChatWorkspace.vue:6585`–`:6616` |
| TL-02 | 当前会话还没有消息 | 文本 | — | 会话 ∧ 消息数 0 ∧ 未运行 ∧ 未处于恢复遮罩 | — | — | `<ref>/components/chat-workspace/timeline/PlainTimeline.vue:42`–`:44` |
| TL-03 | （无文案，仅读屏） | 加载条 | — | 会话 ∧ 正在加载更早历史 | — | — | `<ref>/components/chat-workspace/timeline/PlainTimeline.vue:48`–`:66` |
| TL-04 | （无文案） | 容器（用户消息轨道） | — | 会话 ∧ 轨道可用 | — | — | `<ref>/components/chat-workspace/timeline/PlainTimeline.vue:68`–`:81` |
| TL-05 | （无文案） | 标记（每条用户消息） | — | TL-04 ∧ 每条用户消息 | — | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessageRail.vue` |
| TL-06 | （无文案） | 占位（虚拟化上/下） | — | 会话 ∧ 渲染窗口外仍有行 | — | — | `<ref>/components/chat-workspace/timeline/PlainTimeline.vue:90`–`:96`、`:137`–`:143` |
| TL-07 | （无文案） | 行外壳 | — | 会话 ∧ 每条渲染行 | — | — | `<ref>/components/chat-workspace/timeline/PlainTimeline.vue:97`–`:110` |
| TL-08 | （无文案） | 图标按钮 | ChevronDown | 会话 ∧ 视口距底部超过阈值 | — | — | `<ref>/components/chat-workspace/timeline/PlainTimeline.vue:3610`–`:3617` |
| TL-09 | （无文案） | 骨架屏 | — | 会话切换/恢复中 ∧ 加载遮罩可见 | — | — | `<ref>/components/chat-workspace/timeline/TimelineSkeleton.vue`；挂载点 `ChatWorkspace.vue:6657`–`:6666` |
| TL-10 | 该消息渲染失败。 | 兜底文本 | — | 某行渲染抛错 | — | — | `<ref>/components/chat-workspace/timeline/TimelineErrorBoundary.vue` |

元素数：10

### 3.2 行类型与轮结构（TimelineRow）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| TL-11 | （无文案） | 轮容器 | — | 行类型 = 轮 | — | — | `<ref>/components/chat-workspace/timeline/TimelineRow.vue:4`–`:5` |
| TL-12 | （无文案） | 助手区容器 | — | 轮 ∧ （有内容块 ∨ 有终端信息） | — | — | `<ref>/components/chat-workspace/timeline/TimelineRow.vue:15`–`:18` |
| TL-13 | （无文案） | 轮脚容器 | — | 轮（运行中时整体 inert） | — | — | `<ref>/components/chat-workspace/timeline/TimelineRow.vue:48`–`:54` |
| TL-14 | （本轮产物块） | 组件 | — | 轮 ∧ 非演示模式 ∧ 非只读 ∧ 有 runId ∧ 是最新一轮 | — | — | `<ref>/components/chat-workspace/timeline/TimelineRow.vue:56`–`:65`（元素见 `inventory-panels.md` G） |
| TL-15 | （本轮改动块） | 组件 | — | 轮 ∧ 非演示模式 ∧ 非只读 ∧ 办公模式外 ∧ 有 runId ∧ 有用户消息 ∧ 是最新一轮 | — | — | `<ref>/components/chat-workspace/timeline/TimelineRow.vue:67`–`:77`（元素见 `inventory-panels.md` F） |
| TL-16 | （无文案） | 分隔行 | — | 行类型 = 显示 ∧ 窗口截断 | — | — | `<ref>/components/chat-workspace/timeline/TimelineRow.vue:110`–`:113` |
| TL-17 | （压缩分隔线） | 分隔行 | — | 行类型 = 显示 ∧ 压缩；或角色 = divider | — | — | `<ref>/components/chat-workspace/timeline/TimelineRow.vue:115`–`:119` |
| TL-18 | `{系统通知文本}` | 文本块 | — | 消息 ∧ 系统通知消息 | — | — | `<ref>/components/chat-workspace/timeline/TimelineRow.vue:135`–`:144` |
| TL-19 | `{消息正文}` | 文本块 | — | 消息 ∧ 其它角色 | — | — | `<ref>/components/chat-workspace/timeline/TimelineRow.vue:146`–`:148` |

元素数：9

### 3.3 用户消息（TimelineUserMessage）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| TL-20 | （无文案） | 气泡容器 | — | 用户消息 ∧ 有内容 ∧ 非编辑中 | — | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:952` |
| TL-21 | （无文案） | 图片廊 | — | 用户消息 ∧ 有图片附件 | — | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:902`–`:906` |
| TL-22 | （无文案） | 附件 chip 架 | — | 用户消息 ∧ 有 chip 附件 | — | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:909`–`:919` |
| TL-23 | （无文案） | 引用列表 | — | 用户消息 ∧ 有已发送引用 ∧ 非演示 ∧ 非编辑 | — | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:921`–`:926` |
| TL-24 | `mark {num}` + 选择器 | 标注卡 | — | 用户消息 ∧ 有标注 ∧ 非演示 ∧ 非编辑 | — | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:928`–`:950` |
| TL-25 | 定时唤醒来源 | 文本 | — | 用户消息 ∧ 元数据 `scheduleWakeup` | — | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:955`–`:961` |
| TL-26 | `{消息正文}` | 文本 | — | TL-20 | — | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:963`–`:1053` |
| TL-27 | `{技能名}` | 行内 chip | BoxIcon | 正文中技能提及 | — | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1004`–`:1012` |
| TL-28 | `{文件/目录名}` | 行内按钮 | 文件图标（图片） | 正文中文件提及 | — | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1013`–`:1030` |
| TL-29 | `{commit hash}` | 行内 chip | GitCommitHorizontal | 正文中提交提及 | — | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1031`–`:1039` |
| TL-30 | `{链接标题}` | 行内链接 | Globe / 站点图标 | 正文中链接提及 | — | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1040`–`:1052` |
| TL-31 | 展示全部 / 收起 | 文字按钮 | ChevronDown（展开时旋转） | TL-20 ∧ 内容溢出 | — | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1055`–`:1067` |
| TL-32 | （无文案） | 行内编辑器 | — | 用户消息 ∧ 有文本 ∧ （编辑中 ∨ 无内容） | — | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1073`–`:1108` |
| TL-33 | 取消 | 图标按钮 | X | TL-32 | — | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1090`–`:1096` |
| TL-34 | 发送消息... | 图标按钮 | ArrowUp | TL-32 | — | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1097`–`:1106` |
| TL-35 | `{发送时间}` | 文本 | — | 用户消息 ∧ （有内容 ∨ 有附件）∧ 非编辑 | — | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1118`–`:1123` |
| TL-36 | 取消（撤回） | 图标按钮 | X | 同 TL-35 ∧ 可撤回 | — | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1124`–`:1135` |
| TL-37 | 复制整个消息 | 图标按钮 | Copy / Check | 同 TL-35 | — | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1136`–`:1153` |
| TL-38 | 编辑消息 | 图标按钮 | Pencil | 同 TL-35 ∧ 可编辑 ∧ 有文本 | — | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1154`–`:1170` |
| TL-39 | 从此处分叉 | 图标按钮 | GitFork | 同 TL-35 ∧ 可分叉 ∧ 非弹窗窗口 | — | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1171`–`:1186` |
| TL-40 | 从此处开始侧边对话 | 图标按钮 | GitBranch | 同 TL-35 ∧ 可分支 ∧ 非弹窗窗口 | — | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1187`–`:1202` |
| TL-41 | 从此处开始侧边对话 | 右键菜单项 | GitBranch | 用户消息右键 ∧ 可分支 | — | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1206`–`:1213` |
| TL-42 | 复制 | 右键菜单项 | Copy | 用户消息右键 ∧ 已选中文本 | — | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1214`–`:1221` |
| TL-43 | 复制整个消息 | 右键菜单项 | Copy | 用户消息右键 | — | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1222`–`:1229` |
| TL-44 | （无文案） | 选区引用浮层 | — | 非只读 ∧ 选中文本 ∧ 浮层打开 | — | — | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1237`–`:1244` |

元素数：25

### 3.4 助手消息（TimelineAssistantMessage 及块组件）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| TL-45 | 工作过程 / `工作过程 · {actions}` | 文字按钮（折叠行） | ChevronRight（展开旋转） | 助手消息 ∧ 有被收起的内容块 ∧ 允许显示折叠 | — | — | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:895`–`:915` |
| TL-46 | `已经工作了 {duration}` / `已经工作了 {duration} · {actions}` | 同上 | 同上 | 同上（运行中/完成） | — | — | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:560`–`:574` |
| TL-47 | 已停止 / `已停止 · {duration}` | 同上 | 同上 | 同上（被停止） | — | — | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:556`–`:560` |
| TL-48 | `{duration} 后失败` / `{duration} 后失败 · {actions}` | 同上（警告色） | 同上 | 同上（失败） | — | — | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:542`–`:550` |
| TL-49 | 出错前的工作过程 / `出错前的工作过程 · {actions}` | 同上 | 同上 | 同上（出错且无终端信息） | — | — | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:551`–`:554` |
| TL-50 | （无文案） | 块容器 | — | 助手消息 | — | — | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:917` |
| TL-51 | （工具调用分组） | 折叠组 | — | 连续工具调用被合并 | — | — | `<ref>/components/chat-workspace/timeline/TimelineBlockGroup.vue` |
| TL-52 | 正在思考 / `思考了 {duration}` | 折叠行 | — | 思考块 | — | — | `<ref>/components/chat-workspace/timeline/TimelineThinkingItem.vue` |
| TL-53 | （markdown 正文） | 富文本 | — | 文本块 | — | — | `<ref>/components/chat-workspace/timeline/TimelineTextItem.vue` |
| TL-54 | （工具调用标题行） | 行按钮 | 工具图标 | 工具调用块 | — | — | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue` |
| TL-55 | （工具调用详情） | 展开区 | ChevronRight | 工具调用块 ∧ 已展开 | — | — | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue` |
| TL-56 | `{子代理名}` + `{通知文本}` | 行按钮（chip） | Bot + ProviderIcon | 通知块 ∧ 子代理通知 | — | — | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:986`–`:1045` |
| TL-57 | 运行失败 / 本次运行失败。 | 文本 | — | 块状态失败 | — | — | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:773`–`:776` |
| TL-58 | `{seconds}s 后重试` / 重试中 / 已重试 / 已取消 | 提示 + 按钮 | — | 重试通知块 | — | — | `<ref>/components/chat-workspace/timeline/TimelineRetryNotice.vue:36`–`:47` |
| TL-59 | （终端错误详情标题）+ 复制 | 折叠区 + 图标按钮 | Copy | 终端失败 ∧ 有调试详情 | — | — | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1247`–`:1262` |
| TL-60 | 用户中断 | 通知文本 | — | 运行被中断 | — | — | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:776` |
| TL-61 | 复制整个消息 | 图标按钮 | Copy / Check | 轮末动作条 ∧ 有可复制内容 | — | — | `<ref>/components/chat-workspace/timeline/TimelineAssistantActions.vue:13`–`:20` |
| TL-62 | 从此处分叉 | 图标按钮 | GitFork | 轮末动作条 ∧ 有分叉锚点 ∧ 非弹窗窗口 | — | — | `<ref>/components/chat-workspace/timeline/TimelineAssistantActions.vue:21`–`:28` |
| TL-63 | 重试（或自定义重试文案） | 图标按钮 | RotateCcw | 轮末动作条 ∧ 可重试 | — | — | `<ref>/components/chat-workspace/timeline/TimelineAssistantActions.vue:29`–`:36` |
| TL-64 | `{完成时间}` | 文本 | — | 轮末动作条 ∧ 有完成时间 | — | — | `<ref>/components/chat-workspace/timeline/TimelineAssistantActions.vue:37`–`:41` |
| TL-65 | 复制 / 复制整个消息 | 右键菜单项（2 条） | Copy | 助手消息右键 | — | — | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1298`–`:1315` |

元素数：21

### 3.5 分隔与压缩

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| TL-66 | （无文案） | 分隔线 | — | 窗口截断行 | — | — | `<ref>/components/chat-workspace/timeline/TimelineWindowTruncationDivider.vue` |
| TL-67 | 已压缩 / 自动压缩 + `(部分)` | 分隔行文本 | — | 压缩行（完成态） | — | — | `<ref>/components/chat-workspace/timeline/TimelineCompressionDivider.vue:75`–`:78` |
| TL-68 | 正在压缩 / 压缩失败 / 已跳过 / 已取消 | 分隔行文本 | Loader2 | 压缩行（进行/失败/跳过/取消） | — | — | `<ref>/components/chat-workspace/timeline/TimelineCompressionDivider.vue:63`–`:73` |
| TL-69 | 摘要生成中 / 摘要不可用 / 加载失败 + 重试 | 分隔行内文本/按钮 | — | 压缩行展开 | — | — | `<ref>/components/chat-workspace/timeline/TimelineCompressionDivider.vue:192`–`:217` |

元素数：4

**区 3 元素数：69**

---

## 4. 区：输入区

E/N/R 的输入区由空态（NewThread）挂载 `InputArea`；会话态由 `ChatWorkspace` 底部栈挂载同一个 `InputArea`。包装差异见第 6 节。

### 4.1 输入框与工具行

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| IN-01 | （无文案） | 容器（圆角描边输入块） | — | 常显（空态整块带投影，控制行灰底） | E/N/R | E ≈396,372 582×110 | `<ref>/components/chat-workspace/composer/ConversationComposer.vue:59`–`:66` |
| IN-02 | 发送消息... | 占位文本 | — | 常显（草稿为空） | E/N/R | E ≈425,399（首行文字左上） | `<ref>/components/chat-workspace/composer/InputArea.vue:2334` |
| IN-03 | 斜杠命令菜单 | 浮层 | — | 输入 `/` 且编辑器聚焦 | — | — | `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue:178`–`:183` |
| IN-04 | 提示建议（接受/拒绝） | 行内浮层 | — | 有提示建议 | — | — | `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue` |
| IN-05 | （无文案） | 图标按钮 | Plus | 常显（工具行首个） | E/N/R | E ≈424,461 28×28 | `<ref>/components/chat-workspace/composer/InputArea.vue:2460`–`:2467` |
| IN-06 | （无文案） | 图标按钮（模式选择器） | ChevronsRight / Target | 常显 | E/N/R | E ≈462,461 28×28（隐藏标签 `Agent` 463,456 39×14） | `<ref>/components/chat-workspace/composer/AgentModeSelector.vue:4`–`:30` |
| IN-07 | （无文案） | 图标按钮（技能选择器） | Box | 常显 | E/N/R | E ≈480,461 28×28（隐藏标签 `技能` 491,455 33×18） | `<ref>/components/chat-workspace/composer/ComposerSkillsSelector.vue:9`–`:18` |
| IN-08 | 高（推理强度短标签） | 按钮（四格条 + 文本） | 四格条（自绘） | 当前模型有推理强度选项 | E/N/R | E 802,455 25×18 | `<ref>/components/chat-workspace/model-selector/EffortControl.vue:6`–`:32` |
| IN-09 | `{模型名}` | 按钮（模型选择器） | ProviderIcon | 模型选择器可打开 | E/N/R | E 837,451 110×25；文本 865,455 73×18 | `<ref>/components/chat-workspace/model-selector/ModelSelector.vue:22`–`:48` |
| IN-10 | （无文案） | 图标按钮（ZDR 徽标位） | ShieldCheck 图标（ZdrBadge） | 模型 `zdr === true` ∧ 非纯图标模式 | — | — | `<ref>/components/chat-workspace/model-selector/ModelSelector.vue:41`–`:44` |
| IN-11 | （无文案） | 图标 | Zap | 模型服务档 = priority | — | — | `<ref>/components/chat-workspace/model-selector/ModelSelector.vue:45`–`:49` |
| IN-12 | 发送消息... | 图标按钮（发送） | ArrowUp | 发送动作为「发送」 | E/N/R | E 951,451 25×25 | `<ref>/components/chat-workspace/composer/ConversationComposer.vue:86`–`:90` |
| IN-13 | 停止回复 | 图标按钮（停止） | Square | 发送动作为「停止」 | — | — | `<ref>/components/chat-workspace/composer/ConversationComposer.vue:88`–`:90` |
| IN-14 | Esc | 键帽 | — | 停止动作 ∧ 停止确认已武装 | — | — | `<ref>/components/chat-workspace/composer/ConversationComposer.vue:84`–`:86` |
| IN-15 | `{发送不可用原因}` | 文本 | — | 不允许发送 ∧ 不允许排队 ∧ 发送态已确定 | — | — | `<ref>/components/chat-workspace/composer/InputArea.vue:2309`–`:2310` |
| IN-16 | 重试 | 文字按钮 | — | IN-15 ∧ 有活动会话 ∧ 发送能力未知 | — | — | `<ref>/components/chat-workspace/composer/InputArea.vue:2311` |

元素数：16

### 4.2 附件菜单、紧凑态与上抬区

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| IN-17 | 添加附件 | 菜单项 | Paperclip | 紧凑浮层 ∧ 展开 `+` 菜单 | — | — | `<ref>/components/chat-workspace/composer/InputArea.vue:2371`–`:2374` |
| IN-18 | Agent / 目标 | 菜单项（子菜单容器） | ChevronsRight / Target | IN-17 所在菜单 ∧ 悬停该子菜单 | — | — | `<ref>/components/chat-workspace/composer/InputArea.vue:2375`–`:2406` |
| IN-19 | Agent | 菜单项 | ChevronsRight | IN-18 ∧ 子菜单展开 | — | — | `<ref>/components/chat-workspace/composer/InputArea.vue:2383`–`:2391` |
| IN-20 | 目标 | 菜单项 | Target | IN-18 ∧ 子菜单展开（可用性见「检查中/不可用」提示） | — | — | `<ref>/components/chat-workspace/composer/InputArea.vue:2392`–`:2404` |
| IN-21 | 技能 | 菜单项 | Box | IN-17 所在菜单 | — | — | `<ref>/components/chat-workspace/composer/InputArea.vue:2407`–`:2409` |
| IN-22 | （无文案） | 图标按钮（紧凑态 `+`） | Plus | 紧凑浮层 | — | — | `<ref>/components/chat-workspace/composer/InputArea.vue:2358`–`:2368` |
| IN-23 | 正在编辑排队消息 | 文本 | — | 编辑排队消息 ∧ 非紧凑浮层 | — | — | `<ref>/components/chat-workspace/composer/InputArea.vue:2414`–`:2420` |
| IN-24 | （无文案） | 占位（应用快照展开高度） | — | 应用快照区域高度已确定 | — | — | `<ref>/components/chat-workspace/composer/InputArea.vue:2421`–`:2426` |
| IN-25 | （无文案） | 图片附件廊 | — | 有图片附件 | — | — | `<ref>/components/chat-workspace/composer/InputArea.vue:2427`–`:2432` |
| IN-26 | （无文案） | chip 架 | — | 有引用或 chip 附件 | — | — | `<ref>/components/chat-workspace/composer/InputArea.vue:2433`–`:2453` |

元素数：10

### 4.3 底部栈（会话态，`ChatWorkspace`）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| IN-27 | （无文案） | 容器（底部栈） | — | 会话 ∧ 非全宽面板聊天 | — | — | `<ref>/components/chat-workspace/ChatWorkspace.vue:6671`–`:6743` |
| IN-28 | （活动条：目标） | chip | — | 会话 ∧ 目标进行中 | — | — | `<ref>/components/chat-workspace/composer/activity/GoalActivityChip.vue` |
| IN-29 | （活动条：待办） | chip | — | 会话 ∧ 有待办项 | — | — | `<ref>/components/chat-workspace/composer/activity/TodoActivityChip.vue` |
| IN-30 | （活动条：子代理） | chip | — | 会话 ∧ 有运行中子代理 | — | — | `<ref>/components/chat-workspace/composer/activity/AgentActivityChip.vue` |
| IN-31 | （活动条：分支） | chip | — | 会话 ∧ 有分支项 | — | — | `<ref>/components/chat-workspace/composer/activity/BranchActivityChip.vue` |
| IN-32 | （活动条：唤醒） | chip | — | 会话 ∧ 有唤醒项 ∨ 面板打开 | — | — | `<ref>/components/chat-workspace/composer/activity/TaskActivityChip.vue:21` |
| IN-33 | （活动条：后台进程） | chip | — | 会话 ∧ 有进程 ∨ 面板打开 | — | — | `<ref>/components/chat-workspace/composer/activity/TaskActivityChip.vue:29` |
| IN-34 | （活动条：文件改动） | chip | — | 会话 ∧ 有文件改动摘要 | — | — | `<ref>/components/chat-workspace/composer/activity/FileChangesActivityChip.vue` |
| IN-35 | （排队消息面板） | 容器 | — | 会话 ∧ 排队消息数 > 0 | — | — | `<ref>/components/chat-workspace/composer/FollowUpMessagesView.vue:2`–`:8` |
| IN-36 | `{count} 条消息`（计数文案） | 文本 | List | IN-35 | — | — | `<ref>/components/chat-workspace/composer/FollowUpMessagesView.vue:6` |
| IN-37 | `{状态}` + `{消息文本}` | 行（转向中） | Loader2 | IN-35 ∧ 该条为转向中 | — | — | `<ref>/components/chat-workspace/composer/FollowUpMessagesView.vue:12`–`:19` |
| IN-38 | `{消息文本}` | 行（待发） | — | IN-35 ∧ 该条待发 | — | — | `<ref>/components/chat-workspace/composer/FollowUpMessagesView.vue:20`–`:26` |
| IN-39 | （无文案） | 拖拽把手 | GripVertical | IN-38 ∧ 未锁序 ∧ 悬停该行 | — | — | `<ref>/components/chat-workspace/composer/FollowUpMessagesView.vue:28`–`:33` |
| IN-40 | 编辑 | 图标按钮 | Pencil | IN-38 ∧ 允许编辑 ∧ 悬停该行 | — | — | `<ref>/components/chat-workspace/composer/FollowUpMessagesView.vue:40`–`:45` |
| IN-41 | 移除 | 图标按钮 | Trash2 | IN-38 ∧ 悬停该行 | — | — | `<ref>/components/chat-workspace/composer/FollowUpMessagesView.vue:46`–`:51` |
| IN-42 | 转向 | 文字按钮 | — | IN-38 ∧ 运行中 | — | — | `<ref>/components/chat-workspace/composer/FollowUpMessagesView.vue:52`–`:55` |
| IN-43 | （无文案） | 权限请求面板 | — | 会话 ∧ 有权限交互 | — | — | `<ref>/components/chat-workspace/interactions/InlinePermissionPrompt.vue:1`–`:4` |
| IN-44 | （无文案） | 澄清请求面板 | — | 会话 ∧ 无权限交互 ∧ 有澄清交互 | — | — | `<ref>/components/chat-workspace/interactions/InlineElicitationPrompt.vue` |
| IN-45 | （无文案） | 面板容器（Shell 运行） | Terminal | 会话 ∧ 有活动 shell 运行 | — | — | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:2`–`:10` |
| IN-46 | `{命令}` | 等宽文本 | — | IN-45 | — | — | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:7` |
| IN-47 | `{模式}` | 标签 | — | IN-45 | — | — | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:8`–`:10` |
| IN-48 | `{状态}` | 状态文本 | Loader2 | IN-45 | — | — | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:11`–`:14` |
| IN-49 | 停止 | 按钮 | Square | IN-45 ∧ 运行中 | — | — | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:15`–`:23` |
| IN-50 | 关闭 | 按钮 | X | IN-45 ∧ 非运行中 | — | — | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:24`–`:31` |
| IN-51 | `{输出}` | 等宽块 | — | IN-45 ∧ 有输出 | — | — | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:34`–`:36` |
| IN-52 | `{跳过数}` / `{输出已截断}` / `{进程退出码}` / `{错误}` / `{原因}` / `{录制结果}` | 脚注文本（6 条） | — | IN-45 ∧ 各自条件 | — | — | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:37`–`:59` |

元素数：26

### 4.4 选择器浮层（模型 / 推理强度 / 技能 / 模式）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| IN-53 | 模型选择浮层 | 浮层容器 | — | 模型选择器可打开 ∧ 浮层打开 | — | — | `<ref>/components/chat-workspace/model-selector/ModelSelector.vue:66`–`:80` |
| IN-54 | 标签页（对话 / 子代理策略 / 外部 Agent） | 文本标签（多于一个时才显示） | — | IN-53 ∧ 子代理页 ∨ 外部 Agent 页可用 | — | — | `<ref>/components/chat-workspace/model-selector/ModelSelector.vue:82`–`:105` |
| IN-55 | 搜索模型... | 输入框 | search | IN-53 | — | — | `<ref>/components/chat-workspace/model-selector/ModelSelector.vue:115` 起 |
| IN-56 | 未配置 / 模型加载中… / 加载失败 · 重试 / 暂无可用模型 | 文本（4 条） | — | IN-53 ∧ 各自状态 | — | — | `<ref>/components/chat-workspace/model-selector/ModelSelector.vue:121` 起 |
| IN-57 | 思考强度 | 浮层（滑杆） | — | 点开推理强度按钮 | — | — | `<ref>/components/chat-workspace/model-selector/EffortControl.vue:34`–`:47`；`SmartnessSlider.vue` |
| IN-58 | Fast 模式 | 浮层（选择） | — | 模型支持服务档 | — | — | `<ref>/components/chat-workspace/model-selector/ServiceTierControl.vue` |
| IN-59 | 搜索技能 | 输入框 | search（ThemeIcon） | 技能选择器浮层打开 | — | — | `<ref>/components/chat-workspace/composer/ComposerSkillsSelector.vue:25`–`:26` |
| IN-60 | 加载中 | 文本 | — | 技能浮层打开 ∧ 加载中 | — | — | `<ref>/components/chat-workspace/composer/ComposerSkillsSelector.vue:29` |
| IN-61 | `{错误}` | 文本（错误色） | — | 技能浮层打开 ∧ 出错 | — | — | `<ref>/components/chat-workspace/composer/ComposerSkillsSelector.vue:30` |
| IN-62 | `{无技能文案}` | 文本 | — | 技能浮层打开 ∧ 列表为空 | — | — | `<ref>/components/chat-workspace/composer/ComposerSkillsSelector.vue:31` |
| IN-63 | `{技能名}` + `{技能描述}` | 菜单项 | — | 技能浮层打开 ∧ 每个技能 | — | — | `<ref>/components/chat-workspace/composer/ComposerSkillsSelector.vue:33`–`:37` |
| IN-64 | Agent / 目标 | 菜单项 | ChevronsRight / Target | 模式选择器菜单打开 | — | — | `<ref>/components/chat-workspace/composer/AgentModeSelector.vue:34`–`:63` |
| IN-65 | `{检查中}` / `{不可用}` | 菜单项提示 | Loader2（检查中） | IN-64 的「目标」项 ∧ 可用性未定 | — | — | `<ref>/components/chat-workspace/composer/AgentModeSelector.vue:52`–`:62` |

元素数：13

**区 4 元素数：65**

---

## 5. 区：状态条（StatusBar）

`StatusBar` 只在会话工作区渲染（`ChatWorkspace.vue:6738`–`:6741`），E/N/R 均无，故整区未命中；空态里的权限选择器挂在空态控制行（见第 6 节 NT-14），属同一组件、不同挂载点。

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| SB-01 | （无文案） | 容器（状态条） | — | 会话 ∧ 非紧凑浮层 ∧ 非全宽面板聊天 | — | — | `<ref>/components/chat-workspace/StatusBar.vue:3`–`:5` |
| SB-02 | 默认权限 / 智能权限 / 完全访问权限 | 按钮（权限选择器） | 盾形图标（随权限变） | 会话（工具栏左） | — | — | `<ref>/components/chat-workspace/ToolPermissionSelector.vue:3`–`:6` |
| SB-03 | 默认权限 / 智能权限 / 完全访问权限 | 菜单项（3 条） | 各自图标 | 权限菜单打开 | — | — | `<ref>/components/chat-workspace/ToolPermissionSelector.vue:10`–`:28` |
| SB-04 | 确认开启完全放开？ | 对话框标题 | AlertTriangle | 选择「完全访问权限」 | — | — | `<ref>/components/chat-workspace/ToolPermissionSelector.vue:40`–`:44` |
| SB-05 | 开启后，AI 将减少确认步骤… | 对话框描述 | — | SB-04 | — | — | `<ref>/components/chat-workspace/ToolPermissionSelector.vue:45`–`:47` |
| SB-06 | 我已了解风险，并愿意继续 | 复选框 | — | SB-04 | — | — | `<ref>/components/chat-workspace/ToolPermissionSelector.vue:49`–`:54` |
| SB-07 | 取消 / 确认开启 | 对话框按钮（2 条） | — | SB-04 | — | — | `<ref>/components/chat-workspace/ToolPermissionSelector.vue:56`–`:63` |
| SB-08 | Agent 配置 | 按钮（配置选择器） | — | 会话 ∧ 该会话提供多套 Agent 配置 | — | — | `<ref>/components/chat-workspace/StatusBar.vue:12`–`:21` |
| SB-09 | `{当前分支}` | 按钮（分支选择器） | GitBranch | 会话 ∧ 有分支 ∧ 非办公模式 | — | — | `<ref>/components/chat-workspace/StatusBar.vue:28`–`:35` |
| SB-10 | 搜索分支 | 输入框 | search（ThemeIcon） | 分支菜单打开 | — | — | `<ref>/components/chat-workspace/StatusBar.vue:38`–`:46` |
| SB-11 | 分支 | 分组标题 | — | 分支菜单打开 | — | — | `<ref>/components/chat-workspace/StatusBar.vue:50`–`:52` |
| SB-12 | `{分支名}` | 菜单项 | GitBranch + Check | 分支菜单打开 ∧ 每个分支 | — | — | `<ref>/components/chat-workspace/StatusBar.vue:55`–`:71` |
| SB-13 | 创建新分支 | 菜单项 | Plus | 分支菜单打开 | — | — | `<ref>/components/chat-workspace/StatusBar.vue:75`–`:78` |
| SB-14 | （无文案） | 用量表（上下文用量触发器） | 环形/条形（自绘） | 会话 ∧ 上下文用量可见 | — | — | `<ref>/components/chat-workspace/ContextUsageMeter.vue` |
| SB-15 | 本轮上下文 / 会话消耗：{tokens} tokens / 缓存命中 {pct}% / 速率：{rate} tok/s | 浮层文本（4 条） | — | 悬停或点开用量表 | — | — | `<ref>/i18n/zh-CN.ts:1193`–`:1196`；`ContextUsageMeter.vue` |
| SB-16 | 计划 | 浮层分组标题 | — | 用量浮层 ∧ 计划用量可见 | — | — | `<ref>/components/chat-workspace/StatusBar.vue:91`–`:94` |
| SB-17 | 账户设置 | 浮层文字按钮 | ArrowUpRight | 同上 ∧ 非弹窗窗口 | — | — | `<ref>/components/chat-workspace/StatusBar.vue:96`–`:105` |
| SB-18 | `{用量错误}` | 文本（错误色） | — | 用量浮层 ∧ 有错误 | — | — | `<ref>/components/chat-workspace/StatusBar.vue:106`–`:108` |
| SB-19 | `{无限额度说明}` | 文本 | — | 用量浮层 ∧ 无限额 ∧ 无明细行 | — | — | `<ref>/components/chat-workspace/StatusBar.vue:110`–`:115` |
| SB-20 | `{加载中 / 空文案}` | 文本 | — | 用量浮层 ∧ 无明细行 ∧ 无错误 | — | — | `<ref>/components/chat-workspace/StatusBar.vue:116`–`:118` |
| SB-21 | `{行标签}` + `{pct}%` + `{重置时间}` + 进度条 | 明细行 | — | 用量浮层 ∧ 有明细行 | — | — | `<ref>/components/chat-workspace/StatusBar.vue:120`–`:143` |

元素数：21

**区 5 元素数：21**

---

## 6. 空态（NewThread，单独成节）

空态与右栏同时存在时（R/Q），输入区宽度压缩但元素集合不变；坐标随卡片宽度变化。

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| NT-01 | （无文案） | 品牌图形 | BrandMark（品牌绿） | 空态 | E/N/R | E ≈651,247 56×56 | `<ref>/components/chat-workspace/NewThread.vue:298` |
| NT-02 | 开始构建 | 标题（h1） | — | 空态 | E/N/R | E 651,313 84×28；R 160,313 84×28 | `<ref>/components/chat-workspace/NewThread.vue:299` |
| NT-03 | 加载中 | 状态文本 | — | 空态 ∧ 会话目录初次选择进行中 | — | — | `<ref>/components/chat-workspace/NewThread.vue:300` |
| NT-04 | （无文案） | 容器（输入块 + 阴影） | — | 空态 | E/N/R | E ≈396,372 582×110 | `<ref>/components/chat-workspace/NewThread.vue:306`–`:308` |
| NT-05 | （无文案） | 容器（控制行，圆角底 + 灰底） | — | 空态 | E/N/R | E ≈396,482 582×40 | `<ref>/components/chat-workspace/NewThread.vue:314`–`:325` |
| NT-06 | `{工作区名}` / 选择工作区... | 按钮（工作区选择器） | ThemeIcon:workspace / folder | 空态 | E/N/R | E 433,498 32×14；R 77,498 32×14（文本均为 `工作区`） | `<ref>/components/chat-workspace/NewThread.vue:326`–`:343` |
| NT-07 | 搜索工作区... | 输入框 | search（ThemeIcon） | 工作区下拉打开 | — | — | `<ref>/components/chat-workspace/NewThread.vue:346`–`:353` |
| NT-08 | `{内置工作区名}` | 菜单项 | ThemeIcon:workspace + 单选圈 | 工作区下拉 ∧ 每个内置工作区 | — | — | `<ref>/components/chat-workspace/NewThread.vue:360`–`:380` |
| NT-09 | `{工作区名}` + `{路径}` | 菜单项 | 单选圈 | 工作区下拉 ∧ 每个普通工作区 | — | — | `<ref>/components/chat-workspace/NewThread.vue:385`–`:412` |
| NT-10 | 上一页 / 显示更多 | 文字按钮（2 条） | — | 工作区下拉 ∧ 各自分页条件 | — | — | `<ref>/components/chat-workspace/NewThread.vue:419`–`:420` |
| NT-11 | 添加新工作区 / 正在打开... | 菜单项 | FolderPlus | 工作区下拉 | — | — | `<ref>/components/chat-workspace/NewThread.vue:425`–`:431` |
| NT-12 | 选择分支... / `{分支名}` | 按钮（分支选择器） | GitBranchIcon | 空态 ∧ 支持分支选择 ∧ 非办公模式 | — | — | `<ref>/components/chat-workspace/NewThread.vue:434`–`:452` |
| NT-13 | 搜索分支... / 无分支 / `{分支名}` | 输入框 + 菜单项 | search + GitBranchIcon + Check | 分支下拉打开 | — | — | `<ref>/components/chat-workspace/NewThread.vue:457`–`:499` |
| NT-14 | 默认权限 / 智能权限 / 完全访问权限 | 按钮（权限选择器） | 盾形图标 | 空态（控制行第三项） | E/N/R | E 521,498 42×14；R 165,498 42×14 | `<ref>/components/chat-workspace/NewThread.vue:499`（组件 `ToolPermissionSelector.vue`） |
| NT-15 | 总结 PDF / 改写 Word 文档 / 做一版 PPT / 整理成表格 / 帮我调研 / 翻译 | 推荐卡（6 张） | 各自图标 | 空态 ∧ 办公模式 | — | — | `<ref>/components/chat-workspace/NewThread.vue:505`–`:521` |

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
| OV-04 | （无文案） | 关闭页签按钮 | X | 页签可关闭 ∧ 悬停该页签 | — | — | `<ref>/components/right-panel/RightPanelBar.vue:100`–`:112` |
| OV-05 | `{⌘n}` | 快捷键徽标 | — | 修饰键按住 ∧ 右栏打开 ∧ 页签 > 1 | — | — | `<ref>/components/right-panel/RightPanelBar.vue:93`–`:98` |
| OV-06 | （无文案） | 图标按钮（添加页签） | Plus | 有可添加的页签类型 | R | R ≈953,16 28×28 | `<ref>/components/right-panel/RightPanelBar.vue:381`–`:397` |
| OV-07 | 文件 / 改动 / 浏览器 / 终端 / 侧边对话 / 界面扩展 | 菜单项（按可用性出现） | Folders / GitCompare / Globe / SquareTerminal / MessagesSquare / Blocks | OV-06 菜单打开 | — | — | `<ref>/components/right-panel/RightPanelBar.vue:133`–`:152`、`:399`–`:410` |
| OV-08 | 展开面板 | 图标按钮 | Maximize2 | 非紧凑布局 ∧ 面板未展开 | R | R 994,16 25×25 | `<ref>/components/right-panel/RightPanelBar.vue:419`–`:428` |
| OV-09 | 恢复面板宽度 | 图标按钮 | Minimize2 | 面板已展开 | — | — | `<ref>/components/right-panel/RightPanelBar.vue:419`–`:428` |
| OV-10 | （无文案） | 返回按钮 | ArrowLeft | 全宽替换布局 | — | — | `<ref>/components/right-panel/RightPanelBar.vue:29`–`:38` |
| OV-11 | 筛选文件... | 输入框 | Search | 文件页 ∨ 改动页激活 | R | R ≈400,62 600×24（占位文字左上 ≈425,69） | `<ref>/components/context-panel/ContextPanel.vue:539`–`:553` |
| OV-12 | `{目录名}` | 树节点行（可折叠） | ChevronRight + 文件夹图标 | 文件页激活 ∧ 每个目录 | R | R 462,100 553×18（首项） | `<ref>/components/context-panel/FileTreeNode.vue` |
| OV-13 | `{文件名}` | 树节点行 | 文件图标 | 文件页激活 ∧ 每个文件 | R | R 461,1025 555×18（首项） | `<ref>/components/context-panel/FileTreeNode.vue` |
| OV-14 | （无文案） | 状态块 | — | 文件页 ∧ 首次加载 | — | — | `<ref>/components/context-panel/FilesView.vue:3`–`:5` |
| OV-15 | 正在搜索工作区 / 搜索失败 + 重试 / 搜索不完整 / 结果条数上限 / 无匹配文件 + 说明 | 文本（6 条） | FileIcon | 文件页 ∧ 搜索模式 ∧ 各自状态 | — | — | `<ref>/components/context-panel/FilesView.vue:8`–`:60` |
| OV-16 | 无权限提示 + 重试 | 警告条 | AlertTriangle | 目录读取被拒 | — | — | `<ref>/components/context-panel/ContextPanel.vue:556`–`:576` |
| OV-17 | `{范围}` | 下拉（改动范围） | — | 改动页激活 | — | — | `<ref>/components/context-panel/ContextPanel.vue:440`–`:461` |
| OV-18 | 提交或 Push | 按钮（菜单） | CloudUpload + ChevronDown | 改动页 ∧ 可提交 | — | — | `<ref>/components/context-panel/ContextPanel.vue:465`–`:478` |
| OV-19 | （无文案） | 图标按钮（刷新） | RefreshCw | 改动页激活 | — | — | `<ref>/components/context-panel/ContextPanel.vue:523`–`:531` |

元素数：19

### 7.2 「搜索聊天」弹窗（Q、S、K）

| 元素 ID | 名称（文案原文） | 控件 | 图标 | 条件 | 在状态 | 位置 | 定位 |
|---|---|---|---|---|---|---|---|
| OV-20 | （无文案） | 对话框容器（遮罩 + 弹窗） | — | 全局搜索打开 | Q/S/K | Q 291,293 448×139 | `<ref>/components/sidebar/ChatSearchDialog.vue:185`–`:190` |
| OV-21 | 搜索聊天 | 标题（视觉隐藏） | — | OV-20 | Q/S/K | Q 291,293（宽度采集失真） | `<ref>/components/sidebar/ChatSearchDialog.vue:192` |
| OV-22 | 搜索并打开已有聊天 | 描述（视觉隐藏） | — | OV-20 | Q/S/K | Q 291,316（宽度采集失真） | `<ref>/components/sidebar/ChatSearchDialog.vue:193` |
| OV-23 | 搜索聊天（占位） | 输入框 | search（ThemeIcon） | OV-20 | Q/S/K | Q 331,294 394×48 | `<ref>/components/sidebar/ChatSearchDialog.vue:196`–`:205` |
| OV-24 | 未找到聊天 | 文本 | — | OV-20 ∧ 无结果 | Q/S/K | Q 299,350 432×74 | `<ref>/components/sidebar/ChatSearchDialog.vue:255`–`:261` |
| OV-25 | 加载中 | 状态行 | Loader2 | OV-20 ∧ 结果分页加载中 | — | — | `<ref>/components/sidebar/ChatSearchDialog.vue:209` |
| OV-26 | 上一页 / 显示更多 | 文字按钮（2 条） | — | OV-20 ∧ 各自分页条件 | — | — | `<ref>/components/sidebar/ChatSearchDialog.vue:210`–`:211` |
| OV-27 | 置顶聊天 / 最近聊天 | 分组标题 | — | OV-20 ∧ 有结果 | — | — | `<ref>/components/sidebar/ChatSearchDialog.vue:105`–`:112`、`:217`–`:220` |
| OV-28 | `{会话标题}` + `{工作区名}` + `{快捷键}` | 结果行 | 状态图标（错误/加载/未读/置顶/普通） | OV-20 ∧ 每条结果 | — | — | `<ref>/components/sidebar/ChatSearchDialog.vue:221`–`:252` |

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
| PG-10 | `{空列表标题}` / `{无匹配标题}` + 说明 | 空态 | — | 路由 = 定时任务 ∧ 调度器可用 ∧ 列表为空/无匹配 | — | — | `<ref>/views/ScheduledView.vue:183`–`:187` |
| PG-11 | （任务行） | 列表行 | — | 路由 = 定时任务 ∧ 每条任务 | — | — | `<ref>/components/scheduled/ScheduledJobList.vue` |

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
| PG-29 | （技能卡 / 搜索技能 / 空态 / 详情面板） | 列表与面板 | — | 路由 = 技能 ∧ 商店或我的技能有内容 | — | — | `<ref>/components/skills/SkillCard.vue`、`SkillDetailPanel.vue`、`SkillsMarketplacePanel.vue` |

元素数：18

**区 7 元素数：57**

---

## 8. 合计与差异

### 8.1 计数

| 项 | 数 |
|---|---|
| 标题栏（区 1） | 40 |
| 左栏（区 2） | 106 |
| 时间线（区 3） | 69 |
| 输入区（区 4） | 65 |
| 状态条（区 5） | 21 |
| 空态（第 6 节） | 15 |
| 运行时出现、不属五区（第 7 节） | 57 |
| **模板节点总数（元素级行数）** | **373** |
| 其中五区合计（区 1–5） | 301 |
| 运行时命中（屏上可见，含截图补位） | 77 |
| 运行时命中（DOM 在、屏上不可见：占位 2、hover 动作 2、悬停箭头 1） | 5 |
| 未命中（六态均无） | 291 |
| 运行时独有（无模板声明） | 0 |

口径：
- 「模板节点」按**元素级**计（可交互控件、文案节点、状态位、容器状态），纯布局 `div` 不计。
- 「未命中」= 六个采集状态下都没有出现；原因分布见 §8.2。
- 命中判定优先用截图（JSON 会漏采无文案图标与 svg，见 §0.4）。
- **运行时独有 = 0**：六态里出现的每个节点都能在模板找到声明。唯一「看起来独有」的是右栏 `+ 添加页签` 与 `筛选文件...`——它们模板有、但采集漏采（无 `aria-label`/被 rules 跳过），见 §8.4。

### 8.2 未命中原因分布（291 条）

按**主因**归类，一条只计一次；顺序 = 先判「是否只有会话态渲染」，再判「是否必须交互」，最后判数据/平台/模式。

| 原因 | 说明 | 区 1 | 区 2 | 区 3 | 区 4 | 区 5 | 区 6 | 区 7 | 合计 |
|---|---|---|---|---|---|---|---|---|---|
| 无会话 | 只有会话态才渲染 | 36 | 18 | 68 | 26 | 21 | 0 | 0 | **169** |
| 需交互 | 悬停 / 点击 / 快捷键才出现（菜单、浮层、tooltip、hover 动作、拖入提示） | 0 | 33 | 0 | 20 | 0 | 7 | 7 | **67** |
| 数据 / 平台 / 模式 | 更新态、置顶数据、分页、运行中标记、非 macOS、办公模式、平铺视图、弹窗窗口、diff 页等 | 0 | 31 | 0 | 11 | 0 | 2 | 11 | **55** |
| 合计 | | 36 | 82 | 68 | 57 | 21 | 9 | 18 | **291** |

### 8.3 只在某个状态出现的元素

**只在 E/N（左栏展开的空会话）——25 个**

- `TB-03`（打开上下文面板）
- `LB-01`（交通灯占位）、`LB-10` 至 `LB-15`（不含 `LB-13`，其快捷键徽标需悬停；即收起侧边栏 + 4 个全局入口）、`LB-24` 至 `LB-26`（会话标签 + 添加项目 + 视图切换）、`LB-34` 至 `LB-39`（工作区行 + 图标 + 名称）、`LB-42`、`LB-43`（行内新建/菜单，屏上不可见）、`LB-50`（暂无会话）、`LB-57`（内置工作区分隔线）、`LB-58`（列表底部「显示更多」）、`LB-83`、`LB-84`、`LB-88`、`LB-100`（账户行 + 首字母 + 名称 + 设置齿轮）
- 说明：E 与 N 的 `items` 逐节点相同，无法从采集区分「空」与「新建会话中」；右栏一开（R）这些元素要么消失（左栏收起），要么换位。

**只在空会话态（E/N/R 三态，与左栏开合无关）——15 个**

- `TL-01`（时间线容器）、`NT-01`、`NT-02`、`NT-04` 至 `NT-06`、`NT-14`（品牌图形、开始构建、输入块、控制行、工作区选择器、权限选择器；`NT-03` 需初次选择进行中）、`IN-01`、`IN-02`、`IN-05` 至 `IN-09`、`IN-12`（输入块、占位、附件、模式、技能、推理强度、模型、发送）
- 说明：会话态这些位置被 `ChatHeader` + `PlainTimeline` + 底部栈取代。

**只在 R/Q（左栏收起 + 右栏打开）——11 个**

- `TB-01`、`TB-02`、`TB-04`（交通灯占位、展开侧边栏、关闭上下文面板）
- `OV-01`、`OV-02`、`OV-03`、`OV-06`、`OV-08`、`OV-11`、`OV-12`、`OV-13`（右栏外壳、宽度拖拽、文件页签、添加页签、展开面板、筛选文件、目录行、文件行）

**跨 Q/S/K 三态（同一弹窗叠在不同底色上）——5 个**

- `OV-20` 至 `OV-24`（搜索聊天弹窗、标题、描述、输入框、未找到聊天）
- 说明：`search.json` 是其中唯一「只有弹窗」的采集；S/K 两态里弹窗叠在路由页上，不能据此判断底下的五区元素是否存在。

**只在 S（定时任务页）——9 个**

- `PG-01` 至 `PG-09`（展开侧边栏、筛选任务三键、创建、搜索定时任务、不可用标题与说明）
- 说明：该路由整窗替换主窗口，左栏收起，五区元素全部不出现。

**只在 K（技能 / 插件页）——17 个**

- `PG-12` 至 `PG-28`（技能/插件分段、创建技能、更多操作、技能商店/我的技能、从来源安装整组、已安装来源整组）

**六态均无（会话态或条件未满足）——291 个**

- 见 §8.2；其中区 3（`TL-02` 起）、区 5（`SB-01` 起）、区 1 的会话变体（`TB-05` 至 `TB-40`）、`IN-27` 至 `IN-52`（底部栈）是「只在会话态出现」的整块。

### 8.4 三个需要注意的差异

1. **右栏 `+ 添加页签` 没有可访问名**：`RightPanelBar.vue:381`–`:397` 的按钮既无 `aria-label` 也无 `title`，采集脚本因此跳过它（JSON 缺失、截图可见）。同类还有输入区 `+ 附件`（`InputArea.vue:2460`，仅有 tooltip）与右栏 `筛选文件...`（`ContextPanel.vue:547`，只有 placeholder）。
2. **隐藏标签被当成可见文本采集**：`Agent`（463,456）、`技能`（491,455）是展开后才显示文字的 `span`（`opacity-0`），在 E/N/R 三态都出现在 JSON 里；对账时必须按「不可见」处理。
3. **S / K 两态不是主窗口**：采集里它们把整窗换成了路由页，且左栏收起；任何「主窗口元素在 S/K 缺失」的判断都不成立。

---

## 9. 本轮自检

- 六个状态的 JSON 全部比对：E/N/R/S/K/Q；E 与 N 的 `items` 已确认完全相等，故合并。
- 五区逐区对齐模板：标题栏（`NewThreadHeader` + `ChatHeaderFrame`/`ChatHeader`）、左栏（`AppSidebar` + 6 个子组件）、时间线（`PlainTimeline` + `TimelineRow` 分支）、输入区（`InputArea` + `ConversationComposer` + 底部栈）、状态条（`StatusBar`）。
- 空态（`NewThread`）单列，含 6 张办公模式推荐卡与两处下拉（工作区 / 分支）。
- 运行时可见但不属五区的三块（右栏、搜索弹窗、定时任务页、技能页）单列在第 7 节，未与五区重复计数。
