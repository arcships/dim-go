#!/usr/bin/env python3
"""真实 provider/模型目录管理证据：列表、新建、探测、更新、启用、模型开关、
编辑、重置、偏好与版本冲突。

数据纪律：
- 来源目录只读；夹具由调用方指定，脚本把 fixture/home 复制到独立临时目录后运行。
- 只使用匿名认证与本地 mock /models 端点；不读取、不打印真实凭据。
- 证据只保留 provider/模型标识、版本、状态、计数与错误分类，不打印凭据。

用法：
  scripts/verify-provider-admin.py --fixture <fixture-dir> --server <installed-server>
"""

import argparse
import json
import os
import shutil
import socket
import subprocess
import sys
import tempfile
import threading
import time
import uuid
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

REQUIRED_METHODS = [
    "provider.list",
    "provider.management",
    "provider.create",
    "provider.update",
    "provider.probe",
    "provider.models.set_enabled",
    "provider.models.edit",
    "provider.models.reset",
    "provider.preferences.update",
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
        self.file.write((json.dumps({
            "jsonrpc": "2.0",
            "id": request_id,
            "method": method,
            "params": params,
        }) + "\n").encode())
        while True:
            line = self.file.readline()
            if not line:
                raise RuntimeError("daemon closed the connection")
            message = json.loads(line)
            if message.get("id") == request_id:
                return message


class MockModels(BaseHTTPRequestHandler):
    def do_GET(self):
        if self.path.endswith("/models"):
            body = json.dumps({"data": [
                {"id": "probe-1", "context_window": 32000, "max_output_tokens": 47},
                {"id": "probe-2", "context_window": 16000, "max_output_tokens": 32},
            ]}).encode()
            self.send_response(200)
            self.send_header("content-type", "application/json")
            self.send_header("content-length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
            return
        self.send_response(404)
        self.end_headers()

    def log_message(self, *args):
        pass


def ok(reply, method, failures):
    if "error" in reply:
        failures.append({"method": method, "error": reply["error"]})
        return None
    return reply.get("result")


def find_provider(rpc, provider_id):
    reply = rpc.call("provider.management", {"providerId": provider_id})
    if "error" in reply:
        return None
    for provider in reply.get("result", {}).get("providers", []):
        if provider.get("providerId") == provider_id:
            return provider
    return None


def model_entries(detail):
    if not detail:
        return []
    entries = []
    for entry in detail.get("models") or []:
        model = entry.get("model") or {}
        capabilities = model.get("capabilities") or {}
        entries.append({
            "modelId": model.get("modelId"),
            "displayName": model.get("displayName"),
            "status": model.get("status"),
            "enabled": entry.get("enabled"),
            "contextWindow": capabilities.get("contextWindow"),
            "maxOutputTokens": capabilities.get("maxOutputTokens"),
        })
    return entries


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--fixture", required=True)
    parser.add_argument("--server", required=True)
    parser.add_argument("--keep", action="store_true")
    parser.add_argument("--json", action="store_true")
    args = parser.parse_args()

    if not os.path.isdir(os.path.join(args.fixture, "home")):
        print(f"fixture home missing: {args.fixture}/home", file=sys.stderr)
        return 2
    server = args.server
    if not os.access(server, os.X_OK):
        print(f"server is not executable: {server}", file=sys.stderr)
        return 2

    run_dir = tempfile.mkdtemp()
    daemon = None
    httpd = None
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

        httpd = ThreadingHTTPServer(("127.0.0.1", 0), MockModels)
        threading.Thread(target=httpd.serve_forever, daemon=True).start()
        base_url = f"http://127.0.0.1:{httpd.server_address[1]}/v1"

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
        missing = [name for name in REQUIRED_METHODS if name not in rpc.features]
        if missing:
            summary = {"blocked": True, "missingMethods": missing}
            print(json.dumps(summary, ensure_ascii=False, indent=2) if args.json else summary)
            return 1

        failures = []
        stages = {}
        provider_id = str(uuid.uuid4())
        probe_id = str(uuid.uuid4())

        listing = ok(rpc.call("provider.list", {}), "provider.list", failures)
        stages["listCountBefore"] = len((listing or {}).get("providers") or [])

        models = [
            {
                "modelId": "evidence-1",
                "displayName": "Evidence 1",
                "status": "active",
                "capabilities": {
                    "modalities": ["text"],
                    "toolCalling": True,
                    "streaming": True,
                    "structuredOutput": True,
                    "vision": False,
                    "audio": False,
                    "reasoning": False,
                    "contextWindow": 1000,
                    "maxOutputTokens": 100,
                },
            },
            {
                "modelId": "evidence-2",
                "displayName": "Evidence 2",
                "status": "active",
                "capabilities": {
                    "modalities": ["text"],
                    "toolCalling": True,
                    "streaming": True,
                    "structuredOutput": True,
                    "vision": False,
                    "audio": False,
                    "reasoning": False,
                    "contextWindow": 1000,
                    "maxOutputTokens": 100,
                },
            },
        ]
        created = ok(rpc.call("provider.create", {
            "providerId": provider_id,
            "displayName": "Dim Evidence",
            "connection": {
                "driverKind": "openai-compatible",
                "auth": {"type": "anonymous"},
                "baseUrl": base_url,
            },
            "models": models,
            "enabled": True,
        }), "provider.create", failures)
        detail = find_provider(rpc, provider_id)
        stages["created"] = {
            "providerId": provider_id,
            "version": (detail or {}).get("version"),
            "modelCount": len((detail or {}).get("models") or []),
            "enabled": (detail or {}).get("enabled"),
        }
        version = (detail or {}).get("version")

        probe = ok(rpc.call("provider.probe", {
            "providerId": probe_id,
            "connection": {
                "driverKind": "openai-compatible",
                "auth": {"type": "anonymous"},
                "baseUrl": base_url,
            },
        }), "provider.probe", failures)
        probe_models = (probe or {}).get("models") or []
        stages["probe"] = {
            "modelCount": len(probe_models),
            "ids": [model.get("modelId") for model in probe_models],
            "contextWindows": [
                ((model.get("capabilities") or {}).get("contextWindow"))
                for model in probe_models
            ],
            "createdProvider": find_provider(rpc, probe_id) is not None,
        }

        updated = ok(rpc.call("provider.update", {
            "providerId": provider_id,
            "expectedVersion": version,
            "patch": {"displayName": "Dim Evidence Updated"},
        }), "provider.update", failures)
        detail = find_provider(rpc, provider_id)
        version = (detail or {}).get("version")
        stages["updated"] = {"version": version, "displayName": (detail or {}).get("displayName")}

        ok(rpc.call("provider.update", {
            "providerId": provider_id, "expectedVersion": version, "patch": {"enabled": False},
        }), "provider.update(disable)", failures)
        detail = find_provider(rpc, provider_id)
        version = (detail or {}).get("version")
        stages["disabled"] = {"version": version, "enabled": (detail or {}).get("enabled")}
        ok(rpc.call("provider.update", {
            "providerId": provider_id, "expectedVersion": version, "patch": {"enabled": True},
        }), "provider.update(enable)", failures)
        detail = find_provider(rpc, provider_id)
        version = (detail or {}).get("version")
        stages["enabled"] = {"version": version, "enabled": (detail or {}).get("enabled")}

        ok(rpc.call("provider.models.set_enabled", {
            "providerId": provider_id, "expectedVersion": version, "modelIds": ["evidence-1"],
        }), "provider.models.set_enabled", failures)
        detail = find_provider(rpc, provider_id)
        version = (detail or {}).get("version")
        stages["modelsEnabled"] = {"version": version, "models": model_entries(detail)}

        ok(rpc.call("provider.models.edit", {
            "providerId": provider_id, "expectedVersion": version, "modelId": "evidence-1",
            "edit": {
                "displayName": "Evidence One Edited",
                "contextWindow": 2000,
                "maxOutputTokens": 200,
                "toolCalling": True,
                "vision": False,
            },
        }), "provider.models.edit", failures)
        detail = find_provider(rpc, provider_id)
        version = (detail or {}).get("version")
        stages["modelEdited"] = {"version": version, "models": model_entries(detail)}

        ok(rpc.call("provider.models.reset", {
            "providerId": provider_id, "expectedVersion": version, "modelId": "evidence-1",
        }), "provider.models.reset", failures)
        detail = find_provider(rpc, provider_id)
        version = (detail or {}).get("version")
        stages["modelReset"] = {"version": version, "models": model_entries(detail)}

        preference = rpc.call("provider.preferences.update", {
            "providerId": provider_id, "expectedVersion": version, "modelId": "evidence-1",
            "patch": {"reasoningEffort": {"action": "set", "value": "high"}},
        })
        if "error" in preference:
            preference = rpc.call("provider.preferences.update", {
                "providerId": provider_id, "expectedVersion": version, "modelId": "evidence-1",
                "patch": {"serviceTier": {"action": "clear"}},
            })
        if "error" in preference:
            failures.append({"method": "provider.preferences.update", "error": preference["error"]})
            stages["preferences"] = {"written": False, "error": preference["error"]}
        else:
            detail = find_provider(rpc, provider_id)
            version = (detail or {}).get("version")
            stages["preferences"] = {"written": True, "version": version, "preferences": (detail or {}).get("preferences")}

        before_conflict = find_provider(rpc, provider_id)
        stale = rpc.call("provider.update", {
            "providerId": provider_id, "expectedVersion": 1,
            "patch": {"displayName": "SHOULD NOT APPLY"},
        })
        after_conflict = find_provider(rpc, provider_id)
        stages["staleConflict"] = {
            "error": stale.get("error"),
            "nameBefore": (before_conflict or {}).get("displayName"),
            "nameAfter": (after_conflict or {}).get("displayName"),
            "versionBefore": (before_conflict or {}).get("version"),
            "versionAfter": (after_conflict or {}).get("version"),
        }

        listing = ok(rpc.call("provider.list", {}), "provider.list", failures)
        stages["listCountAfter"] = len((listing or {}).get("providers") or [])
        stages["customInList"] = any(
            provider.get("providerId") == provider_id
            for provider in (listing or {}).get("providers") or []
        )

        conflict_ok = (
            stages["staleConflict"]["error"] is not None
            and (stages["staleConflict"]["error"].get("data") or {}).get("appError") == "provider_conflict"
            and stages["staleConflict"]["nameAfter"] == stages["staleConflict"]["nameBefore"]
            and stages["staleConflict"]["versionAfter"] == stages["staleConflict"]["versionBefore"]
        )
        stages["ok"] = not failures and conflict_ok and stages["customInList"] and not stages["probe"]["createdProvider"]
        summary = {
            "server": server,
            "listCountBefore": stages["listCountBefore"],
            "listCountAfter": stages["listCountAfter"],
            "stages": stages,
            "failures": failures,
        }
        if args.json:
            print(json.dumps(summary, ensure_ascii=False, indent=2))
        else:
            print(f"provider list: {stages['listCountBefore']} -> {stages['listCountAfter']}")
            print(f"created: {stages['created']}")
            print(f"probe: {stages['probe']}")
            print(f"update: {stages['updated']}")
            print(f"enable: {stages['disabled']} -> {stages['enabled']}")
            print(f"models: {stages['modelsEnabled']}")
            print(f"edit: {stages['modelEdited']}")
            print(f"reset: {stages['modelReset']}")
            print(f"preferences: {stages['preferences']}")
            print(f"stale conflict: {stages['staleConflict']}")
            print(f"custom in list: {stages['customInList']}")
        return 0 if stages["ok"] else 1
    finally:
        if httpd is not None:
            httpd.shutdown()
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
