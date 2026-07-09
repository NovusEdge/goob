package pet

import "math/rand"

type Pet struct {
	X, Y       int
	velX, velY int
	FacingLeft bool
	state      string
	anim       string
	target     struct{ x, y int }
	timer      int
	screenW    int
	screenH    int
	frameW     int
	frameH     int
	variant    int
}

func New(screenW, screenH, frameW, frameH int) *Pet {
	return &Pet{
		X:       screenW / 2,
		Y:       screenH - frameH,
		state:   "spawn",
		anim:    "spawn",
		screenW: screenW,
		screenH: screenH,
		frameW:  frameW,
		frameH:  frameH,
	}
}

func (p *Pet) Update(cursorX, cursorY int) {
	ground := p.screenH - p.frameH

	switch p.state {
	case "idle":
		p.anim = pick("idle", "idle2", p.variant)
		p.velX = 0
		p.timer++
		if p.timer > 90 { // faster action picking
			p.timer = 0
			p.decideNextAction()
		}

	case "walk":
		p.anim = "walk"
		dx := p.target.x - p.X
		if abs(dx) < 4 {
			p.state = "idle"
			p.velX = 0
			p.timer = 0
		} else if dx > 0 {
			p.velX = 1
			p.FacingLeft = false
		} else {
			p.velX = -1
			p.FacingLeft = true
		}

	case "chase":
		p.anim = "walk2"
		if cursorX >= 0 && cursorY >= 0 {
			dx := cursorX - p.X - p.frameW/2
			if abs(dx) < 20 {
				p.velX = 0
			} else if dx > 0 {
				p.velX = 4
				p.FacingLeft = false
			} else {
				p.velX = -4
				p.FacingLeft = true
			}
			// chase timer - give up after a while
			p.timer++
			if p.timer > 300 {
				p.state = "idle"
				p.timer = 0
			}
		} else {
			p.state = "idle"
		}

	case "clean":
		p.anim = pick("clean", "clean2", p.variant)
		p.velX = 0
		p.timer++
		if p.timer > 240 {
			p.timer = 0
			p.state = "idle"
		}

	case "sleep":
		p.anim = "sleep"
		p.velX = 0
		p.timer++
		if p.timer > 400 {
			p.timer = 0
			p.state = "idle"
		}

	case "paw":
		p.anim = "paw"
		p.velX = 0
		p.timer++
		if p.timer > 120 {
			p.timer = 0
			p.state = "idle"
		}

	case "jump":
		p.anim = "jump"
		p.velY = -8
		p.state = "airborne"

	case "airborne":
		p.anim = "jump"
		// gravity handled below

	case "scared":
		p.anim = "scared"
		p.velX = 0
		p.timer++
		if p.timer > 90 {
			p.timer = 0
			p.state = "idle"
		}

	case "sit":
		p.anim = pick("sit", "sit2", p.variant)
		p.velX = 0
		p.timer++
		if p.timer > 300 {
			p.timer = 0
			p.state = "idle"
		}

	case "loaf":
		p.anim = "loaf"
		p.velX = 0
		p.timer++
		if p.timer > 400 {
			p.timer = 0
			p.state = "idle"
		}

	case "stretch":
		p.anim = "stretch"
		p.velX = 0
		p.timer++
		if p.timer > 120 {
			p.timer = 0
			p.state = "idle"
		}

	case "yawn":
		p.anim = "yawn"
		p.velX = 0
		p.timer++
		if p.timer > 120 {
			p.timer = 0
			p.state = "idle"
		}

	case "meow":
		p.anim = "meow"
		p.velX = 0
		p.timer++
		if p.timer > 90 {
			p.timer = 0
			p.state = "idle"
		}

	case "roll":
		p.anim = "roll"
		p.velX = 0
		p.timer++
		if p.timer > 150 {
			p.timer = 0
			p.state = "idle"
		}

	case "pickup":
		p.anim = "pickup"
		p.velX = 0
		p.velY = 0
		p.timer++
		if p.timer > 36 { // 6 frames at 8fps
			p.state = "held"
			p.timer = 0
		}

	case "held":
		p.anim = pick("held", "held2", p.variant)
		p.velX = 0
		p.velY = 0

	case "putdown":
		p.anim = "putdown"
		p.velX = 0
		p.timer++
		if p.timer > 36 {
			p.state = "idle"
			p.timer = 0
		}

	case "spawn":
		p.anim = "spawn"
		p.velX = 0
		p.timer++
		if p.timer > 60 { // ~1 sec at 60fps
			p.state = "idle"
			p.timer = 0
		}
	}

	// apply velocities
	p.X += p.velX

	// gravity
	if p.Y < ground {
		p.velY += 1
		if p.velY > 10 {
			p.velY = 10
		}
	} else {
		p.Y = ground
		if p.velY > 0 {
			p.velY = 0
			if p.state == "airborne" {
				p.state = "idle"
			}
		}
	}
	p.Y += p.velY

	p.clampPosition()
}

func (p *Pet) decideNextAction() {
	p.variant = rand.Intn(2)

	if !p.Grounded() {
		return
	}

	actions := []string{
		"walk", "walk", "chase", // movement
		"sit", "clean", "sleep", "paw", "jump",
		"stretch", "yawn", "loaf", "meow", "roll", "idle",
	}
	weights := []int{
		25, 15, 10, // 50% movement total
		8, 6, 5, 5, 5,
		4, 4, 4, 3, 3, 8,
	}

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
		// pick random X target, stay on ground
		p.target.x = rand.Intn(p.screenW - p.frameW)
	}
}

func (p *Pet) clampPosition() {
	if p.X < 0 {
		p.X = 0
		p.velX = 0
	}
	if p.X > p.screenW-p.frameW {
		p.X = p.screenW - p.frameW
		p.velX = 0
	}
	if p.Y < 0 {
		p.Y = 0
		p.velY = 0
	}
	if p.Y > p.screenH-p.frameH {
		p.Y = p.screenH - p.frameH
		p.velY = 0
	}
}

func (p *Pet) Grounded() bool {
	return p.Y >= p.screenH-p.frameH
}

func (p *Pet) Anim() string { return p.anim }
func (p *Pet) Width() int   { return p.frameW }
func (p *Pet) Height() int  { return p.frameH }

func (p *Pet) Scare() {
	if p.state != "scared" && p.state != "held" && p.state != "pickup" && p.state != "putdown" {
		p.state = "scared"
		p.timer = 0
	}
}

func (p *Pet) Jump() {
	if p.Grounded() {
		p.state = "jump"
		p.timer = 0
	}
}

func (p *Pet) Hold(x, y int) {
	if p.state != "held" && p.state != "pickup" {
		p.state = "pickup"
		p.timer = 0
		p.variant = rand.Intn(2)
	}
	p.X = x - p.frameW/2
	p.Y = y - p.frameH/2
}

func (p *Pet) Release() {
	if p.state == "held" || p.state == "pickup" {
		p.state = "putdown"
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
