/root accepts packet digest
9de99ada049b6b653cdf21bc6946b8276b8b12c872bd5ad7fc20a3cb6c115bec.
VC prepared fix/warehouse-receipt-refresh-20261013 at exact dependency 239dcfda1c59846eac8a67d24680acc93613d759
and released Git lease. Fresh worker owns inventory.tsx only; primary runs the
two selected existing receipt cases after release. No wider checks or runtime
actions. Preserve unrelated backend test and unread test_creds.txt.
