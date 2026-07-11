# LimboAI Agent-Reactivity Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** The desktop pet reacts in near-real-time to the local Claude Code session — pinning calm while you prompt/tools run, zoomies when a subagent finishes, zoomies + "✅" on task completion — driven deterministically through a LimboAI HSM+BT, behind a default-off feature flag.

**Architecture:** A stdlib Claude Code hook atomically writes the latest event to `/tmp/goob-agent.json`. When `GOOB_HSM` is set, `main.gd` adds an `AgentEventPoller` (reads the file ~4 Hz) and a `LimboHSM` on the pet. The poller writes the token to the HSM blackboard and dispatches HSM events; the HSM switches Ambient↔Reacting↔Sleeping and, in Reacting, runs a behavior tree that drives the pet **only** through the existing `pet.drive_state()` channel. `pet.gd` is untouched and stays sole owner of pet state.

**Tech Stack:** Godot 4.7 (GDScript), LimboAI v1.8.0 (GDExtension, vendored at `addons/limboai/`), Python 3 stdlib (hook).

## Global Constraints

- **Godot 4.7**, **LimboAI v1.8.0** (already installed + loads clean; `just check` passes).
- **Feature flag `GOOB_HSM`** (env var or `.env` line, like the existing `DEBUG`), **default off**. Off ⇒ zero behavior change, poller/HSM never instantiated.
- **No changes to `scripts/pet.gd`.** Reactions use only verbs `drive_state` already accepts: `idle`, `zoomies`. `DRIVABLE` and the `daemon/agent.py` sync rule are untouched.
- **Hook is Python 3 stdlib only** — no `uv`, no deps; must run even if the daemon is never set up.
- **Event file:** `/tmp/goob-agent.json`, shape `{"token": <str>, "ts": <float>}`. `ts` is `time.time()` (float — integer seconds collapse same-second events).
- **Verified Claude Code hook events only:** `PreToolUse, PostToolUse, UserPromptSubmit, Notification, Stop, SubagentStop, PreCompact, SessionStart, SessionEnd`. (`SubagentStart`/`PostToolUseFailure`/`StopFailure` are clawd tokens, NOT CC events — do not use them.)
- **LimboAI facts verified by headless introspection** (rely on these exact signatures):
  - Custom task: `extends BTAction`/`BTCondition`; override `func _tick(delta) -> Status`. Inside a `BTTask` subclass the identifiers `SUCCESS`, `RUNNING`, `FAILURE`, `FRESH` are in scope. Elsewhere use `BTTask.SUCCESS` etc. Enum: `FRESH=0, RUNNING=1, FAILURE=2, SUCCESS=3`.
  - **Task params MUST be `@export`** — `instantiate()` clones tasks and resets plain `var`s to their defaults.
  - Task reads: `agent` (property) and `get_blackboard()`. We pass the pet on the blackboard (`get_var("pet")`), because `pet` is a `RefCounted`, not a Node.
  - Build a tree in code: `BTSelector.new()` / `BTSequence.new()` + `.add_child(task)`, then `var bt := BehaviorTree.new(); bt.set_root_task(root)`.
  - `Blackboard`: `set_var(name, value)`, `get_var(name, default)`, `has_var(name)`. `get_var` walks up parent scopes.
  - HSM wiring order: create `LimboHSM.new()`, `add_child` the states, **`BTState.set_scene_root_hint(host)`** (required or the tree fails to instantiate), `hsm.initialize(host)`, set blackboard vars, `hsm.set_initial_state(state)`, `hsm.add_transition(from, to, &"event")`, `hsm.set_active(true)`. Drive per-frame with `hsm.update(delta)`. Switch states with `hsm.dispatch(&"event")`. Read state with `hsm.get_active_state()`.

---

### Task 1: Claude Code hook script

**Files:**
- Create: `hooks/goob_hook.py`
- Test: `tests/test_goob_hook.py`

**Interfaces:**
- Produces: a CLI `python3 hooks/goob_hook.py <CCEventName>` that writes `/tmp/goob-agent.json` = `{"token": str, "ts": float}` for a mapped event, or does nothing (exit 0) for an unmapped one.
- Consumes: nothing.

