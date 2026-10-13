# Local UI and password corrections — source completion

Application authority /root accepts the two bounded packets and the final
selector-only correction. All source writers released. Branch:
fix/portal-ui-password-20261013; dependency
31b018ce1a4b49a2f56148459b4efa0893d345ce with dev ancestry 4fa683b.

## Delivered behavior

- No redundant section block when the current section is the only destination
  and no alternate company-context choices exist. Multiple admitted sections
  and inactive-company selection remain available through the internal switcher.
- Switcher links/buttons use dedicated shared styles with natural height,
  wrapping and focus/active states. Current section indication also works on
  deep pages. Container specificity preserves the grid at narrow widths in all
  sidebar variants. Permissions, context sequencing and routing stay unchanged.
- Identity accepts passwords of at least 8 Unicode characters. Existing 128
  maximum, confirmation validation, hashing and stored credentials are preserved.
  All six frontend help texts agree with the shared server validation; creation,
  onboarding, bootstrap, reset and own password change use that same validator.

## Evidence and limits

- [Packet01 result](result-01.md): portal.test.ts, 13 passes after correcting
  one initial test syntax typo. This covers switcher visibility, actual section
  alternatives, company selection, active indication and existing route behavior.
- [Packet02 result](result-02.md): four affected frontend unit files, 20 passes;
  selected Go TestValidatePassword (10 boundary cases) and
  TestPasswordHashRoundTrip passed. Each passing selection was run once.
- [Selector follow-up](result-01-css-followup.md): only chooser container
  specificity changed after the portal unit run; no component behavior changed.
  Final ui.tsx SHA-256:
  e3f7cc4e64276ed941a7fc4ef6dd624e15404438b691cf4e21e02545d46c2936.
  Other final source hashes are in the two packet result files.

No automatic lint, typecheck, build, browser/E2E/integration, React Doctor or
independent review/QA ran. Unit evidence does not establish visual layout or
release readiness. Existing local watchers were left running for the user's
manual testing; no new runtime readiness or live password mutation is claimed.
No database, credentials, bootstrap, migration, private env or remote operation.
The user's untracked test_creds was preserved unread and unstaged.

Canonical completion before-copies and SHA-256 records: entry/completion/.
Version-control may create one explicit-path local task commit; no remote push,
dev/main promotion or deployment is part of this correction.
