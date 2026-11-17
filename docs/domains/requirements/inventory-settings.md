# 设置界面元素清单（参考产品源码穷举）

本文只做元素穷举，不写行为需求。依据只有参考产品源码，未读本仓库任何领域文档。

源码根（相对）：`dim-electron/packages/desktop/packages/renderer/src`。
下文「定位」列的文件名均在该根下（例如 `AccountSettings.vue` 指 `views/settings/account/AccountSettings.vue`，`SkinSettings.vue` 指 `components/settings/SkinSettings.vue`）。

## 口径与约定

1. **元素**：出现在设置界面里、承载界面文案或可交互的单元——分组标题、行（标签+取值算一行）、开关、下拉、输入框、按钮、标签、说明文案、空态、链接、状态徽标、进度条、头像/图像。纯装饰分隔线（`h-px bg-border`）、纯布局容器（无文案的 `div`/`section`）不计行。
2. **名称**取界面文案原文（简体中文，来源 `i18n/zh-CN.ts`）。原文含插值的保留插值形如 `{count}`；有分支文案的用 `/` 并列全部原文。
3. **证据列**给 i18n 键（`i18n:<key>`）；无 i18n、直接写在模板里的字面量标 `字面量`；由 store/后端给出的运行期文案标 `运行期`。
4. **下拉/选择器的选项**：选项是同一个控件的取值，不是独立元素，因此合并为一行，但**全部选项文案逐项写在「名称」列中**，不省略、不抽样。选项本身是独立可见按钮/卡片的（例如主题三选一、界面模式两张卡、字号档位、皮肤卡），一项一行。
5. **条件**写「什么时候出现」（未登录 / 已登录 / 有会话 / 运行中 / 开关打开 / 平台 …）。
6. 每节末尾给「本节元素数」，随后一行「本轮自检」列出已核对的状态/菜单/空态。

以上六条与「证据」列（第 7 列）已由组织方逐条确认，全文件按此口径。

来源入口：`components/settings/sections.ts`（19 节定义与导航顺序）。

---

## S-00 设置壳层与导航（非 19 节之一，`views/SettingsView.vue` + `components/settings/SettingsNav.vue`）

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-shell-01 | 返回 | 按钮（图标 + 文字，侧栏顶部） | — | 侧栏可见（非窄屏详情态） | `SettingsNav.vue:47-60` | i18n:settingsNav.back |
| S-shell-02 | 账户 | 导航按钮（图标 + 文字） | 当前节高亮 | 常显（第 1 项） | `sections.ts:57` · `SettingsNav.vue:66-84` | i18n:settingsNav.account |
| S-shell-03 | 用量 | 导航按钮 | — | 常显（第 2 项） | `sections.ts:58` · `SettingsNav.vue:66-84` | i18n:settingsNav.usage |
| S-shell-04 | 个性化 | 导航按钮 | **默认落地节** | 常显（第 3 项） | `sections.ts:59` · `sections.ts:106` | i18n:settingsNav.personalization |
| S-shell-05 | 外观 | 导航按钮 | — | 常显（第 4 项） | `sections.ts:60` · `SettingsNav.vue:66-84` | i18n:appearanceSettings.title |
| S-shell-06 | 通用 | 导航按钮 | — | 常显（第 5 项） | `sections.ts:61` · `SettingsNav.vue:66-84` | i18n:generalSettings.title |
| S-shell-07 | 快捷键 | 导航按钮 | — | 常显（第 6 项） | `sections.ts:62` · `SettingsNav.vue:66-84` | i18n:settingsNav.keybindings |
| S-shell-08 | 模型供应 | 导航按钮 | — | 常显（第 7 项） | `sections.ts:63` · `SettingsNav.vue:66-84` | i18n:settingsNav.providers |
| S-shell-09 | 多模态 | 导航按钮 | — | 常显（第 8 项） | `sections.ts:64` · `SettingsNav.vue:66-84` | i18n:configurationSettings.modality |
| S-shell-10 | 运行环境 | 导航按钮 | — | 常显（第 9 项） | `sections.ts:65` · `SettingsNav.vue:66-84` | i18n:runtimeSettings.title |
| S-shell-11 | MCP | 导航按钮 | — | 常显（第 10 项） | `sections.ts:66` · `SettingsNav.vue:66-84` | i18n:toolsSettings.mcp |
| S-shell-12 | 通知 | 导航按钮 | — | 常显（第 11 项） | `sections.ts:67` · `SettingsNav.vue:66-84` | i18n:notificationsSettings.title |
| S-shell-13 | 宠物 | 导航按钮 | — | 常显（第 12 项） | `sections.ts:68` · `SettingsNav.vue:66-84` | i18n:petsSettings.title |
| S-shell-14 | 权限 | 导航按钮 | — | 常显（第 13 项） | `sections.ts:69` · `SettingsNav.vue:66-84` | i18n:permissionsSettings.title |
| S-shell-15 | Git | 导航按钮 | — | 常显（第 14 项） | `sections.ts:70` · `SettingsNav.vue:66-84` | i18n:gitSettings.title |
| S-shell-16 | Hooks | 导航按钮 | — | 常显（第 15 项） | `sections.ts:71` · `SettingsNav.vue:66-84` | i18n:settingsNav.hooks |
| S-shell-17 | 插件设置 | 导航按钮 | — | 常显（第 16 项） | `sections.ts:72` · `SettingsNav.vue:66-84` | i18n:settingsNav.pluginSettings |
| S-shell-18 | 外部 Agent | 导航按钮 | — | 常显（第 17 项） | `sections.ts:73` · `SettingsNav.vue:66-84` | i18n:settingsNav.externalAgents |
| S-shell-19 | 归档会话 | 导航按钮 | — | 常显（第 18 项） | `sections.ts:74` · `SettingsNav.vue:66-84` | i18n:settingsNav.archivedThreads |
| S-shell-20 | 关于 | 导航按钮 | — | 常显（第 19 项） | `sections.ts:75` · `SettingsNav.vue:66-84` | i18n:settingsNav.about |
| S-shell-21 | 返回 | 图标按钮（窄屏工具栏） | — | 窄屏（compact）且已进入某一节详情 | `SettingsView.vue:183-192` | i18n:common.back |

本节元素数：21

本轮自检：19 项导航顺序与 labelKey、当前节高亮、默认落地节（个性化）、侧栏「返回」与窄屏工具栏「返回」两个入口、窄屏侧栏/详情互斥。

---

## S-account 账户（`views/settings/account/*`）

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-account-01 | 账户 | 页面标题（h2） | — | 常显 | `AccountSettings.vue:39` | i18n:accountSettings.title |
| S-account-02 | 套餐已到期，Dim 托管模型暂不可用 | 提示横幅标题（role=status） | — | 已登录且套餐已过期 | `AccountSettings.vue:42-53` | i18n:usageSettings.planExpiredBannerTitle |
| S-account-03 | Dim 托管模型权益已停止，订阅套餐后即可恢复使用。 | 说明文案 | — | 同上（横幅内） | `AccountSettings.vue:54-56` | i18n:usageSettings.planExpiredDescription |
| S-account-04 | 选购套餐 | 按钮（sm，购物车图标） | — | 同上（横幅内） | `AccountSettings.vue:58-61` | i18n:usageSettings.planChoose |
| S-account-05 | 首字母头像（显示名/邮箱/`D` 回退） | 图像（圆形，48px） | 账号首字母；无则 `D` | 已登录 | `AccountSignedIn.vue:58-62` | i18n:accountSettings.fallbackInitial |
| S-account-06 | 显示名 | 文本（truncate，semibold） | 显示名 → 邮箱 → `Dim 账户` | 已登录 | `AccountSignedIn.vue:64-66` | i18n:accountSettings.fallbackName |
| S-account-07 | 邮箱 | 文本（次要色，truncate） | — | 已登录且账号带邮箱 | `AccountSignedIn.vue:67-69` | 运行期 |
| S-account-08 | UID | 标签（11px） | — | 已登录且账号带 uid | `AccountSignedIn.vue:70-73` | i18n:accountSettings.uidLabel |
| S-account-09 | UID 值 | 代码块（等宽，truncate，可选中） | — | 同上 | `AccountSignedIn.vue:74-77` | 运行期 |
| S-account-10 | 复制 UID | 图标按钮（title/aria-label） | — | 同上 | `AccountSignedIn.vue:78-87` | i18n:accountSettings.copyUid |
| S-account-11 | 复制成功提示「已复制」/ 失败提示「复制失败」 | toast | — | 点击复制 UID 后 | `AccountSignedIn.vue:35` · `:41` | i18n:common.copied · i18n:accountSettings.uidCopyFailed |
| S-account-12 | 查看套餐 | 按钮（outline，外链图标） | — | 已登录 | `AccountSignedIn.vue:97-104` | i18n:accountSettings.viewPlan |
| S-account-13 | {count} 个模型 | 按钮内后缀文本 | — | 已登录且已解锁模型数 > 0 | `AccountSignedIn.vue:105-110` | i18n:accountSettings.viewPlanModelCount |
| S-account-14 | 退出登录 | 按钮（ghost） | — | 已登录 | `AccountSignedIn.vue:112-120` | i18n:accountSettings.signOut |
| S-account-15 | 账号错误信息 | 错误文案（destructive） | — | 账号 store 有 error | `AccountSignedIn.vue:124-126` | 运行期 |
| S-account-16 | 登录 Dim | 卡片标题（h3） | — | 未登录 | `AccountSignedOut.vue:29` | i18n:accountSettings.signInTitle |
| S-account-17 | 登录后可解锁 Dim 托管模型。未登录时仍可在本地使用 DimAgent。 | 说明文案 | — | 未登录 | `AccountSignedOut.vue:30-32` | i18n:accountSettings.signInDescription |
| S-account-18 | 登录 / 正在登录… | 主按钮（登录中禁用 + 转圈） | 登录 | 未登录 | `AccountSignedOut.vue:36-40` | i18n:accountSettings.login · i18n:accountSettings.signingIn |
| S-account-19 | 登录企业账号 | 按钮（outline） | — | 未登录 | `AccountSignedOut.vue:41-48` | i18n:accountSettings.enterpriseLogin |
| S-account-20 | 账号错误信息 | 错误文案（destructive） | — | 未登录且账号 store 有 error | `AccountSignedOut.vue:49-51` | 运行期 |
| S-account-21 | 登录企业账号 | 弹窗标题 | — | 点击「登录企业账号」后 | `AccountEmailLoginDialog.vue:64` | i18n:accountSettings.enterpriseLogin |
| S-account-22 | 请输入您的企业邮箱账号 | 弹窗描述 | — | 同上 | `AccountEmailLoginDialog.vue:65-67` | i18n:accountSettings.enterpriseLoginDescription |
| S-account-23 | 企业邮箱 | 输入框标签 | — | 同上 | `AccountEmailLoginDialog.vue:72` | i18n:accountSettings.enterpriseEmailLabel |
| S-account-24 | name@company.com | 邮箱输入框（type=email，占位） | 空 | 同上 | `AccountEmailLoginDialog.vue:74-82` | 字面量（placeholder） |
| S-account-25 | 请输入有效的企业邮箱。 | 校验/错误文案 | — | 邮箱为空或不含合法域名，或登录失败 | `AccountEmailLoginDialog.vue:84-86` | i18n:accountSettings.invalidEnterpriseEmail |
| S-account-26 | 登录企业账号 / 正在登录… | 提交按钮（登录中转圈禁用） | 登录企业账号 | 同上 | `AccountEmailLoginDialog.vue:88-94` | i18n:accountSettings.enterpriseLogin · i18n:accountSettings.signingIn |
| S-account-27 | 套餐用量 | 卡片标题（h3） | — | 已登录 | `AccountPlanUsage.vue:427-429` | i18n:usageSettings.planTitle |
| S-account-28 | 刷新 | 图标按钮（aria-label，刷新中转圈） | — | 已登录 | `AccountPlanUsage.vue:430-440` | i18n:usageSettings.refresh |
| S-account-29 | 用量错误信息 | 错误文案（destructive） | — | 用量接口出错 | `AccountPlanUsage.vue:443-445` | 运行期 |
| S-account-30 | 正在加载用量… | 加载文案 | — | 首次加载且尚无用量缓存 | `AccountPlanUsage.vue:447-449` | i18n:usageSettings.creditLoading |
| S-account-31 | DimAgent | 产品名文本（配 BrandMark） | — | 已加载（无错误） | `AccountPlanUsage.vue:466` | 字面量 |
| S-account-32 | 套餐名 · 价格（例：`Pro · ¥xx / 月`） | 副标题文本 | — | 有套餐名或价格 | `AccountPlanUsage.vue:467-469` | 运行期（`planSubtitle`）+ i18n:usageSettings.planPrice |
| S-account-33 | 生效中 / 试用中 / 将到期 / 已取消 / 已过期 / 已逾期 | 状态徽标（pill，过期带警示图标） | — | 服务端返回套餐状态 | `AccountPlanUsage.vue:470-479` | i18n:usageSettings.planStatus* |
| S-account-34 | 到期 {date} · 还剩 {days} 天 / 已于 {date} 到期 | 文本（右对齐） | — | 已知套餐到期日 | `AccountPlanUsage.vue:480-490` | i18n:usageSettings.planExpiresOn · planDaysLeft · planExpiredOn |
| S-account-35 | 额度用量 | 当前分段标签（灰底条内） | 选中 | 有重置额度信息且已知订阅 id | `AccountPlanUsage.vue:503-505` | i18n:usageSettings.creditLabel |
| S-account-36 | 清空窗口用量 + 剩余次数 | 重置 pill（旋转图标 + 次数徽标） | — | 套餐含滚动窗口（`windowStates` 非空） | `AccountPlanUsage.vue:506-536` · 文案 `:281` | i18n:usageSettings.usageResetWindowLabel |
| S-account-37 | 重置当月用量 + 剩余次数 | 重置 pill | — | 有重置额度信息 | `AccountPlanUsage.vue:506-536` · 文案 `:293` | i18n:usageSettings.usageResetMonthlyLabel |
| S-account-38 | 清空窗口用量？/ 重置当月用量？ | 确认弹窗标题 | — | 点击对应 pill | `AccountPlanUsage.vue:540` | i18n:usageSettings.usageResetConfirmWindowTitle · usageResetConfirmMonthlyTitle |
| S-account-39 | 将消耗 1 次重置次数。 | 确认弹窗描述 | — | 同上 | `AccountPlanUsage.vue:541-543` | i18n:usageSettings.usageResetConfirmDescription |
| S-account-40 | 次数明细加载占位条 | 骨架（animate-pulse，aria-hidden） | — | 弹窗打开且明细仍在加载 | `AccountPlanUsage.vue:552-557` | 无文案 |
| S-account-41 | 运营补偿 · 剩余 {count} 次 · {date} 到期 | 明细列表项（ul li） | — | 明细加载完成且有批次 | `AccountPlanUsage.vue:558-562` · 拼接 `:326-334` | i18n:usageSettings.usageResetSourceAdmin · usageResetAvailable · usageResetExpiresOn |
| S-account-42 | 暂无可用 | 空态文案 | — | 明细加载完成但无批次 | `AccountPlanUsage.vue:563-565` | i18n:usageSettings.usageResetNoneAvailable |
| S-account-43 | 无法加载额度明细 | 错误文案（destructive） | — | 明细接口失败 | `AccountPlanUsage.vue:566-572` | i18n:usageSettings.usageResetBatchesError |
| S-account-44 | 取消 | 弹窗按钮（outline） | — | 弹窗打开 | `AccountPlanUsage.vue:575` | i18n:common.cancel |
| S-account-45 | 使用 | 弹窗主按钮（执行重置） | — | 弹窗打开（无批次或加载中禁用） | `AccountPlanUsage.vue:576-581` | i18n:usageSettings.usageResetUse |
| S-account-46 | 重置已完成。 | toast | — | 重置成功且非重放 | `AccountPlanUsage.vue:391` | i18n:usageSettings.usageResetSuccess |
| S-account-47 | 套餐额度已用完 | 警示区块标题（红左边框） | — | 有套餐且额度已 100% 用尽且未过期 | `AccountPlanUsage.vue:587-598` | i18n:usageSettings.creditDepletedTitle |
| S-account-48 | 当前套餐额度已全部使用，Dim 托管模型暂时不可用。 | 警示说明文案 | — | 同上 | `AccountPlanUsage.vue:599-601` | i18n:usageSettings.creditDepletedDescription |
| S-account-49 | 额度用量 | 不限量块标签（日历图标） | — | 套餐为不限量 | `AccountPlanUsage.vue:605-610` | i18n:usageSettings.creditLabel |
| S-account-50 | 不限量 | 值文本（主色） | — | 同上 | `AccountPlanUsage.vue:611-613` | i18n:usageSettings.creditUnlimited |
| S-account-51 | 当前套餐不限用量，无额度上限。 | 说明文案 | — | 同上 | `AccountPlanUsage.vue:615-617` | i18n:usageSettings.creditUnlimitedDescription |
| S-account-52 | 月度额度 | 分组标签（日历图标） | — | 有可用额度（非不限量） | `AccountPlanUsage.vue:625-630` | i18n:usageSettings.creditMonthlyTitle |
| S-account-53 | {date} 重置 | 文本（右对齐，11px） | — | 已知额度重置日 | `AccountPlanUsage.vue:631-633` | i18n:usageSettings.creditResets |
| S-account-54 | {used} / {total} | 用量数值（tabular-nums） | — | 额度有正的总量 | `AccountPlanUsage.vue:636-638` | 运行期（`monthlyAmountText`） |
| S-account-55 | — | 进度条（role=progressbar，按等级变色） | 满宽减已用 | 有可用额度 | `AccountPlanUsage.vue:639-656` | i18n:usageSettings.creditUsed（aria-label） |
| S-account-56 | 已使用 {pct}% | 百分比文本（按等级变色） | — | 有可用额度 | `AccountPlanUsage.vue:657-664` | i18n:usageSettings.creditUsed |
| S-account-57 | 周期限制 | 子分组标题 | — | 存在滚动窗口行 | `AccountPlanUsage.vue:679` | i18n:usageSettings.windowSectionTitle |
| S-account-58 | 以下窗口包含在月度额度内 | 子分组提示 | — | 同上 | `AccountPlanUsage.vue:680` | i18n:usageSettings.windowSectionHint |
| S-account-59 | {hours} 小时窗口（每个窗口一行） | 行：窗口名 + 用量 + 进度条 + 百分比 + 重置倒计时 | — | 每个滚动窗口一行 | `AccountPlanUsage.vue:682-718` | i18n:usageSettings.windowLabel |
| S-account-60 | 5 小时窗口（无月度额度时独立成行） | 行（同上，无分组标题） | — | 无可用额度但有窗口（如仅窗口限额的套餐） | `AccountPlanUsage.vue:726-768` | i18n:usageSettings.windowLabel |
| S-account-61 | Web Search | 功能配额行标签（地球图标） | — | 存在功能配额（`featureMeters` 非空） | `AccountPlanUsage.vue:775-785` | i18n:usageSettings.featureWebSearch |
| S-account-62 | {date} 重置 | 功能配额行右侧日期 | — | 该功能有周期结束时间 | `AccountPlanUsage.vue:786-788` | i18n:usageSettings.creditResets |
| S-account-63 | {used} / {total} 次 | 功能配额用量数值 | — | 功能配额非不限量 | `AccountPlanUsage.vue:791-793` | i18n:usageSettings.featureAmount |
| S-account-64 | 已使用 {pct}% | 功能配额百分比 | — | 功能配额非不限量 | `AccountPlanUsage.vue:794-820` | i18n:usageSettings.creditUsed |
| S-account-65 | 尚未订阅套餐 | 未订阅块标题 | — | 无套餐且 `subscription` 存在但无额度 | `AccountPlanUsage.vue:826-830` | i18n:usageSettings.noPlanTitle |
| S-account-66 | 订阅套餐后即可解锁 Dim 托管模型。 | 未订阅说明 | — | 同上 | `AccountPlanUsage.vue:831-833` | i18n:usageSettings.noPlanDescription |
| S-account-67 | 查看套餐 | 按钮（outline，外链图标） | — | 同上 | `AccountPlanUsage.vue:835-843` | i18n:accountSettings.viewPlan |
| S-account-68 | 暂无用量数据 | 空态文案 | — | 无套餐且无可用额度 | `AccountPlanUsage.vue:846-848` | i18n:usageSettings.creditEmpty |
| S-account-69 | 续订 / 升级（未过期）/ 选购套餐（已过期） | 主按钮（贴右） | 续订 / 升级 | 有套餐 | `AccountPlanUsage.vue:850-861` | i18n:usageSettings.planRenew · planChoose |
| S-account-70 | 推广邀请 | 区块标题（礼物图标） | — | 推广资格 `referral.eligible` 为真 | `AccountPlanUsage.vue:865-872` | i18n:usageSettings.referralTitle |
| S-account-71 | 好友完成首次订阅付费后，你获得其所购套餐额度 {percent}% 的 Credits。 | 规则文案 | — | 推广启用且邀请人返点 > 0 | `AccountPlanUsage.vue:875-877` | i18n:usageSettings.referralRuleInviter |
| S-account-72 | 好友首次订阅可获得所购套餐额度 {percent}% 的 Credits。 | 规则文案 | — | 推广启用且被邀请人返点 > 0 | `AccountPlanUsage.vue:878-880` | i18n:usageSettings.referralRuleInvitee |
| S-account-73 | 推广活动当前未开启，分享链接仍会把好友的注册记在你名下。 | 规则文案 | — | 两侧返点都缺失 | `AccountPlanUsage.vue:881-883` | i18n:usageSettings.referralCampaignOff |
| S-account-74 | 邀请码 | 标签（11px） | — | 有 `affCode` | `AccountPlanUsage.vue:891-894` | i18n:usageSettings.referralInviteCodeLabel |
| S-account-75 | 邀请码值 | 代码块（等宽 semibold，truncate） | — | 同上 | `AccountPlanUsage.vue:896` | 运行期 |
| S-account-76 | 复制 | 按钮（ghost + 复制图标） | — | 同上 | `AccountPlanUsage.vue:897-905` | i18n:common.copy |
| S-account-77 | 邀请网址 | 标签（11px） | — | 有 `inviteUrl` | `AccountPlanUsage.vue:908-911` | i18n:usageSettings.referralInviteUrlLabel |
| S-account-78 | 邀请网址值 | 代码块（等宽，truncate） | — | 同上 | `AccountPlanUsage.vue:913` | 运行期 |
| S-account-79 | 复制 | 按钮 | — | 同上 | `AccountPlanUsage.vue:914-922` | i18n:common.copy |
| S-account-80 | 已邀请 + {count} 人 | 定义列表行 | — | 推广资格为真 | `AccountPlanUsage.vue:929-934` | i18n:usageSettings.referralInvitedLabel · referralPeopleCount |
| S-account-81 | 已付费 + {count} 人 | 定义列表行 | — | 同上 | `AccountPlanUsage.vue:935-940` | i18n:usageSettings.referralPaidLabel · referralPeopleCount |
| S-account-82 | 累计奖励 + {units} Credits | 定义列表行 | — | 服务端返回已发放奖励 | `AccountPlanUsage.vue:941-947` | i18n:usageSettings.referralRewardLabel · referralRewardCredits |

本节元素数：82

本轮自检：未登录（登录卡片 + 两个按钮 + 错误 + 企业邮箱弹窗）与已登录（身份卡 + UID 三态 + 两个按钮 + 错误）两套；过期横幅；套餐卡四态（不限量 / 月度+周期窗口 / 仅窗口 / 无额度空态）、状态徽标 6 种、重置 pill 与确认弹窗六态、额度耗尽提示、未订阅块、底部 CTA 两文案、推广区块（规则两态 + 未开启 + 两条复制 + 三行统计）。

---

## S-usage 用量（`components/settings/UsageSettings.vue` + `views/settings/account/AccountActivity.vue`）

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-usage-01 | 用量 | 页面标题（h2） | — | 常显 | `UsageSettings.vue:16` | i18n:usageSettings.title |
| S-usage-02 | 不可用 | 区块标题（警示图标） | — | App Server 读模式（`appServerRead`） | `AccountActivity.vue:461-465` | i18n:plugins.unavailable |
| S-usage-03 | 目前只有按会话的模型/用量；聚合结果不完整且不是权威数据。 | 说明文案 | — | 同上 | `AccountActivity.vue:466` | i18n:wipCapabilities.global_usage.reason |
| S-usage-04 | 不可用 | 区块标题 | — | 用量能力不可用（`capability_unavailable`） | `AccountActivity.vue:471-479` | i18n:plugins.unavailable |
| S-usage-05 | 查看用量活动需要连接 App Server。 | 说明文案 | — | 同上 | `AccountActivity.vue:480` | i18n:accountSettings.usageUnavailableDescription |
| S-usage-06 | 正在加载... | 加载占位卡 | — | 首次加载中 | `AccountActivity.vue:485-487` | i18n:common.loading |
| S-usage-07 | 用量加载失败 | 错误卡标题（destructive） | — | 加载失败 | `AccountActivity.vue:489-493` | i18n:accountSettings.activityErrorTitle |
| S-usage-08 | 错误详情 | 错误卡正文 | — | 加载失败 | `AccountActivity.vue:494` | 运行期 |
| S-usage-09 | 正在后台导入历史用量。 | 提示条（天蓝色） | — | 回填状态 running | `AccountActivity.vue:500-502` | i18n:accountSettings.backfillRunning |
| S-usage-10 | {processed} / {total} | 进度文本（等宽） | 游标字符串或空 | 回填 running 且有进度 | `AccountActivity.vue:503` | 运行期（`backfillProgressText`） |
| S-usage-11 | 历史用量导入失败，新的用量仍会继续显示。 | 提示条（琥珀色） | — | 回填状态 failed | `AccountActivity.vue:506-508` | i18n:accountSettings.backfillFailed |
| S-usage-12 | 累计 tokens | 概览指标卡（图标 + 数值 + 标签） | 0 | 加载成功 | `AccountActivity.vue:513-521` · 文案 `:80-84` | i18n:accountSettings.lifetimeTokens |
| S-usage-13 | 单日峰值 tokens | 概览指标卡 | 0 | 同上 | `AccountActivity.vue:513-521` · 文案 `:85-90` | i18n:accountSettings.peakDailyTokens |
| S-usage-14 | 最长任务 | 概览指标卡（时长） | `-` | 同上 | `AccountActivity.vue:513-521` · 文案 `:91-96` | i18n:accountSettings.longestTask |
| S-usage-15 | 当前连续 | 概览指标卡（`{count} 天`） | 0 天 | 同上 | `AccountActivity.vue:513-521` · 文案 `:97-102` | i18n:accountSettings.currentStreak |
| S-usage-16 | 最长连续 | 概览指标卡（`{count} 天`） | 0 天 | 同上 | `AccountActivity.vue:513-521` · 文案 `:103-108` | i18n:accountSettings.longestStreak |
| S-usage-17 | Token 活跃度 | 卡片标题（h3） | — | 加载成功 | `AccountActivity.vue:527-533` | i18n:accountSettings.tokenActivity |
| S-usage-18 | 部分数据 · 仅 App Server 会话 | 徽标（灰底小字） | — | App Server 读模式 | `AccountActivity.vue:529-532` | i18n:accountSettings.activityPartial |
| S-usage-19 | {fromDay} - {toDay} | 日期范围文本 | 最近 53 周 | 加载成功 | `AccountActivity.vue:534` | 运行期 |
| S-usage-20 | 刷新 | 按钮（ghost + 图标，加载中转圈） | — | 加载成功 | `AccountActivity.vue:536-548` | i18n:common.refresh |
| S-usage-21 | 还没有用量 | 空态标题（虚线框） | — | 无任何运行（`totalRuns === 0`） | `AccountActivity.vue:551-552` | i18n:accountSettings.activityEmptyTitle |
| S-usage-22 | 完成运行后这里会显示用量。 | 空态说明 | — | 同上 | `AccountActivity.vue:553` | i18n:accountSettings.activityEmptyDescription |
| S-usage-23 | 月份标签（如 `1月`） | 热力图顶部标签行 | — | 有用量数据 | `AccountActivity.vue:557-559` | 运行期（`formatMonth`） |
| S-usage-24 | 热力格（每格一天） | 格子（12px，title/aria-label 带当日明细） | 按 token 分 5 档 | 有用量数据 | `AccountActivity.vue:560-579` | i18n:accountSettings.heatmapTooltip |
| S-usage-25 | 活跃度指标 | 卡片标题（h3） | — | 加载成功 | `AccountActivity.vue:584-585` | i18n:accountSettings.activitySummary |
| S-usage-26 | 总会话数 | 指标行（标签 + 数值） | 0 | 同上 | `AccountActivity.vue:587-590` | i18n:accountSettings.totalThreads |
| S-usage-27 | 总运行数 | 指标行 | 0 | 同上 | `AccountActivity.vue:591-594` | i18n:accountSettings.totalRuns |
| S-usage-28 | 平均 tokens / 运行 | 指标行 | 0 | 同上 | `AccountActivity.vue:595-598` | i18n:accountSettings.averageTokensPerRun |
| S-usage-29 | 最长任务 | 指标行（时长文案） | `-` | 同上 | `AccountActivity.vue:599-602` | i18n:accountSettings.longestTask |
| S-usage-30 | Token 指标 | 卡片标题（h3） | — | 加载成功 | `AccountActivity.vue:606-607` | i18n:accountSettings.tokenSummary |
| S-usage-31 | 输入 tokens | 指标行 | 0 | 同上 | `AccountActivity.vue:609-612` | i18n:accountSettings.inputTokens |
| S-usage-32 | 输出 tokens | 指标行 | 0 | 同上 | `AccountActivity.vue:613-616` | i18n:accountSettings.outputTokens |
| S-usage-33 | 缓存读取 tokens | 指标行 | 0 | 同上 | `AccountActivity.vue:617-620` | i18n:accountSettings.cacheReadTokens |
| S-usage-34 | 缓存写入 tokens | 指标行 | 0 | 同上 | `AccountActivity.vue:621-624` | i18n:accountSettings.cacheWriteTokens |
| S-usage-35 | Cache 指标 | 卡片标题（h3） | — | 加载成功 | `AccountActivity.vue:628-629` | i18n:accountSettings.cacheSummary |
| S-usage-36 | 缓存 tokens | 指标行（读 + 写求和） | 0 | 同上 | `AccountActivity.vue:631-634` | i18n:accountSettings.cacheTokens |
| S-usage-37 | Cache 命中率 | 指标行（`{pct}%`，未知显示 `—`） | — | 同上 | `AccountActivity.vue:635-638` | i18n:accountSettings.cacheHitRate |
| S-usage-38 | 最常用 skills | 卡片标题（h3） | — | 加载成功 | `AccountActivity.vue:646` | i18n:accountSettings.topSkills |
| S-usage-39 | {page} / {totalPages} | 页码范围文本 | — | 有 skill 数据 | `AccountActivity.vue:648-650` | 运行期 |
| S-usage-40 | 上一页 | 图标按钮（左箭头，首页禁用） | — | 有 skill 数据 | `AccountActivity.vue:652-663` | i18n:accountSettings.previousPage |
| S-usage-41 | 下一页 | 图标按钮（右箭头，末页禁用） | — | 有 skill 数据 | `AccountActivity.vue:664-675` | i18n:accountSettings.nextPage |
| S-usage-42 | 这个范围内还没有使用 skills。 | 空态文案 | — | skill 列表为空 | `AccountActivity.vue:679-681` | i18n:accountSettings.noSkills |
| S-usage-43 | $`{skillName}` + `{count} 次` | 列表行（名 + 运行次数） | — | 有 skill 数据 | `AccountActivity.vue:690-699` | i18n:accountSettings.runsCount |
| S-usage-44 | 模型用量 | 卡片标题（h3） | — | 加载成功 | `AccountActivity.vue:706` | i18n:accountSettings.modelUsage |
| S-usage-45 | {page} / {totalPages} | 页码范围文本 | — | 有模型数据 | `AccountActivity.vue:708-710` | 运行期 |
| S-usage-46 | 上一页 | 图标按钮 | — | 有模型数据 | `AccountActivity.vue:712-723` | i18n:accountSettings.previousPage |
| S-usage-47 | 下一页 | 图标按钮 | — | 有模型数据 | `AccountActivity.vue:724-735` | i18n:accountSettings.nextPage |
| S-usage-48 | 这个范围内还没有模型用量。 | 空态文案 | — | 模型列表为空 | `AccountActivity.vue:739-741` | i18n:accountSettings.noModels |
| S-usage-49 | 模型名（`{modelId}`） | 列表行主标题 | — | 有模型数据 | `AccountActivity.vue:757` | 运行期 |
| S-usage-50 | 供应商 id（`{providerId}`） | 列表行副标题 | — | 同上 | `AccountActivity.vue:758` | 运行期 |
| S-usage-51 | 总 tokens / `{count} 次` | 列表行右侧数值 + 说明 | 0 | 同上 | `AccountActivity.vue:760-763` | i18n:accountSettings.totalTokens · runsCount |
| S-usage-52 | 输入 tokens | 列表行子指标（右对齐数值） | 0 | 同上 | `AccountActivity.vue:767-770` | i18n:accountSettings.inputTokens |
| S-usage-53 | 输出 tokens | 子指标 | 0 | 同上 | `AccountActivity.vue:771-774` | i18n:accountSettings.outputTokens |
| S-usage-54 | 缓存读取 | 子指标 | 0 | 同上 | `AccountActivity.vue:775-778` | i18n:accountSettings.cacheReadShort |
| S-usage-55 | 缓存写入 | 子指标 | 0 | 同上 | `AccountActivity.vue:779-782` | i18n:accountSettings.cacheWriteShort |

本节元素数：55

