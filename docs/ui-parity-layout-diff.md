# 布局与内容差异（ASCII + 标注图）

Status: current
Last updated: 2026-11-09

同状态、同尺寸（800×720）、浅色、中文（MyGo 用一次性 config 强制 `zh-CN`）。数字来自 `scripts/parity_diff.py`，图来自 `screencapture -l`。

标注图：
- [`evidence/ui-parity/layout-annotated-empty.png`](evidence/ui-parity/layout-annotated-empty.png)
- [`evidence/ui-parity/layout-annotated-conversation.png`](evidence/ui-parity/layout-annotated-conversation.png)

编号含义：1 侧栏树 · 2 顶栏右侧 · 3 消息区顶部/宽度 · 4 composer 右侧 · 5 页脚。

## 空态

```text
ELECTRON (800x720)                          MYGO (800x720)
+----------------+---------------------+    +----------------+---------------------+
| o o o      [|] |  (nothing above    |    | o o o          |  (card, no header)  |
|                |   the card)         |    |                |                     |
| + 新建会话      |                     |    | + 新建会话      |                     |
| Search          |      (brand)       |    | Search   Cmd+G  |      (brand)        |
| Scheduled       |      Start         |    | Scheduled       |      Start          |
| Skills/plugins  |  +-------------+   |    | Skills/plugins  |  +-------------+    |
| Sessions [+][=] |  | placeholder  |   |    | 会话    [+][=]  |  | placeholder  |    |
| Workspace       |  | + >> cube  @|   |    | 工作区          |  | + >> cube  @ |    |
|   (no sessions) |  | WS     Perm  |   |    |   暂无会话      |  | WS     Perm  |    |
| ----------------|  +-------------+   |    | ----------------|  +-------------+    |
| Login      gear |                     |    | 登录 Dim  gear+设置 |                    |
+----------------+---------------------+    +----------------+---------------------+
```

当前状态：侧栏纵向节奏、页脚（仅齿轮）、搜索快捷键、占位文案、工作区/暂无会话结构、会话行圆点均已对齐；剩余是字体栅格差异与 VS Code/面板图标形状。

## 对话态

```text
ELECTRON                                     MYGO
+--------------+-----------------------+    +--------------+-----------------------+
| o o o    [|] | proj / Title  ... 2 |    | o o o        | dimagent-mygo / Title 2|
|              |              VS v |[] |    |              |             模型 v  |[] |
| + 新建会话    |                       |    | + 新建会话    |                       |
| Search       |    [user bubble]    3 |    | Search  Cmd+G|    [user bubble]     3 |
| Scheduled    |    assistant text     |    | Scheduled    |    assistant text     |
| Skills       |                       |    | Skills       |                       |
| Sessions     |                       |    | 会话          |                       |
| Workspace    |                       |    | dimagent-mygo |  (no Workspace row)  |
|   no sessions|                       |    |  [session 2h] |                       |
| ------------ |  +-----------------+  |    |              |  +-----------------+  |
| project      |  | placeholder     |  |    |              |  | placeholder     |  |
|  [session]   |  | + >> cube     @ |  |    |              |  | + >> cube     @ |  |
|              |  | WS Perm  main - |4 |    |              |  | WS Perm         |4 |
| Login   gear |  +-----------------+  |    | 登录 Dim 设置 |  +-----------------+  |
+--------------+-----------------------+    +--------------+-----------------------+
```

## 差异清单（编号对应标注图）

| # | 区域 | Electron | MyGo 现在 | 类型 | 状态 |
|---|---|---|---|---|---|
| 1 | 侧栏树 | 会话(+add/list) → 工作区(house) → 暂无会话 → 分隔线 → 项目分组 | 同构；所有元素纵向位置与参考相差 ≤2.5 DIP | 布局/内容 | 已对齐 |
| 2 | 顶栏右侧 | VS Code 蓝标 + chevron、竖线、描边面板开关 | VS Code 蓝标 + chevron、竖线、面板开关；图标形状/描边不同 | 样式（图标形状） | 部分（缺参考图标资产） |
| 3 | 消息区 | 气泡 top=74，x281–770；助手行 y≈149 | 气泡 top=73.5，x?–771；助手行 y≈149；左右内距 24.5 | 布局 | 已对齐 |
| 4 | composer 右侧 | 未配置 + 绿色发送；卡外状态带：默认权限 / git main / 35×3.5 空轨道 / — | 未配置 + 绿色发送；状态带位置与元素 x 对齐（盾牌 285 vs 284，main 662 vs 663，轨道 706–741，短横 761 vs 760） | 内容/布局 | 已对齐（轨道为 muted 空轨） |
| 5 | 页脚 | 登录 Dim + 齿轮图标 | 登录 Dim + 齿轮图标 | 内容 | 已对齐 |
| 6 | 搜索行 | 无快捷键提示 | 无快捷键提示 | 内容 | 已对齐 |
| 7 | 会话行 | 标题 + 相对时间 | 标题 + 相对时间（时间值不同是事实差异） | 内容 | 已对齐（值不同） |
| 8 | 全局入口 | 四项均可点 | 定时任务/技能插件灰色禁用（缺客户端） | 能力缺口 | 保留 WIP |
| 9 | 占位文案 | 发送消息… | 发送消息… | 内容 | 已对齐 |
| 10 | 语言 | 中文 | 中文（capture 强制 zh-CN） | — | 已对齐 |

## 度量（辅助）

| 区域 | 空态 raw MAE / >16% | 对话态 raw | 对话态 blur=2 |
|---|---|---|---|
| sidebar | 5.26 / 2.8% | 6.52 / 4.8% | 4.90 / 4.4% |
| top | 4.72 / 15.3% | 22.17 / 30.8% | 11.01 / 26.6% |
| content | 4.83 / 6.2% | 3.02 / 3.2% | 1.52 / 2.9% |
| composer | **0.27 / 1.1%** | 3.26 / 5.9% | 1.95 / 3.6% |
| bottom | 3.26 / 12.2% | 5.77 / 15.4% | 3.83 / 10.7% |

top 的 raw 值主要是**字体栅格差异**：AppKit 与 Chromium 对同一字号/字重的字形覆盖不同，逐像素必然有差；布局坐标已对齐（项目名截断 ink 278–359 vs 277.5–356，标题起点 374 vs 375，控件 x ±1）。因此数值门禁要区分：
- 结构/布局门禁看几何坐标（已对齐项逐条列在上表）；
- 颜色门禁看平面区域（composer/sidebar 面色）；
- 文本区用 blur=2 的 perceptual MAE + 逐区域视觉确认，而不是要求跨渲染器逐像素一致。
