#!/usr/bin/env python3
"""统计某个 leader 会话树（leader + 全部子会话）的 token 用量与费用。

数据源：本机 DimCode 库（只读）。脚本先把库与 WAL 复制到临时目录，再用
SQLite 只读连接查询，退出时删除临时副本。只读取 token 计数字段，不读取
消息正文、提示词或凭据。

用法：
  scripts/cost-report.py                            # 自动找到当前仓库的主会话
  scripts/cost-report.py --leader <session_id>
  scripts/cost-report.py --rate 6.9 --price-miss 0.15 --price-hit 0.003 --price-out 0.6

依赖：Python 3 标准库。
"""

from __future__ import annotations

import argparse
import datetime as dt
import os
import shutil
import sqlite3
import subprocess
import sys
import tempfile
from pathlib import Path

DEFAULT_DB = "~/.dimcode-alpha/rust/dimcode.sqlite"

# 官方价目（美元 / 百万 token），见 https://api-docs.deepseek.com/quick_start/pricing
# 2026-11-11 抓取：deepseek-flash（DeepSeek-V4.1-Flash-1017）
OFF_PEAK = {"miss": 0.15, "hit": 0.003, "out": 0.6}
PEAK_MULTIPLIER = 2.0  # 官方：off-peak = peak 的一半

# 高峰时段：UTC 周一至周五 01:00-04:00 与 06:00-10:00（不含中国法定节假日）
PEAK_WINDOWS = ((1, 4), (6, 10))

UNIT = 1_000_000  # 单价以每百万 token 计

USAGE_PATH = "$.record.event.data.usage.%s"
TS_PATH = "$.record.event.create_at"


def parse_args(argv):
    p = argparse.ArgumentParser(
        description="统计 DimCode 会话树的 token 用量与费用（只读，临时副本）。",
    )
    p.add_argument("--db", default=DEFAULT_DB, help="DimCode 库路径（默认 %(default)s）")
    p.add_argument("--leader", default=None, help="leader 会话 id；缺省按 --project 自动发现")
    p.add_argument("--project", default=None,
                   help="项目目录，用于自动发现主会话与标定范围（默认当前目录）")
    p.add_argument("--rate", type=float, default=6.9,
                   help="人民币汇率（1 USD = N CNY，默认 %(default)s）")
    p.add_argument("--price-miss", type=float, default=OFF_PEAK["miss"],
                   help="低谷 cache-miss 输入单价，USD/1M（默认 %(default)s）")
    p.add_argument("--price-hit", type=float, default=OFF_PEAK["hit"],
                   help="低谷 cache-hit 输入单价，USD/1M（默认 %(default)s）")
    p.add_argument("--price-out", type=float, default=OFF_PEAK["out"],
                   help="低谷输出单价，USD/1M（默认 %(default)s）")
    p.add_argument("--peak-mult", type=float, default=PEAK_MULTIPLIER,
                   help="高峰单价 = 低谷 × 该倍数（默认 %(default)s）")
    p.add_argument("--flat", action="store_true",
                   help="不分峰谷，全部按低谷单价计费")
    p.add_argument("--holidays", default="",
                   help="按高峰处理以外的节假日日期（UTC，逗号分隔 YYYY-MM-DD）；"
                        "留空表示只按 UTC 星期与时段判定")
    p.add_argument("--rate-note", default="用户指定", help="汇率来源说明")
    p.add_argument("--out", default=None, help="写入文件（默认 stdout）")
    return p.parse_args(argv)


def snapshot(db_path: Path, workdir: Path) -> Path:
    """把库与 -wal/-shm 复制到临时目录，返回副本路径。"""
    dst = workdir / "cost.sqlite"
    shutil.copyfile(db_path, dst)
    for suffix in ("-wal", "-shm"):
        side = db_path.with_name(db_path.name + suffix)
        if side.exists():
            shutil.copyfile(side, workdir / ("cost.sqlite" + suffix))
    return dst


def discover_leader(conn: sqlite3.Connection, project: str | None) -> str | None:
    """按项目目录找最近活跃的主会话（无 leader 的会话）。"""
    if project:
        row = conn.execute(
            """
            SELECT s.session_id
            FROM agent_sessions s
            JOIN session_catalog c ON c.session_id = s.session_id
            WHERE s.leader_session_id IS NULL AND c.cwd = ?
            ORDER BY c.updated_at_ms DESC
            LIMIT 1
            """,
            (project,),
        ).fetchone()
        if row:
            return row[0]
    row = conn.execute(
        """
        SELECT s.session_id
        FROM agent_sessions s
        JOIN session_catalog c ON c.session_id = s.session_id
        WHERE s.leader_session_id IS NULL
        ORDER BY c.updated_at_ms DESC
        LIMIT 1
        """
    ).fetchone()
    return row[0] if row else None


