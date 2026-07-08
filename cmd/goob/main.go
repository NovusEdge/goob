package main

import (
	"flag"
	"os"
	"runtime"

	"github.com/NovusEdge/goob/internal/pet"
)

func main() {
	manifestPath := flag.String("manifest", "assets/cat-sprites.json", "path to sprite manifest")
	scale := flag.Int("scale", 6, "sprite scale factor")
	backend := flag.String("backend", "auto", "window backend: auto, raylib, wayland")
	flag.Parse()

	useWayland := *backend == "wayland" ||
		(*backend == "auto" && runtime.GOOS == "linux" && os.Getenv("WAYLAND_DISPLAY") != "")

	if useWayland && runtime.GOOS == "linux" {
		runWayland(*manifestPath, *scale, pet.New)
	} else {
		runRaylib(*manifestPath, *scale, pet.New)
	}
}
