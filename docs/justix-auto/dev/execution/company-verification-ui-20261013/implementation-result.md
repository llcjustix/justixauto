# FE1 implementation result

Status: DONE

Changed behavior:
- Realization settings now show a prominent «Верификация компании» panel before company data, with state-specific guidance, return reason, explicit session-then-company refresh that throws on refetch failure, permission guidance, and protected submit/resubmit handling.
- Seller Admin lists now have an independent verification filter and column. Seller details have a separate verification review panel gated by `platform.companies.access`, including submitted-only verify/return actions, the existing business-type assignment action, and loading/error retry guidance.
- Admin company details and edit forms show tax ID and bank requisites; unrelated edits preserve the existing legal name, registration, tax ID, and bank requisites.

Output SHA-256:
- `web/apps/portal/src/features/realization/pages/settings.tsx`: `26906cda14249f95408e516975d7aa9abb59df6936ab2556a8a4b6bc28ddce96`
- `web/apps/portal/src/features/admin/pages.tsx`: `a41a1eff82c5b644b2eeeaaef5d948cb276a1a0c61d61ee3d17409ef2d686914`

No tests, builds, lint, typecheck, browser, or runtime checks were run, per packet; primary owns the isolated browser pass. Excluded: backend policy/routes, live data, shared components, and all non-owned files.
