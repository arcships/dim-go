# 产品验收矩阵

Status: active（F1–F5、F10、F12a、F14a、A1、A4 已达 L4；D1–D7、其余 F/A 未通过）
Last updated: 2026-11-09

本文回答一个具体问题：**什么才算验收通过**。它和 `go test` 通过是两件事。

## 证据等级

从低到高，低等级证据不能替代高等级：

| 等级 | 含义 | 现状 |
|---|---|---|
| L0 `source-read` | 只读了公开合同/源码，没有运行 | 协议合同 |
| L1 `headless` | Go 测试直接驱动 view tree，不打开窗口 | `internal/ui`、`internal/session` |
| L2 `fake-server` / `fake-daemon` | 进程内 fake，不是真实 daemon | 大部分流/mutation 测试 |
| L3 `real-daemon` | 真实 `dim-app-server` 进程，mock model | 控制面 smoke、session 流 |
| L4 `real-window` | 真实窗口 + 人工/截图对照 | F1/F2/F5 已执行；其余未执行 |
| L5 `design-parity` | token/组件/视觉与 Electron 端逐项对照 | **未执行** |

验收通过 = 目标行达到 L4，且视觉相关行同时达到 L5；L1–L3 只能写成"测试级证据"。

## 功能验收

| # | 项目 | 要求证据 | 当前 | 结论 |
|---|---|---|---|---|
| F1 | 应用启动（无 daemon） | 真实窗口截图 + 未连接状态可见 | L4：`docs/evidence/window-disconnected.png`（`Not connected`） | **已通过（L4）** |
| F2 | 应用启动（真实 daemon，`DIMAGENT_PRODUCT_HOME`） | L3 + 真实窗口 + session 列表可见 | L4：`docs/evidence/window-connected.png` + `scripts/seed-session` 预置会话后，窗口显示 `Sessions 1`、选中行、`main (current)`、side branch、`App Server: Connected`（`61a52a7`） | **已通过（L4）** |
| F3 | session 创建 | 真实窗口点击 → 列表出现新 session（App Server 返回） | L4：窗口点击 `New session`，`Sessions 2` 且新会话被选中（`window-create-session.png`） | **已通过（L4）** |
| F4 | session 切换 | 真实窗口切换 + 历史正确 | L4：窗口点击第二个会话，标题与首屏 history 均切换（`window-switch-session.png`） | **已通过（L4）** |
| F5 | 首屏 history | 真实 daemon 内容在窗口渲染 | L4：`window-connected.png`（seed 会话的 user/assistant 行，`61a52a7`） | **已通过（L4）** |
| F6 | submit / 流式更新 | 真实窗口输入 → 发送 → 增量出现 | L4：窗口输入并点击 Send，user 行与 assistant 行出现（`window-submit.png`）；增量流式未单独观察 | 未通过 |
| F7 | stop | 真实窗口停止 → 状态收敛 | L3 通过；L4 无 | 未通过 |
| F8 | approval 决策 | 真实窗口点击 Approve/Deny → 行消失、无重复发送 | headless + fake-server（bridge 单测）；无 L3/L4 审批流证据 | 未通过 |
| F9 | unknown-result 读回 | submit 超时/断线 → `input_lookup` 结果可见 | L4：daemon 在 Send 前后被停/杀，窗口显示 `Submission result unknown`、保留草稿、提示 Reconnect，不重发（`window-unknown-result.png`）；`input_lookup` 的 accepted/unknown 结果展示仍缺 | 未通过 |
| F10 | 断线 / reattach | 杀掉 daemon → 错误态 → 重连 → 历史重建 | L4：kill daemon 显示 Connection error；点击 Reconnect 后 daemon 重新 spawn、连接恢复、latest-turn history 重建（`window-transport-lost.png`、`window-reattach.png`） | **已通过（L4）** |
| F11 | `subscription_changed` 恢复 | 真实 replay-gap 场景 | 无真实 wire 证据 | 未通过 |
| F12a | branch switching | 真实窗口点击 branch 并渲染该 branch 的 history | L4：点击 side branch 后状态栏显示该 branch、时间线切到其 seed 回合（`window-branch-switch.png`） | **已通过（L4）** |
| F12b | cut point | 选择父分支 sequence 创建 side line | L4：user 行 Branch from here → 输入 prompt → `session.branch_create` → 新分支被选中并渲染（`window-branch-from.png`） | **已通过（L4）** |
| F13 | attachment 上传 | 真实上传并出现在 timeline | Go 侧 `attachment.put` helper 已有（L1/L2 fake-server）；shell 无文件选择器/结果读回；L3/L4 无 | 未通过 |
| F14a | session 删除 | 真实窗口删除 → App Server 确认后行消失 | L4：`session.delete` 结果因流关闭判 unknown，按合同读回 `session.catalog` 确认后移除；删到最后一个清空为 `Sessions 0`（`window-delete-session.png`） | **已通过（L4）** |
| F14b | 关闭终态显示 | `session.closed` 在窗口可见且不可再提交 | L4：确认删除后窗口保留 `Session closed: Deleted` banner、侧栏行移除、composer/Stop/Reconnect 禁用（`window-closed.png`） | **已通过（L4）** |

## 设计 1:1 验收（样式 / 内容 / 动画）

