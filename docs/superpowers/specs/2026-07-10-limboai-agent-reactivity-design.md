# goob — agent-reactive behavior via LimboAI

**Date:** 2026-07-10
**Status:** approved design, pre-implementation

## Goal

Make the pet react, in near-real-time, to what your local AI coding agent
(Claude Code) is doing — thinking when you prompt, busy while tools run,
zoomies on a subagent, celebrate when a task finishes, startle on error, sleep
when the session ends. Idea borrowed from `clawd-on-desk`; the reaction is
deterministic and works with zero daemon, with an optional LLM-narrated
`say` layered on top when the daemon is up.

This is also the first use of **LimboAI** in goob: the pet's top-level behavior
becomes a hierarchical state machine (HSM), and the agent-reaction dispatch
becomes a behavior tree (BT). We learn HSM+BT on one real vertical slice
instead of rewriting the whole brain.

## Architecture

Two orthogonal concerns:

- **Transport** — how an agent event reaches the pet. A Claude Code hook writes
  the latest event to one small JSON file; the pet polls it. No server, no
  ports, no connection errors; the hook writes whether or not the pet/daemon
  are running.
- **Behavior** — how the pet decides what to do. A `LimboHSM` owns top-level
  modes; a BT dispatches the reaction.

```
Claude Code hook  ──stdin JSON──▶  hooks/goob_hook.py (stdlib)
                                        │ atomic write
                                        ▼
                              /tmp/goob-agent.json   {event, ts}
                                        │ AgentEventPoller reads ~4 Hz
                                        ▼  sets blackboard agent_event / agent_event_ts
   LimboHSM (pet root)
   ├── Ambient    ← default; delegates to existing pet.gd action-picking
   ├── Reacting   ← guard: agent_event fresh (< 8 s)
   │   └── BTState → BTPlayer runs the reaction tree (below)
   └── Sleeping   ← agent SessionEnd, or user idle
```

### Reaction tree (inside `Reacting`)

```
BTSelector
├── BTSequence [BTCondition event==done]     → PlayZoomies, EmitSay("✅ done")
├── BTSequence [BTCondition event==error]    → Startle,     EmitSay("💥")
├── BTSequence [BTCondition event==subagent] → PlayZoomies
└── BusyIdle                                  (thinking / working fallback)
```

Custom tasks are GDScript (`extends BTAction`, override `_tick()` →
`SUCCESS/FAILURE/RUNNING`; `_enter/_exit` as needed). They appear live in
LimboAI's visual debugger as they tick.

## Components & interfaces

### 1. `hooks/goob_hook.py`
- **Does:** reads Claude Code hook JSON on stdin, maps the hook event name to a
  goob event token, atomically writes `/tmp/goob-agent.json` = `{"event": <token>, "ts": <unix>}`.
- **Depends on:** Python 3 stdlib only. No uv env, no deps — runs even if the
  daemon is never installed.
- **Invoked by:** a documented `~/.claude/settings.json` hooks snippet (manual
  install; not auto-written into the user's config).
- **Atomic write:** write temp + `os.replace` so the poller never reads a torn file.

### Event mapping (Claude Code hook → goob token)
Only the high-signal events; unknown events are ignored (no file write).

| Claude Code hook   | goob token | HSM/BT result        |
|--------------------|-----------|-----------------------|
| UserPromptSubmit   | thinking  | Reacting → BusyIdle   |
| PreToolUse         | working   | Reacting → BusyIdle   |
| SubagentStart      | subagent  | Reacting → PlayZoomies|
| Stop               | done      | Reacting → Zoomies+say|
| PostToolUseFailure / StopFailure | error | Reacting → Startle+say |
| SessionEnd         | sleep     | Sleeping              |

### 2. `AgentEventPoller` (Godot node)
- **Does:** every ~0.25 s, reads `/tmp/goob-agent.json`; if `ts` changed, sets
  blackboard `agent_event` (token) and `agent_event_ts`. On `sleep`, no
  freshness window — it latches until a newer event arrives.
- **Depends on:** the LimboHSM's blackboard.
- **Freshness:** the HSM treats `agent_event` as stale after 8 s → falls back to
  `Ambient`. Keeps the pet from freezing mid-"working" if the agent is killed.

### 3. `LimboHSM` on the pet
- **States:** `Ambient`, `Reacting` (a `BTState`), `Sleeping`.
- **Ambient:** delegates to existing `pet.gd` action-picking (wander/idle/jump/
  zoomies by `.tres` weights). **`pet.gd` is not rewritten** — absorbed into the
  tree later once the paradigm is comfortable.
- **Transitions:** `Ambient → Reacting` when `agent_event` fresh and != `sleep`;
  `Reacting → Ambient` when stale; `* → Sleeping` when token == `sleep`;
  `Sleeping → Reacting/Ambient` on any fresh non-sleep event.

### 4. Reaction BT tasks (GDScript)
`PlayZoomies`, `Startle`, `BusyIdle`, `EmitSay` — each small, each calls into
existing pet API (`drive_state()` for movement). New reaction anims
(thinking/working) resolve through the creature's `.tres` `aliases` map, with
`idle` as the fallback when a creature has no dedicated anim.

### 5. `EmitSay`
- **Does:** triggers the existing `/tick` path with the agent event included so
  the pet says something about the activity.
- **Layering:** daemon up → LLM `say`; daemon down → canned line from
  `config/comments.json` via the existing `_fallback_comment` path.
- **Reuse:** the `/tick` payload already carries `event`; add the agent token to
  it. No new daemon route.

## Data flow

1. You submit a prompt in Claude Code → `UserPromptSubmit` hook fires →
   `goob_hook.py` writes `{event:"thinking", ts:…}`.
2. `AgentEventPoller` sees the new `ts`, sets blackboard `agent_event="thinking"`.
3. HSM transitions `Ambient → Reacting`; the reaction BT falls through to
   `BusyIdle`; `EmitSay` optionally fires.
4. Tools run (`PreToolUse` → `working`), a subagent starts (`SubagentStart` →
   `subagent` → zoomies), the task ends (`Stop` → `done` → zoomies + "✅").
5. 8 s after the last event with nothing new → stale → HSM falls back to
   `Ambient` and the pet resumes its normal life.

## Error handling

- Hook script never blocks the agent: fast stdin read, atomic write, swallow all
  errors (a broken pet must never wedge Claude Code).
- Missing/torn file → poller reads nothing, blackboard unchanged.
- Daemon down → `EmitSay` uses canned lines; state reactions are unaffected.
- Agent killed mid-task → freshness timeout returns the pet to `Ambient`.

## Testing

One headless GDScript test (`just test` compatible): push a fake `agent_event`
onto the blackboard, tick the HSM, assert (a) it enters `Reacting`, (b) the
reaction BT selects the branch matching the token, (c) a stale timestamp returns
it to `Ambient`. No hook-process or daemon dependency in the test.

## Scope guards (ponytail)

**In:** Claude Code hooks only, one active session, the 6 events above, HSM+BT
for the reaction slice, `pet.gd` preserved under `Ambient`.

**Out (add when actually needed):** clawd's 18-agent registry, permission-
approval bubbles, multi-session tracking, per-agent process detection, migrating
the rest of `pet.gd` into the tree.

## Install note

LimboAI v1.8.0 GDExtension (`addons/limboai/`) is already installed and loads
clean on Godot 4.7. Decide whether to commit the native binary or `.gitignore`
it. CLAUDE.md's "Godot 4.3+" line is stale — the project runs 4.7; fix when docs
are next touched.
