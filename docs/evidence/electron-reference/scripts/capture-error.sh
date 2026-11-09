#!/usr/bin/env bash
# capture-error.sh -- drive dim-electron into the conversation state, then kill
# ONLY the dim-app-server this run spawned and capture the window 4 seconds
# later. The product home must already be seeded (see docs/l5-reference.md §5.9
# and ../../README.md); this driver never wipes it.
#
# Usage:
#   PRODUCT_HOME=/tmp/dim-error-capture/home \
#   DIMCODE_USER_DATA_PATH=/tmp/dim-error-capture/userdata \
#   ./capture-error.sh window-error
#
# The daemon is located by matching the *temporary* PRODUCT_HOME in its command
# line, so a daemon serving the real ~/.dimcode is never touched.
# NOTE: needs winlist.swift compiled to $WORK/winlist (compiled here if missing).
set -uo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
OUT="$(cd "$DIR/.." && pwd)"
OUT_DIR="${EVIDENCE_OUT_DIR:-$OUT}"
WORK="${ELECTRON_ERROR_WORK:-/tmp/dim-error-capture}"
PRODUCT_HOME="${PRODUCT_HOME:?PRODUCT_HOME must point at an already-seeded product home}"
USER_DATA="${DIMCODE_USER_DATA_PATH:-$WORK/userdata}"
LOG="$WORK/app-error.log"
CONSOLE_LOG="$WORK/renderer-console.log"
NAME="${1:-window-error}"
KILL_WAIT="${KILL_WAIT:-4}"

ELECTRON=/Users/eric8810/Code/dim-electron/node_modules/.pnpm/electron@43.6.0/node_modules/electron/dist/Electron.app/Contents/MacOS/Electron
MAIN=/Users/eric8810/Code/dim-electron/packages/desktop/packages/electron-main/dist/app/main.cjs
REFERENCE_REPO=/Users/eric8810/Code/dim-electron

mkdir -p "$WORK" "$OUT_DIR"
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

daemon_pids() {
  ps -axo pid=,command= | grep -F "$PRODUCT_HOME" | grep -F 'dim-app-server' | awk '{print $1}'
}

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
  for pid in $(daemon_pids); do
    kill -TERM "$pid" 2>/dev/null || true
  done
  sleep 0.5
  for pid in $(daemon_pids); do
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
sleep 2

# Enter the conversation: light theme, then click the seeded sidebar row.
node "$DIR/interact-conversation.mjs" light
node "$DIR/interact-conversation.mjs" click-session

DAEMON_PID="$(daemon_pids | head -1)"
if [ -z "$DAEMON_PID" ]; then
  echo "no dim-app-server found for $PRODUCT_HOME; cannot capture daemon loss" >&2
  exit 1
fi
DAEMON_CMD="$(ps -p "$DAEMON_PID" -o command=)"
echo "daemon pid: $DAEMON_PID"
echo "daemon cmd: $DAEMON_CMD"

# Listen to the renderer console while the daemon goes away.
node "$DIR/watch-console.mjs" $((KILL_WAIT + 8)) >"$CONSOLE_LOG" 2>&1 &
WATCH_PID=$!
sleep 1

if [ "${SKIP_KILL:-0}" = "1" ]; then
  # Control run: same window/composer state with the daemon still alive, used
  # only to tell an error-induced UI state apart from the seeded session's own
  # "no model configured" state.
  kill_started="<not killed: SKIP_KILL=1>"
  KILL_EXIT="<n/a>"
  echo "SKIP_KILL=1: leaving daemon $DAEMON_PID alive"
else
  kill_started="$(date '+%Y-%m-%d %H:%M:%S %Z')"
  kill -TERM "$DAEMON_PID" 2>/dev/null
  KILL_EXIT=$?
  echo "kill -TERM $DAEMON_PID exit=$KILL_EXIT at $kill_started"
fi

sleep "$KILL_WAIT"

ALIVE_AFTER_WAIT="$(daemon_pids | tr '\n' ' ')"
echo "daemon pids after ${KILL_WAIT}s: ${ALIVE_AFTER_WAIT:-<none>}"

# Give the supervisor's own respawn (if any) time to surface in the UI.
sleep 2
DAEMON_AFTER="$(daemon_pids | tr '\n' ' ')"
echo "daemon pids at capture time: ${DAEMON_AFTER:-<none>}"

if [ -n "${POST_LOSS_ACTION:-}" ]; then
  node "$DIR/interact-error.mjs" "$POST_LOSS_ACTION" || true
  sleep 1
  POST_LOSS_DAEMON="$(daemon_pids | tr '\n' ' ')"
  echo "daemon pids after post-loss action ${POST_LOSS_ACTION}: ${POST_LOSS_DAEMON:-<none>}"
  node "$DIR/probe-error.mjs" >"$WORK/probe-error-after-action.json" 2>&1 || true
fi

