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
| 源码 | `HEAD = 3fd370b6dd0aac6cfec54b706fe16023427cbfee`，**工作区不干净**（见下） |
| 构建时间 | 2026-11-09 12:10:50 CST |
| App Server 二进制 | `/tmp/dim-2-target/debug/dim-app-server`（Mach-O arm64，401 302 448 B，mtime 2026-11-08 23:38） |
| 窗口库 | `github.com/egoist/mygo v0.8.3`（`mygo.NewWindow`，`Width: 1080`、`Height: 720`） |
| 截图工具 | `/usr/sbin/screencapture -x -o -l <windowid>`（按窗口 ID 截取，只含该窗口，不含桌面和其它应用） |
| 窗口 ID 来源 | `CGWindowListCopyWindowInfo`，按**本进程 PID** 过滤；脚本内用 `swiftc` 现场编译小工具读取 |

采集时 `git status --short` 里已有其他人的未提交改动：`internal/ui/capability.go`、`internal/session/session.go`、`internal/integration/integration.go`、`internal/appserver/*`、`docs/protocol-contract.md`，另有未跟踪的 `internal/appserver/attachment.go`、`branch_test.go` 等。**窗口文字由这些未提交源码决定**，源码再变时截图即过期，需要重跑脚本。每个状态的完整快照记在同名 `.txt` 元数据的 `git_head` / `git_dirty` 字段里。

## 2. 复现命令

```bash
cd /Users/eric8810/Code/dimagent-mygo
./scripts/capture-window.sh                     # 两种状态都截
./scripts/capture-window.sh --state connected   # 只截已连接
```

脚本行为（与本次运行一致）：

1. `go build -o $TMP/dimagent ./cmd/dimagent`，不改动仓库内文件。
2. 未连接态：`env DIMAGENT_PRODUCT_HOME=` 启动，窗口必须显示 `App Server: Not connected`。
3. 已连接态：`DIMAGENT_PRODUCT_HOME=$(mktemp -d)`，`PATH` 前置 `/tmp/dim-2-target/debug`，由 supervisor 自行 spawn 真实 `dim-app-server`；窗口必须显示 `App Server: Connected`。
4. 两种状态都在窗口拿到真实尺寸后 `sleep 1` 再截图；截图后按 PID 结束应用，并按「命令行里含本次临时 home 路径」精确结束本次 spawn 的 daemon；随后 `rm -rf` 临时 home。
5. 全程不读写 `~/.dimcode`，不使用用户的真实 product home。结束时打印 `all requested captures done; no leftover daemon from this run`，并复查没有本次残留的 daemon。

## 3. 采集结果（2026-11-09 12:10 CST）

| 状态 | 采集时间 | 窗口 ID | 窗口位置/尺寸（points） | PNG 像素 | 字节 | PNG 路径 |
|---|---|---|---|---|---|---|
| 未连接 | 12:10:50 → 12:10:51 | 47309 | `486,118,1076,716` | 2160×1440（= 1080×720 @2x） | 195 357 | [`evidence/window-disconnected.png`](evidence/window-disconnected.png) |
| 已连接 | 12:10:52 → 12:10:54 | 47318 | `484,116,1080,720` | 2160×1440（= 1080×720 @2x） | 195 551 | [`evidence/window-connected.png`](evidence/window-connected.png) |

MD5：未连接 `afe8e1f72f381a4e3c643c7f7def6341`，已连接 `2f789c447cf38d1acc3901ddc94daca8`。

机器可读的采集元数据（脚本生成，含源码快照、PID、窗口 ID、bounds、daemon 命令行）：
[`evidence/window-disconnected.txt`](evidence/window-disconnected.txt)、[`evidence/window-connected.txt`](evidence/window-connected.txt)。

已连接态截图时真实在跑的 daemon（取自元数据）：

