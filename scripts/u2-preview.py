#!/usr/bin/env python3
"""Generate docs/domains/U2-visual-preview.html from the v2 token set.

v2 规则：全部沿用 mygo 默认值，只有三类依据能改（门槛 / 证据 / 需求）。
本脚本里的字号、行高、间距、圆角、动效由下方常量推出，不在 HTML 里硬写。
"""
from string import Template

FONT = 14                    # Theme.FontSize（唯一字阶偏离，§1.2）
R_SECONDARY = 0.875          # 框架 CodeSize 比例
R_MICRO = 0.75               # 框架 table 页码标签比例
R_CODE = 0.875               # 框架 CodeSize
LH_BODY = 1.5                # 框架 markdown LineHeight
LH_CODE = 1.4                # 框架 CodeBlockLineHeight
HEADS = [(1.6, 700), (1.35, 650), (1.15, 600), (1.0, 600), (0.95, 600), (0.9, 600)]
SPACING = 4                  # 框架 Theme.Spacing
RADIUS = 6                   # 框架 Theme.Radius
RADIUS_CODE = RADIUS + 2     # 框架 CodeBlockRadius
MOTION_TRANSITION = 200      # 框架 ui/transition.go defaultDuration
MOTION_SPRING = 500          # 框架 Smooth
MOTION_CONTENT = 350         # 框架内容动画
CARET = 1060                 # 框架光标周期（530 半周期）
TOOLTIP, TOOLTIP_WARM = 600, 400

LIGHT = dict(
    bg="#ffffff", surface="#f4f4f5", surface_hover="#e9e9ec", surface_pressed="#dddde1",
    border="#d9d9de", border_strong="#85858b",
    text="#18181b", text_secondary="#5f5f68", accent="#2563eb", accent_hover="#1d4ed8",
    accent_pressed="#1e40af", accent_on="#ffffff", accent_ink="#1d4ed8",
    success="#15803d", warning="#b45309", danger="#b91c1c", focus="#2563eb",
    selection="rgba(37,99,235,0.20)",
)
DARK = dict(
    bg="#18181b", surface="#27272a", surface_hover="#323236", surface_pressed="#3c3c41",
    border="#3f3f46", border_strong="#8b8b94",
    text="#f4f4f5", text_secondary="#a1a1aa", accent="#2563eb", accent_hover="#1d4ed8",
    accent_pressed="#1e40af", accent_on="#ffffff", accent_ink="#60a5fa",
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
    "bg": "窗口底（沿用框架）",
    "surface": "卡片 / 气泡 / 输入卡 / 行底（沿用框架）",
    "surface-hover": "悬停面；选中行底（沿用框架）",
    "surface-pressed": "按下态（沿用框架）",
    "border": "分隔线与普通轮廓（沿用框架）",
    "border-strong": "边界单独承担识别（补：≥3:1）",
    "accent": "强调填充（沿用框架）",
    "accent-hover": "强调填充悬停（沿用框架）",
    "accent-pressed": "强调填充按下（沿用框架）",
}
INK = {
    "text": "正文（沿用框架）",
    "text-secondary": "次要文本（改：AA 4.5）",
    "accent-ink": "强调色作文字 / 图标 / 标记（补）",
    "accent-on": "落在强调填充上的文字（沿用框架）",
    "success": "成功状态（明改 / 暗沿用）",
    "warning": "警告状态（明改 / 暗沿用）",
    "danger": "失败状态（明暗均改）",
    "focus": "焦点环（改：≥3:1）",
    "selection": "文本选区（改：透明度下调）",
}

ORDER = ["bg", "surface", "surface-hover", "surface-pressed", "border", "border-strong",
         "text", "text-secondary", "accent", "accent-hover", "accent-pressed", "accent-on",
         "accent-ink", "success", "warning", "danger", "selection", "focus"]

VARNAME = {k: "--" + k for k in ORDER}


def type_rows_data():
    rows = [
        ("type-body", "正文", FONT, "×1.5", 400, "助手回复、用户消息、Markdown、输入、说明行",
         "我把重试逻辑抽到了 retry.ts，并补了一个测试。"),
        ("type-body-strong", "正文强调", FONT, "×1.5", 600, "Markdown 粗体、对象名", "重试逻辑"),
        ("type-secondary", "次要", FONT * R_SECONDARY, "不设", 400, "摘要行、工具行、时间、用量、按钮",
         "运行了 4 个操作 · 改了 2 个文件 · 12s"),
        ("type-secondary-strong", "次要强调", FONT * R_SECONDARY, "不设", 600, "状态词、键位提示", "等待你同意"),
        ("type-micro", "极小", FONT * R_MICRO, "不设", 400, "计数、徽标、编号", "+12 −3 · 4.2 KB"),
        ("type-code", "代码", FONT * R_CODE, "×1.4", 400, "代码块、行内代码、命令", "go test ./..."),
        ("type-title-row", "标题行会话名", FONT, "不设", 600, "标题行", "payments 重试逻辑"),
    ]
    for i, (size, weight) in enumerate(HEADS):
        rows.append(("type-h%d" % (i + 1), "h%d" % (i + 1), FONT * size, "×1.5", weight,
                     "Markdown %d 级标题" % (i + 1), "重试与超时"))
    return rows


