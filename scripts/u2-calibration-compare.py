#!/usr/bin/env python3
"""Rebuild docs/domains/calibration/compare from the framework examples and
our current screenshots.

The comparison is only a visual acceptance aid: every panel is a crop of a
real render (example or docs/domains/screens/*.png), scaled by an integer
where noted. Run after `go test ./internal/ui -run TestScreens -update`.
"""
from pathlib import Path

from PIL import Image, ImageDraw, ImageFont

ROOT = Path(__file__).resolve().parent.parent
EXAMPLES = ROOT / "docs/domains/calibration/examples"
COMPARE = ROOT / "docs/domains/calibration/compare"
SCREENS = ROOT / "docs/domains/screens"
FONT_PATH = "/System/Library/Fonts/Hiragino Sans GB.ttc"
FONT_SIZE = 15
MARGIN = 12
GAP = 10
LABEL_H = 26
MAX_W = 1200
MIN_W = 640


def font():
    return ImageFont.truetype(FONT_PATH, FONT_SIZE)


def crop(path, box, scale=1):
    im = Image.open(path).convert("RGB").crop(box)
    if scale != 1:
        im = im.resize((im.width * scale, im.height * scale), Image.NEAREST)
    return im


def panel(title, im):
    """A labeled panel with a 1px outline, scaled down if wider than MAX_W."""
    if im.width > MAX_W:
        h = round(im.height * MAX_W / im.width)
        im = im.resize((MAX_W, h), Image.LANCZOS)
    w = max(im.width, MIN_W) + 2
    out = Image.new("RGB", (w, im.height + LABEL_H + 2), "#ffffff")
    d = ImageDraw.Draw(out)
    d.rectangle((0, 0, out.width - 1, out.height - 1), outline="#c9c9cf")
    d.rectangle((1, 1, out.width - 2, LABEL_H), fill="#f4f4f5")
    d.text((8, 5), title, fill="#18181b", font=font())
    out.paste(im, (1, LABEL_H + 1))
    return out


def stack(name, panels):
    width = max(p.width for p in panels) + 2 * MARGIN
    height = sum(p.height for p in panels) + GAP * (len(panels) - 1) + 2 * MARGIN
    out = Image.new("RGB", (width, height), "#e9e9ec")
    y = MARGIN
    for p in panels:
        out.paste(p, (MARGIN, y))
        y += p.height + GAP
    out.save(COMPARE / name)


def main():
    COMPARE.mkdir(parents=True, exist_ok=True)

    stack("buttons.png", [
        panel("example permissionflow · 禁用主按钮（Guide）保留填充、整体 0.5",
              crop(EXAMPLES / "permissionflow.png", (330, 85, 505, 132), 3)),
        panel("ours（修正后）· 禁用发送保留 accent 填充，附件为次按钮",
              crop(SCREENS / "state-default.png", (1240, 845, 1430, 890), 3)),
    ])

    stack("nesting.png", [
        panel("ours（修正后）· 输入卡只画一道 border-strong，TextAreaBase 无外观",
              crop(SCREENS / "state-default.png", (258, 735, 560, 900), 3)),
        panel("example notes · 单容器，编辑器不再描边",
              crop(EXAMPLES / "notes.png", (20, 20, 900, 160))),
        panel("example gallery · 卡为 bg、字段为 surface + border",
              crop(EXAMPLES / "gallery-controls.png", (230, 335, 940, 420))),
    ])

    stack("composer.png", [
        panel("example markdown-chat · 单行输入条 820×56",
              crop(EXAMPLES / "markdown-chat.png", (0, 680, 820, 720))),
        panel("ours（修正后）· 多行输入卡，按钮取框架默认 6/14",
              crop(SCREENS / "state-default.png", (256, 755, 1440, 900))),
    ])

    stack("statusmarks.png", [
        panel("ours（修正后）· 16×16 Heroicons outline 状态位",
              crop(SCREENS / "state-default.png", (460, 140, 700, 200), 3)),
        panel("example gallery · 侧栏 ui.Icon 16px SVG",
              crop(EXAMPLES / "gallery-overview.png", (20, 70, 300, 150), 2)),
    ])

    stack("sidebar.png", [
        panel("example notes · 侧栏 230，行内缩进与选中行由控件负责",
              crop(EXAMPLES / "notes.png", (0, 0, 230, 420))),
        panel("ours（修正后）· 选中样式取框架语义，状态标记占 16px icon 槽",
              crop(SCREENS / "state-running.png", (0, 0, 256, 420))),
    ])

    stack("stream.png", [
        panel("example markdown-chat · 消息流左对齐，正文上限 760",
              crop(EXAMPLES / "markdown-chat.png", (0, 60, 820, 700))),
        panel("ours · 内容列 784 居中（刻意差异 D14）",
              crop(SCREENS / "state-default.png", (256, 48, 1440, 620))),
    ])

    stack("fix-preview.png", [
        panel("before · TextArea 默认外观 + 4/12 内边距 + 禁用换面",
              crop(EXAMPLES / "probe-composer-before.png", (0, 0, 900, 220))),
        panel("after · TextAreaBase + 6/14 内边距 + 禁用不换面",
              crop(EXAMPLES / "probe-composer-v3.png", (0, 0, 900, 220))),
    ])


if __name__ == "__main__":
    main()
