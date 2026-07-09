package main

import (
	"github.com/NovusEdge/goob/internal/pet"
	"github.com/NovusEdge/goob/internal/sysmon"
)

// applyPointer maps the global pointer onto the pet: hold-drag with the left
// button while over the cat (and keep dragging until released), right-click to
// startle. Buttons: 1=left, 2=right (see getGlobalCursor).
func applyPointer(p *pet.Pet, x, y, buttons, w, h int) {
	over := x >= p.X && x < p.X+w && y >= p.Y && y < p.Y+h
	left := buttons&1 != 0
	right := buttons&2 != 0
	switch {
	case left && (over || p.Held()):
		p.Hold(x, y)
	case p.Held():
		p.Release()
	case right && over:
		p.Scare()
	}
}

// moodFrom derives the pet's disposition from system state. A stressed machine
// (hot or nearly dead) makes the cat tired; a busy one makes it alert.
func moodFrom(s sysmon.State) pet.Mood {
	if (s.BatteryPct >= 0 && s.BatteryPct < 15 && !s.Charging) || s.TempC >= 85 {
		return pet.MoodTired
	}
	if s.Building {
		return pet.MoodAlert
	}
	return pet.MoodNeutral
}
