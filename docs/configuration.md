# Configuration

Each creature is one **PetConfig** resource (`.tres`). The bundled `cat.tres` is
the default — `main.gd` loads it unless you assign a different `PetConfig` to the
`config` field on the `Main` node in the Inspector.

Edit a PetConfig visually in the Godot **Inspector**, or hand-edit the `.tres`
text. Nothing here requires scripting.

## Fields

### Visual

| Field | Type | Meaning |
|-------|------|---------|
| `sprite_frames` | SpriteFrames | Optional. If set, overrides the `AnimatedSprite2D`'s own animations. Leave empty to use the ones authored on the node. |
| `scale` | int | Pixel scale-up. The cat uses `5` (32px art → 160px). |

### Mapping

| Field | Type | Meaning |
|-------|------|---------|
| `aliases` | Dictionary | Engine behavior → your animation name, e.g. `{"wander": "walking"}`. Only entries that differ from a direct name match are needed. Engine names: `appear, idle, wander, follow, dash, jump, grab, carry, drop, startle`. |

### Actions (idle fidgets)

`actions` is an array of dictionaries — the expressive things the pet does when
idle. Each entry:

| Key | Type | Meaning |
|-----|------|---------|
| `name` | String | Label (used by mood weights). |
| `anim` | String | An animation in your SpriteFrames to play. |
| `weight` | int | How likely the idle picker chooses it. |
| `loops` | int | How many full animation cycles before returning to idle. |

```
actions = [
  { "name": "nap",   "anim": "sleeping", "weight": 6, "loops": 2 },
  { "name": "groom", "anim": "scratch",  "weight": 6, "loops": 3 },
]
```

### Behavior weights

How often the idle picker chooses each core behavior (relative to each other and
to the actions above):

| Field | Type | Default |
|-------|------|---------|
| `idle_weight` | int | 8 |
| `wander_weight` | int | 25 |
| `follow_weight` | int | 10 |
| `jump_weight` | int | 5 |

### Toggles & movement

| Field | Type | Meaning |
|-------|------|---------|
| `follow_cursor` | bool | If off, the pet never seeks the cursor (only wanders). |
| `gravity` | bool | If off, the pet floats instead of falling (and can't jump). |
| `wander_speed` | int | Pixels/tick while wandering. |
| `follow_speed` | int | Pixels/tick while following the cursor. |

### Moods

The pet samples the machine every ~2s (see [Behavior Model](behavior-model.md)).
Each mood applies **weight multipliers** on top of the base weights, by
behavior/action name (missing = `1.0`, `0` disables):

| Field | Type | Meaning |
|-------|------|---------|
| `alert_weights` | Dictionary | Multipliers when a build/dev process is running. |
| `tired_weights` | Dictionary | Multipliers when the CPU is hot or the battery is low. |
| `alert_reaction` | String | Optional one-shot animation to play when it turns alert (`""` = none). |
| `tired_reaction` | String | Optional one-shot animation when it turns tired. |

```
alert_weights = { "wander": 1.5, "follow": 2.0, "nap": 0.0 }
tired_weights = { "follow": 0.0, "jump": 0.0, "nap": 3.0, "wander": 0.3 }
```

## Making a new creature

1. Author a `SpriteFrames` in the Godot editor (name the animations whatever you like).
2. Create a new `PetConfig` resource (right-click in the FileSystem → New Resource → PetConfig).
3. Set `aliases` so at least `idle` resolves; map `wander`/`follow`/`dash` if it moves.
4. List your `actions`, set weights, `scale`, `follow_cursor`, `gravity`, speeds.
5. Assign it to the `config` field on `Main` (or save it as `cat.tres` to replace the default).

Only an `idle` animation is strictly required — everything else falls back
gracefully. See [Behavior Model](behavior-model.md) for the fallback rules.
