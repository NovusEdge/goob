class_name PetBrain
extends RefCounted

# The behaviour engine. Universal companion verbs live here; the creature's
# animations, expressive actions, weights and personality come from a PetConfig.
# See docs/behavior-model.md. Timers count physics ticks (60/s).
#
# Two tiers of state:
#   BEHAVIORS decide things and run indefinitely:
#     idle    - the hub; loiters then picks a next behaviour
#     wander  - roams to a random spot (bored)
#     follow  - seeks the cursor; on reach -> dash or play
#     play    - bats at the cursor, staying engaged
#     zoomies - a ~10s dart-fest
#     retreat - ambles to a corner then naps
#     jump/airborne - a hop under gravity
#     carry   - being held under the cursor
#   CLIPS are dumb one-shots: play one anim for N ticks, then auto-return. The
#   generic "clip" state backs appear/dash/startle/grab/drop/sleep/<action>. A
#   clip may set `clip_lock` so mood/jiggle can't interrupt it mid-play.

const TURN_PAUSE := 6      # ticks of hesitation before flipping direction
const SETTLE_BIAS := 1.8   # after moving, likelier to rest next
const ROUSE_BIAS := 1.3    # after resting, likelier to get moving next
const RETREAT_NAP_LOOPS := 12  # a corner nap is a good long one
const PLAY_LOOPS := 3      # bat-loops before the pet is satisfied
const PLAY_LEASH := 60     # cursor drifts this far -> chase it again
const REACH_DIST := 30     # "arrived at the cursor" threshold

# Behaviours the LLM may push the pet into (must match daemon agent.py DRIVABLE).
const DRIVABLE := ["idle", "wander", "follow", "jump", "zoomies"]

var cfg: PetConfig
var loop_lens: Dictionary  # animation name -> ticks for one full loop
var play_available := false # does a real "play" animation resolve?

var x := 0
var y := 0
var vel_x := 0
var vel_y := 0
var facing_left := false
var state := "idle"
var anim := "idle"
var timer := 0
var target_x := 0
var screen_w := 0
var screen_h := 0
var frame_w := 0
var frame_h := 0
var mood := 0  # 0 neutral, 1 alert, 2 tired

# generic clip state: play `clip_anim` for `clip_left` ticks, then become
# `clip_next`. `clip_lock` = mood/jiggle can't interrupt it mid-play.
var clip_anim := "idle"
var clip_left := 0
var clip_next := "idle"
var clip_lock := false

var turn_pause := 0
var last_active := false  # was the last chosen behaviour a mover?

var ticks := 0                    # global tick counter for timers
var retreat_at := -1              # tick to retreat at (-1 = off)
var last_zoomies_tick := -1000000 # for the zoomies cooldown
var zoomies_end := 0
var dart_target := 0
var play_loops_left := 0

func setup(sw: int, sh: int, fw: int, fh: int, config: PetConfig, lens: Dictionary, has_play := false) -> void:
	screen_w = sw
	screen_h = sh
	frame_w = fw
	frame_h = fh
	cfg = config
	loop_lens = lens
	play_available = has_play
	x = sw / 2
	y = sh - fh
	retreat_at = _sec_ticks(cfg.retreat_interval_sec) if cfg.retreat_interval_sec > 0.0 else -1
	_clip("appear", _ticks("appear"), "idle")

func _ticks(name: String) -> int:
	var n: int = loop_lens.get(name, 0)
	return n if n > 0 else 60

func _sec_ticks(s: float) -> int:
	return int(s * 60.0)

func _clip(a: String, t: int, nxt: String = "idle", lock := false) -> void:
	state = "clip"
	clip_anim = a
	clip_left = maxi(1, t)
	clip_next = nxt
	clip_lock = lock

