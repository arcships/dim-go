# 设置运行时清单（模板 ∪ 运行时 ∪ 截图）

本文只做元素清点，不写行为需求。

## 0. 口径

### 0.1 三源材料

| 源 | 位置 | 用途 |
|---|---|---|
| 模板 | `components/settings/*Settings.vue`、`views/SettingsView.vue`、`views/settings/account/*.vue`（相对参考产品渲染进程源码根 `packages/desktop/packages/renderer/src`） | 行清单的来源：每个元素一行，`v-if/v-else` 每个分支各成一行 |
| 运行时 | `runtime/settings/&lt;节&gt;.json`（19 份，合计 3354 节点） | 判定「可见」，提供包围盒 `x/y/w/h`、`role`、`icon`、真实文案 |
| 截图 | `runtime/settings/&lt;节&gt;-top.png`、`&lt;节&gt;-bottom.png`（各 2×，2060×1450） | 图标、边框、分隔线等 JSON 不含的视觉信息；`png 像素 = CSS 坐标 × 2` |

运行时 JSON 结构：`{section, count, items:[{chain,tag,role,text,label,icon,x,y,w,h}]}`。一次抓取含多趟（首屏 + 滚到底），同一条目重复出现且 `y` 相差滚动量（内容区步长 686、侧栏步长 56）。本文取**同一节点 y 最大者**作为位置，即未滚动时的文档坐标；重复观测数与节点数的差额在每节统计里给出。

### 0.2 判定与标记

| 标记 | 含义 |
|---|---|
| 可见 | 在运行时 JSON 找到对应节点（同文案 / 同 `role` / 同位置），位置列给运行时包围盒 |
| 未出现（条件） | 模板有、运行时无，且分支条件解释了缺席（运行时走的是另一分支）；条件列写全「分支条件 + 祖先条件」 |
| 未命中 | 模板有、运行时无，且无分支条件可解释（无条件标记却未出现）→ 死代码候选，给行号 |
| 运行时独有（动态生成） | 运行时有、模板无同形节点：`v-for` 按数据生成的行、运行时拼接的文案 |
| 图标 | 模板里的独立图标组件单列一行；运行时该元素标 `icon: true`，但只给出整块包围盒，故位置列填宿主元素包围盒 |

### 0.3 元素计数规则

- **控件**：`Button`/`button`、`Input`、`Textarea`、`Select`、`Switch`、`Slider`、`Checkbox`、`Tabs`、`a` 各计一行；`v-for` 渲染的控件按实例计行（19 个导航按钮计 19 行）。
- **文本**：标题、`Label`、正文、`p`、`dt/dd`、`code`、徽标文本各计一行；同一容器里「标签 + 取值」并列时按模板节点分行。
- **容器**：带 `border`/`bg-card` 的区块（`rounded-lg border`）计一行；JSON 里没有对应行，位置由 png 边框实测（÷2）。
- **图标**：`icon: true` 的节点单列一行（`ArrowLeft`、lucide 图标、`<component :is="item.icon">`）。
- **不计行**：纯布局 `div`（无文案、无边框、无交互）、`ScrollArea`、`KeepAlive`、`component :is` 派发节点。
- **折叠**：控件内部的文字（`<Button><span>文案</span></Button>`）算控件自己的名称，不另计行；`dt`+`dd` 同理。

### 0.4 计数恒等式

```
模板行数 = 可见 + 未出现（条件） + 未命中
运行时节点数 = 可见（含图标行） + 运行时独有
```

---

## S-00 壳层（`views/SettingsView.vue` + `components/settings/SettingsNav.vue` + `sections.ts`）

侧栏固定宽 256（`w-64`），内容列 x 257–998，中间 1px 分隔线 x=228（png 实测）。导航 19 项与顺序取自 `sections.ts` 的 `SETTINGS_SECTION_ORDER`，不按 UI 模式过滤。

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-shell-01 | 返回 | 按钮（图标 + 文字） | 侧栏可见：`v-show="!isCompact || !compactShowingDetail"` | `SettingsNav.vue:53-62` | 可见 | 15,59 202×28 |
| S-shell-02 | ↳ 返回图标 ArrowLeft | 图标 | 同上 | `SettingsNav.vue:60` | 可见（icon:true 于 S-shell-01） | 同上 |
| S-shell-03 | 账户 | 导航按钮 | `v-for="item in sections"`（恒 19 项）；当前节 `disabled` | `SettingsNav.vue:67-85` · `sections.ts:57` | 可见（本节高亮） | 12,105 209×35 |
| S-shell-04 | ↳ 账户图标 UserCircle | 图标 | 同上 | `SettingsNav.vue:83` · `sections.ts:57` | 可见 | 同上 |
| S-shell-05 | 用量 | 导航按钮 | 同上 | `sections.ts:58` | 可见 | 12,140 209×35 |
| S-shell-06 | ↳ 用量图标 BarChart3 | 图标 | 同上 | 同上 | 可见 | 同上 |
| S-shell-07 | 个性化 | 导航按钮 | 同上（默认落地节） | `sections.ts:59` | 可见 | 12,175 209×35 |
| S-shell-08 | ↳ 个性化图标 Sparkles | 图标 | 同上 | 同上 | 可见 | 同上 |
| S-shell-09 | 外观 | 导航按钮 | 同上 | `sections.ts:60` | 可见 | 12,210 209×35 |
| S-shell-10 | ↳ 外观图标 Palette | 图标 | 同上 | 同上 | 可见 | 同上 |
| S-shell-11 | 通用 | 导航按钮 | 同上 | `sections.ts:61` | 可见 | 12,245 209×35 |
| S-shell-12 | ↳ 通用图标 Settings2 | 图标 | 同上 | 同上 | 可见 | 同上 |
| S-shell-13 | 快捷键 | 导航按钮 | 同上 | `sections.ts:62` | 可见 | 12,280 209×35 |
| S-shell-14 | ↳ 快捷键图标 Keyboard | 图标 | 同上 | 同上 | 可见 | 同上 |
| S-shell-15 | 模型供应 | 导航按钮 | 同上 | `sections.ts:63` | 可见 | 12,315 209×35 |
| S-shell-16 | ↳ 模型供应图标 Brain | 图标 | 同上 | 同上 | 可见 | 同上 |
| S-shell-17 | 多模态 | 导航按钮 | 同上 | `sections.ts:64` | 可见 | 12,350 209×35 |
| S-shell-18 | ↳ 多模态图标 SlidersHorizontal | 图标 | 同上 | 同上 | 可见 | 同上 |
| S-shell-19 | 运行环境 | 导航按钮 | 同上 | `sections.ts:65` | 可见 | 12,385 209×35 |
| S-shell-20 | ↳ 运行环境图标 SquareTerminal | 图标 | 同上 | 同上 | 可见 | 同上 |
| S-shell-21 | MCP | 导航按钮 | 同上 | `sections.ts:66` | 可见 | 12,420 209×35 |
| S-shell-22 | ↳ MCP 图标 Wrench | 图标 | 同上 | 同上 | 可见 | 同上 |
| S-shell-23 | 通知 | 导航按钮 | 同上 | `sections.ts:67` | 可见 | 12,455 209×35 |
| S-shell-24 | ↳ 通知图标 Bell | 图标 | 同上 | 同上 | 可见 | 同上 |
| S-shell-25 | 宠物 | 导航按钮 | 同上 | `sections.ts:68` | 可见 | 12,490 209×35 |
| S-shell-26 | ↳ 宠物图标 PawPrint | 图标 | 同上 | 同上 | 可见 | 同上 |
| S-shell-27 | 权限 | 导航按钮 | 同上 | `sections.ts:69` | 可见 | 12,525 209×35 |
| S-shell-28 | ↳ 权限图标 ShieldCheck | 图标 | 同上 | 同上 | 可见 | 同上 |
| S-shell-29 | Git | 导航按钮 | 同上 | `sections.ts:70` | 可见 | 12,560 209×35 |
| S-shell-30 | ↳ Git 图标 GitBranch | 图标 | 同上 | 同上 | 可见 | 同上 |
| S-shell-31 | Hooks | 导航按钮 | 同上 | `sections.ts:71` | 可见 | 12,595 209×35 |
| S-shell-32 | ↳ Hooks 图标 Webhook | 图标 | 同上 | 同上 | 可见 | 同上 |
| S-shell-33 | 插件设置 | 导航按钮 | 同上 | `sections.ts:72` | 可见 | 12,630 209×35 |
| S-shell-34 | ↳ 插件设置图标 Plug | 图标 | 同上 | 同上 | 可见 | 同上 |
| S-shell-35 | 外部 Agent | 导航按钮 | 同上 | `sections.ts:73` | 可见 | 12,665 209×35 |
| S-shell-36 | ↳ 外部 Agent 图标 Bot | 图标 | 同上 | 同上 | 可见 | 同上 |
| S-shell-37 | 归档会话 | 导航按钮 | 同上 | `sections.ts:74` | 可见 | 12,700 209×35 |
| S-shell-38 | ↳ 归档会话图标 Archive | 图标 | 同上 | 同上 | 可见 | 同上 |
| S-shell-39 | 关于 | 导航按钮 | 同上 | `sections.ts:75` | 可见 | 12,735 209×35 |
| S-shell-40 | ↳ 关于图标 Info | 图标 | 同上 | 同上 | 可见 | 同上 |
| S-shell-41 | 返回 | 图标按钮（窄屏工具栏） | `isCompact` 且 `compactShowingDetail` | `SettingsView.vue:177-193` | 未出现（条件：窗口非窄屏） | — |
| S-shell-42 | ↳ 返回图标 ArrowLeft | 图标 | 同上 | `SettingsView.vue:191` | 未出现（条件：窗口非窄屏） | — |

不进表的非感知节点：`legacy-titlebar-spacer` 空 div（`SettingsView.vue:161`）；内容区 `ScrollArea` / `KeepAlive` / `component :is` 派发（`SettingsView.vue:194-215`）；侧栏滚动容器（`SettingsNav.vue:66`）。

本节行数 42（图标 21）｜可见 40｜未出现（条件）2｜未命中 0｜运行时独有 0
---

## 1. 账户（`views/settings/account/AccountSettings.vue`，+ `AccountSignedIn` / `AccountSignedOut` / `AccountPlanUsage`）

png：[账户-top.png](runtime/settings/账户-top.png)、[账户-bottom.png](runtime/settings/账户-bottom.png)。运行时 174 节点合成 4 趟（侧栏 40 + 内容 48 + 侧栏 38 + 内容 48），内容首趟 y 33–807 即文档坐标，去重后 47 个节点。

### 1.1 页首（`AccountSettings.vue`）

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-account-01 | 账户 | 页面标题 | 常显 | `AccountSettings.vue:39` | 可见 | 257,33 741×25 |
| S-account-02 | 到期横幅 | 容器·提示条 | `showExpiredBanner`（`isSignedIn && isPlanExpired`） | `AccountSettings.vue:42` | 未出现（条件：套餐未到期） | — |
| S-account-03 | ↳ 警告图标 TriangleAlert | 图标 | 同上 | `AccountSettings.vue:49` | 未出现（条件） | — |
| S-account-04 | 套餐已到期，Dim 托管模型暂不可用 | 文本（role=status 标题） | 同上 | `AccountSettings.vue:51-53` | 未出现（条件） | — |
| S-account-05 | Dim 托管模型权益已停止，订阅套餐后即可恢复使用。 | 文本 | 同上 | `AccountSettings.vue:54-56` | 未出现（条件） | — |
| S-account-06 | 选购套餐 | 按钮 | 同上 | `AccountSettings.vue:58` | 未出现（条件） | — |
| S-account-07 | ↳ ShoppingCart 图标 | 图标 | 同上 | `AccountSettings.vue:59` | 未出现（条件） | — |

### 1.2 身份卡（`AccountSignedIn.vue`，祖先条件 `isSignedIn`）

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-account-08 | 身份卡 | 容器·卡片 | `isSignedIn`（`AccountSettings.vue:64`） | `AccountSignedIn.vue:48` | 可见（png 边框 y 74–170） | 257,74 740×96 |
| S-account-09 | 头像 initials | 容器·头像 | `isSignedIn` | `AccountSignedIn.vue:58-62` | 可见 | 275,101 42×42 |
| S-account-10 | displayName（eric） | 文本 | `isSignedIn` | `AccountSignedIn.vue:64-66` | 可见 | 331,93 396×18 |
| S-account-11 | email（chenghan@fazhitech.cn） | 文本 | `isSignedIn` + `v-if="email"` | `AccountSignedIn.vue:67-69` | 可见 | 331,111 396×14 |
| S-account-12 | UID | 文本标签 | `isSignedIn` + `v-if="uid"` | `AccountSignedIn.vue:70-73` | 可见 | 331,133 19×17 |
| S-account-13 | uid（u_9560e10671368346） | 文本·值（code） | 同上 | `AccountSignedIn.vue:74-77` | 可见 | 355,130 131×22 |
| S-account-14 | 复制 UID | 图标按钮 | 同上 | `AccountSignedIn.vue:78-85` | 可见 | 492,132 18×18 |
| S-account-15 | ↳ Copy 图标 | 图标 | 同上 | `AccountSignedIn.vue:86` | 可见（icon:true 于 S-account-14） | 同上 |
| S-account-16 | 查看套餐 | 按钮 | `isSignedIn` | `AccountSignedIn.vue:97-111` | 可见 | 741,108 157×28 |
| S-account-17 | ↳ ExternalLink 图标 | 图标 | 同上 | `AccountSignedIn.vue:103` | 可见（icon:true 于 S-account-16） | 同上 |
| S-account-18 | · {count} 个模型（32 个模型） | 文本 | 同上 + `v-if="modelCount > 0"` | `AccountSignedIn.vue:105-109` | 可见 | 834,118 55×9 |
| S-account-19 | 退出登录 | 按钮（`disabled`） | `isSignedIn` | `AccountSignedIn.vue:112-121` | 可见 | 909,108 70×28 |
| S-account-20 | account.error.message | 文本 | `isSignedIn` + `v-if="account.error"` | `AccountSignedIn.vue:124-126` | 未出现（条件：无错误） | — |

### 1.3 未登录分支（`AccountSignedOut.vue`，祖先条件 `v-else`）

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-account-21 | 未登录卡 | 容器·卡片 | `!isSignedIn`（`AccountSettings.vue:65`） | `AccountSignedOut.vue:27` | 未出现（条件：已登录） | — |
| S-account-22 | 登录 Dim | 分组标题 | 同上 | `AccountSignedOut.vue:29` | 未出现（条件） | — |
| S-account-23 | 登录后可解锁 Dim 托管模型。未登录时仍可在本地使用 DimAgent。 | 文本 | 同上 | `AccountSignedOut.vue:32` | 未出现（条件） | — |
| S-account-24 | 正在登录… | 按钮（含 spinner） | 同上 + `v-if="signingIn"` | `AccountSignedOut.vue:36-38` | 未出现（条件） | — |
| S-account-25 | ↳ Loader2 图标 | 图标 | 同上 | `AccountSignedOut.vue:37` | 未出现（条件） | — |
| S-account-26 | 登录 | 按钮 | 同上 + `v-else`（`!signingIn`） | `AccountSignedOut.vue:39` | 未出现（条件） | — |
| S-account-27 | 登录企业账号 | 按钮 | 同上 | `AccountSignedOut.vue:41-48` | 未出现（条件） | — |
| S-account-28 | account.error.message | 文本 | 同上 + `v-if="account.error"` | `AccountSignedOut.vue:49-51` | 未出现（条件） | — |
| S-account-29 | 邮箱登录对话框 | 对话框（`AccountEmailLoginDialog`） | 同上（组件内部按点击打开） | `AccountSignedOut.vue:53` | 未出现（条件） | — |

### 1.4 套餐用量卡（`AccountPlanUsage.vue`，祖先条件 `isSignedIn`）

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-account-30 | 套餐用量 | 容器·卡片 | `isSignedIn` | `AccountPlanUsage.vue:425` | 可见（png 边框 y 188–564） | 257,188 740×376 |
| S-account-31 | 套餐用量 | 分组标题 | 同上 | `AccountPlanUsage.vue:427-429` | 可见 | 275,205 49×18 |
| S-account-32 | 刷新 | 图标按钮（`:disabled="refreshing"`） | 同上 | `AccountPlanUsage.vue:430-438` | 可见 | 951,199 28×28 |
| S-account-33 | ↳ RefreshCw 图标 | 图标 | 同上 | `AccountPlanUsage.vue:439` | 可见（icon:true 于 S-account-32） | 同上 |
| S-account-34 | error | 文本 | 同上 + `v-if="error"` | `AccountPlanUsage.vue:443-445` | 未出现（条件：无错误） | — |
| S-account-35 | 正在加载用量… | 文本 | 同上 + `v-if="loading && !usage"` | `AccountPlanUsage.vue:447-449` | 未出现（条件：已加载） | — |
| S-account-36 | 套餐标识 BrandMark | 容器·图标块 | 同上 + `v-else-if="usage || !error"` | `AccountPlanUsage.vue:458-464` | 可见 | 275,256 32×32 |
| S-account-37 | DimAgent | 文本 | 同上 | `AccountPlanUsage.vue:466` | 可见 | 315,263 59×18 |
| S-account-38 | planSubtitle（法至内部使用 · ¥0.10 / 月） | 文本 | 同上 + `v-if="planSubtitle"` | `AccountPlanUsage.vue:467-469` | 可见 | 383,265 117×14 |
| S-account-39 | statusLabel（生效中） | 状态·徽标 | 同上 + `v-if="statusLabel"` | `AccountPlanUsage.vue:470-479` | 可见 | 509,262 47×20 |
| S-account-40 | ↳ CircleAlert 图标 | 图标 | 同上 + `v-if="isPlanExpired"` | `AccountPlanUsage.vue:477` | 未出现（条件：套餐未到期） | — |
| S-account-41 | 到期/已于 … 到期（到期 2026年12月13日 · 还剩 26 天） | 文本 | 同上 + `v-if="expiryDateLabel"`；`isPlanExpired` 分两支 | `AccountPlanUsage.vue:480-488` | 可见（`v-else` 支） | 810,265 169×14 |

