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

第二轮（§4）：对没有可逐字文案的叙述型条目改按**元素映射**判定。

  1. 元素索引：函数 doc 注释里的元素 ID（一处注释覆盖的元素越少越优先）→ 函数体内注释
     → 清单 symbol 派生的 Go 标识符。支持 `S-account-02–07`、`S-account-10/11/13`、
     `S-about-01…18` 这类区间/并列写法与 `A1-03` → `G-A1-03` 的短写还原，
     得到 元素 → (文件, 渲染函数)。跨域误命中按元素族过滤（设置元素只认设置屏，
     单字母面板元素不认设置屏）。
  2. 按维度在映射到的函数体里找标记（内容 = 函数内中文字面量，其余见
     DIM_MARKERS）；函数体指去注释、去字符串的代码行 + 函数区间内的字符串字面量，
     并向下展开一跳同包被调函数（方法按裸函数名解析）。
  3. 命中 → 已实现（映射 = 元素→文件:函数；标记 = 标记）；映射到位但无标记 →
     未实现（映射到位但没有该维度的标记）；无映射 → 未实现（找不到元素渲染点）。

证据只写 `文件:函数`，不写行号。结果写回需求表的状态列，并生成
docs/reconcile-report.md（第一轮分布与复核 + 第二轮映射判定 + 随机 40 条复核 +
已知局限）。

用法：
    python3 scripts/reconcile.py            # 写回状态列 + 报告
    python3 scripts/reconcile.py --dry-run  # 只打印分布，不改文件
"""

from __future__ import annotations

import argparse
import collections
import dataclasses
import json
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


# ---------------------------------------------------------------- 元素映射（第二轮）

# 元素 ID 的区间 / 并列写法：`S-account-02–07`、`S-account-10/11/13`、
# `TB-20–TB-30`、`LB-11–15`。展开结果必须落在已知元素 ID 集合里，否则丢弃。
ID_SEQ_RE = re.compile(
    r'(?P<base>[A-Za-z][A-Za-z0-9]*(?:-[A-Za-z0-9]+)*?)-(?P<num>\d+)'
    r'(?P<rest>(?:\s*(?:[–—~～]|\.\.\.|…|/|,|、|和|与)\s*'
    r'(?:[A-Za-z][A-Za-z0-9]*(?:-[A-Za-z0-9]+)*?-)?\d+)*)')
ID_CONT_RE = re.compile(
    r'\s*(?P<sep>[–—~～]|\.\.\.|…|/|,|、|和|与)\s*'
    r'(?:(?P<base>[A-Za-z][A-Za-z0-9]*(?:-[A-Za-z0-9]+)*?)-)?(?P<num>\d+)')
TOKEN_ID_RE = re.compile(r'[A-Za-z][A-Za-z0-9]*(?:-[A-Za-z0-9]+)+')
CJK_RE = re.compile(r'[\u4e00-\u9fff]')
CALL_RE = re.compile(r'(?<![A-Za-z0-9_])([a-z][A-Za-z0-9_]*)\s*\(')
PANEL_ELEM_RE = re.compile(r'^[A-Z]-\d+[a-z]?$')

MAP_SAMPLE_SEED = 20261121
MAP_REVIEW_FILE = ROOT / 'scripts' / 'reconcile-review-mapping.json'

MAP_HIT = '映射命中'
MAP_NO_MARK = '映射到位无标记'
MAP_NO_POINT = '无渲染点'
MAP_NEG = '负向'
MAP_BUCKETS = ['逐字', MAP_HIT, MAP_NO_MARK, MAP_NO_POINT, MAP_NEG]

# ---- 附录：五档分流。词表与元素族是分档口径的唯一来源（报告 §5.1 由它生成）。

TRIAGE_DONE_FILE = ROOT / 'scripts' / 'reconcile-triage-done.json'
SCOPE_DONE_FILE = ROOT / 'scripts' / 'reconcile-triage-done-scope.json'
TRIAGE_BUCKETS = ['本版可做', '需协议', '需素材', '需宿主能力', '取证口径']

TRIAGE_MATERIAL_WORDS = (
    '皮肤', '缩略图', '素材', '封面', '波形', '预览图', '用图片', '图片资源', '图标资源',
)
TRIAGE_MATERIAL_RE = re.compile('|'.join(TRIAGE_MATERIAL_WORDS))

TRIAGE_HOST_WORDS = (
    '文件选择器', '目录选择器', '系统对话框', '托盘', '系统通知', '默认应用', 'Finder',
    '全屏', '多显示器', '系统级', '开机自启', '休眠', '日志文件', '伪终端', '自动安装',
    '麦克风', '摄像头',
)
TRIAGE_HOST_RE = re.compile('|'.join(TRIAGE_HOST_WORDS))
# 通用（桌面设置）、通知（系统通知）、运行环境（下载安装）、关于（自动安装）：都要宿主通道。
TRIAGE_HOST_FAMILIES = {'S-general', 'S-notifications', 'S-runtime', 'S-about'}

# 事实、写入或清单在 App Server、protocol v3 没有对应方法或字段的元素族。
TRIAGE_PROTOCOL_FAMILIES = {
    'S-account', 'S-usage', 'S-personalization', 'S-modality', 'S-mcp', 'S-git', 'S-hooks',
    'S-pluginSettings', 'S-externalAgents',
}

# ---- 附录：主窗口与全局面板的五档分流（§6）。口径与 §5 同源，只把「需素材」
# 换成「需事实」：这两份需求没有皮肤/宠物素材，缺的是 App Server 已给、Go 侧
# 尚未投影的事实（当前只有「Agent 配置」一项：session.agent.get/update 已存在，
# Source 还没有对应读数）。

SCOPE_TRIAGE_BUCKETS = ['本版可做', '需协议', '需事实', '需宿主', '取证口径']
SCOPE_REQ_FILES = ('requirements-main.md', 'requirements-global.md')

SCOPE_HOST_WORDS = (
    '保存对话框', '保存面板', '系统保存', '下载', '安装包', '重启安装', '浏览器数据',
    'Chrome', '系统通知', '窗口控件', '窗口拖动', '拖拽窗口', '红绿灯', '全屏',
    '多显示器', '系统级', '托盘',
)
SCOPE_HOST_RE = re.compile('|'.join(SCOPE_HOST_WORDS))
SCOPE_HOST_DIMS = {'多窗口', '拖拽', '输入法'}

# 协议 v3 没有对应方法/字段的元素族：成员审批明细、沙箱越权卡、子代理模型卡、
# 会话 Fork 副本、市场安装、浏览器数据导入/清理；以及左栏里只存在于参考产品
# 会话目录、protocol 不投影的未读点、改动摘要、定时来源、逐会话分支。
SCOPE_PROTOCOL_FAMILIES = {
    'LB-67', 'LB-70', 'LB-74', 'LB-76', 'LB-81', 'LB-82',
    'G-C2', 'G-C3', 'G-G2', 'G-H2', 'G-H3', 'G-H6', 'G-H7',
}
SCOPE_PROTOCOL_WORDS = (
    '未读', '改动摘要', '定时来源', '成员', '审批历史', '会话复制', '快照过期',
    '插件市场', '市场安装', '下载进度', 'Cookie', '电脑操作', '沙箱',
    '子代理模型', '分支不一致',
)
SCOPE_FACT_WORDS = ('Agent 配置', '多套 Agent 配置')


# 横向维度：判定在屏根 / 主题 / App 层（§4.3、§4.8），元素函数里不出现标记。
TRIAGE_HORIZONTAL_DIMS = {
    '窄窗', '深色', '暗色', '记忆', '高DPI', '键盘', '动画', '性能', '输入法', '多窗口',
    '拖拽', '数据量', '密度', '空间', '位置', '悬停', '展开后', '语言', '多语言',
}

DIM_CONTENT = '__content__'

# 维度 → 标记组。primary = 领导给定的八组；ext = 为其余维度补的同类组
# （报告里单列，便于后续收紧）。
DIM_MARKERS: dict[str, tuple[str, str]] = {
    '交互': (r'Clicked|Pressed|Key|OnSubmit|Menu', 'primary'),
    '无障碍': (r'Label|Role|AccessHidden|Aria', 'primary'),
    '样式': (r'TextColor|Radius|Padding|Width|Height|Font|Type', 'primary'),
    '异常': (r'(?i)err|错误|失败|重试|Reason', 'primary'),
    '记忆': (r'(?i)prefs|persist|View\.', 'primary'),
    '持久化': (r'(?i)prefs|persist|View\.', 'primary'),
    '动画': (r'Animate|Duration|ease|Ease', 'primary'),
    '窄窗': (r'narrow|Narrow|Compact|compact', 'primary'),
    '内容': (DIM_CONTENT, 'primary'),
    '文案': (DIM_CONTENT, 'primary'),
    '状态': (r'State|Disabled|Enabled|Visible|Hidden|Loading|Invalidate', 'ext'),
    '关联': (r'Switch|Navigate|Screen|Section|Tab|Select|Open|Show', 'ext'),
    '深色': (r'Theme|Dark|Light', 'ext'),
    '暗色': (r'Theme|Dark|Light', 'ext'),
    '键盘': (r'Key|Shortcut|Escape|Enter|Cmd|Ctrl|Tab', 'ext'),
    '高DPI': (r'Scale|Dpi|DPI|Px|Pixel', 'ext'),
    '性能': (r'Duration|Throttle|Debounce|Cache|time\.', 'ext'),
    '语言': (DIM_CONTENT, 'ext'),
    '多语言': (DIM_CONTENT, 'ext'),
    '输入法': (r'IME|Composition|Compose|Marked', 'ext'),
    '多窗口': (r'Window', 'ext'),
    '位置': (r'Row|Column|Align|Gap|Padding|Place|Offset', 'ext'),
    '拖拽': (r'Drag|Drop', 'ext'),
    '密度': (r'spaceUnit|Density|Compact|Gap', 'ext'),
    '悬停': (r'Hover|Hovered', 'ext'),
    '展开后': (r'Expand|Collapsed|Visible', 'ext'),
    '数据量': (r'Limit|Count|len\(|Max', 'ext'),
    '空间': (r'spaceUnit|Width|Height|Gap', 'ext'),
}

# 同一维度在 mygo 里的其它写法：用来量「无标记」里有多少是标记组漏词，
# 只作复核参考，不改变判定。
DIM_ADJACENT: dict[str, str] = {
    '交互': r'ButtonBase|Hovered|OnClick|Chosen|Selectable|OnKey|OnHover',
    '样式': (r'Background|BorderColor|BorderWidth|Opacity|Gap|GapX|GapY|AlignItems|'
             r'Justify|Margin|Stroke|Fill|MinWidth|MaxWidth|Size|Level'),
    '无障碍': r'Access\(|Tooltip|Level|Hint|DescribedBy|Focusable',
    '异常': r'Alert|Banner|Toast|Warn|Notice|Fallback',
    '记忆': r'Screen\.\w+|\.\w*Open =|lastPrefs|prefs|persist',
    '窄窗': r'Wrap|MinWidth|MaxWidth|Width|Size',
    '动画': r'After|Opacity|Fade|Spin',
    '键盘': r'Shortcut|TabStop|Focusable|Focus',
    '高DPI': r'Scale|Round|Size',
    '深色': r'TextMuted|Surface|Border|Background',
    '状态': r'Disabled|Visible|Hidden|Empty|Shown',
    '关联': r'Switch|Open|Close|Section|Screen',
}


def elem_family(elem_id: str) -> str:
    """元素族：settings（S-<节>-NN）/ panel（单字母-NN，右栏与时间线）/ other。"""
    if elem_id.startswith('S-') and elem_id.count('-') >= 2:
        sec = elem_id.split('-')[1]
        if not sec.isdigit():
            return 'settings'
    if PANEL_ELEM_RE.match(elem_id):
        return 'panel'
    return 'other'


def family_ok(elem_id: str, rel: str) -> bool:
    """跨域误命中过滤：设置元素只认设置屏；单字母面板元素不认设置屏。"""
    name = rel.rsplit('/', 1)[-1]
    fam = elem_family(elem_id)
    if fam == 'settings':
        return name.startswith('settings')
    if fam == 'panel':
        return not name.startswith('settings')
    return True


def _pad_elem(base: str, num: int, width: int) -> str:
    return f'{base}-{num:0{width}d}' if width >= 2 else f'{base}-{num}'


def build_elem_aliases(known: set[str]) -> dict[str, str]:
    """全局面板的短写：`G-A1-03` 在代码里常写成 `A1-03`；`S-accounts-12` 同理。"""
    aliases: dict[str, str] = {}
    dup: set[str] = set()
    for eid in known:
        if eid.startswith('G-'):
            short = eid[2:]
        elif re.match(r'^S-[A-Za-z][A-Za-z0-9]*-', eid):
            short = eid[2:]
        else:
            continue
        if short in aliases and aliases[short] != eid:
            dup.add(short)
        aliases[short] = eid
    for short in dup:
        aliases.pop(short, None)
    return aliases


def expand_element_ids(text: str, known: set[str], aliases: dict[str, str] | None = None,
                       ) -> list[str]:
    """从一段注释里抽元素 ID：原文写法 + 区间/并列写法展开 + 短写还原，只留已知 ID。"""
    out: list[str] = []
    short_map = aliases or {}

    def add(eid: str) -> None:
        eid = short_map.get(eid, eid)
        if eid in known and eid not in out:
            out.append(eid)

    for m in ID_SEQ_RE.finditer(text):
        base, num_s = m.group('base'), m.group('num')
        width, cur_num = len(num_s), int(num_s)
        add(_pad_elem(base, cur_num, width))
        for c in ID_CONT_RE.finditer(m.group('rest') or ''):
            sep, b, n = c.group('sep'), c.group('base'), int(c.group('num'))
            if sep in '–—~～' or sep in ('...', '…'):
                if b is not None:
                    base, width = b, len(c.group('num'))
                if cur_num <= n <= cur_num + 200:
                    for k in range(cur_num + 1, n + 1):
                        add(_pad_elem(base, k, width))
                cur_num = n
            else:
                base, width = (b or base), len(c.group('num'))
                add(_pad_elem(base, n, width))
                cur_num = n
    for tok in TOKEN_ID_RE.findall(text):
        add(tok)
    return out


@dataclasses.dataclass(frozen=True)
class MapPoint:
    rel: str
    func: str
    source: str          # doc（函数 doc 注释）/ body（函数体内注释）/ symbol（清单符号）
    span: int = 0        # 同一处注释里出现的元素个数：越小越具体

    def label(self) -> str:
        return f'{self.rel}:{self.func}'


def doc_comment_above(gf: GoFile, line_no: int) -> str:
    """函数声明正上方紧邻的 // 注释块（Go doc 注释）。"""
    out: list[str] = []
    k = line_no - 1
    while k >= 1:
        s = gf.lines[k - 1].strip()
        if s.startswith('//'):
            out.append(s[2:])
            k -= 1
            continue
        break
    return '\n'.join(reversed(out))