func update(cursor_x: int, cursor_y: int) -> void:
	var ground := screen_h - frame_h
	ticks += 1

	match state:
		"clip":
			anim = clip_anim
			vel_x = 0
			clip_left -= 1
			if clip_left <= 0:
				state = clip_next
				timer = 0
		"idle":
			anim = "idle"
			vel_x = 0
			timer += 1
			if _retreat_due():
				_start_retreat()
			elif timer > _idle_delay():
				timer = 0
				_decide()
		"wander":
			anim = "wander"
			var dx := target_x - x
			if abs(dx) < 4:
				state = "idle"
				vel_x = 0
				timer = 0
			elif dx > 0:
				_move(false, cfg.wander_speed)
			else:
				_move(true, cfg.wander_speed)
		"retreat":
			anim = "wander"
			var dx := target_x - x
			if abs(dx) < 4:
				_clip("sleep", RETREAT_NAP_LOOPS * _ticks("sleep"), "idle")
			elif dx > 0:
				_move(false, cfg.wander_speed)
			else:
				_move(true, cfg.wander_speed)
		"follow":
			anim = "follow"
			if cfg.follow_cursor and cursor_x >= 0 and cursor_y >= 0:
				var dx := cursor_x - x - frame_w / 2
				if abs(dx) < REACH_DIST:
					# a ground pet can't reach a cursor held a body-height+ above
					# its head — it stops underneath and sulks instead.
					if cfg.gravity and cursor_y < y - frame_h:
						_clip("sad", _ticks("sad"), "idle")
					elif cfg.follow_reach == "play" and play_available:
						state = "play"
						play_loops_left = PLAY_LOOPS
						timer = 0
					else:
						_clip("dash", _ticks("dash"), "idle", true)
				elif dx > 0:
					_move(false, cfg.follow_speed)
				else:
					_move(true, cfg.follow_speed)
				timer += 1
				if timer > 300:
					state = "idle"
					timer = 0
			else:
				state = "idle"
		"play":
			anim = "play"
			vel_x = 0
			if cursor_x < 0:
				state = "idle"
				timer = 0
			else:
				var pdx := cursor_x - x - frame_w / 2
				facing_left = pdx < 0
				if abs(pdx) > PLAY_LEASH:
					state = "follow"
					timer = 0
				else:
					timer += 1
					if timer >= _ticks("play"):
						timer = 0
						play_loops_left -= 1
						if play_loops_left <= 0:
							state = "idle"
		"zoomies":
			anim = "dash"
			if ticks >= zoomies_end:
				state = "idle"
				timer = 0
			else:
				var sp := roundi(cfg.follow_speed * cfg.zoomies_speed_mult)
				var dx := dart_target - x
				if abs(dx) < 8:
					dart_target = _new_dart()
					dx = dart_target - x
				if dx > 0:
					vel_x = sp
					facing_left = false
				else:
					vel_x = -sp
					facing_left = true
		"jump":
			anim = "jump"
			vel_y = -8
			state = "airborne"
		"airborne":
			anim = "jump"
		"carry":
			anim = "carry"
			vel_x = 0
			vel_y = 0

	x += vel_x

	# gravity — a carried pet hangs from the cursor; a floaty creature never falls
	if cfg.gravity and not held():
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
	var base := cfg.idle_delay
	match mood:
		1: return int(base * 0.5)
		2: return int(base * 1.7)
		_: return base

func _retreat_due() -> bool:
	return retreat_at >= 0 and ticks >= retreat_at

func _start_retreat() -> void:
	retreat_at = ticks + _sec_ticks(cfg.retreat_interval_sec)
	# nearest bottom corner
	target_x = 0 if x < (screen_w - frame_w) / 2 else screen_w - frame_w
	state = "retreat"

func _start_zoomies() -> void:
	last_zoomies_tick = ticks
	zoomies_end = ticks + _sec_ticks(cfg.zoomies_duration_sec)
	dart_target = _new_dart()
	state = "zoomies"

func _new_dart() -> int:
	var dist := 150 + randi() % 350
	var t := x + dist if randi() % 2 == 0 else x - dist
	return clampi(t, 0, screen_w - frame_w)

# _decide builds the weighted candidate set (core behaviours + config actions),
# applies the current mood's multipliers, and rolls one.
func _decide() -> void:
	var names: Array = []
	var weights: Array = []
	var actions_by_name := {}

	names.append("idle")
	weights.append(float(cfg.idle_weight))
	names.append("wander")
	weights.append(float(cfg.wander_weight))
	if cfg.follow_cursor:
		names.append("follow")
		weights.append(float(cfg.follow_weight))
	if cfg.gravity:
		names.append("jump")
		weights.append(float(cfg.jump_weight))
	if cfg.zoomies_weight > 0 and ticks - last_zoomies_tick >= _sec_ticks(cfg.zoomies_cooldown_sec):
		names.append("zoomies")
		weights.append(float(cfg.zoomies_weight))
	for a in cfg.actions:
		var nm := String(a.get("name", "action"))
		names.append(nm)
		weights.append(float(a.get("weight", 1)))
		actions_by_name[nm] = a

	var mm := _mood_mult()
	var total := 0.0
	for i in names.size():
		weights[i] *= float(mm.get(names[i], 1.0))
		weights[i] *= _chain_factor(names[i], actions_by_name)
		if weights[i] < 0.0:
			weights[i] = 0.0
		total += weights[i]

	if total <= 0.0:
		state = "idle"
		return

	var roll := randf() * total
	var acc := 0.0
	for i in names.size():
		acc += weights[i]
		if roll < acc:
			_pick(names[i], actions_by_name.get(names[i], null))
			return
	state = "idle"