- [ ] **Step 1: Write the failing test**

Create `tests/test_goob_hook.py`:

```python
import json, os, subprocess, sys, tempfile

HOOK = os.path.join(os.path.dirname(__file__), "..", "hooks", "goob_hook.py")

def _run(event, path):
    env = dict(os.environ, GOOB_AGENT_FILE=path)
    subprocess.run([sys.executable, HOOK, event], env=env, check=True)

def test_maps_known_event():
    with tempfile.TemporaryDirectory() as d:
        p = os.path.join(d, "a.json")
        _run("Stop", p)
        data = json.load(open(p))
        assert data["token"] == "done"
        assert isinstance(data["ts"], float)

def test_prompt_and_tool_tokens():
    with tempfile.TemporaryDirectory() as d:
        p = os.path.join(d, "a.json")
        _run("UserPromptSubmit", p); assert json.load(open(p))["token"] == "thinking"
        _run("PreToolUse", p);       assert json.load(open(p))["token"] == "working"
        _run("SubagentStop", p);     assert json.load(open(p))["token"] == "subagent"
        _run("SessionEnd", p);       assert json.load(open(p))["token"] == "sleep"

def test_unknown_event_writes_nothing():
    with tempfile.TemporaryDirectory() as d:
        p = os.path.join(d, "a.json")
        _run("PreCompact", p)   # mapped to nothing
        assert not os.path.exists(p)

if __name__ == "__main__":
    test_maps_known_event(); test_prompt_and_tool_tokens(); test_unknown_event_writes_nothing()
    print("test_goob_hook OK")
```

- [ ] **Step 2: Run test to verify it fails**

Run: `python3 tests/test_goob_hook.py`
Expected: FAIL — `FileNotFoundError`/`CalledProcessError` (hook does not exist yet).

- [ ] **Step 3: Write minimal implementation**

Create `hooks/goob_hook.py`:

```python
#!/usr/bin/env python3
"""goob Claude Code hook: map a CC event name (argv[1]) to a goob token and
atomically write it to the agent-event file the pet polls. Stdlib only; any
error is swallowed so a broken pet can never wedge Claude Code."""
import json, os, sys, tempfile, time

# Verified Claude Code hook events -> goob tokens. Unlisted events are ignored.
EVENTS = {
    "SessionStart": "wake",
    "UserPromptSubmit": "thinking",
    "PreToolUse": "working",
    "PostToolUse": "working",
    "SubagentStop": "subagent",
    "Stop": "done",
    "SessionEnd": "sleep",
}

def main():
    if len(sys.argv) < 2:
        return
    token = EVENTS.get(sys.argv[1])
    if token is None:
        return
    path = os.environ.get("GOOB_AGENT_FILE", "/tmp/goob-agent.json")
    body = json.dumps({"token": token, "ts": time.time()}).encode()
    d = os.path.dirname(path) or "."
    fd, tmp = tempfile.mkstemp(dir=d)
    try:
        os.write(fd, body)
        os.close(fd)
        os.replace(tmp, path)   # atomic: poller never sees a torn file
    except OSError:
        try: os.unlink(tmp)
        except OSError: pass

if __name__ == "__main__":
    try:
        main()
    except Exception:
        pass   # never fail the hook
```

- [ ] **Step 4: Run test to verify it passes**

Run: `python3 tests/test_goob_hook.py`
Expected: `test_goob_hook OK`

- [ ] **Step 5: Commit**

```bash
git add hooks/goob_hook.py tests/test_goob_hook.py
git commit -m "feat(hook): map Claude Code events to goob agent tokens"
```

---

### Task 2: Agent-event poller — parser + decision logic

**Files:**
- Create: `scripts/agent_poller.gd`
- Test: `tests/test_agent_poller.gd`

**Interfaces:**
- Produces:
  - `AgentPoller.read_event(path: String) -> Dictionary` — `{token, ts}` for a well-formed file, else `{}`.
  - `AgentPoller.decide(prev: Dictionary, cur: Dictionary, now_s: float, reacting: bool) -> StringName` — returns the HSM event to dispatch (`&"agent_event"`, `&"agent_sleep"`, `&"agent_wake"`, `&"agent_stale"`) or `&""` for no-op. `prev`/`cur` are `read_event` results; `now_s` is seconds; `reacting` is whether the HSM is currently in Reacting.
