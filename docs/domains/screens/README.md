# 界面截图：两类证据

Status: 现行（2026-11-13）
Owner: U2 界面视觉（由 U5 实现产出、U2 判读）
依据: `docs/project-organization.md` §9.10（可视图产物身份）、§9.14（交付物）、`docs/domains/U2-forms.md`（形式库与基线表）

这里有两类图，用途不同，**不能互相替代**。看图前先确认它是哪一类。

## 一、真窗口（`window-*.png`）

由 `scripts/capture-windows.py` 截取：`go build` 出二进制，逐态启动，按窗口号用
`screencapture -x -o -l <窗口号>` 抓窗口本身（`-o` 去掉阴影）。Retina 下是 2x 像素，
文件名里的尺寸按逻辑像素（DIP）算。

| 文件 | 逻辑尺寸 | 内容 |
|---|---:|---|
| `window-default.png` | 1440×900 | 默认态（真红绿灯、自绘标题行） |
| `window-running.png` | 1440×900 | 运行中（队列条、停止槽） |
| `window-waiting.png` | 1440×900 | 等待你同意（审批卡占输入位） |
| `window-failed.png` | 1440×900 | 失败 / 结果不确定 |
| `window-empty.png` | 1440×900 | 空态 |
| `window-default-900.png` | 900×900 | 三档宽度对照 · regular |
| `window-default-720.png` | 720×900 | 三档宽度对照 · compact 边界 |
| `window-titlebar-closeup.png` | 1440×120 | 标题行特写（红绿灯、会话名、无系统标题栏、无分隔线） |
| `window-default-variant-branch{,-900,-720}.png` | 1440/900/720×900 | 分支切换：标题行分支标记 + 原生菜单（三分支状态之一） |
| `window-default-variant-goal{,-900,-720}.png` | 1440/900/720×900 | 目标设置中：面差输入框 + 「保存 / 取消」文字按钮 |
| `variant-rich.png`（软件渲染 1x） | 1440×900 | 富文本回复：h2/h3、两种列表、行内代码、代码块、引用、表格、链接 |
| `window-default-variant-branch-end{,-900,-720}.png` | 1440/900/720×900 | 分支 · 结束分支确认态：标记换成「确认结束分支？」+「取消」 |
| `baseline/controls-{1,2}x.png` | 320×620 | 官方控件的对照表（mygo 默认主题，逐控件一行） |
| `baseline/compare-{default,waiting,settings}.png` | 见各图 | 三屏并排：左**官方示例真实切片**（`calibration/examples/`，坐标写在图上与 `U0-control-baseline.md` §3）/ 右我们同尺寸切片（`scripts/compare-screens.py`） |
| `window-running-variant-tasks{,-900,-720}.png` | 1440/900/720×900 | 后台任务运行中：右栏「后台」标签下的任务清单（720 时右栏整屏替代主区） |
| `window-default-variant-settings{,-900,-720}.png` | 1440/900/720×900 | 设置屏 · **外观分节**（主题下拉 + 动效开关）；左导航 224、顶部返回入口 |
| `window-default-variant-settings-rules{,-900,-720}.png` | 1440/900/720×900 | 设置屏 · **命令规则分节**，停在**清空确认态**（危险动作两步） |
| `window-default-variant-ext{,-900,-720}.png` | 1440/900/720×900 | 设置 · **扩展**分节：四页签 + 插件列表（8 行，名称/版本/状态点/开关） |
| `window-default-variant-ext-failed{,-900,-720}.png` | 1440/900/720×900 | 设置 · 扩展**读取失败态**：叹号 + 原因 + 重试入口 |
| `window-default-variant-tier-confirm{,-900,-720}.png` | 1440/900/720×900 | 设置 · 权限档位**两步确认** |
| `window-default-variant-ext-detail{,-900,-720}.png` | 1440/900/720×900 | 扩展**详情**（二级页）：描述、来源、能力、启停、卸载 |
| `window-default-variant-ext-select{,-900,-720}.png` | 1440/900/720×900 | 扩展**选择态**：复选框 + 工具条（已选 2 / 启用 / 停用 / 取消） |
| `window-default-variant-mcp-test{,-900,-720}.png` | 1440/900/720×900 | MCP **连接测试**：进行中 + 成功两行 |
| `window-default-variant-mcp-fail{,-900/720}.png`（含 900/720） | 1440/900/720×900 | MCP **测试失败** + 导入回执 |
| `window-default-variant-shell{,-900,-720}.png` | 1440/900/720×900 | 输入区**命令模式（`!`）空闲**：标记「! 命令」+ 退出 + 三角运行槽 |
| `window-default-variant-shell-run{,-900,-720}.png` | 1440/900/720×900 | 输入区命令模式**运行中**：右栏终端同步显示命令（720 档右栏整屏替代主区，此时只看到终端——输入区的标记在窄窗被整屏替换，见 U1-ia §3.1） |
| `window-default-variant-storage{,-900,-720}.png` | 1440/900/720×900 | 设置 · **存储与诊断**：三条读数（带单位）+ 服务版本 + 导出回执 |
| `window-default-variant-appearance-keys{,-900,-720}.png` | 1440/900/720×900 | 设置 · **外观与键位**：主题/动效/时区 + 快捷键提示表（9 行） |
| `window-default-variant-cron{,-900,-720}.png` | 1440/900/720×900 | 设置 · **定时任务**：四行任务（名称/计划/下次/开关/删除） |
| `window-default-variant-cron-delete{,-900,-720}.png` | 1440/900/720×900 | 设置 · 定时任务**删除两步确认** |
| `window-default-variant-runtime{,-900,-720}.png` | 1440/900/720×900 | 设置 · **运行时与模态**：组件清单 + 模态能力 + 提交入口 |
| `window-default-variant-runtime-submit{,-900,-720}.png` | 1440/900/720×900 | 设置 · 运行时：模态任务**提交回执** |
| `window-default-variant-session-menu{,-900,-720}.png` | 1440/900/720×900 | 左栏头部「…」的**会话动作菜单**（重命名/归档/导出/删除） |
| `window-default-variant-rename{,-900,-720}.png` | 1440/900/720×900 | 左栏**重命名行内编辑**：面差输入 + 保存/取消 |
| `window-default-variant-session-delete{,-900,-720}.png` | 1440/900/720×900 | 左栏**删除两步确认**：确认句 + 删除/取消 |
| `window-default-variant-search{,-900,-720}.png` | 1440/900/720×900 | 左栏**搜索面**：放大镜 + 面差输入 + 过滤后的列表 |
| `window-default-variant-term{,-900,-720}.png` | 1440/900/720×900 | 右栏 · **终端（空闲）**：状态点 + 清屏/复制 + 提示行 |
| `window-default-variant-term-running{,-900,-720}.png` | 1440/900/720×900 | 右栏 · **终端（运行中）**：状态点 accent + 中止入口 |
| `window-default-variant-term-disconnected{,-900,-720}.png` | 1440/900/720×900 | 终端**断线**：点 danger + `connection lost` + 重新连接入口 |
| `window-default-variant-term-recovered{,-900,-720}.png` | 1440/900/720×900 | 终端**恢复**：回空闲 + `reconnected` |
| `window-default-variant-term-exited{,-900,-720}.png` | 1440/900/720×900 | 右栏 · **终端（已退出）**：ANSI 颜色、长行折行、多行错误 |
| `connect-default.png` | 1440×900 | `-connect` 连**真实 App Server**（mock 模型）的默认态；无样例数据 |
| `connect-streaming.png` | 1440×900 | `-connect` 连**fake App Server**，首屏历史含流式文本、会话运行中的一帧 |
| `connect-approval.png` | 1440×900 | `-connect` 连**fake App Server**，服务端推送 `approval.request` 后的一帧（审批卡占输入位） |

