# P0 区域布局规格（dim-electron renderer → dimagent-mygo）

Status: draft
Last updated: 2026-11-09
证据等级：**L0 `source-read`**（只读源码，没有运行窗口，也没有截图对照；见 [acceptance.md](acceptance.md) 的证据等级表）

本文把 dim-electron renderer 的 P0 区域布局拆成可直接实现的规格：容器树、布局属性、数值来源（Tailwind class / CSS 变量 + 行号）、响应式与折叠条件、滚动与虚拟化，以及对应 MyGo 的写法和缺口。

## 0. 读法与全局事实

**引用前缀**（行号为 2026-11-09 工作区状态）：

| 前缀 | 含义 |
|---|---|
| `A/` | `dim-electron/packages/desktop/packages/renderer/src/` |
| `B/` | 本仓库根（`dimagent-mygo/`） |
| `mygo ` | `github.com/egoist/mygo/ui` v0.8.3（`B/go.mod:5`），本地 checkout `/tmp/mygo-research/ui` |

**五条会影响所有区域的事实**，后面不再重复解释：

1. **根字号是 14px，不是 16px。** `html { font-size: var(--dimcode-ui-font-size) }`（`A/styles/global.css:389-391`），默认 `--dimcode-ui-font-size: 14px`（`A/styles/global.css:166`），档位 `[12, 13, 14, 16, 18]`（`A/appearance/index.ts:12`）。所以 **所有 rem 类（`text-sm`、`max-w-4xl`、`rounded-xl`、`h-9`）都随字号档位缩放**。
2. **间距类不能用 4px 网格换算。** 仓库没有覆盖 Tailwind 的 `--spacing`（`A/styles/global.css` 中无该变量），`p-3` / `gap-2` / `px-4` 走 Tailwind 默认的 `0.25rem` 刻度；在 14px 根字号下每档 **3.5px**。MyGo `theme.Space(n)` 是 `n × Spacing`，`Spacing` 默认 **4**（`mygo ui/theme.go:86-92`）。直接对档号调参会有约 14% 偏差，映射时要按 DIP 换算，不要照抄档号。
3. **px 字面量是刻意不缩放的。** `--titlebar-height: 48px`（`A/styles/global.css:174`）承载 macOS 红绿灯（同文件 `170-173` 的注释），`--win-titlebar-height: 2.5rem`（`A/styles/global.css:168`）则是 rem，会缩放。
4. **断点按内容宽度，不按媒体查询。** `contentWidth < 720` → `compact`，`< 960` → `regular`，否则 `wide`（`A/stores/layout.ts:6-13`）；`contentWidth` 由 `App.vue` 的 `ResizeObserver` 写入（`A/app/App.vue:72-74`）。
5. **布局层级有固定 z 契约。** `--z-sidebar-base: 0`、`--z-content: 10`、`--z-content-frame: 20`、`--z-overlay-scrim: 40`、`--z-sidebar-floating: 50`（`A/styles/global.css:250-254`），统一过渡 `--layout-transition-ms: 250ms`（`A/styles/global.css:259`，常量见 `A/composables/useLayoutTransition.ts:8`）。

---

## 1. 应用外壳（titlebar + sidebar 壳 + main）

### 1.1 容器树

```text
#root                          block, h-screen w-screen                     A/app/App.vue:77,240
├─ titlebar                    block flow, shrink-0                          A/app/App.vue:242-245
│                              h = --titlebar-height（自绘 chrome）
│                              或 h = --win-titlebar-height（默认 chrome，非 mac）
└─ 内容区                       block, h = calc(100vh - 标题栏高)              A/app/App.vue:248
   └─ SidebarProvider          <div> flex row, h-full w-full                 A/components/ui/sidebar/SidebarProvider.vue:168-173
      │                        内联写 --sidebar-width = effectiveWidth px
      ├─ workspace 行           flex row, h-full w-full, overflow hidden      A/app/App.vue:257
      │  ├─ Sidebar 壳          absolute inset-y-0 left-0, w=(--sidebar-width)
      │  │                     pointer-events-none（只有内部可命中）           A/components/ui/sidebar/Sidebar.vue:122-131
      │  │  └─ aside           absolute, flex column, overflow hidden, ✗滚动   A/components/ui/sidebar/Sidebar.vue:138-171
      │  └─ SidebarInset       <main> flex column, margin-left=(--sidebar-width)
      │                        overflow hidden, z=(--z-content)              A/components/ui/sidebar/SidebarInset.vue:62-78
      │     └─ 内容框          flex column, overflow hidden
      │                        chat 路由：无 margin/无 ring（卡片自绘）
      │                        其他路由：m-[4.5px] rounded-xl ring-[0.5px]     A/app/App.vue:278-287
      │        └─ RouterView   ChatView / SkillsView / ScheduledView …
      ├─ compact 遮罩          fixed inset-0, z=[--z-overlay-scrim + 2]       A/app/App.vue:294-304
      └─ portal root           fixed inset-0, z=(--z-sidebar-floating)        A/components/ui/sidebar/SidebarProvider.vue:182-186
```

### 1.2 布局属性

| 属性 | 值 | 来源 |
|---|---|---|
| 根尺寸 | `h-screen w-screen` | `A/app/App.vue:77` |
| 标题栏高（自绘 chrome） | `h-(--titlebar-height)` = `48px` | `A/components/app/AppTitlebar.vue:16`；`A/styles/global.css:174` |
| 标题栏高（默认 chrome，非 mac） | `h-(--win-titlebar-height)` = `2.5rem` | `A/app/App.vue:243`；`A/styles/global.css:168` |
| 内容区高 | `h-screen`（mac）／`calc(100vh-var(--titlebar-height))`／`calc(100vh-var(--win-titlebar-height))` | `A/app/App.vue:248` |
| 侧栏宽 | `var(--sidebar-width)`，运行时 = `effectiveWidth` px | `A/components/ui/sidebar/Sidebar.vue:126`；`SidebarProvider.vue:170-173` |
| 侧栏默认／最小／最大宽 | 256 / 200 / 360 px | `A/components/ui/sidebar/utils.ts:4-6` |
| 侧栏有效宽 clamp | `min(width, max(200, contentWidth - 408))`（仅 `regular`） | `A/components/ui/sidebar/SidebarProvider.vue:115-118` |
| main 左让位 | `margin-left: var(--sidebar-width)`；`transition: margin-left 250ms ease-out` | `A/components/ui/sidebar/SidebarInset.vue:62-68` |
| 非 chat 内容框 | `m-[4.5px] rounded-xl` + 内 0.5px ring + 外 0.5px shadow | `A/app/App.vue:280` |
| 裁剪 | workspace / inset / 内容框 全部 `overflow-hidden` | `A/app/App.vue:257,272`；`SidebarInset.vue:74` |
| 侧栏展开动画 | `transition-[opacity,transform] duration-[250ms]` | `A/components/ui/sidebar/Sidebar.vue:143` |
| 折叠态位移 | `translateX(-8px)` + `opacity-0` + `inert` | `A/components/ui/sidebar/Sidebar.vue:158-167` |
| 边缘热区（折叠时唤出） | `w-[12px]`，`inset-y-0 left-0` | `A/components/ui/sidebar/Sidebar.vue:194-198` |
| 拖拽调宽手柄 | `absolute inset-y-0 right-0 w-1` | `A/components/ui/sidebar/Sidebar.vue:184-189` |
| compact 抽屉 | `fixed inset-0` + `bottom-[5px] max-w-[85vw] rounded-lg shadow-2xl` | `A/components/ui/sidebar/Sidebar.vue:127,145` |
| hover 预览浮层 | `top-[5px] bottom-[5px] rounded-lg shadow-2xl`，z=`--z-overlay-scrim` | `A/components/ui/sidebar/Sidebar.vue:146-147` |

