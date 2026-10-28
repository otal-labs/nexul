# Contributing to Nexul

Thanks for taking the time. This is the short version; the
[contributor guide](https://nexul.io/docs/contributing/) has the rest.

## Before your first pull request

- Read the [Contributor License Agreement](CLA.md). Nexul is open core
  (AGPL-3.0 outside `ee/`, commercial inside), and the CLA lets the project
  keep licensing both parts. Tick the box in the pull request template to
  agree. See [Licensing](https://nexul.io/docs/contributing/licensing/).
- For anything bigger than a bug fix, open an issue or a discussion first so
  we agree on the shape before you build it.

## Setting up

Go (version in `go.mod`) and Bun are the only prerequisites.

```sh
go run ./server/cmd          # API server
bun install && bun run --cwd web dev   # web app
make test                    # Go tests
```

[Local development](https://nexul.io/docs/contributing/local-development/)
covers the debug compose stack, make targets, and tests.

## Standards

Changes are reviewed against [`practices/`](practices/README.md). Read the
file for your language before writing code; the
[coding standards](https://nexul.io/docs/contributing/coding-standards/) page
is a digest of them.

- Read `AGENTS.md` and `CONTEXT.md` first. `CONTEXT.md` is the vocabulary, and
  it wins any naming argument.
- Durable decisions get an ADR in `docs/adr/`. Open work lives in `.scratch/`.
- Go changes keep `make coverage` green. Web changes pass
  `bun run --cwd web lint` and `bun run --cwd web test`.
- SQL lives in `internal/platform/storage/queries/*.sql`. Run `make sqlc` and
  commit the generated code.

## Pull requests

- One topic per PR, branched from `master`, squash-merged.
- Fill in the PR template. CI must be green before merge.
- UI changes include before and after screenshots, dark and light: 768, 1024,
  and 1440 px for the web app, 320, 375, 414, and 768 px for the website.
- Write commit messages and PR descriptions from the code's point of view:
  what changed and why.

## Bugs and security issues

Bugs go through the issue templates. Security issues don't: see
[SECURITY.md](SECURITY.md).
