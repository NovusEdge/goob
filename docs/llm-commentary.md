# LLM Commentary

goob comments on what your machine is doing. There are two modes, and it
degrades gracefully between them:

- **Canned (default):** no daemon, no key, no network. Godot picks a line
  from `config/comments.json` keyed by mood. Works out of the box.
- **LLM (optional):** run the Python daemon and comments are generated live —
  the pet can also nudge its own behaviour. If the daemon is down,
  unreachable, or errors, goob silently falls back to the canned line. It
  never goes silent.

## Running the daemon

You need [uv](https://docs.astral.sh/uv/). `just daemon` uses uv (reading
`pyproject.toml`) to fetch `litellm` on demand — no `pip`, no manual install.

1. `cp .env.example .env`, then edit `.env` — pick a provider and fill in your
   model + key. The daemon auto-loads `.env` (no need to `source` it); process
   environment variables override it.
2. `just daemon`.

`GOOB_MODEL` is the litellm model string. The daemon logs each request to its
terminal: `goob: tick mood=... -> {say, state}`. It binds `127.0.0.1:8787`,
localhost only.

## Providers & auth

| Provider | `GOOB_MODEL` | Auth |
|----------|-------------|------|
| Google AI Studio | `gemini/gemini-2.5-flash` | `GEMINI_API_KEY=...` |
| OpenAI | `gpt-4o-mini` | `OPENAI_API_KEY=...` |
| Google Vertex AI | `vertex_ai/...` | OAuth only — see below |
| Ollama (local) | `ollama/llama3` (or any pulled model) | none |

**Google AI Studio** is the simplest option — just an API key. Get one at
[aistudio.google.com/apikey](https://aistudio.google.com/apikey). The key must
be allowed to call the Generative Language API — a GCP key restricted away
from it fails with `API_KEY_SERVICE_BLOCKED`.

**OpenAI** just needs `OPENAI_API_KEY`.

**Google Vertex AI** does not accept a plain API key. Authenticate with a
service account (`GOOGLE_APPLICATION_CREDENTIALS=/path/to/sa.json`) or ADC
(`gcloud auth application-default login`), and set `VERTEXAI_PROJECT` +
`VERTEXAI_LOCATION`. Requires the `[google]` extra (`litellm[google]` in
`pyproject.toml`). Your account/service-account needs the
`roles/aiplatform.user` role on the project. Note: Vertex "Express" API keys
are **not** supported here — they'd require the google-genai SDK, which goob
doesn't use.

**Ollama** runs fully local — no key, nothing leaves your machine.

## When it triggers

goob samples "mood" from the system about every 2 seconds:

- `neutral` — default.
- `alert` — a watched build/dev process is running (`go`, `gcc`, `clang`,
  `rustc`, `cargo`, `node`, `npm`, `webpack`, `tsc`, `make`, `cmake`, `ninja`,
  `docker`, `gradle`, `mvn`, `python`, `ld`).
- `tired` — battery below 15% and discharging, or the hottest thermal zone
  ≥ 85°C.

goob only calls the daemon on a **mood change** (an edge — e.g.
neutral→alert when a build starts), behind a **~20s cooldown**. If the mood
never changes, the daemon gets no requests. To see it fire, start or stop a
build (`cargo build`, `make`, ...), or let the battery drop.

One caveat: because `python` is a watched process, the daemon's own Python
process can itself register as "alert" — so mood may sit at alert while the
daemon runs.

When it fires, Godot POSTs facts (`pet_state`, `mood`, `event`, `user_text`)
and the daemon replies with `say` and/or `state`. `say` shows in the speech
bubble; `state` (one of `idle`, `wander`, `follow`, `jump`, `zoomies`) nudges
the pet's behaviour. Invalid or omitted values are ignored.

## Privacy

The daemon only ever sends the model an allowlisted digest — which watched
processes are running, battery, thermal, time — never your full process list
or command lines. Use Ollama for a fully-local setup where nothing leaves
your machine. The daemon's tools are read-only: it can observe the machine
but never change it; the pet's only outputs are a line to say and a
behaviour to switch to.

## Customising

The pet's voice and canned lines are configured under `config/` — see
[Configuration](configuration.md) for details.
