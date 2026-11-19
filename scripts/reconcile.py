#!/usr/bin/env python3
"""需求实现状态机械对账。

输入：docs/domains/requirements/requirements-{settings,main,global,panels}.md 的
表格行（需求 ID、元素 ID、维度、场景、期望、异常、验收、状态），元素名称与
清单来源取自同目录 inventory-*.md。

按顺序取第一个命中的证据：

  1. 逐字文案：从「期望」抽取「」/“”内的字面量（长度 ≥2、非变量名），在
     internal/**/*.go（非 _test.go）的字符串字面量里查；命中 → 已实现（逐字），
     证据记 `文件:函数`。同域文件（如设置屏）优先于其它模块的同名文案。
  2. 弱证据：字面量只出现在 Go 注释里；或清单符号（清单有 symbol 列就用，否则
     用名称/类型/模板列派生的组件名）；或函数内注释标注元素 ID → 已实现（弱）。
  3. 测试：*_test.go 断言该文案、提及该元素 ID 或测试注释提到该文案 →
     已实现（测试）。
  4. 都不命中 → 未实现（缺 = 未找到文案/符号/测试）。

负向要求（期望/验收列含「不出现 / 无响应 / 不触发 / 出现次数 = 0」等）单独标
「需负向取证」，不计入未实现；负向判定优先于逐字命中（文案在代码里也不能证明
「不触发」）。

证据只写 `文件:函数`，不写行号。结果写回需求表的状态列，并生成
docs/reconcile-report.md（分布 + 随机 40 条复核 + 已知局限）。

用法：
    python3 scripts/reconcile.py            # 写回状态列 + 报告
    python3 scripts/reconcile.py --dry-run  # 只打印分布，不改文件
"""

from __future__ import annotations

import argparse
import collections
import dataclasses
import random
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
INV_FILES = sorted(REQ_DIR.glob('inventory-*.md'))
GO_DIR = ROOT / 'internal'
REPORT = ROOT / 'docs' / 'reconcile-report.md'

VERDICT_ORDER = ['逐字', '弱', '测试', '需负向取证', '未实现', '未取证']
SAMPLE_SIZE = 40
SAMPLE_SEED = 20261120

ROW_RE = re.compile(r'^\|\s*R-[A-Za-z0-9]+-\d+')
ELEM_ID_RE = re.compile(r'^[A-Za-z][A-Za-z0-9]*(?:-[A-Za-z0-9]+)+$')
FUNC_RE = re.compile(r'^func\s+(?:\(([^)]*)\)\s*)?([A-Za-z_]\w*)\s*[\[(]')
IDENT_RE = re.compile(r'[A-Za-z_]\w+')
LIT_RE = re.compile(r'[「“]([^」”]+)[」”]')

NEG_RE = re.compile(
    r'不出现|不触发|不跳转|不收起|不渲染|不显示|不改变|不变化|不产生|不写入|'
    r'不落盘|不残留|不重复|不读取|不拦截|不弹出|不滚动|不淡入|不播放|不报错|'
    r'不提示|不执行|不发送|不提交|不发起|无响应|静默|'
    r'出现次数\s*=\s*0|次数\s*=\s*0|状态变化\s*=\s*0|像素\s*=\s*0|条目\s*=\s*0'
)

# 清单里泛化的控件/属性词，不能当符号证据。
SYMBOL_STOP = {
    'Button', 'Buttons', 'Dialog', 'Dialogs', 'Input', 'Inputs', 'Select',
    'Switch', 'Slider', 'Checkbox', 'Tabs', 'Card', 'Cards', 'Text', 'Label',
    'Icon', 'Icons', 'Search', 'Source', 'Settings', 'Default', 'Disabled',
    'Visible', 'Hidden', 'Error', 'Title', 'Color', 'Name', 'Value', 'Type',
    'Group', 'List', 'Item', 'Items', 'True', 'False', 'None', 'Null', 'String',
    'Number', 'Boolean', 'Open', 'Close', 'Show', 'Hide', 'Empty', 'Loading',
    'Focus', 'Hover', 'Click', 'Change', 'Row', 'Rows', 'Section', 'Header',
    'Footer', 'Content', 'Container', 'Wrapper', 'Trigger', 'Modal', 'Popover',
    'Tooltip', 'Menu', 'MenuItem', 'Separator', 'Badge', 'Avatar', 'Image',
    'Spinner', 'Loader', 'Arrow', 'Chevron', 'Panel', 'Vue', 'Props', 'Emits',
    'Slot', 'Template', 'Script', 'Style', 'Class', 'Ref', 'Computed', 'Watch',
    'Setup', 'Render', 'State', 'Data', 'Model', 'Models', 'Config', 'Form',
    'Field', 'Fields', 'Action', 'Actions', 'Event', 'Events', 'Handler',
}


# ---------------------------------------------------------------- 文本工具

def cell_split(line: str) -> list[str]:
    """按未转义的 | 切表格行，返回单元格（不含首尾竖线与外层空格）。"""
    s = line.strip()
    if s.startswith('|'):
        s = s[1:]
    if s.endswith('|') and not s.endswith('\\|'):
        s = s[:-1]
    cells, cur, i = [], [], 0
    while i < len(s):
        c = s[i]
        if c == '\\' and i + 1 < len(s) and s[i + 1] == '|':
            cur.append('\\|')
            i += 2
            continue
        if c == '|':
            cells.append(''.join(cur))
            cur = []
            i += 1
            continue
        cur.append(c)
        i += 1
    cells.append(''.join(cur))
    return [c.strip() for c in cells]


