#!/usr/bin/env python3
"""把 docs/domains/requirements/memory-adjudication.md 的结论并回两份需求文档的状态列。

只改目标行的第 8 列（状态），其余列与行序不动。逐条先按 R-ID 认领，认领不到再按
「元素 ID + 场景」认领（两份文档里该组合唯一时才用），最后打印仍未认领的清单。

    python3 scripts/merge-memory-adjudication.py            # 正式写
    python3 scripts/merge-memory-adjudication.py --dry-run  # 只报数不写
"""

import sys

DOCS = [
    'docs/domains/requirements/requirements-main.md',
    'docs/domains/requirements/requirements-panels.md',
]
TABLE = 'docs/domains/requirements/memory-adjudication.md'
DRY = '--dry-run' in sys.argv


def split_cells(line):
    return [x.strip() for x in line.strip().strip('|').split('|')]


# 表：R-ID -> (元素 ID, 场景, 结论整句)
rows = {}
for line in open(TABLE, encoding='utf-8'):
    if line.startswith('| R-'):
        c = [x.strip() for x in line.strip().strip('|').split(' | ')]
        if len(c) == 7:
            rows[c[0]] = (c[1], c[2], c[5])
if not rows:
    sys.exit('核定表为空：先确认 ' + TABLE + ' 在')

docs = {path: open(path, encoding='utf-8').read().split('\n') for path in DOCS}

# 每份文档的 R-ID 与（元素, 场景）索引
ids = {}
keys = {}
for path, lines in docs.items():
    ids[path] = set()
    keys[path] = {}
    for line in lines:
        if not line.startswith('| R-'):
            continue
        c = split_cells(line)
        if len(c) < 8:
            continue
        ids[path].add(c[0])
        if c[2] == '记忆':  # 兜底只在同维度的行里找，避免撞到同元素同场景的其它维度行
            keys[path].setdefault((c[1], c[3]), []).append(c[0])

# 认领：每条表行落到哪份文档（R-ID 优先；否则元素+场景唯一命中）
owner = {}
for rid, (el, scene, _) in rows.items():
    for path in DOCS:
        if rid in ids[path]:
            owner[rid] = path
            break
    if rid in owner:
        continue
    hits = [path for path in DOCS if len(keys[path].get((el, scene), [])) == 1]
    if len(hits) == 1:
        owner[rid] = hits[0]

orphan = sorted(k for k in rows if k not in owner)
written = 0
for path, lines in docs.items():
    todo = {rid: rows[rid] for rid, p in owner.items() if p == path}
    done = set()
    for i, line in enumerate(lines):
        if not line.startswith('| R-'):
            continue
        c = split_cells(line)
        if len(c) < 8:
            continue
        hit = c[0] if c[0] in todo and c[0] not in done else None
        if hit is None:
            for rid, (el, scene, _) in todo.items():
                if rid not in done and c[2] == '记忆' and c[1] == el and c[3] == scene:
                    hit = rid
                    break
        if hit is None:
            continue
        c[7] = ' ' + todo[hit][2] + ' '
        lines[i] = '| ' + ' | '.join(c) + ' |'
        done.add(hit)
        written += 1
    if not DRY:
        open(path, 'w', encoding='utf-8').write('\n'.join(lines))
    left = sorted(set(todo) - done)
    print(f'{path}: 认领 {len(todo)} 条，写入 {len(done)} 条' + (f'，落空 {left}' if left else ''))

print('合计写入', written, '条（表内', len(rows), '条）；两份文档都认领不到', len(orphan), '条',
      orphan or '')
sys.exit(0 if written == len(rows) else 1)
