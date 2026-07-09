extends SceneTree

# Headless unit test. Run: godot --headless --script res://tests/test_commenter.gd
func _initialize() -> void:
	var only := Commenter.new("res://tests/fixtures/one.json")
	assert(only.pick("alert") == "boom", "single line returns that line")
	assert(only.pick("nope") == "", "missing key returns empty")

	# Two-line list: consecutive picks must never immediately repeat.
	var d := Commenter.new("res://tests/fixtures/two.json")
	var a := d.pick("alert")
	var b := d.pick("alert")
	assert(a != b, "no immediate repeat with >=2 lines")
	assert(a in ["x", "y"] and b in ["x", "y"], "picks are from the list")

	var missing := Commenter.new("res://tests/does-not-exist.json")
	assert(missing.pick("alert") == "", "missing file degrades to empty")

	# Duplicate-valued list (value-based dedup): never repeat consecutively.
	var dup := Commenter.new("res://tests/fixtures/dupes.json")
	var prev := dup.pick("alert")
	for i in range(50):
		var cur := dup.pick("alert")
		assert(cur != prev, "duplicate-value list must never repeat consecutively (got %s twice)" % cur)
		assert(cur in ["a", "b"], "picks are from the list")
		prev = cur

	# Explicit empty array returns empty string.
	var empty_arr := Commenter.new("res://tests/fixtures/empty_arr.json")
	assert(empty_arr.pick("alert") == "", "empty array for a mood returns empty")

	print("test_commenter OK")
	quit()
