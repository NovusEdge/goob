class_name AgentHsm
extends RefCounted

# Builds the top-level pet HSM. Ambient = hands-off (pet.gd self-drives).
# Reacting = a BTState running AgentTree. Sleeping = calm (re-assert idle).
# Transitions are event-based; the poller dispatches them.

const AgentTreeScript := preload("res://scripts/agent_tree.gd")

static func build(host: Node, pet) -> LimboHSM:
	var hsm := LimboHSM.new()
	hsm.name = "AgentHSM"

	var ambient := LimboState.new()
	ambient.name = "Ambient"                     # no-op: pet.gd self-drives

	var reacting := BTState.new()
	reacting.name = "Reacting"
	reacting.set_behavior_tree(AgentTreeScript.build())
	reacting.set_scene_root_hint(host)           # required or the tree won't instantiate

	var sleeping := LimboState.new()
	sleeping.name = "Sleeping"
	sleeping.call_on_update(func(_delta): pet.drive_state("idle"))

	hsm.add_child(ambient)
	hsm.add_child(reacting)
	hsm.add_child(sleeping)
	host.add_child(hsm)

	hsm.initialize(host)
	hsm.get_blackboard().set_var("pet", pet)
	hsm.get_blackboard().set_var("agent_token", "")
	hsm.get_blackboard().set_var("agent_ts", 0.0)
	hsm.set_initial_state(ambient)

	hsm.add_transition(ambient, reacting, &"agent_event")
	hsm.add_transition(reacting, ambient, &"agent_stale")
	hsm.add_transition(sleeping, ambient, &"agent_wake")
	hsm.add_transition(sleeping, reacting, &"agent_event")
	hsm.add_transition(hsm.anystate(), sleeping, &"agent_sleep")

	hsm.set_active(true)
	return hsm
