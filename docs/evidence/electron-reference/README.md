# Electron 参考窗口截图（L5 对照用）

本目录保存 **dim-electron 真实窗口**的截图，作为 [`docs/l5-reference.md`](../../l5-reference.md) 的像素证据。截图由 `screencapture -x -o -l <windowid>` 按窗口 ID 截取，只含该窗口，不含桌面和其它应用。

- 采集时间：2026-11-09 14:30–14:37 CST（空态/设置）；2026-11-09 16:55–16:59 CST（对话态）；2026-11-09 17:44–17:47 CST（断线态）；2026-11-09 21:47 CST（设置 · 外观）；2026-11-09 22:02 CST（搜索浮层 / 设置 route·返回）；2026-11-10 09:42–09:50 CST（hover / focus / menu / loading / motion / approval）
- 源码：dim-electron `HEAD = 3339f4fd47f7b0569e185f1a9d81b82d16447306`（未改动，详见 [`docs/l5-reference.md`](../../l5-reference.md)）
- 窗口：800×720 points @2x → PNG 1600×1440（窗口落在 1x 显示器时 PNG 为 800×720，见证据边界）
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
| [`window-error.png`](window-error.png) | 断线态 | 杀掉本次 daemon 后 6 s；与对话态只差侧栏相对时间 |
| [`window-error-send.png`](window-error-send.png) | 断线态 + 已输入草稿 | 点发送无反应、无错误提示；与 daemon 存活时的同状态截图逐字节相同 |
| [`window-error-recovery.png`](window-error-recovery.png) | 断线后点「新建会话」 | 新 daemon 被静默拉起，窗口切到新会话空态，仍无提示 |
| [`window-search.png`](window-search.png) | 搜索浮层 | 侧栏「搜索」打开居中 Dialog：搜索框 + 「最近聊天」2 行（首行选中、⌘1/⌘2），背景 80% 黑遮罩 |
| [`window-search-results.png`](window-search-results.png) | 搜索过滤 | 输入 `Explain` 后只剩 1 行「Explain what this repository…」，选中态仍为行内 `bg-muted` |
| [`window-settings-route.png`](window-settings-route.png) | 设置路由 `#/settings` | 聊天侧栏被设置导航替换（返回 + 20 项），内容区默认「个性化」，无顶栏 |
| [`window-settings-back.png`](window-settings-back.png) | 从设置返回 `#/chat` | 原会话「Draft a release checklist…」时间线与 composer 都在，侧栏两个会话恢复 |
| [`window-settings-appearance.png`](window-settings-appearance.png) | 设置 · 外观 | 主题卡片/语言下拉/字体 combobox/UI 字号分段/代码字号滑杆/代码主题下拉 + 预览（appearance 采集产出） |
| [`window-settings-appearance-2.png`](window-settings-appearance-2.png) | 设置 · 外观（滚到底） | 代码主题预览与「我的皮肤」卡片（appearance 采集产出） |

每个 PNG 旁边的同名 `.txt` 是采集元数据（状态、动作、时间、app PID、窗口 ID、窗口 bounds、像素尺寸、字节数、MD5、临时 home 路径，对话态两张另有 dim-electron / mygo 的 git HEAD 与 seed 会话 ID）。断线态三张额外记录被杀 daemon 的 PID/命令行、kill 信号与时间、kill 后是否还有 daemon；`window-error-send.txt` 还记录发送探针，`window-error-recovery.txt` 记录 post-loss 动作与新 daemon PID。

搜索/设置四张另有 CDP 抓的 DOM 快照：同名 `.json`（`window-search.json`、`window-search-results.json`、`window-settings-route.json`、`window-settings-back.json`）保存命中节点的 rect、class、testid、route；`window-search.txt` 末尾追加 `interaction_*` 行，`window-search-interactions.json` 保存 ⌘G 打开、Esc 关闭、空态和点遮罩关闭的原始结果。

## 状态 / 动效 / 加载 / 审批（2026-11-10）

同一批 run 使用的仍是 dim-electron `HEAD = 3339f4fd47f7b0569e185f1a9d81b82d16447306` 的已编 dist，窗口 800×720 points @2x（PNG 1600×1440），深色，界面文案 zh-CN。窗口、renderer、preload、main 都是真实的；三类受控输入分别在表中注明。

