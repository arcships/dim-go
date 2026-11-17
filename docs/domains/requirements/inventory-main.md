# 主窗口元素清单

范围：主窗口五区（标题栏 / 左栏 / 时间线 / 输入区 / 状态条）的元素穷举。本文件只列元素，不写行为需求、不写验收。

- 参考产品仓库根记作 `<ref>`；定位写作 `<ref>/<相对路径>:<行号>`，相对 `packages/desktop/packages/renderer/src`。
- 文案原文取自参考产品 `zh-CN` 语言包；`{name}` 形式为插值占位。
- 列义：**控件** = 元素类型；**默认** = 初始可见性（显示 / 隐藏 / — 常显）；**条件** = 可见或可用的判定条件；**定位** = 模板出处。
- 计数约定：同一元素的**不同文案**各占一行（如「显示更多」与「显示更少」）；同一文案在多处渲染只占一行并在定位列出多处。
- 本批交付：主窗口五区全部（左栏、输入区、标题栏、时间线、状态条）。每区末尾附「本轮自检」逐项记录已核状态/菜单/空态。

---

# 屏：主窗口 → 区域：左栏

## A. 栏头（SidebarHeader）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| LB-HDR-01 | （品牌图标，无文案） | 图标 | 显示 | 非 macOS | `<ref>/components/sidebar/SidebarHeader.vue:104` |
| LB-HDR-02 | `DESKTOP_PRODUCT_NAME`（分发配置的桌面产品名） | 文本 | 显示 | 非 macOS | `<ref>/components/sidebar/SidebarHeader.vue:107` |
| LB-HDR-03 | （macOS 交通灯预留位，无文案） | 占位 | 显示 | macOS 且非全屏 | `<ref>/components/sidebar/SidebarHeader.vue:92` |
| LB-HDR-04 | 更新 | 按钮 | 隐藏 | 更新态 `available` 且无错误消息 | `<ref>/components/sidebar/SidebarHeader.vue:58`、`:114` |
| LB-HDR-05 | `{downloadProgressPercent}%` | 按钮 | 隐藏 | 更新态 `downloading` | `<ref>/components/sidebar/SidebarHeader.vue:55` |
| LB-HDR-06 | 正在准备… | 按钮 | 隐藏 | 更新态 `preparing` | `<ref>/components/sidebar/SidebarHeader.vue:56` |
| LB-HDR-07 | 重启安装 | 按钮 | 隐藏 | 更新态 `downloaded` | `<ref>/components/sidebar/SidebarHeader.vue:57` |
| LB-HDR-08 | 重试更新 | 按钮（危险色） | 隐藏 | 更新态 `available` 且带错误消息 | `<ref>/components/sidebar/SidebarHeader.vue:58`、`:121` |
| LB-HDR-09 | `{状态} · v{版本} · {错误消息}` | 提示浮层 | 隐藏 | 悬停更新按钮 | `<ref>/components/sidebar/SidebarHeader.vue:60`、`:136` |
| LB-HDR-10 | 收起侧边栏 ⌘B | 图标按钮 | 显示 | 展开态且允许收起 | `<ref>/components/sidebar/SidebarHeader.vue:146`、`:162` |

元素数：10

## B. 全局入口（GlobalEntries）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| LB-GLB-01 | 新建会话 | 菜单行（图标 + 文本） | 显示 | 禁用态由 `disableNewThread` 决定 | `<ref>/components/sidebar/GlobalEntries.vue:66` |
| LB-GLB-02 | 搜索 | 菜单行（图标 + 文本） | 显示 | — | `<ref>/components/sidebar/GlobalEntries.vue:80` |
| LB-GLB-03 | ⌘G（Windows/Linux：Ctrl+G） | 快捷键徽标 | 隐藏 | 悬停或聚焦该行 | `<ref>/components/sidebar/GlobalEntries.vue:88` |
| LB-GLB-04 | 定时任务 | 菜单行（时钟图标 + 文本） | 显示 | 当前路由为定时任务时高亮 | `<ref>/components/sidebar/GlobalEntries.vue:96` |
| LB-GLB-05 | 技能/插件 | 菜单行（宫格图标 + 文本） | 显示 | 当前路由为技能时高亮 | `<ref>/components/sidebar/GlobalEntries.vue:110` |

元素数：5

## C. 置顶区（PinnedThreads）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| LB-PIN-01 | 已置顶 | 分组标题 | 显示 | 存在置顶内容 | `<ref>/components/sidebar/PinnedThreads.vue:191` |
| LB-PIN-02 | （置顶会话行，复用会话行元素组 F） | 行 | 显示 | 每条置顶会话 | `<ref>/components/sidebar/PinnedThreads.vue:213` |
| LB-PIN-03 | （会话拖拽落点指示线，无文案） | 指示器 | 隐藏 | 拖拽悬停于列表首/尾或某项半区 | `<ref>/components/sidebar/PinnedThreads.vue:197`、`:225` |
| LB-PIN-04 | 上一页 | 文字按钮 | 隐藏 | 存在更早的置顶会话页 | `<ref>/components/sidebar/PinnedThreads.vue:239` |
| LB-PIN-05 | 显示更多 | 文字按钮 | 隐藏 | 存在更多置顶会话页 | `<ref>/components/sidebar/PinnedThreads.vue:240` |
| LB-PIN-06 | （置顶工作区分组，复用工作区分组元素组 E） | 分组 | 显示 | 存在置顶工作区 | `<ref>/components/sidebar/PinnedThreads.vue:242`、`:263` |
| LB-PIN-07 | （整区拖入底色态，无文案） | 容器态 | — | 正在会话拖拽到置顶区 | `<ref>/components/sidebar/PinnedThreads.vue:182` |

元素数：7

## D. 会话列表头与列表外壳（ThreadListHeader / ThreadList）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| LB-LST-01 | 会话 | 分组标题 | 显示 | — | `<ref>/components/sidebar/ThreadListHeader.vue:94` |
| LB-LST-02 | （添加工作区，文件夹加号图标，无文案） | 图标按钮 | 显示 | 选择目录进行中时禁用 | `<ref>/components/sidebar/ThreadListHeader.vue:98` |
| LB-LST-03 | （视图/排序菜单触发，图标随视图切换：分组=列表图标，平铺=宫格图标） | 图标按钮 | 显示 | 办公模式隐藏视图切换器 | `<ref>/components/sidebar/ThreadListHeader.vue:109`、`:117` |
| LB-LST-04 | 排序 | 菜单分组标题 | 显示 | 同上菜单 | `<ref>/components/sidebar/ThreadListHeader.vue:127` |
| LB-LST-05 | 最近活跃 | 菜单项（可勾选） | 显示 | 当前排序为最近活跃时显示勾 | `<ref>/components/sidebar/ThreadListHeader.vue:130` |
| LB-LST-06 | 最近创建 | 菜单项（可勾选） | 显示 | 当前排序为最近创建时显示勾 | `<ref>/components/sidebar/ThreadListHeader.vue:138` |
| LB-LST-07 | （分隔线，无文案） | 分隔线 | 显示 | — | `<ref>/components/sidebar/ThreadListHeader.vue:147` |
| LB-LST-08 | 视图 | 菜单分组标题 | 显示 | — | `<ref>/components/sidebar/ThreadListHeader.vue:149` |
| LB-LST-09 | 按工作区分组 | 菜单项（可勾选） | 显示 | 当前为分组视图时显示勾 | `<ref>/components/sidebar/ThreadListHeader.vue:152` |
| LB-LST-10 | 平铺 | 菜单项（可勾选） | 显示 | 当前为平铺视图时显示勾 | `<ref>/components/sidebar/ThreadListHeader.vue:160` |
| LB-LST-11 | （分隔线，无文案） | 分隔线 | 显示 | 仅分组视图 | `<ref>/components/sidebar/ThreadListHeader.vue:172` |
| LB-LST-12 | 折叠全部 | 菜单项（含图标） | 显示 | 仅分组视图 | `<ref>/components/sidebar/ThreadListHeader.vue:174` |
| LB-LST-13 | 上一页 | 文字按钮 | 隐藏 | 存在上一页工作区 | `<ref>/components/sidebar/ThreadList.vue:239` |
| LB-LST-14 | 显示更多 | 文字按钮 | 隐藏 | 存在更多工作区 | `<ref>/components/sidebar/ThreadList.vue:240` |
| LB-LST-15 | 正在加载... | 行内状态（含旋转图标） | 隐藏 | 当前页会话请求进行中 | `<ref>/components/sidebar/ThreadList.vue:246` |
| LB-LST-16 | 今天 / 昨天 / 更早 / `{日期}` | 日期分组标题 | 显示 | 仅平铺视图，按会话时间归档 | `<ref>/components/sidebar/ThreadList.vue:250` |
| LB-LST-17 | 上一页 | 文字按钮 | 隐藏 | 仅平铺视图且存在上一页 | `<ref>/components/sidebar/ThreadList.vue:253` |
| LB-LST-18 | 显示更多 | 文字按钮（附右箭头） | 显示 | 平铺视图存在更多会话 | `<ref>/components/sidebar/ThreadList.vue:254`、`:266` |
| LB-LST-19 | 显示更少 | 文字按钮（附下箭头） | 隐藏 | 已展开超过首批且无更多 | `<ref>/components/sidebar/ThreadList.vue:261` |
| LB-LST-20 | （列表空白区点击清空多选，无文案） | 容器交互 | — | — | `<ref>/components/sidebar/ThreadList.vue:167`、`:173` |

元素数：20

## E. 工作区分组（ProjectItem）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| LB-PRJ-01 | （工作区行，整行可点击展开/收起） | 行 | 显示 | — | `<ref>/components/sidebar/ProjectItem.vue:282` |
| LB-PRJ-02 | （展开箭头，展开时旋转 90°） | 图标 | 隐藏 | 悬停该行 | `<ref>/components/sidebar/ProjectItem.vue:313` |
| LB-PRJ-03 | （工作区图标：内置工作区=🏠） | 图标 | 显示 | 内置工作区且未悬停 | `<ref>/components/sidebar/ProjectItem.vue:323` |
| LB-PRJ-04 | （工作区图标：展开态=打开文件夹） | 图标 | 显示 | 普通工作区、已展开、未悬停 | `<ref>/components/sidebar/ProjectItem.vue:328` |
| LB-PRJ-05 | （工作区图标：收起态=文件夹） | 图标 | 显示 | 普通工作区、已收起、未悬停 | `<ref>/components/sidebar/ProjectItem.vue:332` |
| LB-PRJ-06 | `{工作区名称}` | 文本（超长截断） | 显示 | — | `<ref>/components/sidebar/ProjectItem.vue:338`、`:341` |
| LB-PRJ-07 | `{工作区绝对路径}` | 提示浮层 | 隐藏 | 悬停名称且有路径 | `<ref>/components/sidebar/ProjectItem.vue:433` |
| LB-PRJ-08 | （折叠态运行中指示，旋转图标，无文案） | 图标 | 隐藏 | 已收起且有线程运行、未悬停 | `<ref>/components/sidebar/ProjectItem.vue:349` |
| LB-PRJ-09 | （行内新建会话，图标按钮，无文案） | 图标按钮 | 隐藏 | 悬停或菜单打开 | `<ref>/components/sidebar/ProjectItem.vue:361` |
| LB-PRJ-10 | （行内更多菜单触发，⋯ 图标，无文案） | 图标按钮 | 隐藏 | 悬停或菜单打开 | `<ref>/components/sidebar/ProjectItem.vue:372` |
| LB-PRJ-11 | 置顶工作区 | 菜单项（含图钉图标） | 显示 | 非内置工作区且未置顶 | `<ref>/components/sidebar/ProjectItem.vue:390` |
| LB-PRJ-12 | 取消置顶工作区 | 菜单项（含图钉图标） | 显示 | 已置顶 | `<ref>/components/sidebar/ProjectItem.vue:396` |
| LB-PRJ-13 | 在文件管理器中显示 | 菜单项（含文件夹图标） | 显示 | 内置工作区或有路径 | `<ref>/components/sidebar/ProjectItem.vue:398` |
| LB-PRJ-14 | 归档会话 | 菜单项（含归档图标） | 显示 | 该工作区会话数 > 0 | `<ref>/components/sidebar/ProjectItem.vue:406` |
| LB-PRJ-15 | （分隔线，无文案） | 分隔线 | 显示 | — | `<ref>/components/sidebar/ProjectItem.vue:414` |
| LB-PRJ-16 | 重命名 | 菜单项（含铅笔图标） | 显示 | — | `<ref>/components/sidebar/ProjectItem.vue:415` |
| LB-PRJ-17 | 移除 | 菜单项（危险色，含垃圾桶图标） | 显示 | 非内置工作区 | `<ref>/components/sidebar/ProjectItem.vue:419` |
| LB-PRJ-18 | 正在加载... | 行内状态（含旋转图标） | 隐藏 | 该工作区会话分页请求中 | `<ref>/components/sidebar/ProjectItem.vue:455` |
| LB-PRJ-19 | 暂无会话 | 空态文本 | 隐藏 | 展开且该工作区无会话 | `<ref>/components/sidebar/ProjectItem.vue:457` |
| LB-PRJ-20 | （组内会话行，复用会话行元素组 F） | 行 | 显示 | 展开后每条会话 | `<ref>/components/sidebar/ProjectItem.vue:465` |
| LB-PRJ-21 | 上一页 | 文字按钮 | 隐藏 | 该工作区存在上一页 | `<ref>/components/sidebar/ProjectItem.vue:470` |
| LB-PRJ-22 | 显示更多 | 文字按钮（附右箭头） | 显示 | 会话数超阈值或还有更多页 | `<ref>/components/sidebar/ProjectItem.vue:473`、`:486` |
| LB-PRJ-23 | 显示更少 | 文字按钮（附下箭头） | 隐藏 | 已展开超过阈值且无更多 | `<ref>/components/sidebar/ProjectItem.vue:481` |
| LB-PRJ-24 | （占位行：未进入视口时的轻量行，含工作区名） | 占位行 | 显示 | 该分组尚未接近视口 | `<ref>/components/sidebar/ProjectItem.vue:494` |
| LB-PRJ-25 | 暂无会话 | 空态文本（占位行内） | 隐藏 | 占位行处于展开态 | `<ref>/components/sidebar/ProjectItem.vue:503` |
| LB-PRJ-26 | （内置工作区与普通工作区之间的分隔线，无文案） | 分隔线 | 显示 | 内置工作区分组之后 | `<ref>/components/sidebar/ProjectItem.vue:511` |
| LB-PRJ-27 | （拖拽中组内容禁用态：半透明 + 灰度，无文案） | 容器态 | — | 该工作区正在被拖动 | `<ref>/components/sidebar/ProjectItem.vue:451` |

元素数：27

## F. 会话行（ThreadItem）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| LB-THR-01 | （会话行容器，选中/激活态高亮） | 行 | 显示 | — | `<ref>/components/sidebar/ThreadItem.vue:234` |
| LB-THR-02 | （错误状态图标，无文案） | 图标 | 隐藏 | 会话出错 | `<ref>/components/sidebar/ThreadItem.vue:259` |
| LB-THR-03 | （运行中状态图标，旋转，无文案） | 图标 | 隐藏 | 运行中且未悬停 | `<ref>/components/sidebar/ThreadItem.vue:263` |
| LB-THR-04 | （悬停置顶切换按钮，图钉图标，无文案） | 图标按钮 | 隐藏 | 悬停且非置顶区 | `<ref>/components/sidebar/ThreadItem.vue:268` |
| LB-THR-05 | （置顶标记，星形图标，无文案） | 图标 | 隐藏 | 已置顶且普通列表 | `<ref>/components/sidebar/ThreadItem.vue:294` |
| LB-THR-06 | （未读圆点，无文案） | 标记 | 隐藏 | 未读 | `<ref>/components/sidebar/ThreadItem.vue:290` |
| LB-THR-07 | （置顶区常显图钉，无文案） | 图标 | 显示 | 置顶区行且未悬停 | `<ref>/components/sidebar/ThreadItem.vue:285` |
| LB-THR-08 | `{会话标题}` | 文本（溢出滚动） | 显示 | — | `<ref>/components/sidebar/ThreadItem.vue:301` |
| LB-THR-09 | 等待操作 | 标签 | 隐藏 | 会话有待处理交互 | `<ref>/components/sidebar/ThreadItem.vue:308` |
| LB-THR-10 | `+{新增} -{删除} ~{修改}` | 文本 | 隐藏 | 有改动摘要且办公模式显示 git 面 | `<ref>/components/sidebar/ThreadItem.vue:316` |
| LB-THR-11 | ⌘1 / Ctrl+1 … | 快捷键徽标 | 隐藏 | 快捷键提示模式开启 | `<ref>/components/sidebar/ThreadItem.vue:327` |
| LB-THR-12 | `{相对时间}` | 文本 | 显示 | 非定时任务来源且未悬停 | `<ref>/components/sidebar/ThreadItem.vue:345` |
| LB-THR-13 | （定时任务来源图标，无文案） | 图标 | 显示 | 会话来源为定时任务且未悬停 | `<ref>/components/sidebar/ThreadItem.vue:339` |
| LB-THR-14 | （悬停归档按钮，归档图标，可访问名「归档会话」） | 图标按钮 | 隐藏 | 悬停该行 | `<ref>/components/sidebar/ThreadItem.vue:354` |
| LB-THR-15 | `{会话标题}` | 悬停卡标题 | 隐藏 | 悬停该行 | `<ref>/components/sidebar/ThreadItem.vue:380` |
| LB-THR-16 | `{所属工作区名}` | 悬停卡行（图标 + 文本） | 隐藏 | 悬停该行 | `<ref>/components/sidebar/ThreadItem.vue:388` |
| LB-THR-17 | `{Git 分支}` | 悬停卡行（分支图标 + 文本） | 隐藏 | 有分支且办公模式显示 git 面 | `<ref>/components/sidebar/ThreadItem.vue:396` |
| LB-THR-18 | 当前分支是 `{currentBranch}`；发送消息后会更新此会话的分支。 | 悬停卡警告 | 隐藏 | 会话分支与当前工作区分支不一致 | `<ref>/components/sidebar/ThreadItem.vue:402` |

元素数：18

## G. 会话右键菜单（ThreadContextMenu）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| LB-CTX-01 | 置顶 | 菜单项 | 显示 | 单选且未置顶 | `<ref>/components/sidebar/ThreadContextMenu.vue:104`、`:109` |
| LB-CTX-02 | 取消置顶 | 菜单项 | 显示 | 单选且已置顶 | `<ref>/components/sidebar/ThreadContextMenu.vue:109` |
| LB-CTX-03 | 置顶 ({count}) | 菜单项 | 显示 | 多选且均未置顶 | `<ref>/components/sidebar/ThreadContextMenu.vue:106` |
| LB-CTX-04 | 取消置顶 ({count}) | 菜单项 | 显示 | 多选且均已置顶 | `<ref>/components/sidebar/ThreadContextMenu.vue:106` |
| LB-CTX-05 | 重命名 | 菜单项 | 显示 | 重命名能力就绪 | `<ref>/components/sidebar/ThreadContextMenu.vue:113`、`:47` |
| LB-CTX-06 | 重命名（{reason}） | 菜单项（禁用） | 显示 | 重命名能力检查中/不可用 | `<ref>/components/sidebar/ThreadContextMenu.vue:49` |
| LB-CTX-07 | 在新窗口中打开 | 菜单项 | 显示 | 单选 | `<ref>/components/sidebar/ThreadContextMenu.vue:124` |
| LB-CTX-08 | 导出 | 菜单项 | 显示 | 单选 | `<ref>/components/sidebar/ThreadContextMenu.vue:132` |
| LB-CTX-09 | （分隔线，无文案） | 分隔线 | 显示 | — | `<ref>/components/sidebar/ThreadContextMenu.vue:140` |
| LB-CTX-10 | 归档 | 菜单项 | 显示 | 单选 | `<ref>/components/sidebar/ThreadContextMenu.vue:142`、`:150` |
| LB-CTX-11 | 归档 ({count}) | 菜单项 | 显示 | 多选 | `<ref>/components/sidebar/ThreadContextMenu.vue:147` |
| LB-CTX-12 | 删除 | 菜单项（危险色） | 显示 | 单选 | `<ref>/components/sidebar/ThreadContextMenu.vue:154`、`:163` |
| LB-CTX-13 | 删除 ({count}) | 菜单项（危险色） | 显示 | 多选 | `<ref>/components/sidebar/ThreadContextMenu.vue:160` |

