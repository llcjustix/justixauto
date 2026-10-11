# P19 stock discovery — technical completion proposal

Status: DRAFT pending application hub `/root` acceptance; no dispatch authority.
Canonical task `product-workflow-improvements-20261011`; original P19a U2/R3,
with R1 company identity and R2 preserved read access. This is one technical
completion packet, not a new canonical backlog item. RESLICE_REQUIRED: no.
Branch `feature/company-commerce-workflows-20261011`; supplied milestone
`ab088d97054fbabac65caae1f3f8bfe7f71fbe74`. Dirty predecessor source is not
claimed to equal that commit. Hub pins actual dispatch SHA and source hashes.
Owned planning artifacts are this file and `packet-19-stock-discovery.md` only.

## Incremental source findings

- P19a requires the dealer to choose supplier stock VINs. The existing
  `useEligibleVehicles` reads `/inventory/vehicle-units` for the active company;
  `Vehicle.List` forcibly sets `CompanyID` from the real principal. Changing
  the query string cannot authorize partner inventory.
- Inventory `VehicleRepository.List/listQuery` already supports a bounded,
  stable VIN/ID ordered query: `Eligible=true` means owner company matches,
  VIN is nonempty and no inventory reservation with status `held` exists;
  `Placement=warehouse` requires a placement. These are inventory-owned tables.
  Reuse this query; do not introduce a foreign-table join or modify visibility.
- Commerce `Directory.Company` projects stored company kind, active access,
  fixed user type and verification. `Partnerships.ActiveBetween` checks the
  active relationship in either direction. P16's read uses `commerce.read` and
  stored active seller identity; verification gates new operations separately.
- P17a already accepts supplier company + vehicle IDs, copies immutable source
  identity, and does not hold stock at request time. Confirmation/acceptance
  reaches inventory's locked hold path. P18b2 preserves this source identity
  through conversion; discovery must neither bypass nor mutate those paths.
- Inventory `Stock.Vehicle` is a trusted point lookup, not a discovery API;
  its `VehicleInfo` includes warehouse/owner fields inappropriate to expose
  wholesale. A new narrow projection avoids coupling HTTP to that struct.
- `app/app.go`, all `module.go` files and `cmd/api/main.go` were deliberately
  not read: P18c owns evolving composition/runtime source. Final handoff is
  mandatory before implementation touches composition. No tests/checks ran.

## Proposed seam

One coherent behavior: an authenticated dealer with `commerce.read` selects an
active partner distributor and reads that distributor's available physical VIN
stock through commerce. Proposed GET `/api/v1/commerce/reservation-stock` takes
required `supplierCompanyId`, optional `modelId`, `limit`, `offset`; returns only
the allowlisted identity DTO in the normal list envelope. Source IDs are usable
by existing reservation POST. There is no stock detail endpoint, public inventory
scope switch, hold, custody query, transaction write, schema or dependency change.

Commerce owns authorization and the HTTP projection. Inventory owns a typed
`Stock.ReservableStock` read using its existing repository query and a fresh small
projection. `internal/app` explicitly maps root-exported inventory types to a
dedicated commerce `ReservationStockDiscovery` port. Keep this port separate from
`Stock` and `ReservationStock` to avoid expanding all mutation-port consumers.
The packet fixes the exact DTO, errors, paging, paths and unit cases.

Read semantics follow P16/R2: `commerce.read`, stored active seller + dealer
caller, stored active seller + distributor supplier and an active partnership.
No new verification or `commerce.trade` prerequisite is added to a read. A
read-only or not-yet-verified dealer may discover an eligible partner's stock;
existing reservation commands retain their permission/verification gates. Do
not infer authorization from principal labels, UI roles or demo fixtures.

Warehouse placement means the same predicate as the established eligible query,
not a newly invented custody rule: owner must be the supplier and some warehouse
placement exists. A vehicle merely stored for another owner is excluded. The
projection does not disclose the warehouse or custodian. If hub requires a
different placement/custody rule, stop for a product decision rather than change
this packet silently.

## Coverage, dependencies and limits

Acceptance IDs SD1–SD8 in the packet are local coverage labels for U2/R3, not
canonical requirements. Inventory projection, commerce authorization, HTTP DTO,
app adapter and downstream P19a consumption all map to explicit unit evidence.
One implementation packet; zero automatic QA/review jobs under the latest
unit-only policy. Independent QA-sized checks and final-SHA shared-interface
mapping are specified but deferred to an explicit request; no integration run
or repeat of already passing units is authorized.

Dependencies: final P16, P17a completion and P18b2 results already read; final
P18c source-writer release and composition handoff still required. P19a stays
blocked for U2 until this packet's final implemented API result exists. The
hub accepts this proposal and packet before dispatch. No canonical board or
contract edit is made by the slicer.

Risks/gaps: live SQL/isolation behavior is not established by unit fakes; read
availability can race later holds and must never be a promise. Cross-module
constructor signatures and root Stock accessor need final P18c handoff and
narrow dispatch-time inspection. Unlisted constructor consumers require a hub
scope extension, never a variadic/optional port fallback. No undecided product
rule is resolved here; alternate warehouse exposure, broader stock search or
new permission/verification policy blocks only the affected extension.

## Source SHA-256 snapshot

