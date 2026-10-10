# Roadmap

> Directional, not dated. Open work lives in `.scratch/`, the decisions behind
> it in `docs/adr/`, and the vocabulary in `CONTEXT.md`.
> The public roadmap on nexul.io is `website/src/pages/roadmap.astro`; move
> an item there when it changes status here.

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
topology canvas, an MCP server over the whole domain, one install command
with a SQLite spine.

### The domain build-out

Every domain built out:

- **Identity & workspace** — users, owner/first-login wizards, private
  invitation links, projects + membership, categories + a swimlane board.
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

### Native install

Every part of Nexul runs as a native service under the OS service manager on
Linux, macOS, and Windows, with no containers of its own and one public port;
OpenObserve sits behind the server at `/openobserve/`. Runners and automations
hosts install on any machine from a one-line command, several to a machine,
each enrolling for its own credential with a one-time code. Remove revokes the
credential and uninstalls the host from its machine, and each automation is
placed on one named automations host.

### Domain-first setup

First run gives the instance its https domain before anything else, unlocked by
a one-time setup code the installer prints: a Cloudflare tunnel, a reverse
proxy that gets a Let's Encrypt certificate, or the owner's own HTTPS. The
GitHub App is then connected from the domain, where sign-in works. The server
installs on port 5123 so the proxy can own 80 and 443.

### Your settings, Devices, and connecting a phone

Settings split by who they affect: your own settings, and the instance's for
whoever holds their permissions, behind a gear in the sidebar footer, and the
workspace's configuration in the Workspace section. Sessions are stored per device, so Devices lists every signed-in
browser, desktop app and phone and signs any of them out on its next request.
Profile links several sign-in accounts to one user, and a phone connects by
scanning a single-use QR code.

### Team

One instance section lists everyone registered and what each person can
reach: open a person to change their role, overrides, or workspaces, or their
account's status, with every workspace change still needing `members:write`
in that workspace.

### Workspace links

Every page reached from a workspace's sidebar carries the workspace's slug in
its URL, so a shared link opens in the right workspace, and project prefixes
are unique per workspace: two workspaces can both have a `WEB` project, and a
ticket key resolves within its workspace (ADR 0089).

### Container logs

What a deployed service prints, read live from Docker through its runner and
never stored (ADR 0091): a Logs section on the stack page with a tab per
service and an errors filter, a link from each Services row, a click on a
service node in the topology canvas, the phone, and `stack_get` for agents.
Reading takes the `stacks:logs` permission, because output can carry secrets.

### Docs folders, pinning and order

A project's docs are grouped into folders, with a default Main folder; move,
rename and delete them from the Docs page, and agents manage them over MCP.
Pin a doc to keep it at the top of the list (kept per browser). The list is
ordered by creation date, with a toggle for last edited, and the Docs and
Memories list pane can be resized.

### A quieter inbox

A doc's edits notify its watchers instead of the whole workspace: its creator,
everyone who edited it, and anyone who chose to watch it from the doc's header
or over MCP; anyone can stop watching, and that sticks. Read notifications are
deleted after 90 days and every notification after 180.

### Automations per workspace

An automation belongs to one workspace and each workspace has its own
switches, and a switched-off automation receives no events. The Decisions
check is a play, switched on per workspace on its auto play.

### Chat and calls

Links in messages are clickable, and links to this instance open in the app as
pills. A call can share the screen full screen, and leaves on its own after
five minutes with nobody else in it. The composer is simpler: Enter sends,
and an emoji button opens a searchable picker. Anyone who reads a
conversation can react to a message with an emoji, from the browser or MCP,
and the phone app shows the reactions.
A computer's setup transcript reads like a play run.

### Project access and private channels

A workspace member, such as a client, can be held to the projects they are
given: their Every project row says None, and each project they may open
carries its own level per area. Every other project stays invisible to them,
names included, over the browser, MCP, search, and live updates, and taking
access away closes an open page on the spot. Channels can be private to the
people in them, and memories always belong to one project. ADRs 0097 to 0099.

