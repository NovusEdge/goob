import os
import tempfile
import unittest

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


if __name__ == "__main__":
    unittest.main()
