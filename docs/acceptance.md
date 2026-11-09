# 产品验收矩阵

Status: draft（F1/F2/F5 已达 L4；F3/F4/F6–F14、D1–D6、A1–A5 未通过）
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
| F12b | cut point | 选择父分支 sequence 创建 side line | 只有 seed 工具按 submitted input sequence 创建；UI 无 cut point 入口 | 未通过 |
| F13 | attachment 上传 | 真实上传并出现在 timeline | Go 侧 `attachment.put` helper 已有（L1/L2 fake-server）；shell 无文件选择器/结果读回；L3/L4 无 | 未通过 |
| F14a | session 删除 | 真实窗口删除 → App Server 确认后行消失 | L4：`session.delete` 结果因流关闭判 unknown，按合同读回 `session.catalog` 确认后移除；删到最后一个清空为 `Sessions 0`（`window-delete-session.png`） | **已通过（L4）** |
| F14b | 关闭终态显示 | `session.closed` 在窗口可见且不可再提交 | 删除瞬间短暂出现 `Session closed`，未单独留存证据；删除成功后选择被清空 | 未通过 |

## 设计 1:1 验收

| # | 项目 | 要求证据 | 当前 | 结论 |
|---|---|---|---|---|
| D1 | 语义 token 层 | token 对照表 + 与 Electron 变量一一映射 | L0：`docs/ui-token-map.md` 覆盖 35/35 颜色字段与 53/53 几何/排版字段，给出 A 侧 light/dark 行号与对应/部分判定；值差异（圆角 rem、Space 刻度、ring 色相）逐项记录 | **已通过（L0 文档映射）** |
| D2 | 主题/皮肤 | 与 Electron 主题数量、切换行为一致 | system/light/dark 已实现并持久化；skin/chrome 明确 WIP | 未通过 |
| D3 | 区域组件 | sidebar、chat-workspace、composer、context-panel、right-panel、settings 等逐一对照 | 并排图覆盖空态/设置/对话/错误四态（`docs/evidence/ui-parity/`）；sidebar 分组+相对时间、composer 宽度、品牌绿、助手行已对齐；composer 工具栏仍是带 WIP 徽标的文字入口 | 未通过 |
| D4 | 字体/密度/间距 | 截图叠加对照 | 已量到并记录：顶栏 41/实测 46、composer 534 vs 520、消息列留白 24.5、侧栏 256；`docs/ui-token-map.md` 记录字号/间距/圆角差异；未做同位置像素叠加 | 未通过 |
| D5 | Markdown/代码块渲染 | 真实内容对照 | 内容层已有代码块头/Copy、行内代码、链接、表格、任务列表和文件 chip；Electron 对话基线只有一轮纯文本 Echo，没有 Markdown 可对照 | 未通过 |
| D6 | 空态/加载/错误态 | 三种状态截图对照 | 空态（compare-empty）与错误态（compare-error）已并排；加载态未采集 | 未通过 |

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