### Instance templates

The Interview and mention chip templates, the built-in plays' instructions,
and the task, bug, and feature body templates each have an instance version,
edited in Settings → Templates, that every new workspace and project starts
from. An unedited workspace follows the instance's Interview and chip layout
live, each editor says whether it follows, matches, or differs, and any
template can be cloned to the instance, another workspace, or another project.
ADR 0103.

### A worktree per thread

Each person chooses where the T3 threads Nexul starts for them begin: the T3
project's folder, or a new git worktree off the branch that folder is on, so
plays and chats running at the same time never edit the same files. The choice
sits beside the model in pairing defaults, and a project link can override it.

### Interview sources

A project's interview can start from what it already has. Sources are pointed
at, never copied: paths in the checkout, docs, memories, pasted text, or the
project it replaces. What the team stands behind is drafted into answers the
person confirms; what was only done, such as a predecessor's code, is asked
about in the follow-ups. New material redrafts only what it touches, and
"Audit via AI" measures the old code against the new rules in a doc that
"To tickets via AI" turns into work. ADRs 0122 and 0123.

### Bots

Anything that posts to a Discord channel webhook posts into a Nexul
conversation by swapping the URL: Discord's payload, limits, and answers, with
embeds shown as cards in the message bubble. Bots are made, renamed,
regenerated, deleted, and restored from a conversation's settings or over MCP,
and a post renaming itself still says which bot sent it. ADRs 0129 and 0131.

### A new look

One page header with breadcrumbs, shared widths and one label style on every
page, then frosted panels over a soft light field with one ember accent,
gradient avatars and its own light mode (ADR 0133), replacing the earlier
monochrome identity. nexul.io and the phone app speak the same language: the
same tokens and Fraunces titles, the site with one header and real product
shots, the phone on solid surfaces with its own motion. Spec of record:
`practices/design-language.md`; tokens in `web/src/index.css`,
`website/src/styles/tokens.css` and `native/src/global.css`.

### Faster everywhere

Measured on heavy data and made faster on every surface. The web app starts
on less than half the JavaScript, and a board or chat update renders only
the rows that changed. The phone sends no requests while you scroll chat and
writes new messages straight from the socket. The server keeps its database
connections warm, answers permission checks once per request, and reads
agent lists one page at a time with totals that count every match. Live
updates refetch only what a change names.

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

In progress: **the phone app** in `native/`, an Android APK and an unsigned iPhone IPA for sideloading, with Inbox, Chat, the board,
docs, deploys and runners on a phone, signed in by scanning a QR code,
with push notifications and over-the-air updates. It is built and in device
testing ahead of its first release; the map is in `.scratch/native-app/`.

In progress: **ticket flow**. A ticket's thread moves into a resizable
pane beside the body on wide screens, and agents leave notes instead of
growing the body: one message in the thread with a markdown file, opened
and edited live in a dialog. Built and walked through on a real install;
the phone check and one editor fix remain, tracked in `.scratch/ticket-flow/`.
Docs, ticket bodies, memories, and notes also gained editable tables.

In progress: **T3 Code orchestrator V2**. A computer whose T3 Code moves to
T3's new orchestrator keeps working without re-pairing, and never goes back.
An agent that hands work to other agents replies with the finished result,
and each helper shows as a pill on the reply that opens its conversation.
OpenCode 2 runs as a provider inside T3 Code, so it needs no harness of its
own. Tracked in `.scratch/t3-orchestrator-v2/`.

In progress: **the interview as questions**. A project's interview stops
being a chat: the Interview page steps through the template's questions as
cards, then an agent reads the answers and the code, asks follow-ups about
the gaps, and writes the interview memory. Answers stay on the page, so a
re-run means changing what changed. The map is in `.scratch/interview-qna/`.

