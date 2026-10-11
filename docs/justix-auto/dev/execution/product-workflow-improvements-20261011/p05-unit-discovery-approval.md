# P05 correction unit-discovery adjustment

The prescribed new Kit shell.test.tsx was not discovered: the current Kit
configuration includes only src/**/*.test.ts. Application hub therefore grants
the current P05 corrective writer these exact test-file renames, using
apply_patch move operations and retaining their pure non-JSX content:

- web/packages/kit/src/shell.test.tsx → shell.test.ts in the same directory.
- web/packages/kit/src/company-verification.test.tsx → company-verification.test.ts
  in the same directory.

The second existing P02 test has no JSX either; current configuration conflicts
with its historical reported .tsx discovery, so its renamed unit must be
explicitly listed in actual output. This is a concrete discovery correction,
not a broad repeat of previous verification. Do not alter Vitest config.

Run only the two named Kit .test.ts files together, and the already authorized
Realization accountant test separately. Preserve all other green selections.
Record exact discovered filenames/counts in result-05-correction.md. This
supersedes only the shell test filename in p05-navigation-correction.md.