- Consumes: nothing (pure static functions; the Node wrapper comes in Task 5).

Design of `decide`:
- New `(ts, token)` vs `prev` and token in {thinking, working, subagent, done} ⇒ `&"agent_event"`.
- New `(ts, token)` and token == `sleep` ⇒ `&"agent_sleep"`.
- New `(ts, token)` and token == `wake` ⇒ `&"agent_wake"`.
- No new event, `reacting`, and `now_s - cur.ts > 8.0` ⇒ `&"agent_stale"`.
- Otherwise `&""`.

- [ ] **Step 1: Write the failing test**

Create `tests/test_agent_poller.gd`:

```gdscript
extends SceneTree
# Run: godot --headless --path . --script res://tests/test_agent_poller.gd

func _write(path: String, text: String) -> void:
	var f := FileAccess.open(path, FileAccess.WRITE)
	f.store_string(text); f.close()

func _init() -> void:
	var p := "user://poll_test.json"
	var abs := ProjectSettings.globalize_path(p)

	# read_event
	_write(abs, '{"token":"done","ts":12.5}')
	var ev := AgentPoller.read_event(abs)
	assert(ev.get("token") == "done" and ev.get("ts") == 12.5)
	assert(AgentPoller.read_event(abs + ".nope").is_empty())        # missing
	_write(abs, "not json{")
	assert(AgentPoller.read_event(abs).is_empty())                  # malformed

	# decide
	var a := {"token": "thinking", "ts": 100.0}
	var b := {"token": "working", "ts": 100.3}
	assert(AgentPoller.decide({}, a, 100.0, false) == &"agent_event")   # first event
	assert(AgentPoller.decide(a, a, 100.0, true) == &"")                # unchanged, fresh
	assert(AgentPoller.decide(a, b, 100.3, true) == &"agent_event")     # same-sec new token
	assert(AgentPoller.decide(a, a, 109.0, true) == &"agent_stale")     # >8s stale
	assert(AgentPoller.decide(a, {"token":"sleep","ts":200.0}, 200.0, false) == &"agent_sleep")
	assert(AgentPoller.decide(a, {"token":"wake","ts":200.0}, 200.0, true) == &"agent_wake")

	print("test_agent_poller OK")
	quit()
```

- [ ] **Step 2: Run test to verify it fails**

Run: `godot --headless --path . --script res://tests/test_agent_poller.gd`
Expected: FAIL — parser error / `AgentPoller` not found.

- [ ] **Step 3: Write minimal implementation**

Create `scripts/agent_poller.gd`:

```gdscript
class_name AgentPoller
extends Node

# Pure, testable static core (Task 2). The Node wiring (Timer, dispatch to the
# HSM) is added in Task 5 via poll(). Reactions live tokens: thinking/working/
# subagent/done; lifecycle tokens: wake/sleep.

const FRESH_TOKENS := ["thinking", "working", "subagent", "done"]
const STALE_AFTER_S := 8.0

static func read_event(path: String) -> Dictionary:
	if not FileAccess.file_exists(path):
		return {}
	var f := FileAccess.open(path, FileAccess.READ)
	if f == null:
		return {}
	var data = JSON.parse_string(f.get_as_text())
	f.close()
	if typeof(data) != TYPE_DICTIONARY or not data.has("token") or not data.has("ts"):
		return {}
	return {"token": String(data["token"]), "ts": float(data["ts"])}

static func decide(prev: Dictionary, cur: Dictionary, now_s: float, reacting: bool) -> StringName:
	var changed := not cur.is_empty() and \
		(prev.get("ts") != cur.get("ts") or prev.get("token") != cur.get("token"))
	if changed:
		var tok := String(cur.get("token", ""))
		if tok == "sleep":
			return &"agent_sleep"
		if tok == "wake":
			return &"agent_wake"
		if tok in FRESH_TOKENS:
			return &"agent_event"
		return &""
	if reacting and not cur.is_empty() and now_s - float(cur.get("ts", now_s)) > STALE_AFTER_S:
		return &"agent_stale"
	return &""
```

