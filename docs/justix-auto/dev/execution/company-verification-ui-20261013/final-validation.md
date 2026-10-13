# Final browser validation — 2026-10-13

PASS for the bounded frontend workflow. Executed by /root after both source
files were complete; Chrome through CUA against actual portal/Vite source with
an isolated in-memory API fixture. No actual company, account or backend writes.

Source identity on fix/company-verification-ui-20261013, dependency 6df3005:
- Realization settings.tsx SHA256: 26906cda14249f95408e516975d7aa9abb59df6936ab2556a8a4b6bc28ddce96
- Admin pages.tsx SHA256: a41a1eff82c5b644b2eeeaaef5d948cb276a1a0c61d61ee3d17409ef2d686914

Reproduce: run fixture-server.cjs with the repository-pinned Node while Vite
serves 127.0.0.1:5191. Fixture ports 5192/5193 are company/platform admins,
5194/5195 are readers. All API calls remain in the fixture; only UI modules/assets
are proxied to Vite. Open /settings or /admin/companies in CUA.

Observed acceptance:
1. Filled company displays enabled primary submission above its profile facts.
2. Submit -> pending in Realization and the separate Admin verification column.
3. Verified filter excludes pending company; reset restores it. Review card
   displays tax ID/bank requisites and submitted-only review actions.
4. Empty return reason is rejected without changing state. Returning with a
   reason updates Admin and client after same-session status refresh.
5. Client edits its address, resubmits, Admin verifies. List/detail and refreshed
   company session show verified; review/submission actions disappear.
6. Company reader cannot submit and receives permission guidance. Platform reader
   cannot review/assign type and sees missing type/permission guidance.
7. Unrelated Admin address edit preserves exact tax ID and bank requisites.
8. Injected failed company reads show client error and Admin loading/error/retry;
   restore fixture reads and both retry paths recover.

Raw structured observations and six successful API writes with revisions 1–6:
[browser-evidence.json](browser-evidence.json). One deliberately blank reason
request was rejected by fixture assertions. This is expected negative evidence,
not an unexpected test failure. Session trace confirms verified company refresh.
Screenshots: /private/tmp/justixauto-verification-admin.jpg and
/private/tmp/justixauto-verification-client.jpg (synthetic data only).

Limitations: fixture responses validate frontend interaction and request shape,
not real backend authorization, persistence, migrations, or every network race.
Successful mutation followed specifically by failed refetch, concurrent editors,
and type assignment by a permitted reviewer were not exercised separately.
No unit tests written/run, builds, typecheck, lint or automatic review/QA loops.
Initial fixture listener was sandbox-denied; auto-approved local-only retry ran
successfully. Python socket read was sandbox-denied; curl collected the same
local fixture receipt. No browser scenario failed. Fixture process and test tabs
are closed; actual application/API/database remain running and unchanged.

Primary accepts source with the above evidence and limits. Source/browser leases
released. Protected backend test SHA256 remains
a4d4b3f67107b5362810dcc96f68f1de8e59cae466dacafb7e100e29a59fe99f;
test_creds.txt remains unread/untracked. No remote promotion authorized/performed.
