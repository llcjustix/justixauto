# Requirement delivery map

Canonical scope: ../../product-workflow-improvements-20261011.md. This map
consumes worker handoffs; it is not an independent review or runtime verdict.
Feature remains IN PROGRESS. Branch and sole writer are in checkpoint.md.

| Requirement | Source status | Final handoffs / remaining work |
| --- | --- | --- |
| R1 fixed company dealer/distributor type; retail for both | Delivered | result-01.md, result-02.md, retail-both-types-decision.md; new reservation party controls continue in R3 |
| R2 verification, setup, minimal Admin creation and legal details | Delivered | result-01.md, result-02.md, result-03-backend.md, result-03c.md; P02 corrects optional type at creation |
| R3 future/stock reservations, advances, purchase/consignment continuation | In progress | P16/P17/P18 backend, P19 UI, P20 custody, P21a agreement/handover/receipt and P21b1–b3 consigned retail lifecycle complete; final handoffs through result-21b3.md. P21b4/P23 actual selector discovery and final custody/return workspace remain |
| R4 same-company warehouse move | Delivered | result-09-10.md; inter-company custody is separate R3/R5 work |
| R5 supplier return request and dealer confirmation | Pending dependent implementation | P20 inventory custody complete; P21 consignment authority/retail eligibility, P22 returns and P23 UI remain |
| R6 CRM passport fields/private copy | Delivered | result-08.md, result-09-10.md |
| R7 Islamic-line terminology | Delivered | result-06-07.md |
| R8 percent-primary exact installment down payment | Delivered | result-06-07.md |
| R9 retail/B2B templates, external files, signed versions/amendments | Delivered | result-11c3.md, result-12a1c.md, result-12a2.md, result-13-15-completion.md; initial partial reports do not establish completion |
| R10 accountant invoice authority and work surface | Delivered including reservation accounting | result-04.md, result-05.md, result-05-correction.md; result-18a1.md and result-18a2-completion.md deliver reservation advance/refund/credit accounting; result-19b.md plus result-19b-history.md deliver Billing and persisted evidence/refund provenance |
| R11 insurance inside sale dialog | Delivered | result-13-15-completion.md |
| R12 clearly demo scoring before first own-installment signature | Delivered | result-14.md, result-14-freshness.md, result-13-15-completion.md |
| R13 accountant acceptance of installments | Delivered | result-04.md, result-05.md, result-05-correction.md; submitted evidence remains pending until accepted |

Completed source means the named implementation and affected unit evidence,
not deployed behavior. Only affected units are authorized; no automatic
integration/review/QA loop, SQL execution, runtime activation or deployment.
Later relevant source edits require their affected unit evidence, without
repeating unrelated passing suites. Final delivery must close R3/R5 and retain
the limitations above. No complete-feature claim is authorized at this checkpoint.
