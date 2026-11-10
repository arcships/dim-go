#!/usr/bin/env bash
# capture-loading-state.sh -- capture the restore-time loading skeleton as a
# real Electron window, using the evidence-only latency proxy.
#
# Why a proxy: SIGSTOPping the daemon while a session row is clicked does not
# switch the active session (the renderer waits for session.open first), so no
# loading UI is visible; that natural case is recorded by capture-states.sh as
# window-loading-session-switch. The app's actual loading surface is the
# timeline skeleton shown while the restored active session opens. To hold that
# state long enough for an OS window screenshot, app-server-delay-proxy.mjs
# forwards the real daemon's NDJSON and delays only session.history /
# session.attach responses (DIM_PROXY_DELAY_MS). App, preload, renderer and
# daemon are unmodified.
#
# Requires:
#   PRODUCT_HOME  an already-seeded product home
#   DIMCODE_USER_DATA_PATH  a throwaway userData whose sidebar-preferences.json
#                           has lastActiveSessionId and prefers the dark theme
#                           (capture-states.sh leaves exactly such a userData)
#
# Usage:
#   PRODUCT_HOME=$WORK/home DIMCODE_USER_DATA_PATH=$WORK/userdata \
#     ./capture-loading-state.sh
set -uo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
OUT="$(cd "$DIR/.." && pwd)"
WORK="${ELECTRON_LOADING_WORK:-/tmp/dim-state-capture}"
PRODUCT_HOME="${PRODUCT_HOME:?PRODUCT_HOME must point at an already-seeded product home}"
SOURCE_USER_DATA="${DIMCODE_USER_DATA_PATH:?DIMCODE_USER_DATA_PATH must point at a userData with lastActiveSessionId}"
USER_DATA="$WORK/userdata-loading"
CDP_PORT="${CDP_PORT:-9336}"
DELAY_MS="${DIM_PROXY_DELAY_MS:-2500}"
LOG="$WORK/app-loading.log"

ELECTRON=/Users/eric8810/Code/dim-electron/node_modules/.pnpm/electron@43.6.0/node_modules/electron/dist/Electron.app/Contents/MacOS/Electron
MAIN=/Users/eric8810/Code/dim-electron/packages/desktop/packages/electron-main/dist/app/main.cjs
REFERENCE_REPO=/Users/eric8810/Code/dim-electron
REAL_SERVER="${REAL_APP_SERVER_BIN:-/tmp/dim-2-target/debug/dim-app-server}"
PROXY="$DIR/app-server-delay-proxy.mjs"

mkdir -p "$WORK" "$OUT"
rm -rf "$USER_DATA"
cp -R "$SOURCE_USER_DATA" "$USER_DATA"
if [ ! -x "$WORK/winlist" ]; then
  swiftc -O "$DIR/winlist.swift" -o "$WORK/winlist" || exit 1
fi

export DIMCODE_HOME="$PRODUCT_HOME"
export DIMCODE_USER_DATA_PATH="$USER_DATA"
export DIMCODE_APP_SERVER_BIN="$PROXY"
export DIM_PROXY_REAL_SERVER="$REAL_SERVER"
export DIM_PROXY_DELAY_MS="$DELAY_MS"
export DIM_PROXY_DELAY_METHODS="${DIM_PROXY_DELAY_METHODS:-session.history}"
export DIM_PROXY_DELAY_ONCE="${DIM_PROXY_DELAY_ONCE:-1}"
export DIMCODE_RENDERER_URL="file:///Users/eric8810/Code/dim-electron/packages/desktop/packages/renderer/dist/index.html"
export DIMCODE_PRELOAD_PATH="/Users/eric8810/Code/dim-electron/packages/desktop/packages/electron-main/dist/preload.cjs"
export DIMCODE_CHROME_HELPER_PATH="/Users/eric8810/Code/dim-electron/packages/desktop/packages/electron-main/dist/bin/dimcode-chrome-native-host.cjs"
export DIMCODE_RESOURCES_ROOT="/Users/eric8810/Code/dim-electron/packages/desktop"
export DIMCODE_DEV_MODE=0
export DIMCODE_DEVTOOLS=0
export DIMCODE_APP_VERSION=0.9.57
export CDP_PORT WINLIST="$WORK/winlist"

