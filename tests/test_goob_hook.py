import json, os, subprocess, sys, tempfile

HOOK = os.path.join(os.path.dirname(__file__), "..", "hooks", "goob_hook.py")

def _run(event, path):
    env = dict(os.environ, GOOB_AGENT_FILE=path)
    subprocess.run([sys.executable, HOOK, event], env=env, check=True)

def test_maps_known_event():
    with tempfile.TemporaryDirectory() as d:
        p = os.path.join(d, "a.json")
        _run("Stop", p)
        data = json.load(open(p))
        assert data["token"] == "done"
        assert isinstance(data["ts"], float)

def test_prompt_and_tool_tokens():
    with tempfile.TemporaryDirectory() as d:
        p = os.path.join(d, "a.json")
        _run("UserPromptSubmit", p); assert json.load(open(p))["token"] == "thinking"
        _run("PreToolUse", p);       assert json.load(open(p))["token"] == "working"
        _run("SubagentStop", p);     assert json.load(open(p))["token"] == "subagent"
        _run("SessionEnd", p);       assert json.load(open(p))["token"] == "sleep"

def test_unknown_event_writes_nothing():
    with tempfile.TemporaryDirectory() as d:
        p = os.path.join(d, "a.json")
        _run("PreCompact", p)   # mapped to nothing
        assert not os.path.exists(p)

if __name__ == "__main__":
    test_maps_known_event(); test_prompt_and_tool_tokens(); test_unknown_event_writes_nothing()
    print("test_goob_hook OK")
