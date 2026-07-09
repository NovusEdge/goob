# goob

A desktop pet that lives on your screen. Wanders, naps, chases your cursor,
and reacts to what your machine is doing. Built in **Godot 4**.

Bring your own spritesheets to turn it into any creature: cat, dog, robot, slime.

## Quick start

Requires **Godot 4** (see [Getting Started](getting-started.md) to install it).

```bash
just run        # or: godot --path .
```

- **Left-drag** to pick it up
- **Right-click** to pet it

It also wanders, chases your cursor on its own, and — with the optional
[LLM daemon](llm-commentary.md) running — comments on what your machine is
doing via speech bubbles.

## Docs

- [Getting Started](getting-started.md) - run it, customize sprites
- [Behavior Model](behavior-model.md) - how the engine thinks, bring your own creature
- [Configuration](configuration.md) - runtime settings
- [LLM Commentary](llm-commentary.md) - the optional ambient-comment daemon
- [Reference](reference.md) - technical details for extending
- [Roadmap](roadmap.md) - what's planned
