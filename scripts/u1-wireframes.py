#!/usr/bin/env python3
"""把 docs/domains/wireframes 的四份线框重出到现行口径（§9.10 时效与作废）。

线框的画布是手写的，这个脚本只做"重出时要一致"的四件事，幂等：重复跑不会叠加。

1. 每份页面顶部写身份块：依据的裁决、对应文档与章节、生成日期、状态；
2. 每个画面标签（`.tag`）同样写身份，出问题能指回裁决与文档；
3. 窗口装饰归界面（UI-19）：画布顶部留一条与窗口控件同高的空带，红绿灯落在
   左栏上方；标题行里不再重复会话名（它是窗口 Title），只留右栏入口；
4. 索引页 `index.html` 整体重出：「画面 × 依据 × 日期 × 状态」一张表 + 图例，
   并登记 comps 的归档。

线框里的框线与间距是画法，不构成 UI-18 的判据（§9.10 第 4 条）。
"""
import io
import re

DATE = "2026-11-13"
STATUS = "现行"
REV = "第 4 版（UI-20…UI-23：图标与文字最后手段）"
DECISIONS = "UI-11 / UI-15 / UI-16 / UI-17 / UI-18 / UI-19（并 §9.0 A1–A4、D1–D6）"

PAGES = {
    "a.html": {
        "doc": "U1-concepts §1、U1-ia §1/§7、U2 §4.4/§4.5、U3 §1.1",
        "shots": [
            ("默认态（摘要折叠）", "UI-11 组头写内容 / UI-16 行即入口 / UI-19 标题行写会话名", "U1-ia §1 R2/R3、U2 §4.4/§4.6"),
            ("摘要展开", "UI-11 计数进展开 / UI-16 方向指示", "U1-ia §1 R3、U3 T7"),
            ("动效关闭", "UI-5 布局与状态不变", "U2 §5.1"),
            ("运行中", "UI-10 队列贴输入 / §9.0 A2 状态归样式", "U1-ia §5、U2 §4.1"),
            ("等待你同意", "UI-1 占输入位 / UI-15 用量在输入卡", "U1-ia §1 R4、U2 §4.1"),
            ("出错/结果不确定", "§9.0 只有「结果不确定」保留文字", "U1-ia §5、U2 §4.1"),
            ("长历史", "UI-12 索引只索引用户输入", "U1-ia §1 R3"),
        ],
    },
    "b.html": {
        "doc": "U1-concepts §2、U1-ia §1/§7、U2 §4.4、U3 §1.1",
        "shots": [
            ("默认态", "UI-11 组头 / UI-12 轮次索引", "U1-ia §1 R3"),
            ("运行中", "UI-10 队列贴输入", "U1-ia §5"),
            ("等待你同意", "UI-1 占输入位", "U1-ia §1 R4"),
            ("出错/结果不确定", "§9.0 A2 状态归样式", "U1-ia §5"),
            ("长历史", "UI-12 索引只索引用户输入", "U1-ia §1 R3"),
        ],
    },
    "ui9-11.html": {
        "doc": "U1-ia §1/§5、U2 §4.1/§4.4/§4.5、U3 §1.1/T10",
        "shots": [
            ("运行中（队列贴输入，组头只给内容）", "UI-10 / UI-11", "U1-ia §5、U2 §4.1"),
            ("失败（组头写内容，状态归图标）", "UI-11 / §9.0 A2", "U1-ia §5、U2 §4.1"),
            ("等待你同意（输入位被占，队列仍在）", "UI-1 / UI-15", "U1-ia §1 R4"),
            ("标题行 · 全默认（会话名 + 用量在输入卡内）", "UI-9 / UI-15 / UI-19", "U1-ia §1 R2/R4、U2 §4.6"),
            ("标题行 · 有偏离（最小形式提示）", "UI-9 / UI-19", "U1-ia §1 R2、U2 §4.6"),
        ],
    },
}


def ident_block(doc):
    return (
        '<div class="ident">'
        f"<b>依据</b> {DECISIONS} · <b>文档</b> {doc} · "
        f"<b>日期</b> {DATE} · <b>版本</b> {REV} · <b>状态</b> {STATUS}"
        '<br><b>图例</b> 线框里的框线、间距与圆角是画法，不构成 UI-18 的判据；'
        "规格以 U2 的 token 与框架默认值为准。窗口控件（红绿灯）由系统绘制，"
        "线框只画它占的那条空带。"
        "</div>"
    )