元素数：13

## H. 账户入口（AccountEntry）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| LB-ACC-01 | （账户入口按钮，整行） | 按钮（触发菜单） | 显示 | 未登录时点击直接进入设置·账户 | `<ref>/components/sidebar/AccountEntry.vue:209` |
| LB-ACC-02 | `{首字母}` | 头像圆标 | 显示 | 已登录 | `<ref>/components/sidebar/AccountEntry.vue:217`、`:220` |
| LB-ACC-03 | （登录中旋转图标，无文案） | 图标 | 隐藏 | 正在登录 | `<ref>/components/sidebar/AccountEntry.vue:218` |
| LB-ACC-04 | （未登录用户图标，无文案） | 图标 | 隐藏 | 未登录 | `<ref>/components/sidebar/AccountEntry.vue:219` |
| LB-ACC-05 | （错误红点，无文案） | 标记 | 隐藏 | 有错误消息且不在登录中 | `<ref>/components/sidebar/AccountEntry.vue:221` |
| LB-ACC-06 | 正在加载... | 主文案 | 隐藏 | 账户状态尚未确认 | `<ref>/components/sidebar/AccountEntry.vue:71`、`:227` |
| LB-ACC-07 | 正在登录… | 主文案 | 隐藏 | 未登录且登录进行中 | `<ref>/components/sidebar/AccountEntry.vue:76`、`:227` |
| LB-ACC-08 | 登录 Dim | 主文案 | 隐藏 | 未登录 | `<ref>/components/sidebar/AccountEntry.vue:77`、`:227` |
| LB-ACC-09 | `{显示名} / {邮箱} / Dim 账户` | 主文案 | 显示 | 已登录（依次回退） | `<ref>/components/sidebar/AccountEntry.vue:78`、`:227` |
| LB-ACC-10 | （设置快捷按钮，齿轮图标，可访问名「设置」） | 图标按钮 | 显示 | 设置路由时高亮 | `<ref>/components/sidebar/AccountEntry.vue:362` |
| LB-ACC-11 | `{首字母}` + `{显示名}` | 菜单身份行 | 显示 | 菜单打开 | `<ref>/components/sidebar/AccountEntry.vue:241` |
| LB-ACC-12 | （分隔线，无文案） | 分隔线 | 显示 | 菜单打开 | `<ref>/components/sidebar/AccountEntry.vue:253` |
| LB-ACC-13 | 用量 | 菜单项（仪表图标） | 显示 | 已登录 | `<ref>/components/sidebar/AccountEntry.vue:256`、`:264` |
| LB-ACC-14 | 剩余 {pct}% | 菜单项右侧数值 | 显示 | 已登录且已取得剩余额度 | `<ref>/components/sidebar/AccountEntry.vue:266` |
| LB-ACC-15 | （用量加载中旋转图标，无文案） | 图标 | 隐藏 | 用量请求进行中 | `<ref>/components/sidebar/AccountEntry.vue:265` |
| LB-ACC-16 | 收起 Pet | 菜单项（笑脸图标） | 显示 | 已登录且 Pet 显示中 | `<ref>/components/sidebar/AccountEntry.vue:281` |
| LB-ACC-17 | 显示 Pet | 菜单项（笑脸图标） | 显示 | 已登录且 Pet 收起 | `<ref>/components/sidebar/AccountEntry.vue:281` |
| LB-ACC-18 | 正在加载... | 菜单项文本 | 隐藏 | 已登录且 Pet 快照未就绪 | `<ref>/components/sidebar/AccountEntry.vue:281` |
| LB-ACC-19 | `{Pet 不可用原因}` | 说明文本 | 隐藏 | Pet 悬浮能力不可用 | `<ref>/components/sidebar/AccountEntry.vue:289` |
| LB-ACC-20 | 邀请朋友 | 菜单项（纸飞机图标） | 隐藏 | 已登录且推广资格就绪 | `<ref>/components/sidebar/AccountEntry.vue:297` |
| LB-ACC-21 | 正在加载... | 菜单项文本（禁用） | 隐藏 | 推广数据请求中 | `<ref>/components/sidebar/AccountEntry.vue:309`、`:318` |
| LB-ACC-22 | 设置 | 菜单项（齿轮图标） | 显示 | 已登录 | `<ref>/components/sidebar/AccountEntry.vue:321`、`:330` |
| LB-ACC-23 | ⌘, / Ctrl+, | 菜单项右侧快捷键 | 显示 | 已登录 | `<ref>/components/sidebar/AccountEntry.vue:331` |
| LB-ACC-24 | 退出登录 | 菜单项（登出图标） | 显示 | 已登录 | `<ref>/components/sidebar/AccountEntry.vue:336` |
| LB-ACC-25 | 登录 Dim | 菜单项（用户图标） | 显示 | 未登录 | `<ref>/components/sidebar/AccountEntry.vue:348`、`:357` |
| LB-ACC-26 | 退出 Dim？ | 确认框标题 | 隐藏 | 点击退出登录后 | `<ref>/components/sidebar/AccountEntry.vue:378` |
| LB-ACC-27 | 退出后，需要重新登录才能使用 Dim 账户。 | 确认框说明 | 隐藏 | 同上 | `<ref>/components/sidebar/AccountEntry.vue:380` |
| LB-ACC-28 | 取消 | 确认框按钮 | 隐藏 | 同上 | `<ref>/components/sidebar/AccountEntry.vue:385` |
| LB-ACC-29 | 退出登录 | 确认框主按钮（危险色） | 隐藏 | 同上，退出进行中时禁用 | `<ref>/components/sidebar/AccountEntry.vue:388` |

元素数：29

## I. 拖入与拖拽指示（AppSidebar / ProjectItem）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| LB-DRP-01 | （拖入提示图标，文件夹加号，无文案） | 图标 | 隐藏 | Finder 文件夹拖入左栏 | `<ref>/components/sidebar/AppSidebar.vue:118` |
| LB-DRP-02 | 松开以添加工作区 | 提示文本 | 隐藏 | 同上 | `<ref>/components/sidebar/AppSidebar.vue:119` |
| LB-DRP-03 | 仅限文件夹 · 同时新建会话 | 提示副文本 | 隐藏 | 同上 | `<ref>/components/sidebar/AppSidebar.vue:120` |
| LB-DRP-04 | （工作区排序落点指示线，无文案） | 指示器 | 隐藏 | 拖动工作区到某一排序位 | `<ref>/components/sidebar/ThreadList.vue:190`、`:215`；`<ref>/components/sidebar/PinnedThreads.vue:255`、`:281` |
| LB-DRP-05 | （新增工作区落位动画，无文案） | 行内动画态 | — | 拖入后新增的工作区首次渲染 | `<ref>/components/sidebar/ProjectItem.vue:288`、`:109` |

元素数：5

**左栏小计：10 + 5 + 7 + 20 + 27 + 18 + 13 + 29 + 5 = 134**

> 本轮自检（左栏）：逐项过了 ①栏头：更新按钮五态、无更新时隐藏、收起按钮的展开/收起与悬浮态；②全局入口 4 项 + 快捷键徽标两平台文案；③置顶区：有/无置顶内容、分页两按钮、拖入高亮；④列表头：添加工作区、视图切换器（分组/平铺图标）、排序两项勾选、视图两项勾选、折叠全部仅分组视图、工作区与平铺两套分页按钮；⑤工作区：内置/普通、展开/收起、悬停控件、折叠运行中指示、菜单五项（置顶两态/显示/归档/重命名/移除）、空态、加载态、占位行、拖拽禁用态；⑥会话行：错误/运行/未读/置顶四类状态图标、悬停置顶与悬停归档、等待操作标签、改动摘要、快捷键徽标、定时来源、悬停卡四段；⑦右键菜单单选/多选两套文案；⑧账户入口：未登录/登录中/已登录/状态未定四态、菜单已登录七项与未登录一项、Pet 三态、退出确认四件；⑨拖入提示两行、排序与新增两种指示；⑩附录对话框：搜索三态（加载/结果/空）、归档确认、重命名校验三态。

## 附录 A-1. 左栏触发的对话框（跨区，与 `inventory-dialogs` 互为索引）

> 本节与独立的对话框清单互为索引：去重时只删重复正文，保留指向。

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| LB-DLG-01 | 搜索聊天 / 搜索并打开已有聊天 | 对话框标题与说明（视觉隐藏） | 隐藏 | 打开搜索对话框 | `<ref>/components/sidebar/ChatSearchDialog.vue:191` |
| LB-DLG-02 | 搜索聊天 | 输入框（占位） | 显示 | 同上 | `<ref>/components/sidebar/ChatSearchDialog.vue:198` |
| LB-DLG-03 | 置顶聊天 / 最近聊天 | 结果分组标题 | 显示 | 该组有结果 | `<ref>/components/sidebar/ChatSearchDialog.vue:218` |
| LB-DLG-04 | `{会话标题}` / `{工作区名}` / `{⌘}{序号}` | 结果行 | 显示 | 有匹配结果（最多 8 条） | `<ref>/components/sidebar/ChatSearchDialog.vue:221`、`:244`、`:247` |
| LB-DLG-05 | 未找到聊天 | 空态文本 | 隐藏 | 无匹配结果 | `<ref>/components/sidebar/ChatSearchDialog.vue:254` |
| LB-DLG-06 | 正在加载... | 行内状态 | 隐藏 | 搜索请求进行中 | `<ref>/components/sidebar/ChatSearchDialog.vue:209` |
| LB-DLG-07 | 上一页 / 显示更多 | 文字按钮 | 隐藏 | 存在更多搜索结果页 | `<ref>/components/sidebar/ChatSearchDialog.vue:210`、`:211` |
| LB-DLG-08 | 归档 {count} 个会话 | 对话框标题 | 隐藏 | 触发归档确认 | `<ref>/components/sidebar/ArchiveThreadDialog.vue:66` |
| LB-DLG-09 | 确定要归档以下会话吗？{names} | 对话框说明 | 隐藏 | 同上 | `<ref>/components/sidebar/ArchiveThreadDialog.vue:69` |
| LB-DLG-10 | 正在运行的会话会被中断，其队列中的后续消息也会被丢弃。 | 警告文本 | 隐藏 | 含运行中的会话 | `<ref>/components/sidebar/ArchiveThreadDialog.vue:74` |
| LB-DLG-11 | `{错误消息}` | 错误文本 | 隐藏 | 归档失败 | `<ref>/components/sidebar/ArchiveThreadDialog.vue:82` |
| LB-DLG-12 | 取消 | 按钮 | 显示 | 同上 | `<ref>/components/sidebar/ArchiveThreadDialog.vue:85` |
| LB-DLG-13 | 归档 ({count}) / 归档中... | 主按钮 | 显示 | 同上，进行中切换文案 | `<ref>/components/sidebar/ArchiveThreadDialog.vue:88` |
| LB-DLG-14 | 重命名会话 / 输入新的会话名称 | 对话框标题与说明 | 隐藏 | 触发重命名 | `<ref>/components/sidebar/RenameThreadDialog.vue:125` |
| LB-DLG-15 | 会话名称 / 输入会话名称 | 标签 + 输入框 | 显示 | 同上 | `<ref>/components/sidebar/RenameThreadDialog.vue:133`、`:134` |
| LB-DLG-16 | `{错误 / 就绪 / 校验提示}` | 行内提示 | 隐藏 | 有错误、就绪状态或校验不通过 | `<ref>/components/sidebar/RenameThreadDialog.vue:150`、`:159`、`:167` |
| LB-DLG-17 | 取消 / 确认 | 按钮 | 显示 | 同上，不可提交时确认禁用 | `<ref>/components/sidebar/RenameThreadDialog.vue:180`、`:188` |

元素数：17（附录，不计入左栏主区小计）

---

# 屏：主窗口 → 区域：输入区

## A. 顶部提示与 Shell 运行面板（InputArea / ShellRunPanel）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| CP-TOP-01 | 会话忙碌或正在同步，暂时无法发送。草稿保留在此窗口。 | 提示文本 | 隐藏 | 既不能发送也不能排队且状态已定 | `<ref>/components/chat-workspace/composer/InputArea.vue:2309` |
| CP-TOP-02 | 重试 | 行内文字按钮 | 隐藏 | 提示出现且发送能力未取到 | `<ref>/components/chat-workspace/composer/InputArea.vue:2311` |
| CP-TOP-03 | （Shell 运行面板容器，无文案） | 面板 | 隐藏 | 当前会话有活动的 shell 运行 | `<ref>/components/chat-workspace/composer/InputArea.vue:2313` |
| CP-TOP-04 | `{命令}` | 等宽文本 | 显示 | 面板存在 | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:24` |
| CP-TOP-05 | 记录到上下文 / 仅本地输出 | 模式徽标 | 显示 | 面板存在，按运行模式切换 | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:25` |
| CP-TOP-06 | 运行中 / 正在停止 / 已完成 / 失败 / 已停止 | 状态文本（含旋转图标） | 显示 | 面板存在，按运行状态切换 | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:28` |
| CP-TOP-07 | 停止 | 按钮（含方块图标） | 隐藏 | 运行中或停止中 | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:32` |
| CP-TOP-08 | 关闭 | 按钮（含叉号图标） | 隐藏 | 运行结束后 | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:44` |
| CP-TOP-09 | `{输出内容}` | 等宽预格式块 | 隐藏 | 有输出 | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:56` |
| CP-TOP-10 | 已跳过 {count} 字节 | 说明文本 | 隐藏 | 有跳过字节 | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:59` |
| CP-TOP-11 | 输出已截断。 | 说明文本 | 隐藏 | 输出被截断 | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:62` |
| CP-TOP-12 | [进程已退出，代码 {code}] | 说明文本 | 隐藏 | 有退出码 | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:65` |
| CP-TOP-13 | `{错误消息}` | 错误文本 | 隐藏 | 运行出错 | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:68` |
| CP-TOP-14 | `{原因}` | 说明文本 | 隐藏 | 无错误但有原因 | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:69` |
| CP-TOP-15 | 输出已写入此会话上下文。 | 说明文本 | 隐藏 | 记录模式且写入成功 | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:70` |
| CP-TOP-16 | `{记录失败消息}` | 错误文本 | 隐藏 | 记录模式且写入失败 | `<ref>/components/chat-workspace/composer/ShellRunPanel.vue:76` |

元素数：16

## B. 输入容器与附件片（InputArea / ConversationComposer / attachments）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| CP-BOX-01 | （输入框容器：圆角描边 + 半透明底） | 容器 | 显示 | 常规模式 | `<ref>/components/chat-workspace/composer/ConversationComposer.vue:60` |
| CP-BOX-02 | （紧凑输入容器：单行横向布局） | 容器 | 隐藏 | compact 覆盖层模式 | `<ref>/components/chat-workspace/composer/ConversationComposer.vue:62` |
| CP-BOX-03 | （拖拽附件高亮态：主色描边 + 光环） | 容器态 | — | 拖拽文件进入输入区 | `<ref>/components/chat-workspace/composer/ConversationComposer.vue:65` |
| CP-BOX-04 | 正在编辑队列消息 · 按 Esc 取消 | 提示文本 | 隐藏 | 正在编辑队列中某条消息 | `<ref>/components/chat-workspace/composer/InputArea.vue:2414` |
| CP-BOX-05 | （应用快照展开槽位，无文案） | 占位 | 隐藏 | 存在待展开的应用快照附件 | `<ref>/components/chat-workspace/composer/InputArea.vue:2421` |
| CP-BOX-06 | （图片附件轨，无文案） | 容器 | 隐藏 | 存在图片附件 | `<ref>/components/chat-workspace/composer/InputArea.vue:2427` |
| CP-BOX-07 | （图片卡：缩略图 + 名称） | 卡片 | 显示 | 每张图片附件 | `<ref>/components/chat-workspace/attachments/ImageAttachmentCard.vue:52`、`:63`、`:71` |
| CP-BOX-08 | 无法加载图片预览 / 图片过大，无法预览 / 不支持此图片格式 | 卡片错误文本 | 隐藏 | 缩略图加载失败 | `<ref>/components/chat-workspace/attachments/ImageAttachmentCard.vue:67` |
| CP-BOX-09 | 重试 | 行内文字按钮 | 隐藏 | 缩略图加载失败且错误非「不支持此格式」 | `<ref>/components/chat-workspace/attachments/ImageAttachmentCard.vue:69` |
| CP-BOX-10 | （图片删除按钮，叉号图标） | 图标按钮 | 隐藏 | 悬停图片卡 | `<ref>/components/chat-workspace/attachments/ImageAttachmentCard.vue:72` |
| CP-BOX-11 | （轨道右缘滚动提示，双箭头图标，无文案） | 遮罩指示 | 隐藏 | 轨道内容横向溢出 | `<ref>/components/chat-workspace/attachments/ImageAttachmentGallery.vue:103` |
| CP-BOX-12 | （图片预览对话框：标题 `{文件名}`） | 对话框 | 隐藏 | 点击图片卡 | `<ref>/components/chat-workspace/attachments/AttachmentImagePreviewDialog.vue:323` |
| CP-BOX-13 | 复制图片 | 对话框按钮 | 显示 | 预览打开 | `<ref>/components/chat-workspace/attachments/AttachmentImagePreviewDialog.vue:337` |
| CP-BOX-14 | 复制全部 ({count}) | 对话框按钮 | 显示 | 预览含多张图片 | `<ref>/components/chat-workspace/attachments/AttachmentImagePreviewDialog.vue:346` |
| CP-BOX-15 | 已复制到剪贴板 | 轻提示 | 隐藏 | 复制成功 | `<ref>/components/chat-workspace/attachments/AttachmentImagePreviewDialog.vue:68` |
| CP-BOX-16 | （附件片架容器，无文案） | 容器 | 隐藏 | 有上下文引用或非图片附件 | `<ref>/components/chat-workspace/composer/InputArea.vue:2433` |
| CP-BOX-17 | `{引用标签}` | 上下文引用片（来源图标 + 文本） | 显示 | 每条输入引用 | `<ref>/components/chat-workspace/composer/ContextReferenceList.vue:50` |
| CP-BOX-18 | （引用片关闭，叉号图标，可访问名「移除批注」） | 图标按钮 | 显示 | 每条输入引用 | `<ref>/components/chat-workspace/composer/ContextReferenceList.vue:56` |
| CP-BOX-19 | 选中文本：`{引用原文}` | 悬停卡字段 | 隐藏 | 悬停引用片且有选中文本 | `<ref>/components/chat-workspace/ContextReferenceHoverContent.vue:103`、`:108` |
| CP-BOX-20 | 用户评论 | 悬停卡字段标签 | 隐藏 | 悬停引用片且可编辑或有评论 | `<ref>/components/chat-workspace/ContextReferenceHoverContent.vue:120` |
| CP-BOX-21 | 编辑批注 | 悬停卡图标按钮（铅笔） | 显示 | 悬停卡可编辑 | `<ref>/components/chat-workspace/ContextReferenceHoverContent.vue:82` |
| CP-BOX-22 | 保存批注 | 悬停卡图标按钮（勾） | 隐藏 | 批注处于编辑态 | `<ref>/components/chat-workspace/ContextReferenceHoverContent.vue:72` |
| CP-BOX-23 | 移除批注 | 悬停卡图标按钮（叉） | 显示 | 悬停卡可编辑 | `<ref>/components/chat-workspace/ContextReferenceHoverContent.vue:91` |
| CP-BOX-24 | `{附件名 / 长文本标签}` | 附件片（类型图标 + 文本） | 显示 | 每个非图片附件 | `<ref>/components/chat-workspace/attachments/AttachmentChip.vue:53`、`:68` |
| CP-BOX-25 | （附件片预览按钮，无文案） | 按钮 | 显示 | 附件可预览 | `<ref>/components/chat-workspace/attachments/AttachmentChip.vue:58` |
| CP-BOX-26 | （附件片删除，叉号图标） | 图标按钮 | 显示 | 附件可移除 | `<ref>/components/chat-workspace/attachments/AttachmentChip.vue:84` |
| CP-BOX-27 | `{附件名}` | 文本附件对话框标题 | 隐藏 | 打开长文本附件编辑 | `<ref>/components/chat-workspace/attachments/TextAttachmentDialog.vue:75` |
| CP-BOX-28 | 正在加载... | 对话框状态 | 隐藏 | 附件内容加载中 | `<ref>/components/chat-workspace/attachments/TextAttachmentDialog.vue:91` |
| CP-BOX-29 | `{错误消息}` | 对话框错误文本 | 隐藏 | 附件加载失败 | `<ref>/components/chat-workspace/attachments/TextAttachmentDialog.vue:99` |
| CP-BOX-30 | （文本编辑区） | 多行输入框 | 显示 | 内容加载完成 | `<ref>/components/chat-workspace/attachments/TextAttachmentDialog.vue:101` |
| CP-BOX-31 | 取消 | 按钮 | 显示 | 可编辑模式 | `<ref>/components/chat-workspace/attachments/TextAttachmentDialog.vue:114` |
| CP-BOX-32 | 保存 | 主按钮（含保存中旋转图标） | 显示 | 可编辑模式，内容无变化时禁用 | `<ref>/components/chat-workspace/attachments/TextAttachmentDialog.vue:117` |

