extends BTCondition
# SUCCESS iff the live agent token equals `want`.

@export var want := ""

func _tick(_delta: float) -> Status:
	return SUCCESS if get_blackboard().get_var("agent_token", "") == want else FAILURE
