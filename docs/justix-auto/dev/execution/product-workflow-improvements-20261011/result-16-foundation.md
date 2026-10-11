# P16 foundation continuation result

Status: DONE. Source writer released after this report. Work stayed on
`feature/company-commerce-workflows-20261011` at the supplied base
`ab088d97054fbabac65caae1f3f8bfe7f71fbe74`; no Git writes were made.
`bash tools/check-git.sh` passed before source edits. Dirty predecessor files
outside the assigned paths were preserved.

## Claim handoff for P17/P18/P21

`model.SupplyClaim` now has nullable `MaterializationUnitOrdinal *int32` in
addition to immutable materialization kind, reference UUID and timestamp. A
materialized claim must have a positive ordinal; VIN allocation uses ordinal
1. A quantity receipt may use one kind/reference pair for several distinct
claims, with ordinals 1, 2, ... for its units. Migration 041 now uniquely
indexes `(materialization_kind, materialization_ref,
materialization_unit_ordinal)`, so applying the same receipt unit twice to
different claims conflicts while distinct units can share the receipt UUID.
Unmaterialized claims retain null reference/ordinal/timestamp and an empty
kind. Callers must provide the ordinal when materializing, use a stable ordinal
on retries, and keep the existing claim ID through conversion.

Migration 041 accepts a materialized claim linked to a reservation without an
order, or linked to an order after conversion. The reservation ID remains
source provenance when order linkage is added. `UpdateClaim` maps the ordinal
and rejects changing state, kind, reference, ordinal or timestamp on an
already materialized claim while allowing its order ownership fields to move.
Held and materialized claims continue to consume lot capacity; released claims
do not. This continuation does not implement allocation/receipt flows or
current/replacement allocation history, which remain P17b/P18b/P21a work.

## Lot rules and evidence

Create and update reject `totalQuantity` above signed SQL `integer` maximum
(2,147,483,647) before converting it to Go `int`. An update with any held or
materialized claims rejects changes to model ID, specification version,
exterior color or interior color. It still permits a quantity increase and
expected-arrival update; a lot with no committed claims may change identity
subject to catalog validation. The existing floor on committed quantity is
unchanged.

The affected service unit family passed once:

`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service -run '^TestSupply'`

The new cases cover the SQL integer boundary on create/update, all four
identity fields under held and materialized claims, permitted quantity/arrival
changes, and identity editing after release. Migration 041 is still unapplied;
there was no database execution, repository integration test, build, lint,
browser check or deployment. The schema uniqueness and repository guard are
therefore static implementation evidence pending normal later integration.
