# Electron 端参考窗口（L5 设计对照基线）

本文件记录一次**真实运行**的 dim-electron 窗口采集：命令、结果、窗口尺寸、截图路径，以及每张图能/不能证明什么。所有结论都来自本次运行的命令输出和截图本身；没看到的不写。

- 截图目录：[`docs/evidence/electron-reference/`](evidence/electron-reference/)，目录内 [`README.md`](evidence/electron-reference/README.md) 是文件清单
- 证据等级定义见 [`docs/acceptance.md`](acceptance.md)：L5 = token/组件/视觉与 Electron 端逐项对照；本文件提供的是 L5 的**像素基线**，不是 mygo 侧的验收结论
- 对照源码的审计结论见 [`docs/ui-parity.md`](ui-parity.md)，本文件补的是它缺的"真实像素"

## 1. 环境与源码状态

| 项 | 值 |
|---|---|
| 机器 | macOS 26.6.2（darwin/arm64），显示器逻辑分辨率 2048×1152，窗口按 2x 抓图 |
| dim-electron | `HEAD = 3339f4fd47f7b0569e185f1a9d81b82d16447306`（2026-11-08 17:22 +0800） |
| 工作区状态 | 采集前 `git status --porcelain` 只有 3 条**既有**的 docs 改动（`M docs/README.md`、两个 untracked `docs/architecture/*.md`），与本次采集无关；采集只写入了被 gitignore 的 `packages/*/dist` |
| Electron | `electron@43.6.0`（`packages/desktop/node_modules` 内已有；本次首次 `require('electron')` 时自动下载了二进制） |
| App 版本 | `@archships/dim-desktop@0.9.57` |
| App Server | `/tmp/dim-2-target/debug/dim-app-server`（Mach-O arm64，401 302 448 B，mtime 2026-11-08 23:38） |
| 截图工具 | `/usr/sbin/screencapture -x -o -l <windowid>` |
| 窗口 ID 来源 | `CGWindowListCopyWindowInfo`，按**本次启动的 Electron 主进程 PID** 过滤；脚本内用 `swiftc` 现场编译小工具读取 |

采集时机器上还有用户自己安装并运行中的 `/Applications/DimAgent.app`（真实 `~/.dimcode`）。本次没有读写它：`DIMCODE_HOME` 和 `DIMCODE_USER_DATA_PATH` 都指向 `/tmp/dim-electron-capture/` 下的临时目录，结束时按 PID 清理。

## 2. 构建（已成功）

dim-electron 工作区里 `node_modules` 已存在，但 `packages/renderer/dist` 与 `packages/electron-main/dist/app/main.cjs` 不存在，因此先构建：

```bash
cd /Users/eric8810/Code/dim-electron/packages/desktop
pnpm build:renderer              # vite build -> packages/renderer/dist；实测 21.9s
node scripts/build-electron-main.mjs   # esbuild -> packages/electron-main/dist；打印 "electron-main build complete"
```

没有跑 `pnpm install`，没有跑完整的 `pnpm build`（它还包含 pet-composition / global-hotkey / procprobe 等原生产物，本次用不到）。构建只写了 gitignore 的 `dist/`，dim-electron 的源码没有被修改。

`packages/electron-main/dist/app/main.cjs` 与 `packages/renderer/dist/index.html` 是后续所有启动的前置条件；`scripts/run-electron.mjs` 在非 dev 模式下会检查这两个文件，缺失直接退出。

## 3. 启动与截图

直接启动 Electron 主进程（不经 `pnpm dev` 的 vite 服务），由自己持有 PID，便于按 PID 取窗口和清理：

