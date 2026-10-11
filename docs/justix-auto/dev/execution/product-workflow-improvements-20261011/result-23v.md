# P23v — Existing inventory ownership and custody projection result

Status: DONE. The sole source-writer lease is released.

`vehicleDTO` now exposes `ownerCompanyId` and `custodianCompanyId` as `*string`
without `omitempty`; therefore present identifiers serialize as JSON strings and
cleared persisted values serialize as JSON `null`. Both `toVehicle` and
`receiptBody` copy the persisted pointers directly, preserving supplier ownership
with dealer custody and customer-delivery clearing without derivation or lookup.

`TestVehicleCustodyProjection` covers owner equals custodian, supplier ownership
with dealer custody, both cleared nil values, and receipt mapping. The assigned
single affected test command passed:

```text
env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/inventory/handler -run '^TestVehicleCustodyProjection'
ok   justixauto/internal/modules/inventory/handler  0.443s
```

`bash tools/check-git.sh` passed before edits. No Git write, SQL/runtime action,
review, QA, or extra checks ran.

## Final source SHA-256

```text
d9d2544b53f24dcbacf7c1f2c2acf387451d8bfe7ec2ea639f953a5ada94d18c  internal/modules/inventory/handler/dto.go
d66d14aabb5dd230ed97b2b520518dd5717ba96454166a318467025e77a63ea4  internal/modules/inventory/handler/vehicle_custody_test.go
```
