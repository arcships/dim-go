# 设置运行时清单（模板 ∪ 运行时 ∪ 截图）

本文只做元素清点，不写行为需求。

## 0. 口径

### 0.1 三源材料

| 源 | 位置 | 用途 |
|---|---|---|
| 模板 | `components/settings/*Settings.vue`、`views/SettingsView.vue`、`views/settings/account/*.vue`（相对参考产品渲染进程源码根 `packages/desktop/packages/renderer/src`） | 行清单的来源：每个元素一行，`v-if/v-else` 每个分支各成一行 |
| 运行时 | `runtime/settings/&lt;节&gt;.json`（19 份，合计 3354 节点） | 判定「可见」，提供包围盒 `x/y/w/h`、`role`、`icon`、真实文案 |
| 截图 | `runtime/settings/&lt;节&gt;-top.png`、`&lt;节&gt;-bottom.png`（各 2×，2060×1450） | 图标、边框、分隔线等 JSON 不含的视觉信息；`png 像素 = CSS 坐标 × 2` |

运行时 JSON 结构：`{section, count, items:[{chain,tag,role,text,label,icon,x,y,w,h}]}`。一次抓取含多趟：侧栏趟与内容趟交替出现，按滚动量偏移（内容区步长 686，侧栏 56，长页更多趟）。本文只取**首趟**（未滚动）的节点作为位置与节点计数，其余趟次当作同一节点的重复观测；因此「首趟内容节点数」小于 JSON 的 `count`（如账户 174 → 48 个内容节点 + 40 个侧栏节点）。

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
可见     = 主体行 + 容器行 + 图标行
运行时内容节点 = 可见主体行 + 折叠内文 + 数据驱动实例 + 运行时独有
```

「可见」中的容器行（png 边框实测）与图标行（与宿主共用包围盒）在运行时 JSON 里没有独立节点，因此 `运行时内容节点 ≥ 可见主体行`，差额按上面第三条拆分，逐节在表内注明。


---

## S-00 壳层（`views/SettingsView.vue` + `components/settings/SettingsNav.vue` + `sections.ts`）

侧栏固定宽 256（`w-64`），内容列 x 257–998，中间 1px 分隔线 x=228（png 实测）。导航 19 项与顺序取自 `sections.ts` 的 `SETTINGS_SECTION_ORDER`，不按 UI 模式过滤。

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-shell-01 | 返回 | 按钮（图标 + 文字） | 侧栏可见：`v-show="!isCompact \|\| !compactShowingDetail"` | `SettingsNav.vue:53-62` | 可见 | 15,59 202×28 |
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

本节行数 42（图标 21、容器 0）｜可见 40（主体 20 / 容器 0 / 图标 20）｜未出现（条件）2｜未命中 0｜运行时内容节点 —（壳层为共用结构，节点计入各节）

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
| S-account-36 | 套餐标识 BrandMark | 容器·图标块 | 同上 + `v-else-if="usage \|\| !error"` | `AccountPlanUsage.vue:458-464` | 可见 | 275,256 32×32 |
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


运行时对齐：48 个内容节点 = 41 个可见主体行 + 7 个折叠内文（按钮内文字、计数 span 等并入宿主行）；容器行 7、图标行 10 与宿主共用包围盒，不额外计数；无模板外节点。

本节行数 112（图标 18、容器 10）｜可见 58（主体 41 / 容器 7 / 图标 10）｜未出现（条件）54｜未命中 0｜运行时内容节点 48

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


运行时 3 个内容节点全部对上（S-usage-01/04/05）；另 2 个节点是卡片图标与容器，落在 S-usage-02/03。`AccountActivity.vue` 的 `v-else` 分支（460 行起的分支树）在本模式下整段不可达，但每一分支都有明确条件，故不计「未命中」。

本节行数 74（图标 9、容器 10）｜可见 5（主体 3 / 容器 1 / 图标 1）｜未出现（条件）69｜未命中 0｜运行时内容节点 3

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


运行时对齐：48 个内容节点 = 38 个可见主体行 + 10 个折叠内文（办公/编程模式卡的标签与描述、系统默认、App Server 模式下不可用 ×2、继承父会话、已保存的 Shell 不可用 ×2 等）+ 0 独有。

本节行数 99（图标 19、容器 0）｜可见 41（主体 38 / 容器 0 / 图标 3）｜未出现（条件）58｜未命中 0｜运行时内容节点 48

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


运行时对齐：53 个内容节点 = 43 个可见主体行 + 10 个折叠内文（主题卡文字、字体卡当前值、语言与代码主题的下拉当前值、皮肤卡内的名称 div 等）+ 0 独有。

本节行数 94（图标 8、容器 3）｜可见 50（主体 43 / 容器 1 / 图标 6）｜未出现（条件）44｜未命中 0｜运行时内容节点 53

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

本节行数 90（图标 3、容器 0）｜可见 30（主体 28 / 容器 0 / 图标 2）｜未出现（条件）60｜未命中 0｜运行时内容节点 30

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

本节行数 27（图标 0、容器 0）｜可见 20（主体 20 / 容器 0 / 图标 0）｜未出现（条件）7｜未命中 0｜运行时内容节点 20

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
| S-configuration-05 | 分组折叠头 `t(group.labelKey)` · `group.providers.length` | 按钮（`v-for="group in providerListGroups"`） | 常显（2 组：已启用 1 / 未启用 127） | `:2128-2143` | 可见（2 实例） | 已启用 234,77 209×25；未启用 234,134 209×25 |
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


运行时对齐（515 节点）：供应商列表 394 项 = 2 个分组头（S-configuration-05）、128 个供应商项（S-configuration-07）、图标节点、搜索框与「添加」；详情 121 项 = 头部 3 项、账户托管提示 2 项、基础 URL 2 项、「模型 32/32」2 项、搜索框 1 项、32 个模型行 ×（名称 + 能力徽标 + 开关）、底部 2 个按钮。列表项与模型行都是数据驱动，按模板节点各记 1 行，实例数写在上表「运行时」列。

本节行数 83（图标 3、容器 2）｜可见 20（主体 16 / 容器 1 / 图标 3）｜未出现（条件）63｜未命中 0｜运行时内容节点 515

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

本节行数 70（图标 4、容器 2）｜可见 40（主体 37 / 容器 1 / 图标 2）｜未出现（条件）30｜未命中 0｜运行时内容节点 70

---

## 9. 运行环境（`components/settings/RuntimeSettings.vue` + `FeatureRuntimeSettings.vue`）

png：[运行环境-top.png](runtime/settings/运行环境-top.png)、[运行环境-bottom.png](runtime/settings/运行环境-bottom.png)。运行时 90 节点 = 侧栏 40 + 内容 12 + 侧栏 38；内容首趟 y 33–365。当前状态：App Server 未声明 `runtime.list`，运行环境条目区显示错误分支；功能组件区显示 OfficeCLI 未安装。

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-runtime-01 | 运行环境 | 页面标题（`h1`） | 常显 | `RuntimeSettings.vue:78` | 可见 | 257,33 446×25 |
| S-runtime-02 | 由 App Server RuntimeManager 管理；Desktop 不下载、不保留第二份运行时。 | 文本 | 常显 | `RuntimeSettings.vue:79` | 可见 | 257,61 446×18 |
| S-runtime-03 | 刷新（含 `RefreshCw`，`:disabled`） | 按钮 | 常显 | `RuntimeSettings.vue:81-91` | 可见 | 934,33 63×28 |
| S-runtime-04 | 正在加载... | 文本（role=status） | `loading` | `RuntimeSettings.vue:94-96` | 未出现（条件：已加载） | — |
| S-runtime-05 | 运行环境条目区 | 容器·卡片 | `v-else`（`entries !== null`） | `RuntimeSettings.vue:98-110` | 未出现（条件：当前走错误分支） | — |
| S-runtime-06 | `error ?? 无法加载运行环境设置。` | 文本（role=status，含 `AlertCircle` 语义） | `entries === null` | `RuntimeSettings.vue:98-104` | 可见（The App Server does not advertise runtime.list…） | 272,111 711×18 |
| S-runtime-07 | 重试 | 按钮 | 同上 | `RuntimeSettings.vue:105-107` | 可见 | 272,139 48×28 |
| S-runtime-08 | 实时运行环境更新不可用；请使用刷新重新读取 owner 条目。 | 文本（role=status） | `watchAvailable === false` | `RuntimeSettings.vue:111-118` | 未出现（条件：未进入条目区） | — |
| S-runtime-09 | `RUNTIME_NAMES[name]` · `statusLabel(...)`（Node.js / Python / uv 三条） | 条目卡（`v-for="name in RUNTIMES"`） | `byName[name] !== undefined` | `RuntimeSettings.vue:120-154` | 未出现（条件：无 runtime 条目） | — |
| S-runtime-10 | ↳ 状态图标（`Loader2` / `CheckCircle2` / `AlertTriangle`） | 图标 | 按状态三支 | `RuntimeSettings.vue:138-148` | 未出现（条件） | — |
| S-runtime-11 | 系统自带 / 下载（来源切换） | 控件·分段组（`aria-label="运行环境来源"`，`:disabled`） | 同上 | `RuntimeSettings.vue:157-180` | 未出现（条件） | — |
| S-runtime-12 | 启动时获取 | 开关（`:disabled`） | 同上 | `RuntimeSettings.vue:185-193` | 未出现（条件） | — |
| S-runtime-13 | 重新安装 / 安装（`Loader2` / `Download`） | 按钮（`:disabled`） | 同上 | `RuntimeSettings.vue:195-209` | 未出现（条件） | — |
| S-runtime-14 | `version` · `path` · 复制路径 | 文本·值 + 按钮 | 同上 + `path !== undefined` | `RuntimeSettings.vue:212-238` | 未出现（条件） | — |
| S-runtime-15 | `reason` | 文本（role=alert） | 同上 + `reason` | `RuntimeSettings.vue:242-249` | 未出现（条件） | — |
| S-runtime-16 | App Server 未返回 {runtime}。 | 文本 | `v-else`（`byName[name]` 缺） | `RuntimeSettings.vue:252-254` | 未出现（条件） | — |
| S-runtime-17 | `error` | 文本（role=alert） | `error !== null` | `RuntimeSettings.vue:257-264` | 未出现（条件：无错误） | — |
| S-runtime-18 | 功能组件 | 分组标题（`h2`） | 常显 | `FeatureRuntimeSettings.vue:52` | 可见 | 257,199 741×21 |
| S-runtime-19 | `store.loadError` | 文本（role=alert） | `store.loadError` | `FeatureRuntimeSettings.vue:53-54` | 未出现（条件：加载成功） | — |
| S-runtime-20 | 重试 | 按钮 | 同上 | `FeatureRuntimeSettings.vue:55` | 未出现（条件） | — |
| S-runtime-21 | 正在加载... | 文本（role=status） | `v-else-if="!store.snapshot"` | `FeatureRuntimeSettings.vue:57` | 未出现（条件：已有快照） | — |
| S-runtime-22 | OfficeCLI | 组件标题（`h3`，`v-for="component in components"`） | `v-else` | `FeatureRuntimeSettings.vue:59-61` | 可见 | 275,253 63×21 |
| S-runtime-23 | 未安装（`featureRuntimes.status.*`） | 状态·徽标（role=status，含状态图标） | 同上 | `FeatureRuntimeSettings.vue:62-65` | 可见 | 942,254 37×18 |
| S-runtime-24 | 创建、读取和编辑 Word、Excel、PowerPoint 文件。 | 文本 | 同上 | `FeatureRuntimeSettings.vue:67` | 可见 | 275,284 704×18 |
| S-runtime-25 | `versionText(component)`（配套版本 1.0.135-desktop.1 · 下载 13.0 MB） | 文本 | 同上 | `FeatureRuntimeSettings.vue:68` | 可见 | 275,312 704×14 |
| S-runtime-26 | 下载进度 `formatBytes` + `Progress` | 文本 + 进度条 | 同上 + `downloadedBytes !== undefined && active(component)` | `FeatureRuntimeSettings.vue:69-71` | 未出现（条件：未在下载） | — |
| S-runtime-27 | `errors[component] ?? …error.message` | 文本（role=alert） | 非空 | `FeatureRuntimeSettings.vue:73` | 未出现（条件：无错误） | — |
| S-runtime-28 | 已安装并验证。新任务立即生效；已有任务需重启应用，已有终端需重新打开。 | 文本 | `status === 'ready'` | `FeatureRuntimeSettings.vue:74` | 未出现（条件：未安装） | — |
| S-runtime-29 | 下载安装 / 验证可用性 | 按钮（`:disabled`） | 非内置组件且未就绪 / 其它 | `FeatureRuntimeSettings.vue:76-77` | 可见（下载安装） | 275,337 70×28 |

本节行数 29（图标 1、容器 1）｜可见 11（主体 11 / 容器 0 / 图标 0）｜未出现（条件）18｜未命中 0｜运行时内容节点 12

---

## 10. MCP（`components/settings/McpSettingsPage.vue` + `McpImportPanel.vue`）

png：[MCP-top.png](runtime/settings/MCP-top.png)、[MCP-bottom.png](runtime/settings/MCP-bottom.png)。运行时 96 节点 = 侧栏 40 + 内容 18 + 侧栏 38；内容首趟 y 33–240。当前状态：列表视图，1 个服务器 `dim-desktop-browser`（user 范围、disconnected、0 个工具）。

模板按 `view` 分两支：`list`（876–1164）与编辑表单（1166 起）。

### 10.1 列表视图

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-mcp-01 | MCP 服务器 | 页面标题 | `view === 'list'` | `:880` | 可见 | 257,33 741×25 |
| S-mcp-02 | 连接外部工具和数据源。 | 文本 | 同上 | `:881-883` | 可见 | 257,61 741×18 |
| S-mcp-03 | 了解更多。 | 文本·链接（含 `ExternalLink`） | 同上 | `:883-890` | 可见 | 395,61 77×18 |
| S-mcp-04 | 服务器 | 分组标题 | 同上 | `:897` | 可见 | 257,113 37×18 |
| S-mcp-05 | 导入（含 `Download`） | 按钮 | 同上 + `canImport` | `:899-908` | 可见 | 819,106 69×32 |
| S-mcp-06 | 添加服务器（含 `Plus`） | 按钮 | 同上 | `:909-912` | 可见 | 894,106 103×32 |
| S-mcp-07 | 导入面板（`McpImportPanel`） | 面板 | `importOpen` | `:916` | 未出现（条件：导入面板未打开） | — |
| S-mcp-08 | 范围 | 文本标签 | 同上 | `:925` | 可见 | 257,158 25×12 |
| S-mcp-09 | 全部范围 / 用户 / 项目（范围下拉） | 控件·下拉 | 同上 | `:926-933` | 可见（当前值「全部范围」） | 292,148 140×32（值 303,155 49×18） |
| S-mcp-10 | MCP 配置问题提示（`warnings`） | 文本容器（含 `AlertTriangle`） | `warnings.length > 0` | `:938-945` | 未出现（条件：无警告） | — |
| S-mcp-11 | `warning.scope · serverId · message` | 列表项（`v-for="warning in warnings"`） | 同上 | `:947-950` | 未出现（条件） | — |
| S-mcp-12 | 加载骨架 | 容器·骨架 | `loading && !loaded` | `:954-956` | 未出现（条件：已加载） | — |
| S-mcp-13 | `mcpStore.error` + 重试 | 文本 + 按钮 | `mcpStore.error` | `:959-963` | 未出现（条件：无错误） | — |
| S-mcp-14 | 还没有配置 MCP 服务器 | 文本·空态 | `servers.length === 0` | `:966-968` | 未出现（条件：已有 1 个服务器） | — |
| S-mcp-15 | 此范围内没有服务器。 | 文本·空态 | `visibleServers.length === 0` | `:970-972` | 未出现（条件：全部范围下有 1 个） | — |
| S-mcp-16 | `server.name`（dim-desktop-browser） | 文本·服务器名（`v-for="server in visibleServers"`） | `v-else` | `:975-984` | 可见 | 272,205 501×18 |
| S-mcp-17 | `server.scope` · `SSE（已弃用）` · `serverStatusLabel` · `serverWarningText` | 文本·元信息 | 同上（SSE / warning 各有 `v-if`） | `:986-996` | 可见（user · disconnected） | 272,226 22×14、300,226 67×14 |
| S-mcp-18 | `getAuthNotice(server)` | 文本 | 同上 + 有认证提示 | `:998-1004` | 未出现（条件：无认证提示） | — |
| S-mcp-19 | {count} 个工具（0 个工具） | 文本·计数（含 `Wrench`） | 同上 | `:1006-1013` | 可见 | 783,209 35×28（数字 810,214 8×18） |
| S-mcp-20 | 认证 / 认证中 | 按钮 | `canAuthenticateServer(server)` | `:1015-1025` | 未出现（条件：该服务器无需认证） | — |
| S-mcp-21 | 取消（认证） | 按钮 | `authCancellableIds.has(server.id)` | `:1026-1035` | 未出现（条件） | — |
| S-mcp-22 | 查看连接 | 图标按钮 | `capabilities.inspect` | `:1036-1046` | 可见 | 829,207 32×32 |
| S-mcp-23 | 断开连接 | 图标按钮（`:disabled`） | `capabilities.disconnect` | `:1048-1059` | 可见 | 871,207 32×32 |
| S-mcp-24 | 配置服务器 | 图标按钮 | 常显 | `:1061-1069` | 可见 | 913,207 32×32 |
| S-mcp-25 | 启用开关 | 开关（`:disabled`） | 常显 | `:1071` | 可见 | 955,214 28×16 |
| S-mcp-26 | 来自插件 | 分组标题 | `pluginServers.length > 0` | `:1082` | 未出现（条件：无插件服务器） | — |
| S-mcp-27 | `plugin.id` · `plugin.transport` · 只读 | 列表项（`v-for="plugin in pluginServers"`） | 同上 | `:1084-1094` | 未出现（条件） | — |
| S-mcp-28 | 需要登录 / 连接成功 / 连接失败（含延迟、工具数、缺权限范围） | 测试结果文本 | 同上 + `getMcpTestResult(plugin.id)` | `:1096-1120` | 未出现（条件） | — |
| S-mcp-29 | 查看连接 / 断开连接 / 测试连接 | 图标按钮 | 同上 | `:1124-1158` | 未出现（条件） | — |

### 10.2 编辑表单视图（`view !== 'list'`，未出现）

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-mcp-30 | 连接自定义 MCP | 页面标题 | `v-else`（`view !== 'list'`） | `:1169` | 未出现（条件：当前在列表视图） | — |
| S-mcp-31 | 文档 | 文本·链接 | 同上 | `:1170-1177` | 未出现（条件） | — |
| S-mcp-32 | 正在加载... | 文本 | `loadingDraft` | `:1181-1184` | 未出现（条件） | — |
| S-mcp-33 | 已存储的值已隐藏，留空即可保留。 · 密钥值存储在用户范围，无法在此项目中修改。 | 文本 | `draft.snapshot` | `:1192-1195` | 未出现（条件） | — |
| S-mcp-34 | 移动作用域说明 + 移动 MCP 条目？对话框（取消 / 确认） | 文本 + 对话框 | `mode === 'edit'` 且 `capabilities.move` | `:1197-1231` | 未出现（条件） | — |
| S-mcp-35 | 名称 | 文本标签 + 输入 | 常显 | `:1237-1238` | 未出现（条件） | — |
| S-mcp-36 | 范围（用户 / 项目） | 文本标签 + 下拉 | 常显 | `:1248-1255` | 未出现（条件） | — |
| S-mcp-37 | STDIO / Streamable HTTP / SSE（已弃用） | 控件·分段按钮（传输类型） | 常显 | `:1261-1290` | 未出现（条件） | — |
| S-mcp-38 | 启动命令 | 文本标签 + 输入 | `transport === 'stdio'` | `:1295-1296` | 未出现（条件） | — |
| S-mcp-39 | 参数 + 替换参数（已存储 {count} 个）+ 添加参数 | 文本标签 + 输入列表 | 同上 | `:1305-1338` | 未出现（条件） | — |
| S-mcp-40 | 环境变量 + 添加环境变量 | 文本标签 + 键值输入列表 | 同上 | `:1342-1364` | 未出现（条件） | — |
| S-mcp-41 | 环境变量透传 + 添加变量 | 文本标签 + 输入列表 | 同上 + `!draft.snapshot` | `:1368-1389` | 未出现（条件） | — |
| S-mcp-42 | 工作目录 | 文本标签 + 输入 | 同上 | `:1393-1394` | 未出现（条件） | — |
| S-mcp-43 | SSE 已弃用，仅用于兼容旧版 MCP 服务器，不推荐用于新配置。请优先使用 Streamable HTTP。 | 文本（role=note） | `transport === 'sse'` | `:1404-1411` | 未出现（条件） | — |
| S-mcp-44 | URL + 替换 URL + 移除已存储值 | 文本标签 + 输入 + 勾选 | `transport !== 'stdio'` | `:1414-1429` | 未出现（条件） | — |
| S-mcp-45 | Bearer 令牌 + 移除已存储的 Bearer 令牌 | 文本标签 + 输入 + 勾选 | 同上 | `:1439-1451` | 未出现（条件） | — |
| S-mcp-46 | 请求头 + 添加请求头 | 文本标签 + 键值输入列表 | 同上 | `:1461-1483` | 未出现（条件） | — |
| S-mcp-47 | 来自环境变量的请求头 + 添加变量 | 文本标签 + 键值输入列表 | 同上 + `!draft.snapshot` | `:1487-1504` | 未出现（条件） | — |
| S-mcp-48 | `formError` · `error`（表单校验） | 文本 | 非空 | `:1510-1512` | 未出现（条件） | — |
| S-mcp-49 | 测试结果（需要登录 / 连接成功 / 失败，含延迟与工具数） | 文本 | `getMcpTestResult(selectedServer.id)` | `:1515-1534` | 未出现（条件） | — |
| S-mcp-50 | 保存 / 测试连接 / 取消等底部动作 | 按钮 | 常显 | `:1535-1659` | 未出现（条件） | — |
| S-mcp-51 | 删除服务器确认（删除服务器 / 取消 / 删除） | 对话框 | 删除动作 | `:1659` 区间内 | 未出现（条件：对话框未打开） | — |

本节行数 51（图标 0、容器 1）｜可见 15（主体 15 / 容器 0 / 图标 0）｜未出现（条件）36｜未命中 0｜运行时内容节点 18

---

## 11. 通知（`components/settings/NotificationsSettings.vue`）

png：[通知-top.png](runtime/settings/通知-top.png)、[通知-bottom.png](runtime/settings/通知-bottom.png)。运行时 85 节点 = 侧栏 40 + 内容 7 + 侧栏 38；内容首趟 y 33–255。三个开关行都是「边框卡 + 标签 + 开关」。

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-notifications-01 | 通知 | 页面标题 | 常显 | `:59` | 可见 | 257,33 741×25 |
| S-notifications-02 | 通知设置卡（3 张） | 容器·卡片 | 常显 | `:62`、`:67`、`:72` | 可见（png 边框） | 257,85 741×56 起，间距 68 |
| S-notifications-03 | 任务完成或失败时通知 | 文本标签 | 常显 | `:63` | 可见 | 272,100 123×18 |
| S-notifications-04 | 任务完成或失败时通知（开关） | 开关 | 常显 | `:64` | 可见 | 955,101 28×16 |
| S-notifications-05 | 通知声音 | 文本标签 | 常显 | `:68` | 可见 | 272,169 49×18 |
| S-notifications-06 | 通知声音（开关） | 开关 | 常显 | `:69` | 可见 | 955,169 28×16 |
| S-notifications-07 | 需要我回应时通知 | 文本标签 | 常显 | `:73` | 可见 | 272,237 98×18 |
| S-notifications-08 | 需要我回应时通知（开关） | 开关 | 常显 | `:74` | 可见 | 955,238 28×16 |


注：3 张卡片容器是同一段模板 `v-for` 外的重复 markup，按 1 行计（运行时无对应节点，位置取 png 边框）。

本节行数 8（图标 0、容器 1）｜可见 8（主体 7 / 容器 1 / 图标 0）｜未出现（条件）0｜未命中 0｜运行时内容节点 7

---

## 12. 宠物（`components/settings/PetsSettings.vue`）

png：[宠物-top.png](runtime/settings/宠物-top.png)、[宠物-bottom.png](runtime/settings/宠物-bottom.png)。运行时 94 节点 = 侧栏 40 + 内容 16 + 侧栏 38；内容首趟 y 33–588。当前状态：Pet 显示组件不可用（2 个内置 Pet 可选），无自定义 Pet。

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-pets-01 | 宠物 | 页面标题 | 常显 | `:247` | 可见 | 257,33 741×25 |
| S-pets-02 | 显示 Pet | 文本标签 | 常显 | `:253` | 可见 | 272,100 135×18 |
| S-pets-03 | Pet 会保持在其他窗口上方。 | 文本 | 常显 | `:254-256` | 可见 | 272,121 135×14 |
| S-pets-04 | 显示 Pet（开关） | 开关（`:disabled`、`aria-label`） | 常显 | `:258` | 可见 | 955,109 28×16 |
| S-pets-05 | `floatingUnavailableNoteKey`（Pet 显示组件不可用…） | 文本·提示条 | `isMainWindow` | `:267-273` | 可见 | 272,149 711×34 |
| S-pets-06 | Pet 大小 | 文本标签 + 滑块（`:disabled`） | `!isMainWindow` | `:275-283` | 未出现（条件：当前是主窗口） | — |
| S-pets-07 | `operationError.message` | 文本（role=alert） | `operationError` | `:297-304` | 未出现（条件：无错误） | — |
| S-pets-08 | 打开素材位置并备份。修复目录权限后，刷新并从备份重新安装。不要覆盖相同 ID 的 Pet。 | 文本 | 同上 + `recoveryPetId !== undefined` | `:304-305` | 未出现（条件） | — |
| S-pets-09 | 打开保留素材（含 `FolderOpen`） | 按钮（`:disabled`） | 同上 | `:306-316` | 未出现（条件） | — |
| S-pets-10 | 重试 | 按钮（`:disabled`） | `snapshot === null \|\| descriptors === null` | `:318-328` | 未出现（条件：已加载） | — |
| S-pets-11 | 选择 Pet | 分组标题 | 常显 | `:333` | 可见 | 257,226 741×21 |
| S-pets-12 | 已保存的 Pet 不可用：{id} · `selectionError` · 在已保存的 Pet 恢复可用前，将显示 {name}。 | 文本 + 预览 | `selectionMissing && snapshot` | `:344-360` | 未出现（条件：已选 Pet 可用） | — |
| S-pets-13 | 正在加载... | 文本 | `descriptors === null` | `:367-374` | 未出现（条件：已加载） | — |
| S-pets-14 | `pet.name` · `pet.description` · `pet.error` | 预览卡（`v-for="pet in descriptors"`） | `v-else` | `:377-401` | 可见（2 个内置 Pet：Dim / dim-fox） | Dim 337,295 586×18；dim-fox 337,380 572×18 |
| S-pets-15 | 选择 / 已选择 | 按钮（`:disabled`） | 同上 + `pet.available` | `:404-414` | 可见 | Dim 937,290 46×28；dim-fox 923,375 60×28 |
| S-pets-16 | 无法渲染这个 Pet 的预览。 | 文本（role=alert） | `previewUnavailable` | `:418-420` | 未出现（条件：预览可渲染） | — |
| S-pets-17 | 自定义 Pets | 分组标题 | 常显 | `:425` | 可见 | 257,460 741×21 |
| S-pets-18 | 创建 | 按钮（含 `Plus`，`:disabled`） | 常显 | `:429-432` | 可见 | 257,495 67×32 |
| S-pets-19 | 从文件夹安装 | 按钮（含 `FolderOpen`，`:disabled`） | 常显 | `:433-436` | 可见 | 330,495 116×32 |
| S-pets-20 | 刷新 | 按钮（含 `RefreshCw`，`:disabled`） | 常显 | `:437-440` | 可见 | 453,495 69×32 |
| S-pets-21 | 打开 Pets 文件夹 | 按钮（含 `FolderOpen`，`:disabled`） | 常显 | `:441-449` | 可见 | 528,495 137×32 |
| S-pets-22 | 尚未安装自定义 Pet。 | 文本·空态（虚线框） | `descriptors !== null && customPets?.length === 0` | `:452-458` | 可见 | 257,540 741×48 |
| S-pets-23 | `pet.name` · `pet.error` + 选择 / 删除 | 自定义 Pet 行（`v-for="pet in customPets"`） | `v-else-if="customPets?.length"` | `:460-491` | 未出现（条件：无自定义 Pet） | — |
| S-pets-24 | 删除 {name}？+（说明两支：已保存选择将切回内置 / 将从托管目录删除） | 对话框 | 点击删除 | `:497-513` | 未出现（条件：对话框未打开） | — |
| S-pets-25 | 取消 | 对话框按钮 | 同上 | `:516-518` | 未出现（条件） | — |
| S-pets-26 | 删除（`operation === 'delete' ? 正在删除… : 删除`） | 对话框按钮 | 同上 | `:519-526` | 未出现（条件） | — |

本节行数 26（图标 0、容器 0）｜可见 14（主体 14 / 容器 0 / 图标 0）｜未出现（条件）12｜未命中 0｜运行时内容节点 16

---

## 13. 权限（`components/settings/SandboxSettings.vue`）

png：[权限-top.png](runtime/settings/权限-top.png)、[权限-bottom.png](runtime/settings/权限-bottom.png)。运行时 100 节点 = 侧栏 40 + 内容 22 + 侧栏 38；内容首趟 y 33–580。当前状态：进程隔离区已加载（两个开关），权限规则表为空。

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-permissions-01 | 权限 | 页面标题 | 常显 | `:287` | 可见 | 257,33 741×25 |
| S-permissions-02 | ↳ `ShieldCheck` 图标 | 图标 | 常显 | `:292` | 可见（png） | 分组标题左侧 |
| S-permissions-03 | 进程隔离 | 分组标题（`h3`） | 常显 | `:294` | 可见 | 296,100 356×18 |
| S-permissions-04 | 控制命令是否在 App Server 沙箱中运行，以及沙箱命令是否可以使用网络。 | 文本 | 常显 | `:295-297` | 可见 | 296,121 356×18 |
| S-permissions-05 | 正在加载... | 文本 | `executionLoading` | `:301-303` | 未出现（条件：已加载） | — |
| S-permissions-06 | `executionError \|\| 加载进程隔离设置失败` + 刷新 | 文本 + 按钮 | `execution === undefined` | `:304-310` | 未出现（条件：已加载） | — |
| S-permissions-07 | 沙箱后端不可用：{reason} | 文本 | `executionUnavailable` | `:313-321` | 未出现（条件：沙箱后端可用） | — |
| S-permissions-08 | 在沙箱中运行命令 | 文本标签 | `v-else` | `:325-327` | 可见 | 272,153 137×12 |
| S-permissions-09 | 使用宿主沙箱后端运行命令。 | 文本 | 同上 | `:328-330` | 可见 | 272,167 137×14 |
| S-permissions-10 | 在沙箱中运行命令（开关） | 开关（`:disabled`） | 同上 | `:332` | 可见 | 955,158 28×16 |
| S-permissions-11 | 允许网络访问 | 文本标签 | 同上 | `:342-344` | 可见 | 272,195 116×12 |
| S-permissions-12 | 允许沙箱命令访问网络。 | 文本 | 同上 | `:345-347` | 可见 | 272,209 116×14 |
| S-permissions-13 | 允许网络访问（开关） | 开关（`:disabled`） | 同上 | `:349` | 可见 | 955,200 28×16 |
| S-permissions-14 | 保存隔离设置（含 `Save`） | 按钮（`:disabled`） | 同上 | `:358-366` | 可见 | 272,237 116×28 |
| S-permissions-15 | `executionFeedback` | 文本 | 同上 + 非空 | `:367-369` | 未出现（条件：未保存过） | — |
| S-permissions-16 | `executionError` | 文本 | 同上 + 非空 | `:370-372` | 未出现（条件：无错误） | — |
| S-permissions-17 | 新命令使用这些设置；正在运行的进程保持启动时的隔离边界。 | 文本 | 同上 | `:375-377` | 可见 | 272,279 711×14 |
| S-permissions-18 | 权限 | 分组标题（`h3`） | 常显 | `:383` | 可见 | 257,358 741×18 |
| S-permissions-19 | Exec command 权限 | 文本标签 | 常显 | `:384` | 可见 | 257,379 741×18 |
| S-permissions-20 | App Server 命令规则作用于 exec 和 execute 调用。 | 文本 | 常显 | `:385-387` | 可见 | 257,400 741×14 |
| S-permissions-21 | 规则 | 小标题（`h4`） | 常显 | `:391` | 可见 | 257,428 741×18 |
| S-permissions-22 | deny 优先于 allow。exact 匹配完整命令词；prefix 匹配开头的词。 | 文本 | 常显 | `:392` | 可见 | 257,452 741×14 |
| S-permissions-23 | 命令 / 匹配 / 决策 / 规则（表头） | 表格表头 | 常显 | `:395-400` | 可见 | 命令 258,474 217×28；匹配 475,474；决策 692,474；规则 916,487（sr-only） |
| S-permissions-24 | 规则行（命令输入 + 匹配 / 决策下拉 + 移除） | 表格行（`v-for="rule in execRules"`） | `execRules` 非空 | `:404-432` | 未出现（条件：无规则） | — |
| S-permissions-25 | `rule.error` | 文本（role=alert） | 同上 + `rule.error` | `:418` | 未出现（条件） | — |
| S-permissions-26 | 添加规则 | 按钮（`:disabled`） | 常显 | `:437` | 可见 | 257,510 72×28 |
| S-permissions-27 | 保存权限（含 `Save`） | 按钮（`:disabled`） | 常显 | `:441-450` | 可见 | 257,552 86×28 |
| S-permissions-28 | `permissionFeedbackMessage` | 文本 | 非空 | `:451-460` | 未出现（条件：未保存过） | — |
| S-permissions-29 | `permissionErrorMessage` | 文本 | 非空 | `:463-465` | 未出现（条件：无错误） | — |
| S-permissions-30 | 命令权限尚未保存 · 离开前保存更改，或丢弃更改。 · 取消 / 丢弃 / 保存 | 对话框 | 离开未保存时 | `:468-477` | 未出现（条件：对话框未打开） | — |

本节行数 30（图标 1、容器 0）｜可见 20（主体 19 / 容器 0 / 图标 1）｜未出现（条件）10｜未命中 0｜运行时内容节点 22

---

## 14. Git（`components/settings/GitSettings.vue`）

png：[Git-top.png](runtime/settings/Git-top.png)、[Git-bottom.png](runtime/settings/Git-bottom.png)。运行时 128 节点 = 侧栏 40 + 内容 25 + 侧栏 38 + 内容 25；内容首趟 y 33–1099。各分组之间为 1px 分隔线。

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-git-01 | Git | 页面标题 | 常显 | `:108` | 可见 | 257,33 741×25 |
| S-git-02 | 分支前缀 | 文本标签 | 常显 | `:114` | 可见 | 257,85 741×18 |
| S-git-03 | feature/（输入框） | 控件·输入 | 常显 | `:116` | 可见（占位符不入 `text`） | 257,113 280×32 |
| S-git-04 | 默认远程仓库 | 文本标签 | 常显 | `:129` | 可见 | 257,202 741×18 |
| S-git-05 | commit 和 push 的默认远程目标 | 文本 | 常显 | `:130-132` | 可见 | 257,223 741×14 |
| S-git-06 | origin（远程下拉） | 控件·下拉 | 常显 | `:134-136` | 可见 | 257,247 77×32（值 268,254 33×18） |
| S-git-07 | `option.label`（远程选项，`v-for`） | 下拉选项 | 下拉展开 | `:139-145` | 未出现（条件：下拉未展开） | — |
| S-git-08 | 默认基线分支 | 文本标签 | 常显 | `:156` | 可见 | 257,336 741×18 |
| S-git-09 | 对比/合并的基准分支 | 文本 | 常显 | `:157-159` | 可见 | 257,357 741×14 |
| S-git-10 | main（基线分支下拉） | 控件·下拉 | 常显 | `:161-163` | 可见 | 257,381 72×32（值 268,388 28×18） |
| S-git-11 | `option.label`（基线分支选项，`v-for`） | 下拉选项 | 下拉展开 | `:166-172` | 未出现（条件：下拉未展开） | — |
| S-git-12 | 始终强制推送 | 文本标签 | 常显 | `:183` | 可见 | 257,470 172×18 |
| S-git-13 | 默认使用 force push（需谨慎使用） | 文本 | 常显 | `:184-186` | 可见 | 257,489 172×14 |
| S-git-14 | 始终强制推送（开关） | 开关 | 常显 | `:188` | 可见 | 970,478 28×16 |
| S-git-15 | 提交后自动推送 | 文本标签 | 常显 | `:197` | 可见 | 257,560 86×18 |
| S-git-16 | 提交后自动推送（开关） | 开关 | 常显 | `:199` | 可见 | 970,560 28×16 |
| S-git-17 | Commit 指令补充 | 文本标签 | 常显 | `:208` | 可见 | 257,634 741×18 |
| S-git-18 | AI 生成 commit message 时的额外提示 | 文本 | 常显 | `:209-211` | 可见 | 257,655 741×14 |
| S-git-19 | 例如：使用中文，遵循 Conventional Commits 规范...（多行输入） | 控件·输入（Textarea） | 常显 | `:213` | 可见（占位符不入 `text`） | 257,680 741×100 |
| S-git-20 | PR 指令补充 | 文本标签 | 常显 | `:226` | 可见 | 257,837 741×18 |
| S-git-21 | AI 生成 PR description 时的额外提示 | 文本 | 常显 | `:227-229` | 可见 | 257,858 741×14 |
| S-git-22 | 例如：包含测试步骤，说明破坏性变更...（多行输入） | 控件·输入（Textarea） | 常显 | `:231` | 可见（占位符不入 `text`） | 257,882 741×100 |
| S-git-23 | Worktree 根目录 | 文本标签 | 常显 | `:244` | 可见 | 257,1039 741×18 |
| S-git-24 | 选择目录...（输入框） | 控件·输入 | 常显 | `:247` | 可见（占位符不入 `text`） | 257,1067 703×32 |
| S-git-25 | 选择目录（含 `FolderOpen`） | 图标按钮 | 常显 | `:253-254` | 可见 | 966,1067 32×32 |

本节行数 25（图标 0、容器 0）｜可见 23（主体 23 / 容器 0 / 图标 0）｜未出现（条件）2｜未命中 0｜运行时内容节点 25

---

## 15. Hooks（`components/settings/HooksSettings.vue` → App Server 模式下为 `HooksAppServerCatalog.vue`）

png：[Hooks-top.png](runtime/settings/Hooks-top.png)、[Hooks-bottom.png](runtime/settings/Hooks-bottom.png)。运行时 104 节点 = 侧栏 40 + 内容 26 + 侧栏 38；内容首趟 y 33–578。当前状态：`appServerMode`，走 `HooksAppServerCatalog`（来源 2 个：用户 / 项目；条目 2 组均为空）。

### 15.1 App Server 目录（当前可见分支）

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-hooks-01 | 命令钩子 | 页面标题 | `appServerMode` | `HooksAppServerCatalog.vue:102` | 可见 | 257,33 68×25 |
| S-hooks-02 | 了解更多 | 按钮（含 `ExternalLink`） | 同上 | `:105-113` | 可见 | 884,33 79×27 |
| S-hooks-03 | 刷新 | 图标按钮（含 `RefreshCw`） | 同上 | `:114-122` | 可见 | 970,33 28×28 |
| S-hooks-04 | `hooks.loadError \|\| 加载设置失败` | 文本 | `hooks.loadError` | `:127-129` | 未出现（条件：加载成功） | — |
| S-hooks-05 | 正在加载... | 文本 | `loading && catalog === null` | `:130-132` | 未出现（条件：已加载） | — |
| S-hooks-06 | 用户文档已关闭所有钩子。 | 文本·提示条 | `catalog?.disableAllHooks` | `:134-140` | 未出现（条件：未全局关闭） | — |
| S-hooks-07 | 钩子会在这些生命周期事件上运行 shell 命令。 | 文本·提示条（含 `TriangleAlert`） | 常显 | `:143-145` | 可见 | 286,96 218×14 |
| S-hooks-08 | 来源 | 小标题 | 常显 | `:149` | 可见 | 257,145 741×14 |
| S-hooks-09 | `sourceTitle(source)` · `sourceStateLabel` · 已启用 / 已禁用 · 只读 | 来源行（`v-for="source in hooks.sources"`） | 常显（2 实例：用户 / 项目） | `:150-183` | 可见 | 用户 272,184 32×32（名 314,186）、正常 345,184、已启用 387,184；项目 272,256（缺失 345,256、已禁用 387,256） |
| S-hooks-10 | ↳ 来源图标（`<component :is>`） | 图标 | 同上 | `:157` | 可见（icon:true） | 272,184 32×32 |
| S-hooks-11 | ↳ `Lock` 图标（插件来源） | 图标 | 同上 + `source.source.startsWith('plugin:')` | `:171-176` | 未出现（条件：无插件来源） | — |
| S-hooks-12 | `source.path` | 文本（等宽） | 同上 + `source.path` | `:180-182` | 可见（/Users/…/.dimcode/hooks.json） | 314,282 561×14 |
| S-hooks-13 | `source.message` | 文本 | 同上 + `source.message` | `:183` | 未出现（条件：无 message） | — |
| S-hooks-14 | 编辑（含 `Pencil`） | 按钮 | `source.source === 'user'` | `:186-195` | 可见 | 914,184 69×28 |
| S-hooks-15 | 启用项目钩子 / 禁用项目钩子 | 文本 + 开关（`:disabled`） | `source.source === 'project'` | `:196-200` | 可见 | 885,257 63×14（开关 955,256 28×16） |
| S-hooks-16 | `failureText(hooks.writeFailure)` | 文本 | `hooks.writeFailure` | `:209-215` | 未出现（条件：无写入失败） | — |
| S-hooks-17 | 条目 | 小标题 | 常显 | `:219` | 可见 | 257,339 741×14 |
| S-hooks-18 | `sourceTitle(source)` · 已启用 / 已禁用 | 条目分组头（`v-for="source in hooks.sources"`） | 常显（2 实例） | `:220-229` | 可见 | 用户 293,373 652×18（已启用 951,375）；项目 293,453（已禁用 951,455） |
| S-hooks-19 | 没有条目 | 文本·空态 | `entriesOf(source).length === 0` | `:231-233` | 可见（2 处） | 258,399 739×33；258,479 739×33 |
| S-hooks-20 | `entry.event` · `entryStateLabel` · 异步 · `entry.command` · 匹配器 / 超时 / 失败时 | 条目行（`v-for="(entry, index) in entriesOf(source)"`） | 有条目 | `:234-254` | 未出现（条件：两组均无条目） | — |
| S-hooks-21 | `failureText(hooks.saveFailure)` | 文本 | `saveFailure && draft === null` | `:260-266` | 未出现（条件：无保存失败） | — |
| S-hooks-22 | 用户修订号 {revision} · 项目修订号 {revision} | 文本 | `catalog` | `:268-270` | 可见 | 257,541 741×17；257,561 741×17 |
| S-hooks-23 | 用户钩子文档 · 编辑用户钩子文档。保存时会按本页读取的修订号替换。 | 对话框标题 + 说明 | 点击 S-hooks-14 | `:273-277` | 未出现（条件：对话框未打开） | — |
| S-hooks-24 | 文档编辑框（`Textarea`） | 控件·输入 | 同上 | `:279-280` | 未出现（条件） | — |
| S-hooks-25 | 文档必须是 JSON 对象。 · `failureText(saveFailure)` | 文本 | 同上 + 校验/保存失败 | `:288-297` | 未出现（条件） | — |
| S-hooks-26 | 取消 / 保存 | 对话框按钮 | 同上 | `:298-304` | 未出现（条件） | — |

### 15.2 非 App Server 分支（`v-else`，未出现）

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-hooks-27 | 命令钩子（本地配置页） | 页面标题 | `!appServerMode` | `HooksSettings.vue:191` | 未出现（条件：App Server 模式） | — |
| S-hooks-28 | 了解更多 · 刷新 | 按钮 + 图标按钮 | 同上 | `:194-211` | 未出现（条件） | — |
| S-hooks-29 | `loadError` · 未配置命令钩子。 | 文本 | 同上 + 错误/空态 | `:216-219` | 未出现（条件） | — |
| S-hooks-30 | 来自配置 / 来自工作区 / 来自插件 | 分组标题 + 来源按钮（`v-for` 三组） | 同上 + 各组非空 | `:222-290` | 未出现（条件） | — |
| S-hooks-31 | `source.title` · `configPath` · {on} 开 · {off} 关 | 来源行内容 | 同上 | `:236-265` | 未出现（条件） | — |
| S-hooks-32 | 钩子编辑/条目对话框 | 对话框 | 同上（点击来源行） | `:295+` | 未出现（条件） | — |

本节行数 32（图标 2、容器 0）｜可见 14（主体 13 / 容器 0 / 图标 1）｜未出现（条件）18｜未命中 0｜运行时内容节点 26

---

## 16. 插件设置（`components/settings/PluginSettings.vue` + `PluginSettingsField.vue`）

png：[插件设置-top.png](runtime/settings/插件设置-top.png)、[插件设置-bottom.png](runtime/settings/插件设置-bottom.png)。运行时 134 节点 = 侧栏 40 + 内容 28 + 侧栏 38 + 内容 28；内容首趟 y 5–761，内容列左边界 x 229（其余节 257）。当前状态：3 个插件（Model log / Prompt suggestion / Web）的设置表单，标签文案来自插件 schema（英文）。

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-pluginSettings-01 | 插件设置 | 页面标题（`h1`） | 常显 | `:234` | 可见 | 229,5 797×25 |
| S-pluginSettings-02 | 配置 App Server 插件声明的设置。 | 文本 | 常显 | `:235` | 可见 | 229,29 797×18 |
| S-pluginSettings-03 | 正在加载插件设置... | 文本 | `loading` | `:238` | 未出现（条件：已加载） | — |
| S-pluginSettings-04 | 当前 App Server 不提供插件设置。 | 文本 | `unavailable` | `:240-242` | 未出现（条件：App Server 提供插件设置） | — |
| S-pluginSettings-05 | `loadError` + 重试 | 文本 + 按钮 | `loadError` | `:244-246` | 未出现（条件：无错误） | — |
| S-pluginSettings-06 | 没有已安装插件提供设置。 | 文本·空态 | `sources.length === 0` | `:249-251` | 未出现（条件：有 3 个插件） | — |
| S-pluginSettings-07 | 用户 / 项目（保存范围切换） | 控件·分段按钮 + 说明 | `projectSources.length > 0` | `:254-271` | 未出现（条件：当前仅在用户范围） | — |
| S-pluginSettings-08 | 项目根目录（`project.cwd`） | 文本标签 + 下拉 + 重新加载 | `activeScope === 'project'` | `:274-317` | 未出现（条件：用户范围） | — |
| S-pluginSettings-09 | 设置将从 {cwd} 读取并写回；该根目录由已打开的项目会话授权。 | 文本 | 同上 | `:319-321` | 未出现（条件） | — |
| S-pluginSettings-10 | 正在加载插件设置... / `projectRootsError` / 当前没有已授权的项目根目录。请在项目中打开会话以完成授权。 | 文本 + 按钮 | 同上（三种项目根状态） | `:279-300` | 未出现（条件） | — |
| S-pluginSettings-11 | 没有已安装插件声明项目级设置。 | 文本 | `activeScope === 'project' && projectSources.length === 0` | `:325-331` | 未出现（条件） | — |
| S-pluginSettings-12 | 尚未选择已授权的项目根目录。 | 文本 | `selectedProjectRoot === null` | `:333-339` | 未出现（条件） | — |
| S-pluginSettings-13 | 插件设置卡（`v-for="state in visibleStates"`） | 容器·卡片 | `visibleStates` 非空（3 实例） | `:350` | 可见（png 边框） | 229,69 797×159 起 |
| S-pluginSettings-14 | `sourceTitle(state.source)`（Model log / Prompt suggestion / Web） | 插件标题（`h2`） | 同上 | `:359` | 可见 | Model log 244,83 59×18；Prompt suggestion 244,238 112×18；Web 244,461 27×18 |
| S-pluginSettings-15 | `sourceDescription(state.source)` | 文本 | 同上 + 非空 | `:360` | 未出现（条件：3 个插件均无描述） | — |
| S-pluginSettings-16 | `state.source.source`（model-log / prompt-suggestion / web） | 文本（等宽） | 同上 | `:361` | 可见 | 244,104 59×17；244,259 112×17；244,482 27×17 |
| S-pluginSettings-17 | 正在加载插件设置... | 文本 | `state.loading` | `:365` | 未出现（条件：已加载） | — |
| S-pluginSettings-18 | `state.loadError` + 重试 | 文本 + 按钮 | `state.loadError` | `:367-369` | 未出现（条件） | — |
| S-pluginSettings-19 | 设置已在别处更改。请重新加载后再保存。+ 重新加载 | 文本 + 按钮 | `state.conflict` | `:373-377` | 未出现（条件：无冲突） | — |
| S-pluginSettings-20 | `field.name`（schema 字段：Enabled / Max Chars / Search Service / Allow Private Hosts / Proxy / Proxy Dns） | 字段（`v-for="field in schemaProperties(...)"`） | `v-else` | `:381` | 可见（表单内 6 个字段实例） | Enabled 244,134；Max Chars 244,321；Search Service 255,524；Allow Private Hosts 244,567；Proxy 244,633；Proxy Dns 244,702 |
| S-pluginSettings-21 | 保存（`state.saving ? 正在保存... : 保存`） | 按钮（`:disabled`） | 常显 | `:394-396` | 可见（3 处） | 244,166 46×28；244,389；244,733 |
| S-pluginSettings-22 | 重新加载 | 按钮（`:disabled`） | 常显 | `:397-399` | 可见（3 处） | 296,166 72×28；296,389；296,733 |
| S-pluginSettings-23 | 已保存 | 文本 | `state.saved` | `:400-402` | 未出现（条件：本次未保存） | — |
| S-pluginSettings-24 | `state.error` | 文本 | `state.error` | `:403-405` | 未出现（条件：无错误） | — |
| S-pluginSettings-25 | `label` · `description` · 必填 `*` | 字段标签（`PluginSettingsField`） | 字段渲染 | `PluginSettingsField.vue:131-142` | 可见（同 S-pluginSettings-20） | 同上 |
| S-pluginSettings-26 | 此值包含密钥字段，当前表单无法安全编辑。 | 文本 | `withheldSchema` / `arrayHasSecrets` | `PluginSettingsField.vue:133-135`、`:172-178` | 未出现（条件：字段不含密钥） | — |
| S-pluginSettings-27 | 布尔字段开关 | 开关 | `type === 'boolean'` / `object` 未必填 | `PluginSettingsField.vue:146`、`:199+` | 可见（Enabled 244,135；Search Service 971,524；Proxy Dns 983,702） | 见上 |
| S-pluginSettings-28 | 数值/字符串字段输入 | 控件·输入 | `type` 为标量 | `PluginSettingsField.vue:199+` | 可见（Max Chars 244,343 767×32；Proxy 244,656 767×32） | 见上 |
| S-pluginSettings-29 | 数组字段项 + 移除 / 添加 | 输入列表 + 按钮 | `type === 'array' && primitiveItems` | `PluginSettingsField.vue:167-195` | 可见（Allow Private Hosts 的「添加」244,591） | 见上 |
| S-pluginSettings-30 | 嵌套对象字段（递归 `PluginSettingsField`） | 字段组 | `objectSchema !== undefined` + `objectConfigured` | `PluginSettingsField.vue:153-154` | 未出现（条件：当前 schema 无嵌套对象展开） | — |


注：字段名与控件形状由插件 schema 决定（`schemaProperties(state.source.schema)`），模板只有一个 `v-for` 字段节点；上表按运行时实际渲染的 6 个字段列出实例位置，行数仍按模板节点计。

本节行数 30（图标 0、容器 1）｜可见 12（主体 11 / 容器 1 / 图标 0）｜未出现（条件）18｜未命中 0｜运行时内容节点 28

---

## 17. 外部 Agent（`components/settings/ExternalAgentsSettings.vue`）

png：[外部Agent-top.png](runtime/settings/外部Agent-top.png)、[外部Agent-bottom.png](runtime/settings/外部Agent-bottom.png)。运行时 82 节点 = 侧栏 40 + 内容 4 + 侧栏 38；内容首趟 y 33–164。当前状态：`unavailable`（App Server 未实现外部代理）。

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-externalAgents-01 | 外部 Agent | 页面标题 | 常显 | `:115` | 可见 | 257,33 741×25 |
| S-externalAgents-02 | 将外部 agent CLI 作为子 agent 启用，主 agent 可委派任务给它们。 | 文本 | 常显 | `:116-118` | 可见 | 257,61 741×18 |
| S-externalAgents-03 | 正在探测外部 Agent… | 文本 | `loading` | `:121-123` | 未出现（条件：探测结束） | — |
| S-externalAgents-04 | `loadError` + 重试 | 文本 + 按钮 | `loadError` | `:125-129` | 未出现（条件：无错误） | — |
| S-externalAgents-05 | 不可用提示卡 | 容器·卡片 | `unavailable` | `:132-140` | 可见（png 边框） | 257,96 741×86 |
| S-externalAgents-06 | App Server 模式下不可用 | 文本 | 同上 | `:137` | 可见 | 272,121 711×18 |
| S-externalAgents-07 | App Server 尚未实现外部代理。此模式下 Desktop 无法检测或配置它们。 | 文本 | 同上 | `:138-140` | 可见 | 272,146 711×18 |
| S-externalAgents-08 | `agent.agentType` 行 | 行容器（`v-for="agent in agents"`） | `v-else`（`!unavailable`） | `:143-154` | 未出现（条件：当前 unavailable） | — |
| S-externalAgents-09 | `agent.version` / 已安装 / 未识别版本 / 未安装 | 文本·状态 | 同上（四支互斥） | `:156-176` | 未出现（条件） | — |
| S-externalAgents-10 | ↳ `TriangleAlert` 图标 + 未能从该文件识别版本号，这个 Agent 可能不可用。 | 图标 + 提示文本 | 同上 + `execPath` 存在但版本未知 | `:161-172` | 未出现（条件） | — |
| S-externalAgents-11 | `agent.execPath` | 文本 · | 同上 + `execPath` | `:176-187` | 未出现（条件） | — |
| S-externalAgents-12 | `rowErrors[agent.agentType]` | 文本（role=alert） | 同上 + 行错误 | `:191-193` | 未出现（条件） | — |
| S-externalAgents-13 | 编辑 / 选择 {agentType} 的可执行文件 | 图标按钮（`Pencil` / `Plus`） | 同上 | `:196-219` | 未出现（条件） | — |
| S-externalAgents-14 | 启用开关 | 开关（`:disabled`） | 同上 | `:221` | 未出现（条件） | — |
| S-externalAgents-15 | 更改将在新对话中生效。 | 文本 | `!unavailable` | `:233-235` | 未出现（条件：当前 unavailable） | — |

本节行数 15（图标 1、容器 1）｜可见 5（主体 4 / 容器 1 / 图标 0）｜未出现（条件）10｜未命中 0｜运行时内容节点 4

---

## 18. 归档会话（`components/settings/ArchivedThreadsSettings.vue`）

png：[归档会话-top.png](runtime/settings/归档会话-top.png)、[归档会话-bottom.png](runtime/settings/归档会话-bottom.png)。运行时 81 节点 = 侧栏 40 + 内容 3 + 侧栏 38；内容首趟 y 33–230。当前状态：无归档会话（空态）。

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-archived-01 | 归档会话 | 页面标题 | 常显 | `:262` | 可见 | 257,33 68×25 |
| S-archived-02 | 全部删除（含 `Trash2`） | 按钮（`:disabled`） | 常显 | `:264-274` | 可见 | 910,33 88×28 |
| S-archived-03 | 删除所有归档会话 · 确定要永久删除 {count} 条归档会话吗？此操作不可撤销。 | 对话框标题 + 说明 | 点击 S-archived-02 | `:276-281` | 未出现（条件：对话框未打开） | — |
| S-archived-04 | 取消 / 删除 | 对话框按钮 | 同上 | `:283-287` | 未出现（条件） | — |
| S-archived-05 | 加载骨架（`v-for="i in 3"`） | 容器·骨架 | `loading` | `:294-307` | 未出现（条件：已加载） | — |
| S-archived-06 | 搜索归档会话 | 控件·输入（含 `Search`） | `threads.length > 0` | `:318-320` | 未出现（条件：无归档会话） | — |
| S-archived-07 | 全部项目（项目筛选下拉） | 控件·下拉 | 同上 | `:326-332` | 未出现（条件） | — |
| S-archived-08 | 全部项目 / 无项目 / `project` | 下拉选项 | 同上 + 下拉展开 | `:335-347` | 未出现（条件） | — |
| S-archived-09 | 项目分组头 `group.label` · {count} 条会话 | 文本（`v-for="group in projectGroups"`） | 同上 | `:354-364` | 未出现（条件） | — |
| S-archived-10 | 更多操作（`MoreHorizontal`） | 图标按钮 | 同上 | `:365-374` | 未出现（条件） | — |
| S-archived-11 | 删除项目内全部会话 · 删除项目内所有会话 · 确定要永久删除 "{project}" 中的 {count} 条归档会话吗？… · 取消 / 删除 | 菜单项 + 对话框 | 同上 + 菜单展开 | `:377-399` | 未出现（条件） | — |
| S-archived-12 | `thread.title` · 已关闭 · `formatDate(archivedAt) 归档` | 会话行（`v-for="(thread, index) in group.threads"`） | 同上 | `:409-430` | 未出现（条件：无归档会话） | — |
| S-archived-13 | 恢复（含 `RotateCcw`） | 按钮（`:disabled`） | 同上 | `:433-444` | 未出现（条件） | — |
| S-archived-14 | 删除归档 · 确定要永久删除 "{title}" 吗？此操作不可撤销。 · 取消 / 删除 | 按钮 + 对话框 | 同上 | `:446-468` | 未出现（条件） | — |
| S-archived-15 | ↳ `Archive` 图标 | 图标 | `filteredThreads.length === 0` | `:482-483` | 可见（png） | 585,209 区域 |
| S-archived-16 | 暂无归档会话 | 文本·空态 | `threads.length === 0` | `:485-486` | 可见 | 585,209 84×21 |
| S-archived-17 | 没有匹配的会话 · 请尝试其他搜索词或项目筛选条件 | 文本·空态 | `v-else`（有会话但筛选为空） | `:488-490` | 未出现（条件：无归档会话） | — |

本节行数 17（图标 1、容器 1）｜可见 4（主体 3 / 容器 0 / 图标 1）｜未出现（条件）13｜未命中 0｜运行时内容节点 3

---

## 19. 关于（`components/settings/AboutSettings.vue`）

png：[关于-top.png](runtime/settings/关于-top.png)、[关于-bottom.png](runtime/settings/关于-bottom.png)。运行时 184 节点 = 侧栏 40 + 内容 53 + 侧栏 38 + 内容 53；内容首趟 y 33–1072。本页没有页面标题（模板首节即「软件更新」），当前状态：自动更新已禁用、2 条公告、8 条 WIP 能力、开发态调试条。

### 19.1 软件更新

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-about-01 | 软件更新 | 文本标签 | 常显 | `:130` | 可见 | 257,33 741×18 |
| S-about-02 | 当前版本: · `currentVersion`（v0.9.57） | 文本 + 文本·值 | 常显 | `:134-136` | 可见 | 257,64 100×18；312,65 44×15 |
| S-about-03 | 新版本: · `updateVersion` | 文本 + 值 | `updateVersion` | `:138-140` | 未出现（条件：无可用版本） | — |
| S-about-04 | 已是最新版本（含 `Check`） | 文本·状态 | `phase === 'idle'` | `:146-148` | 未出现（条件：`phase` 为其它分支） | — |
| S-about-05 | 检查中...（含 `Loader2`） | 文本·状态 | `phase === 'checking'` | `:150-152` | 未出现（条件） | — |
| S-about-06 | 下载中...（含 `Download`） | 文本·状态 | `phase === 'downloading'` | `:154-156` | 未出现（条件） | — |
| S-about-07 | `updateState.message \|\| 有新版本可用`（含 `Download`） | 文本·状态 | `phase === 'available'` | `:158-164` | 未出现（条件） | — |
| S-about-08 | 更新已下载（含 `Check`） | 文本·状态 | `phase === 'downloaded'` | `:166-171` | 未出现（条件） | — |
| S-about-09 | 正在准备…（含 `Loader2`） | 文本·状态 | `phase === 'preparing'` | `:173-175` | 未出现（条件） | — |
| S-about-10 | `updateState.message \|\| 检查更新失败` | 文本·状态 | `phase === 'error'` | `:177-179` | 未出现（条件） | — |
| S-about-11 | 自动更新已禁用 | 文本·状态 | `v-else` | `:180-182` | 可见 | 257,96 86×18 |
| S-about-12 | 检查更新 / 重试 | 按钮 | `phase === 'idle' \|\| 'error'` | `:185-194` | 未出现（条件：不支持自动更新） | — |
| S-about-13 | 检查中... | 按钮（`:disabled`） | `phase === 'checking'` | `:195-204` | 未出现（条件） | — |
| S-about-14 | `downloadProgressPercent%` | 按钮（`:disabled`） | `phase === 'downloading'` | `:205-214` | 未出现（条件） | — |
| S-about-15 | 下载新版本 / `updateState.message` | 按钮 | `phase === 'available'` | `:215-228` | 未出现（条件） | — |
| S-about-16 | 重启安装 | 按钮 | `phase === 'downloaded'` | `:229-237` | 未出现（条件） | — |
| S-about-17 | 正在准备… | 按钮（`:disabled`） | `phase === 'preparing'` | `:238-246` | 未出现（条件） | — |
| S-about-18 | `updateStatusError` | 文本 | 非空 | `:249-251` | 未出现（条件：无错误） | — |

### 19.2 公告

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-about-19 | 公告 | 文本标签 | 常显 | `:257` | 可见 | 257,170 25×18 |
| S-about-20 | 刷新 | 图标按钮（含 `RefreshCw`） | 常显 | `:259-270` | 可见 | 970,170 28×28 |
| S-about-21 | 公告条目（`v-for="announcement in recentAnnouncements"`） | 控件·折叠按钮（`article > button`） | `recentAnnouncements.length > 0` | `:274-288` | 可见（2 条） | 258,213 739×56；258,270 739×56 |
| S-about-22 | `announcement.title` | 文本 | 同上 | `:290` | 可见 | DimAgent Desktop 0.9.65 / CLI 0.5.28（272,224）；0.9.26更新失败解决方案（272,281） |
| S-about-23 | `formatAnnouncementDate(...)` | 文本·时间（`time`） | 同上 | `:291-293` | 可见 | 875,226 83×14；877,283 81×14 |
| S-about-24 | `announcement.summary` | 文本 | 同上 + `summary` | `:295-297` | 可见（2 条） | 272,245 687×14；272,302 687×14 |
| S-about-25 | ↳ `ChevronUp` / `ChevronDown` 图标 | 图标 | 展开 / 收起 | `:299-304` | 可见（收起态，`ChevronDown`） | 条目内 |
| S-about-26 | 公告正文（`MarkdownRender`） | 容器·正文 | `expandedAnnouncementId === announcement.id` | `:307-315` | 未出现（条件：未展开） | — |
| S-about-27 | 暂无公告 | 文本·空态 | `announcementsLoaded` 且为空 | `:332-337` | 未出现（条件：有 2 条公告） | — |
| S-about-28 | 查看更多公告（含 `ExternalLink`） | 按钮 | 常显 | `:339-343` | 可见 | 880,341 118×28 |

### 19.3 暂不可用（WIP 能力）

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-about-29 | 暂不可用 | 文本标签 | 常显 | `:349` | 可见 | 257,397 741×18 |
| S-about-30 | 以下能力仍在 WIP 跟踪中；入口保持可见但禁用，直到 App Server 路径完成。 | 文本 | 常显 | `:350-352` | 可见 | 257,418 741×14 |
| S-about-31 | WIP 能力条目（`v-for="capability in wipCapabilities"`） | 列表项（`li`） | 常显（8 实例） | `:354-355` | 可见（8 行） | 257,443 741×58 起，行距 65 |
| S-about-32 | `capability.label` · `capability.reason` | 文本 | 同上 | `:361-363` | 可见（8 组） | 如 文件历史与回退 268,454 253×18 + 268,475 |
| S-about-33 | WIP | 状态·徽标 | 同上 | `:365` | 可见（8 个） | 950,454 36×20 等 |

### 19.4 开发态调试（`isDev`）

| 元素 ID | 名称 | 类型 | 条件 | 模板 | 运行时 | 位置 |
|---|---|---|---|---|---|---|
| S-about-34 | ↳ `Bug` 图标 | 图标 | `isDev` | `:370-376` | 可见（png） | 272,990 附近 |
| S-about-35 | Update state debugger | 文本标签 | 同上 | `:378` | 可见 | 293,999 408×18 |
| S-about-36 | Development only. Cycles local update UI states without downloading or installing. | 文本 | 同上 | `:379-381` | 可见 | 293,1020 408×14 |
| S-about-37 | `debugStateLabel`（disabled） | 文本·值（code） | 同上 | `:385` | 可见 | 272,1051 50×14 |
| S-about-38 | Next state | 按钮 | 同上 | `:386-388` | 可见 | 900,1044 83×28 |

本节行数 38（图标 2、容器 1）｜可见 21（主体 19 / 容器 0 / 图标 2）｜未出现（条件）17｜未命中 0｜运行时内容节点 53
---

## 20. 汇总

### 20.1 19 节元素数

| # | 节 | 模板行数 | 可见 | 未出现（条件） | 未命中 | 运行时内容节点 |
|---|---|---|---|---|---|---|
| — | 壳层 S-00 | 42 | 40 | 2 | 0 | （计入各节） |
| 1 | 账户 | 112 | 58 | 54 | 0 | 48 |
| 2 | 用量 | 74 | 5 | 69 | 0 | 3 |
| 3 | 个性化 | 99 | 41 | 58 | 0 | 48 |
| 4 | 外观 | 94 | 50 | 44 | 0 | 53 |
| 5 | 通用 | 90 | 30 | 60 | 0 | 30 |
| 6 | 快捷键 | 27 | 20 | 7 | 0 | 20 |
| 7 | 模型供应 | 83 | 20 | 63 | 0 | 515 |
| 8 | 多模态 | 70 | 40 | 30 | 0 | 70 |
| 9 | 运行环境 | 29 | 11 | 18 | 0 | 12 |
| 10 | MCP | 51 | 15 | 36 | 0 | 18 |
| 11 | 通知 | 8 | 8 | 0 | 0 | 7 |
| 12 | 宠物 | 26 | 14 | 12 | 0 | 16 |
| 13 | 权限 | 30 | 20 | 10 | 0 | 22 |
| 14 | Git | 25 | 23 | 2 | 0 | 25 |
| 15 | Hooks | 32 | 14 | 18 | 0 | 26 |
| 16 | 插件设置 | 30 | 12 | 18 | 0 | 28 |
| 17 | 外部 Agent | 15 | 5 | 10 | 0 | 4 |
| 18 | 归档会话 | 17 | 4 | 13 | 0 | 3 |
| 19 | 关于 | 38 | 21 | 17 | 0 | 53 |
| **合计** | 19 节 | **950** | **411** | **539** | **0** | **1001** |
| 合计 | + 壳层 | 992 | 451 | 541 | 0 | — |

### 20.2 四个总账

| 口径 | 数 | 说明 |
|---|---|---|
| 模板节点总数（行） | **992** | 19 节 950 + 壳层 42；壳层的 19 个导航项按实例各计 1 行 |
| 未命中数 | **0** | 模板有、运行时无、且无分支条件可解释的行：不存在 |
| 死代码数 | **0** | 同上；本次没有出现「无条件标记却未渲染」的行（判断标准见 0.2） |
| 运行时独有数 | **0** | 运行时的 1001 个内容节点都能追到模板行（含数据驱动列表与折叠内文） |
| 图标行 | 93 | 模板里的独立图标组件，逐行列出 |
| 容器行 | 34 | 带边框的区块，位置取 png 边框实测 |
| 运行时节点（全量） | 3354 | 含各节重复抓取趟次；首趟内容节点 1001、首趟侧栏节点 760（19×40） |
| 死代码复核 | 3 处候选，均判定为「未出现（条件）」 | `AccountActivity.vue` 的 `v-else` 分支（`appServerRead` 为真时不渲染）、`HooksSettings.vue` 的本地 sources 分支（`appServerMode` 为真时不渲染）、`AccountSignedOut.vue` 整段（`isSignedIn` 为真时不渲染）——三者都有明确开关，不是不可达代码 |

### 20.3 运行时节点与模板行的对应关系

`1001（首趟内容节点）` 与 `411（19 节可见行中扣掉容器行、图标行后的主体行）` 的差额 590，全部落在下面三类，没有一类是模板外的新元素：

1. **同一元素内的副节点**：按钮里的文字 span、下拉里的当前值、`dt`/`dd` 值、徽标里的图标宿主 —— 已在 0.3「折叠」规则里并入宿主行。
2. **数据驱动实例**：`v-for` 按数据渲染的重复行（供应商列表 128 项、模型 32 行、皮肤 6 张、WIP 能力 8 条、公告 2 条、快捷键 8 条等），模板只有 1 个节点、运行时多实例；实例数写在各节表格的「运行时」列。
3. **图标宿主重复计数**：图标行与宿主行共用同一个包围盒，位置列写「同上」。

反方向的差额（`可见行 > 运行时节点`，如 宠物 14 vs 16、通知 8 vs 7、Git 23 vs 25）来自容器行与折叠行，逐节已注明。

### 20.4 观察到的结构性事实（只陈述，不作需求）

- **同一壳层重复 19 次**：每节的运行时 JSON 都含完整侧栏（40 节点：1 个返回 + 19 个导航项 + 各自图标），19 节合计 760 个侧栏节点，是抓取口径造成的重复，不是 19 份 UI。
- **页面标题层级不统一**：`账户` / `用量` / `个性化` / `外观` / `通用` / `快捷键` / `通知` / `宠物` / `权限` / `Git` / `归档会话` / `MCP` 用 `h2`；`运行环境` / `插件设置` / `多模态`（`h1`）尺寸不同；`关于` 没有页面标题，首节直接是「软件更新」。
- **内容列左边界不统一**：`插件设置` 的内容列从 x=229 起（其余 x=257），`模型供应` 是双栏（列表 x 234–443、详情 x 466–980）。
- **App Server 模式压掉的分支**：`用量`（除空态卡外全部）、`运行环境`（runtime 条目区）、`外部 Agent`（agents 列表）、`助手模型/提示词语言`（本仓库设置项）、`Hooks`（本地 sources 目录）在本模式下都不可达，占了 539 条「未出现（条件）」中的大部分。
- **截图口径**：每节两张 png 覆盖首屏与滚到底；`模型供应`、`关于`、`个性化`、`外观`、`多模态`、`通用`、`Git` 的内容高于一屏，跨屏行的位置以 JSON 文档坐标为准（`y` 远超 725）。
