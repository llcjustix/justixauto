#!/usr/bin/env python3
"""Stop local JustixAuto app processes owned by this checkout.

The helper deliberately recognizes a short list of development entrypoints.
Ports help report listeners, but never authorize a signal.
"""

from __future__ import annotations

import os
import re
import signal
import shlex
import subprocess
import sys
import time
from dataclasses import dataclass
from pathlib import Path
from typing import Callable, Iterable


VITE_PORTS = {5191, 5192, 5193, 5194}
GRACE_SECONDS = 20.0


@dataclass(frozen=True)
class Process:
    pid: int
    ppid: int
    uid: int
    cwd: str | None
    command: str
    started: str = ""


@dataclass
class Result:
    ok: bool
    messages: list[str]


def command_output(argv: list[str], allow_empty: bool = False) -> str:
    completed = subprocess.run(argv, text=True, stdout=subprocess.PIPE, stderr=subprocess.DEVNULL, check=False)
    if completed.returncode and not (allow_empty and completed.returncode == 1):
        raise RuntimeError(f"process discovery failed: {argv[0]}")
    return completed.stdout


def process_cwd(pid: int) -> str | None:
    output = command_output(["lsof", "-a", "-p", str(pid), "-d", "cwd", "-Fn"], allow_empty=True)
    for line in output.splitlines():
        if line.startswith("n"):
            return os.path.realpath(line[1:])
    return None


def argv(process: Process) -> list[str]:
    try:
        return shlex.split(process.command)
    except ValueError:
        return []


def basename(value: str) -> str:
    return os.path.basename(value)


def visible_candidate(process: Process, all_processes: dict[int, Process]) -> bool:
    args = argv(process)
    if not args:
        return False
    first = basename(args[0])
    if first in {"zsh", "sh", "fish"}:
        return False
    if first == "bash":
        return args[1:4] == ["tools/go.sh", "run", "./cmd/api"]
    if first in {"air", "concurrently", "dev-api", "go", "npm", "vite", "api"}:
        return True
    if first == "node":
        return any(
            ("node_modules" in value and basename(value) == "vite")
            or value.endswith("node_modules/.bin/concurrently")
            or ("node_modules/concurrently/" in value and basename(value) == "concurrently.js")
            for value in args[1:]
        )
    return False


def discover_processes(pid: int | None = None) -> list[Process]:
    ps_argv = ["ps", "-axo", "pid=,ppid=,uid=,lstart=,command="]
    if pid is not None:
        ps_argv = ["ps", "-p", str(pid), "-o", "pid=,ppid=,uid=,lstart=,command="]
    output = command_output(ps_argv, allow_empty=pid is not None)
    raw: list[Process] = []
    for line in output.splitlines():
        fields = line.split(None, 8)
        if len(fields) != 9:
            continue
        try:
            pid, ppid, uid = (int(value) for value in fields[:3])
        except ValueError:
            continue
        raw.append(Process(pid, ppid, uid, None, fields[8], " ".join(fields[3:8])))
    by_pid = {process.pid: process for process in raw}
    return [
        Process(process.pid, process.ppid, process.uid, process_cwd(process.pid), process.command, process.started)
        for process in raw
        if process.uid == os.getuid() and visible_candidate(process, by_pid)
    ]


def listener_pids(port: int) -> set[int]:
    output = command_output(["lsof", "-nP", f"-iTCP:{port}", "-sTCP:LISTEN", "-t"], allow_empty=True)
    return {int(line) for line in output.splitlines() if line.isdigit()}


def listening_ports(pid: int) -> set[int]:
    output = command_output(["lsof", "-nP", "-a", "-p", str(pid), "-iTCP", "-sTCP:LISTEN", "-Fn"], allow_empty=True)
    ports: set[int] = set()
    for line in output.splitlines():
        if not line.startswith("n"):
            continue
        match = re.search(r":(\d+)(?: \(LISTEN\))?$", line[1:])
        if match:
            ports.add(int(match.group(1)))
    return ports