### 1.5 额度用量切换行（`resetsVisible`）

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-account-42 | 额度用量 | 控件·分段项（选中） | 同上 + `v-if="resetsVisible"` | `AccountPlanUsage.vue:498-505` | 可见 | 279,305 62×24 |
| S-account-43 | 重置当月用量（`row.label`） | 按钮（`v-for="row in resetRows"`；`disabled`） | 同上 | `AccountPlanUsage.vue:506-526` | 可见（1 行实例） | 344,305 123×24 |
| S-account-44 | ↳ RotateCcw 图标 | 图标 | 同上 | `AccountPlanUsage.vue:519` | 可见（png；JSON 未把该 svg 标 icon:true） | 同上 |
| S-account-45 | 剩余次数（`row.availableCount`，0） | 文本·计数 | 同上 | `AccountPlanUsage.vue:527-535` | 可见 | 439,309 18×17 |
| S-account-46 | 重置确认对话框 | 对话框（`AlertDialogContent`） | 同上 + 点击 S-account-43 | `AccountPlanUsage.vue:538-585` | 未出现（条件：对话框未打开） | — |
| S-account-47 | `row.confirmTitle` | 对话框标题 | 同上 | `AccountPlanUsage.vue:540` | 未出现（条件） | — |
| S-account-48 | 将消耗 1 次重置次数。 | 对话框说明 | 同上 | `AccountPlanUsage.vue:541-543` | 未出现（条件） | — |
| S-account-49 | `batchLabel(batch)` | 对话框列表项（`v-for="batch in resetBatches"`） | 同上 + `resetBatches.length > 0` | `AccountPlanUsage.vue:558-561` | 未出现（条件） | — |
| S-account-50 | 暂无可用 | 对话框文本 | 同上 + `v-else-if="resetBatches"` | `AccountPlanUsage.vue:563-565` | 未出现（条件） | — |
| S-account-51 | 无法加载额度明细 | 对话框文本 | 同上 + `v-else-if="resetBatchesFailed"` | `AccountPlanUsage.vue:566-572` | 未出现（条件） | — |
| S-account-52 | 取消 | 对话框按钮 | 同上 | `AccountPlanUsage.vue:575` | 未出现（条件） | — |
| S-account-53 | 使用 | 对话框按钮（`disabled`） | 同上 | `AccountPlanUsage.vue:576-581` | 未出现（条件） | — |

### 1.6 额度三支（耗尽 / 不限量 / 有额度）

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-account-54 | 套餐额度已用完 提示条 | 容器·提示条（role=status） | 同上 + `showDepletedNotice`（有套餐且用量 100% 且未到期） | `AccountPlanUsage.vue:587` | 未出现（条件：未用尽） | — |
| S-account-55 | ↳ 警告图标 CircleAlert | 图标 | 同上 | `AccountPlanUsage.vue:594` | 未出现（条件） | — |
| S-account-56 | 套餐额度已用完 | 文本 | 同上 | `AccountPlanUsage.vue:596-598` | 未出现（条件） | — |
| S-account-57 | 当前套餐额度已全部使用，Dim 托管模型暂时不可用。 | 文本 | 同上 | `AccountPlanUsage.vue:599-601` | 未出现（条件） | — |
| S-account-58 | 额度用量（不限量） | 文本标签 | 同上 + `v-else-if="isUnlimited"` | `AccountPlanUsage.vue:607-610` | 未出现（条件：非不限量套餐） | — |
| S-account-59 | ↳ CalendarDays 图标 | 图标 | 同上 | `AccountPlanUsage.vue:608` | 未出现（条件） | — |
| S-account-60 | 不限量 | 文本·值 | 同上 | `AccountPlanUsage.vue:611-613` | 未出现（条件） | — |
| S-account-61 | 当前套餐不限用量，无额度上限。 | 文本 | 同上 | `AccountPlanUsage.vue:615-617` | 未出现（条件） | — |
| S-account-62 | 月度额度卡 | 容器·卡片 | 同上 + `v-else-if="creditsAvailable"` | `AccountPlanUsage.vue:625` | 可见（png 边框 y 346–416） | 275,346 704×70 |
| S-account-63 | 月度额度 | 文本标签 | 同上 | `AccountPlanUsage.vue:626-630` | 可见 | 290,360 60×14 |
| S-account-64 | ↳ CalendarDays 图标 | 图标 | 同上 | `AccountPlanUsage.vue:628` | 可见（png） | 上述标签左侧 |
| S-account-65 | {date} 重置（2026年12月13日 重置） | 文本 | 同上 + `v-if="creditResetLabel"` | `AccountPlanUsage.vue:631-633` | 可见 | 849,361 115×17 |
| S-account-66 | monthlyAmountText（10.8万 / 30万） | 文本·值 | 同上 + `v-if="monthlyAmountText"` | `AccountPlanUsage.vue:636-638` | 可见 | 290,386 133×18 |
| S-account-67 | 已使用 36%（进度条，`aria-label`） | 状态·进度条 | 同上 + `v-if="!meter.unlimited"` 对应月度额度 | `AccountPlanUsage.vue:639-647` | 可见 | 434,392 450×5 |
| S-account-68 | 已使用 {pct}%（36%） | 文本·值 | 同上 | `AccountPlanUsage.vue:657-664` | 可见 | 894,388 70×14 |

### 1.7 周期限制窗口（`windowRows`）与功能额度（`featureMeterRows`）

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-account-69 | 周期限制 | 文本标签 | 同上 + `creditsAvailable` + `windowRows.length > 0` | `AccountPlanUsage.vue:673-679` | 未出现（条件：无滚动窗口额度） | — |
| S-account-70 | 以下窗口包含在月度额度内 | 文本 | 同上 | `AccountPlanUsage.vue:680` | 未出现（条件） | — |
| S-account-71 | `row.label` | 文本标签（`v-for="row in windowRows"`） | 同上 | `AccountPlanUsage.vue:682-687` | 未出现（条件） | — |
| S-account-72 | `row.amountText` | 文本·值 | 同上 + `v-if="row.amountText"` | `AccountPlanUsage.vue:688-690` | 未出现（条件） | — |
| S-account-73 | 已使用 {pct}%（窗口进度条） | 状态·进度条 | 同上 | `AccountPlanUsage.vue:691-699` | 未出现（条件） | — |
| S-account-74 | 已使用 {pct}%（窗口百分比） | 文本·值 | 同上 | `AccountPlanUsage.vue:709-714` | 未出现（条件） | — |
| S-account-75 | `row.resetLabel` | 文本 | 同上 + `v-if="row.resetLabel"` | `AccountPlanUsage.vue:715-717` | 未出现（条件） | — |
| S-account-76 | （无月度额度时的窗口行）`row.label` | 文本标签 | 同上 + `v-else-if="windowRows.length > 0"` + `v-for="(row, index) in windowRows"` | `AccountPlanUsage.vue:726-736` | 未出现（条件：有月度额度） | — |
| S-account-77 | `row.amountText` | 文本·值 | 同上 + `v-if="row.amountText"` | `AccountPlanUsage.vue:737-739` | 未出现（条件） | — |
| S-account-78 | 已使用 {pct}%（进度条） | 状态·进度条 | 同上 | `AccountPlanUsage.vue:740-748` | 未出现（条件） | — |
| S-account-79 | 已使用 {pct}%（百分比） | 文本·值 | 同上 | `AccountPlanUsage.vue:758-763` | 未出现（条件） | — |
| S-account-80 | `row.resetLabel` | 文本 | 同上 + `v-if="row.resetLabel"` | `AccountPlanUsage.vue:764-766` | 未出现（条件） | — |
| S-account-81 | Web Search 卡 | 容器·卡片 | 同上 + `featureMeterRows.length > 0` + `v-for="meter in featureMeterRows"` | `AccountPlanUsage.vue:775-780` | 可见（png 边框 y 430–500） | 275,430 704×70 |
| S-account-82 | ↳ Globe 图标 | 图标 | 同上 | `AccountPlanUsage.vue:783` | 可见（png） | 上述标签左侧 |
| S-account-83 | `meter.label`（Web Search） | 文本标签 | 同上 | `AccountPlanUsage.vue:785` | 可见 | 290,444 78×14 |
| S-account-84 | `meter.resetLabel`（2026年12月13日 重置） | 文本 | 同上 + `v-if="meter.resetLabel"` | `AccountPlanUsage.vue:786-788` | 可见 | 849,445 115×17 |
| S-account-85 | `meter.amountText`（334 / 1,500 次） | 文本·值 | 同上 + `v-if="meter.amountText"` | `AccountPlanUsage.vue:791-793` | 可见 | 290,470 133×18 |
| S-account-86 | 已使用 22%（功能进度条） | 状态·进度条 | 同上 + `v-if="!meter.unlimited"` | `AccountPlanUsage.vue:794-803` | 可见 | 434,476 450×5 |
| S-account-87 | 已使用 {pct}%（22%） | 文本·值 | 同上 | `AccountPlanUsage.vue:813-820` | 可见 | 894,472 70×14 |

### 1.8 未订阅 / 空 / 续订

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-account-88 | 尚未订阅套餐 | 文本 | 同上 + `v-if="isUnsubscribed"` | `AccountPlanUsage.vue:826-830` | 未出现（条件：已订阅） | — |
| S-account-89 | 订阅套餐后即可解锁 Dim 托管模型。 | 文本 | 同上 | `AccountPlanUsage.vue:831-833` | 未出现（条件） | — |
| S-account-90 | 查看套餐 | 按钮 | 同上 | `AccountPlanUsage.vue:835-842` | 未出现（条件） | — |
| S-account-91 | ↳ ExternalLink 图标 | 图标 | 同上 | `AccountPlanUsage.vue:841` | 未出现（条件） | — |
| S-account-92 | 暂无用量数据 | 文本 | 同上 + `v-else-if="!plan && !creditsAvailable"` | `AccountPlanUsage.vue:846-848` | 未出现（条件：有用量数据） | — |
| S-account-93 | 续订 / 升级（`isPlanExpired ? 选购套餐 : 续订 / 升级`） | 按钮 | 同上 + `v-if="plan"` | `AccountPlanUsage.vue:850-861` | 可见（`v-else` 支） | 882,519 97×28 |
| S-account-94 | ↳ ShoppingCart 图标 | 图标 | 同上 + `v-if="isPlanExpired"` | `AccountPlanUsage.vue:857` | 未出现（条件：未到期） | — |
| S-account-95 | ↳ ArrowUpRight 图标 | 图标 | 同上 + `v-else` | `AccountPlanUsage.vue:858` | 可见（png 可见；JSON 未标 icon） | 上述按钮内 |

### 1.9 推广邀请（`v-if="referral?.eligible"`）

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-account-96 | 推广邀请卡 | 容器·卡片 | `isSignedIn` + `v-if="referral?.eligible"` | `AccountPlanUsage.vue:865` | 可见（png 上边 y 582） | 257,582 740×≥216 |
| S-account-97 | ↳ Gift 图标 | 图标 | 同上 | `AccountPlanUsage.vue:871` | 可见（png） | 296,599 左侧 |
| S-account-98 | 推广邀请 | 分组标题 | 同上 | `AccountPlanUsage.vue:872` | 可见 | 296,599 49×18 |
| S-account-99 | `inviterRuleLabel`（好友完成首次订阅付费后…） | 文本 | 同上 + `v-if="inviterRuleLabel"` | `AccountPlanUsage.vue:875-877` | 可见 | 275,650 704×14 |
| S-account-100 | `inviteeRuleLabel`（好友首次订阅可获得…） | 文本 | 同上 + `v-if="inviteeRuleLabel"` | `AccountPlanUsage.vue:878-880` | 可见 | 275,664 704×14 |
| S-account-101 | `referralCampaignOffLabel` | 文本 | 同上 + `v-if="referralCampaignOffLabel"` | `AccountPlanUsage.vue:881-883` | 未出现（条件：活动进行中） | — |
| S-account-102 | 邀请码 | 文本标签 | 同上 + `v-if="referral?.affCode"` | `AccountPlanUsage.vue:891-894` | 可见 | 275,690 533×17 |
| S-account-103 | affCode（LYIr） | 文本·值（code） | 同上 | `AccountPlanUsage.vue:895-896` | 可见 | 287,720 449×18 |
| S-account-104 | 复制 | 按钮 | 同上 | `AccountPlanUsage.vue:897-904` | 可见 | 743,717 59×25 |
| S-account-105 | ↳ Copy 图标 | 图标 | 同上 | `AccountPlanUsage.vue:903` | 可见（png） | 上述按钮内 |
| S-account-106 | 邀请网址 | 文本标签 | 同上 + `v-if="inviteUrl"` | `AccountPlanUsage.vue:908-911` | 可见 | 275,756 533×17 |
| S-account-107 | inviteUrl（https://dimagent.cn/register?aff=LYIr） | 文本·值（code） | 同上 | `AccountPlanUsage.vue:912-913` | 可见 | 287,788 449×14 |
| S-account-108 | 复制 | 按钮 | 同上 | `AccountPlanUsage.vue:914-921` | 可见 | 743,782 59×25 |
| S-account-109 | ↳ Copy 图标 | 图标 | 同上 | `AccountPlanUsage.vue:920` | 可见（png） | 上述按钮内 |
| S-account-110 | 已邀请 · {count} 人（11 人） | 文本·指标（dt + dd） | 同上 | `AccountPlanUsage.vue:930-933` | 可见 | 840,702 33×17（值 948,701 31×18） |
| S-account-111 | 已付费 · {count} 人（6 人） | 文本·指标（dt + dd） | 同上 | `AccountPlanUsage.vue:936-939` | 可见 | 840,743 33×17（值 956,742 23×18） |
| S-account-112 | 累计奖励 · `referralRewardLabel`（9,125 Credits） | 文本·指标（dt + dd） | 同上 + `v-if="referralRewardLabel"` | `AccountPlanUsage.vue:945-946` | 可见 | 840,784 44×17（值 898,784 81×18） |

本节行数 115（图标 20）｜可见 44｜未出现（条件）71｜未命中 0｜运行时独有 0

运行时对齐：47 个内容节点全部落在表内（44 可见行 + 3 个图标行共用宿主包围盒的重复计数），无模板外节点。
---

## 2. 用量（`components/settings/UsageSettings.vue` + `views/settings/account/AccountActivity.vue`）

png：[用量-top.png](runtime/settings/用量-top.png)、[用量-bottom.png](runtime/settings/用量-bottom.png)。运行时 81 节点 = 侧栏 40 + 内容 3 + 侧栏 38；内容全部为 `appServerRead` 分支。

