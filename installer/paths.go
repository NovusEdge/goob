package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
)

// repoRootOverride is set by --repo-root flag
var repoRootOverride string

// ExpandPath expands ~ to the user's home directory.
func ExpandPath(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(home, path[2:])
	}
	return path
}

// RepoRoot returns the absolute path to the goob repository root.
func RepoRoot() (string, error) {
	// flag override takes precedence
	if repoRootOverride != "" {
		abs, err := filepath.Abs(repoRootOverride)
		if err != nil {
			return "", err
		}
		if _, err := os.Stat(filepath.Join(abs, "hooks", "goob_hook.py")); err != nil {
			return "", errors.New("--repo-root does not contain hooks/goob_hook.py")
		}
		return abs, nil
	}
	// try CWD (if running from repo root)
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(filepath.Join(cwd, "hooks", "goob_hook.py")); err == nil {
		return cwd, nil
	}
	// try parent (in case running from installer/)
	parent := filepath.Dir(cwd)
	if _, err := os.Stat(filepath.Join(parent, "hooks", "goob_hook.py")); err == nil {
		return parent, nil
	}
	// try relative to executable (for shipped binary)
	exe, err := os.Executable()
	if err == nil {
		exeDir := filepath.Dir(exe)
		// binary in installer/, repo is parent
		if _, err := os.Stat(filepath.Join(filepath.Dir(exeDir), "hooks", "goob_hook.py")); err == nil {
			return filepath.Dir(exeDir), nil
		}
		// binary in repo root
		if _, err := os.Stat(filepath.Join(exeDir, "hooks", "goob_hook.py")); err == nil {
			return exeDir, nil
		}
	}
	return "", errors.New("cannot find goob repo; run from repo dir or use --repo-root")
}

// HookPath returns the absolute path to goob_hook.py.
func HookPath() (string, error) {
	root, err := RepoRoot()
	if err != nil {
		return "", err
	}
	p := filepath.Join(root, "hooks", "goob_hook.py")
	if _, err := os.Stat(p); err != nil {
		return "", err
	}
	return p, nil
}

// CodexDispatcherPath returns the absolute path to goob_codex_notify.py.
func CodexDispatcherPath() (string, error) {
	root, err := RepoRoot()
	if err != nil {
		return "", err
	}
	p := filepath.Join(root, "hooks", "goob_codex_notify.py")
	if _, err := os.Stat(p); err != nil {
		return "", err
	}
	return p, nil
}

// EnvPath returns the path to the .env file in the repo root.
func EnvPath() (string, error) {
	root, err := RepoRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, ".env"), nil
}