> 复核方式：先按 `docs/ui-parity-review.md` 逐区域打开原图确认，再看 `scripts/parity_diff.py` 的数字。**只有脚本数字达标、没有视觉复核记录，不算通过；反之也一样。**

判定分三条轴，三条轴都满足才算通过。**token 映射表只是输入，不是通过证据**：样式必须在真实界面上与参考一致。

| 轴 | 通过条件 | 证据门槛 |
|---|---|---|
| S 样式 | 颜色、表面、描边、字体、圆角、间距、图标、阴影、状态色在真实界面与参考一致 | 同窗口尺寸/同状态/同主题截图；逐区域 MAE ≤ 3（0–255）且 delta>16 的像素占比 ≤ 2%，文本区扣除文字差异后同样达标 |
| C 内容 | 同状态下文案、元素顺序、可见性、数据格式一致；Markdown/代码/工具卡内容一致 | 同语言截图 + 文本快照对照；差异必须逐条列出并有理由 |
| M 动画 | hover、面板开合、主题切换、composer 展开、流式文本、折叠动画的时间线与缓动一致 | 帧序列/录屏或可重复的时长与曲线记录；不接受“瞬间重绘” |

| # | 轴 | 项目 | 要求证据 | 当前 | 结论 |
|---|---|---|---|---|---|
| D1 | S | 颜色/表面/描边 | 逐区域视觉复核 + 像素差 | 复核：`docs/ui-parity-review.md`；同尺寸基线 sidebar MAE 19.5/25.97%（空态）、24.97/26.87%（对话），top 164/77% 主要来自原生标题栏；`docs/ui-token-map.md` 只是 L0 映射 | 未通过 |
| D2 | S | 排版/间距/圆角/密度 | 字号/行高/内距/圆角逐项测量 | A 根字号 14px、圆角按 rem 得 4.75/6.75/8.75/12.75、间距 10.5 刻度；B 是 6/8/10/14 与 `Space(3)=12`；顶栏 41 目标实测 46 | 未通过 |
| D3 | S | 图标/阴影/状态色 | 图标集、按钮形态、hover/focus/disabled 对照 | composer 工具栏仍是文字 WIP；阴影/过渡层缺失；状态色未逐项取样 | 未通过 |
| D4 | S | 主题/皮肤/chrome | 主题数量、切换行为、skin/chrome 覆盖 | system/light/dark 可用；6 个 chrome 与 skin 未实现 | 未通过 |
| D5 | C | 元素内容与 Markdown | 同状态文案/顺序/可见性/数据格式 + Markdown 渲染 | 语言默认英文、WIP 徽标与额外区域（Branches/能力卡片）改变内容与顺序；Electron 对话基线只有一轮纯文本，无 Markdown 对照 | 未通过 |
| D6 | M | 动画/过渡 | 帧序列或时长/曲线记录 | MyGo 已实现 150ms `cubic-bezier(0.23,1,0.32,1)` / 250ms `cubic-bezier(0.4,0,0.2,1)`，帧在 `docs/evidence/ui/motion-*.png`，测试断言中间帧；参考侧尚无同操作帧序列，且 mygo 不过渡文字 ink | 未通过 |
| D7 | S+C | 状态覆盖 | 空/加载/错误/悬停/焦点/菜单六态截图 | 空态/错误态有并排；hover 有帧证据；加载、焦点、菜单未逐一采集 | 未通过 |

首轮像素基线（`scripts/parity_diff.py`；区域：sidebar/top/content/composer/bottom）：

| 对照 | sidebar MAE / >16% | top | content | composer | bottom |
|---|---|---|---|---|---|
| 空态：Electron workspace ↔ MyGo disconnected | 15.0 / 25.7% | 4.8 / 17.5% | 6.6 / 6.7% | 9.0 / 8.2% | 11.2 / 21.8% |
| 对话态：Electron conversation ↔ MyGo conversation | 20.3 / 25.0% | 15.9 / 24.2% | 6.4 / 4.9% | 10.5 / 11.0% | 12.3 / 22.9% |

diff 图存于 `docs/evidence/ui-parity/parity-*.diff.png`。这些数字说明现状远未达标，只作为下一轮修改的基线，不是通过证据。

## 无障碍与平台

| # | 项目 | 当前 | 结论 |
|---|---|---|---|
| A1 | 键盘可达性 / Tab 顺序 | 真实窗口 Tab 依次到达侧栏入口与会话行（AX 焦点序列） | **已通过（L4）** |
| A2 | VoiceOver | 无 | 未通过 |
| A3 | 中文/emoji IME | 无 | 未通过 |
| A4 | 复制/粘贴/撤销 | 真实窗口 Cmd+V 粘贴、Cmd+Z 撤销（`window-clipboard-undo.txt`） | **已通过（L4）** |
| A5 | Windows / Linux 运行时 | 只交叉编译 | 未通过 |

## 通过规则

1. 每一项必须有**可复现命令或截图**，写在对应行；没有证据一律 `未通过`。
2. L1–L3 证据只能证明实现行为，不能写成"验收通过"。
3. UI/视觉项必须同时有 L4 与 L5 证据；缺一即未通过。
4. 未实现的能力保持 visible-disabled WIP，不用本地状态补齐，也不计入通过。
5. 当前总体结论：**产品验收未通过**；F1/F2/F5 达到 L4，其余功能、设计与无障碍项只有 L1–L3 测试级证据或尚无证据。