`AccountActivity.vue` 的入口分支链：`appServerRead` → `usageUnavailable` → `loading` → `error` → 其余（`v-else`）。

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-usage-01 | 用量 | 页面标题 | 常显 | `UsageSettings.vue:16` | 可见 | 257,33 741×25 |
| S-usage-02 | 不可用提示卡（App Server 只读） | 容器·卡片 | `appServerRead` | `AccountActivity.vue:461` | 可见（png 边框 y 81–163） | 257,81 740×82 |
| S-usage-03 | ↳ 警告图标 AlertTriangle | 图标 | 同上 | `AccountActivity.vue:463` | 可见（png） | 卡片左侧 |
| S-usage-04 | 不可用 | 分组标题 | 同上 | `AccountActivity.vue:465` | 可见 | 307,104 347×18 |
| S-usage-05 | 目前只有按会话的模型/用量；聚合结果不完整且不是权威数据。 | 文本 | 同上 | `AccountActivity.vue:466` | 可见 | 307,125 347×18 |
| S-usage-06 | 不可用提示卡（需连接） | 容器·卡片 | `!appServerRead` + `usageUnavailable` | `AccountActivity.vue:471-476` | 未出现（条件：当前为 App Server 只读模式） | — |
| S-usage-07 | ↳ 警告图标 AlertTriangle | 图标 | 同上 | `AccountActivity.vue:477` | 未出现（条件） | — |
| S-usage-08 | 不可用 | 分组标题 | 同上 | `AccountActivity.vue:479` | 未出现（条件） | — |
| S-usage-09 | 查看用量活动需要连接 App Server。 | 文本 | 同上 | `AccountActivity.vue:480` | 未出现（条件） | — |
| S-usage-10 | 正在加载... | 文本 | `!appServerRead` + `!usageUnavailable` + `loading` | `AccountActivity.vue:485-487` | 未出现（条件：不加载） | — |
| S-usage-11 | 用量加载失败 | 容器·卡片 | 同上 + `!loading` + `error` | `AccountActivity.vue:489-490` | 未出现（条件：无错误） | — |
| S-usage-12 | ↳ 警告图标 AlertTriangle | 图标 | 同上 | `AccountActivity.vue:491` | 未出现（条件） | — |
| S-usage-13 | 用量加载失败 | 分组标题 | 同上 | `AccountActivity.vue:493` | 未出现（条件） | — |
| S-usage-14 | error | 文本 | 同上 | `AccountActivity.vue:494` | 未出现（条件） | — |
| S-usage-15 | 后台导入提示 | 文本容器 | 同上 + `v-else` + `isBackfilling` | `AccountActivity.vue:500-502` | 未出现（条件） | — |
| S-usage-16 | 正在后台导入历史用量。 | 文本 | 同上 | `AccountActivity.vue:502` | 未出现（条件） | — |
| S-usage-17 | 导入进度（`backfillProgressText`） | 文本 | 同上 + `v-if="backfillProgressText"` | `AccountActivity.vue:503` | 未出现（条件） | — |
| S-usage-18 | 历史用量导入失败，新的用量仍会继续显示。 | 文本 | 同上 + `backfillFailed` | `AccountActivity.vue:506-508` | 未出现（条件） | — |
| S-usage-19 | 概览指标卡（`overviewMetrics`，v-for） | 容器·指标卡 | 同上 | `AccountActivity.vue:513-518` | 未出现（条件） | — |
| S-usage-20 | ↳ 指标图标（`<component :is>`） | 图标 | 同上 | `AccountActivity.vue:518` | 未出现（条件） | — |
| S-usage-21 | `metric.value` | 文本·值 | 同上 | `AccountActivity.vue:519` | 未出现（条件） | — |
| S-usage-22 | `metric.label` | 文本·标签 | 同上 | `AccountActivity.vue:520` | 未出现（条件） | — |
| S-usage-23 | Token 活跃度卡 | 容器·卡片 | 同上 | `AccountActivity.vue:524` | 未出现（条件） | — |
| S-usage-24 | Token 活跃度 | 分组标题 | 同上 | `AccountActivity.vue:527-529` | 未出现（条件） | — |
| S-usage-25 | 部分数据 · 仅 App Server 会话 | 状态·徽标 | 同上（组件内 `appServerRead`） | `AccountActivity.vue:529-532` | 未出现（条件） | — |
| S-usage-26 | `visibleRange.fromDay - visibleRange.toDay` | 文本 | 同上 | `AccountActivity.vue:534` | 未出现（条件） | — |
| S-usage-27 | 刷新 | 图标按钮（`:disabled="loading"`） | 同上 | `AccountActivity.vue:536-547` | 未出现（条件） | — |
| S-usage-28 | ↳ RefreshCw 图标 | 图标 | 同上 | `AccountActivity.vue:546` | 未出现（条件） | — |
| S-usage-29 | 还没有用量 | 文本 | 同上 + `isEmpty` | `AccountActivity.vue:551-552` | 未出现（条件） | — |
| S-usage-30 | 完成运行后这里会显示用量。 | 文本 | 同上 | `AccountActivity.vue:553` | 未出现（条件） | — |
| S-usage-31 | 月份标签（`v-for="label in monthLabels"`） | 文本 | 同上 + `v-else`（`!isEmpty`） | `AccountActivity.vue:558` | 未出现（条件） | — |
| S-usage-32 | 活跃度热力格（`v-for="cell in heatmapCells"`） | 单元格（`aria-label`/`title`） | 同上 | `AccountActivity.vue:570` | 未出现（条件） | — |
| S-usage-33 | 活跃度指标 | 容器·卡片 | 同上 | `AccountActivity.vue:584` | 未出现（条件） | — |
| S-usage-34 | 活跃度指标 | 分组标题 | 同上 | `AccountActivity.vue:585` | 未出现（条件） | — |
| S-usage-35 | 总会话数 · 值（`overview.totalThreads`） | 文本·指标（dt + dd） | 同上 | `AccountActivity.vue:587-589` | 未出现（条件） | — |
| S-usage-36 | 总运行数 · 值（`overview.totalRuns`） | 文本·指标（dt + dd） | 同上 | `AccountActivity.vue:591-593` | 未出现（条件） | — |
| S-usage-37 | 平均 tokens / 运行 · 值 | 文本·指标（dt + dd） | 同上 | `AccountActivity.vue:595-597` | 未出现（条件） | — |
| S-usage-38 | 最长任务 · 值（`formatDuration`） | 文本·指标（dt + dd） | 同上 | `AccountActivity.vue:599-601` | 未出现（条件） | — |
| S-usage-39 | Token 指标 | 容器·卡片 | 同上 | `AccountActivity.vue:606` | 未出现（条件） | — |
| S-usage-40 | Token 指标 | 分组标题 | 同上 | `AccountActivity.vue:607` | 未出现（条件） | — |
| S-usage-41 | 输入 tokens · 值 | 文本·指标（dt + dd） | 同上 | `AccountActivity.vue:609-611` | 未出现（条件） | — |
| S-usage-42 | 输出 tokens · 值 | 文本·指标（dt + dd） | 同上 | `AccountActivity.vue:613-615` | 未出现（条件） | — |
| S-usage-43 | 缓存读取 tokens · 值 | 文本·指标（dt + dd） | 同上 | `AccountActivity.vue:617-619` | 未出现（条件） | — |
| S-usage-44 | 缓存写入 tokens · 值 | 文本·指标（dt + dd） | 同上 | `AccountActivity.vue:621-623` | 未出现（条件） | — |
| S-usage-45 | Cache 指标 | 容器·卡片 | 同上 | `AccountActivity.vue:628` | 未出现（条件） | — |
| S-usage-46 | Cache 指标 | 分组标题 | 同上 | `AccountActivity.vue:629` | 未出现（条件） | — |
| S-usage-47 | 缓存 tokens · 值 | 文本·指标（dt + dd） | 同上 | `AccountActivity.vue:631-633` | 未出现（条件） | — |
| S-usage-48 | Cache 命中率 · 值 | 文本·指标（dt + dd） | 同上 | `AccountActivity.vue:635-637` | 未出现（条件） | — |
| S-usage-49 | 最常用 skills | 容器·卡片 | 同上 | `AccountActivity.vue:644` | 未出现（条件） | — |
| S-usage-50 | 最常用 skills | 分组标题 | 同上 | `AccountActivity.vue:646` | 未出现（条件） | — |
| S-usage-51 | `topSkillsPageRange` | 文本·分页信息 | 同上 + `topSkills.length > 0` | `AccountActivity.vue:647-650` | 未出现（条件） | — |
| S-usage-52 | 上一页 | 图标按钮 | 同上 | `AccountActivity.vue:652-661` | 未出现（条件） | — |
| S-usage-53 | ↳ ChevronLeft 图标 | 图标 | 同上 | `AccountActivity.vue:662` | 未出现（条件） | — |
| S-usage-54 | 下一页 | 图标按钮 | 同上 | `AccountActivity.vue:664-673` | 未出现（条件） | — |
| S-usage-55 | ↳ ChevronRight 图标 | 图标 | 同上 | `AccountActivity.vue:674` | 未出现（条件） | — |
| S-usage-56 | 这个范围内还没有使用 skills。 | 文本 | 同上 + `topSkills.length === 0` | `AccountActivity.vue:679-681` | 未出现（条件） | — |
| S-usage-57 | `$skill.skillName` | 文本·值 | 同上 + `v-else` + `v-for="skill in visibleTopSkills"` | `AccountActivity.vue:690-697` | 未出现（条件） | — |
| S-usage-58 | {count} 次（`skill.runCount`） | 文本 | 同上 | `AccountActivity.vue:698` | 未出现（条件） | — |
| S-usage-59 | 模型用量 | 容器·卡片 | 同上 | `AccountActivity.vue:704` | 未出现（条件） | — |
| S-usage-60 | 模型用量 | 分组标题 | 同上 | `AccountActivity.vue:706` | 未出现（条件） | — |
| S-usage-61 | `modelUsagePageRange` | 文本·分页信息 | 同上 + `modelUsage.length > 0` | `AccountActivity.vue:707-710` | 未出现（条件） | — |
| S-usage-62 | 上一页 | 图标按钮 | 同上 | `AccountActivity.vue:712-721` | 未出现（条件） | — |
| S-usage-63 | ↳ ChevronLeft 图标 | 图标 | 同上 | `AccountActivity.vue:722` | 未出现（条件） | — |
| S-usage-64 | 下一页 | 图标按钮 | 同上 | `AccountActivity.vue:724-733` | 未出现（条件） | — |
| S-usage-65 | ↳ ChevronRight 图标 | 图标 | 同上 | `AccountActivity.vue:734` | 未出现（条件） | — |
| S-usage-66 | 这个范围内还没有模型用量。 | 文本 | 同上 + `modelUsage.length === 0` | `AccountActivity.vue:739-741` | 未出现（条件） | — |
| S-usage-67 | `model.modelId` | 文本·值 | 同上 + `v-else` + `v-for="model in visibleModelUsage"` | `AccountActivity.vue:748-757` | 未出现（条件） | — |
| S-usage-68 | `model.providerId` | 文本 | 同上 | `AccountActivity.vue:758` | 未出现（条件） | — |
| S-usage-69 | `formatCompactNumber(model.totalTokens)` | 文本·值 | 同上 | `AccountActivity.vue:761` | 未出现（条件） | — |
| S-usage-70 | 总 tokens · {count} 次 | 文本 | 同上 | `AccountActivity.vue:762` | 未出现（条件） | — |
| S-usage-71 | 输入 tokens · 值 | 文本·指标 | 同上 | `AccountActivity.vue:767-769` | 未出现（条件） | — |
| S-usage-72 | 输出 tokens · 值 | 文本·指标 | 同上 | `AccountActivity.vue:771-773` | 未出现（条件） | — |
| S-usage-73 | 缓存读取 · 值 | 文本·指标 | 同上 | `AccountActivity.vue:775-777` | 未出现（条件） | — |
| S-usage-74 | 缓存写入 · 值 | 文本·指标 | 同上 | `AccountActivity.vue:779-781` | 未出现（条件） | — |

本节行数 74（图标 11）｜可见 5｜未出现（条件）69｜未命中 0｜运行时独有 0

运行时 3 个内容节点全部对上（S-usage-01/04/05）；另 2 个节点是卡片图标与容器，落在 S-usage-02/03。`AccountActivity.vue` 的 `v-else` 分支（460 行起的分支树）在本模式下整段不可达，但每一分支都有明确条件，故不计「未命中」。
---

## 3. 个性化（`components/settings/PersonalizationSettings.vue` + `SubagentModelSetting` / `SubagentCountSetting` / `ExecShellSetting` / `PromptLocaleSetting` / `CustomInstructionsSettings`）

png：[个性化-top.png](runtime/settings/个性化-top.png)、[个性化-bottom.png](runtime/settings/个性化-bottom.png)。运行时 174 节点 = 侧栏 40 + 内容 48 + 侧栏 38 + 内容 48；内容首趟 y 33–1362。分组之间是 1px 分隔线，png 实测 y=216.5 / 303.5 / 437.5 / 603 / 714。

### 3.1 页首与界面模式

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-personalization-01 | 个性化 | 页面标题 | 常显 | `PersonalizationSettings.vue:111` | 可见 | 257,33 741×25 |
| S-personalization-02 | 界面模式 | 文本标签 | 常显 | `PersonalizationSettings.vue:116` | 可见 | 257,85 741×18 |
| S-personalization-03 | 办公模式隐藏开发细节、以结果为先；编程模式展示完整开发界面。 | 文本 | 常显 | `PersonalizationSettings.vue:119` | 可见 | 257,106 741×14 |
| S-personalization-04 | 办公模式 · 更专注于文档、表格、设计等日常办公，把事做好就行。 | 控件·卡片按钮 | `v-for="option in uiModeOptions"`（2 实例，此项未选中） | `PersonalizationSettings.vue:122-135` | 可见 | 257,131 365×58 |
| S-personalization-05 | 编程模式 · 面向写代码和工程开发，diff、终端、分支等尽在掌控。 | 控件·卡片按钮 | 同上（2 实例，此项选中） | 同上 | 可见 | 632,131 365×58 |

### 3.2 展示 ACP 会话 / 默认打开目标应用

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-personalization-06 | 展示 ACP 会话 | 文本标签 | 常显 | `PersonalizationSettings.vue:143` | 可见 | 257,246 164×12 |
| S-personalization-07 | 显示 ACP 创建的会话及其工作区。 | 文本 | 常显 | `PersonalizationSettings.vue:144` | 可见 | 257,261 164×14 |
| S-personalization-08 | 展示 ACP 会话 | 开关 | 常显（未选中） | `PersonalizationSettings.vue:146` | 可见 | 970,252 28×16 |
| S-personalization-09 | 默认打开目标应用 | 文本标签 | 常显 | `PersonalizationSettings.vue:153` | 可见 | 257,332 741×18 |
| S-personalization-10 | 选择在外部编辑器中打开文件时使用的应用 | 文本 | 常显 | `PersonalizationSettings.vue:156` | 可见 | 257,353 741×14 |
| S-personalization-11 | `invalidDefaultAppMessage` | 文本 | `invalidDefaultAppMessage` | `PersonalizationSettings.vue:157-159` | 未出现（条件：默认应用有效） | — |
| S-personalization-12 | 系统默认（下拉） | 控件·下拉 | 常显 | `PersonalizationSettings.vue:161-181` | 可见 | 257,378 280×32 |
| S-personalization-13 | 选项 `option.label` | 下拉选项（`v-for="option in appOptions"`） | 下拉展开 | `PersonalizationSettings.vue:166-177` | 未出现（条件：下拉未展开） | — |
| S-personalization-14 | ↳ Iconify 图标 `Icon` | 图标 | 同上 + `getEditorIcon(option.value).type === 'iconify'` | `PersonalizationSettings.vue:172` | 未出现（条件） | — |
| S-personalization-15 | ↳ 图片图标 `img` | 图标 | 同上 + `=== 'image'` | `PersonalizationSettings.vue:173` | 未出现（条件） | — |
| S-personalization-16 | ↳ Lucide 图标 `component` | 图标 | 同上 + `=== 'lucide'` | `PersonalizationSettings.vue:174` | 未出现（条件） | — |
| S-personalization-17 | ↳ 首字母图标 `span` | 图标 | 同上 + `v-else` | `PersonalizationSettings.vue:175` | 未出现（条件） | — |

### 3.3 助手模型（`SubagentModelSetting`，第 1 个实例）

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-personalization-18 | 助手模型 | 文本标签（`titleLabel`） | 常显 | `SubagentModelSetting.vue:509-511` | 可见 | 257,466 741×18 |
| S-personalization-19 | 用于工具自动安全审计等轻量工作。 | 文本（`descriptionLabel`） | `v-if="descriptionLabel"` | `SubagentModelSetting.vue:512-514` | 可见 | 257,489 741×14 |
| S-personalization-20 | App Server 模式下不可用（模型选择触发按钮） | 控件·下拉（`:disabled`） | 常显（App Server 模式下选择被禁用） | `SubagentModelSetting.vue:518-539` | 可见 | 257,508 741×32 |
| S-personalization-21 | 助手模型是用于轻量工作（例如代为审批审核）的辅助模型。App Server 模式没有该设置的归属方，因此此控件不可用。新对话的默认模型请在模型选择器中选择。 | 文本 | 常显 | `SubagentModelSetting.vue:757-763` | 可见（`text` 截断于「因此控件不可用。新」，按 png 补全） | 257,547 741×28 |
| S-personalization-22 | 子代理模型 | 文本标签（`titleLabel`） | 常显（第 2 个实例） | `PersonalizationSettings.vue:189` · `SubagentModelSetting.vue:509-511` | 可见 | 257,632 741×18 |
| S-personalization-23 | 继承父会话（模型选择触发按钮） | 控件·下拉 | 常显 | `SubagentModelSetting.vue:518-539` | 可见 | 257,655 741×32 |
| S-personalization-24 | ↳ 触发按钮内图标（inherit / agent-decides / provider / invalid 四支） | 图标 | `currentMode === 'inherit'` 等四支互斥 | `SubagentModelSetting.vue:524-533` | 可见（继承父会话支，`ArrowUpFromLine`） | 上述按钮内 |
| S-personalization-25 | ↳ `Loader2` 图标 | 图标 | `v-if="saving"` | `SubagentModelSetting.vue:536` | 未出现（条件：未保存中） | — |
| S-personalization-26 | ↳ `ChevronDown` 图标 | 图标 | `v-else` | `SubagentModelSetting.vue:537` | 可见（png） | 上述按钮内 |