def build_element_render_index(inv: dict[str, ElementInfo], known_ids: set[str]
                               ) -> dict[str, list[MapPoint]]:
    """元素 ID → 候选渲染点（doc 注释 → 体内注释 → 清单符号，去重后每条最多 8 个）。"""
    idx: dict[str, list[MapPoint]] = collections.defaultdict(list)
    seen: dict[str, set[tuple[str, str]]] = collections.defaultdict(set)
    aliases = build_elem_aliases(known_ids)

    def add(elem_id: str, point: MapPoint) -> None:
        if len(idx[elem_id]) >= 8:
            return
        key = (point.rel, point.func)
        if key in seen[elem_id]:
            return
        seen[elem_id].add(key)
        idx[elem_id].append(point)

    for gf in IMPL:
        for f in gf.funcs:
            doc = doc_comment_above(gf, f.start)
            doc_ids = expand_element_ids(doc, known_ids, aliases)
            for eid in doc_ids:
                if family_ok(eid, gf.rel):
                    add(eid, MapPoint(gf.rel, f.name, 'doc', len(doc_ids)))
            body_comments = [text for text, line in gf.comments if f.start <= line <= f.end]
            if body_comments:
                body_ids = expand_element_ids('\n'.join(body_comments), known_ids, aliases)
                for eid in body_ids:
                    if family_ok(eid, gf.rel):
                        add(eid, MapPoint(gf.rel, f.name, 'body', len(body_ids)))

    for elem_id, info in inv.items():
        if elem_id not in known_ids:
            continue
        for sym in info.symbols:
            hit = SYMBOL_INDEX.get(sym)
            if not hit:
                continue
            rel, func, _line = hit
            if func == '包级':
                continue
            if family_ok(elem_id, rel):
                add(elem_id, MapPoint(rel, func, 'symbol'))

    order = {'doc': 0, 'body': 1, 'symbol': 2}
    for elem_id in idx:
        idx[elem_id].sort(key=lambda p: (order.get(p.source, 3), p.span))
    return dict(idx)


# 函数体检索文本缓存：(rel, func) → (代码 + 字面量, 代码)。字符串已被 scan_go
# 从代码行里抹掉，字面量另取，避免同一段文案被算两次。
_FUNC_TEXT: dict[tuple[str, str], tuple[str, str]] = {}
_MARK_HITS: dict[tuple[str, str, str], tuple[str, str]] = {}
_ADJ_HITS: dict[tuple[str, str, str], str] = {}


def func_text(gf: GoFile, name: str) -> tuple[str, str]:
    key = (gf.rel, name)
    cached = _FUNC_TEXT.get(key)
    if cached is not None:
        return cached
    for f in gf.funcs:
        if f.name != name:
            continue
        code = '\n'.join(gf.code_lines[f.start - 1:f.end])
        lits = '\n'.join(c for c, line in gf.literals if f.start <= line <= f.end)
        val = (code + '\n' + lits, code)
        _FUNC_TEXT[key] = val
        return val
    _FUNC_TEXT[key] = ('', '')
    return _FUNC_TEXT[key]


def _content_marker(text: str) -> str:
    lits = [ln.strip() for ln in text.splitlines() if CJK_RE.search(ln)]
    if not lits:
        return ''
    shown = lits[0][:40]
    if len(lits) > 1:
        return f'文案「{shown}」等 {len(lits)} 条'
    return f'文案「{shown}」'


