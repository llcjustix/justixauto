# P3 acceptance and P4 source lease

2026-10-13, /root. Accept result-03.md as the exact dependency handoff. P3 has
released its lease. All 94 feature moves and one portal workspace/composition are
reported implemented; the three affected unit files passed (6 tests). No build,
installation or runtime evidence is implied. P3's shell hash supersedes P2's hash
only for the assigned obsolete bootstrap removal.

Grant /root/unified_portal_runtime the sole source/test writer lease for the
12 exact paths in packet-04.md. Explicit worker gpt-5.6-terra/medium, 16k total
including completed prep. Root dispatches under the infrastructure authority's
existing ADR-15 acceptance; no new deployment authority. Consume result-03.md
and its pure document policy to align Go request behavior. Preserve `/readyz`
alongside `/healthz`; no private env or real process/service operations.

Run only the two affected pure unit commands, once per relevant change:

```text
env GOCACHE=/private/tmp/justixauto-workflow-go-cache-20261011 bash tools/go.sh test -vet=off ./internal/pkg/webui -run 'TestMount|TestPortal' -count=1
python3 -m unittest discover -s tools -p test_stop_apps.py
```

Keep FakeSystem-only Python tests and httptest/tempfiles-only Go tests. No
listeners, dependency installation, wrappers with vet, build, browser, automatic
review/QA, migrations, workflow execution or Git. Return result-04.md with exact
consumer values, hashes, unit evidence and explicit lease release. Root owns
canonical docs, which already describe the expected WEB_DIR and launch changes.
