# UI 1:1 对齐审计（dim-electron renderer ↔ dimagent-mygo internal/ui）

审计日期：2026-11-09。只读审计，未改动任何源码。

- A = `/Users/eric8810/Code/dim-electron/packages/desktop/packages/renderer/src`
- B = `/Users/eric8810/Code/dimagent-mygo/internal/ui`

## 结论

**当前不是 1:1，差距在第一层（design token 语义层）就已经成立。**

| 维度 | A | B | 判定 |
|---|---|---|---|
| 语义 token 数量 | 156 个 CSS 变量（`styles/global.css` 全文件），其中 118 个非 `--color-*` 语义变量 | `tokens struct` 29 个字段，其中颜色 7 个（[tokens.go:21-51](../internal/ui/tokens.go#L21-L51)） | 不成比例 |
| 颜色语义层 | 三层：原始色 `--skin-tone-*`(20) → 语义 `--background/--card/...`(45) → skin 覆盖层（带 alpha） | 一层：直接从 `ui.Theme` 取 7 个色 | B 缺 2 层 |
| 组件目录 | 19 个顶层组件目录 + 2 个图标组件，601 个 `.vue`、93,133 行 | 7 个 Go 文件、1,035 行（非测试） | B 有部分对应物的只有 3 个目录（sidebar、chat-workspace、markdown），其余 16 个无对应 |
| 主题/皮肤 | 3 主题偏好 × 6 chrome 主题 × 任意 skin.json 皮肤 × 可配代码主题 | mygo Light/Dark，跟随系统，shell 从不设置 | B 缺整套外观层 |
| 视觉证据 | 31 个 Playwright Electron e2e 脚本产出真实窗口截图 | 只有 headless `ui.NewTester`，无一处 golden 图 | B 无像素证据 |

一句话：B 是"能跑通会话闭环的静态壳"，A 是"带语义 token 体系、外观系统和 19 个组件域的完整产品 UI"。B 的 token 语义层目前只有 A 的约 1/6，且缺少 A 用来组织视觉层级的 surface/elevation 概念。

## 1. 审计方法与证据边界

- 只读源码，不做像素对比：本次审计没有运行 A 的 Electron 构建，也没有为 B 出图。本文的结论全部落在可 grep 的源码事实（文件 + 行号）上；第 5 步给出补像素证据的具体做法。
- "1:1" 这里指**聊天壳的视觉与交互对齐**，不是代码对齐。两侧技术栈不同（Vue 3 + CSS ↔ Go + mygo retained-mode），不可能也不应该共享实现，只能共享**语义层 + 组件契约**。
- 不评价业务能力差异（session/approval/timeline 真相都在 App Server，见 `AGENTS.md` 第 2 节），只评价"同一个会话界面看起来/用起来是否一致"。

## 2. Token 对照表

### 2.1 A 的 token 架构（三层）

1. **原始色 `--skin-tone-*`**（20 个）：`styles/global.css:867-888` 浅色、`889-911` 深色，按 `[data-skin-surface-tone]` 切换。
2. **语义 token**（45 个：核心 28 + tooltip 7 + sidebar 10）：`styles/global.css:183-256` 浅色，`272-330` 深色；由 `@theme inline`（`styles/global.css:44-95`）映射成 Tailwind 颜色工具类。
3. **skin / chrome 覆盖层**：`.skin-active.skin-host` 把同一批语义 token 重新以 alpha 绑定到 `--skin-tone-*`（`styles/global.css:923-950`、`1060`）；chrome 主题见 `styles/retro-themes.css`（955 行，`data-chrome-theme` 共 6 个值：`default`、`macos-bigsur`、`osx-tiger`、`windows-98`、`windows-vista`、`windows-xp`）。

### 2.2 语义 token ↔ B tokens 字段

| A 变量（global.css 行号 / 浅色值） | B `tokens` 字段 | 判定 |
|---|---|---|
| `--background` (183, `oklch(1 0 0)`) | 无字段。窗口底色由 mygo `Theme.Background` 提供，shell 未设置（`cmd/dimagent/main.go:958-963` 只传 Title/Width/Height/Content） | **缺** |
| `--foreground` (184, `oklch(0.145 0 0)`) | 无字段。文字靠 `ui.Text` 默认 `Theme.Text` | **缺**（不可寻址，无法按区域覆盖） |
| `--card` / `--card-foreground` (185-186) | 无。卡片用 `k.wash(k.border)` = border@12%（[tokens.go:99](../internal/ui/tokens.go#L99)、[tool.go:109](../internal/ui/tool.go#L109)、[capability.go:83](../internal/ui/capability.go#L83)） | **缺**（用 12% tint 冒充 elevated surface） |
| `--popover` / `--popover-foreground` (187-188) | 无。WIP 卡与工具卡同为 border tint | **缺** |
| `--primary` / `--primary-foreground` (205-206) | `accent = theme.Accent`；`theme.AccentText` 从未读取（[tokens.go:87](../internal/ui/tokens.go#L87)） | **部分**：缺 on-primary 前景配对 |
| `--secondary` / `--secondary-foreground` (211-212) | 无 | **缺** |
| `--muted` / `--muted-foreground` (213-214) | `muted = theme.TextMuted`，只有"次级前景"，没有 muted 填充面 | **名实不符**：A 的 muted 是面，B 的是字 |
| `--accent` / `--accent-foreground` (215-216) | `accent = theme.Accent`（品牌色） | **冲突**：同名不同义。A 的 accent 是中性 hover 填充，品牌色叫 `--primary` |
| `--destructive` / `--destructive-foreground` (217-218) | `danger = theme.Danger`；无 danger-foreground | **部分** |
| `--border` (219) | `border = theme.Border` | **有** |
| `--meter-track` (224) | 无 | **缺** |
| `--card-frame-outer` (226) | 无 | **缺** |
| `--input` (227) | 无（输入框由 mygo `TextArea` 自绘 `Theme.Surface/Border`） | **缺** |
| `--ring` (228) | 无（`theme.Focus` 只在 mygo widget 内部生效） | **缺** |
| `--chart-1..5` (229-233) | 无 | **缺** |
| `--mark` / `--mark-soft` (209-210) | 无 | **缺** |
| `--tooltip` / `-foreground` / `-muted-foreground` / `-border` / `-chip` / `-backdrop` / `-shadow` (192-204；深色 279-291) | 无。mygo 有 `Theme.Inverse/InverseText`，shell 未暴露 | **缺**（7 个 token + `backdrop-filter` + 5 层阴影） |
| `--sidebar` / `-overlay` / `-foreground` / `-primary` / `-primary-foreground` / `-accent` / `-accent-foreground` / `-hover` / `-border` / `-ring` (234-243；深色 316-326) | 无。侧栏底色直接用 `k.surface`（[shell.go:216](../internal/ui/shell.go#L216)），图标/悬停交给 `ui.Sidebar` 自带主题 | **缺**（10 个变量整套） |
| `--radius` + `-sm/-md/-lg/-xl` (44-47, 244) | `cardRadius = theme.Radius`（6，mygo `theme.go:188`），单一档位 | **缺**（A：6/8/10/14 四级） |
| `--dimcode-ui-font-family` (157-161)、`--dimcode-code-font-family` (162-165) | 无。`theme.Font` 未设置（空 = 系统字体） | **缺** |
| `--dimcode-ui-font-size` 14px / `--dimcode-code-font-size` 13px (166-167) | `bodySize = theme.FontSize`（darwin 13 / 其他 14，mygo `theme.go:158-164`） | **部分**：无代码字号概念 |
| `--titlebar-height` 48px (174)、`--win-titlebar-height` 2.5rem | 无（原生系统标题栏） | **缺** |
| `--layout-transition-ms` 250ms (259)、`--ease-*`、`--animate-*` (88-114) | 无 | **缺**（B 无动效 token） |
| `--z-sidebar-base` … `--z-sidebar-floating` (250-254) | 不适用（retained-mode 单一绘制列表） | 合理差异 |
| `--activity-info/-violet/-success/-warning/-shimmer-highlight` (336-340) | 无 | **缺** |
| `--markdown-table-*`/`-link`/`-inline-code-bg`/`-code-block-*`/`-hr`/`-checkbox-*` (18 个，342-359；深色 369-385) | 无。`markdown.Options{Theme: nil}`（[shell.go:205](../internal/ui/shell.go#L205)、[shell.go:449](../internal/ui/shell.go#L449)），走 mygo 默认 Markdown 主题 | **缺**（内容层完全不可主题化） |

### 2.3 B 缺失的语义层（按层计）

1. **Surface 层级梯度**：background → card → popover 三级明度关系（A 深色下 `0.16 → 0.25 → 0.25`，`global.css:272-277`），B 只有一个 `surface` 加 12% border wash，层级靠 tint 近似。
2. **前景配对**：所有 `*-foreground`（primary/secondary/accent/destructive/card/popover/sidebar-primary/sidebar-accent）在 B 中缺失，导致"按钮上的字"只能靠 widget 默认色。
3. **Sidebar 语义集**：10 个变量全缺，侧栏与主区共用一个 `surface`。
4. **Tooltip 玻璃层**：7 个 token + blur/saturate + 多层阴影。
5. **Markdown 内容层**：18 个 token（表格、代码块、行内代码、链接、复选框、hr）。
6. **状态/活动色**：info/violet/success/warning/shimmer 5 个。
7. **排版层**：双字体族（Inter + JetBrains Mono，`global.css:10-18` 自托管）、离散字号档位（12/13/14/16/18，`appearance/index.ts:11-13`）、代码字号 13。
8. **几何/动效层**：radius 四级、间距刻度、过渡时长、缓动、面板进出动画。
9. **状态化表面**：`non-vibrant-window`（`global.css:425-431`）、`prefers-reduced-transparency`（`433-460`）、skin 强制 tone（`867-911`）三套降级/覆盖路径。

### 2.4 度量差异（会直接影响观感）

| 项 | A | B |
|---|---|---|
| 侧栏宽度 | 默认 256、最小 200、最大 360、可拖拽（`components/ui/sidebar/utils.ts:4-6`） | 固定 `Space(55)` = 220 DIP，不可拖拽（[tokens.go:72](../internal/ui/tokens.go#L72)） |
| 状态栏 | `.chat-status-bar` 高 40px、左右 20px、12px 字（`styles/global.css:39-41`）；另有 `h-10`=2.5rem@14px=35 DIP 的读数（见 [ui-layout-spec.md](ui-layout-spec.md) §7，待窗口量测确认） | 无独立状态栏；连接/运行状态塞进 header（[shell.go:295-322](../internal/ui/shell.go#L295-L322)） |
| 标题栏 | 固定 48px，自绘窗口控件（`global.css:174`、`components/app/AppTitlebar.vue`） | 原生系统标题栏 |
| 卡片圆角 | 10 / 8 / 6 / 14 px 四级（`global.css:44-47`、`244`） | 6 px 单档（[tokens.go:75](../internal/ui/tokens.go#L75)） |
| 基础字号 | 14px UI、13px 代码、5 档可选、⌘+/⌘- 缩放（`appearance/index.ts:11-16`、`appearance/zoom-shortcuts.ts`） | 系统字号 13(darwin)/14，四档相对比例 1.15/0.9/0.8（[tokens.go:8-12](../internal/ui/tokens.go#L8-L12)、[79-82](../internal/ui/tokens.go#L79-L82)） |
| 间距单位 | Tailwind 4px 刻度 + 具名 utility | `theme.Space(n)` = n×4 DIP，具名几何字段 17 个（[tokens.go:62-77](../internal/ui/tokens.go#L62-L77)） |
| 密度 | 可切换（字号档位 + 侧栏宽度 + 面板宽度持久化，`ChatWorkspace.vue:217-220`） | 无密度概念 |

## 3. 组件清单差异

A 顶层：19 个组件目录 + 2 个独立图标组件（`components/`）。B 的全部 UI 是 7 个 Go 文件。

### 3.1 逐目录对照

| A 目录（文件数） | B 对应物 | 级别 |
|---|---|---|
| `sidebar/`（22）`AppSidebar.vue`、`ThreadList.vue`、`SidebarHeader.vue`、`PinnedThreads.vue`、`ProjectItem.vue`、`ChatSearchDialog.vue`、`RenameThreadDialog.vue`、`ArchiveThreadDialog.vue`、`ThreadContextMenu.vue`、`useSidebarVirtualRows.ts`、`useProjectDragReorder.ts` | `internal/ui/sidebar.go`：header、全局入口、可选分组线程列表、选中/悬停、200–360 拖拽调宽、双击复位；搜索/置顶/重命名/归档/右键菜单保持 visible-disabled WIP | **部分完成**（`2701701`）：核心列表与量纲已对齐；对话框类仍是 WIP |
| `chat-workspace/`（185；timeline 65、composer 36、interactions 20、model-selector 15、git 7、attachments 6、workflow 3） | `messageRow`（[shell.go:444-459](../internal/ui/shell.go#L444-L459)）+ `composer`（[shell.go:486-500](../internal/ui/shell.go#L486-L500)） | **P0**：见 3.2 |
| `markdown/`（20）`FileChip.vue`、`MarkdownImageNode.vue`、`MarkdownHeadingNode.vue`、`MarkdownInlineCodeNode.vue`、`ImageLightbox.vue`、`LinkContextMenu.vue`、`useMarkdownRender.ts` | 单个 `markdown.View`（[shell.go:449](../internal/ui/shell.go#L449)），默认主题 | **P0**：文件 chip、标题锚点、图片灯箱、链接菜单、代码块运行时全缺 |
| `settings/`（41）+ `views/SettingsView.vue`、`SettingsNav.vue`、19 个分区（`settings/sections.ts:24-43`） | 无 | **P1**：设置面整体缺失（appearance 见第 5 步计划） |
| `context-panel/`（16）文件树、`DiffView.vue`、`FilePreviewView.vue`、`OfficeFilePreview.vue`、`PptxSurfacePreview.vue`、`XlsxPreview.vue` | 无 | **P1** |
| `right-panel/`（1）+ `ChatWorkspace.vue` 承载（`ChatWorkspace.vue:1973-2009`） | 无 | **P1** |
| `app/`（6）`AppTitlebar.vue`、`WindowControls.vue`、`BrandMark.vue`、`StartupOverlay.vue` | 无 | **P2** |
| `scheduled/`（10）+ `ScheduledView.vue` | 无 | **P2** |
| `ui/`（406，shadcn 基础件） | mygo `ui` 提供 button/input/list/markdown 等；缺 dialog/popover/menu/command/toast/tooltip 的 shell 接入 | **P1**（按需接） |
| `market/`（7）、`plugins/`（28）、`skills/`（7）、`provider/`（1） | 无 | **P2** |
| `browser-panel/`（4）、`ui-extensions/`（4）、`mcp-apps/`、`pet/`（2）、`onboarding/`（4）、`account/`（1）、`selection/`（1） | 无 | **P2** |

### 3.2 chat-workspace 内部差距（P0 明细）

| A 组件 | 现状 | B 现状 | 级别 |
|---|---|---|---|
| `ChatWorkspace.vue`（7,329 行）+ `ChatHeaderFrame.vue` + `ChatHeader.vue`（603） | 卡片式聊天框：`chat-workspace-frame`/`chat-workspace-card`（`global.css:20-37`），右侧面板按 margin 让位（`ChatWorkspace.vue:2003-2009`） | 普通 Column，无卡片/描边/圆角框架 | **P0** |
| `timeline/TimelineRow.vue`（442）、`TimelineAssistantMessage.vue`、`TimelineThinkingItem.vue`、`TimelineToolCallItem.vue`、`TimelineFileChanges.vue`、`TimelineArtifacts.vue`、`TimelineRetryNotice.vue`、`TimelineErrorBoundary.vue`、`TimelineCompressionDivider.vue`、`tool-renderers/` | 行类型化、工具卡可展开、thinking 折叠、文件改动、重试、错误边界、压缩分割线 | `internal/ui/timeline.go`（`760413e`）：turn 内 block 分派、thinking 默认折叠（`Thought for Ns`）、tool 卡展开（≤240 DIP 面板）、file changes 紧凑行、独立 error/retry 行 | **部分完成**：`tool-renderers/` 分类面板、压缩分割线、artifacts、用户长文折叠仍是 WIP |
| `composer/ConversationComposer.vue`、`InputArea.vue`、`tiptap/`、`activity/`、`FollowUpMessages.vue`、`ComposerSkillsSelector.vue`、`AgentModeSelector.vue`、`model-selector/`（15） | 富输入 + 附件 + skills + slash 命令 + 模型/模式选择 + 活动条 | `composer.go`（`f157797`）：自适应多行（1–64 行，60–300 DIP）、Enter 发送 / Shift+Enter 换行、提示行、Attach/Mode/Model 的 visible-disabled WIP、Send；缺富文本、skills、slash 命令、活动条 | **部分完成** |
| `interactions/PermissionPrompt.vue`、`ApprovalInputPanel.vue`、`ElicitationPrompt.vue` | 审批/权限/追问三类面板 | `approval_panel.go`（`a2cdb9b`）+ composer 槽位替换（`ffa54c7`）：待审批卡片、call/reason/source、Approve/Deny；缺权限范围、追问、输入块与变体 | **部分完成** |
| `StatusBar.vue`（367）`ContextUsageMeter.vue`、`ChatHeader` 的模型/分支入口 | 底部状态栏：git 分支、上下文用量、账户用量 | 无 | **P1** |
| `SessionTasksPanel.vue`、`BackgroundTaskTerminal.vue`、`TerminalDrawer.vue`、`ProcessDetail.vue` | 后台任务与终端面板 | 无 | **P2** |
| `SideChatPanel.vue`、`SubagentChatPanel.vue` | 侧聊/子代理 | 无 | **P1** |
| `workflow/`、`WakeupDetail.vue` | 工作流/wakeup 详情 | 无 | **P2** |

### 3.3 B 独有、A 没有的

- 侧栏底部"Unavailable capabilities"卡片组（[capability.go:64-91](../internal/ui/capability.go#L64-L91)），把 WIP 能力当产品状态直接列出来。A 的 WIP 只做入口禁用（`packages/shared/src/wip.ts`、`settings/wip-capability-copy.ts`），不列清单。这不是差距，是 B 更强的可观测性，建议保留。
- `wipAction`（[wip.go:35-40](../internal/ui/wip.go#L35-L40)）把 disabled + 原因文本 + `DescribedBy` 绑在一起，A 侧是 Tooltip 文案，语义等价。

## 4. 视觉/交互证据差距

| 证据类型 | A | B |
|---|---|---|
| 真实窗口截图 | 31 个 `scripts/test-electron-*.mjs` e2e，Playwright `page.screenshot()` 落盘（如 `scripts/test-electron-appshots-e2e.mjs:172`、`scripts/test-electron-chat-windows.mjs:241`） | 无。`ui.NewWindow` 只传 Title/Width/Height/Content（`cmd/dimagent/main.go:958-963`），仓库内无截图产物 |
| Headless 像素 | 无（Electron + jsdom 不做像素） | 有 `ui.Tester.Image()` 与 `MYGO_UI_PNG` 导出 PNG（mygo `ui/ui_test.go:17-31`），但**仓库里没有一条用例使用它** |
| 像素回归 | 无 golden；仅 appshots e2e 里用 `scripts/e2e/compare-png.swift` 比较"拍照前后同一个区域" | 无 |
| 关键路径测试 | 230 个 renderer 测试文件 | 12 个测试文件，全是文本/无障碍断言（`shell_test.go:31` 起） |
| 深色模式覆盖 | `styles/global.css:262-330` 全套深色值 + 6 个 chrome 主题各自的深色块 | 无测试、无强制深色用例（mygo 支持 `Tester.SetDark`，mygo `ui/headless.go:483`，B 未用） |
| 主题/皮肤数量 | 3 主题偏好 × 6 chrome（`packages/shared/src/theme.ts:5-12`、`styles/retro-themes.css`）× skin.json 任意皮肤（1 个 window region + 4 个 surface + variants，`packages/shared/src/constants/skin.ts:19,28`、`skin-schema.ts`）× 代码主题（Vitesse 默认，`appearance/code-theme-bg.ts:21-33`） | 2（mygo Light/Dark，跟随系统，`runtime.go:615-630`）；shell 从不 `SetTheme`，也没有外观设置入口 |
| 字体 | 自托管 Inter + JetBrains Mono（`global.css:10-18`），5 档 UI 字号、代码字号、可自定义字体族（`appearance/index.ts:11-22`） | 系统字体，单档字号 |
| 密度 | 侧栏宽度/右侧面板宽度持久化（`ChatWorkspace.vue:217-220`）、`--layout-transition-ms` 统一过渡、zoom 快捷键 | 全部固定常量 |
| i18n | 11 个 locale（`renderer/src/i18n/config.ts`） | 全英文硬编码（`shell.go`、`wip.go`、`capability.go` 全文） |
| 无障碍 | 部分组件有 aria；无统一契约 | 明显更强：Role/Status/Busy/Description/DescribedBy/AccessHidden 成体系（`shell.go:327-348`、`tool.go:129-139`、`wip.go:35-40`） |

结论：A 的"证据"是截图产物（人看），B 的"证据"是无障碍树断言（机器看）。两边都没有像素回归，所以**任何"已 1:1"的说法目前都没有证据支撑**。

## 5. 最小可执行 5 步对齐计划

顺序不能换：token 语义层不做，后面每一步都会在组件里写死颜色，返工成本最高。

### 第 1 步：补 token 语义层（P0，1 个 PR）

- 文件：`internal/ui/tokens.go`。
- 做法：把 `ui.Theme` 已经有但没用上的字段接出来（mygo `ui/theme.go:10-83`：`Background`、`Text`、`SurfaceHover`、`SurfacePressed`、`AccentText`、`AccentHover`、`AccentPressed`、`Selection`、`Focus`、`Inverse`、`InverseText`、`Scrollbar`），在 `newTokens` 里派生 A 的语义名：
  - `background/foreground/card/cardForeground/popover/popoverForeground`（elevation 用 `Color.Mix` 从 background 往 foreground 走，替代现在 `wash(border)` 的 12% tint）；
  - `primary/onPrimary`（现 `accent` 改名，`AccentText` 补上）、`secondary/secondaryForeground`、`muted/mutedForeground`（拆分"面"和"字"）、`accent/accentForeground`（**改回中性 hover 语义**）、`danger/onDanger`；
  - `input`、`ring`（取 `theme.Focus`）、`meterTrack`、`mark` 可选；
  - `sidebar` 10 项 + `tooltip` 7 项（mygo `Inverse/InverseText` 是它的底座）。
- 验收：新增 `internal/ui/tokens_test.go`，对浅/深两套断言"A 的同名 token 落在同一明度区间"（阈值建议 ΔL ≤ 0.03），并断言 `--accent` 与 `--primary` 不再同值。
- 风险：`accent` 语义变更是 breaking，需要同时改 `tool.go:129-139`、`tokens.go:102-148` 的用例。

### 第 2 步：量纲层（P0，同一 PR 或紧随）

- 文件：`internal/ui/tokens.go`。
- 做法：把 A 的量纲搬成结构字段，消灭现在散落的魔数：
  - 侧栏 `Space(55)`=220 → 默认 256、min 200、max 360（对齐 `components/ui/sidebar/utils.ts:4-6`），并加拖拽；
  - 字体：`fontFamily`（Inter 栈）、`monoFamily`（JetBrains Mono 栈）、`uiFontSize` 5 档 `[12,13,14,16,18]`、`codeFontSize` 13（对齐 `appearance/index.ts:11-16`）；
  - 圆角四级（6/8/10/14）；状态栏高 40、标题栏 48；`layoutTransitionMs` 250。
- 验收：`tokens_test.go` 断言数值表；侧栏拖拽后宽度写入配置（沿用现有 config 通道，不新建存储）。
- 风险：字号档位要接设置面，否则第 4 步之前只有常量。

### 第 3 步：组件契约 + P0 组件（2–3 个 PR）

- 先在 `internal/ui` 落一份组件映射（与 A 的 19 个目录一一对应，标注 已实现/部分/WIP/not-planned），作为后续 PR 的验收基线。
- P0 实现顺序（按首屏可见度）：
  1. 侧栏：线程分组（项目/日期段）、搜索框、置顶、悬停态、选中态（对齐 `sidebar/ThreadList.vue`、`PinnedThreads.vue`、`SidebarHeader.vue`）；重命名/归档/拖拽先按 WIP 入口 + `wipAction` 处理。
  2. timeline 行模型：把 `Author` 四值枚举（`shell.go:49-79`）升级成行类型（user / assistant / thinking / tool / approval / file-changes / retry / error），至少补 thinking 折叠、工具卡可展开、代码块带语言头。
  3. composer：多行自适应高度、附件 chip、模型/模式选择器占位、快捷键提示行。
  4. 审批/权限面板：补权限范围与 elicit 追问两种形态。
- 验收：每个组件补 `*_test.go`（`ui.NewTester` 文本 + Role 断言），并在第 5 步出图。

### 第 4 步：外观设置面（P1，1 个 PR）

- 新增 appearance 设置页（对齐 `settings/AppearanceSettings.vue` 513 行 + `SkinSettings.vue` 833 行），先只做：主题（system/light/dark）、字号档位、字体族、密度（侧栏宽度）。
- chrome 主题（6 个）与 skin（`skin-schema.ts` 的 region/variants/淡入淡出 300ms，`skin/skin-resolver.ts`）成本极高，**明确标 WIP 入口**，不要先用假主题顶上。
- 验收：设置改动能通过 `ui.Context.SetTheme`（mygo `ui/api_gen.go:585`）生效并持久化到自有配置，重启后保持；深色用例用 `Tester.SetDark`（mygo `ui/headless.go:483`）覆盖。

### 第 5 步：视觉证据闭环（P1，持续）

- 用 mygo 现成能力出图：`Tester.Image()` + `MYGO_UI_PNG`（mygo `ui/ui_test.go:17-31`），固定 fixture（两个会话、6 类行）输出 1440×900 的 light/dark 两张 PNG。
- 参照物：从 A 用 e2e 脚本抓同一状态（`pnpm test:e2e:*`，脚本模式见 `scripts/test-electron-chat-windows.mjs:238-241`），把两张图并排存档到 `docs/evidence/ui-parity/`。
- 加一条像素回归：用 A 已有的 `scripts/e2e/compare-png.swift` 思路（区域 + 阈值），先在 `internal/ui` 里做"同一 commit 前后"的稳定性回归，再谈跨端对比。
- 验收：CI 里能看到 light/dark 两张图 diff 为 0；跨端对比先人工评审，不设自动阈值（技术上不同渲染器，自动阈值会误报）。

## 6. 结论复述

- **不是 1:1**。差距的根因是 token 语义层只有一层（7 个色）而不是三层（原始色 → 语义 → 皮肤/主题覆盖），组件层因此 16/19 个目录完全没有对应物，另 3 个（sidebar、chat-workspace、markdown）只有骨架。
- 最省力的路径不是"补组件"，而是"先补语义 + 量纲，再补 3 个 P0 组件（侧栏、timeline、composer），最后用截图闭环验证"。前两步不做，后面每加一个组件都会新增一批硬编码色值。
- 已具备的有利条件：mygo 的 `ui.Theme` 已经覆盖 A 需要的绝大部分原始色槽（`Background/Text/SurfaceHover/Selection/Focus/Inverse/...`），B 只是没接线；mygo 的 headless tester 已经能出 PNG，证据闭环不缺工具，缺的是用例。
