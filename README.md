# goob — lil vro 🥀

A desktop pet cat that lives on your screen — wanders, naps, chases your cursor,
and reacts to what your machine is doing, commenting in a speech bubble from a
built-in line list or (optionally) an LLM daemon. Built in **Godot 4**, and not
hardcoded to a cat: bring your own spritesheet.

## Setup

**Required:** [Godot 4](https://godotengine.org/download) (4.3+, developed on
4.7 — the standard build, not the .NET/C# one). It's a single self-contained
binary; the pet has no other dependencies.

Make sure `godot` is on your `PATH` (or pass `GODOT=/path/to/godot` to the
`just` recipes below). [`just`](https://github.com/casey/just) is an optional
convenience — every recipe is a one-liner you can also run by hand.

```bash
just run          # or:  godot --path .
```

That spawns a fullscreen, transparent, always-on-top overlay, click-through
everywhere except the cat itself, so your desktop stays usable.

- **Left-drag** the cat to pick it up and move it.
- **Right-click** to pet it on the head.

> After adding a new `class_name` script, run `just check` once (a headless
> import) before `just run` — Godot only registers new global classes on a
> filesystem scan.

See **[docs/getting-started.md](docs/getting-started.md)** for install details,
the Wayland note, and how to bring your own creature.

### Optional: live LLM commentary

Out of the box goob comments from a built-in list of canned lines — no setup,
no network. Run the optional Python daemon and comments are generated live
instead (and the pet can nudge its own behaviour); stop it and it silently
falls back to canned lines.

Needs [`uv`](https://docs.astral.sh/uv/) and one provider key. Copy
`.env.example` to `.env`, set a key, then:

```bash
just daemon       # or:  uv run python -m daemon.server
```

`uv` pulls the daemon's deps (litellm) from `pyproject.toml` on first run.
Providers (Gemini / OpenAI / Vertex / Ollama), auth, and trigger cadence:
**[docs/llm-commentary.md](docs/llm-commentary.md)**.

### Optional: control-panel TUI

A terminal panel that launches and monitors the pet + daemon (status, CPU,
spend, live logs). Needs the Go toolchain.

```bash
just tui          # or:  cd tui && go run .
```

## How it works

- `main.gd` — window setup (transparent overlay + `mouse_passthrough` clipped to
  the cat's rect), input, mood sampling, the speech bubble, and the frame loop.
- `pet.gd` (`PetBrain`) — the behavior state machine over neutral engine verbs
  (`idle`, `wander`, `follow`, `dash`, `jump`, `zoomies`, grab/drop, `startle`…),
  per-creature fidgets, gravity, drag, and system-driven **moods** (alert when a
  build is running, tired when the CPU is hot or the battery is low).
- `pet_config.gd` (`PetConfig`) + `*.tres` — per-creature data: which SpriteFrames
  animation maps onto each engine verb, plus actions, weights, speeds, and mood
  reactions.
- `daemon/` — the optional Python LLM sidecar (one localhost route, `POST /tick`).
- `tui/` — the optional Go control panel.

## Make it your creature

goob is **not hardcoded to a cat.** A creature is two data pieces, no engine code:

1. A **SpriteFrames** — the animations, authored in the Godot editor.
2. A **PetConfig** `.tres` — maps those animations onto the engine verbs and sets
   the personality (actions, weights, speeds, mood reactions).

Only `idle` is required — every other verb falls back along a chain toward it, so
a minimal sheet still works and a full one shines. The bundled `cat.tres` is the
worked example. Full field reference: **[docs/configuration.md](docs/configuration.md)**;
design: **[docs/behavior-model.md](docs/behavior-model.md)**.

## Roadmap

Shipped: LLM/canned commentary, the control-panel TUI, and the generic
per-creature `PetConfig`. Next: a config UI for the high-level knobs; later,
voice input + TTS. Details in **[docs/roadmap.md](docs/roadmap.md)**.

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
