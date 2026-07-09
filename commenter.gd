class_name Commenter
extends RefCounted

# Picks a canned line for the pet to say, keyed by mood ("neutral"/"alert"/
# "tired"). Guarantees it never returns the same line twice in a row when the
# list has room. Pure data + logic — no scene dependency, headless-testable.

var _lines: Dictionary = {}
var _last: String = ""

func _init(path := "res://comments.json") -> void:
	var f := FileAccess.open(path, FileAccess.READ)
	if f == null:
		return
	var data = JSON.parse_string(f.get_as_text())
	if typeof(data) == TYPE_DICTIONARY:
		_lines = data

func pick(mood_key: String) -> String:
	var arr = _lines.get(mood_key, [])
	if typeof(arr) != TYPE_ARRAY or arr.is_empty():
		return ""
	var choice := String(arr[randi() % arr.size()])
	# no immediate repeat: if we rolled the last line, step to the next one
	if choice == _last and arr.size() > 1:
		var i: int = arr.find(_last)
		choice = String(arr[(i + 1) % arr.size()])
	_last = choice
	return choice
