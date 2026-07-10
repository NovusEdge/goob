#!/usr/bin/env python3
"""goob Claude Code hook: map a CC event name (argv[1]) to a goob token and
atomically write it to the agent-event file the pet polls. Stdlib only; any
error is swallowed so a broken pet can never wedge Claude Code."""
import json, os, sys, tempfile, time

# Verified Claude Code hook events -> goob tokens. Unlisted events are ignored.
EVENTS = {
    "SessionStart": "wake",
    "UserPromptSubmit": "thinking",
    "PreToolUse": "working",
    "PostToolUse": "working",
    "SubagentStop": "subagent",
    "Stop": "done",
    "SessionEnd": "sleep",
}

def main():
    if len(sys.argv) < 2:
        return
    token = EVENTS.get(sys.argv[1])
    if token is None:
        return
    path = os.environ.get("GOOB_AGENT_FILE", "/tmp/goob-agent.json")
    body = json.dumps({"token": token, "ts": time.time()}).encode()
    d = os.path.dirname(path) or "."
    fd, tmp = tempfile.mkstemp(dir=d)
    try:
        os.write(fd, body)
        os.close(fd)
        os.replace(tmp, path)   # atomic: poller never sees a torn file
    except OSError:
        try: os.unlink(tmp)
        except OSError: pass

if __name__ == "__main__":
    try:
        main()
    except Exception:
        pass   # never fail the hook
