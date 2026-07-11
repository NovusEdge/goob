import json, os, subprocess, sys, tempfile

HOOK = os.path.join(os.path.dirname(__file__), "..", "hooks", "goob_codex_notify.py")

def _run_stdin(payload, path):
    env = dict(os.environ, GOOB_AGENT_FILE=path)
    subprocess.run(
        [sys.executable, HOOK],
        input=json.dumps(payload).encode() if payload else b"",
        env=env,
    )

def _run_argv(payload, path):
    env = dict(os.environ, GOOB_AGENT_FILE=path)
    subprocess.run(
        [sys.executable, HOOK, json.dumps(payload)],
        env=env,
    )

def test_turn_complete_via_stdin():
    with tempfile.TemporaryDirectory() as d:
        p = os.path.join(d, "a.json")
        _run_stdin({"event": "turn_complete"}, p)
        data = json.load(open(p))
        assert data["token"] == "done"

def test_turn_complete_via_argv():
    with tempfile.TemporaryDirectory() as d:
        p = os.path.join(d, "a.json")
        _run_argv({"event": "turn_complete"}, p)
        data = json.load(open(p))
        assert data["token"] == "done"

def test_unknown_event_writes_nothing():
    with tempfile.TemporaryDirectory() as d:
        p = os.path.join(d, "a.json")
        _run_stdin({"event": "unknown_event_type"}, p)
        assert not os.path.exists(p)

def test_malformed_json_writes_nothing():
    with tempfile.TemporaryDirectory() as d:
        p = os.path.join(d, "a.json")
        env = dict(os.environ, GOOB_AGENT_FILE=p)
        subprocess.run([sys.executable, HOOK], input=b"not json", env=env)
        assert not os.path.exists(p)

def test_no_event_field_writes_nothing():
    with tempfile.TemporaryDirectory() as d:
        p = os.path.join(d, "a.json")
        _run_stdin({"something": "else"}, p)
        assert not os.path.exists(p)

if __name__ == "__main__":
    test_turn_complete_via_stdin()
    test_turn_complete_via_argv()
    test_unknown_event_writes_nothing()
    test_malformed_json_writes_nothing()
    test_no_event_field_writes_nothing()
    print("test_goob_codex_notify OK")