| 文件 | 状态 | 说明 |
|---|---|---|
| [`window-hover.png`](window-hover.png) | hover | 指针停在侧栏第 2 个会话行；行底色变为 `hover:bg-sidebar-accent/80`，相对时间被左侧 pin（`sidebar-session-pin`）和右侧归档（`sidebar-session-archive`，aria-label「归档会话」）替换。没有出现 tooltip。 |
| [`window-focus.png`](window-focus.png) | focus | 点击 composer 编辑器后按一次 Tab，焦点落在 composer 附件「+」按钮（`data-slot=tooltip-trigger`，rect 25×25，`:focus-visible` 为 true）；3px ring/50 ring 可见，tooltip「添加图片或文件」同时打开。 |
| [`window-menu.png`](window-menu.png) | menu | 右键侧栏会话行打开的会话 context menu：置顶 / 重命名 / 在新窗口中打开 / 导出 / 归档 / 删除；截图时重命名项是 disabled，标签为「重命名（正在检查 SessionTitle…）」（`data-rename-readiness=checking`），菜单项标签与状态以 `dom` 行为准。 |
| [`window-loading.png`](window-loading.png) | loading（restore skeleton） | 恢复上次 active session 时，主区显示 `data-testid=timeline-skeleton`（品牌 mark，无文字）＋ 左下「正在加载...」，侧栏 active 行已选中。延迟来自 `app-server-delay-proxy.mjs` 对一次 `session.history` 响应注入 2500 ms；app 和 daemon 未改。 |
| [`window-loading-session-switch.png`](window-loading-session-switch.png) | loading（自然行为对照） | 对本次 run 的 daemon 发 SIGSTOP 后点另一个会话行并等 0.3 s：窗口仍显示原会话，没有 loading 文案、spinner、skeleton 或 `role=status`；renderer 要等 `session.open` 完成才切换 active session。 |
| [`motion-search-000.png`](motion-search-000.png) [`-075`](motion-search-075.png) [`-150`](motion-search-150.png) [`-250`](motion-search-250.png) | motion（打开） | 侧栏「搜索」打开 Dialog 的 0/75/150/250 ms 帧；0 ms 时遮罩和 Dialog 都还透明，75 ms 半透明，150 ms 已到 Dialog 200 ms 动画的 75%，250 ms 动画已结束、Dialog 完全打开。 |
| [`motion-search-close-000.png`](motion-search-close-000.png) [`-075`](motion-search-close-075.png) [`-150`](motion-search-close-150.png) [`-250`](motion-search-close-250.png) | motion（关闭） | 按 Esc 触发 exit 的 0/75/150/250 ms 帧；250 ms 时 Dialog 已从 DOM 移除，只剩会话。 |
| [`motion-search-frames.json`](motion-search-frames.json) | 动效原始数据 | 每帧的目标偏移、截图前后实际 `Animation.currentTime`、computed opacity/transform、PNG 字节与 MD5；`openTiming`/`closeTiming` 记录 enter/exit 的 duration 与 easing。 |
| [`motion-search.txt`](motion-search.txt) | 动效元数据 | 采集方法、measured_motion、环境与限制。 |
| [`window-approval.png`](window-approval.png) | approval（fixture 后端） | 真实窗口里的命令审批卡：DimAgent 想要执行 `$ cargo test -p demo`，三个选项「拒绝 / 仅这次 / 始终允许此命令」，侧栏行带「等待操作」tag。后端是 `electron-main/tests/fixtures/fake-app-server.mjs`（`FAKE_APP_SERVER_COMMAND_APPROVAL=1`），**不是**真实 `dim-app-server`。 |
| [`window-approval.json`](window-approval.json) | approval 原始数据 | fixture 路径、run 目录、窗口行、PNG 和 DOM 事实。 |

这些 PNG 旁边的 `.txt`（`window-approval` 也有同名 `.json`）记录状态、动作、时间、app PID、窗口 ID/bounds、像素尺寸、字节数、MD5、临时 home/userData、dim-electron/mygo HEAD 与 seed prompt；`dom` 行是当时 CDP 读到的 theme、rows、skeleton、focused、tooltip、menu/prompt 节点事实。`motion-search*` 的帧元数据统一放在 `motion-search-frames.json`。

