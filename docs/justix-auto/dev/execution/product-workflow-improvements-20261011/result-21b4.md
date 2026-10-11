# P21b4 — consigned retail candidate projection result

Status: DONE. The sole P21b4 source-writer lease is released. Work used
`feature/company-commerce-workflows-20261011` at local HEAD
`ab088d97054fbabac65caae1f3f8bfe7f71fbe74`; `bash tools/check-git.sh` passed
before edits. No Git write, SQL/database/runtime action, deployment, build,
lint, typecheck, browser/integration check, review or QA loop ran.

## Final API handoff for P23a

`GET /api/v1/commerce/consignments/retail-candidates?limit=<0..100>&offset=<>=0>`
is a company-scoped read requiring `RequireCompany()` and `retail.read`.
The service repeats the `retail.read` check, takes the dealer only from the
authenticated principal, and has no client company/dealer input. It returns
the standard `httpx.List` envelope with deterministic unit-ID ordering:

```json
{"items":[{"vehicleId":"<uuid>"}],"nextCursor":"<next offset>|null","asOf":"..."}
```

`limit=0` defaults to 50; the maximum is 100; `nextCursor` is the decimal
offset only when the returned page is full. P23a must consume all offset pages,
then hydrate each ID through the existing inventory vehicle-detail route. The
DTO contains no VIN, agreement, party, terms, digest, price, payment, debt,
commission or settlement field and creates no hold or other write.

The repository returns only rows for the active company that are `held`, have
the complete durable handover fields, and have neither a retail-deal association
nor a pending return. Prepared, sale-reserved, pending-return, sold and returned
VINs therefore do not appear; supplier/other-company calls receive no dealer
custody candidates. P21b remains the locked final sale authority.

## Unit evidence

After the final relevant change, the authorized combined selector passed once:

```text
env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/modules/commerce/service ./internal/modules/commerce/handler -run '^(TestConsignmentRetailCandidates|TestConsignmentRetailProjection)'
ok   justixauto/internal/modules/commerce/service  0.475s
ok   justixauto/internal/modules/commerce/handler  (cached)
```

`TestConsignmentRetailCandidates` covers current dealer candidates, all excluded
lifecycle/association states, independent page reads, supplier isolation,
`retail.read` without commerce permission, and no authority mutation.
`TestConsignmentRetailProjection` covers the actual route, redacted one-field
shape, ignored client dealer query, `retail.read`-only authorization, anonymous,
missing-company and stale-context denial. These pure units do not prove database
lock races, installed migration behavior, inventory hydration or runtime release
readiness.

## Final source SHA-256

```text
5bbb610dce3a34a620ce43c4d80baf07b625fc12b5d8f59098fd0e3286a3aac4  internal/modules/commerce/service/consignment.go
33d420eda40974b7507ff432a78a273181eea5372a28340fac62f70058425fc6  internal/modules/commerce/service/consignment_test.go
3578d1aaec4495f1433494aa6ae44a7678da58bedcace38588532e33ef74a532  internal/modules/commerce/service/ports.go
8be56a2bd764560faaea1ebe3b934dd577792b876240b448aeb0f60c245d00f3  internal/modules/commerce/repository/consignment.go
b2b86300c6bbe66b559f8cd766d878fb6be167f878b3b6f3d70005fc0331bba8  internal/modules/commerce/handler/consignment.go
22166c766b9e69183907d71b3528d2485a64cc3e487a265cf809cf9853e9294a  internal/modules/commerce/handler/handler.go
bcf51862d44d0d315a2786ff1a62c87359ed90167ca4d8a9364617fe9daf2b8d  internal/modules/commerce/handler/dto.go
2596bb2461f671953f7f6966834439fb2bf6fc84c22bf92963201d383ffee8d5  internal/modules/commerce/handler/consignment_test.go
31241e439f1d96ae56b8582a4fedaa885d9cd5b80c59440dd8991d55161c094e  internal/modules/commerce/module.go
```
