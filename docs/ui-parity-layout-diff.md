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

空态已基本同构；剩余差异集中在 5 页脚（gear+文字 vs gear）、6 搜索快捷键、10 占位文案、11 未配置标签。

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
| 1 | 侧栏树 | 会话(+add/list) → 工作区(house) → 暂无会话 → 分隔线 → 项目分组 | 空态已同构；对话态只有 会话 → 项目分组，缺 工作区/暂无会话 两行 | 布局/内容 | 修复中（sidebar-details） |
| 2 | 顶栏右侧 | VS Code 打开 + 下拉、竖线、右面板开关 | 模型 ▾、竖线、右面板开关；无 VS Code 入口，模型是静态 WIP | 内容（缺事实源） | 部分保留 |
| 3 | 消息区 | 气泡 top=74，宽 489（x281–770） | 气泡 top=51（高 23px），宽 497（x283–780） | 布局 | 待修（顶部留白） |
| 4 | composer 右侧 | 未配置 + 绿色发送；卡下状态条：默认权限 / git main / 上下文条 / — | 无未配置；context 行有 工作区/默认权限，无分支、无上下文条 | 内容/布局 | 修复中（composer-context） |
| 5 | 页脚 | 登录 Dim + 齿轮图标 | 登录 Dim + 齿轮图标 + “设置”文字 | 内容 | 修复中（sidebar-details） |
| 6 | 搜索行 | 无快捷键提示 | 右侧显示 ⌘G | 内容 | 修复中（sidebar-details） |
| 7 | 会话行 | 标题 + 相对时间 | 状态圆点 + 标题 + 相对时间（时间数据不同是事实差异） | 内容 | 修复中（去掉圆点） |
| 8 | 全局入口 | 四项均可点 | 定时任务/技能插件为灰色禁用（缺客户端） | 能力缺口 | 保留 WIP |
| 9 | 占位文案 | 发送消息… | 给 DimAgent 发消息 | 内容 | 修复中（i18n） |
| 10 | 语言 | 中文 | 中文（capture 已强制 zh-CN） | — | 已对齐 |

## 度量（辅助）

| 区域 | 空态 MAE / >16% | 对话态 |
|---|---|---|
| sidebar | 6.8 / 4.5% | 9.5 / 12.2% |
| top | 4.7 / 15.3% | 21.6 / 30.4% |
| content | 4.9 / 6.3% | 4.6 / 3.9% |
| composer | **0.27 / 1.1%** | 3.7 / 5.8% |
| bottom | 3.3 / 12.2% | 8.7 / 20.3% |

空态 composer 已进入阈值；top 的高值来自顶栏右侧内容差异（VS Code/模型）与消息顶部留白，不是单纯颜色。
