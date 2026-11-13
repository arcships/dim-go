#!/usr/bin/env python3
"""把 U1 / U3 的机器可读清单渲染进各自的领域文档（单一来源 = JSON）。

    python3 scripts/render-lists.py

只重写文档里 `<!-- BEGIN:items -->` 与 `<!-- END:items -->` 之间的表；
清单本体是 `docs/domains/U1-content.json` 与 `docs/domains/U3-actions.json`，
检查器（`internal/ui/lint_test.go`）直接读同一份 JSON，不抄字。
"""
import io
import json

MARK_BEGIN = "<!-- BEGIN:items -->"
MARK_END = "<!-- END:items -->"


def rows_u1(data):
    out = ["| # | 字符串或模式 | 来源信息项 | 判定 | 理由 |", "|---|---|---|---|---|"]
    for it in data["items"]:
        pattern = it["value"] if it["match"] == "exact" else it["value"] + "…"
        out.append(
            f"| {it['id']} | `{pattern}` | {it['item']} | "
            f"{'内容' if it['kind'] == 'content' else '界面语'} | {it['why']} |"
        )
    return "\n".join(out)


def rows_u3(data):
    out = ["| # | 动作 | 后果 | 影响面 | 形式 | 需要确认 | 依据 |",
           "|---|---|---|---|---|---|---|"]
    for it in data["items"]:
        out.append(
            f"| {it['id']} | {it['action']} | {it['consequence']} | {it['scope']} | "
            f"`{it['form']}` | {'是' if it['confirm'] else '否'} | {it['basis']} |"
        )
    return "\n".join(out)


def rewrite(path, data, render):
    s = io.open(path, encoding="utf-8").read()
    start = s.index(MARK_BEGIN) + len(MARK_BEGIN)
    end = s.index(MARK_END)
    s = s[:start] + "\n" + render(data) + "\n" + s[end:]
    io.open(path, "w", encoding="utf-8").write(s)


def main():
    u1 = json.load(io.open("docs/domains/U1-content.json", encoding="utf-8"))
    u3 = json.load(io.open("docs/domains/U3-actions.json", encoding="utf-8"))
    rewrite("docs/domains/U1-content.md", u1, rows_u1)
    rewrite("docs/domains/U3-actions.md", u3, rows_u3)
    print(f"wrote U1-content.md（{len(u1['items'])} 条）与 U3-actions.md（{len(u3['items'])} 条）")


if __name__ == "__main__":
    main()