```bash
WORK=/tmp/dim-electron-capture
export DIMCODE_HOME="$WORK/home"                    # 临时，不碰 ~/.dimcode
export DIMCODE_USER_DATA_PATH="$WORK/userdata"      # 临时 userData
export DIMCODE_APP_SERVER_BIN=/tmp/dim-2-target/debug/dim-app-server
export DIMCODE_RENDERER_URL="file:///Users/eric8810/Code/dim-electron/packages/desktop/packages/renderer/dist/index.html"
export DIMCODE_PRELOAD_PATH=".../packages/electron-main/dist/preload.cjs"
export DIMCODE_CHROME_HELPER_PATH=".../dist/bin/dimcode-chrome-native-host.cjs"
export DIMCODE_RESOURCES_ROOT="/Users/eric8810/Code/dim-electron/packages/desktop"
export DIMCODE_DEV_MODE=0
export DIMCODE_APP_VERSION=0.9.57
cd /Users/eric8810/Code/dim-electron/packages/desktop
"$ELECTRON" --remote-debugging-port=9333 "$MAIN" &
```

- 窗口出现后 `sleep 2` 再截图；窗口 ID 由 `winlist <PID>` 在 layer 0 的窗口里挑**有标题**的最大窗口（`DimAgent`）。同一 PID 下另有 4 个 `0,0,2048,30` 的无标题窗口（层 0）和一个 layer 3 的 800×600 窗口（宠物/辅助窗口），都被排除。
- 应用自己拉起的 daemon（取自 `window-initial.txt`）：

  ```
  81358 /tmp/dim-2-target/debug/dim-app-server --unix /private/tmp/dim-electron-capture/home/rust/daemon.sock
        --no-ws --home /private/tmp/dim-electron-capture/home/rust
        --product-home /tmp/dim-electron-capture/home
  ```

- 交互（点「跳过」、打开设置、切深色、进各设置分区）通过 `playwright-core` 连 `http://127.0.0.1:9333` 的 CDP 完成；截图仍然是 `screencapture` 截**真实窗口**，不是 CDP 的 `Page.captureScreenshot`。
- 驱动脚本：[`scripts/capture-initial.sh`](evidence/electron-reference/scripts/capture-initial.sh)、[`scripts/capture-step.sh`](evidence/electron-reference/scripts/capture-step.sh)、[`scripts/interact.mjs`](evidence/electron-reference/scripts/interact.mjs)、[`scripts/winlist.swift`](evidence/electron-reference/scripts/winlist.swift)。

## 4. 采集结果（2026-11-09 14:30–14:37 CST）

窗口位置/尺寸是 `CGWindowListCopyWindowInfo` 报的 points；PNG 是 2x 像素。九张图对应的窗口都是 800×720 points，PNG 统一 1600×1440。

| 状态 | 截图时间 | 窗口 ID | 窗口 bounds（points） | PNG 像素 | 字节 | MD5 | PNG |
|---|---|---|---|---|---|---|---|
| 首次启动 | 14:30:37 | 48447 | `400,30,800,720` | 1600×1440 | 194 408 | `aa044d5dede8a4e697b4b07dc8a9864e` | [`window-initial.png`](evidence/electron-reference/window-initial.png) |
| 空工作区（浅色） | 14:36:56 | 48586 | `624,152,800,720` | 1600×1440 | 130 620 | `3f4c05843ac1537aea7d3aa072c67f0d` | [`window-workspace.png`](evidence/electron-reference/window-workspace.png) |
| 技能/插件 | 14:37:00 | 48586 | `624,152,800,720` | 1600×1440 | 196 902 | `5ac32d0c44c41f87bc2cdba5ebd3259b` | [`window-skills.png`](evidence/electron-reference/window-skills.png) |
| 设置 · 个性化 | 14:31:15 | 48508 | `400,30,800,720` | 1600×1440 | 245 248 | `05864a3540816db15ba3be2cc9a5f512` | [`window-settings.png`](evidence/electron-reference/window-settings.png) |
| 设置 · 外观 | 14:31:19 | 48508 | `400,30,800,720` | 1600×1440 | 190 612 | `3308a15d57874fda97095ae79820d248` | [`window-appearance.png`](evidence/electron-reference/window-appearance.png) |
| 设置 · 外观（深色） | 14:31:24 | 48508 | `400,30,800,720` | 1600×1440 | 182 498 | `d18597f3a44b2bdc76eb99a293a164cc` | [`window-settings-dark.png`](evidence/electron-reference/window-settings-dark.png) |
| 空工作区（深色） | 14:31:28 | 48508 | `400,30,800,720` | 1600×1440 | 118 774 | `86bae599e41bfbfa7eec313709bcb65b` | [`window-workspace-dark.png`](evidence/electron-reference/window-workspace-dark.png) |
| 设置 · 权限 | 14:31:51 | 48535 | `624,152,800,720` | 1600×1440 | 230 672 | `83f80a3633f06fe4f27878821035a9a4` | [`window-permissions.png`](evidence/electron-reference/window-permissions.png) |
| 点击「新建会话」后 | 14:32:05 | 48554 | `624,152,800,720` | 1600×1440 | 131 330 | `b5fec3b58c2718f182c7ea46cee5b4ac` | [`window-new-session.png`](evidence/electron-reference/window-new-session.png) |

