extends BTAction
# Speak a canned reaction line, debounced on the event ts by the speaker.
# Dormant while no speech surface is wired: with no "speaker" on the blackboard
# this no-ops. The default + complain=false keeps it quiet (no ERROR spam).

@export var text := ""

func _tick(_delta: float) -> Status:
	var spk = get_blackboard().get_var("speaker", null, false)
	if spk != null:
		spk.speak_reaction(text, float(get_blackboard().get_var("agent_ts", 0.0)))
	return SUCCESS