元素数：32

## C. 编辑器本体（ComposerEditor / mentions）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| CP-EDT-01 | 发送消息... | 多行富文本编辑区（占位） | 显示 | 始终；占位由输入区传入 | `<ref>/components/chat-workspace/composer/InputArea.vue:2334`；`<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue:1361` |
| CP-EDT-02 | `{提示建议}` | 幽灵文本 | 隐藏 | 存在可采纳的提示建议且正文为空 | `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue:1353` |
| CP-EDT-03 | `{文件名 / 目录名}` | 文件提及片（图标 + 文本） | — | 文档内插入文件提及 | `<ref>/components/chat-workspace/composer/tiptap/FileMentionNodeView.vue:55` |
| CP-EDT-04 | `{提交短哈希}` | 提交提及片（图标 + 文本） | — | 文档内插入提交提及 | `<ref>/components/chat-workspace/composer/tiptap/CommitMentionNodeView.vue:15` |
| CP-EDT-05 | `{链接地址}` | 链接提及片（站点图标 + 文本） | — | 粘贴链接或插入链接提及 | `<ref>/components/chat-workspace/composer/tiptap/LinkMentionNodeView.vue:40` |
| CP-EDT-06 | `{技能显示名}` | 技能提及片（方块图标 + 文本） | — | 文档内插入技能提及 | `<ref>/components/chat-workspace/composer/tiptap/SkillMentionNodeView.vue:22` |
| CP-EDT-07 | mark `{markId}` | 标记提及 | — | 文档内含标记提及 | `<ref>/components/chat-workspace/composer/tiptap/MarkMentionNodeView.vue:14` |
| CP-EDT-08 | 剪切 | 右键菜单项 | 隐藏 | 编辑器右键且有选区 | `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue:1369` |
| CP-EDT-09 | 复制 | 右键菜单项 | 隐藏 | 编辑器右键且有选区 | `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue:1372` |
| CP-EDT-10 | 粘贴 | 右键菜单项 | 隐藏 | 编辑器右键 | `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue:1375` |
| CP-EDT-11 | （分隔线，无文案） | 分隔线 | 隐藏 | 编辑器右键 | `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue:1378` |
| CP-EDT-12 | 全选 | 右键菜单项 | 隐藏 | 编辑器右键 | `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue:1379` |

元素数：12

## D. @ 提及菜单（ComposerEditor）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| CP-MEN-01 | （提及菜单面板，无标题） | 浮层 | 隐藏 | 输入 `@` 触发 | `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue:1386` |
| CP-MEN-02 | （加载骨架两条，无文案） | 骨架行 | 隐藏 | 提及结果请求中 | `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue:1393` |
| CP-MEN-03 | 未找到文件或目录 | 空态文本 | 隐藏 | 有查询无匹配 | `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue:1399` |
| CP-MEN-04 | 当前无工作区目录 | 空态文本 | 隐藏 | 无工作区目录可搜索 | 文案 `<ref>/i18n/zh-CN.ts:1592`；渲染点 `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue:1403` |
| CP-MEN-05 | 未找到技能 | 空态文本 | 隐藏 | 技能提及无匹配 | 文案 `<ref>/i18n/zh-CN.ts:1594`；渲染点 `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue:1403` |
| CP-MEN-06 | 文件 | 分组标题 | 显示 | 首个结果组为文件 | `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue:1410` |
| CP-MEN-07 | 技能 | 分组标题 | 显示 | 出现技能结果组 | `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue:1410` |
| CP-MEN-08 | `{文件名}` | 结果行主文本 | 显示 | 每条文件结果 | `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue:1424` |
| CP-MEN-09 | `{所在目录}` | 结果行次文本 | 隐藏 | 结果项有目录信息 | `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue:1425` |
| CP-MEN-10 | `{次要信息}` | 结果行尾文本 | 隐藏 | 结果项有次要信息 | `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue:1431` |

元素数：10

## E. / 斜杠菜单（ComposerEditor / desktop-slash-commands）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| CP-SLS-01 | （斜杠菜单面板，无标题） | 浮层 | 隐藏 | 输入 `/` 触发 | `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue:1442` |
| CP-SLS-02 | （加载骨架两条，无文案） | 骨架行 | 隐藏 | 命令列表请求中 | `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue:1449` |
| CP-SLS-03 | 未找到命令或技能 | 空态文本 | 隐藏 | 有查询无匹配 | `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue:1455` |
| CP-SLS-04 | 指令 | 分组标题 | 显示 | 首个结果组为命令 | `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue:1466` |
| CP-SLS-05 | 技能 | 分组标题 | 显示 | 出现技能结果组 | `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue:1466` |
| CP-SLS-06 | `{命令或技能名}` | 结果行主文本 | 显示 | 每条结果 | `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue:1481` |
| CP-SLS-07 | `{描述}` | 结果行右文本 | 隐藏 | 结果项有描述 | `<ref>/components/chat-workspace/composer/tiptap/ComposerEditor.vue:1482` |
| CP-SLS-08 | /agent 切换至 Agent 模式、/goal 切换至 Goal 模式、/pet 唤醒或收起 Pet、/settings 打开设置、/models 打开模型设置、/mcp 打开 MCP 设置、/skills 打开技能、/reload-skills 重新扫描磁盘上的技能、/reload-plugins 重新扫描已安装插件、/compact 压缩当前上下文、/fast Toggle fast mode、/branch（侧聊标题）、/workflow Browse workflows | 内置命令结果项（13 条，按可用性过滤） | 显示 | 桌面端命令，`fast` 不可用时隐藏；弹窗窗口中隐藏设置/技能类命令 | `<ref>/components/chat-workspace/composer/desktop-slash-commands.ts:34`、`:119` |

元素数：8

## F. 底部工具行（InputArea / AgentModeSelector / ComposerSkillsSelector）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| CP-TLS-01 | （加号按钮，无文案） | 图标按钮 | 显示 | 非紧凑模式 | `<ref>/components/chat-workspace/composer/InputArea.vue:2460` |
| CP-TLS-02 | 添加图片或文件 | 提示浮层 | 隐藏 | 悬停加号按钮 | `<ref>/components/chat-workspace/composer/InputArea.vue:2469` |
| CP-TLS-03 | （模式选择按钮，图标；展开时显示当前模式名） | 按钮 | 显示 | 非紧凑模式 | `<ref>/components/chat-workspace/composer/AgentModeSelector.vue:76`、`:87` |
| CP-TLS-04 | 切换 Agent 模式 | 提示浮层 | 隐藏 | 悬停模式按钮 | `<ref>/components/chat-workspace/composer/AgentModeSelector.vue:90` |
| CP-TLS-05 | Agent | 菜单项（双箭头图标，选中打勾） | 显示 | 菜单打开 | `<ref>/components/chat-workspace/composer/AgentModeSelector.vue:96` |
| CP-TLS-06 | Goal | 菜单项（靶心图标，选中打勾） | 显示 | 菜单打开，Goal 能力检查中/不可用时禁用 | `<ref>/components/chat-workspace/composer/AgentModeSelector.vue:106` |
| CP-TLS-07 | （Goal 检查中旋转图标，无文案） | 图标 | 隐藏 | Goal 能力检查中 | `<ref>/components/chat-workspace/composer/AgentModeSelector.vue:117` |
| CP-TLS-08 | Goal 正在启动… | 菜单项禁用提示 | 隐藏 | Goal 能力检查中 | `<ref>/components/chat-workspace/composer/AgentModeSelector.vue:110` |
| CP-TLS-09 | 此会话暂不可用 Goal 模式，Agent 模式仍可使用。 | 菜单项禁用提示 | 隐藏 | Goal 能力不可用 | `<ref>/components/chat-workspace/composer/AgentModeSelector.vue:112` |
| CP-TLS-10 | （技能按钮，方块图标；展开时显示「技能」） | 按钮 | 显示 | 非紧凑模式 | `<ref>/components/chat-workspace/composer/ComposerSkillsSelector.vue:43`、`:47` |
| CP-TLS-11 | 浏览并插入技能 | 提示浮层 | 隐藏 | 悬停技能按钮 | `<ref>/components/chat-workspace/composer/ComposerSkillsSelector.vue:72` |
| CP-TLS-12 | 搜索技能... | 技能面板搜索框（占位） | 显示 | 技能面板打开 | `<ref>/components/chat-workspace/composer/ComposerSkillsSelector.vue:53` |
| CP-TLS-13 | 正在加载... | 技能面板状态 | 隐藏 | 技能列表加载中 | `<ref>/components/chat-workspace/composer/ComposerSkillsSelector.vue:57` |
| CP-TLS-14 | `{错误消息}` | 技能面板错误 | 隐藏 | 技能加载失败 | `<ref>/components/chat-workspace/composer/ComposerSkillsSelector.vue:58` |
| CP-TLS-15 | 无活跃技能 | 技能面板空态 | 隐藏 | 无技能 | `<ref>/components/chat-workspace/composer/ComposerSkillsSelector.vue:59` |
| CP-TLS-16 | `{技能名}` + `{技能描述}` | 技能行 | 显示 | 每条匹配技能 | `<ref>/components/chat-workspace/composer/ComposerSkillsSelector.vue:61`、`:64`、`:65` |
| CP-TLS-17 | （紧凑模式加号菜单，无文案） | 菜单 | 隐藏 | 紧凑覆盖层模式 | `<ref>/components/chat-workspace/composer/InputArea.vue:2358` |
| CP-TLS-18 | 添加图片或文件 | 菜单项（回形针图标） | 显示 | 紧凑菜单打开 | `<ref>/components/chat-workspace/composer/InputArea.vue:2371` |
| CP-TLS-19 | Agent / Goal（附检查中与勾选态） | 子菜单项 | 显示 | 紧凑菜单内悬停模式项 | `<ref>/components/chat-workspace/composer/InputArea.vue:2383`、`:2392` |
| CP-TLS-20 | 技能 | 菜单项（方块图标，可再展开技能面板） | 显示 | 紧凑菜单打开 | `<ref>/components/chat-workspace/composer/InputArea.vue:2407` |

元素数：20

## G. 模型与推理强度（model-selector/**，渲染于输入区右下）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| CP-MDL-01 | `{模型名}` | 模型按钮（供应商图标 + 名称） | 显示 | 可选模型且非紧凑模式 | `<ref>/components/chat-workspace/model-selector/ModelSelector.vue:233`、`:248` |
| CP-MDL-02 | （模型提示浮层：`{模型名}`） | 提示浮层 | 隐藏 | 悬停模型按钮 | `<ref>/components/chat-workspace/model-selector/ModelSelector.vue:260`、`:262` |
| CP-MDL-03 | `{供应商名}` | 提示浮层字段 | 隐藏 | 同上 | `<ref>/components/chat-workspace/model-selector/ModelSelector.vue:263` |
| CP-MDL-04 | 零数据保留 (ZDR) | 提示浮层标题 | 隐藏 | 该模型支持 ZDR | `<ref>/components/chat-workspace/model-selector/ModelSelector.vue:271` |
| CP-MDL-05 | Dim 与模型服务方均不保留你的对话内容，请求处理完成后即被丢弃。 | 提示浮层说明 | 隐藏 | 该模型支持 ZDR | `<ref>/components/chat-workspace/model-selector/ModelSelector.vue:272` |
| CP-MDL-06 | ZDR | 徽标 | 隐藏 | 该模型支持 ZDR 且非紧凑模式 | `<ref>/components/chat-workspace/model-selector/ZdrBadge.vue:24`、`:40` |
| CP-MDL-07 | （Fast 闪电指示，无文案） | 图标 | 隐藏 | 已选优先服务档 | `<ref>/components/chat-workspace/model-selector/ModelSelector.vue:253` |
| CP-MDL-08 | 模型加载中… | 只读按钮（含旋转图标） | 隐藏 | 模型列表加载中 | `<ref>/components/chat-workspace/model-selector/ModelSelector.vue:393` |
| CP-MDL-09 | 加载失败 · 重试 | 按钮（危险色） | 隐藏 | 模型列表加载失败 | `<ref>/components/chat-workspace/model-selector/ModelSelector.vue:404` |
| CP-MDL-10 | 暂无可用模型 | 只读按钮 | 隐藏 | 模型列表为空 | `<ref>/components/chat-workspace/model-selector/ModelSelector.vue:414` |
| CP-MDL-11 | 未配置 | 按钮（打开模型设置） | 隐藏 | 无可用模型来源 | `<ref>/components/chat-workspace/model-selector/ModelSelector.vue:427` |
| CP-MDL-12 | 模型 | 弹层标签页 | 显示 | 弹层打开，存在多标签 | `<ref>/components/chat-workspace/model-selector/ModelSelector.vue:289`、`:304` |
| CP-MDL-13 | 子代理 | 弹层标签页 | 显示 | 支持子代理策略页 | `<ref>/components/chat-workspace/model-selector/ModelSelector.vue:293` |
| CP-MDL-14 | 外部 agent | 弹层标签页 | 显示 | 支持外部 agent 页 | `<ref>/components/chat-workspace/model-selector/ModelSelector.vue:293` |
| CP-MDL-15 | 搜索模型... | 弹层搜索框（占位） | 显示 | 模型标签页 | `<ref>/components/chat-workspace/model-selector/ModelSelector.vue:340` |
| CP-MDL-16 | 最近使用 | 供应商列表项（时钟图标） | 隐藏 | 有可解析的最近模型 | `<ref>/components/chat-workspace/model-selector/ProviderPane.vue:50`、`:60` |
| CP-MDL-17 | （供应商列表分隔线，无文案） | 分隔线 | 隐藏 | 最近使用项存在 | `<ref>/components/chat-workspace/model-selector/ProviderPane.vue:63` |
| CP-MDL-18 | `{供应商名}` | 供应商列表项（图标 + 文本） | 显示 | 每个供应商分组 | `<ref>/components/chat-workspace/model-selector/ProviderPane.vue:65`、`:76` |
| CP-MDL-19 | （当前模型勾选，无文案） | 图标 | 隐藏 | 该行为当前模型 | `<ref>/components/chat-workspace/model-selector/ModelPane.vue:85` |
| CP-MDL-20 | `{模型名}` | 模型行 | 显示 | 每条模型 | `<ref>/components/chat-workspace/model-selector/ModelPane.vue:94` |
| CP-MDL-21 | ZDR | 模型行徽标 | 隐藏 | 该模型支持 ZDR | `<ref>/components/chat-workspace/model-selector/ModelPane.vue:95` |
| CP-MDL-22 | `{模型名}` / `{供应商}` / `{上下文长度}` / ZDR 说明 | 模型行提示浮层 | 隐藏 | 悬停模型行 | `<ref>/components/chat-workspace/model-selector/ModelPane.vue:98`、`:101`、`:105`、`:118` |
| CP-MDL-23 | 没有可用模型 | 模型列表空态 | 隐藏 | 该供应商无模型 | `<ref>/components/chat-workspace/model-selector/ModelPane.vue:63` |
| CP-MDL-24 | （搜索覆盖层，ps：平铺搜索结果） | 浮层 | 隐藏 | 搜索框有输入 | `<ref>/components/chat-workspace/model-selector/SearchOverlay.vue:21` |
| CP-MDL-25 | 默认 / Fast / 自动 / Flex | 分段切换组（按模型支持项渲染） | 隐藏 | 该模型有服务档选项；Fast 项带闪电图标 | `<ref>/components/chat-workspace/model-selector/ServiceTierControl.vue:38`、`:48`、`:59` |
| CP-MDL-26 | `{强度短标签}` | 推理强度按钮（四格条 + 短标签） | 显示 | 当前模型有推理强度选项 | `<ref>/components/chat-workspace/model-selector/EffortControl.vue:117` |
| CP-MDL-27 | 思考强度 | 提示浮层 / 滑杆标题 | 隐藏 | 悬停或打开强度面板 | `<ref>/components/chat-workspace/model-selector/EffortControl.vue:158`、`:177` |
| CP-MDL-28 | 关闭 / 极低 / 低 / 中 / 高 / 极高 / 最高 | 强度滑杆刻度标签 | 隐藏 | 强度面板打开 | `<ref>/components/chat-workspace/model-selector/EffortControl.vue:68`、`:180` |

元素数：28