### 1.3 响应式与折叠条件

| 条件 | 行为 | 来源 |
|---|---|---|
| `contentWidth < 720`（compact） | 侧栏不占位（`SidebarInset.shift` 返回 false）；改成抽屉：`fixed` + z-50 + 遮罩；Esc 关闭、路由变化关闭 | `A/components/ui/sidebar/SidebarInset.vue:25-28`；`Sidebar.vue:145`；`A/app/App.vue:180-194` |
| 抽屉顶边 | mac `0px`，其他 `var(--win-titlebar-height, 0px)` | `A/components/ui/sidebar/Sidebar.vue:164` |
| 折叠（非 compact） | aside 只做 opacity/位移，不参与流；主内容 `margin-left: 0` | `A/components/ui/sidebar/Sidebar.vue:148-160`；`SidebarInset.vue:25-28` |
| 折叠态指针进入左缘 12px | hover 预览浮起（不改持久 `open`） | `A/components/ui/sidebar/Sidebar.vue:45,73-77,194-198` |
| `regular` + 右面板打开 | 自动收起侧栏；面板关闭再展开 | `A/components/chat-workspace/ChatWorkspace.vue:2044-2052` |
| `wide` + 需要宽度不足 | 需要宽 = `sidebarW + panelW + 400`；不足则收，回弹需多 40px（迟滞） | `A/components/chat-workspace/ChatWorkspace.vue:2032-2036,2058-2064`；`223`；`222` |
| 打开/收起过程中 | `openAnimating` 期间时间线不跟随中间宽度 | `A/components/ui/sidebar/SidebarProvider.vue:78-90`；`A/components/chat-workspace/timeline/PlainTimeline.vue:750` |

### 1.4 滚动与虚拟化

外壳本身**没有滚动容器**。滚动全部下沉：侧栏用 `ScrollArea`（§2.4），会话用 `.chat-scroll-viewport`（§3.4）。

### 1.5 MyGo 映射

```go
// 外壳：A 的 flex 行要求子项纵向拉伸，MyGo Row 默认居中（mygo ui/api_gen.go:2110-2111），必须显式 Stretch。
ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
    // 标题栏：px 值固定 48（A 的 --titlebar-height 是 48px 字面量）
    ui.Row(c).Height(48).PaddingX(k.space(0.5)).AlignItems(ui.Center).Children(/* brand, spacer */)

    // 侧栏：不参与流 + inset 用 margin-left 让位（A 是遮挡式 wipe）
    sidebarView(c, k)                       // Width(k.sidebarW).Shrink(0)
    ui.Column(c).Grow(1).Clip().MarginStart(offset).AlignItems(ui.Stretch).Children(mainView)

    // 内容框：非 chat 路由的 4.5px 外框 + 0.5px ring，用 Padding + Border + Radius 表达
    ui.Column(c).Padding(4.5).Border(k.hairline/2, k.border).Radius(radiusXL).Children(content)
})
```

`ui.Row` / `ui.Column` / `ui.Box` 都是同一个盒子，只差方向（`mygo ui/widgets.go:12-20`；文档 `mygo ui/api_gen.go:982-983,1227-1228,2110-2111`）。`Box` 与 `Column` 等价，写 `Column` 更直白。

**缺口：**

- **没有 CSS 变量与 `calc()`。** A 的 `margin-right: calc(var(--right-panel-width) + 4.5px)`（`A/components/chat-workspace/ChatWorkspace.vue:2014-2018`）必须在 Go 里算成 `float32`；`--sidebar-width` 这种"由祖先写入、后代读取"的隐式传值，在 MyGo 里只能靠 `tokens` / 状态字段显式传。
- **没有 `overflow: visible` 与滚动容器之外的可裁剪容器区分。** `Clip()` 只裁子节点绘制（`mygo ui/api_gen.go:4155-4156`），`Scroll` 才是滚动容器（`mygo ui/api_gen.go:2123-2124`）。A 的 `overflow-hidden` 多数是"裁剪"，对应 `Clip()`。
- **没有 `-webkit-app-region: drag`。** A 用 `window-drag` 类把标题栏交给系统拖动（`A/components/app/AppTitlebar.vue:16`）。MyGo 窗口拖动属于窗口/标题栏模型，不是视图属性（`TesterOptions.TitleBar`、`Tester.SetTitleBar`），不在 P0 布局范围，但会影响像素对照的窗口外框。
- **`z-[calc(var(--z-overlay-scrim)+2)]` 这类表达式无法表达。** `ZIndex(int)` 在（`mygo ui/api_gen.go:8316`），但没有变量加法的写法；契约值要在 Go 里算好再传。
- **没有 `inert`。** A 在折叠/被遮挡时设 `inert` + `aria-hidden`（`Sidebar.vue:166-167`；`ChatWorkspace.vue:6489-6490`）。MyGo 有 `Disabled`、`AccessHidden`、`IgnoreInput`，语义要分开选：`IgnoreInput(true)` 最接近 inert 的输入语义。

---

## 2. Sidebar

### 2.1 容器树

```text
aside                    absolute, flex column, overflow hidden               A/components/ui/sidebar/Sidebar.vue:138-150
└─ 内层容器               flex column, h-full w-full, rounded-lg（浮层态）        A/components/ui/sidebar/Sidebar.vue:172-181
   └─ .app-sidebar        relative, flex column, height 100%, min-height 0     A/components/sidebar/AppSidebar.vue:132-139
      ├─ SidebarHeader    <header> shrink-0, h=(--titlebar-height), flex row,
      │                   px-2, mt-[4px], justify-between, ✗滚动                  A/components/sidebar/SidebarHeader.vue:85-90
      ├─ GlobalEntries    block, p-2                                            A/components/sidebar/GlobalEntries.vue:62
      │  └─ ul            flex column, gap-1                                     A/components/ui/sidebar/SidebarMenu.vue:14
      │     └─ li > button h-8 w-full, gap-2, px-2, rounded-md, text-sm          A/components/ui/sidebar/index.ts:18,27
      ├─ ScrollArea       flex-1 min-h-0 ← 唯一滚动容器                           A/components/sidebar/AppSidebar.vue:87
      │  ├─ PinnedThreads flex column, pb-2（有内容才渲染）                        A/components/sidebar/PinnedThreads.vue:178-181
      │  │  ├─ 标题行      h-9, px-3, justify-between                            PinnedThreads.vue:188-193
      │  │  └─ 列表       mt-1, flex column, gap-0.5, px-2 → 行 h-[34px]          PinnedThreads.vue:196,206
      │  └─ ThreadList    flex column, pb-2                                     A/components/sidebar/ThreadList.vue:167
      │     ├─ 标题行      h-9, px-3, justify-between                            A/components/sidebar/ThreadListHeader.vue:91-93
      │     └─ 列表容器    px-2, py-1                                           ThreadList.vue:173
      │        ├─ 分组视图  ProjectItem
      │        │  ├─ 项目行 py-1, px-2, rounded-md, hover=bg-sidebar-accent      ProjectItem.vue:285
      │        │  └─ 线程列 flex column, gap-0.5, pb-1 → 行 h-[34px]             ProjectItem.vue:451,465
      │        └─ flat 视图 日期段 + 行 h-9                                      ThreadList.vue:247-251
      └─ AccountEntry     shrink-0, px-2, pb-2，贴底，不进滚动区                    A/components/sidebar/AccountEntry.vue:205-206
```