同名 `.txt` 是脚本写出的机器可读元数据（含 app PID、daemon 命令行）。

`window-workspace.png` / `window-skills.png` 这两张是用归档后的 [`scripts/capture-step.sh`](evidence/electron-reference/scripts/capture-step.sh) 原样重跑生成的（14:36:56 / 14:37:00），其余几张由同一脚本的早期版本生成、逻辑相同。重跑只改了输出路径的推导方式，因此这两张也是"脚本可复现"的证据。

窗口尺寸与源码一致：`DEFAULT_WINDOW_WIDTH = 800`、`DEFAULT_WINDOW_HEIGHT = 720`（`packages/desktop/packages/electron-main/src/config/constants.ts:54-55`），`vibrancy: 'hud'` + `titleBarStyle: 'hidden'`（`bootstrap.ts:3085-3092`）解释了截图里没有系统标题栏、红黄绿按钮浮在侧栏左上角。

## 5. 每张图

### 5.1 `window-initial.png` — 首次启动

看到：整窗被半透明遮罩压暗，居中一张白色卡片：「欢迎使用 DimAgent」「登录后即可使用 Dim 提供的模型」，按钮「登录 Dim 账号」（绿色实心）/「登录企业账号」/「跳过」，下方一段"首次启动会在后台下载并校验插件所需的 Node.js 运行时"的说明。卡片背后能看到侧栏（新建会话 / 搜索 / 定时任务 / 技能/插件 / 会话 / 工作区 / 暂无会话）和底部「登录 Dim」「设置」。

能证明：全新 `DIMCODE_HOME` 首启会先出登录/欢迎浮层，浮层期间主界面已经在后面渲染。
不能证明：登录后的内容、登录流程本身、企业登录入口的实际行为。

### 5.2 `window-workspace.png` — 空工作区（浅色）

看到：左窄侧栏 + 右侧一块圆角白色主区卡片。侧栏自上而下是「+ 新建会话」「搜索」「定时任务」「技能/插件」，分组标题「会话」右侧两个小图标，「工作区」标题下「暂无会话」；底行「登录 Dim」和齿轮。主区中央是绿色 logo、「开始构建」，下面一张 composer 卡片：「发送消息…」占位、左下 `+` / `»` / 方块三个图标、右下「未配置」和一个浅绿发送箭头；卡片底部还有一行「工作区 ⌄」和「默认权限」。

能证明：空会话状态下主区的真实排版（logo 位置、composer 尺寸和内部三行结构）、侧栏信息层级、浅色主题的实际色值。
不能证明：有内容时的 timeline 排版、消息气泡、工具卡片、审批面板、流式更新。

### 5.3 `window-new-session.png` — 点击「新建会话」后

看到：与 5.2 几乎相同，区别是侧栏「新建会话」拿到灰色选中背景、composer 输入框获得焦点光标；**侧栏仍然是「暂无会话」**，没有多出会话项。

能证明：点击入口有可见的焦点/选中反馈，但在这个未配置模型的状态下没有产生会话行。
不能证明："新建会话"到底卡在哪一步（模型未配置、还是只聚焦 composer 而不落库），本次没有查证，不下结论。

