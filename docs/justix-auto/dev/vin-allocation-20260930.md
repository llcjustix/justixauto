# Empty VIN allocation — 2026-09-30

Status: implemented; affected unit tests passed. User reported an Assign VIN screenshot showing
no vehicle options and the backend error `items: list 1-1000 vehicles`.

Read-only investigation found ChinaCars has no warehouses or received vehicles.
Its accepted 20-car order is valid; ordering does not create physical VIN stock.
The frontend previously submitted an empty allocation list; it now shows a local
vehicle selection error before sending a request and explains how to add stock.

Approved correction: add Russian inventory guidance to the allocation form and
reject an empty vehicle selection locally using the existing field-error UI.
Preserve valid allocation payloads, revision handling, refresh, stock filtering,
and business rules in business-logic §5.5 and §5.7. No backend or data changes.

Branch: `fix/empty-vin-allocation`, source base
`7210c19e73c053c4938c34f79220b7a8ca0a0339`, carrying existing committed work
forward from ancestor dev `0e74dac4258402747e3a5cdc748a671112cf2fd0` without
changing source during checkout. Unrelated user changes are preserved.

Packet: `execution/vin-allocation-20260930/packet.md`.
Result: `execution/vin-allocation-20260930/result.md`.
Source ownership: realization `pages/trade.tsx` and one focused allocation unit
test. One implementation worker; only affected unit tests once per relevant
change. No automatic reviews/QA, React Doctor scan, lint, typecheck, build,
browser/integration checks or live allocation requests. The unit-only user
decision supersedes scoped skill verification commands.

Verification: the focused `order-allocation.test.tsx` run passed once, one file
and three tests. It covers empty inventory, an unselected available vehicle, and
the selected allocation payload with its revision. No live inventory or order
data was changed, and browser/integration behavior remains unverified.
