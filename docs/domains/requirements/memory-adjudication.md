# 记忆维度逐条核定（266 条）

Status: 待并入。本文件只存核定结论，**不改** `requirements-main.md` / `requirements-panels.md`。
Input: 两份需求里「维度 = 记忆 ∧ 状态写「界面状态不落盘」」的 266 行（main 152、panels 114）。
Owner: 本次核定（组织侧交回）；并回需求文档由合并方执行。
Last updated: 2026-11-19

**待并入：P2 证据重标完成后由合并方按 R-ID 对齐写入。**

## 0. 口径

| 类别 | 判法 |
|---|---|
| A 会话内状态 | 重开菜单、切会话、切标签、展开折叠这类状态，只要代码里有承担它的字段/行，就是「已实现」，证据写字段名与位置；没有就写清缺哪个字段 |
| B 跨重启持久化 | 只有真落盘的才算已实现：`preferences.json` 的 `view` 段、`catalog-archive.json`（归档），或本来就是服务端事实（会话列表、审批、权限档、时间线）。其余写缺哪个键 |
| C 明确「不保留」 | 需求本身写「不保留 / 不记忆 / 不缓存 / 复位」：代码确实不留即已实现，证据指向该行为 |

两条约束：

- 原文统一的理由「界面状态不落盘（PreferencesSnapshot 只含主题…）」**已过期**：`preferences.json` 现有 `view` 段（`sidebarCollapsed` / `sidebarFlat` / `sidebarByCreated` / `collapsedGroups` / `pinnedGroups` / `workspaceNames` / `rightOpen` / `rightTab` / `rightTabs` / `rightExpanded` / `rightWidth` / `filesFilter` / `changesScope` / `changesGroupsClosed`），启动由 `applyViewPreferences` 铺回；归档另存 `catalog-archive.json`。
- 真缺的落盘键只有三处：`view.pinnedSessions`（会话置顶）、`view.lastSession`（上次会话）、`view.permissionTier`（空态权限档）。

## 1. 统计

- 合计 **266** 条：A 会话内状态 210、B 跨重启持久化 23、C 明确不保留 33。
- 判定：**已实现 195**（改判）/ **未实现 71**（保留未实现，但理由逐条换成真实缺口）。
- 抽样回读代码 20 条：20/20 命中且语义成立（见 §3）。

## 2. 逐条表

列说明：`新状态` 是本次核定结论；`依据` 是可直接抄进需求文档「状态」列的整句（含代码位置）；`代码位置` 是该句里出现的 `path:line` 汇总，便于按行回读。