### 5.4 `window-skills.png` — 技能/插件

看到：主区换成技能页。顶部两个 tab「技能」「插件」，右上「创建技能」和下拉；页内两个二级 tab「技能商店」「我的技能」。内容是一张「从来源安装」卡片（来源输入框 `/绝对路径 或 https://example.com/skills`、安装 ID（可选）输入框「根据来源生成」、「预览」按钮），下面是「已安装来源」区块（说明"通过 App Server 安装的来源。插件提供的技能仍由对应插件管理。"、右上「刷新」、虚线框里的空状态"尚无来源安装。"）。侧栏的「技能/插件」拿到选中背景。

能证明：技能页的真实布局、表单尺寸、空状态文案；这是新会话之外唯一一个能在未配置模型时正常打开的完整页面。
不能证明：安装流程是否可用（本次没有提交任何来源），插件 tab 的内容（没切过去）。

### 5.5 `window-settings.png` — 设置 · 个性化

看到：整窗换成设置面（左导航 + 右内容，不再是侧栏 + 主区卡片）。左导航顶部「← 返回」，之后依次：账户、用量、个性化、外观、通用、快捷键、模型供应、多模态、运行环境、MCP、通知、宠物、权限、Git、Hooks、插件设置、外部 Agent（滚动后还能看到"归档会话""关于"）。当前选中「个性化」，内容：界面模式（办公模式 / 编程模式两张卡，带选中描边）、「展示 ACP 会话」开关、「默认打开目标应用」下拉（系统默认）、「助手模型」下拉（灰显，文案"App Server 模式下不可用"，下面解释该设置没有归属方）、「子代理模型」下拉（Agent 选择）。

能证明：设置面的导航分区清单和真实宽度、卡片/开关/下拉的视觉规格；也证明"助手模型"在 App Server 模式下是明确 WIP 而不是缺失。
不能证明：各分区的实际功能（除外观、权限外没有逐个打开），以及设置落盘后的行为。

### 5.6 `window-appearance.png` / `window-settings-dark.png` — 外观（浅色 / 深色）

看到：同一页面的两种主题。主题是三张卡片「浅色」「深色」「跟随系统」（当前「跟随系统」有绿点和绿描边）；「程序语言」下拉「简体中文」；字体区 UI 字体 / 代码字体两个下拉（默认）；UI 字号一行七个档位（特小…特大，当前「中」绿底）；代码字号滑杆 12px–20px（当前 13px）；代码主题下拉（Vitesse、我的皮肤、Windows 98/XP/Vista、macOS Big Sur、OS X Tiger、默认）；「我的皮肤」区空状态"暂无自定义皮肤。可用图片新建或导入已有皮肤。"，按钮「用图片新建」「打开目录」。

深色图是点击「深色」后立刻抓的：整窗（含左导航、卡片、下拉、滑杆）同帧切到深色，主题切换不需要重启。

能证明：两套主题的真实色板与组件着色、主题切换的即时性、外观层的完整选项清单。
不能证明：颜色 token 的具体数值（要读 CSS/取色，不在本文件范围）、皮肤与代码主题的实际渲染。

### 5.7 `window-workspace-dark.png` — 空工作区（深色）

看到：与 5.2 同状态、深色主题。侧栏「新建会话」仍是高亮行，主区黑底卡片、logo 和「开始构建」反白，composer 深色描边，「未配置」灰字，发送箭头仍是绿色。

能证明：亮/暗两套主题在同一界面状态下的对应关系，可用来做 mygo `Light/Dark` 的逐项对照。
不能证明：系统外观跟随（本次是手动切主题，不是改系统外观）、其它皮肤。

### 5.8 `window-permissions.png` — 设置 · 权限

