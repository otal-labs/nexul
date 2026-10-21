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

### Plays and guided agent setup

A play is a pre-configured Agent turn fired from a ticket or a doc page, such
as Fix with AI and To tickets via AI, with memories the person selects, doc
threads and persisted trails that read as a transcript, and a harness and
model picker in the run dialog. A paired computer is set up per provider by a
wizard that installs a starting skill set and
confirms through the MCP server; an agent run is refused on a computer or
provider that is not set up, with a link to the wizard. Tickets gain a
developer, a tester and a reporter, found-in and blocked-by links, body
templates per type, a Test this panel with Pass and Fail, and per-branch
deploy overrides. ADRs 0055 to 0057 and 0062 to 0065.

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

### The phone app

An Android APK and an unsigned iPhone IPA for sideloading, built on GitHub and
updated over the air from a self-hosted update server. Inbox, Chat, the board,
docs, deploys and runners, container logs and Your settings on a phone, signed
in by scanning a QR code, with push notifications for the inbox.

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

### Ticket notes

A ticket's thread sits in a resizable pane beside its body, and an agent adds
to the ticket as a note: one message in the thread with a markdown file, opened
and edited live in a dialog, instead of growing the body. Docs, ticket bodies,
memories and notes gained editable tables.

### The interview as questions

A project's interview is a set of question cards on the Interview page, with
no agent involved. Once they are answered, a play run reads the answers and the
code, asks follow-ups about the gaps a round at a time, and writes the interview
memory; changing an answer regenerates it.

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

### Auto plays

A play starts by itself when something happens: a ticket becomes unblocked,
enters a stage, is created, gets a developer or tester, or fails a test, or a
doc is created or settles after edits. Each auto play is set up on the play's
Auto plays tab from dropdowns (when, if, priority, limits, whose computer).
Runs wait in each person's queue, highest priority first, are checked again
before they start, and a daily cap per ticket pauses loops until someone
resumes. A ticket shows what is queued, paused, skipped or didn't run. The
decisions check is now an ordinary play with its own auto play, and
automations start a play by name with `runPlay`.

### A welcome wizard for every role

A newcomer's first sign-in opens a short wizard with their profile, then just
the steps their roles ask for: connect GitHub, add their computer, see their
projects, add a deploy server, get the phone app. Every step can be skipped,
and what is left waits under Welcome in the account menu with a red dot until
it is done or skipped; giving a step again through a role brings it back.
New workspaces start with five template roles (Client, Junior Developer,
Senior Developer, DevOps, Administrator) that the instance edits and adds to
in Settings → Templates. The owner finishes setup without a computer, and
Team shows how far each person got and who each invitation let in. ADR 0149.

### Clarifying a doc

A client writes what they need in a doc. "Clarify via AI" asks the gaps as
rounds of question cards on the doc page and in the phone app, and anyone who
can edit that doc, by role or by a rule on the doc alone, answers, skips, or
adds a question of their own. Once a round finds no gaps, the answers are
written into the doc, the developer closes it, and "To tickets via AI" cuts it
into work. The people answering see only the questions, never the agent, and
the doc is locked only while a round runs. ADR 0121.

### Machine permissions

Each machine carries rules of its own, for a role of any workspace or one
person: See, Deploy to, Manage and Delete, each Allow, Fallback or Deny. A
machine is opt-in, as a project is: someone it doesn't list can't find it in
the Runners page, a picker, the API, MCP or the live updates, and typing its
name into a stack tool gets nothing. Deploy to is checked when a stack is
aimed at a machine and on every deploy a person starts; a deploy the server
starts after a push still runs. Adding a machine is its own permission, and
whoever adds one gets all four actions on it. Only the Owner of the default
workspace holds the instance by right; the Owner of another workspace is full
Owner inside it and reaches a machine or an instance area through a rule. On
upgrade every machine starts with only the instance Owner on it. The Runners
page has a People on this machine row for the instance Owner and Manage
holders. ADR 0151.

### Screens that know your role

A button a person can't use is hidden. A role with Show locked actions on
sees it locked instead, dimmed with the reason, "Your role can't deploy" or
the sentence of the rule that refused it on that one item; Junior Developer,
Senior Developer, DevOps and Administrator ship with it on and Client off. The
web and the phone draw every write the same way, and a test fails when a write
declares a permission its route doesn't take. The things you could make but not
undo have a way back: delete a machine or a tunnel once nothing uses it, with
a refusal that names what does; edit a stack's environment after launch with
values masked for readers; retry a failed deploy on the commit it built;
unlink a ticket's branch or pull request; delete a ticket; archive and restore
docs and workspaces; revoke an unused runner code; mark a notification unread.
ADRs 0152 and 0153.

### Invitations that explain themselves

Every role carries a one-line description, written in the role editor and
shown in the invite dialog and on the invite page. Before sign-in the page
names the instance, each workspace and the roles it gives with what they mean,
and no project. After sign-in each workspace lists the projects its roles reach
now, with Decline beside Accept. A link that expired or was deleted says only
that, and someone who already has an account is turned away without using up
the link meant for someone new. Nexul sends no email: the admin copies the
link and sends it.

### Projects before code

