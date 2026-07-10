class_name DialogFace
extends AnimatedSprite2D

## Floating talking-head. Owns its speech bubble, drag, collapse, and bounce.
## main.gd forwards spoken lines via speak() and sleep state via set_sleeping().
## The face itself is authored in-editor as SpriteFrames animations (default /
## talking / sleeping) — code only calls play(), no per-pack slicing config.

const BUBBLE_TICKS := 240        # ~4s at 60fps physics — matches the old cat bubble
const PILL_SCALE := 0.5          # collapsed size, relative to the editor scale
const DRAG_THRESHOLD := 6.0      # px of motion before a press counts as a drag
const SAVE_PATH := "user://face_window.cfg"

var _base_scale := Vector2.ONE   # editor scale is the size knob; never overwrite it
var _talk_left := 0
var _sleeping := false
var _collapsed := false
var _tween: Tween = null

var _bubble_layer: CanvasLayer = null
var _bubble_panel: PanelContainer = null
var _bubble_label: Label = null

var _ui_layer: CanvasLayer = null
var _collapse_btn: Button = null

var _pressing := false
var _dragging := false
var _press_pos := Vector2.ZERO
var _grab_off := Vector2.ZERO

func _ready() -> void:
	_base_scale = scale
	_make_bubble()
	_make_button()
	_load_saved()          # may set position + _collapsed
	_apply_scale()
	_apply_face()
	_sync_button()

func _physics_process(_dt: float) -> void:
	_sync_button()               # keep the collapse button pinned to the face corner
	if _talk_left > 0:
		_talk_left -= 1
		if _talk_left <= 0:
			_bubble_layer.visible = false
			_apply_face()
		elif not _collapsed:
			_position_bubble()   # track the face if it's dragged mid-speech

# --- public API (called by main.gd) --------------------------------------

func speak(text: String) -> void:
	if _collapsed or text.strip_edges() == "":
		return               # a collapsed pill stays quiet
	_talk_left = BUBBLE_TICKS
	_bubble_label.text = text
	_bubble_layer.visible = true
	_position_bubble()
	_apply_face()
	_bounce()

func set_sleeping(sleeping: bool) -> void:
	if sleeping == _sleeping:
		return
	_sleeping = sleeping
	_apply_face()

# Screen-space rect of the visible face. AnimatedSprite2D is centered, so the
# top-left is position - size/2. The overlay window sits at (0,0) unscaled, so
# world coords == screen coords (same as the cat in main.gd).
func face_rect() -> Rect2:
	var fw := 0.0
	var fh := 0.0
	if sprite_frames != null and sprite_frames.get_animation_names().size() > 0:
		var t := sprite_frames.get_frame_texture(animation, 0)
		if t != null:
			fw = t.get_width() * scale.x
			fh = t.get_height() * scale.y
	return Rect2(position.x - fw / 2.0, position.y - fh / 2.0, fw, fh)

# Hit-test in the face's OWN local space. get_local_mouse_position() undoes every
# transform between the screen cursor and this node (window origin, content scale,
# canvas, the node's own scale), so this is correct regardless of any offset —
# unlike comparing a screen-space cursor against a canvas-space rect.
func over_face() -> bool:
	if sprite_frames == null:
		return false
	var t := sprite_frames.get_frame_texture(animation, 0)
	if t == null:
		return false
	# Centered sprite: local rect is (-half, +half) in unscaled texture pixels;
	# get_local_mouse_position() is already in that same unscaled local space.
	var half := Vector2(t.get_width(), t.get_height()) / 2.0
	return Rect2(-half, half * 2.0).has_point(get_local_mouse_position())

# --- face animation ------------------------------------------------------

func _apply_face() -> void:
	var want := "default"
	if _sleeping:
		want = "sleeping"
	elif _talk_left > 0:
		want = "talking"
	if sprite_frames != null:
		var a := resolve_anim(sprite_frames.get_animation_names(), want)
		if a != "":
			play(a)

# Resolve a wanted face state to an authored animation: direct name, then
# "default", then the first animation (degrade-toward-default, like main._resolve).
static func resolve_anim(names: PackedStringArray, want: String) -> String:
	if want in names:
		return want
	if "default" in names:
		return "default"
	return names[0] if names.size() > 0 else ""

# Quick scale-punch as a line pops. Returns to the current effective scale
# (pill or full), not the raw base, so a bounce while collapsed stays small.
func _bounce() -> void:
	var base := _effective_scale()
	if _tween != null and _tween.is_running():
		_tween.kill()
	scale = base
	_tween = create_tween()
	_tween.tween_property(self, "scale", base * 1.12, 0.075)
	_tween.tween_property(self, "scale", base, 0.075)

# --- bubble --------------------------------------------------------------