```
30243 /tmp/dim-2-target/debug/dim-app-server
      --unix <tmp>/dimagent-product-home.SZRCzg/rust/daemon.sock --no-ws
      --home <tmp>/dimagent-product-home.SZRCzg/rust
      --product-home <tmp>/dimagent-product-home.SZRCzg
```

截图结束后该进程随应用退出而消失，`ps aux | grep '[d]im-app-server'` 无输出；临时 home 已删除。

## 4. 截图上实际看到的内容

两张图版式一致：左侧边栏顶部 `New session`，右侧标题 `No session`，右上状态行 `● Session: Idle  ● App Server: <状态>`；中间 `No messages yet` / `The conversation fills in when the App Server projection arrives.`；底部 composer（`Message DimAgent` + `Send`，左侧 `Attach file` 带 `WIP` 标记）；左下 `Unavailable capabilities` 三条 WIP 卡片。

| 差异点 | 未连接（`window-disconnected.png`） | 已连接（`window-connected.png`） |
|---|---|---|
| 状态行 | `● App Server: Not connected`（灰点） | `● App Server: Connected`（绿点） |
| 侧栏 session 列表 | 空 | 空 |
| 标题 | `No session` | `No session` |

两张图其余像素一致，差异只在状态行的文字和圆点颜色（这也是 MD5 不同的唯一来源）。

WIP 卡片当时渲染出的文案（真实窗口内容，逐字抄录）：

- `Branch switching` — `session.branches` — `helper exists; the directory is read-only and the shell has no branch selection yet (switching is a history read, not an RPC)`
- `Session keys` — `session.keys` — `read helper exists; shell shows no key browser yet`
- `Attachments` — `attachment.put` — `no Go client helper yet`

## 5. L4 判定

能证明的：

- **L4（窗口真实存在）**：`mygo.NewWindow` 在真实 macOS 桌面创建了 `DimAgent` 窗口，窗口服务器按 PID 归属本次启动的进程，截图来自窗口服务器本身，不是渲染器/单测产物。
- **连接状态的窗口化呈现**：不设 `DIMAGENT_PRODUCT_HOME` → `Not connected`；设临时 home 后 supervisor 真实 spawn `/tmp/dim-2-target/debug/dim-app-server` 并握手成功 → `Connected`。这是 `acceptance.md` F2「应用启动（真实 daemon）」的窗口侧证据。
- **WIP 入口在真实窗口里可见且带说明**：三条 `Unavailable capabilities` 与 `Attach file` 的 WIP 标记确实渲染出来，文案与当时源码一致。

不能证明的（这里不写成通过）：

- **F2 要求的「session 列表可见」未证明**：临时 home 是全新的，catalog 为空，截图里侧栏为空。要让列表非空需要先创建 session（属 F3 的点击路径），本次未执行。
- **F3–F14 全部未验证**：没有真实窗口里的点击、输入、submit、stop、approval、断线重连，因此不能据此改动 `acceptance.md` 的任何行。
- **L5（视觉/设计对照）未做**：本次只有 L4 窗口截图，没有与 Electron 设计稿逐项对照。
- 只覆盖浅色外观；深色外观、拖拽缩放、多显示器、菜单栏未采集。

## 6. 已知限制

- 采集需要终端具备「屏幕录制」权限；没有权限时 `screencapture` 会给出黑图或空文件，脚本会非零退出并报错，不会把空图当证据。
- 采集需要 `swiftc`（Xcode command line tools）读窗口 ID；缺失时脚本直接报错，不退回整屏截图，避免把桌面内容截进证据。
- 采集期间窗口会真实弹出并短暂获得焦点，约 2 秒后关闭。
- 窗口服务器报的 bounds 两次不完全相同（`1080×720` 与 `1076×716`），PNG 始终是 1080×720 points @2x；差异原因未查，不影响状态行比对。
- 已连接只到握手 + 空 catalog，未触发任何模型调用，因此不覆盖 F6 的流式更新。