## H. 发送与停止（ConversationComposer）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| CP-SND-01 | （发送按钮，上箭头图标，可访问名「发送消息...」） | 图标按钮 | 显示 | 可发送时可用 | `<ref>/components/chat-workspace/composer/ConversationComposer.vue:87`、`:53` |
| CP-SND-02 | （停止按钮，方块图标，可访问名「停止回复」） | 图标按钮 | 隐藏 | 运行中且无草稿 | `<ref>/components/chat-workspace/composer/ConversationComposer.vue:87`、`:89` |
| CP-SND-03 | Esc | 键帽覆盖层 | 隐藏 | 已点击停止一次，待二次确认 | `<ref>/components/chat-workspace/composer/ConversationComposer.vue:84` |
| CP-SND-04 | 再按一次 Esc 停止。 | 屏幕阅读器文本 | 隐藏 | 同上 | `<ref>/components/chat-workspace/composer/ConversationComposer.vue:86` |

元素数：4

## I. 队列区（FollowUpMessagesView / FollowUpMessages）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| CP-QUE-01 | 队列 · {count} | 队列头（列表图标 + 文本） | 隐藏 | 队列条数 > 0 | `<ref>/components/chat-workspace/composer/FollowUpMessagesView.vue:94`、`:95` |
| CP-QUE-02 | 引导中… | 发送中行状态文本 | 隐藏 | 该条处于引导中 | `<ref>/components/chat-workspace/composer/FollowUpMessagesView.vue:102`、`:105` |
| CP-QUE-03 | `{消息文本}` | 发送中行文本 | 隐藏 | 同上 | `<ref>/components/chat-workspace/composer/FollowUpMessagesView.vue:106` |
| CP-QUE-04 | `{消息文本}` | 待发行文本 | 显示 | 每条待发消息 | `<ref>/components/chat-workspace/composer/FollowUpMessagesView.vue:108`、`:116` |
| CP-QUE-05 | （拖拽把手，竖排点图标，可访问名「调整消息顺序」） | 把手 | 隐藏 | 悬停该行且可排序 | `<ref>/components/chat-workspace/composer/FollowUpMessagesView.vue:112` |
| CP-QUE-06 | 编辑 | 图标按钮（铅笔，悬停出现） | 隐藏 | 悬停该行且允许编辑 | `<ref>/components/chat-workspace/composer/FollowUpMessagesView.vue:117` |
| CP-QUE-07 | 移除 | 图标按钮（垃圾桶，悬停出现） | 隐藏 | 悬停该行 | `<ref>/components/chat-workspace/composer/FollowUpMessagesView.vue:120` |
| CP-QUE-08 | 引导 | 文字按钮 | 隐藏 | 会话运行中且该条可引导 | `<ref>/components/chat-workspace/composer/FollowUpMessagesView.vue:123` |
| CP-QUE-09 | （底部渐隐遮罩，无文案） | 遮罩 | 隐藏 | 队列条数 > 3 | `<ref>/components/chat-workspace/composer/FollowUpMessagesView.vue:129` |

元素数：9

## J. 活动条（ComposerActivityStrip 及其 chip）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| CP-ACT-01 | （活动条容器，无文案） | 容器 | 隐藏 | 任一活动 chip 存在 | `<ref>/components/chat-workspace/composer/activity/ComposerActivityStrip.vue:78` |
| CP-ACT-02 | Goal · `{状态}` | Goal chip（靶心图标） | 隐藏 | 会话有未完成目标 | `<ref>/components/chat-workspace/composer/activity/GoalActivityChip.vue:70` |
| CP-ACT-03 | Goal | Goal 弹层标签 | 隐藏 | 打开 Goal chip | `<ref>/components/chat-workspace/composer/activity/GoalActivityChip.vue:90` |
| CP-ACT-04 | 编辑目标 | 弹层图标按钮（铅笔） | 隐藏 | 目标可编辑 | `<ref>/components/chat-workspace/composer/activity/GoalActivityChip.vue:92` |
| CP-ACT-05 | 暂停目标 | 弹层图标按钮（暂停） | 隐藏 | 目标可暂停 | `<ref>/components/chat-workspace/composer/activity/GoalActivityChip.vue:103` |
| CP-ACT-06 | 继续目标 | 弹层图标按钮（播放） | 隐藏 | 目标可继续 | `<ref>/components/chat-workspace/composer/activity/GoalActivityChip.vue:114` |
| CP-ACT-07 | 清除目标 | 弹层图标按钮（垃圾桶，危险色） | 隐藏 | 目标可清除 | `<ref>/components/chat-workspace/composer/activity/GoalActivityChip.vue:125` |
| CP-ACT-08 | `{目标正文}`（Markdown） | 弹层正文 | 隐藏 | 打开 Goal chip | `<ref>/components/chat-workspace/composer/activity/GoalActivityChip.vue:138` |
| CP-ACT-09 | `{状态}` · `{时长}` | 弹层状态行 | 隐藏 | 打开 Goal chip | `<ref>/components/chat-workspace/composer/activity/GoalActivityChip.vue:161` |
| CP-ACT-10 | `{暂停 / 阻塞原因}` | 弹层原因条 | 隐藏 | 目标暂停或阻塞 | `<ref>/components/chat-workspace/composer/activity/GoalActivityChip.vue:165` |
| CP-ACT-11 | {completed} / {total} 待办 | Todo chip（清单图标） | 隐藏 | 本轮有待办快照 | `<ref>/components/chat-workspace/composer/activity/TodoActivityChip.vue:23`、`:34` |
| CP-ACT-12 | （Todo chip 运行中旋转图标，无文案） | 图标 | 隐藏 | 有进行中的待办 | `<ref>/components/chat-workspace/composer/activity/TodoActivityChip.vue:35` |
| CP-ACT-13 | 任务进度 | Todo 弹层标题 | 隐藏 | 打开 Todo chip | `<ref>/components/chat-workspace/composer/activity/TodoActivityChip.vue:50` |
| CP-ACT-14 | `{待办文本}` | Todo 弹层行（状态图标 + 文本） | 隐藏 | 打开 Todo chip，每条待办 | `<ref>/components/chat-workspace/composer/activity/TodoActivityChip.vue:57`、`:73` |
| CP-ACT-15 | {count} 个智能体 | Agent chip（头像堆叠 + 文本） | 隐藏 | 存在子代理智能体 | `<ref>/components/chat-workspace/composer/activity/AgentActivityChip.vue:75`、`:97` |
| CP-ACT-16 | （Agent chip 运行中绿点，无文案） | 标记 | 隐藏 | 有运行中的智能体 | `<ref>/components/chat-workspace/composer/activity/AgentActivityChip.vue:98` |
| CP-ACT-17 | 子任务 | Agent 弹层标题 | 隐藏 | 打开 Agent chip | `<ref>/components/chat-workspace/composer/activity/AgentActivityChip.vue:110` |
| CP-ACT-18 | （Agent 行头像，无文案） | 头像 | 隐藏 | 每行一个智能体 | `<ref>/components/chat-workspace/composer/activity/AgentActivityChip.vue:129` |
| CP-ACT-19 | `{智能体名}` + 状态图标（运行中/失败/完成） | Agent 行主文本 | 隐藏 | 同上 | `<ref>/components/chat-workspace/composer/activity/AgentActivityChip.vue:144` |
| CP-ACT-20 | `{模型标签}` · `{详情}` | Agent 行次文本 | 隐藏 | 有模型或详情 | `<ref>/components/chat-workspace/composer/activity/AgentActivityChip.vue:149`、`:158`、`:162` |
| CP-ACT-21 | `{时长}` | Agent 行尾文本 | 隐藏 | 该智能体有时长 | `<ref>/components/chat-workspace/composer/activity/AgentActivityChip.vue:172` |
| CP-ACT-22 | 停止 | Agent 行停止按钮（悬停出现） | 隐藏 | 该智能体可停止 | `<ref>/components/chat-workspace/composer/activity/AgentActivityChip.vue:182` |
| CP-ACT-23 | {count} 条分支线 | 分支线 chip（分支图标） | 隐藏 | 存在并行分支线 | `<ref>/components/chat-workspace/composer/activity/BranchActivityChip.vue:38`、`:45` |
| CP-ACT-24 | （分支线 chip 运行中绿点，无文案） | 标记 | 隐藏 | 有运行中的分支线 | `<ref>/components/chat-workspace/composer/activity/BranchActivityChip.vue:46` |
| CP-ACT-25 | 分支线 | 分支弹层标题 | 隐藏 | 打开分支线 chip | `<ref>/components/chat-workspace/composer/activity/BranchActivityChip.vue:58` |
| CP-ACT-26 | `{分支标题}` + 状态图标（运行中/待命） | 分支弹层行 | 隐藏 | 打开分支弹层，每条分支 | `<ref>/components/chat-workspace/composer/activity/BranchActivityChip.vue:65`、`:79` |
| CP-ACT-27 | {count} 项唤醒 · {time} | 唤醒 chip（时钟图标） | 隐藏 | 存在待唤醒事项或面板打开 | `<ref>/components/chat-workspace/composer/activity/TaskActivityChip.vue:51`；文案在 `useComposerActivityViewModel.ts:329` |
| CP-ACT-28 | `{唤醒标题}` / `{元信息}` / `{时间}` / › | 唤醒弹层行 | 隐藏 | 打开唤醒弹层 | `<ref>/components/chat-workspace/composer/activity/TaskActivityChip.vue:69`、`:78`、`:79`、`:81`、`:82` |
| CP-ACT-29 | 暂无待唤醒事项 | 唤醒弹层空态 | 隐藏 | 无待唤醒事项 | `<ref>/components/chat-workspace/composer/activity/TaskActivityChip.vue:86` |
| CP-ACT-30 | {count} 个进程 | 后台进程 chip（终端图标） | 隐藏 | 存在后台进程或面板打开 | `<ref>/components/chat-workspace/composer/activity/TaskActivityChip.vue:51` |
| CP-ACT-31 | （后台进程 chip 旋转图标，无文案） | 图标 | 隐藏 | 有后台进程 | `<ref>/components/chat-workspace/composer/activity/TaskActivityChip.vue:59` |
| CP-ACT-32 | `{进程标题}` / `{元信息}` / `{时间}` / › | 后台进程弹层行 | 隐藏 | 打开进程弹层 | `<ref>/components/chat-workspace/composer/activity/TaskActivityChip.vue:69`、`:78`、`:79`、`:81`、`:82` |
| CP-ACT-33 | （进程行停止按钮） | 图标按钮 | 隐藏 | 该进程可停止 | `<ref>/components/chat-workspace/composer/activity/TaskActivityChip.vue:84` |
| CP-ACT-34 | 暂无运行进程 | 进程弹层空态 | 隐藏 | 无后台进程 | `<ref>/components/chat-workspace/composer/activity/TaskActivityChip.vue:86` |
| CP-ACT-35 | {count} 个文件改动 `+{insertions}` `-{deletions}` | 文件变更 chip（文件图标 + 折叠箭头） | 隐藏 | 本轮有文件改动 | `<ref>/components/chat-workspace/composer/activity/FileChangesActivityChip.vue:45`、`:52`、`:57`、`:58` |
| CP-ACT-36 | 本轮变更 / 本轮变更 · 进行中 | 文件变更弹层范围标签 | 隐藏 | 打开文件变更弹层 | `<ref>/components/chat-workspace/composer/activity/FileChangesActivityChip.vue:70` |
| CP-ACT-37 | 在 Reviews 中打开 | 弹层文字按钮（含外链图标） | 隐藏 | 预览可用 | `<ref>/components/chat-workspace/composer/activity/FileChangesActivityChip.vue:75` |
| CP-ACT-38 | （弹层关闭，叉号图标，可访问名「关闭」） | 图标按钮 | 显示 | 弹层打开 | `<ref>/components/chat-workspace/composer/activity/FileChangesActivityChip.vue:79` |
| CP-ACT-39 | `{错误消息}` + 重试 | 弹层错误行 | 隐藏 | 预览加载失败 | `<ref>/components/chat-workspace/composer/activity/FileChangesActivityChip.vue:85`、`:87` |
| CP-ACT-40 | 正在加载... | 弹层加载态 | 隐藏 | 预览加载中且暂无文件 | `<ref>/components/chat-workspace/composer/activity/FileChangesActivityChip.vue:89` |
| CP-ACT-41 | 没有 Last Run 改动 | 弹层空态 | 隐藏 | 本轮无改动文件 | `<ref>/components/chat-workspace/composer/activity/FileChangesActivityChip.vue:90` |
| CP-ACT-42 | `{文件名}` / `{目录}` / `+{n}` / `-{n}` | 改动文件行 | 隐藏 | 每个改动文件 | `<ref>/components/chat-workspace/composer/activity/FileChangesActivityChip.vue:92`、`:105`、`:106`、`:109`、`:110` |
| CP-ACT-43 | （文件 diff 折叠区：diff 代码块） | 折叠区 | 隐藏 | 展开某个文本可 diff 的文件 | `<ref>/components/chat-workspace/composer/activity/FileChangesActivityChip.vue:113`、`:120` |
| CP-ACT-44 | 无法预览文件 | diff 占位文本 | 隐藏 | diff 不可用 | `<ref>/components/chat-workspace/composer/activity/FileChangesActivityChip.vue:124` |
| CP-ACT-45 | 没有可显示的文本差异。 | diff 占位文本 | 隐藏 | 该文件无文本差异 | `<ref>/components/chat-workspace/composer/activity/FileChangesActivityChip.vue:126` |

元素数：45

**输入区小计：16 + 32 + 12 + 10 + 8 + 20 + 28 + 4 + 9 + 45 = 184**

> 本轮自检（输入区）：逐项过了 ①发送不可用提示与重试；②Shell 面板：模式两态、状态五态、停止/关闭两按钮、输出五类尾注；③容器：常规/紧凑两形态、拖拽高亮；④附件：图片卡三态（加载/成功/失败 + 重试）、轨道滚动提示、图片预览对话框、附件片可预览/不可预览两态、引用片与悬停卡五件、长文本对话框五件；⑤编辑器：占位、提示建议、五种内联 chip、右键四项；⑥`@` 菜单三空态 + 两组标题 + 三列信息；⑦`/` 菜单一空态 + 两组标题 + 13 条内置命令；⑧工具行：加号/模式/技能三按钮与提示、模式菜单两态（含 Goal 检查中/不可用）、技能面板四态、紧凑菜单三项；⑨模型：触发/提示/ZDR/Fast、加载中/失败/空/未配置四态、弹层三页签、搜索、供应商列表、模型行与提示、Fast 分档四值、推理强度滑杆七档；⑩发送/停止两态 + Esc 键帽 + 读屏文本；⑪队列：引导中/待发两态、四按钮、渐隐；⑫活动条：六个 chip 各自的弹层与空态。

---

# 屏：主窗口 → 区域：标题栏

## A. 标题栏框架（ChatHeaderFrame）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TB-FRM-01 | （标题栏容器：可拖动区，顶部圆角） | 容器 | 显示 | — | `<ref>/components/chat-workspace/ChatHeaderFrame.vue:42` |
| TB-FRM-02 | （macOS 交通灯避让区，无文案） | 占位 | 显示 | macOS 且非全屏且左栏收起 | `<ref>/components/chat-workspace/ChatHeaderFrame.vue:49` |
| TB-FRM-03 | （侧栏关闭时的左侧导航组，无文案） | 容器 | 隐藏 | 左栏收起且非独立窗口 | `<ref>/components/chat-workspace/ChatHeaderFrame.vue:50` |
| TB-FRM-04 | （新建会话，方框笔图标，可访问名见下行） | 图标按钮 | 隐藏 | 左栏收起 | `<ref>/components/chat-workspace/ChatHeaderFrame.vue:56` |
| TB-FRM-05 | 在工作区中新建会话 ⌘N | 提示浮层 | 隐藏 | 悬停新建按钮；无工作区时为「新建会话 ⌘N」 | `<ref>/components/chat-workspace/ChatHeaderFrame.vue:58`；文案 `<ref>/components/chat-workspace/ChatHeader.vue:138`、`:137` |
| TB-FRM-06 | `{工作区名}` | 面包屑文本（超长截断） | 隐藏 | 有工作区标题且非默认工作区 | `<ref>/components/chat-workspace/ChatHeaderFrame.vue:61`；条件 `<ref>/components/chat-workspace/ChatHeader.vue:126` |
| TB-FRM-07 | / | 分隔符 | 隐藏 | 面包屑可见 | `<ref>/components/chat-workspace/ChatHeaderFrame.vue:63` |
| TB-FRM-08 | `{会话标题}` | 标题（截断 + 原生提示） | 显示 | — | `<ref>/components/chat-workspace/ChatHeaderFrame.vue:64` |
| TB-FRM-09 | （标题右侧操作槽：会话 ⋯ 菜单） | 插槽 | 隐藏 | 有会话 | `<ref>/components/chat-workspace/ChatHeaderFrame.vue:65` |
| TB-FRM-10 | （右侧动作槽） | 插槽 | 显示 | 搜索条关闭时可见 | `<ref>/components/chat-workspace/ChatHeaderFrame.vue:67` |
| TB-FRM-11 | （标题栏内会话搜索条挂载点） | 浮层 | 隐藏 | 搜索打开 | `<ref>/components/chat-workspace/ChatHeaderFrame.vue:44` |

元素数：11

## B. 侧栏关闭后的导航组（WorkspaceHeaderNavigation）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TB-NAV-01 | （展开侧边栏，可访问名「展开侧边栏」） | 图标按钮 | 显示 | 左栏收起且非独立窗口 | `<ref>/components/chat-workspace/WorkspaceHeaderNavigation.vue:35` |
| TB-NAV-02 | 展开侧边栏 ⌘B | 提示浮层 | 隐藏 | 悬停 | `<ref>/components/chat-workspace/WorkspaceHeaderNavigation.vue:47` |
| TB-NAV-03 | （上一个工作区会话，左箭头） | 图标按钮 | 显示 | 未隐藏会话导航；无上一个时禁用 | `<ref>/components/chat-workspace/WorkspaceHeaderNavigation.vue:53` |
| TB-NAV-04 | 上一个工作区会话 | 提示浮层 | 隐藏 | 悬停 | `<ref>/components/chat-workspace/WorkspaceHeaderNavigation.vue:66` |
| TB-NAV-05 | （下一个工作区会话，右箭头） | 图标按钮 | 显示 | 未隐藏会话导航；无下一个时禁用 | `<ref>/components/chat-workspace/WorkspaceHeaderNavigation.vue:72` |
| TB-NAV-06 | 下一个工作区会话 | 提示浮层 | 隐藏 | 悬停 | `<ref>/components/chat-workspace/WorkspaceHeaderNavigation.vue:85` |

元素数：6