TYPE = type_rows_data()


def lh_of(mode, size):
    if mode == "×1.5":
        return fmt(size * LH_BODY) + "px"
    if mode == "×1.4":
        return fmt(size * LH_CODE) + "px"
    return "normal"


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
                note = "对 accent %.2f / 对 hover %.2f" % (cr(t["accent_on"], t["accent"]), cr(t["accent_on"], t["accent_hover"]))
            elif k == "selection":
                sbg = over("#2563eb" if theme_name == "明" else "#3b82f6", 0.20 if theme_name == "明" else 0.30, t["bg"])
                chip = '<span class="ink" style="background:%s;color:%s">Aa 选区</span>' % (sbg, t["text"])
                note = "选区合成 %s · 正文 %.2f · 次要 %.2f" % (sbg, cr(t["text"], sbg), cr(t["text_secondary"], sbg))
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
        ("次要 / surface", "text_secondary", "surface", 4.5),
        ("次要 / surface-hover", "text_secondary", "surface_hover", 4.5),
        ("次要 / surface-pressed（例外：框架配色，瞬时态）", "text_secondary", "surface_pressed", 4.5),
        ("强调作文字 / surface", "accent_ink", "surface", 4.5),
        ("白字 / accent", "accent_on", "accent", 4.5),
        ("success 作文字 / surface", "success", "surface", 4.5),
        ("warning 作文字 / surface", "warning", "surface", 4.5),
        ("danger 作文字 / surface", "danger", "surface", 4.5),
        ("焦点环 / surface", "focus", "surface", 3.0),
        ("焦点环 / surface-hover", "focus", "surface_hover", 3.0),
        ("焦点环 / surface-pressed", "focus", "surface_pressed", 3.0),
        ("border-strong / surface", "border_strong", "surface", 3.0),
        ("强调标记 accent-ink / surface-hover", "accent_ink", "surface_hover", 3.0),
        ("强调标记 accent-ink / surface-pressed（最差面）", "accent_ink", "surface_pressed", 3.0),
    ]
    rows = []
    for label, a, b, target in checks:
        lv, dv = cr(LIGHT[a], LIGHT[b]), cr(DARK[a], DARK[b])
        ok = lv >= target - 1e-9 and dv >= target - 1e-9
        cell, txt = ("ok", "通过") if ok else ("bad", "不达")
        if not ok and "例外" in label:
            cell, txt = "warn", "例外"
        rows.append('<tr><td>%s</td><td>%.2f</td><td>%.2f</td><td>≥%.1f</td><td class="%s">%s</td></tr>'
                    % (label, lv, dv, target, cell, txt))
    for label, a, b in [("accent 填充 / 暗 surface（例外：有文字标签）", "accent", "surface")]:
        rows.append('<tr><td>%s</td><td>%.2f</td><td>%.2f</td><td>≥3.0</td><td class="warn">例外</td></tr>'
                    % (label, cr(LIGHT[a], LIGHT[b]), cr(DARK[a], DARK[b])))
    return "\n".join(rows)


