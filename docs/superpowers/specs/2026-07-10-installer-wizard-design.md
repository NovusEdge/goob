# goob — setup wizard (`installer/`, Go + Bubbletea)

**Date:** 2026-07-10
**Status:** approved design + Opus review folded in, pre-implementation

## Goal

A single `installer/` binary that walks a user through setting up goob:
detect dependencies, register goob's reaction hook into one or more AI coding
agents (not just Claude Code), and scaffold `.env`. Run via `just install`.

It replaces the fiddly manual steps in `docs/agent-reactivity.md` (hand-editing
`~/.claude/settings.json`) and generalizes them across coding agents through an
extensible **agent registry**.

## Non-goals (ponytail)

- **No system-level installs.** The wizard never runs a package manager or
  downloads binaries. It detects and prints copy-paste commands; the user runs
  them. Cross-distro, no sudo, no trust surface.
- **No network validation** of API keys (no test call). Write-only `.env`.
- **No 18-agent registry.** Ship 3 agents; the registry makes adding more a
  data + one-format-handler exercise, done when actually needed.
- Not a package/release pipeline, not an auto-updater.

## Architecture

A standalone Go module `installer/` (its own `go.mod`, module `goob/installer`),
reusing the charm stack already pinned in `tui/` (Bubbletea 1.3.10, bubbles,
lipgloss). `just install` runs `cd installer && go run .`.

A **linear Bubbletea wizard** — a parent model steps through child step-models:

```
Welcome → Doctor → Agent hooks → Environment → Review & Apply → Done
```

**Plan-then-apply.** No step writes to disk. Each step contributes to a shared
`Plan` — an ordered list of pending mutations. Only the Review & Apply step,
after an explicit confirm, executes the plan.

**Mutations carry a transform, not precomputed bytes.** A running agent (Claude,
Cursor) may rewrite its own config between plan-build and apply; if we stored the
new bytes computed at the Agents step, Apply would clobber the agent's concurrent
edit. Instead each `WriteFile` holds a pure `transform func(current []byte) []byte`
(e.g. `Handler.Install` bound to its agent + hookCmd). `Apply` **re-reads the file,
applies the transform to the fresh bytes, backs up, writes.** The Review screen
renders a diff computed at review time; if the fresh read at Apply differs from
what was previewed, warn before writing.

```go
type Plan struct { Steps []Mutation }
type Mutation interface {
    Describe() string          // one-line summary for the review screen
    Preview() (before, after []byte, err error)  // read current, apply transform — for the diff
    Apply() error              // re-read fresh, transform, backup-then-write
}
// WriteFile{path, backupPath, transform func([]byte) ([]byte, error)}
```

### File structure
```
installer/
  go.mod, go.sum
  main.go            — tea.NewProgram(rootModel)
  wizard.go          — root model: step sequencing, shared Plan + Context
  step_welcome.go
  step_doctor.go     — dependency detection + OS-specific install hints
  step_agents.go     — multi-select agents, build hook mutations
  step_env.go        — .env fields, build .env mutation
  step_apply.go      — render Plan, confirm, execute
  doctor.go          — dep probes (exec.LookPath, --version parse), OS hints
  registry.go        — Agent registry + FormatHandler interface
  agents.go          — the shipped agent data (claude, cursor, gemini)
  formats.go         — FormatHandler impls (settings-json shared by claude/gemini, cursor)
  testdata/          — real captured configs per agent (golden fixtures)
  envfile.go         — parse/merge/serialize .env
  plan.go            — Mutation + WriteFile (backup-then-write)
  *_test.go          — table-driven tests for registry/formats/envfile
```

## Step 1 — Doctor (deps: detect + copy-paste)

Probes, each `{name, found bool, version string, ok bool, hint string}`:

| dep | probe | needed for | min |
|-----|-------|-----------|-----|
| `godot` | `LookPath("godot")` or `$GODOT`; parse `--version` | the pet | **4.7** (project targets it; LimboAI needs ≥4.6) |
| `python3` | `LookPath("python3")` (then `python`) | **agent hooks** (required) | 3.x |
| `uv` | `LookPath("uv")` | daemon (optional) | any |
| `go` | `LookPath("go")` | TUI (optional) | any |

