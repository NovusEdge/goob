# goob × LLM — Voice → Pet Reply Design Brainstorm

Status: brainstorm to react to, not a spec. Bias: lazy-first, stdlib-first, YAGNI-flagged.

Feature: user talks to the cat (voice → text), an LLM decides a reply, the cat
(a) plays an animation state and (b) shows the text in a chat-bubble overlay.
Later: TTS speaks the reply.

---

## 0. The one hard constraint

The frame loop (`raylib.go` / `wayland.go`) runs at 60fps and **must never block**.
An LLM round-trip is 0.5–5s. So the LLM call lives off the loop, and its result
is handed back to `Pet` at a safe point. Everything below follows from that.

The other constraint: `Pet` (`internal/pet/behavior.go`) has zero synchronization
today — the loop is single-threaded and every field is unexported. We keep it that
way. The LLM goroutine does **not** touch `Pet` directly.

---

## 1. Architecture — where the LLM lives and how state gets back

```
 mic ──(push-to-talk)──▶ STT ──text──▶ LLM goroutine ──▶ result channel
                                                              │
 frame loop (60fps) ──── each tick: non-blocking drain ◀──────┘
        │
        ├── p.Say(state, text)      → sets pet state + stashes bubble text
        └── bubble overlay reads current text, resizes, redraws
```

- **Goroutine, not separate process.** One `internal/chat` package with a
  `Session` that owns the `anthropic.Client`. Hotkey handler fires
  `go session.Ask(transcript)`. Simplest thing that works; a subprocess buys us
  nothing here (no sandboxing need, no crash isolation worth the IPC).
- **Result handoff = a buffered channel**, drained non-blockingly in the frame
  loop right where mood is sampled today:
  ```go
  select {
  case r := <-chat.Results:
      p.Say(r.State, r.Text)   // new method, mirrors Scare()/Jump()
  default:
  }
  ```
  This is the whole thread-safety story: the channel is the boundary, `Pet`
  stays single-threaded, no mutex. `r.State` is validated against the canonical
  vocabulary *before* it reaches the channel (see §2).
- **`p.Say(state, text)`** is a new `Pet` method alongside `Scare`/`Jump`. It
  guards like the others (`if p.interruptible()`), sets `p.state`, and records
  the bubble text + a TTL. It should respect `holdStates` so an LLM-chosen
  `meow`/`sit`/`scared` plays naturally and then falls back to idle.
- **Bubble text lifetime** lives on `Pet` (e.g. `bubbleText string`, `bubbleUntil int`
  in frame ticks) so both backends read it the same way — same pattern as
  `Anim()`. Bubble clears itself after N seconds.

YAGNI: no queue of pending replies, no conversation-state machine in the loop,
no streaming tokens into the bubble (nice, but v2). One in-flight request at a
time; ignore/spinner if the user talks again while thinking.

---

## 2. How the LLM picks states

Canonical vocabulary = `holdStates` keys + `idle walk chase run pounce jump sit`.
That's ~20 tokens. The `sprite.Resolve` fallback chain already means an
unknown-to-this-sheet state degrades gracefully, so the cost of a bad pick is low.

**Recommendation: tool use (function calling), not free-text parsing.**
Define one tool the model must call:

```jsonc
// respond(state: enum[...canonical...], text: string)
{ "name": "respond",
  "input_schema": { "type": "object", "additionalProperties": false,
    "required": ["state", "text"],
    "properties": {
      "state": { "type": "string", "enum": ["idle","sit","meow","scared", ...] },
      "text":  { "type": "string", "description": "≤120 chars, spoken as the cat" }
  }}}
```

Why tool use over "parse `STATE: meow\nTEXT: ...`":
- `enum` + `strict: true` (structured tool use, no beta header) makes the state
  field *guaranteed valid* — no parser, no validation branch, no "model wrote
  `meowing`" bugs. The enum **is** the whitelist.
- The enum list is generated in Go from the same map that drives the state
  machine — single source of truth, can't drift.
- `tool_choice: {"type":"tool","name":"respond"}` forces the call every turn.

Keep it to the **manual loop or a single `messages.create`** — there's no
multi-turn tool cycle here (the tool doesn't return data to the model; we just
harvest its input). So: one non-streaming `messages.create`, read
`response.content` for the `tool_use` block, push `{state, text}` to the channel.
No tool-runner needed.

