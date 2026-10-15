# Accounting cabinet frontend packet result

Status: DONE (source handoff; primary owns the requested final focused unit run).

Implemented the seller-only permission-admitted `/accounting` section with shared
BillingPage and profile/security, exact-preset accountant onboarding, removal of
the seller Insurance sidebar item, and a scoped responsive sale-readiness stepper.
The stepper uses only applicable checklist/invoice facts, marks matching submitted
or pending evidence as awaiting review, preserves the delivery focus target, and
opens existing dialog tabs or the server-authorized registration form.

Updated existing coverage:
- `routes.test.tsx`: accounting routes plus retail/commerce authorization and read-only denial.
- `portal.test.ts`: pure-accountant default over stale seller history, mixed-role history, and inactive seller accounting selection.
- `company-employees.test.ts`: exact nine-key preset assignment.
- `cash-sale.test.tsx`, `installment-servicing.test.tsx`: readiness wording/blocked-policy assertions.

Suggested final focused run (not run by this worker):
`pnpm --filter @justixauto/portal test -- routes.test.tsx cash-sale.test.tsx installment-servicing.test.tsx`
and `pnpm --filter @justixauto/kit test -- portal.test.ts company-employees.test.ts`.

Follow-up fixture correction after the first final run: accountant creation now
waits for the preset-role query to enable its button; the inactive-seller route
case uses BrowserRouter so SessionGate remounts retain the selected URL. Root
will rerun only those two existing cases. The original source hashes below are
preserved as the first handoff evidence; replacement test hashes follow them.

Excluded: backend/schema/API changes, canonical documents, runtime/browser work,
tests execution, and protected unrelated files.

SHA-256:
- `a8abbd8ba3919ff05d492a384ce6831c65e85136423f1b5d54cfe5060cd6c6d6` `web/apps/portal/src/sections.tsx`
- `0a1becf104fc6d7a00cb6c383c3ef58d3d424c641696156dbf0ad2932c8c419b` `web/apps/portal/src/routes.test.tsx`
- `40f9529fc4df6017a37f3539d701afd285d3b4ad7046b973c06c576e01a50469` `web/apps/portal/src/features/accounting/navigation.tsx`
- `4e551a0da4adf78b1dc58d58815b0b43bff292df6946332fa2e68ed2c94093bf` `web/apps/portal/src/features/realization/navigation.tsx`
- `1fb065e2e3e063b90725862c46f325d95a402466eb9d8d8b630dbc5fd031b476` `web/apps/portal/src/features/realization/pages/retail.tsx`
- `462274acee4040fafe0dd621500a27513f52ba1a73b41dc952ed720770751158` `web/apps/portal/src/features/realization/delivery-readiness.css`
- `063badd6254910a8303830f2b4a97b05ff6030729f6e7e6cd3e53d403db6fb99` `web/apps/portal/src/features/realization/cash-sale.test.tsx`
- `1ff6924a9d7fd37aa43d00fc26652d95eac84c213ca5865d948d0bbe9f754e62` `web/apps/portal/src/features/realization/installment-servicing.test.tsx`
- `a3d0814214dac886c309fbd81255bc24e876cbee4672cc3fd34c40ab10de1fc3` `web/packages/kit/src/section-selection.ts`
- `560df33bbac3ca6de1c4203a6305b85648b31c6ef51e2fb488ef8dc194635d09` `web/packages/kit/src/portal.tsx`
- `c7e04ac1025626f5f28f1dc8a5007f2b791219c4a23165294069fdbbb649ecf8` `web/packages/kit/src/portal.test.ts`
- `9ab35b981108379bea3120ec3ce549e62b83f39d0d6468145dca98704eba5fc2` `web/packages/kit/src/company-employees.tsx`
- `88082ad9342a004b6e1e5f727c03e0cf0a82cc9950063eddafe9aa64a193888d` `web/packages/kit/src/company-employees.test.ts`

Replacement test SHA-256 after fixture correction:
- `2f9123ede1852fc6864872f2019c4717cb46de79ef36f8ee57445f4c5f186ef9` `web/packages/kit/src/portal.test.ts`
- `5e8adce2720d171c161e8a397d906943d492627c911282113adf3ac76e20d4a2` `web/packages/kit/src/company-employees.test.ts`

Final employee-fixture SHA-256 after replacing an unavailable matcher with the
native DOM attribute assertion:
- `1c37a0b5ec2d7424ade0925a2c732b215b0121f6f36571e05fa6d67172703569` `web/packages/kit/src/company-employees.test.ts`