def session_tree(conn: sqlite3.Connection, leader: str) -> list[str]:
    rows = conn.execute(
        """
        WITH RECURSIVE tree(session_id) AS (
            SELECT ?
            UNION
            SELECT s.session_id FROM agent_sessions s JOIN tree t
                   ON s.leader_session_id = t.session_id
        )
        SELECT session_id FROM tree
        """,
        (leader,),
    ).fetchall()
    return [r[0] for r in rows]


def session_meta(conn: sqlite3.Connection, ids: list[str]) -> dict[str, dict]:
    marks = ",".join("?" * len(ids))
    roles = dict(
        conn.execute(
            f"""
            SELECT o.session_id, json_extract(a.profile, '$.name')
            FROM session_owners o JOIN agents a ON a.agent_id = o.agent_id
            WHERE o.session_id IN ({marks})
            """,
            ids,
        ).fetchall()
    )
    names = {sid: (name or "unknown") for sid, name in roles.items()}
    leaders = dict(
        conn.execute(
            f"SELECT session_id, leader_session_id FROM agent_sessions WHERE session_id IN ({marks})",
            ids,
        ).fetchall()
    )
    return {
        sid: {"agent": names.get(sid, "unknown"), "leader": leaders.get(sid)}
        for sid in ids
    }


def usage_rows(conn: sqlite3.Connection, ids: list[str]) -> list[dict]:
    """逐条取用量事件（只取 token 计数与时间戳，不碰正文）。"""
    marks = ",".join("?" * len(ids))
    rows = conn.execute(
        f"""
        SELECT session_id,
               CAST(json_extract(body, '{TS_PATH}') AS INTEGER) AS ts,
               CAST(json_extract(body, '{USAGE_PATH % "promptTokens"}') AS REAL) AS prompt,
               CAST(json_extract(body, '{USAGE_PATH % "completionTokens"}') AS REAL) AS completion,
               CAST(json_extract(body, '{USAGE_PATH % "totalTokens"}') AS REAL) AS total,
               CAST(json_extract(body, '{USAGE_PATH % "cacheReadTokens"}') AS REAL) AS cache_read,
               CAST(json_extract(body, '{USAGE_PATH % "cacheWriteTokens"}') AS REAL) AS cache_write
        FROM timeline_events
        WHERE kind = 'model/usage' AND session_id IN ({marks})
        """,
        ids,
    ).fetchall()
    out = []
    for sid, ts, prompt, completion, total, cache_read, cache_write in rows:
        prompt = prompt or 0.0
        cache_read = cache_read or 0.0
        out.append(
            {
                "session_id": sid,
                "ts": ts or 0,
                "prompt": prompt,
                "completion": completion or 0.0,
                "total": total or 0.0,
                "cache_read": cache_read,
                "cache_miss": max(prompt - cache_read, 0.0),
                "cache_write": cache_write or 0.0,
            }
        )
    return out


def is_peak(ts_ms: int, holidays: set[str]) -> bool:
    if not ts_ms:
        return False
    moment = dt.datetime.fromtimestamp(ts_ms / 1000, tz=dt.timezone.utc)
    if moment.weekday() >= 5:
        return False
    if moment.date().isoformat() in holidays:
        return False
    return any(start <= moment.hour < end for start, end in PEAK_WINDOWS)


def fmt_int(x: float) -> str:
    return f"{int(round(x)):,}"


def fmt_money(x: float) -> str:
    return f"{x:,.4f}"


def fmt_ts(ts_ms: int) -> str:
    if not ts_ms:
        return "-"
    m = dt.datetime.fromtimestamp(ts_ms / 1000, tz=dt.timezone.utc) + dt.timedelta(hours=8)
    return m.strftime("%m-%d %H:%M")


