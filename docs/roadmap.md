# Roadmap

## Shipped

### Ambient LLM commentary

An opt-in Python daemon that observes read-only machine facts (mood, running
builds, battery/thermals) and lets the pet comment via speech bubbles — LLM
mode with a personality, or a canned-comment fallback with no API key. See
[LLM Commentary](llm-commentary.md).

### Control-panel TUI

`just tui` launches a terminal panel that starts/monitors the pet and daemon
and shows status, CPU, spend, and live logs.

### Generic companion config

"Cat" is decoupled into a per-creature `PetConfig` resource (sprite mapping +
actions + personality + speeds + mood weights), so any sprite works with no
engine code. In-code states use neutral engine verbs; `cat.tres`,
`playful_cat.tres`, and `lazy_cat.tres` ship as examples. See
[Behavior Model](behavior-model.md) and [Configuration](configuration.md).

## Next

### Config UI

High-level knobs: scale, follow-cursor, action weights, etc.

Open question: in-app Godot settings panel vs standalone launcher/wizard.

## Later

### Voice + speech

Push-to-talk STT (local whisper.cpp) feeds the daemon; the pet replies via
bubbles and, eventually, TTS (local piper). Godot's built-in HTTP/audio make
this far simpler than the old Go stack did.

## Tracked issues

Smaller enhancements and help-wanted items live on the issue tracker:

- [#3 FloatPet: floating (no-gravity) pet class](https://github.com/NovusEdge/goob/issues/3)
- [#2 Precise user-idle detection for retreat (ext-idle-notify)](https://github.com/NovusEdge/goob/issues/2)
- [#1 Native Wayland support via GTK4 + layer-shell](https://github.com/NovusEdge/goob/issues/1)