CSS_VARS = """:root{
%s
  --space-1:%dpx; --space-2:%dpx; --space-3:%dpx; --space-4:%dpx;
  --space-6:%dpx; --space-8:%dpx; --space-12:%dpx;
  --row-regular:28px; --row-compact:24px; --control-mini:20px;
  --icon-inline:16px; --icon-block:20px; --dot-status:8px;
  --stroke-hairline:1px; --stroke-focus:2px; --stroke-marker:2px; --hit-min:24px;
  --radius:%dpx; --radius-code:%dpx;
  --motion-transition:%dms; --motion-spring:%dms; --motion-content:%dms;
  --caret-cycle:%dms; --tooltip-delay:%dms;
}
:root[data-theme="dark"]{
%s
}
@media (prefers-color-scheme: dark){
  :root:not([data-theme]){%s}
}""" % (
    theme_vars(LIGHT),
    SPACING * 1, SPACING * 2, SPACING * 3, SPACING * 4, SPACING * 6, SPACING * 8, SPACING * 12,
    RADIUS, RADIUS_CODE,
    MOTION_TRANSITION, MOTION_SPRING, MOTION_CONTENT, CARET, TOOLTIP,
    theme_vars(DARK), theme_vars(DARK),
)

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
        lhv = lh_of(lh, size)
        type_rows.append(
            '<tr><td><code>%s</code></td><td>%s</td><td>%s</td><td>%d</td><td>%s</td>'
            '<td class="sample" style="font-size:%spx;line-height:%s;font-weight:%d">%s</td></tr>'
            % (name, fmt(size), lhv.replace("normal", "框架度量"), w, use, fmt(size), lhv, w, sample)
        )
    type_rows = "\n".join(type_rows)

    type_demo = "\n".join(
        '<div class="type-demo"><span class="spec">%s · %spx / %s / %d · %s</span>'
        '<p style="font-size:%spx;line-height:%s;font-weight:%d">%s</p></div>'
        % (name, fmt(size), lh_of(lh, size).replace("normal", "框架度量"), w, use, fmt(size), lh_of(lh, size), w, sample)
        for name, label, size, lh, w, use, sample in TYPE
    )

    spaces = [(n, SPACING * n) for n in (1, 2, 3, 4, 6, 8, 12)]
    space_rows = "\n".join(
        '<div class="spacerow"><code>space-%d</code><span class="bar" style="width:%dpx"></span><span class="val">%dpx</span></div>'
        % (n, v, v) for n, v in spaces)

    icon_rows = "\n".join(
        '<div class="iconrow"><span class="ibox" style="width:%dpx;height:%dpx">%s</span>'
        '<code>%s · %dpx</code></div>' % (s, s, icon(n, s), n, s)
        for n, s in [("arrow-up", 16), ("stop", 16), ("paperclip", 16), ("chevron-down", 16),
                     ("x", 16), ("exclamation", 16), ("clock", 16), ("check", 16),
                     ("chat", 20)])

    token_rows = []
    for k in ORDER:
        token_rows.append("<tr><td><code>%s</code></td><td><code>%s</code></td><td><code>%s</code></td><td>%s</td></tr>"
                          % (k.replace("_", "-"), LIGHT[k.replace("-", "_")], DARK[k.replace("-", "_")], FILL.get(k) or INK.get(k, "")))
    token_rows.append("<tr><td><code>inverse</code></td><td><code>#18181b</code></td><td><code>#f4f4f5</code></td><td>沿用框架：提示与 toast</td></tr>")
    token_rows.append("<tr><td><code>inverse-text</code></td><td><code>#ffffff</code></td><td><code>#18181b</code></td><td>同上</td></tr>")
    token_rows.append("<tr><td><code>scrollbar</code></td><td><code>rgba(0,0,0,0.32)</code></td><td><code>rgba(255,255,255,0.35)</code></td><td>沿用框架</td></tr>")
    token_rows = "\n".join(token_rows)

    body_lh = "%.2fpx" % (FONT * LH_BODY)
    sec = fmt(FONT * R_SECONDARY)
    micro = fmt(FONT * R_MICRO)
    code_lh = "%.2fpx" % (FONT * R_CODE * LH_CODE)

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
        font=FONT, sec=sec, micro=micro, code=fmt(FONT * R_CODE), body_lh=body_lh, code_lh=code_lh,
        head1=fmt(FONT * HEADS[0][0]), head2=fmt(FONT * HEADS[1][0]), head3=fmt(FONT * HEADS[2][0]),
        head1_lh=fmt(FONT * HEADS[0][0] * LH_BODY), head2_lh=fmt(FONT * HEADS[1][0] * LH_BODY),
        head3_lh=fmt(FONT * HEADS[2][0] * LH_BODY),
        radius=RADIUS, radius_code=RADIUS_CODE,
        s1=SPACING * 1, s2=SPACING * 2, s3=SPACING * 3, s4=SPACING * 4, s6=SPACING * 6,
        s8=SPACING * 8, s12=SPACING * 12,
        motion_transition=MOTION_TRANSITION, motion_spring=MOTION_SPRING,
        motion_content=MOTION_CONTENT, caret=CARET, tooltip=TOOLTIP, tooltip_warm=TOOLTIP_WARM,
        block_spacing=fmt(FONT * 0.75), h_top=fmt(FONT * 1.4), h_bottom=fmt(FONT * 0.55),
        code_pad=fmt(FONT * 0.85),
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
<title>U2 C2 视觉 token 预览（v2 默认优先）</title>
<style>
${css_vars}
*{box-sizing:border-box}
html,body{margin:0}
body{
  background:var(--bg);color:var(--text);
  font-family:-apple-system,BlinkMacSystemFont,"PingFang SC","Hiragino Sans GB","Segoe UI",sans-serif;
  font-size:${font}px;line-height:${body_lh};
  padding:0 0 96px;
}
body[data-motion="off"]{--motion-transition:0ms;--motion-spring:0ms;--motion-content:0ms}
@media (prefers-reduced-motion: reduce){
  body{--motion-transition:0ms;--motion-spring:0ms;--motion-content:0ms}
  .demo-card{transition:none !important}
}
code{font-family:"SF Mono",Menlo,monospace;font-size:${code}px;line-height:${code_lh}}
.wrap{max-width:1180px;margin:0 auto;padding:0 ${s4}px}
header.topbar{
  background:var(--surface);border-bottom:1px solid var(--border);
  padding:${s2}px 0;margin-bottom:${s6}px;
}
header.topbar .wrap{display:flex;align-items:center;gap:${s4}px;flex-wrap:wrap}
h1{font-size:${head1}px;line-height:${head1_lh}px;font-weight:700;margin:0}
.sub{font-size:${sec}px;line-height:${body_lh};color:var(--text-secondary)}
.seg{display:flex;gap:${s1}px;margin-left:auto}
.seg button{
  font:inherit;font-size:${sec}px;height:28px;padding:0 ${s3}px;
  border-radius:${radius}px;border:1px solid var(--border);background:var(--surface);color:var(--text);cursor:pointer;
  transition:background var(--motion-transition) ease-out;
}
.seg button.on{background:var(--accent);border-color:var(--accent);color:var(--accent-on)}
.seg button:focus-visible{outline:2px solid var(--focus);outline-offset:1px}
label.chk{display:flex;align-items:center;gap:${s2}px;font-size:${sec}px;color:var(--text-secondary);cursor:pointer}
h2{font-size:${head2}px;line-height:${head2_lh}px;font-weight:650;margin:${s8}px 0 ${s2}px;padding-top:${s2}px;border-top:1px solid var(--border)}
h3{font-size:${head3}px;line-height:${head3_lh}px;font-weight:600;margin:${s6}px 0 ${s2}px}
p.note{font-size:${sec}px;color:var(--text-secondary);margin:0 0 ${s2}px;max-width:900px}
table{border-collapse:collapse;width:100%;margin:${s2}px 0 ${s4}px;font-size:${sec}px}
th,td{text-align:left;padding:${s1}px ${s3}px;border-bottom:1px solid var(--border);vertical-align:top}
th{font-weight:600;color:var(--text-secondary);white-space:nowrap}
td.sample{white-space:nowrap;overflow:hidden;text-overflow:ellipsis;max-width:340px}
.ok{color:var(--success);font-weight:600}
.bad{color:var(--danger);font-weight:600}
.warn{color:var(--warning);font-weight:600}
/* tokens */
.palettes{display:grid;grid-template-columns:1fr 1fr;gap:${s6}px}
.panel{background:var(--bg);color:var(--text);border:1px solid var(--border);border-radius:${radius}px;padding:${s4}px}
.t-light,.t-dark{color:var(--text)}
.t-light{${light_vars}}
.t-dark{${dark_vars}}
.panel h3{margin-top:0}
.sw{display:flex;gap:${s3}px;align-items:center;padding:${s1}px 0;border-bottom:1px solid var(--border)}
.chipbox{width:56px;flex:0 0 auto}
.fill,.ink{display:block;width:56px;height:28px;border-radius:${radius}px;border:1px solid var(--border);font-size:${sec}px;line-height:26px;text-align:center}
.ink{font-family:-apple-system,sans-serif}
.ringwrap{display:flex;width:56px;height:28px;align-items:center;justify-content:center}
.ring{display:block;width:34px;height:18px;border-radius:${radius}px;background:var(--surface)}
.meta{display:flex;flex-direction:column;font-size:${sec}px;line-height:16px}
.meta b{font-weight:500}
.meta code{font-size:11px}
.meta span{color:var(--text-secondary)}
.meta .ratio{font-size:11px}
/* spacing */
.spacerow{display:flex;align-items:center;gap:${s3}px;margin:${s1}px 0}
.spacerow code{width:76px;color:var(--text-secondary);font-size:${sec}px}
.bar{display:block;height:${s2}px;background:var(--accent-ink);border-radius:2px}
.spacerow .val{font-size:${sec}px;color:var(--text-secondary)}
/* type demo */
.type-demo{padding:${s2}px 0;border-bottom:1px solid var(--border)}
.type-demo p{margin:0;max-width:820px}
.spec{display:block;font-size:${micro}px;color:var(--text-secondary);font-family:"SF Mono",Menlo,monospace}
/* demo frame */
.frame{border:1px solid var(--border);border-radius:${radius}px;padding:${s4}px;background:var(--bg);max-width:820px}
.thread{display:flex;flex-direction:column;gap:${s3}px}
.u{display:flex;justify-content:flex-end}
.bubble{max-width:85%;background:var(--surface);border:1px solid var(--border);border-radius:${radius}px;padding:${s2}px ${s4}px;font-size:${font}px;line-height:${body_lh}}
.md{font-size:${font}px;line-height:${body_lh}}
.md p{margin:0 0 ${block_spacing}px}
.md h4{font-size:${font}px;line-height:${body_lh};font-weight:600;margin:${h_top}px 0 ${h_bottom}px}
.md code.inline{background:var(--surface);border:1px solid var(--border);border-radius:${radius}px;padding:1px ${s1}px;font-size:${code}px}
.codeblock{background:var(--surface);border:1px solid var(--border);border-radius:${radius_code}px;padding:${code_pad}px;margin:${block_spacing}px 0;font-family:"SF Mono",Menlo,monospace;font-size:${code}px;line-height:${code_lh};color:var(--text);white-space:pre;overflow:auto}
.summary{display:flex;align-items:center;gap:${s2}px;font-size:${sec}px;color:var(--text-secondary);padding:${s1}px 0}
.summary .state{font-weight:600}
.tool{display:flex;align-items:center;gap:${s2}px;min-height:24px;font-size:${sec}px;padding-left:${s3}px;border-left:1px solid var(--border);color:var(--text-secondary)}
.notice{display:flex;align-items:flex-start;gap:${s2}px;font-size:${font}px;line-height:${body_lh};padding:${s1}px 0}
.dot{width:8px;height:8px;border-radius:4px;background:var(--accent-ink);flex:0 0 auto;margin-top:${s2}px}
.caret{display:inline-block;width:2px;height:${font}px;background:var(--accent-ink);vertical-align:-2px;margin-left:2px;animation:blink var(--caret-cycle) steps(1,end) infinite}
@keyframes blink{0%,49%{opacity:1}50%,100%{opacity:0}}
body[data-motion="off"] .caret{animation:none}
.approval{background:var(--surface);border:1px solid var(--accent-ink);border-radius:${radius}px;padding:${s4}px;display:flex;flex-direction:column;gap:${s3}px}
.approval .k{font-size:${sec}px;font-weight:600;color:var(--text-secondary)}
.approval .what{font-size:${font}px;line-height:${body_lh}}
.approval .why{font-size:${sec}px;color:var(--text-secondary)}
.actions{display:flex;gap:${s2}px;align-items:center}
button.primary{
  display:inline-flex;align-items:center;gap:${s2}px;
  font:inherit;font-size:${sec}px;height:28px;padding:0 ${s3}px;border-radius:${radius}px;
  border:1px solid var(--accent);background:var(--accent);color:var(--accent-on);cursor:pointer;
  transition:background var(--motion-transition) ease-out;
}
button.primary:hover{background:var(--accent-hover);border-color:var(--accent-hover)}
button.primary:active{background:var(--accent-pressed);border-color:var(--accent-pressed)}
button.primary:focus-visible{outline:2px solid var(--focus);outline-offset:1px}
button.ghost{
  display:inline-flex;align-items:center;gap:${s2}px;
  font:inherit;font-size:${sec}px;height:28px;padding:0 ${s3}px;border-radius:${radius}px;
  border:1px solid var(--border);background:var(--surface);color:var(--text);cursor:pointer;
  transition:background var(--motion-transition) ease-out;
}
button.ghost:hover{background:var(--surface-hover)}
button.ghost:focus-visible{outline:2px solid var(--focus);outline-offset:1px}
button.ghost[disabled]{opacity:.5;cursor:default}
.iconbtn{display:inline-flex;align-items:center;justify-content:center;width:28px;height:28px;border-radius:${radius}px;border:1px solid transparent;background:transparent;color:var(--text);cursor:pointer}
.iconbtn:hover{background:var(--surface-hover)}
.iconbtn:focus-visible{outline:2px solid var(--focus);outline-offset:1px}
.queue{display:flex;align-items:center;gap:${s2}px;height:33px;padding:0 ${s4}px;border:1px solid var(--border);border-radius:${radius}px;background:var(--surface);font-size:${sec}px;color:var(--text-secondary)}
.composer{border:1px solid var(--border);border-radius:${radius}px;background:var(--surface);padding:${s3}px}
.composer .editor{font-size:${font}px;line-height:${body_lh};min-height:44px;color:var(--text)}
.composer.focus{border-color:var(--border-strong)}
.composer .bar2{display:flex;align-items:center;gap:${s2}px;padding-top:${s2}px}
.chip{display:inline-flex;align-items:center;gap:${s1}px;height:20px;padding:0 ${s2}px;border-radius:${radius}px;border:1px solid var(--border);background:var(--surface);font-size:${micro}px;color:var(--text-secondary)}
.spacer{flex:1}
.rowdemo{display:flex;flex-direction:column;max-width:520px;border:1px solid var(--border);border-radius:${radius}px;overflow:hidden}
.rowdemo .row{display:flex;align-items:center;gap:${s2}px;height:28px;padding:0 ${s3}px;font-size:${sec}px;border-bottom:1px solid var(--border)}
.rowdemo .row:last-child{border-bottom:0}
.rowdemo .row.hover{background:var(--surface-hover)}
.rowdemo .row.selected{background:var(--surface-hover);box-shadow:inset 2px 0 0 var(--accent-ink)}
.rowdemo .row.focused{outline:2px solid var(--focus);outline-offset:-2px}
.rowdemo .row.disabled{opacity:.5}
.rowdemo .row .muted{color:var(--text-secondary)}
.empty{display:flex;align-items:center;gap:${s2}px;color:var(--text-secondary);font-size:${sec}px}
.ico{display:block;flex:0 0 auto}
.ibox{display:flex;align-items:center;justify-content:center;border:1px dashed var(--border);border-radius:${radius}px;color:var(--text)}
.iconrow{display:flex;align-items:center;gap:${s3}px;margin:${s1}px 0}
.iconrow code{color:var(--text-secondary);font-size:${sec}px}
/* motion */
.demo-card{border:1px solid var(--border);border-radius:${radius}px;background:var(--surface);padding:0 ${s4}px;overflow:hidden;transition:height var(--motion-transition) ease-out;height:33px}
.demo-card.open{height:112px}
.demo-card p{margin:${s2}px 0;font-size:${sec}px;color:var(--text-secondary)}
.demo-card .body{opacity:0;transition:opacity var(--motion-transition) ease-out}
.demo-card.open .body{opacity:1}
.typing{font-size:${font}px;line-height:${body_lh};min-height:${body_lh};margin:${block_spacing}px 0 0;max-width:640px}
.foot{margin-top:${s8}px;font-size:${micro}px;color:var(--text-secondary)}
@media (max-width:900px){.palettes{grid-template-columns:1fr}}
</style>
</head>
<body>
<header class="topbar"><div class="wrap">
  <div>
    <h1>U2 C2 · 视觉 token 预览</h1>
    <div class="sub">v2 默认优先：字号比例、行高、间距基准、圆角、动效一律取 mygo 默认；只有 AA 门槛、实测证据与 UI 需求允许改。预览只示样式，不代表布局与控件位置（布局归 U1/U3）。</div>
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

