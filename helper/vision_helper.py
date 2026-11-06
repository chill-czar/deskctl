#!/usr/bin/env python3
"""
Vision and Template Matching Helper for deskctl.
Provides fast template matching (find-image) and screen recording management.
"""

import os
import sys
import json
import time
import signal
import argparse
import subprocess
from PIL import Image

def find_template(screen_path, template_path, threshold=0.8):
    try:
        screen_img = Image.open(screen_path).convert("RGB")
        tmpl_img = Image.open(template_path).convert("RGB")
    except Exception as e:
        return {"ok": False, "error": f"Failed to load images: {e}"}

    sw, sh = screen_img.size
    tw, th = tmpl_img.size

    if tw > sw or th > sh:
        return {"ok": False, "error": "Template image is larger than screen image"}

    # Optimization: if template is large, downscale for coarse pass then refine
    # Fast numpy-like pixel buffer comparison in Python
    screen_bytes = screen_img.tobytes()
    tmpl_bytes = tmpl_img.tobytes()

    # Step sampling for speed
    best_score = 0.0
    best_loc = None

    # Use PIL's fast image difference or pixel matching
    # Check 16 sample points in template for quick rejection
    sample_points = []
    step_x = max(1, tw // 4)
    step_y = max(1, th // 4)
    for sy in range(0, th, step_y):
        for sx in range(0, tw, step_x):
            sample_points.append((sx, sy, tmpl_img.getpixel((sx, sy))))

    screen_pix = screen_img.load()
    tmpl_pix = tmpl_img.load()

    total_samples = len(sample_points)
    max_diff = total_samples * 255 * 3

    for y in range(0, sh - th + 1, 2):
        for x in range(0, sw - tw + 1, 2):
            diff = 0
            # Quick check on sample points
            for sx, sy, (tr, tg, tb) in sample_points:
                sr, sg, sb = screen_pix[x + sx, y + sy]
                diff += abs(sr - tr) + abs(sg - tg) + abs(sb - tb)
                if diff > max_diff * (1.0 - threshold):
                    break
            else:
                score = 1.0 - (diff / max_diff)
                if score > best_score:
                    best_score = score
                    best_loc = (x, y)

    if best_loc and best_score >= threshold:
        bx, by = best_loc
        return {
            "ok": True,
            "found": True,
            "match": {"x": bx, "y": by, "w": tw, "h": th},
            "center": {"x": bx + tw // 2, "y": by + th // 2},
            "score": round(best_score, 4)
        }
    else:
        return {
            "ok": True,
            "found": False,
            "best_score": round(best_score, 4),
            "threshold": threshold
        }

def cmd_find_image(args):
    res = find_template(args.screen_image, args.template_image, threshold=args.threshold)
    print(json.dumps(res))

def main():
    parser = argparse.ArgumentParser(description="Vision helper for deskctl")
    sub = parser.add_subparsers(dest="cmd", required=True)

    p_find = sub.add_parser("find-image")
    p_find.add_argument("screen_image", type=str)
    p_find.add_argument("template_image", type=str)
    p_find.add_argument("--threshold", type=float, default=0.8)

    args = parser.parse_args()
    if args.cmd == "find-image":
        cmd_find_image(args)

if __name__ == "__main__":
    main()
