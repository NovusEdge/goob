extends SceneTree

# Run: godot --headless --script res://tests/test_drive_state.gd
func _initialize() -> void:
	var cfg := PetConfig.new()          # defaults: follow_cursor + gravity true
	var pet := PetBrain.new()
	pet.setup(1920, 1080, 64, 64, cfg, {}, false)
	# leave the appear clip: force an interruptible idle
	pet.state = "idle"

	assert(pet.drive_state("wander") == true, "wander is drivable")
	assert(pet.state == "wander", "state applied")

	pet.state = "idle"
	assert(pet.drive_state("explode") == false, "unknown state rejected")
	assert(pet.state == "idle", "state unchanged on reject")

	pet.state = "follow"                 # follow is non-interruptible
	assert(pet.drive_state("wander") == false, "not applied while uninterruptible")

	print("test_drive_state OK")
	quit()
