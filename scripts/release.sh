#!/usr/bin/env bash
# 产出一个可双击运行的 dist/DimAgent.app 与发布包。
#
#   scripts/release.sh                 # 默认版本取自 git describe
#   scripts/release.sh -v 0.2.0        # 指定版本
#   DIM_SERVER_SRC=~/Code/dim-2 scripts/release.sh
#
# 产物（全部在 dist/）：
#   DimAgent.app/                        可双击运行；Contents/MacOS 下同时放 dim-app-server，
#                                        客户端按「可执行文件同目录」发现它，双击即可拉起。
#                                        服务端版本随包写进 Resources/dim-app-server.version 与
#                                        客户端（启动日志第一行、`DimAgent -version` 都读它）。
#   DimAgent-<版本>-macos-<架构>.tar.gz   打包整棵 .app
#   DimAgent-<版本>-macos-<架构>.tar.gz.sha256
#
# 服务端：默认在 DIM_SERVER_SRC（缺省 ~/Code/dim-2）里跑 cargo build --release，
# 构建产物**立即**拷进包：target/ 是可回收缓存，留到最后再拷会拿到空路径。
# 已经有构建产物时用 DIM_SERVER_BIN 指过去，跳过编译。
set -euo pipefail

ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST="$ROOT/dist"
APP="$DIST/DimAgent.app"
BUNDLE_ID="cn.dimagent.client"

# 服务端源码与二进制：源码路径可用 DIM_SERVER_SRC 覆盖（默认产品仓）。
: "${DIM_SERVER_SRC:=$HOME/Code/dim-2}"
SERVER_BIN="${DIM_SERVER_BIN:-}"
VERSION="${VERSION:-}"
SIGN=1

usage() {
	cat <<'EOF'
用法：scripts/release.sh [-v|--version <x.y.z>] [--no-sign] [-h|--help]

  -v, --version <版本>  发布版本；缺省取 git describe，形态一律规整成 x.y.z
      --no-sign         跳过 ad-hoc 代码签名
  -h, --help            显示本帮助

环境变量：
  DIM_SERVER_SRC      App Server 源码目录（默认 ~/Code/dim-2），在其中 cargo build --release
  DIM_SERVER_BIN      已有的 dim-app-server 可执行文件；给出时跳过编译
  DIM_SERVER_VERSION  包内服务端版本；缺省在源码构建时自动从 cargo 取，用现成二进制时需显式给出
  VERSION             与 --version 同义
EOF
}

die() {
	printf 'release: %s\n' "$*" >&2
	exit 1
}

while [[ $# -gt 0 ]]; do
	case "$1" in
	-v | --version)
		[[ $# -ge 2 ]] || die "--version 需要一个值"
		VERSION="$2"
		shift 2
		;;
	--no-sign)
		SIGN=0
		shift
		;;
	-h | --help)
		usage
		exit 0
		;;
	*)
		usage >&2
		die "未知参数 $1"
		;;
	esac
done

# 版本一律规整成 x.y.z：Info.plist 与客户端诊断都只认这个形状。
# 只接受以数字开头的形态（0.1.0、v1.2.3、0.1.0-alpha.18）；认不出来时返回空。
parse_version() {
	local raw="$1"
	if [[ ! "$raw" =~ ^v?([0-9]+)(\.([0-9]+))?(\.([0-9]+))? ]]; then
		return 0
	fi
	local major="${BASH_REMATCH[1]}"
	local minor="${BASH_REMATCH[3]:-0}"
	local patch="${BASH_REMATCH[5]:-0}"
	printf '%s.%s.%s' "$major" "$minor" "$patch"
}

# 回退版本：没有 x.y.z 形态的来源时用代码里的默认版本（入口的 defaultVersion），
# 不写 0.0.0 —— 一个打出来的包至少要知道自己是什么版本。
default_version() {
	local entry="$ROOT/cmd/dimagent-ui/main.go"
	local found=""
	if [[ -f "$entry" ]]; then
		found="$(sed -n 's/^const defaultVersion = "\([0-9][0-9.]*\)".*$/\1/p' "$entry" | head -n 1)"
	fi
	printf '%s' "${found:-0.1.0}"
}