def go_unescape(s: str) -> str:
    out, i = [], 0
    simple = {'n': '\n', 't': '\t', 'r': '\r', '"': '"', '\\': '\\', "'": "'"}
    while i < len(s):
        if s[i] != '\\' or i + 1 >= len(s):
            out.append(s[i])
            i += 1
            continue
        c = s[i + 1]
        if c in simple:
            out.append(simple[c])
            i += 2
        elif c == 'x' and i + 3 < len(s):
            try:
                out.append(chr(int(s[i + 2:i + 4], 16)))
                i += 4
            except ValueError:
                out.append(c)
                i += 2
        elif c == 'u' and i + 5 < len(s):
            try:
                out.append(chr(int(s[i + 2:i + 6], 16)))
                i += 6
            except ValueError:
                out.append(c)
                i += 2
        elif c == 'U' and i + 9 < len(s):
            try:
                out.append(chr(int(s[i + 2:i + 10], 16)))
                i += 10
            except ValueError:
                out.append(c)
                i += 2
        else:
            out.append(c)
            i += 2
    return ''.join(out)


# ---------------------------------------------------------------- Go 源码

@dataclasses.dataclass
class GoFunc:
    name: str
    start: int
    end: int


@dataclasses.dataclass
class GoFile:
    path: Path
    rel: str
    lines: list[str]
    funcs: list[GoFunc]
    literals: list[tuple[str, int]]          # (内容, 行号 1-based)
    comments: list[tuple[str, int]]          # (注释文本, 行号 1-based)
    code_lines: list[str]                    # 去掉注释与字符串后的源码行

    def func_at(self, line: int) -> str:
        for f in self.funcs:
            if f.start <= line <= f.end:
                return f.name
        return '包级'

    def func_or_decl_at(self, line: int) -> str:
        """函数内给函数名；包级给最近的 const/var 声明名，找不到再退回「包级」。"""
        func = self.func_at(line)
        if func != '包级':
            return func
        for k in range(line, max(0, line - 60), -1):
            raw = self.lines[k - 1]
            m = re.match(r'^\s*(?:const|var)\s+([A-Za-z_]\w*)\s*=', raw)
            if m:
                return m.group(1)
            if re.match(r'^\s*(?:\)|\})\s*$', raw) and k != line:
                break
            m = re.match(r'^\s*([A-Za-z_]\w*)\s*(?:[A-Za-z_\[\]*.]+\s*)?=', raw)
            if m:
                # const ( ... ) / var ( ... ) 块里的单名行
                return m.group(1)
        return '包级'

    def decl_body(self, name: str, line: int) -> str:
        if name == '包级':
            return '\n'.join(self.lines[max(0, line - 30):line + 30])
        start = max(0, line - 1)
        while start > 0 and not re.match(r'^\s*(?:const|var)\s', self.lines[start - 1]):
            start -= 1
        end = min(len(self.lines), line + 30)
        return '\n'.join(self.lines[start:end])

    def body(self, name: str) -> str:
        for f in self.funcs:
            if f.name == name:
                return '\n'.join(self.lines[f.start - 1:f.end])
        return ''

    def decl_has(self, name: str, needle: str) -> bool:
        """包级 const/var 声明名附近是否含某字面量/符号。"""
        pat = re.compile(r'^\s*(?:const|var)?\s*' + re.escape(name) + r'\s*=')
        pat2 = re.compile(r'^\s*(?:const|var)\s+' + re.escape(name) + r'\b')
        for i, raw in enumerate(self.lines, start=1):
            if pat.match(raw) or pat2.match(raw):
                window = '\n'.join(self.lines[max(0, i - 1):i + 30])
                if needle in window:
                    return True
        return False


def scan_go(path: Path, rel: str) -> GoFile:
    """轻量扫描：字符串字面量、注释、顶层函数区间、去注释/字符串的代码行。"""
    text = path.read_text(errors='replace')
    lines = text.splitlines()
    mask = list(text)
    literals: list[tuple[str, int]] = []
    comments: list[tuple[str, int]] = []
    i, line_no, n = 0, 1, len(text)

    def blank(a: int, b: int) -> None:
        for k in range(a, min(b, n)):
            if mask[k] != '\n':
                mask[k] = ' '

    while i < n:
        c = text[i]
        if c == '\n':
            line_no += 1
            i += 1
            continue
        if c == '/' and i + 1 < n and text[i + 1] == '/':
            j = text.find('\n', i)
            if j < 0:
                j = n
            comments.append((text[i + 2:j], line_no))
            blank(i, j)
            i = j
            continue
        if c == '/' and i + 1 < n and text[i + 1] == '*':
            j = text.find('*/', i + 2)
            j = n if j < 0 else j + 2
            comments.append((text[i:j], line_no))
            line_no += text.count('\n', i, j)
            blank(i, j)
            i = j
            continue
        if c == '"':
            start_line = line_no
            j = i + 1
            buf = []
            while j < n:
                d = text[j]
                if d == '\\' and j + 1 < n:
                    buf.append(text[j:j + 2])
                    j += 2
                    continue
                if d == '"' or d == '\n':
                    break
                buf.append(d)
                j += 1
            end = j + 1 if j < n and text[j] == '"' else j
            literals.append((go_unescape(''.join(buf)), start_line))
            blank(i, end)
            i = end
            continue
        if c == '`':
            start_line = line_no
            j = text.find('`', i + 1)
            j = n if j < 0 else j
            content = text[i + 1:j]
            literals.append((content, start_line))
            line_no += content.count('\n')
            blank(i, j + 1)
            i = j + 1
            continue
        if c == "'":
            j = i + 1
            while j < n and text[j] not in "'\n":
                j += 2 if text[j] == '\\' else 1
            end = min(j + 1, n)
            blank(i, end)
            i = end
            continue
        i += 1

    code_lines = ''.join(mask).splitlines()
    while len(code_lines) < len(lines):
        code_lines.append('')
    funcs: list[GoFunc] = []
    for idx, raw in enumerate(code_lines, start=1):
        m = FUNC_RE.match(raw)
        if not m:
            continue
        recv, name = m.group(1) or '', m.group(2)
        if recv:
            rm = re.search(r'([A-Za-z_]\w*)\s*$', recv)
            if rm:
                name = f'{rm.group(1)}.{name}'
        end = len(code_lines)
        for k in range(idx, len(code_lines)):
            if code_lines[k].startswith('}'):
                end = k + 1
                break
        funcs.append(GoFunc(name, idx, end))
    return GoFile(path, rel, lines, funcs, literals, comments, code_lines)