## 文件归属

- 本目录的 **search / settings-route / settings-back** 四组（`window-search*`、`window-settings-route*`、`window-settings-back*`、`window-search-interactions.json`）连同 [`scripts/capture-search-settings.sh`](scripts/capture-search-settings.sh)、[`scripts/interact-search-settings.mjs`](scripts/interact-search-settings.mjs) 和本 README 由搜索/路由采集维护。
- **settings · appearance** 两组（`window-settings-appearance*`、`window-settings-appearance-2*`）由 appearance 采集用同一脚本的 `APPEARANCE_ONLY=1` 分支产出（21:47，独立 seed home `/tmp/dim-appearance-capture/home`，单会话）；本 README 只做索引，默认流程不会再写这两个文件名。
- **状态 / 动效 / loading / approval** 一组（`window-hover*`、`window-focus*`、`window-menu*`、`window-loading*`、`motion-search*`、`window-approval*`）连同 [`scripts/capture-states.sh`](scripts/capture-states.sh)、[`scripts/interact-states.mjs`](scripts/interact-states.mjs)、[`scripts/capture-loading-state.sh`](scripts/capture-loading-state.sh)、[`scripts/app-server-delay-proxy.mjs`](scripts/app-server-delay-proxy.mjs)、[`scripts/capture-approval-fixture.mjs`](scripts/capture-approval-fixture.mjs) 由状态采维护。

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

断线态复用同一个 seed 后的 home，在进入对话后**只杀本次 daemon**（按临时 `PRODUCT_HOME` 匹配命令行），等 4 s 再抓窗口：

```bash
cd docs/evidence/electron-reference/scripts
WORK=/tmp/dim-error-capture   # 与上面 seed 用的 home 相同
PRODUCT_HOME="$WORK/home" DIMCODE_USER_DATA_PATH="$WORK/userdata" \
  ELECTRON_ERROR_WORK="$WORK/loss" ./capture-error.sh window-error
PRODUCT_HOME="$WORK/home" DIMCODE_USER_DATA_PATH="$WORK/loss-send/userdata" \
  ELECTRON_ERROR_WORK="$WORK/loss-send" SEND_PROBE=1 ./capture-error.sh window-error-send
PRODUCT_HOME="$WORK/home" DIMCODE_USER_DATA_PATH="$WORK/loss-recovery/userdata" \
  ELECTRON_ERROR_WORK="$WORK/loss-recovery" POST_LOSS_ACTION=new-session \
  ./capture-error.sh window-error-recovery
```

`SKIP_KILL=1` 是 daemon 存活时的对照（控制变量），`EVIDENCE_OUT_DIR` 可把产物写到 `/tmp`。脚本只结束自己启动的 Electron PID，以及命令行里含本次临时 home 的 `dim-app-server`。完整命令、环境变量和失败记录见 [`docs/l5-reference.md`](../../l5-reference.md)。

搜索浮层与设置路由复用同一个 seed 后的 home（两个会话，搜索才有过滤可看）：

```bash
WORK=/tmp/dim-search-settings-capture
mkdir -p "$WORK/home"
(cd /Users/eric8810/Code/dimagent-mygo && go run ./scripts/seed-session \
  -home "$WORK/home" -server /tmp/dim-2-target/debug/dim-app-server \
  -cwd /Users/eric8810/Code/dimagent-mygo \
  -prompt "Draft a release checklist for the desktop client.")
(cd /Users/eric8810/Code/dimagent-mygo && go run ./scripts/seed-session \
  -home "$WORK/home" -server /tmp/dim-2-target/debug/dim-app-server \
  -cwd /Users/eric8810/Code/dimagent-mygo \
  -prompt "Explain what this repository does in one paragraph." -side-branch)
cd docs/evidence/electron-reference/scripts
PRODUCT_HOME="$WORK/home" DIMCODE_USER_DATA_PATH="$WORK/userdata" \
  SEARCH_QUERY=Explain SEED_PROMPT="Draft a release checklist for the desktop client." \
  ./capture-search-settings.sh
```

