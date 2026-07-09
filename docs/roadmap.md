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

## Next

### Generic companion config

Decouple "cat" into a per-creature `PetConfig` resource (mapping + actions +
personality), so any sprite works with no code. Rename the in-code states to
the neutral engine verbs and ship a `cat.tres` that reproduces current
behavior with no regression. See [Behavior Model](behavior-model.md) for the
design.

### Config UI

High-level knobs: scale, follow-cursor, action weights, etc.

Open question: in-app Godot settings panel vs standalone launcher/wizard.

## Later

### Voice + speech

Push-to-talk STT (local whisper.cpp) feeds the daemon; the pet replies via
bubbles and, eventually, TTS (local piper). Godot's built-in HTTP/audio make
this far simpler than the old Go stack did.