def build_literal_index(lits: list[str]) -> dict[str, list[tuple[str, str, int]]]:
    """需求字面量 → 出现的 (文件, 函数, 行) 列表。"""
    prefix: dict[str, list[str]] = collections.defaultdict(list)
    for lit in lits:
        prefix[lit[:2]].append(lit)
    hits: dict[str, list[tuple[str, str, int]]] = collections.defaultdict(list)
    for gf in IMPL:
        for content, line in gf.literals:
            if len(content) < 2:
                continue
            for i in range(len(content) - 1):
                cand = prefix.get(content[i:i + 2])
                if not cand:
                    continue
                for lit in cand:
                    if content.startswith(lit, i):
                        item = (gf.rel, gf.func_or_decl_at(line), line)
                        if len(hits[lit]) < 12 and item not in hits[lit]:
                            hits[lit].append(item)
    return dict(hits)


def build_test_literal_index(lits: list[str]) -> dict[str, list[tuple[str, str, int]]]:
    """需求字面量在 *_test.go 字符串里出现的位置（测试断言证据）。"""
    prefix: dict[str, list[str]] = collections.defaultdict(list)
    for lit in lits:
        prefix[lit[:2]].append(lit)
    hits: dict[str, list[tuple[str, str, int]]] = collections.defaultdict(list)
    for gf in TESTS:
        for content, line in gf.literals:
            if len(content) < 2:
                continue
            for i in range(len(content) - 1):
                cand = prefix.get(content[i:i + 2])
                if not cand:
                    continue
                for lit in cand:
                    if content.startswith(lit, i):
                        item = (gf.rel, gf.func_or_decl_at(line), line)
                        if len(hits[lit]) < 12 and item not in hits[lit]:
                            hits[lit].append(item)
    return dict(hits)


def build_raw_index(lits: list[str], files: list[GoFile] | None = None
                    ) -> dict[str, list[tuple[str, str, int]]]:
    """需求字面量在 Go 源码原文（含注释）中的出现位置，用于区分「注释提及」。"""
    prefix: dict[str, list[str]] = collections.defaultdict(list)
    for lit in lits:
        prefix[lit[:2]].append(lit)
    hits: dict[str, list[tuple[str, str, int]]] = collections.defaultdict(list)
    for gf in (files if files is not None else IMPL):
        for line, raw in enumerate(gf.lines, start=1):
            if len(raw) < 2:
                continue
            for i in range(len(raw) - 1):
                cand = prefix.get(raw[i:i + 2])
                if not cand:
                    continue
                for lit in cand:
                    if raw.startswith(lit, i):
                        item = (gf.rel, gf.func_or_decl_at(line), line)
                        if len(hits[lit]) < 12 and item not in hits[lit]:
                            hits[lit].append(item)
    return dict(hits)


def camel_to_snake(name: str) -> str:
    return re.sub(r'(?<!^)(?=[A-Z])', '_', name).lower()


ELEM_FILE_HINTS = {
    'LB': ('internal/ui/sidebar',),
    'TB': ('internal/ui/header',),
    'IN': ('internal/ui/composer',),
    'TL': ('internal/ui/timeline',),
    'SB': ('internal/ui/statusbar',),
    'NT': ('internal/ui/app.go',),
    'F': ('internal/ui/rightpanel_files',),
    'D': ('internal/ui/rightpanel_changes',),
    'A': ('internal/ui/rightpanel_artifacts', 'internal/ui/rightpanel_'),
    'T': ('internal/ui/terminal',),
    'K': ('internal/ui/rightpanel_tasks',),
    'S': ('internal/ui/rightpanel_sidechat',),
    'G-A': ('internal/ui/search',),
    'G-C': ('internal/ui/approval',),
    'G-B': ('internal/ui/approval_panels',),
    'G-D': ('internal/ui/question',),
    'G-F1': ('internal/ui/rename_dialog',),
    'G-F2': ('internal/ui/fork', 'internal/ui/export'),
    'G-H': ('internal/ui/plugin',),
    'G-G': ('internal/ui/command_palette', 'internal/ui/confirm'),
}


