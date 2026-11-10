#!/usr/bin/env bash
# 用 /tmp 夹具里的真实 provider 配置跑一次会话，验证模型确实产生工具调用。
#
# 前提：夹具由人工准备（auth.json、config.json、home/rust/dimcode.sqlite、
# bin/dimcode-server）。本脚本不读取 ~/.dimcode-alpha 或其他来源目录，也不把
# 夹具内容写回任何来源；每次运行都把 fixture/home 复制到新的临时目录。
#
# 用法：
#   scripts/verify-real-model.sh [--fixture DIR] [--server PATH]
#                                [--prompt TEXT] [--expect TEXT]
#                                [--timeout SECONDS] [--keep]
#
# 成功：工具调用出现且（若给了 --expect）工具结果包含期望文本；退出码 0。
# 失败：没有工具调用、没有工具结果、账号未登录、daemon 启动失败；退出码 1。
set -euo pipefail

FIXTURE="/tmp/dimagent-real-model"
SERVER=""
PROMPT='Use the shell tool to run exactly: printf DIMAGENT_REAL_TOOL_OK. You must call the shell tool; do not answer from memory. Then report the tool output.'
EXPECT="DIMAGENT_REAL_TOOL_OK"
TIMEOUT_SECONDS=180
KEEP=0

while [ $# -gt 0 ]; do
  case "$1" in
    --fixture) FIXTURE="$2"; shift 2 ;;
    --server) SERVER="$2"; shift 2 ;;
    --prompt) PROMPT="$2"; shift 2 ;;
    --expect) EXPECT="$2"; shift 2 ;;
    --timeout) TIMEOUT_SECONDS="$2"; shift 2 ;;
    --keep) KEEP=1; shift ;;
    -h|--help)
      sed -n '2,20p' "$0"
      exit 0
      ;;
    *)
      echo "未知参数: $1" >&2
      exit 2
      ;;
  esac
done

if [ ! -d "$FIXTURE/home" ]; then
  echo "夹具缺少 home 目录: $FIXTURE/home" >&2
  exit 1
fi
if [ ! -f "$FIXTURE/home/rust/dimcode.sqlite" ]; then
  echo "夹具缺少原生存储: $FIXTURE/home/rust/dimcode.sqlite" >&2
  exit 1
fi
if [ -z "$SERVER" ]; then
  SERVER="$FIXTURE/bin/dimcode-server"
fi
if [ ! -x "$SERVER" ]; then
  echo "找不到可执行的 daemon: $SERVER" >&2
  exit 1
fi

RUN_DIR="$(mktemp -d /tmp/dimagent-real-model-run.XXXXXX)"
DAEMON_PID=""
cleanup() {
  if [ -n "$DAEMON_PID" ] && kill -0 "$DAEMON_PID" 2>/dev/null; then
    kill "$DAEMON_PID" 2>/dev/null || true
    for _ in $(seq 1 20); do
      kill -0 "$DAEMON_PID" 2>/dev/null || break
      sleep 0.1
    done
  fi
  if [ "$KEEP" -eq 0 ]; then
    rm -rf "$RUN_DIR"
  fi
}
trap cleanup EXIT

cp -a "$FIXTURE/home" "$RUN_DIR/home"
chmod 700 "$RUN_DIR/home" "$RUN_DIR/home/rust" 2>/dev/null || true
rm -f "$RUN_DIR/home/rust/daemon.sock" \
      "$RUN_DIR/home/rust/daemon.sock.lock" \
      "$RUN_DIR/home/rust/daemon.start.lock"

echo "fixture: $FIXTURE"
echo "run:     $RUN_DIR"
echo "daemon:  $SERVER"

"$SERVER" app-server \
  --product-home "$RUN_DIR/home" \
  --home "$RUN_DIR/home/rust" \
  --unix "$RUN_DIR/home/rust/daemon.sock" \
  --no-ws \
  >"$RUN_DIR/daemon.out" 2>"$RUN_DIR/daemon.err" &
DAEMON_PID=$!

for _ in $(seq 1 100); do
  [ -S "$RUN_DIR/home/rust/daemon.sock" ] && break
  kill -0 "$DAEMON_PID" 2>/dev/null || break
  sleep 0.1
done
if [ ! -S "$RUN_DIR/home/rust/daemon.sock" ]; then
  echo "daemon 没有在 10 秒内就绪" >&2
  tail -20 "$RUN_DIR/daemon.err" >&2 || true
  exit 1
fi

ACCOUNT="$("$SERVER" account dimcode-api-oauth --home "$RUN_DIR/home" 2>&1 || true)"
if ! printf '%s' "$ACCOUNT" | grep -q 'Signed in'; then
  echo "账号未登录，无法验证真实模型: $ACCOUNT" >&2
  exit 1
fi
echo "account: $(printf '%s' "$ACCOUNT" | head -1)"

python3 - "$SERVER" "$RUN_DIR" "$PROMPT" "$TIMEOUT_SECONDS" <<'PY'
import json
import os
import subprocess
import sys

server, run_dir, prompt, timeout = sys.argv[1], sys.argv[2], sys.argv[3], int(sys.argv[4])
json_path = os.path.join(run_dir, "exec.json")
err_path = os.path.join(run_dir, "exec.err")
with open(json_path, "wb") as out, open(err_path, "wb") as err:
    try:
        result = subprocess.run(
            [server, "exec", "--home", os.path.join(run_dir, "home"),
             "--permission", "full-access", "--json", "-p", prompt],
            cwd=run_dir, stdout=out, stderr=err, timeout=timeout,
        )
    except subprocess.TimeoutExpired:
        print(f"exec 超时（{timeout}s）", file=sys.stderr)
        sys.exit(1)
if result.returncode != 0:
    print(f"exec 退出码 {result.returncode}", file=sys.stderr)
    sys.exit(1)
PY

SUMMARY="$RUN_DIR/summary.json"
python3 - "$RUN_DIR/exec.json" "$EXPECT" >"$SUMMARY" <<'PY'
import json
import sys

exec_path, expect = sys.argv[1], sys.argv[2]
calls = []
results = []
session = ""
for line in open(exec_path, "r", errors="replace"):
    try:
        message = json.loads(line)
    except json.JSONDecodeError:
        continue
    if message.get("method") != "session.event":
        continue
    params = message.get("params") or {}
    session = params.get("sessionId") or session
    event = params.get("event") or {}
    if event.get("type") != "step":
        continue
    data = event.get("data") or {}
    if event.get("name") == "tool-call":
        for call in data.get("calls") or []:
            calls.append({
                "tool": call.get("name"),
                "input": call.get("input"),
            })
    if event.get("name") == "tool-result":
        for item in data.get("results") or []:
            result = item.get("result") or {}
            results.append(result.get("content"))
summary = {
    "sessionId": session,
    "toolCalls": calls,
    "toolResults": results,
    "expected": expect,
}
summary["ok"] = bool(calls) and (not expect or any(isinstance(r, str) and expect in r for r in results))
print(json.dumps(summary, ensure_ascii=False))
PY

cat "$SUMMARY"
python3 - "$SUMMARY" <<'PY'
import json
import sys
summary = json.load(open(sys.argv[1]))
if not summary.get("ok"):
    print("未观察到真实工具调用或期望的工具结果", file=sys.stderr)
    sys.exit(1)
PY
echo "REAL_TOOL_CALL_OK"
