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

# run the LLM daemon (opt-in). uv reads pyproject.toml for deps (litellm). Load
# your .env first (`set -a; source .env; set +a`) so the provider key is present.
daemon:
    uv run python -m daemon.server
