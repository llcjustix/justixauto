# P21a2 app-unit compile seam

Application hub accepts the concrete compiler finding reported by the active
P21a2 writer: predecessor commerce/handler/dto.go imports httpx twice. The a1
service selector did not compile that handler; the required a2 application seam
now exposes the duplicate import. No broader handler behavior change is needed.

Add exact temporary source ownership of
internal/modules/commerce/handler/dto.go solely to remove the duplicate identical
httpx import. Preserve all DTO/mapping behavior and other imports. Record the
small correction and source hash in result-21a2.md. This is completion of the
actual failing compiler dependency, not a review/QA or scope expansion.

The owned consignment test fixture's nil catalogue is also an implementation
failure to fix within its existing path. Rerun only the affected service/app
selectors after their fixes; do not add or repeat unrelated checks. No Git,
SQL/runtime or new verification gate is authorized.
