# Contributing to Nexul

Thanks for taking the time. The full contributor guide lives on the docs site;
this page is the short version.

## Before your first pull request

- Read the [Contributor License Agreement](CLA.md). Nexul is open core
  (AGPL-3.0 outside `ee/`, commercial inside), so the project needs a CLA to
  keep licensing both parts. You agree to it by ticking the box in the pull
  request template. See [Licensing](https://nexul.io/docs/contributing/licensing/).
- Open an issue or a discussion first for anything larger than a bug fix, so
  we agree on the shape before you spend time on it.

## Setting up

Go (version pinned in `go.mod`) and Bun are the only prerequisites.

```sh
go run ./server/cmd          # API server
bun install && bun run --cwd web dev   # web app
make test                    # Go tests
```

The debug compose stack, make targets, and test commands are described in
[Local development](https://nexul.io/docs/contributing/local-development/).

## Standards

Every change is reviewed against the files in [`practices/`](practices/README.md).
Read the one for the language you are touching before you write code; the
[coding standards](https://nexul.io/docs/contributing/coding-standards/) page
on the docs site is a digest of them.

- Read `AGENTS.md` and `CONTEXT.md` before touching code. `CONTEXT.md` is the
  vocabulary and it is authoritative.
- Durable decisions get an ADR in `docs/adr/`. Open work is tracked in
  `.scratch/`.
- Go changes keep the coverage gate green (`make coverage`). Web changes pass
  `bun run --cwd web lint` and `bun run --cwd web test`.
- SQL lives in `queries/*.sql`; run `make sqlc` and commit the generated code.

## Pull requests

- One topic per PR, branched from `master`. PRs are squash-merged.
- Fill in the PR template. CI has to be green before merge.
- Frontend changes include screenshots at 320, 375, 414, and 768 px, dark and
  light.
- Write commit messages and PR descriptions from the code's point of view:
  what changed and why it matters.

## Reporting bugs and security issues

Bugs go through the issue templates. Security issues do not: read
[SECURITY.md](SECURITY.md) instead.