<h2 id="s1">1 字阶 · 框架 Markdown 标尺，锚点 ${font}</h2>
<p class="note">级差、字重、标题上下间距全部来自框架（1.6 / 1.35 / 1.15 / 1 / 0.95 / 0.9em，700 / 650 / 600，上 1.4em 下 0.55em，行高 1.5、代码行高 1.4、代码字号 0.875em）。本文只把 <code>Theme.FontSize</code> 定为 ${font}，次要 ${sec} 与极小 ${micro} 用框架自己的 0.875 与 0.75 缩比。"不设"表示交给框架按字体度量排版。</p>
<table>
<thead><tr><th>token</th><th>字号</th><th>行高</th><th>字重</th><th>用在哪</th><th>样张</th></tr></thead>
<tbody>
${type_rows}
</tbody></table>
<h3>逐级样张（同一句话，按级渲染）</h3>
${type_demo}

<h2 id="s2">2 间距 · 基础单位 ${s1}（框架 Theme.Spacing）</h2>
<p class="note">只用 <code>t.Space(n)</code> 的七档，代码里写 <code>t.Space(n)</code> 而不是字面量。Markdown 内部间距（段间 ${block_spacing}、标题上 ${h_top} 下 ${h_bottom}、列表缩进 22.4、代码块内边距 ${code_pad}）是框架的 em 值，不进这套标尺。</p>
${space_rows}
<p class="note">用法：图标与文字 ${s2}；标记与文字 ${s1}；行内元素 ${s1}–${s2}；同一轮内块间 ${s3}；轮次间 ${s6}；卡片内边距 ${s4}；输入卡 ${s3}；队列条与输入卡 ${s2}；内容左右 ${s4}。</p>

