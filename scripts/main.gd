extends Node2D

# Fullscreen, transparent, always-on-top overlay. The window never moves — the
# cat is drawn at an offset inside it (the same stationary-surface idea we landed
# on for GTK, but here it's trivial). Mouse passthrough is clipped to the cat's
# rect so the rest of the desktop stays click-through.

const DEFAULT_CONFIG := "res://creatures/playful_cat.tres"

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
var _last_dbg_key := ""      # dedupe the tagged stdout debug line (for the TUI)
var _last_dbg_frame := 0

# Speech goes to the draggable bubble window (see scripts/bubble_window.gd); the
# roaming cat no longer bubbles above itself. We just forward lines to it.
var bubble: BubbleWindow = null
var last_say := ""          # de-dup: never show the same line twice in a row

const COMMENT_COOLDOWN_TICKS := 1200   # ~20s at 60fps — min gap between comments
const HEARTBEAT_TICKS := 240          # ~90s — pipe up on its own even with no mood change
var commenter: Commenter = null
var frame := 0
var last_comment_frame := -100000
var last_heartbeat_frame := 300 - HEARTBEAT_TICKS   # first heartbeat ~5s after launch

const DAEMON_URL := "http://127.0.0.1:8787/tick"
var http: HTTPRequest = null
var in_flight := false
var pending_mood := 0

# agent-reactivity (opt-in): GOOB_HSM=1 makes the pet react to Claude Code.
var agent_hsm: LimboHSM = null
var agent_poller: AgentPoller = null

var sysmon: SysMon = null

func _mood_key(m: int) -> String:
	match m:
		1: return "alert"
		2: return "tired"
		_: return "neutral"

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
	bubble = get_node_or_null("Bubble") as BubbleWindow
	commenter = Commenter.new()

	http = HTTPRequest.new()
	http.timeout = 6.0
	add_child(http)
	http.request_completed.connect(_on_tick_completed)

	sysmon = SysMon.new()
	_setup_agent_reactivity()

func _setup_agent_reactivity() -> void:
	var flag := OS.get_environment("GOOB_HSM")
	if flag == "" or flag == "0" or flag.to_lower() == "false":
		return
	agent_hsm = AgentHsm.build(self, pet)
	agent_poller = AgentPoller.new()
	add_child(agent_poller)
	agent_poller.setup(agent_hsm)

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
	w.mode = Window.MODE_WINDOWED   # never fullscreen — it kills transparency/passthrough
	w.transparent = true
	w.borderless = true
	w.always_on_top = true
	_apply_screen_size()
	# Re-apply after one frame: when launched detached (e.g. from the TUI) the
	# DisplayServer isn't always ready at _ready() time, so the first size/position
	# silently doesn't stick and the window opens tiny in the top-left corner.
	get_tree().process_frame.connect(_apply_screen_size, CONNECT_ONE_SHOT)

func _apply_screen_size() -> void:
	var scr := DisplayServer.screen_get_size()
	if scr.x <= 0 or scr.y <= 0:
		return                       # display not ready — leave it for the next attempt
	var w := get_window()
	w.size = scr
	w.position = Vector2i.ZERO

