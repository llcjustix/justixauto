# P17a result

Status: RESLICE_REQUIRED.

`bash tools/check-git.sh` passed before edits on
`feature/company-commerce-workflows-20261011` at
`ab088d97054fbabac65caae1f3f8bfe7f71fbe74`. P16 final schemas were consumed.
The partial source change adds reservation model/repository/service/handler,
the new 000042 reservation schema, `commerce-reservation` inventory holder
exports, and app adapter wiring. Future activation creates held P16 claims
with reservation/line provenance; confirmation requires complete terms and
unmatched or initially incomplete request terms wait for exact digest acceptance.

The required command failed twice during compilation:
`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service ./internal/modules/inventory/service`.
First failure: `reservation.go:177: s.requirePartner undefined`; this was
repaired locally. Second failure: `reservation.go:320: unavailableReservationStock
does not implement ReservationStock`, because its `ReleaseReservation` has an
extra company-ID argument. Inventory service was otherwise cached green.
No third repair or app selector was run, as required by the two-failed-repairs
stop rule. No Git writes, database/runtime/OpenAPI/E2E/lint/build checks occurred.

Required corrective packet: fix the exact fallback signature, add the specified
real transition/rollback reservation unit cases and `TestWorkflowPort17`, then
rerun only the original service/inventory command and narrow app selector.
The partial HTTP additions are POST `/commerce/reservations`,
`/:id/supplier-confirmations`, and `/:id/acceptances`; DTO currently exposes
IDs/status/digests/revision/next actions. P17b still owns allocation history.