func _make_bubble() -> void:
	_bubble_layer = CanvasLayer.new()
	add_child(_bubble_layer)
	_bubble_panel = PanelContainer.new()
	var sb := StyleBoxFlat.new()
	sb.bg_color = Color(0.1, 0.1, 0.12, 0.92)
	sb.set_corner_radius_all(10)
	sb.set_content_margin_all(8)
	_bubble_panel.add_theme_stylebox_override("panel", sb)
	_bubble_layer.add_child(_bubble_panel)
	_bubble_label = Label.new()
	_bubble_label.add_theme_color_override("font_color", Color.WHITE)
	_bubble_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	# Real width to wrap within — without it the container collapses to ~1 char.
	_bubble_label.custom_minimum_size = Vector2(240, 0)
	_bubble_panel.add_child(_bubble_label)
	_bubble_layer.visible = false
	# ponytail: StyleBox bubble; swap in the Crusenho nine-patch once it's downloaded.

# Anchor the bubble to the right of the face, flipped left near the right edge,
# clamped on-screen.
func _position_bubble() -> void:
	var r := face_rect()
	var sz := _bubble_panel.size
	var scr := DisplayServer.screen_get_size()
	var bx := r.position.x + r.size.x + 8.0
	if bx + sz.x > scr.x:
		bx = r.position.x - sz.x - 8.0
	var by := r.position.y + r.size.y / 2.0 - sz.y / 2.0
	bx = clampf(bx, 0.0, float(scr.x) - sz.x)
	by = clampf(by, 0.0, float(scr.y) - sz.y)
	_bubble_panel.position = Vector2(bx, by)

# --- collapse button -----------------------------------------------------

func _make_button() -> void:
	_ui_layer = CanvasLayer.new()
	add_child(_ui_layer)
	_collapse_btn = Button.new()
	_collapse_btn.focus_mode = Control.FOCUS_NONE
	_collapse_btn.mouse_default_cursor_shape = Control.CURSOR_POINTING_HAND
	_collapse_btn.custom_minimum_size = Vector2(26, 26)
	_collapse_btn.add_theme_font_size_override("font_size", 16)
	_collapse_btn.pressed.connect(_on_collapse_pressed)
	_ui_layer.add_child(_collapse_btn)
	_refresh_button_label()

# Pin the button to the face's top-right corner in true screen space (the
# global-canvas transform is offset-immune, matching over_face()).
func _sync_button() -> void:
	if _collapse_btn == null or sprite_frames == null:
		return
	var t := sprite_frames.get_frame_texture(animation, 0)
	if t == null:
		return
	var half := Vector2(t.get_width(), t.get_height()) / 2.0
	var top_right := get_global_transform_with_canvas() * Vector2(half.x, -half.y)
	_collapse_btn.position = top_right - Vector2(_collapse_btn.size.x, 0.0)

func _refresh_button_label() -> void:
	_collapse_btn.text = "+" if _collapsed else "–"

func _on_collapse_pressed() -> void:
	_toggle_collapse()
	_refresh_button_label()
	_save()

# --- drag + collapse -----------------------------------------------------

func _unhandled_input(event: InputEvent) -> void:
	if event is InputEventMouseButton and event.button_index == MOUSE_BUTTON_LEFT:
		# get_global_mouse_position() is the cursor in the same canvas/world space
		# as `position`, so grab math is offset-immune.
		var gp := get_global_mouse_position()
		if event.pressed and over_face():
			_pressing = true
			_dragging = false
			_press_pos = gp
			_grab_off = gp - position
		elif not event.pressed and _pressing:
			# Body click is drag-only; collapse is the dedicated corner button.
			if _dragging:
				_save()
			_pressing = false
			_dragging = false
	elif event is InputEventMouseMotion and _pressing:
		var gp := get_global_mouse_position()
		if not _dragging and gp.distance_to(_press_pos) > DRAG_THRESHOLD:
			_dragging = true
		if _dragging:
			position = gp - _grab_off

func _toggle_collapse() -> void:
	_collapsed = not _collapsed
	if _collapsed:
		_talk_left = 0
		_bubble_layer.visible = false
	_apply_scale()

func _effective_scale() -> Vector2:
	return _base_scale * (PILL_SCALE if _collapsed else 1.0)

func _apply_scale() -> void:
	scale = _effective_scale()

# --- persistence ---------------------------------------------------------

func _save() -> void:
	var cfg := ConfigFile.new()
	cfg.set_value("face", "x", position.x)
	cfg.set_value("face", "y", position.y)
	cfg.set_value("face", "collapsed", _collapsed)
	cfg.save(SAVE_PATH)

func _load_saved() -> void:
	var cfg := ConfigFile.new()
	if cfg.load(SAVE_PATH) != OK:
		return
	var p := Vector2(cfg.get_value("face", "x", position.x),
		cfg.get_value("face", "y", position.y))
	# Keep the (centered) face fully on-screen even if a stale save is off-bounds.
	var scr := DisplayServer.screen_get_size()
	var half := face_rect().size / 2.0
	position = Vector2(
		clampf(p.x, half.x, float(scr.x) - half.x),
		clampf(p.y, half.y, float(scr.y) - half.y))
	_collapsed = bool(cfg.get_value("face", "collapsed", false))
