# P18a1 — Reservation evidence, refund and credit ledger

Original P18a, canonical `product-workflow-improvements-20261011`, R3/R10/R13; application hub `/root`. Base `4fa683ba4708bb71031534260ec5cb8742ce9977`; milestone `ab088d97054fbabac65caae1f3f8bfe7f71fbe74`; branch `feature/company-commerce-workflows-20261011`. Record actual predecessor SHA/source hashes at dispatch. Dependencies: P04 authority, P17a/P17b completed reservation/holder interfaces, P16 foundation claim result. Context ceiling 18k; stop with compact findings before exceeding.

## Read and ownership

Read AGENTS.md, docs/justix-auto/dev/agent-workflow.md, internal/AGENTS.md; plan.md execution rules/R3/R10/R13 and S2/S4; p18a-split-proposal.md; original packet-18a.md; result-04.md, result-16-foundation.md; actual P17a/P17b results and accepted reservation-order-allocation-seam.md. Target http-domain.md §1 exact Money/revision, §3 evidence semantics, §5 document privacy only; latest approved behavior overrides old architectural proposals. Read actual reservation interfaces in small slices after source lock acquired; no full board/mocks/fixtures as policy.

Exact writable paths:

- `internal/modules/commerce/model/reservation_payment.go`
- `internal/modules/commerce/service/reservation_payment.go`
- `internal/modules/commerce/service/reservation_payment_test.go`
- `internal/modules/commerce/service/ports.go`
- `internal/modules/commerce/service/reservation.go`
- `internal/modules/commerce/repository/reservation_payment.go`
- `internal/modules/commerce/repository/store.go`
- `migrations/000043_reservation_payments.up.sql`
- `migrations/000043_reservation_payments.down.sql`

Exclusive lock: source-writer. No PostgreSQL/browser/live resource. Run `bash tools/check-git.sh` before application edits. No Git writes or unowned files. Migration 043 must be unused/unapplied; do not rewrite applied schema. Handler/module/invoice integration is P18a2.

## Coherent behavior and transaction contract

Implement reservation evidence and its settlement ledger using existing handler→service→repository architecture, integer Money and existing Files.Share. Use P04 commerce.payments.accept for supplier accountant decisions; preserve buyer submission/current permission, company-admin grants and party/tenant scoping. Do not demand commerce.trade as an extra accountant review permission, recent second factor, or newly verified status for legitimate old-obligation servicing.

Persist immutable evidence ID, reservation/accepted-terms association, amount/currency, external reference, attachment IDs, submitted actor/time and submitted→accepted/rejected decision actor/time/reason. Validate positive exact amount, accepted terms currency and agreed advance/open capacity using accepted plus pending claims; never invent an advance percentage or default. Reuse existing external-evidence validation/business-key semantics. Pending/rejected do not reduce balances; duplicate or stale decisions fail without mutation. A new submission after rejection is a new immutable record, not deletion of history.

Acquire the authoritative reservation/accounting row lock before reading totals or changing evidence, refunds or credits. Adopt the predecessor's documented lot/reservation/order lock order across P17/P18; do not introduce an inverse order. Recheck accepted terms/version/state after locks, including converted/cancelled/refund-settling states. Return a view containing accepted total, pending total/count, accountant-confirmed refunded total, refund obligation/status, credit/order identity and evidence-ever-submitted. The latter includes rejected history. P18b uses this view within its caller's transaction; an unlocked preflight snapshot cannot authorize release/conversion.

Cancellation intent with accepted advance records a full-refund obligation. Merely requesting or uploading a refund does not confirm it. Pending evidence must receive an explicit accountant decision before final settlement; do not freeze a final refund amount and silently ignore later acceptance. Once evidence decisions are complete, accountant confirmation records actual external refund evidence in the exact full accepted amount/currency, preserving the immutable accepted total. Wrong amount, currency, actor, source, duplicate confirmation or credit-linked source fails. Refund confirmation authorizes no bank transfer and performs no fee/retention/partial refund arithmetic.