# 包级常量/变量的中文字面量：渲染函数通常只写 `Text(c, usCopyTitle)`，文案在常量里。
_DECL_LITS: dict[str, dict[str, list[str]]] = {}


def decl_literals(gf: GoFile) -> dict[str, list[str]]:
    cached = _DECL_LITS.get(gf.rel)
    if cached is not None:
        return cached
    out: dict[str, list[str]] = collections.defaultdict(list)
    for content, line in gf.literals:
        if not CJK_RE.search(content) or gf.func_at(line) != '包级':
            continue
        name = gf.func_or_decl_at(line)
        if name == '包级':
            continue
        out[name].append(content.strip())
    _DECL_LITS[gf.rel] = dict(out)
    return _DECL_LITS[gf.rel]


def content_marker(gf: GoFile, name: str) -> str:
    """内容档：函数自己的字面量优先，再看它引用到的包级文案常量。"""
    text, code = func_text(gf, name)
    marker = _content_marker(text)
    if marker:
        return marker
    lits = decl_literals(gf)
    used = [t for t in dict.fromkeys(IDENT_RE.findall(code)) if t in lits]
    extra = '\n'.join(lit for t in used for lit in lits[t][:3])
    return _content_marker(extra)


def marker_in_func(gf: GoFile, name: str, dim: str) -> tuple[str, str] | None:
    """在函数体（含一跳被调函数）里找该维度的标记 → (标记, 命中函数)。"""
    key = (gf.rel, name, dim)
    cached = _MARK_HITS.get(key)
    if cached is not None:
        return cached if cached[0] else None
    sql, _src = DIM_MARKERS.get(dim, (DIM_CONTENT, 'ext'))
    text, code = func_text(gf, name)
    found: tuple[str, str] | None = None
    if text:
        if sql == DIM_CONTENT:
            marker = content_marker(gf, name)
            if marker:
                found = (marker, name)
        else:
            m = re.search(sql, text)
            if m:
                found = (m.group(0).strip(), name)
        if found is None:
            # 一跳：同包里被本函数调用的辅助函数（方法按裸函数名解析）
            bare = name.split('.')[-1]
            for callee in dict.fromkeys(CALL_RE.findall(code)):
                if callee == bare:
                    continue
                target = resolve_callee(gf, callee)
                if target is None:
                    continue
                cgf, cname = target
                ctext, _ = func_text(cgf, cname)
                if not ctext:
                    continue
                if sql == DIM_CONTENT:
                    marker = content_marker(cgf, cname)
                    if marker:
                        found = (marker, cname)
                        break
                else:
                    m = re.search(sql, ctext)
                    if m:
                        found = (m.group(0).strip(), cname)
                        break
    _MARK_HITS[key] = found or ('', '')
    return found


def adjacent_marker(gf: GoFile, name: str, dim: str) -> str:
    """标记组之外、同维度的其它写法（只用于量「无标记」里的漏词）。"""
    key = (gf.rel, name, dim)
    cached = _ADJ_HITS.get(key)
    if cached is not None:
        return cached
    sql = DIM_ADJACENT.get(dim)
    if not sql:
        _ADJ_HITS[key] = ''
        return ''
    text, code = func_text(gf, name)
    hit = ''
    if text:
        m = re.search(sql, code)
        if m:
            hit = m.group(0)
        else:
            bare = name.split('.')[-1]
            for callee in dict.fromkeys(CALL_RE.findall(code)):
                if callee == bare:
                    continue
                target = resolve_callee(gf, callee)
                if target is None:
                    continue
                ctext, _ = func_text(target[0], target[1])
                m = re.search(sql, ctext) if ctext else None
                if m:
                    hit = f'{m.group(0)}@{target[1]}'
                    break
    _ADJ_HITS[key] = hit
    return hit


@dataclasses.dataclass
class MapResult:

    kind: str
    text: str
    point: MapPoint | None = None
    marker: str = ''
    marker_func: str = ''
    note: str = ''


def norm_dim(dim: str) -> str:
    return re.split(r'[（(]', dim.strip())[0].strip()


def judge_mapped(row: ReqRow, inv: dict[str, ElementInfo],
                 index: dict[str, list[MapPoint]], first: Verdict) -> MapResult:
    """第二轮：逐字 / 负向沿用第一轮证据，其余按元素映射 + 维度标记判。"""
    if first.kind == NEG_LABEL:
        return MapResult(MAP_NEG, first.text)
    if first.kind == '逐字':
        return MapResult('逐字', first.text, note='逐字')

    dim = norm_dim(row.dim)
    points = index.get(row.elem_id, [])
    test_hint = ''
    esc = TEST_INDEX.get(row.elem_id)
    if esc:
        test_hint = f'；测试提及 {esc[0]}:{esc[1]}'

    if not points:
        return MapResult(MAP_NO_POINT, f'未实现（找不到元素渲染点）{test_hint}')

    for point in points:
        gf = GO_BY_REL.get(point.rel)
        if gf is None:
            continue
        hit = marker_in_func(gf, point.func, dim)
        if hit:
            marker, where = hit
            via = '' if where == point.func else f'@{where}'
            return MapResult(
                MAP_HIT,
                f'已实现（映射 = {row.elem_id}→{point.label()}；标记 = {marker}{via}）',
                point=point, marker=marker, marker_func=where,
            )

    point = points[0]
    return MapResult(
        MAP_NO_MARK,
        f'未实现（映射到位但没有该维度的标记）：{row.elem_id}→{point.label()}{test_hint}',
        point=point,
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
                 verdicts_by_file: dict[Path, dict[int, Verdict]],
                 map_by_file: dict[Path, dict[int, MapResult]],
                 inv: dict[str, ElementInfo],
                 render_index: dict[str, list[MapPoint]]) -> None:
    all_verdicts: list[Verdict] = []
    for rows in rows_by_file.values():
        all_verdicts.extend(verdicts_by_file[rows[0].path].values())
    total = len(all_verdicts)

    lines: list[str] = []
    lines.append('# 需求实现状态：机械对账报告')
    lines.append('')
    lines.append('Status: 对账报告（脚本生成，可重跑；`python3 scripts/reconcile.py`）')
    lines.append('Scope: 四份需求表的全部条目（`docs/domains/requirements/requirements-*.md`）；'
                 '状态列写回的是 §4「按元素映射判定」的结论，§1–§3 是第一轮记录，'
                 '§5 是设置 19 节的五档分流与本轮做掉的条目，§6 是主窗口与全局面板的'
                 '五档分流（核对状态列，不看口头说明）')
    lines.append('Owner: 组织 agent（验收判定侧）；需求文件与元素清单归 R 与需求分析领域')
    lines.append('')
    lines.append('## 0. 判定口径（第一轮：逐字/符号/测试）')
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
    lines.append('## 1. 第一轮命中分布（每份；状态列已由 §4 覆盖）')
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
    lines.append('## 2. 第一轮随机 %d 条复核' % SAMPLE_SIZE)
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
    lines.append('## 3. 第一轮已知局限')
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
    append_map_section(lines, rows_by_file, verdicts_by_file, map_by_file, inv, render_index)
    append_triage_section(lines, rows_by_file, map_by_file)
    append_scope_triage_section(lines, rows_by_file, map_by_file)
    REPORT.write_text('\n'.join(lines))


# ---------------------------------------------------------------- 第二轮报告

def load_map_reviews() -> dict[str, dict]:
    """人工复核结论（requirement ID → {verdict, note}），没有该文件就留「待复核」。"""
    if not MAP_REVIEW_FILE.exists():
        return {}
    try:
        data = json.loads(MAP_REVIEW_FILE.read_text())
    except (OSError, ValueError):
        return {}
    reviews = data.get('reviews', {}) if isinstance(data, dict) else {}
    return reviews if isinstance(reviews, dict) else {}


def signal_list(gf: GoFile, name: str, limit: int = 8) -> str:
    """函数体直接调用到的构件（`Text`、`TextColor` 这类），供人工复核时判断维度。"""
    _text, code = func_text(gf, name)
    names = list(dict.fromkeys(re.findall(r'\.([A-Z]\w+)\(', code)))
    return ', '.join(names[:limit]) if names else '—'


def pick_map_sample(rows: list[ReqRow], results: dict[int, MapResult],
                    per_file: int = 10) -> list[ReqRow]:
    """第二轮抽样：优先「映射到位但没有标记」，其次命中，最后无渲染点。"""
    rng = random.Random(MAP_SAMPLE_SEED)
    by_kind: dict[str, list[ReqRow]] = collections.defaultdict(list)
    for row in rows:
        by_kind[results[row.line_no].kind].append(row)
    picked: list[ReqRow] = []
    chosen: set[int] = set()
    for kind, n in ((MAP_NO_MARK, 7), (MAP_HIT, 2), (MAP_NO_POINT, 1)):
        pool = by_kind.get(kind, [])
        for row in rng.sample(pool, min(n, len(pool))):
            chosen.add(row.line_no)
            picked.append(row)
    rest = [r for r in rows if r.line_no not in chosen]
    if len(picked) < per_file:
        for row in rng.sample(rest, min(per_file - len(picked), len(rest))):
            picked.append(row)
    return picked


