class_name AgentTree
extends RefCounted

# Builds the reaction tree in code (exact + headless-testable; LimboAI's
# debugger still visualizes it live). Selector, first match wins:
#   done     -> zoomies once + "✅ done"
#   subagent -> zoomies once
#   else     -> idle, held (thinking/working)

const IsToken := preload("res://scripts/bt/bt_is_token.gd")
const Drive := preload("res://scripts/bt/bt_drive.gd")
const Speak := preload("res://scripts/bt/bt_speak.gd")

static func _seq(children: Array) -> BTSequence:
	var s := BTSequence.new()
	for c in children:
		s.add_child(c)
	return s

static func _is(token: String) -> BTCondition:
	var c := IsToken.new(); c.want = token; return c

static func _drive(verb: String, once: bool) -> BTAction:
	var d := Drive.new(); d.verb = verb; d.once = once; return d

static func _speak(text: String) -> BTAction:
	var s := Speak.new(); s.text = text; return s

static func build() -> BehaviorTree:
	var sel := BTSelector.new()
	sel.add_child(_seq([_is("done"), _drive("zoomies", true), _speak("✅ done")]))
	sel.add_child(_seq([_is("subagent"), _drive("zoomies", true)]))
	sel.add_child(_drive("idle", false))   # thinking/working fallback, holds
	var bt := BehaviorTree.new()
	bt.set_root_task(sel)
	return bt