本轮自检：App Server 读模式 WIP 块、能力不可用块、加载中、错误块、回填进行中（含进度）与回填失败；5 个概览指标；活跃度卡（标题、部分数据徽标、日期范围、刷新、空态、月份标签、按 token 分 5 档的热力格与 tooltip）；三张汇总卡共 10 行；两个分页列表（空态、页码、上一页/下一页禁用）。

---

## S-personalization 个性化（`components/settings/PersonalizationSettings.vue` 及其子组件）

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-personalization-01 | 个性化 | 页面标题（h2） | — | 常显（默认落地节） | `PersonalizationSettings.vue:111` | i18n:personalizationSettings.title |
| S-personalization-02 | 界面模式 | 分组标签 | — | 常显 | `PersonalizationSettings.vue:116` | i18n:personalizationSettings.uiModeTitle |
| S-personalization-03 | 办公模式隐藏开发细节、以结果为先；编程模式展示完整开发界面。 | 说明文案 | — | 常显 | `PersonalizationSettings.vue:117-119` | i18n:personalizationSettings.uiModeDescription |
| S-personalization-04 | 办公模式 + 更专注于文档、表格、设计等日常办公，把事做好就行。 | 可选卡片（按钮） | 未选（未选择时运行编程模式） | 常显 | `PersonalizationSettings.vue:122-135` · 文案 `:43` | i18n:modeSelection.simpleLabel · simpleDesc |
| S-personalization-05 | 编程模式 + 面向写代码和工程开发，diff、终端、分支等尽在掌控。 | 可选卡片（按钮） | 未选（运行时等价 pro） | 常显 | `PersonalizationSettings.vue:122-135` · 文案 `:44` | i18n:modeSelection.proLabel · proDesc |
| S-personalization-06 | 切换模式失败，请重试。 | 错误通知（toast） | — | 保存模式失败 | `PersonalizationSettings.vue:57` | i18n:modeSelection.saveFailed |
| S-personalization-07 | 展示 ACP 会话 | 开关行标签 | — | 常显 | `PersonalizationSettings.vue:143` | i18n:personalizationSettings.showAcpSessions |
| S-personalization-08 | 显示 ACP 创建的会话及其工作区。 | 开关行说明 | — | 常显 | `PersonalizationSettings.vue:144` | i18n:personalizationSettings.showAcpSessionsDescription |
| S-personalization-09 | — | 开关（aria-describedby 关联说明） | 关 | 常显 | `PersonalizationSettings.vue:146` | 运行期（`sidebarStore.showAcpSessions`，默认 false） |
| S-personalization-10 | 默认打开目标应用 | 分组标签 | — | 常显 | `PersonalizationSettings.vue:153` | i18n:personalizationSettings.defaultOpenAppTitle |
| S-personalization-11 | 选择在外部编辑器中打开文件时使用的应用 | 说明文案 | — | 常显 | `PersonalizationSettings.vue:154-156` | i18n:personalizationSettings.defaultOpenAppDescription |
| S-personalization-12 | 默认打开应用配置无效 | 错误文案（destructive） | — | 读取已存配置失败 | `PersonalizationSettings.vue:157-159` | i18n:personalizationSettings.invalidDefaultOpenApp |
| S-personalization-13 | 系统默认 / 在 Finder 中显示（Windows：在文件资源管理器中显示）/ VS Code / Cursor / Zed / Terminal / Ghostty / Warp / Xcode / IntelliJ IDEA / WebStorm / PyCharm / Sublime Text / iTerm2 / Alacritty / WezTerm / Kitty | 下拉（17 个选项，每项带应用图标） | 系统默认 | 配置加载完成后可用 | `PersonalizationSettings.vue:161-180` · 选项 `:65-75` · 名称 `open-ide.ts:21-35` | i18n:personalizationSettings.systemDefault · i18n:openIde.showInFinder · 字面量（应用名） |
| S-personalization-14 | 保存设置失败 | 错误通知（toast） | — | 保存默认打开应用失败 | `PersonalizationSettings.vue:86` | i18n:common.saveFailed |
| S-personalization-15 | 助手模型 | 分组标签（`SubagentModelSetting kind="assistant"`） | — | 常显 | `SubagentModelSetting.vue:509-511` · 实例 `PersonalizationSettings.vue:185` | i18n:personalizationSettings.assistantModelTitle |
| S-personalization-16 | 用于工具自动安全审计等轻量工作。 | 说明文案 | — | 助手模型实例 | `SubagentModelSetting.vue:512-514` | i18n:personalizationSettings.assistantModelDescription |
| S-personalization-17 | 触发按钮（显示当前选择：继承父会话 / 指定模型名 / App Server 模式下不可用） | 按钮（outline，整宽，左侧状态图标 + 右侧 chevron） | 继承父会话 | 常显 | `SubagentModelSetting.vue:516-539` · 标签逻辑 `:177-182` | i18n:personalizationSettings.assistantModel* / configurationSettings.subagentModel* |
| S-personalization-18 | 错误条（保存失败信息） | 弹窗顶部错误条 | — | 弹窗打开且出错 | `SubagentModelSetting.vue:546-553` | 运行期 |
| S-personalization-19 | 「模型选择」弹窗步骤 1 标题 | 弹窗标题 | — | 点击触发按钮 | `SubagentModelSetting.vue:557-559` | 同上（`titleLabel`） |
| S-personalization-20 | 继承父会话 | 策略卡（单选按钮 + 勾选态） | 选中（默认） | 步骤 1 | `SubagentModelSetting.vue:563-581` | i18n:personalizationSettings.assistantModelInherit / configurationSettings.subagentModelInherit |
| S-personalization-21 | 助手任务沿用当前父会话模型。父会话切换模型后，助手任务跟着切换。 | 策略卡说明 | — | 步骤 1 | `SubagentModelSetting.vue:579` | i18n:personalizationSettings.assistantModelInheritDescription |
| S-personalization-22 | 指定模型 | 策略卡（带右箭头） | — | 步骤 1 | `SubagentModelSetting.vue:606-637` | i18n:personalizationSettings.assistantModelSpecific |
| S-personalization-23 | 为助手任务指定一个固定模型，例如 approve-for-me 审核。 | 策略卡说明 | — | 步骤 1 | `SubagentModelSetting.vue:623` | i18n:personalizationSettings.assistantModelSpecificDescription |
| S-personalization-24 | 当前已选模型名 | 策略卡内徽标（图标 + 名称） | — | 已选固定模型 | `SubagentModelSetting.vue:624-635` | 运行期 |
| S-personalization-25 | 筛选模型 | 搜索输入框（占位，带放大镜） | 空 | 步骤 2 | `SubagentModelSetting.vue:655-663` | i18n:personalizationSettings.assistantModelFilterPlaceholder |
| S-personalization-26 | 供应商列表（名称） | 左侧可滚动列表按钮 | 首个可用供应商 | 步骤 2 | `SubagentModelSetting.vue:667-679` | 运行期 |
| S-personalization-27 | 没有匹配的模型 | 空态（供应商列） | — | 搜索无匹配供应商 | `SubagentModelSetting.vue:680-682` | i18n:personalizationSettings.assistantModelNoMatches |
| S-personalization-28 | 模型列表（名称 + 描述 + 勾选态 + 右箭头） | 行（role=button，锁定模型置灰不可点） | 当前模型高亮 | 步骤 2 | `SubagentModelSetting.vue:686-709` | 运行期 |
| S-personalization-29 | —— | 模型描述占位（无描述时） | —— | 模型无描述 | `SubagentModelSetting.vue:706-708` | i18n:common.noDescription |
| S-personalization-30 | 没有匹配的模型 | 空态（模型列） | — | 搜索无匹配模型 | `SubagentModelSetting.vue:710-712` | i18n:personalizationSettings.assistantModelNoMatches |
| S-personalization-31 | 思考强度 | 弹窗标题（步骤 3，带返回箭头） | — | 所选模型有推理强度选项 | `SubagentModelSetting.vue:719-728` | i18n:inputArea.effort.submenuTitle |
| S-personalization-32 | 待选模型名 | 步骤 3 顶部模型徽标 | — | 步骤 3 | `SubagentModelSetting.vue:732-735` | 运行期 |
| S-personalization-33 | 关闭 / 极低 / 低 / 中 / 高 / 极高 / 最高 | 强度选项按钮（带强度图标 + 勾选态） | 模型默认强度 | 步骤 3 | `SubagentModelSetting.vue:736-750` | i18n:inputArea.reasoning.none/minimal/low/medium/high/xhigh/max |
| S-personalization-34 | 子代理模型 | 分组标签（`SubagentModelSetting`，无 kind） | — | 常显 | `SubagentModelSetting.vue:509-511` · 实例 `PersonalizationSettings.vue:189` | i18n:configurationSettings.subagentModel |
| S-personalization-35 | 继承父会话 | 策略卡 | 选中（默认） | 子代理模型弹窗步骤 1 | `SubagentModelSetting.vue:563-581` | i18n:configurationSettings.subagentModelInherit |
| S-personalization-36 | 子 agent 直接沿用父 agent 当前的模型。父 agent 换模型，子 agent 跟着换。 | 策略卡说明 | — | 步骤 1 | `SubagentModelSetting.vue:579` | i18n:configurationSettings.subagentModelInheritDescription |
| S-personalization-37 | 模型自主选择 / Agent 选择 | 策略卡（二者按 App Server 策略二选一显示） | — | 步骤 1（仅子代理模型实例有此卡） | `SubagentModelSetting.vue:584-603` | i18n:configurationSettings.subagentModelAgentDecides · subagentModelPolicyAgent |
| S-personalization-38 | Agent 按每个子任务的难度，自动从可用模型里挑最合适的。/ 新成员使用所请求 Agent 配置中的模型；配置未指定时使用默认模型。 | 策略卡说明 | — | 步骤 1 | `SubagentModelSetting.vue:601` | i18n:configurationSettings.subagentModelAgentDecidesDescription · subagentModelPolicyAgentDescription |
| S-personalization-39 | 指定模型 | 策略卡 | — | 步骤 1 | `SubagentModelSetting.vue:606-637` | i18n:configurationSettings.subagentModelSpecific |
| S-personalization-40 | 为所有子 agent 指定一个固定模型。/ 新成员使用此固定模型。已有成员保持原模型。 | 策略卡说明 | — | 步骤 1 | `SubagentModelSetting.vue:623` | i18n:configurationSettings.subagentModelSpecificDescription · subagentModelPolicyFixedDescription |
| S-personalization-41 | 继承父会话 | 策略卡说明（App Server 策略版） | — | 步骤 1 且 App Server 策略可用 | `SubagentModelSetting.vue:579` | i18n:configurationSettings.subagentModelPolicyInheritDescription |
| S-personalization-42 | App Server 策略不可用 | 触发按钮文案（策略不可用时禁用触发） | — | App Server 未提供 Collaboration 模型策略 | `SubagentModelSetting.vue:522` · `:177-178` | i18n:configurationSettings.subagentModelUnavailable |
| S-personalization-43 | 当前 App Server 未提供 Collaboration 模型策略。请升级 App Server 后再修改子代理模型。 | 不可用说明 | — | 同上 | `SubagentModelSetting.vue:757-763` | i18n:configurationSettings.subagentModelUnavailableDescription |
| S-personalization-44 | 子代理总数限制 | 分组标签 | — | 常显 | `SubagentCountSetting.vue:68-70` | i18n:personalizationSettings.subagentCountTitle |
| S-personalization-45 | {count} 个子代理 / 不可配置 | 值文本（右侧） | 100 个子代理 | 常显；App Server 读模式显示「不可配置」 | `SubagentCountSetting.vue:71-75` | i18n:personalizationSettings.subagentCountValue · subagentCountUnavailable |
| S-personalization-46 | — | 滑杆（1–100，步长 1） | 100 | App Server 读模式下禁用 | `SubagentCountSetting.vue:82-91` | 无文案（aria-label 用标题） |
| S-personalization-47 | 保存子代理总数限制失败 | 错误文案（destructive） | — | 读取或保存失败 | `SubagentCountSetting.vue:77-79` | i18n:personalizationSettings.subagentCountSaveFailed |
| S-personalization-48 | App Server 模式没有成员并发设置，因此 Desktop 不会应用子代理上限。 | 说明文案 | — | App Server 读模式 | `SubagentCountSetting.vue:93-99` | i18n:personalizationSettings.subagentCountUnavailableDescription |
| S-personalization-49 | 命令 Shell | 分组标签 | — | 常显 | `ExecShellSetting.vue:183-185` | i18n:personalizationSettings.execShellTitle |
| S-personalization-50 | 选择 Agent 内置 exec 工具使用的 Shell。 | 说明文案 | — | 常显 | `ExecShellSetting.vue:186-188` | i18n:personalizationSettings.execShellDescription |
| S-personalization-51 | 共享的命令 Shell 配置无效：{path} | 错误文案（destructive） | — | 共享配置无法解析 | `ExecShellSetting.vue:191-194` | i18n:personalizationSettings.execShellInvalidConfig |
| S-personalization-52 | 重置为自动 | 按钮（outline） | — | 同上（配置无效） | `ExecShellSetting.vue:195-197` | i18n:personalizationSettings.execShellReset |
| S-personalization-53 | 将无效的命令 Shell 配置重置为自动？ | 系统确认框（`window.confirm`） | — | 点击「重置为自动」 | `ExecShellSetting.vue:140` | i18n:personalizationSettings.execShellResetConfirm |
| S-personalization-54 | 自动 — {shell} / 自动（没有受支持的 Shell）/ 正在探测… / 已保存的 Shell 不可用：{path} | 下拉触发按钮文案 | 自动 | 配置有效 | `ExecShellSetting.vue:201-212` · 文案 `:60-76` | i18n:personalizationSettings.execShellAutomatic · execShellAutomaticUnavailable · execShellDetecting · execShellUnavailablePath |
| S-personalization-55 | 自动 — {shell}（不可选时置灰） | 下拉选项 1 | 无自动目标时禁用 | 下拉展开 | `ExecShellSetting.vue:214-220` | i18n:personalizationSettings.execShellAutomatic |
| S-personalization-56 | 已保存的 Shell 不可用：{path} | 下拉选项（禁用，路径截断 + title） | — | 已存路径不在探测结果中 | `ExecShellSetting.vue:221-230` | i18n:personalizationSettings.execShellUnavailablePath |
| S-personalization-57 | {displayName} · {executablePath} | 下拉选项（每个探测到的 Shell 一项） | — | 下拉展开 | `ExecShellSetting.vue:231-240` | 运行期 |
| S-personalization-58 | 刷新 / 正在探测… | 按钮（outline，探测中改文案禁用） | — | 配置有效 | `ExecShellSetting.vue:245-247` | i18n:common.refresh · i18n:personalizationSettings.execShellDetecting |
| S-personalization-59 | 有 {count} 个 Shell 候选项无法验证 | 说明文案 | — | 探测有失败候选 | `ExecShellSetting.vue:248-250` | i18n:personalizationSettings.execShellPartialFailure |
| S-personalization-60 | 自动选择不可用。请选择可用 Shell，或安装后刷新。 | 警示文案（destructive） | — | 未探测到自动目标 | `ExecShellSetting.vue:253-255` | i18n:personalizationSettings.execShellNone |
| S-personalization-61 | 已保存的 Shell 不可用：{path} | 警示文案（destructive） | — | 已存路径不可用 | `ExecShellSetting.vue:256-258` | i18n:personalizationSettings.execShellUnavailablePath |
| S-personalization-62 | 命令提示符无法从网络共享路径启动此任务。请选择其他 Shell 或本地工作目录。 | 警示文案（destructive） | — | 选中 cmd 且会话 cwd 为 UNC 路径 | `ExecShellSetting.vue:259-261` | i18n:personalizationSettings.execShellCmdUnc |
| S-personalization-63 | 非 UTF-8 命令输出可能显示为字节转义（<0xNN>）。文件工具仍使用 Unicode 路径。 | 说明文案 | — | 所选 Shell 输出编码未知 | `ExecShellSetting.vue:262-264` | i18n:personalizationSettings.execShellUnknownOutput |
| S-personalization-64 | 当前任务继续使用 {shell}。请新建任务以应用此选择。 | 说明文案 | — | 运行中的执行器与所选 Shell 不一致 | `ExecShellSetting.vue:265-267` | i18n:personalizationSettings.execShellCurrentUnchanged |
| S-personalization-65 | TUI 和 Desktop 共享此设置。新建或重新加载的任务使用它；当前任务保持原 Shell。 | 说明文案 | — | 配置有效 | `ExecShellSetting.vue:268-270` | i18n:personalizationSettings.execShellScope |
| S-personalization-66 | 已有任务以其他 Shell 重新加载时，可能降低提示词缓存复用率并消耗额外 token。 | 说明文案 | — | 配置有效 | `ExecShellSetting.vue:271-273` | i18n:personalizationSettings.execShellCache |
| S-personalization-67 | 命令 Shell 已保存 | 成功通知（toast） | — | 保存或重置成功 | `ExecShellSetting.vue:127` · `:149` | i18n:personalizationSettings.execShellSaved |
| S-personalization-68 | 加载命令 Shell 失败 / 接口错误信息 | 错误文案（destructive） | — | 读取或保存失败 | `ExecShellSetting.vue:276-278` | i18n:personalizationSettings.execShellLoadFailed |
| S-personalization-69 | 提示词语言 | 分组标签 | — | **仅 App Server 读模式** | `PromptLocaleSetting.vue:30-32` | i18n:personalizationSettings.promptLocaleTitle |
| S-personalization-70 | 提示词语言已禁用：App Server 会话还没有语言载体，Agent 无法接收此设置。Desktop 不会写入本地提示词语言文件。 | 说明文案 | — | 同上 | `PromptLocaleSetting.vue:33-35` | i18n:personalizationSettings.promptLocaleUnavailableDescription |
| S-personalization-71 | Owner 缺口： | 说明文案（前缀） | — | 同上 | `PromptLocaleSetting.vue:36-37` | i18n:personalizationSettings.promptLocaleOwnerGap |
| S-personalization-72 | arcships/dim#1018 | 外链（新窗口） | — | 同上 | `PromptLocaleSetting.vue:38-43` | 字面量（链接文本） |
| S-personalization-73 | arcships/dim#548 | 外链（新窗口） | — | 同上 | `PromptLocaleSetting.vue:44-50` | 字面量（链接文本） |
| S-personalization-74 | App Server 模式下不可用（占位） | 下拉（禁用，11 个语言选项） | 空 | 同上 | `PromptLocaleSetting.vue:53-60` | i18n:personalizationSettings.promptLocaleUnavailable |
| S-personalization-75 | English / 简体中文 / 日本語 / 한국어 / 繁體中文 / Bahasa Indonesia / Deutsch / Français / Español / Português (Brasil) / Português (Portugal) | 下拉选项（11 项） | 无选中 | 下拉展开（禁用的下拉不展开） | `PromptLocaleSetting.vue:11-23` · `:58` | 字面量（语言名） |
| S-personalization-76 | 自定义指令（分组标题） | 分组标题（h3） | — | 常显 | `CustomInstructionsSettings.vue:85` | i18n:customInstructionsSettings.title |
| S-personalization-77 | 自定义指令说明（App Server 版 / 本地版二选一） | 说明文案 | — | 常显（两套文案按模式切换） | `CustomInstructionsSettings.vue:86-90` | i18n:customInstructionsSettings.appServerDescription · description |
| S-personalization-78 | ~/AGENTS.md（实际路径） | 代码块（等宽，truncate，title 显示全路径） | `~/AGENTS.md` | App Server 读模式 | `CustomInstructionsSettings.vue:99-105` | 字面量（默认显示路径 `:13`） |
| S-personalization-79 | 打开 AGENTS.md | 按钮 | — | App Server 读模式 | `CustomInstructionsSettings.vue:106-112` | i18n:customInstructionsSettings.openAgentsFile |
| S-personalization-80 | AGENTS.md 说明（全局指令文件由 PromptSources 读取） | 说明文案 | — | App Server 读模式 | `CustomInstructionsSettings.vue:114-116` | i18n:customInstructionsSettings.agentsFileNote |
| S-personalization-81 | 打开 AGENTS.md 失败信息 | 错误文案（destructive） | — | 打开失败 | `CustomInstructionsSettings.vue:117-119` | i18n:customInstructionsSettings.openAgentsFileFailed |
| S-personalization-82 | 指令输入区 | 多行文本框（等宽，min-h 280 / max-h 420，滚动） | 空（载入已存文本） | 非 App Server 模式 | `CustomInstructionsSettings.vue:124-129` | i18n:customInstructionsSettings.instructionsPlaceholder |
| S-personalization-83 | 自定义指令（屏幕阅读器标签） | 隐藏标签（sr-only） | — | 非 App Server 模式 | `CustomInstructionsSettings.vue:123` | i18n:customInstructionsSettings.instructionsLabel |
| S-personalization-84 | 保存 | 按钮（无改动或保存中禁用） | 禁用（无改动） | 非 App Server 模式 | `CustomInstructionsSettings.vue:134-136` | i18n:customInstructionsSettings.save |
| S-personalization-85 | 错误信息（保存失败 / 加载失败） | 错误文案（destructive） | — | 保存或加载失败 | `CustomInstructionsSettings.vue:130-133` | i18n:customInstructionsSettings.saveFailed · i18n:common.loadFailed |

本节元素数：85

本轮自检：界面模式未选/办公/编程三态与切换失败；展示 ACP 会话开关；默认打开应用下拉 17 项 + 配置无效错误 + 保存失败；助手模型与子代理模型两个实例（步骤 1 策略卡、步骤 2 搜索/供应商列/模型列/两处空态、步骤 3 强度、锁定与不可用态）；子代理数量滑杆（可配置/不可配置/保存失败）；命令 Shell（无效配置、重置确认、自动/不可用/各 Shell 选项、刷新、5 条说明、保存与错误）；提示词语言（仅 App Server、两处外链、禁用下拉 11 项）；自定义指令两分支（外部文件 + 打开失败、文本框 + 保存禁用）。

---

## S-appearance 外观（`components/settings/AppearanceSettings.vue` + `SkinSettings.vue`）

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-appearance-01 | 外观 | 页面标题（h2） | — | 常显 | `AppearanceSettings.vue:265` | i18n:appearanceSettings.title |
| S-appearance-02 | 主题 | 分组标签 | — | 常显 | `AppearanceSettings.vue:270` | i18n:appearanceSettings.theme |
| S-appearance-03 | 浅色 | 主题卡（按钮，太阳图标，选中带圆点） | — | 常显 | `AppearanceSettings.vue:272-287` · 选项 `:193` | i18n:appearanceSettings.theme.light |
| S-appearance-04 | 深色 | 主题卡（月亮图标） | — | 常显 | `AppearanceSettings.vue:272-287` · 选项 `:194` | i18n:appearanceSettings.theme.dark |
| S-appearance-05 | 跟随系统 | 主题卡（显示器图标） | **选中（默认 system）** | 常显 | `AppearanceSettings.vue:272-287` · 选项 `:195` | i18n:appearanceSettings.theme.system |
| S-appearance-06 | 经典外观使用浅色；切回默认或 macOS 时恢复原明暗偏好。 | 说明文案 | — | 当前为历史外观（经典 chrome 主题） | `AppearanceSettings.vue:289` | i18n:appearanceSettings.chromeClassicLight |
| S-appearance-07 | 程序语言 | 分组标签 | — | 常显 | `AppearanceSettings.vue:297` | i18n:appearanceSettings.language |
| S-appearance-08 | 简体中文 / English / 日本語 / 한국어 / 繁體中文 / Bahasa Indonesia / Deutsch / Français / Español / Português (Brasil) / Português (Portugal) | 下拉（11 个选项） | 当前界面语言 | 常显 | `AppearanceSettings.vue:298-307` · 选项 `:80-96` | i18n:appearanceSettings.language.* |
| S-appearance-09 | 切换语言失败，请重试。 | 错误通知（toast） | — | 语言包加载失败 | `AppearanceSettings.vue:222` | i18n:appearanceSettings.languageSwitchFailed |
| S-appearance-10 | 字体 | 分组标题 | — | 常显 | `AppearanceSettings.vue:315` | i18n:appearanceSettings.fonts |
| S-appearance-11 | UI 字体 | 标签 | — | 常显 | `AppearanceSettings.vue:323` · 配置 `:104` | i18n:appearanceSettings.uiFont |
| S-appearance-12 | 用于应用界面和 Markdown 正文 | 说明文案 | — | 常显 | `AppearanceSettings.vue:324` · 配置 `:105` | i18n:appearanceSettings.uiFontDescription |
| S-appearance-13 | UI 字体选择器（当前字体名 / 默认） | 组合框触发按钮（整宽→sm:w-64，截断 + chevron） | 默认 | 系统字体枚举完成后可用 | `AppearanceSettings.vue:326-343` | i18n:appearanceSettings.fontDefault |
| S-appearance-14 | 代码字体 | 标签 | — | 常显 | `AppearanceSettings.vue:323` · 配置 `:110` | i18n:appearanceSettings.codeFont |
| S-appearance-15 | 用于终端、代码块和差异内容 | 说明文案 | — | 常显 | `AppearanceSettings.vue:324` · 配置 `:111` | i18n:appearanceSettings.codeFontDescription |
| S-appearance-16 | 代码字体选择器（当前字体名 / 默认） | 组合框触发按钮 | 默认 | 同上 | `AppearanceSettings.vue:326-343` | i18n:appearanceSettings.fontDefault |
| S-appearance-17 | 搜索字体 | 组合框内搜索输入（占位） | 空 | 组合框展开 | `AppearanceSettings.vue:348-352` | i18n:appearanceSettings.fontSearchPlaceholder |
| S-appearance-18 | 没有匹配的字体 | 组合框空态 | — | 搜索无匹配 | `AppearanceSettings.vue:354` | i18n:appearanceSettings.fontNoMatches |
| S-appearance-19 | 默认 | 组合框选项（未搜索时置顶，选中带勾） | 选中（未设字体时） | 组合框展开且搜索框为空 | `AppearanceSettings.vue:356-359` | i18n:appearanceSettings.fontDefault |
| S-appearance-20 | {family} | 组合框选项（每个系统字体一项，选中带勾） | — | 组合框展开 | `AppearanceSettings.vue:362-370` | 运行期（系统字体名） |
| S-appearance-21 | 正在加载... | 加载文案 | — | 系统字体尚未枚举完成 | `AppearanceSettings.vue:377-379` | i18n:common.loading |
| S-appearance-22 | 无法读取系统字体列表。 | 错误文案（destructive） | — | 字体枚举失败 | `AppearanceSettings.vue:380-381` | i18n:appearanceSettings.fontEnumerationFailed |
| S-appearance-23 | 重试 / 正在加载... | 按钮（ghost，刷新中改文案并禁用） | — | 字体枚举失败 | `AppearanceSettings.vue:382-384` | i18n:common.retry · common.loading |
| S-appearance-24 | 字体“{font}”已不可用，已恢复为默认字体。 | 提示文案（每个缺失字体一行） | — | 已存字体不存在 | `AppearanceSettings.vue:386-388` | i18n:appearanceSettings.fontMissing |
| S-appearance-25 | 保存字体设置失败，已恢复之前的选择。 | 错误文案（destructive） | — | UI 或代码字体保存失败 | `AppearanceSettings.vue:389-391` | i18n:appearanceSettings.fontSaveFailed |
| S-appearance-26 | UI 字号 | 分组标签 | — | 常显 | `AppearanceSettings.vue:401` | i18n:appearanceSettings.uiFontSize |
| S-appearance-27 | 小 | 刻度文案（左端） | — | 常显 | `AppearanceSettings.vue:404` | i18n:appearanceSettings.small |
| S-appearance-28 | 特小 / 小 / 中 / 大 / 特大 | 字号档位按钮组（5 个，选中为主色） | 中（14px） | 常显 | `AppearanceSettings.vue:406-415` · 档位 `appearance/index.ts:15` | i18n:appearanceSettings.fontSizeXS/S/M/L/XL |
| S-appearance-29 | 大 | 刻度文案（右端） | — | 常显 | `AppearanceSettings.vue:417` | i18n:appearanceSettings.large |
| S-appearance-30 | 代码字号 | 分组标签 | — | 常显 | `AppearanceSettings.vue:429` | i18n:appearanceSettings.codeFontSize |
| S-appearance-31 | {n}px | 数值文本（tabular-nums） | 13px | 常显 | `AppearanceSettings.vue:431` | 运行期（值 + 字面量 `px`） |
| S-appearance-32 | — | 滑杆（12–20，步长 1） | 13 | 常显 | `AppearanceSettings.vue:433-439` | 无文案 |
| S-appearance-33 | 12px / 16px / 20px | 刻度文案（左中右） | — | 常显 | `AppearanceSettings.vue:440-444` | 字面量 |
| S-appearance-34 | 代码主题 | 分组标签 | — | 常显 | `AppearanceSettings.vue:453` | i18n:appearanceSettings.codeTheme |
| S-appearance-35 | 选择主题（占位） | 下拉触发按钮（整宽） | Vitesse | 常显 | `AppearanceSettings.vue:455-458` | i18n:appearanceSettings.selectTheme |
| S-appearance-36 | 浅色 + 深色 | 下拉分组标题 | — | 下拉展开 | `AppearanceSettings.vue:461` | i18n:appearanceSettings.lightAndDark |
| S-appearance-37 | Vitesse / GitHub / One / Catppuccin / Solarized / Everforest / Min | 下拉选项（成对主题 7 项） | Vitesse | 下拉展开 | `AppearanceSettings.vue:462-464` · 家族 `appearance/code-theme-bg.ts:33-39` | 字面量（主题家族名） |
| S-appearance-38 | 深色 | 下拉分组标题 | — | 下拉展开 | `AppearanceSettings.vue:468` | i18n:appearanceSettings.darkOnly |
| S-appearance-39 | Dracula / Nord / Tokyo Night / Monokai | 下拉选项（仅深色 4 项） | — | 下拉展开 | `AppearanceSettings.vue:469-471` · 家族 `appearance/code-theme-bg.ts:40-43` | 字面量（主题家族名） |
| S-appearance-40 | 主题预览（浅色 / 深色 双栏，或单栏） | 预览面板（StreamDiffs 代码块，随字号缩放） | 跟随所选主题 | 常显 | `AppearanceSettings.vue:477-498` | 运行期（预览代码 `:169-173`） |
| S-appearance-41 | 我的皮肤 | 分组标签（皮肤系统） | — | 桌面端提供 skin API（`hasSkinApi()`） | `SkinSettings.vue:503` | i18n:skin.mySkins |
| S-appearance-42 | 加载骨架（3 张卡） | 占位卡（animate-pulse） | — | 皮肤列表加载中 | `SkinSettings.vue:506-508` | 无文案 |
| S-appearance-43 | 皮肤卡（缩略图 + 名称） | 卡片按钮（点击启用），无缩略图时显示首字母 | — | 有已安装皮肤 | `SkinSettings.vue:515-537` | 运行期（皮肤名） |
| S-appearance-44 | 激活标记（圆点） | 状态指示 | — | 该皮肤为当前皮肤且无错误 | `SkinSettings.vue:544-547` | 无文案 |
| S-appearance-45 | 皮肤错误标记（感叹号） | 状态指示（红色圆点） | — | 该皮肤 schema/图片有错 | `SkinSettings.vue:537-543` | 无文案 |
| S-appearance-46 | 删除皮肤 | 图标按钮（hover 显现，仅非当前皮肤） | — | 该皮肤不是当前皮肤 | `SkinSettings.vue:554-563` | i18n:skin.delete（aria-label） |
| S-appearance-47 | 删除皮肤？ | 确认弹窗标题 | — | 点击删除按钮 | `SkinSettings.vue:568` | i18n:skin.deleteTitle |
| S-appearance-48 | “{name}”及其全部资源将被永久删除。 | 确认弹窗描述 | — | 同上 | `SkinSettings.vue:569-571` | i18n:skin.deleteDescription |
| S-appearance-49 | 取消 | 弹窗按钮 | — | 同上 | `SkinSettings.vue:575` | i18n:common.cancel |
| S-appearance-50 | 删除 / 正在删除… | 弹窗主按钮（destructive） | 删除 | 同上 | `SkinSettings.vue:576-582` | i18n:common.delete · i18n:skin.deleting |
| S-appearance-51 | Windows 98 | 复古外观卡（点击启用） | — | 桌面端提供 skin API | `SkinSettings.vue:589-616` · 名称 `:265-271` | 字面量（`chromeNames`） |
| S-appearance-52 | Windows XP | 复古外观卡 | — | 同上 | `SkinSettings.vue:589-616` | 字面量 |
| S-appearance-53 | Windows Vista | 复古外观卡 | — | 同上 | `SkinSettings.vue:589-616` | 字面量 |
| S-appearance-54 | macOS Big Sur | 复古外观卡 | — | 同上 | `SkinSettings.vue:589-616` | 字面量 |
| S-appearance-55 | OS X Tiger | 复古外观卡 | — | 同上 | `SkinSettings.vue:589-616` | 字面量 |
| S-appearance-56 | 默认 | 卡片（无皮肤 + 默认外观，X 图标） | **选中（未选皮肤且默认外观）** | 同上 | `SkinSettings.vue:620-632` | i18n:skin.default |
| S-appearance-57 | 暂无自定义皮肤。可用图片新建或导入已有皮肤。 | 空态提示 | — | 未安装任何自定义皮肤 | `SkinSettings.vue:635-637` | i18n:skin.emptyHint |
| S-appearance-58 | 用图片新建 | 按钮（outline，加号图标） | — | 同上 | `SkinSettings.vue:640-645` | i18n:skin.newFromImage |
| S-appearance-59 | 打开目录 | 按钮（outline，文件夹图标） | — | 同上 | `SkinSettings.vue:646-649` | i18n:skin.openFolder |
| S-appearance-60 | 删除皮肤失败，请重试。 | 错误文案（role=alert） | — | 删除失败 | `SkinSettings.vue:652-654` | i18n:skin.deleteFailed |
| S-appearance-61 | 当前皮肤设置 | 分组标签 | — | 有生效中的皮肤 | `SkinSettings.vue:658-662` | i18n:skin.activeSettings |
| S-appearance-62 | 窗口背景 | 子分组标签 | — | 同上 | `SkinSettings.vue:667` | i18n:skin.windowBackground |
| S-appearance-63 | 图片路径 / 未设置图片 | 代码块（truncate） | 未设置图片 | 同上 | `SkinSettings.vue:672-674` | i18n:skin.noImage |
| S-appearance-64 | 更换 | 按钮（outline，图片图标，打开文件选择） | — | 同上 | `SkinSettings.vue:675-678` | i18n:skin.change |
| S-appearance-65 | — | 图标按钮（X，移除背景图） | — | 已设置背景图 | `SkinSettings.vue:679-684` | 无文案 |
| S-appearance-66 | 透明度 | 标签 + 百分比数值 | 100% | 同上 | `SkinSettings.vue:692-694` | i18n:skin.opacity |
| S-appearance-67 | — | 滑杆（0–1，步长 0.01，实时预览） | 1 | 同上 | `SkinSettings.vue:696-703` | 无文案 |
| S-appearance-68 | 大小 | 标签 + 下拉 | 覆盖 | 同上 | `SkinSettings.vue:708-719` | i18n:skin.size |
| S-appearance-69 | 覆盖 / 包含 / 自适应 | 下拉选项（3 项） | 覆盖 | 下拉展开 | `SkinSettings.vue:715-717` · 选项 `:94-98` | i18n:skin.sizeCover/sizeContain/sizeAuto |
| S-appearance-70 | 位置 | 标签 + 下拉 | 居中 | 同上 | `SkinSettings.vue:723-735` | i18n:skin.position |
| S-appearance-71 | 居中 / 顶部 / 底部 / 左侧 / 右侧 / 左上 / 右上 / 左下 / 右下 | 下拉选项（9 项） | 居中 | 下拉展开 | `SkinSettings.vue:730-732` · 选项 `:99-109` | i18n:skin.pos* |
| S-appearance-72 | 附着方式 | 标签 + 下拉 | 滚动 | 同上 | `SkinSettings.vue:738-755` | i18n:skin.attachment |
| S-appearance-73 | 滚动 / 固定 | 下拉选项（2 项） | 滚动 | 下拉展开 | `SkinSettings.vue:746-748` · 选项 `:110-113` | i18n:skin.attachScroll/attachFixed |
| S-appearance-74 | ⚠ 可能影响滚动性能 | 警示文案（琥珀色） | — | 附着方式为「固定」 | `SkinSettings.vue:751-753` | i18n:skin.fixedWarning |
| S-appearance-75 | 文字颜色 | 标签（配置卡内） | — | 有生效中的皮肤 | `SkinSettings.vue:762` | i18n:skin.foreground |
| S-appearance-76 | 根据背景调整文字颜色，保证可读性。 | 说明文案 | — | 同上 | `SkinSettings.vue:763` | i18n:skin.foregroundDesc |
| S-appearance-77 | 跟随主题 / 浅色 / 深色 | 下拉（3 个选项） | 跟随主题 | 同上 | `SkinSettings.vue:766-776` · 选项 `:75-82` | i18n:skin.foreground.auto/dark/light |
| S-appearance-78 | 高级 | 折叠按钮（整宽，chevron 旋转） | 折叠 | 同上；皮肤出错时自动展开 | `SkinSettings.vue:780-793` | i18n:skin.advanced |
| S-appearance-79 | 外部打开 | 按钮（outline，文件图标） | — | 「高级」已展开 | `SkinSettings.vue:795-799` | i18n:skin.openExternally |
| S-appearance-80 | 皮肤 schema 错误信息 | 错误文案（destructive） | — | 皮肤 store 报错 | `SkinSettings.vue:806-813` | 运行期（`skin.errors.*` 映射） |
| S-appearance-81 | 打开 skin.json | 按钮（outline） | — | 同上 | `SkinSettings.vue:814-817` | i18n:skin.openSkinJson |
| S-appearance-82 | 重新加载 | 按钮（outline，刷新图标） | — | 同上 | `SkinSettings.vue:818-821` | i18n:skin.reload |
| S-appearance-83 | 皮肤已重新加载 | 成功提示条（主色） | — | 重新加载成功（2 秒后消失） | `SkinSettings.vue:826-831` | i18n:skin.reloaded |

