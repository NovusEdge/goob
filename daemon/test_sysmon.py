import os
import tempfile
import unittest

from daemon import sysmon


class SysmonTest(unittest.TestCase):
    def test_watched_processes_matches_allowlist(self):
        with tempfile.TemporaryDirectory() as root:
            _proc(root, "100", "cargo")
            _proc(root, "101", "firefox")   # not watched
            _proc(root, "102", "gcc")
            got = sorted(sysmon.watched_processes(root))
            self.assertEqual(got, ["cargo", "gcc"])

    def test_system_state_reads_battery_and_thermal(self):
        with tempfile.TemporaryDirectory() as power, tempfile.TemporaryDirectory() as thermal:
            _write(os.path.join(power, "BAT0", "capacity"), "42\n")
            _write(os.path.join(power, "BAT0", "status"), "Discharging\n")
            _write(os.path.join(thermal, "thermal_zone0", "temp"), "71000\n")
            st = sysmon.system_state(power, thermal)
            self.assertEqual(st["battery_pct"], 42)
            self.assertFalse(st["charging"])
            self.assertEqual(st["hottest_c"], 71)

    def test_dispatch_routes_and_unknown_is_empty(self):
        self.assertIn("part_of_day", sysmon.dispatch("get_time_context", {}))
        self.assertEqual(sysmon.dispatch("no_such_tool", {}), {})


def _proc(root, pid, comm):
    _write(os.path.join(root, pid, "comm"), comm + "\n")


def _write(path, text):
    os.makedirs(os.path.dirname(path), exist_ok=True)
    with open(path, "w") as f:
        f.write(text)


if __name__ == "__main__":
    unittest.main()
