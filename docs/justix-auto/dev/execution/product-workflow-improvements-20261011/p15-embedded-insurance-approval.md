# P15 embedded insurance component seam

Application hub accepts the frontend preparation finding from
`/root/contracts_sale_frontend`: Kit currently exposes `InsuranceDialog`, not
the packet's assumed `InsuranceCase`. P15 may extract its case body/actions
into an embedded component in the already owned
`web/packages/kit/src/applications.tsx`. Preserve the existing dialog wrapper
and API for current callers; embed the body in the sale dialog without nesting
another dialog. This is the approved R11 reuse intent, not new product scope.

The combined P13/P15 worker remains read-only READY until completed P11, P12
and P14 results and an explicit source-writer grant. No extra checks required.
