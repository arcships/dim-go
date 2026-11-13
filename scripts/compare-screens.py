#!/usr/bin/env python3
"""三屏并排：左边是**官方示例的真实切片**，右边是我们同尺寸的切片（UI-24）。

    python3 scripts/compare-screens.py

参照物只从 docs/domains/calibration/examples/ 的官方示例渲染里裁，不再自造参照屏。
每张图左上角写明来源文件与裁剪坐标，图名与坐标同时写进 docs/domains/U0-control-baseline.md。
"""
import os

from PIL import Image, ImageDraw

BASE = "docs/domains/screens"
EX = "docs/domains/calibration/examples"
GAP = 20
HEAD = 34


def crop(path, box):
    img = Image.open(path).convert("RGB")
    x0, y0, x1, y1 = box
    return img.crop((x0, y0, min(x1, img.width), min(y1, img.height)))


def compose(name, title, left_path, left_box, right_path, right_box):
    left = crop(left_path, left_box)
    right = crop(right_path, right_box)
    h = max(left.height, right.height)
    w = left.width + GAP + right.width
    out = Image.new("RGB", (w, h + HEAD), (250, 250, 251))
    out.paste(left, (0, HEAD))
    out.paste(right, (left.width + GAP, HEAD))
    d = ImageDraw.Draw(out)
    d.text((8, 6), f"{title}", fill=(35, 35, 40))
    d.text((8, 19), f"左：{os.path.basename(left_path)} {left_box}   右：{os.path.basename(right_path)} {right_box}",
           fill=(110, 110, 116))
    path = os.path.join(BASE, "baseline", f"compare-{name}.png")
    out.save(path)
    print(f"{path}  {out.width}×{out.height}  左 {os.path.basename(left_path)}{left_box} / 右 {os.path.basename(right_path)}{right_box}")


def main():
    os.makedirs(os.path.join(BASE, "baseline"), exist_ok=True)
    # 主界面·默认态：两边尽量同构——标题、列表、行内代码、代码块、引用、表格、
    # 链接都出现，差异只剩"我们的块样式对不对"。
    compose("default", "主界面 · 默认态：Markdown 块级排版（标题 / 列表 / 代码 / 引用 / 表格 / 链接）",
            os.path.join(EX, "markdown-chat.png"), (14, 236, 800, 700),
            os.path.join(BASE, "variant-rich.png"), (272, 120, 1058, 584))
    # 等待审批：官方 permissionflow 的卡片行 + 右侧按钮组，对我们的审批卡。
    compose("waiting", "等待审批：卡片行 + 右侧按钮组",
            os.path.join(EX, "permissionflow.png"), (8, 80, 512, 340),
            os.path.join(BASE, "state-waiting.png"), (272, 700, 776, 900))
    # 设置·外观：官方 gallery-overview 的页面标题 + 卡片，对我们的设置内容区。
    compose("settings", "设置 · 外观：页面标题 + 分组",
            os.path.join(EX, "gallery-overview.png"), (220, 40, 720, 340),
            os.path.join(BASE, "variant-settings.png"), (256, 20, 756, 320))


if __name__ == "__main__":
    main()
