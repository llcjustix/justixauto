# Login redirect — final browser validation

Result: PASS. Primary /root, after the complete one-file implementation, with
source frozen. Dependency d078b4b0e200942735d581ea6c4be761d84e9add plus
web/packages/kit/src/portal.tsx SHA-256
89bbae47ae491902f1335e08a7c20806bf23605504eb60705b5c947b7e150fca.
Version-control will commit that unchanged source and this evidence locally.

Actual app: http://127.0.0.1:5191, Chromium, 1440x1000, isolated fresh contexts.
Every API request was intercepted; login/session/context responses were synthetic.
No real account, cookie profile or database was modified. This is frontend
browser behavior evidence, not live backend authorization validation.

Command from project root:

```sh
docs/justix-auto/dev/local/toolchains/node-24.21.0/bin/node /private/tmp/justixauto-login-redirect-browser.cjs
```

The first launch was blocked before any scenario by the macOS sandbox's
Chromium MachPortRendezvous permission restriction. The approved isolated
browser launch then completed one full pass, exit 0, with no source repair or
repeated passing scenarios. No unit tests, lint, typecheck, builds or QA loop.

Reproducible fixture is preserved as browser-regression.cjs, SHA-256
8e34bf29cbe826f3a8487dd22dceebfd6edf34f017cb692aaa36dfe0373879a0.
It uses existing local Node/Playwright/browser binaries and never starts servers.

Observed outcomes:

1. Admin at /admin/companies -> logout -> dealer login: /dashboard, one context
   PUT with If-Match "initial". Dashboard absent while confirmation was held;
   shown after context update and refreshed session. Revoked remembered Admin
   selection ignored. No transient denial or Admin API reads as the new user.
2. Multiple seller memberships: explicit company picker before any context PUT;
   selecting seller-b opens /dashboard with one context change.
3. Seller with a stale /finance/applications?old=1#previous-user URL: /dashboard
   with no context write and previous query/hash discarded.
4. Permitted bank deep URL: /finance/applications?q=keep#part unchanged, no
   context write and the actual applications page visible.
5. User with no grants or memberships: common / entry shows genuine no access,
   no context writes or accidental page admission.

All five scenarios had no uncaught page errors. Screenshot of the first result:
/private/tmp/justixauto-login-redirect-dealer.png, SHA-256
63770f205f358829e1a4d88bba72aa464a771067eb6c1af624ec4d266c0c9d3a.
Browser closed; source/browser leases released. Protected backend test remains
a4d4b3f67107b5362810dcc96f68f1de8e59cae466dacafb7e100e29a59fe99f.
test_creds.txt remains unread. No remote action or runtime restart.
