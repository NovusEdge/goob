class_name AgentPoller
extends Node

# Pure, testable static core (Task 2). The Node wiring (Timer, dispatch to the
# HSM) is added in Task 5 via poll(). Reactions live tokens: thinking/working/
# subagent/done; lifecycle tokens: wake/sleep.

const FRESH_TOKENS := ["thinking", "working", "subagent", "done"]
const STALE_AFTER_S := 8.0

static func read_event(path: String) -> Dictionary:
	if not FileAccess.file_exists(path):
		return {}
	var f := FileAccess.open(path, FileAccess.READ)
	if f == null:
		return {}
	var data = JSON.parse_string(f.get_as_text())
	f.close()
	if typeof(data) != TYPE_DICTIONARY or not data.has("token") or not data.has("ts"):
		return {}
	return {"token": String(data["token"]), "ts": float(data["ts"])}

static func decide(prev: Dictionary, cur: Dictionary, now_s: float, reacting: bool) -> StringName:
	var changed: bool = not cur.is_empty() and \
		(prev.get("ts") != cur.get("ts") or prev.get("token") != cur.get("token"))
	if changed:
		var tok := String(cur.get("token", ""))
		if tok == "sleep":
			return &"agent_sleep"
		if tok == "wake":
			return &"agent_wake"
		if tok in FRESH_TOKENS:
			return &"agent_event"
		return &""
	if reacting and not cur.is_empty() and now_s - float(cur.get("ts", now_s)) > STALE_AFTER_S:
		return &"agent_stale"
	return &""