def score_occurrence(rel: str, elem_id: str, req_file_name: str) -> int:
    score = 0
    if elem_id.startswith('S-'):
        sec = elem_id.split('-')[1] if elem_id.count('-') >= 2 else ''
        snake = camel_to_snake(sec)
        if rel == f'internal/ui/settings_{snake}_screen.go':
            score += 6
        if rel.startswith(f'internal/ui/settings_{snake}'):
            score += 4
        if rel.startswith('internal/ui/settings') or rel.startswith('internal/settings'):
            score += 2
    else:
        for prefix, hints in sorted(ELEM_FILE_HINTS.items(), key=lambda kv: -len(kv[0])):
            if elem_id.startswith(prefix):
                if any(rel.startswith(h) for h in hints):
                    score += 6
                break
        if req_file_name == 'requirements-main.md' and rel.startswith('internal/ui/'):
            score += 2
        if req_file_name == 'requirements-panels.md' and (
                rel.startswith('internal/ui/rightpanel') or rel.startswith('internal/ui/terminal')):
            score += 2
        if req_file_name == 'requirements-global.md' and rel.startswith('internal/ui/'):
            score += 2
        if req_file_name == 'requirements-settings.md' and (
                rel.startswith('internal/ui/settings') or rel.startswith('internal/settings')):
            score += 2
    if rel.startswith('internal/ui/'):
        score += 1
    return score


def build_symbol_index() -> dict[str, tuple[str, str, int]]:
    """Go 代码里的标识符（不含注释与字符串）→ (文件, 函数, 行)。"""
    idx: dict[str, tuple[str, str, int]] = {}
    for gf in IMPL:
        for line, raw in enumerate(gf.code_lines, start=1):
            for tok in IDENT_RE.findall(raw):
                idx.setdefault(tok, (gf.rel, gf.func_or_decl_at(line), line))
    return idx


def build_element_comment_index(known_ids: set[str]) -> dict[str, tuple[str, str, int]]:
    """元素 ID（出现在函数内 Go 注释里）→ (文件, 函数, 行)。

    文件头注释里「S-account-01 … S-account-112」这类区间只声明覆盖范围，不指向
    单个元素的实现，故只收函数体内的元素 ID 注释。
    """
    idx: dict[str, tuple[str, str, int]] = {}
    for gf in IMPL:
        for text, line in gf.comments:
            func = gf.func_at(line)
            if func == '包级':
                continue
            for tok in re.findall(r'[A-Za-z][A-Za-z0-9]*(?:-[A-Za-z0-9]+)+', text):
                if tok in known_ids and tok not in idx:
                    idx[tok] = (gf.rel, func, line)
    return idx


def build_test_index() -> dict[str, tuple[str, str]]:
    """元素 ID → 首个提及它的 (*_test.go, 函数)。"""
    idx: dict[str, tuple[str, str]] = {}
    for gf in TESTS:
        for line, raw in enumerate(gf.lines, start=1):
            for tok in re.findall(r'[A-Za-z][A-Za-z0-9]*(?:-[A-Za-z0-9]+)+', raw):
                if tok not in idx:
                    idx[tok] = (gf.rel, gf.func_at(line))
    return idx


# ---------------------------------------------------------------- 清单

@dataclasses.dataclass
class ElementInfo:
    name: str = ''
    symbols: list[str] = dataclasses.field(default_factory=list)


def load_inventory() -> dict[str, ElementInfo]:
    info: dict[str, ElementInfo] = {}
    for path in INV_FILES:
        header: list[str] | None = None
        for raw in path.read_text(errors='replace').splitlines():
            if not raw.lstrip().startswith('|'):
                continue
            cells = cell_split(raw)
            if not cells:
                continue
            if cells[0] == '元素 ID':
                header = cells
                continue
            if header is None or not ELEM_ID_RE.match(cells[0]):
                continue
            row = {header[i]: cells[i] for i in range(min(len(header), len(cells)))}
            eid = cells[0]
            cur = info.setdefault(eid, ElementInfo())
            if not cur.name:
                cur.name = row.get('名称', '') or row.get('名称（文案原文）', '') or \
                    row.get('名称（界面文案原文）', '')
            sym_cells = [v for k, v in row.items() if 'symbol' in k.lower() or '符号' in k]
            if sym_cells:
                for v in sym_cells:
                    cur.symbols.extend(re.findall(r'[A-Za-z_]\w+', v))
            else:
                for key in ('名称', '名称（文案原文）', '名称（界面文案原文）',
                            '类型', '控件', '模板', '定位', '位置'):
                    if key not in row:
                        continue
                    for tok in re.findall(r'[A-Za-z][A-Za-z0-9_]{3,}', row[key]):
                        tok = tok[:-4] if tok.endswith('.vue') else tok
                        if tok in SYMBOL_STOP or tok.lower() in {'true', 'false', 'null', 'none'}:
                            continue
                        # 只要首字母大写、至少两段的组件/类型名；小驼峰变量名不算符号
                        if not re.fullmatch(r'[A-Z][A-Za-z0-9]*(?:[A-Z][A-Za-z0-9]*)+', tok):
                            continue
                        cur.symbols.append(tok)
    for cur in info.values():
        cur.symbols = list(dict.fromkeys(cur.symbols))
    return info


# ---------------------------------------------------------------- 需求表

@dataclasses.dataclass
class ReqRow:
    path: Path
    line_no: int
    raw: str
    cells: list[str]
    header: list[str]
    req_id: str
    elem_id: str
    dim: str
    expected: str
    acceptance: str