<h2 id="s3">3 颜色语义 · 每主题 18 个 token</h2>
<p class="note">两套同名同角色。强调色只用于可操作 / 当前 / 进行中；成功 / 警告 / 危险只用于状态，只作文字、图标与 1px 描边。18 个里 16 个是框架字段（8 个按 AA 改值），2 个是框架没有字段的补充（border-strong、accent-ink）；inverse / inverse-text / scrollbar 沿用框架。</p>
<div class="palettes">
  <div class="panel t-light"><h3>明色</h3>
${light_swatches}
  </div>
  <div class="panel t-dark"><h3>暗色</h3>
${dark_swatches}
  </div>
</div>

<h2 id="s4">4 状态样式（只改已有元素的属性）</h2>
<p class="note">每条状态落在 U1 §5 指定的既有元素上；不新增容器。语义态至少两项差异（颜色 + 形态 / 文字 / 位置）；悬停与按下是指针反馈，可以只有颜色。优先级：待你决定 &gt; 结果不确定 &gt; 失败 &gt; 运行中 &gt; 可用。</p>

<h3>4.1 运行中</h3>
<div class="frame"><div class="thread">
  <div class="md"><p>我先看一下现在的重试实现，再把公共部分抽出来。<span class="caret"></span></p></div>
  <div class="summary"><span class="dot"></span><span class="state" style="color:var(--text)">正在运行</span><span>· 12s</span></div>
  <div class="tool">${icon_chev}运行测试 · payments/retry</div>
