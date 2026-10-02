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
_Avoid_: Organization, tenant, team (the Team is the instance's people)

**Workspace slug**:
The workspace's name in every URL of its pages (`/otal/board`): lowercase
letters and digits joined by dashes, unique on the instance, derived from the
name at creation and kept through a rename unless changed on purpose, in
Configuration → General or by `workspace_update`; old links then stop working
and are not redirected. The URL decides which workspace is on screen.
_Avoid_: Workspace key, handle, subdomain

**Project**:
The grouping inside a workspace that tickets, docs, project memories,
repositories, and stacks belong to. Only the project wizard makes one
(`project_create` over MCP runs the same use-case), and it arrives with its
status columns, ticket types, and a starter memory. Nothing seeds one: a new
workspace, the owner's first included, has none until the wizard runs.
_Avoid_: Board, app, default project

**Prefix**:
A project's 2-5 character tag, a letter then letters or digits, set once and
unique within its workspace; it starts the project's ticket keys.
_Avoid_: Project code, short name

**Ticket key**:
A ticket's human name, its project's prefix and its number (`WEB-12`). Unique
only within a workspace, so a key lookup names the workspace, and a key in two
of someone's workspaces is refused rather than guessed. The ticket's id stays
the one reference that needs nothing else.
_Avoid_: Ticket id (that is the UUID), ticket number (only the second half)

**Instance**:
One self-hosted install of Nexul, owned by one person or group.
Single-tenant by design.

**Runner**:
A named service on a machine that executes builds and deploys, connected to
the server over a WebSocket with its own host credential. A machine can run
several; the one `nexul install` puts on the instance's own server is named
`instance`.
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
embedded in a ticket body, a doc body, or an always-included memory an
`@Agent` turn inlines travels to the harness as an attachment, capped at
10 MiB per image and 25 MiB per turn; an oversized or non-image reference
becomes an "attachment omitted" note.
_Avoid_: Backend (that is the Go server), Runtime, Driver

**Model options**:
The per-model settings a turn runs with, such as reasoning level, context
window, and fast mode: whatever the harness lists for that model, with its
defaults, read live from the harness and never kept as a list in Nexul. They
are picked beside a model and stored with it (pairing defaults, a project link,
a play's trail, a computer's setup choices); an option left unset runs on the
harness default, and picking another model starts it on its own defaults.
_Avoid_: Traits, knobs, model settings

**Paired computer**:
A user's own machine running a harness, attached to their Nexul account.
It is reached through its computer tunnel, or, for a machine the server can
already reach, by URL. Owned by one user; nobody else can run on it.
_Avoid_: Device, host, runner (a runner builds and deploys)

**Project link**:
One person's choice of paired computer, T3 project, provider, model, and
model options for one project they can open, set in Your settings → T3
pairing → Projects. Each person has their own; nobody else's turns use it.
A person with no link for a project runs there on their own pairing
defaults (ADR 0102).
_Avoid_: Project pairing, shared link

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
Each provider's confirmation also records the version of Nexul's skills it
found; an older one shows as skills out of date, a signal, never a block. A
skills update, one agent turn that rewrites the skills on the computer, records
the new version for every confirmed provider there and confirms nothing.
_Avoid_: Onboarded, verified, setup flag

**Memory**:
A note written for agents, not people: a title, a one-line when-to-use
phrase, and a rich-text body, belonging to exactly one project. It reaches
only that project's turns: an Agent turn on a ticket, doc, or interview
carries the index of its project's memories, a play run names the ones
the user picked for the agent to read first, and a plain chat with no
ticket or doc carries none.
Agents may write memories too. Every save appends a version with its
author, and any version can be reverted to. Cloned, never shared, to
another project. Plural in the UI: "memories".
_Avoid_: Doc (docs are for clients and requirements), skill, note

**Interview**:
The conversation that establishes a project's rules for agents: its stack,
paradigm, testing strategy, principles, and vocabulary, asked one question
at a time by the Interview play and answered by a person, with the agent
able to read the codebase for answers first. Its questions start from the
workspace's Interview template, which is the instance template until the
workspace edits its own. It happens in the project's interview
thread, a conversation of its own shown on the project's Interview page.
Re-running it amends the interview memory rather than starting over.
_Avoid_: Onboarding, questionnaire, setup

**Interview memory**:
The project memory an interview produces, written as rules and kept short,
and included in full in every `@Agent` turn in that project; a play run
names it for the agent to read first.
_Avoid_: Practices doc, guidelines, rules file

**Decisions log**:
A project memory recording only the tickets that changed how the project
works, three lines at most per entry, with reversed decisions marked
superseded so it reads as what is true now. Pulled from the memory index
when relevant, never sent every turn.
_Avoid_: Changelog, history, release notes

**Decisions check**:
The built-in play that fires by itself, once, when a ticket enters a
done-stage column: on the paired computer of the person who moved the card,
or the ticket's developer's when an automation moved it. It adds a
decisions-log entry, marks an older one superseded, or leaves the log alone.
A check that cannot start stays on the ticket as "Decisions check didn't run"
with a way to run it again. Each workspace switches it on or off among the
default automations; it starts off.
_Avoid_: Closing summary, retrospective, done hook

**Play**:
A pre-configured Agent turn a user fires from a ticket page, a doc page, or
a project's Interview page with one button ("Fix with AI", "To tickets via
AI", "Run the interview"). Defined per workspace with a label, a type
(ticket, doc, or interview), a one-line description, base
instructions, an enabled switch, and an excluded-projects list; a ticket
play also names the one stage it shows in. No default memories live on the
definition — the run dialog picks those per run. Runs on the clicking
user's own paired harness and posts into the target's thread; the column
the ticket moves to on success is chosen at run time. Every workspace,
new or existing, is seeded with the same four, "Fix with AI" (ticket,
progress stage), "To tickets via AI" (doc), "Interview" (interview), and
"Test with AI" (ticket, testing stage), as ordinary plays a member
may edit or delete; each keeps a built-in key through renames, and a new
workspace's start from the instance template of their instructions. Seen and fired with `plays:run`, managed with
`plays:read`, `plays:write`, `plays:delete`. A named user can be excluded
from one play: a permission overwrite denying that user `plays:run` on the
play, set from the play's own settings page.
_Avoid_: Agent action, button, automation (that is event-driven code)

**Trail**:
What one press of a play leaves behind: who started it, on which ticket,
doc, or project interview, with which memories and instructions, every step the Agent took,
and how it ended. A step is structured (tool call, tool result, question,
or assistant text, with the tool name, an argument preview, the raw detail,
and its time), never a bare line, so the trail reads as a transcript. The
transcript is the Agent's turn as a conversation: the starter's "Started
<play>" message, one collapsible "Worked for" group per turn holding what
the Agent said between actions and each action as a row (a command names
its command, a file change its path, an MCP call its server and tool), the question card and the answer
where they happened, the final reply as prose, and the notes about the run
itself (a skipped move, a stop, a reconnect) as muted lines.
Persisted, never ephemeral; the "Trail" section on a
ticket, doc, or Interview page lists them, and the target's thread shows the same turn
groups above the Agent's reply, question, or closing note, so the run reads
the same way in the conversation it landed in. States: `starting` at the press, `running` once
the harness accepts, `waiting` while the Agent's question to the starter is
unanswered (the ticket stays put, the silence clock pauses, answering runs
on again in the same trail and session), then one of `done`, `failed`, or
`interrupted`. Stop (the starter or a `plays:write` holder) interrupts it,
from `waiting` too; fifteen minutes of harness silence fails it. A dropped
harness connection does not: the turn is redialed and resumed where it left
off, and fails only if it stays down for five minutes. A ticket
move it makes carries actor kind `play`.
Also records the computer, provider, and model the run used, whether the
starter picked them in the run dialog or they came from the starter's own
project link or pairing defaults (ADR 0058).
_Avoid_: Run history, log, execution

**Doc thread**:
A doc's one conversation, the counterpart of a ticket thread: a real
conversation people reply in, and where a doc play's run lands. Gated by
`docs:thread`, so a reader of the doc need not see the work behind it.

**Note**:
What an agent leaves on a ticket instead of growing its body: one Agent
message in the ticket's thread together with the markdown file it carries,
which anyone who may edit the ticket opens and edits live, the way a doc is
edited. Message and file are one thing: deleting the message deletes the
file. The body stays the ticket's spec and changes only when a person asks.
_Avoid_: Comment, addendum, memory (a memory is written for agents)

**Locked doc**:
A doc set read-only for everyone as a guard against accidental edits: its
title and body refuse every change, from people and agents alike, until it
is unlocked. Locking and unlocking take `docs:lock`, a permission of its own
apart from editing. Starting a doc play locks its doc, whatever the starter
holds, and the doc stays locked after the run ends. Archiving, cloning, and
deleting still work, and a clone starts unlocked.
_Avoid_: Frozen, protected, read-only doc

**Watcher**:
A person who gets a doc's change notifications. A doc's creator becomes one by
creating it, and anyone who saves an edit to its title or body by saving it,
through the browser, MCP, or a play acting for them. Anyone who can open the
doc may start or stop watching it, for themselves only. Stopping sticks: their
own later edits do not make them a watcher again until they choose to watch.
An edit tells every watcher but its editor; a mention tells the person
mentioned whether or not they watch, and does not make them a watcher.
_Avoid_: Follower, subscriber

**Folder**:
A group of a project's docs, one level deep: every doc lives in exactly one
folder of its project. Each project has a default folder, Main, where a new
doc lands unless it was started in another folder; it can be renamed but
never deleted, and deleting any other folder moves its docs there, never
deleting one. A doc copied into another project lands in that project's
default folder. Made, renamed, and deleted, and docs moved between them, with
`docs:write`; a reader sees only the folders holding a doc they can open.
_Avoid_: Category (that groups tickets on the board), section (a heading
inside a doc), collection (the retired workspace-wide doc grouping)

**Channel**:
A workspace conversation its readers post in: a text channel, named with a
leading `#`, or a voice channel. Every member reads a public channel except a
Restricted member; a Private channel only its members read. Created and renamed with
`channels:write`, deleted together with every message in it with
`channels:delete`. Each workspace has one `#general`, made with the workspace,
which can be renamed but never deleted.
_Avoid_: Room, group, group chat

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

**Test target**:
Where a tester checks a ticket in a testing-stage column: the preview
deployment of a branch linked to it, else a shared test environment other
work also lands on. Never production, meaning the default branch's own
deployment or a branch deployment on its network with nothing overridden.
_Avoid_: Test URL, staging

**Stack**:
One repository's worth of deployable containers: a compose file, or a
Dockerfile as a stack of one. The stack is what gets deployed, rolled back,
and torn down; the services inside it are observed, not deployed on their
own. Decided 2026-09-08 for the project wizard.
_Avoid_: Compose project, resource, application

**Instance stack**:
A stack that belongs to the instance rather than to a project: the backing
stack of a gateway, the tunnel or reverse proxy first run deploys among
them. It never builds from a repository, shows on Topology and in
Settings → DNS, and is checked like the topology: holding the stack
action in any of the caller's workspaces is enough.
_Avoid_: System stack, infrastructure project

**Stack slug**:
The DNS- and Docker-safe token derived once from a stack's name at creation,
unique per machine. It is the compose project name, a run stack's container
name, and the checkout directory under the stack root. Never generated or
suffixed: a duplicate is rejected, because every routing mechanism addresses
the container by this name.

**Observation report**:
The list of containers a runner sends back with a terminal deploy result,
one entry per container the stack started: name, image, status, networks with
addresses, and ports (published ones with their host port, the rest as reachable
on the container's networks only). The only source of a service's observed facts;
the compose file remains the only source of its declared ones.
_Avoid_: Status report, inspect result

**Tests repository**:
A second repository a project may attach to hold its tests. Never deployed:
no stack builds from it, so a project still has one repository that ships.
Its presence, or the answer that tests live in the deployed repository, is
what the interview starts from on testing.
_Avoid_: E2E repo, QA repo, test project

**Service** (deploy):
One running container Nexul keeps a record of: name, image, networks,
address, status. A stack deploy creates or updates one service per
container it started; an import adopts one per container found running.
_Avoid_: Container (the Docker thing a service is a record of)

**Container logs**:
What a service's container prints on stdout and stderr, each line with its
timestamp and stream, read live from Docker through the machine's runner and
never stored in Nexul: a snapshot of the last lines, or a live tail that
follows new ones. The stack's own env values are masked before a line leaves
the server. Gated by `stacks:logs`.
_Avoid_: Deploy log (the build and start output Nexul stores per deploy),
service logs (Nexul's own, which OpenObserve holds)

**Stack root**:
The directory on a machine under which every stack's persistent checkout
lives, at `stacks/<slug>/repo` beneath it. Set by the runner that creates the
machine when it enrolls, then configurable per machine; a stack's location is
fixed once it has deployed.

**Unmanaged stack**:
A stack adopted from what was already running on a machine (a manual
`docker run`, a stack some other tool deployed) with no repository attached. Its
containers can be seen, wired, and exposed, but it cannot be deployed until
a repository is attached, which makes it managed.

**Instance upgrade**:
Moving every Nexul service on the instance's server to the newest release of
its channel, or to a chosen one, with `nexul upgrade`: on the host, or from
the UI and the `instance_upgrade` MCP tool, which ask the `instance` runner to
start it detached from its own service. The server records a UI or MCP
upgrade and resolves the record when it boots on the target version.
_Avoid_: Update (that word is the runner's own binary swap), deploy

**Install directory**:
Where `nexul install` keeps an instance's data: `/data/nexul` on a Linux
server and `~/nexul` on a Mac or Windows PC, holding the generated `.env`,
`data/`, `logs/` and `stacks/`. The binaries and service definitions live
outside it. Uninstall keeps it unless purged, and installing into it again
brings the same instance back.
_Avoid_: Checkout (there is no git checkout of Nexul on a server)

**Machine**:
A server runners and automations hosts run on. Runners belong to a machine;
a service targets a machine and any of its runners may take the job, so the
number of runners per machine is the scaling knob. Replaces "target is the runner" from
2026-09-07 once built.
_Avoid_: Host, node, server (in the UI)

**Wizard**:
A guided multi-step flow that ends in something working, reached under
`/wizard/<context>/<step-owner>` (for example `/wizard/onboarding/owner`,
`/wizard/project/service`). Dialogs are not wizards.

**Automation**:
First-party event-driven code: when an event happens, a function runs.
Written in TS/JS against the SDK — **Default automations** ship with the
instance, **Custom automations** are owner-written. An automation belongs to
one workspace: it hears that workspace's events (and instance-level ones),
reads that workspace's secrets, and is switched and configured there; every
workspace has its own copy of each default. An automation connects out to the
instance and acts back through the API with its own scoped token; Nexul never
calls in to it.
_Avoid_: Rule, workflow (the v1 rule engine is gone)

**Automations host**:
A named service that runs the automations placed on it, each automation on
exactly one host. The one `nexul install` puts on the instance's own server is
named `instance` and is where new automations go; more can be installed on
other machines.
_Avoid_: Automations container, runner (a runner builds and deploys)

**Enrollment code**:
A one-time code, valid for an hour, that lets one named runner or automations
host enroll with the instance and receive its host credential. It travels
inside the install command the instance renders and is useless once used.
_Avoid_: Join token, registration token, runner secret

**Host credential**:
The credential one runner or automations host authenticates with, its own
and no other host's, from enrollment until the host is removed. Removal
revokes it, and a host refused with a revoked credential uninstalls itself.
_Avoid_: Runner secret, shared secret, token (a token acts for a person, an
integration, or an automation)

**Connector**:
A third-party tool the instance holds a credential for and calls out to:
GitHub, Cloudflare, LiveKit. One credential per tool per instance, encrypted
at rest. Nexul acting as itself against someone else's API; an
Integration is the opposite direction.
_Avoid_: Integration (the other direction), OAuth app, provider

**Installation**:
One account or organisation Nexul's GitHub App is installed on, granting all
of its repositories or a selection. The installations the GitHub connector's
user can see decide which repositories Nexul reads, so adding an organisation,
or a collaborator installing the App on their own account, is how a repository
becomes visible.
_Avoid_: Install (that is the `nexul install` command), connection, grant

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
found after done is a new ticket found in it. A bug is a ticket whose type is
named `bug`; a type renamed away from it is an ordinary type.
_Avoid_: Regression of, caused by, parent

**Blocked by**:
A ticket's link to another ticket that must reach done first. It shows on
the card and warns before a play runs, but never stops a card moving.
_Avoid_: Depends on, dependency, blocker stage

**Body template**:
The markdown sections a ticket type pre-fills into a new ticket's body, edited
on the type in project settings. A new project's task, bug, and feature types
start from the instance template of the same name. Guidance only: never
validated, and editing it never rewrites a ticket already born from it.
_Avoid_: Ticket template, form, checklist

**Instance template**:
The instance's own version of a template that every workspace or project
starts from: the Interview template, the mention chip template, each built-in
play's instructions, and each default ticket type's body template. A template
resolves code default, then instance template, then workspace or project; one
nobody edited at the instance is the code default. A workspace's Interview and
mention chip templates follow the instance's until the workspace edits its
own, while play instructions and body templates are copied in when the
workspace or project is created and never rewritten. Resetting a workspace's
or project's template gives it the instance's; resetting the instance's gives
the code default. Any template can be cloned over another of the same kind,
plays matched by their built-in key and ticket types by name. Read by every
member, edited with `templates:write`.
_Avoid_: Global template, default template (that is the code's), master copy

### Identity and access

**Permission**:
One capability, written `<domain>:<action>` where the action is `read`,
`write`, or `delete` (`docs:write`, `members:delete`), or a verb the domain
declares for an act that is neither (`plays:run`, `memories:clone`,
`roles:clone`, `docs:thread`, `docs:clone`, `docs:lock`, `stacks:logs`). One vocabulary for every actor: a role, a
scoped token, and the agent are checked against the same values. Checked in
the workspace the entity belongs to; runners, the topology, machines, DNS,
connectors, the instance's own stacks, and the instance itself (its settings,
upgrades, accounts, workspace creation), which belong to none, against every
workspace the caller is in and not a Restricted member of. The Owner of any workspace therefore holds every
instance-level permission, and nobody grants a permission they don't hold. What every member reads (the project list, public
channels, their own DMs and inbox, People) takes membership, not a permission; a Restricted member's project list
is the projects they hold Project access to, and they read no public channel. Each domain is a project area, a
workspace area, or an instance area; a role's project areas reach every project only for members whose Every
project is From role.
_Avoid_: Right, privilege, capability, ACL entry, instance admin (holding the
instance's permissions is what that meant)

**Permission overwrite**:
A per-user allow/deny set layered on top of their role, either workspace-wide
or on one resource. Most-specific wins; deny beats allow inside a layer. The
workspace Owner bypasses all of it.
_Avoid_: Grant, share, ACL

**Restricted member**:
A workspace member whose Every project is None: they see only the projects
they hold Project access to, instead of every project. A project they hold
none on is invisible to them, its name included, and so are projects made
later. Their role still decides the workspace areas; it opens no instance
area, and of channels they read only the Private channels they are in, plus
DMs and the threads of what they can read. The Owner is never one.
_Avoid_: Guest, client (Client is a role someone named), external member

**Project access**:
The levels one person holds in one project, per project area, from None to
Delete. For a Restricted member it is the whole answer inside that project,
with a doc's own sharing on top; everyone else gets a role's project areas
on every project. Set from Team, or carried by an Invitation.
_Avoid_: Project role, project membership, project grant, share

**Every project**:
The first row of a person's access in a workspace: From role, where the
role's project areas apply on every project, new ones included, or None,
which makes them a Restricted member. Every member upgraded from before it
existed is From role.
_Avoid_: All projects, access mode, restricted switch

**Private channel**:
A Channel only its members see and read, rather than every member of the
workspace; to anyone else it reads as not found. Switched either way with
`channels:write`; a member adds people, `channels:write` removes them, and
anyone but the last member may leave. The workspace Owner sees every one.
Its events, a DM's too, reach no Integration or Automation.
`#general` is never private.
_Avoid_: Group, locked channel, hidden channel

**Invitation**:
A single-use bearer link that admits one person to the instance and grants a
chosen Role plus optional Permission overwrites, and the Every project row
with any Project access, in one or more Workspaces. The
link expires after one or seven days and is not bound to a provider identity.
_Avoid_: Allowlist entry, invite code, join link

**Account status**:
Whether a registered User may authenticate to the Instance: active, disabled,
or removed. Workspace membership and Roles remain separate.
_Avoid_: Allowlist status, membership status

**Team**:
Everyone registered on the instance and what each can reach: every account
with its Account status, whether it is online or when it was last seen, and,
per workspace, its Role, workspace-wide Permission overwrites, Every project
row, and Project access. A holder of
`accounts:read` sees all of it, under Instance settings; someone who manages
members in a workspace finds it in Configuration and sees only the workspaces
they manage. A change
inside a workspace always needs `members:write` there, and the Owner role is
never given or taken through it.
_Avoid_: Members (one workspace's roster), registered accounts, users

**Instance settings**:
The group on the Settings page for what belongs to the whole instance rather
than one workspace: Instance, Team, Sign-in providers, Connectors, DNS, and
Templates.
Each entry shows only to a viewer holding its permission in any workspace, so
a role holding one bit reaches that entry and nothing else. Workspace-level
sections stay in Configuration.
_Avoid_: Whole instance, admin settings

**Display name**:
What a person is called wherever they appear: the name they set in their
profile, else their sign-in account's name, else their login. The login
stays where a handle is meant: a chat @mention, a Mention written as
markdown, and the line under a name. A person's Mention chip shows the
display name.
_Avoid_: Username, nickname, name (alone, that is the sign-in account's)

**People**:
A workspace's members as any member sees them: login, display name, and
picture, and nothing else. Readable by every member of that workspace,
whatever their role; managing members is the Team's.
_Avoid_: Directory, roster, members (the managed list with roles)

**Mention**:
An @-reference in a doc, ticket, or memory body to a ticket, a doc, or a
person, stored as a node holding the target's id and shown as a live chip.
A person's holds their user id and its chip shows their picture and
display name from People; as markdown it is `[@login](/people/<user id>)`.
Saving a doc or ticket that newly mentions someone tells them in their
inbox. Chat's `@login` and `@Agent` are plain text in the message instead.
_Avoid_: Tag, ping

**Personal access token**:
A long-lived, revocable credential (`dep_`) carrying exactly one user's own
permissions. What an agent authenticates its MCP connection with. A paired
computer has at most one of its own, "Nexul MCP on <computer>", revoked when
its setup is un-confirmed or it is removed; a saved transcript shows any
personal access token as `[redacted token]`.

**Sign-in identity**:
One provider account attached to a user: GitHub, Google or Discord, keyed by
the provider's own id, with the login it yields. A user holds one per
provider at most and signs in through any of them; the first one created is
the one their login, name and picture follow. Linked from Profile while
signed in, unlinked from there too, never down to none.
_Avoid_: Account (that is the user), login method, connected account (a
Connector is the instance's credential, not a person's)

**Session**:
One signed-in device (`ses_`): a browser, the desktop app, or a phone. Stored
as a row with its platform and label, listed on the user's Devices, signed
out one at a time or everywhere else at once, and expiring 30 days after
last use (90 for a phone). A user's own; no permission bit and no MCP tool,
so an agent can never sign a person out.
_Avoid_: Login token, JWT, cookie, refresh token

**Device**:
What a session belongs to, as the sign-in saw it: the client kind (browser,
desktop, phone), the platform ("Linux", "Android"), and a label (the browser
name, "Nexul desktop", or the phone's model). Shown as "platform · label".
_Avoid_: Paired computer (that runs a harness), machine, host

**Connect code**:
The code a phone trades for its session: twelve Crockford base32 characters
shown as `XXXX-XXXX-XXXX` and inside the QR link `nexul://connect?host=…&code=…`,
valid two minutes, single use, one live code per user with a newer one
replacing the last. Issued only from a signed-in device, never by a personal
access token, so no agent can sign a phone in. Only its hash is stored.
_Avoid_: Pairing code (that pairs a computer), QR token, login code

**Push token**:
The Expo token a phone registers on its own session so the instance can
reach it. One push per inbox notification per phone, titled "Nexul" with an
id-only payload the app fetches the content for; posted from a consumer of
the notification event, never from the request that made it. Signing the
session out drops the token with the row, and a device Expo reports as no
longer registered loses it.
_Avoid_: Device token, FCM token (that is Expo's concern), notification key

**Scoped token**:
The credential an integration (`int_`) or an automation (`dat_`) acts with,
minted with a chosen subset of permissions and revocable on its own. A scoped
token granted `X:write` also receives `X:read` at mint.

**Setup code**:
The code (`nxs_`) that proves someone at the instance's own server is the one
setting it up. The server writes a fresh one on every start until the first
user exists, valid a day, and the installer prints it. Unlocking does not use
it up, so the owner can carry it from the server's address to the domain.
Gone once the first user exists.
_Avoid_: Enrollment code (that enrolls a host), invite code, admin password

**Setup pass**:
What a correct setup code unlocks: a bearer (`nxsp_`) good for an hour that
lets one browser set up the domain and the GitHub App before any user exists.
It reaches only the first-run routes and acts as the identity `setup`. Every
pass stops working the moment the first user exists.
_Avoid_: Session (that belongs to a user), setup token, bootstrap key

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
(`plays:run`, `memories:clone`, `roles:clone`, `docs:thread`, `docs:clone`, `docs:lock`, `stacks:logs`), one
vocabulary shared by roles, token scopes, and the agent. Distinct from auth.

**Auth** — who a user is. Identity and device sessions, via an owner-configured OAuth provider.
Distinct from access.

**Automations** — event-driven code: Default and Custom automations run
functions when events happen, acting through the scoped API, on the
automations hosts they are placed on.

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

**Docs** — documentation as the source of truth, grouped into each project's
folders, with `@` cross-references to tickets and other docs.

**Git provider** — the abstraction over GitHub and future providers: branches,
PRs, webhooks.

**Integrations** — the platform for external services: scoped tokens, signed
webhooks, the store.

**MCP** — the Model Context Protocol adapter, tools shaped per task over the
use-case layer, so LLM agents can drive the product.

**Memories** — the Memory entity: its table, page, permission verbs, and MCP
tools. Never shares a list, a page, or search with Docs.

**Runner** — runner enrollment and removal, the WebSocket protocol, queueing
and dispatch.

**Search** — FTS5 full-text search over docs and tickets. Not a package: each
domain registers its own index and query.

**Tickets** — work items on a Kanban board, linked to docs, branches, and PRs.

**Topology** — the canvas model: nodes, edges, live status.

**Voice** — voice channel LiveKit interaction: join tokens and live
occupancy, over a bring-your-own LiveKit server.

**Workspace** — the workspace boundary itself, its members, its settings, and
notifications. Notifications are a capability here rather than a domain of
their own: every domain publishes events, and the workspace turns the ones a
member cares about into an inbox. A read notification is deleted 90 days after it was
read, and any notification 180 days after it was sent.

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
