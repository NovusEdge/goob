"""goob LLM daemon: one localhost route, POST /tick -> {say?, state?}.

Opt-in (`just daemon`). Provider/model/key come from the environment (litellm's
native vars, e.g. OPENAI_API_KEY; GOOB_MODEL selects the model). Localhost-only.
"""
import json
import os
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

from daemon import agent

HOST, PORT = "127.0.0.1", 8787
MODEL = os.environ.get("GOOB_MODEL", "gpt-4o-mini")

_PERSONALITY = ""


def load_personality(path="PERSONALITY.md"):
    try:
        with open(path) as f:
            return f.read().strip()
    except OSError:
        print("goob: no PERSONALITY.md, using built-in prompt")
        return "You are goob, a terse, affectionate desktop cat. Comment briefly."


def llm_completion(messages, tools, tool_choice):
    """Adapt litellm's response object to the normalized dict run_agent wants."""
    import litellm
    resp = litellm.completion(model=MODEL, messages=messages, tools=tools,
                              tool_choice=tool_choice)
    m = resp.choices[0].message
    calls = [{"id": tc.id,
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
            n = int(self.headers.get("Content-Length", 0))
            facts = json.loads(self.rfile.read(n) or b"{}")
            out = agent.run_agent(facts, _PERSONALITY, llm_completion)
        except Exception as e:            # never 500 the pet; degrade to {}
            print("goob: /tick error:", e)
            out = {}
        body = json.dumps(out).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(body)))
        self.end_headers()
        self.wfile.write(body)

    def log_message(self, *args):
        pass                               # quiet


def main():
    global _PERSONALITY
    _PERSONALITY = load_personality()
    print(f"goob daemon on http://{HOST}:{PORT}  model={MODEL}")
    ThreadingHTTPServer((HOST, PORT), Handler).serve_forever()


if __name__ == "__main__":
    main()