Planning snapshot of targeted source on 2026-10-11, not a clean-commit claim.
Paths below are repository-relative. Evolving composition/runtime files are
intentionally absent; hub adds their final dispatch fingerprints after P18c.
The hashes pin whole files even where only named relevant slices were read.

```text
a222419ecca4935c906454884eb62d6cacace6a5b80395c90a1cee870f8b8524  AGENTS.md
b22213a76ca11a4ce4aeb04323260d1d577f28ef1230d48c15367d074e67650b  internal/AGENTS.md
4f264fb5217a9613b00df455b0661cb9b955e90793a29b17c682cbcd0010830b  docs/justix-auto/dev/agent-workflow.md
5df5a66f4ad3526b36fc5f35c011eff7311eccb0e0b6b9a0e185636122542356  docs/justix-auto/dev/product-workflow-improvements-20261011.md
a2cd3b101a5d720e26aa7a96524b0252aa305b22db75895a52513b6b8777599e  docs/justix-auto/dev/product-workflow-interview-20261011.md
5bd0439c6aca0b5a824c340c4c15692734c70296b0125d82c63a739a21ba65ba  docs/justix-auto/contracts/http-domain.md
657e4718bea9c8302b60864976a948a6fec80c6c73dab6ee986af156c298bece  internal/modules/inventory/service/vehicle.go
87916c574e59385b981cacf6f3d4cc5b7f10bc6a2370c721e3387957fd2f4e31  internal/modules/inventory/service/stock.go
07f978d4355ae20d8217bcb6c1375f939ff9e5f811059664fa649f9cfd452888  internal/modules/inventory/service/ports.go
72f23f26860ccba9d9d62f5ccc43ddf68bb2c802cd247e810fd294a4d7be4972  internal/modules/inventory/model/vehicle.go
5d91771af595f3f2add6218a964c757f275ebe7a89653315fce504a2ba589c19  internal/modules/inventory/repository/vehicle.go
bc21134404313018673c45ca283964bc61ebcc7d237f7ce2d42503a51cf25661  internal/modules/inventory/service/vehicle_filter_test.go
6b410685557f81b75ae1d4b889f44a62dd3f72b17495a903977798657302b313  internal/modules/inventory/repository/vehicle_filter_test.go
55ef54785ee0777c154c2d32464ad7acc8f1ddf2ea69953af971d6b15e64c66b  internal/modules/commerce/service/deps.go
7158dad8f265cb83fe8708a1c74920f29bfa12b75e813727ed4beb4ec08823c6  internal/modules/commerce/service/ports.go
90cb49abb8eff36fcf85fb74ad0faf94f82bba891c2180fb059fd8630b8ea585  internal/modules/commerce/service/supply.go
1bd17920c73aa47090c7ebc67d23b9ee13acf6fa0e88b29f168d9eb26ad56902  internal/modules/commerce/service/partnership.go
2ba2173892c58d3d4ba72b9f0231b79d47dac1bc80f36e5a912579234727610f  internal/modules/commerce/repository/partnership.go
a54bf75d5525f421e6c8802f96867327697375f3418ebef8f7369054b1b53ba8  internal/modules/commerce/service/reservation.go
4fd3d4bba8383bf723a9f493f87d230f338cf1a117d443b29392a24bb7fb4b93  internal/modules/commerce/handler/handler.go
84b4bc569d637b83b7e0daa040b3a3acbe97013a9d2597084fdab0f014433a7a  internal/modules/commerce/handler/supply.go
099c48b63042c401c34207e8130fe72f9e027e93973de73c045e2f4c48ac99b6  internal/pkg/auth/principal.go
0a0383f009851bd39f30191d9a7342272cc3366183c6f276e5e0028e13abcf2a  internal/pkg/database/helpers.go
855c1efe438352be1b6a70e7d2ad31591b515ea06eb29ccdbbfcb0f05db5d174  internal/pkg/httpx/server.go
de8758b9d7ce709447576118ebf0d0eec96c86f5522870416007ab0c243df50c  internal/app/workflow_ports_test.go
f14f059c17a5982d67bf6fbbea6c8ee7270cdb015f245ec93348887aac929d81  web/apps/realization/src/data.ts
f8a0d756be25ba54b71b2b22294676f0e18adc3747f2bbc841d825e87ab88c69  docs/justix-auto/dev/execution/product-workflow-improvements-20261011/plan.md
3b41a095e69227b5984e085805f0c22bef755afc9a09388b30b6e65f9332036a  docs/justix-auto/dev/execution/product-workflow-improvements-20261011/p19-split-proposal.md
812b4d078ff571e42b22935b130db7667162db01dd8685397a5094dbc4b5854c  docs/justix-auto/dev/execution/product-workflow-improvements-20261011/packet-19a.md
8b1fde0cda0d7fc1100675688a6cd0eb6e278260e39c037ddae43aab4e7d38b1  docs/justix-auto/dev/execution/product-workflow-improvements-20261011/result-16.md
3c6db0ff93d026a9c27d9dd6e81241e49be640c7e7768ba482289065408984e5  docs/justix-auto/dev/execution/product-workflow-improvements-20261011/result-17a-completion.md
5a186de1f9e90ef07693462974fd4aa1307c3b98a45eca2564fa12a6f72c287d  docs/justix-auto/dev/execution/product-workflow-improvements-20261011/result-18b2.md
```