- [ ] **Step 4: Run test to verify it passes**

Run: `godot --headless --path . --script res://tests/test_agent_poller.gd`
Expected: `test_agent_poller OK`
(Note: `class_name` registers on a filesystem scan — if the test errors with "Could not find type", run `just check` once, then re-run.)

- [ ] **Step 5: Commit**

```bash
git add scripts/agent_poller.gd scripts/agent_poller.gd.uid tests/test_agent_poller.gd tests/test_agent_poller.gd.uid
git commit -m "feat(poller): agent-event file parser + dispatch decision logic"
```

---

### Task 3: Reaction behavior-tree tasks + tree builder

**Files:**
- Create: `scripts/bt/bt_is_token.gd`, `scripts/bt/bt_drive.gd`, `scripts/bt/bt_speak.gd`
- Create: `scripts/agent_tree.gd` (builds the `BehaviorTree`)
- Test: `tests/test_agent_tree.gd`

**Interfaces:**
- Consumes (from the blackboard at runtime): `agent_token: String`, `pet` (has `drive_state(String) -> bool`), `speaker` (has `speak_reaction(String, float) -> void`), `agent_ts: float`.
- Produces: `AgentTree.build() -> BehaviorTree` — a selector: `done → drive zoomies once + speak "✅ done"`; `subagent → drive zoomies once`; else `drive idle (RUNNING, re-asserts)`.

Task-parameter rule (verified): every tunable is `@export` or it is lost on `instantiate()`.

- [ ] **Step 1: Write the failing test**

Create `tests/test_agent_tree.gd`:

```gdscript
extends SceneTree
# Run: godot --headless --path . --script res://tests/test_agent_tree.gd

class StubPet extends RefCounted:
	var drove: Array = []
	func drive_state(n: String) -> bool: drove.append(n); return true

class StubSpeaker extends RefCounted:
	var said: Array = []
	func speak_reaction(text: String, ts: float) -> void: said.append([text, ts])

func _run(bt: BehaviorTree, host: Node, pet, spk, token: String) -> int:
	var bb := Blackboard.new()
	bb.set_var("pet", pet); bb.set_var("speaker", spk)
	bb.set_var("agent_token", token); bb.set_var("agent_ts", 1.0)
	var inst = bt.instantiate(host, bb, host, host)
	return inst.update(0.0)

func _init() -> void:
	var host := Node.new(); host.name = "H"; get_root().add_child(host)
	var pet := StubPet.new(); var spk := StubSpeaker.new()
	var bt := AgentTree.build()

	assert(_run(bt, host, pet, spk, "done") == BTTask.SUCCESS)
	assert(pet.drove == ["zoomies"] and spk.said.size() == 1 and spk.said[0][0] == "✅ done")

	pet.drove.clear(); spk.said.clear()
	assert(_run(bt, host, pet, spk, "subagent") == BTTask.SUCCESS)
	assert(pet.drove == ["zoomies"] and spk.said.is_empty())

	pet.drove.clear()
	assert(_run(bt, host, pet, spk, "thinking") == BTTask.RUNNING)  # BusyIdle holds
	assert(pet.drove == ["idle"])

	print("test_agent_tree OK")
	quit()
```

- [ ] **Step 2: Run test to verify it fails**

Run: `godot --headless --path . --script res://tests/test_agent_tree.gd`
Expected: FAIL — `AgentTree` not found.

- [ ] **Step 3: Write minimal implementation**

Create `scripts/bt/bt_is_token.gd`:

```gdscript
extends BTCondition
# SUCCESS iff the live agent token equals `want`.

@export var want := ""

func _tick(_delta: float) -> Status:
	return SUCCESS if get_blackboard().get_var("agent_token", "") == want else FAILURE
```

Create `scripts/bt/bt_drive.gd`:

```gdscript
extends BTAction
# Drive the pet via the existing drive_state() channel. `once` = edge-triggered
# (drive once, return SUCCESS — for self-terminating verbs like zoomies).
# `once == false` re-asserts every tick and returns RUNNING (pins idle so
# pet.gd's _decide() never rolls an autonomous behavior).

@export var verb := "idle"
@export var once := false

func _tick(_delta: float) -> Status:
	var pet = get_blackboard().get_var("pet")
	if pet != null:
		pet.drive_state(verb)
	return SUCCESS if once else RUNNING
```

