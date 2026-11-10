#!/usr/bin/env bash
# capture-states.sh -- launch the built dim-electron app against an
# already-seeded product home and capture the D6/D7 reference states plus the
# search-overlay motion frames.
#
# States: hover on a sidebar session row, Tab focus on the composer attach (+)
# button, the session context menu, and the stalled-backend "loading" case
# (daemon SIGSTOPped while a session row is clicked).
# Motion: open/close frames of the search dialog at 0/75/150/250 ms, captured
# through the CDP Animation domain at 0.02x so the OS window screenshot can
# land on each phase.
#
# Requires a seeded home (see docs/evidence/electron-reference/README.md):
#   WORK=/tmp/dim-state-capture
# The product home is never wiped; only the temp userData is.
#
# Usage:
#   PRODUCT_HOME=$WORK/home DIMCODE_USER_DATA_PATH=$WORK/userdata \
#     ./capture-states.sh
set -uo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
OUT="$(cd "$DIR/.." && pwd)"
WORK="${ELECTRON_STATES_WORK:-/tmp/dim-state-capture}"
PRODUCT_HOME="${PRODUCT_HOME:?PRODUCT_HOME must point at an already-seeded product home}"
USER_DATA="${DIMCODE_USER_DATA_PATH:-$WORK/userdata}"
CDP_PORT="${CDP_PORT:-9333}"
LOG="$WORK/app-states.log"

ELECTRON=/Users/eric8810/Code/dim-electron/node_modules/.pnpm/electron@43.6.0/node_modules/electron/dist/Electron.app/Contents/MacOS/Electron
MAIN=/Users/eric8810/Code/dim-electron/packages/desktop/packages/electron-main/dist/app/main.cjs
REFERENCE_REPO=/Users/eric8810/Code/dim-electron
APP_SERVER_BIN="${DIMCODE_APP_SERVER_BIN:-/tmp/dim-2-target/debug/dim-app-server}"

mkdir -p "$WORK" "$OUT"
if [ "${KEEP_USER_DATA:-0}" != "1" ]; then
  rm -rf "$USER_DATA"
fi
mkdir -p "$USER_DATA"
if [ ! -x "$WORK/winlist" ]; then
  swiftc -O "$DIR/winlist.swift" -o "$WORK/winlist" || exit 1
fi

export DIMCODE_HOME="$PRODUCT_HOME"
export DIMCODE_USER_DATA_PATH="$USER_DATA"
export DIMCODE_APP_SERVER_BIN="$APP_SERVER_BIN"
export DIMCODE_RENDERER_URL="file:///Users/eric8810/Code/dim-electron/packages/desktop/packages/renderer/dist/index.html"
export DIMCODE_PRELOAD_PATH="/Users/eric8810/Code/dim-electron/packages/desktop/packages/electron-main/dist/preload.cjs"
export DIMCODE_CHROME_HELPER_PATH="/Users/eric8810/Code/dim-electron/packages/desktop/packages/electron-main/dist/bin/dimcode-chrome-native-host.cjs"
export DIMCODE_RESOURCES_ROOT="/Users/eric8810/Code/dim-electron/packages/desktop"
export DIMCODE_DEV_MODE=0
export DIMCODE_DEVTOOLS=0
export DIMCODE_APP_VERSION=0.9.57
export CDP_PORT

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
  for pid in $(ps -axo pid=,command= | grep -F "$PRODUCT_HOME" | grep -F 'dim-app-server' | awk '{print $1}'); do
    kill -CONT "$pid" 2>/dev/null || true
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

