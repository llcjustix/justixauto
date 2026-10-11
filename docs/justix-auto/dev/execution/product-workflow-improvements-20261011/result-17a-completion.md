# P17a completion result

Status: SOURCE COMPLETE; source writer released. This continues the partial
`result-17a.md` on `feature/company-commerce-workflows-20261011` from local
HEAD `ab088d97054fbabac65caae1f3f8bfe7f71fbe74`, preserving all dirty
predecessor changes. `bash tools/check-git.sh` passed before source edits.
No Git write, deployment, database execution, OpenAPI/integration/browser
check, lint, typecheck, build, review or QA loop was run.

## API and persistence

- `POST /api/v1/commerce/reservations` stores a dealer request without a
  hold. Future requests may omit unknown economic terms. Each immutable
  requested snapshot contains the exact source line IDs, VIN or lot IDs,
  model, specification, colors and quantities, as well as any desired terms.
- `GET /api/v1/commerce/reservations` and `GET .../reservations/:id` scope
  reads to the buyer or supplier company. Structured JSON exposes party IDs
  and names, requested/offered/current snapshots, digests, snapshot revisions,
  aggregate revision, expiry, optional continuation order ID and party-specific
  next actions.
- `POST .../reservations/:id/supplier-confirmations` with If-Match fixes
  positive exact unit price/currency, future expiry and optional positive
  advance/deadline. The supplier may supply one quantity per existing source
  line. A complete exact match to the requested snapshot activates in the
  transaction; any changed or initially incomplete requested terms leave an
  offer without holds.
- `POST .../reservations/:id/acceptances` with If-Match and the exact offered
  digest activates a changed offer. Stale versions/digests, wrong parties or
  status, invalid sources, sold/held VINs and exhausted lots reject. The
  requested snapshot remains intact. An active reservation cannot confirm or
  activate again. Accepted price is never edited in place.

The new migration stores reservation and source-line rows, three JSON term
snapshots with digests and revisions, party IDs, expiry, version, and nullable
future order reference. No order row is fabricated for a reservation.

## Ports, claims and lock order

`commerce.New` now requires the typed `ReservationStock` port, which embeds
the existing order `Stock` port and adds reservation reserve/release methods.
The app adapter supplies the inventory stock service through a narrow holder
interface and creates `inventory.Holder{Type:"commerce-reservation", ID:id}`;
the same bound context reaches inventory's transaction. There is no runtime
assertion or silent fallback. The inventory root still exports its distinct
reservation holder methods for subsequent consumers.

Activation row-locks the reservation, then locks future lots in sorted ID
order, checks their current source identity and capacity, and creates one
held `SupplyClaim` per accepted future unit. Each claim retains the same
reservation ID and source reservation-line ID for P17b/P18b/P21a. Inserts
run in batches of 500 with a context cancellation check per batch; no
arbitrary business quantity cap was added beyond the SQL integer range.
VIN IDs are sorted and passed together to inventory, which locks owned VINs
and checks active holders. Claim inserts, VIN holds, reservation update and
event append share the commerce transaction; an error rolls it back.
P17b still owns VIN allocation, holder reassignment and allocation history.

The coordinator granted one exact extension in
`p17a-supply-handler-seam.md`: `internal/modules/commerce/handler/supply.go`
uses `mapSlice(views, toSupply)` to repair the preexisting P16 mapper
signature mismatch that blocked app/handler compilation. No supply semantics
changed.

## Unit evidence

After final relevant changes, these passed with pinned Go, `-vet=off` and
`GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011`:

- `bash tools/go.sh test -vet=off ./internal/modules/commerce/service`
- `bash tools/go.sh test -vet=off ./internal/modules/commerce/handler -run '^TestReservationHTTP'`
- `bash tools/go.sh test -vet=off ./internal/app -run '^TestWorkflowPort17($|_)'`
- `bash tools/go.sh test -vet=off ./internal/modules/inventory/service`

The commerce service units exercise real request/confirmation/acceptance
transitions through fake repositories and ports: no request hold, exact VIN
activation once, changed quantity awaiting exact buyer digest, stale digest,
oversell, held/sold VIN rejection, wrong party and permission, invalid terms,
claim provenance and transaction rollback after an update failure. Inventory
units exercise distinct reservation/order holder namespaces, idempotent same
holder retry and release. The app case checks holder namespace and bound
context across reserve/release. The handler unit checks structured snapshot
serialization and revision fields. These are unit fakes; the migration and
cross-module database behavior await the separate human integration gate.

Early compile failures in the continuation exposed `l.ID` versus `l.LineID`,
`gorm.Locking` versus `clause.Locking`, and the P16 supply list mapper; all
were repaired before the passing unit selections above. Inventory passed on
its first run after adding the holder unit and was not repeated.