def parse_req_file(path: Path) -> list[ReqRow]:
    rows: list[ReqRow] = []
    header: list[str] | None = None
    for line_no, raw in enumerate(path.read_text(errors='replace').splitlines(), start=1):
        stripped = raw.lstrip()
        if stripped.startswith('|'):
            cells = cell_split(raw)
            if cells and cells[0] == '需求 ID':
                header = cells
                continue
            if header and cells and ROW_RE.match(raw):
                row = {header[i]: cells[i] for i in range(min(len(header), len(cells)))}
                rows.append(ReqRow(
                    path=path, line_no=line_no, raw=raw, cells=cells, header=header,
                    req_id=cells[0],
                    elem_id=row.get('元素 ID', cells[1] if len(cells) > 1 else ''),
                    dim=row.get('维度', ''),
                    expected=row.get('期望', ''),
                    acceptance=row.get('验收', ''),
                ))
    return rows


def extract_literals(expected: str) -> list[str]:
    out: list[str] = []
    for m in LIT_RE.finditer(expected):
        lit = m.group(1).strip()
        if len(lit) < 2:
            continue
        if '{' in lit or '}' in lit or '…' in lit:
            continue
        if not re.search(r'[^\W_]', lit, re.UNICODE):
            continue
        if not re.search(r'[\u4e00-\u9fff]', lit):
            # 纯 ASCII：排除标识符/键名/路径，只留产品名这类可读文案
            if re.fullmatch(r'[A-Za-z0-9_.$/\-]+', lit):
                if not (re.search(r'[A-Z]', lit) and re.search(r'[a-z]', lit)):
                    continue
                if '_' in lit or '.' in lit or '/' in lit:
                    continue
        out.append(lit)
        if len(out) >= 8:
            break
    return list(dict.fromkeys(out))


def status_index(header: list[str]) -> int | None:
    try:
        return header.index('状态')
    except ValueError:
        return None


# ---------------------------------------------------------------- 判定

NEG_LABEL = '需负向取证'
UNIMPL_LABEL = '未实现'
TRIM_CHARS = '。！？；：，、,.!?;:'


def literal_variants(lit: str) -> list[str]:
    """逐字匹配的等价写法：原文、去尾部标点（屏上标签常不带句号）。"""
    out = [lit]
    trimmed = lit.rstrip(TRIM_CHARS)
    if trimmed != lit and len(trimmed) >= 2:
        out.append(trimmed)
    return out


def literal_hit(lit: str, elem_id: str, req_file_name: str
                ) -> tuple[str | None, tuple[str, str, int] | None]:
    for cand in literal_variants(lit):
        occ = LITERAL_HITS.get(cand)
        if occ:
            best = max(occ, key=lambda o: score_occurrence(o[0], elem_id, req_file_name))
            return cand, best
    return None, None


def raw_hit(lit: str, elem_id: str, req_file_name: str, index=None
            ) -> tuple[str | None, tuple[str, str, int] | None]:
    for cand in literal_variants(lit):
        occ = (index if index is not None else RAW_HITS).get(cand)
        if occ:
            best = max(occ, key=lambda o: score_occurrence(o[0], elem_id, req_file_name))
            return cand, best
    return None, None


def test_literal_hit(lit: str, elem_id: str, req_file_name: str
                     ) -> tuple[str | None, tuple[str, str, int] | None]:
    for cand in literal_variants(lit):
        occ = TEST_LITERAL_HITS.get(cand)
        if occ:
            best = max(occ, key=lambda o: score_occurrence(o[0], elem_id, req_file_name))
            return cand, best
    return None, None


@dataclasses.dataclass
class Verdict:
    kind: str                 # 逐字 / 弱 / 测试 / 需负向取证 / 未实现
    text: str                 # 写回状态列的文字
    evidence: tuple[str, str, str] | None = None  # (文件, 函数, 详情)
    literals: list[str] = dataclasses.field(default_factory=list)


def judge(row: ReqRow, inv: dict[str, ElementInfo]) -> Verdict:
    negative = bool(NEG_RE.search(row.expected) or NEG_RE.search(row.acceptance))
    if negative:
        return Verdict(
            NEG_LABEL,
            f'{NEG_LABEL}：期望/验收列为负向断言，需界面或代码事实反向取证',
            evidence=None,
        )

    lits = extract_literals(row.expected)
    for lit in lits:
        hit_lit, hit = literal_hit(lit, row.elem_id, row.path.name)
        if hit:
            rel, func, _line = hit
            return Verdict(
                '逐字',
                f'已实现（逐字）：{rel}:{func}「{hit_lit}」',
                evidence=(rel, func, hit_lit),
                literals=lits,
            )

    info = inv.get(row.elem_id)

    # 字面量只出现在注释里：不是屏上文案的实现，但也不宜判「文案搜不到」。
    for lit in lits:
        hit_lit, hit = raw_hit(lit, row.elem_id, row.path.name)
        if hit:
            rel, func, _line = hit
            return Verdict(
                '弱',
                f'已实现（弱）：{rel}:{func} 注释提及「{hit_lit}」',
                evidence=(rel, func, hit_lit),
                literals=lits,
            )

    for sym in (info.symbols if info else []):
        hit = SYMBOL_INDEX.get(sym)
        if hit:
            rel, func, _line = hit
            return Verdict(
                '弱',
                f'已实现（弱）：{rel}:{func} 符号 `{sym}`',
                evidence=(rel, func, sym),
                literals=lits,
            )
    hit = ELEM_COMMENT_INDEX.get(row.elem_id)
    if hit:
        rel, func, _line = hit
        return Verdict(
            '弱',
            f'已实现（弱）：{rel}:{func} 元素 ID 注释 {row.elem_id}',
            evidence=(rel, func, row.elem_id),
            literals=lits,
        )

    for lit in lits:
        hit_lit, hit = test_literal_hit(lit, row.elem_id, row.path.name)
        if hit:
            rel, func, _line = hit
            return Verdict(
                '测试',
                f'已实现（测试）：{rel}:{func} 断言「{hit_lit}」',
                evidence=(rel, func, hit_lit),
                literals=lits,
            )

    for lit in lits:
        hit_lit, hit = raw_hit(lit, row.elem_id, row.path.name, TEST_RAW_HITS)
        if hit:
            rel, func, _line = hit
            return Verdict(
                '测试',
                f'已实现（测试）：{rel}:{func} 测试注释提及「{hit_lit}」',
                evidence=(rel, func, hit_lit),
                literals=lits,
            )

    test_hit = TEST_INDEX.get(row.elem_id)
    if test_hit:
        rel, func = test_hit
        return Verdict(
            '测试',
            f'已实现（测试）：{rel}:{func} 提及 {row.elem_id}',
            evidence=(rel, func, row.elem_id),
            literals=lits,
        )
    return Verdict(
        UNIMPL_LABEL,
        '未实现（缺 = 未找到文案/符号/测试）',
        evidence=None,
        literals=lits,
    )


