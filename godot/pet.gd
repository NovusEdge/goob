class_name PetBrain
extends RefCounted

# Direct port of internal/pet/behavior.go. Timers count physics ticks (60/s),
# matching the Go frame counts. See main.gd for how it's driven.

var x := 0
var y := 0
var vel_x := 0
var vel_y := 0
var facing_left := false
var state := "spawn"
var anim := "spawn"
var target_x := 0
var timer := 0
var screen_w := 0
var screen_h := 0
var frame_w := 0
var frame_h := 0
var variant := 0
var mood := 0            # 0 neutral, 1 alert, 2 tired
var loop_lens := {}      # anim -> ticks for one full loop

var prev_cursor_x := 0
var cursor_dir := 0
var cursor_seen := false
var jiggle := 0

# Transient states that play an animation for `loops` cycles then return to idle.
const HOLD_STATES := {
	"spawn":   {"anim": "spawn",   "anim2": "",       "loops": 1},
	"pounce":  {"anim": "pounce",  "anim2": "",       "loops": 1},
	"scared":  {"anim": "scared",  "anim2": "",       "loops": 2},
	"paw":     {"anim": "paw",     "anim2": "",       "loops": 3},
	"stretch": {"anim": "stretch", "anim2": "",       "loops": 1},
	"yawn":    {"anim": "yawn",    "anim2": "",       "loops": 1},
	"meow":    {"anim": "meow",    "anim2": "",       "loops": 2},
	"roll":    {"anim": "roll",    "anim2": "",       "loops": 1},
	"clean":   {"anim": "clean",   "anim2": "clean2", "loops": 3},
	"sit":     {"anim": "sit",     "anim2": "sit2",   "loops": 3},
	"loaf":    {"anim": "loaf",    "anim2": "",       "loops": 2},
	"sleep":   {"anim": "sleep",   "anim2": "",       "loops": 2},
	"putdown": {"anim": "putdown", "anim2": "",       "loops": 1},
}

func setup(sw: int, sh: int, fw: int, fh: int, lens: Dictionary) -> void:
	screen_w = sw
	screen_h = sh
	frame_w = fw
	frame_h = fh
	loop_lens = lens
	x = sw / 2
	y = sh - fh
	state = "spawn"
	anim = "spawn"

func loop_len(a: String) -> int:
	var n: int = loop_lens.get(a, 0)
	return n if n > 0 else 60

func update(cursor_x: int, cursor_y: int) -> void:
	var ground := screen_h - frame_h
	_detect_jiggle(cursor_x)

	if HOLD_STATES.has(state):
		var h: Dictionary = HOLD_STATES[state]
		anim = h.anim
		if h.anim2 != "" and variant == 1:
			anim = h.anim2
		vel_x = 0
		timer += 1
		if timer >= h.loops * loop_len(h.anim):
			timer = 0
			state = "idle"
	else:
		match state:
			"idle":
				anim = "idle2" if variant == 1 else "idle"
				vel_x = 0
				timer += 1
				if timer > _idle_delay():
					timer = 0
					_decide_next_action()
			"walk":
				anim = "walk"
				var dx := target_x - x
				if abs(dx) < 4:
					state = "idle"
					vel_x = 0
					timer = 0
				elif dx > 0:
					vel_x = 1
					facing_left = false
				else:
					vel_x = -1
					facing_left = true
			"chase":
				anim = "run"
				if cursor_x >= 0 and cursor_y >= 0:
					var dx := cursor_x - x - frame_w / 2
					if abs(dx) < 30:
						state = "pounce"
						timer = 0
						vel_x = 0
					elif dx > 0:
						vel_x = 5
						facing_left = false
					else:
						vel_x = -5
						facing_left = true
					timer += 1
					if timer > 300:
						state = "idle"
						timer = 0
				else:
					state = "idle"
			"jump":
				anim = "jump"
				vel_y = -8
				state = "airborne"
			"airborne":
				anim = "jump"
			"pickup":
				anim = "pickup"
				vel_x = 0
				vel_y = 0
				timer += 1
				if timer > 36:
					state = "held"
					timer = 0
			"held":
				anim = "held2" if variant == 1 else "held"
				vel_x = 0
				vel_y = 0

	x += vel_x

	# gravity — a carried cat hangs from the cursor, it doesn't fall
	if not held():
		if y < ground:
			vel_y += 1
			if vel_y > 10:
				vel_y = 10
		else:
			y = ground
			if vel_y > 0:
				vel_y = 0
				if state == "airborne":
					state = "idle"
		y += vel_y

	_clamp()

func _idle_delay() -> int:
	match mood:
		1: return 45
		2: return 150
		_: return 90

func _decide_next_action() -> void:
	variant = randi() % 2
	if not grounded():
		return

	var actions := ["walk", "walk", "chase", "sit", "clean", "sleep", "paw",
		"jump", "stretch", "yawn", "loaf", "meow", "roll", "idle"]
	var weights := [25, 15, 10, 8, 6, 5, 5, 5, 4, 4, 4, 3, 3, 8]
	match mood:
		1: weights = [35, 20, 18, 6, 2, 0, 8, 6, 2, 1, 0, 6, 2, 4]
		2: weights = [8, 5, 0, 12, 6, 22, 2, 0, 4, 10, 15, 2, 2, 12]

	var total := 0
	for w in weights:
		total += w
	var roll := randi() % total
	var s := 0
	for i in weights.size():
		s += weights[i]
		if roll < s:
			state = actions[i]
			break

	if state == "walk":
		target_x = randi() % (screen_w - frame_w)

func set_mood(m: int) -> void:
	if m == mood:
		return
	mood = m
	if not grounded() or not _interruptible():
		return
	if m == 2:
		state = "yawn"
		timer = 0
	elif m == 1:
		state = "meow"
		timer = 0

func _interruptible() -> bool:
	return not (state in ["held", "pickup", "putdown", "scared", "chase", "pounce"])

func _detect_jiggle(cx: int) -> void:
	if cx < 0:
		jiggle = 0
		cursor_seen = false
		cursor_dir = 0
		return
	if not cursor_seen:
		cursor_seen = true
		prev_cursor_x = cx
		return
	var dx := cx - prev_cursor_x
	prev_cursor_x = cx
	if jiggle > 0:
		jiggle -= 1
	if abs(dx) < 4:
		return
	var dir := 1 if dx > 0 else -1
	if cursor_dir != 0 and dir != cursor_dir:
		jiggle += 3
	cursor_dir = dir
	if jiggle >= 12 and grounded() and _interruptible():
		state = "chase"
		timer = 0
		jiggle = 0

func _clamp() -> void:
	if x < 0:
		x = 0
		vel_x = 0
	if x > screen_w - frame_w:
		x = screen_w - frame_w
		vel_x = 0
	if y < 0:
		y = 0
		vel_y = 0
	if y > screen_h - frame_h:
		y = screen_h - frame_h
		vel_y = 0

func grounded() -> bool:
	return y >= screen_h - frame_h

func held() -> bool:
	return state == "held" or state == "pickup"

func scare() -> void:
	if not (state in ["scared", "held", "pickup", "putdown"]):
		state = "scared"
		timer = 0

func do_jump() -> void:
	if grounded():
		state = "jump"
		timer = 0

func hold(px: int, py: int) -> void:
	if state != "held" and state != "pickup":
		state = "pickup"
		timer = 0
		variant = randi() % 2
	x = px - frame_w / 2
	y = py - frame_h / 2

func release() -> void:
	if state == "held" or state == "pickup":
		state = "putdown"
		timer = 0
