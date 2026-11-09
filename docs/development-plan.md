# DimAgent MyGo 开发计划

Status: active（波次已完成：10 个 UI 组件 + 4 个主逻辑 worktree 全部合并，进入接线与验收轮。产品验收 **11 通过 / 16 未通过**，见 [acceptance.md](acceptance.md)）
Last updated: 2026-11-09
流程与角色见 [workflow.md](workflow.md)；逐日记录见 [archive/progress-2026-11.md](archive/progress-2026-11.md)。

## 目标与边界

用 MyGo native UI 重实现 DimAgent Desktop，行为与显示对齐 dim-electron。Rust `dim-app-server` 是 session、branch、timeline、approval、model、permission 的唯一事实源；Go 项目不读私有 SQLite/JSONL/session 文件/daemon.log，不建立本地 session/run 真相，不把 Electron IPC 搬进来。

只使用 MyGo 公开 `ui` API；不能复制 `internal/...`。跨 agent 合同以 Go package、wire DTO、golden fixture 为准。

---

## 当前基线

```text
验收：F1–F5、F10、F12a、F14a、A1、A4 通过（11）
      D1–D7、F6–F9、F11、F12b、F13、F14b、A2/A3/A5 未通过（16）
UI：聊天壳、空态、composer、顶栏、侧栏、设置路由、搜索浮层、keys/context/approval 面板
主逻辑：catalog/create/select/submit/stop/approval/delete/branch/reconnect/unknown-result 读回
波次：10 个 UI 组件 + 4 个主逻辑 worktree 已完成并合并（§2/§3）
```

---

## 阶段路线图

### P0 冻结与清理（本轮）

- 清理假入口：add/list、collapse、VS Code WIP、有会话时的“暂无会话” ✅
- 删除未使用 icons/i18n key（`sidebarIcons.list/collapse`、`openInEditor*`、`keyHeaderOpenInEditor` 等）
- 共享文件（`shell.go`、`protocols.go`、`tokens.go`、`icons.go`、`i18n.go`、docs）由 leader 统一改

退出：`gofmt`/`build`/`vet`/`test` 绿，无死代码引用。

### P1 组件并行（已合并，10 个 worktree；待接线与视觉验收）

| # | 组件 | 分支 | 退出条件 |
|---|---|---|---|
| 1 | Sidebar 行 | `feat/sidebar-polish` | 行高 28、文字缩进 24.5、分组/空态间距 |
| 2 | Header 菜单 | `feat/header-menu` | 参考菜单顺序、分隔线、WIP、键盘/Esc |
| 3 | Search 浮层 | `feat/search-polish` | 开合过渡、id 匹配、行图标态、⌘1–8 |
| 4 | Composer 状态 | `feat/composer-states` | 未配置 chip、Send 三态、Sending、错误行 |
| 5 | Markdown 块 | `feat/markdown-blocks` | 代码块头/Copy、链接、表格、任务、文件 chip |
| 6 | Tool 卡 | `feat/tool-cards` | 状态徽标、截断、折叠、错误态 |
| 7 | Approval 面板 | `feat/approval-panel2` | 三决策按钮、危险色、键盘、窄窗 |
| 8 | Context 面板 | `feat/context-panel2` | 文件树/diff/预览/空态、宽度 clamp |
| 9 | Keys 浏览器 | `feat/keys-panel2` | 表头/行高/等宽值/空态/键盘 |
| 10 | Settings 导航 | `feat/settings-nav-extras` | 19 项、Archive/About、紧凑布局 |

合并顺序：1/2/3/4 → 5/6/7/8/9 → 10；每个合并后全量 test/race → 真窗口截图 → 逐区域看图。

### P2 主逻辑并行（已合并，4 个 worktree；UI 接线在 P3）

| 逻辑 | 分支 | 退出条件 | 缺口 |
|---|---|---|---|
| Session methods | `feat/session-methods` | typed title/Rename、错误映射、测试 | 顶栏 Rename、标题 |
| Model catalog | `feat/model-catalog` | 模型列表/当前模型只读；无合同则 WIP | 顶栏/Composer 模型 |
| Resync hardening | `feat/resync-hardening` | subscription_changed/generation/closed/unknown-result | F11 |
| Cmd state | `feat/cmd-state` | 每会话草稿、窗口位置/尺寸、迁移 | 草稿/窗口 |

