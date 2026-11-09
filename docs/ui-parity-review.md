# UI 逐区域视觉复核

Status: current
Last updated: 2026-11-09
证据等级：**L5 复核记录（未通过）**

脚本只能量“差多少像素”，不能判断“内容是不是同一件东西”。这份记录是逐张打开原图看出来的结论；`docs/evidence/ui-parity/parity-*.diff.png` 与 `scripts/parity_diff.py` 的数字只作辅助。

## 布局与内容差异

用 ASCII 并排图和编号标注图沟通差异，见 [ui-parity-layout-diff.md](ui-parity-layout-diff.md)（`layout-annotated-*.png`）。复核时先看这张图，再逐区域看原图。

## 复核方法

- 同状态、同尺寸：Electron 参考与 MyGo 截图都缩放到 **800×720**，浅色主题。
- MyGo 截图由 `scripts/capture-window.sh` 生成：使用一次性 `DIMAGENT_CONFIG_HOME`（强制 light）与固定窗口位置，避免继承审阅者的主题/位置。
- 每个区域单独裁图，存放在 `docs/evidence/ui-parity/review/`：
  `{empty,conv}-{electron,mygo}-{sidebar,top,content,composer,bottom}.png`
- 复核者逐张看过上述 crop；下面是看到的东西，不是脚本推断。

## 空态（Electron workspace ↔ MyGo disconnected）

| 区域 | Electron | MyGo | 判定 |
|---|---|---|---|
| 窗口框架 | 隐藏标题栏，红绿灯浮在侧栏上，内容从 y=0 开始 | 原生深色标题栏（约 23px，“DimAgent”），内容整体下移；浅色主题下标题栏仍是黑色 | 不一致（P0） |
| 侧栏顶部 | 无 wordmark；入口从约 73px 开始 | “DimAgent” + `Update WIP`，入口从约 105px 开始 | 不一致（P0） |
| 侧栏条目 | 新建会话/搜索/定时任务/技能·插件；无 WIP 徽标 | 同四项 + 每项右侧 `WIP`，另有 Session keys、Sessions 0 + Group WIP | 内容/顺序不一致（P0） |
| 会话区 | “会话”标题 + 添加/列表图标；“工作区” + 首页图标；“暂无会话” + 分隔线 | `Sessions 0` + `Group WIP`；“No sessions yet”；无分隔线 | 内容不一致 |
| 侧栏底部 | 仅“登录 Dim” + 齿轮 | Pin/Rename/Archive/Delete（0 会话时仍出现）、“Unavailable capabilities” 卡片、Sign in + Settings | 多余区域（P0） |
| 语言 | 中文 | 英文 | 对照条件不一致（D5） |
| 顶栏 | 空态没有应用顶栏；卡片直接开始 | “App Server: Not connected” + Settings 的 45px 顶栏 | 多余顶栏（P0） |
| Hero | 绿色品牌图形（约 72px）+ “开始构建” 22px，无副标题 | 绿色圆角方块 + 字母 D + “Start building” 28px + 副标题 | 图形/字号/文案均不同（P0） |
| Composer | 461×168 卡片、圆角约 14、有阴影；占位“发送消息…”；`+ / » / 立方体` 图标；右侧“未配置”+ 绿色圆形发送；context row 在卡片内（工作区 ▾ / 默认权限） | 全宽卡片；占位“Message DimAgent”；文字 WIP 按钮 Attach file/Mode/Model；提示行；context row 在卡片外（Workspace WIP / Default permissions WIP）；无阴影 | 结构不一致（P0） |
| 底部 | 无全局状态栏 | 35px 状态栏（Branch/Context/Model/Account/App Server） | 多余区域（P0） |
| 颜色 | 侧栏 `#E9EDF2`、卡片 `#FFFFFF` | 侧栏 `#E9E9EC`、卡片 `#FFFFFF`；标题栏黑 | 接近但未达阈值 |

## 对话态（Electron conversation ↔ MyGo conversation）

