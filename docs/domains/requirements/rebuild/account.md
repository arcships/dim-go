# 重建规格 · 设置 › 账户（首屏）

Status: draft v1（结构树 + 控件映射 + 缺口 + 验收对齐）
Owner: 重建映射规格（本轮）
输入: [`../inventory-settings-rt.md`](../inventory-settings-rt.md) §1 账户（S-account-01 … 112，112 元素）＋ [`../requirements-settings.md`](../requirements-settings.md) §1（R-ACC-001 … 1133，1133 条）＋ 参考产品 `views/settings/account/AccountSettings.vue`、`AccountSignedIn.vue`、`AccountSignedOut.vue`、`AccountPlanUsage.vue`、`components/account/AccountEmailLoginDialog.vue` ＋ mygo v0.8.3 `ui` 包
不做: 不写 Go 代码、不定交互终值（U3）、不改需求与清单、不提交结果包。

**先读这段**：本屏可重建。112 个元素全部能落到 mygo 控件或「容器 + 自绘 `Draw`」上；真正的缺口只有 **10 条**（§3.1），集中在四处——头像尺寸与配色、分档配色进度条、带自定义正文的确认框、按容器宽断点。另有 6 条实现约束（§3.3）不构成缺口，但会决定 view 层的写法。

---

## 0 口径

### 0.1 三源

| 源 | 位置 | 用途 |
|---|---|---|
| 清单 | `../inventory-settings-rt.md` §1.1–§1.9 | 元素 ID、名称、条件、模板行号、运行时包围盒 |
| 需求 | `../requirements-settings.md` §1.1–§1.15 | R-ACC-ID 与元素一一对应（1133 条） |
| 参考源码 | `<参考产品>/packages/desktop/packages/renderer/src/views/settings/account/` | 结构（DOM 层级）、Tailwind 类的确定值、逐字中文文案（`i18n/zh-CN.ts`） |
| 框架 | `mygo:ui/*` v0.8.3（构造器见 `ui/api_gen.go`，实测几何见 `docs/domains/U0-control-baseline.md` §1） | 控件能力、默认尺寸、可覆盖属性 |

### 0.2 尺寸与 token 的写法（依赖声明）

