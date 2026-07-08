package main

import (
	"github.com/NovusEdge/goob/internal/pet"
	"github.com/NovusEdge/goob/internal/window"
)

func main() {
	win := window.New()
	defer win.Close()

	p := pet.New()

	for !win.ShouldClose() {
		p.Update(win.CursorPos())
		win.SetPosition(p.X, p.Y)
		win.SetSize(p.Width(), p.Height())
		win.Draw(p.Frame())
	}
}