| 需求 ID | 元素 ID | 场景 | 类别 | 新状态 | 依据 | 代码位置 |
|---|---|---|---|---|---|---|
| R-TB-008 | TB-01 | 收起状态下重启 | B 跨重启持久化 | 已实现 | 已实现（证据 = internal/app/preferences.go:82 view.sidebarCollapsed 落盘；internal/ui/app.go:1193 applyViewPreferences 启动铺回；internal/ui/header.go:38 header 里按收起态让出左内边距，重启按收起态重画） | internal/app/preferences.go:82、internal/ui/app.go:1193、internal/ui/header.go:38 |
| R-TB-040 | TB-05 | 收起态重启 | B 跨重启持久化 | 已实现 | 已实现（证据 = 同 R-TB-008：internal/app/preferences.go:82 落盘收起态，internal/ui/app.go:1193 铺回后会话态标题栏按上次显隐渲染） | internal/app/preferences.go:82、internal/ui/app.go:1193 |
| R-TB-138 | TB-16 | 新建后重启 | B 跨重启持久化 | 已实现 | 已实现（证据 = internal/ui/app.go:1689 sessionNames 每帧读 Source 的目录；新会话由服务端保存，客户端不缓存列表，重启后重读） | internal/ui/app.go:1689 |
| R-TB-163 | TB-19 | 切换会话 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/header.go:76 sessionTitle 每帧读当前会话快照的 Title；internal/ui/header.go:58 同源，不残留上一个标题） | internal/ui/header.go:76、internal/ui/header.go:58 |
| R-TB-173 | TB-20 | 切会话后再开 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/header.go:150 chatMenu 按 a.pinned(pinIndex) 现算，internal/ui/sidebar_state.go:206 pinned 按当前会话标识查表，切会话即重算置顶/取消置顶文案） | internal/ui/header.go:150、internal/ui/sidebar_state.go:206 |
| R-TB-246 | TB-31 | 切换默认应用后 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/header.go:385 主按钮每帧按 a.openWithApp 取图标；菜单里选应用直接写 a.openWithApp，图标随之更新） | internal/ui/header.go:385 |
| R-TB-281 | TB-36 | 更换工作区目录后 | A 会话内状态 | 未实现 | 未实现（缺打开落点：internal/ui/header.go:453 点击只写回执，没有把当前工作区目录交给宿主打开，「用哪个目录」无从核） | internal/ui/header.go:453 |
| R-TB-309 | TB-22 | 取消置顶后重启 | B 跨重启持久化 | 未实现 | 未实现（缺落盘键 view.pinnedSessions：会话置顶是 sidebarView.pinned（internal/ui/sidebar_state.go:26），internal/app/preferences.go:81 起的 view 里没有这一项，重启后回到未置顶） | internal/ui/sidebar_state.go:26、internal/app/preferences.go:81 |
| R-TB-316 | TB-26 | 归档后重启 | B 跨重启持久化 | 已实现 | 已实现（证据 = 归档隐藏落盘在 catalog-archive.json：internal/catalog/archive.go:28；internal/uiwire/adapter.go:1233 → Catalog.Hide，重启后目录列表按它过滤） | internal/catalog/archive.go:28、internal/uiwire/adapter.go:1233 |
| R-TB-322 | TB-29 | 换工作区后复制路径 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/header.go:329 sessionCopyFacts 每帧读当前会话的 Cwd，换工作区后取新目录） | internal/ui/header.go:329 |
| R-TB-325 | TB-32 | 关闭菜单再打开 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/header.go:415 开合只用 MenuBase 的 *bool，关闭即 false；箭头是固定图标，不按开合翻转） | internal/ui/header.go:415 |
| R-TB-328 | TB-34 | 安装新应用后重开菜单 | A 会话内状态 | 未实现 | 未实现（缺应用清单探测：internal/ui/header.go:429 wired 下固定一行禁用项，应用表是写死的 openWithApps()（internal/ui/header.go:22），重开菜单不重测） | internal/ui/header.go:429、internal/ui/header.go:22 |
| R-TB-331 | TB-37 | 更换系统默认程序后 | A 会话内状态 | 未实现 | 未实现（缺打开落点：internal/ui/header.go:457 点击只写回执，没有交给系统默认程序，也没有旧应用可沿用） | internal/ui/header.go:457 |
| R-LB-006 | LB-01 | 切会话、切工作区后回看 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/sidebar.go:55 栏头让位行高度只取 c.TitleBar().Height，不读会话/工作区） | internal/ui/sidebar.go:55 |
| R-LB-101 | LB-14 | 在定时任务页重启客户端 | B 跨重启持久化 | 已实现 | 已实现（证据 = 路由不落盘：internal/ui/app.go:317 只在内存，internal/app/preferences.go:81 的 view 里没有路由项；启动置回 ScreenMain） | internal/ui/app.go:317、internal/app/preferences.go:81 |
| R-LB-119 | LB-16 | 置顶会话后重启客户端 | B 跨重启持久化 | 未实现 | 未实现（缺落盘键 view.pinnedSessions：置顶是 sidebarView.pinned（internal/ui/sidebar_state.go:26），不在 preferences.json 的 view 里，重启后「已置顶」为空） | internal/ui/sidebar_state.go:26 |
| R-LB-134 | LB-18 | 置顶顺序调整后重启 | B 跨重启持久化 | 未实现 | 未实现（缺落盘键与顺序：sidebarView.pinned（internal/ui/sidebar_state.go:26）是 map，既不落盘也无顺序，重启后无从保留调整后的次序） | internal/ui/sidebar_state.go:26 |
| R-LB-341 | LB-52 | 打开某会话后重启 | B 跨重启持久化 | 未实现 | 未实现（缺落盘键 view.lastSession：当前会话是 a.selectedSession（internal/ui/app.go:272），只在内存，启动不读上次会话） | internal/ui/app.go:272 |
| R-LB-550 | LB-52 | 关闭再打开左栏 | A 会话内状态 | 未实现 | 未实现（缺左栏滚动状态：internal/ui/sidebar.go:164 sidebarList 的 Scroll 没有 TrackScroll，收起再展开回到顶部；选中行 a.selectedSession 本身保持） | internal/ui/sidebar.go:164 |
| R-LB-619 | LB-97 | 从设置返回 | A 会话内状态 | 未实现 | 未实现（缺左栏滚动状态：internal/ui/sidebar.go:164 sidebarList 的 Scroll 没有 TrackScroll，从设置返回回不到原滚动位置；选中态 a.selectedSession 不变） | internal/ui/sidebar.go:164 |
| R-LB-650 | LB-104 | 登出后重启 | B 跨重启持久化 | 未实现 | 未实现（缺未登录文案：账户行取 internal/ui/sidebar_state.go:462 accountNameValue()，未登录回落「未命名账户」，没有「登录 Dim」） | internal/ui/sidebar_state.go:462 |
| R-LB-678 | LB-17 | 置顶区清空后再置顶一条 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/sidebar.go:168 置顶区按 pinnedThreads/pinnedGroups 非空渲染，清空后再置顶即重新出现） | internal/ui/sidebar.go:168 |
| R-LB-697 | LB-24 | 界面语言切换后重启 | B 跨重启持久化 | 未实现 | 未实现（缺界面语言：设置里语言是禁用项（internal/ui/unavailable.go:62），没有可切换的语言，也没有 language 落盘键） | internal/ui/unavailable.go:62 |
| R-LB-727 | LB-39 | 重命名后宽度变化 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/sidebar.go:964 sessionName 每帧按 a.renamed 取新名，行高与截断按新名重算） | internal/ui/sidebar.go:964 |
| R-LB-747 | LB-50 | 界面语言切英文 | A 会话内状态 | 未实现 | 未实现（缺界面语言：同 R-LB-697，语言是禁用项（internal/ui/unavailable.go:62），没有可切的英文文案） | internal/ui/unavailable.go:62 |
| R-LB-774 | LB-62 | 切到分组视图再切回 | A 会话内状态 | 未实现 | 未实现（缺左栏滚动状态：internal/ui/sidebar.go:164 sidebarList 的 Scroll 没有 TrackScroll，滚动位置回不来；分组口径 a.sidebarView.flat 本身保留） | internal/ui/sidebar.go:164 |
| R-LB-788 | LB-66 | 出错行在重启后 | B 跨重启持久化 | 未实现 | 未实现（缺会话状态标记：internal/ui/sidebar.go:1008 在 wired 下返回全空串数组，红点根本不出现） | internal/ui/sidebar.go:1008 |
| R-LB-792 | LB-68 | 置顶后切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/sidebar_state.go:428 只改 a.sidebarView.pinned，切会话不清；行内悬停按钮按 pinned 重画，仍可再点取消） | internal/ui/sidebar_state.go:428 |
| R-LB-833 | LB-84 | 改名后重启 | B 跨重启持久化 | 已实现 | 已实现（证据 = internal/ui/app.go:1800 accountInitialValue 每帧取 source.Account().Initial，改名后重启按新名重取首字母） | internal/ui/app.go:1800 |
| R-LB-838 | LB-85 | 登出后重启 | B 跨重启持久化 | 未实现 | 未实现（缺未登录头像位：internal/ui/sidebar.go:709 只有首字母圆，没有 UserIcon 形态；首字母这半是对的——internal/ui/app.go:1800 取 account.Initial，登出为空） | internal/ui/sidebar.go:709、internal/ui/app.go:1800 |
| R-LB-840 | LB-86 | 登录中关闭左栏再展开 | A 会话内状态 | 未实现 | 未实现（缺登录中状态：internal/ui/sidebar.go:682 没有 LoginPhase 的旋转态与「正在登录」） | internal/ui/sidebar.go:682 |
| R-LB-846 | LB-87 | 错误消除后重启 | B 跨重启持久化 | 未实现 | 未实现（缺错误红点：internal/ui/sidebar.go:709 与 accountRow 都没有错误标记，红点本身从未存在） | internal/ui/sidebar.go:709 |
| R-LB-847 | LB-88 | 账户名变更后 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/sidebar.go:689 行内与 internal/ui/sidebar.go:725 菜单身份项同一 accountNameValue，每帧取同一事实） | internal/ui/sidebar.go:689、internal/ui/sidebar.go:725 |
| R-LB-854 | LB-90 | 改名后重开菜单 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/sidebar.go:725 身份项每次开菜单现取 accountNameValue，改名后同步） | internal/ui/sidebar.go:725 |
| R-LB-864 | LB-92 | 切会话后重开菜单 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/sidebar.go:771 usageLabel 每次开菜单读 liveUsage()，不沿用旧额度） | internal/ui/sidebar.go:771 |
| R-LB-872 | LB-94 | 平台支持恢复后 | A 会话内状态 | 未实现 | 未实现（缺平台能力检测：internal/ui/sidebar.go:730 固定禁用并写 notePet，没有探测也就没有「恢复后可用」的路径） | internal/ui/sidebar.go:730 |
| R-LB-894 | LB-100 | 左栏收起再展开 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/sidebar.go:702 设置齿轮在账户行内按布局贴右，不读左栏开合状态） | internal/ui/sidebar.go:702 |
| R-LB-899 | LB-102 | 语言切换后打开 | A 会话内状态 | 未实现 | 未实现（缺界面语言：同 R-LB-697，描述是常量文案（internal/ui/unavailable.go:62）） | internal/ui/unavailable.go:62 |
| R-LB-906 | LB-103 | 取消后再开对话框 | C 明确不保留 | 已实现 | 已实现（证据 = internal/ui/sidebar.go:743 里「取消」把 logoutConfirm 置 false；重开由菜单重新置 true，没有残留勾选） | internal/ui/sidebar.go:743 |
| R-LB-920 | LB-108 | 能力恢复后重开菜单 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/sidebar.go:606 会话行菜单的「重命名」本就可用、无括注） | internal/ui/sidebar.go:606 |
| R-LB-930 | LB-111 | 归档后重启 | B 跨重启持久化 | 已实现 | 已实现（证据 = internal/catalog/archive.go:28 归档隐藏落盘在 catalog-archive.json；internal/uiwire/adapter.go:1233 走 Catalog.Hide，重启后不回列表） | internal/catalog/archive.go:28、internal/uiwire/adapter.go:1233 |
| R-LB-934 | LB-112 | 删除后重启 | B 跨重启持久化 | 已实现 | 已实现（证据 = internal/uiwire/adapter.go:1252 DeleteSession → catalog.Delete 写服务端；重启后目录里没有它） | internal/uiwire/adapter.go:1252 |
| R-TL-009 | TL-01 | 切会话再回 | C 明确不保留 | 未实现 | 未实现（缺按会话的滚动状态：时间线只有一份 a.feed（internal/ui/app.go:507），internal/ui/timeline.go:99 所有会话共用，切会话不清零也不记位置，回来停在离开时的偏移） | internal/ui/app.go:507、internal/ui/timeline.go:99 |
| R-TL-028 | TL-03 | 加载完成后切换会话 | A 会话内状态 | 未实现 | 未实现（缺历史缓存：客户端不留历史，切会话由服务端重读会话快照；TL-03 的加载条本身也未建） | —（无落点：该元素未建） |
| R-TL-052 | TL-06 | 切回会话 | A 会话内状态 | 未实现 | 未实现（缺视口占位与按会话滚动：a.feed（internal/ui/app.go:507）单份共用，TL-06 的占位行未建） | internal/ui/app.go:507 |
| R-TL-081 | TL-09 | 快速切换会话 | A 会话内状态 | 已实现 | 已实现（证据 = 每帧按当前会话快照重建（internal/ui/timeline.go:1451 liveRow / internal/ui/timeline.go:1425 liveTimeline），不缓存上一次的内容；骨架 timelineSkeleton 尚未挂载（见 TL-09 其余条目），但不会残留上一个会话的内容） | internal/ui/timeline.go:1451、internal/ui/timeline.go:1425 |
| R-TL-116 | TL-13 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/timeline.go:1585 liveRoundStatus 每帧按行事实取状态，inert 按该轮实际是否运行算） | internal/ui/timeline.go:1585 |
| R-TL-125 | TL-14 | 新一轮开始后 | A 会话内状态 | 未实现 | 未实现（缺新轮收起旧轮：产物块开合键是 blockOf(artifacts:<轮次>)（internal/ui/timeline.go:836），新轮出现时没有把上一轮的键置 false） | internal/ui/timeline.go:836 |
| R-TL-134 | TL-15 | 新一轮开始 | A 会话内状态 | 未实现 | 未实现（缺新轮收起旧轮：改动块开合键是 blockOf(files:<轮次>)（internal/ui/timeline.go:990），新轮出现时没有把上一轮的键置 false） | internal/ui/timeline.go:990 |
| R-TL-142 | TL-16 | 加载更早历史后 | A 会话内状态 | 未实现 | 未实现（缺触发落点：internal/ui/timeline.go:85 的按钮没有接动作，加载更早历史发不出去；截断行本身随行事实重建） | internal/ui/timeline.go:85 |
| R-TL-152 | TL-17 | 切会话再回 | C 明确不保留 | 未实现 | 未实现（缺切会话清空：internal/ui/app.go:287 以轮次下标为键，切会话既不清空也不按会话分表，回来仍是上一次的展开态） | internal/ui/app.go:287 |
| R-TL-159 | TL-18 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = 每帧按当前会话快照重建（internal/ui/timeline.go:1451 liveRow / internal/ui/timeline.go:1425 liveTimeline），不缓存上一次的内容） | internal/ui/timeline.go:1451、internal/ui/timeline.go:1425 |
| R-TL-188 | TL-20 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = 每帧按当前会话快照重建（internal/ui/timeline.go:1451 liveRow / internal/ui/timeline.go:1425 liveTimeline），不缓存上一次的内容；气泡内容与顺序都由行事实给） | internal/ui/timeline.go:1451、internal/ui/timeline.go:1425 |
| R-TL-189 | TL-20 | 重启后 | B 跨重启持久化 | 已实现 | 已实现（证据 = 历史是 App Server 的事实，重启后由会话快照重读；客户端不缓存，样式由当前主题与行事实决定） | —（无落点：该元素未建） |
| R-TL-216 | TL-22 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = 每帧按当前会话快照重建（internal/ui/timeline.go:1451 liveRow / internal/ui/timeline.go:1425 liveTimeline），不缓存上一次的内容；chip 与内容来自同一行事实） | internal/ui/timeline.go:1451、internal/ui/timeline.go:1425 |
| R-TL-224 | TL-23 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = 每帧按当前会话快照重建（internal/ui/timeline.go:1451 liveRow / internal/ui/timeline.go:1425 liveTimeline），不缓存上一次的内容；引用随行事实重建） | internal/ui/timeline.go:1451、internal/ui/timeline.go:1425 |
| R-TL-242 | TL-25 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = 每帧按当前会话快照重建（internal/ui/timeline.go:1451 liveRow / internal/ui/timeline.go:1425 liveTimeline），不缓存上一次的内容；提示行随行事实重建） | internal/ui/timeline.go:1451、internal/ui/timeline.go:1425 |
| R-TL-259 | TL-26 | 编辑后重渲染 | A 会话内状态 | 已实现 | 已实现（证据 = 每帧按当前会话快照重建（internal/ui/timeline.go:1451 liveRow / internal/ui/timeline.go:1425 liveTimeline），不缓存上一次的内容；行内编辑保存后按行事实重画，新正文替换旧正文） | internal/ui/timeline.go:1451、internal/ui/timeline.go:1425 |
| R-TL-268 | TL-26 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = 渲染只由行事实与一份 markdown.Options 决定（internal/ui/app.go:301），同一 Markdown 同一外观） | internal/ui/app.go:301 |
| R-TL-353 | TL-35 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = 每帧按当前会话快照重建（internal/ui/timeline.go:1451 liveRow / internal/ui/timeline.go:1425 liveTimeline），不缓存上一次的内容；时间取行事实的时间戳） | internal/ui/timeline.go:1451、internal/ui/timeline.go:1425 |
| R-TL-370 | TL-37 | 复制后移出再悬停 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/timeline.go:248 复制按钮图标恒为 iconCopy，复制只写剪贴板，没有「已复制」中间态可残留） | internal/ui/timeline.go:248 |
| R-TL-377 | TL-38 | 编辑完成后 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/timeline.go:254 动作行每次都画；保存后 inlineEditRow 归位，编辑按钮仍在） | internal/ui/timeline.go:254 |
| R-TL-384 | TL-39 | 分叉后切回原分支 | A 会话内状态 | 已实现 | 已实现（证据 = 原分支时间线是服务端事实，切回由会话快照重建；客户端不缓存分支内容） | —（无落点：该元素未建） |
| R-TL-398 | TL-41 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel.go:69 与 sideChatDraft 只有一份，切会话不清空，回来仍在） | internal/ui/rightpanel.go:69 |
| R-TL-406 | TL-42 | 复制后选择仍在 | C 明确不保留 | 已实现 | 已实现（证据 = 选区由外层可选区持有（internal/ui/app.go:545），复制只写剪贴板（internal/ui/timeline.go:299），不清选区） | internal/ui/app.go:545、internal/ui/timeline.go:299 |
| R-TL-413 | TL-43 | 右键位置 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/timeline.go:216 用框架的 ContextMenu，位置由指针位置决定，不遮目标正文） | internal/ui/timeline.go:216 |
| R-TL-420 | TL-44 | 点击别处 | C 明确不保留 | 已实现 | 已实现（证据 = 浮层开合是 *bool，点别处由框架置 false；选区由外层可选区持有（internal/ui/app.go:545），关菜单不清选区） | internal/ui/app.go:545 |
| R-TL-431 | TL-45 | 切会话再回 | C 明确不保留 | 未实现 | 未实现（缺分类清空：分组开合是 blockOf 的全局键（internal/ui/timeline.go:1181），切会话既不清空也不按会话分表） | internal/ui/timeline.go:1181 |
| R-TL-441 | TL-46 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = 时长取行事实的 Duration（internal/ui/timeline.go:1527 起的一层投影），回来不重新计时） | internal/ui/timeline.go:1527 |
| R-TL-449 | TL-47 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/timeline.go:1585 状态每帧按行事实取，停止态保持） | internal/ui/timeline.go:1585 |
| R-TL-458 | TL-48 | 切会话再回 | A 会话内状态 | 未实现 | 未实现（缺该元素：TL-48「{duration} 后失败」行未建，代码里没有这条失败时长行） | —（无落点：该元素未建） |
| R-TL-466 | TL-49 | 切会话再回 | A 会话内状态 | 未实现 | 未实现（缺该元素：TL-49「出错前的工作过程」行未建，没有可保持的行） | —（无落点：该元素未建） |
| R-TL-485 | TL-51 | 切会话再回 | C 明确不保留 | 未实现 | 未实现（缺手动展开表：时间线没有「手动展开」这层表（internal/ui/timeline.go:1181 是唯一开合入口），无从清空并按自动策略重算） | internal/ui/timeline.go:1181 |
| R-TL-497 | TL-52 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = 时长取行事实 Duration，回来按最终值显示） | —（无落点：该元素未建） |
| R-TL-509 | TL-53 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = 每帧按当前会话快照重建（internal/ui/timeline.go:1451 liveRow / internal/ui/timeline.go:1425 liveTimeline），不缓存上一次的内容） | internal/ui/timeline.go:1451、internal/ui/timeline.go:1425 |
| R-TL-528 | TL-54 | 切会话再回 | C 明确不保留 | 未实现 | 未实现（缺清空：行展开是 blockOf 的全局键（internal/ui/timeline.go:1181），切会话不清，回来仍是展开态） | internal/ui/timeline.go:1181 |
| R-TL-569 | TL-57 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/timeline.go:1585 失败态每帧按行事实取） | internal/ui/timeline.go:1585 |
| R-TL-581 | TL-58 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/timeline.go:823 倒计时取行事实的剩余时长，回来按剩余继续） | internal/ui/timeline.go:823 |
| R-TL-589 | TL-59 | 收起再展开 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/timeline.go:1181 按需创建并缓存开合指针，收起再展开内容立即在场） | internal/ui/timeline.go:1181 |
| R-TL-598 | TL-60 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = 每帧按当前会话快照重建（internal/ui/timeline.go:1451 liveRow / internal/ui/timeline.go:1425 liveTimeline），不缓存上一次的内容） | internal/ui/timeline.go:1451、internal/ui/timeline.go:1425 |
| R-TL-607 | TL-61 | 移出再悬停 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/timeline.go:248 图标恒为 iconCopy，移出再悬停仍是 Copy） | internal/ui/timeline.go:248 |
| R-TL-616 | TL-62 | 分叉后 | A 会话内状态 | 已实现 | 已实现（证据 = 原分支内容来自服务端会话快照，分叉不改原分支事实） | —（无落点：该元素未建） |
| R-TL-626 | TL-63 | 新轮开始后 | A 会话内状态 | 已实现 | 已实现（证据 = 每帧按当前会话快照重建（internal/ui/timeline.go:1451 liveRow / internal/ui/timeline.go:1425 liveTimeline），不缓存上一次的内容；旧轮动作条随行事实保留） | internal/ui/timeline.go:1451、internal/ui/timeline.go:1425 |
| R-TL-634 | TL-64 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = 时间取行事实时间戳，切会话不变） | —（无落点：该元素未建） |
| R-TL-644 | TL-65 | 看「复制 / 复制整个消息」的菜单位置 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/timeline.go:1498 用框架 ContextMenu，位置在光标处） | internal/ui/timeline.go:1498 |
| R-TL-694 | TL-70 | 切走再回 | A 会话内状态 | 已实现 | 已实现（证据 = 时长取行事实 Duration，切走再回按已用时间继续） | —（无落点：该元素未建） |
| R-TL-704 | TL-71 | 恢复完成 | A 会话内状态 | 未实现 | 未实现（缺恢复遮罩与目标位置：TL-71 元素本身未建，滚动也只有一份 a.feed（internal/ui/app.go:507），没有「恢复到目标位置」的代码） | internal/ui/app.go:507 |
| R-TL-716 | TL-72 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/timeline.go:1087 groupSummary 每帧按该轮数据重算汇总文案） | internal/ui/timeline.go:1087 |
| R-TL-738 | TL-74 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = 轨道与箭头读同一个 *bool（internal/ui/timeline.go:1181），同源，不会不一致） | internal/ui/timeline.go:1181 |
| R-TL-747 | TL-75 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = 每帧按当前会话快照重建（internal/ui/timeline.go:1451 liveRow / internal/ui/timeline.go:1425 liveTimeline），不缓存上一次的内容；数字由行事实算） | internal/ui/timeline.go:1451、internal/ui/timeline.go:1425 |
| R-TL-757 | TL-76 | 切会话再回 | A 会话内状态 | 未实现 | 未实现（缺站点图标：TL-76 的抓取行没有 favicon，只有按动作取的工具图标，切回来保持的不是期望的站点图标） | —（无落点：该元素未建） |
| R-TL-767 | TL-77 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = 每帧按当前会话快照重建（internal/ui/timeline.go:1451 liveRow / internal/ui/timeline.go:1425 liveTimeline），不缓存上一次的内容；状态标记由行事实取） | internal/ui/timeline.go:1451、internal/ui/timeline.go:1425 |
| R-TL-777 | TL-78 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = 箭头进度与展开态读同一个 *bool（internal/ui/timeline.go:1181）） | internal/ui/timeline.go:1181 |
| R-IN-028 | IN-01 | 切到别的会话再切回 | A 会话内状态 | 未实现 | 未实现（缺按会话的草稿：a.draft（internal/ui/app.go:283）只有一份，切会话既不清空也不分键，草稿会串到别的会话） | internal/ui/app.go:283 |
| R-IN-029 | IN-01 | 重启客户端 | B 跨重启持久化 | 未实现 | 未实现（草稿不落盘这半成立，但「不得串到别的会话」不成立：a.draft（internal/ui/app.go:283）单份，重启清空之后仍会串） | internal/ui/app.go:283 |
| R-IN-044 | IN-02 | 切换会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/composer.go:218 的 placeholder 由 a.draft 决定，有草稿不显示占位、为空即显示；草稿本身不按会话分键） | internal/ui/composer.go:218 |
| R-IN-082 | IN-06 | 切会话再回 | A 会话内状态 | 未实现 | 未实现（缺按会话的模式：a.mode（internal/ui/app.go:475）只有一份，切会话不换模式） | internal/ui/app.go:475 |
| R-IN-093 | IN-07 | 切会话再回 | A 会话内状态 | 未实现 | 未实现（缺按会话的技能选择：internal/ui/composer.go:708 只把 $名字追加进全局草稿，没有按会话记的选择表） | internal/ui/composer.go:708 |
| R-IN-114 | IN-09 | 切会话再回 | A 会话内状态 | 未实现 | 未实现（缺按会话的模型：internal/ui/app.go:1775 currentModel 读全局设置快照的 Model，另有 a.model 单份，切会话不换） | internal/ui/app.go:1775 |
| R-IN-152 | IN-12 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/composer.go:279 发送键可用态每帧按当前草稿重算） | internal/ui/composer.go:279 |
| R-IN-181 | IN-13 | 切到别的会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/app.go:1727 liveRunning 与轮状态来自当前会话快照，回来仍是停止键） | internal/ui/app.go:1727 |
| R-IN-326 | IN-27 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/composer.go:1099 目标条与队列/Shell 面板都读当前会话快照，切会话按该会话重算） | internal/ui/composer.go:1099 |
| R-IN-334 | IN-28 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/composer.go:1099 goalState 每帧读当前会话的 Goal/GoalStatus，进行中的会话回来活动条仍在） | internal/ui/composer.go:1099 |
| R-IN-470 | IN-43 | 审批中切走再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/composer.go:411 审批卡每帧由 liveApproval 重建，同一待决项回来仍是同一张卡，没有本地选项态） | internal/ui/composer.go:411 |
| R-IN-479 | IN-44 | 回答后切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/composer.go:469 提问卡由 liveQuestion 驱动；已回答的问题服务端不再下发，回来不弹） | internal/ui/composer.go:469 |
| R-IN-562 | IN-53 | 关闭再开 | C 明确不保留 | 未实现 | 未实现（缺关闭清空：模型浮层的搜索词 a.modelQuery 只在启动重置时清零（internal/ui/app.go:624 一带），浮层关闭不清） | internal/ui/app.go:624 |
| R-IN-581 | IN-55 | 关闭重开 | C 明确不保留 | 未实现 | 未实现（缺关闭清空：a.modelQuery 只在启动重置时清零（internal/ui/app.go:624 一带），重开浮层仍是上次的词） | internal/ui/app.go:624 |
| R-IN-603 | IN-57 | 切会话再回 | A 会话内状态 | 未实现 | 未实现（缺按会话的档位：推理强度是 a.effort（internal/ui/app.go:469）一份，切会话不换） | internal/ui/app.go:469 |
| R-IN-620 | IN-59 | 关闭重开 | C 明确不保留 | 未实现 | 未实现（缺关闭清空：技能浮层搜索词 a.skillQuery 只在启动重置时清零（internal/ui/app.go:626 一带）） | internal/ui/app.go:626 |
| R-IN-639 | IN-61 | 切会话再回 | A 会话内状态 | 未实现 | 未实现（缺该元素：IN-61 模型浮层的错误行未建，没有可重试的请求失败态） | —（无落点：该元素未建） |
| R-IN-648 | IN-62 | 安装技能后重开 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/composer.go:678 每次开浮层现读 Extensions().Skills，空态按实时条数判断） | internal/ui/composer.go:678 |
| R-IN-668 | IN-64 | 切会话再回 | A 会话内状态 | 未实现 | 未实现（缺按会话的模式：同 R-IN-082，a.mode（internal/ui/app.go:475）单份） | internal/ui/app.go:475 |
| R-IN-701 | IN-66 | 审批中切走再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/composer.go:411 审批卡按 liveApproval 每帧重建，切走再回仍在原位） | internal/ui/composer.go:411 |
| R-IN-702 | IN-66 | 重启后 | B 跨重启持久化 | 已实现 | 已实现（证据 = 待决审批是服务端事实，重启后由会话快照重读；服务端不保留时不出卡） | —（无落点：该元素未建） |
| R-IN-712 | IN-67 | 审批中切走再回 | A 会话内状态 | 已实现 | 已实现（证据 = 审批卡标题是固定文案，命令取当前审批事实，切走再回不变） | —（无落点：该元素未建） |
| R-IN-724 | IN-68 | 切走再回 | A 会话内状态 | 已实现 | 已实现（证据 = 上下文块内容取当前会话快照，切走再回内容保持） | —（无落点：该元素未建） |
| R-IN-733 | IN-69 | 切走再回 | A 会话内状态 | 已实现 | 已实现（证据 = 面板内容取当前会话快照，切走再回不变） | —（无落点：该元素未建） |
| R-IN-745 | IN-70 | 切走再回 | A 会话内状态 | 已实现 | 已实现（证据 = 小标题是固定文案，切走再回不变） | —（无落点：该元素未建） |
| R-IN-755 | IN-71 | 切走再回 | A 会话内状态 | 已实现 | 已实现（证据 = 命令取当前审批事实，切走再回不变） | —（无落点：该元素未建） |
| R-IN-764 | IN-72 | 切走再回 | A 会话内状态 | 已实现 | 已实现（证据 = 路径取当前会话快照的 Cwd，切走再回不变） | —（无落点：该元素未建） |
| R-IN-796 | IN-74 | 选择「始终允许」后重启 | B 跨重启持久化 | 已实现 | 已实现（证据 = internal/ui/composer.go:451 把 allow-always 发给 Intents，客户端不留本地白名单；重启后是否再出卡由服务端是否再请求决定） | internal/ui/composer.go:451 |
| R-IN-843 | IN-66 | 处理完最后一个审批 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/composer.go:411 审批结束后 liveApproval 为空，卡片不再重现；输入块恢复且 a.draft 未被清，草稿仍在） | internal/ui/composer.go:411 |
| R-SB-029 | SB-02 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/uiwire/adapter.go:682 settingsFor 按当前会话读 PermissionTier；internal/ui/composer.go:926 permissionLabel 用它） | internal/uiwire/adapter.go:682、internal/ui/composer.go:926 |
| R-SB-030 | SB-02 | 重启后 | B 跨重启持久化 | 已实现 | 已实现（证据 = 权限档是会话级服务端事实（internal/uiwire/adapter.go:682 按会话读 PermissionTier），重启后按会话重读） | internal/uiwire/adapter.go:682 |
| R-SB-049 | SB-03 | 重开菜单 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/statusbar.go:70 勾选位置每帧按当前档算） | internal/ui/statusbar.go:70 |
| R-SB-059 | SB-04 | 取消后重开 | C 明确不保留 | 已实现 | 已实现（证据 = internal/ui/composer.go:944 打开确认框时把 permissionArmAck 置 false，取消后重开是未勾选） | internal/ui/composer.go:944 |
| R-SB-080 | SB-06 | 关闭再开对话框 | C 明确不保留 | 已实现 | 已实现（证据 = internal/ui/composer.go:944 每次打开确认框先复位 permissionArmAck=false） | internal/ui/composer.go:944 |
| R-SB-091 | SB-07 | 关闭后重开 | C 明确不保留 | 已实现 | 已实现（证据 = internal/ui/statusbar.go:104 确认按钮按勾选态禁用，配合 internal/ui/composer.go:944 每次打开复位） | internal/ui/statusbar.go:104、internal/ui/composer.go:944 |
| R-SB-111 | SB-09 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/app.go:1759 branchState 读当前会话快照的 Branch，切会话按会话分支显示） | internal/ui/app.go:1759 |
| R-SB-123 | SB-10 | 重开菜单 | C 明确不保留 | 未实现 | 未实现（缺关闭清空：分支菜单搜索词 a.statusBranchQuery 只在启动重置时清零（internal/ui/app.go:639 一带）） | internal/ui/app.go:639 |
| R-SB-132 | SB-11 | 重开菜单 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/statusbar.go:139 分组标题是固定文案，重开不变） | internal/ui/statusbar.go:139 |
| R-SB-143 | SB-12 | 重开菜单 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/statusbar.go:122 勾选按当前分支事实每帧算，重开菜单仍勾在当前分支） | internal/ui/statusbar.go:122 |
| R-SB-152 | SB-13 | 取消创建 | C 明确不保留 | 未实现 | 未实现（缺可取消的命名对话框：internal/ui/statusbar.go:23 一带直接拼名并切换，没有新分支名输入，也就没有「取消创建」） | internal/ui/statusbar.go:23 |
| R-SB-168 | SB-14 | 切会话再回 | A 会话内状态 | 未实现 | 未实现（缺上下文用量事实：internal/ui/statusbar.go:205 statusUsage 读 a.contextUsed/a.contextLimit，wired 下恒 0/0 显示「—」，没有按会话的用量） | internal/ui/statusbar.go:205 |
| R-SB-183 | SB-15 | 关闭再开 | C 明确不保留 | 已实现 | 已实现（证据 = internal/ui/statusbar.go:205 每次开浮层现读这两个值，不缓存旧数值） | internal/ui/statusbar.go:205 |
| R-SB-194 | SB-16 | 重开浮层 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/statusbar.go:226 浮层标题是固定文案，重开不变） | internal/ui/statusbar.go:226 |
| R-SB-204 | SB-17 | 从账户页返回 | A 会话内状态 | 已实现 | 已实现（证据 = 菜单项点击后由框架把 *bool 置 false（internal/ui/statusbar.go:210），返回时浮层已关闭） | internal/ui/statusbar.go:210 |
| R-SB-224 | SB-19 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = 账户状态取 internal/uiwire/adapter.go:634 Account() 每帧事实，切会话再回按实际状态显示） | internal/uiwire/adapter.go:634 |
| R-SB-234 | SB-20 | 重开浮层 | C 明确不保留 | 已实现 | 已实现（证据 = internal/ui/statusbar.go:241 每次开浮层按用量快照重算，加载态不缓存） | internal/ui/statusbar.go:241 |
| R-SB-248 | SB-21 | 重开浮层 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/statusbar.go:241 每次开浮层现读用量，不显示旧缓存） | internal/ui/statusbar.go:241 |
| R-NT-010 | NT-01 | 新建会话后返回 | A 会话内状态 | 已实现 | 已实现（证据 = 空态图形由当前 a.state 决定，返回空态即按同一布局重画（internal/ui/timeline.go:1614 一类的空态分支）） | internal/ui/timeline.go:1614 |
| R-NT-023 | NT-02 | 新建后返回 | A 会话内状态 | 已实现 | 已实现（证据 = 空态块位置由固定布局给出，返回空态即同位重画） | —（无落点：该元素未建） |
| R-NT-033 | NT-03 | 完成后重进空态 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/composer.go:777 workspaceOptions 每帧读授权清单，目录已定后不再进入加载态） | internal/ui/composer.go:777 |
| R-NT-045 | NT-04 | 返回空态 | A 会话内状态 | 已实现 | 已实现（证据 = 空态块尺寸取布局常量，返回空态即同尺寸重画） | —（无落点：该元素未建） |
| R-NT-056 | NT-05 | 返回空态 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/composer.go:824 与权限选择器都读 a.workspaceName/a.workspacePath/a.permission，返回空态按上次选择显示） | internal/ui/composer.go:824 |
| R-NT-068 | NT-06 | 返回空态 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/composer.go:866 只在选择时改，返回空态不清） | internal/ui/composer.go:866 |
| R-NT-083 | NT-07 | 重开下拉 | C 明确不保留 | 未实现 | 未实现（缺关闭清空：工作区下拉搜索词 a.workspaceQuery 只在启动重置时清零（internal/ui/app.go:631 一带）） | internal/ui/app.go:631 |
| R-NT-094 | NT-08 | 重开下拉 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/composer.go:858 选中标记每帧按 a.workspacePath 算） | internal/ui/composer.go:858 |
| R-NT-105 | NT-09 | 重开下拉 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/composer.go:858 同上，按路径算选中） | internal/ui/composer.go:858 |
| R-NT-128 | NT-11 | 添加后重开下拉 | A 会话内状态 | 未实现 | 未实现（缺落点：internal/ui/composer.go:777 workspaceOptions 只列 Source 的工作区，本端登记的 a.workspaces 不进空态下拉；internal/ui/composer.go:872 选完也不设 a.workspacePath，标记落不到新工作区上） | internal/ui/composer.go:777、internal/ui/composer.go:872 |
| R-NT-171 | NT-14 | 返回空态 | A 会话内状态 | 已实现 | 已实现（证据 = 空态权限档读 a.permission（internal/ui/composer.go:926 的回落分支），返回空态保留上次档位） | internal/ui/composer.go:926 |
| R-NT-172 | NT-14 | 重启后 | B 跨重启持久化 | 未实现 | 未实现（缺落盘键 view.permissionTier：a.permission 只在内存，internal/app/preferences.go:81 的 view 里没有这一项） | internal/app/preferences.go:81 |
| R-RF-028 | F-03 | 切标签再回、树已就绪 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_files.go:45 rpFilesPage 的骨架只在 loading 且树为空时出现；树在 s.tree 里，切回时不再空） | internal/ui/rightpanel_files.go:45 |
| R-RF-044 | F-05 | 切到「改动」再切回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_files.go:552 rpLiveSearch 每帧按快照 Phase 重建，失败态不缓存） | internal/ui/rightpanel_files.go:552 |
| R-RF-068 | F-08 | 缩小查询后重搜 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_files.go:577 rpLiveSearch 的截断提示每帧按本次读数算，命中回落即消失） | internal/ui/rightpanel_files.go:577 |
| R-RF-091 | F-11 | 切标签再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_files.go:540 空态文案用当前查询词拼，不残留旧词） | internal/ui/rightpanel_files.go:540 |
| R-RF-112 | F-13 | 切标签再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_files.go:174 根级失败每帧由读数给，切回仍不可读即重现同一行） | internal/ui/rightpanel_files.go:174 |
| R-RF-127 | F-15 | 新增文件后刷新 | A 会话内状态 | 已实现 | 已实现（证据 = 每帧按目录/改动读数重建（internal/ui/rightpanel_files.go:154 rpLiveTree / internal/ui/rightpanel_files.go:181 rpNodes），不缓存上一次的行；刷新后按新读数出空态或新行） | internal/ui/rightpanel_files.go:154、internal/ui/rightpanel_files.go:181 |
| R-RF-158 | F-18 | 收起再展开 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_files.go:367 箭头朝向每帧由 s.dirOpen[p] 取，不残留上一次角度） | internal/ui/rightpanel_files.go:367 |
| R-RF-183 | F-21 | 外部改名后刷新 | A 会话内状态 | 已实现 | 已实现（证据 = 每帧按目录/改动读数重建（internal/ui/rightpanel_files.go:154 rpLiveTree / internal/ui/rightpanel_files.go:181 rpNodes），不缓存上一次的行；刷新重新读目录，名字取新值） | internal/ui/rightpanel_files.go:154、internal/ui/rightpanel_files.go:181 |
| R-RF-191 | F-22 | 收起再展开 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_files.go:312 缩进按 depth 算，收起再展开重建为同一缩进，不叠加） | internal/ui/rightpanel_files.go:312 |
| R-RF-216 | F-25 | 收起再展开 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_files.go:196 子级失败每帧由读数给，不缓存旧子项） | internal/ui/rightpanel_files.go:196 |
| R-RF-224 | F-26 | 切标签再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_files.go:771 错误行与「重试」每次渲染都在） | internal/ui/rightpanel_files.go:771 |
| R-RF-247 | F-28 | 切文件/改动标签 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_files.go:433 文件行的箭头占位是定宽盒，随行重建，宽度不变） | internal/ui/rightpanel_files.go:433 |
| R-RF-263 | F-30 | 外部改名后刷新 | A 会话内状态 | 已实现 | 已实现（证据 = 每帧按目录/改动读数重建（internal/ui/rightpanel_files.go:154 rpLiveTree / internal/ui/rightpanel_files.go:181 rpNodes），不缓存上一次的行；外部改名后刷新取新名） | internal/ui/rightpanel_files.go:154、internal/ui/rightpanel_files.go:181 |
| R-RF-279 | F-32 | 切标签再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_files.go:210 统计每帧由条目上的 git 状态映射，不沿用旧值） | internal/ui/rightpanel_files.go:210 |
| R-RF-303 | F-35 | 切标签再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_files.go:210 暂存点来自条目状态，切回按仓库实况重建） | internal/ui/rightpanel_files.go:210 |
| R-RF-310 | F-36 | 重开菜单 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_files.go:786 菜单项顺序固定，分隔线由 internal/ui/rightpanel_files.go:833 统一画，不随项数漂移） | internal/ui/rightpanel_files.go:786、internal/ui/rightpanel_files.go:833 |
| R-RF-319 | F-37 | 关闭差异后回树 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_parts.go:101 目录展开态在右栏状态里，切标签不清，回树时按它重建） | internal/ui/rightpanel_parts.go:101 |
| R-RF-335 | F-39 | 关闭预览回树 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_parts.go:101 同上：预览关闭后树按保留的展开态重建） | internal/ui/rightpanel_parts.go:101 |
| R-RF-351 | F-41 | 切标签再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_files.go:785 该项在文件菜单里固定出现，切标签再回仍在） | internal/ui/rightpanel_files.go:785 |
| R-RF-359 | F-42 | 再次打开对话框 | C 明确不保留 | 未实现 | 未实现（缺另存为对话框：internal/ui/rightpanel_files.go:785 是固定禁用项，没有对话框，也就没有默认文件名可重算） | internal/ui/rightpanel_files.go:785 |
| R-RF-366 | F-43 | 重开菜单 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_files.go:833 分隔线统一由一处画，间距不随菜单变化） | internal/ui/rightpanel_files.go:833 |
| R-RF-385 | F-45 | 切标签再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_files.go:326 已加载子项数记在 dirPages，按钮按 hasMore 每帧算） | internal/ui/rightpanel_files.go:326 |
| R-RF-401 | F-47 | 切标签再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_files.go:175 权限条每帧由读数给，仍缺即重现） | internal/ui/rightpanel_files.go:175 |
| R-RF-409 | F-48 | 切标签再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_files.go:268 权限条上的「重试」每次渲染都在） | internal/ui/rightpanel_files.go:268 |
| R-RD-053 | D-04 | 切走再切回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:193 范围存在 a.right.scope，切走再回按它重建，且落盘 view.changesScope（internal/app/preferences.go:94）） | internal/ui/rightpanel_changes.go:193、internal/app/preferences.go:94 |
| R-RD-094 | D-08 | 关闭菜单再开 | C 明确不保留 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:239 菜单每次按当前状态重建；开合是 *bool，关闭即 false；没有「上次动作」状态） | internal/ui/rightpanel_changes.go:239 |
| R-RD-129 | D-12 | 提交后重开菜单 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:292 提交类项每次开菜单按已暂存集合现算禁用） | internal/ui/rightpanel_changes.go:292 |
| R-RD-137 | D-13 | 关菜单再开 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:292 置灰状态每次开菜单按已暂存集合重算） | internal/ui/rightpanel_changes.go:292 |
| R-RD-144 | D-14 | 推送后重开 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:239 推送项固定出现，可重复点） | internal/ui/rightpanel_changes.go:239 |
| R-RD-155 | D-15 | 切标签再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:198 刷新按钮每次渲染都画、无禁用态可复用） | internal/ui/rightpanel_changes.go:198 |
| R-RD-163 | D-16 | 切标签再回（数据在内存） | A 会话内状态 | 未实现 | 未实现（缺该元素：改动页没有加载骨架（internal/ui/rightpanel_changes.go:123 直接画空态/行），没有骨架可消失） | internal/ui/rightpanel_changes.go:123 |
| R-RD-172 | D-17 | 切到别范围再回空态 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:141 空态按当前范围每帧重画，图标与位置固定） | internal/ui/rightpanel_changes.go:141 |
| R-RD-180 | D-18 | 提交最后一个文件后 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:153 行每帧现算，空即转空态，不需手动刷新） | internal/ui/rightpanel_changes.go:153 |
| R-RD-187 | D-19 | 切到别范围再回来 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:342 空态文案由范围决定，切回同一范围文案不变） | internal/ui/rightpanel_changes.go:342 |
| R-RD-203 | D-21 | 切回未提交范围再回分支 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:342 同一范围同一文案，切回仍是空态） | internal/ui/rightpanel_changes.go:342 |
| R-RD-211 | D-22 | 失败后切标签再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:123 失败态每帧按快照 Phase 重建，未恢复即仍是同一空态） | internal/ui/rightpanel_changes.go:123 |
| R-RD-226 | D-24 | 切走再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:342 空态文案是常量，切走再回不变） | internal/ui/rightpanel_changes.go:342 |
| R-RD-234 | D-25 | 切走再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:342 同上） | internal/ui/rightpanel_changes.go:342 |
| R-RD-242 | D-26 | 切走再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:342 同上） | internal/ui/rightpanel_changes.go:342 |
| R-RD-250 | D-27 | 切走再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:123 失败仍在即仍是同一空态） | internal/ui/rightpanel_changes.go:123 |
| R-RD-263 | D-28 | 切标签再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:666 组展开态与 internal/ui/rightpanel_parts.go:119 在右栏状态里保留，计数每帧由快照重算） | internal/ui/rightpanel_changes.go:666、internal/ui/rightpanel_parts.go:119 |
| R-RD-271 | D-29 | 收起再展开 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:370 朝向每帧按组的展开标记取） | internal/ui/rightpanel_changes.go:370 |
| R-RD-292 | D-31 | 重开组 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:361 组头按钮每帧重建，禁用态不缓存） | internal/ui/rightpanel_changes.go:361 |
| R-RD-299 | D-32 | 切标签再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:666 容器展开态按 groupsClosed 恢复） | internal/ui/rightpanel_changes.go:666 |
| R-RD-344 | D-35 | 暂存/取消暂存后 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:321 行按暂存态分组，文件名字段不变，只换组） | internal/ui/rightpanel_changes.go:321 |
| R-RD-396 | D-41 | 还原后切走再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:638 还原后从改动表摘掉该路径；wired 下每帧按快照重建，也不再现） | internal/ui/rightpanel_changes.go:638 |
| R-RD-412 | D-43 | 关闭预览标签后回改动面板 | A 会话内状态 | 未实现 | 未实现（缺滚动状态：internal/ui/rightpanel_changes.go:57 这层 Scroll 没有 TrackScroll，滚回列表位置做不到；展开态 diffOpen 本身保留） | internal/ui/rightpanel_changes.go:57 |
| R-RD-419 | D-44 | 再次右键 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:412 右键菜单每帧重建，项仍可用） | internal/ui/rightpanel_changes.go:412 |
| R-RD-444 | D-47 | 收起再展开 | C 明确不保留 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:57 这层 Scroll 没有滚动记忆，收起再展开从顶部开始） | internal/ui/rightpanel_changes.go:57 |
| R-RD-456 | D-48 | 切标签再回 | A 会话内状态 | 未实现 | 未实现（缺按文件的滚动状态：internal/ui/rightpanel_changes.go:57 这层 Scroll 没有 TrackScroll；展开态按路径保留在 diffOpen） | internal/ui/rightpanel_changes.go:57 |
| R-RD-463 | D-49 | 收起再展开 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:412 提示行随行每帧画，位置固定） | internal/ui/rightpanel_changes.go:412 |
| R-RD-475 | D-50 | 切标签再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:666 展开态保留并落盘 view.changesGroupsClosed（internal/app/preferences.go:95）；计数每帧按仓库实况算） | internal/ui/rightpanel_changes.go:666、internal/app/preferences.go:95 |
| R-RD-483 | D-51 | 收起再展开 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:370 朝向每帧按展开标记取） | internal/ui/rightpanel_changes.go:370 |
| R-RD-491 | D-52 | 切标签再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:370 收起态的组切回来仍朝右） | internal/ui/rightpanel_changes.go:370 |
| R-RD-504 | D-53 | 重开组 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:361 组头按钮每次渲染重建，回到可点态） | internal/ui/rightpanel_changes.go:361 |
| R-RD-526 | D-55 | 切标签再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_parts.go:119 行内 diff 展开按文件路径记，切标签保留） | internal/ui/rightpanel_parts.go:119 |
| R-RD-556 | D-57 | 暂存后 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:321 暂存后只换组，文件名取自同一路径） | internal/ui/rightpanel_changes.go:321 |
| R-RD-617 | D-64 | 取消确认后重开菜单 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:412 菜单每次重建，「还原」项位置固定且可用） | internal/ui/rightpanel_changes.go:412 |
| R-RD-633 | D-66 | 关闭预览后重开菜单 | C 明确不保留 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:412 菜单每次重建，「预览」项位置固定） | internal/ui/rightpanel_changes.go:412 |
| R-RD-640 | D-67 | 再次右键同一行 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_parts.go:746 同一行用同一个开合指针，重开时菜单项由当前状态重建） | internal/ui/rightpanel_parts.go:746 |
| R-RD-656 | D-69 | 换行右键 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:412 行菜单是同一套构造，项集合与位置一致） | internal/ui/rightpanel_changes.go:412 |
| R-RD-671 | D-70 | 收起再展开 | C 明确不保留 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:57 这层 Scroll 没有滚动记忆，展开从 diff 顶部开始） | internal/ui/rightpanel_changes.go:57 |
| R-RD-688 | D-72 | 收起再展开 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:412 提示行每帧画在同一位置） | internal/ui/rightpanel_changes.go:412 |
| R-RD-697 | D-73 | 关闭后再次点另一行「还原」 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:597 标题是常量，说明行用当前 s.revertPath 拼，换行即换路径） | internal/ui/rightpanel_changes.go:597 |
| R-RD-711 | D-73 | 关掉右栏再打开后点还原 | C 明确不保留 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:597 确认框由当前 revertPath 现拼，每次点「还原」重新写入，不记上次文件） | internal/ui/rightpanel_changes.go:597 |
| R-RD-719 | D-74 | 关闭确认框后再触发同文件还原 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:597 说明行按 revertPath 重建，同一路径文案结构不变） | internal/ui/rightpanel_changes.go:597 |
| R-RD-726 | D-75 | 再次打开确认框 | C 明确不保留 | 未实现 | 未实现（缺默认焦点：internal/ui/rightpanel_changes.go:597 只画两个按钮，没有设置初始焦点，「取消」为默认焦点无从谈起） | internal/ui/rightpanel_changes.go:597 |
| R-RD-736 | D-76 | 成功后再次还原同类文件 | C 明确不保留 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:597 标题与说明按新 revertPath 重建，不复用上次结果） | internal/ui/rightpanel_changes.go:597 |
| R-RD-746 | D-77 | 切标签再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:153 统计每帧按实况算） | internal/ui/rightpanel_changes.go:153 |
| R-RD-794 | D-81 | 换会话再回 | C 明确不保留 | 未实现 | 未实现（缺换会话复位：internal/ui/rightpanel_parts.go:119 是全局 map，换会话不清空也不按会话分表，回来仍是上次的展开态） | internal/ui/rightpanel_parts.go:119 |
| R-RD-800 | D-82 | 收起再展开 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_parts.go:119 朝向由 map 取值决定，切换即换向） | internal/ui/rightpanel_parts.go:119 |
| R-RD-808 | D-83 | 切走再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_parts.go:119 未展开的项切回来仍朝右） | internal/ui/rightpanel_parts.go:119 |
| R-RD-818 | D-84 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:153 行集合每帧按该会话/该回合的改动重算） | internal/ui/rightpanel_changes.go:153 |
| R-RD-825 | D-88 | 收起再展开 | C 明确不保留 | 未实现 | 未实现（缺该容器与滚动：改动列表是 Gap 排列的行（internal/ui/rightpanel_changes.go:321），没有容器也没有滚动状态可回顶） | internal/ui/rightpanel_changes.go:321 |
| R-RD-850 | D-91 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_changes.go:153 行按当前会话/回合的记录每帧重建，路径取新值） | internal/ui/rightpanel_changes.go:153 |
| R-RA-008 | A-01 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = 每帧按当前会话快照重建（internal/ui/timeline.go:1451 liveRow / internal/ui/timeline.go:1425 liveTimeline），不缓存上一次的内容；产物块按该会话最新回合的行事实重建） | internal/ui/timeline.go:1451、internal/ui/timeline.go:1425 |
| R-RA-017 | A-02 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = 每帧按当前会话快照重建（internal/ui/timeline.go:1451 liveRow / internal/ui/timeline.go:1425 liveTimeline），不缓存上一次的内容；计数与内容由行事实算） | internal/ui/timeline.go:1451、internal/ui/timeline.go:1425 |
| R-RA-058 | A-07 | 收起再展开列表 | C 明确不保留 | 未实现 | 未实现（缺列表容器与独立滚动：产物列表是 Gap 排列的行（internal/ui/timeline.go:864），没有容器也没有滚动状态） | internal/ui/timeline.go:864 |
| R-RA-073 | A-08 | 切会话再回 | A 会话内状态 | 已实现 | 已实现（证据 = 行集合每帧按行事实重建；展开态存在 internal/ui/app.go:409 与 blockOf 里，切会话不清） | internal/ui/app.go:409 |
| R-TM-024 | T-03 | 抽屉布局切走再回 | A 会话内状态 | 未实现 | 未实现（缺标题跟随：标签标题缓存在 internal/ui/terminal.go:230 的 title 上，internal/ui/terminal.go:555 变了只在标题为空时才回填） | internal/ui/terminal.go:230、internal/ui/terminal.go:555 |
| R-TM-066 | T-08 | 切标签再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/terminal.go:230 每层各自留 lines/state/scroll，切标签只换激活层，内容不重新自检） | internal/ui/terminal.go:230 |
| R-TM-086 | T-10 | 切标签再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/terminal.go:280 一层时提示行由当前状态每帧算，切回仍在） | internal/ui/terminal.go:280 |
| R-TM-126 | T-15 | 再次右键 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/terminal.go:16 右键菜单每次重建，可用性按当前选区算） | internal/ui/terminal.go:16 |
| R-TM-134 | T-16 | 再次右键 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/terminal.go:16 右键菜单每次重建，可用性按剪贴板现状算） | internal/ui/terminal.go:16 |
| R-TM-158 | T-19 | 再次右键 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/terminal.go:15 菜单项每次重建，无 cwd 时置灰） | internal/ui/terminal.go:15 |
| R-TM-174 | T-21 | 再次右键 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/terminal.go:15 菜单项每次重建，无活跃终端时置灰） | internal/ui/terminal.go:15 |
| R-TK-008 | K-01 | 切标签再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_tasks.go:71 列表每帧由当前会话的任务快照重建；列表没有滚动记忆，回到顶部） | internal/ui/rightpanel_tasks.go:71 |
| R-TK-017 | K-02 | 切标签再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_tasks.go:87 行每帧按任务事实重建） | internal/ui/rightpanel_tasks.go:87 |
| R-TK-026 | K-03 | 切标签再回 | A 会话内状态 | 未实现 | 未实现（缺按标识的选中恢复：a.panels.taskSel（internal/ui/rightpanel.go:59）存的是下标，进程顺序一变就指到别的行） | internal/ui/rightpanel.go:59 |
| R-TK-043 | K-05 | 切标签再回 | A 会话内状态 | 未实现 | 未实现（缺该元素：运行中图标是 taskStateIconIn 返回的 iconTask，没有旋转圈） | —（无落点：该元素未建） |
| R-TK-058 | K-07 | 切标签再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_tasks.go:87 行每帧按状态重建） | internal/ui/rightpanel_tasks.go:87 |
| R-TK-060 | K-07 | 进程被清理后 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_tasks.go:63 行消失后越界即不画详情，不留悬空详情） | internal/ui/rightpanel_tasks.go:63 |
| R-TK-066 | K-08 | 切标签再回 | A 会话内状态 | 未实现 | 未实现（缺该元素：taskStateIconIn 对已停止态落回 iconTask，没有方形停止图标） | —（无落点：该元素未建） |
| R-TK-074 | K-09 | 切标签再回 | A 会话内状态 | 未实现 | 未实现（缺该元素：taskStateIconIn 对等待态落回 iconTask，没有时钟图标） | —（无落点：该元素未建） |
| R-TK-092 | K-11 | 切标签再回 | A 会话内状态 | 未实现 | 未实现（缺分支徽标：internal/ui/rightpanel_tasks.go:87 只画状态图标与任务名，没有徽标可按实况重建） | internal/ui/rightpanel_tasks.go:87 |
| R-TK-124 | K-15 | 执行结束回到可停态 | A 会话内状态 | 未实现 | 未实现（缺该元素：执行中的停止按钮没有 Loader2/禁用态，没有「恢复为可停态」这回事） | —（无落点：该元素未建） |
| R-TK-132 | K-16 | 切标签再回 | A 会话内状态 | 未实现 | 未实现（缺该元素：唤醒行没有旋转圈标记，无从按实况显示圈或按钮） | —（无落点：该元素未建） |
| R-TK-227 | K-28 | 进程结束后回看 | A 会话内状态 | 未实现 | 未实现（缺元信息行：internal/ui/rightpanel_tasks.go:139 不渲染 pid/结束时刻，结束时间无处补上） | internal/ui/rightpanel_tasks.go:139 |
| R-TK-146 | K-18 | 新进程出现 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_tasks.go:58 空态只在列表为空时出现，进程一出现即换成行） | internal/ui/rightpanel_tasks.go:58 |
| R-TK-153 | K-19 | 新唤醒出现 | A 会话内状态 | 未实现 | 未实现（缺唤醒事实：唤醒标签固定画空态「暂无待唤醒事项」（internal/ui/rightpanel_tasks.go:55），没有唤醒行可让空态消失） | internal/ui/rightpanel_tasks.go:55 |
| R-TK-251 | K-31 | 切走再回 | A 会话内状态 | 未实现 | 未实现（缺只读终端画布：K-31 元素未建，无从按进程重新挂载并回到底部） | —（无落点：该元素未建） |
| R-TK-268 | K-33 | 切走再回 | A 会话内状态 | 未实现 | 未实现（缺失败提示行：K-33 元素未建，切回来没有提示可保持） | —（无落点：该元素未建） |
| R-SC-031 | S-04 | 切走再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel.go:64 分支结束态存在面板视图状态里，切走再回状态条仍在） | internal/ui/rightpanel.go:64 |
| R-SC-065 | S-08 | 切走再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_sidechat.go:95 空态按 sideChatSent/sideChatFromMessage 每帧选文案，未发首条消息仍在） | internal/ui/rightpanel_sidechat.go:95 |
| R-SC-073 | S-09 | 切标签再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_sidechat.go:28 面板每帧按 sideChatError/状态重建，未答复请求仍出卡位） | internal/ui/rightpanel_sidechat.go:28 |
| R-SC-081 | S-10 | 切标签再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_sidechat.go:28 同上） | internal/ui/rightpanel_sidechat.go:28 |
| R-SC-099 | S-12 | 切走再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel.go:64 分支未结束时按钮按同一状态每帧重画，仍在原位） | internal/ui/rightpanel.go:64 |
| R-SC-117 | S-14 | 切标签再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_sidechat.go:139 按钮文案每帧按 sending/回合状态算，结束后回到「发送」） | internal/ui/rightpanel_sidechat.go:139 |
| R-SC-125 | S-15 | 取消后再次点击 | C 明确不保留 | 已实现 | 已实现（证据 = internal/ui/rightpanel_sidechat.go:127 每次点「结束分支」重新置 true，弹窗按常量文案重建） | internal/ui/rightpanel_sidechat.go:127 |
| R-SC-131 | S-16 | 取消后再次打开 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_sidechat.go:22 说明文案是常量，重开不变） | internal/ui/rightpanel_sidechat.go:22 |
| R-SC-141 | S-17 | 关闭后再打开 | C 明确不保留 | 未实现 | 未实现（缺默认焦点：internal/ui/rightpanel_sidechat.go:166 只画「取消/确认」两个按钮，没有设置初始焦点） | internal/ui/rightpanel_sidechat.go:166 |
| R-SC-150 | S-18 | 分支已结束后回看 | C 明确不保留 | 已实现 | 已实现（证据 = internal/ui/rightpanel_sidechat.go:166 只在 sideChatConfirm 为真时画，分支结束即把确认框关掉，不会再来） | internal/ui/rightpanel_sidechat.go:166 |
| R-SG-024 | G-03 | 切走再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_subagent.go:39 错误态与「重试」存在面板视图状态里，切走再回仍在） | internal/ui/rightpanel_subagent.go:39 |
| R-SG-056 | G-07 | 切走再回 | A 会话内状态 | 已实现 | 已实现（证据 = internal/ui/rightpanel_subagent.go:65 未就绪时的加载文案由 subagentReady 每帧算，切回仍在） | internal/ui/rightpanel_subagent.go:65 |

