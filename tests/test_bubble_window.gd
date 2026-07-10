extends SceneTree
# Headless unit test. Run: godot --headless --script res://tests/test_bubble_window.gd

func _init() -> void:
	assert(is_equal_approx(BubbleWindow.hold_secs(0), 5.0))     # floor
	assert(is_equal_approx(BubbleWindow.hold_secs(3), 5.0))     # 3/3.3 < 5 -> floor
	assert(BubbleWindow.hold_secs(30) > 5.0 and BubbleWindow.hold_secs(30) < 12.0)
	assert(is_equal_approx(BubbleWindow.hold_secs(100), 12.0))  # ceil
	print("test_bubble_window OK")
	quit()
