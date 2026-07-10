extends SceneTree
# Run: godot --headless --path . --script res://tests/test_agent_tree.gd

class StubPet extends RefCounted:
	var drove: Array = []
	func drive_state(n: String) -> bool: drove.append(n); return true

class StubSpeaker extends RefCounted:
	var said: Array = []
	func speak_reaction(text: String, ts: float) -> void: said.append([text, ts])

func _run(bt: BehaviorTree, host: Node, pet, spk, token: String) -> int:
	var bb := Blackboard.new()
	bb.set_var("pet", pet); bb.set_var("speaker", spk)
	bb.set_var("agent_token", token); bb.set_var("agent_ts", 1.0)
	var inst = bt.instantiate(host, bb, host, host)
	return inst.update(0.0)

func _init() -> void:
	var host := Node.new(); host.name = "H"; get_root().add_child(host)
	var pet := StubPet.new(); var spk := StubSpeaker.new()
	var bt := AgentTree.build()

	assert(_run(bt, host, pet, spk, "done") == BTTask.SUCCESS)
	assert(pet.drove == ["zoomies"] and spk.said.size() == 1 and spk.said[0][0] == "✅ done")

	pet.drove.clear(); spk.said.clear()
	assert(_run(bt, host, pet, spk, "subagent") == BTTask.SUCCESS)
	assert(pet.drove == ["zoomies"] and spk.said.is_empty())

	pet.drove.clear()
	assert(_run(bt, host, pet, spk, "thinking") == BTTask.RUNNING)  # BusyIdle holds
	assert(pet.drove == ["idle"])

	print("test_agent_tree OK")
	quit()