### 2.2 布局属性

| 属性 | 值 | 来源 |
|---|---|---|
| 侧栏整体 | `flex h-full w-full flex-col`，外层 `overflow-hidden`，`min-height: 0` | `A/components/sidebar/AppSidebar.vue:132-139,174` |
| Header 高 | `h-(--titlebar-height)`（=48px） | `A/components/sidebar/SidebarHeader.vue:87` |
| Header 内边距 / 上边距 | `px-2` / `mt-[4px]` | 同上 `:87` |
| Header 透明占位（mac） | `w-20 h-3 shrink-0` | `A/components/sidebar/SidebarHeader.vue:92-96` |
| Header 品牌块 | `h-4 w-[23px]` + `gap-2 pl-2` | `A/components/sidebar/SidebarHeader.vue:99-107` |
| 更新按钮 | `h-7 min-w-[5.5rem] max-w-[7.5rem] gap-1 px-2 text-xs` | `A/components/sidebar/SidebarHeader.vue:120` |
| 全局入口区 | `p-2`；条目 `h-8`、`gap-2`、`px-2`、`rounded-md` | `A/components/sidebar/GlobalEntries.vue:62`；`A/components/ui/sidebar/index.ts:18,27` |
| 菜单纵向间距 | `gap-1` | `A/components/ui/sidebar/SidebarMenu.vue:14` |
| 列表容器 | `px-2 py-1` | `A/components/sidebar/ThreadList.vue:173` |
| section 标题行 | `h-9 px-3`，文字 `text-xs` | `A/components/sidebar/ThreadListHeader.vue:91-94`；`PinnedThreads.vue:189-192` |
| 线程行 | `h-8 w-full gap-1 rounded-md border-0 px-1 text-sm leading-5` | `A/components/sidebar/ThreadItem.vue:241` |
| 行内状态格 | `h-6 w-6 shrink-0`（图标 `w-3.5 h-3.5`） | `A/components/sidebar/ThreadItem.vue:258,260-297` |
| 行内尾部格 | `h-6 w-6 shrink-0`（时间 `text-[11px]`） | `A/components/sidebar/ThreadItem.vue:336-352` |
| 待办 tag | `px-1.5 py-0.5 ml-1.5 rounded-full`，文字 `text-[10px]` | `A/components/sidebar/ThreadItem.vue:308-313` |
| 快捷键提示 | `absolute right-1 top-1/2 h-6 min-w-6 rounded-full px-2` | `A/components/sidebar/ThreadItem.vue:327-332` |
| 项目行 | `py-1 px-2 rounded-md`，名称 `text-sm font-medium truncate` | `A/components/sidebar/ProjectItem.vue:285,339-341` |
| 项目线程列 | `flex flex-col gap-0.5 pb-1`，行高 `h-[34px]` | `A/components/sidebar/ProjectItem.vue:451,465` |
| "显示更多" | `pl-[26px] pr-2 py-1.5 text-xs`（与状态格对齐） | `A/components/sidebar/ProjectItem.vue:472-479` |
| 置顶行 | `h-[34px] rounded-sm` | `A/components/sidebar/PinnedThreads.vue:206` |
| flat 行 | `h-9`；日期段 `px-2 pt-2 pb-0.5 text-xs` | `A/components/sidebar/ThreadList.vue:248-250` |
| footer 容器 | `shrink-0 px-2 pb-2` | `A/components/sidebar/AccountEntry.vue:205` |
| footer 行高 | `h-[30px]`；按钮 `rounded-lg pl-2 pr-1 gap-2` | `A/components/sidebar/AccountEntry.vue:206,213` |
| footer 头像 | `h-[18px] w-[18px] rounded-full`，字 `text-[9px]` | `A/components/sidebar/AccountEntry.vue:217` |
| footer 设置键 | `size-[30px] rounded-lg` | `A/components/sidebar/AccountEntry.vue:366` |
| 拖入提示层 | `absolute inset-1 z-30 rounded-lg border-dashed px-4 gap-2` | `A/components/sidebar/AppSidebar.vue:116` |
| 项目间分隔线 | `mx-2 my-1 h-px` | `A/components/sidebar/ProjectItem.vue:511-515` |

### 2.3 响应式与折叠条件

| 条件 | 行为 | 来源 |
|---|---|---|
| compact（<720） | 抽屉：`fixed inset-0`，`max-w-[85vw]`，`rounded-lg shadow-2xl`，z=`--z-sidebar-floating`；点 `[data-sidebar-destination]` 或 Esc 关闭 | `A/components/ui/sidebar/Sidebar.vue:145,94-99`；`A/app/App.vue:180-188` |
| 用户拖宽 | `setWidth(clamp(width))`，松手写回 layout-state | `A/components/ui/sidebar/Sidebar.vue:103-107`；`SidebarProvider.vue:122-131` |
| `regular` 下超宽 | 侧栏有效宽被 `contentWidth - 408` 压 | `SidebarProvider.vue:117` |
| 项目折叠 | 子列表 `v-if="isNearViewport || hasProjectFocus"`，折叠时不建 DOM；折叠期间有线程在跑则显示 loading 点 | `A/components/sidebar/ProjectItem.vue:448`；`:136-137,349-356` |
| 项目数超过 5（默认阈值） | 出现"显示更多"，每次 +20 | `A/components/sidebar/AppSidebar.vue:30-34`；`ProjectItem.vue:139,180-188` |
| flat 视图 | 首次 20 条，每次 +20；日期分段随可见片计算 | `A/components/sidebar/ThreadList.vue:66-70,92-99` |
| 办公模式 | flat/group 切换器隐藏，强制 group | `A/components/sidebar/ThreadList.vue:62-64`；`ThreadListHeader.vue:109` |

### 2.4 滚动与虚拟化

| 项 | 事实 | 来源 |
|---|---|---|
| 滚动容器归属 | 唯一：`ScrollArea`（`flex-1 min-h-0`），viewport 高 100% | `A/components/sidebar/AppSidebar.vue:87,142-144` |
| 虚拟化实现 | `useSidebarVirtualRows(items, rowHeight)`：手势/尺寸变化时重算窗口 | `A/components/sidebar/useSidebarVirtualRows.ts:4-31` |
| 视口探测 | `element.closest('[data-slot="scroll-area-viewport"]')` | 同上 `:24` |
| 过扫（overscan） | 上下各 8 行；`end` 初值 40 | 同上 `:7,15-16` |
| 行高 | flat 36；项目内 34；置顶 34 | `ThreadList.vue:109`；`ProjectItem.vue:153`；`PinnedThreads.vue:39` |
| 占位策略 | 上下 `paddingTop/paddingBottom = 行数 × rowHeight` | `useSidebarVirtualRows.ts:39-40` |
| 列表 key | `sessionId`（flat / 项目 / 置顶一致） | `ThreadList.vue:249`；`ProjectItem.vue:465`；`PinnedThreads.vue:205` |
| 已知不一致 | flat 行用 `h-9`（rem），虚拟化按 36px 常量算 → 改字号档位后行高与占位会漂 | `ThreadList.vue:109` vs `:248` |

### 2.5 MyGo 映射

```go
ui.Column(c).Width(k.sidebarW).Shrink(0).Clip().Children(func() {
    header(c, k)                                  // Height(48).PaddingX(k.space(0.5))
    entries(c, k)                                 // Padding(k.space(0.5)) + Column().Gap(GapY)
    ui.List(c, &s.threads, len(s.rows), func(i int) {
        threadRow(c, k, s.rows[i])                // h 32：Row().Height(32).Gap(...).PaddingX(...)
    }).Grow(1).ItemKey(func(i int) any { return s.rows[i].ID })
    footer(c, k)                                  // Height(30) 行 + Padding(k.space(0.5))
})
```

