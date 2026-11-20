#!/usr/bin/env python3
"""负向取证的写回：把「需负向取证」的条目改成用例证据。

输入是下面 MAP（用例名 → 受它证明的需求 ID）与 FAIL（失败的用例 → 现象）。
只改这些行的状态列：

  通过的用例 → 已实现（证据 = <测试名>）
  失败的用例 → 未实现（负向用例失败：<现象>）
  没被任何用例覆盖的行原样保留（仍是「需负向取证」），脚本不替它们下结论。

写回前核对那一行当前状态确实是「需负向取证」：已经是别的结论（例如别的
对账轮次给出了新证据）就不动它，并在输出里列出来。

用法：
    python3 scripts/negative-evidence.py --dry-run
    python3 scripts/negative-evidence.py
"""

from __future__ import annotations

import argparse
import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
REQ_DIR = ROOT / 'docs' / 'domains' / 'requirements'
REQ_FILES = [
    REQ_DIR / 'requirements-settings.md',
    REQ_DIR / 'requirements-main.md',
    REQ_DIR / 'requirements-global.md',
    REQ_DIR / 'requirements-panels.md',
]

NEG_LABEL = '需负向取证'
ROW_RE = re.compile(r'^\|\s*(R-[A-Za-z0-9]+-\d+)\s*\|')

# 用例 → 它证明的条目。一个用例对应一类可观察的负向事实；条目只在这条用例
# 真的证明了它那句「没有发生」时才登记进来。
MAP: dict[str, list[str]] = {
    'TestNegativeApprovalStateHidesComposer': [
        'R-IN-003', 'R-IN-156', 'R-XR-045',
    ],
    'TestNegativeApprovalWithoutLongTermHidesButton': [
        'R-AP-214',
    ],
    'TestNegativeRunningStateHidesSendButton': [
        'R-IN-146', 'R-IN-155',
    ],
    'TestNegativeRepeatedSubmitSendsOnce': [
        'R-IN-136', 'R-IN-140',
    ],
    'TestNegativeSessionSwitchLeavesNoStaleRows': [
        'R-XR-005', 'R-TB-163',
    ],
    'TestNegativeSidebarTextClicksAreInert': [
        'R-LB-123', 'R-LB-160', 'R-LB-182', 'R-LB-198', 'R-LB-322',
    ],
    'TestNegativeSettingsSectionsDoNotShowEachOther': [
        'R-ACC-005', 'R-APP-003', 'R-GEN-003',
    ],
    'TestNegativeScreensShowNoKeyNames': [
        'R-ACC-009', 'R-ACC-044', 'R-ACC-129', 'R-ACC-262', 'R-ACC-264', 'R-ACC-391',
        'R-USG-007', 'R-APP-091', 'R-APP-102', 'R-APP-582', 'R-MOD-022', 'R-RUN-046',
        'R-LB-163',
    ],
    'TestNegativeAccountLeavesNoIdentityWithoutFacts': [
        'R-ACC-067', 'R-ACC-070', 'R-ACC-076', 'R-ACC-085', 'R-ACC-249',
    ],
    'TestNegativeUsageShowsNoFabricatedNumbers': [
        'R-USG-042', 'R-USG-067', 'R-USG-082',
    ],
    'TestNegativeSessionSearchForgetsQuery': [
        'R-GS-045', 'R-GS-055',
    ],
    'TestNegativeSearchWithoutHitsHidesJumpButtons': [
        'R-TB-091', 'R-TB-094',
    ],
    'TestNegativeNoPinnedGroupWithoutPins': [
        'R-LB-112',
    ],
    'TestNegativeStatusBarWithoutUsageShowsNoPercent': [
        'R-SB-158', 'R-XR-090',
    ],
    'TestNegativeTasksWithoutSelectionShowNoDetail': [
        'R-TK-134',
    ],
    'TestNegativeReviewWithoutChangesShowsNoActions': [
        'R-RD-255', 'R-RD-739', 'R-RD-749',
    ],
    'TestNegativeFilesWithoutEntriesShowNoRows': [
        'R-RF-084', 'R-RF-090', 'R-RF-123', 'R-RF-205',
    ],
    'TestNegativeEmptyThreadShowsNoHistory': [
        'R-SB-002',
    ],
}

