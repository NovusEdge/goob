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


if __name__ == "__main__":
    unittest.main()
