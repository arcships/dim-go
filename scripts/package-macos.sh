#!/usr/bin/env bash
# package-macos.sh builds a minimal macOS .app bundle around the MyGo DimAgent
# window and declares the dimagent:// URL scheme in its Info.plist, so Launch
# Services can deliver conversation links to the app.
#
# Usage:
#   scripts/package-macos.sh [--out DIR] [--version VERSION] [--no-sign]
#
# The bundle is ad-hoc signed with `codesign --force --deep --sign -` when
# codesign is available. It is a local development artifact: unsigned by a
# Developer ID, not notarized and not a release build. See docs/release.md.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OUT_DIR="/tmp/dimagent-app"
VERSION="0.1.0"
SIGN=1

APP_NAME="DimAgent"
BUNDLE_ID="dev.dimagent.mygo"
EXECUTABLE="dimagent"
SCHEME="dimagent"
MIN_MACOS="13.0"

usage() {
  sed -n '2,12p' "${BASH_SOURCE[0]}" | sed 's/^# \{0,1\}//'
}

while [ "$#" -gt 0 ]; do
  case "$1" in
    --out)
      [ "$#" -ge 2 ] || { echo "package-macos: --out needs a value" >&2; exit 2; }
      OUT_DIR="$2"
      shift 2
      ;;
    --version)
      [ "$#" -ge 2 ] || { echo "package-macos: --version needs a value" >&2; exit 2; }
      VERSION="$2"
      shift 2
      ;;
    --no-sign) SIGN=0; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "package-macos: unknown argument: $1" >&2; exit 2 ;;
  esac
done

if [ -z "$OUT_DIR" ]; then
  echo "package-macos: --out must not be empty" >&2
  exit 2
fi
if [ -z "$VERSION" ]; then
  echo "package-macos: --version must not be empty" >&2
  exit 2
fi
# The version is written into an XML plist, so keep it to the characters a
# version may use; anything else would need escaping and is a typo anyway.
case "$VERSION" in
  *[!A-Za-z0-9._+-]*)
    echo "package-macos: --version must use only letters, digits, '.', '_', '+' and '-'" >&2
    exit 2
    ;;
esac

APP="$OUT_DIR/$APP_NAME.app"
CONTENTS="$APP/Contents"

# Start from an empty bundle: a leftover file must not end up in the package.
rm -rf "$APP"
mkdir -p "$CONTENTS/MacOS" "$CONTENTS/Resources"

echo "package-macos: building $EXECUTABLE (version $VERSION)" >&2
(
  cd "$REPO_ROOT"
  go build -o "$CONTENTS/MacOS/$EXECUTABLE" ./cmd/dimagent
)

# CFBundleExecutable and CFBundleIdentifier must match the executable and the
# app mygo sees at run time: mygo.App.RegisterURLScheme only succeeds when
# NSBundle reports the dimagent scheme as declared.
cat >"$CONTENTS/Info.plist" <<PLIST
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>CFBundleDevelopmentRegion</key>
	<string>en</string>
	<key>CFBundleDisplayName</key>
	<string>$APP_NAME</string>
	<key>CFBundleExecutable</key>
	<string>$EXECUTABLE</string>
	<key>CFBundleIdentifier</key>
	<string>$BUNDLE_ID</string>
	<key>CFBundleInfoDictionaryVersion</key>
	<string>6.0</string>
	<key>CFBundleName</key>
	<string>$APP_NAME</string>
	<key>CFBundlePackageType</key>
	<string>APPL</string>
	<key>CFBundleShortVersionString</key>
	<string>$VERSION</string>
	<key>CFBundleURLTypes</key>
	<array>
		<dict>
			<key>CFBundleURLName</key>
			<string>$BUNDLE_ID</string>
			<key>CFBundleURLSchemes</key>
			<array>
				<string>$SCHEME</string>
			</array>
		</dict>
	</array>
	<key>CFBundleVersion</key>
	<string>$VERSION</string>
	<key>LSMinimumSystemVersion</key>
	<string>$MIN_MACOS</string>
	<key>NSHighResolutionCapable</key>
	<true/>
	<key>NSPrincipalClass</key>
	<string>NSApplication</string>
	<key>NSSupportsAutomaticGraphicsSwitching</key>
	<true/>
</dict>
</plist>
PLIST

printf 'APPL????' >"$CONTENTS/PkgInfo"

if command -v plutil >/dev/null 2>&1; then
  plutil -lint "$CONTENTS/Info.plist" >/dev/null
fi

SIGNED="no (unsigned)"
if [ "$SIGN" = 0 ]; then
  SIGNED="no (--no-sign; the bundle is unsigned)"
elif command -v codesign >/dev/null 2>&1; then
  echo "package-macos: ad-hoc signing $APP" >&2
  codesign --force --deep --sign - "$APP"
  codesign --verify --strict "$APP"
  SIGNED="yes (ad-hoc: codesign --force --deep --sign -)"
else
  echo "package-macos: codesign not found; leaving the bundle unsigned" >&2
  SIGNED="no (codesign not found; the bundle is unsigned)"
fi

cat <<SUMMARY
Bundle:     $APP
Executable: $CONTENTS/MacOS/$EXECUTABLE
Version:    $VERSION
URL scheme: $SCHEME:// (declared in CFBundleURLTypes)
Signed:     $SIGNED

Run:        open -a "$APP"
Link:       open '$SCHEME://thread/<session-id>'
SUMMARY

if [ "$SIGN" = 0 ]; then
  echo "package-macos: --no-sign requested; the bundle is unsigned" >&2
fi
echo "package-macos: local artifact only; see docs/release.md for limits and verification" >&2