**缺口：**

- **`ui.Sidebar` 的行模型不够。** `SidebarItem` 是固定的 `Row`：`Gap(Space(2))`、`Padding(Space(1.5), Space(2))`、`Shrink(0)`，只承诺 icon + label + `Children`（`mygo ui/sidebar.go:161-200`，文档 `mygo ui/api_gen.go:2281-2310`）。A 的线程行是四段：24px 状态格 + 弹性标题 + 24px 尾部 + 相对时间，且尾部在 hover 时换成归档按钮（`A/components/sidebar/ThreadItem.vue:258-367`）。**P0 用 `ui.List` + 自绘 `ui.Row`，不要用 `ui.SidebarItem`。**
- **侧栏的"绝对层 + inset margin"没有对应物。** MyGo 的 `Split` 是有流分栏 + 可拖分隔条（`mygo ui/api_gen.go:2409-2417`），最小边 40 DIP（同文件 `2412-2413`），与 A 的 200/360 clamp（`A/components/ui/sidebar/utils.ts:4-6`）不同。要 1:1 的遮挡式开合动画就自绘 absolute 层；用 `Split` 则接受模型差异，并在 `Changed` 后自行 clamp。
- **三段式行里的右对齐尾巴需要自己加 `Spacer`。** MyGo 有 `ui.Spacer`（`mygo ui/api_gen.go:2379-2387`），可行，但 `SidebarItem` 不会替你放。
- **`text-[11px]` / `text-[10px]` 没有档位。** B 的字号 token 只有 5 档 UI + 1 档 code（`B/internal/ui/tokens.go:111-117,242-247`）。P0 统一到 `smallSize`，或补一档 `microSize`。
- **hover 才出现的按钮需要按 hover 重建。** MyGo 的 `Hovered()` 在指针进出时重建（`mygo ui/api_gen.go:5479`），在 `List` 里要用 `ListRow.Hovered()`（`mygo ui/list_builder.go:17-20`），否则 hover 不会刷新。
- **项目拖拽排序无公共 API。** `Draggable` / `DragData` 存在（`mygo ui/api_gen.go:4779,4660`），但 A 用的是 HTML5 DnD + 插入指示线（`A/components/sidebar/ThreadList.vue:184-222`）。P0 标 WIP。

---

## 3. Chat workspace（header、timeline 滚动区、composer）

### 3.1 容器树

```text
chat-workspace-frame     flex column, h-full w-full, overflow hidden, rounded-xl
                         framed 时 padding 4.5px，且侧栏打开时 padding-left 0        A/styles/global.css:20-25
└─ body                  relative, flex row, min-h-0, flex-1                        A/components/chat-workspace/ChatWorkspace.vue:6469-6472
   ├─ chat card          flex-1, flex column, h-full, min-w-0, bg-background,
   │                     rounded-xl, z=(--z-content)                                global.css:27-37；ChatWorkspace.vue:6479-6492
   │  ├─ ChatHeaderFrame <header> h=(--titlebar-height), px-3, shrink-0,
   │  │                     overflow-visible                                        A/components/chat-workspace/ChatHeaderFrame.vue:42-43
   │  ├─ timeline 容器    relative flex column, min-h-0 flex-1, overflow hidden,
   │  │                     container-type: inline-size                              ChatWorkspace.vue:6537-6542；:7185-7187
   │  │  ├─ rail 层       absolute inset-0, z-30, pointer-events-none                ChatWorkspace.vue:6543-6552
   │  │  ├─ 滚动区容器    relative, min-h-0, flex-1                                  ChatWorkspace.vue:6554
   │  │  │  └─ PlainTimeline
   │  │  │     └─ .chat-scroll-viewport  flex column, h-full, overflow-y auto ←滚动    A/components/chat-workspace/timeline/PlainTimeline.vue:3452-3456
   │  │  │        └─ 内容列  mx-auto, max-w-4xl, px-3, pt-6, pb-6                     PlainTimeline.vue:3457-3458
   │  │  └─ 底部栈        relative, z-10, shrink-0（不滚动）                            ChatWorkspace.vue:6622-6626
   │  │     └─ 内层       mx-auto, max-w-4xl                                         ChatWorkspace.vue:6628-6631
   │  │        ├─ 活动条  mx-3 mb-2                                                 ChatWorkspace.vue:6633-6637
   │  │        ├─ 追问面板 FollowUpMessages（可折叠）                                  ChatWorkspace.vue:6652-6662
   │  │        ├─ 审批面板 或 InputArea（二选一）                                       ChatWorkspace.vue:6665-6688
   │  │        └─ StatusBar  .chat-status-bar h-10 px-5                              global.css:39-41；ChatWorkspace.vue:6689-6692
   │  └─ right panel aside（P0 之外）
```

### 3.2 布局属性

| 属性 | 值 | 来源 |
|---|---|---|
| 外框 | `flex h-full w-full flex-col overflow-hidden rounded-xl` | `A/styles/global.css:20-21` |
| 外框 padding | `4.5px`；`[data-sidebar-open="true"]` 时 `padding-left: 0` | `A/styles/global.css:22-23` |
| 卡片 | `relative flex h-full min-w-0 flex-col bg-background rounded-xl` | `A/styles/global.css:28` |
| 卡片描边 | `::after` 内 0.5px ring + 外 0.5px shadow（`--card-frame-outer`） | `A/styles/global.css:29-36`；`A/styles/global.css:226` |
| 卡片让位 | `margin-right: calc(var(--right-panel-width) + 4.5px)`（面板开时） | `A/components/chat-workspace/ChatWorkspace.vue:2014-2018` |
| 圆角 | `rounded-xl` → `--radius-xl = calc(var(--radius) + 4px)`，`--radius: 0.625rem` | `A/styles/global.css:43-48,244` |
| header 高 | `h-(--titlebar-height)`（48px） | `A/components/chat-workspace/ChatHeaderFrame.vue:43` |
| header 内边距 | `px-3`，标题组 `mx-3 gap-1.5` | `ChatHeaderFrame.vue:43,60` |
| header mac 占位（侧栏关） | `w-[80px]` | `ChatHeaderFrame.vue:49` |
| 工作区面包屑 | `max-w-[8rem]`，`lg:max-w-[12rem]`，`text-sm truncate` | `ChatHeaderFrame.vue:61-62` |
| 会话标题 | `text-[13px] font-medium truncate` | `ChatHeaderFrame.vue:64` |
| header 右侧动作 | `gap-1`；按钮 `h-8 w-8` | `ChatHeaderFrame.vue:67`；`A/components/chat-workspace/ChatHeader.vue:584-594` |
| timeline 内容列 | `max-w-4xl`（rem 类，56rem；14px 根字号下 784px）+ `px-3 pt-6 pb-6` | `PlainTimeline.vue:3458` |
| 行外层 | `w-full px-4`；`turn` 行内 `space-y-4` | `PlainTimeline.vue:3515-3517` |
| 空态 | `flex min-h-full flex-1 items-center justify-center py-10 text-sm` | `PlainTimeline.vue:3462-3464` |
| 底部留白 | `.timeline-bottom-spacer { height: 40px }` | `PlainTimeline.vue:3721-3723` |
| 回底按钮 | `absolute bottom-4 left-1/2 h-8 w-8 rounded-full border bg-background shadow-md` | `PlainTimeline.vue:3610-3615` |
| 滚动条 | `width/height: 10px`；thumb `min-height: 24px`；默认透明，hover/focus 才显色 | `PlainTimeline.vue:3656-3688`；`A/components/chat-workspace/ChatWorkspace.vue:7246-7278` |
| 滚动条占位 | `scrollbar-gutter: stable` | `PlainTimeline.vue:3627`；`ChatWorkspace.vue:7225` |
| 滚动锚定 | `overflow-anchor: none`（viewport）/`auto`（已完成的启用行） | `ChatWorkspace.vue:7224`；`PlainTimeline.vue:3714-3719` |
| 底部栈 | `relative z-10 shrink-0` + 内层 `mx-auto w-full max-w-4xl` | `ChatWorkspace.vue:6625-6631` |
| 活动条 | `mx-3 mb-2` | `ChatWorkspace.vue:6637` |
| 输入区外层 | `px-3`（compact overlay / 全宽面板时 `px-0`） | `ChatWorkspace.vue:6675-6680` |
| composer 卡片 | `flex-col gap-2 p-3 rounded-xl border` + 外 0.5px shadow | `A/components/chat-workspace/composer/ConversationComposer.vue:61-65` |
| composer 背景 | `bg-background/70 backdrop-blur-xl`（不透明面板时换 `right-panel-chat-opaque-surface`） | `ConversationComposer.vue:64` |
| 编辑器高度 | `.ProseMirror { min-height: 60px; max-height: 300px; overflow-y: auto }` | `A/components/chat-workspace/composer/tiptap/ComposerEditor.vue:1491-1497` |
| 编辑器（compact） | `min-height: 0; max-height: 240px` | `ComposerEditor.vue:1499-1502` |
| composer 工具行 | `flex items-center justify-between`，发送键 `h-7 w-7` | `ConversationComposer.vue:79-90` |
| 状态栏 | `.chat-status-bar { h-10; px-5; text-xs; justify-between }` | `A/styles/global.css:39-41` |