def parse_http_addr(value: str) -> int:
    value = value.strip()
    match = re.fullmatch(r"\[[0-9A-Fa-f:.]+\]:(\d{1,5})", value) or re.fullmatch(
        r"[A-Za-z0-9.-]*:(\d{1,5})", value
    )
    if not match or not 0 < int(match.group(1)) < 65536:
        raise ValueError("HTTP_ADDR must be a host:port or [IPv6]:port address")
    return int(match.group(1))


def dotenv_http_addr(path: Path) -> str | None:
    try:
        lines = path.read_text(encoding="utf-8").splitlines()
    except FileNotFoundError:
        return None
    for line in lines:
        match = re.fullmatch(r"HTTP_ADDR\s*=\s*(.*?)\s*", line)
        if not match:
            continue
        value = match.group(1)
        if len(value) >= 2 and value[0] in "\"'" and value[-1] == value[0]:
            value = value[1:-1]
        elif value.startswith(("'", '\"')) or any(char in value for char in "$`;# "):
            raise ValueError(".env HTTP_ADDR uses unsupported syntax")
        if not value:
            raise ValueError(".env HTTP_ADDR is empty")
        return value
    return None


def api_port(root: Path, environ: dict[str, str]) -> int:
    value = environ.get("HTTP_ADDR") or dotenv_http_addr(root / ".env") or "127.0.0.1:8080"
    return parse_http_addr(value)


def same_process(left: Process, right: Process) -> bool:
    return bool(left.started) and (
        left.pid, left.uid, left.cwd, left.command, left.started
    ) == (
        right.pid, right.uid, right.cwd, right.command, right.started
    )


def is_air(process: Process, root: str) -> bool:
    args = argv(process)
    return process.cwd == root and bool(args) and basename(args[0]) == "air"


def is_concurrently(process: Process, root: str) -> bool:
    args = argv(process)
    if process.cwd != root or not args:
        return False
    entrypoint = basename(args[0]) == "concurrently" or (
        basename(args[0]) == "node" and any(
            value.endswith("node_modules/.bin/concurrently")
            or ("node_modules/concurrently/" in value and basename(value) == "concurrently.js")
            for value in args[1:]
        )
    )
    return entrypoint and "--names" in args and "api,realization,financing,insurance,admin" in args


def is_api(process: Process, root: str) -> bool:
    args = argv(process)
    if process.cwd != root or not args:
        return False
    return (basename(args[0]) == "go" and args[1:3] == ["run", "./cmd/api"]) or os.path.realpath(args[0]) == os.path.join(root, "var/bin/dev-api")


def is_vite(process: Process, app_dirs: set[str]) -> bool:
    args = argv(process)
    return process.cwd in app_dirs and bool(args) and (
        basename(args[0]) == "vite" or (basename(args[0]) == "node" and any("node_modules" in value and basename(value) == "vite" for value in args[1:]))
    )


def is_launch_wrapper(process: Process, root: str, processes: Iterable[Process]) -> bool:
    args = argv(process)
    known = (
        (process.cwd == root and basename(args[0]) == "bash" and args[1:4] == ["tools/go.sh", "run", "./cmd/api"])
        or (process.cwd == root and basename(args[0]) == "npm" and args[1:6] == ["--prefix", "web", "run", "dev", "--workspace"] and len(args) > 6 and args[6].startswith("apps/"))
    )
    return known and any(child.ppid == process.pid and (is_api(child, root) or is_vite(child, app_dirs(root))) for child in processes)


def app_dirs(root: str) -> set[str]:
    return {os.path.join(root, "web", "apps", name) for name in ("realization", "financing", "insurance", "admin")}