| 区域 | Electron | MyGo | 判定 |
|---|---|---|---|
| 顶栏 | 面包屑“dimagent-m… / 会话标题…” + `…` 菜单；右侧 VS Code + 下拉、分隔线、面板开关；高 41 | 标题 + “Session: Idle” + Settings + 面板开关；内容条约 45，其上还有原生标题栏 | 结构与内容不一致（P0） |
| 侧栏 | 工作区分组 + 会话行（标题 + 相对时间），无 wordmark/能力卡片 | 已对齐分组与相对时间；仍保留 wordmark/Update/WIP/Branches/能力卡片/额外 footer；行首多一个状态圆点 | 部分接近，整体不一致 |
| 消息区 | 用户气泡右对齐浅灰；助手纯文本、无作者行；左右留白约 24 | 同方向；助手无作者行；文本/留白接近 | 接近（仍受字体度量影响） |
| Composer | 520 宽卡片；图标工具栏；“未配置”+ 绿色发送；context 状态条在卡片下方但属于卡片 | 全宽；文字 WIP；context row 在卡片外；另有全局状态栏 | 结构不一致（P0） |
| 底部状态 | 权限 / git 分支 / 上下文条 / 短横 | Branch / Context / Model / Account / App Server | 内容不一致 |

## 动画

未采集帧序列。已知事实：Electron 的样式表定义了 hover/主题/面板过渡；MyGo 目前没有过渡 token，主题切换是整帧重绘。**没有帧证据，M 轴不通过。**

## 新路由与浮层（真实窗口，2026-11-10）

- 设置路由：[`evidence/ui/mygo-settings-route.png`](evidence/ui/mygo-settings-route.png)。整窗替换侧栏/聊天壳，Back + 19 项图标导航，默认落“个性化”；Appearance 卡片/Select/Segmented/Slider 与参考同构。导航已对齐参考 19 项（Capabilities 内容并入 About）；差异：图标为近似线条、字体栅格、内容宽度。未命名会话在侧栏/面包屑/搜索显示本地化 Untitled。
- 全局搜索：[`evidence/ui/mygo-search.png`](evidence/ui/mygo-search.png)。⌘G 居中浮层、遮罩、最近聊天、行图标/标题/项目/⌘n；与参考 `window-search.png` 同构。差异：占位文案语言、无置顶分组。
- 会话内查找：[`evidence/ui/mygo-thread-search.png`](evidence/ui/mygo-thread-search.png)。⌘F 顶栏查找条、0/0 计数、上/下/关闭；参考 ThreadSearchBar 未采集截图，结构按源码实现，待参考帧。

## Pin / Archive（真实窗口，2026-11-10）

- 置顶：[`evidence/ui/mygo-pinned.png`](evidence/ui/mygo-pinned.png)。置顶会话排在项目组内最前，行尾 pin glyph；参考没有独立 Pinned 分组，这里保持同一结构。
- 归档：`Settings → Archived sessions` 列出并恢复；侧栏隐藏、搜索不显示；archived/pinned 只存 session id，属桌面本地展示状态（config v3）。

## 会话菜单（真实窗口，2026-11-10）

- [`evidence/window-menu.png`](evidence/window-menu.png)。第一组按参考顺序：Pin、Rename、Open in new window、Export、Archive、Copy link/ID/debug（无合同的保持 WIP）；分隔线后是本项目已有能力：Session keys、End branch、Refresh branches、Delete session、Settings。
- Copy conversation ID / Copy debug info / Copy conversation link 均为真实动作（剪贴板）；link 为 `dimagent://thread/<id>`，启动参数可解析并选中会话；无 bundle 时系统级 scheme 注册失败（LaunchServices 限制），build/package 阶段补 `urlSchemes`。菜单多出 branch/settings 组（功能需要）。

## 设置子页（真实窗口，2026-11-10）

- Models：[`evidence/ui/mygo-settings-models.png`](evidence/ui/mygo-settings-models.png)。读取 `provider.management` catalog；mock daemon 未声明该能力时显示 WIP 原因，不伪造列表。
- Shortcuts：[`evidence/ui/mygo-settings-shortcuts.png`](evidence/ui/mygo-settings-shortcuts.png)。列出真实接线的 ⌘G/⌘F/Enter/Shift+Enter；编辑键位仍 WIP。
- About：[`evidence/ui/mygo-settings-about.png`](evidence/ui/mygo-settings-about.png)。真实构建信息（版本/Go/MyGo/协议/语言）+ 剩余能力缺口。

