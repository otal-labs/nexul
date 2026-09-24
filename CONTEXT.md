# Nexul

An open-source, self-hostable platform that unifies project management,
documentation, CI/CD, and deployment to your own servers — with a first-class
MCP server so LLM agents can drive the whole software development lifecycle.

This file is the project's **ubiquitous language**: the words we use and what
they mean. It is a glossary and nothing else — no specs, no implementation
detail. Decisions live in `docs/adr/`; specs and open work live in `.scratch/`.

## Language

### The product

**The Loop**:
The core product cycle: docs → tickets → code and PRs → deploy → index → repeat.
The thing that makes Nexul one product rather than three bolted together.

**Workspace**:
The top-level content and configuration boundary inside an instance. An
instance hosts multiple workspaces, each an isolation boundary for its
projects, docs, and configuration; members switch between them in the normal
UI via a picker. Sign-in itself stays instance-wide.
_Avoid_: Organization, tenant, team

**Instance**:
One self-hosted install of Nexul, owned by one person or team.
Single-tenant by design.

**Runner**:
The host binary on your server that executes builds and deploys, connected to
the server over a WebSocket.
_Avoid_: Agent, worker, executor

**Agent**:
The LLM participant in chat, mentioned as `@Agent`. Runs on the mentioning
user's own paired Harness and acts with only that user's permissions. Never a
Runner — a Runner executes builds; the Agent converses and drives the product
through MCP.

**Harness**:
The agent tool running on a user's paired computer that executes an Agent
turn (T3 Code today, others later). The server talks to every harness through
one interface, `harness.Client`, with one implementation per kind. An image
embedded in an inlined memory, a ticket body, or a doc body travels to the
harness as an attachment, capped at 10 MiB per image and 25 MiB per turn; an
oversized or non-image reference becomes an "attachment omitted" note.
_Avoid_: Backend (that is the Go server), Runtime, Driver

**Paired computer**:
A user's own machine running a harness, attached to their Nexul account.
It is reached through its computer tunnel, or, for a machine the server can
already reach, by URL. Owned by one user; nobody else can run on it.
_Avoid_: Device, host, runner (a runner builds and deploys)

**Computer tunnel**:
The outbound connection a paired computer keeps open to the instance's own
Cloudflare account, giving it a hostname made from the computer's name plus
eight random characters, which only the Nexul server may reach.
_Avoid_: Relay, VPN, proxy

**Setup confirmation**:
An agent's assessment, recorded through MCP and nowhere else, that a paired
computer is ready for agent work: once overall, and once per provider on
that computer. Unconfirmed means the provider cannot run agent work there.
Nothing ever withdraws it except an agent un-confirming it through MCP.
_Avoid_: Onboarded, verified, setup flag

**Memory**:
A note written for agents, not people: a title, a one-line when-to-use
phrase, and a rich-text body, belonging to the workspace or to one project.
A workspace memory reaches every turn, including a plain chat with no
ticket or doc; a project memory reaches only that project's turns. Every
Agent turn carries the index of the workspace's memories plus, when there
is one, the project's own; a play run inlines the ones the user picked.
Agents may write memories too. Every save appends a version with its
author, and any version can be reverted to. Cloned, never shared, to
another project or to a workspace. Plural in the UI: "memories".
_Avoid_: Doc (docs are for clients and requirements), skill, note

**Interview**:
The conversation that establishes a project's rules for agents: its stack,
paradigm, testing strategy, principles, and vocabulary, asked one question
at a time by the Interview play and answered by a person, with the agent
able to read the codebase for answers first. Its questions start from the
workspace's Interview template.
_Avoid_: Onboarding, questionnaire, setup

**Interview memory**:
The project memory an interview produces, written as rules and kept short,
and included in full in every agent turn in that project.
_Avoid_: Practices doc, guidelines, rules file

**Decisions log**:
A project memory recording only the tickets that changed how the project
works, three lines at most per entry, with reversed decisions marked
superseded so it reads as what is true now. Pulled from the memory index
when relevant, never sent every turn.
_Avoid_: Changelog, history, release notes

