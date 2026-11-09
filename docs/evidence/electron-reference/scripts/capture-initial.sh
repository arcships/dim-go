#!/usr/bin/env bash
# capture-initial.sh -- launch the built dim-electron app against a throwaway
# DIMCODE_HOME, wait for its main window, screenshot it, and shut everything down.
# This is the driver used for window-initial.png (as run 2026-11-09, only the
# output path is derived from the script location instead of being absolute).
#
# Usage: capture-initial.sh <state-name>
# NOTE: needs ../../scripts/winlist.swift compiled to $WORK/winlist first.
set -uo pipefail

REPO_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
WORK="${ELECTRON_CAPTURE_WORK:-/tmp/dim-electron-capture}"
OUT="$REPO_ROOT"
NAME="${1:?state name required}"
KEEP="${2:-}"
LOG="$WORK/app-$NAME.log"

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
echo "app pid: $APP_PID"

cleanup() {
  if kill -0 "$APP_PID" 2>/dev/null; then
    kill -TERM "$APP_PID" 2>/dev/null || true
    for _ in $(seq 1 50); do kill -0 "$APP_PID" 2>/dev/null || break; sleep 0.1; done
    kill -KILL "$APP_PID" 2>/dev/null || true
  fi
  # only daemons whose command line names this run's throwaway home
  for pid in $(ps -axo pid=,command= | grep -F "$WORK/home" | grep -F 'dim-app-server' | awk '{print $1}'); do
    kill -TERM "$pid" 2>/dev/null || true
  done
  sleep 0.5
  for pid in $(ps -axo pid=,command= | grep -F "$WORK/home" | grep -F 'dim-app-server' | awk '{print $1}'); do
    kill -KILL "$pid" 2>/dev/null || true
  done
}
trap cleanup EXIT INT TERM

deadline=$((SECONDS + 60))
WINDOW_ID=""
while [ "$SECONDS" -lt "$deadline" ]; do
  if ! kill -0 "$APP_PID" 2>/dev/null; then
    echo "app exited before showing a window; log tail:" >&2
    tail -20 "$LOG" >&2
    exit 1
  fi
  ALL="$("$WORK/winlist" "$APP_PID")"
  # layer 0 only; prefer a titled window, then the largest area
  LINE="$(printf '%s\n' "$ALL" | awk -F'\t' '
    $2 != 0 { next }
    {
      split($3, b, ",")
      w = b[3] + 0; h = b[4] + 0
      if (w < 400 || h < 300) next
      area = w * h
      if ($5 != "") { if (area > tarea) { tarea = area; trow = $0 } }
      else if (area > uarea) { uarea = area; urow = $0 }
    }
    END { if (trow != "") print trow; else if (urow != "") print urow }')"
  if [ -n "$LINE" ]; then
    WINDOW_ID="$(printf '%s' "$LINE" | cut -f1)"
    break
  fi
  sleep 0.5
done

echo "--- windows of pid $APP_PID ---"
printf '%s\n' "${ALL:-<none>}"

if [ -z "$WINDOW_ID" ]; then
  echo "no window appeared within 60s" >&2
  tail -40 "$LOG" >&2
  exit 1
fi

echo "waiting for first frame to settle"
sleep 5

STARTED_AT="$(date '+%Y-%m-%d %H:%M:%S %Z')"
PNG="$OUT/$NAME.png"
screencapture -x -o -l "$WINDOW_ID" "$PNG"
if [ ! -s "$PNG" ]; then
  echo "screencapture produced no file (Screen Recording permission?)" >&2
  exit 1
fi
PX="$(sips -g pixelWidth -g pixelHeight "$PNG" 2>/dev/null | awk '/pixel/ {printf "%s ", $2}')"

DAEMON="$(ps -axo pid=,command= | grep -F "$WORK/home" | grep -F 'dim-app-server' | head -1)"
{
  printf 'state\t%s\n' "$NAME"
  printf 'captured_at\t%s\n' "$STARTED_AT"
  printf 'app_pid\t%s\n' "$APP_PID"
  printf 'window_id\t%s\n' "$WINDOW_ID"
  printf 'window_row\t%s\n' "${LINE//$'\t'/ | }"
  printf 'png\t%s\n' "$PNG"
  printf 'png_pixels\t%s\n' "${PX% }"
  printf 'png_bytes\t%s\n' "$(wc -c <"$PNG" | tr -d ' ')"
  printf 'dimcode_home\t%s\n' "$WORK/home"
  printf 'daemon\t%s\n' "${DAEMON:-<none>}"
} >"$OUT/$NAME.txt"

echo "window: $LINE"
echo "screenshot: $PNG ($PX px, $(wc -c <"$PNG" | tr -d ' ') bytes)"

if [ "$KEEP" = "--keep" ]; then
  echo "keeping app pid $APP_PID alive"
  trap - EXIT INT TERM
  exit 0
fi

echo "shutting down"
cleanup
trap - EXIT INT TERM
kill -0 "$APP_PID" 2>/dev/null && echo "app still alive" || echo "app stopped"
LEFT="$(ps -axo pid=,command= | grep -F "$WORK/home" | grep -F 'dim-app-server')"
if [ -n "$LEFT" ]; then echo "leftover daemon: $LEFT"; else echo "no leftover daemon from this run"; fi
