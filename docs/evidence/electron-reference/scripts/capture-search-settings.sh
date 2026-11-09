#!/usr/bin/env bash
# capture-search-settings.sh -- launch the built dim-electron app against an
# already-seeded product home and capture the search overlay + the settings
# route (entry, sub-page, back).
#
# Requires a seeded home (see docs/evidence/electron-reference/README.md):
#   WORK=/tmp/dim-search-settings-capture
# The product home is never wiped; only the temp userData is.
#
# Usage:
#   PRODUCT_HOME=$WORK/home DIMCODE_USER_DATA_PATH=$WORK/userdata \
#     ./capture-search-settings.sh
set -uo pipefail

DIR="$(cd "$(dirname "$0")" && pwd)"
OUT="$(cd "$DIR/.." && pwd)"
WORK="${ELECTRON_SEARCH_WORK:-/tmp/dim-search-settings-capture}"
PRODUCT_HOME="${PRODUCT_HOME:?PRODUCT_HOME must point at an already-seeded product home}"
USER_DATA="${DIMCODE_USER_DATA_PATH:-$WORK/userdata}"
SEARCH_QUERY="${SEARCH_QUERY:-Explain}"
CDP_PORT="${CDP_PORT:-9333}"
LOG="$WORK/app-search-settings.log"

ELECTRON=/Users/eric8810/Code/dim-electron/node_modules/.pnpm/electron@43.6.0/node_modules/electron/dist/Electron.app/Contents/MacOS/Electron
MAIN=/Users/eric8810/Code/dim-electron/packages/desktop/packages/electron-main/dist/app/main.cjs
REFERENCE_REPO=/Users/eric8810/Code/dim-electron

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
export DIMCODE_APP_SERVER_BIN=/tmp/dim-2-target/debug/dim-app-server
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
# The restored window can be re-placed (and change display/DPR) a few seconds
# after it first appears, which would make the shot set mix 1x and 2x PNGs.
# Wait until the window row stops changing before driving it.
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
  local name="$1" action="$2" detail="${3:-}"
  local line wid started png px daemon
  sleep 1
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
  daemon="$(ps -axo pid=,command= | grep -F "$PRODUCT_HOME" | grep -F 'dim-app-server' | head -1)"
  {
    printf 'state\t%s\n' "$name"
    printf 'action\t%s\n' "$action"
    printf 'command\t%s\n' "PRODUCT_HOME=$PRODUCT_HOME DIMCODE_USER_DATA_PATH=$USER_DATA $DIR/capture-search-settings.sh"
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
  if [ -n "$detail" ] && [ -s "$detail" ]; then
    {
      printf 'dom_route\t%s\n' "$(jq -c '(.route // .after // {}) | {hash, pathname}' "$detail" 2>/dev/null || echo unknown)"
      if jq -e 'has("before") and (.before | has("hash"))' "$detail" >/dev/null 2>&1; then
        printf 'route_before\t%s\n' "$(jq -r '.before.hash' "$detail")"
        printf 'route_after\t%s\n' "$(jq -r '.after.hash' "$detail")"
      fi
      jq -r '
        def flat(prefix): to_entries[] | "\(prefix).\(.key)\t\(.value | if type == "string" or type == "number" or type == "boolean" then . else tojson end)";
        if .dialog then (.dialog | flat("dialog")) elif .facts then (.facts | flat("facts")) elif .controls then (.controls | flat("controls")) elif .sessionsAfter then ((.sessionsAfter | flat("after")), (.sessionsBefore | flat("before"))) elif .sidebarSearch then (.sidebarSearch | flat("sidebarSearch")) else empty end
      ' "$detail" 2>/dev/null
    } >>"$OUT/$name.txt"
    cp "$detail" "$OUT/$name.json"
  fi
  echo "shot $name: window $wid, $px px, $(wc -c <"$png" | tr -d ' ') bytes"
}

run_action() {
  local action="$1" detail="$2"
  shift 2
  if ! node "$DIR/interact-search-settings.mjs" "$action" "$@" >"$detail"; then
    echo "$action failed; refusing to shoot" >&2
    cat "$detail" >&2
    exit 1
  fi
  cat "$detail"
}

node "$DIR/interact-search-settings.mjs" click-session >"$WORK/detail-conversation.json"
cat "$WORK/detail-conversation.json"

