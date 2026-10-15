# Packet 01 CSS follow-up result

Status: DONE

Changed only `web/packages/kit/src/ui.tsx`: scoped the existing chooser-container grid selector to `.sidebar`, `.ins-sidebar`, and `.admin-sidebar`, retaining all declarations and existing item styles.

Final SHA-256:

```text
e3f7cc4e64276ed941a7fc4ef6dd624e15404438b691cf4e21e02545d46c2936  web/packages/kit/src/ui.tsx
```

No test ran: this selector-only correction changes narrow-layout CSS specificity, which the portal unit suite cannot exercise. No browser, visual, lint, typecheck, build, review, or QA checks ran. Source writer lease released.
