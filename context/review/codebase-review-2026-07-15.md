# Codebase Review - 2026-07-15

**Mode**: full  
**Branch**: main  
**Files reviewed**: ~50 source files (GDScript, Python, Go)

## Executive Summary

| Category | P0 | P1 | P2 | P3 |
|----------|----|----|----|----|
| Logic | 0 | 1 | 0 | 0 |
| Error Handling | 0 | 0 | 2 | 0 |
| Security | 0 | 0 | 1 | 0 |
| API/LLM | 0 | 0 | 1 | 0 |
| Code Quality | 0 | 0 | 0 | 4 |
| **Total** | **0** | **1** | **4** | **4** |

No P0s. One P1 bug in the daemon's multi-step LLM path that can cause 400 errors on certain providers.

## Themes

1. **LLM integration patterns** — The tool_calls format bug and missing timeout are both in the LLM integration layer. Both are containable.
2. **Defensive error handling** — The codebase consistently swallows errors to avoid breaking the pet or caller, which is correct for this use case but masks some edge cases.

---

## Findings

### P1 - Logic Error

| ID | Location | Issue | Recommendation | Effort |
|----|----------|-------|----------------|--------|
| 1 | daemon/server.py:128-131 | **Missing `type` field in tool_calls**: When building the assistant message to echo back to litellm, each tool_call dict is `{"id": ..., "function": {...}}` but OpenAI-compatible APIs require `"type": "function"`. This only fires on multi-step paths (model calls an observer tool, then emits) — the next completion can 400. Single-step ticks work, hiding the bug. | Add `"type": "function"` to each dict in the list comprehension. | S |

### P2 - Edge Cases / Hardening

| ID | Location | Issue | Recommendation | Effort |
|----|----------|-------|----------------|--------|
| 2 | daemon/server.py:112 | **No timeout on litellm.completion**: Relies on litellm's default (can be long). Pet's HTTP client times out first (6s), but the server thread keeps running and still bills spend. Stalled providers can pile up threads. | Pass `timeout=5` to `litellm.completion`. | S |
| 3 | daemon/server.py:141 | **Content-Length not capped**: `int(self.headers.get("Content-Length", 0))` then `read(n)` with no max. Large/negative values could allocate big or behave oddly. Localhost-only mitigates. | Clamp to e.g. 64 KiB and guard against negative. | S |
| 4 | daemon/agent.py:26 | **No length cap on `say`**: If the LLM returns a very long string, it goes straight to the speech bubble. | Cap to ~200 chars in `_validate`. | S |

### P3 - Code Quality

| ID | Location | Issue | Recommendation | Effort |
|----|----------|-------|----------------|--------|
| 5 | daemon/server.py:56 | **Quote stripping logic**: `.strip('"').strip("'")` sequentially mangles mismatched quotes. | Strip only a single matching surrounding quote pair. | S |
| 6 | daemon/server.py:109 | **VERTEXAI_API_KEY not checked**: Docstring lists it but code doesn't look for it. | Align docstring and code. | S |
| 7 | daemon/sysmon.py:47 + scripts/sysmon.gd:47 | **Battery "charging" includes Unknown/Not charging/Full**: Both sysmon implementations return charging=True for any status != "Discharging". | Treat only "Charging"/"Full" as charging if the distinction matters. | S |
| 8 | hooks/goob_hook.py + goob_codex_notify.py | **Duplicated tempfile/atomic-write logic**: Both hooks have the same mkstemp/write/replace pattern. | Factor into a shared module (optional — code is tiny). | S |

---

## Architecture Notes

**Good patterns observed:**
- DRIVABLE sync between `pet.gd` and `daemon/agent.py` is documented and enforced
- Agent-reactivity is cleanly layered (HSM on top of normal behavior, opt-out via env var)
- Defensive error handling throughout — the pet never wedges a caller
- GDScript state machine is well-structured with clear separation of BEHAVIORS vs CLIPS
- Test coverage for agent_poller static logic is solid

**Test gaps:**
- daemon/agent.py multi-step path (the P1 bug) has no test — only single-step ticks are tested
- No integration test for the full Godot → daemon → LLM → response flow
- installer/ has good unit test coverage (formats_test.go, registry_test.go, etc.)

---

## Recommended Fixes (Effort-Weighted)

1. **P1 #1** (S): Add `"type": "function"` to tool_calls in server.py — one-line fix
2. **P2 #2** (S): Add `timeout=5` to litellm.completion
3. **P2 #3** (S): Clamp Content-Length
4. **P2 #4** (S): Cap `say` length in agent.py

All four are quick fixes. The P1 should be fixed immediately.
