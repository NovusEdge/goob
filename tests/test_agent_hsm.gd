extends SceneTree
# Run: godot --headless --path . --script res://tests/test_agent_hsm.gd

class StubPet extends RefCounted:
	var drove: Array = []
	func drive_state(n: String) -> bool: drove.append(n); return true

func _init() -> void:
	var host := Node.new(); host.name = "H"; get_root().add_child(host)
	var pet := StubPet.new()
	var hsm := AgentHsm.build(host, pet)

	assert(hsm.get_active_state() == hsm.get_node("Ambient"))

	# a "done" event -> Reacting -> tree drives zoomies (speech is a no-op)
	hsm.get_blackboard().set_var("agent_token", "done")
	hsm.get_blackboard().set_var("agent_ts", 1.0)
	hsm.dispatch(&"agent_event")
	assert(hsm.get_active_state() == hsm.get_node("Reacting"))
	hsm.update(0.0)
	assert(pet.drove.has("zoomies"))

	# stale -> back to Ambient
	hsm.dispatch(&"agent_stale")
	assert(hsm.get_active_state() == hsm.get_node("Ambient"))

	# sleep from anywhere
	hsm.dispatch(&"agent_sleep")
	assert(hsm.get_active_state() == hsm.get_node("Sleeping"))

	print("test_agent_hsm OK")
	quit()
