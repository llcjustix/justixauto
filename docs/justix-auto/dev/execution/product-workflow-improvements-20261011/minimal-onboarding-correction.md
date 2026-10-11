# Minimal onboarding correction

Coordinator decision, 2026-10-11, based directly on original user requirement #2:
«когда создаешь на адмике название компании и админ логин пароль достаточны».

Admin creation must accept name + first-admin login/password without userType.
An optional dealer/distributor selector may be shown; if omitted the company
stays untyped and unverified until platform Admin assigns the type. A set type
is immutable. Verification and new commercial actions still require a valid type.

P01 implementation made type mandatory; P02 owns the bounded correction in
identity/service/company.go, company_test.go and, if needed, the existing
company_verification_test.go. This is the sole active writer. Run affected identity
service units once for this new change, then the planned P02 frontend units.
Do not claim P01's earlier passing units prove this amended behavior.
