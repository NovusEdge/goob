class_name SysMon
extends RefCounted

# System-state mood source (port of internal/sysmon). Pure /proc + /sys/class
# reads — no scene dependency. main.gd samples this periodically and feeds the
# result into PetBrain.set_mood(); see docs/behavior-model.md.

# Compiler/build processes that mean "a real build is running" -> alert mood.
# Deliberately excludes long-lived runtimes (python, node, go) — they match the
# daemon itself, editors, and the TUI, which would peg mood at alert forever.
const WATCH := ["gcc", "cc1", "clang", "rustc", "cargo", "npm", "webpack",
	"tsc", "make", "cmake", "ninja", "gradle", "mvn", "ld"]

# Reads current system state and returns a mood: 0 neutral, 1 alert, 2 tired.
func poll() -> int:
	var pct := _battery_pct()
	var charging := _charging()
	if (pct >= 0 and pct < 15 and not charging) or _hottest_zone() >= 85:
		return 2 # tired
	if _building():
		return 1 # alert
	return 0

func _building() -> bool:
	var d := DirAccess.open("/proc")
	if d == null:
		return false
	d.list_dir_begin()
	var entry := d.get_next()
	while entry != "":
		if entry.is_valid_int():
			var comm := _read_text("/proc/%s/comm" % entry).strip_edges()
			if comm in WATCH:
				return true
		entry = d.get_next()
	return false

func _battery_pct() -> int:
	for b in _glob("/sys/class/power_supply", "BAT"):
		var s := _read_text("%s/capacity" % b).strip_edges()
		if s.is_valid_int():
			return int(s)
	return -1

func _charging() -> bool:
	for b in _glob("/sys/class/power_supply", "BAT"):
		if _read_text("%s/status" % b).strip_edges() != "Discharging":
			return true
	return false

func _hottest_zone() -> int:
	var hot := 0
	for z in _glob("/sys/class/thermal", "thermal_zone"):
		var s := _read_text("%s/temp" % z).strip_edges()
		if s.is_valid_int():
			var c := int(s) / 1000
			if c > hot:
				hot = c
	return hot

func _glob(dir: String, prefix: String) -> Array:
	var out := []
	var d := DirAccess.open(dir)
	if d == null:
		return out
	d.list_dir_begin()
	var entry := d.get_next()
	while entry != "":
		if entry.begins_with(prefix):
			out.append("%s/%s" % [dir, entry])
		entry = d.get_next()
	return out

func _read_text(path: String) -> String:
	# /proc and /sys report bogus file lengths, so get_as_text() asserts and
	# spams. These are all single-value files — read one line instead.
	var f := FileAccess.open(path, FileAccess.READ)
	if f == null:
		return ""
	return f.get_line()