_DIR_FUNCS: dict[str, dict[str, tuple[GoFile, str]]] = {}


def resolve_callee(gf: GoFile, bare: str) -> tuple[GoFile, str] | None:
    """按裸函数名解析被调函数（同目录、同包；方法名去掉接收者）。"""
    d = gf.rel.rsplit('/', 1)[0] if '/' in gf.rel else ''
    table = _DIR_FUNCS.get(d)
    if table is None:
        table = {}
        for other in IMPL:
            od = other.rel.rsplit('/', 1)[0] if '/' in other.rel else ''
            if od != d:
                continue
            for f in other.funcs:
                table.setdefault(f.name.split('.')[-1], (other, f.name))
        _DIR_FUNCS[d] = table
    return table.get(bare)


def _lit_in_func(gf: GoFile, name: str, lit: str) -> bool:
    text, code = func_text(gf, name)
    if text and lit in text:
        return True
    lits = decl_literals(gf)
    for ident in set(IDENT_RE.findall(code)):
        for content in lits.get(ident, []):
            if lit in content:
                return True
    return False


def func_reaches_literal(gf: GoFile, name: str, lit: str) -> bool:
    """函数体（含引用到的包级文案常量、一跳同包被调函数）里是否出现该字面量。"""
    if _lit_in_func(gf, name, lit):
        return True
    _text, code = func_text(gf, name)
    for callee in dict.fromkeys(CALL_RE.findall(code)):
        if callee == name.split('.')[-1]:
            continue
        target = resolve_callee(gf, callee)
        if target is None:
            continue
        if _lit_in_func(target[0], target[1], lit):
            return True
    return False


def literal_mapping_state(row: ReqRow, first: Verdict,
                          render_index: dict[str, list[MapPoint]]) -> str:
    """逐字档的映射一致性：字面量是否落在该元素的映射函数体（含一跳）内。"""
    points = render_index.get(row.elem_id, [])
    if not points:
        return 'unmapped'
    ev = first.evidence
    if not ev:
        return 'inconsistent'
    lit = ev[2]
    for point in points:
        gf = GO_BY_REL.get(point.rel)
        if gf is None:
            continue
        if func_reaches_literal(gf, point.func, lit):
            return 'consistent'
    return 'inconsistent'


# ---------------------------------------------------------------- 主流程

def append_map_section(lines: list[str],
                       rows_by_file: dict[Path, list[ReqRow]],
                       verdicts_by_file: dict[Path, dict[int, Verdict]],
                       map_by_file: dict[Path, dict[int, MapResult]],
                       inv: dict[str, ElementInfo],
                       render_index: dict[str, list[MapPoint]]) -> None:
    all_rows = [row for rows in rows_by_file.values() for row in rows]
    all_results = [map_by_file[row.path][row.line_no] for row in all_rows]
    total = len(all_results)
    counts = collections.Counter(r.kind for r in all_results)
    judged = total - counts['逐字'] - counts[MAP_NEG]

    lines.append('## 4. 第二轮：按元素映射判定')
    lines.append('')
    lines.append('第一轮把「期望列抽不出字面量」的叙述型需求压成「未实现」，那是下限，不是结论。'
                 '第二轮改按**元素 → 渲染函数 → 维度标记**判。')
    lines.append('')
    lines.append('### 4.1 判定口径')
    lines.append('')
    lines.append('| 判定 | 条件 | 状态列写法 |')
    lines.append('|---|---|---|')
    lines.append('| 逐字 | 第一轮逐字命中（不重判） | 沿用 `已实现（逐字）：…` |')
    lines.append('| 映射命中 | 元素映射到渲染函数，函数体内有该维度的标记 | '
                 '`已实现（映射 = 元素→文件:函数；标记 = 标记）` |')
    lines.append('| 映射到位无标记 | 映射到位，函数体内没有该维度的标记 | '
                 '`未实现（映射到位但没有该维度的标记）：元素→文件:函数` |')
    lines.append('| 无渲染点 | 元素 ID 在 `internal/**/*.go` 的注释里反查不到渲染函数 | '
                 '`未实现（找不到元素渲染点）` |')
    lines.append('| 负向 | 第一轮负向断言（不重判） | 沿用 `需负向取证：…` |')
    lines.append('')
    lines.append('映射来源（按优先级）：函数 doc 注释里的元素 ID（同一处注释覆盖的元素越少越优先）→ '
                 '函数体内注释里的元素 ID → 清单 `symbol` 派生的 Go 标识符。注释里的 '
                 '`S-account-02–07`、`S-account-10/11/13`、`TB-20–TB-30`、`S-about-01…18` '
                 '按区间/并列展开，`A1-03` 这类短写还原成 `G-A1-03`，展开结果必须落在已知元素 ID 集合里。'
                 '跨域误命中按元素族过滤：设置元素只认 `settings*.go`，单字母面板元素不认设置屏。')
    lines.append('')
    lines.append('函数体 = 去注释、去字符串的代码行 + 函数区间内的字符串字面量，'
                 '并向下展开一跳同包（同目录）被调函数，方法按裸函数名解析'
                 '（`a.rpGroup(` 能解析到 `App.rpGroup`）；命中记 `标记 = X@被调函数`。')
    lines.append('')
    lines.append('| 维度 | 标记组 | 来源 |')
    lines.append('|---|---|---|')
    for dim in sorted(DIM_MARKERS):
        sql, src = DIM_MARKERS[dim]
        shown = '函数内中文字面量' if sql == DIM_CONTENT else f'`{sql}`'
        lines.append(f'| {dim} | {shown} | {"领导给定" if src == "primary" else "本轮补充"} |')
    lines.append('')
    lines.append('未列出的维度按「内容」判（函数内中文字面量）。八组之外的维度用补充标记组，'
                 '因为需求里出现的维度只有一半落在八组内；记忆/持久化与异常两组按代码习惯写成'
                 '大小写不敏感（`a.persistPreferences()`、`ErrMsg`），表里给的是实际正则。')
    lines.append('')

    lines.append('### 4.2 分布')
    lines.append('')
    lines.append('| 文件 | 总条数 | 逐字 | 映射命中 | 映射到位无标记 | 无渲染点 | 负向 | 映射命中占比 |')
    lines.append('|---|---:|---:|---:|---:|---:|---:|---:|')
    for path, rows in rows_by_file.items():
        c = collections.Counter(map_by_file[path][r.line_no].kind for r in rows)
        lines.append(
            f'| `{path.name}` | {len(rows)} | {c["逐字"]} | {c[MAP_HIT]} | '
            f'{c[MAP_NO_MARK]} | {c[MAP_NO_POINT]} | {c[MAP_NEG]} | '
            f'{pct(c[MAP_HIT], len(rows))} |')
    lines.append(
        f'| **合计** | **{total}** | **{counts["逐字"]}** | **{counts[MAP_HIT]}** | '
        f'**{counts[MAP_NO_MARK]}** | **{counts[MAP_NO_POINT]}** | **{counts[MAP_NEG]}** | '
        f'**{pct(counts[MAP_HIT], total)}** |')
    lines.append('')
    mapped_rows = counts[MAP_HIT] + counts[MAP_NO_MARK]
    lines.append(f'去掉逐字与负向后可判 {judged} 条：元素映射到位 {mapped_rows} 条'
                 f'（{pct(mapped_rows, judged)}），其中缺该维度标记 {counts[MAP_NO_MARK]} 条；'
                 f'找不到渲染点 {counts[MAP_NO_POINT]} 条（{pct(counts[MAP_NO_POINT], judged)}）。')
    lines.append('')
    lines.append('「无渲染点」是映射的缺口，不是实现的结论：元素 ID 没在 Go 注释里标注，'
                 '就查不到渲染函数。按域看元素映射覆盖率：')
    lines.append('')
    lines.append('| 文件 | 元素数 | 有渲染点 | 覆盖率 |')
    lines.append('|---|---:|---:|---:|')
    for path, rows in rows_by_file.items():
        ids = {r.elem_id for r in rows}
        mapped = sum(1 for e in ids if e in render_index)
        lines.append(f'| `{path.name}` | {len(ids)} | {mapped} | {pct(mapped, len(ids))} |')
    lines.append('')
    miss_groups: collections.Counter = collections.Counter()
    for row in all_rows:
        if row.elem_id in render_index:
            continue
        parts = row.elem_id.split('-')
        head = '-'.join(parts[:2]) if parts[0] in ('S', 'G') and len(parts) >= 3 else parts[0]
        miss_groups[head] += 1
    if miss_groups:
        top = '、'.join(f'`{k}` {n} 条' for k, n in miss_groups.most_common(8))
        lines.append(f'缺标注最多的元素族（按条数）：{top}。')
        lines.append('')

    lines.append('### 4.3 按维度')
    lines.append('')
    lines.append('| 维度 | 条数 | 逐字 | 映射命中 | 映射到位无标记 | 无渲染点 | 负向 | 标记组来源 |')
    lines.append('|---|---:|---:|---:|---:|---:|---:|---|')
    dim_counts: dict[str, collections.Counter] = collections.defaultdict(collections.Counter)
    for row, result in zip(all_rows, all_results):
        dim_counts[norm_dim(row.dim)][result.kind] += 1
    for dim, c in sorted(dim_counts.items(), key=lambda kv: -sum(kv[1].values())):
        _sql, src = DIM_MARKERS.get(dim, (DIM_CONTENT, 'none'))
        source = '领导给定' if src == 'primary' else ('本轮补充' if src == 'ext' else '未定义（按内容）')
        lines.append(f'| {dim} | {sum(c.values())} | {c["逐字"]} | {c[MAP_HIT]} | '
                     f'{c[MAP_NO_MARK]} | {c[MAP_NO_POINT]} | {c[MAP_NEG]} | {source} |')
    lines.append('')
    lines.append('**横向维度的实现层级在元素函数之上**：窄窗、深色、记忆、高 DPI、键盘、动画这一类，'
                 '代码把判定放在屏根 / `App` / 主题层（如 `isCompact(width)`、`persistViewState`、'
                 '主题 token），元素函数本身不出现标记。这些维度的「映射到位但没有标记」'
                 '多数是层级问题，不是元素没实现；要取证得看屏根函数或界面实拍。')
    lines.append('')

    lines.append('### 4.4 逐字档的映射一致性')
    lines.append('')
    cons = collections.Counter()
    inconsistent: list[tuple[ReqRow, Verdict]] = []
    for path, rows in rows_by_file.items():
        for row in rows:
            first = verdicts_by_file[path][row.line_no]
            if first.kind != '逐字':
                continue
            state = literal_mapping_state(row, first, render_index)
            cons[state] += 1
            if state == 'inconsistent':
                inconsistent.append((row, first))
    lines.append(f'逐字 {sum(cons.values())} 条中：字面量落在元素映射函数内 {cons["consistent"]} 条；'
                 f'元素有映射但字面量不在映射函数内 {cons["inconsistent"]} 条；'
                 f'元素没有映射点 {cons["unmapped"]} 条。')
    lines.append('')
    if inconsistent:
        lines.append('映射不一致的前 10 条（逐字证据可能指错实现点）：')
        lines.append('')
        lines.append('| 需求 ID | 元素 ID | 逐字证据 | 元素映射 |')
        lines.append('|---|---|---|---|')
        for row, first in inconsistent[:10]:
            ev = first.evidence or ('—', '—', '—')
            point = render_index.get(row.elem_id, [None])[0]
            where = point.label() if point else '—（无映射点）'
            lines.append(f'| {row.req_id} | {row.elem_id} | `{ev[0]}:{ev[1]}`「{ev[2]}」 | `{where}` |')
        lines.append('')

    append_map_review(lines, rows_by_file, map_by_file, inv, render_index)


