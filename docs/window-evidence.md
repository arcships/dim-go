# 真实窗口截图证据（L4）

本文件只记录**真实运行**出来的窗口截图，不做推断式补全。每条证据都附命令、时间、窗口尺寸和截图路径；看不到的内容不写。

- 采集脚本：[`scripts/capture-window.sh`](../scripts/capture-window.sh)
- 截图目录：[`docs/evidence/`](evidence/)
- 证据等级定义见 [`docs/acceptance.md`](acceptance.md)：L4 = 真实窗口 + 人工/截图对照。

## 1. 环境与源码状态

| 项 | 值 |
|---|---|
| 机器 | macOS 26.6.2（darwin/arm64） |
| Go | go1.27.3 darwin/arm64 |
| 源码 | 两种状态都来自 `HEAD = 596ad67008a4e19cac3227ea5f7bd6e5ac573f1f`，工作区 clean（2026-11-09 15:21） |
| 构建时间 | 2026-11-09 15:21:20 CST |
| App Server 二进制 | `/tmp/dim-2-target/debug/dim-app-server`（Mach-O arm64，401 302 448 B，mtime 2026-11-08 23:38） |
| 窗口库 | `github.com/egoist/mygo v0.8.3`（`mygo.NewWindow`，`Width: 1080`、`Height: 720`） |
| 截图工具 | `/usr/sbin/screencapture -x -o -l <windowid>`（按窗口 ID 截取，只含该窗口，不含桌面和其它应用） |
| 窗口 ID 来源 | `CGWindowListCopyWindowInfo`，按**本进程 PID** 过滤；脚本内用 `swiftc` 现场编译小工具读取 |

2026-11-09 15:21 的这次采集在 `596ad67`（右侧 context panel、i18n、seed side branch 落地）上重跑，两种状态都记录 `git_dirty = clean`；`docs/evidence/window-*.txt` 的 `git_head` / `git_dirty` 是同一事实。源码再变时截图即过期，需要重跑脚本。

## 2. 复现命令

```bash
cd /Users/eric8810/Code/dimagent-mygo
./scripts/capture-window.sh                     # 两种状态都截
./scripts/capture-window.sh --state connected   # 只截已连接
```

脚本行为（与本次运行一致）：

1. `go build -o $TMP/dimagent ./cmd/dimagent`，不改动仓库内文件。
2. 未连接态：`env DIMAGENT_PRODUCT_HOME=` 启动，窗口必须显示 `App Server: Not connected`。
3. 已连接态：默认 `DIMAGENT_PRODUCT_HOME=$(mktemp -d)`；带 `--product-home DIR` 时改用调用方提供的 home（例如 `scripts/seed-session` 预置过的目录）。`PATH` 前置 `/tmp/dim-2-target/debug`，由 supervisor 自行 spawn 真实 `dim-app-server`；窗口必须显示 `App Server: Connected`。
4. 两种状态都在窗口拿到真实尺寸后 `sleep 1` 再截图；截图后按 PID 结束应用，并按「命令行里含本次 home 路径」精确结束本次 spawn 的 daemon；脚本只删除自己 `mktemp` 出来的空 home，`--product-home` 目录由调用方清理。
5. 全程不读写 `~/.dimcode`，不使用用户的真实 product home。结束时打印 `all requested captures done; no leftover daemon from this run`，并复查没有本次残留的 daemon。

## 3. 采集结果（2026-11-09）

| 状态 | 采集时间 | 窗口 ID | 窗口位置/尺寸（points） | PNG 像素 | 字节 | PNG 路径 |
|---|---|---|---|---|---|---|
| 未连接 | 15:21:20 → 15:21:25 | 48715 | `484,116,1080,720` | 2160×1440（= 1080×720 @2x） | 278 690 | [`evidence/window-disconnected.png`](evidence/window-disconnected.png) |
| 已连接（seed + side branch） | 15:21:25 → 15:21:27 | 48725 | `484,116,1080,720` | 2160×1440（= 1080×720 @2x） | 300 056 | [`evidence/window-connected.png`](evidence/window-connected.png) |

MD5：未连接 `5ede22ebcef8e276f0411047290ad29a`，已连接 `b5c1fd84d122c549a7b447215f8f62ec`。

机器可读的采集元数据（脚本生成，含源码快照、PID、窗口 ID、bounds、daemon 命令行）：
[`evidence/window-disconnected.txt`](evidence/window-disconnected.txt)、[`evidence/window-connected.txt`](evidence/window-connected.txt)。

已连接态截图时真实在跑的 daemon（取自元数据）：

```
97432 /tmp/dim-2-target/debug/dim-app-server
      --unix <tmp>/dimagent-seeded-home.eClAW8/rust/daemon.sock --no-ws
      --home <tmp>/dimagent-seeded-home.eClAW8/rust
      --product-home <tmp>/dimagent-seeded-home.eClAW8
```

