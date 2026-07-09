import json
import unittest

from daemon import agent


def _emit_msg(args):
    return {"content": None, "tool_calls": [
        {"id": "1", "function": {"name": "emit", "arguments": json.dumps(args)}}]}


def _tool_then_emit(tool_name, emit_args):
    # First call asks for a tool, second call emits.
    calls = [
        {"content": None, "tool_calls": [
            {"id": "t", "function": {"name": tool_name, "arguments": "{}"}}]},
        _emit_msg(emit_args),
    ]

    def completion(messages, tools, tool_choice):
        return calls.pop(0)
    return completion


class AgentTest(unittest.TestCase):
    def test_emit_is_validated(self):
        comp = lambda messages, tools, tool_choice: _emit_msg(
            {"say": "  hi  ", "state": "wander"})
        out = agent.run_agent({}, "p", comp)
        self.assertEqual(out, {"say": "hi", "state": "wander"})

    def test_bad_state_is_dropped(self):
        comp = lambda m, t, tc: _emit_msg({"say": "x", "state": "explode"})
        self.assertEqual(agent.run_agent({}, "p", comp), {"say": "x"})

    def test_tool_call_then_emit(self):
        out = agent.run_agent({}, "p",
                              _tool_then_emit("get_time_context", {"say": "morning!"}))
        self.assertEqual(out, {"say": "morning!"})

    def test_prose_instead_of_emit_returns_empty(self):
        comp = lambda m, t, tc: {"content": "I think the cat is happy.", "tool_calls": []}
        self.assertEqual(agent.run_agent({}, "p", comp), {})

    def test_never_emits_within_cap_returns_empty(self):
        # Always asks for a tool, never emits -> hits the cap -> {}.
        comp = lambda m, t, tc: {"content": None, "tool_calls": [
            {"id": "t", "function": {"name": "get_system_state", "arguments": "{}"}}]}
        self.assertEqual(agent.run_agent({}, "p", comp, max_steps=3), {})


if __name__ == "__main__":
    unittest.main()
