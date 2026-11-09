#!/usr/bin/env bash
# capture-conversation.sh -- launch the built dim-electron app against an
# already-seeded product home and screenshot the conversation state.
#
# Unlike capture-initial.sh / capture-step.sh this driver does NOT wipe the
# home: the session has to be seeded first (see docs/l5-reference.md §4.1) so
# the window opens with a real timeline row instead of the empty state.
#
# Usage:
#   PRODUCT_HOME=/tmp/dim-conversation-capture/home \
#   DIMCODE_USER_DATA_PATH=/tmp/dim-conversation-capture/userdata \
#   ./capture-conversation.sh window-conversation window-conversation-dark
#
# Each argument is one shot name; the script always drives the same two states
# in order (click session -> light shot, switch 深色 -> dark shot) and stops
# after the shots are taken.
# NOTE: needs ../../scripts/winlist.swift compiled to $WORK/winlist (compiled
# here if missing).
set -uo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
OUT="$(cd "$DIR/.." && pwd)"
WORK="${ELECTRON_CONVERSATION_WORK:-/tmp/dim-conversation-capture}"
PRODUCT_HOME="${PRODUCT_HOME:?PRODUCT_HOME must point at an already-seeded product home}"
USER_DATA="${DIMCODE_USER_DATA_PATH:-$WORK/userdata}"
LOG="$WORK/app-conversation.log"
LIGHT_NAME="${1:-window-conversation}"
DARK_NAME="${2:-window-conversation-dark}"

ELECTRON=/Users/eric8810/Code/dim-electron/node_modules/.pnpm/electron@43.6.0/node_modules/electron/dist/Electron.app/Contents/MacOS/Electron
MAIN=/Users/eric8810/Code/dim-electron/packages/desktop/packages/electron-main/dist/app/main.cjs
REFERENCE_REPO=/Users/eric8810/Code/dim-electron

mkdir -p "$WORK" "$OUT"
# Desktop UI state (right panel, collapsed sidebar) persists in userData and can
# hide the sidebar entry points the driver clicks, so start from a clean
# userData unless KEEP_USER_DATA=1. The product home is never touched here.
if [ "${KEEP_USER_DATA:-0}" != "1" ]; then
  rm -rf "$USER_DATA"
fi
mkdir -p "$USER_DATA"
if [ ! -x "$WORK/winlist" ]; then
  swiftc -O "$DIR/winlist.swift" -o "$WORK/winlist" || exit 1
fi

export DIMCODE_HOME="$PRODUCT_HOME"
export DIMCODE_USER_DATA_PATH="$USER_DATA"
export DIMCODE_APP_SERVER_BIN=/tmp/dim-2-target/debug/dim-app-server
export DIMCODE_RENDERER_URL="file:///Users/eric8810/Code/dim-electron/packages/desktop/packages/renderer/dist/index.html"
export DIMCODE_PRELOAD_PATH="/Users/eric8810/Code/dim-electron/packages/desktop/packages/electron-main/dist/preload.cjs"
export DIMCODE_CHROME_HELPER_PATH="/Users/eric8810/Code/dim-electron/packages/desktop/packages/electron-main/dist/bin/dimcode-chrome-native-host.cjs"
export DIMCODE_RESOURCES_ROOT="/Users/eric8810/Code/dim-electron/packages/desktop"
export DIMCODE_DEV_MODE=0
export DIMCODE_DEVTOOLS=0
export DIMCODE_APP_VERSION=0.9.57

cd /Users/eric8810/Code/dim-electron/packages/desktop
ELECTRON_ENABLE_LOGGING=1 "$ELECTRON" --remote-debugging-port=9333 "$MAIN" >"$LOG" 2>&1 &
APP_PID=$!
echo "app pid: $APP_PID"

cleanup() {
  if kill -0 "$APP_PID" 2>/dev/null; then
    kill -TERM "$APP_PID" 2>/dev/null || true
    for _ in $(seq 1 50); do kill -0 "$APP_PID" 2>/dev/null || break; sleep 0.1; done
    kill -KILL "$APP_PID" 2>/dev/null || true
  fi
  # only daemons whose command line names this run's product home
  for pid in $(ps -axo pid=,command= | grep -F "$PRODUCT_HOME" | grep -F 'dim-app-server' | awk '{print $1}'); do
    kill -TERM "$pid" 2>/dev/null || true
  done
  sleep 0.5
  for pid in $(ps -axo pid=,command= | grep -F "$PRODUCT_HOME" | grep -F 'dim-app-server' | awk '{print $1}'); do
    kill -KILL "$pid" 2>/dev/null || true
  done
}
trap cleanup EXIT INT TERM

