# -*- coding: utf-8 -*-
"""按 design-baseline.md §1–§7 的常量，给四份 requirements-*.md 验收列（形态②）落值。
只改验收单元格；其余列逐字保留。"""
import re, sys, random

TBD_REF = '判据TBD；量 %s；比对项为设计基线（见 design-baseline §10 未覆盖项）'
RES_REF = '比对项为设计基线（见 design-baseline §10 未覆盖项）'

FB = ''

FONTS = [(10,'小徽标'),(10.5,'说明文字'),(11,'键帽'),(12.25,'UI 正文'),(13,'代码'),
         (14,'正文'),(15.75,'对话框标题'),(16.625,'Markdown 二级标题'),(17.5,'页面标题'),(21,'空态主标题')]
LINES = [(14,'说明文字'),(17.5,'UI 正文'),(20,'代码'),(21,'输入框正文'),(24.5,'正文'),(28,'空态主标题')]

def nearest(num, table):
    return min(table, key=lambda t: abs(t[0] - num))

def explicit_size(text):
    for pat in (r'字号\s*(\d+(?:\.\d+)?)\s*(?:px|号)',
                r'(\d+(?:\.\d+)?)\s*(?:px|号)\s*级',
                r'(\d+(?:\.\d+)?)\s*px\s*(?:字|半粗|中粗|粗体|加粗|粗|常规|危险色|警示色|成功色|次要色|主色|灰|正文|说明)'):
        m = re.search(pat, text)
        if m:
            return float(m.group(1))
    return None

def explicit_line(text):
    m = re.search(r'行高\s*(\d+(?:\.\d+)?)', text)
    return float(m.group(1)) if m else None

def hit(ctx, pat):
    return re.search(pat, ctx) is not None

def rule(ctx, rules):
    for pat, v in rules:
        if hit(ctx, pat):
            return v, True
    return None, False

def font_size(ctx):
    n = explicit_size(ctx) or explicit_size(FB)
    if n is not None:
        val, name = nearest(n, FONTS)
        return '字号 %s %spx' % (name, val)
    v, ok = rule(ctx, [
        (r'空态', '字号 空态主标题 21px'),
        (r'对话框标题|弹窗标题', '字号 对话框标题 15.75px'),
        (r'页面标题|页首标题', '字号 页面标题 17.5px'),
        (r'代码|等宽', '字号 代码 13px'),
        (r'键帽', '字号 键帽 11px'),
        (r'徽标', '字号 小徽标 10px'),
        (r'说明|描述|辅助|占位|提示|分组标|弱文字|次要|错误文字|版本|元信息', '字号 说明文字 10.5px'),
        (r'标题', '字号 页面标题 17.5px'),
        (r'多行输入框|输入区块|编辑器正文|编辑器', '字号 正文 14px'),
        (r'正文|消息|Markdown|markdown', '字号 正文 14px'),
    ])
    if hit(ctx, r'R-NT|NT-\d'):
        return '字号 空态主标题 21px'
    return v or '字号 UI 正文 12.25px'

def line_height(ctx):
    n = explicit_line(ctx) or explicit_line(FB)
    if n is not None:
        val, name = nearest(n, LINES)
        return '行高 %s %spx' % (name, val)
    v, ok = rule(ctx, [
        (r'空态', '行高 空态主标题 28px'),
        (r'代码|等宽', '行高 代码 20px'),
        (r'说明|描述|辅助|占位|提示|分组标|弱文字|次要|错误文字', '行高 说明文字 14px'),
        (r'正文|消息|Markdown|markdown', '行高 正文 24.5px'),
        (r'标题|页首', '行高 页面标题 24.5px'),
    ])
    return v or '行高 UI 正文 17.5px'

def color_text(ctx):
    v, ok = rule(ctx, [
        (r'破坏|危险|错误|删除|失败|红字', '字色 破坏 明 #e7000b / 暗 #ff6467'),
        (r'警示|警告', '字色 警告 明 #e17100 / 暗 #ffb900'),
        (r'成功|已连接|通过', '字色 成功 明 #009966 / 暗 #00d492'),
        (r'信息', '字色 信息 明 #2b7fff / 暗 #51a2ff'),
        (r'主色|主按钮|品牌', '字色 主色 明 #339751 / 暗 #63cb7b'),
        (r'侧栏弱|侧栏分组', '字色 侧栏弱文字 明 #8f9294 / 暗 #868b8f'),
        (r'次要|说明|描述|占位|提示|弱|muted|图标|辅助', '字色 次文字 明 #737373 / 暗 #8a8a8a'),
        (r'主文字|前景|标题|正文|高对比', '字色 主文字 明 #0a0a0a / 暗 #ececec'),
        (r'禁用', '字色 禁用 主色 opacity 50% 明 #a3cdad / 暗 #3f6843'),
    ])
    return v or '字色 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a'