func _physics_process(dt: float) -> void:
	frame += 1
	# ~every 2s, sample the machine's mood; comment on transitions.
	mood_timer += 1
	if mood_timer >= 120:
		mood_timer = 0
		var m := sysmon.poll()
		if m != pet.mood:
			_maybe_comment(m, "mood_changed")
		pet.set_mood(m)
	# heartbeat: comment on its own periodically, even without a mood change
	if frame - last_heartbeat_frame >= HEARTBEAT_TICKS:
		last_heartbeat_frame = frame
		_maybe_comment(pet.mood, "heartbeat")

	# Global cursor: mouse_get_position() tracks the pointer across the whole
	# screen (fullscreen overlay), so the pet can follow it anywhere. Clicks still
	# pass through except on the cat, so button state comes from _input.
	var gpos := DisplayServer.mouse_get_position()
	var cx := gpos.x
	var cy := gpos.y

	# drag / scare — button state is from clicks on the cat; position is global.
	if mouse_btn == 1:
		# Cat grab yields to the face: a press over the floating face drags the
		# face, not the cat (they can overlap when the cat is carried up to it).
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

	# Hand cursor over the cat; arrow elsewhere.
	if _over_cat(gpos):
		Input.set_default_cursor_shape(Input.CURSOR_POINTING_HAND)
	else:
		Input.set_default_cursor_shape(Input.CURSOR_ARROW)

	# Compute the readout every frame. The on-screen overlay is gated by DEBUG,
	# but the tagged stdout line is ALWAYS emitted so the TUI can mirror state
	# regardless of DEBUG. Throttled: on state/anim/mood change, plus a 0.5s
	# refresh for position — never per frame (that would flood while moving).
	var s := pet.state
	if s == "clip":
		s = "clip:" + pet.clip_anim
	var moods := ["neutral", "alert", "tired"]
	var anim_name := _resolve(pet.anim)
	var mood_name: String = moods[pet.mood]
	if debug_layer.visible:
		debug_label.text = "state: %s\nanim:  %s\nmood:  %s\npos:   %d,%d" % [
			s, anim_name, mood_name, pet.x, pet.y]
	var key := "%s|%s|%s" % [s, anim_name, mood_name]
	if key != _last_dbg_key or frame - _last_dbg_frame >= 30:
		_last_dbg_key = key
		_last_dbg_frame = frame
		print("goob-dbg: state=%s anim=%s mood=%s pos=%d,%d" % [
			s, anim_name, mood_name, pet.x, pet.y])

	if agent_hsm != null:
		agent_hsm.update(dt)
		# Must be wall-clock (matches hooks/goob_hook.py's time.time()), not engine
		# uptime — otherwise staleness never fires (huge negative delta) and the
		# pet can get stuck in Reacting forever if the agent dies without a
		# Stop/SessionEnd event.
		agent_poller.poll(Time.get_unix_time_from_system())

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
	# One passthrough polygon, but the pointer is only ever over one interactive
	# thing — so track it: the bubble rect while hovering the bubble, else the cat's
	# body. Everything outside stays click-through to the desktop.
	if bubble != null and bubble.over_bubble():
		var r := bubble.panel_rect()
		DisplayServer.window_set_mouse_passthrough(PackedVector2Array([
			r.position,
			Vector2(r.end.x, r.position.y),
			r.end,
			Vector2(r.position.x, r.end.y),
		]))
		return
	var x := pet.x
	var y := pet.y
	DisplayServer.window_set_mouse_passthrough(PackedVector2Array([
		Vector2(x, y),
		Vector2(x + body_w, y),
		Vector2(x + body_w, y + body_h),
		Vector2(x, y + body_h),
	]))

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

# Fire a comment (mood edge or heartbeat), gated by the in-flight guard and the
# cooldown. A blocked attempt is dropped, not queued.
func _maybe_comment(mood: int, event: String) -> void:
	# Let it sleep — no ambient/heartbeat chatter while the cat is napping.
	if event == "heartbeat" and pet.state == "clip" and pet.clip_anim == "sleep":
		return
	if in_flight:
		return                          # one HTTPRequest node = one request
	if frame - last_comment_frame < COMMENT_COOLDOWN_TICKS:
		return
	last_comment_frame = frame
	_request_tick(mood, event)

func _request_tick(mood: int, event: String) -> void:
	pending_mood = mood
	var body := JSON.stringify({
		"pet_state": pet.state,
		"mood": _mood_key(mood),
		"event": event,
		"user_text": null,
	})
	var err := http.request(DAEMON_URL, ["Content-Type: application/json"],
		HTTPClient.METHOD_POST, body)
	if err != OK:
		_fallback_comment(mood)          # daemon unreachable -> canned
	else:
		in_flight = true

func _on_tick_completed(result: int, code: int, _headers: PackedStringArray,
		body: PackedByteArray) -> void:
	in_flight = false
	if result == HTTPRequest.RESULT_SUCCESS and code == 200:
		var data = JSON.parse_string(body.get_string_from_utf8())
		if typeof(data) == TYPE_DICTIONARY:
			var st := String(data.get("state", ""))
			var applied := st != "" and pet.drive_state(st)
			var say := String(data.get("say", ""))
			if say.strip_edges() != "" and say != last_say:
				last_say = say
				if bubble != null:
					bubble.show_line(say)
			elif st != "" and not applied:
				# LLM asked for an action that was rejected and said nothing -> canned
				_fallback_comment(pending_mood)
			return
	_fallback_comment(pending_mood)      # any failure -> canned, never silence

# Canned comment for a mood (also the fallback when the daemon is unreachable).
func _fallback_comment(mood: int) -> void:
	var line := commenter.pick(_mood_key(mood))
	if line != "" and line != last_say:
		last_say = line
		if bubble != null:
			bubble.show_line(line)
