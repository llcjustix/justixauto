# Source SHA-256 inventory

Observed source bytes at planning time; base identity supplied by the hub, no Git
operations performed by slicer. Prior packet output supersedes a base hash only
for that packet's owned files. If another source hash changes, stop and reconcile
with the hub. Canonical docs may receive the hub's already planned amendments;
consume their reported successor digest without reverting them.

| Source | SHA-256 |
|---|---|
| `AGENTS.md` | `a222419ecca4935c906454884eb62d6cacace6a5b80395c90a1cee870f8b8524` |
| `docs/justix-auto/dev/AGENTS.md` | `b2d5153012dfec17e20a5e55a65af858d693cae27adb66ef6a5c440599149b27` |
| `docs/justix-auto/dev/agent-workflow.md` | `4f264fb5217a9613b00df455b0661cb9b955e90793a29b17c682cbcd0010830b` |
| `docs/justix-auto/dev/unified-frontend-20261013.md` | `fbfe534a1ce9736139e08ad5831efbe2234996cb4309e7cf23c3540ab48c942a` |
| `docs/justix-auto/adr-15-unified-frontend.md` | `cc1900be4eda2ae55280bc87f4cd7680a8300b633f79788f458d43d866733859` |
| `docs/justix-auto/dev/product-workflow-improvements-20261011.md` | `32d9c7a33962db365687bdccb1495736d576a2f9e41911ae4c204ebcee931b96` |
| `docs/justix-auto/business-logic.md` | `a9ab1688fc491ab7cec27117258bf43b154459dbd144596ea813b2a2876a0b9d` |
| `docs/justix-auto/open-decisions.md` | `98d573b8a2a8a515dca0df89a5558ebad0cd4a9666e3f2bcd4fe5892210cc959` |
| `web/AGENTS.md` | `b8dee411a0a1043a038d72ccde611f5d12e70a6065a3d60eb64fde72600bdd02` |
| `internal/AGENTS.md` | `b22213a76ca11a4ce4aeb04323260d1d577f28ef1230d48c15367d074e67650b` |
| `tools/AGENTS.md` | `361f5e895fc93d6bb232941ae5f173d868b6ce4038f1b4810138bed8c4c631f6` |
| `deploy/AGENTS.md` | `28d03b9e6d5971860a9e4376d2b9c18d086c06a212697fbb7031244b8171f23a` |
| `web/package.json` | `081a98f5f8e7730c6a11a51d313641f80923ff840ba59638b64110979d0bc51d` |
| `web/package-lock.json` | `9492de1ff808e0483a69fbb9465e0da14774c34afedc1017c251bd2d07236249` |
| `web/vitest.workspace.ts` | `412b017f064d7ba59645d6146b9641759f80f8ad093be7af4c86e6c43300a9ab` |
| `web/packages/kit/src/session.tsx` | `9bc3107f613b0f2c28eff34555b474681bbef2ee0db9058ae504765270489c20` |
| `web/packages/kit/src/shell.tsx` | `e41ec62eb554748bfc594b1adf9471a12c32340479899439294ebfe903035016` |
| `web/packages/kit/src/http.ts` | `c63acb2cf643f360c403482b792c42953de1c2840f68314aef18d110e6483b67` |
| `web/packages/kit/src/index.ts` | `9f56f4fbc3b63e07e530b28f6726caaa01213229868b3723e12ddc3ad315eab9` |
| `web/packages/kit/vitest.config.ts` | `86383a384da55cc7673002878b8496725d6ab57f6af3aac4eafa40f76a6d9ff2` |
| `internal/pkg/webui/webui.go` | `d534c64ba2e045b63ebc7f76e045efea95b5b1746b76e8cf43ff02d9bbc03997` |
| `internal/pkg/webui/webui_test.go` | `a426e589f0874bbdae262c917d7fe6b34a6d28f4829d4669dafb871ef9feee08` |
| `cmd/api/main.go` | `2c880a67608e5af949b6d5a8d6f8f92fd079516382215a42317021e491e79945` |
| `internal/config/config.go` | `121a9a470024884e70deaa9a67e399c2afb92c2b89f6e0c7699151816389f6ef` |
| `Makefile` | `3a4a5e54c58fd1cd31622f1a7008aca50e5a260707354499e514016767646d2b` |
| `tools/stop-apps.py` | `e366d83a3b603933b1f189931af5bbc12f003a93a5291585cbdba2971c53823a` |
| `tools/test_stop_apps.py` | `d0841a3c3e3ba2edf0d6749ef490eb706257ea2dca0fd8b1659f805e7e6234c6` |
| `deploy/Dockerfile` | `3170d8936c8e5599508aa0e54abfaf449e03f458fb1ce372a262ed550b517ab1` |
| `.github/workflows/deploy-dev.yml` | `847eeba76438c581b9e9fb733fa832d683c0e9146b781bde65cc9687efb17aea` |
| `deploy/values.yaml` | `e7ce17260aa81b065a3182fb354eca6028ca8320c001466984151b5d9bb7fd86` |
| `.env.example` | `f4e05836449e422a1023f1ba805258ad3be6263f9711c5db33046199425083fb` |
| `.air.toml` | `87e14c4b41327336e57e184c85d8c89393c87e7f759b6e41bf132020444f9480` |
| `docs/justix-auto/dev/execution/unified-frontend-20261013/frontend-mapping.md` | `ab7e18c19456f3950d079b69a1ee03fd8cd9a2f11f40bd417641da139521c8b6` |
| `docs/justix-auto/dev/execution/unified-frontend-20261013/infra-mapping.md` | `ed3a52f1346a988eb7de3e396206fa7073d25f3ac338a74aecbd9d0eb5280e54` |
| `docs/justix-auto/dev/execution/unified-frontend-20261013/infra-approval.md` | `97e2caf0b497fc1920c05b941831100adb3713025e7bf0ac637f1f7ad075b0c9` |
| `web/apps/realization/index.html` | `590fed84f14272d612968b122727905353b98c1ac078ae0556ab2d3c0d976807` |
| `web/apps/realization/package.json` | `a5180f90c87bb001727681028ae4254e8bd41cb20b04b0e3cbadf5813cd8ce2c` |
| `web/apps/realization/vite.config.ts` | `850a4750e263aa3fc6e3649ffaa7cdc1dd3a98b5400ace6f8039e44d72705a9e` |
| `web/apps/realization/vitest.config.ts` | `e1a8f6f2114126e19b0b706c7e308df66ea8e4e1f1530f3ae6eed09cf6214361` |
| `web/apps/realization/tsconfig.json` | `003b3de5f6d5f4cdd571f27bb75086c1e6057d19ccb7a33fabd89460ffa1adce` |
| `web/apps/financing/index.html` | `3570a1cb1f3bd5ddb522e9147270b384843a59590d479ba7193af3a745b330bb` |
| `web/apps/financing/package.json` | `a5c701bbfa568ef1b49823e8dc2f277bff1e8596013c7b2c588d35a86c4d5418` |
| `web/apps/financing/vite.config.ts` | `1af498ac503eba81df15728dae1d8e28f3aef41b4df814f66fdf8683b0ed1500` |
| `web/apps/financing/vitest.config.ts` | `5150b3598cc619b5bd49a2d9217d82ac8bd110159b3f44efbdebffe4ad3a4231` |
| `web/apps/financing/tsconfig.json` | `003b3de5f6d5f4cdd571f27bb75086c1e6057d19ccb7a33fabd89460ffa1adce` |
| `web/apps/insurance/index.html` | `44322f5dd1f808b896c7817c22f5eb203256e9b877096d7578d4d4cf6ab577aa` |
| `web/apps/insurance/package.json` | `5a8bdbef7d49617121b095ea21395ce23c2b34c8f428a29d3535875efa81bb9c` |
| `web/apps/insurance/vite.config.ts` | `98795820e80f097a0f7d691e6fbf96b01055517541e3efe78472b19e3d04e984` |
| `web/apps/insurance/vitest.config.ts` | `8d8b24c58a766e3c6e005d19f7dbec7ba6cdb676b0ecbe1aa1cb93597562f57e` |
| `web/apps/insurance/tsconfig.json` | `003b3de5f6d5f4cdd571f27bb75086c1e6057d19ccb7a33fabd89460ffa1adce` |
| `web/apps/admin/index.html` | `db7829e503ca86dd231c2e10e45f23d5de9714785b5b291c8a5ac2a380a493a8` |
| `web/apps/admin/package.json` | `44d5556f1b42e88bb9aa075b478564a26be1b92fff34197dbceacac74861b7fe` |
| `web/apps/admin/vite.config.ts` | `5d8ab92e113717f5aa915e361b5073fa56f11f0d7499e143ca39e6bc0e4912c8` |
| `web/apps/admin/vitest.config.ts` | `fa8b0b3b14192a3e80a1b279edccd6ba51881c7ee3681db368510c58933a3abb` |
| `web/apps/admin/tsconfig.json` | `003b3de5f6d5f4cdd571f27bb75086c1e6057d19ccb7a33fabd89460ffa1adce` |

Every feature source hash is recorded separately in move-manifest.md.

