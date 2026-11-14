# D14 界面接入与交付

Status: 现行（2026-11-14）
Owner: D14 界面接线（`internal/uiwire`、`internal/ui` 的接线部分、`cmd/dimagent-ui`）
依据: `docs/project-organization.md` §5 结果包、`docs/domains/U1-ia.md`、`docs/domains/D1-connection.md`、`docs/domains/D11a-settings.md`、`docs/domains/D11b-extensions.md`、`docs/domains/D9-terminal.md`

本批把已经完成的原生界面接到集成层：界面只从 `internal/ui.source.Source` 读事实，写入口只走 `ui.Intents`；适配器 `internal/uiwire.App` 同时 import 两边，只做映射与转调，不新增协议通道、不攒第二份事实。

## 1. 构建与运行

前置：Go 工具链（版本见 `go.mod`）；macOS（窗口与 Unix socket）。不需要任何第三方依赖。

```bash
go build ./...
```

### 1.1 样例模式（`-demo`，默认）

不连 App Server，只用静态样例画六个区域与五种状态；截图与走查用这一模式。

```bash
go run ./cmd/dimagent-ui
go run ./cmd/dimagent-ui -state running
go run ./cmd/dimagent-ui -variant settings
```

### 1.2 连接真实 App Server（`-connect`）

`-server` 指向 App Server 可执行文件；`-mock-model` 让服务端用 mock 模型，不需要账号与网络。
`-data-dir` 是**客户端数据目录**：端点按真实客户端的规则落在 `<数据目录>/rust/daemon.sock`。

```bash
go run ./cmd/dimagent-ui \
  -connect \
  -data-dir <数据目录> \
  -server <app-server 可执行文件> \
  -mock-model
```

服务端已在 `<数据目录>/rust/daemon.sock` 上监听时，`daemon.Config` 不会另起进程：可以直接把界面指向一个已经运行的实例。

`-connect` 下点了会发请求的入口：发送、停止、审批三决定、提问回答/跳过、打开/新建会话、重命名、归档、导出、删除、插件/技能/钩子启停、插件/技能/MCP 卸载、MCP 启停/测试/从来源导入、定时任务启停/删除、`!` 命令运行与终端中止、权限档位切换。其余见 §5。

### 1.3 连接假 App Server（截图与端到端用）

假服务端与真实服务端共用同一条协议路径与同一套 protocol v3 帧（`internal/fakeappserver`）。

```bash
# 终端 1：假服务端（streaming 或 approval）
go run ./cmd/dimagent-fake-server -dir <数据目录> -scenario streaming

# 终端 2：界面
go run ./cmd/dimagent-ui -connect -data-dir <数据目录>
```

`-scenario streaming` 在会话 attach 后推三条 `step/text` 事实；`-scenario approval` 在 attach 后推一条 `approval.request`。

## 2. 依赖

| 依赖 | 用途 | 缺失时 |
|---|---|---|
| Go 工具链 | 构建、测试 | 无法构建 |
| App Server 可执行文件 | `-connect` 的真实服务端 | 界面显示连接失败原因；不白屏 |
| mock 模型（`-mock-model`） | 真实服务端不接账号/网络 | 无账号时提交与流式无法验证 |
| macOS + Unix socket | 窗口、`screencapture`、`daemon.sock` | Windows 命名管道未实现，假服务端直接报错 |

真实服务端的已有缺口：alpha.21 未注册 `session.permission.update`（`-32601 unknown_method`），attach 回执也不带权限档位，因此真实服务端的**提交段**在当前机器上 blocked；连接、目录、默认态已通过（见 §5）。

## 3. 已接（读事实）

| 区域 | 入口 | 状态 |
|---|---|---|
| 连接阶段与断线原因 | `Source.Connection` | 已接；断线时输入位禁用并在原位给原因；控制连接关闭后阶段转 `disconnected`（`TestDisconnectSurfacesPhase`） |
| 左栏会话列表 | `Source.SessionNames`（目录快照） | 已接 |
| 时间线历史与实时事实 | `Source.Session`（投影行） | 已接；服务端 `session.event` 到达后行数增长 |
| 待决审批 / 提问 | `Source.Approval` / `Source.Question` | 已接；命令动作描述来自通知 |
| 设置 · 账号与套餐 | `Source.Account` / `Source.Usage` | 读服务端账号状态、身份与用量聚合项 |
| 设置 · 模型与供应商 | `Source.Settings`（provider/model/权限档位） | 读模型目录快照；档位来自 `session.permission.get` |
| 设置 · 扩展 | `Source.Extensions`（四类） | 读 `plugin.list` / `skills.catalog` / `hooks.catalog` / `mcp.catalog` 快照 |
| 设置 · 定时任务 | `Source.Schedules` | 读 `schedule.list` 快照 |
| 设置 · 运行时与模态 | `Source.Settings`（runtimes/modality） | 读最近一次成功读数；能力缺失显示原因 |
| 设置 · 存储与诊断 | `Source.Settings`（storage） | 读 `storage.report` 读数；导出/打开未接 |
| 右栏终端状态 | `Source.Terminal` | 读 `internal/terminal.Registry` 的存活/退出状态 |
| 连接后初始读数 | `internal/uiwire.Connect → loadFacts` | 账号、模型、存储、运行时、模态、扩展四类、定时任务各读一次；失败留"未读取" |

