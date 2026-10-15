# Packet01 follow-up — chooser container specificity

Application authority /root; bounded correction within existing U2. Dispatch
only after Packet02 releases the exclusive source writer. No new product policy.

Primary observed the adjacent existing responsive selectors while accepting
Packet01: Admin design.css at max-width 580px applies `.admin-sidebar nav {
display:flex;flex-wrap:wrap }`; kit design.css at max-width 600px applies
`.ins-sidebar nav { display:flex }`. Their specificity beats the new unqualified
`.portal-switcher` container rule. Dedicated item styles are already scoped.

Only change the chooser container selector in web/packages/kit/src/ui.tsx to
`.sidebar .portal-switcher, .ins-sidebar .portal-switcher, .admin-sidebar .portal-switcher`.
Retain its declarations and all item styles/behavior. This makes the existing
grid layout win in all three shell variants at narrow widths without changing
the normal page navigation. No `!important`, feature CSS or new abstraction.

Owned writes: web/packages/kit/src/ui.tsx and result-01-css-followup.md in this
execution directory. Before source SHA-256:
56502c5e45d19fa2228f4121b11d365a14e52a677fe7575b129efa2129bd370d.
Use apply_patch after bash tools/check-git.sh. Root/web scoped rules apply.
No other source, credentials, Git or runtime operations.

This is a selector-only correction with unchanged component behavior. No unit
test exercises browser layout; do not add CSS-string assertions or repeat the
already passing 13 portal behavior units. Browser/visual checks remain manual,
per the user's verification policy. Record final source hash and release lease.