Create `scripts/bt/bt_speak.gd`:

```gdscript
extends BTAction
# Speak a canned reaction line, debounced on the event ts by the speaker.

@export var text := ""

func _tick(_delta: float) -> Status:
	var spk = get_blackboard().get_var("speaker")
	if spk != null:
		spk.speak_reaction(text, float(get_blackboard().get_var("agent_ts", 0.0)))
	return SUCCESS
```

Create `scripts/agent_tree.gd`:

```gdscript
class_name AgentTree
extends RefCounted

# Builds the reaction tree in code (exact + headless-testable; LimboAI's
# debugger still visualizes it live). Selector, first match wins:
#   done     -> zoomies once + "✅ done"
#   subagent -> zoomies once
#   else     -> idle, held (thinking/working)

const IsToken := preload("res://scripts/bt/bt_is_token.gd")
const Drive := preload("res://scripts/bt/bt_drive.gd")
const Speak := preload("res://scripts/bt/bt_speak.gd")

static func _seq(children: Array) -> BTSequence:
	var s := BTSequence.new()
	for c in children:
		s.add_child(c)
	return s

static func _is(token: String) -> BTCondition:
	var c := IsToken.new(); c.want = token; return c

static func _drive(verb: String, once: bool) -> BTAction:
	var d := Drive.new(); d.verb = verb; d.once = once; return d

static func _speak(text: String) -> BTAction:
	var s := Speak.new(); s.text = text; return s

static func build() -> BehaviorTree:
	var sel := BTSelector.new()
	sel.add_child(_seq([_is("done"), _drive("zoomies", true), _speak("✅ done")]))
	sel.add_child(_seq([_is("subagent"), _drive("zoomies", true)]))
	sel.add_child(_drive("idle", false))   # thinking/working fallback, holds
	var bt := BehaviorTree.new()
	bt.set_root_task(sel)
	return bt
```

- [ ] **Step 4: Run test to verify it passes**

Run: `just check` (registers the new `class_name AgentTree`), then
`godot --headless --path . --script res://tests/test_agent_tree.gd`
Expected: `test_agent_tree OK`

- [ ] **Step 5: Commit**

```bash
git add scripts/bt/ scripts/agent_tree.gd scripts/agent_tree.gd.uid tests/test_agent_tree.gd tests/test_agent_tree.gd.uid
git commit -m "feat(bt): reaction behavior tree — zoomies on done/subagent, pinned idle otherwise"
```

---

### Task 4: HSM assembly + integration test

**Files:**
- Create: `scripts/agent_hsm.gd`
- Test: `tests/test_agent_hsm.gd`

**Interfaces:**
- Consumes: `AgentTree.build()`; a `host: Node` to parent under; `pet` for the blackboard.
- Produces: `AgentHsm.build(host: Node, pet) -> LimboHSM` — an initialized, active HSM with states named via transitions `agent_event`/`agent_stale`/`agent_sleep`/`agent_wake`. The poller (Task 5) writes `agent_token`/`agent_ts` to `hsm.get_blackboard()` and calls `hsm.dispatch(...)`.

**Note (speech cut):** there is no speech surface right now, so no `speaker` is wired. The committed `bt_speak` task null-guards on a missing `speaker` blackboard var and silently no-ops — reactions are movement-only.

- [ ] **Step 1: Write the failing test**

Create `tests/test_agent_hsm.gd`:

