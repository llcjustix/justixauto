# P08 owned document lookup seam approval

Application hub grants the current P08 writer two exact additional paths:

- internal/modules/documents/service/ports.go
- internal/modules/documents/repository/store.go

Existing Store exposes Share, which writes a grant while fetching ownership;
Readable alone does not establish ownership. Add a read-only Owned lookup by
owner company and file ID. BindPassport must validate ownership, exact
passport-copy purpose and Sensitive before invoking Share. Preserve tenant
not-found semantics and existing private storage access.

Exercise rejection/order through service fakes in the already owned
passport_test.go. Existing affected service units and pure app selector remain;
no database-backed repository test, broader scope or runtime action is added.