IDENT_CSS = (
    ".ident{margin:0 0 12px;font-size:12.25px;line-height:18px;color:#666;"
    "border:1px dashed #ddd;border-radius:6px;padding:8px 10px;max-width:1000px}"
    ".ident b{color:#333}"
    # 窗口控件让位条：红绿灯落在左栏顶部上方（UI-19）
    ".sb .lights{height:28px;display:flex;align-items:center;gap:8px;padding-left:8px}"
    ".sb .lights i{width:10px;height:10px;border-radius:5px;border:1px solid #999;background:#fff;"
    "display:block}"
    ".ph-icon{display:inline-block;width:16px;height:16px;border:1px dashed #999;border-radius:3px;"
    "vertical-align:-3px}"
    ".dot{display:inline-block;width:8px;height:8px;border-radius:4px;background:#333}"
    ".avatar{display:inline-block;width:24px;height:24px;border-radius:12px;background:#ddd;"
    "text-align:center;line-height:24px;font-size:12.25px;color:#333}"
)


def inject_css(s):
    if ".ident{" in s:
        return s
    return s.replace("</style>", IDENT_CSS + "</style>", 1)


def inject_ident(s, doc):
    if 'class="ident"' in s:
        return s
    # 身份块放在标题之后、第一段说明之前
    m = re.search(r"</h1>", s)
    return s[: m.end()] + ident_block(doc) + s[m.end() :]


def refresh_ident(s):
    """把已有的身份块补到当前口径：加版本、加标题行口径那句（幂等）。"""

    def repl(m):
        text = m.group(0)
        # 版本号随每次重出更新（已是当前版本就保持不变）
        text = re.sub(r"第 \d+ 版（[^）]*）", REV, text)
        if "<b>版本</b>" not in text:
            text = text.replace(" · <b>状态</b>", f" · <b>版本</b> {REV} · <b>状态</b>")
        if "虚线方块是图标占位" not in text:
            text = text.replace(
                "</div>", "</div>", 1
            )
            text = text.replace(
                "系统 Title 不算重复。",
                "系统 Title 不算重复。虚线方块是图标占位：图标由设计决定"
                "（U2-forms §3），线框只约束内容、位置与可点区。",
            )
        if "标题行左端写一次会话名" not in text:
            text = text.replace(
                "线框只画它占的那条空带。",
                "线框只画它占的那条空带。标题行左端写一次会话名（红绿灯让位之后），"
                "系统 Title 不算重复。",
            )
        return text

    return re.sub(r'<div class="ident">.*?</div>', repl, s, flags=re.S)


def inject_lights(s):
    # 左栏顶部让出窗口控件那条；幂等
    if 'class="lights"' in s:
        return s
    return s.replace(
        '<div class="sb"><div class="hd">',
        '<div class="sb"><div class="lights"><i></i><i></i><i></i></div><div class="hd">',
    )


def keep_window_title(s):
    """标题行的左端写一次会话名（界面上唯一的可见标题，红绿灯让位之后）。

    评审 R1 之后口径：系统 Title 与自绘标题行不是"各写一遍可见标题"，而是
    "可见的只写一次、系统那份给窗口管理器"。这里把会话名补回标题行左端。
    """
    if 'class="nm"' in s:
        return s
    return s.replace('<div class="tb"><span class="sp">',
                     '<div class="tb"><span class="nm">修发送路径</span><span class="sp">')


def normalize_text(s):
    """把旧口径的说法换成现行口径（幂等）。"""
    reps = [
        ("时间线按内容上限 784 居中", "时间线左缘对齐、行宽随内容"),
        ("标题行 48 写会话名、分支、权限、模型", "标题行 48 是窗口标题栏（红绿灯在左栏上方），只放右栏入口与偏离默认时的最小提示"),
        ("· 内容 784", "· 内容左缘对齐"),
        ("「做了什么」那一行", "轮次组头"),
        ("<tr><td>「做了什么」行</td>", "<tr><td>轮次组头</td>"),
        ("「做了什么」一行", "轮次组头一行"),
        ("内容上限 784 居中", "左缘对齐、行宽随内容"),
        ("内容 784 居中", "内容左缘对齐"),
    ]
    for old, new in reps:
        s = s.replace(old, new)
    return s


