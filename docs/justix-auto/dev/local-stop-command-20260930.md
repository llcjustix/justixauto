# Local app stop command — 2026-09-30

User request: “create make command to stop apps (clear ports)”.

Status: implemented; affected unit tests passed. This bounded local tooling change does not
change product rules or close a product backlog item.

Branch: `infra/local-stop-command`. Base `dev`:
`0e74dac4258402747e3a5cdc748a671112cf2fd0`; carried-forward committed feature:
`3cc124c5a5858a5eaa2991edafc2771ee7232966`. Version control proved ancestry,
fast-forwarded the new branch, and switched at the same source SHA. Existing
unrelated `.claude` changes and the manual-flow QA note remain untouched.

## Approved scope

Application coordinator: `/root`. Infrastructure authority:
`/root/local_db_cleanup` (DevOps scope decision in this session).

- Add `make stop`, one small Python standard-library helper, focused unit
  tests, and README usage. No new dependency or process supervisor.
- Discover existing JustixAuto API/Vite processes and their Air/concurrently
  supervisors, including sessions started before the helper and standalone
  `make api` / `make web` runs.
- Establish ownership from user, repository/app working directory, known
  entrypoint and ancestry. A port number alone never authorizes a signal.
- Stop supervisors and then surviving owned app processes, using graceful
  termination and a bounded wait. Revalidate process identity before signaling;
  force termination applies only to verified surviving owned processes.
- Respect `HTTP_ADDR` environment precedence, safely read `.env` without
  executing it, and default to port 8080. Include observed owned listener ports
  and the four Vite ports 5191–5194.
- Repeated stop is harmless. Report unrelated port occupants without signaling
  them; report failure if app processes or required ports remain occupied.
- Keep database/Docker, user shells and unrelated applications running.

## Execution and verification

Packet: `execution/local-stop-command-20260930/packet.md`.
Worker result: `execution/local-stop-command-20260930/result.md`.
One implementation writer. Only the affected mocked unit tests run, once per
relevant change. No automatic review/QA, build, lint, typecheck, browser or
integration checks. Do not invoke the live stop command while implementing;
the current development session remains running.

No production or deployment action is authorized by this task.

## Result

`make stop` is implemented in `Makefile` through `tools/stop-apps.py`, with
README usage and focused tests in `tools/test_stop_apps.py`.
The final affected mocked suite passed: 10 tests, using
`PYTHONDONTWRITEBYTECODE=1 python3 -m unittest discover -s tools -p 'test_stop_apps.py'`.
No live stop was run; the existing app session remains running. This establishes
unit coverage, not live process-release or release readiness.

The pre-completion canonical record and SHA-256 manifest are preserved under
`var/backups/local-stop-command-20260930/`.
