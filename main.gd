extends Node2D

# Fullscreen, transparent, always-on-top overlay. The window never moves — the
# cat is drawn at an offset inside it (the same stationary-surface idea we landed
# on for GTK, but here it's trivial). Mouse passthrough is clipped to the cat's
# rect so the rest of the desktop stays click-through.

const DEFAULT_CONFIG := "res://playful_cat.tres"

# Per-creature data. Set it in the Inspector to swap creatures; falls back to the
# bundled cat if unset. See docs/behavior-model.md and pet_config.gd.
@export var config: PetConfig

# Engine behaviours the state machine can emit. The config's `aliases` map these
# onto authored animations; FALLBACK degrades anything unmatched toward idle
# (the one animation every creature must have).
const ENGINE_STATES := ["appear", "idle", "wander", "follow", "dash", "jump",
	"grab", "carry", "drop", "startle", "sleep", "play", "sad", "pet"]
const FALLBACK := {
	"idle2": "idle", "wander": "idle", "follow": "wander", "dash": "follow",
	"appear": "idle", "jump": "idle", "grab": "idle", "carry": "idle",
	"drop": "idle", "startle": "idle", "sleep": "idle", "play": "idle",
	"sad": "idle", "pet": "idle",
}

var pet: PetBrain
var sprite: AnimatedSprite2D
var frame_w := 32
var frame_h := 32
var scaled_w := 160
var scaled_h := 160
# The pet's visible *body* — the opaque pixels within the padded frame. All
# movement/clamping/clicking uses this; only drawing uses the full frame.
var body_w := 160
var body_h := 160
var body_off := Vector2i.ZERO  # body's top-left within the frame (scaled px)
var last_anim := ""

var mouse_pos := Vector2i(-1, -1)
var mouse_btn := 0        # 1 = left, 2 = right
var grabbing := false
var grab_off := Vector2i.ZERO

var mood_timer := 0
var debug_layer: CanvasLayer = null
var debug_label: Label = null

const BUBBLE_TICKS := 240   # ~4s at 60fps
var bubble_layer: CanvasLayer = null
var bubble_panel: PanelContainer = null
var bubble_label: Label = null
var bubble_left := 0
var last_say := ""          # de-dup: never show the same line twice in a row

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

	# The art rarely fills the whole frame — find its opaque bounds so the pet is
	# positioned/clamped/clicked by its visible body, not the transparent padding
	# (otherwise it "stops" a frame-of-padding short of the screen edge).
	body_w = scaled_w
	body_h = scaled_h
	body_off = Vector2i.ZERO
	if tex != null:
		var img := tex.get_image()
		if img != null:
			var used := img.get_used_rect()
			if used.size.x > 0 and used.size.y > 0:
				body_off = Vector2i(used.position.x * config.scale, used.position.y * config.scale)
				body_w = used.size.x * config.scale
				body_h = used.size.y * config.scale

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
	pet.setup(usable.end.x, usable.end.y, body_w, body_h, config, lens, _resolve("play") != "idle")

	# DEBUG=true (env var or .env) reveals the authored state panel (top-right).
	debug_layer = $Debug
	debug_label = $Debug/State
	debug_layer.visible = _debug_enabled()
	_make_bubble()

func _make_bubble() -> void:
	bubble_layer = CanvasLayer.new()
	add_child(bubble_layer)
	bubble_panel = PanelContainer.new()
	var sb := StyleBoxFlat.new()
	sb.bg_color = Color(0.1, 0.1, 0.12, 0.92)
	sb.set_corner_radius_all(10)
	sb.set_content_margin_all(8)
	bubble_panel.add_theme_stylebox_override("panel", sb)
	bubble_layer.add_child(bubble_panel)
	bubble_label = Label.new()
	bubble_label.add_theme_color_override("font_color", Color.WHITE)
	bubble_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	bubble_label.custom_minimum_size = Vector2(0, 0)
	bubble_label.size = Vector2(220, 0)
	bubble_panel.add_child(bubble_label)
	bubble_layer.visible = false
	# ponytail: PanelContainer bubble; swap in the Pooklea nine-patch art later
	# once we've confirmed it sizes to arbitrary text.

