# External secrets: secret references to 1Password

**Status:** ready-for-agent

## Problem

Every secret Nexul uses today is a value it stores: stack environment values (plain JSON in `stacks`
until NEXUL-94 seals them), the automation secrets pool, connector credentials and sign-in provider
secrets (sealed with a key derived from `NEXUL_AUTH_SECRET`). A copied database together with that key
yields all of them. Teams that already keep their secrets in a manager have to copy each one into
Nexul and rotate it in two places.

This effort lets a secret be a **secret reference** (`op://vault/item/field`) that Nexul stores in its
place. The side that uses the secret fetches it with its own credential at the moment it needs it.
1Password ships first. Vault and the cloud managers come later behind the same shape.

The decisions and their reasons live in the map's tickets (`map.md`, tickets 01–10) and in ADRs
0181–0183. This spec is the build-ready whole.

**Prerequisite:** NEXUL-94 (Encrypt stack env values at rest) ships first. Literal values stay sealed
by it, and references are stored as references.

## Decisions

- **Scope:** stack environment and branch deploy rule overrides, the automation secrets pool, and the
  secret fields of connectors and sign-in providers. The bootstrap secret stays local (ADR 0183).
- **Resolved where used, with credentials that never reach Nexul** (ADR 0181). The runner resolves
  stack environment, the automations host resolves the pool with one token per workspace held by its
  supervisor (ADR 0182), and the server resolves its own connector and sign-in secrets.
- **1Password first,** through the official SDKs: Go `github.com/1password/onepassword-sdk-go` pinned
  at `v0.4.1` (CGO-free) on the runner and server, and `@1password/sdk` pinned at `0.5.0` on the Bun
  host. Both are version 0 and can break between releases, so they are pinned exactly.
- **Edition:** open source (AGPL).

## Measured facts the design rests on (ticket 07)

- One batch resolve (`ResolveAll`) is one read, whatever its size. Single resolves cost two reads each.
  Creating a client costs no read.
- The account's limits: 1,000 reads an hour per token, and 1,000 reads and writes a day for the whole
  account, shared by every token.
- Client start-up is the slow part: about 2.6 s of CPU in Go and 1.2 s in Bun. A batch of 20 takes
  about 0.6–0.8 s warm.
- Errors are typed: `itemNotFound`, `fieldNotFound`, `vaultNotFound`. A vault the token cannot reach
  answers like a vault that does not exist.
- With no network, client creation fails at once. There is no offline cache.
- The Go SDK embeds a 9.5 MB wasm core, so only `nexul-runner` and `nexul-server` import it, never
  `cmd/nexul`.

## Secret references (ticket 02)

- **Shape:** a value is a reference when the *whole* value starts with a scheme from a closed list
  (`op://` in v1). Anything else is a literal, including `postgres://…` and other URLs. A reference is
  never embedded inside a longer value; a connection string is either stored whole in the manager or
  built with compose's `${VAR}` interpolation.
- **No data shape changes:** stack env and rule overrides stay name-to-value maps, and pool entries
  and connector fields stay strings. Only `CredentialField.Secret` fields of connectors and sign-in
  providers may hold a reference; ids and other non-secret fields are always literal.
- **One parser,** shared by the server's save check, the runner and the server's resolvers (Go), with
  a TypeScript twin for the host and the web editors.
- **Validation at save:** a malformed reference (unknown shape, missing vault, item or field) is
  rejected. Whether the item exists is only known on resolution.
- **Secrecy:** a reference is treated exactly like a value. Readers without `stacks:write` get names
  with `env_masked`. Pool entries stay write-only after save (ADR 0047). Events, MCP results for
  readers and the audit log never carry one.
- **Web UI:** value inputs stay text. A value that parses as a reference shows a tag with the
  manager's name ("1Password"), taken from its scheme, and save errors show inline. This applies in
  the stack env editor, branch rule overrides, the automation secrets form, and secret connector and
  sign-in fields. Nexul never browses a vault.
- **Upgrade:** an existing value that already starts with `op://` becomes a reference. The release
  notes say so in one line.

## Runners (tickets 03, 04, 08)

### Setup

- The bundled runner and named runners can hold a token. Personal (computer) runners never take
  deploys (ADR 0146), so they never do.
- `nexul install runner … --1password-token-file F` at install, and
  `nexul runner 1password <name>` to add, rotate or remove the token later. It reads the token from
  stdin, writes it into the unit's env (the env file on systemd, the plist on launchd, the Windows
  service host's config) and restarts the runner. A token is never a plain flag value, and `upgrade`
  keeps it.
- The bundled runner has its own token, separate from the server's.

### At start

A runner with a token opens one 1Password client when it starts and keeps it for its life. It reports
its managers and their states (`ready` or `token rejected`) with its version, os and arch on connect;
the state is stored on the runner and rides `runner.connected` as a new field.

### A deploy

1. **Dispatch.** A deploy whose stack has references goes only to a runner on the stack's machine that
   reported `ready`. If that runner is offline, the deploy queues for it as work already does. If no
   runner on the machine has ever reported `ready`, Deploy is refused straight away with "no runner
   on <machine> can read 1Password references".