## C. 会话 ⋯ 菜单（ChatHeader title-actions）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TB-MNU-01 | （⋯ 触发按钮，无文案） | 图标按钮 | 显示 | 有会话 | `<ref>/components/chat-workspace/ChatHeader.vue:457` |
| TB-MNU-02 | 置顶 | 菜单项（图钉图标） | 显示 | 未置顶且非独立窗口 | `<ref>/components/chat-workspace/ChatHeader.vue:462` |
| TB-MNU-03 | 取消置顶 | 菜单项（取消置顶图标） | 显示 | 已置顶且非独立窗口 | `<ref>/components/chat-workspace/ChatHeader.vue:464` |
| TB-MNU-04 | 重命名 | 菜单项（右箭头图标） | 显示 | 重命名能力就绪 | `<ref>/components/chat-workspace/ChatHeader.vue:466`、`:120` |
| TB-MNU-05 | 重命名（{reason}） | 菜单项（禁用） | 隐藏 | 重命名能力检查中/不可用 | `<ref>/components/chat-workspace/ChatHeader.vue:120` |
| TB-MNU-06 | 在新窗口中打开 | 菜单项（外链图标） | 显示 | 非独立窗口 | `<ref>/components/chat-workspace/ChatHeader.vue:476` |
| TB-MNU-07 | 导出 | 菜单项（下载图标） | 显示 | — | `<ref>/components/chat-workspace/ChatHeader.vue:480` |
| TB-MNU-08 | 归档 | 菜单项（归档图标） | 显示 | — | `<ref>/components/chat-workspace/ChatHeader.vue:484` |
| TB-MNU-09 | （分隔线，无文案） | 分隔线 | 显示 | — | `<ref>/components/chat-workspace/ChatHeader.vue:488` |
| TB-MNU-10 | 复制会话链接 | 菜单项（链接图标） | 隐藏 | 调试菜单项开启 | `<ref>/components/chat-workspace/ChatHeader.vue:489` |
| TB-MNU-11 | 复制会话 ID | 菜单项（文件代码图标） | 隐藏 | 调试菜单项开启 | `<ref>/components/chat-workspace/ChatHeader.vue:493` |
| TB-MNU-12 | 复制会话路径 | 菜单项（文件夹打开图标） | 显示 | 有工作目录（无则禁用） | `<ref>/components/chat-workspace/ChatHeader.vue:497` |
| TB-MNU-13 | 复制调试信息 | 菜单项（文件代码图标） | 隐藏 | 调试菜单项开启 | `<ref>/components/chat-workspace/ChatHeader.vue:501` |

元素数：13

## D. 打开方式组（ChatHeader right-actions）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TB-OPN-01 | （打开方式按钮组容器，带描边） | 容器 | 显示 | 办公模式隐藏 | `<ref>/components/chat-workspace/ChatHeader.vue:510`、`:515` |
| TB-OPN-02 | （主打开按钮，图标随默认打开方式解析） | 图标按钮 | 显示 | 同上 | `<ref>/components/chat-workspace/ChatHeader.vue:516` |
| TB-OPN-03 | 在 VS Code 中打开 | 主按钮提示 / 可访问名 | 隐藏 | 默认打开方式为 VS Code | `<ref>/components/chat-workspace/ChatHeader.vue:521`、`:177`；`<ref>/components/chat-workspace/open-ide.ts:40` |
| TB-OPN-04 | 在 Cursor 中打开 | 同上 | 隐藏 | 默认打开方式为 Cursor | `<ref>/components/chat-workspace/open-ide.ts:41` |
| TB-OPN-05 | 在 Zed 中打开 | 同上 | 隐藏 | 默认打开方式为 Zed | `<ref>/components/chat-workspace/open-ide.ts:42` |
| TB-OPN-06 | 在 Terminal 中打开 | 同上 | 隐藏 | 默认打开方式为 Terminal | `<ref>/components/chat-workspace/open-ide.ts:43` |
| TB-OPN-07 | 在 Ghostty 中打开 | 同上 | 隐藏 | 默认打开方式为 Ghostty | `<ref>/components/chat-workspace/open-ide.ts:44` |
| TB-OPN-08 | 在 Warp 中打开 | 同上 | 隐藏 | 默认打开方式为 Warp | `<ref>/components/chat-workspace/open-ide.ts:45` |
| TB-OPN-09 | 在 Xcode 中打开 | 同上 | 隐藏 | 默认打开方式为 Xcode | `<ref>/components/chat-workspace/open-ide.ts:46` |
| TB-OPN-10 | 在 IntelliJ IDEA 中打开 | 同上 | 隐藏 | 默认打开方式为 IntelliJ | `<ref>/components/chat-workspace/open-ide.ts:47` |
| TB-OPN-11 | 在 WebStorm 中打开 | 同上 | 隐藏 | 默认打开方式为 WebStorm | `<ref>/components/chat-workspace/open-ide.ts:48` |
| TB-OPN-12 | 在 PyCharm 中打开 | 同上 | 隐藏 | 默认打开方式为 PyCharm | `<ref>/components/chat-workspace/open-ide.ts:49` |
| TB-OPN-13 | 在 Sublime Text 中打开 | 同上 | 隐藏 | 默认打开方式为 Sublime | `<ref>/components/chat-workspace/open-ide.ts:50` |
| TB-OPN-14 | 在 iTerm2 中打开 | 同上 | 隐藏 | 默认打开方式为 iTerm2 | `<ref>/components/chat-workspace/open-ide.ts:51` |
| TB-OPN-15 | 在 Alacritty 中打开 | 同上 | 隐藏 | 默认打开方式为 Alacritty | `<ref>/components/chat-workspace/open-ide.ts:52` |
| TB-OPN-16 | 在 WezTerm 中打开 | 同上 | 隐藏 | 默认打开方式为 WezTerm | `<ref>/components/chat-workspace/open-ide.ts:53` |
| TB-OPN-17 | 在 Kitty 中打开 | 同上 | 隐藏 | 默认打开方式为 Kitty | `<ref>/components/chat-workspace/open-ide.ts:54` |
| TB-OPN-18 | 在 Finder 中显示 / 在文件资源管理器中显示 / 打开所在文件夹 | 主按钮提示 / 可访问名 | 隐藏 | 默认打开方式为文件管理器（按平台选文案） | `<ref>/components/chat-workspace/open-ide.ts:62` |
| TB-OPN-19 | 用系统默认方式打开 | 主按钮提示 / 可访问名 | 显示 | 未解析出默认应用时的回退 | `<ref>/components/chat-workspace/open-ide.ts:57`；`<ref>/components/chat-workspace/ChatHeader.vue:183` |
| TB-OPN-20 | `{默认打开应用配置错误}` | 主按钮提示 | 隐藏 | 默认打开应用配置无效 | `<ref>/components/chat-workspace/ChatHeader.vue:180` |
| TB-OPN-21 | （更多打开方式，下拉箭头，可访问名「更多打开方式」） | 图标按钮 | 显示 | 打开方式组可见 | `<ref>/components/chat-workspace/ChatHeader.vue:528`、`:532` |
| TB-OPN-22 | 正在检测已安装应用… | 菜单项（禁用） | 隐藏 | 应用列表加载中 | `<ref>/components/chat-workspace/ChatHeader.vue:545` |
| TB-OPN-23 | `{应用检测错误}` | 菜单项（禁用） | 隐藏 | 检测失败 | `<ref>/components/chat-workspace/ChatHeader.vue:548` |
| TB-OPN-24 | 在 `{appName}` 中打开 | 菜单项（应用图标） | 隐藏 | 逐个可用应用 | `<ref>/components/chat-workspace/ChatHeader.vue:551`、`:561` |
| TB-OPN-25 | 未检测到可用应用 | 菜单项（禁用） | 隐藏 | 应用列表为空 | `<ref>/components/chat-workspace/ChatHeader.vue:563` |
| TB-OPN-26 | （分隔线，无文案） | 分隔线 | 隐藏 | 有可用应用 | `<ref>/components/chat-workspace/ChatHeader.vue:566` |
| TB-OPN-27 | 在 Finder 中显示 / 在文件资源管理器中显示 / 打开所在文件夹 | 菜单项（文件夹打开图标） | 显示 | 菜单打开 | `<ref>/components/chat-workspace/ChatHeader.vue:567`；文案 `<ref>/components/chat-workspace/open-ide.ts:62` |
| TB-OPN-28 | 用系统默认方式打开 | 菜单项（系统图标） | 显示 | 菜单打开 | `<ref>/components/chat-workspace/ChatHeader.vue:571` |
| TB-OPN-29 | （竖分隔符，无文案） | 分隔符 | 显示 | 打开方式组可见 | `<ref>/components/chat-workspace/ChatHeader.vue:579` |

元素数：29

## E. 右栏开关（ChatHeader / NewThreadHeader）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TB-RPN-01 | （右栏开关，展开/收起图标，可访问名见下两行） | 图标按钮 | 显示 | — | `<ref>/components/chat-workspace/ChatHeader.vue:584`、`:592` |
| TB-RPN-02 | 打开上下文面板 | 可访问名 / 提示浮层 | 显示 | 右栏收起 | `<ref>/components/chat-workspace/ChatHeader.vue:144`、`:596` |
| TB-RPN-03 | 关闭上下文面板 | 可访问名 / 提示浮层 | 隐藏 | 右栏展开 | `<ref>/components/chat-workspace/ChatHeader.vue:144` |

元素数：3

## F. 标题栏内会话搜索条（ThreadSearchBar）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TB-SCH-01 | （搜索条容器，浮于标题栏右上） | 浮层 | 隐藏 | 打开会话内搜索 | `<ref>/components/chat-workspace/ChatHeaderFrame.vue:44`；`<ref>/components/chat-workspace/timeline/ThreadSearchBar.vue:112` |
| TB-SCH-02 | （搜索中旋转图标，无文案） | 图标 | 隐藏 | 搜索请求进行中 | `<ref>/components/chat-workspace/timeline/ThreadSearchBar.vue:118` |
| TB-SCH-03 | （放大镜图标，无文案） | 图标 | 显示 | 非搜索中 | `<ref>/components/chat-workspace/timeline/ThreadSearchBar.vue:123` |
| TB-SCH-04 | 搜索聊天... | 输入框（占位，可访问名「搜索聊天消息」） | 显示 | 搜索条打开 | `<ref>/components/chat-workspace/timeline/ThreadSearchBar.vue:128`、`:135`、`:136` |
| TB-SCH-05 | 关闭搜索 | 图标按钮（叉号） | 显示 | 搜索条打开 | `<ref>/components/chat-workspace/timeline/ThreadSearchBar.vue:141` |
| TB-SCH-06 | 上一个搜索结果 | 图标按钮（上箭头） | 隐藏 | 有关键词；无结果时禁用 | `<ref>/components/chat-workspace/timeline/ThreadSearchBar.vue:160` |
| TB-SCH-07 | 下一个搜索结果 | 图标按钮（下箭头） | 隐藏 | 有关键词；无结果时禁用 | `<ref>/components/chat-workspace/timeline/ThreadSearchBar.vue:172` |
| TB-SCH-08 | {current} / {total} 个结果 | 结果计数文本 | 隐藏 | 有关键词且非搜索中 | `<ref>/components/chat-workspace/timeline/ThreadSearchBar.vue:185`、`:37` |
| TB-SCH-09 | 0 个结果 | 结果计数文本 | 隐藏 | 有关键词且无结果 | `<ref>/components/chat-workspace/timeline/ThreadSearchBar.vue:35` |

元素数：9

## G. 新会话标题栏（NewThreadHeader，标题栏变体）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TB-NTH-01 | （macOS 交通灯避让区，无文案） | 占位 | 显示 | macOS 且非全屏且左栏收起 | `<ref>/components/chat-workspace/NewThreadHeader.vue:38` |
| TB-NTH-02 | （展开侧边栏，左栏打开图标） | 图标按钮 | 隐藏 | 左栏收起 | `<ref>/components/chat-workspace/NewThreadHeader.vue:49` |
| TB-NTH-03 | 展开侧边栏 | 提示浮层 | 隐藏 | 悬停 | `<ref>/components/chat-workspace/NewThreadHeader.vue:59` |
| TB-NTH-04 | （拖动占位区，无文案） | 容器 | 显示 | — | `<ref>/components/chat-workspace/NewThreadHeader.vue:66` |
| TB-NTH-05 | （右栏开关，展开/收起图标） | 图标按钮 | 显示 | — | `<ref>/components/chat-workspace/NewThreadHeader.vue:73`、`:81` |
| TB-NTH-06 | 打开上下文面板 / 关闭上下文面板 | 可访问名 / 提示浮层 | 显示 | 随右栏状态切换 | `<ref>/components/chat-workspace/NewThreadHeader.vue:77`、`:86` |

元素数：6

**标题栏小计：11 + 6 + 13 + 29 + 3 + 9 + 6 = 77**

> 本轮自检（标题栏）：逐项过了 ①左栏开合两态（展开/收起、含 ⌘B 与无快捷键两版提示）；②面包屑三态（有工作区 / 默认工作区隐藏 / 无工作区标题）；③⋯ 菜单 6 项常态 + 3 项调试态 + 重命名就绪/检查中/不可用三态；④打开方式组 15 个应用标签、加载中、检测失败、空列表、Finder/资源管理器/所在文件夹三平台文案、系统默认回退、配置错误回退；⑤右栏开关两态；⑥会话搜索条：加载中、有结果、0 结果、关闭、上下条禁用；⑦新会话标题栏变体。

# 屏：主窗口 → 区域：状态条

## A. 左侧：工具权限与 Agent 配置

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| SB-PRM-01 | （状态条容器，无文案） | 容器 | 显示 | — | `<ref>/components/chat-workspace/StatusBar.vue:221` |
| SB-PRM-02 | 完全访问权限 | 权限按钮（闪电图标） | 显示 | 当前权限为「自动」 | `<ref>/components/chat-workspace/ToolPermissionSelector.vue:165`、`:167` |
| SB-PRM-03 | 智能权限 | 权限按钮（盾牌勾图标） | 隐藏 | 当前权限为「智能」 | `<ref>/components/chat-workspace/ToolPermissionSelector.vue:167`、`:77` |
| SB-PRM-04 | 默认权限 | 权限按钮（盾牌图标） | 隐藏 | 当前权限为「默认」 | `<ref>/components/chat-workspace/ToolPermissionSelector.vue:167`、`:77` |
| SB-PRM-05 | 完全访问权限 | 菜单项（闪电图标，当前项高亮） | 显示 | 菜单打开 | `<ref>/components/chat-workspace/ToolPermissionSelector.vue:177` |
| SB-PRM-06 | 智能权限 | 菜单项（盾牌勾图标） | 显示 | 菜单打开且非 App Server 会话 | `<ref>/components/chat-workspace/ToolPermissionSelector.vue:177`、`:65` |
| SB-PRM-07 | 默认权限 | 菜单项（盾牌图标） | 显示 | 菜单打开 | `<ref>/components/chat-workspace/ToolPermissionSelector.vue:177` |
| SB-PRM-08 | 所有工具自动执行，跳过权限确认 | 菜单项提示说明 | 隐藏 | 悬停「完全访问权限」 | `<ref>/components/chat-workspace/ToolPermissionSelector.vue:194`、`:49` |
| SB-PRM-09 | Agent 自动帮你审核命令和工具操作，不安全命令会询问，常规操作自动放行，安全智能又方便 | 菜单项提示说明 | 隐藏 | 悬停「智能权限」 | `<ref>/components/chat-workspace/ToolPermissionSelector.vue:194`、`:55` |
| SB-PRM-10 | 工作区内写入自动执行，工作区外修改需确认 | 菜单项提示说明 | 隐藏 | 悬停「默认权限」 | `<ref>/components/chat-workspace/ToolPermissionSelector.vue:194`、`:61` |
| SB-PRM-11 | 确认开启完全放开？ | 确认框标题（危险图标） | 隐藏 | 切到完全访问权限前 | `<ref>/components/chat-workspace/ToolPermissionSelector.vue:205` |
| SB-PRM-12 | 开启后，AI 将减少确认步骤，并可直接执行更多操作，包括敏感操作、文件修改或外部执行。\n仅建议在你信任当前任务时使用。 | 确认框说明 | 隐藏 | 同上 | `<ref>/components/chat-workspace/ToolPermissionSelector.vue:209` |
| SB-PRM-13 | 我已了解风险，并愿意继续 | 复选框 | 隐藏 | 同上 | `<ref>/components/chat-workspace/ToolPermissionSelector.vue:214` |
| SB-PRM-14 | 取消 | 按钮 | 隐藏 | 同上 | `<ref>/components/chat-workspace/ToolPermissionSelector.vue:222` |
| SB-PRM-15 | 确认开启 | 主按钮（未勾选弱化，勾选后危险色） | 隐藏 | 同上 | `<ref>/components/chat-workspace/ToolPermissionSelector.vue:223` |
| SB-AGP-01 | `{Agent 配置名}` | 配置按钮（机器人图标；切换中为旋转图标） | 隐藏 | 可用配置 > 1 个 | `<ref>/components/chat-workspace/StatusBar.vue:228`；`<ref>/components/chat-workspace/AgentProfileSelector.vue:50`、`:58` |
| SB-AGP-02 | 请先停止当前轮次再切换配置。 | 按钮提示 | 隐藏 | 本会话正在运行 | `<ref>/components/chat-workspace/AgentProfileSelector.vue:36` |
| SB-AGP-03 | 此 App Server 未提供 Agent 配置。 | 按钮提示 | 隐藏 | 不支持配置更新 | `<ref>/components/chat-workspace/AgentProfileSelector.vue:37` |
| SB-AGP-04 | 切换此会话主线的 Agent 配置。 | 按钮提示 | 显示 | 可切换 | `<ref>/components/chat-workspace/AgentProfileSelector.vue:38` |
| SB-AGP-05 | Agent 配置 | 菜单标题 | 隐藏 | 菜单打开 | `<ref>/components/chat-workspace/AgentProfileSelector.vue:64` |
| SB-AGP-06 | `{配置名}` + `{描述}` | 菜单项（勾选/机器人图标） | 隐藏 | 每个可用配置 | `<ref>/components/chat-workspace/AgentProfileSelector.vue:67`、`:79`、`:80` |

元素数：21

## B. 分支选择器

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| SB-BRN-01 | `{当前分支名}` | 分支按钮（分支图标 + 文本） | 隐藏 | 有 git 目录且分支数据可用；办公模式隐藏 | `<ref>/components/chat-workspace/StatusBar.vue:243`、`:245` |
| SB-BRN-02 | 搜索分支 | 输入框（占位） | 隐藏 | 菜单打开 | `<ref>/components/chat-workspace/StatusBar.vue:252` |
| SB-BRN-03 | 分支 | 分组标题 | 隐藏 | 菜单打开 | `<ref>/components/chat-workspace/StatusBar.vue:262` |
| SB-BRN-04 | `{分支名}` | 分支项（分支图标 + 名称；当前分支带勾并高亮） | 隐藏 | 每条可见分支 | `<ref>/components/chat-workspace/StatusBar.vue:267`、`:278`、`:279` |
| SB-BRN-05 | （分隔线，无文案） | 分隔线 | 隐藏 | 菜单打开 | `<ref>/components/chat-workspace/StatusBar.vue:284` |
| SB-BRN-06 | 创建新分支 | 菜单项（加号图标） | 隐藏 | 菜单打开 | `<ref>/components/chat-workspace/StatusBar.vue:285` |

元素数：6