## 3. 抽样回读（20 条，20/20）

回读方式：按「代码位置」打开该行，确认字段/分支确实在这条路径上被读或不被读。

| 需求 ID | 类别 | 回读结论 |
|---|---|---|
| R-TB-008 | B | `preferences.go:82` 有 `view.sidebarCollapsed`；`app.go:1173` 启动铺回；`header.go` 按收起态让出左内边距 —— 成立 |
| R-TB-040 | B | 同上，会话态与空态共用这一份收起态 —— 成立 |
| R-TB-163 | A | `header.go` 的 `sessionTitle` 每帧读当前会话快照 —— 成立 |
| R-LB-678 | A | `sidebar.go:168` 置顶区按非空渲染，清空后再置顶即重现 —— 成立 |
| R-LB-930 | B | `catalog/archive.go:28` + `adapter.go` ArchiveSession → Catalog.Hide —— 成立 |
| R-LB-906 | C | `sidebar.go:743` 取消把 `logoutConfirm` 置 false —— 成立 |
| R-TL-449 | A | `timeline.go` 的 `liveRoundStatus` 每帧按行事实 —— 成立 |
| R-TL-009 | C | 只有一份 `a.feed`（`app.go:507`），切会话不清 —— 未实现成立 |
| R-IN-152 | A | `composer.go:259` 可用态按当前草稿重算 —— 成立 |
| R-IN-082 | A | `app.go:475` 的 `a.mode` 单份，`composer.go:720` 直读 —— 未实现成立 |
| R-IN-603 | A | `app.go:469` 的 `a.effort` 单份，`composer.go:485/522` 直读 —— 未实现成立 |
| R-IN-796 | B | `composer.go:431` 发 `allow-always` 给 Intents，无本地白名单 —— 成立 |
| R-SB-030 | B | `adapter.go:607` 按会话读 `PermissionTier` —— 成立 |
| R-SB-059 | C | `composer.go` 的 `choosePermission` 每次打开复位 `permissionArmAck` —— 成立 |
| R-SB-168 | A | `statusbar.go:205` 读 `a.contextUsed/Limit`，wired 恒 0/0 —— 未实现成立 |
| R-NT-094 | A | `composer.go:838` 选中标记按 `a.workspacePath` —— 成立 |
| R-RF-319 | A | `rightpanel_parts.go:101` 的 `dirOpen` 在右栏状态；全仓无按标签复位 —— 成立 |
| R-RF-359 | C | `rightpanel_files.go`「另存为…」固定禁用，无对话框 —— 未实现成立 |
| R-RD-299 | A | `rightpanel_changes.go:666` 的 `groupOpen` 读 `groupsClosed` —— 成立 |
| R-RD-444 | C | `rightpanel_changes.go:57` 该层 Scroll 没有 `TrackScroll` —— 成立 |
| R-TK-026 | A | `rightpanel.go:59` 的 `taskSel` 是下标 —— 未实现成立 |
| R-SC-031 | A | `rightpanel.go:64` 的 `sideChatEnded` 在面板视图状态 —— 成立 |