def refresh_tables(s):
    """把线框自带的「行与单元」「控件」表刷到现行口径（幂等）。

    这两张表是线框自带的规格说明；旧口径（组头写计数与时长、标题行常驻
    模型/分支/权限）留在这里，就会在后续评审里被当成有效规格（§9.10 第 4 条）。
    """
    reps = [
        ("<tr><td>「做了什么」折叠</td><td>一句具体内容：运行了 N 个操作 · 改了 N 个文件 · Ns；"
         "带状态标记与行末方向指示（折叠时向右）</td>",
         "<tr><td>组头（折叠）</td><td>一句内容总览：这一组做了什么——动作与对象"
         "（如「读取 composer.go · 改动 composer.go · 执行 go test ./...」）；行首图标表达状态，"
         "行末方向指示指向右；不写状态词、不写计数、不写时长</td>"),
        ("<tr><td>「做了什么」展开</td><td>按发生顺序逐条：序号 · 动作 · 对象 · 结果 · 耗时；"
         "失败条目下加一行原因</td>",
         "<tr><td>组头（展开）</td><td>第一行是计数「N 个操作 · 改了 N 个文件」；其下按发生顺序逐条："
         "序号 · 动作 · 对象 · 结果；失败条目下加一行原因；不写时长</td>"),
        ("<tr><td>模型：默认 ▾</td><td>有会话</td><td>—</td>"
         "<td>model.list / session.agent.update</td></tr>"
         "<tr><td>分支：main</td><td>有会话</td><td>—</td>"
         "<td>session.branches / session.branch_create</td></tr>"
         "<tr><td>权限：工作区内自动</td><td>有会话</td><td>—</td>"
         "<td>session.permission.get / update</td></tr>",
         "<tr><td>偏离默认时的最小提示（分支 / 权限 / 模型）</td>"
         "<td>只在该项偏离默认值时出现；默认值不上屏（UI-9）</td><td>—</td>"
         "<td>session.branches / session.permission.get / model.list</td></tr>"),
        # 方案 B 的操作行：结果词是内容，时长不显示
        ("<tr><td>操作行</td><td>动作 · 对象 · 结果 · 耗时（完成 / 进行中 / 等待你同意 / 失败）</td><td>22.75</td></tr>",
         "<tr><td>操作行</td><td>动作 · 对象 · 结果（完成 / 进行中 / 等待你同意 / 失败）；不写时长</td><td>22.75</td></tr>"),
        # 说明列里的旧口径
        ("折叠时一行，展开时按发生顺序逐条列出（序号、动作、对象、结果、耗时，失败带原因）",
         "折叠时一行写这一组做了什么，展开时按发生顺序逐条列出（序号、动作、对象、结果，失败带原因；不写时长）"),
        ("<tr><td>展开列表</td><td>每条操作 O5：序号、动作、对象、结果、耗时</td></tr>",
         "<tr><td>展开列表</td><td>每条操作 O5：序号、动作、对象、结果（不写时长）</td></tr>"),
        ("<div>标题行<i>会话 O1 名称、分支 O2、权限、模型</i></div>",
         "<div>标题行<i>会话 O1 名称（唯一可见标题）；分支 O2、权限、模型只在偏离默认时以最小形式出现（UI-9）</i></div>"),
        # 区域表里的旧行名
        ("<tr><td>「做了什么」行</td><td>折叠 22.75；展开后每条操作 22.75，失败原因 18</td>",
         "<tr><td>轮次组头</td><td>折叠 22.75；展开后每条操作 22.75，失败原因 18</td>"),
        # 控件表里的字符写法换成图标入口（UI-23）
        ("<tr><td>右栏 ›</td><td>有会话</td><td>—</td><td>本地</td></tr>",
         "<tr><td>右栏入口（图标，读屏名「打开右栏 / 收起右栏」）</td><td>有会话</td><td>—</td><td>本地</td></tr>"),
        ("<tr><td>＋ 附件</td><td>输入区存在</td><td>—</td><td>attachment.put</td></tr>",
         "<tr><td>附件入口（图标，读屏名「附件」）</td><td>输入区存在</td><td>—</td><td>attachment.put</td></tr>"),
        # 方案 A 的开篇偏离台账：不写计数与时长
        ("①一轮里的全部操作收成一行具体摘要（运行了 N 个操作 · 改了 N 个文件 · Ns），可展开成逐条",
         "①一轮里的全部操作收成一行内容总览（这一组做了什么：动作与对象），可展开成逐条"),
    ]
    for old, new in reps:
        s = s.replace(old, new)
    return s