## C. 上下文用量（ContextUsageMeter）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| SB-CTX-01 | （用量触发：细进度条 + 百分比） | 按钮 | 显示 | 上下文用量开关打开 | `<ref>/components/chat-workspace/ContextUsageMeter.vue:31`；`<ref>/components/chat-workspace/StatusBar.vue:293` |
| SB-CTX-02 | `{百分比}` | 触发按钮数值 | 显示 | 同上；未测得时为「—」 | `<ref>/components/chat-workspace/ContextUsageMeter.vue:44` |
| SB-CTX-03 | 本轮上下文 `{tokens}` (`{百分比}`) | 悬停摘要 | 隐藏 | 悬停触发按钮 | `<ref>/components/chat-workspace/ContextUsageMeter.vue:47`、`:55` |
| SB-CTX-04 | · 套餐用量 `{百分比}%` | 悬停摘要尾段 | 隐藏 | 已订阅托管模型且有用量行 | `<ref>/components/chat-workspace/ContextUsageMeter.vue:57`；`<ref>/components/chat-workspace/StatusBar.vue:295` |
| SB-CTX-05 | · 套餐用量 不限量 | 悬停摘要尾段 | 隐藏 | 套餐不限量且无用量行 | `<ref>/components/chat-workspace/StatusBar.vue:295` |
| SB-CTX-06 | 本轮上下文 | 弹层标题 | 隐藏 | 点击触发按钮 | `<ref>/components/chat-workspace/ContextUsageMeter.vue:72` |
| SB-CTX-07 | `{百分比}` | 弹层标题右侧数值 | 隐藏 | 弹层打开 | `<ref>/components/chat-workspace/ContextUsageMeter.vue:73` |
| SB-CTX-08 | （弹层进度条，无文案） | 进度条 | 隐藏 | 弹层打开 | `<ref>/components/chat-workspace/ContextUsageMeter.vue:76` |
| SB-CTX-09 | `{已用} / {上下文窗口}` | 弹层 token 文本 | 隐藏 | 弹层打开；未测得为「—」 | `<ref>/components/chat-workspace/ContextUsageMeter.vue:82`、`:84` |
| SB-CTX-10 | 缓存命中 {pct}% | 弹层文本 | 隐藏 | 有缓存命中率 | `<ref>/components/chat-workspace/ContextUsageMeter.vue:86` |
| SB-CTX-11 | 会话消耗：{tokens} tokens | 弹层文本 | 隐藏 | 弹层打开 | `<ref>/components/chat-workspace/ContextUsageMeter.vue:93` |
| SB-CTX-12 | 速率：{rate} tok/s | 弹层文本 | 隐藏 | 有吞吐数据 | `<ref>/components/chat-workspace/ContextUsageMeter.vue:96` |

元素数：12

## D. 套餐用量段（上下文用量弹层内）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| SB-PLN-01 | 套餐用量 | 分节标题 | 隐藏 | 会话使用托管模型且已登录 | `<ref>/components/chat-workspace/StatusBar.vue:298`、`:300` |
| SB-PLN-02 | 账户 | 文字链接按钮（外链箭头） | 隐藏 | 非独立窗口 | `<ref>/components/chat-workspace/StatusBar.vue:301` |
| SB-PLN-03 | `{用量错误消息}` | 错误文本 | 隐藏 | 用量请求失败 | `<ref>/components/chat-workspace/StatusBar.vue:312` |
| SB-PLN-04 | 当前套餐不限用量，无额度上限。 | 说明文本 | 隐藏 | 不限量且无用量行 | `<ref>/components/chat-workspace/StatusBar.vue:316` |
| SB-PLN-05 | 正在加载用量… | 说明文本 | 隐藏 | 无用量行且请求中 | `<ref>/components/chat-workspace/StatusBar.vue:324` |
| SB-PLN-06 | 暂无用量数据 | 说明文本 | 隐藏 | 无用量行且无请求 | `<ref>/components/chat-workspace/StatusBar.vue:324` |
| SB-PLN-07 | `{窗口标签}`（5 小时窗口 / 1 周窗口 / 1 天窗口 / 月度用量） | 用量行标签 | 隐藏 | 每个用量行 | `<ref>/components/chat-workspace/StatusBar.vue:329`；`<ref>/composables/usePlanUsageRows.ts:99`、`:128` |
| SB-PLN-08 | `{百分比}%` | 用量行数值（按水位红/琥珀/常规） | 隐藏 | 每个用量行 | `<ref>/components/chat-workspace/StatusBar.vue:331` |
| SB-PLN-09 | `{剩余时长}后` | 用量行重置倒计时 | 隐藏 | 该行有重置时间 | `<ref>/components/chat-workspace/StatusBar.vue:339` |
| SB-PLN-10 | （用量行进度条，无文案） | 进度条 | 隐藏 | 每个用量行 | `<ref>/components/chat-workspace/StatusBar.vue:342` |

元素数：10

## E. 新建分支对话框（跨区，属 git 对话框）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| SB-DLG-01 | 创建新分支 | 对话框标题 | 隐藏 | 点击「创建新分支」 | `<ref>/components/chat-workspace/git/NewBranchDialog.vue:148` |
| SB-DLG-02 | 分支名称 | 字段标签 | 显示 | 对话框打开 | `<ref>/components/chat-workspace/git/NewBranchDialog.vue:153` |
| SB-DLG-03 | feature/my-branch | 输入框（占位） | 显示 | 同上 | `<ref>/components/chat-workspace/git/NewBranchDialog.vue:157` |
| SB-DLG-04 | `{分支名校验错误}` | 行内错误 | 隐藏 | 名称非法 | `<ref>/components/chat-workspace/git/NewBranchDialog.vue:161` |
| SB-DLG-05 | 基于 | 字段标签 | 显示 | 对话框打开 | `<ref>/components/chat-workspace/git/NewBranchDialog.vue:167` |
| SB-DLG-06 | 选择基础分支 / 加载中... | 下拉占位 | 显示 | 分支列表加载中显示「加载中...」 | `<ref>/components/chat-workspace/git/NewBranchDialog.vue:170` |
| SB-DLG-07 | `{分支名}` | 下拉选项 | 显示 | 每个可选基础分支 | `<ref>/components/chat-workspace/git/NewBranchDialog.vue:173` |
| SB-DLG-08 | `{错误消息}` | 错误文本 | 隐藏 | 创建失败 | `<ref>/components/chat-workspace/git/NewBranchDialog.vue:184` |
| SB-DLG-09 | 取消 | 按钮 | 显示 | 同上 | `<ref>/components/chat-workspace/git/NewBranchDialog.vue:190` |
| SB-DLG-10 | 创建 | 主按钮（创建中旋转图标） | 显示 | 名称无效时禁用 | `<ref>/components/chat-workspace/git/NewBranchDialog.vue:193` |

元素数：10

**状态条小计：21 + 6 + 12 + 10 + 10 = 59**

> 本轮自检（状态条）：逐项过了 ①三种工具权限的按钮文案/图标、菜单项、三条说明、App Server 会话隐藏「智能权限」、完全放开二次确认（标题/说明/复选框/弱化态与危险态）；②Agent 配置：按钮两态（机器人/切换中）、三种禁用提示、菜单标题、逐项（勾选/名称/描述）、仅 1 个配置时隐藏；③分支：按钮、搜索框、分组标题、当前分支勾选与高亮、分隔线、创建新分支；④上下文用量：触发条 + 百分比、两种悬停摘要（含不限量）、弹层五段、未测得「—」；⑤套餐用量：标题、账户跳转、错误、不限量、加载中、空数据、行（标签/百分比/倒计时/进度条/三档颜色）；⑥新建分支对话框 10 项（含占位与加载态）。

# 屏：主窗口 → 区域：时间线

## A. 时间线外壳（PlainTimeline / TimelineSkeleton）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TL-SHL-01 | 当前会话还没有消息 | 空态文本 | 隐藏 | 无消息且未运行且未在恢复 | `<ref>/components/chat-workspace/timeline/PlainTimeline.vue:3462` |
| TL-SHL-02 | （历史加载条，无文案） | 加载条 | 隐藏 | 正在加载更早历史 | `<ref>/components/chat-workspace/timeline/PlainTimeline.vue:3476` |
| TL-SHL-03 | 正在加载... | 屏幕阅读器文本 | 隐藏 | 同上 | `<ref>/components/chat-workspace/timeline/PlainTimeline.vue:3482` |
| TL-SHL-04 | （用户消息轨，挂载点） | 侧轨 | 隐藏 | 开启消息轨且有宿主 | `<ref>/components/chat-workspace/timeline/PlainTimeline.vue:3489` |
| TL-SHL-05 | （渲染窗口顶部占位，无文案） | 占位 | 隐藏 | 视口上方存在未渲染行 | `<ref>/components/chat-workspace/timeline/PlainTimeline.vue:3507` |
| TL-SHL-06 | （渲染窗口底部占位，无文案） | 占位 | 隐藏 | 视口下方存在未渲染行 | `<ref>/components/chat-workspace/timeline/PlainTimeline.vue:3569` |
| TL-SHL-07 | 正在处理 | 运行活动文本（流光） | 隐藏 | 运行中且未停在重试等待 | `<ref>/components/chat-workspace/timeline/PlainTimeline.vue:3586`、`:1201` |
| TL-SHL-08 | 正在处理 · `{已用时}` | 运行活动文本 | 隐藏 | 同上且有已用时 | `<ref>/components/chat-workspace/timeline/PlainTimeline.vue:1230` |
| TL-SHL-09 | （会话恢复遮罩：品牌动画，无文案） | 遮罩 | 隐藏 | 会话切换/恢复中 | `<ref>/components/chat-workspace/timeline/PlainTimeline.vue:3589`、`:3598` |
| TL-SHL-10 | （回到底部按钮，下箭头，无文案） | 浮动按钮 | 隐藏 | 视口离开底部 | `<ref>/components/chat-workspace/timeline/PlainTimeline.vue:3610` |
| TL-SHL-11 | （时间线加载骨架：品牌动画，无文案） | 占位 | 显示 | 时间线尚未就绪 | `<ref>/components/chat-workspace/timeline/TimelineSkeleton.vue:9` |

元素数：11

## B. 一轮（turn）行容器（TimelineRow）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TL-TRN-01 | （一轮容器，含头像组、悬停组） | 容器 | 显示 | 每个 turn 行 | `<ref>/components/chat-workspace/timeline/TimelineRow.vue:287` |
| TL-TRN-02 | （用户消息块，复用 C 组） | 块 | 显示 | 该轮有用户消息 | `<ref>/components/chat-workspace/timeline/TimelineRow.vue:288` |
| TL-TRN-03 | （助手消息区，复用 D 组） | 块 | 显示 | 该轮有内容块或终态 | `<ref>/components/chat-workspace/timeline/TimelineRow.vue:305` |
| TL-TRN-04 | （折叠脚区容器，无文案） | 容器 | 显示 | — | `<ref>/components/chat-workspace/timeline/TimelineRow.vue:335` |
| TL-TRN-05 | （产物区，复用 M 组） | 区块 | 隐藏 | 最新一轮、非只读、有 run | `<ref>/components/chat-workspace/timeline/TimelineRow.vue:341` |
| TL-TRN-06 | （文件改动区，复用 N 组） | 区块 | 隐藏 | 最新一轮、显示 git 面、有 run 与用户消息 | `<ref>/components/chat-workspace/timeline/TimelineRow.vue:353` |
| TL-TRN-07 | （助手操作行，复用 O 组） | 区块 | 隐藏 | 该轮有内容块或终态 | `<ref>/components/chat-workspace/timeline/TimelineRow.vue:364` |

元素数：7

## C. 用户消息块（TimelineUserMessage）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TL-USR-01 | （消息容器，可聚焦，右对齐） | 容器 | 显示 | 每条用户消息 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:893` |
| TL-USR-02 | （图片附件轨，复用输入区图片卡元素） | 区块 | 隐藏 | 有图片附件 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:902` |
| TL-USR-03 | （附件片架，复用附件 chip 元素） | 区块 | 隐藏 | 有非图片附件 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:908` |
| TL-USR-04 | （已发送上下文引用列表） | 区块 | 隐藏 | 有已发送引用且未编辑 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:922` |
| TL-USR-05 | mark `{序号}` | 标注卡标题 | 隐藏 | 有 mark 标注且未编辑 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:933`、`:941` |
| TL-USR-06 | `{元素选择器}` | 标注卡等宽文本 | 隐藏 | 同上 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:942` |
| TL-USR-07 | `{标注文本片段}` | 标注卡文本 | 隐藏 | 有文本片段 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:944` |
| TL-USR-08 | `{批注内容}` | 标注卡正文（可多行） | 隐藏 | 有批注 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:947` |
| TL-USR-09 | （气泡容器，无文案） | 容器 | 显示 | 有内容且未编辑 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:953` |
| TL-USR-10 | 定时唤醒 | 来源标记 | 隐藏 | 消息由定时唤醒发出 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:957` |
| TL-USR-11 | `{消息正文}` | 文本（超长折叠） | 显示 | 有内容 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:964` |
| TL-USR-12 | `{技能名}` | 技能提及 chip（方块图标 + 文本） | — | 正文含技能提及 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:984` |
| TL-USR-13 | `{文件名}` | 文件提及按钮（文件图标 + 文本，可点击打开） | — | 正文含文件提及 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:993`、`:1008` |
| TL-USR-14 | `{提交短哈希}` | 提交提及 chip（提交图标 + 文本） | — | 正文含提交提及 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1010`、`:1016` |
| TL-USR-15 | `{链接地址}` | 链接提及 chip（站点图标 + 文本） | — | 正文含链接提及 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1018`、`:1039` |
| TL-USR-16 | 展示全部 | 展开按钮（下箭头） | 隐藏 | 正文超出折叠高度且未展开 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1048`、`:1060` |
| TL-USR-17 | 收起 | 收起按钮（箭头翻转） | 隐藏 | 已展开 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1060` |
| TL-USR-18 | （编辑态输入框，复用输入区编辑器） | 富文本编辑区 | 隐藏 | 进入消息编辑 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1064`、`:1070` |
| TL-USR-19 | 取消 | 图标按钮（叉号） | 隐藏 | 编辑态 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1089` |
| TL-USR-20 | 发送消息... | 图标按钮（上箭头，title 为发送消息...） | 隐藏 | 编辑态，无改动时禁用 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1098`、`:1102` |
| TL-USR-21 | `{发送时间}` | 时间文本 | 隐藏 | 悬停/聚焦消息（有内容或附件） | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1112`、`:1117` |
| TL-USR-22 | 取消 | 撤回按钮（叉号，提示为取消） | 隐藏 | 消息可撤回 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1123`、`:1135` |
| TL-USR-23 | 复制整个消息 | 图标按钮（复制/已复制勾） | 隐藏 | 有内容或附件 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1137`、`:1146`、`:1150` |
| TL-USR-24 | 编辑消息 | 图标按钮（铅笔） | 隐藏 | 可编辑且有文本 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1152`、`:1165` |
| TL-USR-25 | 从此处分叉 | 图标按钮（分叉） | 隐藏 | 可分叉且非独立窗口 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1167`、`:1180` |
| TL-USR-26 | 从此处开始侧边对话 | 图标按钮（分支） | 隐藏 | 可分支且非独立窗口 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1182`、`:1195` |
| TL-USR-27 | 从此处开始侧边对话 | 右键菜单项（分支图标） | 隐藏 | 右键消息 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1203`、`:1210` |
| TL-USR-28 | 复制 | 右键菜单项（复制图标） | 隐藏 | 有选中文本 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1212`、`:1218` |
| TL-USR-29 | 复制整个消息 | 右键菜单项（复制图标） | 隐藏 | 有文本或粘贴文本附件 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1220`、`:1226` |
| TL-USR-30 | （划选浮条，复用 R 组） | 浮层 | 隐藏 | 选中正文文本 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1231` |
| TL-USR-31 | （长文本附件对话框，复用输入区对话框） | 对话框 | 隐藏 | 点开已发送的粘贴文本附件 | `<ref>/components/chat-workspace/timeline/TimelineUserMessage.vue:1240` |

元素数：31

## D. 助手消息块（TimelineAssistantMessage）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TL-AST-01 | （工作摘要行：两侧横线夹一个按钮） | 容器 | 隐藏 | 该轮有可折叠的工作块 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:896` |
| TL-AST-02 | 工作过程 | 摘要按钮文本 | 隐藏 | 无时长且无动作汇总 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:574` |
| TL-AST-03 | 已经工作了 {duration} | 摘要按钮文本 | 隐藏 | 有工作时长 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:569` |
| TL-AST-04 | 工作过程 · {actions} | 摘要按钮文本 | 隐藏 | 折叠且有动作汇总 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:572` |
| TL-AST-05 | 已经工作了 {duration} · {actions} | 摘要按钮文本 | 隐藏 | 折叠、有时长、有动作汇总 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:564` |
| TL-AST-06 | {duration} 后失败 | 摘要按钮文本（警示色） | 隐藏 | 该轮失败且有时长 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:549` |
| TL-AST-07 | {duration} 后失败 · {actions} | 摘要按钮文本（警示色） | 隐藏 | 折叠、失败、有时长、有动作汇总 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:544` |
| TL-AST-08 | 出错前的工作过程 | 摘要按钮文本（警示色） | 隐藏 | 该轮失败且无时长 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:554` |
| TL-AST-09 | 出错前的工作过程 · {actions} | 摘要按钮文本（警示色） | 隐藏 | 折叠、失败、无时长、有动作汇总 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:552` |
| TL-AST-10 | 已停止 | 摘要按钮文本 | 隐藏 | 该轮被中断且无时长 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:560` |
| TL-AST-11 | 已停止 · {duration} | 摘要按钮文本 | 隐藏 | 该轮被中断且有时长 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:558` |
| TL-AST-12 | （摘要行箭头，展开时旋转 90°） | 图标 | 隐藏 | 摘要行存在 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:912` |
| TL-AST-13 | （内容块容器，无文案） | 容器 | 显示 | 有内容块 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:917` |
| TL-AST-14 | `{正文 Markdown}` | 文本块 | 显示 | 每个 text 块 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:947`；`<ref>/components/chat-workspace/timeline/TimelineTextItem.vue:118` |
| TL-AST-15 | （媒体块，由 presentation 提供） | 块 | — | 每个 media 块 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:956` |
| TL-AST-16 | （子代理通知：机器人图标，无文案） | 图标 | 隐藏 | 该块为子代理通知 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:966`、`:981` |
| TL-AST-17 | `{智能体名}` | 通知内 agent chip（供应商图标 + 名称） | 隐藏 | 同上 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:984`、`:991` |
| TL-AST-18 | `{智能体模型标签}` | agent chip 提示浮层 | 隐藏 | 悬停 agent chip | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:994` |
| TL-AST-19 | `{子代理通知文案}` | 行内文本 | 隐藏 | 同上 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:998` |
| TL-AST-20 | （通知展开箭头，展开时旋转） | 图标 | 隐藏 | 可跳转或可展开摘要 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1001` |
| TL-AST-21 | `{子代理摘要}` | 等宽摘要区（展开） | 隐藏 | 点击通知且无跳转目标 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1010`、`:1018` |
| TL-AST-22 | （后台任务通知：终端图标，无文案） | 图标 | 隐藏 | 该块为 exec 后台任务通知 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1025`、`:1036` |
| TL-AST-23 | 命令 | 后台任务标签（默认值） | 隐藏 | 无自定义标签 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1037` |
| TL-AST-24 | `{任务标签}` | 后台任务标签 | 隐藏 | 有自定义标签 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1037` |
| TL-AST-25 | 已完成 / 失败 / 已取消 | 后台任务状态文本 | 隐藏 | 按任务终态切换 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1038` |
| TL-AST-26 | （后台任务展开箭头，展开时旋转） | 图标 | 隐藏 | 有输出路径 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1039` |
| TL-AST-27 | Loading... | 后台任务输出占位 | 隐藏 | 输出加载中 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1053` |
| TL-AST-28 | `{输出读取错误}` | 后台任务错误文本 | 隐藏 | 输出读取失败 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1054` |
| TL-AST-29 | `{后台任务输出}` | 等宽输出区 | 隐藏 | 展开且有内容 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1055` |
| TL-AST-30 | （一般通知：层叠图标，无文案） | 图标 | 隐藏 | 其他通知块 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1067` |
| TL-AST-31 | `{通知文案}` | 行内文本 | 隐藏 | 其他通知块 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1080` |
| TL-AST-32 | `{通知文案}` + › | 可点开行（可访问名「打开: {文案}」） | 隐藏 | 通知带子代理跳转目标 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1068`、`:1072` |
| TL-AST-33 | `{状态文本}` | 状态块文本（错误态警示色） | 显示 | 每个 status 块 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1083` |
| TL-AST-34 | （生成图片块容器，无文案） | 容器 | 隐藏 | 有 generated-image 块 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1096` |
| TL-AST-35 | （生成图片，无文案） | 图片 | 隐藏 | 预览加载成功 | `<ref>/components/chat-workspace/timeline/GeneratedImageResultCard.vue:113` |
| TL-AST-36 | （图片加载中旋转图标，无文案） | 图标 | 隐藏 | 预览加载中 | `<ref>/components/chat-workspace/timeline/GeneratedImageResultCard.vue:126` |
| TL-AST-37 | 无法加载图片预览 / 图片过大，无法预览 / 不支持此图片格式 | 图片占位文案 | 隐藏 | 预览失败 | `<ref>/components/chat-workspace/timeline/GeneratedImageResultCard.vue:128` |
| TL-AST-38 | 重试 | 行内文字按钮 | 隐藏 | 预览失败且非不支持格式 | `<ref>/components/chat-workspace/timeline/GeneratedImageResultCard.vue:129` |
| TL-AST-39 | 打开图片 | 浮动图标按钮（悬停出现） | 隐藏 | 显示操作且悬停卡片 | `<ref>/components/chat-workspace/timeline/GeneratedImageResultCard.vue:137` |
| TL-AST-40 | 另存图片 | 浮动图标按钮（悬停出现） | 隐藏 | 同上 | `<ref>/components/chat-workspace/timeline/GeneratedImageResultCard.vue:147` |
| TL-AST-41 | （Plan 草稿块容器，紫色描边） | 容器 | 隐藏 | 有 plan-draft 块 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1108` |
| TL-AST-42 | Plan | 徽标（清单图标 + 文本） | 隐藏 | 同上 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1115`、`:1117` |
| TL-AST-43 | `{计划元信息}` | 徽标右侧文本 | 隐藏 | 有元信息 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1119`、`:1124` |
| TL-AST-44 | `{计划 Markdown 正文}` | Markdown 渲染 | 隐藏 | 同上 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1127`、`:1133` |
| TL-AST-45 | （MCP 应用内联锚点） | 容器 | 隐藏 | 工具块为 MCP 应用且有会话 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1158`；`<ref>/mcp-apps/McpAppInlineAnchor.vue:93` |
| TL-AST-46 | 展开 | MCP 内联展开按钮（可访问名 + 提示） | 隐藏 | 非全屏展示且可展开 | `<ref>/mcp-apps/McpAppInlineAnchor.vue:105`、`:111`、`:117` |
| TL-AST-47 | （终端行图标，随终态变化） | 图标 | 隐藏 | 该轮有终态 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1170`、`:1187` |
| TL-AST-48 | `{终态行文本}`（运行失败 · {reason} / 已停止 / 用户已中断本次运行。等） | 行文本 | 隐藏 | 有终态 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1194`、`:1200` |
| TL-AST-49 | （终端行展开箭头，展开时旋转） | 图标 | 隐藏 | 终态可展开 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1205` |
| TL-AST-50 | 续订 / 升级 | 描边按钮（外链图标） | 隐藏 | 终态带套餐动作且非只读 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1211`、`:1226` |
| TL-AST-51 | `{终态说明}` | 展开区段落 | 隐藏 | 展开终态 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1236` |
| TL-AST-52 | 原始错误信息 | 调试详情标题 | 隐藏 | 展开终态且有宿主或调试行 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1250` |
| TL-AST-53 | 复制 | 图标按钮（复制/已复制勾，可访问名「复制」） | 隐藏 | 同上 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1253` |
| TL-AST-54 | `{运行时宿主}` | 等宽文本 | 隐藏 | 有运行时宿主 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1265` |
| TL-AST-55 | `{调试字段名}` / `{调试字段值}` | 定义列表项 | 隐藏 | 每条调试行 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1273`、`:1277`、`:1281` |
| TL-AST-56 | `{模型名}` | 脚注文本 | 隐藏 | 该轮有模型 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1291` |
| TL-AST-57 | 复制 | 右键菜单项（复制图标） | 隐藏 | 有选中文本 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1297`、`:1303` |
| TL-AST-58 | 复制整个消息 | 右键菜单项（复制图标） | 隐藏 | 有可复制内容 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1305`、`:1311` |
| TL-AST-59 | （划选浮条，复用 R 组） | 浮层 | 隐藏 | 选中正文文本 | `<ref>/components/chat-workspace/timeline/TimelineAssistantMessage.vue:1315` |

