# P10 — Same-company warehouse relocation UI

Requirements: R4. Dependencies: P09. Read plan.md common execution rules and referenced source/contract slices. Application hub /root; base/branch as plan. Exclusive resource: source-writer; no live fixture/browser. Context ceiling: 18k tokens (stop/split if exceeded).

## Exact owned paths

- `web/apps/realization/src/pages/inventory.tsx`
- `web/apps/realization/src/vehicle-move.tsx`
- `web/apps/realization/src/vehicle-move.test.tsx`
- `web/packages/kit/src/messages.ts`

## Behavior, interfaces and data flow

Expose action from VehicleDialog using existing inventory Move endpoint (read handler route). Send {fromWarehouseId,toWarehouseId,occurredAt,evidenceBindingIds:[]} and current context. Source is current placement; target choices only same-company warehouses excluding source. Use actual inventory move permission. Render capacity_exceeded/placement_changed/not_in_warehouse and permission errors; refresh vehicle, source/target warehouse and registry queries. Do not create shipment/approval ceremony or change owner. Do not send unsupported evidence.

## Acceptance and affected units

Target selection and exact payload; no action when no placement/permission; error retains modal and selections; success refreshes both warehouses and current placement; source mismatch forces refresh.

Run each affected command once after the relevant change:

- From `/Users/bakhromachilov/startups/justixauto/web`: `npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/vehicle-move.test.tsx` (Realization only).

Keep these selections separate; repairing a failure reruns only affected suites, not a previously green selection. Require the output to name the intended suites; an omitted suite is not evidence. No other development verification or runtime actions.

Stop: Stop on any new product rule, ownership expansion or context ceiling; report exact missing decision/path. No unrelated refactoring.