## Chrome 主题（真实窗口，2026-11-10）

- [`evidence/ui/mygo-chrome-win98.png`](evidence/ui/mygo-chrome-win98.png)。Windows 98 chrome 已生效：灰底、深蓝选中、方角控件、系统字体；切换后整壳立即变化并持久化。
- 已实现 6 套（default / Windows 98 / XP / Vista / macOS Big Sur / OSX Tiger），颜色/字体/圆角来自参考 `retro-themes.css`；渐变、3D 斜角、阴影、标题栏、vibrancy、滚动条宽度未仿制，按平面绘制。
- 圆角已全部改用 token；图标改为与参考同源的 Lucide 0.577.0（sidebar/settings/header/composer/search/find bar），品牌 mark 与 VS Code 仍为自有/近似。
- Elevation 阴影已加：主卡片/settings/composer/approval/context/search/model 对话框；会话菜单与 keys 面板沿用 mygo 自带阴影/发丝线；渐变/3D/inset 阴影、vibrancy、滚动条宽度仍为平面近似。

## 图标集（真实窗口，2026-11-10）

- [`evidence/ui/mygo-lucide-icons.png`](evidence/ui/mygo-lucide-icons.png)。图标来自本地 `lucide-vue-next@0.577.0`（ISC），统一 24×24 viewBox → 16 DIP，currentColor；覆盖侧栏、设置导航、顶栏、composer、搜索与查找条。
- 品牌 mark 保留自有图形；VS Code 入口为近似 mark（未使用第三方 logo 资产）。

## 顶栏模型入口（真实窗口，2026-11-10）

- [`evidence/window-header-model.png`](evidence/window-header-model.png)。顶栏 trailing 顺序 `… → 模型 chip → panel`，chip 显示当前 provider/model，点击打开与 composer 相同的模型列表；参考的 VS Code 入口未做（无对应功能）。
- 窄窗截断，Settings/Keys 路由打开时不画。

## 状态证据（2026-11-10）

- 已有真实窗口：空态、对话、closed、menu、hover、search、settings、pinned、branch、header model。
- loading 仍无稳定真实帧（本地 mock 过快）；focus ring 只有 headless 断言，没有窗口截图；D7 仍不能通过。

## 像素基线（辅助，不是结论）

同尺寸 800×720，区域按 sidebar/top/content/composer/bottom：

| 对照 | sidebar MAE / >16% | top | content | composer | bottom |
|---|---|---|---|---|---|
| 空态 | 19.5 / 26.0% | **164.2 / 76.9%** | 6.3 / 6.9% | 8.0 / 8.9% | 9.5 / 20.6% |
| 对话态 | 25.0 / 26.9% | **158.3 / 78.7%** | 6.2 / 5.4% | 9.2 / 11.2% | 10.5 / 21.9% |

top 的极端值主要来自“原生标题栏 vs 无标题栏”，这正是必须视觉确认、不能只看总分的例子。

## 门禁规则

1. 任何 D 行必须同时具备：**逐区域视觉复核记录**（谁看的、看了哪些 crop、逐条结论）+ **脚本指标在阈值内**。
2. 数字达标但内容/顺序/可见性不同 → C 轴不通过。
3. 视觉复核认为“看起来不是同一个界面” → 直接不通过，不允许用平均分解释。
4. 动画必须有帧序列/录屏；静态截图不能替代。

当前结论：**D1–D7 仍全部未通过**。

- 样式：布局坐标已对齐（侧栏纵向 ≤2.5 DIP、气泡 top 73.5、正文左右 24.5、状态带元素 x ±2）；仍差字体栅格（AppKit vs Chromium）、VS Code/面板图标形状、skin/chrome。
- 内容：结构、文案、分组、相对时间、占位、未配置、状态带均已对齐；仍差能力禁用项的视觉表达与参考的实际可用入口。
- 动画：已按参考时长/曲线实现并留帧（[ui-motion.md](ui-motion.md)）；参考侧缺同操作帧序列，且文字 ink 不参与过渡。