A project can start with no code: the wizard's Repository step offers "No
repository yet" for a project that is only docs and tickets for now, and
"Attach without deploying" for a repository that isn't ready to ship. Both end
on a Done step that says what was made, keeps the interview offer, and points
to Docs, Board and Invite people. A repository no service builds from shows in
the project's Services as "Attached, not deployed yet" with Deploy this
repository, and New service goes straight to it when it is the only one.

### Debugging an instance without its box

Anyone given the Logs area reads the server's, runners' and the automations
host's logs through Nexul, an agent included, with no shell and no OpenObserve
password. A personal runner keeps its own log, read live by its owner and by
anyone its share allows to read logs. A service that crashes outside a deploy
opens an outage: everyone watching the stack is told, the stack can open a
ticket, and stack plays such as "Investigate a crash" start on a computer
someone picked. Each stack counts its requests per minute, from its proxy
gateway or Cloudflare's analytics, and a rule on its 5xx share does what a
crash does when it trips. A bug in Nexul itself becomes a diagnosed issue
through a triage skill the person's own agent runs, posted only after they
approve the draft.

### View as

A member holding the View Nexul as another member permission opens a person's
tab on Team and presses View as to see Nexul exactly as they do in that
workspace, read only, to check whether their roles give too much or too
little. A bar across the top names who is being viewed and stays until Stop
viewing. Changes, live updates, the person's inbox, direct messages, devices
and computers are off, an Owner can't be viewed, and neither can someone
holding a permission the viewer lacks. The view ends by itself, with the
reason, if the person's roles change under it.

### Clone onto a computer

When a computer has no T3 project for one of a project's repositories, the T3
project picker under Settings, Computers, Projects offers Clone for it. T3
Code clones the repository next to the computer's other projects, or into
~/Code, and the picker selects the new project, so a computer that runs T3
Code from the command line needs no trip to its terminal. Agents do the same
with computer_project_create. When the project has no repository, or the
computer holds every one, Create empty project makes a new folder the same way,
which also gives a computer with no T3 project the one setup needs.

### A computer with no systemd

A container or any Linux host that isn't running systemd can be added as a
computer. The install detects it (or takes --foreground), needs no sudo, and
stays running as `nexul run computer`, a built-in supervisor that runs the
Nexul app and T3 Code, starts either one again 5 seconds after it exits, and
removes the install when the computer is removed from Nexul. It prints the
line that starts it again after a reboot.

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

In progress: **T3 Code orchestrator V2**. A computer whose T3 Code moves to
T3's new orchestrator keeps working without re-pairing, and never goes back.
An agent that hands work to other agents replies with the finished result,
and each helper shows as a pill on the reply that opens its conversation.
OpenCode 2 runs as a provider inside T3 Code, so it needs no harness of its
own. Tracked in `.scratch/t3-orchestrator-v2/`.

In progress: **adding a computer in one command**. Your settings →
Computers → **Add a computer** gives one command to run on the computer; it
installs the Nexul app there (a personal runner) and T3 Code when it is
missing, and the dialog shows the computer connect, T3 Code found and the
pairing done through the app's own connection, then setup. No Cloudflare
tunnel or domain is involved, and the app keeps T3 Code paired and running.
Linux, macOS and Windows work today, and it is the only way to pair: a
computer paired through a tunnel or by URL runs nothing until it moves to the
app from its row, keeping its links, setup and history. Setup gives each
provider its own switch and run, and sets them up side by side; a provider
switched off is left out of plays and chat. Tracked in
`.scratch/personal-runners/` (ADR 0146).


---

## Next — ecosystem & reach

Turn the integration contract into a platform and widen the surface area
without changing the core architecture.

- **Permissions on roles** — people hold several roles, projects show only to
  the roles and people listed in them, and each project, doc, play, channel,
  computer or instance area can allow, fall back or deny any permission for a
  role or one person, with the rule that decided every answer shown
  (shipped, ADR 0148; remaining gaps in `.scratch/permission-followups/`).
- **DNS extensions** — TLS automation (Let's Encrypt) for direct-to-server
  paths, additional registrars behind the provider interface, managed subdomain
  routing for deployed services.
- **Desktop extensions** — auto-update (electron-updater), tray and menu
  niceties, diagnostics.
- **Nexul's own error rate** — an instance rule on the server's own 5xx share,
  telling the instance Owner, the way a stack's error-rate rule does.
- **Semantic search / embeddings** — the event-driven indexing pipeline is the
  seam.
- **Doc comments** — deferred during the domain sessions to keep momentum; the
  access domain regains a `comment` action when this lands.
- **Runner labels and selectors** for routing a job to the right runners,
  beyond the machine a stack targets.
- **Ticket workflow depth** — an activity timeline and custom fields on ticket
  types (`.scratch/ticket-workflow-depth/`); creating a branch from the ticket page
  (`.scratch/branch-from-ticket/`); one search box across every entity
  (`.scratch/global-search/`).
- **Chat and voice depth** — the message features still missing (read
  receipts, typing indicators), interactive
  in-chat approvals instead of today's auto-decline, per-user memories and
  multiple named agents, and on the voice side DM and group calls, `@Agent` in
  a call, and recordings for standups.
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
workspace-scoped integrations
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