**Play**:
A pre-configured Agent turn a user fires from a ticket page or a doc page
with one button ("Fix with AI", "To tickets via AI"). Defined per workspace
with a label, a type (ticket or doc), a one-line description, base
instructions, an enabled switch, and an excluded-projects list; a ticket
play also names the one stage it shows in. No default memories live on the
definition — the run dialog picks those per run. Runs on the clicking
user's own paired harness and posts into the target's thread; the column
the ticket moves to on success is chosen at run time. Every workspace,
new or existing, is seeded with the same pair, "Fix with AI" (ticket,
progress stage) and "To tickets via AI" (doc), as ordinary plays a member
may edit or delete. Seen and fired with `plays:run`, managed with
`plays:read`, `plays:write`, `plays:delete`. A named user can be excluded
from one play: a permission overwrite denying that user `plays:run` on the
play, set from the play's own settings page.
_Avoid_: Agent action, button, automation (that is event-driven code)

**Trail**:
What one press of a play leaves behind: who started it, on which ticket
or doc, with which memories and instructions, every step the Agent took,
and how it ended. A step is structured (tool call, tool result, question,
or assistant text, with the tool name, an argument preview, the raw detail,
and its time), never a bare line, so the trail reads as a transcript. The
transcript is the Agent's turn as a conversation: the starter's "Started
<play>" message, one collapsible "Worked for" group per turn holding what
the Agent said between actions and each action as a row (a command names
its command, a file change its path), the question card and the answer
where they happened, the final reply as prose, and the runner's own notes
(a skipped move, a stop) as muted lines.
Persisted, never ephemeral; the "Trail" section on a
ticket or doc lists them, and the target's thread shows the same turn
groups above the Agent's reply, question, or closing note, so the run reads
the same way in the conversation it landed in. States: `starting` at the press, `running` once
the harness accepts, `waiting` while the Agent's question to the starter is
unanswered (the ticket stays put, the silence clock pauses, answering runs
on again in the same trail and session), then one of `done`, `failed`, or
`interrupted`. Stop (the starter or a `plays:write` holder) interrupts it,
from `waiting` too; fifteen minutes of harness silence fails it. A ticket
move it makes carries actor kind `play`.
Also records the computer, provider, and model the run used, whether the
starter picked them in the run dialog or they came from the project link
or the starter's own pairing defaults (ADR 0058).
_Avoid_: Run history, log, execution

**Doc thread**:
A doc's one conversation, the counterpart of a ticket thread: a real
conversation people reply in, and where a doc play's run lands. Gated by
`docs:thread`, so a reader of the doc need not see the work behind it.

**Voice channel**:
Chat's fifth conversation kind — a Discord-style voice room with screen
share and camera, backed by your own LiveKit server. Carries its own text
chat like any channel.

**Bot**:
A named poster inside one conversation that outside systems drive through
a webhook URL: the URL is the credential and the binding, the payload is
Discord's webhook JSON, and the bot is the message's author with its own
name and avatar. Not a login, not an Integration, and never the Agent.
Gated by `botwebhook:read`, `botwebhook:write`, `botwebhook:delete`;
posting through the URL needs no permission.
_Avoid_: Integration (a separate service with a scoped token), Agent, app

**Occupancy**:
Who's currently in a voice channel, shown live in the channel list without
joining.

**Topology**:
The map of your infrastructure, drawn as a canvas. The canvas JSON *is* the
infra model — not a picture of it.
_Avoid_: Diagram, graph, architecture view

**Base service**:
A deploy service definition that owns branch deploy rules. The thing being
deployed; a branch deployment is always a clone *of* a base service, never a
base service itself.

**Branch deploy rule**:
A base service's mapping from a branch pattern (`main`, or a single trailing
wildcard like `feature/*`) to a docker network, optional hostname template,
derived-service name suffix, and optional overrides of the base's settings
for that branch. Evaluated on every push; there is no
separate "environment" concept — `dev` → QA and `main` → prod are just rules.

