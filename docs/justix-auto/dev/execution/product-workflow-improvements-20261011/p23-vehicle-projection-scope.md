# P23v — Existing inventory ownership and custody projection

Application hub accepts this narrow precursor to original P23/R3: existing
vehicleDTO and both mapper paths omit the already persisted owner/custodian.
Original P23 explicitly requires these distinct facts in vehicle/order details.
This is completion of that handoff, not a new endpoint or product policy.

Execute after P22 and before P23a/original P23, as a separate sole-writer packet
with a fresh worker context ceiling of 6k. Exact source ownership:

- internal/modules/inventory/handler/dto.go
- internal/modules/inventory/handler/vehicle_custody_test.go (new pure units)

Read internal/AGENTS.md, these paths, the relevant Vehicle/VehicleRow model
fields and final result-20.md only. Run tools/check-git.sh before edits. Preserve
all predecessor changes. Add nullable ownerCompanyId and custodianCompanyId
fields from the persisted model to the existing vehicle DTO and consistently
populate both toVehicle and receiptBody. Preserve null after actual customer
delivery. Never derive ownership from placement, invent company names, query
another module's tables or change existing read authorization. No new service,
repository, endpoint, permission, schema or stock transition is needed.

Meaningful pure mapping units cover owner equals custodian, supplier ownership
with dealer custody, cleared nil values, and the receipt mapper. One affected run:

`env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/inventory/handler -run '^TestVehicleCustodyProjection'`

No other checks, Git writes, SQL/runtime, review or QA. Report result-23v.md with
exact JSON nullability, mapping/unit evidence and source hashes; release writer.
P23 frontend owns data.ts and the existing vehicle/order detail hosts already;
it must consume the resulting fields without replacing server authority.
