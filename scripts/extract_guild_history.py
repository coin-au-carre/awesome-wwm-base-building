#!/usr/bin/env python3
"""Extract guild scores from git history, one snapshot per day (Apr 4-24)."""

import json
import subprocess
import csv
import sys
from datetime import date

START = "2026-04-04"
END = "2026-04-24"
OUTPUT = "scripts/guild_history.csv"


def git(*args):
    return subprocess.check_output(["git"] + list(args), text=True)


def one_commit_per_day():
    """Return {date_str: commit_hash} using the latest commit of each day."""
    log = git("log", "--follow", "--format=%H %ci", "--", "data/guilds.json")
    seen = {}
    for line in log.splitlines():
        parts = line.split()
        if len(parts) < 2:
            continue
        commit, day = parts[0], parts[1]
        if START <= day <= END and day not in seen:
            seen[day] = commit
    return dict(sorted(seen.items()))


def load_guilds(commit):
    for path in ("data/guilds.json", "guilds.json"):
        try:
            raw = git("show", f"{commit}:{path}")
            return json.loads(raw)
        except subprocess.CalledProcessError:
            continue
    return None


def main():
    commits = one_commit_per_day()
    print(f"Found {len(commits)} days: {min(commits)} → {max(commits)}")

    rows = []
    for day, commit in commits.items():
        guilds = load_guilds(commit)
        if guilds is None:
            print(f"  WARNING: skipping {day} ({commit[:7]}) — file not found at that commit")
            continue
        for g in guilds:
            name = g.get("name") or g.get("guildName") or "?"
            score = g.get("score", 0)
            if score > 0:
                rows.append({"date": day, "guild": name, "score": score})

    with open(OUTPUT, "w", newline="") as f:
        writer = csv.DictWriter(f, fieldnames=["date", "guild", "score"])
        writer.writeheader()
        writer.writerows(rows)

    print(f"Wrote {len(rows)} rows to {OUTPUT}")

    # Quick sanity: show guild count per day
    from collections import Counter
    counts = Counter(r["date"] for r in rows)
    for day in sorted(counts):
        print(f"  {day}: {counts[day]} guilds")


if __name__ == "__main__":
    main()