### 3.3 响应式与折叠条件

| 条件 | 行为 | 来源 |
|---|---|---|
| timeline 容器宽 < 960（容器查询） | 隐藏用户消息 rail 层 | `A/components/chat-workspace/ChatWorkspace.vue:7189-7193` |
| 右侧面板全宽承接聊天 | `max-w-none`、`px-0`、不透明表面、composer 被 Teleport 到面板覆盖层 | `ChatWorkspace.vue:6630,6677-6679,6622-6623`；`:7020-7029` |
| compact overlay composer | 卡片改 `flex-row items-center gap-1 px-2 py-1.5` | `ConversationComposer.vue:62` |
| 面板打开中 | 卡片 `margin-right` 过渡 250ms；拖宽时 `transition-none` | `ChatWorkspace.vue:6485,2014-2018` |
| 侧栏开合中 | 时间线不跟随中间宽度（`layoutTransitionActive` 早退） | `A/components/chat-workspace/timeline/PlainTimeline.vue:746-752` |
| 右面板宽度 | `clamp(280px, var(--right-panel-width), calc(100% - 400px))` | `ChatWorkspace.vue:221-222,1992-1993` |
| 新线程页 | 独立滚动区，内层 `max-w-4xl px-3`，内容 `max-w-2xl gap-8 p-8` | `ChatWorkspace.vue:6555-6567`；`A/components/chat-workspace/NewThread.vue:294-295` |

### 3.4 滚动与虚拟化

| 项 | 事实 | 来源 |
|---|---|---|
| 滚动容器归属 | `.chat-scroll-viewport`（timeline 内），composer / status bar 在它外面 | `PlainTimeline.vue:3454`；`ChatWorkspace.vue:6622-6693` |
| 列表 key | `row.key`（`v-for` 的 `:key`） | `PlainTimeline.vue:3514` |
| 渲染窗口（行数） | 初值上限 48 行、下限 8 行、平均行高 160px、视口行数 +8 | `PlainTimeline.vue:497-509,560-576` |
| 单次扩展批量 | 40 行 / 像素预算 `max(edgeBuffer, 40×160)` | `PlainTimeline.vue:497,601-608` |
| 边缘缓冲 | `max(600, clientHeight × 1.5)` px | `PlainTimeline.vue:595-599` |
| 组件预算 | 展开行按 block 数计入，上限 240 | `PlainTimeline.vue:523,530-554` |
| 占位 | 顶/底 spacer = 未渲染行的估算高度累加 | `PlainTimeline.vue:673-690` |
| 逐行跳过渲染 | `content-visibility: auto` + `contain-intrinsic-size: auto var(--tl-row-estimate-height, 180px)` | `PlainTimeline.vue:3707-3712` |
| 高度来源 | 已测高度缓存 + 未测估算，按 layout state 失效 | `PlainTimeline.vue:410-435` |
| FollowEnd | 状态机 `'following' | 'detached'`；距底 ≤80px 恢复跟随 | `PlainTimeline.vue:79,227,75-76,3127-3131` |
| 回底按钮阈值 | 距底 >300px 才显示 | `PlainTimeline.vue:75,1636` |
| 恢复期 | cover 覆盖 + 滚动条宽度归零 + 关闭过渡动画 | `PlainTimeline.vue:3589-3599,3647-3664,3741-3744` |
| 历史加载指示 | `sticky top-0 h-0`，轨道 `min(560px, 100% - 64px)`、高 2px | `PlainTimeline.vue:3746-3763` |
| 功能开关 | `timelineCv`、`timelineIncremental`（默认 true） | `A/components/chat-workspace/timeline/timeline-render-policy.ts:36,44` |

### 3.5 MyGo 映射

```go
ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
    ui.Column(c).Grow(1).Clip().Radius(radiusXL).Background(k.card).Children(func() {
        header(c, k)                                    // Height(48).PaddingX(k.space(3))
        ui.Column(c).Grow(1).Clip().AlignItems(ui.Stretch).Children(func() {
            ui.List(c, &s.conversation, len(s.rows), func(i int) {
                rowView(c, k, s.rows[i])                // 6 类行，见 §4
            }).Grow(1).ItemKey(func(i int) any { return s.rows[i].Key })
            composer(c, k)                              // 不在 List 内，Grow(0)
            statusBar(c, k)                             // Height(40).PaddingX(k.space(5))
        })
    })
})
```

`ListState` 直接覆盖 dim 的三个关键行为：`Key`（行身份，历史前插时行状态跟随）、`FollowEnd`（流式增长跟到底、用户离开即停）、`FollowAnimation`（跟随时的滑动，尊重 reduce motion）（`mygo ui/list.go:38-70`）。行高按内容测量、未测行按平均估算（`mygo ui/list.go:13-21,466-472`）。

**缺口：**

