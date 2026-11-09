# 产品验收矩阵

Status: draft（当前没有任何一项达到 `accepted`；已有证据只到测试级）
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
| L4 `real-window` | 真实窗口 + 人工/截图对照 | **未执行** |
| L5 `design-parity` | token/组件/视觉与 Electron 端逐项对照 | **未执行** |

验收通过 = 目标行达到 L4，且视觉相关行同时达到 L5；L1–L3 只能写成"测试级证据"。

## 功能验收

| # | 项目 | 要求证据 | 当前 | 结论 |
|---|---|---|---|---|
| F1 | 应用启动（无 daemon） | 真实窗口截图 + 未连接状态可见 | 无 | 未通过 |
| F2 | 应用启动（真实 daemon，`DIMAGENT_PRODUCT_HOME`） | L3 + 真实窗口 + session 列表可见 | L4 窗口已连真实 daemon（空 catalog，无 session 列表，见 [window-evidence.md](window-evidence.md)） | 未通过 |
| F3 | session 创建 | 真实窗口点击 → 列表出现新 session（App Server 返回） | UI 已接线（headless/fake 测试通过）；L4 无 | 未通过 |
| F4 | session 切换 | 真实窗口切换 + 历史正确 | headless 覆盖，L4 无 | 未通过 |
| F5 | 首屏 history | 真实 daemon 内容在窗口渲染 | L3 协议路径通过；L4 无 | 未通过 |
| F6 | submit / 流式更新 | 真实窗口输入 → 发送 → 增量出现 | L3 通过；无持续 ticker；L4 无 | 未通过 |
| F7 | stop | 真实窗口停止 → 状态收敛 | L3 通过；L4 无 | 未通过 |
| F8 | approval 决策 | 真实窗口点击 Approve/Deny → 行消失、无重复发送 | headless + L3 协议；真实 approval 流未验收 | 未通过 |
| F9 | unknown-result 读回 | submit 超时/断线 → `input_lookup` 结果可见 | UI 已接线 + integration 单测；L4 无 | 未通过 |
| F10 | 断线 / reattach | 杀掉 daemon → 错误态 → 重连 → 历史重建 | 单测；L4 无 | 未通过 |
| F11 | `subscription_changed` 恢复 | 真实 replay-gap 场景 | 无真实 wire 证据 | 未通过 |
| F12 | branch switching / cut point | 真实切换 branch 并渲染 | 只有只读 branch 列表 | 未通过 |
| F13 | attachment 上传 | 真实上传并出现在 timeline | 无 helper | 未通过 |
| F14 | session 关闭/删除 | 终态显示 + 不再可提交 | 单测；L4 无 | 未通过 |

## 设计 1:1 验收

| # | 项目 | 要求证据 | 当前 | 结论 |
|---|---|---|---|---|
| D1 | 语义 token 层 | token 对照表 + 与 Electron 变量一一映射 | 只有 7 个颜色 + 比例间距；对照见 [ui-parity.md](ui-parity.md) | 未通过 |
| D2 | 主题/皮肤 | 与 Electron 主题数量、切换行为一致 | MyGo 单主题 | 未通过 |
| D3 | 区域组件 | sidebar、chat-workspace、composer、context-panel、right-panel、settings 等逐一对照 | 只实现前 3 类的一部分 | 未通过 |
| D4 | 字体/密度/间距 | 截图叠加对照 | 无 | 未通过 |
| D5 | Markdown/代码块渲染 | 真实内容对照 | 基础 Markdown | 未通过 |
| D6 | 空态/加载/错误态 | 三种状态截图对照 | headless 断言 | 未通过 |

## 无障碍与平台

| # | 项目 | 当前 | 结论 |
|---|---|---|---|
| A1 | 键盘可达性 / Tab 顺序 | headless 测试 | 未通过（无真实窗口验证） |
| A2 | VoiceOver | 无 | 未通过 |
| A3 | 中文/emoji IME | 无 | 未通过 |
| A4 | 复制/粘贴/撤销 | 无 | 未通过 |
| A5 | Windows / Linux 运行时 | 只交叉编译 | 未通过 |

## 通过规则

1. 每一项必须有**可复现命令或截图**，写在对应行；没有证据一律 `未通过`。
2. L1–L3 证据只能证明实现行为，不能写成"验收通过"。
3. UI/视觉项必须同时有 L4 与 L5 证据；缺一即未通过。
4. 未实现的能力保持 visible-disabled WIP，不用本地状态补齐，也不计入通过。
5. 当前总体结论：**产品验收未通过**，只有协议与测试级证据。
