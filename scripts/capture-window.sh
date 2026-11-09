#!/usr/bin/env bash
# capture-window.sh captures real screenshots of the MyGo DimAgent window.
#
# Two states are proven:
#   disconnected  no DIMAGENT_PRODUCT_HOME: the window must say "App Server: Not connected"
#   connected     a throwaway DIMAGENT_PRODUCT_HOME plus a real dim-app-server spawned by
#                 the supervisor: the window must say "App Server: Connected"
#
# Safety rules this script keeps:
#   * it never sets DIMAGENT_PRODUCT_HOME to a real user home; every connected run uses
#     its own mktemp -d directory that is deleted afterwards
#   * it only signals the PIDs it started: the app PID, and daemons whose command line
#     contains that run's throwaway home. Other dim-app-server processes are reported,
#     never killed
#   * it fails loudly when the App Server binary is missing instead of faking a
#     connected window
#
# Usage:
#   scripts/capture-window.sh [--state disconnected|connected|both] [--out DIR]
#                             [--server-dir DIR] [--wait SECONDS]
#
# Exit status is non-zero when a requested capture could not be produced.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
STATE="both"
OUT_DIR="$REPO_ROOT/docs/evidence"
SERVER_DIR="${DIMAGENT_APP_SERVER_DIR:-/tmp/dim-2-target/debug}"
WAIT_SECONDS=30
PRODUCT_HOME_DIR=""

while [ "$#" -gt 0 ]; do
  case "$1" in
    --state) STATE="${2:?--state needs a value}"; shift 2 ;;
    --out) OUT_DIR="${2:?--out needs a value}"; shift 2 ;;
    --server-dir) SERVER_DIR="${2:?--server-dir needs a value}"; shift 2 ;;
    --wait) WAIT_SECONDS="${2:?--wait needs a value}"; shift 2 ;;
    --product-home) PRODUCT_HOME_DIR="${2:?--product-home needs a value}"; shift 2 ;;
    -h|--help) sed -n '2,30p' "${BASH_SOURCE[0]}"; exit 0 ;;
    *) echo "capture-window: unknown argument: $1" >&2; exit 2 ;;
  esac
done

case "$STATE" in
  disconnected|connected|both) ;;
  *) echo "capture-window: --state must be disconnected, connected or both" >&2; exit 2 ;;
esac

WORK="$(mktemp -d "${TMPDIR:-/tmp}/dimagent-capture.XXXXXX")"
# A capture must not inherit the reviewer's theme or window position: point the
# desktop preferences at a throwaway config with the light theme, and open the
# window on the primary display so every capture has the same geometry.
CAPTURE_CONFIG="$WORK/config"
mkdir -p "$CAPTURE_CONFIG/dimagent"
printf '{"version":1,"theme":1}\n' >"$CAPTURE_CONFIG/dimagent/settings.json"
APP_PID=""
RUN_HOME=""
HELPER=""

cleanup() {
  local status=$?
  if [ -n "$APP_PID" ] && kill -0 "$APP_PID" 2>/dev/null; then
    kill -TERM "$APP_PID" 2>/dev/null || true
    for _ in $(seq 1 50); do
      kill -0 "$APP_PID" 2>/dev/null || break
      sleep 0.1
    done
    kill -KILL "$APP_PID" 2>/dev/null || true
  fi
  if [ -n "$RUN_HOME" ]; then
    stop_run_daemons "$RUN_HOME"
  fi
  if [ -n "$WORK" ] && [ -d "$WORK" ]; then
    rm -rf "$WORK"
  fi
  exit "$status"
}
trap cleanup EXIT INT TERM

log() { printf '%s\n' "$*"; }

# stop_run_daemons signals only daemons whose command line names this run's home.
stop_run_daemons() {
  local home="$1" pid
  for pid in $(run_daemons "$home"); do
    kill -TERM "$pid" 2>/dev/null || true
  done
  for _ in $(seq 1 50); do
    [ -z "$(run_daemons "$home")" ] && return 0
    sleep 0.1
  done
  for pid in $(run_daemons "$home"); do
    kill -KILL "$pid" 2>/dev/null || true
  done
}

run_daemons() {
  local home="$1"
  ps -axo pid=,command= | grep -F -- "$home" | grep -F 'dim-app-server' | awk '{print $1}' || true
}

