package pet

import "math/rand"

type State int

const (
	Idle State = iota
	Walking
	Sitting
	Sleeping
	Chasing
	Falling
)

type Pet struct {
	X, Y       int
	state      State
	sprite     *Sprite
	target     struct{ x, y int }
	idleTimer  int
	screenW    int
	screenH    int
}

func New() *Pet {
	// ponytail: hardcoded screen size, detect at runtime later
	return &Pet{
		X:       100,
		Y:       100,
		state:   Idle,
		screenW: 1920,
		screenH: 1080,
	}
}

func (p *Pet) Update(cursorX, cursorY int) {
	switch p.state {
	case Idle:
		p.idleTimer++
		if p.idleTimer > 180 { // ~3 sec at 60fps
			p.idleTimer = 0
			p.decideNextAction()
		}
	case Walking:
		p.moveTowardTarget()
	case Chasing:
		p.target.x, p.target.y = cursorX, cursorY
		p.moveTowardTarget()
	case Sitting, Sleeping:
		// chill
	case Falling:
		p.Y += 4
		if p.Y >= p.screenH-64 {
			p.Y = p.screenH - 64
			p.state = Idle
		}
	}
}

func (p *Pet) decideNextAction() {
	switch rand.Intn(4) {
	case 0:
		p.state = Walking
		p.target.x = rand.Intn(p.screenW - 64)
		p.target.y = rand.Intn(p.screenH - 64)
	case 1:
		p.state = Sitting
	case 2:
		p.state = Chasing
	default:
		p.state = Idle
	}
}

func (p *Pet) moveTowardTarget() {
	dx := p.target.x - p.X
	dy := p.target.y - p.Y

	if abs(dx) < 4 && abs(dy) < 4 {
		p.state = Idle
		return
	}

	if dx > 0 {
		p.X += 2
	} else if dx < 0 {
		p.X -= 2
	}
	if dy > 0 {
		p.Y += 2
	} else if dy < 0 {
		p.Y -= 2
	}
}

func (p *Pet) Width() int  { return 64 } // ponytail: fixed size, dynamic from sprite bounds later
func (p *Pet) Height() int { return 64 }
func (p *Pet) Frame() any  { return nil } // placeholder, returns current sprite frame
func (p *Pet) State() State { return p.state }

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
