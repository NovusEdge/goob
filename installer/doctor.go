package main

import (
	"bufio"
	"os"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// Dep represents a dependency check result.
type Dep struct {
	Name     string
	Found    bool
	Path     string
	Version  string
	Ok       bool   // meets minimum version
	Required bool   // vs optional
	Hint     string // install command or URL
}

// MinVersions defines minimum acceptable versions.
var MinVersions = map[string]string{
	"godot":   "4.7",
	"python3": "3.0",
}

// CheckDeps runs all dependency probes.
func CheckDeps() []Dep {
	return []Dep{
		checkGodot(),
		checkPython(),
		checkUV(),
		checkGo(),
	}
}

func checkGodot() Dep {
	d := Dep{Name: "godot", Required: true}
	// check $GODOT env first
	if envPath := os.Getenv("GODOT"); envPath != "" {
		if _, err := os.Stat(envPath); err == nil {
			d.Found = true
			d.Path = envPath
			d.Version = parseGodotVersion(envPath)
		}
	}
	if !d.Found {
		if p, err := exec.LookPath("godot"); err == nil {
			d.Found = true
			d.Path = p
			d.Version = parseGodotVersion(p)
		}
	}
	d.Ok = d.Found && versionGE(d.Version, MinVersions["godot"])
	if !d.Found || !d.Ok {
		d.Hint = "Set GODOT=/path/to/godot or download from https://godotengine.org"
	}
	return d
}

func parseGodotVersion(path string) string {
	out, err := exec.Command(path, "--version").Output()
	if err != nil {
		return ""
	}
	// e.g. "4.7.stable.official"
	parts := strings.Split(strings.TrimSpace(string(out)), ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return strings.TrimSpace(string(out))
}

func checkPython() Dep {
	d := Dep{Name: "python3", Required: true}
	for _, cmd := range []string{"python3", "python"} {
		if p, err := exec.LookPath(cmd); err == nil {
			d.Found = true
			d.Path = p
			d.Version = parsePythonVersion(p)
			break
		}
	}
	d.Ok = d.Found && versionGE(d.Version, MinVersions["python3"])
	if !d.Found {
		d.Hint = installHint("python3")
	}
	return d
}

func parsePythonVersion(path string) string {
	out, err := exec.Command(path, "--version").Output()
	if err != nil {
		return ""
	}
	// "Python 3.12.0"
	s := strings.TrimPrefix(strings.TrimSpace(string(out)), "Python ")
	parts := strings.Split(s, ".")
	if len(parts) >= 2 {
		return parts[0] + "." + parts[1]
	}
	return s
}

func checkUV() Dep {
	d := Dep{Name: "uv", Required: false}
	if p, err := exec.LookPath("uv"); err == nil {
		d.Found = true
		d.Path = p
		d.Ok = true
	} else {
		d.Hint = "curl -LsSf https://astral.sh/uv/install.sh | sh"
	}
	return d
}

func checkGo() Dep {
	d := Dep{Name: "go", Required: false}
	if p, err := exec.LookPath("go"); err == nil {
		d.Found = true
		d.Path = p
		d.Ok = true
	} else {
		d.Hint = installHint("go")
	}
	return d
}

// installHint returns an OS-specific install command.
func installHint(pkg string) string {
	switch runtime.GOOS {
	case "darwin":
		return "brew install " + pkg
	case "linux":
		return linuxInstallHint(pkg)
	default:
		return "Install " + pkg + " from your package manager"
	}
}

func linuxInstallHint(pkg string) string {
	id := osReleaseID()
	switch {
	case id == "arch" || id == "manjaro":
		return "pacman -S " + pkg
	case id == "debian" || id == "ubuntu" || strings.Contains(id, "debian"):
		return "apt install " + pkg
	case id == "fedora" || id == "rhel" || id == "centos":
		return "dnf install " + pkg
	default:
		return "Install " + pkg + " from your package manager"
	}
}

func osReleaseID() string {
	f, err := os.Open("/etc/os-release")
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "ID=") {
			return strings.Trim(strings.TrimPrefix(line, "ID="), `"`)
		}
		if strings.HasPrefix(line, "ID_LIKE=") {
			return strings.Trim(strings.TrimPrefix(line, "ID_LIKE="), `"`)
		}
	}
	return ""
}

// versionGE returns true if v >= min (semver-ish comparison).
var versionRe = regexp.MustCompile(`^(\d+)\.(\d+)`)

func versionGE(v, min string) bool {
	vm := versionRe.FindStringSubmatch(v)
	mm := versionRe.FindStringSubmatch(min)
	if vm == nil || mm == nil {
		return false
	}
	vMaj, _ := strconv.Atoi(vm[1])
	vMin, _ := strconv.Atoi(vm[2])
	mMaj, _ := strconv.Atoi(mm[1])
	mMin, _ := strconv.Atoi(mm[2])
	if vMaj != mMaj {
		return vMaj > mMaj
	}
	return vMin >= mMin
}