# cargo_package_version 取 dim-app-server 的版本：与 hello 回执里的 serverVersion 同源
# （构建时的 CARGO_PKG_VERSION）。先问 cargo metadata，取不到再读工作区 Cargo.toml。
cargo_package_version() {
	local src="$1" version=""
	if command -v python3 >/dev/null; then
		version="$(
			cd "$src" &&
				cargo metadata --no-deps --format-version 1 2>/dev/null |
				python3 -c 'import json, sys
packages = json.load(sys.stdin).get("packages", [])
print(next((p["version"] for p in packages if p.get("name") == "dim-app-server"), ""))' 2>/dev/null || true
		)"
	fi
	if [[ -z "$version" && -f "$src/Cargo.toml" ]]; then
		version="$(sed -n '/^\[workspace\.package\]/,/^\[/s/^version *= *"\([^"]*\)".*/\1/p' \
			"$src/Cargo.toml" | head -n 1)"
	fi
	printf '%s' "$version"
}

if [[ -z "$VERSION" ]]; then
	described="$(git -C "$ROOT" describe --tags 2>/dev/null || true)"
	VERSION="$(parse_version "$described")"
	if [[ -z "$VERSION" ]]; then
		VERSION="$(default_version)"
		printf 'release: git 标签 %s 不是 x.y.z 形态，版本回退为代码默认版本 %s（发布时请用 -v 指定）\n' \
			"${described:-（无标签）}" "$VERSION" >&2
	fi
else
	raw_version="$VERSION"
	VERSION="$(parse_version "$raw_version")"
	[[ -n "$VERSION" ]] || die "版本 $raw_version 不是 x.y.z 形态"
fi
COMMIT="$(git -C "$ROOT" rev-parse --short HEAD 2>/dev/null || echo unknown)"

case "$(uname -m)" in
arm64) ARCH=arm64 ;;
x86_64) ARCH=x86_64 ;;
*) ARCH="$(uname -m)" ;;
esac

command -v go >/dev/null || die "缺少 go（在 PATH 里找不到）"

# 0. 先清空 bundle 骨架：服务端构建产物要立刻拷进包，不能等客户端构建完再拷——
#    target/ 是可回收缓存，客户端构建期间它可能已经空了。
rm -rf "$APP"
mkdir -p "$APP/Contents/MacOS" "$APP/Contents/Resources"

SERVER_VERSION="${DIM_SERVER_VERSION:-}"
# 1. 服务端：默认在源码目录里做 release 构建；给了现成二进制就直接用。
#    构建产物这一步就拷进包，之后不再依赖源码目录里的 target/。
if [[ -z "$SERVER_BIN" ]]; then
	[[ -f "$DIM_SERVER_SRC/Cargo.toml" ]] ||
		die "App Server 源码目录不可用：$DIM_SERVER_SRC（用 DIM_SERVER_SRC 指定，或用 DIM_SERVER_BIN 给现成二进制）"
	command -v cargo >/dev/null || die "缺少 cargo（App Server 需要 cargo build --release）"
	printf 'release: 构建 App Server（%s）\n' "$DIM_SERVER_SRC"
	(cd "$DIM_SERVER_SRC" && cargo build --release -p dim-app-server)
	SERVER_BIN="$DIM_SERVER_SRC/target/release/dim-app-server"
	if [[ -z "$SERVER_VERSION" ]]; then
		SERVER_VERSION="$(cargo_package_version "$DIM_SERVER_SRC")"
	fi
fi
[[ -f "$SERVER_BIN" ]] || die "App Server 可执行文件不存在：$SERVER_BIN"
[[ -x "$SERVER_BIN" ]] || die "App Server 可执行文件不可执行：$SERVER_BIN"
cp "$SERVER_BIN" "$APP/Contents/MacOS/dim-app-server"
chmod +x "$APP/Contents/MacOS/dim-app-server"
if [[ -z "$SERVER_VERSION" ]]; then
	printf 'release: 包内服务端版本未知（用 DIM_SERVER_VERSION 指定）\n' >&2
fi
printf 'release: 服务端已拷进包（版本 %s）\n' "${SERVER_VERSION:-未知}"

