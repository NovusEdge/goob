# Getting Started

## Run

Open the project in Godot 4 and press **F5**, or:

```bash
godot --path .
```

Spawns a fullscreen transparent always-on-top overlay. Click-through everywhere
except the pet itself.

## Controls

| Input | Effect |
|-------|--------|
| Left-drag | Pick up and move |
| Right-click | Startle |
| Jiggle cursor near pet | Summon a chase |

## Custom sprites

Point `MANIFEST_PATH` / `SPRITE_PATH` in `main.gd` at your sheet + manifest.
Only `idle` is required; everything else falls back gracefully.

```json
{
  "sheet": "my-pet.png",
  "frameSize": [32, 32],
  "states": {
    "idle":  { "row": 0, "frames": 4, "fps": 3 },
    "walk":  { "row": 6, "frames": 8, "fps": 8 },
    "sleep": { "row": 47, "frames": 4, "fps": 2 }
  }
}
```

See [Behavior Model](behavior-model.md) for the full config format and how to
build a creature from scratch.

## Credits

Sprite packs included:

- **85-animation cat** by [BowPixel](https://bowpixel.itch.io/meow-cat-85-animation)
- **Simple cat** by [Elthen](https://elthen.itch.io/2d-pixel-art-cat-sprites)
