package main

import (
	"flag"
	"log"
	"os"
	"runtime"

	rl "github.com/gen2brain/raylib-go/raylib"

	"github.com/NovusEdge/goob/internal/pet"
	"github.com/NovusEdge/goob/internal/sprite"
)

func main() {
	manifestPath := flag.String("manifest", "assets/cat-sprites.json", "path to sprite manifest")
	scale := flag.Int("scale", 8, "sprite scale factor")
	flag.Parse()

	// suppress all raylib logs
	rl.SetTraceLogLevel(rl.LogNone)

	// detect screen size
	// ponytail: hardcoded fallback, proper detection needs platform code
	screenW, screenH := 1920, 1080
	if runtime.GOOS != "linux" || os.Getenv("WAYLAND_DISPLAY") == "" {
		rl.SetConfigFlags(rl.FlagWindowHidden)
		rl.InitWindow(1, 1, "")
		screenW = rl.GetMonitorWidth(rl.GetCurrentMonitor())
		screenH = rl.GetMonitorHeight(rl.GetCurrentMonitor())
		rl.CloseWindow()
	}

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
	scaledW, scaledH := frameW*(*scale), frameH*(*scale)
	p := pet.New(screenW, screenH, scaledW, scaledH)

	// check if we can move windows (X11) or not (Wayland)
	canMove := os.Getenv("WAYLAND_DISPLAY") == ""

	for !rl.WindowShouldClose() {
		p.Update(0, 0)
		sheet.Update(p.Anim())

		if canMove {
			rl.SetWindowPosition(p.X, p.Y)
		}
		rl.SetWindowSize(scaledW, scaledH)

		rl.BeginDrawing()
		rl.ClearBackground(rl.Blank)
		sheet.Draw(p.Anim(), *scale)
		rl.EndDrawing()
	}
}