`capture-search-settings.sh` 等窗口出现后先等 bounds 连续 3 次不变再开始驱动（窗口启动后会被重新摆放、DPR 可能从 1x 变 2x），然后：点侧栏会话行 → hover「搜索」→ 点「搜索」→ 截图 → 输入 `SEARCH_QUERY` → 截图 → Esc → ⌘G 再开 / 空态 / 点遮罩关闭（只记事实）→ 点左下齿轮 → 截图 → 点「外观」→ 点「返回」→ 截图。`SEARCH_QUERY`、`CDP_PORT`、`APPEARANCE_ONLY`、`APPEARANCE_SHOT` 是脚本的环境变量开关；设置 · 外观只在 `APPEARANCE_ONLY=1`（或 `APPEARANCE_SHOT=1`）时才截图，避免覆盖 appearance 采集的产物。

状态 / 动效这组复用同一个 2 会话 seed home（命令同上面的搜索/设置），按顺序跑：

```bash
WORK=/tmp/dim-state-capture
cd docs/evidence/electron-reference/scripts
PRODUCT_HOME="$WORK/home" DIMCODE_USER_DATA_PATH="$WORK/userdata" \
  SEED_PROMPT="Draft a release checklist... / Explain what this repository..." \
  ./capture-states.sh

# capture-states.sh 先切深色并停在对话态；它的 userData 留下了 lastActiveSessionId，
# capture-loading-state.sh 复制这份 userData 后用延迟代理抓 restore skeleton。
PRODUCT_HOME="$WORK/home" DIMCODE_USER_DATA_PATH="$WORK/userdata" \
  ./capture-loading-state.sh

# approval 不需要 seed home：用仓库自带 fake app-server fixture 触发命令审批。
node capture-approval-fixture.mjs "$PWD/.." /tmp/dim-approval-capture
```

`capture-states.sh` 的驱动是 `interact-states.mjs`：hover 用 CDP `Input.dispatchMouseEvent`，focus 用真实 Tab 键，menu 用右键，loading-session-switch 在 SIGSTOP 本 run 的 daemon 后点行。`motion` 用 CDP `Animation.setPlaybackRate(0.02)` 放慢同一组 CSS 动画，再用 `screencapture` 按 0/75/150/250 ms 的目标墙钟偏移截帧；帧里实际动画毫秒数看 `motion-search-frames.json`。`capture-loading-state.sh` 通过 [`scripts/app-server-delay-proxy.mjs`](scripts/app-server-delay-proxy.mjs) 只延迟一次 `session.history` 响应；`DIM_PROXY_DELAY_MS`、`DIM_PROXY_DELAY_METHODS`、`DIM_PROXY_DELAY_ONCE` 是它的开关。`capture-approval-fixture.mjs` 用 `scripts/e2e/electron-driver.mjs` 启动真实窗口并指向 `electron-main/tests/fixtures/fake-app-server.mjs`。

## 证据边界