# 2. 客户端：产物直接写进 bundle 里，保证 Contents/MacOS 与服务端同目录。
#    包内服务端版本注入客户端：启动日志第一行与 `DimAgent -version` 都从它来。
printf 'release: 构建客户端（版本 %s）\n' "$VERSION"
LDFLAGS="-s -w -X main.version=$VERSION"
if [[ -n "$SERVER_VERSION" ]]; then
	LDFLAGS="$LDFLAGS -X main.bundledServerVersion=$SERVER_VERSION"
fi
(
	cd "$ROOT"
	go build -trimpath -ldflags "$LDFLAGS" \
		-o "$APP/Contents/MacOS/DimAgent" ./cmd/dimagent-ui
)
chmod +x "$APP/Contents/MacOS/DimAgent"
printf '%s\n' "${SERVER_VERSION:-unknown}" >"$APP/Contents/Resources/dim-app-server.version"

# 3. Info.plist：应用名、版本、包标识与最低系统版本；不引用图标与 nib（界面全在 Go 里）。
cat >"$APP/Contents/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleDevelopmentRegion</key>
	<string>zh_CN</string>
	<key>CFBundleExecutable</key>
	<string>DimAgent</string>
	<key>CFBundleIdentifier</key>
	<string>$BUNDLE_ID</string>
	<key>CFBundleInfoDictionaryVersion</key>
	<string>6.0</string>
	<key>CFBundleName</key>
	<string>DimAgent</string>
	<key>CFBundleDisplayName</key>
	<string>DimAgent</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleShortVersionString</key>
	<string>$VERSION</string>
	<key>CFBundleVersion</key>
	<string>$VERSION</string>
	<key>LSApplicationCategoryType</key>
	<string>public.app-category.developer-tools</string>
	<key>LSMinimumSystemVersion</key>
	<string>13.0</string>
	<key>NSHighResolutionCapable</key>
	<true/>
	<key>NSPrincipalClass</key>
	<string>NSApplication</string>
	<!-- 深链：系统把 dimagent://session/<id>、dimagent://settings/<section>
	     交给本应用。客户端启动时还会把它登记为默认处理者（mygo 的
	     RegisterURLScheme；macOS 要求包内先声明，否则登记会被拒绝）。 -->
	<key>CFBundleURLTypes</key>
	<array>
		<dict>
			<key>CFBundleURLName</key>
			<string>$BUNDLE_ID</string>
			<key>CFBundleURLSchemes</key>
			<array>
				<string>dimagent</string>
			</array>
		</dict>
	</array>
</dict>
</plist>
PLIST
printf 'APPL????' >"$APP/Contents/PkgInfo"
find "$APP" -name '.DS_Store' -delete 2>/dev/null || true

# 4. ad-hoc 签名：本机直接双击运行时，签名让系统把 App Server 子进程算作同一个应用。
if [[ "$SIGN" == 1 ]]; then
	if command -v codesign >/dev/null; then
		printf 'release: ad-hoc 签名\n'
		codesign --force --sign - "$APP/Contents/MacOS/dim-app-server" >/dev/null 2>&1 || true
		codesign --force --sign - --identifier "$BUNDLE_ID" "$APP" >/dev/null 2>&1 ||
			printf 'release: 签名失败（不影响本机运行；分发前需正式签名）\n' >&2
	else
		printf 'release: 找不到 codesign，跳过签名\n' >&2
	fi
fi

# 5. 结构自检：两个可执行文件、版本文件与 Info.plist 必须都在位。
for required in \
	"$APP/Contents/Info.plist" \
	"$APP/Contents/MacOS/DimAgent" \
	"$APP/Contents/MacOS/dim-app-server" \
	"$APP/Contents/Resources/dim-app-server.version"; do
	[[ -e "$required" ]] || die "产物缺少 $required"
done

# 5.0 深链自检：Info.plist 必须声明 dimagent://（缺了它系统不会把链接交给本应用）。
if command -v plutil >/dev/null; then
	schemes="$(plutil -extract CFBundleURLTypes json -o - "$APP/Contents/Info.plist" 2>/dev/null || true)"
	if [[ "$schemes" != *'"dimagent"'* ]]; then
		die "Info.plist 没有声明 dimagent://（CFBundleURLTypes）：$schemes"
	fi
	printf 'release: 深链已声明 dimagent://\n'
