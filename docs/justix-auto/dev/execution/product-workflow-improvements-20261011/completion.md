# Company and commerce workflows — source completion

Application hub `/root`, 2026-10-11. All thirteen requirements in the
[authorized feature](../../product-workflow-improvements-20261011.md) are
implemented in source. Every source writer has released its lease. This closes
this feature's implementation, not the historical application backlog, release
readiness or deployment. The [delivery map](delivery-map.md) identifies final
reports; earlier partial reports remain historical and are not final acceptance.

## Delivered behavior

- R1/R2: company-level fixed dealer/distributor type, reflected in sessions and
  UI; retail is available to both. Admin can create a company from name and
  initial admin credentials, then assign its type and verify it manually.
  Company users complete legal form and requisites themselves. Server and UI
  guard new commerce while preserving setup, history and old obligations.
- R3: dealer requests for existing VINs or future model/specification/color
  quantities; supplier terms and quantity confirmation, buyer acceptance when
  changed, fixed accepted price, optional advance and extension. VIN assignment
  can follow later. Separate mutually accepted purchase/consignment terms
  govern conversion and handover; zero/some/all VIN conversion is supported.
  Accepted advances credit the invoice once. Partial purchase handover requires
  the explicit accepted order agreement, with no invented minimum payment.
- R4/R5: same-company warehouse moves; distinct legal owner and custodian;
  per-VIN supplier return request and dealer confirmation/rejection. Confirmation
  checks current custody, active holds and destination capacity and links one
  physical movement fact. Pending, rejected and confirmed history survives
  reopening and loss of dealer inventory access. Return does not cancel debt.
- R6–R8: CRM passport fields and private file binding; Islamic-line terminology;
  percent-primary installment down payment with exact synchronized money amount.
- R9: editable data-filled retail and B2B templates, private external documents,
  paper signature scans, immutable signed snapshots and separate amendments.
- R10/R13: separate accountant invoice/payment authority and work surface; sale
  displays the invoice. Submitted payment evidence reduces debt only after
  accountant acceptance, including installments. Full advance refunds require
  accountant confirmation of the actual external refund.
- R11/R12: insurance request/status/actions in the sale dialog; explicitly demo
  scoring before the first own-installment contract signature, with persisted
  freshness. No real scoring provider or insurance coverage is inferred.

The user-approved unpaid-advance expiry rule is implemented with a bounded
server runner: automatic release requires that no payment evidence was ever
submitted. Pending, accepted or historical rejected evidence is protected.
The runner is source only in this delivery; it was not activated in a runtime.

## Final consignment and return handoff

P20/P21/P22 maintain one per-VIN authority across consignment handover, retail
reservation/delivery and return. Ordinary stock eligibility requires current
owner and custodian; the dealer's consigned retail discovery exposes only IDs
under `retail.read`. P23a hydrates real inventory facts, handles every page and
preflights manual and offer-seeded sale submission without changing the sale
request body. Server mutation remains the final authority.

P23v maps nullable persisted owner/custodian fields in both existing inventory
DTO paths. P23 displays these actual facts, accepted mode/settlement/partial
handover terms and credit-inclusive invoice balances. Supplier creation uses
the consignment unit revision; dealer decisions use the return request revision
and server-advertised actions. Return history is read independently of current
vehicle access. Failed post-write refresh locks further actions and offers a
refresh-only retry without replaying the successful command.

Final reports: [P20](result-20.md), [P21a1](result-21a1.md),
[P21a2](result-21a2.md), [P21a3](result-21a3.md),
[P21b1](result-21b1.md), [P21b2](result-21b2.md),
[P21b3](result-21b3.md), [P21b4](result-21b4.md),
[P22](result-22.md), [P23v](result-23v.md),
[P23a](result-23a.md), [P23](result-23.md).

## Verification evidence and limits

Affected Go and React unit selectors are recorded with commands, outcomes and
source SHA-256s in each final report. The final P22 return service and app
adapter families passed. P23v's four mapping cases passed. P23a's helper,
offer-seeded form and manual form suites passed (15, 21 and 13 tests), and P23's
final rendered interaction suite passed all 9 tests. Earlier reports cover the
remaining requirements and approved repairs. No aggregate test count or full
final-SHA integration result is inferred from independent packet histories.

P23a's report explicitly records an accidental second run of the unchanged
helper suite. Earlier execution deviations and the milestone commit hook's
generated API snapshot are retained in their reports; they are not hidden by a
claim of perfect process compliance. Subsequent Git commits disable hooks and
do not regenerate OpenAPI.

No lint, typecheck, build, vet/race, browser/E2E, integration, database or
migration execution, runtime restart or deployment was used to establish final
completion. Unit fixtures do not prove PostgreSQL locking/rollback, installed
migration constraints, deployed routes, runtime scheduler behavior or release
readiness. Migrations 035–045 are source changes and were not applied by this
feature execution. Environment activation remains a separate authorized task.

## Git and record provenance

Branch: `feature/company-commerce-workflows-20261011`.
Full feature base: `4fa683ba4708bb71031534260ec5cb8742ce9977`.
Earlier local milestone: `ab088d97054fbabac65caae1f3f8bfe7f71fbe74`.
The final local source commit, explicit staged scope and source status are
recorded in [source-git-result.md](source-git-result.md). The source commit is
followed only by a scoped documentation commit; no push or dev/main promotion
is part of this handoff. Whole-feature provenance is base through the final
source commit, not only the last commit's parent diff.

Canonical pre-completion byte copies and SHA-256s are in
[entry/source-completion/manifest.md](entry/source-completion/manifest.md).
Existing historical specs, mocks and task states are preserved.
