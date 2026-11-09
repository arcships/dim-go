# 发布与打包

Status: active
Last updated: 2026-11-10
适用范围：本仓库的 macOS 本地 `.app` 打包与发布边界。安装、开发和测试命令见 [README](../README.md)；产品验收见 [acceptance.md](acceptance.md)。

## 1. 现状

本仓库还没有发布流水线、Developer ID 签名、公证、安装器或更新通道。当前只有一个本地打包脚本，用来验证真实窗口的 bundle 身份和 `dimagent://` 深链注册：

```bash
scripts/package-macos.sh
```

产物是 ad-hoc 签名的本地开发件，不是可分发版本。功能/视觉验收仍按 [acceptance.md](acceptance.md) 的 L4/L5 标准执行，打包成功不能替代验收。

## 2. macOS 本地 bundle

### 2.1 命令

```bash
scripts/package-macos.sh [--out DIR] [--version VERSION] [--no-sign]
```

| 参数 | 默认值 | 说明 |
|---|---|---|
| `--out DIR` | `/tmp/dimagent-app` | 输出目录；脚本会重建 `DIR/DimAgent.app` |
| `--version VERSION` | `0.1.0` | 写入 `CFBundleShortVersionString` 和 `CFBundleVersion`，并通过 `go build -ldflags "-X main.version=$VERSION"` 写入二进制；只允许字母、数字和 `._+-` |
| `--no-sign` | 不传 | 跳过 ad-hoc 签名 |