def color_bg(ctx):
    v, ok = rule(ctx, [
        (r'悬停|hover', '底色 悬停底 明 #f5f5f5 / 暗 #262626'),
        (r'选中态|选中底|已选中|当前行|当前项', '底色 选中底 明 #f5f5f5 / 暗 #262626'),
        (r'侧栏', '底色 侧栏 明 #f6fdff / 暗 #090c0d'),
        (r'浮层|弹窗|对话框|菜单|下拉', '底色 浮层 明 #ffffff / 暗 #222222'),
        (r'主色|主按钮|实心按钮', '底色 主色 明 #339751 / 暗 #63cb7b'),
        (r'遮罩', '底色 遮罩 bg-black/80 合成 明 #333333'),
        (r'输入框|搜索框|筛选框', '底色 输入框 明 #ffffff / 暗 #0c0c0c'),
        (r'禁用', '底色 禁用 主色 opacity 50% 明 #a3cdad / 暗 #3f6843'),
    ])
    return v or '底色 内容底 明 #ffffff / 暗 #0c0c0c、侧栏 明 #f6fdff / 暗 #090c0d'

def color_any(ctx):
    v, ok = rule(ctx, [
        (r'破坏|危险|错误|删除|失败|红', '色值 破坏 明 #e7000b / 暗 #ff6467'),
        (r'警示|警告', '色值 警告 明 #e17100 / 暗 #ffb900'),
        (r'成功|已连接|通过', '色值 成功 明 #009966 / 暗 #00d492'),
        (r'信息', '色值 信息 明 #2b7fff / 暗 #51a2ff'),
        (r'主色|主按钮|品牌|聚焦环|焦点环', '色值 主色 明 #339751 / 暗 #63cb7b'),
        (r'悬停|hover', '色值 悬停底 明 #f5f5f5 / 暗 #262626'),
        (r'选中态|选中底|已选中|当前行|当前项', '色值 选中底 明 #f5f5f5 / 暗 #262626'),
        (r'边框|描边|分割线|分隔线', '色值 边框 明 #e5e5e5 / 暗 #2e2e2e'),
        (r'浮层|弹窗|对话框|菜单', '色值 浮层 明 #ffffff / 暗 #222222'),
        (r'次要|说明|描述|占位|提示|弱|图标|辅助|muted', '色值 次文字 明 #737373 / 暗 #8a8a8a'),
        (r'主文字|前景|标题|正文|高对比', '色值 主文字 明 #0a0a0a / 暗 #ececec'),
        (r'遮罩', '色值 遮罩 bg-black/80 合成 明 #333333'),
        (r'禁用', '色值 禁用 主色 opacity 50% 明 #a3cdad / 暗 #3f6843'),
        (r'底|背景', '色值 内容底 明 #ffffff / 暗 #0c0c0c'),
    ])
    return v or '色值 主文字 明 #0a0a0a / 暗 #ececec、次文字 明 #737373 / 暗 #8a8a8a'

def font_weight(ctx):
    v, ok = rule(ctx, [
        (r'半粗|semibold|600', '字重 半粗 600'),
        (r'中粗|medium|500', '字重 中粗 500'),
        (r'粗体|加粗|bold|700', '字重 粗 700'),
        (r'常规字重|regular|400 字重', '字重 常规 400'),
    ])
    return v

def radius(ctx):
    v, ok = rule(ctx, [
        (r'胶囊|头像|开关|圆点|计数', '圆角 full'),
        (r'主窗卡片|聊天卡|面板卡|卡片框|卡片|横幅|面板', '圆角 xl 12.75px'),
        (r'对话框|主题卡|卡内分区|浮层|菜单|容器|区块', '圆角 lg 8.75px'),
        (r'按钮|输入框|搜索框|筛选框|下拉|选择器|分段|菜单项|链接', '圆角 md 6.75px'),
        (r'标签|chip|内层|徽标|胶囊', '圆角 sm 4.75px'),
    ])
    return v or '圆角 按钮·输入框 6.75 / 卡内分区 8.75 / 主窗卡 12.75 / 小标签 4.75px'

def stroke_width(ctx):
    v, ok = rule(ctx, [
        (r'焦点环|焦点框|聚焦', '描边 焦点环 3px'),
        (r'卡片外框|外框', '描边 卡片外框 0.5px'),
        (r'主题卡|选中态|选中边', '描边 选中态 1px（只换色不换宽，--primary）'),
    ])
    return v or '描边 常规 1px（卡片外框 0.5px / 焦点环 3px）'