</div></div>
<p class="spec">8px 实心圆 accent-ink · 摘要行 ${sec}/${body_lh} text-secondary · 状态词 600 · 行尾光标 2×${font} accent-ink，周期 ${caret}ms（框架光标同周期）· 发送键文字变「停止」，填充不变</p>

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
<p class="spec">卡面 surface · 1px accent-ink 描边（明 6.10 / 暗 5.86，≥3:1）· 圆角 ${radius}（Theme.Radius）· 内边距 ${s4} · 命令 ${code}/${code_lh} · 主按钮 accent/accent-on 28 高 · 摘要行时钟图标 accent-ink</p>

<h3>4.4 结果不确定</h3>
<div class="frame"><div class="thread">
  <div class="md"><p>已把重试逻辑抽到 <code class="inline">retry.ts</code>。</p></div>
  <div class="notice">${icon_warn}<span>这条消息是否已经发出还不确定，正在核对。</span></div>
</div></div>
<p class="spec">叹号图标 warning 16px · 说明行 ${font}/${body_lh} text · 属于本轮，不是容器 · 不给重试、不给核对按钮</p>

<h3>4.5 不可用</h3>
<div class="frame"><div class="thread">
  <div class="composer" style="opacity:.5">
    <div class="editor">写点什么…</div>
    <div class="bar2"><span class="chip">技能</span><span class="spacer"></span><button class="ghost" disabled>${icon_upload}发送</button></div>
  </div>
  <div class="notice">${icon_warn}<span>连接已断开，输入暂不可用；恢复后内容还在。</span></div>