# The window-id helper lists CGWindowInfo for one PID. macOS has no built-in
# command for that, so it is compiled from a heredoc once per run.
build_window_helper() {
  local src="$WORK/winlist.swift"
  cat >"$src" <<'SWIFT'
import CoreGraphics
import Foundation

guard CommandLine.arguments.count > 1, let want = Int(CommandLine.arguments[1]) else {
    exit(2)
}
let opts: CGWindowListOption = [.optionAll, .excludeDesktopElements]
guard let list = CGWindowListCopyWindowInfo(opts, kCGNullWindowID) as? [[String: Any]] else {
    exit(1)
}
for w in list {
    guard let pid = w[kCGWindowOwnerPID as String] as? Int, pid == want else { continue }
    let num = w[kCGWindowNumber as String] as? Int ?? -1
    let layer = w[kCGWindowLayer as String] as? Int ?? -1
    let name = w[kCGWindowName as String] as? String ?? ""
    var bounds = "0,0,0,0"
    if let b = w[kCGWindowBounds as String] as? [String: Any],
       let x = b["X"] as? Double, let y = b["Y"] as? Double,
       let wd = b["Width"] as? Double, let ht = b["Height"] as? Double {
        bounds = "\(Int(x)),\(Int(y)),\(Int(wd)),\(Int(ht))"
    }
    print("\(num)\t\(layer)\t\(bounds)\t\(name)")
}
SWIFT
  if ! command -v swiftc >/dev/null 2>&1; then
    echo "capture-window: swiftc not found; install Xcode command line tools to read the window id" >&2
    return 1
  fi
  swiftc -O "$src" -o "$WORK/winlist" >/dev/null 2>&1 || {
    echo "capture-window: could not compile the window-id helper" >&2
    return 1
  }
  HELPER="$WORK/winlist"
}

# pick_window prints "<windowid>\t<layer>\t<bounds>\t<title>" for the window
# whose title is the app's own ("DimAgent"); a titleless window is only used
# when no titled one exists. A window the server reports with zero or tiny
# bounds is still animating in, so it is skipped until it has a real size.
pick_window() {
  "$HELPER" "$1" | awk -F'\t' '
    $2 != 0 { next }
    {
      split($3, b, ",")
      if (b[3] + 0 < 100 || b[4] + 0 < 100) next
      if ($4 != "" && titled == "") titled = $0
      else if ($4 == "" && untitled == "") untitled = $0
    }
    END {
      if (titled != "") print titled
      else if (untitled != "") print untitled
    }
  '
}

wait_for_window() {
  local pid="$1" deadline=$((SECONDS + WAIT_SECONDS)) line
  while [ "$SECONDS" -lt "$deadline" ]; do
    if ! kill -0 "$pid" 2>/dev/null; then
      echo "capture-window: app exited before showing a window" >&2
      return 1
    fi
    line="$(pick_window "$pid")"
    if [ -n "$line" ]; then
      printf '%s\n' "$line"
      return 0
    fi
    sleep 0.5
  done
  echo "capture-window: no window appeared within ${WAIT_SECONDS}s" >&2
  return 1
}

