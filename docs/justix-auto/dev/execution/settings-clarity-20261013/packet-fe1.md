# FE1 — Settings clarity

Task: settings-clarity-20261013. Hub: /root.
Base: b38fc75568d1691b70a02c16f1cb1af5c26cd5e8.
Dependency/contract: approved ../../settings-clarity-20261013.md; no runtime-contract changes.
Rules: root/web AGENTS.md, agent-workflow.md; inherited product R1/R2.
Context ceiling: 12,000 tokens; stop/reslice if exceeded.
Inputs/hashes: source-hashes.json; only settings.tsx and relevant shared CSS selectors.

Owned source, exclusively:
- web/apps/portal/src/features/realization/pages/settings.tsx
- web/apps/portal/src/features/realization/pages/settings.css (new)

One behavior: make existing settings understandable without changing domain behavior.

Acceptance coverage (local IDs, not backlog additions):
- SC1: Four selectable sections: Компания; Филиалы и склады; Сотрудники и роли; Личный профиль. Preserve staff/account functions; move work/branch scope and existing branch/warehouse management into their section.
- SC2: Company header places legal form before name, displays fixed dealer/distributor type and labelled cabinet access, retains edit/create actions.
- SC3: Compact verification banner above facts. Verified shows success without blanket warning; explain missing business type/inactive cabinet only when present. Unverified/submitted/returned explain next steps; retain reason, submit/resubmit, manual refresh, sequential session/profile refresh, busy/error/loading, revision and duplicate-submission guards.
- SC4: Реквизиты and Контакты groups expose tax ID, bank details, legal name/registration, country/region/address, separate phone/email. Empty values show Не указано; long values wrap.
- SC5: Import sibling CSS; every new selector scoped under .realization-settings. Bounded width, blue/neutral shared controls, responsive navigation/groups, keyboard focus and accessible selected state; no desktop/mobile page overflow.
- SC6: Preserve forms, permissions, version headers, requests, query refreshes, loading/error/empty states. No required fields, domain rules, dependencies or shared-style edits.

Locks/dependencies: hub reserves sole checkout writer; version_control owns branch/Git. Run mandated repository-root guard before source edits. Preserve dirty backend test; never read test_creds.txt. No remote push, restart, migrations or real company mutations.

Final browser scenarios, primary only after complete implementation at recorded final SHA:
1. Desktop actual Vite source, isolated fixtures: SC1–SC6; four sections, all verification states, permitted/read-only actions, refresh failure/duplicate protection and retained dialogs.
2. Mobile same SHA: SC1/SC4–SC6; navigation, empty/long facts, keyboard access and overflow.

No new tests, unit runs, builds, lint/typecheck, React Doctor or review/QA chains. Stop on ownership drift, changed contracts, unavailable isolated validation or undecided policy; report gaps to hub.