if [ "${SEND_PROBE:-0}" = "1" ]; then
  node "$DIR/interact-error.mjs" type || true
  node "$DIR/probe-error.mjs" >"$WORK/probe-error-typed.json" 2>&1 || true
  node "$DIR/interact-error.mjs" click-send || true
  sleep 1
  node "$DIR/probe-error.mjs" >"$WORK/probe-error-after-send.json" 2>&1 || true
fi

node "$DIR/probe-error.mjs" >"$WORK/probe-error.json" 2>"$WORK/probe-error.err" || true
wait "$WATCH_PID" 2>/dev/null || true

line="$(window_line)"
[ -z "$line" ] && { echo "window disappeared before $NAME" >&2; exit 1; }
wid="$(printf '%s' "$line" | cut -f1)"
started="$(date '+%Y-%m-%d %H:%M:%S %Z')"
png="$OUT_DIR/$NAME.png"
screencapture -x -o -l "$wid" "$png"
if [ ! -s "$png" ]; then
  echo "screencapture produced no file for $NAME (Screen Recording permission?)" >&2
  exit 1
fi
px="$(sips -g pixelWidth -g pixelHeight "$png" 2>/dev/null | awk '/pixel/ {printf "%s ", $2}')"
if [ "${SKIP_KILL:-0}" = "1" ]; then
  ACTION_DESC="control: enter conversation (light, click sidebar row) -> leave daemon alive -> type probe sentence -> click send -> +1s -> capture"
elif [ -n "${POST_LOSS_ACTION:-}" ]; then
  ACTION_DESC="enter conversation (light, click sidebar row) -> kill -TERM <own daemon> -> wait ${KILL_WAIT}s+2s -> ${POST_LOSS_ACTION} -> +1s -> capture"
elif [ "${SEND_PROBE:-0}" = "1" ]; then
  ACTION_DESC="enter conversation (light, click sidebar row) -> kill -TERM <own daemon> -> wait ${KILL_WAIT}s+2s -> type probe sentence -> click send (disabled) -> +1s -> capture"
else
  ACTION_DESC="enter conversation (light, click sidebar row) -> kill -TERM <own daemon> -> wait ${KILL_WAIT}s+2s -> capture"
fi

{
  printf 'state\t%s\n' "$NAME"
  printf 'action\t%s\n' "$ACTION_DESC"
  printf 'command\t%s\n' "PRODUCT_HOME=$PRODUCT_HOME $DIR/capture-error.sh $NAME"
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
  printf 'killed_daemon_pid\t%s\n' "$DAEMON_PID"
  printf 'killed_daemon_command\t%s\n' "$DAEMON_CMD"
  printf 'kill_signal\tTERM\n'
  printf 'kill_started_at\t%s\n' "$kill_started"
  printf 'kill_exit\t%s\n' "$KILL_EXIT"
  printf 'daemon_pids_after_wait\t%s\n' "${ALIVE_AFTER_WAIT:-<none>}"
  printf 'daemon_pids_at_capture\t%s\n' "${DAEMON_AFTER:-<none>}"
  printf 'app_log\t%s\n' "$LOG"
  printf 'console_log\t%s\n' "$CONSOLE_LOG"
  printf 'probe_json\t%s\n' "$WORK/probe-error.json"
  if [ "${SEND_PROBE:-0}" = "1" ]; then
    printf 'send_probe\t1\n'
    printf 'probe_typed_json\t%s\n' "$WORK/probe-error-typed.json"
    printf 'probe_after_send_json\t%s\n' "$WORK/probe-error-after-send.json"
  else
    printf 'send_probe\t0\n'
  fi
  if [ -n "${POST_LOSS_ACTION:-}" ]; then
    printf 'post_loss_action\t%s\n' "$POST_LOSS_ACTION"
    printf 'daemon_pids_after_action\t%s\n' "${POST_LOSS_DAEMON:-<none>}"
    printf 'probe_after_action_json\t%s\n' "$WORK/probe-error-after-action.json"
  fi
  printf 'git_head\t%s\n' "$(git -C "$OUT/../../.." rev-parse HEAD 2>/dev/null || echo unknown)"
  printf 'reference_head\t%s\n' "$(git -C "$REFERENCE_REPO" rev-parse HEAD 2>/dev/null || echo unknown)"
  printf 'seed_session\t%s\n' "${SEED_SESSION_ID:-<unset>}"
  printf 'seed_side_branch\t%s\n' "${SEED_SIDE_BRANCH_ID:-<unset>}"
  printf 'seed_prompt\t%s\n' "${SEED_PROMPT:-<unset>}"
} >"$OUT_DIR/$NAME.txt"
echo "shot $NAME: window $wid, $px px, $(wc -c <"$png" | tr -d ' ') bytes"

cleanup
trap - EXIT INT TERM
kill -0 "$APP_PID" 2>/dev/null && echo "app still alive" || echo "app stopped"
LEFT="$(daemon_pids)"
if [ -n "$LEFT" ]; then echo "leftover daemon: $LEFT"; else echo "no leftover daemon from this run"; fi
