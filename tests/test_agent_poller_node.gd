extends SceneTree
# Run: godot --headless --path . --script res://tests/test_agent_poller_node.gd
# Regression test for the real Node poll() path with wall-clock-consistent
# values — this is what would have caught the ts unit-mismatch bug (main.gd
# was passing engine uptime while hooks/goob_hook.py writes time.time()).

class StubPet extends RefCounted:
	var drove: Array = []
	func drive_state(n: String) -> bool: drove.append(n); return true

func _write(path: String, text: String) -> void:
	var f := FileAccess.open(path, FileAccess.WRITE)
	f.store_string(text); f.close()

func _init() -> void:
	var host := Node.new(); host.name = "H"; get_root().add_child(host)
	var pet := StubPet.new()
	var hsm := AgentHsm.build(host, pet)

	var poller := AgentPoller.new()
	host.add_child(poller)
	poller.setup(hsm)
	poller.agent_file = ProjectSettings.globalize_path("user://poll_node_test.json")

	var t := 1000.0
	_write(poller.agent_file, '{"token":"done","ts":%f}' % t)

	poller.poll(t)
	assert(hsm.get_active_state() == hsm.get_node("Reacting"))
	hsm.update(0.0)
	assert(pet.drove.has("zoomies"))

	# Without touching the file, advance the clock by more than STALE_AFTER_S
	# (8s) in matching wall-clock units — staleness must fire and return the
	# HSM to Ambient. Under the old bug (uptime vs epoch seconds) this never
	# happened.
	poller.poll(t + 10.0)
	assert(hsm.get_active_state() == hsm.get_node("Ambient"))

	print("test_agent_poller_node OK")
	quit()
