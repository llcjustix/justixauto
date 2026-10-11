# Application acceptance: partner stock discovery

Accepted proposal SHA-256:
`320a588a5fa6fe250424973c8a8ebbd4701d66ef85985858e65e36bfa4c9265c`.
Accepted packet SHA-256:
`add7d2cea15f4f923cddd80971d4e1ddd793a872b53db290a2c352a48e330533`.

This closes the concrete original R3/P19a U2 gap: a dealer cannot use its own
tenant inventory list to choose a partner distributor's VIN. The narrow read
projection, active stored party/partnership checks, safe DTO and bounded page
are accepted. Existing inventory visibility and mutation authorization remain
unchanged; no read creates a hold or guarantees later availability.

Application hub explicitly selects fresh gpt-6-sol/high under agent-workflow's
risk-escalation rule. The new tenant boundary and typed inventory/commerce/app
composition require complete authorization and constructor coverage; earlier
partial composition handoffs justify the bounded 16k escalation. No inherited
coordinator model, additional product policy or verification gate.

P18c is source complete/released; its result-18c.md is the required composition
handoff. Preserve its background-free app.New, runtime hook and ledger adapter.
Original packet paths and unit-only verification apply. Additional constructor
callers/fixtures require exact-path reporting before editing, with no optional
port workaround. No migration, runtime activation or deployment is authorized.

The dormant QA definitions remain dormant. Run only relevant unit selections
once after their actual changes; no repeat of expiry, ledger or unrelated
previously passing suites. P19a consumes the final implemented API handoff.
