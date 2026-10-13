# Company form layout — source completion

Application hub /root accepts FE1 source completion against F1–F4. Base
3aeb7b8feddb81cb49b7f0879a041d22ce760643; branch
fix/company-form-layout-20261013; one implementation writer, now released.
Version-control will record the seven scoped source/test paths, three canonical
documents and these execution artifacts in one local commit. No remote action.

Delivered: legalForm followed by name in the first row of Admin and Realization
company forms; full-row seller type below in Admin onboarding with native radios
and dealer default. Existing one-time legacy assignment reuses the field;
already-fixed types, provider kinds, Realization permissions and API stay intact.
Shared controls retain exclusive state, required selection and field errors.

Evidence: /root/company_form_layout_impl reported DONE and released the writer.
implementation-result.md records exact unit commands, seven SHA-256 values and
all failures/repairs. Initial five-file run: 21/25 passed; four harness failures
were repaired, then only the three affected files reran with 21/21 passing.
Four unaffected Realization passes were retained: 25 distinct terminal passes.
The final CSS-only correction targets the fieldset owning both classes, scopes
its legend and permits radio wrapping. No applicable layout unit or browser
check was run. Unit evidence does not establish visual/release readiness.

No automatic review/QA, lint, typecheck, build, browser/E2E/integration or React
Doctor. No backend, schema, runtime, database, credential or remote operations.
Preserved unrelated backend test SHA-256:
a4d4b3f67107b5362810dcc96f68f1de8e59cae466dacafb7e100e29a59fe99f.
test_creds.txt remains unread and outside the commit scope.

Canonical pre-edit copies/hashes are in entry/, entry/dispatch/ and
entry/completion/. Packet acceptance is recorded in approval.md; independent
checks described as dormant in the packet were not scheduled.