元素数：59

## E. 思考块（TimelineThinkingItem）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TL-THK-01 | （思考行容器，整体可点击） | 行按钮 | 显示 | 每个 thinking 块 | `<ref>/components/chat-workspace/timeline/TimelineThinkingItem.vue:97` |
| TL-THK-02 | （灯泡图标，无文案） | 图标 | 显示 | 同上 | `<ref>/components/chat-workspace/timeline/TimelineThinkingItem.vue:102` |
| TL-THK-03 | 正在思考 | 行标题（流光） | 隐藏 | 思考进行中 | `<ref>/components/chat-workspace/timeline/TimelineThinkingItem.vue:108` |
| TL-THK-04 | 思考了 {duration} | 行标题 | 隐藏 | 思考结束 | `<ref>/components/chat-workspace/timeline/TimelineThinkingItem.vue:111` |
| TL-THK-05 | （展开箭头，展开时旋转） | 图标 | 显示 | 同上 | `<ref>/components/chat-workspace/timeline/TimelineThinkingItem.vue:114` |
| TL-THK-06 | `{思考正文}` | 引用块（左侧竖线） | 隐藏 | 展开后 | `<ref>/components/chat-workspace/timeline/TimelineThinkingItem.vue:123`、`:127` |

元素数：6

## F. 工具调用块（TimelineToolCallItem / ToolCallInlineSummary）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TL-TOL-01 | （工具块容器，可聚焦） | 容器 | 显示 | 每个 tool 块 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:729` |
| TL-TOL-02 | （工具行按钮，整行可点） | 行按钮 | 显示 | 同上；无详情且非读类时不可点 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:737`、`:740` |
| TL-TOL-03 | （动作图标，随工具类型变化） | 图标 | 显示 | 同上 | `<ref>/components/chat-workspace/timeline/ToolCallInlineSummary.vue:105`、`:111` |
| TL-TOL-04 | 读取 / 缓存内容 / 写入 / 编辑 / 搜索网络 / 执行 / 后台执行 / 列出文件 / 搜索内容 / 查找 / 获取网页 / 更新任务 / 创建智能体 / 查询智能体状态 / 取消智能体任务 / 查询可用模型 / 查询外部智能体模型 / 询问用户 / 使用技能 / 生成图片 / 浏览器 / 浏览器控制 / 等待 / 操作电脑 / 使用工具 | 工具标签 | 显示 | 按工具类型取其一（共 29 条含插值变体） | `<ref>/components/chat-workspace/timeline/ToolCallInlineSummary.vue:121`；文案 `<ref>/i18n/zh-CN.ts:1599`–`:1646` |
| TL-TOL-05 | `{文件名 / 行区间}` | 内联主参数文本（含「读取 {file} 第{start}到{end}行」等插值） | 隐藏 | 工具有主参数 | `<ref>/components/chat-workspace/timeline/ToolCallInlineSummary.vue:165`、`:171` |
| TL-TOL-06 | （目标文件图标，无文案） | 图标 | 隐藏 | file 类工具有路径 | `<ref>/components/chat-workspace/timeline/ToolCallInlineSummary.vue:122` |
| TL-TOL-07 | （站点 favicon，无文案） | 图标 | 隐藏 | web 抓取有站点 | `<ref>/components/chat-workspace/timeline/ToolCallInlineSummary.vue:129` |
| TL-TOL-08 | `{智能体名}` | 内联 agent chip（供应商图标 + 名称） | 隐藏 | 工具由子代理发起且非独立窗口 | `<ref>/components/chat-workspace/timeline/ToolCallInlineSummary.vue:135`、`:148` |
| TL-TOL-09 | `{智能体模型标签}` | agent chip 提示浮层 | 隐藏 | 悬停 agent chip | `<ref>/components/chat-workspace/timeline/ToolCallInlineSummary.vue:151` |
| TL-TOL-10 | `{智能体名}` | 纯文本 | 隐藏 | 独立窗口内 | `<ref>/components/chat-workspace/timeline/ToolCallInlineSummary.vue:134` |
| TL-TOL-11 | `{外部 agent 类型}` | 供应商图标 + 文本 | 隐藏 | 有外部 agent 类型且无 agent 信息 | `<ref>/components/chat-workspace/timeline/ToolCallInlineSummary.vue:155`、`:163` |
| TL-TOL-12 | 排队中 | 行内状态文本 | 隐藏 | 工具状态为 queued | `<ref>/components/chat-workspace/timeline/ToolCallInlineSummary.vue:173`、`:177` |
| TL-TOL-13 | `+{新增} -{删除}` | diff 统计（滚动数值） | 隐藏 | 工具有 diff 统计 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:753`、`:758` |
| TL-TOL-14 | （搜索站点 favicon 组，无文案） | 图标组（80ms 递进淡入） | 隐藏 | web 搜索且结果站点 > 0 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:762`、`:774` |
| TL-TOL-15 | (0/1) | 结果计数占位（不可见，撑宽用） | 隐藏 | 同上 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:782` |
| TL-TOL-16 | +{超出条数} | 计数文本 | 隐藏 | 搜索站点 > 3 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:783`、`:786` |
| TL-TOL-17 | ✗ | 失败标记 | 隐藏 | web 工具出错 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:792`、`:796` |
| TL-TOL-18 | （展开箭头，展开时旋转 90°） | 图标 | 隐藏 | 工具有详情 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:797` |
| TL-TOL-19 | 输入 | 面板分区标题（吸顶） | 隐藏 | 面板展开且有输入 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:815` |
| TL-TOL-20 | 输出 | 面板分区标题（吸顶） | 隐藏 | 面板展开且有输出 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:822` |
| TL-TOL-21 | $ `{命令}` | exec 面板命令行 | 隐藏 | command 类工具展开 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:817` |
| TL-TOL-22 | cwd: `{工作目录}` | exec 面板工作目录 | 隐藏 | 有 cwd | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:818` |
| TL-TOL-23 | exit code: `{code}` · signal: `{signal}` | exec 面板退出信息 | 隐藏 | 输出为错误态 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:823`、`:824` |
| TL-TOL-24 | `{输出文本}` | exec 面板等宽输出 | 隐藏 | 有输出 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:826` |
| TL-TOL-25 | `{查询标签}`: `{值}` | search 面板查询行 | 隐藏 | search 类工具展开 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:837` |
| TL-TOL-26 | path: `{路径}` | search 面板路径 | 隐藏 | 有次级参数 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:838` |
| TL-TOL-27 | `{搜索结果}` | search 面板等宽输出 | 隐藏 | 有输出 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:844` |
| TL-TOL-28 | `{文件路径}` | file 面板路径（编辑类可点击） | 隐藏 | file 类工具展开 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:852`、`:859` |
| TL-TOL-29 | （代码块 / diff 块） | 代码区 | 隐藏 | 有文件内容或 diff | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:860`；`<ref>/components/chat-workspace/timeline/ToolCodeBlock.vue:180` |
| TL-TOL-30 | `{文件错误输出}` | file 面板等宽输出 | 隐藏 | 有错误文本 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:870`、`:873` |
| TL-TOL-31 | （结果行站点 favicon，无文案） | 图标 | 隐藏 | web search 每条结果 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:890` |
| TL-TOL-32 | `{结果标题}` | 外链文本 | 隐藏 | 同上 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:895`、`:901` |
| TL-TOL-33 | `{hostname}` · `{发布日期}` | 结果元信息 | 隐藏 | 同上 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:903`、`:904` |
| TL-TOL-34 | `{结果摘要}` | 两行截断文本 | 隐藏 | 有 snippet | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:906` |
| TL-TOL-35 | （fetch 头 favicon + 链接，无文案标签） | 头部行 | 隐藏 | web fetch 面板 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:911`、`:917` |
| TL-TOL-36 | `{抓取错误消息}` | 错误文本（警示色） | 隐藏 | 抓取失败 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:926` |
| TL-TOL-37 | `{抓取正文}` | 等宽正文 | 隐藏 | 抓取成功 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:929` |
| TL-TOL-38 | `{问题分组标题}` | ask-user 面板文本 | 隐藏 | ask-user 面板每项 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:944` |
| TL-TOL-39 | `{问题正文}` | ask-user 面板文本 | 隐藏 | 同上 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:945` |
| TL-TOL-40 | `{作答内容}`（自定义答案前置 ✎） | ask-user 面板文本 | 隐藏 | 已作答 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:946`、`:949` |
| TL-TOL-41 | 未作答（{duration}） | ask-user 面板尾行 | 隐藏 | 询问超时 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:951`、`:956` |
| TL-TOL-42 | （待办状态图标：已完成/进行中/未开始） | 图标 | 隐藏 | todos 快照每行 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:971`–`:982` |
| TL-TOL-43 | `{待办内容}` | 待办行文本（已完成加删除线） | 隐藏 | 同上 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:983`、`:986` |
| TL-TOL-44 | `{优先级}` | 优先级色点（原生提示） | 隐藏 | 同上 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:987`、`:990` |
| TL-TOL-45 | `{待办文本行}` | 纯文本行 | 隐藏 | 非快照形式的待办输出 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:994`、`:999` |
| TL-TOL-46 | `{技能描述}` | skill 面板文本 | 隐藏 | skill 类工具且有描述 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:1008` |
| TL-TOL-47 | `{提示词}` | agent 创建面板等宽文本 | 隐藏 | agent create 且有 prompt | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:1017`、`:1018` |
| TL-TOL-48 | `{智能体名}` / `{任务标题}` / `{状态}` | agent 列表面板行（供应商图标） | 隐藏 | agent list 有结果 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:1021`、`:1042`–`:1044` |
| TL-TOL-49 | `{模型 ID}` | agent 模型面板行（供应商图标） | 隐藏 | query_provider_model 有结果 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:1048`、`:1059` |
| TL-TOL-50 | `{外部 agent 类型}` | 面板小标题（供应商图标） | 隐藏 | query_external_model | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:1063`、`:1070` |
| TL-TOL-51 | `{模型名}` + (default) | 外部模型行 | 隐藏 | 同上 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:1072`、`:1078` |
| TL-TOL-52 | `{浏览器面板输入}` / `{浏览器面板输出}` | browser 面板等宽块 | 隐藏 | browser 类工具展开 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:1093`、`:1098` |
| TL-TOL-53 | （CUA 截图组，无文案） | 图片组（最多 3 张） | 隐藏 | cua 面板有截图 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:1109`、`:1110` |
| TL-TOL-54 | +{剩余张数} | 计数文本 | 隐藏 | 截图 > 3 张 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:1115`、`:1116` |
| TL-TOL-55 | `{CUA 输入}` / `{CUA 输出}` | cua 面板等宽块 | 隐藏 | cua 类工具展开 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:1119`、`:1124` |
| TL-TOL-56 | `{原始输入}` / `{原始输出}` | raw 面板等宽块 | 隐藏 | 无专用面板时兜底 | `<ref>/components/chat-workspace/timeline/TimelineToolCallItem.vue:1129`、`:1134`、`:1139` |

元素数：56

## G. 工具块分组（TimelineBlockGroup）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TL-GRP-01 | （分组行按钮，整行可点） | 行按钮 | 显示 | 连续工作块被折叠成组 | `<ref>/components/chat-workspace/timeline/TimelineBlockGroup.vue:350` |
| TL-GRP-02 | （工作流图标，无文案） | 图标 | 显示 | 同上 | `<ref>/components/chat-workspace/timeline/TimelineBlockGroup.vue:356` |
| TL-GRP-03 | 思考{count}轮 | 汇总项 | 隐藏 | 组内思考块 > 0 | `<ref>/components/chat-workspace/timeline/TimelineBlockGroup.vue:365`；文案 `<ref>/i18n/zh-CN.ts:1313` |
| TL-GRP-04 | （汇总项分隔符：` · ` 与 `、`，无独立文案） | 分隔文本 | 隐藏 | 有两个以上汇总项 | `<ref>/components/chat-workspace/timeline/TimelineBlockGroup.vue:365`、`:190` |
| TL-GRP-05 | 读{count}次文件 / 改{count}次文件 / 执行{count}次命令 / 搜{count}次 / 网络搜{count}次 / 抓取{count}次 / 调{count}次子代理 / 浏览器操作{count}次 / 操作电脑{count}次 / 等待{count}次 / 使用{count}次技能 / 其他{count}次 | 分类汇总项（按类别顺序拼接） | 隐藏 | 该类工具块 > 0 | `<ref>/components/chat-workspace/timeline/TimelineBlockGroup.vue:162`、`:184`；文案 `<ref>/i18n/zh-CN.ts:1315`–`:1327` |
| TL-GRP-06 | （展开箭头，展开时旋转 90°） | 图标 | 显示 | 分组行存在 | `<ref>/components/chat-workspace/timeline/TimelineBlockGroup.vue:377` |
| TL-GRP-07 | （分组内容轨：左侧竖线） | 容器 | 隐藏 | 展开后 | `<ref>/components/chat-workspace/timeline/TimelineBlockGroup.vue:383`、`:390` |
| TL-GRP-08 | （轨内子块：思考 / 工具 / 通知，逐块 80ms 递进淡入） | 子块列表 | 隐藏 | 展开后 | `<ref>/components/chat-workspace/timeline/TimelineBlockGroup.vue:396`、`:399`、`:409` |

元素数：8