def control_size(ctx):
    v, ok = rule(ctx, [
        (r'开关', '控件尺寸 开关 28×16'),
        (r'复选框|单选|勾选', '控件尺寸 复选框·单选框 14×14'),
        (r'头像', '控件尺寸 头像 18×18'),
        (r'键帽', '控件尺寸 键帽 32'),
        (r'图标', '控件尺寸 图标 16 / 14 / 12px'),
        (r'输入框|搜索框|筛选框|下拉', '控件尺寸 输入框高 32（行内 25）'),
        (r'徽标', '控件尺寸 徽标高 20（无边框胶囊 18）'),
        (r'会话行|会话项', '控件尺寸 会话行 30'),
        (r'项目行', '控件尺寸 侧栏项目行 21'),
        (r'导航行|设置导航', '控件尺寸 设置导航行 35'),
        (r'面板标签|页签', '控件尺寸 面板标签 25'),
        (r'列表行|菜单项|菜单行', '控件尺寸 列表行 28'),
        (r'大按钮', '控件尺寸 按钮高 大 35'),
        (r'次按钮|次级小按钮|小按钮|size="sm"', '控件尺寸 按钮高 次 28'),
        (r'主按钮|对话框按钮|设置页按钮|危险按钮|描边按钮|实心按钮|确认按钮', '控件尺寸 按钮高 常规 32'),
        (r'行内|密集|文字按钮', '控件尺寸 按钮高 行内 21'),
        (r'按钮', '控件尺寸 按钮高 行内 21 / 小 25 / 次 28 / 常规 32 / 大 35'),
    ])
    return v or '控件尺寸 按钮高 行内 21 / 小 25 / 次 28 / 常规 32 / 大 35'

def layout_size(ctx):
    v, ok = rule(ctx, [
        (r'侧栏', '栏宽·布局 侧栏 200–360（默认 256，实测 360）'),
        (r'右栏', '栏宽·布局 右栏 ≥280（实测 296 / 305）'),
        (r'输入区|输入卡', '栏宽·布局 输入区最大宽 672'),
        (r'设置内容|内容列', '栏宽·布局 设置内容最大宽 1008'),
        (r'导航列|导航栏', '栏宽·布局 设置导航列 224'),
        (r'窗口', '栏宽·布局 最小窗口 480×540'),
        (r'正文列|时间线', '栏宽·布局 聊天内容列最大宽 784'),
    ])
    return v or '栏宽·布局 侧栏 200–360 / 右栏 ≥280 / 聊天列 784 / 输入区 672 / 设置 1008 / 导航 224'

def motion(ctx):
    v, ok = rule(ctx, [
        (r'折叠|展开|收起', '动效 折叠 300ms cubic-bezier(0.16, 1, 0.3, 1)'),
        (r'箭头|旋转', '动效 箭头 150ms cubic-bezier(0.4, 0, 0.2, 1)'),
        (r'颜色|悬停|hover|主题', '动效 颜色过渡 150ms cubic-bezier(0.4, 0, 0.2, 1)'),
        (r'布局|侧栏|右栏|伸缩|让位|平移|宽度变化|进出', '动效 布局过渡 250ms'),
        (r'入场|出现|淡入|进入', '动效 入场 240ms ease-out'),
        (r'离开|消失|淡出|退出', '动效 离开 150ms ease-in'),
    ])
    return v or '动效 布局 250ms / 颜色 150ms / 折叠 300ms；缓动 cubic-bezier(0.16, 1, 0.3, 1) / cubic-bezier(0.4, 0, 0.2, 1)'

CONTROL_HINT = r'按钮|输入框|搜索框|筛选框|开关|列表行|会话行|项目行|工具行|标签|页签|徽标|胶囊|复选框|单选框|导航行|菜单项|菜单行|键帽|头像|图标'
LAYOUT_HINT  = r'侧栏|右栏|栏宽|内容列|输入区|输入卡|窗口|导航列|正文列|时间线'
ICON_HINT    = r'图标|icon|箭头|放大镜|插画|logo'

