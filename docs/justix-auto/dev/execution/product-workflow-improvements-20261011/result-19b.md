# P19b result — reservation advance and refund accounting UI

Status: UI CORE COMPLETE; ORIGINAL P19b HISTORY ACCEPTANCE UNFINISHED because the accounting read DTO omits persisted actor and refund provenance. Sole source-writer lease RELEASED after this report. Work used the main checkout on `feature/company-commerce-workflows-20261011` at supplied HEAD `ab088d97054fbabac65caae1f3f8bfe7f71fbe74`, preserving predecessor dirty source. `bash tools/check-git.sh` passed before source edits. No Git write was made.

## Behavior and handoff

- `ReservationPayment` is the shared accounting surface in the real Purchases reservation detail and the Billing supplier tab. It reads `GET /commerce/reservations/:id/accounting` under the company-scoped key `['reservations', companyId, 'accounting', id]`, displays agreed advance/deadline, accepted and pending amounts, refund obligation and confirmed amount separately, credit order/invoice IDs, and immutable submitted/accepted/rejected evidence with private file links, external reference, date, returned timestamps and decision reason.
- Buyer submission posts the exact decimal-string minor-unit `claimedAmount`, `paidOn`, `externalReference` and uploaded private `attachmentBindingIds` to `/reservations/:id/payment-evidence` with reservation If-Match. Supplier accountant accept/reject posts `reservationRevision`, affirmative confirmation or reason to `/reservation-payment-evidence/:id/{accept,reject}` with the current evidence If-Match. Dealer/supplier trader refund request sends reason; supplier accountant full refund confirmation sends the exact server `refundObligation`, external date/reference/private evidence and affirmative confirmation. No browser-side accepted balance or refund is projected from a submit/request.
- Controls require the actual selected company, permission (`commerce.trade` or `commerce.payments.accept`) and backend `allowedActions`. Review remains available for old obligations without a commercial-verification gate. A synchronous send guard blocks double submission. Failed or uncertain writes refetch the authoritative accounting DTO before retry; a failed read keeps writes locked until explicit retry. A stale selected evidence revision is replaced with the re-read row's revision.
- Billing's `['billing', 'reservations', companyId]` query pages the company-scoped reservation list and reads accounting for rows with accepted advance terms. Its visible queue distinguishes pending review, refund requested, refund confirmed, accepted and credited states. The same panel opens from the queue. Existing supplier invoice and retail invoice tabs remain. After a mutation the panel invalidates reservation list/detail/accounting, Billing queue/invoices, order invoices and order views, then refetches accounting. The server invoice `Paid` and `Outstanding` are rendered unchanged; reservation credit is never added or subtracted a second time.

The original P19b immutable-history acceptance is unfinished. `reservationEvidenceDTO` in `internal/modules/commerce/handler/reservation_payment.go` omits persisted `SubmittedBy` and `DecidedBy`. `reservationAccountingDTO` omits the persisted refund record's `RequestedBy`, `RequestedAt`, `RequestReason`, `ExternalReference`, `AttachmentIDs`, `RefundedOn`, `ConfirmedBy` and `ConfirmedAt`. Those fields exist in `internal/modules/commerce/model/reservation_payment.go`. The narrow continuation must extend the backend read DTO, the `ReservationPaymentEvidence`/`ReservationAccounting` types in `web/apps/realization/src/data.ts`, and the `ReservationPayment` history/refund display in `web/apps/realization/src/reservation-payment.tsx`. It must render returned actor IDs and refund provenance truthfully, with private attachment links; no actor names should be invented. This is remaining original P19b work, not a P19c option. Until then, the UI shows only returned evidence timestamps/reason and confirmed refund amount/status.

## Affected unit evidence

After the final relevant edits:

```text
npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/reservation-payment.test.tsx -t 'P19b reservation accounting'
Test Files 1 passed (1); Tests 5 passed (5)

npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/reservation-workspace.test.tsx -t 'P19a supply and agreement'
Test Files 1 passed (1); Tests 11 passed (11)
```

The initial P19b run passed 4/5 and exposed a stale evidence revision retained in the selected row; after changing review to resolve the row from the freshly read DTO, the affected selection passed 5/5. Named tests render the actual workspace and Billing host with HTTP/session fakes. They cover exact upload and mutation payloads/revisions, pending versus accepted/rejected and refund labels, permission denial, double submit, stale retry, Billing discovery, private file URL, and server credit-inclusive invoice paid/outstanding display. Unit fakes do not establish live database or release readiness. No build, lint, typecheck, React Doctor, browser, integration, DB, review or QA loop was run.

## Final source SHA-256

```text
362394276d96453dc887b8cd07a954dd243eead34bb7d2e8fec29d87b2cc736c  web/apps/realization/src/reservation-payment.tsx
1bdd75a3a28a828350ae5b82d4e6fac9a816be8210ad6ca8b68aaa438d1037ee  web/apps/realization/src/reservation-payment.test.tsx
6a0ed331ce0cb314098e56ff621c4d4e6e196998f8f4393f9444a1d40c6b7a31  web/apps/realization/src/reservation-workspace.tsx
6b650582fa5c9202824a899a29142da5dbe99526dee082c7c811adfc172f3e0a  web/apps/realization/src/data.ts
562a80d03badae9f2a69d5255956c2e7bdf9e82c7d067f43da74e01290700d1f  web/apps/realization/src/pages/billing.tsx
```
