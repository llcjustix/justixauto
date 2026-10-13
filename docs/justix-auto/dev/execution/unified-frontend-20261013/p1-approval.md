# P1 bounded dispatch approval

Application hub accepts P1's four-path scope and lifecycle contract from
packets/packet-01.md (read SHA-256
58a97d3074395cede8d402bcc6cd03e284e67dd5eff915cc91bceab052b29311)
and the plan's P1 handoff. SessionApi.scopeKey is the only public addition;
retain existing session and HTTP signatures, server permissions and wire APIs.
Slicer has released its need to read these changing source files; later packet
artifacts may finish independently. No review/QA gate is being added.

Use a fresh explicit gpt-6-sol/high job, 16k total context, for this bounded risk
escalation: one persistent app needs correct ordering across pending context
changes, late HTTP/CSRF/401 responses, cache retirement and unmounted dialogs.
Preserve all prior source; one writer in the main checkout. This is source-only.

Unit command correction accepted before dispatch: keep the packet's named Kit
project command. Run the two existing Realization suites directly from web:

`npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/accountant.test.tsx src/commercial-access.test.tsx`

Every named suite must appear; prior generic workspace selection omitted the
app. The slicer may materialize this command-only correction and record its new
packet digest without changing accepted source ownership or behavior. No builds,
lint, typecheck, browser, integration, ReactDoctor or dormant QA jobs are needed.
Only affected units once per relevant change; no tests solely for counters.

Assigned writer /root/unified_session, result-01.md. Source ownership is exactly
kit/src/session.tsx, kit/src/http.ts, new session-lifecycle.test.ts and
http-context.test.ts. Stop for a concrete unowned compatibility path if needed.
