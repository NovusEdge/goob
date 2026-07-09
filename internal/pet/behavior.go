package pet

import "math/rand"

type Mood int

const (
	MoodNeutral Mood = iota
	MoodAlert         // something's building/running -> watchful, active
	MoodTired         // hot CPU or low battery -> sleepy, sluggish
)

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
	mood       Mood
	loopFn     func(string) int // anim -> frames for one full loop; set from the sheet

	prevCursorX int
	cursorDir   int  // -1/0/+1, last significant cursor x direction
	cursorSeen  bool // prevCursorX is valid
	jiggle      int  // decaying score; direction reversals push it up
}

// holdStates are transient states that just play an animation for a while, then
// return to idle. loops = how many full animation cycles to play. Deriving the
// frame count from the actual animation (via loopFn) means an animation is never
// cut off mid-loop the way hand-tuned constants used to do.
var holdStates = map[string]struct {
	anim, anim2 string
	loops       int
}{
	"spawn":   {"spawn", "", 1},
	"pounce":  {"pounce", "", 1},
	"scared":  {"scared", "", 2},
	"paw":     {"paw", "", 3},
	"stretch": {"stretch", "", 1},
	"yawn":    {"yawn", "", 1},
	"meow":    {"meow", "", 2},
	"roll":    {"roll", "", 1},
	"clean":   {"clean", "clean2", 3},
	"sit":     {"sit", "sit2", 3},
	"loaf":    {"loaf", "", 2},
	"sleep":   {"sleep", "", 2},
	"putdown": {"putdown", "", 1},
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

// SetLoopFn wires in the sheet's per-animation loop length so hold-state
// durations track the real animation frame counts.
func (p *Pet) SetLoopFn(f func(string) int) { p.loopFn = f }

func (p *Pet) loopLen(anim string) int {
	if p.loopFn == nil {
		return 60 // ponytail: ~1s fallback when sheet lengths aren't wired (e.g. tests)
	}
	if n := p.loopFn(anim); n > 0 {
		return n
	}
	return 60
}

func (p *Pet) Update(cursorX, cursorY int) {
	ground := p.screenH - p.frameH

	p.detectJiggle(cursorX)

	if h, ok := holdStates[p.state]; ok {
		p.anim = h.anim
		if h.anim2 != "" && p.variant == 1 {
			p.anim = h.anim2
		}
		p.velX = 0
		p.timer++
		if p.timer >= h.loops*p.loopLen(h.anim) {
			p.timer = 0
			p.state = "idle"
		}
	} else {
		switch p.state {
		case "idle":
			p.anim = pick("idle", "idle2", p.variant)
			p.velX = 0
			p.timer++
			if p.timer > p.idleDelay() {
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
			p.anim = "run"
			if cursorX >= 0 && cursorY >= 0 {
				dx := cursorX - p.X - p.frameW/2
				if abs(dx) < 30 {
					p.state = "pounce" // close enough - pounce!
					p.timer = 0
					p.velX = 0
				} else if dx > 0 {
					p.velX = 5
					p.FacingLeft = false
				} else {
					p.velX = -5
					p.FacingLeft = true
				}
				p.timer++
				if p.timer > 300 {
					p.state = "idle"
					p.timer = 0
				}
			} else {
				p.state = "idle"
			}

		case "jump":
			p.anim = "jump"
			p.velY = -8
			p.state = "airborne"

		case "airborne":
			p.anim = "jump"
			// gravity handled below

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
		}
	}

	// apply velocities
	p.X += p.velX

	// gravity — but a carried cat hangs from the cursor, it doesn't fall
	if !p.Held() {
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
	}

	p.clampPosition()
}

// detectJiggle watches for the cursor shaking back and forth — a "hey, come
// here" gesture — and summons the cat into a playful chase. Each significant
// x-direction reversal bumps a score that decays every frame, so only reversals
// arriving in quick succession accumulate to the trigger.
func (p *Pet) detectJiggle(cx int) {
	if cx < 0 { // no cursor this frame (e.g. Wayland/no display)
		p.jiggle, p.cursorSeen, p.cursorDir = 0, false, 0
		return
	}
	if !p.cursorSeen {
		p.cursorSeen, p.prevCursorX = true, cx
		return
	}

	dx := cx - p.prevCursorX
	p.prevCursorX = cx
	if p.jiggle > 0 {
		p.jiggle--
	}
	if abs(dx) < 4 { // ignore slow drift; only fast motion counts as a shake
		return
	}

	dir := 1
	if dx < 0 {
		dir = -1
	}
	if p.cursorDir != 0 && dir != p.cursorDir {
		p.jiggle += 3 // reversal outpaces the per-frame decay
	}
	p.cursorDir = dir

	if p.jiggle >= 12 && p.Grounded() && p.interruptible() {
		p.state, p.timer, p.jiggle = "chase", 0, 0
	}
}

// idleDelay is how long to loiter in idle before picking a new action. A tired
// cat dawdles; an alert cat fidgets sooner.
func (p *Pet) idleDelay() int {
	switch p.mood {
	case MoodAlert:
		return 45
	case MoodTired:
		return 150
	default:
		return 90
	}
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

	// Mood reshapes what the cat feels like doing.
	switch p.mood {
	case MoodAlert: // busy machine -> pace, watch, bat at things
		weights = []int{35, 20, 18, 6, 2, 0, 8, 6, 2, 1, 0, 6, 2, 4}
	case MoodTired: // hot / low battery -> flop and doze
		weights = []int{8, 5, 0, 12, 6, 22, 2, 0, 4, 10, 15, 2, 2, 12}
	}

	total := 0
	for _, w := range weights {
		total += w
	}
	roll := rand.Intn(total)
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
	}
}

// SetMood updates the cat's disposition and reacts once on a change:
// slipping into a tired mood makes it yawn, snapping to alert perks it up.
func (p *Pet) SetMood(m Mood) {
	if m == p.mood {
		return
	}
	p.mood = m
	if !p.Grounded() || !p.interruptible() {
		return
	}
	switch m {
	case MoodTired:
		p.state, p.timer = "yawn", 0
	case MoodAlert:
		p.state, p.timer = "meow", 0
	}
}

// interruptible reports whether a spontaneous reaction may hijack the state.
func (p *Pet) interruptible() bool {
	switch p.state {
	case "held", "pickup", "putdown", "scared", "chase", "pounce":
		return false
	}
	return true
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

// Held reports whether the cat is currently being picked up or carried.
func (p *Pet) Held() bool {
	return p.state == "held" || p.state == "pickup"
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
