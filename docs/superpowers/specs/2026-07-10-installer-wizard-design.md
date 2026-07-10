# goob — setup wizard (`installer/`, Go + Bubbletea)

**Date:** 2026-07-10
**Status:** approved design, pre-implementation (Opus review pending)

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
`Plan` — an ordered list of pending mutations (`WriteFile{path, newContent,
backupPath}`). Only the Review & Apply step, after an explicit confirm, executes
the plan. This makes the whole wizard a pure function of user input until one
commit point, and makes "what will this change?" fully inspectable.

```go
type Plan struct { Steps []Mutation }
type Mutation interface {
    Describe() string          // one-line summary for the review screen
    Apply() error              // backup-then-write
}
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
  agents.go          — the shipped agent data (claude, cursor, codex)
  formats.go         — FormatHandler impls (settings-json, cursor, codex)
  envfile.go         — parse/merge/serialize .env
  plan.go            — Mutation + WriteFile (backup-then-write)
  *_test.go          — table-driven tests for registry/formats/envfile
```

## Step 1 — Doctor (deps: detect + copy-paste)

Probes, each `{name, found bool, version string, ok bool, hint string}`:

| dep | probe | needed for | min |
|-----|-------|-----------|-----|
| `godot` | `LookPath("godot")` or `$GODOT`; parse `--version` | the pet | 4.6 (LimboAI) |
| `uv` | `LookPath("uv")` | daemon (optional) | any |
| `go` | `LookPath("go")` | TUI (optional) | any |

Renders a green/red checklist. For a missing/too-old dep, shows an
**OS-specific copy-paste command** — never executed. OS detection: `runtime.GOOS`
(darwin → `brew`), and on Linux read `/etc/os-release` `ID`/`ID_LIKE` →
`pacman -S` / `apt install` / `dnf install`; unknown distro → the upstream URL
(e.g. `https://docs.astral.sh/uv/`). godot is a special case (no universal
package name) → link to the download / note the vendored engine expectation.

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

`hookCmd` is the absolute path to `goob_hook.py` (resolved from the installer's
own location / repo root) invoked in **token-direct** mode — see below.

**Shipped agents (v1):** Claude Code (first-class), Cursor, Codex.
- **Claude Code** — `~/.claude/settings.json`, `hooks.<Event>[].hooks[]` array of
  `{type:"command", command:"…"}`. Known/verified format. EventMap:
  SessionStart→wake, UserPromptSubmit→thinking, PreToolUse→working,
  PostToolUse→working, SubagentStop→subagent, Stop→done, SessionEnd→sleep.
- **Cursor** — `~/.cursor/hooks.json` (Cursor Agent hooks). Format + event names
  verified against Cursor's current hooks docs during implementation.
- **Codex** — `~/.codex/` config. Format + event names verified against Codex's
  current hooks docs during implementation.

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
event. This means the mapping lives in exactly one place per language boundary
and the hook stays a dumb sink. `--token` validates against the known token set
(`wake/thinking/working/subagent/done/sleep`) and ignores unknowns, same
swallow-all-errors contract as today. ~5-line addition; existing event-name mode
and the shipped Claude Code tests are unaffected.

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
path). One key to confirm → `Apply()` each in order, showing per-step
success/failure. On any failure, stop and report what succeeded (backups let the
user restore). Then → Done screen with next steps (run `GOOB_HSM=1 just run`,
etc.).

## Safety

- **Backup before every write:** `plan.go`'s `WriteFile.Apply` copies the
  existing file to `<path>.goob-bak-<n>` before writing (n avoids clobbering a
  prior backup). New files get no backup (nothing to lose).
- **Idempotent:** install detects already-present hooks (no duplicate entries);
  re-running Apply is a no-op if nothing changed.
- **Uninstall:** deselecting an installed agent removes exactly goob's entries,
  leaving other hooks intact.
- **Never touches disk before Apply.** A quit at any earlier step changes
  nothing.
- Path expansion (`~`) and repo-root resolution centralized (one helper), so a
  wrong assumption is fixed in one place.

## Testing

`go test ./...` in `installer/`, table-driven, **no real `~/` writes** — all
handlers/parsers take and return bytes:
- `formats_test.go` — for each FormatHandler: install into empty config, install
  into a config with unrelated hooks (preserved), install-when-already-present
  (idempotent, no dup), remove (leaves unrelated hooks), malformed input →
  error not panic.
- `envfile_test.go` — merge into empty, merge preserving unmanaged keys +
  order + comments, blank key preserves existing.
- `registry_test.go` — every shipped Agent has a non-empty EventMap whose values
  are all valid goob tokens; ConfigPath set; Handler non-nil.
- `doctor_test.go` — version parsing (`godot --version` string → semver, min
  check) and OS-hint selection from a stubbed `/etc/os-release` `ID`.
- A `goob_hook.py` test extension: `--token done` writes `{token:"done"}`;
  `--token bogus` writes nothing; existing event-name test still passes.

Bubbletea view/update wiring is thin glue over the tested core; no TUI-driver
tests in v1 (the logic lives in pure, tested functions). A manual run is the
UI smoke.

## Open items to confirm during implementation

- Exact Cursor (`~/.cursor/hooks.json`) and Codex (`~/.codex/…`) config schemas
  and event names — verify against each agent's current hook docs (clawd-on-desk
  as cross-reference) before writing their handlers/EventMaps.
- Whether Cursor/Codex share a JSON shape close enough to reuse the Claude
  settings-json handler or each needs its own (decide once schemas are confirmed;
  the interface supports either).

## Scope guards

**In:** `installer/` Go module, linear plan-then-apply wizard, doctor
(detect+copypaste), agent registry with 3 shipped agents + install/remove,
token-direct hook mode, `.env` merge, backups, unit tests on the pure core.

**Out (add when needed):** more agents, auto-install of deps, API-key
validation, TUI-driver tests, release/packaging, auto-update, per-agent
permission-approval flows (clawd's HTTP hooks).