cd /Users/eric8810/Code/dim-electron/packages/desktop
ELECTRON_ENABLE_LOGGING=1 "$ELECTRON" --remote-debugging-port="$CDP_PORT" "$MAIN" >"$LOG" 2>&1 &
APP_PID=$!
echo "app pid: $APP_PID"

cleanup() {
  if kill -0 "$APP_PID" 2>/dev/null; then
    kill -TERM "$APP_PID" 2>/dev/null || true
    for _ in $(seq 1 50); do kill -0 "$APP_PID" 2>/dev/null || break; sleep 0.1; done
    kill -KILL "$APP_PID" 2>/dev/null || true
  fi
  for pid in $(ps -axo pid=,command= | grep -F "$PRODUCT_HOME" | grep -E 'dim-app-server|app-server-delay-proxy' | grep -v 'grep' | awk '{print $1}'); do
    kill -CONT "$pid" 2>/dev/null || true
    kill -TERM "$pid" 2>/dev/null || true
  done
  sleep 0.5
  for pid in $(ps -axo pid=,command= | grep -F "$PRODUCT_HOME" | grep -E 'dim-app-server|app-server-delay-proxy' | grep -v 'grep' | awk '{print $1}'); do
    kill -KILL "$pid" 2>/dev/null || true
  done
}
trap cleanup EXIT INT TERM

# Wait for the renderer CDP endpoint; the skeleton itself is polled by Node.
deadline=$((SECONDS + 45))
until curl -s --max-time 1 "http://127.0.0.1:$CDP_PORT/json/list" >/dev/null 2>&1; do
  if ! kill -0 "$APP_PID" 2>/dev/null; then
    echo "app exited before CDP was ready; log tail:" >&2
    tail -30 "$LOG" >&2
    exit 1
  fi
  if [ "$SECONDS" -ge "$deadline" ]; then
    echo "CDP did not come up within 45s; log tail:" >&2
    tail -30 "$LOG" >&2
    exit 1
  fi
  sleep 0.25
done

node "$DIR/interact-states.mjs" wait-skeleton "$APP_PID" "$OUT/window-loading.png" "$WORK/loading-facts.json" >"$WORK/wait-skeleton.log" 2>&1
STATUS=$?
if [ "$STATUS" -ne 0 ]; then
  echo "loading capture failed ($STATUS); see $WORK/wait-skeleton.log and $LOG" >&2
  exit 1
fi

