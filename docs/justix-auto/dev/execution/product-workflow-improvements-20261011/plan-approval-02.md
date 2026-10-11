# Plan amendment acceptance

Coordinator `/root`, 2026-10-11. Accepted revised plan digest:
`f8a0d756be25ba54b71b2b22294676f0e18adc3747f2bbc841d825e87ab88c69`.
Supersedes the initial digest in plan-approval.md only for test-command routing
and the approved P03b valid-principal fixture path. Scope remains 29 packets and
all 13 requirements, with no review/QA loop.

Generic web test:unit selects Kit/Admin and does not execute Realization targets.
Realization units run from the `web/` working directory:
`npm exec -- vitest run --config apps/realization/vitest.config.ts --configLoader runner src/<suite>.test.tsx`.
The previously omitted P02 add-company suite passed separately (1 file/1 test);
the earlier 8 passing frontend tests were not repeated.

Additional exact P03 backend ownership approved during execution:

- `internal/modules/commerce/service/terms_colors_test.go`: valid fixture actor
  may explicitly carry CommercialAccess; denial fixtures remain false.
- `internal/modules/identity/module.go`: only root alias
  `type Profile = service.Profile`, allowing the application directory port to
  name the exported profile without importing an internal service layer.
- `internal/app/workflow_ports_03_test.go`: pure fake-port TestWorkflowPort03;
  confirm the package has no DB fixture/TestMain side effects, then run only
  this affected app unit with -vet=off and the existing isolated GOCACHE.

Only `/root/commercial_backend_guards` holds the source-writer lock. Result
records must list these extensions and their actual unit evidence.