看到：「进程隔离」卡片（说明"控制命令是否在 App Server 沙箱中运行…"、「在沙箱中运行命令」开关、「允许网络访问」开关、「保存隔离设置」按钮、脚注"新命令使用这些设置；正在运行的进程保持启动时的隔离边界。"）；下方「权限 / Exec command 权限」区块（"App Server 命令规则作用于 exec 和 execute 调用。"、「规则」说明"deny 优先于 allow。exact 匹配完整命令词；prefix 匹配开头的词。"、三列表格「命令 / 匹配 / 决策」、「添加规则」、「保存权限」）。

能证明：Electron 端权限设置的完整视觉形态，以及"deny 优先 / exact / prefix"这套规则模型就摆在使用者面前。
不能证明：规则是否真的生效（本次没有改设置，也没有产生 approval 流量）。

## 6. 从像素量到的几何（points = PNG 像素 ÷ 2）

用 PIL 在截图上取颜色转折量出来的，误差约 ±1 pt：

| 区域 | 数值 | 来源 |
|---|---|---|
| 窗口 | 800×720 | 窗口 bounds + `constants.ts:54-55` |
| 侧栏宽 | 256（内容边界在 x=512 px） | `window-workspace.png` 中行扫描 |
| 主区卡片 | 左 256.5、右 796、上 5、下 716 | 同图上下左右扫描；即右侧/底部内距 4，顶部 5 |
| 空状态 composer 卡片 | x 295–756.5（宽 ≈461）、y 365.5–533（高 ≈167.5） | 同图边框扫描；水平方向左右各留 ≈39 |
| composer 底部「工作区 / 默认权限」行 | 卡片内最后 ≈36 高 | 同图 |
| 设置面 | 整窗白板 x 4.5–796；左导航与内容分隔线在 x≈228.5 | `window-settings.png` |

侧栏 256 pt 与 mygo 侧规格的 256 DIP（[`docs/ui-layout-spec.md`](ui-layout-spec.md) §1.2）一致，这是目前唯一一个有像素直接对上的数值；其余数值只能说明"Electron 端长这样"，不能反推 mygo 应该取多少。

## 7. 失败与未覆盖

1. **第一次非 dev 启动没有出窗口**。命令与 §3 相同，只是没有设 `DIMCODE_USER_DATA_PATH`（用了 `--user-data-dir=<temp>`），`DIMCODE_DEV_MODE=0`。进程在打印下面两段后退出，`ps` 里已无进程：

   ```
   [dimcode][feature-runtimes] initialization failed xr [FeatureRuntimeError]: Bundled dim is unavailable
       at tFe (.../dist/app/main.cjs:860:8322) { code: 'FEATURE_STATE_INVALID' }
   DevTools listening on ws://127.0.0.1:9333/devtools/browser/...
   ```

   加上 `DIMCODE_USER_DATA_PATH` 指向临时目录后，同样的非 dev 模式一次成功。**原因未查证**：可能是单实例锁（用户的 `/Applications/DimAgent.app` 正在跑），也可能是别的路径问题；本文件不把它写成结论。
2. **每次启动都会打出一条 `Bundled dim is unavailable`（FEATURE_STATE_INVALID）**。它没有阻止窗口出现，`feature-runtimes` 回落到 `<userData>/feature-runtimes/unavailable`；这会影响功能运行时（如 OCR）相关能力，本次没有验证。
3. **没有会话内容**：全新 home 下没有 session，应用显示「未配置」（无模型）。点了「新建会话」也没有产生会话行。因此**没有** timeline、消息气泡、tool 卡片、thinking、file changes、approval 决策、流式更新、断线重连的像素证据。要补这些，需要真实模型配置（或上游给的 fixture），本次没有伪造。
4. **没有系统外观跟随**：主题是在设置里手切的，没有改 macOS 外观后重抓。
5. **没有 hover / 焦点 / 菜单 / 弹窗态**：只抓了静态停留态。
6. **窗口定位不可用**：早期一次采集时窗口被系统放在屏幕外（`x=-1232`），那批产物已用 800×720 在屏的重抓替换；本次交付的 9 张都是窗口完整在屏，但采集过程本身不控制窗口位置，不能用来证明多屏/定位行为。

