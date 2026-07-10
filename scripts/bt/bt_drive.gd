extends BTAction
# Drive the pet via the existing drive_state() channel. `once` = edge-triggered
# (drive once, return SUCCESS — for self-terminating verbs like zoomies).
# `once == false` re-asserts every tick and returns RUNNING (pins idle so
# pet.gd's _decide() never rolls an autonomous behavior).

@export var verb := "idle"
@export var once := false

func _tick(_delta: float) -> Status:
	var pet = get_blackboard().get_var("pet")
	if pet != null:
		pet.drive_state(verb)
	return SUCCESS if once else RUNNING