window_line() {
  ALL="$("$WORK/winlist" "$APP_PID")"
  printf '%s\n' "$ALL" | awk -F'\t' '
    $2 != 0 { next }
    {
      split($3, b, ",")
      w = b[3] + 0; h = b[4] + 0
      if (w < 400 || h < 300) next
      area = w * h
      if ($5 != "") { if (area > tarea) { tarea = area; trow = $0 } }
      else if (area > uarea) { uarea = area; urow = $0 }
    }
    END { if (trow != "") print trow; else if (urow != "") print urow }'
}

deadline=$((SECONDS + 60)); LINE=""
while [ "$SECONDS" -lt "$deadline" ]; do
  if ! kill -0 "$APP_PID" 2>/dev/null; then
    echo "app exited before showing a window; log tail:" >&2
    tail -20 "$LOG" >&2
    exit 1
  fi
  LINE="$(window_line)"
  [ -n "$LINE" ] && break
  sleep 0.5
done
if [ -z "$LINE" ]; then
  echo "no window appeared within 60s" >&2
  tail -40 "$LOG" >&2
  exit 1
fi
echo "window: $LINE"
sleep 4

shot() {
  local name="$1" action="$2"
  local line wid started png px daemon
  line="$(window_line)"
  [ -z "$line" ] && { echo "window disappeared before $name" >&2; exit 1; }
  wid="$(printf '%s' "$line" | cut -f1)"
  sleep 1
  started="$(date '+%Y-%m-%d %H:%M:%S %Z')"
  png="$OUT/$name.png"
  screencapture -x -o -l "$wid" "$png"
  if [ ! -s "$png" ]; then
    echo "screencapture produced no file for $name (Screen Recording permission?)" >&2
    exit 1
  fi
  px="$(sips -g pixelWidth -g pixelHeight "$png" 2>/dev/null | awk '/pixel/ {printf "%s ", $2}')"
  daemon="$(ps -axo pid=,command= | grep -F "$PRODUCT_HOME" | grep -F 'dim-app-server' | head -1)"
  {
    printf 'state\t%s\n' "$name"
    printf 'action\t%s\n' "$action"
    printf 'command\t%s\n' "PRODUCT_HOME=$PRODUCT_HOME $DIR/capture-conversation.sh $LIGHT_NAME $DARK_NAME"
    printf 'captured_at\t%s\n' "$started"
    printf 'app_pid\t%s\n' "$APP_PID"
    printf 'window_id\t%s\n' "$wid"
    printf 'window_row\t%s\n' "${line//$'\t'/ | }"
    printf 'window_bounds_points\t%s\n' "$(printf '%s' "$line" | cut -f3)"
    printf 'png\t%s\n' "$png"
    printf 'png_pixels\t%s\n' "${px% }"
    printf 'png_bytes\t%s\n' "$(wc -c <"$png" | tr -d ' ')"
    printf 'png_md5\t%s\n' "$(md5 -q "$png")"
    printf 'dimcode_home\t%s\n' "$PRODUCT_HOME"
    printf 'user_data\t%s\n' "$USER_DATA"
    printf 'daemon\t%s\n' "${daemon:-<none>}"
    printf 'git_head\t%s\n' "$(git -C "$OUT/../../.." rev-parse HEAD 2>/dev/null || echo unknown)"
    printf 'reference_head\t%s\n' "$(git -C "$REFERENCE_REPO" rev-parse HEAD 2>/dev/null || echo unknown)"
    printf 'seed_session\t%s\n' "${SEED_SESSION_ID:-<unset>}"
    printf 'seed_side_branch\t%s\n' "${SEED_SIDE_BRANCH_ID:-<unset>}"
    printf 'seed_prompt\t%s\n' "${SEED_PROMPT:-<unset>}"
  } >"$OUT/$name.txt"
  echo "shot $name: window $wid, $px px, $(wc -c <"$png" | tr -d ' ') bytes"
}

node "$DIR/interact-conversation.mjs" light
node "$DIR/interact-conversation.mjs" click-session
shot "$LIGHT_NAME" "settings -> 外观 -> 浅色 -> 返回; click [data-testid=sidebar-session-item] (already-open session; click through the sidebar entry)"

node "$DIR/interact-conversation.mjs" dark
shot "$DARK_NAME" "设置 -> 外观 -> 深色 -> 返回"

node "$DIR/interact-conversation.mjs" probe-branch || true

cleanup
trap - EXIT INT TERM
kill -0 "$APP_PID" 2>/dev/null && echo "app still alive" || echo "app stopped"
LEFT="$(ps -axo pid=,command= | grep -F "$PRODUCT_HOME" | grep -F 'dim-app-server')"
if [ -n "$LEFT" ]; then echo "leftover daemon: $LEFT"; else echo "no leftover daemon from this run"; fi