**Branch deployment**:
The running instance a branch deploy rule maintains for one branch: either
the base service redeployed in place (an exact rule with no name suffix), or
a derived clone service record.

**Preview deployment**:
A branch deployment spawned by a *wildcard* branch deploy rule — one clone
per matching branch, torn down when the branch is deleted.

**Stack**:
One repository's worth of deployable containers: a compose file, or a
Dockerfile as a stack of one. The stack is what gets deployed, rolled back,
and torn down; the services inside it are observed, not deployed on their
own. Decided 2026-09-08 for the project wizard.
_Avoid_: Compose project, resource, application

**Stack slug**:
The DNS- and Docker-safe token derived once from a stack's name at creation,
unique per machine. It is the compose project name, a run stack's container
name, and the checkout directory under the stack root. Never generated or
suffixed: a duplicate is rejected, because every routing mechanism addresses
the container by this name.

**Observation report**:
The list of containers a runner sends back with a terminal deploy result,
one entry per container the stack started: name, image, status, networks with
addresses, published ports. The only source of a service's observed facts;
the compose file remains the only source of its declared ones.
_Avoid_: Status report, inspect result

**Service** (deploy):
One running container Nexul keeps a record of: name, image, networks,
address, status. A stack deploy creates or updates one service per
container it started; an import adopts one per container found running.
_Avoid_: Container (the Docker thing a service is a record of)

**Stack root**:
The directory on a machine under which every stack's persistent checkout
lives (`/data/nexul` by default, `stacks/<slug>/repo` beneath it).
Configurable per machine; a stack's location is fixed once it has deployed.

**Unmanaged stack**:
A stack adopted from what was already running on a machine (a manual
`docker run`, a stack some other tool deployed) with no repository attached. Its
containers can be seen, wired, and exposed, but it cannot be deployed until
a repository is attached, which makes it managed.

