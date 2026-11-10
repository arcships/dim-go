#!/usr/bin/env python3
"""真实协作模型策略与会话 agent 档案证据。

数据纪律：
- 来源目录只读；夹具在 /tmp，脚本把 fixture/home 复制到独立临时目录后运行。
- 只创建临时会话与 prompt-only agent 档案；不打印凭据、不读取来源配置。
- 证据只保留 revision、状态、agent 标识与错误分类。

用法：
  scripts/verify-collaboration-agent.py --fixture /tmp/dimagent-real-model \
      --server ~/.dimcode-alpha/rust/dimcode-server
"""

import argparse
import json
import os
import shutil
import socket
import subprocess
import sys
import tempfile
import time

REQUIRED_FEATURES = [
    "collaboration.model_policy",
    "collaboration.set_model_policy",
    "session.methods",
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


def ok(reply, method, failures):
    if "error" in reply:
        failures.append({"method": method, "error": reply["error"]})
        return None
    return reply.get("result")


def main():
    parser = argparse.ArgumentParser()
    parser.add_argument("--fixture", default="/tmp/dimagent-real-model")
    parser.add_argument("--server", default="")
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

    run_dir = tempfile.mkdtemp(prefix="dimagent-collab-run.")
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
        missing = [name for name in REQUIRED_FEATURES if name not in rpc.features]
        if missing:
            summary = {"blocked": True, "missingFeatures": missing}
            print(json.dumps(summary, ensure_ascii=False, indent=2) if args.json else summary)
            return 1

        failures = []
        stages = {}

        # Collaboration policy: read, fixed, inherit, agent, readback, stale conflict.
        initial = ok(rpc.call("collaboration.model_policy", {}), "collaboration.model_policy", failures)
        revision = (initial or {}).get("revision")
        stages["policyInitial"] = {
            "revision": revision,
            "mode": ((initial or {}).get("policy") or {}).get("mode"),
        }
        revisions = [revision]
        for index, policy in enumerate([
            {"mode": "fixed", "model": {"providerId": "dimcode-api-oauth", "modelId": "deepseek-v4.1-flash"}},
            {"mode": "inherit"},
            {"mode": "agent"},
        ]):
            written = rpc.call("collaboration.set_model_policy", {
                "expectedRevision": revision,
                "policy": policy,
            })
            if "error" in written:
                failures.append({"method": "collaboration.set_model_policy", "error": written["error"]})
                break
            revision = written["result"]["revision"]
            revisions.append(revision)
            reread = rpc.call("collaboration.model_policy", {})
            stages[f"policyWrite{index + 1}"] = {
                "revision": revision,
                "mode": (reread.get("result", {}).get("policy") or {}).get("mode"),
            }
        stale = rpc.call("collaboration.set_model_policy", {
            "expectedRevision": revisions[0],
            "policy": {"mode": "inherit"},
        })
        after_stale = rpc.call("collaboration.model_policy", {})
        stages["policyRevisions"] = revisions
        stages["policyStaleConflict"] = {
            "error": stale.get("error"),
            "revisionAfter": (after_stale.get("result") or {}).get("revision"),
            "modeAfter": ((after_stale.get("result") or {}).get("policy") or {}).get("mode"),
        }

        # Agent profile: list, create prompt-only profile, read current, adopt, read back.
        created_session = ok(rpc.call("session.create", {"cwd": "/tmp", "tools": []}), "session.create", failures)
        session_id = (created_session or {}).get("sessionId")
        methods = ok(rpc.call("session.methods", {"sessionId": session_id}), "session.methods", failures)
        method_names = [method.get("name") for method in (methods or {}).get("methods", [])]
        stages["agentMethods"] = {
            "get": "session.agent.get" in method_names,
            "update": "session.agent.update" in method_names,
        }
        agents = ok(rpc.call("agent.list", {}), "agent.list", failures) or []
        stages["agentListCount"] = len(agents)
        before = ok(rpc.call("session.agent.get", {
            "sessionId": session_id, "branchId": "main",
        }), "session.agent.get", failures)
        stages["agentBefore"] = before

        created = ok(rpc.call("agent.create", {
            "profile": {
                "name": "Evidence Agent",
                "description": "",
                "systemPrompt": "Be terse.",
                "tools": [],
            },
        }), "agent.create", failures)
        agent_id = (created or {}).get("agentId")
        updated = ok(rpc.call("session.agent.update", {
            "sessionId": session_id,
            "branchId": "main",
            "agentId": agent_id,
        }), "session.agent.update", failures)
        after = ok(rpc.call("session.agent.get", {
            "sessionId": session_id, "branchId": "main",
        }), "session.agent.get", failures)
        same = ok(rpc.call("session.agent.update", {
            "sessionId": session_id,
            "branchId": "main",
            "agentId": agent_id,
        }), "session.agent.update", failures)
        stages["agent"] = {
            "agentId": agent_id,
            "before": before,
            "updateOutcome": (updated or {}).get("outcome"),
            "after": after,
            "sameOutcome": (same or {}).get("outcome"),
        }

        policy_conflict = (
            (stages["policyStaleConflict"]["error"] or {}).get("data", {}).get("appError")
            == "collaboration_model_policy_conflict"
            and stages["policyStaleConflict"]["revisionAfter"] == revisions[-1]
        )
        agent_ok = (
            stages["agentMethods"] == {"get": True, "update": True}
            and (after or {}).get("agentId") == agent_id
            and (updated or {}).get("outcome") in ("adopted", "unchanged")
        )
        stages["ok"] = not failures and policy_conflict and agent_ok
        summary = {
            "server": server,
            "sessionId": session_id,
            "stages": stages,
            "failures": failures,
        }
        if args.json:
            print(json.dumps(summary, ensure_ascii=False, indent=2))
        else:
            print(f"policy initial: {stages['policyInitial']}")
            print(f"policy revisions: {stages['policyRevisions']}")
            print(f"policy writes: {[stages.get(f'policyWrite{i}') for i in (1, 2, 3)]}")
            print(f"policy stale conflict: {stages['policyStaleConflict']}")
            print(f"agent methods: {stages['agentMethods']}")
            print(f"agent list count: {stages['agentListCount']}")
            print(f"agent before: {stages['agentBefore']}")
            print(f"agent: {stages['agent']}")
        return 0 if stages["ok"] else 1
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
