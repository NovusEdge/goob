# goob — agent-reactive behavior via LimboAI

**Date:** 2026-07-10
**Status:** approved design + Opus review folded in, pre-implementation

## Goal

Make the pet react, in near-real-time, to what your local Claude Code session
is doing — attentive while you prompt and tools run, zoomies when a subagent
finishes, a celebratory zoomies + "✅" when the task completes, calm when the
session ends. Idea borrowed from `clawd-on-desk`; the reaction is deterministic
and works with zero daemon.

Also the first use of **LimboAI** in goob: the pet's top-level behavior becomes
a hierarchical state machine (HSM), and the agent-reaction dispatch becomes a
behavior tree (BT). We learn HSM+BT on one real vertical slice.

## Feature flag & arbitration

Gated behind **`GOOB_HSM`** (env var or `.env` line, mirroring `DEBUG`), default
off:

- **off** — today's behavior exactly. Poller + HSM never instantiated.
- **on** — `main.gd` adds the `AgentEventPoller` + `LimboHSM` nodes.

**No two-FSM conflict.** `pet.gd` stays the sole owner of `pet.state`, animation,
and position. The HSM drives the pet only through the *existing*
`pet.drive_state(name)` channel — the same entry point the LLM daemon already
uses. `pet.gd` already tolerates external drives. Ambient = HSM hands-off;
Reacting/Sleeping = HSM calls `drive_state(...)`.

**No `pet.gd` changes needed.** Every MVP reaction maps to a verb `drive_state`
already accepts (`idle`, `zoomies`). Startle/error would need `scare()` (a
different channel) and a reliable failure event — neither is in scope (below).

## Architecture

Two orthogonal concerns:

- **Transport** — a Claude Code hook writes the latest event to one small JSON
  file; the pet polls it. No server, no ports; the hook writes whether or not
  the pet/daemon are running.
- **Behavior** — a `LimboHSM` owns top-level modes; a BT dispatches the reaction.

```
Claude Code hook  ──argv event name──▶  hooks/goob_hook.py (stdlib)
                                             │ atomic write
                                             ▼
                                   /tmp/goob-agent.json  {token, ts}
                                             │ AgentEventPoller reads ~4 Hz
                                             ▼
   main.gd (GOOB_HSM on):
     poller sets blackboard token + ts, then hsm.dispatch(<event>)
                                             │
   LimboHSM (on the pet)
   ├── Ambient    ← hands-off; pet.gd self-drives (unchanged)
   ├── Reacting   ← BTState runs the reaction tree
   └── Sleeping   ← calm (drive idle)
```

## Verified event mapping (Claude Code hook → goob token)

Standard CC hook events only (verified against the installed set:
`PreToolUse, PostToolUse, UserPromptSubmit, Notification, Stop, SubagentStop,
PreCompact, SessionStart, SessionEnd`). Unknown/unmapped events → no file write.

| CC hook          | token    | HSM dispatch  | Reaction                    |
|------------------|----------|---------------|-----------------------------|
| SessionStart     | wake     | `agent_wake`  | → Ambient                   |
| UserPromptSubmit | thinking | `agent_event` | BusyIdle (pin idle)         |
| PreToolUse       | working  | `agent_event` | BusyIdle (pin idle)         |
| PostToolUse      | working  | `agent_event` | BusyIdle (pin idle)         |
| SubagentStop     | subagent | `agent_event` | Zoomies (one-shot)          |
| Stop             | done     | `agent_event` | Zoomies + speak "✅" (one-shot)|
| SessionEnd       | sleep    | `agent_sleep` | → Sleeping (calm)           |

**Out of scope (no standard hook / needs pet.gd surface):** error/startle (no
CC failure event — would require parsing the `PostToolUse` payload), permission
bubbles (Notification), PreCompact. Add later.

## Components & interfaces