**`python3` is required, not optional** — the hook is `#!/usr/bin/env python3`
and the generated agent command hardcodes an interpreter. If absent, hooks
silently no-op (the hook swallows all errors), so this is the single most likely
invisible first-run failure. Doctor resolves the interpreter once (`python3`,
falling back to `python`) and the agent-hooks step uses whatever exists.

Renders a green/red checklist. For a missing/too-old dep, shows an
**OS-specific copy-paste command** — never executed. OS detection: `runtime.GOOS`
(darwin → `brew`), and on Linux read `/etc/os-release` `ID`/`ID_LIKE` →
`pacman -S` / `apt install` / `dnf install`; unknown distro → the upstream URL
(e.g. `https://docs.astral.sh/uv/`). **godot** has no universal package name and
often lives off-PATH (a build under `~/Downloads` that moves) → the "missing"
hint tells the user to **set `GODOT=/path/to/godot`** (which the probe honors),
not just a download link. Note: LimboAI is a vendored *addon* (`addons/limboai/`),
so a stock Godot 4.7 build suffices — no custom engine.

Doctor contributes **no** mutations — it's read-only advisory. optional deps
being absent is a warning, not a blocker.

## Step 2 — Agent hooks (the registry)

The core value. An agent is data + a format handler:

```go
type Agent struct {
    ID       string            // "claude-code"
    Name     string            // "Claude Code"
    ConfigPath string          // "~/.claude/settings.json" (~ expanded at runtime)
    Handler  FormatHandler
    EventMap map[string]string // agent's hook-event name → goob token
}

type FormatHandler interface {
    // Read current config (or a zero value if absent), return whether goob's
    // hooks are already present, and produce the new content with goob's hooks
    // added (install) or removed (uninstall). Pure: no disk I/O — takes/returns bytes.
    Installed(current []byte, a Agent) (bool, error)
    Install(current []byte, a Agent, hookCmd string) ([]byte, error)
    Remove(current []byte, a Agent) ([]byte, error)
}
```

**The marker (required).** A goob-owned hook entry is identified by exactly one
thing: **its command string contains `goob_hook.py`.** JSON configs have no
comment field, so this substring IS the marker. Consequences, specified so every
handler author gets it right:
- `Install` = **remove-by-marker, then add** (not "append if absent"). This makes
  re-runs idempotent (no duplicates) AND self-heals the "repo moved → absolute
  path changed" case (the old, now-wrong entry is stripped and replaced). CLAUDE.md
  notes the repo location moves, so this is a real case, not hypothetical.
