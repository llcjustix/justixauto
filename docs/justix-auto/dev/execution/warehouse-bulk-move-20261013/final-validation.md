# Primary final acceptance — warehouse bulk movement

Owner `/root`; 2026-10-14 local. **Finished source and focused HTTP/CUA behavior
accepted. User-local activation authorized below; its execution is recorded
separately by the runtime owner.** No source changes are pending.

## Bound source and HTTP evidence

- Branch `fix/warehouse-bulk-move-20261013`, pre-commit HEAD
  `4b92239779c5f3be1f7ea563123d8a829aee2fbe`.
- The 17 application files are frozen in `final-source.json`, SHA256
  `c687fd6d2a4ae9b110c8aa0b7c6b5d4135703b033226046b710fd3e096df668a`.
- Runtime source fingerprint:
  `702b9507129575c8d6f5cf35ae9f87e08fe5738ed725272959573e2c40f41801`.
- Migration46 up-file:
  `af845b1de029f60e84b05a92afd9c5027d923e4867befbe9070a271c47e0e188`.
- `runtime-result.md` records seven passed HTTP cases, the populated45→46
  rehearsal, 21 rollback variants, identification/correction/provenance,
  bounded row contention and one last-unit move/identify race.
- One initial replay returned409 rather than412. The source-revision preflight
  was repaired; only the affected API and failed/unstarted continuation were
  run again. The successful initial mixed operation was retained. Initial
  failure and repair evidence were not erased.

## Primary CUA evidence

Real portal at `http://localhost:5192`, isolated API8091/PostgreSQL55443 and
synthetic Buyer account. No real user data or login was used for testing.

1. Warehouse footer exposes **Переместить запас**. VIN is a native button;
   Enter opens its vehicle detail and the legacy movement entry remains
   available. Returning to the warehouse and opening bulk movement preserves
   one active dialog.
2. Manual VIN004 selection, select-all for four VINs, explicit first-three
   selection in ascending VIN order, then removal of VIN003 from the preview
   were observed. Final draft retained VIN001 and VIN002.
3. Fractional pending quantity1.5 is rejected with visible validation while
   retaining selected VINs. Two pending batches were then set to1 and2.
4. A controlled disposable destination-capacity update caused a stale submit.
   Both VIN selections and both quantities survived; refreshed free capacity41
   was shown. Explicit confirmation of refreshed stock enabled submission of
   the same draft, without silently substituting vehicles.
5. At390×844, the dialog measured341px wide with scrollWidth341 and viewport390;
   one modal and no horizontal overflow. The temporary viewport override was
   reset after this check. Desktop form layout was also inspected.
6. Enter on **Переместить 5** submitted the mixed move successfully. Source
   occupancy changed13→8; destination occupancy became5. `final-ui-http.json`
   records VIN001/VIN002 and three pending units moved, exactly one source/
   destination revision bump, unchanged global receipt fields and conservation.
   This is state evidence, not a captured network-request-count assertion.
7. At the destination, VIN entry for its local one-unit pending balance saved
   `XTAAA11111B799001`. The success message reported local pending0, the new VIN
   appeared at the destination, and total occupancy remained5. The second
   pending batch retained local2 while the source retained its own balances.
8. The original direct order still showed received6, identified2, pending4
   after all its original-location pending stock had moved away. Its **Ввести
   VIN** action opened the same original batch; it explicitly required a new
   location rather than claiming completion. Selecting **E2E destination**
   rendered the VIN editor with local capacity4. No extra receipt was created.

Screenshots (synthetic data only):

- `/private/tmp/justixauto-warehouse-bulk-form.png` — selected desktop draft.
- `/private/tmp/justixauto-warehouse-bulk-mobile.png` — mobile success view.
- `/private/tmp/justixauto-warehouse-bulk-result.png` — destination stock and
  successful later VIN identification.

The owned CUA tab was closed after inspection; user tabs were not touched.
One React development console warning about `flushSync` inside a lifecycle was
observed. It did not interrupt the exercised flows; its origin is not attributed
to this change and the console is not claimed clean.

## Scope and limits

No automatic unit/lint/typecheck/React Doctor or broad build/review loop was run.
Runtime API compilation was necessary to execute the accepted final behavior.
No full1000/10000 boundary fixture, populated reservation/consignment variant,
transport-drop simulation, double-click timing case, or browser no-access/
no-destination variant was executed. Permissions, stale revisions, invalid
inputs, tenant boundaries and rollback were exercised through HTTP. This
acceptance covers the requested local feature, not production/release readiness.

## Authorized user-local activation

The standing user instruction to run locally and this accepted task authorize
the runtime owner to execute the reviewed `local-activation.py` (SHA256
`41c00a5c9aa95119803db844705bac90bc66971267b3bb83c946a9bf4b470d2e`)
against the recorded private state at
`/private/tmp/justixauto-warehouse-e2e-l6dz41m1/state.json`, **before HEAD changes**.
Reverify exact identities; drain old API37081 while Air31138 is paused, confirm
no other application DB clients, create and validate a private pg_dump, then
resume Air once for its existing build→migrate→restart path. Require clean46,
a single new API8090 listener, readiness204 and equality to the HTTP-verified
API binary. Preserve Vite26877/5191 and the named PostgreSQL55433 container,
original data, private backup and evidence. No reset, down/force migration,
live test seeding, automatic restore, production action or push is authorized.

Runtime may close only the exact disposable fixture resources after activation;
the CUA lease has been released. Canonical completion and local commit wait for
the activation result. Air must not be left silently paused.