COVERED = [
    (r'悬停底色|悬停背景|悬停色', lambda c: '底色 悬停底 明 #f5f5f5 / 暗 #262626（侧栏 明 #e2e9ed / 暗 #282c2b）'),
    (r'选中底色', lambda c: '底色 选中底 明 #f5f5f5 / 暗 #262626'),
    (r'字号', font_size),
    (r'行高', line_height),
    (r'字重', font_weight),
    (r'字色|文字色|文字颜色|文本色|前景色|前景|图标色', color_text),
    (r'底色|背景色', color_bg),
    (r'描边色|边框色|线条色', lambda c: '色值 边框 明 #e5e5e5 / 暗 #2e2e2e'),
    (r'色值|颜色', color_any),
    (r'圆角', radius),
    (r'描边宽|边框宽|描边|边框|线宽', stroke_width),
    (r'间距|间隔|间隙', lambda c: '间距 纵向 10.5 / 14 / 7 / 21、横向 3.5 / 7 / 14px'),
    (r'行距', lambda c: '行距 3.5 / 7 / 10.5 / 14px（3.5px 网格）'),
    (r'内边距|留白', lambda c: '内边距 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28）/ 卡片外框 4.5；档位 3.5·7·10.5·14·21·28px'),
    (r'左缘', lambda c: '内边距 左缘 侧栏 7 / 时间线 10.5 / 设置 21（宽屏 28）'),
    (r'图标尺寸|图标大小', lambda c: '控件尺寸 图标 16 / 14 / 12px'),
    (r'栏宽|列宽', layout_size),
    (r'时长|缓动|过渡|动画|位移', motion),
    (r'尺寸|宽高|高度|项高|宽度', 'SIZE'),
]

UNCOVERED = ['包围盒','坐标','位置','相对位置','对齐线','对齐偏差','对齐','基线','对比度','对比','播报',
    '读屏','可读名','序位','顺序','元素序列','层叠','焦点','键位','可点','状态','展开态','选中态',
    '计数','取值','提示文案','文案','文本','字数','字符数','行数','截断','折行','滚动','时延','帧',
    '像素','彩边','模糊','阴影','不透明度','形状','图标形','几何','轨道色','轨道宽','填充比例',
    '滑块','字体','可见性','出现','消失','时刻','数量','语义','角色','遮挡','重叠','抗锯齿',
    '缩进','段距','内高','折行位置','左缘距','右缘距','描边粗细']

CLEAN_PAT = re.compile(r'`[^`]*`|「[^」]*」|[A-Z]{1,3}-\d+(?:-\d+)?|[\d.]+(?:px|ms|%)?|'
                       r'[的与和及、，,（）()=≥<>·\s/＋+…\.]+')

def tidy(text):
    t = re.sub(r'（`[^`]*`）', '', text)
    t = t.replace('`', '')
    t = re.sub(r'\s+', ' ', t).strip(' ；、，')
    return t

def clean(s):
    return CLEAN_PAT.sub('', s).strip()

HARD_NC = r'时延|帧间隔|帧率|帧|耗时|响应|内存|CPU|请求次数|加载时长'
SEMANTIC_BG = r'破坏|危险|警示|警告|成功|信息'

STATE_WORDS = r'禁用态|禁用时|禁用下|（禁用'

def classify_segment(seg, base_ctx, state_ctx, ctx_fallback):
    global FB
    FB = ctx_fallback or ''
    subjects = re.findall(r'[^、，；\s]+的', seg)
    prop = re.sub(r'[^、，；\s]+的', '', seg)
    clean_subj = ' '.join(x[:-1] for x in subjects)
    states = ' '.join(sorted(set(re.findall(STATE_WORDS, state_ctx))))
    ctx = base_ctx + ' ' + clean_subj + ' ' + prop + ' ' + states
    if re.search(HARD_NC, prop):
        return [], [seg.strip()]
    vals, work = [], prop
    for pat, fn in COVERED:
        if not re.search(pat, work):
            continue
        kw = re.search(pat, work).group(0)
        if kw in ('底色', '背景色') and re.search(SEMANTIC_BG, ctx):
            continue
        if fn == 'SIZE':
            v = size_value(kw, ctx)
            if v is None:
                continue
        else:
            if kw in ('线宽', '描边') and hit(ctx, ICON_HINT) and not hit(ctx, CONTROL_HINT):
                continue  # 图标几何不覆盖（§10.3）
            v = fn(ctx)
            if v is None and ctx_fallback:
                v = fn(ctx + ' ' + ctx_fallback)
            if v is None:
                v = '字重 常规 400 / 中粗 500 / 半粗 600 / 粗 700'
        if v not in vals:
            vals.append(v)
        work = re.sub(pat, '', work)
    if not vals and not work.strip():
        pass
    if not vals and ctx_fallback:
        # 语境不足时用「期望」列兜底一次
        ctx2 = ctx + ' ' + ctx_fallback
        vals2, work2 = [], prop
        for pat, fn in COVERED:
            if not re.search(pat, work2):
                continue
            kw = re.search(pat, work2).group(0)
            if kw in ('底色', '背景色') and re.search(SEMANTIC_BG, ctx2):
                continue
            if fn == 'SIZE':
                v = size_value(kw, ctx2)
                if v is None:
                    continue
            else:
                if kw in ('线宽', '描边') and hit(ctx2, ICON_HINT) and not hit(ctx2, CONTROL_HINT):
                    continue
                v = fn(ctx2)
            if v not in vals2:
                vals2.append(v)
        if vals2:
            vals, work = vals2, work2
    if not vals:
        return [], [tidy(seg)]
    residue = []
    for kw in UNCOVERED:
        if kw in prop and kw not in residue:
            residue.append(kw)
    return vals, residue

