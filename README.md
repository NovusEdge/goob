# goob — lil vro 🥀

A desktop pet that roams your screen. Bring your own spritesheets.

## Features

- Custom spritesheet support (Unity-style grid layouts)
- Multiple behavior states: idle, walk, chase, clean, sleep, paw, jump, scared
- Cursor chasing
- Gravity (pet falls to screen bottom)
- Transparent, click-through window

## Dependencies

### Linux (X11)
```bash
# Arch
sudo pacman -S libx11

# Debian/Ubuntu  
sudo apt install libx11-dev
```

### Linux (Wayland) — WIP
```bash
# Arch
sudo pacman -S gtk4-layer-shell

# Wayland support is work-in-progress
```

### macOS / Windows
No additional dependencies (uses raylib defaults).

## Build

```bash
go build ./cmd/goob
```

## Usage

```bash
./goob                                    # default cat sprite
./goob -manifest assets/my-pet.json       # custom sprite
./goob -scale 6                           # adjust size (default: 8)
./goob 2>/dev/null                        # suppress warnings
```

## Custom Sprites

Drop any Unity-style spritesheet and create a manifest. **Only `idle` is
required** — every other state falls back along a chain until it lands on one
your sheet defines, so a two-row sheet works and a full one shines.

### State vocabulary

The engine drives this fixed set. Declare the ones you have; the rest substitute
via the arrow (ultimately → `idle`). `xN` variants are picked at random if present.

| State | Falls back to | When it plays |
|-------|---------------|---------------|
| `idle` / `idle2` | — | loitering (required) |
| `walk` / `walk2` | idle | wandering to a spot |
| `run` | walk → idle | chasing the cursor |
| `pounce` | paw → idle | reaching the cursor |
| `chase` | (uses `run`) | — |
| `sit` / `sit2` | idle | resting upright |
| `loaf` | sit → idle | loafing |
| `sleep` | loaf → sit → idle | dozing |
| `clean` / `clean2` | idle | grooming |
| `stretch` `yawn` `meow` `roll` `paw` | idle | fidgets |
| `jump` | idle | hopping |
| `scared` | idle | startled (right-click) |
| `spawn` | idle | first appearance |
| `pickup` `held` / `held2` `putdown` | sit / idle | drag & drop |

### Reactive moods

goob samples the machine every ~2s and shifts disposition: **alert** when a
build/dev process is running (paces, watches, bats), **tired** when the CPU is
hot or the battery is low (flops, dozes, yawns). Tweak the watched process names
in `internal/sysmon`.

### Manifest example

```json
{
  "sheet": "my-pet.png",
  "frameSize": [32, 32],
  "states": {
    "idle":    { "row": 0, "frames": 4, "fps": 3 },
    "walk":    { "row": 1, "frames": 6, "fps": 8 },
    "sleep":   { "row": 2, "frames": 2, "fps": 2 },
    "chase":   { "row": 3, "frames": 4, "fps": 10 },
    "jump":    { "row": 4, "frames": 3, "fps": 6 },
    "scared":  { "row": 5, "frames": 2, "fps": 8 }
  }
}
```

## Platform Notes

| Platform | Window Positioning | Cursor Tracking |
|----------|-------------------|-----------------|
| X11      | ✓                 | ✓               |
| Wayland  | ✗ (WIP)           | ✓ (via XWayland)|
| Windows  | ✓                 | TBD             |
| macOS    | ✓                 | TBD             |

## Credits

Sprite packs included:
- **85-animation cat** by [BowPixel](https://bowpixel.itch.io/meow-cat-85-animation) — grey & ginger variants
- **Simple cat** by [Elthen](https://elthen.itch.io/2d-pixel-art-cat-sprites)

Go support these artists!