连接的六阶段顺序保持集成层的约定：`LoadLocalState → Connect → Subscribe → FirstHistory → AllowWrites → Catalog.Refresh`，之后才是一组可选读数；任一项失败都不改变"已连接"阶段，只让对应快照显示未读取或服务端原文。

## 4. 已接动作（点了就发请求）

每条都有 `internal/uiwire/intents_test.go` 的「点了什么 → 服务端收到什么」测试；测试名写在最后一列。

| 视图元素 | `ui.Intents` | app / 领域方法 | 服务端方法 | 测试 |
|---|---|---|---|---|
| 发送槽 | `Submit` | `app.Submit` | `session.submit` | `TestIntentSubmit` |
| 停止槽 | `Stop` | `app.Stop`（本批新增：主线停止） | `session.stop` | `TestIntentStop` |
| 审批 · 允许一次 / 一直允许 / 拒绝 | `RespondApproval` | `app.Respond` | `approval.respond` | `TestIntentApproval` |
| 提问 · 回答 / 跳过 | `RespondQuestion` / `SkipQuestion` | `app.Respond` | `question.respond` | `TestIntentQuestion` / `TestIntentQuestionSkip` |
| 左栏会话行 | `OpenSession` | `app.OpenSession` | `session.attach` | `TestIntentOpenSession` |
| 左栏「新建会话」 | `CreateSession` | `app.CreateSession`（工作目录沿用当前会话） | `session.create` | `TestIntentCreateSession` |
| 会话菜单 · 重命名 | `RenameSession` | `catalog.Catalog.Rename` | `session.rename` | `TestIntentRename` |
| 会话菜单 · 归档 | `ArchiveSession` | `catalog.Catalog.Hide/Unhide` | 无（本地归档） | `TestIntentArchive` |
| 会话菜单 · 删除（两步确认） | `DeleteSession` | `catalog.Catalog.Delete` | `session.delete` | `TestIntentDelete` |
| 会话菜单 · 导出 | `ExportSession` | `catalog.Catalog.Export` | 读历史后写本地 `.md` | `TestIntentExport` |
| 设置 · 权限档位（两步确认） | `SetPermissionTier` | `app.SetPermissionTier` | `session.permission.update` | 界面已发，真实服务端缺该方法 |
| 设置 · 插件 / 技能启停 | `SetExtensionEnabled` | `extensions.SetPluginEnabled/SetSkillEnabled` | `plugin.enable/disable`、`skills.set_enabled` | `TestIntentExtensionToggle` |
| 设置 · 钩子单项启停 | `SetExtensionEnabled("hook")` | `extensions.WriteHook` | `hooks.save` | `TestIntentHookToggle` |
| 设置 · 定时任务启停 | `SetScheduleEnabled` | `extensions.SetScheduledTaskEnabled` | `schedule.set_enabled` | `TestIntentScheduleToggle` |
| 设置 · 定时任务删除（两步确认） | `DeleteSchedule` | `extensions.DeleteScheduledTask` | `schedule.delete` | `TestIntentScheduleDelete` |
| 扩展详情 · 插件卸载 | `UninstallExtension("plugin")` | `extensions.RemovePlugin` | `plugin.remove` | `TestIntentPluginUninstall` |
| 扩展详情 · 技能卸载 | `UninstallExtension("skill")` | `extensions.RemoveSkill` | `skills.remove` | 与插件卸载同一路径 |
| MCP · 启停 | `SetExtensionEnabled("mcp")` | `extensions.SetToolServerEnabled` | `mcp.set_enabled` | `TestIntentMCPToggle` |
| MCP · 卸载 | `UninstallExtension("mcp")` | `extensions.RemoveToolServer` | `mcp.remove` | `TestIntentMCPUninstall` |
| MCP · 连接测试 | `TestMCP` | `extensions.ProbeToolServer` | `mcp.test` | `TestIntentMCPTest` |
| MCP · 从来源导入 | `ImportMCP` | `extensions.PreviewMCPImport` → `ApplyMCPImport` | `mcp.import.preview` → `mcp.import.apply` | `TestIntentMCPImport` |
| 输入区 `!命令` · 运行 | `RunShell` | `app.RunShell`（等回执取运行标识） | `shell.run` | `TestIntentShellRun` |
| 右栏终端 · 中止 | `StopShell` | `app.StopShell` | `shell.stop` | `TestIntentShellStop` |