## 8. dimagent-mygo 侧下一步该对照的区域

按"有像素、能立即对照"的顺序：

1. **侧栏**（`internal/ui/sidebar.go`）：固定 256 pt、入口行（新建会话/搜索/定时任务/技能/插件）、分组标题行 + 两个图标、空态「暂无会话」、底部署名行 + 齿轮。对照 `window-workspace*.png`。
2. **空状态主区**（`internal/ui/shell.go` 的 chat 空态）：logo + 「开始构建」的垂直位置、composer 卡片 ≈461×168 的尺寸与三段结构（输入区 / 图标行 / 权限行）、「未配置」与发送按钮的关系。对照 `window-workspace*.png`。
3. **设置面**：mygo 现在完全没有（[`ui-parity.md`](ui-parity.md) 判为 P1）。Electron 的做法是"整窗替换 + 左导航 17 个分区"，最小子集建议先做「外观 + 权限」两页，直接用 `window-appearance.png`、`window-permissions.png`、`window-settings.png` 当视觉基线。
4. **主题**：`window-workspace.png` ↔ `window-workspace-dark.png`、`window-appearance.png` ↔ `window-settings-dark.png` 是同一状态的成对图，适合做 mygo Light/Dark 的逐区域比对（侧栏底色、卡片底色、描边、主按钮绿色、选中行）。
5. **权限/审批语义**：「deny 优先、exact/prefix 匹配」的规则表（`window-permissions.png`）是 mygo approval 相关 UI 的对照物；但 approval 面板本身**没有像素证据**，不要用权限设置页冒充审批态。
6. **技能/插件**（`window-skills.png`）：mygo 侧对应入口仍标 WIP（`docs/ui-layout-spec.md` 的侧栏 WIP 入口）；对照前先确认是否要保留入口。
7. **对话态整体（P0 缺口）**：先补"有内容的 timeline"像素基线，再谈 1:1。可选做法是用上游 fixture 或真实模型跑一次，再按同样脚本重抓。

## 9. 复现

```bash
# 1) 构建（dim-electron 内，只写 gitignore 的 dist/）
cd /Users/eric8810/Code/dim-electron/packages/desktop
pnpm build:renderer && node scripts/build-electron-main.mjs

# 2) 准备窗口 ID 工具
cd /Users/eric8810/Code/dimagent-mygo/docs/evidence/electron-reference/scripts
mkdir -p /tmp/dim-electron-capture
swiftc -O winlist.swift -o /tmp/dim-electron-capture/winlist

# 3) 采集（每次启动都会重置临时 home，只结束自己启动的进程）
./capture-initial.sh window-initial
./capture-step.sh dismiss-onboarding window-workspace text:技能/插件 window-skills
./capture-step.sh dismiss-onboarding tmp-1 new-session window-new-session
./capture-step.sh dismiss-onboarding tmp-1 open-settings window-settings text:外观 window-appearance \
                  text:深色 window-settings-dark text:返回 window-workspace-dark
./capture-step.sh dismiss-onboarding tmp-1 open-settings tmp-2 text:权限 window-permissions
rm -f tmp-*.png tmp-*.txt
```

脚本里固定了本机的 Electron 路径、dim-electron 路径和 `/tmp/dim-2-target/debug/dim-app-server`；换机器要改这几处常量（`ELECTRON` / `MAIN` / `DIMCODE_APP_SERVER_BIN`）。

## 10. 边界

- 本文件只提供参考基线，不改变 [`docs/acceptance.md`](acceptance.md) 里任何一行的验收状态；mygo 侧要达到 L5，仍需要它自己的对照记录。
- 截图对应的 dim-electron 工作树包含未提交的 docs 变更，源码未改；源码一变（尤其 renderer 的 `styles/global.css` 与侧栏组件），这批图就过期，需要重跑。
- 没有使用、也没有写入用户的真实 `~/.dimcode`；采集结束后 `ps` 复查：无残留 Electron 主进程，无本次临时 home 的 `dim-app-server`。