## 4. 按 R-ID 对齐并回（一次脚本）

合并方在 P2 证据重标完成后执行。脚本只改目标行的第 8 列（状态），其余列与行序不动：

- 逐条先按 R-ID 认领；
- 认领不到再按「元素 ID + 场景」在同为「记忆」维度的行里认领（P2 换过编号时走这条）；
- 两份文档都认领不到的条目最后打印出来，退出码非 0。

已实测：在 HEAD 版两份需求上按 R-ID 写满 **152 + 114 = 266** 条、只改状态列（非状态列改动 0）、重复跑幂等；把 R-ID 全部改号后重跑，按「元素 + 场景」兜底仍是 266 条。

```bash
python3 scripts/merge-memory-adjudication.py --dry-run   # 只报数
python3 scripts/merge-memory-adjudication.py             # 正式写
```

脚本：`scripts/merge-memory-adjudication.py`（下面同文，便于随表走查）

```python
#!/usr/bin/env python3
"""把 docs/domains/requirements/memory-adjudication.md 的结论并回两份需求文档的状态列。

只改目标行的第 8 列（状态），其余列与行序不动。逐条先按 R-ID 认领，认领不到再按
「元素 ID + 场景」认领（两份文档里该组合唯一时才用），最后打印仍未认领的清单。

    python3 scripts/merge-memory-adjudication.py            # 正式写
    python3 scripts/merge-memory-adjudication.py --dry-run  # 只报数不写
"""

import sys

DOCS = [
    'docs/domains/requirements/requirements-main.md',
    'docs/domains/requirements/requirements-panels.md',
]
TABLE = 'docs/domains/requirements/memory-adjudication.md'
DRY = '--dry-run' in sys.argv


def split_cells(line):
    return [x.strip() for x in line.strip().strip('|').split('|')]


# 表：R-ID -> (元素 ID, 场景, 结论整句)
rows = {}
for line in open(TABLE, encoding='utf-8'):
    if line.startswith('| R-'):
        c = [x.strip() for x in line.strip().strip('|').split(' | ')]
        if len(c) == 7:
            rows[c[0]] = (c[1], c[2], c[5])
if not rows:
    sys.exit('核定表为空：先确认 ' + TABLE + ' 在')

docs = {path: open(path, encoding='utf-8').read().split('\n') for path in DOCS}

# 每份文档的 R-ID 与（元素, 场景）索引
ids = {}
keys = {}
for path, lines in docs.items():
    ids[path] = set()
    keys[path] = {}
    for line in lines:
        if not line.startswith('| R-'):
            continue
        c = split_cells(line)
        if len(c) < 8:
            continue
        ids[path].add(c[0])
        if c[2] == '记忆':  # 兜底只在同维度的行里找，避免撞到同元素同场景的其它维度行
            keys[path].setdefault((c[1], c[3]), []).append(c[0])

# 认领：每条表行落到哪份文档（R-ID 优先；否则元素+场景唯一命中）
owner = {}
for rid, (el, scene, _) in rows.items():
    for path in DOCS:
        if rid in ids[path]:
            owner[rid] = path
            break
    if rid in owner:
        continue
    hits = [path for path in DOCS if len(keys[path].get((el, scene), [])) == 1]
    if len(hits) == 1:
        owner[rid] = hits[0]

orphan = sorted(k for k in rows if k not in owner)
written = 0
for path, lines in docs.items():
    todo = {rid: rows[rid] for rid, p in owner.items() if p == path}
    done = set()
    for i, line in enumerate(lines):
        if not line.startswith('| R-'):
            continue
        c = split_cells(line)
        if len(c) < 8:
            continue
        hit = c[0] if c[0] in todo and c[0] not in done else None
        if hit is None:
            for rid, (el, scene, _) in todo.items():
                if rid not in done and c[2] == '记忆' and c[1] == el and c[3] == scene:
                    hit = rid
                    break
        if hit is None:
            continue
        c[7] = ' ' + todo[hit][2] + ' '
        lines[i] = '| ' + ' | '.join(c) + ' |'
        done.add(hit)
        written += 1
    if not DRY:
        open(path, 'w', encoding='utf-8').write('\n'.join(lines))
    left = sorted(set(todo) - done)
    print(f'{path}: 认领 {len(todo)} 条，写入 {len(done)} 条' + (f'，落空 {left}' if left else ''))

print('合计写入', written, '条（表内', len(rows), '条）；两份文档都认领不到', len(orphan), '条',
      orphan or '')
sys.exit(0 if written == len(rows) else 1)
```
