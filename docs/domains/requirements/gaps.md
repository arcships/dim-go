# 未实现分支与依赖清单（下一阶段输入）

Status: 缺口清单（只读盘点，不含实现）
Scope: 主窗口 / 右栏 / 设置 / 全局四区 + 命令面板；文件为组织方案的下一阶段输入
Date: 2026-11-18

## 0 口径

### 0.1 素材（只读）

| 类 | 来源 |
|---|---|
| 元素清单 | `docs/domains/requirements/inventory-main.md`、`inventory-panels.md`、`inventory-settings.md`、`inventory-global.md` 及四份 `*-rt.md` 运行时版 |
| 行为需求 | `docs/domains/requirements/requirements-main/panels/settings/global.md` |
| 清单核对结论 | `docs/domains/requirements/inventory-main-gaps.md`、`inventory-settings-gaps.md`、`review-final.md` |
| 领域边界 | `docs/domains/D1-connection.md`（能力缺失与 §10 四项卡点）、`D7-workspace.md` §1「不做」、`D9-terminal.md` §13、`D14-delivery.md` §5「未接动作」 |
| 代码事实 | `internal/ui/*.go`（各屏文件头的「# 未实现分支 / # 状态 / 未接动作」段）、`internal/ui/source.go`（Source / PanelFacts / Intents）、`internal/uiwire/adapter.go`、`internal/host/update.go` |

### 0.2 「缺什么」四类

| 类 | 含义 |
|---|---|
| **事实字段** | 服务端或本机已有数据，但没进 `Source`/`PanelFacts`，视图读不到；或根本没有这条事实 |
| **协议方法** | protocol v3 没有该方法（或未注册 / 回执未冻结），客户端发不出去 |
| **宿主能力** | 窗口、系统目录选择器、外开、通知授权、隔离容器、浮窗等平台能力 |
| **无（纯 UI）** | 事实与通道都齐，差的是视图这一层（样例未换成读数、按钮没接 Intents、控件未做） |

### 0.3 「能否不改服务端实现」三值

- **能**：只改 Go 侧（接已有读数 / 接已有 Intents / 做本地状态与控件）。
- **需裁决**：先定归属（本地登记还是服务端事实）、或先定产品范围（该项要不要做、哪个平台做）。
- **不能**：必须有新事实字段或新协议方法（含上游注册/发布）才能实现。

### 0.4 排序

四区各成一段；段内按 **能 → 需裁决 → 不能** 排（编号跨批次保持稳定，因此段内编号不连续）。跨区的最优顺序见 §2。

---

## 1 缺口表

### 1.1 全局与命令面板（G）

