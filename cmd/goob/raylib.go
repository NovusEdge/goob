package main

import (
	"log"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/NovusEdge/goob/internal/pet"
	"github.com/NovusEdge/goob/internal/sprite"
	"github.com/NovusEdge/goob/internal/sysmon"
)

func runRaylib(manifestPath string, scale int, newPet func(int, int, int, int) *pet.Pet) {
	rl.SetTraceLogLevel(rl.LogNone)

	// detect screen size
	rl.SetConfigFlags(rl.FlagWindowHidden)
	rl.InitWindow(1, 1, "")
	screenW := rl.GetMonitorWidth(rl.GetCurrentMonitor())
	screenH := rl.GetMonitorHeight(rl.GetCurrentMonitor())
	rl.CloseWindow()

	// create transparent overlay window
	rl.SetConfigFlags(rl.FlagWindowTransparent | rl.FlagWindowUndecorated | rl.FlagWindowTopmost | rl.FlagWindowMousePassthrough)
	rl.InitWindow(64, 64, "goob - lil vro 🥀")
	rl.SetTargetFPS(60)
	defer rl.CloseWindow()

	sheet, err := sprite.Load(manifestPath)
	if err != nil {
		log.Fatal(err)
	}
	defer sheet.Unload()

	frameW, frameH := sheet.FrameSize()
	scaledW, scaledH := frameW*scale, frameH*scale
	p := newPet(screenW, screenH, scaledW, scaledH)
	p.SetLoopFn(sheet.LoopLen)

	frame := 0
	for !rl.WindowShouldClose() {
		if frame%120 == 0 { // ~every 2s: sample the machine's mood
			p.SetMood(moodFrom(sysmon.Read()))
		}
		frame++

		// get global cursor position + button state
		cursorX, cursorY, buttons := getGlobalCursor()
		applyPointer(p, cursorX, cursorY, buttons, scaledW, scaledH)

		p.Update(cursorX, cursorY)
		sheet.Update(p.Anim())

		rl.SetWindowPosition(p.X, p.Y)
		rl.SetWindowSize(scaledW, scaledH)

		rl.BeginDrawing()
		rl.ClearBackground(rl.Blank)
		sheet.Draw(p.Anim(), scale, p.FacingLeft)
		rl.EndDrawing()
	}
}

// getGlobalCursor is implemented in cursor_*.go per platform
