# VIN-EMPTY-01 — local allocation validation

- Task: `docs/justix-auto/dev/vin-allocation-20260930.md`; hub: application `/root`.
- Branch: `fix/empty-vin-allocation`; base: `7210c19e73c053c4938c34f79220b7a8ca0a0339`.
- Dependencies: approved bounded correction and source-identical branch handoff received.
- Context budget: 8k tokens; read targeted symbols only; stop and request reslicing if insufficient.
- Rules: root `AGENTS.md`, `web/AGENTS.md`, `docs/justix-auto/dev/agent-workflow.md`; latest unit-only policy overrides older verification lists.
- Product references: `business-logic.md` §5.5 (VIN availability/uniqueness) and §5.7 (receipt batches); unchanged. OD-02/04/06 are outside scope.
- Locks: one source writer in main checkout; no branch switching, browser or live fixture use.
- Before source edits: `bash tools/check-git.sh`; root must be this project. No worker Git writes.

Owned paths (exact):
- `web/apps/realization/src/pages/trade.tsx`
- `web/apps/realization/src/order-allocation.test.tsx` (new)
- `docs/justix-auto/dev/execution/vin-allocation-20260930/result.md`

Implement only `OrderDialogFooter`'s allocation action and its focused regression coverage:
- R1: add intro: «Назначаются свободные автомобили с VIN со склада. Если список пуст, откройте «Склады», примите автомобили с VIN или введите VIN в ранее принятой партии.»
- R2: before POST, reject `!Array.isArray(v.vehicleIds) || v.vehicleIds.length === 0`; throw existing exported `ApiError` with a Russian `vehicleIds` field error, e.g. «Выберите хотя бы один автомобиль».
- R3: preserve selected `{ items: [{ orderLineId, vehicleId }] }` mapping, endpoint, `If-Match`, refresh, free-vehicle filtering and all unrelated actions.
- Do not change shared kit/forms, backend, inventory data, contracts or canonical records; ordering must not fabricate stock.

Acceptance coverage: one unit file, using exported `OrderDialog`, existing providers and mocked HTTP (see `purchases.test.tsx`):
- U1 → R1/R2: no vehicles; open allocation, observe exact guidance, select a valid order line, submit; Russian field error, no allocation POST.
- U2 → R2: eligible vehicles available but none selected; valid line selected, submit; field error, no allocation POST.
- U3 → R3: select valid line and vehicle; assert unchanged allocation URL/body and revision header; normal success/refresh behavior remains.
- Run once per relevant change with installed Node on PATH: `npm --prefix web run test:unit -- --project @justixauto/realization apps/realization/src/order-allocation.test.tsx`.
- Record command, result and limits in `result.md`; no lint/typecheck/build/React Doctor/browser/integration checks or automatic independent review/QA loops.
- Independent QA packets: 0 under latest user policy; final-SHA integration execution excluded. Shared interfaces remain unchanged; units do not establish release readiness.

Stop conditions: source drift touching this action; need for other owned paths/shared-form changes; unresolved stock/product policy; inability to test locally without wider changes. Report to hub without expanding scope.

SHA-256 source pins (before implementation):
- Task: `6f3e52d658d5210741da2ff62442e7494f3cc211fbabf7c38d40240554b37f27`
- `trade.tsx`: `cad4ba70ab5fad9dcf92b80f6ff3610502d98838bc120440eb5364e3037ab763`
- `web/packages/kit/src/http.ts` (ApiError contract): `c63acb2cf643f360c403482b792c42953de1c2840f68314aef18d110e6483b67`
- `web/packages/kit/src/ui.tsx` (intro/field errors): `f8d9d0462162e5bc1b7d11787bcb96073d792fb392428ffa26a4ef8989d0c773`
- `purchases.test.tsx`: `63d56263d805f78f42bab63ba1e6cc7c48840284ae584a483c8676092a9ed56f`
- `business-logic.md`: `361c10c25a5bf60bf8306317af28c9b2f7c0c1e9c7a28849281b08558262a267`