本节元素数：83

本轮自检：主题三选一 + 经典外观提示；程序语言 11 项与切换失败；字体区（字体未加载、枚举失败 + 重试、缺失字体逐条、保存失败、UI/代码两个 picker 与搜索/空态/默认项）；UI 字号 5 档与两端刻度；代码字号数值 + 滑杆 + 三段刻度；代码主题 11 项（浅深成对 7 + 仅深 4）+ 单/双栏预览；皮肤区（加载骨架、皮肤卡与缩略图/首字母、激活点、错误点、删除确认、5 张复古外观卡、默认卡、空态、两个按钮、删除失败、当前皮肤设置 6 个控件、文字颜色 3 项、高级折叠与外部打开、schema 错误 + 两按钮、重载成功提示）。

---

## S-general 通用（`components/settings/GeneralSettings.vue` + `NetworkSettings.vue` + `BrowserDataSettings.vue` + `DataCleanupDialog.vue`）

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-general-01 | 通用 | 页面标题（h2） | — | 常显 | `GeneralSettings.vue:132` | i18n:generalSettings.title |
| S-general-02 | 保持电脑唤醒 | 行标签 | — | 常显 | `GeneralSettings.vue:140` | i18n:generalSettings.keepAwakeLabel |
| S-general-03 | 运行期间阻止系统休眠。屏幕也会保持常亮。 | 行说明 | — | 常显 | `GeneralSettings.vue:141-143` | i18n:generalSettings.keepAwakeDescription |
| S-general-04 | — | 开关 | 关 | 常显 | `GeneralSettings.vue:145` | 运行期（`general.keepAwake`） |
| S-general-05 | 性能日志 | 行标签 | — | 常显 | `GeneralSettings.vue:150` | i18n:generalSettings.performanceLogLabel |
| S-general-06 | 将各进程 CPU/RSS/IO 采样写入 logs/performance/（每秒一行）。tail 该文件即可实时查看。 | 行说明 | — | 常显 | `GeneralSettings.vue:151-153` | i18n:generalSettings.performanceLogDescription |
| S-general-07 | 本次启动由 DIM_PERF 决定采集，该开关不生效。 | 行说明（补充） | — | 环境变量 `DIM_PERF` 固定了采集 | `GeneralSettings.vue:154-156` | i18n:generalSettings.performanceLogPinned |
| S-general-08 | — | 开关 | 跟随本次启动实际值（开发启动默认开） | 常显；被环境变量固定时禁用 | `GeneralSettings.vue:158-163` | 运行期（`app.perf.enabled`） |
| S-general-09 | 登录时自动启动 | 行标签 | — | 常显 | `GeneralSettings.vue:168` | i18n:generalSettings.openAtLoginLabel |
| S-general-10 | — | 开关 | 关 | 常显 | `GeneralSettings.vue:170-174` | 运行期（`general.openAtLogin`） |
| S-general-11 | 会话数据根目录 | 行标签 | — | 常显 | `GeneralSettings.vue:179` | i18n:generalSettings.dimcodePathLabel |
| S-general-12 | 会话记录、日志与缓存都位于该目录下。留空则使用默认根目录。 保存后需要重启 DimAgent 才能生效。 | 行说明（两段拼接） | — | 常显 | `GeneralSettings.vue:180-182` | i18n:generalSettings.dimcodePathDescription · dimcodePathRestartHint |
| S-general-13 | 例如 `<home>/.dimcode`（原文为示例绝对路径，按文档规则占位） | 路径输入框（等宽，占位） | 空（沿用默认根目录） | 配置加载完成后可编辑 | `GeneralSettings.vue:185-191` | i18n:generalSettings.dimcodePathPlaceholder |
| S-general-14 | 重启 App | 按钮（outline，重启中转圈） | 禁用（无改动时） | 配置已加载且路径有改动才可用 | `GeneralSettings.vue:192-201` | i18n:generalSettings.dimcodePathRestart |
| S-general-15 | 保存/重启失败信息 | 错误文案（destructive） | — | 保存或重启失败 | `GeneralSettings.vue:203-205` | 运行期 |
| S-general-16 | 代理 | 分组标签（网络，`embedded` 内嵌模式） | — | 常显（内嵌时无页面标题与描述） | `NetworkSettings.vue:167` | i18n:networkSettings.proxyTitle |
| S-general-17 | 应用于模型请求、登录 API，以及后续新启动的工具进程。 | 说明文案 | — | 常显 | `NetworkSettings.vue:168-170` | i18n:networkSettings.proxyDescription |
| S-general-18 | 系统代理 | 分段按钮（单选，显示器图标） | **选中（默认 system）** | 常显 | `NetworkSettings.vue:173-192` · 选项 `:36-40` | i18n:networkSettings.modeSystem |
| S-general-19 | 无代理 | 分段按钮（拔头图标） | — | 常显 | `NetworkSettings.vue:173-192` · 选项 `:41-45` | i18n:networkSettings.modeNone |
| S-general-20 | 自定义代理 | 分段按钮（滑杆图标） | — | 常显 | `NetworkSettings.vue:173-192` · 选项 `:46-50` | i18n:networkSettings.modeCustom |
| S-general-21 | 自定义代理 | 行标签（大写小字） | — | 模式为「自定义代理」 | `NetworkSettings.vue:194-201` | i18n:networkSettings.customProxyLabel |
| S-general-22 | 代理 URL / 未配置 | 值文本（等宽，truncate，title） | 未配置 | 模式为「自定义代理」 | `NetworkSettings.vue:202-209` | i18n:networkSettings.customProxyNotSet |
| S-general-23 | 编辑 | 按钮（outline，铅笔图标） | — | 模式为「自定义代理」 | `NetworkSettings.vue:211-220` | i18n:networkSettings.editCustomProxy |
| S-general-24 | 加载网络设置失败。 / 保存网络设置失败。 | 错误文案（destructive） | — | 读取或保存失败 | `NetworkSettings.vue:223-225` | i18n:networkSettings.loadFailed · saveFailed |
| S-general-25 | 自定义代理 | 弹窗标题 | — | 点「编辑」或首次选「自定义代理」 | `ProxyConfigDialog.vue:99` | i18n:networkSettings.customProxyDialogTitle |
| S-general-26 | 输入 http 或 https 代理 URL。 | 弹窗描述 | — | 同上 | `ProxyConfigDialog.vue:100-102` | i18n:networkSettings.customProxyDialogDescription |
| S-general-27 | 代理 URL | 输入框标签 | — | 同上 | `ProxyConfigDialog.vue:106` | i18n:networkSettings.customProxyUrlLabel |
| S-general-28 | http://127.0.0.1:7890 | 输入框（占位，回车即保存） | 已存 URL | 同上 | `ProxyConfigDialog.vue:107-114` | 字面量（placeholder） |
| S-general-29 | 代理 URL 不能为空。 / 请输入有效的 http 或 https 代理 URL。 | 校验错误文案 | — | 为空 / 协议或主机不合法 / 保存失败 | `ProxyConfigDialog.vue:115-117` | i18n:networkSettings.customProxyRequired · customProxyInvalid · saveFailed |
| S-general-30 | 取消 | 弹窗按钮（outline） | — | 同上 | `ProxyConfigDialog.vue:121-123` | i18n:common.cancel |
| S-general-31 | 保存 / 保存中... | 弹窗主按钮 | 保存 | 同上 | `ProxyConfigDialog.vue:124-126` | i18n:common.save · i18n:networkSettings.saving |
| S-general-32 | 浏览器数据 | 卡片标题（h3） | — | 浏览器数据 API 可用 | `BrowserDataSettings.vue:224` | i18n:browserData.title |
| S-general-33 | 网站登录状态和存储使用独立的浏览器配置。 | 卡片描述 | — | 同上 | `BrowserDataSettings.vue:225` | i18n:browserData.description |
| S-general-34 | 无法刷新浏览器数据状态，这不表示导入或清理操作失败。 | 错误文案（role=alert） | — | 状态读取失败 | `BrowserDataSettings.vue:228-230` | i18n:browserData.statusFailed |
| S-general-35 | 重试 | 按钮（outline） | — | 状态读取失败 | `BrowserDataSettings.vue:230` | i18n:common.retry |
| S-general-36 | Chrome 数据 | 行标签 | — | 当前平台支持导入 | `BrowserDataSettings.vue:233-238` | i18n:browserData.chromeLabel |
| S-general-37 | 导入… | 按钮（outline） | — | 同上；变更中禁用 | `BrowserDataSettings.vue:239-247` | i18n:browserData.import |
| S-general-38 | 浏览器存储 | 行标签 | — | 浏览器数据 API 可用 | `BrowserDataSettings.vue:249-250` | i18n:browserData.storageLabel |
| S-general-39 | 清理数据… | 按钮（outline） | — | 同上；变更中禁用 | `BrowserDataSettings.vue:251-259` | i18n:browserData.clear |
| S-general-40 | 导入 Chrome Cookie | 导入弹窗标题 | — | 点「导入…」（扫描随即开始） | `BrowserDataSettings.vue:273` | i18n:browserData.importTitle |
| S-general-41 | 此操作会替换内置浏览器中的全部非分区 Cookie……部分网站可能需要重新登录。 | 弹窗警告描述 | — | 可导入数为 0 时不显示 | `BrowserDataSettings.vue:274` | i18n:browserData.replaceWarning |
| S-general-42 | 正在扫描 Chrome 配置… | 加载态（转圈 + 文案） | — | 扫描中 | `BrowserDataSettings.vue:277-281` | i18n:browserData.scanning |
| S-general-43 | Chrome 配置 | 列表小标题 | — | 扫描结束 | `BrowserDataSettings.vue:283` | i18n:browserData.profileLabel |
| S-general-44 | 配置项（单选圆点 + 显示名 + 能力标签） | 单选行（radiogroup 内，每个配置一行） | 首个可导入配置 | 扫描到配置 | `BrowserDataSettings.vue:284-302` | 运行期（`profile.displayName` + 能力标签） |
| S-general-45 | 可导入 / 请先解锁系统密钥服务，然后重试。 / 请先解锁 KWallet，然后重试。 / 不支持此 Chrome Cookie 格式。 / 不支持此 Chrome Cookie 加密方式。 / 不支持当前系统密钥存储后端。 / 没有可用的受支持系统密钥存储。 | 配置行右侧能力标签 | — | 每个配置按能力显示其一 | `BrowserDataSettings.vue:300` · 映射 `:86-97` | i18n:browserData.ready · unlockSecretService · unlockKwallet · reasonSchema · reasonEncryption · reasonSecretStoreUnsupported · reasonSecretStoreUnavailable |
| S-general-46 | 未找到 Google Chrome 配置。 | 空态文案 | — | 扫描无结果且无错误 | `BrowserDataSettings.vue:303` | i18n:browserData.noProfiles |
| S-general-47 | 正在读取所选配置… | 加载文案（转圈） | — | 正在预览所选配置 | `BrowserDataSettings.vue:305-308` | i18n:browserData.previewing |
| S-general-48 | 能力标签（需先解锁…） | 提示行 | — | 所选配置需用户操作 | `BrowserDataSettings.vue:309-317` | 同上能力标签 i18n |
| S-general-49 | 重试 | 按钮（outline，h-7） | — | 所选配置需用户操作 | `BrowserDataSettings.vue:314-316` | i18n:common.retry |
| S-general-50 | 可导入 {importable} · 会话 {session} · 已过期 {expired} · 分区 {partitioned} | 预览计数（灰底块） | — | 预览成功 | `BrowserDataSettings.vue:318-320` | i18n:browserData.counts |
| S-general-51 | 已导入 {imported} · 失败 {failed} · 已跳过 … | 导入结果（勾选/警示图标） | — | 导入已执行 | `BrowserDataSettings.vue:321-331` | i18n:browserData.importResultCounts |
| S-general-52 | 部分 Cookie 未能导入，部分网站可能需要重新登录。 | 结果补充文案（role=status） | — | 导入有失败项 | `BrowserDataSettings.vue:332-334` | i18n:browserData.importPartial |
| S-general-53 | 导入错误文案（如「Cookie 导入失败，原有登录状态已恢复。」） | 错误文案（role=alert） | — | 扫描/预览/导入失败 | `BrowserDataSettings.vue:336-340` | i18n:browserData.*（`:39-57` 错误码映射） |
| S-general-54 | 诊断：{code} · {operationId} · 阶段：{phase}{details} | 诊断行（等宽，可选中） | — | 错误带诊断信息 | `BrowserDataSettings.vue:339` | i18n:browserData.diagnostic |
| S-general-55 | 重试 | 按钮（outline，h-7） | — | 出错时 | `BrowserDataSettings.vue:341-349` | i18n:common.retry |
| S-general-56 | 取消 / 关闭 | 弹窗按钮（outline；有结果时改「关闭」） | 取消 | 导入弹窗打开 | `BrowserDataSettings.vue:354-357` | i18n:common.cancel · common.close |
| S-general-57 | 导入 / 正在导入 Cookie… / 正在关闭浏览器页面… / 正在更新浏览器数据… / 正在验证浏览器数据… / 正在恢复原有登录状态… | 弹窗主按钮（有结果后隐藏） | 导入 | 有预览且未在忙 | `BrowserDataSettings.vue:358-365` | i18n:browserData.confirmImport · importing · phaseQuiescing · phaseApplying · phaseVerifying · phaseRollingBack |
| S-general-58 | 清理浏览器数据 | 清理弹窗标题 | — | 点「清理数据…」 | `BrowserDataSettings.vue:379` | i18n:browserData.clearTitle |
| S-general-59 | 此操作只会清除内置浏览器中的登录状态、Cookie、本地存储、IndexedDB、Service Worker 和缓存。… | 弹窗描述 | — | 同上 | `BrowserDataSettings.vue:380` | i18n:browserData.clearDescription |
| S-general-60 | 内置浏览器数据已清理。 | 成功文案（勾选图标） | — | 清理成功 | `BrowserDataSettings.vue:383-386` | i18n:browserData.clearSuccess |
| S-general-61 | 清理错误文案（如「浏览器数据未能完全清理，请重试。」） | 错误文案（role=alert） | — | 清理失败 | `BrowserDataSettings.vue:387-390` | i18n:browserData.*（错误码映射） |
| S-general-62 | 取消 / 关闭 | 弹窗按钮（outline） | 取消 | 清理弹窗打开 | `BrowserDataSettings.vue:393-395` | i18n:common.cancel · close |
| S-general-63 | 清理浏览器数据 / 正在清理浏览器数据… / 变更阶段文案 | 弹窗主按钮（destructive，成功后隐藏） | 清理浏览器数据 | 清理弹窗打开 | `BrowserDataSettings.vue:396-399` | i18n:browserData.confirmClear · clearing · phase* |
| S-general-64 | 数据清理 | 行标签 | — | 常显 | `GeneralSettings.vue:211` | i18n:dataCleanup.rowLabel |
| S-general-65 | 移除未使用的本地数据并优化存储。 / 会话存储由 App Server 管理，可在此查看其存储用量报告。 | 行说明（按模式二选一） | — | 常显 | `GeneralSettings.vue:212-214` | i18n:dataCleanup.rowDescription · appServerRowDescription |
| S-general-66 | 查看… | 按钮（outline） | — | 常显 | `GeneralSettings.vue:216-222` | i18n:dataCleanup.review |
| S-general-67 | 数据清理 / 存储用量 | 弹窗标题（按模式） | — | 点「查看…」 | `DataCleanupDialog.vue:342` · 标题 `:48-50` | i18n:dataCleanup.title · appServerTitle |
| S-general-68 | 会保留会话、消息和可恢复的文件历史。 / 由拥有会话存储的 App Server 报告。 | 弹窗描述（按模式） | — | 同上 | `DataCleanupDialog.vue:343-345` · `:51-53` | i18n:dataCleanup.safetyDescription · appServerDescription |
| S-general-69 | 正在扫描本地数据… | 加载文案（aria-live） | — | 弹窗打开后扫描中 | `DataCleanupDialog.vue:348-354` | i18n:dataCleanup.scanning |
| S-general-70 | 数据库 | 报告行（App Server 模式） | — | 就绪且为 App Server 计划 | `DataCleanupDialog.vue:362-365` | i18n:dataCleanup.appServerDatabase |
| S-general-71 | WAL | 报告行（App Server 模式） | 不可用 | 同上 | `DataCleanupDialog.vue:366-369` | i18n:dataCleanup.appServerWal |
| S-general-72 | 可回收 | 报告行（App Server 模式） | 不可用 | 同上 | `DataCleanupDialog.vue:370-373` | i18n:dataCleanup.appServerReclaimable |
| S-general-73 | 会话存储由 App Server 管理，此应用不支持跨归属方的扫描和清理。 | 说明文案 | — | 同上 | `DataCleanupDialog.vue:375-377` | i18n:dataCleanup.appServerOwnership |
| S-general-74 | 如需回收可回收空间，请停止守护进程并离线运行 `dimcode daemon vacuum`。 | 说明文案 | — | 同上 | `DataCleanupDialog.vue:378-380` | i18n:dataCleanup.appServerVacuum |
| S-general-75 | 数据库历史 | 报告行（本地模式） | 约 {size} / 不可用 / 运行时计算 | 就绪且为本地计划 | `DataCleanupDialog.vue:385-390` | i18n:dataCleanup.databaseHistory |
| S-general-76 | 未使用的文件历史备份 | 报告行（本地模式） | 同上 | 同上 | `DataCleanupDialog.vue:391-394` | i18n:dataCleanup.unusedBackups |
| S-general-77 | 临时附件和缓存 | 报告行（本地模式，两项求和） | 同上 | 同上 | `DataCleanupDialog.vue:395-398` | i18n:dataCleanup.temporaryData |
| S-general-78 | 过期快照对象 | 报告行（本地模式） | 同上 | 同上 | `DataCleanupDialog.vue:399-402` | i18n:dataCleanup.expiredSnapshots |
| S-general-79 | 优化数据库时可能需要额外的临时磁盘空间。 | 说明文案 | — | 本地计划就绪 | `DataCleanupDialog.vue:404-406` | i18n:dataCleanup.diskSpaceNotice |
| S-general-80 | 正在优化本地存储… | 运行中标题（aria-live） | — | 清理执行中 | `DataCleanupDialog.vue:409-410` | i18n:dataCleanup.optimizing |
| S-general-81 | 文件历史 | 阶段行（状态符号 ✓ / ● / ! / – / ○） | ○ | 清理执行中 | `DataCleanupDialog.vue:412-418` | i18n:dataCleanup.phaseFileHistory |
| S-general-82 | 未使用的本地数据 | 阶段行 | ○ | 同上 | `DataCleanupDialog.vue:419-425` | i18n:dataCleanup.phaseTemporaryData |
| S-general-83 | 本地数据库 | 阶段行 | ○ | 同上 | `DataCleanupDialog.vue:426-432` | i18n:dataCleanup.phaseDatabase |
| S-general-84 | 清理完成前请保持 DimAgent 开启。 | 说明文案 | — | 清理执行中 | `DataCleanupDialog.vue:434` | i18n:dataCleanup.keepOpen |
| S-general-85 | 清理需要独占访问本地数据。 | 阻塞态标题 | — | 扫描发现阻塞项 | `DataCleanupDialog.vue:437-439` | i18n:dataCleanup.blockedTitle |
| S-general-86 | 请先结束正在运行的任务，然后重试。 / 请关闭其他 DimAgent CLI 或 TUI 进程，然后重试。 / 请关闭所有会话弹出窗口，然后重试。 / 无法完成清理，请重试。 | 阻塞/错误态说明（按阻塞原因） | — | 阻塞或执行失败 | `DataCleanupDialog.vue:439` · 文案 `:272-279` · `:206` | i18n:dataCleanup.blockedDescription · storageBusyDescription · popoutBusyDescription · errorDescription |
| S-general-87 | 存储报告不可用 | 不可用态标题 | — | App Server 不支持该能力 | `DataCleanupDialog.vue:442-449` | i18n:dataCleanup.appServerUnavailableTitle |
| S-general-88 | 能力不可用说明 | 不可用态说明 | — | 同上 | `DataCleanupDialog.vue:448` | 运行期 / i18n:dataCleanup.appServerUnavailableDescription |
| S-general-89 | 清理失败 | 错误态标题 | — | 扫描或执行抛错 | `DataCleanupDialog.vue:451-453` | i18n:dataCleanup.errorTitle |
| S-general-90 | 错误详情 | 错误态说明 | — | 同上 | `DataCleanupDialog.vue:453` | 运行期 |
| S-general-91 | 清理完成 / 部分清理已完成 | 完成态标题 | — | 执行结束（有类别未完成则为「部分」） | `DataCleanupDialog.vue:456-459` | i18n:dataCleanup.completeTitle · partialTitle |
| S-general-92 | 已回收 + 大小 | 完成态行 | — | 执行结束 | `DataCleanupDialog.vue:460-464` | i18n:dataCleanup.reclaimed |
| S-general-93 | 数据库 + 前 → 后 | 完成态行 | — | 执行结束 | `DataCleanupDialog.vue:465-472` | i18n:dataCleanup.database |
| S-general-94 | 已移除临时文件 + 数量 | 完成态行 | — | 执行结束 | `DataCleanupDialog.vue:473-476` | i18n:dataCleanup.filesRemoved |
| S-general-95 | 部分文件历史未能清理。 / 部分临时数据未能清理。 / 数据库未完成压缩。 | 完成态提示列表项 | — | 对应类别未完成 | `DataCleanupDialog.vue:478-484` | i18n:dataCleanup.fileHistoryNotFullyCleaned · temporaryDataNotFullyCleaned · databaseNotCompacted |
| S-general-96 | 关闭 | 弹窗按钮（App Server 报告态） | — | 就绪且为 App Server 计划 | `DataCleanupDialog.vue:488-494` | i18n:common.close |
| S-general-97 | 取消 | 弹窗按钮（outline） | — | 本地计划就绪 | `DataCleanupDialog.vue:496` | i18n:common.cancel |
| S-general-98 | 清理 | 弹窗主按钮 | — | 本地计划就绪 | `DataCleanupDialog.vue:497-499` | i18n:dataCleanup.cleanUp |
| S-general-99 | 取消 | 弹窗按钮（outline） | — | 阻塞/错误/不可用态 | `DataCleanupDialog.vue:502` | i18n:common.cancel |
| S-general-100 | 重试 | 弹窗按钮 | — | 阻塞/错误/不可用态 | `DataCleanupDialog.vue:503` | i18n:common.retry |
| S-general-101 | 关闭 | 弹窗按钮 | — | 完成态 | `DataCleanupDialog.vue:505-507` | i18n:common.close |
| S-general-102 | 取消 | 弹窗按钮（outline，兜底） | — | 其它状态 | `DataCleanupDialog.vue:508-510` | i18n:common.cancel |

本节元素数：102

本轮自检：三个开关默认关与持久化回填、性能日志被环境变量固定时的禁用与提示；数据根目录（不可编辑态、无改动时重启按钮禁用、保存/重启错误）；代理三模式（系统/无/自定义）、自定义代理显示与编辑、弹窗必填/格式校验与保存中；浏览器数据（状态刷新失败 + 重试、Chrome 数据行能力标签 7 种、浏览器存储行、导入弹窗扫描/预览/计数/结果/部分失败/错误与诊断/按钮态、清理弹窗成功/失败/按钮态）；数据清理行两套描述与「查看…」，弹窗扫描/App Server 报告三行 + 两段说明/本地报告四行 + 提示/运行中三阶段 + 保持开启/阻塞三因/不可用/错误/完成三行 + 部分提示/五组按钮分支。

---
## S-keybindings 快捷键（`components/settings/KeyboardShortcutsSettings.vue`）

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-keybindings-01 | 快捷键 | 页面标题（h2） | — | 常显 | `KeyboardShortcutsSettings.vue:126` | i18n:keybindingsSettings.title |
| S-keybindings-02 | 加载键盘快捷键失败。 | 错误文案（destructive） | — | 键位读取失败 | `KeyboardShortcutsSettings.vue:129` | i18n:keybindingsSettings.loadFailed |
| S-keybindings-03 | 搜索快捷键 | 搜索输入框（占位，max-w-md） | 空 | 常显 | `KeyboardShortcutsSettings.vue:130` | i18n:keybindingsSettings.searchPlaceholder |
| S-keybindings-04 | 通用 | 分组标题（h3，scope=app） | — | 常显 | `KeyboardShortcutsSettings.vue:133` | i18n:keybindingsSettings.sectionGeneral |
| S-keybindings-05 | 对话 | 分组标题（h3，scope=chat） | — | 常显 | `KeyboardShortcutsSettings.vue:133` | i18n:keybindingsSettings.sectionChat |
| S-keybindings-06 | 打开设置 | 动作行（名称 + 键位按钮 + 恢复默认） | 见默认键位 | 常显（「通用」组） | `KeyboardShortcutsSettings.vue:135-158` · 定义 `shortcuts/registry.ts:26` | i18n:keybindingsSettings.action.openSettings |
| S-keybindings-07 | 新建线程 | 动作行 | 见默认键位 | 常显（「通用」组） | `KeyboardShortcutsSettings.vue:135-158` · 定义 `shortcuts/registry.ts:27` | i18n:keybindingsSettings.action.newThread |
| S-keybindings-08 | 切换侧边栏 | 动作行 | 见默认键位 | 常显（「通用」组） | `KeyboardShortcutsSettings.vue:135-158` · 定义 `shortcuts/registry.ts:28` | i18n:keybindingsSettings.action.toggleSidebar |
| S-keybindings-09 | 搜索对话 | 动作行 | 见默认键位 | 常显（「通用」组） | `KeyboardShortcutsSettings.vue:135-158` · 定义 `shortcuts/registry.ts:29` | i18n:keybindingsSettings.action.searchChats |
| S-keybindings-10 | 聚焦输入框 | 动作行 | 见默认键位 | 常显（「对话」组） | `KeyboardShortcutsSettings.vue:135-158` · 定义 `shortcuts/registry.ts:30` | i18n:keybindingsSettings.action.focusComposer |
| S-keybindings-11 | 切换更改面板 | 动作行 | 见默认键位 | 常显（「对话」组） | `KeyboardShortcutsSettings.vue:135-158` · 定义 `shortcuts/registry.ts:31` | i18n:keybindingsSettings.action.toggleChangesPanel |
| S-keybindings-12 | 切换终端 | 动作行 | 见默认键位 | 常显（「对话」组） | `KeyboardShortcutsSettings.vue:135-158` · 定义 `shortcuts/registry.ts:32` | i18n:keybindingsSettings.action.toggleTerminal |
| S-keybindings-13 | 提交更改 | 动作行 | 见默认键位 | 常显（「对话」组） | `KeyboardShortcutsSettings.vue:135-158` · 定义 `shortcuts/registry.ts:33` | i18n:keybindingsSettings.action.commitChanges |
| S-keybindings-14 | 当前键位（如 `⌘⇧K`）/ 正在录制… | 键位按钮（点击开始录制，再点取消；录制中监听全局 keydown） | 默认键位 | 每行一个 | `KeyboardShortcutsSettings.vue:145-153` | i18n:keybindingsSettings.recording |
| S-keybindings-15 | 恢复默认设置 | 按钮（ghost） | — | 该动作键位与默认值不同 | `KeyboardShortcutsSettings.vue:154-156` | i18n:keybindingsSettings.reset |
| S-keybindings-16 | 快捷键无效。 | 行内错误（destructive） | — | 录到的组合缺修饰键/isComposing/仅修饰键 | `KeyboardShortcutsSettings.vue:138` | i18n:keybindingsSettings.invalid |
| S-keybindings-17 | 此快捷键与其他快捷键冲突。（{冲突动作名}） | 行内错误（destructive） | — | 录到的键位已被其他动作占用 | `KeyboardShortcutsSettings.vue:139-141` | i18n:keybindingsSettings.conflict |
| S-keybindings-18 | 保存键盘快捷键失败。 | 行内错误（destructive） | — | 持久化失败 | `KeyboardShortcutsSettings.vue:142` | i18n:keybindingsSettings.saveFailed |
| S-keybindings-19 | —— | 空态文案（分组内无匹配动作） | —— | 搜索词无匹配 | `KeyboardShortcutsSettings.vue:159` | i18n:common.noDescription |

本节元素数：19

本轮自检：搜索框空/有词/无匹配空态；录制中/取消/无效组合/冲突/保存失败；默认键位与「恢复默认设置」显隐；两个 scope 分组（通用/对话）；加载失败态。另核对：`keybindingsSettings.enableLabel / modeLabel / testCapture / inputMonitoringBanner / requestPermission` 等 12 个键在渲染端无引用（历史遗留，无对应界面元素），未计入。

---

## S-notifications 通知（`components/settings/NotificationsSettings.vue`）

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-notifications-01 | 通知 | 页面标题（h2） | — | 常显 | `NotificationsSettings.vue:59` | i18n:notificationsSettings.title |
| S-notifications-02 | 任务完成或失败时通知 | 行标签 | — | 常显 | `NotificationsSettings.vue:63` | i18n:notificationsSettings.notifyOnCompleteLabel |
| S-notifications-03 | — | 开关 | 关 | 常显 | `NotificationsSettings.vue:64` | 运行期（`notifications.llmComplete`） |
| S-notifications-04 | 通知声音 | 行标签 | — | 常显 | `NotificationsSettings.vue:68` | i18n:notificationsSettings.soundLabel |
| S-notifications-05 | — | 开关 | 关 | 常显 | `NotificationsSettings.vue:69` | 运行期（`notifications.sound`） |
| S-notifications-06 | 需要我回应时通知 | 行标签 | — | 常显 | `NotificationsSettings.vue:73` | i18n:notificationsSettings.notifyOnPermissionLabel |
| S-notifications-07 | — | 开关 | 关 | 常显 | `NotificationsSettings.vue:74` | 运行期（`notifications.permissionRequired`） |
| S-notifications-08 | 加载设置失败 / 保存设置失败 | 错误通知（toast） | — | 读取或保存失败 | `NotificationsSettings.vue:25` · `:34` · `:43` · `:52` | i18n:common.loadFailed · i18n:common.saveFailed |

