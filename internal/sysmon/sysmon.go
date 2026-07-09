// Package sysmon reads coarse system state from /proc and /sys so the pet can
// react to what the machine is doing. Linux-only; Read() returns a zero State
// on other platforms (all fields harmless defaults).
package sysmon

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type State struct {
	Building   bool // a build/dev process is running (see Watch)
	BatteryPct int  // 0-100, -1 if no battery
	Charging   bool
	TempC      int // hottest thermal zone in °C, 0 if unknown
}

// Watch is the process-name watchlist that flips Building on. Substring match
// against /proc/*/comm. Exported so callers can override for their own tools.
var Watch = []string{"go", "gcc", "cc1", "clang", "rustc", "cargo",
	"node", "npm", "webpack", "tsc", "make", "cmake", "ninja",
	"docker", "gradle", "mvn", "python", "ld"}

func Read() State {
	return State{
		Building:   building(),
		BatteryPct: batteryPct(),
		Charging:   charging(),
		TempC:      hottestZone(),
	}
}

func building() bool {
	procs, _ := filepath.Glob("/proc/[0-9]*/comm")
	for _, p := range procs {
		b, err := os.ReadFile(p)
		if err != nil {
			continue // process vanished mid-scan
		}
		name := strings.TrimSpace(string(b))
		for _, w := range Watch {
			if name == w {
				return true
			}
		}
	}
	return false
}

func batteryPct() int {
	bats, _ := filepath.Glob("/sys/class/power_supply/BAT*/capacity")
	if len(bats) == 0 {
		return -1
	}
	return readInt(bats[0], -1)
}

func charging() bool {
	stats, _ := filepath.Glob("/sys/class/power_supply/BAT*/status")
	for _, s := range stats {
		b, err := os.ReadFile(s)
		if err == nil && strings.TrimSpace(string(b)) != "Discharging" {
			return true
		}
	}
	return false
}

func hottestZone() int {
	zones, _ := filepath.Glob("/sys/class/thermal/thermal_zone*/temp")
	max := 0
	for _, z := range zones {
		milli := readInt(z, 0) // millidegrees C
		if c := milli / 1000; c > max {
			max = c
		}
	}
	return max
}

func readInt(path string, def int) int {
	b, err := os.ReadFile(path)
	if err != nil {
		return def
	}
	n, err := strconv.Atoi(strings.TrimSpace(string(b)))
	if err != nil {
		return def
	}
	return n
}