MCP 导入的来源来自 `mcp.import.sources`（连接后读一次，界面列出来源）；点一个来源先 `mcp.import.preview`，再按预览的 `revision` / `sourceRevision` 发 `mcp.import.apply`。插件/技能/MCP 的标识用服务端 ID，不用展示名。

失败时不会静默：动作在后台协程执行，错误原因显示在内容区顶部的一行（带关闭入口）。

## 5. 未接动作（blocked，缺什么）

| 视图元素 | 缺什么 | 当前表现 / 谁能解除 |
|---|---|---|
| 目标 · 保存 | 协议包没有 goal 写方法；`projection.Goal` 只读，`app` 没有 `SetGoal` | 目标行改为只读展示；需要服务端先定义目标写方法，再由 D1/D2 接入 |
| 设置 · 运行时提交模态任务 | 服务端未注册 `runtime.*`（D1 §7.1 已记）；`settings.Manager.SubmitTask` 还需要 `category` / `operation`，界面只有任务文本 | 提交按钮禁用并显示 "blocked：服务端未注册 runtime.* / 模态任务提交方法"；D1 恢复能力后再接 |
| 设置 · 主题 / 动效 / 时区落盘 | `app` 没有持久化入口 | 保留为视图状态，重启丢失；不偷偷写文件。需要确定本地偏好归属后再接 |

范围外（本轮 A/B 清单未要求，仍是只读或视图状态）：provider / model 选择、命令规则读写、终端启动 / 重连 / 清屏 / 复制、附件 / 分支切换 / 后台任务。

## 6. 已知边界

1. **真实服务端缺权限方法**。alpha.21 的 `session.permission.update` 返回 `-32601 unknown_method`，attach 回执不带权限档位；`internal/uiwire/real_test.go` 在连接、目录、默认态之后把提交段标为 blocked 并打印服务端原文，不当作通过。
2. **Windows / Linux 未验证**。假服务端只支持 Unix socket（Windows 返回明确错误）；窗口形态、红绿灯让位与截图证据都只在 macOS 上产出。
3. **tooltip / 悬停 / 按下态不在截图证据内**。本会话进程无法成为前台（`isActive=false`，窗口 `onscreen=false`），系统 tooltip 不出现；这部分以 `internal/ui` 的测试断言代替。
4. **窗口只提交首帧**。框架对不可见窗口会挂起后续帧；`connect-streaming.png` 与 `connect-approval.png` 取的是事实到达后、窗口首帧的状态，不是逐帧动画。流式增量由端到端测试断言。
5. **重绘靠 250ms 轮询**。集成层目前没有"事实变化"回调，`cmd/dimagent-ui` 在 `-connect` 下每 250ms `Invalidate` 一次；视图每帧仍只从 `Source` 读，不缓存事实。
6. **截图与真实服务的差异**。`connect-default.png` 用真实 App Server + mock 模型；另外两张用假 App Server，协议路径与帧格式一致，但服务端行为是脚本化的，不代表真实模型的输出。

## 7. 验收命令与预期输出

