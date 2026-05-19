#!/usr/bin/env python3
"""Refresh Discord CDN tokens in community-flyer.html from local data files."""

import json
import re
import sys
from pathlib import Path

ROOT = Path(__file__).parent.parent.parent
FLYER = Path(__file__).parent / "community-flyer.html"

html = FLYER.read_text()

guilds = json.loads((ROOT / "data/guilds.json").read_text())
solos  = json.loads((ROOT / "data/solos.json").read_text())
try:
    whatever = json.loads((ROOT / "data/whatever.json").read_text())
except FileNotFoundError:
    whatever = []

# Index guilds+solos by Discord thread ID
thread_index: dict[str, dict] = {}
for entry in guilds + solos:
    t = entry.get("discordThread", "")
    m = re.search(r"/(\d+)$", t)
    if m:
        thread_index[m.group(1)] = entry

def best_image(entry: dict) -> str | None:
    img = entry.get("coverImage")
    if not img:
        imgs = entry.get("screenshots") or []
        img = imgs[0] if imgs else None
    return img

# Index whatever.json entries by id
whatever_index: dict[str, dict] = {e["id"]: e for e in whatever if "id" in e}

changes = 0

# Replace every CDN URL in the flyer
def replace_url(m: re.Match) -> str:
    global changes
    url = m.group(0)

    # Extract thread ID (channel ID in Discord attachment URLs)
    tid_m = re.search(r"/attachments/(\d+)/", url)
    if not tid_m:
        return url
    tid = tid_m.group(1)

    # Check if this thread belongs to a guild/solo
    entry = thread_index.get(tid)
    if entry:
        fresh = best_image(entry)
        if fresh and fresh != url:
            print(f"  {entry.get('name', tid)}: updated")
            changes += 1
            return fresh
        return url

    # Check whatever.json: look for the comment on the preceding line
    # Parse whatever.json id from the nearby HTML comment
    # We search the whole HTML for the id comment near this URL
    return url

# Handle mosaic + featured separately for whatever.json entries
# Step 1: replace guild/solo images by thread ID
html = re.sub(
    r"https://cdn\.discordapp\.com/attachments/[^\s\"']+",
    replace_url,
    html,
)

# Step 2: replace whatever.json images via comment annotations
# Pattern: <!-- In whatever.json Nth image of "id": "XXXX" -->  followed by <img src="...">
def replace_whatever(m: re.Match) -> str:
    global changes
    comment, n_str, entry_id, before_src, old_url, after = (
        m.group(1), m.group(2), m.group(3), m.group(4), m.group(5), m.group(6),
    )
    entry = whatever_index.get(entry_id)
    if not entry:
        print(f"  WARNING: whatever.json entry {entry_id} not found", file=sys.stderr)
        return m.group(0)
    idx = int(n_str) - 1
    imgs = entry.get("images", entry.get("screenshots", []))
    if idx >= len(imgs):
        print(f"  WARNING: whatever.json entry {entry_id} has no image #{n_str}", file=sys.stderr)
        return m.group(0)
    fresh = imgs[idx]
    if fresh != old_url:
        print(f"  whatever.json [{entry_id}] image {n_str}: updated")
        changes += 1
    return comment + before_src + fresh + after

html = re.sub(
    r'(<!--[^>]*?whatever\.json\s+(\d+)(?:st|nd|rd|th)?\s+image\s+of\s+"id":\s*"(\d+)"[^>]*?-->)'
    r'(\s*<img\s+src=")([^"]+)(")',
    replace_whatever,
    html,
    flags=re.DOTALL,
)

FLYER.write_text(html)
print(f"\n{changes} URL(s) updated in {FLYER.name}")
