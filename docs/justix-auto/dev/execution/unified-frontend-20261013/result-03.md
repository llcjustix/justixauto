# P3 portal workspace result

Status: **DONE**. Actor: `/root/unified_portal_workspace`. Branch/base:
`feature/unified-frontend-20261013` / `743bd6c4accadcd50afb59534bb66dabe2f3baf8`.
`bash tools/check-git.sh` passed before edits. P2's accepted `result-02.md`
was read before composition. No Git write, install, network, runtime/server,
browser, lint, typecheck, build, review, QA, or child agent action occurred.
**The P3 source/test writer lease is released by this result.**

## Completed ownership

- All 94 `move` rows in `move-manifest.md` were moved with byte-preserving
  `apply_patch` move operations; the four replaced app entry files and all four
  old app workspace/config sets were removed. A manifest presence check produced
  no missing destination or remaining source row.
- Added the sole `@justixauto/portal` workspace at `web/apps/portal`, including
  base `/`, strict port 5191, portal root, explicit-root Vitest configuration,
  document fallback helper, and a single `mountPortal({ rootId: 'portal-root',
  sections })` entry. The root workspace build target and lockfile now name only
  the portal application identity; locked dependency versions/integrities were
  not changed.
- `sections.tsx` supplies ordered realization, financing, insurance and Admin
  sections. Seller URLs/search remain root-relative; provider and Admin nav
  values are final absolute browser URLs. The seller billing OR permission array
  is preserved. Finance overview actions now navigate to `/finance/*`; Admin
  overview cards/audit navigate to `/admin/*`.
- Removed only the obsolete `mountApp` bootstrap and its imports from
  `web/packages/kit/src/shell.tsx`; the export-star barrel required no change.
  No CSS selector collision required scope changes after the targeted selector
  inspection.

## Document policy API and evidence

`src/document-policy.ts` exports `DocumentRequest`, `isDocumentRequest(request)`
and `isPortalEntryRequest(request)`. It requires GET/HEAD, positive `text/html`
intent and compatible supplied Fetch Metadata; it permits `/index.html` and
section deep links, while excluding API (including bare `/api`), `/healthz`,
`/readyz`, assets/source/module paths, file-shaped paths, malformed encodings
and raw/decoded traversal. The Vite configuration imports this pure helper and
keeps dev HTML-proxy handling, API proxying, no-store entry responses and empty
HEAD bodies.

The sole authorized affected-unit command was run once from `web`:

```text
npm exec -- vitest run --config apps/portal/vitest.config.ts --configLoader runner src/routes.test.tsx src/document-policy.test.ts src/features/realization/navigation.test.tsx
```

Result: `Test Files 3 passed (3); Tests 6 passed (6); exit 0` (Vitest 4.1.11,
1.37s). No other checks were run. Unit success does not establish an installable
build or runtime/deployment behavior.

## Final SHA-256

| Path | SHA-256 |
|---|---|
| `web/apps/portal/package.json` | `c415b202eeb95e67ab9324afa849a362e4acec0bbe57e2e36acc4236aa5d37ee` |
| `web/apps/portal/index.html` | `22922e6433a24e9b7883a4372a6c527bde452dc924e1ccddf226692f06fef418` |
| `web/apps/portal/vite.config.ts` | `59d45ae3f8cbffa2d45bb9e4350605447054a92058aa746b79c51b99877e596a` |
| `web/apps/portal/src/main.tsx` | `69f7e4ad0092f90f08c7a9b7d8bc2549dc0fa1bd7a2d617f3eefd07f8694b67f` |
| `web/apps/portal/src/sections.tsx` | `c822f52a062b10ea7f1e481687df3c8c17fdce8e2906e7d8e2a1645c39d3842c` |
| `web/apps/portal/src/document-policy.ts` | `22f0ec2f8a0ab8eb03fc30483ea1876362fca05a2ceb4f57ad939bf4c4716129` |
| `web/apps/portal/src/document-policy.test.ts` | `b6031afd875ddffdafbeef697653710402737f7b5337e7c3a51d1eeafb53c75b` |
| `web/apps/portal/src/routes.test.tsx` | `77a8b2fa6d8b06ba968031b8324d71285218886dedda2725ef7c4e3692f900f1` |
| `web/packages/kit/src/shell.tsx` | `67f559021003e8aba26f8bdf4dba58df7b7d510f2aefee983c5d69b91e6f58cb` |
| `web/package.json` | `165b9ffae9bb1936595d112ee2654262cce67d006143e66a0708d7e41110f125` |
| `web/package-lock.json` | `25bb5b4798d12f8ce1308171b3cadc8d4f47c3268cb01930f97ab6bf837d286a` |
