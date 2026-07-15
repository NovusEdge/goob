"""goob LLM daemon: one localhost route, POST /tick -> {say?, state?}.

Opt-in (`just daemon`). Config is read from `.env` (auto-loaded) or the process
environment: GOOB_MODEL picks the litellm model, and the provider key from
GOOB_API_KEY / VERTEXAI_API_KEY / GEMINI_API_KEY / etc. Localhost-only.
"""
import json
import os
import threading
import time
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

from daemon import agent

HOST, PORT = "127.0.0.1", 8787
DEFAULT_MODEL = "gpt-4o-mini"

# The system prompt is composed of two files: PROMPT.md is the fixed engine
# contract (tools, the emit format, constraints); PERSONALITY.md is the user's
# editable character/voice. Each falls back to a built-in if absent.
BUILTIN_PROMPT = (
    "You are goob, a small pet living on the user's desktop. Use your read-only "
    "tools to observe the machine, then deliver your reaction by calling the "
    "`emit` tool with an optional short `say` and an optional `state`. Emitting "
    "neither (staying silent) is fine."
)
BUILTIN_PERSONALITY = "You are a dry, affectionate, slightly chaotic cat."

_SYSTEM = ""

_STATS_LOCK = threading.Lock()
_ticks = 0
_spend_usd = 0.0
_last_latency_ms = 0.0


def load_dotenv(path=".env"):
    """Load KEY=VALUE lines from .env into the environment (shell env wins).

    Tiny by design — no python-dotenv dependency. Handles `export KEY=val`,
    `#` comments, blank lines, and surrounding single/double quotes.
    """
    try:
        with open(path) as f:
            lines = f.readlines()
    except OSError:
        return
    for raw in lines:
        line = raw.strip()
        if line.startswith("export "):
            line = line[len("export "):]
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, _, val = line.partition("=")
        key = key.strip()
        val = val.strip().strip('"').strip("'")
        if key and key not in os.environ:   # real shell env takes precedence
            os.environ[key] = val


def _read_md(path):
    try:
        with open(path) as f:
            return f.read().strip()
    except (OSError, UnicodeDecodeError):
        return ""


def load_prompt(path="config/PROMPT.md"):
    """The engine contract. Built-in fallback if the file is missing/unreadable."""
    return _read_md(path) or BUILTIN_PROMPT


def load_personality(path="config/PERSONALITY.md"):
    """The pet's editable voice. Built-in fallback if missing/unreadable."""
    return _read_md(path) or BUILTIN_PERSONALITY


def compose_system(prompt_path="config/PROMPT.md",
                   personality_path="config/PERSONALITY.md"):
    """System prompt = engine contract (PROMPT.md) + character (PERSONALITY.md)."""
    prompt = _read_md(prompt_path)
    personality = _read_md(personality_path)
    if not prompt:
        print("goob: no PROMPT.md, using built-in engine prompt")
        prompt = BUILTIN_PROMPT
    if not personality:
        print("goob: no PERSONALITY.md, using built-in personality")
        personality = BUILTIN_PERSONALITY
    return prompt + "\n\n# Your personality\n\n" + personality


def llm_completion(messages, tools, tool_choice):
    """Adapt litellm's response object to the normalized dict run_agent wants."""
    global _spend_usd
    import litellm
    model = os.environ.get("GOOB_MODEL", DEFAULT_MODEL)
    kwargs = {}
    if model.startswith("vertex_ai"):
        # Vertex authenticates via Google OAuth — ADC (`gcloud auth application-
        # default login`) or a service account in GOOGLE_APPLICATION_CREDENTIALS,
        # which litellm reads directly. A plain API key does NOT work with Vertex.
        for env, kw in (("VERTEXAI_PROJECT", "vertex_project"),
                        ("VERTEXAI_LOCATION", "vertex_location")):
            if os.environ.get(env):
                kwargs[kw] = os.environ[env]
    else:
        # gemini/, openai/, etc. take an explicit API key.
        api_key = os.environ.get("GOOB_API_KEY") or os.environ.get("GEMINI_API_KEY")
        if api_key:
            kwargs["api_key"] = api_key
    resp = litellm.completion(model=model, messages=messages, tools=tools,
                              tool_choice=tool_choice, timeout=5, **kwargs)
    # litellm prices the call; None/AttributeError for unpriceable models (Ollama).
    cost = None
    try:
        cost = resp._hidden_params.get("response_cost")
    except AttributeError:
        cost = None
    if cost is None:
        try:
            cost = litellm.completion_cost(resp)
        except Exception:
            cost = 0.0
    with _STATS_LOCK:
        _spend_usd += cost or 0.0
    m = resp.choices[0].message
    calls = [{"id": tc.id, "type": "function",
              "function": {"name": tc.function.name,
                           "arguments": tc.function.arguments}}
             for tc in (m.tool_calls or [])]
    return {"content": m.content, "tool_calls": calls}


class Handler(BaseHTTPRequestHandler):
    def do_POST(self):
        if self.path != "/tick":
            self.send_error(404)
            return
        try:
            n = min(max(0, int(self.headers.get("Content-Length", 0))), 65536)
            facts = json.loads(self.rfile.read(n) or b"{}")
            t0 = time.perf_counter()
            out = agent.run_agent(facts, _SYSTEM, llm_completion)
            dt_ms = (time.perf_counter() - t0) * 1000.0
            global _ticks, _last_latency_ms
            with _STATS_LOCK:
                _ticks += 1
                _last_latency_ms = dt_ms
            print("goob: tick mood=%r pet_state=%r -> %s"
                  % (facts.get("mood"), facts.get("pet_state"), out))
        except Exception as e:            # never 500 the pet; degrade to {}
            print("goob: /tick error:", e)
            out = {}
        self._send_json(out)

    def do_GET(self):
        if self.path != "/stats":
            self.send_error(404)
            return
        with _STATS_LOCK:
            payload = {
                "model": os.environ.get("GOOB_MODEL", DEFAULT_MODEL),
                "ticks": _ticks,
                "spend_usd": round(_spend_usd, 6),
                "last_latency_ms": round(_last_latency_ms, 1),
            }
        self._send_json(payload)

    def _send_json(self, obj):
        body = json.dumps(obj).encode()
        try:
            self.send_response(200)
            self.send_header("Content-Type", "application/json")
            self.send_header("Content-Length", str(len(body)))
            self.end_headers()
            self.wfile.write(body)
        except (BrokenPipeError, ConnectionError):
            # The pet's HTTP timeout is shorter than a slow LLM call, so it can
            # hang up before we reply. The tick already ran; drop the response
            # quietly instead of letting socketserver dump a traceback.
            print("goob: client gone before reply (tick still counted)")

    def log_message(self, fmt, *args):
        # Log requests to the TUI's log pane, but skip the 1s /stats healthcheck
        # polls — they'd drown out the interesting /tick lines.
        if self.path == "/stats":
            return
        print("goob:", fmt % args)


def main():
    global _SYSTEM
    load_dotenv()
    _SYSTEM = compose_system()
    model = os.environ.get("GOOB_MODEL", DEFAULT_MODEL)
    print(f"goob daemon on http://{HOST}:{PORT}  model={model}")
    ThreadingHTTPServer((HOST, PORT), Handler).serve_forever()


if __name__ == "__main__":
    main()