### 3.4 子代理总数限制（`SubagentCountSetting`）

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-personalization-27 | 子代理总数限制 | 文本标签 | 常显 | `SubagentCountSetting.vue:68-70` | 可见 | 257,743 86×18 |
| S-personalization-28 | 不可配置 | 文本·值 | `appServerReadEnabled`（否则为「{count} 个子代理」） | `SubagentCountSetting.vue:71-75` | 可见 | 949,743 49×18 |
| S-personalization-29 | error | 文本 | `v-if="error"` | `SubagentCountSetting.vue:77-79` | 未出现（条件：无错误） | — |
| S-personalization-30 | 子代理总数限制（滑块） | 控件·滑块（`:disabled`） | 常显 | `SubagentCountSetting.vue:82-92` | 可见 | 257,771 741×5 |
| S-personalization-31 | ↳ 滑块拇指（`role=slider`） | 控件·滑块拇指 | 常显 | 同上（Slider 内部） | 可见 | 984,767 14×14 |
| S-personalization-32 | App Server 模式没有成员并发设置，因此 Desktop 不会应用子代理上限。 | 文本 | `v-if="appServerReadEnabled"` | `SubagentCountSetting.vue:93-99` | 可见 | 257,787 741×14 |

### 3.5 命令 Shell（`ExecShellSetting`）

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-personalization-33 | 命令 Shell | 文本标签 | 常显 | `ExecShellSetting.vue:183-185` | 可见 | 257,858 741×18 |
| S-personalization-34 | 选择 Agent 内置 exec 工具使用的 Shell。 | 文本 | 常显 | `ExecShellSetting.vue:186-188` | 可见 | 257,879 741×14 |
| S-personalization-35 | 共享的命令 Shell 配置无效：{path} | 文本 | `snapshot?.configError` | `ExecShellSetting.vue:191-194` | 未出现（条件：配置有效） | — |
| S-personalization-36 | 重置为自动 | 按钮 | 同上 | `ExecShellSetting.vue:195-197` | 未出现（条件） | — |
| S-personalization-37 | 已保存的 Shell 不可用：…（下拉触发器） | 控件·下拉（`:disabled`） | `v-else`（`!configError`） | `ExecShellSetting.vue:201-212` | 可见 | 257,903 504×32 |
| S-personalization-38 | 自动选择 `automaticLabel` | 下拉选项 | 下拉展开 | `ExecShellSetting.vue:214-220` | 未出现（条件：下拉未展开） | — |
| S-personalization-39 | 已保存的 Shell 不可用：{path} | 下拉选项 | 同上 + `v-if="unavailablePath"` | `ExecShellSetting.vue:221-229` | 未出现（条件：下拉未展开） | — |
| S-personalization-40 | `shell.displayName · shell.executablePath` | 下拉选项（`v-for`） | 同上 | `ExecShellSetting.vue:231-239` | 未出现（条件：下拉未展开） | — |
| S-personalization-41 | 刷新（`loading ? 正在探测… : 刷新`） | 按钮 | `v-else` 下常显 | `ExecShellSetting.vue:245-247` | 可见 | 257,945 48×28 |
| S-personalization-42 | 有 {count} 个 Shell 候选项无法验证 | 文本 | `v-if="snapshot?.detection.failedCandidateCount"` | `ExecShellSetting.vue:248-250` | 未出现（条件：无失败候选项） | — |
| S-personalization-43 | 自动选择不可用。请选择可用 Shell，或安装后刷新。 | 文本 | `v-if="!snapshot.detection.autoTarget"` | `ExecShellSetting.vue:253-255` | 未出现（条件：自动选择可用） | — |
| S-personalization-44 | 已保存的 Shell 不可用：{path}（行内提示） | 文本 | `v-if="unavailablePath"` | `ExecShellSetting.vue:256-258` | 未出现（条件：无不可用路径） | — |
| S-personalization-45 | 命令提示符无法从网络共享路径启动此任务。请选择其他 Shell 或本地工作目录。 | 文本 | `v-if="cmdNetworkShare"` | `ExecShellSetting.vue:259-261` | 未出现（条件：非网络共享路径） | — |
| S-personalization-46 | 非 UTF-8 命令输出可能显示为字节转义（<0xNN>）。文件工具仍使用 Unicode 路径。 | 文本 | `v-if="selectedShell?.outputEncoding === 'unknown'"` | `ExecShellSetting.vue:262-264` | 未出现（条件：编码已知） | — |
| S-personalization-47 | 当前任务继续使用 {shell}。请新建任务以应用此选择。 | 文本 | `v-if="activeMismatch && snapshot?.activeRuntime"` | `ExecShellSetting.vue:265-267` | 未出现（条件：运行中 Shell 一致） | — |
| S-personalization-48 | TUI 和 Desktop 共享此设置。新建或重新加载的任务使用它；当前任务保持原 Shell。 | 文本 | 常显 | `ExecShellSetting.vue:268-270` | 可见 | 257,984 741×14 |
| S-personalization-49 | 已有任务以其他 Shell 重新加载时，可能降低提示词缓存复用率并消耗额外 token。 | 文本 | 常显 | `ExecShellSetting.vue:271-273` | 可见 | 257,1008 741×14 |
| S-personalization-50 | error | 文本 | `v-if="error"` | `ExecShellSetting.vue:276-278` | 未出现（条件：无错误） | — |

### 3.6 提示词语言（`PromptLocaleSetting`）

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-personalization-51 | 提示词语言 | 文本标签 | `appServerReadEnabled` | `PromptLocaleSetting.vue:30-32` | 可见 | 257,1079 741×18 |
| S-personalization-52 | 提示词语言已禁用：App Server 会话还没有语言载体，Agent 无法接收此设置。Desktop 不会写入本地提示词语言文件。 | 文本 | 同上 | `PromptLocaleSetting.vue:33-35` | 可见（`text` 截断于「本地提示」，按模板补全） | 257,1100 741×14 |
| S-personalization-53 | Owner 缺口： | 文本 | 同上 | `PromptLocaleSetting.vue:36-38` | 可见 | 257,1118 741×14 |
| S-personalization-54 | arcships/dim#1018 | 文本·链接 | 同上 | `PromptLocaleSetting.vue:38-43` | 可见 | 327,1118 92×13 |
| S-personalization-55 | · | 文本·分隔点 | 同上 | `PromptLocaleSetting.vue:44` | 未出现（JSON 未捕获该 span；png 未单独判读） | — |
| S-personalization-56 | arcships/dim#548 | 文本·链接 | 同上 | `PromptLocaleSetting.vue:45-50` | 可见 | 428,1118 90×13 |
| S-personalization-57 | App Server 模式下不可用（下拉） | 控件·下拉（`:disabled`） | 同上 | `PromptLocaleSetting.vue:53-56` | 可见 | 257,1142 504×32 |
| S-personalization-58 | `locale.label` | 下拉选项（`v-for`） | 下拉展开 | `PromptLocaleSetting.vue:57-58` | 未出现（条件：下拉未展开） | — |

### 3.7 自定义指令（`CustomInstructionsSettings`）

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-personalization-59 | 自定义指令 | 分组标题 | 常显 | `CustomInstructionsSettings.vue:85` | 可见 | 257,1231 741×18 |
| S-personalization-60 | App Server 在分支加载时从下面的文件读取你的全局自定义指令，并定期重新检查。Desktop 不保存其副本。 | 文本 | `appServerReadEnabled` 分支（否则为另一段说明） | `CustomInstructionsSettings.vue:86-90` | 可见 | 257,1252 672×18 |
| S-personalization-61 | `agentsFilePath`（~/AGENTS.md） | 文本·值（code） | `appServerReadEnabled` | `CustomInstructionsSettings.vue:93-105` | 可见 | 272,1302 587×18 |
| S-personalization-62 | 打开或创建文件 | 按钮 | 同上 | `CustomInstructionsSettings.vue:106-112` | 可见 | 869,1295 114×32 |
| S-personalization-63 | PromptSources 会读取此 AGENTS.md 文件。请在编辑器中修改；更改将在新的运行中生效。 | 文本 | 同上 | `CustomInstructionsSettings.vue:114-116` | 可见 | 257,1348 672×14 |
| S-personalization-64 | `errorText` | 文本 | 同上 + `v-if="errorText"` | `CustomInstructionsSettings.vue:117-119` | 未出现（条件：无错误） | — |
| S-personalization-65 | 自定义指令（`Label`） | 文本标签 | `v-else`（`!appServerReadEnabled`） | `CustomInstructionsSettings.vue:122-123` | 未出现（条件：App Server 只读模式） | — |
| S-personalization-66 | 输入给 DimAgent 的指令...（`Textarea`） | 控件·输入 | 同上 | `CustomInstructionsSettings.vue:124` | 未出现（条件） | — |
| S-personalization-67 | `errorText` | 文本 | 同上 + `v-if="errorText"` | `CustomInstructionsSettings.vue:131-133` | 未出现（条件） | — |
| S-personalization-68 | 保存 | 按钮 | 同上 | `CustomInstructionsSettings.vue:134-136` | 未出现（条件） | — |

### 3.8 模型选择对话框（`SubagentModelSetting` 内部，两处实例共用；对话框未打开）

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-personalization-69 | 模型选择对话框 | 对话框（`DialogContent`） | 点击 S-personalization-20 / -23 | `SubagentModelSetting.vue:541` | 未出现（条件：对话框未打开） | — |
| S-personalization-70 | `error` | 文本 | `v-if="error"` | `SubagentModelSetting.vue:546-552` | 未出现（条件） | — |
| S-personalization-71 | `titleLabel` | 对话框标题 | `step === 1` | `SubagentModelSetting.vue:556-558` | 未出现（条件） | — |
| S-personalization-72 | 继承父会话 · `inheritDescription` | 选项按钮 | 同上 | `SubagentModelSetting.vue:563-580` | 未出现（条件） | — |
| S-personalization-73 | ↳ `ArrowUpFromLine` 图标 | 图标 | 同上 | `SubagentModelSetting.vue:572` | 未出现（条件） | — |
| S-personalization-74 | ↳ `Check` 图标 | 图标 | 同上 + `currentMode === 'inherit'` | `SubagentModelSetting.vue:577` | 未出现（条件） | — |
| S-personalization-75 | `agentDecidesLabel` · `agentDecidesDescription` | 选项按钮 | `allowAgentDecides` | `SubagentModelSetting.vue:584-602` | 未出现（条件） | — |
| S-personalization-76 | ↳ `Sparkles` 图标 | 图标 | 同上 | `SubagentModelSetting.vue:594` | 未出现（条件） | — |
| S-personalization-77 | ↳ `Check` 图标 | 图标 | 同上 + `currentMode === 'agent-decides'` | `SubagentModelSetting.vue:599` | 未出现（条件） | — |
| S-personalization-78 | `specificLabel` · `specificDescription` | 选项按钮 | 同上 | `SubagentModelSetting.vue:606-624` | 未出现（条件） | — |
| S-personalization-79 | ↳ `Target` 图标 | 图标 | 同上 | `SubagentModelSetting.vue:615` | 未出现（条件） | — |
| S-personalization-80 | ↳ `Check` / `ChevronRight` 图标 | 图标 | `currentMode === 'fixed'` / 常显 | `SubagentModelSetting.vue:620-621` | 未出现（条件） | — |
| S-personalization-81 | `currentFixedName` + 供应商图标 | 文本·值 | 同上 + `v-if="currentFixedName"` | `SubagentModelSetting.vue:624-634` | 未出现（条件） | — |
| S-personalization-82 | 返回（步骤 2 头部） | 图标按钮 | `step === 2` | `SubagentModelSetting.vue:644-650` | 未出现（条件） | — |
| S-personalization-83 | ↳ `ChevronLeft` 图标 | 图标 | 同上 | `SubagentModelSetting.vue:650` | 未出现（条件） | — |
| S-personalization-84 | `specificLabel` | 对话框标题 | 同上 | `SubagentModelSetting.vue:652` | 未出现（条件） | — |
| S-personalization-85 | `filterPlaceholder`（搜索框） | 控件·输入 | 同上 | `SubagentModelSetting.vue:655-657` | 未出现（条件） | — |
| S-personalization-86 | ↳ `Search` 图标 | 图标 | 同上 | `SubagentModelSetting.vue:656` | 未出现（条件） | — |
| S-personalization-87 | `provider.name`（供应商列表） | 控件·按钮（`v-for`） | 同上 | `SubagentModelSetting.vue:668-678` | 未出现（条件） | — |
| S-personalization-88 | ↳ 供应商图标 `ProviderIcon` | 图标 | 同上 | `SubagentModelSetting.vue:677` | 未出现（条件） | — |
| S-personalization-89 | `noMatchesLabel` | 文本 | `visibleProviders.length === 0` | `SubagentModelSetting.vue:680-682` | 未出现（条件） | — |
| S-personalization-90 | `model.name` · `model.semanticText` | 选项行（`v-for`，`role="button"`） | 选中供应商后 | `SubagentModelSetting.vue:687-708` | 未出现（条件） | — |
| S-personalization-91 | ↳ `Check` / `ChevronRight` 图标 | 图标 | 同上 | `SubagentModelSetting.vue:700-701` | 未出现（条件） | — |
| S-personalization-92 | `noMatchesLabel` | 文本 | `modelsForActiveProvider.length === 0` | `SubagentModelSetting.vue:710-712` | 未出现（条件） | — |
| S-personalization-93 | 返回（步骤 3 头部） | 图标按钮 | `v-else`（step 3） | `SubagentModelSetting.vue:720-726` | 未出现（条件） | — |
| S-personalization-94 | 思考强度 | 对话框标题 | 同上 | `SubagentModelSetting.vue:728` | 未出现（条件） | — |
| S-personalization-95 | `pendingModel.name` | 文本 | 同上 + `v-if="pendingModel"` | `SubagentModelSetting.vue:732-734` | 未出现（条件） | — |
| S-personalization-96 | `reasoningSelectionLabel(selection)` | 选项按钮（`v-for`） | 同上 | `SubagentModelSetting.vue:737-748` | 未出现（条件） | — |
| S-personalization-97 | ↳ `EffortStrengthIcon` / `Check` 图标 | 图标 | 同上 | `SubagentModelSetting.vue:746-748` | 未出现（条件） | — |
| S-personalization-98 | `unavailableDescription` | 文本 | `v-if="policyUnavailable"` | `SubagentModelSetting.vue:757-763` | 未出现（条件） | — |
| S-personalization-99 | `error` | 文本 | `v-if="error"` | `SubagentModelSetting.vue:765-771` | 未出现（条件） | — |

本节行数 99（图标 20）｜可见 39｜未出现（条件）60｜未命中 0｜运行时独有 0

运行时对齐：48 个内容节点 = 39 个可见主体行 + 9 个折叠内文（办公模式标签/描述、编程模式标签/描述、系统默认、App Server 模式下不可用 ×2、继承父会话、已保存的 Shell 不可用：） + 0 独有。
---

## 4. 外观（`components/settings/AppearanceSettings.vue` + `SkinSettings.vue`）

png：[外观-top.png](runtime/settings/外观-top.png)、[外观-bottom.png](runtime/settings/外观-bottom.png)。运行时 184 节点 = 侧栏 40 + 内容 53 + 侧栏 38 + 内容 53；内容首趟 y 33–1274。

