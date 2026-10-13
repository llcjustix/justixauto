# Manual receipt UI implementation — 2026-10-13

Status: DONE

Implemented the approved manual-receipt presentation in the three owned source
paths. The receipt form now uses scoped, responsive grouped sections for vehicle
specification, receipt method, and vehicle entry; it exposes native radio choices
for both modes and styled native date/quantity controls. The existing model,
exact specification, color, payload, date conversion, revision, guard, refresh,
and submit behavior is retained. The vehicle-colour summary is withheld until
both colours exist.

`VinEditor` has an opt-in `showEmptyRequiredError` presentation prop, defaulting
to the prior visible-required-error behavior for all other consumers. Manual
receipt starts with neutral VIN guidance and enables the existing required error
after a VIN interaction or submit attempt; validation itself is unchanged.

The final browser review found unused horizontal modal space from the initial
`52rem` cap. The scoped wrapper now uses the available modal width
(`width: 100%; min-width: 0`); responsive grid rules are unchanged.

Final source SHA-256:

- `web/apps/portal/src/features/realization/pages/inventory.tsx`: `b1955a0aa49f49ac0a23075bc71d372ac111c5d63ed5e4229c9896b9043207c4`
- `web/apps/portal/src/features/realization/pages/manual-receipt.css`: `1388087d8bdbbe88a26496e10bac41b0752b41e74ea7a704fd0d86a20704ef5e`
- `web/apps/portal/src/features/realization/vin-editor.tsx`: `c57eaabcbabbd515badd8621164b376e4ef3ceb6e01335db2ff52cb0c33a4a71`

Excluded: no tests, browser/runtime actions, Git actions, backend changes,
canonical-document changes, or unrelated files. Per packet, the primary owns
the focused final browser pass. Source-writer lease released.
