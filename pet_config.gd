class_name PetConfig
extends Resource

## One creature = one PetConfig. Pairs a SpriteFrames with how it maps onto the
## engine, which expressive actions it has, and its personality. See
## docs/behavior-model.md.

## --- Visual ---------------------------------------------------------------
## Optional: if set, overrides the AnimatedSprite2D's own SpriteFrames.
@export var sprite_frames: SpriteFrames
@export var scale: int = 6

## --- Mapping --------------------------------------------------------------
## Engine behaviour name -> the animation your SpriteFrames actually has.
## Only what differs from a direct name match is needed. Engine names:
## appear, idle, wander, follow, dash, jump, grab, carry, drop, startle.
@export var aliases: Dictionary = {}

## --- Actions (expressive idle fidgets) ------------------------------------
## Each: { "name": String, "anim": String, "weight": int, "loops": int }
## `anim` is an animation name in your SpriteFrames; `name` is just a label.
@export var actions: Array = []

## --- Autonomous behaviour weights -----------------------------------------
@export var idle_weight: int = 8
@export var wander_weight: int = 25
@export var follow_weight: int = 10
@export var jump_weight: int = 5
## Base ticks the pet loiters in idle before picking a new action (mood-scaled).
@export var idle_delay: int = 90

## --- Toggles & movement ---------------------------------------------------
@export var follow_cursor: bool = true
@export var gravity: bool = true
@export var wander_speed: int = 1
@export var follow_speed: int = 5

## --- Personality behaviours -----------------------------------------------
## Zoomies: a fast dart-fest. weight = picker weight (0 = off).
@export var zoomies_weight: int = 0
@export var zoomies_cooldown_sec: float = 20.0
@export var zoomies_duration_sec: float = 10.0
@export var zoomies_speed_mult: float = 2.5
## Retreat: amble to a corner and nap every N seconds (0 = off).
@export var retreat_interval_sec: float = 0.0
## Which behaviour a cursor-jiggle triggers (e.g. "follow", "startle").
@export var jiggle_reaction: String = "follow"
## What happens when `follow` reaches the cursor: "dash" or "play".
@export var follow_reach: String = "dash"

## --- Moods ----------------------------------------------------------------
## Per-mood weight multipliers by behaviour/action name (missing = 1.0, 0 = off).
@export var alert_weights: Dictionary = {}
@export var tired_weights: Dictionary = {}
## Optional one-shot animation to play when the mood flips (empty = none).
@export var alert_reaction: String = ""
@export var tired_reaction: String = ""