### 4.1 主题 / 程序语言 / 字体

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-appearance-01 | 外观 | 页面标题 | 常显 | `AppearanceSettings.vue:265` | 可见 | 257,33 741×25 |
| S-appearance-02 | 主题 | 文本标签 | 常显 | `AppearanceSettings.vue:270` | 可见 | 257,85 741×18 |
| S-appearance-03 | 浅色 · 主题卡 | 控件·卡片按钮（`v-for="option in themeOptions"`，`disabled`） | 常显 | `AppearanceSettings.vue:272-284` | 可见 | 257,113 240×69 |
| S-appearance-04 | ↳ 主题预览图标 | 图标 | 同上 | `AppearanceSettings.vue:281` | 可见（icon:true 于 S-appearance-03） | 同上 |
| S-appearance-05 | 深色 · 主题卡 | 控件·卡片按钮 | 同上（2/3 实例） | 同上 | 可见 | 507,113 240×69 |
| S-appearance-06 | 跟随系统 · 主题卡 | 控件·卡片按钮 | 同上（3/3 实例，选中标记 `v-if`） | 同上 | 可见 | 758,113 240×69 |
| S-appearance-07 | ↳ 选中标记 `div` | 状态·选中角标 | `classicAppearance ? option.value === 'light' : theme === option.value` | `AppearanceSettings.vue:283` | 可见（跟随系统支） | 上述卡内 |
| S-appearance-08 | 经典外观使用浅色；切回默认或 macOS 时恢复原明暗偏好。 | 文本 | `classicAppearance` | `AppearanceSettings.vue:289` | 未出现（条件：非经典外观） | — |
| S-appearance-09 | 程序语言 | 文本标签 | 常显 | `AppearanceSettings.vue:297` | 可见 | 257,239 741×18 |
| S-appearance-10 | 简体中文（语言下拉） | 控件·下拉 | 常显 | `AppearanceSettings.vue:298-309` | 可见 | 257,267 168×32 |
| S-appearance-11 | `option.labelKey` | 下拉选项（`v-for="option in languageOptions"`） | 下拉展开 | `AppearanceSettings.vue:303-305` | 未出现（条件：下拉未展开） | — |
| S-appearance-12 | 字体 | 文本标签 | 常显 | `AppearanceSettings.vue:315` | 可见 | 257,355 741×18 |
| S-appearance-13 | UI 字体 | 文本标签（`v-for="picker in fontPickers"`，1/2） | 常显 | `AppearanceSettings.vue:317-323` | 可见 | 257,387 153×18 |
| S-appearance-14 | 用于应用界面和 Markdown 正文 | 文本 | 同上 | `AppearanceSettings.vue:324` | 可见 | 257,406 153×14 |
| S-appearance-15 | 默认（UI 字体下拉触发器） | 控件·下拉（`title="picker.family"`） | 同上 | `AppearanceSettings.vue:326-342` | 可见 | 774,387 224×32 |
| S-appearance-16 | ↳ `ChevronDown` 图标 | 图标 | 同上 | `AppearanceSettings.vue:341` | 可见（png；JSON 标 icon:true 于 S-appearance-15） | 同上 |
| S-appearance-17 | 代码字体 | 文本标签（`v-for`，2/2） | 常显 | `AppearanceSettings.vue:317-323` | 可见 | 257,434 137×18 |
| S-appearance-18 | 用于终端、代码块和差异内容 | 文本 | 同上 | `AppearanceSettings.vue:324` | 可见 | 257,453 137×14 |
| S-appearance-19 | 默认（代码字体下拉触发器） | 控件·下拉 | 同上 | `AppearanceSettings.vue:326-342` | 可见 | 774,435 224×32 |
| S-appearance-20 | 搜索字体（`ComboboxInput`） | 控件·输入 | 下拉展开 | `AppearanceSettings.vue:344-348` | 未出现（条件：下拉未展开） | — |
| S-appearance-21 | 没有匹配的字体 | 文本（`ComboboxEmpty`） | 同上 | `AppearanceSettings.vue:354` | 未出现（条件） | — |
| S-appearance-22 | 默认（字体选项，含 `Check`） | 下拉选项 | 同上 + `!fontQuery.trim()` | `AppearanceSettings.vue:355-358` | 未出现（条件） | — |
| S-appearance-23 | `family`（系统字体选项，`v-for`） | 下拉选项 | 同上 | `AppearanceSettings.vue:362-369` | 未出现（条件） | — |
| S-appearance-24 | 正在加载...（字体） | 文本 | `!appearanceStore.fontsLoaded` | `AppearanceSettings.vue:377-379` | 未出现（条件：字体已加载） | — |
| S-appearance-25 | 无法读取系统字体列表。 | 文本 | `appearanceStore.fontEnumerationFailed` | `AppearanceSettings.vue:380-381` | 未出现（条件：枚举成功） | — |
| S-appearance-26 | 重试（`fontRefreshing ? 正在加载... : 重试`） | 按钮 | 同上 | `AppearanceSettings.vue:382-384` | 未出现（条件） | — |
| S-appearance-27 | 字体“{font}”已不可用，已恢复为默认字体。 | 文本（`v-for="font in missingFonts"`） | `missingFonts` 非空 | `AppearanceSettings.vue:386-388` | 未出现（条件：无缺失字体） | — |
| S-appearance-28 | 保存字体设置失败，已恢复之前的选择。 | 文本 | `fontSaveFailed.ui \|\| fontSaveFailed.code` | `AppearanceSettings.vue:389-391` | 未出现（条件：保存成功） | — |

### 4.2 字号 / 代码主题

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-appearance-29 | UI 字号 | 文本标签 | 常显 | `AppearanceSettings.vue:401` | 可见 | 257,528 40×18 |
| S-appearance-30 | 小 | 文本·刻度 | 常显 | `AppearanceSettings.vue:404` | 可见 | 809,529 11×14 |
| S-appearance-31 | 特小 | 控件·分段按钮（`v-for="(size, index) in UI_FONT_SIZE_STEPS"`，1/5） | 常显 | `AppearanceSettings.vue:406-415` | 可见 | 826,524 28×25 |
| S-appearance-32 | 小 | 分段按钮（2/5） | 同上 | 同上 | 可见 | 858,524 28×25 |
| S-appearance-33 | 中 | 分段按钮（3/5） | 同上 | 同上 | 可见 | 889,524 28×25 |
| S-appearance-34 | 大 | 分段按钮（4/5） | 同上 | 同上 | 可见 | 921,524 28×25 |
| S-appearance-35 | 特大 | 分段按钮（5/5） | 同上 | 同上 | 可见 | 952,524 28×25 |
| S-appearance-36 | 大 | 文本·刻度 | 常显 | `AppearanceSettings.vue:417` | 可见 | 987,529 11×14 |
| S-appearance-37 | 代码字号 | 文本标签 | 常显 | `AppearanceSettings.vue:429` | 可见 | 257,606 49×18 |
| S-appearance-38 | `codeFontSize[0]px`（12px） | 文本·值 | 常显 | `AppearanceSettings.vue:431` | 可见 | 968,606 30×18 |
| S-appearance-39 | 代码字号（滑块） | 控件·滑块 | 常显 | `AppearanceSettings.vue:433-439` | 可见（拇指 14×14） | 257,633 14×14（拇指） |
| S-appearance-40 | 12px | 文本·刻度 | 常显 | `AppearanceSettings.vue:441` | 可见 | 257,656 23×14 |
| S-appearance-41 | 16px | 文本·刻度 | 常显 | `AppearanceSettings.vue:442` | 可见 | 614,656 23×14 |
| S-appearance-42 | 20px | 文本·刻度 | 常显 | `AppearanceSettings.vue:443` | 可见 | 973,656 25×14 |
| S-appearance-43 | 代码主题 | 文本标签 | 常显 | `AppearanceSettings.vue:453` | 可见 | 257,727 741×18 |
| S-appearance-44 | GitHub（代码主题下拉） | 控件·下拉 | 常显 | `AppearanceSettings.vue:455-457` | 可见 | 257,759 741×32 |
| S-appearance-45 | 浅色 + 深色（`SelectLabel`） | 下拉分组标题 | 下拉展开 | `AppearanceSettings.vue:461` | 未出现（条件：下拉未展开） | — |
| S-appearance-46 | `f.name`（成对主题，`v-for`） | 下拉选项 | 同上 | `AppearanceSettings.vue:462-464` | 未出现（条件） | — |
| S-appearance-47 | 深色（`SelectLabel`） | 下拉分组标题 | 同上 | `AppearanceSettings.vue:468` | 未出现（条件） | — |
| S-appearance-48 | `f.name`（仅深色主题，`v-for`） | 下拉选项 | 同上 | `AppearanceSettings.vue:469-471` | 未出现（条件） | — |
| S-appearance-49 | 代码主题预览区（`v-for="(panel, index) in previewPanels"`） | 容器·预览块 | 常显 | `AppearanceSettings.vue:481-490` | 可见（png；JSON 无文本节点） | y 800–950 区内 |

### 4.3 我的皮肤（`SkinSettings.vue`，`v-if="hasSkinApi()"`）

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-appearance-50 | 我的皮肤 | 文本标签 | `hasSkinApi()` | `SkinSettings.vue:503` | 可见 | 257,977 741×18 |
| S-appearance-51 | 加载骨架（`v-for="i in 3"`） | 容器·骨架 | 同上 + `isLoading` | `SkinSettings.vue:506-507` | 未出现（条件：已加载） | — |
| S-appearance-52 | Windows 98（皮肤卡） | 控件·卡片按钮（`v-for="skin in skinStore.skins"`，1/6） | 同上 + `v-else` | `SkinSettings.vue:514-551` | 可见 | 257,1005 240×98 |
| S-appearance-53 | Windows XP（皮肤卡，2/6） | 控件·卡片按钮 | 同上 | 同上 | 可见 | 507,1005 240×98 |
| S-appearance-54 | Windows Vista（皮肤卡，3/6） | 控件·卡片按钮 | 同上 | 同上 | 可见 | 758,1005 240×98 |
| S-appearance-55 | macOS Big Sur（皮肤卡，4/6） | 控件·卡片按钮 | 同上 | 同上 | 可见 | 257,1113 240×98 |
| S-appearance-56 | OS X Tiger（皮肤卡，5/6） | 控件·卡片按钮 | 同上 | 同上 | 可见 | 507,1113 240×98 |
| S-appearance-57 | （皮肤缩略图首字母）`getInitial(skin.name)` | 文本 | 同上 + `!skin.thumbnailUrl` | `SkinSettings.vue:529-533` | 未出现（条件：有缩略图） | — |
| S-appearance-58 | ↳ 皮肤异常标记 `AlertTriangle` | 图标 | 同上 + `skin.error` | `SkinSettings.vue:537-541` | 未出现（条件：无异常） | — |
| S-appearance-59 | 皮肤名 `skin.name` | 文本 | 同上 | `SkinSettings.vue:549-551` | 可见（6 项，如 Windows 98） | 258,1086 238×21 |
| S-appearance-60 | 删除皮肤 | 图标按钮 | 同上 + `activeSkinId !== skin.id` | `SkinSettings.vue:554-563` | 未出现（条件：卡片未悬停/不在名单） | — |
| S-appearance-61 | ↳ `Trash2` 图标 | 图标 | 同上 | `SkinSettings.vue:563` | 未出现（条件） | — |
| S-appearance-62 | 删除皮肤？ | 对话框标题 | 同上 + 点击删除 | `SkinSettings.vue:566-568` | 未出现（条件：对话框未打开） | — |
| S-appearance-63 | “{name}”及其全部资源将被永久删除。 | 对话框说明 | 同上 | `SkinSettings.vue:569-571` | 未出现（条件） | — |
| S-appearance-64 | 取消 | 对话框按钮 | 同上 | `SkinSettings.vue:574` | 未出现（条件） | — |
| S-appearance-65 | 删除（`deletingSkinId === skin.id ? 正在删除… : 删除`） | 对话框按钮 | 同上 | `SkinSettings.vue:575-581` | 未出现（条件） | — |
| S-appearance-66 | 内置外观卡（`v-for="id in chromeThemeOptions"`，3 实例） | 控件·卡片按钮 | 同上 | `SkinSettings.vue:588-614` | 可见（3 张，`chromeNames[id]`） | 758,1113 240×98 起 |
| S-appearance-67 | `chromeNames[id]` | 文本 | 同上 | `SkinSettings.vue:612-614` | 可见（默认 之一） | 867,1180 21×14 |
| S-appearance-68 | 默认（新建皮肤占位卡） | 控件·卡片按钮 | 同上 | `SkinSettings.vue:618-627` | 可见 | 758,1113 240×98 |
| S-appearance-69 | ↳ `X` 图标 | 图标 | 同上 | `SkinSettings.vue:624` | 可见（png） | 上述卡内 |
| S-appearance-70 | 暂无自定义皮肤。可用图片新建或导入已有皮肤。 | 文本 | 同上 + `skins.length === 0` | `SkinSettings.vue:635-637` | 可见 | 257,1222 741×14 |
| S-appearance-71 | 用图片新建 | 按钮 | 同上 | `SkinSettings.vue:642-645` | 可见 | 257,1246 105×28 |
| S-appearance-72 | ↳ `Plus` 图标 | 图标 | 同上 | `SkinSettings.vue:643` | 可见（png） | 上述按钮内 |
| S-appearance-73 | 打开目录 | 按钮 | 同上 | `SkinSettings.vue:646-649` | 可见 | 369,1246 93×28 |
| S-appearance-74 | ↳ `FolderOpen` 图标 | 图标 | 同上 | `SkinSettings.vue:647` | 可见（png） | 上述按钮内 |
| S-appearance-75 | `deleteError` | 文本（role=alert） | `v-if="deleteError"` | `SkinSettings.vue:652-654` | 未出现（条件：无错误） | — |
| S-appearance-76 | 当前皮肤设置 | 文本标签 | `v-if="activeSkin"` | `SkinSettings.vue:662` | 未出现（条件：未启用皮肤） | — |
| S-appearance-77 | 窗口背景 | 文本标签 | 同上 | `SkinSettings.vue:667` | 未出现（条件） | — |
| S-appearance-78 | `activeImagePath ?? 未设置图片` | 文本·值（code） | 同上 | `SkinSettings.vue:672-674` | 未出现（条件） | — |
| S-appearance-79 | 更换 | 按钮 | 同上 | `SkinSettings.vue:675-678` | 未出现（条件） | — |
| S-appearance-80 | 移除背景图 | 图标按钮 | 同上 + `activeImagePath` | `SkinSettings.vue:679-685` | 未出现（条件） | — |
| S-appearance-81 | 透明度 · 值（`(activeOpacity[0]*100)`%） | 文本·标签 + 值 | 同上 | `SkinSettings.vue:692-695` | 未出现（条件） | — |
| S-appearance-82 | 透明度（滑块） | 控件·滑块 | 同上 | `SkinSettings.vue:697` | 未出现（条件） | — |
| S-appearance-83 | 大小（`Select`） | 控件·下拉 | 同上 | `SkinSettings.vue:709-717` | 未出现（条件） | — |
| S-appearance-84 | 位置（`Select`） | 控件·下拉 | 同上 | `SkinSettings.vue:724-732` | 未出现（条件） | — |
| S-appearance-85 | 附着方式（`Select`） | 控件·下拉 | 同上 | `SkinSettings.vue:739-748` | 未出现（条件） | — |
| S-appearance-86 | ⚠ 可能影响滚动性能 | 文本 | 同上 + `activeAttachment === 'fixed'` | `SkinSettings.vue:751-753` | 未出现（条件） | — |
| S-appearance-87 | 文字颜色 · 根据背景调整文字颜色，保证可读性。 | 文本标签 + 文本 | 同上 | `SkinSettings.vue:762-763` | 未出现（条件） | — |
| S-appearance-88 | 文字颜色（`Select`） | 控件·下拉 | 同上 | `SkinSettings.vue:765-776` | 未出现（条件） | — |
| S-appearance-89 | 高级 | 折叠按钮 | 同上 | `SkinSettings.vue:783-789` | 未出现（条件：`showAdvanced` 未展开） | — |
| S-appearance-90 | 外部打开 | 按钮 | 同上 + `showAdvanced` | `SkinSettings.vue:796-799` | 未出现（条件） | — |
| S-appearance-91 | `errorMessage` 提示条 | 容器·提示条 | 同上 + `hasError` | `SkinSettings.vue:805-812` | 未出现（条件：无错误） | — |
| S-appearance-92 | 打开 skin.json | 按钮 | 同上 | `SkinSettings.vue:813-816` | 未出现（条件） | — |
| S-appearance-93 | 重新加载 | 按钮 | 同上 | `SkinSettings.vue:817-820` | 未出现（条件） | — |
| S-appearance-94 | 皮肤已重新加载 | 文本 | 同上 + `reloadFeedback` | `SkinSettings.vue:825-830` | 未出现（条件） | — |

本节行数 94（图标 9）｜可见 39｜未出现（条件）55｜未命中 0｜运行时独有 0

运行时对齐：53 个内容节点 = 39 个可见主体行 + 14 个折叠/合并项（3 个主题卡文字 浅色 / 深色 / 跟随系统、2 个字体卡值 默认、语言与代码主题的下拉当前值、5 张皮肤卡内的名称 div 已单列于 S-appearance-59 等） + 0 独有。皮肤区实际渲染 6 张皮肤卡 + 1 张「默认」占位卡，模板里分别是 `skinStore.skins` 与 `chromeThemeOptions` 两个 `v-for`。
---

## 5. 通用（`components/settings/GeneralSettings.vue` + `NetworkSettings` / `BrowserDataSettings` / `DataCleanupDialog` / `ProxyConfigDialog`）

png：[通用-top.png](runtime/settings/通用-top.png)、[通用-bottom.png](runtime/settings/通用-bottom.png)。运行时 138 节点 = 侧栏 40 + 内容 30 + 侧栏 38 + 内容 30；内容首趟 y 33–813。

