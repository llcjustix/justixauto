# P17a implementation completion

result-17a.md is PARTIAL, not product-scope expansion. Original writer stopped
after two compile failures and released by final result; no service acceptance
or app seam tests exist yet. A fresh bounded worker completes original packet
17a and its accepted request-terms clarification. Explicit risk escalation to
gpt-6-sol/high under agent-workflow: compilation failures plus incomplete
reservation service/HTTP acceptance. Ceiling 20k, split honestly if needed.

Known handoff gaps:

- unavailableReservationStock.ReleaseReservation has a mismatched signature.
  Resolve the composition seam with typed dependencies; do not leave an unchecked
  runtime assertion and optional silent fallback for required production stock.
- Partial implementation exposes mutations only. Original packet also requires
  authorized list/detail and complete requested/offered/current terms, source
  line quantities/specification/colors, party IDs, revisions and next actions.
  Consumers must receive structured exact JSON, not raw byte/base64 snapshots.
- Supplier confirmation currently cannot propose quantity changes and digest
  covers economics only. Exact agreement must include accepted quantities/source
  lines, preserving original requested facts; changed quantities require buyer
  acceptance. Activation uses accepted counts and creates holds only once.
- Complete validation/rechecks for positive exact price/currency/advance and
  explicit future expiry/deadline, stock/lot source identity and availability,
  actual parties/types/permission, stale version/digest and duplicate activation.
  No default commercial terms or changed accepted price in place.
- Add the original actual transition/rollback units, inventory holder cases and
  pure TestWorkflowPort17. Batch future claim creation with context checks and
  bounded inserts; no arbitrary business quantity cap.

Own the original packet-17a paths. Add only
internal/modules/commerce/handler/reservation_test.go for focused DTO/HTTP units
covering the newly completed read/snapshot boundary. Report other exact fixture
paths if constructor/interface compilation requires them; no unowned edits.
P17b still owns VIN allocation/reassignment/history, P18 owns money/lifecycle.

Run only affected service/inventory unit selections and the required narrow app
selector, once after relevant changes. Inventory already ran green on partial
source; rerun only if adding/changing the assigned holder behavior/tests. New
handler tests run their narrow TestReservationHTTP family. No DB-backed app
OpenAPI test, review/QA, build/lint/typecheck/browser/runtime/Git actions.
Final report result-17a-completion.md must record actual APIs/claim linkage,
lock ordering, test evidence/exclusions and final source release.