| 编号 | 界面元素/分支 | 所属区 | 缺什么（事实字段 / 协议方法 / 宿主能力） | 归属领域 | 能否在不改服务端的前提下实现 | 备注 |
|---|---|---|---|---|---|---|
| G-01 | 命令面板 · 动作「新建会话」 | 全局 / 命令面板 | 事实字段：工作区（项目根）清单未进 `Source`，而 `CreateSession` 需要 `cwd` | D11a / D7 | 能 | `internal/ui/command_palette.go:694`–`:698` 置灰并写原因；项目根事实已存在于 `internal/settings/projects.go:19`–`:70`，只是未暴露 |
| G-06 | `$` 技能提及弹层 | 全局 / 提及 | 无（`Extensions().Skills` 已有） | D11b | 能 | §A4；与 M-12 同一份读数 |
| G-07 | 斜杠命令弹层 | 全局 / 命令 | 无（命令表是客户端静态表） | D11a | 能 | §B1；`/workflow` 一类依赖 G-21/P-14，其余可先做 |
| G-15 | 更新提示与安装确认 | 全局 / 更新 | 事实字段：`Source` 未暴露 `host.Updater` | D10 | 能 | `internal/host/update.go:284`–`:374` 已有 Check/Download/Switch |
| G-22 | 通知 / 错误 toast 体系 | 全局 / 反馈 | 无（渲染上下文有 `c.Toast`，缺各屏统一的错误与回执宿主） | D11a | 能 | `internal/ui/settings_notifications_screen.go:24` 登记「没有 toast 宿主」；`settings_account_screen.go:1550` 已有用法 |
| G-23 | 外开链接 / 在文件管理器中显示 | 全局 / 宿主 | 无（`internal/workspace/open.go` 已有按平台打开） | D7 | 能 | `internal/ui/header.go:185`、`:222`–`:229` 只写 `headerNote`，不发动作；`settings_mcp_screen.go:266` 同 |
| G-05 | `@` 提及弹层（文件 / 提交 / 技能） | 全局 / 提及 | 事实字段：提交候选需 git 历史；文件与技能两段的事实已有 | D7 / D11b | 需裁决 | §A3；建议先做文件 + 技能两段，提交段单列 |
| G-12 | 企业邮箱登录提交 | 全局 / 账户 | 协议方法：`account.login.begin` 已接，邮箱 + 验证码这段无方法 | D6 | 需裁决 | §K1；先定「桌面是否承接邮箱登录」 |
| G-14 | 首次运行欢迎卡 | 全局 / 首次运行 | 无（纯 UI） | U1 / D11a | 需裁决 | §K5；现为两条自建入口（`internal/ui/composer.go:120`–`:130`），与参考不同 |
| G-21 | 创建定时任务 | 全局 / 定时 | 协议方法（create 未冻结）；清单读已有 | D11b / D1 | 需裁决 | §J5；`SchedulesSnapshot` 已在 `Source` 但无页面 |
| G-02 | 会话内搜索条（ThreadSearchBar） | 全局 / 搜索 | 协议方法：会话内检索与命中定位 | D2 / D1 | 不能 | `inventory-global.md` §A1；全仓无实现 |
| G-03 | 搜索聊天对话框（ChatSearchDialog） | 全局 / 搜索 | 协议方法：跨会话检索 | D2 / D1 | 不能 | §A2；参考是全窗弹窗，现只有左栏行内过滤 |
| G-04 | 时间线命中高亮 | 全局 / 搜索 | 协议方法：命中偏移 | D2 | 不能 | §A5；依赖 G-03 |
| G-08 | 编辑运行中消息 + 编辑确认框 | 全局 / 确认 | 协议方法：编辑 / 重发 | D3 / D5 | 不能 | §E1；`docs/domains/U4-technical.md:218` 记为后续批次 |
| G-09 | Fork 会话（同会话两窗口） | 全局 / 派生 | 协议方法 + 宿主能力（多窗口） | D3 / D10 | 不能 | §G2；本机只允许单窗口持有会话 |
| G-10 | 添加插件 / 市场安装 / 替换确认 / 可更新清单 | 全局 / 导入 | 协议方法（`plugin.install/update` 回执未冻结）+ 宿主能力（目录选择） | D11b / D1 | 不能 | `D1-connection.md` §8 第 18 条；§H1–H5 |
| G-11 | 导入 Chrome Cookie / 清理浏览器数据 | 全局 / 导入 | 宿主能力（浏览器数据目录与解密） | D11a / D10 | 不能 | §H6–H7 |
| G-13 | 微信扫码 / 飞书扫码 | 全局 / 账户 | 协议方法 | D6 | 不能 | §K2–K3 |
| G-16 | Appshots 引导与权限请求 | 全局 / 首次运行 | 宿主能力（截屏与系统权限） | D10 | 不能 | §K7 |
| G-17 | 审批卡变体（沙箱越权 / 子代理模型 / 9 类输入渲染器 / 成员审批） | 全局 / 审批 | 事实字段：需服务端产生对应 `variant` | D5 | 不能 | `inventory-global-rt.md` §10.3；本机默认权限下不可达 |
| G-18 | 提问卡多字段 / 多选 / 倒计时 | 全局 / 提问 | 事实字段 | D5 | 不能 | §D1；现只覆盖单字段单选 |
| G-19 | 宠物活动 | 全局 / 宠物 | 宿主能力（浮窗） | D11a / D10 | 不能 | §I4；`internal/ui/settings_pets_screen.go:33`–`:50` 已登记无渲染器 |
| G-20 | Git 写动作（提交 / 建分支 / 切分支 / 撤销） | 全局 / Git | 协议方法 | D7 | 不能 | `D7-workspace.md` §1「不做」；§J1–J4 |

### 1.2 主窗口（M）

