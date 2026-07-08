package main

import (
	"flag"
	"log"
	"os"
	"runtime"

	"github.com/NovusEdge/goob/internal/pet"
)

func main() {
	manifestPath := flag.String("manifest", "assets/cat-sprites.json", "path to sprite manifest")
	scale := flag.Int("scale", 8, "sprite scale factor")
	backend := flag.String("backend", "auto", "window backend: auto, raylib, wayland")
	flag.Parse()

	useWayland := *backend == "wayland" ||
		(*backend == "auto" && runtime.GOOS == "linux" && os.Getenv("WAYLAND_DISPLAY") != "")

	if useWayland {
		log.Println("wayland backend not yet implemented, falling back to raylib")
		log.Println("(window positioning won't work on wayland)")
	}

	runRaylib(*manifestPath, *scale, pet.New)
}