FACTS="$(tr '\n' ' ' <"$WORK/loading-facts.json" | sed 's/[[:space:]]\+/ /g')"
WIN_LINE="$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["windowLine"])' "$WORK/loading-facts.json")"
WIN_ID="$(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["windowId"])' "$WORK/loading-facts.json")"
PNG="$OUT/window-loading.png"
PX="$(sips -g pixelWidth -g pixelHeight "$PNG" 2>/dev/null | awk '/pixel/ {printf "%s ", $2}')"
PROXY_PID="$(ps -axo pid=,command= | grep -F "$PRODUCT_HOME" | grep -F 'app-server-delay-proxy' | grep -v 'grep' | head -1)"
DAEMON_LINE="$(ps -axo pid=,command= | grep -F "$PRODUCT_HOME" | grep -F 'dim-app-server' | grep -v 'grep' | head -1)"
{
  printf 'state\twindow-loading\n'
  printf 'action\tlaunch with app-server-delay-proxy.mjs (real dim-app-server on <unix>.real; NDJSON proxy delays %s responses by %s ms, once=%s), restore the persisted active session, wait for [data-testid=timeline-skeleton], then screencapture\n' "${DIM_PROXY_DELAY_METHODS:-session.history}" "$DELAY_MS" "${DIM_PROXY_DELAY_ONCE:-1}"
  printf 'captured_at\t%s\n' "$(date '+%Y-%m-%d %H:%M:%S %Z')"
  printf 'app_pid\t%s\n' "$APP_PID"
  printf 'window_id\t%s\n' "$WIN_ID"
  printf 'window_row\t%s\n' "${WIN_LINE//$'\t'/ | }"
  printf 'window_points\t%s\n' "$(printf '%s' "$WIN_LINE" | cut -f3 | awk -F, '{print $3"x"$4}')"
  printf 'png\t%s\n' "$PNG"
  printf 'png_pixels\t%s\n' "${PX% }"
  printf 'png_bytes\t%s\n' "$(wc -c <"$PNG" | tr -d ' ')"
  printf 'png_md5\t%s\n' "$(md5 -q "$PNG")"
  printf 'dimcode_home\t%s\n' "$PRODUCT_HOME"
  printf 'user_data\t%s\n' "$USER_DATA"
  printf 'app_server_bin\t%s\n' "$PROXY"
  printf 'real_app_server_bin\t%s\n' "$REAL_SERVER"
  printf 'delay_methods\t%s\n' "${DIM_PROXY_DELAY_METHODS:-session.history}"
  printf 'delay_ms\t%s\n' "$DELAY_MS"
  printf 'delay_once\t%s\n' "${DIM_PROXY_DELAY_ONCE:-1}"
  printf 'proxy_process\t%s\n' "${PROXY_PID:-<none>}"
  printf 'daemon\t%s\n' "${DAEMON_LINE:-<none>}"
  printf 'natural_behavior\twith the daemon SIGSTOPped, clicking another sidebar session row leaves the previous session on screen and shows no loading text, spinner, skeleton or role=status; the renderer switches only after session.open completes (window-loading-session-switch)\n'
  printf 'normal_restore_control\tthe same userData launched without the proxy restores the last active session and renders its timeline; the proxied run falls back to the new-thread screen after the skeleton, so this capture proves the skeleton surface, not a successful restore under injected latency\n'
  printf 'limits\tthe proxy only delays one session.history response; the app open path does not tolerate the proxied gap and falls back to the new-thread screen (recorded in dom.recovered). Do not read this image as normal restore latency: a normal launch with the same userData showed the conversation. The skeleton is the renderer loading surface (brand mark, data-testid=timeline-skeleton, no text) plus the footer 正在加载... account state\n'
  printf 'locale\tvisible UI strings are zh-CN; document.documentElement.lang is "en" (existing renderer behavior, unchanged)\n'
  printf 'git_head\t%s\n' "$(git -C "$OUT/../../.." rev-parse HEAD 2>/dev/null || echo unknown)"
  printf 'reference_head\t%s\n' "$(git -C "$REFERENCE_REPO" rev-parse HEAD 2>/dev/null || echo unknown)"
  printf 'seed_session\t%s\n' "${SEED_SESSION_ID:-<unset>}"
  printf 'seed_side_branch\t%s\n' "${SEED_SIDE_BRANCH_ID:-<unset>}"
  printf 'seed_prompt\t%s\n' "${SEED_PROMPT:-<unset>}"
  printf 'dom\t%s\n' "$FACTS"
} >"$OUT/window-loading.txt"

echo "shot window-loading: window $WIN_ID, ${PX% } px, $(wc -c <"$PNG" | tr -d ' ') bytes"
echo "skeleton gone: $(python3 -c 'import json,sys;print(json.load(open(sys.argv[1]))["skeletonGone"])' "$WORK/loading-facts.json")"
cleanup
trap - EXIT INT TERM
kill -0 "$APP_PID" 2>/dev/null && echo "app still alive" || echo "app stopped"
LEFT="$(ps -axo pid=,command= | grep -F "$PRODUCT_HOME" | grep -E 'dim-app-server|app-server-delay-proxy' | grep -v 'grep')"
[ -n "$LEFT" ] && echo "leftover daemon/proxy: $LEFT" || echo "no leftover daemon/proxy from this run"
