# P04 provisioning seam approval

Application hub accepts proposal-p04-seams.md, SHA-256
`beecd6a83e7023f73eb1549c4ae7e99d5ec2363def54c7f422a14cd88f984271`.
This is bounded implementation preparation, not an independent review/QA gate.

Additional exact source ownership: identity/service/company.go,
identity/service/company_roles.go and identity/service/company_roles_test.go,
under internal/modules/. Add identity/service to the existing affected Go unit
selection; use the authorized writable GOCACHE and -vet=off.

Create an ordinary, editable, private accountant role for existing and future
seller companies. Both CreateSeller and shared Admin provision must use the
same helper/predicate inside their existing transaction. Do not assign it to
employees automatically, create a new system role, or broaden tenant access.
Preserve existing first company-admin membership.

Migration must preserve custom roles and their permissions. If a live role is
already named Бухгалтер, give the new preset a distinct descriptive name such
as Бухгалтер (счета и оплаты); use a deterministic seed identity/marker for a
precise down migration. Do not identify or delete custom roles by display name.
Report any further collision rather than overwriting existing data. No runtime
DB inspection or migration application is authorized.

Permission keys and invoice read behavior follow the proposal. Preserve branch,
party and tenant checks. No typecheck/build/lint/integration/review additions.