| 编号 | 界面元素/分支 | 所属区 | 缺什么（事实字段 / 协议方法 / 宿主能力） | 归属领域 | 能否在不改服务端的前提下实现 | 备注 |
|---|---|---|---|---|---|---|
| M-01 | 空态控制行 · 工作区选择器 | 主窗口 / 输入区 | 事实字段：工作区清单未进 `Source` | D7 | 能 | `internal/ui/composer.go:601`–`:640` 只改屏内名字；与 G-01 同一缺口 |
| M-03 | 工作区分组与折叠 | 主窗口 / 左栏 | 无（`SessionSnapshot.Cwd` 已有） | D3 / D7 | 能 | `internal/ui/sidebar.go:147`–`:180` 现用样例分组；按 `Cwd` 分组即可 |
| M-07 | 会话行状态标记（运行 / 等待） | 主窗口 / 左栏 | 无（`Running` / `Pending` 已有） | D2 | 能 | `internal/ui/sidebar.go:767`–`:768` 明确「后续批次接」 |
| M-08 | 模型浮层列表 / 最近使用 | 主窗口 / 输入区 | 无（`Settings().Models` / `ProviderRecords` 已有） | D4 / D11a | 能 | `internal/ui/composer.go:490`–`:518` 用 `modelSamples()` 硬编码 |
| M-10 | 选择模型写入 | 主窗口 / 输入区 | 无（`SelectModel` 已实现） | D4 | 能 | `internal/ui/source.go:91`、`internal/uiwire/adapter.go:991`；UI 侧零调用 |
| M-12 | 技能浮层 | 主窗口 / 输入区 | 无（`Extensions().Skills` 已有） | D11b | 能 | `internal/ui/composer.go:540`–`:575` 用 `skillSamples()` |
| M-13 | 模式 Goal | 主窗口 / 输入区 | 无（`ReadGoal` / `SetGoal` 已接） | D12 | 能 | `internal/ui/composer.go:593`–`:598` 置灰并写「缺少目标事实」，与实现不符 |
| M-14 | 权限选择器 / 状态条权限 | 主窗口 / 状态条 | 无（`SetPermissionTier` 已实现） | D5 | 能 | `internal/ui/source.go:90`、`adapter.go:976`；UI 侧零调用 |
| M-16 | 状态条 · 套餐用量段 | 主窗口 / 状态条 | 无（`UsageSnapshot` 已有） | D6 | 能 | `internal/ui/statusbar.go:204` 现走样例读数 |
| M-17 | 标题栏 · 打开方式组 | 主窗口 / 标题栏 | 宿主能力（应用发现与外开） | D10 / D7 | 能 | `internal/ui/header.go:22`–`:29` 四款应用硬编码；点击只写提示 |
| M-18 | 标题栏 ⋯ 菜单 · 三条调试复制项 | 主窗口 / 标题栏 | 无（剪贴板已有） | D11a | 能 | `inventory-main.md` C2；菜单其余六项已接 |
| M-26 | 标题栏 ⋯ 菜单 · 导出 / 归档 | 主窗口 / 标题栏 | 无（`ExportSession` / `ArchiveSession` 已实现） | D3 | 能 | `internal/ui/header.go:146`–`:151` 只写 `headerNote` 与 `a.exported`，不发请求；同菜单「重命名」按 `SessionTitle` 能力禁用（服务端两版都没有 `session.rename`） |
| M-19 | 消息动作：复制 / 重试 / 分叉 | 主窗口 / 时间线 | 无（剪贴板、`RetrySubmit`、`CreateBranch` 都有） | D3 / D2 | 能 | `internal/ui/timeline.go:166`、`:510`；三个 `iconButton` 无点击分支 |
| M-23 | 输入区活动条（ComposerActivityStrip） | 主窗口 / 输入区 | 无（`TaskSnapshot` 已有） | D11a | 能 | 后台任务入口暂挂右栏 ＋ 菜单（`internal/ui/rightpanel_parts.go:466`–`:469`） |
| M-24 | 长会话虚拟化（千轮以上） | 主窗口 / 时间线 | 无（纯 UI 性能） | D11a | 能 | `docs/domains/U4-technical.md:227`「虚拟化未实现」 |
| M-25 | 空态 · 办公模式推荐卡（6 张） | 主窗口 / 空态 | 无 | D11a | 能 | `inventory-main-gaps.md` A-02 |
| M-05 | 会话置顶（已置顶组 + 持久化） | 主窗口 / 左栏 | 事实字段：无置顶事实；本地偏好可扩项 | D11a | 需裁决 | `internal/ui/sidebar.go:155`、`:408`–`:413` 置顶只活在本进程 |
| M-06 | 在新窗口中打开 | 主窗口 / 左栏 | 宿主能力（多窗口） | D10 | 需裁决 | `internal/ui/sidebar.go:441`–`:442` 登记「不在本批次」 |
| M-11 | 思考强度写入 | 主窗口 / 输入区 | 协议方法：无 thinking level 写入口 | D4 / D2 | 需裁决 | `internal/ui/composer.go:434`–`:460` 只改滑杆值 |
| M-02 | 添加新工作区 | 主窗口 / 左栏 | 宿主能力（系统目录选择器）+ 项目根授权方法 | D7 / D10 | 不能 | `internal/ui/sidebar.go:198`、`:254`–`:256` 只留入口 |
| M-04 | 工作区重命名 / 移除 / 在文件管理器中显示 | 主窗口 / 左栏 | 协议方法（项目根写）+ 部分宿主能力 | D3 / D7 | 不能 | `internal/ui/sidebar.go:319`–`:338` 六项菜单只接通「归档会话」 |
| M-09 | 模型浮层 · 子代理 / 外部 agent 页签 | 主窗口 / 输入区 | 事实字段：子代理模型策略、外部代理 | D4 / D11a | 不能 | `internal/ui/composer.go:482`–`:486` 只画页签字 |
| M-15 | 状态条 · 上下文用量 | 主窗口 / 状态条 | 事实字段：无上下文 used/limit | D2 | 不能 | `internal/ui/statusbar.go:176`–`:204`；样例值在 `app.go:409`，清零在 `:584` |
| M-20 | 消息动作：编辑、从此处侧聊 | 主窗口 / 时间线 | 协议方法 + 事实字段 | D3 / D5 | 不能 | 同上位置；编辑见 G-08，侧聊见 P-09、P-11 |
| M-21 | 本轮改动块（TimelineFileChanges） | 主窗口 / 时间线 | 事实字段：条目级明细与 diff 文本 | D7 / D12 | 不能 | 与 P-03 同源；已撤销态无事实 |
| M-22 | 本轮产物块（TimelineArtifacts） | 主窗口 / 时间线 | 事实字段：runId 产物清单 | D2 / D7 | 不能 | `inventory-panels.md` §G |