</div></div>
<p class="spec">整块不透明度 .5 · 不响应悬停与聚焦 · 禁用按钮 surface + 1px border + text-secondary · 原因一行 ${sec}</p>

<h3>4.6 选中 / 悬停 / 聚焦（同一行元素的三种属性变化）</h3>
<div class="rowdemo">
  <div class="row"><span>payments 重试逻辑</span><span class="spacer"></span><span class="muted">12:04</span></div>
  <div class="row hover"><span>payments 重试逻辑（悬停）</span><span class="spacer"></span><span class="muted">12:04</span></div>
  <div class="row selected"><span>payments 重试逻辑（选中）</span><span class="spacer"></span><span class="muted">12:04</span></div>
  <div class="row focused"><span>payments 重试逻辑（键盘聚焦）</span><span class="spacer"></span><span class="muted">12:04</span></div>
  <div class="row disabled"><span>payments 重试逻辑（不可用）</span><span class="spacer"></span><span class="muted">12:04</span></div>
</div>
<p class="spec">悬停：底 surface → surface-hover，${motion_transition}ms，文字不变、不位移 · 选中：底 surface-hover + 2px accent-ink 竖条（不新增 color token：框架唯一选中色是文本选区，叠面次要文本 4.41 / 3.88 不达 4.5）· 聚焦：2px 焦点环外偏 1px · 不可用：.5，悬停不点亮</p>

<h3>4.7 输入区、队列条、发送 / 停止</h3>
<div class="frame"><div class="thread">
  <div class="queue">${icon_clock}排队中 · 第 1 条：再看一下 changes 面板</div>
  <div class="composer focus">
    <div class="editor">把重试逻辑抽出来，并补一个测试<span class="caret"></span></div>
    <div class="bar2">
      <button class="iconbtn">${icon_clip}</button>
      <span class="chip">技能 · review</span>
      <span class="chip">retry.ts · 4.2 KB</span>
      <span class="spacer"></span>
      <button class="primary">${icon_stop}停止</button>
    </div>
  </div>
</div></div>
<p class="spec">输入卡 surface + 1px border（边界单独承担识别，聚焦时转 border-strong），圆角 ${radius}，内边距 ${s3} · 编辑器 ${font}/${body_lh} · 队列条 ${sec}，贴输入卡上方 ${s2} · 按钮 28 高、${sec}、图标 16 · 微型块 ${micro}</p>

<h3>4.8 空态</h3>
<div class="frame"><div class="thread">
  <div class="empty">${icon_block}这个会话还没有消息。写第一条，或先从左边挑一个会话。</div>
</div></div>
<p class="spec">20px 图标 + ${sec} text-secondary 一行 · 不用大字号、不画假内容</p>

<h2 id="s5">5 动效 · 一律取框架默认</h2>
<table>
<thead><tr><th>场景</th><th>时长</th><th>来源</th></tr></thead>
<tbody>
<tr><td>状态 / 颜色 / 背景 / 描边 / 不透明度</td><td>${motion_transition}ms · EaseOut</td><td>框架转场默认</td></tr>
<tr><td>出现 / 消失、折展、队列条进出、附件块增删</td><td>${motion_transition}ms · EaseOut</td><td>同上</td></tr>
<tr><td>列表跟随滚动、行增删</td><td>${motion_spring}ms 弹簧</td><td>框架 Smooth</td></tr>
<tr><td>数值与图标的内容变化</td><td>${motion_content}ms 弹簧</td><td>框架内容动画</td></tr>
<tr><td>光标闪烁</td><td>${caret}ms（半周期 530ms）</td><td>框架光标</td></tr>
<tr><td>提示（tooltip）</td><td>${tooltip}ms / 热态 ${tooltip_warm}ms</td><td>框架提示</td></tr>
<tr><td>逐字出现</td><td>渲染库默认</td><td>接入 markstream-go 时登记实际值</td></tr>
</tbody></table>
<p class="note">不定义自有档位；现役的折叠 300ms 不跟随（框架默认 ${motion_transition}ms 已满足"动效明显、可感知"）。逐字：单位 = 词（中文按字素），到达即出、落后时加速追平，只改不透明度，严格保序（UI-4）。</p>
<div class="demo-card" id="card"><p>折展样张 · ${motion_transition}ms ease-out · 点按钮观察高度过渡</p><p class="body">展开后追加的内容：高度从 33 过渡到 112，只有高度与不透明度在动，没有位移、没有回弹。</p></div>
<div class="actions" style="margin-top:${s2}px">
  <button class="ghost" id="toggle">展开 / 收起</button>
  <button class="ghost" id="type">模拟逐字出现</button>