2. **Resolve first.** The server sends the stack's env unchanged, with references as plain strings.
   The runner resolves every reference of the deploy (env and the rule's overrides) in one batch,
   before pull, build or `compose up`, on every deploy, with no cache.
3. **Failure.** Any failed reference (missing item or field, no access, the manager unreachable, no
   token) fails the deploy and leaves the running containers untouched. The error lists each failing
   variable and its reason (`DB_PASSWORD: field not found`), never a reference or a value. The existing
   Retry is how to try again.
4. **Delivery.** Resolved values go into the 0600 compose `.env` and the run stack's `-e` flags
   exactly as literals do (`writeComposeEnv`). The runner keeps no value after the deploy.
5. **Log.** One deploy-log line: how many references were resolved and how long it took, never which.
6. **Rollback** redeploys with the stack's current env, so it uses each reference's value now.
   Preview deploys follow the same rules.

### Noticing a changed value

- At each deploy the runner writes a fingerprint of every resolved value (a keyed hash, the key
  generated on the machine and never sent) to a 0600 file in its own directory, removed with the
  stack.
- Every `NEXUL_SECRET_CHECK_INTERVAL` (default 1h, minimum 5m, `off` disables), it re-resolves the
  references of every stack it has deployed, preview stacks included, in one batch per token, compares
  fingerprints, and reports which variables of which stacks changed. It never reports values or
  references.
- Nexul shows a notice on the stack page on the web and the phone ("DB_PASSWORD changed since the last
  deploy") beside the existing Redeploy, and publishes a new `service.secrets_changed` event (stack id
  and variable names) pushed live, so automations can react. Anyone who can see the stack sees it, and
  the Redeploy button follows the existing deploy permission. Nothing redeploys automatically. The
  next deploy, retry or rollback clears the notice; there is no dismiss.

## Automations hosts (tickets 05, 08, 10; ADR 0182)

- **One workspace per token host (ADR 0182, as amended):** every automation on a host runs in its
  process as the same OS user and can read its env file, so a host holding a token for a workspace
  runs only that workspace's automations. The `instance` host runs every workspace's defaults, so it
  can hold a token only while the instance has one workspace.
- **Tokens:** set and rotated with `nexul automations 1password <host> --workspace <slug>` from stdin,
  removed with `--remove`, and kept by `upgrade`. The command asks the instance as the host
  (`GET /api/automation-hosts/self/workspace?slug=`) for the workspace's id and the workspaces placed
  on the host, and refuses a token for a second workspace or for a host with another workspace's
  automations. The token is stored as `NEXUL_OP_TOKEN_<id in upper case hex>`: keyed by id, since a
  rename can hand a slug to another workspace.
- **Placement:** the server refuses to create, move or switch on an automation on a host whose
  reported tokens name another workspace (`<host> holds a 1Password token for another workspace`), in
  HTTP and MCP alike. A new workspace's defaults on such an `instance` host are created switched off,
  and switching one on is refused with the same reason. The host itself runs only its token's
  workspace's assignments, and none with tokens of two workspaces.
- **Supervisor only:** tokens live in the supervisor and are stripped from `process.env` before the
  automation threads start, which keeps them out of the SDK's way but is no isolation. There is one
  client per token, opened at host start, and the state of each (`ready` or `token rejected`) is
  reported by workspace id; the host list shows the workspace's current name.
- **Resolution:** an automation's hello passes through the supervisor, which resolves that workspace's
  references and hands the values in, once per connection, when literals are delivered today. Hellos
  of one workspace that arrive within about a second are resolved in one batch, so a host restart
  costs one read per workspace.
- **Failure:** the automation starts with every secret that resolved. A failed name is absent from
  `ctx.secrets`, and the automation's log and page say `<NAME>: field not found`, never the reference.
  A workspace with no token on the host fails each of its references this way.
- **Changed values:** the host checks on the same interval setting, one batch per workspace token.
  When a value changed, it restarts that automation's thread as soon as no run is in flight, or after
  5 minutes at most; a run cut off then is delivered again (ADR 0046).

## The server (ticket 06; ADR 0183)

- **Token:** `nexul install --1password-token-file F` or `nexul server 1password` (stdin) writes it into
  the server's 0600 `.env` and restarts the server; `upgrade` keeps it. It is scoped to a vault holding
  only Nexul's own credentials and checked at start, with its state shown in Settings.
- **Resolution:** connector and sign-in secrets that are references resolve at start and on save (save
  already checks a connector live), then stay in memory.
- **Outage at start:** the server still starts. The affected connector or sign-in button shows
  "1Password unreachable" and retries with backoff, capped at about 100 reads a day. Passkeys and
  existing sessions keep working.
- **Rotation:** a 401/403 from a provider re-resolves that credential's references once, at most every
  5 minutes, and retries the call. A restart re-resolves everything. There is no timed check.
- **`NEXUL_AUTH_SECRET`** is never a reference (ADR 0183).

## States and who hears about them (tickets 03, 05, 10)

- The machine's runner list (web and MCP `machine_list`), the host list and Settings (for the server)
  show each manager's state: a tag with the manager's name, or "token rejected".
- A stack with references whose machine has no ready runner shows a warning in its environment
  section. An automation whose pool has references its host cannot resolve shows a warning on its page.
- When a state turns `token rejected`, one inbox notification goes to the people who can Manage that
  machine, or to the Owner for the server.

## The seam for a second manager (ticket 09)

- **Published shapes name no manager.** Connect reports, stored runner and host state, events and MCP
  fields carry a list of managers, each with its state. The only 1Password-specific string anywhere
  published is the `op://` scheme.
- **Code:** a small `Resolver` interface in Go (runner, server) and one in TypeScript (host): its
  scheme, a batch resolve, and typed errors (`not found`, `field missing`, `no access`,
  `unreachable`). A registry picks a resolver by a reference's scheme. 1Password is the only
  implementation.
- **CLI:** a subcommand per manager (`nexul runner 1password`, `nexul automations 1password`,
  `nexul server 1password`), since each manager's credential has its own shape.

## Install (ticket 10)

An interactive `nexul install` offers an optional prompt for the server's and the bundled runner's
tokens (paste one, or Enter to skip). Without a terminal it never prompts, so nothing is required at
install (ADR 0020). The install's closing summary names the `nexul … 1password` commands in one line.

## Hit every surface

- **UI:** the reference tag and inline errors in four editors; the runner and host list states; stack
  and automation warnings; the stack's changed-value notice on the web (768, 1024, 1440px) and the
  phone's stack page.
- **HTTP gateway:** existing write routes accept references through the save check. New fields on
  runner, host and stack reads.
- **MCP:** `stack_update`, connector and pool writes accept references. `machine_list` carries manager
  states. `stack_get` carries the changed-value notice as variable names.
- **Events:** the managers list on `runner.connected` and the host's update, and the new
  `service.secrets_changed`, each with a `Topics()` entry and `make event-schemas`, new fields beside
  old ones only.
- **Live push:** an audience rule for `service.secrets_changed` (the stack's viewers), `make
  live-topics`, a web follower in the stack hooks, and a `refreshes` line on the phone's stack query.
- **Permissions:** no new permission. Setting a token is machine-side, Redeploy keeps its existing
  check, and the notice follows stack visibility.
- **Reverse states:** every token command also rotates and removes. Removing a token returns the
  runner to "no manager", and its stacks show the warning.
- **Docs:** ADRs 0181–0183; the Secret reference and Secret check terms in `CONTEXT.md`; one new user
  guide page "External secrets" (a service account per runner, host and server, references, the token
  commands, the check, errors), linked in one line from the stack, automation, connector and runner
  pages; the README row turns ✅ when this ships.

## Tests

Against an in-memory fake behind the `Resolver` interface, which can fail in every typed way. There is
no real-token test; ticket 07 measured the real behaviour. The guards to add:

- The parser: schemes in the closed list are references, every other URL is a literal, embedded
  references are literals, and malformed references are rejected at save.
- A deploy with a failing reference fails before any compose command runs, and its error holds no
  reference or value.
- Dispatch sends a deploy with references only to a ready runner, and refuses when none exists.
- An automation thread's environment holds no token, and it is handed only its own workspace's
  values.
- A host restart with several automations in one workspace makes one resolve call.
- Events, MCP reader results and audit rows never contain a reference (beside the existing value
  checks).

## Out of scope

- Building a second manager's resolver (Vault, OpenBao, a cloud manager).
- The bootstrap secret as a reference (ADR 0183).
- Browsing a vault from Nexul.
- Embedding a reference inside a longer value.

## Risks

- **Version-0 SDKs** can break between minor releases. They are pinned exactly, and Dependabot bumps
  are read, not auto-merged.
- **Budget:** 1,000 reads a day for the whole account. Planned use is about 24 reads a day per runner
  and per host token for the check, plus one read per deploy and per host restart per workspace, plus
  the server's capped retries. A large fleet on a small plan should raise the check interval.
- **Bun support** for `@1password/sdk` is not claimed by 1Password. It worked under Bun 1.3.11 with a
  real token (ticket 07), and the pinned version is the one measured.
- **Binary size:** the wasm core adds about 9.5 MB to `nexul-runner` and `nexul-server`.

## Build order

Cut into tickets 11–28 in `issues/` (ready-for-agent), each with its blockers. 11 waits on NEXUL-94,
and 15 can start at once.

1. The parser and save check, with the web tag and inline errors (no resolution yet).
2. The `Resolver` interface, the 1Password resolver and the fake, in Go and TypeScript.
3. The runner: token command and install flag, the start check and its reported state, resolution in
   a deploy, failure and the log line.
4. Dispatch to ready runners, the runner list states, the stack warning and the rejected-token inbox
   notification.
5. The automations host: per-workspace tokens, supervisor resolution and batching, failure, states.
6. The server: token, connector and sign-in resolution, outage handling, 401/403 re-resolve.
7. The changed-value check on runners and hosts, the notice on the web and phone, and
   `service.secrets_changed`.
8. The install prompt, the guide page and the README row.