### 1.3 右栏（P）

| 编号 | 界面元素/分支 | 所属区 | 缺什么（事实字段 / 协议方法 / 宿主能力） | 归属领域 | 能否在不改服务端的前提下实现 | 备注 |
|---|---|---|---|---|---|---|
| P-01 | 文件预览页（tabFilePreview） | 右栏 / 文件 | 事实字段：`PanelFacts` 无 Preview，而 D7 已有 | D7 / D11a | 能 | `internal/ui/rightpanel_parts.go:551`–`:553` 页面留空；`internal/workspace/preview.go:61` 已能读 |
| P-02 | 文件页右键动作（预览 / 外部工具打开 / 文件管理器 / 另存为） | 右栏 / 文件 | 无（外开已有） | D7 | 能 | `internal/ui/rightpanel_files.go:747`–`:781` 只画菜单项 |
| P-05 | 后台页 · 停止按钮 3 秒 / 失焦复位 | 右栏 / 后台 | 无（缺一个计时器） | D11a | 能 | `internal/ui/rightpanel_tasks.go:117` 明确「未实现」 |
| P-10 | 侧边对话 · 结束分支 | 右栏 / 侧聊 | 无（`EndBranch` 已实现） | D3 | 能 | `internal/ui/rightpanel_sidechat.go:147`–`:150` 只置本地状态 |
| P-15 | 右栏整屏替代主区 | 右栏 / 外壳 | 无 | D11a | 能 | `internal/ui/app.go:884`「本批次未做」 |
| P-16 | 目标面板重建 | 右栏 / 目标 | 无（goal 事实已接） | D11a | 能 | `internal/ui/rightpanel.go:113` 仍是有既有形态 |
| P-08 | 唤醒页（tabWakeup） | 右栏 / 唤醒 | 协议方法：唤醒事实归服务端 | D11b / D1 | 需裁决 | `internal/ui/rightpanel_parts.go:547` 页面留空；与 G-21 同一族 |
| P-17 | 终端 · Windows 平台 | 右栏 / 终端 | 宿主能力（Windows 伪终端） | D9 | 需裁决 | `D9-terminal.md` §12；`pty_other.go` 明确不支持 |
| P-18 | 终端 · 滚动缓冲 / 复制粘贴 / 输入法 | 右栏 / 终端 | 无（属自身范围） | D9 | 需裁决 | `D9-terminal.md` §13 第 9 条列为「不做」 |
| P-03 | 改动页 · 条目级明细（暂存分组 / ± 行数 / diff） | 右栏 / 改动 | 事实字段 + 协议方法：D7 git 只读不含这些 | D7 | 不能 | `internal/ui/rightpanel_changes.go:15`–`:22`；`D7-workspace.md` §1「不做」 |
| P-04 | 改动页 · git 写动作 | 右栏 / 改动 | 协议方法 | D7 | 不能 | 同上 `:24`–`:26`；菜单项已按禁用态渲染 |
| P-06 | 后台页 · 详情元信息（pid / 结束时刻 / 日志入口） | 右栏 / 后台 | 事实字段：`TaskSnapshot` 只有 id / 描述 / 状态 | D2 / D9 | 不能 | `internal/ui/rightpanel_tasks.go:138` |
| P-07 | 后台页 · 已停止态 | 右栏 / 后台 | 事实字段：快照只有 `stopping` | D2 | 不能 | `internal/ui/rightpanel_tasks.go:15`–`:16` |
| P-09 | 侧边对话 · 发送 / 引导 / 停止 | 右栏 / 侧聊 | 事实字段：侧聊分支与消息 | D3 / D2 | 不能 | `internal/ui/rightpanel_sidechat.go:151`–`:157` 只做本地回声 |
| P-11 | 子代理面板 · 时间线 | 右栏 / 子代理 | 事实字段：子代理会话流 | D2 / D5 | 不能 | `internal/ui/rightpanel_subagent.go:25`–`:31` 两条样例 |
| P-12 | 子代理面板 · 结束成员 | 右栏 / 子代理 | 事实字段：成员会话标识需从事件推导 | D1 / D5 | 不能 | `D1-connection.md` §10.4 方法已冻结但标识来源绕 |
| P-13 | 浏览器 / 扩展 / MCP App 页 | 右栏 / 外壳 | 宿主能力（独立隔离容器） | D10 / D11b | 不能 | `internal/ui/rightpanel_parts.go:455`、`:458` 禁用；`AGENTS.md` §3 要求隔离容器 |
| P-14 | 工作流页（tabWorkflow） | 右栏 / 外壳 | 协议方法：`workflow.*` 未注册 | D11b / D1 | 不能 | `D1-connection.md` §10.2；上游未随包提供插件 |

