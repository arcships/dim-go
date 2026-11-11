#!/usr/bin/env python3
"""Generate docs/domains/U2-visual-preview.html from the C2 token set."""
from string import Template

LIGHT = dict(
    bg="#ffffff", surface="#f4f4f5", surface_hover="#e9e9ec", surface_pressed="#dddde1",
    surface_selected="#dfe6f4", border_subtle="#e8e8ec", border="#d4d4da", border_strong="#85858b",
    text="#18181b", text_secondary="#5f5f68", accent="#2563eb", accent_hover="#1d4ed8",
    accent_pressed="#1e40af", accent_on="#ffffff", accent_ink="#1d4ed8",
    success="#15803d", warning="#b45309", danger="#b91c1c", focus="#2563eb",
    selection="rgba(37,99,235,0.20)",
)
DARK = dict(
    bg="#18181b", surface="#27272a", surface_hover="#323236", surface_pressed="#3c3c41",
    surface_selected="#2a364b", border_subtle="#2f2f35", border="#3f3f46", border_strong="#8b8b94",
    text="#f4f4f5", text_secondary="#a1a1aa", accent="#2563eb", accent_hover="#1d4ed8",
    accent_pressed="#1e40af", accent_on="#ffffff", accent_ink="#8ab4ff",
    success="#22c55e", warning="#f59e0b", danger="#f87171", focus="#60a5fa",
    selection="rgba(59,130,246,0.30)",
)


def fmt(v):
    s = ("%.2f" % v).rstrip("0").rstrip(".")
    return s


def lin(c):
    c = c / 255
    return c / 12.92 if c <= 0.04045 else ((c + 0.055) / 1.055) ** 2.4


def lum(h):
    h = h.lstrip("#")
    r, g, b = (int(h[i:i + 2], 16) for i in (0, 2, 4))
    return 0.2126 * lin(r) + 0.7152 * lin(g) + 0.0722 * lin(b)


def cr(a, b):
    la, lb = lum(a), lum(b)
    hi, lo = max(la, lb), min(la, lb)
    return (hi + 0.05) / (lo + 0.05)


def over(fg, alpha, bg):
    f, b = fg.lstrip("#"), bg.lstrip("#")
    return "#" + "".join("%02x" % round(int(f[i:i + 2], 16) * alpha + int(b[i:i + 2], 16) * (1 - alpha)) for i in (0, 2, 4))


# ---- token descriptions ---------------------------------------------------
FILL = {
    "bg": "窗口底",
    "surface": "卡片 / 气泡 / 输入卡 / 行底",
    "surface-hover": "悬停面",
    "surface-pressed": "按下态",
    "surface-selected": "选中行底",
    "border-subtle": "分隔线",
    "border": "控件轮廓",
    "border-strong": "单独承担识别（≥3:1）",
    "accent": "强调填充（主按钮）",
    "accent-hover": "强调填充悬停",
    "accent-pressed": "强调填充按下",
}
INK = {
    "text": "正文",
    "text-secondary": "次要文本",
    "accent-ink": "强调色作文字 / 图标 / 竖条",
    "accent-on": "落在强调填充上的文字",
    "success": "成功状态",
    "warning": "警告状态",
    "danger": "失败状态",
    "focus": "焦点环",
    "selection": "文本选区",
}

ORDER = ["bg", "surface", "surface-hover", "surface-pressed", "surface-selected",
         "border-subtle", "border", "border-strong", "text", "text-secondary",
         "accent", "accent-hover", "accent-pressed", "accent-on", "accent-ink",
         "success", "warning", "danger", "selection", "focus"]

VARNAME = {k: "--" + k for k in ORDER}

TYPE = [
    ("type-body", "正文", 14, "22.75px", 400, "助手回复、用户消息、Markdown、输入、说明行", "我把重试逻辑抽到了 retry.ts，并补了一个测试。"),
    ("type-body-strong", "正文强调", 14, "22.75px", 600, "Markdown 粗体、对象名", "重试逻辑"),
    ("type-secondary", "次要", 12.25, "18px", 400, "摘要行、工具行、时间、用量、按钮", "运行了 4 个操作 · 改了 2 个文件 · 12s"),
    ("type-secondary-strong", "次要强调", 12.25, "18px", 600, "状态词、键位提示", "等待你同意"),
    ("type-micro", "极小", 10.5, "14px", 500, "计数、徽标、编号", "+12 −3 · 4.2 KB"),
    ("type-code", "代码", 13, "20px", 400, "代码块、行内代码、命令", "go test ./..."),
    ("type-title-row", "标题行会话名", 14, "20px", 600, "标题行", "payments 重试逻辑"),
    ("type-h1", "h1", 20, "26px", 600, "Markdown 一级标题", "重试与超时"),
    ("type-h2", "h2", 17.5, "24px", 600, "Markdown 二级标题 / 设置节标题", "重试与超时"),
    ("type-h3", "h3", 15.75, "22px", 600, "Markdown 三级标题", "重试与超时"),
    ("type-h4", "h4", 14, "20px", 600, "Markdown 四级标题", "重试与超时"),
]


def theme_vars(t):
    out = []
    for k in ORDER:
        out.append("%s:%s" % (VARNAME[k], t[k.replace("-", "_")]))
    return "".join(s + ";" for s in out)