### 5.1 开关行

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-general-01 | 通用 | 页面标题 | 常显 | `GeneralSettings.vue:132` | 可见 | 257,33 741×25 |
| S-general-02 | 保持电脑唤醒 | 文本标签 | 常显 | `GeneralSettings.vue:140` | 可见 | 272,100 210×18 |
| S-general-03 | 运行期间阻止系统休眠。屏幕也会保持常亮。 | 文本 | 常显 | `GeneralSettings.vue:141-143` | 可见 | 272,119 210×14 |
| S-general-04 | 保持电脑唤醒 | 开关 | 常显 | `GeneralSettings.vue:145` | 可见 | 955,109 28×16 |
| S-general-05 | 性能日志 | 文本标签 | 常显 | `GeneralSettings.vue:150` | 可见 | 272,184 436×18 |
| S-general-06 | 将各进程 CPU/RSS/IO 采样写入 logs/performance/（每秒一行）。tail 该文件即可实时查看。 | 文本 | 常显 | `GeneralSettings.vue:151-153` | 可见 | 272,204 436×14 |
| S-general-07 | 本次启动由 DIM_PERF 决定采集，该开关不生效。 | 文本 | `v-if="performanceLogPinned"` | `GeneralSettings.vue:154-156` | 未出现（条件：未设 DIM_PERF） | — |
| S-general-08 | 性能日志 | 开关（`:disabled`） | 常显 | `GeneralSettings.vue:158` | 可见 | 955,193 28×16 |
| S-general-09 | 登录时自动启动 | 文本标签 | 常显 | `GeneralSettings.vue:168` | 可见 | 272,269 86×18 |
| S-general-10 | 登录时自动启动 | 开关 | 常显 | `GeneralSettings.vue:170` | 可见 | 955,269 28×16 |

### 5.2 会话数据根目录

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-general-11 | 会话数据根目录 | 文本标签 | 常显 | `GeneralSettings.vue:179` | 可见 | 272,337 711×18 |
| S-general-12 | 会话记录、日志与缓存都位于该目录下。留空则使用默认根目录。 保存后需要重启 DimAgent 才能生效。 | 文本 | 常显 | `GeneralSettings.vue:180-182` | 可见 | 272,356 711×14 |
| S-general-13 | 例如 /Users/you/.dimcode（输入框） | 控件·输入（`:disabled`） | 常显 | `GeneralSettings.vue:185` | 可见（占位符不在 JSON 的 `text`） | 272,377 608×32 |
| S-general-14 | 重启 App | 按钮 | 常显 | `GeneralSettings.vue:192-201` | 可见 | 887,377 96×32 |
| S-general-15 | ↳ `Loader2` 图标 | 图标 | `v-if="dimcodeRestartBusy"` | `GeneralSettings.vue:198` | 未出现（条件：未重启中） | — |
| S-general-16 | ↳ `RefreshCw` 图标 | 图标 | `v-else` | `GeneralSettings.vue:199` | 可见（png） | 上述按钮内 |
| S-general-17 | `dimcodeRestartError` | 文本 | `v-if="dimcodeRestartError"` | `GeneralSettings.vue:203-205` | 未出现（条件：无错误） | — |

### 5.3 代理（`NetworkSettings.vue`，`embedded` 模式）

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-general-18 | 网络 | 页面标题 | `!embedded`（独立页） | `NetworkSettings.vue:158-159` | 未出现（条件：以 `embedded` 形式嵌在通用页） | — |
| S-general-19 | 配置 DimAgent 访问网络服务的方式。 | 文本 | 同上 | `NetworkSettings.vue:160-162` | 未出现（条件） | — |
| S-general-20 | 代理 | 文本标签 | 常显 | `NetworkSettings.vue:167` | 可见 | 272,460 711×18 |
| S-general-21 | 应用于模型请求、登录 API，以及后续新启动的工具进程。 | 文本 | 常显 | `NetworkSettings.vue:168-170` | 可见 | 272,481 711×14 |
| S-general-22 | 代理模式切换组 | 控件·分段组（`ToggleGroup`，`:disabled`） | 常显 | `NetworkSettings.vue:173-179` | 可见 | 272,509 711×41 |
| S-general-23 | 系统代理 | 分段项 | `v-for="option in modeOptions"`（1/3，选中） | `NetworkSettings.vue:180-189` | 可见 | 272,509（文字 308,520 49×18） |
| S-general-24 | 无代理 | 分段项（2/3） | 同上 | 同上 | 可见 | 511,509 232×41（文字 547,520 37×18） |
| S-general-25 | 自定义代理 | 分段项（3/3） | 同上 | 同上 | 可见 | 750,509 232×41（文字 786,520 61×18） |
| S-general-26 | ↳ 分段项图标（`<component :is>`） | 图标 | 同上 | `NetworkSettings.vue:187` | 可见（icon:true 于 2、3 项） | 同上 |
| S-general-27 | 自定义代理 | 文本标签 | `mode === 'custom'` | `NetworkSettings.vue:199-201` | 未出现（条件：系统代理模式） | — |
| S-general-28 | `customDisplay` | 文本·值 | 同上（`title="customUrl"`） | `NetworkSettings.vue:202-209` | 未出现（条件） | — |
| S-general-29 | 编辑 | 按钮（含 `Pencil` 图标） | 同上 | `NetworkSettings.vue:211-220` | 未出现（条件） | — |
| S-general-30 | `pageError` | 文本 | `v-if="pageError"` | `NetworkSettings.vue:223-225` | 未出现（条件：无错误） | — |
| S-general-31 | 自定义代理（对话框） | 对话框 | 点击 S-general-29 | `ProxyConfigDialog.vue:96-102` | 未出现（条件：对话框未打开） | — |
| S-general-32 | 输入 http 或 https 代理 URL。 | 对话框说明 | 同上 | `ProxyConfigDialog.vue:100-102` | 未出现（条件） | — |
| S-general-33 | 代理 URL | 文本标签 | 同上 | `ProxyConfigDialog.vue:106` | 未出现（条件） | — |
| S-general-34 | http://127.0.0.1:7890（输入框） | 控件·输入（`:disabled`） | 同上 | `ProxyConfigDialog.vue:107` | 未出现（条件） | — |
| S-general-35 | `visibleError` | 文本 | 同上 + `v-if="visibleError"` | `ProxyConfigDialog.vue:115-117` | 未出现（条件） | — |
| S-general-36 | 取消 | 按钮 | 同上 | `ProxyConfigDialog.vue:121-123` | 未出现（条件） | — |
| S-general-37 | 保存（`saving ? 保存中... : 保存`） | 按钮 | 同上 | `ProxyConfigDialog.vue:124-126` | 未出现（条件） | — |

### 5.4 浏览器数据（`BrowserDataSettings.vue`）

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-general-38 | 浏览器数据 | 分组标题 | 常显 | `BrowserDataSettings.vue:224` | 可见 | 272,597 711×18 |
| S-general-39 | 网站登录状态和存储使用独立的浏览器配置。 | 文本 | 常显 | `BrowserDataSettings.vue:225` | 可见 | 272,618 711×14 |
| S-general-40 | 无法刷新浏览器数据状态，这不表示导入或清理操作失败。 | 文本（role=alert） | `statusFailed` | `BrowserDataSettings.vue:228-229` | 未出现（条件：状态正常） | — |
| S-general-41 | 重试 | 按钮 | 同上 | `BrowserDataSettings.vue:230` | 未出现（条件） | — |
| S-general-42 | Chrome 数据 | 文本·行标签 | `status?.importSupported` | `BrowserDataSettings.vue:233-238` | 可见 | 272,659 74×18 |
| S-general-43 | 导入… | 按钮（`:disabled`） | 同上 | `BrowserDataSettings.vue:239-247` | 可见 | 924,654 59×28 |
| S-general-44 | 浏览器存储 | 文本·行标签 | 常显 | `BrowserDataSettings.vue:250` | 可见 | 272,709 61×18 |
| S-general-45 | 清理数据… | 按钮（`:disabled`） | 常显 | `BrowserDataSettings.vue:251-259` | 可见 | 899,704 83×28 |
| S-general-46 | 导入 Chrome Cookie（对话框） | 对话框 | 点击 S-general-43 | `BrowserDataSettings.vue:264-274` | 未出现（条件：对话框未打开） | — |
| S-general-47 | 此操作会替换内置浏览器中的全部非分区 Cookie… | 对话框说明 | 同上 + `importable !== 0` | `BrowserDataSettings.vue:274` | 未出现（条件） | — |
| S-general-48 | 正在扫描 Chrome 配置… | 文本 | 同上 + `scanning` | `BrowserDataSettings.vue:278-281` | 未出现（条件） | — |
| S-general-49 | Chrome 配置 | 分组标题 | 同上 + `v-else` | `BrowserDataSettings.vue:283` | 未出现（条件） | — |
| S-general-50 | `profile.displayName` · `capabilityLabel(profile.capability)` | 选项行（`role="radio"`，`v-for`） | 同上 + `profiles.length` | `BrowserDataSettings.vue:285-300` | 未出现（条件） | — |
| S-general-51 | 未找到 Google Chrome 配置。 | 文本 | 同上 + `v-else-if="!importError"` | `BrowserDataSettings.vue:303` | 未出现（条件） | — |
| S-general-52 | 正在读取所选配置… | 文本 | 同上 + `previewing` | `BrowserDataSettings.vue:305-308` | 未出现（条件） | — |
| S-general-53 | `capabilityLabel(...)` + 重试 | 文本 + 按钮 | 同上 + `requires_action` | `BrowserDataSettings.vue:309-316` | 未出现（条件） | — |
| S-general-54 | 可导入 {importable} · 会话 {session} · 已过期 {expired} · 分区 {partitioned} | 文本 | 同上 + `v-else-if="preview"` | `BrowserDataSettings.vue:318-320` | 未出现（条件） | — |
| S-general-55 | 已导入 {imported} · 失败 {failed} · 已跳过… | 文本（含 `TriangleAlert` / `Check` 图标） | 同上 + `importResult` | `BrowserDataSettings.vue:321-330` | 未出现（条件） | — |
| S-general-56 | 部分 Cookie 未能导入，部分网站可能需要重新登录。 | 文本（role=status） | 同上 + `failedCount > 0` | `BrowserDataSettings.vue:332-334` | 未出现（条件） | — |
| S-general-57 | `t(importError.key)` · `diagnosticLine(...)` | 文本（role=alert） | 同上 + `importError?.key` | `BrowserDataSettings.vue:336-339` | 未出现（条件） | — |
| S-general-58 | 重试 | 按钮 | 同上 | `BrowserDataSettings.vue:341-349` | 未出现（条件） | — |
| S-general-59 | 关闭 / 取消（`importResult ? 关闭 : 取消`） | 对话框按钮 | 同上 | `BrowserDataSettings.vue:355-357` | 未出现（条件） | — |
| S-general-60 | 导入（`importing ? … : 导入`） | 对话框按钮 | 同上 + `!importResult` | `BrowserDataSettings.vue:358-365` | 未出现（条件） | — |
| S-general-61 | 清理浏览器数据（对话框） | 对话框 | 点击 S-general-45 | `BrowserDataSettings.vue:370-380` | 未出现（条件：对话框未打开） | — |
| S-general-62 | 此操作只会清除内置浏览器中的登录状态、Cookie… | 对话框说明 | 同上 | `BrowserDataSettings.vue:380` | 未出现（条件） | — |
| S-general-63 | 内置浏览器数据已清理。 | 文本（含 `Check`） | 同上 + `clearSucceeded` | `BrowserDataSettings.vue:383-386` | 未出现（条件） | — |
| S-general-64 | `t(clearError.key)` · `diagnosticLine(...)` | 文本（role=alert） | 同上 + `clearError?.key` | `BrowserDataSettings.vue:387-389` | 未出现（条件） | — |
| S-general-65 | 关闭 / 取消（`clearSucceeded ? 关闭 : 取消`） | 对话框按钮 | 同上 | `BrowserDataSettings.vue:393-395` | 未出现（条件） | — |
| S-general-66 | 清理浏览器数据（`clearing ? … : 清理浏览器数据`） | 对话框按钮 | 同上 + `!clearSucceeded` | `BrowserDataSettings.vue:396-399` | 未出现（条件） | — |

### 5.5 数据清理

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-general-67 | 数据清理 | 文本标签 | 常显 | `GeneralSettings.vue:211` | 可见 | 272,779 282×18 |
| S-general-68 | 会话存储由 App Server 管理，可在此查看其存储用量报告。（否则「移除未使用的本地数据并优化存储。」） | 文本 | `appServerMode` 分两支 | `GeneralSettings.vue:212-214` | 可见（App Server 支） | 272,799 282×14 |
| S-general-69 | 查看… | 按钮 | 常显 | `GeneralSettings.vue:216-222` | 可见 | 917,780 66×32 |
| S-general-70 | `dialogTitle` / `dialogDescription` | 对话框标题 + 说明 | 点击 S-general-69 | `DataCleanupDialog.vue:342-345` | 未出现（条件：对话框未打开） | — |
| S-general-71 | 正在扫描本地数据… | 文本 | `state.kind === 'scanning'` | `DataCleanupDialog.vue:348-354` | 未出现（条件） | — |
| S-general-72 | 数据库 / WAL / 可回收（`dt`+`dd` ×3） | 文本·指标 | `state.kind === 'ready' && isAppServerPlan` | `DataCleanupDialog.vue:361-372` | 未出现（条件） | — |
| S-general-73 | 会话存储由 App Server 管理，此应用不支持跨归属方的扫描和清理。 | 文本 | 同上 | `DataCleanupDialog.vue:375-377` | 未出现（条件） | — |
| S-general-74 | 如需回收可回收空间，请停止守护进程并离线运行 `dimcode daemon vacuum`。 | 文本 | 同上 | `DataCleanupDialog.vue:378-380` | 未出现（条件） | — |
| S-general-75 | 数据库历史 / 未使用的文件历史备份 / 临时附件和缓存 / 过期快照对象（`dt`+`dd` ×4） | 文本·指标 | `state.kind === 'ready'`（非 App Server 方案） | `DataCleanupDialog.vue:383-401` | 未出现（条件） | — |
| S-general-76 | 优化数据库时可能需要额外的临时磁盘空间。 | 文本 | 同上 | `DataCleanupDialog.vue:404-406` | 未出现（条件） | — |
| S-general-77 | 正在优化本地存储… | 文本 | `state.kind === 'running'` | `DataCleanupDialog.vue:409-410` | 未出现（条件） | — |
| S-general-78 | phaseIcon / 文件历史 / 未使用的本地数据 / 本地数据库 | 文本·步骤（`v-for` 三阶段） | 同上 | `DataCleanupDialog.vue:411-431` | 未出现（条件） | — |
| S-general-79 | 清理完成前请保持 DimAgent 开启。 | 文本 | 同上 | `DataCleanupDialog.vue:434` | 未出现（条件） | — |
| S-general-80 | 清理需要独占访问本地数据。 · `state.message` | 文本 | `state.kind === 'blocked'` | `DataCleanupDialog.vue:437-439` | 未出现（条件） | — |
| S-general-81 | 存储报告不可用 · `state.message` | 文本 | `state.kind === 'unavailable'` | `DataCleanupDialog.vue:442-448` | 未出现（条件） | — |
| S-general-82 | 清理失败 · `state.message` | 文本 | `state.kind === 'error'` | `DataCleanupDialog.vue:451-453` | 未出现（条件） | — |
| S-general-83 | 清理完成 / 部分清理已完成 | 文本 | `v-else` + `state.partial` | `DataCleanupDialog.vue:456-459` | 未出现（条件） | — |
| S-general-84 | 已回收 / 数据库 / 已移除临时文件（`dt`+`dd` ×3） | 文本·指标 | 同上 | `DataCleanupDialog.vue:460-475` | 未出现（条件） | — |
| S-general-85 | `t(key)`（部分完成提示，`v-for`） | 文本·列表项 | 同上 + `partialNoticeKeys.length > 0` | `DataCleanupDialog.vue:478-483` | 未出现（条件） | — |
| S-general-86 | 关闭 | 对话框按钮 | `state.kind === 'ready' && isAppServerPlan` | `DataCleanupDialog.vue:488-494` | 未出现（条件） | — |
| S-general-87 | 取消 / 清理 | 对话框按钮 | `state.kind === 'ready'` | `DataCleanupDialog.vue:495-499` | 未出现（条件） | — |
| S-general-88 | 取消 / 重试 | 对话框按钮 | `blocked` / `error` / `unavailable` | `DataCleanupDialog.vue:501-503` | 未出现（条件） | — |
| S-general-89 | 关闭 | 对话框按钮 | `state.kind === 'completed'` | `DataCleanupDialog.vue:505-507` | 未出现（条件） | — |
| S-general-90 | 取消 | 对话框按钮 | `v-else` | `DataCleanupDialog.vue:508-510` | 未出现（条件） | — |

本节行数 90（图标 4）｜可见 27｜未出现（条件）63｜未命中 0｜运行时独有 0
---

## 6. 快捷键（`components/settings/KeyboardShortcutsSettings.vue`）

png：[快捷键-top.png](runtime/settings/快捷键-top.png)、[快捷键-bottom.png](runtime/settings/快捷键-bottom.png)。运行时 118 节点 = 侧栏 40 + 内容 20 + 侧栏 38 + 内容 20；内容首趟 y 33–700。

