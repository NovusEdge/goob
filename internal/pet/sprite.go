package pet

// Sprite holds animation data for a pet state
type Sprite struct {
	frames    []Frame
	current   int
	fps       int
	tick      int
}

type Frame struct {
	X, Y, W, H int // region in spritesheet
}

func (s *Sprite) Update() {
	if s == nil || len(s.frames) == 0 {
		return
	}
	s.tick++
	if s.tick >= 60/s.fps {
		s.tick = 0
		s.current = (s.current + 1) % len(s.frames)
	}
}

func (s *Sprite) Current() Frame {
	if s == nil || len(s.frames) == 0 {
		return Frame{}
	}
	return s.frames[s.current]
}