else
	grep -q '<string>dimagent</string>' "$APP/Contents/Info.plist" ||
		die "Info.plist 没有声明 dimagent://"
fi

# 5.1 版本自检：包内客户端要能自己报出包内服务端版本（启动日志第一行用的是同一份）。
if VERSION_LINE="$("$APP/Contents/MacOS/DimAgent" -version 2>/dev/null)"; then
	printf 'release: 包内版本：%s\n' "$VERSION_LINE"
else
	die "包内客户端打印版本失败：$APP/Contents/MacOS/DimAgent -version"
fi

# 6. 打包与校验和：.app 整棵进包，sha256 与包同目录。
PKG="DimAgent-$VERSION-macos-$ARCH.tar.gz"
# 旧包清理：dist/ 只留本次出包的产物，免得几个版本的包混在一起分不清哪个能用。
shopt -s nullglob
for stale in "$DIST"/DimAgent-*.tar.gz "$DIST"/DimAgent-*.tar.gz.sha256; do
	case "$stale" in
	"$DIST/$PKG" | "$DIST/$PKG.sha256") continue ;;
	esac
	rm -f "$stale"
	printf 'release: 清理旧包 %s\n' "$(basename "$stale")"
done
shopt -u nullglob
rm -f "$DIST/$PKG" "$DIST/$PKG.sha256"
(
	cd "$DIST"
	COPYFILE_DISABLE=1 tar -czf "$PKG" DimAgent.app
	shasum -a 256 "$PKG" >"$PKG.sha256"
)

# 7. dist/README：写清这份产物的版本、来源与重出命令（dist 不进仓库）。
cat >"$DIST/README.md" <<README
# dist/：本机发布产物

由 \`scripts/release.sh\` 生成，生成时间 $(date '+%Y-%m-%d %H:%M:%S')。

| 文件 | 说明 |
|---|---|
| \`DimAgent.app\` | 可双击运行；\`Contents/MacOS\` 下同时放 \`dim-app-server\`，客户端按「可执行文件同目录」发现它 |
| \`$PKG\` | 打包整棵 \`.app\` |
| \`$PKG.sha256\` | 包的 sha256 校验和 |

- 版本：\`$VERSION\`；架构：\`$ARCH\`；提交：\`$COMMIT\`。
- 包内 App Server 版本：\`${SERVER_VERSION:-unknown}\`（也写在
  \`DimAgent.app/Contents/Resources/dim-app-server.version\`；\`DimAgent -version\` 可打印）。
- 重出同一版本：\`scripts/release.sh -v $VERSION\`。不带 \`-v\` 时取
  \`git describe\` 的标签；标签不是 x.y.z 形态时用入口代码里的默认版本。
- 每次出包都会删掉其它版本的 \`DimAgent-*.tar.gz\` 与校验和，只保留最近一次。
- 本目录不进仓库（见 \`.gitignore\`），不要手工编辑这里的包。
README

# 8. 汇总：路径、体积与校验和，交给验收照着跑。
human() { du -sh "$1" | awk '{print $1}'; }
printf '\nrelease: 完成（版本 %s，架构 %s，提交 %s）\n' "$VERSION" "$ARCH" "$COMMIT"
printf '  应用      %s（%s）\n' "$APP" "$(human "$APP")"
printf '  客户端    %s（%s）\n' "$APP/Contents/MacOS/DimAgent" "$(ls -lh "$APP/Contents/MacOS/DimAgent" | awk '{print $5}')"
printf '  服务端    %s（%s，版本 %s）\n' "$APP/Contents/MacOS/dim-app-server" \
	"$(ls -lh "$APP/Contents/MacOS/dim-app-server" | awk '{print $5}')" "${SERVER_VERSION:-未知}"
printf '  发布包    %s（%s）\n' "$DIST/$PKG" "$(human "$DIST/$PKG")"
printf '  校验和    %s\n' "$(cat "$DIST/$PKG.sha256")"
printf '\n双击 %s 即可运行；命令行运行：%s\n' "$APP" "$APP/Contents/MacOS/DimAgent"
