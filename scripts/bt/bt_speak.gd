extends BTAction
# Speak a canned reaction line, debounced on the event ts by the speaker.

@export var text := ""

func _tick(_delta: float) -> Status:
	var spk = get_blackboard().get_var("speaker")
	if spk != null:
		spk.speak_reaction(text, float(get_blackboard().get_var("agent_ts", 0.0)))
	return SUCCESS
