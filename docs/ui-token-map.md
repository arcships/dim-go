# UI token 逐项映射（dim-electron renderer ↔ dimagent-mygo）

Status: current
Last updated: 2026-11-09
证据等级：**L0 `source-read`**（只读源码，没有在真实窗口里逐项取样；色值读数引用已有的 [ui-parity.md](ui-parity.md) §5.2）

- `A/` = `/Users/eric8810/Code/dim-electron/packages/desktop/packages/renderer/src`
- `B/` = `/Users/eric8810/Code/dimagent-mygo`（token 定义在 [internal/ui/tokens.go](../internal/ui/tokens.go)，状态栏常量在 [internal/ui/statusbar.go](../internal/ui/statusbar.go)）
- `mygo` = `github.com/egoist/mygo/ui` v0.8.3；本文引用本地 checkout `mygo ui/theme.go`

本文只回答一个问题：**B 的每个 token 在 A 里对应哪个变量或哪条 utility，值是否真的相等。** 组件域差距见 [ui-parity.md](ui-parity.md)，区域布局见 [ui-layout-spec.md](ui-layout-spec.md)。

## 0. 读表前提

1. **A 的颜色是 CSS 变量，几何大多不是。** 颜色在 `A/styles/global.css` 里用 `--token` 定义；间距、行高、圆角使用点大多是 Tailwind class 或 px 字面量，表 2 给的是**使用该值的 `.vue` 行号**，不是变量。
2. **A 的根字号是 14px。** `html { font-size: var(--dimcode-ui-font-size) }`（`A/styles/global.css:390`），默认 14px（`A/styles/global.css:166`）。所有 rem 类按 14px 换算：`px-2`=7、`px-3`=10.5、`px-4`=14、`h-7`=24.5、`h-9`=31.5、`h-10`=35、`gap-3`=10.5。
3. **B 的间距来自 mygo 的 Spacing，不是 Tailwind 刻度。** `theme.Space(n) = n × Spacing`，默认 Spacing=4（`mygo ui/theme.go:189`、`B/internal/ui/tokens.go:368-373`），所以 `Space(3)`=12 而 A 的 `gap-3`/`px-3`=10.5。表 2 里凡是 B 用 `Space(n)` 而 A 用 rem 类的行，都按实际差异标注。
4. **A 的圆角是 rem，会随字号档位缩放。** `--radius: 0.625rem`（`A/styles/global.css:244`），阶梯 `calc(var(--radius) ± Npx)`（`A/styles/global.css:45-48`）。在默认 14px 根字号下实际是 **4.75 / 6.75 / 8.75 / 12.75 DIP**，不是 6/8/10/14；只有根字号为 16px 时才是 6/8/10/14。B 的 `radiusSM..XL` 是固定常量（[tokens.go:53-56](../internal/ui/tokens.go#L53-L56)），不随字号缩放。
5. **同名不等于同值。** B 的 35 个颜色全部由 `ui.Theme` 派生（[tokens.go:407-578](../internal/ui/tokens.go#L407-L578)）；A 的 light/dark 是写死的 oklch。B light 取 `mygo ui/theme.go:169-182`，dark 取 `mygo ui/theme.go:198-212`，再经 `Mix`/`Alpha` 派生。A 直接给最终色值，两者的派生空间没有对应关系，所以表 1 只能判定“语义角色”，不能判定“色值相等”。
6. **A 有第三层 skin/chrome。** 本文只映射基础语义层；`skin.json` 与 6 个 chrome 主题是覆盖层，见 §3.2。

判定口径：

| 判定 | 含义 |
|---|---|
| 对应 | A 有同名（或同角色）变量，且 B 能用单一实色表达同一角色；值可能不同，备注里写 |
| 部分 | A 的值依赖 ≤90% alpha、模糊、多层阴影、vibrancy、渐变，或 A/B 的色相/表面层级不同，B 无法用同一 token 复现 |
| 无 | A 没有可对应的变量/常量，B 的值是自造或派生 |

## 1. 颜色 token → Electron CSS 变量

表 1 覆盖 [tokens.go:263-312](../internal/ui/tokens.go#L263-L312) 的全部 **35 个 `ui.Color` 字段**。A 侧行号成对给出 light / dark。

| B token（定义 / 赋值） | A 变量（light / dark） | A 值（light / dark，原文） | 判定 | 备注 |
|---|---|---|---|---|
| `background` [:263](../internal/ui/tokens.go#L263) / [:419](../internal/ui/tokens.go#L419) | `--background`（183 / 272） | `oklch(1 0 0)` / `oklch(0.16 0 0)` | 对应 | B = `Theme.Background`：light `#ffffff`、dark `#18181b`；light 同值，dark 不同（A 注释自称 ≈`#121213`，`global.css:268-271`） |
| `foreground` [:264](../internal/ui/tokens.go#L264) / [:419](../internal/ui/tokens.go#L419) | `--foreground`（184 / 273） | `oklch(0.145 0 0)` / `oklch(0.9431 0 0)` | 对应 | B `#18181b` / `#f4f4f5` |
| `sidebar` [:265](../internal/ui/tokens.go#L265) / [:425-436](../internal/ui/tokens.go#L425-L436) | `--sidebar`（234 / 316） | `oklch(0.99 0.012 250 / 65%)` / `oklch(0.15 0.006 240 / 60%)` | 部分 | A 是叠在 vibrancy 上的半透明层，`non-vibrant-window`（420-431）与 `prefers-reduced-transparency`（441-454）会改写；B 取 `SurfaceHover` (`#e9e9ec`) / `Surface` (`#27272a`)，不透明。取样差异见 [ui-parity.md](ui-parity.md) §5.2 |
| `card` [:266](../internal/ui/tokens.go#L266) / [:437](../internal/ui/tokens.go#L437) | `--card`（185 / 274） | `oklch(1 0 0)` / `oklch(0.25 0 0)` | 对应 | B = `background.Mix(white, 0.12)`；light 都白，dark B ≈`#343436`（取样 [ui-parity.md](ui-parity.md) §5.2） |
| `cardForeground` [:267](../internal/ui/tokens.go#L267) / [:543](../internal/ui/tokens.go#L543) | `--card-foreground`（186 / 275） | `oklch(0.145 0 0)` / `oklch(0.9431 0 0)` | 对应 | B 直接复用 `foreground` |
| `popover` [:268](../internal/ui/tokens.go#L268) / [:438](../internal/ui/tokens.go#L438) | `--popover`（187 / 276） | `oklch(1 0 0)` / `oklch(0.25 0 0)` | 部分 | A 的 dark popover 与 card **同值**；B 又升一档 `Mix(white, 0.18)`，dark 比 card 更亮（取样 `#424244`，[ui-parity.md](ui-parity.md) §5.2） |
| `popoverForeground` [:269](../internal/ui/tokens.go#L269) / [:545](../internal/ui/tokens.go#L545) | `--popover-foreground`（188 / 277） | `oklch(0.145 0 0)` / `oklch(0.9431 0 0)` | 对应 | B = `foreground` |
| `sidebarForeground` [:273](../internal/ui/tokens.go#L273) / [:547](../internal/ui/tokens.go#L547) | `--sidebar-foreground`（236 / 318） | `oklch(0.145 0 0 / 92%)` / `oklch(0.985 0 0 / 92%)` | 对应 | B 用不透明 `foreground`；A 带 92% alpha |
| `sidebarPrimary` [:274](../internal/ui/tokens.go#L274) / [:548](../internal/ui/tokens.go#L548) | `--sidebar-primary`（237 / 319） | `var(--brand-green)`（180 / 265） | 对应 | B = `primary` = 品牌绿 `#349651` / `#64cb7b`（`brandGreenLight/Dark`，按 oklch 换算的近似实色） |
| `sidebarPrimaryForeground` [:275](../internal/ui/tokens.go#L275) / [:549](../internal/ui/tokens.go#L549) | `--sidebar-primary-foreground`（238 / 320） | `oklch(0.985 0 0 / 96%)` / `oklch(0.985 0 0 / 98%)` | 对应 | B = `onPrimary` / `Theme.AccentText`（白）；A 近白带 2–4% alpha |
| `sidebarAccent` [:276](../internal/ui/tokens.go#L276) / [:550](../internal/ui/tokens.go#L550) | `--sidebar-accent`（239 / 321） | `oklch(0.45 0 0 / 12%)` / `oklch(0.92 0.004 240 / 18%)` | 部分 | A 是低 alpha 中性层；B = `sidebar.Mix(foreground, 0.08)`，方向一致但仍是不透明实色 |
| `sidebarAccentForeground` [:277](../internal/ui/tokens.go#L277) / [:551](../internal/ui/tokens.go#L551) | `--sidebar-accent-foreground`（240 / 322） | `oklch(0.205 0 0 / 90%)` / `oklch(0.985 0 0 / 96%)` | 对应 | B = `foreground`；A 有 90/96% alpha |
| `sidebarHover` [:278](../internal/ui/tokens.go#L278) / [:552](../internal/ui/tokens.go#L552) | `--sidebar-hover`（241 / 323） | `oklch(0.45 0 0 / 12%)` / `oklch(0.92 0.004 240 / 14%)` | 部分 | B = `sidebar.Mix(foreground, 0.06)`，是 6% 实色混合；A 是 12/14% alpha 叠层 |
| `sidebarBorder` [:279](../internal/ui/tokens.go#L279) / [:553](../internal/ui/tokens.go#L553) | `--sidebar-border`（242 / 324） | `oklch(1 0 0 / 42%)` / `oklch(0.3 0 0 / 80%)` | 部分 | B = `Border.Mix(sidebar, 0.35)`；light A 是白色 42% 叠层，B 是实色灰 |
| `sidebarRing` [:280](../internal/ui/tokens.go#L280) / [:554](../internal/ui/tokens.go#L554) | `--sidebar-ring`（243 / 325） | `oklch(0.708 0 0 / 40%)` / `oklch(0.556 0 0 / 45%)` | 部分 | B = 主 `ring`（`Theme.Focus` 蓝色 alpha，缺失时回退 `primary`）；A 是独立灰色 alpha |
| `primary` [:286](../internal/ui/tokens.go#L286) / [:556](../internal/ui/tokens.go#L556) | `--primary`（205 / 292） | `var(--brand-green)`（180 / 265） | 对应 | B = `brandGreenLight/Dark` `#349651` / `#64cb7b`；A = `oklch(0.6000 0.1380 150)` / `oklch(0.7600 0.1500 149)`，是同一品牌的近似实色 |
| `onPrimary` [:287](../internal/ui/tokens.go#L287) / [:557](../internal/ui/tokens.go#L557) | `--primary-foreground`（206 / 293） | `var(--brand-cta-ink)`（181 / 266） | 对应 | B = 白 `#ffffff`（light）/ 近黑苔绿 `#07140a`（dark），与 A 的 `--brand-cta-ink` 方向一致 |
| `secondary` [:288](../internal/ui/tokens.go#L288) / [:558](../internal/ui/tokens.go#L558) | `--secondary`（211 / 297） | `oklch(0.97 0 0)` / `oklch(0.27 0 0)` | 对应 | B = `Theme.Surface` `#f4f4f5` / `#27272a` |
| `secondaryForeground` [:289](../internal/ui/tokens.go#L289) / [:559](../internal/ui/tokens.go#L559) | `--secondary-foreground`（212 / 298） | `oklch(0.205 0 0)` / `oklch(0.9431 0 0)` | 对应 | B = `foreground` |
| `muted` [:290](../internal/ui/tokens.go#L290) / [:560](../internal/ui/tokens.go#L560) | `--muted`（213 / 299） | `oklch(0.97 0 0)` / `oklch(0.27 0 0)` | 对应 | B = `Theme.Surface` |
| `mutedForeground` [:291](../internal/ui/tokens.go#L291) / [:561](../internal/ui/tokens.go#L561) | `--muted-foreground`（214 / 300） | `oklch(0.556 0 0)` / `oklch(0.6334 0 0)` | 对应 | B = `Theme.TextMuted` `#71717a` / `#a1a1aa` |
| `accent` [:292](../internal/ui/tokens.go#L292) / [:562](../internal/ui/tokens.go#L562) | `--accent`（215 / 301） | `oklch(0.97 0 0)` / `oklch(0.27 0 0)` | 对应 | B = `SurfaceHover`，中性 hover 面；语义与 A 一致 |
| `accentForeground` [:293](../internal/ui/tokens.go#L293) / [:563](../internal/ui/tokens.go#L563) | `--accent-foreground`（216 / 302） | `oklch(0.205 0 0)` / `oklch(0.9431 0 0)` | 对应 | B = `foreground` |
| `danger` [:294](../internal/ui/tokens.go#L294) / [:564](../internal/ui/tokens.go#L564) | `--destructive`（217 / 303） | `oklch(0.577 0.245 27.325)` / `oklch(0.704 0.191 22.216)` | 对应 | B = `Theme.Danger` `#dc2626` / `#ef4444`；同角色，不同色值 |
| `onDanger` [:295](../internal/ui/tokens.go#L295) / [:565](../internal/ui/tokens.go#L565) | `--destructive-foreground`（218 / 304） | `oklch(0.985 0 0)` / `oklch(0.985 0 0)` | 对应 | B 由 `readableOn(danger)` 计算（[tokens.go:599-605](../internal/ui/tokens.go#L599-L605)），不是主题字段 |
| `border` [:300](../internal/ui/tokens.go#L300) / [:567](../internal/ui/tokens.go#L567) | `--border`（219 / 306） | `oklch(0.922 0 0)` / `oklch(1 0 0 / 14%)` | 部分 | B = `Theme.Border` `#d9d9de` / `#3f3f46`；dark A 是 alpha，B 是实色 |
| `input` [:301](../internal/ui/tokens.go#L301) / [:568](../internal/ui/tokens.go#L568) | `--input`（227 / 309） | `oklch(0.922 0 0)` / `oklch(1 0 0 / 20%)` | 部分 | B 直接复用 `border`，丢掉 A dark 下 input 20% 与 border 14% 的区别 |
| `ring` [:302](../internal/ui/tokens.go#L302) / [:569](../internal/ui/tokens.go#L569) | `--ring`（228 / 310） | `oklch(0.708 0 0)` / `var(--brand-green)` | 部分 | B = `Theme.Focus`（蓝 55/60% alpha），缺失时回退 `primary`；A light 是灰、dark 是绿实色 |
| `success` [:303](../internal/ui/tokens.go#L303) / [:570](../internal/ui/tokens.go#L570) | `--activity-success`（340 / 366） | `oklch(0.596 0.145 163.225)` / `oklch(0.765 0.177 163.223)` | 部分 | A 基础层没有 `--success`，只有 activity 色槽；B = `Theme.Success` `#16a34a` / `#22c55e` |
| `warning` [:304](../internal/ui/tokens.go#L304) / [:571](../internal/ui/tokens.go#L571) | `--activity-warning`（341 / 367） | `oklch(0.666 0.179 58.318)` / `oklch(0.828 0.189 84.429)` | 部分 | 同上；B = `Theme.Warning` `#d97706` / `#f59e0b` |
| `tooltip` [:308](../internal/ui/tokens.go#L308) / [:573](../internal/ui/tokens.go#L573) | `--tooltip`（192 / 279） | `oklch(0.99 0.004 240 / 50%)` / `oklch(0.32 0.004 240 / 46%)` | 部分 | A 是磨砂半透明面 + backdrop/shadow；B 是 `Theme.Inverse` 实色（缺失时回退 `foreground`），没有玻璃层 |
| `tooltipForeground` [:309](../internal/ui/tokens.go#L309) / [:574](../internal/ui/tokens.go#L574) | `--tooltip-foreground`（193 / 280） | `oklch(0.18 0 0)` / `oklch(0.97 0 0)` | 对应 | B = `Theme.InverseText` |
| `tooltipMutedForeground` [:310](../internal/ui/tokens.go#L310) / [:575](../internal/ui/tokens.go#L575) | `--tooltip-muted-foreground`（194 / 281） | `oklch(0.42 0 0)` / `oklch(0.74 0 0)` | 对应 | B 由 `foreground.Mix(tooltip, 0.30)` 派生，不是主题字段 |
| `tooltipBorder` [:311](../internal/ui/tokens.go#L311) / [:576](../internal/ui/tokens.go#L576) | `--tooltip-border`（195 / 282） | `oklch(0.3 0 0 / 14%)` / `oklch(1 0 0 / 12%)` | 部分 | B 是 `tooltip.Mix(foreground, 0.18)` 实色；A 是低 alpha 边 |
| `tooltipChip` [:312](../internal/ui/tokens.go#L312) / [:577](../internal/ui/tokens.go#L577) | `--tooltip-chip`（196 / 283） | `oklch(0.5 0 0 / 10%)` / `oklch(1 0 0 / 10%)` | 部分 | B 是 `tooltip.Mix(foreground, 0.10)` 实色；A 是 10% alpha |

**计数：35/35 字段都有对照行；21 个“对应”、14 个“部分”、0 个完全没有 A 变量。** 但“对应”只说明语义角色相同，不说明色值相同：`background`（light）与 `card`（light）同值，`bodySize` 与 `text-base` 同为 14 DIP；`ring` 仍是 mygo 蓝色焦点环，与 A 的灰/绿不同。

## 2. 几何 / 排版 token → 变量或常量

表 2 覆盖 [tokens.go:178-313](../internal/ui/tokens.go#L178-L313) 里除颜色外的 **53 个字段**。B 的值是 regular 密度下的值；compact 下 `Space(n)` 随 Spacing 收紧（4 → 3），字面量几何只有标注 `×scale` 的行会额外乘 [densityCompactScale](../internal/ui/tokens.go#L40) = 0.75。

### 2.1 外壳、卡片与间距

| B token（定义 / 赋值） | B 值 | A 来源 | 判定 | 差异 |
|---|---|---|---|---|
| `hairline` [:186](../internal/ui/tokens.go#L186) / [:473](../internal/ui/tokens.go#L473) | 1 | 卡片内 ring `0.5px`（`A/styles/global.css:35`）；Tailwind `border` 1px | 部分 | B 用 1 DIP 画所有描边和分隔线；A 的卡片框是 0.5px 内 ring + 0.5px 外 shadow |
| `gap` [:187](../internal/ui/tokens.go#L187) / [:474](../internal/ui/tokens.go#L474) | `Space(0.5)` = 2 | `gap-1` = 3.5（`A/components/ui/sidebar/SidebarMenu.vue:14`、`A/components/sidebar/ThreadItem.vue:241`） | 部分 | B 比 A 紧 1.5 DIP；语义是“行内部件间距”，但数值不在同一刻度 |
| `gapWide` [:188](../internal/ui/tokens.go#L188) / [:475](../internal/ui/tokens.go#L475) | `Space(3)` = 12 | 没有单一对应：header `gap-1.5`/`gap-1`（`ChatHeaderFrame.vue:60,67`）、composer `gap-2`=7（`ConversationComposer.vue:62`） | 无 | B 把工具栏/底部区间距统一成 12，比 A 的 3.5–7 宽一倍左右 |
| `rowPadY` [:189](../internal/ui/tokens.go#L189) / [:476](../internal/ui/tokens.go#L476) | `Space(1)` = 4 | 对话行没有纵向 padding；turn 间 `space-y-4`=14（`PlainTimeline.vue:3517`）、用户气泡 `py-2`=7（`TimelineUserBubble.vue:5`） | 无 | B 的 4 没有同角色 token；实际行距由 List 与 block gap 决定 |
| `rowPadX` [:190](../internal/ui/tokens.go#L190) / [:477](../internal/ui/tokens.go#L477) | `Space(4)` = 16 | `px-4` = 14（`PlainTimeline.vue:3516`） | 部分 | 值差 2 DIP；**B 当前没有消费点**（`grep k.rowPadX` 只命中定义/赋值） |
| `toolbarPadY` [:191](../internal/ui/tokens.go#L191) / [:482](../internal/ui/tokens.go#L482) | 5.5 ×scale | header 没有纵向 padding，高度由 `h-(--titlebar-height)` 决定（`ChatHeaderFrame.vue:43`） | 无 | B 用上下 5.5 去凑 41 的高度；A 没有这个 token |
| `headerH` [:192](../internal/ui/tokens.go#L192) / [:483](../internal/ui/tokens.go#L483) | 41 | 声明 48（`A/styles/global.css:174`、`ChatHeaderFrame.vue:43`）；参考窗口量到 41（[ui-parity.md](ui-parity.md) §5.2） | 无 | A 的 CSS 声明 48、截图量到 41，差 7 DIP；**没有记录量的是哪一个盒**。B 取 41 作为 `MinHeight`（`header.go:35`），不是变量对应 |
| `toolbarPadX` [:193](../internal/ui/tokens.go#L193) / [:484](../internal/ui/tokens.go#L484) | `Space(3)` = 12 | `px-3` = 10.5（`ChatHeaderFrame.vue:43`） | 部分 | 差 1.5 DIP |
| `edgePad` [:194](../internal/ui/tokens.go#L194) / [:485](../internal/ui/tokens.go#L485) | `Space(3)` = 12 | composer `p-3` = 10.5（`ConversationComposer.vue:62`） | 部分 | 差 1.5 DIP |
| `panelPad` [:195](../internal/ui/tokens.go#L195) / [:486](../internal/ui/tokens.go#L486) | `Space(6)` = 24 | 新线程页 `p-8`=28（`NewThread.vue:294`）；timeline 空态 `py-10`=35（`PlainTimeline.vue:3462`） | 无 | 三种“空态/加载态”内距在 A 里不是同一个值 |
| `composerH` [:196](../internal/ui/tokens.go#L196) / [:487](../internal/ui/tokens.go#L487) | `Space(18)` = 72 | 编辑器 `min-height: 60px`（`ComposerEditor.vue:1492`）、max 300（`:1493`）；卡片高度由内容决定 | 无 | **B 当前没有消费点**；A 没有 composer 总高 token |
| `statusDot` [:197](../internal/ui/tokens.go#L197) / [:488](../internal/ui/tokens.go#L488) | `Space(2)` = 8 | A 没有连接状态圆点 token；最接近的 `h-2 w-2`=7 是项目选中点（`NewThread.vue:370`） | 无 | B 自造的状态点，用于 header/statusbar（`header.go:116,151`、`statusbar.go:254`） |
| `cardRadius` [:198](../internal/ui/tokens.go#L198) / [:489](../internal/ui/tokens.go#L489) | 10 | tool 面板 `rounded-md`=6.75（`TimelineToolCallItem.vue:812`）；composer `rounded-xl`=12.75（`ConversationComposer.vue:61`） | 部分 | A 同一“卡片”在不同组件用两种半径；B 全用 10（`composer.go:72`、`tool.go:150`、`capability.go:123`） |
| `cardPadY` [:199](../internal/ui/tokens.go#L199) / [:490](../internal/ui/tokens.go#L490) | `Space(1)` = 4 | tool 面板标题 `pt-2`=7 / `pb-1`=3.5（`TimelineToolCallItem.vue:815-822`） | 无 | 没有统一的 card 纵向内距 token |
| `cardPadX` [:200](../internal/ui/tokens.go#L200) / [:491](../internal/ui/tokens.go#L491) | `Space(2)` = 8 | `px-3` = 10.5（`TimelineToolCallItem.vue:815-816`） | 部分 | 差 2.5 DIP |
| `framePad` [:204](../internal/ui/tokens.go#L204) / [:495](../internal/ui/tokens.go#L495) | 4.5 | `padding: 4.5px`（`A/styles/global.css:22`） | 对应 | 两边都是 px 字面量；A 在侧栏展开时左 padding 为 0（`:23`），B 同样左 0（`shell.go:367`） |
| `frameRadius` [:205](../internal/ui/tokens.go#L205) / [:496](../internal/ui/tokens.go#L496) | 14 | `rounded-xl`（`A/styles/global.css:21`）→ 默认 14px 根字号下 12.75（`:48,244`） | 部分 | B 固定 14；A 随根字号缩放，18px 档时是 15.25 |
| `heroMark` [:209](../internal/ui/tokens.go#L209) / [:498](../internal/ui/tokens.go#L498) | 44 | `BrandMark h-14` = 3.5rem = 49（`NewThread.vue:298`） | 部分 | B 无资产管线，用文字画标（[tokens.go:78-86](../internal/ui/tokens.go#L78-L86)），故意不追求等值 |
| `heroTitleSize` [:210](../internal/ui/tokens.go#L210) / [:499](../internal/ui/tokens.go#L499) | 2 × UI = 28 | `text-2xl` = 1.5rem = 21（`NewThread.vue:299`） | 部分 | B 比 A 大 7 DIP；A 还有 `font-semibold`，B 用 `Bold()`（`conversation.go:59`） |

### 2.2 时间线（消息行）

| B token（定义 / 赋值） | B 值 | A 来源 | 判定 | 差异 |
|---|---|---|---|---|
| `timelinePadX` [:216](../internal/ui/tokens.go#L216) / [:501](../internal/ui/tokens.go#L501) | 14 ×scale | `px-4`（`PlainTimeline.vue:3516`，行 shell） | 对应 | rem 换算值一致 |
| `timelineBlockGap` [:217](../internal/ui/tokens.go#L217) / [:502](../internal/ui/tokens.go#L502) | 10.5 ×scale | `mb-3`（`TimelineAssistantMessage.vue:968`、`TimelineThinkingItem.vue:96`、`TimelineToolCallItem.vue:730`） | 对应 | 一致 |
| `timelineHeaderH` [:218](../internal/ui/tokens.go#L218) / [:503](../internal/ui/tokens.go#L503) | 24.5 | `h-7`（`TimelineAssistantMessage.vue:906`） | 对应 | 一致 |
| `timelinePadStart` [:219](../internal/ui/tokens.go#L219) / [:504](../internal/ui/tokens.go#L504) | 10.5 ×scale | `pl-3`（`TimelineThinkingItem.vue:125`） | 对应 | 一致 |
| `timelineRail` [:220](../internal/ui/tokens.go#L220) / [:505](../internal/ui/tokens.go#L505) | 2 | `border-l-2`（`TimelineThinkingItem.vue:125`） | 对应 | 一致 |
| `timelinePanelMaxH` [:221](../internal/ui/tokens.go#L221) / [:506](../internal/ui/tokens.go#L506) | 240 | `max-h-[240px]`（`TimelineThinkingItem.vue:125`、`TimelineToolCallItem.vue:812`） | 对应 | px 字面量，一致 |
| `timelineChevron` [:222](../internal/ui/tokens.go#L222) / [:507](../internal/ui/tokens.go#L507) | 14 | `h-4 w-4`（`TimelineThinkingItem.vue:114`、`TimelineToolCallItem.vue:799`） | 对应 | 一致 |
| `timelineUserWidth` [:223](../internal/ui/tokens.go#L223) / [:508](../internal/ui/tokens.go#L508) | 85 | `max-w-[85%]`（`TimelineUserMessage.vue:901`） | 对应 | 一致 |
| `timelineUserPadY` [:224](../internal/ui/tokens.go#L224) / [:509](../internal/ui/tokens.go#L509) | 7 ×scale | `py-2`（`TimelineUserBubble.vue:5`） | 对应 | 一致 |
| `timelineUserPadX` [:225](../internal/ui/tokens.go#L225) / [:510](../internal/ui/tokens.go#L510) | 14 ×scale | `px-4`（`TimelineUserBubble.vue:5`） | 对应 | 一致 |
| `timelineBubbleR` [:226](../internal/ui/tokens.go#L226) / [:511](../internal/ui/tokens.go#L511) | 10 | `rounded-lg`（`TimelineUserBubble.vue:5`）→ 默认 8.75（`global.css:47,244`） | 部分 | B 固定 10；A 随根字号缩放 |
| `timelineFileRowY` [:227](../internal/ui/tokens.go#L227) / [:512](../internal/ui/tokens.go#L512) | 5.25 ×scale | `py-1.5`（`TimelineFileChangesView.vue:105`） | 对应 | 一致 |
| `timelineFileRowX` [:228](../internal/ui/tokens.go#L228) / [:513](../internal/ui/tokens.go#L513) | 7 ×scale | `px-2`（`TimelineFileChangesView.vue:105`） | 对应 | 一致 |
| `timelineFileGap` [:229](../internal/ui/tokens.go#L229) / [:514](../internal/ui/tokens.go#L514) | 10.5 ×scale | `gap-3`（`TimelineFileChangesView.vue:105`） | 对应 | 一致 |
| `timelineFileStatus` [:230](../internal/ui/tokens.go#L230) / [:515](../internal/ui/tokens.go#L515) | 17.5 | `w-5`（`TimelineFileChangesView.vue:112`） | 对应 | 一致 |

### 2.3 侧栏

| B token（定义 / 赋值） | B 值 | A 来源 | 判定 | 差异 |
|---|---|---|---|---|
| `sidebarHeaderH` [:236](../internal/ui/tokens.go#L236) / [:517](../internal/ui/tokens.go#L517) | 48 | `h-(--titlebar-height)`（`SidebarHeader.vue:87`；`global.css:174`） | 对应 | px 字面量 48，一致 |
| `sidebarHeaderTop` [:237](../internal/ui/tokens.go#L237) / [:518](../internal/ui/tokens.go#L518) | 4 ×scale | `mt-[4px]`（`SidebarHeader.vue:87`） | 对应 | 一致 |
| `sidebarSectionH` [:238](../internal/ui/tokens.go#L238) / [:519](../internal/ui/tokens.go#L519) | 31.5 | `h-9`（`ThreadListHeader.vue:92`、`PinnedThreads.vue:189`） | 对应 | rem 换算值一致 |
| `sidebarRowH` [:239](../internal/ui/tokens.go#L239) / [:520](../internal/ui/tokens.go#L520) | 34 | `h-[34px]`（`ProjectItem.vue:465`、`PinnedThreads.vue:206`） | 对应 | px 字面量一致；注意 flat 行用 `h-9`=31.5（`ThreadList.vue:248`），A 自己不一致 |
| `sidebarEntryH` [:240](../internal/ui/tokens.go#L240) / [:521](../internal/ui/tokens.go#L521) | 28 | `h-8`（`ThreadItem.vue:241`、`A/components/ui/sidebar/index.ts:27`） | 对应 | 一致 |
| `sidebarFooterH` [:241](../internal/ui/tokens.go#L241) / [:522](../internal/ui/tokens.go#L522) | 30 | `h-[30px]`（`AccountEntry.vue:206`） | 对应 | 一致 |
| `sidebarCell` [:242](../internal/ui/tokens.go#L242) / [:523](../internal/ui/tokens.go#L523) | 21 | `h-6 w-6`（`ThreadItem.vue:258,336`） | 对应 | 一致 |
| `sidebarGap` [:243](../internal/ui/tokens.go#L243) / [:524](../internal/ui/tokens.go#L524) | 3.5 ×scale | `gap-1`（`SidebarMenu.vue:14`） | 对应 | 一致 |
| `sidebarPadX` [:244](../internal/ui/tokens.go#L244) / [:525](../internal/ui/tokens.go#L525) | 7 ×scale | `px-2`（`SidebarHeader.vue:87`、`ThreadList.vue:173`、`AccountEntry.vue:205`） | 对应 | 一致 |
| `sidebarSectionPadX` [:245](../internal/ui/tokens.go#L245) / [:526](../internal/ui/tokens.go#L526) | 10.5 ×scale | `px-3`（`ThreadListHeader.vue:92`） | 对应 | 一致 |
| `sidebarW` [:249](../internal/ui/tokens.go#L249) / [:528](../internal/ui/tokens.go#L528) | 256 | `SIDEBAR_DEFAULT_WIDTH`（`A/components/ui/sidebar/utils.ts:4`） | 对应 | 一致 |
| `sidebarMinW` [:250](../internal/ui/tokens.go#L250) / [:529](../internal/ui/tokens.go#L529) | 200 | `SIDEBAR_MIN_WIDTH`（`utils.ts:5`） | 对应 | 一致 |
| `sidebarMaxW` [:251](../internal/ui/tokens.go#L251) / [:530](../internal/ui/tokens.go#L530) | 360 | `SIDEBAR_MAX_WIDTH`（`utils.ts:6`） | 对应 | 一致 |

### 2.4 字号、字体族与密度

| B token（定义 / 赋值） | B 值 | A 来源 | 判定 | 差异 |
|---|---|---|---|---|
| `uiFontSize` [:254](../internal/ui/tokens.go#L254) / [:532](../internal/ui/tokens.go#L532) | `snapFontSize(theme.FontSize)`，默认 14 | `--dimcode-ui-font-size: 14px`（`global.css:166`）；档位 `[12,13,14,16,18]`（`A/appearance/index.ts:12`） | 对应 | 档位表与默认值相同（[tokens.go:23-29](../internal/ui/tokens.go#L23-L29)、`A/appearance/index.ts:11-13`） |
| `codeFontSize` [:255](../internal/ui/tokens.go#L255) / [:533](../internal/ui/tokens.go#L533) | 13 | `--dimcode-code-font-size: 13px`（`global.css:167`） | 对应 | 一致 |
| `titleSize` [:256](../internal/ui/tokens.go#L256) / [:534](../internal/ui/tokens.go#L534) | 1.15 × UI = 16.1 | 没有语义变量：header 标题 `text-[13px]`（`ChatHeaderFrame.vue:64`）、侧栏品牌 `text-sm`=12.25（`SidebarHeader.vue:107`） | 部分 | B 的标题比 A 实际用值大 3–4 DIP；A 的 `text-lg`=15.75 是最近档，但没有统一使用 |
| `bodySize` [:257](../internal/ui/tokens.go#L257) / [:535](../internal/ui/tokens.go#L535) | UI = 14 | `text-base` = 1rem = 14（`TimelineAssistantMessage.vue:1085`、`TimelineUserBubble.vue:5`） | 对应 | 一致 |
| `labelSize` [:258](../internal/ui/tokens.go#L258) / [:536](../internal/ui/tokens.go#L536) | 0.9 × UI = 12.6 | `text-sm`=12.25（`ThreadItem.vue:241`）或 `text-[13px]`=13（`ChatHeaderFrame.vue:64`） | 部分 | B 落在两档之间；A 没有 label 语义档 |
| `smallSize` [:259](../internal/ui/tokens.go#L259) / [:537](../internal/ui/tokens.go#L537) | 0.8 × UI = 11.2 | `text-[11px]`=11（`TimelineToolCallItem.vue:815`）、`text-xs`=10.5（`global.css:40`） | 部分 | B 比 `text-xs` 大 0.7 DIP；状态栏文字因此略大 |

**半径阶梯（常量，不是 struct 字段）**：

| B 常量 | B 值 | A 变量（默认 14px 根字号） | 判定 |
|---|---|---|---|
| `radiusSM` [:53](../internal/ui/tokens.go#L53) | 6 | `--radius-sm` = 0.625rem − 4px = **4.75**（`global.css:45,244`；16px 根字号才是 6） | 部分：只在 16px 根字号成立 |
| `radiusMD` [:54](../internal/ui/tokens.go#L54) | 8 | `--radius-md` = **6.75**（`:46,244`；16px 根字号才是 8） | 部分：同上 |
| `radiusLG` [:55](../internal/ui/tokens.go#L55) | 10 | `--radius-lg` = **8.75**（`:47,244`；16px 根字号才是 10） | 部分：同上 |
| `radiusXL` [:56](../internal/ui/tokens.go#L56) | 14 | `--radius-xl` = **12.75**（`:48,244`；16px 根字号才是 14） | 部分：同上 |

**字号档位与默认值**：[fontSizeSteps](../internal/ui/tokens.go#L23) = `[12 13 14 16 18]` 与 `A/appearance/index.ts:12` 完全一致；`defaultUIFontSize`=14（[:28](../internal/ui/tokens.go#L28)）与 `DEFAULT_UI_FONT_SIZE`（`index.ts:11`）一致；`defaultCodeFontSize`=13（[:29](../internal/ui/tokens.go#L29)）与 `DEFAULT_CODE_FONT_SIZE`（`index.ts:13`）一致。

**字体族**：[monospaceFamily](../internal/ui/tokens.go#L47) = `"monospace"`，交给平台解析；A 自托管 Inter + JetBrains Mono（`global.css:10-18`，栈定义在 `global.css:157-165`）。判定 **部分**：B 没有自带字体文件，`fontFamilyStack`（[:378-383](../internal/ui/tokens.go#L378-L383)）只能表达“系统字体 / 平台 monospace”。

**密度**：[densityCompactScale](../internal/ui/tokens.go#L40) = 0.75，作用在 Spacing 和字面几何上。A 没有“间距密度”这个 token（`A/` 里 `grep density` 无命中）：A 的密度是字号档位 + 侧栏/右面板宽度持久化（`ChatWorkspace.vue:218-219`、`A/components/ui/sidebar/utils.ts:4-6`），不是 0.75 的全局缩放。判定 **无对应**。

### 2.5 状态栏（相邻文件，不在 tokens.go）

| B 常量 | B 值 | A 来源 | 判定 | 差异 |
|---|---|---|---|---|
| `statusBarH` [statusbar.go:27](../internal/ui/statusbar.go#L27) | 35 | `.chat-status-bar` 的 `h-10` = 2.5rem = 35（`global.css:39-41`） | 对应 | rem 换算值一致；使用点 [statusbar.go:138](../internal/ui/statusbar.go#L138) |
| `statusBarPadX` [statusbar.go:28](../internal/ui/statusbar.go#L28) | 17.5 | `px-5` = 1.25rem = 17.5（`global.css:40`） | 对应 | 一致 |
| `statusBarMeterW` [statusbar.go:29](../internal/ui/statusbar.go#L29) | 35 | `w-10` = 2.5rem（`ContextUsageMeter.vue:37`） | 对应 | 一致 |
| `statusBarMeterH` [statusbar.go:30](../internal/ui/statusbar.go#L30) | 3.5 | `h-1` = 0.25rem（`ContextUsageMeter.vue:37`） | 对应 | 一致 |

## 3. 单边 token（没有对应物）

### 3.1 B 有、A 没有对应 token

这些是 B 自造的值或派生规则。它们不是“错误”，但不能算 token 对齐。**注意：状态栏高度（35）和 `framePad`（4.5）看似 B 独有，实际在 A 有精确对应（`global.css:39-41`、`:22`），已放在表 2.1/2.5，不在这里。**

| 项 | B 定义 | 为什么 A 没有 |
|---|---|---|
| `toolbarPadY` = 5.5 | [tokens.go:482](../internal/ui/tokens.go#L482) | A header 只声明高度，没有纵向 padding |
| `headerH` = 41 | [tokens.go:483](../internal/ui/tokens.go#L483) | A 声明 48（`global.css:174`），41 是截图实测值（[ui-parity.md](ui-parity.md) §5.2），没有 CSS 变量 |
| `rowPadY` = 4、`rowPadX` = 16 | [tokens.go:476-477](../internal/ui/tokens.go#L476-L477) | A 的行距由 `space-y-4`（14）和 `px-4`（14）决定，不是成对 token；`rowPadX` 在 B 里也没有消费点 |
| `panelPad` = 24 | [tokens.go:486](../internal/ui/tokens.go#L486) | A 的新线程页 `p-8`=28、timeline 空态 `py-10`=35，没有统一空态内距 |
| `composerH` = 72 | [tokens.go:487](../internal/ui/tokens.go#L487) | A 只约束编辑器 min/max-height；**B 当前没有消费点** |
| `statusDot` = 8 | [tokens.go:488](../internal/ui/tokens.go#L488) | A 没有连接状态点 token |
| `heroMark` = 44 | [tokens.go:498](../internal/ui/tokens.go#L498) | A 用品牌资产 `h-14`=49；B 用文字画标，故意不同 |
| `densityCompactScale` = 0.75 | [tokens.go:40](../internal/ui/tokens.go#L40) | A 没有全局间距密度 |
| `washAlpha` = 0.12 | [tokens.go:166](../internal/ui/tokens.go#L166) / [:636](../internal/ui/tokens.go#L636) | A 每个 banner 自己写 alpha class（如 `bg-primary/5`），没有统一 wash 变量 |
| `sidebarRise` 0.03 / `cardRise` 0.12 / `popoverRise` 0.18 | [tokens.go:139-143](../internal/ui/tokens.go#L139-L143) | A 的 surface 直接写 oklch，没有“向白提升的比例” |
| `sidebarHoverRise` 0.06 / `sidebarSelectionRise` 0.18 | [tokens.go:156-160](../internal/ui/tokens.go#L156-L160) | A 用 12/14/18% alpha 叠层，不是向前景混合 |
| `tooltipBorderRise` 0.18 / `tooltipChipRise` 0.10 / `tooltipMutedFade` 0.30 | [tokens.go:149-151](../internal/ui/tokens.go#L149-L151) | A 直接给 tooltip 的 border/chip/muted 值 |
| `monospaceFamily` = `"monospace"` | [tokens.go:47](../internal/ui/tokens.go#L47) | A 自托管 JetBrains Mono 栈（`global.css:162-165`） |

### 3.2 A 有、B 没有对应物

按组列出，行号成对为 light / dark。

| A 变量组 | 行号 | B 现状 | 影响 |
|---|---|---|---|
| `--sidebar-overlay` | 235 / 317（`non-vibrant-window`: 420-431；reduced-transparency: 441-454） | 无字段；侧栏只有 `sidebar` 一个面，没有 overlay/抽屉底面 | 侧栏浮层、抽屉的独立底面无对应 |
| `--tooltip-backdrop`、`--tooltip-shadow` | 197 / 284；198-204 / 285-291 | 只有 5 个 tooltip 颜色字段（[tokens.go:308-312](../internal/ui/tokens.go#L308-L312)） | 磨砂模糊、6 层阴影、噪点纹理都没有 |
| `--card-frame-outer` | 226 / 308 | 无；卡片只用 `hairline` 实色描边（[shell.go:369](../internal/ui/shell.go#L369)） | 卡片的 0.5px 外缘阴影缺失 |
| `--meter-track` | 224 / 305 | 复用 `muted`（[statusbar.go:200](../internal/ui/statusbar.go#L200)） | A 用 alpha 轨道保证在任意表面上可见；B 在 card 上对比更低 |
| `--mark`、`--mark-soft` | 209-210 / 295-296 | 无 | 标注层没有语义槽 |
| `--chart-1..5` | 229-233 / 311-315 | 无 | 图表色板没有对应 |
| `--activity-info`、`--activity-violet`、`--activity-shimmer-highlight` | 338-339、337 / 364-365、363 | B 没有 activity 色组；`success`/`warning` 只借 `--activity-success/-warning` | info/violet/shimmer 三种状态色缺失 |
| `--markdown-*` 17 个 + `timeline-markdown.css` 的 `--ms-*`/`--inline-code-*` 调色板 | 343-359 / 369-385；`timeline-markdown.css:54-107` | `contentTheme` 从 tokens 派生链接、表格、代码块、复选框、分割线（[markdown_render.go:265-311](../internal/ui/markdown_render.go#L265-L311)） | 行内代码的 alpha 合成、代码高亮主题、表格斑马纹/行 hover、checkbox ring 没有对应 |
| `--dimcode-ui-font-family`、`--dimcode-code-font-family` | 157-161、162-165 | `fontFamilyStack` 只有系统/`monospace`（[tokens.go:378-383](../internal/ui/tokens.go#L378-L383)） | 字体族无法逐字体对齐 |
| `--brand-green`、`--brand-cta-ink` | 180-181 / 265-266 | 品牌色来自 `Theme.Accent`（蓝色），不是独立品牌 token | `primary`/`sidebar-primary`/`ring` 色相不同（见表 1） |
| `--titlebar-height`、`--win-titlebar-height` | 174、168 | 无标题栏 token；B 用原生系统标题栏，只有侧栏 header 借用 48 | 自绘 chrome 的 48/35 高度没有对应 |
| `--z-*` 5 个 | 250-254 | 无 | retained-mode 单一绘制列表，**合理差异** |
| `--layout-transition-ms`、`--ease-*`、panel/sidebar 动画 | 259、93-95、98-99、116-152 | B 没有过渡/动画 token（`grep -E 'Anim\|Duration\|Ease'` 只命中文案键名 `timeline.thinkingWithDuration`，不是动画 API） | 侧栏开合、面板进出、hover 过渡都缺失 |
| skin 覆盖层：`--skin-tone-*` 20 个 + `.skin-active.skin-host` 重绑 + `.non-vibrant-window` + `prefers-reduced-transparency` | 867-911；923-971；419-431；433-460 | 只有 system/light/dark + 外观设置；skin/chrome 是 WIP | A 的第三层颜色体系完全没有对应 |
| chrome 主题：`--chrome-*` 30 个变量、6 个 `data-chrome-theme` | `A/styles/retro-themes.css:3-955`（主题块 58/86/112/137/164/191） | 无 | 6 个 chrome 皮肤无法表达 |
| `--dimcode-code-theme-bg-light/dark` | 334-335 | `contentTheme` 用 `card` 当代码块底（[markdown_render.go:292-294](../internal/ui/markdown_render.go#L292-L294)） | 代码主题背景没有对应 |

## 4. 结论：D1 还差什么才算 1:1

按“是否阻塞 token 层 1:1”排序。以下每条都不是组件缺口，而是 token 语义或值本身对不上。

1. **品牌色相不同（最高优先级）。** A 的 `--primary`/`--sidebar-primary`（以及 dark 的 `--ring`）是 `--brand-green`（`global.css:180-181`、`237`、`265-266`、`319`、`310`）；B 的 `primary` 来自 `Theme.Accent`，light `#2563eb`、dark `#3b82f6`（`mygo ui/theme.go:176,205`）。这不是“近似”，是绿/蓝两种品牌。要 1:1，必须给 B 一套品牌主题（至少覆盖 Accent/AccentText、dark 的 onPrimary 近黑、sidebar-primary、dark ring）。
2. **半透明表面与 vibrancy 无对应。** A 的 sidebar 是 65%/60% alpha 叠在窗口 vibrancy 上（`global.css:234,316`），tooltip 是 50%/46% alpha + backdrop（`:192,279`），chrome 主题还会重绑。B 的颜色基本是单层实色（只有 `ring` 直接取 `Theme.Focus` 的 alpha），没有“表面 alpha 叠在窗口材质上”的合成层；`wash` 只做 12% alpha 的 banner（[tokens.go:636](../internal/ui/tokens.go#L636)）。要 1:1，至少要支持“表面 alpha + 合成顺序”，否则 sidebar/tooltip/skin 三层永远差一层。
3. **圆角阶梯实际值不对。** A 是 rem 阶梯，默认 14px 根字号下为 **4.75 / 6.75 / 8.75 / 12.75**（`global.css:45-48,244`）；B 固定 **6 / 8 / 10 / 14**（[tokens.go:53-56](../internal/ui/tokens.go#L53-L56)），只有 16px 根字号才对。连带 `frameRadius`（14 vs 12.75）、`timelineBubbleR`（10 vs 8.75）、composer 的 `cardRadius`（10 vs A 的 `rounded-xl` 12.75）都不对。要么把 B 改成 14px 根字号换算，要么在 D1 明确“以 16px 根字号为基准”并同步文档。
4. **阴影/玻璃 token 缺失。** A 有 `--card-frame-outer`（`global.css:226,308`）、6 层 `--tooltip-shadow`（`:198-204,285-291`）、composer 1px 外阴影（`ConversationComposer.vue:63`）、侧栏抽屉 `shadow-2xl`（`Sidebar.vue:145`）。B 只有 1 DIP 实色 `border`。若 mygo 不支持 box-shadow/backdrop-filter，应把这几项标 WIP，而不是用 border 冒充。
5. **A 的 rem 几何会随字号档位缩放，B 不会。** A 改 `--dimcode-ui-font-size` 后，`px-3`/`h-9`/`rounded-lg` 全部按 14→18 放大 28.6%（`global.css:390`、`appearance/index.ts:12`）；B 只有字号本身变，几何和圆角固定 DIP，只有 compact 密度乘 0.75。要 1:1，B 需要一个“几何随 UI 字号档位缩放”的开关（或明确接受这条差异）。
6. **`Space(n)` 与 Tailwind rem 刻度的 14% 偏差没有消掉。** `gapWide`=12 vs A `gap-2`=7、`edgePad`=12 vs `p-3`=10.5、`cardPadX`=8 vs `px-3`=10.5、`gap`=2 vs `gap-1`=3.5。时间线和侧栏已经按 14px 换算成字面量，外壳/卡片/composer 仍是 4 DIP 网格。D1 要把这些字段也按同一张换算表落地，或改 `Spacing`。
7. **标题字号偏大。** `titleSize`=16.1（1.15×14），A 实际用的是 header `text-[13px]`=13、侧栏品牌 `text-sm`=12.25（`ChatHeaderFrame.vue:64`、`SidebarHeader.vue:107`）；`heroTitleSize`=28 vs A `text-2xl`=21（`NewThread.vue:299`）。字号语义层需要以 A 的实际 utility 为准重标，而不是 1.15/0.9/0.8 的经验系数。
8. **`headerH` 41 没有 CSS 来源。** A 声明 48（`global.css:174`），截图量到 41（[ui-parity.md](ui-parity.md) §5.2），但现有记录没有说明量的是哪一个盒。D1 需要先统一测量口径，再决定以 48 还是 41 为准；在决定前，这个 token 不能算对齐。
9. **缺失的语义槽。** `--sidebar-overlay`、`--meter-track`、`--card-frame-outer`、`--mark/-soft`、`--chart-1..5`、`--activity-info/-violet/-shimmer-highlight`、markdown 行内代码 alpha、`--dimcode-code-theme-bg-*`。其中 `meter-track`/`sidebar-overlay`/`card-frame-outer` 是壳层可见的，应优先补字段。
10. **动效与 z 层。** `--layout-transition-ms`=250ms、`--ease-*`、panel-in 250ms / panel-out 200ms（`global.css:93-95,98-99,116-152,259`）。B 没有过渡 token；z 层在 retained-mode 下可视为合理差异，但过渡不是。

**一句话**：颜色层 35/35 有语义落点，但 16 个只能算“部分”（alpha/玻璃/品牌色/状态色），几何层 53/53 有落点，其中 31 个数值一致、14 个近似、8 个无精确对应。**当前 token 层不是 1:1**；D1 的最小闭环是：品牌主题、alpha 表面、圆角/rem 换算、阴影/玻璃、字号语义、缺失语义槽六项，其余（skin/chrome、动效、markdown 全色板）可以显式标 WIP。

## 5. 计数与方法

- 表 1：35 行 = `tokens` struct 的全部 `ui.Color` 字段（[tokens.go:263-312](../internal/ui/tokens.go#L263-L312)）；判定 19 对应 / 16 部分 / 0 无。
- 表 2：53 行 = 非颜色字段（53）+ 主题指针 1 = 88 个 struct 字段，与 [ui-parity.md](ui-parity.md) §结论 的“88 个字段”一致。判定 31 对应 / 14 部分 / 8 无精确对应。另列半径 4、字号档位 5、默认值 2、状态栏 4。
- A 侧统计口径：`global.css` 中按“行首 `--name:` 定义”去重后共 158 个变量，其中 122 个非 `--color-*`；`--color-*` 是 `@theme inline`（50-87）把语义变量映射给 Tailwind 的别名层，不是新颜色。`retro-themes.css` 另有 30 个 `--chrome-*` 变量，`timeline-markdown.css` 的 `--ms-*`/`--inline-code-*`/`--code-*` 是 markstream 调色板，单独成组。
- 本文没有做像素取样；“值”列引用的是 A 源码原文或已有的截图取样，取样结果以 [ui-parity.md](ui-parity.md) §5.2 为准。