daemon_pid() {
  ps -axo pid=,command= | grep -F "$PRODUCT_HOME" | grep -F 'dim-app-server' | grep -v 'grep' | awk 'NR==1 {print $1}'
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
# The restored window can be re-placed (and change display/DPR) a few seconds
# after it first appears. Wait until the row stops changing before driving it.
stable=0; prev="$LINE"
for _ in $(seq 1 20); do
  sleep 1
  cur="$(window_line)"
  if [ -n "$cur" ] && [ "$cur" = "$prev" ]; then
    stable=$((stable + 1))
    [ "$stable" -ge 3 ] && break
  else
    stable=0
    prev="$cur"
  fi
done
LINE="$(window_line)"
echo "window settled: $LINE"
sleep 2

shot() {
  local name="$1" action="$2" facts_file="${3:-}" delay="${4:-1}"
  local line wid started png px daemon facts
  sleep "$delay"
  line="$(window_line)"
  [ -z "$line" ] && { echo "window disappeared before $name" >&2; exit 1; }
  wid="$(printf '%s' "$line" | cut -f1)"
  started="$(date '+%Y-%m-%d %H:%M:%S %Z')"
  png="$OUT/$name.png"
  screencapture -x -o -l "$wid" "$png"
  if [ ! -s "$png" ]; then
    echo "screencapture produced no file for $name (Screen Recording permission?)" >&2
    exit 1
  fi
  px="$(sips -g pixelWidth -g pixelHeight "$png" 2>/dev/null | awk '/pixel/ {printf "%s ", $2}')"
  daemon="$(ps -axo pid=,command= | grep -F "$PRODUCT_HOME" | grep -F 'dim-app-server' | grep -v 'grep' | head -1)"
  facts=""
  if [ -n "$facts_file" ] && [ -s "$facts_file" ]; then
    facts="$(tr '\n' ' ' <"$facts_file" | sed 's/[[:space:]]\+/ /g')"
  fi
  {
    printf 'state\t%s\n' "$name"
    printf 'action\t%s\n' "$action"
    printf 'captured_at\t%s\n' "$started"
    printf 'app_pid\t%s\n' "$APP_PID"
    printf 'window_id\t%s\n' "$wid"
    printf 'window_row\t%s\n' "${line//$'\t'/ | }"
    printf 'window_points\t%s\n' "$(printf '%s' "$line" | cut -f3 | awk -F, '{print $3"x"$4}')"
    printf 'png\t%s\n' "$png"
    printf 'png_pixels\t%s\n' "${px% }"
    printf 'png_bytes\t%s\n' "$(wc -c <"$png" | tr -d ' ')"
    printf 'png_md5\t%s\n' "$(md5 -q "$png")"
    printf 'dimcode_home\t%s\n' "$PRODUCT_HOME"
    printf 'user_data\t%s\n' "$USER_DATA"
    printf 'daemon\t%s\n' "${daemon:-<none>}"
    printf 'app_server_bin\t%s\n' "$APP_SERVER_BIN"
    printf 'git_head\t%s\n' "$(git -C "$OUT/../../.." rev-parse HEAD 2>/dev/null || echo unknown)"
    printf 'reference_head\t%s\n' "$(git -C "$REFERENCE_REPO" rev-parse HEAD 2>/dev/null || echo unknown)"
    printf 'seed_session\t%s\n' "${SEED_SESSION_ID:-<unset>}"
    printf 'seed_side_branch\t%s\n' "${SEED_SIDE_BRANCH_ID:-<unset>}"
    printf 'seed_prompt\t%s\n' "${SEED_PROMPT:-<unset>}"
    if [ -n "$facts" ]; then
      printf 'dom\t%s\n' "$facts"
    fi
  } >"$OUT/$name.txt"
  echo "shot $name: window $wid, $px px, $(wc -c <"$png" | tr -d ' ') bytes"
}

# Start from a rendered conversation in dark mode.
node "$DIR/interact.mjs" dismiss-onboarding >"$WORK/dismiss-onboarding.log" 2>&1 || true
node "$DIR/interact-conversation.mjs" click-session >"$WORK/click-session.log" 2>&1 || true
node "$DIR/interact-conversation.mjs" dark >"$WORK/dark.log" 2>&1 || true
node "$DIR/interact-states.mjs" state >"$WORK/state-before.json" 2>&1 || true

node "$DIR/interact-states.mjs" hover-row 1 >"$WORK/hover-row.json" 2>&1
shot window-hover "CDP Input.dispatchMouseEvent: pointer parked on the middle of sidebar session row 2; row takes the hover background and replaces the relative time with the pin (left) and archive (right) affordances" "$WORK/hover-row.json" 0

node "$DIR/interact-states.mjs" focus-attach >"$WORK/focus-attach.json" 2>&1
shot window-focus "click [data-composer-editor-input] then one Tab: focus-visible ring (3px ring/50) on the composer attach (+) button; the tooltip 添加图片或文件 opens on focus" "$WORK/focus-attach.json" 0

node "$DIR/interact-states.mjs" open-menu 0 3000 >"$WORK/open-menu.json" 2>&1
shot window-menu "right-click sidebar session row 1: session context menu (置顶 / 重命名 / 在新窗口中打开 / 导出 / 归档 / 删除); the rename item's label and disabled state at capture time are in the dom line (rename readiness is checked when the menu opens)" "$WORK/open-menu.json" 0
node "$DIR/interact-states.mjs" close-menu >"$WORK/close-menu.json" 2>&1

DAEMON_PID="$(daemon_pid)"
if [ -z "$DAEMON_PID" ]; then
  echo "no daemon found for $PRODUCT_HOME; cannot capture the stalled-backend state" >&2
  exit 1
fi
kill -STOP "$DAEMON_PID"
node "$DIR/interact-states.mjs" click-row 1 300 >"$WORK/loading.json" 2>&1
shot window-loading-session-switch "SIGSTOP daemon $DAEMON_PID (the run's own --product-home $PRODUCT_HOME), then click sidebar session row 2 and screenshot 0.3 s later; the window still shows the previous session and no loading row/skeleton, because the renderer waits for session.open before switching. The restore-time skeleton is captured separately as window-loading.png" "$WORK/loading.json" 0
kill -CONT "$DAEMON_PID"
sleep 3

# Motion: the search overlay is the app's animated surface. Frame capture needs
# the window id and writes the PNGs plus motion-search-frames.json into $OUT.
MOTION_LINE="$(window_line)"
MOTION_WID="$(printf '%s' "$MOTION_LINE" | cut -f1)"
node "$DIR/interact-states.mjs" motion "$MOTION_WID" "$OUT" motion-search >"$WORK/motion.json" 2>&1
MOTION_STATUS=$?
{
  printf 'state\tmotion-search\n'
  printf 'action\topen sidebar 搜索 (data-testid=sidebar-search), then Escape; frames at 0/75/150/250 ms of the opening and closing CSS animations\n'
  printf 'captured_at\t%s\n' "$(date '+%Y-%m-%d %H:%M:%S %Z')"
  printf 'app_pid\t%s\n' "$APP_PID"
  printf 'window_id\t%s\n' "$MOTION_WID"
  printf 'window_row\t%s\n' "${MOTION_LINE//$'\t'/ | }"
  printf 'window_points\t%s\n' "$(printf '%s' "$MOTION_LINE" | cut -f3 | awk -F, '{print $3"x"$4}')"
  printf 'png_pattern\t%s\n' "$OUT/motion-search-{000,075,150,250}.png and $OUT/motion-search-close-{000,075,150,250}.png"
  printf 'frame_metadata\t%s\n' "$OUT/motion-search-frames.json"
  printf 'method\tCDP Animation.setPlaybackRate(0.02) slow-motion, screencapture -x -o -l <window id>; actual animation times are the Animation.currentTime values read immediately before/after each screenshot (see frame_metadata)\n'
  printf 'measured_motion\topening: overlay CSSAnimation enter 150 ms (keyframe easing ease, opacity 0->1); dialog CSSAnimation enter 200 ms (keyframe easing ease, opacity 0->1 and scale 0.95->1); duration-200 comes from the dialog class, the overlay has no duration class and uses the tw-animate 150 ms default\n'
  printf 'measured_motion_close\toverlay/dialog exit animations measured on Escape; see frame_metadata closeTiming\n'
  printf 'dimcode_home\t%s\n' "$PRODUCT_HOME"
  printf 'user_data\t%s\n' "$USER_DATA"
  printf 'app_server_bin\t%s\n' "$APP_SERVER_BIN"
  printf 'git_head\t%s\n' "$(git -C "$OUT/../../.." rev-parse HEAD 2>/dev/null || echo unknown)"
  printf 'reference_head\t%s\n' "$(git -C "$REFERENCE_REPO" rev-parse HEAD 2>/dev/null || echo unknown)"
  printf 'motion_exit_status\t%s\n' "$MOTION_STATUS"
  printf 'caption_limits\tthe frame PNGs are real window captures made while the CDP Animation domain ran the same CSS animations at 0.02x; wall-clock offsets are only a scheduling aid, so read frame_metadata for the actual animation millisecond of each PNG. The search dialog is the only animated surface captured here (no panel-switch frames).\n'
  if [ -s "$WORK/motion.json" ]; then
    printf 'dom\t%s\n' "$(tr '\n' ' ' <"$WORK/motion.json" | sed 's/[[:space:]]\+/ /g')"
  fi
} >"$OUT/motion-search.txt"
if [ "$MOTION_STATUS" -ne 0 ]; then
  echo "motion capture failed; see $WORK/motion.json" >&2
  exit 1
fi

echo "done"
cleanup
trap - EXIT INT TERM
kill -0 "$APP_PID" 2>/dev/null && echo "app still alive" || echo "app stopped"
LEFT="$(ps -axo pid=,command= | grep -F "$PRODUCT_HOME" | grep -F 'dim-app-server' | grep -v 'grep')"
[ -n "$LEFT" ] && echo "leftover daemon: $LEFT" || echo "no leftover daemon from this run"