**Instance upgrade**:
Moving the running compose stack to the newest release of its channel from
the UI or the `instance_upgrade` MCP tool. The server records it, the bundled
`instance` runner starts the upgrade helper, and the server resolves the
record when it boots on the target version.
_Avoid_: Update (that word is the runner's own binary swap), deploy

**Upgrade helper**:
The one-shot container named `nexul-upgrade` that pulls the release's images
and restarts the compose project. It is not part of the project, so it
survives the restart that recreates the runner which started it. Its logs
are the trace of a failed upgrade.
_Avoid_: Updater, supervisor

**Machine**:
A server a runner runs on. Runners belong to a machine; a service targets a
machine and any of its runners may take the job, so the number of runners
per machine is the scaling knob. Replaces "target is the runner" from
2026-09-07 once built.
_Avoid_: Host, node, server (in the UI)

**Wizard**:
A guided multi-step flow that ends in something working, reached under
`/wizard/<context>/<step-owner>` (for example `/wizard/onboarding/owner`,
`/wizard/project/service`). Dialogs are not wizards.

**Automation**:
First-party event-driven code: when an event happens, a function runs.
Written in TS/JS against the SDK — **Default automations** ship with the
instance, **Custom automations** are owner-written. An automation connects
out to the instance and acts back through the API with its own scoped token;
Nexul never calls in to it.
_Avoid_: Rule, workflow (the v1 rule engine is gone)

**Automations host**:
The small container bundled with the instance that runs automations — all
Defaults, plus small Custom ones. Bigger Custom automations run wherever the
owner deploys them.

**Connector**:
A third-party tool the instance holds a credential for and calls out to:
GitHub, Cloudflare, LiveKit. One credential per tool per instance, encrypted
at rest. Nexul acting as itself against someone else's API; an
Integration is the opposite direction.
_Avoid_: Integration (the other direction), OAuth app, provider

**Pending version**:
An automation code version that has been pushed or seeded but is not active.
Merging it activates it and respawns the worker. Shipped upgrades land here
too, so an upgrade never silently changes a default's behaviour.
_Avoid_: Draft, staged

**Self-read**:
An automation token reading its own automation record. Always permitted
regardless of scopes; that is how the host loads its state and active code.
Reading yourself is identity, not a grant.

**Integration**:
An external service that listens to signed events and calls the scoped API. A
separate service, never an in-process plugin. Third-party by positioning:
first-party event-driven code is an **Automation**.

**Trust tier**:
How much an integration is vouched for — `verified` or `community`.

**Dial-in**:
An automation's outbound connection to the instance: it connects out,
announces the subscriptions its code declares, and receives events down that
connection. The opposite of the signed-webhook path integrations use.

**Stage**:
One of the five fixed, ordered buckets every status column declares:
`backlog`, `progress`, `review`, `testing`, `done`. Column names are the
owner's; stages are the product's, so rules read the stage and never the
name. Only `done` is terminal.
_Avoid_: Kind, phase, state

**Finished** (ticket):
A ticket with at least one linked PR, none still open, and at least one
merged. PRs closed unmerged are ignored. Publishes `ticket.finished`.
_Avoid_: Done (a status column's name), closed, completed

**Developer / Tester / Reporter**:
The three people on a ticket. The developer builds it and the tester checks
it, one person each and both optional. The reporter filed it, is set once,
and shows as Nexul on behalf of a person when an agent or automation filed
it.
_Avoid_: Assignee, owner, creator

**Found in**:
A bug's link to the ticket it was found in, carried by every bug unless its
reporter marked the origin unknown. A done ticket is never reopened; a bug
found after done is a new ticket found in it.
_Avoid_: Regression of, caused by, parent

**Blocked by**:
A ticket's link to another ticket that must reach done first. It shows on
the card and warns before a play runs, but never stops a card moving.
_Avoid_: Depends on, dependency, blocker stage

### Identity and access

**Permission**:
One capability, written `<domain>:<action>` where the action is `read`,
`write`, or `delete` (`docs:write`, `members:delete`), or a verb the domain
declares for an act that is neither (`plays:run`, `memories:clone`,
`docs:thread`). One vocabulary for every actor: a role, a scoped token, and
the agent are checked against the same values.
_Avoid_: Right, privilege, capability, ACL entry

**Permission overwrite**:
A per-user allow/deny set layered on top of their role, either workspace-wide
or on one resource. Most-specific wins; deny beats allow inside a layer. The
workspace Owner bypasses all of it.
_Avoid_: Grant, share, ACL

**Invitation**:
A single-use bearer link that admits one person to the instance and grants a
chosen Role plus optional Permission overwrites in one or more Workspaces. The
link expires after one or seven days and is not bound to a provider identity.
_Avoid_: Allowlist entry, invite code, join link

**Account status**:
Whether a registered User may authenticate to the Instance: active, disabled,
or removed. Workspace membership and Roles remain separate.
_Avoid_: Allowlist status, membership status

**Personal access token**:
A long-lived, revocable credential (`dep_`) carrying exactly one user's own
permissions. What an agent authenticates its MCP connection with.

**Scoped token**:
The credential an integration (`int_`) or an automation (`dat_`) acts with,
minted with a chosen subset of permissions and revocable on its own. A scoped
token granted `X:write` also receives `X:read` at mint.

**Connection token**:
A signed JWT carrying server information only (instance URL, MCP endpoint,
basic settings) so a standalone client can be pointed at an instance. No
identity, no credentials, therefore not secret.

### The domains

A **domain** is a bounded context under `internal/<domain>/`. It owns its model,
storage, use-cases, events, and adapters, and never imports another domain's
internals.

**Access** — what an actor may do. Permissions and access control. A
permission is `<domain>:<read|write|delete>` (`docs:write`,
`members:delete`), or a verb a domain declares beside those three
(`plays:run`, `memories:clone`, `docs:thread`), one vocabulary shared by
roles, token scopes, and the agent. Distinct from auth.

**Auth** — who a user is. Identity and sessions, via an owner-configured OAuth provider.
Distinct from access.

**Automations** — event-driven code: Default and Custom automations run
functions when events happen, acting through the scoped API.

**Chat** — conversations inside a workspace: channels, direct messages,
threads, and ticket threads, with the mentionable Agent and its memories.

**Code review** — a mirror of the provider's review state, one record per PR.
Nexul does not host review threads.

**Deploy** — stacks, the containers they run, and the act of shipping them to
a machine through a runner.

**DNS** — DNS records and domain management. Cloudflare is the first provider
and is special: it gives both records and tunnels. A **gateway** is a
Nexul-deployed service (tunnel or reverse proxy) giving one docker
network internet reachability via hostnames — the canonical term for what's
elsewhere been called an "exit node" or "entry path"; an **exposure** routes
one hostname through a gateway to a container.

**Docs** — documentation as the source of truth, with `@` cross-references to
tickets and other docs.

**Git provider** — the abstraction over GitHub and future providers: branches,
PRs, webhooks.

**Integrations** — the platform for external services: scoped tokens, signed
webhooks, the store.

**MCP** — the Model Context Protocol adapter, one tool per use-case, so LLM
agents can drive the product.

**Memories** — the Memory entity: its table, page, permission verbs, and MCP
tools. Never shares a list, a page, or search with Docs.

**Runner** — runner lifecycle, the WebSocket protocol, queueing and dispatch.

**Search** — FTS5 full-text search over docs and tickets. Not a package: each
domain registers its own index and query.

**Tickets** — work items on a Kanban board, linked to docs, branches, and PRs.

**Topology** — the canvas model: nodes, edges, live status.

**Voice** — voice channel LiveKit interaction: join tokens and live
occupancy, over a bring-your-own LiveKit server.

**Workspace** — the workspace boundary itself, its members, its settings, and
notifications. Notifications are a capability here rather than a domain of
their own: every domain publishes events, and the workspace turns the ones a
member cares about into an inbox.

### How we talk about the code

**Use-case layer**:
The single source of business behaviour. Both adapters call it; neither
duplicates it.

**Adapter**:
A thin entry point over the use-case layer. There are two: the HTTP/JSON
gateway for the browser, and the MCP server for LLMs.
_Avoid_: Controller, handler (a handler is one function *inside* an adapter)

**Seam**:
A boundary designed for future splitting. The EventBus is *the* seam — the one
place a domain could later become its own process.

**Event catalog**:
The list of every event topic with its producers and consumers, assembled
from each domain's own `Topics()` by `internal/eventcatalog`. Additive-only.

**Outbox**:
Events written in the same transaction as the domain change, then relayed to
the bus — so an event can never be lost or published for a change that rolled
back.

**Composition root**:
A `cmd/main.go`. The only place concrete implementations are wired to
interfaces.

**MCP**:
Model Context Protocol — the standard LLM agents use to call tools. The
"MCP-first" thesis: the MCP server is a first-class adapter, not an afterthought.

### How we talk about the frontend

**The Mono Console**:
The web app's visual spec of record — strictly monochrome, dark-first, no
accent color; color is reserved for badge/status signal only. Tokens in
`web/src/index.css`, spec in the [coding standards](https://nexul.io/docs/contributing/coding-standards/#design-language--the-mono-console).

**Page → Feed → Section → Card**:
The required component hierarchy. A Page composes Feeds; a Feed lists entities;
a Section is a titled group; a Card/Row/Item is one entity.

**Defensive ordering**:
Negative checks first — loading, error, empty — each with an early return,
before the happy path.

**The Frontend Commandments**:
F1–F7 in the [coding standards](https://nexul.io/docs/contributing/coding-standards/#react-the-frontend-commandments). Hard rules for anything in `web/`, not
suggestions.

**Ticker**:
A form that hands credentials or config to a third party verifies them first
as a list of named checks. Each row names one thing the provider must
accept, spins while its request runs, then ticks or crosses with the
provider's own reason, so a failure says exactly which permission or value is
wrong. Continue/Confirm/Set up only unlocks once every row is green.
_Avoid_: Checklist, health check, validation list