模板按 `v-for="scope in ['app', 'chat']"` 分两组，组内再按 `definitionsByScope(scope)` 逐条渲染（8 条），每行 = 名称 + 绑定按钮 + 条件出现的「重置」。

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-keybindings-01 | 快捷键 | 页面标题 | 常显 | `KeyboardShortcutsSettings.vue:126` | 可见 | 257,33 741×25 |
| S-keybindings-02 | `keybindingsSettings.loadFailed` | 文本 | `loadError` | `:129` | 未出现（条件：加载成功） | — |
| S-keybindings-03 | 搜索快捷键…（`searchPlaceholder`） | 控件·输入 | 常显 | `:130` | 可见（占位符不入 `text`） | 257,85 392×32 |
| S-keybindings-04 | 通用 | 分组标题 | `scope === 'app'` | `:133` | 可见 | 257,145 741×18 |
| S-keybindings-05 | 打开设置 | 文本·行标签 | 同上 + `definition` 1/4 | `:137` | 可见 | 272,195 49×18 |
| S-keybindings-06 | ⌘,（绑定按钮） | 控件·按钮 | 同上（未录音） | `:145-153` | 可见 | 937,188 45×32 |
| S-keybindings-07 | 重置 | 按钮 | 同上 + 绑定 ≠ 默认 | `:154-156` | 未出现（条件：仍为默认绑定） | — |
| S-keybindings-08 | 新建线程 | 文本·行标签 | 同上（2/4） | `:137` | 可见 | 272,255 49×18 |
| S-keybindings-09 | ⌘N | 控件·按钮 | 同上 | `:145-153` | 可见 | 932,248 51×32 |
| S-keybindings-10 | 切换侧边栏 | 文本·行标签 | 同上（3/4） | `:137` | 可见 | 272,316 61×18 |
| S-keybindings-11 | ⌘B | 控件·按钮 | 同上 | `:145-153` | 可见 | 933,309 49×32 |
| S-keybindings-12 | 搜索对话 | 文本·行标签 | 同上（4/4） | `:137` | 可见 | 272,376 49×18 |
| S-keybindings-13 | ⌘G | 控件·按钮 | 同上 | `:145-153` | 可见 | 932,369 50×32 |
| S-keybindings-14 | 对话 | 分组标题 | `scope === 'chat'` | `:133` | 可见 | 257,444 741×18 |
| S-keybindings-15 | 聚焦输入框 | 文本·行标签 | 同上（1/4） | `:137` | 可见 | 272,494 61×18 |
| S-keybindings-16 | ⌘L | 控件·按钮 | 同上 | `:145-153` | 可见 | 934,487 48×32 |
| S-keybindings-17 | 切换更改面板 | 文本·行标签 | 同上（2/4） | `:137` | 可见 | 272,554 74×18 |
| S-keybindings-18 | ⌘⇧B | 控件·按钮 | 同上 | `:145-153` | 可见 | 921,547 62×32 |
| S-keybindings-19 | 切换终端 | 文本·行标签 | 同上（3/4） | `:137` | 可见 | 272,615 49×18 |
| S-keybindings-20 | ⌘J | 控件·按钮 | 同上 | `:145-153` | 可见 | 934,608 48×32 |
| S-keybindings-21 | 提交更改 | 文本·行标签 | 同上（4/4） | `:137` | 可见 | 272,675 49×18 |
| S-keybindings-22 | ⌘⇧↵ | 控件·按钮 | 同上 | `:145-153` | 可见 | 917,668 66×32 |
| S-keybindings-23 | 录音中…（`recording`） | 按钮文案分支 | `recordingId === definition.id` | `:152` | 未出现（条件：未在录音） | — |
| S-keybindings-24 | `invalid` | 文本 | `errorById[id] === 'invalid'` | `:138` | 未出现（条件：无键位错误） | — |
| S-keybindings-25 | `conflict`（含冲突键位） | 文本 | `errorById[id] === 'conflict'` | `:139-141` | 未出现（条件：无冲突） | — |
| S-keybindings-26 | `saveFailed` | 文本 | `errorById[id] === 'save'` | `:142` | 未出现（条件：保存成功） | — |
| S-keybindings-27 | `common.noDescription`（空态） | 文本 | `definitionsByScope(scope).length === 0` | `:159` | 未出现（条件：两组均有条目） | — |

本节行数 27（图标 0）｜可见 18｜未出现（条件）9｜未命中 0｜运行时独有 0
---

## 7. 模型供应（`components/settings/ConfigurationSettings.vue`）

png：[模型供应-top.png](runtime/settings/模型供应-top.png)、[模型供应-bottom.png](runtime/settings/模型供应-bottom.png)。运行时 1089 节点 = 首趟 555（侧栏 40 + 供应商列表 394 + 详情 121）+ 次趟 534；内容区是双栏：供应商列表 x 234–443、详情 x 466–980。当前状态为「选中 DimAgent OAuth（由 Dim 账户管理）」，列表 1 个启用分组 + 127 个未启用项。

本节是全 19 节最大的一节（模板 2070–4033 行）。行按「面板/分支」拆：数据驱动的列表（供应商、模型）按 1 行记模板节点，实例数写在运行时列。

### 7.1 左栏供应商列表

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-configuration-01 | 模型供应 | 栏标题 | 常显 | `:2102` | 可见 | 239,20 46×13 |
| S-configuration-02 | 搜索供应商 | 控件·输入 | 常显 | `:2104-2107` | 可见 | 259,48 175×18 |
| S-configuration-03 | ↳ `Search` 图标 | 图标 | 常显 | `:2106` | 可见（png） | 输入框内 |
| S-configuration-04 | 清除搜索 | 图标按钮 | `providerFilter !== ''` | `:2114` | 未出现（条件：未输入搜索词） | — |
| S-configuration-05 | 分组折叠头 `t(group.labelKey)` · `group.providers.length` | 按钮（`v-for="group in providerListGroups"`） | 常显（2 组：已启用 1 / 未启用 127） | `:2128-2143` | 可见（2 实例） | 已启用 234,77 209×25 | 未启用 234,134 209×25 |
| S-configuration-06 | ↳ `ChevronDown` / `ChevronRight` 图标 | 图标 | `!group.collapsed` / `v-else` | `:2140-2141` | 可见（轮廓） | 同上 |
| S-configuration-07 | `provider.name` · 状态点 `●`/`○` | 列表项按钮（`v-for="provider in group.providers"`） | `!group.collapsed` | `:2147-2162` | 可见（128 实例） | 如 DimAgent OAuth 234,103 209×28；302.AI 234,161 209×28 |
| S-configuration-08 | ↳ 供应商图标 `ProviderIcon` | 图标 | 同上 | `:2157` | 可见（icon:true 于已启用项） | 同上 |
| S-configuration-09 | 添加 | 按钮（含 `Plus` 图标） | `canAddCustomProvider` | `:2168-2178` | 可见 | 234,3937 209×28 |

### 7.2 详情：头部与账户托管态（当前可见分支）

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-configuration-10 | 返回 | 图标按钮（窄屏） | `isCompact` | `:2189-2198` | 未出现（条件：非窄屏） | — |
| S-configuration-11 | `selectedProvider.name`（DimAgent OAuth） | 文本·标题 | `v-else-if="selectedProvider"` | `:2394` 内头部 | 可见 | 489,22 98×18 |
| S-configuration-12 | 已启用 / 未启用 | 文本·状态 | 同上 | 同上 | 可见 | 489,40 98×14 |
| S-configuration-13 | 启用开关 | 开关 | 同上 | 同上 | 可见 | 980,30 28×16 |
| S-configuration-14 | 由 Dim 账户管理 | 容器·提示条 + 标题（含 `ShieldCheck`） | `isSelectedProviderAccountManaged` + `isSelectedProviderConfigured` | `:2915-2919` | 可见 | 466,83 542×53（标题 502,92） |
| S-configuration-15 | 此提供商由你的 Dim 账户管理，无法在此编辑。 | 文本 | 同上 | `:2919` | 可见 | 502,112 495×14 |
| S-configuration-16 | 登录后可使用 Dim 模型。点击跳转到账户设置。 | 文本 | 同上 + `v-else`（未登录） | `:2922` | 未出现（条件：已登录） | — |
| S-configuration-17 | 基础 URL | 文本标签 | 同上 | `:2928` | 可见 | 466,153 542×14 |
| S-configuration-18 | `selectedProvider.baseUrl`（https://dimagent.cn/v1） | 文本·值（code） | 同上 | `:2931` | 可见 | 466,172 542×32 |
| S-configuration-19 | `formError` | 文本 | `formError` | `:2940` | 未出现（条件：无错误） | — |
| S-configuration-20 | 模型 · {visible}/{total} | 控件·复选框 + 标签 | 同上 | `:2956` | 可见 | 466,221 14×14（标签 487,221 65×14） |
| S-configuration-21 | 搜索模型名称或 ID | 控件·输入 | `selectedProvider.models.length > 0` | `:2963-2968` | 可见 | 493,248 507×18 |
| S-configuration-22 | 清除搜索（模型） | 图标按钮 | `modelFilter` | `:2968` | 未出现（条件：未输入） | — |
| S-configuration-23 | `formatModelName(model)` + 能力徽标（视觉 / 推理 / 工具 / 自定义 / 媒体能力）+ 开关 | 列表行（`v-for="model in visibleSelectedProviderModels"`） | 同上 | `:2972-3000` | 可见（32 实例，如 vokie-dp-v4-flash 478,287） | 478,287 106×18 起，逐行 +33 |
| S-configuration-24 | 没有匹配的模型 | 文本 | `visibleSelectedProviderModels.length === 0` | `:3006` | 未出现（条件：有匹配） | — |
| S-configuration-25 | 登录成功后将自动拉取可用模型。 | 文本 | `v-else`（无模型） | `:3014` | 未出现（条件：已有模型） | — |
| S-configuration-26 | 登录 Dim 以解锁模型。 | 文本 | 同上 | `:3017` | 未出现（条件） | — |
| S-configuration-27 | 刷新 | 按钮 | 常显（详情底部动作） | `:3031` | 可见 | 840,675 69×28 |
| S-configuration-28 | 查看套餐 | 按钮 | 同上 | `:3040` | 可见 | 915,675 93×28 |

### 7.3 详情：其他认证分支（均未在当前状态出现）

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-configuration-29 | 登录 {provider} 以加载其模型。 | 文本 | `isSelectedProviderOpenAICodex` + `signed-out`/`error` | `:2420-2437` | 未出现（条件：选中的是账户托管供应商） | — |
| S-configuration-30 | 使用设备代码登录 / 使用浏览器登录 | 按钮 | 同上（两种登录方式） | `:2444-2513` | 未出现（条件） | — |
| S-configuration-31 | 回调地址 · `redirectUri` · 打开 URL · 复制 · 确认 | 文本 + 按钮 | `openAICodexAuthState === 'pending-browser'` | `:2517-2538` | 未出现（条件） | — |
| S-configuration-32 | 正在等待授权... · 取消 | 文本 + 按钮 | 同上（设备码流程） | `:3092-3095` | 未出现（条件） | — |
| S-configuration-33 | 套餐: {planType} · 账户: {accountIdPreview} · 刷新 · 退出登录 | 文本 + 按钮 | `openAICodexAuthState === 'authenticated'` | `:2523-2538` | 未出现（条件） | — |
| S-configuration-34 | 已登录。模型开关决定聊天中显示哪些模型。 | 文本 | 同上 | `:3185` | 未出现（条件） | — |
| S-configuration-35 | （禁用态说明） | 文本 | `openAICodexAuthState === 'disabled'` | `:2565` | 未出现（条件） | — |
| S-configuration-36 | `openAICodexVisibleError` | 文本 | 非空 | `:2580` | 未出现（条件） | — |
| S-configuration-37 | OAuth 令牌已生效。模型开关决定聊天中可用的 Grok Build 模型。 | 文本 | `isSelectedProviderGrokBuild` + `authenticated` | `:2788` | 未出现（条件） | — |
| S-configuration-38 | 设备码 · `userCode` · 打开验证页面 · 复制 | 文本 + 按钮 | `grokBuildAuthState === 'pending-device'` | `:2695-2746` | 未出现（条件） | — |
| S-configuration-39 | `grokBuildVisibleError` | 文本 | 非空 | `:2798` | 未出现（条件） | — |
| S-configuration-40 | 登录后加载 Grok Build 模型。 | 文本 | `v-else` | `:2898` | 未出现（条件） | — |
| S-configuration-41 | 登录成功后将自动拉取可用模型。 | 文本 | 通用 OAuth 未登录态 | `:3250` | 未出现（条件） | — |
| S-configuration-42 | `genericOAuthAccountLabel` · 刷新 · 退出登录 | 文本 + 按钮 | `genericOAuthStatus` 已登录 | `:3172-3180` | 未出现（条件） | — |
| S-configuration-43 | `genericOAuthVisibleError` | 文本 | 非空 | `:3197` | 未出现（条件） | — |

### 7.4 详情：API Key / AWS 面板（未出现）

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-configuration-44 | Authentication · API Key / Access Key | 文本标签 + 分段项 | `isSelectedProviderAwsBedrock` 分两支 | `:3260-3265` | 未出现（条件：当前为账户托管） | — |
| S-configuration-45 | Access Key ID / Secret Access Key / Session Token | 控件·输入 | AWS Bedrock 供应商 | `:3271-3279` | 未出现（条件） | — |
| S-configuration-46 | API 密钥 | 控件·输入 | 非 AWS | `:3284` | 未出现（条件） | — |
| S-configuration-47 | Region / 基础 URL | 文本标签 + 输入 | `isSelectedProviderAwsBedrock` 分两支 | `:3294` | 未出现（条件） | — |
| S-configuration-48 | `formError` | 文本 | `formError` | `:3316` | 未出现（条件） | — |
| S-configuration-49 | 连接并获取模型 / 保存 | 按钮 | `selectedProviderSupportsRemoteModelList` 分两支 | `:3324` | 未出现（条件） | — |
| S-configuration-50 | 测试连接 · 保存 | 按钮 | 详情底部动作 | `:3570-3574` | 未出现（条件） | — |
| S-configuration-51 | `usageError` | 文本 | `usageError` | `:3582` | 未出现（条件） | — |
| S-configuration-52 | 用量 · 账户 | 文本标签（面板切换） | 详情页签 | `:3475-3489` | 未出现（条件：当前未展开） | — |
| S-configuration-53 | API Key / Team · 基础 URL · Organization ID · Project ID | 文本标签 + 输入 | 用量/账户面板 | `:3495-3525` | 未出现（条件） | — |
| S-configuration-54 | 还没有模型数据，点击「刷新」获取 | 文本 | 无模型数据 | `:3708` | 未出现（条件：已有模型） | — |

### 7.5 详情：编辑态与危险动作（未出现）

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-configuration-55 | 协议（`ADAPTER_LABELS[adapter]`） | 文本标签 + 单选项 | 编辑态 | `:3340-3343` | 未出现（条件：只读账户托管态） | — |
| S-configuration-56 | 名称 | 文本标签 + 输入 | 编辑态 | `:3346` | 未出现（条件） | — |
| S-configuration-57 | 基础 URL | 文本标签 + 输入 | 编辑态 | `:3350` | 未出现（条件） | — |
| S-configuration-58 | API 密钥 | 文本标签 + 输入 | 编辑态 | `:3354` | 未出现（条件） | — |
| S-configuration-59 | User-Agent + 留空则使用运行时默认值。仅应用于此供应商的模型获取和模型请求。 | 输入 + 文本 | 编辑态 | `:3363-3365` | 未出现（条件） | — |
| S-configuration-60 | 取消 · 保存 | 按钮 | 编辑态 | `:3370-3374` | 未出现（条件） | — |
| S-configuration-61 | API 密钥 · •••••••• · 修改 | 文本·值 + 按钮 | 已存密钥态 | `:3419-3420` | 未出现（条件） | — |
| S-configuration-62 | 断开此提供商 · 断开提供商 · 取消 · 断开连接 | 对话框 + 按钮 | 断开确认 | `:3723-3734` | 未出现（条件：对话框未打开） | — |
| S-configuration-63 | 删除此提供商 · 删除提供商 · 取消 · 删除 | 对话框 + 按钮 | 删除确认 | `:3745-3756` | 未出现（条件） | — |

