class_name BubbleWindow
extends CanvasLayer

## Fixed speech-bubble pinned to the top-right of the viewport (below the debug
## box). show_line() fades a NinePatch panel in, holds it for a reading-time
## duration, fades out. The X dismisses early. Not draggable — deliberately
## simple. Replaces the old face as the speech surface.

signal dismissed
signal shown_fully

const READ_WPS := 3.3
const MIN_SECS := 5.0
const MAX_SECS := 12.0
const FADE_IN := 0.2
const FADE_OUT := 0.4
const MIN_WIDTH := 280.0
const MAX_WIDTH := 400.0
const PAD := 16.0           # MarginContainer padding on each side
const MARGIN := 12.0        # gap from the viewport's top-right corner
const TOP := 104.0          # below the debug box (which ends ~y96)
const FRAME_TEX := "res://assets/ui/UI_Flat_Frame01a.png"
const CROSS_TEX := "res://assets/ui/UI_Flat_ButtonCross01a.png"

var _root: Control = null
var _panel: NinePatchRect = null
var _label: Label = null
var _close: TextureButton = null
var _tween: Tween = null

static func hold_secs(word_count: int) -> float:
	return clampf(float(word_count) / READ_WPS, MIN_SECS, MAX_SECS)

const DEBUG_STAY := false  # disabled for now

func _ready() -> void:
	visible = true
	_build()
	_root.visible = false
	if DEBUG_STAY:
		_label.text = "test bubble — click the X to close me."
		_resize_to_text(_label.text)
		_root.modulate.a = 1.0
		_root.visible = true

func is_showing() -> bool:
	return _root.visible

# --- public API ----------------------------------------------------------

func show_line(text: String) -> void:
	if text.strip_edges() == "":
		return
	_label.text = text
	_resize_to_text(text)
	_root.modulate.a = 0.0
	_root.visible = true
	if _tween != null and _tween.is_running():
		_tween.kill()
	var hold := hold_secs(text.split(" ", false).size())
	_tween = create_tween()
	_tween.tween_property(_root, "modulate:a", 1.0, FADE_IN)
	_tween.tween_interval(hold)
	_tween.tween_callback(shown_fully.emit)
	_tween.tween_property(_root, "modulate:a", 0.0, FADE_OUT)
	_tween.tween_callback(func(): _root.visible = false)

func _dismiss() -> void:
	if _tween != null and _tween.is_running():
		_tween.kill()
	_tween = create_tween()
	_tween.tween_property(_root, "modulate:a", 0.0, 0.15)
	_tween.tween_callback(func(): _root.visible = false)
	dismissed.emit()

# Screen rect + hit-tests for main.gd's passthrough (so the X is clickable).
# get_global_transform_with_canvas maps viewport-local → final screen pixels,
# and over_bubble compares against the global cursor — both robust to the
# window offset / content-scale this compositor applies.
func panel_rect() -> Rect2:
	var t := _panel.get_global_transform_with_canvas()
	return Rect2(t * Vector2.ZERO, _panel.size * t.get_scale())

func over_bubble() -> bool:
	return _root.visible and panel_rect().has_point(Vector2(DisplayServer.mouse_get_position()))

func _close_rect() -> Rect2:
	var t := _close.get_global_transform_with_canvas()
	return Rect2(t * Vector2.ZERO, _close.size * t.get_scale())

# --- build ---------------------------------------------------------------

func _build() -> void:
	_root = Control.new()
	_root.set_anchors_preset(Control.PRESET_FULL_RECT)
	_root.mouse_filter = Control.MOUSE_FILTER_IGNORE
	add_child(_root)

	_panel = NinePatchRect.new()
	_panel.texture = load(FRAME_TEX)
	_panel.patch_margin_left = 8
	_panel.patch_margin_top = 8
	_panel.patch_margin_right = 8
	_panel.patch_margin_bottom = 8
	# ponytail: anchor top-right so it survives viewport resize
	_panel.set_anchors_preset(Control.PRESET_TOP_RIGHT)
	# Initial size/position — _resize_to_text() updates these for actual content
	_panel.size = Vector2(MIN_WIDTH, 80)
	_panel.position = Vector2(-MIN_WIDTH - MARGIN, TOP)
	_root.add_child(_panel)

	var margin := MarginContainer.new()
	margin.set_anchors_preset(Control.PRESET_FULL_RECT)
	for side in ["left", "top", "right", "bottom"]:
		margin.add_theme_constant_override("margin_" + side, PAD)
	margin.mouse_filter = Control.MOUSE_FILTER_IGNORE
	_panel.add_child(margin)

	_label = Label.new()
	_label.autowrap_mode = TextServer.AUTOWRAP_WORD_SMART
	_label.custom_minimum_size = Vector2(MIN_WIDTH - PAD * 2, 0)
	_label.mouse_filter = Control.MOUSE_FILTER_IGNORE
	_label.add_theme_color_override("font_color", Color(0.12, 0.12, 0.14))
	margin.add_child(_label)

	_close = TextureButton.new()
	_close.texture_normal = load(CROSS_TEX)
	_close.mouse_filter = Control.MOUSE_FILTER_IGNORE
	# Position at top-right corner of panel, inset by padding
	_close.position = Vector2(MAX_WIDTH - 17 - 6, 6)  # 17 = button width, 6 = inset
	_panel.add_child(_close)

# Measures the wrapped text extent (via the label's font, clamped to
# MAX_WIDTH) and resizes the panel to fit — so short lines stay compact and
# long ones wrap instead of stretching the bubble sideways.
func _resize_to_text(text: String) -> void:
	var font := _label.get_theme_font("font")
	var font_size := _label.get_theme_font_size("font_size")
	var text_width := MAX_WIDTH - PAD * 2.0
	# Constrain label width so autowrap works
	_label.custom_minimum_size.x = text_width
	_label.custom_maximum_size.x = text_width
	# Calculate wrapped height
	var wrapped := font.get_multiline_string_size(text, HORIZONTAL_ALIGNMENT_LEFT, text_width, font_size)
	var panel_height := wrapped.y + PAD * 2.0 + 8  # +8 for some breathing room
	_panel.size = Vector2(MAX_WIDTH, panel_height)
	_panel.position = Vector2(-MAX_WIDTH - MARGIN, TOP)

# Only interactivity: click the X. Global-cursor hit-test so it works despite
# the passthrough / window offset.
func _unhandled_input(event: InputEvent) -> void:
	if not _root.visible:
		return
	if event is InputEventMouseButton and event.button_index == MOUSE_BUTTON_LEFT and event.pressed:
		if _close_rect().has_point(Vector2(DisplayServer.mouse_get_position())):
			_dismiss()
