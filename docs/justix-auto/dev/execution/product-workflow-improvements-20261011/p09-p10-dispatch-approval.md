# Combined P09/P10 frontend dispatch

Application hub approves one fresh context for the exact union of P09 and P10
after P08 is complete. One source writer, same feature branch, context ceiling
22k; no new production paths or product decisions. Consume result-08.md/current
DTOs before implementing the passport UI.

Implement optional passport fields/private copy uploads in CRM and the existing
same-company warehouse Move action as specified in their packets. Run only the
two affected Realization suites together once: src/customer-passport.test.tsx
and src/vehicle-move.test.tsx, from web/ with its explicit config/runner.
No extra verification/runtime/Git actions. Return result-09-10.md with distinct
R6/R4 behavior, exact endpoints and unit evidence; release writer afterward.
