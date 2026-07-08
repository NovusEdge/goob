package sprite

import (
	"path/filepath"

	rl "github.com/gen2brain/raylib-go/raylib"
)

type Sheet struct {
	texture   rl.Texture2D
	frameW    int
	frameH    int
	anims     map[string]*Animation
}

type Animation struct {
	row    int
	frames int
	fps    int
	tick   int
	frame  int
}

func Load(manifestPath string) (*Sheet, error) {
	m, err := LoadManifest(manifestPath)
	if err != nil {
		return nil, err
	}

	dir := filepath.Dir(manifestPath)
	imgPath := filepath.Join(dir, m.Sheet)
	tex := rl.LoadTexture(imgPath)

	s := &Sheet{
		texture: tex,
		frameW:  m.FrameSize[0],
		frameH:  m.FrameSize[1],
		anims:   make(map[string]*Animation),
	}

	for name, a := range m.States {
		s.anims[name] = &Animation{
			row:    a.Row,
			frames: a.Frames,
			fps:    a.FPS,
		}
	}

	return s, nil
}

func (s *Sheet) Update(state string) {
	a, ok := s.anims[state]
	if !ok {
		a = s.anims["idle"]
	}
	if a == nil {
		return
	}

	a.tick++
	if a.fps > 0 && a.tick >= 60/a.fps {
		a.tick = 0
		a.frame = (a.frame + 1) % a.frames
	}
}

func (s *Sheet) Draw(state string) {
	a, ok := s.anims[state]
	if !ok {
		a = s.anims["idle"]
	}
	if a == nil {
		return
	}

	src := rl.Rectangle{
		X:      float32(a.frame * s.frameW),
		Y:      float32(a.row * s.frameH),
		Width:  float32(s.frameW),
		Height: float32(s.frameH),
	}
	dst := rl.Rectangle{
		X:      0,
		Y:      0,
		Width:  float32(s.frameW),
		Height: float32(s.frameH),
	}
	rl.DrawTexturePro(s.texture, src, dst, rl.Vector2{}, 0, rl.White)
}

func (s *Sheet) FrameSize() (int, int) {
	return s.frameW, s.frameH
}

func (s *Sheet) Unload() {
	rl.UnloadTexture(s.texture)
}
