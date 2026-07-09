package pet

import "testing"

// loopLen stub: every anim is 10 ticks long, so hold durations are predictable.
func newTestPet() *Pet {
	p := New(200, 100, 20, 20) // ground = 80
	p.SetLoopFn(func(string) int { return 10 })
	return p
}

func step(p *Pet, n int) {
	for i := 0; i < n; i++ {
		p.Update(-1, -1)
	}
}

func TestSpawnPlaysFullLoopThenIdles(t *testing.T) {
	p := newTestPet() // starts in "spawn", loops=1, loopLen=10
	step(p, 9)
	if p.state != "spawn" {
		t.Fatalf("spawn ended early at %q", p.state)
	}
	step(p, 1) // 10th tick completes the loop
	if p.state != "idle" {
		t.Fatalf("after full spawn loop want idle, got %q", p.state)
	}
}

func TestHoldStatesReturnToIdle(t *testing.T) {
	for name, h := range holdStates {
		p := newTestPet()
		p.state = name
		p.anim = name
		step(p, h.loops*10+1)
		if p.state != "idle" {
			t.Errorf("%s (loops=%d) did not return to idle, stuck at %q", name, h.loops, p.state)
		}
	}
}

func TestTiredMoodYawns(t *testing.T) {
	p := newTestPet()
	step(p, 11) // clear spawn -> idle
	p.SetMood(MoodTired)
	if p.state != "yawn" {
		t.Fatalf("becoming tired should yawn, got %q", p.state)
	}
}

func TestCursorJiggleTriggersChase(t *testing.T) {
	p := newTestPet()
	step(p, 11) // clear spawn -> idle
	// shake the cursor: alternate far left/right, fast, several times
	x := 100
	summoned := func() bool { return p.state == "chase" || p.state == "pounce" }
	for i := 0; i < 12 && !summoned(); i++ {
		if i%2 == 0 {
			x += 20
		} else {
			x -= 20
		}
		p.Update(x, 50)
	}
	// chase may resolve straight to pounce when the cursor is on top of the cat;
	// either means the jiggle summoned it.
	if !summoned() {
		t.Fatalf("jiggling the cursor should summon the cat, got %q", p.state)
	}
}

func TestSteadyCursorDoesNotTriggerChase(t *testing.T) {
	p := newTestPet()
	step(p, 11)
	for x := 100; x < 400; x += 5 { // smooth one-way drift
		p.Update(x, 50)
	}
	if p.state == "chase" {
		t.Fatal("steady cursor movement should not be read as a jiggle")
	}
}

func TestMoodReweightNeverPanics(t *testing.T) {
	// decideNextAction indexes weights by action; a mismatched slice would panic.
	for _, m := range []Mood{MoodNeutral, MoodAlert, MoodTired} {
		p := newTestPet()
		p.mood = m
		for i := 0; i < 500; i++ {
			p.decideNextAction()
		}
	}
}