### 1.4 设置（S）

| 编号 | 界面元素/分支 | 所属区 | 缺什么（事实字段 / 协议方法 / 宿主能力） | 归属领域 | 能否在不改服务端的前提下实现 | 备注 |
|---|---|---|---|---|---|---|
| S-01 | 设置 19 节的读路径（17 节不读 `Source`） | 设置 / 全局 | 事实字段：`render*` 只收 `*gui.Context`，读数接不进来 | D11a | 能 | `internal/ui/settings_*_screen.go` 全部 `func render*(c *gui.Context)`；`Settings()` / `Extensions()` / `Schedules()` 定义在 `internal/ui/source.go:29`–`:33`；`Extensions()` 与 `Schedules()` UI 侧零消费，`Settings()` 只被 `currentModel()` 取了一项模型名 |
| S-02 | MCP 服务器清单 | 设置 / MCP | 无（`Extensions().MCP` 已有） | D11b | 能 | `internal/ui/settings_mcp_screen.go:29`、`:193` 样例一条 |
| S-06 | 插件设置表单数据 | 设置 / 插件设置 | 无（`Extensions().Plugins` 已有） | D11b | 能 | 同上文件头（3 个插件是样例） |
| S-09 | Hooks 列表与 Lock 图标 | 设置 / Hooks | 无（`Extensions().Hooks` 已有） | D11b | 能 | `internal/ui/settings_hooks_screen.go:22`–`:32` 两组皆空、`plugin:` 来源未画 |
| S-19 | 模型供应 · 写事实（启用 / 开关模型 / 探测 / 新增） | 设置 / 模型供应 | 无（`CreateProvider` 等已实现未接） | D4 | 能 | `internal/ui/source.go:103`–`:107`、`adapter.go:1211`–`:1310`；UI 侧零调用 |
| S-23 | 关于 · 更新状态机 | 设置 / 关于 | 事实字段：`host.Updater` 未进 `Source` | D10 | 能 | `internal/ui/settings_about_screen.go:26`–`:33`；与 G-15 同一缺口 |
| S-05 | 插件设置保存 / 重载 | 设置 / 插件设置 | 协议方法（回执）+ 事实字段 | D11b | 需裁决 | `internal/ui/settings_plugin_settings_screen.go:361`、`:227`、`:283` 只走本屏校验 |
| S-10 | 运行环境 · 条目区 | 设置 / 运行环境 | 协议方法（`runtime.*` 未注册）+ 客户端启动配置 | D1 / D11a | 需裁决 | `internal/ui/settings_runtime_screen.go:22`–`:27`；`D1-connection.md` §10.1 给出可加 `--defaults` 的解法 |
| S-12 | 个性化 · 58 个分支 | 设置 / 个性化 | 协议方法：设置写入无方法 | D11a | 需裁决 | `internal/ui/settings_personalization_screen.go:28`–`:52`（界面模式、ACP、默认打开应用、子代理模型、命令 Shell、提示词语言、自定义指令） |
| S-13 | 权限 · 保存 / 刷新 / 离开拦截 | 设置 / 权限 | 协议方法 + 宿主能力（设置壳的离开钩子） | D11a / D5 | 需裁决 | `internal/ui/settings_permissions_screen.go:36`–`:48`；`WriteCommandRules` 是唯一已接的写 |
| S-15 | 外观 · 持久化 / 皮肤文件选择 | 设置 / 外观 | 事实字段（`PreferencesSnapshot` 只有 4 项）+ 宿主 | D11a / D10 | 需裁决 | `internal/ui/settings_appearance_screen.go:20`–`:22`、`:714`–`:718` |
| S-18 | 模型供应 · 详情六形态与「添加模型」对话框 | 设置 / 模型供应 | 协议方法（`provider.*` 部分已有）+ 未重建 | D11a / D4 | 需裁决 | `internal/ui/settings_configuration_screen.go:33`–`:37`、`:530`、`:869`–`:872` 统一「尚未按参考产品重建」 |
| S-20 | 账户 · 品牌标识与头像资产 | 设置 / 账户 | 宿主能力（素材） | D11a / D6 | 需裁决 | `internal/ui/settings_account_screen.go:44`、`:201` 用占位圆环/首字母 |
| S-27 | 用量 · 四个分支不可达 | 设置 / 用量 | 无（产品形态决定） | D11a | 需裁决 | `internal/ui/settings_usage_screen.go:19`–`:25`：只读模式恒真，其余四支在本产品不成立 |
| S-03 | MCP 写 / 删 / 启停 / 移动 / 测试 / 导入 | 设置 / MCP | 协议方法（`mcp.*` 写与 inspect）；`TestMCP`/`ImportMCP` 的 Intents 已实现 | D11b / D1 | 不能 | 同上 `:32`–`:50`；`internal/ui/source.go:118`–`:120` 零调用 |
| S-04 | MCP 查看连接 / OAuth 认证 | 设置 / MCP | 协议方法 + 宿主能力（浏览器授权） | D11b / D10 | 不能 | 同上 `:18`、`:45`、`:531`–`:536` |
| S-07 | Git 设置 · 远程 / 分支清单 | 设置 / Git | 协议方法（仓库事实） | D7 | 不能 | `internal/ui/settings_git_screen.go:28`–`:29`、`:89` 样例 |
| S-08 | Git 设置 · 保存校验 / 目录选择器 | 设置 / Git | 协议方法 + 宿主能力 | D7 / D10 | 不能 | 同上 `:31`–`:33`、`:192` |
| S-11 | 多模态 · 30 个分支 | 设置 / 多模态 | 协议方法（capabilities 未声明、任务方法、realtime adapter） | D1 / D11a | 不能 | `internal/ui/settings_modality_screen.go:29`–`:45`；`D1-connection.md` §10.3 |
| S-14 | 通知 · 持久化与系统权限 | 设置 / 通知 | 事实字段 + 宿主能力 | D11a / D10 | 不能 | 同上 `:22`–`:32`；退出重进要保持开位需设置事实 |
| S-16 | 快捷键 · 保存与按键捕获 | 设置 / 快捷键 | 事实字段 + 宿主能力 | D11a / D10 | 不能 | `internal/ui/settings_keybindings_screen.go:1`–`:25` 只改屏内状态 |
| S-17 | 通用 · 自启 / 性能日志 / 代理 / 浏览器数据 / 存储清理 | 设置 / 通用 | 宿主能力 + 设置域 | D11a / D10 | 不能 | `internal/ui/settings_general_screen.go:58`–`:80`、`:126` |
| S-21 | 账户 · 用量刷新 / 重置额度「使用」/ 购买页外开 | 设置 / 账户 | 协议方法 + 宿主能力 | D6 | 不能 | 同文件头「未接动作」四条 |
| S-22 | 归档会话 · 归档清单事实 | 设置 / 归档会话 | 事实字段：`Source` 无归档清单 | D3 / D11a | 不能 | `internal/ui/settings_archived_screen.go:20`–`:28` |
| S-24 | 关于 · 公告正文与重拉 | 设置 / 关于 | 事实字段：数据层只有摘要 | D11a | 不能 | 同上 `:382`–`:396`（另 `:26`–`:32` S-about-26） |
| S-25 | 宠物 · 浮窗与写入 | 设置 / 宠物 | 宿主能力 + 宠物领域 | D11a / D10 | 不能 | `internal/ui/settings_pets_screen.go:33`–`:50` |
| S-26 | 外部 Agent | 设置 / 外部 Agent | 协议方法（上游未实现外部代理） | D1 / D11a | 不能 | `internal/ui/settings_external_agents_screen.go:16`、`:90` |

