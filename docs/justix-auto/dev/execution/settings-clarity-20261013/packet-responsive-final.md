# Final responsive correction — /root approved

Final rerun narrowed remaining failure to two CSS issues. Sole writer owns only
realization/pages/settings.css and realization/design.css. Browser paused.
Run root guard first; branch unchanged. Preserve all other source and dirty files.

DOM evidence at 390px: sidebar now 76px; document still 398px. Settings company
card is 274px wide, its grid body has 238px available, but nested verification
Panel forces a 284px auto track via its unwrapped .section-head title + action.
Correct page-locally: company-body grid-template-columns:minmax(0,1fr), direct
children min-width:0, fact-groups min-width:0, and inside the settings company
card allow nested .section-head to flex-wrap with gap. At mobile, stack its title
and actions if needed; keep buttons reachable. No overflow:hidden workaround.

Actual DOM shell class is exactly 'app-shell dealer-shell'. The previous repair
added nonexistent .unified-shell-scoped selectors plus .dealer-shell fallbacks.
Replace these new repair selectors with .app-shell.dealer-shell consistently;
do not change historical unrelated rules. At <=1180px the shared rule hides a
nonexistent .brand-copy, so actual .brand > div:last-child still overflows and
squeezes the J mark. In the same scoped breakpoint hide that brand text child
and set .brand-mark flex:0 0 36px. Preserve existing topbar context controls and
152px mobile offset from the preceding repair. No behavior or shared-kit changes.

This is a diagnosed CSS correction, not a redesign expansion. Use apply_patch;
no tests/browser/Git/runtime/children/canonical edits. Write final source hashes
to implementation-responsive-final.md; release writer promptly. Primary reruns
only settings/mobile shell scenario and finishes the already-planned checks.