def targets(processes: Iterable[Process], root: Path, uid: int) -> list[tuple[str, Process]]:
    root_text = os.path.realpath(root)
    all_processes = list(processes)
    selected: list[tuple[str, Process]] = []
    for process in all_processes:
        if process.uid != uid or process.cwd is None:
            continue
        if is_concurrently(process, root_text) or is_air(process, root_text):
            selected.append(("supervisor", process))
        elif is_api(process, root_text) or is_vite(process, app_dirs(root_text)):
            selected.append(("app", process))
        elif basename(argv(process)[0]) == "api" and process.cwd == root_text and any(
            parent.pid == process.ppid and is_api(parent, root_text) for parent in all_processes
        ):
            selected.append(("app", process))
        elif is_launch_wrapper(process, root_text, all_processes):
            selected.append(("wrapper", process))
    return selected


def stop_apps(
    root: Path,
    environ: dict[str, str] | None = None,
    discover: Callable[[], list[Process]] = discover_processes,
    ports_for: Callable[[int], set[int]] = listening_ports,
    listeners: Callable[[int], set[int]] = listener_pids,
    send: Callable[[int, signal.Signals], None] = os.kill,
    sleep: Callable[[float], None] = time.sleep,
    clock: Callable[[], float] = time.monotonic,
    uid: int | None = None,
) -> Result:
    try:
        expected_ports = VITE_PORTS | {api_port(root, os.environ if environ is None else environ)}
        owner = os.getuid() if uid is None else uid
        initial = targets(discover(), root, owner)
        observed_ports = set().union(*(ports_for(process.pid) for _, process in initial)) if initial else set()
    except (OSError, RuntimeError, ValueError) as error:
        return Result(False, [f"make stop: {error}"])

    def current() -> list[tuple[str, Process]]:
        return targets(discover(), root, owner)

    def revalidate(original: Process) -> Process | None:
        # The real implementation reads one PID here; injected discovery keeps
        # unit fixtures simple and never touches the host process table.
        processes = discover_processes(original.pid) if discover is discover_processes else discover()
        return next((process for process in processes if process.pid == original.pid), None)

    def signal_targets(items: list[tuple[str, Process]], sig: signal.Signals) -> str | None:
        for _, original in items:
            now = revalidate(original)
            if now is None or not same_process(original, now):
                continue
            try:
                send(original.pid, sig)
            except ProcessLookupError:
                continue
            except PermissionError:
                return f"permission denied stopping owned process {original.pid}"
        return None

    try:
        error = signal_targets([item for item in initial if item[0] == "supervisor"], signal.SIGTERM)
        error = error or signal_targets([item for item in initial if item[0] != "supervisor"], signal.SIGTERM)
        if error:
            return Result(False, [f"make stop: {error}"])
        deadline = clock() + GRACE_SECONDS
        survivors = current()
        while survivors and clock() < deadline:
            sleep(0.2)
            survivors = current()
        error = signal_targets(survivors, signal.SIGKILL)
        if error:
            return Result(False, [f"make stop: {error}"])
        if survivors:
            sleep(0.2)
        remaining = current()
        checked_ports = expected_ports | observed_ports
        occupied = {}
        for port in checked_ports:
            pids = listeners(port)
            if pids:
                occupied[port] = pids
    except (OSError, RuntimeError) as error:
        return Result(False, [f"make stop: {error}"])

    messages: list[str] = []
    if remaining:
        messages.append("owned app processes remain: " + ", ".join(str(process.pid) for _, process in remaining))
    if occupied:
        target_pids = {process.pid for _, process in remaining}
        for port, pids in sorted(occupied.items()):
            kind = "owned" if pids & target_pids else "unrelated"
            messages.append(f"{kind} listener(s) remain on port {port}: " + ", ".join(map(str, sorted(pids))))
    if messages:
        return Result(False, [f"make stop: {message}" for message in messages])
    return Result(True, ["make stop: no owned local app processes remain"])


def main() -> int:
    result = stop_apps(Path(__file__).resolve().parents[1])
    print("\n".join(result.messages))
    return 0 if result.ok else 1


if __name__ == "__main__":
    raise SystemExit(main())