**Model: `claude-haiku-4-5`.** This is the one place to deviate from the
Opus-4.8 default — the task is trivial (pick a label + one sentence, low latency
matters, it's a desktop toy fired repeatedly). Haiku is fastest/cheapest and
easily handles constrained tool use. Make it a config knob defaulting to Haiku;
someone who wants wittier replies can point it at `claude-opus-4-8`. Use
`thinking` disabled / omit it (latency), small `max_tokens` (~256). Auth via the
standard `ANTHROPIC_API_KEY` env / `ant auth` resolution — no key handling code.

Persona lives in a **frozen system prompt** (cat personality + "you get the
user's speech, reply in ≤120 chars, always call respond") so it prompt-caches;
per-turn transcript goes in the user message. Optionally feed current mood
(`MoodTired`/`MoodAlert`) as one line of context so the cat acts sleepy when the
CPU is hot.

---

## 3. Chat-bubble overlay — backend-agnostic

A **second transparent, click-through, always-on-top window** that renders the
bubble sprite art + text, positioned above the cat, sized to fit the text.

The trap: raylib (X11) and GTK4 layer-shell (Wayland) are *completely different*
windowing stacks. Don't design one bubble window class — design a tiny interface
and implement it twice, exactly like the pet window already is.

```go
// internal/bubble (or just an interface in cmd/goob)
type Overlay interface {
    Show(text string, x, y int)  // create/move/resize to fit text
    Hide()
}
```

- **raylib backend:** a second `rl` window is awkward (raylib is happiest with
  one window). Two lazy options, in order of preference:
  1. **Draw the bubble in the *same* pet window** — just enlarge the pet
     overlay window when a bubble is active and blit bubble art + text above the
     sprite. No second window, no second event loop. This is by far the least
     code and sidesteps all multi-window pain. The cat window is already
     click-through and topmost.
  2. If the bubble must extend beyond the pet window bounds, a second
     `InitWindow` on another goroutine — but raylib/GLFW dislike multi-window
     multi-goroutine; treat as YAGNI until option 1 proves too cramped.
- **Wayland backend:** a second `gtk_window` + `gtk_layer_init_for_window` is
  natural here (layer-shell is built for multiple overlay surfaces). Reuse the
  `wayland_create_window` / `set_position` / `set_size` C shims almost verbatim
  for a `bubble_*` variant.

**Dynamic resize = measure then size.** Both backends can measure text:
raylib `MeasureTextEx`, GTK/Pango via cairo `text_extents`. Compute
`w,h = textSize + padding + bubbleChrome`, set the window/child size, redraw.
The user-supplied bubble art should be **9-slice** (corners fixed, edges/center
stretched) so one PNG fits any text box — worth asking the artist for a sliceable
bubble rather than fixed-size art.

Recommendation for MVP: **start with raylib option 1 (draw in the pet window)**
and the Wayland second-surface, both behind the `Overlay` interface. Text
rendering is the only genuinely new code.

---

## 4. Voice input — laziest viable STT

**Recommendation: push-to-talk hotkey + local `whisper.cpp` via `exec`.**

- **Push-to-talk, not always-listening.** Always-listening means: a hotword
  engine running 24/7, a ring buffer of your mic, endpointing logic, and a
  standing question of "is this thing sending my kitchen conversations
  somewhere". That's a privacy liability and a pile of complexity for a desktop
  toy. Push-to-talk is one global hotkey: press → record → release → transcribe.
  Clear consent, trivial to reason about, no VAD. Make always-listening a
  someday-maybe, explicitly YAGNI.
- **STT path:** shell out to a local `whisper.cpp` binary (`whisper-cli`/
  `whisper.cpp`) on a temp WAV. Recording the WAV is the laziest via `exec` too
  — `arecord` (ALSA) or `parecord` (PulseAudio/Pipewire) while the key is held,
  SIGINT on release. No Go audio bindings, no cgo audio. Pure `os/exec` +
  stdlib.
  ```
  key down → parecord --format=s16le ... /tmp/goob.wav
  key up   → kill; whisper-cli -f /tmp/goob.wav -otxt → read text → LLM
  ```
- **Local vs cloud:** local whisper.cpp keeps voice off the network (only the
  transcript text goes to Anthropic), no second API key, works offline for the
  STT half. A cloud STT (e.g. an Anthropic-less STT service) is *less* lazy here
  (another key, another SDK, audio upload) — skip it. Small whisper model
  (`base`/`small`) is plenty for short commands.
- **Global hotkey** is the fiddly cross-backend bit (X11 grab vs Wayland
  compositor keybind). Laziest cross-desktop answer: don't grab a global key at
  all — bind the compositor/DE hotkey to send the running goob a signal
  (SIGUSR1) or write to a FIFO, and goob starts recording on that. Punts the
  platform keygrab problem entirely onto the user's existing keybind config.
  (If that's too much to ask users, X11 `XGrabKey` / a Wayland
  `wlr-foreign-toplevel`-adjacent approach later — YAGNI for MVP.)

Dependencies are `exec`-only and optional: if `whisper`/`parecord` aren't
present, the voice path just disables and text-input still works.

---

## 5. TTS (future)

Mark future. Same shape as STT in reverse: after the LLM returns `text`, shell
out to a local TTS (`piper` is the obvious lazy pick — single binary, decent
voices, offline) and pipe its WAV to `aplay`/`paplay`. One goroutine, fire-and-
forget, gated behind a config flag. Optionally animate `meow`/mouth frames while
audio plays by holding the state for the clip duration. No new architecture —
it's another `exec` at the tail of the reply handler. Cloud TTS only if local
quality disappoints. YAGNI until voice-in + bubble feel good.

---

## 6. Minimal MVP slice (build this first)

Cut voice **and** TTS. Prove the loop end-to-end with text:

1. **Hotkey → text prompt.** A key (or SIGUSR1/FIFO per §4) pops a one-line
   text input for what you "say" to the cat. Zero audio deps.
2. **`internal/chat`**: one `messages.create` to `claude-haiku-4-5`, forced
   `respond(state, text)` tool with the enum whitelist, frozen persona system
   prompt. Push `{state, text}` to a buffered channel.
3. **Frame loop** drains the channel each tick → `p.Say(state, text)` (new
   guarded `Pet` method; sets state + `bubbleText` + TTL).
4. **Bubble render**: raylib option 1 (draw bubble art + text in the enlarged
   pet window); Wayland second layer-shell surface. Both behind `Overlay`.
   Placeholder art is fine until the real bubble PNG lands.

That's the entire value loop (input → LLM → animation + bubble) with only
`os/exec`-free additions plus the Anthropic Go SDK. Voice is then a drop-in
front-end (swap the text prompt for record→whisper), and TTS a drop-in back-end.

---

## 7. Open questions / human decisions

- **Bubble art contract:** 9-slice sliceable PNG, or fixed sizes? Tail/pointer
  direction (does it flip with `FacingLeft`)? Text color/font baked into art or
  drawn by us? — affects §3 measure-and-size code.
- **Hotkey mechanism:** in-process global grab vs. "bind it in your DE to signal
  goob". The latter is much lazier and cross-backend; OK to push onto users?
- **Model default:** Haiku-4.5 (fast/cheap toy) vs Opus-4.8 (wittier). Config
  knob resolves it — just confirm the default.
- **Persona source:** hardcoded system prompt, or a user-editable file so people
  can re-personality their cat? (A file is barely more work and very on-brand.)
- **Interrupt policy:** should a chat reply override `held`/`chase`/`scared`
  (currently non-`interruptible`)? Probably let those finish, queue nothing.
- **Conversation memory:** stateless per-utterance (lazy, recommended) vs. keep
  last N turns so the cat "remembers"? Memory = more tokens + a growing slice in
  `chat.Session`. Start stateless.
- **Concurrent speech:** user talks while the cat is thinking — drop, queue, or
  cancel-and-restart? MVP: drop (ignore) with a subtle "thinking" tell.
- **Where does the bubble sit** when the cat is at a screen edge / being
  dragged / airborne? Clamp like `clampPosition`, or hide during held/jump.

---

## Recommended first step

Build the **text-input MVP (§6)**: wire `internal/chat` with a forced
`respond(state, text)` Haiku tool call whose `state` enum is generated from the
existing state map, hand results to a new guarded `p.Say()` via a buffered
channel drained in the frame loop, and render the bubble by enlarging the raylib
pet window (option 1). That exercises the whole voice→LLM→animation+bubble
pipeline with no audio dependencies, so voice (whisper.cpp) and TTS (piper) each
become an isolated `exec` bolt-on afterward.
