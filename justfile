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

# run the LLM daemon (opt-in). Needs `pip install -r requirements.txt` and a
# provider key in the environment, e.g. OPENAI_API_KEY. GOOB_MODEL picks the model.
daemon:
    python3 -m daemon.server
