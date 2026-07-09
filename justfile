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