本节元素数：8

本轮自检：三个开关的默认关态与持久化回填；加载失败与三个保存失败 toast；本节无分组标题、无说明文案、无空态（源码即如此）。

---

## S-pets 宠物（`components/settings/PetsSettings.vue` + `stores/pet.ts`）

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-pets-01 | 宠物 | 页面标题（h2） | — | 常显 | `PetsSettings.vue:247` | i18n:petsSettings.title |
| S-pets-02 | 显示 Pet | 行标签 | — | 常显 | `PetsSettings.vue:253` | i18n:petsSettings.floating |
| S-pets-03 | Pet 会保持在其他窗口上方。 | 行说明 | — | 常显 | `PetsSettings.vue:254-256` | i18n:petsSettings.floatingDescription |
| S-pets-04 | — | 开关（aria-label 显示 Pet） | 关 | 常显；忙/主窗口/快照未加载时禁用 | `PetsSettings.vue:258-264` | 运行期（`petStore.shown`） |
| S-pets-05 | 当前平台不支持显示 Pet。你仍可选择、预览和管理 Pet 文件。 / Pet 显示失败，本次运行已停用；重启应用即可恢复。 / Pet 显示组件不可用，应用安装可能不完整或已损坏；重新安装或更新应用可恢复。 | 提示条（三选一，按不可用原因） | — | 主窗口模式/本进程降级/native addon 缺失 | `PetsSettings.vue:267-273` · 选择逻辑 `stores/pet.ts:79-91` | i18n:petsSettings.platformNote · restartNote · addonNote |
| S-pets-06 | Pet 大小 | 行标签 | — | 非主窗口（`!isMainWindow`） | `PetsSettings.vue:281` | i18n:petsSettings.size |
| S-pets-07 | — | 滑杆（0.5–2，步长 0.1，w-32） | 1 | 非主窗口；忙/快照未加载时禁用 | `PetsSettings.vue:283-293` | 无文案（aria-label Pet 大小） |
| S-pets-08 | 操作错误信息 | 错误块正文（role=alert） | — | 任一 Pet 操作失败 | `PetsSettings.vue:297-303` | i18n:petsSettings.error.* （`:1847-1865`） |
| S-pets-09 | 打开素材位置并备份。修复目录权限后，刷新并从备份重新安装。不要覆盖相同 ID 的 Pet。 | 恢复说明 | — | 删除回滚失败且带回滚 Pet id | `PetsSettings.vue:304-305` | i18n:petsSettings.recoveryInstructions |
| S-pets-10 | 打开保留素材 | 按钮（outline，文件夹图标） | — | 同上 | `PetsSettings.vue:306-316` | i18n:petsSettings.openRetainedMaterials |
| S-pets-11 | 重试 | 按钮（outline） | — | 快照或列表未加载 | `PetsSettings.vue:318-328` | i18n:common.retry |
| S-pets-12 | 选择 Pet | 分组标题（h3） | — | 常显 | `PetsSettings.vue:333` | i18n:petsSettings.chooseTitle |
| S-pets-13 | 已保存的 Pet 不可用：{id} | 提示标题 | — | 已存选择缺失且有快照 | `PetsSettings.vue:344-351` | i18n:petsSettings.missingSelectionTitle |
| S-pets-14 | {code}: {message} | 错误文案（destructive） | — | 同上且带 selectionError | `PetsSettings.vue:353-355` | 运行期 |
| S-pets-15 | 在已保存的 Pet 恢复可用前，将显示 {name}。 | 说明文案 | — | 同上 | `PetsSettings.vue:356-358` | i18n:petsSettings.missingSelectionFallback |
| S-pets-16 | Pet 预览（动画/缩略） | 预览区（PetSettingsPreview，`preview-class` 16 高） | — | Pet 可用且快照存在 | `PetsSettings.vue:360-364` · `:385-392` | 运行期 |
| S-pets-17 | 正在加载... | 加载态（转圈 + 文案） | — | 描述符列表为 null | `PetsSettings.vue:367-374` | i18n:common.loading |
| S-pets-18 | Pet 名称 | 列表行标题（h4） | — | 有 Pet 描述符 | `PetsSettings.vue:395` | 运行期 |
| S-pets-19 | Pet 描述 | 列表行说明 | — | 来源为 custom 且有 description | `PetsSettings.vue:396-398` | 运行期 |
| S-pets-20 | {code}: {message} | 列表行错误（destructive） | — | 该 Pet 不可用（invalid） | `PetsSettings.vue:399-401` | 运行期 |
| S-pets-21 | 选择 / 已选择 | 行按钮（选中项禁用，variant 切换） | 选择 | Pet 可用 | `PetsSettings.vue:404-414` | i18n:petsSettings.select · selected |
| S-pets-22 | 无法渲染这个 Pet 的预览。 | 错误文案（role=alert） | — | 预览组件报 unavailable | `PetsSettings.vue:418-420` | i18n:petsSettings.previewUnavailable |
| S-pets-23 | 自定义 Pets | 分组标题（h3） | — | 常显 | `PetsSettings.vue:425` | i18n:petsSettings.customTitle |
| S-pets-24 | 创建 | 按钮（主按钮，加号图标） | — | 常显；忙时禁用 | `PetsSettings.vue:429-432` | i18n:petsSettings.createPet |
| S-pets-25 | 从文件夹安装 | 按钮（文件夹图标） | — | 常显；忙/未加载时禁用 | `PetsSettings.vue:433-436` | i18n:petsSettings.install |
| S-pets-26 | 刷新 | 按钮（outline，刷新图标） | — | 同上 | `PetsSettings.vue:437-440` | i18n:common.refresh |
| S-pets-27 | 打开 Pets 文件夹 | 按钮（outline） | — | 同上 | `PetsSettings.vue:441-449` | i18n:petsSettings.openFolder |
| S-pets-28 | 尚未安装自定义 Pet。 | 空态（虚线框） | — | 描述符已加载且无 custom Pet | `PetsSettings.vue:452-458` | i18n:petsSettings.customEmpty |
| S-pets-29 | 自定义 Pet 名称 | 行标题（truncate） | — | 有 custom Pet | `PetsSettings.vue:468` | 运行期 |
| S-pets-30 | {code}: {message} | 行错误（destructive） | — | 该 custom Pet 不可用 | `PetsSettings.vue:469-471` | 运行期 |
| S-pets-31 | 选择 / 已选择 | 行按钮 | 选择 | custom Pet 可用 | `PetsSettings.vue:474-481` | i18n:petsSettings.select · selected |
| S-pets-32 | 删除 | 按钮（destructive） | — | 每个 custom Pet；忙时禁用 | `PetsSettings.vue:482-491` | i18n:common.delete |
| S-pets-33 | 删除 {name}？ | 确认弹窗标题 | — | 点「删除」 | `PetsSettings.vue:504-506` | i18n:petsSettings.deleteTitle |
| S-pets-34 | 已保存的选择将切换回内置 Pet。 / 这个自定义 Pet 将从托管的 Pets 文件夹中移除。 | 弹窗描述（二选一） | — | 同上（删的是当前选中项/其他） | `PetsSettings.vue:507-513` | i18n:petsSettings.deleteSelectedDescription · deleteDescription |
| S-pets-35 | 取消 | 弹窗按钮（outline） | — | 同上 | `PetsSettings.vue:516-518` | i18n:common.cancel |
| S-pets-36 | 删除 / 正在删除… | 弹窗主按钮（destructive） | 删除 | 同上 | `PetsSettings.vue:519-526` | i18n:common.delete · i18n:petsSettings.deleting |

本节元素数：36

本轮自检：主窗口模式（平台/重启/组件不可用三种提示 + 隐藏大小滑杆）；浮窗模式的显示开关与大小滑杆禁用态；快照/描述符未加载的加载态与重试；选择缺失回退提示；Pet 不可用行；预览失败；自定义列表空态/行/删除确认（选中项与非选中项两套描述）；操作错误 + 恢复说明 + 打开保留素材。注：源码中「已保存的选择：{id}」与「当前显示：{name}」两行被注释隐藏（`PetsSettings.vue:334-341`），未计入。

---

## S-sandbox 权限（`components/settings/SandboxSettings.vue`，导航名「权限」）

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-sandbox-01 | 权限 | 页面标题（h2） | — | 常显 | `SandboxSettings.vue:287` | i18n:permissionsSettings.title |
| S-sandbox-02 | 进程隔离 | 分组标题（h3，盾牌图标） | — | 常显 | `SandboxSettings.vue:294` | i18n:permissionsSettings.processIsolationTitle |
| S-sandbox-03 | 控制命令是否在 App Server 沙箱中运行，以及沙箱命令是否可以使用网络。 | 说明文案 | — | 常显 | `SandboxSettings.vue:295-297` | i18n:permissionsSettings.processIsolationDescription |
| S-sandbox-04 | 正在加载... | 加载文案 | — | 执行设置加载中 | `SandboxSettings.vue:301-303` | i18n:common.loading |
| S-sandbox-05 | 加载进程隔离设置失败（或接口错误信息） | 错误文案（destructive） | — | 设置为 undefined | `SandboxSettings.vue:304-307` | i18n:permissionsSettings.processIsolationLoadFailed |
| S-sandbox-06 | 刷新 | 按钮（outline） | — | 同上（加载失败） | `SandboxSettings.vue:308-310` | i18n:common.refresh |
| S-sandbox-07 | 沙箱后端不可用：{reason} | 提示条（琥珀色） | reason 缺失时「App Server 未说明原因」 | 后端不可用 | `SandboxSettings.vue:313-321` | i18n:permissionsSettings.processIsolationBackendUnavailable · processIsolationBackendUnknown |
| S-sandbox-08 | 在沙箱中运行命令 | 行标签 | — | 已加载 | `SandboxSettings.vue:325-327` | i18n:permissionsSettings.processIsolationSandbox |
| S-sandbox-09 | 使用宿主沙箱后端运行命令。 | 行说明 | — | 同上 | `SandboxSettings.vue:328-330` | i18n:permissionsSettings.processIsolationSandboxDescription |
| S-sandbox-10 | — | 开关 | 关 | 已加载；后端不可用或保存中禁用 | `SandboxSettings.vue:332-337` | 运行期 |
| S-sandbox-11 | 允许网络访问 | 行标签 | — | 已加载 | `SandboxSettings.vue:342-344` | i18n:permissionsSettings.processIsolationNetwork |
| S-sandbox-12 | 允许沙箱命令访问网络。 | 行说明 | — | 同上 | `SandboxSettings.vue:345-347` | i18n:permissionsSettings.processIsolationNetworkDescription |
| S-sandbox-13 | — | 开关 | 关 | 同上 | `SandboxSettings.vue:349-354` | 运行期 |
| S-sandbox-14 | 保存隔离设置 | 按钮（sm，保存图标） | 禁用（无改动） | 有改动且未保存且可用 | `SandboxSettings.vue:358-366` | i18n:permissionsSettings.processIsolationSave |
| S-sandbox-15 | 进程隔离设置已保存 | 反馈文案 | — | 保存成功后 | `SandboxSettings.vue:367-369` | i18n:permissionsSettings.processIsolationSaved |
| S-sandbox-16 | 保存进程隔离设置失败 | 错误文案（destructive） | — | 保存失败 | `SandboxSettings.vue:370-372` | i18n:permissionsSettings.processIsolationSaveFailed |
| S-sandbox-17 | 新命令使用这些设置；正在运行的进程保持启动时的隔离边界。 | 说明文案 | — | 已加载 | `SandboxSettings.vue:375-377` | i18n:permissionsSettings.processIsolationScope |
| S-sandbox-18 | 权限 | 分组标题（h3，规则区） | — | 常显 | `SandboxSettings.vue:383` | i18n:permissionsSettings.permissionsSection |
| S-sandbox-19 | Exec command 权限 | 标签 | — | 常显 | `SandboxSettings.vue:384` | i18n:permissionsSettings.execCommandsLabel |
| S-sandbox-20 | App Server 命令规则作用于 exec 和 execute 调用。 | 说明文案 | — | 常显 | `SandboxSettings.vue:385-387` | i18n:permissionsSettings.execCommandsDescription |
| S-sandbox-21 | 规则 | 小标题（h4） | — | 常显 | `SandboxSettings.vue:391` | i18n:permissionsSettings.rulesLabel |
| S-sandbox-22 | deny 优先于 allow。exact 匹配完整命令词；prefix 匹配开头的词。 | 规则说明 | — | 常显 | `SandboxSettings.vue:392` | i18n:permissionsSettings.rulesDescription |
| S-sandbox-23 | 命令 | 表头（th） | — | 有规则表 | `SandboxSettings.vue:397` | i18n:permissionsSettings.matcher |
| S-sandbox-24 | 匹配 | 表头（th） | — | 同上 | `SandboxSettings.vue:398` | i18n:permissionsSettings.match |
| S-sandbox-25 | 决策 | 表头（th） | — | 同上 | `SandboxSettings.vue:399` | i18n:permissionsSettings.decision |
| S-sandbox-26 | — | 表头第四列（仅屏幕阅读器「规则」） | — | 同上 | `SandboxSettings.vue:400` | i18n:permissionsSettings.rulesLabel |
| S-sandbox-27 | 命令匹配式 | 行内输入框（等宽，blur 校验） | 空 | 每条规则一行 | `SandboxSettings.vue:406-417` | i18n:permissionsSettings.matcher（aria-label） |
| S-sandbox-28 | 精确 / 前缀 | 行内下拉（2 选项） | 精确 | 每条规则一行 | `SandboxSettings.vue:421-424` | i18n:permissionsSettings.matchExact · matchPrefix |
| S-sandbox-29 | 允许 / 拒绝 | 行内下拉（2 选项） | 允许 | 每条规则一行 | `SandboxSettings.vue:427-430` | i18n:permissionsSettings.actionAllow · actionDeny |
| S-sandbox-30 | 移除 | 按钮（outline，sm） | — | 每条规则一行；保存中禁用 | `SandboxSettings.vue:432` | i18n:permissionsSettings.removeRule |
| S-sandbox-31 | 请输入命令匹配式。 / 命令匹配式不能重复。 | 行内校验错误（role=alert） | — | 匹配式为空/重复 | `SandboxSettings.vue:418` | i18n:permissionsSettings.emptyMatcher · duplicateMatcher |
| S-sandbox-32 | 添加规则 | 按钮（outline，sm） | — | 规则区可用（revision 已取到） | `SandboxSettings.vue:437` | i18n:permissionsSettings.addRule |
| S-sandbox-33 | 保存权限 | 按钮（sm，保存图标） | — | 规则区可用 | `SandboxSettings.vue:441-450` | i18n:permissionsSettings.saveExecPermissions |
| S-sandbox-34 | 已保存。 / 保存失败信息 | 反馈文案（role 随成功/失败切换） | — | 保存权限后 | `SandboxSettings.vue:451-460` | i18n:permissionsSettings.saveExecPermissionsSuccess · saveExecPermissionsFailed |
| S-sandbox-35 | 错误信息 | 错误文案（destructive） | — | 权限读取/写入失败 | `SandboxSettings.vue:463-465` | i18n:permissionsSettings.loadExecPermissionsFailed |
| S-sandbox-36 | 命令权限尚未保存 | 离开确认弹窗标题 | — | 有未保存改动时离开本节 | `SandboxSettings.vue:471` | i18n:permissionsSettings.leaveTitle |
| S-sandbox-37 | 离开前保存更改，或丢弃更改。 | 弹窗描述 | — | 同上 | `SandboxSettings.vue:472` | i18n:permissionsSettings.leaveDescription |
| S-sandbox-38 | 取消 | 弹窗按钮（outline） | — | 同上 | `SandboxSettings.vue:475` | i18n:common.cancel |
| S-sandbox-39 | 丢弃 | 弹窗按钮（outline） | — | 同上 | `SandboxSettings.vue:476` | i18n:permissionsSettings.discard |
| S-sandbox-40 | 保存 | 弹窗主按钮 | — | 同上 | `SandboxSettings.vue:477` | i18n:common.save |

本节元素数：40

本轮自检：进程隔离的加载中/加载失败 + 刷新/后端不可用提示/两个开关的禁用态/保存反馈与错误/范围说明；规则表的空表、多行、行内两条校验错误、移除与添加、保存成功/失败、底部错误；未保存离开的三按钮确认弹窗（取消/丢弃/保存）。注：`permissionsSettings.configPath / defaultAction / inheritDefault / source / builtinSource / userSource / override / resetTitle / resetDescription / effectiveRulesLabel / resetExecPermissions` 等键在本组件无引用，未计入。

---

## S-git Git（`components/settings/GitSettings.vue`）

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-git-01 | Git | 页面标题（h2） | — | 常显 | `GitSettings.vue:108` | i18n:gitSettings.title |
| S-git-02 | 分支前缀 | 标签 | — | 常显 | `GitSettings.vue:114` | i18n:gitSettings.branchPrefixTitle |
| S-git-03 | feature/ | 输入框（占位，max-w-xs，失焦 300ms 防抖保存） | 空 | 常显 | `GitSettings.vue:116-120` | i18n:gitSettings.branchPrefixPlaceholder |
| S-git-04 | 默认远程仓库 | 标签 | — | 常显 | `GitSettings.vue:129` | i18n:gitSettings.defaultRemoteTitle |
| S-git-05 | commit 和 push 的默认远程目标 | 说明文案 | — | 常显 | `GitSettings.vue:130-132` | i18n:gitSettings.defaultRemoteDescription |
| S-git-06 | origin / upstream | 下拉（2 选项，max-w-xs） | origin | 常显 | `GitSettings.vue:134-147` · 选项 `:32-35` | i18n:gitSettings.remoteOrigin · remoteUpstream |
| S-git-07 | 默认基线分支 | 标签 | — | 常显 | `GitSettings.vue:156` | i18n:gitSettings.defaultBaseBranchTitle |
| S-git-08 | 对比/合并的基准分支 | 说明文案 | — | 常显 | `GitSettings.vue:157-159` | i18n:gitSettings.defaultBaseBranchDescription |
| S-git-09 | main / master / develop | 下拉（3 选项） | main | 常显 | `GitSettings.vue:161-174` · 选项 `:37-41` | i18n:gitSettings.branchMain · branchMaster · branchDevelop |
| S-git-10 | 始终强制推送 | 行标签 | — | 常显 | `GitSettings.vue:183` | i18n:gitSettings.alwaysForcePushTitle |
| S-git-11 | 默认使用 force push（需谨慎使用） | 行说明 | — | 常显 | `GitSettings.vue:184-186` | i18n:gitSettings.alwaysForcePushDescription |
| S-git-12 | — | 开关 | 关 | 常显 | `GitSettings.vue:188` | 运行期（`git.forcePush`） |
| S-git-13 | 提交后自动推送 | 行标签 | — | 常显 | `GitSettings.vue:197` | i18n:gitSettings.autoPushTitle |
| S-git-14 | — | 开关 | 关 | 常显 | `GitSettings.vue:199` | 运行期（`git.autoPush`） |
| S-git-15 | Commit 指令补充 | 标签 | — | 常显 | `GitSettings.vue:208` | i18n:gitSettings.commitInstructionsTitle |
| S-git-16 | AI 生成 commit message 时的额外提示 | 说明文案 | — | 常显 | `GitSettings.vue:209-211` | i18n:gitSettings.commitInstructionsDescription |
| S-git-17 | 例如：使用中文，遵循 Conventional Commits 规范... | 多行文本（等宽，min-h 100，防抖保存） | 空 | 常显 | `GitSettings.vue:213-217` | i18n:gitSettings.commitInstructionsPlaceholder |
| S-git-18 | PR 指令补充 | 标签 | — | 常显 | `GitSettings.vue:226` | i18n:gitSettings.prInstructionsTitle |
| S-git-19 | AI 生成 PR description 时的额外提示 | 说明文案 | — | 常显 | `GitSettings.vue:227-229` | i18n:gitSettings.prInstructionsDescription |
| S-git-20 | 例如：包含测试步骤，说明破坏性变更... | 多行文本（等宽，min-h 100） | 空 | 常显 | `GitSettings.vue:231-235` | i18n:gitSettings.prInstructionsPlaceholder |
| S-git-21 | Worktree 根目录 | 标签 | — | 常显 | `GitSettings.vue:244` | i18n:gitSettings.worktreeRootTitle |
| S-git-22 | 选择目录... | 输入框（只读，占位，值来自目录选择） | 空 | 常显 | `GitSettings.vue:247-252` | i18n:gitSettings.selectDirectory |
| S-git-23 | 打开目录 | 图标按钮（outline，size=icon，调系统目录选择） | — | 常显 | `GitSettings.vue:253-255` | 无文案（文件夹图标；保存失败走 toast） |
| S-git-24 | 保存设置失败 / 加载设置失败 / 打开文件失败 | 错误通知（toast） | — | 保存、读取或选目录失败 | `GitSettings.vue:48` · `:75` · `:99` | i18n:common.saveFailed · loadFailed · openFailed |

本节元素数：24

本轮自检：7 个控件的默认值与回填（分支前缀空/远程 origin/基线 main/两个开关关/两个文本域空/目录空）；三个下拉的选项集合全部列出；防抖保存与三处失败 toast。本节无空态、无分组标题。

---

## S-hooks Hooks（`components/settings/HooksSettings.vue` + `HooksAppServerCatalog.vue`）

本节有两个互斥模式：非 App Server 模式走 `HooksSettings.vue` 的「来源卡片 + 详情弹窗」，App Server 模式整体换成 `HooksAppServerCatalog.vue` 的只读目录 + 用户文档编辑器。两套界面各自穷举。

### 模式 A：本地/legacy（`HooksSettings.vue`）

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-hooks-01 | 命令钩子 | 页面标题（h2） | — | 非 App Server 模式 | `HooksSettings.vue:191` | i18n:hooksSettings.title |
| S-hooks-02 | 了解更多 | 按钮（外链图标，打开指南） | — | 常显 | `HooksSettings.vue:194-202` | i18n:hooksSettings.learnMore |
| S-hooks-03 | 刷新 | 图标按钮（aria-label/title，刷新中转圈） | — | 常显 | `HooksSettings.vue:203-212` | i18n:hooksSettings.refresh |
| S-hooks-04 | 加载失败信息 | 错误文案（destructive） | — | 读取钩子失败 | `HooksSettings.vue:216-218` | i18n:common.loadFailed |
| S-hooks-05 | 未配置命令钩子。 | 空态文案 | — | 加载完成且无任何来源 | `HooksSettings.vue:219` | i18n:hooksSettings.empty |
| S-hooks-06 | 来自配置 | 小标题（大写小字） | — | 有 config 来源 | `HooksSettings.vue:223` | i18n:hooksSettings.fromConfig |
| S-hooks-07 | 全局配置（来源标题）+ 配置路径 + {on} 开 · {off} 关 | 来源卡片按钮（设置图标 + 右侧 chevron） | — | 有 config 来源 | `HooksSettings.vue:224-243` | i18n:hooksSettings.globalConfig · statusCount |
| S-hooks-08 | 来自工作区 | 小标题 | — | 有 workspace 来源 | `HooksSettings.vue:247` | i18n:hooksSettings.fromWorkspace |
| S-hooks-09 | {工作区目录名} + 配置路径 + {on} 开 · {off} 关 | 来源卡片按钮（仓库图标） | — | 每个工作区来源一张 | `HooksSettings.vue:248-267` | 运行期 + i18n:hooksSettings.statusCount |
| S-hooks-10 | 来自插件 | 小标题 | — | 有插件来源 | `HooksSettings.vue:271` | i18n:hooksSettings.fromPlugins |
| S-hooks-11 | {插件名} + 配置路径 + {on} 开 · {off} 关 | 来源卡片按钮（插件图标） | — | 每个插件来源一张 | `HooksSettings.vue:272-291` | 运行期 + i18n:hooksSettings.statusCount |
| S-hooks-12 | 来源标题 | 详情弹窗标题（按来源类型给图标） | — | 点击任一张来源卡 | `HooksSettings.vue:296-301` | 运行期 |
| S-hooks-13 | 配置文件路径 | 弹窗描述（等宽，truncate，title） | — | 同上 | `HooksSettings.vue:302` | 运行期 |
| S-hooks-14 | 钩子会在这些生命周期事件上运行 shell 命令。 | 提示条（灰底 + 警示图标） | — | 同上 | `HooksSettings.vue:305-308` | i18n:hooksSettings.runNote |
| S-hooks-15 | 事件名（如 `SessionStart`） | 事件分组标题 | — | 该来源有该事件 | `HooksSettings.vue:317-323` | 运行期 |
| S-hooks-16 | 会话开始时 / 子代理启动时 / 工具运行前 / 工具需要授权时 / 工具运行后 / 压缩历史前 / 压缩历史后 / 用户提交提示时 / 子代理停止时 / 代理回复结束时 | 事件说明（按事件名映射） | — | 同上 | `HooksSettings.vue:321` | i18n:hooksSettings.eventDesc.* |
| S-hooks-17 | Hook {index} | 钩子行标题 | — | 每个钩子一行 | `HooksSettings.vue:332` | i18n:hooksSettings.hookLabel |
| S-hooks-18 | 显示配置文件 | 图标按钮（aria-label/title，在文件管理器中定位） | — | 每个钩子行 | `HooksSettings.vue:333-341` | i18n:hooksSettings.revealConfig |
| S-hooks-19 | 展开/收起详情 | 图标按钮（aria-expanded，chevron 翻转） | 收起 | 每个钩子行 | `HooksSettings.vue:342-353` | i18n:hooksSettings.toggleDetails |
| S-hooks-20 | — | 开关（单钩子启停） | 开 | 每个钩子行；待写入时禁用 | `HooksSettings.vue:354-359` | 运行期 |
| S-hooks-21 | 处理方式 | 详情行（dt/dd） | 命令 | 详情展开 | `HooksSettings.vue:366-367` | i18n:hooksSettings.fieldHandler · handlerCommand |
| S-hooks-22 | 命令 | 详情行（等宽 break-all） | — | 详情展开 | `HooksSettings.vue:369-370` | i18n:hooksSettings.fieldCommand |
| S-hooks-23 | 匹配器 | 详情行（等宽） | — | 详情展开且钩子带 matcher | `HooksSettings.vue:372-375` | i18n:hooksSettings.fieldMatcher |
| S-hooks-24 | 超时 + {seconds}s | 详情行 | — | 详情展开且带 timeoutSeconds | `HooksSettings.vue:377-380` | i18n:hooksSettings.fieldTimeout · timeoutSeconds |
| S-hooks-25 | 状态消息 | 详情行 | — | 详情展开且带 statusMessage | `HooksSettings.vue:382-385` | i18n:hooksSettings.fieldStatusMessage |
| S-hooks-26 | 更新钩子失败，请重试。 | 错误文案（destructive） | — | 钩子开关写入失败 | `HooksSettings.vue:392` | i18n:hooksSettings.toggleFailed |
| S-hooks-27 | 显示配置文件失败 | 错误通知（toast） | — | 定位文件失败 | `HooksSettings.vue:103` | i18n:hooksSettings.revealFailed |

### 模式 B：App Server 目录（`HooksAppServerCatalog.vue`）

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-hooks-28 | 命令钩子 | 页面标题（h2） | — | App Server 模式 | `HooksAppServerCatalog.vue:102` | i18n:hooksSettings.title |
| S-hooks-29 | 了解更多 | 按钮（外链图标） | — | 常显 | `HooksAppServerCatalog.vue:105-113` | i18n:hooksSettings.learnMore |
| S-hooks-30 | 刷新 | 图标按钮（刷新中转圈） | — | 常显 | `HooksAppServerCatalog.vue:114-123` | i18n:hooksSettings.refresh |
| S-hooks-31 | 加载失败信息 / 加载设置失败 | 错误文案（destructive） | — | 目录读取失败 | `HooksAppServerCatalog.vue:127-129` | 运行期 · i18n:common.loadFailed |
| S-hooks-32 | 正在加载... | 加载文案 | — | 首次加载中 | `HooksAppServerCatalog.vue:130-132` | i18n:common.loading |
| S-hooks-33 | 用户文档已关闭所有钩子。 | 提示条（警示图标） | — | 目录 `disableAllHooks` 为真 | `HooksAppServerCatalog.vue:134-141` | i18n:hooksSettings.disableAllHooksNotice |
| S-hooks-34 | 钩子会在这些生命周期事件上运行 shell 命令。 | 提示条 | — | 常显（目录模式下） | `HooksAppServerCatalog.vue:143-146` | i18n:hooksSettings.runNote |
| S-hooks-35 | 来源 | 小标题 | — | 常显 | `HooksAppServerCatalog.vue:149` | i18n:hooksSettings.sources |
| S-hooks-36 | 用户 / 项目 / {插件来源 id} | 来源卡标题 | — | 每个来源一张卡 | `HooksAppServerCatalog.vue:161` · 映射 `:37-41` | i18n:hooksSettings.userSource · projectSource |
| S-hooks-37 | 正常 / 缺失 / 无效 | 来源状态徽标 | 正常 | 同上 | `HooksAppServerCatalog.vue:162-165` · 映射 `:43-47` | i18n:hooksSettings.stateOk · stateMissing · stateInvalid |
| S-hooks-38 | 已启用 / 已禁用 | 来源启用徽标（启用为绿色描边） | — | 同上 | `HooksAppServerCatalog.vue:166-170` | i18n:hooksSettings.enabled · disabled |
| S-hooks-39 | 只读 | 徽标（锁图标） | — | 插件来源（`plugin:` 前缀） | `HooksAppServerCatalog.vue:171-178` | i18n:hooksSettings.readOnly |
| S-hooks-40 | 配置文件路径 | 等宽文案（truncate，title） | — | 来源带 path | `HooksAppServerCatalog.vue:180-182` | 运行期 |
| S-hooks-41 | 来源错误信息 | 错误文案（destructive） | — | 来源带 message | `HooksAppServerCatalog.vue:183` | 运行期 |
| S-hooks-42 | 编辑 | 按钮（outline，铅笔图标） | — | 仅 user 来源 | `HooksAppServerCatalog.vue:186-195` | i18n:hooksSettings.editUserDocument |
| S-hooks-43 | 禁用项目钩子 / 启用项目钩子 | 行内说明（随状态切换） | — | 仅 project 来源 | `HooksAppServerCatalog.vue:196-199` | i18n:hooksSettings.disableProject · enableProject |
| S-hooks-44 | — | 开关（项目钩子总开关） | 开 | 仅 project 来源；写入中禁用 | `HooksAppServerCatalog.vue:200-205` | 运行期 |
| S-hooks-45 | 钩子配置在读取后已更改，未写入任何内容；草稿已保留。 / 钩子修改被拒绝，未写入任何内容；草稿已保留。 / 修改已保存，但无法重新读取钩子目录。 | 写入失败文案（destructive） | — | 冲突 / 拒绝 / 回读失败 | `HooksAppServerCatalog.vue:209-215` · 映射 `:57-61` | i18n:hooksSettings.conflictError · refusedError · readBackError |
| S-hooks-46 | 条目 | 小标题 | — | 常显 | `HooksAppServerCatalog.vue:219` | i18n:hooksSettings.entries |
| S-hooks-47 | 来源名 + 已启用 / 已禁用 | 条目卡头 | — | 每个来源一张卡 | `HooksAppServerCatalog.vue:226-230` | 运行期 + i18n:hooksSettings.enabled · disabled |
| S-hooks-48 | 没有条目 | 空态文案 | — | 该来源无条目 | `HooksAppServerCatalog.vue:231-233` | i18n:hooksSettings.noEntries |
| S-hooks-49 | 事件名 | 条目行标题 | — | 每个条目一行 | `HooksAppServerCatalog.vue:241` | 运行期 |
| S-hooks-50 | 运行 / 不支持 / 无效 | 条目状态徽标 | 运行 | 每个条目 | `HooksAppServerCatalog.vue:242-245` · 映射 `:49-53` | i18n:hooksSettings.entryStateOk · entryStateUnsupported · entryStateInvalid |
| S-hooks-51 | 异步 | 徽标 | — | 条目 `async` 为真 | `HooksAppServerCatalog.vue:246-248` | i18n:hooksSettings.async |
| S-hooks-52 | 命令 | 等宽文案（break-all） | — | 条目带 command | `HooksAppServerCatalog.vue:250` | 运行期 |
| S-hooks-53 | 匹配器: {matcher} | 元信息 | — | 条目带 matcher | `HooksAppServerCatalog.vue:252` | i18n:hooksSettings.fieldMatcher |
| S-hooks-54 | 超时: {seconds}s | 元信息 | — | 每个条目 | `HooksAppServerCatalog.vue:253` | i18n:hooksSettings.fieldTimeout · timeoutSeconds |
| S-hooks-55 | 失败时: {onFailure} | 元信息 | — | 每个条目 | `HooksAppServerCatalog.vue:254` | i18n:hooksSettings.fieldOnFailure |
| S-hooks-56 | 保存失败文案（同上三种映射） | 错误文案（destructive） | — | 保存失败且无草稿 | `HooksAppServerCatalog.vue:260-266` | i18n:hooksSettings.conflictError · refusedError |
| S-hooks-57 | 用户修订号 {revision} | 小字文案 | — | 目录已加载 | `HooksAppServerCatalog.vue:269` | i18n:hooksSettings.userRevision |
| S-hooks-58 | 项目修订号 {revision} | 小字文案 | — | 目录已加载 | `HooksAppServerCatalog.vue:270` | i18n:hooksSettings.projectsRevision |
| S-hooks-59 | 用户钩子文档 | 编辑弹窗标题 | — | 点「编辑」 | `HooksAppServerCatalog.vue:276` | i18n:hooksSettings.editUserDocumentTitle |
| S-hooks-60 | 编辑用户钩子文档。保存时会按本页读取的修订号替换。 | 弹窗描述 | — | 同上 | `HooksAppServerCatalog.vue:277` | i18n:hooksSettings.editUserDocumentDescription |
| S-hooks-61 | — | 多行文本（等宽，min-h 280，不校验拼写，max-h 420 滚动） | 当前用户文档 | 同上 | `HooksAppServerCatalog.vue:279-287` | 无文案 |
| S-hooks-62 | 文档必须是 JSON 对象。 | 校验错误（destructive） | — | 编辑内容非 JSON 对象 | `HooksAppServerCatalog.vue:288-290` | i18n:hooksSettings.invalidDocument |
| S-hooks-63 | 保存失败文案 | 弹窗内错误（destructive） | — | 保存失败 | `HooksAppServerCatalog.vue:291-297` | i18n:hooksSettings.conflictError · refusedError |
| S-hooks-64 | 取消 | 弹窗按钮（outline） | — | 弹窗打开 | `HooksAppServerCatalog.vue:299-301` | i18n:common.cancel |
| S-hooks-65 | 保存 | 弹窗主按钮 | — | 弹窗打开；保存中禁用 | `HooksAppServerCatalog.vue:302-304` | i18n:common.save |

