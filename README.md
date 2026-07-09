# goob — lil vro 🥀

A desktop pet cat that lives on your screen — wanders, naps, chases your cursor,
and reacts to what your machine is doing. Built in **Godot 4**. Bring your own
spritesheets.

## Run

Open the project in Godot 4 and press **F5**, or from the CLI:

```bash
godot --path .
```

It spawns a fullscreen, transparent, always-on-top overlay. The window is
click-through everywhere except the cat itself, so your desktop stays usable.

- **Left-drag** the cat to pick it up and move it.
- **Right-click** to startle it.
- **Jiggle your cursor** near it to summon a playful chase.

## How it works

- `main.gd` — window setup (transparent overlay + `mouse_passthrough` clipped to
  the cat's rect), input, and the frame loop. Builds the sprite animations and
  per-animation loop lengths at runtime from the JSON manifest.
- `pet.gd` (`PetBrain`) — the behavior state machine: idle/walk/chase/pounce,
  fidgets (sit, clean, sleep, stretch, yawn, loaf, roll…), gravity, drag, and
  system-driven **moods** (alert when a build is running, tired when the CPU is
  hot or the battery is low).
- `assets/*.json` — Unity-style spritesheet manifests (`row`, `frames`, `fps`).

## Custom sprites

Point `MANIFEST_PATH`/`SPRITE_PATH` in `main.gd` at your sheet + manifest. Only
`idle` is required — every other state falls back along a chain
(`run → walk → idle`, `sleep → loaf → sit → idle`, …) so a minimal sheet still
works and a full one shines.

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

## Roadmap

- **Generic companion config** — decouple "cat" into a per-creature `PetConfig`
  resource (mapping + actions + personality), so any sprite works with no code.
  Design: `docs/behavior-model.md`.
- **Config UI** for the high-level knobs (scale, follow-cursor, action weights…).
  Open question on form: in-app Godot settings panel (recommended, one stack) vs.
  a standalone launcher/wizard. TBD.
- **LLM integration** - voice input, the pet picks states and replies via chat
  bubbles (TTS later). See `docs/roadmap.md`.

## Credits

Sprite packs included:
- **85-animation cat** by [BowPixel](https://bowpixel.itch.io/meow-cat-85-animation) — grey & ginger
- **Simple cat** by [Elthen](https://elthen.itch.io/2d-pixel-art-cat-sprites)
- **Emote speech bubbles (32p)** by [Pooklea](https://pooklea.itch.io/emote-speech-bubble-32p)

Go support these artists!

---

*Previously implemented in Go (raylib + GTK4 layer-shell); that version lives in
git history — see the `wip: Go …` commit on the `godot-port` branch. Dragging a
native-Wayland surface fought the compositor at every turn, so we moved to Godot,
where the transparent overlay + click-through + input all come for free.*