def append_map_review(lines: list[str],
                      rows_by_file: dict[Path, list[ReqRow]],
                      map_by_file: dict[Path, dict[int, MapResult]],
                      inv: dict[str, ElementInfo],
                      render_index: dict[str, list[MapPoint]]) -> None:
    reviews = load_map_reviews()
    all_rows = [row for rows in rows_by_file.values() for row in rows]
    all_results = [map_by_file[row.path][row.line_no] for row in all_rows]
    lines.append('### 4.5 随机 %d 条复核（重点：映射到位但没有标记）' % SAMPLE_SIZE)
    lines.append('')
    lines.append(f'抽样种子 `{MAP_SAMPLE_SEED}`，每份 10 条 = 7 条「映射到位无标记」+ 2 条「映射命中」'
                 '+ 1 条「无渲染点」。人工结论存 `scripts/reconcile-review-mapping.json`'
                 '（需求 ID → 结论），脚本重跑时自动带出；「函数体信号」是该函数直接调用到的构件。')
    lines.append('')
    agree = reviewed = 0
    cat_count: collections.Counter = collections.Counter()
    per_file = SAMPLE_SIZE // len(rows_by_file)
    for path, rows in rows_by_file.items():
        sample = pick_map_sample(rows, map_by_file[path], per_file)
        lines.append(f'#### {path.name}')
        lines.append('')
        lines.append('| 需求 ID | 元素 ID | 维度 | 判定 | 映射 | 函数体信号 | 人工复核 |')
        lines.append('|---|---|---|---|---|---|---|')
        for row in sample:
            result = map_by_file[path][row.line_no]
            where = f'`{result.point.label()}`' if result.point else '—'
            if result.marker:
                where += f' 标记 {result.marker}'
            if result.point:
                gf = GO_BY_REL.get(result.point.rel)
                signals = signal_list(gf, result.point.func) if gf else '—'
            else:
                signals = '—'
            note = reviews.get(row.req_id)
            if note:
                reviewed += 1
                cat = note.get('category', '')
                verdict = note.get('verdict', '?')
                if verdict == '确认':
                    agree += 1
                if cat:
                    cat_count[cat] += 1
                mark = f'{verdict}（{cat}）：{note.get("note", "")}' if cat else \
                    f'{verdict}：{note.get("note", "")}'
            else:
                mark = '待复核'
            lines.append(f'| {row.req_id} | {row.elem_id} | {norm_dim(row.dim)} | {result.kind} | '
                         f'{where} | {signals} | {mark} |')
        lines.append('')
    if reviewed:
        lines.append(f'人工复核 {reviewed} 条：与机器判定一致 {agree} 条，有分歧 {reviewed - agree} 条。'
                     + ('分歧构成：' + '、'.join(f'{k} {n}' for k, n in cat_count.most_common()) + '。'
                        if cat_count else ''))
    else:
        lines.append('人工复核：待填（`scripts/reconcile-review-mapping.json` 未提供或未命中抽样）。')
    lines.append('')

    lines.append('### 4.6 「缺标记」里有多少是标记组漏词')
    lines.append('')
    lines.append('对「映射到位但没有该维度的标记」的条目，再查一遍同维度在 mygo 里的其它写法'
                 '（`DIM_ADJACENT`，只作参考、不改判定）。这类条目更像标记组不够宽，不像元素缺实现。')
    lines.append('')
    leak: dict[str, collections.Counter] = collections.defaultdict(collections.Counter)
    examples: dict[str, str] = {}
    for row, result in zip(all_rows, all_results):
        if result.kind != MAP_NO_MARK or result.point is None:
            continue
        gf = GO_BY_REL.get(result.point.rel)
        if gf is None:
            continue
        dim = norm_dim(row.dim)
        hit = adjacent_marker(gf, result.point.func, dim)
        if not hit:
            continue
        leak[dim]['漏词'] += 1
        examples.setdefault(dim, f'{row.req_id} `{hit}`')
    lines.append('| 维度 | 缺标记条数 | 其中函数体有同类写法 | 例子 |')
    lines.append('|---|---:|---:|---|')
    dim_counts: dict[str, collections.Counter] = collections.defaultdict(collections.Counter)
    for row in all_rows:
        dim_counts[norm_dim(row.dim)][map_by_file[row.path][row.line_no].kind] += 1
    for dim, c in sorted(dim_counts.items(), key=lambda kv: -kv[1][MAP_NO_MARK]):
        miss = c[MAP_NO_MARK]
        if not miss:
            continue
        n = leak.get(dim, collections.Counter())['漏词']
        lines.append(f'| {dim} | {miss} | {n}（{pct(n, miss)}） | {examples.get(dim, "—")} |')
    lines.append('')

    lines.append('### 4.7 最该先修的 10 个「映射到位但缺标记」元素')
    lines.append('')
    lines.append('排序用「真缺」＝映射到位、缺该维度的标记，且函数体里连同维度的其它写法'
                 '（§4.6 的 `DIM_ADJACENT`）也没有；「缺标记条数」是全量。'
                 '维度缺口列是该元素缺标记的维度构成（按条数）。')
    lines.append('')
    miss_by_elem: dict[str, collections.Counter] = collections.defaultdict(collections.Counter)
    real_by_elem: collections.Counter = collections.Counter()
    rows_by_elem: collections.Counter = collections.Counter()
    for row in all_rows:
        result = map_by_file[row.path][row.line_no]
        rows_by_elem[row.elem_id] += 1
        if result.kind != MAP_NO_MARK or result.point is None:
            continue
        dim = norm_dim(row.dim)
        miss_by_elem[row.elem_id][dim] += 1
        gf = GO_BY_REL.get(result.point.rel)
        if gf is not None and not adjacent_marker(gf, result.point.func, dim):
            real_by_elem[row.elem_id] += 1
    ranked = sorted(real_by_elem.items(), key=lambda kv: (-kv[1], kv[0]))[:10]
    lines.append('| # | 元素 ID | 名称 | 真缺条数 | 缺标记条数 | 覆盖需求 | 维度缺口 | 映射点 |')
    lines.append('|---:|---|---|---:|---:|---:|---|---|')
    for i, (elem_id, real) in enumerate(ranked, start=1):
        c = miss_by_elem.get(elem_id, collections.Counter())
        info = inv.get(elem_id)
        name = (info.name if info else '')[:24] or '—'
        dims = '、'.join(f'{d} {n}' for d, n in c.most_common(4))
        point = render_index.get(elem_id, [None])[0]
        where = f'`{point.label()}`' if point else '—'
        lines.append(f'| {i} | {elem_id} | {name} | {real} | {sum(c.values())} | '
                     f'{rows_by_elem[elem_id]} | {dims} | {where} |')
    lines.append('')

    lines.append('### 4.8 第二轮已知局限')
    lines.append('')
    lines.append('- **映射靠注释**：清单没有 `symbol` 列，映射实际依赖 Go 注释里的元素 ID；'
                 '注释没写的元素一律「无渲染点」，那是缺映射，不等于缺实现。')
    lines.append('- **一处注释覆盖多元素**：`S-account-02–07` 这类区间把整段元素挂到同一函数，'
                 '函数里的标记会算到区间内每个元素头上。')
    lines.append('- **「内容」几乎必命中**：渲染函数里只要有一个中文字面量就算命中，'
                 '所以内容档的「映射命中」只说明有文案，不说明是本元素那句。')
    lines.append('- **`err` 标记过宽**：异常档的 `(?i)err` 会匹配 `error`/`ErrMsg` 一类通用名字，'
                 '异常档命中偏松；异常档缺标记反而更可信。')
    lines.append('- **标记组窄于实现词汇**：样式档只认 `TextColor|Radius|Padding|Width|Height|Font|Type`，'
                 '用 `Background`/`BorderColor`/`Opacity`/`Gap`/`AlignItems` 写的样式会被判「无标记」。')
    lines.append('- **只展开一跳**：被调函数再往下调用的辅助函数不查，深层复用的标记会漏。')
    lines.append('- **横向维度按元素函数查**：窄窗 / 深色 / 记忆 / 高 DPI / 键盘的实现层级在屏根与主题层，'
                 '这类「映射到位但缺标记」不能当作元素缺陷读。')
    lines.append('- **按维度判，不看数值**：命中只说明函数里出现过该类标记，'
                 '不证明取值等于期望（字号、间距、颜色仍要界面取证）。')
    lines.append('- **逐字档不重判**：第一轮逐字证据可能指错实现点（§4.4 给了不一致条数与样例）。')
    lines.append('')


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
GO_BY_REL: dict[str, GoFile] = {}