本节元素数：65

本轮自检：模式 A 三组来源（配置/工作区/插件）有与无、来源卡与详情弹窗、详情展开前后、四条可选详情行、开关写入失败、定位文件失败、空态与加载失败；模式 B 的 disableAllHooks 提示、来源三态徽标 + 启用徽标 + 只读、来源错误、写入失败三种映射、条目空态/三态徽标/异步徽标、修订号、编辑弹窗的 JSON 校验与保存失败、两个按钮。

---

## S-pluginSettings 插件设置（`components/settings/PluginSettings.vue` + `PluginSettingsField.vue`）

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-pluginSettings-01 | 插件设置 | 页面标题（h1） | — | 常显 | `PluginSettings.vue:234` | i18n:pluginSettings.title |
| S-pluginSettings-02 | 配置 App Server 插件声明的设置。 | 说明文案 | — | 常显 | `PluginSettings.vue:235` | i18n:pluginSettings.description |
| S-pluginSettings-03 | 正在加载插件设置... | 加载文案 | — | 首次加载 | `PluginSettings.vue:238` | i18n:pluginSettings.loading |
| S-pluginSettings-04 | 当前 App Server 不提供插件设置。 | 不可用文案 | — | `unavailable` 为真 | `PluginSettings.vue:240-242` | i18n:pluginSettings.unavailable |
| S-pluginSettings-05 | 加载错误信息 | 错误文案（destructive） | — | 加载失败 | `PluginSettings.vue:244-245` | 运行期 |
| S-pluginSettings-06 | 重试 | 按钮（outline） | — | 加载失败 | `PluginSettings.vue:246` | i18n:pluginSettings.retry |
| S-pluginSettings-07 | 没有已安装插件提供设置。 | 空态文案 | — | 来源列表为空 | `PluginSettings.vue:249-251` | i18n:pluginSettings.empty |
| S-pluginSettings-08 | 用户 | 作用域按钮（secondary 表示选中） | 选中 | 存在项目级来源时显示作用域切换 | `PluginSettings.vue:255-262` | i18n:pluginSettings.scopeUser |
| S-pluginSettings-09 | 项目 | 作用域按钮 | 未选 | 同上 | `PluginSettings.vue:263-270` | i18n:pluginSettings.scopeProject |
| S-pluginSettings-10 | 选择将这些设置保存到用户范围，还是保存到已授权的项目。 | 作用域提示 | — | 同上 | `PluginSettings.vue:271` | i18n:pluginSettings.scopeHint |
| S-pluginSettings-11 | 正在加载插件设置... | 项目根加载文案 | — | 项目作用域且根目录加载中 | `PluginSettings.vue:279-285` | i18n:pluginSettings.loading |
| S-pluginSettings-12 | 项目根加载错误信息 | 错误文案（destructive） | — | 项目根加载失败 | `PluginSettings.vue:286-291` | 运行期 |
| S-pluginSettings-13 | 重试 | 按钮（outline） | — | 同上 | `PluginSettings.vue:292` | i18n:pluginSettings.retry |
| S-pluginSettings-14 | 当前没有已授权的项目根目录。请在项目中打开会话以完成授权。 | 空态文案 | — | 无已授权项目根 | `PluginSettings.vue:294-299` | i18n:pluginSettings.projectNoRoots |
| S-pluginSettings-15 | 重新加载 | 按钮（outline） | — | 同上 | `PluginSettings.vue:300` | i18n:pluginSettings.reload |
| S-pluginSettings-16 | 项目根目录 | 标签（for 下拉） | — | 有已授权项目根 | `PluginSettings.vue:304` | i18n:pluginSettings.projectLabel |
| S-pluginSettings-17 | {cwd}（每个已授权项目一项） | 原生下拉（h-8，min-w-60） | 第一个项目根 | 同上 | `PluginSettings.vue:305-314` | 运行期 |
| S-pluginSettings-18 | 重新加载 | 按钮（outline） | — | 同上 | `PluginSettings.vue:315-317` | i18n:pluginSettings.reload |
| S-pluginSettings-19 | 设置将从 {cwd} 读取并写回；该根目录由已打开的项目会话授权。 | 说明文案 | — | 同上 | `PluginSettings.vue:319-321` | i18n:pluginSettings.projectHint |
| S-pluginSettings-20 | 没有已安装插件声明项目级设置。 | 空态文案 | — | 项目作用域且无项目级来源 | `PluginSettings.vue:325-331` | i18n:pluginSettings.projectEmpty |
| S-pluginSettings-21 | 尚未选择已授权的项目根目录。 | 空态文案 | — | 项目作用域且未选根 | `PluginSettings.vue:333-339` | i18n:pluginSettings.projectNoSelection |
| S-pluginSettings-22 | 没有已安装插件提供设置。 | 空态文案（当前作用域无可见来源） | — | 可见状态为空 | `PluginSettings.vue:341-347` | i18n:pluginSettings.empty |
| S-pluginSettings-23 | 插件名称 | 来源卡标题（h2） | — | 每个插件来源一张卡 | `PluginSettings.vue:359` | 运行期（schema title） |
| S-pluginSettings-24 | 插件描述 | 来源卡说明 | — | schema 带 description | `PluginSettings.vue:360` | 运行期 |
| S-pluginSettings-25 | {source} 插件 id | 等宽小字 | — | 每个来源卡 | `PluginSettings.vue:361` | 运行期 |
| S-pluginSettings-26 | 正在加载插件设置... | 卡内加载文案 | — | 该来源加载中 | `PluginSettings.vue:365` | i18n:pluginSettings.loading |
| S-pluginSettings-27 | 来源加载错误信息 | 错误文案（destructive） | — | 该来源加载失败 | `PluginSettings.vue:367-368` | 运行期 |
| S-pluginSettings-28 | 重试 | 按钮（outline） | — | 同上 | `PluginSettings.vue:369` | i18n:pluginSettings.retry |
| S-pluginSettings-29 | 设置已在别处更改。请重新加载后再保存。 | 冲突提示（琥珀色） | — | 修订号冲突 | `PluginSettings.vue:373-377` | i18n:pluginSettings.conflict |
| S-pluginSettings-30 | 重新加载 | 按钮（outline） | — | 同上 | `PluginSettings.vue:375-377` | i18n:pluginSettings.reload |
| S-pluginSettings-31 | {字段标题}（+ 必填时红色 `*`） | 字段标签（schema title，缺省回落字段名） | — | 每个 schema 字段一行 | `PluginSettingsField.vue:131` · `:32-34` | 运行期 |
| S-pluginSettings-32 | {字段描述} | 字段说明（schema description） | — | schema 带 description | `PluginSettingsField.vue:132` · `:35` | 运行期 |
| S-pluginSettings-33 | — | 布尔字段开关 | 取文档值 | schema 类型 boolean | `PluginSettingsField.vue:199-208` | 运行期 |
| S-pluginSettings-34 | — | 数字输入（带 min/max） | 取文档值 | schema 类型 integer/number | `PluginSettingsField.vue:211-221` | 运行期 |
| S-pluginSettings-35 | {enum 选项} | 原生下拉（枚举值） | 取文档值 | 字符串字段带 enum 且非密文 | `PluginSettingsField.vue:227-235` | 运行期 |
| S-pluginSettings-36 | — | 文本/密码输入（密文用 password） | 取文档值 | 字符串字段 | `PluginSettingsField.vue:236-243` | 运行期 |
| S-pluginSettings-37 | 已存储；留空表示保留 | 密文字段占位文案 | 占位 | 密文字段且已存值 | `PluginSettingsField.vue:240` | i18n:pluginSettings.storedSecret |
| S-pluginSettings-38 | 清除 | 按钮（ghost） | — | 密文字段且已存值 | `PluginSettingsField.vue:244-253` | i18n:pluginSettings.clearSecret |
| S-pluginSettings-39 | 此值包含密钥字段，当前表单无法安全编辑。 | 提示文案 | — | 字段（或对象/数组/回退值）含密钥 | `PluginSettingsField.vue:133-135` · `:172-178` · `:260-266` | i18n:pluginSettings.unsupportedSecret |
| S-pluginSettings-40 | — | 对象分组卡 + 启用开关 | 未配置 | schema 为 object / nullable-object | `PluginSettingsField.vue:138-152` | 运行期 |
| S-pluginSettings-41 | — | 数组项输入 + 移除 | 取文档值 | schema 为原始类型数组 | `PluginSettingsField.vue:179-192` | 运行期 |
| S-pluginSettings-42 | 移除 | 按钮（ghost，sm） | — | 每个数组项 | `PluginSettingsField.vue:188-190` | i18n:pluginSettings.removeItem |
| S-pluginSettings-43 | 添加 | 按钮（outline，sm） | — | 数组字段 | `PluginSettingsField.vue:193-195` | i18n:pluginSettings.addItem |
| S-pluginSettings-44 | — | JSON 文本域（5 行，等宽，blur 解析） | 取文档值 | 其它类型（回退） | `PluginSettingsField.vue:268-274` | 运行期 |
| S-pluginSettings-45 | 请输入有效的 JSON。 | 校验错误（destructive） | — | JSON 解析失败 | `PluginSettingsField.vue:275` | i18n:pluginSettings.invalidJson |
| S-pluginSettings-46 | 保存 / 正在保存... | 表单主按钮（submit） | 保存 | 每个来源卡 | `PluginSettings.vue:394-396` | i18n:pluginSettings.save · saving |
| S-pluginSettings-47 | 重新加载 | 按钮（outline） | — | 每个来源卡 | `PluginSettings.vue:397-399` | i18n:pluginSettings.reload |
| S-pluginSettings-48 | 已保存 | 反馈文案 | — | 该来源保存成功 | `PluginSettings.vue:400-402` | i18n:pluginSettings.saved |
| S-pluginSettings-49 | 保存错误信息 | 错误文案（destructive） | — | 该来源保存失败 | `PluginSettings.vue:403-405` | 运行期 |

本节元素数：49

本轮自检：顶层 loading/unavailable/loadError+重试/empty 四态；作用域切换（有项目来源才出现）；项目根 4 态（加载中/错误/未授权空/下拉+重新加载+提示）；项目作用域的三种空态；来源卡的加载中/加载失败/冲突/正常表单；字段六种形态（boolean/integer/enum/string/密文/对象/数组/JSON 回退）与三类密钥不可编辑提示；每个来源卡的保存中/已保存/保存错误。

---

## S-externalAgents 外部 Agent（`components/settings/ExternalAgentsSettings.vue`）

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-externalAgents-01 | 外部 Agent | 页面标题（h2） | — | 常显 | `ExternalAgentsSettings.vue:115` | i18n:externalAgentsSettings.title |
| S-externalAgents-02 | 将外部 agent CLI 作为子 agent 启用，主 agent 可委派任务给它们。 | 说明文案 | — | 常显 | `ExternalAgentsSettings.vue:116-118` | i18n:externalAgentsSettings.description |
| S-externalAgents-03 | 正在探测外部 Agent… | 加载文案（aria-live） | — | 首次探测中 | `ExternalAgentsSettings.vue:121-123` | i18n:externalAgentsSettings.loading |
| S-externalAgents-04 | 加载外部 Agent 失败。 | 错误文案（destructive） | — | 探测失败 | `ExternalAgentsSettings.vue:125-126` | i18n:externalAgentsSettings.loadFailed |
| S-externalAgents-05 | 重试 | 按钮（outline） | — | 同上 | `ExternalAgentsSettings.vue:127-129` | i18n:common.retry |
| S-externalAgents-06 | App Server 模式下不可用 | 不可用标题 | — | `unavailable` 为真 | `ExternalAgentsSettings.vue:137` | i18n:externalAgentsSettings.unavailableTitle |
| S-externalAgents-07 | App Server 尚未实现外部代理。此模式下 Desktop 无法检测或配置它们。 | 不可用说明 | — | 同上 | `ExternalAgentsSettings.vue:138-140` | i18n:externalAgentsSettings.unavailableDescription |
| S-externalAgents-08 | {agentType} | Agent 行标签（Label，for 开关） | — | 每个探测到的 agent 一行 | `ExternalAgentsSettings.vue:152-154` | 运行期 |
| S-externalAgents-09 | {version} | 版本文本（truncate，title） | — | agent 带 version | `ExternalAgentsSettings.vue:157-159` | 运行期 |
| S-externalAgents-10 | 已安装 | 状态文本 | — | 已安装但无版本号 | `ExternalAgentsSettings.vue:160` | i18n:externalAgentsSettings.installed |
| S-externalAgents-11 | 未识别版本 | 警示文本（琥珀色 + tooltip 解释） | — | 有 execPath 但无版本 | `ExternalAgentsSettings.vue:161-173` | i18n:externalAgentsSettings.versionUnknown |
| S-externalAgents-12 | 未能从该文件识别版本号，这个 Agent 可能不可用。 | tooltip 内容 | — | 同上 | `ExternalAgentsSettings.vue:172` | i18n:externalAgentsSettings.versionUnknownHint |
| S-externalAgents-13 | 未安装 | 状态文本 | — | 无 execPath | `ExternalAgentsSettings.vue:174` | i18n:externalAgentsSettings.notInstalled |
| S-externalAgents-14 | {execPath} | 等宽路径文本（truncate + tooltip 全路径） | — | 有 execPath | `ExternalAgentsSettings.vue:177-188` | 运行期 |
| S-externalAgents-15 | 行错误信息（更新失败等） | 错误文案（role=alert） | — | 该行开关或路径更新失败 | `ExternalAgentsSettings.vue:191-193` | i18n:externalAgentsSettings.toggleFailed · pathUpdateFailed |
| S-externalAgents-16 | 编辑 {agentType} 的可执行文件 / 选择 {agentType} 的可执行文件 | 图标按钮（铅笔/加号，aria-label + tooltip） | — | 每个 agent 行；写入中禁用 | `ExternalAgentsSettings.vue:196-220` | i18n:externalAgentsSettings.editExecutableLabel · selectExecutableLabel |
| S-externalAgents-17 | 编辑 / 选择可执行文件 | tooltip 文案 | — | 悬停上者 | `ExternalAgentsSettings.vue:215-219` | i18n:externalAgentsSettings.editExecutable · selectExecutable |
| S-externalAgents-18 | — | 开关（启用该外部 agent） | 关 | 每个 agent 行；无路径且未启用时禁用 | `ExternalAgentsSettings.vue:221-227` | 运行期 |
| S-externalAgents-19 | 更改将在新对话中生效。 | 底部说明 | — | 非 unavailable | `ExternalAgentsSettings.vue:233-235` | i18n:externalAgentsSettings.restartHint |

本节元素数：19

本轮自检：加载中 / 加载失败 + 重试 / App Server 不可用 / 正常列表四态；每行三种版本态（有版本 / 已安装无版本 / 未识别版本含 tooltip / 未安装）；行错误；编辑与选择两种图标与 tooltip；开关禁用条件；底部重启提示。

---

## S-archived 归档会话（`components/settings/ArchivedThreadsSettings.vue`）

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-archived-01 | 归档会话 | 页面标题（h2） | — | 常显 | `ArchivedThreadsSettings.vue:262` | i18n:archivedThreads.title |
| S-archived-02 | 全部删除 | 按钮（outline，红色描边，垃圾桶图标） | 禁用（无会话时） | 常显；加载中/无会话/删除中禁用 | `ArchivedThreadsSettings.vue:266-274` | i18n:archivedThreads.deleteAll |
| S-archived-03 | 删除所有归档会话 | 确认弹窗标题 | — | 点「全部删除」 | `ArchivedThreadsSettings.vue:278` | i18n:archivedThreads.deleteAllTitle |
| S-archived-04 | 确定要永久删除 {count} 条归档会话吗？此操作不可撤销。 | 确认弹窗描述 | — | 同上 | `ArchivedThreadsSettings.vue:279-281` | i18n:archivedThreads.deleteAllDescription |
| S-archived-05 | 取消 | 弹窗按钮 | — | 同上 | `ArchivedThreadsSettings.vue:284` | i18n:common.cancel |
| S-archived-06 | 删除 | 弹窗主按钮（destructive） | — | 同上；删除中禁用 | `ArchivedThreadsSettings.vue:285-287` | i18n:common.delete |
| S-archived-07 | 加载骨架（3 张卡） | 骨架（animate-pulse，无文案） | — | 列表加载中 | `ArchivedThreadsSettings.vue:294-310` | 无文案 |
| S-archived-08 | 搜索归档会话 | 搜索输入框（带放大镜，sm:max-w-xs） | 空 | 有会话 | `ArchivedThreadsSettings.vue:318-325` | i18n:archivedThreads.searchPlaceholder |
| S-archived-09 | 全部项目 / 无项目 / {项目路径} | 项目筛选下拉（带图层/文件夹图标） | 全部项目 | 有会话 | `ArchivedThreadsSettings.vue:326-349` | i18n:archivedThreads.allProjects · noProject |
| S-archived-10 | {项目分组名} | 分组行标题（文件夹图标） | — | 每个项目分组 | `ArchivedThreadsSettings.vue:359` | 运行期 |
| S-archived-11 | {count} 条会话 | 分组计数 | — | 同上 | `ArchivedThreadsSettings.vue:362-364` | i18n:archivedThreads.chatCount |
| S-archived-12 | 更多操作 | 图标按钮（省略号，打开下拉菜单） | — | 同上；删除中禁用 | `ArchivedThreadsSettings.vue:365-375` | i18n:archivedThreads.projectMoreActions |
| S-archived-13 | 删除项目内全部会话 | 下拉菜单项（destructive，红色） | — | 展开「更多操作」 | `ArchivedThreadsSettings.vue:380-386` | i18n:archivedThreads.deleteAllInProject |
| S-archived-14 | 删除项目内所有会话 | 确认弹窗标题 | — | 点该菜单项 | `ArchivedThreadsSettings.vue:390` | i18n:archivedThreads.deleteProjectTitle |
| S-archived-15 | 确定要永久删除 "{project}" 中的 {count} 条归档会话吗？此操作不可撤销。 | 确认弹窗描述 | — | 同上 | `ArchivedThreadsSettings.vue:391-393` | i18n:archivedThreads.deleteProjectDescription |
| S-archived-16 | 取消 | 弹窗按钮 | — | 同上 | `ArchivedThreadsSettings.vue:396` | i18n:common.cancel |
| S-archived-17 | 删除 | 弹窗主按钮（destructive） | — | 同上 | `ArchivedThreadsSettings.vue:397-399` | i18n:common.delete |
| S-archived-18 | {thread.title} | 会话行标题（truncate） | — | 每个归档会话一行 | `ArchivedThreadsSettings.vue:418-420` | 运行期 |
| S-archived-19 | 已关闭 | 徽标（10px 描边） | — | 该会话 `closed` 为真 | `ArchivedThreadsSettings.vue:421-426` | i18n:archivedThreads.closed |
| S-archived-20 | {date} 归档 | 时间文案 | — | 每个会话行 | `ArchivedThreadsSettings.vue:428-430` | i18n:archivedThreads.archivedAt |
| S-archived-21 | 恢复 | 按钮（outline，sm，回退箭头） | — | 每个会话行；已关闭时禁用（title 解释） | `ArchivedThreadsSettings.vue:434-444` | i18n:archivedThreads.restore |
| S-archived-22 | 删除 | 按钮（outline，红色描边） | — | 每个会话行 | `ArchivedThreadsSettings.vue:446-456` | i18n:common.delete |
| S-archived-23 | 删除归档 | 确认弹窗标题 | — | 点行内「删除」 | `ArchivedThreadsSettings.vue:459` | i18n:archivedThreads.deleteArchiveTitle |
| S-archived-24 | 确定要永久删除 "{title}" 吗？此操作不可撤销。 | 确认弹窗描述 | — | 同上 | `ArchivedThreadsSettings.vue:460-462` | i18n:archivedThreads.deleteArchiveDescription |
| S-archived-25 | 取消 | 弹窗按钮 | — | 同上 | `ArchivedThreadsSettings.vue:465` | i18n:common.cancel |
| S-archived-26 | 删除 | 弹窗主按钮（destructive） | — | 同上 | `ArchivedThreadsSettings.vue:466-468` | i18n:common.delete |
| S-archived-27 | 暂无归档会话 | 空态标题（大圆图标 + 虚线框） | — | 无任何归档会话 | `ArchivedThreadsSettings.vue:482-487` | i18n:archivedThreads.emptyTitle |
| S-archived-28 | 没有匹配的会话 | 空态标题 | — | 有会话但被搜索/筛选过滤为空 | `ArchivedThreadsSettings.vue:489` | i18n:archivedThreads.noResultsTitle |
| S-archived-29 | 请尝试其他搜索词或项目筛选条件 | 空态说明 | — | 同上 | `ArchivedThreadsSettings.vue:490` | i18n:archivedThreads.noResultsDescription |
| S-archived-30 | 已恢复会话 / 恢复会话失败 / 加载归档会话失败 | 通知（toast） | — | 恢复成功或失败、列表加载失败 | 文案映射见 i18n | i18n:archivedThreads.restoreSuccess · restoreFailed · loadFailed |
| S-archived-31 | 已删除会话 / 删除会话失败 / 删除失败，会话可能正在使用中 | 通知（toast） | — | 单条删除成功或失败 | 文案映射见 i18n | i18n:archivedThreads.deleteSuccess · deleteFailed · deleteInUse |
| S-archived-32 | 已删除所有归档会话 / 删除完成，但有 {count} 条失败。部分会话可能仍在使用中。 | 通知（toast） | — | 批量删除完成（全成功/部分失败） | 文案映射见 i18n | i18n:archivedThreads.deleteAllSuccess · deleteAllPartialFailed |

本节元素数：32

本轮自检：加载骨架 / 搜索 + 项目筛选（全部/无项目/各项目）/ 分组行与「更多操作」菜单 / 三层删除确认（全部、项目内、单条）/ 会话行「已关闭」徽标与恢复禁用 / 两种空态（无数据、无匹配）/ 四类 toast。

---

## S-about 关于（`components/settings/AboutSettings.vue` + `app/App.vue` 的安装确认）

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-about-01 | 软件更新 | 分组标签 | — | 常显 | `AboutSettings.vue:130` | i18n:aboutSettings.softwareUpdateTitle |
| S-about-02 | 当前版本: v{version} | 文本 | 取运行版本，未知显示 `-` | 常显 | `AboutSettings.vue:134-137` | i18n:aboutSettings.currentVersion |
| S-about-03 | 新版本: v{version} | 文本（主色） | — | 已知可用更新版本 | `AboutSettings.vue:138-141` | i18n:aboutSettings.newVersion |
| S-about-04 | 已是最新版本 | 状态行（绿色勾） | — | phase=idle | `AboutSettings.vue:146-149` | i18n:aboutSettings.upToDate |
| S-about-05 | 检查中... | 状态行（转圈） | — | phase=checking | `AboutSettings.vue:150-153` | i18n:aboutSettings.checkingUpdates |
| S-about-06 | 下载中... | 状态行（下载图标） | — | phase=downloading | `AboutSettings.vue:154-157` | i18n:aboutSettings.downloadingUpdate |
| S-about-07 | 有新版本可用（或上游 message） | 状态行（主色/错误色） | 有新版本可用 | phase=available | `AboutSettings.vue:158-165` | i18n:aboutSettings.updateAvailable |
| S-about-08 | 更新已下载 | 状态行（绿色勾） | — | phase=downloaded | `AboutSettings.vue:166-172` | i18n:aboutSettings.updateDownloaded |
| S-about-09 | 正在准备… | 状态行（转圈） | — | phase=preparing | `AboutSettings.vue:173-176` | i18n:aboutSettings.preparingUpdate |
| S-about-10 | 检查更新失败（或上游 message） | 状态行（destructive） | 检查更新失败 | phase=error | `AboutSettings.vue:177-179` | i18n:aboutSettings.checkUpdateFailed |
| S-about-11 | 自动更新已禁用 | 状态行（次要色） | — | 其它 phase（自动更新关闭） | `AboutSettings.vue:180-182` | i18n:aboutSettings.autoUpdateDisabled |
| S-about-12 | 检查更新 / 重试 | 按钮（outline，sm） | 检查更新 | phase=idle 或 error | `AboutSettings.vue:185-194` | i18n:common.checkForUpdates · common.retry |
| S-about-13 | 检查中... | 按钮（禁用） | — | phase=checking | `AboutSettings.vue:195-204` | i18n:aboutSettings.checkingUpdates |
| S-about-14 | {n}% | 按钮（禁用，显示下载进度） | — | phase=downloading | `AboutSettings.vue:205-214` | 运行期 |
| S-about-15 | 下载新版本 / 重试更新 / 更新 | 按钮（outline） | 下载新版本 | phase=available | `AboutSettings.vue:215-228` | i18n:aboutSettings.downloadNewVersion · retryUpdate · i18n:sidebar.update |
| S-about-16 | 重启安装 | 按钮（主按钮） | — | phase=downloaded | `AboutSettings.vue:229-237` | i18n:aboutSettings.restartAndInstall |
| S-about-17 | 正在准备… | 按钮（禁用） | — | phase=preparing | `AboutSettings.vue:238-246` | i18n:aboutSettings.preparingUpdate |
| S-about-18 | 更新状态错误信息 | 错误文案（destructive） | — | 状态查询失败 | `AboutSettings.vue:249-251` | 运行期 |
| S-about-19 | 公告 | 分组标签 | — | 常显 | `AboutSettings.vue:257` | i18n:aboutSettings.announcementsTitle |
| S-about-20 | 刷新 | 图标按钮（aria-label/title，刷新中转圈） | — | 常显；刷新中禁用 | `AboutSettings.vue:259-271` | i18n:common.refresh |
| S-about-21 | {公告标题} | 公告项按钮（可展开，整宽） | 收起 | 有公告（最多 3 条） | `AboutSettings.vue:280-290` | 运行期 |
| S-about-22 | {发布日期} | 时间文本（datetime 属性） | — | 每条公告 | `AboutSettings.vue:291-293` | 运行期 |
| S-about-23 | {公告摘要} | 说明文案 | — | 公告带 summary | `AboutSettings.vue:295-297` | 运行期 |
| S-about-24 | {公告正文} | Markdown 渲染区（role=region，展开态） | — | 展开该条公告 | `AboutSettings.vue:307-328` | 运行期 |
| S-about-25 | 暂无公告 | 空态（虚线框居中） | — | 公告已加载且为空 | `AboutSettings.vue:332-337` | i18n:aboutSettings.noAnnouncements |
| S-about-26 | 查看更多公告 | 按钮（outline，sm，外链图标） | — | 常显 | `AboutSettings.vue:339-344` | i18n:aboutSettings.viewAnnouncements |
| S-about-27 | 暂不可用 | 分组标签 | — | 常显 | `AboutSettings.vue:349` | i18n:aboutSettings.wipTitle |
| S-about-28 | 以下能力仍在 WIP 跟踪中；入口保持可见但禁用，直到 App Server 路径完成。 | 说明文案 | — | 常显 | `AboutSettings.vue:350-352` | i18n:aboutSettings.wipDescription |
| S-about-29 | 文件历史与回退 / 删除分支 / 重命名分支 / 分支独立模型 / 跨客户端归档 / MCP 交互询问 / 全局用量账本 / 远程附件读取 | WIP 能力项标签（8 项，一项一行） | — | 常显（WIP 列表固定） | `AboutSettings.vue:355-366` · 文案 `wip-capability-copy.ts:16-47` | i18n:wipCapabilities.*.label |
| S-about-30 | {对应原因的原文} | WIP 项原因文案 | — | 每项 | `AboutSettings.vue:363` | i18n:wipCapabilities.*.reason |
| S-about-31 | WIP | 徽标（secondary） | — | 每项 | `AboutSettings.vue:365` | 字面量 |
| S-about-32 | 安装更新 | 确认弹窗标题（全局挂载） | — | 点「重启安装」 | `app/App.vue:321` | i18n:aboutSettings.installUpdateTitle |
| S-about-33 | DimAgent 会先停止正在运行的任务并保存会话，然后重启安装更新。 | 确认弹窗描述 | — | 同上 | i18n 定义 | i18n:aboutSettings.installUpdateDescription |
| S-about-34 | Update state debugger | DEV 调试区标题 | — | `import.meta.env.DEV` | `AboutSettings.vue:378` | 字面量（英文） |
| S-about-35 | Development only. Cycles local update UI states without downloading or installing. | DEV 调试区说明 | — | 同上 | `AboutSettings.vue:379-381` | 字面量（英文） |
| S-about-36 | {当前 phase / 下载百分比} | DEV 调试状态码（等宽） | — | 同上 | `AboutSettings.vue:385` | 运行期 |
| S-about-37 | Next state | DEV 调试按钮（outline，sm） | — | 同上 | `AboutSettings.vue:386-388` | 字面量（英文） |

本节元素数：37

本轮自检：更新卡 9 种 phase 状态行 + 6 种按钮态（含下载百分比、preparing 双禁用）+ 状态错误；公告卡有数据/展开/收起/空态/刷新中；WIP 列表 8 项全列（label + reason + 徽标）；安装确认弹窗（标题 + 描述，挂在 App.vue）；DEV-only 调试区（标题/说明/状态码/按钮）。

---

