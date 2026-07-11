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

def _run_token(token, path):
    env = dict(os.environ, GOOB_AGENT_FILE=path)
    subprocess.run([sys.executable, HOOK, "--token", token], env=env, check=True)

def test_token_mode_valid():
    with tempfile.TemporaryDirectory() as d:
        p = os.path.join(d, "a.json")
        _run_token("done", p)
        data = json.load(open(p))
        assert data["token"] == "done"

def test_token_mode_all_tokens():
    with tempfile.TemporaryDirectory() as d:
        for tok in ["wake", "thinking", "working", "subagent", "done", "sleep"]:
            p = os.path.join(d, f"{tok}.json")
            _run_token(tok, p)
            assert json.load(open(p))["token"] == tok

def test_token_mode_invalid_writes_nothing():
    with tempfile.TemporaryDirectory() as d:
        p = os.path.join(d, "a.json")
        _run_token("bogus", p)
        assert not os.path.exists(p)

def test_token_mode_missing_arg():
    with tempfile.TemporaryDirectory() as d:
        p = os.path.join(d, "a.json")
        env = dict(os.environ, GOOB_AGENT_FILE=p)
        subprocess.run([sys.executable, HOOK, "--token"], env=env)
        assert not os.path.exists(p)

if __name__ == "__main__":
    test_maps_known_event(); test_prompt_and_tool_tokens(); test_unknown_event_writes_nothing()
    test_token_mode_valid(); test_token_mode_all_tokens()
    test_token_mode_invalid_writes_nothing(); test_token_mode_missing_arg()
    print("test_goob_hook OK")