**`connect-*.png` 的来源**：三张都由 `cmd/dimagent-ui -connect -data-dir <目录>` 产出，界面只从 `internal/uiwire` 的 `Source` 读事实，没有样例数据进入视图。`connect-default.png` 的服务端是真实 App Server（路径用 `<app-server 可执行文件>` 占位）；`connect-streaming.png` 与 `connect-approval.png` 的服务端是 **fake App Server**（`internal/fakeappserver`，由 `cmd/dimagent-fake-server` 启动），**协议路径与真实一致**（同一个 `<data-dir>/rust/daemon.sock`、同一套 protocol v3 帧）。流式增量与审批出现/应答/消失的逐条断言见 `go test ./internal/uiwire -run EndToEnd -v`，不只靠截图。
**取帧限制**：本会话窗口 `onscreen=false`，框架对不可见窗口只提交首帧；两张 fake 图取的是窗口首帧的状态，不是逐帧动画。流式文本放在首屏历史里，实时增量由 `internal/uiwire` 的端到端测试断言。界面在可见窗口里每 250ms 重绘一次，把订阅收到的变化画出来。

**能证明什么**：macOS 红绿灯由系统画在左栏顶部的让位条里；没有系统标题栏；标题行是自绘的、左端是会话名；窗口圆角与阴影由系统给。
**外观（主题）**：真窗口截图跟随系统外观——本轮系统为深色，所以 `window-*.png` 全是深色，同一轮评审看到的是同一套观感。软件渲染（`state-*.png` / `variant-*.png`）默认浅色，深色由 `variant-dark.png` 覆盖；两类的对照关系见下表。