def build_report(args, leader, tree, meta, rows) -> str:
    agg = {sid: {"n": 0, "prompt": 0.0, "completion": 0.0, "total": 0.0,
                 "cache_read": 0.0, "cache_write": 0.0, "t0": 0, "t1": 0,
                 "usd": 0.0}
           for sid in tree}
    for r in rows:
        a = agg[r["session_id"]]
        a["n"] += 1
        for key in ("prompt", "completion", "total", "cache_read", "cache_write"):
            a[key] += r[key]
        a["t0"] = r["ts"] if not a["t0"] else min(a["t0"], r["ts"])
        a["t1"] = max(a["t1"], r["ts"])

    rate = args.rate
    holidays = {d.strip() for d in args.holidays.split(",") if d.strip()}
    miss_c = 0.0
    hit_c = 0.0
    out_c = 0.0
    nocache_c = 0.0
    offpeak_cost = 0.0
    peak_tokens = {"miss": 0.0, "hit": 0.0, "out": 0.0}
    offpeak_tokens = {"miss": 0.0, "hit": 0.0, "out": 0.0}
    for r in rows:
        peak = is_peak(r["ts"], holidays)
        mult = args.peak_mult if (peak and not args.flat) else 1.0
        bucket = peak_tokens if peak else offpeak_tokens
        bucket["miss"] += r["cache_miss"]
        bucket["hit"] += r["cache_read"]
        bucket["out"] += r["completion"]
        miss_c += r["cache_miss"] * args.price_miss * mult / UNIT
        hit_c += r["cache_read"] * args.price_hit * mult / UNIT
        out_c += r["completion"] * args.price_out * mult / UNIT
        nocache_c += r["prompt"] * args.price_miss * mult / UNIT
        offpeak_cost += (r["cache_miss"] * args.price_miss + r["cache_read"] * args.price_hit
                         + r["completion"] * args.price_out) / UNIT
        agg[r["session_id"]]["usd"] += (r["cache_miss"] * args.price_miss + r["cache_read"] * args.price_hit
                                        + r["completion"] * args.price_out) * mult / UNIT
    total_usd = miss_c + hit_c + out_c
    total_cny = total_usd * rate

    tmin = min((a["t0"] for a in agg.values() if a["t0"]), default=0)
    tmax = max((a["t1"] for a in agg.values() if a["t1"]), default=0)

    L = []
    ap = L.append
    ap("### 复跑输出（脚本自动生成，勿手改）")
    ap("")
    ap(f"- 生成时间：{dt.datetime.now().astimezone().strftime('%Y-%m-%d %H:%M %Z')}")
    shown_db = str(args.db)
    home = os.path.expanduser("~")
    if shown_db.startswith(home):
        shown_db = "~" + shown_db[len(home):]
    ap(f"- 数据源：`{shown_db}` 的只读临时副本；表 `timeline_events`，`kind='model/usage'`")
    ap(f"- leader 会话：`{leader}`；会话数：{len(tree)}")
    ap(f"- 数据窗口（北京时间）：{fmt_ts(tmin)} → {fmt_ts(tmax)}")
    ap(f"- 单价（USD/1M tokens）：miss {args.price_miss}、hit {args.price_hit}、"
       f"out {args.price_out}；高峰 = 低谷 × {args.peak_mult}"
       + ("；本次 --flat，全部按低谷计价" if args.flat else ""))
    ap(f"- 汇率：1 USD = {rate} CNY（{args.rate_note}）")
    ap("")

    ap("| 角色 | 领域 agent | 请求数 | prompt | completion | total | cache_read | cache_write | 费用（USD） | 费用（CNY） | 首末（北京时间） |")
    ap("|---|---|---:|---:|---:|---:|---:|---:|---:|---:|---|")
    leader_agent = meta.get(leader, {}).get("agent", "leader")
    ordered = sorted(tree, key=lambda s: (agg[s]["t0"] or 0))
    for sid in ordered:
        a = agg[sid]
        role = "组织" if sid == leader else "子会话"
        name = leader_agent if sid == leader else meta[sid]["agent"]
        ap(f"| {role} | {name} | {a['n']} | {fmt_int(a['prompt'])} | "
           f"{fmt_int(a['completion'])} | {fmt_int(a['total'])} | "
           f"{fmt_int(a['cache_read'])} | {fmt_int(a['cache_write'])} | "
           f"{fmt_money(a['usd'])} | {fmt_money(a['usd'] * rate)} | "
           f"{fmt_ts(a['t0'])} → {fmt_ts(a['t1'])} |")
    tot = {k: sum(a[k] for a in agg.values()) for k in
           ("n", "prompt", "completion", "total", "cache_read", "cache_write")}
    ap(f"| **合计** | {len(tree)} 个会话 | {tot['n']} | {fmt_int(tot['prompt'])} | "
       f"{fmt_int(tot['completion'])} | {fmt_int(tot['total'])} | "
       f"{fmt_int(tot['cache_read'])} | {fmt_int(tot['cache_write'])} | "
       f"{fmt_money(total_usd)} | {fmt_money(total_cny)} | "
       f"{fmt_ts(tmin)} → {fmt_ts(tmax)} |")
    ap("")

    ap("| 计费项 | tokens | 单价（USD/1M） | 金额（USD） | 金额（CNY） |")
    ap("|---|---:|---:|---:|---:|")
    ap(f"| cache-miss 输入 | {fmt_int(tot['prompt'] - tot['cache_read'])} | {args.price_miss} | "
       f"{fmt_money(miss_c)} | {fmt_money(miss_c * rate)} |")
    ap(f"| cache-hit 输入 | {fmt_int(tot['cache_read'])} | {args.price_hit} | "
       f"{fmt_money(hit_c)} | {fmt_money(hit_c * rate)} |")
    ap(f"| 输出 | {fmt_int(tot['completion'])} | {args.price_out} | "
       f"{fmt_money(out_c)} | {fmt_money(out_c * rate)} |")
    ap(f"| **合计** | — | — | **{fmt_money(total_usd)}** | **{fmt_money(total_cny)}** |")
    ap("")

    ap("| 对照口径 | 金额（USD） | 金额（CNY） | 相对实际 |")
    ap("|---|---:|---:|---:|")
    rows_cmp = [
        ("本次实际（按事件时刻分峰谷、计缓存折扣）", total_usd),
        ("若全部按低谷价（缓存仍计价）", offpeak_cost),
        ("若全部按高峰价（缓存仍计价）", offpeak_cost * args.peak_mult),
        ("不吃缓存：全部输入按 miss 价（分峰谷）", nocache_c),
    ]
    for label, usd in rows_cmp:
        delta = "—" if not total_usd else f"{(usd / total_usd - 1) * 100:+.1f}%"
        ap(f"| {label} | {fmt_money(usd)} | {fmt_money(usd * rate)} | {delta} |")
    ap("")
    ap(f"- 命中率：cache-hit {fmt_int(tot['cache_read'])} / prompt {fmt_int(tot['prompt'])} = "
       f"{(tot['cache_read'] / tot['prompt'] * 100 if tot['prompt'] else 0):.1f}%")
    ap(f"- 高峰 token：miss {fmt_int(peak_tokens['miss'])}、hit {fmt_int(peak_tokens['hit'])}、"
       f"out {fmt_int(peak_tokens['out'])}；低谷 token：miss {fmt_int(offpeak_tokens['miss'])}、"
       f"hit {fmt_int(offpeak_tokens['hit'])}、out {fmt_int(offpeak_tokens['out'])}")
    ap(f"- 峰谷价差：高峰单价 = 低谷 × {args.peak_mult}；若全落在高峰，同等用量为 "
       f"{fmt_money(rows_cmp[2][1])} USD（{fmt_money(rows_cmp[2][1] * rate)} CNY），"
       f"是低谷的 {args.peak_mult:.1f} 倍")
    ap("")
    return "\n".join(L)


