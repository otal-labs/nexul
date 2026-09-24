# 36 — Test with AI play

**Type:** implementation
**Status:** ready-for-agent
**Blocked by:** 33, 34
**Decided in:** ticket 10

## What to build

Seed a Test with AI play shown only in testing-stage columns. It follows the interview memory's testing strategy, checks the live URL against the acceptance criteria, runs the project's tests, adds a test for the criteria where the interview calls for an automated suite, then passes or fails the ticket as a person would, signed Nexul on behalf of the starter.

## Acceptance criteria

- [ ] The play appears only in testing columns
- [ ] A failure posts the bug template to the thread like a person's Fail

## Surfaces

- Seeded per workspace like the other defaults

## Read first

`practices/go.md`, `practices/architecture.md`, `practices/testing.md`, and the decision tickets above.

## Verification

`go test ./internal/...`, `make lint`, `make coverage`.

## Files likely touched

- `internal/plays/usecase.go (seeds)`

**Size:** S
