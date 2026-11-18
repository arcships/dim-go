"""设置行为需求自查：元素覆盖、每元素条数、维度覆盖、重复条目标识。

用法：python3 scripts/check-req.py [文档路径]
默认检查 docs/domains/requirements/requirements-settings.md，对照
docs/domains/requirements/inventory-settings-rt.md 的元素清单（19 节 + 壳层）。
"""

import collections
import re
import sys

DOC = sys.argv[1] if len(sys.argv) > 1 else 'docs/domains/requirements/requirements-settings.md'
INV = 'docs/domains/requirements/inventory-settings-rt.md'

ROW = re.compile(r'^\|\s*(R-[A-Z]+-\d+)\s*\|\s*(S-[A-Za-z]+-\d+)\s*\|\s*([^|]+?)\s*\|', re.M)
ELEM = re.compile(r'^\|\s*(S-[A-Za-z]+-\d+)\s*\|', re.M)

doc = open(DOC).read()
inv = open(INV).read()

rows = ROW.findall(doc)
order = list(dict.fromkeys(m.group(1) for m in ELEM.finditer(inv)))

counts = collections.Counter(r[1] for r in rows)
dims = collections.defaultdict(set)
for _, el, dim in rows:
    dims[el].add(dim.strip())

dups = [k for k, v in collections.Counter(r[0] for r in rows).items() if v > 1]
print(f'条目总数: {len(rows)}  重复条目 ID: {len(dups)}')
if dups:
    print('  ' + ', '.join(dups[:20]))

by_section = collections.OrderedDict()
for el in order:
    by_section.setdefault(el.split('-')[1], []).append(el)

total_rows = 0
for sec, elems in by_section.items():
    got = [e for e in elems if e in counts]
    n = sum(counts[e] for e in got)
    total_rows += n
    if not got:
        print(f'{sec:16s} 元素 0/{len(elems)}')
        continue
    print(f'{sec:16s} 元素 {len(got)}/{len(elems)}  条目 {n:5d}  '
          f'平均 {n / len(got):.2f}  最少 {min(counts[e] for e in got)}  '
          f'最多 {max(counts[e] for e in got)}  最少维度 {min(len(dims[e]) for e in got)}')

print(f'合计: 元素 {len(order)}  条目 {total_rows}  平均 {total_rows / len(order):.2f}')

missing = [e for e in order if e not in counts]
print(f'未出条目的元素: {len(missing)}' + ('  ' + ' '.join(missing) if missing else ''))
under = [(e, counts[e]) for e in order if e in counts and counts[e] < 8]
under += [(e, len(dims[e])) for e in order if e in counts and len(dims[e]) < 5]
print(f'低于下限（<8 条或 <5 维度）: {len(under)}' + ('  ' + str(under[:10]) if under else ''))