---

## 2 前 10 项建议（按「能先做」排序）

判据三条：不改服务端 → 事实/意图已经存在 → 一处接线能带动多处入口。

| # | 建议项 | 为什么排这里 | 收口判据 |
|---|---|---|---|
| 1 | **M-08 模型浮层接 `Settings().Models`** | 全界面最高频入口，`modelSamples()` 是唯一硬编码模型清单；读数已在 `SettingsSnapshot.Models` | 连上 Source 后浮层里不出现任何样例模型名 |
| 2 | **M-10 选择模型写入（`SelectModel`）** | 通道已通到服务端，UI 侧零调用；与第 1 项同一处代码 | 点模型 → 服务端收到 `provider.select`，标题行模型名随之变 |
| 3 | **M-01 + G-01 工作区清单进 `Source`** | 解开两处死路：命令面板「新建会话」与空态工作区选择器；事实已在 `settings.projects` | 选工作区 → `session.create` 带正确 `cwd` |
| 4 | **M-12 技能浮层接 `Extensions().Skills`** | 与第 1 项同形，`skillSamples()` 可整体删除 | 浮层条目数与 `Extensions().Skills` 一致 |
| 5 | **M-14 权限档位接线（`SetPermissionTier`）** | 状态条与输入区两处入口，`session.permission.update` 已有能力门控 | 切档 → 服务端收到 `session.permission.update`，状态条文案随事实变 |
| 6 | **M-13 Goal 模式接线** | 功能已通（`ReadGoal`/`SetGoal`/`PauseGoal`/`ResumeGoal`/`ClearGoal` 全接），只有菜单在置灰 | 菜单不再写「缺少目标事实」 |
| 7 | **S-02 + S-09 两屏接事实（MCP、Hooks）** | `Extensions().MCP` / `.Hooks` 已有读数，两屏现在都是样例或空态 | 两屏行数与读数一致；`ReadError` / `CapabilityMissing` 有渲染分支 |
| 8 | **S-19 模型供应写事实** | `CreateProvider` / `UpdateProvider` / `ProbeProvider` / `SetProviderModelEnabled` 四个 Intents 已实现未接 | 四个动作各打印对应方法名与次数 |
| 9 | **P-01 文件预览页（`PanelFacts` 加 Preview）** | 唯一「后端已能、前端留空」的整页；`internal/workspace/preview.go:61` 现成 | 点文件 → 预览页显示内容，二进制与截断有说明 |
| 10 | **M-19 消息动作接线（复制 / 重试 / 分叉）** | 三个 `iconButton` 无点击分支，通道全在（剪贴板、`RetrySubmit`、`CreateBranch`） | 三个动作各发一次请求或写一次剪贴板 |