def swatches(t, theme_name):
    rows = []
    for k in ORDER:
        name = k.replace("_", "-")
        v = t[k.replace("-", "_")]
        if k in INK:
            if k == "accent-on":
                chip = '<span class="ink" style="background:%s;color:%s">Aa</span>' % (t["accent"], t["accent_on"])
                note = "对 accent 5.17 / 对 hover 6.70"
            elif k == "selection":
                sbg = over("#2563eb" if theme_name == "明" else "#3b82f6", 0.20 if theme_name == "明" else 0.30, t["bg"])
                chip = '<span class="ink" style="background:%s;color:%s">Aa 选区</span>' % (sbg, t["text"])
                note = "选区合成 %s · 正文 %.2f" % (sbg, cr(t["text"], sbg))
            elif k == "focus":
                chip = ('<span class="ringwrap"><span class="ring" style="box-shadow:0 0 0 1px %s inset, 0 0 0 3px %s"></span></span>'
                        % (t["surface"], v))
                note = "对 bg %.2f · 对 surface %.2f" % (cr(v, t["bg"]), cr(v, t["surface"]))
            else:
                chip = '<span class="ink" style="background:%s;color:%s">Aa</span>' % (t["surface"], v)
                note = "对 bg %.2f · 对 surface %.2f" % (cr(v, t["bg"]), cr(v, t["surface"]))
        else:
            chip = '<span class="fill" style="background:%s"></span>' % v
            note = "对 bg %.2f · 对 surface %.2f" % (cr(v, t["bg"]), cr(v, t["surface"]))
        rows.append(
            '<div class="sw"><div class="chipbox">%s</div><div class="meta"><b>%s</b><code>%s</code>'
            '<span class="ratio">%s</span></div></div>'
            % (chip, name, v, note)
        )
    return "\n".join(rows)


def contrast_rows():
    checks = [
        ("正文 / bg", "text", "bg", 4.5), ("正文 / surface", "text", "surface", 4.5),
        ("次要 / surface", "text_secondary", "surface", 4.5), ("次要 / surface-hover", "text_secondary", "surface_hover", 4.5),
        ("次要 / surface-selected", "text_secondary", "surface_selected", 4.5),
        ("强调作文字 / surface", "accent_ink", "surface", 4.5),
        ("白字 / accent", "accent_on", "accent", 4.5),
        ("success 作文字 / surface", "success", "surface", 4.5),
        ("warning 作文字 / surface", "warning", "surface", 4.5),
        ("danger 作文字 / surface", "danger", "surface", 4.5),
        ("焦点环 / surface", "focus", "surface", 3.0),
        ("焦点环 / surface-hover", "focus", "surface_hover", 3.0),
        ("border-strong / surface", "border_strong", "surface", 3.0),
        ("选中条 accent-ink / surface-selected", "accent_ink", "surface_selected", 3.0),
    ]
    rows = []
    for label, a, b, target in checks:
        lv, dv = cr(LIGHT[a], LIGHT[b]), cr(DARK[a], DARK[b])
        ok = lv >= target - 1e-9 and dv >= target - 1e-9
        rows.append('<tr><td>%s</td><td>%.2f</td><td>%.2f</td><td>≥%.1f</td><td class="%s">%s</td></tr>'
                    % (label, lv, dv, target, "ok" if ok else "bad", "通过" if ok else "不达"))
    for label, a, b in [("accent 填充 / 暗 surface（例外：有文字标签）", "accent", "surface")]:
        rows.append('<tr><td>%s</td><td>%.2f</td><td>%.2f</td><td>≥3.0</td><td class="warn">例外</td></tr>'
                    % (label, cr(LIGHT[a], LIGHT[b]), cr(DARK[a], DARK[b])))
    return "\n".join(rows)


CSS_VARS = """:root{
%s
  --space-1:3.5px; --space-2:7px; --space-3:10.5px; --space-4:14px;
  --space-6:21px; --space-8:28px; --space-12:42px;
  --row-regular:28px; --row-compact:24.5px; --control-mini:21px;
  --icon-inline:16px; --icon-block:20px; --dot-status:8px;
  --stroke-hairline:1px; --stroke-focus:2px; --stroke-marker:2px; --hit-min:24px;
  --radius-sm:4px; --radius-control:6px; --radius-card:8px; --radius-panel:12px;
  --motion-fast:120ms; --motion-base:200ms; --motion-slow:320ms;
  --reveal-catchup:400ms; --reveal-fade:120ms; --caret-cycle:1060ms; --tooltip-delay:600ms;
}
:root[data-theme="dark"]{
%s
}
@media (prefers-color-scheme: dark){
  :root:not([data-theme]){%s}
}""" % (theme_vars(LIGHT), theme_vars(DARK), theme_vars(DARK))