- 这些图证明**这台机器、这个 commit、这个窗口尺寸**下的真实渲染结果，不证明其它平台、DPI、字体或系统外观下的样子。
- 9 张空态/设置图（14:30–14:37）里会话列表为空；2 张对话态图里只有 `seed-session` 种进 App Server 的**一轮**输入 + mock 模型的 `Echo: …` 回复（原因和边界见 `docs/l5-reference.md` §5.9–§5.11）。两组图都不能用来证明真实模型回答、多轮滚动、工具卡片、流式更新或审批态。
- 3 张断线图只证明"本次 daemon 被杀后窗口长什么样"：没有可见错误 UI，输入框仍可编辑，点发送无反应，下一次用户动作会静默拉起新 daemon。它们不证明其它错误类型（模型/工具/提交失败）、已配置模型下的发送路径或自动重连的确定性；细节见 `docs/l5-reference.md` §5.12。
- 搜索/设置四张来自**一次** run（同一 `window_id`、同一个 2 会话 seed home、`PNG 1600×1440`）；搜索浮层不改变 route（仍是 `#/chat`），打开结果也是 `openThread`，所以「搜索是否换页」这个问题的答案是不换。没有置顶会话，因此看不到「置顶聊天」分组；⌘1–⌘8 的实际跳转与真实模型/多轮搜索都没验证。
- 设置路由：齿轮进入后 URL 是 `#/settings`（section 不写进 URL），内容默认「个性化」；只有点「外观」才 `router.replace` 成 `#/settings/appearance`。返回是 `[data-testid=settings-back]` → `#/chat` 并 `selectSession` 恢复原会话；紧凑布局下才走 `settings-detail-back` 与「返回列表」两级行为，本窗口（800×720）未触发紧凑布局。
- 搜索的交互事实（`⌘G` 能打开同一浮层、Esc 与点遮罩都能关闭、关闭后焦点回到打开前的元素、空查询文案是「未找到聊天」、结果行 42px 带 ⌘1…⌘8 序号）与两张图同源记录在 `window-search.txt` 的 `interaction_*` 行和 `window-search-interactions.json`；置顶分组没测到（没有置顶会话），⌘n 的实际跳转、↑↓ 移动选中（只在一次性探针里验过，未落盘）和真实模型/多轮搜索都没验证。
- 设置 · 外观两张来自**另一次** run（21:47，独立 home、单会话），因此侧栏会话数与四张不同；它们的控件清单在 `window-settings-appearance.txt` 里，不要拿它们和四张做像素级对照。
- 截图分辨率取决于窗口所在显示器：同一脚本在 1x 显示器上会产出 800×720 的 PNG，2x 上是 1600×1440（21:47 的中间产物是 1x，已被 2x 版替换掉同名文件）。窗口启动后会被重新摆放（本次从 `240,30` 移到 `624,152`），脚本因此先等 bounds 稳定，避免一组图混用两种 DPR。
- 状态四张（hover / focus / menu / loading-session-switch）来自**同一次** run（同一 `window_id`、同一个 2 会话 seed home，深色）；loading 的 skeleton 是**另一次** run（复制该 userData、延迟代理），approval 也来自独立 run（fixture 后端），不要拿它们的侧栏内容或窗口位置做像素级对照。hover 用的是 CDP `Input.dispatchMouseEvent`，不是 OS 级 CGEvent；focus 是真实 Tab 键，ring 与 tooltip 同时可见；menu 只打开菜单，没有激活任何菜单项，重命名 readiness 在截图时仍是 `checking`（`dom` 行有原值）。
- loading 的两种事实要分开读：自然 SIGSTOP 后点另一个会话行**没有**任何 loading UI（`window-loading-session-switch`），renderer 要等 `session.open` 完成才切 active session；`window-loading.png` 的 skeleton 是延迟代理注入一次 2500 ms `session.history` 响应后抓到的 restore 态，证明 renderer 确实有 `timeline-skeleton` 和「正在加载...」加载面，也证明它会在延迟消失后消失。该代理 run 在 skeleton 之后回落到了 new-thread 屏（`dom.recovered`），所以这张图不证明正常 restore 的时延或成功路径；同一 userData 不走代理的正常启动会恢复会话和时间线。
- motion 帧是 CDP `Animation.setPlaybackRate(0.02)` 下的同一组 CSS 动画，再由 OS 窗口截图截取；墙钟偏移只负责触发，真实动画时间看 `motion-search-frames.json` 的 `actualAnimationMs`（0 ms 帧约 8–10 ms，75 帧约 74–77 ms，150 帧约 149–152 ms）。测得的打开动画是 overlay `enter` 150 ms、Dialog `enter` 200 ms，keyframe easing 为 `ease`（computed animation 显示 `linear`，因为 easing 写在 keyframe 上），250 ms 时动画已结束（Dialog 仍打开）；关闭的 250 ms 帧 Dialog 已移除。没有采面板切换或其它动效。
- approval 是 **fixture 后端**的审批面：窗口/ renderer / preload / main 是真的，daemon 是 `electron-main/tests/fixtures/fake-app-server.mjs`，审批请求和 `$ cargo test -p demo` 来自 `FAKE_APP_SERVER_COMMAND_APPROVAL=1`。生产 mock model（`dim-app-server --mock-model` / `EchoModel`）只回文本、不调用工具，本机也没有真实 provider/model 配置，因此**没有**采到真实 daemon 的 approval round trip；这张图只证明审批卡/选项的渲染，不证明点击 Approve/Deny 后的 `approval.respond`、权限规则持久化或真实模型工具调用。
- 这一组的界面文案是 zh-CN，但 `document.documentElement.lang` 仍是 `en`（既有 renderer 行为，未改）；所有图只证明这台机器、这个 commit、这个窗口尺寸和深色主题下的渲染。
