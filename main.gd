extends Node2D

# Fullscreen, transparent, always-on-top overlay. The window never moves — the
# cat is drawn at an offset inside it (the same stationary-surface idea we landed
# on for GTK, but here it's trivial). Mouse passthrough is clipped to the cat's
# rect so the rest of the desktop stays click-through.

const DEFAULT_CONFIG := "res://cat.tres"

# Per-creature data. Set it in the Inspector to swap creatures; falls back to the
# bundled cat if unset. See docs/behavior-model.md and pet_config.gd.
@export var config: PetConfig

# Engine behaviours the state machine can emit. The config's `aliases` map these
# onto authored animations; FALLBACK degrades anything unmatched toward idle
# (the one animation every creature must have).
const ENGINE_STATES := ["appear", "idle", "wander", "follow", "dash", "jump",
	"grab", "carry", "drop", "startle"]
const FALLBACK := {
	"idle2": "idle", "wander": "idle", "follow": "wander", "dash": "follow",
	"appear": "idle", "jump": "idle", "grab": "idle", "carry": "idle",
	"drop": "idle", "startle": "idle",
}

var pet: PetBrain
var sprite: AnimatedSprite2D
var frame_w := 32
var frame_h := 32
var scaled_w := 160
var scaled_h := 160
var last_anim := ""

var mouse_pos := Vector2i(-1, -1)
var mouse_btn := 0        # 1 = left, 2 = right
var grabbing := false
var grab_off := Vector2i.ZERO

var mood_timer := 0

func _ready() -> void:
	_setup_window()

	if config == null:
		config = load(DEFAULT_CONFIG)
	if config == null:
		push_error("main.gd: no PetConfig (set one in the Inspector or provide %s)" % DEFAULT_CONFIG)
		return

	sprite = _find_sprite()
	if sprite == null:
		push_error("main.gd: no AnimatedSprite2D child found")
		return
	if config.sprite_frames != null:
		sprite.sprite_frames = config.sprite_frames
	if sprite.sprite_frames == null:
		push_error("main.gd: no SpriteFrames (author on the node or set config.sprite_frames)")
		return
	sprite.scale = Vector2(config.scale, config.scale)

	# frame size from the first authored frame (all frames are the same size)
	var tex := sprite.sprite_frames.get_frame_texture(_resolve("idle"), 0)
	if tex != null:
		frame_w = tex.get_width()
		frame_h = tex.get_height()
	scaled_w = frame_w * config.scale
	scaled_h = frame_h * config.scale

	# loop lengths for every authored animation + every engine behaviour
	var lens := {}
	for a in sprite.sprite_frames.get_animation_names():
		lens[a] = _anim_ticks(a)
	for s in ENGINE_STATES:
		lens[s] = _anim_ticks(_resolve(s))

	# Bound the pet to the usable area (excludes panels/taskbars) so it doesn't
	# walk under a taskbar that's rendered on a compositor layer above us.
	var usable := DisplayServer.screen_get_usable_rect()
	pet = PetBrain.new()
	pet.setup(usable.end.x, usable.end.y, scaled_w, scaled_h, config, lens)

func _find_sprite() -> AnimatedSprite2D:
	for c in get_children():
		if c is AnimatedSprite2D:
			return c
	return null

func _setup_window() -> void:
	get_viewport().transparent_bg = true
	var w := get_window()
	w.transparent = true
	w.borderless = true
	w.always_on_top = true
	var scr := DisplayServer.screen_get_size()
	w.size = scr
	w.position = Vector2i.ZERO

func _physics_process(_dt: float) -> void:
	# ~every 2s, sample the machine's mood
	mood_timer += 1
	if mood_timer >= 120:
		mood_timer = 0
		pet.set_mood(_read_mood())

	var cx := -1
	var cy := -1
	if mouse_pos.x >= 0 and (_over_cat(mouse_pos) or mouse_btn != 0):
		cx = mouse_pos.x
		cy = mouse_pos.y

	# drag / scare — window is stationary so mouse_pos is absolute, no feedback
	if mouse_btn == 1:
		if not grabbing and _over_cat(mouse_pos):
			grabbing = true
			grab_off = Vector2i(mouse_pos.x - pet.x, mouse_pos.y - pet.y)
		if grabbing:
			pet.hold(mouse_pos.x - grab_off.x + scaled_w / 2, mouse_pos.y - grab_off.y + scaled_h / 2)
	elif pet.held():
		grabbing = false
		pet.release()
	elif mouse_btn == 2 and _over_cat(mouse_pos):
		grabbing = false
		pet.scare()
	else:
		grabbing = false

	pet.update(cx, cy)

	# AnimatedSprite2D is centered; pet.x/y is the top-left, like the Go version
	sprite.position = Vector2(pet.x + scaled_w / 2.0, pet.y + scaled_h / 2.0)
	sprite.flip_h = pet.facing_left

	var a := _resolve(pet.anim)
	if a != last_anim:
		last_anim = a
		sprite.play(a)

	_update_passthrough()

