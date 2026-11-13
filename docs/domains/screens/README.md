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

**能证明什么**：macOS 红绿灯由系统画在左栏顶部的让位条里；没有系统标题栏；标题行是自绘的、左端是会话名；窗口圆角与阴影由系统给。
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
**不能证明什么**：系统窗口装饰、真实指针形状、真实输入法与屏读器行为。

## 三、重新产出

```bash
# 软件渲染（会重写 state-*、variant-*、titlebar-closeup、wordless-check）
go test ./internal/ui -run 'TestScreens|TestTitleBarCloseup|TestWordlessCheck' -update

# 真窗口（逐态启动、按窗口号截图；需要屏幕录制权限）
go build -o /tmp/dimagent-ui ./cmd/dimagent-ui
python3 scripts/capture-windows.py

# 基线量测（同一批图上的数值）
python3 scripts/ui-baseline.py docs/domains/screens/state-default.png
```

窗口号取自 `CGWindowListCopyWindowInfo`（`/tmp/winid.swift` 的等价脚本在 `scripts/capture-windows.py` 里），按 owner `dimagent-ui` 与窗口名里的「界面骨架」认窗口；不依赖辅助功能权限。