| 类别 | 外观 | 覆盖 |
|---|---|---|
| `window-*.png` | 跟随系统（本轮深色） | 五种状态、三档宽度、六个走查态、设置两态 |
| `state-*.png` / `variant-*.png` | 软件渲染默认浅色 | 五个状态与各变体；深色见 `variant-dark.png` |

**不能证明什么**：悬停 / 按下 / 焦点态与 tooltip。本会话里进程无法成为前台（`NSRunningApplication.activate` 返回 true 而 `isActive=false`，窗口 `onscreen=false`），窗口收不到 hover 事件，系统 tooltip 不会出现；这部分以测试断言代替（`TestSendSlotIsIconWithTooltip`、`TestSelfDrawnRowsHaveStates`）。Windows 与 Linux 的形态同样不在证据内。

## 二、软件渲染（`state-*.png`、`variant-*.png`、`wordless-check.png`）

由 `internal/ui/snapshot_test.go` 用框架的无窗口渲染（`ui.NewTester`）产出，1440×900，与真窗口同尺寸但不含系统装饰。

| 文件 | 内容 |
|---|---|
| `state-default.png` / `state-running.png` / `state-waiting.png` / `state-failed.png` / `state-empty.png` | 五个状态 |
| `variant-summary-open.png` | 组头展开 |
| `variant-right-panel.png` | 右栏展开 |
| `variant-sidebar-collapsed.png` | 左栏收起 |
| `variant-context-near.png` | 上下文将满（圆点 + warning） |
| `variant-titlebar.png` | 标题栏让位条（红绿灯位置是截图工具画的示意） |
| `variant-draft.png` | 输入框有草稿（量基线用） |
| `titlebar-closeup.png` | 标题行特写（软件渲染版） |
| `wordless-check.png` | 遮字测试：盖住发送槽文字，检查"读历史 / 提交 / 处理一次审批"是否仍认得出 |

**能证明什么**：每个状态的元素与形式、排版基线的数值（`scripts/ui-baseline.py`）、像素级的悬停/按下取色（测试里断言）、遮字后仍可完成首片任务。
**外观（主题）**：真窗口截图跟随系统外观——本轮系统为深色，所以 `window-*.png` 全是深色，同一轮评审看到的是同一套观感。软件渲染（`state-*.png` / `variant-*.png`）默认浅色，深色由 `variant-dark.png` 覆盖；两类的对照关系见下表。

| 类别 | 外观 | 覆盖 |
|---|---|---|
| `window-*.png` | 跟随系统（本轮深色） | 五种状态、三档宽度、六个走查态、设置两态 |
| `state-*.png` / `variant-*.png` | 软件渲染默认浅色 | 五个状态与各变体；深色见 `variant-dark.png` |

**不能证明什么**：系统窗口装饰、真实指针形状、真实输入法与屏读器行为。

## 三、重新产出

```bash
# 软件渲染（会重写 state-*、variant-*、titlebar-closeup、wordless-check）
go test ./internal/ui -run 'TestScreens|TestTitleBarCloseup|TestWordlessCheck' -update

# 真窗口（逐态启动、按窗口号截图；需要屏幕录制权限）
go build -o /tmp/dimagent-ui ./cmd/dimagent-ui
python3 scripts/capture-windows.py   # 五个状态 + 三档宽度 + 二十八个走查态（branch / branch-end / goal / tasks / settings / settings-rules / ext / ext-failed / term / term-running / term-exited）

# 基线量测（同一批图上的数值）
python3 scripts/ui-baseline.py docs/domains/screens/state-default.png
```

窗口号取自 `CGWindowListCopyWindowInfo`（`/tmp/winid.swift` 的等价脚本在 `scripts/capture-windows.py` 里），按 owner `dimagent-ui` 与窗口名里的「界面骨架」认窗口；不依赖辅助功能权限。