# 跑起来失败的用例 → 屏上实际发生的现象（写进状态列，交回实现方）。
FAIL: dict[str, str] = {
    'TestNegativeRepeatedSubmitSendsOnce':
        '请求未回来时连点三次「发送」发出 3 条提交（缺在途守卫，可重复提交）',
    'TestNegativeApprovalWithoutLongTermHidesButton':
        '审批快照没有长期规则时仍渲染「始终允许此命令」按钮（呈禁用态，期望不出现）',
    'TestNegativeSearchWithoutHitsHidesJumpButtons':
        '零命中时仍渲染「上一个 / 下一个搜索结果」按钮（呈禁用态，期望不出现）',
}


OURS = ('已实现（证据 = TestNegative', '未实现（负向用例失败')


def is_ours(status: str) -> bool:
    """这一行的状态是本脚本写的（或还没写过）时才允许覆盖。"""
    return status.startswith(NEG_LABEL) or status.startswith(OURS)


def row_status(line: str) -> str:
    cells = line.rstrip('\n').rstrip()
    if not cells.endswith('|'):
        return ''
    body = cells[:-1]
    return body[body.rfind('|') + 1:].strip()


def replace_status(line: str, status: str) -> str:
    """把一行的状态列换成 status；列两侧的空格统一成一个。"""
    stripped = line.rstrip('\n')
    body = stripped[:-1] if stripped.endswith('|') else stripped
    head = body[:body.rfind('|') + 1].rstrip()
    return head + ' ' + status + ' |\n'


def main() -> int:
    ap = argparse.ArgumentParser()
    ap.add_argument('--dry-run', action='store_true')
    args = ap.parse_args()

    passed = {test for test in MAP if test not in FAIL}
    target = {}
    for test, ids in MAP.items():
        for rid in ids:
            target[rid] = test

    seen: dict[str, str] = {}
    changed = kept = 0
    per_file = {}
    for path in REQ_FILES:
        lines = path.read_text(encoding='utf-8').splitlines(keepends=True)
        out = []
        n = 0
        for line in lines:
            match = ROW_RE.match(line)
            if not match or match.group(1) not in target:
                out.append(line)
                continue
            rid = match.group(1)
            test = target[rid]
            current = row_status(line)
            seen[rid] = current
            if not is_ours(current):
                out.append(line)
                kept += 1
                continue
            if test in FAIL:
                status = f'未实现（负向用例失败：{FAIL[test]}）'
            elif test in passed:
                status = f'已实现（证据 = {test}）'
            else:
                out.append(line)
                continue
            out.append(replace_status(line, status))
            changed += 1
            n += 1
        per_file[path.name] = n
        if not args.dry_run and n:
            path.write_text(''.join(out), encoding='utf-8')

    missing = sorted(set(target) - set(seen))
    stale = sorted(rid for rid, status in seen.items() if not is_ours(status))

    print(f'用例 {len(MAP)} 条（通过 {len(passed)}，失败 {len(FAIL)}），登记条目 {len(target)} 条')
    for name, n in per_file.items():
        print(f'  {name}: 改 {n} 行')
    print(f'状态改成用例证据 {changed} 行；当前状态已不是「需负向取证」而跳过 {kept} 行')
    if missing:
        print('  ! 表里找不到的条目：' + '、'.join(missing))
    if stale:
        print('  ! 状态已被别的结论覆盖（未改动）：' + '、'.join(stale))
    if args.dry_run:
        print('（dry-run：没有写文件）')
    return 0


if __name__ == '__main__':
    sys.exit(main())
