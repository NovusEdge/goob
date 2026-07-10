extends SceneTree
# Run: godot --headless --path . --script res://tests/test_agent_poller.gd

func _write(path: String, text: String) -> void:
	var f := FileAccess.open(path, FileAccess.WRITE)
	f.store_string(text); f.close()

func _init() -> void:
	var p := "user://poll_test.json"
	var abs := ProjectSettings.globalize_path(p)

	# read_event
	_write(abs, '{"token":"done","ts":12.5}')
	var ev := AgentPoller.read_event(abs)
	assert(ev.get("token") == "done" and ev.get("ts") == 12.5)
	assert(AgentPoller.read_event(abs + ".nope").is_empty())        # missing
	_write(abs, "not json{")
	assert(AgentPoller.read_event(abs).is_empty())                  # malformed

	# decide
	var a := {"token": "thinking", "ts": 100.0}
	var b := {"token": "working", "ts": 100.3}
	assert(AgentPoller.decide({}, a, 100.0, false) == &"agent_event")   # first event
	assert(AgentPoller.decide(a, a, 100.0, true) == &"")                # unchanged, fresh
	assert(AgentPoller.decide(a, b, 100.3, true) == &"agent_event")     # same-sec new token
	assert(AgentPoller.decide(a, a, 109.0, true) == &"agent_stale")     # >8s stale
	assert(AgentPoller.decide(a, {"token":"sleep","ts":200.0}, 200.0, false) == &"agent_sleep")
	assert(AgentPoller.decide(a, {"token":"wake","ts":200.0}, 200.0, true) == &"agent_wake")

	print("test_agent_poller OK")
	quit()
