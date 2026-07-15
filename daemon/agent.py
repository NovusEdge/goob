"""The tool-use loop. The model observes via read-only tools, then MUST call
the single terminal `emit` tool to deliver its reaction. We never parse free-
form JSON from prose — emit's validated arguments are the only sanctioned
answer; anything else degrades to {} (Godot then shows a canned line).
"""
import json

from daemon import sysmon

# The only states the LLM may push the pet into (must match pet.gd DRIVABLE).
DRIVABLE = ["idle", "wander", "follow", "jump", "zoomies"]

EMIT_TOOL = {"type": "function", "function": {
    "name": "emit",
    "description": ("Deliver the pet's reaction and finish. Call exactly once "
                    "when done. Omit a field to skip it."),
    "parameters": {"type": "object", "properties": {
        "say": {"type": "string",
                "description": "A short, in-character line for the speech bubble."},
        "state": {"type": "string", "enum": DRIVABLE,
                  "description": "Behaviour to switch the pet into."},
    }},
}}


def _validate(args):
    out = {}
    say = args.get("say")
    if isinstance(say, str) and say.strip():
        out["say"] = say.strip()[:200]
    state = args.get("state")
    if state in DRIVABLE:
        out["state"] = state
    return out


def run_agent(facts, personality, completion, max_steps=3):
    messages = [
        {"role": "system", "content": personality},
        {"role": "user", "content": json.dumps(facts)},
    ]
    tools = sysmon.OBSERVER_TOOLS + [EMIT_TOOL]
    for step in range(max_steps):
        # Force emit on the final step so the loop always terminates cleanly.
        force = ({"type": "function", "function": {"name": "emit"}}
                 if step == max_steps - 1 else "auto")
        msg = completion(messages, tools, force)
        calls = msg.get("tool_calls") or []
        if not calls:
            return {}          # model returned prose instead of a tool call
        messages.append({"role": "assistant", "content": msg.get("content"),
                         "tool_calls": calls})
        for c in calls:
            name = c["function"]["name"]
            try:
                args = json.loads(c["function"]["arguments"] or "{}")
            except json.JSONDecodeError:
                args = {}
            if name == "emit":
                return _validate(args)
            result = sysmon.dispatch(name, args)
            messages.append({"role": "tool", "tool_call_id": c["id"],
                             "content": json.dumps(result)})
    return {}
