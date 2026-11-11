# 18 — Send deploys with references only to a ready runner, and show which runners are ready

**Status:** resolved

**Blocked by:** 16

Read first: `practices/go.md`, `practices/architecture.md`, `practices/testing.md`, `practices/react-guide.md`, `practices/design-language.md`, the spec (Runners, A deploy;
States and who hears about them).

## What to build

- Dispatch (`internal/runner/handler.go`, which pools on the machine) sends a deploy whose stack has
  references only to a runner that reported `ready`; one that is offline is waited for as work already is.
- Deploy is refused straight away (`ErrConflict`, over HTTP and MCP) when no runner on the machine has
  ever reported `ready`: "no runner on <machine> can read 1Password references".
- Web: the machine's runner list shows a tag per manager, or "token rejected". A stack with references
  whose machine has no ready runner shows a warning in its environment section.

## Acceptance criteria

- [ ] On a machine with a ready and a not-ready runner, a deploy with references only reaches the ready
      one; a deploy without references reaches either.
- [ ] The refusal is tested over HTTP and MCP.
- [ ] Component tests for the tag and the warning; screenshots at 768, 1024 and 1440px.

Verify: `make lint`, `go test ./internal/runner/... ./internal/deploy/...`, `bun run lint`,
`typecheck` and `test` in `web/`.

## Comments

Built: `secret_managers` on `deploy.requested` and a dispatch fit that needs a runner ready for each; the refusal in `Service.start`, shared by Deploy, Retry, Rollback, HTTP and MCP; `secretref.Schemes` and `ManagerName` (the runner's failure wording uses it too); the runner list tag; and the stack environment warning. A runner last reported unreachable makes the deploy wait rather than refusing it, since its token is valid. A push-triggered preview refused this way is skipped, as other conflicts are. Checked against a real server: a stack with a reference on a machine with no ready runner gets 409, while one with a ready runner and one with no reference get 201. Screenshots at 768, 1024 and 1440px.
