# P19b history completion result

Status: DONE. The accounting read DTO now returns each evidence record's persisted `submittedBy` and optional `decidedBy`, plus an optional `refund` record. The refund record contains `status`, `requestedBy`, `requestedAt`, `requestReason`, and only when persisted as confirmed: exact integer-minor `amount`, `externalReference`, `attachmentIds`, `refundedOn`, `confirmedBy`, and `confirmedAt`. Existing aggregate fields, actions, and mutation request bodies are unchanged.

The shared `ReservationPayment` surface renders actor IDs and the returned request provenance. Its requested state continues to use aggregate `refundObligation`; it does not treat the record's unfinalized amount as money owed or returned. It renders the external refund amount/date/reference, confirming actor/time, and authenticated private attachment links only for a confirmed record; a requested refund has no fabricated confirmation display.

## Source hashes

```text
ae538eb31181f26c2141c3185f3a50e08e6bff9964422894c61340a5bbbb2eb6  internal/modules/commerce/handler/reservation_payment.go
c4e3ff6896235a0e0b629c2fa845cfd43e43b05fa9d51970869c94fecec8b24f  internal/modules/commerce/handler/reservation_payment_history_test.go
828035caa0ec21059bdde0b047dbe106557469d2ac69c848408501f0bb8136d7  web/apps/realization/src/data.ts
a46c819bef4872b758d93b8270ba2bc26428484995c73afe028c6a27178a10a4  web/apps/realization/src/reservation-payment.tsx
f673c1fbbaaf335f4c1ca6b7411c6e67cbeae5ff13a6fdf63a50bf2e434fe0cd  web/apps/realization/src/reservation-payment.test.tsx
```

## Unit evidence

```text
env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/handler -run '^TestReservationAccountingHistory'
ok  justixauto/internal/modules/commerce/handler  0.650s

npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/reservation-payment.test.tsx -t 'P19b reservation accounting'
Test Files  1 passed (1); Tests 6 passed (6)
```

The handler unit covers no refund, requested refund with all confirmation-only fields absent, and confirmed refund with a large exact amount, date, actor IDs, and private attachment ID. The rendered interaction fetches accounting through the original workspace host and verifies submitted/decision actors, requested reason, confirmation-only data, and a private refund link.

Limitations: this packet exposes persisted IDs only; it does not resolve display names, alter authorization, mutate accounting, or prove database/runtime behavior.

Writer release: RELEASED. No Git writes were made.
