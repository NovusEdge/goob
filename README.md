# goob — lil vro 🥀

A desktop pet that roams your screen. Bring your own spritesheets.

## Usage

```bash
go build ./cmd/goob
./goob -manifest assets/cat-sprites.json
```

## Custom Sprites

Drop any Unity-style spritesheet and create a manifest:

```json
{
  "sheet": "my-pet.png",
  "frameSize": [32, 32],
  "states": {
    "idle":  { "row": 0, "frames": 4, "fps": 3 },
    "walk":  { "row": 1, "frames": 6, "fps": 8 },
    "sleep": { "row": 2, "frames": 2, "fps": 2 }
  }
}
```

## Credits

Default cat sprites by [Elthen](https://elthen.itch.io/2d-pixel-art-cat-sprites) — go support them!
