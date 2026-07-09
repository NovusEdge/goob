# Roadmap

## In progress

### Generic companion config

Decouple "cat" into a per-creature `PetConfig` resource (mapping + actions +
personality), so any sprite works with no code.

See [Behavior Model](behavior-model.md) for the design.

### Config UI

High-level knobs: scale, follow-cursor, action weights, etc.

Open question: in-app Godot settings panel vs standalone launcher/wizard.

## Planned

### LLM integration

Voice input, the pet picks states and replies via chat bubbles. TTS later.

Key constraints:
- Frame loop runs at 60fps, must never block
- LLM calls happen off-loop via goroutine/channel
- Tool use with enum whitelist for state selection
- Local whisper.cpp for STT (push-to-talk, not always-listening)
- Local piper for TTS

MVP: text input first, voice bolted on after.