ICONS = {
    "paperclip": '<path d="M18.375 12.739l-7.693 7.693a4.5 4.5 0 01-6.364-6.364l10.94-10.94A3 3 0 1119.5 7.372L8.552 18.32m.009-.01l-.01.01m5.699-9.941l-7.81 7.81a1.5 1.5 0 002.112 2.13"/>',
    "chevron-down": '<path d="M19.5 8.25l-7.5 7.5-7.5-7.5"/>',
    "x": '<path d="M6 18L18 6M6 6l12 12"/>',
    "exclamation": '<path d="M12 9v3.75m-9.303 3.376c-.866 1.5.217 3.374 1.948 3.374h14.71c1.73 0 2.813-1.874 1.948-3.374L13.949 3.378c-.866-1.5-3.032-1.5-3.898 0L2.697 16.126zM12 15.75h.007v.008H12v-.008z"/>',
    "clock": '<path d="M12 6v6h4.5m4.5 0a9 9 0 11-18 0 9 9 0 0118 0z"/>',
    "check": '<path d="M4.5 12.75l6 6 9-13.5"/>',
    "arrow-up": '<path d="M12 19.5v-15m0 0l-6.75 6.75M12 4.5l6.75 6.75"/>',
    "stop": '<path d="M5.25 7.5A2.25 2.25 0 017.5 5.25h9a2.25 2.25 0 012.25 2.25v9a2.25 2.25 0 01-2.25 2.25h-9A2.25 2.25 0 015.25 16.5v-9z"/>',
    "chat": '<path d="M8.625 12a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H8.25m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0H12m4.125 0a.375.375 0 11-.75 0 .375.375 0 01.75 0zm0 0h-.375M21 12c0 4.556-4.03 8.25-9 8.25a9.764 9.764 0 01-2.555-.337A5.972 5.972 0 015.41 20.97a5.969 5.969 0 01-.474-.065 4.48 4.48 0 00.978-2.025c.09-.457-.133-.901-.467-1.226C3.93 16.178 3 14.189 3 12c0-4.556 4.03-8.25 9-8.25s9 3.694 9 8.25z"/>',
}


def icon(name, size, cls=""):
    return ('<svg class="ico %s" width="%s" height="%s" viewBox="0 0 24 24" fill="none" '
            'stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round" '
            'aria-hidden="true">%s</svg>' % (cls, size, size, ICONS[name]))


def build():
    type_rows = []
    for name, label, size, lh, w, use, sample in TYPE:
        type_rows.append(
            '<tr><td><code>%s</code></td><td>%s</td><td>%s</td><td>%d</td><td>%s</td>'
            '<td class="sample" style="font-size:%spx;line-height:%s;font-weight:%d">%s</td></tr>'
            % (name, fmt(size), lh, w, use, size, lh, w, sample)
        )
    type_rows = "\n".join(type_rows)

    type_demo = "\n".join(
        '<div class="type-demo"><span class="spec">%s · %spx / %s / %d · %s</span>'
        '<p style="font-size:%spx;line-height:%s;font-weight:%d">%s</p></div>'
        % (name, fmt(size), lh, w, use, size, lh, w, sample) for name, _, size, lh, w, use, sample in TYPE
    )

    spaces = [(1, 3.5), (2, 7), (3, 10.5), (4, 14), (6, 21), (8, 28), (12, 42)]
    space_rows = "\n".join(
        '<div class="spacerow"><code>space-%d</code><span class="bar" style="width:%spx"></span><span class="val">%spx</span></div>'
        % (n, v, v) for n, v in spaces)

    icon_rows = "\n".join(
        '<div class="iconrow"><span class="ibox" style="width:%dpx;height:%dpx">%s</span>'
        '<code>%s · %dpx</code></div>' % (s, s, icon(n, s), n, s)
        for n, s in [("arrow-up", 16), ("stop", 16), ("paperclip", 16), ("chevron-down", 16),
                     ("x", 16), ("exclamation", 16), ("clock", 16), ("check", 16),
                     ("x", 20)])

    token_rows = []
    for k in ORDER:
        token_rows.append("<tr><td><code>%s</code></td><td><code>%s</code></td><td><code>%s</code></td><td>%s</td></tr>"
                          % (k.replace("_", "-"), LIGHT[k.replace("-", "_")], DARK[k.replace("-", "_")], FILL.get(k) or INK.get(k, "")))
    for k in ("inverse", "inverse-text", "scrollbar"):
        pass
    token_rows.append("<tr><td><code>inverse</code></td><td><code>#18181b</code></td><td><code>#f4f4f5</code></td><td>沿用框架：提示与 toast</td></tr>")
    token_rows.append("<tr><td><code>inverse-text</code></td><td><code>#ffffff</code></td><td><code>#18181b</code></td><td>同上</td></tr>")
    token_rows.append("<tr><td><code>scrollbar</code></td><td><code>rgba(0,0,0,0.32)</code></td><td><code>rgba(255,255,255,0.35)</code></td><td>沿用框架</td></tr>")
    token_rows = "\n".join(token_rows)

    doc = Template(HTML).substitute(
        css_vars=CSS_VARS,
        light_vars=theme_vars(LIGHT),
        dark_vars=theme_vars(DARK),
        light_swatches=swatches(LIGHT, "明"),
        dark_swatches=swatches(DARK, "暗"),
        type_rows=type_rows,
        type_demo=type_demo,
        space_rows=space_rows,
        icon_rows=icon_rows,
        token_rows=token_rows,
        contrast_rows=contrast_rows(),
        icon_upload=icon("arrow-up", 16), icon_stop=icon("stop", 16),
        icon_clip=icon("paperclip", 16), icon_chev=icon("chevron-down", 16),
        icon_x=icon("x", 16), icon_warn=icon("exclamation", 16),
        icon_clock=icon("clock", 16), icon_check=icon("check", 16),
        icon_block=icon("chat", 20),
    )
    open("docs/domains/U2-visual-preview.html", "w").write(doc)
    print("wrote docs/domains/U2-visual-preview.html", len(doc), "bytes")


