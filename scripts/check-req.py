"""设置行为需求自查：元素覆盖、每元素条数、维度覆盖、重复条目标识。

用法：python3 scripts/check-req.py [文档路径]
默认检查 docs/domains/requirements/requirements-settings.md，对照
docs/domains/requirements/inventory-settings-rt.md 的元素清单。
"""

import collections
import re
import sys

DOC = sys.argv[1] if len(sys.argv) > 1 else 'docs/domains/requirements/requirements-settings.md'
INV = 'docs/domains/requirements/inventory-settings-rt.md'
SECTIONS = ('account', 'usage', 'personalization', 'appearance')

ROW = re.compile(r'^\|\s*(R-[A-Z]+-\d+)\s*\|\s*(S-[a-z]+-\d+)\s*\|\s*([^|]+?)\s*\|', re.M)

doc = open(DOC).read()
inv = open(INV).read()

rows = ROW.findall(doc)
elems = collections.OrderedDict(
    (m.group(1), None) for m in re.finditer(r'^\|\s*(S-[a-z]+-[0-9]+)\s*\|', inv, re.M)
)

counts = collections.Counter(r[1] for r in rows)
dims = collections.defaultdict(set)
for _, el, dim in rows:
    dims[el].add(dim.strip())

dups = [k for k, v in collections.Counter(r[0] for r in rows).items() if v > 1]
print(f'条目总数: {len(rows)}  重复条目 ID: {len(dups)}')
if dups:
    print('  ' + ', '.join(dups[:20]))

total_elems = total_rows = 0
for sec in SECTIONS:
    names = [e for e in elems if e.split('-')[1] == sec]
    got = [e for e in names if e in counts]
    sec_rows = sum(counts[e] for e in got)
    total_elems += len(names)
    total_rows += sec_rows
    lo = min((counts[e] for e in got), default=0)
    hi = max((counts[e] for e in got), default=0)
    lo_dims = min((len(dims[e]) for e in got), default=0)
    print(f'{sec:16s} 元素 {len(got)}/{len(names)}  条目 {sec_rows:5d}  '
          f'平均 {sec_rows / max(len(got), 1):.2f}  最少 {lo}  最多 {hi}  最少维度 {lo_dims}')

missing = [e for e in elems if e.split('-')[1] in SECTIONS and e not in counts]
print(f'四节合计: 元素 {total_elems}  条目 {total_rows}  平均 {total_rows / total_elems:.2f}')
print(f'未出条目的元素: {len(missing)}' + ('  ' + ' '.join(missing) if missing else ''))
under = [(e, counts[e]) for e in elems if e.split('-')[1] in SECTIONS and e in counts and counts[e] < 8]
under += [(e, len(dims[e])) for e in elems if e.split('-')[1] in SECTIONS and e in counts and len(dims[e]) < 5]
print(f'低于下限（<8 条或 <5 维度）: {len(under)}' + ('  ' + str(under[:10]) if under else ''))