func _pick(nm: String, action) -> void:
	last_active = nm in ["wander", "follow", "jump", "zoomies"]
	if action != null:
		var a_anim := String(action.get("anim", "idle"))
		var loops := int(action.get("loops", 1))
		_clip(a_anim, loops * _ticks(a_anim), "idle")
	elif nm == "wander":
		target_x = randi() % maxi(1, screen_w - frame_w)
		state = "wander"
	elif nm == "follow":
		state = "follow"
		timer = 0
	elif nm == "jump":
		state = "jump"
		timer = 0
	elif nm == "zoomies":
		_start_zoomies()
	else:
		state = "idle"

# A pet that just moved tends to settle (rest); a pet that just rested tends to
# get moving. This makes behaviour read as intentional rhythm, not coin flips.
func _chain_factor(nm: String, actions_by_name: Dictionary) -> float:
	var restful := nm == "idle" or actions_by_name.has(nm)
	if last_active and restful:
		return SETTLE_BIAS
	if not last_active and not restful:
		return ROUSE_BIAS
	return 1.0

# Move toward a direction, hesitating briefly when it has to turn around so it
# doesn't twitch back and forth on the spot.
func _move(want_left: bool, speed: int) -> void:
	if want_left != facing_left:
		facing_left = want_left
		turn_pause = TURN_PAUSE
	if turn_pause > 0:
		turn_pause -= 1
		vel_x = 0
	else:
		vel_x = -speed if want_left else speed

func _mood_mult() -> Dictionary:
	match mood:
		1: return cfg.alert_weights
		2: return cfg.tired_weights
		_: return {}

func set_mood(m: int) -> void:
	if m == mood:
		return
	mood = m
	if held() or not _interruptible():
		return
	var r := ""
	if m == 2:
		r = cfg.tired_reaction
	elif m == 1:
		r = cfg.alert_reaction
	if r != "":
		_clip(r, _ticks(r), "idle")

func _interruptible() -> bool:
	if held():
		return false
	if state in ["follow", "play", "zoomies", "retreat"]:
		return false
	if state == "clip" and clip_lock:
		return false
	return true

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
	if not cfg.gravity:
		return true
	return y >= screen_h - frame_h

func held() -> bool:
	return state == "carry" or (state == "clip" and clip_next == "carry")

func scare() -> void:
	if held():
		return
	if state == "clip" and clip_anim == "startle":
		return
	_clip("startle", _ticks("startle"), "idle", true)

# Right-click: a gentle pet on the head. Low-priority (unlocked) so a mood flip
# can still take over. Falls back to idle until a "pet" anim is authored.
func pet_touch() -> void:
	if held():
		return
	if state == "clip" and clip_anim == "pet":
		return
	_clip("pet", _ticks("pet"), "idle")

# Externally-chosen behaviour (from the LLM). Whitelisted + guarded so a bad
# model turn can't wedge the pet; returns false (no-op) if it couldn't apply.
func drive_state(name: String) -> bool:
	if held() or not _interruptible():
		return false
	match name:
		"idle":
			state = "idle"
			timer = 0
		"wander":
			target_x = randi() % maxi(1, screen_w - frame_w)
			state = "wander"
		"follow":
			if not cfg.follow_cursor:
				return false
			state = "follow"
			timer = 0
		"jump":
			if not (cfg.gravity and grounded()):
				return false
			state = "jump"
			timer = 0
		"zoomies":
			_start_zoomies()
		_:
			return false
	return true

func do_jump() -> void:
	if cfg.gravity and grounded():
		state = "jump"
		timer = 0

func hold(px: int, py: int) -> void:
	if not held():
		_clip("grab", _ticks("grab"), "carry")
	x = px - frame_w / 2
	y = py - frame_h / 2

func release() -> void:
	if held():
		_clip("drop", _ticks("drop"), "idle")