截图结束后该进程随应用退出而消失，`ps aux | grep '[d]im-app-server'` 无输出；临时 home 已删除。

### 3.1 预置会话（seed → 窗口）

空 home 只能证明"连接成功"，不能证明会话列表与首屏 history。为此增加了 `scripts/seed-session`：用公开 `session.create`（可选 `session.submit`）在临时 home 里预置一个会话，再让窗口连接同一 home。

```bash
HOME_DIR=$(mktemp -d /tmp/dimagent-seeded-home.XXXXXX)
go run ./scripts/seed-session -home "$HOME_DIR" \
  -server /tmp/dim-2-target/debug/dim-app-server \
  -prompt "Explain what this repository does in one paragraph." \
  -side-branch
./scripts/capture-window.sh --state connected --product-home "$HOME_DIR"
```

2026-11-09 15:21 的结果（`git_head = 596ad67`，`git_dirty = clean`）：

- 侧栏 `Sessions 1`，选中行标题是 prompt 的前缀；`Branches` 显示 `main (current)`、真实 `session.branch_create` 产出的 side branch（`8bd478fc-…`）与 `Refresh`。
- 主区显示 user 气泡（完整 prompt）与 assistant 行（mock model 的 `Echo: ...`），即首屏 history 来自真实 daemon。
- 状态栏 `Branch: main`、`App Server: Connected`。
- 这组证据支撑 F2（真实 daemon + 会话列表）、F5（首屏 history 渲染）和 F12 的侧栏 branch 目录展示；它不覆盖窗口内点击创建/切换/提交，也不覆盖 cut point。

### 3.2 真实窗口交互（Accessibility 驱动，2026-11-09）

macOS 的 Accessibility 已授权（`AXIsProcessTrusted() == true`），因此窗口内的点击、输入和截图都是真实事件，不是 mock。驱动工具 `scripts/ax/axdo.swift`、窗口 ID 工具 `scripts/ax/windowid.swift` 已入库，可重跑。

在 `596ad67` 之后、配合 `fix(ui): select sessions on row clicks and respawn on reconnect` 的窗口里依次验证：

| 验收 | 操作 | 结果 | 截图 |
|---|---|---|---|
| F3 创建会话 | 点击侧栏 `New session` | `Sessions 2`，新会话被选中，标题是 App Server 返回的 session id，`Notices: 1` | [`evidence/window-create-session.png`](evidence/window-create-session.png) |
| F4 切换会话 | 点击第二个会话行 | 标题切到该会话，首屏 history（user + assistant 行）渲染 | [`evidence/window-switch-session.png`](evidence/window-switch-session.png) |
| F6 提交 | 在 composer 输入 `Acceptance: reply with one short sentence.`，点击圆形 Send | 出现 user 气泡与 assistant `Echo:` 行 | [`evidence/window-submit.png`](evidence/window-submit.png) |
| F10 断线重连 | `kill` 掉本次 daemon | 窗口保持打开，显示 `App Server: Connection error` 与 `Reconnect` | [`evidence/window-transport-lost.png`](evidence/window-transport-lost.png) |
| F10 重连恢复 | 点击 `Reconnect` | supervisor 重新 spawn `--mock-model` daemon，`App Server: Connected`，latest-turn history 重建 | [`evidence/window-reattach.png`](evidence/window-reattach.png) |

这一组同时暴露并修复了两个真实缺陷：

1. 侧栏行点击只移动 List 光标、不触发 `OnChange`（只绑 `Selected` 时），真实窗口无法切换会话；现在行自身也发布选择。
2. `Reconnect` 只重建 session stream，不先重连控制面；daemon 已退出时永远无法恢复。现在先 `Supervisor.Reconnect()`（自有 daemon 会被重新 spawn），再 `Reattach`。

仍未由窗口点击覆盖：stop（mock model 立即结束，没有可停止的窗口）、approval 决策（mock model 不发审批）、unknown-result 读回（需注入故障）、附件（无文件选择 API）、cut point。这些不写成通过。

## 4. 截图上实际看到的内容

两张图都来自 `596ad67`：未连接是空 catalog，已连接是 seed 会话 + side branch。共同部分是主区卡片、hero/composer 结构和侧栏入口。

未连接（`window-disconnected.png`）：