- **`design-baseline.md` 尚未生成**：本批重建规格里凡引用它定义的设计常量，一律写成 `DS.<名>` 占位，并在 [`§5`](#5-依赖与未定项) 登记依赖。占位不填猜测值。
- **已经能定的值直接引用现役文档，不用占位**：
  - 间距 → `docs/domains/U2-visual.md` §2.2 的 `space-1/2/3/4/6/8/12`（= `t.Space(1…12)`，基础 4）；
  - 字阶 → 同文 §1.1 的 `type-h2 / type-h3 / type-body / type-body-strong / type-secondary / type-secondary-strong / type-micro / type-code`；
  - 颜色 → 同文 §3.2 / §3.3 的 `bg / surface / surface-hover / surface-pressed / border / text / text-secondary / accent / accent-on / accent-ink / success / warning / danger / selection / focus`；
  - 控件默认几何 → `docs/domains/U0-control-baseline.md` §1（Button 288×29.31、内边距 6/14、圆角 6、描边 1；Switch 36×20；Text 单行行盒 15.31；`Theme.Radius=6`、`Spacing=4`）。
- 参考产品用 Tailwind 命名值给规格（`p-5`、`gap-3`、`text-sm`…），本规格把它落到 `space-N` / `type-*` 上；**落不下的一律标 `DS.<名>`**（例：卡片圆角 `rounded-lg` = 8，我们 `Theme.Radius` = 6 —— 差 2，由 design-baseline 裁决，见 D-02）。

### 0.3 运行时包围盒的读法

清单的位置列是**采集坐标**，不是设计值。三处交叉验证表明采集时视口处于缩放状态（身份卡 `p-5`=20 对应量测左边距 18；`text-xl` 行盒 28 对应量测 25；`h-8`=32 对应量测 28、`h-12`=48 对应量测 42）：

- 本规格**只把包围盒当相对锚点**（谁的左缘与谁对齐、谁的宽度是基准），不作为像素验收目标；
- 验收绝对像素一律回落到 `DS.*` 与 `U0-control-baseline` 的实测值；
- 内容列基准：清单记 x 257–998（内容列宽 741）；身份卡与套餐卡同宽 740、同左缘 257，**卡宽 = 内容列宽 − 0 或 − 1（边框取整）**。这条是本屏所有横向布局的基准线。

---

## 1 结构树

### 1.1 层级定义

| 层 | 名字 | 判据 | 数量 |
|---|---|---|---:|
| L0 | 屏幕 | 设置 › 账户 一节 | 1 |
| L1 | 区域 | 页面上可整块出现/消失、或带独立滚动与卡框的带 | 4（A–D） |
| L2 | 分组 | 卡内以分隔线、间距或「标题 + 内容」成组的块 | 18 |
| L3 | 元素 | 清单行（`S-account-NN`），与需求一一对应 | 112 |
| L4 | 元素内件 | 复合元素内部的构件（图标、值、计数朵），不单独出清单行 | 见各行标注 |

层数：**5 层（含屏幕）／核心 4 层（区域 → 分组 → 元素）**。

### 1.2 树

```
L0  屏幕 · 设置 › 账户                      [Column · 上下滚动 · Gap space-5 · PaddingY DS.page-pad]
│
├─ L1 A 页首带                              [Column · Gap space-1]
│   └─ L2 A.1 页面标题
│       └─ L3 S-account-01  「账户」                        ← Text / type-h2
│
├─ L1 B 已登录带  (cond: isSignedIn)        [Column · Gap space-5]
│   ├─ L2 B.1 到期横幅  (cond: planExpired) [Row · Wrap · GapX space-3 · GapY space-3 · Padding space-3/space-4 · Radius DS.card-r · Border 1 danger@α]
│   │   ├─ L3 S-account-02  容器·提示条（role=status）      ← Row + Role(RoleStatus)
│   │   ├─ L3 S-account-03  ↳ TriangleAlert                 ← Icon 16 danger
│   │   ├─ L3 S-account-04  「套餐已到期，Dim 托管模型暂不可用」  ← Text type-body-strong danger
│   │   ├─ L3 S-account-05  「Dim 托管模型权益已停止，订阅套餐后即可恢复使用。」 ← Text type-secondary danger@α
│   │   ├─ L3 S-account-06  「选购套餐」                     ← Button(outline) + Icon
│   │   └─ L3 S-account-07  ↳ ShoppingCart                   ← Icon 14
│   │
│   ├─ L2 B.2 身份卡                        [Row · Wrap · GapX space-4 · GapY space-3 · Padding space-5 · Radius DS.card-r · Border 1 border · Background surface]
│   │   ├─ L4 身份块  [Row · Grow 1 · MinWidth DS.identity-min · Gap space-4]
│   │   │   ├─ L3 S-account-08  身份卡（容器 / 分组语义）    ← Row + Role(RoleGroup)
│   │   │   ├─ L3 S-account-09  头像 initials                ← Avatar（覆盖 Size/Background）
│   │   │   └─ L4 文本列  [Column · Grow 1 · MinWidth 0]
│   │   │       ├─ L3 S-account-10  displayName              ← Text type-body-strong · SingleLine
│   │   │       ├─ L3 S-account-11  email                    ← Text type-secondary · SingleLine
│   │   │       └─ L4 UID 行  [Row · Gap space-1.5 · Margin(space-1.5,0,0,0)]
│   │   │           ├─ L3 S-account-12  「UID」              ← Text type-micro text-secondary
│   │   │           ├─ L3 S-account-13  uid 值（code 芯片）  ← Box + Text type-code mono
│   │   │           ├─ L3 S-account-14  复制 UID             ← ButtonBase(自绘) · Label/Tooltip
│   │   │           └─ L3 S-account-15  ↳ Copy               ← Icon 12
│   │   └─ L4 动作组  [Row · Wrap · GapX space-3 · GapY space-2 · Shrink 0]
│   │       ├─ L3 S-account-16  「查看套餐」+ 模型数         ← Button(outline) + Children 富标签
│   │       │   ├─ L3 S-account-17  ↳ ExternalLink           ← Icon 14
│   │       │   └─ L3 S-account-18  「· 32 个模型」          ← Text type-secondary text-secondary
│   │       └─ L3 S-account-19  「退出登录」                 ← Button(ghost) · Disabled(refreshing)
│   │
│   └─ L2 B.3 账号错误行  (cond: account.error)
│       └─ L3 S-account-20  account.error.message            ← Text type-secondary danger
│
├─ L1 B' 未登录带  (cond: !isSignedIn)      [Column · Gap space-5]
│   ├─ L2 B'.1 未登录卡                      [Column · Gap space-4 · Padding space-6 · Radius DS.card-r · Border 1 border · Background surface]
│   │   ├─ L4 标题组  [Column · Gap space-2]
│   │   │   ├─ L3 S-account-21  未登录卡（容器 / 分组语义）  ← Column + Role(RoleGroup)
│   │   │   ├─ L3 S-account-22  「登录 Dim」                 ← Text type-h3 · Level(3)
│   │   │   └─ L3 S-account-23  「登录后可解锁 Dim 托管模型。未登录时仍可在本地使用 DimAgent。」 ← Text type-body text-secondary
│   │   └─ L4 动作组  [Row · Wrap · Gap space-3]
│   │       ├─ L3 S-account-24  「正在登录…」                ← PrimaryButton + Spinner
│   │       ├─ L3 S-account-25  ↳ Loader2（旋转）            ← Spinner 16（不是静态 Icon）
│   │       ├─ L3 S-account-26  「登录」                     ← PrimaryButton
│   │       ├─ L3 S-account-27  「登录企业账号」             ← Button(outline)
│   │       └─ L3 S-account-28  account.error.message        ← Text type-secondary danger
│   └─ L2 B'.2 邮箱登录对话框  (cond: 点击 S-27)   [Modal · 面板 Width DS.dialog-w]
│       ├─ L3 S-account-29  邮箱登录对话框                   ← Modal + 自绘面板
│       └─ L4 面板内件：DialogTitle「登录企业账号」· DialogDescription「请输入您的企业邮箱账号」· Label「企业邮箱」+ TextInput(placeholder「name@company.com」)· 错误行 · PrimaryButton「登录企业账号」
│
├─ L1 C 套餐用量卡                          [Column · Radius DS.card-r · Border 1 border · Background surface · Clip]
│   ├─ L2 C.1 卡头                          [Row · Justify SpaceBetween · AlignItems Center · MinHeight DS.card-header-h · Padding space-3/space-5 · BorderWidth 0,0,1,0]
│   │   ├─ L3 S-account-30  套餐用量（卡容器 / 分组语义）    ← Column + Role(RoleGroup)
│   │   ├─ L3 S-account-31  「套餐用量」                     ← Text type-body-strong · Level(3)
│   │   ├─ L3 S-account-32  刷新                             ← ButtonBase 28×28 · Disabled(refreshing)
│   │   └─ L3 S-account-33  ↳ RefreshCw                      ← Icon 16 + Rotate(Loop)
│   ├─ L2 C.2 状态行（错误 / 加载）
│   │   ├─ L3 S-account-34  error                            ← Text type-secondary danger · Padding space-4/space-5
│   │   └─ L3 S-account-35  「正在加载用量…」                ← Text type-secondary text-secondary
│   ├─ L2 C.3 主体（量测 y 256–536）        [Column · Gap space-4 · Padding space-5/space-5]
│   │   ├─ L4 服务行            [Row · Wrap · GapX space-2.5 · GapY space-2 · AlignItems Center]
│   │   │   ├─ L3 S-account-36  套餐标识 BrandMark           ← Box 36×36 Radius 8 + Center + Icon 18
│   │   │   ├─ L3 S-account-37  「DimAgent」                 ← Text type-body-strong
│   │   │   ├─ L3 S-account-38  planSubtitle                 ← Text type-secondary text-secondary · SingleLine
│   │   │   ├─ L3 S-account-39  statusLabel（生效中）        ← Badge（覆盖 Padding/Background/TextColor）
│   │   │   ├─ L3 S-account-40  ↳ CircleAlert（到期支）      ← Icon 12
│   │   │   └─ L3 S-account-41  「到期 … · 还剩 … 天」       ← Text type-secondary text-secondary · MarginStart(Auto)
│   │   ├─ L4 重置段  (cond: resetsVisible) [Row · Wrap · Gap space-1 · Padding space-1 · Radius DS.chip-r-lg · Background surface-hover]
│   │   │   ├─ L3 S-account-42  「额度用量」（选中项样式）   ← Box 芯片（非控件）+ Text
│   │   │   ├─ L3 S-account-43  「重置当月用量」             ← ButtonBase + Icon + Text + 计数朵 · Disabled
│   │   │   ├─ L3 S-account-44  ↳ RotateCcw                  ← Icon 12
│   │   │   └─ L3 S-account-45  剩余次数（0）                ← Box 计数朵 + Text type-micro tnum
│   │   ├─ L4 重置确认对话框  (cond: 点击 S-43 → 打开)  [Modal + 自绘面板]
│   │   │   ├─ L3 S-account-46  重置确认对话框               ← Modal（不是 AlertDialog）
│   │   │   ├─ L3 S-account-47  确认标题（「重置当月用量？」）← Text type-h2
│   │   │   ├─ L3 S-account-48  「将消耗 1 次重置次数。」     ← Text type-body text-secondary
│   │   │   ├─ L3 S-account-49  批次列表项（v-for）          ← Column + Text type-secondary（每批 1 行）
│   │   │   ├─ L3 S-account-50  「暂无可用」                 ← Text type-secondary text-secondary
│   │   │   ├─ L3 S-account-51  「无法加载额度明细」         ← Text type-secondary danger
│   │   │   ├─ L3 S-account-52  「取消」                     ← Button
│   │   │   └─ L3 S-account-53  「使用」                     ← PrimaryButton · Disabled
│   │   ├─ L4 耗尽提示  (cond: showDepletedNotice) [Row · Gap space-3 · Padding space-3/space-4 · Radius DS.card-r · Border 1 danger@α]
│   │   │   ├─ L3 S-account-54  套餐额度已用完 提示条        ← Row + Role(RoleStatus) + 左侧 3px 条（见 C-02）
│   │   │   ├─ L3 S-account-55  ↳ CircleAlert                ← Icon 16 danger
│   │   │   ├─ L3 S-account-56  「套餐额度已用完」           ← Text type-body-strong danger
│   │   │   └─ L3 S-account-57  「当前套餐额度已全部使用，Dim 托管模型暂时不可用。」 ← Text type-secondary danger@α · LineHeight DS.relaxed
│   │   ├─ L4 不限量支  (cond: isUnlimited) [Column · Padding space-3.5/space-4 · Radius DS.card-r · Border 1 border@α]
│   │   │   ├─ L3 S-account-58  「额度用量」（不限量标签）   ← Row · Gap space-1.5 · Text type-secondary-strong
│   │   │   ├─ L3 S-account-59  ↳ CalendarDays               ← Icon 14 text-secondary
│   │   │   ├─ L3 S-account-60  「不限量」                   ← Text type-secondary-strong accent-ink
│   │   │   └─ L3 S-account-61  「当前套餐不限用量，无额度上限。」 ← Text type-secondary text-secondary
│   │   ├─ L4 月度额度支  (cond: creditsAvailable) [Column · Padding space-3.5/space-4 · Radius DS.card-r · Border 1 border@α]
│   │   │   ├─ L4 头行  [Row · AlignItems Baseline · Justify SpaceBetween · Gap space-3]
│   │   │   │   ├─ L3 S-account-62  月度额度卡              ← Column（上面这个框）
│   │   │   │   ├─ L3 S-account-63  「月度额度」            ← Row · Gap space-1.5 · Text type-secondary-strong
│   │   │   │   ├─ L3 S-account-64  ↳ CalendarDays          ← Icon 14 text-secondary
│   │   │   │   └─ L3 S-account-65  「{date} 重置」         ← Text type-micro text-secondary
│   │   │   ├─ L4 用量行  [Row · AlignItems Center · Gap space-3 · Margin(space-2.5,0,0,0)]
│   │   │   │   ├─ L3 S-account-66  月度额度值（10.8万 / 30万） ← Text type-body-strong tnum · Width DS.meter-amount-w
│   │   │   │   ├─ L3 S-account-67  月度进度条              ← Box + Draw（自绘，见 G-02）· Role(RoleProgress)
│   │   │   │   └─ L3 S-account-68  「已使用 36%」          ← Text type-secondary-strong tnum · Width DS.meter-pct-w
│   │   │   └─ L4 周期限制窗口  (cond: windowRows.length>0) [Column · Margin(space-3,0,0,0) · BorderWidth(1,0,0,0) · Padding(space-2.5,0,0,0)]
│   │   │       ├─ L3 S-account-69  「周期限制」            ← Text type-micro-strong
│   │   │       ├─ L3 S-account-70  「以下窗口包含在月度额度内」 ← Text type-micro text-secondary
│   │   │       └─ L4 窗口行 ×n  [Row · Gap space-2.5 · Margin(space-2,0,0,0)]
│   │   │           ├─ L3 S-account-71  row.label                ← Text type-secondary-strong · Width DS.window-label-w · SingleLine
│   │   │           ├─ L3 S-account-72  row.amountText           ← Text type-secondary tnum text-secondary
│   │   │           ├─ L3 S-account-73  窗口进度条               ← Box + Draw · Role(RoleProgress) · Height DS.meter-h-thin
│   │   │           ├─ L3 S-account-74  「已使用 {pct}%」        ← Text type-micro tnum · TextAlign End
│   │   │           └─ L3 S-account-75  row.resetLabel           ← Text type-micro text-secondary
│   │   ├─ L4 周期限制（无月度额度支）(cond: !creditsAvailable && windowRows.length>0)
│   │   │   └─ 窗口行 ×n（结构同上，进度条高用 DS.meter-h）：
│   │   │       ├─ L3 S-account-76  row.label                    ← Text type-secondary-strong · Width DS.window-label-w
│   │   │       ├─ L3 S-account-77  row.amountText               ← Text type-secondary tnum text-secondary
│   │   │       ├─ L3 S-account-78  进度条                       ← Box + Draw · Role(RoleProgress)
│   │   │       ├─ L3 S-account-79  「已使用 {pct}%」            ← Text type-micro tnum · TextAlign End
│   │   │       └─ L3 S-account-80  row.resetLabel               ← Text type-micro text-secondary
│   │   ├─ L4 功能额度支  (cond: featureMeterRows.length>0) [Column · Gap space-3.5 · Padding space-3.5/space-4 · Radius DS.card-r · Border 1 border@α]
│   │   │   ├─ L3 S-account-81  Web Search 卡（v-for 每项一框） ← Column（上面这个框）
│   │   │   ├─ L4 头行  [Row · Baseline · Justify SpaceBetween · Gap space-3]
│   │   │   │   ├─ L3 S-account-82  ↳ Globe                    ← Icon 14 text-secondary
│   │   │   │   ├─ L3 S-account-83  meter.label（Web Search）  ← Row · Gap space-1.5 · Text type-secondary-strong
│   │   │   │   └─ L3 S-account-84  meter.resetLabel           ← Text type-micro text-secondary
│   │   │   └─ L4 用量行  [Row · Gap space-3 · Margin(space-2.5,0,0,0)]
│   │   │       ├─ L3 S-account-85  meter.amountText（334 / 1,500 次） ← Text type-body-strong tnum · Width DS.meter-amount-w
│   │   │       ├─ L3 S-account-86  功能进度条                 ← Box + Draw · Role(RoleProgress)
│   │   │       └─ L3 S-account-87  「已使用 22%」             ← Text type-secondary-strong tnum · Width DS.meter-pct-w
│   │   ├─ L4 未订阅行  (cond: isUnsubscribed)
│   │   │   ├─ L3 S-account-88  「尚未订阅套餐」               ← Text type-body-strong
│   │   │   ├─ L3 S-account-89  「订阅套餐后即可解锁 Dim 托管模型。」 ← Text type-secondary text-secondary
│   │   │   ├─ L3 S-account-90  「查看套餐」                   ← Button(outline)
│   │   │   └─ L3 S-account-91  ↳ ExternalLink                 ← Icon 14
│   │   ├─ L3 S-account-92  「暂无用量数据」  (cond: !plan && !creditsAvailable) ← Text type-secondary text-secondary
│   │   └─ L4 CTA 行  (cond: plan) [Row · Justify End · Padding(space-1,0,0,0)]
│   │       ├─ L3 S-account-93  「续订 / 升级」                ← PrimaryButton + Icon
│   │       ├─ L3 S-account-94  ↳ ShoppingCart（到期支）        ← Icon 14
│   │       └─ L3 S-account-95  ↳ ArrowUpRight（生效支）        ← Icon 14
│   └─ （C 卡到此为止；卡外为下一区域）
│
└─ L1 D 推广邀请卡  (cond: referral.eligible)  [Column · Radius DS.card-r · Border 1 border · Background surface · Clip]
    ├─ L2 D.1 卡头  [Row · AlignItems Center · Gap space-2 · MinHeight DS.card-header-h · Padding space-3/space-5 · BorderWidth 0,0,1,0]
    │   ├─ L3 S-account-96  推广邀请卡（容器 / 分组语义）     ← Column + Role(RoleGroup)
    │   ├─ L3 S-account-97  ↳ Gift                            ← Icon 16 text-secondary
    │   └─ L3 S-account-98  「推广邀请」                      ← Text type-body-strong · Level(3)
    ├─ L2 D.2 规则说明  [Column · Padding space-5]
    │   ├─ L3 S-account-99  inviterRuleLabel                  ← Text type-secondary text-secondary · Unselectable
    │   ├─ L3 S-account-100 inviteeRuleLabel                  ← Text type-secondary text-secondary · Unselectable
    │   └─ L3 S-account-101 referralCampaignOffLabel          ← Text type-secondary text-secondary · Unselectable
    ├─ L2 D.3 邀请码块  [Column（左列） · Gap space-2.5；码行 Row · Gap space-2 · Padding space-1.5/space-3/space-1.5/space-1.5 · Radius DS.card-r · Border 1 border · Background surface@α]
    │   ├─ L3 S-account-102 「邀请码」                        ← Text type-micro text-secondary
    │   ├─ L3 S-account-103 affCode（LYIr，code）             ← Text type-code mono type-body-strong · SingleLine · Selectable
    │   ├─ L3 S-account-104 「复制」                          ← Button(ghost) · Height DS.chip-btn-h
    │   └─ L3 S-account-105 ↳ Copy                             ← Icon 12
    ├─ L2 D.4 邀请网址块  [Column · Gap space-2.5；码行结构同 D.3]
    │   ├─ L3 S-account-106 「邀请网址」                      ← Text type-micro text-secondary
    │   ├─ L3 S-account-107 inviteUrl（code）                 ← Text type-code mono type-secondary · SingleLine · Selectable
    │   ├─ L3 S-account-108 「复制」                          ← Button(ghost) · Height DS.chip-btn-h
    │   └─ L3 S-account-109 ↳ Copy                             ← Icon 12
    └─ L2 D.5 指标组  [Column（宽窗）／ Row 三列（窄窗） · Margin(space-3.5,0,0,0) · BorderWidth(1,0,0,0) 或 (0,0,0,1) · Padding(space-3,0,0,space-5) 或 (0,0,0,space-5)]
        ├─ L4 指标项  [Row · Justify SpaceBetween · AlignItems Center · MinHeight DS.metric-row-h · BorderWidth 0,0,1,0]
        │   ├─ L3 S-account-110 「已邀请」/「11 人」          ← Text type-micro text-secondary + Text type-body-strong tnum
        │   ├─ L3 S-account-111 「已付费」/「6 人」           ← 同上
        │   └─ L3 S-account-112 「累计奖励」/「9,125 Credits」← 同上
```

### 1.3 容器布局表

| 容器（层） | 方向 | 间距 | 内边距 | 对齐 | 尺寸 / 边界 | 依据 |
|---|---|---|---|---|---|---|
| L0 滚动视口 | Column | — | 0 | — | Fill；`Scroll`，滚动条随指针出现（框架默认行为，与参考一致） | 参考 `usage-scroll-viewport`；`mygo:docs/ui/scroll.md` §Scroll bars |
| L0 页面内容列 | Column | `space-5`（20） | `PaddingY DS.page-pad-y`（参考 `p-6`/`md:p-8` = 24/32）· 水平 0 | — | `MaxWidth DS.content-max`（参考 `max-w-6xl` = 1152）；卡宽 = 内容列宽 | 参考 `mx-auto w-full space-y-5 p-6 md:p-8` |
| B 已登录带 / B' 未登录带 | Column | `space-5` | 0 | — | — | 参考 `space-y-5` |
| B.1 到期横幅 | Row + `Wrap` | `GapX space-3` · `GapY space-3` | `space-3` 上下 / `space-4` 左右 | `AlignItems Center` | 与卡同宽；`Radius DS.card-r`、`Border(1, danger@α)`、`Background(danger@α)` | 参考 `flex flex-wrap items-center gap-3 rounded-lg border border-destructive/25 bg-destructive/5 px-4 py-3` |
| B.2 身份卡 | Row + `Wrap` | `GapX space-4` · `GapY space-3` | `space-5`（20） | `AlignItems Center` | `Radius DS.card-r`、`Border(1, border)`、`Background(surface)` | 参考 `rounded-lg border border-border bg-card p-5` |
| B.2 身份块 | Row | `Gap space-4` | 0 | `AlignItems Center` | `Grow 1`、`MinWidth DS.identity-min`（参考 `min-w-[12rem]` = 192） | 参考注释：面板宽度与窗口无关，必须有真实最小宽 |
| B.2 动作组 | Row + `Wrap` | `GapX space-3` · `GapY space-2` | 0 | `AlignItems Center` | `Shrink 0` | 参考 `flex shrink-0 flex-wrap items-center gap-x-3 gap-y-2` |
| B'.1 未登录卡 | Column | `space-4`（16） | `space-6`（24） | `AlignItems Start` | 同上卡框 | 参考 `space-y-4 rounded-lg border border-border bg-card p-6` |
| C 套餐用量卡 | Column | 0 | 0 | — | `Radius DS.card-r`、`Border(1, border)`、`Background(surface)`、`Clip` | 参考 `overflow-hidden rounded-lg border border-border bg-card` |
| C.1 卡头 | Row | `Gap space-3` | `space-3` 上下 / `space-5` 左右 | `Justify SpaceBetween` · `AlignItems Center` | `MinHeight DS.card-header-h`（参考 `min-h-14` = 56）、下边线 `BorderWidth(0,0,1,0)` `border@α` | 参考 `flex min-h-14 items-center justify-between gap-3 border-b border-border/70 px-5 py-3` |
| C.3 主体 | Column | `space-4`（16） | `space-5` / `space-5` | — | — | 参考 `space-y-4 px-5 py-5` |
| C.3 服务行 | Row + `Wrap` | `GapX space-2.5` · `GapY space-2` | 0 | `AlignItems Center` | — | 参考 `flex flex-wrap items-center gap-x-2.5 gap-y-2` |
| C.3 重置段 | Row + `Wrap` | `Gap space-1` | `space-1` | `AlignItems Center` | `Radius DS.chip-r-lg`（参考 `rounded-lg` 8）、`Background(surface-hover)`（参考 `bg-muted`） | 参考 `inline-flex flex-wrap items-center gap-1 rounded-lg bg-muted p-1` |
| C.3 额度框（月度 / 不限量 / 功能 / 无月度窗口） | Column | `space-3.5`（14，功能额度框）/ 项内自排 | `space-3.5` 上下 / `space-4` 左右 | — | `Radius DS.card-r`、`Border(1, border@α)`（参考 `border-border/70`） | 参考 `rounded-lg border border-border/70 px-4 py-3.5` |
| C.3 用量行 | Row | `Gap space-3` | 0 | `AlignItems Center` | 金额列 `Width DS.meter-amount-w`（参考 `9.5rem` = 152）、百分比列 `Width DS.meter-pct-w`（参考 `5rem` = 80）、进度条 `Grow 1` + `MinWidth 0` | 参考 `.usage-meter-amount{width:9.5rem}` / `.usage-meter-percent{width:5rem}`（注释：两处必须同值，否则两条进度条不对齐） |
| C.3 窗口行 | Row | `Gap space-2.5` | 0 | `AlignItems Center` | 标签 `Width DS.window-label-w`（参考 `w-24` = 96）、进度条 `Grow 1` | 参考 `mt-2 flex items-center gap-2.5` / `w-24 shrink-0 truncate` |
| C.3 CTA 行 | Row | — | `Padding(space-1,0,0,0)` | `Justify End` | — | 参考 `flex justify-end pt-1` |
| D 推广邀请卡 | Column | 0 | 0 | — | 同上卡框 | 参考 `overflow-hidden rounded-lg border border-border bg-card` |
| D.3/D.4 码行 | Row | `Gap space-2` | `space-1.5` 上下 / 右 `space-1.5` / 左 `space-3` | `AlignItems Center` | `Radius DS.card-r`、`Border(1, border)`、`Background(surface@α)`（参考 `bg-muted/40`） | 参考 `flex items-center gap-2 rounded-lg border border-border bg-muted/40 py-1.5 pl-3 pr-1.5` |
| D.5 指标组（宽窗） | Column（每项 Row） | — | `PaddingStart space-5` | — | 项 `MinHeight DS.metric-row-h`（参考 `min-h-11` = 44）、`BorderWidth(0,0,1,0)` | 参考 `lg:min-w-[11rem] lg:max-w-[20rem] lg:border-l lg:pl-5` |
| D.5 指标组（窄窗） | Row（`Grow 1` 三列） | `GapX space-4` | `Padding(space-3,0,0,0)` | — | 项内 `Column`、`Gap space-0.5` | 参考 `grid grid-cols-3 gap-x-4 border-t pt-3` |

---

## 2 控件映射表

### 2.1 通用约定

| 项 | 约定 |
|---|---|
| 主题 | `ui.Theme` 取 `docs/domains/U2-visual.md` §3.2/§3.3 的 17 个 token 映射（`Background/Surface/SurfaceHover/SurfacePressed/Border/Text/TextMuted/Accent/AccentHover/AccentPressed/AccentText/Success/Warning/Danger/Selection/Focus` + 常量 `accent-ink`）；`Radius`/`Spacing`/`FocusRingWidth`/`FocusRingOffset` 沿用框架默认（6/4/2/1） |
| 字号 | 一律用 `type-*` 的行号（`FontSize`），不写字面量 |
| 图标 | `ui.Icon(c, svg)`，默认高度 = 字号；本屏三档 `Icon 16`（块级）、`Icon 14`（按钮内）、`Icon 12`（微件内），用 `.Size(16,16)` 等固定 |
| 图标资产 | mygo 不自带图标集，lucide 的 MIT SVG 由 `//go:embed` 引入；本屏需要 12 个：`TriangleAlert / ShoppingCart / UserCircle / Copy / ExternalLink / Loader2 / RefreshCw / CircleAlert / RotateCcw / CalendarDays / Globe / Gift`（加 `ArrowUpRight` 共 13）—— 依赖见 D-03 |
| 按钮 | 框架只给 `Button`（描边 + 面）与 `PrimaryButton`（强调填充）；本屏还要 ghost（`退出登录`、`刷新`、两个`复制`）、outline（`选购套餐`、`查看套餐`、`登录企业账号`）、icon-only（`刷新`、`复制 UID`）→ 见 C-01 |
| 按钮内图标 + 文字 | `ui.Button(c, "").Children(func(){ Icon; Text(…).SingleLine() })`（`mygo:docs/ui/icon.md` §In buttons） |
| 可选中文本 | 身份名、邮箱、UID、邀请码、邀请网址 → 父容器 `.Selectable()`；次要文本（计数、用量、徽标、规则说明）`.Unselectable()`（U2 §3.6） |
| 数字 | 所有数值文本 `.FontFeatures("tnum")`（参考 `tabular-nums`） |
| 截断 | 参考 `truncate` → `.SingleLine()`；`line-clamp` 无 → 用 `.MaxLines(n)` |
| 状态区 | 参考 `role="status" aria-live="polite"` → `.Role(ui.RoleStatus)`（`mygo:ui/access.go`；播报行为见 `mygo:docs/ui/accessibility.md` §Landmarks, live regions） |
| 回调命名 | 沿用参考产品的方法名，去掉 Vue 前缀：`handleViewPlan` / `handleSignOut` / `handleCopyUID` / `handleCopy(text)` / `handleRefresh` / `handleResetDialogOpen` / `handleApplyReset` / `handlePersonalSignIn` / `handleEnterpriseSignIn` / `handleResetBanner`（无）|
| 外部打开 | `选购套餐`/`查看套餐`/`续订 / 升级` 用系统浏览器打开购买页（`ui.Link` 不够：外观是按钮）→ 按钮 `OnClick` 走宿主 open 外部 URL，并做连点去重（R-ACC-055） |

### 2.2 映射表（112 条）

| # | 元素 ID | 元素（逐字文案） | mygo 控件 | 关键属性 | 状态 / 回调 | 需求 ID |
|---:|---|---|---|---|---|---|
| 1 | S-account-01 | 「账户」 | `ui.Text` | `type-h2`；`.Role(ui.RoleHeading).Level(2)` | 常显；跟随语言 | R-ACC-001–009 |
| 2 | S-account-02 | 到期横幅 | `ui.Row` | `.Wrap().GapX(space-3).GapY(space-3).Padding(space-3, space-4).Radius(DS.card-r).Border(1, danger@α).Background(danger@α).Role(ui.RoleStatus)` | `isSignedIn && isPlanExpired` | R-ACC-010–021 |
| 3 | S-account-03 | ↳ TriangleAlert | `ui.Icon` | `.Size(16,16).TextColor(danger)`；不单列 label（装饰） | 随横幅同帧出现 | R-ACC-022–029 |
| 4 | S-account-04 | 「套餐已到期，Dim 托管模型暂不可用」 | `ui.Text` | `type-body-strong`；`danger`；`.SingleLine()`（窄窗 `.MaxLines(2)`） | 文案恒不插套餐名/日期 | R-ACC-030–037 |
| 5 | S-account-05 | 「Dim 托管模型权益已停止，订阅套餐后即可恢复使用。」 | `ui.Text` | `type-secondary`；`danger@α`；`.Selectable()` | 点按钮后文案不变 | R-ACC-038–045 |
| 6 | S-account-06 | 「选购套餐」 | `ui.Button`（outline 自绘，C-01） | `.Height(DS.btn-h-sm)`、`Padding(space-1.5, space-3)`、`Border(1, border)`、`Background(transparent)`；`FocusRing` 由框架给 | `OnClick → handleViewPlan`；悬停 `surface-hover`、按下 `surface-pressed` | R-ACC-046–057 |
| 7 | S-account-07 | ↳ ShoppingCart | `ui.Icon` | `.Size(14,14)`，颜色随按钮文字 | 不单独播报 | R-ACC-058–065 |
| 8 | S-account-08 | 身份卡 | `ui.Row` | `.Wrap().GapX(space-4).GapY(space-3).Padding(space-5).Radius(DS.card-r).Border(1, border).Background(surface).Role(ui.RoleGroup)` | 已登录常显；`.Selectable()`（卡内文本） | R-ACC-066–095 |
| 9 | S-account-09 | 头像 initials | `ui.Avatar(name, image)` + `.Size(DS.avatar, DS.avatar)` | 覆盖 `.Background(accent).TextColor(accent-on)`；参考 48 → `DS.avatar`（G-01） | 首字取显示名；空名走占位 | R-ACC-096–105 |
| 10 | S-account-10 | displayName（eric） | `ui.Text` | `type-body-strong`；`.SingleLine()`；`text` | 空值回落 `fallbackName`「Dim 账户」 | R-ACC-106–115 |
| 11 | S-account-11 | email | `ui.Text` | `type-secondary`；`text-secondary`；`.SingleLine()` | `v-if email`；`.Unselectable()` | R-ACC-116–124 |
| 12 | S-account-12 | 「UID」 | `ui.Text` | `type-micro`；`text-secondary`；`.Unselectable()` | `v-if uid` | R-ACC-125–132 |
| 13 | S-account-13 | uid 值 | `ui.Box` + `ui.Text` | Box：`.Padding(space-0.5, space-1.5).Radius(4).Border(1, border).Background(surface@α)`；Text：`.Font("SF Mono, Menlo, monospace").FontSize(type-code).SingleLine()` | 可选中；`.MinWidth(0)` 参与收缩 | R-ACC-133–142 |
| 14 | S-account-14 | 复制 UID | `ui.ButtonBase`（自绘 icon-only） | `.Size(18,18)`、`Radius(4)`、悬停 `surface-hover`、`.Label("复制 UID").Tooltip("复制 UID")` | `OnClick → handleCopyUID`；成功后 `c.Toast("已复制")`，失败 `c.AddToast({Type:"error",Title:"复制失败"})` | R-ACC-143–172 |
| 15 | S-account-15 | ↳ Copy | `ui.Icon` | `.Size(12,12)`；`text-secondary` → 悬停 `text` | 不单独播报 | R-ACC-173–180 |
| 16 | S-account-16 | 「查看套餐」 | `ui.Button`（outline）+ `.Children` | `.Height(DS.btn-h-sm)`、`Padding(space-1.5, space-3)`；内件顺序见 17/18 | `OnClick → handleViewPlan` | R-ACC-181–200 |
| 17 | S-account-17 | ↳ ExternalLink | `ui.Icon` | `.Size(14,14)`，色随按钮 | — | R-ACC-201–208 |
| 18 | S-account-18 | 「· 32 个模型」 | `ui.Text` ×2（`·` 与计数） | 均 `type-secondary`、`text-secondary`；`·` 标 `.AccessHidden(true)`（参考 `aria-hidden`） | `modelCount > 0` 才出现 | R-ACC-209–217 |
| 19 | S-account-19 | 「退出登录」 | `ui.Button`（ghost 自绘，C-01） | `.Height(DS.btn-h-sm)`、`:disabled=refreshing` → `.Disabled(true)`；禁用 0.5（框架） | `OnClick → handleSignOut` | R-ACC-218–237 |
| 20 | S-account-20 | account.error.message | `ui.Text` | `type-secondary`；`danger`；`.Margin(space-3,0,0,0)` | `v-if account.error`；不替换卡片 | R-ACC-238–247 |
| 21 | S-account-21 | 未登录卡 | `ui.Column` | `.Gap(space-4).Padding(space-6).Radius(DS.card-r).Border(1, border).Background(surface).Role(ui.RoleGroup)` | `!isSignedIn` | R-ACC-248–257 |
| 22 | S-account-22 | 「登录 Dim」 | `ui.Text` | `type-h3`；`.Role(ui.RoleHeading).Level(3)` | 常显（本卡内） | R-ACC-258–265 |
| 23 | S-account-23 | 「登录后可解锁 Dim 托管模型。未登录时仍可在本地使用 DimAgent。」 | `ui.Text` | `type-body`；`text-secondary` | — | R-ACC-266–274 |
| 24 | S-account-24 | 「正在登录…」 | `ui.PrimaryButton` + `Children` | `.Disabled(signingIn)`；内件：`Spinner(16)` + `Text("正在登录…").SingleLine()` | `signingIn` 支；与 26 互斥 | R-ACC-275–286 |
| 25 | S-account-25 | ↳ Loader2 | `ui.Spinner` | `.Size(16,16).Label("正在登录")` | 持续旋转（系统降动效时仍转） | R-ACC-287–294 |
| 26 | S-account-26 | 「登录」 | `ui.PrimaryButton` | `.Disabled(signingIn)` | `OnClick → handlePersonalSignIn` | R-ACC-295–314 |
| 27 | S-account-27 | 「登录企业账号」 | `ui.Button`（outline） | `.Disabled(signingIn)` | `OnClick → 打开 S-29` | R-ACC-315–329 |
| 28 | S-account-28 | account.error.message | `ui.Text` | `type-secondary`；`danger` | `v-if account.error` | R-ACC-330–339 |
| 29 | S-account-29 | 邮箱登录对话框 | `ui.Modal` + `.Children` 自绘面板 | 面板 `.Width(DS.dialog-w)`、`Radius(DS.card-r)`、`Padding(space-6)`、`Background(bg)`、`Shadow(0,8,24,0,…)`；内件：`Text("登录企业账号") type-h3` / `Text("请输入您的企业邮箱账号") type-body text-secondary` / `Text("企业邮箱") type-body-strong` / `TextInput(placeholder "name@company.com")` / 错误 `Text type-secondary danger` / `PrimaryButton("登录企业账号")` | 打开清空输入；Enter 提交；无取消按钮（参考如此） | R-ACC-340–356 |
| 30 | S-account-30 | 套餐用量（卡容器） | `ui.Column` | `.Radius(DS.card-r).Border(1, border).Background(surface).Clip().Role(ui.RoleGroup)` | 常显（已登录） | R-ACC-357–386 |
| 31 | S-account-31 | 「套餐用量」 | `ui.Text` | `type-body-strong`；`.Role(ui.RoleHeading).Level(3)` | — | R-ACC-387–394 |
| 32 | S-account-32 | 刷新 | `ui.ButtonBase`（icon-only） | `.Size(28,28)`（参考 `h-8 w-8`）、`Radius(4)`、`.Label("刷新").Tooltip("刷新")`；`.Disabled(refreshing)` | `OnClick → handleRefresh`；最短旋转时长由上层保 | R-ACC-395–410 |
| 33 | S-account-33 | ↳ RefreshCw | `ui.Icon` | `.Size(16,16)`；刷新中 `.Rotate(e.Loop("spin", time.Second, ui.Linear) * 360)` | 刷新中转，静止时不转 | R-ACC-411–418 |
| 34 | S-account-34 | error | `ui.Text` | `type-secondary`；`danger`；`.Padding(space-4, space-5)` | `v-if error` | R-ACC-419–427 |
| 35 | S-account-35 | 「正在加载用量…」 | `ui.Text` | `type-secondary`；`text-secondary`；同上内边距 | `loading && !usage` | R-ACC-428–435 |
| 36 | S-account-36 | 套餐标识 BrandMark | `ui.Box` + `.Center()` + `ui.Icon` | Box `.Size(36,36).Radius(DS.chip-r-lg).Background(accent@α)`；到期支换 `Background(surface-hover)` + `TextColor(text-secondary)` | 随套餐到期切色 | R-ACC-436–443 |
| 37 | S-account-37 | 「DimAgent」 | `ui.Text` | `type-body-strong` | 不翻译 | R-ACC-444–451 |
| 38 | S-account-38 | planSubtitle（法至内部使用 · ¥0.10 / 月） | `ui.Text` | `type-secondary`；`text-secondary`；`.SingleLine().MinWidth(0)` | 无副标题不渲染 | R-ACC-452–460 |
| 39 | S-account-39 | statusLabel（生效中） | `ui.Badge` | 覆盖 `.Padding(space-0.5, space-2)`、`.Radius(999)`、`.Background(accent@α).TextColor(accent-ink)`；到期支 `danger@α` / `danger`；徽标字号 = 框架 `FontSize-2`（= 11） | `statusLabel` 非空 | R-ACC-461–472 |
| 40 | S-account-40 | ↳ CircleAlert（到期支） | `ui.Icon` | `.Size(12,12)`；`danger` | `isPlanExpired` | R-ACC-473–480 |
| 41 | S-account-41 | 「到期 2026年12月13日 · 还剩 26 天」 | `ui.Text` | `type-secondary`；`text-secondary`；`.MarginStart(ui.Auto)`；到期支文案「已于 … 到期」 | 到期 / 生效两支；`.SingleLine()` | R-ACC-481–490 |
| 42 | S-account-42 | 「额度用量」（选中项样式） | `ui.Box` + `ui.Text` | Box `.Padding(space-1, space-2.5).Radius(DS.chip-r).Background(surface).Shadow(0,1,2,0,…)`；Text `type-secondary-strong`；**不是** `Segmented`（参考是静态 `span`，不是可切换项） | `resetsVisible` | R-ACC-491–500 |
| 43 | S-account-43 | 「重置当月用量」 | `ui.ButtonBase` + `Children` | `.Padding(space-1, space-2.5).Radius(DS.chip-r)`；内件：Icon 12 + Text + 计数朵；`.Disabled(不可用).Tooltip(resetRowHint)`；悬停 `surface@α` + `text` | `OnClick → handleResetDialogOpen(row)`；`v-for` 每种重置类型一颗 | R-ACC-501–530 |
| 44 | S-account-44 | ↳ RotateCcw | `ui.Icon` | `.Size(12,12)` | 色随胶囊文字 | R-ACC-531–538 |
| 45 | S-account-45 | 剩余次数（0） | `ui.Box` + `ui.Text` | Box `.MinWidth(18).Padding(0, space-1.5).Radius(999)`；`Background(accent@α)`（>0）/`Background(border@α)`（=0）；临界到期 `.Border(1, danger@α)`（C-05）；Text `type-micro` + `.Bold().FontFeatures("tnum")` | 值随 `row.availableCount` | R-ACC-539–546 |
| 46 | S-account-46 | 重置确认对话框 | `ui.Modal`（**不用** `AlertDialog`，G-03） | 面板同 29；`.Modal()` 语义 + 焦点圈在面板内 | `openResetType === row.type` | R-ACC-547–554 |
| 47 | S-account-47 | row.confirmTitle（「重置当月用量？」） | `ui.Text` | `type-h2`；`.Role(ui.RoleHeading).Level(2)` | 逐类型换标题 | R-ACC-555–562 |
| 48 | S-account-48 | 「将消耗 1 次重置次数。」 | `ui.Text` | `type-body`；`text-secondary` | — | R-ACC-563–570 |
| 49 | S-account-49 | batchLabel(batch)（v-for） | `ui.Column` + `ui.Text` | Column `.Gap(space-1)`；Text `type-secondary`；每批一行 | `resetBatches.length > 0` | R-ACC-571–582 |
| 50 | S-account-50 | 「暂无可用」 | `ui.Text` | `type-secondary`；`text-secondary` | `resetBatches` 已回且为空 | R-ACC-583–590 |
| 51 | S-account-51 | 「无法加载额度明细」 | `ui.Text` | `type-secondary`；`danger` | `resetBatchesFailed` | R-ACC-591–598 |
| 52 | S-account-52 | 「取消」 | `ui.Button` | 与 53 同排右对齐（`Justify End`、`Gap space-2`） | 关闭对话框 | R-ACC-599–606 |
| 53 | S-account-53 | 「使用」 | `ui.PrimaryButton` | `.Disabled(resetApplyDisabled())`；默认按钮（Enter 触发） | `OnClick → handleApplyReset(row.type)` | R-ACC-607–618 |
| 54 | S-account-54 | 套餐额度已用完 提示条 | `ui.Row` | `.Gap(space-3).Padding(space-3, space-4).Radius(DS.card-r).Border(1, danger@α).Background(danger@α).Role(ui.RoleStatus)`；左侧 3px 条见 C-02 | `showDepletedNotice` | R-ACC-619–628 |
| 55 | S-account-55 | ↳ CircleAlert | `ui.Icon` | `.Size(16,16)`；`danger`；`.Margin(2,0,0,0)`（参考 `mt-0.5`） | 随提示条 | R-ACC-629–636 |
| 56 | S-account-56 | 「套餐额度已用完」 | `ui.Text` | `type-body-strong`；`danger` | — | R-ACC-637–644 |
| 57 | S-account-57 | 「当前套餐额度已全部使用，Dim 托管模型暂时不可用。」 | `ui.Text` | `type-secondary`；`danger@α`；`.LineHeight(DS.relaxed)`（参考 `leading-relaxed`） | — | R-ACC-645–652 |
| 58 | S-account-58 | 「额度用量」（不限量支） | `ui.Row` + `ui.Text` | Row `.Gap(space-1.5).AlignItems(Center)`；Text `type-secondary-strong` | `isUnlimited` | R-ACC-653–660 |
| 59 | S-account-59 | ↳ CalendarDays | `ui.Icon` | `.Size(14,14)`；`text-secondary` | — | R-ACC-661–668 |
| 60 | S-account-60 | 「不限量」 | `ui.Text` | `type-secondary-strong`；`accent-ink`；`.MarginStart(ui.Auto)`（参考 `justify-between`） | — | R-ACC-669–676 |
| 61 | S-account-61 | 「当前套餐不限用量，无额度上限。」 | `ui.Text` | `type-secondary`；`text-secondary`；`.Margin(space-1.5,0,0,0)` | — | R-ACC-677–684 |
| 62 | S-account-62 | 月度额度卡 | `ui.Column` | `.Padding(space-3.5, space-4).Radius(DS.card-r).Border(1, border@α)` | `creditsAvailable` | R-ACC-685–694 |
| 63 | S-account-63 | 「月度额度」 | `ui.Row` + `ui.Text` | Row `.Gap(space-1.5).AlignItems(Center)`；Text `type-secondary-strong` | — | R-ACC-695–702 |
| 64 | S-account-64 | ↳ CalendarDays | `ui.Icon` | `.Size(14,14)`；`text-secondary` | — | R-ACC-703–710 |
| 65 | S-account-65 | 「2026年12月13日 重置」 | `ui.Text` | `type-micro`；`text-secondary`；`.SingleLine()` | `creditResetLabel` 非空 | R-ACC-711–718 |
| 66 | S-account-66 | 月度额度值（10.8万 / 30万） | `ui.Text` | `type-body-strong`；`.FontFeatures("tnum").Width(DS.meter-amount-w).SingleLine()` | `monthlyAmountText` 非空 | R-ACC-719–726 |
| 67 | S-account-67 | 月度进度条（已使用 36%） | `ui.Box` + `.Draw`（自绘，G-02） | `.Height(DS.meter-h).Radius(999).Background(DS.meter-track).Clip().Grow(1).MinWidth(0)`；`Draw` 内 `p.Fill(fill, 999)`；`.Role(ui.RoleProgress).Range(0,100,pct).Label("已使用 36%")` | 填充色：`normal`→`accent`、`warning`→`warning`、`critical/depleted`→`danger`；宽度动画 200ms（`mygo:docs/ui/transitions.md`） | R-ACC-727–756 |
| 68 | S-account-68 | 「已使用 36%」 | `ui.Text` | `type-secondary-strong`；`.FontFeatures("tnum").Width(DS.meter-pct-w).TextAlign(ui.End).SingleLine()`；色按档（`accent-ink`/`warning`/`danger`） | 随档位换色 | R-ACC-757–766 |
| 69 | S-account-69 | 「周期限制」 | `ui.Text` | `type-micro` + `.Bold()` | `windowRows.length > 0` | R-ACC-767–774 |
| 70 | S-account-70 | 「以下窗口包含在月度额度内」 | `ui.Text` | `type-micro`；`text-secondary` | 同上 | R-ACC-775–782 |
| 71 | S-account-71 | row.label（如「5 小时窗口」） | `ui.Text` | `type-secondary-strong`；`.Width(DS.window-label-w).SingleLine()` | `v-for` 每窗口一行 | R-ACC-783–790 |
| 72 | S-account-72 | row.amountText | `ui.Text` | `type-secondary`；`text-secondary`；`.FontFeatures("tnum").SingleLine()` | `row.amountText` 非空 | R-ACC-791–798 |
| 73 | S-account-73 | 窗口进度条 | `ui.Box` + `.Draw` | `.Height(DS.meter-h-thin).Radius(999).Background(DS.meter-track).Grow(1)`；`.Role(ui.RoleProgress).Range(0,100,row.percent).Label(row.label)` | 同 67 的分档配色 | R-ACC-799–806 |
| 74 | S-account-74 | 「已使用 {pct}%」 | `ui.Text` | `type-micro`；`.FontFeatures("tnum").TextAlign(ui.End).SingleLine()`；色按档 | — | R-ACC-807–814 |
| 75 | S-account-75 | row.resetLabel | `ui.Text` | `type-micro`；`text-secondary`；`.SingleLine()` | `row.resetLabel` 非空 | R-ACC-815–822 |
| 76 | S-account-76 | row.label（无月度额度支） | `ui.Text` | 同 71 | `!creditsAvailable && windowRows.length > 0` | R-ACC-823–830 |
| 77 | S-account-77 | row.amountText（同上） | `ui.Text` | 同 72 | 同上 | R-ACC-831–838 |
| 78 | S-account-78 | 进度条（同上） | `ui.Box` + `.Draw` | 同 73，高度用 `DS.meter-h`（参考 `h-1.5`） | 同上 | R-ACC-839–846 |
| 79 | S-account-79 | 「已使用 {pct}%」（同上） | `ui.Text` | 同 74 | 同上 | R-ACC-847–854 |
| 80 | S-account-80 | row.resetLabel（同上） | `ui.Text` | 同 75 | 同上 | R-ACC-855–862 |
| 81 | S-account-81 | Web Search 卡 | `ui.Column`（`v-for` 每 meter 一框） | `.Gap(space-3.5).Padding(space-3.5, space-4).Radius(DS.card-r).Border(1, border@α)` | `featureMeterRows.length > 0` | R-ACC-863–872 |
| 82 | S-account-82 | ↳ Globe | `ui.Icon` | `.Size(14,14)`；`text-secondary` | — | R-ACC-873–880 |
| 83 | S-account-83 | meter.label（Web Search） | `ui.Row` + `ui.Text` | 同 63 结构；Text `type-secondary-strong` | — | R-ACC-881–888 |
| 84 | S-account-84 | meter.resetLabel | `ui.Text` | 同 65 | 非空才渲染 | R-ACC-889–896 |
| 85 | S-account-85 | meter.amountText（334 / 1,500 次） | `ui.Text` | 同 66 | 非空才渲染 | R-ACC-897–904 |
| 86 | S-account-86 | 功能进度条 | `ui.Box` + `.Draw` | 同 67；`meter.unlimited` 时不渲染进 87 | — | R-ACC-905–912 |
| 87 | S-account-87 | 「已使用 22%」 | `ui.Text` | 同 68 | — | R-ACC-913–922 |
| 88 | S-account-88 | 「尚未订阅套餐」 | `ui.Text` | `type-body-strong`（参考 `text-sm font-medium`） | `isUnsubscribed` | R-ACC-923–930 |
| 89 | S-account-89 | 「订阅套餐后即可解锁 Dim 托管模型。」 | `ui.Text` | `type-secondary`；`text-secondary` | 同上 | R-ACC-931–938 |
| 90 | S-account-90 | 「查看套餐」 | `ui.Button`（outline）+ `Children` | 同 16 的尺寸与内件顺序 | `OnClick → handleViewPlan` | R-ACC-939–948 |
| 91 | S-account-91 | ↳ ExternalLink | `ui.Icon` | `.Size(14,14)` | — | R-ACC-949–956 |
| 92 | S-account-92 | 「暂无用量数据」 | `ui.Text` | `type-secondary`；`text-secondary` | `!plan && !creditsAvailable` | R-ACC-957–964 |
| 93 | S-account-93 | 「续订 / 升级」 | `ui.PrimaryButton` + `Children` | 内件：Icon 14 + `Text(…).SingleLine()`（`plan` 存在才渲染整行） | `OnClick → handleRenewPlan`（去 pricing 页） | R-ACC-965–974 |
| 94 | S-account-94 | ↳ ShoppingCart（到期支） | `ui.Icon` | `.Size(14,14)` | `isPlanExpired` | R-ACC-975–982 |
| 95 | S-account-95 | ↳ ArrowUpRight（生效支） | `ui.Icon` | `.Size(14,14)` | `!isPlanExpired` | R-ACC-983–990 |
| 96 | S-account-96 | 推广邀请卡 | `ui.Column` | `.Radius(DS.card-r).Border(1, border).Background(surface).Clip().Role(ui.RoleGroup)` | `referral.eligible` | R-ACC-991–1000 |
| 97 | S-account-97 | ↳ Gift | `ui.Icon` | `.Size(16,16)`；`text-secondary` | — | R-ACC-1001–1008 |
| 98 | S-account-98 | 「推广邀请」 | `ui.Text` | `type-body-strong`；`.Role(ui.RoleHeading).Level(3)` | — | R-ACC-1009–1016 |
| 99 | S-account-99 | inviterRuleLabel（「好友完成首次订阅付费后，你获得其所购套餐额度 {percent}% 的 Credits。」） | `ui.Text` | `type-secondary`；`text-secondary`；`.Unselectable()` | 非空才渲染 | R-ACC-1017–1024 |
| 100 | S-account-100 | inviteeRuleLabel（「好友首次订阅可获得所购套餐额度 {percent}% 的 Credits。」） | `ui.Text` | 同 99 | 非空才渲染 | R-ACC-1025–1032 |
| 101 | S-account-101 | referralCampaignOffLabel（「推广活动当前未开启，分享链接仍会把好友的注册记在你名下。」） | `ui.Text` | 同 99 | `referralCampaignOffLabel` 非空（活动未开启） | R-ACC-1033–1040 |
| 102 | S-account-102 | 「邀请码」 | `ui.Text` | `type-micro`；`text-secondary`；`.Margin(0,0,space-1,0)` | `referral.affCode` 非空 | R-ACC-1041–1048 |
| 103 | S-account-103 | affCode（LYIr） | `ui.Text` | `.Font("SF Mono, Menlo, monospace").FontSize(type-code).Bold().SingleLine().Grow(1).Selectable()` | 可选中复制 | R-ACC-1049–1056 |
| 104 | S-account-104 | 「复制」 | `ui.Button`（ghost）+ `Children` | `.Height(DS.chip-btn-h)`（参考 `h-7`）、`Padding(0, space-2)`、字号 `type-micro`；内件 Icon 12 + Text | `OnClick → handleCopy(affCode)`；`c.Toast("已复制")` | R-ACC-1057–1066 |
| 105 | S-account-105 | ↳ Copy | `ui.Icon` | `.Size(12,12)` | — | R-ACC-1067–1074 |
| 106 | S-account-106 | 「邀请网址」 | `ui.Text` | 同 102 | `inviteUrl` 非空 | R-ACC-1075–1082 |
| 107 | S-account-107 | inviteUrl（https://dimagent.cn/register?aff=LYIr） | `ui.Text` | `.Font(mono).FontSize(type-code).SingleLine().Grow(1).Selectable()`；不加粗（参考 `text-xs`） | 可选中复制 | R-ACC-1083–1090 |
| 108 | S-account-108 | 「复制」 | `ui.Button`（ghost）+ `Children` | 同 104 | `OnClick → handleCopy(inviteUrl)` | R-ACC-1091–1100 |
| 109 | S-account-109 | ↳ Copy | `ui.Icon` | `.Size(12,12)` | — | R-ACC-1101–1108 |
| 110 | S-account-110 | 「已邀请」/「11 人」 | `ui.Row` + `ui.Text` ×2 | Row `.Justify(ui.SpaceBetween).AlignItems(Center).MinHeight(DS.metric-row-h).BorderWidth(0,0,1,0)`；dt `type-micro` `text-secondary`；dd `type-body-strong` + `.FontFeatures("tnum")` | 指标随数据 | R-ACC-1109–1116 |
| 111 | S-account-111 | 「已付费」/「6 人」 | 同 110 | 同上 | — | R-ACC-1117–1124 |
| 112 | S-account-112 | 「累计奖励」/「9,125 Credits」 | 同 110 | 同上；最后一项 `.BorderWidth(0,0,0,0)` | `referralRewardLabel` 非空 | R-ACC-1125–1133 |

合计 **112 条映射**（= 清单元素数 112，全覆盖）。

---

## 3 无对应控件清单

### 3.1 框架没有的（10 条）

| ID | 参考产品的做法 | 为什么框架不够 | 替代方案 / 自绘计划 | 影响元素 |
|---|---|---|---|---|
| G-01 | 头像 48×48 正圆，主色底 + 主色反白字母，取显示名首字（中文取首字） | `ui.Avatar` 尺寸固定 `FontSize×2.25`（13 → 29.25），底色由名字哈希取 HSL，无法给主色/前景色 | ① 先试 `ui.Avatar(name, image).Size(DS.avatar, DS.avatar).Background(accent).TextColor(accent-on)`（`Size`/`Background`/`TextColor` 是 Element 通用属性，可覆盖；字母字号随尺寸 0.4 倍自算）；② 若覆盖后字母字号仍固定，则退化为自绘：`ui.Box().Size(42,42).Radius(21).Background(accent).Center().Role(ui.RoleImage).Label(displayName)` + `ui.Text(首字).Bold().TextColor(accent-on)`；③ 远端头像仍然走 `ui.Avatar(name, *ui.Bitmap)` 的图片分支（`Fit(Cover)` 裁圆已内建） | S-account-09 |
| G-02 | 进度条：轨道 `bg-meter-track`、填充按 `normal/warning/critical/depleted` 四档换 `primary/amber-500/destructive`、高 6（1.5）/4（1）、圆角全圆、`role=progressbar` + `aria-valuenow` | `ui.Progress` 轨道固定 `Theme.Border`、填充固定 `Theme.Accent`，没有颜色入参；`ui.Meter` 有 `Warning`/`Critical` 档色但默认色是 `Success`（绿）而不是强调蓝，且没有自定轨道色 | 自绘：`ui.Box().Height(DS.meter-h).Radius(999).Background(DS.meter-track).Clip().Grow(1)` + `.Draw(func(p, r){ p.Fill(Rect{…w=pct*r.W…}, fill, 999) })`；无障碍用 `.Role(ui.RoleProgress).Range(0,100,pct).Label(文案)`；`fill` 由档位映射（`accent` / `warning` / `danger`）。**不新增第三方依赖** | S-account-67、73、78、86 |
| G-03 | 重置确认框带自定义正文：标题 + 说明 + **一批批次列表**（加载中显示骨架条 / 空 / 失败三支）+ 取消 / 使用 | `ui.AlertDialog(c, open, title, message, buttons...)` 的正文只有 `message string`，塞不下列表与三支状态 | 用 `ui.Modal` + 自绘面板：标题 `Text type-h2`、说明 `Text type-body`、列表 `Column` + `Text`、底部 `Row(Justify End)` + `Button`/`PrimaryButton`；语义上仍要「默认按钮 + Escape 取消 + 点外不关」→ 面板 `.Modal()`、主按钮 `.AutoFocus()`，Escape 由 `Modal` 的 `Dismissed()` 分支处理（参考 `AlertDialog` 的等价行为在 `mygo:docs/ui/alert-dialog.md` 有描述，Modal 需自行补齐） | S-account-46–53 |
| G-04 | 批次加载骨架条：`h-4 w-2/3 animate-pulse rounded bg-muted` | 无 skeleton / 骨架控件，也无「按秒呼吸」的内建样式（有 `Loop`/`Transition` 但需自己接） | `ui.Box().Height(16).WidthPercent(66).Radius(6).Background(surface-hover)` + `.Opacity(skeletonOpacity)`，其中 `skeletonOpacity` = `e.Loop("pulse", 1.2*time.Second, ui.EaseInOut)` 映射到 0.5–1.0；`.AccessHidden(true)`（参考 `aria-hidden`）；动效降级见 `mygo:docs/ui/motion.md` | S-account-46（内件） |
| G-05 | 按**容器**宽度断点：`md:p-8`、`sm:flex-row`、`lg:grid-cols-1` / `lg:min-w-[11rem]`、`flex-wrap` 塌落 | `ui.Context` 没有宽度查询（只有 `Valid/Invalidate/Key/Resolve/Services` 等），也没有 media / container query；`Element.Bounds()` 只能读到上一帧 | 在 app 状态里保存内容列宽（由宿主窗口尺寸 + 侧栏宽算出，`mygo.Window.Size()`），view 里按三个阈值二选一构建：`DS.bp-page-pad`（参考 768）、`DS.bp-row`（参考 640）、`DS.bp-split`（参考 1024）；首帧按窗口宽先算，滚动区宽度变化时重算。`flex-wrap` 本身有 `.Wrap()`，不需要断点 | L0 页面、B.2、C.3 指标组、D.5 |
| G-06 | 半透明语义色：`destructive/5`、`destructive/25`、`primary/10`、`muted/40`、`border/70`、`bg-meter-track` | 主题只有 **17 个不透明语义 token**（U2 §3.1「颜色只经语义 token 使用」），没有 α 变体；`bg-meter-track` 在框架里根本没有对应字段 | 在 view 层定一条派生规则并登记：`token@α` = `token.Alpha(α)`（`ui.Color.Alpha` 存在），α 档位取 `DS.alpha-05 / 10 / 20 / 25 / 40 / 70`；`meter-track` 作为**一个**派生常量（建议 `border@α`，需 design-baseline 定值）。落点清单见 §3.2 | 2、9、36、39、43、45、54、57、62、67、73、78、81、86、96、103、107 |
| G-07 | ghost / outline / icon-only 三种按钮变体（`variant="ghost"`、`variant="outline"`、`size="icon"`） | 框架只有 `Button`（`Surface` 底 + 1px 描边）与 `PrimaryButton`（强调填充） | 用 `ui.ButtonBase`（无外观，自带焦点环、Enter/Space、Clicked）自绘三种变体，样式常量放一处：outline = `Border(1, border)` + `Background(transparent)` + 悬停 `surface-hover`；ghost = 无边框无底 + 悬停 `surface-hover`；icon-only = ghost + 固定方形尺寸 + `.Label(...)`/`.Tooltip(...)` | 6、14、16、19、27、32、43、52、90、104、108 |
| G-08 | `text-box: trim-both cap alphabetic` + `tracking-tight` | 框架没有文本盒裁切（`FixedLineHeight` 只能定行盒总高，不能按 cap/alphabetic 裁边），也没有字距收紧的语义 API（`LetterSpacing` 有，但要按字号算） | 放弃光学裁切：按钮内一律 `Row.AlignItems(Center)` + 固定 16/14/12 图标盒；若中文标签仍比图标低 0.5–1px，用 `.Margin(0,0,DS.label-nudge,0)` 逐处补偿（**偏差登记为可接受**，落进 R-ACC-049/051 的容差）；`tracking-tight` 不实现（`type-h2` 已有自己的字距） | 4、6、16、43、90、93、104 |
| G-09 | 滚动条宽 10，`scrollbar-gutter: stable`，拇指悬停才显色 | 滚动条只受主题控制（`Theme.Scrollbar` / `ScrollbarWidth`，默认 6），没有 per-container 覆盖；`scrollbar-gutter` 无对应（框架滚动条不占位） | 全局 `Theme.ScrollbarWidth = DS.scrollbar-w`（参考 10；改主题是全局决定，需 A/C 领域确认，见 D-04）；悬停态框架已内建（「指针在容器上才显示拇指」`mygo:docs/ui/scroll.md` §Scroll bars），与参考一致，不需另做 | L0 滚动视口 |
| G-10 | lucide 图标集 + 品牌标识 `BrandMark`（自有 logo 组件） | mygo 不自带图标；`ui.SVG` 需要外部 SVG 源码（`ui.MustParseSVG` + `//go:embed`） | 由图标资产清单引入 lucide（MIT）13 个 + 品牌标识 1 个；`BrandMark` 是产品自有图形，必须由设计/资产侧出 SVG（本文件只登记需求，不代画） | 3、7、15、17、25、33、40、44、55、59、64、82、91、94、95、97、105、109、36 |

### 3.2 无对应 token 的派生色（G-06 的落点）

| 参考类 | 用途 | 派生写法 | 用在哪 |
|---|---|---|---|
| `bg-destructive/5` | 到期横幅底、耗尽提示底 | `danger.Alpha(0.05)` | 2、54 |
| `border-destructive/25` `/20` `/40` | 横幅/提示框描边、计数朵临界描边 | `danger.Alpha(0.25/0.20/0.40)` | 2、45、54 |
| `text-destructive/80` | 横幅与耗尽提示的说明行 | `danger.Alpha(0.80)` | 5、57 |
| `bg-primary/10` | 品牌块底、状态徽标底、计数朵底 | `accent.Alpha(0.10)` | 36、39、45 |
| `bg-muted/40`（码行底）、`border-border/70`（框线）、`border-border/50`（指标分行） | 邀请码行、额度框、指标行 | `surface.Alpha(0.40)`、`border.Alpha(0.70/0.50)` | 62、81、103、107、110–112 |
| `bg-meter-track` | 进度条轨道 | `DS.meter-track`（建议取 `border.Alpha(α)`，值待定） | 67、73、78、86 |

### 3.3 有替代但会决定写法的约束（不算缺口）

| ID | 约束 | 处理 |
|---|---|---|
| C-01 | 三种按钮变体要自绘 | 在 `internal/ui` 的一处集中定义（与 `U0-control-baseline` §2 的 override 层同址），禁止每个 view 各写一遍 |
| C-02 | 耗尽提示的**左边框 3px**（`border-l-[3px]`）与其余 1px 不同色 | mygo 的 `BorderWidth(t,r,b,l)` 支持分边宽，但颜色只有一个 `BorderColor` → 用一个 `.Absolute().Width(3).Background(danger)` 的左侧条，或用 `InsetShadow` 画 3px 内阴影；两案都要保证不改变文字位置 |
| C-03 | 按钮内多段富文本（`查看套餐` + `·` + `32 个模型`） | `Button(label)` 只接受单串 → 用 `Button(c, "").Children(...)` 拼 `Icon` + 3 个 `Text`；`·` 用 `.AccessHidden(true)`；整串的无障碍名用 `.Label("查看套餐，32 个模型")` |
| C-04 | 两处进度条的**金额列与百分比列必须同宽**，否则两条不齐 | `DS.meter-amount-w` / `DS.meter-pct-w` 只允许定义一次并在 4 个用量行复用（参考源码注释明确写了这条） |
| C-05 | `ring-1`（计数朵临界到期描边）不占布局 | 框架 `Border` 会占位（`box-sizing: border-box`，占内宽）→ 常态就挂 `Border(1, Transparent)`，到期时只换色，避免尺寸跳动 |
| C-06 | 刷新按钮「最短旋转时长」（参考有 `MIN_MANUAL_REFRESH_SPIN_MS`） | 不是控件问题，但会决定 `Disabled(refreshing)` 与 `Loop` 的时序：`refreshing` 由上层计时器保证不短于 `DS.spin-min`，不要在 view 里用一次性延时 |

---

## 4 验收对齐

### 4.1 分组 → 需求 ID（两级：分组区间 + 元素区间）

| 分组 | 区域 | 元素 | 需求 ID | 条数 | 验收时先看什么 |
|---|---|---|---|---|---:|---|
| A.1 页面标题 | A | S-account-01 | R-ACC-001–009 | 9 | 文案逐字、y 位置、滚动行为、二级标题角色 |
| B.1 到期横幅 | B | S-account-02–07 | R-ACC-010–065 | 56 | 条件出现、宽度与卡同宽、4 类物件、可点外不响应 |
| B.2 身份卡 | B | S-account-08–19 | R-ACC-066–237 | 172 | 740×96 位置、5 类物件、缺字段塌落、按钮态 |
| B.3 账号错误行 | B | S-account-20 | R-ACC-238–247 | 10 | 只在错误行，不替换卡片 |
| B'.1 未登录卡 | B' | S-account-21–28 | R-ACC-248–339 | 92 | 卡框、标题、两个入口、登录中态 |
| B'.2 邮箱登录对话框 | B' | S-account-29 | R-ACC-340–356 | 17 | 清空、校验、Enter 提交 |
| C.1 卡头 | C | S-account-30–33 | R-ACC-357–418 | 62 | 卡框、标题、刷新禁用与旋转 |
| C.2 状态行 | C | S-account-34–35 | R-ACC-419–435 | 17 | 错误/加载互斥 |
| C.3 服务行 | C | S-account-36–41 | R-ACC-436–490 | 55 | 品牌块、徽标、到期文案两支 |
| C.4 重置段 + 确认框 | C | S-account-42–53 | R-ACC-491–618 | 128 | 计数朵、禁用、对话框三支与默认按钮 |
| C.5 耗尽提示 | C | S-account-54–57 | R-ACC-619–652 | 34 | 条件、左侧 3px 条、对比度 |
| C.6 不限量支 | C | S-account-58–61 | R-ACC-653–684 | 32 | 值用强调色文字、无进度条 |
| C.7 月度额度 | C | S-account-62–68 | R-ACC-685–766 | 82 | 两列同宽、进度条四档色、百分比色 |
| C.8 周期限制窗口 | C | S-account-69–80 | R-ACC-767–862 | 96 | 两种高度的进度条、标签列宽 96、换行 |
| C.9 功能额度 | C | S-account-81–87 | R-ACC-863–922 | 60 | 框内多条、`unlimited` 时无条 |
| C.10 未订阅 / 空 | C | S-account-88–92 | R-ACC-923–964 | 42 | 三支互斥、按钮去向 |
| C.11 续订 CTA | C | S-account-93–95 | R-ACC-965–990 | 26 | 到期/生效两支图标与文案 |
| D.1–D.5 推广邀请 | D | S-account-96–112 | R-ACC-991–1133 | 143 | 码/网址两行可选中、复制 toast、指标组断点 |

合计 **1133 条 = 清单 112 元素全覆盖**，无未挂需求元素。

### 4.2 逐元素区间（实现后按此表逐条对）

| 元素 | 需求 | 元素 | 需求 | 元素 | 需求 | 元素 | 需求 |
|---|---|---|---|---|---|---|---|
| 01 | 001–009 | 29 | 340–356 | 57 | 645–652 | 85 | 897–904 |
| 02 | 010–021 | 30 | 357–386 | 58 | 653–660 | 86 | 905–912 |
| 03 | 022–029 | 31 | 387–394 | 59 | 661–668 | 87 | 913–922 |
| 04 | 030–037 | 32 | 395–410 | 60 | 669–676 | 88 | 923–930 |
| 05 | 038–045 | 33 | 411–418 | 61 | 677–684 | 89 | 931–938 |
| 06 | 046–057 | 34 | 419–427 | 62 | 685–694 | 90 | 939–948 |
| 07 | 058–065 | 35 | 428–435 | 63 | 695–702 | 91 | 949–956 |
| 08 | 066–095 | 36 | 436–443 | 64 | 703–710 | 92 | 957–964 |
| 09 | 096–105 | 37 | 444–451 | 65 | 711–718 | 93 | 965–974 |
| 10 | 106–115 | 38 | 452–460 | 66 | 719–726 | 94 | 975–982 |
| 11 | 116–124 | 39 | 461–472 | 67 | 727–756 | 95 | 983–990 |
| 12 | 125–132 | 40 | 473–480 | 68 | 757–766 | 96 | 991–1000 |
| 13 | 133–142 | 41 | 481–490 | 69 | 767–774 | 97 | 1001–1008 |
| 14 | 143–172 | 42 | 491–500 | 70 | 775–782 | 98 | 1009–1016 |
| 15 | 173–180 | 43 | 501–530 | 71 | 783–790 | 99 | 1017–1024 |
| 16 | 181–200 | 44 | 531–538 | 72 | 791–798 | 100 | 1025–1032 |
| 17 | 201–208 | 45 | 539–546 | 73 | 799–806 | 101 | 1033–1040 |
| 18 | 209–217 | 46 | 547–554 | 74 | 807–814 | 102 | 1041–1048 |
| 19 | 218–237 | 47 | 555–562 | 75 | 815–822 | 103 | 1049–1056 |
| 20 | 238–247 | 48 | 563–570 | 76 | 823–830 | 104 | 1057–1066 |
| 21 | 248–257 | 49 | 571–582 | 77 | 831–838 | 105 | 1067–1074 |
| 22 | 258–265 | 50 | 583–590 | 78 | 839–846 | 106 | 1075–1082 |
| 23 | 266–274 | 51 | 591–598 | 79 | 847–854 | 107 | 1083–1090 |
| 24 | 275–286 | 52 | 599–606 | 80 | 855–862 | 108 | 1091–1100 |
| 25 | 287–294 | 53 | 607–618 | 81 | 863–872 | 109 | 1101–1108 |
| 26 | 295–314 | 54 | 619–628 | 82 | 873–880 | 110 | 1109–1116 |
| 27 | 315–329 | 55 | 629–636 | 83 | 881–888 | 111 | 1117–1124 |
| 28 | 330–339 | 56 | 637–644 | 84 | 889–896 | 112 | 1125–1133 |

### 4.3 验收槽位（哪些需求靠本规格就能判，哪些要靠 design-baseline）

| 需求维度 | 条数（本屏） | 本规格是否足以判定 | 缺什么 |
|---|---:|---|---|
| 内容（文案逐字） | 约 190 | 是 | — （文案已逐字抄自 `i18n/zh-CN.ts`） |
| 状态 / 关联 / 异常 / 记忆 | 约 480 | 是（条件与互斥关系已写进结构树的条件标注） | 数据层契约（账号、订阅、推广三份接口） |
| 交互 / 动画 | 约 170 | 部分 | 过渡时长/曲线（`DS.motion-*`）；`Transition` 的具体参数 |
| 样式（色/字号/间距/圆角） | 约 250 | 否（`判据TBD`） | **design-baseline.md**：`DS.card-r`、`DS.page-pad-y`、`DS.avatar`、`DS.meter-*`、`DS.alpha-*`、`DS.*-w` |
| 无障碍 / 键盘 | 约 40 | 是（角色与 label 已定） | 读屏实测（要设备） |

---

## 5 依赖与未定项

| ID | 依赖 | 影响 | 谁来定 |
|---|---|---|---|
| D-01 | `design-baseline.md` 尚未生成 | 本文件全部 `DS.*` 占位（约 26 个，见下表） | 设计基线领域 |
| D-02 | 卡片圆角：参考 `rounded-lg` = 8，我们 `Theme.Radius` = 6 | 本屏 5 个卡框 + 3 个内框全部受影响 | design-baseline 裁决「卡 8 / 控件 6」还是「统一 6」 |
| D-03 | 图标资产清单（lucide 13 个 + 品牌标识 1 个） | 3、7、15、17、25、33、40、44、55、59、64、82、91、94、95、97、105、109、36 | 资产/设计侧出 SVG；`//go:embed` 落在哪个包待定 |
| D-04 | `Theme.ScrollbarWidth` 改 10 是全局决定 | 影响所有滚动容器，不只本屏 | 需 A/C 领域确认 |
| D-05 | 主题取 `Theme.FontSize = 14`（U2 §1.2 的偏离） | `type-*` 全部按 14 算；若回退到 13，本屏字号同比缩小，间距不变 | 已在 U2 留了回退方案 |
| D-06 | 需求与参考源码的一处不一致：R-ACC-109 期望「未命名」，参考实现回落的是 `accountSettings.fallbackName` =「Dim 账户」 | S-account-10 的空值分支 | 需求侧修正，或实现按需求改文案（二者取一，需裁决） |
| D-07 | 参考产品的日期/金额本地化格式（「2026年12月13日」「到期 … · 还剩 26 天」「10.8万 / 30万」） | S-account-41、65、66、84 | 数据层返回已本地化字符串还是给原始数值 + 前端格式化 |

`DS.*` 占位清单（26 个，交 design-baseline 一次定完）：

| 名 | 用途 | 参考值（Tailwind / 声明） |
|---|---|---|
| `DS.page-pad-y` | 页面上下内边距 | `p-6` / `md:p-8` = 24 / 32 |
| `DS.content-max` | 内容列最大宽 | `max-w-6xl` = 1152 |
| `DS.card-r` | 卡片圆角 | `rounded-lg` = 8 |
| `DS.chip-r` / `DS.chip-r-lg` | 芯片圆角 | `rounded-md` = 6 / `rounded-lg` = 8 |
| `DS.card-header-h` | 卡头最小高 | `min-h-14` = 56 |
| `DS.btn-h-sm` | 小按钮高 | `h-8` = 32 |
| `DS.chip-btn-h` | 芯片内按钮高 | `h-7` = 28 |
| `DS.avatar` | 头像边长 | `h-12 w-12` = 48 |
| `DS.identity-min` | 身份块最小宽 | `min-w-[12rem]` = 192 |
| `DS.dialog-w` | 对话框宽 | `sm:max-w-md` = 448 |
| `DS.meter-h` / `DS.meter-h-thin` | 进度条高 | `h-1.5` = 6 / `h-1` = 4 |
| `DS.meter-amount-w` | 金额列宽 | `9.5rem` = 152 |
| `DS.meter-pct-w` | 百分比列宽 | `5rem` = 80 |
| `DS.window-label-w` | 窗口标签列宽 | `w-24` = 96 |
| `DS.metric-row-h` | 指标行最小高 | `min-h-11` = 44 |
| `DS.meter-track` | 进度条轨道色 | `bg-meter-track`（值未在本次材料中出现） |
| `DS.alpha-05/10/20/25/40/50/70` | 半透明档位（G-06） | `destructive/5`、`primary/10`、`danger/20`、`destructive/25`、`muted/40`、`border/50`、`border/70` |
| `DS.bp-page-pad` / `DS.bp-row` / `DS.bp-split` | 三个容器断点（G-05） | 768 / 640 / 1024 |
| `DS.relaxed` | 说明行行高倍数 | `leading-relaxed` = 1.625 |
| `DS.label-nudge` | 按钮标签光学补偿（G-08） | 参考用 `text-box` 裁切，无像素值可抄；待实测 |
| `DS.motion-fast` | 按钮过渡时长 | 参考 `150ms`，曲线 `cubic-bezier(.23,1,.32,1)` |
| `DS.spin-min` | 刷新最短旋转时长 | 参考常量 `MIN_MANUAL_REFRESH_SPIN_MS`，值待取 |
| `DS.scrollbar-w` | 滚动条宽 | 参考 10，框架默认 6 |
