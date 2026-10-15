# Bounded continuation after old API exit

Primary `/root`, 2026-10-14, extends `final-validation.md` operational acceptance.
The first activation stopped at `verified` with a process identity error.
Runtime inspection established: recorded old API37081 is absent,8090 has no
listener, Air31138 remains paused with the same start/executable, Vite26877/5191
is preserved, schema is clean45 and resume count is0. No backup or migration
was reported. A termination polling race is the likely explanation, not a
retrospectively proven signal trace.

Preserve the original receipt unchanged at
`/private/tmp/justixauto-warehouse-local-activation-bkf4lka1/activation.json`.
The runtime owner is authorized to amend only its `local-activation.py` with
`--continue-drained-receipt`. Bind the original receipt/hash and accepted
source/HEAD, verify original Air/Vite/PostgreSQL identities again, require old
PID absent, no8090 listener, clean45 and zero other DB clients. Do not repeat
shutdown. Use a new continuation receipt, fresh private backup and archive
validation, then exactly one Air resume and the accepted clean46/readiness/
binary checks. Correct termination polling to accept disappearance only with
the old listener gone, and continue rejecting PID reuse. Record revised helper
hash and continuation outcome in `runtime-result.md`.

This is recovery within the authorized local cutover, not permission for a
reset, force/down migration, automatic restore, live fixture write or unrelated
process action. Application source and HEAD remain frozen. Any further failure
requires diagnosis before another runtime mutation.
