# Final browser validation — settings and reservations

Result: PASS for the bounded UI change. Actor: /root using CUA, after both
implementation packets completed. Initial observations and diagnosed responsive
repairs are in browser-initial.md and the two repair handoffs. Only affected
scenarios were repeated. No unit additions/runs, lint, typecheck, build,
React Doctor or automatic review/QA chain.

Source identity: validated-source-hashes.json, based on b38fc75568d1691b70a02c16f1cb1af5c26cd5e8
on fix/settings-clarity-20261013. These exact source hashes are to be verified
by version_control in the final local commit; no later source edit is authorized.

Environment: actual portal served by existing Vite :5191; fixture-server.cjs
proxies frontend assets only and handles all /api calls in memory. Company admin,
reader and distributor fixtures used :5192–5194. Local API/database untouched.

Verified observations:

- Desktop settings groups legal/bank/tax details and contacts, displays explicit
  missing values, fixed company type, cabinet access and a concise verified
  result. All four settings sections display their existing information/actions.
- Final settings at 390px: document width 390px, compact sidebar 76px. Nested
  verification heading/actions fit. Long unbroken address also gives width390.
  Desktop 1440px layout remains readable. Reader has no company-edit action.
- Returned verification shows its reason and resubmit action. Resubmit changes
  to pending guidance and removes the submit action. Contact edit succeeds with
  tax ID and bank details preserved; fixture checks If-Match.
- Reservation request uses grouped labelled controls and strict Autocomplete.
  Keyboard supplier selection and VIN search/add work; effective selection count
  includes both an added VIN and the current selected VIN.
- Two VIN request captures quantity '1' for each and desiredTerms {}. Future-lot
  quantity '3' and price '25000.50' capture amountMinor '2500050', currency USD.
  Future request was repeated after readiness-guard correction and still passes.
- Empty supplier message is specific and submit disabled. Missing selection,
  uncommitted supplier search and zero quantity disable submission. Changing
  supplier clears vehicles and count. Advance without a date displays validation.
- At 390px, request dialog width358, scrollWidth356, document width390. Desktop
  request and distributor supply-edit form have consistently sized controls.
  Supply model/version/colors/date prefill; redundant add-supply action is absent.

Fixture journal: fixture-observed.json, five expected commands, zero assertion
failures (three reservation requests, verification resubmit, company edit).
Screenshots: /private/tmp/justixauto-settings-redesign.jpg,
/private/tmp/justixauto-reservation-redesign.jpg,
/private/tmp/justixauto-settings-mobile.jpg.

Limits: UI/fixture evidence, not live backend authorization, concurrency, payment,
fulfilment or release readiness. Native datetime input rendering/prefill checked;
tool-driven date filling did not commit, so no dated-term submission claim.
Existing supply lock, server failure/reconciliation and financial detail workflows
were preserved, not exhaustively rerun. Browser tabs closed and viewport overrides
reset; temporary fixture process stopped separately. Existing local app preserved.
