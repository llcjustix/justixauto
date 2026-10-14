# P2 switcher result

Status: DONE — source lease released 2026-10-14.

Changed `web/packages/kit/src/portal.tsx`: `displayedDestinations` removes the
Realization ↔ Accounting reciprocal links only for the rendered sidebar list.
The list now controls both sidebar existence and mapped links, while company
choices and all route/admission/selection logic retain their original inputs.

SHA-256 (`web/packages/kit/src/portal.tsx`):
`c1b0c1a5fc862d0d18637d77afc22f520f63428c327f5909907861495ce7c1a3`.

Excluded: P1 billing/CSS/tests, protected identity test, `test_creds.txt`,
canonical docs, Git operations, and all checks (per packet; root owns the final
affected-unit pass).