- **`max-w-4xl` 居中列没有组合语义。** MyGo 有 `MaxWidth`（`mygo ui/api_gen.go:6186`）但没有 auto margin；`MaxWidth` + `AlignSelf(ui.Center)` 需要外层是 Row 方向，否则 `AlignItems` 会把列拉满。且 A 的 4xl 是 rem，随字号缩放；B 必须自己算 `56 × 字号`。
- **`content-visibility: auto` 无对应物。** A 用它跳过屏外已完成行的布局/绘制（`PlainTimeline.vue:3707-3712`），并用 CSS 变量给未挂载行一个估算高度参与滚动坐标。MyGo 没有 CSS containment API，只能完全依赖 `List` 的虚拟化 + 它自己的估算（`mygo ui/list.go:13-21`）；"已挂载行保留真实高度、未挂载行用估算"这条 A 的行为在 B 里由 List 内部实现，**不要在外层再叠一层估算**。
- **阈值参数缺失。** A 的"80px 恢复跟随 / 300px 显示回底按钮"（`PlainTimeline.vue:75-76`）是布局参数；`List` 只暴露 `FollowEnd` 布尔。要么自己 `TrackScroll`（`mygo ui/api_gen.go:8010`）+ 读 `ScrollState` 判断距离，要么把这个差异写进验收基线。
- **滚动条没有尺寸旋钮。** `Scrollbars(bool)` / `KeepScrollbars(bool)`（`mygo ui/api_gen.go:7344,5764-5771`）能表达"悬停显示"和"占位"，但没有 10px 宽 / 24px thumb 的尺寸。
- **`overscroll-contain`、`mask-image` 渐变、`scroll-behavior: auto` 没有直接 API**（`PlainTimeline.vue:3624-3638,3454`）。首片可省，标 WIP。

---

## 4. 消息行（user / assistant / thinking / tool / approval-inline / file-changes）

### 4.1 容器树

```text
timeline-row-shell            w-full px-4（turn 行内 space-y-4）                  PlainTimeline.vue:3515-3517
└─ TimelineRow                row.kind === 'turn'                                 A/.../TimelineRow.vue:286-382
   └─ div.group               block；子项靠各自 mb 堆叠
      ├─ TimelineUserMessage  flex column, items-end, gap-2                       TimelineUserMessage.vue:893-900
      │  └─ 内层              ml-auto, max-w-[85%], flex column, items-end, gap-2   :901
      │     ├─ 图片附件       ImageAttachmentGallery                              :902-906
      │     ├─ chip 架        flex wrap, justify-end, gap-2                        :908-920
      │     ├─ 引用列表       TimelineContextReferenceList                         :922-925
      │     ├─ mark 注释      space-y-1, mb-1；卡片 px-3 py-2 rounded-lg           :928-951
      │     └─ 气泡           rounded-lg, bg-muted, px-4 py-2                      TimelineUserBubble.vue:5
      │        └─ 折叠按钮    `h-7 px-2 text-xs`（内容超高才出现）                   :1048-1061
      ├─ 助手块容器           TimelineAssistantMessage                            TimelineRow.vue:301-333
      │  ├─ 工作摘要行        grid 3 列, min-h-7, my-2, gap-2                      TimelineAssistantMessage.vue:896-915
      │  ├─ blocks 容器       mb-2                                                :917
      │  │  └─ 每块           Thinking / Text / Tool / BlockGroup（均 mb-3）        :939-957；TimelineThinkingItem.vue:96；TimelineToolCallItem.vue:730；TimelineBlockGroup.vue:294
      │  ├─ turn 终止行       mb-3；可展开区 border-l + pl-3 + ml-1.5               :1170-1235
      │  └─ 模型标签          mt-2, gap-2, text-[11px]                             :1291-1293
      └─ 底部区               collapse-grid + space-y-2（文件改动 / 产物 / 动作）    TimelineRow.vue:335-380
```

### 4.2 布局属性

| 行类型 | 属性 | 来源 |
|---|---|---|
| 用户行 | 外层 `flex flex-col items-end gap-2`；内层 `ml-auto max-w-[85%]` | `A/.../TimelineUserMessage.vue:895,901` |
| 用户气泡 | `rounded-lg bg-muted px-4 py-2 text-base leading-relaxed` | `A/.../TimelineUserBubble.vue:5` |
| 用户长文折叠 | `max-height: 300px; overflow: hidden` + 底部 3rem 渐隐 mask | `TimelineUserMessage.vue:1272-1279` |
| 用户行滚动留白 | `scroll-margin-block-start: 72px`（rail 跳转用） | `TimelineUserMessage.vue:1253` |
| 附件 chip 架 | `flex flex-wrap justify-end gap-2` | `TimelineUserMessage.vue:911` |
| mark 注释卡 | `rounded-lg border border-primary/20 bg-primary/5 px-3 py-2 text-sm` | `TimelineUserMessage.vue:935` |
| 助手行根 | `min-w-0`，无背景/无内边距（贴列宽） | `A/.../TimelineAssistantMessage.vue:886-888` |
| 工作摘要行 | `my-2 grid min-h-7 grid-cols-[minmax(0,1fr)_auto_minmax(0,1fr)] items-center gap-2` | `TimelineAssistantMessage.vue:897-899` |
| 摘要按钮 | `h-7 grid-cols-[1fr_14px] gap-1.5 rounded px-2 text-base` | `:904-906` |
| blocks 容器 | `mb-2`；每块 `mb-3` | `:917`；`TimelineThinkingItem.vue:96`、`TimelineToolCallItem.vue:730` |
| thinking 行 | 头 `h-auto w-full justify-start gap-1 p-0 text-base`；正文 `max-h-[240px] overflow-auto border-l-2 pl-3 text-base` | `TimelineThinkingItem.vue:97-99,123-127` |
| tool 行 | 头 `flex min-w-0 items-center gap-1`，按钮 `p-0 gap-1 text-base` | `TimelineToolCallItem.vue:736-739,749-752` |
| tool 展开面板 | `mt-1 max-h-[240px] overflow-auto rounded-md border bg-card` | `TimelineToolCallItem.vue:812` |
| tool 面板小标题 | `sticky top-0 px-3 pt-2 pb-1 text-[11px]` | `:815,822` |
| tool 面板正文 | `px-3 pb-2 font-mono text-xs leading-5` | `:816,826` |
| 工具组头 | `h-auto w-full justify-start gap-1 p-0 text-base` | `TimelineBlockGroup.vue:295-298` |
| 工具组导轨 | `border-l border-border/60 pt-1 pl-3 ml-1.5 overflow-hidden` | `TimelineBlockGroup.vue:334-338` |
| 文件改动头 | `flex items-center justify-between gap-2`，文字 `text-sm` | `A/.../TimelineFileChangesView.vue:53-58` |
| 文件改动列表 | `mt-0.5 overflow-hidden rounded-md border` | `:95-99` |
| 文件改动行 | `gap-3 border-b px-2 py-1.5 text-sm`，状态字母 `w-5 font-mono text-xs` | `:103-119` |
| 折叠机制 | `grid-template-rows: 0fr → 1fr`，`300ms cubic-bezier(0.16,1,0.3,1)` | `A/.../TimelineDetailsRegion.vue:34-49` |

### 4.3 响应式与折叠条件

| 条件 | 行为 | 来源 |
|---|---|---|
| 容器宽 < 960 | 隐藏用户消息 rail（不影响行内布局） | `A/components/chat-workspace/ChatWorkspace.vue:7189-7193` |
| 用户内容超 300px | 折叠 + 显示"展开"按钮；展开后按钮变"收起" | `TimelineUserMessage.vue:1272-1275,1048-1061` |
| turn 未结束 | 底部区 `collapse-grid` 收起（`assistantIsRunning`） | `TimelineRow.vue:335-340` |
| turn 有失败终止 | 工作摘要按钮转 `text-activity-warning` | `TimelineAssistantMessage.vue:907` |
| 工具调用无详情 | 头部按钮 `cursor-default`，无 chevron | `TimelineToolCallItem.vue:740,797-802` |
| 办公模式 | 文件改动 / git 相关块隐藏（`uiMode.visibility.*`） | `TimelineRow.vue:354`；`A/components/sidebar/ThreadItem.vue:317` |

### 4.4 滚动与虚拟化

