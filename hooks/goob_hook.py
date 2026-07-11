#!/usr/bin/env python3
"""goob hook: map an event or token to a goob token and atomically write it
to the agent-event file the pet polls. Stdlib only; any error is swallowed
so a broken pet can never wedge the calling agent.

Usage:
  python3 goob_hook.py <CCEventName>    # Claude Code event -> token
  python3 goob_hook.py --token <tok>    # direct token (validated)
"""
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

# Valid goob tokens for --token mode
VALID_TOKENS = {"wake", "thinking", "working", "subagent", "done", "sleep"}

def main():
    if len(sys.argv) < 2:
        return
    # --token mode: direct token, validated
    if sys.argv[1] == "--token":
        if len(sys.argv) < 3:
            return
        token = sys.argv[2]
        if token not in VALID_TOKENS:
            return
    else:
        # event-name mode: map CC event to token
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
