# 11 — Recognise a secret reference and reject a malformed one on every save

**Status:** resolved

**Blocked by:** NEXUL-94 (Encrypt stack env values at rest), in the Nexul tracker

Read first: `practices/go.md`, `practices/architecture.md`, `practices/testing.md`, `practices/mcp.md`, ADR 0181, the spec (Secret references).

## What to build

- A small Go package (`internal/platform/secretref`): `Parse(value)` answers literal, reference
  (scheme, vault, item, optional section, field) or malformed. Only schemes in the closed list
  (`op://`) count; every other URL is a literal; a reference must be the whole value.
- The save check on every write path that can hold a secret: stack env and branch rule overrides
  (`internal/deploy` validation), the automation pool (`internal/automations/secrets.go` `Set`), and
  connector secret fields and app config secrets (`internal/connectors`: `CredentialField.Secret`
  fields, `AppConfig.ClientSecret`, `AppConfig.PrivateKey`). A malformed reference is `ErrInvalid`
  naming the variable or field, never echoing the value. Non-secret connector fields stay literal.
- MCP and HTTP reach it through the same use-cases; no new route or tool.

## Acceptance criteria

- [ ] A table test of `Parse`: `op://v/i/f` and `op://v/i/s/f` are references, `postgres://…`,
      `https://…` and `x op://v/i/f` are literals, `op://v` and `op:///i/f` are malformed.
- [ ] Each write path rejects a malformed reference over HTTP and MCP, and its error holds no part of the
      value.
- [ ] Masked reads, events and the audit row treat a reference exactly like a value (existing value-leak
      tests extended with a reference).

Verify: `make lint`, `go test ./internal/...`, `make coverage`.

## Comments

Built: `internal/platform/secretref` (`Parse`, `Check`, the closed scheme list) and the save check in stack env and rule overrides, the automation pool, and connector secret fields and app config secrets. Segments may hold spaces, since 1Password names can, but not leading or trailing whitespace. Checked against the real API: a malformed reference is refused naming the variable, a reference and a URL save.
