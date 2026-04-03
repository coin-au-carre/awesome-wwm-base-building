#!/usr/bin/env python3
"""
Crop a game UI grid screenshot into individual tile images.

Usage:
    python3 crop_tiles.py <image_path> [--prefix NAME] [--dry-run]

The script auto-detects:
  - Grid boundaries (dark background bands)
  - Number of columns and rows dynamically

Output files are saved next to the source image in a subfolder named after
the prefix. The prefix defaults to the image filename stem (e.g. floor.png -> floor).
"""

import argparse
import os
import sys
import numpy as np
from PIL import Image

TILE_SIZE = 200


def find_dark_bands(signal, threshold, min_width=3, offset=0):
    """Return list of (start, end) for contiguous runs below threshold."""
    in_band = False
    bands = []
    start = 0
    for i, v in enumerate(signal):
        if v < threshold and not in_band:
            in_band = True
            start = i
        elif v >= threshold and in_band:
            in_band = False
            if i - start >= min_width:
                bands.append((start + offset, i - 1 + offset))
    if in_band and len(signal) - start >= min_width:
        bands.append((start + offset, len(signal) - 1 + offset))
    return bands


def merge_close_bands(bands, gap=20):
    """Merge bands that are within `gap` pixels of each other."""
    if not bands:
        return bands
    merged = [list(bands[0])]
    for b in bands[1:]:
        if b[0] - merged[-1][1] <= gap:
            merged[-1][1] = b[1]
        else:
            merged.append(list(b))
    return [tuple(b) for b in merged]


def detect_grid(arr):
    """
    Auto-detect tile grid boundaries.
    Returns (col_ranges, row_ranges) as lists of (start, end) pixel ranges.
    """
    col_brightness = arr.mean(axis=(0, 2))
    row_brightness = arr.mean(axis=(1, 2))

    # Threshold: just above the dark background floor
    col_threshold = col_brightness.min() + 5
    row_threshold = row_brightness.min() + 5

    col_bands = merge_close_bands(find_dark_bands(col_brightness, col_threshold, min_width=3))
    row_bands = merge_close_bands(find_dark_bands(row_brightness, row_threshold, min_width=3))

    if len(col_bands) < 2:
        raise ValueError(f"Could not detect column separators (threshold={col_threshold:.1f}).")
    if len(row_bands) < 2:
        raise ValueError(f"Could not detect row separators (threshold={row_threshold:.1f}).")

    # Content ranges = gaps between consecutive dark bands
    all_col_ranges = [
        (col_bands[i][1] + 1, col_bands[i + 1][0] - 1)
        for i in range(len(col_bands) - 1)
        if col_bands[i + 1][0] - 1 > col_bands[i][1] + 1
    ]

    # Filter out narrow false columns (< 50% of widest column)
    max_w = max(e - s for s, e in all_col_ranges)
    col_ranges = [(s, e) for s, e in all_col_ranges if (e - s) >= max_w * 0.5]

    all_row_ranges = [
        (row_bands[i][1] + 1, row_bands[i + 1][0] - 1)
        for i in range(len(row_bands) - 1)
        if row_bands[i + 1][0] - 1 > row_bands[i][1] + 1
    ]

    if not all_row_ranges:
        raise ValueError("No row content ranges found.")

    # Filter out short UI rows (buttons/tabs): keep rows >= 50% of the tallest
    max_h = max(e - s for s, e in all_row_ranges)
    row_ranges = [(s, e) for s, e in all_row_ranges if (e - s) >= max_h * 0.5]

    return col_ranges, row_ranges


def crop_tiles(image_path, prefix=None, dry_run=False):
    img = Image.open(image_path)
    arr = np.array(img)

    if prefix is None:
        prefix = os.path.splitext(os.path.basename(image_path))[0]

    print(f"Image: {image_path}  ({img.size[0]}x{img.size[1]})")
    print(f"Prefix: {prefix}")

    col_ranges, row_ranges = detect_grid(arr)

    print(f"Detected: {len(row_ranges)} row(s) x {len(col_ranges)} col(s)")
    for i, (s, e) in enumerate(col_ranges):
        print(f"  col {i+1}: x={s}-{e}  ({e-s}px)")
    for i, (s, e) in enumerate(row_ranges):
        print(f"  row {i+1}: y={s}-{e}  ({e-s}px)")

    if dry_run:
        print("Dry run — no files written.")
        return

    out_dir = os.path.join(os.path.dirname(image_path), prefix)
    os.makedirs(out_dir, exist_ok=True)

    half = TILE_SIZE // 2
    for r, (y1, y2) in enumerate(row_ranges):
        # Anchor to bottom of cell so items extending upward aren't clipped
        cy = y2 - half
        for c, (x1, x2) in enumerate(col_ranges):
            cx = (x1 + x2) // 2
            tile = img.crop((cx - half, cy - half, cx + half, cy + half))
            name = f"{prefix}_{r+1}{c+1}.png"
            tile.save(os.path.join(out_dir, name), format="PNG")
            print(f"  Saved {name}  (center={cx},{cy})")

    print(f"Done → {out_dir}/")


def main():
    parser = argparse.ArgumentParser(description="Crop game UI grid screenshot into tiles.")
    parser.add_argument("image", help="Path to the source screenshot")
    parser.add_argument("--prefix", help="Tile name prefix. Defaults to the image filename stem.")
    parser.add_argument("--dry-run", action="store_true",
                        help="Detect grid and print info without saving files")
    args = parser.parse_args()

    if not os.path.isfile(args.image):
        print(f"Error: file not found: {args.image}", file=sys.stderr)
        sys.exit(1)

    try:
        crop_tiles(args.image, prefix=args.prefix, dry_run=args.dry_run)
    except ValueError as e:
        print(f"Error: {e}", file=sys.stderr)
        sys.exit(1)


if __name__ == "__main__":
    main()
