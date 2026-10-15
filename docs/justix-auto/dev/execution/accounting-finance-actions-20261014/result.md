# Result: accounting finance actions UI

Status: DONE

Implemented the default-deny Accounting workspace context and enabled it only at
`/accounting/billing`. Invoice issue/void and payment accept/reject handlers now
also reject outside that context while retaining their existing server-projected
action and permission guards. Realization keeps history and proof/refund-request
flows, with accountant guidance and read-only grouped review outside Accounting.

Accounting Billing now retains B2B supplier orders that have no issued invoice and
opens their existing order issuance flow only with the Accounting scope and
`commerce.invoices.manage`. B2B copy is neutral for both counterparties. The
retail accountant queue includes eligible reserved deals even after another
invoice exists; outside Accounting it is a view/history surface. No backend,
schema, money, or automatic installment-schedule behavior was changed.

Focused final tests for the primary (not run by this packet):

- `web/apps/portal/src/features/realization/accountant.test.tsx`
- `web/apps/portal/src/features/realization/cash-sale.test.tsx`
- `web/apps/portal/src/features/realization/installment-payment-review.test.tsx`
- `web/apps/portal/src/features/realization/installment-servicing.test.tsx`
- `web/apps/portal/src/features/realization/reservation-payment.test.tsx`

Suggested selectors cover `Выставить счёт`, `Выставить счёт за автомобиль`,
`Проверить весь чек`, `Принять весь чек`, `Принять аванс`, and `Подтвердить
полный возврат` under the Accounting provider. Negative coverage retains granted
actions in inputs while asserting commerce and reserve acceptance are absent
outside the scope.

Exclusions: no backend/schema/money-policy changes, no monthly automatic-invoice
change, no runtime/browser/live-data work, no Git action, and no unrelated files
(including `internal/modules/identity/service/company_contract_test.go`) edited.

SHA-256:

```
598873f090be68a7e324f4e6cf1801a105b38758ea385b3444781fc72833a35d  web/apps/portal/src/features/accounting/finance-workspace.tsx
283cc85aad1c96f6c254cb1eac7778da47c990fcc1a9df4bde99e3ae5478488c  web/apps/portal/src/features/accounting/navigation.tsx
0c83b7e9ac6c299d227ffb48b404a72c024030438b638d16353f3f1a8a916354  web/apps/portal/src/features/realization/pages/billing.tsx
fa0692e81085dddd73bb454945559d5027273b7b82a3448fb554af9fbfb56788  web/apps/portal/src/features/realization/pages/retail.tsx
d837705bf91c9c3ea1fd51265f522d8ceeb1acd063e76ba37a8c74c57e4727e0  web/apps/portal/src/features/realization/pages/trade.tsx
eeb27b3774446b303518990403b694159cf8be0400615f97cb0a0c05266c0d40  web/apps/portal/src/features/realization/installment-payment-review.tsx
c6882db827a0b1315795977187f8f2cb9017d11a30980f26e491736831720fa9  web/apps/portal/src/features/realization/reservation-payment.tsx
f53ecf88caf17465cc9f1ba4daa80aac673397545b2981ab556dfe255248b366  web/apps/portal/src/features/realization/accountant.test.tsx
448d5e56b92fa0d735d90a8b0aafd514c7b44bae217dc69e01aa062b54a75c7a  web/apps/portal/src/features/realization/cash-sale.test.tsx
f06f8bcc3784afee9bec4d78fc599d8e3a96e3450a0c48624c8e3b7d25815bf4  web/apps/portal/src/features/realization/installment-payment-review.test.tsx
0cc7dd505fb873722390b87ffaa1fc8625b6adb064a4bbe9cc72aef507986761  web/apps/portal/src/features/realization/installment-servicing.test.tsx
879fd622cd6a538d1586368d5e1baf2e2e2ff9d1dff3d7678bcaf77cfec44652  web/apps/portal/src/features/realization/reservation-payment.test.tsx
```