def main(argv: list[str]) -> int:
    args = parse_args(argv)
    db_path = Path(os.path.expanduser(args.db))
    if not db_path.exists():
        print(f"库不存在：{db_path}", file=sys.stderr)
        return 2
    project = args.project
    if project is None:
        try:
            project = subprocess.run(
                ["git", "rev-parse", "--show-toplevel"],
                capture_output=True, text=True, check=True,
            ).stdout.strip()
        except Exception:
            project = os.getcwd()

    workdir = Path(tempfile.mkdtemp(prefix="dimcode-cost-"))
    try:
        copy = snapshot(db_path, workdir)
        conn = sqlite3.connect(f"file:{copy}?mode=ro", uri=True)
        try:
            leader = args.leader or discover_leader(conn, project)
            if not leader:
                print("找不到主会话，请用 --leader 指定", file=sys.stderr)
                return 2
            exists = conn.execute(
                "SELECT 1 FROM agent_sessions WHERE session_id = ?", (leader,)
            ).fetchone()
            if not exists:
                print(f"库里没有会话 {leader}", file=sys.stderr)
                return 2
            tree = session_tree(conn, leader)
            meta = session_meta(conn, tree)
            rows = usage_rows(conn, tree)
        finally:
            conn.close()
        report = build_report(args, leader, tree, meta, rows)
        if args.out:
            Path(args.out).write_text(report, encoding="utf-8")
        else:
            print(report)
        return 0
    finally:
        shutil.rmtree(workdir, ignore_errors=True)


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
