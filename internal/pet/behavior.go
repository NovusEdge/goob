package pet

import "math/rand"

type Pet struct {
	X, Y      int
	state     string
	anim      string // current animation name
	target    struct{ x, y int }
	timer     int
	screenW   int
	screenH   int
	frameW    int
	frameH    int
	variant   int // for picking idle/idle2, walk/walk2, etc
}

func New(screenW, screenH, frameW, frameH int) *Pet {
	return &Pet{
		X:       screenW / 2,
		Y:       screenH - frameH - 100,
		state:   "idle",
		anim:    "idle",
		screenW: screenW,
		screenH: screenH,
		frameW:  frameW,
		frameH:  frameH,
	}
}

func (p *Pet) Update(cursorX, cursorY int) {
	switch p.state {
	case "idle":
		p.anim = pick("idle", "idle2", p.variant)
		p.timer++
		if p.timer > 180 {
			p.timer = 0
			p.decideNextAction()
		}

	case "walk":
		p.anim = pick("walk", "walk2", p.variant)
		p.moveTowardTarget(2)

	case "chase":
		p.anim = "walk2"
		p.target.x, p.target.y = cursorX-p.frameW/2, cursorY-p.frameH/2
		p.moveTowardTarget(4)

	case "clean":
		p.anim = pick("clean", "clean2", p.variant)
		p.timer++
		if p.timer > 240 {
			p.timer = 0
			p.state = "idle"
		}

	case "sleep":
		p.anim = "sleep"
		p.timer++
		if p.timer > 400 {
			p.timer = 0
			p.state = "idle"
		}

	case "paw":
		p.anim = "paw"
		p.timer++
		if p.timer > 120 {
			p.timer = 0
			p.state = "idle"
		}

	case "jump":
		p.anim = "jump"
		p.Y -= 3
		p.timer++
		if p.timer > 30 {
			p.state = "fall"
			p.timer = 0
		}

	case "fall":
		p.anim = "jump"
		p.Y += 4
		if p.Y >= p.screenH-p.frameH {
			p.Y = p.screenH - p.frameH
			p.state = "idle"
		}

	case "scared":
		p.anim = "scared"
		p.timer++
		if p.timer > 90 {
			p.timer = 0
			p.state = "idle"
		}
	}

	p.clampPosition()
}

func (p *Pet) decideNextAction() {
	p.variant = rand.Intn(2)
	actions := []string{"walk", "clean", "sleep", "paw", "idle"}
	weights := []int{30, 20, 15, 15, 20} // percentages

	roll := rand.Intn(100)
	sum := 0
	for i, w := range weights {
		sum += w
		if roll < sum {
			p.state = actions[i]
			break
		}
	}

	if p.state == "walk" {
		p.target.x = rand.Intn(p.screenW - p.frameW)
		p.target.y = rand.Intn(p.screenH - p.frameH)
	}
}

func (p *Pet) moveTowardTarget(speed int) {
	dx := p.target.x - p.X
	dy := p.target.y - p.Y

	if abs(dx) < speed && abs(dy) < speed {
		p.state = "idle"
		p.timer = 0
		return
	}

	if dx > 0 {
		p.X += speed
	} else if dx < 0 {
		p.X -= speed
	}
	if dy > 0 {
		p.Y += speed
	} else if dy < 0 {
		p.Y -= speed
	}
}

func (p *Pet) clampPosition() {
	if p.X < 0 {
		p.X = 0
	}
	if p.X > p.screenW-p.frameW {
		p.X = p.screenW - p.frameW
	}
	if p.Y < 0 {
		p.Y = 0
	}
	if p.Y > p.screenH-p.frameH {
		p.Y = p.screenH - p.frameH
	}
}

func (p *Pet) Anim() string   { return p.anim }
func (p *Pet) Width() int     { return p.frameW }
func (p *Pet) Height() int    { return p.frameH }

// Scare triggers the scared animation
func (p *Pet) Scare() {
	if p.state != "scared" {
		p.state = "scared"
		p.timer = 0
	}
}

// Jump triggers a jump
func (p *Pet) Jump() {
	if p.state != "jump" && p.state != "fall" {
		p.state = "jump"
		p.timer = 0
	}
}

func pick(a, b string, variant int) string {
	if variant == 0 {
		return a
	}
	return b
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
