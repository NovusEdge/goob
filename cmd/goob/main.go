package main

import (
	"flag"
	"log"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/NovusEdge/goob/internal/pet"
	"github.com/NovusEdge/goob/internal/sprite"
)

func main() {
	manifestPath := flag.String("manifest", "assets/cat-sprites.json", "path to sprite manifest")
	flag.Parse()

	// get screen size before creating window
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

	sheet, err := sprite.Load(*manifestPath)
	if err != nil {
		log.Fatal(err)
	}
	defer sheet.Unload()

	frameW, frameH := sheet.FrameSize()
	p := pet.New(screenW, screenH, frameW, frameH)

	for !rl.WindowShouldClose() {
		// ponytail: GetMousePosition is window-relative, need screen coords
		// this is a platform-specific problem we'll solve later
		mouseX, mouseY := int(rl.GetMousePosition().X)+p.X, int(rl.GetMousePosition().Y)+p.Y

		p.Update(mouseX, mouseY)
		sheet.Update(p.Anim())

		rl.SetWindowPosition(p.X, p.Y)
		rl.SetWindowSize(frameW, frameH)

		rl.BeginDrawing()
		rl.ClearBackground(rl.Blank)
		sheet.Draw(p.Anim())
		rl.EndDrawing()
	}
}