### P3 接线轮（P1/P2 后，leader 改 shell.go）

- 顶栏菜单回调：Pin/Rename/Archive/Export/Copy/Open in new window（有合同才接）
- 模型选择器：catalog → 顶栏/Composer；无事实显示“未配置”
- 路由：Scheduled、Skills 两个 WIP 页面；设置 19 分区
- 通知/错误中心：ActionError/ActionNotices/Notice 统一展示、可关闭/复制
- 会话内消息搜索（ThreadSearchBar，区别于全局 ⌘G）
- F9：unknown submit 的 `input_lookup` 结果展示
- 清理：未使用 glyph/key、protocols 注释、组件协议文档

退出：所有入口有真实行为或明确 WIP；截图覆盖 chat/settings/search/keys/context/approval。

### P4 功能验收补齐（F 行）

| 项 | 缺口 | 完成条件 | 依赖 |
|---|---|---|---|
| F6 流式 | mock 瞬回 | 真实模型/慢速 fixture：token 增量 + 截图/录屏 | 真实模型/上游 |
| F7 stop | mock 太快 | 运行态 Stop → 状态收敛 L4 | 同上 |
| F8 approval | mock 无审批 | 审批 → Approve/Deny → 行消失 | 同上 |
| F9 lookup | 只显示 unknown | lookup 结果（accepted/unknown）呈现 | P3 |
| F11 subscription_changed | 无真实 wire | 真实断流恢复或 fail-closed | 上游/故障注入 |
| F12b cut point | 无 UI | 选择父 sequence 创建 side line | 协议 + UI |
| F13 附件 | mygo 无文件选择 | 选择 → attachment.put → timeline | MyGo 上游 |
| F14b closed 终态 | 仅瞬时 | 稳定展示、不可提交 | P3 |

### P5 设计 1:1（D 行）

| 项 | 剩余工作 | 退出条件 |
|---|---|---|
| D1 颜色/表面 | 字体栅格、图标资产、vibrancy | 平面取样 + 视觉复核；差异进豁免表 |
| D2 排版/间距 | 行高 28/缩进、字体度量、compact | 坐标 ≤2 DIP；字号/行高一致 |
| D3 图标/状态色 | 官方资产或等价图形、hover/focus/disabled | 状态截图 + 视觉复核 |
| D4 主题/皮肤 | 6 chrome、skin 封面/导入、代码主题 | 数量/切换一致；资产就位 |
| D5 内容 | WIP vs 参考可用、菜单文案、真实 Markdown | 同语言截图 + 文本快照；逐条豁免 |
| D6 动画 | 参考帧序列、文字 ink、search enter/exit | 两边同操作帧序列 |
| D7 状态 | loading/hover/focus/menu/approval/unknown | 每态截图 + 复核 |

### P6 无障碍与平台（A 行）

- A2 VoiceOver 走查（侧栏/对话/composer/审批/设置）
- A3 中文/emoji IME、候选窗、Enter 边界
- A5 Windows/Linux 构建、chrome、字体、快捷键、路径、打包
- reduced motion 真实窗口验证

### P7 管理与扩展面（M5）

- 设置剩余分区：Account/Usage/Personalization/General/Shortcuts/Models/Multimodal/Runtime/MCP/Notifications/Pet/Permissions/Git/Hooks/Plugins/External Agent/Archive/About
- Scheduled tasks、Skills/plugins/market、Permissions 编辑器、always-allow grants
- Approval replay 集合替换、question/elicitation
- Goal/title/tasks/workflows
- 多窗口/托盘/通知、Account/login/usage

### P8 发布与质量

- 打包：macOS .app（签名/公证）、Windows/Linux、版本/更新通道
- 安全：日志脱敏、崩溃报告 opt-in、无本地 session 存储审计
- 性能：大 session 滚动/虚拟化、内存、首屏、重连抖动
- 测试基建：真实模型 fixture、故障注入、动画帧对比、视觉回归 CI、跨 DPI
- 文档：protocol/projection/component 同步、README、证据索引、#21 同步
- 退役 Electron 路径（如目标包含）

---

## 全量 backlog（按域）

