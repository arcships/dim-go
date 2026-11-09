# UI 截图证据（headless）

本目录保存 `internal/ui` 静态壳的 headless 截图，用来证明固定 fixture 在 1440×900 下的渲染是稳定的（两主题、非空、尺寸正确、light/dark 像素不同）。生成代码是 [`internal/ui/screenshot_test.go`](../../../internal/ui/screenshot_test.go)。

## 生成命令

从仓库根目录运行：

```bash
DIMAGENT_UI_SCREENSHOT_DIR="$PWD/docs/evidence/ui" go test ./internal/ui -run TestScreenshotFrames -v
```

- 未设置 `DIMAGENT_UI_SCREENSHOT_DIR` 时，测试只做断言，不写任何文件（`go test ./internal/ui` 保持只读）。
- 设置后写入本目录的 `shell-light.png` 和 `shell-dark.png`，目录不存在会自动创建。
- 截图 API 来自 mygo v0.8.3 模块源码：`ui.NewTester(view, width, height)`、`(*Tester).SetDark(bool)`、`(*Tester).Image() *image.RGBA`。mygo 自带的 `MYGO_UI_PNG` 只服务于该模块自己的 `ui/ui_test.go`，不是公开 API，本项目不读取它。

## 产物

| 文件 | 主题 | 尺寸 | 大小 |
|---|---|---|---|
| [`shell-light.png`](shell-light.png) | Light | 1440×900 | 124,523 B |
| [`shell-dark.png`](shell-dark.png) | Dark | 1440×900 | 126,932 B |

![shell light](shell-light.png)

![shell dark](shell-dark.png)

文件大小随字体栅格化和压缩实现变化，不作为断言；测试断言的是尺寸、非空和两主题像素不同。

## Fixture 内容

固定 fixture 是手填的视图状态，不连 App Server，不读 SQLite/JSONL/session 文件/daemon.log：

- 两个 session：`Fix flaky screenshot test`（选中）、`Review App Server reconnect`。
- 四行对话，覆盖四种 `Author`：
  - user：纯文本；
  - assistant：Markdown（列表 + Go fenced code block）；
  - tool：`bash` 卡片，`Complete` 状态，含 Params/Result 单行预览；
  - approval：`exec` 待审批行，带 `BranchID`/`RequestID` 身份，渲染真实的 `Approve`/`Deny` 按钮（fixture 的回调是空实现，不发请求）。
- composer 有草稿 `Rerun the suite and attach the new PNGs.`。
- `Connection: ConnReady`、`Run: RunIdle`，让状态行和 Send 按钮处于正常态。

## 测试断言

`TestScreenshotFrames` 对 light/dark 两个 `Tester` 分别检查：

1. `Image()` 非 nil，尺寸严格为 1440×900；
2. 画面不止一种颜色（非空白帧）；
3. light 与 dark 的像素缓冲不相同。

只有设置了 `DIMAGENT_UI_SCREENSHOT_DIR` 才额外编码 PNG。

## 证据边界

**截图只证明 headless 渲染稳定，不等于与 Electron 端 1:1。** 它不覆盖真实窗口合成、字体/DPI 差异、系统外观跟随、动效、IME、VoiceOver 或 Windows/Linux 运行时；token 语义层与 Electron 的差距见 [`docs/ui-parity.md`](../../ui-parity.md)。这些 PNG 是渲染回归证据，不是产品验收证据。
