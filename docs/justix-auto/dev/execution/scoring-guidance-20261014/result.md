DONE

Implemented the revised independent-signing behavior: the retail insurance tab/tabpanel is «Страхование»; optional demo scoring no longer blocks or warns on contract signing; the shared server signing guard is removed while contract, transaction, terms, permission, and handover-insurance behavior stays unchanged. Existing service cases now cover an unscored first signature and installment finalization without scoring.

No tests, runtime, browser, build, lint, typecheck, Git writes, or canonical-document edits were run by this worker. Follow-up fixture repair: `installment-servicing.test.tsx` now defaults to no permissions and grants `retail.invoices.manage` only to the existing invoice-capable case. Excluded: insurance/handover policy, application permission gates, DTO allowed-actions (already independent), and the unrelated identity test/test_creds.txt.

SHA-256:
- d6b9ac5cf8ac078f738adae30341ebc1589c117b268ce2203c87eb64508f12e0 web/apps/portal/src/features/realization/pages/retail.tsx
- 2f0eb834b2d3563d1067625fe45d4a573ec45d4a176e32dad08dde9ffe5cbecf web/apps/portal/src/features/realization/contract-workspace.tsx
- 23d8a4eca856f9734bbc64275b7895be63b280e8beb0cf237f40828c2209b7ef web/apps/portal/src/features/realization/sale-insurance.tsx
- 247b47eb47bb2f2f91b6ef2b0fdf275a634211c314f0c8ccea0419f5dc31f267 internal/modules/retail/service/contract.go
- 9d01090941269c0df91a5eadbb3055cf72cc94ce1ba0e28ee5a1fe4b116577a5 internal/modules/retail/service/scoring.go
- 9c38ba68c38d3c5da5500f5a78465195e60dc7e89c62c865902112346cd4cb69 internal/modules/retail/service/scoring_test.go
- cc01b45e119a1cba00616cdb8d2222b17c3d612716757f32e3a39cf5370458f3 internal/modules/retail/service/contract_test.go
- 86d3df69e1a699bcac86abddccffaddc9f1ae0b33a270b1e96593bcd79e5b9f7 web/apps/portal/src/features/realization/installment-servicing.test.tsx