func _input(event: InputEvent) -> void:
	if event is InputEventMouseMotion:
		mouse_pos = Vector2i(event.position)
	elif event is InputEventMouseButton:
		mouse_pos = Vector2i(event.position)
		if event.pressed:
			if event.button_index == MOUSE_BUTTON_LEFT:
				mouse_btn = 1
			elif event.button_index == MOUSE_BUTTON_RIGHT:
				mouse_btn = 2
		elif event.button_index == mouse_btn_index(mouse_btn):
			mouse_btn = 0

func mouse_btn_index(b: int) -> int:
	return MOUSE_BUTTON_LEFT if b == 1 else MOUSE_BUTTON_RIGHT

func _over_cat(p: Vector2i) -> bool:
	return p.x >= pet.x and p.x < pet.x + scaled_w and p.y >= pet.y and p.y < pet.y + scaled_h

func _update_passthrough() -> void:
	var x := pet.x
	var y := pet.y
	var poly := PackedVector2Array([
		Vector2(x, y),
		Vector2(x + scaled_w, y),
		Vector2(x + scaled_w, y + scaled_h),
		Vector2(x, y + scaled_h),
	])
	DisplayServer.window_set_mouse_passthrough(poly)

# Map a canonical state name onto an animation the authored SpriteFrames has:
# direct name, then ALIAS, then walk the FALLBACK chain toward idle.
func _resolve(state_name: String) -> String:
	var sf := sprite.sprite_frames
	var n := state_name
	for _i in FALLBACK.size() + 2:
		if sf.has_animation(n):
			return n
		if config.aliases.has(n) and sf.has_animation(config.aliases[n]):
			return config.aliases[n]
		if FALLBACK.has(n):
			n = FALLBACK[n]
		else:
			break
	if sf.has_animation("idle"):
		return "idle"
	var names := sf.get_animation_names()
	return names[0] if names.size() > 0 else "idle"

# One full animation loop in 60Hz physics ticks (matches _physics_process rate).
func _anim_ticks(anim: String) -> int:
	var sf := sprite.sprite_frames
	if not sf.has_animation(anim):
		return 0
	var fps := sf.get_animation_speed(anim)
	if fps <= 0:
		return 0
	return int(sf.get_frame_count(anim) * (60.0 / fps))

# --- sysmon: mood from system state (port of internal/sysmon) ---

const WATCH := ["go", "gcc", "cc1", "clang", "rustc", "cargo", "node", "npm",
	"webpack", "tsc", "make", "cmake", "ninja", "docker", "gradle", "mvn",
	"python", "ld"]

func _read_mood() -> int:
	var pct := _battery_pct()
	var charging := _charging()
	if (pct >= 0 and pct < 15 and not charging) or _hottest_zone() >= 85:
		return 2 # tired
	if _building():
		return 1 # alert
	return 0

func _building() -> bool:
	var d := DirAccess.open("/proc")
	if d == null:
		return false
	d.list_dir_begin()
	var entry := d.get_next()
	while entry != "":
		if entry.is_valid_int():
			var comm := _read_text("/proc/%s/comm" % entry).strip_edges()
			if comm in WATCH:
				return true
		entry = d.get_next()
	return false

func _battery_pct() -> int:
	for b in _glob("/sys/class/power_supply", "BAT"):
		var s := _read_text("%s/capacity" % b).strip_edges()
		if s.is_valid_int():
			return int(s)
	return -1

func _charging() -> bool:
	for b in _glob("/sys/class/power_supply", "BAT"):
		if _read_text("%s/status" % b).strip_edges() != "Discharging":
			return true
	return false

func _hottest_zone() -> int:
	var hot := 0
	for z in _glob("/sys/class/thermal", "thermal_zone"):
		var s := _read_text("%s/temp" % z).strip_edges()
		if s.is_valid_int():
			var c := int(s) / 1000
			if c > hot:
				hot = c
	return hot

func _glob(dir: String, prefix: String) -> Array:
	var out := []
	var d := DirAccess.open(dir)
	if d == null:
		return out
	d.list_dir_begin()
	var entry := d.get_next()
	while entry != "":
		if entry.begins_with(prefix):
			out.append("%s/%s" % [dir, entry])
		entry = d.get_next()
	return out

func _read_text(path: String) -> String:
	# /proc and /sys report bogus file lengths, so get_as_text() asserts and
	# spams. These are all single-value files — read one line instead.
	var f := FileAccess.open(path, FileAccess.READ)
	if f == null:
		return ""
	return f.get_line()
