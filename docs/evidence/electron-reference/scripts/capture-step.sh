#!/usr/bin/env bash
# capture-step.sh -- launch the built dim-electron app, drive the renderer over
# CDP with ../../scripts/interact.mjs actions, screenshot the real window after
# each step, then shut down. Driver used for every window-*.png below except
# window-initial.png (only the output path and the work dir are parameterised
# compared with the run on 2026-11-09).
#
# Usage: capture-step.sh <action> <shot-name> [<action> <shot-name> ...]
# NOTE: needs ../../scripts/winlist.swift compiled to $WORK/winlist first.
set -uo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="${ELECTRON_CAPTURE_WORK:-/tmp/dim-electron-capture}"
OUT="$REPO_ROOT"
LOG="$WORK/app-step.log"

rm -rf "$WORK/home" "$WORK/userdata"
mkdir -p "$WORK/home" "$WORK/userdata" "$OUT"

ELECTRON=/Users/eric8810/Code/dim-electron/node_modules/.pnpm/electron@43.6.0/node_modules/electron/dist/Electron.app/Contents/MacOS/Electron
MAIN=/Users/eric8810/Code/dim-electron/packages/desktop/packages/electron-main/dist/app/main.cjs

export DIMCODE_HOME="$WORK/home"
export DIMCODE_USER_DATA_PATH="$WORK/userdata"
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

cleanup() {
  if kill -0 "$APP_PID" 2>/dev/null; then
    kill -TERM "$APP_PID" 2>/dev/null || true
    for _ in $(seq 1 50); do kill -0 "$APP_PID" 2>/dev/null || break; sleep 0.1; done
    kill -KILL "$APP_PID" 2>/dev/null || true
  fi
  for pid in $(ps -axo pid=,command= | grep -F "$WORK/home" | grep -F 'dim-app-server' | awk '{print $1}'); do
    kill -TERM "$pid" 2>/dev/null || true
  done
  sleep 0.5
  for pid in $(ps -axo pid=,command= | grep -F "$WORK/home" | grep -F 'dim-app-server' | awk '{print $1}'); do
    kill -KILL "$pid" 2>/dev/null || true
  done
}
trap cleanup EXIT INT TERM

deadline=$((SECONDS + 60)); WINDOW_ID=""
while [ "$SECONDS" -lt "$deadline" ]; do
  kill -0 "$APP_PID" 2>/dev/null || { echo "app exited early"; tail -20 "$LOG"; exit 1; }
  ALL="$("$WORK/winlist" "$APP_PID")"
  LINE="$(printf '%s\n' "$ALL" | awk -F'\t' '
    $2 != 0 { next }
    { split($3, b, ","); w = b[3] + 0; h = b[4] + 0; if (w < 400 || h < 300) next
      area = w * h
      if ($5 != "") { if (area > tarea) { tarea = area; trow = $0 } }
      else if (area > uarea) { uarea = area; urow = $0 } }
    END { if (trow != "") print trow; else if (urow != "") print urow }')"
  [ -n "$LINE" ] && { WINDOW_ID="$(printf '%s' "$LINE" | cut -f1)"; break; }
  sleep 0.5
done
[ -z "$WINDOW_ID" ] && { echo "no window"; tail -40 "$LOG"; exit 1; }
echo "window: $LINE"
sleep 4
cd "$WORK"

while [ "$#" -ge 2 ]; do
  action="$1"; name="$2"; shift 2
  echo "--- action: $action"
  node "$REPO_ROOT/scripts/interact.mjs" "$action" || true
  sleep 2
  png="$OUT/$name.png"
  screencapture -x -o -l "$WINDOW_ID" "$png"
  px="$(sips -g pixelWidth -g pixelHeight "$png" 2>/dev/null | awk '/pixel/ {printf "%s ", $2}')"
  echo "shot $name: $png ($px px, $(wc -c <"$png" | tr -d ' ') bytes)"
  {
    printf 'state\t%s\n' "$name"
    printf 'action\t%s\n' "$action"
    printf 'captured_at\t%s\n' "$(date '+%Y-%m-%d %H:%M:%S %Z')"
    printf 'app_pid\t%s\n' "$APP_PID"
    printf 'window_id\t%s\n' "$WINDOW_ID"
    printf 'window_row\t%s\n' "${LINE//$'\t'/ | }"
    printf 'png\t%s\n' "$png"
    printf 'png_pixels\t%s\n' "${px% }"
    printf 'png_bytes\t%s\n' "$(wc -c <"$png" | tr -d ' ')"
    printf 'dimcode_home\t%s\n' "$WORK/home"
  } >"$OUT/$name.txt"
done

echo "shutting down"
cleanup
trap - EXIT INT TERM
kill -0 "$APP_PID" 2>/dev/null && echo "app still alive" || echo "app stopped"
LEFT="$(ps -axo pid=,command= | grep -F "$WORK/home" | grep -F 'dim-app-server')"
[ -n "$LEFT" ] && echo "leftover daemon: $LEFT" || echo "no leftover daemon from this run"
