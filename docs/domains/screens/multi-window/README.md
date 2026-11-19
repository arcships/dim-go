# 多窗口与深链的截图证据

由 `cmd/dimagent-ui` 产出（连接模式；服务端是 dev-only 的假 App Server，协议路径与
真实一致）。三组事实：

| 文件 | 事实 |
|---|---|
| `two-windows.png` / `two-windows-2.png` | 同进程两个窗口各开一个会话：主窗口是「发送路径排队」，第二个窗口是「参考实现对照」；两个窗口各自选中、内容互不串 |
| `deeplink-session-reused.png` | `dimagent://session/<会话>` 落在已有窗口上：不新开窗口，主窗口换到该会话 |
| `deeplink-settings.png` | `dimagent://settings/<分节>` 落在已有窗口上：主窗口进设置屏的「通知」节 |
| `deeplink-unknown.png` | 未知目标给可读原因：回执行写「未知的链接目标 …：可用 dimagent://session/<会话> 或 dimagent://settings/<分节>」，不开窗口 |

产出方法（仓库根下执行；`<数据目录>` 是临时目录，`<服务端>` 是假 App Server
可执行文件）：

```bash
go build -o /tmp/dimagent-ui ./cmd/dimagent-ui
go build -o /tmp/dimagent-fake-server ./cmd/dimagent-fake-server
/tmp/dimagent-fake-server -dir <数据目录> -scenario streaming &   # 先起服务端

# 双窗口：-new-window 在第二个窗口打开另一个会话
/tmp/dimagent-ui -data-dir <数据目录> -no-focus -width 880 -height 680 \
  -screenshot <输出>/two-windows.png -new-window s2

# 深链：会话链接复用已有窗口（只出一张图，说明没有第二个窗口）
/tmp/dimagent-ui -data-dir <数据目录> -no-focus -width 880 -height 680 \
  -screenshot <输出>/deeplink-session-reused.png dimagent://session/s2

# 深链：设置分节
/tmp/dimagent-ui -data-dir <数据目录> -no-focus -width 880 -height 680 \
  -screenshot <输出>/deeplink-settings.png dimagent://settings/notifications

# 未知目标：回执行上的可读原因
/tmp/dimagent-ui -data-dir <数据目录> -no-focus -width 880 -height 680 \
  -screenshot <输出>/deeplink-unknown.png dimagent://nope/thing
```

`-screenshot` 按窗口顺序写：第一个窗口用给定路径，其余加 `-2`、`-3` 后缀，一次跑完
全部窗口。系统交付的深链（macOS 的 `openURLs`，应用未运行时由系统拉起后到达）与
命令行上直接给的 `dimagent://` 走同一条路由；窗口位置尺寸按窗口索引记在本地偏好
的 `windows` 数组里（0 是主窗口）。