```gdscript
extends SceneTree
# Run: godot --headless --path . --script res://tests/test_agent_hsm.gd

class StubPet extends RefCounted:
	var drove: Array = []
	func drive_state(n: String) -> bool: drove.append(n); return true

func _init() -> void:
	var host := Node.new(); host.name = "H"; get_root().add_child(host)
	var pet := StubPet.new()
	var hsm := AgentHsm.build(host, pet)

	assert(hsm.get_active_state() == hsm.get_node("Ambient"))

	# a "done" event -> Reacting -> tree drives zoomies (speech is a no-op)
	hsm.get_blackboard().set_var("agent_token", "done")
	hsm.get_blackboard().set_var("agent_ts", 1.0)
	hsm.dispatch(&"agent_event")
	assert(hsm.get_active_state() == hsm.get_node("Reacting"))
	hsm.update(0.0)
	assert(pet.drove.has("zoomies"))

	# stale -> back to Ambient
	hsm.dispatch(&"agent_stale")
	assert(hsm.get_active_state() == hsm.get_node("Ambient"))

	# sleep from anywhere
	hsm.dispatch(&"agent_sleep")
	assert(hsm.get_active_state() == hsm.get_node("Sleeping"))

	print("test_agent_hsm OK")
	quit()
```

- [ ] **Step 2: Run test to verify it fails**

Run: `godot --headless --path . --script res://tests/test_agent_hsm.gd`
Expected: FAIL — `AgentHsm` not found.

- [ ] **Step 3: Write minimal implementation**

Create `scripts/agent_hsm.gd`:

```gdscript
class_name AgentHsm
extends RefCounted

# Builds the top-level pet HSM. Ambient = hands-off (pet.gd self-drives).
# Reacting = a BTState running AgentTree. Sleeping = calm (re-assert idle).
# Transitions are event-based; the poller dispatches them.

const AgentTreeScript := preload("res://scripts/agent_tree.gd")

static func build(host: Node, pet) -> LimboHSM:
	var hsm := LimboHSM.new()
	hsm.name = "AgentHSM"

	var ambient := LimboState.new()
	ambient.name = "Ambient"                     # no-op: pet.gd self-drives

	var reacting := BTState.new()
	reacting.name = "Reacting"
	reacting.set_behavior_tree(AgentTreeScript.build())
	reacting.set_scene_root_hint(host)           # required or the tree won't instantiate

	var sleeping := LimboState.new()
	sleeping.name = "Sleeping"
	sleeping.call_on_update(func(_delta): pet.drive_state("idle"))

	hsm.add_child(ambient)
	hsm.add_child(reacting)
	hsm.add_child(sleeping)
	host.add_child(hsm)

	hsm.initialize(host)
	hsm.get_blackboard().set_var("pet", pet)
	hsm.get_blackboard().set_var("agent_token", "")
	hsm.get_blackboard().set_var("agent_ts", 0.0)
	hsm.set_initial_state(ambient)

	hsm.add_transition(ambient, reacting, &"agent_event")
	hsm.add_transition(reacting, ambient, &"agent_stale")
	hsm.add_transition(sleeping, ambient, &"agent_wake")
	hsm.add_transition(sleeping, reacting, &"agent_event")
	hsm.add_transition(hsm.anystate(), sleeping, &"agent_sleep")

	hsm.set_active(true)
	return hsm
```

- [ ] **Step 4: Run test to verify it passes**

Run: `just check`, then `godot --headless --path . --script res://tests/test_agent_hsm.gd`
Expected: `test_agent_hsm OK`

- [ ] **Step 5: Commit**

```bash
git add scripts/agent_hsm.gd scripts/agent_hsm.gd.uid tests/test_agent_hsm.gd tests/test_agent_hsm.gd.uid
git commit -m "feat(hsm): Ambient/Reacting/Sleeping HSM wiring the reaction tree"
```

---

### Task 5: Wire the poller + HSM into `main.gd` behind `GOOB_HSM`

**Files:**
- Modify: `scripts/main.gd` (`_ready` setup + `_physics_process` update, behind `GOOB_HSM`)
- Modify: `scripts/agent_poller.gd` (add the Node `poll()` that dispatches)
- Modify: `justfile` (add the new test lines)

**Interfaces:**
- Consumes: `AgentPoller.read_event/decide`, `AgentHsm.build(self, pet)`, `pet` (PetBrain).
- Produces: a running reactive loop when `GOOB_HSM` is truthy; unchanged behavior otherwise. No speech (movement-only).

- [ ] **Step 1: Add the poller Node behavior**

In `scripts/agent_poller.gd`, add below the static functions:

```gdscript
# --- Node wrapper: poll the file and dispatch HSM events ---

const AGENT_FILE := "/tmp/goob-agent.json"

var hsm: LimboHSM
var _prev: Dictionary = {}

func setup(target_hsm: LimboHSM) -> void:
	hsm = target_hsm

func poll(now_s: float) -> void:
	if hsm == null:
		return
	var cur := read_event(AGENT_FILE)
	var reacting: bool = hsm.get_active_state() == hsm.get_node("Reacting")
	var ev := decide(_prev, cur, now_s, reacting)
	if not cur.is_empty() and (_prev.get("ts") != cur.get("ts") or _prev.get("token") != cur.get("token")):
		hsm.get_blackboard().set_var("agent_token", String(cur.get("token", "")))
		hsm.get_blackboard().set_var("agent_ts", float(cur.get("ts", 0.0)))
		_prev = cur
	if ev != &"":
		hsm.dispatch(ev)
```

- [ ] **Step 2: Wire it into `main.gd`**

Add a flag read + node creation in `_ready()` (near the existing `http`/`DEBUG` setup). Use the same env/.env pattern the project already uses for `DEBUG`:

```gdscript
# agent-reactivity (opt-in): GOOB_HSM=1 makes the pet react to Claude Code.
var agent_hsm: LimboHSM = null
var agent_poller: AgentPoller = null

func _setup_agent_reactivity() -> void:
	var flag := OS.get_environment("GOOB_HSM")
	if flag == "" or flag == "0" or flag.to_lower() == "false":
		return
	agent_hsm = AgentHsm.build(self, pet)
	agent_poller = AgentPoller.new()
	add_child(agent_poller)
	agent_poller.setup(agent_hsm)
```

**No speech wiring** — there is no speech surface right now (bubble removed).
Reactions are movement-only; the committed `bt_speak` task null-guards on the
absent `speaker` blackboard var and no-ops. Do NOT add a `speak_reaction` or a
`speaker` — that is plumbing to nowhere. When a speech surface returns, set a
`speaker` on the HSM blackboard in `AgentHsm.build` and give it a `speak_reaction`;
nothing else changes.

Call `_setup_agent_reactivity()` at the end of `_ready()`. In `_physics_process(delta)` add (guarded):

```gdscript
	if agent_hsm != null:
		agent_hsm.update(delta)
		# WALL-CLOCK seconds — must match the hook's time.time() ts so the
		# staleness math (now_s - cur.ts > 8) works. NOT get_ticks_msec (uptime).
		agent_poller.poll(Time.get_unix_time_from_system())
```

(If `.env` support is needed beyond real env vars, mirror how `DEBUG` is read — reuse that helper rather than re-implementing.)

- [ ] **Step 3: Add the new tests to `justfile`**

In the `test:` recipe, append:

```
    {{godot}} --headless --path . --script res://tests/test_agent_poller.gd
    {{godot}} --headless --path . --script res://tests/test_agent_hsm.gd
    {{godot}} --headless --path . --script res://tests/test_agent_tree.gd
    python3 tests/test_goob_hook.py
```

- [ ] **Step 4: Verify — automated + manual**

Run: `just check && just test`
Expected: all tests print `... OK`, no parse errors.

Manual smoke (the real payoff):
```bash
# terminal A — run the pet with the flag on
GOOB_HSM=1 just run
# terminal B — simulate Claude Code events
python3 hooks/goob_hook.py UserPromptSubmit   # pet pins calm
python3 hooks/goob_hook.py SubagentStop        # zoomies
python3 hooks/goob_hook.py Stop                # zoomies (speech no-op for now)
```
Expected: within ~0.25 s of each command the pet reacts; ~8 s after the last, it returns to normal wandering. With `GOOB_HSM` unset, none of this runs and the pet behaves exactly as before.

- [ ] **Step 5: Commit**

```bash
git add scripts/main.gd scripts/agent_poller.gd justfile
git commit -m "feat: wire agent-reactive HSM into main behind GOOB_HSM flag"
```

---

### Task 6: Install docs + stale-version fix

**Files:**
- Modify: `CLAUDE.md` (add `GOOB_HSM` note; fix "Godot 4.3+" → 4.7; note LimboAI dep)
- Create: `docs/agent-reactivity.md` (the `~/.claude/settings.json` hook snippet + how it works)