### 1. `hooks/goob_hook.py`
- **Does:** takes the CC event name as `argv[1]`, maps it to a goob token via a
  dict, atomically writes `/tmp/goob-agent.json` = `{"token": <str>, "ts": <float>}`.
  Unknown event → exit 0, no write. Any error is swallowed (never wedge CC).
- **`ts` is `time.time()` (float)** — integer seconds would collapse two events
  in the same second (UserPromptSubmit then PreToolUse) into one.
- **Atomic:** write temp in the same dir + `os.replace`, so the poller never
  reads a torn file.
- **Depends on:** Python 3 stdlib only. Runs even if the daemon is never set up.
- **Installed via** a documented `~/.claude/settings.json` hooks snippet (manual;
  not auto-written).

### 2. `AgentEventPoller` — `scripts/agent_poller.gd` (Node)
- **Does:** on a 0.25 s `Timer`, reads `/tmp/goob-agent.json`. Tracks the last
  seen `(ts, token)`. On a **new** `(ts, token)` pair: writes `agent_token` +
  `agent_ts` to the shared blackboard, then `hsm.dispatch(<event for token>)`.
- **Freshness:** each tick, if the last event is older than 8 s and the HSM is in
  Reacting, dispatch `agent_stale` (→ Ambient). Keeps the pet from freezing mid
  "working" if the agent is killed.
- **Blackboard/HSM refs:** handed `hsm` (and `hsm.get_blackboard()`) at
  construction by `main.gd`. Writes the HSM root scope so both the guards and the
  BT conditions resolve the same vars (`get_var` walks up parent scopes).
- **Pure parser** `AgentPoller.read_event(path) -> Dictionary` (static): returns
  `{token, ts}` or `{}` — unit-testable without a scene tree.
- **Threading:** main-thread `FileAccess` read of a tiny atomically-replaced file
  is negligible and correct (scene tree is single-threaded).

### 3. HSM — `scripts/agent_hsm.gd`
- **Builds** a `LimboHSM` with three vanilla `LimboState`s (delegation via
  `call_on_enter`/`call_on_update`, no subclasses):
  - `Ambient` — no-op (pet.gd self-drives).
  - `Reacting` — a `BTState` (below).
  - `Sleeping` — `call_on_update` re-asserts `drive_state("idle")` (calm).
- **Transitions** (event-based; LimboHSM fires on `dispatch`, not polled guards):
  - `add_transition(Ambient, Reacting, &"agent_event")`
  - `add_transition(Reacting, Ambient, &"agent_stale")`
  - `add_transition(hsm.anystate(), Sleeping, &"agent_sleep")`
  - `add_transition(Sleeping, Ambient, &"agent_wake")` and
    `add_transition(Sleeping, Reacting, &"agent_event")`
- **Init:** `hsm.initialize(pet_agent_node)`, `hsm.set_active(true)`; `main.gd`
  calls `hsm.update(delta)` from `_physics_process` (or set update mode).

### 4. Reaction BT — `scripts/bt/` custom `BTAction` tasks
Built in code (`BehaviorTree.new().set_root_task(...)`), so it's exact and
headless-testable; LimboAI's debugger still visualizes it live.

```
BTSelector
├── BTSequence [IsToken("done")]     → Zoomies, Speak("✅ done")
├── BTSequence [IsToken("subagent")] → Zoomies
└── BusyIdle                          (thinking / working; the fallback)
```

- **`IsToken`** (`BTCondition`): `SUCCESS` iff `blackboard.get_var("agent_token")`
  equals the exported `token` param, else `FAILURE`.
- **`Zoomies`** (`BTAction`): **edge-triggered** — `_tick` calls
  `agent.drive_state("zoomies")` **once** and returns `SUCCESS`. Never re-assert:
  re-calling `_start_zoomies` each frame re-randomizes the dart target = jitter
  (harmless today only because `_interruptible()` blocks re-entry — do not rely
  on that). `zoomies` self-terminates (~10 s) on its own.
