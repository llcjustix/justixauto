# P19b persisted history DTO and display completion

Application hub assigns the exact remaining original history acceptance from
result-19b.md. UI core and five named interactions are delivered; persisted
actor and refund provenance was not exposed by the existing accounting DTO.
No domain/schema/payment/permission change is needed. This is implementation
completion, not an optional enhancement or review/QA loop.

Fresh worker, fixed role model/effort, 16k context ceiling. Exact owned paths:

- internal/modules/commerce/handler/reservation_payment.go
- internal/modules/commerce/handler/reservation_payment_history_test.go
- web/apps/realization/src/data.ts
- web/apps/realization/src/reservation-payment.tsx
- web/apps/realization/src/reservation-payment.test.tsx

Map existing ReservationEvidence.SubmittedBy/DecidedBy as truthful actor IDs,
keeping timestamps/reasons and optionality. Add an optional structured refund
read record with its status, requestedBy/requestedAt/requestReason, exact amount,
externalReference, attachmentIds, refundedOn, confirmedBy/confirmedAt. Source
only the actual persisted model fields already present in ReservationMoneyView.
Do not invent actor names, evidence, business transitions or refunded dates.
Absent/unconfirmed fields remain absent; the aggregate status/amounts/actions
and existing request DTOs remain backward-compatible.

Render these returned values in the shared ReservationPayment history surface,
including private attachment links using its existing authenticated helper.
Requested and confirmed refund remain distinct. Confirming externally returned
money is still the existing accountant mutation; reading a record performs
no action. Keep all original company/party/permission guards and integer money.

Meaningful affected units: pure handler mapping for missing/requested/confirmed
refund and evidence actors, exact large money, dates/optional fields and private
IDs (no public URL or document content); rendered payment history after real
DTO fetch showing submitted/decision actors, request reason and confirmed
external refund/date/attachments without fabricating an unconfirmed result.

Run each affected selection once after the relevant changes:

`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/handler -run '^TestReservationAccountingHistory'`

From web:
`npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/reservation-payment.test.tsx -t 'P19b reservation accounting'`

No rerun of unchanged P19a workspace, service money ledger or other checks.
No lint/typecheck/build/React Doctor/browser/DB/runtime/integration/review/QA.
Check project Git root before source edits, apply_patch, no worker Git writes.
Assigned result: result-19b-history.md with final DTO contract, source hashes,
units, remaining limitations and explicit writer release. Canonical docs remain
root-owned; this assigned worker report is permitted.