- 行**自身不滚动**，唯一的滚动是 §3.4 的 viewport；行内的长内容用固定上限 + 内层滚动：thinking `max-h-[240px]`（`TimelineThinkingItem.vue:125`）、tool 面板 `max-h-[240px]`（`TimelineToolCallItem.vue:812`）、subagent 摘要 `max-h-[240px]`（`TimelineAssistantMessage.vue:1017`）。
- 行 key 是 `row.key`（`PlainTimeline.vue:3514`），行内容按 `row.kind` 分派（`TimelineRow.vue:286-418`）。
- 用户行有 rail 跳转锚点 `data-timeline-user-message-id`（`TimelineUserMessage.vue:897`），rail 宿主由 `ChatWorkspace` 提供：`absolute left-[15px] top-1/2 h-0 w-[22px]`（`ChatWorkspace.vue:6548-6551`）。

### 4.5 MyGo 映射

```go
func rowView(c *ui.Context, k tokens, r Row) {
    switch r.Kind {
    case RowUser:
        // 右对齐 85% 气泡：Row().Justify(ui.End) + MaxWidthPercent(85)
        ui.Row(c).Justify(ui.End).Children(func() {
            ui.Column(c).MaxWidthPercent(85).Padding(k.space(4), k.space(4)).Radius(radiusMD).
                Background(k.muted).Children(func() { ui.Text(c, r.Text) })
        })
    case RowAssistant:
        ui.Column(c).Children(func() { blockViews(c, k, r.Blocks) })
    case RowThinking:
        // 折叠头 + 240 DIP 上限的滚动正文
        ui.Column(c).Children(func() {
            ui.Row(c).Gap(k.gap).Children(/* lightbulb, label, chevron */)
            if r.Expanded {
                ui.Scroll(c).MaxHeight(k.space(60)).PaddingStart(k.space(3)).
                    BorderStartWidth(k.hairline).Children(/* thinking text */)
            }
        })
    case RowTool:
        ui.Column(c).Children(func() {
            ui.Row(c).Gap(k.gap).Children(/* icon, summary, chevron */)
        })
    }
}
```

**缺口：**

- **`ml-auto max-w-[85%]` 语义可表达，但需要换算。** `MaxWidthPercent(85)` 是相对父宽（`mygo ui/api_gen.go:6196`），与 A 一致；但 A 的 300px 折叠上限（`TimelineUserMessage.vue:1273`）要写成 `MaxHeight(300)`，**不随字号缩放**，而 A 的 300px 是 px 字面量，恰好一致。
- **渐隐 mask 无 API。** `mask-image: linear-gradient(...)`（`TimelineUserMessage.vue:1278-1279`）在 MyGo 里只有 `Masks(layers...)` / `MaskLinearGradient`（`mygo ui/api_gen.go:6128,6080`）。首片可以只做截断，把"没有渐隐"写进视觉差异清单。
- **折叠动画的 `0fr → 1fr` 网格技巧没有对应物。** MyGo 的动画是"对某个数值属性做 `Animate`/`Keyframes`"（`mygo ui/api_gen.go` 的 `Animate`, `Keyframes`），没有 `grid-template-rows` 这种结构性过渡。P0 用显隐（`Hidden`）或对 `Height` 做动画，并接受与 A 不同的缓动。
- **9 种工具面板 + 10 种审批 renderer 需要逐个映射。** A 的工具面板按 `isCommand/isSearch/isFile/isWeb` 分支（`TimelineToolCallItem.vue:811-880`），渲染器注册在 `A/components/chat-workspace/timeline/tool-renderers/`；B 现在只有一种工具卡（[ui-parity.md](ui-parity.md) §3.2）。P0 只做 command / file / search / generic 四种，其余 WIP。
- **任意字号（`text-[13px]`、`text-[11px]`、`text-[10px]`）没有档位。** 同 §2.5。
- **无障碍可以比 A 更强，不要照抄 A 的空白。** A 只在部分节点有 `data-testid`，行级 role/label 不完整；MyGo 有 `Role`/`Label`/`Access*`（`B/internal/ui/capability.go`、`wip.go:35-40` 的既有用法），turn 行应标 `RoleGroup` + 用户/助手标签。

---

## 5. Approval 面板

### 5.1 容器树

```text
底部栈 → mx-auto max-w-4xl                                                ChatWorkspace.vue:6628-6631
└─ InlinePermissionPrompt（优先于 InputArea，二者互斥）                      ChatWorkspace.vue:6665-6688
   └─ PermissionPrompt 根   shrink-0, bg-background, px-3, pb-3             A/.../PermissionPrompt.vue:214-221
      └─ 卡片              rounded-lg, border, bg-card, p-4                :223-226
         ├─ 标题           text-base font-semibold                        :227
         ├─ 成员审批块     mt-3, rounded-md, bg-muted/30, px-3 py-2        :229-258
         ├─ 输入面板       ApprovalInputPanel class="mt-3"                 :260-267
         │  └─ 具体面板    max-h-[240px], overflow-auto, rounded-md,border :CommandApprovalPanel.vue:17-32
         │                 ├─ sticky 小标题 px-3 pt-2 pb-1 text-[11px]     :23-25
         │                 └─ 正文 px-3 pb-2 font-mono text-xs leading-5    :26-31
         ├─ reviewer 说明  mt-3, rounded-md, bg-muted/50, px-3 py-2         :269-284
         ├─ 错误           mt-3 text-sm text-destructive                  :286
         └─ 动作行         mt-3, flex, items-center, justify-end, gap-2     :288-298
```

### 5.2 布局属性

| 属性 | 值 | 来源 |
|---|---|---|
| 面板占位 | composer 的槽位（不是浮层、不是 dialog）；`shrink-0` | `A/components/chat-workspace/ChatWorkspace.vue:6665-6687`；`PermissionPrompt.vue:220` |
| 根内边距 | `px-3 pb-3`（上边距由底部栈的活动条 `mb-2` 提供） | `PermissionPrompt.vue:220`；`ChatWorkspace.vue:6637` |
| 卡片 | `rounded-lg border border-border bg-card p-4` | `PermissionPrompt.vue:225` |
| 卡片内距 | 标题下内容统一 `mt-3` | `PermissionPrompt.vue:233,260,269,286,288` |
| 输入面板上限 | `max-h-[240px] overflow-auto rounded-md border bg-card` | `CommandApprovalPanel.vue:19-21`；`GenericApprovalPanel.vue:13-15` |
| 输入面板 sticky 头 | `sticky top-0 z-10 bg-card px-3 pt-2 pb-1 text-[11px]` | `CommandApprovalPanel.vue:23-25` |
| 动作行 | `mt-3 flex items-center justify-end gap-2` | `PermissionPrompt.vue:288` |
| 沙箱变体动作行 | `mt-3 flex flex-wrap items-center justify-end gap-2` | `PermissionPrompt.vue:312` |
| 子代理模型变体 | 动作行带 `border-t pt-3` | `PermissionPrompt.vue:356` |
| 按钮 | `size="sm"`；主选项 `default`，其余 `outline` | `PermissionPrompt.vue:289-297,176-182` |
| 选项顺序 | reject → allow_once → 其余 | `PermissionPrompt.vue:160-168` |
| 成员审批时可见选项 | 只留 `allow_once` / `reject_once` | `PermissionPrompt.vue:170-174` |
| 键盘 | Enter / `y` = 允许一次；Esc / `n` = 拒绝；输入控件聚焦时不拦 | `PermissionPrompt.vue:187-204` |
| 变体分支 | tool call / sandbox 升级 / 子代理模型 | `PermissionPrompt.vue:223,302,325` |

