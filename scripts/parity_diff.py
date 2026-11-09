#!/usr/bin/env python3
"""Measure per-region visual parity between a reference and a candidate frame.

Both inputs are resized to the reference size, then five regions are compared:
the sidebar, the top bar, the main content, the composer and the bottom strip.
For each region the tool reports mean absolute error per channel (0-255) and the
share of pixels whose largest channel delta exceeds 16. A diff image is written
next to the report so a reviewer can see where the frames diverge.

This is a measurement tool for D-series acceptance, not a pass/fail gate by
itself: the thresholds live in docs/acceptance.md.
"""

from __future__ import annotations

import argparse
import json
from pathlib import Path

from PIL import Image, ImageChops

REGIONS = {
    "sidebar": (0.0, 0.0, 0.32, 1.0),
    "top": (0.32, 0.0, 1.0, 0.06),
    "content": (0.32, 0.06, 1.0, 0.75),
    "composer": (0.32, 0.75, 1.0, 0.95),
    "bottom": (0.32, 0.95, 1.0, 1.0),
}


def crop(size: tuple[int, int], box: tuple[float, float, float, float]) -> tuple[int, int, int, int]:
    width, height = size
    left, top, right, bottom = box
    return (round(left * width), round(top * height), round(right * width), round(bottom * height))


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("reference")
    parser.add_argument("candidate")
    parser.add_argument("--out", default="")
    args = parser.parse_args()

    reference = Image.open(args.reference).convert("RGB")
    candidate = Image.open(args.candidate).convert("RGB").resize(reference.size, Image.LANCZOS)
    diff = ImageChops.difference(reference, candidate)

    report = {"reference": args.reference, "candidate": args.candidate, "size": list(reference.size), "regions": {}}
    for name, box in REGIONS.items():
        rect = crop(reference.size, box)
        ref = reference.crop(rect)
        got = candidate.crop(rect)
        delta = diff.crop(rect)
        pixels = list(delta.getdata())
        count = len(pixels)
        total = 0
        over = 0
        for r, g, b in pixels:
            worst = max(r, g, b)
            total += r + g + b
            if worst > 16:
                over += 1
        report["regions"][name] = {
            "rect": rect,
            "mae": round(total / (count * 3), 2),
            "over16": round(100.0 * over / count, 2),
        }
    print(json.dumps(report, indent=2))

    out = Path(args.out) if args.out else Path(args.candidate).with_suffix(".diff.png")
    diff.save(out)


if __name__ == "__main__":
    main()