Planned: **clarifying a doc**. A client writes what they need in a doc;
"Clarify via AI" asks the gaps as rounds of question cards on the doc
page and in the phone app, the client answers, and once nothing is left
the answers are written into the doc. The map is in `.scratch/doc-clarify/`.

Planned: **projects before code**. The project wizard offers "No
repository yet" and "Attach without deploying" as plain choices instead of a
muted skip, both end on a Done screen that says what was made, and a project
with a repository waiting to deploy shows the way back to deploying it. The
map is in `.scratch/project-paths/`.

Planned: **auto plays**. A play starts by itself when something happens:
a ticket becomes unblocked, enters a stage, fails a test, or a doc settles
after edits. Each auto play is composed on the play's settings page from
dropdowns (when, if, priority, limits, whose computer), runs queue per
person with bugs first if you say so, and a daily cap per ticket stops
loops. Automations can start a play in code with `runPlay`. The map is in
`.scratch/auto-plays/`.

---

## Next — ecosystem & reach

Turn the integration contract into a platform and widen the surface area
without changing the core architecture.

- **Permissions per entity** — allow, fall back to the role, or deny each
  permission on a project, doc, play, computer or instance area for everyone,
  a role or one person, and see which rule decided any answer
  (`.scratch/permission-overrides/`, ADR 0148).
- **Add a computer in one command** — a personal runner installs, pairs and
  keeps T3 Code paired through its own connection, so pairing needs no
  Cloudflare tunnel or domain (`.scratch/personal-runners/`, ADR 0146).
- **DNS extensions** — TLS automation (Let's Encrypt) for direct-to-server
  paths, additional registrars behind the provider interface, managed subdomain
  routing for deployed services.
- **Desktop extensions** — auto-update (electron-updater), tray and menu
  niceties, diagnostics.
- **Semantic search / embeddings** — the event-driven indexing pipeline is the
  seam.
- **Doc comments** — deferred during the domain sessions to keep momentum; the
  access domain regains a `comment` action when this lands.
- **Runner labels and selectors** for routing a job to the right runners,
  beyond the machine a stack targets.
- **Ticket workflow depth** — blocked tickets, ticket-type body templates and
  required relations, comments and an activity timeline
  (`.scratch/ticket-workflow-depth/`); creating a branch from the ticket page
  (`.scratch/branch-from-ticket/`); one search box across every entity
  (`.scratch/global-search/`).
- **Chat and voice depth** — the message features still missing (read
  receipts, typing indicators), interactive
  in-chat approvals instead of today's auto-decline, per-user memories and
  multiple named agents, and on the voice side DM and group calls, `@Agent` in
  a call, and recordings for standups.
- **Integrations polish** — the expose-a-service flow and the voice channel
  UI, postponed until after the public release (`.scratch/integrations/`,
  tickets 04 and 11).
- **Smaller gaps**, too thin for a spec: offline doc edits do not survive a
  closed tab (the collaboration state is memory-only); a project's docs list
  has no manual ordering; MCP tools for automation versions and
  secrets; the GitHub webhook subscribes only `pull_request`; permission
  overwrites are wired for documents only, and the grid has no `comment`
  action; the profile display-name override is read only by the wizard; a
  503 carries no `Retry-After`; no
  default automation posts deploy results to a chat channel, and automations
  cannot yet auto-push from a connected git repository; a failed deploy is never
  rolled back automatically (rollback is the manual one-click action only — a
  best-effort automatic one driven by the reported `failed` status, not by a
  health probe, was specified and never built, and the post-start
  health-check probe was dropped with nothing replacing it); a requested
  review notifies nobody, though sign-in already guarantees every user a
  linked provider identity to notify.

**Exit criterion:** agents connect from the desktop app with a connection token.

Shelved for now, specs kept: the integration store (`.scratch/integration-store/`)
and a second git host, GitLab and Gitea (`.scratch/second-git-host/`), and
workspace-scoped runners and integrations
(`.scratch/workspace-scoped-automation-surfaces/`).

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