## S-runtime 运行环境（`components/settings/RuntimeSettings.vue` + `FeatureRuntimeSettings.vue`）

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-runtime-01 | 运行环境 | 页面标题（h1） | — | 常显 | `RuntimeSettings.vue:78` | i18n:runtimeSettings.title |
| S-runtime-02 | 由 App Server RuntimeManager 管理；Desktop 不下载、不保留第二份运行时。 | 说明文案 | — | 常显 | `RuntimeSettings.vue:79` | i18n:runtimeSettings.ownerDescription |
| S-runtime-03 | 刷新 | 按钮（outline，sm，刷新图标） | — | 常显；加载中禁用/转圈 | `RuntimeSettings.vue:81-91` | i18n:runtimeSettings.refresh |
| S-runtime-04 | 正在加载... | 加载文案（role=status） | — | 条目未加载且无错误 | `RuntimeSettings.vue:94-96` | i18n:common.loading |
| S-runtime-05 | 无法加载运行环境设置。（或接口错误信息） | 错误文案（destructive，role=status） | 无法加载运行环境设置。 | 条目为 null 且有错误 | `RuntimeSettings.vue:98-104` | i18n:runtimeSettings.loadFailed |
| S-runtime-06 | 重试 | 按钮（outline，sm） | — | 同上 | `RuntimeSettings.vue:105-107` | i18n:common.retry |
| S-runtime-07 | 实时运行环境更新不可用；请使用刷新重新读取 owner 条目。 | 提示文案（role=status） | — | `watchAvailable === false` | `RuntimeSettings.vue:111-118` | i18n:runtimeSettings.watchUnavailable |
| S-runtime-08 | Node.js / Python / uv | 运行时卡标题（h2，三张卡固定顺序） | — | 已加载 | `RuntimeSettings.vue:120-132` · 名称表 `:14-18` | 字面量 |
| S-runtime-09 | 未启用 / 安装中 / 失败 / 就绪 / 启动中 / 已停止 | 状态标签（就绪绿色，其余琥珀色，带勾/警示/转圈图标） | — | 每张卡 | `RuntimeSettings.vue:133-154` · `:26-29` | i18n:runtimeSettings.status.off · acquiring · failed · ready · starting · down |
| S-runtime-10 | 系统自带 | 来源按钮（ghost，选中带描边，aria-pressed） | 视 owner 值 | 每张卡；pending/acquiring 时禁用 | `RuntimeSettings.vue:158-169` | i18n:runtimeSettings.sourceSystem |
| S-runtime-11 | 下载 | 来源按钮（同上） | 视 owner 值 | 同上 | `RuntimeSettings.vue:170-181` | i18n:runtimeSettings.sourceDownload |
| S-runtime-12 | 启动时获取 | 开关 + 文案（同一 label） | 视 owner 值 | 每张卡；pending/acquiring 时禁用 | `RuntimeSettings.vue:185-194` | i18n:runtimeSettings.auto |
| S-runtime-13 | 安装 / 下载 / 重新安装 | 按钮（sm，下载图标，进行中转圈） | 安装 | 每张卡；仅 off/failed/down 可点 | `RuntimeSettings.vue:195-209` | i18n:runtimeSettings.install · downloadManaged · reinstall |
| S-runtime-14 | {version} | 版本号（tabular-nums） | — | 卡已有 path 且带 version | `RuntimeSettings.vue:214` | 运行期 |
| S-runtime-15 | {path} | 路径文本（等宽，可点击复制，role=button + tabindex） | — | 卡已有 path | `RuntimeSettings.vue:215-227` | i18n:runtimeSettings.copyPath（aria-label） |
| S-runtime-16 | 复制路径 | 图标按钮（aria-label/title） | — | 卡已有 path | `RuntimeSettings.vue:229-239` | i18n:runtimeSettings.copyPath |
| S-runtime-17 | 路径已复制 / 无法复制路径 | 通知（toast） | — | 复制成功/失败 | `RuntimeSettings.vue:62` · `:65` | i18n:runtimeSettings.pathCopied · pathCopyFailed |
| S-runtime-18 | {失败原因原文} | 卡片内错误（role=alert，警示图标） | — | 该运行时有 reason | `RuntimeSettings.vue:242-250` | 运行期 |
| S-runtime-19 | App Server 未返回 {runtime}。 | 缺卡文案 | — | owner 未返回该运行时 | `RuntimeSettings.vue:252-254` | i18n:runtimeSettings.entryMissing |
| S-runtime-20 | {全局错误原文} | 底部错误（role=alert） | — | store error 非空 | `RuntimeSettings.vue:257-265` | 运行期 |
| S-runtime-21 | 功能组件 | 分组标题（h2） | — | 常显 | `FeatureRuntimeSettings.vue:52` | i18n:featureRuntimes.title |
| S-runtime-22 | {加载错误原文} | 错误文案（role=alert） | — | 功能组件快照加载失败 | `FeatureRuntimeSettings.vue:53-55` | 运行期 |
| S-runtime-23 | 重试 | 按钮（outline，sm） | — | 同上 | `FeatureRuntimeSettings.vue:55` | i18n:common.retry |
| S-runtime-24 | 正在加载... | 加载文案（role=status） | — | 快照未就绪 | `FeatureRuntimeSettings.vue:57` | i18n:common.loading |
| S-runtime-25 | OfficeCLI | 组件卡标题（h3） | — | 快照就绪 | `FeatureRuntimeSettings.vue:61` | 字面量 |
| S-runtime-26 | 未安装 / 正在下载安装 / 正在验证可用性 / 已安装并验证 / 失败 / 不支持此平台 | 状态文本（role=status，活动时转圈） | — | 每张组件卡 | `FeatureRuntimeSettings.vue:62-65` | i18n:featureRuntimes.status.* |
| S-runtime-27 | 创建、读取和编辑 Word、Excel、PowerPoint 文件。 | 组件说明 | — | 同上 | `FeatureRuntimeSettings.vue:67` | i18n:featureRuntimes.officeDescription |
| S-runtime-28 | 配套版本 {version} · 下载 {size} / 配套版本 {version} · 随应用内置 | 版本行 | — | 有描述符 | `FeatureRuntimeSettings.vue:68` · `:34-40` | i18n:featureRuntimes.versionDownload · versionBundled |
| S-runtime-29 | {已下载} / {总大小} | 下载进度文本（tabular-nums） | — | 下载中且已知已下载字节 | `FeatureRuntimeSettings.vue:69-72` | 运行期 |
| S-runtime-30 | — | 进度条（Progress） | — | 同上 | `FeatureRuntimeSettings.vue:71` | 无文案 |
| S-runtime-31 | {错误原文} | 错误文案（role=alert） | — | 操作失败或描述符带 error | `FeatureRuntimeSettings.vue:73` | 运行期 |
| S-runtime-32 | 已安装并验证。新任务立即生效；已有任务需重启应用，已有终端需重新打开。 | 完成提示 | — | 组件状态 ready | `FeatureRuntimeSettings.vue:74` | i18n:featureRuntimes.activationHint |
| S-runtime-33 | 下载安装 | 按钮（sm，主按钮） | — | 非内置组件且未 ready；活动/unsupported 时禁用 | `FeatureRuntimeSettings.vue:76` | i18n:featureRuntimes.install |
| S-runtime-34 | 验证可用性 | 按钮（outline，sm） | — | 内置组件或已 ready | `FeatureRuntimeSettings.vue:77` | i18n:featureRuntimes.verify |

本节元素数：34

本轮自检：运行环境三张卡（Node.js/Python/uv）各自 6 种状态、来源二选一、启动时获取、安装按钮三文案、路径复制（点击与按钮两种入口 + 两条 toast）、卡片错误与「App Server 未返回」缺卡、watch 不可用提示、加载/错误/重试、全局错误；功能组件卡的加载失败/加载中/错误/进度/完成提示/两个按钮文案。

---

## S-modality 多模态（`components/settings/ModalityDefaultsSetting.vue` + `ModalityModelPicker.vue` + `ModalityTaskPanel.vue` + `ModalityRealtimePanel.vue`）

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-modality-01 | 多模态 | 页面标题（h1） | — | 常显 | `ModalityDefaultsSetting.vue:200` | i18n:configurationSettings.modality |
| S-modality-02 | — | 加载转圈（标题右侧，无文案） | — | 设置加载中 | `ModalityDefaultsSetting.vue:202` | 无文案 |
| S-modality-03 | {不可用原因原文} | 提示条（灰底 + 警示图标） | — | `unavailableReason` 非空 | `ModalityDefaultsSetting.vue:205-212` | 运行期 |
| S-modality-04 | {错误原文} | 错误条（destructive） | — | `errorText` 非空 | `ModalityDefaultsSetting.vue:214-220` | 运行期 |
| S-modality-05 | 图像 / 视频 / 语音与音频 / 创作与工具 / 实时 | 分类页签（5 个，整宽等分，选中下划线） | 图像 | 宽屏（`!isCompact`） | `ModalityDefaultsSetting.vue:237-260` · 分组 `:43-70` | i18n:configurationSettings.modalityGroup.* |
| S-modality-06 | {已配置} / {总数} | 页签计数（tabular-nums） | — | 每个页签/下拉项 | `ModalityDefaultsSetting.vue:253-255` | 运行期 |
| S-modality-07 | {分类名} + {已配置}/{总数} | 分类下拉（紧凑模式，整宽） | 当前分类 | 窄屏（`isCompact`） | `ModalityDefaultsSetting.vue:223-235` | i18n:configurationSettings.modalityGroup.* |
| S-modality-08 | {分类名} | 面板标题（h2） | — | 该分类面板 | `ModalityDefaultsSetting.vue:270-271` | i18n:configurationSettings.modalityGroup.* |
| S-modality-09 | {已配置} / {总数} | 面板右上计数 | — | 同上 | `ModalityDefaultsSetting.vue:272-274` | 运行期 |
| S-modality-10 | 图像生成 / 图像编辑 / 图像增强 / 图像理解 / 视频生成 / 视频编辑 / 视频增强 / 语音合成 / 语音转写 / 音乐生成 / 音频生成 / 素材生成 / 动作生成 / OCR（文字识别）/ 实时语音 / 实时转写 / 实时数字人（+ legacy 能力） | 能力行标签 | — | 每个能力一行 | `ModalityDefaultsSetting.vue:278-286` | i18n:configurationSettings.modalityCapability.* |
| S-modality-11 | 选择提供商 / 选择模型 / {提供商} / {模型} / 自动 | 模型选择器触发按钮（整宽，左图标 + 文本 + chevron；未选时为占位；非默认时挂「自动」徽标；不可用时文本转 destructive） | 未设置 | 每个能力行 | `ModalityModelPicker.vue:210-244` | i18n:configurationSettings.modalityUnsetProvider · modalityUnsetModel · modalityAutomatic |
| S-modality-12 | 搜索提供商或模型 | 选择器内搜索框（占位，h-8） | 空 | 点开选择器 | `ModalityModelPicker.vue:249-254` | i18n:configurationSettings.modalitySearch |
| S-modality-13 | 提供商 | 左列小标题 | — | 点开选择器 | `ModalityModelPicker.vue:259-261` | i18n:configurationSettings.modalityProviders |
| S-modality-14 | {提供商名} | 提供商选项（图标 + 名称 + 选中勾） | 当前提供商 | 点开选择器 | `ModalityModelPicker.vue:263-278` | 运行期 |
| S-modality-15 | 无可用提供商 | 左列空态 | — | 搜索无匹配提供商 | `ModalityModelPicker.vue:279-284` | i18n:configurationSettings.modalityNoProviders |
| S-modality-16 | 模型 | 右列小标题 | — | 点开选择器 | `ModalityModelPicker.vue:289-291` | i18n:configurationSettings.modalityModels |
| S-modality-17 | {模型名}（+ 不可用原因细节） | 模型选项（两行，不可用置灰禁用，选中勾） | 当前模型 | 点开选择器 | `ModalityModelPicker.vue:293-316` | 运行期 |
| S-modality-18 | 无可用模型 | 右列空态 | — | 该提供商无可用模型 | `ModalityModelPicker.vue:317-322` | i18n:configurationSettings.modalityNoModels |
| S-modality-19 | 未设置 | 文字按钮（link 样式，清除已选） | — | 该能力有默认值（`operation.default`） | `ModalityModelPicker.vue:328-338` | i18n:configurationSettings.unset |
| S-modality-20 | {操作不可用原因原文} | 能力行说明（小字） | — | `operationReason` 有值 | `ModalityDefaultsSetting.vue:297-303` | 运行期 |
| S-modality-21 | 任务 | 分组标题（h2） | — | 常显（多模态页底部） | `ModalityTaskPanel.vue:309` | i18n:configurationSettings.modalityTask.title |
| S-modality-22 | 提交并跟踪由 App Server 拥有的 Modality 任务。 | 说明文案 | — | 常显 | `ModalityTaskPanel.vue:310` | i18n:configurationSettings.modalityTask.description |
| S-modality-23 | App Server 未声明 Modality 任务方法。 | 不可用提示（灰底 + 警示图标） | — | 任务能力不支持 | `ModalityTaskPanel.vue:313-320` | i18n:configurationSettings.modalityTask.unavailable |
| S-modality-24 | 提交任务 | 小标题（h3） | — | 常显（不可用时控件禁用） | `ModalityTaskPanel.vue:323` | i18n:configurationSettings.modalityTask.submitTitle |
| S-modality-25 | 类别 | 标签 + 输入框 | 空 | 同上 | `ModalityTaskPanel.vue:325-333` | i18n:configurationSettings.modalityTask.category |
| S-modality-26 | 操作 | 标签 + 输入框 | 空 | 同上 | `ModalityTaskPanel.vue:334-342` | i18n:configurationSettings.modalityTask.operation |
| S-modality-27 | 参数（JSON 对象） | 标签 + 多行文本（等宽，4 行） | 空 | 同上 | `ModalityTaskPanel.vue:344-353` | i18n:configurationSettings.modalityTask.params |
| S-modality-28 | 文件（JSON 数组） | 标签 + 多行文本（等宽，3 行） | 空 | 同上 | `ModalityTaskPanel.vue:354-363` | i18n:configurationSettings.modalityTask.files |
| S-modality-29 | 提供商（可选） | 标签 + 输入框 | 空 | 同上 | `ModalityTaskPanel.vue:365-373` | i18n:configurationSettings.modalityTask.modelProvider |
| S-modality-30 | 模型（可选） | 标签 + 输入框 | 空 | 同上 | `ModalityTaskPanel.vue:374-382` | i18n:configurationSettings.modalityTask.modelModel |
| S-modality-31 | 客户端任务 ID（可选） | 标签 + 输入框 | 空 | 同上 | `ModalityTaskPanel.vue:383-391` | i18n:configurationSettings.modalityTask.clientTaskId |
| S-modality-32 | 提交任务 / 提交中… | 按钮（主按钮，提交中转圈） | 提交任务 | 任务能力可用 | `ModalityTaskPanel.vue:394-403` | i18n:configurationSettings.modalityTask.submit · submitting |
| S-modality-33 | {校验/提交错误原文} | 错误文案（destructive） | — | JSON 校验失败或提交失败 | `ModalityTaskPanel.vue:405-407` | i18n:configurationSettings.modalityTask.invalidJson · paramsObject · filesArray · modelPair |
| S-modality-34 | 所有者任务 | 小标题（h3） | — | 常显 | `ModalityTaskPanel.vue:412` | i18n:configurationSettings.modalityTask.listTitle |
| S-modality-35 | 全部状态 / {各任务状态} | 状态筛选下拉（原生 select，切换即刷新） | 全部状态 | 常显（不可用时禁用） | `ModalityTaskPanel.vue:414-423` | i18n:configurationSettings.modalityTask.allStates |
| S-modality-36 | 刷新 | 按钮（刷新图标 + 文案，加载中转圈） | — | 同上 | `ModalityTaskPanel.vue:424-433` | i18n:configurationSettings.modalityTask.listRefresh |
| S-modality-37 | {列表错误原文} | 错误文案（destructive） | — | 列表加载失败 | `ModalityTaskPanel.vue:437` | 运行期 |
| S-modality-38 | 没有任务。 | 空态文案 | — | 列表为空 | `ModalityTaskPanel.vue:438-440` | i18n:configurationSettings.modalityTask.listEmpty |
| S-modality-39 | {task_id} · {category}.{operation} · {state} · {created_at} | 任务行按钮（点击选中查看详情） | — | 每个任务一行 | `ModalityTaskPanel.vue:442-453` | 运行期 |
| S-modality-40 | 加载更多 | 按钮（outline） | — | 还有更多页 | `ModalityTaskPanel.vue:456-465` | i18n:configurationSettings.modalityTask.listLoadMore |
| S-modality-41 | 任务详情 | 小标题（h3） | — | 已选中任务 | `ModalityTaskPanel.vue:469` | i18n:configurationSettings.modalityTask.details |
| S-modality-42 | 状态 | 详情行（dt/dd） | — | 已选中任务 | `ModalityTaskPanel.vue:471` | i18n:configurationSettings.modalityTask.state |
| S-modality-43 | 修订 | 详情行 | — | 同上 | `ModalityTaskPanel.vue:472` | i18n:configurationSettings.modalityTask.revision |
| S-modality-44 | 更新时间 | 详情行 | — | 同上 | `ModalityTaskPanel.vue:473` | i18n:configurationSettings.modalityTask.updatedAt |
| S-modality-45 | 进度 | 详情行 | — | 任务带 progress.message | `ModalityTaskPanel.vue:474` | i18n:configurationSettings.modalityTask.progress |
| S-modality-46 | {code}: {message} | 任务错误（destructive） | — | 任务带 error | `ModalityTaskPanel.vue:476-478` | 运行期 |
| S-modality-47 | {动作错误原文} | 错误文案（destructive） | — | 刷新/等待/取消失败 | `ModalityTaskPanel.vue:479` | 运行期 |
| S-modality-48 | 等待在超时后返回；任务仍为 {state}。 | 等待提示 | — | 等待超时 | `ModalityTaskPanel.vue:480` | i18n:configurationSettings.modalityTask.waitTimedOut |
| S-modality-49 | 刷新状态 | 按钮（outline） | — | 已选中任务 | `ModalityTaskPanel.vue:483-490` | i18n:configurationSettings.modalityTask.status |
| S-modality-50 | 等待超时（毫秒） | 数字输入（0–600000，w-28） | 视脚本默认 | 同上 | `ModalityTaskPanel.vue:491-501` | i18n:configurationSettings.modalityTask.waitTimeout |
| S-modality-51 | 等待 / 等待中… | 按钮（outline，等待中转圈） | 等待 | 同上 | `ModalityTaskPanel.vue:502-511` | i18n:configurationSettings.modalityTask.wait · waiting |
| S-modality-52 | 取消任务 / 取消中… | 按钮（outline，取消中转圈） | 取消任务 | 同上 | `ModalityTaskPanel.vue:512-521` | i18n:configurationSettings.modalityTask.cancel · cancelling |
| S-modality-53 | 结果文件 | 小标题（h4） | — | 任务有 result | `ModalityTaskPanel.vue:525` | i18n:configurationSettings.modalityTask.resultFiles |
| S-modality-54 | {name} · {media_type} · {bytes} | 结果文件行（等宽） | — | 每个结果文件一行 | `ModalityTaskPanel.vue:527-528` | 运行期 |
| S-modality-55 | 下载 / 下载中… | 按钮（outline，下载中转圈） | 下载 | 每个结果文件；下载中禁用 | `ModalityTaskPanel.vue:529-539` | i18n:configurationSettings.modalityTask.download · downloading |
| S-modality-56 | {下载错误原文} | 错误文案（destructive） | — | 下载失败 | `ModalityTaskPanel.vue:542` | 运行期 |
| S-modality-57 | 选择一个任务以查看详情。 | 提示文案 | — | 任务能力可用但未选中 | `ModalityTaskPanel.vue:545` | i18n:configurationSettings.modalityTask.noSelection |
| S-modality-58 | 实时会话 | 分组标题（h2） | — | 常显（多模态页底部） | `ModalityRealtimePanel.vue:223` | i18n:configurationSettings.modalityRealtime.title |
| S-modality-59 | 通过 App Server 打开并驱动 Modality 实时会话。 | 说明文案 | — | 常显 | `ModalityRealtimePanel.vue:224` | i18n:configurationSettings.modalityRealtime.description |
| S-modality-60 | App Server 未声明 Modality 实时方法。 | 不可用提示 | — | 实时能力不支持 | `ModalityRealtimePanel.vue:227-234` | i18n:configurationSettings.modalityRealtime.unavailable |
| S-modality-61 | 此构建中没有可用的实时操作。 | 提示（灰底） | — | 支持但操作列表为空 | `ModalityRealtimePanel.vue:236-243` | i18n:configurationSettings.modalityRealtime.noOperations |
| S-modality-62 | 此构建不支持音频帧：Desktop 没有安全的音频采集或编码器。文本帧仍可使用。 | 提示（灰底，常显） | — | 实时面板常显 | `ModalityRealtimePanel.vue:245-251` | i18n:configurationSettings.modalityRealtime.audioUnavailable |
| S-modality-63 | 操作 | 标签 + 下拉（选项为 owner 返回的 operations） | 首个操作 | 支持且有操作且无进行中会话 | `ModalityRealtimePanel.vue:255-265` | i18n:configurationSettings.modalityRealtime.operation |
| S-modality-64 | 参数（JSON 对象） | 标签 + 输入框（等宽） | 空 | 同上 | `ModalityRealtimePanel.vue:266-274` | i18n:configurationSettings.modalityRealtime.params |
| S-modality-65 | 提供商（可选） | 标签 + 输入框 | 空 | 同上 | `ModalityRealtimePanel.vue:277-285` | i18n:configurationSettings.modalityTask.modelProvider |
| S-modality-66 | 模型（可选） | 标签 + 输入框 | 空 | 同上 | `ModalityRealtimePanel.vue:286-294` | i18n:configurationSettings.modalityTask.modelModel |
| S-modality-67 | 打开会话 / 打开中… | 按钮（主按钮，打开中转圈） | 打开会话 | 可打开时可用 | `ModalityRealtimePanel.vue:297-306` | i18n:configurationSettings.modalityRealtime.open · opening |
| S-modality-68 | 关闭会话 / 关闭中… | 按钮（outline，关闭中转圈） | 关闭会话 | 有会话且未结束 | `ModalityRealtimePanel.vue:307-316` | i18n:configurationSettings.modalityRealtime.close · closing |
| S-modality-69 | {打开错误原文} | 错误文案（destructive） | — | 打开会话失败 | `ModalityRealtimePanel.vue:318` | 运行期 |
| S-modality-70 | {关闭错误原文} | 错误文案（destructive） | — | 关闭会话失败 | `ModalityRealtimePanel.vue:319` | 运行期 |
| S-modality-71 | 会话 {id}（{transport}） | 会话信息行 | — | 已打开会话 | `ModalityRealtimePanel.vue:322-325` | i18n:configurationSettings.modalityRealtime.connected |
| S-modality-72 | 会话已结束：{reason} | 结束提示 | — | 会话已结束 | `ModalityRealtimePanel.vue:326-328` | i18n:configurationSettings.modalityRealtime.ended |
| S-modality-73 | 所有者返回了直连传输。Desktop 未实现直连提供商传输，因此无法在此发送帧。 | 提示（灰底） | — | 会话 transport 为 direct | `ModalityRealtimePanel.vue:329-336` | i18n:configurationSettings.modalityRealtime.directUnavailable |
| S-modality-74 | 文本帧（JSON） | 标签 + 多行文本（等宽，3 行） | 空 | 非 direct 传输的会话 | `ModalityRealtimePanel.vue:338-347` | i18n:configurationSettings.modalityRealtime.textFrame |
| S-modality-75 | 发送帧 / 发送中… | 按钮（outline，发送中转圈） | 发送帧 | 同上；不可发送时禁用 | `ModalityRealtimePanel.vue:348-357` | i18n:configurationSettings.modalityRealtime.send · sending |
| S-modality-76 | {发送错误原文} | 错误文案（destructive） | — | 发送失败 | `ModalityRealtimePanel.vue:359` | 运行期 |
| S-modality-77 | 提供商事件 | 小标题（h3） | — | 已打开会话 | `ModalityRealtimePanel.vue:362` | i18n:configurationSettings.modalityRealtime.events |
| S-modality-78 | 暂无事件。 | 空态文案 | — | 事件列表为空 | `ModalityRealtimePanel.vue:363` | i18n:configurationSettings.modalityRealtime.noEvents |
| S-modality-79 | {时间} {JSON} | 事件行（等宽，max-h-56 滚动） | — | 每个事件一行 | `ModalityRealtimePanel.vue:364-369` | 运行期 |

本节元素数：79

本轮自检：5 个分类各自的页签/下拉/面板标题/计数；每个能力行的选择器（未设置/已选/自动徽标/不可用红字）；选择器内搜索、两列、两处空态、清除入口、操作原因；任务面板的不可用/提交表单 7 个字段/校验错误/任务列表 4 态（错误、空、行、加载更多）/详情 4 行 + 错误 + 动作错误 + 等待超时/三个动作按钮/结果文件与下载/未选中提示；实时面板的三种提示、表单、四个按钮态、会话信息、结束、直连不可用、文本帧发送、事件列表与空态。

---

## S-mcp MCP（`components/settings/McpSettingsPage.vue` + `McpImportPanel.vue`）

本节有四种形态：列表视图、配置/新建表单、查看连接弹窗、导入面板。四种形态连续编号。

### 列表视图

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-mcp-01 | MCP 服务器 | 页面标题（h2） | — | 列表视图 | `McpSettingsPage.vue:880` | i18n:toolsSettings.mcpServersTitle |
| S-mcp-02 | 连接外部工具和数据源。 | 说明文案 | — | 同上 | `McpSettingsPage.vue:882-883` | i18n:toolsSettings.mcpServersDescription |
| S-mcp-03 | 了解更多。 | 链接（外链，modelcontextprotocol.io） | — | 同上 | `McpSettingsPage.vue:884-892` | i18n:toolsSettings.learnMore |
| S-mcp-04 | 服务器 | 小标题（h3） | — | 同上 | `McpSettingsPage.vue:897` | i18n:toolsSettings.servers |
| S-mcp-05 | 导入 | 按钮（outline，下载图标，开合导入面板） | 收起 | `canImport` 为真 | `McpSettingsPage.vue:905-913` | i18n:toolsSettings.mcpImportOpen |
| S-mcp-06 | 添加服务器 | 按钮（主按钮，加号） | — | 列表视图 | `McpSettingsPage.vue:908-913` | i18n:toolsSettings.addServer |
| S-mcp-07 | 范围 | 标签（for 下拉） | — | 列表视图 | `McpSettingsPage.vue:925` | i18n:toolsSettings.scope |
| S-mcp-08 | 全部范围 / 用户 / 项目 | 下拉（w-40） | 全部范围 | 列表视图 | `McpSettingsPage.vue:926-936` | i18n:toolsSettings.scopeAll · scopeUser · scopeProject |
| S-mcp-09 | {n} MCP configuration issue(s) need attention. | 警告横幅标题（琥珀色，警示图标） | — | store 有 warnings | `McpSettingsPage.vue:938-945` | 字面量（英文，模板内拼单复数） |
| S-mcp-10 | {scope} · {serverId 或 path} · {message} | 警告明细列表项 | — | 同上（每条警告一行） | `McpSettingsPage.vue:946-951` | 运行期 |
| S-mcp-11 | 加载骨架（两条 56px 灰块） | 骨架（animate-pulse，无文案） | — | 首次加载中 | `McpSettingsPage.vue:953-956` | 无文案 |
| S-mcp-12 | {加载错误原文} | 错误块（destructive） | — | store.error 非空 | `McpSettingsPage.vue:958-960` | 运行期 |
| S-mcp-13 | 重试 | 按钮（outline，sm） | — | 同上 | `McpSettingsPage.vue:961-963` | i18n:common.retry |
| S-mcp-14 | 还没有配置 MCP 服务器 | 空态（虚线框） | — | 无任何服务器 | `McpSettingsPage.vue:965-967` | i18n:toolsSettings.noMcpServersConfigured |
| S-mcp-15 | 此范围内没有服务器。 | 空态（虚线框） | — | 有服务器但被范围筛选清空 | `McpSettingsPage.vue:969-971` | i18n:toolsSettings.scopeEmpty |
| S-mcp-16 | {server.name 或 id} | 服务器行标题（truncate，title） | — | 每个可见服务器一行 | `McpSettingsPage.vue:977-980` | 运行期 |
| S-mcp-17 | {scope} | 行内范围标签 | — | 每行 | `McpSettingsPage.vue:986` | 运行期 |
| S-mcp-18 | SSE（已弃用） | 行内标签 | — | 该服务器 transport 为 sse | `McpSettingsPage.vue:987` | i18n:toolsSettings.sseDeprecatedLabel |
| S-mcp-19 | {状态标签原文} | 行内状态文本 | — | 每行 | `McpSettingsPage.vue:988` | 运行期（`serverStatusLabel`） |
| S-mcp-20 | {警告文案} | 行内警告（destructive，truncate + title） | — | 该服务器有配置警告 | `McpSettingsPage.vue:989-996` | 运行期 |
| S-mcp-21 | {认证提示文案} | 行内提示（destructive，truncate + title） | — | 该服务器需要 OAuth 登录 | `McpSettingsPage.vue:998-1004` | 运行期（`getAuthNotice`） |
| S-mcp-22 | {count} 个工具 | 工具数徽标（扳手图标 + 数字，title/aria-label） | — | 每行 | `McpSettingsPage.vue:1005-1013` | i18n:toolsSettings.toolCount |
| S-mcp-23 | 认证 / 认证中 | 按钮（outline，宽 9.75rem，认证中转圈） | 认证 | 服务器可认证且未认证 | `McpSettingsPage.vue:1015-1025` | i18n:toolsSettings.authenticate · authenticating |
| S-mcp-24 | 取消 | 按钮（outline，X 图标，取消进行中的认证） | — | 认证可取消 | `McpSettingsPage.vue:1026-1035` | i18n:common.cancel |
| S-mcp-25 | 查看连接 | 图标按钮（活动图标，aria-label/title） | — | 能力支持 inspect | `McpSettingsPage.vue:1032-1044` | i18n:toolsSettings.mcpInspect |
| S-mcp-26 | 断开连接 | 图标按钮（电源图标） | — | 能力支持 disconnect | `McpSettingsPage.vue:1045-1057` | i18n:toolsSettings.mcpDisconnect |
| S-mcp-27 | 配置服务器 | 图标按钮（齿轮图标） | — | 每行常显 | `McpSettingsPage.vue:1058-1067` | i18n:toolsSettings.configureServer |
| S-mcp-28 | — | 开关（启用/禁用该服务器） | 开（`enabled !== false`） | 每行；写入中禁用 | `McpSettingsPage.vue:1068-1073` | 运行期 |
| S-mcp-29 | 来自插件 | 小标题（h3） | — | 存在插件提供的服务器 | `McpSettingsPage.vue:1081-1083` | i18n:toolsSettings.fromPlugins |
| S-mcp-30 | {plugin.id} | 插件服务器行标题 | — | 每个插件服务器一行 | `McpSettingsPage.vue:1090` | 运行期 |
| S-mcp-31 | {transport} | 行内 transport 文本 | — | 每行 | `McpSettingsPage.vue:1093` | 运行期 |
| S-mcp-32 | 只读 | 行内标签 | — | 每行（插件来源） | `McpSettingsPage.vue:1094` | i18n:toolsSettings.mcpReadOnly |
| S-mcp-33 | 需要登录 · 已存储的凭据已过期。 · 缺少权限范围：{scopes} / 连接成功 · 延迟 {latency}ms · {count} 个工具 / {错误} ({reason}) | 测试结果区（绿/红按 ok 着色） | — | 该插件服务器已测试过 | `McpSettingsPage.vue:1096-1120` | i18n:toolsSettings.mcpAuthRequired · mcpAuthExpired · mcpMissingScopes · connectionSucceeded · latencyMs · toolCount · connectionFailed |
| S-mcp-34 | 查看连接 | 图标按钮 | — | 能力支持 inspect | `McpSettingsPage.vue:1122-1134` | i18n:toolsSettings.mcpInspect |
| S-mcp-35 | 断开连接 | 图标按钮 | — | 能力支持 disconnect | `McpSettingsPage.vue:1135-1147` | i18n:toolsSettings.mcpDisconnect |
| S-mcp-36 | 测试连接 | 按钮（outline，测试中转圈） | — | 每个插件服务器行 | `McpSettingsPage.vue:1148-1159` | i18n:toolsSettings.testConnection |

