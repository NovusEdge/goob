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
	# no immediate repeat: if we rolled the last line, pick from the others
	if choice == _last:
		var others: Array = []
		for v in arr:
			if String(v) != _last:
				others.append(String(v))
		if not others.is_empty():
			choice = String(others[randi() % others.size()])
	_last = choice
	return choice
