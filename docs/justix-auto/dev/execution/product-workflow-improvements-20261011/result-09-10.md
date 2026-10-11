# P09/P10 frontend result

Status: DONE. `bash tools/check-git.sh` passed before source edits. No Git writes, runtime, browser, lint, typecheck, build, review, or QA actions occurred.

## P09 / R6 — CRM passport profile

- `web/apps/realization/src/customer-passport.tsx` supplies a dedicated customer modal with every P08 optional field under the profile request: series, number, issuer, issue/expiry/birth dates, personal ID, registered address, and `passportCopyFileIds`.
- Upload uses the existing `FieldSpec` file mechanism with `purpose: 'passport-copy'`; the P08 server marks and verifies it as sensitive before binding. The UI sends the uploaded ID as `passportCopyFileIds`, preserves existing IDs on edit, and renders stored copies only through authenticated `fileUrl(id)` download routes.
- `web/apps/realization/src/pages/crm.tsx` posts to `POST /retail/customers` and patches `PATCH /retail/customers/{id}` using the current customer revision. The CRM table remains limited to display name and phone, with no passport data in search or columns.

## P10 / R4 — warehouse move

- `web/apps/realization/src/vehicle-move.tsx` replaces the old inline action from `VehicleDialog`. It requires `inventory.vehicles.move`, requires a current warehouse placement, excludes the source warehouse from target choices, and posts `POST /inventory/vehicle-units/{id}/warehouse-moves` with `{fromWarehouseId,toWarehouseId,occurredAt,evidenceBindingIds:[]}`.
- Success refreshes vehicle/registry, warehouse list, source stock, and every possible target stock. `placement_changed` refreshes the same current placement data while retaining the modal and its selections; existing mapped capacity, placement, out-of-warehouse, and permission errors remain visible in the form.

## Unit evidence

- From `web/`: `npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/customer-passport.test.tsx src/vehicle-move.test.tsx` — PASS: 2 files, 6 tests.

## Exclusions

No unowned source path changed. Pre-existing dirty edits in `data.ts`, `inventory.tsx`, and `messages.ts` were preserved; this packet changed only its approved additions and replacements.