- **`BusyIdle`** (`BTAction`): re-asserts `agent.drive_state("idle")` every
  `_tick` and returns `RUNNING`. This holds — resetting `timer` each frame stops
  pet.gd's `_decide()` from rolling an autonomous behavior. A single edge call
  would flicker back to wandering within ~1 s.
- **`Speak`** (`BTAction`): calls the debounced `main.speak_reaction(text, ts)`
  once per event `ts`, returns `SUCCESS`.

### 5. `Speak` / `main.speak_reaction(text, ts)`
- **Does:** if `ts` differs from the last spoken `ts`, call
  `dialog_face.speak(text)` directly — **no HTTP**. Race-free.
- **Why not `/tick`:** `main.gd` has one `HTTPRequest` guarded by `in_flight`; an
  out-of-band tick returns `ERR_BUSY` and the reaction BT re-runs every frame
  while the event is fresh, so a raw tick would spam. Direct `speak()` sidesteps
  both. LLM-narrated agent events are a **future** layer (piggyback the token on
  the next scheduled `_maybe_comment`, keyed on `ts`).

## Data flow

1. You prompt → `UserPromptSubmit` hook → `goob_hook.py` writes `{token:"thinking", ts}`.
2. Poller sees new `(ts, token)` → sets blackboard, `hsm.dispatch(&"agent_event")`.
3. HSM `Ambient → Reacting`; BT falls through to `BusyIdle` → pet pins calm.
4. Tools run (`working`, same). Subagent finishes (`subagent` → zoomies once).
   Task ends (`Stop` → `done` → zoomies + "✅").
5. 8 s with no new event → poller dispatches `agent_stale` → `Reacting → Ambient`;
   pet resumes its normal life.
6. `SessionEnd` → `agent_sleep` → `Sleeping` (calm) until a new event wakes it.

## Error handling

- Hook never blocks CC: fast, atomic write, swallow all errors.
- Missing/torn file → poller reads nothing (`{}`), no dispatch.
- Daemon irrelevant — reactions and the emoji beat are daemon-free.
- Agent killed mid-task → 8 s freshness returns the pet to Ambient.
- Reactions are **best-effort**: `drive_state` no-ops when the pet is mid
  `follow`/`play`/`zoomies`/`retreat` (`_interruptible()`), so an occasional
  `done` beat can be missed. Acceptable for a pet; documented, not fixed.

## Testing

Two headless tests (`just test` style — `extends SceneTree`, `assert`, `quit()`):

- `tests/test_agent_poller.gd` — `AgentPoller.read_event` on a temp file:
  well-formed → `{token, ts}`; missing file → `{}`; malformed JSON → `{}`.
- `tests/test_agent_hsm.gd` — build the HSM with a stub agent exposing
  `drive_state(name)` (records calls); dispatch `agent_event` with
  `agent_token="done"` → asserts active state is `Reacting`, the BT drove
  `zoomies`, and `speak` fired once; dispatch `agent_stale` → back to `Ambient`.

Neither test touches the hook process or the daemon.

## Scope guards (ponytail)

**In:** feature-flagged (`GOOB_HSM`, default off), Claude Code hooks only, one
active session, the 6 verified tokens, HSM+BT for the reaction slice, `pet.gd`
untouched (driven only via `drive_state`).

**Out (add when needed):** error/startle beat, permission bubbles, PreCompact,
LLM-narrated agent events, clawd's 18-agent registry, multi-session tracking,
migrating the rest of `pet.gd` into the tree, a dedicated sleep/think anim clip.

## Notes

- LimboAI v1.8.0 GDExtension is committed at `addons/limboai/` and loads clean on
  Godot 4.7 (`just check` passes). Bundled `README.md` lists 4.6 / up to 1.7.x —
  the v1.8.0 release adds 4.7 support; verified by a clean headless load.
- CLAUDE.md's "Godot 4.3+" line is stale — the project runs 4.7; fix when docs
  are next touched.