def refresh_words(s):
    """把界面语换成形式（UI-23）：线框里只画"这里是图标"的占位，不画图标本身。

    图标由设计决定（`docs/domains/U2-forms.md` §3），线框只约束内容、位置与可点区。
    图例里已写明这一点；虚线方块就是图标占位。
    """
    icon = '<span class="ph-icon" title="{0}"></span>'
    reps = [
        ('<div class="hd"><span>会话</span><span>＋</span></div>',
         '<div class="hd">' + icon.format("新建会话") + icon.format("搜索") +
         icon.format("收起左栏") + '</div>'),
        ('<button class="btn">右栏 ›</button>', '<button class="btn">' + icon.format("右栏") + '</button>'),
        ('<button class="ghost">＋ 附件</button>', '<button class="ghost">' + icon.format("附件") + '</button>'),
        ('<span class="ctx">上下文 12%</span>', icon.format("上下文用量") + '<span class="ctx">12%</span>'),
        ('<span class="ph">写点什么，或把文件拖进来</span>', ''),
        ('<div class="bot">账号 · 已登录</div>',
         '<div class="bot"><span class="dot" title="连接状态"></span><span class="avatar" title="账号">E</span></div>'),
        ('<span>排队中 1</span>', '<span class="ph-icon" title="排队中"></span>'),
        ('<div class="t">需要你同意：跑一条命令</div>', '<div class="t">' + icon.format("需要你决定") + '</div>'),
        ('<span class="mark">⌘↵ 发送</span>', ''),
        ('<span class="mark">一直允许只对简单命令有效</span>', ''),
    ]
    for old, new in reps:
        s = s.replace(old, new)
    return s


def refresh_tags(s):
    """给已有画面标签补/更新版本号（幂等）。"""
    s = re.sub(
        rf"· {DATE} · 第 \d+ 版 · {STATUS}</div>",
        f"· {DATE} · {REV} · {STATUS}</div>",
        s,
    )
    return s.replace(f"· {DATE} · {STATUS}</div>", f"· {DATE} · {REV} · {STATUS}</div>")


def reemit_tags(s, shots):
    """给每个 .tag 补上它的依据与文档，按画面顺序一一对应。"""
    seen = [0]

    def repl(m):
        text = m.group(1)
        if "依据" in text:
            return m.group(0)
        i = seen[0]
        seen[0] += 1
        if i >= len(shots):
            return m.group(0)
        name, why, doc = shots[i]
        return (
            f'<div class="tag">{text} · 依据 {why} · 文档 {doc} · {DATE} · {STATUS}</div>'
        )

    return re.sub(r'<div class="tag">([^<]*)</div>', repl, s)


def main():
    for name, meta in PAGES.items():
        path = f"docs/domains/wireframes/{name}"
        s = io.open(path, encoding="utf-8").read()
        s = inject_css(s)
        s = normalize_text(s)
        s = inject_ident(s, meta["doc"])
        s = refresh_ident(s)
        s = inject_lights(s)
        s = keep_window_title(s)
        s = reemit_tags(s, meta["shots"])
        s = refresh_tables(s)
        s = refresh_words(s)
        s = refresh_tags(s)
        io.open(path, "w", encoding="utf-8").write(s)
        print("re-emitted", path)

    index()


