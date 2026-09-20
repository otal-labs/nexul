# Roadmap

> Directional, not dated. Open work lives in `.scratch/`, the decisions behind
> it in `docs/adr/`, and the vocabulary in `CONTEXT.md`.

---

## First

- [ ] Decide whether an `architecture.md` at the repository root is worth
  keeping, or whether the docs site's architecture page and the ADRs are
  enough. The README no longer carries the architecture table.
- [ ] Take product screenshots on a clean instance once the repository is
  public. The website uses an illustrated example workflow; the README
  carries no screenshot until clean captures are available.

---

## Shipped

### The Loop (MVP)

A self-hostable instance where the full SDLC loop works end to end: GitHub OAuth
login, versioned docs, tickets, PR linking, deploys via host runner, live
topology canvas, an MCP server over the whole domain, one `docker-compose up`
with a SQLite spine.

### The domain build-out

Every domain built out:

- **Identity & workspace** — users, owner/first-login wizards, the member
  allowlist, projects + membership, categories + a swimlane board.
- **The event-driven middle** — full GitHub event normalization, automations
  with `ticket.finished`, notifications.
- **Deploy & runner** — service definitions, runner visibility, repo-driven
  builds.
- **Topology v2** — service / network / external node kinds, auto-managed
  service nodes.
- **Docs evolution** — access and permissions, a rich editor, `@`-mentions with
  live chips, real-time collaboration.
- **Machine credentials & ecosystem gateway** — personal access tokens,
  connection tokens, and the initial integrations contract: scoped tokens,
  signed webhooks-out, published event schemas, OpenAPI.
- **Desktop client** — an Electron thin shell over the served web app,
  bootstrapped by the connection token.
- **DNS** — a Cloudflare-first provider abstraction: records, a setup-wizard
  hook, and tunnels for zero-open-port hosting.

### The design pass

A monochrome identity — "The Mono Console", light and dark as true inversions
of each other, color reserved for status signal — rolled out page by page
across every nav page. Spec of record: the Mono Console spec; tokens
in `web/src/index.css`.

---

## Now — improving what's built

The platform works. The work now is raising the quality of each domain, one
domain at a time. Requirements that were written but never built now live as
needs-triage specs in `.scratch/`, alongside the efforts already in flight.

Before any production or public use, the four standing items in
`.scratch/pre-release/issues/` come due: fix the GitHub App callback URLs once
the public domain exists, security-review the integration model before the
store accepts third parties, register the Cloudflare OAuth app, and settle the
canvas node kinds.

Parked until the repository migration lands: **bots**, webhook-driven bots
that post into any conversation with Discord's payload and get their own tab
in Settings. The wayfinder map in `.scratch/bots/` has the Discord contract,
the bot model, and the URL and limits decided; it resumes with the mentions
and notifications ticket, then the two prototypes, the surfaces walk, and the
spec.

---

## Next — ecosystem & reach

Turn the integration contract into a platform and widen the surface area
without changing the core architecture.

- **DNS extensions** — TLS automation (Let's Encrypt) for direct-to-server
  paths, additional registrars behind the provider interface, managed subdomain
  routing for deployed services.
- **Integration store** — registry, store UI, the install/consent flow, a
  scaffolder, and the Discord reference integration end to end. Spec:
  `.scratch/integration-store/`.
- **Desktop extensions** — auto-update (electron-updater), tray and menu
  niceties, diagnostics.
- **Additional git providers** — GitLab, Gitea. The provider interface makes it
  additive, and the auth user model is already provider-agnostic. Spec:
  `.scratch/second-git-host/`.
- **Semantic search / embeddings** — the event-driven indexing pipeline is the
  seam.
- **Doc comments** — deferred during the domain sessions to keep momentum; the
  access domain regains a `comment` action when this lands.
- **Per-runner credentials** so one machine can be revoked on its own
  (`.scratch/per-runner-credentials/`), and labels/selectors for runner routing.
- **Ticket workflow depth** — blocked tickets, ticket-type body templates and
  required relations, comments and an activity timeline
  (`.scratch/ticket-workflow-depth/`); creating a branch from the ticket page
  (`.scratch/branch-from-ticket/`); one search box across every entity
  (`.scratch/global-search/`).
- **Chat and voice depth** — the message features v1 left out (emoji
  reactions, read receipts, typing indicators, private channels), interactive
  in-chat approvals instead of today's auto-decline, per-user memories and
  multiple named agents, and on the voice side DM and group calls, `@Agent` in
  a call, and recordings for standups.
- **Integrations polish** — the expose-a-service flow and the voice channel
  UI, postponed until after the public release (`.scratch/integrations/`,
  tickets 04 and 11).
- **Workspace-scoped runners, automations, and integrations** — all three are
  still instance-wide, so an automation or integration installed in one
  workspace sees every workspace's events
  (`.scratch/workspace-scoped-automation-surfaces/`).
- **Smaller gaps**, too thin for a spec: offline doc edits do not survive a
  closed tab (the collaboration state is memory-only); a project's docs list
  has no manual or recency ordering; MCP tools for automation versions and
  secrets; the GitHub webhook subscribes only `pull_request`; permission
  overwrites are wired for documents only, and the grid has no `comment`
  action; the profile display-name override is read only by the wizard; the
  HTTP error envelope carries only `message`/`code`, so field-level validation
  errors have no machine-readable shape and a 503 carries no `Retry-After`; no
  default automation posts deploy results to a chat channel, and automations
  cannot yet auto-push from a connected git repository; a failed deploy is never
  rolled back automatically (rollback is the manual one-click action only — a
  best-effort automatic one driven by the reported `failed` status, not by a
  health probe, was specified and never built, and the post-start
  health-check probe was dropped with nothing replacing it); a requested
  review notifies nobody, though sign-in already guarantees every user a
  linked provider identity to notify.
- **Second agent harness (OpenCode 2)** — once OpenCode 2 leaves beta. The
  seam is in place (`internal/harness`, ADR 0054): implement `harness.Client`
  for it, register the kind, and let the pair form pick a kind.

**Exit criterion:** an owner can install the Discord integration from the store
and see deploy events land in a channel; teams not on GitHub can use
Nexul; agents connect from the desktop app with a connection token.

---

## Principles across all phases

- SQLite is the spine. No dual-backend abstraction, ever.
- MCP-first: every new capability is a use-case first, then both adapters
  (HTTP + MCP) get it by construction. Agents act as users; an execution
  started through MCP records the token's user with an `:mcp` provenance
  suffix (ADR 0049).
- One binary per role. No new infrastructure dependencies unless the domain
  requirements explicitly justify them.
- Integrations are **external services with scoped tokens**, never in-process
  plugins; they run on an isolated Docker network with only the gateway
  reachable.
- Event payloads are designed for publication — versioned, additive,
  secret-free.
- Open core: AGPL-3.0 outside `ee/`, a commercial license inside it.
