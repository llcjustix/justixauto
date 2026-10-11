# P12a1 continuation result — DONE

Preflight: `bash tools/check-git.sh` reported `GIT CHECK OK:
/Users/bakhromachilov/startups/justixauto`; work stayed on
`feature/company-commerce-workflows-20261011`. Dirty predecessor paths were
preserved. This continuation changed only original P12a1 contract service and
its pure unit fixture/tests; the existing typed repository/module/migration
seam was not changed.

Completed service API for P12a2:

- `NewContract(d Deps, parties ContractParties) *Contract` and
  `NewActualOrderParties(d Deps) *ActualOrderParties` remain the construction
  seam. `(*ActualOrderParties).OrderPartiesFor(ctx, viewerCompanyID, orderID)
  (*model.Order, error)` reads the actual order then rechecks `Order.Party`.
- `Preview(ctx, principal, orderID, DraftInput) (*ContractPreview, error)`
  requires `commerce.read`, captures authoritative order/party/allocation
  snapshots, and returns plain-text `Content`, template ID/version, raw
  `PartySnapshot`/`TermsSnapshot`, digest, and `SourceOrderRevision`. It
  creates no row/grant/event and does not update the order revision.
- `TemplateDetail(ctx, principal, templateID) (*model.ContractTemplate, error)`
  requires `commerce.read` and scopes the template to its recording company.
- `DraftCreate(ctx, principal, orderID, expectedOrder, DraftInput)`;
  `DraftEdit(ctx, principal, contractID, expectedOrder, expectedDraft,
  DraftInput)`; and `Sign(ctx, principal, contractID, expectedOrder,
  expectedDraft, SignInput)` retain order If-Match and draft CAS. Draft edit
  preserves omitted fields, permits explicit clearing only when another valid
  source remains, and grants each newly added external file only to its author.
- `Amend(ctx, principal, draftID, priorSignedVersionID, expectedOrder,
  expectedDraft, SignInput)` consumes that existing private draft into the same
  signed amendment row. It validates that the visible parent is signed and for
  the same actual order, never changes the parent, and accepts no invented
  parent revision.

Literal placeholders are `buyer.{id,name,legalName,registrationNumber,taxId,
bankRequisites,address,phone,email}`, corresponding `supplier.*`, and
`order.{route,deliveryTerms,warrantyTerms,serviceTerms,lines,paymentSchedule}`.
The four scalar order fields substitute their JSON string value; lines and
schedule substitute their raw accepted JSON. Unknown terms remain untouched in
the raw snapshot. The service uses transaction-bound order reads and requisite
lookups, and signs unioned/deduplicated external and signed-scan IDs.

Evidence: after the final template-detail addition, `env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash
tools/go.sh test -vet=off ./internal/modules/commerce/service -run
'^TestContract'` passed once. The preceding lifecycle change had its own
focused passing selector before this final narrow addition. The added cases cover
read-only preview, template revision/plain text, omitted versus explicit-empty
draft edits, new-file author grants, both-party signed history, same-ID
amendment/immutable parent, exact future JSON, stale terms, and injected file,
contract CAS, order CAS, and event failures with deep-copy transaction rollback.

Excluded: P12a2 production adapter/HTTP/DTO work, app/handler tests, migration
execution, database concurrency validation, review/QA, lint/build/vet/browser
or integration checks, and Git operations. The unchanged pure Store wiring
selector was intentionally not rerun under the unit-only continuation rule.
