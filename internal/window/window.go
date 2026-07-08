package window

import rl "github.com/gen2brain/raylib-go/raylib"

type Window struct {
	// ponytail: raylib handles the window internally
}

func New() *Window {
	rl.SetConfigFlags(rl.FlagWindowTransparent | rl.FlagWindowUndecorated | rl.FlagWindowTopmost)
	rl.InitWindow(64, 64, "goob")
	rl.SetTargetFPS(60)
	return &Window{}
}

func (w *Window) Close() {
	rl.CloseWindow()
}

func (w *Window) ShouldClose() bool {
	return rl.WindowShouldClose()
}

func (w *Window) CursorPos() (int, int) {
	// ponytail: raylib gives window-relative pos, need screen-relative via platform call later
	pos := rl.GetMousePosition()
	return int(pos.X), int(pos.Y)
}

func (w *Window) SetPosition(x, y int) {
	rl.SetWindowPosition(x, y)
}

func (w *Window) SetSize(width, height int) {
	rl.SetWindowSize(width, height)
}

func (w *Window) Draw(frame any) {
	rl.BeginDrawing()
	rl.ClearBackground(rl.Blank)
	// ponytail: placeholder, draw actual sprite frame here
	rl.DrawRectangle(0, 0, 64, 64, rl.Red)
	rl.EndDrawing()
}
