#!/usr/bin/env python3
"""用 /tmp 夹具跑一次真实模态任务，并记录能力、状态、分页字节数与实时门禁。

数据纪律：
- 来源目录只读；脚本只读 --fixture，把 fixture/home 复制到独立临时目录后运行。
- 证据只打印状态、标识、文件名字节数与分页数；不打印图像内容、OCR 文本或凭据。
- 本机服务端把 modality.* 作为 daemon 级方法声明在 hello.features；脚本按此门禁。

macOS 提示：Apple Vision 的权限按可执行文件路径生效。用夹具里复制出来的
dimcode-server 跑 ocr.recognize 会得到 executor_error；请用已安装路径：
  --server ~/.dimcode-alpha/rust/dimcode-server
或 npm 安装目录里的同一个二进制。

用法：
  scripts/verify-modality.py --fixture /tmp/dimagent-real-model \
      --server ~/.dimcode-alpha/rust/dimcode-server \
      --image /tmp/dimagent-modality-ocr.png
"""

import argparse
import base64
import json
import os
import shutil
import socket
import subprocess
import sys
import tempfile
import time

MODALITY_METHODS = [
    "modality.capabilities",
    "modality.task.submit",
    "modality.task.status",
    "modality.task.list",
    "modality.task.result",
    "modality.task.cancel",
    "modality.realtime.open",
    "modality.realtime.send",
    "modality.realtime.close",
]


class RPC:
    def __init__(self, socket_path):
        self.sock = socket.socket(socket.AF_UNIX, socket.SOCK_STREAM)
        self.sock.connect(socket_path)
        self.file = self.sock.makefile("rwb", buffering=0)
        self.next_id = 0
        self.features = set()
        hello = self.call("hello", {"protocolVersion": 3, "clientKind": "other"})
        self.features = set(hello.get("result", {}).get("features", []))

    def call(self, method, params):
        self.next_id += 1
        request_id = self.next_id
        payload = json.dumps({
            "jsonrpc": "2.0",
            "id": request_id,
            "method": method,
            "params": params,
        })
        self.file.write((payload + "\n").encode())
        while True:
            line = self.file.readline()
            if not line:
                raise RuntimeError("daemon closed the connection")
            message = json.loads(line)
            if message.get("id") == request_id:
                return message


def require_result(reply, method):
    if "error" in reply:
        raise RuntimeError(f"{method} failed: {json.dumps(reply['error'], ensure_ascii=False)}")
    return reply["result"]


def available_operations(snapshot):
    available = []
    for category in snapshot.get("categories", []):
        for operation in category.get("operations", []):
            if operation.get("available"):
                effective = operation.get("effective") or {}
                available.append({
                    "id": f"{category['category']}.{operation['operation']}",
                    "options": len(operation.get("options", [])),
                    "effectiveProvider": effective.get("providerId"),
                    "effectiveModel": effective.get("modelId"),
                })
    return available


def realtime_state(snapshot):
    for category in snapshot.get("categories", []):
        if category.get("category") != "realtime":
            continue
        return {
            "available": category.get("available", False),
            "reasons": category.get("reasons", []),
            "operations": [
                {
                    "operation": operation.get("operation"),
                    "available": operation.get("available", False),
                    "reasons": operation.get("reasons", []),
                }
                for operation in category.get("operations", [])
            ],
        }
    return {"available": False, "reasons": [{"dimension": "catalog", "detail": "realtime category absent"}], "operations": []}


def submit_and_wait(rpc, image_bytes, image_name, media_type, prompt_mode, client_id, timeout):
    params = {
        "category": "ocr",
        "operation": "recognize",
        "params": {"outputMode": prompt_mode},
        "files": [{
            "name": image_name,
            "media_type": media_type,
            "data_base64": base64.b64encode(image_bytes).decode(),
        }],
        "client_task_id": client_id,
    }
    submitted = require_result(rpc.call("modality.task.submit", params), "modality.task.submit")
    task = submitted["task"]
    task_id = task["task_id"]
    states = [task.get("state")]
    deadline = time.time() + timeout
    while time.time() < deadline:
        current = require_result(
            rpc.call("modality.task.status", {"task_id": task_id}),
            "modality.task.status",
        )["task"]
        state = current.get("state")
        if state != states[-1]:
            states.append(state)
        if state in ("succeeded", "failed", "cancelled", "interrupted"):
            return task_id, states, current
        time.sleep(0.15)
    raise RuntimeError(f"task {task_id} did not finish within {timeout}s; states={states}")