</div>
<p class="typing" id="typing"></p>
<p class="spec">减少动效开启后：所有转场、逐字出现、弹簧效果变 0（颜色与不透明度也要显式置 0，框架对 NonMotion 默认仍会动），光标常亮，布局与状态不变（UI-5）。</p>

<h2 id="s6">6 图标与密度</h2>
<p class="note">线性、单色、24 网格、描边 1.5；渲染 16 → 1.0px，20 → 1.25px。界面不用实心图标；8px 状态点由方形圆角绘制，不是图标。来源：Heroicons v2.2.0（MIT）；供应商标志用 ailogo，仅作识别。框架不自带图标集，选型是本文必须做的一条决策。</p>
${icon_rows}
<p class="note">行高与图标：图标盒 ≤ 行高 − ${s2}（2×space-1）。28 行 → 16（余 6 / 6）；24 行 → 16（余 4 / 4）；20 行 → 只用文字级标记（12）。图标不单独落行，行盒内光学居中；16px 图标按钮命中区 ≥ 24×24。行高档位 28 / 24 / 20 来自现役实测 28 / 24.5 / 21 按 4 网格取整。</p>

<h2 id="s7">7 可访问性 · 对比度矩阵</h2>
<table>
<thead><tr><th>组合</th><th>明</th><th>暗</th><th>目标</th><th>结论</th></tr></thead>
<tbody>
${contrast_rows}
</tbody></table>
<p class="note">例外三条：<code>accent</code> 填充在暗色面上的边界 2.88（按钮有文字标签，识别由标签承担 5.17）；禁用态 .5 由 WCAG 豁免；次要文本落暗色按下面上 4.28（框架自己的配色，按下是 ≤${motion_transition}ms 的指针瞬时态）。焦点环 2px + 外偏 1px、不参与布局，只在键盘导航出现；交互目标 ≥ 24×24；文本缩放 100–200% 由框架乘 FontSize；高对比度沿用框架 <code>Theme.follow</code>，本文不另写覆盖。</p>

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
<td><code>body</code> ${font} / ${body_lh} / 400<br><code>secondary</code> ${sec} / 框架度量 / 400<br><code>micro</code> ${micro} / 框架度量 / 400<br><code>code</code> ${code} / ${code_lh} / 400<br><code>title-row</code> ${font} / 框架度量 / 600<br><code>h1–h6</code> ${head1} / ${head2} / ${head3} / ${font} / 1.5 倍行高</td>
<td>${s1} / ${s2} / ${s3} / ${s4} / ${s6} / 32 / 48<br>基础单位 ${s1}（框架）</td>
<td><code>row</code> 28 / 24 / 20<br><code>icon</code> 16 / 20<br><code>dot</code> 8<br><code>stroke</code> 1 · 焦点 2+1 · 标记 2<br><code>radius</code> ${radius}（代码块 ${radius_code}）<br><code>hit</code> ≥ 24</td>
<td><code>transition</code> ${motion_transition}ms<br><code>spring</code> ${motion_spring}ms<br><code>content</code> ${motion_content}ms<br><code>caret</code> ${caret}ms<br><code>tooltip</code> ${tooltip}/${tooltip_warm}ms</td>
</tr>
</tbody></table>
<h3>可复制：CSS 变量</h3>
<pre class="copy">:root{
${light_vars}
  --space-1:${s1}px; --space-2:${s2}px; --space-3:${s3}px; --space-4:${s4}px;
  --space-6:${s6}px; --space-8:32px; --space-12:48px;
  --row-regular:28px; --row-compact:24px; --control-mini:20px;
  --icon-inline:16px; --icon-block:20px; --dot-status:8px;
  --stroke-hairline:1px; --stroke-focus:2px; --stroke-marker:2px; --hit-min:24px;
  --radius:${radius}px; --radius-code:${radius_code}px;
  --motion-transition:${motion_transition}ms; --motion-spring:${motion_spring}ms; --motion-content:${motion_content}ms;
  --caret-cycle:${caret}ms; --tooltip-delay:${tooltip}ms;
}
:root[data-theme="dark"]{
${dark_vars}
}</pre>

<p class="foot">U2 界面视觉 · C2 v2（默认优先）· 数值来源：docs/domains/U2-visual.md §8。对比度按 WCAG 2.1 公式计算，可对表内值重算复核。</p>
</div>

<script>
(function(){
  var root=document.documentElement;
  function setTheme(mode){
    if(mode==='system'){
      root.removeAttribute('data-theme');
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
  var text='逐字出现：单位是词，中文按字，到达即出、落后时加速追平；只改不透明度，不重排。';
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