- `Remove` = remove-by-marker (drops exactly goob's entries, leaves the rest).
- `Installed` = "any marker-matching entry present."
- This also lets the wizard adopt hooks a user added by hand via the manual doc
  (they contain `goob_hook.py` too) instead of duplicating them.

`hookCmd` is the absolute path to `goob_hook.py` invoked in **token-direct** mode
(see below). **Resolution:** `os.Executable()` is wrong here — under `just install`
(`go run .`) it points at a temp build dir. Resolve `hookCmd` from the working
directory instead: `<repo>/hooks/goob_hook.py`, where `<repo>` is `..` relative to
`installer/` (made absolute), overridable with a `--repo-root` flag. Centralize in
one path helper and unit-test that the resolved file exists.

**Shipped agents (v1):** Claude Code (first-class), Cursor, Gemini CLI.
- **Claude Code** — `~/.claude/settings.json`, `hooks.<Event>[].hooks[]` array of
  `{type:"command", command:"…"}`. Known/verified format. EventMap:
  SessionStart→wake, UserPromptSubmit→thinking, PreToolUse→working,
  PostToolUse→working, SubagentStop→subagent, Stop→done, SessionEnd→sleep.
- **Cursor** — `~/.cursor/hooks.json` (Cursor Agent hooks). Format + event names
  verified against Cursor's current hooks docs during implementation.
- **Gemini CLI** — `~/.gemini/settings.json` command hooks. A settings.json shape;
  reuses the Claude handler if close enough, else its own. Event names verified
  during implementation.
- **Free add:** CodeBuddy (`~/.codebuddy/settings.json`) is Claude-Code-compatible
  — the same handler with a different path + EventMap. Add as a data entry if wanted.

**Codex is explicitly OUT of v1.** Its config is TOML (`~/.codex/config.toml`, no
Go stdlib TOML) and its hook model is a single `notify` program that branches on
the event type *at runtime* — you cannot pin a static `--token`, so it would need
both a TOML handler and a dispatcher shim that re-introduces per-agent logic in
Python, defeating the token-direct design. Add later as a distinct sub-project.

> Implementation note: `clawd-on-desk` (`hooks/*-install.js`) already solved the
> exact per-agent config shapes and is the reference to verify against; but goob
> writes token-direct commands, not clawd's payloads.

**Behavior:** multi-select which agents to wire (checkbox list, showing each
agent's current state: *not installed / installed / config missing*). For each
selected agent, the step adds a `WriteFile` mutation from
`Handler.Install(current, agent, hookCmd)`. Deselecting an installed agent
produces a `Remove` mutation. **Idempotent** (re-running detects present hooks),
**backed up** before write.

### Hook change: token-direct mode (`goob_hook.py`)

To keep the hook agent-agnostic, `goob_hook.py` gains a token-direct form:

```
python3 goob_hook.py --token <tok>     # NEW: write <tok> directly (validated)
python3 goob_hook.py <CCEventName>     # existing: Claude-Code event → token
```

The per-agent event→token mapping lives in the Go **registry**, which generates
each agent's config to call `goob_hook.py --token <mapped-token>` on the agent's
event. `--token` guards a missing arg (`len(argv) < 3` → write nothing), validates
against the known token set (`wake/thinking/working/subagent/done/sleep`), ignores
unknowns — same swallow-all-errors contract as today. ~5-line addition; existing
event-name mode and the shipped Claude Code tests are unaffected.

**One honest caveat — the Claude Code map is duplicated.** `goob_hook.py`'s
`EVENTS` dict (for event-name mode, kept for the manual doc) and the Go registry's
Claude `EventMap` both encode SessionStart→wake…SessionEnd→sleep — the same map in
two languages. We do **not** claim single-source. Instead: `EVENTS` stays canonical
and a **cross-language test** asserts the Go Claude `EventMap` equals it (parse
`goob_hook.py`'s `EVENTS` in the Go test, compare). `docs/agent-reactivity.md` gets
a line: "prefer `just install`; the manual `settings.json` snippet is the fallback."
Wizard-generated (`--token`) and manual (event-name) entries interoperate because
both carry the `goob_hook.py` marker, so the wizard recognizes and can replace either.

## Step 3 — Environment (`.env`)

Fields: `GOOB_MODEL` (text, default `gpt-4o-mini`), an API key (masked input,
written as `GOOB_API_KEY`), and toggles `GOOB_HSM` and `DEBUG`. **Write-only** —
no provider call. Produces one mutation that **merges** into an existing `.env`:
parse existing `KEY=VALUE` lines, overwrite only the keys the wizard manages,
preserve everything else and ordering, back up first. Leaving the key field
blank writes no key line (keeps any existing one).

`envfile.go` is a tiny parser/merger (goob's own `.env` format: `KEY=val`,
`export KEY=val`, `#` comments, quotes — mirror `daemon/server.py::load_dotenv`
so the two agree). No dependency.

## Step 4 — Review & Apply

Renders the full `Plan`: each mutation's `Describe()` (file, action, backup
path). One key to confirm → `Apply()` each in order. **Continue-on-error** (the
writes are independent — one failing agent must not skip the rest), collecting
per-mutation success/failure, then report all with backup paths (backups let the
user restore). A mutation whose transform yields **unchanged** bytes is a no-op:
no write, no backup. Then → Done screen with next steps (run `GOOB_HSM=1 just run`,
register/verify, etc.).

## Safety

- **Backup on change:** `WriteFile.Apply` copies the existing file to a single
  `<path>.goob-bak` (overwrite — no growing pile of `-<n>` files) **only when the
  transform actually changes the content.** New files get no backup (nothing to
  lose); unchanged content is skipped entirely.
- **Idempotent via the marker:** `Install` = remove-by-marker + add, so
  re-running produces identical content (no duplicate entries) and Apply becomes a
  no-op.
- **Uninstall:** deselecting an installed agent removes exactly goob's entries,
  leaving other hooks intact.
- **Never touches disk before Apply.** A quit at any earlier step changes
  nothing.
- Path expansion (`~`) and repo-root resolution centralized (one helper), so a
  wrong assumption is fixed in one place.

## Testing

`go test ./...` in `installer/`, table-driven, **no real `~/` writes** — all
handlers/parsers take and return bytes. The hard, genuinely-risky part is each
real agent's schema, so tests must run against **real captured configs, not
hand-guessed strawmen**:
- `testdata/` — a **golden fixture per agent captured from the real tool**: a real
  Claude `settings.json` containing unrelated hooks + MCP servers, a real Cursor
  `hooks.json`, a real Gemini `settings.json`. "Preserve unrelated content" is
  tested against reality.
- `formats_test.go` — for each handler: install into empty, install into the
  golden fixture (all unrelated content preserved), install-when-present
  (idempotent, no dup, marker replace), **round-trip: Install → Remove returns the
  original bytes** (proves the marker is exact and removal is clean), malformed
  input → error not panic.
- `envfile_test.go` — merge into empty, merge preserving unmanaged keys + order +
  comments, blank key preserves existing.
- `registry_test.go` — every shipped Agent: non-empty EventMap whose values are all
  valid goob tokens, ConfigPath set, Handler non-nil.
- `crossmap_test.go` — the Go Claude `EventMap` equals `goob_hook.py`'s `EVENTS`
  (parse the Python dict, compare) — catches CC-map drift across the two languages.
- `doctor_test.go` — version parsing (`godot --version` / `python3 --version` →
  semver, min check) and OS-hint selection from a stubbed `/etc/os-release` `ID`;
  hookCmd path helper resolves to an existing file.
- `hooks/test_goob_hook.py` extension: `--token done` writes `{token:"done"}`;
  `--token bogus` and `--token` (no value) write nothing; existing event-name test
  still passes.

**Prove the third agent's interface-fit BEFORE building it** — the first
implementation task confirms Cursor's and Gemini's real schemas fit the
FormatHandler (capture the golden fixtures), so a mismatch surfaces before the
handlers are written, not after.

Bubbletea view/update wiring is thin glue over the tested core; no TUI-driver
tests in v1 (the logic lives in pure, tested functions). A manual run is the
UI smoke.

## Open items to confirm during implementation (first task)

- Capture real `~/.cursor/hooks.json` and `~/.gemini/settings.json` (+ event
  names) as golden fixtures and confirm both fit the FormatHandler interface
  **before** writing the handlers (clawd-on-desk `hooks/*-install.js` as
  cross-reference). If Gemini's shape diverges, either give it its own handler or
  swap it for CodeBuddy (reuses the Claude handler outright).
- Whether Cursor/Gemini reuse the Claude settings-json handler or each needs its
  own (decide once the fixtures are captured; the interface supports either).

## Scope guards

**In:** `installer/` Go module, linear plan-then-apply wizard (transform-carrying
mutations, re-read at apply), doctor (detect + copy-paste, incl. required
`python3`), agent registry with 3 shipped agents (Claude Code, Cursor, Gemini) +
marker-based install/remove, token-direct hook mode, `.env` merge, single
backup-on-change, unit tests on the pure core with real golden fixtures.

**Out (add when needed):** Codex (TOML + runtime-dispatch `notify` — a distinct
sub-project needing a TOML handler + dispatcher shim), more agents, auto-install
of deps, API-key validation, TUI-driver tests, release/packaging, auto-update,
per-agent permission-approval flows (clawd's HTTP hooks).
