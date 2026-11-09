# Electron 参考窗口截图（L5 对照用）

本目录保存 **dim-electron 真实窗口**的截图，作为 [`docs/l5-reference.md`](../../l5-reference.md) 的像素证据。截图由 `screencapture -x -o -l <windowid>` 按窗口 ID 截取，只含该窗口，不含桌面和其它应用。

- 采集时间：2026-11-09 14:30–14:37 CST（空态/设置）；2026-11-09 16:55–16:59 CST（对话态）
- 源码：dim-electron `HEAD = 3339f4fd47f7b0569e185f1a9d81b82d16447306`（未改动，详见 [`docs/l5-reference.md`](../../l5-reference.md)）
- 窗口：800×720 points @2x → PNG 1600×1440
- 运行环境：临时 `DIMCODE_HOME` 与临时 `DIMCODE_USER_DATA_PATH`，**未读写用户的 `~/.dimcode`**

## 文件

| 文件 | 状态 | 说明 |
|---|---|---|
| [`window-initial.png`](window-initial.png) | 首次启动 | 欢迎/登录浮层（背景被遮罩压暗） |
| [`window-workspace.png`](window-workspace.png) | 空工作区（浅色） | 侧栏 + 空状态主区 + composer |
| [`window-new-session.png`](window-new-session.png) | 点击「新建会话」后 | 侧栏仍是「暂无会话」 |
| [`window-skills.png`](window-skills.png) | 技能/插件页 | 技能商店 / 我的技能 |
| [`window-settings.png`](window-settings.png) | 设置 · 个性化 | 整窗替换为设置面 |
| [`window-appearance.png`](window-appearance.png) | 设置 · 外观（浅色） | 主题/语言/字体/字号/代码主题/皮肤 |
| [`window-settings-dark.png`](window-settings-dark.png) | 设置 · 外观（深色） | 同一页面切深色 |
| [`window-workspace-dark.png`](window-workspace-dark.png) | 空工作区（深色） | 与 `window-workspace.png` 同状态 |
| [`window-permissions.png`](window-permissions.png) | 设置 · 权限 | 进程隔离 + exec 命令规则 |
| [`window-conversation.png`](window-conversation.png) | 对话态（浅色） | 侧栏工作区分组 + 一轮 user/assistant + composer |
| [`window-conversation-dark.png`](window-conversation-dark.png) | 对话态（深色） | 与 `window-conversation.png` 同状态 |

每个 PNG 旁边的同名 `.txt` 是采集元数据（状态、动作、时间、app PID、窗口 ID、窗口 bounds、像素尺寸、字节数、MD5、临时 home 路径，对话态两张另有 dim-electron / mygo 的 git HEAD 与 seed 会话 ID）。

## 复现

驱动脚本在 [`scripts/`](scripts/)：

```bash
cd docs/evidence/electron-reference/scripts
swiftc -O winlist.swift -o /tmp/dim-electron-capture/winlist   # 读窗口 ID 的小工具
./capture-initial.sh window-initial
./capture-step.sh dismiss-onboarding window-workspace text:技能/插件 window-skills
```

对话态要先在**同一个临时 home** 里种一个会话，再让窗口打开它（`capture-conversation.sh` 不会清空 home）：

```bash
WORK=/tmp/dim-conversation-capture
mkdir -p "$WORK/home"
(cd /Users/eric8810/Code/dimagent-mygo && go run ./scripts/seed-session \
  -home "$WORK/home" -server /tmp/dim-2-target/debug/dim-app-server \
  -cwd /Users/eric8810/Code/dimagent-mygo \
  -prompt "Explain what this repository does in one paragraph." -side-branch)
cd docs/evidence/electron-reference/scripts
PRODUCT_HOME="$WORK/home" DIMCODE_USER_DATA_PATH="$WORK/userdata" \
  ./capture-conversation.sh window-conversation window-conversation-dark
```

`seed-session` 通过公开协议建会话并提交一轮输入，随后关掉自己的 daemon；`capture-conversation.sh` 用同一个 home 拉起 Electron，清空临时 userData（桌面 UI 状态会持久化，见 `docs/l5-reference.md` §7.7），依次切浅色、点侧栏会话行、切深色后截图。

脚本只结束自己启动的 Electron PID，以及命令行里含本次临时 home 的 `dim-app-server`。完整命令、环境变量和失败记录见 [`docs/l5-reference.md`](../../l5-reference.md)。

## 证据边界

- 这些图证明**这台机器、这个 commit、这个窗口尺寸**下的真实渲染结果，不证明其它平台、DPI、字体或系统外观下的样子。
- 9 张空态/设置图里会话列表为空；2 张对话态图里只有 `seed-session` 种进 App Server 的**一轮**输入 + mock 模型的 `Echo: …` 回复（原因和边界见 `docs/l5-reference.md` §5.9–§5.11）。两组图都不能用来证明真实模型回答、多轮滚动、工具卡片、流式更新或审批态。
