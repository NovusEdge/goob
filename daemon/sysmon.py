"""Read-only system observers, exposed to the LLM as tools.

Ports goob's GDScript sysmon (main.gd). Note: Python's read() ignores the
bogus st_size that /proc and /sys report, so — unlike the GDScript — a plain
read() is correct here. Do NOT reintroduce line-by-line parsing.
"""
import os
from datetime import datetime

# Watched build/dev processes (mirrors main.gd WATCH).
WATCH = {
    "go", "gcc", "cc1", "clang", "rustc", "cargo", "node", "npm", "webpack",
    "tsc", "make", "cmake", "ninja", "docker", "gradle", "mvn", "python", "ld",
}


def _read(path):
    try:
        with open(path) as f:
            return f.read().strip()
    except OSError:
        return ""


def watched_processes(proc_root="/proc"):
    """Names of running processes that are in the WATCH allowlist."""
    found = []
    try:
        entries = os.listdir(proc_root)
    except OSError:
        return found
    for pid in entries:
        if not pid.isdigit():
            continue
        comm = _read(os.path.join(proc_root, pid, "comm"))
        if comm in WATCH and comm not in found:
            found.append(comm)
    return found


def _battery(power_root):
    for name in _glob(power_root, "BAT"):
        cap = _read(os.path.join(power_root, name, "capacity"))
        if cap.isdigit():
            status = _read(os.path.join(power_root, name, "status"))
            return int(cap), status != "Discharging"
    return -1, False


def _hottest(thermal_root):
    hot = 0
    for name in _glob(thermal_root, "thermal_zone"):
        temp = _read(os.path.join(thermal_root, name, "temp"))
        if temp.lstrip("-").isdigit():
            hot = max(hot, int(temp) // 1000)
    return hot


def _glob(root, prefix):
    try:
        return [n for n in os.listdir(root) if n.startswith(prefix)]
    except OSError:
        return []


def system_state(power_root="/sys/class/power_supply",
                 thermal_root="/sys/class/thermal"):
    pct, charging = _battery(power_root)
    return {"battery_pct": pct, "charging": charging,
            "hottest_c": _hottest(thermal_root)}


def time_context():
    now = datetime.now()
    h = now.hour
    part = ("night" if h < 6 else "morning" if h < 12
            else "afternoon" if h < 18 else "evening")
    return {"time": now.strftime("%H:%M"), "part_of_day": part}


OBSERVER_TOOLS = [
    {"type": "function", "function": {
        "name": "list_watched_processes",
        "description": "Which watched build/dev processes are currently running.",
        "parameters": {"type": "object", "properties": {}},
    }},
    {"type": "function", "function": {
        "name": "get_system_state",
        "description": "Battery percent, whether charging, hottest thermal zone (C).",
        "parameters": {"type": "object", "properties": {}},
    }},
    {"type": "function", "function": {
        "name": "get_time_context",
        "description": "Local time and rough part of day.",
        "parameters": {"type": "object", "properties": {}},
    }},
]


def dispatch(name, args):
    if name == "list_watched_processes":
        return {"running": watched_processes()}
    if name == "get_system_state":
        return system_state()
    if name == "get_time_context":
        return time_context()
    return {}
