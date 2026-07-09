# Getting Started

## Prerequisites

goob is a **Godot 4 project** — you need the Godot 4 engine to run it. That's the
only requirement: Godot ships as a single self-contained binary, and the pet has
no other dependencies.

1. Get **Godot 4** (4.3 or newer; developed on 4.7) — the standard build, not the
   .NET/C# one:
   - Download from [godotengine.org/download](https://godotengine.org/download), **or**
   - Arch: `pacman -S godot` · macOS: `brew install godot` · Windows: `winget install GodotEngine.GodotEngine`
2. Make sure `godot` is on your `PATH` (or note the binary path — you can pass it
   as `GODOT=/path/to/godot`).
3. Clone this repo.

## Run

From the repo root:

```bash
just run          # if you have `just` installed
# or
godot --path .
```

Or open the folder in the Godot editor and press **F5**.

It spawns a fullscreen, transparent, always-on-top overlay. It's click-through
everywhere except the pet itself, so your desktop stays fully usable.

> **Wayland note:** goob is developed on Wayland and relies on a transparent
> overlay + mouse passthrough. If transparency or click-through misbehaves,
> that's compositor-dependent — start there.

> **Tip:** after adding a new `class_name` script, run `just check` once (a
> headless import) before `just run`. Godot only registers new global classes on
> a filesystem scan, so a fresh script can otherwise trip a "Could not find type"
> parser error.

## Controls

| Input | Effect |
|-------|--------|
| Left-drag | Pick up and move |
| Right-click | Startle |
| Jiggle cursor near pet | Summon it to follow you |

## Make it your creature

goob is **not hardcoded to a cat.** A creature is defined by two data pieces —
no engine code:

1. A **SpriteFrames** — the animations, authored in the Godot editor (on the
   `AnimatedSprite2D` under `Main`, or referenced from the config).
2. A **PetConfig** resource (`.tres`) — maps those animations onto the engine's
   behaviors and sets the creature's personality (actions, weights, speeds).

The bundled cat is `cat.tres`. To build your own, see
[Configuration](configuration.md) for every knob, and
[Behavior Model](behavior-model.md) for how the engine thinks.

## Credits

Sprite packs included:

- **85-animation cat** by [BowPixel](https://bowpixel.itch.io/meow-cat-85-animation)
- **Simple cat** by [Elthen](https://elthen.itch.io/2d-pixel-art-cat-sprites)