def size_value(kw, ctx):
    if kw in ('栏宽', '列宽', '宽度'):
        if not hit(ctx, LAYOUT_HINT):
            return None
        return layout_size(ctx)
    if kw in ('尺寸', '宽高'):
        if hit(ctx, ICON_HINT):
            return '控件尺寸 图标 16 / 14 / 12px'
        if hit(ctx, CONTROL_HINT):
            return control_size(ctx)
        if hit(ctx, LAYOUT_HINT):
            return layout_size(ctx)
        return None
    if hit(ctx, CONTROL_HINT):
        return control_size(ctx)
    return None

def split_props(seg):
    """按 、，与和及 切成单个属性，保护「」与 ` ` 内的字面文本。"""
    saved = []
    def mask(m):
        saved.append(m.group(0))
        return '\x00%d\x00' % (len(saved) - 1)
    tmp = re.sub(r'「[^」]*」|`[^`]*`', mask, seg)
    parts = [p for p in re.split(r'、|，|与|和|及', tmp) if p.strip()]
    out = []
    for p in parts:
        out.append(re.sub(r'\x00(\d+)\x00', lambda m: saved[int(m.group(1))], p).strip())
    return [p for p in out if p]

def rewrite_cell(acc, dim, eid, expect):
    m = re.search(r'判据TBD；量\s*(.*?)；比对项为设计基线', acc, re.S)
    if not m:
        return None
    q = m.group(1).strip()
    base_ctx = dim + ' ' + eid
    vals, residue = [], []
    for seg in [s for s in re.split(r'；', q) if s.strip()]:
        subs = [s for s in split_props(seg)] or [seg]
        for one in subs:
            v, r = classify_segment(one, base_ctx, q, expect)
            vals += v
            residue += r
    def dedup(seq):
        out = []
        for s in seq:
            if s not in out:
                out.append(s)
        return out
    vals = dedup(vals)
    residue = dedup([x for x in residue if clean(x)])
    if not vals:
        return TBD_REF % q
    out = '判据 = ' + '；'.join(vals)
    if residue:
        out += '；其余（' + '、'.join(residue) + '）TBD，' + RES_REF
    return out

FILES = ['requirements-main.md','requirements-panels.md','requirements-settings.md','requirements-global.md']
ROOT = 'docs/domains/requirements/'

def main():
    write = '--write' in sys.argv
    report = {}
    kepts = []
    samples = []
    for f in FILES:
        path = ROOT + f
        lines = open(path, encoding='utf-8').read().split('\n')
        changed = kept = 0
        for i, ln in enumerate(lines):
            if not ln.startswith('|') or '判据TBD' not in ln:
                continue
            cells = ln.split('|')
            if len(cells) != 9:
                continue
            acc = cells[-2]
            if '判据TBD' not in acc:
                continue
            new = rewrite_cell(acc, cells[3].strip(), cells[2].strip(), cells[5].strip())
            if new is None:
                continue
            if new.startswith('判据 ='):
                changed += 1
                samples.append((f, cells[1].strip(), cells[3].strip(), cells[5].strip()[:60], acc.strip(), new))
            else:
                kept += 1
                kepts.append((f, cells[1].strip(), cells[3].strip(), acc.strip(), new))
            cells[-2] = ' ' + new + ' '
            lines[i] = '|'.join(cells)
        if write:
            open(path, 'w', encoding='utf-8').write('\n'.join(lines))
        report[f] = (changed, kept)
        print('%-26s 落值 %5d  仍TBD %5d' % (f, changed, kept))
    import json
    json.dump(samples, open('/tmp/tbd/changed.json', 'w'), ensure_ascii=False)
    json.dump(kepts, open('/tmp/tbd/kepts.json', 'w'), ensure_ascii=False)
    random.seed(23)
    for s in random.sample(samples, min(28, len(samples))):
        print('---', s[0], s[1], s[2], '|', s[3])
        print('  BEF:', s[4]); print('  AFT:', s[5])
    print('MODE', 'WRITE' if write else 'DRY')

main()