### 7.6 添加自定义提供商 / 添加模型

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-configuration-64 | 添加自定义提供商 | 面板标题 | `panelMode === 'add-custom'` | `:2200-2202` | 未出现（条件：当前在详情态） | — |
| S-configuration-65 | 协议（`ADAPTER_LABELS`） | 文本标签 + 5 个单选项（openai / openai-responses / anthropic / gemini / new-api） | 同上 | `:2206-2222` | 未出现（条件） | — |
| S-configuration-66 | 名称 | 文本标签 + 输入（占位「我的自定义提供商」） | 同上 | `:2228-2229` | 未出现（条件） | — |
| S-configuration-67 | 基础 URL | 文本标签 + 输入 | 同上 | `:2233-2234` | 未出现（条件） | — |
| S-configuration-68 | API 密钥 | 文本标签 + 输入（占位 `sk-...`） | 同上 | `:2238-2239` | 未出现（条件） | — |
| S-configuration-69 | User-Agent + 留空则使用运行时默认值… | 输入 + 文本 | 同上 | `:2243-2245` | 未出现（条件） | — |
| S-configuration-70 | 模型 · {visible}/{total} | 复选框 + 标签 | 同上 | `:2251-2260` | 未出现（条件） | — |
| S-configuration-71 | 添加模型 · 刷新 | 按钮（含 `Plus` / `RefreshCw`） | 同上 | `:2263-2281` | 未出现（条件） | — |
| S-configuration-72 | 搜索模型名称或 ID | 控件·输入 | `draftModels.length > 0` | `:2286-2295` | 未出现（条件） | — |
| S-configuration-73 | `formatModelName(model)` + 能力徽标 + 开关 | 列表行（`v-for="model in visibleDraftModels"`） | 同上 | `:2311-2351` | 未出现（条件） | — |
| S-configuration-74 | 没有匹配的模型 | 文本 | `visibleDraftModels.length === 0` | `:2357-2362` | 未出现（条件） | — |
| S-configuration-75 | 还没有模型。可以手动添加，也可以在提供商支持模型列表时点击刷新。 | 文本 | `v-else` | `:2364-2369` | 未出现（条件） | — |
| S-configuration-76 | `formError` | 文本 | `formError` | `:2373` | 未出现（条件） | — |
| S-configuration-77 | 保存（`isSaving` 时 `Loader2`，否则 `ChevronRight`） | 按钮 | 常显 | `:2385-2386` | 未出现（条件） | — |
| S-configuration-78 | 添加自定义模型 · 从空白配置开始；搜索内容会预填为模型 ID。 | 对话框标题 + 说明 | 添加模型对话框 | `:3796-3799` | 未出现（条件：对话框未打开） | — |
| S-configuration-79 | 使用模板创建 · 选择预设配置作为起点，保存前仍可修改所有字段。 | 对话框标题 + 说明 | 模板创建对话框 | `:3803-3806` | 未出现（条件） | — |
| S-configuration-80 | 没有匹配的模型 | 文本 | 模板/搜索空态 | `:3813` | 未出现（条件） | — |

### 7.7 其他分支

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-configuration-81 | 加载骨架（`Skeleton` ×4 / ×4） | 容器·骨架 | `loading && !loaded` | `:2078-2089` | 未出现（条件：已加载） | — |
| S-configuration-82 | 选择一个提供商查看详情 | 文本·空态 | `v-else`（无选中） | `:3769` | 未出现（条件：已选中 DimAgent OAuth） | — |
| S-configuration-83 | 由 Dim 账户管理 · 登录后可使用 Dim 模型。点击跳转到账户设置。 | 文本 | `isSelectedProviderAccountManaged` + 未登录 | `:2915-2922` | 未出现（条件：已登录） | — |

本节行数 83（图标 3）｜可见 16｜未出现（条件）67｜未命中 0｜运行时独有 0

运行时对齐（515 节点）：供应商列表 394 项 = 2 个分组头（S-configuration-05）、128 个供应商项（S-configuration-07）、图标节点、搜索框与「添加」；详情 121 项 = 头部 3 项、账户托管提示 2 项、基础 URL 2 项、「模型 32/32」2 项、搜索框 1 项、32 个模型行 ×（名称 + 能力徽标 + 开关）、底部 2 个按钮。列表项与模型行都是数据驱动，按模板节点各记 1 行，实例数写在上表「运行时」列。
---

## 8. 多模态（`components/settings/ModalityDefaultsSetting.vue` + `ModalityModelPicker` / `ModalityTaskPanel` / `ModalityRealtimePanel`）

png：[多模态-top.png](runtime/settings/多模态-top.png)、[多模态-bottom.png](runtime/settings/多模态-bottom.png)。运行时 218 节点 = 侧栏 40 + 内容 70 + 侧栏 38 + 内容 70；内容首趟 y 33–1564。当前状态：App Server 未声明 modality 能力，5 个能力分组各 0/N，任务与实时面板均处于「未声明方法」分支。

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-modality-01 | 多模态 | 页面标题 | 常显 | `ModalityDefaultsSetting.vue:200` | 可见 | 257,33 46×25 |
| S-modality-02 | ↳ `Loader2` 图标 | 图标 | `loading` | `:202` | 未出现（条件：已加载） | — |
| S-modality-03 | `unavailableReason` | 文本·提示条（含 `AlertCircle`） | `unavailableReason` | `:205-211` | 可见（The App Server does not expose modality.capabilities.） | 288,82 315×18 |
| S-modality-04 | `errorText` | 文本·提示条 | `errorText` | `:214-219` | 未出现（条件：无错误） | — |
| S-modality-05 | 能力分组页签条 | 控件·标签栏（`TabsList`，`aria-label="多模态"`） | `!isCompact` | `:237-243` | 可见 | 257,124 741×73 |
| S-modality-06 | 图像 · 0 / 4 | 控件·页签（`v-for="group in capabilityGroups"`，1/5） | 同上 | `:243-256` | 可见 | 257,124 148×72（文字 267,134、计数 267,171） |
| S-modality-07 | 视频 · 0 / 3 | 页签（2/5） | 同上 | 同上 | 可见 | 405,124 148×72 |
| S-modality-08 | 语音与音频 · 0 / 4 | 页签（3/5） | 同上 | 同上 | 可见 | 553,124 148×72 |
| S-modality-09 | 创作与工具 · 0 / 3 | 页签（4/5） | 同上 | 同上 | 可见 | 701,124 148×72 |
| S-modality-10 | 实时 · 0 / 3 | 页签（5/5） | 同上 | 同上 | 可见 | 849,124 148×72 |
| S-modality-11 | 分组下拉（窄屏） | 控件·下拉 | `isCompact` | `:223-232` | 未出现（条件：非窄屏） | — |
| S-modality-12 | 图像（当前分组面板） | 容器·面板（`tabpanel`） | `!isCompact \|\| activeGroupId === group.id` | `:263-270` | 可见 | 257,217 741×326 |
| S-modality-13 | 图像 | 分组标题（`h2`） | 同上 | `:271` | 可见 | 257,217 27×21 |
| S-modality-14 | 0 / 4 | 文本·计数 | 同上 | `:272-274` | 可见 | 974,222 23×14 |
| S-modality-15 | 图像生成 · 选择模型 | 能力行 + 模型选择器（`v-for="capabilityId in displayedCapabilityIds(group)"`） | 同上 | `:278-288` | 可见（4 行：图像生成 / 图像编辑 / 图像增强 / 图像理解） | 259（行标签 257,259 588×18；选择器 257,284 588×28） |
| S-modality-16 | ↳ `triggerPlaceholder`（选择模型） | 文本·占位 | 同上（未选模型） | `ModalityModelPicker.vue:240-242` | 可见 | 268,289 49×18 |
| S-modality-17 | ↳ `ChevronDown` 图标 | 图标 | 同上 | `ModalityModelPicker.vue:243` | 可见（png） | 选择器内 |
| S-modality-18 | `providerName / modelName` · 自动 | 文本·已选模型 | 同上 + `selected` | `ModalityModelPicker.vue:219-238` | 未出现（条件：未选模型） | — |
| S-modality-19 | `operationReason(capabilityId)` | 文本 | 同上 + 非空 | `:297-303` | 未出现（条件：无禁用原因） | — |
| S-modality-20 | 模型选择弹层 | 气泡（`PopoverContent`） | 点击 S-modality-15 的选择器 | `ModalityModelPicker.vue:247-322` | 未出现（条件：弹层未打开） | — |
| S-modality-21 | 搜索提供商或模型 | 控件·输入 | 同上 | `ModalityModelPicker.vue:249` | 未出现（条件） | — |
| S-modality-22 | 提供商 · `group.providerName` | 分组标题 + 列表项（`v-for`） | 同上 | `ModalityModelPicker.vue:261-274` | 未出现（条件） | — |
| S-modality-23 | 无可用提供商 | 文本 | 同上 + `filteredGroups.length === 0` | `ModalityModelPicker.vue:279-284` | 未出现（条件） | — |
| S-modality-24 | 模型 · `modelName(...)` | 分组标题 + 列表项（`v-for`） | 同上 | `ModalityModelPicker.vue:291-305` | 未出现（条件） | — |
| S-modality-25 | `option.reason.detail` | 文本 | 同上 + 有禁用原因 | `ModalityModelPicker.vue:310-315` | 未出现（条件） | — |
| S-modality-26 | 无可用模型 | 文本 | 同上 + `activeModels.length === 0` | `ModalityModelPicker.vue:317-322` | 未出现（条件） | — |
| S-modality-27 | 任务 | 分组标题（`h2`） | 常显 | `ModalityTaskPanel.vue:309` | 可见 | 257,600 741×21 |
| S-modality-28 | 提交并跟踪由 App Server 拥有的 Modality 任务。 | 文本 | 常显 | `ModalityTaskPanel.vue:310` | 可见 | 257,624 741×18 |
| S-modality-29 | App Server 未声明 Modality 任务方法。 | 文本·提示条 | `!enabled` | `ModalityTaskPanel.vue:313-319` | 可见 | 288,663 223×18 |
| S-modality-30 | 提交任务 | 分组标题（`h3`） | 常显 | `ModalityTaskPanel.vue:323` | 可见 | 272,716 642×18 |
| S-modality-31 | 类别 | 文本标签 + 输入（`:disabled`） | 常显 | `ModalityTaskPanel.vue:325-327` | 可见 | 272,745 25×15（输入 272,762 316×32） |
| S-modality-32 | 操作 | 文本标签 + 输入（`:disabled`） | 常显 | `ModalityTaskPanel.vue:334-336` | 可见 | 598,745 25×15（输入 598,762 316×32） |
| S-modality-33 | 参数（JSON 对象） | 文本标签 + 多行输入（`:disabled`） | 常显 | `ModalityTaskPanel.vue:344-346` | 可见 | 272,805 110×15（输入 272,821 642×72） |
| S-modality-34 | 文件（JSON 数组） | 文本标签 + 多行输入（`:disabled`） | 常显 | `ModalityTaskPanel.vue:354-356` | 可见 | 272,909 110×15（输入 272,926 642×58） |
| S-modality-35 | 提供商（可选） | 文本标签 + 输入（`:disabled`） | 常显 | `ModalityTaskPanel.vue:365-367` | 可见 | 272,1000 86×15（输入 272,1016 207×32） |
| S-modality-36 | 模型（可选） | 文本标签 + 输入（`:disabled`） | 常显 | `ModalityTaskPanel.vue:374-376` | 可见 | 489,1000 74×15（输入 489,1016 207×32） |
| S-modality-37 | 客户端任务 ID（可选） | 文本标签 + 输入（`:disabled`） | 常显 | `ModalityTaskPanel.vue:383-385` | 可见 | 707,1000 126×15（输入 707,1016 207×32） |
| S-modality-38 | 提交任务（`submitting ? 提交中… : 提交任务`） | 按钮（`:disabled`） | 常显 | `ModalityTaskPanel.vue:394-403` | 可见 | 272,1058 70×32 |
| S-modality-39 | ↳ `Loader2` 图标 | 图标 | `submitting` | `ModalityTaskPanel.vue:401` | 未出现（条件：未提交中） | — |
| S-modality-40 | `submitError` | 文本 | `submitError` | `ModalityTaskPanel.vue:405-407` | 未出现（条件：无错误） | — |
| S-modality-41 | 所有者任务 | 分组标题（`h3`） | 常显 | `ModalityTaskPanel.vue:412` | 可见 | 272,1141 61×18 |
| S-modality-42 | 状态筛选（全部状态 / `state`） | 控件·下拉（`:disabled`） | 常显 | `ModalityTaskPanel.vue:414-422` | 可见 | 738,1134 101×32 |
| S-modality-43 | 刷新 | 按钮（含 `RefreshCw`，`:disabled`） | 常显 | `ModalityTaskPanel.vue:424-433` | 可见 | 845,1134 69×32 |
| S-modality-44 | `listError` | 文本 | `listError` | `ModalityTaskPanel.vue:437` | 未出现（条件：无错误） | — |
| S-modality-45 | 没有任务。 | 文本 | `v-else-if="tasks.length === 0"` | `ModalityTaskPanel.vue:438-440` | 可见 | 272,1176 642×18 |
| S-modality-46 | `task.task_id` · `category.task.operation` · `state` · `created_at` | 任务列表项（`v-for="task in tasks"`） | `v-else` | `ModalityTaskPanel.vue:441-452` | 未出现（条件：任务为空） | — |
| S-modality-47 | 加载更多 | 按钮 | `hasMore` | `ModalityTaskPanel.vue:456-465` | 未出现（条件） | — |
| S-modality-48 | 任务详情 | 分组标题（`h3`） | `selected` | `ModalityTaskPanel.vue:469` | 未出现（条件：未选中任务） | — |
| S-modality-49 | 状态 / 修订 / 更新时间 / 进度（`dt`+`dd` ×4） | 文本·指标 | 同上 | `ModalityTaskPanel.vue:470-474` | 未出现（条件） | — |
| S-modality-50 | `selected.error.code: selected.error.message` | 文本 | 同上 + `selected.error` | `ModalityTaskPanel.vue:476-478` | 未出现（条件） | — |
| S-modality-51 | `actionError` · `waitNote` | 文本 | 同上 | `ModalityTaskPanel.vue:479-480` | 未出现（条件） | — |
| S-modality-52 | 刷新状态 | 按钮 | 同上 | `ModalityTaskPanel.vue:483-490` | 未出现（条件） | — |
| S-modality-53 | 等待超时（毫秒） | 文本标签 + 输入 | 同上 | `ModalityTaskPanel.vue:491-493` | 未出现（条件） | — |
| S-modality-54 | 等待（`waiting ? 等待中… : 等待`） | 按钮 | 同上 | `ModalityTaskPanel.vue:502-511` | 未出现（条件） | — |
| S-modality-55 | 实时会话 | 分组标题（`h2`） | 常显 | `ModalityRealtimePanel.vue:223` | 可见 | 257,1251 741×21 |
| S-modality-56 | 通过 App Server 打开并驱动 Modality 实时会话。 | 文本 | 常显 | `ModalityRealtimePanel.vue:224` | 可见 | 257,1276 741×18 |
| S-modality-57 | App Server 未声明 Modality 实时方法。 | 文本·提示条 | `!enabled` | `ModalityRealtimePanel.vue:227-233` | 可见 | 288,1314 223×18 |
| S-modality-58 | 此构建中没有可用的实时操作。 | 文本 | `v-else-if="operations.length === 0"` | `ModalityRealtimePanel.vue:236-242` | 未出现（条件：走了 `!enabled`） | — |
| S-modality-59 | 此构建不支持音频帧：Desktop 没有安全的音频采集或编码器。文本帧仍可使用。 | 文本·提示条 | 常显 | `ModalityRealtimePanel.vue:245-250` | 可见 | 288,1360 443×18 |
| S-modality-60 | 操作（`value`） | 文本标签 + 下拉（`:disabled`） | 常显 | `ModalityRealtimePanel.vue:255-263` | 可见 | 272,1414 25×15（下拉 272,1431 316×32） |
| S-modality-61 | 参数（JSON 对象） | 文本标签 + 输入（`:disabled`） | 常显 | `ModalityRealtimePanel.vue:266-268` | 可见 | 598,1414 110×15（输入 598,1431 316×32） |
| S-modality-62 | 提供商（可选） | 文本标签 + 输入（`:disabled`） | 常显 | `ModalityRealtimePanel.vue:277-279` | 可见 | 272,1474 86×15（输入 272,1490 316×32） |
| S-modality-63 | 模型（可选） | 文本标签 + 输入（`:disabled`） | 常显 | `ModalityRealtimePanel.vue:286-288` | 可见 | 598,1474 74×15（输入 598,1490 316×32） |
| S-modality-64 | 打开会话（`opening ? 打开中… : 打开会话`） | 按钮（`:disabled`） | 常显 | `ModalityRealtimePanel.vue:296-306` | 可见 | 272,1532 70×32 |
| S-modality-65 | 关闭会话（`closing ? 关闭中… : 关闭会话`） | 按钮（`:disabled`） | 常显 | `ModalityRealtimePanel.vue:307-316` | 可见 | 349,1532 72×32 |
| S-modality-66 | `openError` · `closeError` | 文本 | 非空 | `ModalityRealtimePanel.vue:318-319` | 未出现（条件：无错误） | — |
| S-modality-67 | 会话 {id}（{transport}） | 文本 | `session` | `ModalityRealtimePanel.vue:322-325` | 未出现（条件：未开会话） | — |
| S-modality-68 | 会话已结束：{reason} | 文本 | 同上 + `endedReason` | `ModalityRealtimePanel.vue:326-328` | 未出现（条件） | — |
| S-modality-69 | 所有者返回了直连传输。Desktop 未实现直连提供商传输，因此无法在此发送帧。 | 文本 | 同上 + `transport === 'direct'` | `ModalityRealtimePanel.vue:329-335` | 未出现（条件） | — |
| S-modality-70 | 帧发送区（文本帧输入 + 发送） | 容器 + 控件 | 同上 + `v-else` | `ModalityRealtimePanel.vue:337+` | 未出现（条件） | — |

本节行数 70（图标 4）｜可见 48｜未出现（条件）22｜未命中 0｜运行时独有 0
