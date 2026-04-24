#!/usr/bin/env python3
"""Render guild ranking race video."""

import pandas as pd
import matplotlib.pyplot as plt
import matplotlib.font_manager as fm
import bar_chart_race as bcr

CSV    = "scripts/guild_history.csv"
OUTPUT = "scripts/guild_race.mp4"
FONT   = "Noto Sans CJK JP"
TOP_N  = 20

# Ensure font cache includes system fonts installed after pip
fm.fontManager.__init__(fm.fontManager)
plt.rcParams["font.family"] = FONT

df   = pd.read_csv(CSV)
wide = df.pivot_table(index="date", columns="guild", values="score", aggfunc="max")
wide = wide.sort_index().ffill()

# Keep only guilds that ever reach the top 30
top_guilds = wide.max().nlargest(30).index
wide = wide[top_guilds]

print(f"Rendering {len(wide)} days, top {TOP_N} guilds visible → {OUTPUT}")

bcr.bar_chart_race(
    df=wide,
    filename=OUTPUT,
    orientation="h",
    sort="desc",
    n_bars=TOP_N,
    steps_per_period=15,
    period_length=800,
    figsize=(14, 8),
    cmap="dark24",
    title="Awesome WWM — Guild Rankings",
    title_size=16,
    bar_label_size=0,       # hide raw scores
    tick_label_size=10,
    period_label={"x": 0.98, "y": 0.05, "ha": "right", "size": 28,
                  "color": "#555555", "fontweight": "bold"},
    filter_column_colors=True,
)

print("Done.")