HTML = r"""<!doctype html>
<html lang="zh-CN">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>U2 C2 视觉 token 预览</title>
<style>
${css_vars}
*{box-sizing:border-box}
html,body{margin:0}
body{
  background:var(--bg);color:var(--text);
  font-family:-apple-system,BlinkMacSystemFont,"PingFang SC","Hiragino Sans GB","Segoe UI",sans-serif;
  font-size:14px;line-height:22.75px;
  padding:0 0 96px;
}
body[data-motion="off"]{--motion-fast:0ms;--motion-base:0ms;--motion-slow:0ms;--reveal-fade:0ms}
@media (prefers-reduced-motion: reduce){
  body{--motion-fast:0ms;--motion-base:0ms;--motion-slow:0ms;--reveal-fade:0ms}
  .demo-card{transition:none !important}
}
code{font-family:"SF Mono",Menlo,monospace;font-size:13px;line-height:20px}
.wrap{max-width:1180px;margin:0 auto;padding:0 24px}
header.topbar{
  background:var(--surface);border-bottom:1px solid var(--border-subtle);
  padding:7px 0;margin-bottom:21px;
}
header.topbar .wrap{display:flex;align-items:center;gap:14px;flex-wrap:wrap}
h1{font-size:20px;line-height:26px;font-weight:600;margin:0}
.sub{font-size:12.25px;line-height:18px;color:var(--text-secondary)}
.seg{display:flex;gap:3.5px;margin-left:auto}
.seg button{
  font:inherit;font-size:12.25px;line-height:18px;height:28px;padding:0 10.5px;
  border-radius:6px;border:1px solid var(--border);background:var(--surface);color:var(--text);cursor:pointer;
}
.seg button.on{background:var(--accent);border-color:var(--accent);color:var(--accent-on)}
.seg button:focus-visible{outline:2px solid var(--focus);outline-offset:1px}
label.chk{display:flex;align-items:center;gap:7px;font-size:12.25px;line-height:18px;color:var(--text-secondary);cursor:pointer}
h2{font-size:17.5px;line-height:24px;font-weight:600;margin:42px 0 10.5px;padding-top:10.5px;border-top:1px solid var(--border-subtle)}
h3{font-size:15.75px;line-height:22px;font-weight:600;margin:21px 0 7px}
p.note{font-size:12.25px;line-height:18px;color:var(--text-secondary);margin:0 0 10.5px;max-width:860px}
table{border-collapse:collapse;width:100%;margin:7px 0 14px;font-size:12.25px;line-height:18px}
th,td{text-align:left;padding:5.25px 10.5px;border-bottom:1px solid var(--border-subtle);vertical-align:top}
th{font-weight:600;color:var(--text-secondary);white-space:nowrap}
td.sample{white-space:nowrap;overflow:hidden;text-overflow:ellipsis;max-width:340px}
.ok{color:var(--success);font-weight:600}
.bad{color:var(--danger);font-weight:600}
.warn{color:var(--warning);font-weight:600}
/* tokens */
.palettes{display:grid;grid-template-columns:1fr 1fr;gap:21px}
.panel{background:var(--bg);color:var(--text);border:1px solid var(--border);border-radius:8px;padding:14px}
.t-light,.t-dark{color:var(--text)}
.t-light{${light_vars}}
.t-dark{${dark_vars}}
.panel h3{margin-top:0}
.sw{display:flex;gap:10.5px;align-items:center;padding:3.5px 0;border-bottom:1px solid var(--border-subtle)}
.chipbox{width:56px;flex:0 0 auto}
.fill,.ink{display:block;width:56px;height:28px;border-radius:4px;border:1px solid var(--border-subtle);font-size:12.25px;line-height:26px;text-align:center}
.ink{font-family:-apple-system,sans-serif}
.ringwrap{display:block;width:56px;height:28px;display:flex;align-items:center;justify-content:center}
.ring{display:block;width:34px;height:18px;border-radius:4px;background:var(--surface)}
.meta{display:flex;flex-direction:column;font-size:12.25px;line-height:16px}
.meta b{font-weight:500}
.meta code{font-size:11px}
.meta span{color:var(--text-secondary)}
.meta .ratio{font-size:11px}
/* spacing */
.spacerow{display:flex;align-items:center;gap:10.5px;margin:3.5px 0}
.spacerow code{width:76px;color:var(--text-secondary);font-size:12.25px}
.bar{display:block;height:10.5px;background:var(--accent);border-radius:2px}
.spacerow .val{font-size:12.25px;color:var(--text-secondary)}
/* type demo */
.type-demo{padding:7px 0;border-bottom:1px solid var(--border-subtle)}
.type-demo p{margin:0;max-width:820px}
.spec{display:block;font-size:10.5px;line-height:14px;color:var(--text-secondary);font-family:"SF Mono",Menlo,monospace}
/* demo frame */
.frame{border:1px solid var(--border);border-radius:8px;padding:14px;background:var(--bg);max-width:820px}
.thread{display:flex;flex-direction:column;gap:10.5px}
.u{display:flex;justify-content:flex-end}
.bubble{max-width:85%;background:var(--surface);border:1px solid var(--border);border-radius:8px;padding:7px 14px;font-size:14px;line-height:22.75px}
.md{font-size:14px;line-height:22.75px}
.md p{margin:0 0 10.5px}
.md h4{font-size:14px;line-height:20px;font-weight:600;margin:14px 0 3.5px}
.md code.inline{background:var(--surface);border:1px solid var(--border-subtle);border-radius:4px;padding:1px 3.5px;font-size:13px}
.codeblock{background:var(--surface);border:1px solid var(--border-subtle);border-radius:8px;padding:10.5px;margin:10.5px 0;font-family:"SF Mono",Menlo,monospace;font-size:13px;line-height:20px;color:var(--text);white-space:pre;overflow:auto}
.summary{display:flex;align-items:center;gap:7px;font-size:12.25px;line-height:18px;color:var(--text-secondary);padding:3.5px 0}
.summary .state{font-weight:600}
.tool{display:flex;align-items:center;gap:7px;min-height:24.5px;font-size:12.25px;line-height:18px;padding-left:10.5px;border-left:1px solid var(--border);color:var(--text-secondary)}
.notice{display:flex;align-items:flex-start;gap:7px;font-size:14px;line-height:22.75px;padding:3.5px 0}
.dot{width:8px;height:8px;border-radius:4px;background:var(--accent-ink);flex:0 0 auto;margin-top:7px}
.caret{display:inline-block;width:2px;height:14px;background:var(--accent-ink);vertical-align:-2px;margin-left:2px;animation:blink var(--caret-cycle) steps(1,end) infinite}
@keyframes blink{0%,49%{opacity:1}50%,100%{opacity:0}}
body[data-motion="off"] .caret{animation:none}
.approval{background:var(--surface);border:1px solid var(--accent);border-radius:8px;padding:14px;display:flex;flex-direction:column;gap:10.5px}
.approval .k{font-size:12.25px;line-height:18px;font-weight:600;color:var(--text-secondary)}
.approval .what{font-size:14px;line-height:22.75px}
.approval .why{font-size:12.25px;line-height:18px;color:var(--text-secondary)}
.actions{display:flex;gap:7px;align-items:center}
button.primary{
  display:inline-flex;align-items:center;gap:7px;
  font:inherit;font-size:12.25px;line-height:18px;height:28px;padding:0 10.5px;border-radius:6px;
  border:1px solid var(--accent);background:var(--accent);color:var(--accent-on);cursor:pointer;
  transition:background var(--motion-fast) ease-out;
}
button.primary:hover{background:var(--accent-hover);border-color:var(--accent-hover)}
button.primary:active{background:var(--accent-pressed);border-color:var(--accent-pressed)}
button.primary:focus-visible{outline:2px solid var(--focus);outline-offset:1px}
button.ghost{
  display:inline-flex;align-items:center;gap:7px;
  font:inherit;font-size:12.25px;line-height:18px;height:28px;padding:0 10.5px;border-radius:6px;
  border:1px solid var(--border);background:var(--surface);color:var(--text);cursor:pointer;
  transition:background var(--motion-fast) ease-out;
}
button.ghost:hover{background:var(--surface-hover)}
button.ghost:focus-visible{outline:2px solid var(--focus);outline-offset:1px}
button.ghost[disabled]{opacity:.5;cursor:default}
.iconbtn{display:inline-flex;align-items:center;justify-content:center;width:28px;height:28px;border-radius:6px;border:1px solid transparent;background:transparent;color:var(--text);cursor:pointer}
.iconbtn:hover{background:var(--surface-hover)}
.iconbtn:focus-visible{outline:2px solid var(--focus);outline-offset:1px}
.queue{display:flex;align-items:center;gap:7px;height:32px;padding:0 14px;border:1px solid var(--border-subtle);border-radius:6px;background:var(--surface);font-size:12.25px;line-height:18px;color:var(--text-secondary)}
.composer{border:1px solid var(--border);border-radius:12px;background:var(--surface);padding:10.5px}
.composer .editor{font-size:14px;line-height:22.75px;min-height:44px;color:var(--text)}
.composer .editor.focus{border-color:var(--border-strong)}
.composer .bar2{display:flex;align-items:center;gap:7px;padding-top:7px}
.chip{display:inline-flex;align-items:center;gap:3.5px;height:21px;padding:0 7px;border-radius:4px;border:1px solid var(--border);background:var(--surface);font-size:10.5px;line-height:14px;color:var(--text-secondary)}
.spacer{flex:1}
.rowdemo{display:flex;flex-direction:column;gap:0;max-width:520px;border:1px solid var(--border-subtle);border-radius:8px;overflow:hidden}
.rowdemo .row{display:flex;align-items:center;gap:7px;height:28px;padding:0 10.5px;font-size:12.25px;line-height:18px;border-bottom:1px solid var(--border-subtle)}
.rowdemo .row:last-child{border-bottom:0}
.rowdemo .row.hover{background:var(--surface-hover)}
.rowdemo .row.selected{background:var(--surface-selected);box-shadow:inset 2px 0 0 var(--accent-ink)}
.rowdemo .row.focused{outline:2px solid var(--focus);outline-offset:-2px}
.rowdemo .row.disabled{opacity:.5}
.rowdemo .row .muted{color:var(--text-secondary)}
.empty{display:flex;align-items:center;gap:7px;color:var(--text-secondary);font-size:12.25px;line-height:18px}
.ico{display:block;flex:0 0 auto}
.ibox{display:flex;align-items:center;justify-content:center;border:1px dashed var(--border);border-radius:4px;color:var(--text)}
.iconrow{display:flex;align-items:center;gap:10.5px;margin:3.5px 0}
.iconrow code{color:var(--text-secondary);font-size:12.25px}
/* motion */
.demo-card{border:1px solid var(--border);border-radius:8px;background:var(--surface);padding:0 14px;overflow:hidden;transition:height var(--motion-base) ease-out;height:32px}
.demo-card.open{height:104px}
.demo-card p{margin:7px 0;font-size:12.25px;line-height:18px;color:var(--text-secondary)}
.demo-card .body{opacity:0;transition:opacity var(--motion-base) ease-out}
.demo-card.open .body{opacity:1}
.typing{font-size:14px;line-height:22.75px;min-height:45.5px;margin:10.5px 0 0;max-width:640px}
.typing span{animation:fadein var(--reveal-fade) linear both}
@keyframes fadein{from{opacity:0}to{opacity:1}}
body[data-motion="off"] .typing span{animation:none}
pre.copy{background:var(--surface);border:1px solid var(--border-subtle);border-radius:8px;padding:10.5px;overflow:auto;font-size:11px;line-height:16px;color:var(--text-secondary);white-space:pre-wrap;word-break:break-word}
.foot{margin-top:42px;font-size:10.5px;line-height:14px;color:var(--text-secondary)}
@media (max-width:900px){.palettes{grid-template-columns:1fr}}
</style>
</head>
<body>
<header class="topbar"><div class="wrap">
  <div>
    <h1>U2 C2 · 视觉 token 预览</h1>
    <div class="sub">预览只示样式，不代表布局与控件位置（布局归 U1/U3）。数值与 <code>docs/domains/U2-visual.md</code> 一致。</div>
  </div>
  <div class="seg">
    <button data-theme-btn="system" class="on">跟随系统</button>
    <button data-theme-btn="light">明</button>
    <button data-theme-btn="dark">暗</button>
    <label class="chk"><input type="checkbox" id="motion"> 减少动效</label>
  </div>
</div></header>

<div class="wrap">
<p class="note">判定方式：数值对 §8 的 token 表；对比度按 WCAG 2.1 相对亮度公式重算；每个区块下方给出该区块的可量属性。所有尺寸是最小可复现单位，缩放窗口不改变任何数值。</p>

<h2 id="s1">1 字阶 · 7 个字号 / 3 个字重 / 11 个语义级</h2>
<table>
<thead><tr><th>token</th><th>字号</th><th>行高</th><th>字重</th><th>用在哪</th><th>样张</th></tr></thead>
<tbody>
${type_rows}
</tbody></table>
<h3>逐级样张（同一句话，按级渲染）</h3>
${type_demo}

<h2 id="s2">2 间距 · 基础单位 3.5</h2>
<p class="note">只用七档；5.25 / 17.5 是尺寸（缩进、行高），不是间距。段距 10.5 = 段间 33.25 ÷ 行间 22.75 ≈ 1.46。</p>
${space_rows}
<p class="note">样式表中的用法：图标与文字 7；标记与文字 3.5；Markdown 段间 10.5；同一轮内块间 10.5；轮次间 21；卡片内边距 14；输入卡 10.5；队列条与输入卡之间 7；内容左右 14。</p>

<h2 id="s3">3 颜色语义 · 每主题 20 个 token</h2>
<p class="note">两套同名同角色。强调色只用于可操作 / 当前 / 进行中；成功 / 警告 / 危险只用于状态，只作文字、图标与 1px 描边。</p>
<div class="palettes">
  <div class="panel t-light"><h3>明色</h3>
${light_swatches}
  </div>
  <div class="panel t-dark"><h3>暗色</h3>
${dark_swatches}
  </div>
</div>

<h2 id="s4">4 状态样式（只改已有元素的属性）</h2>
<p class="note">每条状态落在 U1 C1 §5 指定的既有元素上；不新增容器。每个状态至少两项差异（颜色 + 形态 / 文字 / 位置）。优先级：待你决定 &gt; 结果不确定 &gt; 失败 &gt; 运行中 &gt; 可用。</p>

<h3>4.1 运行中</h3>
<div class="frame"><div class="thread">
  <div class="md"><p>我先看一下现在的重试实现，再把公共部分抽出来。<span class="caret"></span></p></div>
  <div class="summary"><span class="dot"></span><span class="state" style="color:var(--text)">正在运行</span><span>· 12s</span></div>
  <div class="tool">${icon_chev}运行测试 · payments/retry</div>
</div></div>
<p class="spec">8px 实心圆 accent-ink · 摘要行 12.25/18 text-secondary · 状态词 600 · 行尾光标 2×14 accent-ink，周期 1060ms · 发送键文字变「停止」，填充不变</p>

<h3>4.2 失败</h3>
<div class="frame"><div class="thread">
  <div class="summary">${icon_x}<span class="state">失败</span><span>· 2.1s</span><span>· 4 个操作 · 1 个失败</span></div>
  <div class="tool">${icon_x}运行测试 · payments/retry</div>
  <div class="notice">${icon_x}<span><b style="font-weight:600">原因</b>：权限不足，命令没有执行。</span></div>
</div></div>
<p class="spec">叉图标 danger 16px · 摘要行文字 text-secondary（对 surface 5.75:1）· 原因正文 text，不染红、不加底色、不加重试入口</p>

<h3>4.3 等待你同意（替换输入位，不与输入并存 · UI-1）</h3>
<div class="frame"><div class="thread">
  <div class="summary">${icon_clock}<span class="state">等待你同意</span><span>· 1 个操作</span></div>
  <div class="approval">
    <div class="k">要执行</div>
    <div class="what">运行 <code class="inline">go test ./...</code>，会读取当前工作区并写入缓存目录。</div>
    <div class="codeblock">go test ./...</div>
    <div class="actions">
      <button class="primary">同意</button>
      <button class="ghost">拒绝</button>
      <button class="ghost">一直允许</button>
    </div>
  </div>
</div></div>
<p class="spec">卡面 surface · 1px accent 描边 · 圆角 8 · 内边距 14 · 命令 code 13/20 · 主按钮 accent/accent-on 28 高 · 摘要行时钟图标 accent-ink</p>

<h3>4.4 结果不确定</h3>
<div class="frame"><div class="thread">
  <div class="md"><p>已把重试逻辑抽到 <code class="inline">retry.ts</code>。</p></div>
  <div class="notice">${icon_warn}<span>这条消息是否已经发出还不确定，正在核对。</span></div>
</div></div>
<p class="spec">叹号图标 warning 16px · 说明行 14/22.75 text · 属于本轮，不是容器 · 不给重试、不给核对按钮</p>

<h3>4.5 不可用</h3>
<div class="frame"><div class="thread">
  <div class="composer" style="opacity:.5">
    <div class="editor">写点什么…</div>
    <div class="bar2"><span class="chip">技能</span><span class="spacer"></span><button class="ghost" disabled>${icon_upload}发送</button></div>
  </div>
  <div class="notice">${icon_warn}<span>连接已断开，输入暂不可用；恢复后内容还在。</span></div>
</div></div>
<p class="spec">整块不透明度 .5 · 不响应悬停与聚焦 · 禁用按钮 surface + 1px border + text-secondary · 原因一行 12.25/18</p>

<h3>4.6 选中 / 悬停 / 聚焦（同一行元素的三种属性变化）</h3>
<div class="rowdemo">
  <div class="row"><span>payments 重试逻辑</span><span class="spacer"></span><span class="muted">12:04</span></div>
  <div class="row hover"><span>payments 重试逻辑（悬停）</span><span class="spacer"></span><span class="muted">12:04</span></div>
  <div class="row selected"><span>payments 重试逻辑（选中）</span><span class="spacer"></span><span class="muted">12:04</span></div>
  <div class="row focused"><span>payments 重试逻辑（键盘聚焦）</span><span class="spacer"></span><span class="muted">12:04</span></div>
  <div class="row disabled"><span>payments 重试逻辑（不可用）</span><span class="spacer"></span><span class="muted">12:04</span></div>
</div>
<p class="spec">悬停：底 surface → surface-hover，120ms，文字不变、不位移 · 选中：底 surface-selected + 2px accent-ink 竖条 · 聚焦：2px 焦点环外偏 1px，不参与布局 · 不可用：.5，悬停不点亮</p>

<h3>4.7 输入区、队列条、发送 / 停止</h3>
<div class="frame"><div class="thread">
  <div class="queue">${icon_clock}排队中 · 第 1 条：再看一下 changes 面板</div>
  <div class="composer">
    <div class="editor focus">把重试逻辑抽出来，并补一个测试<span class="caret"></span></div>
    <div class="bar2">
      <button class="iconbtn">${icon_clip}</button>
      <span class="chip">技能 · review</span>
      <span class="chip">retry.ts · 4.2 KB</span>
      <span class="spacer"></span>
      <button class="primary">${icon_stop}停止</button>
    </div>
  </div>
</div></div>
<p class="spec">输入卡 surface + 1px border，圆角 12，内边距 10.5 · 编辑器 14/22.75 · 聚焦时边框转 border-strong 并加焦点环 · 队列条 12.25/18，贴输入卡上方 7 · 按钮 28 高、12.25、图标 16</p>

<h3>4.8 空态</h3>
<div class="frame"><div class="thread">
  <div class="empty">${icon_block}这个会话还没有消息。写第一条，或先从左边挑一个会话。</div>
</div></div>
<p class="spec">20px 图标 + 12.25/18 text-secondary 一行 · 不用大字号、不画假内容</p>

<h2 id="s5">5 动效 · 三档</h2>
<table>
<thead><tr><th>token</th><th>时长</th><th>曲线</th><th>用途</th></tr></thead>
<tbody>
<tr><td><code>motion-fast</code></td><td>120ms</td><td>EaseOut</td><td>颜色、背景、描边、不透明度、焦点环、逐字淡入</td></tr>
<tr><td><code>motion-base</code></td><td>200ms</td><td>EaseOut</td><td>出现 / 消失、折展、队列条进出</td></tr>
<tr><td><code>motion-slow</code></td><td>320ms</td><td>EaseInOut</td><td>区域级：审批卡替换输入区、右栏开合</td></tr>
</tbody></table>
<p class="note">逐字：单位 = 词（中文按字素），最多落后 400ms 追平，每单位淡入 120ms，只改不透明度；严格保序（UI-4）。光标周期 1060ms。首片不用弹簧。任何状态过渡 ≤ 320ms。</p>
<div class="demo-card" id="card"><p>折展样张 · 200ms ease-out · 点按钮观察高度过渡</p><p class="body">展开后追加的内容：高度从 32 过渡到 104，只有高度与不透明度在动，没有位移、没有回弹。</p></div>
<div class="actions" style="margin-top:7px">
  <button class="ghost" id="toggle">展开 / 收起</button>
  <button class="ghost" id="type">模拟逐字出现</button>
</div>
<p class="typing" id="typing"></p>
<p class="spec">减少动效开启后：三档时长变 0，逐字整段出现，光标常亮，布局与状态不变（UI-5）。</p>

<h2 id="s6">6 图标与密度</h2>
<p class="note">线性、单色、24 网格、描边 1.5；渲染 16 → 1.0px，20 → 1.25px。界面不用实心图标；8px 状态点由方形圆角绘制，不是图标。来源：Heroicons v2.2.0（MIT）；供应商标志用 ailogo，仅作识别。</p>
${icon_rows}
<p class="note">行高与图标：图标盒 ≤ 行高 − 7。28 行 → 16（余 6 / 6）；24.5 行 → 16（余 4.25）；21 行 → 只用文字级标记。图标不单独落行，行盒内光学居中；16px 图标按钮命中区 ≥ 24×24。</p>

<h2 id="s7">7 可访问性 · 对比度矩阵</h2>
<table>
<thead><tr><th>组合</th><th>明</th><th>暗</th><th>目标</th><th>结论</th></tr></thead>
<tbody>
${contrast_rows}
</tbody></table>
<p class="note">焦点环 2px + 外偏 1px、不参与布局，只在键盘导航出现；交互目标 ≥ 24×24；文本缩放 100–200% 时行高按倍数放大、状态文字不截断；高对比度模式下 border 取 border-strong、text-secondary 取 text、环加宽到 3px。</p>

<h2 id="s8">8 token 清单</h2>
<table>
<thead><tr><th>颜色 token</th><th>明</th><th>暗</th><th>角色</th></tr></thead>
<tbody>
${token_rows}
</tbody></table>
<table>
<thead><tr><th>字号 / 行高 / 字重</th><th>间距</th><th>尺寸与描边</th><th>动效</th></tr></thead>
<tbody>
<tr>
<td><code>body</code> 14 / 22.75 / 400<br><code>secondary</code> 12.25 / 18 / 400<br><code>micro</code> 10.5 / 14 / 500<br><code>code</code> 13 / 20 / 400<br><code>title-row</code> 14 / 20 / 600<br><code>h1</code> 20 / 26 / 600<br><code>h2</code> 17.5 / 24 / 600<br><code>h3</code> 15.75 / 22 / 600<br><code>h4</code> 14 / 20 / 600</td>
<td>3.5 / 7 / 10.5 / 14 / 21 / 28 / 42<br>基础单位 3.5</td>
<td><code>row</code> 28 / 24.5 / 21<br><code>icon</code> 16 / 20<br><code>dot</code> 8<br><code>stroke</code> 1 · 焦点 2+1 · 标记 2<br><code>radius</code> 4 / 6 / 8 / 12<br><code>hit</code> ≥ 24</td>
<td><code>fast</code> 120ms<br><code>base</code> 200ms<br><code>slow</code> 320ms<br><code>reveal</code> 400 / 120ms<br><code>caret</code> 1060ms<br><code>tooltip</code> 600ms</td>
</tr>
</tbody></table>
<h3>可复制：CSS 变量</h3>
<pre class="copy">:root{
${light_vars}
  --space-1:3.5px; --space-2:7px; --space-3:10.5px; --space-4:14px;
  --space-6:21px; --space-8:28px; --space-12:42px;
  --row-regular:28px; --row-compact:24.5px; --control-mini:21px;
  --icon-inline:16px; --icon-block:20px; --dot-status:8px;
  --stroke-hairline:1px; --stroke-focus:2px; --stroke-marker:2px; --hit-min:24px;
  --radius-sm:4px; --radius-control:6px; --radius-card:8px; --radius-panel:12px;
  --motion-fast:120ms; --motion-base:200ms; --motion-slow:320ms;
  --reveal-catchup:400ms; --reveal-fade:120ms; --caret-cycle:1060ms; --tooltip-delay:600ms;
}
:root[data-theme="dark"]{
${dark_vars}
}</pre>

<p class="foot">U2 界面视觉 · C2 首片 · 数值来源：docs/domains/U2-visual.md §8。对比度按 WCAG 2.1 公式计算，可对表内值重算复核。</p>
</div>

<script>
(function(){
  var root=document.documentElement;
  function setTheme(mode){
    if(mode==='system'){
      root.removeAttribute('data-theme');
      document.body.setAttribute('data-theme-auto','');
    }else{
      root.setAttribute('data-theme',mode);
    }
    document.querySelectorAll('[data-theme-btn]').forEach(function(b){
      b.classList.toggle('on', b.getAttribute('data-theme-btn')===mode);
    });
  }
  document.querySelectorAll('[data-theme-btn]').forEach(function(b){
    b.addEventListener('click',function(){setTheme(b.getAttribute('data-theme-btn'));});
  });
  if(window.matchMedia && matchMedia('(prefers-color-scheme: dark)').matches){
    setTheme('system');
  }
  var motion=document.getElementById('motion');
  motion.addEventListener('change',function(){
    document.body.setAttribute('data-motion', motion.checked?'off':'on');
  });
  var card=document.getElementById('card');
  document.getElementById('toggle').addEventListener('click',function(){
    card.classList.toggle('open');
  });
  var t=document.getElementById('typing'), timer=null;
  var text='逐字出现：单位是词，中文按字，最多落后 400ms 追平，每个单位淡入 120ms；只改不透明度，不重排。';
  document.getElementById('type').addEventListener('click',function(){
    if(timer){clearInterval(timer);}
    t.innerHTML='';
    var i=0;
    timer=setInterval(function(){
      if(i>=text.length){clearInterval(timer);timer=null;return;}
      var s=document.createElement('span');
      s.textContent=text[i++];
      t.appendChild(s);
    },28);
  });
})();
</script>
</body>
</html>
"""

if __name__ == "__main__":
    build()