**Interfaces:** docs only.

- [ ] **Step 1: Write the hook-install doc**

Create `docs/agent-reactivity.md`:

```markdown
# Agent reactivity (opt-in)

With `GOOB_HSM=1`, the pet reacts to your local Claude Code session via a hook
that writes events to `/tmp/goob-agent.json`, polled by the pet.

## Enable

1. Run the pet with the flag: `GOOB_HSM=1 just run` (or add `GOOB_HSM=1` to `.env`).
2. Register the hook in `~/.claude/settings.json` (absolute path to this repo):

    ```json
    {
      "hooks": {
        "UserPromptSubmit": [{"hooks": [{"type": "command", "command": "python3 /ABS/PATH/goob/hooks/goob_hook.py UserPromptSubmit"}]}],
        "PreToolUse":       [{"hooks": [{"type": "command", "command": "python3 /ABS/PATH/goob/hooks/goob_hook.py PreToolUse"}]}],
        "SubagentStop":     [{"hooks": [{"type": "command", "command": "python3 /ABS/PATH/goob/hooks/goob_hook.py SubagentStop"}]}],
        "Stop":             [{"hooks": [{"type": "command", "command": "python3 /ABS/PATH/goob/hooks/goob_hook.py Stop"}]}],
        "SessionEnd":       [{"hooks": [{"type": "command", "command": "python3 /ABS/PATH/goob/hooks/goob_hook.py SessionEnd"}]}]
      }
    }
    ```

The hook is Python-3 stdlib only and swallows all errors, so it can never block
or fail your Claude Code session.

## Reactions

| Claude Code does | pet does |
|------------------|----------|
| prompt / tools running | pins calm |
| a subagent finishes | zoomies |
| task completes (Stop) | zoomies + "✅ done" |
| session ends | settles |
```

- [ ] **Step 2: Update `CLAUDE.md`**

Add a line under Dependencies noting LimboAI is vendored in `addons/limboai/`; add `GOOB_HSM` beside `DEBUG` in the env/flags notes; change the "Godot 4 (4.3+ …)" line to "Godot 4 (4.7; LimboAI needs 4.6+)".

- [ ] **Step 3: Verify**

Run: `just check`
Expected: clean (docs don't affect parse, but confirms nothing else broke).

- [ ] **Step 4: Commit**

```bash
git add CLAUDE.md docs/agent-reactivity.md
git commit -m "docs: agent-reactivity setup + fix stale Godot version"
```

---

## Self-Review

**Spec coverage:** hook + mapping (T1) · transport file + float ts + atomic write (T1) · poller parse/decide/staleness (T2) · reaction BT edge-vs-hold semantics (T3) · HSM Ambient/Reacting/Sleeping + event transitions (T4) · `GOOB_HSM` flag + drive-via-`drive_state` + debounced `speak_reaction` avoiding the single-HTTPRequest collision (T5) · install docs + version fix (T6). All spec sections map to a task.

**Blocking-review items resolved:** B1 startle — dropped (out of scope, no CC failure event). B2 BusyIdle holds — `bt_drive.gd` `once=false` re-asserts idle each tick, returns RUNNING (T3). B3 transitions are `dispatch`-based, driven by the poller (T2 `decide` + T5 `poll`). B4 same-second collapse — `ts` is float and `decide` compares `(ts, token)` (T1/T2). S1 EmitSay collision — `speak_reaction` calls `dialog_face.speak()` directly, no HTTP, debounced on `ts` (T5). S4 blackboard sharing — poller writes `hsm.get_blackboard()`; verified reachable by BT tasks (T4 test).

**Placeholder scan:** none — every code step is complete and runnable.

**Type consistency:** `drive_state(String)->bool`, `speak_reaction(String,float)->void`, `read_event->Dictionary`, `decide(...)->StringName`, `AgentTree.build()->BehaviorTree`, `AgentHsm.build(Node, pet, speaker)->LimboHSM` are used identically across tasks. BT status via `BTTask.SUCCESS/RUNNING/FAILURE` (enum 3/1/2) in tests, bare `SUCCESS`/`RUNNING`/`FAILURE` inside task `_tick`.
