extends Node2D

# Fullscreen, transparent, always-on-top overlay. The window never moves — the
# cat is drawn at an offset inside it (the same stationary-surface idea we landed
# on for GTK, but here it's trivial). Mouse passthrough is clipped to the cat's
# rect so the rest of the desktop stays click-through.

const MANIFEST_PATH := "res://assets/cat-grey.json"
const SPRITE_PATH := "res://assets/cat-grey.png"
const SCALE := 5

# Canonical fallback chain (port of sprite.Resolve): only "idle" is required,
# everything else degrades toward it so BYO sheets just work.
const FALLBACK := {
	"idle2": "idle", "walk": "idle", "walk2": "walk", "run": "walk",
	"pounce": "paw", "paw": "idle", "sit": "idle", "sit2": "sit",
	"loaf": "sit", "sleep": "loaf", "clean": "idle", "clean2": "clean",
	"stretch": "idle", "yawn": "idle", "meow": "idle", "roll": "idle",
	"jump": "idle", "scared": "idle", "spawn": "idle", "pickup": "idle",
	"putdown": "idle", "held": "sit", "held2": "held",
}

var pet: PetBrain
var sprite: AnimatedSprite2D
var manifest: Dictionary
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

	manifest = _load_manifest(MANIFEST_PATH)
	frame_w = int(manifest.frameSize[0])
	frame_h = int(manifest.frameSize[1])
	scaled_w = frame_w * SCALE
	scaled_h = frame_h * SCALE

	sprite = _find_sprite()
	if sprite == null:
		push_error("main.gd: no AnimatedSprite2D child found")
		return
	sprite.sprite_frames = _build_frames(load(SPRITE_PATH), manifest)
	sprite.scale = Vector2(SCALE, SCALE)

	# Bound the cat to the usable area (excludes panels/taskbars) so it doesn't
	# walk under a taskbar that's rendered on a compositor layer above us.
	var usable := DisplayServer.screen_get_usable_rect()
	pet = PetBrain.new()
	pet.setup(usable.end.x, usable.end.y, scaled_w, scaled_h, _loop_lengths(manifest))

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

func _resolve(name: String) -> String:
	var n := name
	for _i in FALLBACK.size() + 1:
		if manifest.states.has(n):
			return n
		if not FALLBACK.has(n):
			break
		n = FALLBACK[n]
	return "idle"

# --- sprite setup ---

func _load_manifest(path: String) -> Dictionary:
	var f := FileAccess.open(path, FileAccess.READ)
	return JSON.parse_string(f.get_as_text())

func _build_frames(tex: Texture2D, m: Dictionary) -> SpriteFrames:
	var fw := int(m.frameSize[0])
	var fh := int(m.frameSize[1])
	var sf := SpriteFrames.new()
	sf.remove_animation("default")
	for name in m.states:
		var a: Dictionary = m.states[name]
		sf.add_animation(name)
		sf.set_animation_speed(name, a.fps)
		sf.set_animation_loop(name, true)
		for i in int(a.frames):
			var at := AtlasTexture.new()
			at.atlas = tex
			at.region = Rect2(i * fw, int(a.row) * fh, fw, fh)
			sf.add_frame(name, at)
	return sf

func _loop_lengths(m: Dictionary) -> Dictionary:
	var out := {}
	for name in m.states:
		var a: Dictionary = m.states[name]
		var fps := int(a.fps)
		out[name] = int(a.frames) * (60 / fps) if fps > 0 else 0
	return out

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
	var name := d.get_next()
	while name != "":
		if name.is_valid_int():
			var comm := _read_text("/proc/%s/comm" % name).strip_edges()
			if comm in WATCH:
				return true
		name = d.get_next()
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
	var name := d.get_next()
	while name != "":
		if name.begins_with(prefix):
			out.append("%s/%s" % [dir, name])
		name = d.get_next()
	return out

func _read_text(path: String) -> String:
	# /proc and /sys report bogus file lengths, so get_as_text() asserts and
	# spams. These are all single-value files — read one line instead.
	var f := FileAccess.open(path, FileAccess.READ)
	if f == null:
		return ""
	return f.get_line()
