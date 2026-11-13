#!/usr/bin/env python3
"""量测界面截图的排版基线（UI-22、§9.14 第 3 层）。

用法（窗口逻辑宽度**必须显式给**：截图的像素宽度推不出 DIP，Retina 2x 会让默认
值把 720 档算成 547，这正是要防的错）：

    python3 scripts/ui-baseline.py --width 1440 docs/domains/screens/state-default.png
    python3 scripts/ui-baseline.py --width 720  docs/domains/screens/window-default-720.png

输出每块区域的**首个暗像素列**（文字左缘）与侧栏的图标列/文字列，单位 DIP：
缩放 = 截图像素宽 / 给的窗口逻辑宽。没给宽度、或像素宽不是整数倍时报错退出，
不回落默认值。

判据（U2-forms §4）：主区标题行、时间线内容、输入卡文字、审批卡文字是同一个
数值；侧栏会话行有图标列与文字列各一条，跨行一致。
"""
import argparse
import sys

from PIL import Image

# 墨迹判据：与背景的亮度差超过 DELTA。亮色主题里 text #18181b 与背景 #ffffff
# 差得远；暗色主题里 text #fafafa 与背景 #18181b 同样差得远——用差值而不是
# 绝对阈值，两套外观用同一条判据（此前用绝对阈值，深色截图会把整片背景当成墨迹）。
DELTA = 60

# 各区域的取样带（DIP）：避开分隔线、卡片轮廓与图标盒的竖直方向。
# y 为负表示相对窗口底边。
REGIONS = {
    "标题行文字": (4, 44),
    "时间线内容": (110, 400),
    "输入卡文字": (-100, -70),
    "审批卡文字": (-170, -140),
}

SIDEBAR = 256


def luma(r, g, b):
    return 0.299 * r + 0.587 * g + 0.114 * b


def background_luma(img):
    """取窗口背景的亮度：左上角的空白区（标题行上方的让位条）。"""
    px = img.load()
    samples = [(4, 4), (img.width - 4, 4), (4, img.height - 4)]
    vals = [luma(*px[x, y][:3]) for x, y in samples]
    vals.sort()
    return vals[1]


def dark_left(img, x0, y0, y1, scale, run=0):
    """返回 [x0, ∞) 内第 run 段"有墨迹"的列（从 0 起）的起点（DIP）。

    按"列里有没有墨迹"把图切成段：一段图标、一段文字，中间是空白。墨迹 =
    与背景亮度差超过 DELTA，深浅两套外观同一判据。
    """
    px = img.load()
    bg = background_luma(img)
    runs = 0
    was_dark = False
    for x in range(int(x0 * scale), img.width):
        dark = False
        for y in range(int(y0 * scale), min(int(y1 * scale), img.height)):
            r, g, b = px[x, y][:3]
            if abs(luma(r, g, b) - bg) > DELTA:
                dark = True
                break
        if dark and not was_dark:
            if runs == run:
                return x / scale
            runs += 1
        was_dark = dark
    return None


def measure(path, window_width, window_height):
    img = Image.open(path).convert("RGB")
    scale = img.width / window_width
    if abs(scale - round(scale)) > 1e-6:
        raise SystemExit(
            f"{path}: 截图宽 {img.width}px 不是窗口宽 {window_width} 的整数倍；"
            "请确认 --width 是这个窗口的逻辑宽度"
        )
    # 横纵比必须一致：只看宽度会被"1440 逻辑宽 + 2x 像素"骗过去（720 档就长这样）。
    height_scale = img.height / window_height
    if abs(height_scale - scale) > 1e-6:
        raise SystemExit(
            f"{path}: 横纵缩放不一致（宽 {scale}x、高 {height_scale}x）；"
            f"请确认 --width/--height 是这个窗口的逻辑尺寸"
        )
    out = {"截图": path, "像素": f"{img.width}×{img.height}", "缩放": f"{round(scale)}x"}

    for name, (y0, y1) in REGIONS.items():
        if name == "审批卡文字" and "waiting" not in path:
            continue
        if y0 < 0:
            y0, y1 = window_height + y0, window_height + y1
        out[name] = dark_left(img, SIDEBAR, y0, y1, scale)

    # 侧栏：会话行的图标列与文字列（头部与页脚只有图标，不参与文字列）。
    # 一个会话行可能先有状态图标、再有文字：两段都报，标注是第几段。
    out["侧栏行第 1 段"] = dark_left(img, 0, 118, 136, scale)
    out["侧栏行第 2 段"] = dark_left(img, 0, 118, 136, scale, run=1)
    return out


def main():
    ap = argparse.ArgumentParser(
        description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    ap.add_argument("--width", type=int, required=True,
                    help="窗口逻辑宽度（DIP）：1440 / 900 / 720")
    ap.add_argument("--height", type=int, default=900, help="窗口逻辑高度（DIP），默认 900")
    ap.add_argument("images", nargs="+", help="截图路径")
    args = ap.parse_args()

    for path in args.images:
        row = measure(path, args.width, args.height)
        print(f"== {path}（{row['像素']}，{row['缩放']}，窗口 {args.width}×{args.height}）")
        for key, value in row.items():
            if key in ("截图", "像素", "缩放"):
                continue
            if value is None:
                print(f"  {key}: —（该区域没有暗像素）")
            else:
                print(f"  {key}: {value:.1f}")


if __name__ == "__main__":
    sys.exit(main())
