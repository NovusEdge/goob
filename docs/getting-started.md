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
| Right-click | Pet it on the head |

## LLM commentary (optional)

goob comments on what your machine is doing. Out of the box it uses a built-in
list of canned lines — no setup, no network. For live, in-character remarks,
run the optional Python daemon (needs [uv](https://docs.astral.sh/uv/)):

1. `cp .env.example .env`, then edit `.env` — pick a provider block and fill in
   your model + key (Vertex AI, Google AI Studio, OpenAI, or local Ollama).
2. `just daemon` — it auto-loads `.env` and uses `uv` (reading `pyproject.toml`)
   to fetch litellm, so there's no manual install and no `pip`.

While the daemon runs, goob's comments are generated live and it can nudge its
own behaviour. Stop the daemon and goob silently falls back to canned lines.

### Configuring the pet

Everything tweakable lives in `config/`:

| File | What it is | Edit it? |
|------|-----------|----------|
| `config/PERSONALITY.md` | The pet's **voice/character** — fed to the LLM as its personality. | Yes — make it yours. |
| `config/PROMPT.md` | The **engine contract** (tools, the `emit` output format, constraints). | Rarely — only if you change how the daemon works. |
| `config/comments.json` | The **canned lines** shown when no daemon is running, keyed by mood (`neutral`/`alert`/`tired`). | Yes — add your own. |

The LLM's system prompt is `PROMPT.md` + `PERSONALITY.md` composed together, so
you can rewrite the personality without touching the mechanics. Both fall back
to a built-in default if the file is missing.

**Fully local / private:** litellm speaks to Ollama too — install Ollama, then
`export GOOB_MODEL=ollama/llama3` (or any pulled model). Nothing leaves your
machine. Only the daemon ever reads system state, and only an allowlisted
digest (watched processes, battery, thermal) — never your full process list.

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
- **Emote speech bubbles (32p)** by [Pooklea](https://pooklea.itch.io/emote-speech-bubble-32p)
