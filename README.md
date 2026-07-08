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

Drop any Unity-style spritesheet and create a manifest:

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

Default cat sprites by [Elthen](https://elthen.itch.io/2d-pixel-art-cat-sprites) — go support them!