## H. 压缩分隔块（TimelineCompressionDivider）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TL-CMP-01 | （压缩分隔行：左横线 + 图标 + 文案 + 右横线） | 分隔块 | 显示 | 每个压缩记录 | `<ref>/components/chat-workspace/timeline/TimelineCompressionDivider.vue:115` |
| TL-CMP-02 | 正在压缩上下文 | 分隔文案（流光） | 隐藏 | 压缩进行中 | `<ref>/components/chat-workspace/timeline/TimelineCompressionDivider.vue:64`、`:155` |
| TL-CMP-03 | 上下文已压缩 | 分隔文案（归档图标） | 显示 | 手动压缩完成 | `<ref>/components/chat-workspace/timeline/TimelineCompressionDivider.vue:78` |
| TL-CMP-04 | 上下文已自动压缩 | 分隔文案（归档图标） | 隐藏 | 自动压缩完成 | `<ref>/components/chat-workspace/timeline/TimelineCompressionDivider.vue:77` |
| TL-CMP-05 | 上下文压缩失败 | 分隔文案（警示图标） | 隐藏 | 压缩失败 | `<ref>/components/chat-workspace/timeline/TimelineCompressionDivider.vue:67` |
| TL-CMP-06 | 上下文压缩已跳过 | 分隔文案 | 隐藏 | 压缩被跳过 | `<ref>/components/chat-workspace/timeline/TimelineCompressionDivider.vue:70` |
| TL-CMP-07 | 上下文压缩已取消 | 分隔文案 | 隐藏 | 压缩被取消 | `<ref>/components/chat-workspace/timeline/TimelineCompressionDivider.vue:73` |
| TL-CMP-08 | (部分) | 文案后缀 | 隐藏 | 覆盖范围为部分近期 | `<ref>/components/chat-workspace/timeline/TimelineCompressionDivider.vue:75` |
| TL-CMP-09 | （展开箭头，展开时旋转 90°） | 图标 | 隐藏 | 有失败详情或摘要可读 | `<ref>/components/chat-workspace/timeline/TimelineCompressionDivider.vue:135` |
| TL-CMP-10 | `{失败码}` | 摘要面板等宽文本 | 隐藏 | 展开失败详情且有失败码 | `<ref>/components/chat-workspace/timeline/TimelineCompressionDivider.vue:176`、`:181` |
| TL-CMP-11 | `{失败消息}` | 摘要面板文本 | 隐藏 | 展开失败详情 | `<ref>/components/chat-workspace/timeline/TimelineCompressionDivider.vue:183`、`:184` |
| TL-CMP-12 | 正在加载摘要… | 摘要面板状态 | 隐藏 | 摘要加载中 | `<ref>/components/chat-workspace/timeline/TimelineCompressionDivider.vue:187`、`:192` |
| TL-CMP-13 | `{摘要正文}` | 摘要面板文本 | 隐藏 | 摘要可用 | `<ref>/components/chat-workspace/timeline/TimelineCompressionDivider.vue:194`、`:199` |
| TL-CMP-14 | 这条旧压缩记录没有保存摘要。 | 摘要面板状态 | 隐藏 | 摘要不可用 | `<ref>/components/chat-workspace/timeline/TimelineCompressionDivider.vue:201`、`:206` |
| TL-CMP-15 | 无法加载摘要。 | 摘要面板状态 | 隐藏 | 摘要加载失败 | `<ref>/components/chat-workspace/timeline/TimelineCompressionDivider.vue:213`、`:214` |
| TL-CMP-16 | 重试 | 描边按钮 | 隐藏 | 摘要加载失败 | `<ref>/components/chat-workspace/timeline/TimelineCompressionDivider.vue:216`、`:217` |

元素数：16

## I. 窗口截断分隔（TimelineWindowTruncationDivider）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TL-TRC-01 | （左右横线，无文案） | 分隔线 | 显示 | 出现截断行 | `<ref>/components/chat-workspace/timeline/TimelineWindowTruncationDivider.vue:11`、`:13` |
| TL-TRC-02 | Part of this response cannot be rendered safely in this bounded view. | 分隔文案（硬编码英文，无 i18n） | 显示 | 同上 | `<ref>/components/chat-workspace/timeline/TimelineWindowTruncationDivider.vue:12` |

元素数：2

## J. 重试提示块（TimelineRetryNotice）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TL-RTY-01 | （警示图标 + 错误分类文案） | 提示行 | 显示 | 每次供应商重试 | `<ref>/components/chat-workspace/timeline/TimelineRetryNotice.vue:44`、`:46` |
| TL-RTY-02 | {seconds} 秒后重试 | 倒计时按钮（可点击立即重试） | 显示 | 运行中、等待中、可重试 | `<ref>/components/chat-workspace/timeline/TimelineRetryNotice.vue:39`、`:47` |
| TL-RTY-03 | 正在重试 | 倒计时按钮文本 | 隐藏 | 倒计时归零 | `<ref>/components/chat-workspace/timeline/TimelineRetryNotice.vue:39` |
| TL-RTY-04 | 已重试 | 状态文本（不可点） | 隐藏 | 已触发重试 | `<ref>/components/chat-workspace/timeline/TimelineRetryNotice.vue:36`、`:48` |
| TL-RTY-05 | 已取消重试 | 状态文本（不可点） | 隐藏 | 已取消或本轮不再运行 | `<ref>/components/chat-workspace/timeline/TimelineRetryNotice.vue:37`、`:48` |

元素数：5

## K. 行渲染错误（TimelineErrorBoundary）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TL-ERR-01 | （警示三角图标，无文案） | 图标 | 隐藏 | 该行渲染抛错 | `<ref>/components/chat-workspace/timeline/TimelineErrorBoundary.vue:38` |
| TL-ERR-02 | 该消息渲染失败。 | 错误行文本 | 隐藏 | 同上 | `<ref>/components/chat-workspace/timeline/TimelineErrorBoundary.vue:39` |

元素数：2

## L. 产物区（TimelineArtifacts / TimelineArtifactsView）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TL-ART-01 | {count} 个产物 | 头部计数文本 | 隐藏 | 本轮有产物 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:30`、`:15` |
| TL-ART-02 | 提交或推送 | 描边按钮（云上传图标 + 下拉箭头） | 隐藏 | 工作区有分支 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:34`、`:42` |
| TL-ART-03 | 提交 | 菜单项（提交图标） | 隐藏 | 菜单打开；运行中/无分支/无产物时禁用 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:47` |
| TL-ART-04 | 提交并推送 | 菜单项（云上传图标） | 隐藏 | 同上 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:54` |
| TL-ART-05 | 推送 | 菜单项（云上传图标） | 隐藏 | 同上 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:61` |
| TL-ART-06 | `{文件名}` | 产物行（文件图标 + 名称） | 隐藏 | 每个产物；文件已删除时整行半透明 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:78`、`:93` |
| TL-ART-07 | 已删除 | 行内状态文本 | 隐藏 | 产物文件不存在 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:96`、`:100` |
| TL-ART-08 | 无法预览文件 | 行内状态文本 | 隐藏 | 产物不可打开 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:103` |
| TL-ART-09 | 打开 | 文字按钮（外链图标） | 隐藏 | 产物存在且可打开 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:105`、`:113` |
| TL-ART-10 | （下拉箭头按钮，无文案） | 图标按钮 | 隐藏 | 本地操作或下载可用 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:116` |
| TL-ART-11 | 下载 | 菜单项（下载图标） | 隐藏 | 下载可用 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:128` |
| TL-ART-12 | 用默认应用打开 | 菜单项（外链图标） | 隐藏 | 本地操作可用 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:131` |
| TL-ART-13 | 在文件管理器中显示 | 菜单项（文件夹打开图标） | 隐藏 | 本地操作可用 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:138` |
| TL-ART-14 | 全部 {count} 个产物 | 展开按钮（右箭头） | 隐藏 | 产物 > 3 且未展开 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:152`、`:160` |
| TL-ART-15 | 收起 | 收起按钮（下箭头） | 隐藏 | 已展开 | `<ref>/components/chat-workspace/timeline/TimelineArtifactsView.vue:159`、`:160` |

元素数：15

## M. 文件改动区（TimelineFileChanges / TimelineFileChangesView）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TL-FCH-01 | {count} 个文件改动 | 摘要按钮（箭头 + 文本） | 隐藏 | 有改动且计数 > 0 | `<ref>/components/chat-workspace/timeline/TimelineFileChangesView.vue:54`、`:62` |
| TL-FCH-02 | {count} 个文件改动（已撤销） | 摘要按钮文本 | 隐藏 | 本轮改动已撤销 | `<ref>/components/chat-workspace/timeline/TimelineFileChangesView.vue:15` |
| TL-FCH-03 | 撤销 | 文字按钮（撤销图标） | 隐藏 | 可撤销 | `<ref>/components/chat-workspace/timeline/TimelineFileChangesView.vue:65`、`:76` |
| TL-FCH-04 | 重新应用 | 文字按钮（重做图标） | 隐藏 | 已撤销且可重做 | `<ref>/components/chat-workspace/timeline/TimelineFileChangesView.vue:79`、`:89` |
| TL-FCH-05 | M / A / D / ? | 状态字母 | 显示 | 每个改动文件行 | `<ref>/components/chat-workspace/timeline/TimelineFileChangesView.vue:111`、`:115` |
| TL-FCH-06 | `{文件路径}` | 文件行文本（超长截断） | 显示 | 同上 | `<ref>/components/chat-workspace/timeline/TimelineFileChangesView.vue:116` |
| TL-FCH-07 | +{insertions}/-{deletions} | 行尾统计 | 显示 | 有行数统计；无统计时为 binary | `<ref>/components/chat-workspace/timeline/TimelineFileChangesView.vue:119`、`:37` |
| TL-FCH-08 | 用默认应用打开 | 右键菜单项（外链图标） | 隐藏 | 本地操作可用 | `<ref>/components/chat-workspace/timeline/TimelineFileChangesView.vue:123` |
| TL-FCH-09 | 在文件管理器中显示 | 右键菜单项（文件夹打开图标） | 隐藏 | 本地操作可用 | `<ref>/components/chat-workspace/timeline/TimelineFileChangesView.vue:130` |
| TL-FCH-10 | （分隔线，无文案） | 分隔线 | 隐藏 | 本地操作可用 | `<ref>/components/chat-workspace/timeline/TimelineFileChangesView.vue:137` |
| TL-FCH-11 | 复制路径 | 右键菜单项（复制图标） | 隐藏 | 有绝对路径 | `<ref>/components/chat-workspace/timeline/TimelineFileChangesView.vue:138` |
| TL-FCH-12 | 复制相对路径 | 右键菜单项（复制图标） | 隐藏 | 文件在工作区内 | `<ref>/components/chat-workspace/timeline/TimelineFileChangesView.vue:145` |
| TL-FCH-13 | 撤销改动 / 重新应用改动 | 确认对话框标题 | 隐藏 | 点击撤销或重新应用 | `<ref>/components/chat-workspace/git/UndoChangesDialog.vue:103` |
| TL-FCH-14 | 撤销本轮的文件改动？ / 重新应用本轮的文件改动？ | 确认对话框问句 | 隐藏 | 同上 | `<ref>/components/chat-workspace/git/UndoChangesDialog.vue:106` |
| TL-FCH-15 | M / A / D + `{路径}` + `{统计}` | 对话框文件行 | 隐藏 | 同上 | `<ref>/components/chat-workspace/git/UndoChangesDialog.vue:110`、`:119`、`:121`、`:123` |
| TL-FCH-16 | +{count} 更多 | 对话框溢出提示 | 隐藏 | 文件数超出列表上限 | `<ref>/components/chat-workspace/git/UndoChangesDialog.vue:128` |
| TL-FCH-17 | 取消 | 按钮 | 隐藏 | 同上 | `<ref>/components/chat-workspace/git/UndoChangesDialog.vue:131` |
| TL-FCH-18 | 撤销改动 / 重新应用 | 主按钮（撤销为危险色，含旋转图标） | 隐藏 | 同上 | `<ref>/components/chat-workspace/git/UndoChangesDialog.vue:134`、`:141` |

元素数：18

## N. 助手操作行（TimelineAssistantActions）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TL-ACT-01 | 复制整个消息 | 图标按钮（复制/已复制勾） | 隐藏 | 有可复制内容；默认悬停可见 | `<ref>/components/chat-workspace/timeline/TimelineAssistantActions.vue:139`、`:152` |
| TL-ACT-02 | 从此处分叉 | 图标按钮（分叉） | 隐藏 | 可分叉且非独立窗口 | `<ref>/components/chat-workspace/timeline/TimelineAssistantActions.vue:154`、`:167` |
| TL-ACT-03 | 重试 | 图标按钮（回转箭头） | 隐藏 | 可重试 | `<ref>/components/chat-workspace/timeline/TimelineAssistantActions.vue:169`、`:182` |
| TL-ACT-04 | 重新加载 | 重试按钮提示 | 隐藏 | 右栏「最新一轮」复用本组件时 | `<ref>/components/chat-workspace/timeline/LatestTurnPreview.vue:130` |
| TL-ACT-05 | `{完成时间}` | 时间文本 | 隐藏 | 有完成时间 | `<ref>/components/chat-workspace/timeline/TimelineAssistantActions.vue:184`、`:189` |

元素数：5

## O. 用户消息轨（TimelineUserMessageRail）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TL-RAIL-01 | （刻度按钮，宽度按长度变化，无文案） | 刻度 | 显示 | 每条用户消息一个；可访问名「Jump to user message: {预览}」 | `<ref>/components/chat-workspace/timeline/TimelineUserMessageRail.vue:679`、`:687` |
| TL-RAIL-02 | `{用户消息预览}` | 悬浮卡首行 | 隐藏 | 悬停/聚焦刻度 | `<ref>/components/chat-workspace/timeline/TimelineUserMessageRail.vue:696`、`:702` |
| TL-RAIL-03 | `{助手回复预览}` | 悬浮卡次行 | 隐藏 | 该条有助手回复 | `<ref>/components/chat-workspace/timeline/TimelineUserMessageRail.vue:705`、`:709` |
| TL-RAIL-04 | `{变更类型}` + `{文件名}` | 悬浮卡改动 chip（最多 4 条） | 隐藏 | 该条有改动路径 | `<ref>/components/chat-workspace/timeline/TimelineUserMessageRail.vue:723`、`:733`、`:736` |
| TL-RAIL-05 | No file changes | 悬浮卡空态（硬编码英文） | 隐藏 | 该条改动为 0 | `<ref>/components/chat-workspace/timeline/TimelineUserMessageRail.vue:714`、`:717` |
| TL-RAIL-06 | +{剩余数} | 悬浮卡溢出计数 | 隐藏 | 改动 > 4 条 | `<ref>/components/chat-workspace/timeline/TimelineUserMessageRail.vue:739`、`:743` |
| TL-RAIL-07 | User message timeline, {n} messages | 密集模式命中区可访问名（硬编码英文） | 显示 | 刻度密集时启用 | `<ref>/components/chat-workspace/timeline/TimelineUserMessageRail.vue:600`、`:604` |

元素数：7

## P. 搜索命中高亮（TimelineSearchHighlightedText / TimelineSearchDomText）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TL-SRC-01 | （命中高亮底，无文案） | 文本高亮 | 隐藏 | 会话内搜索有命中 | `<ref>/components/chat-workspace/timeline/TimelineSearchHighlightedText.vue:1`；`<ref>/components/chat-workspace/timeline/TimelineSearchDomText.vue:1` |
| TL-SRC-02 | （当前命中样式，无文案） | 高亮态 | 隐藏 | 命中为当前定位项 | `<ref>/components/chat-workspace/timeline/TimelineSearchHighlightedText.vue:1` |

元素数：2

## Q. 划选浮条（SelectionPopover）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TL-SEL-01 | 加入对话 | 胶囊按钮（引用图标） | 显示 | 在消息内划选文本 | `<ref>/components/selection/SelectionPopover.vue:113`、`:115` |
| TL-SEL-02 | 标注问题 | 胶囊按钮（星芒图标） | 显示 | 同上 | `<ref>/components/selection/SelectionPopover.vue:117`、`:119` |
| TL-SEL-03 | 添加可选评论… | 多行输入框（占位） | 隐藏 | 展开标注模式 | `<ref>/components/selection/SelectionPopover.vue:129`、`:132` |
| TL-SEL-04 | 提交 | 次级按钮（引用图标） | 隐藏 | 展开标注模式 | `<ref>/components/selection/SelectionPopover.vue:136`、`:144` |
| TL-SEL-05 | 发送失败。请重试，或将其添加到输入框。 | 错误文本 | 隐藏 | 提交失败 | `<ref>/components/selection/SelectionPopover.vue:147`、`:149` |

元素数：5

## R. 右栏「最新一轮」预览（LatestTurnPreview，跨区）

| 元素 ID | 名称（文案原文） | 控件 | 默认 | 条件 | 定位 |
|---|---|---|---|---|---|
| TL-LTP-01 | 最新一轮 | 折叠头按钮（下箭头） | 显示 | 右栏展示最新一轮 | `<ref>/components/chat-workspace/timeline/LatestTurnPreview.vue:100`、`:107` |
| TL-LTP-02 | 正在处理 | 折叠头按钮文本 | 隐藏 | 该轮运行中 | `<ref>/components/chat-workspace/timeline/LatestTurnPreview.vue:107` |
| TL-LTP-03 | （内容区，复用一轮行容器与全部块） | 容器 | 显示 | 展开态（默认展开） | `<ref>/components/chat-workspace/timeline/LatestTurnPreview.vue:111` |

元素数：3

**时间线小计：11 + 7 + 31 + 59 + 6 + 56 + 8 + 16 + 2 + 5 + 2 + 15 + 18 + 5 + 7 + 2 + 5 + 3 = 258**

> 本轮自检（时间线）：逐项过了 ①外壳五态（空会话 / 历史加载 / 运行活动带时长 / 会话恢复遮罩 / 回到底部）；②轮容器：用户块 + 助手块 + 折叠脚区（产物 / 文件改动 / 操作行）；③用户消息：附件轨与片架、引用列表、mark 标注四字段、正文五种内联 chip、折叠展开两态、编辑态（取消/发送）、操作行六按钮、右键三项、划选浮条、已发送长文本对话框；④助手消息：工作摘要 9 种文案组合、文本/媒体块、子代理通知 6 项、后台任务通知 8 项（含 Loading... 与错误）、一般通知两态、状态块、生成图片卡 6 项、Plan 草稿 4 项、MCP 内联锚点、终端行 12 项（含「续订 / 升级」与原始错误信息详情）、模型脚注、右键两项；⑤思考块 6 项；⑥工具块：行内 18 项 + 16 类工具面板（exec / search / file / web search / web fetch / ask-user / todos 两形态 / skill / agent 四形态 / browser / cua / raw）；⑦分组行 8 项（13 类分类汇总文案）；⑧压缩分隔 16 项（六态文案 + 部分后缀 + 摘要面板五态 + 重试）；⑨窗口截断、重试提示四态、行渲染错误；⑩产物区 15 项、文件改动区 18 项（含撤销/重新应用对话框）；⑪助手操作行 5 项；⑫消息轨 7 项（含两处硬编码英文）；⑬搜索高亮、划选浮条 5 项、右栏最新一轮 3 项。

## 计数与颗粒度说明

| 区域 | 分组数 | 元素数 |
|---|---|---|
| 左栏 | 9（A–I） | 134 |
| 左栏附录（跨区对话框，与 `inventory-dialogs` 互为索引） | 1（A-1） | 17 |
| 输入区 | 10（A–J） | 184 |
| 标题栏 | 7（A–G） | 77 |
| 时间线 | 18（A–R） | 258 |
| 状态条 | 5（A–E） | 59 |

五区合计 729 个元素行（不含左栏附录为 712）。按 A 方案（颗粒度保持现状，状态变体保留独立行）计。

与预估的偏差及原因：左栏 134（预估 40–80）、输入区 184（60–120）、标题栏 77（10–20）、时间线 258（80–150）、状态条 59（10–20）。偏差来自三点：①「同一元素不同文案各占一行」使状态型元素展开（更新按钮 5 态、Shell 面板 5 态、工作摘要 9 态、压缩分隔 8 态、工具标签 29 条）；②子组件被完整展开而非只记入口（模型选择器 28 项、活动条 45 项、打开方式组 29 项、工具面板 15 类、消息轨 7 项）；③跨区附属面板按区域就近挂载（左栏 A-1 对话框、状态条 E 新建分支对话框、时间线 R 右栏预览），未单独成区。

已知与参考产品不一致/待产品决策处（清单如实记录原文）：
- `TimelineWindowTruncationDivider` 文案与 `TimelineUserMessageRail` 的「No file changes」「User message timeline, N messages」为硬编码英文，未走 i18n。
- 时间线终端行残留 `Loading...` 英文字面量（后台任务输出加载态）。
- 工具面板里「输入」「输出」两处分区标题在模板内硬编码中文，未走 i18n。
- 输入区 `inputArea.reasoning.*` 与工具标签存在大量插值变体，实现时需按同一 key 归一。
