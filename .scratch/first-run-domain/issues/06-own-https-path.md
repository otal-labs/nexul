# 06 — "I already have HTTPS" path

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 04

## What to build

A single field for the https address the owner's own proxy already serves, checked by the shared finish from 04, with a line explaining that the proxy must forward to `http://<server>:<port>`.

## Acceptance criteria

- [ ] An https address that reaches this instance is accepted; anything else shows why not
- [ ] http:// addresses are refused with the reason

## Read first

`practices/react-guide.md`, `internal/auth/service.go` (`VerifyInstanceURL`).