def read_results(rpc, task, page_bytes):
    files = ((task.get("result") or {}).get("files") or [])
    reads = []
    for entry in files:
        name = entry.get("name")
        total = 0
        pages = 0
        eof = False
        while not eof and pages < 1000:
            reply = require_result(
                rpc.call("modality.task.result", {
                    "task_id": task["task_id"],
                    "name": name,
                    "offset": total,
                    "length": page_bytes,
                }),
                "modality.task.result",
            )
            chunk = base64.b64decode(reply.get("data_base64", ""))
            total += len(chunk)
            pages += 1
            eof = bool(reply.get("eof"))
        reads.append({"name": name, "bytes": total, "declaredBytes": entry.get("bytes"), "pages": pages, "eof": eof})
    return reads


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--fixture", default="/tmp/dimagent-real-model")
    parser.add_argument("--server", default="")
    parser.add_argument("--image", default="/tmp/dimagent-modality-ocr.png")
    parser.add_argument("--page-bytes", type=int, default=64)
    parser.add_argument("--timeout", type=int, default=120)
    parser.add_argument("--keep", action="store_true")
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()

    if not os.path.isdir(os.path.join(args.fixture, "home")):
        print(f"fixture home missing: {args.fixture}/home", file=sys.stderr)
        return 2
    server = args.server or os.path.join(args.fixture, "bin", "dimcode-server")
    if not os.access(server, os.X_OK):
        print(f"server is not executable: {server}", file=sys.stderr)
        return 2
    if not os.path.isfile(args.image):
        print(f"image missing: {args.image}", file=sys.stderr)
        return 2

    run_dir = tempfile.mkdtemp(prefix="dimagent-modality-run.")
    daemon = None
    try:
        home = os.path.join(run_dir, "home")
        shutil.copytree(os.path.join(args.fixture, "home"), home)
        os.chmod(run_dir, 0o700)
        os.chmod(home, 0o700)
        native = os.path.join(home, "rust")
        os.chmod(native, 0o700)
        for name in ("daemon.sock", "daemon.sock.lock", "daemon.start.lock"):
            try:
                os.unlink(os.path.join(native, name))
            except FileNotFoundError:
                pass
        daemon = subprocess.Popen(
            [
                server, "app-server",
                "--product-home", home,
                "--home", native,
                "--unix", os.path.join(native, "daemon.sock"),
                "--no-ws",
            ],
            stdout=open(os.path.join(run_dir, "daemon.out"), "wb"),
            stderr=open(os.path.join(run_dir, "daemon.err"), "wb"),
            stdin=subprocess.DEVNULL,
            start_new_session=True,
        )
        socket_path = os.path.join(native, "daemon.sock")
        for _ in range(100):
            if os.path.exists(socket_path):
                break
            if daemon.poll() is not None:
                raise RuntimeError("daemon exited before creating its socket")
            time.sleep(0.1)
        if not os.path.exists(socket_path):
            raise RuntimeError("daemon did not create its socket within 10s")

        rpc = RPC(socket_path)
        missing = [name for name in MODALITY_METHODS if name not in rpc.features]
        if missing:
            raise RuntimeError(f"hello.features is missing modality methods: {missing}")

        snapshot = require_result(rpc.call("modality.capabilities", {}), "modality.capabilities")
        available = available_operations(snapshot)
        realtime = realtime_state(snapshot)

        image_bytes = open(args.image, "rb").read()
        image_name = os.path.basename(args.image)
        media_type = "image/png" if image_name.lower().endswith(".png") else "image/jpeg"
        task_id, states, finished = submit_and_wait(
            rpc, image_bytes, image_name, media_type, "text",
            f"dim-modality-evidence-{int(time.time())}", args.timeout,
        )
        reads = read_results(rpc, finished, args.page_bytes) if finished.get("state") == "succeeded" else []

        # Cancel: submit once more and cancel immediately; report the final state.
        cancel_params = {
            "category": "ocr",
            "operation": "recognize",
            "params": {"outputMode": "text"},
            "files": [{
                "name": image_name,
                "media_type": media_type,
                "data_base64": base64.b64encode(image_bytes).decode(),
            }],
            "client_task_id": f"dim-modality-cancel-{int(time.time())}",
        }
        cancel_submit = require_result(rpc.call("modality.task.submit", cancel_params), "modality.task.submit")
        cancel_id = cancel_submit["task"]["task_id"]
        cancel_reply = rpc.call("modality.task.cancel", {"task_id": cancel_id})
        cancel_state = None
        cancel_error = None
        if "error" in cancel_reply:
            cancel_error = cancel_reply["error"]
        else:
            cancel_state = (cancel_reply.get("result", {}).get("task") or {}).get("state")
        cancel_final = require_result(
            rpc.call("modality.task.status", {"task_id": cancel_id}),
            "modality.task.status",
        )["task"].get("state")

        realtime_attempt = None
        if realtime.get("available") and any(op.get("available") for op in realtime.get("operations", [])):
            operation = next(op["operation"] for op in realtime["operations"] if op.get("available"))
            opened = rpc.call("modality.realtime.open", {"operation": f"realtime.{operation}", "params": {}})
            if "error" not in opened:
                connection_id = opened["result"].get("connectionId")
                sent = rpc.call("modality.realtime.send", {
                    "connectionId": connection_id,
                    "frame": {"kind": "text", "data": "hello"},
                })
                closed = rpc.call("modality.realtime.close", {"connectionId": connection_id})
                realtime_attempt = {
                    "operation": operation,
                    "connectionId": connection_id,
                    "sendOk": "error" not in sent,
                    "closeOk": "error" not in closed,
                }
            else:
                realtime_attempt = {"operation": operation, "error": opened["error"]}
        else:
            opened = rpc.call("modality.realtime.open", {"operation": "realtime.voice", "params": {}})
            realtime_attempt = {
                "blocked": True,
                "reason": "capability reports no available realtime operation",
                "openError": opened.get("error"),
            }

        summary = {
            "fixture": args.fixture,
            "server": server,
            "modalityMethodsPresent": not missing,
            "availableOperations": available,
            "task": {
                "taskId": task_id,
                "states": states,
                "terminal": finished.get("state"),
                "error": finished.get("error"),
                "reads": reads,
            },
            "cancel": {
                "taskId": cancel_id,
                "replyState": cancel_state,
                "error": cancel_error,
                "finalState": cancel_final,
            },
            "realtime": {
                "capability": realtime,
                "attempt": realtime_attempt,
            },
        }
        if args.json:
            print(json.dumps(summary, ensure_ascii=False, indent=2))
        else:
            print(f"server: {server}")
            print(f"available operations: {[item['id'] for item in available]}")
            print(f"task: {task_id} states={states} terminal={finished.get('state')} error={finished.get('error')}")
            for read in reads:
                print(f"  result {read['name']}: bytes={read['bytes']} declared={read['declaredBytes']} pages={read['pages']} eof={read['eof']}")
            print(f"cancel: {cancel_id} reply={cancel_state} final={cancel_final} error={cancel_error}")
            print(f"realtime capability available={realtime.get('available')} reasons={realtime.get('reasons')}")
            print(f"realtime attempt: {json.dumps(realtime_attempt, ensure_ascii=False)}")

        ok = finished.get("state") == "succeeded" and bool(reads) and all(read["eof"] for read in reads)
        return 0 if ok else 1
    finally:
        if daemon is not None and daemon.poll() is None:
            daemon.terminate()
            try:
                daemon.wait(timeout=5)
            except subprocess.TimeoutExpired:
                daemon.kill()
        if not args.keep:
            shutil.rmtree(run_dir, ignore_errors=True)
        else:
            print(f"run dir kept: {run_dir}", file=sys.stderr)


if __name__ == "__main__":
    sys.exit(main())
