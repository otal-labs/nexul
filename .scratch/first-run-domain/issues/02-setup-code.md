# 02 — Setup code while no owner exists

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** None — can start immediately

## What to build

While no user exists, the server writes a fresh one-time setup code on every boot to `<data>/enroll/setup` (0600, `hostcred.MintCode`, only the hash stored with an expiry), and deletes the file once the first user exists. The installer waits for the file like it waits for the runner code and prints `Setup code  nxs_…` in the summary. `nexul status` prints the current code while the file exists.

## Acceptance criteria

- [ ] Fresh boot with no users: file exists, hash stored, a restart rotates it
- [ ] First user created: file gone, stored codes invalid
- [ ] Install summary and `nexul status` show the code; neither prints it once an owner exists

## Read first

`practices/go.md`, `practices/testing.md`, `internal/runner/enrollment.go` (`WriteInstanceEnrollment`), `internal/platform/hostcred`, `server/cmd/workers.go`, `internal/install/install.go` (`waitCode`).