### 配置 / 新建表单

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-mcp-37 | 连接自定义 MCP | 页面标题（h2） | — | `view !== 'list'` | `McpSettingsPage.vue:1169` | i18n:toolsSettings.connectCustomMcp |
| S-mcp-38 | 文档 | 链接（外链小字） | — | 同上 | `McpSettingsPage.vue:1170-1177` | i18n:toolsSettings.docs |
| S-mcp-39 | 正在加载... | 加载态（转圈 + 文案） | — | 编辑草稿加载中 | `McpSettingsPage.vue:1181-1184` | i18n:common.loading |
| S-mcp-40 | 已存储的值已隐藏，留空即可保留。 | 提示（灰底） | — | 编辑已有条目（有 snapshot） | `McpSettingsPage.vue:1186-1196` | i18n:toolsSettings.mcpWithheldValue |
| S-mcp-41 | 密钥值存储在用户范围，无法在此项目中修改。 | 提示（灰底） | — | snapshot 标记密钥在用户范围 | `McpSettingsPage.vue:1193-1195` | i18n:toolsSettings.mcpSecretsInUserScope |
| S-mcp-42 | 将此条目移动到另一个作用域；确认框会说明启用状态和密钥的处理方式。 | 提示（灰底/白底小字） | — | 编辑态、有修订号、能力支持 move | `McpSettingsPage.vue:1197-1206` · 文案 `:1202` | i18n:toolsSettings.mcpMoveHint |
| S-mcp-43 | 移动到项目 / 移动到用户 | 按钮（outline，sm，左右箭头图标） | — | 同上 | `McpSettingsPage.vue:1207-1216` | i18n:toolsSettings.mcpMoveToProject · mcpMoveToUser |
| S-mcp-44 | 移动 MCP 条目？ | 移动确认弹窗标题 | — | 点移动按钮 | `McpSettingsPage.vue:1219` | i18n:toolsSettings.mcpMoveConfirmTitle |
| S-mcp-45 | 将 {id} 移动到项目作用域？… / 将 {id} 移动到用户作用域？… | 移动确认描述 | — | 同上（按目标作用域二选一） | `McpSettingsPage.vue:1220-1224` | i18n:toolsSettings.mcpMoveConfirmProject · mcpMoveConfirmUser |
| S-mcp-46 | 同 ID 的插件条目优先级更高；在移除该插件条目之前，移动后的用户条目会被隐藏。 | 追加说明 | — | 移动目标被插件条目遮蔽 | `McpSettingsPage.vue:1222-1224` | i18n:toolsSettings.mcpMoveShadowedByPlugin |
| S-mcp-47 | 取消 | 弹窗按钮 | — | 移动确认弹窗 | `McpSettingsPage.vue:1228` | i18n:common.cancel |
| S-mcp-48 | 确认 | 弹窗主按钮 | — | 同上 | `McpSettingsPage.vue:1229-1232` | i18n:common.confirm |
| S-mcp-49 | 名称 | 标签 | — | 表单 | `McpSettingsPage.vue:1237` | i18n:toolsSettings.name |
| S-mcp-50 | my-server | 名称输入框（编辑态已有条目时禁用） | 空 | 同上 | `McpSettingsPage.vue:1238-1245` | i18n:toolsSettings.serverNamePlaceholder |
| S-mcp-51 | 范围 | 标签（for 下拉） | — | 同上 | `McpSettingsPage.vue:1248` | i18n:toolsSettings.scope |
| S-mcp-52 | 用户 / 项目 | 下拉（编辑态禁用；无项目 cwd 时「项目」禁用） | 用户 | 同上 | `McpSettingsPage.vue:1249-1260` | i18n:toolsSettings.scopeUser · scopeProject |
| S-mcp-53 | STDIO | transport 分段按钮 | 选中（新建默认） | 三段切换，编辑态仅当原 transport 匹配时可用 | `McpSettingsPage.vue:1262-1272` | i18n:toolsSettings.stdio |
| S-mcp-54 | Streamable HTTP | transport 分段按钮 | — | 同上 | `McpSettingsPage.vue:1273-1283` | i18n:toolsSettings.streamableHttp |
| S-mcp-55 | SSE（已弃用） | transport 分段按钮 | — | 同上 | `McpSettingsPage.vue:1284-1294` | i18n:toolsSettings.sseDeprecatedLabel |
| S-mcp-56 | SSE 已弃用，仅用于兼容旧版 MCP 服务器，不推荐用于新配置。请优先使用 Streamable HTTP。 | 弃用提示（琥珀色，警示图标） | — | transport 为 sse | `McpSettingsPage.vue:1405-1412` | i18n:toolsSettings.sseDeprecatedNotice |
| S-mcp-57 | 启动命令 | 标签 | — | transport 为 stdio | `McpSettingsPage.vue:1295` | i18n:toolsSettings.commandToLaunch |
| S-mcp-58 | openai-dev-mcp serve-sqlite | 输入框（占位） | 空 | 同上 | `McpSettingsPage.vue:1296-1301` | i18n:toolsSettings.commandExamplePlaceholder |
| S-mcp-59 | 参数 | 标签 | — | 同上 | `McpSettingsPage.vue:1305` | i18n:toolsSettings.arguments |
| S-mcp-60 | 替换参数（已存储 {count} 个） | 复选框 + 文案 | 未勾 | stdio 且编辑已有条目 | `McpSettingsPage.vue:1313-1317` | i18n:toolsSettings.mcpReplaceArguments |
| S-mcp-61 | 参数值 | 行内输入框（每项一行） | 取草稿/空 | 新建或勾选替换参数 | `McpSettingsPage.vue:1319-1335` | 运行期 |
| S-mcp-62 | 删除该参数 | 图标按钮（垃圾桶） | — | 每个参数行 | `McpSettingsPage.vue:1330-1332` | 无文案 |
| S-mcp-63 | 添加参数 | 按钮（secondary，整宽，加号） | — | 新建或勾选替换参数 | `McpSettingsPage.vue:1336-1338` | i18n:toolsSettings.addArgument |
| S-mcp-64 | 环境变量 | 标签 | — | transport 为 stdio | `McpSettingsPage.vue:1342` | i18n:toolsSettings.environmentVariables |
| S-mcp-65 | 键 / 值（编辑已有条目时值占位为「已存储的值已隐藏，留空即可保留。」） | 键值对输入 | 空 | 每个环境变量行 | `McpSettingsPage.vue:1344-1361` | i18n:toolsSettings.key · value · mcpWithheldValue |
| S-mcp-66 | 删除该环境变量 | 图标按钮（垃圾桶） | — | 每个环境变量行 | `McpSettingsPage.vue:1357-1359` | 无文案 |
| S-mcp-67 | 添加环境变量 | 按钮（secondary，整宽） | — | stdio | `McpSettingsPage.vue:1362-1364` | i18n:toolsSettings.addEnvironmentVariable |
| S-mcp-68 | 环境变量透传 | 标签 | — | stdio 且为新建 | `McpSettingsPage.vue:1368` | i18n:toolsSettings.environmentVariablePassthrough |
| S-mcp-69 | 透传变量名 | 行内输入（每项一行） | 空 | 同上 | `McpSettingsPage.vue:1370-1386` | 运行期 |
| S-mcp-70 | 删除该透传变量 | 图标按钮（垃圾桶） | — | 每个透传行 | `McpSettingsPage.vue:1383-1385` | 无文案 |
| S-mcp-71 | 添加变量 | 按钮（secondary，整宽） | — | 同上 | `McpSettingsPage.vue:1387-1389` | i18n:toolsSettings.addVariable |
| S-mcp-72 | 工作目录 | 标签 | — | transport 为 stdio | `McpSettingsPage.vue:1393` | i18n:toolsSettings.workingDirectory |
| S-mcp-73 | ~/code | 输入框（占位） | 空 | 同上 | `McpSettingsPage.vue:1394-1399` | i18n:toolsSettings.workingDirectoryPlaceholder |
| S-mcp-74 | URL | 标签 | — | transport 为 http/sse | `McpSettingsPage.vue:1414` | i18n:toolsSettings.url |
| S-mcp-75 | {脱敏 endpoint} | 等宽文案 | — | 编辑已有条目 | `McpSettingsPage.vue:1415-1419` | 运行期（`redactedEndpoint`） |
| S-mcp-76 | 替换 URL（路径和查询已隐藏） | 复选框 + 文案 | 未勾 | 编辑已有条目 | `McpSettingsPage.vue:1424-1428` | i18n:toolsSettings.mcpReplaceUrl |
| S-mcp-77 | https://mcp.example.com/mcp | URL 输入框（占位） | 空 | 新建或勾选替换 URL | `McpSettingsPage.vue:1431-1435` | i18n:toolsSettings.urlExamplePlaceholder |
| S-mcp-78 | Bearer 令牌 | 标签 | — | transport 为 http/sse | `McpSettingsPage.vue:1439` | i18n:toolsSettings.bearerTokenEnvVar |
| S-mcp-79 | 移除已存储的 Bearer 令牌 | 复选框 + 文案 | 未勾 | 编辑已有条目且已配置认证 | `McpSettingsPage.vue:1446-1450` | i18n:toolsSettings.mcpRemoveBearer |
| S-mcp-80 | Bearer 令牌值（编辑态占位「已存储的值已隐藏，留空即可保留。」） | 输入框 | 空 | 未勾选移除 | `McpSettingsPage.vue:1452-1456` | i18n:toolsSettings.mcpWithheldValue |
| S-mcp-81 | 请求头 | 标签 | — | transport 为 http/sse | `McpSettingsPage.vue:1461` | i18n:toolsSettings.headers |
| S-mcp-82 | 键 / 值（编辑态值占位隐藏提示） | 请求头键值行 | 空 | 每个请求头一行 | `McpSettingsPage.vue:1462-1479` | i18n:toolsSettings.key · value |
| S-mcp-83 | 删除该请求头 | 图标按钮（垃圾桶） | — | 每个请求头行 | `McpSettingsPage.vue:1475-1477` | 无文案 |
| S-mcp-84 | 添加请求头 | 按钮（secondary，整宽） | — | 同上 | `McpSettingsPage.vue:1481-1483` | i18n:toolsSettings.addHeader |
| S-mcp-85 | 来自环境变量的请求头 | 标签 | — | http/sse 且为新建 | `McpSettingsPage.vue:1487` | i18n:toolsSettings.headersFromEnvironmentVariables |
| S-mcp-86 | 键 / `ENV_VAR` | 键值行（第二列固定占位 ENV_VAR） | 空 | 每个 env header 一行 | `McpSettingsPage.vue:1489-1500` | 字面量（`ENV_VAR`） |
| S-mcp-87 | 删除该行 | 图标按钮（垃圾桶） | — | 每个 env header 行 | `McpSettingsPage.vue:1499-1501` | 无文案 |
| S-mcp-88 | 添加变量 | 按钮（secondary，整宽） | — | 同上 | `McpSettingsPage.vue:1502-1504` | i18n:toolsSettings.addVariable |
| S-mcp-89 | {表单/字段错误列表} | 错误区（destructive，逐条） | — | 校验失败或保存失败 | `McpSettingsPage.vue:1510-1513` | i18n:toolsSettings.nameRequired · nameTooLong · duplicateServerId · commandRequired · urlRequired · urlInvalid · invalidEnvVar · duplicateKey · authorizationHeaderConflict · keyAndValueRequired · invalidEnvVarName · mcpConflictReread |
| S-mcp-90 | 需要登录 · 已存储的凭据已过期。 · 缺少权限范围：{scopes} / 连接成功 · … / {错误} ({reason}) | 测试结果条（绿/红） | — | 已选服务器且有测试结果 | `McpSettingsPage.vue:1515-1548` | i18n:toolsSettings.mcpAuthRequired · mcpAuthExpired · mcpMissingScopes · connectionSucceeded · latencyMs · toolCount · connectionFailed |
| S-mcp-91 | {tool.name} | 测试结果内的工具名列表项 | — | 测试成功且返回工具 | `McpSettingsPage.vue:1582-1584` | 运行期 |
| S-mcp-92 | 测试连接 | 按钮（outline，测试中转圈） | — | 已选服务器 | `McpSettingsPage.vue:1552-1560` | i18n:toolsSettings.testConnection |
| S-mcp-93 | 删除此服务器 | 按钮（ghost，红色，垃圾桶） | — | 已选服务器 | `McpSettingsPage.vue:1562-1571` | i18n:toolsSettings.deleteServer |
| S-mcp-94 | 删除 MCP 服务器 | 确认弹窗标题 | — | 点删除 | `McpSettingsPage.vue:1575` | i18n:toolsSettings.deleteServerTitle |
| S-mcp-95 | 确定要删除 "{name}" 吗？此操作无法撤销。 | 确认弹窗描述 | — | 同上 | `McpSettingsPage.vue:1576-1578` | i18n:toolsSettings.deleteServerDescription |
| S-mcp-96 | 取消 | 弹窗按钮 | — | 同上 | `McpSettingsPage.vue:1581` | i18n:common.cancel |
| S-mcp-97 | 删除 | 弹窗主按钮 | — | 同上 | `McpSettingsPage.vue:1582-1584` | i18n:common.delete |
| S-mcp-98 | 取消 | 表单底部按钮（outline） | — | 表单常显 | `McpSettingsPage.vue:1591-1594` | i18n:common.cancel |
| S-mcp-99 | 保存 | 表单主按钮（保存中转圈，不可保存时禁用） | — | 表单常显 | `McpSettingsPage.vue:1595-1599` | i18n:toolsSettings.save |

### 查看连接弹窗

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-mcp-100 | 查看连接 | 弹窗标题 | — | 点「查看连接」 | `McpSettingsPage.vue:1608` | i18n:toolsSettings.mcpInspect |
| S-mcp-101 | 当前活动连接代次和生效配置，这些值只读。 | 弹窗描述 | — | 同上 | `McpSettingsPage.vue:1609` | i18n:toolsSettings.mcpInspectDescription |
| S-mcp-102 | 已启用 / 已禁用 | 状态文案 | — | 有 inspect 结果 | `McpSettingsPage.vue:1613-1615` | i18n:toolsSettings.enabled · disabled |
| S-mcp-103 | 没有活动连接 | 空态文案 | — | 连接列表为空 | `McpSettingsPage.vue:1616-1618` | i18n:toolsSettings.mcpNoLiveConnections |
| S-mcp-104 | {状态} · 已存储的凭据已过期。 · {count} 个工具 · {失败原因} | 连接项（逐个连接一块） | — | 有活连接 | `McpSettingsPage.vue:1619-1636` | i18n:toolsSettings.mcpAuthExpired · toolCount |
| S-mcp-105 | 生效配置 | 小标题（h4） | — | 有 configuration | `McpSettingsPage.vue:1635` | i18n:toolsSettings.mcpEffectiveConfiguration |
| S-mcp-106 | {command} · {cwd} · 参数: {n} · 环境变量: {names} / {endpoint} · 请求头: {names} | 生效配置内容 | — | 同上（stdio 或 http 二选一） | `McpSettingsPage.vue:1637-1656` | i18n:toolsSettings.arguments · environmentVariables · headers |

### 导入面板（`McpImportPanel.vue`）

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-mcp-107 | 导入 MCP 服务器 | 面板标题（h3） | — | 点「导入」展开 | `McpImportPanel.vue:319` | i18n:toolsSettings.mcpImportTitle |
| S-mcp-108 | 从其他本地客户端发现 MCP 服务器，或选择 JSON 文件。界面只显示字段名；导入和替换的条目默认禁用。 | 面板描述 | — | 同上 | `McpImportPanel.vue:320` | i18n:toolsSettings.mcpImportDescription |
| S-mcp-109 | 导入到：用户配置 / 导入到：项目 {cwd} | 目标说明 | — | 同上 | `McpImportPanel.vue:321-326` | i18n:toolsSettings.mcpImportDestinationUser · mcpImportDestinationProject |
| S-mcp-110 | 关闭 | 按钮（outline，sm，X 图标） | — | 同上；忙时禁用 | `McpImportPanel.vue:327-330` | i18n:common.close |
| S-mcp-111 | 发现 | 按钮（outline，sm，放大镜/转圈） | — | 能力支持 discover | `McpImportPanel.vue:333-344` | i18n:toolsSettings.mcpImportDiscover |
| S-mcp-112 | 选择 JSON 文件 | 按钮（outline，sm，JSON 图标） | — | 能力支持 preview | `McpImportPanel.vue:345-352` | i18n:toolsSettings.mcpImportChooseFile |
| S-mcp-113 | {通知原文} | 成功提示（role=status，绿色） | — | 有 notice | `McpImportPanel.vue:354-356` | 运行期 |
| S-mcp-114 | {错误原文} | 错误提示（role=alert，红色） | — | 有 error | `McpImportPanel.vue:357-359` | 运行期 |
| S-mcp-115 | 其他本地客户端的存储 | 小标题（h4） | — | 已执行发现 | `McpImportPanel.vue:362` | i18n:toolsSettings.mcpImportDiscoveryTitle |
| S-mcp-116 | 只有在你预览时，Dim 才会在 App Server 内读取存储，绝不显示或返回其中的值。 | 说明文案 | — | 同上 | `McpImportPanel.vue:363` | i18n:toolsSettings.mcpImportDiscoveryDescription |
| S-mcp-117 | 未找到已知存储。 | 空态（role=status） | — | 发现结果为空 | `McpImportPanel.vue:364-366` | i18n:toolsSettings.mcpImportNoStores |
| S-mcp-118 | {id} · {client} · {scope} · {state} + {path} + {entries} 个条目 / {error} | 发现结果行 | — | 每个发现的存储一行 | `McpImportPanel.vue:367-387` | i18n:toolsSettings.mcpImportEntries |
| S-mcp-119 | 预览 | 按钮（outline，sm） | — | 该存储 state 为 ready | `McpImportPanel.vue:381-387` | i18n:common.preview |
| S-mcp-120 | 导入预览 · {source} · {revision} | 预览面板标题 | — | 已有预览 | `McpImportPanel.vue:392-395` | i18n:toolsSettings.mcpImportPreviewTitle |
| S-mcp-121 | 关闭 | 按钮（outline，sm） | — | 同上 | `McpImportPanel.vue:396-398` | i18n:common.close |
| S-mcp-122 | 来自存储：{id}。内容留在 App Server 内。 / 来自文件：{name}。 | 来源说明 | — | 预览来自存储或文件 | `McpImportPanel.vue:399-404` | i18n:toolsSettings.mcpImportFromStore · mcpImportFromFile |
| S-mcp-123 | 不显示任何值，只列出变更的字段名。导入和替换的条目默认禁用。 | 说明文案 | — | 有预览 | `McpImportPanel.vue:405` | i18n:toolsSettings.mcpImportNoValues |
| S-mcp-124 | 导入到项目时，工作区文件只写入公开骨架。…会写入用户范围的 mcp-secrets 记录。 | 项目写入提示（琥珀色） | — | 预览来源为项目 | `McpImportPanel.vue:406-409` | i18n:toolsSettings.mcpImportProjectNotice |
| S-mcp-125 | 不会携带的顶层成员：{names}。/ 不会携带的顶层成员：无。 | 说明文案 | — | 有预览 | `McpImportPanel.vue:410` | i18n:toolsSettings.mcpImportUnknown · mcpImportUnknownNone |
| S-mcp-126 | 此预览之后 MCP 源已变更。请重新预览后再应用。 + 重试 | 错误文案 + 按钮 | — | 预览已过期 | `McpImportPanel.vue:411-416` | i18n:toolsSettings.mcpImportStale · i18n:common.retry |
| S-mcp-127 | 导入与当前源冲突，未写入任何内容；请重新发送此预览或调整条目。（+ 冲突细节） | 错误文案 | — | 导入冲突 | `McpImportPanel.vue:418-420` | i18n:toolsSettings.mcpImportConflict |
| S-mcp-128 | 源已写入，但隐藏记录未写入。请替换相同条目以完成导入。 / 无法将已写入的条目与此文档匹配。请预览目标源后手动替换。 + 完成部分导入 | 错误文案 + 按钮 | — | 部分导入 | `McpImportPanel.vue:421-431` | i18n:toolsSettings.mcpImportPartial · mcpImportPartialUnrecoverable · mcpImportRetryReplace |
| S-mcp-129 | 文档中没有 mcpServers 条目。 | 空态（role=status） | — | 预览条目为空 | `McpImportPanel.vue:432-434` | i18n:toolsSettings.mcpImportNoEntries |
| S-mcp-130 | {entry.id} · {status} · {transport} | 条目块标题 | — | 每个预览条目一块 | `McpImportPanel.vue:435-442` | 运行期 |
| S-mcp-131 | 变更字段：{fields} | 说明文案 | — | 条目有 changes | `McpImportPanel.vue:443-445` | i18n:toolsSettings.mcpImportChanged |
| S-mcp-132 | 不会携带：{fields} | 说明文案 | — | 条目有 unsupported | `McpImportPanel.vue:446-448` | i18n:toolsSettings.mcpImportUnsupported |
| S-mcp-133 | 另有 {count} 个字段名未列出。 | 说明文案 | — | 条目有 omitted | `McpImportPanel.vue:449-451` | i18n:toolsSettings.mcpImportOmitted |
| S-mcp-134 | {条目错误原文} | 错误文案（destructive） | — | 条目带 error | `McpImportPanel.vue:452` | 运行期 |
| S-mcp-135 | 此条目无法解码，只能跳过。 | 说明文案 | — | 条目 status 为 invalid | `McpImportPanel.vue:453-455` | i18n:toolsSettings.mcpImportInvalidHint |
| S-mcp-136 | 操作 + 导入 / 跳过 / 替换现有条目 / 以新 ID 导入 | 标签 + 原生下拉 | 导入 | 非 invalid 条目 | `McpImportPanel.vue:456-471` | i18n:toolsSettings.mcpImportAction · mcpImportActionImport · Skip · Replace · Rename |
| S-mcp-137 | 新 ID | 标签 + 输入框 | 空 | 该条目操作为「以新 ID 导入」 | `McpImportPanel.vue:472-480` | i18n:toolsSettings.mcpImportNewId |
| S-mcp-138 | 应用导入 | 按钮（主按钮，下载图标/转圈） | — | 能力支持 apply 且预览中 | `McpImportPanel.vue:482-488` | i18n:toolsSettings.mcpImportApply |

本节元素数：138

本轮自检：列表视图的加载骨架/加载错误+重试/两种空态（无服务器、范围筛选空）/警告横幅与明细/普通行（范围、SSE 弃用、状态、警告、认证提示、工具数）/认证三态（可认证、认证中、可取消）/inspect 与 disconnect 能力开关/开关写入中/插件服务器行（transport、只读、测试三种结果）/编辑与新建表单（名称、范围、三种 transport、SSE 弃用说明、stdio 五个字段组、http/sse 四个字段组、脱敏与三个「替换」勾选、错误区、测试结果与工具列表、删除确认、底部取消/保存）/查看连接弹窗（启用态、无连接空态、连接项、生效配置两种形态）/导入面板（发现与预览入口、发现空态与行、预览失效/冲突/部分导入、条目四态与操作下拉、新 ID、应用按钮）。

---

## S-configuration 模型供应（`components/settings/ConfigurationSettings.vue`）

本节是左右分栏：左侧供应商列表（含「添加自定义提供商」入口），右侧详情面板按供应商类型切换成 6 种形态（添加自定义 / OpenAI Codex OAuth / Grok Build device flow / 账户托管只读 / 通用托管 OAuth / 未配置表单 / 已配置凭据+模型），另有「添加模型」命令面板与「模型编辑」弹窗。连续编号。

### 左侧列表

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-configuration-01 | 加载骨架（左 4 条 + 右 4 块） | 骨架（Skeleton，无文案） | — | store 未加载完成 | `ConfigurationSettings.vue:2081-2104` | 无文案 |
| S-configuration-02 | 模型供应 | 侧栏小标题（大写小字） | — | 已加载 | `ConfigurationSettings.vue:2118` | i18n:configurationSettings.providers |
| S-configuration-03 | 搜索供应商 | 搜索输入框（放大镜，Esc 清空） | 空 | 已加载 | `ConfigurationSettings.vue:2122-2130` | i18n:configurationSettings.searchProviders |
| S-configuration-04 | 清除搜索 | 图标按钮（X，aria-label） | — | 搜索框有内容 | `ConfigurationSettings.vue:2131-2138` | i18n:configurationSettings.clearSearch |
| S-configuration-05 | 已启用 + {count} | 分组折叠按钮（chevron，aria-expanded） | 展开 | 已加载 | `ConfigurationSettings.vue:2128-2152` · 分组 `:553-559` | i18n:configurationSettings.enabledProviders |
| S-configuration-06 | 未启用 + {count} | 分组折叠按钮 | 折叠 | 已加载 | `ConfigurationSettings.vue:2128-2152` · 分组 `:560-565` | i18n:configurationSettings.disabledProviders |
| S-configuration-07 | {提供商名}（+ 右侧 ● 已配置 / ○ 未配置） | 供应商项按钮（图标 + 跑马灯名称 + 状态点） | 首项高亮 | 每个供应商一项 | `ConfigurationSettings.vue:2156-2175` | 运行期 |
| S-configuration-08 | 添加 | 按钮（加号，进入「添加自定义提供商」） | — | `dimProviderOnly` 不为真 | `ConfigurationSettings.vue:2178-2191` | i18n:configurationSettings.add |
| S-configuration-09 | 返回 | 图标按钮（左箭头，aria-label） | — | 窄屏（compact）详情态 | `ConfigurationSettings.vue:2198-2207` | i18n:common.back |

### 形态 A：添加自定义提供商

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-configuration-10 | 添加自定义提供商 | 面板标题（h3） | — | 点「添加」 | `ConfigurationSettings.vue:2212` | i18n:configurationSettings.addCustomProvider |
| S-configuration-11 | 协议 | 标签 + 5 个单选：OpenAI 兼容 / OpenAI Responses / Anthropic / Gemini / New API | OpenAI 兼容 | 添加表单 | `ConfigurationSettings.vue:2216-2237` · 标签 `:204-210` | i18n:configurationSettings.adapter.* |
| S-configuration-12 | 名称 | 标签 | — | 同上 | `ConfigurationSettings.vue:2231` | i18n:configurationSettings.name |
| S-configuration-13 | 我的自定义提供商 | 输入框（占位） | 空 | 同上 | `ConfigurationSettings.vue:2233` | i18n:configurationSettings.customProviderNamePlaceholder |
| S-configuration-14 | 基础 URL | 标签 | — | 同上 | `ConfigurationSettings.vue:2236` | i18n:configurationSettings.baseUrl |
| S-configuration-15 | https://api.openai.com/v1（随协议变化的占位） | 输入框 | 该协议默认地址 | 同上 | `ConfigurationSettings.vue:2237` · `:3096` | i18n:configurationSettings.customBaseUrlPlaceholder |
| S-configuration-16 | API 密钥 | 标签 | — | 同上 | `ConfigurationSettings.vue:2241` | i18n:providerSettings.apiKey |
| S-configuration-17 | sk-... | 密码输入框（占位） | 空 | 同上 | `ConfigurationSettings.vue:2242` | i18n:configurationSettings.apiKeyPlaceholder |
| S-configuration-18 | User-Agent | 标签 | — | 同上 | `ConfigurationSettings.vue:2246` | i18n:configurationSettings.userAgent |
| S-configuration-19 | Mozilla/5.0 ... | 输入框（等宽，占位） | 空 | 同上 | `ConfigurationSettings.vue:2247` | 字面量（placeholder） |
| S-configuration-20 | 留空则使用运行时默认值。仅应用于此供应商的模型获取和模型请求。 | 说明文案 | — | 同上 | `ConfigurationSettings.vue:2248` | i18n:configurationSettings.userAgentHint |
| S-configuration-21 | 模型 · {可见数} / {总数} | 全选复选框 + 文案（三态 indeterminate） | 未全选 | 草稿模型非空 | `ConfigurationSettings.vue:2253-2266` | i18n:configurationSettings.models |
| S-configuration-22 | 添加模型 | 按钮（outline，sm，加号） | — | 添加表单 | `ConfigurationSettings.vue:2268-2275` | i18n:configurationSettings.addModel |
| S-configuration-23 | 刷新（获取模型） | 图标按钮（刷新图标，获取中转圈） | — | 可获取模型时可用 | `ConfigurationSettings.vue:2276-2284` | 无文案（图标按钮） |
| S-configuration-24 | 搜索模型名称或 ID | 搜索输入框（占位，Esc 清空） | 空 | 草稿模型非空 | `ConfigurationSettings.vue:2287-2299` | i18n:configurationSettings.searchModels |
| S-configuration-25 | 清除搜索 | 图标按钮（X，aria-label） | — | 搜索框有内容 | `ConfigurationSettings.vue:2300-2307` | i18n:configurationSettings.clearSearch |
| S-configuration-26 | {模型名} + 自定义 / 工具 / 视觉 / 推理 / 媒体能力 徽标 + {count}k 上下文 · {count}k 输出 + 启用开关 | 草稿模型行（点击打开模型编辑） | 按选择集 | 每个草稿模型一行 | `ConfigurationSettings.vue:2318-2364` | i18n:configurationSettings.modelCustomized · modelCapabilityTools · modelCapabilityVision · modelCapabilityReasoning · modelCapabilityMedia · modelContextK · modelOutputK |
| S-configuration-27 | 没有匹配的模型 | 空态（搜索无匹配） | — | 草稿模型非空但过滤后为空 | `ConfigurationSettings.vue:2365-2370` | i18n:configurationSettings.noMatchingModels |
| S-configuration-28 | 还没有模型。可以手动添加，也可以在提供商支持模型列表时点击刷新。 | 空态（虚线框） | — | 草稿模型为空 | `ConfigurationSettings.vue:2371-2377` | i18n:configurationSettings.manualModelListEmpty |
| S-configuration-29 | {表单错误原文} | 错误条（destructive，警示图标） | — | 校验失败 | `ConfigurationSettings.vue:2379-2387` | 运行期 |
| S-configuration-30 | 取消 | 按钮（outline） | — | 添加表单 | `ConfigurationSettings.vue:2390` | i18n:common.cancel |
| S-configuration-31 | 保存 | 按钮（保存中转圈 / chevron） | — | 同上；不可保存时禁用 | `ConfigurationSettings.vue:2391-2396` | i18n:common.save |

### 详情头（所有形态共用）

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-configuration-32 | {提供商名} | 详情头标题（truncate） | — | 已选中供应商 | `ConfigurationSettings.vue:2399-2401` | 运行期 |
| S-configuration-33 | 未登录 / 登录中 / 等待设备 / 已连接 / 错误 / 已禁用 / OAuth 错误 / 等待设备授权 | 状态标签 | — | 已选中供应商（按类型与 OAuth 状态映射） | `ConfigurationSettings.vue:2402` · 映射 `:445-480` | i18n:configurationSettings.openAICodexStatus* · grokBuildStatus* |
| S-configuration-34 | — | 开关（启用/停用该供应商） | 视供应商 | 已选中；保存中/获取模型/需先登录/需先配置时禁用 | `ConfigurationSettings.vue:2405-2418` | 运行期 |

### 形态 B：OpenAI Codex OAuth

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-configuration-35 | 登录 OpenAI 后可在 DimAgent 中使用 Codex 模型。 | 未登录说明 | — | 状态 signed-out 或 error | `ConfigurationSettings.vue:2427-2430` | i18n:configurationSettings.openAICodexSignedOutDescription |
| S-configuration-36 | 使用 OpenAI 登录 | 按钮（sm，外链图标，进行中转圈） | — | 同上 | `ConfigurationSettings.vue:2436-2446` | i18n:configurationSettings.openAICodexSignInWithOpenAI |
| S-configuration-37 | 请将浏览器地址栏中的 callback URL 复制到 DimAgent 完成登录。 / Copy the callback URL from your browser address bar into DimAgent to complete sign-in. | 等待提示（转圈） | — | 状态 pending-browser | `ConfigurationSettings.vue:2450-2456` | i18n:configurationSettings.openAICodexWaitingForBrowser |
| S-configuration-38 | 取消 | 按钮（outline，sm） | — | 同上 | `ConfigurationSettings.vue:2458-2464` | i18n:common.cancel |
| S-configuration-39 | 打开 URL | 按钮（outline，sm，外链图标） | — | 同上；无授权 URL 时禁用 | `ConfigurationSettings.vue:2472-2481` | i18n:configurationSettings.openAICodexOpenUrl |
| S-configuration-40 | 复制 | 按钮（outline，sm，复制图标） | — | 同上；无授权 URL 时禁用 | `ConfigurationSettings.vue:2482-2490` | i18n:common.copy |
| S-configuration-41 | Callback URL | 标签 + 输入框（等宽，占位「粘贴完整 callback URL 或 query string」） | 空 | 同上 | `ConfigurationSettings.vue:2492-2506` | i18n:configurationSettings.openAICodexCallbackUrl · openAICodexCallbackPlaceholder |
| S-configuration-42 | 确认 | 提交按钮（sm，发送图标；空值或忙时禁用） | — | 同上 | `ConfigurationSettings.vue:2508-2514` | i18n:common.confirm |
| S-configuration-43 | {redirect uri} | 小字回显（truncate） | — | 有 redirect uri | `ConfigurationSettings.vue:2516-2518` | 运行期 |
| S-configuration-44 | {账户标签} | 已连接卡标题（主色，truncate） | — | 状态 authenticated | `ConfigurationSettings.vue:2526-2528` | 运行期（或「OpenAI 账户」回退） |
| S-configuration-45 | 套餐: {planType} | 小字行 | — | 已连接且带 planType | `ConfigurationSettings.vue:2532` | i18n:configurationSettings.openAICodexPlan |
| S-configuration-46 | 账户: {accountIdPreview} | 小字行 | — | 已连接且带 accountIdPreview | `ConfigurationSettings.vue:2535` | i18n:configurationSettings.openAICodexAccount |
| S-configuration-47 | 刷新 | 按钮（outline，sm，刷新中转圈） | — | 已连接 | `ConfigurationSettings.vue:2540-2548` | i18n:common.refresh |
| S-configuration-48 | 退出登录 | 按钮（outline，sm） | — | 已连接 | `ConfigurationSettings.vue:2549-2556` | i18n:accountSettings.signOut |
| S-configuration-49 | 令牌已生效。模型开关决定聊天中可用的 Codex 模型。 | 说明文案 | — | 已连接 | `ConfigurationSettings.vue:2561` | i18n:configurationSettings.openAICodexAuthenticatedDescription |
| S-configuration-50 | 环境设置已禁用 OpenAI Codex。 | 提示条（灰底） | — | 状态 disabled | `ConfigurationSettings.vue:2568-2574` | i18n:configurationSettings.openAICodexDisabledDescription |
| S-configuration-51 | 基础 URL | 标签 + 等宽只读值 | 供应商 baseUrl 或协议默认地址 | 面板内常显 | `ConfigurationSettings.vue:2576-2581` | i18n:configurationSettings.baseUrl |
| S-configuration-52 | {OAuth 错误原文} | 错误条（destructive） | — | 登录/刷新失败 | `ConfigurationSettings.vue:2583-2590` | i18n:configurationSettings.openAICodexOAuthFailed |
| S-configuration-53 | 模型 · {可见数} / {总数} | 全选复选框 + 文案 | 未全选 | 已连接且模型可选中 | `ConfigurationSettings.vue:2598-2609` | i18n:configurationSettings.models |
| S-configuration-54 | 搜索模型名称或 ID | 搜索输入框 | 空 | 有模型 | `ConfigurationSettings.vue:2610-2618` | i18n:configurationSettings.searchModels |
| S-configuration-55 | {模型名} + 视觉 / 推理 徽标 + 开关 | 模型行（Codex 面板） | 按选择集 | 每行一个模型 | `ConfigurationSettings.vue:2620-2652` | i18n:configurationSettings.modelCapabilityVision · modelCapabilityReasoning |
| S-configuration-56 | 没有匹配的模型 | 空态 | — | 搜索无匹配 | `ConfigurationSettings.vue:2653-2655` | i18n:configurationSettings.noMatchingModels |
| S-configuration-57 | 登录成功后将自动拉取可用模型。 | 空态（虚线框） | — | 无模型 | `ConfigurationSettings.vue:2656-2660` | i18n:configurationSettings.modelsAutoLoadAfterSignIn |