# ---------------------------------------------------------------- 五档分流附录

def triage_done() -> dict[str, dict[str, str]]:
    """本轮做掉的需求登记（需求 ID → {note, test}），没有文件就是空表。

    登记项可以写成一句字符串（只有实现点），也可以写成 {note, test}：test 是证明
    这条行为的测试名，附录里一并列出，方便核对「做没做」。
    """
    if not TRIAGE_DONE_FILE.exists():
        return {}
    try:
        data = json.loads(TRIAGE_DONE_FILE.read_text())
    except (OSError, ValueError):
        return {}
    done = data.get('done', {}) if isinstance(data, dict) else {}
    if not isinstance(done, dict):
        return {}
    out: dict[str, dict[str, str]] = {}
    for req_id, value in done.items():
        if isinstance(value, dict):
            out[req_id] = {k: str(v) for k, v in value.items()}
        else:
            out[req_id] = {'note': str(value), 'test': ''}
    return out


def elem_family(elem_id: str) -> str:
    """元素族：`S-appearance-50` → `S-appearance`；`IN-66` → `IN`。"""
    parts = elem_id.split('-')
    if parts[0] == 'S' and len(parts) >= 3:
        return 'S-' + parts[1]
    return parts[0]


def elem_index(elem_id: str) -> int:
    """元素序号：`S-appearance-50` → 50；不是 `族-序号` 形状时返回 -1。"""
    tail = elem_id.rsplit('-', 1)[-1]
    return int(tail) if tail.isdigit() else -1


def elem_needs_assets(elem_id: str) -> bool:
    """皮肤（`S-appearance-50` 起）与宠物要图片素材；没有素材连形态都画不出来。"""
    family = elem_family(elem_id)
    if family == 'S-pets':
        return True
    return family == 'S-appearance' and elem_index(elem_id) >= 50


def triage_bucket(row: ReqRow, kind: str) -> tuple[str, str]:
    """把一条未实现分到五档之一，返回 (档位, 命中规则)。

    规则按顺序取第一个命中（口径见附录 §5.1）：
    素材 → 宿主 → 协议 → 取证口径 → 本版可做。
    前三档说的是「缺什么才能做」；取证口径说的是「这条判定本身要补证据」——
    无渲染点、横向维度、语言包缺失都属这一类，它们不指向一个待实现的功能。
    """
    blob = ' '.join((row.expected, row.acceptance, row.cells[3] if len(row.cells) > 3 else ''))
    family = elem_family(row.elem_id)
    if TRIAGE_MATERIAL_RE.search(row.expected) or elem_needs_assets(row.elem_id):
        return '需素材', '期望列素材词 / 皮肤·宠物元素'
    if TRIAGE_HOST_RE.search(blob) or family in TRIAGE_HOST_FAMILIES:
        return '需宿主能力', '宿主接口词 / 元素族'
    if family in TRIAGE_PROTOCOL_FAMILIES:
        return '需协议', '元素族在服务端'
    if kind == MAP_NO_POINT:
        return '取证口径', '无渲染点（先补锚点再判）'
    if norm_dim(row.dim) in TRIAGE_HORIZONTAL_DIMS:
        return '取证口径', '横向维度'
    if '语言包' in blob:
        return '取证口径', '无 i18n 层'
    return '本版可做', '本地偏好 / 本地草稿 / 已有事实'


def pick_triage_sample(rows: list[ReqRow], limit: int = 10) -> list[ReqRow]:
    """每档取代表性 10 条：先在元素族之间摊开，再按需求 ID 补齐。"""
    picked: list[ReqRow] = []
    seen: set[str] = set()
    for row in rows:
        family = elem_family(row.elem_id)
        if family in seen:
            continue
        seen.add(family)
        picked.append(row)
        if len(picked) >= limit:
            return picked
    for row in rows:
        if row in picked:
            continue
        picked.append(row)
        if len(picked) >= limit:
            break
    return picked


