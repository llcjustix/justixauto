import importlib.util
import signal
import sys
import unittest
from pathlib import Path
from unittest.mock import patch


SPEC = importlib.util.spec_from_file_location("stop_apps", Path(__file__).with_name("stop-apps.py"))
stop = importlib.util.module_from_spec(SPEC)
assert SPEC.loader
sys.modules[SPEC.name] = stop
SPEC.loader.exec_module(stop)


ROOT = Path("/repo")
UID = 501


def proc(pid, command, cwd="/repo", ppid=1, uid=UID):
    return stop.Process(pid, ppid, uid, cwd, command, "Tue Sep 30 10:00:00 2026")


class FakeSystem:
    def __init__(self, processes=(), ports=None, remove_on_term=True):
        self.processes = list(processes)
        self.ports = ports or {}
        self.remove_on_term = remove_on_term
        self.sent = []
        self.now = 0.0

    def discover(self):
        return list(self.processes)

    def listeners(self, port):
        return set(self.ports.get(port, set()))

    def ports_for(self, pid):
        return {port for port, pids in self.ports.items() if pid in pids}

    def send(self, pid, sig):
        self.sent.append((pid, sig))
        if sig == signal.SIGKILL or self.remove_on_term:
            self.processes = [item for item in self.processes if item.pid != pid]
            for pids in self.ports.values():
                pids.discard(pid)

    def sleep(self, seconds):
        self.now += seconds

    def clock(self):
        return self.now

    def run(self, **kwargs):
        options = dict(
            environ={"HTTP_ADDR": "127.0.0.1:8080"},
            discover=self.discover,
            ports_for=self.ports_for,
            listeners=self.listeners,
            send=self.send,
            sleep=self.sleep,
            clock=self.clock,
            uid=UID,
        )
        options.update(kwargs)
        return stop.stop_apps(
            ROOT,
            **options,
        )


class StopAppsTest(unittest.TestCase):
    def test_stops_legacy_supervisors_before_api_and_vite(self):
        air = proc(10, "/tools/air", ppid=2)
        concurrently = proc(11, "node /repo/web/node_modules/.bin/concurrently --names api,realization,financing,insurance,admin", ppid=2)
        api = proc(20, "/repo/var/bin/dev-api", ppid=10)
        vite = proc(21, "node /repo/web/node_modules/vite/bin/vite --host 127.0.0.1", "/repo/web/apps/realization", 11)
        fake = FakeSystem([air, concurrently, api, vite], {8080: {20}, 5191: {21}})

        result = fake.run()

        self.assertTrue(result.ok)
        self.assertEqual([pid for pid, _ in fake.sent], [10, 11, 20, 21])
        self.assertTrue(all(sig == signal.SIGTERM for _, sig in fake.sent))

    def test_stops_standalone_go_and_vite(self):
        api = proc(20, "/pinned/go run ./cmd/api")
        vite = proc(21, "node /repo/web/node_modules/vite/bin/vite --host 127.0.0.1", "/repo/web/apps/admin")
        fake = FakeSystem([api, vite], {8080: {20}, 5194: {21}})

        self.assertTrue(fake.run().ok)
        self.assertEqual({20, 21}, {pid for pid, _ in fake.sent})

    def test_stops_reparented_compiled_go_api_child_that_owns_port(self):
        parent = proc(20, "/pinned/go run ./cmd/api")
        child = proc(21, "/private/tmp/go-build123/exe/api", ppid=20)
        fake = FakeSystem([parent, child], {8080: {21}})

        original_send = fake.send

        def send(pid, sig):
            original_send(pid, sig)
            if pid == 20:
                fake.processes = [
                    stop.Process(item.pid, 1, item.uid, item.cwd, item.command, item.started)
                    if item.pid == 21 else item
                    for item in fake.processes
                ]

        self.assertTrue(fake.run(send=send).ok)
        self.assertEqual([(20, signal.SIGTERM), (21, signal.SIGTERM)], fake.sent)

    def test_foreign_and_shell_like_processes_are_never_targets(self):
        candidates = [
            proc(1, "/repo/var/bin/dev-api", uid=999),
            proc(2, "/repo/var/bin/dev-api", "/other/repo"),
            proc(3, "zsh", "/repo"),
            proc(4, "postgres", "/repo"),
            proc(5, "node /repo/web/node_modules/vite/bin/vite", "/repo/web/apps/realization-copy"),
        ]
        self.assertEqual([], stop.targets(candidates, ROOT, UID))

    def test_revalidates_pid_identity_before_signal(self):
        original = proc(20, "/repo/var/bin/dev-api")
        reused = proc(20, "postgres")
        calls = 0

        def discover():
            nonlocal calls
            calls += 1
            return [original] if calls == 1 else [reused]

        result = stop.stop_apps(ROOT, environ={"HTTP_ADDR": "127.0.0.1:8080"}, discover=discover,
                                ports_for=lambda _: set(), listeners=lambda _: set(), send=lambda *_: self.fail("signaled reused PID"),
                                sleep=lambda _: None, clock=lambda: 20, uid=UID)
        self.assertTrue(result.ok)

    def test_escalates_only_surviving_owned_process(self):
        api = proc(20, "/repo/var/bin/dev-api")
        fake = FakeSystem([api], {8080: {20}}, remove_on_term=False)

        self.assertTrue(fake.run().ok)
        self.assertEqual([(20, signal.SIGTERM), (20, signal.SIGKILL)], fake.sent)

    def test_http_addr_precedence_and_safe_dotenv_parsing(self):
        self.assertEqual(8090, stop.parse_http_addr("[::1]:8090"))
        self.assertEqual(8080, stop.api_port(ROOT, {"HTTP_ADDR": "127.0.0.1:8080"}))
        with self.assertRaises(ValueError):
            stop.parse_http_addr("8080")
        with self.assertRaises(ValueError):
            stop.parse_http_addr("host:70000")

    def test_dotenv_supports_plain_and_quoted_values_only(self):
        path = Path("/mock/.env")
        with patch.object(Path, "read_text", return_value="HTTP_ADDR='127.0.0.1:8090'\n"):
            self.assertEqual("127.0.0.1:8090", stop.dotenv_http_addr(path))
        with patch.object(Path, "read_text", return_value="HTTP_ADDR=$(bad)\n"):
            with self.assertRaises(ValueError):
                stop.dotenv_http_addr(path)

    def test_empty_is_idempotent_and_unrelated_listener_fails_without_signal(self):
        empty = FakeSystem()
        self.assertTrue(empty.run().ok)
        foreign = FakeSystem(ports={8080: {99}})
        result = foreign.run()
        self.assertFalse(result.ok)
        self.assertEqual([], foreign.sent)
        self.assertIn("unrelated listener", " ".join(result.messages))

    def test_permission_denied_and_discovery_failure_fail_closed(self):
        api = proc(20, "/repo/var/bin/dev-api")
        fake = FakeSystem([api])

        def denied(*_):
            raise PermissionError

        result = fake.run(send=denied)
        self.assertFalse(result.ok)
        self.assertIn("permission denied", result.messages[0])
        result = stop.stop_apps(ROOT, environ={"HTTP_ADDR": "127.0.0.1:8080"},
                                discover=lambda: (_ for _ in ()).throw(RuntimeError("ps unavailable")), uid=UID)
        self.assertFalse(result.ok)


if __name__ == "__main__":
    unittest.main()
