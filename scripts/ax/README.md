# Accessibility 驱动（macOS）

这两个 Swift 工具让真实窗口验收可以在脚本里复现：不修改应用，只通过 macOS
Accessibility 发送点击/按键，再配合 `screencapture -l <windowid>` 截图。

前置条件：

- 终端进程已获得「辅助功能」权限（`AXIsProcessTrusted() == true`）。
- 已安装 Xcode command line tools（`swiftc`）。

编译：

```bash
swiftc -o /tmp/axdo scripts/ax/axdo.swift
swiftc -o /tmp/windowid scripts/ax/windowid.swift
```

用法（`<pid>` 是 `go run ./cmd/dimagent` 编译出的真实进程，不是 `go run` 本身）：

```bash
/tmp/axdo dump <pid>                     # 打印 AX 树
/tmp/axdo click <pid> "New session"      # 按标题/描述触发 AXPress
/tmp/axdo rows <pid> Sessions            # 列出 AXTable 的行
/tmp/axdo clickrow <pid> Sessions 1      # 选中行（部分视图只移动光标）
/tmp/axdo frame <pid> "Composer"         # 元素屏幕坐标与尺寸
/tmp/axdo clickat <pid> 620 452          # 在坐标处发送真实鼠标点击
/tmp/axdo set <pid> "Composer" "text"    # 写入 AXValue
/tmp/axdo key <pid> 125                  # 发送虚拟键码（125 = Down）
/tmp/windowid <pid>                      # 窗口 id 与尺寸，用于 screencapture
```

已知限制：

- `ui.List` 只绑 `Selected` 时，AXPress/坐标点击可能只移动光标；侧栏为此在行上直接发布选择（见 `internal/ui/sidebar.go` 的 `threadRow`）。
- 驱动依赖真实窗口获得焦点；采集脚本会短暂抢占焦点，用完请关闭应用。
- 这里只用于证据采集，不是产品运行时依赖。
