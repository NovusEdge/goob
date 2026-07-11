# Agent reactivity

The pet reacts to your local Claude Code session via a hook that writes events
to `/tmp/goob-agent.json`, polled by the pet. On by default; set `GOOB_HSM=0`
to disable.

## Setup

Register the hook in `~/.claude/settings.json` (or run `just install`):

    ```json
    {
      "hooks": {
        "SessionStart":     [{"hooks": [{"type": "command", "command": "python3 /ABS/PATH/goob/hooks/goob_hook.py SessionStart"}]}],
        "UserPromptSubmit": [{"hooks": [{"type": "command", "command": "python3 /ABS/PATH/goob/hooks/goob_hook.py UserPromptSubmit"}]}],
        "PreToolUse":       [{"hooks": [{"type": "command", "command": "python3 /ABS/PATH/goob/hooks/goob_hook.py PreToolUse"}]}],
        "PostToolUse":      [{"hooks": [{"type": "command", "command": "python3 /ABS/PATH/goob/hooks/goob_hook.py PostToolUse"}]}],
        "SubagentStop":     [{"hooks": [{"type": "command", "command": "python3 /ABS/PATH/goob/hooks/goob_hook.py SubagentStop"}]}],
        "Stop":             [{"hooks": [{"type": "command", "command": "python3 /ABS/PATH/goob/hooks/goob_hook.py Stop"}]}],
        "SessionEnd":       [{"hooks": [{"type": "command", "command": "python3 /ABS/PATH/goob/hooks/goob_hook.py SessionEnd"}]}]
      }
    }
    ```

    (All seven events the hook maps. Drop any you don't want the pet reacting to.)

The hook is Python-3 stdlib only and swallows all errors, so it can never block
or fail your Claude Code session.

## Reactions

| Claude Code does | pet does |
|------------------|----------|
| prompt / tools running | pins calm |
| a subagent finishes | zoomies |
| task completes (Stop) | zoomies |
| session ends | settles |