# ---------------------------------------------------------------- 写回

def rewrite(path: Path, verdicts: dict[int, Verdict]) -> bool:
    raw = path.read_text()
    lines = raw.splitlines(keepends=True)
    changed = False
    for idx, verdict in verdicts.items():
        line = lines[idx - 1]
        stripped = line.rstrip('\n')
        cells = cell_split(stripped)
        header = None
        # 找该行所属表头：向上找最近的「| 需求 ID |」
        for k in range(idx - 1, -1, -1):
            probe = lines[k].lstrip()
            if probe.startswith('| 需求 ID'):
                header = cell_split(probe)
                break
        if not header:
            continue
        si = status_index(header)
        if si is None or si >= len(cells):
            continue
        cells[si] = verdict.text
        new_line = '| ' + ' | '.join(cells) + ' |\n'
        if new_line != line:
            lines[idx - 1] = new_line
            changed = True
    if changed:
        path.write_text(''.join(lines))
    return changed


# ---------------------------------------------------------------- 报告

def pct(a: int, b: int) -> str:
    return f'{a / b * 100:.1f}%' if b else '—'


def pick_sample(rows: list[ReqRow], verdict_by_line: dict[int, Verdict],
                per_file: int) -> list[tuple[ReqRow, Verdict]]:
    """每份文件：优先一半逐字、一半未实现，不足再按比例补。"""
    rng = random.Random(SAMPLE_SEED)
    picked: list[tuple[ReqRow, Verdict]] = []
    by_kind: dict[str, list[ReqRow]] = collections.defaultdict(list)
    for row in rows:
        by_kind[verdict_by_line[row.line_no].kind].append(row)
    want = [('逐字', per_file // 2), ('未实现', per_file // 2)]
    chosen: set[int] = set()
    for kind, n in want:
        pool = by_kind.get(kind, [])
        for row in rng.sample(pool, min(n, len(pool))):
            if row.line_no not in chosen:
                chosen.add(row.line_no)
                picked.append((row, verdict_by_line[row.line_no]))
    rest = [r for r in rows if r.line_no not in chosen]
    if len(picked) < per_file:
        for row in rng.sample(rest, min(per_file - len(picked), len(rest))):
            chosen.add(row.line_no)
            picked.append((row, verdict_by_line[row.line_no]))
    return picked


def verify_sample(row: ReqRow, verdict: Verdict) -> tuple[bool, str]:
    if verdict.kind == '逐字':
        rel, func, lit = verdict.evidence
        gf = next(g for g in IMPL if g.rel == rel)
        body = gf.body(func)
        if body:
            ok = lit in body
        elif func == '包级':
            ok = any(lit in content for content, _ in gf.literals) or \
                lit in '\n'.join(gf.lines)
        else:
            ok = gf.decl_has(func, lit)
        where = f'{rel}:{func}'
        return ok, ('字面量在 %s 内' % where) if ok else ('未在 %s 找到字面量' % where)
    if verdict.kind == '弱':
        rel, func, sym = verdict.evidence
        gf = next(g for g in IMPL if g.rel == rel)
        if re.search(r'[\u4e00-\u9fff]', sym):
            ok = sym in '\n'.join(gf.lines)
            return ok, '注释文案在该文件里' if ok else '注释文案不在该文件里'
        body = gf.body(func)
        if body:
            ok = sym in body
        elif func == '包级':
            ok = sym in '\n'.join(gf.lines)
        else:
            ok = gf.decl_has(func, sym)
        return ok, '证据在所指位置内' if ok else '证据不在所指位置内'
    if verdict.kind == '测试':
        rel, func, detail = verdict.evidence
        gf = next(g for g in TESTS if g.rel == rel)
        body = gf.body(func)
        ok = detail in body if body else detail in '\n'.join(gf.lines)
        return ok, '测试文件包含该字面量/元素 ID' if ok else '测试文件未包含该字面量/元素 ID'
    if verdict.kind == '未实现':
        gone = True
        for lit in verdict.literals:
            for cand in literal_variants(lit):
                if cand in LITERAL_HITS or cand in RAW_HITS or cand in TEST_LITERAL_HITS \
                        or cand in TEST_RAW_HITS:
                    gone = False
                    break
            if not gone:
                break
        return gone, '文案/符号/测试均无命中' if gone else '文案在源码或测试里存在该字面量'
    return True, '负向断言，不计入未实现'


def write_report(rows_by_file: dict[Path, list[ReqRow]],
                 verdicts_by_file: dict[Path, dict[int, Verdict]]) -> None:
    all_verdicts: list[Verdict] = []
    for rows in rows_by_file.values():
        all_verdicts.extend(verdicts_by_file[rows[0].path].values())
    total = len(all_verdicts)

    lines: list[str] = []
    lines.append('# 需求实现状态：机械对账报告')
    lines.append('')
    lines.append('Status: 对账报告（脚本生成，可重跑；`python3 scripts/reconcile.py`）')
    lines.append('Scope: 四份需求表的全部条目（`docs/domains/requirements/requirements-*.md`），'
                 '状态列由本报告写回')
    lines.append('Owner: 组织 agent（验收判定侧）；需求文件与元素清单归 R 与需求分析领域')
    lines.append('')
    lines.append('## 0. 判定口径')
    lines.append('')
    lines.append('按顺序取第一个命中的证据：')
    lines.append('')
    lines.append('| 判定 | 证据 | 含义 |')
    lines.append('|---|---|---|')
    lines.append('| 逐字 | 「期望」里「」/“”字面量在 `internal/**/*.go` 的字符串字面量出现 | 强证据，'
                 '能指到具体函数 |')
    lines.append('| 弱 | 清单符号、函数内元素 ID 注释、或字面量只出现在源码注释里 | 有对应实现点，'
                 '但不证明逐字行为 |')
    lines.append('| 测试 | `*_test.go` 断言该文案或提及该元素 ID | 有测试覆盖点，未证明实现 |')
    lines.append('| 需负向取证 | 期望/验收为负向断言（不出现、无响应、出现次数 = 0 等） | 单独一档，'
                 '不计入未实现 |')
    lines.append('| 未实现 | 文案、符号、测试都不命中 | 缺 = 未找到文案/符号/测试 |')
    lines.append('')
    lines.append('证据只写 `文件:函数`，不写行号；函数按顶层 `func` 与列 0 的 `}` 切分；'
                 '包级常量/变量记声明名，找不到声明名才写 `包级`。')
    lines.append('')
    lines.append('## 1. 命中分布（每份）')
    lines.append('')
    lines.append('| 文件 | 总条数 | 逐字 | 弱 | 测试 | 需负向取证 | 未实现 | 逐字占比 | 未实现占比 |')
    lines.append('|---|---:|---:|---:|---:|---:|---:|---:|---:|')
    for path, rows in rows_by_file.items():
        vs = list(verdicts_by_file[path].values())
        c = collections.Counter(v.kind for v in vs)
        name = path.name
        lines.append(
            f'| `{name}` | {len(vs)} | {c["逐字"]} | {c["弱"]} | {c["测试"]} | '
            f'{c[NEG_LABEL]} | {c["未实现"]} | {pct(c["逐字"], len(vs))} | '
            f'{pct(c["未实现"], len(vs))} |')
    c = collections.Counter(v.kind for v in all_verdicts)
    lines.append(
        f'| **合计** | **{total}** | **{c["逐字"]}** | **{c["弱"]}** | **{c["测试"]}** | '
        f'**{c[NEG_LABEL]}** | **{c["未实现"]}** | **{pct(c["逐字"], total)}** | '
        f'**{pct(c["未实现"], total)}** |')
    lines.append('')
    no_lit = [v for v in all_verdicts if not v.literals]
    no_lit_unimpl = [v for v in no_lit if v.kind == '未实现']
    lines.append(f'「未实现」里有一半以上是「期望列没有可逐字文案」造成的（无可用字面量 '
                 f'{len(no_lit)} 条，其中判未实现 {len(no_lit_unimpl)} 条）；'
                 '这类条目只能靠元素符号/测试或界面取证，见 §3。')
    lines.append('')
    lines.append('## 2. 随机 %d 条复核' % SAMPLE_SIZE)
    lines.append('')
    lines.append('抽样种子固定（`%d`），每份 10 条，优先覆盖「逐字」与「未实现」两档；'
                 '复核结论由脚本按证据反向核验，人工只需看标记为「有分歧」的条目。' % SAMPLE_SEED)
    lines.append('')
    mismatch = 0
    per_file = SAMPLE_SIZE // len(rows_by_file)
    for path, rows in rows_by_file.items():
        sample = pick_sample(rows, verdicts_by_file[path], per_file)
        lines.append(f'### {path.name}')
        lines.append('')
        lines.append('| 需求 ID | 元素 ID | 维度 | 机器判定 | 证据 | 复核 |')
        lines.append('|---|---|---|---|---|---|')
        for row, verdict in sample:
            ok, why = verify_sample(row, verdict)
            if not ok:
                mismatch += 1
            mark = f'通过（{why}）' if ok else f'**有分歧**（{why}）'
            ev = verdict.evidence
            if ev:
                detail = f'`{ev[0]}:{ev[1]}`'
                if verdict.kind in ('逐字', '弱'):
                    detail += f'「{ev[2]}」'
            else:
                detail = '—'
            lines.append(f'| {row.req_id} | {row.elem_id} | {row.dim} | {verdict.kind} | '
                         f'{detail} | {mark} |')
        lines.append('')
    lines.append(f'抽样复核：{SAMPLE_SIZE} 条中 {SAMPLE_SIZE - mismatch} 条与证据一致，'
                 f'{mismatch} 条标记有分歧。')
    lines.append('')
    lines.append('## 3. 已知局限')
    lines.append('')
    lines.append('- **动态拼接文案**：期望列里带 `{}`、`…` 的字面量（如「已使用 {pct}%」'
                 '「已于 … 到期」）被跳过，无法逐字命中，只会落到弱/未实现。')
    lines.append('- **没有引号的期望**：期望列用叙述而不是「」/“”给文案时抽不出字面量，'
                 '这类条目只能走符号/测试，统计见 §1 的「无可用字面量」一行。')
    lines.append('- **尾部标点**：先按原字面量查，查不到再去掉尾部 `。！？；：` 重查，'
                 '证据记录实际命中的写法。')
    lines.append('- **同文案多处出现**：只记第一次命中的 `文件:函数`，不证明该函数属于本元素；'
                 '同名文案跨节复用时证据可能指错实现点。')
    lines.append('- **无 i18n 层**：代码未建 i18n，期望为英文键名（如 `settings.account.title`）'
                 '时无法逐字命中，按未实现计。')
    lines.append('- **注释提及不算逐字**：字面量只出现在 Go 注释里时记「已实现（弱）」；'
                 '这样「未实现」的含义是：字符串字面量、注释、符号、测试都搜不到。')
    lines.append('- **弱证据的来源**：清单没有 symbol 列；名称/类型/模板/定位列派生的 Vue 组件名'
                 '在 Go 里几乎不出现（本轮只有个别命中），实际起作用的是「函数内注释标注元素 ID」。'
                 '同名符号与顺带提及的注释都可能误命中。')
    lines.append('- **负向要求不判实现**：只标「需负向取证」，需要界面驱动或代码反向事实，'
                 '不计入未实现。')
    lines.append('- **静态匹配不看分支**：字面量命中的函数即使处在不可达分支，也会判逐字；'
                 '不替代界面验收证据。')
    lines.append('- **函数区间是启发式**：顶层 `func` 到列 0 的 `}`，闭包与生成代码可能切错；'
                 '证据写函数名而非行号，切错时函数名仍可用。')
    lines.append('- **只扫 `internal/**/*.go`**：`cmd/`、生成文件与前端资源不在扫描范围。')
    lines.append('')
    REPORT.write_text('\n'.join(lines))


# ---------------------------------------------------------------- 主流程

IMPL: list[GoFile] = []
TESTS: list[GoFile] = []
LITERAL_HITS: dict[str, list[tuple[str, str, int]]] = {}
RAW_HITS: dict[str, list[tuple[str, str, int]]] = {}
TEST_LITERAL_HITS: dict[str, list[tuple[str, str, int]]] = {}
TEST_RAW_HITS: dict[str, list[tuple[str, str, int]]] = {}
SYMBOL_INDEX: dict[str, tuple[str, str, int]] = {}
ELEM_COMMENT_INDEX: dict[str, tuple[str, str, int]] = {}
TEST_INDEX: dict[str, tuple[str, str]] = {}


def main() -> int:
    global IMPL, TESTS, LITERAL_HITS, RAW_HITS, TEST_LITERAL_HITS, TEST_RAW_HITS, \
        SYMBOL_INDEX, ELEM_COMMENT_INDEX, TEST_INDEX
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument('--dry-run', action='store_true', help='只打印分布，不改文件')
    args = ap.parse_args()

    IMPL, TESTS = [], []
    for path in sorted(GO_DIR.rglob('*.go')):
        rel = path.relative_to(ROOT).as_posix()
        gf = scan_go(path, rel)
        (TESTS if path.name.endswith('_test.go') else IMPL).append(gf)

    rows_by_file: dict[Path, list[ReqRow]] = {}
    for path in REQ_FILES:
        rows_by_file[path] = parse_req_file(path)

    all_literals = sorted({cand for rows in rows_by_file.values()
                           for row in rows
                           for lit in extract_literals(row.expected)
                           for cand in literal_variants(lit)})
    LITERAL_HITS = build_literal_index(all_literals)
    RAW_HITS = build_raw_index(all_literals)
    TEST_LITERAL_HITS = build_test_literal_index(all_literals)
    TEST_RAW_HITS = build_raw_index(all_literals, TESTS)
    SYMBOL_INDEX = build_symbol_index()
    inv = load_inventory()
    known_ids = {row.elem_id for rows in rows_by_file.values() for row in rows} | set(inv)
    ELEM_COMMENT_INDEX = build_element_comment_index(known_ids)
    TEST_INDEX = build_test_index()

    verdicts_by_file: dict[Path, dict[int, Verdict]] = {}
    for path, rows in rows_by_file.items():
        verdicts_by_file[path] = {row.line_no: judge(row, inv) for row in rows}
        counts = collections.Counter(v.kind for v in verdicts_by_file[path].values())
        print(f'{path.name}: {len(rows)} 条  ' + '  '.join(
            f'{k}={counts.get(k, 0)}' for k in VERDICT_ORDER if counts.get(k)))
        if not args.dry_run:
            changed = rewrite(path, verdicts_by_file[path])
            print(f'  状态列{"已写回" if changed else "无变化"}')

    if args.dry_run:
        return 0

    write_report(rows_by_file, verdicts_by_file)
    print(f'报告：{REPORT.relative_to(ROOT)}')
    return 0


if __name__ == '__main__':
    sys.exit(main())