### 形态 C：Grok Build device flow

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-configuration-58 | 登录 SuperGrok 或 Premium+ 后，即可在 DimAgent 中使用 Grok Build 模型。 | 未登录说明 | — | 状态 signed-out 或 error | `ConfigurationSettings.vue:2671-2675` | i18n:configurationSettings.grokBuildSignedOutDescription |
| S-configuration-59 | 使用 SuperGrok / Premium+ 登录 | 按钮（sm，外链图标） | — | 同上 | `ConfigurationSettings.vue:2681-2689` | i18n:configurationSettings.grokBuildSignInWithSuperGrok |
| S-configuration-60 | 环境设置已禁用 Grok Build OAuth。 | 提示条（灰底） | — | 状态 disabled | `ConfigurationSettings.vue:2690-2696` | i18n:configurationSettings.grokBuildDisabledDescription |
| S-configuration-61 | 正在等待授权... | 等待提示（转圈） | — | 状态 pending-device | `ConfigurationSettings.vue:2700-2704` | i18n:configurationSettings.grokBuildWaitingForDevice |
| S-configuration-62 | 取消 | 按钮（outline，sm） | — | 同上 | `ConfigurationSettings.vue:2705-2712` | i18n:common.cancel |
| S-configuration-63 | 设备码 | 小字标签 | — | 同上 | `ConfigurationSettings.vue:2714-2717` | i18n:configurationSettings.grokBuildDeviceCode |
| S-configuration-64 | {userCode} | 等宽大号验证码 | — | 同上 | `ConfigurationSettings.vue:2718-2720` | 运行期 |
| S-configuration-65 | 打开验证页面 | 按钮（outline，sm，外链图标） | — | 同上；无验证 URI 时禁用 | `ConfigurationSettings.vue:2725-2733` | i18n:configurationSettings.grokBuildOpenVerification |
| S-configuration-66 | 复制 | 按钮（outline，sm，复制图标） | — | 同上 | `ConfigurationSettings.vue:2734-2742` | i18n:common.copy |
| S-configuration-67 | {verificationUri} | 小字回显（truncate） | — | 同上 | `ConfigurationSettings.vue:2744-2746` | 运行期 |
| S-configuration-68 | {账户标签} | 已连接卡标题 | — | 状态 authenticated | `ConfigurationSettings.vue:2755` | 运行期（或「Grok Build 账户」回退） |
| S-configuration-69 | 套餐: {planType} | 小字行 | — | 已连接且带 planType | `ConfigurationSettings.vue:2758` | i18n:configurationSettings.grokBuildPlan |
| S-configuration-70 | 账户: {accountIdPreview} | 小字行 | — | 已连接且带 accountIdPreview | `ConfigurationSettings.vue:2761` | i18n:configurationSettings.grokBuildAccount |
| S-configuration-71 | 刷新 | 按钮（outline，sm） | — | 已连接 | `ConfigurationSettings.vue:2765-2773` | i18n:common.refresh |
| S-configuration-72 | 退出登录 | 按钮（outline，sm） | — | 已连接 | `ConfigurationSettings.vue:2774-2781` | i18n:accountSettings.signOut |
| S-configuration-73 | OAuth 令牌已生效。模型开关决定聊天中可用的 Grok Build 模型。 | 说明文案 | — | 已连接 | `ConfigurationSettings.vue:2787` | i18n:configurationSettings.grokBuildAuthenticatedDescription |
| S-configuration-74 | 基础 URL | 标签 + 等宽只读值（默认 `https://api.x.ai/v1`） | 供应商 baseUrl | 面板内常显 | `ConfigurationSettings.vue:2790-2795` | i18n:configurationSettings.baseUrl |
| S-configuration-75 | {OAuth 错误原文} | 错误条（destructive） | — | 登录/刷新失败 | `ConfigurationSettings.vue:2798-2806` | i18n:configurationSettings.grokBuildOAuthFailed |
| S-configuration-76 | 模型 · {可见数} / {总数} | 全选复选框 + 文案 | 未全选 | 已连接 | `ConfigurationSettings.vue:2811-2822` | i18n:configurationSettings.models |
| S-configuration-77 | {模型名} + 自定义/工具/视觉/推理/媒体 徽标 + 上下文/输出 + 开关（点击进模型编辑，aria-label「修改模型: {name}」） | 模型行（Grok 面板） | 按选择集 | 每行一个模型 | `ConfigurationSettings.vue:2834-2883` | i18n:configurationSettings.editModel · modelCustomized · modelCapability* |
| S-configuration-78 | 登录后加载 Grok Build 模型。 | 空态（虚线框） | — | 无模型 | `ConfigurationSettings.vue:2895-2899` | i18n:configurationSettings.grokBuildModelsSignInPrompt |

### 形态 D：账户托管（只读）

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-configuration-79 | 由 Dim 账户管理 | 横幅标题（锁图标，整块可点跳账户设置） | — | 供应商由账户托管 | `ConfigurationSettings.vue:2907-2925` | i18n:configurationSettings.managedByAccount |
| S-configuration-80 | 此提供商由你的 Dim 账户管理，无法在此编辑。 / 登录后可使用 Dim 模型。点击跳转到账户设置。 | 横幅说明（按下文二选一） | — | 已连接 / 未登录 | `ConfigurationSettings.vue:2916-2923` | i18n:configurationSettings.managedByAccountDescription · signInManagedDescription |
| S-configuration-81 | 基础 URL | 标签 + 等宽只读值 | 供应商 baseUrl 或协议默认 | 面板内常显 | `ConfigurationSettings.vue:2927-2932` | i18n:configurationSettings.baseUrl |
| S-configuration-82 | {表单错误原文} | 错误条（destructive） | — | 有错误 | `ConfigurationSettings.vue:2934-2942` | 运行期 |
| S-configuration-83 | 模型 · {可见数} / {总数} | 全选复选框 + 文案 | 未全选 | 托管供应商 | `ConfigurationSettings.vue:2943-2952` | i18n:configurationSettings.models |
| S-configuration-84 | 搜索模型名称或 ID | 搜索输入框 | 空 | 有模型 | `ConfigurationSettings.vue:2953-2961` | i18n:configurationSettings.searchModels |
| S-configuration-85 | {模型名} + 视觉 / 推理 徽标 + 开关 | 模型行（托管面板） | 按选择集 | 每行一个模型 | `ConfigurationSettings.vue:2966-2992` | i18n:configurationSettings.modelCapabilityVision · modelCapabilityReasoning |
| S-configuration-86 | 登录成功后将自动拉取可用模型。 / 登录 Dim 以解锁模型。 | 空态（虚线框，按下文二选一） | — | 无模型（已配置 / 未配置） | `ConfigurationSettings.vue:3009-3017` | i18n:configurationSettings.modelsAutoLoadAfterSignIn · signInToUnlockModels |
| S-configuration-87 | 刷新 | 按钮（outline，sm，刷新图标） | — | Dim OAuth 或账户托管；未登录时禁用 | `ConfigurationSettings.vue:3020-3029` | i18n:common.refresh |
| S-configuration-88 | 查看套餐 | 按钮（outline，sm，外链图标） | — | 同上；未登录时禁用 | `ConfigurationSettings.vue:3030-3037` | i18n:accountSettings.viewPlan |

### 形态 E：通用托管 OAuth

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-configuration-89 | 登录 {provider} 以加载其模型。 | 未登录说明 | — | 状态 signed-out 或 error | `ConfigurationSettings.vue:3049-3053` | i18n:configurationSettings.oauthSignedOutDescription |
| S-configuration-90 | 使用设备代码登录 | 按钮（sm） | — | 登录方式包含 device | `ConfigurationSettings.vue:3060-3068` | i18n:configurationSettings.oauthSignInDevice |
| S-configuration-91 | 使用浏览器登录 | 按钮（sm，浏览器方式） | — | 登录方式包含 browser；未填回调地址时禁用 | `ConfigurationSettings.vue:3071-3080` | i18n:configurationSettings.oauthSignInBrowser |
| S-configuration-92 | 回调地址 | 标签 + 输入框（等宽） | 视脚本 | 浏览器方式 | `ConfigurationSettings.vue:3076-3080` | i18n:configurationSettings.oauthRedirectUri |
| S-configuration-93 | 正在等待授权... | 等待提示（设备流程） | — | 状态 pending-device | `ConfigurationSettings.vue:3085-3089` | i18n:configurationSettings.grokBuildWaitingForDevice |
| S-configuration-94 | 取消 | 按钮（outline，sm） | — | 同上 | `ConfigurationSettings.vue:3090-3097` | i18n:common.cancel |
| S-configuration-95 | 设备码 | 小字标签 | — | 设备流程且有 userCode | `ConfigurationSettings.vue:3096-3099` | i18n:configurationSettings.grokBuildDeviceCode |
| S-configuration-96 | {userCode} | 等宽验证码 | — | 同上 | `ConfigurationSettings.vue:3100-3101` | 运行期 |
| S-configuration-97 | 打开验证页面 | 按钮（outline，sm） | — | 设备流程 | `ConfigurationSettings.vue:3102-3110` | i18n:configurationSettings.grokBuildOpenVerification |
| S-configuration-98 | 复制 | 按钮（outline，sm） | — | 设备流程 | `ConfigurationSettings.vue:3111-3118` | i18n:common.copy |
| S-configuration-99 | 刷新 | 按钮（outline，sm） | — | 设备流程 | `ConfigurationSettings.vue:3119-3126` | i18n:common.refresh |
| S-configuration-100 | 请将浏览器地址栏中的 callback URL 复制到 DimAgent 完成登录。… | 等待提示（浏览器流程） | — | 状态 pending-browser | `ConfigurationSettings.vue:3123-3130` | i18n:configurationSettings.openAICodexWaitingForBrowser |
| S-configuration-101 | 取消 | 按钮（outline，sm） | — | 同上 | `ConfigurationSettings.vue:3131-3138` | i18n:common.cancel |
| S-configuration-102 | 打开 URL | 按钮（outline，sm） | — | 浏览器流程 | `ConfigurationSettings.vue:3141-3149` | i18n:configurationSettings.openAICodexOpenUrl |
| S-configuration-103 | 复制 | 按钮（outline，sm） | — | 浏览器流程 | `ConfigurationSettings.vue:3150-3157` | i18n:common.copy |
| S-configuration-104 | Callback URL | 标签 + 输入框（占位「粘贴完整 callback URL 或 query string」） | 空 | 浏览器流程 | `ConfigurationSettings.vue:3160-3173` | i18n:configurationSettings.openAICodexCallbackUrl · openAICodexCallbackPlaceholder |
| S-configuration-105 | 确认 | 提交按钮（sm） | — | 同上；空值或忙时禁用 | `ConfigurationSettings.vue:3174-3181` | i18n:common.confirm |
| S-configuration-106 | {redirect uri} | 小字回显 | — | 有 redirectUri | `ConfigurationSettings.vue:3182-3184` | 运行期 |
| S-configuration-107 | {账户标签} | 已连接卡标题 | — | 状态 authenticated | `ConfigurationSettings.vue:3193` | 运行期 |
| S-configuration-108 | 刷新 | 按钮（outline，sm） | — | 已连接 | `ConfigurationSettings.vue:3196-3204` | i18n:common.refresh |
| S-configuration-109 | 退出登录 | 按钮（outline，sm） | — | 已连接 | `ConfigurationSettings.vue:3205-3212` | i18n:accountSettings.signOut |
| S-configuration-110 | 已登录。模型开关决定聊天中显示哪些模型。 | 说明文案 | — | 已连接 | `ConfigurationSettings.vue:3214-3216` | i18n:configurationSettings.oauthAuthenticatedDescription |
| S-configuration-111 | 基础 URL | 标签 + 等宽只读值 | 供应商 baseUrl | 面板内常显 | `ConfigurationSettings.vue:3219-3224` | i18n:configurationSettings.baseUrl |
| S-configuration-112 | {OAuth 错误原文} | 错误条（destructive） | — | 登录/刷新失败 | `ConfigurationSettings.vue:3226-3229` | 运行期 |
| S-configuration-113 | 模型 · {可见数} / {总数} | 全选复选框 + 文案 | 未全选 | 面板内 | `ConfigurationSettings.vue:3237-3246` | i18n:configurationSettings.models |
| S-configuration-114 | 搜索模型名称或 ID | 搜索输入框 | 空 | 有模型 | `ConfigurationSettings.vue:3247-3255` | i18n:configurationSettings.searchModels |
| S-configuration-115 | {模型名} + 视觉/推理徽标 + 开关 | 模型行 | 按选择集 | 每行一个模型 | `ConfigurationSettings.vue:3260-3278` | i18n:configurationSettings.modelCapabilityVision · modelCapabilityReasoning |
| S-configuration-116 | 登录成功后将自动拉取可用模型。 | 空态（虚线框） | — | 无模型 | `ConfigurationSettings.vue:3279-3281` | i18n:configurationSettings.modelsAutoLoadAfterSignIn |

### 形态 F：未配置（setup 表单）

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-configuration-117 | Authentication | 标签（AWS Bedrock 专用，英文硬编码） | — | 供应商为 AWS Bedrock | `ConfigurationSettings.vue:3287` | 字面量（英文） |
| S-configuration-118 | API Key / Access Key | 下拉（2 选项） | API Key | 同上 | `ConfigurationSettings.vue:3288-3295` | 字面量（英文） |
| S-configuration-119 | Access Key ID | 标签 + 密码输入（占位） | 空 | Bedrock 且选 Access Key | `ConfigurationSettings.vue:3296-3300` | 字面量（英文） |
| S-configuration-120 | Secret Access Key | 标签 + 密码输入 | 空 | 同上 | `ConfigurationSettings.vue:3301-3305` | 字面量（英文） |
| S-configuration-121 | Session Token | 标签 + 密码输入 | 空 | 同上 | `ConfigurationSettings.vue:3306-3310` | 字面量（英文） |
| S-configuration-122 | API 密钥 | 标签（非 Bedrock 路径） | — | 未配置且非 Bedrock | `ConfigurationSettings.vue:3313` | i18n:providerSettings.apiKey |
| S-configuration-123 | sk-... | 密码输入框（占位，回车直连） | 空 | 同上 | `ConfigurationSettings.vue:3314-3319` | i18n:configurationSettings.apiKeyPlaceholder |
| S-configuration-124 | Region / 基础 URL | 标签 | — | 未配置 | `ConfigurationSettings.vue:3321` | 字面量（`Region`）· i18n:configurationSettings.baseUrl |
| S-configuration-125 | us-east-1（Region）/ {基础 URL} | 输入框（Region 占位 us-east-1；非 Bedrock 时不可编辑则置灰） | 空 | 同上 | `ConfigurationSettings.vue:3322-3334` | 字面量（`us-east-1`） |
| S-configuration-126 | {表单错误原文} | 错误条（destructive） | — | 校验失败 | `ConfigurationSettings.vue:3336-3344` | 运行期 |
| S-configuration-127 | 连接并获取模型 / 保存 | 按钮（整宽，主按钮，获取中转圈） | 连接并获取模型 | 未配置；不可连接时禁用 | `ConfigurationSettings.vue:3347-3352` | i18n:configurationSettings.connectAndFetchModels · i18n:common.save |

### 形态 G：已配置（凭据 + 用量 + 模型）

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-configuration-128 | 协议 | 标签 + 等宽只读值（自定义供应商） | 视供应商 | 自定义供应商 | `ConfigurationSettings.vue:3335-3340` | i18n:configurationSettings.protocol |
| S-configuration-129 | 名称 | 标签 + 输入框（自定义供应商可改） | 供应商名 | 自定义供应商 | `ConfigurationSettings.vue:3341-3344` | i18n:configurationSettings.name |
| S-configuration-130 | 基础 URL | 标签 + 等宽输入框 | 供应商 baseUrl | 自定义供应商 | `ConfigurationSettings.vue:3345-3348` | i18n:configurationSettings.baseUrl |
| S-configuration-131 | API 密钥 | 标签 + 密码输入（已存值时占位「已保存，如需更换请重新输入」） | 空 | 自定义供应商 | `ConfigurationSettings.vue:3349-3356` | i18n:providerSettings.apiKey · configurationSettings.savedReplacePrompt |
| S-configuration-132 | User-Agent | 标签 + 输入框 + 说明 | 空 | 自定义供应商 | `ConfigurationSettings.vue:3357-3361` | i18n:configurationSettings.userAgent · userAgentHint |
| S-configuration-133 | 取消 / 保存 | 两个按钮（连接信息有改动才启用取消/保存） | 保存 | 自定义供应商 | `ConfigurationSettings.vue:3362-3367` | i18n:common.cancel · save |
| S-configuration-134 | Authentication / API 密钥 | 标签（Bedrock 显示 Authentication） | — | 非自定义供应商 | `ConfigurationSettings.vue:3371-3374` | 字面量 · i18n:providerSettings.apiKey |
| S-configuration-135 | Access Key ID / Secret Access Key / Session Token (optional) | 密码输入（占位文案） | 空 | Bedrock 且编辑密钥并选 Access Key | `ConfigurationSettings.vue:3383-3388` | 字面量（英文） |
| S-configuration-136 | sk-... / 已保存，如需更换请重新输入 | 密码输入框（占位） | 空 | 编辑密钥且非 Bedrock 多字段 | `ConfigurationSettings.vue:3389-3395` | i18n:configurationSettings.apiKeyPlaceholder · savedReplacePrompt |
| S-configuration-137 | 取消 / 确认 | 编辑密钥的两个按钮 | — | 编辑密钥态 | `ConfigurationSettings.vue:3398-3403` | i18n:common.cancel · confirm |
| S-configuration-138 | Access Key · •••••••• / {掩码密钥} | 等宽掩码值 | — | 非编辑密钥态 | `ConfigurationSettings.vue:3406-3408` | 运行期（掩码为硬编码点号） |
| S-configuration-139 | 修改 | 按钮（outline，sm，进入编辑密钥） | — | 非编辑密钥态 | `ConfigurationSettings.vue:3409` | i18n:configurationSettings.edit |
| S-configuration-140 | Region / 基础 URL | 标签 | — | 已配置 | `ConfigurationSettings.vue:3414-3416` | 字面量 · i18n:configurationSettings.baseUrl |
| S-configuration-141 | us-east-1 / {基础 URL} + 确认 | 输入框 + 确认按钮（可编辑基础 URL 的供应商） | 当前值 | 基础 URL 可编辑 | `ConfigurationSettings.vue:3418-3448` | 字面量 · i18n:common.confirm |
| S-configuration-142 | {基础 URL} | 等宽只读值 | 当前值 | 基础 URL 不可编辑 | `ConfigurationSettings.vue:3450-3452` | 运行期 |
| S-configuration-143 | {表单错误原文} | 错误条（destructive） | — | 校验失败 | `ConfigurationSettings.vue:3454-3462` | 运行期 |
| S-configuration-144 | 用量 | 小标题（用量区块） | — | 该供应商有 usage 投影 | `ConfigurationSettings.vue:3485` | i18n:usageSettings.title |
| S-configuration-145 | 刷新 | 图标按钮（h-6 w-6，aria-label/title） | — | 同上；加载中/未启用/未配置时禁用 | `ConfigurationSettings.vue:3486-3494` | i18n:common.refresh |
| S-configuration-146 | 账户 | 标签 + 下拉（API Key / Team） | API Key | 供应商支持团队用量 | `ConfigurationSettings.vue:3497-3506` | i18n:configurationSettings.openAICodexAccount |
| S-configuration-147 | 基础 URL / API 密钥 | 标签 + 输入框（separate_api_key 模式） | 当前值 | 用量凭据模式为 separate_api_key | `ConfigurationSettings.vue:3508-3523` | i18n:configurationSettings.baseUrl · providerSettings.apiKey |
| S-configuration-148 | Organization ID / Project ID | 标签 + 输入框（zhipu_team 模式） | 当前值 | 用量凭据模式为 zhipu_team | `ConfigurationSettings.vue:3525-3535` | 字面量（英文） |
| S-configuration-149 | Access Key ID / Secret Access Key | 标签 + 密码输入（volcengine_aksk 模式） | 当前值 | 用量凭据模式为 volcengine_aksk | `ConfigurationSettings.vue:3537-3555` | 字面量（英文） |
| S-configuration-150 | 测试连接 | 按钮（outline，sm，测试中转圈） | — | 非 API Key 模式 | `ConfigurationSettings.vue:3561-3568` | i18n:toolsSettings.testConnection |
| S-configuration-151 | 保存 | 按钮（sm，保存中转圈） | — | 用量凭据非默认模式或支持团队用量 | `ConfigurationSettings.vue:3569-3572` | i18n:common.save |
| S-configuration-152 | {用量摘要} | 用量投影区（ProviderAccountUsageSummary：加载中/未配置/空/窗口行与百分比） | — | 有 displayedProviderUsage | `ConfigurationSettings.vue:3574-3578` · 组件 `provider/ProviderAccountUsageSummary.vue:44-76` | i18n:usageSettings.creditLoading · creditEmpty · inputArea.notConfigured · creditUsed · windowLabel · creditResets |
| S-configuration-153 | {用量错误原文} | 错误文案（destructive） | — | 用量查询失败 | `ConfigurationSettings.vue:3579` | 运行期 |
| S-configuration-154 | 模型 · {可见数} / {总数} | 全选复选框 + 文案 | 未全选 | 已配置，模型非空 | `ConfigurationSettings.vue:3591-3599` | i18n:configurationSettings.models |
| S-configuration-155 | 添加模型 | 按钮（outline，sm，加号） | — | 已配置 | `ConfigurationSettings.vue:3602-3609` | i18n:configurationSettings.addModel |
| S-configuration-156 | 刷新（远程模型列表） | 图标按钮（刷新图标） | — | 供应商支持远程模型列表 | `ConfigurationSettings.vue:3610-3620` | 无文案（图标按钮） |
| S-configuration-157 | 搜索模型名称或 ID | 搜索输入框（Esc 清空） | 空 | 有模型 | `ConfigurationSettings.vue:3622-3630` | i18n:configurationSettings.searchModels |
| S-configuration-158 | {模型名} + 自定义/工具/视觉/推理/媒体 徽标 + 上下文/输出 + 开关（点击进模型编辑） | 模型行 | 按选择集 | 每行一个模型 | `ConfigurationSettings.vue:3640-3688` | i18n:configurationSettings.modelCustomized · modelCapability* · modelContextK · modelOutputK |
| S-configuration-159 | 没有匹配的模型 | 空态 | — | 搜索无匹配 | `ConfigurationSettings.vue:3689-3691` | i18n:configurationSettings.noMatchingModels |
| S-configuration-160 | 还没有模型数据，点击「刷新」获取 | 空态（虚线框） | — | 无模型 | `ConfigurationSettings.vue:3693-3699` | i18n:configurationSettings.noModelsYet |
| S-configuration-161 | 断开此提供商 | 按钮（ghost，红色，垃圾桶） | — | 预置供应商、非账户托管、可管理 | `ConfigurationSettings.vue:3705-3714` | i18n:configurationSettings.disconnectProvider |
| S-configuration-162 | 断开提供商 | 确认弹窗标题 | — | 点「断开此提供商」 | `ConfigurationSettings.vue:3727` | i18n:configurationSettings.disconnectProviderTitle |
| S-configuration-163 | 确定要断开 "{name}" 吗？将清除已保存的 API 密钥和模型列表，提供商配置仍会保留以便之后重新连接。 | 确认弹窗描述 | — | 同上 | `ConfigurationSettings.vue:3729-3731` | i18n:configurationSettings.disconnectProviderDescription |
| S-configuration-164 | 取消 / 断开连接 | 弹窗两个按钮 | — | 同上 | `ConfigurationSettings.vue:3733-3734` | i18n:common.cancel · configurationSettings.disconnect |
| S-configuration-165 | 删除此提供商 | 按钮（ghost，红色，垃圾桶） | — | 自定义供应商 | `ConfigurationSettings.vue:3745-3754` | i18n:configurationSettings.deleteProvider |
| S-configuration-166 | 删除提供商 | 确认弹窗标题 | — | 点「删除此提供商」 | `ConfigurationSettings.vue:3748-3750` | i18n:configurationSettings.deleteProviderTitle |
| S-configuration-167 | 确定要删除 "{name}" 吗？删除后它会从提供商列表和聊天模型列表中移除。 | 确认弹窗描述 | — | 同上 | `ConfigurationSettings.vue:3751-3753` | i18n:configurationSettings.deleteProviderDescription |
| S-configuration-168 | 取消 / 删除 | 弹窗两个按钮 | — | 同上 | `ConfigurationSettings.vue:3763-3764` | i18n:common.cancel · i18n:common.delete |
| S-configuration-169 | 选择一个提供商查看详情 | 空态文案（居中，h-32） | — | 详情面板无选中供应商 | `ConfigurationSettings.vue:3766-3770` | i18n:configurationSettings.selectProviderDetail |

### 「添加模型」命令面板（CommandDialog）

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-configuration-170 | 添加模型 | 命令面板标题 | — | 点「添加模型」 | `ConfigurationSettings.vue:3775-3779` | i18n:configurationSettings.addModel |
| S-configuration-171 | 搜索模型... | 面板描述 + 搜索输入框占位 | 空 | 同上 | `ConfigurationSettings.vue:3778` · `:3781-3785` | i18n:inputArea.searchModels |
| S-configuration-172 | 添加自定义模型 | 首条命令项（加号图标） | 选中第 0 项 | 面板打开 | `ConfigurationSettings.vue:3787-3801` | i18n:configurationSettings.addCustomModel |
| S-configuration-173 | 从空白配置开始；搜索内容会预填为模型 ID。 | 项说明 | — | 同上 | `ConfigurationSettings.vue:3798` | i18n:configurationSettings.addCustomModelDescription |
| S-configuration-174 | 使用模板创建 | 分组小标题 | — | 面板打开 | `ConfigurationSettings.vue:3803` | i18n:configurationSettings.createFromTemplate |
| S-configuration-175 | 选择预设配置作为起点，保存前仍可修改所有字段。 | 分组说明 | — | 同上 | `ConfigurationSettings.vue:3804-3806` | i18n:configurationSettings.createFromTemplateDescription |
| S-configuration-176 | 没有匹配的模型 | 空态 | — | 有搜索词且无匹配模板 | `ConfigurationSettings.vue:3808-3813` | i18n:configurationSettings.subagentModelNoMatches |
| S-configuration-177 | {供应商名} + {模板数} | 供应商分组按钮（可折叠，aria-expanded） | 展开状态视脚本 | 每个供应商一组 | `ConfigurationSettings.vue:3819-3837` | 运行期 |
| S-configuration-178 | {模型名} + {model.id} · 工具/视觉/推理/媒体 | 模板项（CommandItem，缩进） | — | 该组合展开 | `ConfigurationSettings.vue:3838-3858` | i18n:configurationSettings.modelCapability* |

### 「模型编辑」弹窗

| 元素 ID | 名称（界面文案原文） | 控件 | 默认 | 条件 | 定位 | 证据 |
|---|---|---|---|---|---|---|
| S-configuration-179 | 修改模型 / 添加模型 | 弹窗标题 | 修改模型 | 打开模型编辑 | `ConfigurationSettings.vue:3870-3872` | i18n:configurationSettings.editModel · addModel |
| S-configuration-180 | {模板来源 / 供应商名 / 表单名称} | 弹窗描述 | — | 同上 | `ConfigurationSettings.vue:3874-3876` | 运行期 |
| S-configuration-181 | 模型 ID | 标签 + 输入框（编辑态禁用） | 当前值 | 弹窗打开 | `ConfigurationSettings.vue:3882-3884` | i18n:configurationSettings.modelId |
| S-configuration-182 | 显示名称 | 标签 + 输入框 | 当前值 | 同上 | `ConfigurationSettings.vue:3885-3887` | i18n:configurationSettings.modelDisplayName |
| S-configuration-183 | 上下文长度 | 标签 + 数字输入（inputmode=numeric） | 当前值 | 同上 | `ConfigurationSettings.vue:3888-3890` | i18n:configurationSettings.modelContextWindow |
| S-configuration-184 | 最大输出 | 标签 + 数字输入 | 当前值 | 同上 | `ConfigurationSettings.vue:3891-3893` | i18n:configurationSettings.modelMaxOutput |
| S-configuration-185 | 对话端点 | 标签 + 下拉（自动 + 各端点类型） | 自动 | New API 供应商且可编辑端点 | `ConfigurationSettings.vue:3899-3910` · 标签 `:212-223` | i18n:configurationSettings.modelChatEndpoint · i18n:inputArea.serviceTier.auto |
| S-configuration-186 | 已检测: {endpoints} | 说明文案 | — | 有已检测端点 | `ConfigurationSettings.vue:3912-3916` | i18n:configurationSettings.modelDetectedEndpoints |
| S-configuration-187 | 此处不支持: {endpoints} | 说明文案 | — | 有不支持端点 | `ConfigurationSettings.vue:3917-3921` | i18n:configurationSettings.modelUnsupportedEndpoints |
| S-configuration-188 | 工具 | 能力开关行（label + switch） | 视模型 | 弹窗打开 | `ConfigurationSettings.vue:3925-3927` | i18n:configurationSettings.modelCapabilityTools |
| S-configuration-189 | 视觉 | 能力开关行 | 视模型 | 同上 | `ConfigurationSettings.vue:3929-3931` | i18n:configurationSettings.modelCapabilityVision |
| S-configuration-190 | 推理 | 能力开关行 | 视模型 | 同上 | `ConfigurationSettings.vue:3933-3935` | i18n:configurationSettings.modelCapabilityReasoning |
| S-configuration-191 | 媒体能力 | 分组开关行（switch） | 关 | 弹窗打开 | `ConfigurationSettings.vue:3939-3941` | i18n:configurationSettings.modelCapabilityMedia |
| S-configuration-192 | 选择该模型可用于哪些媒体操作。 | 说明文案 | — | 同上 | `ConfigurationSettings.vue:3943-3945` | i18n:configurationSettings.modelCapabilityMediaDescription |
| S-configuration-193 | 图像 / 视频 / 音频 / 其他 | 媒体能力分组标签（4 组） | — | 媒体能力开关打开 | `ConfigurationSettings.vue:3948` · 分组 `:120-156` | i18n:configurationSettings.modelCapabilityGroup* |
| S-configuration-194 | 图像生成 / 图像编辑 / 图像增强 / 视频生成 / 视频编辑 / 视频增强 / 语音合成 / 语音转写 / 音乐生成 / 音频生成 / 素材生成 / 动作生成 | 媒体能力复选框项（label + checkbox） | 视模型 | 同上 | `ConfigurationSettings.vue:3951-3962` | i18n:configurationSettings.modalityCapability.* |
| S-configuration-195 | 思考强度 | 标签 | — | 推理开关打开 | `ConfigurationSettings.vue:3968` | i18n:inputArea.effort.submenuTitle |
| S-configuration-196 | 关闭 / 极低 / 低 / 中 / 高 / 极高 / 最高 | 强度按钮组（选中为主按钮） | 视模型 | 同上 | `ConfigurationSettings.vue:3970-3982` | i18n:inputArea.reasoning.* |
| S-configuration-197 | {模型表单错误原文} | 错误条（destructive） | — | 校验失败 | `ConfigurationSettings.vue:3985-3992` | i18n:configurationSettings.modelIdRequired · modelIdDuplicate · modelLimitInvalid · modelCapabilityMediaRequired · modelCapabilityMediaUnsupported |
| S-configuration-198 | 删除 | 按钮（ghost，红色，垃圾桶） | — | 编辑态且可删除该模型 | `ConfigurationSettings.vue:4001-4011` | i18n:common.delete |
| S-configuration-199 | 还原默认 | 按钮（ghost，回退图标） | — | 可还原（自定义过该模型） | `ConfigurationSettings.vue:4012-4021` | i18n:configurationSettings.resetModelConfig |
| S-configuration-200 | 取消 | 按钮（outline，sm） | — | 弹窗底部 | `ConfigurationSettings.vue:4024` | i18n:common.cancel |
| S-configuration-201 | 保存 | 按钮（sm，保存中转圈） | — | 弹窗底部 | `ConfigurationSettings.vue:4025-4028` | i18n:common.save |

本节元素数：201

本轮自检：加载骨架到左右分栏；左栏搜索（空/有词/清除）、两个分组折叠、每项 ●/○ 状态、「添加」入口显隐、窄屏返回；六种详情形态逐一走过——添加自定义（5 协议单选、5 个输入、User-Agent 说明、模型全选三态、添加/刷新/搜索/清除、模型行五类徽标与限额、两种空态、错误、取消/保存）；Codex OAuth（未登录/等待浏览器/已连接/已禁用四态 + 错误 + 模型区两空态）；Grok device flow（未登录/已禁用/等待设备/已连接 + 错误 + 无模型空态）；账户托管（两种横幅说明 + 模型区 + 两按钮）；通用托管 OAuth（四种状态 + 设备码/浏览器两套等待 + 已连接 + 模型空态）；未配置 setup（Bedrock 与普通两路、连接并获取/保存）；已配置（自定义与预置两路凭据编辑、基础 URL 可编辑/只读、用量区块五种凭据模式 + 测试/保存 + 用量摘要 + 错误、模型区四态、断开/删除两个确认弹窗）；无选中空态；「添加模型」命令面板（自定义项、模板分组折叠、无匹配空态、模板项徽标）；模型编辑弹窗（五个基础字段、New API 端点选择与已检测/不支持、三个能力开关、媒体能力分组 4×复选框、思考强度按钮组、错误、删除/还原默认/取消/保存）。

---

## 汇总（19 节全覆盖）

| # | 节（导航顺序） | 元素数 |
|---|---|---|
| — | S-00 设置壳层与导航（非 19 节之一） | 21 |
| 1 | account 账户 | 82 |
| 2 | usage 用量 | 55 |
| 3 | personalization 个性化 | 85 |
| 4 | appearance 外观 | 83 |
| 5 | general 通用 | 102 |
| 6 | keybindings 快捷键 | 19 |
| 7 | configuration 模型供应 | 201 |
| 8 | modality 多模态 | 79 |
| 9 | runtime 运行环境 | 34 |
| 10 | mcp MCP | 138 |
| 11 | notifications 通知 | 8 |
| 12 | pets 宠物 | 36 |
| 13 | sandbox 权限 | 40 |
| 14 | git Git | 24 |
| 15 | hooks Hooks | 65 |
| 16 | pluginSettings 插件设置 | 49 |
| 17 | externalAgents 外部 Agent | 19 |
| 18 | archived 归档会话 | 32 |
| 19 | about 关于 | 37 |
| — | **19 节小计** | **1188** |
| — | **含壳层合计** | **1209** |

覆盖情况：`components/settings/sections.ts` 的 19 个 `SettingsSectionId` 全部有对应小节；每节表格连续编号 1..N、无重号缺号；每节末有「本节元素数」与「本轮自检」。

口径执行记录：
- 下拉/选择器选项合并为一行，但全部选项文案逐项写在「名称」列（如协议 5 项、代码主题 11 项、打开应用 17 项、语言 11 项、皮肤位置 9 项）。
- S-00 保留并标注非 19 节之一。
- 共用组件结构只穷举一遍（`SubagentModelSetting` 弹窗、`AccountActivity`），实例分别占行并标差异。
- 分隔线与无文案容器不计；纯骨架占位保留 1 行并注明「无文案」。
- 第 7 列「证据」为固定列：`i18n:<键>` / `字面量` / `运行期`。
- 源码中存在但渲染端无引用的 i18n 键（如 `keybindingsSettings.testCapture`、`permissionsSettings.resetTitle`）在对应节的「本轮自检」里注明「无对应界面元素，未计入」，可追溯、不静默丢弃。
