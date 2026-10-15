# Packet 01 result — section switcher

Status: DONE

Changed only the packet-owned sources:

- `web/packages/kit/src/portal.tsx` — builds the existing inactive-company choices once; renders the chooser only for more than one permitted destination or at least one inactive-company choice; marks the current section active by section ID.
- `web/packages/kit/src/ui.tsx` — adds shared, sidebar-scoped switcher item rules with natural height, wrapping, active, hover and keyboard-focus states.
- `web/packages/kit/src/portal.test.ts` — covers sole platform Admin and sole company-section chooser absence, multiple permitted sections without inactive company choices, current deep-page section indication, and preserves inactive-company selection coverage.

Verification:

```text
cd web && npm run test:unit -- --project @justixauto/kit packages/kit/src/portal.test.ts
Test Files  1 passed (1)
Tests  13 passed (13)
```

An initial invocation found a test-file parenthesis typo; the one-line syntax repair was followed by the passing invocation above. No browser, visual, lint, typecheck, build, review or QA checks ran.

Final SHA-256:

```text
41cc642f06d82c53bb91e6f40f0a28e1dd54886af5ca17c001cb9daaee1bfec5  web/packages/kit/src/portal.tsx
56502c5e45d19fa2228f4121b11d365a14e52a677fe7575b129efa2129bd370d  web/packages/kit/src/ui.tsx
bd4156b0e544d07f7bb2d3fe1196c75461092e355fa4cf6fb6ef07967be0c8f2  web/packages/kit/src/portal.test.ts
```

Excluded: Shell insertion points, global/feature CSS, permission and selection helpers, routing/session sequencing, persistence, canonical records, credentials, runtime and Git operations. Writer lease released.
