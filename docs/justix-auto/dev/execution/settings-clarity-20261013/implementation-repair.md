# Final-browser repair handoff

Status: DONE

`bash tools/check-git.sh` passed before editing on `fix/settings-clarity-20261013` at `b38fc75568d1691b70a02c16f1cb1af5c26cd5e8`. The repair manifest hashes matched the four input paths before changes.

Final owned source hashes:

- `web/apps/portal/src/features/realization/design.css` — `081c6a62c237f2ccc2e3ad0efd32b7df0b6f66b360bf7bf60f5a63392f4c77a9`
- `web/apps/portal/src/features/realization/pages/settings.css` — `fc9c7492a9bd8ed476590c86ba0bdb670cc1549e6e4708306656a7a738dc61cf`
- `web/apps/portal/src/features/realization/reservation-workspace.tsx` — `2162729918078c7f6478e6680fd351c12528f1d254b004bcf42cb8b6d988ca84`
- `web/apps/portal/src/features/realization/reservation-workspace.css` — `3f010a861da93ccbc7272862add1bc4ca8e0459bd6a904cb4b3d4c5baf908458`

Repair behavior: the realization dealer shell restores the compact sidebar at the shared narrow breakpoint and allows narrow topbar context controls to wrap; settings actions/status labels wrap locally. Reservation errors use supported `danger` notices, supplier/VIN/future-lot empties are specific, the add-supply action is hidden while in supply mode, and request submission is disabled while the same supplier/selection/loading/error conditions that block `submit()` are unmet. Raw owned controls retain scoped sizing while shared Autocomplete inputs are no longer targeted.

Preserved: the browser-passing two-VIN empty-terms request and future-lot quantity-3/2500050-USD request shape, all API/financial/session/shell behavior outside the CSS layout fix, and unrelated dirty files. No tests, browser, build, lint, typecheck, Git, or runtime actions were run. Source-writer lease released for primary's affected-scenario rerun.