### 会话与运行
- [ ] 重命名/标题（P2）
- [ ] Pin/Rename/Archive/Export/Copy/Open in new window（P3）
- [ ] 删除/关闭终态稳定展示（P3/P4）
- [ ] Cut point / side chat（P4）
- [ ] Goal/title/tasks/workflows（P7）
- [ ] Question/elicitation、always-allow grants（P7）
- [ ] Approval replay 集合替换（P7）
- [ ] 多窗口/托盘/通知（P7）

### 事件与恢复
- [ ] subscription_changed 真实恢复（P2/P4）
- [ ] unknown-result 全路径读回展示（P3/P4）
- [ ] 断线/重连/daemon restart UI 状态（部分完成）
- [ ] 持续 live ticker / run progress（P3）

### 输入与内容
- [ ] 附件选择/上传/读回（P4，MyGo API）
- [ ] 会话内消息搜索（P3）
- [ ] Markdown 真实内容对照（P5）
- [ ] 工具卡/thinking/file changes 全状态（P1/P5）
- [ ] 代码主题 + 预览（P5/P7）

### 外观与平台
- [ ] skin/chrome 资产与 resolver（P5）
- [ ] 6 个 chrome、vibrancy（P5）
- [ ] 字体族/代码字体分离（P5）
- [ ] compact/窄窗全页（P5/P7）
- [ ] 高 DPI/多显示器基线（P8）

### 设置与管理
- [ ] 19 分区内容（P1/P7）
- [ ] Scheduled tasks（P7）
- [ ] Skills/plugins/market（P7）
- [ ] Permissions 编辑器 + grants（P7）
- [ ] Account/Usage（P7）
- [ ] MCP/Runtime/Notifications/Pet/Git/Hooks/External Agent（P7）

### 质量与发布
- [ ] 真实模型验收环境（P4）
- [ ] 故障注入 harness（P4/P8）
- [ ] 动画参考帧 + 对比（P5）
- [ ] 视觉回归 CI（P8）
- [ ] VoiceOver/IME/Windows/Linux（P6）
- [ ] 打包/签名/更新（P8）
- [ ] 性能与安全审计（P8）

---

## 依赖与阻塞

| 阻塞 | 影响 | 处理 |
|---|---|---|
| 真实模型/审批流量 | F6/F7/F8、D5/D6 | 需模型配置或上游 fixture；不伪造 |
| MyGo 文件选择 API | F13 | 等公开 API；保持 visible-disabled |
| MyGo 文字 ink 过渡 | D6 | 记录渲染器限制；颜色/面板已实现 |
| Electron 专有资产 | D3 图标、skins | 获取资产或批准等价图形 |
| vibrancy/半透明 | D1/D4 sidebar | 待确认 mygo 支持 |
| 上游模型/scheduled/plugin 方法 | P3/P7 | 确认公开合同；缺失 WIP |
| VoiceOver/IME 自动化 | A2/A3 | 人工/系统级验证 |

## 合并与验收门禁

1. worktree：owner 自测 → leader 合并 → 全量 test/race → 真窗口截图 → 逐区域复核。
2. 共享文件只在接线轮由 leader 改；owner 报告需要的字段/key。
3. D 行通过：视觉复核 + 结构坐标 + 平面取样 + 动画帧；像素 MAE 仅辅助。
4. 未通过项不得本地补齐；不可实现项进豁免表并确认。

## 文档地图

| 想了解 | 读 |
|---|---|
| 流程、角色、门禁 | [workflow.md](workflow.md) |
| 验收标准与状态 | [acceptance.md](acceptance.md) |
| 布局/内容差异 | [ui-parity-layout-diff.md](ui-parity-layout-diff.md) |
| 逐区域视觉复核 | [ui-parity-review.md](ui-parity-review.md) |
| token 映射 | [ui-token-map.md](ui-token-map.md) |
| 动画 | [ui-motion.md](ui-motion.md) |
| wire 合同 | [protocol-contract.md](protocol-contract.md) |
| projection | [projection-contract.md](projection-contract.md) |
| 组件协议 | [ui-component-protocols.md](ui-component-protocols.md) |
| 历史进展 | [archive/progress-2026-11.md](archive/progress-2026-11.md) |
