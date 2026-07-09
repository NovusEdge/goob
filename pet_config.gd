class_name PetConfig
extends Resource

## One creature = one PetConfig. Pairs a SpriteFrames with how it maps onto the
## engine, which expressive actions it has, and its personality. See
## docs/behavior-model.md.

## --- Visual ---------------------------------------------------------------
## Optional: if set, overrides the AnimatedSprite2D's own SpriteFrames.
@export var sprite_frames: SpriteFrames
@export var scale: int = 5

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

## --- Toggles & movement ---------------------------------------------------
@export var follow_cursor: bool = true
@export var gravity: bool = true
@export var wander_speed: int = 1
@export var follow_speed: int = 5

## --- Moods ----------------------------------------------------------------
## Per-mood weight multipliers by behaviour/action name (missing = 1.0, 0 = off).
@export var alert_weights: Dictionary = {}
@export var tired_weights: Dictionary = {}
## Optional one-shot animation to play when the mood flips (empty = none).
@export var alert_reaction: String = ""
@export var tired_reaction: String = ""