> 备选（第 11）：G-07 斜杠命令弹层——纯客户端命令表，能一次补齐输入区一个空位；`/workflow` 一类随 P-14 后置。

---

## 3 统计

### 3.1 总量

| 区 | 缺口数 |
|---|---:|
| 全局与命令面板（G） | 23 |
| 主窗口（M） | 26 |
| 右栏（P） | 18 |
| 设置（S） | 27 |
| **合计** | **94** |

### 3.2 按「能否不改服务端实现」

| 判定 | 条数 | 占比 |
|---|---:|---:|
| 能（纯 UI / 本地状态 / 接已有读数与 Intents） | 34 | 36.2% |
| 需裁决（先定归属或产品范围） | 18 | 19.1% |
| 不能（需新事实字段或新协议方法） | 42 | 44.7% |

### 3.3 按缺因

| 缺因 | 条数 | 说明 |
|---|---:|---|
| 无（纯 UI 未做） | 30 | 读数与通道都在，差视图这一层；也是「能」的绝大部分 |
| 事实字段 | 25 | 含「本机已有但没进 Source」（D7 preview、`settings.projects`、`host.Updater`）与「根本没有这条事实」（归档清单、产物、上下文用量） |
| 协议方法 | 28 | 集中在 git 写、会话搜索、MCP 写与认证、`runtime.*`/`workflow.*`/`modality.*`、设置写入、编辑运行中消息 |
| 宿主能力 | 11 | 目录选择器、多窗口、隔离容器、浮窗、通知授权、截图权限、Windows 终端 |
| **合计** | **94** | — |

