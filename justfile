# goob — Godot 4 desktop pet
# Override the binary with:  GODOT=/path/to/godot just run

godot := env_var_or_default("GODOT", "godot")

default:
    @just --list

# run the pet
run:
    {{godot}} --path .

# open the project in the Godot editor
edit:
    {{godot}} -e --path .

# headless parse/import check (also registers new class_names)
check:
    {{godot}} --headless --path . --editor --quit

# headless unit tests
test:
    {{godot}} --headless --path . --script res://tests/test_commenter.gd
    {{godot}} --headless --path . --script res://tests/test_drive_state.gd
    {{godot}} --headless --path . --script res://tests/test_agent_poller.gd
    {{godot}} --headless --path . --script res://tests/test_agent_hsm.gd
    {{godot}} --headless --path . --script res://tests/test_agent_tree.gd
    python3 tests/test_goob_hook.py

# run the LLM daemon (opt-in). uv reads pyproject.toml for deps (litellm). Load
# your .env first (`set -a; source .env; set +a`) so the provider key is present.
daemon:
    uv run python -u -m daemon.server

# run the control-panel TUI (launches/monitors the pet + daemon)
tui:
    cd tui && go run .
