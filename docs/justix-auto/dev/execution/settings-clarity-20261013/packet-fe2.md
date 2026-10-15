# FE2 — Reservation form clarity

Task: settings-clarity-20261013/FE2. Hub/acceptance: /root.
Base: b38fc75568d1691b70a02c16f1cb1af5c26cd5e8.
Branch: fix/settings-clarity-20261013.
Contract: ../../settings-clarity-20261013.md, “Added user scope: reservation form”; inherited R3 semantics.
Dependencies: FE1 releases sole source-writer lock; primary accepts this packet.
Rules: root/web AGENTS.md, agent-workflow.md, final-only validation.
Inputs: reservation-source-hashes.json. Context ceiling: 12,000 tokens.

Exclusive source ownership:
- web/apps/portal/src/features/realization/reservation-workspace.tsx
- web/apps/portal/src/features/realization/reservation-workspace.css (new)

One behavior: make existing reservation/supply workflows readable without changing financial or API behavior.

Source map: workspace/list 31–65; SupplyForm 67–113; TermsFields 131–140; RequestForm 142–192; detail 339–446. Detail returns a separate Modal; apply .reservation-workspace inside both Modal bodies. Shared Autocomplete accepts string-valued onChange and strict options; read-only interface at web/packages/kit/src/autocomplete.tsx.

Acceptance coverage (local IDs):
- RC1: Clear request/supply headings and actions; hide redundant request action in request mode. Consistent list/detail/supply presentation and preserved close/back navigation.
- RC2: Group supplier, physical VIN/future-lot selection and optional terms. Replace primary selectors with shared strict Autocomplete: supplier, search model, VIN, future lot, supply model/specification/colors. Preserve committed IDs, “all models” clearing, dependent resets, pagination, deduplication, metadata and locked supply fields; search text never authorizes submission.
- RC3: Label all controls; match shared control sizing; responsive grids, spacing, visible keyboard focus and descriptive empty supplier/stock/lot guidance. Distinguish loading/error/empty and keep retry. Retain actual datetime-local inputs.
- RC4: Show effective selected quantity beside submit, including the current selected VIN as existing payload logic does. Preserve VIN/future-lot exclusivity and add/remove behavior. Disable invalid/busy/duplicate actions using existing validity, permissions and availability; never relax submit guards or make stale selections eligible.
- RC5: Retain request bodies, toMinor/BigInt arithmetic, quantity validation, optional-term semantics, explicit confirmation, revision/digest/allowedActions, sending refs, reconcile locks, session/access guards and refresh behavior. No new financial rule or API change.
- RC6: Import sibling CSS; every new selector under .reservation-workspace, targeting owned classes to avoid restyling nested payment/shared controls. No global/shared-file edits, dependencies, overflow or permanent fixtures.

Final validation, primary only after FE1+FE2 finish, recorded final SHA: desktop/mobile actual Vite with isolated data; RC1–RC6, empty supplier/loading/error, keyboard selection, stock/future quantities and resets, optional terms and captured request shapes, read-only/busy/stale states, supply locks and detail controls. Include settings navigation regression and CSS isolation in that same pass. No tests written/run before completion; no automatic unit/build/lint/typecheck/review/QA chain.

Locks/stops: sole checkout writer and primary browser; version_control owns Git. Run repository-root guard before editing. Preserve unrelated dirty backend test and unread test_creds.txt. No runtime actions, real mutations, push or canonical edits. Stop on changed hashes, ownership conflict or undecided rules; report RESLICE_REQUIRED if scope must expand.