> 三类缺因合计 64 条（= 94 − 30 条纯 UI）。一行只记**主**缺因；「事实字段 + 协议方法」双缺的行（P-03、M-20、S-07、S-08 等）记在事实字段或协议方法一侧，另一侧写进备注。

### 3.4 三个集中面

1. **设置 19 节里 17 节没有读路径**（S-01）：不是 17 个独立缺陷，是一个结构性缺口——`render*` 签名不带 `*App`，接线前所有设置屏都只能读屏内状态。它同时解释了 S-02/S-06/S-09 为什么还在用样例。
2. **Intents 已实现、UI 零调用**共 16 个：`SetPermissionTier`、`SelectModel`、`CreateProvider`、`UpdateProvider`、`ProbeProvider`、`SetProviderModelEnabled`、`WaitLogin`、`CancelLogin`、`SetExtensionEnabled`、`UninstallExtension`、`TestMCP`、`ImportMCP`、`SetScheduleEnabled`、`DeleteSchedule`、`CancelModalityTask`、`RetrySubmit`。这是最便宜的一批（`internal/uiwire/adapter.go` 里已有完整实现与错误分类）。
3. **「样本数据源」集中在四处**：`modelSamples()` / `skillSamples()`（`internal/ui/composer.go:513`、`:567`）、`rightSample()`（`rightpanel_sample.go:37`）、`subagentSamples()`（`rightpanel_subagent.go:25`）、`mcpServers`（`settings_mcp_screen.go:193`）。前三处的真实读数都已存在。

---

## 附 A 清单侧缺口（不计入上表）

来自两份核对结论，属「清单/证据」缺口而非实现缺口；不补齐会影响需求条目可验收性。

| 编号 | 项 | 出处 | 归属 | 建议 |
|---|---|---|---|---|
| C-01 | 主区空态缺 5 行（品牌 Logo、开始构建、控制行容器、工作区选择器、权限选择器渲染点） | `inventory-main-gaps.md` G-01…G-05 | 清单 owner | 按该文「一句话修法」补行 |
| C-02 | 主区空态多 2 组（标题栏组 5 行、状态条整区需补条件） | 同上 M-01…M-02 | 清单 owner | 条件列补「有会话」 |
| C-03 | 个性化缺 2 行（助手模型不可用说明、子代理 Agent 选择三态） | `inventory-settings-gaps.md` SG-01…SG-02 | 清单 owner | 新增 S-personalization-17b、34b |
| C-04 | 界面模式两张卡的「默认」列被读成「都不高亮」 | 同上 SG-03 | 清单 owner | 默认列改成「编程模式」 |
| C-05 | 全量需求的判据主语错配（抽样 13/80，外推约 2,600 条） | `review-final.md` §5、§6 | 需求 owner | 按「期望讲位置→判据量坐标」逐条对改 |
| C-06 | 四份需求 57.7%–82.1% 走 `判据TBD；比对项为设计基线` | 同上 §5 要点 3 | 组织 + 基线 owner | 基线文档落地前，这批条目只能「按设计值判」 |
| C-07 | 全局清单证据列全缺（425 项只有源码定位，无裁片） | `inventory-global.md`「未收工项」1 | 组织 | 该屏在补齐前不进行为需求阶段 |

## 附 B 边界说明

- 本清单只判「有没有事实/通道/控件」，不判视觉与几何；像素类结论在 `docs/domains/requirements/design-baseline.md` 与各屏截图。
- 「不能」不等于「不做」：其中 P-14、S-26、S-11 依赖上游发布物或 adapter，客户端无法自行解锁；G-20、P-03、P-04 属 D7 明确不做的写路径与明细，是否改边界需要组织先裁。
- 「需裁决」的 18 条有共同形态：**本地登记 vs 服务端事实**（置顶、快捷键、通知、外观、代理）与 **产品范围**（多窗口、Windows 终端、首次运行与欢迎卡、素材资产）。这些不解决，实现会先写错归属。
