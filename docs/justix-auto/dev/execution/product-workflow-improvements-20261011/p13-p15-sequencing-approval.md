# Contract/scoring UI sequencing refinement

Application hub approves P11 → P12 → P14 → combined P13/P15. This preserves
logical dependencies: P14 consumes P11; P13 consumes P11/P12; P15 consumes
P13/P14. No simultaneous source writers or new product scope.

P14's scoring contract will therefore be available before one frontend worker
implements the overlapping contract/insurance sale-dialog workspace. Exact owned
source union is P13 + P15, fresh context ceiling 28k; split if needed. Consume
actual predecessor endpoints/DTOs, not draft assumptions. B2B contract UI
remains included; scoring/insurance applies to the approved retail flow.

Run only the two affected Realization files together once after both changes:
src/contract-workspace.test.tsx and src/sale-insurance.test.tsx, with the explicit
Realization config/runner from web/. No broad checks or runtime actions.
Return result-13-15.md with separate R9/R11/R12 evidence.