def append_triage_section(lines: list[str],
                          rows_by_file: dict[Path, list[ReqRow]],
                          map_by_file: dict[Path, dict[int, MapResult]]) -> None:
    """附录：把「设置 19 节」的未实现条目按五档分，并核对本轮做掉的条目。"""
    settings_path = next((p for p in rows_by_file if p.name == 'requirements-settings.md'), None)
    lines.append('## 5. 附录：设置 19 节的未实现分流（五档）')
    lines.append('')
    if settings_path is None:
        lines.append('没有读到 `requirements-settings.md`，本节跳过。')
        lines.append('')
        return
    rows = rows_by_file[settings_path]
    results = map_by_file[settings_path]
    unimplemented = [row for row in rows if results[row.line_no].kind in (MAP_NO_MARK, MAP_NO_POINT)]
    total = len(rows)
    lines.append('本节只分 `requirements-settings.md`（19 节 + 壳层）。分流的对象是**现行状态列**的'
                 f'未实现条目（§4 口径：映射到位无标记 + 无渲染点），共 {len(unimplemented)} 条；'
                 '第一轮逐字口径的 6,140 条是同一批需求的下限计数，两者不是两批条目，'
                 '差别在判定口径（§1 的「未实现」含「期望列抽不出字面量」的叙述型条目）。')
    lines.append('')

    buckets: dict[str, list[tuple[ReqRow, str]]] = {name: [] for name in TRIAGE_BUCKETS}
    for row in unimplemented:
        bucket, rule = triage_bucket(row, results[row.line_no].kind)
        buckets[bucket].append((row, rule))
    dims = collections.Counter(norm_dim(row.dim) for row in unimplemented)

    lines.append('### 5.1 分档口径')
    lines.append('')
    lines.append('| 档 | 判据（按顺序取第一个命中） | 含义 |')
    lines.append('|---|---|---|')
    lines.append(f'| 需素材 | 期望列命中素材词（{"、".join(TRIAGE_MATERIAL_WORDS)}） | '
                 '缺图片素材（皮肤图、缩略图、封面）就画不出，写实现也验不了 |')
    lines.append(f'| 需宿主能力 | 期望/场景/验收列命中宿主接口词（{len(TRIAGE_HOST_WORDS)} 个），'
                 f'或元素族属 {", ".join(f"`{f}`" for f in sorted(TRIAGE_HOST_FAMILIES))} | '
                 '要系统对话框、托盘、通知、休眠、下载安装等宿主通道 |')
    lines.append(f'| 需协议 | 元素族属 '
                 f'{", ".join(f"`{f}`" for f in sorted(TRIAGE_PROTOCOL_FAMILIES))} | '
                 '事实、写入或清单在 App Server，protocol v3 没有对应方法或字段 |')
    lines.append('| 取证口径 | 判定为无渲染点，或维度属横向维度（' +
                 '、'.join(sorted(TRIAGE_HORIZONTAL_DIMS)) + '），或场景是「语言包缺失回落」 | '
                 '这一档不是待实现功能，是判定要补证据：补元素锚点后重判，或界面实拍取证 |')
    lines.append('| 本版可做 | 其余 | 本地偏好、本地草稿、已有事实能落地；本节列出这些条目 |')
    lines.append('')
    lines.append('表里的词表与元素族是**一次判定**，脚本按它逐条分，不逐条人工过；'
                 '分档只说「要什么才能做」，不改状态列（状态列仍由 §4 判）。'
                 '「无渲染点」按 §4.2 的口径归取证档：那是映射缺口，不是实现结论；'
                 '把它记成「需协议」会把缺注释的元素算成缺服务端方法。')
    lines.append('')

    lines.append('### 5.2 分档计数')
    lines.append('')
    lines.append('| 档 | 条数 | 占比 | 主要维度 | 主要元素族 |')
    lines.append('|---|---:|---:|---|---|')
    for name in TRIAGE_BUCKETS:
        items = buckets[name]
        if not items:
            lines.append(f'| {name} | 0 | 0.0% | — | — |')
            continue
        dim_c = collections.Counter(norm_dim(row.dim) for row, _rule in items)
        fam_c = collections.Counter(elem_family(row.elem_id) for row, _rule in items)
        dims_top = '、'.join(f'{d} {n}' for d, n in dim_c.most_common(3))
        fams_top = '、'.join(f'`{f}` {n}' for f, n in fam_c.most_common(3))
        lines.append(f'| {name} | {len(items)} | {pct(len(items), len(unimplemented))} | '
                     f'{dims_top} | {fams_top} |')
    lines.append(f'| **合计** | **{len(unimplemented)}** | **100.0%** | — | — |')
    lines.append('')
    lines.append('未实现条目的维度构成（全量，供对照）：' +
                 '、'.join(f'{d} {n}' for d, n in dims.most_common(10)) + '。')
    lines.append('')

    lines.append('### 5.3 每档代表条目（每档 10 条）')
    lines.append('')
    for name in TRIAGE_BUCKETS:
        items = buckets[name]
        lines.append(f'#### {name}（{len(items)} 条）')
        lines.append('')
        if not items:
            lines.append('无。')
            lines.append('')
            continue
        lines.append('| 需求 ID | 元素 ID | 维度 | 期望 | 判定 | 分档依据 |')
        lines.append('|---|---|---|---|---|---|')
        for row in pick_triage_sample([r for r, _ in items]):
            where = results[row.line_no].kind
            expected = row.expected.replace('|', '/')[:36] or '—'
            rule = triage_bucket(row, where)[1]
            lines.append(f'| {row.req_id} | {row.elem_id} | {norm_dim(row.dim)} | '
                         f'{expected} | {where} | {rule} |')
        lines.append('')

    done = triage_done()
    lines.append('### 5.4 本轮做掉的条目')
    lines.append('')
    if not done:
        lines.append('本轮没有登记做掉的条目（`scripts/reconcile-triage-done.json` 为空）。')
        lines.append('')
        return
    implemented = check_done(done, rows_by_file, map_by_file)
    ok = sum(1 for _req, state, _ev in implemented if state == '已实现')
    lines.append(f'登记 {len(implemented)} 条：现状为「已实现」{ok} 条，不一致 '
                 f'{len(implemented) - ok} 条。登记表在 `scripts/reconcile-triage-done.json`，'
                 '重跑时逐条核对状态列，做没做不看口头说明。')
    lines.append('')
    lines.append('**「不一致」不等于没做**：§4 的判据只看「元素有没有渲染点 + 函数体里有没有该维度的'
                 '标记」，不看动作有没有接上写通道。这一轮改的是写通道与本端落盘（改键落盘、权限规则'
                 '真写入、归档行读事实并接恢复/删除），渲染点与标记本来就在，所以有登记的条目现状'
                 '仍是「未实现（映射到位但没有该维度的标记）」——那一条缺的是「记忆」「异常」这类横向'
                 '维度的标记，不是行为。清单里每条都写了证明它的测试名，做没做按测试核。')
    lines.append('')
    lines.append('| 需求 ID | 实现点 | 证明它的测试 | 现状 |')
    lines.append('|---|---|---|---|')
    for req_id, state, evidence in implemented:
        entry = done[req_id]
        test = entry.get('test', '') or '—'
        lines.append(f'| {req_id} | {entry.get("note", "")} | {test} | {state}{evidence} |')
    lines.append('')


def scope_elem_family(elem_id: str) -> str:
    """主窗口 / 全局元素族：`LB-70` → `LB-70`，`G-C1-04` → `G-C1`。"""
    parts = elem_id.split('-')
    if len(parts) >= 2:
        return '-'.join(parts[:2])
    return elem_id


def scope_triage_bucket(row: ReqRow, kind: str) -> tuple[str, str]:
    """把主窗口 / 全局面板的一条未实现分到五档，返回 (档位, 命中规则)。

    顺序：宿主 → 协议 → 事实 → 取证口径 → 本版可做。前三档说的是「缺什么」；
    取证口径说的是「判定缺证据」，不指向待实现功能。
    """
    blob = ' '.join((row.expected, row.acceptance, row.cells[3] if len(row.cells) > 3 else ''))
    family = scope_elem_family(row.elem_id)
    if (SCOPE_HOST_RE.search(blob) or TRIAGE_HOST_RE.search(blob)
            or norm_dim(row.dim) in SCOPE_HOST_DIMS):
        return '需宿主', '宿主接口词 / 多窗口·拖拽·输入法'
    if family in SCOPE_PROTOCOL_FAMILIES or any(w in blob for w in SCOPE_PROTOCOL_WORDS):
        return '需协议', '元素族在 protocol v3 无通道 / 文案指向服务端事实'
    if any(w in blob for w in SCOPE_FACT_WORDS) and norm_dim(row.dim) not in TRIAGE_HORIZONTAL_DIMS:
        return '需事实', '协议已有方法，Source 未投影'
    if kind == MAP_NO_POINT:
        return '取证口径', '无渲染点（先补锚点再判）'
    if norm_dim(row.dim) in TRIAGE_HORIZONTAL_DIMS:
        return '取证口径', '横向维度'
    if '语言包' in blob:
        return '取证口径', '无 i18n 层'
    return '本版可做', '本地视图状态 / 已有事实'


def pick_scope_sample(rows: list[ReqRow], limit: int = 10) -> list[ReqRow]:
    """每档取代表 10 条：先摊开元素族，再按需求 ID 补齐。"""
    picked: list[ReqRow] = []
    seen: set[str] = set()
    for row in rows:
        family = scope_elem_family(row.elem_id)
        if family in seen:
            continue
        seen.add(family)
        picked.append(row)
        if len(picked) >= limit:
            return picked
    for row in rows:
        if row in picked:
            continue
        picked.append(row)
        if len(picked) >= limit:
            break
    return picked


def scope_done() -> dict[str, dict[str, str]]:
    """主窗口 / 全局面板本轮做掉的需求登记；没有文件就是空表。"""
    if not SCOPE_DONE_FILE.exists():
        return {}
    try:
        data = json.loads(SCOPE_DONE_FILE.read_text())
    except (OSError, ValueError):
        return {}
    done = data.get('done', {}) if isinstance(data, dict) else {}
    if not isinstance(done, dict):
        return {}
    out: dict[str, dict[str, str]] = {}
    for req_id, value in done.items():
        if isinstance(value, dict):
            out[req_id] = {k: str(v) for k, v in value.items()}
        else:
            out[req_id] = {'note': str(value), 'test': ''}
    return out


