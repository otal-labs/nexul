# Storage queries live in .sql files and sqlc generates the Go for them

Every insert and update in `internal/platform/storage` hand-wrote the column
list, a `VALUES (?, ?, …)` row, and the argument list, and all three had to be
kept in sync by eye; 27 files carried inserts with six or more placeholders
and deploys had 17. A column added to the schema but forgotten in one of the
three lists compiles fine and fails at runtime.

Decided 2026-09-08: queries move to `internal/platform/storage/queries/*.sql`
and sqlc generates the `Params` structs and typed functions in
`internal/platform/storage/sqlcgen` against the existing migrations
directory, so adding a column is one edit in the query plus a migration and
nothing can drift. Named parameters were rejected because they still leave
three lists to hand-maintain, and a struct-driven query builder was rejected
because it hides the SQL and adds a runtime dependency. The domain repo
interfaces are untouched: the `*Repo` structs stay as thin wrappers that map
domain types onto the generated code, so the domain packages and their fakes
never see sqlc. Generated code is committed and coverage-exempt; CI fails
when it is stale (`sqlc diff`). Queries sqlc cannot express (dynamic filters,
optional `WHERE` clauses) stay hand-written with a one-line comment saying
so. The pattern is in the Go section of the [coding standards](https://nexul.io/docs/contributing/coding-standards/#go).