脚本从仓库根目录执行 `go build -o DimAgent.app/Contents/MacOS/dimagent ./cmd/dimagent`，生成 `Info.plist` 和 `PkgInfo`，然后按需签名。需要 Go 1.27.3 或更新的 toolchain，见 [README 开发前提](../README.md#开发前提)。

包脚本把同一个 `--version` 通过 `-ldflags "-X main.version=$VERSION"` 写入二进制，因此三个地方读到同一个值：`Info.plist` 的 `CFBundleShortVersionString` / `CFBundleVersion`、`dimagent -version`（或 `--version`）输出的一行，以及应用内 Settings → About 的 Version 行。未打包的 `go build` / `go run` 不传 ldflags，版本保持 `dev`；About 页对空版本同样显示 `dev`，不会留空行。

### 2.2 产物结构

```text
/tmp/dimagent-app/DimAgent.app/
└── Contents/
    ├── Info.plist
    ├── PkgInfo                 # APPL????
    ├── MacOS/
    │   └── dimagent            # go build 产物，可执行
    └── Resources/              # 当前为空；保留给 mygo.PathResources
```

### 2.3 Info.plist 字段

| 字段 | 值 | 用途 |
|---|---|---|
| `CFBundlePackageType` | `APPL` | 声明这是应用 bundle |
| `CFBundleExecutable` | `dimagent` | 与 `Contents/MacOS/dimagent` 一致 |
| `CFBundleIdentifier` | `dev.dimagent.mygo` | Launch Services 和深链注册使用的 bundle id |
| `CFBundleName` / `CFBundleDisplayName` | `DimAgent` | Finder、Dock 和系统 UI 显示名 |
| `CFBundleShortVersionString` / `CFBundleVersion` | `--version` | 版本号 |
| `LSMinimumSystemVersion` | `13.0` | 最低 macOS 版本 |
| `NSHighResolutionCapable` | `true` | Retina 渲染 |
| `NSPrincipalClass` | `NSApplication` | 让 AppKit 以普通 GUI 应用启动 |
| `CFBundleURLTypes` | scheme `dimagent` | 声明深链 scheme |

字段与 mygo `v0.8.3` 的 `mygo build` 生成的 bundle 保持一致（`cmd/mygo/bundle.go` 的 `infoPlist`），但没有图标、本地化目录和资源。

### 2.4 URL scheme

应用注册 `dimagent://`，当前只有 `thread` 路由：

```text
dimagent://thread/<session-id>
```

- `CFBundleURLTypes` 让 Launch Services 知道这个 bundle 声明了 `dimagent`。未声明的 scheme 在 macOS 上无法注册；开发态直接运行未打包的二进制时，`mygo.App.RegisterURLScheme` 会失败并把原因写入日志。
- bundle 启动时调用 `mygo.App.RegisterURLScheme("dimagent")`，尝试把该 bundle 设为默认 handler。它只保证「这个 bundle 声明了 scheme」，不保证 macOS 一定把默认 handler 换成它；Launch Services 会按自己的优先级选择，已安装且声明同一 scheme 的 app 通常保持默认。
- 当前产品基线 `/Applications/DimAgent.app`（bundle id `com.dimcode.app`）也声明了 `dimagent` 和 `dimcode`。在实测环境里它是系统默认 handler，因此裸 `open 'dimagent://...'` 会交给它，而不是 `/tmp` 里的本地 bundle。本地 bundle 用 `open -a` 指定路径验证。
- 链接只携带 session id。窗口在 App Server 返回的 catalog 里查找这一行，再走和侧栏点击相同的选择路径；catalog 没有这个 id 时显示 `Open conversation ...` 错误并保持当前选择，不创建本地行、不读 daemon 存储。
- session id 可以通过窗口菜单的 Copy conversation link（复制完整链接）、Copy conversation ID，或 `scripts/seed-session` 的输出获得。

验证深链（显式指定本地 bundle）：

```bash
open -a /tmp/dimagent-app/DimAgent.app 'dimagent://thread/<session-id>'
```

已启动的窗口会被带到前台并切到该会话，stderr 会记录
`dimagent: open conversation "<id>" from a deep link`。

要验证系统级默认 handler（裸 `open 'dimagent://thread/<id>'`），需要先移除（并注销 LaunchServices 记录）或停用同样声明 `dimagent` 的竞争 app，或者把 MyGo 构建换成独立 scheme；后者要改 `internal/ui` 的 scheme 常量，不在打包脚本范围内。`open -b dev.dimagent.mygo` 也不能代替：`/tmp` 下的 bundle 在 macOS 26.6.2 上实测返回
`LSCopyApplicationURLsForBundleIdentifier() failed`，用路径的 `open -a` 才是可用的本地调用方式。

### 2.5 未签名 / 未公证的限制

- 默认只做 ad-hoc 签名（`codesign --force --deep --sign -`），没有 Developer ID，也没有公证。其他机器从网络下载后会被 Gatekeeper 拦下；本机从 `/tmp` 构建、没有 quarantine 属性时可以直接运行。
- 没有 hardened runtime、entitlements 或安装器；重建二进制后，系统对辅助功能/屏幕录制等 TCC 授权的记录可能与新签名不再匹配，需要重新授权。
- 只构建当前机器架构，不生成 universal binary；Intel 机器需要在 Intel 上重新构建。Windows/Linux 打包不在本脚本范围内。
- 没有应用图标、DMG、自动更新或崩溃上报；版本号写入 `Info.plist`、二进制的 `-version` 输出和 About 页，但没有发布通道或更新检查。
- `/tmp/dimagent-app` 是临时路径，Launch Services 会把 bundle 标记为 in-temp-dir，并按较低优先级处理。实测它不能被 `open -b dev.dimagent.mygo` 按 bundle id 解析，也不能接管已被安装 app 声明的 `dimagent` scheme；删除或移动 bundle 后旧 handler 同样会失效。
- 产物不是发布件，也不能作为产品验收证据；真实窗口验收仍需按 [acceptance.md](acceptance.md) 采集证据。

### 2.6 本地验证步骤

```bash
# 0. 未打包的构建打印 dev，不启动窗口或 daemon
go run ./cmd/dimagent -version

# 1. 构建 bundle 并检查 plist/签名
scripts/package-macos.sh --out /tmp/dimagent-app --version 0.1.0
/tmp/dimagent-app/DimAgent.app/Contents/MacOS/dimagent --version
plutil -lint /tmp/dimagent-app/DimAgent.app/Contents/Info.plist
codesign -dv --verbose=4 /tmp/dimagent-app/DimAgent.app

# 2. 用 throwaway product home 预置一个真实 session（需要 dim-app-server）
WORK="$(mktemp -d /tmp/dimagent-package.XXXXXX)"
go run ./scripts/seed-session \
  -home "$WORK/home" \
  -server /tmp/dim-2-target/debug/dim-app-server \
  -prompt "package check"
# 输出即为 <session-id>

# 3. 带 throwaway 配置启动 bundle（不读写真实用户配置）
# supervisor 只按 PATH 查找 dim-app-server，因此把它的目录显式带进去
open -a /tmp/dimagent-app/DimAgent.app \
  --env PATH="/tmp/dim-2-target/debug:$PATH" \
  --env DIMAGENT_PRODUCT_HOME="$WORK/home" \
  --env DIMAGENT_CONFIG_HOME="$WORK/config" \
  --env DIMAGENT_MOCK_MODEL=1

# 4. 验证深链选择会话（显式指定本地 bundle）
open -a /tmp/dimagent-app/DimAgent.app 'dimagent://thread/<session-id>'

# 5. 清理：先退出应用，再删除临时目录
```

`open --env` 和 `open -a <path> <url>` 在 macOS 26.6.2 上实测可用；旧系统上可以改为直接运行 bundle 内可执行文件，环境变量会正常继承：

```bash
PATH="/tmp/dim-2-target/debug:$PATH" \
DIMAGENT_PRODUCT_HOME="$WORK/home" \
DIMAGENT_CONFIG_HOME="$WORK/config" \
DIMAGENT_MOCK_MODEL=1 \
  /tmp/dimagent-app/DimAgent.app/Contents/MacOS/dimagent
```

### 2.7 已实测与未覆盖

已实测（macOS 26.6.2 arm64，2026-11-10）：

- `scripts/package-macos.sh --out /tmp/dimagent-app --version 0.1.0` 生成 `DimAgent.app`；`plutil -lint` 通过；`codesign -dv` 显示 `Signature=adhoc`、`Identifier=dev.dimagent.mygo`，`codesign --verify --deep --strict` 通过。
- `go run ./cmd/dimagent -version`（以及 `--version`）输出 `DimAgent dev (protocol v3)` 并退出 0，不启动窗口或 daemon；用 `--no-sign` 打包后，`Contents/MacOS/dimagent -version` 输出 `DimAgent 0.1.0 (protocol v3)`，与 `--version 0.1.0` 写入的 plist 一致。
- 用 throwaway home 预置两个真实 session 后，bundle 通过 `open -a ... --env ...` 启动，窗口显示两个会话和 `mock/deterministic` 模型，App Server 连接正常。
- `open -a /tmp/dimagent-app/DimAgent.app 'dimagent://thread/<session-id>'` 把窗口切到 seed 的会话；stderr 记录 `open conversation "<id>" from a deep link`。
- `lsregister -dump` 中该 bundle 的 `claimed schemes` 包含 `dimagent:`，`CFBundleURLTypes` 声明有效。
- 裸 `open 'dimagent://thread/<id>'` 不会到达该 bundle：系统默认 handler 是 `/Applications/DimAgent.app`（`com.dimcode.app`），它也声明 `dimagent`。为避免驱动已安装的 DimAgent 和真实用户数据，没有实际执行裸 `open`；这是通过默认 handler 查询确认的。`open -b dev.dimagent.mygo <url>` 失败并返回 `LSCopyApplicationURLsForBundleIdentifier() failed while trying to determine the application with bundle identifier dev.dimagent.mygo.`

未覆盖：Developer ID 签名/公证、把 `/tmp` bundle 变成系统默认 handler、Intel 和 universal binary、图标与 DMG、安装/卸载、自动更新、其他 macOS 大版本上的 Launch Services 行为。
