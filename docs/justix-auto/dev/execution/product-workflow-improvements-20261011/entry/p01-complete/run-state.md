# Requirements interview state

- Updated: 2026-10-11.
- Project: `justix-auto`.
- Current phase: authorized sequential implementation of the user's 13 workflow changes.
- Durable record: [interview and decisions](../dev/product-workflow-interview-20261011.md).
- Confirmed this round: consignment settlement terms are agreed per order;
  future vehicle reservations may start without VIN and allocate VINs later.
- User accepted the proposed reservation flow with «да давай»: negotiated
  terms, price fixed at confirmation, buyer acceptance of changed terms,
  free reservation by default with optional agreed advance, VIN/quantity holds,
  accountant-confirmed advance, expiry of free reservations and agreed
  handling of pending/accepted money, supplier-approved extensions and
  continuation as purchase or consignment.
- Latest user direction: «go start» after proposed contract approach. Implement
  auto-filled templates plus uploaded documents, paper signing/scans, immutable
  signed versions and separate amendments. Both retail/B2B coverage is a stated
  coordinator assumption, not an explicit answer choosing between scopes.
- User explicitly resolved current questions: legacy companies manually classified/
  verified in Admin; new commercial actions blocked, history and servicing preserved.
  Reservation advances refunded in full with accountant confirmation; partly paid
  purchase handover requires explicit order agreement.
- User explicitly confirmed automatic expiry of an unpaid reservation after
  the required-advance deadline if no evidence was submitted. Pending/accepted
  evidence prevents automatic release. No known reservation interview gate remains.
- Canonical implementation: `../dev/product-workflow-improvements-20261011.md`.
- Next: accept first slicer packet, dispatch one implementation worker at a time,
  execute affected unit tests only and track coverage of all 13 requested changes.
- Existing architecture, stack and product documents reused; no regeneration
  of the original spec, mocks or historical backlog.
- Agents: task_slicer preparing packets; version_control created feature branch;
  source writer not yet dispatched. Branch `feature/company-commerce-workflows-20261011`,
  base `4fa683ba4708bb71031534260ec5cb8742ce9977`.
- Verification: read-only source inspection only; no tests/build/browser,
  review/QA loops, runtime or deployment performed. Only Git branch preparation.
- Interview and state were amended after user acceptance; pre-amendment
  SHA-256 and recoverable copies are in
  `../dev/execution/product-workflow-interview-20261011/entry/reservation-approval/manifest.md`.
- Original business logic, spec, mocks and historical backlog remain unchanged.
- Before implementation-state promotion, exact snapshots/SHA-256 captured in
  `../dev/execution/product-workflow-improvements-20261011/entry/implementation-start/manifest.md`.