func _show_bubble(text: String) -> void:
	if text == "":
		return
	bubble_label.text = text
	bubble_left = BUBBLE_TICKS
	bubble_layer.visible = true

func _find_sprite() -> AnimatedSprite2D:
	for c in get_children():
		if c is AnimatedSprite2D:
			return c
	return null

# DEBUG=true (env var or a .env line) shows a live state readout, top-right.
func _debug_enabled() -> bool:
	if OS.get_environment("DEBUG").to_lower() in ["true", "1", "yes"]:
		return true
	var f := FileAccess.open("res://.env", FileAccess.READ)
	if f == null:
		return false
	while not f.eof_reached():
		var line := f.get_line().strip_edges()
		var idx := line.find("=")
		if idx > 0 and line.substr(0, idx).strip_edges() == "DEBUG":
			return line.substr(idx + 1).strip_edges().to_lower() in ["true", "1", "yes"]
	return false

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

	# Global cursor: mouse_get_position() tracks the pointer across the whole
	# screen (fullscreen overlay), so the pet can follow it anywhere. Clicks still
	# pass through except on the cat, so button state comes from _input.
	var gpos := DisplayServer.mouse_get_position()
	var cx := gpos.x
	var cy := gpos.y

	# drag / scare — button state is from clicks on the cat; position is global.
	if mouse_btn == 1:
		if not grabbing and _over_cat(gpos):
			grabbing = true
			grab_off = Vector2i(gpos.x - pet.x, gpos.y - pet.y)
		if grabbing:
			pet.hold(gpos.x - grab_off.x + body_w / 2, gpos.y - grab_off.y + body_h / 2)
	elif pet.held():
		grabbing = false
		pet.release()
	elif mouse_btn == 2 and _over_cat(gpos):
		grabbing = false
		pet.pet_touch()
	else:
		grabbing = false

	pet.update(cx, cy)

	# pet.x/y is the body's top-left; place the (centered) frame so the body lands
	# there, accounting for the transparent padding offset inside the frame.
	sprite.position = Vector2(
		pet.x - body_off.x + scaled_w / 2.0,
		pet.y - body_off.y + scaled_h / 2.0)
	sprite.flip_h = pet.facing_left

	var a := _resolve(pet.anim)
	if a != last_anim:
		last_anim = a
		sprite.play(a)

	_update_passthrough()

	if bubble_layer.visible:
		bubble_left -= 1
		if bubble_left <= 0:
			bubble_layer.visible = false
		else:
			var sz := bubble_panel.size
			var bx: float = pet.x + body_w / 2.0 - sz.x / 2.0
			var by: float = pet.y - sz.y - 8.0
			var scr := DisplayServer.screen_get_size()
			bx = clampf(bx, 0.0, float(scr.x) - sz.x)
			by = clampf(by, 0.0, float(scr.y) - sz.y)
			bubble_panel.position = Vector2(bx, by)

	if debug_layer.visible:
		var s := pet.state
		if s == "clip":
			s = "clip:" + pet.clip_anim
		var moods := ["neutral", "alert", "tired"]
		debug_label.text = "state: %s\nanim:  %s\nmood:  %s\npos:   %d,%d" % [
			s, _resolve(pet.anim), moods[pet.mood], pet.x, pet.y]

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
	return p.x >= pet.x and p.x < pet.x + body_w and p.y >= pet.y and p.y < pet.y + body_h

func _update_passthrough() -> void:
	var x := pet.x
	var y := pet.y
	var poly := PackedVector2Array([
		Vector2(x, y),
		Vector2(x + body_w, y),
		Vector2(x + body_w, y + body_h),
		Vector2(x, y + body_h),
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