def append_scope_triage_section(lines: list[str],
                                rows_by_file: dict[Path, list[ReqRow]],
                                map_by_file: dict[Path, dict[int, MapResult]]) -> None:
    """附录：主窗口（main）与全局面板（global）的未实现五档分流与本轮登记。"""
    lines.append('## 6. 附录：主窗口与全局面板的未实现分流（五档）')
    lines.append('')
    picked_files = [(p, rows_by_file[p]) for p in rows_by_file
                    if p.name in SCOPE_REQ_FILES]
    if not picked_files:
        lines.append('没有读到主窗口 / 全局面板的需求表，本节跳过。')
        lines.append('')
        return

    # 登记对象是现行状态列的未实现（§4 口径），不是 §1 的第一轮下限。
    per_file: dict[str, list[ReqRow]] = {}
    per_results: dict[str, dict[int, MapResult]] = {}
    for path, rows in picked_files:
        results = map_by_file[path]
        per_file[path.name] = [row for row in rows
                               if results[row.line_no].kind in (MAP_NO_MARK, MAP_NO_POINT)]
        per_results[path.name] = results
    all_rows = [row for rows in per_file.values() for row in rows]
    total = len(all_rows)
    first_round = {'requirements-main.md': 881, 'requirements-global.md': 554}

    lines.append('本节只分 `requirements-main.md`（主窗口五区）与 `requirements-global.md`'
                 '（全局面板八类）。分流的对象是**现行状态列**的未实现条目（§4 口径：'
                 '映射到位无标记 + 无渲染点）：主窗口 '
                 f'{len(per_file.get("requirements-main.md", []))} 条、全局面板 '
                 f'{len(per_file.get("requirements-global.md", []))} 条，合计 {total} 条。')
    lines.append('')
    lines.append('领导给的 881 / 554 是 §1 第一轮「未实现」的下限计数（只看字面量、符号、'
                 '测试），与本节不是同一批：第二轮把「元素映射到位、但函数体里没有该维度'
                 '标记」也算成未实现，所以全局面板从 554 涨到 '
                 f'{len(per_file.get("requirements-global.md", []))} 条；主窗口从 '
                 f'{first_round["requirements-main.md"]} 条到 '
                 f'{len(per_file.get("requirements-main.md", []))} 条，差别很小。')
    lines.append('')

    lines.append('### 6.1 分档口径')
    lines.append('')
    lines.append('| 档 | 判据（按顺序取第一个命中） | 含义 |')
    lines.append('|---|---|---|')
    lines.append(f'| 需宿主 | 期望/场景/验收列命中宿主接口词（{len(SCOPE_HOST_WORDS)} 个，另继承 §5 的词表），'
                 '或维度属多窗口、拖拽、输入法 | 要系统保存面板、窗口拖动、全屏、托盘、下载安装等宿主通道 |')
    lines.append(f'| 需协议 | 元素族属 '
                 f'{", ".join(f"`{f}`" for f in sorted(SCOPE_PROTOCOL_FAMILIES))}，'
                 '或文案命中服务端事实词（' + '、'.join(SCOPE_PROTOCOL_WORDS) + '） | '
                 'protocol v3 没有对应方法或字段（未读、改动摘要、逐会话分支、成员审批明细、'
                 '会话 Fork 副本、市场安装、浏览器数据） |')
    lines.append(f'| 需事实 | 文案命中（{"、".join(SCOPE_FACT_WORDS)}） | '
                 '协议已有方法（如 `session.agent.get/update`），Go 侧 Source 还没投影出该读数 |')
    lines.append('| 取证口径 | 判定为无渲染点，或维度属横向维度（' +
                 '、'.join(sorted(TRIAGE_HORIZONTAL_DIMS)) + '），或场景是「语言包缺失回落」 | '
                 '这一档不是待实现功能，是判定要补证据：补元素锚点后重判，或界面实拍取证 |')
    lines.append('| 本版可做 | 其余 | 本端视图状态、已有事实或已有写通道能落地 |')
    lines.append('')
    lines.append('分档只说「要什么才能做」，不改状态列；状态列仍由 §4 判。')
    lines.append('')

    lines.append('### 6.2 分档计数')
    lines.append('')
    lines.append('| 文件 | 现行未实现 | 本版可做 | 需协议 | 需事实 | 需宿主 | 取证口径 |')
    lines.append('|---|---:|---:|---:|---:|---:|---:|')
    combined: dict[str, list[tuple[ReqRow, str]]] = {name: [] for name in SCOPE_TRIAGE_BUCKETS}
    for path, rows in picked_files:
        results = per_results[path.name]
        counts = {name: 0 for name in SCOPE_TRIAGE_BUCKETS}
        for row in per_file[path.name]:
            bucket, rule = scope_triage_bucket(row, results[row.line_no].kind)
            counts[bucket] += 1
            combined[bucket].append((row, rule))
        lines.append(f'| `{path.name}` | {len(per_file[path.name])} | '
                     + ' | '.join(str(counts[name]) for name in SCOPE_TRIAGE_BUCKETS) + ' |')
    lines.append(f'| **合计** | **{total}** | '
                 + ' | '.join(f'**{len(combined[name])}**' for name in SCOPE_TRIAGE_BUCKETS)
                 + ' |')
    lines.append('')
    if not combined['需事实']:
        lines.append('「需事实」本轮为 0：唯一的需事实项（状态条的 Agent 配置，'
                     '`session.agent.get/update` 已存在但 Source 没有投影）已在本轮接上；'
                     'SB-08 剩下的窄窗细节按横向维度归取证口径。')
        lines.append('')
    for name in SCOPE_TRIAGE_BUCKETS:
        items = combined[name]
        if not items:
            continue
        dim_c = collections.Counter(norm_dim(row.dim) for row, _rule in items)
        fam_c = collections.Counter(scope_elem_family(row.elem_id) for row, _rule in items)
        lines.append(f'- **{name}** {len(items)} 条（{pct(len(items), total)}）：维度 '
                     + '、'.join(f'{d} {n}' for d, n in dim_c.most_common(3))
                     + '；元素族 ' + '、'.join(f'`{f}` {n}' for f, n in fam_c.most_common(5)) + '。')
    lines.append('')

    lines.append('### 6.3 每档代表条目（每档 10 条）')
    lines.append('')
    for name in SCOPE_TRIAGE_BUCKETS:
        items = combined[name]
        lines.append(f'#### {name}（{len(items)} 条）')
        lines.append('')
        if not items:
            lines.append('无。')
            lines.append('')
            continue
        lines.append('| 需求 ID | 元素 ID | 维度 | 期望 | 判定 | 分档依据 |')
        lines.append('|---|---|---|---|---|---|')
        for row in pick_scope_sample([r for r, _ in items]):
            result = per_results[row.path.name][row.line_no]
            expected = row.expected.replace('|', '/')[:36] or '—'
            rule = scope_triage_bucket(row, result.kind)[1]
            lines.append(f'| {row.req_id} | {row.elem_id} | {norm_dim(row.dim)} | '
                         f'{expected} | {result.kind} | {rule} |')
        lines.append('')

    done = scope_done()
    lines.append('### 6.4 本轮做掉的条目（主窗口 / 全局面板）')
    lines.append('')
    if not done:
        lines.append('本轮没有登记（`scripts/reconcile-triage-done-scope.json` 为空）。')
        lines.append('')
        return
    implemented = check_done(done, rows_by_file, map_by_file)
    ok = sum(1 for _req, state, _ev in implemented if state == '已实现')
    lines.append(f'登记 {len(implemented)} 条：现状为「已实现」{ok} 条，不一致 '
                 f'{len(implemented) - ok} 条。做没做按状态列与测试名核。')
    lines.append('')
    if len(implemented) - ok > 0:
        lines.append('**「不一致」不等于没做**：§4 的判据只看「元素有没有渲染点 + 函数体里有没有'
                     '该维度的标记」，不看滚动状态、写通道与状态行这类没有字符串/符号可抓的行为。'
                     '清单里每条都写了证明它的测试名，做没做按测试核。')
        lines.append('')
    lines.append('| 需求 ID | 实现点 | 证明它的测试 | 现状 |')
    lines.append('|---|---|---|---|')
    for req_id, state, evidence in implemented:
        entry = done[req_id]
        test = entry.get('test', '') or '—'
        lines.append(f'| {req_id} | {entry.get("note", "")} | {test} | {state}{evidence} |')
    lines.append('')


def check_done(done: dict[str, dict[str, str]],
               rows_by_file: dict[Path, list[ReqRow]],
               map_by_file: dict[Path, dict[int, MapResult]]) -> list[tuple[str, str, str]]:
    """核对登记表：每条登记的需求现在必须已是「已实现（…）」。"""
    index: dict[str, tuple[ReqRow, MapResult]] = {}
    for path, rows in rows_by_file.items():
        for row in rows:
            index.setdefault(row.req_id, (row, map_by_file[path][row.line_no]))
    out: list[tuple[str, str, str]] = []
    for req_id in done:
        hit = index.get(req_id)
        if hit is None:
            out.append((req_id, '找不到这条需求', ''))
            continue
        row, result = hit
        if result.kind in (MAP_NO_MARK, MAP_NO_POINT):
            out.append((req_id, f'仍为未实现（{result.kind}）', ''))
            continue
        evidence = f'（{result.kind}）'
        out.append((req_id, '已实现', evidence))
    return out


def main() -> int:
    global IMPL, TESTS, LITERAL_HITS, RAW_HITS, TEST_LITERAL_HITS, TEST_RAW_HITS, \
        SYMBOL_INDEX, ELEM_COMMENT_INDEX, TEST_INDEX, GO_BY_REL
    ap = argparse.ArgumentParser(description=__doc__)
    ap.add_argument('--dry-run', action='store_true', help='只打印分布，不改文件')
    args = ap.parse_args()

    IMPL, TESTS, GO_BY_REL = [], [], {}
    for path in sorted(GO_DIR.rglob('*.go')):
        rel = path.relative_to(ROOT).as_posix()
        gf = scan_go(path, rel)
        GO_BY_REL[rel] = gf
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
    RENDER_INDEX = build_element_render_index(inv, known_ids)
    print(f'元素映射：{len(RENDER_INDEX)} 个元素有渲染点（已知元素 {len(known_ids)} 个）')

    verdicts_by_file: dict[Path, dict[int, Verdict]] = {}
    map_by_file: dict[Path, dict[int, MapResult]] = {}
    for path, rows in rows_by_file.items():
        verdicts_by_file[path] = {row.line_no: judge(row, inv) for row in rows}
        map_by_file[path] = {row.line_no: judge_mapped(row, inv, RENDER_INDEX, verdicts_by_file[path][row.line_no])
                             for row in rows}
        counts = collections.Counter(v.kind for v in map_by_file[path].values())
        print(f'{path.name}: {len(rows)} 条  ' + '  '.join(
            f'{k}={counts.get(k, 0)}' for k in MAP_BUCKETS if counts.get(k)))
        if not args.dry_run:
            changed = rewrite(path, {
                ln: Verdict(kind=r.kind, text=r.text) for ln, r in map_by_file[path].items()
            })
            print(f'  状态列{"已写回" if changed else "无变化"}')

    if args.dry_run:
        return 0

    write_report(rows_by_file, verdicts_by_file, map_by_file, inv, RENDER_INDEX)
    print(f'报告：{REPORT.relative_to(ROOT)}')
    return 0


if __name__ == '__main__':
    sys.exit(main())
