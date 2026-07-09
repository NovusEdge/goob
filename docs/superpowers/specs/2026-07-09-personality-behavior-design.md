# goob — Personality & Behavior Refinement — Design

**Date:** 2026-07-09
**Status:** Approved (design); pending implementation plan
**Scope:** Ground pets only. Floating pets ([#3](https://github.com/NovusEdge/goob/issues/3)) and precise user-idle detection ([#2](https://github.com/NovusEdge/goob/issues/2)) are out of scope.

## Summary

Make the pet's behavior express a configurable **personality**, driven entirely by
data (a `PetConfig`). Add three new behaviors — `retreat`, `zoomies`, `play` — and
the config knobs that let personalities weight and shape them. Ship two preset
configs (`lazy_cat.tres`, `playful_cat.tres`) alongside the existing `cat.tres`.

## Goals

- Personality = which `PetConfig` you load. No separate "personality system."
- New behaviors: corner-retreat naps, zoomies bursts, and cursor play.
- Move the last hardcoded personality-relevant constant (`idle_delay`) into config.
- Two example personalities that read as distinctly lazy vs. playful.
- Nothing regresses for `cat.tres`; all new fields have neutral defaults.

## Non-goals

- Floating / no-gravity pets — deferred, [#3](https://github.com/NovusEdge/goob/issues/3).
- Precise "sleep while the user is away" idle detection — stubbed, [#2](https://github.com/NovusEdge/goob/issues/2).
- An energy/pent-up system — considered and rejected (zoomies is random-weighted).
- A `flee` behavior — "bothered" pets use `startle` (recoil in place).

## Personality model (Option A)

A personality **is** a `PetConfig`. The existing mood system stays exactly as the
transient multiplier layer. The picker in `pet.gd` `_decide()` already computes:

```
final_weight = base_weight(config) × mood_multiplier × chain_factor
```

Nothing new is needed here — a lazy config and a playful config simply feed
different base weights (and speeds, and action loop counts), and mood scaling
lands on top of each. This also captures the **non-weight** half of personality
(nap length, movement speeds, loiter time), which lives directly in the config.

## PetConfig — new fields

| Field | Type | Default | Meaning |
|-------|------|---------|---------|
| `idle_delay` | int | 90 | Base loiter ticks before the pet picks a new action. Replaces the hardcoded value; still mood-scaled (alert ×0.5, tired ×1.7 — matching today's 45/150). |
| `zoomies_weight` | int | 0 | Picker weight for zoomies (0 = disabled). |
| `zoomies_cooldown_sec` | float | 20.0 | Minimum seconds between zoomies. |
| `zoomies_duration_sec` | float | 10.0 | How long a zoomies burst lasts. |
| `zoomies_speed_mult` | float | 2.5 | Dart speed as a multiple of `follow_speed`. |
| `retreat_interval_sec` | float | 0.0 | Seconds between cadence retreats (0 = off). |
| `jiggle_reaction` | String | `"follow"` | Which behavior a cursor-jiggle triggers. |
| `follow_reach` | String | `"dash"` | What happens when `follow` reaches the cursor: `"dash"` or `"play"`. |

Time fields are seconds (user-friendly); the engine converts to 60 Hz ticks.

## New behaviors

### retreat

- **Trigger:** a cadence timer. Every `retreat_interval_sec` (if > 0), when the pet
  is grounded and interruptible, it retreats.
- **Motion:** walk to the **nearest bottom corner** (target x = 0 or `screen_w -
  body_w`, whichever is closer), wander-style.
- **On arrival:** a long nap (reuse the `nap`/sleep animation; ~2–3× a normal nap).
- **Idle trigger (stubbed):** a `_user_idle()` hook returns `false` for now. When a
  real idle source lands ([#2](https://github.com/NovusEdge/goob/issues/2)), idle →
  retreat + sleep until input resumes. The hook is the only seam; nothing else changes.

### zoomies

- **Trigger:** a normal weighted picker entry (`zoomies_weight`), gated by a
  cooldown of `zoomies_cooldown_sec` so it can't chain. (The drain isn't modeled;
  the cooldown is the whole gate.)
- **Motion:** for `zoomies_duration_sec` (~10s), dart back and forth in fast local
  bursts — pick a short random distance + direction at `follow_speed ×
  zoomies_speed_mult`, reverse on reaching it or hitting an edge, repeat.
- **Animation:** the fast-run anim (resolves via the normal alias chain, e.g.
  `sprint`/`running`).
- **End:** → `idle`.

### play

- **Trigger:** entered from `follow` when it reaches the cursor (~within 30px) **and**
  `follow_reach == "play"` **and** a `play` animation resolves. Otherwise the current
  `dash` fires (unchanged default — non-playful creatures are unaffected).
- **Engagement loop:** face the cursor, loop the `play` animation in place. If the
  cursor moves ~60px+ away, return to `follow` to catch up, then play again.
- **End:** after a few play loops (or the cursor leaves the area), satisfied → `idle`.
- **Animation:** a new `play` anim the creature authors, mapped via
  `aliases = {"play": "<anim>"}`.

## idle_delay lift

`_idle_delay()` currently returns hardcoded 45/150/90. Change it to read
`cfg.idle_delay` as the neutral base and apply the mood scale (alert ≈ ×0.5,
tired ≈ ×1.7). This lets "lazy loiters longer / playful fidgets sooner" be a
config choice.

## Pickup animation (documentation only)

The engine already has `grab` → `carry` → `drop` states (they fire on drag) and
resolves them through the alias chain — today they fall back to `idle` because the
cat has no pickup art. Authoring `pickup`/`hold`/`putdown` animations and adding
`aliases = {"grab":"pickup", "carry":"hold", "drop":"putdown"}` makes them play,
no code. Document this in `configuration.md` / `behavior-model.md`.

## Preset configs (illustrative values; final tuning in implementation)

**lazy_cat.tres** — rests a lot, roams little, doesn't like to be bothered:
- `idle_weight` high, `wander_weight`/`follow_weight` low, `jump_weight` 0
- `idle_delay` long (~180), `nap` loops long
- `zoomies_weight` 0 (off)
- `retreat_interval_sec` ~900 (retreats to corners to nap)
- `jiggle_reaction: "startle"`, `follow_reach: "dash"`
- `alert_weights` modest (barely perks up); `tired_weights` boosts nap heavily

**playful_cat.tres** — busy, follows, zoomies, plays:
- `wander_weight`/`follow_weight` high, `jump_weight` 0
- `idle_delay` short (~45), `nap` loops short
- `zoomies_weight` high, cooldown ~20s, duration ~10s, speed ×2.5
- `retreat_interval_sec` 0 (off)
- `jiggle_reaction: "follow"`, `follow_reach: "play"` (needs a `play` anim; falls back to `dash` until authored)
- `alert_weights` amplify wander/follow/zoomies; `tired_weights` fall back to rest

`cat.tres` stays as the balanced default; all new fields default to neutral so it
behaves as it does today.

## Files touched

- `pet_config.gd` — new `@export` fields.
- `pet.gd` — `retreat`, `zoomies`, `play` states/logic; `_user_idle()` stub;
  `_idle_delay()` reads config; cooldown/interval timers; reuse the existing
  restful/active split.
- `cat.tres` — add neutral defaults for new fields (no behavior change).
- `lazy_cat.tres`, `playful_cat.tres` — new presets.
- `docs/configuration.md`, `docs/behavior-model.md` — document new fields,
  behaviors, pickup aliasing.

## Testing / acceptance

Verified live via the Godot MCP (`run_project` → `get_debug_output` → `stop_project`)
and by eyeball. Acceptance:

- `cat.tres` behaves as before (no regression).
- `lazy_cat.tres` visibly rests more, retreats to a corner and naps, `startle`s on jiggle.
- `playful_cat.tres` wanders/follows more, does a ~10s zoomies burst, and (with a
  play anim) bats at the cursor on reach; falls back to `dash` without one.
- No runtime errors; only the known intentional integer-division warnings.

## Deferred / linked

- [#2](https://github.com/NovusEdge/goob/issues/2) — precise user-idle detection (retreat Trigger B).
- [#3](https://github.com/NovusEdge/goob/issues/3) — FloatPet class.
