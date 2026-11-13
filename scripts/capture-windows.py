#!/usr/bin/env python3
"""Capture the real window of cmd/dimagent-ui for each state.

Run from the repository root. For each state: launch the binary, wait for the
first frame, find the window by CGWindowList, capture it (and a title-bar
closeup), then terminate the process.
"""
import re
import subprocess
import sys
import time

BIN = "/tmp/dimagent-ui"
SHOTS = "docs/domains/screens"
STATES = ["default", "running", "waiting", "failed", "empty"]
# 三档宽度对照（U1-ia §0：compact <720 / regular 720–959 / wide ≥960）。
WIDTHS = [1440, 900, 720]
# 三类新信息的走查态（分支切换 / 目标设置中 / 后台任务运行中）。
VARIANTS = ["branch", "branch-end", "goal", "tasks", "settings", "settings-rules", "ext", "ext-failed"]


def windows():
    out = subprocess.run(["swift", "/tmp/winid.swift"], capture_output=True, text=True, check=True).stdout
    rows = []
    for line in out.splitlines():
        parts = line.split("\t")
        if len(parts) >= 4:
            rows.append(parts)
    return rows


def find_window(pid):
    for row in windows():
        if len(row) < 7:
            continue
        num, owner, name = row[0], row[1], row[2]
        # 只认界面骨架自己的窗口：owner 是进程名，窗口名带「界面骨架」。
        if "dimagent-ui" not in owner.lower() or "界面骨架" not in name:
            continue
        x, y, w, h = (int(v) for v in row[3:7])
        return int(num), x, y, w, h
    return None


def capture(state, width=1440, height=900, suffix=None, variant=""):
    args = [BIN, "-state", state, "-width", str(width), "-height", str(height)]
    if variant:
        args += ["-variant", variant]
    proc = subprocess.Popen(args,
                            stdout=open(f"/tmp/dimagent-ui-{state}.log", "w"),
                            stderr=subprocess.STDOUT)
    try:
        win = None
        for _ in range(20):
            time.sleep(0.3)
            win = find_window(proc.pid)
            if win:
                break
        if not win:
            print(f"{state}: window not found (pid {proc.pid}, poll {proc.poll()})", file=sys.stderr)
            return False
        num, x, y, w, h = win
        time.sleep(0.8)  # let the first frame settle
        path = f"{SHOTS}/window-{state}{suffix or ''}.png"
        # 窗口刚出现时 screencapture 偶尔取不到它的后备缓冲：重试三次，仍失败就
        # 报出状态并继续，不静默当成功。
        last = None
        for attempt in range(3):
            r = subprocess.run(["screencapture", "-x", "-o", "-l", str(num), path],
                               capture_output=True)
            if r.returncode == 0:
                last = None
                break
            last = r
            time.sleep(0.8)
        if last is not None:
            print(f"{state}: capture failed for window {num}: {last.stderr.decode()[:200]}",
                  file=sys.stderr)
            return False
        print(f"{state}: window {num} at {x},{y} {w}x{h} -> {path}")
        if state == "default" and suffix is None:
            # 标题行特写直接从这一帧裁，不再用 -R 取屏幕坐标：窗口可能落在负坐标上
            # （副屏/多显示器），-R 会失败。
            from PIL import Image

            closeup = f"{SHOTS}/window-titlebar-closeup.png"
            img = Image.open(path)
            scale = img.width / w
            img.crop((0, 0, img.width, int(120 * scale))).save(closeup)
            print(f"{state}: titlebar closeup -> {closeup}（{img.width}×{int(120 * scale)}）")
        return True
    finally:
        proc.terminate()
        try:
            proc.wait(timeout=5)
        except subprocess.TimeoutExpired:
            proc.kill()


def main():
    ok = True
    for state in STATES:
        if not capture(state):
            ok = False
    for width in WIDTHS:
        suffix = None if width == 1440 else f"-{width}"
        if not capture("default", width=width, suffix=suffix):
            ok = False
    for variant in VARIANTS:
        state = "running" if variant == "tasks" else "default"
        # 设置两态：默认（外观分节）与命令规则的清空确认态。
        for width in WIDTHS:
            suffix = f"-variant-{variant}" if width == 1440 else f"-variant-{variant}-{width}"
            if not capture(state, width=width, suffix=suffix, variant=variant):
                ok = False
    sys.exit(0 if ok else 1)


if __name__ == "__main__":
    main()