Expose one transaction-bound release-eligibility guard: no pending evidence and either no accepted advance or full accountant-confirmed refund, with no converted/order-owned obligation. The payment service never releases stock/claims. P18b1 owns final cancellation and idempotent release; original rejected evidence still exists after settlement. Once cancellation/refund/credit becomes terminal, stale payment submission/decision cannot revive the reservation or create uncovered money.

Provide transaction-bound source-order credit linkage for P18b2: exact reservation/order/party/currency, accepted evidence provenance, immutable amount, no outstanding pending/refund settlement, and one durable source→order link. A retry cannot create another credit or redirect one to another order. Credited money cannot also be refunded through reservation cancellation. A zero-advance conversion needs no fabricated accepted payment. P18b2 passes its existing Store/Bind context so order/claims/holders/credit/event share commit or rollback.

Ledger schema/repository must support a nullable issued-invoice association with durable uniqueness for P18a2; order credit exists before an invoice. Expose locked lookup/attach/read primitives with source identity and clear zero-credit ordinary-order behavior. Do not implement invoice calculations here and do not create commerce_payment_evidence from reservation evidence. P18a2 consumes the same credit once, preserving invoice face total/schedule. All ledger tables remain commerce-owned; Files.Share uses only existing document service port through bound context. No new app adapter or foreign-module table access.

Files are private and shared only between legitimate parties with correct source binding; foreign files fail. Refund evidence uses the actual submitting owner's document authority, never a hardcoded buyer-only owner. Preserve read authorization, audit actor/time/reason and safe event references. Fake transaction rollback must include document sharing, reservation state/revision, evidence, refunds, credits and events; errors cannot leave a partial accepted/refunded/credited state.

## Affected meaningful unit cases

Keep cases in the owned service test file; use actual public service actions and transaction fakes, not helper-only assertions.

- `TestReservationPayment`: free-by-default/no fabricated debt; submitted/rejected/accepted lifecycle; pending capacity, overpayment/currency/exact integer validation, immutable reference, rejected-history flag, duplicate/stale action, tenant/party/permission denial and permitted accountant without trade. Private share owner/bound context and share failure rollback.
- `TestReservationRefund`: pending blocks settlement/release; accepted cancellation creates full obligation without paying it; accept/reject remaining pending before final refund; exact full accountant-confirmed amount only; wrong currency/actor/amount/duplicate denied; accepted total stays immutable; guard becomes eligible after confirmation; no second confirmation/release authorization through converted source.
- `TestReservationCredit`: immutable source/order link and accepted provenance; zero money; wrong party/order/currency; pending/refund conflict; duplicate/redirect attempt; credit-versus-refund/cancel stale interleaving using locked-state rereads; caller transaction rollback on credit write/CAS/event failure.
- Across families inject repository, share, revision and event failures and deep-copy fake state to establish unit-level all-or-nothing behavior. Lock-observation/interleaving fakes are evidence of rechecks, not real SQL concurrency proof.

Run once after the relevant change from repo root:

`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service -run '^(TestReservationPayment|TestReservationRefund|TestReservationCredit)($|_)'`

Use these exact family names or suffixed variants so the selector actually runs the cases. No other automatic checks, integration/DB tests or QA/review jobs. If existing shared fixture compilation needs an unowned path, report exact path; do not run broader tests to hide it.

## Handoff and stop

Report changed paths/hashes, actual before/after SHA supplied by hub/version-control, method signatures, expected revisions, lock order and transaction ownership, schema uniqueness, authoritative money view, refund state transitions, source credit/invoice-association contract and exact command/result/exclusions. Feed P18a2 and P18b1/b2; no fabricated route names or release-readiness claim.

Stop on missing predecessor linkage/lock primitive, unavailable source-writer, required unowned path, applied migration collision, new fee/threshold/schedule/transfer policy, unsupported document transaction guarantee, or 18k context ceiling. Do not resolve gaps from demo fixtures. If canonical scope changes report RESLICE_REQUIRED; otherwise return a bounded continuation proposal.
