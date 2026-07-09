import http.client
import json
import os
import tempfile
import threading
import unittest
from http.server import ThreadingHTTPServer

from daemon import server


class LoadPersonalityTest(unittest.TestCase):
    def test_missing_path_returns_fallback(self):
        result = server.load_personality(path="/nonexistent/PERSONALITY.md")
        self.assertIsInstance(result, str)
        self.assertNotEqual(result, "")

    def test_invalid_utf8_returns_fallback(self):
        fd, path = tempfile.mkstemp()
        try:
            with os.fdopen(fd, "wb") as f:
                f.write(b"\xff\xfe bad")
            result = server.load_personality(path=path)
            self.assertIsInstance(result, str)
            self.assertNotEqual(result, "")
        finally:
            os.remove(path)


class LoadDotenvTest(unittest.TestCase):
    def test_parses_and_respects_shell_precedence(self):
        fd, path = tempfile.mkstemp()
        try:
            with os.fdopen(fd, "w") as f:
                f.write(
                    "# a comment\n"
                    "\n"
                    "GOOB_TEST_PLAIN=hello\n"
                    "export GOOB_TEST_EXPORT=world\n"
                    'GOOB_TEST_QUOTED="spaced value"\n'
                    "GOOB_TEST_SHELL=fromfile\n"
                )
            os.environ.pop("GOOB_TEST_PLAIN", None)
            os.environ.pop("GOOB_TEST_EXPORT", None)
            os.environ.pop("GOOB_TEST_QUOTED", None)
            os.environ["GOOB_TEST_SHELL"] = "fromshell"  # already set → wins
            server.load_dotenv(path=path)
            self.assertEqual(os.environ["GOOB_TEST_PLAIN"], "hello")
            self.assertEqual(os.environ["GOOB_TEST_EXPORT"], "world")   # export stripped
            self.assertEqual(os.environ["GOOB_TEST_QUOTED"], "spaced value")  # quotes stripped
            self.assertEqual(os.environ["GOOB_TEST_SHELL"], "fromshell")  # shell env not overridden
        finally:
            os.remove(path)
            for k in ("GOOB_TEST_PLAIN", "GOOB_TEST_EXPORT", "GOOB_TEST_QUOTED", "GOOB_TEST_SHELL"):
                os.environ.pop(k, None)

    def test_missing_file_is_noop(self):
        server.load_dotenv(path="/nonexistent/.env")  # must not raise


class StatsEndpointTest(unittest.TestCase):
    def setUp(self):
        server._PERSONALITY = "test personality"
        self._orig = server.llm_completion
        # Stub: return an immediate `emit` tool call so run_agent finishes in one
        # step without hitting the network. Signature matches llm_completion.
        server.llm_completion = lambda messages, tools, tool_choice: {
            "content": None,
            "tool_calls": [{"id": "1", "function": {
                "name": "emit", "arguments": '{"say": "hi"}'}}],
        }
        self.srv = ThreadingHTTPServer(("127.0.0.1", 0), server.Handler)
        self.port = self.srv.server_address[1]
        self.thread = threading.Thread(target=self.srv.serve_forever, daemon=True)
        self.thread.start()

    def tearDown(self):
        self.srv.shutdown()
        server.llm_completion = self._orig

    def _get(self, path):
        c = http.client.HTTPConnection("127.0.0.1", self.port)
        c.request("GET", path)
        r = c.getresponse()
        return r.status, r.read()

    def _post(self, path, body):
        c = http.client.HTTPConnection("127.0.0.1", self.port)
        c.request("POST", path, json.dumps(body))
        r = c.getresponse()
        return r.status, r.read()

    def test_stats_shape_and_tick_increment(self):
        status, body = self._get("/stats")
        self.assertEqual(status, 200)
        data = json.loads(body)
        self.assertEqual(set(data), {"model", "ticks", "spend_usd", "last_latency_ms"})
        before = data["ticks"]
        self._post("/tick", {"event": "test"})
        _, body2 = self._get("/stats")
        self.assertEqual(json.loads(body2)["ticks"], before + 1)

    def test_unknown_get_returns_404(self):
        status, _ = self._get("/nope")
        self.assertEqual(status, 404)


if __name__ == "__main__":
    unittest.main()