INDEX = """<!doctype html><html lang="zh-CN"><head><meta charset="utf-8">
<title>U1 B2 线框索引（{date} · {status}）</title><style>
*{{box-sizing:border-box;margin:0;padding:0}}
body{{font:14px/1.5 -apple-system,BlinkMacSystemFont,"PingFang SC","Segoe UI",sans-serif;color:#333;background:#fff;padding:24px}}
h1{{font-size:20px;line-height:28px;margin-bottom:8px}}
h2{{font-size:15px;line-height:22px;margin:24px 0 8px}}
p.lead{{color:#666;margin-bottom:10px;max-width:1000px}}
table{{border-collapse:collapse;margin:8px 0 16px;font-size:12.25px;max-width:1100px}}
th,td{{border:1px solid #ddd;padding:4px 8px;text-align:left;vertical-align:top}}
th{{background:#f5f5f5}}
code{{font-family:ui-monospace,Menlo,monospace;font-size:12px}}
.ident{{margin:0 0 12px;font-size:12.25px;line-height:18px;color:#666;border:1px dashed #ddd;border-radius:6px;padding:8px 10px;max-width:1000px}}
.ident b{{color:#333}}
.strip{{display:flex;gap:12px;flex-wrap:wrap}}
.cv{{width:320px;border:1px solid #ddd;border-radius:6px;padding:12px}}
.cv b{{display:block;margin-bottom:6px}}
</style></head><body>
<h1>U1 B2 线框索引</h1>
<div class="ident"><b>依据</b> {decisions} · <b>文档</b> U1-concepts、U1-ia §1/§5/§7、U2 §4.4/§4.5、U3 §1.1/T7/T10 · <b>日期</b> {date} · <b>版本</b> {rev} · <b>状态</b> {status}<br>
<b>图例</b> 线框里的框线、间距与圆角是画法，不构成 UI-18 的判据；规格以 U2 的 token 与框架默认值为准。线框里的文案、状态、位置、可点区必须与裁决一致（§9.10 第 4 条）。窗口控件由系统绘制，线框只画它占的那条空带。标题行左端写一次会话名（红绿灯让位之后），系统 Title 不算重复。虚线方块是图标占位：图标由设计决定（U2-forms §3），线框只约束内容、位置与可点区。</div>

<h2>画面 × 依据 × 日期 × 状态</h2>
<table>
<tr><th>页</th><th>画面</th><th>依据</th><th>文档</th><th>日期</th><th>状态</th></tr>
{rows}
</table>

<h2>三份线框</h2>
<div class="strip">
 <div class="cv"><b>a.html · 方案 A：轮次组头</b><p class="lead">一轮的全部动作收成一行组头，写这一组做了什么；点内容展开逐条。<br><a href="a.html">看线框 →</a></p></div>
 <div class="cv"><b>b.html · 方案 B：轮次索引</b><p class="lead">长会话左缘一排针，点一下跳到那一轮；操作逐条列出。<br><a href="b.html">看线框 →</a></p></div>
 <div class="cv"><b>ui9-11.html · 修正对照</b><p class="lead">UI-9/10/11 与 UI-15/16 的五个对照画面。<br><a href="ui9-11.html">看对照 →</a></p></div>
</div>

<h2>归档</h2>
<p class="lead"><code>comps/comp1–3.html</code>（2026-11-11 整屏稿）已作废并移入 <code>comps/archive-2026-11-11/</code>：它们引用已删除的 <code>border-strong</code>、网页式卡片与圆角，且没有任何文档引用（§9.10 第 5 条）。</p>
</body></html>
"""


def index():
    rows = []
    pages = [
        ("a.html", "方案 A：轮次组头"),
        ("b.html", "方案 B：轮次索引"),
        ("ui9-11.html", "UI-9/10/11 + UI-15/16 对照"),
    ]
    for page, title in pages:
        for name, why, doc in PAGES[page]["shots"]:
            rows.append(
                f"<tr><td>{page}</td><td>{title} · {name}</td><td>{why}</td>"
                f"<td>{doc}</td><td>{DATE}</td><td>{STATUS}</td></tr>"
            )
    doc = INDEX.format(
        date=DATE,
        rev=REV,
        status=STATUS,
        decisions=DECISIONS,
        rows="\n".join(rows),
    )
    io.open("docs/domains/wireframes/index.html", "w", encoding="utf-8").write(doc)
    print("re-emitted docs/domains/wireframes/index.html")


if __name__ == "__main__":
    main()