```bash
# 1) 五步端到端（假服务端）：列会话 → 历史行 → 提交 → 流式增长 → 审批出现/应答/消失
go test ./internal/uiwire -run EndToEnd -v -count=1
# 预期：=== RUN   TestEndToEndFiveSteps
#       ① 目录会话 1 个，首项 "s1"
#       ② 打开 "s1"，历史 2 行
#       ③ 提交已发出（session.submit 1 次）
#       ④ 时间线 2 → 3 行
#       ⑤ 审批出现："exec：启动进程"
#       ⑤ 审批已应答并消失
#       --- PASS: TestEndToEndFiveSteps
#       TestRealServerConnectDefault 在真实服务端存在时运行：连接/目录/默认态通过，
#       提交段 SKIP 并打印 "session.permission.update -32601 unknown_method"

# 2) 断线可复现：服务端断开全部连接后阶段转 disconnected，并带原因
go test ./internal/uiwire -run TestDisconnectSurfacesPhase -v -count=1
# 预期：断线原因：app server connection closed: EOF；--- PASS

# 3) 点的动作真的发请求：每条都打印方法名与次数
go test ./internal/uiwire -run TestIntent -v -count=1
# 预期（节选）：
#   点了「发送」→ 服务端收到 session.submit（1 次），text=你好
#   点了「停止」→ 服务端收到 session.stop（1 次），sessionId=s1
#   点了「允许一次」→ 服务端收到 approval.respond（1 次），decision=allow
#   点了「回答」→ 服务端收到 question.respond（1 次）
#   点了「s2」→ 服务端收到 session.attach（2 次），当前会话=s2
#   点了「新建会话」→ 服务端收到 session.create（1 次），cwd=/work
#   点了「重命名 · 保存」→ 服务端收到 session.rename（1 次）
#   点了「归档」→ 目录列表 0 个会话（本地隐藏，不发服务端请求）
#   点了「删除 · 确认」→ 服务端收到 session.delete（1 次）
#   点了「导出」→ 本地写出 dimagent-export-<时间戳>.md
#   点了「插件 · 启用开关」→ 服务端收到 plugin.disable（1 次）
#   点了「插件 · 卸载」→ 服务端收到 plugin.remove（1 次）
#   点了「钩子 · 开关」→ 服务端收到 hooks.save（1 次）
#   点了「定时任务 · 开关」→ 服务端收到 schedule.set_enabled（1 次），job_id=job-1 enabled=false
#   点了「定时任务 · 删除」→ 服务端收到 schedule.delete（1 次），job_id=job-1
#   点了「MCP · 启用开关」→ 服务端收到 mcp.set_enabled（1 次）
#   点了「MCP · 测试」→ 服务端收到 mcp.test（1 次）
#   点了「MCP · 卸载」→ 服务端收到 mcp.remove（1 次）
#   点了「从 Cursor 导入」→ mcp.import.preview（1 次），from=src-1
#     随后 mcp.import.apply（1 次），from=src-1 expectedVersion=rev-2 sourceRevision=src-rev-1
#   点了「运行命令」→ 服务端收到 shell.run（1 次），command=echo hi
#   点了「终端 · 中止」→ 服务端收到 shell.stop（1 次），runId=run-srv-1
#   --- PASS: TestIntent...（22 条）

# 4) 接上事实后不出现样例常量（逐屏渲染断言）
go test ./internal/ui -run TestWiredScreensExcludeSamples -v -count=1
# 预期：--- PASS: TestWiredScreensExcludeSamples

# 5) 构建、静态检查、全量测试
go build ./...
go vet ./...
go test ./... -count=1
# 预期：全部 ok；uiwire 的真实服务端测试按 §6.1 标 blocked

# 6) 界面检查器：20 个检查项与 14 条种子
scripts/ui-lint
# 预期：检查器全绿：20 个检查项；--- PASS: TestUILint
scripts/ui-lint --seeds
# 预期：14 条种子逐条报出；--- PASS: TestUILintSeeds
```

## 8. 产物位置

| 文件 | 内容 |
|---|---|
| `internal/uiwire/adapter.go` | `Source` / `Intents` 适配器；连接后可选读数 |
| `internal/ui/source.go` | 界面侧的只读快照契约（会话、审批、账号、设置、扩展、定时任务、终端） |
| `internal/ui/app.go` | `-connect` 下只重置视图状态、不灌样例；后台意图与错误行 |
| `internal/ui/composer.go` | 发送/停止/审批三决定/提问回答与跳过 |
| `internal/ui/sidebar.go` | 新建、打开、重命名、归档、导出、删除 |
| `internal/ui/settings*.go`、`extensions*.go`、`terminal.go` | 四块改读 `Source` 快照；权限档位、扩展开关/卸载/测试、终端运行/中止 |
| `cmd/dimagent-ui/main.go` | `-connect` 模式、`OnChange` 重绘、250ms 刷新 |
| `cmd/dimagent-fake-server/main.go` | 截图用的假服务端（dev-only，不是产品入口） |
| `internal/uiwire/e2e_test.go` | 五步端到端、断线 |
| `internal/uiwire/intents_test.go` | 点击 → 服务端收到 / 本地写出（22 条动作） |
| `internal/app/session.go` | `App.Stop`：主线停止的写意图入口 |
| `internal/app/sender.go`、`internal/app/extensions.go` | `plugin.remove` / `schedule.*` 分派、`mcp.import.sources/preview` 读取 |
| `internal/extensions/*.go` | 插件/技能/钩子/MCP 启停与卸载、定时任务启停/删除、MCP 导入来源与预览 |
| `internal/uiwire/real_test.go` | 真实服务端：连接 + 目录 + 默认态；提交段 blocked |
| `docs/domains/screens/connect-{default,streaming,approval}.png` | 三种连接态的截图（来源见 `docs/domains/screens/README.md`） |
