# Workflow

## Crop building menu screenshots into tiles

Name each screenshot after its category (e.g. `floor.png`, `wall.png`, `roof.png`).
Use the `/crop-tiles` command (or run the script directly).

### Slash command (via Claude)

```
/crop-tiles "screenshots/guild construction/Basic Structure/floor.png"
/crop-tiles "screenshots/guild construction/Basic Structure/wall.png"
```

### Script directly

```bash
python3 scripts/crop_tiles.py "screenshots/guild construction/Basic Structure/floor.png"
python3 scripts/crop_tiles.py "<image_path>" --prefix custom_name
python3 scripts/crop_tiles.py "<image_path>" --dry-run   # preview without saving
```

### Options

| Flag | Default | Description |
|------|---------|-------------|
| `--prefix NAME` | filename stem | Tile name prefix (e.g. `floor`, `wall`, `roof`) |
| `--dry-run` | off | Print detected grid without saving files |

Columns and rows are detected automatically from the image.

### Output

Tiles are saved next to the source image in a subfolder named after the prefix:

```
screenshots/guild construction/Basic Structure/
├── floor.png
├── floor/
│   ├── floor_11.png  floor_12.png  ...  floor_15.png
│   └── floor_21.png  floor_22.png  ...  floor_25.png
└── wall/
    ├── wall_11.png  ...
```

Row and column indices start at 1 (top-left = `_11`).