if [ "${APPEARANCE_ONLY:-0}" = "1" ]; then
  # Re-capture just settings · 外观: enter #/settings from the conversation,
  # then click the nav entry and verify the pane heading before shooting.
  run_action open-settings "$WORK/detail-window-settings-route.json"
  run_action open-appearance "$WORK/detail-window-settings-appearance.json"
  shot window-settings-appearance "click settings nav 外观 ([data-testid=settings-nav-appearance]) from #/settings" "$WORK/detail-window-settings-appearance.json"
  run_action scroll-appearance "$WORK/detail-window-settings-appearance-2.json"
  shot window-settings-appearance-2 "scroll settings · 外观 content pane to the bottom" "$WORK/detail-window-settings-appearance-2.json"
  cleanup
  trap - EXIT INT TERM
  kill -0 "$APP_PID" 2>/dev/null && echo "app still alive" || echo "app stopped"
  LEFT="$(ps -axo pid=,command= | grep -F "$PRODUCT_HOME" | grep -F 'dim-app-server')"
  if [ -n "$LEFT" ]; then echo "leftover daemon: $LEFT"; else echo "no leftover daemon from this run"; fi
  exit 0
fi

run_action hover-search "$WORK/detail-hover.json"

run_action open-search "$WORK/detail-window-search.json"
shot window-search "click [data-testid=sidebar-search] (⌘G entry) from the seeded conversation" "$WORK/detail-window-search.json"

run_action type-query "$WORK/detail-window-search-results.json" "$SEARCH_QUERY"
shot window-search-results "typed ${SEARCH_QUERY} into [data-testid=chat-search-input]" "$WORK/detail-window-search-results.json"

run_action escape-search "$WORK/detail-window-search-escape.json"

# Interaction probes: the ⌘G shortcut path, the empty state and the scrim
# click-to-dismiss. They produce no screenshot; their facts are merged into
# window-search-interactions.json and appended to window-search.txt.
run_action shortcut-open "$WORK/detail-search-shortcut.json"
run_action empty-state "$WORK/detail-search-empty.json"
run_action escape-search "$WORK/detail-search-escape2.json"
run_action open-search "$WORK/detail-search-reopen.json"
run_action backdrop-click "$WORK/detail-search-backdrop.json"
jq -s '{shortcut_open: .[0], empty_state: .[1], escape_after_shortcut: .[2], reopened_by_click: .[3].dialog, backdrop_click: .[4]}' \
  "$WORK/detail-search-shortcut.json" "$WORK/detail-search-empty.json" "$WORK/detail-search-escape2.json" \
  "$WORK/detail-search-reopen.json" "$WORK/detail-search-backdrop.json" \
  >"$OUT/window-search-interactions.json"
{
  printf 'interaction_shortcut_open\t%s\n' "$(jq -c '{hash: .shortcut_open.route.hash, dialog: .shortcut_open.dialog.dialogPresent, focused: .shortcut_open.dialog.focusedTestId, rowCount: .shortcut_open.dialog.rowCount, highlightedFirst: (.shortcut_open.dialog.rows[0].cls | contains("bg-muted"))}' "$OUT/window-search-interactions.json")"
  printf 'interaction_escape\t%s\n' "$(jq -c '{dialogCount: .escape_after_shortcut.dialogCountAfterEscape, focused: .escape_after_shortcut.focusedAfterEscape, hash: .escape_after_shortcut.after.hash}' "$OUT/window-search-interactions.json")"
  printf 'interaction_empty_state\t%s\n' "$(jq -c '{hash: .empty_state.route.hash, rowCount: .empty_state.dialog.rowCount, groups: .empty_state.dialog.groupLabels, emptyText: .empty_state.dialog.emptyState}' "$OUT/window-search-interactions.json")"
  printf 'interaction_backdrop_click\t%s\n' "$(jq -c '{dialogCount: .backdrop_click.dialogCountAfterBackdropClick, focused: .backdrop_click.focusedAfterBackdropClick, hash: .backdrop_click.after.hash}' "$OUT/window-search-interactions.json")"
} >>"$OUT/window-search.txt"

run_action open-settings "$WORK/detail-window-settings-route.json"
shot window-settings-route "click [data-testid=sidebar-account-settings-shortcut] (left-bottom gear) from the conversation" "$WORK/detail-window-settings-route.json"

run_action open-appearance "$WORK/detail-window-settings-appearance.json"
# window-settings-appearance* belongs to the appearance slice (run this same
# script with APPEARANCE_ONLY=1); this run only needs 外观 as the section the
# 返回 test starts from, so it must not overwrite those files.
if [ "${APPEARANCE_SHOT:-0}" = "1" ]; then
  shot window-settings-appearance "click settings nav 外观 (appearance section)" "$WORK/detail-window-settings-appearance.json"
fi

run_action back "$WORK/detail-window-settings-back.json"
shot window-settings-back "click 返回 ([data-testid=settings-back]) from settings · 外观" "$WORK/detail-window-settings-back.json"

cleanup
trap - EXIT INT TERM
kill -0 "$APP_PID" 2>/dev/null && echo "app still alive" || echo "app stopped"
LEFT="$(ps -axo pid=,command= | grep -F "$PRODUCT_HOME" | grep -F 'dim-app-server')"
if [ -n "$LEFT" ]; then echo "leftover daemon: $LEFT"; else echo "no leftover daemon from this run"; fi