capture_state() {
  local name="$1" product_home="$2"
  local meta="$OUT_DIR/window-$name.txt"
  local png="$OUT_DIR/window-$name.png"
  local log_file="$WORK/app-$name.log"

  APP_PID=""
  RUN_HOME="$product_home"

  log "== $name =="
  local -a env_prefix=()
  if [ -n "$product_home" ]; then
    env_prefix=(env "DIMAGENT_PRODUCT_HOME=$product_home")
    log "product home: $product_home"
  else
    # An unset variable cannot leak a real home in; the app then shows the
    # unconnected state.
    env_prefix=(env DIMAGENT_PRODUCT_HOME=)
    log "product home: (none)"
  fi

  "${env_prefix[@]}" \
    "DIMAGENT_CONFIG_HOME=$CAPTURE_CONFIG" \
    "DIMAGENT_WINDOW_X=40" "DIMAGENT_WINDOW_Y=40" \
    "$WORK/dimagent" >"$log_file" 2>&1 &
  APP_PID=$!
  log "app pid: $APP_PID"

  local started_at
  started_at="$(date '+%Y-%m-%d %H:%M:%S %Z')"

  local line window_id layer bounds title
  line="$(wait_for_window "$APP_PID")"
  IFS=$'\t' read -r window_id layer bounds title <<<"$line"

  # Let the first frame settle before capturing.
  sleep 1

  local captured_at
  captured_at="$(date '+%Y-%m-%d %H:%M:%S %Z')"

  screencapture -x -o -l "$window_id" "$png"

  if [ ! -s "$png" ]; then
    echo "capture-window: screencapture produced no file (Screen Recording permission?)" >&2
    return 1
  fi

  local px
  px="$(sips -g pixelWidth -g pixelHeight "$png" 2>/dev/null | awk '/pixel/ {printf "%s ", $2}')"

  local daemons
  daemons="$(ps -axo pid=,command= | grep -F -- "${product_home:-/nonexistent}" | grep -F 'dim-app-server' || true)"

  {
    printf 'state\t%s\n' "$name"
    printf 'started_at\t%s\n' "$started_at"
    printf 'captured_at\t%s\n' "$captured_at"
    printf 'built_at\t%s\n' "$BUILT_AT"
    printf 'git_head\t%s\n' "$GIT_HEAD"
    printf 'git_dirty\t%s\n' "${GIT_DIRTY:-clean}"
    printf 'product_home\t%s\n' "${product_home:-<unset>}"
    printf 'app_pid\t%s\n' "$APP_PID"
    printf 'window_id\t%s\n' "$window_id"
    printf 'window_title\t%s\n' "$title"
    printf 'window_bounds_points\t%s\n' "$bounds"
    printf 'png\t%s\n' "$png"
    printf 'png_pixels\t%s\n' "${px% }"
    printf 'png_bytes\t%s\n' "$(wc -c <"$png" | tr -d ' ')"
    printf 'daemon\t%s\n' "${daemons:-<none>}"
    printf 'app_log\t%s\n' "$(sed -n '1,20p' "$log_file" | tr '\n' ' ')"
  } >"$meta"

  log "window: id=$window_id title=$title bounds=$bounds (points)"
  log "screenshot: $png ($px px)"
  log "metadata: $meta"

  # Stop the app, then only this run's daemon, before the next state.
  if kill -0 "$APP_PID" 2>/dev/null; then
    kill -TERM "$APP_PID" 2>/dev/null || true
    for _ in $(seq 1 50); do
      kill -0 "$APP_PID" 2>/dev/null || break
      sleep 0.1
    done
    kill -KILL "$APP_PID" 2>/dev/null || true
  fi
  APP_PID=""

  if [ -n "$product_home" ]; then
    stop_run_daemons "$product_home"
  fi
  RUN_HOME=""

  log "capture written: $png"
  log ""
}

if [ ! -x "$SERVER_DIR/dim-app-server" ] && [ "$STATE" != "disconnected" ]; then
  cat >&2 <<EOF
capture-window: no App Server binary at $SERVER_DIR/dim-app-server
The connected state cannot be captured without one, and it is not faked here.
Build it, or pass --server-dir DIR / set DIMAGENT_APP_SERVER_DIR.
EOF
  [ "$STATE" = "connected" ] && exit 1
  log "capture-window: connected state will be skipped (disconnected only)"
  STATE="disconnected"
fi

if ! command -v go >/dev/null 2>&1; then
  echo "capture-window: go not found on PATH" >&2
  exit 1
fi

mkdir -p "$OUT_DIR"
log "building ./cmd/dimagent"
(cd "$REPO_ROOT" && go build -o "$WORK/dimagent" ./cmd/dimagent)
build_window_helper

# The evidence is only meaningful for the source it was built from, and this
# tree may carry other people's uncommitted work. Record both.
GIT_HEAD="$(git -C "$REPO_ROOT" rev-parse HEAD 2>/dev/null || echo unknown)"
GIT_DIRTY="$(git -C "$REPO_ROOT" status --short 2>/dev/null | tr '\n' ';' || true)"
BUILT_AT="$(date '+%Y-%m-%d %H:%M:%S %Z')"

# The supervisor looks up dim-app-server on PATH when no path is configured.
export PATH="$SERVER_DIR:$PATH"
if [ "$STATE" = "disconnected" ] || [ "$STATE" = "both" ]; then
  capture_state disconnected ""
fi
if [ "$STATE" = "connected" ] || [ "$STATE" = "both" ]; then
  # capture_state clears RUN_HOME when it finishes, so the directory to delete
  # is kept in its own variable.
  if [ -n "$PRODUCT_HOME_DIR" ]; then
    # A caller-provided home is reused as-is (for example a seeded one) and is
    # never deleted here: it belongs to the caller.
    capture_state connected "$PRODUCT_HOME_DIR"
  else
    connected_home="$(mktemp -d "${TMPDIR:-/tmp}/dimagent-product-home.XXXXXX")"
    capture_state connected "$connected_home"
    rm -rf "$connected_home"
  fi
fi

# Final honesty check: no daemon of ours may survive the run.
leftover="$(ps -axo pid=,command= | grep -F 'dimagent-product-home' | grep -F 'dim-app-server' || true)"
if [ -n "$leftover" ]; then
  echo "capture-window: leftover daemon(s) from this run:" >&2
  echo "$leftover" >&2
  exit 1
fi

log "all requested captures done; no leftover daemon from this run"
log "screenshots:"
ls -l "$OUT_DIR"/window-*.png
