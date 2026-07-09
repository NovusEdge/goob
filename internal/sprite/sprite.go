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

// fallback is the canonical state graph: if a sheet lacks an animation, the
// engine walks these edges until it hits one the sheet defines. Only "idle" is
// truly required — everything degrades toward it. This is the whole BYO-sprite
// contract: declare the states you have, the rest gracefully substitute.
var fallback = map[string]string{
	"idle2":   "idle",
	"walk":    "idle",
	"walk2":   "walk",
	"run":     "walk",
	"pounce":  "paw",
	"paw":     "idle",
	"sit":     "idle",
	"sit2":    "sit",
	"loaf":    "sit",
	"sleep":   "loaf",
	"clean":   "idle",
	"clean2":  "clean",
	"stretch": "idle",
	"yawn":    "idle",
	"meow":    "idle",
	"roll":    "idle",
	"jump":    "idle",
	"scared":  "idle",
	"spawn":   "idle",
	"pickup":  "idle",
	"putdown": "idle",
	"held":    "sit",
	"held2":   "held",
}

// Resolve maps a canonical state to the closest animation a sheet defines,
// walking the fallback graph. has reports whether a given name exists. Used by
// both rendering backends so BYO fallback behaves identically everywhere.
func Resolve(name string, has func(string) bool) string {
	for i := 0; i < len(fallback)+1; i++ {
		if has(name) {
			return name
		}
		next, ok := fallback[name]
		if !ok {
			break
		}
		name = next
	}
	return "idle"
}

func (s *Sheet) resolve(name string) string { return Resolve(name, s.Has) }

func (s *Sheet) Update(state string) {
	a := s.anims[s.resolve(state)]
	if a == nil {
		return
	}

	a.tick++
	if a.fps > 0 && a.tick >= 60/a.fps {
		a.tick = 0
		a.frame = (a.frame + 1) % a.frames
	}
}

func (s *Sheet) Draw(state string, scale int, flipX bool) {
	a := s.anims[s.resolve(state)]
	if a == nil {
		return
	}

	srcW := float32(s.frameW)
	if flipX {
		srcW = -srcW // negative width flips horizontally
	}

	src := rl.Rectangle{
		X:      float32(a.frame * s.frameW),
		Y:      float32(a.row * s.frameH),
		Width:  srcW,
		Height: float32(s.frameH),
	}
	dst := rl.Rectangle{
		X:      0,
		Y:      0,
		Width:  float32(s.frameW * scale),
		Height: float32(s.frameH * scale),
	}
	rl.DrawTexturePro(s.texture, src, dst, rl.Vector2{}, 0, rl.White)
}

func (s *Sheet) FrameSize() (int, int) {
	return s.frameW, s.frameH
}

// LoopLen returns how many 60fps ticks one full loop of anim takes, matching
// the frame-advance cadence in Update. 0 if the anim is unknown.
func (s *Sheet) LoopLen(anim string) int {
	a := s.anims[s.resolve(anim)]
	if a == nil || a.fps <= 0 {
		return 0
	}
	return a.frames * (60 / a.fps)
}

// Has reports whether the sheet defines an animation for name.
func (s *Sheet) Has(name string) bool {
	_, ok := s.anims[name]
	return ok
}

func (s *Sheet) Unload() {
	rl.UnloadTexture(s.texture)
}
