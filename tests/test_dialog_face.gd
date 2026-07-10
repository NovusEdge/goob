extends SceneTree
# Headless unit test. Run: godot --headless --script res://tests/test_dialog_face.gd

func _init() -> void:
	var have := PackedStringArray(["default", "talking"])
	assert(DialogFace.resolve_anim(have, "talking") == "talking")   # direct match
	assert(DialogFace.resolve_anim(have, "sleeping") == "default")  # fallback to default
	assert(DialogFace.resolve_anim(PackedStringArray(["only"]), "x") == "only")  # first
	assert(DialogFace.resolve_anim(PackedStringArray(), "x") == "")  # none
	print("test_dialog_face OK")
	quit()