### 5.3 响应式与折叠条件

| 条件 | 行为 | 来源 |
|---|---|---|
| 有权限交互 | 审批面板**替换** composer，输入框整个消失 | `ChatWorkspace.vue:6665-6688` |
| 全宽面板聊天 | 外层 `px-0`、不透明表面 | `ChatWorkspace.vue:6677-6679` |
| 面板不自己定宽 | 吃 `max-w-4xl` 列宽；窄窗时随列收窄 | `ChatWorkspace.vue:6628-6631` |
| 有 elicit 追问 | 换成 `InlineElicitationPrompt`，与权限面板同级互斥 | `ChatWorkspace.vue:6670-6674` |

### 5.4 滚动与虚拟化

- 审批面板本身不滚动；只在内层的 tool 输入块内滚动（`max-h-[240px] overflow-auto`，`CommandApprovalPanel.vue:21`；`GenericApprovalPanel.vue:15`）。
- 面板随 composer 槽位一起位于**非滚动区**，长会话滚动时审批面板始终可见（`ChatWorkspace.vue:6622-6693`）。

### 5.5 MyGo 映射

```go
ui.Column(c).Padding(k.space(3)).PaddingY(0).Children(func() {
    ui.Column(c).Padding(k.space(4)).Radius(radiusMD).Background(k.card).
        Border(k.hairline, k.border).Children(func() {
            ui.Text(c, title).Bold()
            approvalInput(c, k, req)                                  // §5.1 的输入面板
            ui.Row(c).Justify(ui.End).Gap(k.gap).Children(func() {
                for _, opt := range orderedOptions {                  // reject → allow_once → 其余
                    ui.Button(c, opt.Label).OnClick(...)
                }
            })
        })
})
```

**缺口：**

- **`max-h-[240px]` 的滚动块要给尺寸。** `ui.Scroll` 要求给尺寸或 `Grow`（`mygo ui/api_gen.go:2123-2124`），所以写 `ui.Scroll(c).MaxHeight(k.space(60))`，`60 × 4 = 240`（`mygo ui/theme.go:86-92`）。
- **sticky 小标题要验证。** `Sticky()` 存在（`mygo ui/api_gen.go:7705`），但在 `Scroll` 内的行为需要 spike，不能假定等同 CSS `position: sticky`（`CommandApprovalPanel.vue:23`）。
- **"全局键盘但忽略输入控件"要自己实现。** A 的判断是：事件目标不是 body、不在面板内、或落在 button/input/textarea/contenteditable 上就不处理（`PermissionPrompt.vue:187-193`）。MyGo 需要自己组合 `Focused()` / `FocusWithin()` / `KeyShortcuts`。
- **产物差异要先声明。** A 有 10 种 approval renderer（`A/components/chat-workspace/interactions/ApprovalInputPanel.vue:25-34`），B 目前只有"两个按钮"（[ui-parity.md](ui-parity.md) §3.2）。P0 只做 command / file / generic，其余标 WIP，不要用假面板顶上（`B/docs/workflow.md` 的 WIP 规则）。

---

## 6. P0 实现顺序与验收方式

顺序按"改一行会影响多少下游"排：外壳定了，子区域的坐标才稳定；token 没定，每个区域都会写死数值。

| # | P0 区域 | 实现顺序 | 前置 | 验收方式（headless 断言） | 验收方式（截图对照） |
|---|---|---|---|---|---|
| 1 | 应用外壳（窗口行、侧栏让位、卡片外框） | 1 | 无 | `Tester.NewTester(view, 1440, 900)` 后 `Tester.Find("Sessions")` 的 `Rect.X ≈ 0`、卡片左缘 ≈ 256；`Tester.SetSize(700, 900)` 后侧栏不再占位（抽屉关时 `Find` 不到会话行） | 与 A 的 1440×900 header 截图并排；卡片 4.5px 缝隙、0.5px 描边可见 |
| 2 | Sidebar（header / 全局入口 / 滚动列表 / footer） | 2 | 1 | `AccessByRole(ui.RoleTreeItem, ...)` 计数 = 行数；`Find("← 折叠")` 在展开时存在；footer 贴底：footer 的 `Rect.Y + H ≈ 窗口高`；`Tester.Scroll` 后 footer 不动 | 侧栏 256 / 200 / 360 三档截图；含置顶段、含"显示更多" |
| 3 | Timeline 行模型（6 类行 + List/FollowEnd） | 3 | 1 | 追加一行后 `Tester.Find(最后一行)` 仍在视口内（FollowEnd）；`Tester.Scroll` 离底后追加不再拉回；thinking 头文本 = "Thought for Ns" 形态；tool 行点击后展开面板且高度 ≤ 240 | 固定 fixture（user/assistant/thinking/tool/approval/file-changes 各 1）light/dark 各一张 |
| 4 | Composer（卡片 + 自动高度 + 状态栏） | 4 | 3 | 空态编辑器高 = 60；输入 20 行后高 = 300 且内部滚动；`Tester.Type` 后发送键可点；状态栏高 = 40 | composer 空态 / 3 行 / 上限三张；与 A 的 `chat-status-bar` 对齐 |
| 5 | Approval 面板（command/file/generic） | 5 | 4 | 有交互时 composer 文本不存在（`HasText(placeholder) == false`）；按钮顺序 = reject 在首位；`Tester.Key` Enter 触发 allow_once；Esc 触发 reject | 三种面板各一张；240px 上限与 sticky 标题 |
| 6 | 响应式收口（compact 抽屉、<960 rail、auto-collapse） | 6 | 1–5 | `SetSize(700,900)` 后 `AccessByRole(ui.RoleDialog, ...)` 存在、按 Esc 后消失；宽 1000 + 右面板开时侧栏自动收起；`SetSize` 回 1440 后（带 40px 迟滞）恢复 | 700 / 1000 / 1440 三档；抽屉态与展开态各一张 |

**验收的硬约束（不要跳过）：**

1. 每一行的 headless 断言都必须是几何或文本事实（`Tester.Find` / `Tester.Texts` / `Tester.Access`），不要用"渲染没崩"当通过（对齐 `B/docs/acceptance.md` 的证据等级：L1 只是测试级）。
2. 截图对照的 A 侧产出用现成脚本：`packages/desktop/scripts/test-electron-chat-windows.mjs:238-241`（`page.screenshot({ animations: 'disabled' })`）与 `packages/desktop/scripts/test-electron-chat-workspace-structure-e2e.mjs`（用 `getBoundingClientRect` 做结构断言，`:721,745` 是同类写法）。B 侧用 `Tester.Image()` + `MYGO_UI_PNG`。
3. 像素对照先人工评审，不设自动阈值；两边渲染器不同，自动阈值会误报（同 [ui-parity.md](ui-parity.md) §5 第 5 步）。

## 7. 事实边界

- 本文所有数值都来自源码阅读（L0），**没有在真实窗口里量过**。
- 不在范围内：right panel、context panel、settings、terminal、market/plugins/skills、pet、browser。
- 已发现的两处 A 侧内部不一致，实现时不要照抄：
  1. flat 侧栏行用 `h-9`（rem，随字号缩放），虚拟化按 36px 常量算（`A/components/sidebar/ThreadList.vue:109` vs `:248`）。
  2. 侧栏 header 与 chat header 都用 `--titlebar-height`（48px 固定），但 compact 抽屉的顶边用 `--win-titlebar-height`（2.5rem，随字号缩放）（`A/components/ui/sidebar/Sidebar.vue:164` vs `A/styles/global.css:168,174`）——非 mac 上字号非 14px 时两者会差几像素。
