# goob — behavior model

How the desktop companion thinks, and how to make it *your* creature (cat, dog,
robot, slime, blob, ghost…) without touching code.

The guiding split:

> **Engine = what a companion *does* (universal, in code).**
> **Creature = how *your* creature expresses it (data, in a config).**

The engine knows how to loiter, wander, follow your cursor, get picked up, and
react to your machine. It does *not* know it's a cat. Which animation plays for
each behavior, which expressive fidgets exist, how often they happen, and how
fast it moves — all of that is **data you supply per creature**.

---

## 1. Engine core behaviors

These are the universal building blocks. They have real logic (movement,
physics, cursor tracking, drag), so they live in code. Names are **neutral
verbs** — nothing cat-specific.

| Behavior   | What it does                                   | Leaves to        |
|------------|------------------------------------------------|------------------|
| `appear`   | one-shot intro when the pet spawns             | `idle`           |
| `idle`     | the hub — loiters, then picks a next action    | (any)            |
| `wander`   | roams to a random spot when bored              | `idle` on arrival|
| `follow`   | seeks the cursor out (wants attention / pets)  | `dash` / `idle`  |
| `dash`     | the burst on reaching the cursor               | `idle`           |
| `jump`     | a hop (uses gravity)                           | `idle` on landing|
| `grab`     | being picked up (drag start)                   | `carry`          |
| `carry`    | held under the cursor                          | `drop` on release|
| `drop`     | put down                                       | `idle` (falls)   |
| `startle`  | spooked by a right-click or cursor jiggle      | `idle`           |

`fall` (gravity) isn't a discrete state — it's a rule applied whenever the pet
isn't grounded and isn't being carried.

**`wander` vs `follow`** are deliberately two behaviors, not one: `wander` is
aimless — the pet is bored and drifts to a random spot. `follow` is *motivated* —
it seeks the cursor out because it wants attention (a hook for future interaction
like pets/cuddles). Same locomotion, different intent, and a creature can enable
one without the other.

**Renamed from the old cat-flavored names:** `chase→follow`, `pounce→dash`,
`walk→wander`, `scared→startle`, `spawn→appear`, `pickup→grab`, `held→carry`,
`putdown→drop`.

### Toggles

Some core behaviors are optional per creature — a potted-plant companion
probably shouldn't chase your mouse:

- `follow_cursor` — off = it never seeks the cursor (stays put / only wanders).
- `gravity` — off = it floats (a ghost, a UI sprite) instead of falling.

---

## 2. Creature actions (the expressive fidgets)

Everything *characterful* a creature does when idle is **data**, not code —
because it all follows one pattern: *play an animation for a while, then return
to idle.* So each action is just four fields:

```
{ name, anim, weight, loops }
```

- `name`   — what you call it (`nap`, `recharge`, `jiggle`)
- `anim`   — which authored animation to play
- `weight` — how likely the picker chooses it (relative to others + `wander`)
- `loops`  — how many full animation cycles to play before returning to idle

Example action sets for different creatures:

| Creature | Actions |
|----------|---------|
| **Cat**   | nap, groom, loaf, stretch, yawn, sound (meow), roll, sit |
| **Dog**   | sleep, scratch, wag, bark, dig, sit, shake |
| **Robot** | recharge, scan, glitch, beep, idle-spin |
| **Slime** | jiggle, split, ooze, bubble |

You add / remove / reweight these freely. That's the "change states and
behavior" knob — no code, just a list.

---

## 3. Reactions

Behaviors triggered by the world instead of the idle picker:

- **Mood (system state).** Every ~2s the pet samples the machine:
  - a build/dev process running → **alert** (moves/follows more, rests less)
  - hot CPU or low battery → **tired** (rests/naps more, no chasing)

  A mood reshapes the action weights and the idle cadence. Which actions each
  mood favors is configurable.

- **Cursor jiggle.** Shaking the cursor near the pet summons a reaction
  (default: `follow`). Configurable which behavior it triggers.

- **Drag / right-click.** Left-drag → `grab`/`carry`/`drop`. Right-click →
  `startle`. (Engine-level, always on.)

---

## 4. Mapping: canonical behavior → your animation

The engine emits **canonical names** (`follow`, `nap`, `appear`…). Your
SpriteFrames has whatever animation names *you* authored. The **alias map**
bridges them:

```
aliases = {
    "wander":  "walking",
    "follow":  "running",
    "dash":    "sprint",
    "startle": "scared",
    "appear":  "appear",     # already matches — alias optional
    "idle":    "idle",       # already matches
    "nap":     "sleeping",
    "groom":   "scratch",
}
```

### The cat reference (this repo's default)

The included cat sheet authors: `appear, idle, walking, running, sprint,
scared, sleeping, scratch`. It maps cleanly:

| Engine / action | Cat animation |
|-----------------|---------------|
| `appear`  | `appear`   |
| `idle`    | `idle`     |
| `wander`  | `walking`  |
| `follow`  | `running`  |
| `dash`    | `sprint`   |
| `startle` | `scared`   |
| `nap`     | `sleeping` |
| `groom`   | `scratch`  |

## 5. Fallback resolution

If a creature doesn't map or author some behavior, it degrades gracefully
instead of erroring. Resolution order for any canonical name:

1. an authored animation with that exact name, else
2. its `alias`, else
3. walk the fallback chain toward `idle` (`follow → wander → idle`,
   `nap → rest → idle`, …), else
4. `idle` (the one animation every creature must have).

**Only `idle` is required.** A creature with just `idle` + `walking` still runs —
everything else falls back to those.

---

## 6. PetConfig (the per-creature data asset)

Everything above is bundled into one resource per creature (`cat.tres`,
`robot.tres`, …):

```
PetConfig:
  sprite_frames : SpriteFrames   # the authored animations
  scale         : int            # pixel scale-up (e.g. 5)
  aliases       : Dictionary     # engine/action name -> authored anim
  actions       : Array          # [{name, anim, weight, loops}, ...]
  wander_weight : int            # how often it wanders vs. fidgets
  follow_cursor : bool
  gravity       : bool
  wander_speed  : int
  follow_speed  : int
  mood_actions  : Dictionary     # mood -> weight tweaks / favored actions
```

Swap the resource → swap the creature. `main.gd` loads a `PetConfig`; the state
machine reads its `actions` + weights; core behaviors resolve through `aliases`.

---

## 7. Bring your own creature

1. Author a `SpriteFrames` in the Godot editor (animations named however you like).
2. Create a `PetConfig` resource pointing at it.
3. Fill `aliases` so at least `idle` resolves; map `wander`/`follow` if it moves.
4. List your `actions` (the fidgets) with weights.
5. Set `scale`, `follow_cursor`, `gravity`, speeds.

No scripting required.

---

## Status

- **Implemented:** the engine core behaviors + moods + jiggle + drag, animations
  driven from an authored `SpriteFrames`, alias + fallback resolution. Currently
  the alias/weights live in `main.gd` / `pet.gd` (cat defaults).
- **Planned:** extract the per-creature data into a `PetConfig` resource and
  rename the in-code states to the neutral verbs above, shipping a `cat.tres`
  that reproduces today's behavior. Then the whole table above becomes editable
  data.