- 左侧栏：`DimAgent` 品牌行 + `Update`（WIP）；全局入口 `New session`、`Search sessions`（⌘G）、`Scheduled tasks`（WIP）、`Skills and plugins`（WIP）、`Session keys`（WIP）；`Sessions 0` + `Group`（WIP）；`No sessions yet`；线程操作 `Pin`/`Rename`/`Archive`（WIP）；`Unavailable capabilities` 两张 WIP 卡（`Session keys`、`Attachments`）；底部账户行 `Sign in to Dim`（WIP）与 `Settings` 按钮。
- 右主区：圆角卡片（外边距 + 1px 边框），卡片内空态 hero：蓝色 `D` 方块 + `Start building` + `Ask DimAgent to change code, run a command or explain a session.`；右上 `● App Server: Not connected`（灰点）与 `Settings`。
- composer：草稿框、`Enter to send · Shift+Enter for a new line`、工具行 `Attach file`/`Mode`/`Model`（WIP）与圆形 `Send`（空草稿禁用）、底部 context row `Workspace WIP` / `Default permissions WIP`。
- 状态栏：`Branch: WIP · Context: WIP · Model: WIP · Account: WIP · App Server: Not connected`。

已连接（`window-connected.png`）：

- 左侧栏：`Sessions 1` + 选中的 seed 会话；`Branches` 显示 `main (current)`、side branch（`8bd478fc-…`）与 `Refresh`；`Session keys` 不再是 WIP 卡（keys 面板已接线），`Unavailable capabilities` 只剩 `Attachments`。
- 主区：标题是 prompt 前缀；user 气泡 + assistant `Echo: ...` 行来自真实 daemon 的首屏 history；右上 `Session: Idle`、`● App Server: Connected`（绿点）。
- 状态栏：`Branch: main`、`Context: WIP`、`Model: WIP`、`Account: WIP`、`App Server: Connected`。
- composer 与 WIP 入口与未连接图一致。

WIP 卡片当时渲染出的文案（真实窗口内容，逐字抄录）：

- `Session keys`（仅未连接图）— `session.keys` — `helper exists; shell has no key browser callback`
- `Attachments` — `attachment.put` — `helper exists; shell has no file picker and keeps unknown upload results disabled`

两张图除连接状态外还差会话列表、history、branch 目录和 WIP 卡片数量，不能只用状态行解释 MD5 差异。

## 5. L4 判定

能证明的：

- **L4（窗口真实存在）**：`mygo.NewWindow` 在真实 macOS 桌面创建了 `DimAgent` 窗口，窗口服务器按 PID 归属本次启动的进程，截图来自窗口服务器本身，不是渲染器/单测产物。
- **连接状态的窗口化呈现**：不设 `DIMAGENT_PRODUCT_HOME` → `Not connected`；设临时 home 后 supervisor 真实 spawn `/tmp/dim-2-target/debug/dim-app-server` 并握手成功 → `Connected`。这是 `acceptance.md` F2「应用启动（真实 daemon）」的窗口侧证据。
- **F2 的 session 列表可见**：seed 后侧栏显示 `Sessions 1`、选中行和 branch 目录。
- **F5 的首屏 history 渲染**：user 气泡与 assistant `Echo: ...` 行来自真实 daemon 的 history。
- **F12 的 branch 目录展示**：`main (current)` 与 `session.branch_create` 产出的 side branch 同时可见，`Refresh` 可用。
- **WIP 入口在真实窗口里可见且带说明**：未连接图有 `Session keys` 和 `Attachments` 两张卡，已连接图只剩 `Attachments`；`Attach file` 的 WIP 标记也渲染出来，文案与当时源码一致。

不能证明的（这里不写成通过）：

- **F3 / F4 的窗口内点击路径未验证**：没有在真实窗口里点击 New session 或切换 session/branch，只看到了 seed 数据。
- **F6–F11、F13、F14 未验证**：没有真实窗口里的输入、submit、stop、approval、断线重连、附件上传或删除操作。
- **F12 的 cut point 未验证**：窗口里能看到 branch 目录，但没有在窗口内切换并渲染另一个 branch，也没有 cut point 入口。
- **L5（视觉/设计对照）未做**：本次只有 L4 窗口截图，没有与 Electron 设计稿逐项对照。
- 只覆盖浅色外观；深色外观、拖拽缩放、多显示器、菜单栏未采集。

## 6. 已知限制

- 采集需要终端具备「屏幕录制」权限；没有权限时 `screencapture` 会给出黑图或空文件，脚本会非零退出并报错，不会把空图当证据。
- 采集需要 `swiftc`（Xcode command line tools）读窗口 ID；缺失时脚本直接报错，不退回整屏截图，避免把桌面内容截进证据。
- 采集期间窗口会真实弹出并短暂获得焦点，约 2 秒后关闭。
- 窗口服务器报的 bounds 两次不完全相同（`1080×720` 与 `1076×716`），PNG 始终是 1080×720 points @2x；差异原因未查，不影响状态行比对。
- 已连接截图证明 seed 的会话列表、首屏 history 和 branch 目录，但没有触发窗口内交互或持续模型流，因此不覆盖 F4/F6 的点击与流式更新。
