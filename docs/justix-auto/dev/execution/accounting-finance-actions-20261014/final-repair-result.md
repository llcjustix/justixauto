# Final frontend fixture repairs

Status: DONE

Repaired targeted frontend fixtures only. Accountant fixtures now give an
additional retail invoice candidate its projected issuance action and use a
complete B2B order, waiting for the order action form before submitting its due
date. The B2B assertion checks the order revision and the queue disappearance.
The cash guidance assertion matches the current accountant-cabinet copy.

`installment-servicing.test.tsx` scopes only direct financial-decision cases to
Accounting, uses modal-local grouped-review selectors, keeps the read-only
outside label, and adds the saved-draft monthly invoice command case: checkbox,
empty command body, and `deal-rev-7` If-Match; the no-scope half asserts no
button or request. The order-workspace fixture now supplies the existing session
provider and a realistic session response.

No test command was run. Root should rerun the targeted frontend failures from
the prior logs, including the selectors `Проверить весь чек`, `Просмотреть чек`,
`Выставить счета по графику`, and `Выставить счёт`.

Timing correction: the B2B queue-disappearance assertion now waits for the
post-submit Billing query invalidation.

SHA-256:

```
03b4fd62e8cccb746ecffe8e8f85fe5dda443c8661447920d5f37ffa1b7c3dda  web/apps/portal/src/features/realization/accountant.test.tsx
9924112cf4af9be76980fa83c0b15b74dcefa56fcdb1c91b453f3f57e41a2003  web/apps/portal/src/features/realization/cash-sale.test.tsx
0efedbeaf41a687472d4cd191c87e21c462bb498245abe70e14ede0bc5dd1b67  web/apps/portal/src/features/realization/installment-servicing.test.tsx
6bd581ff203f22bfa3f05aa89fa807990c7daec237e7053c52dd4bf1db7ff0a8  web/apps/portal/src/features/realization/order-workspace.test.tsx
91dbe4e8964b31e62203b656cf8afc57151213a2ff00e74e9e97286d40ed5352  web/apps/portal/src/features/realization/pages/billing.tsx
```
