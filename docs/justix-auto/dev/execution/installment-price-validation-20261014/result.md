DONE

- `previewTerms` now rejects a missing or invalid price before percent conversion with clear price guidance.
- Percent-origin amount display is derived from the current typed percent and current price using the existing BigInt helper; money-origin and fixed invoice behavior remain unchanged.
- Source SHA-256: `ec04dacd9f977ae623493abe6fd2b191e92d6d5a00ad0fc5433199603329aef3` (`web/apps/portal/src/features/realization/installment-terms-form.tsx`).
- Evidence: `bash tools/check-git.sh` passed before the source edit. No tests or runtime checks were run, as required by this packet; unrelated `internal/modules/identity/service/company_contract_test.go` and `test_creds.txt` were untouched.
