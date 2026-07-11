#!/usr/bin/env python3
"""goob Codex dispatcher: receives Codex notify payloads (JSON on stdin or argv)
and maps them to goob tokens. Stdlib only; swallows all errors.

Codex calls: notify = ["python3", "/path/to/goob_codex_notify.py"]
The notification JSON has an event field we map to a goob token.
"""
import json, os, sys, tempfile, time

# Codex event types -> goob tokens (conservative mapping)
CODEX_EVENTS = {
    "turn_complete": "done",
    "agent_idle": "sleep",
    "user_input": "thinking",
}

def write_token(token):
    """Atomically write token to the agent-event file."""
    path = os.environ.get("GOOB_AGENT_FILE", "/tmp/goob-agent.json")
    body = json.dumps({"token": token, "ts": time.time()}).encode()
    d = os.path.dirname(path) or "."
    fd, tmp = tempfile.mkstemp(dir=d)
    try:
        os.write(fd, body)
        os.close(fd)
        os.replace(tmp, path)
    except OSError:
        try: os.unlink(tmp)
        except OSError: pass

def main():
    # try reading JSON from stdin first, then argv
    payload = None
    try:
        if not sys.stdin.isatty():
            data = sys.stdin.read()
            if data.strip():
                payload = json.loads(data)
    except Exception:
        pass
    if payload is None and len(sys.argv) > 1:
        try:
            payload = json.loads(sys.argv[1])
        except Exception:
            pass
    if not isinstance(payload, dict):
        return
    event = payload.get("event") or payload.get("type")
    if not event:
        return
    token = CODEX_EVENTS.get(event)
    if token:
        write_token(token)

if __name__ == "__main__":
    try:
        main()
    except Exception:
        pass
