# UI 1:1 对齐审计（dim-electron renderer ↔ dimagent-mygo internal/ui）

审计日期：2026-11-09（首轮只读审计）。复核：2026-11-09 Wave 3 落地后按当前代码和已有截图更新数字与状态；本轮没有重做跨端像素对比。

- A = `/Users/eric8810/Code/dim-electron/packages/desktop/packages/renderer/src`
- B = `/Users/eric8810/Code/dimagent-mygo/internal/ui`

## 结论

**当前不是 1:1。** B 已补上语义 token 层和 P0/P1 聊天壳，但 A 的皮肤/chrome 覆盖层、剩余组件域和跨端像素对照仍未完成。

| 维度 | A | B | 判定 |
|---|---|---|---|
| 语义 token 数量 | 156 个 CSS 变量（`styles/global.css` 全文件），其中 118 个非 `--color-*` 语义变量 | `tokens struct` 88 个字段，其中 `ui.Color` 35 个（[tokens.go:169-330](../internal/ui/tokens.go#L169-L330)） | 数量接近，但组织方式不同 |
| 颜色语义层 | 三层：原始色 `--skin-tone-*`(20) → 语义 `--background/--card/...`(45) → skin 覆盖层（带 alpha） | 两层：从 mygo 的 light/dark `ui.Theme` 派生 surface 梯度、前景配对、sidebar/tooltip 语义集；没有 skin/chrome 覆盖层 | B 缺第三层 |
| 组件目录 | 19 个顶层组件目录 + 2 个图标组件，601 个 `.vue`、93,133 行 | 23 个 Go 文件、7,594 行（非测试） | B 部分对应 sidebar、chat-workspace、markdown、settings、context-panel/right-panel、statusbar 与 mygo 基础件；约 13 个目录仍无对应物 |
| 主题/皮肤 | 3 主题偏好 × 6 chrome 主题 × 任意 skin.json 皮肤 × 可配代码主题 | system/light/dark + 外观设置（字号/字体/密度/代码字号，已持久化）；chrome/skin 明确 WIP | B 缺 chrome/skin |
| 视觉证据 | 31 个 Playwright Electron e2e 脚本产出真实窗口截图 | 6 张 headless 截图（`docs/evidence/ui/`）+ 2 张真实窗口截图（`docs/evidence/window-*.png`）+ Electron 参考基线；没有跨端叠加对照 | B 有像素证据，但未做 L5 对照 |

一句话：B 已从"只有一层颜色的静态壳"推进到"有语义 token、P0/P1 组件和截图证据的聊天壳"，但皮肤系统、剩余组件域和跨端视觉对照仍缺，不能称为 1:1。

## 1. 审计方法与证据边界

- 首轮只读源码，不做像素对比；本轮复核同样不重做跨端对照，只把与代码矛盾的旧状态改成当前事实。B 侧截图见 §4。
- "1:1" 这里指**聊天壳的视觉与交互对齐**，不是代码对齐。两侧技术栈不同（Vue 3 + CSS ↔ Go + mygo retained-mode），不可能也不应该共享实现，只能共享**语义层 + 组件契约**。
- 不评价业务能力差异（session/approval/timeline 真相都在 App Server，见 `AGENTS.md` 第 2 节），只评价"同一个会话界面看起来/用起来是否一致"。

## 2. Token 对照表

### 2.1 A 的 token 架构（三层）

1. **原始色 `--skin-tone-*`**（20 个）：`styles/global.css:867-888` 浅色、`889-911` 深色，按 `[data-skin-surface-tone]` 切换。
2. **语义 token**（45 个：核心 28 + tooltip 7 + sidebar 10）：`styles/global.css:183-256` 浅色，`272-330` 深色；由 `@theme inline`（`styles/global.css:44-95`）映射成 Tailwind 颜色工具类。
3. **skin / chrome 覆盖层**：`.skin-active.skin-host` 把同一批语义 token 重新以 alpha 绑定到 `--skin-tone-*`（`styles/global.css:923-950`、`1060`）；chrome 主题见 `styles/retro-themes.css`（955 行，`data-chrome-theme` 共 6 个值：`default`、`macos-bigsur`、`osx-tiger`、`windows-98`、`windows-vista`、`windows-xp`）。

### 2.2 语义 token ↔ B tokens 字段（当前）

| A 语义组（global.css 行号） | B `tokens` 字段 | 判定 |
|---|---|---|
| `--background` / `--foreground` (183-184) | `background`、`foreground` | **有** |
| `--card` / `--popover` + 前景 (185-188) | `card`、`cardForeground`、`popover`、`popoverForeground` | **有** |
| `--primary` / `--secondary` / `--muted` / `--accent` / `--destructive` + 前景 (205-218) | `primary`/`onPrimary`、`secondary`/`secondaryForeground`、`muted`/`mutedForeground`、`accent`/`accentForeground`、`danger`/`onDanger` | **有**；`accent` 已改回中性 hover，品牌色在 `primary` |
| `--border` / `--input` / `--ring` (219, 227-228) | `border`、`input`、`ring` | **有** |
| `--sidebar*` 10 个 (234-243) | `sidebar` 及 8 个子字段 | **9/10**：缺 `--sidebar-overlay` |
| `--tooltip*` 7 个 (192-204) | `tooltip`、`tooltipForeground`、`tooltipMutedForeground`、`tooltipBorder`、`tooltipChip` | **5/7**：缺 backdrop 与多层阴影 |
| `--radius` + `-sm/-md/-lg/-xl` (44-47, 244) | `radiusSM=6`、`radiusMD=8`、`radiusLG=10`、`radiusXL=14` | **有** |
| 字体族 (157-165) | `fontFamilyStack`：系统字体或 `monospace` 平台栈 | **部分**：未自托管 Inter / JetBrains Mono |
| 字号 (166-167) | 5 档 UI 字号 `[12,13,14,16,18]` + 独立 `codeFontSize` | **有**（默认值可能不同） |
| `--meter-track` (224) | 复用 `muted` | **有（复用）** |
| `--mark` / `--mark-soft` (209-210) | 无 | **缺** |
| `--chart-1..5` (229-233) | 无 | **缺** |
| `--activity-info/-violet/-success/-warning/-shimmer-highlight` (336-340) | `success`、`warning`；info/violet/shimmer 无 | **部分** |
| `--markdown-table-*`/`-link`/`-inline-code-bg`/`-code-block-*`/`-hr`/`-checkbox-*` (18 个) | `markdown_render.go` 的 `contentTheme(k)` 从 tokens 派生链接、表格、代码块、复选框和分割线 | **部分**：代码高亮主题、图片/右键菜单仍缺 |
| `--titlebar-height`、`--layout-transition-ms`、`--ease-*` (174, 259, 88-114) | 无（原生系统标题栏，无动效 token） | **缺** |
| `--z-sidebar-*` (250-254) | 不适用（retained-mode 单一绘制列表） | 合理差异 |

### 2.3 B 仍缺的语义层

1. **皮肤/chrome 覆盖层**：没有 A 的 `skin.json` / 6 个 chrome 主题 / 原始色 → 语义 → 覆盖层的第三级；当前只有 system/light/dark 两套 theme。
2. **状态化表面**：`non-vibrant-window`（`global.css:425-431`）、`prefers-reduced-transparency`（`433-460`）和 skin 强制 tone（`867-911`）没有对应路径。
3. **剩余色槽**：chart 1–5、mark/mark-soft、activity info/violet/shimmer。
4. **Tooltip 玻璃层**：backdrop-filter 与多层阴影未做；sidebar overlay 也没有单独字段。
5. **排版**：没有自托管 Inter / JetBrains Mono，字体选择只有系统字体和 monospace。
6. **几何/动效**：标题栏高度、过渡时长、缓动和面板进出动画仍无 token。
7. **Markdown 内容层**：图像灯箱、链接右键菜单和代码高亮主题仍缺。

### 2.4 度量差异（会直接影响观感）

| 项 | A | B |
|---|---|---|
| 侧栏宽度 | 默认 256、最小 200、最大 360、可拖拽（`components/ui/sidebar/utils.ts:4-6`） | 默认 256、200–360、可拖拽调宽、双击复位（[tokens.go:70-73](../internal/ui/tokens.go#L70-L73)、`sidebar.go`） |
| 状态栏 | `.chat-status-bar` 高 40px、左右 20px、12px 字（`styles/global.css:39-41`）；另一读数为 35 DIP（见 [ui-layout-spec.md](ui-layout-spec.md) §7） | 已实现底部状态栏，按 14px 根字号取 35 DIP（[statusbar.go:19-31](../internal/ui/statusbar.go#L19-L31)）；branch + 连接已接，context/model/account 是 WIP |
| 标题栏 | 固定 48px，自绘窗口控件（`global.css:174`、`components/app/AppTitlebar.vue`） | 原生系统标题栏 |
| 卡片圆角 | 10 / 8 / 6 / 14 px 四级（`global.css:44-47`、`244`） | 6 / 8 / 10 / 14 四级（[tokens.go:49-54](../internal/ui/tokens.go#L49-L54)） |
| 基础字号 | 14px UI、13px 代码、5 档可选、⌘+/⌘- 缩放（`appearance/index.ts:11-16`、`appearance/zoom-shortcuts.ts`） | 5 档 UI 字号 + 代码字号，可在外观设置里选并持久化；无缩放快捷键 |
| 间距单位 | Tailwind 4px 刻度 + 具名 utility | `theme.Space(n)` = n×4 DIP + 具名几何字段；compact 密度按 3/4 收紧 |
| 密度 | 可切换（字号档位 + 侧栏宽度 + 面板宽度持久化，`ChatWorkspace.vue:217-220`） | regular/compact + 侧栏宽度已持久化；面板宽度未做 |

## 3. 组件清单差异

A 顶层：19 个组件目录 + 2 个独立图标组件（`components/`）。B 的 UI 是 23 个 Go 文件（非测试）。

### 3.1 逐目录对照

| A 目录（文件数） | B 对应物 | 级别 |
|---|---|---|
| `sidebar/`（22）`AppSidebar.vue`、`ThreadList.vue`、`SidebarHeader.vue`、`PinnedThreads.vue`、`ProjectItem.vue`、`ChatSearchDialog.vue`、`RenameThreadDialog.vue`、`ArchiveThreadDialog.vue`、`ThreadContextMenu.vue`、`useSidebarVirtualRows.ts`、`useProjectDragReorder.ts` | `internal/ui/sidebar.go`：header、全局入口、可选分组线程列表、本地会话搜索、选中/悬停、200–360 拖拽调宽、双击复位；置顶/重命名/归档/右键菜单保持 visible-disabled WIP | **部分完成**：核心列表、搜索与量纲已对齐；对话框类仍是 WIP |
| `chat-workspace/`（185；timeline 65、composer 36、interactions 20、model-selector 15、git 7、attachments 6、workflow 3） | `timeline.go` + `conversation.go` + `composer.go` + `header.go` + `approval_panel.go`；主区卡片框架已实现（`aaac7b6`） | **部分完成**：见 3.2 |
| `markdown/`（20）`FileChip.vue`、`MarkdownImageNode.vue`、`MarkdownHeadingNode.vue`、`MarkdownInlineCodeNode.vue`、`ImageLightbox.vue`、`LinkContextMenu.vue`、`useMarkdownRender.ts` | `markdown_render.go`（`d5bbcb5` + `302efea`）：内容层 token、代码块头 + 语言 + Copy、行内代码、链接、表格、任务列表、文件 chip；assistant 行已接入 | **部分完成**：图片灯箱、链接右键菜单、代码高亮主题仍缺 |
| `settings/`（41）+ `views/SettingsView.vue`、`SettingsNav.vue`、19 个分区（`settings/sections.ts:24-43`） | `settings.go`（`697ed25` + `925f071`）：17 个分区的导航；Appearance 页可编辑（主题/字号/字体/宽度/密度/代码字号/语言），其余 16 个分区是说明缺口原因的 WIP 页 | **部分完成**：导航与外观页已落地；其余分区未做 |
| `context-panel/`（16）文件树、`DiffView.vue`、`FilePreviewView.vue`、`OfficeFilePreview.vue`、`PptxSurfacePreview.vue`、`XlsxPreview.vue` | `context_panel.go`（`c05bf50` + `d16b27d`）：文件树、diff、预览、空态与 WIP，已作为右侧栏绘制并可由工具栏切换；仍无 changed-file/diff 数据源 | **部分完成** |
| `right-panel/`（1）+ `ChatWorkspace.vue` 承载（`ChatWorkspace.vue:1973-2009`） | 右侧 context panel 已按 320 DIP 绘制，窄窗口让位；面板内容仍是调用方传入的 `ContextPanelView` | **部分完成** |
| `app/`（6）`AppTitlebar.vue`、`WindowControls.vue`、`BrandMark.vue`、`StartupOverlay.vue` | 无（原生系统标题栏） | **P2** |
| `scheduled/`（10）+ `ScheduledView.vue` | 无 | **P2** |
| `ui/`（406，shadcn 基础件） | mygo `ui` 提供 button/input/list/markdown 等；缺 dialog/popover/menu/command/toast/tooltip 的 shell 接入 | **P1**（按需接） |
| `market/`（7）、`plugins/`（28）、`skills/`（7）、`provider/`（1） | 无 | **P2** |
| `browser-panel/`（4）、`ui-extensions/`（4）、`mcp-apps/`、`pet/`（2）、`onboarding/`（4）、`account/`（1）、`selection/`（1） | 无 | **P2** |
| 状态栏（`chat-workspace/StatusBar.vue`） | `statusbar.go`（`697ed25`）：branch、连接、context 用量条、model/account WIP、窄宽度省略 | **部分完成** |
| keys 面板 | `keys_panel.go`（`3fef65f` + `a2ce8b8`）：`session.keys` 快照、过滤、选中、未加载 WIP | **有** |

### 3.2 chat-workspace 内部差距

| A 组件 | 现状 | B 现状 | 级别 |
|---|---|---|---|
| `ChatWorkspace.vue`（7,329 行）+ `ChatHeaderFrame.vue` + `ChatHeader.vue`（603） | 卡片式聊天框：`chat-workspace-frame`/`chat-workspace-card`（`global.css:20-37`），右侧面板按 margin 让位（`ChatWorkspace.vue:2003-2009`） | 主区圆角卡片 + 外边距 + 描边（`aaac7b6`）；右侧 context panel 按 320 DIP 绘制并在窄窗口让位（`d16b27d`） | **部分完成** |
| `timeline/TimelineRow.vue`（442）、`TimelineAssistantMessage.vue`、`TimelineThinkingItem.vue`、`TimelineToolCallItem.vue`、`TimelineFileChanges.vue`、`TimelineArtifacts.vue`、`TimelineRetryNotice.vue`、`TimelineErrorBoundary.vue`、`TimelineCompressionDivider.vue`、`tool-renderers/` | 行类型化、工具卡可展开、thinking 折叠、文件改动、重试、错误边界、压缩分割线 | `internal/ui/timeline.go`（`760413e`）+ `toolrender.go`（`b1f73a0`）：turn 内 block 分派、thinking 默认折叠、tool 卡展开、command/file/search/generic 分类面板、file changes、独立 error/retry 行 | **部分完成**：压缩分割线、artifacts、用户长文折叠仍是 WIP |
| `composer/ConversationComposer.vue`、`InputArea.vue`、`tiptap/`、`activity/`、`FollowUpMessages.vue`、`ComposerSkillsSelector.vue`、`AgentModeSelector.vue`、`model-selector/`（15） | 富输入 + 附件 + skills + slash 命令 + 模型/模式选择 + 活动条 | `composer.go`（`f157797` + `cd5cc8c`）：自适应多行、Enter 发送 / Shift+Enter 换行、提示行、工具行、底部 context row、圆形 Send；Attach/Mode/Model 是 WIP | **部分完成**：缺富文本、skills、slash 命令、活动条 |
| `interactions/PermissionPrompt.vue`、`ApprovalInputPanel.vue`、`ElicitationPrompt.vue` | 审批/权限/追问三类面板 | `approval_panel.go`（`a2cdb9b` + `ffa54c7`）：待审批卡片、call/reason/source、输入块与变体、Approve/Deny | **部分完成**：缺权限范围、追问、always-allow |
| `StatusBar.vue`（367）`ContextUsageMeter.vue`、`ChatHeader` 的模型/分支入口 | 底部状态栏：git 分支、上下文用量、账户用量 | `statusbar.go`：branch 与连接已接；context/model/account 用 WIP 徽标，不伪造读数 | **部分完成** |
| `SessionTasksPanel.vue`、`BackgroundTaskTerminal.vue`、`TerminalDrawer.vue`、`ProcessDetail.vue` | 后台任务与终端面板 | 无 | **P2** |
| `SideChatPanel.vue`、`SubagentChatPanel.vue` | 侧聊/子代理 | 无 | **P1** |
| `workflow/`、`WakeupDetail.vue` | 工作流/wakeup 详情 | 无 | **P2** |

### 3.3 B 独有、A 没有的

- 侧栏底部"Unavailable capabilities"卡片组（[capability.go:64-91](../internal/ui/capability.go#L64-L91)），把 WIP 能力当产品状态直接列出来。A 的 WIP 只做入口禁用（`packages/shared/src/wip.ts`、`settings/wip-capability-copy.ts`），不列清单。这不是差距，是 B 更强的可观测性，建议保留。
- `wipAction`（[wip.go:35-40](../internal/ui/wip.go#L35-L40)）把 disabled + 原因文本 + `DescribedBy` 绑在一起，A 侧是 Tooltip 文案，语义等价。

## 4. 视觉/交互证据差距

| 证据类型 | A | B |
|---|---|---|
| 真实窗口截图 | 31 个 `scripts/test-electron-*.mjs` e2e，Playwright `page.screenshot()` 落盘（如 `scripts/test-electron-appshots-e2e.mjs:172`、`scripts/test-electron-chat-windows.mjs:241`） | `docs/evidence/window-disconnected.png` 与 `window-connected.png` 两张真实窗口截图（`scripts/capture-window.sh`，见 [window-evidence.md](window-evidence.md)） |
| Headless 像素 | 无（Electron + jsdom 不做像素） | `screenshot_test.go` 用 `Tester.Image()` 输出 light/dark × shell/composer/settings 六张 PNG（`DIMAGENT_UI_SCREENSHOT_DIR`，存于 `docs/evidence/ui/`） |
| 像素回归 | 无 golden；仅 appshots e2e 里用 `scripts/e2e/compare-png.swift` 比较"拍照前后同一个区域" | 无 golden/自动 diff；截图用例只断言"非空且 light/dark 不同" |
| 关键路径测试 | 230 个 renderer 测试文件 | 27 个测试文件、244 个 top-level 测试（`internal/ui`） |
| 深色模式覆盖 | `styles/global.css:262-330` 全套深色值 + 6 个 chrome 主题各自的深色块 | 3 张 dark 截图（shell/composer/settings）+ `Tester.SetDark` 用例 + Markdown 深色断言 |
| 主题/皮肤数量 | 3 主题偏好 × 6 chrome（`packages/shared/src/theme.ts:5-12`、`styles/retro-themes.css`）× skin.json 任意皮肤（1 个 window region + 4 个 surface + variants，`packages/shared/src/constants/skin.ts:19,28`、`skin-schema.ts`）× 代码主题（Vitesse 默认，`appearance/code-theme-bg.ts:21-33`） | system/light/dark + 外观设置（字号/字体/密度/代码字号/语言，已持久化）；chrome 与 skin 明确 WIP |
| 字体 | 自托管 Inter + JetBrains Mono（`global.css:10-18`），5 档 UI 字号、代码字号、可自定义字体族（`appearance/index.ts:11-22`） | 系统字体或平台 monospace，5 档 UI 字号 + 代码字号 |
| 密度 | 侧栏宽度/右侧面板宽度持久化（`ChatWorkspace.vue:217-220`）、`--layout-transition-ms` 统一过渡、zoom 快捷键 | regular/compact + 侧栏宽度持久化；无面板宽度、过渡和 zoom |
| i18n | 11 个 locale（`renderer/src/i18n/config.ts`） | en + zh-CN（`i18n.go`，已持久化语言选择；默认英文） |
| 无障碍 | 部分组件有 aria；无统一契约 | 明显更强：Role/Status/Busy/Description/DescribedBy/AccessHidden 成体系（`shell.go:327-348`、`tool.go:129-139`、`wip.go:35-40`） |

结论：A 的"证据"是大量真实窗口截图（人看），B 现在同时有 headless 截图、两张真实窗口截图和无障碍树断言；但两边仍没有跨端像素回归，所以**任何"已 1:1"的说法目前都没有证据支撑**。

## 5. 对齐计划与当前进度

### 第 1 步：token 语义层（已完成）

- `internal/ui/tokens.go` 已派生 surface 梯度（background → sidebar → card → popover）、前景配对、sidebar/tooltip 语义集、中性 accent 和四级圆角。
- 验收：`tokens_test.go` 断言浅/深两套的明度关系，并断言 `accent` 与 `primary` 不再同值。

### 第 2 步：量纲层（已完成）

- 侧栏默认 256、min 200、max 360，可拖拽；字号 5 档 + 代码字号；圆角 6/8/10/14；状态栏 35 DIP；compact 密度 3/4。
- 验收：`tokens_test.go` 数值断言；侧栏宽度和外观偏好写入 `internal/config`。

### 第 3 步：组件契约 + P0 组件（已完成）

- 侧栏（分组、搜索、选中/悬停、拖拽）、timeline 行模型、composer、审批面板均已落地；区域合同见 [ui-component-protocols.md](ui-component-protocols.md)。
- 验收：每个区域有 `*_test.go`，并有 headless 截图。

### 第 4 步：外观设置面（部分完成）

- `settings.go` 已提供 17 个分区的导航；Appearance 页支持主题、字号档位、字体族、密度、侧栏宽度、代码字号和语言，并通过 `internal/config` 持久化。
- chrome 主题（6 个）与 skin 成本高，仍是 WIP；其余 16 个设置分区只显示缺口说明。
- 验收：设置改动生效并在重启后保持；深色用例用 `Tester.SetDark` 覆盖。

### 第 5 步：视觉证据闭环（部分完成）

- 已用 `Tester.Image()` 输出 6 张 1440×900 PNG；真实窗口已有 disconnected/connected 两张截图；Electron 参考窗口已采集（`docs/evidence/electron-reference/`）。
- 仍未做：把 MyGo 与 Electron 同状态图并排存档、跨端像素对比、CI 中的稳定性 diff。现有截图不能当作 L5 通过证据。

## 5.1 真实像素基线对照（2026-11-09）

`docs/evidence/electron-reference/` 已采集真实 Electron 窗口（`docs/l5-reference.md`），这是 L5 的对照基线，不是 mygo 侧结论。首轮指出的结构差距中，前四项已在 Wave 3 落地：

1. **主区是一张卡片**：已实现（`aaac7b6`）。
2. **空态 hero**：已实现（`aaac7b6`，品牌方块 + `Start building` + 副标题）。
3. **composer 结构**：底部 context row 与圆形 Send 已实现（`cd5cc8c`）；工具行仍是 WIP 入口。
4. **侧栏**：全局入口、账户行与设置按钮已实现（`6788846`）。
5. **语言**：MyGo 已支持 en/zh-CN，默认仍是英文；参考窗口是中文界面。

这些只说明结构已补上，不等于像素对齐：还没有把两边同状态截图并排比较，也没有逐区域记录差异。

### 5.2 首轮并排与取样（2026-11-09）

- 并排图：[`evidence/ui-parity/compare-empty.png`](evidence/ui-parity/compare-empty.png)（Electron workspace ↔ MyGo 空态）、[`compare-settings.png`](evidence/ui-parity/compare-settings.png)（设置面）。
- 取样 light：Electron 侧栏 `#E9EDF2`、卡片 `#FFFFFF`；MyGo 侧栏 `#E9E9EC`、卡片 `#FFFFFF`。与参考 `#E9EDF2` 仅差冷色偏移（约 6/255），层级方向一致。
- 取样 dark：Electron 卡片亮于背景；MyGo 背景 `#18181B`、卡片 `#343436`、popover `#424244`、侧栏 `#27272A`，方向一致。
- 本轮修复：elevation 从“朝 foreground 混”（light 下把卡片压到背景以下）改为“朝白色提升”；侧栏列表补 `Background(k.sidebar)`，不再被 `ui.List` 默认背景覆盖。

仍未做：逐区域像素叠加、字体度量对照、hover/focus/弹窗态、skin/chrome。

## 6. 结论复述

- **不是 1:1**。B 已有 35 个颜色字段的语义层，但没有 A 的 skin/chrome 第三层；约 13/19 个组件目录仍无对应物，sidebar、chat-workspace、markdown、settings、context-panel/right-panel、statusbar 为部分完成。
- 下一步不是重做 token，而是补剩余组件域（right-panel 的 changed-file 数据源、scheduled、market/plugins/skills、app titlebar 等）并完成跨端像素对照。
- 已具备的有利条件：token 语义层、P0/P1 组件、headless 截图和真实窗口截图都已存在；缺的是跨端对照和剩余组件，而不是工具链。
